BEGIN;

CREATE TABLE tl_validation_reports (
    id bigserial primary key NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    reported_at timestamp without time zone NOT NULL,
    feed_version_id bigint REFERENCES feed_versions(id) NOT NULL
);
CREATE INDEX ON tl_validation_reports(feed_version_id);

CREATE TABLE tl_validation_trip_update_stats (
    id bigserial primary key NOT NULL,
    validation_report_id bigint REFERENCES tl_validation_reports(id) NOT NULL,
    agency_id text NOT NULL,
    route_id text NOT NULL,
    trip_scheduled_ids jsonb,
    trip_scheduled_count int NOT NULL,
    trip_match_count int NOT NULL
);
CREATE INDEX ON tl_validation_trip_update_stats(validation_report_id);

CREATE TABLE tl_validation_vehicle_position_stats (
    id bigserial primary key NOT NULL,
    validation_report_id bigint REFERENCES tl_validation_reports(id) NOT NULL,
    agency_id text NOT NULL,
    route_id text NOT NULL,
    trip_scheduled_ids jsonb,
    trip_scheduled_count int NOT NULL,
    trip_match_count int NOT NULL
);
CREATE INDEX ON tl_validation_vehicle_position_stats(validation_report_id);


COMMIT; 