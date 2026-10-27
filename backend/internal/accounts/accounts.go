// Package accounts is organizations, their accounts, and signing in (spec 4,
// 4A, 8, 8A). Every query names the organization, which comes from the
// address the person opened (package site) or from their access token, never
// from what the client sends.
package accounts

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Transcentive/PhoenixEvent/backend/internal/config"
	"github.com/Transcentive/PhoenixEvent/backend/internal/email"
	"github.com/Transcentive/PhoenixEvent/backend/internal/password"
	"github.com/Transcentive/PhoenixEvent/backend/internal/token"
)

// Link lifetimes (8A.2). A first Organization Administrator's link to set a
// password is an invitation in all but name, so it lasts as long as one.
const (
	ConfirmLifetime     = 24 * time.Hour
	ResetLifetime       = time.Hour
	SetPasswordLifetime = 7 * 24 * time.Hour
)

// Failed logins (8A.2): 5 for one username within 15 minutes pause that
// username for 15 minutes.
const (
	maxFailures   = 5
	failureWindow = 15 * time.Minute
)

type Organization struct {
	ID   string `json:"-"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Account struct {
	ID             string  `json:"id"`
	OrganizationID string  `json:"-"`
	Email          string  `json:"email"`
	Username       *string `json:"username"`
}

// Session is what a successful login or refresh hands the client.
type Session struct {
	AccessToken  string    `json:"access_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	Account      Account   `json:"account"`
	RefreshToken string    `json:"-"`
}

var (
	ErrNotFound          = errors.New("not found")
	ErrBadCredentials    = errors.New("That username and password do not match an account in this organization. Check both and try again, or reset your password.")
	ErrTooManyAttempts   = errors.New("Too many attempts. Try again in 15 minutes.")
	ErrEmailNotConfirmed = errors.New("Confirm your email first.")
	ErrLinkInvalid       = errors.New("This link has ended.")
	ErrSessionEnded      = errors.New("Your session has ended. Log in again.")
)

// FieldError is a problem with one field of a form, shown under that field
// (screen E4).
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type FieldErrors []FieldError

func (f FieldErrors) Error() string {
	parts := make([]string, len(f))
	for i, e := range f {
		parts[i] = e.Field + ": " + e.Message
	}
	return strings.Join(parts, "; ")
}

type Service struct {
	db     *pgxpool.Pool
	mail   email.Sender
	site   config.Site
	tokens *token.Issuer
	now    func() time.Time
}

func NewService(db *pgxpool.Pool, mail email.Sender, site config.Site, tokens *token.Issuer) *Service {
	return &Service{db: db, mail: mail, site: site, tokens: tokens, now: time.Now}
}

// OrganizationBySlug finds the organization at an address.
func (s *Service) OrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	o := &Organization{}
	err := s.db.QueryRow(ctx, `SELECT id, name, slug FROM organizations WHERE slug = $1`, slug).
		Scan(&o.ID, &o.Name, &o.Slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return o, err
}

// MatchLimit is the most organizations MatchOrganizations offers at once.
const MatchLimit = 10

// MatchOrganizations is A1's Organization field (4A.7): the organizations
// whose name or address name contains what was typed, ignoring case.
// Organization names are not secret (spec v1.01).
func (s *Service) MatchOrganizations(ctx context.Context, typed string) ([]Organization, error) {
	typed = strings.TrimSpace(typed)
	if typed == "" {
		return []Organization{}, nil
	}
	pattern := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(typed) + "%"
	rows, err := s.db.Query(ctx, `
		SELECT id, name, slug FROM organizations
		WHERE name ILIKE $1 OR slug ILIKE $1
		ORDER BY lower(name), slug
		LIMIT $2`, pattern, MatchLimit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Organization])
}

