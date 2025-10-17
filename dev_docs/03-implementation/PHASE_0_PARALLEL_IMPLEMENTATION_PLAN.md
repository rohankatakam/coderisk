# Phase 0 + Adaptive System: Parallel Implementation Plan

**Created:** October 10, 2025
**Status:** Ready for Execution
**Execution Mode:** 4 Parallel Claude Code Agents + 1 Manager Agent
**Target:** Optimize current system (without ARC) to achieve 70-80% FP reduction

---

## Executive Summary

**Objective:** Implement Phase 0 pre-analysis + adaptive thresholds + confidence-driven investigation to maximize performance **before** ARC intelligence integration.

**Why This Order:**
1. **Phase 0 + Adaptive System First** → Optimize foundation, reduce FPs by 70-80%
2. **Validate on Test Repository** → Ensure system performs optimally
3. **Then Add ARC Intelligence** → Layer ARC patterns on optimized foundation

**Expected Outcomes (Without ARC):**
- False Positive Rate: 50% → 10-15% (70-80% reduction)
- Average Latency: 2,500ms → 662ms (3.8x faster)
- Insight Quality: Generic → Domain-specific (3x better)

**Timeline:** 2-3 weeks with 4 parallel agents

---

## Architecture Context

### Key Documents to Reference

**Before starting, agents should be familiar with:**

1. **[decisions/005-confidence-driven-investigation.md](../01-architecture/decisions/005-confidence-driven-investigation.md)**
   - Complete architecture decision for Phase 0 + adaptive system
   - Implementation timeline and expected outcomes
   - Validation metrics and A/B testing framework

2. **[agentic_design.md](../01-architecture/agentic_design.md)**
   - Three-phase architecture (Phase 0 → Phase 1 → Phase 2)
   - Confidence-driven investigation loop
   - Expected performance distribution

3. **[testing/MODIFICATION_TYPES_AND_TESTING.md](testing/MODIFICATION_TYPES_AND_TESTING.md)**
   - 10 modification types (security, docs, config, etc.)
   - Phase 0 detection rules for each type
   - Risk escalation/skip logic

4. **[testing/TEST_RESULTS_OCT_2025.md](testing/TEST_RESULTS_OCT_2025.md)**
   - Current system gaps and false positives
   - Empirical evidence of performance issues
   - Target scenarios to validate against

5. **[DEVELOPMENT_WORKFLOW.md](../DEVELOPMENT_WORKFLOW.md)**
   - Code quality standards and security guardrails
   - 12-factor principles application
   - Testing and validation requirements

---

## Parallel Execution Strategy

### Agent Roles and Dependencies

```
┌─────────────────────────────────────────────────────────────┐
│                    MANAGER AGENT (You)                       │
│  - Monitors all 4 agents                                    │
│  - Coordinates handoffs                                     │
│  - Integrates final outputs                                 │
│  - Resolves conflicts                                       │
└─────────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
┌───────▼──────┐   ┌───────▼──────┐   ┌───────▼──────┐   ┌───────▼──────┐
│  AGENT 1     │   │  AGENT 2     │   │  AGENT 3     │   │  AGENT 4     │
│  Phase 0     │   │  Adaptive    │   │  Confidence  │   │  Test &      │
│  Foundation  │   │  Configs     │   │  Loop        │   │  Validation  │
│              │   │              │   │              │   │              │
│  Week 1-2    │   │  Week 1      │   │  Week 2      │   │  Week 1-3    │
│  Independent │   │  Independent │   │  Depends on  │   │  Continuous  │
│              │   │              │   │  Agent 1     │   │  Integration │
└──────────────┘   └──────────────┘   └──────────────┘   └──────────────┘
```

**Execution Philosophy:**
- **Agents 1, 2, 4:** Run in parallel from Day 1 (independent)
- **Agent 3:** Starts after Agent 1 completes Phase 0 foundation
- **Manager (You):** Review checkpoints, coordinate handoffs, integrate work

---

## Agent 1: Phase 0 Pre-Analysis Foundation

**Working Directory:** `/Users/rohankatakam/Documents/brain/coderisk-go`

**Objective:** Implement Phase 0 pre-analysis layer with security/docs/config detection

**Duration:** Week 1-2 (10-12 days)

**Dependencies:** None (independent)

---

### Scope of Work

**Deliverables:**
1. Phase 0 pre-analysis engine (security, docs, env detection)
2. Modification type classifier (10 types from MODIFICATION_TYPES_AND_TESTING.md)
3. Force escalation logic (security → CRITICAL, production config → HIGH)
4. Skip analysis logic (docs-only → LOW, return immediately)
5. Integration with existing `crisk check` command
6. Unit tests for all modification types

