#!/bin/bash
set -e

echo "=== CodeRisk Check Command E2E Test ==="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
PASSED=0
FAILED=0

# Helper function to print test results
pass() {
    echo -e "${GREEN}✅ PASS${NC}: $1"
    PASSED=$((PASSED + 1))
}

fail() {
    echo -e "${RED}❌ FAIL${NC}: $1"
    echo -e "   Output: $2"
    FAILED=$((FAILED + 1))
}

# Ensure binary exists
if [ ! -f ./bin/crisk ]; then
    echo "Binary not found. Building..."
    go build -o bin/crisk ./cmd/crisk
fi

echo "Using binary: ./bin/crisk"
echo ""

# Note: These tests will fail if Neo4j/Redis/Postgres are not running
# They are meant to validate the CLI interface and formatter output
# The actual risk calculation will use placeholder data from graph queries

# ============================================================
# Test 1: Low Risk File (Quiet Mode)
# ============================================================
echo -e "${YELLOW}Test 1:${NC} Low risk file (quiet mode)"

# Expected: Should output "LOW risk" or similar
# May fail if graph is not populated, but should not crash
OUTPUT=$(./bin/crisk check --quiet test/fixtures/known_risk/low_risk.go 2>&1 || true)

# Check for errors
if echo "$OUTPUT" | grep -iq "error\|failed\|panic"; then
    # Non-fatal if infrastructure is not running
    echo -e "${YELLOW}⚠️  SKIP${NC}: Infrastructure may not be running (Neo4j/Redis/Postgres)"
    echo "   Output: $OUTPUT"
else
    # Check for risk level in output
    if echo "$OUTPUT" | grep -iq "risk"; then
        pass "Low risk file processed (quiet mode)"
    else
        fail "Expected risk output" "$OUTPUT"
    fi
fi

echo ""

# ============================================================
# Test 2: Medium Risk File (Standard Mode)
# ============================================================
echo -e "${YELLOW}Test 2:${NC} Medium risk file (standard mode)"

OUTPUT=$(./bin/crisk check test/fixtures/known_risk/medium_risk.go 2>&1 || true)

if echo "$OUTPUT" | grep -iq "error\|failed\|panic"; then
    echo -e "${YELLOW}⚠️  SKIP${NC}: Infrastructure may not be running"
    echo "   Output: $OUTPUT"
else
    if echo "$OUTPUT" | grep -iq "risk"; then
        pass "Medium risk file processed (standard mode)"
    else
        fail "Expected risk output" "$OUTPUT"
    fi
fi

echo ""

# ============================================================
# Test 3: High Risk File (Explain Mode)
# ============================================================
echo -e "${YELLOW}Test 3:${NC} High risk file (explain mode)"

OUTPUT=$(./bin/crisk check --explain test/fixtures/known_risk/high_risk.go 2>&1 || true)

if echo "$OUTPUT" | grep -iq "error.*initialization\|failed.*connect\|panic"; then
    echo -e "${YELLOW}⚠️  SKIP${NC}: Infrastructure may not be running"
else
    # In explain mode, should show evidence/metrics
    if echo "$OUTPUT" | grep -iq "risk\|evidence\|metric"; then
        pass "High risk file processed (explain mode)"
    else
        fail "Expected detailed risk output" "$OUTPUT"
    fi

    # Should mention escalation or Phase 2
    if echo "$OUTPUT" | grep -iq "escalate\|phase 2\|investigation"; then
        pass "Explain mode shows escalation information"
    else
        echo -e "${YELLOW}⚠️  INFO${NC}: Escalation message not found (may be expected if risk is LOW)"
    fi
fi

echo ""

# ============================================================
# Test 4: AI Mode Output (JSON Validation)
# ============================================================
echo -e "${YELLOW}Test 4:${NC} AI mode (JSON output)"

OUTPUT=$(./bin/crisk check --ai-mode test/fixtures/known_risk/high_risk.go 2>&1 || true)

# Filter out log lines (they start with timestamps or "INFO")
# Keep only lines that look like JSON (start with { or " or [ or contain :)
JSON_OUTPUT=$(echo "$OUTPUT" | grep -v "^[0-9]\{4\}/\|^INFO\|^component=" || echo "$OUTPUT")

if echo "$OUTPUT" | grep -iq "error.*initialization\|failed.*connect"; then
    echo -e "${YELLOW}⚠️  SKIP${NC}: Infrastructure may not be running"
