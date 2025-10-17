# CodeRisk Gap Fix Task Index

**Purpose:** Organized task documents for fixing all E2E test gaps
**Created:** October 6, 2025
**Based On:** [E2E_TEST_GAP_ANALYSIS.md](E2E_TEST_GAP_ANALYSIS.md) and [E2E_TEST_FINAL_REPORT.md](E2E_TEST_FINAL_REPORT.md)

---

## Quick Start

**For Claude Code Agent Sessions:**

Each task document below is designed to be run in a fresh Claude Code agent session. They are self-contained with:
- Complete context & problem statement
- Architecture documentation references
- Step-by-step implementation instructions
- Testing & validation procedures
- Success criteria checklist

**Recommended Execution Order:**
1. **P0 (CRITICAL)** - Run first to unblock core value prop
2. **P1 (HIGH)** - Run in parallel or sequence (3 tasks)
3. **P2 (MEDIUM)** - Run after P1 tasks complete

---

## Task Documents Overview

### 🔴 P0 - CRITICAL (Must Fix First)

#### [TASK_P0_PHASE2_INTEGRATION.md](TASK_P0_PHASE2_INTEGRATION.md)
**Gap:** C1 - Phase 2 LLM Investigation Never Runs
**Priority:** P0 (BLOCKS CORE VALUE PROPOSITION)
**Estimated Time:** 2-3 hours
**Impact:** 85% of advertised functionality blocked

**What You'll Fix:**
- Integrate Phase 2 escalation into `cmd/crisk/check.go`
- Create real client instantiation (temporal, incidents, LLM)
- Add output formatting for investigation results
- Handle missing API key gracefully

**Success Criteria:**
- Phase 2 runs when risk threshold exceeded ✅
- Investigation trace displayed in all verbosity modes ✅
- LLM called with proper evidence context ✅

**Claude Code Prompt:**
```
Read TASK_P0_PHASE2_INTEGRATION.md and implement Phase 2 integration into the check command.
Follow the step-by-step instructions, ensure all tests pass, and cite 12-factor principles
(Factor 8 and Factor 10) in code comments.
```

---

### 🟡 P1 - HIGH (Missing Advertised Features)

#### [TASK_P1_EDGE_CREATION_FIXES.md](TASK_P1_EDGE_CREATION_FIXES.md)
**Gaps:** A1 (CO_CHANGED edges) + B1 (CAUSED_BY edges)
**Priority:** P1 (MISSING ADVERTISED FEATURES)
**Estimated Time:** 3-5 hours
**Impact:** Layer 2 & 3 graph queries fail, risk assessment incomplete

**What You'll Fix:**
- **Part 1 (Gap A1):** CO_CHANGED edge creation in Neo4j
  - Add timeout handling for git history parsing
  - Add edge verification after creation
  - Batch edge creation for performance

- **Part 2 (Gap B1):** CAUSED_BY edge creation in Neo4j
  - Fix edge creation in incident linker
  - Verify graph client interface compatibility
  - Add error handling and logging

- **Bonus:** Fix incident search NULL handling bug

**Success Criteria:**
- CO_CHANGED edges persist to Neo4j (>0 count) ✅
- CAUSED_BY edges created on incident link ✅
- Performance targets met (<20ms, <50ms) ✅
- Integration tests pass ✅

**Claude Code Prompt:**
```
Read TASK_P1_EDGE_CREATION_FIXES.md and fix both CO_CHANGED and CAUSED_BY edge creation.
This addresses Gaps A1 and B1. Follow the implementation steps for both parts, add timeout
handling, verification queries, and ensure all integration tests pass.
```

---

#### [TASK_P1_AI_MODE_COMPLETION.md](TASK_P1_AI_MODE_COMPLETION.md)
**Gap:** UX1 - AI Mode JSON Incomplete
**Priority:** P1 (MISSING ADVERTISED FEATURE)
**Estimated Time:** 4-6 hours
**Impact:** AI assistants (Claude Code, Cursor) cannot integrate

**What You'll Fix:**
- Generate AI assistant action prompts (auto-fix suggestions)
- Calculate blast radius from graph (1-3 hop queries)
- Get temporal coupling data from CO_CHANGED edges
- Identify hotspots (high churn + low coverage)
- Populate all missing JSON fields per spec

