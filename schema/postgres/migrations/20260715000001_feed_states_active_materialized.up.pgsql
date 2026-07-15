BEGIN;

-- Split the active feed version from the materialized/visible one.
-- active_feed_version_id is the current version for every importable feed (including
-- feeds excluded from global queries) and drives retention. materialized_feed_version_id
-- is the visible/materialized version, NULL when the feed is excluded. feed_version_id is
-- kept as a transitional mirror of materialized_feed_version_id until its readers migrate.
-- exclude_from_global is the visibility decision, seeded once from the DMFR tag and owned
-- here thereafter.
ALTER TABLE public.feed_states
    ADD COLUMN active_feed_version_id bigint REFERENCES feed_versions(id),
    ADD COLUMN materialized_feed_version_id bigint REFERENCES feed_versions(id),
    ADD COLUMN exclude_from_global boolean NOT NULL DEFAULT false;

-- Seed exclude_from_global from the DMFR feed tag.
UPDATE feed_states SET exclude_from_global = true
FROM current_feeds
WHERE current_feeds.id = feed_states.feed_id
  AND current_feeds.feed_tags->>'exclude_from_global_query' = 'true';

-- Mirror the current visible pointer; this matches what is actually materialized today.
UPDATE feed_states SET materialized_feed_version_id = feed_version_id;

-- Backfill active_feed_version_id = the newest started importable version per feed, using
-- the active-selection logic without the global-exclude filter (status/deleted still gate),
-- so excluded feeds get a retained active version immediately and unimport protects it.
WITH importable_fvs AS (
    SELECT fv.id AS feed_version_id,
        fv.feed_id,
        (now() at time zone fvsw.default_timezone)::date AS local_time,
        greatest(
            fvsw.feed_start_date,
            fvsw.earliest_calendar_date,
            fv.earliest_calendar_date
        ) AS greatest_start_date,
        greatest(ff.fetched_at, fv.fetched_at) AS greatest_fetched_at
    FROM feed_versions fv
        JOIN feed_version_gtfs_imports fvgi ON fvgi.feed_version_id = fv.id
        JOIN feed_version_service_windows fvsw ON fvsw.feed_version_id = fv.id
        LEFT JOIN LATERAL (
            SELECT * FROM feed_fetches WHERE feed_fetches.feed_version_id = fv.id LIMIT 1
        ) ff ON TRUE
    WHERE fvsw.default_timezone != ''
        AND fvgi.success = TRUE
        AND fvgi.schedule_removed = FALSE
),
selected_fvs AS (
    SELECT DISTINCT ON (cf.id) cf.id AS feed_id,
        importable_fvs.feed_version_id
    FROM current_feeds cf
        JOIN importable_fvs ON importable_fvs.feed_id = cf.id
    WHERE importable_fvs.greatest_start_date <= importable_fvs.local_time
        AND cf.deleted_at IS NULL
        AND (
            cf.feed_tags->'status' IS NULL
            OR cf.feed_tags->>'status' NOT IN ('archived', 'unpublished')
        )
    ORDER BY cf.id ASC,
        importable_fvs.greatest_fetched_at DESC
)
UPDATE feed_states
SET active_feed_version_id = selected_fvs.feed_version_id
FROM selected_fvs
WHERE feed_states.feed_id = selected_fvs.feed_id;

-- materialized_feed_version_id becomes the global-query / materialized-table join key when
-- its readers migrate off feed_version_id; give it the same unique index now.
CREATE UNIQUE INDEX index_feed_states_on_materialized_feed_version_id
    ON public.feed_states (materialized_feed_version_id);

COMMIT;
