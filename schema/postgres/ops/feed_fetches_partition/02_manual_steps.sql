-- Manual steps around the changeover migration (20260624000003_feed_fetches_changeover).
-- The swap itself is in that migration; these are the parts that don't belong in
-- migration history (env-specific checks and permissions).

-- BEFORE migrating ------------------------------------------------------------
-- Views/matviews/functions bind to the table OID, not the name, so anything depending
-- on feed_fetches would silently keep reading the old table after the rename. Expect
-- zero rows; recreate any dependents found against the new table.
--   SELECT DISTINCT dep.relname, dep.relkind
--   FROM pg_depend d
--   JOIN pg_rewrite r ON r.oid = d.objid
--   JOIN pg_class  dep ON dep.oid = r.ev_class
--   WHERE d.refobjid = 'feed_fetches'::regclass AND dep.relname <> 'feed_fetches';

-- Capture existing grants so they can be replayed below:
--   SELECT grantee, privilege_type
--   FROM information_schema.role_table_grants WHERE table_name = 'feed_fetches';

-- Run order (queue paused throughout):
--   1. pause the fetch queue
--   2. dbmigrate  -> applies the structure + changeover migrations (swap)
--   3. 01_seed_partitions.sql  -> seed current + forward months (DEFAULT leaves empty -> no race)
--   4. permission replay (below)
--   5. 03_backfill.sql  -> backfill static + storage_key rows, verify, drop old
--   6. resume the fetch queue

-- AFTER migrating (production) -------------------------------------------------
-- Replicate owner + grants the rename did not carry. Adjust to the roles captured above;
-- example only:
--   ALTER TABLE feed_fetches OWNER TO onestop;
--   GRANT SELECT, INSERT, UPDATE, DELETE ON feed_fetches TO onestop;
--   GRANT SELECT ON feed_fetches TO readonly;
