# Session 6: Risk Calculation & Validation

**Created:** October 4, 2025
**Purpose:** Validate and enhance Phase 1 risk calculation, wire into formatters
**Estimated Duration:** 1-2 days
**Dependencies:** None (can start immediately)

---

## Overview

You are **validating and enhancing** the Phase 1 risk calculation engine, ensuring accurate risk levels (LOW/MEDIUM/HIGH) based on coupling metrics, temporal co-change, and test coverage.

**Your Goal:** Ensure `crisk check` returns accurate, actionable risk assessments that developers trust, with correct integration across all verbosity levels.

**Key Principle:** Risk calculation must be **defensible** - every HIGH risk should be backed by clear metric violations that developers agree represent real risk.

---

## File Ownership

### You Own (Create/Modify Fully)

**Risk Calculation:**
- `internal/risk/phase1.go` - Validate/enhance Phase 1 metric calculations
- `internal/risk/scoring.go` - Validate/enhance risk level thresholds
- `internal/risk/calculator_test.go` - Comprehensive unit tests (create new)

**Tests:**
- `test/integration/test_check_e2e.sh` - End-to-end check command test (create new)
- `test/fixtures/known_risk/` - Test files with known risk levels (create new)

### You May Modify (Minor Changes Only)

**CLI Integration:**
- `cmd/crisk/check.go` - Wire risk calculation to formatters (~20-30 lines)

### You Read Only (No Modifications)

**Existing Infrastructure:**
- `internal/models/models.go` - RiskResult, FileRisk, Metric structs
- `internal/output/quiet.go` - Quiet formatter (Session 2)
- `internal/output/standard.go` - Standard formatter (Session 2)
- `internal/output/explain.go` - Explain formatter (Session 2)
- `internal/output/ai_mode.go` - AI Mode formatter (Session 3)
- `internal/graph/query.go` - Neo4j queries for metrics

**Reference Documentation:**
- `dev_docs/01-architecture/risk_assessment_methodology.md` - Your specification
- `dev_docs/03-implementation/integration_guides/risk_calculation.md` - Implementation guide
- `dev_docs/03-implementation/NEXT_STEPS.md` - Task 3 details

---

## Implementation Plan

### Step 1: Read and Understand (1 hour)

**Read these files in order:**

1. `dev_docs/01-architecture/risk_assessment_methodology.md`
   - Focus on: Phase 1 metrics (coupling, co-change, test ratio)
   - Understand: Threshold definitions (LOW/MEDIUM/HIGH)
   - Understand: Metric weightings and composite score

2. `dev_docs/03-implementation/integration_guides/risk_calculation.md`
   - Focus on: Implementation guide for Phase 1 metrics
   - Understand: Graph queries for coupling and co-change
   - Understand: Test coverage calculation

3. `internal/models/models.go`
   - Read: `RiskResult`, `FileRisk`, `Metric` struct definitions
   - Understand: Data structures for risk output

4. `internal/graph/query.go`
   - Understand: Available Neo4j queries for metrics
   - Understand: How to query coupling and co-change data

**Ask yourself:**
- What are the exact threshold values for each metric?
- How do we combine multiple metrics into a single risk level?
- What happens when metrics conflict (e.g., high coupling but great tests)?

---

### Step 2: Validate Existing Risk Logic (2-3 hours)

**Goal:** Ensure existing risk calculation matches specification.

#### 2.1: Review `internal/risk/phase1.go`

**Check if this file exists:**
```bash
ls -l internal/risk/phase1.go
```

**If it exists:**
- Read the implementation
- Verify metric calculations match spec
- Identify any bugs or deviations from spec

**If it doesn't exist:**
- You need to create it from scratch (see Step 3)

**What to validate:**

1. **Coupling Metric:**
   - Formula: `coupling_score = (direct_dependencies + indirect_dependencies) / total_files`
   - Thresholds:
     - LOW: <0.1 (file depends on <10% of codebase)
     - MEDIUM: 0.1-0.3
     - HIGH: >0.3

