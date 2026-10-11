# PhoenixEvent specification addendum 0.10

## Event creation governance and wireframe handoff

Version 0.10 addendum, October 5, 2026 (America/New_York). It extends the requirements baseline, version 0.1 of October 2, 2026, which stays as written. Where the two differ, this addendum governs the topics it covers.

Audience: Claude Code and anyone implementing the new codebase. The wireframes show what people see. This document records what the wireframes cannot show: business rules, data, and logic.

Each item carries one of three labels.

- **Confirmed:** stated by the project owner.
- **Proposed:** an implementation choice awaiting the owner's approval. Treat it as a starting design and keep it easy to change.
- **Open:** undecided. Build nothing that depends on one answer.

## 1. Sources

- Requirements baseline 0.1: "Organization Event Collaboration Application, specification and implementation roadmap".
- Design system: https://claude.ai/artifact/LiN9xyts9t51qw9DCKk5mM (a copy of its brand book and tokens is in this project as `claude/phoenixevent-design-system.md`).
- Wireframes: https://claude.ai/artifact/VhSCCfeaJ79GLSwcc7eDR1

The wireframes hold 50 screens on ten pages.

| Page | Screens | Ids |
| --- | --- | --- |
| Registration | 6 | 1 to 6 |
| Accounts | 7 | A1 to A7 |
| Organization | 4 | O1 to O4 |
| Event setup | 6 | S1 to S6 |
| Communication | 5 | C1 to C5 |
| Billing | 3 | B1 to B3 |
| Chat | 2 | H1, H2 |
| States | 5 | E1 to E5 |
| Dialogs | 7 | D1 to D7 |
| Governance | 5 | G1 to G5 |

## 2. What belongs where

| Layer | What it is | Where it is recorded | Who decides |
| --- | --- | --- | --- |
| Wireframe | Screens, their order, the components on them, the wording | The wireframes canvas | Project owner, with design |
| Business rules | What each organization is allowed to configure, and the rules that hold for every organization | This specification | Project owner |
| Logic and code | The data model, the permission checks, the state changes, the transactions that enforce the rules | The codebase, built from this specification | Implementers |

A wireframe shows one organization on one day. The same screen serves every organization because the rules behind it are data, set per organization, and the code reads that data. A button such as "Create event" is wireframe. Who sees it, what it is called, and what happens after it is pressed are business rules and code.

Bracketed text on a wireframe, such as `[Password rules, to be decided]`, marks an Open item. Implement the surrounding screen and leave the bracketed part configurable or stubbed.

## 3. Event creation governance

### 3.1 Requirement (Confirmed)

Organizations differ in who may bring an event into existence, and the application must support each organization's own policy. The owner's examples:

- A corporation works top down. The CEO creates an event, or the board of directors does, or a committee drawn from the board, or a research committee of executives, employees and perhaps one director, or the chair of the event board.
- A volunteer organization, community organization or charity works by proposal. Employees, executives or directors propose an event, and the board of directors, the CEO, the C-suite, or a majority of employees approves it.

For-profit and non-profit organizations are major variants of this requirement.

### 3.2 Terms and model

**Terms the owner has approved for now. They may change. (Confirmed for now)**

- **Subscriber:** a person who pays for access.
- **Registrant:** a person signed up for an event.

**Terms from the baseline, used here as the baseline uses them.**

- **Organization:** a PTA, community committee, bank, business or similar, which owns its members, events and policies.
- **Member:** a person who belongs to an organization.
- **Event:** a named activity owned by one organization, such as Halloween Bash.
- **Station:** a named part of an event, such as Face Painting.
- **Chair:** the person who heads an event board or a station.
- **Event board:** the group that governs one event.

**Terms this addendum adds. (Proposed)**

