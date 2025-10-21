BEGIN;

CREATE INDEX ON tl_materialized_active_stops(stop_name);
CREATE INDEX ON tl_materialized_active_stops(location_type);
CREATE INDEX ON tl_materialized_active_stops(stop_code);

CREATE INDEX ON tl_materialized_active_routes(route_short_name);
CREATE INDEX ON tl_materialized_active_routes(route_long_name);
CREATE INDEX ON tl_materialized_active_routes(route_type);

CREATE INDEX ON tl_materialized_active_agencies(agency_name);

COMMIT;
