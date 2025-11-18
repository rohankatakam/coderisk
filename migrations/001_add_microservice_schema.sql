-- Migration 001: Add microservice architecture schema columns
-- Based on DATA_SCHEMA_REFERENCE.md and microservice_arch.md specifications

-- ========================================
-- PART 1: github_repositories - Add ingestion tracking and force-push detection
-- ========================================

ALTER TABLE github_repositories
ADD COLUMN IF NOT EXISTS html_url TEXT,
ADD COLUMN IF NOT EXISTS clone_url TEXT,
ADD COLUMN IF NOT EXISTS created_at TIMESTAMP WITHOUT TIME ZONE,
ADD COLUMN IF NOT EXISTS pushed_at TIMESTAMP WITHOUT TIME ZONE,
ADD COLUMN IF NOT EXISTS size INTEGER,
ADD COLUMN IF NOT EXISTS stargazers_count INTEGER,
ADD COLUMN IF NOT EXISTS watchers_count INTEGER,
ADD COLUMN IF NOT EXISTS language TEXT,
ADD COLUMN IF NOT EXISTS forks_count INTEGER,
ADD COLUMN IF NOT EXISTS open_issues_count INTEGER,
ADD COLUMN IF NOT EXISTS is_fork BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS is_private BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS ingestion_started_at TIMESTAMP WITHOUT TIME ZONE,
ADD COLUMN IF NOT EXISTS ingestion_completed_at TIMESTAMP WITHOUT TIME ZONE,
ADD COLUMN IF NOT EXISTS ingestion_status TEXT DEFAULT 'pending' CHECK (ingestion_status IN ('pending', 'in_progress', 'completed', 'failed')),
ADD COLUMN IF NOT EXISTS parent_shas_hash TEXT; -- SHA256 hash for force-push detection

COMMENT ON COLUMN github_repositories.parent_shas_hash IS 'SHA256(concatenated parent SHAs) for detecting force-pushes and history rewrites';
COMMENT ON COLUMN github_repositories.ingestion_status IS 'Tracks overall ingestion pipeline status: pending → in_progress → completed/failed';

-- ========================================
-- PART 2: github_commits - Add topological ordering
-- ========================================

ALTER TABLE github_commits
ADD COLUMN IF NOT EXISTS tree_sha TEXT,
ADD COLUMN IF NOT EXISTS parent_shas TEXT[],
ADD COLUMN IF NOT EXISTS topological_index INTEGER;

-- Create index for topological ordering (used by crisk-atomize)
CREATE INDEX IF NOT EXISTS idx_commits_topological
ON github_commits(repo_id, topological_index)
WHERE topological_index IS NOT NULL;

COMMENT ON COLUMN github_commits.topological_index IS 'Topological sort order from git rev-list --topo-order (parent before child). Critical for correct semantic processing.';
COMMENT ON COLUMN github_commits.parent_shas IS 'Array of parent commit SHAs. Used for topological computation and force-push detection.';

-- ========================================
-- PART 3: github_issues - Add timeline tracking
-- ========================================

-- Check if github_issue_timeline table exists, create if missing
CREATE TABLE IF NOT EXISTS github_issue_timeline (
    id BIGSERIAL PRIMARY KEY,
    issue_id BIGINT NOT NULL REFERENCES github_issues(id) ON DELETE CASCADE,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    actor TEXT,
    created_at TIMESTAMP WITHOUT TIME ZONE,
    commit_sha TEXT,
    source_issue_number INTEGER,
    source_pr_number INTEGER,
    raw_data JSONB
);

CREATE INDEX IF NOT EXISTS idx_timeline_issue ON github_issue_timeline(issue_id, event_type);
CREATE INDEX IF NOT EXISTS idx_timeline_commit ON github_issue_timeline(repo_id, commit_sha) WHERE commit_sha IS NOT NULL;

COMMENT ON TABLE github_issue_timeline IS 'Timeline events for issues/PRs (cross-references, closes events). Enables incident linking.';

-- ========================================
-- PART 4: code_blocks - Add canonical_file_path for graph consistency
-- ========================================

-- Add canonical_file_path to code_blocks (if not exists)
ALTER TABLE code_blocks
ADD COLUMN IF NOT EXISTS canonical_file_path TEXT,
ADD COLUMN IF NOT EXISTS path_at_creation TEXT;

-- Update existing records to populate canonical_file_path from file_path
UPDATE code_blocks
SET canonical_file_path = file_path
WHERE canonical_file_path IS NULL;

