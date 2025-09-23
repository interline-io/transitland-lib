BEGIN;

ALTER TABLE gtfs_stops DROP COLUMN stop_access;

COMMIT;
