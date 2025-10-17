# Session 3 Prompt: AI Mode Output & Prompt Generation (Level 4)

**Duration:** 4-5 days
**Owner:** Claude Code Session 3
**Dependencies:** Formatter interface from Session 2

---

## Context

You are implementing the **AI Mode Output & Prompt Generation** component of CodeRisk's Developer Experience (DX) Foundation phase. This is **Session 3 of 3 parallel sessions** working on different parts of the codebase simultaneously.

**Your role:** Create machine-readable JSON output for AI assistants (Claude Code, Cursor, Copilot) with ready-to-execute fix prompts.

**Other sessions (DO NOT MODIFY THEIR FILES):**
- Session 1: Building pre-commit hook in `cmd/crisk/hook.go` and `internal/git/`
- Session 2: Building adaptive verbosity in `internal/output/` (creates formatter.go interface you'll use)

**DEPENDENCY:** Wait for Session 2 to create `internal/output/formatter.go` before implementing your AIFormatter.

---

## High-Level Goal

Implement AI Mode (Level 4 verbosity) that outputs:
1. **Machine-readable JSON** following schema v1.0
2. **Ready-to-execute AI prompts** for auto-fixing issues
3. **Confidence scores** to determine auto-fix safety
4. **Rich context** (graph analysis, history, team patterns)

**Key difference from human modes:** AI Mode includes actionable prompts AI assistants can execute to improve code **before the user sees it**.

---

## Your File Ownership

### Files YOU CREATE (your responsibility):
- `internal/output/ai_mode.go` - Level 4 formatter (implements formatter.go interface)
- `internal/ai/prompt_generator.go` - Generate fix prompts per issue type
- `internal/ai/confidence.go` - Calculate auto-fix confidence scores
- `internal/ai/templates.go` - Prompt templates for common fixes
- `schemas/ai-mode-v1.0.json` - JSON schema definition
- `test/integration/test_ai_mode.sh` - Integration tests

### Files YOU MODIFY:
- `cmd/crisk/check.go` - Add `--ai-mode` flag only (~10 lines)
- `internal/models/risk_result.go` - Extend with AI-specific fields

### Files YOU MUST WAIT FOR (Session 2 creates):
- `internal/output/formatter.go` - **WAIT for this!** You'll implement the Formatter interface

### Files YOU READ ONLY (do not modify):
- `internal/output/quiet.go` - Session 2's Level 1
- `internal/output/standard.go` - Session 2's Level 2
- `internal/output/explain.go` - Session 2's Level 3

---

## Reading List (READ THESE FIRST)

**MUST READ before coding:**
1. `dev_docs/03-implementation/integration_guides/ux_ai_mode_output.md` - Your implementation guide (full JSON schema)
2. `dev_docs/03-implementation/PARALLEL_SESSION_PLAN.md` - Coordination with other sessions
3. `dev_docs/00-product/developer_experience.md` - UX requirements (§2.4 AI-Assistant Mode)
4. `dev_docs/03-implementation/phases/phase_dx_foundation.md` - Phase overview

**Reference as needed:**
5. `dev_docs/DEVELOPMENT_WORKFLOW.md` - Go development guardrails
6. `internal/models/risk_result.go` - Data model you'll extend

---

## Step-by-Step Implementation Plan

### Step 1: Read Documentation (1 hour)
- [ ] Read all files in "Reading List" section above
- [ ] Study the full JSON schema in `ux_ai_mode_output.md` (this is your spec!)
- [ ] Understand prompt generation requirements
- [ ] Review confidence scoring thresholds (>0.85 = auto-fix)
- [ ] **ASK USER:** "I've read the documentation and studied the AI Mode JSON schema. Should I proceed?"

---

### Step 2: Wait for Session 2 Interface (DO NOT SKIP!)

**CRITICAL:** Session 2 must create `internal/output/formatter.go` before you can proceed.

**Check if interface exists:**
```bash
ls -la internal/output/formatter.go
```

**If file doesn't exist:**
**ASK USER:** "Session 2 hasn't created internal/output/formatter.go yet. Should I wait or can I proceed assuming the interface structure from the plan?"

**If file exists, verify interface:**
```go
// Expected interface from Session 2:
type Formatter interface {
    Format(result *models.RiskResult, w io.Writer) error
}

type VerbosityLevel int

const (
    VerbosityQuiet VerbosityLevel = iota
    VerbosityStandard
    VerbosityExplain
    VerbosityAIMode  // <- You'll implement this
)
```

**ASK USER (CHECKPOINT):** "✅ Formatter interface found at internal/output/formatter.go. Interface verified. Should I proceed with extending the RiskResult model?"

---

### Step 3: Extend Risk Result Model (2-3 hours)

**File: `internal/models/risk_result.go` (EXTEND, don't replace)**

**Add AI-specific fields to Issue struct:**
```go
type Issue struct {
    // ... existing fields (don't modify) ...

    // AI Mode specific fields (YOU ADD THESE)
    AutoFixable       bool     `json:"auto_fixable"`
    FixType           string   `json:"fix_type"` // "generate_tests", "add_error_handling", etc.
    FixConfidence     float64  `json:"fix_confidence"`
    AIPromptTemplate  string   `json:"ai_prompt_template"`
    ExpectedFiles     []string `json:"expected_files"`
    EstimatedLines    int      `json:"estimated_lines"`
}
```

**Add AI-specific fields to RiskResult struct:**
```go
type RiskResult struct {
    // ... existing fields (don't modify) ...

    // Commit control (YOU ADD THESE)
    ShouldBlock                   bool   `json:"should_block_commit"`
    BlockReason                   string `json:"block_reason"`
    OverrideAllowed               bool   `json:"override_allowed"`
    OverrideRequiresJustification bool   `json:"override_requires_justification"`

    // Performance metrics (YOU ADD THESE)
    Performance Performance `json:"performance"`
}

// New struct (YOU ADD THIS)
type Performance struct {
    TotalDurationMS int                    `json:"total_duration_ms"`
    Breakdown       map[string]int         `json:"breakdown"`
    CacheEfficiency map[string]interface{} `json:"cache_efficiency"`
}
```

**Checkpoint:** Verify model compiles
```bash
go build ./internal/models
```

**ASK USER (CHECKPOINT):** "✅ Extended internal/models/risk_result.go with AI-specific fields. No conflicts with existing fields. Model compiles. Should I proceed with AI Mode formatter?"

---

### Step 4: Implement AI Mode Formatter (4-5 hours)

**File: `internal/output/ai_mode.go`**

**Requirements from ux_ai_mode_output.md:**
- Output valid JSON following schema v1.0
- Include all 10 schema sections (meta, risk, files, graph_analysis, etc.)
- Generate `ai_assistant_actions[]` array with ready-to-execute prompts
- Include confidence scores and auto-fix flags

```go
package output

import (
    "encoding/json"
    "io"
    "time"

    "github.com/coderisk/coderisk-go/internal/models"
)

// AIFormatter outputs machine-readable JSON for AI assistants
// Schema v1.0: https://coderisk.com/schemas/ai-mode/v1.0.json
type AIFormatter struct {
    Version string
}

func NewAIFormatter() *AIFormatter {
    return &AIFormatter{Version: "1.0"}
}

func (f *AIFormatter) Format(result *models.RiskResult, w io.Writer) error {
    output := f.buildAIModeOutput(result)

    encoder := json.NewEncoder(w)
    encoder.SetIndent("", "  ")
    return encoder.Encode(output)
}

func (f *AIFormatter) buildAIModeOutput(result *models.RiskResult) map[string]interface{} {
    return map[string]interface{}{
        "meta":                  f.buildMeta(result),
        "risk":                  f.buildRisk(result),
        "files":                 f.buildFiles(result),
        "graph_analysis":        f.buildGraphAnalysis(result),
        "investigation_trace":   f.buildTrace(result),
        "recommendations":       f.buildRecommendations(result),
        "ai_assistant_actions":  f.buildAIActions(result),
        "contextual_insights":   f.buildInsights(result),
        "performance":           f.buildPerformance(result),
        "should_block_commit":   result.ShouldBlock,
        "block_reason":          result.BlockReason,
        "override_allowed":      result.OverrideAllowed,
        "override_requires_justification": result.OverrideRequiresJustification,
    }
}

func (f *AIFormatter) buildMeta(result *models.RiskResult) map[string]interface{} {
    return map[string]interface{}{
        "version":        f.Version,
        "timestamp":      time.Now().Format(time.RFC3339),
        "duration_ms":    result.Duration.Milliseconds(),
        "branch":         result.Branch,
        "files_analyzed": result.FilesChanged,
        "agent_hops":     len(result.InvestigationTrace),
        "cache_hit":      result.CacheHit,
    }
}

func (f *AIFormatter) buildRisk(result *models.RiskResult) map[string]interface{} {
    return map[string]interface{}{
        "level":      result.RiskLevel,
        "score":      result.RiskScore,
        "confidence": result.Confidence,
    }
}

func (f *AIFormatter) buildFiles(result *models.RiskResult) []map[string]interface{} {
    files := []map[string]interface{}{}

    for _, file := range result.Files {
        fileData := map[string]interface{}{
            "path":          file.Path,
            "language":      file.Language,
            "lines_changed": file.LinesChanged,
            "risk_score":    file.RiskScore,
            "metrics":       file.Metrics,
            "issues":        f.transformIssues(file.Issues),
            "dependencies":  file.Dependencies,
            "history":       file.History,
            "incidents":     file.Incidents,
        }
        files = append(files, fileData)
    }

    return files
}

func (f *AIFormatter) transformIssues(issues []models.Issue) []map[string]interface{} {
    transformed := []map[string]interface{}{}

    for _, issue := range issues {
        issueData := map[string]interface{}{
            "id":                   issue.ID,
            "severity":             issue.Severity,
            "category":             issue.Category,
            "line_start":           issue.LineStart,
            "line_end":             issue.LineEnd,
            "function":             issue.Function,
            "message":              issue.Message,
            "impact_score":         issue.ImpactScore,
            "fix_priority":         issue.FixPriority,
            "estimated_fix_time_min": issue.EstimatedFixTimeMin,
            "auto_fixable":         issue.AutoFixable,
            "fix_command":          issue.FixCommand,
        }
        transformed = append(transformed, issueData)
    }

    return transformed
}

func (f *AIFormatter) buildGraphAnalysis(result *models.RiskResult) map[string]interface{} {
    return map[string]interface{}{
        "blast_radius":      result.BlastRadius,
        "temporal_coupling": result.TemporalCoupling,
        "hotspots":         result.Hotspots,
    }
}

func (f *AIFormatter) buildTrace(result *models.RiskResult) []map[string]interface{} {
    trace := []map[string]interface{}{}

    for _, hop := range result.InvestigationTrace {
        hopData := map[string]interface{}{
            "hop":                hop.Hop,
            "node_type":          hop.NodeType,
            "node_id":            hop.NodeID,
            "action":             hop.Action,
            "metrics_calculated": hop.MetricsCalculated,
            "decision":           hop.Decision,
            "reasoning":          hop.Reasoning,
            "confidence":         hop.Confidence,
            "duration_ms":        hop.DurationMS,
        }
        trace = append(trace, hopData)
    }

    return trace
}

func (f *AIFormatter) buildRecommendations(result *models.RiskResult) map[string]interface{} {
    return map[string]interface{}{
        "critical": f.filterRecommendations(result.Recommendations, "critical"),
        "high":     f.filterRecommendations(result.Recommendations, "high"),
        "medium":   f.filterRecommendations(result.Recommendations, "medium"),
    }
}

func (f *AIFormatter) filterRecommendations(recs []models.Recommendation, priority string) []map[string]interface{} {
    filtered := []map[string]interface{}{}

    for _, rec := range recs {
        if rec.Priority == priority {
            recData := map[string]interface{}{
                "priority":           rec.Priority,
                "action":             rec.Action,
                "target":             rec.Target,
                "reason":             rec.Reason,
                "estimated_time_min": rec.EstimatedTimeMin,
                "auto_fixable":       rec.AutoFixable,
                "suggestions":        rec.Suggestions,
            }
            filtered = append(filtered, recData)
        }
    }

    return filtered
}

func (f *AIFormatter) buildAIActions(result *models.RiskResult) []map[string]interface{} {
    actions := []map[string]interface{}{}

    for _, issue := range result.Issues {
        if issue.AutoFixable {
            action := map[string]interface{}{
                "action_type":      issue.FixType,
                "confidence":       issue.FixConfidence,
                "ready_to_execute": issue.FixConfidence > 0.85,
                "prompt":           issue.AIPromptTemplate,
                "expected_files":   issue.ExpectedFiles,
                "estimated_lines":  issue.EstimatedLines,
            }

            if issue.FixConfidence <= 0.85 {
                action["reason"] = "confidence_below_threshold"
            }

            actions = append(actions, action)
        }
    }

    return actions
}

func (f *AIFormatter) buildInsights(result *models.RiskResult) map[string]interface{} {
    return map[string]interface{}{
        "similar_past_changes": result.SimilarPastChanges,
        "team_patterns":        result.TeamPatterns,
        "file_reputation":      result.FileReputation,
    }
}

func (f *AIFormatter) buildPerformance(result *models.RiskResult) map[string]interface{} {
    return map[string]interface{}{
        "total_duration_ms": result.Performance.TotalDurationMS,
        "breakdown":         result.Performance.Breakdown,
        "cache_efficiency":  result.Performance.CacheEfficiency,
    }
}
```

**Checkpoint:** Build AI Mode formatter
```bash
go build ./internal/output
```

**ASK USER:** "AI Mode formatter implemented. Outputs JSON following schema v1.0. Should I proceed with prompt generation?"

---

### Step 5: Implement Prompt Generator (3-4 hours)

**File: `internal/ai/prompt_generator.go`**

**Requirements:**
- Generate ready-to-execute prompts for common fix types
- Include context from graph analysis
- Be specific to the project/language

```go
package ai

import (
    "fmt"
    "strings"
    "github.com/coderisk/coderisk-go/internal/models"
)

// PromptGenerator generates AI-executable prompts for fixes
type PromptGenerator struct{}

func NewPromptGenerator() *PromptGenerator {
    return &PromptGenerator{}
}

// GeneratePrompt creates a prompt for the given issue
func (g *PromptGenerator) GeneratePrompt(issue models.Issue, file models.FileRisk) string {
    switch issue.FixType {
    case "generate_tests":
        return g.generateTestPrompt(issue, file)
    case "add_error_handling":
        return g.generateErrorHandlingPrompt(issue, file)
    case "reduce_coupling":
        return g.generateCouplingPrompt(issue, file)
    case "fix_security":
        return g.generateSecurityPrompt(issue, file)
    default:
        return ""
    }
}

func (g *PromptGenerator) generateTestPrompt(issue models.Issue, file models.FileRisk) string {
    return fmt.Sprintf(`Generate comprehensive unit and integration tests for %s in %s (lines %d-%d).

Coverage requirements:
- Happy path: Valid inputs, expected outputs
- Edge cases: Boundary conditions, null values
- Error cases: Invalid inputs, exceptions
- Integration: Dependencies and external calls

Framework: Use %s testing framework matching project conventions.

Context:
- File language: %s
- Current coverage: %.1f%%
- Function: %s

Generate tests in separate test file following naming convention: %s`,
        issue.Function,
        file.Path,
        issue.LineStart,
        issue.LineEnd,
        g.detectTestFramework(file.Language),
        file.Language,
        file.Metrics["test_coverage"],
        issue.Function,
        g.testFileName(file.Path),
    )
}

func (g *PromptGenerator) generateErrorHandlingPrompt(issue models.Issue, file models.FileRisk) string {
    return fmt.Sprintf(`Add robust error handling to %s in %s (lines %d-%d).

Requirements:
- Add try/catch or error checks for network calls
- Implement exponential backoff retry logic
- Handle specific errors: TimeoutError, ConnectionError, HTTPError
- Log errors with context for debugging

Use these patterns:
- Library: %s (if available, otherwise implement custom)
- Max retries: 3
- Backoff: 100ms, 200ms, 400ms

Context:
- Language: %s
- Similar incident history: %s`,
        issue.Function,
        file.Path,
        issue.LineStart,
        issue.LineEnd,
        g.detectRetryLibrary(file.Language),
        file.Language,
        g.formatIncidentHistory(file.Incidents),
    )
}

func (g *PromptGenerator) generateCouplingPrompt(issue models.Issue, file models.FileRisk) string {
    coupledFiles := []string{}
    for _, dep := range file.Dependencies {
        coupledFiles = append(coupledFiles, dep.File)
    }

    return fmt.Sprintf(`Suggest refactoring options to reduce coupling between %s and %s.

Current coupling:
- Co-change frequency: %.1f%% (high)
- Direct dependencies: %d
- Temporal coupling strength: %.2f

Consider these patterns:
1. Repository pattern - Extract data access interface
2. Dependency injection - Inject dependencies via constructor
3. Interface segregation - Define minimal interfaces

Provide 2-3 refactoring options with:
- Pros/cons of each approach
- Estimated effort (time)
- Risk level

Context:
- Language: %s
- Project architecture: %s`,
        file.Path,
        strings.Join(coupledFiles, ", "),
        issue.ImpactScore * 10,
        len(file.Dependencies),
        issue.ImpactScore / 10,
        file.Language,
        "microservices", // TODO: detect from project structure
    )
}

func (g *PromptGenerator) generateSecurityPrompt(issue models.Issue, file models.FileRisk) string {
    return fmt.Sprintf(`Fix security vulnerability in %s (lines %d-%d).

Issue: %s
Severity: %s

Security requirements:
- Input validation: Sanitize all user inputs
- Authentication: Verify user permissions
- Encryption: Use secure protocols (TLS 1.2+)
- Secrets: Use environment variables, not hardcoded

Best practices for %s:
%s

Context:
- File: %s
- Function: %s`,
        file.Path,
        issue.LineStart,
        issue.LineEnd,
        issue.Message,
        issue.Severity,
        file.Language,
        g.securityBestPractices(file.Language),
        file.Path,
        issue.Function,
    )
}

// Helper functions
func (g *PromptGenerator) detectTestFramework(language string) string {
    frameworks := map[string]string{
        "python":     "pytest",
        "javascript": "jest",
        "typescript": "jest",
        "go":         "testing (Go standard)",
        "java":       "JUnit",
    }
    if fw, ok := frameworks[language]; ok {
        return fw
    }
    return "appropriate testing framework"
}

func (g *PromptGenerator) detectRetryLibrary(language string) string {
    libraries := map[string]string{
        "python":     "tenacity",
        "javascript": "axios-retry",
        "typescript": "axios-retry",
        "go":         "go-retryablehttp",
    }
    if lib, ok := libraries[language]; ok {
        return lib
    }
    return "custom retry implementation"
}

func (g *PromptGenerator) testFileName(filePath string) string {
    ext := filepath.Ext(filePath)
    base := strings.TrimSuffix(filePath, ext)
    return fmt.Sprintf("%s_test%s", base, ext)
}

func (g *PromptGenerator) formatIncidentHistory(incidents []models.Incident) string {
    if len(incidents) == 0 {
        return "No recent incidents"
    }

    summaries := []string{}
    for _, inc := range incidents {
        summaries = append(summaries, fmt.Sprintf("%s (%s)", inc.Summary, inc.Date))
    }
    return strings.Join(summaries, "; ")
}

func (g *PromptGenerator) securityBestPractices(language string) string {
    practices := map[string]string{
        "python":     "- Use parameterized queries (SQLAlchemy)\n- Validate with pydantic\n- OWASP Top 10 compliance",
        "javascript": "- Sanitize with DOMPurify\n- Use helmet.js for headers\n- CSRF tokens required",
        "go":         "- Use prepared statements\n- bcrypt for passwords\n- TLS 1.3 minimum",
    }
    if p, ok := practices[language]; ok {
        return p
    }
    return "- Follow OWASP guidelines\n- Input validation\n- Least privilege principle"
}
```

**Checkpoint:** Test prompt generation
**ASK USER:** "Prompt generator implemented. Generates prompts for 4 fix types (tests, error handling, coupling, security). Should I proceed with confidence scoring?"

---

### Step 6: Implement Confidence Scoring (2-3 hours)

**File: `internal/ai/confidence.go`**

**Requirements:**
- Calculate confidence based on issue type, complexity, and context
- Threshold: >0.85 = ready_to_execute

```go
package ai

import (
    "github.com/coderisk/coderisk-go/internal/models"
)

// ConfidenceCalculator determines auto-fix confidence
type ConfidenceCalculator struct{}

func NewConfidenceCalculator() *ConfidenceCalculator {
    return &ConfidenceCalculator{}
}

// Calculate computes confidence score (0.0-1.0) for auto-fixing
func (c *ConfidenceCalculator) Calculate(issue models.Issue, file models.FileRisk) float64 {
    baseConfidence := c.baseConfidenceByType(issue.FixType)

    // Adjust based on context
    complexityPenalty := c.complexityPenalty(file.Metrics["complexity"])
    historicalBonus := c.historicalBonus(file.Incidents)
    languageBonus := c.languageBonus(file.Language)

    confidence := baseConfidence + complexityPenalty + historicalBonus + languageBonus

    // Clamp to [0.0, 1.0]
    if confidence > 1.0 {
        confidence = 1.0
    }
    if confidence < 0.0 {
        confidence = 0.0
    }

    return confidence
}

func (c *ConfidenceCalculator) baseConfidenceByType(fixType string) float64 {
    // Base confidence by fix type
    baseScores := map[string]float64{
        "generate_tests":      0.92, // High confidence (well-defined task)
        "add_error_handling":  0.85, // Medium-high (straightforward)
        "fix_security":        0.80, // Medium (requires domain knowledge)
        "reduce_coupling":     0.65, // Low-medium (architectural decision)
        "reduce_complexity":   0.60, // Low (subjective refactoring)
    }

    if score, ok := baseScores[fixType]; ok {
        return score
    }
    return 0.50 // Default: requires review
}

func (c *ConfidenceCalculator) complexityPenalty(complexity interface{}) float64 {
    if complexity == nil {
        return 0.0
    }

    complexityValue, ok := complexity.(float64)
    if !ok {
        return 0.0
    }

    // Penalty for high complexity
    if complexityValue > 15 {
        return -0.10 // High complexity = lower confidence
    } else if complexityValue > 10 {
        return -0.05 // Medium complexity = slight penalty
    }
    return 0.0
}

func (c *ConfidenceCalculator) historicalBonus(incidents []models.Incident) float64 {
    if len(incidents) == 0 {
        return 0.05 // No incidents = slightly higher confidence
    }

    // More incidents = more context = higher confidence
    if len(incidents) >= 3 {
        return 0.10 // Learn from past mistakes
    } else if len(incidents) >= 1 {
        return 0.05
    }

    return 0.0
}

func (c *ConfidenceCalculator) languageBonus(language string) float64 {
    // Higher confidence for well-supported languages
    supportedLanguages := map[string]float64{
        "python":     0.05,
        "javascript": 0.05,
        "typescript": 0.05,
        "go":         0.03,
        "java":       0.03,
    }

    if bonus, ok := supportedLanguages[language]; ok {
        return bonus
    }
    return -0.05 // Lower confidence for less common languages
}

// ShouldAutoFix determines if confidence is high enough for auto-fix
func (c *ConfidenceCalculator) ShouldAutoFix(confidence float64) bool {
    return confidence > 0.85
}
```

**Checkpoint:** Test confidence calculation
**ASK USER:** "Confidence scoring implemented. Thresholds: >0.85 = auto-fix, 0.6-0.85 = review, <0.6 = manual. Should I proceed with prompt templates?"

---

### Step 7: Create Prompt Templates (1-2 hours)

**File: `internal/ai/templates.go`**

**Purpose:** Store reusable prompt templates

```go
package ai

// PromptTemplates contains reusable AI prompt templates
var PromptTemplates = map[string]string{
    "generate_tests": `Generate comprehensive unit and integration tests for {{.Function}} in {{.File}} (lines {{.LineStart}}-{{.LineEnd}}).

Coverage requirements:
- Happy path: Valid inputs, expected outputs
- Edge cases: Boundary conditions, null values
- Error cases: Invalid inputs, exceptions
- Integration: Dependencies and external calls

Framework: Use {{.TestFramework}} matching project conventions.

Context:
- File language: {{.Language}}
- Current coverage: {{.Coverage}}%
- Function: {{.Function}}`,

    "add_error_handling": `Add robust error handling to {{.Function}} in {{.File}} (lines {{.LineStart}}-{{.LineEnd}}).

Requirements:
- Add try/catch or error checks for network calls
- Implement exponential backoff retry logic
- Handle specific errors: TimeoutError, ConnectionError, HTTPError
- Log errors with context

Use: {{.RetryLibrary}} (if available, otherwise custom)`,

    "reduce_coupling": `Suggest refactoring options to reduce coupling between {{.File}} and {{.CoupledWith}}.

Current coupling:
- Co-change frequency: {{.CoChangePercent}}%
- Temporal coupling strength: {{.CouplingStrength}}

Provide 2-3 options with pros/cons.`,

    "fix_security": `Fix security vulnerability in {{.File}} (lines {{.LineStart}}-{{.LineEnd}}).

Issue: {{.Message}}
Severity: {{.Severity}}

Apply security best practices for {{.Language}}.`,
}
```

**ASK USER:** "Prompt templates created. Should I proceed with CLI integration?"

---

### Step 8: Integrate with Check Command (1 hour)

**File: `cmd/crisk/check.go` (MINIMAL CHANGES)**

Add ONE flag:
```go
func init() {
    checkCmd.Flags().Bool("ai-mode", false, "Output machine-readable JSON for AI assistants")
    // ... existing flags from Session 1 & 2 ...

    // Mutually exclusive with other verbosity flags
    checkCmd.MarkFlagsMutuallyExclusive("quiet", "explain", "ai-mode")
}
```

Modify `runCheck()` to support AI mode:
```go
func runCheck(cmd *cobra.Command, args []string) error {
    // ... existing logic ...

    // Determine verbosity level
    var level output.VerbosityLevel
    aiMode, _ := cmd.Flags().GetBool("ai-mode")

    if aiMode {
        level = output.VerbosityAIMode
    } else {
        // ... existing logic from Session 2 ...
    }

    // Generate AI prompts and confidence scores if AI mode
    if level == output.VerbosityAIMode {
        promptGen := ai.NewPromptGenerator()
        confCalc := ai.NewConfidenceCalculator()

        // Enhance issues with AI data
        for i := range result.Issues {
            file := findFile(result.Files, result.Issues[i].File)

            result.Issues[i].AIPromptTemplate = promptGen.GeneratePrompt(result.Issues[i], file)
            result.Issues[i].FixConfidence = confCalc.Calculate(result.Issues[i], file)
            result.Issues[i].AutoFixable = result.Issues[i].AIPromptTemplate != ""
        }
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
./bin/crisk check --ai-mode <file> | jq .
```

**ASK USER:** "CLI integration complete. AI mode flag added. Should I proceed with JSON schema validation?"

---

### Step 9: Create JSON Schema (1-2 hours)

**File: `schemas/ai-mode-v1.0.json`**

**Purpose:** JSON schema for validation and documentation

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "CodeRisk AI Mode Output",
  "version": "1.0",
  "type": "object",
  "required": ["meta", "risk", "files"],
  "properties": {
    "meta": {
      "type": "object",
      "properties": {
        "version": {"type": "string"},
        "timestamp": {"type": "string", "format": "date-time"},
        "duration_ms": {"type": "integer"},
        "branch": {"type": "string"},
        "files_analyzed": {"type": "integer"},
        "agent_hops": {"type": "integer"},
        "cache_hit": {"type": "boolean"}
      },
      "required": ["version", "timestamp", "duration_ms"]
    },
    "risk": {
      "type": "object",
      "properties": {
        "level": {
          "type": "string",
          "enum": ["NONE", "LOW", "MEDIUM", "HIGH", "CRITICAL"]
        },
        "score": {"type": "number", "minimum": 0, "maximum": 10},
        "confidence": {"type": "number", "minimum": 0, "maximum": 1}
      },
      "required": ["level", "score"]
    },
    "ai_assistant_actions": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "action_type": {"type": "string"},
          "confidence": {"type": "number", "minimum": 0, "maximum": 1},
          "ready_to_execute": {"type": "boolean"},
          "prompt": {"type": "string"},
          "expected_files": {"type": "array", "items": {"type": "string"}},
          "estimated_lines": {"type": "integer"}
        },
        "required": ["action_type", "confidence", "ready_to_execute", "prompt"]
      }
    },
    "should_block_commit": {"type": "boolean"},
    "block_reason": {"type": "string"},
    "override_allowed": {"type": "boolean"}
  }
}
```

**Checkpoint:** Validate output against schema
```bash
./bin/crisk check --ai-mode <file> > output.json
ajv validate -s schemas/ai-mode-v1.0.json -d output.json
```

**ASK USER:** "JSON schema created. Should I validate AI mode output against it?"

---

### Step 10: Integration Testing (2-3 hours)

**File: `test/integration/test_ai_mode.sh`**

```bash
#!/bin/bash
set -e

