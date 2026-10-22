package token

import (
	"testing"
	"time"
)

func TestAccessRoundTrip(t *testing.T) {
	iss := NewIssuer("0123456789abcdef0123456789abcdef")
	raw, expires, err := iss.Access("acc", "org", "ses")
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Until(expires); d < 14*time.Minute || d > 16*time.Minute {
		t.Errorf("expires in %v, want about %v", d, AccessLifetime)
	}
	c, err := iss.Check(raw)
	if err != nil {
		t.Fatal(err)
	}
	if c.AccountID != "acc" || c.OrganizationID != "org" || c.SessionID != "ses" {
		t.Errorf("claims = %+v", c)
	}
}

func TestCheckRefuses(t *testing.T) {
	iss := NewIssuer("0123456789abcdef0123456789abcdef")
	raw, _, _ := iss.Access("acc", "org", "ses")

	other := NewIssuer("ffffffffffffffffffffffffffffffff")
	if _, err := other.Check(raw); err == nil {
		t.Error("a token signed with another secret was accepted")
	}

	later := NewIssuer("0123456789abcdef0123456789abcdef")
	later.now = func() time.Time { return time.Now().Add(AccessLifetime + time.Minute) }
	if _, err := later.Check(raw); err == nil {
		t.Error("an expired token was accepted")
	}

	if _, err := iss.Check(raw + "x"); err == nil {
		t.Error("a tampered token was accepted")
	}
}

func TestSecretAndHash(t *testing.T) {
	a, _ := Secret()
	b, _ := Secret()
	if a == b || len(a) != 64 {
		t.Errorf("secrets %q and %q", a, b)
	}
	if h := Hash(a); h == a || h != Hash(a) || h == Hash(b) {
		t.Error("hash is not a stable one-way value")
	}
}
