# Handoff: open questions and work in hand

Read this at the start of a session. Remove an item only once the owner has
answered it and the answer is recorded in the specification.

## Work in hand (November 10, 2026)

- **Phase 2 exit, built and not committed.** `backend/internal/server/isolation_test.go`
  tests that one organization's token, sign-in cookie and email links are
  refused at another organization's address. It found that a logout sent to
  one organization's address could end another organization's session; logout
  now ends only a session of the organization at its address
  (`accounts.Logout` takes the organization). All backend tests pass. Waiting
  for the owner to test logging out and back in on the Mac, then his word to
  commit and push.
- **Phase 3, O3 Policies and S1 Create event, built and not committed
  (November 8, 2026).** Migration 000004 (organization_policies, events);
  `accounts/policies.go`, `accounts/events.go`; routes GET/PUT /policies,
  GET/POST /events; pages O3 (`/policies`), S1 (`/events/new`), events on O1.
  Follows the owner's rulings of November 6 and 8, 2026, drafted as spec
  v1.10 (16B.6, 4A.11 Approval, 7B.6), which waits to be put in the claude.ai
  project. All checks pass. Waiting for the owner's test on the Mac, then his
  word to commit. S1 has no phone drawing; it uses the shared phone layout.

- **Phase 3, edit and delete events, and forming the event board, built and
  not committed (November 9, 2026).** Migration 000005 (event_board);
  `accounts/board.go`; Event board page (S7, S8) at `/events/:id/board` with
  Appoint board member (D1); Edit event and Delete event. Editing follows the
  owner's ruling: Organization Administrator, Event Administrator, Event
  Chair (the chair as soon as appointed). Two permissions are suggestions
  waiting for the owner: Delete event held by those who may edit; Remove a
  board member by Organization Leadership while a board-first board forms.
  Replace chair (D2) is not built. All checks pass; waiting for his test.
- **To test forming a board on the Mac.** admin@example.org holds only
  Organization Administrator. Forming a board belongs to Organization
  Leadership, so first give that account Organization Leadership (Organization
  officers (O8) → Change roles). Only confirmed accounts can be appointed:
  Administrator and Ramruva; January (JollyRoger@gmail.com) is unconfirmed.
  The dev database is at migration 5 (clean) and holds one event, Turkey
  Triathlon.
