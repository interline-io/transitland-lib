-- Index feed_versions by the feed contents checksum.
--
-- sha1 has had an index since 20220527171244, sha1_dir never did. Lookups that
-- accept either checksum ask for "sha1 = ? OR sha1_dir = ?" -- what
-- GetFeedVersionBySHA1 runs on every static fetch, and now what the
-- FeedVersionFilter.sha1 filter runs behind the feed version REST and GraphQL
-- endpoints. Postgres cannot combine an indexed and an unindexed column into a
-- BitmapOr, so without this index the whole predicate degrades to a sequential
-- scan of feed_versions.
BEGIN;

CREATE INDEX IF NOT EXISTS feed_versions_sha1_dir_idx ON feed_versions (sha1_dir);

COMMIT;