**Files to Create:**
- `internal/analysis/phase0/detector.go` - Main Phase 0 orchestrator
- `internal/analysis/phase0/security.go` - Security keyword detection
- `internal/analysis/phase0/documentation.go` - Documentation skip logic
- `internal/analysis/phase0/environment.go` - Production config detection
- `internal/analysis/phase0/modification_types.go` - Type classifier
- `internal/analysis/phase0/*_test.go` - Comprehensive tests

**Files to Modify:**
- `cmd/crisk/check.go` - Integrate Phase 0 before Phase 1
- `internal/investigation/risk_assessment.go` - Accept Phase 0 results

---

### Implementation Steps

**Checkpoint 1: Security Keyword Detection (Days 1-3)**

1. Read [decisions/005-confidence-driven-investigation.md](../01-architecture/decisions/005-confidence-driven-investigation.md) §"Decision" → "1. Phase 0: Adaptive Pre-Analysis"
2. Read [testing/MODIFICATION_TYPES_AND_TESTING.md](testing/MODIFICATION_TYPES_AND_TESTING.md) → Type 9: Security changes
3. Implement `internal/analysis/phase0/security.go`:
   - `DetectSecurityKeywords(filePath, diff) (bool, []string)` - Returns true + keywords found
   - Keywords: auth, login, password, session, token, jwt, crypto, encrypt, decrypt, hash, salt, permission, role, admin, sudo
   - Check file path patterns: `*auth*.go`, `*security*.go`, `*permission*.go`
4. Write comprehensive tests with false positive scenarios
5. **STOP and report:** "Security detection complete. Found X keywords, tested Y scenarios. Ready for review."

**Checkpoint 2: Documentation Skip Logic (Days 4-5)**

1. Read [testing/TEST_RESULTS_OCT_2025.md](testing/TEST_RESULTS_OCT_2025.md) → Scenario 1 (README.md false positive)
2. Implement `internal/analysis/phase0/documentation.go`:
   - `IsDocumentationOnly(files, diffs) (bool, string)` - Returns true if all changes are docs
   - Doc extensions: `.md`, `.txt`, `.rst`, `.adoc`, comment-only changes
   - Check for runtime impact: if ANY code change, return false
3. Test with mixed scenarios (docs + code, docs-only, comment-only)
4. **STOP and report:** "Documentation skip logic complete. Tested Z scenarios. Ready for review."

**Checkpoint 3: Environment Detection (Days 6-7)**

1. Read [testing/MODIFICATION_TYPES_AND_TESTING.md](testing/MODIFICATION_TYPES_AND_TESTING.md) → Type 3: Configuration
2. Implement `internal/analysis/phase0/environment.go`:
   - `DetectEnvironment(filePath) (env string, isProduction bool)`
   - Production patterns: `.env.production`, `prod.config`, `production.yaml`, `/config/prod/`
   - Development patterns: `.env.local`, `dev.config`, `development.yaml`
3. Test with various config file patterns
4. **STOP and report:** "Environment detection complete. Tested config patterns. Ready for review."

**Checkpoint 4: Modification Type Classifier (Days 8-10)**

1. Read [testing/MODIFICATION_TYPES_AND_TESTING.md](testing/MODIFICATION_TYPES_AND_TESTING.md) → All 10 types
2. Implement `internal/analysis/phase0/modification_types.go`:
   - `ClassifyModification(file, diff) ModificationType` - Returns type enum
   - Support multi-type detection (file can be security + behavioral)
   - Risk aggregation: `final_risk = MAX(type_risks) + Σ(other_types × 0.3)`
3. Write tests for each type + multi-type scenarios
4. **STOP and report:** "Type classifier complete. Classified 10 types. Ready for review."

**Checkpoint 5: Phase 0 Orchestrator (Days 11-12)**

1. Read [agentic_design.md](../01-architecture/agentic_design.md) § "2.1. Three-Phase Architecture"
2. Implement `internal/analysis/phase0/detector.go`:
   - `RunPhase0(ctx, files, diffs, repoMetadata) Phase0Result`
   - Call security, docs, env detectors
   - Return: `{ForceEscalate, SkipAnalysis, ModificationType, SelectedConfig, Reason}`
3. Integrate into `cmd/crisk/check.go` BEFORE Phase 1
4. Write integration tests
5. **STOP and report:** "Phase 0 complete and integrated. Ran X scenarios. Ready for final review."

---

### Human-in-Loop Checkpoints

**After each checkpoint, agent will:**
1. Run all tests: `go test ./internal/analysis/phase0/...`
2. Report results: pass/fail counts, coverage percentage
3. Show sample outputs for 3-5 test cases
4. **WAIT for manager approval** before proceeding to next checkpoint

