package accounts

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Transcentive/PhoenixEvent/backend/internal/config"
	"github.com/Transcentive/PhoenixEvent/backend/internal/email"
	"github.com/Transcentive/PhoenixEvent/backend/internal/testdb"
	"github.com/Transcentive/PhoenixEvent/backend/internal/token"
)

// outbox keeps every message instead of sending it.
type outbox struct {
	mu   sync.Mutex
	sent []email.Message
}

func (o *outbox) Send(_ context.Context, m email.Message) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.sent = append(o.sent, m)
	return nil
}

var tokenInLink = regexp.MustCompile(`token=([0-9a-f]{64})`)

// lastToken returns the token in the newest message to addr.
func (o *outbox) lastToken(t *testing.T, addr string) string {
	t.Helper()
	o.mu.Lock()
	defer o.mu.Unlock()
	for i := len(o.sent) - 1; i >= 0; i-- {
		if o.sent[i].To == addr {
			if m := tokenInLink.FindStringSubmatch(o.sent[i].Text); m != nil {
				return m[1]
			}
		}
	}
	t.Fatalf("no link was emailed to %s", addr)
	return ""
}

func (o *outbox) count(addr string) int {
	o.mu.Lock()
	defer o.mu.Unlock()
	n := 0
	for _, m := range o.sent {
		if m.To == addr {
			n++
		}
	}
	return n
}

var testSite = config.Site{OrgDomain: "phoenixevent.localhost", Scheme: "http", Port: "5174"}

func newTestService(t *testing.T) (*Service, *outbox) {
	t.Helper()
	box := &outbox{}
	return NewService(testdb.Pool(t), box, testSite, token.NewIssuer("0123456789abcdef0123456789abcdef")), box
}

const goodPassword = "velvet tractor oyster pilgrim"

// newConfirmedAccount registers an account at org and opens its link.
func newConfirmedAccount(t *testing.T, s *Service, box *outbox, org *Organization, addr, username string) {
	t.Helper()
	ctx := context.Background()
	if _, err := s.Register(ctx, org, addr, username, goodPassword); err != nil {
		t.Fatalf("register %s: %v", addr, err)
	}
	if err := s.ConfirmEmail(ctx, org, box.lastToken(t, addr)); err != nil {
		t.Fatalf("confirm %s: %v", addr, err)
	}
}

func TestCreateOrganizationAndFirstAdministrator(t *testing.T) {
	s, box := newTestService(t)
	ctx := context.Background()

	org, err := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateOrganization(ctx, "Another", "lincolnpta", "x@example.org"); err == nil {
		t.Error("a second organization took the same address name")
	}
	if _, err := s.CreateOrganization(ctx, "Bad", "-bad", "x@example.org"); err == nil {
		t.Error("an address name starting with a hyphen was accepted")
	}

	// No password yet: the login is refused like any wrong password.
	if _, err := s.Login(ctx, org, "admin@example.org", goodPassword); !errors.Is(err, ErrBadCredentials) {
		t.Fatalf("login before setting a password: %v", err)
	}
	link := box.lastToken(t, "admin@example.org")
	if purpose, login, err := s.LinkPurpose(ctx, org, link); err != nil || purpose != "set_password" || login != "admin@example.org" {
		t.Fatalf("link purpose = %q, %v", purpose, err)
	}
	if err := s.SetPassword(ctx, org, link, goodPassword); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Login(ctx, org, "admin@example.org", goodPassword); err != nil {
		t.Fatalf("login after setting the password: %v", err)
	}

	var role string
	if err := s.db.QueryRow(ctx, `SELECT role FROM organization_roles WHERE organization_id = $1`, org.ID).Scan(&role); err != nil || role != "organization_administrator" {
		t.Errorf("role = %q, %v", role, err)
	}
}

