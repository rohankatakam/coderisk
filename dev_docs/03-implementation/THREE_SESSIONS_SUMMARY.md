# Three Parallel Sessions: Quick Reference

**Created:** October 4, 2025
**Purpose:** High-level summary for managing 3 parallel Claude Code sessions

---

## Session Overview

| Session | Focus | Duration | Files Created | Critical Path |
|---------|-------|----------|---------------|---------------|
| **Session 1** | Pre-commit Hook & Git Integration | 3-4 days | `cmd/crisk/hook.go`, `internal/git/`, `internal/audit/` | Can start immediately |
| **Session 2** | Adaptive Verbosity (Levels 1-3) | 3-4 days | `internal/output/formatter.go`, formatters | **START FIRST** (creates interface) |
| **Session 3** | AI Mode (Level 4) & Prompts | 4-5 days | `internal/output/ai_mode.go`, `internal/ai/` | Waits for Session 2 interface |

**Total Time (Parallel):** 4-5 days (vs 10-12 days sequential)

---

## Start Order

1. **START Session 2 FIRST** → Creates `internal/output/formatter.go` interface
2. **WAIT for Checkpoint 1** → Formatter interface created ✅
3. **START Sessions 1 & 3 in parallel** → Import formatter interface

---

## File Ownership (No Conflicts)

### Session 1 Owns:
- ✅ `cmd/crisk/hook.go` (NEW)
- ✅ `internal/git/staged.go` (NEW)
- ✅ `internal/git/repo.go` (NEW)
- ✅ `internal/audit/override.go` (NEW)
- ⚠️ `cmd/crisk/check.go` (adds `--pre-commit` flag only)

### Session 2 Owns:
- ✅ `internal/output/formatter.go` (NEW - **CRITICAL, create first!**)
- ✅ `internal/output/quiet.go` (NEW)
- ✅ `internal/output/standard.go` (NEW)
- ✅ `internal/output/explain.go` (NEW)
- ✅ `internal/config/verbosity.go` (NEW)
- ⚠️ `cmd/crisk/check.go` (adds `--quiet`, `--explain` flags)

### Session 3 Owns:
- ✅ `internal/output/ai_mode.go` (NEW)
- ✅ `internal/ai/prompt_generator.go` (NEW)
- ✅ `internal/ai/confidence.go` (NEW)
- ✅ `internal/ai/templates.go` (NEW)
- ✅ `schemas/ai-mode-v1.0.json` (NEW)
- ⚠️ `cmd/crisk/check.go` (adds `--ai-mode` flag only)
- ⚠️ `internal/models/risk_result.go` (extends with AI fields)

**Shared File:** `cmd/crisk/check.go` - Each session adds ONE flag (~10 lines each)

---

## Critical Checkpoints

### Checkpoint 1: Formatter Interface Created (Session 2)
**Trigger:** Session 2 creates `internal/output/formatter.go`

**Verification:**
```bash
ls -la internal/output/formatter.go
go build ./internal/output
```

**Action:** Notify Sessions 1 & 3 they can proceed

---

### Checkpoint 2: Model Extensions (Session 3)
**Trigger:** Session 3 extends `internal/models/risk_result.go`

**Verification:**
```bash
go build ./internal/models
# Verify no conflicts with existing fields
```

**Action:** Sessions 1 & 2 can use extended model

---

### Checkpoint 3: CLI Integration (All Sessions)
**Trigger:** All sessions have modified `cmd/crisk/check.go`

**Verification:**
```bash
go build ./cmd/crisk
./bin/crisk check --help  # Verify all flags present
```

**Action:** Run integration tests

---

### Checkpoint 4: End-to-End (Final)
**Trigger:** All sessions complete

**Verification:**
```bash
# Session 1
./bin/crisk hook install
git commit  # Test hook

# Session 2
./bin/crisk check --quiet <file>
./bin/crisk check <file>
./bin/crisk check --explain <file>

# Session 3
./bin/crisk check --ai-mode <file> | jq .
ajv validate -s schemas/ai-mode-v1.0.json -d output.json
```

**Action:** Mark phase complete

---

## Communication Protocol

### Human's Role (You)

**At each checkpoint:**
1. **Review data** - Check test results, output examples, performance
2. **Verify no conflicts** - Ensure file ownership boundaries respected
3. **Give approval** - Confirm session can proceed to next step

