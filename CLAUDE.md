# PhoenixEvent: instructions for Claude

## Wording

- We do not use Row as a metaphor; it is only relevant in a db table or a visible table.
- US spelling, in replies, code comments, documents and commits.

## The specification governs

- `docs/specification/` holds the specification. The highest version number is
  the current one. Read the sections that bear on a task before starting it.
- **Labels.** Confirmed means the owner stated it. Proposed is a starting design
  awaiting the owner's approval: keep it easy to change. Open is undecided:
  build nothing that depends on one answer, and ask the owner.
- Where a lettered section (2A, 5D) and a numbered section differ, the lettered
  section governs.
- The specification is revised in the claude.ai project "Phoenix Event". A new
  version is added to `docs/specification/` beside the earlier ones. A version
  already here is never overwritten or edited.
- `docs/README.md` describes the rest of `docs/`: the design system, the 62
  wireframe screens, the roadmap, and The Mighty Queue reference.

## Where the project stands

Roadmap Phase 1, the repository and development foundation, is built and runs
on the Mac. Phase 2 (accounts, organization context and RBAC) is under way: its
first step, organizations and sign-in (the create-organization server command,
screens A1 to A7, E1 and E2), is built and was tested by the owner on the Mac
and a phone on October 22, 2026. Nothing has been deployed to the droplet.

On a development Mac each organization is at
`http://<name>.phoenixevent.localhost:5174`. A phone on the same Wi-Fi reaches
the organization `LAN_ORGANIZATION` names in `backend/.env` at the Mac's Wi-Fi
address, which `./scripts/run-local.sh` prints. Emails are written to
`.local/logs/api.log`.

## Stack (spec 2A.2)

The same technology, at the same versions, as The Mighty Queue (2A.1).

| Layer | Technology |
| --- | --- |
| Backend | Go 1.25, Gin 1.12, pgx 5 with the queries written in the Go code (no sqlc) |
| Database | PostgreSQL 16; migrations applied by the server at startup with golang-migrate |
| Web client | React 18, TypeScript 5.6, Vite 6, React Query 5, React Router 6, on Node.js 20 |
| Styling | The design system's own stylesheet, the `pe-` classes and tokens. No Bootstrap (2A.4) |
| Hosting | Ubuntu 22.04, no containers anywhere; the Go binary under systemd; nginx; Let's Encrypt |
| Email | SMTP through Rackspace; written to the server log when no SMTP server is configured |
| Tests | Go's testing package; Vitest, React Testing Library and MSW |

## Layout

```
backend/
  cmd/server/          entry point
  internal/config/     settings read from the environment
  internal/database/   connection pool, migration runner
  internal/email/      Sender interface; SMTP and log adapters
  internal/storage/    Store interface; local disk adapter
  internal/server/     Gin router and handlers
  migrations/          numbered SQL pairs, applied at startup
frontend/
  src/styles/          tokens.css and components.css are the design system's; base.css is page-level
  src/api/             calls to the API
  src/pages/           one component per screen
  src/test/            Vitest setup, MSW, tests
deploy/                systemd unit, nginx site template, nginx snippets
scripts/               setup, run, test, check, deploy
docs/                  specification, design system, wireframes, roadmap
```

## Commands

```sh
./scripts/setup-local-db.sh   # once: role, database, backend/.env
./scripts/run-local.sh        # API on 8082; web client at http://phoenixevent.localhost:5174 (general address) and http://<org>.phoenixevent.localhost:5174
./scripts/check.sh            # every automated check; run before committing
./scripts/test.sh             # backend tests only, against phoenixevent_test
./scripts/deploy.sh root@droplet
```

## Rules

- **Tests never touch the development database.** Use `./scripts/test.sh`. Never
  source `backend/.env` and run `go test` in the same shell. The tests refuse
  any database whose name does not end in `_test`.
- **Before anything that can destroy a database's contents, take a backup and
  ask the owner.** Both, in that order.
- **The owner tests a change in local development before it is deployed.**
  Committing and pushing to GitHub is not a deploy. Anything that changes the
  droplet waits for the owner's word.
- **The droplet is shared** with The Mighty Queue and tomsherlock.space during
  testing (2A.4). PhoenixEvent only adds to it: port 8082, its own database, its
  own service, its own nginx sites. Never alter or restart what is already there.
- **The organization comes from the server-side session**, never from an
  identifier the client supplies (sections 4 and 4A). Every stored record carries
  its organization, and every query filters by it.
- **Approval, cancellation, appointment and the like are explicit actions**, each
  in one transaction with an audit record, never free field updates (section 2).
- Handlers stay thin: read the request, call a domain service, write JSON.
- A schema change is a new numbered pair in `backend/migrations/`. A migration
  that has been deployed is never edited.
- Secrets live in `.env` files, which are never committed.
- The design system's two stylesheets are copies of the files in
  `docs/design-system/` and `docs/wireframes/source/theme.css`. Change the design
  in the claude.ai project, then copy the new files in; do not edit them here.
