-- Migration 013: Add last_modified_date column for static vs dynamic separation
-- Reference: microservice_arch.md lines 105-147 - Ownership Static vs Dynamic Strategy
-- Reference: DATA_SCHEMA_REFERENCE.md lines 303-307 - last_modified_date column specification
-- Author: Claude (Static/Dynamic Architecture Alignment)
-- Date: 2025-11-19

-- ============================================================================
-- Add last_modified_date column (STATIC timestamp)
-- ============================================================================

-- Add last_modified_date column to store static modification timestamp
ALTER TABLE code_blocks ADD COLUMN IF NOT EXISTS last_modified_date TIMESTAMP;

-- ============================================================================
-- Backfill last_modified_date from existing data
-- ============================================================================

-- For blocks that have staleness_days populated, backfill last_modified_date
-- by calculating: NOW() - (staleness_days * INTERVAL '1 day')
UPDATE code_blocks
SET last_modified_date = NOW() - (staleness_days || ' days')::INTERVAL
WHERE last_modified_date IS NULL
  AND staleness_days IS NOT NULL;

-- For blocks without staleness_days, use last modification from code_block_changes
-- This query finds the most recent commit that modified each block
UPDATE code_blocks cb
SET last_modified_date = subq.last_modified
FROM (
    SELECT DISTINCT ON (cbc.block_id)
        cbc.block_id,
        c.author_date AS last_modified
    FROM code_block_changes cbc
    JOIN github_commits c ON c.sha = cbc.commit_sha AND c.repo_id = cbc.repo_id
    WHERE cbc.repo_id = cb.repo_id
    ORDER BY cbc.block_id, c.author_date DESC
) subq
WHERE cb.id = subq.block_id
  AND cb.last_modified_date IS NULL;

-- For blocks still without last_modified_date, use first_seen_sha as fallback
UPDATE code_blocks cb
SET last_modified_date = gc.author_date
FROM github_commits gc
WHERE gc.sha = cb.first_seen_sha
  AND cb.last_modified_date IS NULL
  AND gc.author_date IS NOT NULL;

-- ============================================================================
-- Create indexes for query performance
-- ============================================================================

-- Index for staleness queries (query-time calculation)
CREATE INDEX IF NOT EXISTS idx_code_blocks_last_modified
ON code_blocks(repo_id, last_modified_date DESC NULLS LAST)
WHERE last_modified_date IS NOT NULL;

-- ============================================================================
-- Deprecate staleness_days column (add comment)
-- ============================================================================

COMMENT ON COLUMN code_blocks.staleness_days IS
'DEPRECATED: This column should NOT be used. Compute staleness dynamically at query time using: EXTRACT(DAY FROM (NOW() - last_modified_date))::INTEGER. Rationale: Staleness changes daily, violating static/dynamic separation principle. See microservice_arch.md section 5 for architectural rationale.';

-- ============================================================================
-- Migration complete
-- ============================================================================

DO $$
DECLARE
    total_blocks INTEGER;
    with_last_modified INTEGER;
    backfill_percentage NUMERIC;
BEGIN
    -- Count total blocks
    SELECT COUNT(*) INTO total_blocks FROM code_blocks;

    -- Count blocks with last_modified_date
    SELECT COUNT(*) INTO with_last_modified
    FROM code_blocks
    WHERE last_modified_date IS NOT NULL;

    -- Calculate backfill percentage
    IF total_blocks > 0 THEN
        backfill_percentage := (with_last_modified::NUMERIC / total_blocks::NUMERIC) * 100;
    ELSE
        backfill_percentage := 0;
    END IF;

    RAISE NOTICE 'Migration 013 complete: Added last_modified_date column';
    RAISE NOTICE '  Total blocks: %', total_blocks;
    RAISE NOTICE '  Blocks with last_modified_date: % (%.1f%%)', with_last_modified, backfill_percentage;

    IF backfill_percentage < 95 THEN
        RAISE WARNING 'Backfill coverage below 95%% (%.1f%%). Some blocks may have NULL last_modified_date.', backfill_percentage;
    END IF;
END $$;
