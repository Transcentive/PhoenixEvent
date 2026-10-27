// Package password holds the password rule (spec 8A.1) and the hashing that
// stores a password (2A.3: bcrypt, as in The Mighty Queue).
package password

import (
	_ "embed"
	"errors"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	MinLength = 12
	MaxLength = 64

	// The same cost as The Mighty Queue.
	bcryptCost = 12
)

// common-passwords.txt is the list of common and leaked passwords that 8A.1
// refuses. It was built on October 20, 2026 from three public lists in
// SecLists (Passwords/Common-Credentials: xato-net-10-million-passwords-1000000,
// Pwdb_top-1000000, 100k-most-used-passwords-NCSC), keeping only entries of
// 12 to 64 characters, since shorter ones are refused by length alone, in
// lower case, with widely published examples such as "correct horse battery
// staple" added.
//
//go:embed common-passwords.txt
var commonList string

var common = func() map[string]struct{} {
	set := make(map[string]struct{}, 80000)
	for line := range strings.SplitSeq(commonList, "\n") {
		if line != "" {
			set[line] = struct{}{}
		}
	}
	return set
}()

var (
	ErrTooShort = errors.New("A password has at least 12 characters.")
	ErrTooLong  = errors.New("A password has at most 64 characters.")
	ErrCommon   = errors.New("This password is on the list of common passwords. Choose another.")
)

// trim removes spaces at the start and end of a password, when it is set
// and at login, so a stray space from copying and pasting locks nobody out
// (8A.1, v1.05). Spaces inside it count.
func trim(pw string) string {
	return strings.TrimSpace(pw)
}

// Check applies the rule of 8A.1: 12 to 64 characters, any characters, and
// not on the list. Characters are counted as a person sees them, so "é" is
// one.
func Check(pw string) error {
	pw = trim(pw)
	n := utf8.RuneCountInString(pw)
	switch {
	case n < MinLength:
		return ErrTooShort
	case n > MaxLength:
		return ErrTooLong
	}
	lower := strings.ToLower(pw)
	if _, found := common[lower]; found {
		return ErrCommon
	}
	// "correcthorsebatterystaple" and "correct horse battery staple" are the
	// same password to a guesser.
	if _, found := common[strings.ReplaceAll(lower, " ", "")]; found {
		return ErrCommon
	}
	return nil
}

func Hash(pw string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(trim(pw)), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func Matches(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(trim(pw))) == nil
}
