CREATE EXTENSION postgis; CREATE EXTENSION hstore;
CREATE EXTENSION pg_trgm; CREATE EXTENSION unaccent; CREATE TEXT SEARCH CONFIGURATION tl ( COPY = simple ); ALTER TEXT SEARCH CONFIGURATION tl ALTER MAPPING FOR hword, hword_part, word WITH unaccent;
CREATE TABLE public.gtfs_calendars (
    id bigint NOT NULL,
    service_id character varying NOT NULL,
    monday integer NOT NULL,
    tuesday integer NOT NULL,
    wednesday integer NOT NULL,
    thursday integer NOT NULL,
    friday integer NOT NULL,
    saturday integer NOT NULL,
    sunday integer NOT NULL,
    start_date date NOT NULL,
    end_date date NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL,
    generated boolean NOT NULL
);
CREATE TABLE public.feed_versions (
    id bigint NOT NULL,
    feed_id bigint NOT NULL,
    feed_type character varying DEFAULT 'gtfs'::character varying NOT NULL,
    file character varying DEFAULT ''::character varying NOT NULL,
    earliest_calendar_date date NOT NULL,
    latest_calendar_date date NOT NULL,
    sha1 character varying NOT NULL,
    md5 character varying,
    tags public.hstore,
    fetched_at timestamp without time zone NOT NULL,
    imported_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    import_level integer DEFAULT 0 NOT NULL,
    url character varying DEFAULT ''::character varying NOT NULL,
    file_raw character varying,
    sha1_raw character varying,
    md5_raw character varying,
    file_feedvalidator character varying,
    deleted_at timestamp without time zone,
    sha1_dir character varying,
    name text,
    description text,
    created_by text,
    updated_by text
);
CREATE TABLE public.tl_census_values (
    geography_id bigint NOT NULL,
    table_id bigint NOT NULL,
    table_values jsonb DEFAULT '{}'::jsonb NOT NULL
);
CREATE TABLE public.gtfs_stop_times (
    feed_version_id bigint NOT NULL,
    trip_id bigint NOT NULL,
    stop_id bigint NOT NULL,
    arrival_time integer NOT NULL,
    departure_time integer NOT NULL,
    stop_sequence integer NOT NULL,
    shape_dist_traveled double precision,
    pickup_type smallint,
    drop_off_type smallint,
    timepoint smallint,
    interpolated smallint,
    stop_headsign text
)
PARTITION BY HASH (feed_version_id);
CREATE TABLE public.tl_agency_places (
    id bigint NOT NULL,
    feed_version_id bigint NOT NULL,
    agency_id bigint NOT NULL,
    count integer NOT NULL,
    rank double precision NOT NULL,
    name character varying,
    adm1name character varying,
    adm0name character varying,
    best_match boolean,
    best_match_type integer
);
CREATE SEQUENCE public.agency_places_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.agency_places_id_seq OWNED BY public.tl_agency_places.id;
CREATE TABLE public.current_feeds (
    id bigint NOT NULL,
    onestop_id character varying NOT NULL,
    url character varying,
    spec character varying DEFAULT 'gtfs'::character varying NOT NULL,
    tags public.hstore,
    last_fetched_at timestamp without time zone,
    last_imported_at timestamp without time zone,
    version integer,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    created_or_updated_in_changeset_id integer,
    geometry public.geography(Geometry,4326),
    active_feed_version_id integer,
    edited_attributes character varying[] DEFAULT '{}'::character varying[],
    name character varying,
    type character varying,
    auth jsonb DEFAULT '{}'::jsonb NOT NULL,
    urls jsonb DEFAULT '{}'::jsonb NOT NULL,
    deleted_at timestamp without time zone,
    last_successful_fetch_at timestamp without time zone,
    last_fetch_error character varying DEFAULT ''::character varying NOT NULL,
    license jsonb DEFAULT '{}'::jsonb NOT NULL,
    other_ids jsonb DEFAULT '{}'::jsonb NOT NULL,
    associated_feeds jsonb DEFAULT '[]'::jsonb NOT NULL,
    languages jsonb DEFAULT '[]'::jsonb NOT NULL,
    feed_namespace_id character varying DEFAULT ''::character varying NOT NULL,
    file character varying DEFAULT ''::character varying NOT NULL,
    textsearch tsvector GENERATED ALWAYS AS (((setweight(to_tsvector('public.tl'::regconfig, (onestop_id)::text), 'A'::"char") || setweight(to_tsvector('public.tl'::regconfig, (COALESCE(name, ''::character varying))::text), 'A'::"char")) || setweight(to_tsvector('public.tl'::regconfig, COALESCE((urls ->> 'static_current'::text), ''::text)), 'B'::"char"))) STORED,
    feed_tags jsonb
);
CREATE SEQUENCE public.current_feeds_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.current_feeds_id_seq OWNED BY public.current_feeds.id;
CREATE TABLE public.current_operators (
    id integer NOT NULL,
    name character varying,
    tags public.hstore,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    onestop_id character varying,
    geometry public.geography(Geometry,4326),
    created_or_updated_in_changeset_id integer,
    version integer,
    timezone character varying,
    short_name character varying,
    website character varying,
    country character varying,
    state character varying,
    metro character varying,
    edited_attributes character varying[] DEFAULT '{}'::character varying[],
    associated_feeds jsonb DEFAULT '{}'::jsonb NOT NULL,
    deleted_at timestamp without time zone,
    textsearch tsvector GENERATED ALWAYS AS (((setweight(to_tsvector('public.tl'::regconfig, (COALESCE(name, ''::character varying))::text), 'A'::"char") || setweight(to_tsvector('public.tl'::regconfig, (COALESCE(short_name, ''::character varying))::text), 'A'::"char")) || setweight(to_tsvector('public.tl'::regconfig, (COALESCE(onestop_id, ''::character varying))::text), 'A'::"char"))) STORED,
    operator_tags jsonb
);
CREATE SEQUENCE public.current_operators_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.current_operators_id_seq OWNED BY public.current_operators.id;
CREATE TABLE public.current_operators_in_feed (
    id integer NOT NULL,
    gtfs_agency_id character varying,
    version integer,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    operator_id integer,
    feed_id integer,
    created_or_updated_in_changeset_id integer
);
CREATE SEQUENCE public.current_operators_in_feed_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.current_operators_in_feed_id_seq OWNED BY public.current_operators_in_feed.id;
CREATE TABLE public.ext_faresv2_areas (
    area_id text NOT NULL,
    area_name text NOT NULL,
    greater_area_id text,
    geometry public.geography(Polygon,4326) NOT NULL,
    internal_notes text,
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL
);
CREATE SEQUENCE public.ext_faresv2_areas_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.ext_faresv2_areas_id_seq OWNED BY public.ext_faresv2_areas.id;
CREATE TABLE public.ext_faresv2_fare_capping (
    fare_product_id text,
    eligible_cap_id text,
    fare_container_id text,
    duration_amount double precision,
    duration_unit integer,
    duration_type integer,
    offset_amount integer,
    offset_unit integer,
    service_id text,
    timeframe_id text,
    area_id text,
    network_id text,
    cap_amount double precision,
    currency text,
    internal_notes text,
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL
);
CREATE SEQUENCE public.ext_faresv2_fare_capping_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.ext_faresv2_fare_capping_id_seq OWNED BY public.ext_faresv2_fare_capping.id;
CREATE TABLE public.ext_faresv2_fare_containers (
    fare_container_id text NOT NULL,
    fare_container_name text NOT NULL,
    minimum_initial_purchase double precision,
    amount double precision,
    currency text,
    rider_category_id text,
    internal_notes text,
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL
);
CREATE SEQUENCE public.ext_faresv2_fare_containers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.ext_faresv2_fare_containers_id_seq OWNED BY public.ext_faresv2_fare_containers.id;
CREATE TABLE public.ext_faresv2_fare_leg_rules (
    leg_group_id text NOT NULL,
    fare_leg_name text,
    network_id text,
    from_area_id text,
    to_area_id text,
    contains_area_id text,
    is_symmetrical integer,
    from_timeframe_id text,
    to_timeframe_id text,
    min_distance double precision,
    max_distance double precision,
    distance_type integer,
    service_id text,
    amount double precision,
    min_amount double precision,
    max_amount double precision,
    currency text,
    fare_product_id text,
    fare_container_id text,
    rider_category_id text,
    eligible_cap_id text,
    internal_notes text,
    generated boolean,
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL
);
CREATE SEQUENCE public.ext_faresv2_fare_leg_rules_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.ext_faresv2_fare_leg_rules_id_seq OWNED BY public.ext_faresv2_fare_leg_rules.id;
CREATE TABLE public.ext_faresv2_fare_products (
    fare_product_id text NOT NULL,
    fare_product_name text NOT NULL,
    rider_category_id text,
    fare_container_id text,
    bundle_amount integer,
    duration_start integer,
    duration_amount double precision,
    duration_unit integer,
    duration_type integer,
    offset_amount integer,
    offset_unit integer,
    service_id text,
    timeframe_id text,
    timeframe_type integer,
    cap_required integer,
    eligible_cap_id text,
    amount double precision,
    min_amount double precision,
    max_amount double precision,
    currency text,
    internal_notes text,
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL
);
CREATE SEQUENCE public.ext_faresv2_fare_products_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.ext_faresv2_fare_products_id_seq OWNED BY public.ext_faresv2_fare_products.id;
CREATE TABLE public.ext_faresv2_fare_timeframes (
    timeframe_id text NOT NULL,
    start_time integer NOT NULL,
    end_time integer NOT NULL,
    internal_notes text,
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL
);
CREATE SEQUENCE public.ext_faresv2_fare_timeframes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.ext_faresv2_fare_timeframes_id_seq OWNED BY public.ext_faresv2_fare_timeframes.id;
CREATE TABLE public.ext_faresv2_fare_transfer_rules (
    from_leg_group_id text,
    to_leg_group_id text,
    is_symmetrical integer,
    spanning_limit integer,
    transfer_id text,
    transfer_sequence integer,
    duration_limit integer,
    duration_limit_type integer,
    fare_transfer_type integer,
    amount double precision,
    min_amount double precision,
    max_amount double precision,
    currency text,
    fare_product_id text,
    fare_container_id text,
    rider_category_id text,
    eligible_cap_id text,
    internal_notes text,
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL
);
CREATE SEQUENCE public.ext_faresv2_fare_transfer_rules_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.ext_faresv2_fare_transfer_rules_id_seq OWNED BY public.ext_faresv2_fare_transfer_rules.id;
CREATE TABLE public.ext_faresv2_rider_categories (
    rider_category_id text NOT NULL,
    rider_category_name text NOT NULL,
    min_age integer,
    max_age integer,
    eligibility_url text,
    internal_notes text,
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL
);
CREATE SEQUENCE public.ext_faresv2_rider_categories_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.ext_faresv2_rider_categories_id_seq OWNED BY public.ext_faresv2_rider_categories.id;
CREATE TABLE public.ext_plus_calendar_attributes (
    id bigint NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id bigint NOT NULL,
    service_id bigint NOT NULL,
    service_description text NOT NULL
);
CREATE SEQUENCE public.ext_plus_calendar_attributes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.ext_plus_calendar_attributes_id_seq OWNED BY public.ext_plus_calendar_attributes.id;
CREATE TABLE public.ext_plus_directions (
    id bigint NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id bigint NOT NULL,
    route_id bigint NOT NULL,
    direction_id text NOT NULL,
    direction text NOT NULL
);
CREATE SEQUENCE public.ext_plus_directions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.ext_plus_directions_id_seq OWNED BY public.ext_plus_directions.id;
CREATE TABLE public.ext_plus_fare_rider_categories (
    id bigint NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id bigint NOT NULL,
    fare_id bigint NOT NULL,
    rider_category_id integer NOT NULL,
    price text NOT NULL,
    expiration_date date,
    commencement_date date
);
CREATE SEQUENCE public.ext_plus_fare_rider_categories_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.ext_plus_fare_rider_categories_id_seq OWNED BY public.ext_plus_fare_rider_categories.id;
CREATE TABLE public.ext_plus_farezone_attributes (
    id bigint NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id bigint NOT NULL,
    zone_id text NOT NULL,
    zone_name text NOT NULL
);
CREATE SEQUENCE public.ext_plus_farezone_attributes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.ext_plus_farezone_attributes_id_seq OWNED BY public.ext_plus_farezone_attributes.id;
CREATE TABLE public.ext_plus_realtime_routes (
    id bigint NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id bigint NOT NULL,
    route_id bigint NOT NULL,
    realtime_enabled integer NOT NULL
);
CREATE SEQUENCE public.ext_plus_realtime_routes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.ext_plus_realtime_routes_id_seq OWNED BY public.ext_plus_realtime_routes.id;
CREATE TABLE public.ext_plus_realtime_stops (
    id bigint NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id bigint NOT NULL,
    trip_id bigint NOT NULL,
    stop_id bigint NOT NULL,
    realtime_stop_id text NOT NULL
);
CREATE SEQUENCE public.ext_plus_realtime_stops_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.ext_plus_realtime_stops_id_seq OWNED BY public.ext_plus_realtime_stops.id;
CREATE TABLE public.ext_plus_realtime_trips (
    id bigint NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id bigint NOT NULL,
    trip_id bigint NOT NULL,
    realtime_trip_id text NOT NULL
);
CREATE SEQUENCE public.ext_plus_realtime_trips_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.ext_plus_realtime_trips_id_seq OWNED BY public.ext_plus_realtime_trips.id;
CREATE TABLE public.ext_plus_rider_categories (
    id bigint NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id bigint NOT NULL,
    agency_id bigint NOT NULL,
    rider_category_id integer NOT NULL,
    rider_category_description text NOT NULL
);
CREATE SEQUENCE public.ext_plus_rider_categories_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.ext_plus_rider_categories_id_seq OWNED BY public.ext_plus_rider_categories.id;
CREATE TABLE public.ext_plus_stop_attributes (
    id bigint NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id bigint NOT NULL,
    stop_id bigint NOT NULL,
    accessibility_id integer NOT NULL,
    cardinal_direction text NOT NULL,
    relative_position text NOT NULL,
    stop_city text NOT NULL
);
CREATE SEQUENCE public.ext_plus_stop_attributes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.ext_plus_stop_attributes_id_seq OWNED BY public.ext_plus_stop_attributes.id;
CREATE TABLE public.ext_plus_timepoints (
    id bigint NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id bigint NOT NULL,
    trip_id bigint NOT NULL,
    stop_id bigint NOT NULL
);
CREATE SEQUENCE public.ext_plus_timepoints_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.ext_plus_timepoints_id_seq OWNED BY public.ext_plus_timepoints.id;
CREATE TABLE public.feed_states (
    id bigint NOT NULL,
    feed_id bigint NOT NULL,
    feed_version_id bigint,
    last_fetched_at timestamp without time zone,
    last_successful_fetch_at timestamp without time zone,
    last_fetch_error character varying DEFAULT ''::character varying NOT NULL,
    feed_realtime_enabled boolean DEFAULT false NOT NULL,
    feed_priority integer,
    tags json,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_import_retention_period integer DEFAULT 90 NOT NULL
);
CREATE SEQUENCE public.feed_states_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.feed_states_id_seq OWNED BY public.feed_states.id;
CREATE TABLE public.feed_version_file_infos (
    id bigint NOT NULL,
    feed_version_id bigint NOT NULL,
    name text NOT NULL,
    size bigint NOT NULL,
    rows bigint NOT NULL,
    columns integer NOT NULL,
    sha1 text NOT NULL,
    header text NOT NULL,
    csv_like boolean NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL
);
CREATE SEQUENCE public.feed_version_file_infos_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.feed_version_file_infos_id_seq OWNED BY public.feed_version_file_infos.id;
CREATE TABLE public.feed_version_gtfs_imports (
    id bigint NOT NULL,
    success boolean NOT NULL,
    import_log text NOT NULL,
    exception_log text NOT NULL,
    import_level integer NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id bigint NOT NULL,
    in_progress boolean DEFAULT false NOT NULL,
    skip_entity_error_count jsonb,
    warning_count jsonb,
    entity_count jsonb,
    generated_count jsonb,
    skip_entity_reference_count jsonb,
    skip_entity_filter_count jsonb,
    skip_entity_marked_count jsonb,
    interpolated_stop_time_count integer
);
CREATE SEQUENCE public.feed_version_gtfs_imports_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.feed_version_gtfs_imports_id_seq OWNED BY public.feed_version_gtfs_imports.id;
CREATE TABLE public.feed_version_service_levels (
    id bigint NOT NULL,
    feed_version_id bigint NOT NULL,
    route_id text,
    start_date date NOT NULL,
    end_date date NOT NULL,
    agency_name text NOT NULL,
    route_short_name text NOT NULL,
    route_long_name text NOT NULL,
    route_type integer NOT NULL,
    monday bigint NOT NULL,
    tuesday bigint NOT NULL,
    wednesday bigint NOT NULL,
    thursday bigint NOT NULL,
    friday bigint NOT NULL,
    saturday bigint NOT NULL,
    sunday bigint NOT NULL
);
CREATE SEQUENCE public.feed_version_service_levels_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.feed_version_service_levels_id_seq OWNED BY public.feed_version_service_levels.id;
CREATE SEQUENCE public.feed_versions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.feed_versions_id_seq OWNED BY public.feed_versions.id;
CREATE TABLE public.gtfs_agencies (
    id bigint NOT NULL,
    agency_id character varying NOT NULL,
    agency_name character varying NOT NULL,
    agency_url character varying NOT NULL,
    agency_timezone character varying NOT NULL,
    agency_lang character varying NOT NULL,
    agency_phone character varying NOT NULL,
    agency_fare_url character varying NOT NULL,
    agency_email character varying NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL,
    textsearch tsvector GENERATED ALWAYS AS ((((setweight(to_tsvector('public.tl'::regconfig, (agency_name)::text), 'A'::"char") || setweight(to_tsvector('public.tl'::regconfig, (agency_url)::text), 'B'::"char")) || setweight(to_tsvector('public.tl'::regconfig, (agency_email)::text), 'C'::"char")) || setweight(to_tsvector('public.tl'::regconfig, (agency_id)::text), 'B'::"char"))) STORED
);
CREATE SEQUENCE public.gtfs_agencies_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_agencies_id_seq OWNED BY public.gtfs_agencies.id;
CREATE TABLE public.gtfs_calendar_dates (
    id bigint NOT NULL,
    date date NOT NULL,
    exception_type integer NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL,
    service_id bigint NOT NULL
);
CREATE SEQUENCE public.gtfs_calendar_dates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_calendar_dates_id_seq OWNED BY public.gtfs_calendar_dates.id;
CREATE SEQUENCE public.gtfs_calendars_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_calendars_id_seq OWNED BY public.gtfs_calendars.id;
CREATE TABLE public.gtfs_fare_attributes (
    id bigint NOT NULL,
    fare_id character varying NOT NULL,
    price double precision NOT NULL,
    currency_type character varying NOT NULL,
    payment_method integer NOT NULL,
    transfer_duration integer NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL,
    agency_id bigint,
    transfers integer
);
CREATE SEQUENCE public.gtfs_fare_attributes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_fare_attributes_id_seq OWNED BY public.gtfs_fare_attributes.id;
CREATE TABLE public.gtfs_fare_rules (
    id bigint NOT NULL,
    origin_id character varying NOT NULL,
    destination_id character varying NOT NULL,
    contains_id character varying NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL,
    route_id bigint,
    fare_id bigint
);
CREATE SEQUENCE public.gtfs_fare_rules_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_fare_rules_id_seq OWNED BY public.gtfs_fare_rules.id;
CREATE TABLE public.gtfs_feed_infos (
    id bigint NOT NULL,
    feed_publisher_name character varying NOT NULL,
    feed_publisher_url character varying NOT NULL,
    feed_lang character varying NOT NULL,
    feed_start_date date,
    feed_end_date date,
    feed_version_name character varying NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL
);
CREATE SEQUENCE public.gtfs_feed_infos_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_feed_infos_id_seq OWNED BY public.gtfs_feed_infos.id;
CREATE TABLE public.gtfs_frequencies (
    id bigint NOT NULL,
    start_time integer NOT NULL,
    end_time integer NOT NULL,
    headway_secs integer NOT NULL,
    exact_times integer NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL,
    trip_id bigint NOT NULL
);
CREATE SEQUENCE public.gtfs_frequencies_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_frequencies_id_seq OWNED BY public.gtfs_frequencies.id;
CREATE TABLE public.gtfs_levels (
    id bigint NOT NULL,
    feed_version_id bigint NOT NULL,
    level_id character varying NOT NULL,
    level_index double precision NOT NULL,
    level_name character varying NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    geometry public.geography(Polygon,4326),
    parent_station bigint
);
CREATE SEQUENCE public.gtfs_levels_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_levels_id_seq OWNED BY public.gtfs_levels.id;
CREATE TABLE public.gtfs_pathways (
    id bigint NOT NULL,
    feed_version_id bigint NOT NULL,
    pathway_id character varying NOT NULL,
    from_stop_id bigint NOT NULL,
    to_stop_id bigint NOT NULL,
    pathway_mode integer NOT NULL,
    is_bidirectional integer NOT NULL,
    length double precision NOT NULL,
    traversal_time integer NOT NULL,
    stair_count integer NOT NULL,
    max_slope double precision NOT NULL,
    min_width double precision NOT NULL,
    signposted_as character varying NOT NULL,
    reverse_signposted_as character varying NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL
);
CREATE SEQUENCE public.gtfs_pathways_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_pathways_id_seq OWNED BY public.gtfs_pathways.id;
CREATE TABLE public.gtfs_routes (
    id bigint NOT NULL,
    route_id character varying NOT NULL,
    route_short_name character varying NOT NULL,
    route_long_name character varying NOT NULL,
    route_desc character varying NOT NULL,
    route_type integer NOT NULL,
    route_url character varying NOT NULL,
    route_color character varying NOT NULL,
    route_text_color character varying NOT NULL,
    route_sort_order integer NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL,
    agency_id bigint NOT NULL,
    textsearch tsvector GENERATED ALWAYS AS ((((setweight(to_tsvector('public.tl'::regconfig, (route_short_name)::text), 'A'::"char") || setweight(to_tsvector('public.tl'::regconfig, (route_long_name)::text), 'A'::"char")) || setweight(to_tsvector('public.tl'::regconfig, (route_desc)::text), 'B'::"char")) || setweight(to_tsvector('public.tl'::regconfig, (route_id)::text), 'C'::"char"))) STORED,
    network_id text,
    as_route integer
);
CREATE SEQUENCE public.gtfs_routes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_routes_id_seq OWNED BY public.gtfs_routes.id;
CREATE TABLE public.gtfs_shapes (
    id bigint NOT NULL,
    shape_id character varying NOT NULL,
    generated boolean DEFAULT false NOT NULL,
    geometry public.geography(LineStringM,4326) NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL
);
CREATE SEQUENCE public.gtfs_shapes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_shapes_id_seq OWNED BY public.gtfs_shapes.id;
CREATE TABLE public.gtfs_stop_times_0 (
    feed_version_id bigint NOT NULL,
    trip_id bigint NOT NULL,
    stop_id bigint NOT NULL,
    arrival_time integer NOT NULL,
    departure_time integer NOT NULL,
    stop_sequence integer NOT NULL,
    shape_dist_traveled double precision,
    pickup_type smallint,
    drop_off_type smallint,
    timepoint smallint,
    interpolated smallint,
    stop_headsign text
);
ALTER TABLE ONLY public.gtfs_stop_times ATTACH PARTITION public.gtfs_stop_times_0 FOR VALUES WITH (modulus 10, remainder 0);
CREATE TABLE public.gtfs_stop_times_1 (
    feed_version_id bigint NOT NULL,
    trip_id bigint NOT NULL,
    stop_id bigint NOT NULL,
    arrival_time integer NOT NULL,
    departure_time integer NOT NULL,
    stop_sequence integer NOT NULL,
    shape_dist_traveled double precision,
    pickup_type smallint,
    drop_off_type smallint,
    timepoint smallint,
    interpolated smallint,
    stop_headsign text
);
ALTER TABLE ONLY public.gtfs_stop_times ATTACH PARTITION public.gtfs_stop_times_1 FOR VALUES WITH (modulus 10, remainder 1);
CREATE TABLE public.gtfs_stop_times_2 (
    feed_version_id bigint NOT NULL,
    trip_id bigint NOT NULL,
    stop_id bigint NOT NULL,
    arrival_time integer NOT NULL,
    departure_time integer NOT NULL,
    stop_sequence integer NOT NULL,
    shape_dist_traveled double precision,
    pickup_type smallint,
    drop_off_type smallint,
    timepoint smallint,
    interpolated smallint,
    stop_headsign text
);
ALTER TABLE ONLY public.gtfs_stop_times ATTACH PARTITION public.gtfs_stop_times_2 FOR VALUES WITH (modulus 10, remainder 2);
CREATE TABLE public.gtfs_stop_times_3 (
    feed_version_id bigint NOT NULL,
    trip_id bigint NOT NULL,
    stop_id bigint NOT NULL,
    arrival_time integer NOT NULL,
    departure_time integer NOT NULL,
    stop_sequence integer NOT NULL,
    shape_dist_traveled double precision,
    pickup_type smallint,
    drop_off_type smallint,
    timepoint smallint,
    interpolated smallint,
    stop_headsign text
);
ALTER TABLE ONLY public.gtfs_stop_times ATTACH PARTITION public.gtfs_stop_times_3 FOR VALUES WITH (modulus 10, remainder 3);
CREATE TABLE public.gtfs_stop_times_4 (
    feed_version_id bigint NOT NULL,
    trip_id bigint NOT NULL,
    stop_id bigint NOT NULL,
    arrival_time integer NOT NULL,
    departure_time integer NOT NULL,
    stop_sequence integer NOT NULL,
    shape_dist_traveled double precision,
    pickup_type smallint,
    drop_off_type smallint,
    timepoint smallint,
    interpolated smallint,
    stop_headsign text
);
ALTER TABLE ONLY public.gtfs_stop_times ATTACH PARTITION public.gtfs_stop_times_4 FOR VALUES WITH (modulus 10, remainder 4);
CREATE TABLE public.gtfs_stop_times_5 (
    feed_version_id bigint NOT NULL,
    trip_id bigint NOT NULL,
    stop_id bigint NOT NULL,
    arrival_time integer NOT NULL,
    departure_time integer NOT NULL,
    stop_sequence integer NOT NULL,
    shape_dist_traveled double precision,
    pickup_type smallint,
    drop_off_type smallint,
    timepoint smallint,
    interpolated smallint,
    stop_headsign text
);
ALTER TABLE ONLY public.gtfs_stop_times ATTACH PARTITION public.gtfs_stop_times_5 FOR VALUES WITH (modulus 10, remainder 5);
CREATE TABLE public.gtfs_stop_times_6 (
    feed_version_id bigint NOT NULL,
    trip_id bigint NOT NULL,
    stop_id bigint NOT NULL,
    arrival_time integer NOT NULL,
    departure_time integer NOT NULL,
    stop_sequence integer NOT NULL,
    shape_dist_traveled double precision,
    pickup_type smallint,
    drop_off_type smallint,
    timepoint smallint,
    interpolated smallint,
    stop_headsign text
);
ALTER TABLE ONLY public.gtfs_stop_times ATTACH PARTITION public.gtfs_stop_times_6 FOR VALUES WITH (modulus 10, remainder 6);
CREATE TABLE public.gtfs_stop_times_7 (
    feed_version_id bigint NOT NULL,
    trip_id bigint NOT NULL,
    stop_id bigint NOT NULL,
    arrival_time integer NOT NULL,
    departure_time integer NOT NULL,
    stop_sequence integer NOT NULL,
    shape_dist_traveled double precision,
    pickup_type smallint,
    drop_off_type smallint,
    timepoint smallint,
    interpolated smallint,
    stop_headsign text
);
ALTER TABLE ONLY public.gtfs_stop_times ATTACH PARTITION public.gtfs_stop_times_7 FOR VALUES WITH (modulus 10, remainder 7);
CREATE TABLE public.gtfs_stop_times_8 (
    feed_version_id bigint NOT NULL,
    trip_id bigint NOT NULL,
    stop_id bigint NOT NULL,
    arrival_time integer NOT NULL,
    departure_time integer NOT NULL,
    stop_sequence integer NOT NULL,
    shape_dist_traveled double precision,
    pickup_type smallint,
    drop_off_type smallint,
    timepoint smallint,
    interpolated smallint,
    stop_headsign text
);
ALTER TABLE ONLY public.gtfs_stop_times ATTACH PARTITION public.gtfs_stop_times_8 FOR VALUES WITH (modulus 10, remainder 8);
CREATE TABLE public.gtfs_stop_times_9 (
    feed_version_id bigint NOT NULL,
    trip_id bigint NOT NULL,
    stop_id bigint NOT NULL,
    arrival_time integer NOT NULL,
    departure_time integer NOT NULL,
    stop_sequence integer NOT NULL,
    shape_dist_traveled double precision,
    pickup_type smallint,
    drop_off_type smallint,
    timepoint smallint,
    interpolated smallint,
    stop_headsign text
);
ALTER TABLE ONLY public.gtfs_stop_times ATTACH PARTITION public.gtfs_stop_times_9 FOR VALUES WITH (modulus 10, remainder 9);
CREATE TABLE public.gtfs_stop_times_unpartitioned (
    id bigint NOT NULL,
    arrival_time integer NOT NULL,
    departure_time integer NOT NULL,
    stop_sequence integer NOT NULL,
    stop_headsign character varying NOT NULL,
    pickup_type integer NOT NULL,
    drop_off_type integer NOT NULL,
    shape_dist_traveled double precision NOT NULL,
    timepoint integer NOT NULL,
    interpolated integer DEFAULT 0 NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL,
    trip_id bigint NOT NULL,
    stop_id bigint NOT NULL
);
CREATE SEQUENCE public.gtfs_stop_times_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_stop_times_id_seq OWNED BY public.gtfs_stop_times_unpartitioned.id;
CREATE TABLE public.gtfs_stops (
    id bigint NOT NULL,
    stop_id character varying NOT NULL,
    stop_code character varying NOT NULL,
    stop_name character varying NOT NULL,
    stop_desc character varying NOT NULL,
    zone_id character varying NOT NULL,
    stop_url character varying NOT NULL,
    location_type integer NOT NULL,
    stop_timezone character varying NOT NULL,
    wheelchair_boarding integer NOT NULL,
    geometry public.geography(Point,4326) NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL,
    parent_station bigint,
    level_id bigint,
    textsearch tsvector GENERATED ALWAYS AS (((((setweight(to_tsvector('public.tl'::regconfig, (stop_name)::text), 'A'::"char") || setweight(to_tsvector('public.tl'::regconfig, (stop_desc)::text), 'B'::"char")) || setweight(to_tsvector('public.tl'::regconfig, (stop_code)::text), 'C'::"char")) || setweight(to_tsvector('public.tl'::regconfig, (stop_url)::text), 'C'::"char")) || setweight(to_tsvector('public.tl'::regconfig, (stop_id)::text), 'D'::"char"))) STORED,
    area_id text
);
CREATE SEQUENCE public.gtfs_stops_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_stops_id_seq OWNED BY public.gtfs_stops.id;
CREATE TABLE public.gtfs_transfers (
    id bigint NOT NULL,
    transfer_type integer NOT NULL,
    min_transfer_time integer,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL,
    from_stop_id bigint NOT NULL,
    to_stop_id bigint NOT NULL,
    as_route integer
);
CREATE SEQUENCE public.gtfs_transfers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_transfers_id_seq OWNED BY public.gtfs_transfers.id;
CREATE TABLE public.gtfs_trips (
    id bigint NOT NULL,
    trip_id character varying NOT NULL,
    trip_headsign character varying NOT NULL,
    trip_short_name character varying NOT NULL,
    direction_id integer NOT NULL,
    block_id character varying NOT NULL,
    wheelchair_accessible integer NOT NULL,
    bikes_allowed integer NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL,
    route_id bigint NOT NULL,
    shape_id bigint,
    stop_pattern_id integer NOT NULL,
    service_id bigint NOT NULL,
    journey_pattern_id text NOT NULL,
    journey_pattern_offset integer NOT NULL
);
CREATE SEQUENCE public.gtfs_trips_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_trips_id_seq OWNED BY public.gtfs_trips.id;
CREATE TABLE public.tl_route_headways (
    id bigint NOT NULL,
    feed_version_id bigint NOT NULL,
    route_id bigint NOT NULL,
    selected_stop_id bigint NOT NULL,
    service_id bigint,
    direction_id integer,
    headway_secs integer,
    dow_category integer,
    service_date date,
    service_seconds integer,
    stop_trip_count integer,
    headway_seconds_morning_count integer,
    headway_seconds_morning_min integer,
    headway_seconds_morning_mid integer,
    headway_seconds_morning_max integer,
    headway_seconds_midday_count integer,
    headway_seconds_midday_min integer,
    headway_seconds_midday_mid integer,
    headway_seconds_midday_max integer,
    headway_seconds_afternoon_count integer,
    headway_seconds_afternoon_min integer,
    headway_seconds_afternoon_mid integer,
    headway_seconds_afternoon_max integer,
    headway_seconds_night_count integer,
    headway_seconds_night_min integer,
    headway_seconds_night_mid integer,
    headway_seconds_night_max integer
);
CREATE SEQUENCE public.route_headways_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.route_headways_id_seq OWNED BY public.tl_route_headways.id;
CREATE TABLE public.tl_agency_geometries (
    agency_id bigint NOT NULL,
    feed_version_id bigint NOT NULL,
    geometry public.geography(Polygon,4326),
    centroid public.geography(Point,4326)
);
CREATE TABLE public.tl_agency_onestop_ids (
    id bigint NOT NULL,
    feed_version_id bigint NOT NULL,
    agency_id bigint NOT NULL,
    onestop_id text NOT NULL
);
CREATE SEQUENCE public.tl_agency_onestop_ids_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.tl_agency_onestop_ids_id_seq OWNED BY public.tl_agency_onestop_ids.id;
CREATE TABLE public.tl_census_datasets (
    id bigint NOT NULL,
    dataset_name text NOT NULL,
    year_min integer NOT NULL,
    year_max integer NOT NULL,
    url text NOT NULL
);
CREATE SEQUENCE public.tl_census_datasets_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.tl_census_datasets_id_seq OWNED BY public.tl_census_datasets.id;
CREATE TABLE public.tl_census_fields (
    id bigint NOT NULL,
    table_id bigint NOT NULL,
    field_name text NOT NULL,
    field_title text NOT NULL
);
CREATE SEQUENCE public.tl_census_fields_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.tl_census_fields_id_seq OWNED BY public.tl_census_fields.id;
CREATE TABLE public.tl_census_geographies (
    id bigint NOT NULL,
    source_id bigint NOT NULL,
    layer_name text NOT NULL,
    geoid text,
    name text,
    aland numeric,
    awater numeric,
    geometry public.geography(Polygon,4326)
);
CREATE SEQUENCE public.tl_census_geographies_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.tl_census_geographies_id_seq OWNED BY public.tl_census_geographies.id;
CREATE TABLE public.tl_census_sources (
    id bigint NOT NULL,
    dataset_id bigint NOT NULL,
    source_name text NOT NULL,
    url text NOT NULL,
    sha1 text NOT NULL
);
CREATE SEQUENCE public.tl_census_sources_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.tl_census_sources_id_seq OWNED BY public.tl_census_sources.id;
CREATE TABLE public.tl_census_tables (
    id bigint NOT NULL,
    dataset_id bigint NOT NULL,
    table_name text NOT NULL,
    table_title text NOT NULL,
    table_group text NOT NULL
);
CREATE SEQUENCE public.tl_census_tables_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.tl_census_tables_id_seq OWNED BY public.tl_census_tables.id;
CREATE TABLE public.tl_ext_fare_networks (
    network_id text NOT NULL,
    as_route integer NOT NULL,
    target_feed_onestop_id text NOT NULL,
    target_route_id text NOT NULL,
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL
);
CREATE SEQUENCE public.tl_ext_fare_networks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.tl_ext_fare_networks_id_seq OWNED BY public.tl_ext_fare_networks.id;
CREATE TABLE public.tl_ext_gtfs_stops (
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL,
    target_feed_onestop_id text NOT NULL,
    target_stop_id text NOT NULL,
    inactive boolean DEFAULT false NOT NULL
);
CREATE TABLE public.tl_feed_version_geometries (
    feed_version_id bigint NOT NULL,
    geometry public.geography(Polygon,4326),
    centroid public.geography(Point,4326)
);
CREATE VIEW public.tl_vw_agency_operators AS
 SELECT agencies.id AS agency_id,
    agencies.agency_name,
    tlao.onestop_id AS agency_onestop_id,
    feed_versions.id AS feed_version_id,
    feed_versions.sha1 AS feed_version_sha1,
    cf.id AS feed_id,
    cf.onestop_id AS feed_onestop_id,
    cf.feed_namespace_id,
    coif.onestop_id AS coif_onestop_id,
    co.id AS operator_id,
    co.name AS operator_name,
    co.short_name AS operator_short_name,
    co.onestop_id AS operator_onestop_id,
    (co.tags)::json AS operator_tags,
    co.associated_feeds AS operator_associated_feeds,
    COALESCE(c2.merged_onestop_id, co.onestop_id) AS onestop_id,
    tlp.name AS city_name,
    tlp.adm1name,
    tlp.adm0name,
    tlp_cache.places_cache,
    concat(agencies.agency_name, ' ', cf.onestop_id, ' ', COALESCE(c2.merged_onestop_id, co.onestop_id), ' ', co.name, ' ', co.short_name) AS search_tokens,
    ((((setweight(to_tsvector('public.tl'::regconfig, (COALESCE(agencies.agency_name, ''::character varying))::text), 'A'::"char") || setweight(to_tsvector('public.tl'::regconfig, (COALESCE(co.name, ''::character varying))::text), 'A'::"char")) || setweight(to_tsvector('public.tl'::regconfig, (COALESCE(co.short_name, ''::character varying))::text), 'A'::"char")) || setweight(to_tsvector('public.tl'::regconfig, (COALESCE(c2.merged_onestop_id, co.onestop_id, ''::character varying))::text), 'C'::"char")) || setweight(to_tsvector('public.tl'::regconfig, COALESCE(array_to_string(tlp_cache.places_cache, ','::text), ''::text)), 'D'::"char")) AS textsearch
   FROM ((((((((public.gtfs_agencies agencies
     JOIN public.feed_versions ON ((feed_versions.id = agencies.feed_version_id)))
     JOIN public.current_feeds cf ON ((cf.id = feed_versions.feed_id)))
     LEFT JOIN public.tl_agency_onestop_ids tlao ON ((tlao.agency_id = agencies.id)))
     LEFT JOIN LATERAL ( SELECT tlp_1.name,
            tlp_1.adm1name,
            tlp_1.adm0name
           FROM public.tl_agency_places tlp_1
          WHERE ((tlp_1.agency_id = agencies.id) AND (tlp_1.rank > (0.2)::double precision))
          ORDER BY tlp_1.rank DESC
         LIMIT 1) tlp ON (true))
     LEFT JOIN LATERAL ( SELECT array_agg((((((COALESCE(tlp_1.adm0name, ''::character varying))::text || ' / '::text) || (COALESCE(tlp_1.adm1name, ''::character varying))::text) || ' / '::text) || (COALESCE(tlp_1.name, ''::character varying))::text)) AS places_cache
           FROM public.tl_agency_places tlp_1
          WHERE ((tlp_1.agency_id = agencies.id) AND (tlp_1.rank > (0.2)::double precision))) tlp_cache ON (true))
     LEFT JOIN LATERAL ( SELECT co_1.onestop_id
           FROM (public.current_operators_in_feed coif_1
             JOIN public.current_operators co_1 ON ((co_1.id = coif_1.operator_id)))
          WHERE ((coif_1.feed_id = cf.id) AND ((coif_1.gtfs_agency_id)::text = (agencies.agency_id)::text))) coif ON (true))
     LEFT JOIN LATERAL ( SELECT COALESCE(coif.onestop_id, (NULLIF((cf.feed_namespace_id)::text, ''::text))::character varying, (tlao.onestop_id)::character varying) AS merged_onestop_id) c2 ON (true))
     FULL JOIN public.current_operators co ON (((co.onestop_id)::text = (c2.merged_onestop_id)::text)));
