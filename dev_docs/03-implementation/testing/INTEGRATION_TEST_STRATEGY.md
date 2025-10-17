# Integration Test Strategy: Claude Code + CodeRisk

**Purpose:** High-level testing strategy for validating CodeRisk integration with AI coding assistants
**Last Updated:** October 9, 2025
**Audience:** Developers, QA engineers, Product managers
**Status:** Implementation guide for E2E validation

> **Cross-reference:** For technical implementation, see [E2E_TEST_SUMMARY.md](E2E_TEST_SUMMARY.md). For user workflows, see [../00-product/developer_experience.md](../../00-product/developer_experience.md)

---

## The Big Picture

We need to validate that CodeRisk works seamlessly with AI coding assistants (Claude Code, Cursor, GitHub Copilot) to create a **safety net for AI-generated code**. The goal is to test the complete workflow where:

1. **AI generates code** → Claude Code makes edits
2. **CodeRisk evaluates risk** → `crisk check` analyzes the changes
3. **Feedback loop** → If high risk, AI fixes issues and re-checks
4. **Safe commit** → Only commit when risk is LOW

This mimics real-world developer workflows where AI tools help write code quickly, but CodeRisk ensures quality and safety.

---

## Test Objectives

### Primary Goals

1. **Validate Claude Code Interface** - Ensure Claude Code can successfully call `crisk` commands and interpret results
2. **Verify AI Mode Integration** - Test that `--ai-mode` JSON output is consumable by AI assistants for automated fixes
3. **Confirm Risk Evaluation Accuracy** - Validate that CodeRisk correctly identifies high/medium/low risk scenarios

### Success Criteria

- ✅ Claude Code can execute `crisk check` without errors
- ✅ AI Mode JSON is well-formed and contains actionable data
- ✅ Risk levels (HIGH/MEDIUM/LOW) match expected outcomes
- ✅ Feedback loop works: AI can fix issues and re-validate until LOW risk
- ✅ Entire workflow completes without manual intervention

---

## Test Scenario Overview

### Scenario 1: Happy Path - Low Risk Code ✅

**Objective:** Verify that well-written, low-risk code gets approved immediately

**Setup:**
- Simple code change: Add a new utility function with tests
- Expected risk: LOW (good test coverage, low coupling, no incidents)

**Test Flow:**
1. Claude Code creates a new file with a simple utility function
2. Claude Code creates corresponding test file with >80% coverage
3. Run `crisk check <file>`
4. Verify output shows LOW risk
5. Claude Code commits the change

**Expected Outcome:**
- ✅ Risk assessment: LOW
- ✅ Phase 1 completes in <200ms
- ✅ Phase 2 does not trigger (no escalation)
- ✅ Commit proceeds without issues

**Validates:**
- Basic `crisk check` functionality
- Low-risk detection
- Fast performance for simple cases

---

### Scenario 2: High Risk Code - Escalation & Fix Loop 🔴

**Objective:** Verify that high-risk code triggers Phase 2 investigation and can be iteratively improved

**Setup:**
- Complex code change: Modify a critical authentication file
- Expected risk: HIGH (high coupling, low test coverage, past incidents)

**Test Flow:**

#### Iteration 1: Initial HIGH Risk
1. Claude Code makes a change to `auth.py` (authentication logic)
2. Run `crisk check auth.py`
3. Verify Phase 2 triggers (coupling >10, incidents >0)
4. Check `--explain` output shows investigation trace
5. Verify risk level: HIGH with specific reasons

**Expected Phase 1 Results:**
- Coupling: 15 files depend on `auth.py`
- Test coverage: 0.2 (below threshold of 0.3)
- Co-change frequency: 0.8 with `session.py`
- **Result:** ⚠️ HIGH RISK → Escalate to Phase 2

**Expected Phase 2 Investigation:**
- Hop 1: Calculate coupling → 15 files affected
- Hop 2: Check ownership history → Owner changed 10 days ago (new owner)
- Hop 3: Search incidents → Similar change caused bug 2 weeks ago
- **Synthesis:** HIGH risk (confidence: 87%)

