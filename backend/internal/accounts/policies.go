package accounts

import (
	"context"
	"errors"
	"slices"

	"github.com/jackc/pgx/v5"
)

// Policies are the settings on O3 that the first release acts on (16B.6).
// Waiting lists and lowering a station's capacity are shown on O3, marked as
// coming in a later release, and are not stored yet.
type Policies struct {
	ApproveRegistrations bool   `json:"approve_registrations"`
	Participation        string `json:"participation"`
	EventAdminHoldsSeat  bool   `json:"event_admin_holds_seat"`
}

var participations = []string{"subscribers", "registrants", "both"}

// startingPolicies are an organization's before an Organization
// Administrator first saves O3: as the wireframe draws them, and "Holds no
// seat" (7B.7).
var startingPolicies = Policies{ApproveRegistrations: true, Participation: "both", EventAdminHoldsSeat: false}

// GetPolicies opens O3. Only an Organization Administrator sets policies, so
// only they open the page (7B.6).
func (s *Service) GetPolicies(ctx context.Context, org *Organization, actorID string) (*Policies, error) {
	if ok, err := s.holds(ctx, org, actorID, RoleAdministrator); err != nil || !ok {
		return nil, orForbidden(err)
	}
	return s.policies(ctx, s.db, org)
}

func (s *Service) policies(ctx context.Context, db execQuerier, org *Organization) (*Policies, error) {
	p := startingPolicies
	err := db.QueryRow(ctx, `
		SELECT approve_registrations, participation, event_admin_holds_seat
		FROM organization_policies WHERE organization_id = $1`, org.ID).
		Scan(&p.ApproveRegistrations, &p.Participation, &p.EventAdminHoldsSeat)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	return &p, nil
}

// SetPolicies is O3's Save policies: one transaction with an audit record.
func (s *Service) SetPolicies(ctx context.Context, org *Organization, actorID string, p Policies) (*Policies, error) {
	if !slices.Contains(participations, p.Participation) {
		return nil, FieldErrors{{"participation", "Choose who participates in an event."}}
	}
	if ok, err := s.holds(ctx, org, actorID, RoleAdministrator); err != nil || !ok {
		return nil, orForbidden(err)
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	before, err := s.policies(ctx, tx, org)
	if err != nil {
		return nil, err
	}
	if *before == p {
		return before, nil
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO organization_policies (organization_id, approve_registrations, participation, event_admin_holds_seat, updated_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (organization_id) DO UPDATE SET
			approve_registrations = EXCLUDED.approve_registrations,
			participation = EXCLUDED.participation,
			event_admin_holds_seat = EXCLUDED.event_admin_holds_seat,
			updated_by = EXCLUDED.updated_by,
			updated_at = now()`,
		org.ID, p.ApproveRegistrations, p.Participation, p.EventAdminHoldsSeat, actorID); err != nil {
		return nil, err
	}
	if err := audit(ctx, tx, org.ID, actorID, "policies.changed", map[string]any{"from": before, "to": p}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &p, nil
}
