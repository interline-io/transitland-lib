BEGIN;

CREATE TABLE feed_version_service_windows (
    id bigserial primary key,
    feed_version_id bigint REFERENCES feed_versions(id),
    feed_start_date date,
    feed_end_date date,
    earliest_calendar_date date,
    latest_calendar_date date,
    fallback_week date,
    default_timezone text,
    created_at timestamp without time zone DEFAULT NOW() NOT NULL,
    updated_at timestamp without time zone DEFAULT NOW() NOT NULL
);

CREATE UNIQUE INDEX ON feed_version_service_windows(feed_version_id);
CREATE INDEX ON feed_version_service_windows(feed_start_date);
CREATE INDEX ON feed_version_service_windows(feed_end_date);
CREATE INDEX ON feed_version_service_windows(earliest_calendar_date);
CREATE INDEX ON feed_version_service_windows(latest_calendar_date);

COMMIT;