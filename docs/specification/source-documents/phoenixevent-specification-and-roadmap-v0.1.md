# Organization Event Collaboration Application
## Specification and implementation roadmap

Version 0.1 — requirements baseline, October 2, 2026 (America/New_York).

This document consolidates the requirements agreed in the planning conversation. It describes a new application, not changes to the Drupal repositories. Confirmed requirements, proposed implementation choices, and unresolved decisions are distinguished below. No new application repository, production deployment, or complete application implementation is claimed by this document.

## 1. Purpose and scope

Enable people to work together in an organization’s event, divided into stations or departments. Organizations include PTAs, community committees, banks, and businesses. Each organization controls its memberships, events, governance, registration policies, communication, and subscription access.

Use Event_Registration and event_station as behavioral references. Build a new codebase with Go as the backend language. Legacy Drupal code is a reference rather than an implementation dependency; known legacy defects must not be copied into the new application. Data migration from Drupal is not yet specified.

The application must support multiple organizations, scoped RBAC, configurable terminology and role titles, capacity-controlled registration, optional waiting lists, paid features, announcements, attachments, optional chat, and themes.

## 2. Technology stack and hosting

The planning baseline is Go with Chi for the HTTP API; TypeScript, React, and Next.js for the frontend; PostgreSQL for persistence; pgx and sqlc for typed database access; and Caddy for HTTPS and reverse proxying. Next.js supports server-rendered public event pages for search visibility. Chi uses Go’s standard net/http conventions. Caddy simplifies HTTPS on a directly managed server.

Deployment must use direct Linux installation, without Docker or containers. The initial host is DigitalOcean; the design should allow a later move to AWS or another provider. Proposed deployment uses a Go executable and a Node.js/Next.js process managed by systemd, with Caddy installed directly. Managed PostgreSQL is preferred; local PostgreSQL remains an option. Exact versions, Linux distribution, database hosting, and infrastructure sizing remain open.

The Go application must use structs, methods, interfaces, and composition to model domain objects. CRUD operations must enforce business rules, permissions, and applicable subscription entitlements. Approval, cancellation, chair appointment, and waiting-list promotion should be explicit domain actions rather than unrestricted field updates.

## 3. Domain model

- User account: a person’s identity, credentials, email verification state, and profile.
- Organization: owns memberships, policies, events, titles, communication, and subscription settings.
- Organization membership: connects a user to an organization with organization-specific roles and paid-access status. A person may join multiple organizations.
- Event: a named activity owned by one organization, containing stations and governed by an event board. Example names include Halloween Bash and Valentine Dance.
- Event board: event governance body with a chair and configurable membership capacity. Capacity includes the event chair.
- Board membership: a person’s appointment to an event board and role within it.
- Station: a named part of an event with a chair and configurable member capacity. Capacity includes the station chair.
- Event registration: a person’s participation in an event, subject to organization registration policy.
- Station assignment: a registered person’s participation in a station, subject to event assignment and station capacity rules.
- Waiting-list entry: an optional request for a full station, with lifecycle and promotion rules to be defined.
- Role and permission assignment: grants capabilities at system, organization, event, or station scope.
- Subscription plan and organization subscription: define purchased organization-level capabilities and seat entitlements.
- Seat allocation and individual payment: track organization-funded or individually funded access under an organization’s subscription arrangement.
- Announcement: scoped communication with links and supported attachments/media.
- Attachment: a file with ownership, metadata, visibility, and access rules tied to its parent scope.
- Conversation, participant, and message: optional chat resources at organization, event, station, or direct-message scope.
- Theme: a color palette with optional font-family and layout settings.

Relationships must preserve ownership: stations belong to events; events belong to organizations; registrations and station assignments cannot cross organization boundaries. Proposed technical entities include authenticated sessions, one-time verification/reset tokens, audit records, email-delivery jobs, and payment-provider event records.

## 4. Organization isolation and authentication context

