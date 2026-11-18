package accounts

import (
	"context"
	"errors"
	"testing"
)

// Before an Event Chair exists (5D.3, November 11, 2026): a unanimous board
// vote, the member recused, and an Organization Leadership agent's approval.
func TestRemovalBeforeChair(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	admin := adminOf(t, s, org)
	lead := confirmed(t, s, org, "lead@example.org", "lead")
	dana := confirmed(t, s, org, "dana@example.org", "dreyes")
	lee := confirmed(t, s, org, "lee@example.org", "lee")
	sam := confirmed(t, s, org, "sam@example.org", "sam")
	s.SetRoles(ctx, org, admin, lead.ID, []string{RoleLeadership})
	e, _ := s.CreateEvent(ctx, org, admin, NewEvent{Name: "Valentine Dance", BoardCapacity: "5", BoardOrder: "board_first"})
	for _, p := range []string{dana.ID, lee.ID, sam.ID} {
		if err := s.Appoint(ctx, org, lead.ID, e.ID, p, PositionMember); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.StartRemoval(ctx, org, sam.ID, e.ID, sam.ID, ""); !errors.Is(err, ErrForbidden) {
		t.Errorf("a member started their own removal: %v", err)
	}
	// Dana starts it; her vote counts in favor. Sam is recused.
	if err := s.StartRemoval(ctx, org, dana.ID, e.ID, sam.ID, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.StartRemoval(ctx, org, lee.ID, e.ID, sam.ID, ""); fieldOf(err) != "person" {
		t.Errorf("a second open removal: %v", err)
	}
	b, _ := s.GetBoard(ctx, org, sam.ID, e.ID)
	r := b.Removals[0]
	if r.CanVote || len(r.Waiting) != 1 || len(r.Votes) != 1 {
		t.Errorf("Sam's view of his removal = %+v", r)
	}
	if err := s.VoteOnRemoval(ctx, org, sam.ID, e.ID, r.ID, false); !errors.Is(err, ErrForbidden) {
		t.Errorf("the recused member voted: %v", err)
	}
	if err := s.DecideRemoval(ctx, org, lead.ID, e.ID, r.ID, true); err != nil {
		t.Fatal(err)
	}
	if p, _ := positionOf(ctx, s.db, e.ID, sam.ID); p != PositionMember {
		t.Error("removed before the vote was unanimous")
	}
	// Lee votes against, then changes to in favor while voting is open.
	if err := s.VoteOnRemoval(ctx, org, lee.ID, e.ID, r.ID, false); err != nil {
		t.Fatal(err)
	}
	if err := s.VoteOnRemoval(ctx, org, lee.ID, e.ID, r.ID, true); err != nil {
		t.Fatal(err)
	}
	if p, _ := positionOf(ctx, s.db, e.ID, sam.ID); p != PositionMember {
		t.Error("removed before the voting period closed")
	}
	if err := s.CloseVoting(ctx, org, dana.ID, e.ID, r.ID, nil); !errors.Is(err, ErrForbidden) {
		t.Errorf("a Board Member closed the voting before a chair exists: %v", err)
	}
	if err := s.CloseVoting(ctx, org, lead.ID, e.ID, r.ID, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.VoteOnRemoval(ctx, org, lee.ID, e.ID, r.ID, false); err == nil {
		t.Error("a vote changed after the voting period closed")
	}
	if p, _ := positionOf(ctx, s.db, e.ID, sam.ID); p != "" {
		t.Errorf("after a unanimous vote and approval, Sam is still %q", p)
	}

	// One vote against declines it.
	if err := s.StartRemoval(ctx, org, dana.ID, e.ID, lee.ID, ""); err != nil {
		t.Fatal(err)
	}
	b, _ = s.GetBoard(ctx, org, dana.ID, e.ID)
	id := b.Removals[0].ID
	if err := s.EditRemoval(ctx, org, dana.ID, e.ID, id, "Moved away"); err != nil {
		t.Fatal(err)
	}
	if err := s.WithdrawRemoval(ctx, org, lee.ID, e.ID, id); !errors.Is(err, ErrForbidden) {
		t.Errorf("someone else withdrew it: %v", err)
	}
	if err := s.WithdrawRemoval(ctx, org, dana.ID, e.ID, id); err != nil {
		t.Fatal(err)
	}
	if err := s.Appoint(ctx, org, lead.ID, e.ID, sam.ID, PositionMember); err != nil {
		t.Fatal(err)
	}
	if err := s.StartRemoval(ctx, org, dana.ID, e.ID, lee.ID, ""); err != nil {
		t.Fatal(err)
	}
	b, _ = s.GetBoard(ctx, org, dana.ID, e.ID)
	if err := s.VoteOnRemoval(ctx, org, sam.ID, e.ID, b.Removals[0].ID, false); err != nil {
		t.Fatal(err)
	}
	if err := s.CloseVoting(ctx, org, lead.ID, e.ID, b.Removals[0].ID, nil); err != nil {
		t.Fatal(err)
	}
	if b, _ = s.GetBoard(ctx, org, dana.ID, e.ID); len(b.Removals) != 0 {
		t.Errorf("a vote against left it open after closing: %+v", b.Removals)
	}
	if p, _ := positionOf(ctx, s.db, e.ID, lee.ID); p != PositionMember {
		t.Error("removed despite a vote against")
	}
}

// The Event Chair: a unanimous board vote, the chair recused, and an
// Organization Leadership agent's approval; Leadership may start it. The
// Event Chair alone removes the Event Administrator.
func TestRemovingTheEventChair(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	admin := adminOf(t, s, org)
	lead := confirmed(t, s, org, "lead@example.org", "lead")
	dana := confirmed(t, s, org, "dana@example.org", "dreyes")
	lee := confirmed(t, s, org, "lee@example.org", "lee")
	sam := confirmed(t, s, org, "sam@example.org", "sam")
	maria := confirmed(t, s, org, "maria@example.org", "maria")
	s.SetRoles(ctx, org, admin, lead.ID, []string{RoleLeadership})
	e, _ := s.CreateEvent(ctx, org, admin, NewEvent{Name: "Spring Fair", BoardCapacity: "5", BoardOrder: "chair_first"})
	s.Appoint(ctx, org, lead.ID, e.ID, dana.ID, PositionChair)
	s.Appoint(ctx, org, dana.ID, e.ID, lee.ID, PositionMember)
	s.Appoint(ctx, org, dana.ID, e.ID, sam.ID, PositionMember)
	s.Appoint(ctx, org, lead.ID, e.ID, maria.ID, PositionAdmin)

	if err := s.RemoveFromBoard(ctx, org, lee.ID, e.ID, maria.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("a Board Member removed the Event Administrator: %v", err)
	}
	if err := s.RemoveFromBoard(ctx, org, dana.ID, e.ID, maria.ID); err != nil {
		t.Fatal(err)
	}

	if err := s.StartRemoval(ctx, org, lead.ID, e.ID, dana.ID, ""); err != nil {
		t.Fatal(err)
	}
	b, _ := s.GetBoard(ctx, org, dana.ID, e.ID)
	r := b.Removals[0]
	if !r.NeedsVote || !r.Approved || r.CanVote || len(r.Waiting) != 2 {
		t.Errorf("the chair's view = %+v", r)
	}
	s.VoteOnRemoval(ctx, org, lee.ID, e.ID, r.ID, true)
	if err := s.VoteOnRemoval(ctx, org, sam.ID, e.ID, r.ID, true); err != nil {
		t.Fatal(err)
	}
	// The Deputy Board Chair closes the vote on the chair's removal (7B.8).
	for _, who := range []string{dana.ID, lead.ID, lee.ID} {
		if err := s.CloseVoting(ctx, org, who, e.ID, r.ID, nil); !errors.Is(err, ErrForbidden) {
			t.Errorf("closed the vote on the chair's removal without being the Deputy: %v", err)
		}
	}
	if err := s.RecordDeputy(ctx, org, lee.ID, e.ID, sam.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("a Board Member recorded the Deputy: %v", err)
	}
	if err := s.RecordDeputy(ctx, org, dana.ID, e.ID, lee.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.CloseVoting(ctx, org, lee.ID, e.ID, r.ID, nil); err != nil {
		t.Fatal(err)
	}
	if p, _ := positionOf(ctx, s.db, e.ID, dana.ID); p != "" {
		t.Errorf("after the vote, the Deputy's close and approval, the chair is still %q", p)
	}
}

// The Event Administrator role has no vote; an Event Administrator who holds
// a seat sits as an Event Board Member and votes (owner's rulings, November
// 11, 2026).
func TestEventAdministratorVotesOnlyWithASeat(t *testing.T) {
	for _, seat := range []bool{false, true} {
		s, _ := newTestService(t)
		ctx := context.Background()
		org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
		admin := adminOf(t, s, org)
		lead := confirmed(t, s, org, "lead@example.org", "lead")
		dana := confirmed(t, s, org, "dana@example.org", "dreyes")
		lee := confirmed(t, s, org, "lee@example.org", "lee")
		maria := confirmed(t, s, org, "maria@example.org", "maria")
		s.SetRoles(ctx, org, admin, lead.ID, []string{RoleLeadership})
		pol, err := s.GetPolicies(ctx, org, admin)
		if err != nil {
			t.Fatal(err)
		}
		pol.EventAdminHoldsSeat = seat
		if _, err := s.SetPolicies(ctx, org, admin, *pol); err != nil {
			t.Fatal(err)
		}
		e, _ := s.CreateEvent(ctx, org, admin, NewEvent{Name: "Valentine Dance", BoardCapacity: "5", BoardOrder: "board_first"})
		s.Appoint(ctx, org, lead.ID, e.ID, dana.ID, PositionMember)
		s.Appoint(ctx, org, lead.ID, e.ID, lee.ID, PositionMember)
		if err := s.Appoint(ctx, org, lead.ID, e.ID, maria.ID, PositionAdmin); err != nil {
			t.Fatal(err)
		}
		if err := s.StartRemoval(ctx, org, dana.ID, e.ID, lee.ID, ""); err != nil {
			t.Fatal(err)
		}
		b, _ := s.GetBoard(ctx, org, maria.ID, e.ID)
		if r := b.Removals[0]; r.CanVote != seat {
			t.Errorf("seat %v: the Event Administrator may vote = %v", seat, r.CanVote)
		}
	}
}

// Abstentions: a Board Member who does not vote abstains; the Event Board
// decides whether the vote carries, and the close records it.
func TestAbstentions(t *testing.T) {
	s, _ := newTestService(t)
	ctx := context.Background()
	org, _ := s.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	admin := adminOf(t, s, org)
	lead := confirmed(t, s, org, "lead@example.org", "lead")
	dana := confirmed(t, s, org, "dana@example.org", "dreyes")
	lee := confirmed(t, s, org, "lee@example.org", "lee")
	sam := confirmed(t, s, org, "sam@example.org", "sam")
	s.SetRoles(ctx, org, admin, lead.ID, []string{RoleLeadership})
	e, _ := s.CreateEvent(ctx, org, admin, NewEvent{Name: "Valentine Dance", BoardCapacity: "5", BoardOrder: "board_first"})
	for _, p := range []string{dana.ID, lee.ID, sam.ID} {
		s.Appoint(ctx, org, lead.ID, e.ID, p, PositionMember)
	}
	// Dana starts it (in favor); Lee abstains; Leadership approves.
	if err := s.StartRemoval(ctx, org, dana.ID, e.ID, sam.ID, ""); err != nil {
		t.Fatal(err)
	}
	b, _ := s.GetBoard(ctx, org, lead.ID, e.ID)
	id := b.Removals[0].ID
	s.DecideRemoval(ctx, org, lead.ID, e.ID, id, true)
	if err := s.CloseVoting(ctx, org, lead.ID, e.ID, id, nil); fieldOf(err) != "carries" {
		t.Errorf("closed with an abstention and no decision: %v", err)
	}
	yes := true
	if err := s.CloseVoting(ctx, org, lead.ID, e.ID, id, &yes); err != nil {
		t.Fatal(err)
	}
	if p, _ := positionOf(ctx, s.db, e.ID, sam.ID); p != "" {
		t.Errorf("the board decided the vote carries, but Sam is still %q", p)
	}
}
