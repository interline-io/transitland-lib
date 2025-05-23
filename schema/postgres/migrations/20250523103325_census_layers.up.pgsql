BEGIN;

ALTER TABLE tl_census_datasets rename column dataset_name to name;
ALTER TABLE tl_census_datasets add column description text;
ALTER TABLE tl_census_datasets alter column year_min drop not null;
ALTER TABLE tl_census_datasets alter column year_max drop not null;
ALTER TABLE tl_census_datasets alter column url drop not null;

ALTER TABLE tl_census_sources rename column source_name to name;
ALTER TABLE tl_census_sources add column description text;

CREATE TABLE tl_census_layers (
    id bigserial PRIMARY KEY,
    dataset_id bigint REFERENCES tl_census_datasets(id) not null,
    name text NOT NULL,
    description text,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL
);
CREATE INDEX ON tl_census_layers (name);
CREATE INDEX ON tl_census_layers (dataset_id);

ALTER TABLE tl_census_geographies ADD COLUMN layer_id bigint REFERENCES tl_census_layers(id);
ALTER TABLE tl_census_geographies DROP COLUMN layer_name;
CREATE INDEX ON tl_census_geographies (layer_id);

COMMIT;