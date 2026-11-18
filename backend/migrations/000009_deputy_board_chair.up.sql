-- The Deputy Board Chair (spec 7B.8, owner's rulings of November 11, 2026;
-- the name is subject to change): a Board Member the Event Board elects, who
-- serves in the place of the Board Chair when the Board Chair cannot serve.
-- The Deputy keeps a Board Member's seat and vote.
ALTER TABLE event_board ADD COLUMN deputy boolean NOT NULL DEFAULT false;
ALTER TABLE event_board ADD CONSTRAINT event_board_deputy_is_member CHECK (NOT deputy OR position = 'member');
CREATE UNIQUE INDEX event_board_one_deputy ON event_board (event_id) WHERE deputy;