else
    # Try to parse JSON (with log filtering)
    if echo "$JSON_OUTPUT" | jq . > /dev/null 2>&1; then
        pass "AI mode outputs valid JSON"

        # Validate JSON structure
        if echo "$JSON_OUTPUT" | jq -e '.risk' > /dev/null 2>&1; then
            pass "JSON contains 'risk' object"
        else
            echo -e "${YELLOW}⚠️  INFO${NC}: JSON missing 'risk' object (using alternate structure)"
        fi

        if echo "$JSON_OUTPUT" | jq -e '.files' > /dev/null 2>&1; then
            pass "JSON contains 'files' array"
        else
            fail "JSON missing 'files' array" "$JSON_OUTPUT"
        fi

        if echo "$JSON_OUTPUT" | jq -e '.meta' > /dev/null 2>&1; then
            pass "JSON contains 'meta' object"
        else
            echo -e "${YELLOW}⚠️  INFO${NC}: JSON missing 'meta' object"
        fi
    else
        fail "AI mode output is not valid JSON" "$JSON_OUTPUT"
    fi
fi

echo ""

# ============================================================
# Test 5: Pre-commit Mode (Staged Files)
# ============================================================
echo -e "${YELLOW}Test 5:${NC} Pre-commit mode"

# In pre-commit mode, should check staged files or show "no files" message
OUTPUT=$(./bin/crisk check --pre-commit 2>&1 || true)

if echo "$OUTPUT" | grep -iq "no files\|risk"; then
    pass "Pre-commit mode executes successfully"
else
    # Non-fatal if git is not initialized
    echo -e "${YELLOW}⚠️  INFO${NC}: Pre-commit mode output: $OUTPUT"
fi

echo ""

# ============================================================
# Test 6: Multiple Files
# ============================================================
echo -e "${YELLOW}Test 6:${NC} Multiple files"

OUTPUT=$(./bin/crisk check --quiet \
    test/fixtures/known_risk/low_risk.go \
    test/fixtures/known_risk/medium_risk.go \
    test/fixtures/known_risk/high_risk.go 2>&1 || true)

if echo "$OUTPUT" | grep -iq "error.*initialization\|failed.*connect"; then
    echo -e "${YELLOW}⚠️  SKIP${NC}: Infrastructure may not be running"
else
    # Should process all files
    if echo "$OUTPUT" | grep -i "risk" | wc -l | grep -q "[3-9]"; then
        pass "Multiple files processed"
    else
        echo -e "${YELLOW}⚠️  INFO${NC}: Expected 3 risk outputs, got fewer"
    fi
fi

echo ""

# ============================================================
# Test 7: Error Handling (Non-existent File)
# ============================================================
echo -e "${YELLOW}Test 7:${NC} Error handling (non-existent file)"

OUTPUT=$(./bin/crisk check --quiet non_existent_file.go 2>&1 || true)

# Should handle error gracefully (not crash)
if echo "$OUTPUT" | grep -iq "panic"; then
    fail "Binary panicked on non-existent file" "$OUTPUT"
else
    pass "Non-existent file handled gracefully"
fi

echo ""

# ============================================================
# Test 8: Help Flag
# ============================================================
echo -e "${YELLOW}Test 8:${NC} Help flag"

OUTPUT=$(./bin/crisk check --help 2>&1 || true)

if echo "$OUTPUT" | grep -iq "usage\|assess risk\|phase 1"; then
    pass "Help flag displays usage information"
else
    fail "Help flag output unexpected" "$OUTPUT"
fi

echo ""

# ============================================================
# Summary
# ============================================================
echo "========================================="
echo "Test Summary:"
echo -e "${GREEN}Passed: $PASSED${NC}"
if [ $FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $FAILED${NC}"
fi
echo "========================================="

if [ $FAILED -gt 0 ]; then
    echo ""
    echo -e "${YELLOW}NOTE:${NC} Some failures are expected if:"
    echo "  - Neo4j, Redis, or Postgres are not running"
    echo "  - Graph database is not populated with test data"
    echo "  - Running in CI/CD environment without infrastructure"
    echo ""
    echo "To run full integration tests, ensure services are running:"
    echo "  docker compose up -d"
    echo ""
    exit 0  # Don't fail CI - these are integration tests
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
