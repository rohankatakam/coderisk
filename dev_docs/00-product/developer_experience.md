# Developer Experience (DX): Seamless Risk Assessment for Vibe Coding

**Last Updated:** October 3, 2025
**Purpose:** Define the optimal user experience for automatic risk assessment across team sizes, coding styles, and git workflows

> **Cross-reference:** See [developer_workflows.md](developer_workflows.md) for workflow patterns and [user_personas.md](user_personas.md) for user needs

---

## Design Philosophy

**Core Principle:** CodeRisk should be **invisible when safe, visible when risky**—like autocorrect for code safety.

**Key Tenets:**
1. **Zero-friction activation** - No configuration, works on first `crisk check`
2. **Instant feedback** - <2s for cached results, <5s for cold start
3. **Actionable guidance** - Tell developers *what to fix*, not just *what's wrong*
4. **Adaptive verbosity** - Quiet by default, detailed on request
5. **Vibe coding native** - Designed for AI-generated code velocity

---

## The Vibe Coding Challenge

### Problem: AI Code Generates Faster Than Humans Can Review

**Traditional Manual Coding:**
```
Developer writes code: 50-100 lines/hour
Developer reviews own code: Built-in (continuous)
Risk: Low (human writes what they understand)
```

**Vibe Coding (Claude Code/Cursor):**
```
AI generates code: 500-1000 lines/hour (10x faster)
Developer reviews AI code: Often cursory (trust AI)
Risk: HIGH (developer may not understand all implications)
```

**The Gap:** Developers commit AI-generated code with less scrutiny than hand-written code.

**CodeRisk's Role:** Automated pre-commit reviewer that matches AI velocity.

---

## Seamless Integration Points

### 1. Pre-Commit Automatic Assessment (Primary UX)

**Goal:** Risk check happens automatically before every commit, no developer action required.

#### Option A: Git Pre-Commit Hook (Recommended)

**Installation:**
```bash
# One-time setup after crisk init
crisk hook install

# This creates .git/hooks/pre-commit:
#!/bin/bash
crisk check --pre-commit --quiet
exit $?
```

**User Experience:**
```bash
# Developer commits (manual or AI-generated code)
git add src/auth.py src/middleware/rate_limit.py
git commit -m "Add rate limiting to auth endpoint"

# Hook triggers automatically (developer sees):
🔍 CodeRisk: Analyzing 2 changed files... (1.2s)

✅ LOW risk - Safe to commit
   - Test coverage: 78%
   - Coupling score: 3/10
   - No incidents in changed files

[main abc1234] Add rate limiting to auth endpoint
 2 files changed, 145 insertions(+), 12 deletions(-)
```

**If Risk Detected:**
```bash
git commit -m "Add payment processing"

# Hook triggers:
🔍 CodeRisk: Analyzing 5 changed files... (2.1s)

⚠️  MEDIUM risk detected:

   Issues:
   1. payment_handler.py has 0% test coverage
   2. stripe_client.py calls 8 other functions (high coupling)
   3. payment_handler.py changed with database.py in 90% of commits

   Recommendations:
   - Add tests for payment_handler.py
   - Review coupling with database.py

   Run 'crisk check --explain' for investigation details

❌ Commit blocked. Fix issues or override with 'git commit --no-verify'
```

**Developer Decision Tree:**
```
Risk detected → Developer has 3 choices:

1. Fix issues (recommended)
   → Add tests, reduce coupling
   → Re-commit (auto-checks again)

2. Override (low friction)
   → git commit --no-verify -m "..."
   → Logs override for team visibility

3. Get details
   → crisk check --explain
   → See full investigation trace
   → Make informed decision
```

#### Option B: IDE Extension (VS Code, Future)

**Real-Time Feedback:**
```
Developer modifies file in VS Code
→ CodeRisk extension detects change
→ Runs lightweight check in background
→ Shows risk indicator in status bar

🟢 Low Risk  |  🟡 Medium Risk  |  🔴 High Risk

Click indicator → See detailed issues
```

**Vibe Coding Integration:**
```
Developer uses Claude Code to generate feature
→ Claude writes 5 files in 30 seconds
→ CodeRisk extension auto-scans on save
→ Shows issues inline before commit

Example:
src/auth.py:45 - ⚠️ No test coverage for this function
src/auth.py:67 - 🔴 High coupling with user_service.py (85% co-change)
```

---

### 2. Adaptive Verbosity (Context-Aware Output)