**Success Criteria:**
- `ai_assistant_actions[]` populated with actionable prompts ✅
- Graph analysis includes blast radius, coupling, hotspots ✅
- Recommendations categorized (critical/high/medium/future) ✅
- JSON schema matches developer_experience.md 100% ✅

**Claude Code Prompt:**
```
Read TASK_P1_AI_MODE_COMPLETION.md and complete the AI Mode JSON output schema.
Implement AI action generation, graph analysis functions, and enhance the converter.
Cite 12-factor Factor 4 (Tools are structured outputs) in code comments.
```

---

### 🔵 P2 - MEDIUM (Testing & Validation)

#### [TASK_P2_TESTING_VALIDATION.md](TASK_P2_TESTING_VALIDATION.md)
**Gaps:** Testing & validation for Layers 2 & 3
**Priority:** P2 (TESTING & VALIDATION)
**Estimated Time:** 2-3 hours
**Impact:** Automated regression prevention

**What You'll Fix:**
- Create Layer 2 integration test (CO_CHANGED edge validation)
- Create Layer 3 integration test (CAUSED_BY edge validation)
- Add performance benchmark tests
- Fix `--version` flag (minor bug)
- Update Makefile with integration test targets

**Success Criteria:**
- All integration tests pass consistently ✅
- Performance benchmarks validate (<20ms, <50ms) ✅
- CLI bugs fixed ✅
- CI/CD integration working ✅

**Claude Code Prompt:**
```
Read TASK_P2_TESTING_VALIDATION.md and add integration tests for Layers 2 & 3.
Create automated validation scripts, performance benchmarks, and fix minor CLI bugs.
Update the Makefile and ensure all tests are repeatable.
```

---

## Execution Strategy

### Option 1: Sequential Execution (Safe, Slower)

**Recommended for:** Single developer, learning codebase

```bash
# Session 1: Fix P0 (2-3 hours)
# Run TASK_P0_PHASE2_INTEGRATION.md
# Validate: Phase 2 runs successfully

# Session 2: Fix P1 Part 1 (3-5 hours)
# Run TASK_P1_EDGE_CREATION_FIXES.md
# Validate: Both edge types persist to Neo4j

# Session 3: Fix P1 Part 2 (4-6 hours)
# Run TASK_P1_AI_MODE_COMPLETION.md
# Validate: AI Mode JSON complete

# Session 4: Add P2 Tests (2-3 hours)
# Run TASK_P2_TESTING_VALIDATION.md
# Validate: All integration tests pass
```

**Total Time:** 11-17 hours sequential

---

### Option 2: Parallel Execution (Fast, Requires Coordination)

**Recommended for:** Team with 3-4 developers

```bash
# Developer 1 (CRITICAL PATH)
Session A: TASK_P0_PHASE2_INTEGRATION.md (2-3 hours)
# Phase 2 must work before full end-to-end demo

# Developer 2 (GRAPH TEAM)
Session B: TASK_P1_EDGE_CREATION_FIXES.md (3-5 hours)
# Edge creation is independent of Phase 2

# Developer 3 (OUTPUT TEAM)
Session C: TASK_P1_AI_MODE_COMPLETION.md (4-6 hours)
# AI Mode depends on edge fixes for graph analysis
# Can start immediately if using mock data initially

# Developer 4 (QA/TESTING)
Session D: TASK_P2_TESTING_VALIDATION.md (2-3 hours)
# Can start early, will need to update as fixes land
```

**Total Time:** ~6 hours parallel (longest task is AI Mode at 4-6 hours)

---

### Option 3: Hybrid (Balanced)

**Recommended for:** Small team (2 developers)

```bash
# Developer 1 (Backend)
Session 1: TASK_P0_PHASE2_INTEGRATION.md (2-3 hours)
Session 2: TASK_P1_EDGE_CREATION_FIXES.md (3-5 hours)
# Total: 5-8 hours

# Developer 2 (Output/Testing)
Session 1: TASK_P1_AI_MODE_COMPLETION.md (4-6 hours)
Session 2: TASK_P2_TESTING_VALIDATION.md (2-3 hours)
# Total: 6-9 hours

# Both developers finish within ~9 hours
```

---

## Dependency Graph

