BEGIN;

ALTER TABLE tl_validation_reports ADD COLUMN reported_at_local timestamp without time zone;
ALTER TABLE tl_validation_reports ADD COLUMN reported_at_local_timezone text;

COMMIT;