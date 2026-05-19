BEGIN;

CREATE TABLE public.tl_feed_version_geohashes (
    id bigserial primary key not null,
    feed_version_id bigint REFERENCES feed_versions(id) not null,
    geohash text not null,
    stop_count int not null
);

CREATE UNIQUE INDEX tl_feed_version_geohashes_feed_version_id_geohash_idx
    ON public.tl_feed_version_geohashes (feed_version_id, geohash);

CREATE INDEX tl_feed_version_geohashes_geohash_idx
    ON public.tl_feed_version_geohashes (geohash);

COMMIT;
