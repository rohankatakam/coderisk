# UX Integration Guide: Adaptive Verbosity

**Purpose:** Implement 4-level adaptive verbosity system for context-aware CLI output
**Last Updated:** October 4, 2025
**Reference:** [developer_experience.md](../../00-product/developer_experience.md) §2 - Adaptive Verbosity

> **Design Principle:** Show enough information to act, nothing more. (12-factor: Factor 3 - Own your context window)

---

## Overview

CodeRisk implements 4 verbosity levels that progressively disclose complexity based on user context:

1. **Level 1: Quiet** (`--quiet`) - One-line summary for pre-commit hooks
2. **Level 2: Standard** (default) - Issues + recommendations for interactive CLI
3. **Level 3: Explain** (`--explain`) - Full investigation trace for debugging
4. **Level 4: AI Mode** (`--ai-mode`) - Machine-readable JSON for AI assistants

---

## Implementation Strategy

### 1. CLI Flag Design

**File:** `cmd/crisk/check.go`

```go
var checkCmd = &cobra.Command{
    Use:   "check [file]",
    Short: "Analyze file for risk",
    RunE:  runCheck,
}

func init() {
    checkCmd.Flags().Bool("quiet", false, "Output one-line summary (for pre-commit hooks)")
    checkCmd.Flags().Bool("explain", false, "Show full investigation trace")
    checkCmd.Flags().Bool("ai-mode", false, "Output machine-readable JSON for AI assistants")
    checkCmd.Flags().String("format", "text", "Output format: text, json, ai")

    // Mutually exclusive flags
    checkCmd.MarkFlagsMutuallyExclusive("quiet", "explain", "ai-mode")
}
```

### 2. Output Formatter Interface

**File:** `internal/output/formatter.go`

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

// NewFormatter creates appropriate formatter based on flags
func NewFormatter(level VerbosityLevel) Formatter {
    switch level {
    case VerbosityQuiet:
        return &QuietFormatter{}
    case VerbosityStandard:
        return &StandardFormatter{}
    case VerbosityExplain:
        return &ExplainFormatter{}
    case VerbosityAIMode:
        return &AIFormatter{}
    default:
        return &StandardFormatter{}
    }
}
```

### 3. Level 1: Quiet Formatter

**File:** `internal/output/quiet.go`

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

**Output Examples:**
```bash
# Success
✅ LOW risk

# Risk detected
⚠️  MEDIUM risk: 3 issues detected
Run 'crisk check' for details
```

### 4. Level 2: Standard Formatter

**File:** `internal/output/standard.go`

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
    fmt.Fprintf(w, "Branch: %s\n", result.Branch)
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
    fmt.Fprintf(w, "Run 'crisk check --explain' for investigation trace\n")

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

**Output Example:**
```
🔍 CodeRisk Analysis
Branch: feature/auth-improvements
Files changed: 3
Risk level: MEDIUM

Issues:
1. ⚠️  auth.py - No test coverage (0%)
2. ⚠️  auth_middleware.py - High coupling (8 dependencies)
3. ℹ️  user_service.py - Changed with auth.py in 85% of commits

Recommendations:
- Add tests for auth.py
- Review dependencies in auth_middleware.py

Run 'crisk check --explain' for investigation trace
```

### 5. Level 3: Explain Formatter

**File:** `internal/output/explain.go`

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
    fmt.Fprintf(w, "Completed: %s (%.1fs)\n", result.EndTime.Format(time.RFC3339), result.Duration.Seconds())
    fmt.Fprintf(w, "Agent hops: %d\n\n", len(result.InvestigationTrace))

    // Investigation trace (hop-by-hop)
    for i, hop := range result.InvestigationTrace {
        fmt.Fprintf(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
        fmt.Fprintf(w, "Hop %d: %s (Starting point)\n", i+1, hop.NodeID)
        fmt.Fprintf(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

        // Changed functions/entities
        if len(hop.ChangedEntities) > 0 {
            fmt.Fprintf(w, "Changed functions:\n")
            for _, entity := range hop.ChangedEntities {
                fmt.Fprintf(w, "  - %s (lines %d-%d)\n", entity.Name, entity.StartLine, entity.EndLine)
            }
            fmt.Fprintf(w, "\n")
        }

        // Metrics calculated
        if len(hop.Metrics) > 0 {
            fmt.Fprintf(w, "Metrics calculated:\n")
            for _, metric := range hop.Metrics {
                status := "✅"
                if metric.Threshold != nil && metric.Value > *metric.Threshold {
                    status = "❌"
                } else if metric.Warning != nil && metric.Value > *metric.Warning {
                    status = "⚠️ "
                }
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
```

### 6. Level 4: AI Mode Formatter

**File:** `internal/output/ai_mode.go`

