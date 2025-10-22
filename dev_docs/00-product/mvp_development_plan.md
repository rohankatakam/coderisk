# MVP Development Plan: Complete Risk Assessment

**Created:** October 21, 2025
**Timeline:** 4 weeks (Week 1-4 of strategic plan)
**Goal:** Working `crisk check` with Phase 1 + Phase 2 risk assessment
**Success Criteria:** <200ms (Phase 1), <5s (Phase 2), <15% FP rate, 2-3 beta users successful

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
- **Temporal Analysis**: Commit history, developer ownership, co-change patterns
- **Incident Tracking**: Issue ingestion, incident-to-file linking, BM25 search
- **Output System**: 4 verbosity levels (quiet, standard, explain, AI mode)

### 🚧 In Progress
- **Risk Assessment**: Phase 1 metrics implemented, Phase 2 LLM integration missing
- **Metrics**: Coupling, co-change, test ratio calculated but not integrated into `crisk check`
- **Test Coverage**: 45% (target: >60% for MVP)

### ❌ Missing (MVP Blockers)
1. LLM client integration (OpenAI/Anthropic)
2. Phase 2 risk assessment (LLM-based high-risk analysis)
3. End-to-end `crisk check` flow (Phase 1 → escalation → Phase 2 → output)
4. Performance optimization (<200ms, <5s targets)
5. False positive tracking and feedback mechanism
6. Integration testing with real repositories

---

## Functional Requirements (No Gaps)

### FR-1: LLM Client Integration

**Requirement:** Support OpenAI and Anthropic APIs via user-provided API keys (BYOK model)

