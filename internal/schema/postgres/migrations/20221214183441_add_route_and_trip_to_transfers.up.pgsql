BEGIN;

ALTER TABLE gtfs_transfers ADD COLUMN from_route_id bigint;
ALTER TABLE gtfs_transfers ADD COLUMN to_route_id bigint;
CREATE INDEX ON gtfs_transfers(from_route_id);
CREATE INDEX ON gtfs_transfers(to_route_id);
ALTER TABLE gtfs_transfers ADD CONSTRAINT gtfs_transfers_from_route_id_fkey foreign key (from_route_id) REFERENCES gtfs_routes(id);
ALTER TABLE gtfs_transfers ADD CONSTRAINT gtfs_transfers_to_route_id_fkey foreign key (to_route_id) REFERENCES gtfs_routes(id);

ALTER TABLE gtfs_transfers ADD COLUMN from_trip_id bigint;
ALTER TABLE gtfs_transfers ADD COLUMN to_trip_id bigint;
CREATE INDEX ON gtfs_transfers(from_trip_id);
CREATE INDEX ON gtfs_transfers(to_trip_id);
ALTER TABLE gtfs_transfers ADD CONSTRAINT gtfs_transfers_from_trip_id_fkey foreign key (from_trip_id) REFERENCES gtfs_trips(id);
ALTER TABLE gtfs_transfers ADD CONSTRAINT gtfs_transfers_to_trip_id_fkey foreign key (to_trip_id) REFERENCES gtfs_trips(id);

COMMIT;