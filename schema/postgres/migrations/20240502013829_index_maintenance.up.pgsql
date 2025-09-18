BEGIN;

drop index gtfs_stops_feed_version_id_geometry_idx;

create index on tl_agency_onestop_ids(onestop_id) include (agency_id,feed_version_id);
create index on tl_route_onestop_ids(onestop_id) include (route_id,feed_version_id);
create index on tl_stop_onestop_ids(onestop_id) include (stop_id,feed_version_id);

drop index index_route_stops_on_route_id;
create unique index on tl_route_stops(route_id,stop_id);

create index on ne_10m_admin_1_states_provinces(admin) include (name, iso_a2, iso_3166_2);
create index on ne_10m_admin_1_states_provinces(name) include (admin, iso_a2, iso_3166_2);

COMMIT;