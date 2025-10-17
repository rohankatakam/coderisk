# Phase 2 Investigation Implementation - Claude Code Session Prompt

**Purpose:** Complete prompt for implementing proper agentic LLM investigation with full 3-layer graph utilization
**Session Type:** Implementation (not research)
**Estimated Duration:** 2-3 weeks (7 tasks, 17-24 hours total)

---

## Session Objective

Implement proper Phase 2 LLM investigation that utilizes all 3 graph layers (Structure, Temporal, Incidents) to provide specific, actionable risk assessments with multi-file context awareness.

**Current State:** Graph construction complete (421 files, 2,562 functions, 49,654 CO_CHANGED edges) but risk assessment only uses CO_CHANGED edges → 50% false positive rate

**Target State:** All 3 layers utilized, multi-file context awareness, specific recommendations, <15% false positive rate

---

## Context Documents to Read

### **MUST READ (Essential Context - Read First)**

1. **[coderisk-go/dev_docs/03-implementation/PHASE2_INVESTIGATION_ROADMAP.md](../PHASE2_INVESTIGATION_ROADMAP.md)**
   - Complete implementation roadmap with 7 tasks
   - Technical approach for each task
   - Success criteria and testing approach
   - **READ THIS FIRST** - Contains all implementation details

2. **[coderisk-go/dev_docs/03-implementation/VALIDATION_TESTING_SUMMARY.md](VALIDATION_TESTING_SUMMARY.md)**
   - Comprehensive testing findings (6 validation tests)
   - Critical gaps identified with evidence
   - Data layer utilization analysis
   - Understand what's broken and why

3. **[coderisk-go/dev_docs/DEVELOPMENT_WORKFLOW.md](../DEVELOPMENT_WORKFLOW.md)**
   - Development process and conventions
   - How to structure commits
   - Testing requirements
   - Code review guidelines

### **Architecture & Design (Reference as Needed)**

4. **[coderisk-go/dev_docs/01-architecture/agentic_design.md](../01-architecture/agentic_design.md)**
   - LLM investigation framework (Phase 0, Phase 1, Phase 2)
   - Evidence-based reasoning approach
   - Confidence-driven investigation loop
   - **Sections to focus on:**
     - Section 2.1: Three-Phase Architecture
     - Section 4.2: LLM Decision Loop
     - Section 8: Key Design Decisions

5. **[coderisk-go/dev_docs/01-architecture/risk_assessment_methodology.md](../01-architecture/risk_assessment_methodology.md)**
   - Metric formulas and thresholds
   - Tier 1 (always calculated) vs Tier 2 (on-demand) metrics
   - Cypher query patterns for each metric
   - **Sections to focus on:**
     - Section 2: Tier 1 Metrics (coupling, co-change, test_ratio)
     - Section 3: Tier 2 Metrics (ownership_churn, incident_similarity)
     - Section 6.2: Cypher Query Patterns

6. **[coderisk-go/dev_docs/01-architecture/system_overview_layman.md](../01-architecture/system_overview_layman.md)**
   - High-level system explanation
   - Real example walkthrough (payment_processor.py scenario)
   - Why agentic search is efficient
   - **Use this for conceptual understanding**

### **Product Requirements (Understand User Needs)**

7. **[coderisk-go/dev_docs/00-product/developer_experience.md](../00-product/developer_experience.md)**
   - User workflows and pain points
   - Expected output formats
   - What makes recommendations "actionable"
   - **Focus on:** Section 3 (Actionable Guidance), Section 5 (Workflow Integration)

8. **[coderisk-go/dev_docs/00-product/user_personas.md](../00-product/user_personas.md)**
   - Ben the Developer (primary user)
   - What developers need from risk assessment
   - Success criteria from user perspective
   - **Focus on:** Ben's pain points and value proposition sections

### **Implementation Status (Current State)**

9. **[coderisk-go/dev_docs/03-implementation/status.md](status.md)**
   - Current implementation state
   - Component status (what's done, what's not)
   - Dependencies and testing status

---

## Implementation Approach

### **Step 1: Understand Current Implementation** (30 minutes)

**Files to Read:**
- `cmd/crisk/check.go` - Entry point for risk assessment
- `internal/risk/phase0.go` - Pre-analysis (currently minimal)
- `internal/risk/phase1.go` - Baseline assessment (coupling, co-change, test_ratio)
- `internal/risk/phase2.go` - LLM investigation (currently incomplete)
- `internal/agent/investigator.go` - Investigation orchestration
- `internal/metrics/` - Metric calculation implementations

