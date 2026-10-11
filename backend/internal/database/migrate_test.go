package database

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
)

// throwawayURL returns the database the tests may use, and refuses anything
// that could be a database someone cares about. The guard is here, and not
// only in scripts/test.sh, so it holds however the tests are started.
func throwawayURL(t *testing.T) string {
	t.Helper()
	raw := os.Getenv("DATABASE_URL")
	if raw == "" {
		t.Skip("DATABASE_URL is not set; run ./scripts/test.sh")
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("DATABASE_URL does not parse: %v", err)
	}
	name := strings.TrimPrefix(u.Path, "/")
	if !strings.HasSuffix(name, "_test") {
		t.Fatalf("refusing to run against database %q: its name must end in _test", name)
	}
	if host := u.Hostname(); host != "localhost" && host != "127.0.0.1" && host != "::1" {
		t.Fatalf("refusing to run against host %q: tests run on this machine only", host)
	}
	return raw
}

func TestRunMigrations(t *testing.T) {
	databaseURL := throwawayURL(t)

	// Twice: the second run must find nothing to do and still succeed, since
	// the server runs this on every start.
	for i := 0; i < 2; i++ {
		if err := RunMigrations(databaseURL, "../../migrations"); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}

	ctx := context.Background()
	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	var value string
	if err := pool.QueryRow(ctx, `SELECT value FROM app_info WHERE key = 'installed_at'`).Scan(&value); err != nil {
		t.Fatalf("app_info was not created by the first migration: %v", err)
	}
	if value == "" {
		t.Error("installed_at is empty")
	}
}
