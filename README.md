# PhoenixEvent

People working an organization's event together, divided into stations.
Multiple organizations, each with its own address, accounts and data.

The specification is in `docs/specification/`; the highest version number is
current. `docs/README.md` describes the design system, wireframes and roadmap
beside it.

**Status:** roadmap Phase 1, the repository and development foundation, is
built. There are no application features yet.

## Local setup on macOS (no containers)

Needs Go 1.25, Node.js 20 and PostgreSQL 16 from Homebrew running on port 5433.

```sh
./scripts/setup-local-db.sh    # once: role, database, backend/.env
./scripts/run-local.sh         # API on 8082, web client on http://localhost:5174
```

The page at http://localhost:5174 shows **Ready** when the web client reaches
the API and the API reaches the database. The API applies its migrations each
time it starts.

## Checks

```sh
./scripts/check.sh             # format, vet, backend tests, web client types, tests and build
```

Backend tests that use a database run against `phoenixevent_test`, which
`scripts/test.sh` creates. They refuse any other database.

## Deployment

`docs/installation.md` covers the droplet: the one-time setup, the nginx site
and certificate for each address, and `./scripts/deploy.sh`.
