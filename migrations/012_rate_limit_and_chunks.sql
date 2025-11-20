-- Migration 012: Rate Limit and Chunk Metadata
-- Adds chunk processing tracking to github_commits for better observability
-- Author: Agent 1 (Schema Migrations)
-- Date: 2025-11-19

-- ============================================================================
-- PART 1: Add chunk processing columns to github_commits
-- ============================================================================

-- Add columns to track diff chunk processing
ALTER TABLE github_commits
ADD COLUMN IF NOT EXISTS diff_chunks_processed INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS diff_chunks_skipped INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS diff_truncation_reason TEXT;

COMMENT ON COLUMN github_commits.diff_chunks_processed IS 'Number of diff chunks successfully processed by LLM (for rate limit tracking and observability)';
COMMENT ON COLUMN github_commits.diff_chunks_skipped IS 'Number of diff chunks skipped due to budget/rate limits';
COMMENT ON COLUMN github_commits.diff_truncation_reason IS 'Reason for truncating diff processing (e.g., "Exceeded budget", "Rate limit hit"). NULL if no truncation occurred.';

-- ============================================================================
-- PART 2: Create index for chunk tracking queries
-- ============================================================================

-- Index for finding commits with skipped chunks (monitoring queries)
CREATE INDEX IF NOT EXISTS idx_commits_chunk_tracking
ON github_commits(diff_chunks_processed, diff_chunks_skipped)
WHERE diff_chunks_skipped > 0;

COMMENT ON INDEX idx_commits_chunk_tracking IS 'Performance index for monitoring queries: find commits with skipped chunks due to rate limiting';

-- ============================================================================
-- VALIDATION: Verify migration success
-- ============================================================================

DO $$
DECLARE
    chunks_processed_exists BOOLEAN;
    chunks_skipped_exists BOOLEAN;
    truncation_reason_exists BOOLEAN;
    index_exists BOOLEAN;
BEGIN
    -- Check all columns exist
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'github_commits' AND column_name = 'diff_chunks_processed'
    ) INTO chunks_processed_exists;

    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'github_commits' AND column_name = 'diff_chunks_skipped'
    ) INTO chunks_skipped_exists;

    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'github_commits' AND column_name = 'diff_truncation_reason'
    ) INTO truncation_reason_exists;

    -- Check index exists
    SELECT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'idx_commits_chunk_tracking'
    ) INTO index_exists;

    -- Validate all components exist
    IF NOT chunks_processed_exists THEN
        RAISE EXCEPTION 'Migration 012 failed: diff_chunks_processed column not created';
    END IF;

    IF NOT chunks_skipped_exists THEN
        RAISE EXCEPTION 'Migration 012 failed: diff_chunks_skipped column not created';
    END IF;

    IF NOT truncation_reason_exists THEN
        RAISE EXCEPTION 'Migration 012 failed: diff_truncation_reason column not created';
    END IF;

    IF NOT index_exists THEN
        RAISE EXCEPTION 'Migration 012 failed: chunk tracking index not created';
    END IF;

    RAISE NOTICE '✅ Migration 012 completed successfully:';
    RAISE NOTICE '   - diff_chunks_processed column added';
    RAISE NOTICE '   - diff_chunks_skipped column added';
    RAISE NOTICE '   - diff_truncation_reason column added';
    RAISE NOTICE '   - Chunk tracking index created';
    RAISE NOTICE '   - Rate limit observability now supported';
END $$;
