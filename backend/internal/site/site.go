// Package site reads which organization a request is for from the address
// the person opened (spec 4A.1, 4A.5). The address carries the choice of
// organization; nothing the client sends in a body or a header of its own
// choosing ever does.
package site

import (
	"net"
	"strings"
)

type Kind int

const (
	// Unknown is an address that is neither the general address nor one
	// organization's address.
	Unknown Kind = iota
	// General is the general address, where screen A1 finds a person's
	// organization.
	General
	// Organization is one organization's address.
	Organization
)

// Address is what Resolve found.
type Address struct {
	Kind Kind
	// Slug is the organization's address name, set for Organization only.
	// Whether an organization has that name is for the caller to look up.
	Slug string
}

// Resolve reads host, as in a request's Host header, against orgDomain, such
// as phoenixevent.tomsherlock.space. lincolnpta.phoenixevent.tomsherlock.space
// is the organization lincolnpta; phoenixevent.tomsherlock.space is the
// general address; a.b.phoenixevent.tomsherlock.space and every other
// address are Unknown.
func Resolve(host, orgDomain string) Address {
	host = strings.ToLower(strings.TrimSuffix(stripPort(host), "."))
	orgDomain = strings.ToLower(orgDomain)
	if host == "" || orgDomain == "" {
		return Address{}
	}
	if host == orgDomain {
		return Address{Kind: General}
	}
	slug, found := strings.CutSuffix(host, "."+orgDomain)
	if !found || slug == "" || strings.Contains(slug, ".") {
		return Address{}
	}
	return Address{Kind: Organization, Slug: slug}
}

func stripPort(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}
