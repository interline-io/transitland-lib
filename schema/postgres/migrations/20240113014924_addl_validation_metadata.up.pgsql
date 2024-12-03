BEGIN;

ALTER TABLE tl_validation_reports ADD COLUMN validator text;
ALTER TABLE tl_validation_reports ADD COLUMN validator_version text;
ALTER TABLE tl_validation_reports ADD COLUMN success bool;
ALTER TABLE tl_validation_reports ADD COLUMN failure_reason text;

COMMIT;