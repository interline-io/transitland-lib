BEGIN;

ALTER TABLE tl_census_geographies DROP COLUMN geometry;
ALTER TABLE tl_census_geographies ADD COLUMN geometry geography(MultiPolygon, 4326);
CREATE INDEX tl_census_geographies_geometry_idx ON public.tl_census_geographies USING gist (geometry);

COMMIT;