**Expected Recommendations:**
```
Critical Actions (must do before commit):
1. Add integration tests for authentication flow
2. Review files that depend on auth.py for breaking changes
3. Consult with previous owner (Alice) about incident #234

High Priority:
1. Increase test coverage to >0.5
2. Add error handling for new authentication paths
```

#### Iteration 2: Claude Code Fixes Issues
4. Claude Code reads the recommendations
5. Claude Code adds integration tests → coverage increases to 0.6
6. Run `crisk check auth.py` again
7. Verify risk level: MEDIUM (tests improved, but coupling still high)

**Expected Results:**
- Coupling: Still 15 files (structural, can't change)
- Test coverage: 0.6 (improved! ✅)
- Incidents: Still flagged (historical data)
- **Result:** ⚠️ MEDIUM RISK

#### Iteration 3: Final Review
8. Claude Code adds documentation and error handling
9. Run `crisk check auth.py` one more time
10. Verify risk level: LOW (test coverage good, documentation added)
11. Claude Code commits the change

**Expected Final Results:**
- Coupling: 15 files (acceptable with good tests)
- Test coverage: 0.6 (above threshold ✅)
- Documentation: Present ✅
- **Result:** ✅ LOW RISK → Safe to commit

**Validates:**
- Phase 2 escalation logic
- LLM investigation trace
- Iterative improvement workflow
- AI can parse recommendations and fix issues

---

### Scenario 3: AI Mode JSON Validation 🤖

**Objective:** Verify that `--ai-mode` produces machine-readable JSON for automated fixes

**Setup:**
- Moderate-risk code change: Refactor a data processing function
- Expected: AI Mode should provide actionable prompts

**Test Flow:**
1. Claude Code modifies `data_processor.py`
2. Run `crisk check --ai-mode data_processor.py > output.json`
3. Validate JSON structure matches specification
4. Check that AI actions are present and actionable

**Expected AI Mode Output Structure:**
```json
{
  "overall_risk": "MEDIUM",
  "files": [...],
  "ai_assistant_actions": [
    {
      "action_type": "add_test",
      "file_path": "data_processor.py",
      "function": "process_batch",
      "confidence": 0.85,
      "ready_to_execute": true,
      "prompt": "Add unit tests for process_batch function. Current coverage: 0.3, target: 0.5. Focus on edge cases: empty input, large batches, invalid data types.",
      "estimated_lines": 25,
      "risk_reduction": 0.4
    },
    {
      "action_type": "refactor",
      "confidence": 0.72,
      "ready_to_execute": false,
      "prompt": "Consider extracting batch validation logic into a separate function to reduce coupling (current: 12 dependencies).",
      "estimated_lines": 15,
      "risk_reduction": 0.2
    }
  ],
  "graph_analysis": {
    "blast_radius": {
      "total_affected_files": 8,
      "direct_dependents": 3,
      "transitive_dependents": 5
    },
    "temporal_coupling": [
      {
        "file_a": "data_processor.py",
        "file_b": "data_validator.py",
        "frequency": 0.85,
        "co_changes": 12
      }
    ],
    "hotspots": [
      {
        "file": "data_processor.py",
        "score": 0.78,
        "reason": "high_churn_low_coverage"
      }
    ]
  },
  "recommendations": {
    "critical": [
      {
        "action": "Add tests for edge cases",
        "auto_fixable": true,
        "estimated_time": 15
      }
    ]
  }
}
```

**Validation Steps:**
1. Parse JSON successfully (well-formed)
2. Check `ai_assistant_actions` array has >0 items
3. Verify `ready_to_execute: true` actions have complete prompts
4. Validate `graph_analysis.blast_radius` is populated
5. Confirm `temporal_coupling` includes co-changed files
6. Check `recommendations.critical` has actionable items

**Claude Code Uses JSON to:**
1. Read the first `ready_to_execute: true` action
2. Execute the prompt (add tests)
3. Re-run `crisk check --ai-mode`
4. Verify `risk_reduction` occurred
5. Continue until `overall_risk: "LOW"`

**Validates:**
- AI Mode JSON schema completeness
- Actionable prompts for automated fixes
- Graph analysis data accuracy
- Silent quality improvement workflow

---

### Scenario 4: Performance & Timeout Testing ⚡

**Objective:** Verify CodeRisk performs within specified time limits

**Setup:**
- Large codebase: 1,000+ files in monorepo
- Various risk levels: LOW, MEDIUM, HIGH

**Test Cases:**

#### Test 4.1: Phase 1 Performance
- **File:** Simple utility change
- **Expected:** Phase 1 completes in <200ms
- **Measured:** Actual time via `time crisk check <file>`
- **Pass criteria:** <200ms for 95% of checks

#### Test 4.2: Phase 2 Performance
- **File:** High-risk core service change
- **Expected:** Phase 2 completes in 3-5 seconds
- **Measured:** Investigation trace timestamps
- **Pass criteria:** <5 seconds for full investigation

#### Test 4.3: Large File Set
- **Files:** 50 files changed in one commit
- **Expected:** Batch processing within 30 seconds
- **Measured:** Total runtime for `crisk check --all`
- **Pass criteria:** <30 seconds for 50 files

#### Test 4.4: Timeout Handling
- **Setup:** Intentionally slow git history (100,000 commits)
- **Expected:** Graceful timeout after 3 minutes
- **Measured:** Timeout logs and error messages
- **Pass criteria:** Useful error message, no crash

**Validates:**
- Performance targets met
- Timeout handling works correctly
- System remains responsive under load

---

## Test Environment Setup

### Prerequisites

1. **CodeRisk Installation:**
   - Build: `make build`
   - Install: `./crisk --version` shows correct version
   - Docker services: Neo4j, PostgreSQL, Redis running

2. **Test Repository: omnara-ai/omnara**
   - **Repository:** https://github.com/omnara-ai/omnara
   - **Type:** Real-world Next.js + TypeScript monorepo
   - **Size:** 421 files (TypeScript: 286, Python: 129, JavaScript: 6)
   - **Structure:** Multi-package monorepo with apps/ and packages/
   - **Git History:** Active development with >50 commits
   - **Why this repo:**
     - Real-world complexity (authentication, API, UI components)
     - Multiple languages and frameworks
     - Good mix of high/medium/low risk files
     - Active development = temporal patterns exist
     - Already used in E2E testing (proven baseline)

3. **Incident Database:**
   - Pre-seeded with sample incidents from omnara development
   - Linked to specific files/commits in the omnara repo
   - Ensures incident-based risk detection works with real scenarios

4. **Environment Variables:**
   ```bash
   export OPENAI_API_KEY="sk-..."  # For Phase 2 LLM investigation
   export CODERISK_DEBUG="true"     # For verbose logging
   ```

### Test Data from omnara Repository

**Low Risk Files (examples):**
- `packages/utils/src/string-helpers.ts` - Utility functions
- `packages/ui/src/components/button.tsx` - Simple UI component
- Has comprehensive tests (coverage >0.8)
- Low coupling (2-3 dependencies)
- No past incidents

**High Risk Files (examples):**
- `apps/web/src/app/api/auth/[...nextauth]/route.ts` - Authentication
- `apps/web/src/lib/db/schema.ts` - Database schema
- High coupling (15+ dependencies)
- Low test coverage (<0.3)
- Critical path (many imports)
- Temporal coupling with related auth/session files

**Medium Risk Files (examples):**
- `apps/web/src/app/api/data/route.ts` - API endpoints
- `packages/core/src/data-processor.ts` - Data processing
- Moderate coupling (6-8 dependencies)
- Medium test coverage (0.4-0.6)
- Some co-change patterns with related files

---

## Test Execution Strategy

### Phase-by-Phase Testing with omnara Repository

Complete step-by-step validation of the entire CodeRisk workflow using the omnara repository.

#### Phase 0: Initial Setup & Environment Validation

**Objective:** Ensure all prerequisites are met before testing

**Steps:**
1. **Clone omnara repository**
   ```bash
   cd /tmp
   git clone https://github.com/omnara-ai/omnara.git
   cd omnara
   ```

2. **Verify Docker services running**
   ```bash
   docker ps | grep -E 'neo4j|postgres|redis'
   # Expected: 3 containers running
   ```

3. **Verify crisk build**
   ```bash
   /path/to/crisk --version
   # Expected: Version number displayed
   ```

4. **Set environment variables**
   ```bash
   export OPENAI_API_KEY="sk-..."
   export CODERISK_DEBUG="true"
   ```

**Pass Criteria:**
- ✅ omnara repo cloned successfully (421 files)
- ✅ All 3 Docker containers healthy
- ✅ crisk binary exists and shows version
- ✅ Environment variables set

---

#### Phase 1: Graph Construction (init-local)

**Objective:** Validate Layer 1, 2, 3 graph construction from omnara repository

**Test: Graph Construction - Layer 1 (Code Structure)**

**Command:**
```bash
cd /tmp/omnara
/path/to/crisk init-local
```

**Expected Output:**
```
✅ Found 421 source files: JavaScript (6), TypeScript (286), Python (129)
✅ Connected to Neo4j
⏳ Parsing source code with Tree-sitter...
✅ Parsed 421 files (0 failed)
✅ Entities extracted: ~5,527 total
  - Files: 421
  - Functions: ~2,563
  - Classes: ~454
  - Imports: ~2,089
⏳ Creating graph structure...
✅ Graph construction complete (~5,106 edges)
⏳ Starting temporal analysis (window: 90 days)...
```

**Validation Queries (Neo4j):**
```cypher
// 1. Verify File nodes
MATCH (f:File) RETURN count(f) as file_count
// Expected: 421

// 2. Verify Function nodes
MATCH (fn:Function) RETURN count(fn) as function_count
// Expected: ~2,560

// 3. Verify Class nodes
MATCH (c:Class) RETURN count(c) as class_count
// Expected: ~454

// 4. Verify Import nodes
MATCH (i:Import) RETURN count(i) as import_count
// Expected: ~2,089

// 5. Verify CONTAINS edges (File -> Function/Class)
MATCH ()-[r:CONTAINS]->() RETURN count(r) as contains_edges
// Expected: >3,000

// 6. Verify IMPORTS edges (File -> File)
MATCH ()-[r:IMPORTS]->() RETURN count(r) as import_edges
// Expected: >2,000
```

**Pass Criteria:**
- ✅ All 421 files parsed successfully
- ✅ Node counts match expected values (±1%)
- ✅ CONTAINS and IMPORTS edges created
- ✅ No parsing errors
- ✅ Total time <60 seconds

---

**Test: Graph Construction - Layer 2 (Temporal Analysis)**

**Expected Output (continuation):**
```
✅ Git history parsed: 50+ commits
✅ Co-changes calculated: ~150 pairs (frequency >= 0.3)
✅ Storing CO_CHANGED edges...
✅ Temporal analysis complete
  - Commits: 50+
  - Developers: 3-5
  - CO_CHANGED edges: ~300 (bidirectional)
```

**Validation Queries:**
```cypher
// 1. Verify CO_CHANGED edges exist
MATCH ()-[r:CO_CHANGED]->() RETURN count(r) as co_change_edges
// Expected: >0 (should be ~300)

// 2. Check specific co-change pattern
MATCH (a:File)-[r:CO_CHANGED]-(b:File)
WHERE r.frequency >= 0.7
RETURN a.path, b.path, r.frequency, r.co_changes
ORDER BY r.frequency DESC
LIMIT 5
// Expected: High-frequency pairs (auth + session files, etc.)

// 3. Verify co-change properties
MATCH ()-[r:CO_CHANGED]->()
WHERE r.frequency IS NOT NULL AND r.co_changes IS NOT NULL
RETURN count(r) as edges_with_properties
// Expected: All edges have frequency and co_changes properties
```

**Pass Criteria:**
- ✅ CO_CHANGED edges created (count >0)
- ✅ Edges have frequency property (0.0-1.0)
- ✅ Edges have co_changes count
- ✅ Temporal analysis completes without timeout (<3 min)
- ✅ High-frequency pairs make logical sense

---

**Test: Graph Construction - Layer 3 (Incident Database)**

**Setup:** Create sample incidents linked to omnara files

**Commands:**
```bash
# Create incident 1: Authentication bug
INCIDENT_1=$(./crisk incident create \
  "NextAuth session timeout" \
  "Users getting logged out unexpectedly after 5 minutes" \
  --severity critical \
  --root-cause "Session expiry misconfiguration in [...nextauth]/route.ts" \
  | grep -oP 'ID: \K[a-f0-9-]+')

# Link to file
./crisk incident link "$INCIDENT_1" \
  "apps/web/src/app/api/auth/[...nextauth]/route.ts" \
  --line 42 \
  --function "authOptions"

# Create incident 2: Database schema issue
INCIDENT_2=$(./crisk incident create \
  "Database migration failure" \
  "Schema update broke production queries" \
  --severity high \
  --root-cause "Missing null check in schema.ts" \
  | grep -oP 'ID: \K[a-f0-9-]+')

# Link to file
./crisk incident link "$INCIDENT_2" \
  "apps/web/src/lib/db/schema.ts" \
  --line 15
```

**Validation Queries:**
```cypher
// 1. Verify Incident nodes
MATCH (i:Incident) RETURN count(i) as incident_count
// Expected: 2

// 2. Verify CAUSED_BY edges
MATCH ()-[r:CAUSED_BY]->() RETURN count(r) as caused_by_edges
// Expected: 2

// 3. Check specific incident linkage
MATCH (i:Incident {id: "$INCIDENT_1"})-[r:CAUSED_BY]->(f:File)
RETURN i.title, f.path, r.confidence, r.line_number
// Expected: 1 row with correct file path and line number

// 4. Verify incident search works
// (via PostgreSQL BM25)
```

**PostgreSQL Validation:**
```bash
./crisk incident search "session"
# Expected: Returns "NextAuth session timeout" incident

./crisk incident search "schema"
# Expected: Returns "Database migration failure" incident
```

**Pass Criteria:**
- ✅ Incidents created in PostgreSQL
- ✅ Incident nodes created in Neo4j
- ✅ CAUSED_BY edges link incidents to files
- ✅ BM25 search finds incidents (<50ms)
- ✅ All incident properties preserved (severity, root cause, etc.)

---

#### Phase 2: Risk Assessment (crisk check)

**Objective:** Validate Phase 1 risk calculation on omnara files

**Test: Low Risk File Detection**

**Command:**
```bash
cd /tmp/omnara
/path/to/crisk check packages/ui/src/components/button.tsx
```

**Expected Output:**
```
🔍 CodeRisk Analysis
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

File: packages/ui/src/components/button.tsx
Risk Level: ✅ LOW

Metrics:
  Coupling: 3 files depend on this (threshold: 10)
  Co-change frequency: 0.2 (threshold: 0.7)
  Test coverage: 0.85 (threshold: 0.3)
  Incidents: 0 linked incidents

Phase 1 completed in 24ms
```

**Pass Criteria:**
- ✅ Risk level: LOW
- ✅ Phase 1 time <200ms
- ✅ Phase 2 does NOT trigger
- ✅ Metrics calculated correctly

---

**Test: High Risk File Detection + Phase 2 Escalation**

**Command:**
```bash
cd /tmp/omnara
export OPENAI_API_KEY="sk-..."
/path/to/crisk check apps/web/src/app/api/auth/[...nextauth]/route.ts
```

**Expected Phase 1 Output:**
```
🔍 CodeRisk Analysis
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

File: apps/web/src/app/api/auth/[...nextauth]/route.ts
Risk Level: ⚠️  HIGH

Metrics:
  Coupling: 18 files depend on this (threshold: 10) ❌
  Co-change frequency: 0.82 with session.ts (threshold: 0.7) ❌
  Test coverage: 0.15 (threshold: 0.3) ❌
  Incidents: 1 linked incident (critical severity) ❌

Phase 1 completed in 31ms

🔍 Escalating to Phase 2 (LLM investigation)...
```

**Expected Phase 2 Output:**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Investigation Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Key Evidence:
1. [structural] High coupling: 18 files depend on this authentication route
2. [temporal] Strong co-change pattern with session.ts (frequency: 0.82)
3. [incidents] Critical incident: "NextAuth session timeout" (14 days ago)
4. [testing] Low test coverage: 15% (target: >30%)

Risk Level: HIGH (confidence: 89%)

Summary: This authentication route is critical infrastructure with
high coupling and a recent incident. The strong temporal coupling with
session.ts suggests these files are tightly bound. Low test coverage
increases risk of regression.

Recommendations:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Critical (must address before commit):
  1. Add integration tests for authentication flow
  2. Review the 18 dependent files for breaking changes
  3. Investigate incident #$INCIDENT_1 to avoid regression

High Priority:
  1. Increase test coverage to >50%
  2. Add error handling for edge cases
  3. Document session timeout configuration

Investigation completed in 4.2s (3 hops, 892 tokens)
```

**Pass Criteria:**
- ✅ Phase 1 detects HIGH risk
- ✅ Phase 2 automatically triggers
- ✅ Investigation trace shows logical hops
- ✅ Recommendations are specific and actionable
- ✅ Total time <5 seconds
- ✅ Confidence score provided

---

**Test: AI Mode JSON Output**

**Command:**
```bash
cd /tmp/omnara
/path/to/crisk check --ai-mode apps/web/src/app/api/auth/[...nextauth]/route.ts > output.json
cat output.json | jq '.'
```

**Expected JSON Structure:**
```json
{
  "overall_risk": "HIGH",
  "timestamp": "2025-10-09T...",
  "repository": "omnara-ai/omnara",
  "branch": "main",
  "files": [
    {
      "path": "apps/web/src/app/api/auth/[...nextauth]/route.ts",
      "risk_level": "HIGH",
      "metrics": {
        "coupling": 18,
        "co_change_frequency": 0.82,
        "test_coverage": 0.15,
        "incident_count": 1
      },
      "issues": [...]
    }
  ],
  "ai_assistant_actions": [
    {
      "action_type": "add_test",
      "file_path": "apps/web/src/app/api/auth/[...nextauth]/route.ts",
      "confidence": 0.89,
      "ready_to_execute": true,
      "prompt": "Add integration tests for NextAuth authentication flow...",
      "estimated_lines": 45,
      "risk_reduction": 0.4
    }
  ],
  "graph_analysis": {
    "blast_radius": {
      "total_affected_files": 18,
      "direct_dependents": 12,
      "transitive_dependents": 6
    },
    "temporal_coupling": [
      {
        "file_a": "apps/web/src/app/api/auth/[...nextauth]/route.ts",
        "file_b": "apps/web/src/lib/auth/session.ts",
        "frequency": 0.82,
        "co_changes": 15
      }
    ],
    "hotspots": [
      {
        "file": "apps/web/src/app/api/auth/[...nextauth]/route.ts",
        "score": 0.89,
        "reason": "high_coupling_incident_history"
      }
    ]
  },
  "investigation_trace": [
    {
      "hop": 1,
      "action": "calculate_coupling",
      "nodes_visited": [...],
      "decision": "High coupling detected, check temporal patterns"
    },
    {
      "hop": 2,
      "action": "get_co_changed_files",
      "decision": "Strong co-change with session.ts, check incidents"
    },
    {
      "hop": 3,
      "action": "search_incidents",
      "decision": "Critical incident found, escalate to HIGH risk"
    }
  ],
  "recommendations": {
    "critical": [
      {
        "action": "Add integration tests for authentication flow",
        "reason": "Low test coverage (15%) + critical incident history",
        "auto_fixable": true,
        "estimated_time": 30
      }
    ]
  }
}
```

**Validation:**
```bash
# 1. JSON is well-formed
cat output.json | jq '.' > /dev/null && echo "✅ Valid JSON"

# 2. Check ai_assistant_actions populated
jq '.ai_assistant_actions | length' output.json
# Expected: >0

# 3. Verify graph_analysis data
jq '.graph_analysis.blast_radius.total_affected_files' output.json
# Expected: 18

# 4. Check investigation trace
jq '.investigation_trace | length' output.json
# Expected: 3 (hops)

# 5. Verify recommendations
jq '.recommendations.critical | length' output.json
# Expected: >0
```

**Pass Criteria:**
- ✅ JSON is well-formed and parseable
- ✅ `ai_assistant_actions` array has actionable prompts
- ✅ `graph_analysis` includes blast radius, temporal coupling, hotspots
- ✅ `investigation_trace` shows hop-by-hop reasoning
- ✅ `recommendations` are categorized (critical/high/medium)
- ✅ All data types match specification

---

#### Phase 3: Iterative Fix Loop (Claude Code Workflow)

**Objective:** Validate that Claude Code can iteratively improve code until LOW risk

**Test: High Risk → Medium Risk → Low Risk**

**Starting State:**
```bash
# Initial check shows HIGH risk
/path/to/crisk check apps/web/src/app/api/auth/[...nextauth]/route.ts
# Output: HIGH risk (coupling: 18, coverage: 0.15, incidents: 1)
```

**Iteration 1: Add Tests**
```bash
# Claude Code action: Create test file
# apps/web/src/app/api/auth/__tests__/auth.test.ts

# Re-check
/path/to/crisk check apps/web/src/app/api/auth/[...nextauth]/route.ts
# Expected: MEDIUM risk (coupling: 18, coverage: 0.65, incidents: 1)
# Improvement: Test coverage increased
```

**Iteration 2: Add Documentation & Error Handling**
```bash
# Claude Code action: Add JSDoc comments and try-catch blocks

# Re-check
/path/to/crisk check apps/web/src/app/api/auth/[...nextauth]/route.ts
# Expected: LOW risk (coupling: 18, coverage: 0.65, incidents: 1)
# Note: Coupling can't change (structural), but coverage + docs = acceptable risk
```

**Pass Criteria:**
- ✅ Each iteration shows measurable improvement
- ✅ Risk level decreases: HIGH → MEDIUM → LOW
- ✅ Metrics change appropriately (coverage increases)
- ✅ Final state is LOW risk and committable
- ✅ No infinite loops or crashes

---

#### Phase 4: Performance Benchmarking

**Test: Phase 1 Performance (<200ms target)**

```bash
# Test 10 low-risk files
for file in $(find packages/ui/src/components -name "*.tsx" | head -10); do
  time /path/to/crisk check "$file" --quiet
done | grep real | awk '{print $2}'
# Expected: All times <200ms
```

**Test: Phase 2 Performance (<5s target)**

```bash
# Test 5 high-risk files
time /path/to/crisk check apps/web/src/app/api/auth/[...nextauth]/route.ts
# Expected: <5 seconds total (including LLM calls)
```

**Test: Batch Processing**

```bash
# Check all TypeScript files in apps/web
time /path/to/crisk check apps/web/src/**/*.ts
# Expected: <30 seconds for ~50 files
```

**Pass Criteria:**
- ✅ 95% of Phase 1 checks <200ms
- ✅ Phase 2 investigations <5 seconds
- ✅ Batch processing scales linearly
- ✅ No memory leaks or crashes

---

### Automated Test Suite

Create shell scripts for each scenario:

**`test/integration/test_claude_code_workflow.sh`**
- Runs Scenario 1 (Happy Path)
- Validates LOW risk detection
- Checks performance

**`test/integration/test_risk_escalation.sh`**
- Runs Scenario 2 (High Risk + Fix Loop)
- Validates Phase 2 triggers correctly
- Simulates iterative fixes

**`test/integration/test_ai_mode.sh`**
- Runs Scenario 3 (AI Mode JSON)
- Validates JSON schema
- Checks actionable prompts

**`test/integration/test_performance.sh`**
- Runs Scenario 4 (Performance)
- Measures timing
- Validates timeout handling

### Manual Testing (Claude Code Session)

**Prompt for Claude Code:**

```
I want you to test the CodeRisk integration workflow. Please:

1. Read test/fixtures/known_risk/high_risk.go
2. Make a small change to the HighRiskFunction (add a new parameter)
3. Run: ./crisk check test/fixtures/known_risk/high_risk.go
4. Show me the output and explain the risk level
5. If risk is HIGH or MEDIUM, read the recommendations
6. Make changes to reduce risk (add tests, refactor, etc.)
7. Re-run crisk check
8. Repeat steps 6-7 until risk is LOW
9. Once LOW risk, commit the changes with: git commit -m "fix: improve HighRiskFunction safety"

Report back at each step with the risk level and any actions you took.
```

**Expected Claude Code Behavior:**
- Executes each step sequentially
- Interprets crisk output correctly
- Makes appropriate code changes based on recommendations
- Iterates until LOW risk achieved
- Commits only when safe

**Success Criteria:**
- ✅ Claude Code completes workflow without errors
- ✅ Code quality improves with each iteration
- ✅ Final commit has LOW risk
- ✅ No manual intervention required

---

## Validation Checklist

### 1. Claude Code Interface ✅
- [ ] Can execute `crisk check <file>` successfully
- [ ] Can parse standard output (risk level, recommendations)
- [ ] Can parse `--explain` output (investigation trace)
- [ ] Can parse `--ai-mode` JSON output
- [ ] Handles errors gracefully (missing API key, file not found)

### 2. AI Mode Integration ✅
- [ ] JSON is well-formed and parseable
- [ ] `ai_assistant_actions` array populated for risky code
- [ ] Prompts are clear and actionable
- [ ] `ready_to_execute` flag accurately indicates automation-safe actions
- [ ] `graph_analysis` contains meaningful data
- [ ] `recommendations` prioritized correctly (critical > high > medium)

### 3. Risk Evaluation Accuracy ✅
- [ ] LOW risk files correctly identified (good tests, low coupling)
- [ ] HIGH risk files correctly identified (poor tests, high coupling, incidents)
- [ ] MEDIUM risk files show nuanced assessment
- [ ] Phase 2 escalation triggers at correct thresholds
- [ ] Investigation trace shows logical reasoning
- [ ] Recommendations are relevant to actual risks

### 4. Feedback Loop ✅
- [ ] AI can make changes based on recommendations
- [ ] Re-running check shows improved risk level
- [ ] Iterative improvements eventually reach LOW risk
- [ ] Workflow completes without infinite loops
- [ ] Each iteration shows measurable improvement

### 5. Performance ✅
- [ ] Phase 1 completes in <200ms (95% of checks)
- [ ] Phase 2 completes in <5 seconds (when needed)
- [ ] Large file sets process in reasonable time
- [ ] Timeout handling prevents runaway processes
- [ ] Cache hit rate >80% for repeated checks

---

## Expected Outcomes

### Developer Experience

**Before CodeRisk:**
- AI generates 500 lines of code
- Developer commits without full review
- Incident occurs in production 2 days later
- 4 hours to debug and fix
- **Cost:** $400 in engineering time + potential downtime

**With CodeRisk:**
- AI generates 500 lines of code
- Run `crisk check` → HIGH risk detected
- Review recommendations, add tests, improve error handling
- Re-check → MEDIUM risk
- Add documentation, refactor complex logic
- Re-check → LOW risk
- Commit with confidence
- **Cost:** 15 minutes upfront, no production incident
- **Savings:** $400 + prevented downtime

### System Confidence

After completing these tests, we can confidently claim:

1. ✅ **CodeRisk integrates seamlessly with AI coding assistants**
2. ✅ **AI Mode enables automated quality improvements**
3. ✅ **Risk evaluation is accurate and actionable**
4. ✅ **Performance meets real-world requirements**
5. ✅ **Safety net prevents risky AI-generated code from reaching production**

---

## Next Steps

1. **Implement Automated Tests** - Create shell scripts for each scenario
2. **Run Manual Claude Code Session** - Validate end-to-end workflow
3. **Measure Performance** - Collect timing data across scenarios
4. **Document Findings** - Create test report with pass/fail results
5. **Iterate on UX** - Improve error messages and recommendations based on testing
6. **Record Demo** - Show successful Claude Code + CodeRisk workflow

---

## References

- **E2E Test Results:** [E2E_TEST_SUMMARY.md](E2E_TEST_SUMMARY.md)
- **Developer Experience:** [../../00-product/developer_experience.md](../../00-product/developer_experience.md)
- **System Architecture:** [../../01-architecture/system_overview_layman.md](../../01-architecture/system_overview_layman.md)
- **Testing Implementation:** [test_ai_mode.sh](../../../test/integration/test_ai_mode.sh)

---

**Last Updated:** October 9, 2025
**Status:** Ready for implementation
**Owner:** QA + Engineering team
