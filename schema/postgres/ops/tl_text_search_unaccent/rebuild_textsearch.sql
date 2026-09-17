-- Rebuild stored textsearch vectors after migration 20260917000001_tl_text_search_unaccent.
--
-- Not in the migration: it rewrites every agency, route and stop row, which a fresh database
-- doesn't need. Run once after migrating, outside a transaction block so each feed version
-- commits on its own; safe to re-run. Until it finishes, a search with a non-ASCII word misses
-- rows written before the migration.

-- Registry tables, one statement each. Assigning a column to itself recomputes the
-- generated textsearch column that depends on it.
UPDATE current_feeds SET name = name;
UPDATE current_operators SET name = name;
UPDATE current_operators_in_feed SET resolved_name = resolved_name;

-- Imported entities, one feed version per transaction. The materialized tables hold copies
-- of the active rows' vectors, so they are refreshed from the rebuilt rows.
DO $$
DECLARE
    v_fvid bigint;
BEGIN
    FOR v_fvid IN SELECT id FROM feed_versions ORDER BY id LOOP
        UPDATE gtfs_agencies SET agency_name = agency_name WHERE feed_version_id = v_fvid;
        UPDATE gtfs_routes SET route_long_name = route_long_name WHERE feed_version_id = v_fvid;
        UPDATE gtfs_stops SET stop_name = stop_name WHERE feed_version_id = v_fvid;
        UPDATE tl_materialized_active_agencies m SET textsearch = a.textsearch
            FROM gtfs_agencies a WHERE a.id = m.id AND m.feed_version_id = v_fvid;
        UPDATE tl_materialized_active_routes m SET textsearch = r.textsearch
            FROM gtfs_routes r WHERE r.id = m.id AND m.feed_version_id = v_fvid;
        UPDATE tl_materialized_active_stops m SET textsearch = s.textsearch
            FROM gtfs_stops s WHERE s.id = m.id AND m.feed_version_id = v_fvid;
        COMMIT;
        RAISE NOTICE 'rebuild_textsearch: feed version % done', v_fvid;
    END LOOP;
END
$$;
