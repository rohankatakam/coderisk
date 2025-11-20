-- Migration 007: Add missing UNIQUE constraints for ON CONFLICT clauses
-- Fixes "no unique or exclusion constraint matching" errors
-- Author: Claude (Schema Alignment Migration)
-- Date: 2025-11-18

-- ============================================================================
-- Add UNIQUE constraint to github_issue_commit_refs
-- Required by temporal correlation ON CONFLICT clause (staging.go:750)
-- ============================================================================

CREATE UNIQUE INDEX IF NOT EXISTS idx_issue_commit_refs_unique
ON github_issue_commit_refs(repo_id, issue_number, COALESCE(commit_sha, ''), COALESCE(pr_number, 0), COALESCE(detection_method, ''));

-- ============================================================================
-- Add UNIQUE constraint to github_pull_request_files
-- Required by PR file staging ON CONFLICT clause
-- ============================================================================

-- First check if this table exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'github_pull_request_files') THEN
        CREATE UNIQUE INDEX IF NOT EXISTS idx_pr_files_unique
        ON github_pull_request_files(pr_id, file_path);
    END IF;
END $$;

-- ============================================================================
-- Migration complete
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE 'Migration 007 complete: Added UNIQUE constraints for ON CONFLICT';
END $$;
