# MVP Development Plan: Complete Risk Assessment

**Created:** October 21, 2025
**Updated:** October 22, 2025 (Sequential Analysis Chain)
**Timeline:** 4 weeks (Week 1-4 of strategic plan)
**Goal:** Working `crisk check` with Sequential Analysis Chain (8 specialized agents with validation)
**Success Criteria:** <5s total (p95), <15% FP rate, 2-3 beta users successful

**Core Value Proposition:**
> **"Automate the due diligence developers should do before committing"**
>
> Answer critical questions:
> - Who owns this code? Should I coordinate with them?
> - What files depend on this change? What might break?
> - Has this pattern caused incidents before? Am I repeating history?
> - Did I forget to update related files that usually change together?

**NOT:** "Temporal coupling detector" or "Graph-based static analysis"
**YES:** "Pre-commit due diligence assistant" that prevents regressions through ownership, blast radius, and incident context

**Reference Documents:**
- [mvp_vision.md](mvp_vision.md) - Product vision and scope ("Automated Due Diligence Before Code Review")
- [developer_experience.md](developer_experience.md) - UX examples showing due diligence framing
- [../01-architecture/agentic_design.md](../01-architecture/agentic_design.md) - Investigation flow
- [../01-architecture/risk_assessment_methodology.md](../01-architecture/risk_assessment_methodology.md) - Risk calculation
- [../03-implementation/status.md](../03-implementation/status.md) - Current implementation state
- [strategic_decision_framework.md](strategic_decision_framework.md) - 4-week parallel path strategy

---

## Current State (What We Have)

### ✅ Complete
- **Graph Construction**: 3-layer ingestion (structure, temporal, incidents) via `crisk init`
- **CLI Framework**: Cobra commands (`init`, `check`, `hook`, `incident`, `config`, `login`, `logout`, `whoami`, `status`)
- **Infrastructure**: Neo4j (local Docker), SQLite (validation DB), Viper (config), Tree-sitter (AST parsing)
- **Git Utilities**: Repo detection, diff parsing, history analysis, co-change detection
- **Temporal Analysis**: Commit history, developer ownership, co-change patterns (dynamic computation, no pre-calculated edges)
- **Incident Tracking**: Issue ingestion, incident-to-file linking, BM25 search
- **Output System**: 4 verbosity levels (quiet, standard, explain, AI mode)
- **Phase 1 Metrics**: Coupling, co-change, incident count with adaptive config (basic implementation in `internal/metrics/`)
- **Basic LLM Client**: OpenAI/Anthropic support in `internal/llm/client.go` (single completion method)

### 🚧 Partially Complete (Needs Alignment/Refactoring)
- **crisk check**: Phase 1 metrics work, but uses old SimpleInvestigator instead of 8-agent chain
- **Auth System**: Stub implementation in `cmd/crisk/login.go`, not connected to coderisk.dev
- **Graph Schema**: Mostly aligned with simplified schema, needs verification (no [:CO_CHANGED_WITH], branch filtering)

### ❌ Missing (MVP Blockers)
1. **Sequential Analysis Chain**: 8-agent system with 5 phases (current: single SimpleInvestigator call)
2. **Tier 0 Heuristic Filter**: Trivial change detection (<50ms target)
3. **Phase 1 Standardization**: Consolidate 7 Cypher queries into `internal/risk/collector.go`
4. **LLM Multi-Agent Support**: Gemini Flash 2.0, parallel execution, agent_executor.go
5. **Production Auth Bridge**: Device flow OAuth connecting CLI ↔ coderisk.dev
6. **FR-6 Output Format**: Due diligence checklist in all 4 verbosity modes
7. **False Positive Tracking**: Feedback mechanism with `crisk feedback` command
8. **Integration Tests**: End-to-end testing with real repositories
9. **Performance Optimization**: Achieve <5s total (p95) for full chain

---

## Functional Requirements (No Gaps)

### FR-1: LLM Client Integration (Multi-Agent Support)

**Requirement:** Support OpenAI models only for 8 specialized agents via user-provided API keys (BYOK model)

**SCOPE DECISION (2025-10-22):** MVP limited to OpenAI only to match coderisk-frontend implementation. The frontend only supports OpenAI API key validation and storage. Post-MVP can add Gemini/Anthropic support.

**Model Selection:**
- **Fast Model**: GPT-4o-mini (~300-500ms per agent, ~$0.0001/call) - For quick analysis agents
- **Deep Model**: GPT-4o (~1-2s per agent, ~$0.003/call) - For synthesis and validation agents

**Acceptance Criteria:**
- User can configure API key via environment variable (`OPENAI_API_KEY`)
- User can configure API key via config file (`~/.coderisk/config.yaml`)
- User can configure API key via coderisk-frontend web interface
- Support for parallel agent execution (8 agents can run simultaneously)
- Support for sequential chaining (agents can use prior agent outputs)
- Support for model switching (GPT-4o-mini for fast agents, GPT-4o for deep reasoning)
- Error handling: No API key → graceful degradation (Tier 0 only, warn user)
- Error handling: Rate limit → retry with exponential backoff (3 attempts)
- Error handling: Timeout → fail gracefully, log error
- Token usage logging per agent for cost transparency

**Implementation Notes:**
- Use official SDK: `github.com/sashabaranov/go-openai`
- 8 specialized prompt templates (one per agent: incident, blast_radius, cochange, ownership, quality, patterns, synthesizer, validator)
- Agent prompts: Max tokens 2000, temperature 0.1 (consistent, focused)
- Support for context passing between agents (sequential chain)
- Model selection per agent:
  - Agents 1-5 (Phase 2 specialists): GPT-4o-mini (fast, cheap)
  - Agent 6 (patterns): GPT-4o-mini (pattern matching is straightforward)
  - Agent 7 (synthesizer): GPT-4o (deep reasoning needed for synthesis)
  - Agent 8 (validator): GPT-4o (deep reasoning needed for fact-checking)

**Current Status:**
- ✅ `internal/llm/client.go` exists (OpenAI support with GPT-4o-mini)
- ✅ `internal/agent/llm_client.go` exists (legacy OpenAI client with GPT-4o)
- 🟡 Multi-agent support partially implemented (agent_executor.go skeleton exists)
- ❌ Parallel execution not implemented
- ✅ Frontend support confirmed (OpenAI-only)

**Files to Create:**
- None (skeleton files already exist)

**Files to Update:**
- `internal/llm/client.go` - Add multi-agent interface, model switching, context passing, token tracking
- `internal/llm/agent_executor.go` - Implement parallel and sequential execution

**Files to Remove:**
- `internal/llm/gemini.go` - Remove Gemini skeleton (out of scope for MVP)

**Reference:** [../01-architecture/prompt_engineering_design.md](../01-architecture/prompt_engineering_design.md) (future vision, simplified for MVP)

---

### FR-2: Tier 0 - Heuristic Filter (Trivial Change Detection)

**Requirement:** Ultra-fast filter to skip analysis for truly trivial changes (no LLM, no graph queries)

**Trivial Change Detection:**
```go
// Only these cases skip ALL analysis (Tier 1 + Tier 2)
1. Whitespace-only changes (no functional code)
2. Test-only changes (no production code, no infra tests)
3. Documentation-only changes (markdown with no code snippets)
4. Config files that pass schema validation
```

