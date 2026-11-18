package accounts

import (
	"context"
	"errors"
	"testing"
)

// O3 (16B.6, 7C.4, 7B.7): starting values, only an Organization
// Administrator opens or saves it, and a save is audited.
func TestPolicies(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	admin := adminOf(t, s, org)
	dana := register(t, s, org, "dana@example.org", "dreyes")

	if _, err := s.GetPolicies(ctx, org, dana.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("a member opened O3: %v", err)
	}
	p, err := s.GetPolicies(ctx, org, admin)
	if err != nil || *p != startingPolicies || p.EventAdminHoldsSeat {
		t.Fatalf("starting policies = %+v, %v", p, err)
	}
	want := Policies{ApproveRegistrations: false, Participation: "registrants", EventAdminHoldsSeat: true}
	if _, err := s.SetPolicies(ctx, org, dana.ID, want); !errors.Is(err, ErrForbidden) {
		t.Errorf("a member saved O3: %v", err)
	}
	if _, err := s.SetPolicies(ctx, org, admin, Policies{Participation: "members"}); fieldOf(err) != "participation" {
		t.Errorf("an unknown answer to who participates: %v", err)
	}
	if _, err := s.SetPolicies(ctx, org, admin, want); err != nil {
		t.Fatal(err)
	}
	if p, _ := s.GetPolicies(ctx, org, admin); *p != want {
		t.Errorf("saved policies = %+v", p)
	}
	var audits int
	s.db.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE action = 'policies.changed' AND organization_id = $1`, org.ID).Scan(&audits)
	if audits != 1 {
		t.Errorf("audit records = %d", audits)
	}

	// Another organization keeps its own starting values.
	other, _ := s.CreateOrganization(ctx, "Riverside Community Committee", "riverside", "admin@example.org")
	if p, _ := s.GetPolicies(ctx, other, adminOf(t, s, other)); *p != startingPolicies {
		t.Errorf("another organization's policies = %+v", p)
	}
}

// S1 (16B.6, 5A.4, 5E): only an Organization Administrator creates; name,
// a board of at least 1 and the order are required; the event is private and
// listed only in its own organization.
func TestCreateEvent(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	admin := adminOf(t, s, org)
	dana := register(t, s, org, "dana@example.org", "dreyes")

	ok := NewEvent{Name: " Halloween Bash ", BoardCapacity: "5", BoardOrder: "board_first"}
	if _, err := s.CreateEvent(ctx, org, dana.ID, ok); !errors.Is(err, ErrForbidden) {
		t.Errorf("a member created an event: %v", err)
	}

	_, err := s.CreateEvent(ctx, org, admin, NewEvent{Name: " ", Date: "31/10/2026", BoardCapacity: "0"})
	var fields FieldErrors
	if !errors.As(err, &fields) {
		t.Fatalf("expected field errors, got %v", err)
	}
	got := map[string]bool{}
	for _, f := range fields {
		got[f.Field] = true
	}
	for _, f := range []string{"name", "date", "board_capacity", "board_order"} {
		if !got[f] {
			t.Errorf("no error for %s: %v", f, fields)
		}
	}
	if _, err := s.CreateEvent(ctx, org, admin, NewEvent{Name: "X", BoardCapacity: "2.5", BoardOrder: "chair_first"}); fieldOf(err) != "board_capacity" {
		t.Errorf("a fraction of a seat: %v", err)
	}

	e, err := s.CreateEvent(ctx, org, admin, ok)
	if err != nil {
		t.Fatal(err)
	}
	if e.Name != "Halloween Bash" || e.Visibility != "private" || e.BoardCapacity != 5 || e.Date != nil || e.Location != nil {
		t.Errorf("created %+v", e)
	}
	if _, err := s.CreateEvent(ctx, org, admin, NewEvent{Name: "Spring Fair", Date: "2027-04-17", Location: "Gym",
		BoardCapacity: "3", BoardOrder: "chair_first"}); err != nil {
		t.Fatal(err)
	}
	list, err := s.Events(ctx, org, admin)
	if err != nil || len(list) != 2 || list[0].Name != "Spring Fair" || *list[0].Date != "2027-04-17" || list[0].BoardOrder != "chair_first" {
		t.Errorf("events = %+v, %v", list, err)
	}
	other, _ := s.CreateOrganization(ctx, "Riverside Community Committee", "riverside", "admin@example.org")
	if list, _ := s.Events(ctx, other, adminOf(t, s, other)); len(list) != 0 {
		t.Errorf("another organization sees %+v", list)
	}
}

// Editing an event (owner's ruling of November 9, 2026): an Organization
// Administrator may; a member may not; the order stays as created (5E.4);
// another organization's event is out of reach.
func TestUpdateEvent(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	admin := adminOf(t, s, org)
	dana := register(t, s, org, "dana@example.org", "dreyes")
	e, err := s.CreateEvent(ctx, org, admin, NewEvent{Name: "Turkey Triathlon", BoardCapacity: "5", BoardOrder: "board_first"})
	if err != nil {
		t.Fatal(err)
	}

	if list, _ := s.Events(ctx, org, dana.ID); len(list) != 1 || list[0].CanEdit {
		t.Errorf("a member may edit: %+v", list)
	}
	if list, _ := s.Events(ctx, org, admin); len(list) != 1 || !list[0].CanEdit {
		t.Errorf("an administrator may not edit: %+v", list)
	}
	if _, err := s.GetEvent(ctx, org, dana.ID, e.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("a member opened Edit event: %v", err)
	}
	change := NewEvent{Name: "Turkey Trot", Date: "2026-11-28", Location: "High School Parking Lot", BoardCapacity: "7", BoardOrder: "chair_first"}
	if _, err := s.UpdateEvent(ctx, org, dana.ID, e.ID, change); !errors.Is(err, ErrForbidden) {
		t.Errorf("a member edited: %v", err)
	}
	if _, err := s.UpdateEvent(ctx, org, admin, e.ID, NewEvent{Name: "", BoardCapacity: "0"}); fieldOf(err) == "" {
		t.Errorf("an empty name was saved: %v", err)
	}
	got, err := s.UpdateEvent(ctx, org, admin, e.ID, change)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Turkey Trot" || *got.Date != "2026-11-28" || got.BoardCapacity != 7 || got.BoardOrder != "board_first" {
		t.Errorf("updated %+v", got)
	}
	if again, _ := s.GetEvent(ctx, org, admin, e.ID); again.Name != "Turkey Trot" || again.BoardOrder != "board_first" {
		t.Errorf("stored %+v", again)
	}
	var audits int
	s.db.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE action = 'event.changed'`).Scan(&audits)
	if audits != 1 {
		t.Errorf("audit records = %d", audits)
	}

	other, _ := s.CreateOrganization(ctx, "Riverside Community Committee", "riverside", "admin@example.org")
	if _, err := s.GetEvent(ctx, other, adminOf(t, s, other), e.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("another organization opened the event: %v", err)
	}
	if _, err := s.GetEvent(ctx, org, admin, "not-an-id"); !errors.Is(err, ErrNotFound) {
		t.Errorf("a malformed id: %v", err)
	}
}
