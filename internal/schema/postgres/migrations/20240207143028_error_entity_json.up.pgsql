BEGIN;

ALTER TABLE tl_validation_report_error_exemplars ADD COLUMN entity_json jsonb;

COMMIT;
