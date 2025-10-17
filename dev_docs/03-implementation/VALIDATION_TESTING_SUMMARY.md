# Validation Testing Summary

**Date:** October 13, 2025
**Purpose:** Document findings from comprehensive 3-layer risk assessment validation
**Status:** Testing Complete - Critical gaps identified

---

## Testing Overview

**Repository Tested:** omnara-ai/omnara (full history clone)
**Graph Metrics:**
- 421 files (Layer 1)
- 2,562 functions, 454 classes (Layer 1)
- 276 commits, 10 developers (Layer 2)
- 419 MODIFIES edges, 49,654 CO_CHANGED edges (Layer 2)
- 71 issues, 180 PRs (Layer 3)

**Test Scenarios:** 6 systematic tests covering LOW/MEDIUM/HIGH risk patterns

---

## Test Results Summary

| Test | Expected Risk | Actual Risk | Pass/Fail | Critical Gap |
|------|---------------|-------------|-----------|---------------|
| 1. High Coupling File | HIGH | HIGH | ✅ Pass | Minor: Generic recommendations |
| 2. Commit History | MEDIUM | HIGH | ❌ Fail | **MODIFIES edges not analyzed** |
| 3. Low-Risk File | LOW | LOW | ⚠️ Partial | No explanatory rationale |
| 4. Co-Changed Pair | MEDIUM | HIGH (both) | ❌ Fail | **No multi-file awareness** |
| 5. New Function | MEDIUM | HIGH | ❌ Fail | **Layer 1 data NOT used** |
| 6. Multi-File Commit | HIGH | LOW (after commit) | ❌ Fail | **No commit support** |

**Overall Assessment:** 1/6 tests passed, 3 critical gaps, 2 partial passes

---

## Critical Findings

### What's Working ✅

1. **CO_CHANGED Edge Detection**
   - Successfully detects high temporal coupling (80% frequency threshold)
   - Accurate classification of HIGH risk for files with strong co-change patterns
   - Fast execution (130ms for LOW risk files)

2. **Database Connectivity**
   - All three databases (Neo4j, Redis, Postgres) connect successfully
   - Graph queries execute without errors
   - Cache infrastructure operational

3. **Layer 2 CO_CHANGED Data Utilization**
   - 49,654 co-changed edges being queried and used in risk assessment
   - Frequency metrics accurately calculated
   - Temporal coupling patterns detected

### What's NOT Working ❌

#### Gap 1: MODIFIES Edges Completely Ignored (Layer 2 Temporal)
**Test Evidence:** Test 2 (Commit History)
- **Graph Data:** File has 4 MODIFIES edges (commit history) from ishaanforthewin@gmail.com
- **Expected Output:** "File modified 4 times in last 90 days (moderate churn)"
- **Actual Output:** Only mentions CO_CHANGED edges, MODIFIES edges never queried
- **Impact:** No churn/hotspot analysis, ownership information missing

**Root Cause:** Phase 2 investigation only queries CO_CHANGED relationships, never queries MODIFIES edges despite 419 edges available in graph.

#### Gap 2: Layer 1 Structural Data Completely Unused
**Test Evidence:** Test 5 (New Function Addition)
- **Graph Data:** 2,562 functions, 454 classes ingested
- **Test Action:** Added new export function to TypeScript file
- **Expected Output:** "Added new function codeRiskTestFunction() (7 lines)"
- **Actual Output:** `modification_types=[]` in logs, no structural analysis
- **Impact:** Cannot detect new functions, classes, or architectural changes

**Root Cause:** No AST diffing during `crisk check`. Phase 0 pre-analysis doesn't compare old/new file structure.

#### Gap 3: No Multi-File Context Awareness
**Test Evidence:** Test 4 (Co-Changed Pair)
- **Graph Data:** Two files with 100% co-change frequency (changed together 1/1 times)
- **Test Action:** Modified both files simultaneously
- **Expected Output:** "You're changing both files that change together 100%—this is expected but risky"
- **Actual Output:** Two separate HIGH risk warnings with no mention of relationship
- **Impact:** Misses when user correctly modifies coupled files together

**Root Cause:** Files analyzed independently, no cross-file CO_CHANGED query between changed files.

#### Gap 4: Generic Recommendations Only
**Test Evidence:** All tests showed generic advice
- **Common Output:** "Investigate temporal coupling patterns"
- **Expected:** "Add integration tests for auth.py + user_service.py interactions (85% co-change frequency)"
- **Impact:** User doesn't know what specific action to take

**Root Cause:** LLM prompts don't emphasize specificity, no recommendation templates provided.

