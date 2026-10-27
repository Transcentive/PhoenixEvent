package accounts

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/Transcentive/PhoenixEvent/backend/internal/password"
)

// The levels of 8B.4, from wider to narrower. Public means not logged in.
const (
	LevelPublic       = "public"
	LevelOrganization = "organization"
	LevelEvent        = "event"
	LevelStation      = "station"
	LevelPrivate      = "private"
)

var levels = map[string]bool{LevelPublic: true, LevelOrganization: true, LevelEvent: true, LevelStation: true, LevelPrivate: true}

// The three names of 8B.2.
const (
	ShownEmail        = "email"
	ShownUsername     = "username"
	ShownPersonalName = "personal_name"
)

var roleLabels = map[string]string{
	"organization_administrator": "Organization Administrator",
	"organization_leadership":    "Organization Leadership",
}

// Profile is the owner's own view (screen A8).
type Profile struct {
	ID            string   `json:"id"`
	Email         string   `json:"email"`
	Username      *string  `json:"username"`
	FirstName     *string  `json:"first_name"`
	MiddleName    *string  `json:"middle_name"`
	LastName      *string  `json:"last_name"`
	NameShown     string   `json:"name_shown"`
	Name          string   `json:"name"`
	EmailLevel    string   `json:"email_level"`
	UsernameLevel string   `json:"username_level"`
	NameLevel     string   `json:"name_level"`
	RolesLevel    string   `json:"roles_level"`
	Roles         []string `json:"roles"`
	// NameChoice is the person's own choice of name shown; empty while the
	// rule of 8B.2 decides.
	NameChoice *string `json:"name_choice"`

	chosen *string
}

// SharedProfile is what another person sees (screen A9): the name shown,
// and each element only when its owner has made it visible to the viewer.
type SharedProfile struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	PersonalName *string  `json:"personal_name,omitempty"`
	Username     *string  `json:"username,omitempty"`
	Email        *string  `json:"email,omitempty"`
	Roles        []string `json:"roles,omitempty"`
}

// ProfileChange is what A8's Save profile sends (8B.3, v1.06). Empty
// strings clear the username and the personal name.
type ProfileChange struct {
	Username   string `json:"username"`
	FirstName  string `json:"first_name"`
	MiddleName string `json:"middle_name"`
	LastName   string `json:"last_name"`
}

// VisibilityChange is what A8's Profile visibility Save sends. An empty
// name shown hands the choice back to the rule of 8B.2.
type VisibilityChange struct {
	NameShown     string `json:"name_shown"`
	EmailLevel    string `json:"email_level"`
	UsernameLevel string `json:"username_level"`
	NameLevel     string `json:"name_level"`
	RolesLevel    string `json:"roles_level"`
}

// startingEmailLevel is the email address's first level: organization
// members while it is the only name, otherwise private (8B.4, v1.04).
func startingEmailLevel(username *string) string {
	if username != nil {
		return LevelPrivate
	}
	return LevelOrganization
}

func (p *Profile) personalName() *string {
	if p.FirstName == nil || p.LastName == nil {
		return nil
	}
	parts := []string{*p.FirstName}
	if p.MiddleName != nil {
		parts = append(parts, *p.MiddleName)
	}
	name := strings.Join(append(parts, *p.LastName), " ")
	return &name
}

func (p *Profile) has(shown string) bool {
	switch shown {
	case ShownEmail:
		return true
	case ShownUsername:
		return p.Username != nil
	case ShownPersonalName:
		return p.personalName() != nil
	}
	return false
}

// settle works out the name shown: the person's choice when it is present,
// else the rule of 8B.2, the personal name, else the username, else the
// email address.
func (p *Profile) settle() {
	switch {
	case p.chosen != nil && p.has(*p.chosen):
		p.NameShown = *p.chosen
	case p.personalName() != nil:
		p.NameShown = ShownPersonalName
	case p.Username != nil:
		p.NameShown = ShownUsername
	default:
		p.NameShown = ShownEmail
	}
	p.Name = p.valueOf(p.NameShown)
	p.NameChoice = p.chosen
}

func (p *Profile) valueOf(shown string) string {
	switch shown {
	case ShownUsername:
		return *p.Username
	case ShownPersonalName:
		return *p.personalName()
	}
	return p.Email
}

