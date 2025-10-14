#!/bin/bash
# CodeRisk Init E2E Test - omnara-ai/omnara
# Tests all 3 layers work correctly

set -e

echo "🧪 CodeRisk Init E2E Test"
echo "Repository: omnara-ai/omnara"
echo "=========================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test tracker
TESTS_PASSED=0
TESTS_FAILED=0

test_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ PASS${NC}: $2"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗ FAIL${NC}: $2"
        ((TESTS_FAILED++))
    fi
}

echo "[1/5] Cleaning environment..."
docker compose down -v > /dev/null 2>&1
docker compose up -d
echo "Waiting for services to start (30s)..."
sleep 30

# Check services
echo ""
echo "[2/5] Checking services..."
docker compose ps | grep -q "Up" && test_result 0 "Docker services running" || test_result 1 "Docker services failed"

curl -s -o /dev/null -w "%{http_code}" http://localhost:7475 | grep -q "200" && test_result 0 "Neo4j accessible" || test_result 1 "Neo4j not accessible"

# Run init
echo ""
echo "[3/5] Running crisk init omnara-ai/omnara..."
echo -e "${YELLOW}Note: This will take 1-2 minutes${NC}"
echo ""

if ./crisk init omnara-ai/omnara; then
    test_result 0 "Init command completed"
else
    test_result 1 "Init command failed"
    exit 1
fi

# Validate Layer 1 (Structure)
echo ""
echo "[4/5] Validating Layer 1 (Structure)..."

FILE_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
    "MATCH (f:File) RETURN count(f) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

if [ "$FILE_COUNT" -gt 30 ]; then
    test_result 0 "File nodes exist ($FILE_COUNT files)"
else
    test_result 1 "File nodes missing or insufficient"
fi

FUNC_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
    "MATCH (fn:Function) RETURN count(fn) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

if [ "$FUNC_COUNT" -gt 50 ]; then
    test_result 0 "Function nodes exist ($FUNC_COUNT functions)"
else
    test_result 1 "Function nodes missing or insufficient"
fi

# Validate Layer 2 (Temporal)
echo ""
echo "Validating Layer 2 (Temporal)..."

COMMIT_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
    "MATCH (c:Commit) RETURN count(c) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

if [ "$COMMIT_COUNT" -gt 100 ]; then
    test_result 0 "Commit nodes exist ($COMMIT_COUNT commits)"
else
    test_result 1 "Commit nodes missing or insufficient"
fi

DEV_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
    "MATCH (d:Developer) RETURN count(d) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

if [ "$DEV_COUNT" -gt 2 ]; then
    test_result 0 "Developer nodes exist ($DEV_COUNT developers)"
else
    test_result 1 "Developer nodes missing"
fi

# Validate Layer 3 (Incidents)
echo ""
echo "Validating Layer 3 (Incidents)..."

ISSUE_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
    "MATCH (i:Issue) RETURN count(i) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

if [ "$ISSUE_COUNT" -gt 5 ]; then
    test_result 0 "Issue nodes exist ($ISSUE_COUNT issues)"
else
    test_result 1 "Issue nodes missing or insufficient"
fi

# Cross-layer relationships
echo ""
echo "[5/5] Validating cross-layer relationships..."

IMPORTS_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
    "MATCH ()-[r:IMPORTS]->() RETURN count(r) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

if [ "$IMPORTS_COUNT" -gt 20 ]; then
    test_result 0 "IMPORTS relationships exist ($IMPORTS_COUNT)"
else
    test_result 1 "IMPORTS relationships missing"
fi

MODIFIES_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
    "MATCH ()-[r:MODIFIES]->() RETURN count(r) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

if [ "$MODIFIES_COUNT" -gt 100 ]; then
    test_result 0 "MODIFIES relationships exist ($MODIFIES_COUNT)"
else
    test_result 1 "MODIFIES relationships missing"
fi

# Summary
echo ""
echo "=========================================="
echo "Test Summary:"
echo -e "  ${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "  ${RED}Failed: $TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ ALL TESTS PASSED${NC}"
    echo ""
    echo "🎯 Next steps:"
    echo "  • Browse graph: http://localhost:7475"
    echo "  • Credentials: neo4j / CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
    echo "  • Test risk check: crisk check <file>"
    exit 0
else
    echo -e "${RED}❌ SOME TESTS FAILED${NC}"
    echo ""
    echo "Check logs:"
    echo "  docker compose logs"
    exit 1
fi
