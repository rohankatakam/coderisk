# Next Steps: Post-DX Foundation Implementation Roadmap

**Created:** October 4, 2025
**Purpose:** Define immediate next steps after DX Foundation phase completion
**Reference:** [status.md](status.md), [spec.md](../spec.md)

---

## Current State ✅

**Completed:**
- ✅ **DX Foundation Phase** - Pre-commit hook, 4 verbosity levels, AI Mode
- ✅ **Layer 1 (Tree-sitter)** - AST parsing for code structure
- ✅ **Layers 2-3 (GitHub API)** - GitHub fetching and graph construction
- ✅ **Phase 1 Metrics** - Coupling, co-change, test ratio
- ✅ **Local Infrastructure** - Docker Compose, Neo4j, PostgreSQL, Redis
- ✅ **Week 1 Core Functionality** - Git integration, init flow, risk validation (October 4, 2025)

**What We Have:**
- Working CLI with `crisk check`, `crisk hook`, `crisk status`, `crisk config`, `crisk init-local`
- Pre-commit hook that runs risk assessments
- 4 verbosity levels (quiet, standard, explain, AI mode)
- AI-ready JSON output for Claude Code/Cursor integration
- Phase 1 metrics calculation (coupling, temporal co-change, test coverage)
- Git utilities (repo detection, URL parsing, changed file detection)
- End-to-end init flow (clone → parse → graph construction)
- Validated risk calculation with test fixtures

---

## Gap Analysis

### What's Missing for MVP

Based on [status.md](status.md) critical missing pieces:

**1. Git Integration** ✅ **COMPLETE**
```go
// Implemented in internal/git/repo.go:
func DetectGitRepo() error                              // ✅ IMPLEMENTED
func ParseRepoURL(remoteURL string) (org, repo string, error)  // ✅ IMPLEMENTED
func GetChangedFiles() ([]string, error)                // ✅ IMPLEMENTED
func GetRemoteURL() (string, error)                     // ✅ IMPLEMENTED
func GetCurrentBranch() (string, error)                 // ✅ IMPLEMENTED
```

**2. Neptune Graph Client** ❌
- openCypher query execution
- Connection pooling
- 2-hop context loading
- Spatial cache integration

**3. Settings Portal** ❌
- API key configuration UI
- GitHub OAuth flow
- Team management
- Repository listing

