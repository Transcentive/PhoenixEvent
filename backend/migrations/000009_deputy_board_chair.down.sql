DROP INDEX event_board_one_deputy;
ALTER TABLE event_board DROP CONSTRAINT event_board_deputy_is_member;
ALTER TABLE event_board DROP COLUMN deputy;