2. **Temporal Co-change Metric:**
   - Formula: `co_change_score = files_changed_together / total_commits_for_file`
   - Thresholds:
     - LOW: <0.2 (rarely changes with other files)
     - MEDIUM: 0.2-0.5
     - HIGH: >0.5

3. **Test Coverage Ratio:**
   - Formula: `test_ratio = test_lines / (source_lines + test_lines)`
   - Thresholds:
     - HIGH RISK: <0.2 (less than 20% test code)
     - MEDIUM RISK: 0.2-0.5
     - LOW RISK: >0.5 (more test code than source)

**Human Checkpoint: Validation Complete**
- **When:** After reviewing existing implementation
- **You Ask:** "✅ Reviewed `internal/risk/phase1.go`. Found [X issues / no issues]. Should I proceed with enhancements?"
- **If issues found:** List the issues and proposed fixes

---

### Step 3: Implement/Enhance Phase 1 Metrics (4-5 hours)

**File:** `internal/risk/phase1.go`

**Implementation Template:**

```go
package risk

import (
    "context"
    "fmt"
    "github.com/anthropics/coderisk-go/internal/graph"
    "github.com/anthropics/coderisk-go/internal/models"
)

// CalculatePhase1Risk computes risk based on Phase 1 metrics:
// - Coupling (direct + indirect dependencies)
// - Temporal Co-change (files changed together)
// - Test Coverage Ratio (test lines / total lines)
func CalculatePhase1Risk(ctx context.Context, filePath string, graphClient *graph.Client) (*models.FileRisk, error) {
    // Metric 1: Coupling
    coupling, err := calculateCouplingScore(ctx, filePath, graphClient)
    if err != nil {
        return nil, fmt.Errorf("coupling calculation failed: %w", err)
    }

    // Metric 2: Temporal Co-change
    coChange, err := calculateCoChangeScore(ctx, filePath, graphClient)
    if err != nil {
        return nil, fmt.Errorf("co-change calculation failed: %w", err)
    }

    // Metric 3: Test Coverage Ratio
    testRatio, err := calculateTestRatio(ctx, filePath, graphClient)
    if err != nil {
        return nil, fmt.Errorf("test ratio calculation failed: %w", err)
    }

    // Combine metrics into overall risk level
    riskLevel := determineRiskLevel(coupling, coChange, testRatio)

    return &models.FileRisk{
        FilePath: filePath,
        Level:    riskLevel,
        Metrics: []models.Metric{
            {Name: "coupling", Value: coupling, Threshold: 0.3},
            {Name: "co_change", Value: coChange, Threshold: 0.5},
            {Name: "test_ratio", Value: testRatio, Threshold: 0.2},
        },
    }, nil
}

// calculateCouplingScore computes coupling as:
// (direct_dependencies + indirect_dependencies) / total_files
func calculateCouplingScore(ctx context.Context, filePath string, graphClient *graph.Client) (float64, error) {
    // Query Neo4j for direct dependencies
    directDeps, err := graphClient.GetDirectDependencies(ctx, filePath)
    if err != nil {
        return 0, err
    }

    // Query Neo4j for indirect dependencies (2 hops)
    indirectDeps, err := graphClient.GetIndirectDependencies(ctx, filePath, 2)
    if err != nil {
        return 0, err
    }

    // Query total file count
    totalFiles, err := graphClient.GetTotalFileCount(ctx)
    if err != nil {
        return 0, err
    }

    if totalFiles == 0 {
        return 0, nil
    }

    coupling := float64(len(directDeps)+len(indirectDeps)) / float64(totalFiles)
    return coupling, nil
}

// calculateCoChangeScore computes temporal co-change as:
// files_changed_together / total_commits_for_file
func calculateCoChangeScore(ctx context.Context, filePath string, graphClient *graph.Client) (float64, error) {
    // Query commit history for this file
    commits, err := graphClient.GetFileCommits(ctx, filePath)
    if err != nil {
        return 0, err
    }

    if len(commits) == 0 {
        return 0, nil
    }

    // Count commits where >3 files changed together
    coChangeCount := 0
    for _, commit := range commits {
        if commit.FilesChanged > 3 {
            coChangeCount++
        }
    }

    coChange := float64(coChangeCount) / float64(len(commits))
    return coChange, nil
}

// calculateTestRatio computes test coverage as:
// test_lines / (source_lines + test_lines)
func calculateTestRatio(ctx context.Context, filePath string, graphClient *graph.Client) (float64, error) {
    // Get source file stats
    sourceStats, err := graphClient.GetFileStats(ctx, filePath)
    if err != nil {
        return 0, err
    }

    // Find corresponding test file
    testFilePath := convertToTestPath(filePath)
    testStats, err := graphClient.GetFileStats(ctx, testFilePath)
    if err != nil {
        // Test file doesn't exist -> ratio = 0
        return 0, nil
    }

    totalLines := sourceStats.LineCount + testStats.LineCount
    if totalLines == 0 {
        return 0, nil
    }

    testRatio := float64(testStats.LineCount) / float64(totalLines)
    return testRatio, nil
}

// convertToTestPath converts source file path to test file path
// e.g., "pkg/foo/bar.go" -> "pkg/foo/bar_test.go"
func convertToTestPath(sourcePath string) string {
    // Implementation depends on language
    // For Go: replace .go with _test.go
    // For Python: replace .py with _test.py or find test_*.py
    // TODO: Support multiple languages
    return strings.Replace(sourcePath, ".go", "_test.go", 1)
}

// determineRiskLevel combines metric scores into overall risk level
func determineRiskLevel(coupling, coChange, testRatio float64) string {
    // High risk if ANY metric exceeds threshold
    if coupling > 0.3 || coChange > 0.5 || testRatio < 0.2 {
        return "HIGH"
    }

    // Medium risk if any metric is borderline
    if coupling > 0.1 || coChange > 0.2 || testRatio < 0.5 {
        return "MEDIUM"
    }

    return "LOW"
}
```

