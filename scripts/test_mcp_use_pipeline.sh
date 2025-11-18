#!/bin/bash
# Test script for mcp-use pipeline (repo_id=11)
# This script runs the indexers and validates the complete pipeline

set -e  # Exit on error

REPO_ID=11
REPO_PATH="/Users/rohankatakam/Documents/brain/mcp-use"
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
PG_HOST="localhost"
PG_PORT="5433"
PG_USER="coderisk"
PG_DB="coderisk"

echo "═══════════════════════════════════════════════════════════════"
echo "  CodeRisk Pipeline Test - mcp-use (repo_id=$REPO_ID)"
echo "═══════════════════════════════════════════════════════════════"
echo ""

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check prerequisites
echo "[1/6] Checking prerequisites..."
echo ""

# Check databases
if ! docker ps | grep -q coderisk-postgres; then
    echo -e "${RED}✗ PostgreSQL not running${NC}"
    echo "  Start with: docker start coderisk-postgres"
    exit 1
fi
echo -e "${GREEN}✓ PostgreSQL running${NC}"

if ! docker ps | grep -q coderisk-neo4j; then
    echo -e "${RED}✗ Neo4j not running${NC}"
    echo "  Start with: docker start coderisk-neo4j"
    exit 1
fi
echo -e "${GREEN}✓ Neo4j running${NC}"

# Check binaries
if [ ! -f "./bin/crisk-index-incident" ]; then
    echo -e "${RED}✗ Binaries not built${NC}"
    echo "  Build with: make build"
    exit 1
fi
echo -e "${GREEN}✓ Binaries built${NC}"

# Check repo exists
if [ ! -d "$REPO_PATH" ]; then
    echo -e "${RED}✗ Repository not found: $REPO_PATH${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Repository exists${NC}"
echo ""

# Show current data status
echo "[2/6] Current data status..."
echo ""

PGPASSWORD=$PGPASSWORD psql -h $PG_HOST -p $PG_PORT -U $PG_USER -d $PG_DB -c "
SELECT
    'Commits' as entity,
    COUNT(*) as postgres_count,
    COUNT(*) FILTER (WHERE topological_index IS NOT NULL) as with_topo
FROM github_commits WHERE repo_id = $REPO_ID
UNION ALL
SELECT
    'Files',
    COUNT(*),
    COUNT(*) FILTER (WHERE historical_paths IS NOT NULL)
FROM file_identity_map WHERE repo_id = $REPO_ID
UNION ALL
SELECT
    'CodeBlocks',
    COUNT(*),
    COUNT(*) FILTER (WHERE canonical_file_path IS NOT NULL)
FROM code_blocks WHERE repo_id = $REPO_ID
UNION ALL
SELECT
    'With Incidents',
    COUNT(*) FILTER (WHERE incident_count > 0),
    COUNT(*) FILTER (WHERE incident_count IS NOT NULL)
FROM code_blocks WHERE repo_id = $REPO_ID
UNION ALL
SELECT
    'With Risk Scores',
    COUNT(*) FILTER (WHERE risk_score IS NOT NULL),
    COUNT(*)
FROM code_blocks WHERE repo_id = $REPO_ID;
" 2>/dev/null || echo -e "${YELLOW}⚠ Could not query database${NC}"

echo ""

# Ask user if they want to run indexers
echo -e "${YELLOW}[3/6] Ready to run indexers?${NC}"
echo ""
echo "This will populate:"
echo "  • Incident counts and temporal summaries"
echo "  • Ownership metrics (staleness, familiarity)"
echo "  • Coupling analysis and risk scores"
echo ""
read -p "Continue? (y/N) " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Test cancelled."
    exit 0
fi

# Run indexers
echo ""
echo "[4/6] Running indexers..."
echo ""

echo "  Running incident indexer..."
if ./bin/crisk-index-incident --repo-id $REPO_ID 2>&1 | tee /tmp/crisk-incident.log; then
    echo -e "${GREEN}  ✓ Incident indexer completed${NC}"
else
    echo -e "${RED}  ✗ Incident indexer failed${NC}"
    echo "  See /tmp/crisk-incident.log for details"
    exit 1
fi

echo ""
echo "  Running ownership indexer..."
if ./bin/crisk-index-ownership --repo-id $REPO_ID 2>&1 | tee /tmp/crisk-ownership.log; then
    echo -e "${GREEN}  ✓ Ownership indexer completed${NC}"
else
    echo -e "${RED}  ✗ Ownership indexer failed${NC}"
    echo "  See /tmp/crisk-ownership.log for details"
    exit 1
fi

