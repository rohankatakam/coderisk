# Task: Complete AI Mode JSON Output (P1 - HIGH)

**Priority:** P1 - MISSING ADVERTISED FEATURE
**Estimated Time:** 4-6 hours
**Gap Reference:** Gap UX1 from [E2E_TEST_FINAL_REPORT.md](E2E_TEST_FINAL_REPORT.md#finding-4-ai-mode-json-incomplete-gap-ux1-confirmed--high)

---

## Context & Problem Statement

**Current Issue:** `--ai-mode` flag outputs partial JSON with critical fields missing

**Evidence:**
- Test 4.2 (Verbosity Level 4: AI Mode) - PARTIAL PASS
- `ai_assistant_actions[]` array is always empty
- `graph_analysis` fields incomplete (blast_radius, temporal_coupling, hotspots all empty/zero)
- `investigation_trace[]` missing (requires Phase 2 integration)
- `recommendations` object incomplete

**What Works:**
- ✅ `--ai-mode` flag exists in check.go
- ✅ Basic JSON structure in `internal/output/converter.go`
- ✅ File metrics included

**What's Missing:**
- ❌ AI assistant action prompts (auto-fixable suggestions)
- ❌ Blast radius calculation (affected files from graph)
- ❌ Temporal coupling data (from Layer 2 CO_CHANGED edges)
- ❌ Hotspot identification (high churn + low coverage)
- ❌ Confidence scores and estimated fix times

**Impact:**
- AI assistants (Claude Code, Cursor) cannot integrate with CodeRisk
- Silent quality improvement workflow impossible
- Limits adoption by AI coding tools

---

## Before You Start

### 1. Read Documentation (REQUIRED)

```bash
# Expected AI Mode Schema
cat dev_docs/00-product/developer_experience.md | sed -n '290,575p'

# 12-Factor Principles
cat dev_docs/12-factor-agents-main/content/factor-04-tools-are-structured-outputs.md

# Existing converter implementation
cat internal/output/converter.go
cat internal/output/types.go
```

### 2. Review Current AI Mode Output

```bash
# Run check in AI mode
./crisk check --ai-mode <file> | jq '.' | head -50

# Compare with spec in developer_experience.md
diff <(./crisk check --ai-mode <file> | jq 'keys') \
     <(cat dev_docs/00-product/developer_experience.md | grep -oP '"[a-z_]+":\s' | sed 's/://; s/"//g' | sort -u)
```

---

## Implementation Tasks

### Task 1: Enhance AI Mode JSON Schema

**File:** `internal/output/types.go`

**Add Missing Types:**

```go
package output

import "time"

// AIJSONOutput is the complete schema for --ai-mode
// Reference: dev_docs/00-product/developer_experience.md lines 290-575
type AIJSONOutput struct {
    AIAssistantActions []AIAssistantAction `json:"ai_assistant_actions"`
    BlockReason        string              `json:"block_reason,omitempty"`
    Branch             string              `json:"branch"`
    ContextualInsights ContextualInsights  `json:"contextual_insights"`
    Files              []FileAnalysis      `json:"files"`
    GraphAnalysis      GraphAnalysis       `json:"graph_analysis"`
    InvestigationTrace []InvestigationHop  `json:"investigation_trace"`
    OverallRisk        string              `json:"overall_risk"`
    Recommendations    Recommendations     `json:"recommendations"`
    Repository         string              `json:"repository"`
    Timestamp          time.Time           `json:"timestamp"`
}

// AIAssistantAction represents an auto-fixable action for AI coding assistants
type AIAssistantAction struct {
    ActionType      string   `json:"action_type"`      // "add_test", "refactor", "add_error_handling", etc.
    Confidence      float64  `json:"confidence"`        // 0.0-1.0
    Description     string   `json:"description"`       // Human-readable description
    EstimatedLines  int      `json:"estimated_lines"`   // Code change size estimate
    FilePath        string   `json:"file_path"`
    Function        string   `json:"function,omitempty"`
    LineNumber      int      `json:"line_number,omitempty"`
    Prompt          string   `json:"prompt"`            // Ready-to-execute prompt for AI
    ReadyToExecute  bool     `json:"ready_to_execute"`  // Can be run without human review
    RiskReduction   float64  `json:"risk_reduction"`    // Expected risk score improvement
}

// GraphAnalysis provides graph-based risk insights
type GraphAnalysis struct {
    BlastRadius      BlastRadius           `json:"blast_radius"`
    Hotspots         []Hotspot             `json:"hotspots"`
    TemporalCoupling []TemporalCouplingPair `json:"temporal_coupling"`
}

// BlastRadius shows impact of changes
type BlastRadius struct {
    DirectDependents    int      `json:"direct_dependents"`
    TotalAffectedFiles  int      `json:"total_affected_files"`
    CriticalPaths       []string `json:"critical_paths,omitempty"`       // Files on critical execution paths
    TransitiveDependents int     `json:"transitive_dependents"`
}

// Hotspot identifies risky areas in codebase
type Hotspot struct {
    File          string  `json:"file"`
    Score         float64 `json:"score"`           // 0.0-1.0 risk score
    Reason        string  `json:"reason"`          // "high_churn_low_coverage", "incident_prone", etc.
    ChurnRate     float64 `json:"churn_rate"`      // Changes per week
    TestCoverage  float64 `json:"test_coverage"`   // 0.0-1.0
    IncidentCount int     `json:"incident_count"`
}

// TemporalCouplingPair shows files that change together
type TemporalCouplingPair struct {
    FileA      string  `json:"file_a"`
    FileB      string  `json:"file_b"`
    Frequency  float64 `json:"frequency"`   // 0.0-1.0
    CoChanges  int     `json:"co_changes"`  // Number of times changed together
    WindowDays int     `json:"window_days"` // Analysis window
}

// ContextualInsights provides historical context
type ContextualInsights struct {
    FileReputation      map[string]float64    `json:"file_reputation"`       // File path → stability score
    SimilarPastChanges  []SimilarChange       `json:"similar_past_changes"`
    TeamPatterns        *TeamPattern          `json:"team_patterns,omitempty"`
}

// SimilarChange represents historically similar code changes
type SimilarChange struct {
    CommitSHA    string    `json:"commit_sha"`
    FilesChanged []string  `json:"files_changed"`
    Similarity   float64   `json:"similarity"`   // 0.0-1.0
    Outcome      string    `json:"outcome"`      // "success", "incident", "reverted"
    Date         time.Time `json:"date"`
}

// TeamPattern shows team-specific risk patterns
type TeamPattern struct {
    PeakRiskHours    []int   `json:"peak_risk_hours"`     // Hours of day (0-23)
    SafeReviewers    []string `json:"safe_reviewers"`     // Developers with low FP rate
    RiskyFileTypes   []string `json:"risky_file_types"`   // Extensions with high incident rate
}

// Recommendations provides actionable next steps
type Recommendations struct {
    Critical []Recommendation `json:"critical"`  // Must do before merge
    High     []Recommendation `json:"high"`      // Should do before merge
    Medium   []Recommendation `json:"medium"`    // Consider doing
    Future   []Recommendation `json:"future"`    // Technical debt / improvements
}

// Recommendation is a single suggested action
type Recommendation struct {
    Action         string `json:"action"`
    Reason         string `json:"reason"`
    EstimatedTime  int    `json:"estimated_time"`  // Minutes
    AutoFixable    bool   `json:"auto_fixable"`
    Priority       string `json:"priority"`        // "critical", "high", "medium", "low"
}

// InvestigationHop represents one step in Phase 2 investigation
// (Populated when Phase 2 runs)
type InvestigationHop struct {
    Hop                int                    `json:"hop"`
    NodeType           string                 `json:"node_type"`
    NodeID             string                 `json:"node_id"`
    Action             string                 `json:"action"`
    MetricsCalculated  []MetricResult         `json:"metrics_calculated"`
    Decision           string                 `json:"decision"`
    Reasoning          string                 `json:"reasoning"`
    Confidence         float64                `json:"confidence"`
    DurationMS         int64                  `json:"duration_ms"`
}

// MetricResult represents a calculated metric from investigation
type MetricResult struct {
    Name  string      `json:"name"`
    Value interface{} `json:"value"`
}
```

---

### Task 2: Implement AI Action Generation

**File:** `internal/output/ai_actions.go` (NEW FILE)

**Purpose:** Generate ready-to-execute AI prompts based on detected issues

```go
package output

import (
    "fmt"
    "github.com/coderisk/coderisk-go/internal/risk"
)

// GenerateAIActions creates actionable prompts for AI coding assistants
// 12-factor: Factor 4 - Tools are structured outputs
func GenerateAIActions(result *risk.AssessmentResult) []AIAssistantAction {
    var actions []AIAssistantAction

    // Generate actions based on detected issues
    for _, issue := range result.Issues {
        switch issue.Type {
        case "low_test_coverage":
            actions = append(actions, generateTestAction(result.FilePath, issue))

        case "high_coupling":
            actions = append(actions, generateDecouplingAction(result.FilePath, issue))

        case "missing_error_handling":
            actions = append(actions, generateErrorHandlingAction(result.FilePath, issue))

        case "incident_prone":
            actions = append(actions, generateIncidentPreventionAction(result.FilePath, issue))

        case "complex_function":
            actions = append(actions, generateRefactorAction(result.FilePath, issue))
        }
    }

    // Sort by confidence (highest first)
    sortActionsByConfidence(actions)

    return actions
}

func generateTestAction(filePath string, issue risk.Issue) AIAssistantAction {
    coverage := issue.Metrics["coverage"].(float64)
    missingLines := int((1.0 - coverage) * issue.Metrics["total_lines"].(float64))

    return AIAssistantAction{
        ActionType:      "add_test",
        Confidence:      0.9, // High confidence - test addition is straightforward
        Description:     fmt.Sprintf("Add unit tests to improve coverage from %.1f%% to 80%%", coverage*100),
        EstimatedLines:  missingLines / 2, // Rough estimate: 1 test line per 2 code lines
        FilePath:        filePath,
        Prompt: fmt.Sprintf(`Add comprehensive unit tests for %s to achieve 80%% coverage.

Current coverage: %.1f%%
Focus on:
- Edge cases and error conditions
- Critical business logic paths
- Input validation

Generate tests using the project's existing test framework.`, filePath, coverage*100),
        ReadyToExecute: true,
        RiskReduction:  0.3, // Tests reduce risk by ~30%
    }
}

