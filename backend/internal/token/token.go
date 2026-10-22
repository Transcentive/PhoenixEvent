// Package token issues and checks the tokens that keep a person signed in
// (spec 2A.3): a short-lived access token, a JWT the client sends with each
// request, and a long-lived refresh token kept in an HttpOnly cookie. Each
// access token names the one organization the person logged in to (sections
// 4 and 4A), and the session it belongs to, so that logging out ends it at
// once.
package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AccessLifetime is short, since an access token cannot be withdrawn before
// it expires except by revoking its session, which every request checks.
const AccessLifetime = 15 * time.Minute

// RefreshLifetime is how long a person stays logged in without using the
// site (8A.2). Each use starts it again.
const RefreshLifetime = 30 * 24 * time.Hour

const issuer = "phoenixevent"

// Claims is who is asking and in which organization. It carries no roles:
// those are read from the database on every request, so a change reaches the
// person on their next request.
type Claims struct {
	AccountID      string `json:"aid"`
	OrganizationID string `json:"oid"`
	SessionID      string `json:"sid"`
	jwt.RegisteredClaims
}

type Issuer struct {
	secret []byte
	now    func() time.Time
}

func NewIssuer(secret string) *Issuer {
	return &Issuer{secret: []byte(secret), now: time.Now}
}

// Access signs an access token and returns it with its expiry.
func (i *Issuer) Access(accountID, organizationID, sessionID string) (string, time.Time, error) {
	now := i.now().UTC()
	expires := now.Add(AccessLifetime)
	claims := Claims{
		AccountID:      accountID,
		OrganizationID: organizationID,
		SessionID:      sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   accountID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign access token: %w", err)
	}
	return signed, expires, nil
}

var ErrInvalid = errors.New("invalid or expired token")

// Check verifies an access token's signature and expiry and returns its
// claims.
func (i *Issuer) Check(raw string) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		return i.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithTimeFunc(i.now),
	)
	if err != nil || !parsed.Valid || claims.AccountID == "" || claims.OrganizationID == "" || claims.SessionID == "" {
		return nil, ErrInvalid
	}
	return claims, nil
}

// Secret returns a new random value for a refresh token or an emailed link.
func Secret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Hash is what the database keeps of a secret, so that a copy of the
// database holds no usable token.
func Hash(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}
