BEGIN;

-- Soft-delete marker for job artifacts. Setting deleted_at hides the artifact from
-- reads (the jobserver filters deleted_at IS NULL) without dropping the row or its
-- stored bytes; a later blob-culling job reaps the bytes and the row. NULL = live.
ALTER TABLE tl_job_artifacts ADD COLUMN deleted_at timestamp without time zone;

COMMIT;
