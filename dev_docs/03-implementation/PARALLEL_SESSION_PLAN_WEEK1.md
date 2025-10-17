# Parallel Session Plan: Week 1 Core Functionality

**Created:** October 4, 2025
**Purpose:** Coordinate 3 parallel Claude Code sessions to complete Week 1 core functionality
**Reference:** [NEXT_STEPS.md](NEXT_STEPS.md), [status.md](status.md)

---

## Overview

This plan splits Week 1 implementation (Complete Core Functionality) into 3 **independent, non-overlapping** sessions that can run in parallel:

- **Session 4:** Git Integration Functions (`internal/git/`, git utilities)
- **Session 5:** Init Flow Orchestration (`cmd/crisk/init.go`, end-to-end graph construction)
- **Session 6:** Risk Calculation & Validation (`internal/risk/`, Phase 1 metrics)

**Key Design:**
- Each session owns **separate directories and files** (no file conflicts)
- Sessions communicate via **shared interfaces** (ingestion, risk calculation)
- Human verification at critical checkpoints (git parsing, init flow, risk accuracy)

**Estimated Duration:** 2-3 days parallel (vs 5-7 days sequential)

---

## File Ownership Map

### Session 4: Git Integration Functions

**Owns:**
- `internal/git/repo.go` (create new - git utilities)
- `internal/git/repo_test.go` (create new - unit tests)
- `test/integration/test_git_integration.sh` (create new)

**Modifies:**
- `cmd/crisk/init.go` (replace git stubs with real functions - ~20 lines)
- `cmd/crisk/check.go` (replace getChangedFiles stub - ~10 lines)

**Reads (no modification):**
- `internal/git/staged.go` (from Session 1, already exists)
- `dev_docs/03-implementation/NEXT_STEPS.md` (Task 1)

**Dependencies:** None (can start immediately)

---

### Session 5: Init Flow Orchestration

**Owns:**
- `cmd/crisk/init.go` (complete implementation - orchestration)
- `test/integration/test_init_e2e.sh` (create new)
- `internal/ingestion/orchestrator.go` (create new - optional)

**Modifies:**
- `internal/ingestion/processor.go` (add progress reporting - minor)

**Reads (no modification):**
- `internal/ingestion/clone.go` (Layer 1, already exists)
- `internal/ingestion/walker.go` (Layer 1, already exists)
- `internal/github/` (Layers 2-3, already exists)
- `internal/graph/` (graph construction, already exists)
- `dev_docs/03-implementation/integration_guides/cli_integration.md`
- `dev_docs/03-implementation/NEXT_STEPS.md` (Task 2)

**Dependencies:**
- **WAIT for Session 4** to implement git functions before using them in init.go
- Can start planning/reading immediately, implement after Session 4 Checkpoint 1

---

### Session 6: Risk Calculation & Validation

**Owns:**
- `internal/risk/phase1.go` (validate/enhance existing)
- `internal/risk/scoring.go` (validate/enhance existing)
- `internal/risk/calculator_test.go` (create new)
- `test/integration/test_check_e2e.sh` (create new)

**Modifies:**
- `cmd/crisk/check.go` (wire risk calculation to formatters - ~20 lines)

**Reads (no modification):**
- `internal/models/models.go` (data models from Session 2)
- `internal/output/` (formatters from Session 2)
- `dev_docs/01-architecture/risk_assessment_methodology.md`
- `dev_docs/03-implementation/NEXT_STEPS.md` (Task 3)

**Dependencies:** None (can start immediately, works with existing code)

---

## Shared Interface Definitions

### Interface 1: Git Functions (Defined by Session 4, used by Session 5)

**File:** `internal/git/repo.go`

```go
package git

// DetectGitRepo checks if current directory is a git repository
func DetectGitRepo() error

// ParseRepoURL extracts org and repo name from git remote URL
// Handles both HTTPS and SSH formats
func ParseRepoURL(remoteURL string) (org, repo string, err error)

// GetChangedFiles returns list of files changed in working directory
func GetChangedFiles() ([]string, error)

// GetRemoteURL returns the remote URL for the repository
func GetRemoteURL() (string, error)

// GetCurrentBranch returns the current branch name
func GetCurrentBranch() (string, error)
```

