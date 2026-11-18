package accounts

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Removing a person from an event board (5D.3, rulings of November 11, 2026).
// A Board Member or the Event Chair leaves the board only upon resignation,
// termination, incapacitating illness or death. Termination with
// reason, and removal before an Event Chair exists, go through a removal:
//
//   - Before an Event Chair exists: a Board Member starts the removal of
//     another Board Member; every other seat on the board votes, and the vote
//     must be unanimous (the member is recused); an Organization Leadership
//     agent approves.
//   - A Board Member, once the Event Chair exists: Organization Leadership and
//     the Event Chair must both agree; either may start it.
//   - The Event Chair: a Board Member or Organization Leadership starts it;
//     the board votes unanimously (the Event Chair is recused); an
//     Organization Leadership agent approves.
//
// Every other position associated with the event (today, the Event
// Administrator) is removed by the Event Chair alone: RemoveFromBoard.

// Removal is one removal on the Event board page, with what the person
// asking may do on it.
type Removal struct {
	ID          string `json:"id"`
	Person      string `json:"person"`
	Name        string `json:"name"`
	Position    string `json:"position"`
	Reason      string `json:"reason"`
	StartedBy   string `json:"started_by"`
	NeedsVote   bool   `json:"needs_vote"`
	NeedsChair  bool   `json:"needs_chair"`
	ChairAgreed bool   `json:"chair_agreed"`
	Approved    bool   `json:"leadership_approved"`
	Votes       []Vote `json:"votes"`
	// MyVote is the asking person's vote, if any; VotingClosed once the
	// Event Board has closed the voting period.
	MyVote       *bool `json:"my_vote"`
	VotingClosed bool  `json:"voting_closed"`
	// Voters who have not voted yet.
	Waiting []string `json:"waiting"`

	CanVote    bool `json:"can_vote"`
	CanClose   bool `json:"can_close_voting"`
	CanApprove bool `json:"can_approve"` // as an Organization Leadership agent
	CanAgree   bool `json:"can_agree"`   // as the Event Chair
	CanEdit    bool `json:"can_edit"`    // the reason, and withdrawing
}

// Vote is one board member's vote on a removal.
type Vote struct {
	Name    string `json:"name"`
	InFavor bool   `json:"in_favor"`
}

var (
	errOpenAlready = FieldErrors{{"person", "A removal of this person is already open."}}
)

type boardState struct {
	order      string
	seats      map[string]string // account id -> position
	chair      string
	deputy     string
	leadership bool
	adminSeat  bool
}

