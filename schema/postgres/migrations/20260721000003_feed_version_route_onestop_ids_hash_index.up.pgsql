-- Same equality-only AllowPrevious access as feed_version_stop_onestop_ids: a
-- hash index on onestop_id replaces the covering btree at roughly half the size.
-- Concurrent build, in its own single-statement migration: CREATE INDEX
-- CONCURRENTLY cannot run inside a transaction block.
CREATE INDEX CONCURRENTLY IF NOT EXISTS feed_version_route_onestop_ids_onestop_id_idx
    ON feed_version_route_onestop_ids USING hash (onestop_id);