**Acceptance Criteria:**
- User can configure API key via environment variable (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`)
- User can configure API key via config file (`~/.coderisk/config.yaml`)
- Client auto-detects which provider based on available API key (prefer Anthropic if both present)
- Error handling: No API key → graceful degradation (Phase 1 only, warn user)
- Error handling: Rate limit → retry with exponential backoff (3 attempts)
- Error handling: Timeout → fail gracefully, log error
- Token usage logging for cost transparency

**Implementation Notes:**
- Use official SDKs: `github.com/openai/openai-go` and `github.com/anthropics/anthropic-sdk-go`
- Single prompt template for both providers (see [Prompt Template](#prompt-template) below)
- Max tokens: 1000 (cost control)
- Temperature: 0.2 (deterministic risk assessment)

**Files to Create:**
- `internal/llm/client.go` - Interface definition
- `internal/llm/openai.go` - OpenAI implementation
- `internal/llm/anthropic.go` - Anthropic implementation
- `internal/llm/client_test.go` - Unit tests

**Reference:** [../01-architecture/prompt_engineering_design.md](../01-architecture/prompt_engineering_design.md) (future vision, simplified for MVP)

---

### FR-2: Phase 1 Risk Assessment (Baseline Metrics)

**Requirement:** Fast heuristic-based risk assessment using graph data (no LLM)

**Metrics (5 Core):**
1. **Coupling**: Number of incoming dependencies (imports, calls)
   - Formula: `coupling_score = min(incoming_edges / 10, 1.0)`
   - Threshold: >10 incoming edges = HIGH risk

2. **Co-change Frequency**: How often file changes with other files
   - Formula: `co_change_score = max(co_change_rate across all pairs)`
   - Threshold: >0.7 co-change rate = HIGH risk

3. **Test Ratio**: Ratio of test files to implementation files
   - Formula: `test_ratio = test_files / (implementation_files + 1)`
   - Threshold: <0.3 test ratio = MEDIUM risk, <0.1 = HIGH risk

4. **Churn**: Frequency of changes to this file
   - Formula: `churn_score = min(commit_count_last_30_days / 20, 1.0)`
   - Threshold: >20 commits in 30 days = MEDIUM risk

5. **Incident History**: Past incidents linked to this file
   - Formula: `incident_score = min(incident_count / 3, 1.0)`
   - Threshold: >3 incidents = HIGH risk, >1 = MEDIUM risk

**Risk Level Calculation:**
```
HIGH risk: coupling >10 OR co_change >0.7 OR incidents >3
MEDIUM risk: test_ratio <0.3 OR churn >20 OR incidents >1
LOW risk: All others
```

**Acceptance Criteria:**
- Phase 1 completes in <200ms (p50 latency)
- Returns risk level (LOW/MEDIUM/HIGH) for each changed file
- Returns evidence: which metrics triggered risk level
- Cache results for 15 minutes (filesystem cache)

**Files to Update:**
- `internal/risk/calculator.go` - Already exists, validate formulas match above
- `internal/metrics/*.go` - Validate existing implementations
- `cmd/crisk/check.go` - Integrate Phase 1 into `crisk check` command

**Reference:** [../01-architecture/risk_assessment_methodology.md](../01-architecture/risk_assessment_methodology.md)

---

### FR-3: Phase 2 Risk Assessment (LLM-Guided Due Diligence)

**Requirement:** LLM-based due diligence analysis for HIGH-risk files only (escalation from Phase 1)

**Core Purpose:** Answer the developer's due diligence questions using LLM reasoning over graph context

**Escalation Logic:**
- If Phase 1 returns HIGH risk → Trigger Phase 2
- If Phase 1 returns MEDIUM/LOW → Skip Phase 2 (fast path)

**Due Diligence Context Fetching:**

1. **Ownership Context:**
   - Primary owner (developer with most commits)
   - Last modifier (who touched it recently?)
   - Recent modifications (what changed and why?)
   - Commit frequency (is this file actively maintained?)

2. **Blast Radius Analysis:**
   - Dependent files (what will break if this changes?)
   - High-impact dependencies (critical systems affected)
   - Dependency depth (how far does impact propagate?)

3. **Co-change Pattern Detection:**
   - Files that frequently change together
   - Co-change rate (how often they change together)
   - Last time they changed together
   - Forgotten update risk (did developer miss related files?)

4. **Incident History:**
   - Past incidents linked to this file
   - Recent incidents (last 90 days)
   - Incident patterns (what failed before?)
   - Similar past changes that caused failures

5. **Test Coverage:**
   - Current test ratio
   - Test files associated with this file
   - Test gaps (untested critical functions)

**LLM Analysis (Due Diligence Questions):**

Send context to LLM with prompt that asks:
1. **Coordination Question:** Should the developer coordinate with the file owner or recent contributors?
2. **Blast Radius Question:** What dependent files might break? Should they be checked?
3. **Forgotten Updates Question:** Based on co-change patterns, what files likely need updating too?
4. **Incident Prevention Question:** Is this similar to a past incident? What should be done differently?
5. **Test Coverage Question:** Is test coverage adequate for this change?

**LLM Response Parsing:**
- Risk level: LOW/MEDIUM/HIGH/CRITICAL
- Due diligence summary (2-3 sentences)
- Coordination needed: Who to ping and why
- Forgotten updates: Files likely missed
- Incident risk: Similar patterns and prevention
- Recommendations: Prioritized action items

**Acceptance Criteria:**
- Phase 2 completes in <5s (p95 latency including LLM call)
- Only triggers for HIGH-risk files from Phase 1
- Returns structured due diligence assessment (ownership, blast radius, coordination, incidents)
- **Output framed as actionable due diligence** (not abstract risk metrics)
- Graceful degradation if LLM fails (fall back to Phase 1 + basic ownership info)
- Token usage logged for cost transparency

**Files to Create:**
- `internal/agent/investigator.go` - Phase 2 orchestration (due diligence questions)
- `internal/agent/context.go` - Graph context fetching (ownership, blast radius, co-change, incidents)
- `internal/agent/prompt.go` - Due diligence prompt template management
- `internal/agent/due_diligence.go` - Structured response parsing

**Reference:** [../01-architecture/agentic_design.md](../01-architecture/agentic_design.md) (simplified - no multi-hop navigation for MVP)

---

### FR-4: End-to-End `crisk check` Flow

**Requirement:** Complete workflow from git diff to risk report

**Flow:**
```
1. User runs: crisk check [file_path]
   ├─ If no file_path: Analyze all changed files (git diff)
   ├─ If file_path provided: Analyze specific file

2. Phase 1 (Baseline Metrics)
   ├─ For each changed file:
   │  ├─ Calculate 5 core metrics (coupling, co-change, test ratio, churn, incidents)
   │  ├─ Determine risk level (LOW/MEDIUM/HIGH)
   │  └─ Cache result (15-min TTL)

3. Escalation Decision
   ├─ If ALL files LOW/MEDIUM → Return Phase 1 results (fast path)
   ├─ If ANY file HIGH → Proceed to Phase 2 for HIGH files only

4. Phase 2 (LLM Analysis) - HIGH-risk files only
   ├─ Fetch graph context (dependencies, co-changes, incidents)
   ├─ Call LLM with context + prompt
   ├─ Parse LLM response (risk level, reasoning, recommendations)
   └─ If LLM fails → Fall back to Phase 1 result

5. Output Generation (see FR-5)
   ├─ Aggregate results across all files
   ├─ Determine overall risk (highest risk level)
   ├─ Format output based on verbosity level
   └─ Exit with appropriate code (0=LOW, 1=MEDIUM, 2=HIGH, 3=CRITICAL)
```

**Acceptance Criteria:**
- `crisk check` works with no arguments (analyzes git diff)
- `crisk check path/to/file.py` works for specific file
- Phase 1 always runs (<200ms)
- Phase 2 only runs for HIGH-risk files (<5s total)
- Exit codes match risk levels (for pre-commit hook integration)
- Progress indicators shown during analysis (Phase 1 → Phase 2)

**Files to Update:**
- `cmd/crisk/check.go` - Main orchestration logic

---

### FR-5: Output Formatting (4 Verbosity Levels)

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

### FR-6: Performance Optimization

**Requirement:** Meet latency targets for MVP

**Targets:**
- Phase 1 (p50): <200ms
- Phase 2 (p95): <5s
- Cache hit rate: >30%

**Optimizations:**

1. **Neo4j Query Optimization**
   - Add indexes on frequently queried properties:
     - `File.path`
     - `Commit.sha`
     - `Function.name`
   - Use `LIMIT` clauses in queries
   - Batch queries where possible

2. **Filesystem Caching**
   - Cache Phase 1 results: 15-min TTL
   - Cache key: `sha256(file_path + git_sha)`
   - Cache location: `.coderisk/cache/phase1/`
   - Max cache size: 100MB (LRU eviction)

3. **LLM Call Optimization**
   - Context pruning: Max 10 dependencies, 5 co-change partners
   - Max tokens: 1000 (input + output)
   - Timeout: 10s (fail gracefully if exceeded)

4. **Parallel Processing**
   - Analyze multiple files in parallel (Phase 1)
   - Max parallelism: 4 workers (avoid Neo4j overload)

**Acceptance Criteria:**
- Benchmarks show p50 <200ms (Phase 1)
- Benchmarks show p95 <5s (Phase 2)
- Cache hit rate >30% on repeated `crisk check`

**Files to Update:**
- `internal/graph/*.go` - Add query optimization
- `internal/cache/*.go` - Implement Phase 1 caching
- `cmd/crisk/check.go` - Add parallel processing

---

### FR-7: False Positive Tracking

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

### FR-8: Configuration Management

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
  provider: "anthropic"  # "openai" or "anthropic"
  api_key: "sk-ant-..."  # Or use env var ANTHROPIC_API_KEY
  max_tokens: 1000
  timeout_seconds: 10

thresholds:
  coupling: 10
  co_change: 0.7
  test_ratio_medium: 0.3
  test_ratio_high: 0.1
  churn_days: 30
  churn_count: 20
  incidents_medium: 1
  incidents_high: 3

output:
  default_verbosity: "standard"  # "quiet", "standard", "explain", "ai"
  show_progress: true

cache:
  enabled: true
  ttl_minutes: 15
  max_size_mb: 100
```

**Environment Variables:**
- `OPENAI_API_KEY` - OpenAI API key
- `ANTHROPIC_API_KEY` - Anthropic API key
- `CODERISK_VERBOSITY` - Default verbosity level
- `CODERISK_CONFIG` - Path to config file

**Acceptance Criteria:**
- User can configure via file, env vars, or flags
- Priority order respected
- `crisk config get llm.provider` shows current value
- `crisk config set llm.provider openai` updates config file
- Validation: Invalid values rejected with helpful error

**Files to Update:**
- `internal/config/*.go` - Already exists, add LLM/threshold config
- `cmd/crisk/config.go` - Already exists, add get/set commands

---

### FR-9: Integration Testing

**Requirement:** Validate end-to-end flow with real repositories

**Test Repositories:**
1. **Small**: `commander.js` (~50 files, 5K LOC)
2. **Medium**: `omnara` (~400 files, 50K LOC)
3. **Large**: `hashicorp/terraform-exec` (~200 files, 30K LOC)

**Test Scenarios:**

#### Scenario 1: Phase 1 Only (LOW/MEDIUM risk)
```bash
cd /tmp/commander.js
crisk init
git checkout -b test-change
# Make low-risk change (add comment)
crisk check
# Expected: Phase 1 only, <200ms, LOW risk
```

#### Scenario 2: Phase 1 → Phase 2 Escalation (HIGH risk)
```bash
cd /tmp/omnara
crisk init
# Make high-risk change (modify core file with high coupling)
crisk check
# Expected: Phase 1 → Phase 2, <5s, HIGH risk, LLM recommendations
```

#### Scenario 3: Performance Benchmark
```bash
cd /tmp/terraform-exec
crisk init
# Measure init time (target: <10 min)
crisk check
# Measure check time (target: <200ms Phase 1)
```

#### Scenario 4: Cache Hit Rate
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

### FR-10: Beta-Ready Release

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

## Prompt Template

**Phase 2 LLM Prompt (Due Diligence Focus):**

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

5. TEST COVERAGE
   - Current coverage: {test_ratio}
   - Test files: {test_files}

GIT DIFF:
{git_diff}

PHASE 1 RISK ASSESSMENT: {phase1_risk_level}
EVIDENCE: {phase1_metrics}

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

**Token Budget:** Max 1500 tokens (input + output) - increased to include richer context

**Fallback:** If LLM fails, return Phase 1 result + basic ownership/dependency info

---

## Testing Strategy

### Unit Tests (Target: >60% Coverage)

**Priority Packages:**
- `internal/llm/` - LLM client integration (80% coverage target)
- `internal/risk/` - Risk calculation (70% coverage target)
- `internal/agent/` - Phase 2 orchestration (60% coverage target)
- `internal/feedback/` - Feedback tracking (70% coverage target)

**Test Cases:**
- LLM API errors (no key, rate limit, timeout)
- Risk calculation edge cases (no data, missing metrics)
- Cache hit/miss scenarios
- Parallel processing edge cases

### Integration Tests

**Scenarios (see FR-9):**
1. Phase 1 only (LOW/MEDIUM risk)
2. Phase 1 → Phase 2 escalation (HIGH risk)
3. Performance benchmarks
4. Cache hit rate validation

**Test Data:**
- Real repositories (commander.js, omnara, terraform-exec)
- Known risky commits (from customer interviews)

### Performance Benchmarks

**Metrics:**
| Metric | Target | How to Measure |
|--------|--------|----------------|
| Phase 1 (p50) | <200ms | Benchmark 100 files |
| Phase 2 (p95) | <5s | Benchmark 20 HIGH-risk files |
| `crisk init` | <10 min | Test with medium repo (omnara) |
| Cache hit rate | >30% | Run `crisk check` twice, measure hits |

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
- ❌ Complex agent orchestration (just one LLM call)
- ❌ Advanced metrics (stick to 5 core)
- ❌ Perfect test coverage (60% is enough)

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
- ✅ LLM client integration working (OpenAI + Anthropic)
- ✅ Phase 1 risk assessment <200ms
- ✅ Phase 2 risk assessment <5s
- ✅ End-to-end `crisk check` flow complete
- ✅ All 4 verbosity levels working
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

### Week 1: LLM Integration + Phase 2
- ✅ OpenAI client implemented
- ✅ Anthropic client implemented
- ✅ Phase 2 orchestration complete
- ✅ Prompt template finalized
- ✅ Error handling (no key, rate limits, timeouts)

### Week 2: Integration Testing + Performance
- ✅ Integration tests passing (3 repos)
- ✅ Performance targets met (<200ms, <5s)
- ✅ Cache hit rate >30%
- ✅ Neo4j query optimization

### Week 3: False Positive Tracking + Hardening
- ✅ Feedback command implemented
- ✅ Stats calculation working
- ✅ Edge case handling complete
- ✅ Error messages improved

### Week 4: Beta-Ready Release
- ✅ Test coverage >60%
- ✅ Homebrew formula working
- ✅ Documentation complete
- ✅ 2-3 beta users onboarded

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