**What to Look For:**
- How Phase 1 calls metric calculations
- How Phase 2 queries graph (what edges are used/ignored)
- Where LLM prompts are defined
- What evidence is passed to LLM

### **Step 2: Implement P0 Tasks First** (Week 1: 8-12 hours)

**Priority Order:**

**Task 5: Verify Test Coverage** (1-2 hours - Start here for quick win)
- Query graph to check if TESTS relationships exist
- If missing, fix ingestion; if present, wire into Phase 1
- **Goal:** Test coverage appears in output

**Task 1: MODIFIES Edge Analysis** (2-3 hours)
- Create `internal/metrics/ownership_churn.go`
- Implement Cypher query for MODIFIES edges
- Wire into Phase 2 investigation when LLM requests
- **Goal:** "File modified 4 times in 90 days" appears in output

**Task 2: Structural Change Detection** (3-4 hours)
- Create `internal/treesitter/diff_analyzer.go` for AST diffing
- Modify `internal/risk/phase0.go` to detect NEW_FUNCTION, MODIFIED_FUNCTION, etc.
- Pass modification_types to Phase 1/2
- **Goal:** "Added new function authenticate_user()" appears in output

**Task 3: Multi-File Context** (4-5 hours)
- Create `internal/risk/multi_file_analyzer.go`
- Detect when multiple files provided, query CO_CHANGED between them
- Provide holistic assessment instead of independent file analysis
- **Goal:** "You're changing files that change together 85%" appears in output

**Week 1 Deliverable:** All 3 layers utilized in risk assessment

### **Step 3: Improve Investigation Intelligence** (Week 2: 6-8 hours)

**Task 4: Better LLM Prompts** (2-3 hours)
- Rewrite prompts in `internal/ai/prompts.go` per agentic_design.md templates
- Add specificity requirements and recommendation templates
- Format evidence as natural language with specific values
- **Goal:** Recommendations reference specific files, metrics, patterns

**Task 6: Confidence-Based Stopping** (2-3 hours)
- Add confidence scoring to investigation loop
- Stop early when confidence ≥ 0.85
- Track breakthrough points (when risk level changes)
- **Goal:** Average hops reduced from 3.0 → ~2.0

**Week 2 Deliverable:** Specific recommendations, early stopping, multi-file detection

### **Step 4: Add Workflow Support** (Week 3: 3-4 hours)

**Task 7: Commit-Based Analysis** (3-4 hours)
- Add commit reference detection in `cmd/crisk/check.go`
- Parse commit references (HEAD, SHA, ranges) to extract files
- Ensure consistent graph state queries
- **Goal:** `crisk check HEAD` works correctly

**Week 3 Deliverable:** Full pre-commit workflow support

---

## Testing Strategy

### **After Each Task:**

Run specific validation test that task should fix:

```bash
# Task 1 (MODIFIES): Test 2 - Commit History
./crisk check apps/web/src/components/landing/HeroSection.tsx
# Verify: "File modified 4 times in last 90 days"

# Task 2 (Structural): Test 5 - New Function
echo "export function test() {}" >> src/test.ts
git add src/test.ts
./crisk check src/test.ts
# Verify: "Added new function test()"

# Task 3 (Multi-File): Test 4 - Co-Changed Pair
./crisk check \
  apps/mobile/src/components/chat/index.ts \
  apps/mobile/src/app/_layout.tsx
# Verify: "Files change together 100% of the time"

# Task 7 (Commit): Test 6 - Multi-File Commit
git add file1.py file2.py
git commit -m "Test commit"
./crisk check HEAD
# Verify: No errors, holistic assessment
```

### **End-to-End Validation:**

After all tasks complete, re-run full validation test suite:

```bash
# Should now pass 5/6 tests (vs 1/6 before)
# See VALIDATION_TESTING_SUMMARY.md for test details
```

---

## Development Guidelines

### **Follow DEVELOPMENT_WORKFLOW.md:**

1. **Branch Naming:** `feature/task-N-short-description`
   - Example: `feature/task-1-modifies-edge-analysis`

2. **Commit Messages:** Use conventional commits
   - `feat: Add ownership_churn metric calculation`
   - `fix: Query MODIFIES edges in Phase 2`
   - `test: Add churn detection validation test`

3. **Testing Requirements:**
   - Unit tests for new metric calculations
   - Integration tests for graph queries
   - End-to-end test for full workflow

4. **Code Review Checklist:**
   - Does it fix the validation test gap?
   - Are Cypher queries optimized (<50ms)?
   - Are error cases handled gracefully?
   - Is evidence formatted for LLM clarity?

### **Implementation Patterns:**

