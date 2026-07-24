-- feed_version_stop_onestop_ids is looked up only by exact onestop_id equality
-- (AllowPrevious resolution), so a hash index fits: it stores just the hash of
-- the long onestop_id key, landing at roughly half the size of the covering
-- btree it replaces (see the next migration) with equivalent lookup cost.
-- Concurrent build, in its own single-statement migration: CREATE INDEX
-- CONCURRENTLY cannot run inside a transaction block.
CREATE INDEX CONCURRENTLY IF NOT EXISTS feed_version_stop_onestop_ids_onestop_id_idx
    ON feed_version_stop_onestop_ids USING hash (onestop_id);