**Key Implementation Notes:**

1. **Graph Client Methods:**
   - Check if these methods exist in `internal/graph/query.go`
   - If not, you may need to implement simple Neo4j queries
   - Example: `GetDirectDependencies()` should query `MATCH (f:File {path: $path})-[:IMPORTS]->(dep) RETURN dep`

2. **Test Path Conversion:**
   - Start with Go convention (`_test.go`)
   - Future: Support Python (`test_*.py`), JavaScript (`*.spec.js`), etc.

3. **Risk Level Logic:**
   - Current: "Fail on any HIGH threshold" (conservative)
   - Alternative: Weighted average (more nuanced)
   - **Choose conservative approach** for Phase 1 (safer to over-warn)

**Human Checkpoint: Phase 1 Implementation**
- **When:** After implementing all 3 metric calculations
- **You Ask:** "✅ Phase 1 metrics implemented (coupling, co-change, test ratio). Should I test with sample files?"
- **Test Command:** `go run ./cmd/crisk check <sample-file>`

---

### Step 4: Create Test Fixtures (2 hours)

**Goal:** Create files with **known risk levels** to validate calculation accuracy.

**Directory:** `test/fixtures/known_risk/`

#### 4.1: Low Risk File

**File:** `test/fixtures/known_risk/low_risk.go`

```go
package fixtures

// LowRiskFunction has minimal dependencies, great test coverage
func LowRiskFunction(x int) int {
    return x * 2
}
```

**Corresponding Test:** `test/fixtures/known_risk/low_risk_test.go`

```go
package fixtures

import "testing"

func TestLowRiskFunction(t *testing.T) {
    if result := LowRiskFunction(5); result != 10 {
        t.Errorf("Expected 10, got %d", result)
    }
}

func TestLowRiskFunctionZero(t *testing.T) {
    if result := LowRiskFunction(0); result != 0 {
        t.Errorf("Expected 0, got %d", result)
    }
}

func TestLowRiskFunctionNegative(t *testing.T) {
    if result := LowRiskFunction(-3); result != -6 {
        t.Errorf("Expected -6, got %d", result)
    }
}
```