#### Gap 5: No Commit-Based Analysis
**Test Evidence:** Test 6 (Multi-File Commit)
- **Test Action:** `crisk check HEAD` after creating commit
- **Expected Output:** Analyze all files in commit holistically
- **Actual Output 1:** HEAD treated as filename → error
- **Actual Output 2:** After commit, same file shows different risk (HIGH → LOW)
- **Impact:** Cannot analyze commits, inconsistent risk assessment

**Root Cause:** No git commit reference parsing, risk calculation uses inconsistent graph state.

#### Gap 6: No Explanatory Rationale for LOW Risk
**Test Evidence:** Test 3 (Low-Risk File)
- **Result:** Risk level correct (LOW)
- **Output:** Single line: "Risk level: LOW" with no explanation
- **Expected:** "LOW risk because: no coupling, no co-changes, isolated change"
- **Impact:** User doesn't learn why change is safe

**Root Cause:** Output formatter only displays issues/warnings, skips positive evidence for LOW risk.

### Missing Features ⚠️

1. **Structural Change Detection** - Adding functions/classes not detected
2. **Churn/Hotspot Analysis** - File modification frequency not reported
3. **Specific Co-Changed Files** - Doesn't list which files are historically coupled
4. **Integration Test Suggestions** - When modifying coupled files together
5. **Commit-Based Workflow** - No pre-commit hook support for commit analysis
6. **Explanatory Rationale for LOW risk** - Silent success, no educational feedback
7. **Recent Committer Information** - Developer data not utilized
8. **Issue/PR Correlation** - Layer 3 incident data completely unused

---

## Data Layer Utilization Analysis

### Layer 1 (Structure): ❌ NOT Used
**Evidence:** Test 5 showed `modification_types=[]` despite adding new function
- **Data Available:** 421 files, 2,562 functions, 454 classes, 2,090 imports
- **Data Used in Risk Assessment:** NONE
- **Status:** Ingested but completely ignored during `crisk check`

**Expected Use Cases:**
- Detect new functions/classes being added
- Analyze complexity changes
- Identify architectural modifications
- Recommend tests for new code

### Layer 2 (Temporal): ⚠️ Partially Used
**Evidence:** CO_CHANGED used, MODIFIES ignored
- **Data Available:** 276 commits, 10 developers, 419 MODIFIES edges, 49,654 CO_CHANGED edges
- **Data Used:** Only CO_CHANGED relationships (frequency > 80% threshold)
- **Data Ignored:** MODIFIES edges (commit history, authorship)
- **Status:** 50% utilization - only 1 of 2 temporal data types used

**Expected Use Cases (Missing):**
- Churn analysis: "File modified 12 times in 90 days"
- Ownership transitions: "Owner changed from Bob to Alice 14 days ago"
- Hotspot detection: "High churn + low coverage = elevated risk"

### Layer 3 (Incidents): ❌ NOT Used
**Evidence:** No test mentioned issues or PRs in output
- **Data Available:** 71 issues, 180 PRs ingested
- **Data Used in Risk Assessment:** NONE
- **Status:** Completely unused despite ingestion

**Expected Use Cases:**
- Incident similarity: "Similar to INC-123: Auth timeout failure"
- Historical pattern detection: "This file caused 3 incidents in last 6 months"
- Risk elevation: "Past incidents + high coupling = CRITICAL risk"

---

## Performance Metrics

| Metric | Target | Actual | Pass? | Notes |
|--------|--------|--------|-------|-------|
| LOW risk check time | <200ms | 130ms | ✅ Yes | Excellent performance |
| HIGH risk check time | <5s | ~100ms | ✅ Yes | Phase 2 fast (but incomplete) |
| False positives observed | 0 | 3/6 tests | ❌ No | 50% FP rate from gaps |
| False negatives observed | 0 | 1/6 tests | ❌ No | Test 5: should flag new code |

**Performance Insight:** Phase 2 runs fast (~100ms) but doesn't do enough analysis. Speed is good, but value is low.

---

## Architectural Issues

### Issue 1: File-Centric Design
**Problem:** Each file processed independently, no batch context
**Impact:** Cannot detect multi-file patterns (co-changed pairs)
**Solution:** Add `AnalyzeMultipleFiles()` function with cross-file queries

### Issue 2: Single Risk Signal Dependency
**Problem:** Risk assessment primarily driven by CO_CHANGED frequency (80% threshold)
**Impact:** Ignores other signals (churn, structure, incidents)
**Solution:** Multi-signal evidence aggregation via LLM synthesis

### Issue 3: No Diff Analysis
**Problem:** `crisk check` doesn't analyze what changed, only which files changed
**Impact:** Cannot detect structural changes (new functions, classes)
**Solution:** Add tree-sitter AST diffing in Phase 0

