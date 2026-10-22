#!/usr/bin/env bash
#
# Start PhoenixEvent on this Mac: the API on 8082 and the web client on 5174.
# PostgreSQL must already be running (brew services start postgresql@16).
#
#   ./scripts/run-local.sh
#
# Ctrl-C stops both. The API's log is .local/logs/api.log.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$REPO_ROOT/backend/.env"
LOG="$REPO_ROOT/.local/logs/api.log"

say() { printf '\033[1m==> %s\033[0m\n' "$1"; }

[ -f "$ENV_FILE" ] || { echo "No backend/.env. Run ./scripts/setup-local-db.sh first." >&2; exit 1; }
[ -d "$REPO_ROOT/frontend/node_modules" ] || { say "Installing the web client's packages"; (cd "$REPO_ROOT/frontend" && npm ci); }
mkdir -p "$(dirname "$LOG")"

PORT="$(grep -E '^PORT=' "$ENV_FILE" | head -1 | cut -d= -f2-)"
PORT="${PORT:-8082}"

if lsof -tnP -iTCP:"$PORT" -sTCP:LISTEN >/dev/null 2>&1; then
  echo "Port $PORT is already in use. Stop whatever holds it, then run this again." >&2
  exit 1
fi

say "Building the API"
(cd "$REPO_ROOT/backend" && go build -o "$REPO_ROOT/.local/phoenixevent-server" ./cmd/server)

# A phone cannot look up the .localhost names, so the Mac's Wi-Fi address
# stands for the organization LAN_ORGANIZATION names in backend/.env.
LAN="$(ipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1 2>/dev/null || true)"
LAN_ORG="$(grep -E '^LAN_ORGANIZATION=' "$ENV_FILE" | head -1 | cut -d= -f2-)"

say "Starting the API on $PORT"
(
  set -a; . "$ENV_FILE"; set +a
  if [ -n "$LAN" ] && [ -n "$LAN_ORG" ]; then export LAN_ADDRESS="$LAN:5174"; fi
  cd "$REPO_ROOT/backend"
  exec "$REPO_ROOT/.local/phoenixevent-server"
) >"$LOG" 2>&1 &
API_PID=$!
trap 'kill $API_PID 2>/dev/null || true' EXIT

for _ in $(seq 1 30); do
  curl -fsS -o /dev/null -m 2 "http://127.0.0.1:$PORT/api/v1/health" 2>/dev/null && break
  if ! kill -0 $API_PID 2>/dev/null; then
    echo "The API stopped while starting. Its log:" >&2
    cat "$LOG" >&2
    exit 1
  fi
  sleep 1
done
echo "  health: $(curl -s -m 2 "http://127.0.0.1:$PORT/api/v1/health")"

say "Starting the web client: http://phoenixevent.localhost:5174"
if [ -n "$LAN" ] && [ -n "$LAN_ORG" ]; then
  echo "  On a phone on the same Wi-Fi: http://$LAN:5174 is $LAN_ORG"
fi
cd "$REPO_ROOT/frontend"
# --host listens on the Wi-Fi as well, or the phone gets nothing.
npm run dev -- --host
