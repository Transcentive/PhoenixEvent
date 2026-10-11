# TextField

A labelled single-line input with an optional hint and an error state. Classes: `pe-field` wrapping `pe-field__label`, `pe-input`, and `pe-field__hint` or `pe-field__error`.

You provide: the label text, the input's `id`, `name` and `autocomplete`, and hint or error copy.

- The label sits above the input and is always visible. Placeholder text stays empty.
- For an error, add `pe-field--error` to the wrapper, set `aria-invalid="true"`, and point `aria-describedby` at the error line. The error line leads with the `alert` icon and says how to fix the value.
- When an organization renames Event or Station, the label uses the configured term in its singular or plural form.
