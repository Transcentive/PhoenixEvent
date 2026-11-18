package accounts

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Event is one event as O1, Organization home, lists it.
type Event struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Date          *string `json:"date"`
	Location      *string `json:"location"`
	Description   *string `json:"description"`
	BoardCapacity int     `json:"board_capacity"`
	Visibility    string  `json:"visibility"`
	BoardOrder    string  `json:"board_order"`
	// Whether the person asking may edit the event (16B.6).
	CanEdit bool `json:"can_edit"`
	// Whether the board has its chair and board members (5E.4).
	Formed bool `json:"formed"`
}

// NewEvent is what S1, Create event, sends. The board's capacity arrives as
// typed, so that a wrong entry is answered under its field.
type NewEvent struct {
	Name          string `json:"name"`
	Date          string `json:"date"`
	Location      string `json:"location"`
	Description   string `json:"description"`
	BoardCapacity string `json:"board_capacity"`
	BoardOrder    string `json:"board_order"`
}

var boardOrders = []string{"board_first", "chair_first"}

// Events lists the organization's events. Every event of the first release
// is private, and every member sees it once logged in (16B.6).
func (s *Service) Events(ctx context.Context, org *Organization, actorID string) ([]Event, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, to_char(event_date, 'YYYY-MM-DD'), location, description, board_capacity, visibility, board_order
		FROM events WHERE organization_id = $1
		ORDER BY event_date NULLS LAST, lower(name)`, org.ID)
	if err != nil {
		return nil, err
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Event, error) {
		var e Event
		err := r.Scan(&e.ID, &e.Name, &e.Date, &e.Location, &e.Description, &e.BoardCapacity, &e.Visibility, &e.BoardOrder)
		return e, err
	})
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []Event{}
	}
	for i := range list {
		if list[i].CanEdit, err = s.canEditEvent(ctx, org, actorID, list[i].ID); err != nil {
			return nil, err
		}
		if err := s.db.QueryRow(ctx, `
			SELECT bool_or(position = 'chair') IS TRUE AND bool_or(position = 'member') IS TRUE
			FROM event_board WHERE event_id = $1`, list[i].ID).Scan(&list[i].Formed); err != nil {
			return nil, err
		}
	}
	return list, nil
}

// canEditEvent: the Organization Administrator, the event's Event
// Administrator and its Event Chair may edit an event by default (owner's
// ruling of November 9, 2026). The Event Chair need not wait for the board to
// be formed.
func (s *Service) canEditEvent(ctx context.Context, org *Organization, actorID, eventID string) (bool, error) {
	if ok, err := s.holds(ctx, org, actorID, RoleAdministrator); err != nil || ok {
		return ok, err
	}
	p, err := positionOf(ctx, s.db, eventID, actorID)
	return p == PositionChair || p == PositionAdmin, err
}

// GetEvent opens an event for editing. A person who may not edit it, or an
// event of another organization, is answered alike (E3).
func (s *Service) GetEvent(ctx context.Context, org *Organization, actorID, eventID string) (*Event, error) {
	e, err := s.event(ctx, s.db, org, eventID)
	if err != nil {
		return nil, err
	}
	if e.CanEdit, err = s.canEditEvent(ctx, org, actorID, eventID); err != nil || !e.CanEdit {
		return nil, orForbidden(err)
	}
	return e, nil
}

func (s *Service) event(ctx context.Context, db execQuerier, org *Organization, eventID string) (*Event, error) {
	var e Event
	err := db.QueryRow(ctx, `
		SELECT id, name, to_char(event_date, 'YYYY-MM-DD'), location, description, board_capacity, visibility, board_order
		FROM events WHERE organization_id = $1 AND id::text = $2`, org.ID, eventID).
		Scan(&e.ID, &e.Name, &e.Date, &e.Location, &e.Description, &e.BoardCapacity, &e.Visibility, &e.BoardOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &e, err
}

// UpdateEvent is Edit event's Save event: the fields of S1, checked as S1
// checks them, in one transaction with an audit record. The order, board
// first or chair first, is not changed here: whether it may change after the
// event is created is open (5E.4).
func (s *Service) UpdateEvent(ctx context.Context, org *Organization, actorID, eventID string, in NewEvent) (*Event, error) {
	before, err := s.GetEvent(ctx, org, actorID, eventID)
	if err != nil {
		return nil, err
	}
	in.BoardOrder = before.BoardOrder
	e, err := checkEvent(in)
	if err != nil {
		return nil, err
	}
	e.ID, e.CanEdit = before.ID, true
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT 1 FROM events WHERE id = $1 FOR UPDATE`, e.ID); err != nil {
		return nil, err
	}
	pol, err := s.policies(ctx, tx, org)
	if err != nil {
		return nil, err
	}
	var held int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM event_board WHERE event_id = $1 AND (position <> 'event_administrator' OR $2)`,
		e.ID, pol.EventAdminHoldsSeat).Scan(&held); err != nil {
		return nil, err
	}
	if e.BoardCapacity < held {
		return nil, FieldErrors{{"board_capacity", strconv.Itoa(held) + " seats are taken. The capacity cannot be lower."}}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE events SET name = $3, event_date = $4, location = $5, description = $6, board_capacity = $7
		WHERE organization_id = $1 AND id = $2`,
		org.ID, e.ID, e.Name, e.Date, e.Location, e.Description, e.BoardCapacity); err != nil {
		return nil, err
	}
	if err := audit(ctx, tx, org.ID, actorID, "event.changed", map[string]any{"event": e.ID, "from": before, "to": e}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return e, nil
}

