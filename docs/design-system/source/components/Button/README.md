# Button

Starts a domain action such as Register, Approve, Appoint chair or Join waiting list. Classes: `pe-btn` plus one of `pe-btn--primary`, `pe-btn--secondary`, `pe-btn--quiet`, `pe-btn--danger`.

You provide: the label, the variant, and an optional leading icon (inline SVG with class `pe-icon`).

- One `pe-btn--primary` per view, for the action the screen exists for.
- Label the action with a verb and its object: "Remove from station", "Approve registration".
- `pe-btn--danger` carries the `alert` icon and names what is removed. In Greystone its fill is close to the primary fill, so the words and the icon carry the meaning.
- A button the person lacks permission or entitlement for is left out of the page. Use `aria-disabled="true"` for an action that exists and is waiting on something, such as a station at capacity, and say why beside it.
- Minimum height is 44px in every variant.
