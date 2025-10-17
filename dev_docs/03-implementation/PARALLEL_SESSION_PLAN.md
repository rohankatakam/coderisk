# Parallel Session Plan: DX Foundation Implementation

**Created:** October 4, 2025
**Purpose:** Coordinate 3 parallel Claude Code sessions to implement Developer Experience (DX) Foundation phase
**Reference:** [phase_dx_foundation.md](phases/phase_dx_foundation.md)

---

## Overview

This plan splits the DX Foundation implementation into 3 **independent, non-overlapping** sessions that can run in parallel:

- **Session 1:** Pre-commit Hook & Git Integration (`cmd/crisk/hook.go`, `internal/git/`)
- **Session 2:** Adaptive Verbosity Levels 1-3 (`internal/output/`, formatters for quiet/standard/explain)
- **Session 3:** AI Mode Output & Prompts (`internal/output/ai_mode.go`, `internal/ai/`)

**Key Design:**
- Each session owns **separate directories and files** (no file conflicts)
- Sessions communicate via **shared interfaces** (defined below)
- Human verification at critical checkpoints (data validation, performance, integration)

---

## File Ownership Map

### Session 1: Pre-commit Hook & Git Integration

**Owns:**
- `cmd/crisk/hook.go` (create new)
- `internal/git/staged.go` (create new)
- `internal/git/repo.go` (create new)
- `internal/audit/override.go` (create new)
- `.coderisk.yml.example` (create new)
- `test/integration/test_pre_commit_hook.sh` (create new)

**Modifies:**
- `cmd/crisk/check.go` (add `--pre-commit` flag only, minimal changes)
- `.gitignore` (add `.coderisk/` directory)

**Reads (no modification):**
- `internal/output/formatter.go` (interface defined by Session 2)
- `dev_docs/03-implementation/integration_guides/ux_pre_commit_hook.md`

---

### Session 2: Adaptive Verbosity (Levels 1-3)

**Owns:**
- `internal/output/formatter.go` (create new - interface)
- `internal/output/quiet.go` (create new - Level 1)
- `internal/output/standard.go` (create new - Level 2)
- `internal/output/explain.go` (create new - Level 3)
- `internal/output/formatter_test.go` (create new)
- `internal/config/verbosity.go` (create new)
- `test/integration/test_verbosity.sh` (create new)

**Modifies:**
- `cmd/crisk/check.go` (add `--quiet`, `--explain` flags and wire formatters)

**Reads (no modification):**
- `internal/models/risk_result.go` (existing model)
- `dev_docs/03-implementation/integration_guides/ux_adaptive_verbosity.md`

---

### Session 3: AI Mode Output & Prompts (Level 4)

**Owns:**
- `internal/output/ai_mode.go` (create new - Level 4)
- `internal/ai/prompt_generator.go` (create new)
- `internal/ai/confidence.go` (create new)
- `internal/ai/templates.go` (create new)
- `schemas/ai-mode-v1.0.json` (create new)
- `test/integration/test_ai_mode.sh` (create new)

**Modifies:**
- `cmd/crisk/check.go` (add `--ai-mode` flag only)
- `internal/models/risk_result.go` (extend with AI-specific fields)

**Reads (no modification):**
- `internal/output/formatter.go` (interface from Session 2)
- `dev_docs/03-implementation/integration_guides/ux_ai_mode_output.md`

---

## Shared Interface Definitions

### Interface 1: Output Formatter (Defined by Session 2, used by Sessions 1 & 3)

**File:** `internal/output/formatter.go`

```go
package output

import (
    "io"
    "github.com/coderisk/coderisk-go/internal/models"
)

// Formatter defines output formatting interface
type Formatter interface {
    Format(result *models.RiskResult, w io.Writer) error
}

// VerbosityLevel determines output detail
type VerbosityLevel int

const (
    VerbosityQuiet VerbosityLevel = iota   // Level 1
    VerbosityStandard                      // Level 2
    VerbosityExplain                       // Level 3
    VerbosityAIMode                        // Level 4
)

// NewFormatter creates appropriate formatter based on level
func NewFormatter(level VerbosityLevel) Formatter {
    switch level {
    case VerbosityQuiet:
        return &QuietFormatter{}
    case VerbosityStandard:
        return &StandardFormatter{}
    case VerbosityExplain:
        return &ExplainFormatter{}
    case VerbosityAIMode:
        return &AIFormatter{}
    default:
        return &StandardFormatter{}
    }
}
```

**Session 2 creates this file first**, then Sessions 1 & 3 import it.

---

### Interface 2: Risk Result Extensions (Session 3 extends, all sessions use)

**File:** `internal/models/risk_result.go` (existing, Session 3 extends)

