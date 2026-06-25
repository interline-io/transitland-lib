-- Create the monthly partitions for the realtime/gbfs branches.
--
-- This is intentionally NOT in the migration: a fresh/test database relies on the
-- RANGE DEFAULT leaves and needs no months, so migrations stay fast. Production runs
-- this once right after the changeover migration, while the queue is still paused, so
-- the DEFAULT leaves are empty and the current + forward months create cleanly; then
-- the queue resumes and live rows land in real month partitions. The top-up worker calls
-- feed_fetches_add_month() monthly to stay ahead. The function lives here, not in the
-- migration: it is a persistent object installed by running this script, untracked by
-- migrations so it can change freely. (The worker presumes it is installed, or can
-- CREATE OR REPLACE it itself.)

-- Create one monthly partition under a month-partitioned subtree. Idempotent.
-- Bounds are [first-of-month, first-of-next-month): lower inclusive, upper exclusive.
CREATE OR REPLACE FUNCTION feed_fetches_add_month(p_parent text, p_month date)
RETURNS text LANGUAGE plpgsql AS $$
DECLARE
    v_child text;
    v_start date := date_trunc('month', p_month)::date;
    v_end   date := (date_trunc('month', p_month) + interval '1 month')::date;
BEGIN
    IF p_parent NOT IN ('feed_fetches_rt', 'feed_fetches_gbfs') THEN
        RAISE EXCEPTION 'feed_fetches_add_month: % is not a month-partitioned subtree', p_parent;
    END IF;
    v_child := format('%s_%s', p_parent, to_char(v_start, 'YYYY_MM'));
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L)',
        v_child, p_parent, v_start, v_end);
    RETURN v_child;
END;
$$;

-- One month, one subtree (idempotent):
--   SELECT feed_fetches_add_month('feed_fetches_rt', date '2026-07-01');

-- Seed a buffer: each month-partitioned subtree, from 3 months back through 3 years ahead.
-- Backwards months only matter if you backfill recent realtime/gbfs (e.g. storage_key
-- rows); pure-static backfill needs none. Adjust the range as desired.
DO $$
DECLARE
    v_parent text;
    v_month  date;
BEGIN
    FOREACH v_parent IN ARRAY ARRAY['feed_fetches_rt', 'feed_fetches_gbfs'] LOOP
        v_month := date_trunc('month', now()) - interval '3 months';
        WHILE v_month < date_trunc('month', now()) + interval '3 years' LOOP
            PERFORM feed_fetches_add_month(v_parent, v_month::date);
            v_month := v_month + interval '1 month';
        END LOOP;
    END LOOP;
END $$;

-- Monthly top-up (what the worker runs): create the next month a bit ahead of time.
--   SELECT feed_fetches_add_month(p, (date_trunc('month', now()) + interval '2 months')::date)
--   FROM unnest(ARRAY['feed_fetches_rt','feed_fetches_gbfs']) AS p;