func generateDecouplingAction(filePath string, issue risk.Issue) AIAssistantAction {
    couplingCount := issue.Metrics["coupling"].(int)

    return AIAssistantAction{
        ActionType:      "refactor",
        Confidence:      0.7, // Medium confidence - refactoring needs human review
        Description:     fmt.Sprintf("Reduce coupling from %d dependencies to <10", couplingCount),
        EstimatedLines:  50, // Typical refactor size
        FilePath:        filePath,
        Prompt: fmt.Sprintf(`Refactor %s to reduce coupling from %d dependencies.

Techniques to consider:
- Extract interfaces for external dependencies
- Apply dependency injection pattern
- Move business logic to separate modules
- Use facade pattern to simplify dependencies

Maintain existing behavior and test coverage.`, filePath, couplingCount),
        ReadyToExecute: false, // Requires human review
        RiskReduction:  0.4,   // Decoupling significantly reduces risk
    }
}

func generateErrorHandlingAction(filePath string, issue risk.Issue) AIAssistantAction {
    function := issue.Location.Function

    return AIAssistantAction{
        ActionType:      "add_error_handling",
        Confidence:      0.85,
        Description:     fmt.Sprintf("Add error handling to %s", function),
        EstimatedLines:  10,
        FilePath:        filePath,
        Function:        function,
        LineNumber:      issue.Location.Line,
        Prompt: fmt.Sprintf(`Add comprehensive error handling to function %s in %s at line %d.

Requirements:
- Handle all error cases explicitly
- Log errors with context (file, function, operation)
- Return wrapped errors with helpful messages
- Add input validation if missing

Follow the project's error handling patterns.`, function, filePath, issue.Location.Line),
        ReadyToExecute: true,
        RiskReduction:  0.25,
    }
}