**Session 3 adds these fields:**
```go
type Issue struct {
    // ... existing fields ...

    // AI Mode specific (Session 3 adds)
    AutoFixable       bool     `json:"auto_fixable"`
    FixType           string   `json:"fix_type"`
    FixConfidence     float64  `json:"fix_confidence"`
    AIPromptTemplate  string   `json:"ai_prompt_template"`
    ExpectedFiles     []string `json:"expected_files"`
    EstimatedLines    int      `json:"estimated_lines"`
}

type RiskResult struct {
    // ... existing fields ...

    // Commit control (Session 3 adds)
    ShouldBlock                   bool   `json:"should_block_commit"`
    BlockReason                   string `json:"block_reason"`
    OverrideAllowed               bool   `json:"override_allowed"`
    OverrideRequiresJustification bool   `json:"override_requires_justification"`
}
```

**All sessions** can read this file, only **Session 3** modifies it.

---

## Critical Checkpoints (Human Verification)

### Checkpoint 1: Interface Definition (After Session 2 starts)
**When:** Session 2 creates `internal/output/formatter.go`
**Verify:**
- [ ] Interface compiles successfully
- [ ] VerbosityLevel enum is complete (4 levels)
- [ ] NewFormatter() factory pattern works

**Action:** Once verified, notify Sessions 1 & 3 to proceed with imports

---

### Checkpoint 2: Model Extensions (After Session 3 modifies)
**When:** Session 3 extends `internal/models/risk_result.go`
**Verify:**
- [ ] New fields don't conflict with existing fields
- [ ] JSON tags are correct
- [ ] All sessions can still build

**Action:** Once verified, Sessions 1 & 2 can use extended model

---

### Checkpoint 3: CLI Flag Integration (After all sessions modify check.go)
**When:** All sessions have added their flags to `cmd/crisk/check.go`
**Verify:**
- [ ] Flags are mutually exclusive (--quiet, --explain, --ai-mode)
- [ ] Pre-commit mode works (`--pre-commit` from Session 1)
- [ ] No merge conflicts in check.go

**Action:** Run `go build ./cmd/crisk` to verify

---

### Checkpoint 4: End-to-End Integration (Final)
**When:** All sessions complete
**Verify:**
- [ ] `crisk hook install` works (Session 1)
- [ ] Hook triggers on commit (Session 1)
- [ ] Quiet mode outputs correctly (Session 2)
- [ ] Standard mode shows issues (Session 2)
- [ ] Explain mode shows trace (Session 2)
- [ ] AI mode outputs valid JSON (Session 3)
- [ ] Performance: <2s cached, <5s cold

**Action:** Run integration test suite

---

## Coordination Protocol

### Session Start Order
1. **Session 2 starts first** → Creates `internal/output/formatter.go` interface
2. **Wait for Checkpoint 1** ✅
3. **Sessions 1 & 3 start in parallel** → Import formatter interface

### Communication via Checkpoints
- Each session implements a **checkpoint verification step**
- **Human reviews data** at each checkpoint before proceeding
- Sessions **ask for confirmation** before critical operations

### Merge Strategy
- Each session works on **separate files** (no merge conflicts)
- `cmd/crisk/check.go` is the **only shared file** (minimal changes per session)
- **Final integration:** Run `go build` to verify all pieces work together

---

## Success Criteria

### Functional
- [ ] Pre-commit hook installs and runs automatically
- [ ] All 4 verbosity levels work correctly
- [ ] AI Mode outputs valid JSON matching schema v1.0
- [ ] Overrides are logged to audit trail

### Performance
- [ ] Pre-commit check <2s (p50, cached)
- [ ] Pre-commit check <5s (p95, cold)
- [ ] AI Mode generation overhead <200ms

### Quality
- [ ] 80%+ unit test coverage for all new code
- [ ] Integration tests pass for each session
- [ ] End-to-end test passes

---

## Session Prompts

See separate files:
- `SESSION_1_PROMPT.md` - Pre-commit Hook & Git Integration
- `SESSION_2_PROMPT.md` - Adaptive Verbosity (Levels 1-3)
- `SESSION_3_PROMPT.md` - AI Mode Output & Prompts (Level 4)

---

## Timeline Estimate

**Session 1:** 3-4 days (pre-commit hook, git utils, audit logging)
**Session 2:** 3-4 days (3 formatters, CLI integration, tests)
**Session 3:** 4-5 days (AI mode formatter, prompt generation, confidence scoring)

**Total (parallel):** 4-5 days (vs 10-12 days sequential)

---

## References

- [phase_dx_foundation.md](phases/phase_dx_foundation.md) - Full phase plan
- [ux_pre_commit_hook.md](integration_guides/ux_pre_commit_hook.md) - Session 1 guide
- [ux_adaptive_verbosity.md](integration_guides/ux_adaptive_verbosity.md) - Session 2 guide
- [ux_ai_mode_output.md](integration_guides/ux_ai_mode_output.md) - Session 3 guide
- [developer_experience.md](../00-product/developer_experience.md) - UX requirements
