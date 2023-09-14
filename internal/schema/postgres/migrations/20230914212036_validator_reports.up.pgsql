BEGIN;

CREATE TABLE tl_validator_reports (
    id bigserial primary key not null,
    feed_version_id bigint REFERENCES feed_versions(id),
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    static_url text,
    static_sha1 text
);
CREATE INDEX ON tl_validator_reports(feed_version_id);


CREATE TABLE tl_validator_error_groups (
    id bigserial primary key not null,
    report_id bigint REFERENCES tl_validator_reports(id),
    filename text,
    error_type text,
    count int
);
CREATE INDEX ON tl_validator_error_groups(report_id);

CREATE TABLE tl_validator_errors (
    id bigserial primary key not null,
    error_group_id bigint REFERENCES tl_validator_error_groups(id),
    error text
);
CREATE INDEX ON tl_validator_errors(error_group_id);


COMMIT;