**Acceptance Criteria:**
- Tier 0 completes in <50ms (p50 latency)
- Returns: TRIVIAL (skip analysis) or NEEDS_ANALYSIS (proceed to Tier 1)
- No false negatives (never marks risky change as TRIVIAL)
- Estimates: ~5-10% of changes are TRIVIAL

**Files to Create:**
- `internal/risk/heuristic.go` - Trivial change detection logic
- `internal/risk/heuristic_test.go` - Unit tests

**Reference:** [../01-architecture/risk_assessment_methodology.md](../01-architecture/risk_assessment_methodology.md)

---

### FR-3: Sequential Analysis Chain - Data Collection & Specialized Agents

**Requirement:** Deep, reliable risk assessment using sequential chain of 8 specialized LLM agents

**Core Insight:** Multi-agent architecture with validation provides most reliable, comprehensive due diligence. Sequential chain allows agents to build on each other's analysis while validation prevents false positives.

**Architecture Overview:**
```
Phase 1: Data Collection (parallel queries)
    ↓
Phase 2: Specialized Analysis (5 agents, parallel)
    ↓
Phase 3: Cross-File Patterns (1 agent)
    ↓
Phase 4: Master Synthesis (1 agent)
    ↓
Phase 5: Validation (1 agent)
```

---

#### Phase 1: Data Collection (~180ms)

**7 Core Cypher Queries (Run for ALL changed files):**
```cypher
1. Recent Bug Fixes (15-25ms):
   // Find PRs that fixed bugs/incidents affecting this file
   MATCH (pr:PR)-[:FIXES]->(i:Issue)
   MATCH (pr)-[:MODIFIES]->(f:File {path: $file_path})
   WHERE pr.merged_at > datetime() - duration('P90D')
     AND (i.labels CONTAINS 'bug' OR i.labels CONTAINS 'incident' OR i.labels CONTAINS 'critical')
   RETURN i.number, i.title, i.labels, pr.number as fix_pr, pr.merged_at
   ORDER BY pr.merged_at DESC
   LIMIT 5

2. Dependency Count (10-20ms):
   MATCH (f:File {path: $file_path})<-[:IMPORTS|CALLS*1..2]-(dependent:File)
   RETURN count(DISTINCT dependent) as dependent_count,
          collect(DISTINCT dependent.path)[0..10] as sample_dependents

3. Co-change Partners (30-60ms, computed dynamically):
   // NOTE: No pre-calculated [:CO_CHANGED_WITH] edges
   // Compute co-change frequency on-the-fly from commit history
   MATCH (f:File {path: $file_path})<-[:MODIFIES]-(c1:Commit)-[:ON_BRANCH]->(b:Branch {is_default: true})
   WHERE c1.author_date > datetime() - duration('P90D')
   WITH f, collect(c1) as commits
   UNWIND commits as c
   MATCH (c)-[:MODIFIES]->(other:File)
   WHERE other.path <> $file_path
   WITH other, count(c) as co_changes, size(commits) as total
   WITH other, co_changes, toFloat(co_changes)/toFloat(total) as frequency
   WHERE frequency > 0.7
   RETURN other.path, frequency, co_changes
   ORDER BY frequency DESC
   LIMIT 5

4. Ownership (30ms):
   MATCH (f:File {path: $file_path})<-[:MODIFIES]-(c:Commit)-[:ON_BRANCH]->(b:Branch {is_default: true})
   MATCH (c)<-[:AUTHORED]-(d:Developer)
   WITH d, count(c) as commits, max(c.author_date) as last_modified
   ORDER BY commits DESC LIMIT 1
   RETURN d.email, d.name, commits, last_modified

5. Blast Radius (40ms):
   MATCH (f:File {path: $file_path})<-[:IMPORTS|CALLS*1..3]-(dependent:File)
   RETURN count(DISTINCT dependent) as dependent_count,
          collect(DISTINCT dependent.path)[0..20] as sample_dependents

6. Bug Fix History (Deep) (60ms):
   // Comprehensive bug fix context for this file
   MATCH (pr:PR)-[:FIXES]->(i:Issue)
   MATCH (pr)-[:MODIFIES]->(f:File {path: $file_path})
   WHERE pr.merged_at > datetime() - duration('P180D')
   RETURN i.number, i.title, i.labels, i.state,
          pr.number as fix_pr, pr.title as fix_title, pr.merged_at
   ORDER BY pr.merged_at DESC
   LIMIT 10

7. Recent Commits (40ms):
   MATCH (f:File {path: $file_path})<-[:MODIFIES]-(c:Commit)-[:ON_BRANCH]->(b:Branch {is_default: true})
   MATCH (c)<-[:AUTHORED]-(d:Developer)
   RETURN c.sha, c.message, d.email, c.author_date, c.additions, c.deletions
   ORDER BY c.author_date DESC
   LIMIT 5
```

**Total Query Time: ~180ms per file** (all 7 queries, parallelized)

**Output per file: ~1,600 tokens** (metadata + all query results)

**Note:** Test coverage removed - no reliable way to determine this from graph schema. Would require external code coverage tools integration (deferred to post-MVP).

**See:** [../01-architecture/simplified_graph_schema.md](../01-architecture/simplified_graph_schema.md) for full schema

---

#### Phase 2: Specialized Analysis (~500ms, parallel)

**5 Specialized Agents (run simultaneously):**

**Agent 1: Bug Fix History Specialist**
- Input: File diff + bug_fix_prs + bug_fix_history + recent_commits
- Task: Analyze past bug fixes affecting this file, detect regression patterns
- Output: Recent bugs, fix patterns, regression risks, similar past fixes
- Prompt focus: "Has this file been part of recent bug fixes? Is this change similar to past fixes or potentially reintroducing similar bugs?"

**Agent 2: Blast Radius Specialist**
- Input: File diff + dependencies + blast_radius + co_change
- Task: Assess impact scope, identify affected systems
- Output: Dependent systems, high-impact files, integration test recommendations
- Prompt focus: "What will break if this changes? What systems are impacted?"

**Agent 3: Co-change & Forgotten Updates Specialist**
- Input: File diff + co_change + recent_commits + bug_fix_history + all_changed_files
- Task: Identify temporal coupling, detect forgotten updates
- Output: Forgotten files, co-change patterns, historical evidence of forgotten updates
- Prompt focus: "What files should change together? Did developer forget to update related files?"

**Agent 4: Ownership & Coordination Specialist**
- Input: File diff + ownership + recent_commits + bug_fix_history
- Task: Determine coordination needs, identify who to contact
- Output: Primary owner, last modifier, who to ping, suggested reviewers
- Prompt focus: "Who owns this? Should developer coordinate before committing?"

**Agent 5: Code Quality & Change Scope Specialist**
- Input: File diff + recent_commits + bug_fix_history
- Task: Assess change size, complexity, and quality concerns
- Output: Change risk level, complexity assessment, quality recommendations
- Prompt focus: "Is this change too large? Are there code quality concerns?"