**Session 5 waits for Session 4 to create this**, then imports it.

---

### Interface 2: Ingestion Orchestration (Session 5 creates, Session 6 uses)

**File:** `cmd/crisk/init.go`

Session 5 implements the full init flow that Session 6 will use for testing:

```go
// runInit orchestrates the complete initialization:
// 1. Detect git repo
// 2. Clone repository (Layer 1)
// 3. Detect languages (GitHub API)
// 4. Parse AST (Tree-sitter)
// 5. Fetch GitHub data (Layer 2)
// 6. Construct graph (Layer 3)
// 7. Report statistics
func runInit(cmd *cobra.Command, args []string) error
```

Session 6 uses this to initialize test repositories for risk validation.

---

## Critical Checkpoints

### Checkpoint 1: Git Functions Ready (Session 4)

**Trigger:** Session 4 implements `internal/git/repo.go`

**Verification:**
```bash
# Build and test
go build ./internal/git
go test ./internal/git/... -v

# Verify functions work
cd /path/to/git/repo
go run scripts/test_git_functions.go
```

**Action:** Notify Session 5 that git functions are ready to use

**YOU ASK:** "✅ Git functions implemented at internal/git/repo.go. All tests pass (X/X). Should I notify Session 5?"

---

### Checkpoint 2: Init Flow Complete (Session 5)

**Trigger:** Session 5 completes `crisk init` orchestration

**Verification:**
```bash
# Build binary
go build -o bin/crisk ./cmd/crisk

# Test init
cd /path/to/test/repo
./bin/crisk init

# Verify output shows:
# ✅ Repository detected
# ⏳ Cloning... Parsing... Fetching... Building graph...
# ✅ Initialization complete! (Statistics)
```

**Action:** Session 6 can now use `crisk init` for test setup

**YOU ASK:** "✅ Init flow complete. End-to-end test passes. Statistics: [paste output]. Should I proceed?"

---

### Checkpoint 3: Risk Validation Complete (Session 6)

**Trigger:** Session 6 validates Phase 1 risk calculation

**Verification:**
```bash
# Test risk calculation
./bin/crisk check --quiet test_file.go
# Expected: ✅ LOW risk or ⚠️ MEDIUM risk

./bin/crisk check --explain test_file.go
# Expected: Full trace with metrics

./bin/crisk check --ai-mode test_file.go | jq '.risk.level'
# Expected: "LOW" / "MEDIUM" / "HIGH"
```

**Action:** All sessions complete, ready for integration

**YOU ASK:** "✅ Risk validation complete. All verbosity modes work. Risk levels accurate. Integration tests: [X/X pass]. Ready for final integration?"

---

### Checkpoint 4: Final Integration (All Sessions)

**Trigger:** All 3 sessions complete

**Verification:**
```bash
# Full end-to-end flow
cd /path/to/new/repo
crisk init                  # Session 5
crisk hook install          # Session 1 (already done)
git add .
git commit -m "test"        # Should trigger hook
                           # Uses Session 4 (git functions)
                           # Uses Session 6 (risk calculation)
                           # Uses Session 2 (verbosity formatters)

# Expected: Complete flow works
```

**Action:** Mark Week 1 complete

---

## Coordination Protocol

### Session Start Order

**Option A (All parallel):**
1. **Sessions 4 & 6 start immediately** → Independent work
2. **Session 5 starts immediately** → Reads docs, plans, waits for Checkpoint 1
3. **After Checkpoint 1** → Session 5 implements using git functions

**Option B (Staggered start - safer):**
1. **Session 4 starts first** → Implements git functions
2. **Wait 30-60 min for Checkpoint 1** ✅
3. **Sessions 5 & 6 start in parallel** → Session 5 uses git functions

**Recommended:** Option B (staggered) - ensures Session 5 doesn't wait long

### Communication via Checkpoints

- Each session implements a **checkpoint verification step**
- **Human reviews data** at each checkpoint before proceeding
- Sessions **ask for confirmation** before critical operations

