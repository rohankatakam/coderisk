#!/bin/bash
# Migration Testing Framework
# Tests migrations 010, 011, 012 in an isolated test database
# Author: Agent 1 (Schema Migrations)
# Date: 2025-11-19

set -e  # Exit on first error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Database configuration
export PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
DB_HOST="localhost"
DB_PORT="5433"
DB_USER="coderisk"
TEST_DB="coderisk_migration_test"
PROD_DB="coderisk"

echo "========================================"
echo "Testing Database Migrations (010-012)"
echo "========================================"
echo ""

# ============================================================================
# STEP 1: Setup test database
# ============================================================================

echo -e "${YELLOW}[1/6] Setting up test database...${NC}"
dropdb -h $DB_HOST -p $DB_PORT -U $DB_USER --if-exists $TEST_DB 2>/dev/null || true
createdb -h $DB_HOST -p $DB_PORT -U $DB_USER $TEST_DB

echo "✓ Test database created: $TEST_DB"
echo ""

# ============================================================================
# STEP 2: Copy production schema to test database
# ============================================================================

echo -e "${YELLOW}[2/6] Copying production schema (without data)...${NC}"

# Dump schema from production using docker container
docker exec coderisk-postgres pg_dump -U $DB_USER $PROD_DB --schema-only > /tmp/schema_dump.sql 2>/dev/null || {
    echo -e "${RED}✗ Failed to dump production schema${NC}"
    dropdb -h $DB_HOST -p $DB_PORT -U $DB_USER $TEST_DB
    exit 1
}

# Apply schema to test database
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $TEST_DB -f /tmp/schema_dump.sql > /dev/null 2>&1 || {
    echo -e "${RED}✗ Failed to apply schema to test database${NC}"
    dropdb -h $DB_HOST -p $DB_PORT -U $DB_USER $TEST_DB
    rm /tmp/schema_dump.sql
    exit 1
}

# Cleanup
rm /tmp/schema_dump.sql

echo "✓ Production schema copied to test database"
echo ""

# ============================================================================
# STEP 3: Apply new migrations (010, 011, 012)
# ============================================================================

echo -e "${YELLOW}[3/6] Applying new migrations...${NC}"

# Migration 010: Function Signatures
echo "  [010] Adding function signature support..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $TEST_DB -f migrations/010_add_function_signatures.sql || {
    echo -e "${RED}✗ Migration 010 failed${NC}"
    dropdb -h $DB_HOST -p $DB_PORT -U $DB_USER $TEST_DB
    exit 1
}

# Migration 011: Function Identity Map
echo "  [011] Creating function identity map..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $TEST_DB -f migrations/011_function_identity_map.sql || {
    echo -e "${RED}✗ Migration 011 failed${NC}"
    dropdb -h $DB_HOST -p $DB_PORT -U $DB_USER $TEST_DB
    exit 1
}

# Migration 012: Chunk Metadata
echo "  [012] Adding chunk metadata..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $TEST_DB -f migrations/012_rate_limit_and_chunks.sql || {
    echo -e "${RED}✗ Migration 012 failed${NC}"
    dropdb -h $DB_HOST -p $DB_PORT -U $DB_USER $TEST_DB
    exit 1
}

echo "✓ All migrations applied successfully"
echo ""

# ============================================================================
# STEP 4: Validation Tests for Migration 010
# ============================================================================

echo -e "${YELLOW}[4/6] Testing Migration 010 (Function Signatures)...${NC}"

# Test: Verify signature column exists
SIGNATURE_COL=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $TEST_DB -tAc \
    "SELECT COUNT(*) FROM information_schema.columns WHERE table_name='code_blocks' AND column_name='signature';")

if [ "$SIGNATURE_COL" != "1" ]; then
    echo -e "${RED}✗ Test failed: signature column not found${NC}"
    dropdb -h $DB_HOST -p $DB_PORT -U $DB_USER $TEST_DB
    exit 1
fi

# Test: Verify UNIQUE constraint includes signature
CONSTRAINT_COLS=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $TEST_DB -tAc \
    "SELECT COUNT(*) FROM information_schema.constraint_column_usage WHERE constraint_name='code_blocks_canonical_unique';")

if [ "$CONSTRAINT_COLS" != "4" ]; then
    echo -e "${RED}✗ Test failed: UNIQUE constraint incorrect (expected 4 columns)${NC}"
    dropdb -h $DB_HOST -p $DB_PORT -U $DB_USER $TEST_DB
    exit 1