func generateIncidentPreventionAction(filePath string, issue risk.Issue) AIAssistantAction {
    incidentID := issue.Metrics["incident_id"].(string)
    incidentTitle := issue.Metrics["incident_title"].(string)

    return AIAssistantAction{
        ActionType:      "prevent_incident",
        Confidence:      0.95, // High confidence - historical incident is strong signal
        Description:     fmt.Sprintf("Add safeguards against incident: %s", incidentTitle),
        EstimatedLines:  30,
        FilePath:        filePath,
        Prompt: fmt.Sprintf(`This file was involved in incident %s: "%s"

Add safeguards to prevent recurrence:
1. Review incident post-mortem and root cause
2. Add specific checks/validations to prevent the failure mode
3. Add monitoring/logging to detect similar issues early
4. Consider circuit breaker or fallback logic if applicable

Reference incident %s for details.`, incidentID, incidentTitle, incidentID),
        ReadyToExecute: false, // Requires reviewing incident details
        RiskReduction:  0.6,   // Preventing known incidents is high value
    }
}

func generateRefactorAction(filePath string, issue risk.Issue) AIAssistantAction {
    complexity := issue.Metrics["complexity"].(int)
    function := issue.Location.Function

    return AIAssistantAction{
        ActionType:      "refactor",
        Confidence:      0.75,
        Description:     fmt.Sprintf("Simplify %s (complexity: %d → <10)", function, complexity),
        EstimatedLines:  complexity * 2, // Refactor often increases line count initially
        FilePath:        filePath,
        Function:        function,
        LineNumber:      issue.Location.Line,
        Prompt: fmt.Sprintf(`Refactor function %s in %s to reduce cyclomatic complexity from %d to <10.

Techniques:
- Extract helper functions for complex logic blocks
- Replace nested conditionals with guard clauses
- Simplify boolean expressions
- Consider strategy pattern for complex branching

Maintain 100%% test coverage during refactor.`, function, filePath, complexity),
        ReadyToExecute: false,
        RiskReduction:  0.35,
    }
}

