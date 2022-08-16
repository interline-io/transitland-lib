BEGIN;

CREATE TABLE public.gtfs_areas (
    id bigserial primary key not null,
    feed_version_id bigint REFERENCES feed_versions(id),
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,

    area_id text NOT NULL,
    area_name text,

    --- interline extensions
    agency_ids jsonb,
    geometry public.geography(Polygon,4326)
);
CREATE INDEX ON gtfs_areas(feed_version_id);
CREATE UNIQUE INDEX ON gtfs_areas(feed_version_id,area_id);


CREATE TABLE public.gtfs_stop_areas (
    id bigserial primary key not null,
    feed_version_id bigint REFERENCES feed_versions(id),
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,

    area_id bigint not null REFERENCES gtfs_areas(id),
    stop_id bigint not null REFERENCES gtfs_stops(id)
);
CREATE INDEX ON gtfs_stop_areas(feed_version_id);
CREATE INDEX ON gtfs_stop_areas(area_id);
CREATE INDEX ON gtfs_stop_areas(stop_id);
CREATE UNIQUE INDEX ON gtfs_stop_areas(area_id,stop_id);


CREATE TABLE public.gtfs_fare_leg_rules (
    id bigserial primary key not null,
    feed_version_id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,

    leg_group_id text,
    network_id text,
    from_area_id text,
    to_area_id text,
    fare_product_id text,
    
    -- interline extension
    transfer_only int 
);
CREATE INDEX ON gtfs_fare_leg_rules(feed_version_id);
CREATE UNIQUE INDEX ON gtfs_fare_leg_rules(feed_version_id,network_id,from_area_id,to_area_id,fare_product_id);


CREATE TABLE public.gtfs_fare_transfer_rules (
    id bigserial primary key not null,
    feed_version_id bigint REFERENCES feed_versions(id),
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,

    from_leg_group_id text,
    to_leg_group_id text,
    transfer_count integer,
    duration_limit integer,
    duration_limit_type integer,
    fare_transfer_type integer,
    fare_product_id text,

    -- interline extension
    filter_fare_product_id text
);
CREATE INDEX ON gtfs_fare_transfer_rules(feed_version_id);
CREATE UNIQUE INDEX ON gtfs_fare_transfer_rules(feed_version_id, from_leg_group_id, to_leg_group_id, fare_product_id, transfer_count, duration_limit);


CREATE TABLE public.gtfs_fare_products (
    id bigserial primary key not null,
    feed_version_id bigint REFERENCES feed_versions(id),
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,

    fare_product_id text,
    fare_product_name text,
    amount double precision,
    currency text,

    --- interline extensions
    rider_category_id text,
    fare_container_id text,
    duration_start integer,
    duration_amount double precision,
    duration_unit integer,
    duration_type integer
);
CREATE INDEX ON gtfs_fare_products(feed_version_id);
CREATE UNIQUE INDEX ON gtfs_fare_products(feed_version_id,fare_product_id,rider_category_id,fare_container_id);


CREATE TABLE public.gtfs_fare_containers (
    id bigserial primary key not null,
    feed_version_id bigint REFERENCES feed_versions(id),
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,

    --- interline extensions
    fare_container_id text NOT NULL,
    fare_container_name text,
    minimum_initial_purchase double precision,
    amount double precision,
    currency text
);
CREATE INDEX ON gtfs_fare_containers(feed_version_id);
CREATE UNIQUE INDEX ON gtfs_fare_containers(feed_version_id,fare_container_id);

CREATE TABLE public.gtfs_rider_categories (
    id bigserial primary key not null,
    feed_version_id bigint REFERENCES feed_versions(id),
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,

    --- interline extensions
    rider_category_id text NOT NULL,
    rider_category_name text NOT NULL,
    min_age integer,
    max_age integer,
    eligibility_url text
);
CREATE INDEX ON gtfs_rider_categories(feed_version_id);
-- CREATE UNIQUE INDEX ON gtfs_rider_categories(feed_version_id,rider_category_id);

COMMIT;