echo "=== AI Mode Integration Test ==="

# Build binary
go build -o bin/crisk ./cmd/crisk

# Test file (mock risky file)
TEST_FILE="test_auth.go"
echo "package main\nfunc auth() {}" > $TEST_FILE

# Test 1: AI Mode outputs valid JSON
echo "Test 1: AI Mode JSON validation..."
OUTPUT=$(./bin/crisk check --ai-mode $TEST_FILE)
if echo "$OUTPUT" | jq . > /dev/null 2>&1; then
    echo "✅ PASS: Valid JSON output"
else
    echo "❌ FAIL: Invalid JSON"
    exit 1
fi

# Test 2: Schema validation
echo "Test 2: Schema validation..."
echo "$OUTPUT" > /tmp/ai_mode_output.json
if ajv validate -s schemas/ai-mode-v1.0.json -d /tmp/ai_mode_output.json > /dev/null 2>&1; then
    echo "✅ PASS: Output matches schema v1.0"
else
    echo "❌ FAIL: Schema validation failed"
    exit 1
fi

# Test 3: AI actions array exists
echo "Test 3: AI actions array..."
if echo "$OUTPUT" | jq '.ai_assistant_actions' | grep -q "action_type"; then
    echo "✅ PASS: AI actions array present"
else
    echo "❌ FAIL: AI actions missing"
    exit 1