**Manager reviews:**
- Test coverage (aim for 80%+)
- False positive scenarios handled correctly
- Code follows [DEVELOPMENT_WORKFLOW.md](../DEVELOPMENT_WORKFLOW.md) standards
- Approve or request adjustments before agent continues

---

### Initial Prompt for Agent 1

```
I need you to implement Phase 0 pre-analysis for our code risk assessment system.

CONTEXT:
- Read dev_docs/01-architecture/decisions/005-confidence-driven-investigation.md (full ADR)
- Read dev_docs/01-architecture/agentic_design.md § "Three-Phase Architecture"
- Read dev_docs/03-implementation/testing/MODIFICATION_TYPES_AND_TESTING.md (10 types)
- Read dev_docs/03-implementation/testing/TEST_RESULTS_OCT_2025.md (current gaps)
- Read dev_docs/DEVELOPMENT_WORKFLOW.md (code quality standards)

OBJECTIVE:
Implement Phase 0 pre-analysis layer that detects security changes, documentation-only changes, and production configs BEFORE expensive Phase 1/2 analysis.

EXECUTION:
Follow dev_docs/03-implementation/PHASE_0_PARALLEL_IMPLEMENTATION_PLAN.md § "Agent 1" checkpoints 1-5.

After EACH checkpoint:
1. Run tests
2. Report results with sample outputs
3. STOP and wait for my approval before continuing

Start with Checkpoint 1: Security Keyword Detection.

Ready? Let's begin.
```

---

## Agent 2: Adaptive Configuration System

**Working Directory:** `/Users/rohankatakam/Documents/brain/coderisk-go`

**Objective:** Implement domain-aware adaptive thresholds (Python web vs Go backend)

**Duration:** Week 1 (5-7 days)

**Dependencies:** None (independent)

---

### Scope of Work

**Deliverables:**
1. Adaptive configuration selector (language + domain detection)
2. Pre-defined configs for Python/Go/TypeScript × web/backend/frontend
3. Domain inference engine (web, backend, ML, frontend)
4. Integration with Phase 1 baseline assessment
5. Configuration validation on test repository
6. Unit tests for all configurations

**Files to Create:**
- `internal/analysis/config/adaptive.go` - Config selector
- `internal/analysis/config/configs.go` - Pre-defined configs
- `internal/analysis/config/domain_inference.go` - Domain detector
- `internal/analysis/config/*_test.go` - Tests

**Files to Modify:**
- `internal/investigation/baseline.go` - Use adaptive thresholds instead of fixed

---

### Implementation Steps

**Checkpoint 1: Domain Inference (Days 1-2)**

1. Read [decisions/005-confidence-driven-investigation.md](../01-architecture/decisions/005-confidence-driven-investigation.md) § "2. Adaptive Configuration Selection"
2. Implement `internal/analysis/config/domain_inference.go`:
   - `InferDomain(repoMetadata RepoMetadata) string` - Returns: web, backend, frontend, ml, cli
   - Check: framework imports (Flask/Django → web, React → frontend)
   - Check: file structure (src/server → backend, src/app → frontend)
   - Check: package.json scripts ("start": "react-scripts" → frontend)
3. Test with omnara test repository (should detect "web" domain)
4. **STOP and report:** "Domain inference complete. Tested on X repos. Ready for review."

**Checkpoint 2: Configuration Definitions (Days 3-4)**

1. Read [agentic_design.md](../01-architecture/agentic_design.md) § "Phase 1: Baseline Assessment"
2. Implement `internal/analysis/config/configs.go`:
   - Define `RiskConfig` struct: `{CouplingThreshold, CoChangeThreshold, TestRatioThreshold}`
   - Define configs for: python_web, python_backend, go_backend, typescript_frontend, default
   - Rationale: Web apps have higher natural coupling, Go is more modular
3. Write tests validating threshold reasonableness
4. **STOP and report:** "Configs defined. Created Y configs. Ready for review."

**Checkpoint 3: Config Selector (Days 5-6)**

1. Implement `internal/analysis/config/adaptive.go`:
   - `SelectConfig(repoMetadata) RiskConfig` - Returns appropriate config
   - Use language (from repo metadata) + domain (from inference)
   - Fallback to default config if no match
2. Test with various repo types
3. **STOP and report:** "Config selector complete. Tested Z repo types. Ready for review."

**Checkpoint 4: Integration (Day 7)**

1. Modify `internal/investigation/baseline.go`:
   - Accept `RiskConfig` parameter in baseline assessment
   - Replace hardcoded thresholds with config values
   - Log which config was selected for observability
2. Write integration tests
3. **STOP and report:** "Adaptive configs integrated. Baseline uses dynamic thresholds. Ready for final review."

---

### Human-in-Loop Checkpoints

