BEGIN;

CREATE TABLE tl_validation_report_error_groups (
    id bigserial primary key NOT NULL,
    validation_report_id bigint REFERENCES tl_validation_reports(id) NOT NULL,
    filename text not null,
    field text not null,
    error_type text not null,
    error_code text not null,
    count integer not null
);
CREATE INDEX ON tl_validation_report_error_groups(validation_report_id);

CREATE TABLE tl_validation_report_error_exemplars (
    id bigserial primary key NOT NULL,
    validation_report_error_group_id bigint REFERENCES tl_validation_report_error_groups(id) not null,
    line int not null,
    entity_id text not null,
    value text not null,
    message text not null,
    geometries geography
);
CREATE INDEX ON tl_validation_report_error_exemplars(validation_report_error_group_id);

COMMIT;