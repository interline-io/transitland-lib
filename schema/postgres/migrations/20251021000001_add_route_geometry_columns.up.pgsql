BEGIN;

-- Add geometry column to tl_materialized_active_routes for spatial search
-- Populated with ST_Simplify() in PostGIS, full geometry in SQLite
ALTER TABLE tl_materialized_active_routes 
    ADD COLUMN geometry_simplified geometry(LineString,4326);

-- Add spatial index for distance queries
CREATE INDEX ON tl_materialized_active_routes USING gist(geometry_simplified);

COMMIT;