- **Body:** a named group of members inside one organization, such as the Board of Directors, the C-suite, an Events Committee, a Research Committee, or All Employees. Each organization names its own bodies and decides who is in each. A body may have a chair. "Body" is a working word and the owner may rename it.
- **Proposal:** a member's request that an event be created, waiting for approval.
- **Proposer:** the member who makes a proposal.
- **Approver:** a member who has a say in whether a proposal goes ahead.
- **Event creation rule:** one statement of an organization's policy on who may create events. Example: "Any employee may propose an event, and the Board of Directors approves by majority."

**The model. (Proposed)**

Every organization writes its policy as one or more event creation rules. Each rule answers five questions.

| Question | Possible answers |
| --- | --- |
| 1. Who may start an event? | People holding a named role, the members of a named body, a named member, or every member |
| 2. Does the event exist at once, or does it need approval? | **Create directly:** the event exists at once. **Propose:** the event exists after approval. |
| 3. Who approves? (when the answer to 2 is Propose) | People holding a named role, the members of a named body, or a named member |
| 4. How do the approvers decide? (when the answer to 2 is Propose) | **Any one approver** says yes. **All approvers** say yes. **A share of the approvers** says yes, such as more than half. A rule may also set a quorum, which is a minimum number of approvers who must have voted. |
| 5. Who chairs the new event's board? | The person who started the event, a person the approvers choose, or a person appointed later |

An organization may hold several rules at once, and a member may use any rule that names them. Example for a corporation with two rules:

- "The CEO may create an event directly."
- "Any employee may propose an event, and the Board of Directors approves by majority."

Under these two rules the CEO creates an event with no approval, and an employee's event exists only after the board approves it.

For-profit and non-profit organizations use the same five questions with different answers. The application treats them the same way and reads each organization's own rules.

### 3.3 The owner's examples as rules (Proposed)

| Example | Who may start | Create directly or propose | Who approves | How they decide |
| --- | --- | --- | --- | --- |
| The CEO creates an event | The CEO | Create directly | Nobody | No decision needed |
| The board of directors creates an event | Members of the Board of Directors | Propose | The Board of Directors | More than half of the board |
| A board committee creates an event | Members of the Events Committee | Create directly | Nobody | No decision needed |
| A mixed research committee creates an event | Members of the Research Committee | Create directly | Nobody | No decision needed |
| The chair of an event board creates an event | An Event Chair | Create directly | Nobody | No decision needed |
| Staff propose, the board approves | All Employees | Propose | The Board of Directors | More than half of the board |
| Staff propose, the CEO approves | All Employees | Propose | The CEO | Any one approver |
| Staff propose, the C-suite approves | All Employees | Propose | The C-suite | Open: any one approver, or a share |
| Staff propose, a majority of employees approves | All Employees | Propose | All Employees | More than half of all employees |

In the second example a director proposes and the whole board votes. Whether one director may instead create the event alone, acting for the board, is Open and is answered rule by rule.

A new organization starts from one of two ready-made sets of rules and edits them afterward:

- **Direct:** organization administrators create events directly.
- **Propose and approve:** every member may propose, and one named body approves.

### 3.4 The life of a proposal (Proposed)

A proposal is in one of five states: Draft, Submitted, Approved, Declined, or Withdrawn.

**Writing a draft.** A member may start a draft only when their organization has a rule, currently switched on, that lets them propose events. Example: the organization's rule says "any employee may propose an event, and the Board of Directors approves". An employee may start a draft. A member who is not an employee, and whom no other rule names, may not: the "Propose event" button is absent for them and the server refuses the action.

The draft holds the proposed event's details: its name, date, location, description, who can see it, and the size of its event board. These are the fields of the Create event screen (S1). Only the proposer can see or change a draft, and the proposer may save it as many times as they like.

**Submitting.** When the proposer submits, the application does seven things in order. If any one of them fails, the proposal stays a Draft and nothing else changes.