**Expected Risk:** **LOW** (test ratio ~0.75, minimal coupling, no co-change)

#### 4.2: Medium Risk File

**File:** `test/fixtures/known_risk/medium_risk.go`

```go
package fixtures

import (
    "github.com/anthropics/coderisk-go/internal/graph"
    "github.com/anthropics/coderisk-go/internal/models"
)

// MediumRiskFunction has moderate coupling, some tests
func MediumRiskFunction(g *graph.Client) (*models.FileRisk, error) {
    // Couples to 2 packages (graph, models)
    // Has tests but not comprehensive
    return &models.FileRisk{Level: "LOW"}, nil
}
```

**Corresponding Test:** `test/fixtures/known_risk/medium_risk_test.go`

```go
package fixtures

import "testing"

func TestMediumRiskFunction(t *testing.T) {
    // Only 1 test for a more complex function
    result, err := MediumRiskFunction(nil)
    if err != nil {
        t.Errorf("Unexpected error: %v", err)
    }
    if result == nil {
        t.Error("Expected non-nil result")
    }
}
```

**Expected Risk:** **MEDIUM** (coupling ~0.15, test ratio ~0.4)

#### 4.3: High Risk File

**File:** `test/fixtures/known_risk/high_risk.go`

```go
package fixtures

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "os"
    "strings"

    "github.com/anthropics/coderisk-go/internal/graph"
    "github.com/anthropics/coderisk-go/internal/github"
    "github.com/anthropics/coderisk-go/internal/ingestion"
    "github.com/anthropics/coderisk-go/internal/models"
    "github.com/anthropics/coderisk-go/internal/output"
    "github.com/anthropics/coderisk-go/internal/risk"
)

// HighRiskFunction has high coupling, NO tests, complex logic
func HighRiskFunction(ctx context.Context, db *sql.DB, apiURL string) error {
    // Couples to 10+ packages
    // No corresponding test file
    // Complex logic with network calls, database operations

    resp, err := http.Get(apiURL)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)
    var data map[string]interface{}
    json.Unmarshal(body, &data)

    // Simulate database write
    _, err = db.ExecContext(ctx, "INSERT INTO data VALUES (?)", data)
    return err
}
```

**NO CORRESPONDING TEST FILE** (intentional)

**Expected Risk:** **HIGH** (coupling >0.3, test ratio = 0, network + DB calls)

---

### Step 5: Create Unit Tests (3-4 hours)

**File:** `internal/risk/calculator_test.go`

**Test Coverage Target:** 80%+

**Test Template:**

