-- https://codebeautify.org/sqlformatter
CREATE TABLE IF NOT EXISTS "current_feeds" (
  "id" integer primary key autoincrement, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "onestop_id" varchar(255) NOT NULL,
  "spec" varchar(255) NOT NULL,
  "deleted_at" datetime,
  "license" BLOB,
  "auth" BLOB,
  "urls" BLOB,
  "languages" BLOB,
  "associated_feeds" BLOB,
  "feed_namespace_id" varchar(255) NOT NULL,
  "name" varchar(255),
  "file" varchar(255) NOT NULL,
  "feed_tags" BLOB
);
CREATE INDEX idx_current_feeds_onestop_id ON "current_feeds"(onestop_id);

CREATE TABLE IF NOT EXISTS "current_operators" (
  "id" integer primary key autoincrement, 
  "onestop_id" varchar(255) NOT NULL,
  "name" varchar(255) NOT NULL,
  "short_name" varchar(255) NOT NULL,
  "website" varchar(255) NOT NULL,
  "operator_tags" BLOB,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "deleted_at" datetime
);
CREATE INDEX idx_current_operators_onestop_id ON "current_operators"(onestop_id);

CREATE TABLE IF NOT EXISTS "current_operators_in_feed" (
  "id" integer primary key autoincrement, 
  "operator_id" integer not null,
  "feed_id" integer not null,
  "gtfs_agency_id" varchar(255)
);

CREATE TABLE IF NOT EXISTS "feed_version_gtfs_imports" (
  "id" integer primary key autoincrement, 
  "feed_version_id" integer NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "success" bool,
  "import_log" blob,
  "in_progress" bool,
  "exception_log" blob,
  "import_level" integer not null,
  "interpolated_stop_time_count" integer not null,
  "skip_entity_error_count" blob,
  "skip_entity_reference_count" blob,
  "skip_entity_marked_count" blob,
  "skip_entity_filter_count" blob,
  "generated_count" blob,
  "warning_count" blob,
  "entity_count" blob
);

CREATE TABLE IF NOT EXISTS "gtfs_stops" (
  "id" integer primary key autoincrement, 
  "feed_version_id" integer NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "stop_id" varchar(255) NOT NULL, 
  "stop_name" varchar(255) NOT NULL, 
  "stop_code" varchar(255) NOT NULL, 
  "stop_desc" varchar(255) NOT NULL, 
  "zone_id" varchar(255) NOT NULL, 
  "stop_url" varchar(255) NOT NULL, 
  "location_type" integer NOT NULL, 
  "parent_station" integer, 
  "stop_timezone" varchar(255) NOT NULL, 
  "wheelchair_boarding" integer NOT NULL, 
  "level_id" integer,
  "geometry" BLOB NOT NULL
);
CREATE INDEX idx_gtfs_stops_stop_id ON "gtfs_stops"(stop_id);
CREATE INDEX idx_gtfs_stops_parent_station ON "gtfs_stops"(parent_station);
CREATE INDEX idx_gtfs_stops_feed_version_id ON "gtfs_stops"(feed_version_id);

