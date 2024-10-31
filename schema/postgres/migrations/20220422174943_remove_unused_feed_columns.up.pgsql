BEGIN;

ALTER TABLE current_feeds DROP COLUMN feed_namespace_id;
ALTER TABLE current_feeds DROP COLUMN other_ids;
ALTER TABLE current_feeds DROP COLUMN associated_feeds;

COMMIT;
