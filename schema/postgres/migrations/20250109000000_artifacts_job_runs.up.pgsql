BEGIN;

-- Job runs table: optional context for artifacts, tracks workflow execution
CREATE TABLE job_runs (
    id bigserial PRIMARY KEY,
    job_type text NOT NULL,
    job_args jsonb NOT NULL DEFAULT '{}',  -- Job arguments (map[string]any)
    status text NOT NULL CHECK (status IN ('pending', 'running', 'success', 'failed', 'cancelled')),
    started_at timestamptz,
    completed_at timestamptz,
    metadata jsonb NOT NULL DEFAULT '{}',
    metrics jsonb NOT NULL DEFAULT '{}',
    log_summary text,
    error_message text,
    created_by text,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    
    -- Ensure completed_at is set for finalized statuses
    CONSTRAINT job_runs_completion_check CHECK (
        (status IN ('success', 'failed', 'cancelled') AND completed_at IS NOT NULL) OR
        (status IN ('pending', 'running') AND completed_at IS NULL)
    )
);

-- Artifacts table: tracks derived data products, exports, and analyses
CREATE TABLE artifacts (
    id bigserial PRIMARY KEY,
    name text NOT NULL,
    artifact_type text NOT NULL,
    storage_type text NOT NULL CHECK (storage_type IN ('inline', 's3', 'azure')),
    inline_json_data jsonb,  -- For inline storage of small artifacts (JSONB for structured data)
    storage_url text,  -- Full storage URL: s3://bucket.s3.region.amazonaws.com/path or az://account/container/path
    content_type text,
    size_bytes bigint,
    metadata jsonb NOT NULL DEFAULT '{}',
    job_run_id bigint REFERENCES job_runs(id) ON DELETE SET NULL,  -- Optional: artifact belongs to at most one job run
    created_by text,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    
    -- Ensure either inline_json_data (inline) or storage_url (external) is set
    CONSTRAINT artifacts_storage_check CHECK (
        (storage_type = 'inline' AND inline_json_data IS NOT NULL AND storage_url IS NULL) OR
        (storage_type IN ('s3', 'azure') AND storage_url IS NOT NULL AND inline_json_data IS NULL)
    ),
    
    -- Size limit for inline storage (10MB default, configurable)
    CONSTRAINT artifacts_inline_size_check CHECK (
        storage_type != 'inline' OR pg_column_size(inline_json_data) <= 10485760
    )
);

-- Link artifacts to feed versions for lineage tracking
CREATE TABLE artifacts_feed_versions (
    artifact_id bigint NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    feed_version_id bigint NOT NULL REFERENCES feed_versions(id) ON DELETE CASCADE,
    relationship_type text NOT NULL CHECK (relationship_type IN ('input', 'output')),
    PRIMARY KEY (artifact_id, feed_version_id, relationship_type)
);

-- Indexes for artifacts
CREATE INDEX artifacts_artifact_type_idx ON artifacts(artifact_type, created_at DESC);
CREATE INDEX artifacts_created_by_idx ON artifacts(created_by, created_at DESC) WHERE created_by IS NOT NULL;
CREATE INDEX artifacts_storage_url_idx ON artifacts(storage_url) WHERE storage_url IS NOT NULL;
CREATE INDEX artifacts_job_run_id_idx ON artifacts(job_run_id) WHERE job_run_id IS NOT NULL;
CREATE INDEX artifacts_metadata_idx ON artifacts USING GIN(metadata);

-- Indexes for job_runs
CREATE INDEX job_runs_status_idx ON job_runs(status, created_at DESC);
CREATE INDEX job_runs_job_type_idx ON job_runs(job_type, created_at DESC);
CREATE INDEX job_runs_created_by_idx ON job_runs(created_by, created_at DESC) WHERE created_by IS NOT NULL;
CREATE INDEX job_runs_job_args_idx ON job_runs USING GIN(job_args);
CREATE INDEX job_runs_metadata_idx ON job_runs USING GIN(metadata);
CREATE INDEX job_runs_metrics_idx ON job_runs USING GIN(metrics);

-- Indexes for join tables
CREATE INDEX artifacts_feed_versions_artifact_id_idx ON artifacts_feed_versions(artifact_id);
CREATE INDEX artifacts_feed_versions_feed_version_id_idx ON artifacts_feed_versions(feed_version_id);

COMMIT;

