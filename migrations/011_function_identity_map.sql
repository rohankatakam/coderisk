-- Migration 011: Function Identity Map
-- Tracks function renames over time to maintain graph consistency
-- Author: Agent 1 (Schema Migrations)
-- Date: 2025-11-19

-- ============================================================================
-- PART 1: Create function_identity_map table
-- ============================================================================

CREATE TABLE IF NOT EXISTS function_identity_map (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    block_id BIGINT NOT NULL REFERENCES code_blocks(id) ON DELETE CASCADE,
    historical_name TEXT NOT NULL,
    signature TEXT,
    commit_sha TEXT NOT NULL,
    rename_date TIMESTAMP WITHOUT TIME ZONE NOT NULL,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW()
);

COMMENT ON TABLE function_identity_map IS 'Tracks function rename history. Enables graph consistency when functions are renamed (similar to file_identity_map for files).';
COMMENT ON COLUMN function_identity_map.block_id IS 'Current code_block ID (after rename)';
COMMENT ON COLUMN function_identity_map.historical_name IS 'Previous function name before this rename';
COMMENT ON COLUMN function_identity_map.signature IS 'Function signature at time of rename (optional)';
COMMENT ON COLUMN function_identity_map.commit_sha IS 'Commit where the rename occurred';
COMMENT ON COLUMN function_identity_map.rename_date IS 'Timestamp of the rename commit';

-- ============================================================================
-- PART 2: Create indexes for function_identity_map
-- ============================================================================

-- Index for lookups by block_id (find all historical names for a function)
CREATE INDEX IF NOT EXISTS idx_function_identity_map_block
ON function_identity_map(block_id);

-- Index for reverse lookups (find current block_id by historical name)
CREATE INDEX IF NOT EXISTS idx_function_identity_map_name
ON function_identity_map(repo_id, historical_name);

-- Index for commit-based queries
CREATE INDEX IF NOT EXISTS idx_function_identity_map_commit
ON function_identity_map(commit_sha);

COMMENT ON INDEX idx_function_identity_map_block IS 'Fast lookup of all historical names for a given function';
COMMENT ON INDEX idx_function_identity_map_name IS 'Fast reverse lookup: historical name → current block_id';
COMMENT ON INDEX idx_function_identity_map_commit IS 'Find all renames in a specific commit';

-- ============================================================================
-- PART 3: Add historical_block_names JSONB column to code_blocks
-- ============================================================================

-- Add JSONB column for quick access to all historical names
ALTER TABLE code_blocks
ADD COLUMN IF NOT EXISTS historical_block_names JSONB DEFAULT '[]'::JSONB;

COMMENT ON COLUMN code_blocks.historical_block_names IS 'JSONB array of all historical names for this function. Enables fast "has this function ever been named X?" queries. Example: ["handleLogin", "processLogin"]';

-- ============================================================================
-- PART 4: Create GIN index for JSONB containment queries
-- ============================================================================

-- GIN index for fast containment queries (@> operator)
CREATE INDEX IF NOT EXISTS idx_code_blocks_historical_names
ON code_blocks USING GIN(historical_block_names);

COMMENT ON INDEX idx_code_blocks_historical_names IS 'GIN index for fast JSONB containment queries. Enables: WHERE historical_block_names @> ''["oldName"]''';

-- ============================================================================
-- VALIDATION: Verify migration success
-- ============================================================================

DO $$
DECLARE
    table_exists BOOLEAN;
    historical_names_exists BOOLEAN;
    block_index_exists BOOLEAN;
    name_index_exists BOOLEAN;
    commit_index_exists BOOLEAN;
    gin_index_exists BOOLEAN;
BEGIN
    -- Check function_identity_map table exists
    SELECT EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_name = 'function_identity_map'
    ) INTO table_exists;

    -- Check historical_block_names column exists
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'code_blocks' AND column_name = 'historical_block_names'
    ) INTO historical_names_exists;

    -- Check indexes exist
    SELECT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'idx_function_identity_map_block'
    ) INTO block_index_exists;

    SELECT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'idx_function_identity_map_name'
    ) INTO name_index_exists;

    SELECT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'idx_function_identity_map_commit'
    ) INTO commit_index_exists;

    SELECT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'idx_code_blocks_historical_names'
    ) INTO gin_index_exists;

    -- Validate all components exist
    IF NOT table_exists THEN
        RAISE EXCEPTION 'Migration 011 failed: function_identity_map table not created';
    END IF;

    IF NOT historical_names_exists THEN
        RAISE EXCEPTION 'Migration 011 failed: historical_block_names column not created';
    END IF;

    IF NOT block_index_exists THEN
        RAISE EXCEPTION 'Migration 011 failed: block index not created';
    END IF;

    IF NOT name_index_exists THEN
        RAISE EXCEPTION 'Migration 011 failed: name index not created';
    END IF;

    IF NOT commit_index_exists THEN
        RAISE EXCEPTION 'Migration 011 failed: commit index not created';
    END IF;

    IF NOT gin_index_exists THEN
        RAISE EXCEPTION 'Migration 011 failed: GIN index not created';
    END IF;

    RAISE NOTICE '✅ Migration 011 completed successfully:';
    RAISE NOTICE '   - function_identity_map table created';
    RAISE NOTICE '   - 3 B-tree indexes created (block, name, commit)';
    RAISE NOTICE '   - historical_block_names JSONB column added';
    RAISE NOTICE '   - GIN index created for fast containment queries';
    RAISE NOTICE '   - Function rename tracking now supported';
END $$;
