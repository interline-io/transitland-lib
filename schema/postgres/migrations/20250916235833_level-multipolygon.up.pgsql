BEGIN;
-- 1. Rename the old column
ALTER TABLE gtfs_levels
    RENAME COLUMN geometry TO geometry_old;
-- 2. Add the new column as MultiPolygon
ALTER TABLE gtfs_levels
    ADD COLUMN geometry geography(MultiPolygon, 4326);
-- 3. Convert Polygon to MultiPolygon and copy data
UPDATE gtfs_levels
    SET geometry = CASE
        WHEN GeometryType(geometry_old) = 'MULTIPOLYGON' THEN geometry_old
        WHEN GeometryType(geometry_old) = 'POLYGON' THEN ST_Multi(geometry_old::geometry)::geography
        ELSE NULL
    END;
-- 4. Drop the old column
ALTER TABLE gtfs_levels DROP COLUMN geometry_old;
COMMIT;