BEGIN;

alter table tl_route_geometries add column length double precision;
alter table tl_route_geometries add column max_segment_length double precision;

COMMIT;