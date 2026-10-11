# The Mighty Queue — tech stack and hosting (reference)

**What this is:** a reference describing a different project, The Mighty Queue,
for use in the Phoenix Event project. It is not Phoenix Event's own stack.

**As of:** 2026-10-08. The hosting facts were read from the droplet that day.
The tech stack was read from the specification (v4.16.1) and the dependency
files that day.

## 1. Tech stack

| Layer | Technology |
| --- | --- |
| Database | PostgreSQL 16; two databases, active and archive |
| Backend | Go 1.25 with Gin 1.12; pgx 5 for the database; migrations applied by the server at startup (golang-migrate) |
| Sign-in | JWT access token plus a refresh token in an HttpOnly cookie; bcrypt for passwords |
| Web client | React 18, TypeScript 5.6, Bootstrap 5.3, Vite 6, React Query 5, React Router 6 |
| Mobile client | React Native 0.86 on Expo SDK 57, React 19, TypeScript 6, React Navigation 7, React Query 5, react-native-svg |
| Mobile build | Expo cloud builds (EAS Build) and EAS Update |
| Hosting | Ubuntu droplet on DigitalOcean, no containers: the Go binary under systemd, nginx in front, TLS by Let's Encrypt |
| Email | SMTP through Rackspace |
| Exchange rates | Frankfurter |
| Tests | Go's testing package; Vitest, React Testing Library and MSW on the web; Jest on mobile |
| Development | macOS: Homebrew PostgreSQL 16 on port 5433, Vite on 5173, Expo bundler on 8083, Android emulator |

Planned, not built or not checked: Playwright end-to-end tests (roadmap
Phase 6); OneDrive and S3 storage adapters and OAuth sign-in providers
(roadmap Phase 5).

## 2. Hosting on DigitalOcean

### The droplet

| | |
| --- | --- |
| Provider | DigitalOcean; DNS for `tomsherlock.space` is also there |
| Address | `68.183.59.71` |
| Operating system | Ubuntu 22.04.4 LTS |
| Size | 1 vCPU, 957 MB memory (559 MB available when read) |
| Main disk | 25 GB, 13 GB used (54%) |
| Archive volume | 25 GB block storage, 12 MB used (1%) |
| Uptime | 16 weeks 4 days without a restart |
| Firewall | ufw active: SSH rate-limited, web ports (80, 443) open |

### What runs on it

- **Two sites share the droplet:** `tomsherlock.space` and
  `themightyqueue.tomsherlock.space`.
- **Ports already taken:** 8080 by the other service on the droplet, 8081 by
  The Mighty Queue's API. PostgreSQL is on 5432, bound to localhost only.
- **No containers.** PostgreSQL 16, the Go binary under systemd
  (`mightyqueue.service`) and nginx are installed directly on the host.
- **nginx is the only publicly bound process.** It serves the built web
  client, proxies `/api/` to the backend, and handles TLS through
  Let's Encrypt.
- **Two databases in one PostgreSQL instance**, active and archive, the
  archive on the block-storage volume through a tablespace.
- **Response headers:** `nosniff`, `X-Frame-Options DENY`,
  `Referrer-Policy same-origin`. HSTS is deliberately not set.

### State when read (2026-10-08)

| | |
| --- | --- |
| Site | `https://themightyqueue.tomsherlock.space` answers 200 |
| Build | `78d3c5069569`, deployed 2026-09-29 |
| Database schema | 60 (as of the 2026-09-29 deploy; not reread) |
| Services | mightyqueue, nginx and PostgreSQL 16 all active |
| Certificate | Expires 2026-11-14; renewal timer not checked |
| Backups | 17 pre-deploy dump folders under `/root/mq-backups`, 2.1 MB in all; the newest is `pre-deploy-2026-09-29` |

### How it is deployed

- `scripts/deploy.sh` builds on the Mac, then ships the built web client, the
  migrations, the server binary and the application's nginx snippets to the
  droplet and restarts the service.
- Before every deploy both databases are dumped on the droplet, and the live
  tasks are compared before and after.
- The mobile client is never deployed to the droplet. It is built and updated
  through Expo and only talks to the API there.

## 3. Points that matter to another project on this droplet

- It is a small machine: 1 vCPU and under 1 GB of memory, already serving two
  sites.
- Ports 8080 and 8081 are in use.
- The front door is nginx with Let's Encrypt, not Caddy.
- PostgreSQL 16 is already installed and running there.
