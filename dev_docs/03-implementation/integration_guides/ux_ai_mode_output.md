# UX Integration Guide: AI Mode Output

**Purpose:** Implement machine-readable JSON output for AI assistant integration (Claude Code, Cursor, Copilot)
**Last Updated:** October 4, 2025
**Reference:** [developer_experience.md](../../00-product/developer_experience.md) §2.4 - AI-Assistant Mode

> **Design Principle:** Dense, structured data over human-friendly formatting. Prioritize machine parsability and actionability. (12-factor: Factor 4 - Tools are structured outputs)

---

## Overview

AI Mode (`--ai-mode`) provides a machine-readable JSON schema optimized for AI coding assistants to:

1. **Parse results** without regex/string manipulation (5ms JSON.parse vs 100-500ms text parsing)
2. **Make intelligent decisions** based on confidence scores and metrics
3. **Auto-fix issues** using ready-to-execute AI prompts
4. **Learn patterns** from graph analysis and historical context

**Key difference from human modes:** AI Mode includes actionable prompts, confidence scores, auto-fix flags, and rich context that AI assistants can consume to improve code **before the user even sees it**.

---

## JSON Schema v1.0

### Schema URL
```
https://coderisk.com/schemas/ai-mode/v1.0.json
```

### Root Structure

```json
{
  "meta": { /* ... */ },
  "risk": { /* ... */ },
  "files": [ /* ... */ ],
  "graph_analysis": { /* ... */ },
  "investigation_trace": [ /* ... */ ],
  "recommendations": { /* ... */ },
  "ai_assistant_actions": [ /* ... */ ],
  "contextual_insights": { /* ... */ },
  "performance": { /* ... */ },
  "should_block_commit": true,
  "block_reason": "high_risk_critical_function_no_tests",
  "override_allowed": true,
  "override_requires_justification": true
}
```

---

## Schema Sections

### 1. Meta (Request Context)

```json
{
  "meta": {
    "version": "1.0",
    "timestamp": "2025-10-04T14:23:17Z",
    "duration_ms": 2134,
    "branch": "feature/auth-improvements",
    "base_branch": "main",
    "commit_sha": "abc1234def5678",
    "files_analyzed": 3,
    "agent_hops": 4,
    "cache_hit": false
  }
}
```

**Fields:**
- `version` (string): Schema version (semantic versioning)
- `timestamp` (ISO 8601): When analysis completed
- `duration_ms` (int): Total analysis time in milliseconds
- `branch` (string): Current branch name
- `base_branch` (string): Base branch for comparison (e.g., "main")
- `commit_sha` (string, optional): Git commit SHA if available
- `files_analyzed` (int): Number of files analyzed
- `agent_hops` (int): Number of graph traversal hops (0 = Phase 1 only)
- `cache_hit` (bool): Whether result was cached

### 2. Risk (Summary Assessment)

```json
{
  "risk": {
    "level": "MEDIUM",
    "score": 6.2,
    "confidence": 0.87,
    "trend": "increasing",
    "previous_score": 4.1
  }
}
```

**Fields:**
- `level` (enum): `"NONE"`, `"LOW"`, `"MEDIUM"`, `"HIGH"`, `"CRITICAL"`
- `score` (float): Numeric risk score 0-10 (quantified risk)
- `confidence` (float): Confidence in assessment 0.0-1.0
- `trend` (enum, optional): `"increasing"`, `"decreasing"`, `"stable"`
- `previous_score` (float, optional): Previous risk score for trend

### 3. Files (Per-File Details)

