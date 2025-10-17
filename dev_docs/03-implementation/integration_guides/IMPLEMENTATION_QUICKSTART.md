# CodeRisk Phase 2 Investigation - Implementation Quickstart

**Status:** Ready to implement (documentation complete)
**Last Updated:** October 13, 2025

---

## What We Discovered

Comprehensive validation testing of the 3-layer risk assessment system revealed:

- ✅ **Graph construction is perfect** - All 3 layers properly ingested
- ❌ **Risk assessment only uses 10% of graph data** - Major utilization gaps
- 🎯 **Result:** 1/6 tests passed, 50% false positive rate

---

## The Problem

**Current State:**
```
User runs: crisk check auth.py

System does:
1. Queries CO_CHANGED edges ✅
2. Returns generic "HIGH risk: investigate coupling" ⚠️

System DOESN'T:
❌ Query MODIFIES edges (commit history)
❌ Detect structural changes (new functions)
❌ Consider multi-file context
❌ Provide specific recommendations
```

**Target State:**
```
User runs: crisk check auth.py user_service.py

System does:
1. Queries CO_CHANGED edges ✅
2. Queries MODIFIES edges (churn analysis) ✅
3. Detects structural changes (AST diff) ✅
4. Detects multi-file coupling ✅
5. Returns: "HIGH risk: auth.py and user_service.py change together 85%
   → Add integration tests for auth + user_service interactions" ✅
```

---

## The Solution: 7 Implementation Tasks

### **P0 (Critical) - Week 1: Fix Core Metrics**
1. **Task 5:** Verify test coverage calculation (1-2h) - Quick win
2. **Task 1:** MODIFIES edge analysis - Add churn/ownership metrics (2-3h)
3. **Task 2:** Structural change detection - Detect new functions via AST diff (3-4h)
4. **Task 3:** Multi-file context - Detect co-changed pairs being modified (4-5h)

### **P1 (High) - Week 2: Improve Intelligence**
5. **Task 4:** Better LLM prompts - Specific recommendations (2-3h)
6. **Task 6:** Confidence-based stopping - Early exit when confident (2-3h)

### **P2 (Medium) - Week 3: Add Workflow**
7. **Task 7:** Commit-based analysis - Support `crisk check HEAD` (3-4h)

**Total Time:** 17-24 hours (2-3 weeks)

---

## How to Start Implementation

### **Step 1: Open New Claude Code Session**

Copy this prompt into a new Claude Code session:

```
I need to implement proper Phase 2 LLM investigation for CodeRisk that utilizes all 3 graph layers (Structure, Temporal, Incidents). The graph construction is complete and working, but risk assessment only uses CO_CHANGED edges, resulting in a 50% false positive rate.

Please read these essential documents to understand context:

1. coderisk-go/dev_docs/03-implementation/IMPLEMENTATION_SESSION_PROMPT.md - Complete session guide (READ THIS FIRST)
2. coderisk-go/dev_docs/03-implementation/PHASE2_INVESTIGATION_ROADMAP.md - 7 implementation tasks with technical details
3. coderisk-go/dev_docs/03-implementation/VALIDATION_TESTING_SUMMARY.md - Gap analysis from testing
4. coderisk-go/dev_docs/DEVELOPMENT_WORKFLOW.md - Development conventions

After reading, help me implement the 7 tasks in priority order:

P0 Tasks (Week 1):
- Task 5: Verify test coverage calculation works
- Task 1: Add MODIFIES edge analysis (churn/ownership metrics)
- Task 2: Add structural change detection (AST diffing)
- Task 3: Add multi-file context awareness

P1 Tasks (Week 2):
- Task 4: Improve LLM prompts for specific recommendations
- Task 6: Add confidence-based early stopping

P2 Tasks (Week 3):
- Task 7: Add commit-based analysis support

Start by reading IMPLEMENTATION_SESSION_PROMPT.md and confirming you understand the context, then we'll begin with Task 5 (quickest win).
```

### **Step 2: Session Will Guide You**

The session prompt (IMPLEMENTATION_SESSION_PROMPT.md) contains:
- ✅ All context documents to read
- ✅ Step-by-step implementation approach
- ✅ Code patterns and examples
- ✅ Testing strategy after each task
- ✅ Common pitfalls to avoid
- ✅ Success criteria and validation

### **Step 3: Validate Each Task**

After implementing each task, run specific validation test:

```bash
# Task 1 (MODIFIES): Test churn detection
./crisk check apps/web/src/components/landing/HeroSection.tsx
# Expected: "File modified 4 times in last 90 days"

# Task 2 (Structural): Test function detection
echo "export function test() {}" >> src/test.ts
git add src/test.ts
./crisk check src/test.ts
# Expected: "Added new function test()"

# Task 3 (Multi-File): Test co-changed pair detection
./crisk check \
  apps/mobile/src/components/chat/index.ts \
  apps/mobile/src/app/_layout.tsx
# Expected: "Files change together 100% of the time"
```

---

## Key Documents Created

### **Primary Implementation Guides:**

