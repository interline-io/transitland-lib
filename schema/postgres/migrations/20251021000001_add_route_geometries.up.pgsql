BEGIN;

-- Add geometry columns to materialized active routes table
ALTER TABLE tl_materialized_active_routes 
  ADD COLUMN geometry_centroid geography(Point, 4326),
  ADD COLUMN geometry_simplified geography(LineString, 4326);

-- Create spatial indexes for distance ordering and spatial queries
CREATE INDEX ON tl_materialized_active_routes USING GIST(geometry_centroid);

CREATE INDEX ON tl_materialized_active_routes USING GIST(geometry_simplified);

-- Populate existing rows (if any) with geometry data from tl_route_geometries
UPDATE tl_materialized_active_routes 
SET 
  geometry_centroid = (
    SELECT ST_Centroid(geometry) 
    FROM tl_route_geometries 
    WHERE route_id = tl_materialized_active_routes.id 
    LIMIT 1
  ),
  geometry_simplified = (
    SELECT ST_Simplify(geometry::geometry, 0.01)
    FROM tl_route_geometries 
    WHERE route_id = tl_materialized_active_routes.id 
    LIMIT 1
  );

COMMIT;