CREATE MATERIALIZED VIEW public.tl_mv_active_agency_operators AS
 SELECT row_number() OVER (ORDER BY tlvw.agency_id, tlvw.operator_id) AS id,
    tlvw.agency_id,
    tlvw.agency_name,
    tlvw.agency_onestop_id,
    tlvw.feed_version_id,
    tlvw.feed_version_sha1,
    tlvw.feed_id,
    tlvw.feed_onestop_id,
    tlvw.feed_namespace_id,
    tlvw.coif_onestop_id,
    tlvw.operator_id,
    tlvw.operator_name,
    tlvw.operator_short_name,
    tlvw.operator_onestop_id,
    tlvw.operator_tags,
    tlvw.operator_associated_feeds,
    tlvw.onestop_id,
    tlvw.city_name,
    tlvw.adm1name,
    tlvw.adm0name,
    tlvw.places_cache,
    tlvw.search_tokens,
    tlvw.textsearch
   FROM (public.tl_vw_agency_operators tlvw
     LEFT JOIN public.feed_states USING (feed_version_id))
  WHERE ((tlvw.agency_id IS NULL) OR (feed_states.id IS NOT NULL))
  ORDER BY tlvw.agency_id, tlvw.operator_id
  WITH NO DATA;
CREATE TABLE public.tl_route_geometries (
    route_id bigint NOT NULL,
    feed_version_id bigint NOT NULL,
    shape_id bigint NOT NULL,
    direction_id integer NOT NULL,
    generated boolean NOT NULL,
    geometry public.geography(LineString,4326) NOT NULL,
    centroid public.geography(Point,4326) NOT NULL
);
CREATE TABLE public.tl_route_onestop_ids (
    id bigint NOT NULL,
    feed_version_id bigint NOT NULL,
    route_id bigint NOT NULL,
    onestop_id text NOT NULL
);
CREATE TABLE public.tl_stop_onestop_ids (
    id bigint NOT NULL,
    feed_version_id bigint NOT NULL,
    stop_id bigint NOT NULL,
    onestop_id text NOT NULL
);
CREATE VIEW public.tl_retained_feed_versions AS
 WITH most_recent_fv AS (
         SELECT DISTINCT ON (fv_1.feed_id) fv_1.id
           FROM ((public.feed_versions fv_1
             JOIN public.current_feeds ON ((current_feeds.id = fv_1.feed_id)))
             JOIN public.feed_states ON ((feed_states.feed_id = current_feeds.id)))
          ORDER BY fv_1.feed_id, fv_1.fetched_at DESC
        ), recent_fvs AS (
         SELECT fv_1.id
           FROM ((public.feed_versions fv_1
             JOIN public.current_feeds ON ((current_feeds.id = fv_1.feed_id)))
             JOIN public.feed_states ON ((feed_states.feed_id = current_feeds.id)))
          WHERE (fv_1.fetched_at >= (CURRENT_DATE - make_interval(days => feed_states.feed_version_import_retention_period)))
        ), active_fv AS (
         SELECT fv_1.id
           FROM (public.feed_versions fv_1
             JOIN public.feed_states ON ((feed_states.feed_version_id = fv_1.id)))
        ), importable_fvs AS (
         SELECT DISTINCT ON (recent_fvs.id) recent_fvs.id
           FROM recent_fvs
        UNION
         SELECT active_fv.id
           FROM active_fv
        UNION
         SELECT most_recent_fv.id
           FROM most_recent_fv
        )
 SELECT fv.id
   FROM ((importable_fvs
     JOIN public.feed_versions fv USING (id))
     JOIN public.current_feeds cf ON ((cf.id = fv.feed_id)))
  WHERE ((cf.deleted_at IS NULL) AND ((cf.spec)::text = 'gtfs'::text))
  ORDER BY fv.fetched_at;