```
TASK_P0_PHASE2_INTEGRATION
  ↓ (required for full demo)
  ├── TASK_P1_EDGE_CREATION_FIXES
  │   ├── Part 1: CO_CHANGED edges (Gap A1)
  │   └── Part 2: CAUSED_BY edges (Gap B1)
  │       ↓ (edge data needed for graph analysis)
  │       └── TASK_P1_AI_MODE_COMPLETION
  │           ├── Uses CO_CHANGED for temporal coupling
  │           └── Uses CAUSED_BY for incident analysis
  │
  └── TASK_P2_TESTING_VALIDATION
      ├── Validates P0 (Phase 2 integration)
      ├── Validates P1 (edge creation)
      └── Validates P1 (AI mode output)
```

**Key Dependencies:**
- **P0 → Demo:** Phase 2 must work for end-to-end demo
- **P1 Edges → P1 AI Mode:** Graph analysis needs edge data
- **P0 + P1 → P2:** Tests validate all fixes

---

## Testing After Each Task

### After P0 (Phase 2 Integration)
```bash
# Validate Phase 2 runs
export OPENAI_API_KEY="sk-..."
./crisk check <high-risk-file>
# Expected: Investigation trace shown ✅
```

### After P1 Part 1 (Edge Creation)
```bash
# Validate CO_CHANGED edges
docker exec coderisk-neo4j cypher-shell \
  "MATCH ()-[r:CO_CHANGED]->() RETURN count(r)"
# Expected: >0 ✅

# Validate CAUSED_BY edges
./crisk incident link <id> <file>
docker exec coderisk-neo4j cypher-shell \
  "MATCH ()-[r:CAUSED_BY]->() RETURN count(r)"
# Expected: ≥1 ✅
```

### After P1 Part 2 (AI Mode)
```bash
# Validate AI Mode JSON
./crisk check --ai-mode <file> | jq '.ai_assistant_actions | length'
# Expected: >0 ✅

./crisk check --ai-mode <file> | jq '.graph_analysis.blast_radius.total_affected_files'
# Expected: ≥0 ✅
```

### After P2 (Testing)
```bash
# Run integration tests
make integration
# Expected: All tests pass ✅
```

---

## Success Validation Checklist

**After ALL tasks complete, verify against original E2E report:**

- [ ] ✅ **Test 3.2 (Phase 2):** PASS - Investigation runs and displays results
- [ ] ✅ **Test 1.3 (Layer 2):** PASS - CO_CHANGED edges >0 in Neo4j
- [ ] ✅ **Test 2.2 (Layer 3):** PASS - CAUSED_BY edge created
- [ ] ✅ **Test 4.2 (AI Mode):** PASS - Full JSON schema matches spec
- [ ] ✅ **Performance:** <20ms co-change, <50ms incident search
- [ ] ✅ **Integration Tests:** All pass consistently

**Re-run full E2E test:**
```bash
# Use same test repository
cd /tmp/omnara
/path/to/crisk init-local
# Verify: CO_CHANGED edges created ✅

./crisk incident create "Test" "..." --severity high
INCIDENT_ID="..."
./crisk incident link "$INCIDENT_ID" "src/file.ts" --line 1
# Verify: CAUSED_BY edge created ✅

export OPENAI_API_KEY="sk-..."
./crisk check src/file.ts --explain
# Verify: Phase 2 investigation shown ✅

./crisk check src/file.ts --ai-mode | jq '.ai_assistant_actions'
# Verify: Actions populated ✅
```

---

## Developer Resources

### Before Starting Any Task

**1. Read Development Workflow:**
```bash
cat dev_docs/DEVELOPMENT_WORKFLOW.md
```

**2. Check 12-Factor Principles:**
```bash
cat dev_docs/12-factor-agents-main/README.md
```

**3. Review Architecture:**
```bash
cat dev_docs/01-architecture/system_overview_layman.md | head -100
```

### During Implementation

**Follow checklist from DEVELOPMENT_WORKFLOW.md:**
- [ ] Read spec.md relevant sections
- [ ] Check security constraints (if applicable)
- [ ] Read 12-factor principles for your task type
- [ ] Implement with proper error handling
- [ ] Add structured logging (slog)
- [ ] Write tests
- [ ] Run quality checks (build, test, fmt)
- [ ] Cite 12-factor principles in code

### After Implementation

**Quality Gates (from DEVELOPMENT_WORKFLOW.md):**
```bash
go build ./...           # Compiles ✅
go test ./...            # Unit tests pass ✅
go test -race ./...      # No race conditions ✅
go fmt ./...             # Formatted ✅
go mod tidy              # Dependencies clean ✅
make integration         # Integration tests pass ✅
```

