BEGIN;

drop table tl_ext_gtfs_stops;
create index on tl_route_headways(selected_stop_id);
create index on tl_route_headways(service_id);

COMMIT;
