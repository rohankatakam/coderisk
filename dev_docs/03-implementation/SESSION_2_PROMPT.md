# Session 2 Prompt: Adaptive Verbosity System (Levels 1-3)

**Duration:** 3-4 days
**Owner:** Claude Code Session 2
**Dependencies:** None (can start immediately, creates interface for other sessions)

---

## Context

You are implementing the **Adaptive Verbosity System** component of CodeRisk's Developer Experience (DX) Foundation phase. This is **Session 2 of 3 parallel sessions** working on different parts of the codebase simultaneously.

**Your role:** Create the output formatting system with 4 verbosity levels (you implement levels 1-3, Session 3 implements level 4).

**Other sessions (DO NOT MODIFY THEIR FILES):**
- Session 1: Building pre-commit hook in `cmd/crisk/hook.go` and `internal/git/`
- Session 3: Building AI Mode (Level 4) in `internal/output/ai_mode.go` and `internal/ai/`

**CRITICAL:** You create `internal/output/formatter.go` interface that **both other sessions depend on**. Create this file FIRST.

---

## High-Level Goal

Implement progressive disclosure output system:
1. **Level 1 (Quiet):** One-line summary for pre-commit hooks (`--quiet`)
2. **Level 2 (Standard):** Issues + recommendations (default CLI)
3. **Level 3 (Explain):** Full investigation trace with hop-by-hop reasoning (`--explain`)
4. **Level 4 (AI Mode):** JSON for AI assistants (`--ai-mode`) - **Session 3 implements this**

---

## Your File Ownership

### Files YOU CREATE (your responsibility):
- `internal/output/formatter.go` - **CREATE THIS FIRST** (interface for all sessions)
- `internal/output/quiet.go` - Level 1 formatter
- `internal/output/standard.go` - Level 2 formatter
- `internal/output/explain.go` - Level 3 formatter
- `internal/output/formatter_test.go` - Unit tests
- `internal/config/verbosity.go` - Verbosity detection logic
- `test/integration/test_verbosity.sh` - Integration tests

### Files YOU MODIFY (minimal changes):
- `cmd/crisk/check.go` - Add `--quiet` and `--explain` flags, wire formatters (~30 lines)

### Files YOU READ ONLY (do not modify):
- `internal/models/risk_result.go` - Existing model (read structure only)
- `internal/output/ai_mode.go` - Session 3 will create this (implements your interface)

---

## Reading List (READ THESE FIRST)

**MUST READ before coding:**
1. `dev_docs/03-implementation/integration_guides/ux_adaptive_verbosity.md` - Your implementation guide
2. `dev_docs/03-implementation/PARALLEL_SESSION_PLAN.md` - Coordination with other sessions
3. `dev_docs/00-product/developer_experience.md` - UX requirements (§2 Adaptive Verbosity)
4. `dev_docs/03-implementation/phases/phase_dx_foundation.md` - Phase overview

**Reference as needed:**
5. `dev_docs/DEVELOPMENT_WORKFLOW.md` - Go development guardrails
6. `internal/models/risk_result.go` - Understand the data model you'll format

---

## Step-by-Step Implementation Plan

### Step 1: Read Documentation (30 min)
- [ ] Read all files in "Reading List" section above
- [ ] Understand verbosity requirements from `ux_adaptive_verbosity.md`
- [ ] Understand coordination plan from `PARALLEL_SESSION_PLAN.md`
- [ ] Study `internal/models/risk_result.go` structure
- [ ] **ASK USER:** "I've read the documentation and studied the RiskResult model. Should I proceed with interface creation?"

---

### Step 2: Create Formatter Interface (1 hour) - **DO THIS FIRST!**

**File: `internal/output/formatter.go`** - **CRITICAL: Other sessions need this!**

