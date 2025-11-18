-- Migration: Add File Identity Map for Canonical Path Resolution
-- Purpose: Pre-compute file rename history to enable accurate file identity resolution
-- Date: 2025-11-17
-- Ref: ingestion_aws.md - Pipeline 1.0 (Pre-Ingestion: File Identity Resolution)

-- ============================================================================
-- TABLE: file_identity_map
-- Stores canonical file paths and their complete rename history
-- ============================================================================
-- This table solves the "file identity crisis" problem where chronological
-- commit processing creates separate File/CodeBlock nodes for renamed files.
--
-- Strategy:
-- 1. Pre-Ingestion Phase: Run `git log --follow` on all files at HEAD
-- 2. Store mapping: canonical_path (current) -> historical_paths (all past names)
-- 3. Graph Construction: Reference canonical_path in CodeBlock nodes
-- 4. Search: Instant lookup using canonical paths (no runtime git operations)
-- ============================================================================

CREATE TABLE IF NOT EXISTS file_identity_map (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,

    -- Identity mapping
    canonical_path TEXT NOT NULL,        -- Current path at HEAD (e.g., "shared/utils.py")
    historical_paths JSONB NOT NULL,     -- Array of all historical paths: ["src/utils.py", "lib/utils.py", "shared/utils.py"]

    -- Lifecycle tracking
    first_seen_commit_sha VARCHAR(40),   -- First commit where this file appeared (under any name)
    last_modified_commit_sha VARCHAR(40), -- Most recent commit that modified this file
    last_modified_at TIMESTAMP,          -- When the file was last modified

    -- Status tracking
    status TEXT DEFAULT 'active',        -- 'active', 'deleted', 'renamed'
    deleted_at TIMESTAMP,                -- If file was deleted, when did it happen?

    -- Metadata
    language TEXT,                       -- Detected language (for filtering)
    file_type TEXT,                      -- 'source', 'test', 'config', 'docs'

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    last_updated_at TIMESTAMP DEFAULT NOW(),

    -- Constraints
    CONSTRAINT unique_repo_canonical UNIQUE(repo_id, canonical_path),
    CONSTRAINT valid_status CHECK(status IN ('active', 'deleted', 'renamed'))
);

-- ============================================================================
-- INDEXES for efficient queries
-- ============================================================================

-- Primary lookup: Find canonical path by repo
CREATE INDEX IF NOT EXISTS idx_file_identity_repo
    ON file_identity_map(repo_id);

-- Reverse lookup: Find canonical path from any historical path
-- This is the critical index for search queries
CREATE INDEX IF NOT EXISTS idx_file_identity_historical
    ON file_identity_map USING GIN(historical_paths);

-- Filter by status (e.g., only active files)
CREATE INDEX IF NOT EXISTS idx_file_identity_status
    ON file_identity_map(repo_id, status)
    WHERE status = 'active';

-- Filter by language for language-specific queries
CREATE INDEX IF NOT EXISTS idx_file_identity_language
    ON file_identity_map(repo_id, language)
    WHERE language IS NOT NULL;

-- Lookup by last modified (for incremental update optimization)
CREATE INDEX IF NOT EXISTS idx_file_identity_last_modified
    ON file_identity_map(last_modified_at DESC);

COMMENT ON TABLE file_identity_map IS 'Pre-computed canonical file paths and rename history for accurate file identity resolution across commits';
COMMENT ON COLUMN file_identity_map.canonical_path IS 'Current file path at HEAD - used as canonical identifier in graph';
COMMENT ON COLUMN file_identity_map.historical_paths IS 'JSONB array of all historical paths this file has had, ordered chronologically';

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Function to update last_updated_at timestamp
CREATE OR REPLACE FUNCTION update_file_identity_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER file_identity_map_updated_at
    BEFORE UPDATE ON file_identity_map
    FOR EACH ROW
    EXECUTE FUNCTION update_file_identity_timestamp();

-- ============================================================================
-- LOOKUP HELPER FUNCTION
-- ============================================================================

-- Function to resolve any historical path to its canonical path
-- Usage: SELECT resolve_canonical_path(1, 'src/old_utils.py') -> 'shared/utils.py'
CREATE OR REPLACE FUNCTION resolve_canonical_path(
    p_repo_id BIGINT,
    p_historical_path TEXT
)
RETURNS TEXT AS $$
DECLARE
    v_canonical_path TEXT;
BEGIN
    -- First try exact match on canonical_path
    SELECT canonical_path INTO v_canonical_path
    FROM file_identity_map
    WHERE repo_id = p_repo_id
      AND canonical_path = p_historical_path
      AND status = 'active'
    LIMIT 1;

    -- If not found, search historical_paths JSONB array
    IF v_canonical_path IS NULL THEN
        SELECT canonical_path INTO v_canonical_path
        FROM file_identity_map
        WHERE repo_id = p_repo_id
          AND historical_paths @> jsonb_build_array(p_historical_path)
          AND status = 'active'
        LIMIT 1;
    END IF;

    RETURN v_canonical_path;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION resolve_canonical_path IS 'Resolves any historical file path to its current canonical path';

-- ============================================================================
-- BATCH LOOKUP HELPER FUNCTION
-- ============================================================================

-- Function to resolve multiple historical paths in one query
-- Usage: SELECT batch_resolve_canonical_paths(1, ARRAY['src/a.py', 'lib/b.py'])
-- Returns: JSON object mapping input paths to canonical paths
CREATE OR REPLACE FUNCTION batch_resolve_canonical_paths(
    p_repo_id BIGINT,
    p_historical_paths TEXT[]
)
RETURNS JSONB AS $$
DECLARE
    v_result JSONB := '{}'::JSONB;
    v_path TEXT;
    v_canonical TEXT;
BEGIN
    FOREACH v_path IN ARRAY p_historical_paths
    LOOP
        v_canonical := resolve_canonical_path(p_repo_id, v_path);
        IF v_canonical IS NOT NULL THEN
            v_result := v_result || jsonb_build_object(v_path, v_canonical);
        END IF;
    END LOOP;

    RETURN v_result;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION batch_resolve_canonical_paths IS 'Batch resolves multiple historical paths to canonical paths in one query';

-- ============================================================================
-- VALIDATION QUERIES (for testing after migration)
-- ============================================================================

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'file_identity_map') THEN
        RAISE EXCEPTION 'Migration failed: file_identity_map table not created';
    END IF;

    -- Verify GIN index was created
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE tablename = 'file_identity_map'
          AND indexname = 'idx_file_identity_historical'
    ) THEN
        RAISE EXCEPTION 'Migration failed: GIN index on historical_paths not created';
    END IF;

    -- Verify helper functions exist
    IF NOT EXISTS (
        SELECT 1 FROM pg_proc
        WHERE proname = 'resolve_canonical_path'
    ) THEN
        RAISE EXCEPTION 'Migration failed: resolve_canonical_path function not created';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_proc
        WHERE proname = 'batch_resolve_canonical_paths'
    ) THEN
        RAISE EXCEPTION 'Migration failed: batch_resolve_canonical_paths function not created';
    END IF;

    RAISE NOTICE '✅ Migration successful: file_identity_map infrastructure created';
    RAISE NOTICE '   - Table: file_identity_map';
    RAISE NOTICE '   - Indexes: 5 (including GIN index for historical_paths)';
    RAISE NOTICE '   - Functions: resolve_canonical_path, batch_resolve_canonical_paths';
    RAISE NOTICE '   - Trigger: last_updated_at auto-update';
END $$;
