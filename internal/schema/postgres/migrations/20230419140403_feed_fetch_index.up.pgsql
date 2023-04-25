BEGIN;

create index concurrently on feed_fetches(feed_id,fetched_at) where success = true;
create index concurrently on feed_fetches(feed_id,fetched_at) where success = false;
create index on gtfs_agencies(feed_version_id) include (agency_id);

COMMIT;