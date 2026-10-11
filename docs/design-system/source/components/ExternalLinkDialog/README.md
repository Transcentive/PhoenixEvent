# ExternalLinkDialog

The warning shown when a person follows a link that leaves the application. It shows the destination and offers Continue and Cancel. Classes: `pe-scrim` behind a `pe-dialog` with `pe-dialog__title`, `__body`, `__dest` and `__actions`.

You provide: the full destination address and the organization name.

- The destination shows in full in `mono`, wrapping at any character, so the whole address is readable.
- Two actions, in this order: Cancel (secondary), then Continue (primary, with the `external-link` icon). Focus starts on Cancel. Escape cancels.
- Use `role="alertdialog"` with `aria-modal="true"`, labelled by the title and described by the body line.
- Internal links go straight to their target, where the access check runs.