### Merge Strategy

- Each session works on **separate files** (minimal conflicts)
- `cmd/crisk/init.go` and `cmd/crisk/check.go` are **shared files** (small changes per session)
- **Final integration:** Run `go build && go test ./...` to verify

---

## Success Criteria

### Functional
- [ ] `crisk init` works end-to-end (clone → parse → fetch → build graph)
- [ ] Git functions detect repo, parse URLs, get changed files
- [ ] Phase 1 risk calculation returns accurate levels (LOW/MEDIUM/HIGH)
- [ ] All verbosity modes show correct risk data
- [ ] Pre-commit hook blocks HIGH risk commits (using all components)

### Performance
- [ ] `crisk init` completes in <30s for small repos (~1K files)
- [ ] `crisk check` completes in <2s (cached)
- [ ] Git function overhead <100ms

### Quality
- [ ] 80%+ unit test coverage for new code
- [ ] Integration tests pass (git, init, check)
- [ ] No build errors, all packages compile

---

## Session Prompts

See separate files:
- `SESSION_4_PROMPT.md` - Git Integration Functions
- `SESSION_5_PROMPT.md` - Init Flow Orchestration
- `SESSION_6_PROMPT.md` - Risk Calculation & Validation

---

## Timeline Estimate

**Session 4:** 1-2 days (git utilities, parsing logic, tests)
**Session 5:** 1-2 days (init orchestration, progress reporting, e2e test)
**Session 6:** 1-2 days (risk validation, integration with formatters)

**Total (parallel):** 2-3 days (vs 5-7 days sequential)

**Speedup:** ~2.5x faster

---

## What Could Go Wrong

### Issue: Session 5 starts before Session 4 creates git functions
**Prevention:** Staggered start (Option B) or Session 5 waits for Checkpoint 1
**Recovery:** Session 5 creates temporary stubs, replaces with real functions after Checkpoint 1

### Issue: Git URL parsing fails for edge cases
**Prevention:** Session 4 tests multiple URL formats (HTTPS, SSH, Git protocol)
**Recovery:** Fallback to prompting user for org/repo name

### Issue: Init flow hangs on large repositories
**Prevention:** Session 5 adds timeout and progress reporting
**Recovery:** Add `--skip-layers` flag to debug which layer is slow

### Issue: Risk calculation returns unexpected levels
**Prevention:** Session 6 tests with known-risk files (zero tests, high coupling)
**Recovery:** Session 6 adjusts thresholds in spec.md and retests

---

## Quick Commands for Verification

**Check git functions (Checkpoint 1):**
```bash
go test ./internal/git/... -v
```

**Test init flow (Checkpoint 2):**
```bash
./bin/crisk init
# Verify statistics output
```

**Validate risk levels (Checkpoint 3):**
```bash
./bin/crisk check --quiet <file>
./bin/crisk check --explain <file>
./bin/crisk check --ai-mode <file> | jq '.risk.level'
```

**Full integration test (Checkpoint 4):**
```bash
./test/integration/test_week1_integration.sh
```

---

## After All Sessions Complete

**Update documentation:**
- [ ] Mark Week 1 tasks complete in [NEXT_STEPS.md](NEXT_STEPS.md)
- [ ] Update [status.md](status.md) with completion status
- [ ] Add Week 1 entry to [IMPLEMENTATION_LOG.md](IMPLEMENTATION_LOG.md)

**Celebrate!** 🎉
- Week 1 complete in 2-3 days (vs 5-7 days)
- Core functionality working end-to-end
- Ready for Week 2-3 (LLM Investigation)

---

## References

- [NEXT_STEPS.md](NEXT_STEPS.md) - Detailed Week 1 tasks
- [status.md](status.md) - Current implementation status
- [cli_integration.md](integration_guides/cli_integration.md) - Init flow guide
- [risk_assessment_methodology.md](../01-architecture/risk_assessment_methodology.md) - Risk calculation spec

---

**Created:** October 4, 2025
**Status:** Ready to execute
**Next Review:** After each checkpoint
