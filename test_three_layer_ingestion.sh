#!/bin/bash
# Three-Layer Ingestion Test After Refactoring
# Tests that all three layers (Structure, Temporal, Incidents) work correctly

set -e

echo "=================================================="
echo "Three-Layer Ingestion Test - Post Refactoring"
echo "=================================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

BINARY="./crisk"
PASSED=0
FAILED=0

pass() {
    echo -e "${GREEN}✅ PASS:${NC} $1"
    ((PASSED++))
}

fail() {
    echo -e "${RED}❌ FAIL:${NC} $1"
    ((FAILED++))
}

info() {
    echo -e "${BLUE}ℹ️  ${NC} $1"
}

section() {
    echo ""
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

# Prerequisites
section "Prerequisites Check"

if [ ! -f "$BINARY" ]; then
    fail "Binary not found. Building..."
    go build -o crisk ./cmd/crisk || exit 1
    pass "Binary built successfully"
fi

if ! docker ps | grep -q coderisk-neo4j; then
    fail "Neo4j not running. Start with: docker compose up -d"
    exit 1
fi
pass "Neo4j is running"

# Clear Neo4j database
info "Clearing Neo4j database..."
docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (n) DETACH DELETE n" > /dev/null 2>&1
pass "Neo4j database cleared"

echo ""

# Use a small test repository
TEST_REPO="https://github.com/tj/commander.js"
info "Test repository: $TEST_REPO"
echo ""

# ==================================================
# LAYER 1: Structure (Tree-sitter AST parsing)
# ==================================================
section "Layer 1: Structure Ingestion (Tree-sitter)"

info "Running: crisk init $TEST_REPO"
OUTPUT=$($BINARY init $TEST_REPO 2>&1)

if echo "$OUTPUT" | grep -q "Repository detected"; then
    pass "Repository URL parsed correctly"
else
    fail "Repository URL parsing failed"
    echo "$OUTPUT"
fi

if echo "$OUTPUT" | grep -q "Parsing.*files"; then
    pass "Tree-sitter parsing initiated"
else
    fail "Tree-sitter parsing not detected"
fi

# Verify Layer 1: Function and File nodes
info "Verifying Layer 1 nodes in Neo4j..."

FILE_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (f:File) RETURN count(f) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

if [ "$FILE_COUNT" -gt 0 ]; then
    pass "Layer 1: Found $FILE_COUNT File nodes"
else
    fail "Layer 1: No File nodes created"
fi

FUNCTION_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (fn:Function) RETURN count(fn) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

if [ "$FUNCTION_COUNT" -gt 0 ]; then
    pass "Layer 1: Found $FUNCTION_COUNT Function nodes"
else
    fail "Layer 1: No Function nodes created"
fi

# Check IMPORTS relationships
IMPORT_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH ()-[r:IMPORTS]->() RETURN count(r) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

if [ "$IMPORT_COUNT" -gt 0 ]; then
    pass "Layer 1: Found $IMPORT_COUNT IMPORTS relationships"
else
    info "Layer 1: No IMPORTS relationships (may be normal for small repos)"
fi

# ==================================================
# LAYER 2: Temporal (Git history analysis)
# ==================================================
section "Layer 2: Temporal Ingestion (Git History)"

info "Verifying Layer 2 nodes in Neo4j..."

COMMIT_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (c:Commit) RETURN count(c) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

if [ "$COMMIT_COUNT" -gt 0 ]; then
    pass "Layer 2: Found $COMMIT_COUNT Commit nodes"
else
    fail "Layer 2: No Commit nodes created"
fi

DEVELOPER_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (d:Developer) RETURN count(d) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

if [ "$DEVELOPER_COUNT" -gt 0 ]; then
    pass "Layer 2: Found $DEVELOPER_COUNT Developer nodes"
else
    fail "Layer 2: No Developer nodes created"
fi

# Check AUTHORED relationships
AUTHORED_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH ()-[r:AUTHORED]->() RETURN count(r) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

if [ "$AUTHORED_COUNT" -gt 0 ]; then
    pass "Layer 2: Found $AUTHORED_COUNT AUTHORED relationships"
else
    fail "Layer 2: No AUTHORED relationships created"
fi

# Check MODIFIED relationships
MODIFIED_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH ()-[r:MODIFIED]->() RETURN count(r) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

if [ "$MODIFIED_COUNT" -gt 0 ]; then
    pass "Layer 2: Found $MODIFIED_COUNT MODIFIED relationships"
else
    fail "Layer 2: No MODIFIED relationships created"
fi

# Check CO_CHANGED relationships
CO_CHANGED_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH ()-[r:CO_CHANGED]->() RETURN count(r) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

if [ "$CO_CHANGED_COUNT" -gt 0 ]; then
    pass "Layer 2: Found $CO_CHANGED_COUNT CO_CHANGED relationships"
else
    info "Layer 2: No CO_CHANGED relationships (may be normal for small repos)"
fi

# ==================================================
# LAYER 3: Incidents (GitHub Issues integration)
# ==================================================
section "Layer 3: Incident Ingestion (GitHub Issues)"

info "Note: Layer 3 requires GitHub token. Checking if issues were ingested..."

ISSUE_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (i:Issue) RETURN count(i) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

if [ "$ISSUE_COUNT" -gt 0 ]; then
    pass "Layer 3: Found $ISSUE_COUNT Issue nodes"

    # Check LINKED_TO relationships
    LINKED_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
      "MATCH ()-[r:LINKED_TO]->() RETURN count(r) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

    if [ "$LINKED_COUNT" -gt 0 ]; then
        pass "Layer 3: Found $LINKED_COUNT LINKED_TO relationships"
    else
        info "Layer 3: No LINKED_TO relationships"
    fi
else
    info "Layer 3: No Issue nodes (GitHub token may not be configured)"
    info "Layer 3: This is expected if GITHUB_TOKEN is not set"
fi

# ==================================================
# Summary
# ==================================================
section "Test Summary"

TOTAL=$((PASSED + FAILED))
echo "Total Tests: $TOTAL"
echo -e "${GREEN}Passed: $PASSED${NC}"
if [ $FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED${NC}"
else
    echo -e "${GREEN}Failed: 0${NC}"
fi

echo ""
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}✅ ALL TESTS PASSED${NC}"
    echo -e "${GREEN}Three-layer ingestion working correctly!${NC}"
    echo -e "${GREEN}========================================${NC}"
    exit 0
else
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}❌ SOME TESTS FAILED${NC}"
    echo -e "${RED}Review failures above${NC}"
    echo -e "${RED}========================================${NC}"
    exit 1
fi
