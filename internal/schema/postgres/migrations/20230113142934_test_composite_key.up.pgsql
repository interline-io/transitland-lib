BEGIN;

ALTER TABLE gtfs_routes DROP CONSTRAINT fk_rails_e5eb0f1573;
ALTER TABLE gtfs_routes DROP COLUMN agency_id;
ALTER TABLE gtfs_routes ADD COLUMN agency_id text;

ALTER TABLE ONLY gtfs_routes ADD CONSTRAINT fk_gtfs_routes_agency_id FOREIGN KEY (feed_version_id, agency_id) REFERENCES gtfs_agencies(feed_version_id, agency_id);

COMMIT;