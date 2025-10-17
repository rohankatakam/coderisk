# Modification Types & Comprehensive Testing Strategy

**Purpose:** Taxonomy of code modification types, impact assessment framework, and expanded testing scenarios
**Last Updated:** October 10, 2025
**Audience:** Developers, QA engineers, AI assistants
**Status:** Design document for test expansion

> **Cross-reference:** This document extends [INTEGRATION_TEST_STRATEGY.md](INTEGRATION_TEST_STRATEGY.md) with modification-aware testing and complements [agentic_design.md](../../01-architecture/agentic_design.md) investigation workflow.

---

## Table of Contents

1. [Modification Type Taxonomy](#1-modification-type-taxonomy)
2. [Impact Assessment Framework](#2-impact-assessment-framework)
3. [Pre-Planning Stage Analysis](#3-pre-planning-stage-analysis)
4. [Expanded Testing Scenarios](#4-expanded-testing-scenarios)
5. [Implementation Recommendations](#5-implementation-recommendations)

---

## 1. Modification Type Taxonomy

### 1.1. Core Modification Types

We identify **10 distinct modification types** based on their structural, behavioral, and contextual characteristics:

#### Type 1: **Structural Changes** - Code Architecture Modifications

**Definition:** Changes to file/module organization, imports, dependencies, or class hierarchies

**Characteristics:**
- Affects graph topology (adds/removes nodes or edges)
- High coupling impact potential
- Often requires coordinated multi-file changes

**Subtypes:**
- **1A: Refactoring** - Moving code without changing behavior
  - File/directory restructuring
  - Class extraction/merging
  - Function decomposition
- **1B: Dependency Changes** - Adding/removing imports
  - New library dependencies
  - Internal module reorganization
  - Dependency injection pattern changes
- **1C: API Surface Changes** - Public interface modifications
  - Function signature changes
  - Class interface additions/removals
  - Module export changes

**Risk Indicators:**
- High structural coupling (>10 dependents)
- Changes to central modules (high betweenness centrality)
- API breakage potential (public vs. private scope)

**Typical Impact:** **HIGH** - Affects multiple files through structural dependencies

---

#### Type 2: **Behavioral Changes** - Logic & Algorithm Modifications

**Definition:** Changes to business logic, algorithms, control flow, or computational behavior

**Characteristics:**
- Affects runtime behavior
- Requires understanding of business domain
- High potential for regression bugs

**Subtypes:**
- **2A: Core Logic Changes** - Business rule modifications
  - Conditional logic changes (if/switch statements)
  - Loop modifications
  - State machine transitions
- **2B: Algorithm Changes** - Computational approach modifications
  - Data structure changes
  - Performance optimizations
  - Algorithm replacements
- **2C: Data Flow Changes** - How data moves through system
  - Variable assignments
  - Function call chains
  - Data transformation pipelines

**Risk Indicators:**
- Low test coverage (<0.3)
- Complex cyclomatic complexity (>10)
- Historical incident linkage

**Typical Impact:** **MODERATE to HIGH** - Depends on test coverage and complexity

---

#### Type 3: **Configuration Changes** - Settings & Parameters

**Definition:** Changes to configuration files, environment variables, constants, or feature flags

**Characteristics:**
- Often runtime-dependent
- May not be validated until deployment
- Can have cascading effects across environments

**Subtypes:**
- **3A: Environment Configuration** - Deployment-specific settings
  - Environment variables (`.env` files)
  - Infrastructure config (Docker, K8s)
  - Database connection strings
- **3B: Application Configuration** - Runtime behavior settings
  - Feature flags
  - Thresholds and limits
  - API keys and credentials
- **3C: Build Configuration** - Compilation/bundling settings
  - Package dependencies (`package.json`, `go.mod`)
  - Build scripts and toolchain config
  - Compiler flags

**Risk Indicators:**
- Changes to production environment configs
- Database schema-related configs
- Security-sensitive values (API keys, secrets)

**Typical Impact:** **LOW to CRITICAL** - Highly environment-dependent

---

#### Type 4: **Interface Changes** - Cross-System Boundaries

**Definition:** Changes to APIs, database schemas, message formats, or external integrations

**Characteristics:**
- Affects system boundaries
- Backward compatibility concerns
- Often requires coordinated deployments

**Subtypes:**
- **4A: API Changes** - HTTP endpoints, RPC interfaces
  - REST/GraphQL schema changes
  - Request/response format modifications
  - Versioning changes
- **4B: Database Schema Changes** - Data model modifications
  - Table/column additions/deletions
  - Constraint changes
  - Migration scripts
- **4C: Message Format Changes** - Event/queue payloads
  - Message broker schemas
  - WebSocket protocols
  - Inter-service contracts

**Risk Indicators:**
- Breaking changes (non-backward compatible)
- Missing migration scripts
  - Lack of versioning strategy

**Typical Impact:** **HIGH to CRITICAL** - Can break production systems

---

#### Type 5: **Testing Changes** - Test Code Modifications

**Definition:** Changes to test files, test fixtures, mocks, or testing infrastructure

**Characteristics:**
- Should not affect production behavior
- Validates other changes
- Can hide bugs if tests are weakened

**Subtypes:**
- **5A: Test Coverage Expansion** - Adding new tests
  - Unit test additions
  - Integration test additions
  - E2E test additions
- **5B: Test Maintenance** - Updating existing tests
  - Fixing broken tests
  - Updating assertions
  - Refactoring test code
- **5C: Test Infrastructure** - Testing tooling changes
  - Test framework updates
  - Mock/stub changes
  - CI/CD pipeline modifications

**Risk Indicators:**
- **Positive:** Test additions to previously uncovered code
- **Negative:** Test deletions or weakened assertions
- Flaky test fixes without addressing root cause

**Typical Impact:** **LOW** (if adding tests) to **MODERATE** (if weakening tests)

---

#### Type 6: **Documentation Changes** - Non-Code Content

**Definition:** Changes to comments, README files, API documentation, or inline docs

**Characteristics:**
- Zero runtime impact
- Improves maintainability
- Can indicate misunderstanding if docs diverge from code

**Subtypes:**
- **6A: Inline Documentation** - Code comments
  - Function/class docstrings
  - Inline comments
  - Type annotations
- **6B: External Documentation** - Separate doc files
  - README files
  - API documentation
  - Architecture diagrams
- **6C: Documentation Infrastructure** - Doc generation tools
  - Swagger/OpenAPI specs
  - Doc site generators
  - Style guides

**Risk Indicators:**
- **Negative:** Docs that contradict code behavior
- **Positive:** Documentation for previously undocumented complex code

**Typical Impact:** **VERY LOW** - Documentation-only changes are safe

---

#### Type 7: **Temporal-Pattern Changes** - Hotspot Modifications

**Definition:** Changes to files with high churn rates, recent incidents, or frequent co-changes

**Characteristics:**
- Location-based risk (where, not what)
- Historical context matters
- Often indicates ongoing issues

**Subtypes:**
- **7A: Hotspot Files** - High churn locations
  - Files changed >10 times in last 30 days
  - "God objects" with many responsibilities
  - Configuration files changed frequently
- **7B: Incident-Prone Files** - Historical bug locations
  - Files linked to >2 incidents
  - Recently fixed bugs (<30 days)
  - Files in incident blast radius
- **7C: Co-Change Clusters** - Tightly coupled file groups
  - Files with >0.7 co-change frequency
  - Always-together file pairs
  - Temporal coupling indicators

**Risk Indicators:**
- High co-change frequency (>0.7)
- Recent incident linkage (<30 days)
- Ownership churn (multiple owners in 90 days)

**Typical Impact:** **MODERATE to HIGH** - Historical patterns predict future issues

---

#### Type 8: **Ownership Changes** - Code Stewardship Transitions

**Definition:** Changes made by new contributors or to files outside developer's usual domain

**Characteristics:**
- Knowledge-based risk
- Familiarity matters
- Onboarding concerns

**Subtypes:**
- **8A: New Contributor Changes** - First-time changes
  - First commit to repository
  - First change to specific subsystem
  - Onboarding period (<3 months)
- **8B: Cross-Domain Changes** - Outside usual area
  - Backend dev modifying frontend
  - Database changes by API developers
  - Infrastructure changes by app developers
- **8C: Ownership Transitions** - Primary maintainer changes
  - Code owner reassignment
  - Team handoffs
  - Bus factor concerns

**Risk Indicators:**
- No prior commits to file
- Owner changed in last 30 days
- Complex code (high cyclomatic complexity) + new owner

**Typical Impact:** **MODERATE** - Increases review scrutiny needs

---

#### Type 9: **Security-Sensitive Changes** - Critical Path Modifications

**Definition:** Changes to authentication, authorization, data validation, or sensitive data handling

**Characteristics:**
- High consequence of failure
- Requires security review
- Compliance concerns

**Subtypes:**
- **9A: Authentication Changes** - Identity verification
  - Login/logout logic
  - Session management
  - Token generation/validation
- **9B: Authorization Changes** - Access control
  - Permission checks
  - Role-based access control (RBAC)
  - Resource ownership validation
- **9C: Data Protection Changes** - Sensitive data handling
  - Encryption/decryption
  - PII handling
  - Data sanitization

**Risk Indicators:**
- Changes to security-critical files (auth, permissions)
- Addition of new data access paths
- Modification of validation logic

**Typical Impact:** **CRITICAL** - Security bugs have severe consequences

---

#### Type 10: **Performance-Critical Changes** - Optimization & Scaling

**Definition:** Changes affecting system performance, resource usage, or scalability

**Characteristics:**
- Non-functional concerns
- May require load testing
- Can introduce subtle bugs

**Subtypes:**
- **10A: Algorithm Optimization** - Computational efficiency
  - O(n²) → O(n log n) improvements
  - Caching implementations
  - Query optimizations
- **10B: Resource Management** - Memory/CPU usage
  - Connection pooling
  - Memory leak fixes
  - Batch processing
- **10C: Concurrency Changes** - Parallel execution
  - Threading/goroutine changes
  - Lock modifications
  - Async/await patterns

**Risk Indicators:**
- Changes to hot paths (profiler data)
- Concurrency primitives (locks, channels)
- Database query modifications

**Typical Impact:** **MODERATE to HIGH** - Performance bugs are hard to detect

---

### 1.2. Modification Type Detection Logic

**Implementation Strategy:**

```
For each changed file in git diff:
  1. Analyze diff hunks
  2. Extract modification signals
  3. Classify into primary and secondary types
  4. Calculate type-specific risk score
  5. Aggregate to file-level risk

Signals to extract from diff:
  - Lines added/deleted/modified
  - Import statement changes → Type 1B
  - Function signature changes → Type 1C
  - Control flow changes (if/for/while) → Type 2A
  - Configuration file patterns → Type 3
  - API route definitions → Type 4A
  - Schema migration files → Type 4B
  - Test file naming patterns → Type 5
  - Comment-only changes → Type 6
  - File path analysis → Type 7 (hotspots)
  - Git author history → Type 8
  - Security keyword matches (auth, crypto, validate) → Type 9
  - Performance keyword matches (cache, pool, async) → Type 10
```

**Heuristics:**

| Signal | Detection Pattern | Type |
|--------|------------------|------|
| Import changes | `import`, `require`, `use` statements | 1B |
| Function signature | `def`, `function`, `func` with param changes | 1C |
| Control flow | `if`, `for`, `while`, `switch` statements | 2A |
| Config files | `.env`, `config.json`, `.yml` files | 3 |
| API routes | `@app.route`, `router.`, `http.Handle` | 4A |
| Schema changes | `migrations/`, `*.sql`, `schema.ts` | 4B |
| Test files | `*_test.py`, `*.test.js`, `test_*.go` | 5 |
| Comments only | All hunks start with `#`, `//`, `/*` | 6 |
| Auth keywords | `login`, `authenticate`, `authorize`, `session` | 9 |
| Performance keywords | `cache`, `pool`, `async`, `concurrent` | 10 |

---

## 2. Impact Assessment Framework

### 2.1. Risk Calculation per Modification Type

Each modification type has **intrinsic risk characteristics** that combine with **contextual factors** (coupling, coverage, incidents).

**Base Risk Scores (0.0 - 1.0):**

| Type | Base Risk | Rationale |
|------|-----------|-----------|
| Type 1 (Structural) | 0.7 | High blast radius potential |
| Type 2 (Behavioral) | 0.8 | Direct logic changes |
| Type 3 (Configuration) | 0.5 | Environment-dependent |
| Type 4 (Interface) | 0.9 | Cross-system impacts |
| Type 5 (Testing) | 0.2 | Should be safe (if adding tests) |
| Type 6 (Documentation) | 0.1 | Zero runtime impact |
| Type 7 (Temporal) | 0.6 | Historical patterns |
| Type 8 (Ownership) | 0.4 | Knowledge risk |
| Type 9 (Security) | 1.0 | Critical consequences |
| Type 10 (Performance) | 0.7 | Subtle bug potential |

**Contextual Multipliers:**

```
final_risk = base_risk × coupling_multiplier × coverage_multiplier × incident_multiplier

Coupling multiplier:
  - coupling ≤ 5: 1.0 (no change)
  - 5 < coupling ≤ 10: 1.2 (20% increase)
  - coupling > 10: 1.5 (50% increase)

Coverage multiplier:
  - coverage ≥ 0.7: 0.8 (20% reduction)
  - 0.3 ≤ coverage < 0.7: 1.0 (no change)
  - coverage < 0.3: 1.3 (30% increase)

Incident multiplier:
  - No incidents: 1.0
  - 1-2 incidents: 1.2 (20% increase)
  - >2 incidents: 1.5 (50% increase)
```

**Example Calculation:**

```
Scenario: Behavioral change (Type 2) to auth.py

Base risk: 0.8 (Type 2 behavioral change)
Coupling: 15 files depend on auth.py → multiplier: 1.5
Test coverage: 0.25 → multiplier: 1.3
Incidents: 1 linked incident → multiplier: 1.2

Final risk = 0.8 × 1.5 × 1.3 × 1.2 = 1.872 (capped at 1.0) = CRITICAL
```

### 2.2. Multi-Type Changes

Many commits contain **multiple modification types**. The final risk is:

```
final_risk = MAX(type_risks) + Σ(other_type_risks × 0.3)

Rationale:
  - Highest risk type dominates
  - Additional types add 30% of their individual risk
  - Capped at 1.0 (CRITICAL)
```

**Example:**

```
Commit modifies:
  - auth.py: Type 2 (behavioral) → risk: 0.95
  - auth_test.py: Type 5 (testing) → risk: 0.20
  - auth.md: Type 6 (documentation) → risk: 0.10

Final risk = MAX(0.95, 0.20, 0.10) + (0.20 + 0.10) × 0.3
           = 0.95 + 0.09
           = 1.04 → capped at 1.0 (CRITICAL)
```

### 2.3. Impact Categorization

**Impact Levels:**

| Final Risk | Impact | Description |
|------------|--------|-------------|
| 0.0 - 0.25 | **LOW** | Safe changes, low review burden |
| 0.25 - 0.50 | **MODERATE** | Standard review process |
| 0.50 - 0.75 | **HIGH** | Enhanced review, add tests |
| 0.75 - 1.0 | **CRITICAL** | Security review, thorough testing |

---

## 3. Pre-Planning Stage Analysis

### 3.1. Should We Add a Pre-Planning Stage?

**Question:** Should [agentic_design.md](../../01-architecture/agentic_design.md) include a **Phase 0** that analyzes git diff and categorizes modification types before Phase 1 baseline assessment?

**Current Architecture:**

```
User runs: crisk check

→ Phase 1: Baseline Assessment (Tier 1 metrics)
  ├─ Structural coupling
  ├─ Temporal co-change
  └─ Test coverage ratio

→ Decision: LOW risk? → Return
            HIGH risk? → Proceed to Phase 2

→ Phase 2: LLM Investigation (Tier 2 metrics)
  └─ Ownership, incidents, synthesis
```

**Proposed Pre-Planning Stage:**

```
User runs: crisk check

→ Phase 0: Modification Type Analysis (NEW)
  ├─ Parse git diff
  ├─ Detect modification types (1-10)
  ├─ Calculate base risk per type
  └─ Pre-filter files by intrinsic risk

→ Phase 1: Baseline Assessment (Tier 1 metrics)
  └─ Enhanced with modification type context

→ Phase 2: LLM Investigation (Tier 2 metrics)
  └─ Type-aware investigation prompts
```

### 3.2. Analysis: Is Phase 0 Needed?

**Arguments FOR Phase 0:**

✅ **Type-Specific Risk Awareness**
- Security changes (Type 9) should **always escalate** to Phase 2, regardless of coupling
- Documentation changes (Type 6) could **skip Phase 1 entirely** (zero risk)
- Configuration changes (Type 3) need **environment-specific checks**

✅ **Better Phase 2 Prompts**
- LLM gets modification type context: "This is a security-sensitive authentication change"
- Enables type-specific investigation: "Check for authentication bypass vulnerabilities"
- Reduces generic LLM prompts, increases specificity

✅ **Pre-Filtering Low-Risk Changes**
- Documentation-only changes don't need graph queries
- Test-only changes skip coupling analysis
- Saves Phase 1 computation for ~20% of changes

✅ **Explainability**
- Users understand why risk is HIGH: "Security-sensitive + high coupling"
- Risk breakdown: "Base risk: 0.9 (Type 9), multiplied by coupling: 1.5"

**Arguments AGAINST Phase 0:**

❌ **Already Partially Covered**
- Phase 1 metrics implicitly capture some types:
  - Structural changes → detected by coupling changes
  - Testing changes → detected by test coverage ratio
  - Temporal patterns → detected by co-change frequency
- Adding Phase 0 might be redundant

❌ **Diff Parsing Complexity**
- Requires parsing git diff hunks (line-by-line analysis)
- Language-specific detection (Python imports vs. Go imports)
- Maintenance burden as languages/patterns evolve

❌ **Performance Overhead**
- Phase 0 adds ~50-100ms for diff parsing
- May not justify cost if most changes still need Phase 1
- Only saves time for doc-only/test-only changes (~20%)

❌ **False Positives in Type Detection**
- Keyword matching has noise:
  - `authenticate_user()` function name → Type 9, even if just renaming
  - `cache.clear()` → Type 10, even if trivial
  - Imports added but unused → Type 1, but no real impact
- Could escalate unnecessarily

### 3.3. Recommendation: **Conditional Phase 0**

**Decision:** Add **lightweight Phase 0** for **specific high-value scenarios**:

**Phase 0 should run when:**
1. **Security keyword detection** → Always escalate to Phase 2 (Type 9)
2. **Documentation-only changes** → Skip Phase 1/Phase 2 (Type 6)
3. **Schema migrations detected** → Always escalate to Phase 2 (Type 4B)

**Phase 0 should NOT run when:**
- Mixed changes (code + docs) → Just run Phase 1
- No clear type signals → Rely on Phase 1 metrics
- Fast-path needed (pre-commit hook) → Skip Phase 0 overhead

**Implementation:**

```go
// Phase 0: Fast Type Detection (50ms budget)
func PreAnalyzeChanges(diff GitDiff) PhaseZeroResult {
    // Quick heuristics only
    if diff.IsDocumentationOnly() {
        return PhaseZeroResult{
            Skip: true,
            Reason: "Documentation-only change (Type 6)",
        }
    }

    if diff.ContainsSecurityKeywords() {
        return PhaseZeroResult{
            ForceEscalate: true,
            ModificationType: Type9_Security,
            Reason: "Security-sensitive keywords detected",
        }
    }

    if diff.IsSchemaMigration() {
        return PhaseZeroResult{
            ForceEscalate: true,
            ModificationType: Type4B_Schema,
            Reason: "Database schema migration detected",
        }
    }

    // Default: proceed to Phase 1
    return PhaseZeroResult{
        Skip: false,
        ForceEscalate: false,
        DetectedTypes: detectModificationTypes(diff), // For context only
    }
}
```

**Benefits:**
- ✅ Catches critical cases (security, schema)
- ✅ Skips unnecessary analysis (docs-only)
- ✅ Low overhead (simple keyword matching)
- ✅ Falls back to Phase 1 for complex cases

**Integration with agentic_design.md:**

The current [agentic_design.md](../../01-architecture/agentic_design.md) two-phase architecture is **already well-designed**. Phase 0 should be:
- **Optional enhancement**, not core requirement
- **Documented as future improvement** in agentic_design.md §2.1
- **Implemented incrementally** (start with security keyword detection)

**Update to agentic_design.md:**

Add new section:

```markdown
### 2.1.1. Optional Phase 0: Pre-Analysis (Future Enhancement)

**Goal:** Fast heuristic checks before Phase 1 baseline

**Implementation Status:** Not yet implemented (planned for v2.0)

**Use Cases:**
1. Security keyword detection → Force escalate to Phase 2
2. Documentation-only changes → Skip all risk analysis
3. Schema migrations → Force escalate with specialized prompts

**Budget:** <50ms (keyword matching only, no deep parsing)

**Decision Logic:**
- If security keywords → `ForceEscalatePhase2 = true`
- If docs-only → `SkipAnalysis = true, Risk = LOW`
- Else → Proceed to Phase 1

See [MODIFICATION_TYPES_AND_TESTING.md](../../03-implementation/testing/MODIFICATION_TYPES_AND_TESTING.md) for full modification type taxonomy.
```

---

## 4. Expanded Testing Scenarios

### 4.1. Type-Specific Test Scenarios

Building on [INTEGRATION_TEST_STRATEGY.md](INTEGRATION_TEST_STRATEGY.md), we add **modification-type-aware tests**:

---

#### Scenario 5: Structural Refactoring (Type 1) 🏗️

**Objective:** Validate risk assessment for file/module reorganization

**Setup:**
- Repository: omnara-ai/omnara
- Change: Move `apps/web/src/lib/auth/session.ts` → `packages/auth/src/session.ts`
- Update imports in 12 dependent files
- Expected: HIGH risk (high coupling, multi-file change)

**Test Flow:**

**Step 1: Initial Assessment**
```bash
# Make refactoring changes (simulate moving file)
crisk check packages/auth/src/session.ts apps/web/src/lib/utils/helper.ts (... 10 more files)
```

**Expected Phase 1 Output:**
```
Risk Level: ⚠️  HIGH

Modification Type: STRUCTURAL (Type 1B - Dependency Changes)
Base Risk: 0.7 (structural changes)

Metrics:
  Coupling: 12 files affected (threshold: 10) ❌
  Co-change frequency: 0.65 (multi-file refactoring pattern)
  Test coverage: 0.55 (moderate, but multi-file)

Phase 1 completed in 45ms

Escalating to Phase 2 (multi-file structural change detected)...
```

**Expected Phase 2 Investigation:**
```
Key Evidence:
1. [structural] 12-file refactoring detected
2. [modification_type] Type 1B: Dependency reorganization
3. [temporal] Session.ts has 0.82 co-change frequency with auth route
4. [recommendation] Run full integration test suite

Risk Level: HIGH (confidence: 0.92)

Recommendations:
  Critical:
    1. Run full test suite (structural changes affect many files)
    2. Verify all import paths are updated correctly
    3. Check for circular dependency introduction

  High Priority:
    1. Add integration test for authentication flow
    2. Deploy to staging environment before production
```

**Pass Criteria:**
- ✅ Type 1 (Structural) correctly identified
- ✅ Multi-file change triggers HIGH risk
- ✅ Specific refactoring recommendations provided
- ✅ Coupling metric reflects all affected files

---

#### Scenario 6: Configuration Change (Type 3) ⚙️

**Objective:** Test environment-specific risk assessment

**Setup:**
- Change: Update `DATABASE_URL` in `.env.production`
- Expected: Depends on value (HIGH if production, LOW if dev)

**Test Flow:**

**Test 6A: Production Environment Config**
```bash
# Change production database URL
crisk check .env.production
```

**Expected Output:**
```
Risk Level: ⚠️  CRITICAL

Modification Type: CONFIGURATION (Type 3A - Environment Configuration)
Base Risk: 0.5 → Escalated to 1.0 (production environment detected)

Configuration Analysis:
  Environment: PRODUCTION (high-risk environment)
  Changed values: DATABASE_URL, REDIS_URL
  Sensitive: YES (database credentials)

Recommendations:
  Critical:
    1. Verify connection string syntax
    2. Test database connectivity in staging first
    3. Have rollback plan ready
    4. Notify on-call team before deployment
```

**Test 6B: Development Environment Config**
```bash
# Change dev database URL
crisk check .env.development
```

**Expected Output:**
```
Risk Level: ✅ LOW

Modification Type: CONFIGURATION (Type 3A - Environment Configuration)
Base Risk: 0.5 → Reduced to 0.2 (development environment)

Configuration Analysis:
  Environment: DEVELOPMENT (low-risk environment)
  Changed values: DATABASE_URL
  Sensitive: NO (local development only)

Recommendations:
  Standard:
    1. Restart development server to apply changes
```

**Pass Criteria:**
- ✅ Environment detection (production vs. development)
- ✅ Risk escalation for production configs
- ✅ Sensitive value detection (credentials, API keys)

---

#### Scenario 7: Security-Sensitive Change (Type 9) 🔐

**Objective:** Test security keyword detection and mandatory escalation

**Setup:**
- Change: Modify authentication logic in `auth/[...nextauth]/route.ts`
- Keywords: `login`, `session`, `validate`
- Expected: CRITICAL risk, always escalate

**Test Flow:**

```bash
# Modify authentication function
crisk check apps/web/src/app/api/auth/[...nextauth]/route.ts
```

**Expected Phase 0 Output (NEW - Pre-Analysis):**
```
⚠️  Security-sensitive keywords detected: login, authenticate, session

Phase 0: Pre-Analysis
  Modification Type: SECURITY (Type 9A - Authentication)
  Base Risk: 1.0 (CRITICAL)
  Force Escalate: YES (security changes always require LLM review)

Proceeding to Phase 2 (skipping Phase 1 baseline)...
```

**Expected Phase 2 Output:**
```
Risk Level: 🔴 CRITICAL

Modification Type: SECURITY (Type 9A - Authentication Changes)

Security Analysis:
  - Changed function: authOptions (session configuration)
  - Impact: All authenticated users
  - Historical: 1 critical incident (session timeout bug)

Key Evidence:
1. [security] Authentication logic modification
2. [incidents] Similar incident: "NextAuth session timeout" (14 days ago)
3. [coupling] 18 files depend on this authentication route
4. [recommendation] Security review required

Recommendations:
  Critical (must do before commit):
    1. Conduct security review with security team
    2. Test authentication flow with multiple user roles
    3. Verify session timeout behavior
    4. Check for authentication bypass vulnerabilities
    5. Add security regression tests

  High Priority:
    1. Deploy to staging with monitoring
    2. Review incident #INCIDENT_1 to prevent regression
```

**Pass Criteria:**
- ✅ Security keyword detection (Phase 0)
- ✅ Forced escalation to Phase 2
- ✅ CRITICAL risk level assigned
- ✅ Security-specific recommendations

---

#### Scenario 8: Performance Optimization (Type 10) ⚡

**Objective:** Test performance-critical change detection

**Setup:**
- Change: Add caching to database query function
- Keywords: `cache`, `memoize`, `Redis`
- Expected: MODERATE to HIGH (performance changes need testing)

**Test Flow:**

```bash
# Add caching logic
crisk check apps/web/src/lib/db/queries.ts
```

**Expected Output:**
```
Risk Level: ⚠️  HIGH

Modification Type: PERFORMANCE (Type 10A - Algorithm Optimization)
Base Risk: 0.7 (performance-critical)

Performance Analysis:
  Changed function: getUserData (database query)
  Optimization: Added Redis caching layer
  Potential issues: Cache invalidation, stale data

Metrics:
  Coupling: 8 files call getUserData
  Test coverage: 0.4 (needs cache-specific tests)

Recommendations:
  Critical:
    1. Add tests for cache hit/miss scenarios
    2. Add tests for cache invalidation logic
    3. Verify cache TTL is appropriate

  High Priority:
    1. Load test to verify performance improvement
    2. Monitor cache hit rate in staging
    3. Add cache metrics/observability
```

**Pass Criteria:**
- ✅ Performance keyword detection
- ✅ Cache-specific recommendations
- ✅ Test coverage for cache logic
- ✅ Load testing recommendation

---

#### Scenario 9: Multi-Type Change (Combined) 🔀

**Objective:** Test risk aggregation for commits with multiple modification types

**Setup:**
- File 1: `auth.ts` - Type 2 (Behavioral) + Type 9 (Security)
- File 2: `auth_test.ts` - Type 5 (Testing)
- File 3: `auth.md` - Type 6 (Documentation)
- Expected: Risk dominated by highest-risk type (Type 9)

**Test Flow:**

```bash
# Check all files in commit
crisk check auth.ts auth_test.ts auth.md
```

**Expected Output:**
```
Risk Level: 🔴 CRITICAL

Multi-Type Change Detected:
  Primary: Type 9 (Security) - auth.ts
  Secondary: Type 2 (Behavioral) - auth.ts
  Supporting: Type 5 (Testing) - auth_test.ts
  Supporting: Type 6 (Documentation) - auth.md

Risk Calculation:
  auth.ts: 1.0 (Type 9 security change)
  auth_test.ts: 0.2 (Type 5 test addition) × 0.3 = 0.06
  auth.md: 0.1 (Type 6 docs) × 0.3 = 0.03

  Final Risk = MAX(1.0, 0.2, 0.1) + (0.06 + 0.03) = 1.09 → capped at 1.0

Overall Assessment:
  - Security change dominates risk profile
  - Test additions reduce risk slightly (good practice)
  - Documentation additions are positive

Recommendations:
  [Same as Scenario 7 - Security-sensitive]
```

**Pass Criteria:**
- ✅ Multiple types detected
- ✅ Risk aggregation formula applied
- ✅ Highest-risk type dominates
- ✅ Positive changes (tests, docs) acknowledged

---

#### Scenario 10: Documentation-Only Change (Type 6) 📝

**Objective:** Test skip logic for zero-risk changes

**Setup:**
- Change: Update README.md and inline comments
- No code changes
- Expected: Skip Phase 1 and Phase 2, immediate LOW risk

**Test Flow:**

```bash
# Check documentation-only commit
crisk check README.md src/utils/helpers.ts (only comments changed)
```

**Expected Phase 0 Output:**
```
Phase 0: Pre-Analysis

Documentation-Only Change Detected (Type 6)
  Files: README.md, src/utils/helpers.ts (comments only)
  Runtime Impact: ZERO

Skipping Phase 1 and Phase 2 (no risk analysis needed)

Risk Level: ✅ LOW

Assessment: Safe to commit (documentation improvements)
```

**Pass Criteria:**
- ✅ Documentation-only detection
- ✅ Phase 1/Phase 2 skipped
- ✅ Instant LOW risk result
- ✅ Performance <10ms (no graph queries)

---

#### Scenario 11: Ownership Risk (Type 8) 👤

**Objective:** Test new contributor or cross-domain change detection

**Setup:**
- New contributor's first commit to `critical_service.go`
- File has high complexity (cyclomatic complexity: 15)
- Expected: MODERATE to HIGH risk due to unfamiliarity

**Test Flow:**

```bash
# Simulate new contributor
GIT_AUTHOR_NAME="Alice Newbie" GIT_AUTHOR_EMAIL="alice@example.com" \
crisk check internal/services/critical_service.go
```

**Expected Output:**
```
Risk Level: ⚠️  MODERATE → HIGH (ownership factor)

Modification Type: BEHAVIORAL (Type 2) + OWNERSHIP (Type 8A)

Ownership Analysis:
  Current author: Alice Newbie (alice@example.com)
  File history: 0 prior commits by this author
  Primary owner: Bob Expert (bob@example.com) - 45 commits
  Complexity: High (cyclomatic complexity: 15)

Risk Factors:
  - New contributor to this file (ownership risk)
  - High complexity code
  - No prior familiarity

Recommendations:
  Critical:
    1. Request review from Bob Expert (primary owner)
    2. Pair program or knowledge transfer session
    3. Add extra test coverage for changed sections

  High Priority:
    1. Review file documentation
    2. Ensure understanding of side effects
```

**Pass Criteria:**
- ✅ Ownership analysis performed
- ✅ New contributor detected
- ✅ Risk escalation due to unfamiliarity
- ✅ Code owner review recommended

---

#### Scenario 12: Temporal Hotspot (Type 7) 🔥

**Objective:** Test high-churn file detection

**Setup:**
- File changed 15 times in last 30 days
- 2 linked incidents
- High co-change frequency (0.85) with related file
- Expected: HIGH risk due to historical instability

**Test Flow:**

```bash
# Check hotspot file
crisk check apps/web/src/services/payment_processor.ts
```

**Expected Output:**
```
Risk Level: ⚠️  HIGH

Modification Type: TEMPORAL HOTSPOT (Type 7A + 7B)

Temporal Analysis:
  Churn rate: 15 changes in last 30 days (HIGH)
  Incident history: 2 linked incidents
    - INC-123: "Payment timeout" (21 days ago)
    - INC-087: "Double charge bug" (45 days ago)
  Co-change pattern: 0.85 frequency with billing_service.ts

Key Evidence:
1. [temporal] High churn indicates ongoing issues
2. [incidents] Two recent production incidents
3. [coupling] Always changes with billing_service.ts
4. [recommendation] Extra scrutiny needed

Recommendations:
  Critical:
    1. Review incident reports: INC-123, INC-087
    2. Verify changes don't reintroduce previous bugs
    3. Add regression tests for past incidents

  High Priority:
    1. Consider refactoring to stabilize this component
    2. Check if billing_service.ts also needs updating
    3. Add monitoring/alerting for payment failures
```

**Pass Criteria:**
- ✅ Churn rate calculated correctly
- ✅ Incident linkage detected
- ✅ Co-change pattern identified
- ✅ Historical context provided

---

### 4.2. Test Matrix

| Scenario | Type | Risk | Phase 0 | Phase 1 | Phase 2 | Key Validation |
|----------|------|------|---------|---------|---------|----------------|
| 5. Structural Refactoring | 1 | HIGH | Optional | ✅ | ✅ | Multi-file impact |
| 6A. Production Config | 3 | CRITICAL | ✅ (env detect) | ✅ | ✅ | Environment awareness |
| 6B. Dev Config | 3 | LOW | ✅ (env detect) | ✅ | ❌ | Risk reduction |
| 7. Security Change | 9 | CRITICAL | ✅ (keywords) | ❌ Skip | ✅ Force | Keyword detection |
| 8. Performance Optimization | 10 | HIGH | Optional | ✅ | ✅ | Cache-specific tests |
| 9. Multi-Type | 2+5+6+9 | CRITICAL | ✅ | ✅ | ✅ | Risk aggregation |
| 10. Docs-Only | 6 | LOW | ✅ (skip logic) | ❌ Skip | ❌ Skip | Fast-path |
| 11. Ownership Risk | 8 | MODERATE | Optional | ✅ | ✅ | Author analysis |
| 12. Temporal Hotspot | 7 | HIGH | Optional | ✅ | ✅ | Churn + incidents |

---

### 4.3. Performance Benchmarks

**Target Latencies with Phase 0:**

| Scenario | Phase 0 | Phase 1 | Phase 2 | Total | Status |
|----------|---------|---------|---------|-------|--------|
| Docs-only (Type 6) | 10ms | 0ms (skip) | 0ms (skip) | **10ms** | ✅ 20x faster |
| Security (Type 9) | 15ms | 0ms (skip) | 3.5s | **3.5s** | ✅ Same as current |
| Standard (Type 2) | 0ms (skip Phase 0) | 180ms | 0ms | **180ms** | ✅ No overhead |
| High-risk (Type 2+9) | 15ms | 0ms (skip) | 4.0s | **4.0s** | ✅ Acceptable |

**Performance Goals:**
- Phase 0 should complete in <50ms (keyword matching only)
- Docs-only changes should return in <20ms total
- No performance regression for standard checks (Phase 0 is opt-in)

---

## 5. Implementation Recommendations

### 5.1. Incremental Rollout Plan

**Phase 1: Foundation (v1.1)**
- ✅ Already implemented: Phase 1 baseline, Phase 2 investigation
- 🔄 Add modification type taxonomy to documentation
- 🔄 Create test scenarios 5-12
- 🔄 Validate current system handles scenarios without Phase 0

**Phase 2: Basic Type Detection (v1.2)**
- Add `internal/analysis/modification_types.go`
- Implement keyword-based detection:
  - Security keywords (Type 9)
  - Documentation-only detection (Type 6)
  - Configuration file detection (Type 3)
- Add Phase 0 as **opt-in flag**: `crisk check --with-type-analysis`
- Test with scenarios 7, 10, 6A/6B

**Phase 3: Full Type Detection (v1.3)**
- Expand to all 10 modification types
- Integrate with Phase 1 metrics (contextual multipliers)
- Update LLM prompts with type context
- Test with all scenarios (5-12)

**Phase 4: Auto-Enable (v2.0)**
- Make Phase 0 default (opt-out via `--no-type-analysis`)
- Add telemetry for type detection accuracy
- Tune keyword dictionaries based on user feedback

### 5.2. Files to Create/Modify

**New Files:**
1. `internal/analysis/modification_types.go` - Type detection logic
2. `internal/analysis/diff_parser.go` - Git diff parsing
3. `internal/analysis/keyword_matcher.go` - Security/performance keyword matching
4. `test/integration/test_modification_types.sh` - Test scenarios 5-12

**Modified Files:**
1. `cmd/crisk/check.go` - Add Phase 0 integration
2. `internal/agent/investigator.go` - Type-aware LLM prompts
3. `dev_docs/01-architecture/agentic_design.md` - Document Phase 0
4. `dev_docs/03-implementation/testing/INTEGRATION_TEST_STRATEGY.md` - Reference this doc

### 5.3. Testing Strategy

**Unit Tests:**
- Type detection accuracy (keyword matching)
- Multi-type aggregation formula
- Risk calculation per type

**Integration Tests:**
- Scenarios 5-12 in real omnara repository
- Phase 0 performance benchmarks
- Skip logic for docs-only changes

**Validation:**
- False positive rate tracking
- User feedback on type accuracy
- Performance impact measurement

---

## 6. Appendix

### 6.1. Modification Type Quick Reference

| Type | Name | Base Risk | Key Indicators | Phase 0 Action |
|------|------|-----------|----------------|----------------|
| 1 | Structural | 0.7 | Import changes, refactoring | Optional |
| 2 | Behavioral | 0.8 | Logic changes, control flow | Optional |
| 3 | Configuration | 0.5 | `.env`, config files | Check environment |
| 4 | Interface | 0.9 | API, schema, contracts | Optional |
| 5 | Testing | 0.2 | `*_test.*` files | Optional |
| 6 | Documentation | 0.1 | Docs, comments only | **Skip Phase 1/2** |
| 7 | Temporal | 0.6 | Hotspots, incidents | Optional |
| 8 | Ownership | 0.4 | New contributors | Optional |
| 9 | Security | 1.0 | Auth, crypto, validation | **Force escalate** |
| 10 | Performance | 0.7 | Cache, async, optimization | Optional |

### 6.2. Security Keywords (Type 9)

```
Authentication: login, logout, signin, signout, authenticate, auth, session, token, jwt, oauth, saml
Authorization: authorize, permission, role, access, grant, deny, admin, privilege, acl, rbac
Cryptography: encrypt, decrypt, hash, sign, verify, crypto, cipher, key, secret, password
Validation: validate, sanitize, escape, filter, parse, serialize, deserialize
Sensitive Data: pii, ssn, credit_card, personal, private, confidential
```

### 6.3. Performance Keywords (Type 10)

```
Caching: cache, memoize, redis, memcached, cdn
Concurrency: async, await, promise, goroutine, thread, lock, mutex, channel, concurrent, parallel
Optimization: optimize, performance, fast, slow, bottleneck, profile
Resource Management: pool, connection, memory, leak, gc, allocation
Database: query, index, join, transaction, batch
```

---

**Related Documentation:**
- [INTEGRATION_TEST_STRATEGY.md](INTEGRATION_TEST_STRATEGY.md) - Overall testing strategy
- [agentic_design.md](../../01-architecture/agentic_design.md) - Investigation workflow
- [risk_assessment_methodology.md](../../01-architecture/risk_assessment_methodology.md) - Risk calculation details

---

**Last Updated:** October 10, 2025
**Status:** Design document for Phase 0 implementation and test expansion
**Next Steps:**
1. Review with team
2. Implement Phase 0 (opt-in) in v1.2
3. Create test scenarios 5-12
4. Validate modification type detection accuracy