CREATE TABLE IF NOT EXISTS "gtfs_pathways" (
  "id" integer primary key autoincrement, 
  "feed_version_id" integer NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "pathway_id" varchar(255) NOT NULL,
  "from_stop_id" integer NOT NULL,
  "to_stop_id" integer NOT NULL,
  "pathway_mode" integer NOT NULL,
  "is_bidirectional" integer NOT NULL,
  "length" real NOT NULL,
  "traversal_time" integer NOT NULL,
  "stair_count" integer NOT NULL,
  "max_slope" real NOT NULL,
  "min_width" real NOT NULL,
  "signposted_as" varchar(255) NOT NULL,
  "reverse_signposted_as" varchar(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS "gtfs_levels" (
  "id" integer primary key autoincrement, 
  "feed_version_id" integer NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "level_id" varchar(255) NOT NULL,
  "level_index" real NOT NULL,
  "level_name" varchar(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS "gtfs_shapes" (
  "id" integer primary key autoincrement, 
  "feed_version_id" integer NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "shape_id" varchar(255) NOT NULL, 
  "generated" bool NOT NULL,
  "geometry" BLOB NOT NULL
);
CREATE INDEX idx_gtfs_shapes_shape_id ON "gtfs_shapes"(shape_id);
CREATE INDEX idx_gtfs_shapes_feed_version_id ON "gtfs_shapes"(feed_version_id);

CREATE TABLE IF NOT EXISTS "feed_versions" (
  "feed_id" integer, 
  "feed_type" varchar(255) NOT NULL, 
  "sha1" varchar(255) NOT NULL, 
  "sha1_dir" varchar(255) NOT NULL, 
  "file" varchar(255) NOT NULL, 
  "url" varchar(255) NOT NULL, 
  "earliest_calendar_date" datetime NOT NULL, 
  "latest_calendar_date" datetime NOT NULL, 
  "fetched_at" datetime NOT NULL, 
  "id" integer primary key autoincrement, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "created_by" varchar(255),
  "updated_by" varchar(255),
  "name" varchar(255),
  "description" varchar(255)
);
CREATE INDEX idx_feed_versions_sha1 ON "feed_versions"("sha1");
CREATE INDEX idx_feed_versions_earliest_calendar_date ON "feed_versions"(earliest_calendar_date);
CREATE INDEX idx_feed_versions_latest_calendar_date ON "feed_versions"(latest_calendar_date);
CREATE INDEX idx_feed_versions_feed_id ON "feed_versions"(feed_id);
CREATE INDEX idx_feed_versions_feed_type ON "feed_versions"(feed_type);

CREATE TABLE IF NOT EXISTS "feed_version_file_infos" (
  "id" integer primary key autoincrement, 
  "feed_version_id" integer NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "name" varchar(255) not null,
  "size" integer not null,
  "rows" integer not null,
  "columns" int not null,
  "sha1" varchar(255) not null,
  "header" varchar(255) not null,
  "csv_like" bool not null
);
CREATE INDEX idx_feed_version_file_infos_feed_version_id ON "feed_version_file_infos"(feed_version_id);

CREATE TABLE IF NOT EXISTS "feed_version_service_levels" (
    "id" integer primary key autoincrement,
    "feed_version_id" integer NOT NULL,
    "route_id" varchar(255),
    "start_date" datetime NOT NULL,
    "end_date" datetime NOT NULL,
    "agency_name" varchar(255) NOT NULL,
    "route_short_name" varchar(255) NOT NULL,
    "route_long_name" varchar(255) NOT NULL,
    "route_type" integer NOT NULL,
    "monday" integer NOT NULL,
    "tuesday" integer NOT NULL,
    "wednesday" integer NOT NULL,
    "thursday" integer NOT NULL,
    "friday" integer NOT NULL,
    "saturday" integer NOT NULL,
    "sunday" integer NOT NULL
);
CREATE INDEX idx_feed_version_service_levels_feed_version_id ON "feed_version_service_levels"(feed_version_id);

CREATE TABLE IF NOT EXISTS "feed_states" (
    "id" integer primary key autoincrement, 
    "feed_id" integer NOT NULL,
    "feed_version_id" integer,
    "feed_realtime_enabled" bool not null,
    "feed_priority" integer,
    "last_fetched_at" datetime,
    "last_successful_fetch_at" datetime,
    "last_imported_at" datetime,
    "last_fetch_error" varchar(255) NOT NULL,
    "tags" BLOB,
    "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
    "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "geometry" BLOB
);

CREATE TABLE IF NOT EXISTS "gtfs_feed_infos" (
  "feed_publisher_name" varchar(255) NOT NULL, 
  "feed_publisher_url" varchar(255) NOT NULL, 
  "feed_lang" varchar(255) NOT NULL, 
  "feed_start_date" datetime, 
  "feed_end_date" datetime, 
  "feed_version_name" varchar(255) NOT NULL, 
  "id" integer primary key autoincrement, 
  "feed_version_id" integer NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX idx_gtfs_feed_infos_feed_version_id ON "gtfs_feed_infos"(feed_version_id);

CREATE TABLE IF NOT EXISTS "gtfs_frequencies" (
  "trip_id" int NOT NULL, 
  "start_time" int NOT NULL, 
  "end_time" int NOT NULL, 
  "headway_secs" integer NOT NULL, 
  "exact_times" integer NOT NULL, 
  "id" integer primary key autoincrement, 
  "feed_version_id" integer NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX idx_gtfs_frequencies_trip_id ON "gtfs_frequencies"(trip_id);
CREATE INDEX idx_gtfs_frequencies_feed_version_id ON "gtfs_frequencies"(feed_version_id);

CREATE TABLE IF NOT EXISTS "gtfs_trips" (
  "route_id" int NOT NULL, 
  "service_id" int NOT NULL, 
  "trip_id" varchar(255) NOT NULL, 
  "trip_headsign" varchar(255) NOT NULL, 
  "trip_short_name" varchar(255) NOT NULL, 
  "direction_id" integer NOT NULL, 
  "block_id" varchar(255) NOT NULL, 
  "shape_id" int, 
  "wheelchair_accessible" integer NOT NULL, 
  "bikes_allowed" integer NOT NULL, 
  "stop_pattern_id" integer NOT NULL, 
  "id" integer primary key autoincrement, 
  "feed_version_id" integer NOT NULL, 
  "journey_pattern_id" integer NOT NULL,
  "journey_pattern_offset" integer NOT NULL,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX idx_gtfs_trips_route_id ON "gtfs_trips"(route_id);
CREATE INDEX idx_gtfs_trips_service_id ON "gtfs_trips"(service_id);
CREATE INDEX idx_gtfs_trips_trip_id ON "gtfs_trips"(trip_id);
CREATE INDEX idx_gtfs_trips_shape_id ON "gtfs_trips"(shape_id);
CREATE INDEX idx_gtfs_trips_stop_pattern_id ON "gtfs_trips"(stop_pattern_id);
CREATE INDEX idx_gtfs_trips_feed_version_id ON "gtfs_trips"(feed_version_id);

CREATE TABLE IF NOT EXISTS "gtfs_agencies" (
  "agency_id" varchar(255) NOT NULL, 
  "agency_name" varchar(255) NOT NULL, 
  "agency_url" varchar(255) NOT NULL, 
  "agency_timezone" varchar(255) NOT NULL, 
  "agency_lang" varchar(255) NOT NULL, 
  "agency_phone" varchar(255) NOT NULL, 
  "agency_fare_url" varchar(255) NOT NULL, 
  "agency_email" varchar(255) NOT NULL, 
  "id" integer primary key autoincrement, 
  "feed_version_id" int NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX idx_gtfs_agencies_agency_id ON "gtfs_agencies"(agency_id);
CREATE INDEX idx_gtfs_agencies_feed_version_id ON "gtfs_agencies"(feed_version_id);

CREATE TABLE IF NOT EXISTS "gtfs_transfers" (
  "from_stop_id" int NOT NULL, 
  "to_stop_id" int NOT NULL, 
  "transfer_type" integer NOT NULL, 
  "min_transfer_time" integer, 
  "id" integer primary key autoincrement, 
  "feed_version_id" integer NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX idx_gtfs_transfers_transfer_type ON "gtfs_transfers"(transfer_type);
CREATE INDEX idx_gtfs_transfers_feed_version_id ON "gtfs_transfers"(feed_version_id);
CREATE INDEX idx_gtfs_transfers_from_stop_id ON "gtfs_transfers"(from_stop_id);
CREATE INDEX idx_gtfs_transfers_to_stop_id ON "gtfs_transfers"(to_stop_id);

CREATE TABLE IF NOT EXISTS "gtfs_calendars" (
  "service_id" varchar(255) NOT NULL, 
  "monday" integer NOT NULL, 
  "tuesday" integer NOT NULL, 
  "wednesday" integer NOT NULL, 
  "thursday" integer NOT NULL, 
  "friday" integer NOT NULL, 
  "saturday" integer NOT NULL, 
  "sunday" integer NOT NULL, 
  "start_date" datetime NOT NULL, 
  "end_date" datetime NOT NULL, 
  "generated" bool NOT NULL, 
  "id" integer primary key autoincrement, 
  "feed_version_id" integer NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX idx_gtfs_calendars_wednesday ON "gtfs_calendars"("wednesday");
CREATE INDEX idx_gtfs_calendars_start_date ON "gtfs_calendars"(start_date);
CREATE INDEX idx_gtfs_calendars_feed_version_id ON "gtfs_calendars"(feed_version_id);
CREATE INDEX idx_gtfs_calendars_end_date ON "gtfs_calendars"(end_date);
CREATE INDEX idx_gtfs_calendars_service_id ON "gtfs_calendars"(service_id);
CREATE INDEX idx_gtfs_calendars_monday ON "gtfs_calendars"("monday");
CREATE INDEX idx_gtfs_calendars_tuesday ON "gtfs_calendars"("tuesday");
CREATE INDEX idx_gtfs_calendars_thursday ON "gtfs_calendars"("thursday");
CREATE INDEX idx_gtfs_calendars_friday ON "gtfs_calendars"("friday");
CREATE INDEX idx_gtfs_calendars_saturday ON "gtfs_calendars"("saturday");
CREATE INDEX idx_gtfs_calendars_sunday ON "gtfs_calendars"("sunday");

CREATE TABLE IF NOT EXISTS "gtfs_calendar_dates" (
  "service_id" int NOT NULL, 
  "date" datetime NOT NULL, 
  "exception_type" integer NOT NULL, 
  "id" integer primary key autoincrement, 
  "feed_version_id" integer NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX idx_gtfs_calendar_dates_date ON "gtfs_calendar_dates"("date");
CREATE INDEX idx_gtfs_calendar_dates_exception_type ON "gtfs_calendar_dates"(exception_type);
CREATE INDEX idx_gtfs_calendar_dates_feed_version_id ON "gtfs_calendar_dates"(feed_version_id);
CREATE INDEX idx_gtfs_calendar_dates_service_id ON "gtfs_calendar_dates"(service_id);

CREATE TABLE IF NOT EXISTS "gtfs_routes" (
  "route_id" varchar(255) NOT NULL, 
  "agency_id" int NOT NULL, 
  "route_short_name" varchar(255) NOT NULL, 
  "route_long_name" varchar(255) NOT NULL, 
  "route_desc" varchar(255) NOT NULL, 
  "route_type" integer NOT NULL, 
  "route_url" varchar(255) NOT NULL, 
  "route_color" varchar(255) NOT NULL, 
  "route_text_color" varchar(255) NOT NULL, 
  "route_sort_order" integer NOT NULL, 
  "id" integer primary key autoincrement, 
  "feed_version_id" integer NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX idx_gtfs_routes_route_id ON "gtfs_routes"(route_id);
CREATE INDEX idx_gtfs_routes_agency_id ON "gtfs_routes"(agency_id);
CREATE INDEX idx_gtfs_routes_route_type ON "gtfs_routes"(route_type);
CREATE INDEX idx_gtfs_routes_feed_version_id ON "gtfs_routes"(feed_version_id);

CREATE TABLE IF NOT EXISTS "gtfs_stop_times" (
  "trip_id" int NOT NULL, 
  "arrival_time" int NOT NULL, 
  "departure_time" int NOT NULL, 
  "stop_id" int NOT NULL, 
  "stop_sequence" integer NOT NULL, 
  "stop_headsign" varchar(255), 
  "pickup_type" integer, 
  "drop_off_type" integer, 
  "shape_dist_traveled" real, 
  "timepoint" integer, 
  "interpolated" integer, 
  "feed_version_id" integer NOT NULL
);
CREATE INDEX idx_stop_times_trip_id ON "gtfs_stop_times"(trip_id);
CREATE INDEX idx_gtfs_stop_times_stop_id ON "gtfs_stop_times"(stop_id);
CREATE INDEX idx_gtfs_stop_times_feed_version_id ON "gtfs_stop_times"(feed_version_id);

CREATE TABLE IF NOT EXISTS "gtfs_fare_rules" (
  "fare_id" int NOT NULL, 
  "route_id" int, 
  "origin_id" varchar(255) NOT NULL, 
  "destination_id" varchar(255) NOT NULL, 
  "contains_id" varchar(255) NOT NULL, 
  "id" integer primary key autoincrement, 
  "feed_version_id" integer NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX idx_gtfs_fare_rules_fare_id ON "gtfs_fare_rules"(fare_id);
CREATE INDEX idx_gtfs_fare_rules_feed_version_id ON "gtfs_fare_rules"(feed_version_id);

CREATE TABLE IF NOT EXISTS "gtfs_fare_attributes" (
  "fare_id" varchar(255) NOT NULL, 
  "price" real NOT NULL, 
  "currency_type" varchar(255) NOT NULL, 
  "payment_method" integer NOT NULL, 
  "transfers" varchar(255), 
  "agency_id" int, 
  "transfer_duration" integer NOT NULL, 
  "id" integer primary key autoincrement, 
  "feed_version_id" integer NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX idx_gtfs_fare_attributes_fare_id ON "gtfs_fare_attributes"(fare_id);
CREATE INDEX idx_gtfs_fare_attributes_feed_version_id ON "gtfs_fare_attributes"(feed_version_id);


-------------------