func TestRegisterConfirmAndLogin(t *testing.T) {
	s, box := newTestService(t)
	ctx := context.Background()
	org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")

	_, err := s.Register(ctx, org, "not an address", "da@na", "short")
	var fields FieldErrors
	if !errors.As(err, &fields) || len(fields) != 3 {
		t.Fatalf("a form wrong in all three fields: %v", err)
	}

	if _, err := s.Register(ctx, org, "dana@example.org", "dreyes", goodPassword); err != nil {
		t.Fatal(err)
	}
	// 8A.2: refused until the link is opened, and only after the password
	// is right, so the message gives nothing away.
	if _, err := s.Login(ctx, org, "dreyes", "wrong password, quite long"); !errors.Is(err, ErrBadCredentials) {
		t.Errorf("wrong password before confirming: %v", err)
	}
	if _, err := s.Login(ctx, org, "dreyes", goodPassword); !errors.Is(err, ErrEmailNotConfirmed) {
		t.Errorf("right password before confirming: %v", err)
	}

	link := box.lastToken(t, "dana@example.org")
	if err := s.ConfirmEmail(ctx, org, link); err != nil {
		t.Fatal(err)
	}
	if err := s.ConfirmEmail(ctx, org, link); !errors.Is(err, ErrLinkInvalid) {
		t.Errorf("a link used twice: %v", err)
	}

	// 4A.8: the username and the email address both log in, without regard
	// to capitals.
	for _, login := range []string{"dreyes", "DReyes", "dana@example.org", "Dana@Example.org"} {
		if _, err := s.Login(ctx, org, login, goodPassword); err != nil {
			t.Errorf("login as %q: %v", login, err)
		}
	}

	// Unique within the organization, without regard to capitals.
	_, err = s.Register(ctx, org, "DANA@example.org", "", goodPassword)
	if !errors.As(err, &fields) || fields[0].Field != "email" {
		t.Errorf("a second account with the same email: %v", err)
	}
	_, err = s.Register(ctx, org, "other@example.org", "DREYES", goodPassword)
	if !errors.As(err, &fields) || fields[0].Field != "username" {
		t.Errorf("a second account with the same username: %v", err)
	}
}

// 4A.7: each organization has its own accounts, even under one email address.
func TestAccountsBelongToOneOrganization(t *testing.T) {
	s, box := newTestService(t)
	ctx := context.Background()
	pta, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	riverside, _ := s.CreateOrganization(ctx, "Riverside Community Committee", "riverside", "admin2@example.org")

	newConfirmedAccount(t, s, box, pta, "dana@example.org", "dana")
	if _, err := s.Login(ctx, riverside, "dana@example.org", goodPassword); !errors.Is(err, ErrBadCredentials) {
		t.Fatalf("a PTA account logged in at Riverside: %v", err)
	}
	// The same email address and username may make a second account there.
	newConfirmedAccount(t, s, box, riverside, "dana@example.org", "dana")

	sess, err := s.Login(ctx, pta, "dana", goodPassword)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := s.tokens.Check(sess.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(ctx, pta, claims); err != nil {
		t.Errorf("the PTA token at the PTA: %v", err)
	}
	if _, err := s.Authenticate(ctx, riverside, claims); !errors.Is(err, ErrSessionEnded) {
		t.Errorf("the PTA token was accepted at Riverside: %v", err)
	}
	if _, err := s.Refresh(ctx, riverside, sess.RefreshToken); !errors.Is(err, ErrSessionEnded) {
		t.Errorf("the PTA refresh token was accepted at Riverside: %v", err)
	}

	// A1 emails both, and an address with no account gets nothing.
	if err := s.FindOrganizations(ctx, "dana@example.org"); err != nil {
		t.Fatal(err)
	}
	last := box.sent[len(box.sent)-1]
	if !strings.Contains(last.Text, "http://lincolnpta.phoenixevent.localhost:5174/") || !strings.Contains(last.Text, "http://riverside.phoenixevent.localhost:5174/") {
		t.Errorf("the lookup email lacks a login link:\n%s", last.Text)
	}
	before := len(box.sent)
	if err := s.FindOrganizations(ctx, "nobody@example.org"); err != nil || len(box.sent) != before {
		t.Errorf("an address with no account: %v, %d emails sent", err, len(box.sent)-before)
	}
}

// 8A.2: 5 failures for one username within 15 minutes pause it for 15 minutes.
func TestFailedLoginsPause(t *testing.T) {
	s, box := newTestService(t)
	ctx := context.Background()
	org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	newConfirmedAccount(t, s, box, org, "dana@example.org", "dreyes")

	now := time.Now()
	s.now = func() time.Time { return now }
	// Failures under the username and the email address count together.
	for i, login := range []string{"dreyes", "dreyes", "dana@example.org", "DREYES", "dreyes"} {
		if _, err := s.Login(ctx, org, login, "not the password at all"); !errors.Is(err, ErrBadCredentials) {
			t.Fatalf("failure %d: %v", i+1, err)
		}
	}
	if _, err := s.Login(ctx, org, "dreyes", goodPassword); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("the right password during the pause: %v", err)
	}
	s.now = func() time.Time { return now.Add(16 * time.Minute) }
	if _, err := s.Login(ctx, org, "dreyes", goodPassword); err != nil {
		t.Fatalf("after the pause: %v", err)
	}
}

