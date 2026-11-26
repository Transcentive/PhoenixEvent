package accounts

import (
	"context"
	"slices"
	"testing"
)

// O2, Members (4A.10): those taking part in an event, which an
// appointment makes them (5D.1), with their roles; email and roles follow the
// levels the member chose (8B.4).
func TestMembers(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	admin := adminOf(t, s, org)
	lead := confirmed(t, s, org, "lead@example.org", "lead")
	dana := confirmed(t, s, org, "dana@example.org", "dana")
	lee := confirmed(t, s, org, "lee@example.org", "lee")
	sam := confirmed(t, s, org, "sam@example.org", "sam")
	confirmed(t, s, org, "pat@example.org", "pat")
	if err := s.SetRoles(ctx, org, admin, lead.ID, []string{RoleLeadership}); err != nil {
		t.Fatal(err)
	}
	dance, _ := s.CreateEvent(ctx, org, admin, NewEvent{Name: "Valentine Dance", BoardCapacity: "3", BoardOrder: "board_first"})
	fair, _ := s.CreateEvent(ctx, org, admin, NewEvent{Name: "Book Fair", BoardCapacity: "3", BoardOrder: "board_first"})
	for _, a := range []struct{ event, person string }{{dance.ID, dana.ID}, {dance.ID, lee.ID}, {fair.ID, sam.ID}} {
		if err := s.Appoint(ctx, org, lead.ID, a.event, a.person, PositionMember); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RecordChair(ctx, org, lead.ID, dance.ID, dana.ID); err != nil {
		t.Fatal(err)
	}

	list, err := s.Members(ctx, org, admin)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, m := range list {
		names = append(names, m.Name)
	}
	if !slices.Equal(names, []string{"dana", "lee", "sam"}) {
		t.Fatalf("listed %v; want only those on a board", names)
	}
	if !slices.Equal(list[0].Roles, []string{"Event Chair, Valentine Dance"}) {
		t.Errorf("dana = %+v", list[0])
	}

	// Dana shows her roles and email to event participants only.
	if _, err := s.db.Exec(ctx, `UPDATE accounts SET roles_level = 'event', email_level = 'event', name_shown = 'username' WHERE id = $1`, dana.ID); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		viewer    string
		roles     int
		withEmail bool
	}{{lee.ID, 1, true}, {sam.ID, 0, false}, {dana.ID, 1, true}} {
		list, _ := s.Members(ctx, org, c.viewer)
		d := list[0]
		if len(d.Roles) != c.roles || (d.Email != nil) != c.withEmail {
			t.Errorf("viewer %s sees dana as %+v", c.viewer, d)
		}
	}
}
