# StatusBadge

A pill that states one state with an icon, a word and a color role. Classes: `pe-badge` plus `pe-badge--positive`, `pe-badge--caution`, `pe-badge--critical` or `pe-badge--neutral`.

You provide: the state. The icon, word and role come from this table.

| State | Icon | Role |
| --- | --- | --- |
| Approved, Open | `check-circle` | positive |
| Public | `globe` | positive |
| Pending approval | `clock` | caution |
| Waiting list | `hourglass` | caution |
| Full | `ban` | critical |
| Cancelled | `x-circle` | neutral |
| Unlisted | `eye-off` | neutral |
| Private | `lock` | neutral |

- Every badge carries all three: icon, word, color. In Greystone the four roles are greys of different lightness, so the icon and the word identify the state.
- The word is the state itself in sentence case, one or two words.
- One badge per fact. A station that is full and private shows two badges.
