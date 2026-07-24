-- Drop the old covering btree on onestop_id, superseded by the hash index in the
-- previous migration -- the same change made to feed_version_stop_onestop_ids,
-- where the INCLUDE payload was shown never to deliver index-only scans. Concurrent
-- drop, in its own single-statement migration: DROP INDEX CONCURRENTLY cannot run
-- inside a transaction block.
DROP INDEX CONCURRENTLY IF EXISTS feed_version_route_onestop_id_onestop_id_entity_id_feed_ver_idx;
