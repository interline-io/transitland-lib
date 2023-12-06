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
  "name" varchar(255),
  "description" varchar(255),
  "file" varchar(255) NOT NULL,
  "feed_tags" BLOB
);
CREATE INDEX idx_current_feeds_onestop_id ON "current_feeds"(onestop_id);

CREATE TABLE IF NOT EXISTS "current_operators" (
  "id" integer primary key autoincrement, 
  "onestop_id" varchar(255) NOT NULL,
  "file" varchar(255),
  "name" varchar(255),
  "short_name" varchar(255),
  "website" varchar(255),
  "operator_tags" BLOB,
  "associated_feeds" BLOB,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "deleted_at" datetime
);
CREATE INDEX idx_current_operators_onestop_id ON "current_operators"(onestop_id);

CREATE TABLE IF NOT EXISTS "current_operators_in_feed" (
  "id" integer primary key autoincrement, 
  "operator_id" integer not null,
  "feed_id" integer not null,
  "gtfs_agency_id" varchar(255),
  "resolved_onestop_id" varchar(255),
  "resolved_gtfs_agency_id" varchar(255),
  "resolved_name" varchar(255),
  "resolved_short_name" varchar(255),
  "resolved_places" varchar(255)
);

CREATE TABLE IF NOT EXISTS "feed_version_gtfs_imports" (
  "id" integer primary key autoincrement, 
  "feed_version_id" integer NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "success" bool,
  "schedule_removed" bool,
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
  "tts_stop_name" text,
  "platform_code" text,
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
  "deleted_at" datetime,
  "created_by" varchar(255),
  "updated_by" varchar(255),
  "name" varchar(255),
  "description" varchar(255),
  "fragment" varchar(255)
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
  "csv_like" bool not null,
  "values_count" blob,
  "values_unique" blob
);
CREATE INDEX idx_feed_version_file_infos_feed_version_id ON "feed_version_file_infos"(feed_version_id);

CREATE TABLE IF NOT EXISTS "feed_version_service_windows" (
  "id" integer primary key autoincrement, 
  "feed_version_id" integer NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "feed_start_date" datetime,
  "feed_end_date" datetime,
  "earliest_calendar_date" datetime,
  "latest_calendar_date" datetime,
  "default_timezone" varchar(255),
  "fallback_week" datetime
);
CREATE INDEX feed_version_service_windows_feed_version_id ON "feed_version_service_windows"(feed_version_id);


CREATE TABLE IF NOT EXISTS "feed_version_service_levels" (
    "id" integer primary key autoincrement,
    "feed_version_id" integer NOT NULL,
    "start_date" datetime NOT NULL,
    "end_date" datetime NOT NULL,
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
    "public" bool not null,
    "feed_priority" integer,
    "fetch_wait" integer,
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
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "default_lang" varchar(255),
  "feed_contact_email" varchar(255),
  "feed_contact_url" varchar(255)
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
  "from_route_id" int, 
  "to_route_id" int, 
  "from_trip_id" int, 
  "to_trip_id" int, 
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
  "continuous_pickup" integer,
  "continuous_drop_off" integer,
  "id" integer primary key autoincrement, 
  "feed_version_id" integer NOT NULL, 
  "network_id" varchar(255),
  "as_route" integer,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL
);
CREATE INDEX idx_gtfs_routes_route_id ON "gtfs_routes"(route_id);
CREATE INDEX idx_gtfs_routes_agency_id ON "gtfs_routes"(agency_id);
CREATE INDEX idx_gtfs_routes_route_type ON "gtfs_routes"(route_type);
CREATE INDEX idx_gtfs_routes_feed_version_id ON "gtfs_routes"(feed_version_id);

