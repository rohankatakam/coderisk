# Phase 2 Investigation Implementation Roadmap

**Created:** October 13, 2025
**Status:** Active Development
**Purpose:** 7-task roadmap to implement proper agentic LLM investigation using all 3 graph layers

---

## Context: Current State Analysis

### What's Working ✅
- **Graph Construction:** All 3 layers fully ingested (421 files, 2,562 functions, 49,654 CO_CHANGED edges, 419 MODIFIES edges)
- **Phase 1 Baseline:** Structural coupling and co-change frequency detection operational
- **Phase 2 Framework:** Investigation loop structure exists but underutilizes graph data

### Critical Gaps Identified ❌

From comprehensive validation testing (see [VALIDATION_TESTING_SUMMARY.md](VALIDATION_TESTING_SUMMARY.md)):

1. **Layer 2 MODIFIES Edges Unused:** 419 MODIFIES edges in graph, but risk assessment never queries them → No churn/hotspot analysis
2. **Layer 1 Structural Data Ignored:** 2,562 functions/454 classes ingested but not analyzed during risk assessment → No detection of new functions/classes
3. **Weak Phase 2 Investigation:** Only queries CO_CHANGED edges (80% threshold), gives generic recommendations
4. **No Multi-File Context:** Files analyzed independently → Misses when user modifies co-changed pairs together
5. **No Commit-Based Workflow:** Cannot analyze `HEAD` or commit references → Inconsistent risk before/after commit

---

## Implementation Strategy

**Philosophy:** Fix data utilization, not data collection. Graph has everything we need—risk assessment just doesn't query it properly.

**Approach:**
- Phase 1: Fix core metric calculations (Tasks 1, 2, 5)
- Phase 2: Improve investigation logic (Tasks 3, 4, 6)
- Phase 3: Add workflow support (Task 7)

**Expected Outcome:**
- All 3 layers utilized in risk assessment
- Specific, actionable recommendations (not generic)
- Multi-file relationship detection
- 70-80% false positive reduction

---

## Task 1: Implement MODIFIES Edge Analysis (Churn/Hotspot Detection)

**Priority:** P0 - Critical
**Estimated Time:** 2-3 hours
**Complexity:** Low (single metric implementation)

### Problem Statement
Phase 2 investigation only queries CO_CHANGED edges. Test results show files with 4 MODIFIES edges (commit history) but output never mentions modification frequency or churn patterns.

### Technical Gap
- **Expected (from design docs):** LLM requests `ownership_churn` metric → System queries MODIFIES edges → Returns commit frequency, primary owner, ownership transitions
- **Actual (from testing):** `modification_types=[]` in all logs, MODIFIES edges never queried

### Implementation Approach

**Files to Create:**
- `internal/metrics/ownership_churn.go` - New Tier 2 metric

**Files to Modify:**
- `internal/agent/investigator.go` - Add ownership_churn to available metrics
- `internal/risk/phase2.go` - Wire metric into investigation loop

**Key Components:**

1. **Metric Calculation Function**
   - Query MODIFIES edges for last 90 days
   - Aggregate by developer email
   - Identify current owner (most commits in last 30 days)
   - Identify previous owner (most commits in days 31-90)
   - Calculate days_since_transition

2. **Cypher Query Pattern**
   ```
   MATCH (f:File {path: $file_path})<-[:MODIFIES]-(c:Commit)-[:AUTHORED]->(d:Developer)
   WHERE c.timestamp > datetime() - duration({days: 90})
   WITH d.email as owner, count(c) as commit_count
   ORDER BY commit_count DESC
   RETURN owner, commit_count
   LIMIT 2
   ```

3. **Risk Thresholds** (from [risk_assessment_methodology.md](../01-architecture/risk_assessment_methodology.md))
   - `modify_count > 10` in 90 days → HIGH churn
   - `5 < modify_count <= 10` → MEDIUM churn
   - `modify_count <= 5` → LOW churn
   - `days_since_transition < 30` → HIGH risk (ownership instability)

4. **Evidence Format**
   ```
   "File modified 12 times in last 90 days (HIGH churn)"
   "Primary owner changed from bob@example.com to alice@example.com 14 days ago"
   ```

