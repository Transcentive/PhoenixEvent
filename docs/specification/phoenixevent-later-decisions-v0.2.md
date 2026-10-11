# PhoenixEvent: decisions set aside for later

Version 0.2, October 9, 2026. Each entry is a decision the owner has chosen to take later, with the date it was recorded and where the specification mentions it.

## 1. Moving from Node.js 20 to a supported version

Recorded October 9, 2026. Specification 0.49, section 2A.4.

One caution, which I added to the specification: Node.js 20 reached the end of its support on April 30, 2026, and no longer gets security fixes. It works with Vite 6, so it's fine for now. At some point both applications should move together to a supported version such as Node.js 22, so they stay the same. That is a separate decision for later.

- **Applies to:** PhoenixEvent and The Mighty Queue, together.
- **Current version:** Node.js 20.20.2, on the owner's Mac.
- **Status:** Open, held for later by the owner.

## 2. One organization per browser at an organization's own address

Recorded October 8, 2026. Specification 0.38 onward, section 4A.5.

The rule that a browser is logged in to one organization at a time holds across subdomains through the shared cookie `pe_org`. At an address an organization owns, such as `events.lincolnpta.org`, the browser never sends that cookie, so the rule cannot be enforced the same way. Two approaches were put to the owner: accept the gap, or send every login at an organization's own address through `login.phoenixevent.com`. The owner accepted neither and holds the decision.

- **Applies to:** PhoenixEvent, when organizations use addresses they own, after the first release.
- **Status:** Open, held for later by the owner.
