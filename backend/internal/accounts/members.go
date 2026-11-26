package accounts

import (
	"context"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Member is one person on O2, Members and roles (4A.10).
type Member struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Email *string  `json:"email,omitempty"`
	Roles []string `json:"roles"`
}

var positionLabels = map[string]string{
	"chair":               "Event Chair",
	"member":              "Board Member",
	"event_administrator": "Event Administrator",
}

// Members lists the organization's members who take part in an event, current
// or past (4A.10). Appointment to an event board makes a person a participant
// (5D.1); registrations and stations add to this once they are built. Each
// member shows their roles in those events. The email address and the roles
// follow the levels the member chose (8B.4): public and organization members
// are seen by every member; event participants by those taking part in the
// same event; station members and private by nobody here until stations exist.
func (s *Service) Members(ctx context.Context, org *Organization, viewerID string) ([]Member, error) {
	type seat struct {
		account, event, eventName, position string
		deputy                              bool
	}
	rows, err := s.db.Query(ctx, `
		SELECT b.account_id, b.event_id, e.name, b.position, b.deputy
		FROM event_board b JOIN events e ON e.id = b.event_id AND e.organization_id = b.organization_id
		WHERE b.organization_id = $1
		ORDER BY e.event_date NULLS LAST, e.name, b.position`, org.ID)
	if err != nil {
		return nil, err
	}
	seats, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (seat, error) {
		var x seat
		err := r.Scan(&x.account, &x.event, &x.eventName, &x.position, &x.deputy)
		return x, err
	})
	if err != nil {
		return nil, err
	}
	viewerEvents := map[string]bool{}
	byAccount := map[string][]seat{}
	order := []string{}
	for _, x := range seats {
		if x.account == viewerID {
			viewerEvents[x.event] = true
		}
		if _, ok := byAccount[x.account]; !ok {
			order = append(order, x.account)
		}
		byAccount[x.account] = append(byAccount[x.account], x)
	}
	members := []Member{}
	for _, id := range order {
		p, err := s.GetProfile(ctx, org, id)
		if err != nil {
			return nil, err
		}
		self := id == viewerID
		shares := func() bool {
			for _, x := range byAccount[id] {
				if viewerEvents[x.event] {
					return true
				}
			}
			return false
		}
		sees := func(level string) bool {
			switch level {
			case LevelPublic, LevelOrganization:
				return true
			case LevelEvent:
				return self || shares()
			}
			return self
		}
		m := Member{ID: p.ID, Name: p.Name, Roles: []string{}}
		if sees(p.EmailLevel) {
			m.Email = &p.Email
		}
		for _, x := range byAccount[id] {
			if !sees(p.RolesLevel) {
				break
			}
			if p.RolesLevel == LevelEvent && !self && !viewerEvents[x.event] {
				continue
			}
			label := positionLabels[x.position] + ", " + x.eventName
			if x.deputy {
				label += " (Deputy Board Chair)"
			}
			m.Roles = append(m.Roles, label)
		}
		members = append(members, m)
	}
	slices.SortFunc(members, func(a, b Member) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	return members, nil
}