```go
package risk

import (
    "context"
    "testing"

    "github.com/anthropics/coderisk-go/internal/graph"
    "github.com/stretchr/testify/assert"
)

func TestCalculateCouplingScore(t *testing.T) {
    ctx := context.Background()

    // Mock graph client
    mockGraph := &graph.MockClient{
        DirectDeps:   []string{"pkg/a.go", "pkg/b.go"},
        IndirectDeps: []string{"pkg/c.go"},
        TotalFiles:   100,
    }

    score, err := calculateCouplingScore(ctx, "test.go", mockGraph)
    assert.NoError(t, err)
    assert.Equal(t, 0.03, score) // (2 + 1) / 100 = 0.03
}

func TestCalculateCoChangeScore(t *testing.T) {
    ctx := context.Background()

    mockGraph := &graph.MockClient{
        Commits: []graph.Commit{
            {FilesChanged: 5}, // Co-change
            {FilesChanged: 2}, // Not co-change
            {FilesChanged: 10}, // Co-change
        },
    }

    score, err := calculateCoChangeScore(ctx, "test.go", mockGraph)
    assert.NoError(t, err)
    assert.Equal(t, 0.67, score, 0.01) // 2/3 commits had >3 files
}

func TestCalculateTestRatio(t *testing.T) {
    ctx := context.Background()

    mockGraph := &graph.MockClient{
        FileStats: map[string]graph.FileStats{
            "pkg/foo.go":      {LineCount: 100},
            "pkg/foo_test.go": {LineCount: 150},
        },
    }

    score, err := calculateTestRatio(ctx, "pkg/foo.go", mockGraph)
    assert.NoError(t, err)
    assert.Equal(t, 0.6, score) // 150 / (100 + 150) = 0.6
}

func TestDetermineRiskLevel(t *testing.T) {
    tests := []struct {
        name      string
        coupling  float64
        coChange  float64
        testRatio float64
        expected  string
    }{
        {"Low risk", 0.05, 0.1, 0.7, "LOW"},
        {"Medium coupling", 0.2, 0.1, 0.6, "MEDIUM"},
        {"High coupling", 0.4, 0.1, 0.6, "HIGH"},
        {"Low test coverage", 0.05, 0.1, 0.1, "HIGH"},
        {"High co-change", 0.05, 0.6, 0.6, "HIGH"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := determineRiskLevel(tt.coupling, tt.coChange, tt.testRatio)
            assert.Equal(t, tt.expected, result)
        })
    }
}

func TestCalculatePhase1Risk_KnownFiles(t *testing.T) {
    ctx := context.Background()
    graphClient := setupTestGraphClient(t)

    // Test low risk file
    lowRisk, err := CalculatePhase1Risk(ctx, "test/fixtures/known_risk/low_risk.go", graphClient)
    assert.NoError(t, err)
    assert.Equal(t, "LOW", lowRisk.Level)

    // Test medium risk file
    mediumRisk, err := CalculatePhase1Risk(ctx, "test/fixtures/known_risk/medium_risk.go", graphClient)
    assert.NoError(t, err)
    assert.Equal(t, "MEDIUM", mediumRisk.Level)

    // Test high risk file
    highRisk, err := CalculatePhase1Risk(ctx, "test/fixtures/known_risk/high_risk.go", graphClient)
    assert.NoError(t, err)
    assert.Equal(t, "HIGH", highRisk.Level)
}
```

**Run Tests:**
```bash
go test ./internal/risk/... -v -cover
```

**Expected Coverage:** >80%

**Human Checkpoint: Unit Tests Complete**
- **When:** After implementing all unit tests
- **You Ask:** "✅ Unit tests implemented. Coverage: [X]%. All tests passing. Should I proceed with integration tests?"

---

### Step 6: Wire Risk Calculation into `crisk check` (2 hours)

**File:** `cmd/crisk/check.go`

**Goal:** Connect risk calculation to all formatters (quiet, standard, explain, AI mode).

**Implementation:**

```go
// In cmd/crisk/check.go

func runCheck(cmd *cobra.Command, args []string) error {
    ctx := cmd.Context()

    // Get file path
    filePath := args[0]

    // Initialize graph client
    graphClient, err := graph.NewClient(ctx)
    if err != nil {
        return fmt.Errorf("failed to connect to graph: %w", err)
    }
    defer graphClient.Close()

    // Calculate risk (Phase 1)
    fileRisk, err := risk.CalculatePhase1Risk(ctx, filePath, graphClient)
    if err != nil {
        return fmt.Errorf("risk calculation failed: %w", err)
    }

    // Determine output format (based on flags)
    var formatter output.Formatter
    if cmd.Flags().Lookup("ai-mode").Changed {
        formatter = output.NewAIModeFormatter()
    } else if cmd.Flags().Lookup("explain").Changed {
        formatter = output.NewExplainFormatter()
    } else if cmd.Flags().Lookup("quiet").Changed {
        formatter = output.NewQuietFormatter()
    } else {
        formatter = output.NewStandardFormatter()
    }

    // Format and print result
    result := &models.RiskResult{
        Files: []models.FileRisk{*fileRisk},
    }
    output := formatter.Format(result)
    fmt.Println(output)

    // Exit with appropriate code
    if fileRisk.Level == "HIGH" {
        return fmt.Errorf("HIGH risk detected")
    }

    return nil
}
```

