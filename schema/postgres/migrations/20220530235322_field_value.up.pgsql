BEGIN;

ALTER TABLE gtfs_translations ADD COLUMN field_value TEXT;

COMMIT;