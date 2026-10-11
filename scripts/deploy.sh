#!/usr/bin/env bash
#
# Build on the Mac, ship to the droplet, restart (spec 2A.3). Run from the Mac:
#
#   ./scripts/deploy.sh root@droplet
#
# What lands on the server is one binary, the migrations, a directory of
# static files and the application's two nginx snippets. The database is
# dumped on the droplet before anything changes.
#
# Migrations are not run separately: the binary applies them on start and
# fails loudly if one is rejected.
#
# The nginx site files are never shipped. Each carries an address and the
# certificate lines certbot writes.
set -euo pipefail

TARGET="${1:-}"
APP_ROOT=/srv/phoenixevent/app
SERVICE_USER=phoenixevent
DB=phoenixevent
BACKUP_ROOT=/root/pe-backups
NGINX_SNIPPET_DIR=/etc/nginx/snippets

[ -n "$TARGET" ] || { echo "Usage: $0 user@host" >&2; exit 1; }

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

say() { printf '\n\033[1m== %s\033[0m\n' "$1"; }

# Every ssh and rsync below shares ONE connection. The droplet rate-limits SSH
# (ufw limit 22/tcp), and a deploy that opened a dozen connections would lock
# itself out partway through.
CTL="/tmp/pe-deploy-$$.sock"
SSH_OPTS=(-o ControlMaster=auto -o ControlPath="$CTL" -o ControlPersist=120 -o BatchMode=yes)
cleanup_ssh() { ssh -o ControlPath="$CTL" -O exit "$TARGET" 2>/dev/null || true; }
trap cleanup_ssh EXIT

rsh() { ssh "${SSH_OPTS[@]}" "$@"; }

say "Checks"
if [ -n "$(git status --porcelain)" ]; then
  echo "  ! Working tree is dirty. You are about to deploy something that is not committed."
  printf '  Continue? [y/N] '
  read -r reply
  [ "$reply" = "y" ] || exit 1
fi
BUILD="$(git rev-parse --short=12 HEAD)"
echo "  deploying $BUILD on $(git branch --show-current)"