// CreateEvent is S1's Create event. In the first release an Organization
// Administrator creates events directly (16B.2, feature 10; 7B.6). The
// event name and a board of at least 1 seat are required; the date,
// location and description are optional (16B.6).
func (s *Service) CreateEvent(ctx context.Context, org *Organization, actorID string, in NewEvent) (*Event, error) {
	if ok, err := s.holds(ctx, org, actorID, RoleAdministrator); err != nil || !ok {
		return nil, orForbidden(err)
	}
	e, err := checkEvent(in)
	if err != nil {
		return nil, err
	}
	e.CanEdit = true
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err := tx.QueryRow(ctx, `
		INSERT INTO events (organization_id, name, event_date, location, description, board_capacity, board_order, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		org.ID, e.Name, e.Date, e.Location, e.Description, e.BoardCapacity, e.BoardOrder, actorID).Scan(&e.ID); err != nil {
		return nil, err
	}
	if err := audit(ctx, tx, org.ID, actorID, "event.created", map[string]any{
		"event": e.ID, "name": e.Name, "board_capacity": e.BoardCapacity, "board_order": e.BoardOrder,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return e, nil
}

// checkEvent checks S1's fields (16B.6, 5A.4 step 1).
func checkEvent(in NewEvent) (*Event, error) {
	var problems FieldErrors
	e := Event{Name: strings.TrimSpace(in.Name), Visibility: "private", BoardOrder: in.BoardOrder,
		Location: optional(in.Location), Description: optional(in.Description)}
	if e.Name == "" {
		problems = append(problems, FieldError{"name", "Enter the event's name."})
	}
	if d := strings.TrimSpace(in.Date); d != "" {
		if _, err := time.Parse("2006-01-02", d); err != nil {
			problems = append(problems, FieldError{"date", "Enter the date as month, day and year."})
		} else {
			e.Date = &d
		}
	}
	n, err := strconv.Atoi(strings.TrimSpace(in.BoardCapacity))
	if err != nil || n < 1 {
		problems = append(problems, FieldError{"board_capacity", "Enter a whole number of seats, at least 1."})
	}
	e.BoardCapacity = n
	if !slices.Contains(boardOrders, in.BoardOrder) {
		problems = append(problems, FieldError{"board_order", "Choose which comes first, the board or the chair."})
	}
	if problems != nil {
		return nil, problems
	}
	return &e, nil
}

// DeleteEvent deletes an event and its board (CRUD). It is held by those who
// may edit the event.
func (s *Service) DeleteEvent(ctx context.Context, org *Organization, actorID, eventID string) error {
	e, err := s.GetEvent(ctx, org, actorID, eventID)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM events WHERE organization_id = $1 AND id = $2`, org.ID, e.ID); err != nil {
		return err
	}
	if err := audit(ctx, tx, org.ID, actorID, "event.deleted", map[string]any{"event": e}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