**Design Principle:** Show enough information to act, nothing more.

#### Level 1: Quiet Mode (Pre-Commit Hook Default)
```bash
crisk check --quiet

# Output (success):
✅ LOW risk

# Output (issues):
⚠️ MEDIUM risk: 3 issues detected
Run 'crisk check' for details
```

#### Level 2: Standard Mode (CLI Default)
```bash
crisk check

# Output:
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

#### Level 3: Explain Mode (Full Investigation)
```bash
crisk check --explain

# Output:
🔍 CodeRisk Investigation Report
Started: 2025-10-03 14:23:15
Completed: 2025-10-03 14:23:17 (2.1s)
Agent hops: 4

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Hop 1: auth.py (Starting point)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Changed functions:
  - authenticate_user() (lines 45-67)
  - validate_token() (lines 89-102)

Metrics calculated:
  ✅ Complexity: 6 (target: <10)
  ❌ Test coverage: 0% (target: >70%)
  ⚠️  Import count: 8 (high coupling)

Agent decision: Investigate callers (high coupling signal)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Hop 2: user_service.py (Caller investigation)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Relationship:
  - Calls authenticate_user() 12 times
  - Co-changed with auth.py in 17 of 20 commits (85%)

Temporal coupling analysis:
  - Strong coupling (>75% threshold)
  - Recent co-changes: 5 in last 30 days

Agent decision: Check incident history

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Hop 3: Incident database (Historical risk)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Incidents affecting auth.py:
  - INC-453 (2025-09-15): Auth timeout caused user_service failure
  - INC-401 (2025-08-22): Token validation bug

Pattern detected: auth.py + user_service.py coupling issues

Agent decision: Check test coverage

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Hop 4: Test suite analysis
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Test files found: None for auth.py
Test coverage: 0%

Historical incident correlation:
  - Files with <30% coverage: 3x more incidents
  - auth.py has incident history + 0% coverage = HIGH risk

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Final Assessment
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Risk Level: MEDIUM → HIGH (elevated due to incident history)

Evidence:
  1. Zero test coverage on changed functions
  2. Strong temporal coupling with user_service.py (85%)
  3. 2 incidents in last 90 days involving these files
  4. High coupling (8 dependencies)

Recommendations (priority order):
  1. Add integration tests for authenticate_user() + user_service interactions
  2. Add unit tests for validate_token()
  3. Review coupling: Can user_service use an interface instead of direct calls?
  4. Consider circuit breaker pattern (given timeout history)

Suggested next steps:
  → crisk suggest-tests auth.py
  → crisk show-coupling auth.py user_service.py
```

#### Level 4: AI-Assistant Mode (Machine-Readable)

**Goal:** Optimized output for AI assistants (Claude Code, Cursor, Copilot) to process and act on.

**Design Principle:** Dense, structured data over human-friendly formatting. Prioritize machine parsability and actionability.

**Usage:**
```bash
crisk check --ai-mode
# or
crisk check --format ai
```

**Output Format (Structured JSON-like):**
```json
{
  "meta": {
    "version": "1.0",
    "timestamp": "2025-10-03T14:23:17Z",
    "duration_ms": 2134,
    "branch": "feature/auth-improvements",
    "base_branch": "main",
    "commit_sha": "abc1234",
    "files_analyzed": 3,
    "agent_hops": 4,
    "cache_hit": false
  },
  "risk": {
    "level": "MEDIUM",
    "score": 6.2,
    "confidence": 0.87,
    "trend": "increasing",
    "previous_score": 4.1
  },
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
        },
        {
          "id": "HIGH_COUPLING",
          "severity": "medium",
          "category": "architecture",
          "coupled_with": ["user_service.py", "database.py"],
          "coupling_strength": 0.85,
          "co_change_frequency": 0.85,
          "message": "Strong temporal coupling detected",
          "impact_score": 6.2,
          "fix_priority": 2,
          "estimated_fix_time_min": 120,
          "auto_fixable": false,
          "suggestions": [
            "Extract interface for user_service interactions",
            "Use dependency injection pattern",
            "Add integration tests for coupled components"
          ]
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
  ],
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
  },
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
    },
    {
      "hop": 3,
      "node_type": "incident",
      "node_id": "INC-453",
      "action": "similarity_search",
      "similarity_score": 0.91,
      "decision": "check_tests",
      "reasoning": "similar_incident_found",
      "confidence": 0.85,
      "duration_ms": 512
    },
    {
      "hop": 4,
      "node_type": "tests",
      "node_id": "test_suite",
      "action": "coverage_analysis",
      "coverage_found": 0.0,
      "decision": "finalize_high_risk",
      "reasoning": "zero_coverage_plus_incident_history",
      "confidence": 0.87,
      "duration_ms": 488
    }
  ],
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
    "medium": [
      {
        "priority": 3,
        "action": "add_error_handling",
        "target": "src/auth.py:67-89",
        "reason": "network_calls_without_retry",
        "estimated_time_min": 20,
        "auto_fixable": true,
        "command": "crisk fix-with-ai --error-handling src/auth.py:67-89"
      }
    ]
  },
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
  ],
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
  },
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
  },
  "should_block_commit": true,
  "block_reason": "high_risk_critical_function_no_tests",
  "override_allowed": true,
  "override_requires_justification": true
}
```

**Key Differences from Human Modes:**

1. **Dense Metrics:** Every metric quantified (no subjective "high/low")
2. **Machine Actions:** Explicit `ai_assistant_actions` array with ready-to-execute prompts
3. **Confidence Scores:** All decisions include confidence levels (0.0-1.0)
4. **Structured Data:** Nested JSON, no emojis, no formatting
5. **Auto-Fixable Flags:** Boolean indicating if AI can fix automatically
6. **Estimated Times:** Time estimates for AI to plan work
7. **Prompt Templates:** Ready-to-use prompts for AI assistants
8. **Graph Traversal:** Full investigation trace with reasoning
9. **Contextual Learning:** Team patterns, file reputation, historical outcomes

**AI Assistant Integration Example (Claude Code):**

```python
# Claude Code internally processes crisk output
result = subprocess.run(["crisk", "check", "--ai-mode"], capture_output=True)
data = json.loads(result.stdout)

