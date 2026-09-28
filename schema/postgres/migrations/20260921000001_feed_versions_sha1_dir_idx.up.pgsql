-- Lets "sha1 = ? OR sha1_dir = ?" use a BitmapOr instead of a sequential scan.
BEGIN;

CREATE INDEX IF NOT EXISTS feed_versions_sha1_dir_idx ON feed_versions (sha1_dir);

COMMIT;