```go
package output

import (
    "io"
    "github.com/coderisk/coderisk-go/internal/models"
)

// Formatter defines output formatting interface
type Formatter interface {
    Format(result *models.RiskResult, w io.Writer) error
}

// VerbosityLevel determines output detail
type VerbosityLevel int

const (
    VerbosityQuiet VerbosityLevel = iota   // Level 1: One-line summary
    VerbosityStandard                      // Level 2: Issues + recommendations
    VerbosityExplain                       // Level 3: Full investigation trace
    VerbosityAIMode                        // Level 4: Machine-readable JSON
)

// NewFormatter creates appropriate formatter based on level
func NewFormatter(level VerbosityLevel) Formatter {
    switch level {
    case VerbosityQuiet:
        return &QuietFormatter{}
    case VerbosityStandard:
        return &StandardFormatter{}
    case VerbosityExplain:
        return &ExplainFormatter{}
    case VerbosityAIMode:
        return &AIFormatter{} // Session 3 will implement this
    default:
        return &StandardFormatter{}
    }
}
```

**Checkpoint:** Build the interface
```bash
go build ./internal/output/
```

**ASK USER (CRITICAL CHECKPOINT):**
"✅ Formatter interface created at internal/output/formatter.go
- Interface compiles successfully
- VerbosityLevel enum has 4 levels (Quiet, Standard, Explain, AIMode)
- NewFormatter() factory is ready

**NOTIFY OTHER SESSIONS:** Sessions 1 and 3 can now import this interface.

Should I proceed with implementing Level 1 (Quiet) formatter?"

**WAIT FOR USER CONFIRMATION BEFORE PROCEEDING**

---

### Step 3: Implement Level 1 - Quiet Formatter (1-2 hours)

**File: `internal/output/quiet.go`**

**Requirements from ux_adaptive_verbosity.md:**
- One-line summary for pre-commit hooks
- Show risk level on success: `✅ LOW risk`
- Show issue count on failure: `⚠️ MEDIUM risk: 3 issues detected`
- Suggest next step: `Run 'crisk check' for details`

```go
package output

import (
    "fmt"
    "io"
    "github.com/coderisk/coderisk-go/internal/models"
)

// QuietFormatter outputs one-line summary (for pre-commit hooks)
type QuietFormatter struct{}

func (f *QuietFormatter) Format(result *models.RiskResult, w io.Writer) error {
    // Success case
    if result.RiskLevel == "LOW" || result.RiskLevel == "NONE" {
        fmt.Fprintf(w, "✅ %s risk\n", result.RiskLevel)
        return nil
    }

    // Risk detected case
    issueCount := len(result.Issues)
    fmt.Fprintf(w, "⚠️  %s risk: %d issues detected\n", result.RiskLevel, issueCount)
    fmt.Fprintf(w, "Run 'crisk check' for details\n")

    return nil
}
```

**Checkpoint:** Test quiet formatter
**ASK USER:** "Level 1 (Quiet) formatter implemented. Should I proceed with Level 2 (Standard)?"

---

### Step 4: Implement Level 2 - Standard Formatter (2-3 hours)

**File: `internal/output/standard.go`**

**Requirements:**
- Show analysis header (branch, files changed, risk level)
- List issues with severity icons
- Provide recommendations
- Suggest `crisk check --explain` for more details

```go
package output

import (
    "fmt"
    "io"
    "github.com/coderisk/coderisk-go/internal/models"
)

// StandardFormatter outputs issues + recommendations (default)
type StandardFormatter struct{}

func (f *StandardFormatter) Format(result *models.RiskResult, w io.Writer) error {
    // Header
    fmt.Fprintf(w, "🔍 CodeRisk Analysis\n")
    if result.Branch != "" {
        fmt.Fprintf(w, "Branch: %s\n", result.Branch)
    }
    fmt.Fprintf(w, "Files changed: %d\n", result.FilesChanged)
    fmt.Fprintf(w, "Risk level: %s\n\n", result.RiskLevel)

    // Issues
    if len(result.Issues) > 0 {
        fmt.Fprintf(w, "Issues:\n")
        for i, issue := range result.Issues {
            fmt.Fprintf(w, "%d. %s %s - %s\n",
                i+1,
                severityEmoji(issue.Severity),
                issue.File,
                issue.Message,
            )
        }
        fmt.Fprintf(w, "\n")
    }

    // Recommendations
    if len(result.Recommendations) > 0 {
        fmt.Fprintf(w, "Recommendations:\n")
        for _, rec := range result.Recommendations {
            fmt.Fprintf(w, "- %s\n", rec)
        }
        fmt.Fprintf(w, "\n")
    }

    // Next steps
    if result.RiskLevel != "LOW" && result.RiskLevel != "NONE" {
        fmt.Fprintf(w, "Run 'crisk check --explain' for investigation trace\n")
    }

    return nil
}

func severityEmoji(severity string) string {
    switch severity {
    case "HIGH", "CRITICAL":
        return "🔴"
    case "MEDIUM":
        return "⚠️ "
    case "LOW":
        return "ℹ️ "
    default:
        return "•"
    }
}
```