# Decision tree
if data["should_block_commit"]:
    if data["ai_assistant_actions"]:
        # Offer to auto-fix
        fixable_actions = [a for a in data["ai_assistant_actions"] if a["ready_to_execute"]]

        if fixable_actions:
            print("CodeRisk detected issues. I can fix these automatically:")
            for action in fixable_actions:
                print(f"  - {action['action_type']} (confidence: {action['confidence']})")

            if user_confirms():
                for action in fixable_actions:
                    # Execute AI fix using provided prompt
                    execute_ai_prompt(action["prompt"])

        # Re-check after fixes
        result = subprocess.run(["crisk", "check", "--ai-mode"], capture_output=True)
        data = json.loads(result.stdout)

        if not data["should_block_commit"]:
            print("✅ All issues resolved! Safe to commit.")
```

**Cursor Integration Example:**

```typescript
// Cursor processes AI mode output
const criskResult = execSync('crisk check --ai-mode').toString();
const analysis = JSON.parse(criskResult);

// Show inline diagnostics
analysis.files.forEach(file => {
  file.issues.forEach(issue => {
    vscode.languages.createDiagnostic(
      file.path,
      new vscode.Range(issue.line_start, 0, issue.line_end, 0),
      issue.message,
      issue.severity === 'high' ? vscode.DiagnosticSeverity.Error : vscode.DiagnosticSeverity.Warning
    );
  });
});

// Offer AI fixes via code actions
if (analysis.ai_assistant_actions.length > 0) {
  registerCodeActionProvider({
    provideCodeActions: () => {
      return analysis.ai_assistant_actions
        .filter(a => a.ready_to_execute)
        .map(action => ({
          title: `Fix: ${action.action_type}`,
          command: 'cursor.executeAIPrompt',
          arguments: [action.prompt]
        }));
    }
  });
}
```

**Benefits of AI Mode:**

1. **No Parsing Required:** Structured JSON, not human text
2. **Actionable Prompts:** Ready-to-execute AI prompts included
3. **Confidence Scoring:** AI knows when it should/shouldn't act
4. **Rich Context:** Full graph analysis, history, team patterns
5. **Performance Data:** AI can optimize based on timing
6. **Auto-Fix Flags:** Clear signal when automation is safe
7. **Progressive Enhancement:** AI can choose to use detailed data or ignore it

**Performance Characteristics:**

- **Output Size:** ~5-10KB JSON (vs ~2KB human text)
- **Generation Time:** +200ms vs standard mode (more computation)
- **Parsing Time:** <5ms (JSON.parse) vs 100-500ms (regex/parsing human text)
- **Net Benefit:** Faster for AI to consume despite larger payload

**Versioning:**

```json
{
  "meta": {
    "version": "1.0",
    "schema_url": "https://coderisk.com/schemas/ai-mode/v1.0.json"
  }
}
```

- Schema versioned separately from CodeRisk
- AI assistants can validate against schema
- Backward compatibility guarantee within major version

---

### 3. Team Size Adaptive UX

Different teams need different levels of automation and friction.

#### Solo Developer / Side Project (1 person)

**UX Goal:** Personal assistant, minimal friction

**Configuration:**
```bash
crisk config set mode solo
```

**Behavior:**
- Pre-commit hook: WARNING only (never blocks)
- Verbosity: Standard
- Override: Easy (single flag)
- Focus: Education + safety

**Example:**
```bash
git commit -m "Add feature X"