-- Add comment explaining the critical importance of canonical paths
COMMENT ON COLUMN code_blocks.canonical_file_path IS 'Current file path at HEAD (resolved via file_identity_map). CRITICAL: This enables perfect graph consistency even after file renames. All CodeBlocks for a renamed file reference the SAME canonical path.';
COMMENT ON COLUMN code_blocks.path_at_creation IS 'File path when this block was first created (historical reference). Preserves context.';

-- Update UNIQUE constraint to use canonical_file_path instead of file_path
-- (Note: start_line is excluded to handle line shifts - fuzzy LLM entity resolution handles duplicates)
DO $$
BEGIN
    -- Drop old constraint if exists
    ALTER TABLE code_blocks DROP CONSTRAINT IF EXISTS code_blocks_repo_file_block_unique;
    ALTER TABLE code_blocks DROP CONSTRAINT IF EXISTS code_blocks_unique_block;

    -- Add new constraint with canonical_file_path (excluding start_line)
    ALTER TABLE code_blocks
    ADD CONSTRAINT code_blocks_canonical_unique
    UNIQUE (repo_id, canonical_file_path, block_name);

EXCEPTION
    WHEN duplicate_table THEN NULL;
    WHEN duplicate_object THEN NULL;
END $$;

-- ========================================
-- PART 5: ingestion_jobs - Job tracking table
-- ========================================