func sortActionsByConfidence(actions []AIAssistantAction) {
    // Sort descending by confidence, then by risk reduction
    // Implementation: standard sort with custom comparator
}
```

---

### Task 3: Implement Graph Analysis Functions

**File:** `internal/output/graph_analysis.go` (NEW FILE)

**Purpose:** Calculate blast radius, identify hotspots, get temporal coupling

```go
package output

import (
    "context"
    "github.com/coderisk/coderisk-go/internal/graph"
    "github.com/coderisk/coderisk-go/internal/temporal"
    "log/slog"
)

// CalculateBlastRadius determines impact of changing a file
func CalculateBlastRadius(ctx context.Context, filePath string, graphClient graph.Backend) BlastRadius {
    radius := BlastRadius{}

    // Query 1: Direct dependents (1-hop)
    directQuery := `
        MATCH (f:File {path: $filePath})<-[:IMPORTS]-(dep:File)
        RETURN count(DISTINCT dep) as count
    `

    result, err := graphClient.Query(ctx, directQuery, map[string]interface{}{
        "filePath": filePath,
    })
    if err != nil {
        slog.Warn("blast radius query failed", "error", err)
        return radius
    }

    if len(result) > 0 {
        radius.DirectDependents = int(result[0]["count"].(int64))
    }

    // Query 2: Transitive dependents (2-3 hops)
    transitiveQuery := `
        MATCH (f:File {path: $filePath})<-[:IMPORTS*1..3]-(dep:File)
        RETURN count(DISTINCT dep) as count
    `

    result, err = graphClient.Query(ctx, transitiveQuery, map[string]interface{}{
        "filePath": filePath,
    })
    if err == nil && len(result) > 0 {
        radius.TransitiveDependents = int(result[0]["count"].(int64))
        radius.TotalAffectedFiles = radius.TransitiveDependents
    }

    // Query 3: Critical paths (files on critical execution paths)
    // Heuristic: Files imported by >10 other files
    criticalQuery := `
        MATCH (f:File {path: $filePath})<-[:IMPORTS*1..2]-(dep:File)
        WHERE size((dep)<-[:IMPORTS]-()) > 10
        RETURN DISTINCT dep.path as path
        LIMIT 5
    `

    result, err = graphClient.Query(ctx, criticalQuery, map[string]interface{}{
        "filePath": filePath,
    })
    if err == nil {
        for _, row := range result {
            if path, ok := row["path"].(string); ok {
                radius.CriticalPaths = append(radius.CriticalPaths, path)
            }
        }
    }

    return radius
}