// CreateOrganization is the System Administrator's server command (4A.6): the
// organization, its first Organization Administrator, and the email with the
// link to set a password.
func (s *Service) CreateOrganization(ctx context.Context, name, slug, adminEmail string) (*Organization, error) {
	name = strings.TrimSpace(name)
	slug = strings.ToLower(strings.TrimSpace(slug))
	adminEmail, err := normalizeEmail(adminEmail)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, errors.New("the organization needs a name")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	org := &Organization{Name: name, Slug: slug}
	if err := tx.QueryRow(ctx,
		`INSERT INTO organizations (name, slug) VALUES ($1, $2) RETURNING id`, name, slug,
	).Scan(&org.ID); err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("the address name %q is already taken", slug)
		}
		if isCheckViolation(err) {
			return nil, fmt.Errorf("the address name %q may hold only lower-case letters, digits and hyphens, and may not start or end with a hyphen", slug)
		}
		return nil, err
	}
	var accountID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO accounts (organization_id, email) VALUES ($1, $2) RETURNING id`, org.ID, adminEmail,
	).Scan(&accountID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO organization_roles (organization_id, account_id, role) VALUES ($1, $2, 'organization_administrator')`,
		org.ID, accountID); err != nil {
		return nil, err
	}
	raw, err := s.newLink(ctx, tx, accountID, "set_password", SetPasswordLifetime)
	if err != nil {
		return nil, err
	}
	if err := audit(ctx, tx, org.ID, "", "organization.created", map[string]any{"name": name, "slug": slug, "admin_email": adminEmail}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	link := s.site.OrgURL(slug) + "set-password?token=" + raw
	err = s.mail.Send(ctx, email.Message{
		To:      adminEmail,
		Subject: "Set your password for " + name + " on PhoenixEvent",
		Text: "You are the Organization Administrator of " + name + " on PhoenixEvent.\n\n" +
			"Open this link to set your password:\n" + link + "\n\n" +
			"The link works once and expires after 7 days. Then log in at " + s.site.OrgURL(slug) + "\n",
	})
	if err != nil {
		return org, fmt.Errorf("the organization was created, but the email did not go: %w", err)
	}
	return org, nil
}

