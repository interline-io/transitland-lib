BEGIN;

ALTER TABLE gtfs_stops ADD COLUMN stop_access integer;

COMMIT;
