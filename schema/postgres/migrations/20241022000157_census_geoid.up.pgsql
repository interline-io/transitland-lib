BEGIN;

ALTER TABLE tl_census_values DROP COLUMN geography_id;
ALTER TABLE tl_census_values ADD COLUMN geoid TEXT NOT NULL;
CREATE INDEX ON tl_census_values(geoid);

COMMIT;