**Key Changes:**
- Import `internal/risk` package
- Call `risk.CalculatePhase1Risk()`
- Pass result to existing formatters (from Session 2 & 3)
- Return error if HIGH risk (for pre-commit hook blocking)

**Human Checkpoint: CLI Integration**
- **When:** After wiring risk calculation into check command
- **You Ask:** "✅ Risk calculation wired into `crisk check`. Should I test all verbosity levels?"
- **Test Commands:**
```bash
./bin/crisk check --quiet test/fixtures/known_risk/low_risk.go
./bin/crisk check test/fixtures/known_risk/medium_risk.go
./bin/crisk check --explain test/fixtures/known_risk/high_risk.go
./bin/crisk check --ai-mode test/fixtures/known_risk/high_risk.go
```

---

### Step 7: Create End-to-End Integration Test (2 hours)

**File:** `test/integration/test_check_e2e.sh`

**Template:**

```bash
#!/bin/bash
set -e

echo "=== CodeRisk Check Command E2E Test ==="

# Ensure binary exists
if [ ! -f ./bin/crisk ]; then
    echo "Building binary..."
    go build -o bin/crisk ./cmd/crisk
fi

# Test 1: Low risk file (quiet mode)
echo "Test 1: Low risk file (quiet mode)"
OUTPUT=$(./bin/crisk check --quiet test/fixtures/known_risk/low_risk.go 2>/dev/null || true)
if echo "$OUTPUT" | grep -q "LOW risk"; then
    echo "✅ PASS: Low risk detected"
else
    echo "❌ FAIL: Expected LOW risk"
    echo "Output: $OUTPUT"
    exit 1
fi

# Test 2: Medium risk file (standard mode)
echo "Test 2: Medium risk file (standard mode)"
OUTPUT=$(./bin/crisk check test/fixtures/known_risk/medium_risk.go 2>/dev/null || true)
if echo "$OUTPUT" | grep -q "MEDIUM risk"; then
    echo "✅ PASS: Medium risk detected"
else
    echo "❌ FAIL: Expected MEDIUM risk"
    exit 1
fi

# Test 3: High risk file (explain mode)
echo "Test 3: High risk file (explain mode)"
OUTPUT=$(./bin/crisk check --explain test/fixtures/known_risk/high_risk.go 2>/dev/null || true)
if echo "$OUTPUT" | grep -q "HIGH risk"; then
    echo "✅ PASS: High risk detected"
else
    echo "❌ FAIL: Expected HIGH risk"
    exit 1
fi

# Verify explanation includes metrics
if echo "$OUTPUT" | grep -q "coupling" && echo "$OUTPUT" | grep -q "test_ratio"; then
    echo "✅ PASS: Explanation includes metrics"
else
    echo "❌ FAIL: Explanation missing metrics"
    exit 1
fi

# Test 4: High risk file (AI mode)
echo "Test 4: High risk file (AI mode)"
OUTPUT=$(./bin/crisk check --ai-mode test/fixtures/known_risk/high_risk.go 2>/dev/null || true)

# Validate JSON
if echo "$OUTPUT" | jq -e '.risk.level == "HIGH"' >/dev/null 2>&1; then
    echo "✅ PASS: AI Mode JSON valid, risk level correct"
else
    echo "❌ FAIL: AI Mode JSON invalid or risk level incorrect"
    exit 1
fi

# Verify AI actions array exists
if echo "$OUTPUT" | jq -e '.ai_assistant_actions | length > 0' >/dev/null 2>&1; then
    echo "✅ PASS: AI actions array present"
else
    echo "❌ FAIL: AI actions array missing"
    exit 1
fi

# Test 5: Exit code for HIGH risk
echo "Test 5: Exit code for HIGH risk"
./bin/crisk check test/fixtures/known_risk/high_risk.go 2>/dev/null || EXIT_CODE=$?
if [ $EXIT_CODE -ne 0 ]; then
    echo "✅ PASS: Non-zero exit code for HIGH risk"
else
    echo "❌ FAIL: Should return non-zero exit code for HIGH risk"
    exit 1
fi

echo "=== All tests passed ==="
```

