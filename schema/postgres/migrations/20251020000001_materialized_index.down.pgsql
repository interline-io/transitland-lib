BEGIN;

-- Drop materialized index tables in reverse order
DROP TABLE IF EXISTS tl_materialized_index_state;
DROP TABLE IF EXISTS tl_materialized_active_agencies;
DROP TABLE IF EXISTS tl_materialized_active_stops;
DROP TABLE IF EXISTS tl_materialized_active_routes;

COMMIT;