**Metric Calculation Pattern:**
```go
// internal/metrics/ownership_churn.go
func CalculateOwnershipChurn(ctx context.Context, filePath string, graphClient graph.Client) (*MetricResult, error) {
    // 1. Query MODIFIES edges for last 90 days
    // 2. Aggregate by developer
    // 3. Identify current/previous owner
    // 4. Calculate days_since_transition
    // 5. Return MetricResult with evidence string
}
```

**LLM Prompt Enhancement Pattern:**
```go
// internal/ai/prompts.go
func BuildDecisionPrompt(evidence []EvidenceItem) string {
    return fmt.Sprintf(`
You are investigating a code change for risk. Current evidence:

Tier 1 Metrics:
- Structural coupling: %d files depend on this
- Co-change frequency: %.0f%%
- Test coverage: %.0f%%

%s

Based on this evidence, decide your next action:
1. CALCULATE_METRIC: Request ownership or incident data
2. EXPAND_GRAPH: Load 2-hop neighbors
3. FINALIZE: Enough evidence to assess risk

Respond with JSON: {"action": "...", "reasoning": "...", "confidence": X.XX}
`, coupling, coChangeFreq*100, testCoverage*100, formatEvidenceChain(evidence))
}
```

**Multi-File Analysis Pattern:**
```go
// internal/risk/multi_file_analyzer.go
func AnalyzeMultipleFiles(ctx context.Context, files []string) (*RiskAssessment, error) {
    // 1. Run Phase 1 on each file independently
    // 2. Query CO_CHANGED edges BETWEEN the files
    // 3. If any pair has frequency > 0.7, flag as coupled set
    // 4. Run holistic Phase 2 with cross-file context
    // 5. Return combined assessment
}
```

---

## Success Criteria

### **Technical Validation:**

- ✅ All 3 layers (Structure, Temporal, Incidents) mentioned in output
- ✅ MODIFIES edges queried (churn/ownership data shown)
- ✅ Structural changes detected (new functions/classes)
- ✅ Multi-file relationships explained
- ✅ Recommendations include specific file names and metrics

### **Test Results:**

- ✅ Test pass rate: 1/6 → 5/6 (83% pass rate)
- ✅ False positive rate: 50% → <15%
- ✅ Specific recommendations: 0% → 80%+
- ✅ Average investigation hops: 3.0 → ~2.0

### **User Experience:**

Run these commands and verify outputs match expectations:

```bash
# High coupling file - Should show churn + co-changes + specific recommendations
./crisk check apps/web/src/types/dashboard.ts --explain

# Expected output should include:
# ✅ "260 co-changed relationships"
# ✅ "File modified X times in last 90 days"
# ✅ Specific recommendation: "Add integration tests for dashboard.ts + InstanceDetail.tsx"

# Multi-file commit - Should detect relationship
git add auth.py user_service.py
git commit -m "Add authentication timeout handling"
./crisk check HEAD

# Expected output should include:
# ✅ "Analyzing commit abc1234: Add authentication timeout handling"
# ✅ "auth.py and user_service.py change together 85% of the time"
# ✅ "Add integration tests for auth + user_service interactions"
```

---

## Common Pitfalls to Avoid

### **1. Don't Modify Ingestion (Graph Construction is Correct)**
- Problem is in risk assessment, NOT data collection
- Graph has all the data we need, just not being queried properly
- Focus on `internal/risk/` and `internal/metrics/`, not `internal/ingestion/`

### **2. Don't Break Existing CO_CHANGED Detection**
- CO_CHANGED edge detection works correctly (Test 1 passed)
- Add MODIFIES analysis alongside it, don't replace it
- Both temporal signals should coexist

### **3. Ensure Cypher Queries Are Optimized**
- Target: <50ms per query
- Use indexed properties (file_path, timestamp)
- Limit result sets (LIMIT 10 for co-changed files)

### **4. Format Evidence for LLM Clarity**
- Bad: `{"coupling": 12}`
- Good: `"File is imported by 12 other modules (HIGH coupling)"`
- LLM needs natural language with context, not raw numbers

### **5. Test Incrementally (Don't Batch All Tasks)**
- Validate each task before moving to next
- Failing to test early = cascading bugs later
- Use validation test suite for regression testing

---

## Questions to Ask During Implementation

### **When Implementing Metrics:**
- Is the Cypher query using the correct edge type (MODIFIES vs CO_CHANGED)?
- Is the evidence string specific and actionable?
- Is the metric cached in Redis (15-min TTL)?
- Does it handle missing data gracefully (file with no commits)?

### **When Modifying LLM Prompts:**
- Does the prompt emphasize specificity in recommendations?
- Are recommendation templates provided for common patterns?
- Is evidence formatted as natural language (not JSON)?
- Does confidence scoring criteria match ADR-005?

