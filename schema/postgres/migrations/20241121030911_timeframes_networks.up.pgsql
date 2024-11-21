BEGIN;
CREATE TABLE gtfs_timeframes (
    id bigserial primary key NOT NULL,
    feed_version_id bigint REFERENCES feed_versions(id) NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    timeframe_group_id text not null,
    start_time int,
    end_time int,
    service_id bigint references gtfs_calendars(id)
);
CREATE INDEX ON gtfs_timeframes(feed_version_id);
CREATE INDEX ON gtfs_timeframes(timeframe_group_id);
CREATE TABLE gtfs_networks (
    id bigserial primary key NOT NULL,
    feed_version_id bigint REFERENCES feed_versions(id) NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    network_id text not null,
    network_name text
);
CREATE INDEX ON gtfs_networks(feed_version_id);
CREATE INDEX ON gtfs_networks(network_id);
CREATE TABLE gtfs_route_networks (
    id bigserial primary key NOT NULL,
    feed_version_id bigint REFERENCES feed_versions(id) NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    network_id bigint references gtfs_networks(id) not null,
    route_id bigint references gtfs_routes(id) not null
);
CREATE INDEX ON gtfs_route_networks(feed_version_id);
CREATE INDEX ON gtfs_route_networks(network_id);
CREATE INDEX ON gtfs_route_networks(route_id);
COMMIT;