```go
package output

import (
    "encoding/json"
    "io"
    "github.com/coderisk/coderisk-go/internal/models"
)

// AIFormatter outputs machine-readable JSON for AI assistants
// Schema: https://coderisk.com/schemas/ai-mode/v1.0.json
type AIFormatter struct {
    Version string
}

func NewAIFormatter() *AIFormatter {
    return &AIFormatter{Version: "1.0"}
}

func (f *AIFormatter) Format(result *models.RiskResult, w io.Writer) error {
    output := f.transformToAISchema(result)

    encoder := json.NewEncoder(w)
    encoder.SetIndent("", "  ")
    return encoder.Encode(output)
}

func (f *AIFormatter) transformToAISchema(result *models.RiskResult) map[string]interface{} {
    return map[string]interface{}{
        "meta": map[string]interface{}{
            "version":        f.Version,
            "timestamp":      result.EndTime,
            "duration_ms":    result.Duration.Milliseconds(),
            "branch":         result.Branch,
            "files_analyzed": result.FilesChanged,
            "agent_hops":     len(result.InvestigationTrace),
            "cache_hit":      result.CacheHit,
        },
        "risk": map[string]interface{}{
            "level":      result.RiskLevel,
            "score":      result.RiskScore,
            "confidence": result.Confidence,
        },
        "files": f.transformFiles(result.Files),
        "graph_analysis": map[string]interface{}{
            "blast_radius":      result.BlastRadius,
            "temporal_coupling": result.TemporalCoupling,
            "hotspots":         result.Hotspots,
        },
        "investigation_trace": f.transformTrace(result.InvestigationTrace),
        "recommendations": map[string]interface{}{
            "critical": f.filterRecommendations(result.Recommendations, "critical"),
            "high":     f.filterRecommendations(result.Recommendations, "high"),
            "medium":   f.filterRecommendations(result.Recommendations, "medium"),
        },
        "ai_assistant_actions":  f.generateAIActions(result),
        "should_block_commit":   result.ShouldBlock,
        "block_reason":          result.BlockReason,
        "override_allowed":      result.OverrideAllowed,
    }
}

func (f *AIFormatter) generateAIActions(result *models.RiskResult) []map[string]interface{} {
    actions := []map[string]interface{}{}

    // Generate actions for each fixable issue
    for _, issue := range result.Issues {
        if issue.AutoFixable {
            action := map[string]interface{}{
                "action_type":       issue.FixType,
                "confidence":        issue.FixConfidence,
                "ready_to_execute":  issue.FixConfidence > 0.85,
                "prompt":            issue.AIPromptTemplate,
                "expected_files":    issue.ExpectedFiles,
                "estimated_lines":   issue.EstimatedLines,
            }
            actions = append(actions, action)
        }
    }

    return actions
}
```

**JSON Schema Reference:** See [developer_experience.md](../../00-product/developer_experience.md) §2.4 for full schema.

---

## Integration with CLI

### Usage Examples

```bash
# Level 1: Quiet (pre-commit hook)
crisk check --quiet
# Output: ✅ LOW risk

# Level 2: Standard (default)
crisk check src/auth.py
# Output: Issues + recommendations

# Level 3: Explain (debugging)
crisk check src/auth.py --explain
# Output: Full investigation trace

# Level 4: AI Mode (for Claude Code/Cursor)
crisk check src/auth.py --ai-mode
# Output: JSON schema with AI actions
```

### Environment-Based Defaults

**File:** `internal/config/verbosity.go`

```go
package config

import "os"

// GetDefaultVerbosity returns appropriate default based on environment
func GetDefaultVerbosity() VerbosityLevel {
    // Pre-commit hook context (GIT_AUTHOR_DATE set by git)
    if os.Getenv("GIT_AUTHOR_DATE") != "" {
        return VerbosityQuiet
    }

    // CI/CD context
    if os.Getenv("CI") == "true" {
        return VerbosityStandard
    }

    // AI assistant context (detected by special env var)
    if os.Getenv("CRISK_AI_MODE") == "1" {
        return VerbosityAIMode
    }

    // Interactive terminal (default)
    return VerbosityStandard
}
```

---

## Testing Strategy

### Unit Tests

**File:** `internal/output/formatter_test.go`

```go
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
```

### Integration Tests

```bash
# Test each verbosity level end-to-end
./test/integration/test_verbosity.sh
```

---

## Performance Considerations

**Formatting overhead by level:**
- **Quiet:** <1ms (minimal formatting)
- **Standard:** <5ms (issues + recommendations)
- **Explain:** <10ms (trace formatting)
- **AI Mode:** <20ms (JSON serialization, ~10KB output)

**Caching strategy:**
- All levels use same underlying `RiskResult`
- Formatting happens at output time (no redundant computation)
- AI Mode JSON is cached separately (larger payload)

---

## Next Steps

1. Implement formatters in `internal/output/` package
2. Wire formatters into `cmd/crisk/check.go`
3. Add environment detection for smart defaults
4. Create integration tests for each level
5. Document AI Mode JSON schema (see [ux_ai_mode_output.md](ux_ai_mode_output.md))

---

**See also:**
- [developer_experience.md](../../00-product/developer_experience.md) - UX design philosophy
- [ux_pre_commit_hook.md](ux_pre_commit_hook.md) - Pre-commit hook integration (uses quiet mode)
- [ux_ai_mode_output.md](ux_ai_mode_output.md) - AI Mode JSON schema details
- [DEVELOPMENT_WORKFLOW.md](../../DEVELOPMENT_WORKFLOW.md) - Implementation guardrails