1. **It checks the event details.** The event has a name, and the event board has room for at least 1 person.
2. **It finds the rule that applies.** It looks at the organization's switched-on rules that let this person propose today. With one such rule, that rule applies. With several, the proposer picks one on the Propose event screen (G3), which shows each rule by who would approve. With none, the application refuses.
3. **It lists the approvers by name.** A role becomes everyone who holds that role today. A body becomes everyone in that body today. A named member is that member. Each person appears once. The number of people on the list is called N. With nobody on the list, the application refuses.
4. **It works out how many approvals are needed.** This number is called R and comes from the table below.
5. **It records the result on the proposal:** which rule applied, how the approvers decide, any quorum, who will chair, the list of approvers, N, R, and the time.
6. **It marks the proposal Submitted.** From now on the event details are locked.
7. **It emails each approver** a link to the Proposal review screen (G4).

Example: an employee submits under the rule "any employee may propose, and the Board of Directors approves by majority". The board has 7 directors today. The approvers are those 7 people, N is 7, and R is 4.

**Why the list is recorded.** The approvers, N and R written down at step 5 decide the proposal. If the organization later edits the rule, gives the role to someone new, or adds a director, this proposal keeps the approvers and numbers it was submitted with. The one exception is an approver who is removed, covered in 3.8.

**A proposer who is also an approver.** A director who proposes to their own board stays on the list of approvers and votes like every other director.

**Deciding.** Each approver either approves or declines, and may add a comment. An approver may change their answer while the proposal is Submitted.

The outcome is counted from five numbers. **N** is the number of approvers on the recorded list. **A** is the number who currently approve. **D** is the number who currently decline. **U** is the number yet to answer, so N = A + D + U. **R** is the number of approvals needed.

- The proposal becomes Approved the moment A reaches R.
- The proposal becomes Declined the moment A + U falls below R. At that point R is out of reach even if every remaining approver approved.
- Either outcome is final. Answers arriving after it are refused.

| How the approvers decide | R, approvals needed | Approved when | Declined when |
| --- | --- | --- | --- |
| Any one approver | 1 | 1 approver approves | Every approver has declined (D = N) |
| All approvers | N | Every approver has approved (A = N) | 1 approver declines |
| A share, more than P percent | The smallest whole number greater than P percent of N | A = R | D = N - R + 1 |
| A share, at least P percent | The smallest whole number equal to or greater than P percent of N | A = R | D = N - R + 1 |

Worked examples for "more than half":

| Approvers (N) | R | Approved at | Declined at | A tie |
| --- | --- | --- | --- | --- |
| A board of 7 | 4 | The 4th approval | The 4th decline | Cannot occur |
| A board of 6 | 4 | The 4th approval | The 3rd decline | 3 to 3 is declined |
| All Employees, 40 people | 21 | The 21st approval | The 20th decline | 20 to 20 is declined |

- A share is measured against all N approvers. An approver who never answers counts toward neither side, so a proposal can stay Submitted until it is withdrawn or a deadline closes it. Deadlines are Open in 3.10.
- A quorum Q, when a rule sets one, adds a second condition for approval: A + D must reach Q. With a share measured against all N approvers, a quorum changes the outcome only when Q is greater than R.

**Withdrawing.** The proposer may withdraw a proposal that is a Draft or Submitted. A Withdrawn proposal creates no event and accepts no further answers.

**Approval creates the event.** The moment a proposal becomes Approved, the application creates the event from the proposal's details and links the two. Both happen together: a proposal is never Approved without its event, and one proposal never produces two events.

**Choices in this section with Open alternatives.**

- At step 2, the organization could rank its rules, so that the highest-ranked rule applies and the proposer makes no choice.
- The organization could bar a proposer from voting on their own proposal, which lowers N by 1.
- A share could be measured against the answers given (A + D) instead of all N approvers. That decides faster, and it needs a quorum and a deadline to be safe.
- For "any one approver", the first answer of either kind could settle the proposal.

### 3.5 What the application keeps (Proposed)