⚠️  MEDIUM risk:
   - Missing tests (consider adding)
   - High coupling detected

✅ Committed anyway (solo mode)
💡 Tip: Run 'crisk suggest-tests' for test ideas
```

---

#### Startup / Small Team (2-10 people)

**UX Goal:** Fast but safe, balance velocity with quality

**Configuration:**
```bash
crisk config set mode team
crisk config set block-on high  # Block only on HIGH/CRITICAL
```

**Behavior:**
- Pre-commit hook: Blocks on HIGH/CRITICAL
- Verbosity: Standard with suggestions
- Override: Allowed but logged
- Focus: Prevent major issues, allow minor risks

**Example:**
```bash
git commit -m "Add Stripe integration"

🔴 HIGH risk detected:
   - payment.py handles money but has 0% tests
   - No error handling for network failures

❌ Commit blocked
💡 AI can help: Run 'crisk suggest-fixes --ai'

# Override if urgent:
git commit --no-verify -m "..."
(Override logged to team dashboard)
```

---

#### Growth Company (10-50 people)

**UX Goal:** Enforce standards, reduce review burden

**Configuration:**
```bash
crisk config set mode standard
crisk config set block-on medium  # Block on MEDIUM+
crisk config set ci-required true
```

**Behavior:**
- Pre-commit hook: Blocks on MEDIUM+
- CI check: Required (cannot merge without pass)
- Override: Requires justification message
- Focus: Consistent quality bar

**Example:**
```bash
git commit -m "Refactor auth module"

⚠️  MEDIUM risk:
   - Test coverage dropped from 78% → 65%
   - 3 new dependencies added

Provide override reason (or Ctrl+C to cancel):
> Refactor needed for new OAuth flow, tests updated in next commit

✅ Committed with override
📝 Reason logged for PR review
```

---

#### Enterprise (50-500+ people)

**UX Goal:** Compliance, audit trail, strict gates

**Configuration:**
```bash
crisk config set mode enterprise
crisk config set block-on low       # Block on everything
crisk config set require-approval true
crisk config set audit-log true
```

**Behavior:**
- Pre-commit hook: Blocks on LOW+ (very strict)
- Override: Requires manager approval (2FA)
- CI check: Multiple stages (security, compliance, risk)
- Focus: Compliance, audit, zero tolerance

**Example:**
```bash
git commit -m "Update PII handling"

⚠️  MEDIUM risk:
   - Compliance policy: PII changes require security review
   - Test coverage: 75% (policy requires 80%)

❌ Commit blocked

Options:
  1. Fix issues and re-commit
  2. Request manager override: crisk override request --ticket SEC-456

Audit log: /var/log/coderisk/audit.log
```

---

## Vibe Coding Specific UX Patterns

### Pattern 0: Native AI Assistant Integration (Using AI Mode)

**Problem:** AI coding assistants (Claude Code, Cursor) need structured data to make intelligent decisions about code safety.

**Solution:** AI Mode provides machine-readable output that AI assistants can consume and act on automatically.

**Integration Flow:**

```bash
# User working in Claude Code prompts:
> "Add payment processing with Stripe"

# Claude Code generates code (internal process):
1. Generate initial code based on prompt
2. Before presenting to user, run: crisk check --ai-mode
3. Parse JSON output
4. If issues detected with auto_fixable=true:
   - Automatically fix issues using provided ai_prompt_template
   - Re-run crisk check --ai-mode
   - Present final, safer code to user
5. Present to user with risk annotation

