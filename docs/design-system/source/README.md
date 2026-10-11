PhoenixEvent is the interface for people working an organization's event together, divided into stations. It serves PTAs, community committees, banks and businesses, so it reads plainly, works for a first-time volunteer, and keeps its meaning in every theme.

## Content fundamentals

- **Labels and names are separate.** "Event" and "Station" are default labels an administrator can replace with terms such as Function, Activity, Festival or Department. "Halloween Bash" and "Valentine Dance" are names. Write copy with the label as a variable in singular and plural form: "Join a {station}", "3 {stations} open".
- **Role titles are display text.** Show the event's title for a role when it has one (Event Director, Event Chair, Event Executive, Event Manager) and the organization's title otherwise. Permissions keep their own stable names.
- **Sentence case everywhere:** "Log in", "Forgot password?", "Join waiting list". Names keep their own capitals.
- **Buttons are a verb and its object:** "Approve registration", "Appoint chair", "Remove from station". The toggle on a password reads "Show" or "Hide".
- **State what is true, in numbers:** "9 of 10 places taken. Includes the chair." "3 on the waiting list."
- **Address the person as "you".** The organization is named: "This link opens a site outside Lincoln Elementary PTA."
- **Errors say how to fix the value:** "Enter 1 or more. Capacity includes the chair."
- **Words only.** Emoji stay out of interface copy. Icons come from the Icons group.

## Visual foundations

**Themes.** Three built-in themes share every token name. Set one with `data-theme` on the root or on any container.

| Theme | id | Character |
| --- | --- | --- |
| Sunbeam | `sunbeam` | Light. Warm paper `surface`, sun-gold `brand`. The default. |
| Moonshadow | `moonshadow` | Dark. Night-blue `surface`, pale moon `brand`. |
| Greystone | `greystone` | Neutral greys for people who see color differently. Roles differ by lightness, and icons and words carry the meaning. |

**Color.**

- Set the page on `surface`. Put cards, inputs, dialogs and the login panel on `surface-raised`. Use `surface-sunken` for meter tracks and inset boxes.
- Set text in `ink`. Set hints and metadata in `ink-muted`.
- Use `brand` as a fill only, with `on-brand` on top. It marks the one primary button in a view.
- Set links and quiet buttons in `brand-ink`, always underlined.
- Use `border-strong` for every edge a person needs to find: inputs, buttons, meter tracks, dialogs. Use `border` for decorative hairlines.
- States use four roles, each a solid color on its own `-surface` tint: `positive` (open, approved, public), `caution` (pending, waiting list, nearly full), `critical` (full, error, destructive), `neutral` (cancelled, private, unlisted). `positive` is blue and `critical` is red, and they differ in lightness.
- A state always appears as icon, word and color together. See StatusBadge.
- Put `scrim` behind dialogs.

**Contrast.** `ink`, `ink-muted` and `brand-ink` hold 4.5:1 on all three surfaces in all three themes. Each state role holds 4.5:1 on its tint, on `surface` and on `surface-raised`. `border-strong` and `focus` hold 3:1 on all three surfaces. `on-brand` on `brand` and `on-critical` on `critical` hold 4.5:1. A new theme passes the same pairs.

**Focus.** Every interactive element shows a solid 3px `focus` outline with a 2px offset on keyboard focus.

**Type.** One family, `sans`, the system interface face, so pages load with no font download. `mono` shows addresses. Use `display` for the event name on a public page, `heading-1` to `heading-3` for titles, `body` for copy and inputs, `small` for hints and metadata, `label` for badges and scope lines. Body copy is 16px on a 24px line.

**Spacing.** A 4px base: `space-1` to `space-12`. Use `space-4` between form fields, `space-6` inside cards and dialogs, `space-8` between page sections.

**Radius.** `radius-md` for buttons and inputs, `radius-lg` for cards and dialogs, `radius-pill` for badges and meter tracks, `radius-sm` for inset boxes.

**Shadow.** `shadow-raised` lifts cards from `surface`. `shadow-overlay` lifts dialogs above `scrim`. Borders define edges, and shadows only add depth.

**Layout.** Controls are at least 44px tall. Forms are one column, with labels above inputs. Cards hold one subject each.

**Motion.** Still by default. State changes apply at once.

## Iconography

Fourteen line icons on a 24px grid with a 2px stroke, in the Icons group. Inline them with `stroke="currentColor"` so each icon takes the color of its label.

- **Levels:** `organization`, `event`, `station`. In `ink`, leading the scope line of announcements, navigation and page titles.
- **States:** `check-circle` (approved, open), `clock` (pending approval), `hourglass` (waiting list), `ban` (full), `x-circle` (cancelled), `alert` (error, warning).
- **Visibility:** `globe` (public), `eye-off` (unlisted), `lock` (private).
- **Links and files:** `external-link` after any link that leaves the application, `paperclip` before an attachment.

Each icon sits beside its word. An icon standing alone takes an `aria-label` with the same word.

## Logo

PhoenixEvent has no mark yet. Set the name in `sans` at weight 700, as one word with a capital P and a capital E.

## Components

All components are CSS classes in `components/bundle.css`, prefixed `pe-`, and read only tokens. Button, TextField, PasswordField and LoginForm cover account screens. StatusBadge and CapacityMeter cover registration and station capacity. AnnouncementCard and ExternalLinkDialog cover communication. ThemeSampler shows one card in all three themes.
