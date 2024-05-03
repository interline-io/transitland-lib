BEGIN;

ALTER TABLE tl_validation_report_error_groups ADD COLUMN group_key text NOT NULL DEFAULT 0;

COMMIT;