- **Bodies:** each body's name, its members, and its chair if it has one.
- **Event creation rules:** each rule's five answers from 3.2, and whether the rule is switched on.
- **Proposals:** who proposed, the proposed event's details, which rule applied, the recorded approvers with N and R, the state, the times, and the event that resulted.
- **Decisions:** which approver answered, whether they approved or declined, their comment, and when. Each approver has one current answer per proposal.

Each body, rule, proposal and decision belongs to exactly one organization. A body, rule, proposal or decision in one organization is invisible to every other organization, in line with section 4 of the baseline.

Appendix A gives the storage names for implementers.

### 3.6 Who may do what (Proposed)

Five permissions cover this section.

1. Create an event directly.
2. Propose an event.
3. Decide a proposal.
4. Manage the organization's event creation rules.
5. Manage the organization's bodies.

A member holds the first three through the organization's rules: a rule that names them as someone who may start an event gives permission 1 or 2, and a place on a proposal's recorded list of approvers gives permission 3. Permissions 4 and 5 belong to organization administration.

Being a subscriber gives none of these permissions, and none of them takes the place of paid access where a feature needs it.

Each permission has a fixed name in the code that stays the same when an organization renames a role title. Appendix A lists the names.

### 3.7 Actions (Proposed)

Each of the following is one deliberate, named action in the application, in the manner section 2 of the baseline requires for approval and chair appointment.

- Create an event directly.
- Save a proposal draft. Submit a proposal. Withdraw a proposal.
- Record an approver's answer.
- Create the event from an approved proposal. This happens as part of the answer that completes the approval.
- Save an event creation rule. Retire an event creation rule.
- Create a body. Add a member to a body. Remove a member from a body. Set a body's chair.

Every action checks that the person is allowed to do it in the organization they are logged in to, happens completely or leaves everything as it was, and is written to the audit record. Appendix A lists the action names used in the code.

### 3.8 Rules that hold for every organization (Proposed)

- An event comes into existence in two ways only: a member creates it directly, or an approved proposal creates it.
- A member may use a rule only while the rule still names them at the moment they act. Example: an employee who leaves the organization's staff can no longer start a draft under "any employee may propose".
- A Submitted proposal is decided by the approvers and numbers recorded when it was submitted, even when the organization later edits its rules.
- When an approver is removed from the organization or from the approving body, their answer stops counting. Whether N and R are then worked out again for the smaller group is Open.
- Two approvers answering at the same moment produce one outcome and at most one event.
- An organization always has at least one rule switched on, so that events can be created.

### 3.9 Effect on the wireframes

The Create event screen (S1) shows the case where a member creates an event directly. Five further screens, drawn on the Governance page of the wireframes, cover rules, bodies and proposals:

| Id | Screen | Purpose |
| --- | --- | --- |
| G1 | Event creation rules | List, add and edit the organization's rules; start from a ready-made set |
| G2 | Bodies | Create a body, choose its members and its chair |
| G3 | Propose event | The S1 screen with "Submit proposal" and a line naming who approves |
| G4 | Proposal review | Approvers read the proposal, see the answers so far, and approve or decline |
| G5 | My proposals | The proposer follows each proposal's state and may withdraw it |

The button on O1 follows the organization's rules. A member whom a rule lets create events directly sees "Create event". A member whom the rules let only propose sees "Propose event". A member whom no rule names sees no button. The O1 wireframe draws both buttons side by side with a note saying a member sees one of them, and links to screens G1, G2, G4 and G5.

### 3.10 Open decisions

- Whether a body that starts an event needs its own vote first, or one member may act for it.
- How a small body such as a C-suite decides: any one approver, or a share.
- Deadlines on proposals, and what happens when one passes with the proposal still Submitted.
- Whether approvers may edit a proposal, or only approve or decline it as written.
- Whether a Declined proposal may be revised and submitted again, or starts again as a new proposal.
- Whether the same approach later governs other decisions, such as approving registrants or changing waiting list policy. The design above keeps that possible.
- Whether a body may also serve as an event board, or the event board stays its own group as in the baseline. This addendum keeps them separate.
- The word "body" itself, which the owner may replace with a term the organizations use.
- The definitions of subscriber and registrant in 3.2, which the owner approved for now and may change.

