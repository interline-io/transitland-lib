BEGIN;

ALTER TABLE tl_validation_trip_update_stats ADD COLUMN trip_scheduled_not_matched int not null;
ALTER TABLE tl_validation_trip_update_stats ADD COLUMN trip_rt_ids jsonb;
ALTER TABLE tl_validation_trip_update_stats ADD COLUMN trip_rt_count int not null;
ALTER TABLE tl_validation_trip_update_stats ADD COLUMN trip_rt_matched int not null;
ALTER TABLE tl_validation_trip_update_stats ADD COLUMN trip_rt_not_matched int not null;

ALTER TABLE tl_validation_vehicle_position_stats ADD COLUMN trip_scheduled_not_matched int not null;
ALTER TABLE tl_validation_vehicle_position_stats ADD COLUMN trip_rt_ids jsonb;
ALTER TABLE tl_validation_vehicle_position_stats ADD COLUMN trip_rt_count int not null;
ALTER TABLE tl_validation_vehicle_position_stats ADD COLUMN trip_rt_matched int not null;
ALTER TABLE tl_validation_vehicle_position_stats ADD COLUMN trip_rt_not_matched int not null;


COMMIT;