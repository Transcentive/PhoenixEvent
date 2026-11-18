package accounts

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Forming the event board (5E, 7B.6, 7B.7): screens S7 (board first), S8
// (chair first) and D1 (Appoint board member).

// Seat positions on an event board.
const (
	PositionChair  = "chair"
	PositionMember = "member"
	PositionAdmin  = "event_administrator"
)

// Seat is one person on the event board.
type Seat struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position string `json:"position"`
	// Deputy: this Board Member is the Deputy Board Chair (7B.8).
	Deputy bool `json:"deputy"`
}

// Candidate is a member of the organization who may be appointed.
type Candidate struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Board is what the Event board page shows, with what the person asking may
// do on it. Every member may read it (CRUD); the actions follow 7B.6.
type Board struct {
	Event          Event       `json:"event"`
	Seats          []Seat      `json:"seats"`
	SeatsTaken     int         `json:"seats_taken"`
	AdminHoldsSeat bool        `json:"event_admin_holds_seat"`
	Formed         bool        `json:"formed"`
	Candidates     []Candidate `json:"candidates"`
	// What the person asking may do.
	CanAppointChair   bool `json:"can_appoint_chair"`
	CanRecordChair    bool `json:"can_record_chair"`
	CanAppointMembers bool `json:"can_appoint_members"`
	CanAppointAdmin   bool `json:"can_appoint_admin"`
	// CanRemove: the Event Chair removes the Event Administrator alone
	// (5D.3, November 11, 2026).
	CanRemove bool `json:"can_remove"`
	// CanRecordDeputy: the Event Chair, or Organization Leadership before an
	// Event Chair exists, records the board's election (7B.8).
	CanRecordDeputy bool `json:"can_record_deputy"`
	// CanReplaceChair: Organization Leadership replaces the Event Board Chair
	// (D2, 7B.6).
	CanReplaceChair bool `json:"can_replace_chair"`
	// People whose removal the person asking may start (5D.3).
	CanStartRemoval []string  `json:"can_start_removal"`
	Removals        []Removal `json:"removals"`
}

var (
	errBoardFull = FieldErrors{{"person", "Every seat on the board is taken. Raise the board's capacity on Edit event first."}}
	errOnBoard   = FieldErrors{{"person", "This person already holds a seat on the board."}}
)

// positionOf returns the actor's position on the event's board, or "".
func positionOf(ctx context.Context, db execQuerier, eventID, accountID string) (string, error) {
	var p string
	err := db.QueryRow(ctx, `SELECT position FROM event_board WHERE event_id = $1 AND account_id = $2`, eventID, accountID).Scan(&p)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return p, err
}

