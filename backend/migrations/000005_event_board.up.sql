-- Phase 3: forming the event board (5E, 7B.6, 7B.7; screens S7, S8, D1).
-- A seat on an event board is held by one person in one position: the chair,
-- a board member, or the Event Administrator. The Event Administrator holds
-- a seat only when the organization's policy says so (7B.7); the entry is
-- kept either way, so the Event Administrator is known to the event.
CREATE TABLE event_board (
    event_id        uuid NOT NULL REFERENCES events (id) ON DELETE CASCADE,
    organization_id uuid NOT NULL REFERENCES organizations (id),
    account_id      uuid NOT NULL REFERENCES accounts (id),
    position        text NOT NULL CHECK (position IN ('chair', 'member', 'event_administrator')),
    appointed_by    uuid NOT NULL REFERENCES accounts (id),
    appointed_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id, account_id)
);
-- One chair and one Event Administrator per event.
CREATE UNIQUE INDEX event_board_one_chair ON event_board (event_id) WHERE position = 'chair';
CREATE UNIQUE INDEX event_board_one_admin ON event_board (event_id) WHERE position = 'event_administrator';
CREATE INDEX event_board_account ON event_board (organization_id, account_id);