CREATE SEQUENCE public.tl_route_onestop_ids_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.tl_route_onestop_ids_id_seq OWNED BY public.tl_route_onestop_ids.id;
CREATE TABLE public.tl_route_stops (
    feed_version_id bigint NOT NULL,
    agency_id bigint NOT NULL,
    route_id bigint NOT NULL,
    stop_id bigint NOT NULL
);
CREATE TABLE public.tl_stop_external_references (
    id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL,
    target_feed_onestop_id text NOT NULL,
    target_stop_id text NOT NULL
);
CREATE SEQUENCE public.tl_stop_onestop_ids_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.tl_stop_onestop_ids_id_seq OWNED BY public.tl_stop_onestop_ids.id;
CREATE VIEW public.tl_tile_active_routes AS
 SELECT gtfs_routes.id,
    gtfs_routes.route_id,
    gtfs_routes.route_short_name,
    gtfs_routes.route_long_name,
    tl_route_onestop_ids.onestop_id,
    fv.sha1 AS feed_version_sha1,
    cf.onestop_id AS feed_onestop_id,
    tl_route_geometries.geometry,
    tl_route_geometries.generated,
    public.st_length(tl_route_geometries.geometry) AS geometry_length,
    gtfs_agencies.agency_id,
    gtfs_agencies.agency_name,
        CASE
            WHEN ((gtfs_routes.route_color)::text = ''::text) THEN NULL::text
            WHEN ("substring"((gtfs_routes.route_color)::text, 1, 1) <> '#'::text) THEN ('#'::text || lower((gtfs_routes.route_color)::text))
            ELSE lower((gtfs_routes.route_color)::text)
        END AS route_color,
        CASE
            WHEN (gtfs_routes.route_type <= 7) THEN gtfs_routes.route_type
            WHEN ((gtfs_routes.route_type >= 100) AND (gtfs_routes.route_type <= 199)) THEN 2
            WHEN ((gtfs_routes.route_type >= 200) AND (gtfs_routes.route_type <= 299)) THEN 3
            WHEN ((gtfs_routes.route_type >= 300) AND (gtfs_routes.route_type <= 399)) THEN 2
            WHEN ((gtfs_routes.route_type >= 300) AND (gtfs_routes.route_type <= 399)) THEN 2
            WHEN ((gtfs_routes.route_type >= 400) AND (gtfs_routes.route_type <= 699)) THEN 1
            WHEN ((gtfs_routes.route_type >= 700) AND (gtfs_routes.route_type <= 899)) THEN 3
            WHEN ((gtfs_routes.route_type >= 800) AND (gtfs_routes.route_type <= 999)) THEN 0
            WHEN ((gtfs_routes.route_type >= 1000) AND (gtfs_routes.route_type <= 1099)) THEN 4
            WHEN ((gtfs_routes.route_type >= 1100) AND (gtfs_routes.route_type <= 1199)) THEN 3
            WHEN ((gtfs_routes.route_type >= 1200) AND (gtfs_routes.route_type <= 1299)) THEN 4
            WHEN ((gtfs_routes.route_type >= 1300) AND (gtfs_routes.route_type <= 1399)) THEN 6
            WHEN ((gtfs_routes.route_type >= 1400) AND (gtfs_routes.route_type <= 1499)) THEN 7
            WHEN (gtfs_routes.route_type >= 1500) THEN 3
            ELSE 3
        END AS route_type,
        CASE
            WHEN ((tl_route_headways.headway_secs = 0) OR (tl_route_headways.headway_secs IS NULL)) THEN 1000000
            ELSE tl_route_headways.headway_secs
        END AS headway_secs
   FROM (((((((public.gtfs_routes
     JOIN public.feed_states USING (feed_version_id))
     JOIN public.feed_versions fv ON ((fv.id = gtfs_routes.feed_version_id)))
     JOIN public.current_feeds cf ON ((cf.id = fv.feed_id)))
     JOIN public.tl_route_geometries ON ((tl_route_geometries.route_id = gtfs_routes.id)))
     JOIN public.gtfs_agencies ON ((gtfs_agencies.id = gtfs_routes.agency_id)))
     LEFT JOIN public.tl_route_onestop_ids ON ((tl_route_onestop_ids.route_id = gtfs_routes.id)))
     LEFT JOIN public.tl_route_headways ON (((tl_route_headways.route_id = gtfs_routes.id) AND (tl_route_headways.dow_category = 1))));
