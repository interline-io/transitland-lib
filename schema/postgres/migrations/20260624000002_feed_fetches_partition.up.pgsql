BEGIN;

-- New partitioned feed_fetches, built alongside the existing table. The next migration
-- (feed_fetches_changeover) swaps it in for feed_fetches.
--
-- Layout: LIST (url_type) at the top.
--   static_*                 -> one leaf, kept forever, NOT time-partitioned.
--   the three realtime_*     -> feed_fetches_rt, sub-partitioned RANGE (fetched_at) monthly.
--   gbfs_auto_discovery      -> feed_fetches_gbfs, sub-partitioned RANGE (fetched_at) monthly.
--                               old months of both are dropped by the cull job.
--   anything else            -> DEFAULT leaf, so an unexpected url_type can never fail to route.
--
-- fetched_at becomes NOT NULL here: it is a partition key and part of the primary key.
-- The writer always sets it (RTFetch/StaticFetch default it to now); the backfill
-- COALESCEs legacy NULLs to created_at.

CREATE TABLE feed_fetches_new (
    id                     bigint  NOT NULL DEFAULT nextval('feed_fetches_id_seq'::regclass),
    feed_id                bigint  NOT NULL,
    url_type               text    NOT NULL,
    url                    text    NOT NULL,
    success                boolean NOT NULL,
    fetched_at             timestamp without time zone NOT NULL,
    fetch_error            text,
    response_size          integer,
    response_code          integer,
    response_sha1          text,
    feed_version_id        bigint,
    created_at             timestamp without time zone NOT NULL DEFAULT now(),
    updated_at             timestamp without time zone NOT NULL DEFAULT now(),
    response_ttfb_ms       integer,
    response_time_ms       integer,
    validation_duration_ms integer,
    upload_duration_ms     integer,
    storage_key            text,
    -- A PK on a partitioned table must contain every partition-key column, at every level.
    PRIMARY KEY (id, url_type, fetched_at),
    FOREIGN KEY (feed_id)         REFERENCES current_feeds(id),
    FOREIGN KEY (feed_version_id) REFERENCES feed_versions(id)
) PARTITION BY LIST (url_type);

-- Static: kept forever, one leaf for every static_* type.
CREATE TABLE feed_fetches_static PARTITION OF feed_fetches_new
    FOR VALUES IN ('static_current', 'static_planned', 'static_historic');

-- Realtime (all three types) and GBFS: month-partitioned; month leaves are created
-- by feed_fetches_add_month().
CREATE TABLE feed_fetches_rt PARTITION OF feed_fetches_new
    FOR VALUES IN ('realtime_alerts', 'realtime_trip_updates', 'realtime_vehicle_positions')
    PARTITION BY RANGE (fetched_at);
CREATE TABLE feed_fetches_gbfs PARTITION OF feed_fetches_new
    FOR VALUES IN ('gbfs_auto_discovery') PARTITION BY RANGE (fetched_at);

-- LIST catch-all for unexpected/future url_types.
CREATE TABLE feed_fetches_unknown PARTITION OF feed_fetches_new DEFAULT;

-- RANGE backstop under each month-partitioned subtree: rows land here when no month
-- partition covers their fetched_at (e.g. before the buffer is seeded). Keep these
-- empty in production by seeding ahead, or the matching month partition can't be created.
CREATE TABLE feed_fetches_rt_default PARTITION OF feed_fetches_rt DEFAULT;
CREATE TABLE feed_fetches_gbfs_default PARTITION OF feed_fetches_gbfs DEFAULT;

-- Indexes are declared on the parent and propagate to all current/future partitions.
-- Primary per-feed recency access pattern (scheduler, latest-per-feed).
CREATE INDEX ON feed_fetches_new (feed_id, fetched_at) INCLUDE (success);
-- Feed-version lookups (dbfinder).
CREATE INDEX ON feed_fetches_new (feed_version_id);
-- Dropped vs. the old table: the standalone (fetched_at) and (success) indexes and the
-- two partial (feed_id, fetched_at) WHERE success indexes. Range pruning plus the
-- INCLUDE (success) composite above cover their cases; re-add if a plan regresses.

COMMIT;