// GetTemporalCoupling retrieves co-change patterns from Layer 2
func GetTemporalCoupling(ctx context.Context, filePath string, graphClient graph.Backend, minFrequency float64) []TemporalCouplingPair {
    var pairs []TemporalCouplingPair

    query := `
        MATCH (f:File {path: $filePath})-[r:CO_CHANGED]-(other:File)
        WHERE r.frequency >= $minFrequency
        RETURN other.path as file_b,
               r.frequency as frequency,
               r.co_changes as co_changes,
               r.window_days as window_days
        ORDER BY r.frequency DESC
        LIMIT 10
    `

    result, err := graphClient.Query(ctx, query, map[string]interface{}{
        "filePath":     filePath,
        "minFrequency": minFrequency,
    })
    if err != nil {
        slog.Warn("temporal coupling query failed", "error", err)
        return pairs
    }

    for _, row := range result {
        pair := TemporalCouplingPair{
            FileA:      filePath,
            FileB:      row["file_b"].(string),
            Frequency:  row["frequency"].(float64),
            CoChanges:  int(row["co_changes"].(int64)),
            WindowDays: int(row["window_days"].(int64)),
        }
        pairs = append(pairs, pair)
    }

    return pairs
}

// IdentifyHotspots finds risky areas in the codebase
func IdentifyHotspots(ctx context.Context, result *risk.AssessmentResult, graphClient graph.Backend, temporalClient temporal.TemporalClient) []Hotspot {
    var hotspots []Hotspot

    // Hotspot criteria:
    // 1. High churn (many recent changes)
    // 2. Low test coverage
    // 3. Incident-prone (linked to incidents)
    // 4. Complex code (high cyclomatic complexity)

    // Check current file
    if shouldBeHotspot(result) {
        hotspot := Hotspot{
            File:         result.FilePath,
            Score:        calculateHotspotScore(result),
            Reason:       determineHotspotReason(result),
            ChurnRate:    result.Metrics.ChurnRate,
            TestCoverage: result.Metrics.TestCoverage,
            IncidentCount: result.Metrics.IncidentCount,
        }
        hotspots = append(hotspots, hotspot)
    }

    // Query for related hotspots (co-changed files with similar issues)
    query := `
        MATCH (f:File {path: $filePath})-[r:CO_CHANGED]-(other:File)
        WHERE r.frequency > 0.6
        RETURN other.path as path
        LIMIT 5
    `

    queryResult, err := graphClient.Query(ctx, query, map[string]interface{}{
        "filePath": result.FilePath,
    })
    if err == nil {
        for _, row := range queryResult {
            // Note: Would need metrics for other files to calculate score
            // For now, flag as potential hotspot
            hotspot := Hotspot{
                File:   row["path"].(string),
                Score:  0.6, // Default score for co-changed files
                Reason: "frequent_co_change",
            }
            hotspots = append(hotspots, hotspot)
        }
    }

    return hotspots
}

func shouldBeHotspot(result *risk.AssessmentResult) bool {
    return (result.Metrics.Complexity > 10 && result.Metrics.TestCoverage < 0.5) ||
           result.Metrics.IncidentCount > 0 ||
           result.Metrics.ChurnRate > 5.0  // >5 changes per week
}

