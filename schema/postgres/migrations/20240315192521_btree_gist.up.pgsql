CREATE EXTENSION btree_gist;

BEGIN;

create index on gtfs_stops USING gist(feed_version_id,geometry);

COMMIT;