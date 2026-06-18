BEGIN;

CREATE TABLE public.gtfs_fare_leg_join_rules (
    id bigserial primary key not null,
    feed_version_id bigint NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,

    from_network_id text,
    to_network_id text,
    from_stop_id text,
    to_stop_id text
);
CREATE INDEX ON gtfs_fare_leg_join_rules(feed_version_id);
CREATE UNIQUE INDEX ON gtfs_fare_leg_join_rules(feed_version_id, from_network_id, to_network_id, from_stop_id, to_stop_id);

COMMIT;
