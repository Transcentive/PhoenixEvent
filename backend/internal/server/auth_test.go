package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Transcentive/PhoenixEvent/backend/internal/accounts"
	"github.com/Transcentive/PhoenixEvent/backend/internal/config"
	"github.com/Transcentive/PhoenixEvent/backend/internal/email"
	"github.com/Transcentive/PhoenixEvent/backend/internal/testdb"
	"github.com/Transcentive/PhoenixEvent/backend/internal/token"
)

type outbox struct {
	mu   sync.Mutex
	sent []email.Message
}

func (o *outbox) Send(_ context.Context, m email.Message) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.sent = append(o.sent, m)
	return nil
}

func (o *outbox) lastToken(t *testing.T) string {
	t.Helper()
	o.mu.Lock()
	defer o.mu.Unlock()
	m := regexp.MustCompile(`token=([0-9a-f]{64})`).FindStringSubmatch(o.sent[len(o.sent)-1].Text)
	if m == nil {
		t.Fatal("no link in the last email")
	}
	return m[1]
}

const (
	ptaHost       = "lincolnpta.phoenixevent.localhost:5174"
	riversideHost = "riverside.phoenixevent.localhost:5174"
	generalHost   = "phoenixevent.localhost:5174"
	lanHost       = "192.168.1.189:5174"
	goodPassword  = "velvet tractor oyster pilgrim"
)

type fixture struct {
	router *gin.Engine
	box    *outbox
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	site := config.Site{OrgDomain: "phoenixevent.localhost", Scheme: "http", Port: "5174",
		LANAddress: lanHost, LANOrganization: "lincolnpta"}
	tokens := token.NewIssuer("0123456789abcdef0123456789abcdef")
	box := &outbox{}
	svc := accounts.NewService(testdb.Pool(t), box, site, tokens)

	ctx := context.Background()
	pta, err := svc.CreateOrganization(ctx, "Lincoln Elementary PTA", "lincolnpta", "admin@example.org")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateOrganization(ctx, "Riverside Community Committee", "riverside", "admin2@example.org"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Register(ctx, pta, "dana@example.org", "dreyes", goodPassword); err != nil {
		t.Fatal(err)
	}
	if err := svc.ConfirmEmail(ctx, pta, box.lastToken(t)); err != nil {
		t.Fatal(err)
	}
	return &fixture{router: New(Deps{DB: fakeDB{}, Build: "test", Accounts: svc, Tokens: tokens, Site: site}), box: box}
}

type request struct {
	method, host, path string
	body               any
	bearer             string
	cookies            []*http.Cookie
}

func (f *fixture) do(t *testing.T, r request) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	if r.body != nil {
		_ = json.NewEncoder(&body).Encode(r.body)
	}
	req := httptest.NewRequest(r.method, r.path, &body)
	req.Host = r.host
	req.Header.Set("Content-Type", "application/json")
	if r.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+r.bearer)
	}
	for _, c := range r.cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

func cookie(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %s", rec.Body.String())
	}
	return body
}

func TestAddressKinds(t *testing.T) {
	f := newFixture(t)
	if rec := f.do(t, request{method: "GET", host: generalHost, path: "/api/v1/address"}); rec.Code != 200 || decode(t, rec)["kind"] != "general" {
		t.Errorf("general address: %d %s", rec.Code, rec.Body)
	}
	rec := f.do(t, request{method: "GET", host: ptaHost, path: "/api/v1/address"})
	org, _ := decode(t, rec)["organization"].(map[string]any)
	if rec.Code != 200 || org["name"] != "Lincoln Elementary PTA" {
		t.Errorf("organization address: %d %s", rec.Code, rec.Body)
	}
	for _, host := range []string{"nosuch.phoenixevent.localhost:5174", "example.com"} {
		if rec := f.do(t, request{method: "GET", host: host, path: "/api/v1/address"}); rec.Code != 404 {
			t.Errorf("%s: %d, want 404", host, rec.Code)
		}
		if rec := f.do(t, request{method: "POST", host: host, path: "/api/v1/auth/login", body: map[string]string{"login": "dreyes", "password": goodPassword}}); rec.Code != 404 {
			t.Errorf("login at %s: %d, want 404", host, rec.Code)
		}
	}
	// The A1 lookup lives at the general address only.
	if rec := f.do(t, request{method: "POST", host: ptaHost, path: "/api/v1/find-organization", body: map[string]string{"email": "dana@example.org"}}); rec.Code != 404 {
		t.Errorf("lookup at an organization's address: %d", rec.Code)
	}
	if rec := f.do(t, request{method: "POST", host: generalHost, path: "/api/v1/find-organization", body: map[string]string{"email": "dana@example.org"}}); rec.Code != 202 {
		t.Errorf("lookup at the general address: %d", rec.Code)
	}
}