**Agent Context Strategy: Overlapping Context**
- Each agent receives full context for assigned files (prevents missing information)
- Agents can reference adjacent data (e.g., bug fix agent sees ownership for coordination suggestions)

**Acceptance Criteria:**
- Phase 2 completes in <500ms (all 5 agents run in parallel)
- Each agent outputs structured JSON
- All agents use Gemini Flash 2.0 (~$0.0001 total for 5 agents)

**Files to Create:**
- `internal/risk/agents/incident.go` - Agent 1 implementation
- `internal/risk/agents/blast_radius.go` - Agent 2 implementation
- `internal/risk/agents/cochange.go` - Agent 3 implementation
- `internal/risk/agents/ownership.go` - Agent 4 implementation
- `internal/risk/agents/quality.go` - Agent 5 implementation

**Reference:** [../01-architecture/risk_assessment_methodology.md](../01-architecture/risk_assessment_methodology.md)

---

### FR-4: Sequential Analysis Chain - Synthesis & Validation

**Requirement:** Combine agent outputs into final due diligence report with validation to prevent false positives

**Core Purpose:** Transform specialized agent analyses into actionable due diligence checklist (FR-6 format) with fact-checking

---

#### Phase 3: Cross-File Pattern Detection (~800ms)

**Agent 6: Pattern Detector**
- Input: All files + all 5 agent outputs + raw query data
- Task: Detect systemic risks across multiple files
- Output: Cross-file patterns, systemic risks, architectural concerns
- Prompt focus: "Are there patterns across files? Architectural violations? Cascading risks?"

**Pattern Examples:**
- "All payment files changed, but fraud_detector.py not updated"
- "Changes to processor.py may cause timeout cascade to reporting.py"
- "Core authentication files modified without updating dependent services"

**Acceptance Criteria:**
- Detects risks no single-file agent would catch
- Outputs structured JSON with pattern descriptions
- Completes in <800ms

**Files to Create:**
- `internal/risk/agents/patterns.go` - Agent 6 implementation

---

#### Phase 4: Master Synthesis (~1,000ms)

**Agent 7: Synthesizer**
- Input: All 6 agent outputs + pattern detection + raw data
- Task: Create final due diligence report in FR-6 format
- Output: Structured report with ownership, blast radius, forgotten updates, incidents, recommendations
- Prompt focus: "Combine all analyses into due diligence checklist with prioritized recommendations"

**Output Format (FR-6 Standard Mode):**
```
👤 OWNERSHIP
   • Last modified by X, Y owns this file
   → Consider pinging @X or @Y

🔗 BLAST RADIUS
   • N files depend on this
   → Changes may break downstream systems

🔄 FORGOTTEN UPDATES?
   • file.py changed together in N% of commits
   → You likely need to update file.py too

⚠️ INCIDENT HISTORY
   • N past incidents
   → Similar changes caused failures recently

RECOMMENDATIONS (prioritized):
  1. CRITICAL: [from agents]
  2. HIGH: [from agents]
  3. MEDIUM: [from agents]
```

**Recommendation Prioritization:**
1. CRITICAL: Coordination with recent contributors, forgotten updates with incident history
2. HIGH: Test coverage gaps, blast radius concerns
3. MEDIUM: Architectural improvements, refactoring suggestions

**Acceptance Criteria:**
- Output matches FR-6 format exactly (all 4 verbosity levels)
- Combines all agent insights without losing information
- Prioritizes recommendations correctly
- Completes in <1,000ms

**Files to Create:**
- `internal/risk/agents/synthesizer.go` - Agent 7 implementation
- `internal/output/fr6_formatter.go` - FR-6 format generator

---

#### Phase 5: Validation & Fact-Checking (~800ms)

**Agent 8: Validator**
- Input: Final report from Agent 7 + raw query data
- Task: Verify every claim against raw data, catch hallucinations
- Output: Validated report or corrections
- Prompt focus: "Cross-check all facts. Flag any hallucinations, fabrications, or unsupported claims."

**Validation Checks:**
- All file names exist in raw data
- All developer names/emails exist in ownership data
- All incident numbers exist in incident data
- All co-change percentages match query results
- All dependency counts match query results
- All recommendations grounded in actual data

**Error Handling:**
- If hallucinations found → Correct them, mark severity
- If fabrications found → Remove them, log warning
- If unsupported claims → Request clarification from Agent 7 or remove

**Primary False-Positive Prevention:**
This validation stage is the key mechanism for ensuring reliability and preventing false positives.

**Acceptance Criteria:**
- Catches all hallucinated file/developer/incident names
- Corrects wrong statistics automatically
- Final report is 100% grounded in raw data
- Completes in <800ms

**Files to Create:**
- `internal/risk/agents/validator.go` - Agent 8 implementation
- `internal/risk/validation.go` - Validation utilities

---

**Total Sequential Chain Time: ~3.3 seconds**
- Phase 1: 180ms (7 queries)
- Phase 2: 500ms (5 agents parallel)
- Phase 3: 800ms (patterns)
- Phase 4: 1,000ms (synthesis)
- Phase 5: 800ms (validation)

**Total Cost per 5-file PR: ~$0.0016** (8 agents × ~$0.0002 each)

**Reference:** [../01-architecture/agentic_design.md](../01-architecture/agentic_design.md) (simplified sequential chain for MVP)

---

### FR-5: End-to-End `crisk check` Flow (Sequential Analysis Chain)

**Requirement:** Complete workflow from git diff to risk report using 5-phase sequential chain

**Flow:**
```
1. User runs: crisk check [file_path]
   ├─ If no file_path: Analyze all changed files (git diff)
   ├─ If file_path provided: Analyze specific file

2. Tier 0: Heuristic Filter (<50ms)
   ├─ For each changed file:
   │  ├─ Check if trivial (whitespace, test-only, docs-only)
   │  ├─ If TRIVIAL → Mark as SAFE, skip to output
   │  └─ If NEEDS_ANALYSIS → Proceed to Phase 1
   └─ Estimated: 5-10% of files are TRIVIAL

3. Phase 1: Data Collection (~180ms per file)
   ├─ For each non-trivial file:
   │  ├─ Extract metadata (path, diff, additions, deletions, hunks)
   │  ├─ Run all 7 Cypher queries in parallel
   │  └─ Store complete dataset

4. Phase 2: Specialized Analysis (~500ms, parallel)
   ├─ Launch 5 agents simultaneously:
   │  ├─ Agent 1: Incident Risk Specialist
   │  ├─ Agent 2: Blast Radius Specialist
   │  ├─ Agent 3: Co-change & Forgotten Updates Specialist
   │  ├─ Agent 4: Ownership & Coordination Specialist
   │  └─ Agent 5: Code Quality & Change Scope Specialist
   └─ Each agent outputs structured JSON

5. Phase 3: Cross-File Pattern Detection (~800ms)
   └─ Agent 6: Pattern Detector analyzes all files for systemic risks

6. Phase 4: Master Synthesis (~1,000ms)
   └─ Agent 7: Synthesizer creates final report in FR-6 format

7. Phase 5: Validation (~800ms)
   └─ Agent 8: Validator fact-checks report, prevents false positives

8. Output Generation (see FR-6)
   ├─ Format validated report based on verbosity level
   ├─ Determine overall risk (highest risk level)
   └─ Exit with appropriate code (0=SAFE, 1=REVIEW, 2=HIGH, 3=CRITICAL)
```

