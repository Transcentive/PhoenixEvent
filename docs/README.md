# PhoenixEvent: handoff to the build

Handoff pack, version 1, October 10, 2026. It holds the design work done in the claude.ai project "Phoenix Event", copied into the repository so that the session building the application can read it. Nothing in this folder is application code, except the design system's stylesheet, which section 2A.4 of the specification makes PhoenixEvent's own.

## Where things are

| Path | What it is |
| --- | --- |
| `../CLAUDE.md` | Instructions for Claude on this project. Read automatically at the start of every session. |
| `specification/phoenixevent-specification-v0.78.md` | The specification. It governs everything else here. |
| `specification/phoenixevent-later-decisions-v0.2.md` | Decisions the owner holds for later. |
| `specification/source-documents/` | The requirements baseline 0.1 and addendum 0.10, which the specification merges (section 19). |
| `reference/The-Mighty-Queue-tech-stack-and-hosting-v2.md` | The Mighty Queue's stack and hosting, which section 2A aligns PhoenixEvent with. |
| `design-system/source/` | The design system: `tokens.json`, the component stylesheet `components/bundle.css` (the `pe-` classes), each component's README and preview, and the brand book in `README.md`. |
| `design-system/icons/`, `design-system/previews/` | The icons as SVG, and pictures of the components in the three themes. |
| `wireframes/screens/` | A picture of each of the 59 screens. `wireframes/screens-index.md` lists them by id and page. |
| `wireframes/source/` | The source of each screen, for exact wording and layout. `theme.css` holds the tokens as CSS variables, one block per theme; `bundle.css` loads after it. These files were drawn for the wireframe canvas and do not run on their own. |
| `roadmap/phoenixevent-roadmap.html` | The visual roadmap. Sections 16 to 16B of the specification hold the same content. |

## How to read the specification

- **Labels.** Confirmed means the owner stated it. Proposed means a design choice awaiting the owner's approval. Open means undecided: build nothing that depends on one answer.
- **Lettered sections govern.** Where a lettered section (such as 2A or 5D) and a numbered baseline section differ, the lettered section governs the topic it covers.
- **Bracketed text on a wireframe** marks an Open item. Build the screen around it and leave that part configurable or stubbed (section 20).
- **Section 21** lists the rules the wireframes imply, for the code to enforce.

## Phase 1: repository and development foundation

Section 16 of the specification, as section 2A.3 amends it.

- **Delivers:** a modular Go and React structure, PostgreSQL migrations with golang-migrate, database access with pgx (no sqlc), automated checks, configuration handling, first deployment notes, an email adapter and a file storage interface.
- **Done when:** a repeatable setup on a plain Linux host works, the API and the web client build, migrations run, and health checks pass.
- **Stack:** section 2A.2 (Go 1.25, Gin, pgx 5, golang-migrate; React 18, TypeScript, Vite 6; PostgreSQL 16; Ubuntu 22.04, nginx, Let's Encrypt, systemd; Node.js 20.20.2 per 2A.4). The design system's stylesheet replaces Bootstrap (2A.4).
- **Testing host:** section 2A.4. The existing droplet, a service on port 8082, a separate `phoenixevent` database in the PostgreSQL 16 already there.
- **Deployment:** section 2A.3. A script on the Mac builds and ships to the droplet, with a database dump before every deploy.

The first release, which Phases 2 and 3 build toward, is scoped in section 16B.

## Before Phase 1 starts

Section 16B.5 lists the owner's setup tasks: write access to the repository for the building session, DNS for the test addresses, the Rackspace SMTP account, and CLAUDE.md in the repository (done by this pack).

## When the specification changes

The specification is revised in the claude.ai project, and each revision is a new version with a new number. A new version is added to `specification/` beside the earlier ones. A version already here is never overwritten.