# What user sees:
> I've added Stripe payment processing. CodeRisk analysis:
> ✅ Risk level: LOW (automatically fixed 3 issues)
>
> Auto-fixed issues:
> - Added input validation for payment amounts
> - Added error handling for network failures
> - Generated test suite (78% coverage)
>
> Files created:
> - src/payment_handler.py
> - src/stripe_client.py
> - tests/test_payment.py
```

**Cursor Integration Example:**

```typescript
// Cursor monitors file changes
onFileChange(async (file) => {
  if (file.wasGeneratedByAI) {
    // Run CodeRisk in AI mode
    const result = await exec('crisk check --ai-mode --files ' + file.path);
    const analysis = JSON.parse(result);

    // Auto-fix high-confidence issues
    const autoFixable = analysis.ai_assistant_actions.filter(
      a => a.ready_to_execute && a.confidence > 0.85
    );

    if (autoFixable.length > 0) {
      showNotification(`Fixing ${autoFixable.length} issues in AI-generated code...`);

      for (const action of autoFixable) {
        // Execute fix using Cursor's AI
        await cursorAI.execute(action.prompt);
      }

      // Re-analyze
      const recheck = await exec('crisk check --ai-mode --files ' + file.path);
      const recheckAnalysis = JSON.parse(recheck);

      if (recheckAnalysis.risk.level === 'LOW') {
        showNotification('✅ All issues resolved automatically');
      }
    }
  }
});
```

**Key Benefits:**

1. **Silent Quality Improvement:** AI assistants fix issues before user even sees them
2. **Learning Loop:** AI assistants learn what CodeRisk flags, generate better code over time
3. **Confidence-Based Action:** Only auto-fix when confidence > threshold
4. **Transparent to User:** User sees final, safe code with annotation of what was fixed

**AI Mode Data Used:**

- `ai_assistant_actions[]` - Ready-to-execute prompts
- `auto_fixable` flags - Which issues can be auto-fixed
- `confidence` scores - Whether to act automatically
- `estimated_fix_time_min` - For progress indicators
- `investigation_trace[]` - Understanding why issue exists

**Progressive Enhancement:**

```typescript
// Basic integration (just show issues)
if (analysis.should_block_commit) {
  showWarning(analysis.risk.level);
}

// Advanced integration (auto-fix high confidence)
if (analysis.ai_assistant_actions.filter(a => a.confidence > 0.9).length > 0) {
  autoFix();
}

// Expert integration (use full graph analysis)
const couplingIssues = analysis.graph_analysis.temporal_coupling
  .filter(c => c.strength > 0.8);
// Suggest architectural improvements based on coupling
```

---

### Pattern 1: AI Generation Feedback Loop (Manual Developer Workflow)

**Problem:** AI generates code fast, developer commits without thorough review.

**Solution:** Inject CodeRisk into AI workflow with explicit developer control.

**UX Flow:**
```bash
# Developer prompts Claude Code
> "Build a real-time notification system with WebSockets"

# Claude generates 8 files, 650 lines in 45 seconds

# Developer reviews briefly, looks good
git add .

# About to commit
git commit -m "Add real-time notifications"

# Pre-commit hook triggers
🔍 Analyzing AI-generated code... (2.3s)

🔴 CRITICAL risk in AI-generated code:

   Security Issues:
   1. websocket_handler.py - No input validation (XSS risk)
   2. notification_service.py - No rate limiting (DoS risk)
   3. redis_client.py - Hardcoded credentials (secrets exposure)

   Quality Issues:
   4. 0% test coverage across 8 files
   5. Very high complexity (15-20 per function)

❌ Commit blocked

💡 AI can fix these! Run:
   crisk fix-with-ai --issues security,tests

# Developer runs AI fix
crisk fix-with-ai --issues security,tests

# CodeRisk prompts Claude to fix:
> "Fix the following security issues in websocket_handler.py:
> 1. Add input validation to prevent XSS
> 2. Add rate limiting to prevent DoS
> Also add comprehensive tests."

# Claude fixes issues

# Developer re-commits
git commit -m "Add real-time notifications (security hardened)"

🔍 Analyzing... (1.8s)

⚠️  MEDIUM risk (much better!)
   - Security issues resolved ✅
   - Test coverage: 65% (acceptable for v1)
   - Complexity still high (can improve later)

✅ Committed
💡 Follow-up: Consider refactoring for lower complexity
```

**Key UX Elements:**
- Catches AI security mistakes
- Suggests AI-powered fixes (close the loop)
- Allows incremental improvement (perfect is enemy of good)

---

### Pattern 2: Rapid Iteration Mode (Prototype → Production)

**Problem:** Vibe coders prototype fast but need to harden before merging.

**Solution:** "Prototype mode" vs "Production mode"

**UX Flow:**

```bash
# Phase 1: Prototype (velocity focus)
git checkout -b spike/new-feature
crisk config set mode prototype