A user may belong to multiple organizations but may access only one organization at a time. The user must log out of the current organization and log into another to change organization context. There must be no silent organization switch that retains the prior organization session.

Every protected request, file request, search, notification feed, and live chat connection must be checked against the currently authenticated organization and scoped permissions. Logging out must invalidate the relevant session and close associated live connections. A user’s role in one organization must not grant authority in another.

Public event pages are an explicitly permitted exception to authenticated membership access: organizations control which event information is public. Private organization information must not be exposed through public pages, search metadata, caches, or public attachment URLs.

The exact meaning of “one organization at a time” across multiple browsers/devices, and how organization selection fits a two-field login screen, remain open. Proposed design binds the server-side session to an organization, rather than trusting an organization ID supplied by the client.

## 5. Governance, capacity, and registration

Each organization controls an event through its board, headed by a chair. Board membership capacity is configurable and includes that chair. Each station has a chair and configurable capacity that includes its chair. Appointment, replacement, and removal actions must preserve these limits.

Organization policy determines registration conditions, including any approval requirement. Organization configuration also determines whether a person may join multiple stations in one event. A proposed representation is maximum stations per participant per event, with a value of one for single-station participation, a higher limit, or unlimited; the default and available policy options need confirmation.

Full stations may offer an optional waiting list. The confirmed governance rule is:

- If the organization prohibits waiting lists before the event board is formed, the board cannot enable them.
- If the organization requests a waiting list before board formation, the board may disable it.
- If the organization makes no prior determination, the board may enable or disable waiting lists.

Record the prior organization determination and board decision separately. Policies after board formation, authority to change an earlier prohibition, list ordering, promotion, acceptance deadlines, and handling existing entries when disabling a list are unresolved.

Proposed implementation uses PostgreSQL transactions and constraints/locking to prevent concurrent assignments from exceeding capacity or creating duplicate registrations. Whether pending registrations reserve capacity, and whether board or station chairs must have event registrations, must be decided before final schema design.

## 6. Configurable vocabulary and titles

Default domain labels include event and station. Administrators must be able to substitute terms such as function, activity, festival, or department. Labels are distinct from individual names: Festival may be the label while Halloween Bash is the event name. Internal resource identifiers and permission names remain stable when labels change.

Terminology configuration scope remains undecided. Singular/plural forms and consistent use in forms, navigation, announcements, emails, and accessibility labels are proposed requirements for implementation review.

Role titles, such as Event Director, Event Chair, Event Executive, and Event Manager, are initialized at organization level and may be overridden at event level. If an event has no override, it inherits the organization title. Renaming a displayed role title must not alter RBAC permissions. Whether inherited titles update immediately after organization changes remains open.

## 7. RBAC, CRUD, and paid entitlements

Support role-based access control with system, organization, event, and station scopes. Organization leadership, organization administrators, event administrators/boards/chairs, and station chairs need separately definable permissions. Exact roles and the permission matrix remain to be finalized; examples in this specification do not grant automatic unrestricted access.

CRUD applies to domain resources, with relationship and lifecycle rules enforced. Deletion versus archival, retention, reassignment of chairs, and removal of resources that already have memberships, registrations, files, or payment history require explicit policy decisions.

Paid entitlements and RBAC are separate checks. A paid subscription does not itself grant administrative permission, and administrative permission does not automatically bypass a paid entitlement. Rules for overrides or complimentary access remain open.

Subscriptions are anchored at organization level and may include a set number of individual member seats. Another organization subscription arrangement permits members to pay individually while remaining members of that organization. Both organizations and individuals may purchase applicable access. Organization membership and paid-access state must be represented separately.

Unresolved billing rules include hybrid seat/individual-payment arrangements, the scope of an individual purchase across organizations, who counts against seat limits, plan prices, renewal/cancellation, expiration behavior, refunds, and the payment provider. The full set of paid features will be revisited. Chat is normally subscription- or upgrade-gated.