```json
{
  "files": [
    {
      "path": "src/auth.py",
      "language": "python",
      "lines_changed": 45,
      "risk_score": 7.8,
      "metrics": {
        "complexity": 8,
        "test_coverage": 0.0,
        "coupling_score": 6,
        "churn_rate": 0.23,
        "incident_count": 2,
        "last_incident_days": 18
      },
      "issues": [
        {
          "id": "TEST_COVERAGE_ZERO",
          "severity": "high",
          "category": "quality",
          "line_start": 45,
          "line_end": 67,
          "function": "authenticate_user",
          "message": "No test coverage for critical auth function",
          "impact_score": 8.5,
          "fix_priority": 1,
          "estimated_fix_time_min": 30,
          "auto_fixable": true,
          "fix_command": "crisk fix-with-ai --tests src/auth.py:45-67"
        }
      ],
      "dependencies": {
        "imports": ["user_service", "database", "jwt", "bcrypt"],
        "called_by": ["api.py", "middleware.py"],
        "calls": ["user_service.get_user", "database.query", "jwt.encode"]
      },
      "history": {
        "commits_90d": 12,
        "authors": ["alice@example.com", "bob@example.com"],
        "primary_author": "alice@example.com",
        "author_ownership": 0.75,
        "ownership_changed_days": 5,
        "hotspot": true,
        "hotspot_score": 0.82
      },
      "incidents": [
        {
          "id": "INC-453",
          "date": "2025-09-15",
          "severity": "high",
          "summary": "Auth timeout cascade failure",
          "related_files": ["auth.py", "user_service.py"],
          "similarity_score": 0.91
        }
      ]
    }
  ]
}
```

**Issue Object Fields:**
- `id` (string): Unique issue identifier (e.g., "TEST_COVERAGE_ZERO")
- `severity` (enum): `"low"`, `"medium"`, `"high"`, `"critical"`
- `category` (enum): `"quality"`, `"security"`, `"architecture"`, `"performance"`
- `line_start`, `line_end` (int): Source location
- `function` (string, optional): Function name if applicable
- `message` (string): Human-readable issue description
- `impact_score` (float): Impact score 0-10
- `fix_priority` (int): Priority ranking (1 = highest)
- `estimated_fix_time_min` (int): Estimated fix time in minutes
- `auto_fixable` (bool): Whether AI can fix automatically
- `fix_command` (string, optional): CLI command to trigger fix

### 4. Graph Analysis (Structural Insights)

```json
{
  "graph_analysis": {
    "blast_radius": {
      "direct_dependents": 12,
      "indirect_dependents": 47,
      "total_affected_files": 59,
      "critical_path_depth": 5
    },
    "temporal_coupling": [
      {
        "file_a": "auth.py",
        "file_b": "user_service.py",
        "strength": 0.85,
        "commits": 17,
        "total_commits": 20,
        "window_days": 90,
        "last_co_change": "2025-10-01"
      }
    ],
    "hotspots": [
      {
        "file": "auth.py",
        "score": 0.82,
        "reason": "high_churn_low_coverage",
        "churn": 0.23,
        "coverage": 0.0,
        "incidents": 2
      }
    ]
  }
}
```

**Purpose:** AI assistants use this to understand architectural impact and suggest refactoring patterns.

### 5. Investigation Trace (Agent Decision Path)

```json
{
  "investigation_trace": [
    {
      "hop": 1,
      "node_type": "file",
      "node_id": "auth.py",
      "action": "analyze_changed_file",
      "metrics_calculated": ["complexity", "coverage", "coupling"],
      "decision": "investigate_callers",
      "reasoning": "high_coupling_detected",
      "confidence": 0.91,
      "duration_ms": 456
    },
    {
      "hop": 2,
      "node_type": "file",
      "node_id": "user_service.py",
      "action": "analyze_caller",
      "relationship": "temporal_coupling",
      "strength": 0.85,
      "decision": "check_incidents",
      "reasoning": "strong_coupling_with_incident_history",
      "confidence": 0.88,
      "duration_ms": 678
    }
  ]
}
```

**Purpose:** AI assistants can trace LLM reasoning to understand why risk was flagged and use similar logic in their own analysis.

### 6. Recommendations (Actionable Guidance)

```json
{
  "recommendations": {
    "critical": [
      {
        "priority": 1,
        "action": "add_tests",
        "target": "src/auth.py:authenticate_user",
        "reason": "zero_coverage_critical_function",
        "estimated_time_min": 30,
        "auto_fixable": true,
        "command": "crisk fix-with-ai --tests src/auth.py:45-67",
        "ai_prompt_template": "Generate comprehensive unit and integration tests for authenticate_user() in src/auth.py. Cover: happy path, invalid tokens, expired tokens, rate limiting, error handling."
      }
    ],
    "high": [
      {
        "priority": 2,
        "action": "reduce_coupling",
        "target": "src/auth.py",
        "coupled_with": "user_service.py",
        "reason": "temporal_coupling_85_percent",
        "estimated_time_min": 120,
        "auto_fixable": false,
        "suggestions": [
          "Extract UserRepository interface",
          "Use dependency injection for user_service",
          "Add integration tests for auth + user_service interactions"
        ]
      }
    ],
    "medium": []
  }
}
```

