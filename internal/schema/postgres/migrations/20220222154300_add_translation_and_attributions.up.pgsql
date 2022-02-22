BEGIN;

ALTER TABLE gtfs_feed_infos ADD COLUMN default_lang text;
ALTER TABLE gtfs_feed_infos ADD COLUMN feed_contact_email text;
ALTER TABLE gtfs_feed_infos ADD COLUMN feed_contact_url text;

ALTER TABLE gtfs_routes ADD COLUMN continuous_pickup int;
ALTER TABLE gtfs_routes ADD COLUMN continuous_drop_off int;

ALTER TABLE gtfs_stops ADD COLUMN tts_stop_name text;
ALTER TABLE gtfs_stops ADD COLUMN platform_code text;

ALTER TABLE gtfs_stop_times ADD COLUMN continuous_pickup int;
ALTER TABLE gtfs_stop_times ADD COLUMN continuous_drop_off int;

CREATE TABLE gtfs_translations (
    id bigserial primary key,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL REFERENCES feed_versions(id),
    table_name text not null,
    field_name text,
    language text,
    translation text,
    record_id text,
    record_sub_id text
);
CREATE INDEX ON gtfs_translations(feed_version_id);

CREATE TABLE gtfs_attributions (
    id bigserial primary key,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    feed_version_id bigint NOT NULL REFERENCES feed_versions(id),
    organization_name text not null,
    agency_id bigint REFERENCES gtfs_stops(id),
    route_id bigint REFERENCES gtfs_routes(id),
    trip_id bigint REFERENCES gtfs_trips(id),
    is_producer int,
    is_operator int,
    attribution_id text,
    attribution_url text,
    attribution_email text,
    attribution_phone text
);
CREATE INDEX ON gtfs_attributions(feed_version_id);


COMMIT;