# AI generates prototype
git commit -m "WIP: prototype"

🔍 Analyzing...
⚠️  HIGH risk detected (expected in prototype mode)
✅ Committed (prototype mode allows high risk)
💡 Remember to harden before merging to main

# Phase 2: Production hardening
git checkout -b feature/new-feature-prod
git merge spike/new-feature

crisk config set mode production
crisk harden

# CodeRisk analyzes and suggests hardening:
📋 Production Hardening Checklist:

Security:
  ❌ Add input validation (3 locations)
  ❌ Add authentication checks (5 endpoints)
  ❌ Remove console.log() statements (8 locations)

Quality:
  ❌ Add tests (0% → target 70%)
  ❌ Add error handling (12 missing try/catch)
  ❌ Add logging (audit trail)

Performance:
  ⚠️  Add caching (N+1 query detected)
  ⚠️  Add rate limiting (public endpoints)

Run 'crisk fix-with-ai --checklist' to fix automatically

# Developer fixes issues incrementally
crisk fix-with-ai --checklist

# Commits after each fix
git commit -m "Add input validation"  # ✅
git commit -m "Add test suite"        # ✅
git commit -m "Add error handling"    # ✅

# Final check
crisk check --mode production

✅ LOW risk - Production ready!
🎉 Safe to create PR
```

---

### Pattern 3: Confidence Scoring (Learn Over Time)

**Problem:** Not all AI suggestions are equal quality.

**Solution:** Track which AI prompts lead to risky code.

**UX Flow:**

```bash
# Developer's 5th time using Claude to add auth feature
> "Add JWT authentication with Redis session store"

# Claude generates code
# CodeRisk checks and sees pattern

🔍 Analyzing AI-generated authentication code...

ℹ️  Pattern detected: You've added auth features 5 times recently
   Historical quality:
   - 3 times required rework (HIGH risk detected)
   - 2 times passed first time (LOW risk)

   Suggestion: This is a complex area. Extra review recommended.

