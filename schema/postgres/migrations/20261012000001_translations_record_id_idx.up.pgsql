-- Supports FeedVersion.translations filtered by record_id; without it, the
-- filter reads every translation in the feed version.
BEGIN;

CREATE INDEX IF NOT EXISTS gtfs_translations_feed_version_id_record_id_idx ON gtfs_translations (feed_version_id, record_id);

COMMIT;
