BEGIN;

ALTER TABLE tl_validation_report_error_groups ADD COLUMN level int NOT NULL DEFAULT 0;
ALTER TABLE tl_validation_reports ADD COLUMN includes_static bool NOT NULL DEFAULT false;
ALTER TABLE tl_validation_reports ADD COLUMN includes_rt bool NOT NULL DEFAULT false;

COMMIT;