fi

# Test 4: Confidence scoring
echo "Test 4: Confidence scores..."
if echo "$OUTPUT" | jq '.ai_assistant_actions[0].confidence' | grep -q "[0-9]"; then
    echo "✅ PASS: Confidence scores present"
else
    echo "❌ FAIL: Confidence scores missing"
    exit 1
fi

# Test 5: Ready to execute flag
echo "Test 5: Ready to execute flag..."
READY=$(echo "$OUTPUT" | jq '.ai_assistant_actions[0].ready_to_execute')
if [ "$READY" == "true" ] || [ "$READY" == "false" ]; then
    echo "✅ PASS: ready_to_execute flag set"
else
    echo "❌ FAIL: ready_to_execute flag missing"
    exit 1
fi

# Cleanup
rm -f $TEST_FILE /tmp/ai_mode_output.json

echo "=== All AI Mode tests passed! ==="
```

**Make executable:** `chmod +x test/integration/test_ai_mode.sh`

**Checkpoint:** Run integration tests
**ASK USER:** "Integration test ready. Should I execute: `./test/integration/test_ai_mode.sh`?"

---

### Step 11: Final Validation & Documentation (1-2 hours)

**Validation checklist:**
- [ ] Run `go build ./internal/output ./internal/ai` - Verify compiles
- [ ] Run `./bin/crisk check --ai-mode <file> | jq .` - Verify JSON output
- [ ] Validate against schema: `ajv validate -s schemas/ai-mode-v1.0.json -d output.json`
- [ ] Test prompt generation for each fix type
- [ ] Verify confidence scores are in range [0.0, 1.0]
- [ ] Check `ready_to_execute` flag is set correctly (confidence > 0.85)

**Output validation:**
**ASK USER:** "AI Mode output example:
[paste JSON output with jq formatting]

Validation results:
- JSON valid: YES/NO
- Schema v1.0: PASS/FAIL
- AI actions present: YES/NO
- Confidence scores valid: YES/NO

Is this output acceptable per the design spec?"

**Performance check:**
- [ ] Measure AI Mode overhead: Compare `crisk check` vs `crisk check --ai-mode`
- [ ] Target: <200ms overhead for AI mode generation
- [ ] **ASK USER:** "AI Mode overhead: [X]ms. Target: <200ms. Is this acceptable?"

**Documentation:**
- [ ] Update `dev_docs/03-implementation/status.md`:
  - Mark AI Mode (L4) as ✅ Complete
  - Mark AI prompt generation as ✅ Complete
  - Mark confidence scoring as ✅ Complete

**ASK USER:** "Session 3 complete! All tests pass. JSON schema validated. Should I update status.md and mark deliverables complete?"

---

## Critical Checkpoints (Human Verification Required)

### Checkpoint 1: After Step 2 (Formatter Interface Wait)
**YOU ASK:** "Waiting for Session 2 to create internal/output/formatter.go. Has it been created? Can I proceed?"
**WAIT FOR:** User confirmation

### Checkpoint 2: After Step 3 (Model Extension)
**YOU ASK:** "Extended internal/models/risk_result.go with AI fields. No conflicts. Should I proceed?"
**WAIT FOR:** User confirmation

### Checkpoint 3: After Step 9 (JSON Schema)
**YOU ASK:** "JSON schema created. Should I validate AI mode output against it?"
**WAIT FOR:** User confirmation and review of validation results

### Checkpoint 4: Final (Before completion)
**YOU ASK:** "All deliverables complete. Schema validated. Performance: [X]ms overhead. Ready to mark session complete?"
**WAIT FOR:** User final approval

---

## Coordination with Other Sessions

### WAIT FOR (Session 2 creates this):
- **`internal/output/formatter.go`** - You implement the AIFormatter for VerbosityAIMode

### DO NOT MODIFY (other sessions own these):
- `cmd/crisk/hook.go` - Session 1
- `internal/git/*` - Session 1
- `internal/output/quiet.go` - Session 2
- `internal/output/standard.go` - Session 2
- `internal/output/explain.go` - Session 2

### YOU MODIFY (shared file):
- `internal/models/risk_result.go` - **EXTEND ONLY, don't break existing fields**

---

## Success Criteria

**Functional:**
- [ ] AI Mode outputs valid JSON matching schema v1.0
- [ ] AI actions array contains ready-to-execute prompts
- [ ] Confidence scores determine auto-fix safety
- [ ] Prompts are generated for 4+ fix types
- [ ] `--ai-mode` flag works in CLI

**Performance:**
- [ ] AI Mode generation overhead <200ms

**Quality:**
- [ ] JSON validates against schema
- [ ] Integration tests pass
- [ ] No conflicts with other sessions

---

## Error Handling

**If you encounter issues:**
1. **Formatter interface not found:** Wait for Session 2 or ask user
2. **Model conflicts:** Check you're only adding new fields, not modifying existing
3. **JSON validation fails:** Debug schema vs output mismatch
4. **Build errors:** Verify imports and dependencies

**Always ask before:**
- Modifying files not in your ownership list
- Changing existing model fields
- Adding new dependencies
- Making breaking changes

---

## Final Deliverables

When complete, you should have:
1. ✅ AI Mode formatter (implements formatter.go interface)
2. ✅ Prompt generator (4+ fix types)
3. ✅ Confidence calculator (with >0.85 threshold)
4. ✅ Prompt templates library
5. ✅ Extended RiskResult model with AI fields
6. ✅ JSON schema v1.0 definition
7. ✅ CLI integration (--ai-mode flag)
8. ✅ Integration test suite
9. ✅ Updated status.md documentation

---

## Questions to Ask During Implementation

- "I've read the AI Mode JSON schema. Should I proceed?"
- "Has Session 2 created internal/output/formatter.go? Can I import it?"
- "Extended RiskResult model. No conflicts. Should I proceed?"
- "AI Mode formatter implemented. Should I proceed with prompt generation?"
- "Confidence scoring implemented. Should I proceed with templates?"
- "CLI integration complete. Should I proceed with JSON schema?"
- "Integration test ready. Should I run it?"
- "JSON schema validated. Performance: [X]ms. Is this acceptable?"
- "Session 3 complete! Should I update status.md?"

**Remember:** You depend on Session 2's formatter interface. Wait for it before implementing AIFormatter!
