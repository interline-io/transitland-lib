DROP VIEW IF EXISTS vw_active_agencies;
CREATE VIEW vw_active_agencies AS
SELECT
    gtfs_agencies.*,
    agency_geometries.geometry,
    agency_geometries.centroid
FROM
    gtfs_agencies
INNER JOIN feed_states USING(feed_version_id)
LEFT OUTER JOIN agency_geometries ON agency_geometries.agency_id = gtfs_agencies.id;

DROP VIEW IF EXISTS vw_active_routes;
CREATE VIEW vw_active_routes AS
SELECT
    gtfs_routes.*,
    route_geometries.geometry,
    route_geometries.centroid,
    route_geometries.generated
FROM gtfs_routes
INNER JOIN feed_states USING(feed_version_id) 
LEFT OUTER JOIN route_geometries on route_geometries.route_id = gtfs_routes.id;

DROP VIEW IF EXISTS vw_active_stops;
CREATE VIEW vw_active_stops AS
SELECT gtfs_stops.* FROM gtfs_stops INNER JOIN feed_states USING(feed_version_id);

DROP VIEW IF EXISTS tile_active_stops;
CREATE VIEW tile_active_stops AS
SELECT 
    gtfs_stops.id,
    gtfs_stops.stop_id,
    gtfs_stops.stop_name,
    gtfs_stops.geometry
FROM gtfs_stops 
INNER JOIN feed_states USING(feed_version_id);

DROP VIEW IF EXISTS tile_active_routes;
CREATE VIEW tile_active_routes AS
SELECT
    gtfs_routes.id,
    gtfs_routes.agency_id,
    gtfs_routes.route_id,
    gtfs_routes.route_short_name,
    gtfs_routes.route_long_name,
    (CASE
        WHEN route_color = '' THEN NULL
        WHEN SUBSTRING(route_color,1,1) != '#' THEN '#'||lower(route_color)
        ELSE lower(route_color)
        END
    ) AS route_color,
    (CASE
        WHEN route_type <= 7 THEN route_type
        WHEN route_type BETWEEN 100 AND 199 THEN 2
        WHEN route_type BETWEEN 200 AND 299 THEN 3
        WHEN route_type BETWEEN 300 AND 399 THEN 2
        WHEN route_type BETWEEN 300 AND 399 THEN 2
        WHEN route_type BETWEEN 400 AND 699 THEN 1
        WHEN route_type BETWEEN 700 AND 899 THEN 3
        WHEN route_type BETWEEN 800 AND 999 THEN 0
        WHEN route_type BETWEEN 1000 AND 1099 THEN 4
        WHEN route_type BETWEEN 1100 AND 1199 THEN 3 -- NOT a bus but not used
        WHEN route_type BETWEEN 1200 AND 1299 THEN 4
        WHEN route_type BETWEEN 1300 AND 1399 THEN 6
        WHEN route_type BETWEEN 1400 AND 1499 THEN 7
        WHEN route_type >= 1500 THEN 3 -- misc -> bus
        ELSE 3 -- misc -> bus
        END
    ) AS route_type,
    route_geometries.geometry,
    route_geometries.centroid,
    route_geometries.generated,
    (CASE
        WHEN headway_secs = 0 OR headway_secs IS NULL THEN 1000000 -- make life easier
        ELSE headway_secs
        END
    ) AS headway_secs,
    gtfs_agencies.agency_name,
    current_feeds.onestop_id
FROM gtfs_routes
INNER JOIN feed_states USING(feed_version_id) 
INNER JOIN current_feeds ON current_feeds.id = feed_states.feed_id
INNER JOIN gtfs_agencies ON gtfs_agencies.id = gtfs_routes.agency_id
LEFT OUTER JOIN route_geometries on route_geometries.route_id = gtfs_routes.id
LEFT OUTER JOIN route_headways ON route_headways.route_id = gtfs_routes.id;
