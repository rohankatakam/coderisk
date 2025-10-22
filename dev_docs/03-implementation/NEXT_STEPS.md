# Next Steps: MVP Implementation Roadmap

**Last Updated:** October 17, 2025
**Purpose:** Actionable roadmap for MVP completion (4-6 weeks)
**Reference:** [status.md](status.md), [mvp_vision.md](../00-product/mvp_vision.md)

---

## Current State ✅

**Completed (Ready for MVP):**
- ✅ 3-layer graph ingestion (structure, temporal, incidents)
- ✅ Pre-commit hook integration
- ✅ 4 verbosity levels (quiet, standard, explain, AI mode)
- ✅ Docker Compose deployment (local Neo4j)
- ✅ CLI commands (`init`, `check`, `hook`, `incident`, `config`)
- ✅ Git utilities (detection, diff, history analysis)
- ✅ Tree-sitter AST parsing (Python, JS, TS, Go)
- ✅ Development tooling (GoReleaser, GitHub Actions, Homebrew)

**In Progress (Current Sprint):**
- 🚧 Risk assessment integration
- 🚧 LLM client (OpenAI/Anthropic)
- 🚧 Performance optimization
- 🚧 Test coverage improvements

---

## MVP Scope Reminder

**✅ In Scope (Local-First MVP):**
- Local Neo4j + Docker deployment
- Two-phase risk assessment (fast baseline + LLM for high-risk)
- Five simple metrics (coupling, co-change, test ratio, churn, incidents)
- Pre-commit hook workflow
- BYOK model (user's API key)
- Solo developers + small teams

**❌ Out of Scope (Post-MVP):**
- Cloud deployment (Neptune, K8s, Lambda)
- Settings portal / GitHub OAuth
- Multi-tenancy / team features
- Public repository caching
- Complex agent orchestration

---

## Week-by-Week Roadmap

### Week 1-2 (Current): Risk Assessment Integration

**Goal:** Complete end-to-end risk assessment in `crisk check`

**Tasks:**
1. ✅ Phase 1 baseline metrics (coupling, co-change, test ratio)
2. 🚧 Phase 2 LLM integration (single call for high-risk files)
3. 🚧 Threshold configuration (configurable risk levels)
4. 🚧 Integration testing with real repositories

**Deliverables:**
- Working `crisk check` with Phase 1 + 2
- LLM client (OpenAI/Anthropic)
- Integration tests passing

**Success Criteria:**
- Phase 1 completes in <200ms
- Phase 2 completes in <5s
- Accurate risk levels (LOW/MEDIUM/HIGH)

---

### Week 3-4: Performance & Validation

**Goal:** Achieve performance targets and validate accuracy

**Tasks:**
1. Performance optimization (caching, query optimization)
2. False positive tracking implementation
3. User feedback mechanism
4. Real-world repository testing

**Deliverables:**
- Performance targets met (<200ms, <5s)
- False positive tracking system
- Validation with 5-10 repositories

**Success Criteria:**
- p50 latency <200ms (Phase 1)
- p95 latency <5s (Phase 2)
- False positive rate <10%

---

### Week 5-6: Production Hardening & Beta

**Goal:** Production-ready quality and beta user testing

**Tasks:**
1. Error handling improvements
2. Edge case coverage
3. Unit test coverage >70%
4. Beta user onboarding (10 users)

**Deliverables:**
- Production-ready codebase
- Comprehensive test suite
- Beta user feedback

**Success Criteria:**
- Test coverage >70%
- 10 beta users successfully using crisk
- No critical bugs

---

### Week 7 (Optional): Launch Preparation

**Goal:** Polish and launch

**Tasks:**
1. Documentation polish
2. Bug fixes from beta
3. Performance tuning
4. Launch preparation

**Deliverables:**
- Complete user documentation
- Stable v1.0 release
- Public announcement

---

## Immediate Priorities (This Sprint)

### P0 - Critical (Must Complete This Week)

**1. LLM Client Integration**
- Implement OpenAI client
- Implement Anthropic client
- User API key configuration (environment variable)
- Error handling (no API key, rate limits, timeouts)

**Files:**
- `internal/llm/openai.go` (new)
- `internal/llm/anthropic.go` (new)
- `internal/llm/client.go` (interface)

**2. Phase 2 Integration**
- Connect Phase 2 to `crisk check` command
- Escalation logic (Phase 1 → Phase 2 when high-risk)
- Single LLM call (no multi-hop navigation in MVP)
- Display results in all verbosity modes

**Files:**
- `cmd/crisk/check.go` (update)
- `internal/agent/investigator.go` (simplify)

**3. Integration Testing**
- Test with real repositories (omnara, react, next.js)
- Validate risk levels match expectations
- Measure performance

**Files:**
- `test/integration/test_risk_assessment.sh` (new)

---

### P1 - Important (Next Week)

**4. Performance Optimization**
- Cache Phase 1 results (filesystem)
- Optimize Neo4j queries (use indexes)
- Reduce LLM token usage (context pruning)

**5. Configuration Management**
- API key setup (env var, config file, keychain)
- Threshold configuration (coupling, co-change, etc.)
- Verbosity defaults

**6. False Positive Tracking**
- User feedback command (`crisk feedback --false-positive`)
- Store feedback in SQLite
- Display stats (`crisk stats --false-positives`)

---

### P2 - Nice to Have (Later)

**7. Monitoring & Logging**
- Basic metrics (check count, phase escalation rate)
- Error logging
- Performance tracking

**8. Documentation**
- User guide (installation, usage)
- Troubleshooting guide
- FAQ

---

## Technical Decisions Needed

### 1. LLM Provider Priority

**Options:**
- A) Support only OpenAI initially
- B) Support both OpenAI + Anthropic from start
- C) Support Anthropic only (cheaper, better for code)

