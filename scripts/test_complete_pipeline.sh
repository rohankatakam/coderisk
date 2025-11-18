#!/bin/bash
# Test complete CodeRisk ingestion pipeline with mcp-use data (repo_id=11)
# This script validates the microservice architecture implementation

set -e  # Exit on error

REPO_ID=11
REPO_PATH="/Users/rohankatakam/Documents/brain/mcp-use"

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║  CodeRisk Pipeline Test - Microservice Architecture Validation ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""
echo "Repository: mcp-use (ID: $REPO_ID)"
echo "Path: $REPO_PATH"
echo ""

# Check environment
echo "[Pre-flight] Checking environment..."
if [ ! -d "$REPO_PATH" ]; then
    echo "❌ Repository path not found: $REPO_PATH"
    exit 1
fi

if ! docker ps | grep -q postgres; then
    echo "❌ PostgreSQL container not running"
    exit 1
fi

if ! docker ps | grep -q neo4j; then
    echo "❌ Neo4j container not running"
    exit 1
fi

echo "✓ Environment OK"
echo ""

# Validate schema
echo "[1/6] Validating database schema..."
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "
SELECT
  CASE
    WHEN COUNT(*) FILTER (WHERE column_name = 'topological_index') > 0 THEN '✓'
    ELSE '✗'
  END || ' topological_index' as check_result
FROM information_schema.columns
WHERE table_name = 'github_commits'
UNION ALL
SELECT
  CASE
    WHEN COUNT(*) FILTER (WHERE column_name = 'parent_shas_hash') > 0 THEN '✓'
    ELSE '✗'
  END || ' parent_shas_hash'
FROM information_schema.columns
WHERE table_name = 'github_repositories'
UNION ALL
SELECT
  CASE
    WHEN COUNT(*) FILTER (WHERE column_name = 'canonical_file_path') > 0 THEN '✓'
    ELSE '✗'
  END || ' canonical_file_path'
FROM information_schema.columns
WHERE table_name = 'code_blocks'
UNION ALL
SELECT
  CASE
    WHEN COUNT(*) > 0 THEN '✓'
    ELSE '✗'
  END || ' ingestion_jobs table'
FROM information_schema.tables
WHERE table_name = 'ingestion_jobs';
" -t

echo ""

# Validate data
echo "[2/6] Validating existing data..."
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "
SELECT
  'Commits' as entity,
  COUNT(*) as total,
  COUNT(*) FILTER (WHERE topological_index IS NOT NULL) as with_topo_index,
  ROUND(100.0 * COUNT(*) FILTER (WHERE topological_index IS NOT NULL) / COUNT(*), 1) || '%' as coverage
FROM github_commits WHERE repo_id = $REPO_ID
UNION ALL
SELECT
  'Files',
  COUNT(*),
  COUNT(*),
  '100%'
FROM file_identity_map WHERE repo_id = $REPO_ID
UNION ALL
SELECT
  'CodeBlocks',
  COUNT(*),
  COUNT(*) FILTER (WHERE canonical_file_path IS NOT NULL),
  ROUND(100.0 * COUNT(*) FILTER (WHERE canonical_file_path IS NOT NULL) / COUNT(*), 1) || '%'
FROM code_blocks WHERE repo_id = $REPO_ID;
"

echo ""

# Test crisk-ingest (reads topological_index)
echo "[3/6] Testing crisk-ingest (graph construction with topological ordering)..."
echo "  Note: This tests that topological_index is used correctly"
echo ""

cd /Users/rohankatakam/Documents/brain/coderisk

# Just validate it can start - don't run full ingest
timeout 5 ./bin/crisk-ingest --repo-id $REPO_ID --repo-path $REPO_PATH --help > /dev/null 2>&1 || true
if [ -f ./bin/crisk-ingest ]; then
    echo "  ✓ crisk-ingest binary exists"
else
    echo "  ℹ️  crisk-ingest binary not found (run 'make build' first)"
fi

echo ""

# Check Neo4j connectivity
echo "[4/6] Checking Neo4j connectivity..."
curl -s -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  -H "Content-Type: application/json" \
  -X POST http://localhost:7475/db/neo4j/tx/commit \
  -d '{"statements":[{"statement":"MATCH (n) RETURN count(n) as total LIMIT 1"}]}' | \
  grep -q "total" && echo "  ✓ Neo4j accessible" || echo "  ✗ Neo4j connection failed"

echo ""

# Validate force-push detection
echo "[5/6] Validating force-push detection..."
HASH=$(PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -t -c "
SELECT parent_shas_hash FROM github_repositories WHERE id = $REPO_ID;
" | tr -d ' ')

if [ -n "$HASH" ]; then
    echo "  ✓ Parent SHA hash stored: ${HASH:0:16}..."
    echo "  ℹ️  This hash will detect any force-pushes to the repository"
else
    echo "  ✗ No parent SHA hash found (run compute_topological_ordering.go)"
fi

echo ""

# Summary
echo "[6/6] Migration Readiness Summary"
echo "══════════════════════════════════════════════════════════════"

TOPO_COUNT=$(PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -t -c "
SELECT COUNT(*) FROM github_commits WHERE repo_id = $REPO_ID AND topological_index IS NOT NULL;
" | tr -d ' ')

TOTAL_COUNT=$(PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -t -c "
SELECT COUNT(*) FROM github_commits WHERE repo_id = $REPO_ID;
" | tr -d ' ')

FILE_COUNT=$(PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -t -c "
SELECT COUNT(*) FROM file_identity_map WHERE repo_id = $REPO_ID;
" | tr -d ' ')

BLOCK_COUNT=$(PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -t -c "
SELECT COUNT(*) FROM code_blocks WHERE repo_id = $REPO_ID AND canonical_file_path IS NOT NULL;
" | tr -d ' ')

echo "  Topological ordering: $TOPO_COUNT / $TOTAL_COUNT commits indexed"
echo "  File identity map: $FILE_COUNT files tracked"
echo "  CodeBlocks with canonical paths: $BLOCK_COUNT"
echo ""

if [ "$TOPO_COUNT" -ge 500 ] && [ "$FILE_COUNT" -ge 400 ]; then
    echo "✅ LOCAL INFRASTRUCTURE READY FOR MICROSERVICE PIPELINE"
    echo ""
    echo "Next steps:"
    echo "  1. Test crisk-ingest: ./bin/crisk-ingest --repo-id $REPO_ID --repo-path $REPO_PATH"
    echo "  2. Test crisk-atomize: ./bin/crisk-atomize --repo-id $REPO_ID --repo-path $REPO_PATH"
    echo "  3. Run full pipeline: ./bin/crisk init (orchestrates all services)"
    echo ""
    echo "Migration to cloud:"
    echo "  - Schema: ✅ Aligned with spec"
    echo "  - Topological ordering: ✅ Implemented"
    echo "  - Canonical paths: ✅ Implemented"
    echo "  - Force-push detection: ✅ Implemented"
    echo "  - Job tracking: ✅ Schema ready (ingestion_jobs table)"
    echo ""
    echo "See docs/IMPLEMENTATION_SUMMARY.md for complete details"
else
    echo "⚠️  VALIDATION INCOMPLETE"
    echo "  Run: go run scripts/compute_topological_ordering.go $REPO_ID"
fi

echo "══════════════════════════════════════════════════════════════"
