BEGIN;

alter table tl_route_geometries drop column shape_id;

COMMIT;