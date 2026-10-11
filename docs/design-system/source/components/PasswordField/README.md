# PasswordField

A password input with a Show/Hide button. The password is hidden by default. Classes: `pe-field`, `pe-password` wrapping a `pe-input` and a `pe-btn pe-btn--secondary`.

You provide: the label, the input's `id` and `autocomplete` value (`current-password` at login, `new-password` at reset), and the click handler that swaps the input type.

- The button reads "Show" while hidden and "Hide" while visible. It is a real `button type="button"` with `aria-controls` pointing at the input and `aria-pressed` reflecting the state.
- A `role="status"` line announces "Password is visible." or "Password is hidden." after each press.
- The button keeps a fixed minimum width so the field holds its size when the word changes.
