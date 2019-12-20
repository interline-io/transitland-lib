CREATE OR REPLACE FUNCTION activate_feed_version(fvid bigint) RETURNS integer AS $$
DECLARE
    fid integer;
    fvid ALIAS for $1;
    foundgeom integer;
BEGIN

SELECT feed_id INTO STRICT fid FROM feed_versions WHERE feed_versions.id = fvid;
RAISE NOTICE 'activate_feed_version fid: % fvid: %', fid, fvid;

RAISE NOTICE '... setting feed_states feed_version_id';
UPDATE feed_states SET feed_version_id = fvid WHERE feed_states.feed_id = fid;

RAISE NOTICE '... active_agencies delete';
DELETE FROM active_agencies USING feed_versions WHERE active_agencies.feed_version_id = feed_versions.id AND feed_versions.feed_id = fid;

RAISE NOTICE '... active_stops delete';
DELETE FROM active_stops USING feed_versions WHERE active_stops.feed_version_id = feed_versions.id AND feed_versions.feed_id = fid;

RAISE NOTICE '... active_routes delete';
DELETE FROM active_routes USING feed_versions WHERE active_routes.feed_version_id = feed_versions.id AND feed_versions.feed_id = fid;

---------------

RAISE NOTICE '... active_agencies insert';
INSERT INTO active_agencies(
    id              ,
    feed_version_id ,
    agency_id       ,
    agency_name     ,
    agency_url      ,
    agency_timezone ,
    agency_lang     ,
    agency_phone    ,
    agency_fare_url ,
    agency_email    ,
    created_at      ,
    updated_at      ,
    geometry        ,
    centroid
)
SELECT 
    gtfs_agencies.id              ,
    gtfs_agencies.feed_version_id ,
    gtfs_agencies.agency_id       ,
    gtfs_agencies.agency_name     ,
    gtfs_agencies.agency_url      ,
    gtfs_agencies.agency_timezone ,
    gtfs_agencies.agency_lang     ,
    gtfs_agencies.agency_phone    ,
    gtfs_agencies.agency_fare_url ,
    gtfs_agencies.agency_email    ,
    gtfs_agencies.created_at      ,
    gtfs_agencies.updated_at      ,
    agency_geometries.geometry, 
    agency_geometries.centroid
FROM gtfs_agencies 
LEFT OUTER JOIN agency_geometries ON agency_geometries.agency_id = gtfs_agencies.id
WHERE gtfs_agencies.feed_version_id = fvid;

---------------

RAISE NOTICE '... active_stops insert';
INSERT INTO active_stops(
    id                  ,
    feed_version_id     ,
    parent_station      ,
    stop_id             ,
    stop_code           ,
    stop_name           ,
    stop_desc           ,
    zone_id             ,
    stop_url            ,
    location_type       ,
    stop_timezone       ,
    wheelchair_boarding ,
    geometry            ,
    created_at          ,
    updated_at          ,
    level_id            
) SELECT  
    gtfs_stops.id                  ,
    gtfs_stops.feed_version_id     ,
    gtfs_stops.parent_station      ,
    gtfs_stops.stop_id             ,
    gtfs_stops.stop_code           ,
    gtfs_stops.stop_name           ,
    gtfs_stops.stop_desc           ,
    gtfs_stops.zone_id             ,
    gtfs_stops.stop_url            ,
    gtfs_stops.location_type       ,
    gtfs_stops.stop_timezone       ,
    gtfs_stops.wheelchair_boarding ,
    gtfs_stops.geometry            ,
    gtfs_stops.created_at          ,
    gtfs_stops.updated_at          ,
    gtfs_stops.level_id            
FROM
    gtfs_stops
WHERE feed_version_id = fvid;

---------------

RAISE NOTICE '... active_routes insert';
INSERT INTO active_routes (
    id               ,
    feed_version_id  ,
    agency_id        ,
    route_id         ,
    route_short_name ,
    route_long_name  ,
    route_desc       ,
    route_type       ,
    route_url        ,
    route_color      ,
    route_text_color ,
    route_sort_order ,
    created_at       ,
    updated_at       ,
    geometry         ,
    geometry_z14     ,
    geometry_z10     ,
    geometry_z6      ,
    centroid         
)
SELECT
    gtfs_routes.id               ,
    gtfs_routes.feed_version_id  ,
    gtfs_routes.agency_id        ,
    gtfs_routes.route_id         ,
    gtfs_routes.route_short_name ,
    gtfs_routes.route_long_name  ,
    gtfs_routes.route_desc       ,
    gtfs_routes.route_type       ,
    gtfs_routes.route_url        ,
    gtfs_routes.route_color      ,
    gtfs_routes.route_text_color ,
    gtfs_routes.route_sort_order ,
    gtfs_routes.created_at       ,
    gtfs_routes.updated_at       ,
    route_geometries.geometry         ,
    route_geometries.geometry_z14     ,
    route_geometries.geometry_z10     ,
    route_geometries.geometry_z6      ,
    route_geometries.centroid         
FROM
    gtfs_routes
LEFT OUTER JOIN route_geometries ON route_geometries.route_id = gtfs_routes.id
WHERE gtfs_routes.feed_version_id = fvid AND route_geometries.direction_id = 0;

RETURN 0;
END;
$$ LANGUAGE plpgsql;