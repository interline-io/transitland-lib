BEGIN;

ALTER TABLE feed_versions ADD COLUMN fragment text;

COMMIT;
