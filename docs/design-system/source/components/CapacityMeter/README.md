# CapacityMeter

Shows how many places of a station or event board are taken. Classes: `pe-meter`, with `pe-meter--caution` or `pe-meter--critical`, wrapping `pe-meter__row`, `pe-meter__track` and `pe-meter__fill`.

You provide: the station or board name, the count taken, the capacity, and the fill width as a percentage.

- The count always appears in words: "9 of 10 places taken". The bar supports the sentence.
- The chair counts toward capacity. Say "Includes the chair" beside the count.
- Fill role: `positive` below 80 percent, `pe-meter--caution` from 80 percent, `pe-meter--critical` at capacity. These thresholds are a starting proposal.
- At capacity, pair the meter with the Full StatusBadge and, where the waiting list is enabled, the number waiting.