func calculateHotspotScore(result *risk.AssessmentResult) float64 {
    score := 0.0

    // Factor 1: Complexity (0-0.3)
    if result.Metrics.Complexity > 15 {
        score += 0.3
    } else if result.Metrics.Complexity > 10 {
        score += 0.15
    }

    // Factor 2: Test coverage (0-0.3)
    score += (1.0 - result.Metrics.TestCoverage) * 0.3

    // Factor 3: Incident history (0-0.4)
    score += float64(min(result.Metrics.IncidentCount, 3)) * 0.133

    return min(score, 1.0)
}

func determineHotspotReason(result *risk.AssessmentResult) string {
    if result.Metrics.IncidentCount > 0 {
        return "incident_prone"
    }
    if result.Metrics.Complexity > 10 && result.Metrics.TestCoverage < 0.5 {
        return "high_churn_low_coverage"
    }
    if result.Metrics.ChurnRate > 5.0 {
        return "high_churn"
    }
    return "complex_code"
}

func min(a, b float64) float64 {
    if a < b {
        return a
    }
    return b
}
```

---

### Task 4: Update Main Converter

**File:** `internal/output/converter.go`

**Update `ToAIMode` function:**

```go
package output

import (
    "context"
    "time"
    "github.com/coderisk/coderisk-go/internal/risk"
    "github.com/coderisk/coderisk-go/internal/graph"
)

// ToAIMode converts risk assessment to AI-consumable JSON
// 12-factor: Factor 4 - Tools are structured outputs
func ToAIMode(ctx context.Context, result *risk.AssessmentResult, graphClient graph.Backend, investigation *Investigation) *AIJSONOutput {
    output := &AIJSONOutput{
        Branch:     result.Branch,
        Repository: result.Repository,
        Timestamp:  time.Now(),
        OverallRisk: result.RiskLevel,
    }

    // Generate AI assistant actions (auto-fix prompts)
    output.AIAssistantActions = GenerateAIActions(result)

    // Calculate graph analysis
    output.GraphAnalysis = GraphAnalysis{
        BlastRadius:      CalculateBlastRadius(ctx, result.FilePath, graphClient),
        TemporalCoupling: GetTemporalCoupling(ctx, result.FilePath, graphClient, 0.5),
        Hotspots:         IdentifyHotspots(ctx, result, graphClient, nil), // temporalClient optional
    }

    // Contextual insights
    output.ContextualInsights = ContextualInsights{
        FileReputation: map[string]float64{
            result.FilePath: calculateFileReputation(result),
        },
        SimilarPastChanges: findSimilarChanges(ctx, result, graphClient),
    }

    // Convert file analysis
    output.Files = []FileAnalysis{
        {
            Path:         result.FilePath,
            Language:     result.Language,
            LinesChanged: result.LinesChanged,
            RiskScore:    result.RiskScore,
            Metrics:      result.Metrics,
            Issues:       result.Issues,
        },
    }

    // Investigation trace (if Phase 2 ran)
    if investigation != nil {
        output.InvestigationTrace = convertInvestigationToTrace(investigation)
    }

    // Recommendations
    output.Recommendations = generateRecommendations(result, output.AIAssistantActions)

    // Block reason (if HIGH/CRITICAL risk)
    if result.RiskLevel == "HIGH" || result.RiskLevel == "CRITICAL" {
        output.BlockReason = determineBlockReason(result)
    }

    return output
}

func calculateFileReputation(result *risk.AssessmentResult) float64 {
    // Reputation = 1.0 - (normalized risk factors)
    reputation := 1.0

    // Penalize for incidents
    reputation -= float64(result.Metrics.IncidentCount) * 0.15

    // Penalize for low test coverage
    reputation -= (1.0 - result.Metrics.TestCoverage) * 0.2

    // Penalize for high complexity
    if result.Metrics.Complexity > 15 {
        reputation -= 0.2
    }

    return max(reputation, 0.0)
}

func findSimilarChanges(ctx context.Context, result *risk.AssessmentResult, graphClient graph.Backend) []SimilarChange {
    // Query git history for similar file patterns
    // For now, return empty (requires git integration)
    return []SimilarChange{}
}

