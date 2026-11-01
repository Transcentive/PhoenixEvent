package accounts

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

// The two organization roles that are stored (7B.1). Member is every
// account and is not stored.
const (
	RoleAdministrator = "organization_administrator"
	RoleLeadership    = "organization_leadership"
)

var organizationRoles = []string{RoleAdministrator, RoleLeadership}

// ErrForbidden refuses an action the person's roles do not allow (7B.6).
var ErrForbidden = errors.New("This needs a role you do not hold here.")

// Officer is one person on O8, Organization officers (4A.10).
type Officer struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}

// PersonRoles is what D5, Change roles, opens with.
type PersonRoles struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}

// RolesOf returns the organization roles an account holds, as stored.
func (s *Service) RolesOf(ctx context.Context, org *Organization, accountID string) ([]string, error) {
	rows, err := s.db.Query(ctx,
		`SELECT role FROM organization_roles WHERE account_id = $1 AND organization_id = $2 ORDER BY role`, accountID, org.ID)
	if err != nil {
		return nil, err
	}
	roles, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if roles == nil {
		roles = []string{}
	}
	return roles, err
}

func (s *Service) holds(ctx context.Context, org *Organization, accountID, role string) (bool, error) {
	roles, err := s.RolesOf(ctx, org, accountID)
	return slices.Contains(roles, role), err
}

// Officers lists everyone who holds Organization Leadership or Organization
// Administrator, by the name each has chosen to show (8B.2). Every member
// sees the list (4A.10, v1.07).
func (s *Service) Officers(ctx context.Context, org *Organization) ([]Officer, error) {
	rows, err := s.db.Query(ctx,
		`SELECT DISTINCT account_id FROM organization_roles WHERE organization_id = $1`, org.ID)
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	officers := []Officer{}
	for _, id := range ids {
		p, err := s.GetProfile(ctx, org, id)
		if err != nil {
			return nil, err
		}
		officers = append(officers, Officer{ID: p.ID, Name: p.Name, Roles: p.Roles})
	}
	slices.SortFunc(officers, func(a, b Officer) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	return officers, nil
}

// RolesFor is D5's starting point. Only an Organization Administrator may
// open it (7B.6).
func (s *Service) RolesFor(ctx context.Context, org *Organization, actorID, personID string) (*PersonRoles, error) {
	if ok, err := s.holds(ctx, org, actorID, RoleAdministrator); err != nil || !ok {
		return nil, orForbidden(err)
	}
	p, err := s.GetProfile(ctx, org, personID)
	if err != nil {
		return nil, err
	}
	roles, err := s.RolesOf(ctx, org, personID)
	if err != nil {
		return nil, err
	}
	return &PersonRoles{ID: p.ID, Name: p.Name, Roles: roles}, nil
}

// SetRoles is D5's Save roles: one transaction with an audit record. It
// never removes the organization's last Organization Administrator, the
// actor included (7B.6, v1.07).
func (s *Service) SetRoles(ctx context.Context, org *Organization, actorID, personID string, roles []string) error {
	want := []string{}
	for _, r := range roles {
		if !slices.Contains(organizationRoles, r) {
			return FieldErrors{{"roles", "Choose among the organization roles."}}
		}
		if !slices.Contains(want, r) {
			want = append(want, r)
		}
	}
	slices.Sort(want)
	p, err := s.GetProfile(ctx, org, personID)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// Locking every administrator's entry keeps two administrators from
	// removing each other at the same moment.
	rows, err := tx.Query(ctx, `
		SELECT account_id FROM organization_roles
		WHERE organization_id = $1 AND role = $2 FOR UPDATE`, org.ID, RoleAdministrator)
	if err != nil {
		return err
	}
	admins, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	if !slices.Contains(admins, actorID) {
		return ErrForbidden
	}
	if slices.Contains(admins, personID) && !slices.Contains(want, RoleAdministrator) && len(admins) == 1 {
		return FieldErrors{{"roles", p.Name + " is the last Organization Administrator of " + org.Name +
			". Give another member Organization Administrator first."}}
	}
	rows, err = tx.Query(ctx,
		`SELECT role FROM organization_roles WHERE account_id = $1 AND organization_id = $2 ORDER BY role`, personID, org.ID)
	if err != nil {
		return err
	}
	had, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	if had == nil {
		had = []string{}
	}
	if slices.Equal(had, want) {
		return nil
	}
	if _, err := tx.Exec(ctx, `DELETE FROM organization_roles WHERE account_id = $1 AND organization_id = $2`, personID, org.ID); err != nil {
		return err
	}
	for _, r := range want {
		if _, err := tx.Exec(ctx,
			`INSERT INTO organization_roles (organization_id, account_id, role) VALUES ($1, $2, $3)`, org.ID, personID, r); err != nil {
			return err
		}
	}
	if err := audit(ctx, tx, org.ID, actorID, "roles.changed", map[string]any{
		"person": personID, "from": had, "to": want,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func orForbidden(err error) error {
	if err != nil {
		return err
	}
	return ErrForbidden
}
