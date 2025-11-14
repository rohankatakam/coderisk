-- Migration: Add Code-Block Atomization Schema
-- Purpose: Enable function-level risk analysis for "crisk check" demo
-- Date: 2025-11-14
-- Ref: POSTGRES_STAGING_VALIDATION_REPORT.md

-- ============================================================================
-- TABLE 1: code_blocks
-- Stores individual functions, methods, classes extracted from commits
-- ============================================================================
CREATE TABLE IF NOT EXISTS code_blocks (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,

    -- Location
    file_path TEXT NOT NULL,
    block_name TEXT NOT NULL,          -- e.g., "updateTableEditor", "TableEditor::render"
    block_type TEXT NOT NULL,          -- "function", "method", "class", "component"
    start_line INTEGER,
    end_line INTEGER,

    -- Metadata
    language TEXT,                     -- "typescript", "python", "go", "javascript"
    signature TEXT,                    -- Function signature for context

    -- Lifecycle tracking
    first_seen_commit_sha VARCHAR(40),
    last_modified_commit_sha VARCHAR(40),
    last_modified_at TIMESTAMP,

    -- Computed risk metrics (updated by Pipeline 3)
    incident_count INTEGER DEFAULT 0,
    total_modifications INTEGER DEFAULT 0,
    staleness_days INTEGER,

    -- Ownership tracking
    original_author_email VARCHAR(255),
    last_modifier_email VARCHAR(255),

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT unique_repo_file_block UNIQUE(repo_id, file_path, block_name)
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_code_blocks_repo_file ON code_blocks(repo_id, file_path);
CREATE INDEX IF NOT EXISTS idx_code_blocks_name ON code_blocks(block_name);
CREATE INDEX IF NOT EXISTS idx_code_blocks_incidents ON code_blocks(incident_count DESC) WHERE incident_count > 0;
CREATE INDEX IF NOT EXISTS idx_code_blocks_type ON code_blocks(block_type);
CREATE INDEX IF NOT EXISTS idx_code_blocks_last_modified ON code_blocks(last_modified_at DESC);

COMMENT ON TABLE code_blocks IS 'Individual code blocks (functions, methods, classes) extracted from commits for function-level risk analysis';

-- ============================================================================
-- TABLE 2: code_block_modifications
-- Tracks every commit that modified a specific code block
-- ============================================================================
CREATE TABLE IF NOT EXISTS code_block_modifications (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL,
    code_block_id BIGINT NOT NULL REFERENCES code_blocks(id) ON DELETE CASCADE,
    commit_sha VARCHAR(40) NOT NULL,

    -- Modification metadata
    author_email VARCHAR(255),
    modified_at TIMESTAMP,
    additions INTEGER DEFAULT 0,
    deletions INTEGER DEFAULT 0,

    -- LLM-extracted context
    patch_snippet TEXT,                -- Relevant portion of the diff affecting this block
    modification_summary TEXT,         -- LLM-generated summary (optional)

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT unique_block_commit UNIQUE(code_block_id, commit_sha)
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_block_mods_block ON code_block_modifications(code_block_id);
CREATE INDEX IF NOT EXISTS idx_block_mods_commit ON code_block_modifications(commit_sha);
CREATE INDEX IF NOT EXISTS idx_block_mods_author ON code_block_modifications(author_email);
CREATE INDEX IF NOT EXISTS idx_block_mods_date ON code_block_modifications(modified_at DESC);

COMMENT ON TABLE code_block_modifications IS 'History of all modifications to each code block across commits';

-- ============================================================================
-- TABLE 3: code_block_incidents
-- Links code blocks to incidents they caused (for WAS_ROOT_CAUSE_IN edges)
-- ============================================================================
CREATE TABLE IF NOT EXISTS code_block_incidents (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL,
    code_block_id BIGINT NOT NULL REFERENCES code_blocks(id) ON DELETE CASCADE,
    issue_id BIGINT NOT NULL REFERENCES github_issues(id) ON DELETE CASCADE,

    -- Evidence and confidence
    confidence DECIMAL(3,2) NOT NULL CHECK (confidence >= 0.70 AND confidence <= 1.00),
    evidence_source TEXT NOT NULL,    -- "llm_extraction", "timeline_event", "commit_message", "pr_description"
    evidence_text TEXT,                -- Snippet of text that links the block to the incident

    -- Fix tracking
    fix_commit_sha VARCHAR(40),        -- The commit that fixed this incident
    fixed_at TIMESTAMP,

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT unique_block_issue UNIQUE(code_block_id, issue_id)
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_block_incidents_block ON code_block_incidents(code_block_id);
CREATE INDEX IF NOT EXISTS idx_block_incidents_issue ON code_block_incidents(issue_id);
CREATE INDEX IF NOT EXISTS idx_block_incidents_confidence ON code_block_incidents(confidence DESC);
CREATE INDEX IF NOT EXISTS idx_block_incidents_repo ON code_block_incidents(repo_id);

COMMENT ON TABLE code_block_incidents IS 'Links code blocks to incidents they caused, with confidence scores and evidence';

-- ============================================================================
-- TABLE 4: code_block_co_changes
-- Tracks code blocks that frequently change together (coupling risk)
-- ============================================================================
CREATE TABLE IF NOT EXISTS code_block_co_changes (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL,
    block_a_id BIGINT NOT NULL REFERENCES code_blocks(id) ON DELETE CASCADE,
    block_b_id BIGINT NOT NULL REFERENCES code_blocks(id) ON DELETE CASCADE,

    -- Co-change metrics
    co_change_count INTEGER DEFAULT 0,
    co_change_rate DECIMAL(3,2),      -- 0.00-1.00 (what % of commits that touched A also touched B)
    last_co_changed_at TIMESTAMP,
    last_co_change_commit_sha VARCHAR(40),

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT unique_block_pair UNIQUE(block_a_id, block_b_id),
    CONSTRAINT ordered_pair CHECK(block_a_id < block_b_id)  -- Ensure (A,B) not (B,A) to avoid duplicates
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_block_cochange_a ON code_block_co_changes(block_a_id);
CREATE INDEX IF NOT EXISTS idx_block_cochange_b ON code_block_co_changes(block_b_id);
CREATE INDEX IF NOT EXISTS idx_block_cochange_rate ON code_block_co_changes(co_change_rate DESC) WHERE co_change_rate >= 0.50;
CREATE INDEX IF NOT EXISTS idx_block_cochange_count ON code_block_co_changes(co_change_count DESC) WHERE co_change_count >= 3;

COMMENT ON TABLE code_block_co_changes IS 'Tracks code blocks that frequently change together across commits';

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Function to update code_block.updated_at timestamp
CREATE OR REPLACE FUNCTION update_code_block_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER code_blocks_updated_at
    BEFORE UPDATE ON code_blocks
    FOR EACH ROW
    EXECUTE FUNCTION update_code_block_timestamp();

-- Function to update co_change.updated_at timestamp
CREATE OR REPLACE FUNCTION update_co_change_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER code_block_co_changes_updated_at
    BEFORE UPDATE ON code_block_co_changes
    FOR EACH ROW
    EXECUTE FUNCTION update_co_change_timestamp();

-- ============================================================================
-- VALIDATION QUERIES (for testing after migration)
-- ============================================================================

-- Check that all tables were created
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'code_blocks') THEN
        RAISE EXCEPTION 'Migration failed: code_blocks table not created';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'code_block_modifications') THEN
        RAISE EXCEPTION 'Migration failed: code_block_modifications table not created';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'code_block_incidents') THEN
        RAISE EXCEPTION 'Migration failed: code_block_incidents table not created';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'code_block_co_changes') THEN
        RAISE EXCEPTION 'Migration failed: code_block_co_changes table not created';
    END IF;

    RAISE NOTICE '✅ Migration successful: All 4 tables created';
    RAISE NOTICE '   - code_blocks';
    RAISE NOTICE '   - code_block_modifications';
    RAISE NOTICE '   - code_block_incidents';
    RAISE NOTICE '   - code_block_co_changes';
END $$;
