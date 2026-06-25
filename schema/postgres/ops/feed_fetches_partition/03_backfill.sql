-- Backfill the keep-set from the retained old table into the new partitioned table,
-- then drop the old table.
--
-- Keep-set: all static history; any row that points at an archived blob
-- (storage_key IS NOT NULL) so the cull can never orphan the RT archive; and the most
-- recent p_rt_keep of realtime_* fetches (default 3 months). Everything else in old --
-- older realtime and all un-archived gbfs -- is intentionally dropped with the old table.
--
-- The RT keep window is month-aligned (date_trunc('month', now()) - p_rt_keep) so it lines
-- up exactly with the seeded RT partitions (the seed went 3 months back); every kept RT row
-- therefore lands in a real month partition, never the RT default. The cutoff is captured
-- once up front so it can't drift forward across the multi-hour, per-batch-committed run.
--
-- Runs AFTER the changeover: it reads the old table's final committed state, so any
-- stragglers that landed in old just after the swap are captured too. Batched with a
-- per-batch COMMIT so it is online (bounded locks/WAL) and resumable; ON CONFLICT makes
-- re-runs idempotent. The shared sequence guarantees old ids never collide with live ids.
--
-- fetched_at is COALESCEd to created_at because legacy rows may have NULL fetched_at and
-- the new column is NOT NULL. Old static rows route to feed_fetches_static; recent realtime
-- and storage_key rows route to their month partition if seeded, else the type's DEFAULT leaf.

CREATE OR REPLACE PROCEDURE feed_fetches_backfill(p_batch bigint DEFAULT 100000, p_rt_keep interval DEFAULT interval '3 months')
LANGUAGE plpgsql AS $$
DECLARE
    v_id        bigint := 0;
    v_max       bigint;
    v_ins       bigint;
    v_total     bigint := 0;
    v_batch     bigint := 0;
    v_batches   bigint;
    v_rt_cutoff timestamptz := date_trunc('month', now()) - p_rt_keep;
BEGIN
    SELECT max(id) INTO v_max FROM feed_fetches_old;
    v_batches := v_max / p_batch + 1;
    RAISE NOTICE 'feed_fetches_backfill: start, max id %, % batches of %, keeping realtime_* since %', v_max, v_batches, p_batch, v_rt_cutoff;
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
          AND (url_type LIKE 'static_%'
               OR storage_key IS NOT NULL
               OR (url_type LIKE 'realtime_%' AND COALESCE(fetched_at, created_at) >= v_rt_cutoff))
        ON CONFLICT DO NOTHING;

        GET DIAGNOSTICS v_ins = ROW_COUNT;
        v_total := v_total + v_ins;
        v_id := v_id + p_batch;
        v_batch := v_batch + 1;
        COMMIT;

        RAISE NOTICE 'feed_fetches_backfill: batch %/% (% done), +% this batch, % copied so far',
            v_batch, v_batches,
            round(100.0 * v_batch / NULLIF(v_batches, 0), 1)::text || '%',
            v_ins, v_total;
    END LOOP;
    RAISE NOTICE 'feed_fetches_backfill: done, % rows copied', v_total;
END;
$$;

-- Run in autocommit (e.g. plain psql), NOT inside a transaction block: the per-batch
-- COMMIT above raises "invalid transaction termination" if CALL runs inside one. Avoid
-- BEGIN/COMMIT wrappers, psql --single-transaction, and clients that auto-wrap statements.
--   CALL feed_fetches_backfill();                          -- default: 3 months of realtime_*
--   CALL feed_fetches_backfill(100000, interval '6 months');  -- keep more recent realtime

-- Verify before dropping. Counts should match; run in the same calendar month as the
-- backfill so the month-aligned RT cutoff is identical (adjust the interval if you passed
-- a non-default p_rt_keep):
--   SELECT count(*) FROM feed_fetches_old
--     WHERE url_type LIKE 'static_%' OR storage_key IS NOT NULL
--        OR (url_type LIKE 'realtime_%' AND COALESCE(fetched_at, created_at) >= date_trunc('month', now()) - interval '3 months');
--   SELECT count(*) FROM feed_fetches
--     WHERE url_type LIKE 'static_%' OR storage_key IS NOT NULL
--        OR (url_type LIKE 'realtime_%' AND fetched_at >= date_trunc('month', now()) - interval '3 months');

-- Reclaim the ~500 GB. No inbound FKs reference feed_fetches, and the sequence ownership
-- was already moved to the new table in the changeover, so this is clean.
--   DROP TABLE feed_fetches_old;
