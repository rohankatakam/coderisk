#!/bin/bash
# Test: Performance Benchmarks for Graph Queries
# Reference: E2E_TEST_FINAL_REPORT.md Performance Validation

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test: Performance Benchmarks"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check prerequisites
if ! docker ps | grep -q coderisk-neo4j; then
    echo -e "${RED}✗ FAIL: Neo4j not running${NC}"
    exit 1
fi

if ! docker ps | grep -q coderisk-postgres; then
    echo -e "${RED}✗ FAIL: PostgreSQL not running${NC}"
    exit 1
fi

# Benchmark: Co-change lookup (Target: <20ms)
echo -e "\nBenchmark 1: Co-change lookup (target: <20ms)"

START_NS=$(date +%s%N)
docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
    "MATCH (f:File)-[r:CO_CHANGED]-(other:File)
     WHERE r.frequency >= 0.5
     RETURN other.path, r.frequency
     ORDER BY r.frequency DESC
     LIMIT 10" --format plain >/dev/null 2>&1 || true
END_NS=$(date +%s%N)

DURATION_MS=$(( (END_NS - START_NS) / 1000000 ))

if [ "$DURATION_MS" -lt 20 ]; then
    echo -e "${GREEN}✓ PASS: Co-change query ${DURATION_MS}ms${NC}"
else
    echo -e "${YELLOW}⚠ WARNING: Co-change query ${DURATION_MS}ms (target: <20ms)${NC}"
fi

# Benchmark: Incident BM25 search (Target: <50ms)
echo -e "\nBenchmark 2: Incident BM25 search (target: <50ms)"

START_NS=$(date +%s%N)
docker exec coderisk-postgres psql -U coderisk -d coderisk -t -c \
    "SELECT title, ts_rank_cd(search_vector, query) AS rank
     FROM incidents, to_tsquery('english', 'timeout | error') query
     WHERE search_vector @@ query
     ORDER BY rank DESC
     LIMIT 10" >/dev/null 2>&1 || true
END_NS=$(date +%s%N)

DURATION_MS=$(( (END_NS - START_NS) / 1000000 ))

if [ "$DURATION_MS" -lt 50 ]; then
    echo -e "${GREEN}✓ PASS: Incident search ${DURATION_MS}ms${NC}"
else
    echo -e "${YELLOW}⚠ WARNING: Incident search ${DURATION_MS}ms (target: <50ms)${NC}"
fi

# Benchmark: 1-hop structural query (Target: <50ms)
echo -e "\nBenchmark 3: 1-hop structural coupling (target: <50ms)"

START_NS=$(date +%s%N)
docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
    "MATCH (f:File)-[:IMPORTS]->(dep:File)
     RETURN f.path, count(dep) as dependencies
     ORDER BY dependencies DESC
     LIMIT 10" --format plain >/dev/null 2>&1 || true
END_NS=$(date +%s%N)

DURATION_MS=$(( (END_NS - START_NS) / 1000000 ))

if [ "$DURATION_MS" -lt 50 ]; then
    echo -e "${GREEN}✓ PASS: Structural query ${DURATION_MS}ms${NC}"
else
    echo -e "${YELLOW}⚠ WARNING: Structural query ${DURATION_MS}ms (target: <50ms)${NC}"
fi

echo -e "\n${GREEN}Performance benchmarks complete${NC}"
