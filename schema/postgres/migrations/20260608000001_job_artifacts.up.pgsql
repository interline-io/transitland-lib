BEGIN;

-- Files produced by background jobs (exports, reports, generated CSVs) and made
-- available to the submitting user for download. One row per artifact; the bytes
-- live in the configured storage backend (request.Store) under storage_key.
--
-- job_id is OPAQUE and intentionally NOT a foreign key: there is no canonical
-- jobs table in this schema. River owns river_job; the local and Argo backends
-- have no row here at all. job_id is the cross-backend correlation key only
-- (river_job id as text / local uuid / argo workflow name).
--
-- user_id is copied from Job.Opts.UserID at create time for display/audit ONLY;
-- it is never used as an authorization gate (downloads authorize via the job's
-- AccessPolicy, not this column).
CREATE TABLE tl_job_artifacts (
    id bigserial primary key NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    job_id text NOT NULL,
    job_kind text NOT NULL DEFAULT '',
    user_id text NOT NULL DEFAULT '',
    filename text NOT NULL,
    content_type text NOT NULL DEFAULT 'application/octet-stream',
    size_bytes bigint NOT NULL DEFAULT 0,
    sha1 text NOT NULL DEFAULT '',
    storage_key text NOT NULL
);
CREATE INDEX ON tl_job_artifacts(job_id);
CREATE INDEX ON tl_job_artifacts(user_id);

COMMIT;