## 8. Accounts, login, and email

The application must have a configurable email facility for account email confirmation and password reset. Users may provide a username; if none is supplied, their email address becomes the username. The login screen has username and password fields. Passwords are hidden by default and can be revealed with an accessible Show/Hide button. Include a Forgot password? link.

Proposed security behavior uses expiring, single-use email-verification and password-reset links, password hashing, rate limiting, and generic reset responses that do not disclose account existence. Credentials and delivery-provider secrets must not be embedded in source or documentation. Email delivery should use configurable SMTP or a provider adapter so it is independent of the hosting platform.

Open decisions: whether verification is required before login or restricted access is permitted; username uniqueness and normalization; whether email login remains accepted after choosing a username; email-change behavior; password policy; token lifetimes; and email branding/delivery provider.

## 9. Public pages and discoverability

Public event pages must support strong search-engine visibility. Organizations must be able to hide events when they prefer. Proposed visibility modes are public/indexable, unlisted/noindex, and private/authenticated; the final vocabulary and controls need approval.

Server-rendered content, page titles/descriptions, canonical URLs, and sitemaps should reflect the effective visibility. Hiding an event must remove its private information and files from public responses. Search exclusion alone does not protect confidential content. Previously indexed content may remain in external search caches after visibility changes.

## 10. Optional chat

Chat must support organization conversations, event conversations, station conversations, and direct messages. It is optional and can be activated by a role with the required permission, such as a system administrator. Access is normally unlocked by a subscription payment or upgrade payment.

Direct messages are permitted only among people who belong to the currently logged-in organization. Membership shared elsewhere is insufficient for a message in the current organization context. Group access follows the corresponding organization/event/station membership and RBAC rules.

Chat must not support attachments. Activation scope, moderation, message retention/deletion, history after membership removal, notifications, and behavior when entitlement expires remain open. Proposed implementation uses authenticated live connections and persists messages in PostgreSQL; the transport and delivery guarantees remain implementation choices.

## 11. Attachments and announcements

Support still-image uploads in jpg, jpeg, and png, and documents in md, html, pdf, doc, and docx. Attachments are supported at organization, event, and station levels; chat attachments are excluded. Access follows the parent’s organization, scope, permissions, and effective visibility.

Announcements pages exist at organization, event, and station levels. Authorized organization leadership/c-suite and the relevant chairs can post announcements according to scoped RBAC. Announcements support downloadable documents, displayed images, uploaded playable videos, embedded external videos, and internal/external links.

Clicking an external link must show a warning with the destination and allow the user to continue or cancel. Internal links must still enforce access checks. External video embeds are supported; a notice before loading third-party content is a proposed behavior, not yet a separately confirmed policy.

Proposed file handling includes actual-type validation, configured size limits, generated storage identifiers, sanitized HTML/Markdown previews, and permission checks on downloads. Preview/conversion of doc/docx and PDF files, upload scanning, storage service, retention, and quotas remain design decisions. Arbitrary uploaded scripts or unsanitized HTML must not execute in the application origin.

Supported video formats and upload size limits are explicitly pending task-review items.

## 12. Themes, icons, and accessibility

Support configurable color palettes and optionally layouts and font families. The three built-in themes are Sunbeam (light), Moonshadow (dark), and Greystone (neutral colors intended for colorblind users). System administrators, organization administrators, and event administrators can add themes within their authorized scope.

Levels, states, and locations may be represented by icons paired with theme colors. Proposed accessibility rules require text/accessible labels and readable contrast, with meaning preserved without color alone. Greystone’s palette must be evaluated for accessibility rather than assumed accessible solely because its colors are neutral.

Open decisions: theme inheritance and precedence, whether users may choose personal themes, admin restrictions on overrides, layout customization boundaries, font provisioning, and icon/label/color mappings.

## 13. Proposed architecture and operational requirements

