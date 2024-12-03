BEGIN;

ALTER TABLE feed_version_service_levels ALTER COLUMN agency_name DROP NOT NULL;
ALTER TABLE feed_version_service_levels ALTER COLUMN route_short_name DROP NOT NULL;
ALTER TABLE feed_version_service_levels ALTER COLUMN route_long_name DROP NOT NULL;
ALTER TABLE feed_version_service_levels ALTER COLUMN route_type DROP NOT NULL;

COMMIT;