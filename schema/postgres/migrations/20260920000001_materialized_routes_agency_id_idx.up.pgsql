-- Index tl_materialized_active_routes by agency.
--
-- The column is materialized but was never indexed, so a routes query filtered by
-- agency_ids or operator_onestop_id had no way into the table by agency and read all
-- of it. Agency.routes was unaffected: it reads gtfs_routes, which is indexed.
BEGIN;

CREATE INDEX IF NOT EXISTS tl_materialized_active_routes_agency_id_idx ON tl_materialized_active_routes (agency_id);

COMMIT;
