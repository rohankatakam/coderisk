-- Migration 006: Create processing checkpoint views
-- Aligns with ingestion_aws.md documentation
-- Author: Claude (Schema Alignment Migration)
-- Date: 2025-11-18

-- ============================================================================
-- Create views for incremental processing
-- ============================================================================

CREATE OR REPLACE VIEW v_unprocessed_commits AS
  SELECT * FROM github_commits WHERE processed_at IS NULL;

CREATE OR REPLACE VIEW v_unprocessed_issues AS
  SELECT * FROM github_issues WHERE processed_at IS NULL;

CREATE OR REPLACE VIEW v_unprocessed_prs AS
  SELECT * FROM github_pull_requests WHERE processed_at IS NULL;

-- ============================================================================
-- Migration complete
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE 'Migration 006 complete: Created processing checkpoint views';
END $$;
