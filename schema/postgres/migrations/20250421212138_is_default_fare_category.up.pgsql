BEGIN;

ALTER TABLE gtfs_rider_categories ADD COLUMN is_default_fare_category integer;

COMMIT;