## 4. Rules the wireframes imply, for the code to enforce

These are already visible on screens. Each is a business rule or a logic requirement, listed so that none is read as decoration.

| Screens | Rule | Label |
| --- | --- | --- |
| A1, A2, A7, E3 | A session belongs to one organization. Changing organization means logging out and logging in. The server binds the session to the organization. | Confirmed |
| A2, E1 | Login has two fields. A failed login returns one message for every cause. | Confirmed fields, Proposed message |
| A3 | A blank username defaults to the email address. | Confirmed |
| A5 | The reset request returns the same response whether or not an account matches. | Proposed |
| A4, A6, E2 | Confirmation and reset links work once and expire. | Proposed |
| 1 to 5, S5, D3 | Capacity includes the chair. Assignments cannot pass capacity, including under simultaneous requests. | Confirmed |
| 3, 4, 6, O3, D3 | A person joins at most the configured number of stations per event. | Confirmed rule, Proposed representation |
| 2, S4 | Registration follows the organization's approval policy. | Confirmed |
| O3, S2 | Waiting list governance: an organization prohibition binds the board; an organization request may be turned off by the board; with no determination the board chooses. Both decisions are stored separately. | Confirmed |
| 5, S6 | Waiting list order and promotion. | Open |
| O4, S2 | Labels and role titles are display text. Event titles override organization titles. Permissions keep stable names. | Confirmed |
| O2, B2, D5 | Roles and paid access are separate records and separate checks. | Confirmed |
| S1, C1 | Event visibility modes: public, unlisted, private. Private information stays out of public responses. | Proposed modes, Confirmed protection |
| C2, C3 | Announcements at three levels, posted by leadership and the relevant chairs. | Confirmed |
| C3 | Attachment types: jpg, jpeg, png, md, html, pdf, doc, docx. Video formats and size limits. | Confirmed types, Open limits |
| C4 | Leaving the application shows the destination and offers Continue and Cancel. | Confirmed |
| C5, D7 | Three built-in themes. An added theme passes the contrast check before it can be chosen. | Confirmed themes, Proposed check |
| B1 to B3 | Organization seats and individual payment. | Confirmed arrangements, Open rules |
| H1, H2 | Chat at four scopes, text only, direct messages within the current organization, gated by plan. | Confirmed |
| E4 | Form errors: a summary linking to each field, the field marked, a line saying how to fix it. | Proposed |
| D4 | Every removing action asks for confirmation and names what is removed. | Proposed |

## 5. Roadmap changes

These add to the phases of the baseline roadmap. No durations are assigned.

**Visual roadmap.** The roadmap is displayed as a timeline, a dependency diagram, progress charts and phase descriptions at https://claude.ai/artifact/KVzDxND3nxjtcqe1NDrYHz. That page and this section describe the same roadmap. The timeline shows order only, because no dates are set.

**The phases in order, with status as of October 5, 2026.**

| Phase | Name | Depends on | Status |
| --- | --- | --- | --- |
| Design | Design system | Nothing | Complete |
| Design | Wireframes, 45 screens | Design system | Complete |
| Design | Governance screens G1 to G5 | Design system | Complete, drawn from the Proposed model in section 3 |
| 0 | Resolve foundational decisions | Nothing | In progress |
| 1 | Repository and development foundation | Phase 0 | Yet to start |
| 2 | Accounts, organization context, and RBAC | Phase 1 | Yet to start |
| 3 | Events, boards, stations, and registration | Phase 2 and screens G1 to G5 | Yet to start |
| 4 | Public pages, announcements, media, and themes | Phase 3 | Yet to start |
| 5 | Subscriptions and individual payment | Phase 2 | Yet to start |
| 6 | Optional chat | Phases 3 and 5 | Yet to start |
| 7 | Release validation and direct deployment | Phases 4 and 6 | Yet to start |

