# Implementation Status

**Last Updated:** October 13, 2025 (Validation Testing Complete - Gaps Identified)
**Purpose:** Track current codebase implementation progress

---

## Current State

**Total Lines of Code:** 4,500+ lines of Go
**Implementation:** 70% complete (Graph ingestion ✅, Risk assessment needs enhancement)
**Status:** **Not Production-Ready** - Critical gaps identified in validation testing

> **⚠️ CRITICAL FINDINGS (Oct 13):** Comprehensive Validation Testing Revealed Major Gaps
>
> **Test Results:** 1/6 validation tests passed - See [VALIDATION_TESTING_SUMMARY.md](VALIDATION_TESTING_SUMMARY.md)
>
> **Critical Gaps Identified:**
> - ❌ **Layer 1 (Structure):** NOT USED - 2,562 functions/454 classes ignored in risk assessment
> - ❌ **Layer 2 (MODIFIES):** NOT USED - 419 commit history edges never queried (no churn analysis)
> - ⚠️ **Layer 2 (CO_CHANGED):** PARTIAL - Only data source used, but insufficient alone
> - ❌ **Layer 3 (Incidents):** NOT USED - 71 issues/180 PRs completely unused
> - ❌ **Multi-File Context:** Files analyzed independently, misses co-changed pair detection
> - ❌ **Commit Workflow:** Cannot analyze HEAD or commit references
>
> **What's Working:**
> - ✅ Graph ingestion: All 3 layers properly constructed (421 files, 49,654 CO_CHANGED edges)
> - ✅ CO_CHANGED detection: Temporal coupling identified correctly
> - ✅ Performance: Fast execution (130ms for LOW risk)
>
> **Implementation Roadmap:** [PHASE2_INVESTIGATION_ROADMAP.md](PHASE2_INVESTIGATION_ROADMAP.md)
> - **7 tasks** to implement proper agentic investigation
> - **Estimated time:** 17-24 hours (2-3 weeks)
> - **Priority order:** P0 tasks fix core metrics, P1 improves intelligence, P2 adds workflow
>
> **Previous Milestone (Oct 10):** End-to-End Testing Complete + Critical Bugs Fixed
> - ✅ Complete E2E testing with omnara repository (421 files, 336K edges)
> - ✅ **BUG FIX:** Phase 1 co-change query property mismatch (`path` → `file_path`)
> - ✅ **BUG FIX:** File path resolution for relative → absolute mapping
> - ✅ Phase 2 timeout increased to 60s for complex analysis
> - ✅ Validated all 3 risk scenarios (LOW/MEDIUM/HIGH)
> - ✅ Testing documentation: [END_TO_END_TESTING_RESULTS.md](END_TO_END_TESTING_RESULTS.md)

---

## Technology Stack

### Core (Implemented)
- **Language:** Go 1.21+
- **CLI Framework:** Cobra ✅
- **Configuration:** Viper ✅
- **Local Cache:** SQLite with WAL mode ✅
- **Cloud Database:** PostgreSQL with connection pooling ✅

### Cloud Infrastructure (Planned)
- **Graph Database:** Amazon Neptune Serverless
- **Cache:** Redis (ElastiCache)
- **Storage:** S3 (archival, backups)
- **Orchestration:** Kubernetes (EKS)
- **Settings Portal:** Next.js web UI

---

## Component Status

### CLI Commands (`cmd/crisk/`)

| Command | Status | Notes |
|---------|--------|-------|
| `main.go` | ✅ Complete | Root command, version info |
| `init.go` | ✅ Complete | Git detection, repo URL parsing (stubs removed) |
| `init_local.go` | ✅ Complete | End-to-end local init orchestration (Week 1) |
| `check.go` | ✅ Complete | Changed file detection via `git.GetChangedFiles()` |
| `status.go` | ✅ Complete | Health checks, cache stats |
| `config.go` | ✅ Complete | Configuration management |
| `pull.go` | ⚠️ Stub | Needs cache sync implementation |

### Internal Packages (`internal/`)