**Total Time: ~3.3 seconds** (Phase 2 runs in parallel)

**Acceptance Criteria:**
- `crisk check` works with no arguments (analyzes git diff)
- `crisk check path/to/file.py` works for specific file
- Tier 0 always runs (<50ms)
- Sequential chain completes in <5s (p95)
- All files get full analysis (no conditional execution except Tier 0)
- Exit codes match risk levels (for pre-commit hook integration)
- Progress indicators shown during analysis (Phase 1 → 2 → 3 → 4 → 5)
- Cost transparency: Per-agent token usage shown in --explain mode
- Validation prevents false positives

**Files to Create:**
- `internal/risk/collector.go` - Phase 1 data collection
- `internal/risk/chain_orchestrator.go` - Sequential chain coordination
- `cmd/crisk/check.go` - Main entry point (updated)

**Files to Update:**
- Remove `internal/risk/tier1.go`, `internal/risk/tier2.go` (replaced by agents)

---

### FR-6: Output Formatting (4 Verbosity Levels)

**Requirement:** Adaptive output based on verbosity level

**Verbosity Levels:**

#### 1. Quiet Mode (`--quiet` or `-q`)
```
HIGH (2 files)
```
- One line: Risk level + file count
- Exit code only (for CI/CD)

#### 2. Standard Mode (default)
```
⚠️  HIGH RISK - Pre-commit due diligence needed

payment_processor.py (HIGH)
  📋 DUE DILIGENCE CHECKLIST:

  👤 OWNERSHIP
     • Last modified by Alice 2 days ago (bug fix INC-453)
     • Bob owns this file (80% of commits)
     → Consider pinging @alice or @bob before making changes

  🔗 BLAST RADIUS
     • 15 files depend on this (fraud_detector.py, reporting.py, webhooks.py, ...)
     → Changes here may break downstream systems

  🔄 FORGOTTEN UPDATES?
     • fraud_detector.py changed with this file in 17/20 commits (85%)
     → You likely need to update fraud_detector.py too

  ⚠️  INCIDENT HISTORY
     • 3 past incidents linked to this file
     • INC-453 (2 days ago): Timeout cascade in payment flow
     → Similar changes caused failures recently

RECOMMENDATIONS:
  1. Coordinate with @alice (she just fixed INC-453 here)
  2. Review fraud_detector.py - likely needs update too
  3. Add tests for payment_processor.py (0% coverage)
  4. Add timeout handling (pattern from INC-453)
```
- Summary + due diligence checklist + coordination plan

#### 3. Explain Mode (`--explain` or `-e`)
```
[Standard output PLUS:]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
PHASE 1 ANALYSIS (145ms)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

payment_processor.py:
  Metrics Calculated:
    ✅ Ownership: Bob (80% commits), Alice (last modified 2 days ago)
    ⚠️  Coupling: 15 incoming dependencies → HIGH
    ⚠️  Co-change: 0.85 with fraud_detector.py (17/20 commits) → HIGH
    ⚠️  Test coverage: 0% → CRITICAL
    ⚠️  Incidents: 3 linked (INC-453, INC-421, INC-387) → HIGH

  Risk Level: HIGH (escalating to Phase 2)
  Reason: Multiple high-risk signals + incident history

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
PHASE 2 ANALYSIS (3.2s)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

LLM Provider: GPT-4o-mini
Tokens: 1,234 input, 456 output
Cost: $0.002 (charged to your OpenAI account)

Due Diligence Assessment:
  Risk Level: HIGH → CRITICAL (elevated due to incident pattern)

  Coordination Needed:
    • Contact @alice - she just worked on this for INC-453
    • Contact @bob - primary owner, needs visibility
    • Contact @sarah - fraud expert, fraud_detector.py likely impacted

  Forgotten Updates Detected:
    • fraud_detector.py changed together in 85% of commits
    • Last time payment_processor.py changed without fraud_detector.py: INC-453
    • High likelihood of breakage if not updated together

  Incident Pattern:
    • Similar to INC-453 (timeout cascade)
    • Payment changes → fraud_detector timeouts → cascade failure
    • Prevention: Add timeout handling, update fraud_detector.py

  LLM Reasoning:
    "This file has a strong coupling pattern with fraud_detector.py.
    Historical data shows that 85% of payment_processor.py changes
    also require fraud_detector.py updates. The recent INC-453 incident
    2 days ago suggests Alice made a critical fix. Changing this file
    now without coordination risks regression."

Recommendations (prioritized):
  1. 🔴 CRITICAL: Ping @alice - understand her INC-453 fix before proceeding
  2. 🔴 CRITICAL: Review fraud_detector.py - likely needs parallel update
  3. 🟡 HIGH: Add integration tests (payment + fraud detection)
  4. 🟡 HIGH: Add timeout handling (pattern from INC-453)
  5. 🟢 MEDIUM: Consider refactoring to reduce coupling
```
- Includes timing, metrics, escalation logic, LLM details
- **Due diligence framing**: Ownership, coordination, forgotten updates, incident patterns

#### 4. AI Mode (`--ai`)
```json
{
  "overall_risk": "HIGH",
  "due_diligence_required": true,
  "files": [
    {
      "path": "payment_processor.py",
      "risk_level": "HIGH",
      "ownership": {
        "primary_owner": "bob@company.com",
        "last_modified_by": "alice@company.com",
        "last_modified_date": "2025-10-19",
        "last_modified_reason": "bug fix INC-453"
      },
      "blast_radius": {
        "dependent_files_count": 15,
        "high_impact_files": [
          "fraud_detector.py",
          "reporting.py",
          "webhooks.py"
        ]
      },
      "coordination": {
        "should_contact_owner": true,
        "should_contact_others": [
          "@alice (just worked on INC-453)",
          "@sarah (fraud expert)"
        ],
        "reason": "Recent incident fix + high coupling"
      },
      "forgotten_updates": {
        "likely_files": ["fraud_detector.py"],
        "co_change_rate": 0.85,
        "reason": "Changed together in 17/20 commits"
      },
      "incident_history": {
        "count": 3,
        "recent": "INC-453",
        "pattern": "timeout cascade",
        "similar_risk": true
      },
      "phase1": {
        "duration_ms": 45,
        "metrics": {
          "coupling": 15,
          "co_change": 0.85,
          "test_coverage": 0.0,
          "incidents": 3,
          "churn": 12
        }
      },
      "phase2": {
        "duration_ms": 3200,
        "llm_provider": "openai",
        "tokens_used": 1690,
        "cost_usd": 0.002,
        "reasoning": "Strong coupling pattern with fraud_detector.py. Recent INC-453 suggests high regression risk.",
        "recommendations": [
          "Ping @alice - understand her INC-453 fix",
          "Review fraud_detector.py - likely needs update",
          "Add integration tests",
          "Add timeout handling"
        ]
      }
    }
  ]
}
```
- JSON output for IDE/CI integration
- **Includes due diligence fields**: ownership, coordination, forgotten updates, incident patterns

