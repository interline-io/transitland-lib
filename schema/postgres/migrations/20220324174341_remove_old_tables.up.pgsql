BEGIN;

drop table tl_ext_gtfs_stops;
create index on tl_route_headways(selected_stop_id);
create index on tl_route_headways(service_id);
create index on ext_plus_stop_attributes(stop_id);

alter table gtfs_attributions drop constraint gtfs_attributions_agency_id_fkey;
alter table gtfs_attributions add constraint gtfs_attributions_agency_id_fkey foreign key (agency_id) REFERENCES gtfs_agencies(id);
create index on gtfs_attributions(agency_id);
create index on gtfs_attributions(route_id);
create index on gtfs_attributions(trip_id);

create index on gtfs_levels(parent_station);

COMMIT;