func (p *Profile) levelOf(shown string) string {
	switch shown {
	case ShownUsername:
		return p.UsernameLevel
	case ShownPersonalName:
		return p.NameLevel
	}
	return p.EmailLevel
}

var uuidShape = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// GetProfile reads one account's profile within the organization.
func (s *Service) GetProfile(ctx context.Context, org *Organization, accountID string) (*Profile, error) {
	if !uuidShape.MatchString(accountID) {
		return nil, ErrNotFound
	}
	p := &Profile{}
	err := s.db.QueryRow(ctx, `
		SELECT id, email, username, first_name, middle_name, last_name, name_shown,
		       email_level, username_level, name_level, roles_level
		FROM accounts WHERE id = $1 AND organization_id = $2`, accountID, org.ID).
		Scan(&p.ID, &p.Email, &p.Username, &p.FirstName, &p.MiddleName, &p.LastName, &p.chosen,
			&p.EmailLevel, &p.UsernameLevel, &p.NameLevel, &p.RolesLevel)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx,
		`SELECT role FROM organization_roles WHERE account_id = $1 AND organization_id = $2 ORDER BY role`, p.ID, org.ID)
	if err != nil {
		return nil, err
	}
	roles, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	p.Roles = []string{}
	for _, r := range roles {
		p.Roles = append(p.Roles, roleLabels[r])
	}
	p.settle()
	return p, nil
}