5. **LLM Integration**
   - LLM sees: "High coupling (12 files) + HIGH co-change (85%)"
   - LLM decides: "CALCULATE_METRIC: ownership_churn"
   - System executes query, returns evidence
   - LLM synthesizes: "High coupling + ownership transition = elevated risk"

### Success Criteria
- ✅ Test with HeroSection.tsx (4 MODIFIES edges) shows churn metric in output
- ✅ Ownership transitions detected when present
- ✅ Churn patterns incorporated into final risk assessment
- ✅ Recommendations mention specific commit frequency data

### Testing Approach
```bash
# Use file with known MODIFIES edges from validation testing
./crisk check apps/web/src/components/landing/HeroSection.tsx

# Expected output should include:
# "File has 4 commits in last 90 days (moderate churn)"
# "Primary owner: ishaanforthewin@gmail.com"
```

### References
- [agentic_design.md](../01-architecture/agentic_design.md#42-llm-decision-loop-only-if-high-risk) - LLM decision loop
- [risk_assessment_methodology.md](../01-architecture/risk_assessment_methodology.md#31-ownership-churn) - Metric formula
- [graph_ontology.md](../01-architecture/graph_ontology.md) - MODIFIES edge schema

---

## Task 2: Add Structural Change Detection (Layer 1 Utilization)

**Priority:** P0 - Critical
**Estimated Time:** 3-4 hours
**Complexity:** Medium (requires git diff + tree-sitter integration)

### Problem Statement
Adding new functions/classes not detected. All test outputs show `modification_types=[]`. Layer 1 has 2,562 functions and 454 classes but risk assessment ignores structural changes.

### Technical Gap
- **Expected:** Phase 0 pre-analysis detects NEW_FUNCTION, MODIFIED_FUNCTION, NEW_CLASS via tree-sitter diff
- **Actual:** No AST diff analysis during check, only file-level change detection

### Implementation Approach

**Files to Modify:**
- `internal/risk/phase0.go` - Add structural change detection
- `internal/treesitter/diff_analyzer.go` (new) - AST diffing logic
- `cmd/crisk/check.go` - Pass modification types to Phase 1/2

**Key Components:**

1. **Pre-Analysis Flow** (Phase 0)
   - Get staged/working changes via git diff
   - For each changed file:
     - Parse old version (HEAD) with tree-sitter
     - Parse new version (working copy) with tree-sitter
     - Diff ASTs to detect: NEW_FUNCTION, MODIFIED_FUNCTION, DELETED_FUNCTION, NEW_CLASS, MODIFIED_CLASS
   - Return modification_types array

2. **Modification Type Detection**
   - Compare function signatures across old/new ASTs
   - Detect additions: Function in new AST but not old
   - Detect modifications: Function signature/body changed
   - Detect deletions: Function in old AST but not new

3. **Context Propagation**
   - Phase 0 returns: `modification_types: ["NEW_FUNCTION: authenticate_user()"]`
   - Phase 1 sees modification context (no logic change yet)
   - Phase 2 LLM sees: "User added new function authenticate_user() (45 lines)"
   - LLM uses this for specific recommendations

4. **LLM Reasoning Enhancement**
   ```
   Evidence:
   - Modification type: NEW_FUNCTION (authenticate_user, 45 lines)
   - Coupling: 12 files depend on this module
   - Co-change: 85% with user_service.py

   Synthesis:
   "Adding new authentication function to highly coupled module. Recommend:
   1. Add unit tests for authenticate_user()
   2. Add integration tests for auth + user_service interactions (85% co-change)"
   ```

### Success Criteria
- ✅ `modification_types` populated in logs (not empty array)
- ✅ New function detection works for TypeScript, Python, Go
- ✅ Phase 2 output mentions specific structural changes
- ✅ Recommendations tailored to modification type (new code vs refactor)

### Testing Approach
```bash
# Add new function to a file
echo "export function testFunction() { return true; }" >> src/test.ts
git add src/test.ts

# Run check
./crisk check src/test.ts

# Expected output:
# "Added new function testFunction() (1 line)"
# "Recommend: Add tests for new function"
```

### References
- [agentic_design.md](../01-architecture/agentic_design.md#21-three-phase-architecture-updated-oct-2025) - Phase 0 pre-analysis
- [ADR-005](../01-architecture/decisions/005-confidence-driven-investigation.md) - Modification type awareness
- [treesitter package](../../internal/treesitter/) - Existing AST parsing infrastructure

---

## Task 3: Multi-File Context Awareness (Holistic Analysis)

**Priority:** P0 - Critical
**Estimated Time:** 4-5 hours
**Complexity:** High (cross-file analysis logic)

### Problem Statement
When user modifies 2+ files that are co-changed together, system analyzes each independently and misses the relationship. Test showed modifying two 100% co-changed files resulted in separate HIGH risk warnings with no mention they're historically coupled.

### Technical Gap
- **Expected:** "You're changing auth.py and user_service.py which are changed together 85% of the time—this is expected but risky"
- **Actual:** Two separate outputs: "auth.py HIGH risk (coupling)", "user_service.py HIGH risk (coupling)" with no cross-file context

### Implementation Approach

**Files to Create:**
- `internal/risk/multi_file_analyzer.go` - Cross-file relationship detection

**Files to Modify:**
- `cmd/crisk/check.go` - Detect multiple file arguments, call multi-file analyzer
- `internal/output/formatters.go` - Format multi-file context in output

**Key Components:**

1. **Multi-File Detection**
   ```
   Input: crisk check file1.py file2.py file3.py

   Step 1: Run Phase 1 on each file independently
   Step 2: Query CO_CHANGED edges BETWEEN the changed files
   Step 3: If any pair has co_change > 0.7:
     - Flag as "coupled file set"
     - Run holistic Phase 2 investigation
   ```

2. **Cross-File CO_CHANGED Query**
   ```
   MATCH (f1:File {path: $file1})-[r:CO_CHANGED]->(f2:File {path: $file2})
   RETURN r.frequency, r.co_changes
   ```
   - Run for all pairs: (file1, file2), (file1, file3), (file2, file3)
   - Identify strongly coupled pairs (frequency > 0.7)

3. **Holistic Risk Assessment**
   - Individual risks: auth.py (HIGH), user_service.py (HIGH)
   - Cross-file relationship: 85% co-change frequency
   - Combined assessment: "EXPECTED but HIGH risk—you're changing both parts of coupled system"

4. **Enhanced Recommendations**
   ```
   Standard (single file):
   "Add tests for auth.py"

   Multi-file aware:
   "Add integration tests for auth.py + user_service.py interactions (85% co-change pattern)"
   ```

5. **LLM Context Enhancement**
   ```
   LLM Prompt Addition:
   "User is modifying 2 files simultaneously:
   - auth.py (12 dependencies, 75% co-change with user_service.py)
   - user_service.py (8 dependencies, 75% co-change with auth.py)

   These files are historically coupled. Assess risk holistically."
   ```

### Success Criteria
- ✅ Detects when multiple files are co-changed pairs
- ✅ Output mentions cross-file relationship explicitly
- ✅ Recommendations include integration testing for coupled pairs
- ✅ Risk level considers holistic context (not just individual files)

### Testing Approach
```bash
# Use known co-changed pair from validation testing
./crisk check \
  apps/mobile/src/components/chat/index.ts \
  apps/mobile/src/app/_layout.tsx

# Expected output:
# "⚠️ You're modifying 2 files that change together 100% of the time"
# "This is expected but risky—recommend integration tests"
```

### References
- [developer_experience.md](../00-product/developer_experience.md#pattern-1-ai-generation-feedback-loop-manual-developer-workflow) - Multi-file workflow UX
- [agentic_design.md](../01-architecture/agentic_design.md#8-key-design-decisions) - Why multi-file context matters

---

## Task 4: Improve LLM Decision Prompts (Evidence-Based Reasoning)

**Priority:** P1 - High
**Estimated Time:** 2-3 hours
**Complexity:** Medium (prompt engineering)

### Problem Statement
Current LLM prompts produce generic recommendations like "Investigate temporal coupling patterns". Testing showed same recommendation for all HIGH risk files regardless of specific risk factors.

### Technical Gap
- **Expected:** Specific recommendations based on evidence chain (e.g., "Add integration tests for auth + user_service interactions")
- **Actual:** Generic advice that doesn't reference specific files, metrics, or patterns

### Implementation Approach

**Files to Modify:**
- `internal/ai/prompts.go` - Rewrite decision and synthesis prompts
- `internal/agent/investigator.go` - Format evidence chain for LLM

**Key Components:**

1. **Structured Decision Prompt** (from [agentic_design.md](../01-architecture/agentic_design.md#42-llm-decision-loop-only-if-high-risk))
   ```
   You are investigating a code change for risk. Current evidence:

   Tier 1 Metrics (always calculated):
   - Structural coupling: {coupling} files depend on this
   - Co-change frequency: {co_change}
   - Test coverage ratio: {test_ratio}

   {evidence_chain}

   Based on this evidence, decide your next action:
   1. CALCULATE_METRIC: Request ownership or incident data
   2. EXPAND_GRAPH: Load 2-hop neighbors for broader context
   3. FINALIZE: Enough evidence to assess risk

   Respond with JSON: {"action": "...", "reasoning": "...", "target": "..."}
   ```

2. **Enhanced Synthesis Prompt**
   ```
   Synthesize a risk assessment based on all gathered evidence:

   Evidence:
   {evidence_chain}

   Provide:
   1. Risk level: LOW, MEDIUM, HIGH
   2. Confidence: 0.0-1.0
   3. Key factors: List 3-5 evidence points supporting your conclusion
   4. Recommendations: Actionable next steps (be SPECIFIC - reference exact files, metrics, patterns)

   Example of SPECIFIC recommendation:
   ✅ "Add integration tests for auth.py + user_service.py interactions (85% co-change frequency)"
   ❌ "Investigate temporal coupling patterns"

   Respond with JSON.
   ```

3. **Evidence Chain Formatting**
   - Format metrics as natural language before passing to LLM
   - Include specific values, not just abstract signals
   - Example:
     ```
     Evidence:
     1. Structural coupling: 12 files import this module (HIGH)
     2. Temporal coupling: Changes with user_service.py in 85% of commits
     3. Ownership: Primary owner changed from bob@example.com to alice@example.com 14 days ago
     4. Incident history: Similar to INC-453 "Auth timeout cascade failure"
     ```

4. **Recommendation Templates**
   - Provide LLM with patterns for specific recommendations
   - Templates based on evidence combinations:
     - High coupling + new function → "Add tests for {function_name}()"
     - High co-change + multi-file → "Add integration tests for {file1} + {file2}"
     - High churn + incident → "Review {incident_id} before merging"

### Success Criteria
- ✅ Recommendations reference specific files by name
- ✅ Recommendations include specific metrics (e.g., "85% co-change frequency")
- ✅ Each recommendation has clear action (add tests, review coupling, etc.)
- ✅ No generic "investigate X" recommendations without specifics

### Testing Approach
```bash
# Run check on high-coupling file
./crisk check apps/web/src/types/dashboard.ts --explain

# Verify output contains:
# ✅ Specific file names in recommendations
# ✅ Specific metric values (percentages, counts)
# ✅ Actionable next steps (not vague advice)
```

### References
- [prompt_engineering_design.md](../01-architecture/prompt_engineering_design.md) - LLM prompt architecture
- [developer_experience.md](../00-product/developer_experience.md#3-actionable-guidance) - UX principle: tell developers what to DO

---

## Task 5: Verify Test Coverage Ratio Calculation

**Priority:** P0 - Critical
**Estimated Time:** 1-2 hours
**Complexity:** Low (verification + potential fix)

### Problem Statement
Test coverage ratio not mentioned in any validation test outputs. Unclear if metric calculation is broken or if test files aren't being linked during ingestion.

### Technical Gap
- **Expected:** Phase 1 calculates test_ratio, outputs like "Test coverage: 33% (MEDIUM)"
- **Actual:** No test coverage mentioned in any output

### Implementation Approach

**Files to Verify:**
- `internal/metrics/test_ratio.go` - Metric calculation logic
- `internal/ingestion/processor.go` - TESTS relationship creation

**Investigation Steps:**

1. **Query Graph for TESTS Relationships**
   ```cypher
   MATCH (test:File)-[:TESTS]->(source:File)
   RETURN count(*) as test_relationship_count
   ```
   - If count = 0 → Ingestion not creating TESTS relationships
   - If count > 0 → Metric calculation not querying properly

2. **Verify Test File Detection Logic**
   - Check if ingestion identifies test files by naming convention:
     - Python: `test_*.py`, `*_test.py`
     - JavaScript/TypeScript: `*.test.ts`, `*.spec.ts`
     - Go: `*_test.go`
   - Check if test files in `tests/`, `__tests__/`, `spec/` directories detected

3. **Verify Metric Calculation**
   - If TESTS relationships exist, trace why metric isn't calculated:
     - Check Phase 1 calls `calculateTestRatio()`
     - Check test_ratio included in Phase 1 output
     - Check threshold logic: ratio < 0.3 should trigger HIGH signal

4. **Fix Root Cause**
   - Scenario A: TESTS relationships missing → Fix ingestion to create edges
   - Scenario B: Metric not called → Wire into Phase 1 baseline
   - Scenario C: Output not displaying → Fix formatter

### Success Criteria
- ✅ TESTS relationships exist in graph (query returns > 0)
- ✅ Test coverage ratio appears in Phase 1 output
- ✅ Low coverage (< 30%) triggers MEDIUM/HIGH signal
- ✅ Recommendations mention test coverage when relevant

### Testing Approach
```bash
# Query graph to verify TESTS edges
docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  "MATCH (test:File)-[:TESTS]->(source:File) RETURN count(*)"

# If 0, fix ingestion and re-run init-local
# Then test check command
./crisk check apps/web/src/types/dashboard.ts

# Expected output:
# "Test coverage: 0% (HIGH)" or similar
```

### References
- [risk_assessment_methodology.md](../01-architecture/risk_assessment_methodology.md#23-test-coverage-ratio) - Metric formula
- [graph_ontology.md](../01-architecture/graph_ontology.md) - TESTS relationship schema

---

## Task 6: Add Confidence-Based Early Stopping

**Priority:** P1 - High
**Estimated Time:** 2-3 hours
**Complexity:** Medium (LLM confidence scoring)

### Problem Statement
Current investigation always runs 3 hops (fixed limit). Design docs specify confidence-based stopping: if LLM reaches confidence ≥ 0.85, stop early. Wastes time and LLM calls on obvious cases.

### Technical Gap
- **Expected:** LLM stops after 1-2 hops when evidence is clear (e.g., HIGH coupling + HIGH co-change + past incident = obvious HIGH risk)
- **Actual:** Always runs 3 iterations even when first hop provides conclusive evidence

### Implementation Approach

**Files to Modify:**
- `internal/agent/investigator.go` - Investigation loop with confidence checking
- `internal/ai/confidence_scorer.go` (new) - Confidence calculation logic

**Key Components:**

1. **Confidence-Based Loop** (from [ADR-005](../01-architecture/decisions/005-confidence-driven-investigation.md))
   ```
   for iteration := 0; iteration < 5; iteration++ {
       // LLM reviews evidence and assesses confidence
       decision := llm_decide_action(context)

       if decision.confidence >= 0.85 {
           // High confidence, stop early
           logger.Info("Stopping early due to high confidence")
           break
       }

       // Execute next action (CALCULATE_METRIC or EXPAND_GRAPH)
       executeAction(decision)
   }
   ```

2. **Confidence Scoring in LLM Prompt**
   ```
   After reviewing the evidence, assess your confidence in the risk level:

   Confidence: 0.0-1.0
   - 1.0: Extremely confident (multiple independent strong signals)
   - 0.8-0.9: High confidence (2+ strong signals align)
   - 0.5-0.7: Moderate confidence (mixed signals)
   - <0.5: Low confidence (insufficient or conflicting evidence)

   Return: {"confidence": X.XX, "reasoning": "..."}
   ```

3. **Early Stopping Scenarios**
   - **High Confidence Case:** HIGH coupling (15 files) + HIGH co-change (90%) + recent incident (similarity 0.95)
     - Stop after 1 hop with confidence 0.95
   - **Moderate Confidence Case:** MEDIUM coupling (7 files) + MEDIUM co-change (60%) + no incidents
     - Calculate 1 more metric (ownership churn), stop after 2 hops
   - **Low Confidence Case:** Mixed signals, expand to 2-hop neighbors for more context

4. **Breakthrough Tracking**
   - Log when evidence causes risk level to change (LOW → MEDIUM → HIGH)
   - LLM explains what new evidence caused breakthrough
   - Example: "Adding incident similarity raised risk from MEDIUM → HIGH"

### Success Criteria
- ✅ Investigation stops early (1-2 hops) when confidence ≥ 0.85
- ✅ Average hop count reduced from 3.0 to ~2.0
- ✅ LLM calls reduced by ~30% (cost savings)
- ✅ Output explains confidence level and stopping reason

### Testing Approach
```bash
# Test with obvious high-risk file (high coupling + high co-change)
./crisk check apps/web/src/types/dashboard.ts --explain

# Expected output:
# "Investigation completed after 1 hop (confidence: 0.92)"
# "Stopping early: High coupling + high co-change provides conclusive evidence"
```

### References
- [ADR-005](../01-architecture/decisions/005-confidence-driven-investigation.md) - Confidence-driven investigation rationale
- [agentic_design.md](../01-architecture/agentic_design.md#21-three-phase-architecture-updated-oct-2025) - Adaptive investigation loop

---

## Task 7: Add Commit-Based Analysis Support

**Priority:** P2 - Medium
**Estimated Time:** 3-4 hours
**Complexity:** Medium (git commit parsing)

### Problem Statement
Cannot analyze commits directly (`crisk check HEAD` treats HEAD as filename). Test showed risk changes before/after commit (same file showed HIGH → LOW after committing). Need consistent commit-based workflow.

### Technical Gap
- **Expected:** `crisk check HEAD` extracts changed files from commit, analyzes them
- **Actual:** Error: "HEAD: no such file or directory"

### Implementation Approach

**Files to Modify:**
- `cmd/crisk/check.go` - Detect commit references, extract changed files
- `internal/git/commit_parser.go` (new) - Parse commit references

**Key Components:**

1. **Commit Reference Detection**
   ```
   Input: crisk check HEAD
   Input: crisk check abc1234
   Input: crisk check HEAD~1
   Input: crisk check main..feature-branch

   Step 1: Detect if argument is commit reference (not file path)
   Step 2: Parse commit reference to extract changed files
   Step 3: Run risk assessment on extracted files
   ```

2. **Git Commit Parsing**
   ```bash
   # Get changed files in commit
   git diff-tree --no-commit-id --name-only -r HEAD

   # Get changed files in commit range
   git diff --name-only main..feature-branch
   ```

3. **Commit Context Propagation**
   - Include commit metadata in risk assessment context:
     - Commit SHA
     - Commit message
     - Author
     - Timestamp
   - LLM uses commit message for incident similarity search

4. **Multi-File Commit Analysis**
   ```
   Input: crisk check HEAD

   Step 1: Extract changed files (e.g., auth.py, user_service.py)
   Step 2: Run multi-file analysis (Task 3)
   Step 3: Include commit message in incident search (Task 1)
   Step 4: Return holistic commit-level risk assessment
   ```

5. **Consistent Risk Assessment**
   - Fix issue where risk changes after commit:
     - Before commit (staged): auth.py shows HIGH risk
     - After commit: auth.py shows LOW risk
   - Root cause: Graph query uses HEAD, not working copy
   - Solution: Always analyze against consistent graph state

### Success Criteria
- ✅ `crisk check HEAD` works without error
- ✅ Commit references (SHA, HEAD~1, ranges) parsed correctly
- ✅ Multi-file commits analyzed holistically
- ✅ Risk assessment consistent before/after commit
- ✅ Commit message used for incident similarity search

### Testing Approach
```bash
# Make multi-file commit
git add file1.py file2.py
git commit -m "Add authentication feature"

# Analyze commit
./crisk check HEAD

# Expected output:
# "Analyzing commit abc1234: Add authentication feature"
# "Changed files: file1.py, file2.py"
# [Multi-file risk assessment]
```

### References
- [developer_experience.md](../00-product/developer_experience.md#1-pre-commit-automatic-assessment-primary-ux) - Pre-commit workflow
- Git documentation for commit references

---

## Implementation Order & Dependencies

### Phase 1: Core Metrics (Week 1)
**Goal:** Fix fundamental metric calculations

```
Task 5 (Verify Test Coverage)
    ↓
Task 1 (MODIFIES Analysis) ← Independent
    ↓
Task 2 (Structural Detection) ← Depends on Phase 0 framework
```

**Deliverable:** Phase 2 uses all 3 layers (Structure, Temporal, Incidents)

### Phase 2: Investigation Intelligence (Week 2)
**Goal:** Improve LLM decision-making and multi-file awareness

```
Task 4 (Better Prompts) ← Requires Task 1 evidence
    ↓
Task 3 (Multi-File Context) ← Requires improved prompts
    ↓
Task 6 (Confidence Stopping) ← Requires better prompts
```

**Deliverable:** Specific recommendations, early stopping, multi-file detection

### Phase 3: Workflow Support (Week 3)
**Goal:** Add commit-based analysis

```
Task 7 (Commit Analysis) ← Depends on Task 3 (multi-file)
```

**Deliverable:** Full pre-commit workflow support

---

## Success Metrics

### Quantitative Targets
- **False Positive Rate:** 50% → <15% (from validation testing baseline)
- **Average Investigation Hops:** 3.0 → ~2.0 (confidence stopping)
- **LLM Cost per Check:** $0.03-0.05 → $0.02-0.03 (early stopping)
- **Specific Recommendations:** 0% → 80%+ (task 4 improvement)

### Qualitative Goals
- ✅ All 3 layers visible in output
- ✅ Churn/hotspot analysis present
- ✅ Structural changes detected
- ✅ Multi-file relationships explained
- ✅ Recommendations reference specific files, metrics, patterns

### User Experience Validation
Test scenarios that previously failed:
- ❌ **Test 2 (Commit History):** Now shows "4 commits in 90 days" (Task 1)
- ❌ **Test 4 (Co-Changed Pair):** Now detects "changing both files that are 100% coupled" (Task 3)
- ❌ **Test 5 (New Function):** Now shows "Added function testFunction()" (Task 2)
- ❌ **Test 6 (Multi-File Commit):** Now supports `crisk check HEAD` (Task 7)

---

## Integration Testing Plan

### After Task 1 (MODIFIES Analysis)
```bash
# Verify churn detection
./crisk check apps/web/src/components/landing/HeroSection.tsx

# Expected: "File modified 4 times in last 90 days"
```

### After Task 2 (Structural Detection)
```bash
# Add new function
echo "export function test() {}" >> src/test.ts
git add src/test.ts

./crisk check src/test.ts

# Expected: "Added new function test()"
```

### After Task 3 (Multi-File Context)
```bash
# Test co-changed pair
./crisk check \
  apps/mobile/src/components/chat/index.ts \
  apps/mobile/src/app/_layout.tsx

# Expected: "These files change together 100% of the time"
```

### After All Tasks (End-to-End)
```bash
# Commit multi-file change
git add auth.py user_service.py
git commit -m "Add authentication timeout retry logic"

./crisk check HEAD

# Expected:
# - Structural changes detected (new functions)
# - Churn metrics shown
# - Multi-file coupling explained
# - Specific recommendations with file names and metrics
# - Incident similarity to "timeout" keywords
```

---

## References

**Architecture Documents:**
- [agentic_design.md](../01-architecture/agentic_design.md) - LLM investigation framework
- [risk_assessment_methodology.md](../01-architecture/risk_assessment_methodology.md) - Metric formulas
- [graph_ontology.md](../01-architecture/graph_ontology.md) - Graph schema

**Implementation Status:**
- [status.md](status.md) - Current implementation state
- [VALIDATION_TESTING_SUMMARY.md](VALIDATION_TESTING_SUMMARY.md) - Gap analysis findings

**Development Process:**
- [DEVELOPMENT_WORKFLOW.md](../DEVELOPMENT_WORKFLOW.md) - How to implement these tasks

**Design Decisions:**
- [ADR-005](../01-architecture/decisions/005-confidence-driven-investigation.md) - Confidence-based stopping
- [ADR-003](../01-architecture/decisions/003-postgresql-fulltext-search.md) - Incident search approach

---

**Last Updated:** October 13, 2025
**Status:** Ready for implementation
**Estimated Total Time:** 17-24 hours (2-3 weeks parallel)
