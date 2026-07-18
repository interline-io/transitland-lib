BEGIN;

-- Supports "latest successful static_current fetch per feed" lookups (feed version
-- import list, public feed list): seek (feed_id, url_type) and read the newest row by
-- fetched_at, instead of scanning a feed's fetches backward and filtering url_type and
-- success per row. feed_version_id rides in INCLUDE so the lookup stays index-only.
CREATE INDEX IF NOT EXISTS feed_fetches_feed_id_url_type_fetched_at_success_idx
    ON feed_fetches (feed_id, url_type, fetched_at DESC)
    INCLUDE (feed_version_id)
    WHERE success;

COMMIT;
