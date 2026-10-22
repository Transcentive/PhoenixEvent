#!/usr/bin/env bash
#
# Run the backend tests against a throwaway database.
#
#   ./scripts/test.sh                                  # everything
#   ./scripts/test.sh ./internal/database/ -run TestRunMigrations
#
# Tests that touch the database need one nobody minds losing. This script
# points them at phoenixevent_test, creating it if needed.
#
# The tests themselves refuse any database whose name does not end in _test,
# and any host but this machine, so this script is the convenience and the
# test helper is the safeguard. Never source backend/.env and run `go test` in
# the same shell: that points the tests at the development database.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PGBIN="${PGBIN:-/usr/local/opt/postgresql@16/bin}"
PGPORT="${PGPORT:-5433}"
PGHOST="${PGHOST:-localhost}"
TEST_DB=phoenixevent_test
ENV_FILE="$REPO_ROOT/backend/.env"

say() { printf '\033[1m==> %s\033[0m\n' "$1"; }

[ -f "$ENV_FILE" ] || { echo "No backend/.env. Run ./scripts/setup-local-db.sh first." >&2; exit 1; }

# The throwaway borrows the development credentials and differs only in the
# database name.
DEV_URL="$(grep '^DATABASE_URL=' "$ENV_FILE" | cut -d= -f2-)"
TEST_URL="$(printf '%s' "$DEV_URL" | sed "s|/phoenixevent?|/$TEST_DB?|")"

# If the substitution above ever fails to match, stop here.
case "$TEST_URL" in
  *"/$TEST_DB?"*) ;;
  *) echo "Refusing: could not derive a throwaway URL from DATABASE_URL." >&2; exit 1 ;;
esac

say "Throwaway database: $TEST_DB"
if ! "$PGBIN/psql" -h "$PGHOST" -p "$PGPORT" -d postgres -tAc \
     "SELECT 1 FROM pg_database WHERE datname='$TEST_DB'" | grep -q 1; then
  owner="$(printf '%s' "$DEV_URL" | sed -E 's|^postgres://([^:]+):.*|\1|')"
  "$PGBIN/createdb" -h "$PGHOST" -p "$PGPORT" -O "$owner" "$TEST_DB"
  echo "  created"
else
  echo "  exists"
fi

say "Running tests"
cd "$REPO_ROOT/backend"
# -p 1: the packages share the one throwaway database, and each empties it
# before its tests, so they take turns.
if [ $# -gt 0 ]; then
  DATABASE_URL="$TEST_URL" go test -p 1 "$@"
else
  DATABASE_URL="$TEST_URL" go test -p 1 ./...
fi
