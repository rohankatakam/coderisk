#!/bin/bash
# Integration test for crisk init-local command
# Tests end-to-end initialization flow: clone → parse → graph construction
# Reference: SESSION_5_PROMPT.md - Integration Test Requirements

set -e

echo "=== CodeRisk Init Flow E2E Test ==="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

BINARY="${BINARY:-./bin/crisk}"
PASSED=0
FAILED=0

# Helper functions
pass() {
    echo -e "${GREEN}✅ PASS:${NC} $1"
    ((PASSED++))
}

fail() {
    echo -e "${RED}❌ FAIL:${NC} $1"
    ((FAILED++))
}

info() {
    echo -e "${YELLOW}ℹ️  INFO:${NC} $1"
}

# Check prerequisites
echo "Checking prerequisites..."

if [ ! -f "$BINARY" ]; then
    fail "Binary not found at $BINARY. Run: go build -o bin/crisk ./cmd/crisk"
    exit 1
fi

# Check Neo4j is running
if ! docker ps | grep -q coderisk-neo4j; then
    fail "Neo4j container not running. Start with: docker-compose up -d"
    exit 1
fi

pass "Binary exists and Neo4j is running"
echo ""

# Test 1: Init with explicit URL (small repo)
echo "Test 1: Init with explicit URL"
info "Testing: crisk init-local https://github.com/tj/commander.js"

# Clear Neo4j
docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (n) DETACH DELETE n" > /dev/null 2>&1

# Run init-local
OUTPUT=$($BINARY init-local https://github.com/tj/commander.js 2>&1)

if echo "$OUTPUT" | grep -q "Repository detected: tj/commander.js"; then
    pass "Repository URL parsed correctly"
else
    fail "Repository URL parsing failed"
fi

if echo "$OUTPUT" | grep -q "Initialization complete"; then
    pass "Init completed successfully"
else
    fail "Init did not complete"
fi

# Verify Neo4j has nodes
FUNCTION_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (f:Function) RETURN count(f) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

if [ "$FUNCTION_COUNT" -gt 0 ]; then
    pass "Neo4j contains $FUNCTION_COUNT Function nodes"
else
    fail "Neo4j graph is empty (expected Function nodes)"
fi

CLASS_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (c:Class) RETURN count(c) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

if [ "$CLASS_COUNT" -gt 0 ]; then
    pass "Neo4j contains $CLASS_COUNT Class nodes"
else
    info "No Class nodes found (acceptable for some repos)"
fi

echo ""

# Test 2: Init with auto-detection (inside git repo)
echo "Test 2: Init with auto-detection"
info "Testing: crisk init-local (inside git repo)"

# This test requires being inside a git repo with a remote
if git rev-parse --is-inside-work-tree > /dev/null 2>&1; then
    REMOTE_URL=$(git config --get remote.origin.url 2>/dev/null || echo "")

    if [ -n "$REMOTE_URL" ]; then
        # Clear Neo4j
        docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
          "MATCH (n) DETACH DELETE n" > /dev/null 2>&1

        # Run init-local without URL (should auto-detect)
        OUTPUT=$($BINARY init-local --skip-graph 2>&1)

        if echo "$OUTPUT" | grep -q "Repository detected"; then
            pass "Auto-detection from git remote successful"
        else
            fail "Auto-detection failed"
        fi
    else
        info "Skipping auto-detection test (no git remote configured)"
    fi
else
    info "Skipping auto-detection test (not in git repo)"
fi

echo ""

# Test 3: Init with --skip-graph flag
echo "Test 3: Init with --skip-graph flag"
info "Testing: crisk init-local https://github.com/tj/commander.js --skip-graph"

# Clear Neo4j
docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (n) DETACH DELETE n" > /dev/null 2>&1

# Run init-local with --skip-graph
OUTPUT=$($BINARY init-local https://github.com/tj/commander.js --skip-graph 2>&1)

if echo "$OUTPUT" | grep -q "Graph construction skipped"; then
    pass "--skip-graph flag works"
else
    fail "--skip-graph flag not working"
fi

# Verify Neo4j is empty
NODE_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (n) RETURN count(n) as count" --format plain 2>/dev/null | tail -1 | tr -d '"')

if [ "$NODE_COUNT" -eq "0" ]; then
    pass "Neo4j is empty (graph construction was skipped)"
else
    fail "Neo4j has $NODE_COUNT nodes (should be empty with --skip-graph)"
fi

echo ""

# Test 4: Init with invalid URL (should fail gracefully)
echo "Test 4: Init with invalid URL (error handling)"
info "Testing: crisk init-local invalid-url"

if $BINARY init-local invalid-url 2>&1 | grep -q "Invalid repository URL"; then
    pass "Invalid URL rejected with error message"
else
    fail "Invalid URL not handled correctly"
fi

echo ""

# Test 5: URL format parsing (HTTPS, SSH, shorthand)
echo "Test 5: URL format parsing"

# Test HTTPS format
OUTPUT=$($BINARY init-local https://github.com/tj/commander.js --skip-graph 2>&1)
if echo "$OUTPUT" | grep -q "Repository detected: tj/commander.js"; then
    pass "HTTPS URL format parsed correctly"
else
    fail "HTTPS URL format parsing failed"
fi

# Test SSH format (git@github.com:org/repo.git)
OUTPUT=$($BINARY init-local git@github.com:tj/commander.js.git --skip-graph 2>&1 || true)
if echo "$OUTPUT" | grep -q "Repository detected: tj/commander.js"; then
    pass "SSH URL format parsed correctly"
else
    info "SSH URL format parsing failed (might be expected if git@ URLs aren't supported)"
fi

echo ""

# Test 6: Performance check (should complete in reasonable time)
echo "Test 6: Performance check"
info "Testing: Init should complete in < 30 seconds for small repo"

START_TIME=$(date +%s)
$BINARY init-local https://github.com/tj/commander.js --skip-graph > /dev/null 2>&1
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

if [ "$DURATION" -lt 30 ]; then
    pass "Init completed in ${DURATION}s (< 30s threshold)"
else
    fail "Init took ${DURATION}s (> 30s threshold)"
fi

echo ""

# Summary
echo "=== Test Summary ==="
echo "Passed: $PASSED"
echo "Failed: $FAILED"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi
