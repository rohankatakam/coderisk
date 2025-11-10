-- Migration: Add investigation checkpoints table for directive agent
-- Purpose: Store investigation state for pause/resume workflows
-- Date: 2025-01-09

-- Create investigations table for checkpoint storage
CREATE TABLE IF NOT EXISTS investigations (
    id TEXT PRIMARY KEY,
    phase TEXT NOT NULL,
    terminal_state TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Context
    file_paths JSONB NOT NULL,
    repository_id BIGINT,
    phase1_results JSONB,

    -- State data (full JSON snapshot for flexibility)
    state_data JSONB NOT NULL,

    -- Decision history
    decisions JSONB NOT NULL DEFAULT '[]'::jsonb,

    -- Resumption
    can_resume BOOLEAN NOT NULL DEFAULT true,
    resume_data JSONB,

    -- Final assessment
    final_risk_level TEXT,
    final_confidence DOUBLE PRECISION,
    final_summary TEXT,

    -- Indexing
    CONSTRAINT terminal_state_check CHECK (terminal_state IN (
        'SAFE_TO_COMMIT',
        'RISKS_UNRESOLVED',
        'BLOCKED_WAITING',
        'INVESTIGATION_INCOMPLETE',
        'INVESTIGATION_ABORTED'
    )),
    CONSTRAINT phase_check CHECK (phase IN (
        'INITIALIZED',
        'PHASE_1_RUNNING',
        'PHASE_2_INVESTIGATING',
        'AWAITING_HUMAN',
        'ASSESSMENT_COMPLETE'
    ))
);

-- Index for quick lookup by ID (primary key already indexed)
-- Index for finding resumable investigations
CREATE INDEX IF NOT EXISTS idx_investigations_can_resume
    ON investigations(can_resume)
    WHERE can_resume = true;

-- Index for finding recent investigations
CREATE INDEX IF NOT EXISTS idx_investigations_updated_at
    ON investigations(updated_at DESC);

-- Index for finding investigations by phase
CREATE INDEX IF NOT EXISTS idx_investigations_phase
    ON investigations(phase);

-- Index for finding investigations by repository
CREATE INDEX IF NOT EXISTS idx_investigations_repository_id
    ON investigations(repository_id)
    WHERE repository_id IS NOT NULL;

-- GIN index on file_paths for querying investigations by file
CREATE INDEX IF NOT EXISTS idx_investigations_file_paths
    ON investigations USING GIN(file_paths);

-- Comments for documentation
COMMENT ON TABLE investigations IS 'Stores investigation state for directive agent pause/resume workflows';
COMMENT ON COLUMN investigations.id IS 'Unique investigation ID (16-character hex)';
COMMENT ON COLUMN investigations.phase IS 'Current phase of investigation';
COMMENT ON COLUMN investigations.terminal_state IS 'Final state if investigation is complete';
COMMENT ON COLUMN investigations.file_paths IS 'Array of file paths being investigated';
COMMENT ON COLUMN investigations.state_data IS 'Complete investigation state snapshot as JSON';
COMMENT ON COLUMN investigations.decisions IS 'Array of decision points and user responses';
COMMENT ON COLUMN investigations.can_resume IS 'Whether investigation can be resumed';
COMMENT ON COLUMN investigations.resume_data IS 'Arbitrary key-value data for resumption';