// GetBoard opens the Event board page.
func (s *Service) GetBoard(ctx context.Context, org *Organization, actorID, eventID string) (*Board, error) {
	e, err := s.event(ctx, s.db, org, eventID)
	if err != nil {
		return nil, err
	}
	b := &Board{Event: *e}
	if b.Event.CanEdit, err = s.canEditEvent(ctx, org, actorID, eventID); err != nil {
		return nil, err
	}
	pol, err := s.policies(ctx, s.db, org)
	if err != nil {
		return nil, err
	}
	b.AdminHoldsSeat = pol.EventAdminHoldsSeat
	rows, err := s.db.Query(ctx, `SELECT account_id, position, deputy FROM event_board WHERE event_id = $1 ORDER BY appointed_at`, eventID)
	if err != nil {
		return nil, err
	}
	type entry struct {
		id, position string
		deputy       bool
	}
	entries, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (entry, error) {
		var x entry
		return x, r.Scan(&x.id, &x.position, &x.deputy)
	})
	if err != nil {
		return nil, err
	}
	b.Seats = []Seat{}
	hasChair, members := false, 0
	for _, x := range entries {
		p, err := s.GetProfile(ctx, org, x.id)
		if err != nil {
			return nil, err
		}
		b.Seats = append(b.Seats, Seat{ID: x.id, Name: p.Name, Position: x.position, Deputy: x.deputy})
		switch x.position {
		case PositionChair:
			hasChair = true
		case PositionMember:
			members++
		}
	}
	b.SeatsTaken = seatsTaken(entries2positions(b.Seats), pol.EventAdminHoldsSeat)
	b.Formed = hasChair && members > 0

	leadership, err := s.holds(ctx, org, actorID, RoleLeadership)
	if err != nil {
		return nil, err
	}
	mine, err := positionOf(ctx, s.db, eventID, actorID)
	if err != nil {
		return nil, err
	}
	boardFirst := e.BoardOrder == "board_first"
	b.CanAppointChair = leadership && !hasChair && !boardFirst
	b.CanRecordChair = leadership && !hasChair && boardFirst && members > 0
	b.CanAppointMembers = (leadership && boardFirst && !hasChair) || (hasChair && mine == PositionChair)
	b.CanAppointAdmin = leadership
	b.CanRemove = hasChair && mine == PositionChair
	b.CanRecordDeputy = members > 0 && ((hasChair && mine == PositionChair) || (!hasChair && leadership))
	b.CanReplaceChair = leadership && hasChair
	st, err := s.boardState(ctx, s.db, org, actorID, eventID, false)
	if err != nil {
		return nil, err
	}
	b.CanStartRemoval = []string{}
	for _, x := range b.Seats {
		if st.canStart(actorID, x.ID) {
			b.CanStartRemoval = append(b.CanStartRemoval, x.ID)
		}
	}
	if b.Removals, err = s.openRemovals(ctx, org, actorID, eventID, st); err != nil {
		return nil, err
	}
	if b.CanAppointChair || b.CanAppointMembers || b.CanAppointAdmin || b.CanReplaceChair {
		if b.Candidates, err = s.candidates(ctx, org, b.Seats); err != nil {
			return nil, err
		}
	}
	if b.Candidates == nil {
		b.Candidates = []Candidate{}
	}
	return b, nil
}

func entries2positions(seats []Seat) []string {
	out := make([]string, len(seats))
	for i, x := range seats {
		out[i] = x.Position
	}
	return out
}

// seatsTaken counts the chair and the board members, and the Event
// Administrator only when the organization says that position holds a seat
// (7B.7).
func seatsTaken(positions []string, adminHoldsSeat bool) int {
	n := 0
	for _, p := range positions {
		if p != PositionAdmin || adminHoldsSeat {
			n++
		}
	}
	return n
}

