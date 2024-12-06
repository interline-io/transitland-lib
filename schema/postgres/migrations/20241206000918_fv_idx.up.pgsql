BEGIN;

create index on feed_versions(feed_id) include (fetched_at, id, sha1);

COMMIT;