**Run Test:**
```bash
chmod +x test/integration/test_check_e2e.sh
./test/integration/test_check_e2e.sh
```

---

### Step 8: Validate Risk Accuracy (2-3 hours)

**Goal:** Ensure risk levels match developer intuition.

#### 8.1: Test with Real Files

**Pick 5-10 files from `internal/` and validate risk:**

```bash
./bin/crisk check --explain internal/graph/builder.go
./bin/crisk check --explain internal/github/client.go
./bin/crisk check --explain internal/ingestion/clone.go
./bin/crisk check --explain cmd/crisk/main.go
./bin/crisk check --explain internal/models/models.go
```

**For each file, ask:**
1. Does the risk level match your intuition?
2. Are the metric values reasonable?
3. Are the thresholds too strict or too lenient?

#### 8.2: Adjust Thresholds (if needed)

**If risk levels don't match intuition:**

**Edit:** `internal/risk/scoring.go`

**Adjust thresholds:**
```go
// Original thresholds
const (
    CouplingHighThreshold  = 0.3
    CouplingMediumThreshold = 0.1
    CoChangeHighThreshold  = 0.5
    CoChangeMediumThreshold = 0.2
    TestRatioHighThreshold = 0.2  // Below this = HIGH risk
    TestRatioMediumThreshold = 0.5
)

// Adjusted thresholds (if needed)
const (
    CouplingHighThreshold  = 0.4  // Less strict (fewer false positives)
    CouplingMediumThreshold = 0.15
    // ... etc
)
```

**Re-run tests after adjustments:**
```bash
./test/integration/test_check_e2e.sh
```

**Human Checkpoint: Risk Validation**
- **When:** After testing with real files and adjusting thresholds
- **You Ask:** "✅ Risk validation complete. Tested [X] files. Thresholds adjusted: [Y/N]. Risk levels accurate: [Y/N]. Should I proceed?"

---

## Critical Checkpoints

### Checkpoint 1: Phase 1 Metrics Implemented

**Trigger:** Step 3 complete (all metric calculations implemented)

**Verification:**
```bash
go test ./internal/risk/... -v

# Expected: All tests pass
```

**YOU ASK:** "✅ Phase 1 metrics implemented (coupling, co-change, test ratio). All unit tests passing. Should I proceed with integration?"

---

### Checkpoint 2: Integration with Formatters Complete

**Trigger:** Step 6 complete (risk wired into `crisk check`)

**Verification:**
```bash
./bin/crisk check --quiet test/fixtures/known_risk/low_risk.go
./bin/crisk check --explain test/fixtures/known_risk/high_risk.go
./bin/crisk check --ai-mode test/fixtures/known_risk/high_risk.go | jq '.risk.level'

# Expected: All verbosity levels show correct risk
```

**YOU ASK:** "✅ Risk calculation integrated with all formatters. Tested quiet, standard, explain, AI mode. All working. Should I proceed with validation?"

---

### Checkpoint 3: Risk Validation Complete

**Trigger:** Step 8 complete (tested with real files, thresholds validated)

**Verification:**
```bash
./test/integration/test_check_e2e.sh

# Expected: All tests pass
```

**YOU ASK:** "✅ Risk validation complete. Integration tests: [X/X pass]. Risk levels accurate on real files. Ready to mark Session 6 complete?"

---

## Success Criteria

