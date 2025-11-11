-- https://codebeautify.org/sqlformatter
CREATE TABLE IF NOT EXISTS "current_feeds" (
  "id" integer primary key autoincrement,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "onestop_id" varchar(255),
  "spec" varchar(255),
  "deleted_at" datetime,
  "license" BLOB,
  "auth" BLOB,
  "urls" BLOB,
  "languages" BLOB,
  "name" varchar(255),
  "description" varchar(255),
  "file" varchar(255),
  "feed_tags" BLOB
);
CREATE INDEX idx_current_feeds_onestop_id ON "current_feeds"(onestop_id);
CREATE TABLE IF NOT EXISTS "current_operators" (
  "id" integer primary key autoincrement,
  "onestop_id" varchar(255),
  "file" varchar(255),
  "name" varchar(255),
  "short_name" varchar(255),
  "website" varchar(255),
  "operator_tags" BLOB,
  "associated_feeds" BLOB,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
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
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
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
  "entity_count" blob,
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);
CREATE TABLE IF NOT EXISTS "gtfs_stops" (
  "id" integer primary key autoincrement,
  "feed_version_id" integer NOT NULL,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "stop_id" varchar(255),
  "stop_name" varchar(255),
  "stop_code" varchar(255),
  "stop_desc" varchar(255),
  "zone_id" varchar(255),
  "stop_url" varchar(255),
  "location_type" integer,
  "parent_station" integer,
  "stop_timezone" varchar(255),
  "wheelchair_boarding" integer,
  "level_id" integer,
  "tts_stop_name" text,
  "platform_code" text,
  "area_id" text,
  "geometry" BLOB,
  "textsearch" TEXT,
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(parent_station) references gtfs_stops(id),
  foreign key(level_id) references gtfs_levels(id)
);
CREATE INDEX idx_gtfs_stops_stop_id ON "gtfs_stops"(stop_id);
CREATE INDEX idx_gtfs_stops_parent_station ON "gtfs_stops"(parent_station);
CREATE INDEX idx_gtfs_stops_feed_version_id ON "gtfs_stops"(feed_version_id);
CREATE TABLE IF NOT EXISTS "gtfs_pathways" (
  "id" integer primary key autoincrement,
  "feed_version_id" integer NOT NULL,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "pathway_id" varchar(255),
  "from_stop_id" integer,
  "to_stop_id" integer,
  "pathway_mode" integer,
  "is_bidirectional" integer,
  "length" real,
  "traversal_time" integer,
  "stair_count" integer,
  "max_slope" real,
  "min_width" real,
  "signposted_as" varchar(255),
  "reverse_signposted_as" varchar(255),
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(from_stop_id) references gtfs_stops(id),
  foreign key(to_stop_id) references gtfs_stops(id)
);
CREATE TABLE IF NOT EXISTS "gtfs_levels" (
  "id" integer primary key autoincrement,
  "feed_version_id" integer NOT NULL,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "level_id" varchar(255),
  "level_index" real,
  "level_name" varchar(255),
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);
CREATE TABLE IF NOT EXISTS "gtfs_shapes" (
  "id" integer primary key autoincrement,
  "feed_version_id" integer NOT NULL,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "shape_id" varchar(255),
  "generated" bool,
  "geometry" BLOB,
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);
CREATE INDEX idx_gtfs_shapes_shape_id ON "gtfs_shapes"(shape_id);
CREATE INDEX idx_gtfs_shapes_feed_version_id ON "gtfs_shapes"(feed_version_id);
CREATE TABLE IF NOT EXISTS "feed_versions" (
  "feed_id" integer,
  "sha1" varchar(255),
  "sha1_dir" varchar(255),
  "file" varchar(255),
  "url" varchar(255),
  "earliest_calendar_date" datetime,
  "latest_calendar_date" datetime,
  "fetched_at" datetime,
  "id" integer primary key autoincrement,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "deleted_at" datetime,
  "created_by" varchar(255),
  "updated_by" varchar(255),
  "name" varchar(255),
  "description" varchar(255),
  "fragment" varchar(255),
  foreign key(feed_id) references current_feeds(id)
);
CREATE INDEX idx_feed_versions_sha1 ON "feed_versions"("sha1");
CREATE INDEX idx_feed_versions_earliest_calendar_date ON "feed_versions"(earliest_calendar_date);
CREATE INDEX idx_feed_versions_latest_calendar_date ON "feed_versions"(latest_calendar_date);
CREATE INDEX idx_feed_versions_feed_id ON "feed_versions"(feed_id);
CREATE TABLE IF NOT EXISTS "feed_version_file_infos" (
  "id" integer primary key autoincrement,
  "feed_version_id" integer NOT NULL,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "name" varchar(255) not null,
  "size" integer not null,
  "rows" integer not null,
  "columns" integer not null,
  "sha1" varchar(255) not null,
  "header" varchar(255) not null,
  "csv_like" bool not null,
  "values_count" blob,
  "values_unique" blob,
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);
CREATE INDEX idx_feed_version_file_infos_feed_version_id ON "feed_version_file_infos"(feed_version_id);
CREATE TABLE IF NOT EXISTS "feed_version_service_windows" (
  "id" integer primary key autoincrement,
  "feed_version_id" integer not null,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "feed_start_date" datetime,
  "feed_end_date" datetime,
  "earliest_calendar_date" datetime,
  "latest_calendar_date" datetime,
  "default_timezone" varchar(255),
  "fallback_week" datetime,
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);
CREATE INDEX feed_version_service_windows_feed_version_id ON "feed_version_service_windows"(feed_version_id);
CREATE TABLE IF NOT EXISTS "feed_version_service_levels" (
  "id" integer primary key autoincrement,
  "feed_version_id" integer not null,
  "start_date" datetime,
  "end_date" datetime,
  "monday" integer,
  "tuesday" integer,
  "wednesday" integer,
  "thursday" integer,
  "friday" integer,
  "saturday" integer,
  "sunday" integer,
  foreign key(feed_version_id) REFERENCES feed_versions(id)
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
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(feed_id) references current_feeds(id)
);
CREATE TABLE IF NOT EXISTS "gtfs_feed_infos" (
  "feed_publisher_name" varchar(255),
  "feed_publisher_url" varchar(255),
  "feed_lang" varchar(255),
  "feed_start_date" datetime,
  "feed_end_date" datetime,
  "feed_version_name" varchar(255),
  "id" integer primary key autoincrement,
  "feed_version_id" integer NOT NULL,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "default_lang" varchar(255),
  "feed_contact_email" varchar(255),
  "feed_contact_url" varchar(255),
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);
CREATE INDEX idx_gtfs_feed_infos_feed_version_id ON "gtfs_feed_infos"(feed_version_id);
CREATE TABLE IF NOT EXISTS "gtfs_frequencies" (
  "trip_id" integer NOT NULL,
  "start_time" int,
  "end_time" int,
  "headway_secs" integer,
  "exact_times" integer,
  "id" integer primary key autoincrement,
  "feed_version_id" integer NOT NULL,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(trip_id) references gtfs_trips(id)
);
CREATE INDEX idx_gtfs_frequencies_trip_id ON "gtfs_frequencies"(trip_id);
CREATE INDEX idx_gtfs_frequencies_feed_version_id ON "gtfs_frequencies"(feed_version_id);
CREATE TABLE IF NOT EXISTS "gtfs_trips" (
  "route_id" integer NOT NULL,
  "service_id" integer NOT NULL,
  "trip_id" varchar(255),
  "trip_headsign" varchar(255),
  "trip_short_name" varchar(255),
  "direction_id" integer,
  "block_id" varchar(255),
  "shape_id" int,
  "wheelchair_accessible" integer,
  "bikes_allowed" integer,
  "stop_pattern_id" integer,
  "id" integer primary key autoincrement,
  "feed_version_id" integer NOT NULL,
  "journey_pattern_id" integer,
  "journey_pattern_offset" integer,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(route_id) references gtfs_routes(id),
  foreign key(service_id) references gtfs_calendars(id)
);
CREATE INDEX idx_gtfs_trips_route_id ON "gtfs_trips"(route_id);
CREATE INDEX idx_gtfs_trips_service_id ON "gtfs_trips"(service_id);
CREATE INDEX idx_gtfs_trips_trip_id ON "gtfs_trips"(trip_id);
CREATE INDEX idx_gtfs_trips_shape_id ON "gtfs_trips"(shape_id);
CREATE INDEX idx_gtfs_trips_stop_pattern_id ON "gtfs_trips"(stop_pattern_id);
CREATE INDEX idx_gtfs_trips_feed_version_id ON "gtfs_trips"(feed_version_id);
CREATE TABLE IF NOT EXISTS "gtfs_agencies" (
  "agency_id" varchar(255),
  "agency_name" varchar(255),
  "agency_url" varchar(255),
  "agency_timezone" varchar(255),
  "agency_lang" varchar(255),
  "agency_phone" varchar(255),
  "agency_fare_url" varchar(255),
  "agency_email" varchar(255),
  "id" integer primary key autoincrement,
  "feed_version_id" integer NOT NULL,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "textsearch" TEXT,
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);
CREATE INDEX idx_gtfs_agencies_agency_id ON "gtfs_agencies"(agency_id);
CREATE INDEX idx_gtfs_agencies_feed_version_id ON "gtfs_agencies"(feed_version_id);
CREATE TABLE IF NOT EXISTS "gtfs_transfers" (
  "from_stop_id" int,
  "to_stop_id" int,
  "from_route_id" int,
  "to_route_id" int,
  "from_trip_id" int,
  "to_trip_id" int,
  "transfer_type" integer,
  "min_transfer_time" integer,
  "id" integer primary key autoincrement,
  "feed_version_id" integer NOT NULL,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  foreign key(feed_version_id) REFERENCES feed_versions(id)
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
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  foreign key(feed_version_id) REFERENCES feed_versions(id)
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
  "service_id" integer NOT NULL,
  "date" datetime NOT NULL,
  "exception_type" integer NOT NULL,
  "id" integer primary key autoincrement,
  "feed_version_id" integer NOT NULL,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(service_id) references gtfs_calendars(id)
);
CREATE INDEX idx_gtfs_calendar_dates_date ON "gtfs_calendar_dates"("date");
CREATE INDEX idx_gtfs_calendar_dates_exception_type ON "gtfs_calendar_dates"(exception_type);
CREATE INDEX idx_gtfs_calendar_dates_feed_version_id ON "gtfs_calendar_dates"(feed_version_id);
CREATE INDEX idx_gtfs_calendar_dates_service_id ON "gtfs_calendar_dates"(service_id);
CREATE TABLE IF NOT EXISTS "gtfs_routes" (
  "route_id" varchar(255),
  "agency_id" integer NOT NULL,
  "route_short_name" varchar(255),
  "route_long_name" varchar(255),
  "route_desc" varchar(255),
  "route_type" integer,
  "route_url" varchar(255),
  "route_color" varchar(255),
  "route_text_color" varchar(255),
  "route_sort_order" integer,
  "continuous_pickup" integer,
  "continuous_drop_off" integer,
  "id" integer primary key autoincrement,
  "feed_version_id" integer NOT NULL,
  "network_id" varchar(255),
  "as_route" integer,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "textsearch" TEXT,
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(agency_id) references gtfs_agencies(id)
);
CREATE INDEX idx_gtfs_routes_route_id ON "gtfs_routes"(route_id);
CREATE INDEX idx_gtfs_routes_agency_id ON "gtfs_routes"(agency_id);
CREATE INDEX idx_gtfs_routes_route_type ON "gtfs_routes"(route_type);
CREATE INDEX idx_gtfs_routes_feed_version_id ON "gtfs_routes"(feed_version_id);
CREATE TABLE IF NOT EXISTS "gtfs_stop_times" (
  "trip_id" integer NOT NULL,
  "arrival_time" int,
  "departure_time" int,
  "stop_id" integer NOT NULL,
  "stop_sequence" integer,
  "stop_headsign" varchar(255),
  "pickup_type" integer,
  "drop_off_type" integer,
  "shape_dist_traveled" real,
  "timepoint" integer,
  "continuous_pickup" integer,
  "continuous_drop_off" integer,
  "interpolated" integer,
  "feed_version_id" integer NOT NULL,
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(trip_id) references gtfs_trips(id),
  foreign key(stop_id) references gtfs_stops(id)
);
CREATE INDEX idx_stop_times_trip_id ON "gtfs_stop_times"(trip_id);
CREATE INDEX idx_gtfs_stop_times_stop_id ON "gtfs_stop_times"(stop_id);
CREATE INDEX idx_gtfs_stop_times_feed_version_id ON "gtfs_stop_times"(feed_version_id);
CREATE TABLE IF NOT EXISTS "gtfs_fare_rules" (
  "fare_id" integer NOT NULL,
  "route_id" int,
  "origin_id" varchar(255),
  "destination_id" varchar(255),
  "contains_id" varchar(255),
  "id" integer primary key autoincrement,
  "feed_version_id" integer NOT NULL,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(fare_id) references gtfs_fare_attributes(id)
);
CREATE INDEX idx_gtfs_fare_rules_fare_id ON "gtfs_fare_rules"(fare_id);
CREATE INDEX idx_gtfs_fare_rules_feed_version_id ON "gtfs_fare_rules"(feed_version_id);
CREATE TABLE IF NOT EXISTS "gtfs_fare_attributes" (
  "fare_id" varchar(255),
  "price" real,
  "currency_type" varchar(255),
  "payment_method" integer,
  "transfers" varchar(255),
  "agency_id" int,
  "transfer_duration" integer,
  "id" integer primary key autoincrement,
  "feed_version_id" integer NOT NULL,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);
