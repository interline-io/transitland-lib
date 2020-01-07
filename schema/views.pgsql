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
