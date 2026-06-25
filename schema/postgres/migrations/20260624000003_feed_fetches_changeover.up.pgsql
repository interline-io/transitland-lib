BEGIN;

-- Changeover: swap the partitioned feed_fetches_new in for feed_fetches.
--
-- Run manually (dbmigrate) with the fetch queue paused. The renames take a brief
-- ACCESS EXCLUSIVE lock; with writers paused there is no contention, and nothing has
-- landed in the new RANGE DEFAULT leaves yet — so the forward partitions can be seeded
-- cleanly immediately afterward (ops/feed_fetches_partition/01_seed_partitions.sql).
-- Do not resume the queue before seeding: once current-month realtime rows are in a
-- DEFAULT leaf, that month's partition can no longer be created.
--
-- Owner/grant replay is a manual ops step (rename carries neither).

ALTER TABLE feed_fetches     RENAME TO feed_fetches_old;
ALTER TABLE feed_fetches_new RENAME TO feed_fetches;

-- Same global sequence (its value is already past every existing id, so backfilled old
-- ids never collide with new live ids). Reassign ownership off the old table so dropping
-- it never drops the sequence.
ALTER SEQUENCE feed_fetches_id_seq OWNED BY feed_fetches.id;

-- Drop the old table only when it is empty. On a fresh/test database that is the empty
-- init table, so this leaves a partitioned feed_fetches everywhere. On production the
-- old table has rows, EXISTS short-circuits to true, and it is kept for the backfill
-- (ops/feed_fetches_partition/03_backfill.sql) and dropped manually afterward.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM feed_fetches_old) THEN
        DROP TABLE feed_fetches_old;
    END IF;
END $$;

COMMIT;
