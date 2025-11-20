-- Migration 010: Add Function Signatures
-- Enables function overloading detection by adding signature column to code_blocks
-- Author: Agent 1 (Schema Migrations)
-- Date: 2025-11-19

-- ============================================================================
-- PART 1: Add signature column to code_blocks
-- ============================================================================

-- Add signature column (nullable for backwards compatibility)
ALTER TABLE code_blocks
ADD COLUMN IF NOT EXISTS signature TEXT;

COMMENT ON COLUMN code_blocks.signature IS 'Normalized function signature (e.g., "(user:string,pass:string)"). Enables detection of function overloading and more precise identity tracking.';

-- ============================================================================
-- PART 2: Backfill existing rows to prevent constraint violations
-- ============================================================================

-- Set signature to empty string for all existing rows
-- This prevents constraint violations during migration
UPDATE code_blocks
SET signature = ''
WHERE signature IS NULL;

-- ============================================================================
-- PART 3: Update UNIQUE constraint to include signature
-- ============================================================================

DO $$
BEGIN
    -- Drop old constraint if exists
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'code_blocks_canonical_unique'
    ) THEN
        ALTER TABLE code_blocks DROP CONSTRAINT code_blocks_canonical_unique;
        RAISE NOTICE 'Dropped old constraint: code_blocks_canonical_unique';
    END IF;

    -- Add new constraint with signature
    -- This allows multiple functions with same name but different signatures
    ALTER TABLE code_blocks
    ADD CONSTRAINT code_blocks_canonical_unique
    UNIQUE (repo_id, canonical_file_path, block_name, signature);

    RAISE NOTICE 'Created new constraint: UNIQUE(repo_id, canonical_file_path, block_name, signature)';

EXCEPTION
    WHEN duplicate_table THEN
        RAISE NOTICE 'Constraint already exists, skipping';
    WHEN duplicate_object THEN
        RAISE NOTICE 'Constraint already exists, skipping';
END $$;

-- ============================================================================
-- PART 4: Add performance index
-- ============================================================================

-- Create B-tree index for fast lookups
CREATE INDEX IF NOT EXISTS idx_code_blocks_signature
ON code_blocks(repo_id, canonical_file_path, block_name, signature);

COMMENT ON INDEX idx_code_blocks_signature IS 'Performance index for function signature lookups and uniqueness checks';

-- ============================================================================
-- VALIDATION: Verify migration success
-- ============================================================================

DO $$
DECLARE
    signature_column_exists BOOLEAN;
    constraint_exists BOOLEAN;
    index_exists BOOLEAN;
BEGIN
    -- Check signature column exists
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'code_blocks' AND column_name = 'signature'
    ) INTO signature_column_exists;

    -- Check constraint exists
    SELECT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'code_blocks_canonical_unique'
    ) INTO constraint_exists;

    -- Check index exists
    SELECT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'idx_code_blocks_signature'
    ) INTO index_exists;

    IF NOT signature_column_exists THEN
        RAISE EXCEPTION 'Migration 010 failed: signature column not created';
    END IF;

    IF NOT constraint_exists THEN
        RAISE EXCEPTION 'Migration 010 failed: unique constraint not created';
    END IF;

    IF NOT index_exists THEN
        RAISE EXCEPTION 'Migration 010 failed: performance index not created';
    END IF;

    RAISE NOTICE '✅ Migration 010 completed successfully:';
    RAISE NOTICE '   - signature column added to code_blocks';
    RAISE NOTICE '   - UNIQUE constraint updated to include signature';
    RAISE NOTICE '   - Performance index created';
    RAISE NOTICE '   - Function overloading now supported';
END $$;