func TestRefreshLogoutAndReset(t *testing.T) {
	s, box := newTestService(t)
	ctx := context.Background()
	org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	newConfirmedAccount(t, s, box, org, "dana@example.org", "")

	first, err := s.Login(ctx, org, "dana@example.org", goodPassword)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Refresh(ctx, org, first.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Refresh(ctx, org, first.RefreshToken); !errors.Is(err, ErrSessionEnded) {
		t.Errorf("a used refresh token worked again: %v", err)
	}

	// Logging out ends the session at once, though the access token has
	// not expired.
	claims, _ := s.tokens.Check(second.AccessToken)
	if err := s.Logout(ctx, second.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(ctx, org, claims); !errors.Is(err, ErrSessionEnded) {
		t.Errorf("a logged-out session still authenticates: %v", err)
	}

	// A reset ends every session.
	third, _ := s.Login(ctx, org, "dana@example.org", goodPassword)
	if err := s.RequestReset(ctx, org, "DANA@example.org"); err != nil {
		t.Fatal(err)
	}
	link := box.lastToken(t, "dana@example.org")
	if err := s.SetPassword(ctx, org, link, "short"); err == nil {
		t.Error("a short new password was accepted")
	}
	if err := s.SetPassword(ctx, org, link, "a whole new password here"); err != nil {
		t.Fatal(err)
	}
	claims, _ = s.tokens.Check(third.AccessToken)
	if _, err := s.Authenticate(ctx, org, claims); !errors.Is(err, ErrSessionEnded) {
		t.Errorf("a session survived the reset: %v", err)
	}
	if _, err := s.Login(ctx, org, "dana@example.org", "a whole new password here"); err != nil {
		t.Errorf("the new password: %v", err)
	}

	// A reset for an address with no account sends nothing and says nothing.
	before := box.count("nobody@example.org")
	if err := s.RequestReset(ctx, org, "nobody@example.org"); err != nil || box.count("nobody@example.org") != before {
		t.Errorf("reset for an unknown address: %v", err)
	}
}

func TestLinksExpire(t *testing.T) {
	s, box := newTestService(t)
	ctx := context.Background()
	org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	newConfirmedAccount(t, s, box, org, "dana@example.org", "")

	if err := s.RequestReset(ctx, org, "dana@example.org"); err != nil {
		t.Fatal(err)
	}
	link := box.lastToken(t, "dana@example.org")
	now := time.Now()
	s.now = func() time.Time { return now.Add(ResetLifetime + time.Minute) }
	if err := s.SetPassword(ctx, org, link, "a whole new password here"); !errors.Is(err, ErrLinkInvalid) {
		t.Errorf("a reset link after 1 hour: %v", err)
	}

	// Asking again ends the earlier link: only the newest works.
	s.now = time.Now
	_ = s.RequestReset(ctx, org, "dana@example.org")
	older := box.lastToken(t, "dana@example.org")
	_ = s.RequestReset(ctx, org, "dana@example.org")
	if err := s.SetPassword(ctx, org, older, "a whole new password here"); !errors.Is(err, ErrLinkInvalid) {
		t.Errorf("an older link still worked: %v", err)
	}
}