echo ""
echo "  Running coupling indexer..."
if ./bin/crisk-index-coupling --repo-id $REPO_ID 2>&1 | tee /tmp/crisk-coupling.log; then
    echo -e "${GREEN}  ✓ Coupling indexer completed${NC}"
else
    echo -e "${RED}  ✗ Coupling indexer failed${NC}"
    echo "  See /tmp/crisk-coupling.log for details"
    exit 1
fi

echo ""

# Validate consistency
echo "[5/6] Validating Postgres ↔ Neo4j consistency..."
echo ""

if ./bin/crisk-sync --repo-id $REPO_ID --mode validate-only 2>&1 | tee /tmp/crisk-sync.log; then
    echo -e "${GREEN}✓ Validation passed${NC}"
else
    EXIT_CODE=$?
    if [ $EXIT_CODE -eq 1 ]; then
        echo -e "${YELLOW}⚠ Warnings detected (variance 90-95%)${NC}"
    else
        echo -e "${RED}✗ Validation failed (variance <90%)${NC}"
    fi
    echo "  See /tmp/crisk-sync.log for details"
fi

echo ""

# Show final results
echo "[6/6] Final results - Top 10 Risky CodeBlocks"
echo ""

PGPASSWORD=$PGPASSWORD psql -h $PG_HOST -p $PG_PORT -U $PG_USER -d $PG_DB -c "
SELECT
    LEFT(canonical_file_path, 50) as file_path,
    LEFT(block_name, 30) as block,
    ROUND(risk_score::numeric, 2) as risk,
    incident_count as incidents,
    staleness_days as stale_days,
    co_change_count as couplings
FROM code_blocks
WHERE repo_id = $REPO_ID
    AND risk_score IS NOT NULL
ORDER BY risk_score DESC
LIMIT 10;
" 2>/dev/null || echo -e "${RED}✗ Could not query results${NC}"

echo ""

# Summary statistics
echo "═══════════════════════════════════════════════════════════════"
echo "  Summary Statistics"
echo "═══════════════════════════════════════════════════════════════"
echo ""

PGPASSWORD=$PGPASSWORD psql -h $PG_HOST -p $PG_PORT -U $PG_USER -d $PG_DB -c "
SELECT
    COUNT(*) as total_blocks,
    COUNT(*) FILTER (WHERE risk_score IS NOT NULL) as with_risk_score,
    COUNT(*) FILTER (WHERE risk_score >= 75) as high_risk,
    COUNT(*) FILTER (WHERE risk_score >= 50 AND risk_score < 75) as medium_risk,
    COUNT(*) FILTER (WHERE risk_score < 50) as low_risk,
    ROUND(AVG(risk_score)::numeric, 2) as avg_risk_score,
    ROUND(MAX(risk_score)::numeric, 2) as max_risk_score,
    SUM(incident_count) as total_incidents,
    COUNT(*) FILTER (WHERE incident_count > 0) as blocks_with_incidents
FROM code_blocks
WHERE repo_id = $REPO_ID;
" 2>/dev/null || echo -e "${RED}✗ Could not query statistics${NC}"

echo ""

# Neo4j validation
echo "Neo4j Graph Statistics:"
echo ""

curl -s -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
    -H "Content-Type: application/json" \
    -X POST http://localhost:7475/db/neo4j/tx/commit \
    -d '{
        "statements":[
            {"statement":"MATCH (n) WHERE n.repo_id = 11 RETURN labels(n)[0] as type, count(*) as count ORDER BY count DESC"}
        ]
    }' | python3 -c "
import sys, json
try:
    r = json.load(sys.stdin)
    if r.get('results') and r['results'][0].get('data'):
        data = r['results'][0]['data']
        print('  Node Type          Count')
        print('  ─────────────────  ─────')
        for row in data:
            if row.get('row'):
                print(f'  {row[\"row\"][0]:20} {row[\"row\"][1]}')
except:
    print('  Could not query Neo4j')
" 2>/dev/null

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo -e "${GREEN}✓ Pipeline test completed!${NC}"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "Logs saved to:"
echo "  /tmp/crisk-incident.log"
echo "  /tmp/crisk-ownership.log"
echo "  /tmp/crisk-coupling.log"
echo "  /tmp/crisk-sync.log"
echo ""
echo "Next steps:"
echo "  • Test MCP server: ./bin/crisk-check-server"
echo "  • Query via CLI: ./bin/crisk check <file> --explain"
echo "  • Inspect Neo4j: http://localhost:7475 (neo4j/CHANGE_THIS_PASSWORD_IN_PRODUCTION_123)"
echo ""
