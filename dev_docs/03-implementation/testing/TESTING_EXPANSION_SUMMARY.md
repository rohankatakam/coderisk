# Testing Expansion Summary

**Purpose:** Executive summary of modification-type testing strategy and recommendations
**Last Updated:** October 10, 2025
**Audience:** Product managers, QA leads, developers
**Status:** Ready for implementation

> **Full Details:** See [MODIFICATION_TYPES_AND_TESTING.md](MODIFICATION_TYPES_AND_TESTING.md) for complete taxonomy and test scenarios.

---

## Executive Summary

We analyzed the CodeRisk testing strategy and identified **10 distinct modification types** that require different testing approaches. This document summarizes:

1. **Key modification types** and their risk profiles
2. **8 new test scenarios** (expanding on the existing 4)
3. **Recommendation on pre-planning stage** (Phase 0)
4. **Implementation priorities**

---

## 1. Modification Type Categories

### High-Priority Types (Always Test Thoroughly)

| Type | Risk | Example | Testing Focus |
|------|------|---------|---------------|
| **Type 9: Security** | CRITICAL | Authentication, authorization, encryption | Security review, bypass testing, regression tests |
| **Type 4: Interface** | HIGH-CRITICAL | API changes, database schema, message formats | Backward compatibility, migration testing, contract validation |
| **Type 2: Behavioral** | MODERATE-HIGH | Business logic, algorithms, control flow | Unit tests, edge cases, regression coverage |
| **Type 1: Structural** | HIGH | Refactoring, dependency changes, API surface | Integration tests, multi-file impact, coupling analysis |

### Medium-Priority Types (Standard Testing)

| Type | Risk | Example | Testing Focus |
|------|------|---------|---------------|
| **Type 10: Performance** | MODERATE-HIGH | Caching, concurrency, optimization | Load testing, resource monitoring, benchmark validation |
| **Type 7: Temporal Hotspot** | MODERATE-HIGH | High-churn files, incident-prone areas | Historical bug review, extra scrutiny, regression tests |
| **Type 8: Ownership** | MODERATE | New contributor changes, cross-domain edits | Code review, knowledge transfer, pairing |
| **Type 3: Configuration** | LOW-CRITICAL | Environment configs, feature flags, build settings | Environment validation, rollback testing, staging deployment |

### Low-Priority Types (Minimal Testing)

| Type | Risk | Example | Testing Focus |
|------|------|---------|---------------|
| **Type 5: Testing** | LOW-MODERATE | Test file additions, test maintenance | Test quality review, avoid weakened assertions |
| **Type 6: Documentation** | VERY LOW | README, comments, API docs | None (skip risk analysis) |

---

## 2. Impact Assessment Framework

### Risk Calculation Formula

```
final_risk = base_risk × coupling_multiplier × coverage_multiplier × incident_multiplier

Where:
  - base_risk: Type-specific (0.1 - 1.0)
  - coupling_multiplier: 1.0 (≤5 deps) to 1.5 (>10 deps)
  - coverage_multiplier: 0.8 (≥70% coverage) to 1.3 (<30% coverage)
  - incident_multiplier: 1.0 (no incidents) to 1.5 (>2 incidents)
```

### Example Calculation

**Scenario:** Security change (Type 9) to `auth.py`
- Base risk: 1.0 (security-critical)
- Coupling: 15 dependents → 1.5x
- Test coverage: 25% → 1.3x
- Incidents: 1 linked → 1.2x
- **Final risk:** 1.0 × 1.5 × 1.3 × 1.2 = **2.34 → capped at 1.0 (CRITICAL)**

---

## 3. Expanded Test Scenarios

### Current Coverage (from INTEGRATION_TEST_STRATEGY.md)

✅ **Scenario 1:** Happy Path - Low Risk Code
✅ **Scenario 2:** High Risk Code - Escalation & Fix Loop
✅ **Scenario 3:** AI Mode JSON Validation
✅ **Scenario 4:** Performance & Timeout Testing

### New Scenarios (Modification-Type Aware)