⚠️  MEDIUM risk:
   - Similar to previous attempt that had security issue (INC-453)
   - Missing rate limiting (you've been burned by this before)

Recommendations based on your history:
  1. Add rate limiting (you forgot this last time)
  2. Add integration tests (your weak spot for auth)
  3. Review token expiration logic (caused INC-453)

💡 Learn mode: Track your progress over time
   Next time, AI might suggest these automatically
```

**Personalized Learning:**
- Tracks individual developer patterns
- Learns which types of code they struggle with
- Provides personalized suggestions
- Celebrates improvement ("Your auth code quality has improved 50%!")

---

## Error Messages & Guidance

### Principle: Tell Developers What To DO, Not Just What's Wrong

**Bad Error Message:**
```
❌ High coupling detected
```

**Good Error Message:**
```
⚠️  High coupling detected: auth.py calls 8 other functions

Why this matters:
  - Changes to auth.py likely affect 8+ files
  - Increases chance of breaking unrelated features
  - Makes code harder to test in isolation

What to do:
  1. Review if all 8 dependencies are necessary
  2. Consider dependency injection for easier testing
  3. Run 'crisk show-coupling auth.py' to see dependency graph

Similar past issue: INC-453 (auth timeout cascade failure)
```

---

### Actionable Commands (Always Provide Next Step)

Every warning includes a suggested command:

```bash
⚠️  No test coverage
→ crisk suggest-tests auth.py

⚠️  High coupling
→ crisk show-coupling auth.py

⚠️  Incident history
→ crisk show-incidents auth.py

🔴 Security risk
→ crisk fix-with-ai --security

⚠️  Complex function
→ crisk suggest-refactor auth.py:45
```

---

## Performance & Timing UX

### The 2-Second Rule

**Principle:** Risk check must complete <2s for cached, <5s for cold start.

**Why:**
- <2s feels instant (doesn't break flow)
- 2-5s acceptable (developer expects some analysis)
- >5s frustrating (developer considers `--no-verify`)

**UX for Slow Checks:**

```bash
git commit -m "Large refactor"

🔍 CodeRisk: Analyzing 45 changed files...

[Progress bar]
▓▓▓▓▓▓▓▓▓▓▓░░░░░ 65% (analyzing temporal coupling)

Estimated: 3s remaining

# If >5s, show what's taking time:
⏱️  Large changeset detected (45 files)
   This will take ~7 seconds

   Pro tip: Consider smaller commits for faster checks
   Or use: crisk check --quick (less accurate, faster)
```

**Quick Mode (Escape Hatch):**
```bash
crisk check --quick

# Skips:
- Temporal coupling analysis (slow)
- Incident similarity (slow)
- Deep dependency analysis (slow)

# Still checks:
- Test coverage (fast)
- Complexity (fast)
- Import count (fast)

Result: <1s but lower accuracy
```

---

## CLI Interaction Patterns

### 1. Progressive Disclosure

**Start simple, reveal complexity on demand:**

```bash
crisk check
# Shows: Risk level + summary

crisk check --verbose
# Shows: Risk level + issues + metrics

crisk check --explain
# Shows: Full investigation trace

crisk check --explain --json
# Shows: Machine-readable output
```

### 2. Smart Defaults

**No configuration required, but customizable:**

```bash
# Works immediately (smart defaults)
crisk check

# Customize if needed
crisk config set block-on high
crisk config set cache-ttl 900
crisk config set team-mode true
```

### 3. Composability

**Commands work together:**

```bash
# Pipeline example
crisk check --format json | jq '.issues[] | select(.severity=="high")'

# Git integration
git diff main | crisk check --stdin

# CI integration
crisk check --fail-on critical --output sarif > report.sarif
```

---

## Integration UX Patterns

### Pre-Commit Hook UX

**Design Goals:**
1. Non-intrusive (runs in <2s)
2. Clear output (one-line summary)
3. Easy override (standard git flag)

**Implementation:**
```bash
# .git/hooks/pre-commit
#!/bin/bash
CRISK_OUTPUT=$(crisk check --pre-commit 2>&1)
CRISK_EXIT=$?

if [ $CRISK_EXIT -eq 0 ]; then
  # Low risk or warnings only
  echo "✅ CodeRisk: Safe to commit"
  exit 0
elif [ $CRISK_EXIT -eq 1 ]; then
  # Medium/High risk (blockable)
  echo "$CRISK_OUTPUT"
  echo ""
  echo "Override with: git commit --no-verify"
  exit 1
else
  # Error (crisk failed to run)
  echo "⚠️  CodeRisk check failed (allowing commit)"
  echo "$CRISK_OUTPUT"
  exit 0  # Don't block on tool failure
fi
```

**User Experience:**
```bash
# Success case (fast)
git commit -m "Fix typo"
✅ CodeRisk: Safe to commit
[main abc1234] Fix typo

# Risk detected case
git commit -m "Add payment processing"
🔴 HIGH risk: Missing tests for payment handling
Override with: git commit --no-verify

# Override
git commit --no-verify -m "Add payment processing"
⚠️  CodeRisk check bypassed (logged)
[main def5678] Add payment processing
```

---

### CI/CD Pipeline UX

**Design Goals:**
1. Fast feedback (run in parallel with tests)
2. Clear failure reasons (GitHub PR comments)
3. Non-blocking for low risk

**GitHub Actions Integration:**
```yaml
name: CodeRisk Check
on: [pull_request]
jobs:
  risk-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
        with:
          fetch-depth: 0  # Full history for temporal analysis

      - name: CodeRisk Analysis
        uses: coderisk/action@v1
        with:
          fail-on: high  # Block PR on HIGH/CRITICAL
          comment: true   # Post results as PR comment

      - name: Upload Results
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: coderisk-report
          path: coderisk-report.json
```

**PR Comment Output:**
```markdown
## 🔍 CodeRisk Analysis

**Risk Level:** ⚠️ MEDIUM

### Summary
- Files changed: 8
- New code: 450 lines
- Test coverage: 72% (↓ from 78%)
- Coupling score: 6/10

### Issues Found

#### ⚠️ Medium Risk (2)
1. **payment_handler.py** - High coupling with database.py
   - Co-changed in 15 of 18 commits (83%)
   - Recommendation: Consider using repository pattern

2. **stripe_client.py** - No error handling for network failures
   - Past incident: INC-453 (timeout cascade)
   - Recommendation: Add circuit breaker pattern

#### ℹ️ Low Priority (1)
3. **utils.py** - Complexity increased from 8 → 12
   - Still within acceptable range (<15)
   - Consider refactoring if it grows further

### Recommendations
1. Add integration tests for payment_handler + database interactions
2. Add retry logic with exponential backoff to stripe_client
3. Review coupling between payment and database layers

### Historical Context
- Similar refactor in PR #234 caused incident INC-453
- Team's average test coverage: 75% (you're at 72%)

