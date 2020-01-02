CREATE OR REPLACE FUNCTION after_feed_version_import(fvid bigint) RETURNS integer AS $$
DECLARE
    fid integer;
    fvid ALIAS for $1;
    foundgeom integer;
BEGIN

SELECT feed_id INTO STRICT fid FROM feed_versions WHERE feed_versions.id = fvid;
RAISE NOTICE 'after_feed_version_import fid: % fvid: %', fid, fvid;

RAISE NOTICE '... feed_version_geometries delete';
DELETE FROM feed_version_geometries WHERE feed_version_id = fvid;
RAISE NOTICE '... route_stops delete';
DELETE FROM route_stops WHERE feed_version_id = fvid;
RAISE NOTICE '... agency_geometries delete';
DELETE FROM agency_geometries WHERE feed_version_id = fvid;
RAISE NOTICE '... route_geometries delete';
DELETE FROM route_geometries WHERE feed_version_id = fvid;

-- feed_version_geometries
RAISE NOTICE '... feed_version_geometries insert';
INSERT INTO feed_version_geometries(feed_version_id,geometry,centroid)
SELECT
    gtfs_stops.feed_version_id,
    ST_Buffer(ST_ConvexHull(ST_Collect(gtfs_stops.geometry::geometry)), 0.01),
    ST_GeometricMedian(ST_Collect(gtfs_stops.geometry::geometry))
FROM gtfs_stops 
WHERE gtfs_stops.feed_version_id = fvid
GROUP BY gtfs_stops.feed_version_id;

-- route stops
RAISE NOTICE '... route_stops insert';
INSERT INTO route_stops(feed_version_id, agency_id, stop_id, route_id)
WITH a AS (
SELECT
    gtfs_stop_times.stop_id,
    gtfs_trips.route_id
from
    gtfs_stop_times
inner join gtfs_trips on gtfs_stop_times.trip_id = gtfs_trips.id
where gtfs_stop_times.feed_version_id = fvid
group by gtfs_stop_times.stop_id, gtfs_trips.route_id)
SELECT
    fvid as feed_version_id,
    gtfs_routes.agency_id,
    a.stop_id,
    a.route_id
FROM a
INNER JOIN gtfs_routes ON gtfs_routes.id = a.route_id;

-- agency_geometries
RAISE NOTICE '... agency_geometries insert';
INSERT INTO agency_geometries(feed_version_id,agency_id,geometry,centroid)
SELECT
    route_stops.feed_version_id,
    route_stops.agency_id,
    ST_Buffer(ST_ConvexHull(ST_Collect(gtfs_stops.geometry::geometry)), 0.01),
    ST_GeometricMedian(ST_Collect(gtfs_stops.geometry::geometry))
FROM route_stops 
INNER JOIN gtfs_stops ON gtfs_stops.id = route_stops.stop_id
WHERE route_stops.feed_version_id = fvid
GROUP BY (route_stops.feed_version_id, route_stops.agency_id);

-- route geometries
RAISE NOTICE '... route_geometries insert';
INSERT INTO route_geometries(
    feed_version_id,
    route_id,
    direction_id,
    shape_id,
    generated,
    geometry,
    geometry_z14,
    geometry_z10,
    geometry_z6,
    centroid
    )
WITH
best_shapes AS (
    SELECT
        gtfs_routes.id as route_id,
        gtfs_trips.direction_id,
        gtfs_trips.shape_id,
        count(*) as count
    FROM gtfs_routes 
    INNER JOIN gtfs_trips on gtfs_trips.route_id = gtfs_routes.id 
    WHERE gtfs_routes.feed_version_id = fvid
    GROUP BY (gtfs_routes.id, gtfs_trips.shape_id, gtfs_trips.direction_id) 
)
SELECT 
    DISTINCT ON (best_shapes.route_id, best_shapes.direction_id)
    gtfs_shapes.feed_version_id as feed_version_id,
    best_shapes.route_id,
    best_shapes.direction_id,
    best_shapes.shape_id,
    gtfs_shapes.generated,
    ST_Simplify(ST_Force2D(gtfs_shapes.geometry::geometry), 0.00001, true),
    ST_Simplify(ST_Force2D(gtfs_shapes.geometry::geometry), 0.0001, true),
    ST_Simplify(ST_Force2D(gtfs_shapes.geometry::geometry), 0.001, true),
    ST_Simplify(ST_Force2D(gtfs_shapes.geometry::geometry), 0.01, false),
    ST_Centroid(ST_Force2D(gtfs_shapes.geometry::geometry))
FROM best_shapes 
INNER JOIN gtfs_shapes ON gtfs_shapes.id = best_shapes.shape_id
ORDER BY best_shapes.route_id, best_shapes.direction_id, count DESC;

RETURN 0;
END;
$$ LANGUAGE plpgsql;