#### 🏗️ **Scenario 5: Structural Refactoring (Type 1)**
**Test:** Move file, update 12 imports
**Expected:** HIGH risk due to multi-file impact
**Validation:** Coupling metric reflects all affected files, refactoring-specific recommendations

#### ⚙️ **Scenario 6: Configuration Changes (Type 3)**
**Test 6A:** Production `.env` change → CRITICAL risk (environment detection)
**Test 6B:** Development `.env` change → LOW risk (safe environment)
**Validation:** Environment-aware risk assessment, sensitive value detection

#### 🔐 **Scenario 7: Security-Sensitive Changes (Type 9)**
**Test:** Modify authentication logic with security keywords
**Expected:** CRITICAL risk, forced Phase 2 escalation
**Validation:** Keyword detection, security review recommendations, bypass testing suggestions

#### ⚡ **Scenario 8: Performance Optimization (Type 10)**
**Test:** Add Redis caching to database queries
**Expected:** HIGH risk, cache-specific testing needed
**Validation:** Performance keyword detection, cache invalidation tests, load testing recommendations

#### 🔀 **Scenario 9: Multi-Type Changes (Combined)**
**Test:** Single commit with security + behavioral + testing + docs changes
**Expected:** Risk dominated by highest type (security = CRITICAL)
**Validation:** Risk aggregation formula, type prioritization

#### 📝 **Scenario 10: Documentation-Only (Type 6)**
**Test:** README and comment changes only
**Expected:** Skip Phase 1/2, instant LOW risk
**Validation:** Fast-path logic (<20ms total), no graph queries

#### 👤 **Scenario 11: Ownership Risk (Type 8)**
**Test:** New contributor's first change to complex file
**Expected:** MODERATE to HIGH risk, ownership escalation
**Validation:** Author history analysis, code owner review recommendation

#### 🔥 **Scenario 12: Temporal Hotspot (Type 7)**
**Test:** File changed 15× in 30 days with 2 incidents
**Expected:** HIGH risk, historical pattern detection
**Validation:** Churn rate calculation, incident linkage, co-change detection

---

## 4. Pre-Planning Stage (Phase 0) Recommendation

### Current Architecture

```
crisk check → Phase 1 (Tier 1 metrics) → Phase 2 (LLM investigation if HIGH)
```

### Proposed Enhancement (Optional Phase 0)

```
crisk check → Phase 0 (Type Detection) → Phase 1 (Enhanced with type context) → Phase 2
```

### Decision: **Conditional Phase 0** (Not Always Required)

**Recommendation:** Add **lightweight Phase 0** for specific high-value scenarios only:

✅ **Phase 0 SHOULD run when:**
1. **Security keywords detected** (`login`, `auth`, `crypto`) → Force escalate to Phase 2
2. **Documentation-only changes** (no code modified) → Skip Phase 1/2 entirely
3. **Schema migrations** (`*.sql`, `migrations/`) → Force escalate with specialized prompts

❌ **Phase 0 SHOULD NOT run when:**
- Mixed changes (code + docs) → Just run Phase 1 (existing metrics sufficient)
- No clear type signals → Rely on Phase 1 metrics (no overhead)
- Fast-path needed (pre-commit hook) → Skip Phase 0 (latency sensitive)

**Rationale:**
- Current Phase 1 metrics **already detect** most risky changes (coupling, co-change, coverage)
- Phase 0 adds value **only for special cases** (security, docs-only, schema)
- Avoids complexity and false positives from over-aggressive type detection
- <50ms overhead acceptable for high-value scenarios

### Integration with agentic_design.md

**Status:** Current [agentic_design.md](../../01-architecture/agentic_design.md) architecture is **well-designed**.

**Recommendation:** Add Phase 0 as **optional enhancement**, not core requirement:
- Document in agentic_design.md §2.1.1 as "Future Enhancement"
- Implement incrementally (start with security keyword detection)
- Make opt-in via flag: `crisk check --with-type-analysis`

---

## 5. Implementation Priorities

