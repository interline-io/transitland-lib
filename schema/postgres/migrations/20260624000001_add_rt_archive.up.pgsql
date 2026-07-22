BEGIN;

-- Per-feed RT archive retention in days (0 disables) and the archived object key.
ALTER TABLE public.feed_states ADD COLUMN rt_retention_period integer NOT NULL DEFAULT 0;
ALTER TABLE public.feed_fetches ADD COLUMN storage_key text;

COMMIT;