CREATE INDEX idx_gtfs_fare_attributes_fare_id ON "gtfs_fare_attributes"(fare_id);
CREATE INDEX idx_gtfs_fare_attributes_feed_version_id ON "gtfs_fare_attributes"(feed_version_id);
-------------------
CREATE TABLE IF NOT EXISTS "tl_agency_onestop_ids" (
  "id" integer primary key autoincrement,
  "feed_version_id" integer not null,
  "agency_id" integer not null,
  "onestop_id" varchar(255),
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(agency_id) references gtfs_agencies(id)
);
CREATE TABLE IF NOT EXISTS "tl_stop_onestop_ids" (
  "id" integer primary key autoincrement,
  "feed_version_id" integer not null,
  "stop_id" integer not null,
  "onestop_id" varchar(255),
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(stop_id) references gtfs_stops(id)
);
CREATE TABLE IF NOT EXISTS "tl_route_onestop_ids" (
  "id" integer primary key autoincrement,
  "feed_version_id" integer not null,
  "route_id" integer not null,
  "onestop_id" varchar(255),
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(route_id) references gtfs_routes(id)
);
CREATE TABLE IF NOT EXISTS "tl_agency_places" (
  "id" integer primary key autoincrement,
  "feed_version_id" integer not null,
  "agency_id" integer not null,
  "rank" real not null,
  "name" varchar(255),
  "adm1name" varchar(255),
  "adm0name" varchar(255),
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(agency_id) references gtfs_agencies(id)
);
CREATE TABLE IF NOT EXISTS "tl_route_stops" (
  "id" integer primary key autoincrement,
  "feed_version_id" integer not null,
  "agency_id" integer not null,
  "route_id" integer not null,
  "stop_id" integer not null,
  foreign key(feed_version_id) REFERENCES feed_versions(id) foreign key(agency_id) references gtfs_agencies(id),
  foreign key(stop_id) references gtfs_stops(id),
  foreign key(route_id) references gtfs_routes(id)
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
  "departures" blob,
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(route_id) references gtfs_routes(id),
  foreign key(selected_stop_id) references gtfs_stops(id)
);
CREATE TABLE IF NOT EXISTS "tl_feed_version_geometries" (
  "id" integer primary key autoincrement,
  "feed_version_id" integer not null,
  "geometry" blob,
  "centroid" blob,
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);
CREATE TABLE IF NOT EXISTS "tl_agency_geometries" (
  "id" integer primary key autoincrement,
  "feed_version_id" integer not null,
  "agency_id" integer not null,
  "geometry" blob,
  "centroid" blob,
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(agency_id) references gtfs_agencies(id)
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
  "combined_geometry" blob,
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(route_id) references gtfs_routes(id)
);
---------------
CREATE TABLE IF NOT EXISTS "gtfs_translations" (
  "id" integer primary key autoincrement,
  "feed_version_id" integer NOT NULL,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "table_name" varchar(255),
  "field_name" varchar(255),
  "field_value" varchar(255),
  "language" varchar(255),
  "translation" varchar(255),
  "record_id" varchar(255),
  "record_sub_id" varchar(255),
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);
CREATE INDEX idx_gtfs_translations_feed_version_id ON "gtfs_translations"(feed_version_id);
CREATE TABLE IF NOT EXISTS "gtfs_attributions" (
  "id" integer primary key autoincrement,
  "feed_version_id" integer NOT NULL,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
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
  "attribution_phone" varchar(255),
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(agency_id) references gtfs_agencies(id),
  foreign key(trip_id) references gtfs_trips(id)
);
CREATE INDEX idx_gtfs_attributions_feed_version_id ON "gtfs_attributions"(feed_version_id);
CREATE TABLE tl_stop_external_references (
  "id" integer primary key autoincrement,
  "stop_id" integer not null,
  "feed_version_id" integer NOT NULL,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "target_feed_onestop_id" varchar(255),
  "target_stop_id" varchar(255),
  "inactive" bool,
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(stop_id) REFERENCES gtfs_stops(id)
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
  "response_time_ms" int,
  "response_ttfb_ms" int,
  "response_sha1" varchar(255),
  "validation_duration_ms" int,
  "upload_duration_ms" int,
  "feed_version_id" int,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);
