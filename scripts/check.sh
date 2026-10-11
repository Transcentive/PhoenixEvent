#!/usr/bin/env bash
#
# Every automated check, in one command. Run it before committing.
#
#   ./scripts/check.sh
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
say() { printf '\n\033[1m== %s\033[0m\n' "$1"; }

say "Backend: format"
unformatted="$(cd "$REPO_ROOT/backend" && gofmt -l .)"
if [ -n "$unformatted" ]; then
  echo "Run gofmt on:" >&2
  echo "$unformatted" >&2
  exit 1
fi
echo "  formatted"

say "Backend: vet"
(cd "$REPO_ROOT/backend" && go vet ./...)

say "Backend: tests"
"$REPO_ROOT/scripts/test.sh"

say "Web client: types"
(cd "$REPO_ROOT/frontend" && npm run --silent typecheck)

say "Web client: tests"
(cd "$REPO_ROOT/frontend" && npm test --silent)

say "Web client: build"
(cd "$REPO_ROOT/frontend" && npm run --silent build)

say "All checks passed"
