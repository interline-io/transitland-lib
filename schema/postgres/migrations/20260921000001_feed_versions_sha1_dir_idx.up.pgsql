-- Supports the FeedVersionFilter.sha1_dir filter and "sha1 = ? OR sha1_dir = ?"
-- lookups on fetch, which otherwise fall back to a sequential scan.
BEGIN;

CREATE INDEX IF NOT EXISTS feed_versions_sha1_dir_idx ON feed_versions (sha1_dir);

COMMIT;