### Functional Requirements
- [ ] `crisk check` returns accurate risk levels (LOW/MEDIUM/HIGH)
- [ ] All 3 Phase 1 metrics implemented (coupling, co-change, test ratio)
- [ ] Risk calculation works with all verbosity levels (quiet, standard, explain, AI mode)
- [ ] Test fixtures with known risk levels pass validation
- [ ] HIGH risk files return non-zero exit code (for pre-commit hook)

### Accuracy Requirements
- [ ] Low risk files: <5% false positives (incorrectly flagged as MEDIUM/HIGH)
- [ ] High risk files: <5% false negatives (missed HIGH risk)
- [ ] Metric values match manual calculation (spot check 5 files)
- [ ] Thresholds validated against developer intuition

### Quality Requirements
- [ ] 80%+ unit test coverage for risk calculation
- [ ] Integration test passes for all risk levels
- [ ] Error handling for missing graph data
- [ ] Clear metric explanations in --explain mode

---

## What Could Go Wrong

### Issue: False positives (too many HIGH risk warnings)
**Prevention:** Conservative thresholds, validate with real files
**Recovery:** Adjust thresholds in `scoring.go`, re-test

### Issue: Neo4j queries too slow (>2s per file)
**Prevention:** Add indexes on `:File` nodes, limit query depth
**Recovery:** Cache metric calculations, add `--use-cache` flag

### Issue: Test ratio calculation fails for non-Go files
**Prevention:** Implement language-specific test path detection
**Recovery:** Return neutral score (0.5) if test file detection fails

### Issue: Coupling calculation includes test files (inflates score)
**Prevention:** Exclude `*_test.go` files from dependency graph
**Recovery:** Filter test files in graph query

### Issue: Risk levels don't match developer intuition
**Prevention:** Test with 10+ real files, get feedback
**Recovery:** Adjust thresholds, document rationale in risk_assessment_methodology.md

---

## Quick Commands for Verification

**Build:**
```bash
go build -o bin/crisk ./cmd/crisk
```

**Test risk calculation:**
```bash
./bin/crisk check --explain test/fixtures/known_risk/high_risk.go
```

**Run unit tests:**
```bash
go test ./internal/risk/... -v -cover
```

**Run integration tests:**
```bash
./test/integration/test_check_e2e.sh
```

**Validate with real files:**
```bash
for file in internal/**/*.go; do
    echo "Checking $file..."
    ./bin/crisk check --quiet "$file"
done
```

---

## References

**Architecture:**
- [risk_assessment_methodology.md](../01-architecture/risk_assessment_methodology.md) - Your specification
- [agentic_design.md](../01-architecture/agentic_design.md) - Phase 1 vs Phase 2 distinction

**Implementation Guides:**
- [risk_calculation.md](integration_guides/risk_calculation.md) - Implementation details
- [graph_integration.md](integration_guides/graph_integration.md) - Neo4j queries

**Data Models:**
- [models.go](../../internal/models/models.go) - RiskResult, FileRisk, Metric structs

**Coordination:**
- [PARALLEL_SESSION_PLAN_WEEK1.md](PARALLEL_SESSION_PLAN_WEEK1.md) - Overall plan
- [SESSION_2_PROMPT.md](SESSION_2_PROMPT.md) - Formatters (your output target)
- [SESSION_3_PROMPT.md](SESSION_3_PROMPT.md) - AI Mode (your JSON target)

**Next Steps:**
- [NEXT_STEPS.md](NEXT_STEPS.md) - Week 1 Task 3 (this session)

---

## After Session Complete

**Update documentation:**
- Mark Week 1 Task 3 complete in [NEXT_STEPS.md](NEXT_STEPS.md)
- Add entry to [IMPLEMENTATION_LOG.md](IMPLEMENTATION_LOG.md)
- Update [status.md](status.md) with risk calculation completion

**Coordinate with other sessions:**
- All Week 1 tasks complete → Ready for final integration (Checkpoint 4)

---

**Created:** October 4, 2025
**Status:** Ready to execute immediately
**Estimated Duration:** 1-2 days
**Owner:** Session 6