Start with a modular Go application rather than independently deployed services. Separate identity/session, organization authorization, event governance, registration, billing entitlements, communication, media, and presentation configuration into domain modules. Domain services enforce invariants; persistence adapters use pgx/sqlc; external adapters handle email, payment, and file storage. This organization is proposed and may change with the high-level architecture.

Use PostgreSQL migrations with tenant ownership recorded on organization-scoped resources. Proposed defenses include ownership constraints, centralized authorization, and optional row-level security after a design review. Protect tenant boundaries in background jobs, caches, live connections, search, and file delivery as well as API handlers.

Proposed operational capabilities include structured logs, health checks, audit records for sensitive administrative actions, email retry handling, database backups with restore exercises, and documented upgrades/rollback. Performance objectives, availability, recovery targets, retention, and expected organization/event sizes are not yet set.

## 14. Direct-installation documentation plan

Deliver the finalized technology stack and installation guide in Markdown, HTML, and PDF, generated from a common source. This specification is provided in those formats now; it is not a tested production installation manual.

The future guide must specify supported OS/runtime versions, required ports, service accounts, directories, secrets configuration, PostgreSQL provisioning, schema migrations, Go builds, Next.js builds, systemd units, Caddy HTTPS configuration, email/payment/storage setup, health checks, backup/restore, upgrades, and rollback.

Proposed deployment sequence: prepare the Linux host; install pinned prerequisites; configure database and service credentials; build and install the API/frontend; apply reviewed migrations; start systemd services; configure Caddy and DNS; verify real workflows; configure backups and monitoring. Validate the exact commands on a representative non-container host before labeling the guide tested. Hosting changes should require provisioning and data movement, not a rewrite of business logic.

## 15. Acceptance criteria

- A user belonging to two organizations can access only the authenticated organization’s protected resources; access to another requires logout and a new organization login.
- Server-side requests and file/chat access cannot be authorized by changing a client-supplied organization identifier.
- Chair appointments count against station/board capacity, and simultaneous assignments cannot overbook a station.
- The configured multiple-station rule is enforced per participant and event.
- Waiting-list decisions follow all three confirmed governance cases, including a pre-board prohibition that the board cannot override.
- Organization role titles appear by default; event overrides change display text without altering permissions.
- Login has the specified two fields, password visibility control, and reset link; omitted usernames default to email.
- Chat works at all four requested scopes, checks permissions/entitlements, restricts direct messages to the active organization, and rejects attachments.
- Supported images/documents attach to organizations, events, and stations with scoped downloads; announcements support both uploaded and embedded video.
- External-link navigation requires the destination warning and can be cancelled.
- Public event pages expose intended search content; private events and attachments reject unauthorized access.
- Sunbeam, Moonshadow, and Greystone preserve icon/text meaning and meet the agreed accessibility target.
- Deployment and documentation use direct installation without containers, and final guides are downloadable in all three formats.

Acceptance criteria that depend on open policy choices will be expanded when those choices are settled. A successful login or build alone is insufficient to establish full application readiness.

## 16. Roadmap

This roadmap is a proposed sequence, not a calendar commitment. Features below remain in scope even when deferred to later phases. No duration estimates are assigned until architecture, staffing, and release scope are known.

### Phase 0 — resolve foundational decisions

Deliver a reviewed requirements baseline, domain lifecycle diagrams, tenant/session model, permission matrix, entitlement model, and chosen initial release scope. Select the repository name/ownership, runtime versions, and direct-host deployment targets. Resolve organization login behavior, capacity reservation, chair membership, and payment/membership rules before committing to the schema.

Exit: high-level architecture and outstanding assumptions are explicitly reviewed; no proposed policy is silently treated as confirmed.

### Phase 1 — repository and development foundation

Create the new repository and modular Go/Next.js structure. Set up PostgreSQL migrations, sqlc generation, direct local setup, automated checks, configuration handling, and initial deployment documentation. Keep legacy repositories as references. Select an email adapter and file-storage interface.