### Phase 1: Immediate (v1.1)
- ✅ Validate current system handles 8 new scenarios without Phase 0
- 🔄 Create test scripts for scenarios 5-12
- 🔄 Document modification types in [MODIFICATION_TYPES_AND_TESTING.md](MODIFICATION_TYPES_AND_TESTING.md)

### Phase 2: Near-Term (v1.2)
- Add basic Phase 0 for **security keywords** and **docs-only** detection
- Implement as opt-in flag: `--with-type-analysis`
- Test scenarios 7, 10 with Phase 0 enabled

### Phase 3: Future (v2.0)
- Expand Phase 0 to all 10 modification types
- Type-aware LLM prompts in Phase 2
- Default-enable Phase 0 (with opt-out)

---

## 6. Expected Benefits

### Testing Improvements
- **+8 new test scenarios** covering edge cases (security, config, ownership, hotspots)
- **Type-specific validation** (e.g., security bypass testing, cache invalidation tests)
- **Better coverage** of real-world change patterns

### Risk Assessment Improvements
- **Security changes** always escalate (no false negatives)
- **Documentation-only changes** skip analysis (faster workflow)
- **Multi-type changes** correctly prioritize highest risk

### Developer Experience
- **Faster feedback** for low-risk changes (docs skip Phase 1/2)
- **More specific recommendations** (e.g., "add cache invalidation tests" vs generic "add tests")
- **Better explainability** ("CRITICAL due to security keywords + high coupling")

---

## 7. Key Metrics to Track

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Phase 0 overhead** | <50ms | Time to detect types (keyword matching) |
| **False positive rate** | <3% | User feedback on type detection accuracy |
| **Coverage of new scenarios** | 100% | All 8 new scenarios passing |
| **Performance regression** | 0% | No slowdown for standard checks |

---

## 8. Next Steps

1. **Review with team** - Validate modification type taxonomy and risk formulas
2. **Create test implementations** - Scenarios 5-12 in `test/integration/`
3. **Implement Phase 0 (opt-in)** - Security keyword detection + docs-only skip logic
4. **Update agentic_design.md** - Document Phase 0 as future enhancement
5. **Validate against omnara repo** - Run new scenarios on real codebase
6. **Iterate based on feedback** - Tune keyword dictionaries, risk multipliers

---

## 9. Quick Reference

### Modification Types by Risk

**Always Escalate (CRITICAL):**
- Type 9: Security (auth, crypto, validation)
- Type 4: Interface (schema, API breaking changes)

**Usually Escalate (HIGH):**
- Type 2: Behavioral (logic, algorithms)
- Type 1: Structural (refactoring, multi-file)
- Type 10: Performance (concurrency, caching)

**Sometimes Escalate (MODERATE):**
- Type 7: Temporal Hotspot (high churn, incidents)
- Type 8: Ownership (new contributors)
- Type 3: Configuration (environment-dependent)

**Rarely Escalate (LOW):**
- Type 5: Testing (test additions)
- Type 6: Documentation (zero runtime impact)

### Test Scenarios by Priority

**Must Have (P0):**
- Scenario 7: Security changes
- Scenario 6A: Production config changes

**Should Have (P1):**
- Scenario 5: Structural refactoring
- Scenario 9: Multi-type changes
- Scenario 12: Temporal hotspots

**Nice to Have (P2):**
- Scenario 8: Performance optimization
- Scenario 10: Docs-only (fast-path validation)
- Scenario 11: Ownership risk

---

## Related Documentation

- **[MODIFICATION_TYPES_AND_TESTING.md](MODIFICATION_TYPES_AND_TESTING.md)** - Full taxonomy, formulas, and detailed test scenarios
- **[INTEGRATION_TEST_STRATEGY.md](INTEGRATION_TEST_STRATEGY.md)** - Overall testing strategy and existing scenarios
- **[agentic_design.md](../../01-architecture/agentic_design.md)** - Investigation workflow architecture
- **[risk_assessment_methodology.md](../../01-architecture/risk_assessment_methodology.md)** - Risk calculation details

---

**Last Updated:** October 10, 2025
**Status:** Ready for review and implementation
**Owner:** QA + Engineering team
