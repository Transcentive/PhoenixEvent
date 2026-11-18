package accounts

import (
	"context"
	"errors"
	"testing"
)

func confirmed(t *testing.T, s *Service, org *Organization, email, username string) *Account {
	t.Helper()
	a := register(t, s, org, email, username)
	if _, err := s.db.Exec(context.Background(), `UPDATE accounts SET email_confirmed_at = now() WHERE id = $1`, a.ID); err != nil {
		t.Fatal(err)
	}
	return a
}

// Board first (5E.1, S7): Organization Leadership appoints board members,
// then records the chair they chose; the chair then appoints, edits and
// removes. Capacity holds.
func TestBoardFirst(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	admin := adminOf(t, s, org)
	lead := confirmed(t, s, org, "lead@example.org", "lead")
	dana := confirmed(t, s, org, "dana@example.org", "dreyes")
	lee := confirmed(t, s, org, "lee@example.org", "lee")
	sam := confirmed(t, s, org, "sam@example.org", "sam")
	if err := s.SetRoles(ctx, org, admin, lead.ID, []string{RoleLeadership}); err != nil {
		t.Fatal(err)
	}
	e, _ := s.CreateEvent(ctx, org, admin, NewEvent{Name: "Valentine Dance", BoardCapacity: "2", BoardOrder: "board_first"})

	if err := s.Appoint(ctx, org, admin, e.ID, dana.ID, PositionMember); !errors.Is(err, ErrForbidden) {
		t.Errorf("an Organization Administrator without Leadership appointed: %v", err)
	}
	if err := s.Appoint(ctx, org, lead.ID, e.ID, dana.ID, PositionChair); !errors.Is(err, ErrForbidden) {
		t.Errorf("appointed a chair on a board-first board: %v", err)
	}
	for _, p := range []string{dana.ID, lee.ID} {
		if err := s.Appoint(ctx, org, lead.ID, e.ID, p, PositionMember); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Appoint(ctx, org, lead.ID, e.ID, sam.ID, PositionMember); fieldOf(err) != "person" {
		t.Errorf("appointed beyond capacity: %v", err)
	}
	if err := s.Appoint(ctx, org, lead.ID, e.ID, dana.ID, PositionMember); fieldOf(err) != "person" {
		t.Errorf("appointed twice: %v", err)
	}
	if ok, _ := s.canEditEvent(ctx, org, dana.ID, e.ID); ok {
		t.Error("a board member may edit")
	}
	if err := s.RecordChair(ctx, org, lead.ID, e.ID, sam.ID); fieldOf(err) != "person" {
		t.Errorf("recorded a chair off the board: %v", err)
	}
	if err := s.RecordChair(ctx, org, lead.ID, e.ID, dana.ID); err != nil {
		t.Fatal(err)
	}
	b, err := s.GetBoard(ctx, org, dana.ID, e.ID)
	if err != nil || !b.Formed || b.SeatsTaken != 2 || !b.CanAppointMembers || b.CanRecordChair || !b.Event.CanEdit {
		t.Errorf("board after the chair = %+v, %v", b, err)
	}
	// The chair edits the event; capacity cannot fall below the seats taken.
	if _, err := s.UpdateEvent(ctx, org, dana.ID, e.ID, NewEvent{Name: "Valentine Dance", BoardCapacity: "1"}); fieldOf(err) != "board_capacity" {
		t.Errorf("lowered capacity below the seats: %v", err)
	}
	if _, err := s.UpdateEvent(ctx, org, dana.ID, e.ID, NewEvent{Name: "Valentine Dance", BoardCapacity: "3"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Appoint(ctx, org, lead.ID, e.ID, sam.ID, PositionMember); !errors.Is(err, ErrForbidden) {
		t.Errorf("Leadership appointed after the chair exists: %v", err)
	}
	if err := s.Appoint(ctx, org, dana.ID, e.ID, sam.ID, PositionMember); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveFromBoard(ctx, org, dana.ID, e.ID, sam.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("the chair removed a Board Member alone: %v", err)
	}
	// Termination of a Board Member: the Event Chair and
	// Organization Leadership both agree (5D.3).
	if err := s.StartRemoval(ctx, org, lee.ID, e.ID, sam.ID, ""); !errors.Is(err, ErrForbidden) {
		t.Errorf("a Board Member started a termination once the chair exists: %v", err)
	}
	if err := s.StartRemoval(ctx, org, dana.ID, e.ID, sam.ID, ""); err != nil {
		t.Fatal(err)
	}
	b, _ = s.GetBoard(ctx, org, lead.ID, e.ID)
	if len(b.Removals) != 1 || !b.Removals[0].CanApprove || b.Removals[0].NeedsVote {
		t.Fatalf("removals = %+v", b.Removals)
	}
	if err := s.DecideRemoval(ctx, org, lead.ID, e.ID, b.Removals[0].ID, true); err != nil {
		t.Fatal(err)
	}
	if p, _ := positionOf(ctx, s.db, e.ID, sam.ID); p != "" {
		t.Errorf("after both agreed, Sam is still %q", p)
	}
	// A plain member reads the board and may do nothing on it.
	b, err = s.GetBoard(ctx, org, sam.ID, e.ID)
	if err != nil || len(b.Seats) != 2 || b.CanAppointMembers || b.CanAppointAdmin || b.Event.CanEdit {
		t.Errorf("a member's view = %+v, %v", b, err)
	}
}

// Chair first (5E.1, S8), the Event Administrator (7B.7) and deleting an
// event (CRUD).
func TestChairFirstAndEventAdministrator(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	admin := adminOf(t, s, org)
	lead := confirmed(t, s, org, "lead@example.org", "lead")
	dana := confirmed(t, s, org, "dana@example.org", "dreyes")
	maria := confirmed(t, s, org, "maria@example.org", "maria")
	lee := confirmed(t, s, org, "lee@example.org", "lee")
	s.SetRoles(ctx, org, admin, lead.ID, []string{RoleLeadership})
	e, _ := s.CreateEvent(ctx, org, admin, NewEvent{Name: "Spring Fair", BoardCapacity: "2", BoardOrder: "chair_first"})

	if err := s.Appoint(ctx, org, lead.ID, e.ID, lee.ID, PositionMember); !errors.Is(err, ErrForbidden) {
		t.Errorf("appointed a member before the chair: %v", err)
	}
	if err := s.Appoint(ctx, org, lead.ID, e.ID, dana.ID, PositionChair); err != nil {
		t.Fatal(err)
	}
	// The Event Chair edits before the board is formed.
	if ok, _ := s.canEditEvent(ctx, org, dana.ID, e.ID); !ok {
		t.Error("the Event Chair may not edit before the board is formed")
	}
	// No seat by default (7B.7): the Event Administrator does not count.
	if err := s.Appoint(ctx, org, dana.ID, e.ID, maria.ID, PositionAdmin); !errors.Is(err, ErrForbidden) {
		t.Errorf("the chair appointed the Event Administrator: %v", err)
	}
	if err := s.Appoint(ctx, org, lead.ID, e.ID, maria.ID, PositionAdmin); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.canEditEvent(ctx, org, maria.ID, e.ID); !ok {
		t.Error("the Event Administrator may not edit")
	}
	if err := s.Appoint(ctx, org, dana.ID, e.ID, lee.ID, PositionMember); err != nil {
		t.Fatal(err)
	}
	b, _ := s.GetBoard(ctx, org, lead.ID, e.ID)
	if b.SeatsTaken != 2 || !b.Formed || len(b.Seats) != 3 {
		t.Errorf("board = %+v", b)
	}
	// Replacing the Event Administrator.
	if err := s.Appoint(ctx, org, lead.ID, e.ID, admin, PositionAdmin); err != nil {
		t.Fatal(err)
	}
	if p, _ := positionOf(ctx, s.db, e.ID, maria.ID); p != "" {
		t.Errorf("the former Event Administrator is still %q", p)
	}

	if err := s.DeleteEvent(ctx, org, lee.ID, e.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("a board member deleted the event: %v", err)
	}
	if err := s.DeleteEvent(ctx, org, dana.ID, e.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.Events(ctx, org, admin); len(list) != 0 {
		t.Errorf("events after delete = %+v", list)
	}
}

// D2, Replace chair (5D.3, owner's rulings of November 18, 2026): Board
// Members have first right of refusal, then the organization at large (event
// participants come with registration); Organization Leadership records the
// answers and chooses; a chair who resigns stays a Board Member; the board's
// size does not change.
func TestReplaceChair(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	admin := adminOf(t, s, org)
	lead := confirmed(t, s, org, "lead@example.org", "lead")
	dana := confirmed(t, s, org, "dana@example.org", "dreyes")
	lee := confirmed(t, s, org, "lee@example.org", "lee")
	sam := confirmed(t, s, org, "sam@example.org", "sam")
	kim := confirmed(t, s, org, "kim@example.org", "kim")
	s.SetRoles(ctx, org, admin, lead.ID, []string{RoleLeadership})
	e, _ := s.CreateEvent(ctx, org, admin, NewEvent{Name: "Spring Fair", BoardCapacity: "3", BoardOrder: "chair_first"})
	s.Appoint(ctx, org, lead.ID, e.ID, dana.ID, PositionChair)
	s.Appoint(ctx, org, dana.ID, e.ID, lee.ID, PositionMember)
	s.Appoint(ctx, org, dana.ID, e.ID, sam.ID, PositionMember)

	if b, _ := s.GetBoard(ctx, org, lead.ID, e.ID); !b.CanReplaceChair {
		t.Error("Organization Leadership may not replace the chair")
	}
	if b, _ := s.GetBoard(ctx, org, dana.ID, e.ID); b.CanReplaceChair {
		t.Error("the Event Chair may replace the chair")
	}
	req := ReplaceChairRequest{Why: ReplaceResigned, Answers: map[string]bool{lee.ID: true, sam.ID: true}, Person: lee.ID}
	if err := s.ReplaceChair(ctx, org, dana.ID, e.ID, req); !errors.Is(err, ErrForbidden) {
		t.Errorf("the Event Chair replaced the chair: %v", err)
	}
	// Every Board Member must have answered.
	if err := s.ReplaceChair(ctx, org, lead.ID, e.ID, ReplaceChairRequest{Why: ReplaceResigned, Answers: map[string]bool{lee.ID: true}, Person: lee.ID}); fieldOf(err) != "answers" {
		t.Errorf("missing answer = %v", err)
	}
	// While a Board Member accepts, the organization at large waits.
	if err := s.ReplaceChair(ctx, org, lead.ID, e.ID, ReplaceChairRequest{Why: ReplaceResigned, Answers: map[string]bool{lee.ID: true, sam.ID: false}, Person: kim.ID}); fieldOf(err) != "person" {
		t.Errorf("went past a Board Member who accepted: %v", err)
	}
	// Several accept: Organization Leadership chooses. Dana resigns and stays.
	if err := s.ReplaceChair(ctx, org, lead.ID, e.ID, req); err != nil {
		t.Fatal(err)
	}
	if p, _ := positionOf(ctx, s.db, e.ID, lee.ID); p != PositionChair {
		t.Errorf("Lee is %q", p)
	}
	if p, _ := positionOf(ctx, s.db, e.ID, dana.ID); p != PositionMember {
		t.Errorf("Dana, who resigned, is %q", p)
	}
	// Every Board Member declines: the organization at large. The board is
	// full and Lee resigns to stay on it, so its size holds.
	all := map[string]bool{dana.ID: false, sam.ID: false}
	if err := s.ReplaceChair(ctx, org, lead.ID, e.ID, ReplaceChairRequest{Why: ReplaceResigned, Answers: all, Person: kim.ID}); fieldOf(err) != "person" {
		t.Errorf("full board = %v", err)
	}
	// Illness: Lee leaves the board, and Kim takes the seat.
	if err := s.ReplaceChair(ctx, org, lead.ID, e.ID, ReplaceChairRequest{Why: ReplaceIllness, Answers: all, Person: kim.ID}); err != nil {
		t.Fatal(err)
	}
	if p, _ := positionOf(ctx, s.db, e.ID, lee.ID); p != "" {
		t.Errorf("Lee, replaced because of illness, is %q", p)
	}
	if p, _ := positionOf(ctx, s.db, e.ID, kim.ID); p != PositionChair {
		t.Errorf("Kim is %q", p)
	}
	if err := s.ReplaceChair(ctx, org, lead.ID, e.ID, ReplaceChairRequest{Why: "bored", Answers: all, Person: dana.ID}); fieldOf(err) != "why" {
		t.Errorf("why = %v", err)
	}
}