**Purpose:** AI assistants prioritize fixes based on `priority` and `auto_fixable` flags.

### 7. AI Assistant Actions (Ready-to-Execute Prompts)

```json
{
  "ai_assistant_actions": [
    {
      "action_type": "generate_tests",
      "confidence": 0.92,
      "ready_to_execute": true,
      "prompt": "Generate unit tests for authenticate_user() function in src/auth.py (lines 45-67). Include tests for: valid credentials, invalid credentials, expired tokens, missing tokens, rate limit exceeded. Use pytest framework matching project conventions.",
      "expected_files": ["tests/test_auth.py"],
      "estimated_lines": 120
    },
    {
      "action_type": "add_error_handling",
      "confidence": 0.85,
      "ready_to_execute": true,
      "prompt": "Add try/except blocks with exponential backoff retry logic to network calls in src/auth.py lines 67-89. Handle: TimeoutError, ConnectionError, HTTPError. Use tenacity library if available, otherwise implement custom retry.",
      "expected_files": ["src/auth.py"],
      "estimated_lines": 15
    },
    {
      "action_type": "refactor_coupling",
      "confidence": 0.65,
      "ready_to_execute": false,
      "reason": "requires_architectural_decision",
      "prompt": "Suggest refactoring options to reduce coupling between auth.py and user_service.py. Consider: repository pattern, dependency injection, interface segregation. Provide 2-3 options with tradeoffs.",
      "expected_files": ["multiple"],
      "estimated_lines": "unknown"
    }
  ]
}
```

**Fields:**
- `action_type` (enum): `"generate_tests"`, `"add_error_handling"`, `"refactor_coupling"`, `"fix_security"`, `"reduce_complexity"`
- `confidence` (float): Confidence in auto-fix success 0.0-1.0
- `ready_to_execute` (bool): Whether AI should auto-fix (confidence > 0.85 threshold)
- `prompt` (string): Ready-to-use prompt for AI assistant
- `expected_files` (array): Files that will be created/modified
- `estimated_lines` (int | string): Estimated lines of code

**Usage by AI assistants:**
```typescript
// Cursor/Claude Code integration
const actions = analysis.ai_assistant_actions.filter(a => a.ready_to_execute);

for (const action of actions) {
  if (action.confidence > 0.9) {
    // High confidence: auto-fix silently
    await aiAssistant.execute(action.prompt);
  } else if (action.confidence > 0.85) {
    // Medium confidence: show user before fixing
    const approved = await showConfirmation(action);
    if (approved) await aiAssistant.execute(action.prompt);
  }
}
```

### 8. Contextual Insights (Learning from History)

```json
{
  "contextual_insights": {
    "similar_past_changes": [
      {
        "commit_sha": "def5678",
        "date": "2025-09-10",
        "author": "bob@example.com",
        "files_changed": ["auth.py", "user_service.py"],
        "outcome": "incident_INC-453",
        "lesson": "auth_changes_require_integration_tests"
      }
    ],
    "team_patterns": {
      "avg_test_coverage": 0.75,
      "your_coverage": 0.0,
      "percentile": 5,
      "team_avg_coupling": 4.2,
      "your_coupling": 6.0,
      "recommendation": "below_team_standards"
    },
    "file_reputation": {
      "auth.py": {
        "incident_density": 0.167,
        "team_avg": 0.05,
        "classification": "high_risk_file",
        "extra_review_recommended": true
      }
    }
  }
}
```

**Purpose:** AI assistants learn from team patterns and historical outcomes to generate better code over time.

### 9. Performance (Execution Metrics)

```json
{
  "performance": {
    "total_duration_ms": 2134,
    "breakdown": {
      "git_analysis": 234,
      "tree_sitter_parsing": 456,
      "graph_queries": 678,
      "llm_reasoning": 512,
      "metric_calculation": 254
    },
    "cache_efficiency": {
      "queries": 47,
      "cache_hits": 28,
      "cache_hit_rate": 0.596
    }
  }
}
```

**Purpose:** AI assistants can optimize based on timing (e.g., skip slow operations if under time pressure).

### 10. Commit Control (Block/Allow Decision)

```json
{
  "should_block_commit": true,
  "block_reason": "high_risk_critical_function_no_tests",
  "override_allowed": true,
  "override_requires_justification": true
}
```