**Acceptance Criteria:**
- All 4 modes work correctly
- Output matches examples above
- Exit codes consistent across modes
- Progress indicators only shown in standard/explain modes

**Files to Update:**
- `internal/output/*.go` - Already exists, integrate Phase 2 results
- `cmd/crisk/check.go` - Pass verbosity flag to output formatter

**Reference:** [developer_experience.md](developer_experience.md) - UX design

---

### FR-7: Query Strategy - Fixed Query Library

**Requirement:** Safe, predictable Cypher queries for risk assessment

**Design Decision:** Use **fixed query library** with parameterized templates (no dynamic query generation)

**Core Insight:** With a simplified schema (8 nodes, 13 edges), we need only 7 core query patterns to cover all risk signals. Running all queries per file is fast enough (~180ms total) and simpler than dynamic selection.

**Query Library (7 Fixed Templates):**

**Phase 1 Queries (Always run for every changed file):**
1. Recent Bug Fixes - PRs that fixed bugs/incidents affecting this file (last 90 days)
2. Dependency Count - Files depending on this file (depth 1-2)
3. Co-change Partners - Files frequently changed together (>70% rate, last 90 days)
4. Ownership - Primary owner by commit count (last 90 days)
5. Blast Radius - Full dependency graph with sample dependents (depth 3)
6. Bug Fix History (Deep) - All bug fix PRs in last 180 days with issue details
7. Recent Commits - Last 5 commits with diffs and authors

**All 7 queries run in Phase 1 (data collection) before any LLM agents execute**

**All queries:**
- Parameterized (file path, time window, depth limits)
- Include `LIMIT` clauses (max 50 rows)
- Filter by default branch (`is_default: true`) for temporal queries
- Audited and tested
- No dynamic query generation (fixed templates only)

**LLM Role:** Agents interpret query results (not generate queries)

