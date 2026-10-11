# AnnouncementCard

One announcement at organization, event or station level, with its links and attachments. Classes: `pe-card pe-announcement`, with `pe-announcement__scope`, `__title`, `__body`, `__meta` and a `pe-attachments` list.

You provide: the level and its name, the title, the body, the author's name and displayed role title, the date, and the attachment list (file name, type, size).

- The scope line leads with the level icon (`organization`, `event` or `station`) and the configured label for that level.
- The author line shows the role title as displayed for that event: the event override when one exists, the organization title otherwise.
- A link that leaves the application carries the `external-link` icon and opens the ExternalLinkDialog.
- Each attachment row shows the `paperclip` icon, the file name as a link, then type and size. Supported types: jpg, jpeg, png, md, html, pdf, doc, docx.
- Images display inside the body. Uploaded and embedded video sit in a 16:9 frame between the body and the attachments.