func optional(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

// UpdateProfile is A8's Save profile: the username and the personal name.
func (s *Service) UpdateProfile(ctx context.Context, org *Organization, accountID string, c ProfileChange) (*Profile, error) {
	p, err := s.GetProfile(ctx, org, accountID)
	if err != nil {
		return nil, err
	}
	var problems FieldErrors
	p.Username = optional(c.Username)
	if p.Username != nil && strings.Contains(*p.Username, "@") {
		problems = append(problems, FieldError{"username", "A username cannot contain @."})
	}
	p.FirstName, p.MiddleName, p.LastName = optional(c.FirstName), optional(c.MiddleName), optional(c.LastName)
	if p.FirstName == nil && (p.LastName != nil || p.MiddleName != nil) {
		problems = append(problems, FieldError{"first_name", "Enter a first name, or clear the personal name."})
	}
	if p.LastName == nil && (p.FirstName != nil || p.MiddleName != nil) {
		problems = append(problems, FieldError{"last_name", "Enter a last name, or clear the personal name."})
	}
	if len(problems) > 0 {
		return nil, problems
	}
	// A cleared name that was the one chosen hands the choice back to 8B.2.
	if p.chosen != nil && !p.has(*p.chosen) {
		p.chosen = nil
	}
	p.settle()
	if !p.nameVisible() {
		return nil, FieldErrors{{"profile", "Saving this would leave no name visible to organization members. In Profile visibility, make another name visible to organization members and show it, then save your profile again."}}
	}
	return p, s.write(ctx, org, p, "profile.updated", map[string]any{
		"username": p.Username, "first_name": p.FirstName, "middle_name": p.MiddleName, "last_name": p.LastName,
	})
}

// UpdateVisibility is A8's Profile visibility Save: the level of each
// element and the name shown, among the names already saved.
func (s *Service) UpdateVisibility(ctx context.Context, org *Organization, accountID string, c VisibilityChange) (*Profile, error) {
	p, err := s.GetProfile(ctx, org, accountID)
	if err != nil {
		return nil, err
	}
	var problems FieldErrors
	for _, f := range []struct{ field, level string }{
		{"email_level", c.EmailLevel}, {"username_level", c.UsernameLevel}, {"name_level", c.NameLevel}, {"roles_level", c.RolesLevel},
	} {
		if !levels[f.level] {
			problems = append(problems, FieldError{f.field, "Choose who can see it."})
		}
	}
	p.EmailLevel, p.UsernameLevel, p.NameLevel, p.RolesLevel = c.EmailLevel, c.UsernameLevel, c.NameLevel, c.RolesLevel
	p.chosen = optional(c.NameShown)
	if p.chosen != nil && !p.has(*p.chosen) {
		problems = append(problems, FieldError{"name_shown", "Only what you have saved in your profile can be chosen."})
	}
	if len(problems) > 0 {
		return nil, problems
	}
	p.settle()
	if !p.nameVisible() {
		return nil, FieldErrors{{"name_shown", "The name shown must be visible at least to organization members. Choose Public or Organization members for it, or show another name."}}
	}
	return p, s.write(ctx, org, p, "profile.visibility_changed", map[string]any{
		"name_shown": p.chosen, "email_level": p.EmailLevel, "username_level": p.UsernameLevel,
		"name_level": p.NameLevel, "roles_level": p.RolesLevel,
	})
}

// nameVisible is the rule of 8B.4: the name shown is visible at least to
// organization members, which also keeps one name from being private.
func (p *Profile) nameVisible() bool {
	l := p.levelOf(p.NameShown)
	return l == LevelPublic || l == LevelOrganization
}

// write saves the whole profile and its audit record in one transaction.
func (s *Service) write(ctx context.Context, org *Organization, p *Profile, action string, detail map[string]any) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		UPDATE accounts SET username = $3, first_name = $4, middle_name = $5, last_name = $6, name_shown = $7,
		       email_level = $8, username_level = $9, name_level = $10, roles_level = $11, updated_at = $12
		WHERE id = $1 AND organization_id = $2`,
		p.ID, org.ID, p.Username, p.FirstName, p.MiddleName, p.LastName, p.chosen,
		p.EmailLevel, p.UsernameLevel, p.NameLevel, p.RolesLevel, s.now())
	if isUniqueViolation(err) {
		return FieldErrors{{"username", "That username is taken in " + org.Name + ". Choose another."}}
	}
	if err != nil {
		return err
	}
	if err := audit(ctx, tx, org.ID, p.ID, action, detail); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ErrWrongPassword refuses a change that asks for the current password.
var ErrWrongPassword = errors.New("That is not your current password.")

// ChangeEmail is A8's Change email address (8B.3): the new address and the
// current password, no link sent, unique within the organization, kept in
// the audit record.
func (s *Service) ChangeEmail(ctx context.Context, org *Organization, accountID, newEmail, pw string) error {
	newEmail, err := normalizeEmail(newEmail)
	if err != nil {
		return FieldErrors{{"email", "Enter an email address, such as dana@example.org."}}
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var old string
	var hash *string
	err = tx.QueryRow(ctx, `SELECT email, password_hash FROM accounts WHERE id = $1 AND organization_id = $2 FOR UPDATE`,
		accountID, org.ID).Scan(&old, &hash)
	if err != nil {
		return err
	}
	if hash == nil || !password.Matches(*hash, pw) {
		return FieldErrors{{"password", ErrWrongPassword.Error()}}
	}
	_, err = tx.Exec(ctx, `UPDATE accounts SET email = $3, updated_at = $4 WHERE id = $1 AND organization_id = $2`,
		accountID, org.ID, newEmail, s.now())
	if isUniqueViolation(err) {
		return FieldErrors{{"email", "Another account in " + org.Name + " has this email address."}}
	}
	if err != nil {
		return err
	}
	if err := audit(ctx, tx, org.ID, accountID, "email.changed", map[string]any{"from": old, "to": newEmail}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Shared is another person's profile as the viewer may see it (A9). Both
// are in the organization, so the viewer sees what is public or for
// organization members. Event participants and station members are added
// with events and stations (Phase 3); until then nobody shares one.
func (s *Service) Shared(ctx context.Context, org *Organization, viewerID, personID string) (*SharedProfile, error) {
	p, err := s.GetProfile(ctx, org, personID)
	if err != nil {
		return nil, err
	}
	sees := func(level string) bool { return level == LevelPublic || level == LevelOrganization }
	out := &SharedProfile{ID: p.ID, Name: p.Name}
	if sees(p.NameLevel) {
		out.PersonalName = p.personalName()
	}
	if sees(p.UsernameLevel) {
		out.Username = p.Username
	}
	if sees(p.EmailLevel) {
		out.Email = &p.Email
	}
	if sees(p.RolesLevel) && len(p.Roles) > 0 {
		out.Roles = p.Roles
	}
	return out, nil
}
