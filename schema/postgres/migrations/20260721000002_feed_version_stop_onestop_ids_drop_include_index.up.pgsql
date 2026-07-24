-- Drop the old covering btree on onestop_id, superseded by the hash index in the
-- previous migration. Its INCLUDE (entity_id, feed_version_id) payload never
-- delivered index-only scans in practice -- the table churns constantly, so its
-- visibility map stays stale and lookups fetch from the heap anyway. Concurrent
-- drop, in its own single-statement migration: DROP INDEX CONCURRENTLY cannot run
-- inside a transaction block.
DROP INDEX CONCURRENTLY IF EXISTS feed_version_stop_onestop_ids_onestop_id_entity_id_feed_ver_idx;