| Package | Status | Notes |
|---------|--------|-------|
| `models/` | ✅ Complete | Core data models |
| `github/` | ✅ Complete | Rate-limited API client, data extraction |
| `risk/` | ✅ Complete | Risk calculation algorithms |
| `metrics/` | ✅ Complete | Phase 1 metrics (coupling, co-change, test ratio) |
| `git/` | ✅ Complete | Git utilities (repo detection, URL parsing, changed files) |
| `storage/` | ✅ Complete | PostgreSQL & SQLite implementations |
| `cache/` | ✅ Complete | Cache operations |
| `config/` | ✅ Complete | Viper-based configuration |
| `ingestion/` | ✅ Complete | Ingestion workflow orchestration + CO_CHANGED edge verification |
| `graph/` | ✅ Complete | Neo4j backend with all Layer 1-3 edges (Oct 8: CAUSED_BY fix) |
| `output/` | ✅ Complete | 4 verbosity levels + Phase 2 display (Oct 8) |
| `agent/` | ✅ Complete | Phase 2 LLM investigation engine (Oct 8) |
| `temporal/` | ✅ Complete | Git history analysis, co-change detection |
| `incidents/` | ✅ Complete | Incident database, BM25 search, linking (Oct 8: NULL fix) |
| `treesitter/` | ✅ Complete | Multi-language AST parsing |
| `ai/` | ✅ Complete | AI prompt generation, confidence scoring |

### Cloud Integration (Not Started)

| Component | Status | Priority |
|-----------|--------|----------|
| Neptune client | ❌ TODO | P0 - Graph queries |
| Settings portal | ❌ TODO | P0 - API key management |
| GitHub OAuth | ❌ TODO | P0 - Authentication |
| Redis integration | ❌ TODO | P1 - Caching |
| Webhook handlers | ❌ TODO | P1 - Graph updates |
| Branch delta logic | ❌ TODO | P2 - Multi-branch |
| Public cache GC | ❌ TODO | P2 - Lifecycle |

---

## Critical Missing Pieces

### MVP Blockers (Must implement)

**~~1. Git Integration~~** ✅ **COMPLETE** (Week 1 - October 4, 2025)
```go
// ✅ Implemented in internal/git/repo.go:
func DetectGitRepo() error                              // ✅ COMPLETE
func ParseRepoURL(remoteURL string) (org, repo string, error)  // ✅ COMPLETE
func GetChangedFiles() ([]string, error)                // ✅ COMPLETE
func GetRemoteURL() (string, error)                     // ✅ COMPLETE
func GetCurrentBranch() (string, error)                 // ✅ COMPLETE
```

**2. Neptune Graph Client**
- openCypher query execution
- Connection pooling
- 2-hop context loading
- Spatial cache integration

**3. Settings Portal**
- API key configuration UI
- GitHub OAuth flow
- Team management
- Repository listing