1. **[IMPLEMENTATION_SESSION_PROMPT.md](coderisk-go/dev_docs/03-implementation/IMPLEMENTATION_SESSION_PROMPT.md)**
   - Complete Claude Code session prompt
   - Context documents to read
   - Step-by-step implementation approach
   - **USE THIS to start the session**

2. **[PHASE2_INVESTIGATION_ROADMAP.md](coderisk-go/dev_docs/03-implementation/PHASE2_INVESTIGATION_ROADMAP.md)**
   - Detailed technical specification for 7 tasks
   - Problem statement, approach, success criteria for each
   - Cypher query examples
   - Testing strategy
   - **REFERENCE THIS during implementation**

3. **[VALIDATION_TESTING_SUMMARY.md](coderisk-go/dev_docs/03-implementation/VALIDATION_TESTING_SUMMARY.md)**
   - Complete gap analysis from 6 validation tests
   - What's working vs what's broken
   - Layer-by-layer utilization analysis
   - **UNDERSTAND THIS to know what to fix**

### **Supporting Documents:**

4. **[status.md](coderisk-go/dev_docs/03-implementation/status.md)** - Updated with critical findings
5. **[DEVELOPMENT_WORKFLOW.md](coderisk-go/dev_docs/DEVELOPMENT_WORKFLOW.md)** - Development conventions

---

## Success Metrics

**Before Implementation:**
- ❌ Test pass rate: 1/6 (17%)
- ❌ False positive rate: 50%
- ❌ Layer utilization: CO_CHANGED only (10% of data)
- ❌ Specific recommendations: 0%

**After Implementation:**
- ✅ Test pass rate: 5/6 (83%)
- ✅ False positive rate: <15%
- ✅ Layer utilization: All 3 layers (Structure, Temporal, Incidents)
- ✅ Specific recommendations: 80%+

**Example of Success:**
```bash
# Before (generic)
$ crisk check auth.py
🔴 HIGH risk
Issues:
1. High coupling (12 files)
Recommendations:
- Investigate temporal coupling patterns

# After (specific)
$ crisk check auth.py user_service.py
🔴 HIGH risk (2 files)
Issues:
1. auth.py modified 12 times in last 90 days (HIGH churn)
2. auth.py and user_service.py change together 85% of the time
3. Added new function authenticate_user() (45 lines)
4. Test coverage: 30% (LOW)
Recommendations:
1. Add integration tests for auth.py + user_service.py interactions (85% co-change pattern)
2. Add unit tests for new function authenticate_user()
3. Consider refactoring to reduce coupling (12 dependencies is high for authentication code)
```

---

## Architecture References

For deeper understanding (read during implementation, not before):

- **[agentic_design.md](coderisk-go/dev_docs/01-architecture/agentic_design.md)** - LLM investigation framework
- **[risk_assessment_methodology.md](coderisk-go/dev_docs/01-architecture/risk_assessment_methodology.md)** - Metric formulas
- **[system_overview_layman.md](coderisk-go/dev_docs/01-architecture/system_overview_layman.md)** - Conceptual explanation
- **[developer_experience.md](coderisk-go/dev_docs/00-product/developer_experience.md)** - User workflows
- **[user_personas.md](coderisk-go/dev_docs/00-product/user_personas.md)** - User needs

---

## FAQ

### Q: Why not just re-ingest the graph?
**A:** Graph construction is perfect. Problem is in risk assessment logic (how we query/use the graph), not data collection. Re-ingesting won't fix the code that queries it.

### Q: Can I work on tasks in parallel?
**A:** Tasks 1, 2, 5 are independent and can be parallelized. Task 3 depends on Tasks 1-2. Task 4 depends on 1-3. Task 6 depends on 4. Task 7 depends on 3.

### Q: What if I get stuck?
**A:** IMPLEMENTATION_SESSION_PROMPT.md has "Support Resources" section with:
- Graph query help (schema docs, Neo4j browser)
- LLM integration help (prompt templates)
- Testing help (validation test details)

### Q: How long will this take?
**A:** 17-24 hours total estimated. If working full-time, ~1 week for P0, ~3-4 days for P1, ~2-3 days for P2. Part-time: 2-3 weeks.

### Q: Is this a rewrite?
**A:** No! 70% of code stays the same. We're enhancing what exists:
- Graph construction: No changes (working perfectly)
- Phase 1 baseline: Minor additions (test coverage verification)
- Phase 2 investigation: Major additions (new metrics, better prompts)

---

## Ready to Start?

1. ✅ Copy the Claude Code prompt from "Step 1" above
2. ✅ Open new Claude Code session
3. ✅ Paste prompt and let it guide you
4. ✅ Follow IMPLEMENTATION_SESSION_PROMPT.md step-by-step
5. ✅ Validate each task before moving to next

**The documentation is complete and comprehensive. You have everything needed to implement successfully.**

---

**Questions?** See IMPLEMENTATION_SESSION_PROMPT.md "Support Resources" section.

**Track Progress:** Update status.md after each task completion.

**Good luck! 🚀**
