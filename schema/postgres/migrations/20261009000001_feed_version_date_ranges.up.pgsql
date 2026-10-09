-- The feed version that answers for each feed on each date, for queries given a date
-- instead of a feed's active version. A null bound is open-ended, and a feed's ranges
-- never overlap.
BEGIN;

CREATE TABLE feed_version_date_ranges (
    id bigserial PRIMARY KEY,
    feed_id bigint NOT NULL REFERENCES current_feeds(id),
    feed_version_id bigint NOT NULL REFERENCES feed_versions(id),
    start_date date,
    end_date date,
    CHECK (start_date <= end_date),
    CONSTRAINT feed_version_date_ranges_no_overlap EXCLUDE USING gist (feed_id WITH =, daterange(start_date, end_date, '[]') WITH &&)
);

CREATE INDEX ON feed_version_date_ranges(feed_version_id);

COMMIT;
