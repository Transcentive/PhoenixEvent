package password

import (
	"strings"
	"testing"
)

// The table of 8A.1, row for row.
func TestCheckSpecExamples(t *testing.T) {
	tests := []struct {
		pw   string
		want error
	}{
		{"velvet tractor oyster pilgrim", nil},
		{"Mx7#rTq2!vLp", nil},
		{"password1234", ErrCommon},
		{"correct horse battery staple", ErrCommon},
		{"Lantern7", ErrTooShort},
	}
	for _, tt := range tests {
		if got := Check(tt.pw); got != tt.want {
			t.Errorf("Check(%q) = %v, want %v", tt.pw, got, tt.want)
		}
	}
}

func TestCheckLimits(t *testing.T) {
	if err := Check(strings.Repeat("é", 12)); err != nil {
		t.Errorf("12 accented letters: %v, want accepted", err)
	}
	if err := Check(strings.Repeat("q", 64)); err != nil {
		t.Errorf("64 characters: %v, want accepted", err)
	}
	if err := Check(strings.Repeat("q", 65)); err != ErrTooLong {
		t.Errorf("65 characters: %v, want %v", err, ErrTooLong)
	}
	if err := Check("PASSWORD1234"); err != ErrCommon {
		t.Errorf("capitals do not hide a common password: got %v", err)
	}
	if err := Check("correcthorsebatterystaple"); err != ErrCommon {
		t.Errorf("spaces removed: got %v", err)
	}
}

func TestHash(t *testing.T) {
	hash, err := Hash("velvet tractor oyster pilgrim")
	if err != nil {
		t.Fatal(err)
	}
	if !Matches(hash, "velvet tractor oyster pilgrim") {
		t.Error("the password does not match its own hash")
	}
	if Matches(hash, "velvet tractor oyster pilgrIm") {
		t.Error("a different password matches")
	}
}

// v1.05: spaces at the start and end are removed when a password is set and
// at login; spaces inside it count.
func TestOuterSpacesAreRemoved(t *testing.T) {
	hash, err := Hash("SysAdmin#1234! ")
	if err != nil {
		t.Fatal(err)
	}
	if !Matches(hash, "SysAdmin#1234!") || !Matches(hash, "  SysAdmin#1234!") {
		t.Error("an outer space made a difference")
	}
	if Matches(hash, "SysAdmin #1234!") {
		t.Error("an inner space made no difference")
	}
	if err := Check("   short pass   "); err != ErrTooShort {
		t.Errorf("outer spaces counted toward the length: %v", err)
	}
}