----------------------
CREATE TABLE gtfs_areas (
  "id" integer primary key autoincrement,
  "feed_version_id" int not null,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "area_id" varchar(255) not null,
  "area_name" varchar(255),
  --- interline extensions
  "agency_ids" blob,
  "geometry" blob,
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);
CREATE TABLE gtfs_stop_areas (
  "id" integer primary key autoincrement,
  "feed_version_id" int not null,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "area_id" integer not null references gtfs_areas(id),
  "stop_id" integer not null references gtfs_stops(id),
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(area_id) references gtfs_areas(id),
  foreign key(stop_id) references gtfs_stops(id)
);
CREATE TABLE gtfs_fare_leg_rules (
  "id" integer primary key autoincrement,
  "feed_version_id" int not null,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "leg_group_id" varchar(255),
  "network_id" varchar(255),
  "from_area_id" varchar(255),
  "to_area_id" varchar(255),
  "fare_product_id" varchar(255),
  "transfer_only" integer,
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);
CREATE TABLE gtfs_fare_transfer_rules (
  "id" integer primary key autoincrement,
  "feed_version_id" int not null,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  from_leg_group_id varchar(255),
  to_leg_group_id varchar(255),
  transfer_count int,
  duration_limit int,
  duration_limit_type int,
  fare_transfer_type int,
  fare_product_id varchar(255),
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);
CREATE TABLE gtfs_fare_products (
  "id" integer primary key autoincrement,
  "feed_version_id" int not null,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
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
  duration_type int,
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);
CREATE TABLE gtfs_fare_media (
  "id" integer primary key autoincrement,
  "feed_version_id" int not null,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  --- interline extensions
  fare_media_id varchar(255),
  fare_media_name varchar(255),
  fare_media_type int,
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);
CREATE TABLE gtfs_rider_categories (
  "id" integer primary key autoincrement,
  "feed_version_id" int not null,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  --- interline extensions
  rider_category_id varchar(255),
  rider_category_name varchar(255),
  min_age int,
  max_age int,
  is_default_fare_category int,
  eligibility_url varchar(255),
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);

