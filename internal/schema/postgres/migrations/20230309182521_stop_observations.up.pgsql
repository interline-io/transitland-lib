CREATE TABLE ext_performance_stop_observations (
    id bigserial primary key,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL,
    trip_id text,
    route_id text,
    agency_id text,
    direction_id integer,
    trip_start_time integer,
    trip_start_date date,
    schedule_relationship text,
    vehicle_id text,
    source text,
    stop_sequence integer,
    observed_arrival_time integer,
    observed_departure_time integer,
    uncertainty integer,
    dwell_time_secs integer,
    scheduled_dwell_time_secs integer,
    occupancy_status int,
    occupancy_percentage int,
    observed_arrival_delay int,
    scheduled_arrival_time int,
    scheduled_departure_time int,
    from_stop_id text,
    to_stop_id text,
    distance float,
    duration float,
    speed_mph float,
    build_id text,
    source_id text
);

CREATE INDEX ON ext_performance_stop_observations(build_id);
CREATE INDEX ON ext_performance_stop_observations(feed_version_id);
CREATE INDEX ON ext_performance_stop_observations(trip_id);
CREATE INDEX ON ext_performance_stop_observations(trip_start_date,trip_id);
CREATE INDEX ON ext_performance_stop_observations(from_stop_id);
CREATE INDEX ON ext_performance_stop_observations(to_stop_id);
CREATE INDEX ON ext_performance_stop_observations(source);
