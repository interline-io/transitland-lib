DROP TABLE IF EXISTS ext_plus_calendar_attributes;
CREATE TABLE ext_plus_calendar_attributes (
    id serial primary key,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL REFERENCES feed_versions(id),
    service_id text not null,
    service_description text not null
);


DROP TABLE IF EXISTS ext_plus_directions;
CREATE TABLE ext_plus_directions (
    id serial primary key,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL REFERENCES feed_versions(id),
    route_id text not null,
    direction_id text not null,
    direction text not null
);


DROP TABLE IF EXISTS ext_plus_fare_rider_categories;
CREATE TABLE ext_plus_fare_rider_categories (
    id serial primary key,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL REFERENCES feed_versions(id),
    fare_id text not null,
    rider_category_id integer not null,
    price text not null,
    expiration_date date not null, 
    commencement_date date not null
);


DROP TABLE IF EXISTS ext_plus_farezone_attributes;
CREATE TABLE ext_plus_farezone_attributes (
    id serial primary key,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL REFERENCES feed_versions(id),
    zone_id text not null,
    zone_name text not null
);


DROP TABLE IF EXISTS ext_plus_realtime_routes;
CREATE TABLE ext_plus_realtime_routes (
    id serial primary key,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL REFERENCES feed_versions(id),
    route_id text not null,
    realtime_enabled integer not null
);


DROP TABLE IF EXISTS ext_plus_realtime_stops;
CREATE TABLE ext_plus_realtime_stops (
    id serial primary key,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL REFERENCES feed_versions(id),
    trip_id text not null,
    stop_id text not null,
    realtime_stop_id text not null
);



DROP TABLE IF EXISTS ext_plus_realtime_trips;
CREATE TABLE ext_plus_realtime_trips (
    id serial primary key,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL REFERENCES feed_versions(id),
    trip_id text not null,
    realtime_trip_id text not null
);



DROP TABLE IF EXISTS ext_plus_rider_categories;
CREATE TABLE ext_plus_rider_categories (
    id serial primary key,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL REFERENCES feed_versions(id),
    agency_id text not null,
    rider_category_id integer not null,
    rider_category_description text not null
);


DROP TABLE IF EXISTS ext_plus_stop_attributes;
CREATE TABLE ext_plus_stop_attributes (
    id serial primary key,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL REFERENCES feed_versions(id),
    stop_id text not null,
    accessibility_id integer not null,
    cardinal_direction text not null,
    relative_position text not null,
    stop_city text not null
);


DROP TABLE IF EXISTS ext_plus_timepoints;
CREATE TABLE ext_plus_timepoints (
    id serial primary key,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL REFERENCES feed_versions(id),
    trip_id text not null,
    stop_id text not null
);