CREATE TABLE gtfs_timeframes (
  "id" integer primary key autoincrement,
  "feed_version_id" int not null,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  timeframe_group_id varchar(255),
  start_time int,
  end_time int,
  service_id int,
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(service_id) REFERENCES gtfs_calendars(id)
);

CREATE TABLE gtfs_networks (
  "id" integer primary key autoincrement,
  "feed_version_id" int not null,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  network_id varchar(255),
  network_name varchar(255),
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);

CREATE TABLE gtfs_route_networks (
  "id" integer primary key autoincrement,
  "feed_version_id" int not null,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  network_id int,
  route_id int,
  foreign key(feed_version_id) REFERENCES feed_versions(id),
  foreign key(network_id) REFERENCES gtfs_networks(id),
  foreign key(route_id) REFERENCES gtfs_routes(id)
);

CREATE TABLE tl_validation_reports (
  "id" integer primary key autoincrement,
  "feed_version_id" int not null,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP,
  "reported_at" datetime,
  "reported_at_local" datetime,
  "reported_at_local_timezone" varchar(255),
  "success" bool,
  "includes_static" bool,
  "includes_rt" bool,
  "validator" varchar(255),
  "validator_version" VARCHAR(255),
  "failure_reason" VARCHAR(255),
  "file" varchar(255),
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);
CREATE TABLE tl_validation_trip_update_stats (
  "id" integer primary key autoincrement,
  "validation_report_id" integer NOT NULL,
  "agency_id" varchar(255) NOT NULL,
  "route_id" varchar(255) NOT NULL,
  "trip_scheduled_ids" blob,
  "trip_scheduled_count" integer NOT NULL,
  "trip_match_count" integer NOT NULL,
  "trip_scheduled_not_matched" integer not null,
  "trip_rt_ids" blob,
  "trip_rt_count" integer not null,
  "trip_rt_matched" integer not null,
  "trip_rt_not_matched" integer not null,
  "trip_rt_not_found_ids" blob,
  "trip_rt_added_ids" blob,
  "trip_rt_not_found_count" integer not null,
  "trip_rt_added_count" integer not null,
  foreign key(validation_report_id) references tl_validation_reports(id)
);
CREATE TABLE tl_validation_vehicle_position_stats (
  "id" integer primary key autoincrement,
  "validation_report_id" integer NOT NULL,
  "agency_id" varchar(255) NOT NULL,
  "route_id" varchar(255) NOT NULL,
  "trip_scheduled_ids" blob,
  "trip_scheduled_count" integer NOT NULL,
  "trip_match_count" integer NOT NULL,
  "trip_scheduled_not_matched" integer not null,
  "trip_rt_ids" blob,
  "trip_rt_count" integer not null,
  "trip_rt_matched" integer not null,
  "trip_rt_not_matched" integer not null,
  "trip_rt_not_found_ids" blob,
  "trip_rt_added_ids" blob,
  "trip_rt_not_found_count" integer not null,
  "trip_rt_added_count" integer not null,
  foreign key(validation_report_id) references tl_validation_reports(id)
);
CREATE TABLE tl_validation_report_error_groups (
  "id" integer primary key autoincrement,
  "validation_report_id" integer NOT NULL,
  "filename" varchar(255) not null,
  "field" varchar(255) not null,
  "error_type" varchar(255) not null,
  "error_code" varchar(255) not null,
  "group_key" varchar(255) not null,
  "count" integer not null,
  "level" integer not null,
  foreign key(validation_report_id) references tl_validation_reports(id)
);
CREATE TABLE tl_validation_report_error_exemplars (
  "id" integer primary key autoincrement,
  "validation_report_error_group_id" integer not null,
  "line" integer not null,
  "entity_id" varchar(255) not null,
  "value" varchar(255) not null,
  "message" varchar(255) not null,
  "geometry" blob,
  "entity_json" blob,
  foreign key(validation_report_error_group_id) references tl_valdiation_report_error_groups(id)
);
CREATE TABLE feed_version_agency_onestop_ids (
  "feed_version_id" integer not null,
  "entity_id" varchar(255) not null,
  "onestop_id" varchar(255) not null,
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);
CREATE TABLE feed_version_route_onestop_ids (
  "feed_version_id" integer not null,
  "entity_id" varchar(255) not null,
  "onestop_id" varchar(255) not null,
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);
CREATE TABLE feed_version_stop_onestop_ids (
  "feed_version_id" integer not null,
  "entity_id" varchar(255) not null,
  "onestop_id" varchar(255) not null,
  foreign key(feed_version_id) REFERENCES feed_versions(id)
);