The baseline lists Phases 0 to 7 in one order. The "Depends on" column is a reading of that order and is Proposed.

**What this addendum adds to the phases.**

- **Phase 0, resolve foundational decisions.** Add: approve or revise the governance model in section 3, and settle the Open items in 3.10 that affect the schema (body votes, decision rule for small bodies, relation of bodies to the event board).
- **Phase 2, accounts, organization context, and RBAC.** Add: bodies and body membership; the five permissions in 3.6; rule matching as part of scoped authorization.
- **Phase 3, events, boards, stations, and registration.** Add, before event CRUD: event creation rules, presets, proposals, decisions, and the two event creation actions. Event CRUD then builds on events that already exist.
- **Wireframes.** Screens G1 to G5 are drawn. They follow the Proposed model in section 3 and change if that model changes. The three views for joining a station by approval, listed in 8.3, are yet to be drawn.

## 6. Acceptance criteria added

- An organization configured with the Direct preset lets an organization administrator create an event at once, and offers other members no create or propose action.
- An organization configured with Propose and approve lets a member submit a proposal, and the event exists only after the named body's decision rule is met.
- Each example in section 3.3 can be configured through rules alone, with no code change.
- A proposal submitted under one rule is decided under that rule after the organization edits its rules.
- Two approvers deciding at the same moment produce exactly one event.
- A member of one organization cannot see or decide a proposal in another organization, including by changing a client-supplied identifier.
- The Organization home button reads "Create event", "Propose event", or is absent, according to the rules the person matches.

## 7. Status

This addendum is documentation only. No repository, schema, or code exists for the new application. The governance model in section 3 is Proposed and awaits the owner's review. Changes made during implementation update this document under a new version number.

## 8. Joining a station

Recorded October 5, 2026, from the owner's statement. This is a business rule. It is recorded here for development in Claude chat or Claude Code, and the wireframes do not yet show all of it.

### 8.1 The rule (Confirmed)

1. **Who may ask.** A user may submit a request to join a specific station. The user may be a registrant, a subscriber, a member or a worker.
2. **Joining at once.** Joining a station may mean that the user becomes a member of the station upon submitting the request, if the station has open spots.
3. **Joining with approval.** Joining a station may instead require approval. The approver is the station's chair, station lead or station manager.
4. **When that position is open.** If the station has nobody in that position yet, the responsibility for approving falls to the Event Board Chair.

Example: Haunted Hallway has 9 of 10 places taken and Dana Reyes is its Station Chair. If Haunted Hallway lets people join at once, a user who asks to join takes the 10th place immediately. If Haunted Hallway requires approval, the request goes to Dana Reyes. If Haunted Hallway had no chair yet, the request would go to the Event Board Chair of Halloween Bash.

### 8.2 How this fits what is already recorded

- **Capacity.** Baseline section 5 says a station's capacity includes its chair and that assignments cannot pass capacity. Joining at once and joining by approval both stay inside that limit.
- **Stations per person.** Baseline section 5 says the organization sets how many stations one person may join in an event. A request to join counts against that limit in a way that is still Open (see 8.4).
- **Waiting lists.** Baseline section 5 covers what a full station may offer. This section covers a station with open spots.
- **Role titles.** Baseline section 6 says role titles are display text that an organization or event may rename. "Station chair", "station lead" and "station manager" are read here as three titles for the one position that heads a station. This reading is Proposed (see 8.4).

### 8.3 Effect on the wireframes

The Station screen (4) shows only joining at once: one "Join station" button. Three things are yet to be drawn:

- The same screen for a station that requires approval, with a button that sends a request.
- The requester's view while the request waits, and after it is approved or declined.
- The approver's list of requests for a station, for the station's head and for the Event Board Chair when the station has no head.

### 8.4 Open decisions

