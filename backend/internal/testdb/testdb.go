// Package testdb opens the throwaway database for tests that need one. It
// refuses any database whose name does not end in _test, and any host but
// this machine, however the tests are started.
package testdb

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Transcentive/PhoenixEvent/backend/internal/database"
)

// Pool returns a pool on the throwaway database, migrated, with every table
// emptied.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	raw := os.Getenv("DATABASE_URL")
	if raw == "" {
		t.Skip("DATABASE_URL is not set; run ./scripts/test.sh")
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("DATABASE_URL does not parse: %v", err)
	}
	if name := strings.TrimPrefix(u.Path, "/"); !strings.HasSuffix(name, "_test") {
		t.Fatalf("refusing to run against database %q: its name must end in _test", name)
	}
	if host := u.Hostname(); host != "localhost" && host != "127.0.0.1" && host != "::1" {
		t.Fatalf("refusing to run against host %q: tests run on this machine only", host)
	}
	_, here, _, _ := runtime.Caller(0)
	if err := database.RunMigrations(raw, filepath.Join(filepath.Dir(here), "../../migrations")); err != nil {
		t.Fatal(err)
	}
	pool, err := database.NewPool(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(context.Background(),
		`TRUNCATE audit_log, login_failures, email_links, sessions, organization_roles, accounts, organizations`); err != nil {
		t.Fatal(err)
	}
	return pool
}