-- Materialized active routes table
CREATE TABLE tl_materialized_active_routes (
    -- Primary route data
    id INTEGER NOT NULL,
    route_id TEXT NOT NULL,
    route_short_name TEXT,
    route_long_name TEXT,
    route_desc TEXT,
    route_type INTEGER,
    route_url TEXT,
    route_color TEXT,
    route_text_color TEXT,
    route_sort_order INTEGER,
    agency_id INTEGER,
    network_id TEXT,
    as_route INTEGER,
    continuous_pickup INTEGER,
    continuous_drop_off INTEGER,
    feed_version_id INTEGER NOT NULL,

    -- Derived
    gtfs_agency_id TEXT,
    agency_name TEXT,
    feed_id INTEGER NOT NULL,
    onestop_id TEXT,
    
    -- Materialization metadata
    materialized_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,

    -- Search optimization
    textsearch TEXT,
    
    -- Spatial column (full geometry in SQLite, simplified in PostGIS)
    geometry_simplified BLOB,
    
    FOREIGN KEY (feed_id) REFERENCES current_feeds(id),
    FOREIGN KEY (feed_version_id) REFERENCES feed_versions(id)
);

-- Indexes for tl_materialized_active_routes
CREATE UNIQUE INDEX tl_materialized_active_routes_id_idx ON tl_materialized_active_routes(id);
CREATE INDEX tl_materialized_active_routes_route_id_idx ON tl_materialized_active_routes(route_id);
CREATE INDEX tl_materialized_active_routes_feed_version_id_idx ON tl_materialized_active_routes(feed_version_id);
CREATE INDEX tl_materialized_active_routes_onestop_id_idx ON tl_materialized_active_routes(onestop_id);