**After each checkpoint, agent will:**
1. Run tests: `go test ./internal/analysis/config/...`
2. Show example config selection for 3 different repo types
3. **WAIT for manager approval** before proceeding

**Manager reviews:**
- Threshold values are reasonable (not too strict or too loose)
- Domain inference is accurate for test repository
- Integration doesn't break existing tests

---

### Initial Prompt for Agent 2

```
I need you to implement adaptive configuration system for domain-aware risk thresholds.

CONTEXT:
- Read dev_docs/01-architecture/decisions/005-confidence-driven-investigation.md § "Adaptive Configuration Selection"
- Read dev_docs/01-architecture/agentic_design.md § "Phase 1 Baseline"
- Read dev_docs/DEVELOPMENT_WORKFLOW.md (code standards)

OBJECTIVE:
Implement adaptive threshold system that selects appropriate coupling/co-change/test-ratio thresholds based on repository language and domain (Python web has different thresholds than Go backend).

EXECUTION:
Follow dev_docs/03-implementation/PHASE_0_PARALLEL_IMPLEMENTATION_PLAN.md § "Agent 2" checkpoints 1-4.

After EACH checkpoint:
1. Run tests
2. Show sample config selections
3. STOP and wait for my approval

Start with Checkpoint 1: Domain Inference.

Ready? Let's begin.
```

---

## Agent 3: Confidence-Driven Investigation Loop

**Working Directory:** `/Users/rohankatakam/Documents/brain/coderisk-go`

**Objective:** Replace fixed 3-hop limit with confidence-based stopping

**Duration:** Week 2 (5-7 days)

**Dependencies:** Agent 1 (needs Phase 0 modification types for LLM prompts)

---

### Scope of Work

**Deliverables:**
1. Confidence assessment LLM prompt
2. Breakthrough detection (track risk level changes)
3. Dynamic hop limit (1-5 iterations, stop when confidence ≥0.85)
4. Enhanced LLM prompts with modification type context
5. Investigation trace with confidence history
6. Tests for early stopping and extended investigation

**Files to Create:**
- `internal/agent/confidence.go` - Confidence assessment logic
- `internal/agent/breakthroughs.go` - Breakthrough tracking
- `internal/agent/prompts/confidence_prompt.go` - LLM confidence prompt
- `internal/agent/*_test.go` - Tests

**Files to Modify:**
- `internal/agent/investigator.go` - Replace fixed loop with confidence-driven
- `internal/agent/prompts/decision_prompt.go` - Add modification type context

---

### Implementation Steps

**Checkpoint 1: Confidence Assessment Prompt (Days 1-2)**

**⚠️ WAIT:** This checkpoint starts AFTER Agent 1 completes Phase 0 modification types.

1. Read [decisions/005-confidence-driven-investigation.md](../01-architecture/decisions/005-confidence-driven-investigation.md) § "3. Confidence-Driven Investigation"
2. Implement `internal/agent/prompts/confidence_prompt.go`:
   - Define confidence assessment prompt (see ADR-005 example)
   - Prompt asks: "How confident are you? (0.0-1.0) Should we gather more evidence?"
   - Structured JSON output: `{confidence: float, reasoning: string, next_action: string}`
3. Test with mock LLM responses
4. **STOP and report:** "Confidence prompt defined. Tested with X scenarios. Ready for review."

**Checkpoint 2: Breakthrough Detection (Days 3-4)**

1. Read [decisions/005-confidence-driven-investigation.md](../01-architecture/decisions/005-confidence-driven-investigation.md) § "5. Breakthrough Detection"
2. Implement `internal/agent/breakthroughs.go`:
   - `TrackBreakthrough(hop, evidence, riskBefore, riskAfter, reasoning)`
   - Significant change threshold: `abs(riskAfter - riskBefore) > 0.2`
   - Store in investigation trace for explainability
3. Write tests for breakthrough scenarios
4. **STOP and report:** "Breakthrough tracking complete. Tested Y scenarios. Ready for review."

**Checkpoint 3: Confidence-Driven Loop (Days 5-6)**

1. Read [agentic_design.md](../01-architecture/agentic_design.md) § "Phase 2: Confidence-Driven Investigation"
2. Modify `internal/agent/investigator.go`:
   - Replace `for hop := 0; hop < 3; hop++` with `while confidence < 0.85 && iteration < 5`
   - After each hop, call confidence assessment
   - If confidence ≥ 0.85, break early (FINALIZE)
   - Track confidence history in investigation trace
3. Write tests for early stopping and extended investigation
4. **STOP and report:** "Confidence loop implemented. Early stop works in Z cases. Ready for review."

**Checkpoint 4: Modification Type Context (Day 7)**

