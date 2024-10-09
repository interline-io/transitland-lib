BEGIN;

-- these indexes are covered by other compound indexes
drop index index_gtfs_calendar_dates_on_service_id;
drop index index_gtfs_calendar_dates_on_exception_type;
drop index tl_census_tables_dataset_id_idx;
drop index feed_version_service_levels_feed_version_id_idx;
drop index feed_fetches_feed_id_idx;
drop index index_gtfs_trips_on_trip_headsign;
drop index index_gtfs_trips_on_trip_short_name;

COMMIT;