-- Materialized active stops table
CREATE TABLE tl_materialized_active_stops (
    -- Primary stop data
    id INTEGER NOT NULL,
    stop_id TEXT NOT NULL,
    stop_code TEXT,
    stop_name TEXT,
    stop_desc TEXT,
    zone_id TEXT,
    stop_url TEXT,
    location_type INTEGER,
    stop_timezone TEXT,
    parent_station INTEGER,
    level_id INTEGER,
    platform_code TEXT,
    tts_stop_name TEXT,
    area_id TEXT,
    wheelchair_boarding INTEGER,
    geometry BLOB,
    
    -- Derived
    feed_version_id INTEGER NOT NULL,
    feed_id INTEGER NOT NULL,
    onestop_id TEXT,
    
    -- Materialization metadata
    materialized_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    
    -- Search optimization
    textsearch TEXT,
    
    FOREIGN KEY (feed_id) REFERENCES current_feeds(id),
    FOREIGN KEY (feed_version_id) REFERENCES feed_versions(id)
);

-- Indexes for tl_materialized_active_stops
CREATE UNIQUE INDEX tl_materialized_active_stops_id_idx ON tl_materialized_active_stops(id);
CREATE INDEX tl_materialized_active_stops_stop_id_idx ON tl_materialized_active_stops(stop_id);
CREATE INDEX tl_materialized_active_stops_feed_version_id_idx ON tl_materialized_active_stops(feed_version_id);
CREATE INDEX tl_materialized_active_stops_onestop_id_idx ON tl_materialized_active_stops(onestop_id);