func generateRecommendations(result *risk.AssessmentResult, actions []AIAssistantAction) Recommendations {
    recs := Recommendations{
        Critical: []Recommendation{},
        High:     []Recommendation{},
        Medium:   []Recommendation{},
        Future:   []Recommendation{},
    }

    // Convert AI actions to recommendations
    for _, action := range actions {
        rec := Recommendation{
            Action:        action.Description,
            Reason:        action.Prompt[:min(len(action.Prompt), 100)] + "...",
            EstimatedTime: action.EstimatedLines * 2, // 2 min per line rough estimate
            AutoFixable:   action.ReadyToExecute,
            Priority:      determinePriority(action.Confidence, action.RiskReduction),
        }

        switch rec.Priority {
        case "critical":
            recs.Critical = append(recs.Critical, rec)
        case "high":
            recs.High = append(recs.High, rec)
        case "medium":
            recs.Medium = append(recs.Medium, rec)
        default:
            recs.Future = append(recs.Future, rec)
        }
    }

    // Add domain-specific recommendations based on metrics
    if result.Metrics.TestCoverage < 0.5 {
        recs.Critical = append(recs.Critical, Recommendation{
            Action:        "Add unit tests to reach 80% coverage",
            Reason:        "Critical code with insufficient test coverage",
            EstimatedTime: 45,
            AutoFixable:   true,
            Priority:      "critical",
        })
    }

    return recs
}

func determinePriority(confidence, riskReduction float64) string {
    score := confidence * riskReduction
    if score > 0.5 {
        return "critical"
    } else if score > 0.3 {
        return "high"
    } else if score > 0.15 {
        return "medium"
    }
    return "low"
}

func determineBlockReason(result *risk.AssessmentResult) string {
    reasons := []string{}

    if result.Metrics.IncidentCount > 0 {
        reasons = append(reasons, "Previously caused incidents")
    }
    if result.Metrics.Coupling > 15 {
        reasons = append(reasons, "High coupling (blast radius)")
    }
    if result.Metrics.TestCoverage < 0.3 {
        reasons = append(reasons, "Critically low test coverage")
    }

    if len(reasons) == 0 {
        return "High risk detected"
    }

    return strings.Join(reasons, "; ")
}

func convertInvestigationToTrace(investigation *Investigation) []InvestigationHop {
    trace := make([]InvestigationHop, len(investigation.Hops))
    for i, hop := range investigation.Hops {
        trace[i] = InvestigationHop{
            Hop:               i + 1,
            NodeType:          hop.NodeType,
            NodeID:            hop.NodeID,
            Action:            hop.Action,
            MetricsCalculated: convertMetrics(hop.MetricsCalculated),
            Decision:          hop.NextAction,
            Reasoning:         hop.Reasoning,
            Confidence:        hop.Confidence,
            DurationMS:        hop.Duration.Milliseconds(),
        }
    }
    return trace
}

func convertMetrics(metrics []Metric) []MetricResult {
    results := make([]MetricResult, len(metrics))
    for i, m := range metrics {
        results[i] = MetricResult{
            Name:  m.Name,
            Value: m.Value,
        }
    }
    return results
}
```

---

## Testing Instructions

### Test 1: AI Mode Output Structure

```bash
# Generate AI mode output
./crisk check --ai-mode apps/web/src/app/page.tsx > ai_output.json

# Validate JSON structure
jq '.' ai_output.json

# Check required fields exist
jq 'keys | sort' ai_output.json
# Expected: ["ai_assistant_actions", "branch", "contextual_insights", "files", "graph_analysis", "investigation_trace", "overall_risk", "recommendations", "repository", "timestamp"]

# Verify ai_assistant_actions is populated
jq '.ai_assistant_actions | length' ai_output.json
# Expected: >0 (if issues detected)

# Verify graph_analysis
jq '.graph_analysis.blast_radius.total_affected_files' ai_output.json
# Expected: >0 (if file has dependencies)

jq '.graph_analysis.temporal_coupling | length' ai_output.json
# Expected: >0 (if CO_CHANGED edges exist - requires Gap A1 fix first)
```

### Test 2: AI Assistant Action Prompts

```bash
# Check action prompts are ready-to-execute
jq '.ai_assistant_actions[0]' ai_output.json