fi

# Test: Verify function overloading works (insert two functions with same name, different signatures)
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $TEST_DB > /dev/null 2>&1 << 'EOSQL'
-- Create a test repository if it doesn't exist
INSERT INTO github_repositories (id, name, full_name, owner)
VALUES (999999, 'test-repo', 'test/test-repo', 'test')
ON CONFLICT DO NOTHING;

-- Insert two overloaded functions with same name but different signatures
INSERT INTO code_blocks (repo_id, canonical_file_path, block_name, signature, block_type)
VALUES (999999, 'auth.ts', 'login', '(user:string)', 'function');

INSERT INTO code_blocks (repo_id, canonical_file_path, block_name, signature, block_type)
VALUES (999999, 'auth.ts', 'login', '(user:string,pass:string)', 'function');

-- Verify both were inserted
DO $$
DECLARE
    count INTEGER;
BEGIN
    SELECT COUNT(*) INTO count FROM code_blocks
    WHERE repo_id = 999999 AND block_name = 'login';

    IF count != 2 THEN
        RAISE EXCEPTION 'Expected 2 overloaded functions, found %', count;
    END IF;
END $$;
EOSQL

if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Test failed: Function overloading test failed${NC}"
    dropdb -h $DB_HOST -p $DB_PORT -U $DB_USER $TEST_DB
    exit 1
fi

echo "  ✓ Signature column exists"
echo "  ✓ UNIQUE constraint correct (4 columns)"
echo "  ✓ Function overloading works"
echo ""

# ============================================================================
# STEP 5: Validation Tests for Migration 011
# ============================================================================

echo -e "${YELLOW}[5/6] Testing Migration 011 (Function Identity Map)...${NC}"

# Test: Verify function_identity_map table exists
TABLE_EXISTS=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $TEST_DB -tAc \
    "SELECT COUNT(*) FROM information_schema.tables WHERE table_name='function_identity_map';")

if [ "$TABLE_EXISTS" != "1" ]; then
    echo -e "${RED}✗ Test failed: function_identity_map table not found${NC}"
    dropdb -h $DB_HOST -p $DB_PORT -U $DB_USER $TEST_DB
    exit 1
fi

# Test: Verify historical_block_names column exists
HISTORICAL_COL=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $TEST_DB -tAc \
    "SELECT COUNT(*) FROM information_schema.columns WHERE table_name='code_blocks' AND column_name='historical_block_names';")

if [ "$HISTORICAL_COL" != "1" ]; then
    echo -e "${RED}✗ Test failed: historical_block_names column not found${NC}"
    dropdb -h $DB_HOST -p $DB_PORT -U $DB_USER $TEST_DB
    exit 1
fi

# Test: Verify GIN index exists
GIN_INDEX=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $TEST_DB -tAc \
    "SELECT COUNT(*) FROM pg_indexes WHERE indexname='idx_code_blocks_historical_names';")

if [ "$GIN_INDEX" != "1" ]; then
    echo -e "${RED}✗ Test failed: GIN index not found${NC}"
    dropdb -h $DB_HOST -p $DB_PORT -U $DB_USER $TEST_DB
    exit 1
fi

# Test: Insert rename history and query it
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $TEST_DB > /dev/null 2>&1 << 'EOSQL'
-- Get the block_id of one of our test functions
DO $$
DECLARE
    test_block_id BIGINT;
BEGIN
    SELECT id INTO test_block_id FROM code_blocks
    WHERE repo_id = 999999 AND block_name = 'login' LIMIT 1;

    -- Insert rename history
    INSERT INTO function_identity_map (repo_id, block_id, historical_name, signature, commit_sha, rename_date)
    VALUES (999999, test_block_id, 'handleLogin', '(user:string)', 'abc123', NOW());

    INSERT INTO function_identity_map (repo_id, block_id, historical_name, signature, commit_sha, rename_date)
    VALUES (999999, test_block_id, 'processLogin', '(user:string)', 'def456', NOW());

    -- Update historical_block_names JSONB array
    UPDATE code_blocks
    SET historical_block_names = '["handleLogin", "processLogin"]'::JSONB
    WHERE id = test_block_id;

    -- Test JSONB containment query
    IF NOT EXISTS (
        SELECT 1 FROM code_blocks
        WHERE historical_block_names @> '["handleLogin"]'::JSONB
    ) THEN
        RAISE EXCEPTION 'JSONB containment query failed';
    END IF;
END $$;
EOSQL