-- Materialized active agencies table
CREATE TABLE tl_materialized_active_agencies (
    -- Primary agency data
    id INTEGER NOT NULL,
    agency_id TEXT NOT NULL,
    agency_name TEXT,
    agency_url TEXT,
    agency_timezone TEXT,
    agency_lang TEXT,
    agency_phone TEXT,
    agency_fare_url TEXT,
    agency_email TEXT,
    
    -- Feed metadata
    feed_version_id INTEGER NOT NULL,
    feed_id INTEGER NOT NULL,
    onestop_id TEXT,
    
    -- Materialization metadata
    materialized_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    
    -- Search optimization
    textsearch TEXT,
    
    FOREIGN KEY (feed_id) REFERENCES current_feeds(id),
    FOREIGN KEY (feed_version_id) REFERENCES feed_versions(id)
);

-- Indexes for tl_materialized_active_agencies
CREATE UNIQUE INDEX tl_materialized_active_agencies_id_idx ON tl_materialized_active_agencies(id);
CREATE INDEX tl_materialized_active_agencies_agency_id_idx ON tl_materialized_active_agencies(agency_id);
CREATE INDEX tl_materialized_active_agencies_feed_version_id_idx ON tl_materialized_active_agencies(feed_version_id);
CREATE INDEX tl_materialized_active_agencies_onestop_id_idx ON tl_materialized_active_agencies(onestop_id);

