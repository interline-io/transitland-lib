BEGIN;

CREATE INDEX ON gtfs_trips(feed_version_id,journey_pattern_id);

COMMIT;