BEGIN;

ALTER TABLE tl_census_geographies ADD COLUMN adm0_name text;
ALTER TABLE tl_census_geographies ADD COLUMN adm0_iso text;
ALTER TABLE tl_census_geographies ADD COLUMN adm1_name text;
ALTER TABLE tl_census_geographies ADD COLUMN adm1_iso text;

COMMIT;