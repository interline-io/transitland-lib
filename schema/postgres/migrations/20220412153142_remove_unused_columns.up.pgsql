BEGIN;

ALTER TABLE tl_route_headways DROP CONSTRAINT fk_rails_078ffc5894;
ALTER TABLE tl_route_headways DROP COLUMN service_id;

COMMIT;

