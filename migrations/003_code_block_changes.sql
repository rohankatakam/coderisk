-- Migration 003: Create code_block_changes table
-- Replaces deprecated code_block_modifications table
-- Aligns with DATA_SCHEMA_REFERENCE.md lines 282-306
-- Author: Claude (Schema Alignment Migration)
-- Date: 2025-11-18

-- ============================================================================
-- Create code_block_changes table (spec-compliant)
-- ============================================================================

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
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes per DATA_SCHEMA_REFERENCE.md line 304-305
CREATE INDEX IF NOT EXISTS idx_code_block_changes_commit ON code_block_changes(commit_sha);
CREATE INDEX IF NOT EXISTS idx_code_block_changes_block ON code_block_changes(block_id) WHERE block_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_code_block_changes_repo ON code_block_changes(repo_id);

-- ============================================================================
-- Migrate data from code_block_modifications to code_block_changes
-- ============================================================================

-- Only run migration if code_block_modifications exists and has data
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'code_block_modifications') THEN
        INSERT INTO code_block_changes (
            repo_id, commit_sha, block_id, canonical_file_path, commit_time_path,
            block_type, block_name, change_type, lines_added, lines_deleted,
            change_summary, created_at
        )
        SELECT
            cb.repo_id,
            cbm.commit_sha,
            cbm.code_block_id AS block_id,
            COALESCE(cb.canonical_file_path, cb.file_path) AS canonical_file_path,
            cb.file_path AS commit_time_path,
            cb.block_type,
            cb.block_name,
            CASE
                WHEN cbm.change_type = 'create' THEN 'created'
                WHEN cbm.change_type = 'modify' THEN 'modified'
                WHEN cbm.change_type = 'delete' THEN 'deleted'
                ELSE 'modified'
            END AS change_type,
            COALESCE(cbm.additions, 0) AS lines_added,
            COALESCE(cbm.deletions, 0) AS lines_deleted,
            cbm.raw_llm_output::text AS change_summary,
            cbm.created_at
        FROM code_block_modifications cbm
        JOIN code_blocks cb ON cb.id = cbm.code_block_id
        ON CONFLICT DO NOTHING;

        RAISE NOTICE 'Migrated % rows from code_block_modifications to code_block_changes',
            (SELECT COUNT(*) FROM code_block_changes);
    END IF;
END $$;

-- ============================================================================
-- NOTE: We keep code_block_modifications table for rollback safety
-- Drop it manually after verifying migration: DROP TABLE code_block_modifications CASCADE;
-- ============================================================================
