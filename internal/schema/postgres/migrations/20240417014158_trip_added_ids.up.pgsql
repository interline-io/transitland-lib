BEGIN;

ALTER TABLE tl_validation_trip_update_stats ADD COLUMN trip_rt_added_ids jsonb;
ALTER TABLE tl_validation_trip_update_stats ADD COLUMN trip_rt_added_count int not null;
ALTER TABLE tl_validation_trip_update_stats ADD COLUMN trip_rt_not_found_ids jsonb;
ALTER TABLE tl_validation_trip_update_stats ADD COLUMN trip_rt_not_found_count int not null;

ALTER TABLE tl_validation_vehicle_position_stats ADD COLUMN trip_rt_added_ids jsonb;
ALTER TABLE tl_validation_vehicle_position_stats ADD COLUMN trip_rt_added_count int not null;
ALTER TABLE tl_validation_vehicle_position_stats ADD COLUMN trip_rt_not_found_ids jsonb;
ALTER TABLE tl_validation_vehicle_position_stats ADD COLUMN trip_rt_not_found_count int not null;


COMMIT;