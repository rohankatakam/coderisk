-- Dead Letter Queue for failed commits
-- Stores commits that failed processing for manual review and retry

CREATE TABLE IF NOT EXISTS dead_letter_queue (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    commit_sha TEXT NOT NULL,
    error_message TEXT,
    error_stack TEXT,
    retry_count INTEGER DEFAULT 0,
    last_retry_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    metadata JSONB DEFAULT '{}', -- Additional context (file_path, block_name, etc.)
    UNIQUE(repo_id, commit_sha)
);

CREATE INDEX IF NOT EXISTS idx_dlq_repo_id ON dead_letter_queue(repo_id);
CREATE INDEX IF NOT EXISTS idx_dlq_created_at ON dead_letter_queue(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_dlq_retry_count ON dead_letter_queue(retry_count) WHERE retry_count < 5;

COMMENT ON TABLE dead_letter_queue IS 'Stores commits that failed processing during atomization';
COMMENT ON COLUMN dead_letter_queue.retry_count IS 'Number of retry attempts (max 5 before manual intervention required)';
COMMENT ON COLUMN dead_letter_queue.metadata IS 'JSON containing additional context: {file_path, block_name, llm_error, etc.}';