CREATE VIEW public.tl_tile_active_stops AS
 SELECT gtfs_stops.id,
    gtfs_stops.stop_id,
    gtfs_stops.stop_name,
    gtfs_stops.geometry
   FROM (public.gtfs_stops
     JOIN public.feed_states USING (feed_version_id));
ALTER TABLE ONLY public.current_feeds ALTER COLUMN id SET DEFAULT nextval('public.current_feeds_id_seq'::regclass);
ALTER TABLE ONLY public.current_operators ALTER COLUMN id SET DEFAULT nextval('public.current_operators_id_seq'::regclass);
ALTER TABLE ONLY public.current_operators_in_feed ALTER COLUMN id SET DEFAULT nextval('public.current_operators_in_feed_id_seq'::regclass);
ALTER TABLE ONLY public.ext_faresv2_areas ALTER COLUMN id SET DEFAULT nextval('public.ext_faresv2_areas_id_seq'::regclass);
ALTER TABLE ONLY public.ext_faresv2_fare_capping ALTER COLUMN id SET DEFAULT nextval('public.ext_faresv2_fare_capping_id_seq'::regclass);
ALTER TABLE ONLY public.ext_faresv2_fare_containers ALTER COLUMN id SET DEFAULT nextval('public.ext_faresv2_fare_containers_id_seq'::regclass);
ALTER TABLE ONLY public.ext_faresv2_fare_leg_rules ALTER COLUMN id SET DEFAULT nextval('public.ext_faresv2_fare_leg_rules_id_seq'::regclass);
ALTER TABLE ONLY public.ext_faresv2_fare_products ALTER COLUMN id SET DEFAULT nextval('public.ext_faresv2_fare_products_id_seq'::regclass);
ALTER TABLE ONLY public.ext_faresv2_fare_timeframes ALTER COLUMN id SET DEFAULT nextval('public.ext_faresv2_fare_timeframes_id_seq'::regclass);
ALTER TABLE ONLY public.ext_faresv2_fare_transfer_rules ALTER COLUMN id SET DEFAULT nextval('public.ext_faresv2_fare_transfer_rules_id_seq'::regclass);
ALTER TABLE ONLY public.ext_faresv2_rider_categories ALTER COLUMN id SET DEFAULT nextval('public.ext_faresv2_rider_categories_id_seq'::regclass);
ALTER TABLE ONLY public.ext_plus_calendar_attributes ALTER COLUMN id SET DEFAULT nextval('public.ext_plus_calendar_attributes_id_seq'::regclass);
ALTER TABLE ONLY public.ext_plus_directions ALTER COLUMN id SET DEFAULT nextval('public.ext_plus_directions_id_seq'::regclass);
ALTER TABLE ONLY public.ext_plus_fare_rider_categories ALTER COLUMN id SET DEFAULT nextval('public.ext_plus_fare_rider_categories_id_seq'::regclass);
ALTER TABLE ONLY public.ext_plus_farezone_attributes ALTER COLUMN id SET DEFAULT nextval('public.ext_plus_farezone_attributes_id_seq'::regclass);
ALTER TABLE ONLY public.ext_plus_realtime_routes ALTER COLUMN id SET DEFAULT nextval('public.ext_plus_realtime_routes_id_seq'::regclass);
ALTER TABLE ONLY public.ext_plus_realtime_stops ALTER COLUMN id SET DEFAULT nextval('public.ext_plus_realtime_stops_id_seq'::regclass);
ALTER TABLE ONLY public.ext_plus_realtime_trips ALTER COLUMN id SET DEFAULT nextval('public.ext_plus_realtime_trips_id_seq'::regclass);
ALTER TABLE ONLY public.ext_plus_rider_categories ALTER COLUMN id SET DEFAULT nextval('public.ext_plus_rider_categories_id_seq'::regclass);
ALTER TABLE ONLY public.ext_plus_stop_attributes ALTER COLUMN id SET DEFAULT nextval('public.ext_plus_stop_attributes_id_seq'::regclass);
ALTER TABLE ONLY public.ext_plus_timepoints ALTER COLUMN id SET DEFAULT nextval('public.ext_plus_timepoints_id_seq'::regclass);
ALTER TABLE ONLY public.feed_states ALTER COLUMN id SET DEFAULT nextval('public.feed_states_id_seq'::regclass);
ALTER TABLE ONLY public.feed_version_file_infos ALTER COLUMN id SET DEFAULT nextval('public.feed_version_file_infos_id_seq'::regclass);
ALTER TABLE ONLY public.feed_version_gtfs_imports ALTER COLUMN id SET DEFAULT nextval('public.feed_version_gtfs_imports_id_seq'::regclass);
ALTER TABLE ONLY public.feed_version_service_levels ALTER COLUMN id SET DEFAULT nextval('public.feed_version_service_levels_id_seq'::regclass);
ALTER TABLE ONLY public.feed_versions ALTER COLUMN id SET DEFAULT nextval('public.feed_versions_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_agencies ALTER COLUMN id SET DEFAULT nextval('public.gtfs_agencies_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_calendar_dates ALTER COLUMN id SET DEFAULT nextval('public.gtfs_calendar_dates_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_calendars ALTER COLUMN id SET DEFAULT nextval('public.gtfs_calendars_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_fare_attributes ALTER COLUMN id SET DEFAULT nextval('public.gtfs_fare_attributes_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_fare_rules ALTER COLUMN id SET DEFAULT nextval('public.gtfs_fare_rules_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_feed_infos ALTER COLUMN id SET DEFAULT nextval('public.gtfs_feed_infos_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_frequencies ALTER COLUMN id SET DEFAULT nextval('public.gtfs_frequencies_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_levels ALTER COLUMN id SET DEFAULT nextval('public.gtfs_levels_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_pathways ALTER COLUMN id SET DEFAULT nextval('public.gtfs_pathways_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_routes ALTER COLUMN id SET DEFAULT nextval('public.gtfs_routes_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_shapes ALTER COLUMN id SET DEFAULT nextval('public.gtfs_shapes_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_stop_times_unpartitioned ALTER COLUMN id SET DEFAULT nextval('public.gtfs_stop_times_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_stops ALTER COLUMN id SET DEFAULT nextval('public.gtfs_stops_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_transfers ALTER COLUMN id SET DEFAULT nextval('public.gtfs_transfers_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_trips ALTER COLUMN id SET DEFAULT nextval('public.gtfs_trips_id_seq'::regclass);
ALTER TABLE ONLY public.tl_agency_onestop_ids ALTER COLUMN id SET DEFAULT nextval('public.tl_agency_onestop_ids_id_seq'::regclass);
ALTER TABLE ONLY public.tl_agency_places ALTER COLUMN id SET DEFAULT nextval('public.agency_places_id_seq'::regclass);
ALTER TABLE ONLY public.tl_census_datasets ALTER COLUMN id SET DEFAULT nextval('public.tl_census_datasets_id_seq'::regclass);
ALTER TABLE ONLY public.tl_census_fields ALTER COLUMN id SET DEFAULT nextval('public.tl_census_fields_id_seq'::regclass);
ALTER TABLE ONLY public.tl_census_geographies ALTER COLUMN id SET DEFAULT nextval('public.tl_census_geographies_id_seq'::regclass);
ALTER TABLE ONLY public.tl_census_sources ALTER COLUMN id SET DEFAULT nextval('public.tl_census_sources_id_seq'::regclass);
ALTER TABLE ONLY public.tl_census_tables ALTER COLUMN id SET DEFAULT nextval('public.tl_census_tables_id_seq'::regclass);
ALTER TABLE ONLY public.tl_ext_fare_networks ALTER COLUMN id SET DEFAULT nextval('public.tl_ext_fare_networks_id_seq'::regclass);
ALTER TABLE ONLY public.tl_route_headways ALTER COLUMN id SET DEFAULT nextval('public.route_headways_id_seq'::regclass);
ALTER TABLE ONLY public.tl_route_onestop_ids ALTER COLUMN id SET DEFAULT nextval('public.tl_route_onestop_ids_id_seq'::regclass);
ALTER TABLE ONLY public.tl_stop_onestop_ids ALTER COLUMN id SET DEFAULT nextval('public.tl_stop_onestop_ids_id_seq'::regclass);
ALTER TABLE ONLY public.tl_agency_places
    ADD CONSTRAINT agency_places_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.current_feeds
    ADD CONSTRAINT current_feeds_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.current_operators_in_feed
    ADD CONSTRAINT current_operators_in_feed_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.current_operators
    ADD CONSTRAINT current_operators_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.ext_faresv2_areas
    ADD CONSTRAINT ext_faresv2_areas_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.ext_faresv2_fare_capping
    ADD CONSTRAINT ext_faresv2_fare_capping_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.ext_faresv2_fare_containers
    ADD CONSTRAINT ext_faresv2_fare_containers_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.ext_faresv2_fare_leg_rules
    ADD CONSTRAINT ext_faresv2_fare_leg_rules_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.ext_faresv2_fare_products
    ADD CONSTRAINT ext_faresv2_fare_products_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.ext_faresv2_fare_timeframes
    ADD CONSTRAINT ext_faresv2_fare_timeframes_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.ext_faresv2_fare_transfer_rules
    ADD CONSTRAINT ext_faresv2_fare_transfer_rules_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.ext_faresv2_rider_categories
    ADD CONSTRAINT ext_faresv2_rider_categories_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.ext_plus_calendar_attributes
    ADD CONSTRAINT ext_plus_calendar_attributes_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.ext_plus_directions
    ADD CONSTRAINT ext_plus_directions_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.ext_plus_fare_rider_categories
    ADD CONSTRAINT ext_plus_fare_rider_categories_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.ext_plus_farezone_attributes
    ADD CONSTRAINT ext_plus_farezone_attributes_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.ext_plus_realtime_routes
    ADD CONSTRAINT ext_plus_realtime_routes_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.ext_plus_realtime_stops
    ADD CONSTRAINT ext_plus_realtime_stops_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.ext_plus_realtime_trips
    ADD CONSTRAINT ext_plus_realtime_trips_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.ext_plus_rider_categories
    ADD CONSTRAINT ext_plus_rider_categories_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.ext_plus_stop_attributes
    ADD CONSTRAINT ext_plus_stop_attributes_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.ext_plus_timepoints
    ADD CONSTRAINT ext_plus_timepoints_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.feed_states
    ADD CONSTRAINT feed_states_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.feed_version_file_infos
    ADD CONSTRAINT feed_version_file_infos_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.feed_version_gtfs_imports
    ADD CONSTRAINT feed_version_gtfs_imports_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.feed_version_service_levels
    ADD CONSTRAINT feed_version_service_levels_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.feed_versions
    ADD CONSTRAINT feed_versions_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_agencies
    ADD CONSTRAINT gtfs_agencies_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_calendar_dates
    ADD CONSTRAINT gtfs_calendar_dates_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_calendars
    ADD CONSTRAINT gtfs_calendars_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_fare_attributes
    ADD CONSTRAINT gtfs_fare_attributes_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_fare_rules
    ADD CONSTRAINT gtfs_fare_rules_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_feed_infos
    ADD CONSTRAINT gtfs_feed_infos_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_frequencies
    ADD CONSTRAINT gtfs_frequencies_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_levels
    ADD CONSTRAINT gtfs_levels_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_pathways
    ADD CONSTRAINT gtfs_pathways_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_routes
    ADD CONSTRAINT gtfs_routes_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_shapes
    ADD CONSTRAINT gtfs_shapes_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_stop_times
    ADD CONSTRAINT gtfs_stop_times_pkey1 PRIMARY KEY (feed_version_id, trip_id, stop_sequence);
ALTER TABLE ONLY public.gtfs_stop_times_0
    ADD CONSTRAINT gtfs_stop_times_0_pkey PRIMARY KEY (feed_version_id, trip_id, stop_sequence);
ALTER TABLE ONLY public.gtfs_stop_times_1
    ADD CONSTRAINT gtfs_stop_times_1_pkey PRIMARY KEY (feed_version_id, trip_id, stop_sequence);
ALTER TABLE ONLY public.gtfs_stop_times_2
    ADD CONSTRAINT gtfs_stop_times_2_pkey PRIMARY KEY (feed_version_id, trip_id, stop_sequence);
ALTER TABLE ONLY public.gtfs_stop_times_3
    ADD CONSTRAINT gtfs_stop_times_3_pkey PRIMARY KEY (feed_version_id, trip_id, stop_sequence);
ALTER TABLE ONLY public.gtfs_stop_times_4
    ADD CONSTRAINT gtfs_stop_times_4_pkey PRIMARY KEY (feed_version_id, trip_id, stop_sequence);
ALTER TABLE ONLY public.gtfs_stop_times_5
    ADD CONSTRAINT gtfs_stop_times_5_pkey PRIMARY KEY (feed_version_id, trip_id, stop_sequence);
ALTER TABLE ONLY public.gtfs_stop_times_6
    ADD CONSTRAINT gtfs_stop_times_6_pkey PRIMARY KEY (feed_version_id, trip_id, stop_sequence);
ALTER TABLE ONLY public.gtfs_stop_times_7
    ADD CONSTRAINT gtfs_stop_times_7_pkey PRIMARY KEY (feed_version_id, trip_id, stop_sequence);
ALTER TABLE ONLY public.gtfs_stop_times_8
    ADD CONSTRAINT gtfs_stop_times_8_pkey PRIMARY KEY (feed_version_id, trip_id, stop_sequence);
ALTER TABLE ONLY public.gtfs_stop_times_9
    ADD CONSTRAINT gtfs_stop_times_9_pkey PRIMARY KEY (feed_version_id, trip_id, stop_sequence);
ALTER TABLE ONLY public.gtfs_stop_times_unpartitioned
    ADD CONSTRAINT gtfs_stop_times_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_stops
    ADD CONSTRAINT gtfs_stops_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_transfers
    ADD CONSTRAINT gtfs_transfers_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_trips
    ADD CONSTRAINT gtfs_trips_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.tl_route_headways
    ADD CONSTRAINT route_headways_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.tl_agency_onestop_ids
    ADD CONSTRAINT tl_agency_onestop_ids_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.tl_census_datasets
    ADD CONSTRAINT tl_census_datasets_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.tl_census_fields
    ADD CONSTRAINT tl_census_fields_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.tl_census_geographies
    ADD CONSTRAINT tl_census_geographies_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.tl_census_sources
    ADD CONSTRAINT tl_census_sources_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.tl_census_tables
    ADD CONSTRAINT tl_census_tables_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.tl_ext_fare_networks
    ADD CONSTRAINT tl_ext_fare_networks_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.tl_ext_gtfs_stops
    ADD CONSTRAINT tl_ext_gtfs_stops_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.tl_route_onestop_ids
    ADD CONSTRAINT tl_route_onestop_ids_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.tl_stop_external_references
    ADD CONSTRAINT tl_stop_external_references_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.tl_stop_onestop_ids
    ADD CONSTRAINT tl_stop_onestop_ids_pkey PRIMARY KEY (id);
CREATE INDEX "#c_operators_cu_in_changeset_id_index" ON public.current_operators USING btree (created_or_updated_in_changeset_id);
CREATE INDEX agency_places_agency_id_idx ON public.tl_agency_places USING btree (agency_id);
CREATE INDEX agency_places_best_match_idx ON public.tl_agency_places USING btree (best_match);
CREATE INDEX agency_places_feed_version_id_idx ON public.tl_agency_places USING btree (feed_version_id);
CREATE INDEX current_feeds_feed_tags_idx ON public.current_feeds USING btree (feed_tags);
CREATE INDEX current_feeds_textsearch_idx ON public.current_feeds USING gin (textsearch);
CREATE INDEX current_oif ON public.current_operators_in_feed USING btree (created_or_updated_in_changeset_id);
CREATE INDEX current_operators_operator_tags_idx ON public.current_operators USING btree (operator_tags);
CREATE INDEX current_operators_textsearch_idx ON public.current_operators USING gin (textsearch);
CREATE INDEX ext_plus_calendar_attributes_feed_version_id_idx ON public.ext_plus_calendar_attributes USING btree (feed_version_id);
CREATE INDEX ext_plus_calendar_attributes_service_id_idx ON public.ext_plus_calendar_attributes USING btree (service_id);
CREATE INDEX ext_plus_directions_feed_version_id_idx ON public.ext_plus_directions USING btree (feed_version_id);
CREATE INDEX ext_plus_directions_route_id_idx ON public.ext_plus_directions USING btree (route_id);
CREATE INDEX ext_plus_fare_rider_categories_fare_id_idx ON public.ext_plus_fare_rider_categories USING btree (fare_id);
CREATE INDEX ext_plus_fare_rider_categories_feed_version_id_idx ON public.ext_plus_fare_rider_categories USING btree (feed_version_id);
CREATE INDEX ext_plus_fare_rider_categories_rider_category_id_idx ON public.ext_plus_fare_rider_categories USING btree (rider_category_id);
CREATE INDEX ext_plus_farezone_attributes_feed_version_id_idx ON public.ext_plus_farezone_attributes USING btree (feed_version_id);
CREATE INDEX ext_plus_realtime_routes_feed_version_id_idx ON public.ext_plus_realtime_routes USING btree (feed_version_id);
CREATE INDEX ext_plus_realtime_routes_route_id_idx ON public.ext_plus_realtime_routes USING btree (route_id);
CREATE INDEX ext_plus_realtime_stops_feed_version_id_idx ON public.ext_plus_realtime_stops USING btree (feed_version_id);
CREATE INDEX ext_plus_realtime_stops_stop_id_idx ON public.ext_plus_realtime_stops USING btree (stop_id);
CREATE INDEX ext_plus_realtime_stops_trip_id_idx ON public.ext_plus_realtime_stops USING btree (trip_id);
CREATE INDEX ext_plus_realtime_trips_feed_version_id_idx ON public.ext_plus_realtime_trips USING btree (feed_version_id);
CREATE INDEX ext_plus_realtime_trips_trip_id_idx ON public.ext_plus_realtime_trips USING btree (trip_id);
CREATE INDEX ext_plus_rider_categories_agency_id_idx ON public.ext_plus_rider_categories USING btree (agency_id);
CREATE INDEX ext_plus_rider_categories_feed_version_id_idx ON public.ext_plus_rider_categories USING btree (feed_version_id);
CREATE INDEX ext_plus_stop_attributes_feed_version_id_idx ON public.ext_plus_stop_attributes USING btree (feed_version_id);
CREATE INDEX ext_plus_timepoints_feed_version_id_idx ON public.ext_plus_timepoints USING btree (feed_version_id);
CREATE INDEX ext_plus_timepoints_stop_id_idx ON public.ext_plus_timepoints USING btree (stop_id);
CREATE INDEX ext_plus_timepoints_trip_id_idx ON public.ext_plus_timepoints USING btree (trip_id);
CREATE INDEX feed_version_file_infos_feed_version_id_idx ON public.feed_version_file_infos USING btree (feed_version_id);
CREATE INDEX feed_version_file_infos_name_idx ON public.feed_version_file_infos USING btree (name);
CREATE INDEX feed_version_file_infos_sha1_idx ON public.feed_version_file_infos USING btree (sha1);
CREATE INDEX feed_version_service_levels_end_date_idx ON public.feed_version_service_levels USING btree (end_date);
CREATE INDEX feed_version_service_levels_feed_version_id_idx ON public.feed_version_service_levels USING btree (feed_version_id);
CREATE UNIQUE INDEX feed_version_service_levels_feed_version_id_route_id_start__idx ON public.feed_version_service_levels USING btree (feed_version_id, route_id, start_date, end_date);
CREATE INDEX feed_version_service_levels_route_id_idx ON public.feed_version_service_levels USING btree (route_id);
CREATE INDEX feed_version_service_levels_start_date_idx ON public.feed_version_service_levels USING btree (start_date);
CREATE INDEX feed_versions_fetched_at_idx ON public.feed_versions USING btree (fetched_at);
CREATE INDEX gtfs_agencies_textsearch_idx ON public.gtfs_agencies USING gin (textsearch);
CREATE INDEX gtfs_calendar_dates_service_id_exception_type_date_idx ON public.gtfs_calendar_dates USING btree (service_id, exception_type, date);
CREATE INDEX gtfs_feed_infos_feed_version_id_idx ON public.gtfs_feed_infos USING btree (feed_version_id);
CREATE INDEX gtfs_routes_textsearch_idx ON public.gtfs_routes USING gin (textsearch);
CREATE INDEX gtfs_stop_times_feed_version_id_trip_id_stop_id_idx ON ONLY public.gtfs_stop_times USING btree (feed_version_id, trip_id, stop_id);
CREATE INDEX gtfs_stop_times_0_feed_version_id_trip_id_stop_id_idx ON public.gtfs_stop_times_0 USING btree (feed_version_id, trip_id, stop_id);
CREATE INDEX gtfs_stop_times_stop_id_idx ON ONLY public.gtfs_stop_times USING btree (stop_id);
CREATE INDEX gtfs_stop_times_0_stop_id_idx ON public.gtfs_stop_times_0 USING btree (stop_id);
CREATE INDEX gtfs_stop_times_trip_id_idx ON ONLY public.gtfs_stop_times USING btree (trip_id);
CREATE INDEX gtfs_stop_times_0_trip_id_idx ON public.gtfs_stop_times_0 USING btree (trip_id);
CREATE INDEX gtfs_stop_times_1_feed_version_id_trip_id_stop_id_idx ON public.gtfs_stop_times_1 USING btree (feed_version_id, trip_id, stop_id);
CREATE INDEX gtfs_stop_times_1_stop_id_idx ON public.gtfs_stop_times_1 USING btree (stop_id);
CREATE INDEX gtfs_stop_times_1_trip_id_idx ON public.gtfs_stop_times_1 USING btree (trip_id);
CREATE INDEX gtfs_stop_times_2_feed_version_id_trip_id_stop_id_idx ON public.gtfs_stop_times_2 USING btree (feed_version_id, trip_id, stop_id);
CREATE INDEX gtfs_stop_times_2_stop_id_idx ON public.gtfs_stop_times_2 USING btree (stop_id);
CREATE INDEX gtfs_stop_times_2_trip_id_idx ON public.gtfs_stop_times_2 USING btree (trip_id);
CREATE INDEX gtfs_stop_times_3_feed_version_id_trip_id_stop_id_idx ON public.gtfs_stop_times_3 USING btree (feed_version_id, trip_id, stop_id);
CREATE INDEX gtfs_stop_times_3_stop_id_idx ON public.gtfs_stop_times_3 USING btree (stop_id);
CREATE INDEX gtfs_stop_times_3_trip_id_idx ON public.gtfs_stop_times_3 USING btree (trip_id);
CREATE INDEX gtfs_stop_times_4_feed_version_id_trip_id_stop_id_idx ON public.gtfs_stop_times_4 USING btree (feed_version_id, trip_id, stop_id);
CREATE INDEX gtfs_stop_times_4_stop_id_idx ON public.gtfs_stop_times_4 USING btree (stop_id);
CREATE INDEX gtfs_stop_times_4_trip_id_idx ON public.gtfs_stop_times_4 USING btree (trip_id);
CREATE INDEX gtfs_stop_times_5_feed_version_id_trip_id_stop_id_idx ON public.gtfs_stop_times_5 USING btree (feed_version_id, trip_id, stop_id);
CREATE INDEX gtfs_stop_times_5_stop_id_idx ON public.gtfs_stop_times_5 USING btree (stop_id);
CREATE INDEX gtfs_stop_times_5_trip_id_idx ON public.gtfs_stop_times_5 USING btree (trip_id);
CREATE INDEX gtfs_stop_times_6_feed_version_id_trip_id_stop_id_idx ON public.gtfs_stop_times_6 USING btree (feed_version_id, trip_id, stop_id);
CREATE INDEX gtfs_stop_times_6_stop_id_idx ON public.gtfs_stop_times_6 USING btree (stop_id);
CREATE INDEX gtfs_stop_times_6_trip_id_idx ON public.gtfs_stop_times_6 USING btree (trip_id);
CREATE INDEX gtfs_stop_times_7_feed_version_id_trip_id_stop_id_idx ON public.gtfs_stop_times_7 USING btree (feed_version_id, trip_id, stop_id);
CREATE INDEX gtfs_stop_times_7_stop_id_idx ON public.gtfs_stop_times_7 USING btree (stop_id);
CREATE INDEX gtfs_stop_times_7_trip_id_idx ON public.gtfs_stop_times_7 USING btree (trip_id);
CREATE INDEX gtfs_stop_times_8_feed_version_id_trip_id_stop_id_idx ON public.gtfs_stop_times_8 USING btree (feed_version_id, trip_id, stop_id);
CREATE INDEX gtfs_stop_times_8_stop_id_idx ON public.gtfs_stop_times_8 USING btree (stop_id);
CREATE INDEX gtfs_stop_times_8_trip_id_idx ON public.gtfs_stop_times_8 USING btree (trip_id);
CREATE INDEX gtfs_stop_times_9_feed_version_id_trip_id_stop_id_idx ON public.gtfs_stop_times_9 USING btree (feed_version_id, trip_id, stop_id);
CREATE INDEX gtfs_stop_times_9_stop_id_idx ON public.gtfs_stop_times_9 USING btree (stop_id);
CREATE INDEX gtfs_stop_times_9_trip_id_idx ON public.gtfs_stop_times_9 USING btree (trip_id);
CREATE INDEX gtfs_stops_textsearch_idx ON public.gtfs_stops USING gin (textsearch);
CREATE INDEX gtfs_trips_journey_pattern_id_idx ON public.gtfs_trips USING btree (journey_pattern_id);
CREATE INDEX index_agency_geometries_on_centroid ON public.tl_agency_geometries USING gist (centroid);
CREATE INDEX index_agency_geometries_on_feed_version_id ON public.tl_agency_geometries USING btree (feed_version_id);
CREATE INDEX index_agency_geometries_on_geometry ON public.tl_agency_geometries USING gist (geometry);
CREATE UNIQUE INDEX index_agency_geometries_unique ON public.tl_agency_geometries USING btree (agency_id);
CREATE INDEX index_current_feeds_on_active_feed_version_id ON public.current_feeds USING btree (active_feed_version_id);
CREATE INDEX index_current_feeds_on_auth ON public.current_feeds USING btree (auth);
CREATE INDEX index_current_feeds_on_created_or_updated_in_changeset_id ON public.current_feeds USING btree (created_or_updated_in_changeset_id);
CREATE INDEX index_current_feeds_on_geometry ON public.current_feeds USING gist (geometry);
CREATE UNIQUE INDEX index_current_feeds_on_onestop_id ON public.current_feeds USING btree (onestop_id);
CREATE INDEX index_current_feeds_on_urls ON public.current_feeds USING btree (urls);
CREATE INDEX index_current_operators_in_feed_on_feed_id ON public.current_operators_in_feed USING btree (feed_id);
CREATE INDEX index_current_operators_in_feed_on_operator_id ON public.current_operators_in_feed USING btree (operator_id);
CREATE INDEX index_current_operators_on_geometry ON public.current_operators USING gist (geometry);
CREATE UNIQUE INDEX index_current_operators_on_onestop_id ON public.current_operators USING btree (onestop_id);
CREATE INDEX index_current_operators_on_tags ON public.current_operators USING btree (tags);
CREATE INDEX index_current_operators_on_updated_at ON public.current_operators USING btree (updated_at);
CREATE UNIQUE INDEX index_feed_states_on_feed_id ON public.feed_states USING btree (feed_id);
CREATE UNIQUE INDEX index_feed_states_on_feed_priority ON public.feed_states USING btree (feed_priority);
CREATE UNIQUE INDEX index_feed_states_on_feed_version_id ON public.feed_states USING btree (feed_version_id);
CREATE INDEX index_feed_version_geometries_on_centroid ON public.tl_feed_version_geometries USING gist (centroid);
CREATE INDEX index_feed_version_geometries_on_geometry ON public.tl_feed_version_geometries USING gist (geometry);
CREATE UNIQUE INDEX index_feed_version_geometries_unique ON public.tl_feed_version_geometries USING btree (feed_version_id);
CREATE UNIQUE INDEX index_feed_version_gtfs_imports_on_feed_version_id ON public.feed_version_gtfs_imports USING btree (feed_version_id);
CREATE INDEX index_feed_version_gtfs_imports_on_success ON public.feed_version_gtfs_imports USING btree (success);
CREATE INDEX index_feed_versions_on_earliest_calendar_date ON public.feed_versions USING btree (earliest_calendar_date);
CREATE INDEX index_feed_versions_on_feed_type_and_feed_id ON public.feed_versions USING btree (feed_type, feed_id);
CREATE INDEX index_feed_versions_on_latest_calendar_date ON public.feed_versions USING btree (latest_calendar_date);
CREATE INDEX index_gtfs_agencies_on_agency_id ON public.gtfs_agencies USING btree (agency_id);
CREATE INDEX index_gtfs_agencies_on_agency_name ON public.gtfs_agencies USING btree (agency_name);
CREATE UNIQUE INDEX index_gtfs_agencies_unique ON public.gtfs_agencies USING btree (feed_version_id, agency_id);
CREATE INDEX index_gtfs_calendar_dates_on_date ON public.gtfs_calendar_dates USING btree (date);
CREATE INDEX index_gtfs_calendar_dates_on_exception_type ON public.gtfs_calendar_dates USING btree (exception_type);
CREATE INDEX index_gtfs_calendar_dates_on_feed_version_id ON public.gtfs_calendar_dates USING btree (feed_version_id);
CREATE INDEX index_gtfs_calendar_dates_on_service_id ON public.gtfs_calendar_dates USING btree (service_id);
CREATE INDEX index_gtfs_calendars_on_end_date ON public.gtfs_calendars USING btree (end_date);
CREATE UNIQUE INDEX index_gtfs_calendars_on_feed_version_id_and_service_id ON public.gtfs_calendars USING btree (feed_version_id, service_id);
CREATE INDEX index_gtfs_calendars_on_friday ON public.gtfs_calendars USING btree (friday);
CREATE INDEX index_gtfs_calendars_on_monday ON public.gtfs_calendars USING btree (monday);
CREATE INDEX index_gtfs_calendars_on_saturday ON public.gtfs_calendars USING btree (saturday);
CREATE INDEX index_gtfs_calendars_on_service_id ON public.gtfs_calendars USING btree (service_id);
CREATE INDEX index_gtfs_calendars_on_start_date ON public.gtfs_calendars USING btree (start_date);
CREATE INDEX index_gtfs_calendars_on_sunday ON public.gtfs_calendars USING btree (sunday);
CREATE INDEX index_gtfs_calendars_on_thursday ON public.gtfs_calendars USING btree (thursday);
CREATE INDEX index_gtfs_calendars_on_tuesday ON public.gtfs_calendars USING btree (tuesday);
CREATE INDEX index_gtfs_calendars_on_wednesday ON public.gtfs_calendars USING btree (wednesday);
CREATE INDEX index_gtfs_fare_attributes_on_agency_id ON public.gtfs_fare_attributes USING btree (agency_id);
CREATE INDEX index_gtfs_fare_attributes_on_fare_id ON public.gtfs_fare_attributes USING btree (fare_id);
CREATE UNIQUE INDEX index_gtfs_fare_attributes_unique ON public.gtfs_fare_attributes USING btree (feed_version_id, fare_id);
CREATE INDEX index_gtfs_fare_rules_on_fare_id ON public.gtfs_fare_rules USING btree (fare_id);
CREATE INDEX index_gtfs_fare_rules_on_feed_version_id ON public.gtfs_fare_rules USING btree (feed_version_id);
CREATE INDEX index_gtfs_fare_rules_on_route_id ON public.gtfs_fare_rules USING btree (route_id);
CREATE INDEX index_gtfs_frequencies_on_feed_version_id ON public.gtfs_frequencies USING btree (feed_version_id);
CREATE INDEX index_gtfs_frequencies_on_trip_id ON public.gtfs_frequencies USING btree (trip_id);
CREATE UNIQUE INDEX index_gtfs_levels_unique ON public.gtfs_levels USING btree (feed_version_id, level_id);
CREATE INDEX index_gtfs_pathways_on_from_stop_id ON public.gtfs_pathways USING btree (from_stop_id);
CREATE INDEX index_gtfs_pathways_on_level_id ON public.gtfs_levels USING btree (level_id);
CREATE INDEX index_gtfs_pathways_on_pathway_id ON public.gtfs_pathways USING btree (pathway_id);
CREATE INDEX index_gtfs_pathways_on_to_stop_id ON public.gtfs_pathways USING btree (to_stop_id);
CREATE UNIQUE INDEX index_gtfs_pathways_unique ON public.gtfs_pathways USING btree (feed_version_id, pathway_id);
CREATE INDEX index_gtfs_routes_on_agency_id ON public.gtfs_routes USING btree (agency_id);
CREATE INDEX index_gtfs_routes_on_feed_version_id_agency_id ON public.gtfs_routes USING btree (feed_version_id, id, agency_id);
CREATE INDEX index_gtfs_routes_on_route_desc ON public.gtfs_routes USING btree (route_desc);
CREATE INDEX index_gtfs_routes_on_route_id ON public.gtfs_routes USING btree (route_id);
CREATE INDEX index_gtfs_routes_on_route_long_name ON public.gtfs_routes USING btree (route_long_name);
CREATE INDEX index_gtfs_routes_on_route_short_name ON public.gtfs_routes USING btree (route_short_name);
CREATE INDEX index_gtfs_routes_on_route_type ON public.gtfs_routes USING btree (route_type);
CREATE UNIQUE INDEX index_gtfs_routes_unique ON public.gtfs_routes USING btree (feed_version_id, route_id);
CREATE INDEX index_gtfs_shapes_on_generated ON public.gtfs_shapes USING btree (generated);
CREATE INDEX index_gtfs_shapes_on_geometry ON public.gtfs_shapes USING gist (geometry);
CREATE INDEX index_gtfs_shapes_on_shape_id ON public.gtfs_shapes USING btree (shape_id);
CREATE UNIQUE INDEX index_gtfs_shapes_unique ON public.gtfs_shapes USING btree (feed_version_id, shape_id);
CREATE INDEX index_gtfs_stop_times_on_feed_version_id_trip_id_stop_id ON public.gtfs_stop_times_unpartitioned USING btree (feed_version_id, trip_id, stop_id);
CREATE INDEX index_gtfs_stop_times_on_stop_id ON public.gtfs_stop_times_unpartitioned USING btree (stop_id);
CREATE INDEX index_gtfs_stop_times_on_trip_id ON public.gtfs_stop_times_unpartitioned USING btree (trip_id);
CREATE UNIQUE INDEX index_gtfs_stop_times_unique ON public.gtfs_stop_times_unpartitioned USING btree (feed_version_id, trip_id, stop_sequence);
CREATE INDEX index_gtfs_stops_on_geometry ON public.gtfs_stops USING gist (geometry);
CREATE INDEX index_gtfs_stops_on_location_type ON public.gtfs_stops USING btree (location_type);
CREATE INDEX index_gtfs_stops_on_parent_station ON public.gtfs_stops USING btree (parent_station);
CREATE INDEX index_gtfs_stops_on_stop_code ON public.gtfs_stops USING btree (stop_code);
CREATE INDEX index_gtfs_stops_on_stop_desc ON public.gtfs_stops USING btree (stop_desc);
CREATE INDEX index_gtfs_stops_on_stop_id ON public.gtfs_stops USING btree (stop_id);
CREATE INDEX index_gtfs_stops_on_stop_name ON public.gtfs_stops USING btree (stop_name);
CREATE UNIQUE INDEX index_gtfs_stops_unique ON public.gtfs_stops USING btree (feed_version_id, stop_id);
CREATE INDEX index_gtfs_transfers_on_feed_version_id ON public.gtfs_transfers USING btree (feed_version_id);
CREATE INDEX index_gtfs_transfers_on_from_stop_id ON public.gtfs_transfers USING btree (from_stop_id);
CREATE INDEX index_gtfs_transfers_on_to_stop_id ON public.gtfs_transfers USING btree (to_stop_id);
CREATE INDEX index_gtfs_trips_on_route_id ON public.gtfs_trips USING btree (route_id);
CREATE INDEX index_gtfs_trips_on_service_id ON public.gtfs_trips USING btree (service_id);
CREATE INDEX index_gtfs_trips_on_shape_id ON public.gtfs_trips USING btree (shape_id);
CREATE INDEX index_gtfs_trips_on_trip_headsign ON public.gtfs_trips USING btree (trip_headsign);
CREATE INDEX index_gtfs_trips_on_trip_id ON public.gtfs_trips USING btree (trip_id);
CREATE INDEX index_gtfs_trips_on_trip_short_name ON public.gtfs_trips USING btree (trip_short_name);
CREATE UNIQUE INDEX index_gtfs_trips_unique ON public.gtfs_trips USING btree (feed_version_id, trip_id);
CREATE INDEX index_route_geometries_on_centroid ON public.tl_route_geometries USING gist (centroid);
CREATE INDEX index_route_geometries_on_feed_version_id ON public.tl_route_geometries USING btree (feed_version_id);
CREATE INDEX index_route_geometries_on_geometry ON public.tl_route_geometries USING gist (geometry);
CREATE INDEX index_route_geometries_on_shape_id ON public.tl_route_geometries USING btree (shape_id);
CREATE UNIQUE INDEX index_route_geometries_unique ON public.tl_route_geometries USING btree (route_id, direction_id);
CREATE INDEX index_route_stops_on_agency_id ON public.tl_route_stops USING btree (agency_id);
CREATE INDEX index_route_stops_on_feed_version_id ON public.tl_route_stops USING btree (feed_version_id);
CREATE INDEX index_route_stops_on_route_id ON public.tl_route_stops USING btree (route_id);
CREATE INDEX index_route_stops_on_stop_id ON public.tl_route_stops USING btree (stop_id);
CREATE INDEX route_headways_feed_version_id_idx ON public.tl_route_headways USING btree (feed_version_id);
CREATE UNIQUE INDEX tl_agency_onestop_ids_agency_id_idx ON public.tl_agency_onestop_ids USING btree (agency_id);
CREATE INDEX tl_agency_onestop_ids_feed_version_id_idx ON public.tl_agency_onestop_ids USING btree (feed_version_id);
CREATE INDEX tl_agency_onestop_ids_onestop_id_idx ON public.tl_agency_onestop_ids USING btree (onestop_id);
CREATE INDEX tl_agency_places_adm0name_idx ON public.tl_agency_places USING gin (adm0name public.gin_trgm_ops);
CREATE INDEX tl_agency_places_adm1name_idx ON public.tl_agency_places USING gin (adm1name public.gin_trgm_ops);
CREATE INDEX tl_agency_places_name_idx ON public.tl_agency_places USING gin (name public.gin_trgm_ops);
CREATE UNIQUE INDEX tl_census_datasets_dataset_name_idx ON public.tl_census_datasets USING btree (dataset_name);
CREATE INDEX tl_census_fields_field_name_idx ON public.tl_census_fields USING btree (field_name);
CREATE UNIQUE INDEX tl_census_fields_table_id_field_name_idx ON public.tl_census_fields USING btree (table_id, field_name);
CREATE INDEX tl_census_geographies_geoid_idx ON public.tl_census_geographies USING btree (geoid);
CREATE INDEX tl_census_geographies_geometry_idx ON public.tl_census_geographies USING gist (geometry);
CREATE INDEX tl_census_geographies_layer_name_idx ON public.tl_census_geographies USING btree (layer_name);
CREATE INDEX tl_census_geographies_source_id_idx ON public.tl_census_geographies USING btree (source_id);
CREATE INDEX tl_census_sources_dataset_id_idx ON public.tl_census_sources USING btree (dataset_id);
CREATE INDEX tl_census_tables_dataset_id_idx ON public.tl_census_tables USING btree (dataset_id);
CREATE UNIQUE INDEX tl_census_tables_dataset_id_table_name_idx ON public.tl_census_tables USING btree (dataset_id, table_name);
CREATE INDEX tl_census_tables_table_name_idx ON public.tl_census_tables USING btree (table_name);
CREATE UNIQUE INDEX tl_census_values_geography_id_table_id_idx ON public.tl_census_values USING btree (geography_id, table_id);
CREATE INDEX tl_ext_gtfs_stops_feed_version_id_idx ON public.tl_ext_gtfs_stops USING btree (feed_version_id);
CREATE INDEX tl_ext_gtfs_stops_target_feed_onestop_id_idx ON public.tl_ext_gtfs_stops USING btree (target_feed_onestop_id);
CREATE INDEX tl_ext_gtfs_stops_target_stop_id_idx ON public.tl_ext_gtfs_stops USING btree (target_stop_id);
CREATE INDEX tl_mv_active_agency_operators_adm0name_idx ON public.tl_mv_active_agency_operators USING btree (adm0name);
CREATE INDEX tl_mv_active_agency_operators_adm1name_idx ON public.tl_mv_active_agency_operators USING btree (adm1name);
CREATE INDEX tl_mv_active_agency_operators_agency_id_idx ON public.tl_mv_active_agency_operators USING btree (agency_id);
CREATE INDEX tl_mv_active_agency_operators_agency_id_idx1 ON public.tl_mv_active_agency_operators USING btree (agency_id);
CREATE INDEX tl_mv_active_agency_operators_city_name_idx ON public.tl_mv_active_agency_operators USING btree (city_name);
CREATE INDEX tl_mv_active_agency_operators_feed_id_idx ON public.tl_mv_active_agency_operators USING btree (feed_id);
CREATE INDEX tl_mv_active_agency_operators_feed_id_idx1 ON public.tl_mv_active_agency_operators USING btree (feed_id);
CREATE INDEX tl_mv_active_agency_operators_feed_onestop_id_idx ON public.tl_mv_active_agency_operators USING btree (feed_onestop_id);
CREATE INDEX tl_mv_active_agency_operators_feed_onestop_id_idx1 ON public.tl_mv_active_agency_operators USING btree (feed_onestop_id);
CREATE INDEX tl_mv_active_agency_operators_id_idx ON public.tl_mv_active_agency_operators USING btree (id);
CREATE INDEX tl_mv_active_agency_operators_onestop_id_idx ON public.tl_mv_active_agency_operators USING btree (onestop_id);
CREATE INDEX tl_mv_active_agency_operators_search_tokens_idx ON public.tl_mv_active_agency_operators USING gin (search_tokens public.gin_trgm_ops);
CREATE INDEX tl_mv_active_agency_operators_textsearch_idx ON public.tl_mv_active_agency_operators USING gin (textsearch);
CREATE UNIQUE INDEX tl_route_headways_route_dow_category_id_idx ON public.tl_route_headways USING btree (route_id, dow_category);
CREATE INDEX tl_route_onestop_ids_feed_version_id_idx ON public.tl_route_onestop_ids USING btree (feed_version_id);
CREATE INDEX tl_route_onestop_ids_onestop_id_idx ON public.tl_route_onestop_ids USING btree (onestop_id);
CREATE UNIQUE INDEX tl_route_onestop_ids_route_id_idx ON public.tl_route_onestop_ids USING btree (route_id);
CREATE INDEX tl_stop_external_references_feed_version_id_idx ON public.tl_stop_external_references USING btree (feed_version_id);
CREATE INDEX tl_stop_external_references_target_feed_onestop_id_idx ON public.tl_stop_external_references USING btree (target_feed_onestop_id);
CREATE INDEX tl_stop_external_references_target_stop_id_idx ON public.tl_stop_external_references USING btree (target_stop_id);
CREATE INDEX tl_stop_onestop_ids_feed_version_id_idx ON public.tl_stop_onestop_ids USING btree (feed_version_id);
CREATE INDEX tl_stop_onestop_ids_onestop_id_idx ON public.tl_stop_onestop_ids USING btree (onestop_id);
CREATE UNIQUE INDEX tl_stop_onestop_ids_stop_id_idx ON public.tl_stop_onestop_ids USING btree (stop_id);
ALTER INDEX public.gtfs_stop_times_feed_version_id_trip_id_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_0_feed_version_id_trip_id_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_pkey1 ATTACH PARTITION public.gtfs_stop_times_0_pkey;
ALTER INDEX public.gtfs_stop_times_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_0_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_trip_id_idx ATTACH PARTITION public.gtfs_stop_times_0_trip_id_idx;
ALTER INDEX public.gtfs_stop_times_feed_version_id_trip_id_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_1_feed_version_id_trip_id_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_pkey1 ATTACH PARTITION public.gtfs_stop_times_1_pkey;
ALTER INDEX public.gtfs_stop_times_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_1_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_trip_id_idx ATTACH PARTITION public.gtfs_stop_times_1_trip_id_idx;
ALTER INDEX public.gtfs_stop_times_feed_version_id_trip_id_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_2_feed_version_id_trip_id_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_pkey1 ATTACH PARTITION public.gtfs_stop_times_2_pkey;
ALTER INDEX public.gtfs_stop_times_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_2_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_trip_id_idx ATTACH PARTITION public.gtfs_stop_times_2_trip_id_idx;
ALTER INDEX public.gtfs_stop_times_feed_version_id_trip_id_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_3_feed_version_id_trip_id_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_pkey1 ATTACH PARTITION public.gtfs_stop_times_3_pkey;
ALTER INDEX public.gtfs_stop_times_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_3_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_trip_id_idx ATTACH PARTITION public.gtfs_stop_times_3_trip_id_idx;
ALTER INDEX public.gtfs_stop_times_feed_version_id_trip_id_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_4_feed_version_id_trip_id_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_pkey1 ATTACH PARTITION public.gtfs_stop_times_4_pkey;
ALTER INDEX public.gtfs_stop_times_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_4_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_trip_id_idx ATTACH PARTITION public.gtfs_stop_times_4_trip_id_idx;
ALTER INDEX public.gtfs_stop_times_feed_version_id_trip_id_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_5_feed_version_id_trip_id_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_pkey1 ATTACH PARTITION public.gtfs_stop_times_5_pkey;
ALTER INDEX public.gtfs_stop_times_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_5_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_trip_id_idx ATTACH PARTITION public.gtfs_stop_times_5_trip_id_idx;
ALTER INDEX public.gtfs_stop_times_feed_version_id_trip_id_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_6_feed_version_id_trip_id_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_pkey1 ATTACH PARTITION public.gtfs_stop_times_6_pkey;
ALTER INDEX public.gtfs_stop_times_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_6_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_trip_id_idx ATTACH PARTITION public.gtfs_stop_times_6_trip_id_idx;
ALTER INDEX public.gtfs_stop_times_feed_version_id_trip_id_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_7_feed_version_id_trip_id_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_pkey1 ATTACH PARTITION public.gtfs_stop_times_7_pkey;
ALTER INDEX public.gtfs_stop_times_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_7_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_trip_id_idx ATTACH PARTITION public.gtfs_stop_times_7_trip_id_idx;
ALTER INDEX public.gtfs_stop_times_feed_version_id_trip_id_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_8_feed_version_id_trip_id_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_pkey1 ATTACH PARTITION public.gtfs_stop_times_8_pkey;
ALTER INDEX public.gtfs_stop_times_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_8_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_trip_id_idx ATTACH PARTITION public.gtfs_stop_times_8_trip_id_idx;
ALTER INDEX public.gtfs_stop_times_feed_version_id_trip_id_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_9_feed_version_id_trip_id_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_pkey1 ATTACH PARTITION public.gtfs_stop_times_9_pkey;
ALTER INDEX public.gtfs_stop_times_stop_id_idx ATTACH PARTITION public.gtfs_stop_times_9_stop_id_idx;
ALTER INDEX public.gtfs_stop_times_trip_id_idx ATTACH PARTITION public.gtfs_stop_times_9_trip_id_idx;
ALTER TABLE ONLY public.ext_faresv2_areas
    ADD CONSTRAINT ext_faresv2_areas_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.ext_faresv2_fare_capping
    ADD CONSTRAINT ext_faresv2_fare_capping_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.ext_faresv2_fare_containers
    ADD CONSTRAINT ext_faresv2_fare_containers_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.ext_faresv2_fare_leg_rules
    ADD CONSTRAINT ext_faresv2_fare_leg_rules_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.ext_faresv2_fare_products
    ADD CONSTRAINT ext_faresv2_fare_products_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.ext_faresv2_fare_timeframes
    ADD CONSTRAINT ext_faresv2_fare_timeframes_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.ext_faresv2_fare_transfer_rules
    ADD CONSTRAINT ext_faresv2_fare_transfer_rules_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.ext_faresv2_rider_categories
    ADD CONSTRAINT ext_faresv2_rider_categories_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.ext_plus_calendar_attributes
    ADD CONSTRAINT ext_plus_calendar_attributes_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.ext_plus_calendar_attributes
    ADD CONSTRAINT ext_plus_calendar_attributes_service_id_fkey FOREIGN KEY (service_id) REFERENCES public.gtfs_calendars(id);
ALTER TABLE ONLY public.ext_plus_directions
    ADD CONSTRAINT ext_plus_directions_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.ext_plus_directions
    ADD CONSTRAINT ext_plus_directions_route_id_fkey FOREIGN KEY (route_id) REFERENCES public.gtfs_routes(id);
ALTER TABLE ONLY public.ext_plus_fare_rider_categories
    ADD CONSTRAINT ext_plus_fare_rider_categories_fare_id_fkey FOREIGN KEY (fare_id) REFERENCES public.gtfs_fare_attributes(id);
ALTER TABLE ONLY public.ext_plus_fare_rider_categories
    ADD CONSTRAINT ext_plus_fare_rider_categories_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.ext_plus_farezone_attributes
    ADD CONSTRAINT ext_plus_farezone_attributes_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.ext_plus_realtime_routes
    ADD CONSTRAINT ext_plus_realtime_routes_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.ext_plus_realtime_routes
    ADD CONSTRAINT ext_plus_realtime_routes_route_id_fkey FOREIGN KEY (route_id) REFERENCES public.gtfs_routes(id);
ALTER TABLE ONLY public.ext_plus_realtime_stops
    ADD CONSTRAINT ext_plus_realtime_stops_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.ext_plus_realtime_stops
    ADD CONSTRAINT ext_plus_realtime_stops_stop_id_fkey FOREIGN KEY (stop_id) REFERENCES public.gtfs_stops(id);
ALTER TABLE ONLY public.ext_plus_realtime_stops
    ADD CONSTRAINT ext_plus_realtime_stops_trip_id_fkey FOREIGN KEY (trip_id) REFERENCES public.gtfs_trips(id);
ALTER TABLE ONLY public.ext_plus_realtime_trips
    ADD CONSTRAINT ext_plus_realtime_trips_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.ext_plus_realtime_trips
    ADD CONSTRAINT ext_plus_realtime_trips_trip_id_fkey FOREIGN KEY (trip_id) REFERENCES public.gtfs_trips(id);
ALTER TABLE ONLY public.ext_plus_rider_categories
    ADD CONSTRAINT ext_plus_rider_categories_agency_id_fkey FOREIGN KEY (agency_id) REFERENCES public.gtfs_agencies(id);
ALTER TABLE ONLY public.ext_plus_rider_categories
    ADD CONSTRAINT ext_plus_rider_categories_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.ext_plus_stop_attributes
    ADD CONSTRAINT ext_plus_stop_attributes_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.ext_plus_stop_attributes
    ADD CONSTRAINT ext_plus_stop_attributes_stop_id_fkey FOREIGN KEY (stop_id) REFERENCES public.gtfs_stops(id);
ALTER TABLE ONLY public.ext_plus_timepoints
    ADD CONSTRAINT ext_plus_timepoints_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.ext_plus_timepoints
    ADD CONSTRAINT ext_plus_timepoints_stop_id_fkey FOREIGN KEY (stop_id) REFERENCES public.gtfs_stops(id);
ALTER TABLE ONLY public.ext_plus_timepoints
    ADD CONSTRAINT ext_plus_timepoints_trip_id_fkey FOREIGN KEY (trip_id) REFERENCES public.gtfs_trips(id);
ALTER TABLE ONLY public.feed_version_file_infos
    ADD CONSTRAINT feed_version_file_infos_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.feed_version_service_levels
    ADD CONSTRAINT feed_version_service_levels_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_trips
    ADD CONSTRAINT fk_rails_05ead08753 FOREIGN KEY (shape_id) REFERENCES public.gtfs_shapes(id);
ALTER TABLE ONLY public.tl_route_headways
    ADD CONSTRAINT fk_rails_078ffc5894 FOREIGN KEY (service_id) REFERENCES public.gtfs_calendars(id);
ALTER TABLE ONLY public.gtfs_transfers
    ADD CONSTRAINT fk_rails_0cc6ff288a FOREIGN KEY (from_stop_id) REFERENCES public.gtfs_stops(id);
ALTER TABLE ONLY public.tl_route_headways
    ADD CONSTRAINT fk_rails_19cb5c8c5c FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.tl_agency_geometries
    ADD CONSTRAINT fk_rails_1bfa787783 FOREIGN KEY (agency_id) REFERENCES public.gtfs_agencies(id);
ALTER TABLE ONLY public.tl_route_stops
    ADD CONSTRAINT fk_rails_1dee96ee31 FOREIGN KEY (agency_id) REFERENCES public.gtfs_agencies(id);
ALTER TABLE ONLY public.tl_route_stops
    ADD CONSTRAINT fk_rails_1f4cc828f8 FOREIGN KEY (route_id) REFERENCES public.gtfs_routes(id);
ALTER TABLE ONLY public.gtfs_stop_times_unpartitioned
    ADD CONSTRAINT fk_rails_22a671077b FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.feed_version_gtfs_imports
    ADD CONSTRAINT fk_rails_2d141782c9 FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_stop_times_unpartitioned
    ADD CONSTRAINT fk_rails_30ced0baa8 FOREIGN KEY (stop_id) REFERENCES public.gtfs_stops(id);
ALTER TABLE ONLY public.gtfs_fare_rules
    ADD CONSTRAINT fk_rails_33e9869c97 FOREIGN KEY (route_id) REFERENCES public.gtfs_routes(id);
ALTER TABLE ONLY public.gtfs_stops
    ADD CONSTRAINT fk_rails_3a83952954 FOREIGN KEY (parent_station) REFERENCES public.gtfs_stops(id);
ALTER TABLE ONLY public.gtfs_calendars
    ADD CONSTRAINT fk_rails_42538db9b2 FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.feed_states
    ADD CONSTRAINT fk_rails_5189447149 FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_frequencies
    ADD CONSTRAINT fk_rails_6e6295037f FOREIGN KEY (trip_id) REFERENCES public.gtfs_trips(id);
ALTER TABLE ONLY public.tl_route_geometries
    ADD CONSTRAINT fk_rails_71ddc895e1 FOREIGN KEY (route_id) REFERENCES public.gtfs_routes(id);
ALTER TABLE ONLY public.tl_agency_places
    ADD CONSTRAINT fk_rails_736d85abf8 FOREIGN KEY (agency_id) REFERENCES public.gtfs_agencies(id);
ALTER TABLE ONLY public.tl_agency_places
    ADD CONSTRAINT fk_rails_782a6056d8 FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_calendar_dates
    ADD CONSTRAINT fk_rails_7a365f570b FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.tl_feed_version_geometries
    ADD CONSTRAINT fk_rails_8398615a04 FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_shapes
    ADD CONSTRAINT fk_rails_84a74e83d8 FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_stops
    ADD CONSTRAINT fk_rails_860ffa5a40 FOREIGN KEY (level_id) REFERENCES public.gtfs_levels(id);
ALTER TABLE ONLY public.tl_route_stops
    ADD CONSTRAINT fk_rails_86271126ad FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.tl_agency_geometries
    ADD CONSTRAINT fk_rails_8a1bd61db9 FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_fare_attributes
    ADD CONSTRAINT fk_rails_8a3ca847de FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_pathways
    ADD CONSTRAINT fk_rails_8d7bf46256 FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.tl_route_headways
    ADD CONSTRAINT fk_rails_93324ef20d FOREIGN KEY (selected_stop_id) REFERENCES public.gtfs_stops(id);
ALTER TABLE ONLY public.feed_states
    ADD CONSTRAINT fk_rails_99eaedcf98 FOREIGN KEY (feed_id) REFERENCES public.current_feeds(id);
ALTER TABLE ONLY public.tl_route_headways
    ADD CONSTRAINT fk_rails_9a487f871b FOREIGN KEY (route_id) REFERENCES public.gtfs_routes(id);
ALTER TABLE ONLY public.gtfs_transfers
    ADD CONSTRAINT fk_rails_a030c4a2a9 FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_routes
    ADD CONSTRAINT fk_rails_a5ff5a2ceb FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_pathways
    ADD CONSTRAINT fk_rails_a668e1e0ac FOREIGN KEY (to_stop_id) REFERENCES public.gtfs_stops(id);
ALTER TABLE ONLY public.gtfs_agencies
    ADD CONSTRAINT fk_rails_a7e0c4685b FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_trips
    ADD CONSTRAINT fk_rails_a839da033a FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_fare_attributes
    ADD CONSTRAINT fk_rails_b096f74e03 FOREIGN KEY (agency_id) REFERENCES public.gtfs_agencies(id);
ALTER TABLE ONLY public.feed_versions
    ADD CONSTRAINT fk_rails_b5365c3cf3 FOREIGN KEY (feed_id) REFERENCES public.current_feeds(id);
ALTER TABLE ONLY public.gtfs_stop_times_unpartitioned
    ADD CONSTRAINT fk_rails_b5a47190ac FOREIGN KEY (trip_id) REFERENCES public.gtfs_trips(id);
ALTER TABLE ONLY public.tl_route_geometries
    ADD CONSTRAINT fk_rails_b9fc0ae4ad FOREIGN KEY (shape_id) REFERENCES public.gtfs_shapes(id);
ALTER TABLE ONLY public.gtfs_fare_rules
    ADD CONSTRAINT fk_rails_bd7d178423 FOREIGN KEY (fare_id) REFERENCES public.gtfs_fare_attributes(id);
ALTER TABLE ONLY public.gtfs_fare_rules
    ADD CONSTRAINT fk_rails_c336ea9f1a FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_levels
    ADD CONSTRAINT fk_rails_c5fba46e47 FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.tl_route_geometries
    ADD CONSTRAINT fk_rails_c858a218e2 FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_calendar_dates
    ADD CONSTRAINT fk_rails_ca504bc01f FOREIGN KEY (service_id) REFERENCES public.gtfs_calendars(id);
ALTER TABLE ONLY public.tl_route_stops
    ADD CONSTRAINT fk_rails_cc9fde6bb7 FOREIGN KEY (stop_id) REFERENCES public.gtfs_stops(id);
ALTER TABLE ONLY public.gtfs_stops
    ADD CONSTRAINT fk_rails_cf4bc79180 FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_frequencies
    ADD CONSTRAINT fk_rails_d1b468024b FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_trips
    ADD CONSTRAINT fk_rails_d2c6f99d5e FOREIGN KEY (service_id) REFERENCES public.gtfs_calendars(id);
ALTER TABLE ONLY public.gtfs_pathways
    ADD CONSTRAINT fk_rails_df846a6b54 FOREIGN KEY (from_stop_id) REFERENCES public.gtfs_stops(id);
ALTER TABLE ONLY public.gtfs_transfers
    ADD CONSTRAINT fk_rails_e1c56f7da4 FOREIGN KEY (to_stop_id) REFERENCES public.gtfs_stops(id);
ALTER TABLE ONLY public.gtfs_routes
    ADD CONSTRAINT fk_rails_e5eb0f1573 FOREIGN KEY (agency_id) REFERENCES public.gtfs_agencies(id);
ALTER TABLE ONLY public.gtfs_feed_infos
    ADD CONSTRAINT fk_rails_eb863abbac FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_trips
    ADD CONSTRAINT fk_rails_mid93550f50 FOREIGN KEY (route_id) REFERENCES public.gtfs_routes(id);
ALTER TABLE ONLY public.gtfs_levels
    ADD CONSTRAINT gtfs_levels_parent_station_fkey FOREIGN KEY (parent_station) REFERENCES public.gtfs_stops(id);
ALTER TABLE public.gtfs_stop_times
    ADD CONSTRAINT gtfs_stop_times_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE public.gtfs_stop_times
    ADD CONSTRAINT gtfs_stop_times_stop_id_fkey FOREIGN KEY (stop_id) REFERENCES public.gtfs_stops(id);
ALTER TABLE public.gtfs_stop_times
    ADD CONSTRAINT gtfs_stop_times_trip_id_fkey FOREIGN KEY (trip_id) REFERENCES public.gtfs_trips(id);
ALTER TABLE ONLY public.tl_agency_onestop_ids
    ADD CONSTRAINT tl_agency_onestop_ids_agency_id_fkey FOREIGN KEY (agency_id) REFERENCES public.gtfs_agencies(id);
ALTER TABLE ONLY public.tl_agency_onestop_ids
    ADD CONSTRAINT tl_agency_onestop_ids_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.tl_census_fields
    ADD CONSTRAINT tl_census_fields_table_id_fkey FOREIGN KEY (table_id) REFERENCES public.tl_census_tables(id);
ALTER TABLE ONLY public.tl_census_geographies
    ADD CONSTRAINT tl_census_geographies_source_id_fkey FOREIGN KEY (source_id) REFERENCES public.tl_census_sources(id);
ALTER TABLE ONLY public.tl_census_sources
    ADD CONSTRAINT tl_census_sources_dataset_id_fkey FOREIGN KEY (dataset_id) REFERENCES public.tl_census_datasets(id);
ALTER TABLE ONLY public.tl_census_tables
    ADD CONSTRAINT tl_census_tables_dataset_id_fkey FOREIGN KEY (dataset_id) REFERENCES public.tl_census_datasets(id);
ALTER TABLE ONLY public.tl_census_values
    ADD CONSTRAINT tl_census_values_geography_id_fkey FOREIGN KEY (geography_id) REFERENCES public.tl_census_geographies(id);
ALTER TABLE ONLY public.tl_census_values
    ADD CONSTRAINT tl_census_values_table_id_fkey FOREIGN KEY (table_id) REFERENCES public.tl_census_tables(id);
ALTER TABLE ONLY public.tl_ext_fare_networks
    ADD CONSTRAINT tl_ext_fare_networks_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.tl_ext_gtfs_stops
    ADD CONSTRAINT tl_ext_gtfs_stops_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.tl_ext_gtfs_stops
    ADD CONSTRAINT tl_ext_gtfs_stops_id_fkey FOREIGN KEY (id) REFERENCES public.gtfs_stops(id);
ALTER TABLE ONLY public.tl_route_onestop_ids
    ADD CONSTRAINT tl_route_onestop_ids_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.tl_route_onestop_ids
    ADD CONSTRAINT tl_route_onestop_ids_route_id_fkey FOREIGN KEY (route_id) REFERENCES public.gtfs_routes(id);
ALTER TABLE ONLY public.tl_stop_external_references
    ADD CONSTRAINT tl_stop_external_references_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.tl_stop_external_references
    ADD CONSTRAINT tl_stop_external_references_id_fkey FOREIGN KEY (id) REFERENCES public.gtfs_stops(id);
ALTER TABLE ONLY public.tl_stop_onestop_ids
    ADD CONSTRAINT tl_stop_onestop_ids_feed_version_id_fkey FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.tl_stop_onestop_ids
    ADD CONSTRAINT tl_stop_onestop_ids_stop_id_fkey FOREIGN KEY (stop_id) REFERENCES public.gtfs_stops(id);