if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Test failed: Rename history test failed${NC}"
    dropdb -h $DB_HOST -p $DB_PORT -U $DB_USER $TEST_DB
    exit 1
fi

echo "  ✓ function_identity_map table exists"
echo "  ✓ historical_block_names column exists"
echo "  ✓ GIN index exists"
echo "  ✓ Rename history tracking works"
echo "  ✓ JSONB containment queries work"
echo ""

# ============================================================================
# STEP 6: Validation Tests for Migration 012
# ============================================================================

echo -e "${YELLOW}[6/6] Testing Migration 012 (Chunk Metadata)...${NC}"

# Test: Verify all chunk columns exist
CHUNKS_PROCESSED=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $TEST_DB -tAc \
    "SELECT COUNT(*) FROM information_schema.columns WHERE table_name='github_commits' AND column_name='diff_chunks_processed';")

CHUNKS_SKIPPED=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $TEST_DB -tAc \
    "SELECT COUNT(*) FROM information_schema.columns WHERE table_name='github_commits' AND column_name='diff_chunks_skipped';")

TRUNCATION_REASON=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $TEST_DB -tAc \
    "SELECT COUNT(*) FROM information_schema.columns WHERE table_name='github_commits' AND column_name='diff_truncation_reason';")

if [ "$CHUNKS_PROCESSED" != "1" ] || [ "$CHUNKS_SKIPPED" != "1" ] || [ "$TRUNCATION_REASON" != "1" ]; then
    echo -e "${RED}✗ Test failed: Chunk metadata columns not found${NC}"
    dropdb -h $DB_HOST -p $DB_PORT -U $DB_USER $TEST_DB
    exit 1
fi

# Test: Verify index exists
CHUNK_INDEX=$(psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $TEST_DB -tAc \
    "SELECT COUNT(*) FROM pg_indexes WHERE indexname='idx_commits_chunk_tracking';")

if [ "$CHUNK_INDEX" != "1" ]; then
    echo -e "${RED}✗ Test failed: Chunk tracking index not found${NC}"
    dropdb -h $DB_HOST -p $DB_PORT -U $DB_USER $TEST_DB
    exit 1
fi

# Test: Insert and query chunk metadata
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $TEST_DB > /dev/null 2>&1 << 'EOSQL'
-- Create test commit
INSERT INTO github_commits (id, repo_id, sha, message, author_name, author_email, committed_at)
VALUES (999999, 999999, 'test123', 'Test commit', 'Test User', 'test@example.com', NOW())
ON CONFLICT DO NOTHING;

-- Update with chunk metadata
UPDATE github_commits SET
    diff_chunks_processed = 50,
    diff_chunks_skipped = 5,
    diff_truncation_reason = 'Exceeded budget'
WHERE id = 999999;

-- Verify data was stored correctly
DO $$
DECLARE
    processed INTEGER;
    skipped INTEGER;
    reason TEXT;
BEGIN
    SELECT diff_chunks_processed, diff_chunks_skipped, diff_truncation_reason
    INTO processed, skipped, reason
    FROM github_commits WHERE id = 999999;

    IF processed != 50 OR skipped != 5 OR reason != 'Exceeded budget' THEN
        RAISE EXCEPTION 'Chunk metadata not stored correctly';
    END IF;
END $$;
EOSQL

if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Test failed: Chunk metadata test failed${NC}"
    dropdb -h $DB_HOST -p $DB_PORT -U $DB_USER $TEST_DB
    exit 1
fi

echo "  ✓ diff_chunks_processed column exists"
echo "  ✓ diff_chunks_skipped column exists"
echo "  ✓ diff_truncation_reason column exists"
echo "  ✓ Chunk tracking index exists"
echo "  ✓ Chunk metadata storage works"
echo ""

# ============================================================================
# CLEANUP
# ============================================================================

echo -e "${YELLOW}Cleaning up test database...${NC}"
dropdb -h $DB_HOST -p $DB_PORT -U $DB_USER $TEST_DB
echo "✓ Test database dropped"
echo ""

# ============================================================================
# SUCCESS
# ============================================================================

echo "========================================"
echo -e "${GREEN}✓ All migration tests passed!${NC}"
echo "========================================"
echo ""
echo "Summary:"
echo "  • Migration 010: Function signatures ✓"
echo "  • Migration 011: Function identity map ✓"
echo "  • Migration 012: Chunk metadata ✓"
echo ""
echo "Migrations are ready to apply to production."
echo ""

exit 0
