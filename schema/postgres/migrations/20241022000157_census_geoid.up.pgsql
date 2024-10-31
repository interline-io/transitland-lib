BEGIN;

ALTER TABLE tl_census_values DROP COLUMN geography_id;
ALTER TABLE tl_census_values ADD COLUMN geoid TEXT NOT NULL;
CREATE INDEX ON tl_census_values(geoid);
CREATE UNIQUE INDEX ON tl_census_values(geoid, table_id);

ALTER TABLE tl_census_values ADD COLUMN source_id bigint not null REFERENCES tl_census_sources(id);
CREATE INDEX ON tl_census_values(source_id);

ALTER TABLE tl_census_tables ADD COLUMN table_details text;

ALTER TABLE tl_census_fields ADD COLUMN column_order double precision;

COMMIT;