// Register creates an account at the organization (screen A3) and sends the
// confirmation link (A4).
func (s *Service) Register(ctx context.Context, org *Organization, emailAddr, username, pw string) (*Account, error) {
	var problems FieldErrors
	emailAddr, err := normalizeEmail(emailAddr)
	if err != nil {
		problems = append(problems, FieldError{"email", "Enter an email address, such as dana@example.org."})
	}
	username = strings.TrimSpace(username)
	if strings.Contains(username, "@") {
		problems = append(problems, FieldError{"username", "A username cannot contain @. Leave it empty to use your email address."})
	}
	if err := password.Check(pw); err != nil {
		problems = append(problems, FieldError{"password", err.Error()})
	}
	if len(problems) > 0 {
		return nil, problems
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	hash, err := password.Hash(pw)
	if err != nil {
		return nil, err
	}
	acc := &Account{OrganizationID: org.ID, Email: emailAddr}
	if username != "" {
		acc.Username = &username
	}
	err = tx.QueryRow(ctx,
		`INSERT INTO accounts (organization_id, email, username, password_hash, email_level) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		org.ID, emailAddr, acc.Username, hash, startingEmailLevel(acc.Username),
	).Scan(&acc.ID)
	if isUniqueViolation(err) {
		if strings.Contains(constraintName(err), "username") {
			return nil, FieldErrors{{"username", "That username is taken in " + org.Name + ". Choose another."}}
		}
		return nil, FieldErrors{{"email", "An account with this email address already exists in " + org.Name + ". Log in, or reset your password."}}
	}
	if err != nil {
		return nil, err
	}
	raw, err := s.newLink(ctx, tx, acc.ID, "confirm_email", ConfirmLifetime)
	if err != nil {
		return nil, err
	}
	if err := audit(ctx, tx, org.ID, acc.ID, "account.created", map[string]any{"email": emailAddr}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return acc, s.sendConfirmation(ctx, org, emailAddr, raw)
}

// ResendConfirmation sends a new confirmation link, and says nothing about
// whether the address has an account here.
func (s *Service) ResendConfirmation(ctx context.Context, org *Organization, emailAddr string) error {
	emailAddr, err := normalizeEmail(emailAddr)
	if err != nil {
		return nil
	}
	var accountID string
	err = s.db.QueryRow(ctx,
		`SELECT id FROM accounts WHERE organization_id = $1 AND lower(email) = lower($2) AND email_confirmed_at IS NULL AND password_hash IS NOT NULL`,
		org.ID, emailAddr).Scan(&accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	raw, err := s.newLink(ctx, s.db, accountID, "confirm_email", ConfirmLifetime)
	if err != nil {
		return err
	}
	return s.sendConfirmation(ctx, org, emailAddr, raw)
}

func (s *Service) sendConfirmation(ctx context.Context, org *Organization, to, raw string) error {
	return s.mail.Send(ctx, email.Message{
		To:      to,
		Subject: "Confirm your email for " + org.Name + " on PhoenixEvent",
		Text: "Open this link to confirm your email address and finish creating your account at " + org.Name + ":\n" +
			s.site.OrgURL(org.Slug) + "confirm-email?token=" + raw + "\n\n" +
			"The link works once and expires after 24 hours. If you did not create this account, ignore this email.\n",
	})
}

// LinkPurpose reports what an unused, unexpired link is for, so screen A6 can
// show E2 before the person types a password.
// The login is the account's username, or its email address when it has
// none, so A6 can say whose password is being set.
func (s *Service) LinkPurpose(ctx context.Context, org *Organization, raw string) (purpose, login string, err error) {
	err = s.db.QueryRow(ctx, `
		SELECT l.purpose, coalesce(a.username, a.email) FROM email_links l JOIN accounts a ON a.id = l.account_id
		WHERE l.token_hash = $1 AND a.organization_id = $2 AND l.used_at IS NULL AND l.expires_at > $3`,
		token.Hash(raw), org.ID, s.now()).Scan(&purpose, &login)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrLinkInvalid
	}
	return purpose, login, err
}

// ConfirmEmail uses a confirmation link.
func (s *Service) ConfirmEmail(ctx context.Context, org *Organization, raw string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	accountID, err := s.useLink(ctx, tx, org, raw, "confirm_email")
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE accounts SET email_confirmed_at = coalesce(email_confirmed_at, $2), updated_at = $2 WHERE id = $1`,
		accountID, s.now()); err != nil {
		return err
	}
	if err := audit(ctx, tx, org.ID, accountID, "account.email_confirmed", nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RequestReset sends a reset link if the username or email address matches
// an account here, and says nothing either way (screen A5).
func (s *Service) RequestReset(ctx context.Context, org *Organization, login string) error {
	acc, _, err := s.findByLogin(ctx, org, login)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	raw, err := s.newLink(ctx, s.db, acc.ID, "reset_password", ResetLifetime)
	if err != nil {
		return err
	}
	return s.mail.Send(ctx, email.Message{
		To:      acc.Email,
		Subject: "Set a new password for " + org.Name + " on PhoenixEvent",
		Text: "Open this link to set a new password for your account at " + org.Name + ":\n" +
			s.site.OrgURL(org.Slug) + "reset-password?token=" + raw + "\n\n" +
			"The link works once and expires after 1 hour. If you did not ask for this, ignore this email; your password stays as it is.\n",
	})
}

// SetPassword uses a reset link, or a first Organization Administrator's
// link (4A.6), to set a new password. Every session the account had ends, so
// a person who reset a stolen password locks the thief out.
func (s *Service) SetPassword(ctx context.Context, org *Organization, raw, pw string) error {
	if err := password.Check(pw); err != nil {
		return FieldErrors{{"password", err.Error()}}
	}
	hash, err := password.Hash(pw)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	accountID, err := s.useLink(ctx, tx, org, raw, "reset_password", "set_password")
	if err != nil {
		return err
	}
	now := s.now()
	// The link reached the inbox, which confirms the address too.
	if _, err := tx.Exec(ctx,
		`UPDATE accounts SET password_hash = $2, email_confirmed_at = coalesce(email_confirmed_at, $3), updated_at = $3 WHERE id = $1`,
		accountID, hash, now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE sessions SET revoked_at = $2 WHERE account_id = $1 AND revoked_at IS NULL`, accountID, now); err != nil {
		return err
	}
	if err := audit(ctx, tx, org.ID, accountID, "account.password_set", nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Login signs a person in to the organization with a username or email
// address and a password (screen A2).
func (s *Service) Login(ctx context.Context, org *Organization, login, pw string) (*Session, error) {
	login = strings.TrimSpace(login)
	acc, hash, err := s.findByLogin(ctx, org, login)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	// Failures count against the account when one matches, so its username
	// and its email address share one count; otherwise against what was
	// typed.
	key := "login:" + strings.ToLower(login)
	if acc != nil {
		key = "account:" + acc.ID
	}
	now := s.now()
	var failures int
	if err := s.db.QueryRow(ctx,
		`SELECT count(*) FROM login_failures WHERE organization_id = $1 AND login_key = $2 AND failed_at > $3`,
		org.ID, key, now.Add(-failureWindow)).Scan(&failures); err != nil {
		return nil, err
	}
	if failures >= maxFailures {
		return nil, ErrTooManyAttempts
	}

	if acc == nil || hash == "" {
		// The same work as a real check, so the answer's timing does not
		// tell whether the username exists.
		password.Matches(dummyHash, pw)
		return nil, s.recordFailure(ctx, org, key)
	}
	if !password.Matches(hash, pw) {
		return nil, s.recordFailure(ctx, org, key)
	}

	var confirmed bool
	if err := s.db.QueryRow(ctx, `SELECT email_confirmed_at IS NOT NULL FROM accounts WHERE id = $1`, acc.ID).Scan(&confirmed); err != nil {
		return nil, err
	}
	if !confirmed {
		return nil, ErrEmailNotConfirmed
	}

	if _, err := s.db.Exec(ctx,
		`DELETE FROM login_failures WHERE organization_id = $1 AND login_key = $2`, org.ID, key); err != nil {
		return nil, err
	}
	raw, err := token.Secret()
	if err != nil {
		return nil, err
	}
	var sessionID string
	if err := s.db.QueryRow(ctx,
		`INSERT INTO sessions (organization_id, account_id, refresh_token_hash, expires_at) VALUES ($1, $2, $3, $4) RETURNING id`,
		org.ID, acc.ID, token.Hash(raw), now.Add(token.RefreshLifetime)).Scan(&sessionID); err != nil {
		return nil, err
	}
	if err := audit(ctx, s.db, org.ID, acc.ID, "session.login", nil); err != nil {
		return nil, err
	}
	return s.session(acc, sessionID, raw)
}

func (s *Service) recordFailure(ctx context.Context, org *Organization, key string) error {
	if _, err := s.db.Exec(ctx,
		`INSERT INTO login_failures (organization_id, login_key, failed_at) VALUES ($1, $2, $3)`,
		org.ID, key, s.now()); err != nil {
		return err
	}
	return ErrBadCredentials
}

// Refresh exchanges a refresh token for a new access token and a new refresh
// token, and starts the 30 days again (8A.2).
func (s *Service) Refresh(ctx context.Context, org *Organization, raw string) (*Session, error) {
	now := s.now()
	acc := &Account{}
	var sessionID string
	err := s.db.QueryRow(ctx, `
		SELECT s.id, a.id, a.organization_id, a.email, a.username
		FROM sessions s JOIN accounts a ON a.id = s.account_id
		WHERE s.refresh_token_hash = $1 AND s.organization_id = $2
		  AND s.revoked_at IS NULL AND s.expires_at > $3`,
		token.Hash(raw), org.ID, now).Scan(&sessionID, &acc.ID, &acc.OrganizationID, &acc.Email, &acc.Username)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSessionEnded
	}
	if err != nil {
		return nil, err
	}
	next, err := token.Secret()
	if err != nil {
		return nil, err
	}
	tag, err := s.db.Exec(ctx, `
		UPDATE sessions SET refresh_token_hash = $3, last_used_at = $4, expires_at = $5
		WHERE id = $1 AND refresh_token_hash = $2 AND revoked_at IS NULL`,
		sessionID, token.Hash(raw), token.Hash(next), now, now.Add(token.RefreshLifetime))
	if err != nil {
		return nil, err
	}
	// Two tabs refreshing at once: the second finds its token already
	// replaced and must log in again rather than share the session.
	if tag.RowsAffected() != 1 {
		return nil, ErrSessionEnded
	}
	return s.session(acc, sessionID, next)
}

// Logout ends the session the refresh token belongs to (8A.2: at once).
func (s *Service) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	var orgID, accountID string
	err := s.db.QueryRow(ctx, `
		UPDATE sessions SET revoked_at = $2 WHERE refresh_token_hash = $1 AND revoked_at IS NULL
		RETURNING organization_id, account_id`, token.Hash(raw), s.now()).Scan(&orgID, &accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return audit(ctx, s.db, orgID, accountID, "session.logout", nil)
}

// Authenticate checks an access token's session on every request: still
// open, and in the organization at this address. A session revoked by
// logging out or by a password reset is refused at once, though its token
// has not expired.
func (s *Service) Authenticate(ctx context.Context, org *Organization, claims *token.Claims) (*Account, error) {
	if claims.OrganizationID != org.ID {
		return nil, ErrSessionEnded
	}
	acc := &Account{}
	err := s.db.QueryRow(ctx, `
		SELECT a.id, a.organization_id, a.email, a.username
		FROM sessions s JOIN accounts a ON a.id = s.account_id
		WHERE s.id = $1 AND s.account_id = $2 AND s.organization_id = $3
		  AND s.revoked_at IS NULL AND s.expires_at > $4`,
		claims.SessionID, claims.AccountID, org.ID, s.now()).Scan(&acc.ID, &acc.OrganizationID, &acc.Email, &acc.Username)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSessionEnded
	}
	return acc, err
}

// FindOrganizations emails a link to each organization that has an account
// under the address (screen A1, 4A.7), and says nothing on screen.
func (s *Service) FindOrganizations(ctx context.Context, emailAddr string) error {
	emailAddr, err := normalizeEmail(emailAddr)
	if err != nil {
		return nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT o.name, o.slug FROM accounts a JOIN organizations o ON o.id = a.organization_id
		WHERE lower(a.email) = lower($1) ORDER BY o.name`, emailAddr)
	if err != nil {
		return err
	}
	orgs, err := pgx.CollectRows(rows, pgx.RowToStructByPos[orgLink])
	if err != nil {
		return err
	}
	if len(orgs) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("You have an account at these organizations on PhoenixEvent. Open a link to go to its login page:\n\n")
	for _, o := range orgs {
		b.WriteString(o.Name + "\n" + s.site.OrgURL(o.Slug) + "\n\n")
	}
	b.WriteString("If you did not ask for this, ignore this email.\n")
	return s.mail.Send(ctx, email.Message{
		To:      emailAddr,
		Subject: "Your organizations on PhoenixEvent",
		Text:    b.String(),
	})
}

// orgLink is one organization in FindOrganizations' email.
type orgLink struct {
	Name string
	Slug string
}

func (s *Service) session(acc *Account, sessionID, refresh string) (*Session, error) {
	access, expires, err := s.tokens.Access(acc.ID, acc.OrganizationID, sessionID)
	if err != nil {
		return nil, err
	}
	return &Session{AccessToken: access, ExpiresAt: expires, Account: *acc, RefreshToken: refresh}, nil
}

// findByLogin finds the account a username or an email address names in the
// organization, matched without regard to capitals (4A.8). A username never
// contains @, so the two can never name different accounts.
func (s *Service) findByLogin(ctx context.Context, org *Organization, login string) (*Account, string, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return nil, "", ErrNotFound
	}
	column := "username"
	if strings.Contains(login, "@") {
		column = "email"
	}
	acc := &Account{}
	var hash *string
	err := s.db.QueryRow(ctx,
		`SELECT id, organization_id, email, username, password_hash FROM accounts
		 WHERE organization_id = $1 AND lower(`+column+`) = lower($2)`,
		org.ID, login).Scan(&acc.ID, &acc.OrganizationID, &acc.Email, &acc.Username, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	if hash == nil {
		return acc, "", nil
	}
	return acc, *hash, nil
}

type execQuerier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// newLink stores a new single-use link for the account and returns the secret
// that goes in the email. Earlier unused links for the same purpose end, so
// only the newest one works.
func (s *Service) newLink(ctx context.Context, db execQuerier, accountID, purpose string, lifetime time.Duration) (string, error) {
	raw, err := token.Secret()
	if err != nil {
		return "", err
	}
	now := s.now()
	if _, err := db.Exec(ctx,
		`UPDATE email_links SET used_at = $3 WHERE account_id = $1 AND purpose = $2 AND used_at IS NULL`,
		accountID, purpose, now); err != nil {
		return "", err
	}
	if _, err := db.Exec(ctx,
		`INSERT INTO email_links (account_id, purpose, token_hash, created_at, expires_at) VALUES ($1, $2, $3, $4, $5)`,
		accountID, purpose, token.Hash(raw), now, now.Add(lifetime)); err != nil {
		return "", err
	}
	return raw, nil
}

// useLink marks a link used and returns its account. An unknown, used,
// expired or wrong-purpose link, or one for another organization, is
// ErrLinkInvalid, so screen E2 reveals nothing about which.
func (s *Service) useLink(ctx context.Context, tx pgx.Tx, org *Organization, raw string, purposes ...string) (string, error) {
	var accountID string
	err := tx.QueryRow(ctx, `
		UPDATE email_links l SET used_at = $3
		FROM accounts a
		WHERE a.id = l.account_id AND l.token_hash = $1 AND a.organization_id = $2
		  AND l.used_at IS NULL AND l.expires_at > $3 AND l.purpose = ANY($4)
		RETURNING l.account_id`,
		token.Hash(raw), org.ID, s.now(), purposes).Scan(&accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrLinkInvalid
	}
	return accountID, err
}

func audit(ctx context.Context, db execQuerier, orgID, accountID, action string, detail map[string]any) error {
	if detail == nil {
		detail = map[string]any{}
	}
	_, err := db.Exec(ctx,
		`INSERT INTO audit_log (organization_id, account_id, action, detail) VALUES (NULLIF($1, '')::uuid, NULLIF($2, '')::uuid, $3, $4)`,
		orgID, accountID, action, detail)
	return err
}

func normalizeEmail(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	addr, err := mail.ParseAddress(raw)
	if err != nil || addr.Address != raw || !strings.Contains(raw, "@") {
		return "", errors.New("not an email address")
	}
	return raw, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}

func constraintName(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}
	return ""
}

// dummyHash is a bcrypt hash of a random value at the same cost as real ones.
var dummyHash = func() string {
	secret, _ := token.Secret()
	h, _ := password.Hash(secret)
	return h
}()