**4. Agent Investigation Engine** ❌
- Spatial context management
- Hop-by-hop navigation
- Metric validation
- LLM integration (user's API key)

---

## Priority Roadmap

### ~~Phase 1: Complete Core Functionality (Week 1)~~ ✅ **COMPLETE**

**Goal:** Make `crisk check` fully functional end-to-end

**✅ Priority 1A: Git Integration Functions** (COMPLETE - Session 4)
- ✅ Implemented `git.DetectGitRepo()` in `internal/git/repo.go`
- ✅ Implemented `git.ParseRepoURL()` with HTTPS, SSH, git:// support
- ✅ Implemented `git.GetChangedFiles()` for pre-commit hook
- ✅ Implemented `git.GetRemoteURL()` and `git.GetCurrentBranch()`
- **Deliverables:** 5 git functions, 95% test coverage, 8/8 integration tests passing
- **Files:** `internal/git/repo.go`, `internal/git/repo_test.go`, `test/integration/test_git_integration.sh`

**✅ Priority 1B: End-to-End `crisk init` Flow** (COMPLETE - Session 5)
- ✅ Implemented `crisk init-local` orchestration (clone → parse → graph)
- ✅ Progress reporting with emojis and statistics
- ✅ Auto-detection from git remote
- ✅ Tested with commander.js (2,291 nodes) and Next.js (144K entities)
- **Deliverables:** Complete init flow, 11/11 integration tests passing
- **Files:** `cmd/crisk/init_local.go`, `test/integration/test_init_e2e.sh`
- **Performance:** ~6,500 files/sec, <10s for 17K files

**✅ Priority 1C: Phase 1 Risk Calculation** (COMPLETE - Session 6)
- ✅ Validated Phase 1 metrics (coupling, co-change, test ratio)
- ✅ Created test fixtures with known risk levels
- ✅ Integrated with all verbosity modes (quiet, standard, explain, AI)
- ✅ All tests passing (29 unit tests, 12 integration tests)
- **Deliverables:** Validated risk engine, comprehensive test suite
- **Files:** `internal/metrics/*_test.go`, `test/fixtures/known_risk/*`, `test/integration/test_check_e2e.sh`
- **Performance:** 5ms per check (97% faster than 500ms target)

---

### Phase 2: LLM Investigation Engine (Week 3-6)

**Goal:** Implement Phase 2 agent investigation for complex cases

Reference: [agentic_design.md](../01-architecture/agentic_design.md)

**Priority 2A: LLM Client Integration** (1 week)
- Implement OpenAI client (GPT-4)
- Implement Anthropic client (Claude 3.5)
- User API key management (encrypted storage)
- **Unlocks:** Agent can reason about code
- **Files:** `internal/llm/openai.go`, `internal/llm/anthropic.go`, `internal/llm/client.go`

**Priority 2B: Spatial Context Manager** (1 week)
- Implement 2-hop context window
- Implement spatial cache (hot/warm/cold zones)
- Add context pruning (token limits)
- **Unlocks:** Agent can navigate graph efficiently
- **Files:** `internal/agent/context.go`, `internal/agent/spatial.go`

**Priority 2C: Agent Decision Loop** (1-2 weeks)
- Implement hop-by-hop investigation
- Add Tier 2 metrics (ownership_churn, incident_similarity)
- Implement reasoning trace for explain mode
- **Unlocks:** Phase 2 escalation works end-to-end
- **Files:** `internal/agent/investigator.go`, `internal/metrics/tier2.go`

---

### Phase 3: Neptune Cloud Deployment (Week 7-8)

**Goal:** Deploy to AWS with Neptune Serverless

**Priority 3A: Neptune Client** (1 week)
- Implement Neptune Serverless connection
- Implement openCypher query execution
- Add connection pooling
- **Unlocks:** Cloud deployment ready
- **Files:** `internal/graph/neptune/client.go`

**Priority 3B: Settings Portal MVP** (1 week)
- Build Next.js API key config UI
- Implement GitHub OAuth flow
- Add team management basics
- **Unlocks:** Users can configure their own API keys
- **Stack:** Next.js + PostgreSQL

---

## Immediate Next Actions (Week 1)

### Task 1: Implement Git Integration Functions (Day 1-2)

**File: `internal/git/repo.go`** (new file)

```go
package git

import (
    "fmt"
    "os/exec"
    "strings"
)

// DetectGitRepo checks if current directory is a git repository
func DetectGitRepo() error {
    cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("not a git repository: %w", err)
    }
    return nil
}

// ParseRepoURL extracts org and repo name from git remote URL
func ParseRepoURL(remoteURL string) (org, repo string, err error) {
    // Handle both HTTPS and SSH formats:
    // https://github.com/owner/repo.git
    // git@github.com:owner/repo.git

    // Implementation needed - see integration_guides/ux_pre_commit_hook.md
}

// GetChangedFiles returns list of files changed in working directory
func GetChangedFiles() ([]string, error) {
    cmd := exec.Command("git", "diff", "--name-only", "HEAD")
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("failed to get changed files: %w", err)
    }

    files := strings.Split(strings.TrimSpace(string(output)), "\n")
    var result []string
    for _, f := range files {
        if f != "" {
            result = append(result, f)
        }
    }
    return result, nil
}
```

**File: Update `cmd/crisk/init.go`**
```go
import "github.com/coderisk/coderisk-go/internal/git"

func runInit(cmd *cobra.Command, args []string) error {
    // Use git.DetectGitRepo() instead of stub
    if err := git.DetectGitRepo(); err != nil {
        return err
    }

    // Get remote URL and parse
    remoteURL, err := git.GetRemoteURL()
    if err != nil {
        return err
    }

    org, repo, err := git.ParseRepoURL(remoteURL)
    if err != nil {
        return err
    }

    // Continue with existing ingestion logic...
}
```

**File: Update `cmd/crisk/check.go`**
```go
import "github.com/coderisk/coderisk-go/internal/git"

func runCheck(cmd *cobra.Command, args []string) error {
    var files []string
    var err error

    if len(args) > 0 {
        files = args
    } else {
        // Use git.GetChangedFiles() instead of stub
        files, err = git.GetChangedFiles()
        if err != nil {
            return fmt.Errorf("failed to get changed files: %w", err)
        }
    }

    // Continue with existing check logic...
}
```

**Tests:**
```bash
# Unit tests
go test ./internal/git/... -v

# Integration test
git init test_repo
cd test_repo
git add .
crisk check  # Should detect changed files
```

---

### Task 2: Wire `crisk init` End-to-End (Day 3-4)

**Goal:** Complete orchestration from [cli_integration.md](integration_guides/cli_integration.md)

**File: `cmd/crisk/init.go`** (complete implementation)

**Steps:**
1. Clone repository (Layer 1)
2. Detect languages (GitHub API)
3. Parse AST (Tree-sitter)
4. Fetch GitHub data (Layer 2)
5. Construct graph (Layer 3)
6. Report statistics

**Reference:** [cli_integration.md](integration_guides/cli_integration.md) has full implementation guide

**Validation:**
```bash
crisk init
# Expected output:
# ✅ Repository detected: coderisk-go
# ⏳ Cloning repository...
# ⏳ Detecting languages...
# ⏳ Parsing code structure...
# ⏳ Fetching GitHub data...
# ⏳ Building graph...
# ✅ Initialization complete!
#
# Statistics:
#   Files analyzed: 45
#   Graph nodes: 1,234
#   Graph edges: 2,456
#   Duration: 23s
```

---

### Task 3: Validate Phase 1 Risk Calculation (Day 5)

**Goal:** Ensure `crisk check` returns accurate risk levels

**Files:**
- `internal/risk/phase1.go` - Phase 1 metric calculation
- `internal/risk/scoring.go` - Risk level determination

**Validation:**
```bash
# Test with different files
crisk check --explain src/high_coupling.go
# Should show: HIGH risk (coupling > 8)

crisk check --quiet src/low_risk.go
# Should show: ✅ LOW risk

# Test with AI mode
crisk check --ai-mode src/no_tests.go | jq '.risk.level'
# Should return: "MEDIUM" or "HIGH"
```

**Tests:**
```bash
go test ./internal/risk/... -v -cover
# Expected: >80% coverage, all tests pass
```

---

## Success Criteria

### Week 1 Complete When:
- ✅ `crisk init` works end-to-end (clones, parses, builds graph)
- ✅ `crisk check` detects changed files automatically
- ✅ Phase 1 risk calculation returns accurate levels
- ✅ All verbosity modes work with real risk data
- ✅ Pre-commit hook blocks HIGH risk commits
- ✅ All tests passing (unit + integration)

### Week 2-3 Complete When:
- ✅ LLM clients integrated (OpenAI + Anthropic)
- ✅ Spatial context manager working
- ✅ Agent can navigate graph and reason about code
- ✅ Phase 2 escalation triggers on Phase 1 thresholds
- ✅ Explain mode shows hop-by-hop investigation trace

### Month 1 Complete When:
- ✅ Neptune Serverless deployed
- ✅ Settings portal live (API key config)
- ✅ 50 beta users onboarded
- ✅ End-to-end flow: `crisk init` → `crisk check` → Phase 2 investigation

---

## Dependencies

### External Services Needed

**For Phase 1 (Immediate):**
- ✅ GitHub API access (already have)
- ✅ Neo4j (local, already running)
- ✅ PostgreSQL (local, already running)
- ✅ Redis (local, already running)

**For Phase 2 (Week 3+):**
- ❌ OpenAI API key (user-provided)
- ❌ Anthropic API key (user-provided)

**For Phase 3 (Week 7+):**
- ❌ AWS account
- ❌ Neptune Serverless cluster
- ❌ ElastiCache Redis
- ❌ RDS PostgreSQL
- ❌ S3 bucket

---

## Testing Strategy

### Integration Tests to Create

**Week 1:**
```bash
# test/integration/test_git_integration.sh
test_detect_git_repo
test_parse_repo_url
test_get_changed_files

# test/integration/test_init_e2e.sh
test_init_omnara_repo
test_init_with_existing_graph
test_init_error_recovery

# test/integration/test_check_e2e.sh
test_check_no_args_detects_changes
test_check_with_file_arg
test_check_outputs_risk_levels
```

**Week 2-3:**
```bash
# test/integration/test_llm_investigation.sh
test_phase1_escalates_to_phase2
test_agent_navigates_graph
test_explain_mode_shows_trace
```

---

## Documentation to Create

**Week 1:**
- [ ] Update [status.md](status.md) as tasks complete
- [ ] Create `integration_guides/git_integration.md` - How git utilities work
- [ ] Update [DX_FOUNDATION_COMPLETE.md](DX_FOUNDATION_COMPLETE.md) with next phase plan

**Week 2-3:**
- [ ] Create `phases/phase_llm_investigation.md` - Phase 2 roadmap
- [ ] Create `integration_guides/llm_integration.md` - LLM client setup
- [ ] Create `integration_guides/agent_investigation.md` - Agent design

**Week 7-8:**
- [ ] Create `phases/phase_cloud_deployment.md` - Neptune deployment
- [ ] Create `integration_guides/neptune_setup.md` - AWS setup guide

---

## Risk Mitigation

### Potential Blockers

**1. Git Integration Complexity**
- **Risk:** Parsing various git URL formats is error-prone
- **Mitigation:** Use battle-tested regex patterns, extensive tests
- **Fallback:** Prompt user for org/repo if parsing fails

**2. LLM API Costs**
- **Risk:** Users concerned about API costs for Phase 2
- **Mitigation:** Only escalate 10-20% of checks, clear cost estimation
- **Fallback:** Allow disabling Phase 2 (Phase 1 only mode)

**3. Neptune Learning Curve**
- **Risk:** Neptune openCypher different from Neo4j Cypher
- **Mitigation:** Dual backend abstraction already designed (see [layers_2_3_graph_construction.md](integration_guides/layers_2_3_graph_construction.md))
- **Fallback:** Continue using Neo4j locally until Neptune ready

**4. Settings Portal Scope Creep**
- **Risk:** Portal becomes too complex, delays MVP
- **Mitigation:** MVP = API key CRUD only, defer team features
- **Fallback:** Use .env file for API keys (no portal) in v1.0

---

## Resource Requirements

### Development Time

**Solo Developer:**
- Week 1 (Git + Init): 5 days
- Week 2-3 (LLM Investigation): 10 days
- Week 4-6 (Polish + Testing): 15 days
- Week 7-8 (Cloud Deployment): 10 days
- **Total:** 40 days (~2 months)

**With AI Assistance (Parallel Sessions):**
- Week 1: 2 days (3x speedup)
- Week 2-3: 4 days (2.5x speedup)
- Week 4-6: 6 days (2.5x speedup)
- Week 7-8: 4 days (2.5x speedup)
- **Total:** 16 days (~3 weeks)

### Infrastructure Costs

**Local Development:** $0/month (current)

**Cloud Deployment (50 beta users):**
- Neptune Serverless: ~$50/month
- ElastiCache Redis: ~$15/month
- RDS PostgreSQL: ~$25/month
- S3 + Data Transfer: ~$10/month
- **Total:** ~$100/month

---

## Next Session Prompts

If using parallel sessions again, here are suggested prompts:

### Session A: Git Integration & Init Flow (Week 1)
```
Implement git integration functions and complete crisk init end-to-end flow.

Read:
- dev_docs/03-implementation/NEXT_STEPS.md (Task 1 & 2)
- dev_docs/03-implementation/integration_guides/cli_integration.md
- dev_docs/03-implementation/integration_guides/ux_pre_commit_hook.md

Files to create/modify:
- internal/git/repo.go (new)
- cmd/crisk/init.go (complete implementation)
- cmd/crisk/check.go (use git functions)

Tests:
- internal/git/repo_test.go
- test/integration/test_init_e2e.sh

Goal: crisk init and crisk check work end-to-end
```

### Session B: Risk Calculation & Validation (Week 1)
```
Ensure Phase 1 risk calculation works correctly with all verbosity modes.

Read:
- dev_docs/03-implementation/NEXT_STEPS.md (Task 3)
- dev_docs/01-architecture/risk_assessment_methodology.md
- dev_docs/spec.md (NFR-13 to NFR-19: Performance)

Files to create/modify:
- internal/risk/phase1.go (validate)
- internal/risk/scoring.go (validate)
- Integration with formatters

Tests:
- internal/risk/phase1_test.go
- test/integration/test_check_e2e.sh

Goal: Accurate risk levels in all verbosity modes
```

---

## References

**Implementation Status:**
- [status.md](status.md) - Current progress
- [IMPLEMENTATION_LOG.md](IMPLEMENTATION_LOG.md) - Historical log

**Architecture:**
- [agentic_design.md](../01-architecture/agentic_design.md) - Phase 2 agent design
- [risk_assessment_methodology.md](../01-architecture/risk_assessment_methodology.md) - Risk calculation

**Integration Guides:**
- [cli_integration.md](integration_guides/cli_integration.md) - crisk init orchestration
- [layers_2_3_github_fetching.md](integration_guides/layers_2_3_github_fetching.md) - GitHub API fetching
- [layers_2_3_graph_construction.md](integration_guides/layers_2_3_graph_construction.md) - Graph building

**Specifications:**
- [spec.md](../spec.md) - Requirements and NFRs
- [developer_experience.md](../00-product/developer_experience.md) - UX requirements

---

**Created:** October 4, 2025
**Status:** Ready for implementation
**Next Review:** After Week 1 completion