func (s *Service) boardState(ctx context.Context, db execQuerier, org *Organization, actorID, eventID string, lock bool) (*boardState, error) {
	q := `SELECT board_order FROM events WHERE organization_id = $1 AND id::text = $2`
	if lock {
		q += ` FOR UPDATE`
	}
	st := &boardState{seats: map[string]string{}}
	if err := db.QueryRow(ctx, q, org.ID, eventID).Scan(&st.order); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	rows, err := db.Query(ctx, `SELECT account_id, position, deputy FROM event_board WHERE event_id::text = $1`, eventID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id, p string
		var deputy bool
		if err := rows.Scan(&id, &p, &deputy); err != nil {
			return nil, err
		}
		st.seats[id] = p
		if deputy {
			st.deputy = id
		}
		if p == PositionChair {
			st.chair = id
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	pol, err := s.policies(ctx, db, org)
	if err != nil {
		return nil, err
	}
	st.adminSeat = pol.EventAdminHoldsSeat
	if st.leadership, err = s.holds(ctx, org, actorID, RoleLeadership); err != nil {
		return nil, err
	}
	return st, nil
}

// voters are the Event Chair and the Event Board Members, except the person
// whose removal it is. The Event Administrator role has no voting power, but
// an Event Administrator who holds a seat on the board (7B.7) sits as an
// Event Board Member and votes as one (owner's rulings, November 11, 2026).
func (st *boardState) voters(person string) []string {
	out := []string{}
	for id := range st.seats {
		if id != person && st.onBoard(id) {
			out = append(out, id)
		}
	}
	return out
}

// onBoard: the person sits on the board with a vote.
func (st *boardState) onBoard(id string) bool {
	switch st.seats[id] {
	case PositionChair, PositionMember:
		return true
	case PositionAdmin:
		return st.adminSeat
	}
	return false
}

// canStart says whether the actor may start the removal of person.
func (st *boardState) canStart(actorID, person string) bool {
	if actorID == person {
		return false
	}
	switch st.seats[person] {
	case PositionMember:
		if st.chair == "" {
			return st.onBoard(actorID)
		}
		return actorID == st.chair || st.leadership
	case PositionChair:
		return st.onBoard(actorID) || st.leadership
	}
	return false
}

// StartRemoval creates a removal. The person starting it counts as agreeing
// on their side: a Board Member's vote in favor, the Event Chair's
// agreement, or Organization Leadership's approval.
func (s *Service) StartRemoval(ctx context.Context, org *Organization, actorID, eventID, personID, reason string) error {
	reason = strings.TrimSpace(reason)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	st, err := s.boardState(ctx, tx, org, actorID, eventID, true)
	if err != nil {
		return err
	}
	position := st.seats[personID]
	if position != PositionMember && position != PositionChair {
		return FieldErrors{{"person", "Choose a Board Member or the Event Chair."}}
	}
	if !st.canStart(actorID, personID) {
		return ErrForbidden
	}
	needsVote := position == PositionChair || st.chair == ""
	needsChair := !needsVote
	var chairAgreed, approved any
	if needsChair && actorID == st.chair {
		chairAgreed = actorID
	}
	if st.leadership && !st.onBoard(actorID) || st.leadership && needsChair && actorID != st.chair {
		approved = actorID
	}
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO board_removals (event_id, organization_id, account_id, position, reason, needs_vote, needs_chair,
		                            chair_agreed_by, leadership_approved_by, started_by)
		SELECT id, organization_id, $3, $4, $5, $6, $7, $8, $9, $10 FROM events WHERE organization_id = $1 AND id::text = $2
		RETURNING id`,
		org.ID, eventID, personID, position, reason, needsVote, needsChair, chairAgreed, approved, actorID).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return errOpenAlready
		}
		return err
	}
	if needsVote && st.onBoard(actorID) {
		if _, err := tx.Exec(ctx, `INSERT INTO board_removal_votes (removal_id, account_id, in_favor) VALUES ($1, $2, true)`, id, actorID); err != nil {
			return err
		}
	}
	if err := audit(ctx, tx, org.ID, actorID, "board.removal_started", map[string]any{
		"event": eventID, "removal": id, "person": personID, "position": position, "reason": reason,
	}); err != nil {
		return err
	}
	if err := s.settle(ctx, tx, org, actorID, eventID, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type removalRow struct {
	id, eventID, person, position, reason, startedBy, status string
	needsVote, needsChair                                    bool
	chairAgreed, approved                                    *string
	votingClosed                                             bool
}

func (s *Service) removal(ctx context.Context, db execQuerier, org *Organization, eventID, removalID string, lock bool) (*removalRow, error) {
	q := `SELECT id, event_id, account_id, position, reason, started_by, status, needs_vote, needs_chair, chair_agreed_by, leadership_approved_by, voting_closed_at IS NOT NULL
		FROM board_removals WHERE organization_id = $1 AND event_id::text = $2 AND id::text = $3`
	if lock {
		q += ` FOR UPDATE`
	}
	var r removalRow
	err := db.QueryRow(ctx, q, org.ID, eventID, removalID).Scan(&r.id, &r.eventID, &r.person, &r.position, &r.reason,
		&r.startedBy, &r.status, &r.needsVote, &r.needsChair, &r.chairAgreed, &r.approved, &r.votingClosed)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &r, err
}

func (s *Service) votesOn(ctx context.Context, db execQuerier, removalID string) (map[string]bool, error) {
	rows, err := db.Query(ctx, `SELECT account_id, in_favor FROM board_removal_votes WHERE removal_id = $1 ORDER BY voted_at`, removalID)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for rows.Next() {
		var id string
		var v bool
		if err := rows.Scan(&id, &v); err != nil {
			return nil, err
		}
		out[id] = v
	}
	return out, rows.Err()
}

// settle closes an open removal once it is decided: declined on any vote
// against; otherwise, once every voter is in favor (when a vote is needed),
// the Event Chair agrees (when needed) and Organization Leadership approves,
// the person leaves the board.
func (s *Service) settle(ctx context.Context, tx pgx.Tx, org *Organization, actorID, eventID, removalID string) error {
	r, err := s.removal(ctx, tx, org, eventID, removalID, true)
	if err != nil || r.status != "open" {
		return err
	}
	st, err := s.boardState(ctx, tx, org, actorID, eventID, false)
	if err != nil {
		return err
	}
	if st.seats[r.person] != r.position {
		// The person no longer holds that seat (resigned or replaced).
		return s.close(ctx, tx, org, actorID, r, "declined")
	}
	if r.needsVote && !r.votingClosed {
		return nil // the Event Board has not closed the voting period
	}
	if r.needsChair && r.chairAgreed == nil || r.approved == nil {
		return nil
	}
	if _, err := tx.Exec(ctx, `DELETE FROM event_board WHERE event_id = $1 AND account_id = $2`, r.eventID, r.person); err != nil {
		return err
	}
	return s.close(ctx, tx, org, actorID, r, "removed")
}

func (s *Service) close(ctx context.Context, tx pgx.Tx, org *Organization, actorID string, r *removalRow, status string) error {
	if _, err := tx.Exec(ctx, `UPDATE board_removals SET status = $2, closed_at = now() WHERE id = $1`, r.id, status); err != nil {
		return err
	}
	return audit(ctx, tx, org.ID, actorID, "board.removal_"+status, map[string]any{
		"event": r.eventID, "removal": r.id, "person": r.person, "position": r.position,
	})
}

// VoteOnRemoval records a board member's vote. A vote is final.
func (s *Service) VoteOnRemoval(ctx context.Context, org *Organization, actorID, eventID, removalID string, inFavor bool) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	r, err := s.removal(ctx, tx, org, eventID, removalID, true)
	if err != nil {
		return err
	}
	st, err := s.boardState(ctx, tx, org, actorID, eventID, false)
	if err != nil {
		return err
	}
	if r.status != "open" || !r.needsVote || r.votingClosed || !slices.Contains(st.voters(r.person), actorID) {
		return ErrForbidden
	}
	// A vote may change until the voting period is closed.
	if _, err := tx.Exec(ctx, `
		INSERT INTO board_removal_votes (removal_id, account_id, in_favor) VALUES ($1, $2, $3)
		ON CONFLICT (removal_id, account_id) DO UPDATE SET in_favor = EXCLUDED.in_favor, voted_at = now()`,
		r.id, actorID, inFavor); err != nil {
		return err
	}
	if err := audit(ctx, tx, org.ID, actorID, "board.removal_vote", map[string]any{"event": eventID, "removal": r.id, "in_favor": inFavor}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// canCloseVoting: the Event Chair records that the Event Board closed the
// voting period; before an Event Chair exists, Organization Leadership does
// (owner's agreement, November 11, 2026). When the Event Chair is the one
// being removed, the Deputy Board Chair closes it (7B.8).
func (st *boardState) canCloseVoting(actorID string, r *removalRow) bool {
	if r.status != "open" || !r.needsVote || r.votingClosed {
		return false
	}
	if st.chair == "" {
		return st.leadership
	}
	if r.person == st.chair {
		return st.deputy != "" && actorID == st.deputy
	}
	return actorID == st.chair
}

// CloseVoting records that the Event Board closed the voting period. A vote
// against declines the removal. A Board Member who did not vote is
// abstaining; when there are abstentions, the person recording the close
// also records the Event Board's decision on whether the vote carries
// (owner's rulings, November 11, 2026). Another removal of the same person
// is a new round.
func (s *Service) CloseVoting(ctx context.Context, org *Organization, actorID, eventID, removalID string, carries *bool) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	r, err := s.removal(ctx, tx, org, eventID, removalID, true)
	if err != nil {
		return err
	}
	st, err := s.boardState(ctx, tx, org, actorID, eventID, false)
	if err != nil {
		return err
	}
	if !st.canCloseVoting(actorID, r) {
		return ErrForbidden
	}
	votes, err := s.votesOn(ctx, tx, r.id)
	if err != nil {
		return err
	}
	against, abstaining := false, false
	for _, id := range st.voters(r.person) {
		in, voted := votes[id]
		against = against || voted && !in
		abstaining = abstaining || !voted
	}
	unanimous := !against
	if !against && abstaining {
		if carries == nil {
			return FieldErrors{{"carries", "Record whether the Event Board decided the vote carries with the abstentions."}}
		}
		unanimous = *carries
	}
	if _, err := tx.Exec(ctx, `UPDATE board_removals SET voting_closed_at = now() WHERE id = $1`, r.id); err != nil {
		return err
	}
	if err := audit(ctx, tx, org.ID, actorID, "board.removal_voting_closed", map[string]any{"event": eventID, "removal": r.id, "carries": unanimous, "abstentions": abstaining}); err != nil {
		return err
	}
	if !unanimous {
		if err := s.close(ctx, tx, org, actorID, r, "declined"); err != nil {
			return err
		}
	} else if err := s.settle(ctx, tx, org, actorID, eventID, r.id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// DecideRemoval is Organization Leadership's approval, or the Event Chair's
// agreement; declining closes the removal. A person who holds both decides
// for both.
func (s *Service) DecideRemoval(ctx context.Context, org *Organization, actorID, eventID, removalID string, approve bool) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	r, err := s.removal(ctx, tx, org, eventID, removalID, true)
	if err != nil {
		return err
	}
	st, err := s.boardState(ctx, tx, org, actorID, eventID, false)
	if err != nil {
		return err
	}
	asLeadership := st.leadership && r.approved == nil && actorID != r.person
	asChair := r.needsChair && r.chairAgreed == nil && actorID == st.chair
	if r.status != "open" || !asLeadership && !asChair {
		return ErrForbidden
	}
	if !approve {
		if err := s.close(ctx, tx, org, actorID, r, "declined"); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	if asLeadership {
		if _, err := tx.Exec(ctx, `UPDATE board_removals SET leadership_approved_by = $2 WHERE id = $1`, r.id, actorID); err != nil {
			return err
		}
	}
	if asChair {
		if _, err := tx.Exec(ctx, `UPDATE board_removals SET chair_agreed_by = $2 WHERE id = $1`, r.id, actorID); err != nil {
			return err
		}
	}
	if err := audit(ctx, tx, org.ID, actorID, "board.removal_approved", map[string]any{
		"event": eventID, "removal": r.id, "as_leadership": asLeadership, "as_chair": asChair,
	}); err != nil {
		return err
	}
	if err := s.settle(ctx, tx, org, actorID, eventID, r.id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// EditRemoval changes the reason; only the person who started it, while it
// is open.
func (s *Service) EditRemoval(ctx context.Context, org *Organization, actorID, eventID, removalID, reason string) error {
	r, err := s.removal(ctx, s.db, org, eventID, removalID, false)
	if err != nil {
		return err
	}
	if r.status != "open" || r.startedBy != actorID {
		return ErrForbidden
	}
	reason = strings.TrimSpace(reason)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE board_removals SET reason = $2 WHERE id = $1`, r.id, reason); err != nil {
		return err
	}
	if err := audit(ctx, tx, org.ID, actorID, "board.removal_edited", map[string]any{"event": eventID, "removal": r.id, "reason": reason}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// WithdrawRemoval deletes an open removal; only the person who started it.
func (s *Service) WithdrawRemoval(ctx context.Context, org *Organization, actorID, eventID, removalID string) error {
	r, err := s.removal(ctx, s.db, org, eventID, removalID, false)
	if err != nil {
		return err
	}
	if r.status != "open" || r.startedBy != actorID {
		return ErrForbidden
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM board_removals WHERE id = $1`, r.id); err != nil {
		return err
	}
	if err := audit(ctx, tx, org.ID, actorID, "board.removal_withdrawn", map[string]any{"event": eventID, "removal": r.id, "person": r.person}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// openRemovals lists the event's open removals for the Event board page.
func (s *Service) openRemovals(ctx context.Context, org *Organization, actorID, eventID string, st *boardState) ([]Removal, error) {
	rows, err := s.db.Query(ctx, `SELECT id FROM board_removals WHERE organization_id = $1 AND event_id::text = $2 AND status = 'open' ORDER BY started_at`,
		org.ID, eventID)
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	name := func(id string) (string, error) {
		p, err := s.GetProfile(ctx, org, id)
		if err != nil {
			return "", err
		}
		return p.Name, nil
	}
	out := []Removal{}
	for _, id := range ids {
		r, err := s.removal(ctx, s.db, org, eventID, id, false)
		if err != nil {
			return nil, err
		}
		x := Removal{ID: r.id, Person: r.person, Position: r.position, Reason: r.reason, NeedsVote: r.needsVote,
			NeedsChair: r.needsChair, ChairAgreed: r.chairAgreed != nil, Approved: r.approved != nil, Votes: []Vote{}, Waiting: []string{}}
		if x.Name, err = name(r.person); err != nil {
			return nil, err
		}
		if x.StartedBy, err = name(r.startedBy); err != nil {
			return nil, err
		}
		if r.needsVote {
			votes, err := s.votesOn(ctx, s.db, r.id)
			if err != nil {
				return nil, err
			}
			for _, v := range st.voters(r.person) {
				n, err := name(v)
				if err != nil {
					return nil, err
				}
				if in, ok := votes[v]; ok {
					x.Votes = append(x.Votes, Vote{Name: n, InFavor: in})
					if v == actorID {
						x.MyVote = &in
					}
				} else {
					x.Waiting = append(x.Waiting, n)
				}
				if v == actorID && !r.votingClosed {
					x.CanVote = true
				}
			}
		}
		x.VotingClosed = r.votingClosed
		x.CanClose = st.canCloseVoting(actorID, r)
		x.CanApprove = st.leadership && !x.Approved && actorID != r.person
		x.CanAgree = r.needsChair && !x.ChairAgreed && actorID == st.chair
		x.CanEdit = r.startedBy == actorID
		out = append(out, x)
	}
	return out, nil
}
