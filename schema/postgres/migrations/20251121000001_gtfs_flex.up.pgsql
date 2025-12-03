BEGIN;

-- GTFS-Flex: location_groups.txt
CREATE TABLE public.gtfs_location_groups (
    id bigserial primary key not null,
    feed_version_id bigint REFERENCES feed_versions(id) not null,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    
    location_group_id text NOT NULL,
    location_group_name text
);
CREATE INDEX ON gtfs_location_groups(feed_version_id);
CREATE UNIQUE INDEX ON gtfs_location_groups(feed_version_id, location_group_id);


-- GTFS-Flex: location_group_stops.txt
CREATE TABLE public.gtfs_location_group_stops (
    id bigserial primary key not null,
    feed_version_id bigint REFERENCES feed_versions(id) not null,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    
    location_group_id bigint not null REFERENCES gtfs_location_groups(id),
    stop_id bigint not null REFERENCES gtfs_stops(id)
);
CREATE INDEX ON gtfs_location_group_stops(feed_version_id);
CREATE INDEX ON gtfs_location_group_stops(location_group_id);
CREATE INDEX ON gtfs_location_group_stops(stop_id);
CREATE UNIQUE INDEX ON gtfs_location_group_stops(location_group_id, stop_id);


-- GTFS-Flex: booking_rules.txt
CREATE TABLE public.gtfs_booking_rules (
    id bigserial primary key not null,
    feed_version_id bigint REFERENCES feed_versions(id) not null,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    
    booking_rule_id text NOT NULL,
    booking_type integer NOT NULL,
    prior_notice_duration_min integer,
    prior_notice_duration_max integer,
    prior_notice_last_day integer,
    prior_notice_last_time integer,
    prior_notice_start_day integer,
    prior_notice_start_time integer,
    prior_notice_service_id text,
    message text,
    pickup_message text,
    drop_off_message text,
    phone_number text,
    info_url text,
    booking_url text
);
CREATE INDEX ON gtfs_booking_rules(feed_version_id);
CREATE UNIQUE INDEX ON gtfs_booking_rules(feed_version_id, booking_rule_id);


-- GTFS-Flex: locations.geojson
CREATE TABLE public.gtfs_locations (
    id bigserial primary key not null,
    feed_version_id bigint REFERENCES feed_versions(id) not null,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    
    location_id text NOT NULL,
    stop_name text,
    stop_desc text,
    zone_id text,
    stop_url text,
    -- Geometry can be Polygon or MultiPolygon
    geometry public.geography(Geometry,4326)
);
CREATE INDEX ON gtfs_locations(feed_version_id);
CREATE UNIQUE INDEX ON gtfs_locations(feed_version_id, location_id);
CREATE INDEX ON gtfs_locations USING GIST(geometry);


-- GTFS-Flex: Add new fields to stop_times
ALTER TABLE public.gtfs_stop_times 
    ADD COLUMN location_group_id bigint REFERENCES gtfs_location_groups(id),
    ADD COLUMN location_id bigint REFERENCES gtfs_locations(id),
    ADD COLUMN start_pickup_drop_off_window integer,
    ADD COLUMN end_pickup_drop_off_window integer,
    ADD COLUMN pickup_booking_rule_id bigint REFERENCES gtfs_booking_rules(id),
    ADD COLUMN drop_off_booking_rule_id bigint REFERENCES gtfs_booking_rules(id),
    ADD COLUMN mean_duration_factor double precision,
    ADD COLUMN mean_duration_offset double precision,
    ADD COLUMN safe_duration_factor double precision,
    ADD COLUMN safe_duration_offset double precision;

-- GTFS-Flex: Drop not-null constraint on stop_id (now conditionally required)
ALTER TABLE public.gtfs_stop_times 
    ALTER COLUMN stop_id DROP NOT NULL;

COMMIT;