- **Who chooses the mode.** Whether "join at once" or "join with approval" is set by the organization, by the event board, or station by station.
- **Worker.** What a worker is, and how a worker differs from a member.
- **Chair, lead, manager.** Whether these are three titles for one position, as read in 8.2, or separate positions that a station may hold at the same time. If separate, which of them approves.
- **Registering for the event first.** Whether a user must be a registrant of the event before asking to join one of its stations. The current wireframes assume yes: a person registers for the event, is approved, and then chooses stations. The rule in 8.1 names registrants as one kind of user among four, which suggests the answer may be no.
- **Holding a spot.** Whether a request that awaits approval holds a spot, and what happens when the last spot is taken while a request waits.
- **A full station.** Whether a request to a full station becomes a waiting list entry where the waiting list is on.
- **The stations-per-person limit.** Whether a waiting request counts toward the limit, or only an approved one.
- **No Event Board Chair.** Who approves when the station has no head and the event board has no chair either.
- **After a decline.** What the user is told, and whether they may ask again.
- **Withdrawing.** Whether the user may withdraw a request that is waiting.

### 8.5 Roadmap

This rule belongs to Phase 3 (events, boards, stations, and registration). The open decisions in 8.4 that shape the data, namely who chooses the mode, the three titles, and registering first, belong to Phase 0.

## Appendix A. Names for implementers (Proposed)

This appendix is for Claude Code. It gives the names the code and the database use for the things section 3 describes in plain words. Every name here is Proposed.

**Stored records**

| Section 3 calls it | Stored as | Holds |
| --- | --- | --- |
| Body | `body` | organization id, name, optional chair |
| A member's place in a body | `body_membership` | body id, organization membership id |
| Event creation rule | `event_creation_rule` | organization id, who may start, direct or proposal, approvers, decision rule, threshold, quorum, chair on creation, active flag |
| Proposal | `event_proposal` | organization id, proposer, applied rule id, recorded approvers, N, R, a copy of the decision rule, threshold, quorum and chair on creation, proposed event fields, state, timestamps, resulting event id |
| Decision | `proposal_decision` | proposal id, approver, approve or decline, comment, timestamp; one current decision per approver |

Every record carries its organization id, and every query filters by the organization of the server-side session.

**Fixed values**

| Section 3 calls it | Value in code |
| --- | --- |
| Create directly | `direct` |
| Propose | `proposal` |
| Any one approver | `any_one` |
| All approvers | `all` |
| A share of the approvers | `share`, with a threshold percent and a comparison of "more than" or "at least" |
| Chair: the person who started the event | `initiator` |
| Chair: a person the approvers choose | `chosen_by_approvers` |
| Chair: appointed later | `appointed_later` |
| Draft, Submitted, Approved, Declined, Withdrawn | `draft`, `submitted`, `approved`, `declined`, `withdrawn` |

**Permission names**

| Permission in 3.6 | Name in code |
| --- | --- |
| Create an event directly | `event.create_direct` |
| Propose an event | `event.propose` |
| Decide a proposal | `event_proposal.decide` |
| Manage event creation rules | `event_creation_rule.manage` |
| Manage bodies | `body.manage` |

**Action names**

| Action in 3.7 | Name in code |
| --- | --- |
| Create an event directly | `CreateEventDirect` |
| Save a proposal draft | `SaveProposalDraft` |
| Submit a proposal | `SubmitProposal` |
| Withdraw a proposal | `WithdrawProposal` |
| Record an approver's answer | `RecordProposalDecision` |
| Create the event from an approved proposal | `CreateEventFromProposal` |
| Save or retire an event creation rule | `SaveEventCreationRule`, `RetireEventCreationRule` |
| Create a body, add or remove a member, set its chair | `CreateBody`, `AddBodyMember`, `RemoveBodyMember`, `SetBodyChair` |

Each action runs in one PostgreSQL transaction and writes an audit record. `SubmitProposal` performs the seven steps of 3.4 inside its transaction. `RecordProposalDecision` calls `CreateEventFromProposal` inside its own transaction when the answer brings A up to R.
