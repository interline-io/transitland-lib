CREATE OR REPLACE FUNCTION unimport_feed_version(fvid bigint) RETURNS integer AS $$
DECLARE
    fvid ALIAS for $1;
BEGIN

RAISE NOTICE 'unimport_feed_version: %', fvid;

RAISE NOTICE '... deleting tl tables';
DELETE FROM "active_agencies" WHERE feed_version_id = fvid;
DELETE FROM "active_routes" WHERE feed_version_id = fvid;
DELETE FROM "active_stops" WHERE feed_version_id = fvid;
DELETE FROM "agency_geometries" WHERE feed_version_id = fvid;
DELETE FROM "route_geometries" WHERE feed_version_id = fvid;
DELETE FROM "route_stops" WHERE feed_version_id = fvid;
DELETE FROM "feed_version_geometries" WHERE feed_version_id = fvid;

RAISE NOTICE '... deleting anonymous gtfs entities';
DELETE FROM "gtfs_stop_times" WHERE feed_version_id = fvid;
DELETE FROM "gtfs_transfers" WHERE feed_version_id = fvid;
DELETE FROM "gtfs_calendar_dates" WHERE feed_version_id = fvid;
DELETE FROM "gtfs_feed_infos" WHERE feed_version_id = fvid;
DELETE FROM "gtfs_frequencies" WHERE feed_version_id = fvid;
DELETE FROM "gtfs_pathways" WHERE feed_version_id = fvid;
DELETE FROM "gtfs_fare_rules" WHERE feed_version_id = fvid;

RAISE NOTICE '... deleting named gtfs entities';
DELETE FROM "gtfs_fare_attributes" WHERE feed_version_id = fvid;
DELETE FROM "gtfs_trips" WHERE feed_version_id = fvid;
DELETE FROM "gtfs_shapes" WHERE feed_version_id = fvid;
DELETE FROM "gtfs_calendars" WHERE feed_version_id = fvid;
DELETE FROM "gtfs_routes" WHERE feed_version_id = fvid;
DELETE FROM "gtfs_stops" WHERE feed_version_id = fvid;
DELETE FROM "gtfs_agencies" WHERE feed_version_id = fvid;
DELETE FROM "gtfs_levels" WHERE feed_version_id = fvid;

RAISE NOTICE '... deleting import records';
DELETE FROM "feed_version_gtfs_imports" WHERE feed_version_id = fvid;

RAISE NOTICE '... unsetting feed_state feed_version_id';
UPDATE "feed_states" SET feed_version_id = null WHERE feed_version_id = fvid;

RETURN 0;
END;
$$ LANGUAGE plpgsql;