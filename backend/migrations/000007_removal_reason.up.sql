-- The owner's ruling of November 11, 2026: "termination with cause" is now
-- simply "termination"; the Cause field is renamed Reason and is optional.
ALTER TABLE board_removals RENAME COLUMN cause TO reason;
