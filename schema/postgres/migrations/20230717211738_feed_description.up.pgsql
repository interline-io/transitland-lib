BEGIN;

ALTER TABLE current_feeds ADD COLUMN description text;

COMMIT;