-- Job runs table: optional context for artifacts, tracks workflow execution
CREATE TABLE IF NOT EXISTS "job_runs" (
  "id" integer primary key autoincrement,
  "job_type" text NOT NULL,
  "status" text NOT NULL CHECK (status IN ('pending', 'running', 'success', 'failed', 'cancelled')),
  "started_at" datetime,
  "completed_at" datetime,
  "metadata" BLOB NOT NULL DEFAULT '{}',
  "metrics" BLOB NOT NULL DEFAULT '{}',
  "log_summary" text,
  "error_message" text,
  "created_by" text,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,
  "updated_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Artifacts table: tracks files, reports, and analysis results
CREATE TABLE IF NOT EXISTS "artifacts" (
  "id" integer primary key autoincrement,
  "name" text NOT NULL,
  "artifact_type" text NOT NULL,
  "storage_type" text NOT NULL CHECK (storage_type IN ('inline', 's3', 'azure')),
  "inline_json_data" BLOB,  -- For inline storage of small artifacts (JSONB/BLOB for structured data)
  "storage_url" text,  -- Full storage URL: s3://bucket.s3.region.amazonaws.com/path or az://account/container/path
  "content_type" text,
  "size_bytes" integer,
  "metadata" BLOB NOT NULL DEFAULT '{}',
  "job_run_id" integer,  -- Optional: artifact belongs to at most one job run
  "created_by" text,
  "created_at" datetime DEFAULT CURRENT_TIMESTAMP NOT NULL,
  FOREIGN KEY (job_run_id) REFERENCES job_runs(id) ON DELETE SET NULL
);

-- Link artifacts to feed versions for lineage tracking
CREATE TABLE IF NOT EXISTS "artifacts_feed_versions" (
  "artifact_id" integer NOT NULL,
  "feed_version_id" integer NOT NULL,
  "relationship_type" text NOT NULL CHECK (relationship_type IN ('input', 'output')),
  PRIMARY KEY (artifact_id, feed_version_id, relationship_type),
  FOREIGN KEY (artifact_id) REFERENCES artifacts(id) ON DELETE CASCADE,
  FOREIGN KEY (feed_version_id) REFERENCES feed_versions(id) ON DELETE CASCADE
);

-- Indexes for artifacts
CREATE INDEX idx_artifacts_artifact_type ON "artifacts"(artifact_type, created_at DESC);
CREATE INDEX idx_artifacts_created_by ON "artifacts"(created_by, created_at DESC);
CREATE INDEX idx_artifacts_storage_url ON "artifacts"(storage_url);
CREATE INDEX idx_artifacts_job_run_id ON "artifacts"(job_run_id);

-- Indexes for job_runs
CREATE INDEX idx_job_runs_status ON "job_runs"(status, created_at DESC);
CREATE INDEX idx_job_runs_job_type ON "job_runs"(job_type, created_at DESC);
CREATE INDEX idx_job_runs_created_by ON "job_runs"(created_by, created_at DESC);

-- Indexes for join tables
CREATE INDEX idx_artifacts_feed_versions_artifact_id ON "artifacts_feed_versions"(artifact_id);
CREATE INDEX idx_artifacts_feed_versions_feed_version_id ON "artifacts_feed_versions"(feed_version_id);