package accounts

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func register(t *testing.T, s *Service, org *Organization, addr, username string) *Account {
	t.Helper()
	acc, err := s.Register(context.Background(), org, addr, username, goodPassword)
	if err != nil {
		t.Fatal(err)
	}
	return acc
}

func fieldOf(err error) string {
	var fields FieldErrors
	if !errors.As(err, &fields) || len(fields) == 0 {
		return ""
	}
	return fields[0].Field
}

// 8B.2 and 8B.4, with the starting levels of v1.04.
func TestProfileNameShownAndStartingLevels(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")

	dana := register(t, s, org, "dana@example.org", "")
	p, err := s.GetProfile(ctx, org, dana.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "dana@example.org" || p.EmailLevel != LevelOrganization {
		t.Errorf("email only: name %q, email level %q", p.Name, p.EmailLevel)
	}
	if p.UsernameLevel != LevelOrganization || p.NameLevel != LevelOrganization || p.RolesLevel != LevelOrganization {
		t.Errorf("starting levels = %+v", p)
	}
	lee := register(t, s, org, "lee@example.org", "lee")
	if p, _ := s.GetProfile(ctx, org, lee.ID); p.Name != "lee" || p.EmailLevel != LevelPrivate {
		t.Errorf("with a username: name %q, email level %q", p.Name, p.EmailLevel)
	}

	if p, err = s.UpdateProfile(ctx, org, dana.ID, ProfileChange{Username: "dreyes", FirstName: "Dana", LastName: "Reyes"}); err != nil {
		t.Fatal(err)
	}
	if p.Name != "Dana Reyes" || p.NameShown != ShownPersonalName {
		t.Errorf("after a personal name: %q (%s)", p.Name, p.NameShown)
	}
	vis := VisibilityChange{NameShown: ShownUsername, EmailLevel: LevelPrivate, UsernameLevel: LevelOrganization, NameLevel: LevelOrganization, RolesLevel: LevelOrganization}
	if p, err = s.UpdateVisibility(ctx, org, dana.ID, vis); err != nil || p.Name != "dreyes" {
		t.Errorf("chose the username: %q %v", p.Name, err)
	}
	// Clearing the chosen username hands the choice back to 8B.2.
	if p, err = s.UpdateProfile(ctx, org, dana.ID, ProfileChange{FirstName: "Dana", LastName: "Reyes"}); err != nil || p.Name != "Dana Reyes" {
		t.Errorf("cleared the username: %v %v", p, err)
	}
	// Clearing the only visible name is refused: the email address is private.
	if _, err := s.UpdateProfile(ctx, org, dana.ID, ProfileChange{}); fieldOf(err) != "profile" {
		t.Errorf("cleared every visible name: %v", err)
	}
}

func TestProfileRefusals(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	register(t, s, org, "lee@example.org", "lee")
	dana := register(t, s, org, "dana@example.org", "")

	profile := map[string]struct {
		change ProfileChange
		field  string
	}{
		"a first name alone":         {ProfileChange{FirstName: "Dana"}, "last_name"},
		"a username with @":          {ProfileChange{Username: "d@r"}, "username"},
		"a username taken, any case": {ProfileChange{Username: "LEE"}, "username"},
	}
	for name, tc := range profile {
		if _, err := s.UpdateProfile(ctx, org, dana.ID, tc.change); fieldOf(err) != tc.field {
			t.Errorf("%s: %v, want a problem with %s", name, err, tc.field)
		}
	}
	ok := VisibilityChange{EmailLevel: LevelOrganization, UsernameLevel: LevelOrganization, NameLevel: LevelOrganization, RolesLevel: LevelOrganization}
	visibility := map[string]struct {
		change func(*VisibilityChange)
		field  string
	}{
		"no such level":                {func(c *VisibilityChange) { c.RolesLevel = "friends" }, "roles_level"},
		"showing a name not saved":     {func(c *VisibilityChange) { c.NameShown = ShownUsername }, "name_shown"},
		"the only name made private":   {func(c *VisibilityChange) { c.EmailLevel = LevelPrivate }, "name_shown"},
		"the name shown for a station": {func(c *VisibilityChange) { c.EmailLevel = LevelStation }, "name_shown"},
	}
	for name, tc := range visibility {
		c := ok
		tc.change(&c)
		if _, err := s.UpdateVisibility(ctx, org, dana.ID, c); fieldOf(err) != tc.field {
			t.Errorf("%s: %v, want a problem with %s", name, err, tc.field)
		}
	}
	if p, _ := s.GetProfile(ctx, org, dana.ID); p.Username != nil || p.FirstName != nil || p.EmailLevel != LevelOrganization {
		t.Errorf("a refused change was written: %+v", p)
	}
}

func TestChangeEmail(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	register(t, s, org, "lee@example.org", "")
	dana := register(t, s, org, "dana@example.org", "")

	if err := s.ChangeEmail(ctx, org, dana.ID, "dana@new.example", "not the password"); fieldOf(err) != "password" {
		t.Errorf("wrong password: %v", err)
	}
	if err := s.ChangeEmail(ctx, org, dana.ID, "LEE@example.org", goodPassword); fieldOf(err) != "email" {
		t.Errorf("another account's address: %v", err)
	}
	if err := s.ChangeEmail(ctx, org, dana.ID, "dana@new.example", goodPassword); err != nil {
		t.Fatal(err)
	}
	if p, _ := s.GetProfile(ctx, org, dana.ID); p.Email != "dana@new.example" {
		t.Errorf("email = %q", p.Email)
	}
	var n int
	_ = s.db.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE action = 'email.changed' AND account_id = $1`, dana.ID).Scan(&n)
	if n != 1 {
		t.Errorf("audit records = %d, want 1", n)
	}
}

// A9 shows what is visible to organization members, never what is private
// or kept for an event or station, and never a person of another
// organization.
func TestSharedProfile(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	other, _ := s.CreateOrganization(ctx, "Riverside Community Committee", "riverside", "admin2@example.org")
	lee := register(t, s, org, "lee@example.org", "lee")
	dana := register(t, s, org, "dana@example.org", "")
	stranger := register(t, s, other, "sam@example.org", "")

	if _, err := s.UpdateProfile(ctx, org, dana.ID, ProfileChange{Username: "dreyes", FirstName: "Dana", LastName: "Reyes"}); err != nil {
		t.Fatal(err)
	}
	_, err := s.UpdateVisibility(ctx, org, dana.ID, VisibilityChange{
		EmailLevel: LevelPrivate, UsernameLevel: LevelStation, NameLevel: LevelOrganization, RolesLevel: LevelPublic})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Shared(ctx, org, lee.ID, dana.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Dana Reyes" || got.PersonalName == nil || got.Email != nil || got.Username != nil {
		t.Errorf("shared view = %+v", got)
	}
	if _, err := s.Shared(ctx, org, lee.ID, stranger.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("another organization's person: %v", err)
	}
	if _, err := s.Shared(ctx, org, lee.ID, "not-an-id"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a malformed id: %v", err)
	}
	if strings.Contains(got.Name, "@") {
		t.Error("the email address shows as the name")
	}
}
