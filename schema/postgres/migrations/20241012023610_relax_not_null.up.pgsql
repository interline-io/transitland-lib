BEGIN;

alter table gtfs_frequencies alter column exact_times drop not null;
alter table gtfs_fare_attributes alter column transfer_duration drop not null;

alter table gtfs_fare_rules alter column origin_id drop not null;
alter table gtfs_fare_rules alter column destination_id drop not null;
alter table gtfs_fare_rules alter column contains_id drop not null;

COMMIT;
