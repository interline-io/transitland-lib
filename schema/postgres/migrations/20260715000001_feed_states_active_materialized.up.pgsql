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

-- Initialize both pointers to the current visible version: materialized_ matches what is actually
-- materialized today, and active_ protects that same version from unimport. Feeds excluded from
-- global queries have a null feed_version_id, so their active_ starts null and is filled by the
-- first set-active reconcile after deploy (safe: the unimport guard skips a feed whose active_ is
-- null until then). Kept to feed_states only so the migration does not scan feed_versions.
UPDATE feed_states
SET materialized_feed_version_id = feed_version_id,
    active_feed_version_id = feed_version_id;

-- materialized_feed_version_id becomes the global-query / materialized-table join key when
-- its readers migrate off feed_version_id; give it the same unique index now.
CREATE UNIQUE INDEX index_feed_states_on_materialized_feed_version_id
    ON public.feed_states (materialized_feed_version_id);

COMMIT;
