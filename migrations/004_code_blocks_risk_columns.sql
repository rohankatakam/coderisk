-- Migration 004: Add missing risk signal columns to code_blocks
-- Aligns with DATA_SCHEMA_REFERENCE.md lines 238-278
-- Author: Claude (Schema Alignment Migration)
-- Date: 2025-11-18

-- ============================================================================
-- Add missing structural columns
-- ============================================================================

ALTER TABLE code_blocks ADD COLUMN IF NOT EXISTS canonical_file_path TEXT;
ALTER TABLE code_blocks ADD COLUMN IF NOT EXISTS path_at_creation TEXT;
ALTER TABLE code_blocks ADD COLUMN IF NOT EXISTS signature TEXT;
ALTER TABLE code_blocks ADD COLUMN IF NOT EXISTS complexity_estimate INTEGER;

-- Rename columns to match spec (if old names exist)
DO $$
BEGIN
    -- Rename first_seen_commit_sha to first_seen_commit (if exists)
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'code_blocks' AND column_name = 'first_seen_commit_sha') THEN
        ALTER TABLE code_blocks RENAME COLUMN first_seen_commit_sha TO first_seen_commit;
    END IF;

    -- Rename last_modified_commit_sha to last_modified_commit (if exists)
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_name = 'code_blocks' AND column_name = 'last_modified_commit_sha') THEN
        ALTER TABLE code_blocks RENAME COLUMN last_modified_commit_sha TO last_modified_commit;
    END IF;

    -- Add first_seen_commit if doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'code_blocks' AND column_name = 'first_seen_commit') THEN
        ALTER TABLE code_blocks ADD COLUMN first_seen_commit TEXT;
    END IF;

    -- Add last_modified_commit if doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'code_blocks' AND column_name = 'last_modified_commit') THEN
        ALTER TABLE code_blocks ADD COLUMN last_modified_commit TEXT;
    END IF;
END $$;

-- ============================================================================
-- Add risk signal columns (DATA_SCHEMA_REFERENCE.md lines 259-268)
-- ============================================================================

-- Temporal risk signals (populated by crisk-index-incident)
ALTER TABLE code_blocks ADD COLUMN IF NOT EXISTS incident_count INTEGER DEFAULT 0;
ALTER TABLE code_blocks ADD COLUMN IF NOT EXISTS last_incident_date TIMESTAMP;
ALTER TABLE code_blocks ADD COLUMN IF NOT EXISTS temporal_summary TEXT;

-- Ownership risk signals (populated by crisk-index-ownership)
ALTER TABLE code_blocks ADD COLUMN IF NOT EXISTS original_author_email TEXT;
ALTER TABLE code_blocks ADD COLUMN IF NOT EXISTS last_modifier_email TEXT;
ALTER TABLE code_blocks ADD COLUMN IF NOT EXISTS staleness_days INTEGER;
ALTER TABLE code_blocks ADD COLUMN IF NOT EXISTS familiarity_map JSONB;

-- Coupling risk signals (populated by crisk-index-coupling)
ALTER TABLE code_blocks ADD COLUMN IF NOT EXISTS co_change_count INTEGER DEFAULT 0;
ALTER TABLE code_blocks ADD COLUMN IF NOT EXISTS avg_coupling_rate FLOAT;

-- Final composite risk score (populated by crisk-index-coupling)
ALTER TABLE code_blocks ADD COLUMN IF NOT EXISTS risk_score FLOAT;

-- ============================================================================
-- Update UNIQUE constraint to match spec (line 273)
-- Exclude start_line from UNIQUE to handle line shifts during refactoring
-- ============================================================================

-- Drop old UNIQUE constraint if it includes start_line
DO $$
BEGIN
    -- Find and drop any UNIQUE constraints that include start_line
    EXECUTE (
        SELECT 'ALTER TABLE code_blocks DROP CONSTRAINT ' || conname || ';'
        FROM pg_constraint
        WHERE conrelid = 'code_blocks'::regclass
          AND contype = 'u'
          AND array_position(
              ARRAY(SELECT attname FROM pg_attribute
                    WHERE attrelid = conrelid AND attnum = ANY(conkey)),
              'start_line'
          ) IS NOT NULL
        LIMIT 1
    );
EXCEPTION
    WHEN OTHERS THEN
        -- No constraint to drop or error occurred
        NULL;
END $$;

-- Create correct UNIQUE constraint per spec (excludes start_line)
CREATE UNIQUE INDEX IF NOT EXISTS idx_code_blocks_identity
ON code_blocks(repo_id, canonical_file_path, block_name);

-- ============================================================================
-- Create indexes per DATA_SCHEMA_REFERENCE.md lines 274-276
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_code_blocks_lookup
ON code_blocks(repo_id, canonical_file_path, block_name);

CREATE INDEX IF NOT EXISTS idx_code_blocks_hotspots
ON code_blocks(repo_id, incident_count DESC NULLS LAST)
WHERE incident_count > 0;

CREATE INDEX IF NOT EXISTS idx_code_blocks_risk
ON code_blocks(repo_id, risk_score DESC NULLS LAST)
WHERE risk_score IS NOT NULL;

-- ============================================================================
-- Populate canonical_file_path from file_path (backfill for existing data)
-- ============================================================================

UPDATE code_blocks
SET canonical_file_path = file_path
WHERE canonical_file_path IS NULL AND file_path IS NOT NULL;

UPDATE code_blocks
SET path_at_creation = file_path
WHERE path_at_creation IS NULL AND file_path IS NOT NULL;

-- ============================================================================
-- Migration complete
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE 'Migration 004 complete: Added % risk signal columns to code_blocks',
        (SELECT COUNT(*) FROM information_schema.columns
         WHERE table_name = 'code_blocks'
         AND column_name IN ('incident_count', 'original_author_email', 'risk_score'));
END $$;
