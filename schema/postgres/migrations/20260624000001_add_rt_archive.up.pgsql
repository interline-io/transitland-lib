BEGIN;

-- RT message archive: per-feed retention period in days (0 disables archiving;
-- matches feed_version_import_retention_period) and the object key where each
-- fetched message was archived. Deletion is by date-prefix, so retention is
-- day-granular.
ALTER TABLE public.feed_states ADD COLUMN rt_retention_period integer NOT NULL DEFAULT 0;
ALTER TABLE public.feed_fetches ADD COLUMN storage_key text;

COMMIT;
