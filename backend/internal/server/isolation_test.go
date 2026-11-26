package server

import (
	"net/http"
	"testing"
)

// signIn logs a person in at host and returns the access token, the
// sign-in cookie and the account id.
func (f *fixture) signIn(t *testing.T, host, login string) (string, *http.Cookie, string) {
	t.Helper()
	rec := f.do(t, request{method: "POST", host: host, path: "/api/v1/auth/login", body: map[string]string{"login": login, "password": goodPassword}})
	if rec.Code != 200 {
		t.Fatalf("login %s at %s: %d %s", login, host, rec.Code, rec.Body)
	}
	body := decode(t, rec)
	access, _ := body["access_token"].(string)
	id, _ := body["account"].(map[string]any)["id"].(string)
	return access, cookie(rec, sessionCookie), id
}

// The Phase 2 exit (section 16A): access across organizations is refused at
// every signed-in route, for links in emails, and when a session moves to
// another organization's address.
func TestEveryRouteStaysInItsOrganization(t *testing.T) {
	f := newFixture(t)

	// A member of Riverside, made through the API as a person would.
	if rec := f.do(t, request{method: "POST", host: riversideHost, path: "/api/v1/auth/register", body: map[string]string{"email": "sam@example.org", "username": "sam", "password": goodPassword}}); rec.Code != 201 && rec.Code != 200 {
		t.Fatalf("register at Riverside: %d %s", rec.Code, rec.Body)
	}
	confirm := f.box.lastToken(t)

	// A link from one organization's email does nothing at another's address.
	if rec := f.do(t, request{method: "POST", host: ptaHost, path: "/api/v1/auth/confirm-email", body: map[string]string{"token": confirm}}); rec.Code < 400 {
		t.Errorf("Riverside's confirmation link at the PTA: %d, want a refusal", rec.Code)
	}
	if rec := f.do(t, request{method: "POST", host: riversideHost, path: "/api/v1/auth/confirm-email", body: map[string]string{"token": confirm}}); rec.Code != 200 {
		t.Fatalf("confirm at Riverside: %d %s", rec.Code, rec.Body)
	}
	if rec := f.do(t, request{method: "POST", host: ptaHost, path: "/api/v1/auth/forgot-password", body: map[string]string{"login": "dana@example.org"}}); rec.Code >= 400 {
		t.Fatalf("forgot password at the PTA: %d", rec.Code)
	}
	reset := f.box.lastToken(t)
	if rec := f.do(t, request{method: "POST", host: riversideHost, path: "/api/v1/auth/reset-password", body: map[string]string{"token": reset, "password": "another velvet tractor oyster"}}); rec.Code < 400 {
		t.Errorf("the PTA's reset link at Riverside: %d, want a refusal", rec.Code)
	}

	// A person with the same login at each organization is two accounts.
	if rec := f.do(t, request{method: "POST", host: riversideHost, path: "/api/v1/auth/login", body: map[string]string{"login": "dreyes", "password": goodPassword}}); rec.Code != 401 {
		t.Errorf("the PTA's login at Riverside: %d, want 401", rec.Code)
	}

	pta, ptaSession, dana := f.signIn(t, ptaHost, "dreyes")
	riverside, _, sam := f.signIn(t, riversideHost, "sam")

	routes := []struct{ method, path string }{
		{"GET", "/api/v1/me"},
		{"GET", "/api/v1/profile"},
		{"PUT", "/api/v1/profile"},
		{"PUT", "/api/v1/profile/visibility"},
		{"POST", "/api/v1/profile/email"},
		{"GET", "/api/v1/people/" + dana},
		{"GET", "/api/v1/people/" + dana + "/roles"},
		{"PUT", "/api/v1/people/" + dana + "/roles"},
		{"GET", "/api/v1/officers"},
		{"GET", "/api/v1/members"},
	}
	for _, r := range routes {
		// A token from one organization is refused at the other's address.
		if rec := f.do(t, request{method: r.method, host: riversideHost, path: r.path, bearer: pta, body: map[string]any{}}); rec.Code != 401 {
			t.Errorf("%s %s with the PTA's token at Riverside: %d, want 401", r.method, r.path, rec.Code)
		}
		// Without a token, nothing.
		if rec := f.do(t, request{method: r.method, host: ptaHost, path: r.path, body: map[string]any{}}); rec.Code != 401 {
			t.Errorf("%s %s without a token: %d, want 401", r.method, r.path, rec.Code)
		}
	}

	// At its own address, a session reaches no one in the other organization:
	// the same answer as for a person who does not exist (E3).
	for _, path := range []string{"/api/v1/people/" + sam, "/api/v1/people/" + sam + "/roles"} {
		if rec := f.do(t, request{method: "GET", host: ptaHost, path: path, bearer: pta}); rec.Code != 404 {
			t.Errorf("GET %s from the PTA: %d, want 404", path, rec.Code)
		}
	}
	for _, path := range []string{"/api/v1/people/" + dana, "/api/v1/people/" + dana + "/roles"} {
		if rec := f.do(t, request{method: "GET", host: riversideHost, path: path, bearer: riverside}); rec.Code != 404 {
			t.Errorf("GET %s from Riverside: %d, want 404", path, rec.Code)
		}
	}

	// The sign-in cookie does not renew a session at another address.
	if rec := f.do(t, request{method: "POST", host: riversideHost, path: "/api/v1/auth/refresh", cookies: []*http.Cookie{ptaSession}}); rec.Code != 401 {
		t.Errorf("the PTA's sign-in cookie at Riverside: %d, want 401", rec.Code)
	}
	// Logging out at Riverside leaves the PTA session as it was.
	if rec := f.do(t, request{method: "POST", host: riversideHost, path: "/api/v1/auth/logout", cookies: []*http.Cookie{ptaSession}}); rec.Code >= 500 {
		t.Errorf("logout at Riverside with the PTA's cookie: %d", rec.Code)
	}
	if rec := f.do(t, request{method: "POST", host: ptaHost, path: "/api/v1/auth/refresh", cookies: []*http.Cookie{ptaSession}}); rec.Code != 200 {
		t.Errorf("the PTA session after a logout at Riverside: %d, want 200", rec.Code)
	}
}
