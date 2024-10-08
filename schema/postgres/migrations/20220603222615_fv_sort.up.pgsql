BEGIN;

create index on gtfs_stops(feed_version_id,id);

COMMIT;