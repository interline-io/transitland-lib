-- Create tl_job_artifacts table for storing job outputs
CREATE TABLE tl_job_artifacts (
    id UUID PRIMARY KEY,
    job_id BIGINT NOT NULL REFERENCES river_job(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    inline_data JSONB,           -- For inline JSONB data
    cloud_storage_ref JSONB,     -- For cloud storage references
    
    -- Indexes
    INDEX idx_tl_job_artifacts_job_id (job_id),
    INDEX idx_tl_job_artifacts_created_at (created_at ASC)
);

-- Create tl_job_logs table for storing job execution logs
CREATE TABLE tl_job_logs (
    id UUID PRIMARY KEY,
    job_id BIGINT NOT NULL REFERENCES river_job(id) ON DELETE CASCADE,
    level TEXT NOT NULL, -- 'info', 'warn', 'error', 'debug'
    message TEXT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    metadata JSONB, -- Additional structured data
    
    -- Indexes
    INDEX idx_tl_job_logs_job_id (job_id),
    INDEX idx_tl_job_logs_timestamp (timestamp ASC),
    INDEX idx_tl_job_logs_level (level)
);