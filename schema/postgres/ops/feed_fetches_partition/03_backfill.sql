-- Backfill the keep-set from the retained old table into the new partitioned table,
-- then drop the old table.
--
-- Keep-set: all static history, plus any row that points at an archived blob
-- (storage_key IS NOT NULL) so the cull can never orphan the RT archive. Everything
-- else in old (un-archived realtime/gbfs) is intentionally dropped with the old table.
--
-- Runs AFTER the changeover: it reads the old table's final committed state, so any
-- stragglers that landed in old just after the swap are captured too. Batched with a
-- per-batch COMMIT so it is online (bounded locks/WAL) and resumable; ON CONFLICT makes
-- re-runs idempotent. The shared sequence guarantees old ids never collide with live ids.
--
-- fetched_at is COALESCEd to created_at because legacy rows may have NULL fetched_at and
-- the new column is NOT NULL. Old static rows route to feed_fetches_static; old
-- storage_key rows route to their month partition if seeded, else the type's DEFAULT leaf.

CREATE OR REPLACE PROCEDURE feed_fetches_backfill(p_batch bigint DEFAULT 100000)
LANGUAGE plpgsql AS $$
DECLARE
    v_id  bigint := 0;
    v_max bigint;
BEGIN
    SELECT max(id) INTO v_max FROM feed_fetches_old;
    WHILE v_id <= v_max LOOP
        INSERT INTO feed_fetches (
            id, feed_id, url_type, url, success, fetched_at, fetch_error,
            response_size, response_code, response_sha1, feed_version_id,
            created_at, updated_at, response_ttfb_ms, response_time_ms,
            validation_duration_ms, upload_duration_ms, storage_key)
        SELECT
            id, feed_id, url_type, url, success,
            COALESCE(fetched_at, created_at), fetch_error,
            response_size, response_code, response_sha1, feed_version_id,
            created_at, updated_at, response_ttfb_ms, response_time_ms,
            validation_duration_ms, upload_duration_ms, storage_key
        FROM feed_fetches_old
        WHERE id > v_id AND id <= v_id + p_batch
          AND (url_type LIKE 'static_%' OR storage_key IS NOT NULL)
        ON CONFLICT DO NOTHING;

        v_id := v_id + p_batch;
        COMMIT;
    END LOOP;
END;
$$;

-- Run in autocommit (e.g. plain psql), NOT inside a transaction block: the per-batch
-- COMMIT above raises "invalid transaction termination" if CALL runs inside one. Avoid
-- BEGIN/COMMIT wrappers, psql --single-transaction, and clients that auto-wrap statements.
--   CALL feed_fetches_backfill();

-- Verify before dropping (counts should match the keep-set):
--   SELECT count(*) FROM feed_fetches_old WHERE url_type LIKE 'static_%' OR storage_key IS NOT NULL;
--   SELECT count(*) FROM feed_fetches      WHERE url_type LIKE 'static_%' OR storage_key IS NOT NULL;

-- Reclaim the ~500 GB. No inbound FKs reference feed_fetches, and the sequence ownership
-- was already moved to the new table in the changeover, so this is clean.
--   DROP TABLE feed_fetches_old;