**Fields:**
- `should_block_commit` (bool): Whether pre-commit hook should block
- `block_reason` (enum): Reason code for blocking (e.g., `"high_risk_critical_function_no_tests"`, `"security_vulnerability_detected"`)
- `override_allowed` (bool): Whether `--no-verify` is allowed
- `override_requires_justification` (bool): Whether override needs logged reason (enterprise mode)

---

## Implementation

### 1. AI Mode Formatter

**File:** `internal/output/ai_mode.go`

```go
package output

import (
    "encoding/json"
    "io"
    "time"

    "github.com/coderisk/coderisk-go/internal/models"
)

// AIFormatter outputs machine-readable JSON for AI assistants
type AIFormatter struct {
    Version string
}

func NewAIFormatter() *AIFormatter {
    return &AIFormatter{Version: "1.0"}
}

func (f *AIFormatter) Format(result *models.RiskResult, w io.Writer) error {
    output := map[string]interface{}{
        "meta": f.buildMeta(result),
        "risk": f.buildRisk(result),
        "files": f.buildFiles(result),
        "graph_analysis": f.buildGraphAnalysis(result),
        "investigation_trace": f.buildTrace(result),
        "recommendations": f.buildRecommendations(result),
        "ai_assistant_actions": f.buildAIActions(result),
        "contextual_insights": f.buildInsights(result),
        "performance": f.buildPerformance(result),
        "should_block_commit": result.ShouldBlock,
        "block_reason": result.BlockReason,
        "override_allowed": result.OverrideAllowed,
        "override_requires_justification": result.OverrideRequiresJustification,
    }

    encoder := json.NewEncoder(w)
    encoder.SetIndent("", "  ")
    return encoder.Encode(output)
}

func (f *AIFormatter) buildAIActions(result *models.RiskResult) []map[string]interface{} {
    actions := []map[string]interface{}{}

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

            if !action["ready_to_execute"].(bool) {
                action["reason"] = "confidence_below_threshold"
            }

            actions = append(actions, action)
        }
    }

    return actions
}
```

### 2. Model Extensions

**File:** `internal/models/risk_result.go` (extend existing)

```go
type Issue struct {
    // ... existing fields ...

    // AI Mode specific fields
    AutoFixable       bool     `json:"auto_fixable"`
    FixType           string   `json:"fix_type"` // "generate_tests", "add_error_handling", etc.
    FixConfidence     float64  `json:"fix_confidence"`
    AIPromptTemplate  string   `json:"ai_prompt_template"`
    ExpectedFiles     []string `json:"expected_files"`
    EstimatedLines    int      `json:"estimated_lines"`
}

type RiskResult struct {
    // ... existing fields ...

    // Commit control
    ShouldBlock                   bool   `json:"should_block_commit"`
    BlockReason                   string `json:"block_reason"`
    OverrideAllowed               bool   `json:"override_allowed"`
    OverrideRequiresJustification bool   `json:"override_requires_justification"`

    // Performance
    Performance Performance `json:"performance"`
}

type Performance struct {
    TotalDurationMS int                    `json:"total_duration_ms"`
    Breakdown       map[string]int         `json:"breakdown"`
    CacheEfficiency map[string]interface{} `json:"cache_efficiency"`
}
```

---

## AI Assistant Integration Examples

### Example 1: Claude Code Auto-Fix

```typescript
// Claude Code processes AI mode output
const result = execSync('crisk check --ai-mode').toString();
const analysis = JSON.parse(result);

// Auto-fix high-confidence issues silently
const autoFixable = analysis.ai_assistant_actions.filter(
  a => a.ready_to_execute && a.confidence > 0.9
);

if (autoFixable.length > 0) {
  console.log(`Auto-fixing ${autoFixable.length} issues...`);

  for (const action of autoFixable) {
    // Execute AI prompt to fix issue
    await claudeAPI.execute(action.prompt);
  }

  // Re-check after fixes
  const recheckResult = execSync('crisk check --ai-mode').toString();
  const recheckAnalysis = JSON.parse(recheckResult);

  if (recheckAnalysis.risk.level === 'LOW') {
    console.log('✅ All issues resolved automatically');
  } else {
    console.log('⚠️  Some issues require manual review');
  }
}
```

### Example 2: Cursor Inline Diagnostics