**Checkpoint:** Test standard formatter
**ASK USER:** "Level 2 (Standard) formatter implemented. Output includes header, issues, and recommendations. Should I proceed with Level 3 (Explain)?"

---

### Step 5: Implement Level 3 - Explain Formatter (3-4 hours)

**File: `internal/output/explain.go`**

**Requirements:**
- Show full investigation report header
- Display hop-by-hop investigation trace
- Show metrics calculated at each hop
- Show agent decisions and reasoning
- Display final assessment with evidence

```go
package output

import (
    "fmt"
    "io"
    "time"
    "github.com/coderisk/coderisk-go/internal/models"
)

// ExplainFormatter outputs full investigation trace
type ExplainFormatter struct{}

func (f *ExplainFormatter) Format(result *models.RiskResult, w io.Writer) error {
    // Header
    fmt.Fprintf(w, "🔍 CodeRisk Investigation Report\n")
    fmt.Fprintf(w, "Started: %s\n", result.StartTime.Format(time.RFC3339))
    fmt.Fprintf(w, "Completed: %s (%.1fs)\n",
        result.EndTime.Format(time.RFC3339),
        result.Duration.Seconds())
    fmt.Fprintf(w, "Agent hops: %d\n\n", len(result.InvestigationTrace))

    // Investigation trace (hop-by-hop)
    for i, hop := range result.InvestigationTrace {
        fmt.Fprintf(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
        fmt.Fprintf(w, "Hop %d: %s\n", i+1, hop.NodeID)
        fmt.Fprintf(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

        // Changed entities
        if len(hop.ChangedEntities) > 0 {
            fmt.Fprintf(w, "Changed functions:\n")
            for _, entity := range hop.ChangedEntities {
                fmt.Fprintf(w, "  - %s (lines %d-%d)\n",
                    entity.Name, entity.StartLine, entity.EndLine)
            }
            fmt.Fprintf(w, "\n")
        }

        // Metrics calculated
        if len(hop.Metrics) > 0 {
            fmt.Fprintf(w, "Metrics calculated:\n")
            for _, metric := range hop.Metrics {
                status := f.metricStatus(metric)
                fmt.Fprintf(w, "  %s %s: %.1f (target: <%.1f)\n",
                    status, metric.Name, metric.Value, *metric.Threshold)
            }
            fmt.Fprintf(w, "\n")
        }

        // Agent decision
        fmt.Fprintf(w, "Agent decision: %s\n", hop.Decision)
        if hop.Reasoning != "" {
            fmt.Fprintf(w, "Reasoning: %s\n", hop.Reasoning)
        }
        fmt.Fprintf(w, "\n")
    }

    // Final assessment
    fmt.Fprintf(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
    fmt.Fprintf(w, "Final Assessment\n")
    fmt.Fprintf(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
    fmt.Fprintf(w, "Risk Level: %s\n\n", result.RiskLevel)

    // Evidence
    if len(result.Evidence) > 0 {
        fmt.Fprintf(w, "Evidence:\n")
        for i, evidence := range result.Evidence {
            fmt.Fprintf(w, "  %d. %s\n", i+1, evidence)
        }
        fmt.Fprintf(w, "\n")
    }

    // Recommendations (prioritized)
    if len(result.Recommendations) > 0 {
        fmt.Fprintf(w, "Recommendations (priority order):\n")
        for i, rec := range result.Recommendations {
            fmt.Fprintf(w, "  %d. %s\n", i+1, rec)
        }
        fmt.Fprintf(w, "\n")
    }

    // Next steps
    if len(result.NextSteps) > 0 {
        fmt.Fprintf(w, "Suggested next steps:\n")
        for _, step := range result.NextSteps {
            fmt.Fprintf(w, "  → %s\n", step)
        }
    }

    return nil
}

func (f *ExplainFormatter) metricStatus(metric models.Metric) string {
    if metric.Threshold != nil && metric.Value > *metric.Threshold {
        return "❌"
    } else if metric.Warning != nil && metric.Value > *metric.Warning {
        return "⚠️ "
    }
    return "✅"
}
```