### **When Adding Multi-File Logic:**
- Does it query CO_CHANGED edges BETWEEN changed files?
- Does it explain the relationship explicitly in output?
- Does it recommend integration tests for coupled pairs?
- Does it handle single-file case gracefully (no regression)?

---

## Expected Code Changes Summary

**New Files (Create):**
- `internal/metrics/ownership_churn.go` - Tier 2 metric for churn analysis
- `internal/treesitter/diff_analyzer.go` - AST diffing for structural changes
- `internal/risk/multi_file_analyzer.go` - Cross-file relationship detection
- `internal/ai/confidence_scorer.go` - Confidence calculation logic
- `internal/git/commit_parser.go` - Commit reference parsing

**Modified Files (Enhance):**
- `cmd/crisk/check.go` - Add multi-file detection, commit reference handling
- `internal/risk/phase0.go` - Add structural change detection
- `internal/risk/phase1.go` - Ensure test_ratio is calculated and displayed
- `internal/risk/phase2.go` - Wire MODIFIES metric into investigation
- `internal/agent/investigator.go` - Add confidence-based stopping
- `internal/ai/prompts.go` - Rewrite decision and synthesis prompts
- `internal/output/formatters.go` - Format multi-file context in output

**Files to Leave Alone (No Changes Needed):**
- `internal/ingestion/` - Ingestion working correctly
- `internal/graph/` - Graph client operational
- `internal/temporal/` - CO_CHANGED edge creation correct

---

## Final Checklist

Before marking complete, verify:

- [ ] All 7 tasks implemented per PHASE2_INVESTIGATION_ROADMAP.md
- [ ] 5/6 validation tests passing (see VALIDATION_TESTING_SUMMARY.md)
- [ ] Unit tests added for new metrics
- [ ] Integration tests validate graph queries
- [ ] End-to-end test confirms full workflow
- [ ] Code follows DEVELOPMENT_WORKFLOW.md conventions
- [ ] Documentation updated (if API contracts changed)
- [ ] Performance targets met (<50ms queries, <15% FP rate)

---

## How to Start (First Actions)

1. **Open new Claude Code session** (this is a clean slate)

2. **Read essential documents** (1-2 hours):
   - PHASE2_INVESTIGATION_ROADMAP.md (primary guide)
   - VALIDATION_TESTING_SUMMARY.md (understand failures)
   - agentic_design.md (architecture context)
   - risk_assessment_methodology.md (metric formulas)

3. **Explore current implementation** (30 minutes):
   - Read `cmd/crisk/check.go` to understand flow
   - Trace how Phase 1 → Phase 2 escalation works
   - Identify where MODIFIES edges should be queried but aren't

4. **Start with Task 5** (quickest win, 1 hour):
   - Query graph: `MATCH (test:File)-[:TESTS]->(source) RETURN count(*)`
   - If 0, fix ingestion; if >0, wire into Phase 1
   - Validate: Test coverage appears in `crisk check` output

5. **Move to Task 1** (core fix, 2-3 hours):
   - Create `internal/metrics/ownership_churn.go`
   - Implement MODIFIES edge query per risk_assessment_methodology.md
   - Test with HeroSection.tsx (4 MODIFIES edges)

6. **Continue through roadmap** (Tasks 2, 3, 4, 6, 7 in order)

---

## Support Resources

**If Stuck on Graph Queries:**
- See [graph_ontology.md](../01-architecture/graph_ontology.md) for schema
- See [risk_assessment_methodology.md Section 6.2](../01-architecture/risk_assessment_methodology.md#62-cypher-query-patterns) for query examples
- Test queries directly in Neo4j browser: `http://localhost:7474`

**If Stuck on LLM Integration:**
- See [agentic_design.md Section 4.2](../01-architecture/agentic_design.md#42-llm-decision-loop-only-if-high-risk) for prompt templates
- See [prompt_engineering_design.md](../01-architecture/prompt_engineering_design.md) for context management

**If Stuck on Testing:**
- See [VALIDATION_TESTING_SUMMARY.md](VALIDATION_TESTING_SUMMARY.md) for test scenarios
- Run specific test after each task to validate fix
- Use `--explain` flag to see full investigation trace

---

**Session Owner:** Implementation Team
**Session Type:** Development (code implementation)
**Estimated Duration:** 17-24 hours (2-3 weeks with testing)
**Expected Outcome:** Production-ready Phase 2 investigation with <15% FP rate

**Good luck! Start with PHASE2_INVESTIGATION_ROADMAP.md and work through tasks systematically.**