1. Read [testing/MODIFICATION_TYPES_AND_TESTING.md](testing/MODIFICATION_TYPES_AND_TESTING.md) § "Type-Aware LLM Prompts"
2. Modify `internal/agent/prompts/decision_prompt.go`:
   - Add modification type to LLM context: "SECURITY CHANGE (Type 9A - Authentication)"
   - Provide type-specific guidance: "Check for bypass vulnerabilities, ensure auth flow tests exist"
3. Test with different modification types
4. **STOP and report:** "Type-aware prompts complete. LLM gets specific guidance. Ready for final review."

---

### Human-in-Loop Checkpoints

**After each checkpoint, agent will:**
1. Run tests: `go test ./internal/agent/...`
2. Show sample confidence progression: hop 1 (0.3) → hop 2 (0.6) → hop 3 (0.9) → STOP
3. **WAIT for manager approval** before proceeding

**Manager reviews:**
- Confidence threshold (0.85) is reasonable
- Early stopping works correctly (doesn't truncate critical investigations)
- Breakthrough detection captures significant risk changes

---

### Initial Prompt for Agent 3

```
I need you to implement confidence-driven investigation loop with dynamic stopping criteria.

CONTEXT:
- Read dev_docs/01-architecture/decisions/005-confidence-driven-investigation.md § "Confidence-Driven Investigation"
- Read dev_docs/01-architecture/agentic_design.md § "Phase 2"
- Read dev_docs/03-implementation/testing/MODIFICATION_TYPES_AND_TESTING.md (for type-aware prompts)
- Read dev_docs/DEVELOPMENT_WORKFLOW.md (code standards)

OBJECTIVE:
Replace fixed 3-hop investigation with confidence-based stopping. System stops when LLM confidence ≥0.85 OR max 5 hops reached.

⚠️ DEPENDENCY: Wait for Agent 1 to complete Phase 0 modification types before starting.

EXECUTION:
Follow dev_docs/03-implementation/PHASE_0_PARALLEL_IMPLEMENTATION_PLAN.md § "Agent 3" checkpoints 1-4.

After EACH checkpoint:
1. Run tests
2. Show confidence progression examples
3. STOP and wait for my approval

Coordinate with me when Agent 1 is ready, then start Checkpoint 1.

Ready? Let's coordinate timing.
```

---

## Agent 4: Testing & Continuous Validation

**Working Directory:** `/Users/rohankatakam/Documents/brain/coderisk-go/test_sandbox/omnara`

**Objective:** Validate all implementations against test scenarios continuously

**Duration:** Week 1-3 (continuous, parallel with other agents)

**Dependencies:** Consumes outputs from Agents 1, 2, 3 as they complete

---

### Scope of Work

**Deliverables:**
1. E2E test suite for Phase 0 scenarios (TEST_RESULTS_OCT_2025.md)
2. Adaptive config validation tests
3. Confidence loop validation tests
4. Performance benchmarks (latency, FP rate)
5. Regression tests (ensure existing functionality not broken)
6. Continuous integration feedback to other agents

**Files to Create:**
- `test/e2e/phase0_test.go` - Phase 0 scenario tests
- `test/e2e/adaptive_config_test.go` - Config validation
- `test/e2e/confidence_loop_test.go` - Confidence-driven investigation
- `test/benchmark/performance_test.go` - Latency benchmarks
- `test/integration/omnara_scenarios.sh` - Test scenarios in omnara repo

**Files in test_sandbox/omnara:**
- Use existing files for realistic testing
- Create new test scenarios as needed

---

### Implementation Steps

**Checkpoint 1: Phase 0 Scenario Tests (Days 1-3, after Agent 1 checkpoint 2)**

1. Read [testing/TEST_RESULTS_OCT_2025.md](testing/TEST_RESULTS_OCT_2025.md) → All 8 scenarios
2. Implement `test/e2e/phase0_test.go`:
   - Test Scenario 1: README.md (docs-only → should skip Phase 1/2)
   - Test Scenario 2: auth.py (security → should force CRITICAL)
   - Test Scenario 3: .env.production (production config → should force HIGH)
   - Test Scenario 8: comments-only → should skip
   - Validate: Latency <50ms for Phase 0 decisions
3. Run tests against latest Agent 1 code
4. **STOP and report:** "Phase 0 scenarios pass/fail: X/Y. Issues found: Z. Ready for review."

**Checkpoint 2: Adaptive Config Validation (Days 4-6, after Agent 2 completes)**

1. Implement `test/e2e/adaptive_config_test.go`:
   - Test: omnara repo should select appropriate config (detect domain)
   - Test: Python web repo should use higher coupling threshold
   - Test: Go backend repo should use stricter thresholds
   - Validate: Config selection happens in <10ms
2. Run tests on omnara + synthetic repo metadata
3. **STOP and report:** "Config validation: X/Y pass. Threshold accuracy: Z%. Ready for review."

**Checkpoint 3: Confidence Loop Validation (Days 7-10, after Agent 3 completes)**

1. Implement `test/e2e/confidence_loop_test.go`:
   - Test: LOW risk file stops at hop 1 (confidence 0.9 immediately)
   - Test: HIGH risk file continues to hop 3-4 (confidence slowly increases)
   - Test: Ambiguous case uses all 5 hops (confidence never reaches 0.85)
   - Validate: Average latency 30% faster than fixed 3-hop
2. Run tests on variety of scenarios
3. **STOP and report:** "Confidence loop: X/Y scenarios pass. Average speedup: Z%. Ready for review."

**Checkpoint 4: Performance Benchmarks (Days 11-15, after all agents complete)**

1. Implement `test/benchmark/performance_test.go`:
   - Benchmark: Average latency (target: <662ms weighted average)
   - Benchmark: False positive rate (target: <15%)
   - Benchmark: Insight quality (manual review of recommendations)
2. Run on omnara test repository (50+ test files)
3. Generate performance report
4. **STOP and report:** "Performance benchmarks complete. Latency: X ms, FP rate: Y%. Ready for final review."

**Checkpoint 5: Regression Tests (Days 16-20, final validation)**

1. Run ALL existing tests: `go test ./...`
2. Validate no regressions in:
   - Existing Phase 1 baseline (should still work without Phase 0 if needed)
   - Existing Phase 2 investigation (should work with new confidence loop)
   - CLI commands (crisk check, crisk init, etc.)
3. Generate regression report
4. **STOP and report:** "Regression tests: X/Y pass. Regressions found: Z. Ready for sign-off."

---

### Human-in-Loop Checkpoints

**After each checkpoint, agent will:**
1. Run all relevant tests
2. Generate test report: pass/fail counts, performance metrics
3. Highlight any failures or regressions
4. **WAIT for manager approval** + **Coordinate with other agents** if issues found

**Manager reviews:**
- Test coverage is comprehensive
- Performance targets are met
- Any failures are investigated (bug in implementation or test?)
- Approve or request other agents to fix issues

---

### Initial Prompt for Agent 4

```
I need you to implement comprehensive E2E testing and validation for Phase 0 + adaptive system.

CONTEXT:
- Read dev_docs/03-implementation/testing/TEST_RESULTS_OCT_2025.md (8 test scenarios)
- Read dev_docs/01-architecture/decisions/005-confidence-driven-investigation.md (expected outcomes)
- Read dev_docs/DEVELOPMENT_WORKFLOW.md (testing standards)

OBJECTIVE:
Continuously validate implementations from Agents 1, 2, 3 as they complete. Ensure no regressions, meet performance targets, and catch integration issues early.

WORKING DIRECTORY: test_sandbox/omnara (use real codebase for realistic testing)

EXECUTION:
Follow dev_docs/03-implementation/PHASE_0_PARALLEL_IMPLEMENTATION_PLAN.md § "Agent 4" checkpoints 1-5.

Your job is to:
1. Test each agent's output as they complete checkpoints
2. Report pass/fail + performance metrics
3. Coordinate with me to flag issues to other agents
4. STOP after each checkpoint and wait for my approval

Start with setting up test infrastructure, then run Checkpoint 1 once Agent 1 completes their Checkpoint 2.

Ready? Let's set up the test framework.
```

---

## Manager Agent Workflow (Human-Driven)

**Your Role:** Orchestrate, coordinate, integrate, and resolve conflicts

---

### Phase 1: Initialization (Day 1)

**Tasks:**
1. **Read this document completely** (you are here)
2. **Verify test repository is initialized:**
   ```bash
   cd test_sandbox/omnara
   ls -la  # Should have source files
   cd ../..
   ```
3. **Open 4 Claude Code sessions** (in addition to this manager session):
   - Session 1: Root directory (`/Users/rohankatakam/Documents/brain/coderisk-go`)
   - Session 2: Root directory
   - Session 3: Root directory
   - Session 4: Test sandbox (`/Users/rohankatakam/Documents/brain/coderisk-go/test_sandbox/omnara`)

4. **Paste initial prompts** (from this document) into each agent session
5. **Agents 1, 2, 4 start immediately** (independent work)
6. **Agent 3 waits** for Agent 1 to complete Phase 0 types

---

### Phase 2: Active Monitoring (Weeks 1-3)

**Daily Tasks:**
1. **Check each agent's progress** (read their latest message)
2. **When an agent reports checkpoint completion:**
   - Review test results (pass/fail counts, coverage)
   - Review sample outputs (does it make sense?)
   - Ask clarifying questions if needed
   - **Approve** (agent continues) or **Request changes** (agent fixes)

3. **Coordinate handoffs:**
   - When Agent 1 completes Checkpoint 4 → Tell Agent 3 to start
   - When Agent 1 completes Checkpoint 2 → Tell Agent 4 to run Checkpoint 1
   - When Agent 2 completes → Tell Agent 4 to run Checkpoint 2
   - When Agent 3 completes → Tell Agent 4 to run Checkpoint 3

4. **Resolve conflicts:**
   - If Agent 4 finds bugs in Agent 1's code → Tell Agent 1 to fix
   - If Agents 1 and 2 modify same file → Coordinate merge strategy
   - If performance targets not met → Adjust implementation or targets

---

### Phase 3: Integration (Week 3)

**Tasks:**
1. **All agents report final completion** (all checkpoints done)
2. **Collect outputs from each agent:**
   - Agent 1: Phase 0 implementation summary
   - Agent 2: Adaptive configs summary
   - Agent 3: Confidence loop summary
   - Agent 4: Final test report + performance benchmarks

3. **In this manager session, ask Claude to:**
   ```
   All 4 agents have completed their work. Here are their final outputs:

   AGENT 1 (Phase 0):
   [paste Agent 1's final summary]

   AGENT 2 (Adaptive Configs):
   [paste Agent 2's final summary]

   AGENT 3 (Confidence Loop):
   [paste Agent 3's final summary]

   AGENT 4 (Testing):
   [paste Agent 4's test report]

   Please:
   1. Identify any integration gaps or conflicts
   2. Create a final integration checklist
   3. Guide me through merging all work
   4. Run final validation tests
   5. Generate deployment readiness report
   ```

4. **Follow integration instructions** from manager Claude
5. **Run final validation:**
   ```bash
   go test ./...
   go build -o bin/crisk ./cmd/crisk
   ./bin/crisk check <test-file>  # Verify end-to-end
   ```

6. **Document completion:**
   - Update [status.md](status.md) with completed components
   - Create performance report in [testing/](testing/)
   - Celebrate! 🎉

---

### Phase 4: Post-Implementation (Week 4)

**Tasks:**
1. **Run comprehensive test suite** on real repositories
2. **Measure actual performance:**
   - False positive rate (compare before/after)
   - Average latency (compare before/after)
   - Insight quality (manual review of recommendations)

3. **Document results** in `dev_docs/03-implementation/testing/PHASE_0_RESULTS.md`
4. **Update ADR-005 status** from "Proposed" to "Accepted"
5. **Plan next phase:** ARC intelligence integration (once we have mined data)

---

## Success Criteria

### Performance Targets (Without ARC)

**Must achieve BEFORE proceeding to ARC integration:**

| Metric | Baseline | Target | Measurement |
|--------|----------|--------|-------------|
| **False Positive Rate** | 50% | ≤15% | User feedback on test scenarios |
| **Average Latency** | 2,500ms | ≤700ms | Weighted average across test suite |
| **Phase 0 Coverage** | 0% | 80%+ | % of checks benefiting from Phase 0 |
| **Early Stop Rate** | 0% | 40%+ | % of investigations stopping <3 hops |
| **Test Coverage** | TBD | 80%+ | Go coverage on new code |
| **Regression Tests** | TBD | 100% pass | All existing tests still pass |

### Quality Gates

**Phase 0 must:**
- ✅ Correctly identify security changes (100% accuracy on test scenarios)
- ✅ Skip documentation-only changes (100% accuracy)
- ✅ Detect production configs (100% accuracy)
- ✅ Complete in <50ms average

**Adaptive Configs must:**
- ✅ Correctly infer domain for omnara test repo
- ✅ Select appropriate thresholds for 3+ domain types
- ✅ Reduce false positives by ≥20% compared to fixed thresholds

**Confidence Loop must:**
- ✅ Stop early (hop 1-2) when LOW risk is obvious (≥30% of cases)
- ✅ Continue investigation (hop 4-5) when HIGH risk needs validation (≥10% of cases)
- ✅ Improve average latency by ≥30%

---

## Risk Mitigation

### Potential Issues and Solutions

**Issue 1: Agents produce conflicting code**
- **Mitigation:** Agents work in separate modules (phase0/, config/, agent/)
- **If conflict:** Manager resolves by reviewing both implementations, choosing better approach
- **Prevention:** Clear module boundaries in this plan

**Issue 2: Tests fail due to integration gaps**
- **Mitigation:** Agent 4 runs continuous integration tests
- **If failure:** Agent 4 reports to manager → Manager assigns fix to responsible agent
- **Prevention:** Frequent checkpoints with manager approval

**Issue 3: Performance targets not met**
- **Mitigation:** Agent 4 tracks performance at each checkpoint
- **If below target:** Manager adjusts implementation (e.g., increase Phase 0 skip rate, adjust confidence threshold)
- **Prevention:** Incremental validation throughout implementation

**Issue 4: Agent gets stuck or goes off-track**
- **Mitigation:** Human-in-loop checkpoints every 2-3 days
- **If stuck:** Manager reviews progress, provides guidance, or re-prompts agent
- **Prevention:** Detailed checkpoint instructions in this plan

**Issue 5: Scope creep (agents implement beyond plan)**
- **Mitigation:** Clear scope definitions in each agent section
- **If occurring:** Manager redirects agent back to checkpoint objectives
- **Prevention:** Emphasize "follow the plan" in initial prompts

---

## Next Steps After This Implementation

### Once Phase 0 + Adaptive System is Complete and Validated:

**Immediate (Week 4):**
1. Deploy to staging environment
2. Run on real repositories (not just test_sandbox)
3. Collect user feedback on false positive reduction
4. Document actual performance metrics

**Short-term (Month 2):**
1. Iterate on thresholds based on real-world data
2. Add more domain configs if needed (ML, CLI, etc.)
3. Fine-tune confidence threshold (0.85 may need adjustment)
4. Implement A/B testing framework for continuous improvement

**Medium-term (Month 3+):**
1. **Begin ARC intelligence integration** (requires mined data)
2. Add hybrid pattern recombination (from [arc_intelligence_architecture.md](../01-architecture/arc_intelligence_architecture.md))
3. Implement incident attribution pipeline
4. Build federated pattern learning

**The optimized foundation (Phase 0 + Adaptive) will make ARC integration much more effective.**

---

## Appendix: Quick Reference

### Key Files Modified by Each Agent

**Agent 1 (Phase 0):**
- `internal/analysis/phase0/*.go` (NEW)
- `cmd/crisk/check.go` (MODIFIED)
- `internal/investigation/risk_assessment.go` (MODIFIED)

**Agent 2 (Adaptive Configs):**
- `internal/analysis/config/*.go` (NEW)
- `internal/investigation/baseline.go` (MODIFIED)

**Agent 3 (Confidence Loop):**
- `internal/agent/confidence.go` (NEW)
- `internal/agent/breakthroughs.go` (NEW)
- `internal/agent/investigator.go` (MODIFIED)
- `internal/agent/prompts/*.go` (MODIFIED)

**Agent 4 (Testing):**
- `test/e2e/*.go` (NEW)
- `test/benchmark/*.go` (NEW)
- `test/integration/*.sh` (NEW)

### Estimated Effort by Agent

| Agent | Work Days | Calendar Days | Complexity |
|-------|-----------|---------------|------------|
| Agent 1 | 10-12 | 12-15 | Medium (5 checkpoints) |
| Agent 2 | 5-7 | 7-10 | Low (4 checkpoints) |
| Agent 3 | 5-7 | 7-10 | Medium (4 checkpoints, depends on Agent 1) |
| Agent 4 | 15-20 | 20-25 | High (5 checkpoints, continuous) |
| **Total** | **35-46 work days** | **2-3 weeks with 4 parallel agents** | **Medium-High** |

### Communication Protocol

**Agent → Manager:**
- After each checkpoint: "Checkpoint X complete. Tests: Y/Z pass. Coverage: W%. Ready for review."
- If blocked: "Blocked on [issue]. Need guidance on [specific question]."
- If dependency needed: "Waiting for Agent X to complete [checkpoint] before proceeding."

**Manager → Agent:**
- Approval: "Approved. Proceed to next checkpoint."
- Request changes: "Please fix [specific issue] before continuing."
- Coordination: "Agent X has completed [checkpoint]. You can now start [dependent work]."

---

## Final Checklist for Manager (You)

**Before starting:**
- [ ] Read this entire document
- [ ] Verify test_sandbox/omnara has test files
- [ ] Open 5 Claude Code sessions (1 manager + 4 agents)
- [ ] Verify all agents can access dev_docs/

**During execution:**
- [ ] Review each checkpoint completion
- [ ] Approve or request changes within 1 day
- [ ] Coordinate Agent 1 → Agent 3 handoff
- [ ] Monitor Agent 4 test results continuously
- [ ] Resolve conflicts between agents if needed

**After completion:**
- [ ] Collect all agent outputs
- [ ] Run integration with manager Claude
- [ ] Validate all tests pass
- [ ] Measure performance against targets
- [ ] Document results
- [ ] Update ADR-005 status

---

**You are now ready to orchestrate the parallel implementation. Good luck! 🚀**
