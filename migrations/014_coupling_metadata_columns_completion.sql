-- Migration 014 Completion: Fix remaining code_block_coupling schema issues
-- This script completes the schema migration after migration 014 was partially applied
-- Reference: DATA_SCHEMA_REFERENCE.md lines 407-420 - code_block_coupling table specification
-- Author: Claude (Static/Dynamic Architecture Alignment)
-- Date: 2025-11-19

-- ============================================================================
-- Fix column names to match spec
-- ============================================================================

-- Rename last_co_changed_at → last_co_change
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'code_block_coupling' AND column_name = 'last_co_changed_at') THEN
        ALTER TABLE code_block_coupling RENAME COLUMN last_co_changed_at TO last_co_change;
        RAISE NOTICE 'Renamed last_co_changed_at → last_co_change';
    END IF;
END $$;

-- Add first_co_change column
ALTER TABLE code_block_coupling ADD COLUMN IF NOT EXISTS first_co_change TIMESTAMP;

-- Drop deprecated co_change_rate column (replaced by co_change_percentage)
ALTER TABLE code_block_coupling DROP COLUMN IF EXISTS co_change_rate;

-- Drop deprecated 'reason' column (not in spec)
ALTER TABLE code_block_coupling DROP COLUMN IF EXISTS reason;

-- ============================================================================
-- Add block_a_id index for fast lookups
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_code_block_coupling_block_a
ON code_block_coupling(block_a_id);

-- ============================================================================
-- Verify schema matches spec
-- ============================================================================

DO $$
DECLARE
    col_count INTEGER;
BEGIN
    -- Count required columns
    SELECT COUNT(*) INTO col_count
    FROM information_schema.columns
    WHERE table_name = 'code_block_coupling'
      AND column_name IN (
        'id', 'repo_id', 'block_a_id', 'block_b_id',
        'co_change_count', 'co_change_percentage',
        'first_co_change', 'last_co_change',
        'computed_at', 'window_start', 'window_end', 'created_at'
      );

    IF col_count = 12 THEN
        RAISE NOTICE 'Migration 014 completion: Schema validated - all 12 required columns present';
    ELSE
        RAISE WARNING 'Migration 014 completion: Schema validation failed - only % of 12 columns present', col_count;
    END IF;
END $$;