**Critical decisions requiring your input:**
- Interface design validation (Checkpoint 1)
- Model extension approval (Checkpoint 2)
- Output format validation (each session's final step)
- Performance verification (<2s cached, <5s cold, <200ms AI overhead)

### Sessions' Role

**Each session will:**
1. **ASK before proceeding** at critical points
2. **WAIT for your approval** before moving forward
3. **REPORT results** (test output, performance metrics, examples)
4. **STAY in file boundaries** (no unauthorized file edits)

---

## Session Prompts (Full Instructions)

**Detailed prompts with step-by-step plans:**
- [SESSION_1_PROMPT.md](SESSION_1_PROMPT.md) - Pre-commit Hook & Git Integration
- [SESSION_2_PROMPT.md](SESSION_2_PROMPT.md) - Adaptive Verbosity (Levels 1-3)
- [SESSION_3_PROMPT.md](SESSION_3_PROMPT.md) - AI Mode (Level 4) & Prompts

**Coordination plan:**
- [PARALLEL_SESSION_PLAN.md](PARALLEL_SESSION_PLAN.md) - Full file ownership map and checkpoints

---

## Success Criteria (All Sessions)

### Functional
- [ ] Pre-commit hook installs and runs automatically (Session 1)
- [ ] 4 verbosity levels work correctly (Sessions 2 & 3)
- [ ] AI Mode outputs valid JSON with prompts (Session 3)
- [ ] All flags are mutually exclusive

### Performance
- [ ] Pre-commit check <2s cached, <5s cold (Session 1)
- [ ] Formatting overhead <10ms (Session 2)
- [ ] AI Mode overhead <200ms (Session 3)

### Quality
- [ ] 80%+ unit test coverage (all sessions)
- [ ] Integration tests pass (all sessions)
- [ ] No file conflicts or merge issues

---

## Running the Sessions

### Step 1: Open 3 Claude Code Windows

**Window 1 (Session 2 - Start First!):**
```bash
cd /Users/rohankatakam/Documents/brain/coderisk-go
# Paste prompt from SESSION_2_PROMPT.md
```

**Window 2 (Session 1 - Wait for Checkpoint 1):**
```bash
cd /Users/rohankatakam/Documents/brain/coderisk-go
# Paste prompt from SESSION_1_PROMPT.md
```

**Window 3 (Session 3 - Wait for Checkpoint 1):**
```bash
cd /Users/rohankatakam/Documents/brain/coderisk-go
# Paste prompt from SESSION_3_PROMPT.md
```

### Step 2: Monitor Progress

**Session 2 will ask:**
> "✅ Formatter interface created. Should notify other sessions?"

**Your response:** ✅ Approve → Tell Sessions 1 & 3 to proceed

### Step 3: Manage Checkpoints

Each session will pause at critical points and ask for approval:
- Session 1: After git utils, after hook install, before integration test
- Session 2: After interface, after formatters, before integration test
- Session 3: After waiting for interface, after model extension, before JSON schema

**Your job:** Review → Approve → Move to next checkpoint

### Step 4: Final Integration

When all sessions complete:
1. Verify all tests pass
2. Run end-to-end integration
3. Check performance metrics
4. Update status.md

---

## What Could Go Wrong

### Issue: Sessions modify same file simultaneously
**Prevention:** Strict file ownership map (see PARALLEL_SESSION_PLAN.md)
**Recovery:** Each session only modifies assigned files; `check.go` has minimal changes per session

### Issue: Session 3 starts before Session 2 creates interface
**Prevention:** Start order (Session 2 first, wait for Checkpoint 1)
**Recovery:** Session 3 will ask "Has Session 2 created formatter.go?" before proceeding

### Issue: Build breaks due to dependency
**Prevention:** Checkpoints verify builds at each step
**Recovery:** Session reports error, you review, provide guidance

### Issue: Test failures
**Prevention:** Each session runs tests before proceeding
**Recovery:** Session waits for your review before continuing

---

## Expected Timeline

**Day 1:**
- Session 2: Create interface, implement Levels 1-2 ✅
- Session 1: Git utils, hook install ✅
- Session 3: Wait for interface, then extend model ✅

**Day 2:**
- Session 2: Implement Level 3, CLI integration ✅
- Session 1: CLI integration, override tracking ✅
- Session 3: AI Mode formatter, prompt generation ✅

**Day 3:**
- Session 2: Unit tests, integration tests ✅
- Session 1: Integration tests, config templates ✅
- Session 3: Confidence scoring, templates ✅

**Day 4:**
- Session 2: Final validation ✅
- Session 1: Final validation ✅
- Session 3: JSON schema, integration tests ✅

**Day 5 (buffer):**
- All sessions: Final integration, performance tuning
- You: Review, approve, mark phase complete

---

## Quick Commands for Verification

**Check interface exists (Checkpoint 1):**
```bash
ls -la internal/output/formatter.go
```

**Verify builds (anytime):**
```bash
go build ./internal/output ./internal/git ./internal/ai ./internal/audit
go build ./cmd/crisk
```

**Run all tests (before final approval):**
```bash
go test ./internal/output/... -v
go test ./internal/git/... -v
go test ./internal/ai/... -v
./test/integration/test_pre_commit_hook.sh
./test/integration/test_verbosity.sh
./test/integration/test_ai_mode.sh
```

**Validate AI Mode JSON:**
```bash
./bin/crisk check --ai-mode <file> | jq . > output.json
ajv validate -s schemas/ai-mode-v1.0.json -d output.json
```

---

## After All Sessions Complete

**Update documentation:**
- [ ] Mark DX Foundation phase as ✅ Complete in status.md
- [ ] Update IMPLEMENTATION_LOG.md with phase completion notes
- [ ] Add any lessons learned

**Celebrate!** 🎉
- 3 parallel sessions completed in 4-5 days
- Pre-commit hook working
- 4 verbosity levels implemented
- AI Mode ready for Claude Code/Cursor integration

**Next phase:** LLM Investigation (Phase 2) - See [agentic_design.md](../01-architecture/agentic_design.md)

---

## References

- [phase_dx_foundation.md](phases/phase_dx_foundation.md) - Full phase plan
- [PARALLEL_SESSION_PLAN.md](PARALLEL_SESSION_PLAN.md) - Detailed coordination plan
- [developer_experience.md](../00-product/developer_experience.md) - UX requirements
- [DOCUMENTATION_WORKFLOW.md](../DOCUMENTATION_WORKFLOW.md) - Documentation guidelines