Exit: repeatable non-container developer setup, API/frontend builds, migration execution, and health checks pass.

### Phase 2 — accounts, organization context, and RBAC

Implement accounts, username/email rules, verification/reset, login/logout, memberships, scoped authorization, and active organization sessions. Add server-side isolation and session tests. Define system-versus-organization administrator boundaries.

Exit: cross-organization access is denied across API, UI, files, and session transitions; verified authentication flows work with a test email service.

### Phase 3 — events, boards, stations, and registration

Implement CRUD/lifecycle actions, role titles, chairs, capacities, registrations, station assignments, organization registration policies, and optional waiting lists. Add concurrency checks and policy tests. Implement agreed terminology controls.

Exit: realistic event setup and registration work, capacity/duplicate constraints hold under concurrent requests, and waiting-list governance is enforced.

### Phase 4 — public pages, announcements, media, and themes

Implement public visibility/SEO, attachments, announcements at all levels, external-link warnings, video uploads/embeds, and the three basic themes. Add configured theme extensions, icon mappings, accessible forms, and scoped media delivery.

Exit: public/private behavior, supported downloads/media, external navigation, and accessibility are validated against agreed limits and policies.

### Phase 5 — subscriptions and individual payment

Implement organization plans, seat allocation, individual-funded access, paid entitlements, provider integration, and audited billing transitions. Implement the selected expiration and cancellation policies. Test payment event retries and idempotency.

Exit: agreed billing models work without mixing membership, payment state, and authorization, and replayed provider events cannot duplicate access/payment changes.

### Phase 6 — optional chat

Implement organization/event/station/direct conversations, authorized activation, paid gating, live session checks, and the agreed moderation/retention behavior. Enforce attachment exclusion and current-organization direct messaging.

Exit: live and historical access respect organization/session boundaries, logout terminates access, and subscription/permission gates behave correctly.

### Phase 7 — release validation and direct deployment

Run end-to-end workflows, accessibility review, tenant isolation checks, concurrency/load checks, backup restoration, and upgrade/rollback exercises. Finalize direct installation instructions in Markdown/HTML/PDF and deploy to an agreed staging host before production release.

Exit: required checks pass or diagnosed defects are explicitly accepted for release; tested documentation, support procedures, and production approval are complete. Source commits, environment activation, downloadable documents, and live deployment are distinct actions and must be reported separately.

## 17. Task review and unresolved decisions

- Decide terminology scope and singular/plural configuration.
- Define theme inheritance, personal choices, layout/font boundaries, and icon/label/color mappings.
- Define supported video formats and upload size limits for videos, images, and documents.
- Define attachment quotas/storage, previews, scanning, and retention.
- Decide email-verification timing, username rules, password policy, and token lifetimes.
- Decide organization selection during login and exclusivity across tabs/devices/sessions.
- Define the exact RBAC matrix, elevated system access, and audit requirements.
- Define board/station chair registration rules, capacity changes, approval reservation, and assignment defaults.
- Define waiting-list ordering/promotion, list shutdown, and policy changes after board formation.
- Define chat activation scope, moderation, notifications, retention, and entitlement-expiration behavior.
- Choose paid features, hybrid billing rules, individual-payment scope, seat counting, payment provider, prices, and renewal/refund rules.
- Finalize public/unlisted/private modes and any public announcement visibility.
- Define archival/deletion rules for domain resources and historical records.
- Choose versions, OS, database host, server sizing, availability/performance/recovery targets, and deployment automation.
- Decide whether legacy data migration is required and select repository identity.
- Define initial release scope and estimates after the architecture is supplied.

## 18. Deliverable status

This requirements baseline and roadmap are downloadable documentation only. They do not create or push a Git repository, activate a new cloud environment version, or deploy a public application. Implementation has not begun for the new codebase. Changes made during implementation must update this common documentation source and regenerate all formats.