**Checkpoint:** Test explain formatter
**ASK USER:** "Level 3 (Explain) formatter implemented. Shows hop-by-hop trace with metrics and decisions. Should I proceed with verbosity detection logic?"

---

### Step 6: Implement Verbosity Detection (1-2 hours)

**File: `internal/config/verbosity.go`**

**Requirements:**
- Detect environment context (pre-commit hook, CI, AI assistant)
- Return appropriate default verbosity level

```go
package config

import (
    "os"
    "github.com/coderisk/coderisk-go/internal/output"
)

// GetDefaultVerbosity returns appropriate default based on environment
func GetDefaultVerbosity() output.VerbosityLevel {
    // Pre-commit hook context (GIT_AUTHOR_DATE set by git)
    if os.Getenv("GIT_AUTHOR_DATE") != "" {
        return output.VerbosityQuiet
    }

    // CI/CD context
    if os.Getenv("CI") == "true" {
        return output.VerbosityStandard
    }

    // AI assistant context (detected by special env var)
    if os.Getenv("CRISK_AI_MODE") == "1" {
        return output.VerbosityAIMode
    }

    // Interactive terminal (default)
    return output.VerbosityStandard
}
```

**Checkpoint:** Test verbosity detection
**ASK USER:** "Verbosity detection implemented. Defaults to quiet in git hooks, standard in CLI. Should I proceed with CLI integration?"

---

### Step 7: Integrate with Check Command (1-2 hours)

**File: `cmd/crisk/check.go` (MINIMAL CHANGES)**

Add flags and wire formatters:

```go
func init() {
    checkCmd.Flags().Bool("quiet", false, "Output one-line summary (for pre-commit hooks)")
    checkCmd.Flags().Bool("explain", false, "Show full investigation trace")
    // Note: --ai-mode flag will be added by Session 3

    // Mutually exclusive flags
    checkCmd.MarkFlagsMutuallyExclusive("quiet", "explain")
}

func runCheck(cmd *cobra.Command, args []string) error {
    // ... existing file detection logic ...

    // Run risk assessment
    result, err := runRiskAssessment(cmd.Context(), files)
    if err != nil {
        return err
    }

    // Determine verbosity level
    var level output.VerbosityLevel
    quiet, _ := cmd.Flags().GetBool("quiet")
    explain, _ := cmd.Flags().GetBool("explain")

    if quiet {
        level = output.VerbosityQuiet
    } else if explain {
        level = output.VerbosityExplain
    } else {
        level = config.GetDefaultVerbosity()
    }

    // Format output
    formatter := output.NewFormatter(level)
    if err := formatter.Format(result, os.Stdout); err != nil {
        return err
    }

    return nil
}
```

**Checkpoint:** Build and test
```bash
go build ./cmd/crisk
./bin/crisk check --quiet <file>
./bin/crisk check <file>  # default
./bin/crisk check --explain <file>
```

**ASK USER:** "CLI integration complete. Binary compiles. Should I proceed with unit tests?"

---

### Step 8: Unit Testing (2-3 hours)

**File: `internal/output/formatter_test.go`**

```go
package output

import (
    "bytes"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/coderisk/coderisk-go/internal/models"
)

func TestQuietFormatter(t *testing.T) {
    tests := []struct {
        name     string
        result   *models.RiskResult
        expected string
    }{
        {
            name: "low risk",
            result: &models.RiskResult{RiskLevel: "LOW"},
            expected: "✅ LOW risk\n",
        },
        {
            name: "medium risk with issues",
            result: &models.RiskResult{
                RiskLevel: "MEDIUM",
                Issues:    []models.Issue{{Message: "No tests"}},
            },
            expected: "⚠️  MEDIUM risk: 1 issues detected\nRun 'crisk check' for details\n",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var buf bytes.Buffer
            formatter := &QuietFormatter{}
            err := formatter.Format(tt.result, &buf)

            assert.NoError(t, err)
            assert.Equal(t, tt.expected, buf.String())
        })
    }
}

func TestStandardFormatter(t *testing.T) {
    // Similar tests for StandardFormatter
}

func TestExplainFormatter(t *testing.T) {
    // Similar tests for ExplainFormatter
}

func TestNewFormatter(t *testing.T) {
    tests := []struct {
        level    VerbosityLevel
        expected string
    }{
        {VerbosityQuiet, "*output.QuietFormatter"},
        {VerbosityStandard, "*output.StandardFormatter"},
        {VerbosityExplain, "*output.ExplainFormatter"},
    }

    for _, tt := range tests {
        formatter := NewFormatter(tt.level)
        assert.Contains(t, fmt.Sprintf("%T", formatter), tt.expected)
    }
}
```