**Recommendation:** B (both) - Users have preferences, BYOK model supports both

---

### 2. API Key Storage

**Options:**
- A) Environment variable only (`OPENAI_API_KEY`)
- B) Config file (`~/.coderisk/config.yaml`)
- C) OS Keychain (macOS Keychain, Windows Credential Manager)
- D) All of the above

**Recommendation:** D (all) - Start with env var, add config file, then keychain

---

### 3. Phase 2 Escalation Threshold

**Options:**
- A) Fixed threshold (coupling >10, co-change >0.7)
- B) Configurable threshold (user can adjust)
- C) Adaptive threshold (learns from feedback)

**Recommendation:** B (configurable) - Default thresholds, allow user customization

---

## Dependencies & Blockers

### External Dependencies

**Required:**
- User's OpenAI/Anthropic API key (BYOK)
- Docker installed (for local Neo4j)
- Git installed (for repository analysis)

**Optional:**
- GitHub API token (for enhanced history analysis)

---

### Internal Blockers

**None currently** - All MVP features can be implemented with existing infrastructure

---

## Testing Strategy

### Unit Tests (Target: >70% Coverage)

**Priority packages:**
- `internal/llm/` - LLM clients
- `internal/risk/` - Risk assessment
- `internal/metrics/` - Metric calculation
- `internal/agent/` - Investigation logic

**Current coverage:** ~45%
**Gap:** Need 400+ additional test cases

---

### Integration Tests

**Scenarios:**
1. End-to-end `crisk check` with Phase 1 only
2. End-to-end `crisk check` with Phase 2 escalation
3. Performance testing (latency targets)
4. Real repository testing (omnara, react, etc.)

**Test repositories:**
- Small: commander.js (~50 files)
- Medium: omnara (~400 files)
- Large: next.js (~1,500 files)

---

### Performance Benchmarks

**Targets:**
| Metric | Target | How to Test |
|--------|--------|-------------|
| Phase 1 latency (p50) | <200ms | Benchmark with 100 files |
| Phase 2 latency (p95) | <5s | Test with high-risk files |
| `crisk init` time | <10 min | Test with medium repo (omnara) |
| Cache hit rate | >30% | Test repeated checks |

---

## Success Metrics (MVP Launch)

### Functional Requirements

- ✅ `crisk init` works for local repos
- ✅ `crisk check` provides accurate risk assessment
- ✅ Pre-commit hook blocks high-risk commits
- ✅ All 4 verbosity levels functional
- ✅ Performance targets met

### Non-Functional Requirements

- ✅ Test coverage >70%
- ✅ No critical bugs
- ✅ Documentation complete
- ✅ 10 beta users successful

### User Validation

- ✅ Users can install via Homebrew/curl
- ✅ Users can run `crisk init` successfully
- ✅ Users find risk assessments helpful
- ✅ False positive rate acceptable (<10%)

---

## Post-MVP (Future Considerations)

**Deferred to validate demand first:**
- Cloud deployment (Neptune, managed service)
- Settings portal / GitHub OAuth
- Team features (shared graphs)
- Public repository caching
- Advanced metrics (complexity, maintainability)

**See archived docs:**
- [../99-archive/02-operations-future-vision/](../99-archive/02-operations-future-vision/) - Cloud operations
- [../99-archive/01-architecture-future-vision/](../99-archive/01-architecture-future-vision/) - Cloud architecture

---

## Resources & References

**Implementation:**
- [status.md](status.md) - Current implementation status
- [DEVELOPMENT_AND_DEPLOYMENT.md](DEVELOPMENT_AND_DEPLOYMENT.md) - Development workflow

**Product:**
- [../00-product/mvp_vision.md](../00-product/mvp_vision.md) - MVP scope
- [../00-product/developer_experience.md](../00-product/developer_experience.md) - UX design

**Architecture:**
- [../01-architecture/mvp_architecture_overview.md](../01-architecture/mvp_architecture_overview.md) - System design
- [../01-architecture/agentic_design.md](../01-architecture/agentic_design.md) - Investigation flow
- [../01-architecture/risk_assessment_methodology.md](../01-architecture/risk_assessment_methodology.md) - Risk calculation

**Operations:**
- [../02-operations/local_deployment_operations.md](../02-operations/local_deployment_operations.md) - Deployment

---

**Last Updated:** October 17, 2025
**Next Review:** Weekly (update after each sprint)