```typescript
// Cursor shows inline warnings with AI fix actions
const analysis = JSON.parse(execSync('crisk check --ai-mode').toString());

analysis.files.forEach(file => {
  file.issues.forEach(issue => {
    // Create VS Code diagnostic
    vscode.languages.createDiagnostic(
      file.path,
      new vscode.Range(issue.line_start, 0, issue.line_end, 0),
      issue.message,
      issue.severity === 'high' ? vscode.DiagnosticSeverity.Error : vscode.DiagnosticSeverity.Warning
    );
  });
});

// Offer AI fix via code action
registerCodeActionProvider({
  provideCodeActions: (document, range) => {
    const actions = analysis.ai_assistant_actions
      .filter(a => a.ready_to_execute)
      .map(action => ({
        title: `CodeRisk: ${action.action_type}`,
        command: 'cursor.executeAIPrompt',
        arguments: [action.prompt]
      }));

    return actions;
  }
});
```

---

## Schema Versioning

### Version Header

```json
{
  "meta": {
    "version": "1.0",
    "schema_url": "https://coderisk.com/schemas/ai-mode/v1.0.json"
  }
}
```

### Backward Compatibility

- **Minor version changes (1.0 → 1.1):** Add optional fields only
- **Major version changes (1.0 → 2.0):** Breaking schema changes
- AI assistants should validate against `schema_url` and gracefully handle unknown fields

### Future Schema Extensions (v1.1+)

Potential additions:
- `"suggested_reviewers"` - Team members with expertise in changed files
- `"ci_integration"` - CI/CD pipeline recommendations
- `"cost_estimate"` - LLM cost for investigation (if Phase 2 used)
- `"security_score"` - Separate security risk score

---

## Performance Characteristics

| Metric | Value | Notes |
|--------|-------|-------|
| Output size | 5-10KB | Depends on file count and graph depth |
| Generation time | +200ms vs standard mode | More computation for AI fields |
| Parsing time (AI) | <5ms | JSON.parse is fast |
| Net benefit | Faster for AI | Despite larger payload, structured parsing saves 100-500ms |

**Trade-off:** 200ms extra generation time, but AI assistants save 500ms+ in parsing → **300ms net benefit**

---

## Testing Strategy

### Unit Tests

```go
func TestAIModeFormatter(t *testing.T) {
    result := &models.RiskResult{
        RiskLevel: "MEDIUM",
        Issues: []models.Issue{
            {
                AutoFixable:      true,
                FixType:          "generate_tests",
                FixConfidence:    0.92,
                AIPromptTemplate: "Generate tests for...",
            },
        },
    }

    var buf bytes.Buffer
    formatter := NewAIFormatter()
    err := formatter.Format(result, &buf)

    assert.NoError(t, err)

    var output map[string]interface{}
    json.Unmarshal(buf.Bytes(), &output)

    // Validate schema
    assert.Equal(t, "1.0", output["meta"].(map[string]interface{})["version"])
    assert.True(t, output["should_block_commit"].(bool))

    // Validate AI actions
    actions := output["ai_assistant_actions"].([]interface{})
    assert.Len(t, actions, 1)
    assert.Equal(t, "generate_tests", actions[0].(map[string]interface{})["action_type"])
    assert.True(t, actions[0].(map[string]interface{})["ready_to_execute"].(bool))
}
```

### JSON Schema Validation

```bash
# Validate output against JSON schema
crisk check --ai-mode | jq . > output.json
ajv validate -s schemas/ai-mode-v1.0.json -d output.json
```

---

## Next Steps

1. Implement `internal/output/ai_mode.go` with JSON schema generation
2. Extend `internal/models/risk_result.go` with AI-specific fields
3. Generate AI prompt templates for common fixes (tests, error handling, etc.)
4. Create JSON schema file at `schemas/ai-mode-v1.0.json`
5. Add integration tests for AI assistant workflows
6. Document AI Mode in user-facing docs

---

**See also:**
- [developer_experience.md](../../00-product/developer_experience.md) - AI Mode design philosophy
- [ux_adaptive_verbosity.md](ux_adaptive_verbosity.md) - All verbosity levels
- [ux_pre_commit_hook.md](ux_pre_commit_hook.md) - Pre-commit integration
- [DEVELOPMENT_WORKFLOW.md](../../DEVELOPMENT_WORKFLOW.md) - Implementation guardrails