CREATE TABLE IF NOT EXISTS "gtfs_stop_times" (
  "trip_id" int NOT NULL, 
  "arrival_time" int, 
  "departure_time" int,
  "stop_id" int NOT NULL, 
  "stop_sequence" integer NOT NULL, 
  "stop_headsign" varchar(255), 
  "pickup_type" integer, 
  "drop_off_type" integer, 
  "shape_dist_traveled" real, 
  "timepoint" integer, 
  "continuous_pickup" integer,
  "continuous_drop_off" integer,
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

CREATE TABLE IF NOT EXISTS "tl_agency_onestop_ids" (
  "id" integer primary key autoincrement, 
  "feed_version_id" integer not null,
  "agency_id" integer not null,
  "onestop_id" varchar(255)
);

CREATE TABLE IF NOT EXISTS "tl_stop_onestop_ids" (
  "id" integer primary key autoincrement, 
  "feed_version_id" integer not null,
  "stop_id" integer not null,
  "onestop_id" varchar(255)
);

CREATE TABLE IF NOT EXISTS "tl_route_onestop_ids" (
  "id" integer primary key autoincrement, 
  "feed_version_id" integer not null,
  "route_id" integer not null,
  "onestop_id" varchar(255)
);

CREATE TABLE IF NOT EXISTS "tl_agency_places" (
  "id" integer primary key autoincrement, 
  "feed_version_id" integer not null,
  "agency_id" integer not null,
  "rank" real not null,
  "name" varchar(255),
  "adm1name" varchar(255),
  "adm0name" varchar(255)
);

CREATE TABLE IF NOT EXISTS "tl_route_stops" (
  "id" integer primary key autoincrement, 
  "feed_version_id" integer not null,
  "agency_id" integer not null,
  "route_id" integer not null,
  "stop_id" integer not null
);

CREATE TABLE IF NOT EXISTS "tl_route_headways" (
  "id" integer primary key autoincrement, 
  "feed_version_id" integer not null,
  "route_id" integer not null,
  "selected_stop_id" integer not null,
  "service_id" integer,
  "direction_id" integer,
  "headway_secs" integer,
  "dow_category" integer,
  "service_date" datetime,
  "service_seconds" integer,
  "stop_trip_count" integer,
  "departures" blob
);

CREATE TABLE IF NOT EXISTS "tl_feed_version_geometries" (
  "id" integer primary key autoincrement, 
  "feed_version_id" integer not null,
  "geometry" blob,
  "centroid" blob
);

CREATE TABLE IF NOT EXISTS "tl_agency_geometries" (
  "id" integer primary key autoincrement, 
  "feed_version_id" integer not null,
  "agency_id" integer not null,
  "geometry" blob,
  "centroid" blob
);

CREATE TABLE IF NOT EXISTS "tl_route_geometries" (
  "id" integer primary key autoincrement, 
  "feed_version_id" integer not null,
  "route_id" integer not null,
  "generated" bool not null,
  "shape_id" integer,
  "direction_id" integer,
  "length" real,
  "max_segment_length" real,
  "first_point_max_distance" real,  
  "geometry" blob,
  "centroid" blob,
  "combined_geometry" blob
);

---------------

CREATE TABLE IF NOT EXISTS "gtfs_translations" (
  "id" integer primary key autoincrement, 
  "feed_version_id" int NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "table_name" varchar(255),
  "field_name" varchar(255),
  "field_value" varchar(255),
  "language" varchar(255),
  "translation" varchar(255),
  "record_id" varchar(255),
  "record_sub_id" varchar(255)
);
CREATE INDEX idx_gtfs_translations_feed_version_id ON "gtfs_translations"(feed_version_id);

CREATE TABLE IF NOT EXISTS "gtfs_attributions" (
  "id" integer primary key autoincrement, 
  "feed_version_id" int NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "organization_name" varchar(255),
  "agency_id" integer,
  "route_id" integer,
  "trip_id" integer,
  "is_producer" int,
  "is_operator" int,
  "is_authority" int,
  "attribution_id" varchar(255),
  "attribution_url" varchar(255),
  "attribution_email" varchar(255),
  "attribution_phone" varchar(255)
);
CREATE INDEX idx_gtfs_attributions_feed_version_id ON "gtfs_attributions"(feed_version_id);

CREATE TABLE tl_stop_external_references (
  "id" integer primary key autoincrement, 
  "feed_version_id" int NOT NULL, 
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "target_feed_onestop_id" varchar(255) NOT NULL,
  "target_stop_id" varchar(255) NOT NULL,
  "inactive" bool
);

CREATE TABLE feed_fetches (
    "id" integer primary key autoincrement,
    "feed_id" int,
    "url_type" varchar(255) not null,
    "url" varchar(255) not null,
    "success" bool NOT NULL,
    "fetched_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,
    "fetch_error" varchar(255),
    "response_size" int,
    "response_code" int,
    "response_sha1" varchar(255),
    "feed_version_id" int, 
    "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
    "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL
);




----------------------



CREATE TABLE gtfs_areas (
    "id" integer primary key autoincrement,
    "feed_version_id" int, 
    "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
    "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,

    "area_id" varchar(255) not null,
    "area_name" varchar(255),

    --- interline extensions
    "agency_ids" blob,
    "geometry" blob
);


CREATE TABLE gtfs_stop_areas (
    "id" integer primary key autoincrement,
    "feed_version_id" int, 
    "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
    "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,

    "area_id" int not null,
    "stop_id" int not null
);


CREATE TABLE gtfs_fare_leg_rules (
    "id" integer primary key autoincrement,
    "feed_version_id" int, 
    "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
    "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,

    "leg_group_id" varchar(255),
    "network_id" varchar(255),
    "from_area_id" varchar(255),
    "to_area_id" varchar(255),
    "fare_product_id" varchar(255),
    "transfer_only" integer
);


CREATE TABLE gtfs_fare_transfer_rules (
    "id" integer primary key autoincrement,
    "feed_version_id" int, 
    "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
    "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,

    from_leg_group_id varchar(255),
    to_leg_group_id varchar(255),
    transfer_count int,
    duration_limit int,
    duration_limit_type int,
    fare_transfer_type int,
    fare_product_id varchar(255)
);


CREATE TABLE gtfs_fare_products (
    "id" integer primary key autoincrement,
    "feed_version_id" int, 
    "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
    "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,

    fare_product_id varchar(255),
    fare_product_name varchar(255),
    amount real,
    currency varchar(255),

    --- interline extensions
    rider_category_id varchar(255),
    fare_media_id varchar(255),
    duration_start int,
    duration_amount real,
    duration_unit int,
    duration_type int
);


CREATE TABLE gtfs_fare_media (
    "id" integer primary key autoincrement,
    "feed_version_id" int, 
    "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
    "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,

    --- interline extensions
    fare_media_id varchar(255) NOT NULL,
    fare_media_name varchar(255),
    fare_media_type int
);


CREATE TABLE gtfs_rider_categories (
    "id" integer primary key autoincrement,
    "feed_version_id" int, 
    "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL, 
    "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,

    --- interline extensions
    rider_category_id varchar(255) NOT NULL,
    rider_category_name varchar(255) NOT NULL,
    min_age int,
    max_age int,
    eligibility_url varchar(255)
);



CREATE TABLE tl_validation_reports (
"id" integer primary key autoincrement,
"feed_version_id" int,
"created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,
"updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,
"reported_at" datetime NOT NULL
);

CREATE TABLE tl_validation_trip_update_stats (
"id" integer primary key autoincrement,
"validation_report_id" int NOT NULL,
"agency_id" varchar(255) NOT NULL,
"route_id" varchar(255) NOT NULL,
"trip_scheduled_ids" varchar(255),
"trip_scheduled_count" int NOT NULL,
"trip_match_count" int NOT NULL
);


CREATE TABLE tl_validation_vehicle_position_stats (
"id" integer primary key autoincrement,
"validation_report_id" int NOT NULL,
"agency_id" varchar(255) NOT NULL,
"route_id" varchar(255) NOT NULL,
"trip_scheduled_ids" varchar(255),
"trip_scheduled_count" int NOT NULL,
"trip_match_count" int NOT NULL
);









