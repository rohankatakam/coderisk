-- Migration 008: Add UNIQUE constraint to code_block_changes
-- Required for ON CONFLICT clause in atomizer
-- Author: Claude (Schema Alignment Migration)
-- Date: 2025-11-18

-- ============================================================================
-- Add UNIQUE constraint for atomizer ON CONFLICT
-- ============================================================================

-- The atomizer uses: ON CONFLICT (repo_id, commit_sha, block_id)
-- block_id can be NULL, so we need to handle that with COALESCE
CREATE UNIQUE INDEX IF NOT EXISTS idx_code_block_changes_unique
ON code_block_changes(repo_id, commit_sha, COALESCE(block_id, 0));

-- ============================================================================
-- Migration complete
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE 'Migration 008 complete: Added UNIQUE constraint to code_block_changes';
END $$;
