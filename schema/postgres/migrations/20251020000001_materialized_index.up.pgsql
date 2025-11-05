BEGIN;

-- Materialized active routes table
CREATE TABLE tl_materialized_active_routes (
    -- Primary route data
    id bigint NOT NULL,
    route_id text NOT NULL,
    route_short_name text,
    route_long_name text,
    route_desc text,
    route_type integer,
    route_url text,
    route_color text,
    route_text_color text,
    route_sort_order int,
    agency_id bigint,
    network_id text,
    as_route integer,
    continuous_pickup integer,
    continuous_drop_off integer,
    feed_version_id bigint NOT NULL,

    -- Derived
    gtfs_agency_id text,
    agency_name text,
    feed_id bigint NOT NULL,
    onestop_id text,
    
    -- Materialization metadata
    materialized_at timestamp without time zone DEFAULT now() NOT NULL,

    -- Search optimization
    textsearch tsvector,
    
    FOREIGN KEY (feed_id) REFERENCES current_feeds(id),
    FOREIGN KEY (feed_version_id) REFERENCES feed_versions(id)
);

-- Indexes for tl_materialized_active_routes
CREATE UNIQUE INDEX ON tl_materialized_active_routes(id);
CREATE INDEX ON tl_materialized_active_routes(route_id);
CREATE INDEX ON tl_materialized_active_routes(feed_version_id);
CREATE INDEX ON tl_materialized_active_routes(onestop_id);
CREATE INDEX ON tl_materialized_active_routes USING gin(textsearch);

-- Materialized active stops table
CREATE TABLE tl_materialized_active_stops (
    -- Primary stop data
    id bigint NOT NULL,
    stop_id text NOT NULL,
    stop_code text,
    stop_name text,
    stop_desc text,
    zone_id text,
    stop_url text,
    location_type integer,
    stop_timezone text,
    parent_station bigint,
    level_id bigint,
    platform_code text,
    tts_stop_name text,
    area_id text,
    wheelchair_boarding integer,
    geometry geography(Point,4326),
    
    -- Derived
    feed_version_id bigint NOT NULL,
    feed_id bigint NOT NULL,
    onestop_id text,
    
    -- Materialization metadata
    materialized_at timestamp without time zone DEFAULT now() NOT NULL,
    
    -- Search optimization
    textsearch tsvector,
    
    FOREIGN KEY (feed_id) REFERENCES current_feeds(id),
    FOREIGN KEY (feed_version_id) REFERENCES feed_versions(id)
);


-- Indexes for tl_materialized_active_stops
CREATE UNIQUE INDEX ON tl_materialized_active_stops(id);
CREATE INDEX ON tl_materialized_active_stops(stop_id);
CREATE INDEX ON tl_materialized_active_stops(feed_version_id);
CREATE INDEX ON tl_materialized_active_stops(onestop_id);
CREATE INDEX ON tl_materialized_active_stops USING gin(textsearch);
CREATE INDEX ON tl_materialized_active_stops USING gist(geometry);

-- Materialized active agencies table
CREATE TABLE tl_materialized_active_agencies (
    -- Primary agency data
    id bigint NOT NULL,
    agency_id text NOT NULL,
    agency_name text,
    agency_url text,
    agency_timezone text,
    agency_lang text,
    agency_phone text,
    agency_fare_url text,
    agency_email text,
    
    -- Feed metadata
    feed_version_id bigint NOT NULL,
    feed_id bigint NOT NULL,
    onestop_id text,
    
    -- Materialization metadata
    materialized_at timestamp without time zone DEFAULT now() NOT NULL,
    
    -- Search optimization
    textsearch tsvector,
    
    FOREIGN KEY (feed_id) REFERENCES current_feeds(id),
    FOREIGN KEY (feed_version_id) REFERENCES feed_versions(id)
);

-- Indexes for tl_materialized_active_agencies
CREATE UNIQUE INDEX ON tl_materialized_active_agencies(id);
CREATE INDEX ON tl_materialized_active_agencies(agency_id);
CREATE INDEX ON tl_materialized_active_agencies(feed_version_id);
CREATE INDEX ON tl_materialized_active_agencies(onestop_id);
CREATE INDEX ON tl_materialized_active_agencies USING gin(textsearch);

COMMIT;