BEGIN;

CREATE TABLE public.tl_feed_version_geohashes (
    feed_version_id bigint REFERENCES feed_versions(id) not null,
    geohash text not null,
    stop_count int not null,
    primary key (feed_version_id, geohash)
);

COMMIT;
