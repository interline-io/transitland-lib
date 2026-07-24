BEGIN;

-- Per-feed retention for onestop_id stats (feed_version_agency/route/stop_onestop_ids),
-- which exist only to power AllowPrevious lookups. 0 keeps them forever, N>0 culls versions
-- older than N days. Mirrors the rt_retention_period convention.
ALTER TABLE public.feed_states
    ADD COLUMN onestop_id_retention_period integer NOT NULL DEFAULT 0;

COMMIT;