CREATE TABLE IF NOT EXISTS ingestion_jobs (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    job_type TEXT NOT NULL CHECK (job_type IN ('stage', 'ingest', 'atomize', 'index-incident', 'index-ownership', 'index-coupling')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    started_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITHOUT TIME ZONE,
    error_message TEXT,
    metadata JSONB,
    CONSTRAINT unique_running_job UNIQUE (repo_id, job_type, status) DEFERRABLE
);

CREATE INDEX IF NOT EXISTS idx_jobs_repo ON ingestion_jobs(repo_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON ingestion_jobs(status, started_at DESC) WHERE status IN ('running', 'failed');

COMMENT ON TABLE ingestion_jobs IS 'Tracks microservice execution state per repository. Enables monitoring, SLA reporting, and failure recovery.';
COMMENT ON COLUMN ingestion_jobs.metadata IS 'Job-specific stats: commits_fetched, nodes_created, llm_calls, processing_time_sec, etc.';

-- ========================================
-- PART 6: code_block_changes - Add commit_time_path for path resolution tracking
-- ========================================

CREATE TABLE IF NOT EXISTS code_block_changes (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    commit_sha TEXT NOT NULL,
    block_id BIGINT REFERENCES code_blocks(id) ON DELETE SET NULL,
    canonical_file_path TEXT NOT NULL,
    commit_time_path TEXT NOT NULL,
    block_type TEXT,
    block_name TEXT NOT NULL,
    change_type TEXT NOT NULL CHECK (change_type IN ('created', 'modified', 'deleted', 'renamed')),
    old_name TEXT,
    lines_added INTEGER DEFAULT 0,
    lines_deleted INTEGER DEFAULT 0,
    complexity_delta INTEGER,
    change_summary TEXT,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_block_changes_commit ON code_block_changes(commit_sha);
CREATE INDEX IF NOT EXISTS idx_block_changes_block ON code_block_changes(block_id) WHERE block_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_block_changes_repo ON code_block_changes(repo_id);

COMMENT ON TABLE code_block_changes IS 'Change events extracted from commits via LLM. Tracks both canonical_file_path (for graph lookups) and commit_time_path (for historical context).';
COMMENT ON COLUMN code_block_changes.canonical_file_path IS 'Resolved canonical path using file_identity_map. Used for graph consistency.';
COMMENT ON COLUMN code_block_changes.commit_time_path IS 'File path as it appeared in the commit diff. Preserves historical context.';

-- ========================================
-- PART 7: code_block_imports - Dependency tracking
-- ========================================

CREATE TABLE IF NOT EXISTS code_block_imports (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    source_block_id BIGINT NOT NULL REFERENCES code_blocks(id) ON DELETE CASCADE,
    target_module TEXT NOT NULL,
    target_symbol TEXT,
    import_type TEXT CHECK (import_type IN ('internal', 'external')),
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_block_imports_source ON code_block_imports(source_block_id);
CREATE INDEX IF NOT EXISTS idx_block_imports_target ON code_block_imports(repo_id, target_module);

COMMENT ON TABLE code_block_imports IS 'Explicit dependency edges extracted from code (IMPORTS_FROM relationships). Enables coupling analysis.';

-- ========================================
-- PART 8: code_block_incidents - Incident linking
-- ========================================

CREATE TABLE IF NOT EXISTS code_block_incidents (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    block_id BIGINT NOT NULL REFERENCES code_blocks(id) ON DELETE CASCADE,
    issue_id BIGINT NOT NULL REFERENCES github_issues(id) ON DELETE CASCADE,
    commit_sha TEXT,
    incident_date TIMESTAMP WITHOUT TIME ZONE,
    resolution_date TIMESTAMP WITHOUT TIME ZONE,
    incident_type TEXT,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    UNIQUE(block_id, issue_id)
);

CREATE INDEX IF NOT EXISTS idx_block_incidents_block ON code_block_incidents(block_id, incident_date DESC);
CREATE INDEX IF NOT EXISTS idx_block_incidents_issue ON code_block_incidents(issue_id);

COMMENT ON TABLE code_block_incidents IS 'Links incidents (issues) to CodeBlocks that were modified to fix them. Powers temporal risk signals.';

-- ========================================
-- PART 9: code_block_coupling - Co-change relationships
-- ========================================

CREATE TABLE IF NOT EXISTS code_block_coupling (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    block_a_id BIGINT NOT NULL REFERENCES code_blocks(id) ON DELETE CASCADE,
    block_b_id BIGINT NOT NULL REFERENCES code_blocks(id) ON DELETE CASCADE,
    co_change_count INTEGER NOT NULL DEFAULT 0,
    total_changes INTEGER NOT NULL DEFAULT 0,
    coupling_rate FLOAT NOT NULL DEFAULT 0.0,
    first_co_change TIMESTAMP WITHOUT TIME ZONE,
    last_co_change TIMESTAMP WITHOUT TIME ZONE,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    UNIQUE(block_a_id, block_b_id),
    CHECK(block_a_id < block_b_id)
);

CREATE INDEX IF NOT EXISTS idx_block_coupling_a ON code_block_coupling(block_a_id, coupling_rate DESC);
CREATE INDEX IF NOT EXISTS idx_block_coupling_rate ON code_block_coupling(repo_id, coupling_rate DESC);

COMMENT ON TABLE code_block_coupling IS 'Co-change relationships between CodeBlocks (implicit coupling). Identifies blocks that change together.';

-- ========================================
-- PART 10: Add risk indexing columns to code_blocks
-- ========================================

ALTER TABLE code_blocks
ADD COLUMN IF NOT EXISTS incident_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS last_incident_date TIMESTAMP WITHOUT TIME ZONE,
ADD COLUMN IF NOT EXISTS temporal_summary TEXT,
ADD COLUMN IF NOT EXISTS original_author_email TEXT,
ADD COLUMN IF NOT EXISTS last_modifier_email TEXT,
ADD COLUMN IF NOT EXISTS staleness_days INTEGER,
ADD COLUMN IF NOT EXISTS familiarity_map JSONB,
ADD COLUMN IF NOT EXISTS co_change_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS avg_coupling_rate FLOAT,
ADD COLUMN IF NOT EXISTS risk_score FLOAT;

-- Create indexes for risk queries
CREATE INDEX IF NOT EXISTS idx_code_blocks_risk ON code_blocks(repo_id, risk_score DESC) WHERE risk_score IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_code_blocks_incidents ON code_blocks(repo_id, incident_count DESC) WHERE incident_count > 0;
CREATE INDEX IF NOT EXISTS idx_code_blocks_staleness ON code_blocks(repo_id, staleness_days DESC) WHERE staleness_days IS NOT NULL;

COMMENT ON COLUMN code_blocks.temporal_summary IS 'LLM-generated summary of incident history (e.g., "History of crashes and save failures"). Populated by crisk-index-incident.';
COMMENT ON COLUMN code_blocks.familiarity_map IS 'JSONB map of developer emails to modification counts. Enables knowledge risk assessment.';
COMMENT ON COLUMN code_blocks.risk_score IS 'Composite risk score (0-100) combining temporal, ownership, and coupling signals. Calculated by crisk-index-coupling (final step).';

-- ========================================
-- VALIDATION: Verify all critical columns exist
-- ========================================

DO $$
DECLARE
    missing_columns TEXT[] := ARRAY[]::TEXT[];
BEGIN
    -- Check critical columns
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'github_commits' AND column_name = 'topological_index') THEN
        missing_columns := array_append(missing_columns, 'github_commits.topological_index');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'github_repositories' AND column_name = 'parent_shas_hash') THEN
        missing_columns := array_append(missing_columns, 'github_repositories.parent_shas_hash');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'code_blocks' AND column_name = 'canonical_file_path') THEN
        missing_columns := array_append(missing_columns, 'code_blocks.canonical_file_path');
    END IF;

    IF array_length(missing_columns, 1) > 0 THEN
        RAISE EXCEPTION 'Migration incomplete. Missing columns: %', array_to_string(missing_columns, ', ');
    END IF;

    RAISE NOTICE '✅ Migration 001 completed successfully. All microservice schema columns added.';
END $$;
