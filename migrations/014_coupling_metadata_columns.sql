-- Migration 014: Add coupling metadata columns for static vs dynamic separation
-- Reference: microservice_arch.md lines 120-214 - Coupling Static vs Dynamic Strategy
-- Reference: DATA_SCHEMA_REFERENCE.md lines 405-474 - code_block_coupling table specification
-- Author: Claude (Static/Dynamic Architecture Alignment)
-- Date: 2025-11-19

-- ============================================================================
-- Add coupling metadata columns (STATIC timestamps tracking analysis window)
-- ============================================================================

-- Add computed_at to track when coupling analysis was performed
ALTER TABLE code_block_coupling ADD COLUMN IF NOT EXISTS computed_at TIMESTAMP DEFAULT NOW();

-- Add window_start and window_end to track analysis time window
ALTER TABLE code_block_coupling ADD COLUMN IF NOT EXISTS window_start TIMESTAMP;
ALTER TABLE code_block_coupling ADD COLUMN IF NOT EXISTS window_end TIMESTAMP;

-- ============================================================================
-- Rename coupling_rate to co_change_percentage for clarity
-- ============================================================================

-- Rename coupling_rate to co_change_percentage (0.0-1.0 range)
DO $$
BEGIN
    -- Check if coupling_rate column exists
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'code_block_coupling' AND column_name = 'coupling_rate') THEN
        ALTER TABLE code_block_coupling RENAME COLUMN coupling_rate TO co_change_percentage;
    END IF;

    -- Add co_change_percentage if doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'code_block_coupling' AND column_name = 'co_change_percentage') THEN
        ALTER TABLE code_block_coupling ADD COLUMN co_change_percentage FLOAT;
    END IF;
END $$;

-- ============================================================================
-- Remove total_changes column (deprecated - now computed from co_change_count/co_change_percentage)
-- ============================================================================

-- Drop total_changes column if it exists (deprecated per static/dynamic separation)
ALTER TABLE code_block_coupling DROP COLUMN IF EXISTS total_changes;

-- ============================================================================
-- Backfill window_start and window_end from existing data
-- ============================================================================

-- For existing rows, set window_end to the latest co-change date
-- and window_start to 12 months before window_end
UPDATE code_block_coupling
SET
    window_end = last_co_change,
    window_start = last_co_change - INTERVAL '365 days'
WHERE window_end IS NULL
  AND last_co_change IS NOT NULL;

-- For rows without last_co_change, use created_at as fallback
UPDATE code_block_coupling
SET
    window_end = created_at,
    window_start = created_at - INTERVAL '365 days'
WHERE window_end IS NULL
  AND created_at IS NOT NULL;

-- ============================================================================
-- Update indexes to reflect new schema
-- ============================================================================

-- Drop old index on coupling_rate (if exists)
DROP INDEX IF EXISTS idx_code_block_coupling_rate;
DROP INDEX IF EXISTS idx_code_block_coupling_repo_rate;

-- Create index on co_change_percentage for ranked queries
CREATE INDEX IF NOT EXISTS idx_code_block_coupling_percentage
ON code_block_coupling(repo_id, co_change_percentage DESC NULLS LAST)
WHERE co_change_percentage IS NOT NULL;

-- Create index on block_a_id for fast lookups
CREATE INDEX IF NOT EXISTS idx_code_block_coupling_block_a
ON code_block_coupling(block_a_id);

-- ============================================================================
-- Add column comments documenting static vs dynamic separation
-- ============================================================================

COMMENT ON COLUMN code_block_coupling.co_change_count IS
'STATIC: Historical count of times both blocks changed together within the analysis window. Never changes after computation.';

COMMENT ON COLUMN code_block_coupling.co_change_percentage IS
'STATIC: Historical co-change rate (0.0-1.0) within the analysis window. Never changes after computation.';

COMMENT ON COLUMN code_block_coupling.computed_at IS
'STATIC: Timestamp when this coupling analysis was performed. Enables tracking staleness of coupling data.';

COMMENT ON COLUMN code_block_coupling.window_start IS
'STATIC: Start of analysis time window (e.g., 12 months ago from window_end). Defines historical period analyzed.';

COMMENT ON COLUMN code_block_coupling.window_end IS
'STATIC: End of analysis time window (e.g., date of analysis run). Defines historical period analyzed.';

COMMENT ON TABLE code_block_coupling IS
'Stores STATIC co-change relationships between code blocks. DYNAMIC properties (incident weights, recency multipliers, combined coupling scores, Top-K rankings) are computed at query time by MCP server. See microservice_arch.md section 6 for architectural rationale.';

-- ============================================================================
-- Migration complete
-- ============================================================================

DO $$
DECLARE
    total_couplings INTEGER;
    with_metadata INTEGER;
    metadata_percentage NUMERIC;
BEGIN
    -- Count total coupling edges
    SELECT COUNT(*) INTO total_couplings FROM code_block_coupling;

    -- Count edges with complete metadata
    SELECT COUNT(*) INTO with_metadata
    FROM code_block_coupling
    WHERE computed_at IS NOT NULL
      AND window_start IS NOT NULL
      AND window_end IS NOT NULL;

    -- Calculate metadata completeness percentage
    IF total_couplings > 0 THEN
        metadata_percentage := (with_metadata::NUMERIC / total_couplings::NUMERIC) * 100;
    ELSE
        metadata_percentage := 0;
    END IF;

    RAISE NOTICE 'Migration 014 complete: Added coupling metadata columns';
    RAISE NOTICE '  Total coupling edges: %', total_couplings;
    RAISE NOTICE '  Edges with metadata: % (%.1f%%)', with_metadata, metadata_percentage;

    IF metadata_percentage < 95 AND total_couplings > 0 THEN
        RAISE WARNING 'Metadata coverage below 95%% (%.1f%%). Some edges may have NULL metadata columns.', metadata_percentage;
    END IF;
END $$;