- **Event board message (November 10, 2026, owner's request).** Someone
  without Organization Leadership sees "You need Organization Leadership to
  appoint the chair" (or the board members), linking to Organization officers.
  Flow chart of the four chair-first steps: https://claude.ai/artifact/WfKj12tHoUCtfkT5h3JZHk
- **Commit, when the owner gives the word.** Everything above is uncommitted:
  the Phase 2 exit fix, O3 and S1, edit and delete, the event board. Each
  commit message ends with "Architect: Tom Sherlock
  <transcent.tech@gmail.com>", then the Co-Authored-By line.

## Spec v1.10 draft

`docs/drafts/phoenixevent-specification-v1.10-draft.md`, built from v1.09,
holds every ruling below. The owner puts it in the claude.ai project "Phoenix
Event"; the version that comes back is added to `docs/specification/`, and
then this draft is deleted. The draft is never itself copied into
`docs/specification/`.

## Decisions log, November 6 to 9, 2026

Rulings by the owner, newest last. A later ruling on the same matter replaces
an earlier one.

- **Nov 6.** First-release questions 1, 3, 4, 5 accepted as recommended: Create
  event (S1) asks board first or chair first per event; visibility shown as one
  "Private" line; after Create event go to Organization home (O1); name and
  board capacity (at least 1) required, the rest optional.
- **Nov 6.** Policies (O3) shows all four sections; Waiting lists and Lowering
  capacity shown, marked as a later release.
- **Nov 6.** 4A.11 Approval rewritten in his words, marked Confirmed with the
  date: approval is required when O3's "Approve each registration before the
  person chooses stations" is selected; approved by the Event Board Chair,
  Event Board Members or Event Administrator. The 7B.6 entry "Decide whether
  an accepted invitation's registration needs approval" removed.
- **Nov 8.** "Who participates" on O3 offers subscribers, registrants or both
  (the October 16 rulings replace 16B.2's "registrants only"). Standing rule:
  each new decision on an earlier issue takes priority.
- **Nov 9.** Editing an event: Organization Administrator, Event Administrator
  and Event Chair hold the permission by default. The Event Chair does not
  wait for the board to exist.
- **Nov 9.** Build Forming the board (S7, S8) and Appoint (D1) now.
- **Nov 9.** Coding design rules for every project (in
  `~/.claude/coding-rules.md`, shown on the Projects Dashboard): CRUD, every
  object agents can create can be read, edited, updated and deleted, each with
  its permission; name a screen by its descriptive name, its code alongside
  ("Create event (S1)"), never the code alone.
- **Nov 9.** Claude does not create business rules; it may suggest them. A
  limit on who may act, even a temporary one, is a business rule.

## Ruled

- 2026-11-10: Delete event is held by those who may edit the event (approved).
- 2026-11-11: 5D.3 amended: a seat holder is replaced only upon resignation,
  termination, incapacitating illness or death.
- 2026-11-11: termination and removal before an Event Chair exists
  (spec v1.10 draft 5D.3). Replaces my suggestion that Organization Leadership
  removes a board member alone while the board forms. Built 2026-11-11 on the
  Event board page (migration 000006, Removals card, Start removal dialog);
  checks green, uncommitted, awaiting his test. Dev DB backed up first.
  Then Reason (optional) for Cause (000007), the seated Event Administrator
  votes as an Event Board Member, votes change until Close voting (000008).
- Abstentions recorded at Close voting; Deputy Board Chair (7B.8, migration
  000009): elected by the board, recorded by the Event Chair (Organization
  Leadership before a chair); closes voting on the Event Chair's removal.
- 2026-11-18: Appoint board member (D1) lists all three roles (Event Chair,
  Board Member, Event Administrator); those the signed-in person may not appoint
  are greyed out with the reason (spec v1.10 draft 16B.6). Built, 45 frontend
  tests green, uncommitted, awaiting his test.
- 2026-11-18: build Replace chair (D2) now. Ruled the same day (spec v1.10
  draft 5D.3): new chair offered first to the Board Members, then the event's
  participants, then the organization at large, each with right of refusal; D2
  records why (resigned, illness, death); a chair who resigns stays a Board
  Member. Then: the offer is made outside PhoenixEvent and Organization
  Leadership records the answers on D2; Leadership chooses when several
  accept; a full board is the Event Board's decision, size unchanged by default.
  BUILT: Replace chair button on the Event board's Chair card, D2 dialog,
  POST /events/:id/board/replace-chair (ReplaceChair in board.go), no
  migration. All checks green, uncommitted, awaiting his test. Ruled the same
  day: Replace chair (D2) refuses while a removal of the Event Chair is open,
  until it is finished. REVISIT on Friday, November 20, 2026 (his date): this
  refusal, and with it how Replace chair (D2) works: answers recorded and the
  choice among those who accepted; the organization at large once every Board
  Member declines (no event participants yet); a resigning chair stays a Board
  Member, illness or death leaves the board; no seat added to a full board.
  2026-11-18: his test of Replace chair (D2) and of Appoint board member (D1)
  is delayed until after the Friday review, since the workflow may change.

- 2026-11-18: Organization home (O1) links shown as a menu strip, darker than
  the page (his request). `.pe-links` in base.css; checks green, uncommitted,
  awaiting his look. The O1 wireframe still shows plain links.

- 2026-11-18: everything above marked uncommitted was committed and pushed
  as a backup at his word (de2f099); still awaiting his test.

## Suggestions waiting for the owner's ruling

Built as suggestions, easy to change; not rules until he rules.
- (none)

## Not built yet

- Replace chair (D2) for a Station Chair (stations not built yet).
- Appointment making the person a participant (5D): waits for registration.
- A phone drawing of Create event (S1); it uses the shared phone layout.

## Five questions before building O3 and S1 (answered; awaiting spec v1.10)

Answered by the owner on November 6 and 8, 2026: S1 asks board first or chair
first; O3 shows all four sections, the later two marked, with "Who
participates" offering all three choices; S1 shows private as one line;
Create event leads to O1 until S7 and S8 are built; name and board capacity
(at least 1) required. Also the 4A.11 Approval paragraph rewritten and the
7B.6 entry "Decide whether an accepted invitation's registration needs
approval" removed. Drafted as spec v1.10 (see "Spec v1.10 draft"); remove this item once v1.10
is in `docs/specification/`.

## Other open items

- **Logging in as Ramruva for step 3.** The owner said testing step 3 required
  logging in as ramruva.kshatriaya; no earlier message said so. Asked where he
  saw it; no answer yet. Step 3 was tested and pushed (a6b64a3) since.
- **O8 shows roles set to Private**, for the time being, by the owner's ruling
  of November 1, 2026 (spec v1.08). Revisit after review.
