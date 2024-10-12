BEGIN;

alter table gtfs_frequencies alter column exact_times drop not null;
alter table gtfs_fare_attributes alter column transfer_duration drop not null;

COMMIT;
