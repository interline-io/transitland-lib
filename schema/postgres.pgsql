CREATE EXTENSION postgis;
CREATE EXTENSION hstore;
CREATE TABLE public.current_feeds (
    id integer NOT NULL,
    onestop_id character varying NOT NULL,
    url character varying NOT NULL,
    spec character varying DEFAULT 'gtfs'::character varying NOT NULL,
    tags public.hstore,
    last_fetched_at timestamp without time zone,
    last_imported_at timestamp without time zone,
    license_name character varying,
    license_url character varying,
    license_use_without_attribution character varying,
    license_create_derived_product character varying,
    license_redistribute character varying,
    version integer,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    created_or_updated_in_changeset_id integer,
    geometry public.geography(Geometry,4326),
    license_attribution_text text,
    active_feed_version_id integer,
    edited_attributes character varying[] DEFAULT '{}'::character varying[],
    name character varying,
    type character varying,
    auth jsonb DEFAULT '{}'::jsonb NOT NULL,
    urls jsonb DEFAULT '{}'::jsonb NOT NULL,
    last_successful_fetch_at timestamp without time zone,
    deleted_at timestamp without time zone,
    last_fetch_error character varying NOT NULL,
    license jsonb DEFAULT '{}'::jsonb NOT NULL,
    other_ids jsonb DEFAULT '{}'::jsonb NOT NULL,
    associated_feeds jsonb DEFAULT '{}'::jsonb NOT NULL,
    languages jsonb DEFAULT '{}'::jsonb NOT NULL,
    feed_namespace_id character varying DEFAULT ''::character varying NOT NULL
);
CREATE SEQUENCE public.current_feeds_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.current_feeds_id_seq OWNED BY public.current_feeds.id;
CREATE TABLE public.feed_version_imports (
    id integer NOT NULL,
    feed_version_id integer,
    created_at timestamp without time zone,
    updated_at timestamp without time zone,
    success boolean,
    import_log text,
    exception_log text,
    validation_report text,
    import_level integer,
    operators_in_feed json,
    in_progress boolean DEFAULT false NOT NULL
);
CREATE SEQUENCE public.feed_version_imports_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.feed_version_imports_id_seq OWNED BY public.feed_version_imports.id;
CREATE TABLE public.feed_versions (
    id integer NOT NULL,
    feed_id integer NOT NULL,
    feed_type character varying NOT NULL,
    file character varying NOT NULL,
    earliest_calendar_date date NOT NULL,
    latest_calendar_date date NOT NULL,
    sha1 character varying NOT NULL,
    tags public.hstore,
    fetched_at timestamp without time zone NOT NULL,
    imported_at timestamp without time zone,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    import_level integer DEFAULT 0 NOT NULL,
    url character varying NOT NULL,
    file_raw character varying,
    sha1_raw character varying,
    file_feedvalidator character varying,
    deleted_at timestamp without time zone
);
CREATE SEQUENCE public.feed_versions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.feed_versions_id_seq OWNED BY public.feed_versions.id;
CREATE TABLE public.gtfs_agencies (
    id integer NOT NULL,
    agency_id character varying NOT NULL,
    agency_name character varying NOT NULL,
    agency_url character varying NOT NULL,
    agency_timezone character varying NOT NULL,
    agency_lang character varying NOT NULL,
    agency_phone character varying NOT NULL,
    agency_fare_url character varying NOT NULL,
    agency_email character varying NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL
);
CREATE SEQUENCE public.gtfs_agencies_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_agencies_id_seq OWNED BY public.gtfs_agencies.id;
CREATE TABLE public.gtfs_calendar_dates (
    id integer NOT NULL,
    date date NOT NULL,
    exception_type integer NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL,
    service_id integer NOT NULL
);
CREATE SEQUENCE public.gtfs_calendar_dates_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_calendar_dates_id_seq OWNED BY public.gtfs_calendar_dates.id;
CREATE TABLE public.gtfs_calendars (
    id integer NOT NULL,
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
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL,
    generated boolean NOT NULL
);
CREATE SEQUENCE public.gtfs_calendars_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_calendars_id_seq OWNED BY public.gtfs_calendars.id;
CREATE TABLE public.gtfs_fare_attributes (
    id integer NOT NULL,
    fare_id character varying NOT NULL,
    price double precision NOT NULL,
    currency_type character varying NOT NULL,
    payment_method integer NOT NULL,
    transfer_duration integer NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL,
    agency_id integer,
    transfers integer NOT NULL
);
CREATE SEQUENCE public.gtfs_fare_attributes_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_fare_attributes_id_seq OWNED BY public.gtfs_fare_attributes.id;
CREATE TABLE public.gtfs_fare_rules (
    id integer NOT NULL,
    origin_id character varying NOT NULL,
    destination_id character varying NOT NULL,
    contains_id character varying NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL,
    route_id integer,
    fare_id integer
);
CREATE SEQUENCE public.gtfs_fare_rules_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_fare_rules_id_seq OWNED BY public.gtfs_fare_rules.id;
CREATE TABLE public.gtfs_feed_infos (
    id integer NOT NULL,
    feed_publisher_name character varying NOT NULL,
    feed_publisher_url character varying NOT NULL,
    feed_lang character varying NOT NULL,
    feed_start_date date,
    feed_end_date date,
    feed_version_name character varying NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL
);
CREATE SEQUENCE public.gtfs_feed_infos_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_feed_infos_id_seq OWNED BY public.gtfs_feed_infos.id;
CREATE TABLE public.gtfs_frequencies (
    id integer NOT NULL,
    start_time integer NOT NULL,
    end_time integer NOT NULL,
    headway_secs integer NOT NULL,
    exact_times integer NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL,
    trip_id integer NOT NULL
);
CREATE SEQUENCE public.gtfs_frequencies_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_frequencies_id_seq OWNED BY public.gtfs_frequencies.id;
CREATE TABLE public.gtfs_imports (
    id integer NOT NULL,
    succeeded boolean NOT NULL,
    import_log text NOT NULL,
    exception_log text NOT NULL,
    import_level integer NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL
);
CREATE SEQUENCE public.gtfs_imports_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_imports_id_seq OWNED BY public.gtfs_imports.id;
CREATE TABLE public.gtfs_routes (
    id integer NOT NULL,
    route_id character varying NOT NULL,
    route_short_name character varying NOT NULL,
    route_long_name character varying NOT NULL,
    route_desc character varying NOT NULL,
    route_type integer NOT NULL,
    route_url character varying NOT NULL,
    route_color character varying NOT NULL,
    route_text_color character varying NOT NULL,
    route_sort_order integer NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL,
    agency_id integer NOT NULL
);
CREATE SEQUENCE public.gtfs_routes_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_routes_id_seq OWNED BY public.gtfs_routes.id;
CREATE TABLE public.gtfs_shapes (
    id integer NOT NULL,
    shape_id character varying NOT NULL,
    generated boolean DEFAULT false NOT NULL,
    geometry public.geography(LineStringM,4326) NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL
);
CREATE SEQUENCE public.gtfs_shapes_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_shapes_id_seq OWNED BY public.gtfs_shapes.id;
CREATE TABLE public.gtfs_stop_times (
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
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL,
    trip_id integer NOT NULL,
    stop_id integer NOT NULL
);
CREATE SEQUENCE public.gtfs_stop_times_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_stop_times_id_seq OWNED BY public.gtfs_stop_times.id;
CREATE TABLE public.gtfs_stops (
    id integer NOT NULL,
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
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL,
    parent_station integer,
    level_id character varying NOT NULL
);
CREATE SEQUENCE public.gtfs_stops_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_stops_id_seq OWNED BY public.gtfs_stops.id;
CREATE TABLE public.gtfs_transfers (
    id integer NOT NULL,
    transfer_type integer NOT NULL,
    min_transfer_time integer NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL,
    from_stop_id integer NOT NULL,
    to_stop_id integer NOT NULL
);
CREATE SEQUENCE public.gtfs_transfers_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_transfers_id_seq OWNED BY public.gtfs_transfers.id;
CREATE TABLE public.gtfs_trips (
    id integer NOT NULL,
    trip_id character varying NOT NULL,
    trip_headsign character varying NOT NULL,
    trip_short_name character varying NOT NULL,
    direction_id integer NOT NULL,
    block_id character varying NOT NULL,
    wheelchair_accessible integer NOT NULL,
    bikes_allowed integer NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    feed_version_id integer NOT NULL,
    route_id integer NOT NULL,
    shape_id integer,
    stop_pattern_id integer NOT NULL,
    service_id integer NOT NULL
);
CREATE SEQUENCE public.gtfs_trips_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.gtfs_trips_id_seq OWNED BY public.gtfs_trips.id;
ALTER TABLE ONLY public.current_feeds ALTER COLUMN id SET DEFAULT nextval('public.current_feeds_id_seq'::regclass);
ALTER TABLE ONLY public.feed_version_imports ALTER COLUMN id SET DEFAULT nextval('public.feed_version_imports_id_seq'::regclass);
ALTER TABLE ONLY public.feed_versions ALTER COLUMN id SET DEFAULT nextval('public.feed_versions_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_agencies ALTER COLUMN id SET DEFAULT nextval('public.gtfs_agencies_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_calendar_dates ALTER COLUMN id SET DEFAULT nextval('public.gtfs_calendar_dates_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_calendars ALTER COLUMN id SET DEFAULT nextval('public.gtfs_calendars_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_fare_attributes ALTER COLUMN id SET DEFAULT nextval('public.gtfs_fare_attributes_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_fare_rules ALTER COLUMN id SET DEFAULT nextval('public.gtfs_fare_rules_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_feed_infos ALTER COLUMN id SET DEFAULT nextval('public.gtfs_feed_infos_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_frequencies ALTER COLUMN id SET DEFAULT nextval('public.gtfs_frequencies_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_imports ALTER COLUMN id SET DEFAULT nextval('public.gtfs_imports_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_routes ALTER COLUMN id SET DEFAULT nextval('public.gtfs_routes_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_shapes ALTER COLUMN id SET DEFAULT nextval('public.gtfs_shapes_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_stop_times ALTER COLUMN id SET DEFAULT nextval('public.gtfs_stop_times_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_stops ALTER COLUMN id SET DEFAULT nextval('public.gtfs_stops_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_transfers ALTER COLUMN id SET DEFAULT nextval('public.gtfs_transfers_id_seq'::regclass);
ALTER TABLE ONLY public.gtfs_trips ALTER COLUMN id SET DEFAULT nextval('public.gtfs_trips_id_seq'::regclass);
ALTER TABLE ONLY public.current_feeds
    ADD CONSTRAINT current_feeds_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.feed_version_imports
    ADD CONSTRAINT feed_version_imports_pkey PRIMARY KEY (id);
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
ALTER TABLE ONLY public.gtfs_imports
    ADD CONSTRAINT gtfs_imports_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_routes
    ADD CONSTRAINT gtfs_routes_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_shapes
    ADD CONSTRAINT gtfs_shapes_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_stop_times
    ADD CONSTRAINT gtfs_stop_times_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_stops
    ADD CONSTRAINT gtfs_stops_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_transfers
    ADD CONSTRAINT gtfs_transfers_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gtfs_trips
    ADD CONSTRAINT gtfs_trips_pkey PRIMARY KEY (id);
CREATE INDEX index_current_feeds_on_active_feed_version_id ON public.current_feeds USING btree (active_feed_version_id);
CREATE INDEX index_current_feeds_on_auth ON public.current_feeds USING btree (auth);
CREATE INDEX index_current_feeds_on_created_or_updated_in_changeset_id ON public.current_feeds USING btree (created_or_updated_in_changeset_id);
CREATE INDEX index_current_feeds_on_geometry ON public.current_feeds USING gist (geometry);
CREATE UNIQUE INDEX index_current_feeds_on_onestop_id ON public.current_feeds USING btree (onestop_id);
CREATE INDEX index_current_feeds_on_urls ON public.current_feeds USING btree (urls);
CREATE UNIQUE INDEX index_feed_version_imports_on_feed_version_id ON public.feed_version_imports USING btree (feed_version_id);
CREATE INDEX index_feed_versions_on_earliest_calendar_date ON public.feed_versions USING btree (earliest_calendar_date);
CREATE INDEX index_feed_versions_on_feed_type_and_feed_id ON public.feed_versions USING btree (feed_type, feed_id);
CREATE INDEX index_feed_versions_on_latest_calendar_date ON public.feed_versions USING btree (latest_calendar_date);
CREATE INDEX index_gtfs_agencies_on_agency_id ON public.gtfs_agencies USING btree (agency_id);
CREATE INDEX index_gtfs_agencies_on_agency_name ON public.gtfs_agencies USING btree (agency_name);
CREATE INDEX index_gtfs_agencies_on_feed_version_id ON public.gtfs_agencies USING btree (feed_version_id);
CREATE UNIQUE INDEX index_gtfs_agencies_unique ON public.gtfs_agencies USING btree (feed_version_id, agency_id);
CREATE INDEX index_gtfs_calendar_dates_on_date ON public.gtfs_calendar_dates USING btree (date);
CREATE INDEX index_gtfs_calendar_dates_on_exception_type ON public.gtfs_calendar_dates USING btree (exception_type);
CREATE INDEX index_gtfs_calendar_dates_on_feed_version_id ON public.gtfs_calendar_dates USING btree (feed_version_id);
CREATE INDEX index_gtfs_calendar_dates_on_service_id ON public.gtfs_calendar_dates USING btree (service_id);
CREATE INDEX index_gtfs_calendars_on_end_date ON public.gtfs_calendars USING btree (end_date);
CREATE INDEX index_gtfs_calendars_on_feed_version_id ON public.gtfs_calendars USING btree (feed_version_id);
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
CREATE INDEX index_gtfs_fare_attributes_on_feed_version_id ON public.gtfs_fare_attributes USING btree (feed_version_id);
CREATE UNIQUE INDEX index_gtfs_fare_attributes_unique ON public.gtfs_fare_attributes USING btree (feed_version_id, fare_id);
CREATE INDEX index_gtfs_fare_rules_on_fare_id ON public.gtfs_fare_rules USING btree (fare_id);
CREATE INDEX index_gtfs_fare_rules_on_feed_version_id ON public.gtfs_fare_rules USING btree (feed_version_id);
CREATE INDEX index_gtfs_fare_rules_on_route_id ON public.gtfs_fare_rules USING btree (route_id);
CREATE UNIQUE INDEX index_gtfs_feed_info_unique ON public.gtfs_feed_infos USING btree (feed_version_id);
CREATE INDEX index_gtfs_feed_infos_on_feed_version_id ON public.gtfs_feed_infos USING btree (feed_version_id);
CREATE INDEX index_gtfs_frequencies_on_feed_version_id ON public.gtfs_frequencies USING btree (feed_version_id);
CREATE INDEX index_gtfs_frequencies_on_trip_id ON public.gtfs_frequencies USING btree (trip_id);
CREATE INDEX index_gtfs_imports_on_feed_version_id ON public.gtfs_imports USING btree (feed_version_id);
CREATE INDEX index_gtfs_imports_on_succeeded ON public.gtfs_imports USING btree (succeeded);
CREATE INDEX index_gtfs_routes_on_agency_id ON public.gtfs_routes USING btree (agency_id);
CREATE INDEX index_gtfs_routes_on_feed_version_id ON public.gtfs_routes USING btree (feed_version_id);
CREATE INDEX index_gtfs_routes_on_route_desc ON public.gtfs_routes USING btree (route_desc);
CREATE INDEX index_gtfs_routes_on_route_id ON public.gtfs_routes USING btree (route_id);
CREATE INDEX index_gtfs_routes_on_route_long_name ON public.gtfs_routes USING btree (route_long_name);
CREATE INDEX index_gtfs_routes_on_route_short_name ON public.gtfs_routes USING btree (route_short_name);
CREATE INDEX index_gtfs_routes_on_route_type ON public.gtfs_routes USING btree (route_type);
CREATE UNIQUE INDEX index_gtfs_routes_unique ON public.gtfs_routes USING btree (feed_version_id, route_id);
CREATE INDEX index_gtfs_shapes_on_feed_version_id ON public.gtfs_shapes USING btree (feed_version_id);
CREATE INDEX index_gtfs_shapes_on_generated ON public.gtfs_shapes USING btree (generated);
CREATE INDEX index_gtfs_shapes_on_geometry ON public.gtfs_shapes USING gist (geometry);
CREATE INDEX index_gtfs_shapes_on_shape_id ON public.gtfs_shapes USING btree (shape_id);
CREATE UNIQUE INDEX index_gtfs_shapes_unique ON public.gtfs_shapes USING btree (feed_version_id, shape_id);
CREATE INDEX index_gtfs_stop_times_on_arrival_time ON public.gtfs_stop_times USING btree (arrival_time);
CREATE INDEX index_gtfs_stop_times_on_departure_time ON public.gtfs_stop_times USING btree (departure_time);
CREATE INDEX index_gtfs_stop_times_on_feed_version_id ON public.gtfs_stop_times USING btree (feed_version_id);
CREATE INDEX index_gtfs_stop_times_on_stop_id ON public.gtfs_stop_times USING btree (stop_id);
CREATE INDEX index_gtfs_stop_times_on_trip_id ON public.gtfs_stop_times USING btree (trip_id);
CREATE UNIQUE INDEX index_gtfs_stop_times_unique ON public.gtfs_stop_times USING btree (feed_version_id, trip_id, stop_sequence);
CREATE INDEX index_gtfs_stops_on_feed_version_id ON public.gtfs_stops USING btree (feed_version_id);
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
CREATE INDEX index_gtfs_trips_on_feed_version_id ON public.gtfs_trips USING btree (feed_version_id);
CREATE INDEX index_gtfs_trips_on_route_id ON public.gtfs_trips USING btree (route_id);
CREATE INDEX index_gtfs_trips_on_service_id ON public.gtfs_trips USING btree (service_id);
CREATE INDEX index_gtfs_trips_on_shape_id ON public.gtfs_trips USING btree (shape_id);
CREATE INDEX index_gtfs_trips_on_trip_headsign ON public.gtfs_trips USING btree (trip_headsign);
CREATE INDEX index_gtfs_trips_on_trip_id ON public.gtfs_trips USING btree (trip_id);
CREATE INDEX index_gtfs_trips_on_trip_short_name ON public.gtfs_trips USING btree (trip_short_name);
CREATE UNIQUE INDEX index_gtfs_trips_unique ON public.gtfs_trips USING btree (feed_version_id, trip_id);
ALTER TABLE ONLY public.gtfs_transfers
    ADD CONSTRAINT fk_rails_0cc6ff288a FOREIGN KEY (from_stop_id) REFERENCES public.gtfs_stops(id);
ALTER TABLE ONLY public.gtfs_stop_times
    ADD CONSTRAINT fk_rails_22a671077b FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_stop_times
    ADD CONSTRAINT fk_rails_30ced0baa8 FOREIGN KEY (stop_id) REFERENCES public.gtfs_stops(id);
ALTER TABLE ONLY public.gtfs_calendars
    ADD CONSTRAINT fk_rails_42538db9b2 FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_trips
    ADD CONSTRAINT fk_rails_5093550f50 FOREIGN KEY (route_id) REFERENCES public.gtfs_routes(id);
ALTER TABLE ONLY public.gtfs_frequencies
    ADD CONSTRAINT fk_rails_6e6295037f FOREIGN KEY (trip_id) REFERENCES public.gtfs_trips(id);
ALTER TABLE ONLY public.gtfs_calendar_dates
    ADD CONSTRAINT fk_rails_7a365f570b FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_fare_attributes
    ADD CONSTRAINT fk_rails_8a3ca847de FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_transfers
    ADD CONSTRAINT fk_rails_a030c4a2a9 FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_routes
    ADD CONSTRAINT fk_rails_a5ff5a2ceb FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_agencies
    ADD CONSTRAINT fk_rails_a7e0c4685b FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_trips
    ADD CONSTRAINT fk_rails_a839da033a FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.feed_version_imports
    ADD CONSTRAINT fk_rails_b3ec636673 FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_stop_times
    ADD CONSTRAINT fk_rails_b5a47190ac FOREIGN KEY (trip_id) REFERENCES public.gtfs_trips(id);
ALTER TABLE ONLY public.gtfs_fare_rules
    ADD CONSTRAINT fk_rails_bd7d178423 FOREIGN KEY (fare_id) REFERENCES public.gtfs_fare_attributes(id);
ALTER TABLE ONLY public.gtfs_fare_rules
    ADD CONSTRAINT fk_rails_c336ea9f1a FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_calendar_dates
    ADD CONSTRAINT fk_rails_ca504bc01f FOREIGN KEY (service_id) REFERENCES public.gtfs_calendars(id);
ALTER TABLE ONLY public.gtfs_stops
    ADD CONSTRAINT fk_rails_cf4bc79180 FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_frequencies
    ADD CONSTRAINT fk_rails_d1b468024b FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
ALTER TABLE ONLY public.gtfs_trips
    ADD CONSTRAINT fk_rails_d2c6f99d5e FOREIGN KEY (service_id) REFERENCES public.gtfs_calendars(id);
ALTER TABLE ONLY public.gtfs_transfers
    ADD CONSTRAINT fk_rails_e1c56f7da4 FOREIGN KEY (to_stop_id) REFERENCES public.gtfs_stops(id);
ALTER TABLE ONLY public.gtfs_routes
    ADD CONSTRAINT fk_rails_e5eb0f1573 FOREIGN KEY (agency_id) REFERENCES public.gtfs_agencies(id);
ALTER TABLE ONLY public.gtfs_feed_infos
    ADD CONSTRAINT fk_rails_eb863abbac FOREIGN KEY (feed_version_id) REFERENCES public.feed_versions(id);