### Issue 4: Static Threshold (80% Hard-Coded)
**Problem:** CO_CHANGED frequency threshold hard-coded at 0.8
**Impact:** May be too strict for some repos, too lenient for others
**Solution:** Adaptive thresholds based on repository patterns

### Issue 5: No Commit Workflow Support
**Problem:** Cannot parse commit references (HEAD, SHA, ranges)
**Impact:** Pre-commit workflow broken, inconsistent risk assessment
**Solution:** Add git commit reference parser

### Issue 6: Data Pipeline Disconnect
**Problem:** Ingestion collects rich data (3 layers), but risk assessment only uses CO_CHANGED edges
**Impact:** Wasted computation during init, unused graph data
**Solution:** Wire Layer 1 and MODIFIES edges into Phase 2

---

## Recommendations

### **Primary Recommendation: PATH 3 - Enhance Current System**

**Justification:**
- ✅ **Graph Foundation Solid:** All 3 layers properly ingested with correct data
- ✅ **CO_CHANGED Detection Works:** Core temporal coupling logic is sound
- ❌ **Major Gaps in Utilization:** Only using ~10% of available graph data
- ❌ **Architecture Limiting:** File-centric design blocks multi-file analysis
- ⚠️ **Medium Effort Required:** Significant enhancements needed but no full rewrite

**Why Not PATH 2 (Fresh Hard Ingestion):**
- Graph data is complete and correct (verified in all tests)
- Problem is in risk assessment logic, NOT data ingestion
- Re-ingesting won't fix architectural issues

---

## Priority Bug Fixes

### [P0 - Critical] Must Fix for Production

