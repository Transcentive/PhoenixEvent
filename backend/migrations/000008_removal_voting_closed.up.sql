-- The owner's ruling of November 11, 2026: votes may change until the Event
-- Board closes the voting period; the Event Chair records the close, or
-- Organization Leadership before an Event Chair exists.
ALTER TABLE board_removals ADD COLUMN voting_closed_at timestamptz;