**Checkpoint:** Run unit tests
```bash
go test ./internal/output/... -v -cover
```

**ASK USER:** "Unit tests written. Coverage: [X]%. All tests pass. Should I proceed with integration tests?"

---

### Step 9: Integration Testing (2-3 hours)

**File: `test/integration/test_verbosity.sh`**

```bash
#!/bin/bash
set -e

echo "=== Adaptive Verbosity Integration Test ==="

# Build binary
go build -o bin/crisk ./cmd/crisk

# Test file (mock risky file)
TEST_FILE="test_auth.go"
echo "package main\n// No tests" > $TEST_FILE

# Test 1: Quiet mode
echo "Test 1: Quiet mode..."
OUTPUT=$(./bin/crisk check --quiet $TEST_FILE)
if echo "$OUTPUT" | grep -q "risk"; then
    echo "✅ PASS: Quiet mode outputs risk level"
else
    echo "❌ FAIL: Quiet mode output incorrect"
    exit 1
fi

# Test 2: Standard mode (default)
echo "Test 2: Standard mode..."
OUTPUT=$(./bin/crisk check $TEST_FILE)
if echo "$OUTPUT" | grep -q "Issues:" && echo "$OUTPUT" | grep -q "Recommendations:"; then
    echo "✅ PASS: Standard mode shows issues and recommendations"
else
    echo "❌ FAIL: Standard mode output incorrect"
    exit 1
fi

# Test 3: Explain mode
echo "Test 3: Explain mode..."
OUTPUT=$(./bin/crisk check --explain $TEST_FILE)
if echo "$OUTPUT" | grep -q "Investigation Report" && echo "$OUTPUT" | grep -q "Hop"; then
    echo "✅ PASS: Explain mode shows investigation trace"
else
    echo "❌ FAIL: Explain mode output incorrect"
    exit 1
fi

# Test 4: Mutually exclusive flags
echo "Test 4: Mutually exclusive flags..."
if ./bin/crisk check --quiet --explain $TEST_FILE 2>&1 | grep -q "mutually exclusive"; then
    echo "✅ PASS: Flags are mutually exclusive"
else
    echo "❌ FAIL: Flags should be mutually exclusive"
    exit 1
fi

# Cleanup
rm -f $TEST_FILE

echo "=== All verbosity tests passed! ==="
```

**Make executable:** `chmod +x test/integration/test_verbosity.sh`

**Checkpoint:** Run integration tests
**ASK USER:** "Integration test ready. Should I execute: `./test/integration/test_verbosity.sh`?"

---

### Step 10: Final Validation & Documentation (1 hour)

**Validation checklist:**
- [ ] Run `go build ./internal/output` - Verify compiles
- [ ] Run `go test ./internal/output/... -v -cover` - Verify >80% coverage
- [ ] Run `./test/integration/test_verbosity.sh` - Verify end-to-end
- [ ] Test each verbosity level manually:
  ```bash
  ./bin/crisk check --quiet <file>
  ./bin/crisk check <file>
  ./bin/crisk check --explain <file>
  ```

**Output validation:**
- [ ] Quiet: One line, shows risk level ✅
- [ ] Standard: Issues + recommendations, readable format ✅
- [ ] Explain: Full trace with hops, metrics, decisions ✅

**ASK USER:** "All verbosity levels tested. Output examples:
[paste quiet output]
[paste standard output]
[paste explain output]

Are these outputs acceptable per the design spec?"