**P0-1: Enable Layer 1 Structural Analysis**
- **Issue:** `modification_types=[]` in all tests, structural changes ignored
- **Impact:** Cannot detect new functions, classes, architectural changes
- **Effort:** 3-4 hours
- **Fix:** Implement AST diff analysis in Phase 0 pre-analysis
- **See:** [PHASE2_INVESTIGATION_ROADMAP.md Task 2](PHASE2_INVESTIGATION_ROADMAP.md#task-2-add-structural-change-detection-layer-1-utilization)

**P0-2: Utilize MODIFIES Edges (Layer 2 Temporal)**
- **Issue:** 419 MODIFIES edges in graph but never queried
- **Impact:** Missing churn/hotspot analysis, no ownership data
- **Effort:** 2-3 hours
- **Fix:** Add ownership_churn metric calculation, query MODIFIES edges in Phase 2
- **See:** [PHASE2_INVESTIGATION_ROADMAP.md Task 1](PHASE2_INVESTIGATION_ROADMAP.md#task-1-implement-modifies-edge-analysis-churnhotspot-detection)

**P0-3: Add Multi-File Context Awareness**
- **Issue:** Files analyzed independently, missing when co-changed pairs modified together
- **Impact:** Cannot provide holistic commit-level risk assessment
- **Effort:** 4-5 hours
- **Fix:** Implement `AnalyzeMultipleFiles()` with cross-file CO_CHANGED queries
- **See:** [PHASE2_INVESTIGATION_ROADMAP.md Task 3](PHASE2_INVESTIGATION_ROADMAP.md#task-3-multi-file-context-awareness-holistic-analysis)

### [P1 - High] Should Fix Soon

**P1-4: Provide Specific Recommendations**
- **Issue:** All recommendations generic ("Investigate temporal coupling")
- **Impact:** Poor user experience, no actionable guidance
- **Effort:** 2-3 hours
- **Fix:** Enhance LLM prompts with specificity requirements and templates
- **See:** [PHASE2_INVESTIGATION_ROADMAP.md Task 4](PHASE2_INVESTIGATION_ROADMAP.md#task-4-improve-llm-decision-prompts-evidence-based-reasoning)

**P1-5: Fix Test Coverage Calculation**
- **Issue:** Test coverage ratio never mentioned in outputs
- **Effort:** 1-2 hours
- **Fix:** Verify TESTS relationship creation, ensure metric queried in Phase 1
- **See:** [PHASE2_INVESTIGATION_ROADMAP.md Task 5](PHASE2_INVESTIGATION_ROADMAP.md#task-5-verify-test-coverage-ratio-calculation)

**P1-6: Add Confidence-Based Early Stopping**
- **Issue:** Always runs 3 hops even when evidence is conclusive
- **Effort:** 2-3 hours
- **Fix:** Implement confidence scoring, stop when confidence ≥ 0.85
- **See:** [PHASE2_INVESTIGATION_ROADMAP.md Task 6](PHASE2_INVESTIGATION_ROADMAP.md#task-6-add-confidence-based-early-stopping)

### [P2 - Medium] Nice to Have

**P2-7: Implement Commit-Based Analysis**
- **Issue:** Cannot analyze `HEAD` or commit references
- **Effort:** 3-4 hours
- **Fix:** Add git commit reference parser, ensure consistent graph state
- **See:** [PHASE2_INVESTIGATION_ROADMAP.md Task 7](PHASE2_INVESTIGATION_ROADMAP.md#task-7-add-commit-based-analysis-support)

**P2-8: Add LOW Risk Explanations**
- **Issue:** No rationale provided for LOW risk assessments
- **Effort:** 1 hour
- **Fix:** Update output formatter to show positive evidence

**P2-9: Utilize Layer 3 Incident Data**
- **Issue:** 71 issues/180 PRs never mentioned in output
- **Effort:** 3-4 hours
- **Fix:** Query CAUSED_BY, FIXED_BY edges, implement incident similarity

---

## Next Steps

### Immediate Actions (This Week)

1. **Read Implementation Roadmap** - [PHASE2_INVESTIGATION_ROADMAP.md](PHASE2_INVESTIGATION_ROADMAP.md)
2. **Start with Task 1** - MODIFIES edge analysis (quickest win)
3. **Validate Fix** - Re-run Test 2 (Commit History) to verify output

### Short-Term Goals (2-3 Weeks)

- Complete P0 tasks (Tasks 1, 2, 3, 5)
- All 3 layers utilized in risk assessment
- Multi-file relationship detection working
- Re-run all 6 validation tests → Target 5/6 passing

### Success Criteria

**Technical Metrics:**
- ✅ All 3 layers (Structure, Temporal, Incidents) used in risk assessment
- ✅ MODIFIES edges analyzed for churn patterns
- ✅ Multi-file context awareness implemented
- ✅ Specific, actionable recommendations (not generic)

**User Experience Validation:**
- ✅ Test 2: Shows "4 commits in 90 days" (churn detected)
- ✅ Test 4: Detects "changing both files that are 100% coupled"
- ✅ Test 5: Shows "Added function codeRiskTestFunction()"
- ✅ Test 6: Supports `crisk check HEAD` workflow

**Quality Metrics:**
- False Positive Rate: 50% → <15%
- Test Pass Rate: 1/6 → 5/6
- Specific Recommendations: 0% → 80%+

---

## Appendix: Test-by-Test Details

### Test 1: High Coupling File (apps/web/src/types/dashboard.ts)
- **Expected:** HIGH risk due to 260 CO_CHANGED relationships
- **Actual:** HIGH risk (80% co-change frequency)
- **Pass:** ✅ Yes
- **Gap:** Generic recommendation ("Investigate temporal coupling patterns")

### Test 2: Commit History (apps/web/src/components/landing/HeroSection.tsx)
- **Expected:** MEDIUM risk due to 4 MODIFIES edges
- **Actual:** HIGH risk (only CO_CHANGED mentioned)
- **Pass:** ❌ No - MODIFIES edges not analyzed
- **Gap:** File has 4 commits but this isn't mentioned in output

### Test 3: Low-Risk Stable File (src/backend/api/__init__.py)
- **Expected:** LOW risk (no temporal edges)
- **Actual:** LOW risk (130ms, fast)
- **Pass:** ⚠️ Partial - No explanatory rationale
- **Gap:** Output is minimal, user doesn't learn why it's safe

### Test 4: Co-Changed Pair (chat/index.ts + app/_layout.tsx)
- **Expected:** MEDIUM risk (modifying coupled files together is expected)
- **Actual:** HIGH risk for both files (analyzed separately)
- **Pass:** ❌ No - No multi-file awareness
- **Gap:** Doesn't detect user is changing both parts of coupled system

### Test 5: New Function Addition (apps/web/src/types/dashboard.ts)
- **Expected:** MEDIUM risk (new code in high-coupling file)
- **Actual:** HIGH risk (no structural analysis)
- **Pass:** ❌ No - Layer 1 data NOT used
- **Gap:** `modification_types=[]`, doesn't detect new function

### Test 6: Multi-File Commit
- **Expected:** HIGH risk (multiple coupled files in commit)
- **Actual 1:** HEAD treated as filename (error)
- **Actual 2:** LOW risk after commit (inconsistent)
- **Pass:** ❌ No - No commit support
- **Gap:** Cannot analyze commits, risk changes after commit

---

**Document Owner:** Risk Assessment Team
**Review Schedule:** After each P0 fix implementation
**Related Documents:**
- [PHASE2_INVESTIGATION_ROADMAP.md](PHASE2_INVESTIGATION_ROADMAP.md) - Implementation tasks
- [status.md](status.md) - Current implementation status
- [DEVELOPMENT_WORKFLOW.md](../DEVELOPMENT_WORKFLOW.md) - How to implement fixes
