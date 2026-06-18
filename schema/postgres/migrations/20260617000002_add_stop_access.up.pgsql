BEGIN;

ALTER TABLE gtfs_stops ADD COLUMN stop_access integer;
ALTER TABLE tl_materialized_active_stops ADD COLUMN stop_access integer;

COMMIT;
