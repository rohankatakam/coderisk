# Task: Integrate Phase 2 LLM Investigation (P0 - CRITICAL)

**Priority:** P0 - BLOCKS CORE VALUE PROPOSITION
**Estimated Time:** 2-3 hours
**Gap Reference:** Gap C1 from [E2E_TEST_FINAL_REPORT.md](E2E_TEST_FINAL_REPORT.md#finding-1-phase-2-never-runs-gap-c1-confirmed--critical)

---

## Context & Problem Statement

**Current Issue:** Phase 2 LLM investigation never runs despite complete implementation existing.

**Evidence:**
- Test 3.2 (Phase 2 Integration) - FAILED
- `cmd/crisk/check.go:153-159` only prints a message instead of calling the investigator
- All Phase 2 code exists but is NOT integrated into the check command
- **Impact:** 85% of advertised functionality is blocked

**What Works:**
- ✅ LLM Client (`internal/agent/llm_client.go`) - OpenAI-only implementation
- ✅ Real Adapters (`internal/agent/adapters.go`) - RealTemporalClient, RealIncidentsClient
- ✅ Investigator (`internal/agent/investigator.go`) - Hop logic, evidence collection

**What's Missing:**
- ❌ Phase 2 escalation logic in `cmd/crisk/check.go`
- ❌ Client instantiation and investigator call
- ❌ Output formatting for investigation results

---

## Before You Start

### 1. Read Documentation (REQUIRED)

```bash
# Architecture & Requirements
cat dev_docs/01-architecture/system_overview_layman.md | grep -A 50 "Phase 2"
cat dev_docs/00-product/developer_experience.md | grep -A 100 "Investigation Summary"

# 12-Factor Principles
cat dev_docs/12-factor-agents-main/content/factor-08-own-your-control-flow.md
cat dev_docs/12-factor-agents-main/content/factor-10-small-focused-agents.md

# Existing Implementation
cat internal/agent/investigator.go | head -100
cat internal/agent/adapters.go
cat internal/agent/llm_client.go
```

### 2. Review Current Check Implementation

```bash
cat cmd/crisk/check.go | grep -A 30 "ShouldEscalate"
```

**Current Placeholder Code (lines 153-159):**
```go
if result.ShouldEscalate {
    hasHighRisk = true
    if !preCommit {
        fmt.Println("\n⚠️  HIGH RISK - Would escalate to Phase 2 (LLM investigation)")
        fmt.Println("    Phase 2 requires LLM API key (set PHASE2_ENABLED=true)")
        // ❌ Phase 2 never actually runs
    }
}
```

---

## Implementation Tasks

### Task 1: Add Phase 2 Escalation Logic to check.go

**File:** `cmd/crisk/check.go`
**Location:** Lines 153-180 (replace placeholder)

**Implementation Steps:**

1. **Import Required Packages:**
```go
import (
    // ... existing imports ...
    "os"
    "github.com/coderisk/coderisk-go/internal/agent"
)
```

2. **Replace Placeholder with Real Integration:**

```go
// Replace lines 153-159 with:
if result.ShouldEscalate {
    hasHighRisk = true

    // Get OpenAI API key from environment
    apiKey := os.Getenv("OPENAI_API_KEY")

    if apiKey == "" {
        // No API key - show message and continue
        if !preCommit {
            fmt.Println("\n⚠️  HIGH RISK detected")
            fmt.Println("    Set OPENAI_API_KEY to enable Phase 2 LLM investigation")
            fmt.Println("    Example: export OPENAI_API_KEY=sk-...")
        }
        continue  // Move to next file
    }

    // Phase 2: LLM Investigation
    // 12-factor: Factor 8 - Own your control flow (selective investigation)
    if !preCommit {
        fmt.Println("\n🔍 Escalating to Phase 2 (LLM investigation)...")
    }

    // Create real clients for evidence collection
    temporalClient, err := agent.NewRealTemporalClient(repoPath)
    if err != nil {
        slog.Warn("temporal client creation failed", "error", err)
        temporalClient = nil  // Continue without temporal data
    }

    // Get incidents database from context
    // Note: incidentsDB should be created earlier in check.go
    // If not available, create it from config
    var incidentsClient *agent.RealIncidentsClient
    if incidentsDB != nil {
        incidentsClient = agent.NewRealIncidentsClient(incidentsDB)
    }

    // Create LLM client
    llmClient, err := agent.NewLLMClient(apiKey)
    if err != nil {
        fmt.Printf("❌ LLM client error: %v\n", err)
        continue
    }

    // Create investigator
    // 12-factor: Factor 10 - Small, focused agents (investigator is specialized)
    investigator := agent.NewInvestigator(llmClient, temporalClient, incidentsClient)

    // Run Phase 2 investigation
    investigation, err := investigator.Investigate(ctx, filePath, result)
    if err != nil {
        fmt.Printf("⚠️  Investigation failed: %v\n", err)
        continue
    }

    // Display results based on verbosity mode
    if aiMode {
        // AI Mode: Include investigation trace in JSON
        output.DisplayPhase2JSON(investigation, result)
    } else if explainMode {
        // Explain Mode: Show full hop-by-hop trace
        output.DisplayPhase2Trace(investigation)
    } else {
        // Standard Mode: Show summary with recommendations
        output.DisplayPhase2Summary(investigation)
    }
}
```

---

### Task 2: Create Output Display Functions

**File:** `internal/output/phase2.go` (NEW FILE)

**Purpose:** Format Phase 2 investigation results for different verbosity levels

**Implementation:**

```go
package output

import (
    "fmt"
    "github.com/coderisk/coderisk-go/internal/agent"
)

// DisplayPhase2Summary shows investigation summary (standard mode)
func DisplayPhase2Summary(investigation *agent.Investigation) {
    fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Printf("Investigation Summary\n")
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

    // Show key evidence
    fmt.Println("\nKey Evidence:")
    for i, evidence := range investigation.Evidence {
        fmt.Printf("%d. %s\n", i+1, evidence.Summary)
    }

    // Show final assessment
    fmt.Printf("\nRisk Level: %s (confidence: %.0f%%)\n",
        investigation.FinalRisk.Level,
        investigation.FinalRisk.Confidence*100)

    // Show recommendations
    if len(investigation.Recommendations) > 0 {
        fmt.Println("\nRecommendations:")
        for i, rec := range investigation.Recommendations {
            fmt.Printf("%d. [%s] %s\n", i+1, formatTime(rec.EstimatedTime), rec.Action)
        }
    }
}

// DisplayPhase2Trace shows full hop-by-hop investigation (explain mode)
func DisplayPhase2Trace(investigation *agent.Investigation) {
    fmt.Println("\n🔍 CodeRisk Investigation Report")
    fmt.Printf("Started: %s\n", investigation.StartTime.Format("2006-01-02 15:04:05"))
    fmt.Printf("Completed: %s (%.1fs)\n",
        investigation.EndTime.Format("2006-01-02 15:04:05"),
        investigation.Duration.Seconds())
    fmt.Printf("Agent hops: %d\n", len(investigation.Hops))

    // Show each hop
    for i, hop := range investigation.Hops {
        fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        fmt.Printf("Hop %d: %s\n", i+1, hop.NodeID)
        fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

        if hop.Action != "" {
            fmt.Printf("\nAction: %s\n", hop.Action)
        }

        if hop.MetricsCalculated != nil && len(hop.MetricsCalculated) > 0 {
            fmt.Println("\nMetrics calculated:")
            for _, metric := range hop.MetricsCalculated {
                fmt.Printf("  %s: %v\n", metric.Name, metric.Value)
            }
        }

        if hop.Reasoning != "" {
            fmt.Printf("\nReasoning: %s\n", hop.Reasoning)
        }

        if hop.NextAction != "" {
            fmt.Printf("→ Decision: %s\n", hop.NextAction)
        }
    }

    // Show final assessment
    fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Println("Final Assessment")
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

    DisplayPhase2Summary(investigation)
}

// DisplayPhase2JSON includes investigation trace in AI mode JSON
func DisplayPhase2JSON(investigation *agent.Investigation, baseResult *risk.AssessmentResult) {
    // This will be integrated with existing AI mode output in converter.go
    // Add investigation_trace array to JSON
    // See Task P1-2 (AI Mode completion) for full implementation

    // For now, convert investigation to trace format
    trace := convertInvestigationToTrace(investigation)

    // Output will be handled by existing AI mode converter
    // Just ensure investigation data is passed through
    fmt.Printf("  \"investigation_trace\": %s,\n", toJSON(trace))
}

// Helper functions
func formatTime(minutes int) string {
    if minutes < 60 {
        return fmt.Sprintf("%dmin", minutes)
    }
    hours := minutes / 60
    mins := minutes % 60
    if mins == 0 {
        return fmt.Sprintf("%dh", hours)
    }
    return fmt.Sprintf("%dh %dmin", hours, mins)
}

func convertInvestigationToTrace(investigation *agent.Investigation) []map[string]interface{} {
    trace := make([]map[string]interface{}, len(investigation.Hops))
    for i, hop := range investigation.Hops {
        trace[i] = map[string]interface{}{
            "hop":              i + 1,
            "node_type":        hop.NodeType,
            "node_id":          hop.NodeID,
            "action":           hop.Action,
            "metrics_calculated": hop.MetricsCalculated,
            "decision":         hop.NextAction,
            "reasoning":        hop.Reasoning,
            "confidence":       hop.Confidence,
            "duration_ms":      hop.Duration.Milliseconds(),
        }
    }
    return trace
}
```

---

### Task 3: Ensure Dependencies Are Available

**File:** `cmd/crisk/check.go`

**Check these exist in check.go scope:**

1. **Repository Path:**
```go
// Should exist near top of runCheck() function
repoPath := getRepoPath()  // or similar
```

2. **Incidents Database:**
```go
// If not exists, add near database initialization:
var incidentsDB *incidents.Database
if dbConn != nil {
    incidentsDB = incidents.NewDatabase(dbConn)
}
```

3. **Context with Timeout:**
```go
// Should exist for Phase 1 queries
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

---

### Task 4: Add Error Handling & Logging

**Add structured logging throughout:**

```go
// Before Phase 2
slog.Info("phase 2 escalation",
    "file", filePath,
    "phase1_risk", result.RiskLevel,
    "api_key_present", apiKey != "")

// After investigation
slog.Info("phase 2 complete",
    "file", filePath,
    "hops", len(investigation.Hops),
    "final_risk", investigation.FinalRisk.Level,
    "duration_ms", investigation.Duration.Milliseconds())
```

---

## Testing Instructions

### Test 1: Build & Compile

```bash
go build -o crisk ./cmd/crisk
# Expected: No errors
```

### Test 2: Phase 2 Without API Key

```bash
# Create high-risk scenario (or use existing test file)
./crisk check <some-file>

# Expected output:
# ⚠️  HIGH RISK detected
# Set OPENAI_API_KEY to enable Phase 2 LLM investigation
```

### Test 3: Phase 2 With API Key

```bash
export OPENAI_API_KEY="sk-..."  # Use real or test key

./crisk check <high-risk-file>

# Expected output (standard mode):
# 🔍 Escalating to Phase 2 (LLM investigation)...
#
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Investigation Summary
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# [... evidence, recommendations ...]
```

### Test 4: Explain Mode

```bash
export OPENAI_API_KEY="sk-..."
./crisk check --explain <high-risk-file>

# Expected: Full hop-by-hop trace
# Hop 1: <file>
#   Action: calculate_coupling
#   Metrics: ...
#   → Decision: check_ownership
# [... etc ...]
```

### Test 5: AI Mode

```bash
export OPENAI_API_KEY="sk-..."
./crisk check --ai-mode <high-risk-file> | jq '.investigation_trace'

# Expected: JSON array with hop objects
# [
#   {"hop": 1, "node_type": "file", "action": "...", ...},
#   ...
# ]
```

---

## Validation Criteria

**Success Criteria:**
- [ ] ✅ Build succeeds with no errors
- [ ] ✅ Without API key: Shows helpful message, doesn't crash
- [ ] ✅ With API key: Phase 2 runs and shows investigation
- [ ] ✅ Standard mode: Shows summary with evidence & recommendations
- [ ] ✅ Explain mode: Shows full hop-by-hop trace
- [ ] ✅ AI mode: Includes `investigation_trace` array in JSON
- [ ] ✅ Logging: Structured logs for Phase 2 execution
- [ ] ✅ Performance: Phase 2 completes in <10s for typical case

**Code Quality:**
- [ ] ✅ Error handling: All errors properly wrapped with context
- [ ] ✅ Context: Timeouts set for LLM calls
- [ ] ✅ Logging: Uses `slog` for structured logging
- [ ] ✅ 12-factor: Cites Factor 8 (control flow) and Factor 10 (small agents)

---

## Reference Examples from Spec

### Expected Output (from system_overview_layman.md:315-357)

```
🔍 CodeRisk: Analyzing... (3.4s)

🔴 HIGH risk detected (confidence: 87%)

Investigation Summary:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Hop 1: payment_processor.py
  ✓ Calculated coupling: 18 files (HIGH)
  → Decision: Check ownership stability

Hop 2: Ownership analysis
  ⚠ Code owner: Bob → Alice (14 days ago)
  → Decision: Search for similar incidents

Hop 3: Incident search
  🚨 Found INC-892 (89% match): "Stripe timeout cascade"
     - Affected transactions.py (same 85% co-change pattern)
     - Occurred 25 days ago
  → Decision: FINALIZE (strong evidence)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Key Evidence:
1. 18 files depend on payment_processor.py (blast radius)
2. Changes with transactions.py 85% of the time
3. New code owner (14 days) unfamiliar with incident history
4. Similar to INC-892: Stripe timeout → transaction cascade
5. Only 33% test coverage for critical payment code

Recommendations:
1. [45min] Add integration tests: Stripe timeout + retry + rollback
2. [15min] Alice: Review INC-892 post-mortem before merging
3. [Future] Consider circuit breaker to prevent transactions.py cascade
```

---

## Troubleshooting

### Issue: "investigator undefined"

**Solution:** Add import:
```go
import "github.com/coderisk/coderisk-go/internal/agent"
```

### Issue: "incidentsDB undefined"

**Solution:** Create database client earlier in check.go:
```go
// Near database setup
incidentsDB := incidents.NewDatabase(sqlxDB)
```

### Issue: "repoPath undefined"

**Solution:** Extract from git or use working directory:
```go
repoPath, _ := git.GetRepoRoot()
if repoPath == "" {
    repoPath, _ = os.Getwd()
}
```

### Issue: LLM API timeout

**Solution:** Add context timeout:
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

---

## Commit Message Template

```
Integrate Phase 2 LLM investigation into check command

Implements Phase 2 escalation when Phase 1 detects high risk.
Creates real clients (temporal, incidents, LLM) and runs investigator.
Adds output formatting for standard, explain, and AI modes.

12-factor: Factor 8 - Own your control flow (selective investigation)
12-factor: Factor 10 - Small, focused agents (investigator specialization)

Fixes: Gap C1 (Phase 2 never runs)
Tests: Validated with high-risk file scenarios
Performance: <10s for typical 3-hop investigation

- Add Phase 2 escalation logic to cmd/crisk/check.go
- Create internal/output/phase2.go for result formatting
- Add structured logging for investigation tracing
- Handle missing API key gracefully
```

---

## Success Validation

After implementation, verify against original test:

```bash
# Re-run Test 3.2 from E2E_TEST_FINAL_REPORT.md
export OPENAI_API_KEY="sk-..."
./crisk check apps/web/src/app/page.tsx

# Expected: ✅ PASS
# - Phase 2 triggers if risk detected
# - Investigation trace shown
# - Recommendations provided
# - Performance <10s
```

**This task is COMPLETE when:**
- Phase 2 runs when risk threshold exceeded ✅
- All 3 verbosity modes work (standard, explain, AI) ✅
- Code follows DEVELOPMENT_WORKFLOW.md guidelines ✅
- Tests pass and performance targets met ✅