**Benefits:**
- ✅ Predictable performance (~180ms for all 7 queries)
- ✅ Simple to audit and test
- ✅ No query injection risk
- ✅ All data collected upfront (agents don't wait on queries)
- ✅ All queries aligned with solidified graph schema

**Tradeoffs:**
- ⚠️ May run queries that return no results (~20ms per unused query)
- ✅ But total query time still only ~180ms (acceptable)

**Files to Create:**
- `internal/risk/queries.go` - 7 fixed query template functions
- `internal/risk/input_normalizer.go` - Git diff → structured ChangeSet parser

**Reference:** [../01-architecture/simplified_graph_schema.md](../01-architecture/simplified_graph_schema.md) - Query examples

---

### FR-9: Graph Schema Simplification

**Requirement:** Use simplified graph schema with minimal edge types and branch support for data integrity

**Core Decisions:**
1. **Remove [:CO_CHANGED_WITH] pre-calculated edges** - Compute dynamically instead
2. **Add minimal Branch support** - Prevent feature branch pollution in metrics

**Rationale:**
- Pre-calculated co-change edges add complexity without meaningful performance benefit
- Adds ~1,200 edges to graph for typical medium repo
- Requires complex ingestion step (calculate from commits, store edges)
- Data becomes stale (only updated on `crisk init`)
- Can compute co-change frequency dynamically in ~50ms with proper indexes
- **Branch tracking needed**: Without branches, feature branch commits pollute main branch metrics (ownership, co-change, incidents)

**Simplified Schema:**
- **8 Node Types**: File, Function, Class, Developer, Commit, Branch, Issue, PR
- **13 Edge Types**: CONTAINS, CALLS, IMPORTS, AUTHORED, MODIFIES, LINKED_TO, FIXES, ON_BRANCH, FROM_BRANCH, TO_BRANCH, MERGED_AS (removed: CO_CHANGED_WITH)
- **Total edges**: ~10,000 structure + ~100 branch edges = 10,100 (vs ~11,200 with co-change edges) = -10% complexity

**Branch Support (Minimal Scope):**
- **Branch Node**: `(:Branch {name, is_default})`
- **Branch Edges**:
  - `(Commit)-[:ON_BRANCH]->(Branch)` - Which branch this commit is on
  - `(PR)-[:FROM_BRANCH]->(Branch)` - Source branch (head_ref)
  - `(PR)-[:TO_BRANCH]->(Branch)` - Target branch (base_ref)
  - `(PR)-[:MERGED_AS]->(Commit)` - Link PR to merge commit
- **Query Filtering**: All temporal queries (co-change, ownership, incidents) filter by `is_default: true`
- **Data Source**: GitHub API already provides `head_ref`, `base_ref`, `merge_commit_sha` in PR data (PostgreSQL stores this)

**Benefits:**
- ✅ Simpler schema (easier to understand and maintain)
- ✅ Faster ingestion (skip co-change calculation step)
- ✅ Always-fresh data (compute from latest commits)
- ✅ Flexible thresholds (can adjust frequency filter per query)
- ✅ Less storage (fewer edges overall)
- ✅ **Data integrity** (branch filtering prevents feature branch pollution)

**Tradeoffs:**
- ⚠️ Co-change query takes ~50ms instead of ~10ms
- ⚠️ Branch filtering adds ~5-10ms to queries
- ✅ But still meets <600ms Tier 1 target (~100ms queries + 500ms LLM = 600ms)
- ✅ Cache results at application level (15-min TTL) for repeated queries

**Implementation:**
- Remove `internal/graph/builder.go:calculateCoChangedEdges()`
- Add `internal/graph/branches.go` - Branch node creation and linking
- Update Tier 1 queries to compute co-change dynamically AND filter by default branch
- Update Tier 2 queries to filter by default branch
- Add filesystem cache for co-change results

**Files to Create:**
- `internal/graph/branches.go` - Branch ingestion logic (extract from PR data in PostgreSQL)

**Files to Update:**
- `internal/graph/builder.go` - Add branch node creation step, remove calculateCoChangedEdges()
- `internal/risk/tier1.go` - Update queries to include branch filtering
- `internal/risk/tier2.go` - Update queries to include branch filtering

**Reference:** [../01-architecture/simplified_graph_schema.md](../01-architecture/simplified_graph_schema.md)

---

### FR-10: Performance Optimization

**Requirement:** Meet latency targets for MVP (Sequential Analysis Chain)

**Targets:**
- Tier 0 (p50): <50ms (heuristic filter)
- Phase 1 (p50): <180ms (all 7 queries)
- Phase 2 (p50): <500ms (5 agents, parallel)
- Phases 3-5 (p95): <2.6s (sequential: patterns, synthesis, validation)
- **Total (p95): <5s** (Tier 0 + all 5 phases)

**Optimizations:**

1. **Neo4j Query Optimization**
   - Add indexes on frequently queried properties:
     - `File.path` (PRIMARY - used by all queries)
     - `Commit.sha`
     - `Commit.author_date` (for temporal filtering)
     - `Branch.name` (for branch filtering)
     - `Issue.created_at` (for recent incident filtering)
     - `Developer.email`
   - Use `LIMIT` clauses in all queries (max 50 results)
   - Phase 1: 7 queries run in parallel (~180ms total with dynamic co-change + branch filtering)
   - **No [:CO_CHANGED_WITH] edge computation** - simplifies ingestion by 30%
   - **Branch filtering** - All temporal queries filter by `is_default: true`

2. **Parallel Agent Execution (Phase 2)**
   - Run 5 specialized agents simultaneously
   - Each agent gets independent LLM call (no blocking)
   - Total Phase 2 time = slowest agent (~500ms), not sum of all agents
   - Use goroutines for parallel execution

3. **LLM Call Optimization**
   - All agents: Gemini Flash 2.0 (300-500ms per call)
   - Max tokens per agent: 2000 input, 500 output
   - 10s timeout per agent (fail gracefully if exceeded)
   - Context pruning: Max 20 dependencies, 10 co-change partners, 10 recent commits

4. **Sequential Optimization (Phases 3-5)**
   - Phase 3 (patterns): Single agent, focused on cross-file analysis only
   - Phase 4 (synthesis): Single agent, combines pre-computed agent outputs (minimal LLM work)
   - Phase 5 (validation): Fact-checking only (deterministic checks + minimal LLM)

5. **No Caching (Simpler for MVP)**
   - Agent outputs are always fresh (no stale data)
   - Validation ensures accuracy (no cached false positives)
   - Total time <5s is acceptable for MVP
   - Post-MVP: Consider caching Phase 1 query results if needed

**Acceptance Criteria:**
- Benchmarks show p50 <50ms (Tier 0)
- Benchmarks show p50 <180ms (Phase 1 queries)
- Benchmarks show p50 <500ms (Phase 2 parallel agents)
- Benchmarks show p95 <5s (total chain)
- All files get complete analysis (no conditional skipping)

**Files to Update:**
- `internal/graph/*.go` - Add query optimization and indexes
- `internal/risk/chain_orchestrator.go` - Parallel agent execution
- `cmd/crisk/check.go` - Sequential phase coordination

---

### FR-11: False Positive Tracking

**Requirement:** User feedback mechanism to improve accuracy over time

**Commands:**

```bash
# User provides feedback on false positive
crisk feedback --false-positive "payment_processor.py" --reason "This change was safe, no coupling issue"

# View false positive stats
crisk stats --false-positives
```

**Data Storage:**
- SQLite database (`.coderisk/validation.db`)
- Schema:
  ```sql
  CREATE TABLE feedback (
    id INTEGER PRIMARY KEY,
    file_path TEXT,
    git_sha TEXT,
    risk_level TEXT,
    was_false_positive BOOLEAN,
    reason TEXT,
    timestamp INTEGER
  );
  ```

**Stats Output:**
```
False Positive Rate: 12.5% (5/40 checks)

By Risk Level:
  HIGH: 20% (2/10)
  MEDIUM: 10% (3/30)
  LOW: 0% (0/0)

Recent False Positives:
  payment_processor.py (HIGH) - "No coupling issue"
  auth.py (MEDIUM) - "Tests exist but not detected"
```

**Acceptance Criteria:**
- User can submit feedback via `crisk feedback`
- Feedback stored in SQLite
- Stats calculated and displayed via `crisk stats`
- FP rate calculation: `false_positives / total_checks`

**Files to Create:**
- `internal/feedback/tracker.go` - Feedback collection
- `internal/feedback/stats.go` - Stats calculation
- `cmd/crisk/feedback.go` - CLI command (new)
- `cmd/crisk/stats.go` - Update to include FP stats

---

### FR-12: Configuration Management

**Requirement:** User can configure API keys, thresholds, verbosity defaults

**Configuration Sources (priority order):**
1. Command-line flags (highest priority)
2. Environment variables
3. Config file (`~/.coderisk/config.yaml`)
4. Defaults (lowest priority)

**Config File Format:**
```yaml
# ~/.coderisk/config.yaml
llm:
  provider: "openai"  # OpenAI only for MVP
  api_key: "sk-..."  # Or use env var OPENAI_API_KEY
  fast_model: "gpt-4o-mini"  # For Agents 1-6 (fast analysis)
  deep_model: "gpt-4o"  # For Agents 7-8 (synthesis, validation)
  max_tokens_fast: 2000
  max_tokens_deep: 4000
  timeout_seconds: 30

thresholds:
  # Thresholds now primarily used in Tier 1 LLM context
  # (not hard-coded rules like before)
  dependency_count_high: 20  # Flag for Tier 1 context
  co_change_rate_high: 0.7   # Flag for Tier 1 context
  incident_lookback_days: 90  # Recent incidents window

output:
  default_verbosity: "standard"  # "quiet", "standard", "explain", "ai"
  show_progress: true
  show_cost: true  # Show cost in --explain mode

cache:
  enabled: true
  ttl_minutes: 15
  max_size_mb: 100
```

**Environment Variables:**
- `OPENAI_API_KEY` - OpenAI API key (required for MVP)
- `CODERISK_VERBOSITY` - Default verbosity level
- `CODERISK_CONFIG` - Path to config file

**Acceptance Criteria:**
- User can configure via file, env vars, or flags
- Priority order respected
- `crisk config get llm.fast_model` shows current value
- `crisk config set llm.fast_model gpt-4o-mini` updates config file
- Validation: Invalid values rejected with helpful error
- OpenAI API key validation via coderisk-frontend

**Files to Update:**
- `internal/config/*.go` - Already exists, add LLM/threshold config
- `cmd/crisk/config.go` - Already exists, add get/set commands

---

### FR-13: Integration Testing

**Requirement:** Validate end-to-end flow with real repositories

**Test Repositories:**
1. **Small**: `commander.js` (~50 files, 5K LOC)
2. **Medium**: `omnara` (~400 files, 50K LOC)
3. **Large**: `hashicorp/terraform-exec` (~200 files, 30K LOC)

**Test Scenarios:**

#### Scenario 1: Tier 0 Only (TRIVIAL changes)
```bash
cd /tmp/commander.js
crisk init
git checkout -b test-change
# Make trivial change (whitespace only)
crisk check
# Expected: Tier 0 only, <50ms, SAFE risk
```

#### Scenario 2: Tier 1 Only (SAFE/REVIEW changes)
```bash
cd /tmp/commander.js
# Make low-risk change (add comment, minor refactor)
crisk check
# Expected: Tier 0 → Tier 1, <600ms, SAFE or REVIEW_NEEDED
```

#### Scenario 3: Full Escalation (Tier 0 → Tier 1 → Tier 2)
```bash
cd /tmp/omnara
crisk init
# Make high-risk change (modify core file with incidents)
crisk check
# Expected: Tier 0 → Tier 1 → Tier 2, <5s, ESCALATE + due diligence
```

#### Scenario 4: Performance Benchmark
```bash
cd /tmp/terraform-exec
crisk init
# Measure init time (target: <10 min)
crisk check
# Measure check time (target: <600ms Tier 1)
```

#### Scenario 5: Cache Hit Rate
```bash
crisk check  # First run (cold cache)
crisk check  # Second run (should hit cache)
crisk stats  # Verify cache hit rate >30%
```

**Acceptance Criteria:**
- All scenarios pass
- Performance targets met
- No crashes or errors
- Output format correct

**Files to Create:**
- `test/integration/test_risk_assessment.sh` - Shell script for scenarios
- `test/integration/README.md` - Test documentation

---

### FR-14: Beta-Ready Release

**Requirement:** Installable, usable product for 2-3 beta users

**Installation:**
```bash
# Homebrew (primary)
brew tap rohankatakam/coderisk
brew install crisk

# Direct download (fallback)
curl -fsSL https://coderisk.dev/install.sh | sh
```

**Homebrew Formula:**
- Update `homebrew-coderisk/Formula/crisk.rb`
- Include binaries for macOS (Intel + ARM), Linux (x64, ARM)
- Include Docker Compose file (for Neo4j)

**Documentation (Minimum Viable):**

1. **README.md** (Quick start)
   - Installation instructions
   - Basic usage (`crisk init`, `crisk check`)
   - Pre-commit hook setup

2. **INSTALLATION.md** (Detailed setup)
   - System requirements
   - Docker setup
   - API key configuration
   - Troubleshooting

3. **USAGE.md** (Command reference)
   - All commands documented
   - Examples for each verbosity level
   - Configuration options

4. **TROUBLESHOOTING.md** (Common issues)
   - Neo4j connection errors
   - LLM API key issues
   - Performance problems
   - False positive feedback

**Acceptance Criteria:**
- 2-3 beta users can install via Homebrew
- Users can run `crisk init` successfully
- Users can run `crisk check` and get useful results
- Documentation sufficient for self-service

**Files to Update:**
- `README.md` - Update with beta installation instructions
- Create: `docs/INSTALLATION.md`, `docs/USAGE.md`, `docs/TROUBLESHOOTING.md`

---

## Prompt Templates (3-Tier System)

### Tier 1 Prompt (Fast Triage):

```
You are a quick risk screener for code changes.

FILE: {file_path}
DIFF:
{git_diff}

QUICK CONTEXT:
- Recent incidents (last 90 days): {incident_count}
  {incident_list}
- Files depending on this: {dependent_count}
  {sample_dependents}
- Co-change partners (>70% rate):
  {co_change_partners}

TASK: Determine if this change needs deeper analysis.

Respond with ONE of:
1. SAFE - Low risk, no coordination needed
   Example: Minor refactor, logging, comments in low-traffic file

2. REVIEW_NEEDED - Medium risk, developer should double-check
   Example: Logic changes in moderately-coupled file

3. ESCALATE - High risk, needs detailed due diligence
   Example: Changes to file with recent incidents or high coupling

Response format (JSON):
{
  "decision": "SAFE|REVIEW_NEEDED|ESCALATE",
  "one_line_reason": "Brief explanation (max 10 words)",
  "key_concern": "Primary risk factor (if any)"
}
```

### Tier 2 Prompt (Deep Due Diligence):

```
You are a code risk assessment expert helping developers perform pre-commit due diligence.

A developer is about to commit changes to: {file_path}

DUE DILIGENCE CONTEXT:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
1. OWNERSHIP
   - Primary owner: {owner_name} ({owner_email})
   - Last modified: {last_modified_date} by {last_modifier}
   - Commit frequency: {commit_count} commits in last 30 days

2. BLAST RADIUS (What depends on this?)
   - {coupling_count} files depend on this file:
     {dependency_list}

3. CO-CHANGE PATTERNS (What should change together?)
   - Files that frequently change with this file:
     {co_change_partners}
   - Forgotten updates may cause: {potential_breakage}

4. INCIDENT HISTORY (Has this failed before?)
   - Past incidents: {incident_count}
     {incident_summaries}
   - Pattern: {incident_pattern}

5. RECENT CODE CHANGES (Last 5 commits)
   {recent_patches}

   Example format:
   Commit abc123 (Alice, 2 days ago): "Fix timeout cascade"
     payment_processor.py: +15 -3
     @@ -42,7 +42,10 @@ def process_payment():
     -    result = api.charge()
     +    try:
     +        result = api.charge(timeout=5)
     +    except TimeoutError:
     +        return rollback()

6. TEST COVERAGE
   - Current coverage: {test_ratio}
   - Test files: {test_files}

CURRENT CHANGE (Git Diff):
{git_diff}

TIER 1 TRIAGE: {tier1_decision}
REASON: {tier1_reason}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
YOUR TASK: Provide a pre-commit risk assessment and action plan

Consider these due diligence questions:
1. Should the developer coordinate with the file owner before making this change?
2. What dependent files might break? Should they be updated together?
3. Is this similar to a past incident pattern? What should be done differently?
4. Is test coverage adequate for this change?
5. Who should review this change (beyond the owner)?

Respond in JSON format:
{
  "risk_level": "LOW|MEDIUM|HIGH|CRITICAL",
  "due_diligence_summary": "1-2 sentence summary of key risks",
  "coordination_needed": {
    "should_contact_owner": true/false,
    "should_contact_others": ["@alice (fraud expert)", "@bob (knows session logic)"],
    "reason": "why coordination is needed"
  },
  "forgotten_updates": {
    "likely_forgotten_files": ["file1.py", "file2.py"],
    "reason": "these files changed together in 15/20 commits"
  },
  "incident_risk": {
    "similar_incident": "INC-453",
    "pattern": "timeout cascade in auth flow",
    "prevention": "Add timeout handling, test with session_manager.py"
  },
  "recommendations": [
    "Action 1 (most important)",
    "Action 2",
    "Action 3"
  ]
}
```

**Key Differences from Generic Risk Assessment:**
- ✅ **Ownership-focused** - Surfaces who to talk to
- ✅ **Coordination-focused** - Identifies forgotten updates
- ✅ **Incident-focused** - Prevents repeating history
- ✅ **Actionable** - Tells developer exactly what to do

**Token Budget:**
- Tier 1: Max 500 tokens (input + output) - minimal context for triage
- Tier 2: Max 1500 tokens (input + output) - full context for due diligence

**Fallback:** If Tier 2 LLM fails, return Tier 1 result + basic ownership/dependency info

---

## Testing Strategy

### Unit Tests (Target: >60% Coverage)

**Priority Packages:**
- `internal/llm/` - LLM client integration (80% coverage target)
- `internal/risk/` - Risk calculation and 3-tier orchestration (70% coverage target)
- `internal/risk/tier1.go` - Tier 1 triage logic (70% coverage target)
- `internal/risk/tier2.go` - Tier 2 due diligence orchestration (60% coverage target)
- `internal/feedback/` - Feedback tracking (70% coverage target)

**Test Cases:**
- LLM API errors (no key, rate limit, timeout) for both Tier 1 and Tier 2
- Tier 0 heuristic edge cases (mixed changes, edge formatting)
- Tier 1 escalation logic (SAFE vs REVIEW_NEEDED vs ESCALATE)
- Tier 2 context fetching edge cases (no data, missing metrics)
- Cache hit/miss scenarios
- Parallel processing edge cases

### Integration Tests

**Scenarios (see FR-10):**
1. Tier 0 only (TRIVIAL changes)
2. Tier 1 only (SAFE/REVIEW_NEEDED changes)
3. Full escalation (Tier 0 → Tier 1 → Tier 2)
4. Performance benchmarks
5. Cache hit rate validation

**Test Data:**
- Real repositories (commander.js, omnara, terraform-exec)
- Known risky commits (from customer interviews)

### Performance Benchmarks

**Metrics:**
| Metric | Target | How to Measure |
|--------|--------|----------------|
| Tier 0 (p50) | <50ms | Benchmark 100 files with heuristic filter |
| Tier 1 (p50) | <600ms | Benchmark 100 non-trivial files |
| Tier 2 (p95) | <5s | Benchmark 20 ESCALATE files |
| `crisk init` | <10 min | Test with medium repo (omnara) |
| Cache hit rate | >30% | Run `crisk check` twice, measure Tier 1 cache hits |
| Cost per check | <$0.001 | Track LLM costs across mixed workloads |

**Tools:**
- Go benchmarks (`go test -bench`)
- Shell scripts (`time crisk check`)
- Prometheus metrics (future)

---

## Out of Scope (MVP)

**Explicitly NOT building for MVP:**
- ❌ Cloud deployment (Neptune, K8s, Lambda)
- ❌ Settings portal / web UI
- ❌ GitHub OAuth
- ❌ Multi-tenancy / team features
- ❌ Public repository caching
- ❌ Branch delta graphs (main branch only)
- ❌ Advanced metrics (stick to 7 core queries)
- ❌ Perfect test coverage (60% is enough)
- ❌ Test coverage detection from graph (no reliable schema relationship)
- ❌ LLM-based query generation (fixed query library instead)
- ❌ Intent-based query selection (always run all 7 queries)
- ❌ Query cost estimation and sandboxing
- ❌ Schema registry with validation
- ❌ Escalation/triage logic (replaced with Sequential Analysis Chain)
- ❌ Conditional agent execution (all 8 agents always run)
- ❌ Caching of agent results (always fresh analysis)
- ❌ Multi-provider support (OpenAI only for MVP, Gemini/Anthropic deferred to v2)

**Deferred to v2 (after customer validation):**
- Multi-hop graph navigation (see [../01-architecture/agentic_design.md](../01-architecture/agentic_design.md))
- Metric validation database (see [../01-architecture/risk_assessment_methodology.md](../01-architecture/risk_assessment_methodology.md))
- Cloud shared cache (see [../02-operations/public_caching.md](../02-operations/public_caching.md))
- Team sync features (see [../02-operations/team_and_branching.md](../02-operations/team_and_branching.md))

---

## Placeholder: Unclear Requirements

**Items we're uncertain about (document as we learn from customer discovery):**

### 1. Escalation Thresholds
- **Current assumption**: Coupling >10 → HIGH risk
- **Uncertainty**: Is this threshold correct? Should it be configurable?
- **Customer discovery question**: "What coupling level would make you nervous?"

### 2. LLM Provider Preference
- **Current assumption**: Support both OpenAI and Anthropic
- **Uncertainty**: Do users care? Should we just pick one?
- **Customer discovery question**: "Do you prefer OpenAI or Anthropic? Why?"

### 3. Verbosity Default
- **Current assumption**: Standard mode is default
- **Uncertainty**: Do users want quiet mode by default for speed?
- **Customer discovery question**: "How much detail do you want in output?"

### 4. Pre-commit Hook Blocking
- **Current assumption**: HIGH/CRITICAL blocks commit (requires `--no-verify`)
- **Uncertainty**: Is this too aggressive? Should MEDIUM also block?
- **Customer discovery question**: "Would you want commits blocked on MEDIUM risk?"

### 5. Cache TTL
- **Current assumption**: 15-minute cache TTL
- **Uncertainty**: Too short? Too long?
- **Validation approach**: Measure cache hit rate in beta, adjust if <30%

---

## Success Criteria (End of Week 4)

### Functional Requirements ✅
- ✅ LLM client integration working (OpenAI + Anthropic, dual-model support)
- ✅ Tier 0 heuristic filter <50ms
- ✅ Tier 1 fast LLM triage <600ms
- ✅ Tier 2 deep LLM investigation <5s
- ✅ End-to-end `crisk check` flow complete (3-tier escalation)
- ✅ All 4 verbosity levels working (human vs AI mode)
- ✅ False positive tracking implemented
- ✅ Configuration management complete

### Non-Functional Requirements ✅
- ✅ Test coverage >60%
- ✅ Integration tests passing on 3 repos
- ✅ Performance benchmarks meet targets
- ✅ Cache hit rate >30%
- ✅ Documentation sufficient for beta users

### Beta Validation ✅
- ✅ Homebrew installation working
- ✅ 2-3 beta users successfully using crisk
- ✅ False positive rate measured (<15% target)
- ✅ Customer feedback collected

---

## Weekly Milestones

### Week 1: Foundation & Cleanup (Current)
- 🚧 Update mvp_development_plan.md with current status
- 🚧 Delete dead code (tier1.go, tier2.go, simple_investigator.go)
- 🚧 Verify graph schema alignment with simplified_graph_schema.md
- 🚧 Audit internal/ packages for bloat
- ⏳ Create internal/risk/collector.go (standardize 7 queries)
- ⏳ Create internal/risk/heuristic.go (Tier 0 filter)

### Week 2: Core Agent System
- ⏳ Create internal/llm/gemini.go + agent_executor.go
- ⏳ Create 5 specialized agents (incident, blast_radius, cochange, ownership, quality)
- ⏳ Implement parallel execution (Phase 2 in <500ms)
- ⏳ Unit tests for agent system

### Week 3: Chain Completion & Output
- ⏳ Create Agent 6, 7, 8 (patterns, synthesizer, validator)
- ⏳ Create internal/risk/chain_orchestrator.go
- ⏳ Create internal/output/fr6_formatter.go (FR-6 format)
- ⏳ Update cmd/crisk/check.go (full integration)
- ⏳ Integration tests with real repos

### Week 4: Production Auth & Polish
- ⏳ Update cmd/crisk/login.go (device flow OAuth)
- ⏳ Create frontend API endpoints (coderisk-frontend/app/api/cli/)
- ⏳ Create internal/feedback/ (false positive tracking)
- ⏳ Performance optimization (<3.3s target)
- ⏳ Documentation updates
- ⏳ Fresh git repo creation (remove dev_docs from history)

---

## Related Documents

**Product Strategy:**
- [mvp_vision.md](mvp_vision.md) - Product vision
- [competitive_analysis.md](competitive_analysis.md) - Market positioning
- [strategic_decision_framework.md](strategic_decision_framework.md) - 4-week plan

**Architecture:**
- [../01-architecture/agentic_design.md](../01-architecture/agentic_design.md) - Investigation flow (simplified for MVP)
- [../01-architecture/risk_assessment_methodology.md](../01-architecture/risk_assessment_methodology.md) - Risk formulas
- [../01-architecture/prompt_engineering_design.md](../01-architecture/prompt_engineering_design.md) - Prompt design (simplified for MVP)

**Implementation:**
- [../03-implementation/status.md](../03-implementation/status.md) - Current status
- [../03-implementation/NEXT_STEPS.md](../03-implementation/NEXT_STEPS.md) - Technical roadmap

---

**Last Updated:** October 21, 2025
**Next Review:** Weekly - update with progress and blockers
**Completion Target:** Week 4 (November 18, 2025)