---

## Troubleshooting

### Common Issues

**Issue:** "Phase 2 doesn't trigger"
- **Check:** `OPENAI_API_KEY` set?
- **Check:** Risk threshold exceeded? (coupling >0.5 or incidents >0)
- **See:** [TASK_P0_PHASE2_INTEGRATION.md](TASK_P0_PHASE2_INTEGRATION.md#troubleshooting)

**Issue:** "CO_CHANGED edges still 0"
- **Check:** Init-local completed without timeout?
- **Check:** Temporal analysis logs show edge creation?
- **See:** [TASK_P1_EDGE_CREATION_FIXES.md](TASK_P1_EDGE_CREATION_FIXES.md#troubleshooting)

**Issue:** "CAUSED_BY edge not found"
- **Check:** Incident node exists in Neo4j?
- **Check:** File node exists with matching path?
- **See:** [TASK_P1_EDGE_CREATION_FIXES.md](TASK_P1_EDGE_CREATION_FIXES.md#task-21-debug-and-fix-edge-creation-in-linker)

**Issue:** "AI Mode JSON missing fields"
- **Check:** Graph client passed to converter?
- **Check:** CO_CHANGED edges exist for temporal coupling?
- **See:** [TASK_P1_AI_MODE_COMPLETION.md](TASK_P1_AI_MODE_COMPLETION.md#troubleshooting)

---

## Final Deliverable

**After all tasks complete, create summary PR:**

```bash
git checkout -b fix/all-e2e-gaps

# Commit each task separately
git add <P0 files>
git commit -m "$(cat TASK_P0_PHASE2_INTEGRATION.md | grep -A 20 'Commit Message')"

git add <P1 Part 1 files>
git commit -m "$(cat TASK_P1_EDGE_CREATION_FIXES.md | grep -A 20 'Commit Message')"

git add <P1 Part 2 files>
git commit -m "$(cat TASK_P1_AI_MODE_COMPLETION.md | grep -A 20 'Commit Message')"

git add <P2 files>
git commit -m "$(cat TASK_P2_TESTING_VALIDATION.md | grep -A 20 'Commit Message')"

# Create PR
git push origin fix/all-e2e-gaps
gh pr create --title "Fix all E2E test gaps (P0-P2)" \
  --body "Implements fixes for Gaps C1, A1, B1, UX1 from E2E test report.

  - ✅ P0: Phase 2 integration (Gap C1)
  - ✅ P1: CO_CHANGED edges (Gap A1)
  - ✅ P1: CAUSED_BY edges (Gap B1)
  - ✅ P1: AI Mode completion (Gap UX1)
  - ✅ P2: Integration tests

  All E2E tests now pass. System readiness: 100%"
```

---

## Next Steps After Fixes

**1. Update Documentation**
```bash
# Update implementation status
vim dev_docs/03-implementation/status.md
# Change Phase 2: 0% → 100%
# Change Layer 2: 95% → 100%
# Change Layer 3: 100% → 100% (verified)
# Change Overall: 85% → 100%
```

**2. Record Demo**
```bash
# Show end-to-end workflow
./crisk init-local          # Layer 1, 2, 3 creation
./crisk incident create ... # Layer 3 incident
./crisk incident link ...   # CAUSED_BY edge
export OPENAI_API_KEY="..."
./crisk check --explain     # Phase 2 investigation
./crisk check --ai-mode     # AI Mode output
```

**3. Performance Validation**
```bash
# Run benchmarks
make test-performance
# Verify all targets met (<20ms, <50ms)
```

**4. User Testing**
```bash
# Test with real repositories
./crisk init-local  # On production repo
./crisk check       # On actual PR changes
# Collect feedback, iterate
```

---

## Questions?

**For architecture questions:**
- Check [spec.md](spec.md)
- Review [dev_docs/01-architecture/](dev_docs/01-architecture/)
- Consult [12-factor-agents-main/](dev_docs/12-factor-agents-main/)

**For implementation questions:**
- Follow [DEVELOPMENT_WORKFLOW.md](dev_docs/DEVELOPMENT_WORKFLOW.md)
- See examples in workflow doc
- Ask in team chat / create issue

**For task-specific questions:**
- Each task doc has troubleshooting section
- Check E2E test report for evidence
- Review existing code patterns

---

**Let's ship it! 🚀**
