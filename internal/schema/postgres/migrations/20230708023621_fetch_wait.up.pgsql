BEGIN;

ALTER TABLE feed_states ADD COLUMN fetch_wait integer;

END;