say "Build: backend (linux/amd64)"
mkdir -p .local
( cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w -X main.build=$BUILD" -o ../.local/phoenixevent-server-linux ./cmd/server )
echo "  .local/phoenixevent-server-linux  $(du -h .local/phoenixevent-server-linux | cut -f1)"

say "Build: web client"
( cd frontend && npm ci --silent && npm run --silent build )
echo "  frontend/dist  $(du -sh frontend/dist | cut -f1)"

say "Dump the database"
# Before anything on the droplet changes (spec 2A.3). The dump is read back
# with pg_restore --list, so a dump that cannot be read stops the deploy.
STAMP="$(date -u +%Y-%m-%d-%H%M%S)"
rsh "$TARGET" bash -euo pipefail <<EOF2
  DIR=$BACKUP_ROOT/pre-deploy-$STAMP
  mkdir -p "\$DIR"
  chmod 700 $BACKUP_ROOT "\$DIR"
  cd /tmp
  sudo -u postgres pg_dump -Fc $DB > "\$DIR/$DB.dump"
  pg_restore --list "\$DIR/$DB.dump" >/dev/null
  echo "  \$DIR/$DB.dump  \$(du -h "\$DIR/$DB.dump" | cut -f1)"
EOF2

say "Ship"
# The binary goes to a temporary name and is moved into place after the unit
# stops, so a failed transfer cannot leave a half-written binary behind.
rsync -az --delete -e "ssh ${SSH_OPTS[*]}" frontend/dist/ "$TARGET:$APP_ROOT/frontend/dist/"
rsync -az --delete -e "ssh ${SSH_OPTS[*]}" backend/migrations/ "$TARGET:$APP_ROOT/backend/migrations/"
rsync -az -e "ssh ${SSH_OPTS[*]}" .local/phoenixevent-server-linux "$TARGET:$APP_ROOT/backend/phoenixevent-server.new"

say "Swap and restart"
rsh "$TARGET" bash -euo pipefail <<EOF2
  systemctl stop phoenixevent
  mv "$APP_ROOT/backend/phoenixevent-server.new" "$APP_ROOT/backend/phoenixevent-server"
  chmod 755 "$APP_ROOT/backend/phoenixevent-server"
  chown -R $SERVICE_USER:$SERVICE_USER "$APP_ROOT"
  systemctl start phoenixevent
EOF2

say "Ship: nginx snippets"
# Tested before nginx is reloaded, and the previous files are put back if the
# test fails, so a bad rule cannot take down the sites on this droplet.
# No --delete: /etc/nginx/snippets holds the host's own files too.
rsync -az -e "ssh ${SSH_OPTS[*]}" deploy/nginx/snippets/ "$TARGET:/tmp/pe-nginx/"
rsh "$TARGET" bash -euo pipefail <<EOF2
  DIR=$NGINX_SNIPPET_DIR
  STAGE=/tmp/pe-nginx
  changed=no
  for f in "\$STAGE"/*.conf; do
    t="\$DIR/\$(basename "\$f")"
    if [ ! -f "\$t" ] || ! cmp -s "\$f" "\$t"; then changed=yes; fi
  done
  if [ "\$changed" = no ]; then
    echo "  unchanged"
    rm -rf "\$STAGE"
    exit 0
  fi
  rm -rf /tmp/pe-nginx.prev
  mkdir -p /tmp/pe-nginx.prev
  for f in "\$STAGE"/*.conf; do
    t="\$DIR/\$(basename "\$f")"
    [ -f "\$t" ] && cp "\$t" "/tmp/pe-nginx.prev/\$(basename "\$f")" || true
  done
  for f in "\$STAGE"/*.conf; do install -m 0644 "\$f" "\$DIR/\$(basename "\$f")"; done
  if nginx -t >/dev/null 2>&1; then
    systemctl reload nginx
    echo "  installed and reloaded"
  else
    for f in "\$STAGE"/*.conf; do
      p="/tmp/pe-nginx.prev/\$(basename "\$f")"
      if [ -f "\$p" ]; then cp "\$p" "\$DIR/\$(basename "\$f")"; else rm -f "\$DIR/\$(basename "\$f")"; fi
    done
    echo "  ! nginx refused the new snippets. The previous ones are back in place" >&2
    echo "    and nginx was not reloaded. nginx said:" >&2
    nginx -t >&2 2>&1 || true
    rm -rf "\$STAGE" /tmp/pe-nginx.prev
    exit 1
  fi
  rm -rf "\$STAGE" /tmp/pe-nginx.prev
EOF2

say "Verify"
# The port is read from the deployed .env. On a shared host a wrong number
# would health-check a neighbor's service and report a success that means
# nothing, so the answer must also name this build.
PORT="$(rsh "$TARGET" "grep -E '^PORT=' $APP_ROOT/backend/.env | head -1 | cut -d= -f2-" 2>/dev/null || true)"
PORT="${PORT:-8082}"
echo "  health-checking 127.0.0.1:$PORT for build $BUILD"

body="$(rsh "$TARGET" "for _ in \$(seq 1 30); do
  b=\$(curl -s -m 2 http://127.0.0.1:$PORT/api/v1/health 2>/dev/null || true)
  case \"\$b\" in *'\"build\":\"$BUILD\"'*'\"status\":\"ok\"'*) echo \"\$b\"; exit 0 ;; esac
  sleep 1
done; echo \"\$b\"" 2>/dev/null || true)"

case "$body" in
  *"\"build\":\"$BUILD\""*"\"status\":\"ok\""*)
    echo "  $body"
    echo "  deployed"
    ;;
  *)
    echo "  ! health check did not pass. It answered: $body" >&2
    echo "    Recent log:" >&2
    rsh "$TARGET" "journalctl -u phoenixevent -n 30 --no-pager" >&2
    exit 1
    ;;
esac
