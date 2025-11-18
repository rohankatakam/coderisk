-- Migration 002: Align existing table schemas with microservice spec
-- Fixes column name mismatches between existing schema and DATA_SCHEMA_REFERENCE.md

-- ========================================
-- PART 1: github_issue_timeline - Add missing columns
-- ========================================

-- Add repo_id if missing (for consistency with spec)
ALTER TABLE github_issue_timeline
ADD COLUMN IF NOT EXISTS repo_id BIGINT REFERENCES github_repositories(id) ON DELETE CASCADE;

-- Populate repo_id from issues table
UPDATE github_issue_timeline t
SET repo_id = i.repo_id
FROM github_issues i
WHERE t.issue_id = i.id AND t.repo_id IS NULL;

-- Add commit_sha column (standardize name from source_sha)
ALTER TABLE github_issue_timeline
ADD COLUMN IF NOT EXISTS commit_sha TEXT;

-- Copy data from source_sha to commit_sha
UPDATE github_issue_timeline
SET commit_sha = source_sha
WHERE commit_sha IS NULL AND source_sha IS NOT NULL;

-- Add actor column (standardize name from actor_login)
ALTER TABLE github_issue_timeline
ADD COLUMN IF NOT EXISTS actor TEXT;

UPDATE github_issue_timeline
SET actor = actor_login
WHERE actor IS NULL AND actor_login IS NOT NULL;

-- Add source_issue_number and source_pr_number
ALTER TABLE github_issue_timeline
ADD COLUMN IF NOT EXISTS source_issue_number INTEGER,
ADD COLUMN IF NOT EXISTS source_pr_number INTEGER;

-- Create index on commit_sha (spec requirement)
CREATE INDEX IF NOT EXISTS idx_timeline_commit
ON github_issue_timeline(repo_id, commit_sha)
WHERE commit_sha IS NOT NULL;

-- ========================================
-- PART 2: code_block_incidents - Align column names
-- ========================================

-- Add columns matching spec (if different from existing)
ALTER TABLE code_block_incidents
ADD COLUMN IF NOT EXISTS block_id BIGINT REFERENCES code_blocks(id) ON DELETE CASCADE,
ADD COLUMN IF NOT EXISTS commit_sha TEXT,
ADD COLUMN IF NOT EXISTS incident_date TIMESTAMP WITHOUT TIME ZONE,
ADD COLUMN IF NOT EXISTS resolution_date TIMESTAMP WITHOUT TIME ZONE,
ADD COLUMN IF NOT EXISTS incident_type TEXT;

-- Migrate data from existing columns
UPDATE code_block_incidents
SET block_id = code_block_id
WHERE block_id IS NULL;

UPDATE code_block_incidents
SET commit_sha = fix_commit_sha
WHERE commit_sha IS NULL AND fix_commit_sha IS NOT NULL;

UPDATE code_block_incidents
SET resolution_date = fixed_at
WHERE resolution_date IS NULL AND fixed_at IS NOT NULL;

-- Create index on block_id, incident_date (spec requirement)
CREATE INDEX IF NOT EXISTS idx_block_incidents_block_date
ON code_block_incidents(block_id, incident_date DESC)
WHERE block_id IS NOT NULL;

-- ========================================
-- PART 3: code_block_coupling - Align column names
-- ========================================

-- Add columns matching spec
ALTER TABLE code_block_coupling
ADD COLUMN IF NOT EXISTS repo_id BIGINT REFERENCES github_repositories(id) ON DELETE CASCADE,
ADD COLUMN IF NOT EXISTS coupling_rate FLOAT,
ADD COLUMN IF NOT EXISTS total_changes INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS first_co_change TIMESTAMP WITHOUT TIME ZONE,
ADD COLUMN IF NOT EXISTS last_co_change TIMESTAMP WITHOUT TIME ZONE;

-- Migrate data from existing columns
UPDATE code_block_coupling
SET coupling_rate = co_change_rate::FLOAT
WHERE coupling_rate IS NULL AND co_change_rate IS NOT NULL;

UPDATE code_block_coupling
SET last_co_change = last_co_changed_at
WHERE last_co_change IS NULL AND last_co_changed_at IS NOT NULL;

-- Populate repo_id from code_blocks
UPDATE code_block_coupling cc
SET repo_id = cb.repo_id
FROM code_blocks cb
WHERE cc.block_a_id = cb.id AND cc.repo_id IS NULL;

-- Create indexes matching spec
CREATE INDEX IF NOT EXISTS idx_block_coupling_a_rate
ON code_block_coupling(block_a_id, coupling_rate DESC)
WHERE coupling_rate IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_block_coupling_repo_rate
ON code_block_coupling(repo_id, coupling_rate DESC)
WHERE repo_id IS NOT NULL;

-- ========================================
-- PART 4: Verify critical data integrity
-- ========================================

DO $$
DECLARE
    orphaned_timeline INTEGER;
    orphaned_incidents INTEGER;
    orphaned_coupling INTEGER;
BEGIN
    -- Check for orphaned records
    SELECT COUNT(*) INTO orphaned_timeline
    FROM github_issue_timeline
    WHERE repo_id IS NULL;

    SELECT COUNT(*) INTO orphaned_incidents
    FROM code_block_incidents
    WHERE block_id IS NULL;

    SELECT COUNT(*) INTO orphaned_coupling
    FROM code_block_coupling
    WHERE repo_id IS NULL;

    IF orphaned_timeline > 0 THEN
        RAISE WARNING 'Found % timeline records without repo_id', orphaned_timeline;
    END IF;

    IF orphaned_incidents > 0 THEN
        RAISE WARNING 'Found % incident records without block_id', orphaned_incidents;
    END IF;

    IF orphaned_coupling > 0 THEN
        RAISE WARNING 'Found % coupling records without repo_id', orphaned_coupling;
    END IF;

    RAISE NOTICE '✅ Migration 002 completed. Schema aligned with microservice spec.';
    RAISE NOTICE '   Timeline records: %', (SELECT COUNT(*) FROM github_issue_timeline);
    RAISE NOTICE '   Incident records: %', (SELECT COUNT(*) FROM code_block_incidents);
    RAISE NOTICE '   Coupling records: %', (SELECT COUNT(*) FROM code_block_coupling);
END $$;
