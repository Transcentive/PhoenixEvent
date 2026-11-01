package accounts

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func adminOf(t *testing.T, s *Service, org *Organization) string {
	t.Helper()
	var id string
	if err := s.db.QueryRow(context.Background(),
		`SELECT id FROM accounts WHERE organization_id = $1 AND email = 'admin@example.org'`, org.ID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// 7B.6 and v1.07: only an Organization Administrator changes roles, and
// the last one is never removed.
func TestSetRoles(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	admin := adminOf(t, s, org)
	dana := register(t, s, org, "dana@example.org", "dreyes")
	lee := register(t, s, org, "lee@example.org", "lee")

	if err := s.SetRoles(ctx, org, dana.ID, lee.ID, []string{RoleLeadership}); !errors.Is(err, ErrForbidden) {
		t.Errorf("a member changed roles: %v", err)
	}
	if _, err := s.RolesFor(ctx, org, dana.ID, lee.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("a member opened Change roles: %v", err)
	}
	if err := s.SetRoles(ctx, org, admin, dana.ID, []string{"system_administrator"}); fieldOf(err) != "roles" {
		t.Errorf("a role outside the organization's: %v", err)
	}

	// The only administrator cannot remove the role from themselves.
	if err := s.SetRoles(ctx, org, admin, admin, []string{RoleLeadership}); fieldOf(err) != "roles" {
		t.Errorf("removed the last administrator: %v", err)
	}

	if err := s.SetRoles(ctx, org, admin, dana.ID, []string{RoleAdministrator, RoleLeadership}); err != nil {
		t.Fatal(err)
	}
	if r, _ := s.RolesFor(ctx, org, admin, dana.ID); !slices.Equal(r.Roles, []string{RoleAdministrator, RoleLeadership}) || r.Name != "dreyes" {
		t.Errorf("Dana's roles = %+v", r)
	}
	// With a second administrator, the first may step down.
	if err := s.SetRoles(ctx, org, admin, admin, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.SetRoles(ctx, org, admin, lee.ID, []string{RoleLeadership}); !errors.Is(err, ErrForbidden) {
		t.Errorf("a former administrator changed roles: %v", err)
	}
	if err := s.SetRoles(ctx, org, dana.ID, dana.ID, nil); fieldOf(err) != "roles" {
		t.Errorf("Dana removed herself as the last administrator: %v", err)
	}

	officers, err := s.Officers(ctx, org)
	if err != nil {
		t.Fatal(err)
	}
	if len(officers) != 1 || officers[0].ID != dana.ID ||
		!slices.Equal(officers[0].Roles, []string{"Organization Administrator", "Organization Leadership"}) {
		t.Errorf("officers = %+v", officers)
	}

	var changes int
	_ = s.db.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE organization_id = $1 AND action = 'roles.changed'`, org.ID).Scan(&changes)
	if changes != 2 {
		t.Errorf("audit records = %d, want 2", changes)
	}
}

// Roles in one organization give nothing in another (4).
func TestRolesStayInTheirOrganization(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	pta, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	riverside, _ := s.CreateOrganization(ctx, "Riverside Community Committee", "riverside", "chair@example.org")
	admin := adminOf(t, s, pta)
	sam := register(t, s, riverside, "sam@example.org", "")

	if err := s.SetRoles(ctx, riverside, admin, sam.ID, []string{RoleAdministrator}); !errors.Is(err, ErrForbidden) {
		t.Errorf("a PTA administrator changed roles at Riverside: %v", err)
	}
	if err := s.SetRoles(ctx, pta, admin, sam.ID, []string{RoleAdministrator}); !errors.Is(err, ErrNotFound) {
		t.Errorf("a Riverside account was found at the PTA: %v", err)
	}
	officers, _ := s.Officers(ctx, riverside)
	if len(officers) != 1 || officers[0].Name != "chair@example.org" {
		t.Errorf("Riverside officers = %+v", officers)
	}
}