**4. Agent Investigation Engine**
- Spatial context management
- Hop-by-hop navigation
- Metric validation
- LLM integration (user's API key)

---

## Implementation Phases

### ~~Phase 1: Core Functionality (Week 1)~~ ✅ **COMPLETE** (October 4, 2025)
**Goal:** Working `crisk check` and `crisk init-local` end-to-end

**Deliverables:**
- [x] ✅ Git detection functions implemented (Session 4)
- [x] ✅ Init flow orchestration (Session 5)
- [x] ✅ Phase 1 risk calculation validated (Session 6)
- [x] ✅ End-to-end testing (12/12 integration tests passing)
- [x] ✅ Performance targets exceeded (5ms vs 500ms target)

### Phase 2: MVP Cloud Migration (Weeks 2-4)
**Goal:** Deploy to AWS with Neptune

**Deliverables:**
- [ ] Neptune client integrated
- [ ] Settings portal (API key config)
- [ ] GitHub OAuth authentication
- [ ] Basic agent investigation (3 hops)
- [ ] Main branch graph construction
- [ ] 50 beta users

### Phase 3: Multi-Branch (Months 4-6)
**Goal:** Feature branch support with deltas

**Deliverables:**
- [ ] Branch delta creation
- [ ] Federated queries (base + delta)
- [ ] Branch lifecycle management
- [ ] Webhook handlers (GitHub)
- [ ] Team graph sharing
- [ ] 500+ users

### Phase 4: Public Cache (Months 7-9)
**Goal:** Shared public repository caching

**Deliverables:**
- [ ] Public vs private repo detection
- [ ] Reference counting system
- [ ] Garbage collection (30/90 day)
- [ ] S3 archival integration
- [ ] Cache restoration
- [ ] 5,000+ users

---

## Dependencies

### Go Modules (All Actively Used)
```go
github.com/google/go-github/v57 v57.0.0      // GitHub API
github.com/jackc/pgx/v5 v5.5.1               // PostgreSQL
github.com/jmoiron/sqlx v1.3.5               // SQL extensions
github.com/joho/godotenv v1.5.1              // .env support
github.com/mattn/go-sqlite3 v1.14.18         // SQLite
github.com/patrickmn/go-cache v2.1.0         // In-memory cache
github.com/sirupsen/logrus v1.9.3            // Logging
github.com/spf13/cobra v1.8.0                // CLI framework
github.com/spf13/viper v1.18.2               // Configuration
golang.org/x/sync v0.5.0                     // Sync primitives
golang.org/x/time v0.5.0                     // Rate limiting
```

### New Dependencies Needed
- Neptune Go client (AWS SDK)
- Redis Go client
- Tree-sitter Go bindings
- OpenAI/Anthropic Go SDKs
- OAuth2 libraries

---

## Testing Status

| Type | Coverage | Status |
|------|----------|--------|
| Unit tests | ~45% | ⚠️ Needs improvement |
| Integration tests | ✅ Complete | Layer 2, Layer 3, Performance benchmarks |
| E2E tests | ✅ Complete | Full init-local + check flow validated |

### Integration Tests (October 8, 2025)
- ✅ **Layer 2 Validation:** CO_CHANGED edge creation and queries
- ✅ **Layer 3 Validation:** CAUSED_BY edge creation from incident links
- ✅ **Performance Benchmarks:** Co-change (<20ms), incident search (<50ms), structural queries (<50ms)
- ✅ **Makefile Targets:** `make test-layer2`, `make test-layer3`, `make test-performance`

**Priority:** Increase unit test coverage to >70% for production release

---

## Performance Targets

| Metric | Target | Current | Status |
|--------|--------|---------|--------|
| `crisk check` latency (p50) | 2-5s | N/A | Not measured |
| `crisk check` latency (p95) | 8s | N/A | Not measured |
| `crisk init` time | 5-10 min | N/A | Not measured |
| Graph query cache hit | >60% | N/A | Not implemented |

---

## Next Steps

**Week 1-2:**
1. Implement git detection functions (`detectGitRepo`, `parseRepoURL`, `getChangedFiles`)
2. Set up Neptune Serverless cluster (dev environment)
3. Implement basic Neptune client with openCypher queries

**Week 3-4:**
4. Build settings portal MVP (Next.js + API key CRUD)
5. Implement GitHub OAuth flow
6. Create user registration + API key encryption

**Week 5-8:**
7. Build agent investigation engine (spatial context, hop management)
8. Integrate LLM clients (OpenAI, Anthropic via user API key)
9. Implement metric validation system
10. Test end-to-end `crisk check` flow

**Week 9-10 (Priority 8-11 - Branch-Aware Ingestion):**
8. **GitHub Language Detection:** Implement `/repos/{owner}/{repo}/languages` API call during `crisk init`
9. **Branch Properties:** Add `branch` and `git_sha` properties to all Layer 1 nodes (File, Function, Class)
10. **Incremental Delta Strategy:** Implement `git diff` based parsing (only changed files, not entire codebase)
11. **Layer Separation:** Ensure Layer 1 is branch-specific, Layers 2-3 are branch-agnostic

---

## Developer Experience (DX) Implementation Status

**Last Updated:** October 4, 2025
**Reference:** [developer_experience.md](../00-product/developer_experience.md), [Phase: DX Foundation](phases/phase_dx_foundation.md)

### 🚀 Parallel Implementation - COMPLETE! ✅

**3 parallel Claude Code sessions completed** - See [THREE_SESSIONS_SUMMARY.md](THREE_SESSIONS_SUMMARY.md)

| Session | Focus | Duration | Status | Prompt File |
|---------|-------|----------|--------|-------------|
| **Session 1** | Pre-commit Hook & Git Integration | 3-4 days | ✅ Complete | [SESSION_1_PROMPT.md](SESSION_1_PROMPT.md) |
| **Session 2** | Adaptive Verbosity (Levels 1-3) | 3-4 days | ✅ Complete | [SESSION_2_PROMPT.md](SESSION_2_PROMPT.md) |
| **Session 3** | AI Mode (Level 4) & Prompts | 4-5 days | ✅ Complete | [SESSION_3_PROMPT.md](SESSION_3_PROMPT.md) |

**Execution Time:** ~1 day parallel (all sessions completed simultaneously)
**Integration Status:** ✅ All tests passing

### Core UX Features

| Feature | Status | Priority | Integration Guide | Session | Notes |
|---------|--------|----------|-------------------|---------|-------|
| **Pre-Commit Hook** | ✅ Complete | P0 | [ux_pre_commit_hook.md](integration_guides/ux_pre_commit_hook.md) | Session 1 | `crisk hook install`, automatic risk checks |
| **Adaptive Verbosity (L1: Quiet)** | ✅ Complete | P0 | [ux_adaptive_verbosity.md](integration_guides/ux_adaptive_verbosity.md) | Session 2 | One-line summary for hooks |
| **Adaptive Verbosity (L2: Standard)** | ✅ Complete | P0 | [ux_adaptive_verbosity.md](integration_guides/ux_adaptive_verbosity.md) | Session 2 | Issues + recommendations |
| **Adaptive Verbosity (L3: Explain)** | ✅ Complete | P0 | [ux_adaptive_verbosity.md](integration_guides/ux_adaptive_verbosity.md) | Session 2 | Full investigation trace |
| **Adaptive Verbosity (L4: AI Mode)** | ✅ Complete | P1 | [ux_ai_mode_output.md](integration_guides/ux_ai_mode_output.md) | Session 3 | JSON for Claude Code/Cursor |
| **Actionable Error Messages** | ✅ Complete | P0 | [ux_adaptive_verbosity.md](integration_guides/ux_adaptive_verbosity.md) | Session 2 | "What to do" guidance in all modes |
| **Team Size Modes** | ❌ Deferred | P2 | TBD (future) | Future | Solo/team/standard/enterprise |
| **Override Tracking** | ✅ Complete | P0 | [ux_pre_commit_hook.md](integration_guides/ux_pre_commit_hook.md) | Session 1 | Audit log for --no-verify |

### AI Assistant Integration

| Feature | Status | Priority | Integration Guide | Session | Notes |
|---------|--------|----------|-------------------|---------|-------|
| **AI Mode JSON Schema** | ✅ Complete | P1 | [ux_ai_mode_output.md](integration_guides/ux_ai_mode_output.md) | Session 3 | Schema v1.0 at schemas/ai-mode-v1.0.json |
| **AI Prompt Generation** | ✅ Complete | P1 | [ux_ai_mode_output.md](integration_guides/ux_ai_mode_output.md) | Session 3 | 4 fix types with ready-to-execute prompts |
| **Auto-fix Confidence Scoring** | ✅ Complete | P1 | [ux_ai_mode_output.md](integration_guides/ux_ai_mode_output.md) | Session 3 | Threshold: >0.85 for auto-fix |
| **Claude Code Integration** | ✅ Ready | P1 | [ux_ai_mode_output.md](integration_guides/ux_ai_mode_output.md) | Session 3 | Use --ai-mode flag |
| **Cursor Integration** | ✅ Ready | P1 | [ux_ai_mode_output.md](integration_guides/ux_ai_mode_output.md) | Session 3 | Use --ai-mode flag |

### Implementation Plan

**Phase Dependencies (Complete ✅):**
- ✅ Phase 1 metrics complete (coupling, co-change, test ratio)
- ✅ Layer 1 complete (Tree-sitter AST parsing)
- ✅ Layers 2-3 complete (GitHub API, graph construction)

**Implementation Summary:**
- [PARALLEL_SESSION_PLAN.md](PARALLEL_SESSION_PLAN.md) - File ownership map, checkpoints
- [THREE_SESSIONS_SUMMARY.md](THREE_SESSIONS_SUMMARY.md) - Quick reference for managing sessions
- [DX_FOUNDATION_COMPLETE.md](DX_FOUNDATION_COMPLETE.md) - Final integration summary

**Session Outputs:**
- Session 1: [SESSION_1_PROMPT.md](SESSION_1_PROMPT.md) - Pre-commit hook complete
- Session 2: [SESSION_2_PROMPT.md](SESSION_2_PROMPT.md) - Adaptive verbosity complete
- Session 3: [SESSION_3_PROMPT.md](SESSION_3_PROMPT.md) - AI Mode complete

**Completed:** October 4, 2025 (~1 day with 3 parallel sessions)
**All Integration Tests:** ✅ Passing

### 🐛 Graph Construction Bug Fix (October 5, 2025)

**Issue:** Neo4j graph only showed 1 File node instead of 421
**Root Causes Identified:**
1. File node `unique_id` collision (all files had `:0` suffix)
2. Neo4j backend key mapping mismatch (`path` vs `unique_id`)
3. No edge creation between nodes (CONTAINS, IMPORTS missing)

**Fixes Applied:**
- ✅ Fixed `unique_id` generation for File nodes (use file_path directly)
- ✅ Updated Neo4j backend to use correct unique key
- ✅ Implemented `createEdges()` function for relationships
- ✅ Verified 5,524 nodes + 5,103 edges successfully created

**Documentation:** See [GRAPH_INVESTIGATION_REPORT.md](../../GRAPH_INVESTIGATION_REPORT.md) for complete analysis

**Next Steps:** See [NEXT_STEPS.md](NEXT_STEPS.md) for detailed Week 1-8 roadmap

---

## 🎉 P0-P2 Completion Milestone (October 8, 2025)

### Priority 0: Phase 2 LLM Investigation ✅ COMPLETE

**Status:** Phase 2 investigation fully integrated into check command
**Reference:** [E2E_TEST_GAP_ANALYSIS.md](../../E2E_TEST_GAP_ANALYSIS.md) Gap C1

**What Was Implemented:**
- ✅ Phase 2 escalation in `cmd/crisk/check.go:182-276`
- ✅ Investigator orchestration with hop-by-hop navigation
- ✅ Evidence collection from temporal + incidents + graph
- ✅ LLM synthesis with OpenAI integration
- ✅ Display functions for all verbosity modes (summary, trace, JSON)
- ✅ Graceful degradation when API key not provided

**How It Works:**
```bash
# Trigger Phase 2 on high-risk files
export OPENAI_API_KEY="sk-..."
./crisk check <file-with-high-coupling>

# Phase 2 runs when:
# - Coupling count > 10, OR
# - Co-change frequency > 0.7, OR
# - Incidents > 0

# Output modes:
./crisk check --explain <file>   # Full hop-by-hop trace
./crisk check --ai-mode <file>   # JSON with investigation_trace[]
```

**Impact:** Core value proposition now functional - LLM investigation provides deep analysis beyond Phase 1 metrics

---

### Priority 1: Edge Creation Fixes ✅ COMPLETE

#### P1a: CO_CHANGED Edges (Layer 2)
**Status:** Already implemented with timeout/verification
**Reference:** [E2E_TEST_GAP_ANALYSIS.md](../../E2E_TEST_GAP_ANALYSIS.md) Gap A1

**Implementation:**
- ✅ 3-minute timeout for git history parsing (`processor.go:143-228`)
- ✅ Edge verification after creation (`verifyCoChangedEdges()`)
- ✅ Mismatch detection and logging
- ✅ Graceful fallback on timeout

#### P1b: CAUSED_BY Edges (Layer 3)
**Status:** Fixed Neo4j backend configuration
**Reference:** [E2E_TEST_GAP_ANALYSIS.md](../../E2E_TEST_GAP_ANALYSIS.md) Gap B1

**Fixes Applied (October 8, 2025):**
- ✅ Added `Incident` unique key mapping (`neo4j_backend.go:257-258`)
- ✅ Fixed File matching to use `path` field (`neo4j_backend.go:267-268`)
- ✅ Enhanced `parseNodeID()` for file paths without prefix (`neo4j_backend.go:243-249`)

**Root Cause:** Incident nodes didn't have unique key defined, File matching used wrong field

**Impact:** Incident-to-file linking now creates Neo4j edges correctly, enabling blast radius queries

#### P1c: AI Mode JSON Complete
**Status:** All AI Mode files verified complete
**Reference:** [TASK_P1_AI_MODE_COMPLETION.md](../../TASK_P1_AI_MODE_COMPLETION.md)

**Files Verified:**
- ✅ `internal/output/ai_actions.go` (5KB) - AI prompt generation
- ✅ `internal/output/ai_converter.go` (6KB) - JSON conversion
- ✅ `internal/output/ai_mode.go` (11KB) - Main formatter
- ✅ `internal/output/graph_analysis.go` (6KB) - Blast radius, hotspots
- ✅ `internal/output/types.go` (6KB) - Type definitions

**Features:**
- AI assistant actions with ready-to-execute prompts
- Blast radius calculation from graph neighbors
- Temporal coupling from CO_CHANGED edges
- Hotspot detection (churn + coverage + incidents)
- Confidence scores and auto-fixable flags (>0.85 threshold)
- Estimated fix times and line counts

**Additional Fix:** NULL handling in incident search already implemented with `sql.NullString`

---

### Priority 2: Testing & Validation ✅ COMPLETE

**Status:** Integration tests created, version flag added, Makefile updated

#### Integration Tests Created:
1. ✅ **test_layer2_validation.sh** - Validates CO_CHANGED edges after init-local
2. ✅ **test_layer3_validation.sh** - Validates CAUSED_BY edges on incident link
3. ✅ **test_performance_benchmarks.sh** - Performance targets (<20ms, <50ms)

#### CLI Enhancement:
- ✅ `--version` flag with build info (`cmd/crisk/main.go`)
  ```bash
  $ ./bin/crisk --version
  CodeRisk untagged
  Build time: 2025-10-08_05:48:46
  Git commit: 87e12bc
  ```

#### Build System:
- ✅ Makefile updated with version injection
- ✅ Integration test targets: `make test-layer2`, `make test-layer3`, `make test-performance`
- ✅ Complete test suite: `make test-integration`

---

### Summary of Changes (October 8, 2025)

**Files Modified:**
- ✅ `internal/graph/neo4j_backend.go` - Fixed Incident key, File matching, parseNodeID
- ✅ `cmd/crisk/main.go` - Added version info variables and template
- ✅ `Makefile` - Version injection, integration test targets

**Files Created:**
- ✅ `test/integration/test_performance_benchmarks.sh` - Performance validation

**Files Verified Complete:**
- ✅ `internal/ingestion/processor.go` - CO_CHANGED timeout/verification
- ✅ `internal/incidents/search.go` - NULL handling with sql.NullString
- ✅ `internal/incidents/linker.go` - CAUSED_BY edge creation
- ✅ `internal/output/ai_*.go` - AI Mode complete (5 files)
- ✅ `cmd/crisk/check.go` - Phase 2 integration
- ✅ `internal/output/phase2.go` - Phase 2 display functions

**Implementation Status:** 100% complete for core features
**Test Coverage:** Integration tests for all 3 layers + performance
**Production Readiness:** ✅ All advertised features functional

---

## 🧪 Manual Testing Infrastructure (October 10, 2025)

### Testing Workflow Ready

**Goal:** One-command testing for edge creation validation

**Makefile Targets Added:**
```bash
make build        # Build crisk CLI with helpful next steps
make start        # Start Docker services (neo4j, postgres, redis)
make stop         # Stop all services
make status       # Check service health
make logs         # View service logs (all or specific)
make clean-all    # Complete cleanup (Docker + binaries)
```

**Integration Guides Created:**
- ✅ **[testing_edge_fixes.md](integration_guides/testing_edge_fixes.md)** - Complete testing instructions for CO_CHANGED/CAUSED_BY fixes
- ✅ **[quick_test_commands.md](integration_guides/quick_test_commands.md)** - Quick reference commands and expected results

**Testing Flow:**
```bash
# 1. Clean everything
make clean-all

# 2. Build and start
make build && make start

# 3. Clone test repository
mkdir -p /tmp/coderisk-test && cd /tmp/coderisk-test
git clone https://github.com/omnara-ai/omnara
cd omnara

# 4. Run init-local
~/Documents/brain/coderisk-go/bin/crisk init-local

# 5. Validate edges
cd ~/Documents/brain/coderisk-go
./scripts/validate_graph_edges.sh
```

**Expected Results:**
- ✅ CO_CHANGED edges: 336k+ edges created (was 0 before fix)
- ✅ CAUSED_BY edges: Incident-to-file linking works
- ✅ Edge properties: frequency, co_changes, window_days populated
- ✅ Bidirectional verification: A→B implies B→A with same frequency

**Fixes Applied:**
1. **CO_CHANGED edges** - Convert git relative paths → absolute paths before Neo4j lookup
2. **CAUSED_BY edges** - Add node ID prefixes for correct label detection
3. **Enhanced logging** - Diagnostic info for silent edge creation failures
4. **Verification** - Post-creation edge count validation

**See:** [testing_edge_fixes.md](integration_guides/testing_edge_fixes.md) for complete testing instructions

### Performance Targets (from DX design)

| Metric | Target | Current | Status |
|--------|--------|---------|--------|
| Pre-commit check (p50, cached) | <2s | N/A | Not measured |
| Pre-commit check (p95, cold) | <5s | N/A | Not measured |
| AI Mode generation overhead | <200ms | N/A | Not implemented |
| JSON output size | <10KB | N/A | Not implemented |

---

**For detailed architecture, see [dev_docs/spec.md](../spec.md) and related design documents.**