**Documentation:**
- [ ] Update `dev_docs/03-implementation/status.md`:
  - Mark adaptive verbosity (L1: Quiet) as ✅ Complete
  - Mark adaptive verbosity (L2: Standard) as ✅ Complete
  - Mark adaptive verbosity (L3: Explain) as ✅ Complete

**ASK USER:** "Session 2 complete! All tests pass. Coverage: [X]%. Should I update status.md and mark deliverables complete?"

---

## Critical Checkpoints (Human Verification Required)

### Checkpoint 1: After Step 2 (Interface Creation) - **CRITICAL!**
**YOU ASK:** "✅ Formatter interface created at internal/output/formatter.go. Other sessions can now import it. Should I proceed with Level 1?"
**WAIT FOR:** User confirmation + notification to Sessions 1 & 3

### Checkpoint 2: After Step 5 (All Formatters Done)
**YOU ASK:** "All 3 formatters implemented (Quiet, Standard, Explain). Should I proceed with CLI integration?"
**WAIT FOR:** User confirmation

### Checkpoint 3: After Step 8 (Unit Tests)
**YOU ASK:** "Unit tests complete. Coverage: [X]%. All pass: YES/NO. Should I proceed with integration tests?"
**WAIT FOR:** User confirmation

### Checkpoint 4: Final (Before completion)
**YOU ASK:** "All deliverables complete. Output validated. Ready to mark session complete?"
**WAIT FOR:** User final approval

---

## Coordination with Other Sessions

### YOU CREATE (other sessions depend on this):
- **`internal/output/formatter.go`** - **CREATE THIS FIRST!**
  - Session 1 needs this for pre-commit hook quiet mode
  - Session 3 needs this to implement AIFormatter

### DO NOT MODIFY (other sessions own these):
- `cmd/crisk/hook.go` - Session 1
- `internal/git/*` - Session 1
- `internal/output/ai_mode.go` - Session 3
- `internal/ai/*` - Session 3

### NOTIFY OTHER SESSIONS WHEN:
- **After Step 2:** Formatter interface is ready
  - **ASK USER:** "Should I notify Sessions 1 & 3 that formatter.go is ready?"

---

## Success Criteria

**Functional:**
- [ ] Formatter interface created and usable
- [ ] Level 1 (Quiet): One-line summary works
- [ ] Level 2 (Standard): Issues + recommendations display correctly
- [ ] Level 3 (Explain): Full trace with hops shows
- [ ] Flags are mutually exclusive (--quiet, --explain)

**Quality:**
- [ ] 80%+ unit test coverage
- [ ] Integration tests pass
- [ ] Output is readable and actionable

**Performance:**
- [ ] Formatting overhead <10ms for all levels

---

## Error Handling

**If you encounter issues:**
1. **Model structure unclear:** Read `internal/models/risk_result.go` carefully
2. **Build errors:** Verify imports and package structure
3. **Test failures:** Report to user with details
4. **Output formatting issues:** Ask user for clarification on design

**Always ask before:**
- Modifying files not in your ownership list
- Changing the formatter interface (other sessions depend on it)
- Adding new dependencies
- Making breaking changes

---

## Final Deliverables

When complete, you should have:
1. ✅ Formatter interface (`internal/output/formatter.go`)
2. ✅ Level 1: Quiet formatter
3. ✅ Level 2: Standard formatter
4. ✅ Level 3: Explain formatter
5. ✅ Verbosity detection logic
6. ✅ CLI integration (--quiet, --explain flags)
7. ✅ Unit tests (80%+ coverage)
8. ✅ Integration test suite
9. ✅ Updated status.md documentation

---

## Questions to Ask During Implementation

- "I've read the documentation. Should I proceed with interface creation?"
- "✅ Formatter interface created. Should notify other sessions?"
- "Level 1 (Quiet) implemented. Should I proceed with Level 2?"
- "All formatters done. Should I proceed with CLI integration?"
- "Unit tests complete. Coverage: [X]%. Should I proceed with integration tests?"
- "Integration test ready. Should I run it?"
- "Output validated. Are these formats acceptable?"
- "Session 2 complete! Should I update status.md?"

**Remember:** You create the interface that other sessions depend on. **Create formatter.go FIRST!**