// candidates are the organization's members who hold no seat on the board.
func (s *Service) candidates(ctx context.Context, org *Organization, seats []Seat) ([]Candidate, error) {
	rows, err := s.db.Query(ctx, `SELECT id FROM accounts WHERE organization_id = $1 AND email_confirmed_at IS NOT NULL`, org.ID)
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	out := []Candidate{}
	for _, id := range ids {
		if slices.ContainsFunc(seats, func(x Seat) bool { return x.ID == id }) {
			continue
		}
		p, err := s.GetProfile(ctx, org, id)
		if err != nil {
			return nil, err
		}
		out = append(out, Candidate{ID: id, Name: p.Name})
	}
	slices.SortFunc(out, func(a, b Candidate) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	return out, nil
}

// Appoint is D1 (board member or Event Administrator) and S8's Appoint chair.
// It checks the permission against the board as it stands, inside one
// transaction with the event locked, and writes an audit record.
func (s *Service) Appoint(ctx context.Context, org *Organization, actorID, eventID, personID, position string) error {
	if !slices.Contains([]string{PositionChair, PositionMember, PositionAdmin}, position) {
		return FieldErrors{{"position", "Choose the role on the board."}}
	}
	if _, err := s.GetProfile(ctx, org, personID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return FieldErrors{{"person", "Choose a member."}}
		}
		return err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var order string
	var capacity int
	if err := tx.QueryRow(ctx, `SELECT board_order, board_capacity FROM events WHERE organization_id = $1 AND id::text = $2 FOR UPDATE`,
		org.ID, eventID).Scan(&order, &capacity); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	rows, err := tx.Query(ctx, `SELECT account_id, position FROM event_board WHERE event_id = $1`, eventID)
	if err != nil {
		return err
	}
	type entry struct{ id, position string }
	entries, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (entry, error) {
		var x entry
		return x, r.Scan(&x.id, &x.position)
	})
	if err != nil {
		return err
	}
	hasChair, mine, positions, oldAdmin := false, "", []string{}, ""
	for _, x := range entries {
		if x.id == personID {
			return errOnBoard
		}
		if x.position == PositionChair {
			hasChair = true
		}
		if x.position == PositionAdmin {
			oldAdmin = x.id
		}
		if x.id == actorID {
			mine = x.position
		}
		positions = append(positions, x.position)
	}
	leadership, err := s.holds(ctx, org, actorID, RoleLeadership)
	if err != nil {
		return err
	}
	allowed := false
	switch position {
	case PositionChair:
		allowed = leadership && !hasChair && order == "chair_first"
	case PositionMember:
		allowed = (leadership && order == "board_first" && !hasChair) || (hasChair && mine == PositionChair)
	case PositionAdmin:
		allowed = leadership
	}
	if !allowed {
		return ErrForbidden
	}
	pol, err := s.policies(ctx, tx, org)
	if err != nil {
		return err
	}
	if position != PositionAdmin || pol.EventAdminHoldsSeat {
		taken := seatsTaken(positions, pol.EventAdminHoldsSeat)
		if oldAdmin != "" && position == PositionAdmin {
			taken-- // the new Event Administrator takes the old one's seat
		}
		if taken >= capacity {
			return errBoardFull
		}
	}
	if position == PositionAdmin && oldAdmin != "" {
		// Organization Leadership appoints and replaces the Event
		// Administrator (7B.7).
		if _, err := tx.Exec(ctx, `DELETE FROM event_board WHERE event_id = $1 AND account_id = $2`, eventID, oldAdmin); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO event_board (event_id, organization_id, account_id, position, appointed_by)
		VALUES ($1, $2, $3, $4, $5)`, eventID, org.ID, personID, position, actorID); err != nil {
		return err
	}
	if err := audit(ctx, tx, org.ID, actorID, "board.appointed", map[string]any{
		"event": eventID, "person": personID, "position": position, "replaced": oldAdmin,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RecordChair is S7's Record the board's choice: Organization Leadership
// records the chair the board members agreed on, from among them (5E.4).
func (s *Service) RecordChair(ctx context.Context, org *Organization, actorID, eventID, personID string) error {
	if ok, err := s.holds(ctx, org, actorID, RoleLeadership); err != nil || !ok {
		return orForbidden(err)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var order string
	if err := tx.QueryRow(ctx, `SELECT board_order FROM events WHERE organization_id = $1 AND id::text = $2 FOR UPDATE`,
		org.ID, eventID).Scan(&order); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if order != "board_first" {
		return ErrForbidden
	}
	var chairs int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM event_board WHERE event_id = $1 AND position = 'chair'`, eventID).Scan(&chairs); err != nil {
		return err
	}
	if chairs > 0 {
		return ErrForbidden
	}
	tag, err := tx.Exec(ctx, `UPDATE event_board SET position = 'chair', deputy = false WHERE event_id = $1 AND account_id::text = $2 AND position = 'member'`,
		eventID, personID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return FieldErrors{{"person", "Choose a board member."}}
	}
	if err := audit(ctx, tx, org.ID, actorID, "board.chair_recorded", map[string]any{"event": eventID, "person": personID}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RemoveFromBoard is the Event Chair removing a position other than a Board
// Member or the Event Chair, which the Event Chair alone may do (5D.3,
// November 11, 2026): today, the Event Administrator. Board Members and the
// Event Chair leave through a removal (removals.go).
func (s *Service) RemoveFromBoard(ctx context.Context, org *Organization, actorID, eventID, personID string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	st, err := s.boardState(ctx, tx, org, actorID, eventID, true)
	if err != nil {
		return err
	}
	if st.chair == "" || actorID != st.chair {
		return ErrForbidden
	}
	if st.seats[personID] != PositionAdmin {
		return ErrForbidden
	}
	if _, err := tx.Exec(ctx, `DELETE FROM event_board WHERE event_id::text = $1 AND account_id::text = $2`, eventID, personID); err != nil {
		return err
	}
	if err := audit(ctx, tx, org.ID, actorID, "board.removed", map[string]any{"event": eventID, "person": personID, "position": PositionAdmin}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RecordDeputy records the Event Board's election of the Deputy Board Chair
// from among the Board Members (7B.8): the Event Chair records it, or
// Organization Leadership before an Event Chair exists. Recording another
// Board Member replaces the one before; an empty person clears it.
func (s *Service) RecordDeputy(ctx context.Context, org *Organization, actorID, eventID, personID string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	st, err := s.boardState(ctx, tx, org, actorID, eventID, true)
	if err != nil {
		return err
	}
	if !(st.chair != "" && actorID == st.chair || st.chair == "" && st.leadership) {
		return ErrForbidden
	}
	if personID != "" && st.seats[personID] != PositionMember {
		return FieldErrors{{"person", "Choose a Board Member."}}
	}
	if _, err := tx.Exec(ctx, `UPDATE event_board SET deputy = false WHERE event_id::text = $1 AND deputy`, eventID); err != nil {
		return err
	}
	if personID != "" {
		if _, err := tx.Exec(ctx, `UPDATE event_board SET deputy = true WHERE event_id::text = $1 AND account_id::text = $2`, eventID, personID); err != nil {
			return err
		}
	}
	if err := audit(ctx, tx, org.ID, actorID, "board.deputy_recorded", map[string]any{"event": eventID, "person": personID}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Why an Event Board Chair is replaced (5D.3). Termination goes through a
// removal (removals.go), which empties the chair's seat.
const (
	ReplaceResigned = "resigned"
	ReplaceIllness  = "illness"
	ReplaceDeath    = "death"
)

// ReplaceChairRequest is D2, Replace chair, for the Event Board Chair. The
// position is offered outside PhoenixEvent; Organization Leadership records
// each Board Member's answer here (owner, November 18, 2026).
type ReplaceChairRequest struct {
	Why string `json:"why"`
	// Answers from the Board Members, by account: true accepted, false
	// declined. Every Board Member must have answered.
	Answers map[string]bool `json:"answers"`
	// Person is the new Event Chair.
	Person string `json:"person"`
}

var (
	errChairBoardFull = FieldErrors{{"person", "Every seat on the board is taken. The Event Board decides what happens when the board is full; by default the board's size does not change."}}
	errChairRemoval   = FieldErrors{{"person", "A removal of the Event Chair is open. Finish or withdraw it before replacing the chair."}}
)

// boardMembers are the seat holders who sit as Board Members: the Board
// Members, and the Event Administrator when the organization gives that
// position a seat (7B.7).
func boardMembers(seats map[string]string, adminHoldsSeat bool) []string {
	out := []string{}
	for id, p := range seats {
		if p == PositionMember || p == PositionAdmin && adminHoldsSeat {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return out
}

// ReplaceChair is D2 for the Event Board Chair (5D.3, 7B.6): Organization
// Leadership replaces the chair after the chair resigns, an illness leaves the
// chair unable to resign, or the chair dies. The Board Members have first
// right of refusal, the event's participants second, the organization at
// large third (owner, November 18, 2026). Event participants come with
// registration, so today the position goes to the organization at large once
// every Board Member has declined. When several accept, Organization
// Leadership chooses. A chair who resigns stays on the board as a Board
// Member; one replaced because of illness or death leaves it.
func (s *Service) ReplaceChair(ctx context.Context, org *Organization, actorID, eventID string, req ReplaceChairRequest) error {
	if !slices.Contains([]string{ReplaceResigned, ReplaceIllness, ReplaceDeath}, req.Why) {
		return FieldErrors{{"why", "Choose why the chair is replaced."}}
	}
	if ok, err := s.holds(ctx, org, actorID, RoleLeadership); err != nil || !ok {
		return orForbidden(err)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var capacity int
	if err := tx.QueryRow(ctx, `SELECT board_capacity FROM events WHERE organization_id = $1 AND id::text = $2 FOR UPDATE`,
		org.ID, eventID).Scan(&capacity); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	rows, err := tx.Query(ctx, `SELECT account_id::text, position FROM event_board WHERE event_id::text = $1`, eventID)
	if err != nil {
		return err
	}
	seats := map[string]string{}
	chair := ""
	type entry struct{ id, position string }
	entries, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (entry, error) {
		var x entry
		return x, r.Scan(&x.id, &x.position)
	})
	if err != nil {
		return err
	}
	positions := []string{}
	for _, x := range entries {
		seats[x.id] = x.position
		positions = append(positions, x.position)
		if x.position == PositionChair {
			chair = x.id
		}
	}
	if chair == "" {
		return ErrForbidden
	}
	var open int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM board_removals WHERE event_id::text = $1 AND account_id::text = $2 AND status = 'open'`,
		eventID, chair).Scan(&open); err != nil {
		return err
	}
	if open > 0 {
		return errChairRemoval
	}
	pol, err := s.policies(ctx, tx, org)
	if err != nil {
		return err
	}
	members := boardMembers(seats, pol.EventAdminHoldsSeat)
	accepted := []string{}
	for _, id := range members {
		a, ok := req.Answers[id]
		if !ok {
			return FieldErrors{{"answers", "Record every Board Member's answer."}}
		}
		if a {
			accepted = append(accepted, id)
		}
	}
	if req.Person == "" {
		return FieldErrors{{"person", "Choose the new chair."}}
	}
	fromBoard := slices.Contains(members, req.Person)
	switch {
	case len(accepted) > 0 && !slices.Contains(accepted, req.Person):
		return FieldErrors{{"person", "Choose one of the Board Members who accepted."}}
	case len(accepted) == 0 && seats[req.Person] != "":
		return FieldErrors{{"person", "Choose a member of the organization who holds no seat on the board."}}
	}
	if !fromBoard {
		if _, err := s.GetProfile(ctx, org, req.Person); err != nil {
			if errors.Is(err, ErrNotFound) {
				return FieldErrors{{"person", "Choose the new chair."}}
			}
			return err
		}
		taken := seatsTaken(positions, pol.EventAdminHoldsSeat)
		if req.Why != ReplaceResigned {
			taken-- // the chair leaves the board
		}
		if taken >= capacity {
			return errChairBoardFull
		}
	}
	if req.Why == ReplaceResigned {
		if _, err := tx.Exec(ctx, `UPDATE event_board SET position = 'member', deputy = false WHERE event_id::text = $1 AND account_id::text = $2`, eventID, chair); err != nil {
			return err
		}
	} else if _, err := tx.Exec(ctx, `DELETE FROM event_board WHERE event_id::text = $1 AND account_id::text = $2`, eventID, chair); err != nil {
		return err
	}
	if fromBoard {
		if _, err := tx.Exec(ctx, `UPDATE event_board SET position = 'chair', deputy = false WHERE event_id::text = $1 AND account_id::text = $2`, eventID, req.Person); err != nil {
			return err
		}
	} else if _, err := tx.Exec(ctx, `
		INSERT INTO event_board (event_id, organization_id, account_id, position, appointed_by)
		VALUES ($1, $2, $3, 'chair', $4)`, eventID, org.ID, req.Person, actorID); err != nil {
		return err
	}
	if err := audit(ctx, tx, org.ID, actorID, "board.chair_replaced", map[string]any{
		"event": eventID, "previous": chair, "person": req.Person, "why": req.Why, "answers": req.Answers,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