**Detailed Report:** [View Full Investigation](https://coderisk.internal/reports/pr-456)

---
🤖 Powered by CodeRisk | [Docs](https://docs.coderisk.com) | [Override Instructions](https://docs.coderisk.com/override)
```

---

## Future UX Enhancements

### 1. VS Code Extension (Phase 2)

**Real-time feedback as you code:**
- Inline warnings (like ESLint)
- Status bar risk indicator
- CodeLens actions ("Run CodeRisk", "Show coupling")
- Problems panel integration

### 2. AI-Powered Fix Suggestions (Enabled via AI Mode)

**Already available via `--ai-mode`:**

AI Mode (Level 4 verbosity) provides:
- Ready-to-execute AI prompts in `ai_assistant_actions[]`
- Confidence scores for auto-fix safety
- Auto-fixable flags for each issue
- Estimated fix times

**Manual invocation:**
```bash
crisk check

⚠️  MEDIUM risk: Missing tests

💡 Let AI fix this?
   → crisk fix-with-ai --tests

# Prompts Claude:
> "Generate tests for auth.py covering:
> - Happy path authentication
> - Invalid token handling
> - Rate limit enforcement"
```

**Automatic invocation (Claude Code/Cursor):**
```typescript
// AI assistant reads AI mode output
const analysis = JSON.parse(await exec('crisk check --ai-mode'));

// Auto-fix high-confidence issues silently
analysis.ai_assistant_actions
  .filter(a => a.confidence > 0.9 && a.ready_to_execute)
  .forEach(action => aiAssistant.execute(action.prompt));
```

### 3. Team Dashboard (Phase 4)

**Visualize team patterns:**
- Which developers generate highest risk code?
- Which file types have most issues?
- Trend over time (improving or degrading?)
- Team leaderboard (gamification)

---

## Key UX Principles Summary

1. **Invisible when safe, visible when risky**
   - Don't annoy developers with noise
   - Surface critical issues clearly

2. **Velocity-preserving**
   - Never block unnecessarily
   - <2s for fast feedback
   - Easy override path

3. **Actionable guidance**
   - Every warning includes "what to do"
   - Suggested commands for next steps
   - Link to full investigation

4. **Team-size adaptive**
   - Solo: Educational, non-blocking
   - Startup: Block on critical only
   - Enterprise: Strict, audit trail

5. **Vibe coding native**
   - Designed for AI-generated code
   - AI-powered fix suggestions (via AI Mode)
   - Pattern learning

6. **Progressive disclosure (4 levels)**
   - Level 1: Quiet (one-line summary)
   - Level 2: Standard (issues + recommendations)
   - Level 3: Explain (full investigation trace)
   - **Level 4: AI Mode (machine-readable structured data)**

7. **AI assistant integration**
   - Machine-readable JSON output (`--ai-mode`)
   - Ready-to-execute AI prompts included
   - Confidence scoring for auto-fix safety
   - Silent quality improvement (fix before user sees code)

8. **Learn and improve**
   - Track developer patterns
   - Personalized suggestions
   - Celebrate progress

---

## Success Metrics (UX Quality)

**Adoption Metrics:**
- % of developers with pre-commit hook installed (target: >80%)
- % of commits with CodeRisk check (target: >90%)
- % of overrides (target: <10%)

**Satisfaction Metrics:**
- Developer NPS (target: >40)
- "CodeRisk saved me from an incident" stories (target: 5/month)
- Time to first value (target: <5 minutes from install)

**Performance Metrics:**
- p50 check latency (target: <2s)
- p95 check latency (target: <5s)
- Cache hit rate (target: >60%)

**AI Assistant Integration Metrics:**
- % of AI-generated code checked before commit (target: >95%)
- % of issues auto-fixed by AI assistants (target: >60%)
- AI assistant adoption rate (Claude Code/Cursor users) (target: >50%)
- Silent fix success rate (fixes applied without user intervention) (target: >80%)

---

**Related Documentation:**
- [developer_workflows.md](developer_workflows.md) - Workflow integration patterns
- [user_personas.md](user_personas.md) - User needs and pain points
- [spec.md](../spec.md) - Technical requirements (NFR-26 to NFR-30: Usability)
- [agentic_design.md](../01-architecture/agentic_design.md) - Agent investigation UX