# Expected format:
# {
#   "action_type": "add_test",
#   "confidence": 0.9,
#   "description": "Add unit tests to improve coverage...",
#   "estimated_lines": 25,
#   "file_path": "apps/web/src/app/page.tsx",
#   "prompt": "Add comprehensive unit tests for...",  ← Should be full, actionable prompt
#   "ready_to_execute": true,
#   "risk_reduction": 0.3
# }
```

### Test 3: Recommendations

```bash
# Check recommendations are categorized
jq '.recommendations.critical | length' ai_output.json
# Expected: >0 (for high-risk files)

jq '.recommendations.critical[0]' ai_output.json
# Expected:
# {
#   "action": "Add unit tests...",
#   "reason": "...",
#   "estimated_time": 45,
#   "auto_fixable": true,
#   "priority": "critical"
# }
```

### Test 4: Compare with Spec

```bash
# Extract spec schema
cat dev_docs/00-product/developer_experience.md | \
  sed -n '/```json/,/```/p' | \
  grep -v '```' > spec_schema.json

# Compare actual vs spec
diff <(jq 'keys | sort' spec_schema.json) <(jq 'keys | sort' ai_output.json)
# Expected: No differences
```

---

## Validation Criteria

**Success Criteria:**
- [ ] ✅ All required JSON fields present (matches spec)
- [ ] ✅ `ai_assistant_actions[]` populated with actionable prompts
- [ ] ✅ Graph analysis includes blast radius, temporal coupling, hotspots
- [ ] ✅ Recommendations categorized (critical, high, medium, future)
- [ ] ✅ Confidence scores and estimated times included
- [ ] ✅ Auto-fixable flags set correctly
- [ ] ✅ Investigation trace included (when Phase 2 runs)

**Code Quality:**
- [ ] ✅ 12-factor citation (Factor 4: Tools are structured outputs)
- [ ] ✅ Graph queries optimized (<50ms each)
- [ ] ✅ Error handling for missing graph data
- [ ] ✅ Structured logging for debugging

---

## Commit Message Template

```
Complete AI Mode JSON output schema

Implements full AI assistant integration schema per developer_experience.md.
Generates ready-to-execute prompts, graph analysis, and recommendations.

12-factor: Factor 4 - Tools are structured outputs

**Added:**
- AI assistant action generation with confidence scores
- Blast radius calculation from graph (1-3 hop queries)
- Temporal coupling data from CO_CHANGED edges
- Hotspot identification (churn + coverage + incidents)
- Contextual insights (file reputation, similar changes)
- Categorized recommendations (critical/high/medium/future)

**Enhanced Types:**
- AIAssistantAction with ready-to-execute prompts
- GraphAnalysis with blast radius, coupling, hotspots
- Recommendations with auto-fixable flags
- InvestigationHop for Phase 2 trace

Fixes: Gap UX1 (AI Mode incomplete)
Tests: Validated against spec schema
Performance: Graph queries <50ms each

- internal/output/types.go: Complete schema
- internal/output/ai_actions.go: Generate AI prompts
- internal/output/graph_analysis.go: Graph queries
- internal/output/converter.go: Enhanced ToAIMode()
```

---

## Success Validation

```bash
# Re-run Test 4.2 from E2E report
./crisk check --ai-mode apps/web/src/app/page.tsx | jq '.' > final_output.json

# Verify against spec
jq 'keys | sort' final_output.json
# Expected: All keys from developer_experience.md present ✅

# Verify AI actions
jq '.ai_assistant_actions | map(.ready_to_execute) | unique' final_output.json
# Expected: [true, false] (mix of auto-fixable and review-required) ✅

# Verify graph analysis
jq '.graph_analysis | keys' final_output.json
# Expected: ["blast_radius", "hotspots", "temporal_coupling"] ✅
```

**This task is COMPLETE when:**
- AI Mode JSON matches spec 100% ✅
- AI assistants can parse and use the output ✅
- Auto-fix prompts are actionable ✅
- Performance <200ms total for JSON generation ✅
