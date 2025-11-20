-- Migration 005: Create code_block_imports table
-- Aligns with DATA_SCHEMA_REFERENCE.md lines 309-325
-- Author: Claude (Schema Alignment Migration)
-- Date: 2025-11-18

-- ============================================================================
-- Create code_block_imports table
-- ============================================================================

CREATE TABLE IF NOT EXISTS code_block_imports (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    source_block_id BIGINT NOT NULL REFERENCES code_blocks(id) ON DELETE CASCADE,
    target_module TEXT NOT NULL,
    target_symbol TEXT,
    import_type TEXT CHECK (import_type IN ('internal', 'external')),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes per DATA_SCHEMA_REFERENCE.md lines 323-324
CREATE INDEX IF NOT EXISTS idx_code_block_imports_source
ON code_block_imports(source_block_id);

CREATE INDEX IF NOT EXISTS idx_code_block_imports_target
ON code_block_imports(repo_id, target_module);

-- ============================================================================
-- Migration complete
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE 'Migration 005 complete: Created code_block_imports table';
END $$;