// A1's Organization field matches the name and the address name (spec v1.02).
func TestMatchOrganizations(t *testing.T) {
	f := newFixture(t)
	match := func(host, q string) (int, []string) {
		rec := f.do(t, request{method: "GET", host: host, path: "/api/v1/organizations?q=" + q})
		var body struct {
			Organizations []struct {
				Name     string `json:"name"`
				LoginURL string `json:"login_url"`
			} `json:"organizations"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		var names []string
		for _, o := range body.Organizations {
			names = append(names, o.Name+" "+o.LoginURL)
		}
		return rec.Code, names
	}
	cases := map[string][]string{
		"lincoln": {"Lincoln Elementary PTA http://lincolnpta.phoenixevent.localhost:5174/login"},
		"PTA":     {"Lincoln Elementary PTA http://lincolnpta.phoenixevent.localhost:5174/login"},
		"rivers":  {"Riverside Community Committee http://riverside.phoenixevent.localhost:5174/login"},
		"e":       {"Lincoln Elementary PTA http://lincolnpta.phoenixevent.localhost:5174/login", "Riverside Community Committee http://riverside.phoenixevent.localhost:5174/login"},
		"%25":     nil,
		"":        nil,
	}
	for q, want := range cases {
		code, got := match(generalHost, q)
		if code != 200 || strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("q=%q: %d %v, want %v", q, code, got, want)
		}
	}
	if code, _ := match(ptaHost, "lincoln"); code != 404 {
		t.Errorf("at an organization's address: %d, want 404", code)
	}
}

// 4A.5: the sign-in cookie is for this address only; pe_org is for every
// organization's address; logging out deletes both.
func TestLoginCookiesAndOneOrganizationPerBrowser(t *testing.T) {
	f := newFixture(t)

	rec := f.do(t, request{method: "POST", host: ptaHost, path: "/api/v1/auth/login", body: map[string]string{"login": "dreyes", "password": goodPassword}})
	if rec.Code != 200 {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	access, _ := decode(t, rec)["access_token"].(string)
	session, org := cookie(rec, sessionCookie), cookie(rec, orgCookie)
	if session == nil || session.Domain != "" || session.Path != sessionCookiePath || !session.HttpOnly {
		t.Errorf("sign-in cookie = %+v", session)
	}
	if org == nil || org.Value != "lincolnpta" || org.Domain != "phoenixevent.localhost" {
		t.Errorf("organization cookie = %+v", org)
	}
	if strings.Contains(rec.Body.String(), session.Value) {
		t.Error("the refresh token is in the response body")
	}

	// The access token works here and nowhere else.
	if rec := f.do(t, request{method: "GET", host: ptaHost, path: "/api/v1/me", bearer: access}); rec.Code != 200 {
		t.Errorf("me at the PTA: %d", rec.Code)
	}
	if rec := f.do(t, request{method: "GET", host: riversideHost, path: "/api/v1/me", bearer: access}); rec.Code != 401 {
		t.Errorf("the PTA's token at Riverside: %d, want 401", rec.Code)
	}
	if rec := f.do(t, request{method: "GET", host: ptaHost, path: "/api/v1/me"}); rec.Code != 401 {
		t.Errorf("me without a token: %d, want 401", rec.Code)
	}

	// The same browser at Riverside: told it is logged in to the PTA, with
	// the way to log out of it, and its login refused.
	rec = f.do(t, request{method: "GET", host: riversideHost, path: "/api/v1/address", cookies: []*http.Cookie{org}})
	other, _ := decode(t, rec)["logged_in_elsewhere"].(map[string]any)
	if other["name"] != "Lincoln Elementary PTA" || !strings.HasPrefix(other["logout_url"].(string), "http://lincolnpta.phoenixevent.localhost:5174/logout?return=") {
		t.Errorf("logged_in_elsewhere = %v", other)
	}
	rec = f.do(t, request{method: "POST", host: riversideHost, path: "/api/v1/auth/login", cookies: []*http.Cookie{org}, body: map[string]string{"login": "x", "password": "y"}})
	if rec.Code != 409 {
		t.Errorf("login at Riverside while logged in to the PTA: %d, want 409", rec.Code)
	}

	// Refresh renews both cookies and replaces the refresh token.
	rec = f.do(t, request{method: "POST", host: ptaHost, path: "/api/v1/auth/refresh", cookies: []*http.Cookie{session}})
	renewed := cookie(rec, sessionCookie)
	if rec.Code != 200 || renewed == nil || renewed.Value == session.Value || cookie(rec, orgCookie) == nil {
		t.Fatalf("refresh: %d, cookie %+v", rec.Code, renewed)
	}
	access, _ = decode(t, rec)["access_token"].(string)

	// Logging out deletes both cookies and ends the session at once.
	rec = f.do(t, request{method: "POST", host: ptaHost, path: "/api/v1/auth/logout", cookies: []*http.Cookie{renewed}})
	if c := cookie(rec, sessionCookie); rec.Code != 200 || c == nil || c.MaxAge >= 0 {
		t.Errorf("logout sign-in cookie: %d %+v", rec.Code, c)
	}
	if c := cookie(rec, orgCookie); c == nil || c.MaxAge >= 0 || c.Domain != "phoenixevent.localhost" {
		t.Errorf("logout organization cookie: %+v", c)
	}
	if rec := f.do(t, request{method: "GET", host: ptaHost, path: "/api/v1/me", bearer: access}); rec.Code != 401 {
		t.Errorf("me after logout: %d, want 401", rec.Code)
	}
}

// On a development Mac, the Wi-Fi address stands for one organization so a
// phone can reach it; its organization cookie is for that address only.
func TestLANAddress(t *testing.T) {
	f := newFixture(t)
	rec := f.do(t, request{method: "GET", host: lanHost, path: "/api/v1/address"})
	org, _ := decode(t, rec)["organization"].(map[string]any)
	if rec.Code != 200 || org["name"] != "Lincoln Elementary PTA" {
		t.Fatalf("address at the Wi-Fi address: %d %s", rec.Code, rec.Body)
	}
	rec = f.do(t, request{method: "POST", host: lanHost, path: "/api/v1/auth/login", body: map[string]string{"login": "dreyes", "password": goodPassword}})
	if c := cookie(rec, orgCookie); rec.Code != 200 || c == nil || c.Domain != "" {
		t.Errorf("login at the Wi-Fi address: %d, organization cookie %+v", rec.Code, c)
	}
	if rec := f.do(t, request{method: "GET", host: "192.168.1.189:9999", path: "/api/v1/address"}); rec.Code != 404 {
		t.Errorf("another port at that address: %d, want 404", rec.Code)
	}
}

func TestLoginErrors(t *testing.T) {
	f := newFixture(t)
	login := func(pw string) *httptest.ResponseRecorder {
		return f.do(t, request{method: "POST", host: ptaHost, path: "/api/v1/auth/login", body: map[string]string{"login": "dreyes", "password": pw}})
	}
	for i := 0; i < 5; i++ {
		if rec := login("not the password at all"); rec.Code != 401 || decode(t, rec)["code"] != "bad_credentials" {
			t.Fatalf("failure %d: %d %s", i+1, rec.Code, rec.Body)
		}
	}
	if rec := login(goodPassword); rec.Code != 429 {
		t.Errorf("sixth attempt: %d, want 429", rec.Code)
	}

	rec := f.do(t, request{method: "POST", host: ptaHost, path: "/api/v1/auth/register", body: map[string]string{"email": "lee@example.org", "password": "password1234"}})
	fields, _ := decode(t, rec)["fields"].([]any)
	if rec.Code != 422 || len(fields) != 1 {
		t.Errorf("register with a common password: %d %s", rec.Code, rec.Body)
	}
	rec = f.do(t, request{method: "POST", host: ptaHost, path: "/api/v1/auth/register", body: map[string]string{"email": "lee@example.org", "password": goodPassword}})
	if rec.Code != 201 {
		t.Fatalf("register: %d %s", rec.Code, rec.Body)
	}
	rec = f.do(t, request{method: "POST", host: ptaHost, path: "/api/v1/auth/login", body: map[string]string{"login": "lee@example.org", "password": goodPassword}})
	if rec.Code != 403 || decode(t, rec)["code"] != "email_not_confirmed" {
		t.Errorf("login before confirming: %d %s", rec.Code, rec.Body)
	}

	// A PTA link means nothing at Riverside.
	link := f.box.lastToken(t)
	if rec := f.do(t, request{method: "POST", host: riversideHost, path: "/api/v1/auth/confirm-email", body: map[string]string{"token": link}}); rec.Code != 410 {
		t.Errorf("PTA link at Riverside: %d, want 410", rec.Code)
	}
	if rec := f.do(t, request{method: "POST", host: ptaHost, path: "/api/v1/auth/confirm-email", body: map[string]string{"token": link}}); rec.Code != 200 {
		t.Errorf("confirm: %d", rec.Code)
	}
}
