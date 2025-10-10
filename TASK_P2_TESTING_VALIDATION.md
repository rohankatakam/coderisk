# Task: Add Integration Tests & Validation (P2 - MEDIUM)

**Priority:** P2 - TESTING & VALIDATION
**Estimated Time:** 2-3 hours
**Gap Reference:** Testing gaps from [E2E_TEST_FINAL_REPORT.md](E2E_TEST_FINAL_REPORT.md#medium-priority-p2---testing--validation)

---

## Context & Problem Statement

**Current Issue:** Missing integration tests and validation for Layers 2 & 3

**Gaps Identified:**
1. No automated test for CO_CHANGED edge creation (Layer 2)
2. No automated test for CAUSED_BY edge creation (Layer 3)
3. No performance benchmarks for graph queries
4. No verification of post-init graph state
5. Minor CLI bugs (NULL handling, --version flag)

**What Exists:**
- ✅ Unit tests for internal packages
- ✅ Manual E2E test procedures
- ✅ Integration test framework structure

**What's Missing:**
- ❌ Automated Layer 2/3 validation tests
- ❌ Performance benchmarks
- ❌ Post-init verification scripts
- ❌ CLI bug fixes

---

## Before You Start

### 1. Read Documentation (REQUIRED)

```bash
# Review existing tests
ls -la test/integration/
cat test/integration/test_pre_commit_hook.sh
cat test/integration/test_verbosity.sh

# Check test framework
cat Makefile | grep -A 5 "test\|integration"

# Review E2E test findings
cat E2E_TEST_FINAL_REPORT.md | grep -A 20 "Test 1.3\|Test 2.2"
```

### 2. Understand Test Requirements

From [E2E_TEST_FINAL_REPORT.md](E2E_TEST_FINAL_REPORT.md):
- **Test 1.3:** CO_CHANGED edges - Expected >0, Actual 0 (FAIL)
- **Test 2.2:** CAUSED_BY edges - Expected 1, Actual 0 (FAIL)
- **Performance:** <20ms for co-change, <50ms for incident search

---

## Implementation Tasks

### Task 1: Layer 2 Validation Test (CO_CHANGED Edges)

**File:** `test/integration/test_layer2_cochanged.sh` (NEW)

**Purpose:** Verify CO_CHANGED edges are created during init-local

```bash
#!/bin/bash
# Test: Layer 2 CO_CHANGED Edge Creation
# Reference: E2E_TEST_FINAL_REPORT.md Test 1.3

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CRISK_BIN="$PROJECT_ROOT/crisk"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test: Layer 2 CO_CHANGED Edge Creation"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check prerequisites
if [ ! -f "$CRISK_BIN" ]; then
    echo -e "${RED}✗ FAIL: crisk binary not found at $CRISK_BIN${NC}"
    echo "Run: make build"
    exit 1
fi

if ! docker ps | grep -q coderisk-neo4j; then
    echo -e "${RED}✗ FAIL: Neo4j container not running${NC}"
    echo "Run: docker compose up -d"
    exit 1
fi

# Setup test repository
TEST_REPO="/tmp/coderisk-test-layer2-$$"
echo "Setting up test repository: $TEST_REPO"

mkdir -p "$TEST_REPO"
cd "$TEST_REPO"

# Initialize git with test files
git init
git config user.name "Test User"
git config user.email "test@example.com"

# Create test files that will co-change
cat > fileA.js <<'EOF'
export function funcA() {
    return "A";
}
EOF

cat > fileB.js <<'EOF'
import { funcA } from './fileA.js';

export function funcB() {
    return funcA() + "B";
}
EOF

git add fileA.js fileB.js
git commit -m "Initial commit"

# Make co-changes (change both files together)
for i in {1..5}; do
    echo "// Change $i" >> fileA.js
    echo "// Change $i" >> fileB.js
    git add fileA.js fileB.js
    git commit -m "Co-change $i"
done

# Make solo change to fileA (to not have 100% co-change)
echo "// Solo change" >> fileA.js
git add fileA.js
git commit -m "Solo change to fileA"

echo "Test repository created with 5 co-changes"

# Run init-local
echo "Running init-local..."
if ! "$CRISK_BIN" init-local 2>&1 | tee /tmp/init-local-output.log; then
    echo -e "${YELLOW}⚠ WARNING: init-local had errors${NC}"
    # Continue anyway to check if edges were created
fi

# Wait for processing to complete
sleep 2

# Query Neo4j for CO_CHANGED edges
echo "Querying Neo4j for CO_CHANGED edges..."

EDGE_COUNT=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
    "MATCH ()-[r:CO_CHANGED]->() RETURN count(r) as count" --format plain 2>/dev/null | tail -1 || echo "0")

echo "CO_CHANGED edge count: $EDGE_COUNT"

# Check specific edge between fileA and fileB
SPECIFIC_EDGE=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
    "MATCH (a:File)-[r:CO_CHANGED]-(b:File)
     WHERE a.path CONTAINS 'fileA.js' AND b.path CONTAINS 'fileB.js'
     RETURN r.frequency as frequency, r.co_changes as co_changes" --format plain 2>/dev/null | tail -1 || echo "")

# Validation
if [ "$EDGE_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✓ PASS: CO_CHANGED edges created (count: $EDGE_COUNT)${NC}"

    if [ -n "$SPECIFIC_EDGE" ]; then
        echo -e "${GREEN}✓ PASS: fileA.js <-> fileB.js co-change detected${NC}"
        echo "  Details: $SPECIFIC_EDGE"

        # Validate frequency (should be ~0.83 = 5 co-changes / 6 total fileA commits)
        FREQUENCY=$(echo "$SPECIFIC_EDGE" | awk '{print $1}')
        if (( $(echo "$FREQUENCY >= 0.7" | bc -l) )); then
            echo -e "${GREEN}✓ PASS: Frequency >= 0.7 (actual: $FREQUENCY)${NC}"
        else
            echo -e "${YELLOW}⚠ WARNING: Frequency < 0.7 (actual: $FREQUENCY)${NC}"
        fi
    else
        echo -e "${YELLOW}⚠ WARNING: Specific fileA-fileB edge not found${NC}"
    fi

    # Performance test
    echo "Performance test: CO_CHANGE query..."
    START_MS=$(($(date +%s%N)/1000000))

    docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
        "MATCH (f:File)-[r:CO_CHANGED]-(other:File)
         WHERE f.path CONTAINS 'fileA.js' AND r.frequency >= 0.5
         RETURN other.path, r.frequency
         LIMIT 10" --format plain >/dev/null 2>&1

    END_MS=$(($(date +%s%N)/1000000))
    DURATION_MS=$((END_MS - START_MS))

    if [ "$DURATION_MS" -lt 20 ]; then
        echo -e "${GREEN}✓ PASS: Query performance ${DURATION_MS}ms (target: <20ms)${NC}"
    else
        echo -e "${YELLOW}⚠ WARNING: Query performance ${DURATION_MS}ms (target: <20ms)${NC}"
    fi

    # Cleanup
    rm -rf "$TEST_REPO"
    exit 0
else
    echo -e "${RED}✗ FAIL: No CO_CHANGED edges created${NC}"
    echo ""
    echo "Debug information:"
    echo "1. Check init-local logs:"
    tail -20 /tmp/init-local-output.log
    echo ""
    echo "2. Check Neo4j connectivity:"
    docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
        "MATCH (n) RETURN labels(n)[0] as type, count(n) as count LIMIT 5" --format plain 2>&1 || true
    echo ""
    echo "Possible causes:"
    echo "- Temporal analysis timed out (check logs for 'temporal analysis')"
    echo "- Neo4j transaction not committed (Gap A1)"
    echo "- CO_CHANGED edge creation code not called"

    # Cleanup
    rm -rf "$TEST_REPO"
    exit 1
fi
```

**Make executable:**
```bash
chmod +x test/integration/test_layer2_cochanged.sh
```

---

### Task 2: Layer 3 Validation Test (CAUSED_BY Edges)

**File:** `test/integration/test_layer3_causedby.sh` (NEW)

```bash
#!/bin/bash
# Test: Layer 3 CAUSED_BY Edge Creation
# Reference: E2E_TEST_FINAL_REPORT.md Test 2.2

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CRISK_BIN="$PROJECT_ROOT/crisk"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test: Layer 3 CAUSED_BY Edge Creation"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Prerequisites check
if [ ! -f "$CRISK_BIN" ]; then
    echo -e "${RED}✗ FAIL: crisk binary not found${NC}"
    exit 1
fi

if ! docker ps | grep -q coderisk-neo4j; then
    echo -e "${RED}✗ FAIL: Neo4j not running${NC}"
    exit 1
fi

if ! docker ps | grep -q coderisk-postgres; then
    echo -e "${RED}✗ FAIL: PostgreSQL not running${NC}"
    exit 1
fi

# Setup test repository
TEST_REPO="/tmp/coderisk-test-layer3-$$"
mkdir -p "$TEST_REPO"
cd "$TEST_REPO"

git init
git config user.name "Test User"
git config user.email "test@example.com"

# Create test file
cat > test_file.ts <<'EOF'
export function criticalFunction() {
    // Critical business logic
    return "important";
}
EOF

git add test_file.ts
git commit -m "Add test file"

# Run init-local to create File nodes
echo "Running init-local..."
"$CRISK_BIN" init-local >/dev/null 2>&1

# Create incident
echo "Creating incident..."
INCIDENT_OUTPUT=$("$CRISK_BIN" incident create \
    "Test Incident" \
    "This is a test incident for validation" \
    --severity critical \
    --root-cause "Test root cause")

INCIDENT_ID=$(echo "$INCIDENT_OUTPUT" | grep -oP 'ID: \K[a-f0-9-]+' || echo "")

if [ -z "$INCIDENT_ID" ]; then
    echo -e "${RED}✗ FAIL: Could not create incident${NC}"
    exit 1
fi

echo "Created incident: $INCIDENT_ID"

# Link incident to file
echo "Linking incident to file..."
if ! "$CRISK_BIN" incident link "$INCIDENT_ID" "test_file.ts" --line 2 --function "criticalFunction"; then
    echo -e "${RED}✗ FAIL: Could not link incident${NC}"
    exit 1
fi

# Wait for processing
sleep 1

# Verify PostgreSQL link
echo "Verifying PostgreSQL incident_files link..."
PG_LINK_COUNT=$(docker exec coderisk-postgres psql -U coderisk -d coderisk -t -c \
    "SELECT COUNT(*) FROM incident_files WHERE incident_id = '$INCIDENT_ID'" 2>/dev/null | xargs || echo "0")

if [ "$PG_LINK_COUNT" -eq 1 ]; then
    echo -e "${GREEN}✓ PASS: PostgreSQL link created${NC}"
else
    echo -e "${RED}✗ FAIL: PostgreSQL link not found (count: $PG_LINK_COUNT)${NC}"
    exit 1
fi

# Verify Neo4j Incident node
echo "Verifying Neo4j Incident node..."
INCIDENT_NODE=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
    "MATCH (i:Incident {id: '$INCIDENT_ID'}) RETURN i.title" --format plain 2>/dev/null | tail -1 || echo "")

if [ -n "$INCIDENT_NODE" ]; then
    echo -e "${GREEN}✓ PASS: Incident node exists in Neo4j${NC}"
else
    echo -e "${RED}✗ FAIL: Incident node not found in Neo4j${NC}"
    exit 1
fi

# Verify CAUSED_BY edge (THE CRITICAL TEST)
echo "Verifying CAUSED_BY edge..."
CAUSED_BY_EDGE=$(docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
    "MATCH (i:Incident {id: '$INCIDENT_ID'})-[r:CAUSED_BY]->(f:File)
     RETURN f.path as file, r.confidence as confidence, r.line_number as line" --format plain 2>/dev/null | tail -1 || echo "")

if [ -n "$CAUSED_BY_EDGE" ] && [ "$CAUSED_BY_EDGE" != "0" ]; then
    echo -e "${GREEN}✓ PASS: CAUSED_BY edge created${NC}"
    echo "  Details: $CAUSED_BY_EDGE"

    # Validate properties
    if echo "$CAUSED_BY_EDGE" | grep -q "test_file.ts"; then
        echo -e "${GREEN}✓ PASS: Edge points to correct file${NC}"
    fi

    if echo "$CAUSED_BY_EDGE" | grep -q "1.0"; then
        echo -e "${GREEN}✓ PASS: Confidence = 1.0 (manual link)${NC}"
    fi

    if echo "$CAUSED_BY_EDGE" | grep -q "2"; then
        echo -e "${GREEN}✓ PASS: Line number = 2${NC}"
    fi

    # Cleanup
    rm -rf "$TEST_REPO"
    exit 0
else
    echo -e "${RED}✗ FAIL: CAUSED_BY edge not found${NC}"
    echo ""
    echo "Debug information:"
    echo "1. Incident node query:"
    docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
        "MATCH (i:Incident {id: '$INCIDENT_ID'}) RETURN i" --format plain 2>&1 || true
    echo ""
    echo "2. All edges from incident:"
    docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
        "MATCH (i:Incident {id: '$INCIDENT_ID'})-[r]->(n) RETURN type(r), labels(n)" --format plain 2>&1 || true
    echo ""
    echo "3. File nodes:"
    docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
        "MATCH (f:File) WHERE f.path CONTAINS 'test_file' RETURN f.path, f.unique_id LIMIT 3" --format plain 2>&1 || true
    echo ""
    echo "Possible causes (Gap B1):"
    echo "- Edge creation code not calling Neo4j properly"
    echo "- Transaction not committed"
    echo "- File path mismatch (node ID vs edge target)"

    # Cleanup
    rm -rf "$TEST_REPO"
    exit 1
fi
```

**Make executable:**
```bash
chmod +x test/integration/test_layer3_causedby.sh
```

---

### Task 3: Performance Benchmark Test

**File:** `test/integration/test_performance_benchmarks.sh` (NEW)

```bash
#!/bin/bash
# Test: Performance Benchmarks for Graph Queries
# Reference: E2E_TEST_FINAL_REPORT.md Performance Validation

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test: Performance Benchmarks"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Benchmark: Co-change lookup (Target: <20ms)
echo "Benchmark 1: Co-change lookup (target: <20ms)"

START_NS=$(date +%s%N)
docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
    "MATCH (f:File)-[r:CO_CHANGED]-(other:File)
     WHERE f.path =~ '.*\\.ts' AND r.frequency >= 0.5
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
echo "Benchmark 2: Incident BM25 search (target: <50ms)"

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
echo "Benchmark 3: 1-hop structural coupling (target: <50ms)"

START_NS=$(date +%s%N)
docker exec coderisk-neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" \
    "MATCH (f:File)-[:IMPORTS]->(dep:File)
     WHERE f.path =~ '.*\\.ts'
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

echo ""
echo "Performance summary:"
echo "- Co-change lookup: ${DURATION_MS}ms (target: <20ms)"
echo "- Incident search: ${DURATION_MS}ms (target: <50ms)"
echo "- Structural query: ${DURATION_MS}ms (target: <50ms)"
```

**Make executable:**
```bash
chmod +x test/integration/test_performance_benchmarks.sh
```

---

### Task 4: Fix Minor CLI Bugs

#### Bug 1: Incident Search NULL Handling

**File:** `internal/incidents/search.go`

Already covered in [TASK_P1_EDGE_CREATION_FIXES.md](TASK_P1_EDGE_CREATION_FIXES.md#task-23-fix-cli-null-handling-bug) - use `sql.NullString` for nullable fields.

#### Bug 2: Add --version Flag

**File:** `cmd/crisk/main.go`

```go
package main

import (
    "fmt"
    // ... existing imports ...
)

var (
    Version   = "dev"      // Set by build: -ldflags "-X main.Version=1.0.0"
    BuildTime = "unknown"  // Set by build: -ldflags "-X main.BuildTime=$(date)"
    GitCommit = "unknown"  // Set by build: -ldflags "-X main.GitCommit=$(git rev-parse HEAD)"
)

var rootCmd = &cobra.Command{
    Use:   "crisk",
    Short: "CodeRisk performs sub-5-second risk analysis on your code changes",
    Long: `CodeRisk performs sub-5-second risk analysis on your code changes,
helping you catch potential issues before they reach production.`,
    Version: Version, // ← Add this
}

func init() {
    // ... existing flags ...

    // Custom version template
    rootCmd.SetVersionTemplate(`CodeRisk {{.Version}}
Build time: ` + BuildTime + `
Git commit: ` + GitCommit + `
`)
}
```

**Update Makefile:**

```makefile
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")

LDFLAGS := -X main.Version=$(VERSION) \
           -X main.BuildTime=$(BUILD_TIME) \
           -X main.GitCommit=$(GIT_COMMIT)

build:
	go build -ldflags "$(LDFLAGS)" -o crisk ./cmd/crisk
```

---

### Task 5: Update Integration Test Runner

**File:** `Makefile` (update test targets)

```makefile
.PHONY: test integration test-layer2 test-layer3 test-performance

# Unit tests
test:
	go test -v -cover ./...

# All integration tests
integration: test-layer2 test-layer3 test-performance
	@echo "All integration tests complete"

# Layer 2 validation
test-layer2:
	@echo "Running Layer 2 (CO_CHANGED) validation..."
	@./test/integration/test_layer2_cochanged.sh

# Layer 3 validation
test-layer3:
	@echo "Running Layer 3 (CAUSED_BY) validation..."
	@./test/integration/test_layer3_causedby.sh

# Performance benchmarks
test-performance:
	@echo "Running performance benchmarks..."
	@./test/integration/test_performance_benchmarks.sh

# Full test suite (unit + integration)
test-all: test integration
	@echo "Full test suite complete"
```

---

## Testing Instructions

### Test 1: Run Individual Integration Tests

```bash
# Build first
make build

# Ensure Docker services running
docker compose up -d

# Test Layer 2
./test/integration/test_layer2_cochanged.sh
# Expected: ✓ PASS (after Gap A1 fix)

# Test Layer 3
./test/integration/test_layer3_causedby.sh
# Expected: ✓ PASS (after Gap B1 fix)

# Performance benchmarks
./test/integration/test_performance_benchmarks.sh
# Expected: All < target times
```

### Test 2: Run Full Integration Suite

```bash
make integration
# Expected: All tests pass
```

### Test 3: Verify --version Flag

```bash
make build
./crisk --version

# Expected output:
# CodeRisk v1.0.0 (or current version)
# Build time: 2025-10-06_12:34:56
# Git commit: abc123def
```

### Test 4: CI/CD Integration

Add to GitHub Actions workflow:

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  integration-tests:
    runs-on: ubuntu-latest
    services:
      neo4j:
        image: neo4j:5
        env:
          NEO4J_AUTH: neo4j/CHANGE_THIS_PASSWORD_IN_PRODUCTION_123
        ports:
          - 7687:7687
          - 7474:7474

      postgres:
        image: postgres:15
        env:
          POSTGRES_DB: coderisk
          POSTGRES_USER: coderisk
          POSTGRES_PASSWORD: coderisk123
        ports:
          - 5432:5432

    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Build
        run: make build

      - name: Run integration tests
        run: make integration
```

---

## Validation Criteria

**Success Criteria:**
- [ ] ✅ Layer 2 test passes (CO_CHANGED edges validated)
- [ ] ✅ Layer 3 test passes (CAUSED_BY edges validated)
- [ ] ✅ Performance benchmarks pass (<20ms, <50ms)
- [ ] ✅ `--version` flag works
- [ ] ✅ NULL handling fixed in incident search
- [ ] ✅ All tests automated and repeatable
- [ ] ✅ CI/CD integration added

**Code Quality:**
- [ ] ✅ Tests follow existing integration test patterns
- [ ] ✅ Clear pass/fail output with colors
- [ ] ✅ Debug information provided on failures
- [ ] ✅ Tests clean up after themselves

---

## Commit Message Template

```
Add integration tests for Layers 2 & 3 validation

Implements automated validation for CO_CHANGED and CAUSED_BY edge creation.
Adds performance benchmarks and fixes minor CLI bugs.

**Integration Tests:**
- test_layer2_cochanged.sh: Validates CO_CHANGED edges post-init
- test_layer3_causedby.sh: Validates CAUSED_BY edges on incident link
- test_performance_benchmarks.sh: Query performance validation

**CLI Fixes:**
- Add --version flag with build info
- Fix NULL handling in incident search (sql.NullString)

**Makefile Updates:**
- Add integration test targets (test-layer2, test-layer3, test-performance)
- Add version info to build (LDFLAGS)

Fixes: Testing gaps from E2E report
Tests: All integration tests pass
Performance: <20ms co-change, <50ms incident search

- test/integration/test_layer2_cochanged.sh: NEW
- test/integration/test_layer3_causedby.sh: NEW
- test/integration/test_performance_benchmarks.sh: NEW
- cmd/crisk/main.go: Add --version flag
- internal/incidents/search.go: Fix NULL handling
- Makefile: Add integration targets
```

---

## Success Validation

After implementation, all tests should pass:

```bash
# Full test suite
make test-all

# Expected output:
# ✓ Unit tests: 45/45 passed
# ✓ Layer 2 CO_CHANGED test: PASS
# ✓ Layer 3 CAUSED_BY test: PASS
# ✓ Performance benchmarks: PASS
# Full test suite complete
```

**This task is COMPLETE when:**
- All integration tests pass consistently ✅
- Performance targets met ✅
- CLI bugs fixed ✅
- CI/CD integration working ✅
