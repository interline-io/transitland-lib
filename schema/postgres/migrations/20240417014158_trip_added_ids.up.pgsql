BEGIN;

ALTER TABLE tl_validation_trip_update_stats ADD COLUMN trip_rt_added_ids jsonb;
ALTER TABLE tl_validation_trip_update_stats ADD COLUMN trip_rt_added_count int;
ALTER TABLE tl_validation_trip_update_stats ADD COLUMN trip_rt_not_found_ids jsonb;
ALTER TABLE tl_validation_trip_update_stats ADD COLUMN trip_rt_not_found_count int;
UPDATE tl_validation_trip_update_stats SET trip_rt_added_count = 0, trip_rt_not_found_count = 0;
ALTER TABLE tl_validation_trip_update_stats ALTER COLUMN trip_rt_added_count SET NOT NULL;

ALTER TABLE tl_validation_vehicle_position_stats ADD COLUMN trip_rt_added_ids jsonb;
ALTER TABLE tl_validation_vehicle_position_stats ADD COLUMN trip_rt_added_count int;
ALTER TABLE tl_validation_vehicle_position_stats ADD COLUMN trip_rt_not_found_ids jsonb;
ALTER TABLE tl_validation_vehicle_position_stats ADD COLUMN trip_rt_not_found_count int;
UPDATE tl_validation_vehicle_position_stats SET trip_rt_added_count = 0, trip_rt_not_found_count = 0;
ALTER TABLE tl_validation_vehicle_position_stats ALTER COLUMN trip_rt_added_count SET NOT NULL;

COMMIT;