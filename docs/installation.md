# PhoenixEvent: installation on the droplet

First deployment notes, written with Phase 1 on October 11, 2026. **These
steps have not yet been run on the droplet.** They become the tested
installation guide of specification section 14 once they have been.

They follow specification 2A.3 and 2A.4: testing runs on the droplet that
already serves The Mighty Queue and tomsherlock.space, as a third service on
port 8082 with its own database in the PostgreSQL 16 already there. Nothing
here installs a package or changes what is already running.

## What the droplet ends up with

| Piece | Where |
| --- | --- |
| The API | `/srv/phoenixevent/app/backend/phoenixevent-server`, run by the `phoenixevent` systemd unit as the `phoenixevent` user, listening on `127.0.0.1:8082` |
| Its settings | `/srv/phoenixevent/app/backend/.env`, readable by the service user only |
| Migrations | `/srv/phoenixevent/app/backend/migrations/`, applied by the API each time it starts |
| The web client | `/srv/phoenixevent/app/frontend/dist/`, served by nginx |
| Stored files | `/srv/phoenixevent/data/files/` |
| The database | `phoenixevent`, owned by the role `phoenixevent`, in the existing PostgreSQL 16 |
| nginx | One site file per address in `/etc/nginx/sites-available/`, and two snippets in `/etc/nginx/snippets/` |
| Database dumps | `/root/pe-backups/pre-deploy-<date and time>/`, one before every deploy |

## Before the first deploy

### 1. DNS (the owner, at DigitalOcean)

Add an A record pointing to the droplet for `phoenixevent.tomsherlock.space`
and for each test organization's address, such as
`lincolnpta.phoenixevent.tomsherlock.space` (specification 16B.5).

### 2. Provision the droplet (once)

From the Mac, copy the two files over, then run the script on the droplet:

```sh
scp scripts/setup-droplet-shared.sh deploy/phoenixevent.service root@droplet:/root/
ssh root@droplet 'cd /root && ./setup-droplet-shared.sh'
```

It adds the PostgreSQL role and database, the service user, the directories,
the environment file and the systemd unit. It is safe to run again.

### 3. Email (when the Rackspace account exists)

Edit `/srv/phoenixevent/app/backend/.env` on the droplet and fill in
`SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD` and `EMAIL_FROM`.
Until then the API writes each message to its log in place of sending it.

## Deploy

From the Mac, on a clean working tree:

```sh
./scripts/deploy.sh root@droplet
```

The script builds the API and the web client on the Mac, dumps the database on
the droplet, ships the build, restarts the service, installs the nginx
snippets, and finishes only when the health check on the droplet answers `ok`
with the build it just shipped.

## An nginx site and certificate for each address

After the first deploy, so the snippets exist. On the droplet, once per address:

```sh
ADDRESS=lincolnpta.phoenixevent.tomsherlock.space
cp phoenixevent.conf /etc/nginx/sites-available/$ADDRESS.conf     # from deploy/nginx/
sed -i "s/PHOENIXEVENT_ADDRESS/$ADDRESS/g" /etc/nginx/sites-available/$ADDRESS.conf
ln -sf /etc/nginx/sites-available/$ADDRESS.conf /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx
certbot --nginx -d $ADDRESS
```

certbot adds the HTTPS listener, the certificate and the redirect from plain
HTTP to the site file. Each address gets its own certificate (specification
4A.5).

## Checking it

```sh
ssh root@droplet 'systemctl is-active phoenixevent; curl -s http://127.0.0.1:8082/api/v1/health'
curl -s https://lincolnpta.phoenixevent.tomsherlock.space/api/v1/health
```

Both health checks answer `{"build":"…","database":"ok","status":"ok"}`.

## Going back

- **The code:** check out the earlier commit on the Mac and deploy it.
- **The database:** each deploy's dump is in `/root/pe-backups/`. Restoring one
  replaces the database's contents, so it is done only on the owner's word:
  `sudo -u postgres pg_restore --clean --if-exists -d phoenixevent <file>`.
