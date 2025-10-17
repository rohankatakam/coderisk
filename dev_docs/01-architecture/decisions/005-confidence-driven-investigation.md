# ADR 005: Confidence-Driven Investigation with Adaptive Thresholds

**Status:** Proposed
**Date:** October 10, 2025
**Decision Makers:** Architecture Team
**Related Documents:**
- [agentic_design.md](../agentic_design.md)
- [arc_intelligence_architecture.md](../arc_intelligence_architecture.md)
- [risk_assessment_methodology.md](../risk_assessment_methodology.md)

---

## Context

Our current Phase 2 investigation uses a **fixed 3-hop limit** for LLM-guided evidence gathering. Analysis of test results ([TEST_RESULTS_OCT_2025.md](../../03-implementation/testing/TEST_RESULTS_OCT_2025.md)) and research into optimal search strategies (inspired by Google DeepMind's tree search paper on scientific software generation) revealed critical gaps:

### Current System Limitations

1. **Fixed hop count inefficient:**
   - LOW risk changes waste 2-3 hops gathering unnecessary evidence
   - HIGH risk changes may need >3 hops for confident assessment
   - No stopping criterion based on evidence quality

2. **Fixed thresholds cause false positives:**
   - `coupling > 10` treats Python web apps (naturally high coupling) same as Go microservices
   - `test_ratio < 0.3` flags documentation files (incorrect)
   - No domain/language awareness

3. **Missing Phase 0 pre-analysis:**
   - Security-sensitive files (auth.py) get LOW risk → false negatives
   - Documentation files (README.md) escalate to Phase 2 → 1,351x slower than needed
   - Production configs (.env.production) downgraded to MINIMAL → critical false negative

4. **No pattern recombination:**
   - ARC patterns matched individually
   - Missing hybrid insights (e.g., "auth coupling + no tests" = critical pattern)

### Research Insights

Google DeepMind's tree search paper demonstrates that **confidence-based stopping** + **adaptive configuration** achieves superhuman performance:

```python
# Their approach (for scientific software):
while confidence < threshold and budget_remaining:
    evidence = gather_next_metric()
    confidence = assess_current_confidence()
    if confidence >= threshold:
        break  # Stop early, high confidence

# vs Fixed approach (our current):
for hop in range(3):  # Always 3 hops, regardless of confidence
    gather_metric()
```

**Key finding:** Systems that adapt based on **confidence scores** and **domain context** achieve:
- 44% of hybrid solutions outperform both parents (pattern recombination)
- Adaptive thresholds reduce false positives 20-30% (domain-aware configs)
- Early stopping when confident reduces latency 30-40%

---

## Decision

We will implement a **four-part enhancement** to the investigation system:

### 1. Phase 0: Adaptive Pre-Analysis

**Add lightweight pre-analysis layer** (budget: <50ms):

```python
def phase_0_pre_analysis(file_path, diff, repo_metadata):
    """
    Fast heuristics before Phase 1 baseline
    """
    # Security keyword detection (Type 9)
    if contains_security_keywords(file_path, diff):
        return PreAnalysisResult(
            force_escalate=True,
            modification_type="SECURITY",
            reason="Auth/crypto/validation keywords detected"
        )

    # Documentation-only skip (Type 6)
    if is_documentation_only(file_path, diff):
        return PreAnalysisResult(
            skip_analysis=True,
            risk_level="LOW",
            reason="Zero runtime impact (docs only)"
        )

    # Environment detection (Type 3)
    if is_production_config(file_path):
        return PreAnalysisResult(
            force_escalate=True,
            modification_type="PRODUCTION_CONFIG",
            reason="Production environment configuration"
        )

    # Select domain-aware config
    config = select_adaptive_config(repo_metadata)

    return PreAnalysisResult(
        proceed_to_phase1=True,
        config=config
    )
```

**Modification types from MODIFICATION_TYPES_AND_TESTING.md:**
- Type 1: Structural (refactoring, imports)
- Type 2: Behavioral (logic changes)
- Type 3: Configuration (env files, feature flags)
- Type 4: Interface (API, schema changes)
- Type 5: Testing (test additions)
- Type 6: Documentation (comments, README)
- Type 7: Temporal hotspot (high churn files)
- Type 8: Ownership (new contributor)
- Type 9: Security (auth, crypto, validation)
- Type 10: Performance (caching, concurrency)

### 2. Adaptive Configuration Selection

**Domain-aware threshold configs** (inspired by paper's 8 preset configurations):

```python
RISK_CONFIGS = {
    "python_web": {
        "coupling_threshold": 15,      # Web apps have higher coupling
        "co_change_threshold": 0.75,
        "test_ratio_threshold": 0.4,
    },
    "go_backend": {
        "coupling_threshold": 8,       # Go services more modular
        "co_change_threshold": 0.6,
        "test_ratio_threshold": 0.5,
    },
    "typescript_frontend": {
        "coupling_threshold": 20,      # React components highly coupled
        "co_change_threshold": 0.8,
        "test_ratio_threshold": 0.3,
    },
    "default": {
        "coupling_threshold": 10,
        "co_change_threshold": 0.7,
        "test_ratio_threshold": 0.3,
    }
}

def select_adaptive_config(repo_metadata):
    language = repo_metadata.primary_language
    domain = infer_domain(repo_metadata)  # web, backend, ML, etc.
    config_key = f"{language}_{domain}"
    return RISK_CONFIGS.get(config_key, RISK_CONFIGS["default"])
```

### 3. Confidence-Driven Investigation

**Replace fixed hop count with confidence threshold:**

```python
def investigate_with_confidence(context):
    """
    Stop when confidence > threshold, not fixed hop count
    """
    confidence = 0.0
    iteration = 0
    max_iterations = 5  # Upper bound for cost control
    confidence_threshold = 0.85

    while confidence < confidence_threshold and iteration < max_iterations:
        # LLM decides next metric/expansion
        decision = llm_decide_action(context)

        # Execute and track confidence
        evidence = execute_decision(decision)
        context.add_evidence(evidence)

        # Track breakthroughs (when risk level changes significantly)
        if abs(current_risk - previous_risk) > 0.2:
            context.record_breakthrough(
                hop=iteration,
                evidence=evidence,
                risk_change=(previous_risk, current_risk)
            )

        # LLM assesses: "How confident are you NOW?"
        confidence = llm_assess_confidence(context)

        iteration += 1

    return llm_synthesize_risk(context, confidence)
```

**LLM confidence assessment prompt:**
```
Based on evidence gathered so far:
{evidence_chain}

How confident are you in the risk assessment? (0.0-1.0)

Consider:
- Do you have enough evidence to make a decisive call?
- Are there contradicting signals that need resolution?
- Would gathering more evidence significantly change your assessment?

Respond with JSON:
{
  "confidence": 0.85,
  "reasoning": "High coupling (12 deps) + low test coverage (0.25) + security file = clear HIGH risk. Additional evidence unlikely to change assessment.",
  "next_action": "FINALIZE"  // or "GATHER_MORE_EVIDENCE"
}
```

### 4. ARC Pattern Recombination

**Hybrid pattern matching** (inspired by paper's 44% improvement from recombination):

```python
def find_hybrid_arc_matches(file_path, context):
    """
    Combine multiple ARC patterns for richer insights
    """
    # Stage 1: Exact matches (current approach)
    exact_arcs = query_exact_arc_matches(file_path)

    # Stage 2: Combine complementary ARCs (NEW)
    if len(exact_arcs) >= 2:
        hybrid_patterns = []

        for arc_a, arc_b in combinations(exact_arcs, 2):
            # Check if ARCs are complementary (different dimensions)
            if are_complementary(arc_a, arc_b):
                hybrid = {
                    "primary_arcs": [arc_a.arc_id, arc_b.arc_id],
                    "hybrid_insight": synthesize_hybrid_insight(arc_a, arc_b, context),
                    "severity": max(arc_a.severity, arc_b.severity),
                    "confidence": min(arc_a.confidence, arc_b.confidence) * 0.9
                }
                hybrid_patterns.append(hybrid)

        return {
            "exact_matches": exact_arcs,
            "hybrid_insights": hybrid_patterns
        }

    return {"exact_matches": exact_arcs, "hybrid_insights": []}

def are_complementary(arc_a, arc_b):
    """
    Check if two ARCs address different risk dimensions
    """
    dimensions = {
        "coupling": ["ARC-2025-001", "ARC-2025-034"],
        "testing": ["ARC-2025-045", "ARC-2025-067"],
        "temporal": ["ARC-2025-012", "ARC-2025-023"],
    }

    arc_a_dims = [dim for dim, arcs in dimensions.items() if arc_a.arc_id in arcs]
    arc_b_dims = [dim for dim, arcs in dimensions.items() if arc_b.arc_id in arcs]

    # Complementary if addressing different dimensions
    return len(set(arc_a_dims) & set(arc_b_dims)) == 0
```

---

## Consequences

### Positive

**1. Dramatic False Positive Reduction (70-80%)**

From test results analysis:
- Phase 0 security detection: Prevents auth.py getting LOW risk (currently false negative)
- Phase 0 docs skip: README.md no longer escalates to Phase 2 (1,351x speedup)
- Phase 0 env detection: .env.production gets CRITICAL not MINIMAL
- Adaptive thresholds: 20-30% FP reduction by domain awareness

**Combined impact:** 50% current FP rate → **10-15%** target FP rate

**2. Improved Insight Quality (3-5x)**

- Hybrid ARC matching: "Auth coupling + no tests" instead of generic "high coupling"
- Domain-specific recommendations: "Add React component tests" vs "add tests"
- Breakthrough tracking: Explain WHICH evidence caused risk change

**3. Optimized Latency**

- 80% of checks: <200ms (Phase 0 skip or early Phase 1 exit)
- 20% needing Phase 2: Stop early when confident (30-40% faster)
- Current: All checks take 2-5s (no early stopping)

**Expected distribution:**
```
Phase 0 skip (docs/configs): 20% of checks, <10ms
Phase 1 only (LOW risk):    60% of checks, 50-200ms
Phase 2 (HIGH risk):        20% of checks, 2-4s (vs current 3-5s)

Weighted average: 0.2×10ms + 0.6×100ms + 0.2×3s = 662ms
vs Current: 2,500ms (all checks)
Improvement: 3.8x faster on average
```

**4. Strategic Moat Enhancement**

- ARC pattern recombination → Better incident prediction (network effects)
- Breakthrough tracking → Continuous learning from investigation traces
- Domain adaptation → Learns optimal thresholds per repo type

### Negative

**1. Increased Complexity**

- Phase 0 adds new code path (but well-bounded: <50ms budget)
- Multiple configs to maintain (mitigated by validation on historical data)
- Confidence scoring requires LLM call (adds $0.001/check)

**2. Risk of Over-Optimization**

- Adaptive thresholds might fit training data too closely
- Need to validate configs on holdout repositories
- Breakthrough tracking could bias toward recent patterns

**Mitigation:**
- A/B test adaptive vs fixed thresholds (track FP rates)
- Regular config validation on new repos
- Monitor breakthrough patterns for overfitting

### Neutral

**1. Implementation Timeline**

- Phase 0: 1-2 weeks (security/docs/env detection)
- Adaptive configs: 1 week (config selection + validation)
- Confidence-driven: 2 weeks (LLM integration + testing)
- ARC recombination: 3-4 weeks (depends on ARC database implementation)

**Total:** 7-9 weeks for full implementation

---

## Implementation Plan

### Week 1-2: Phase 0 Foundation

**Priority 0 (Biggest FP reduction):**

1. Security keyword detection
   ```go
   // internal/analysis/phase0/security.go
   func DetectSecurityKeywords(filePath, content string) bool {
       keywords := []string{"auth", "login", "password", "session", "token", "jwt", "crypto", "encrypt"}
       // ...
   }
   ```

2. Documentation skip logic
   ```go
   // internal/analysis/phase0/documentation.go
   func IsDocumentationOnly(filePath string, diff GitDiff) bool {
       docExtensions := []string{".md", ".txt", ".rst"}
       // ...
   }
   ```

3. Environment detection
   ```go
   // internal/analysis/phase0/environment.go
   func IsProductionConfig(filePath string) (bool, string) {
       if strings.Contains(filePath, ".env.production") {
           return true, "production"
       }
       // ...
   }
   ```

### Week 3: Adaptive Configurations

4. Domain inference from repo metadata
   ```go
   // internal/analysis/config/adaptive.go
   func SelectConfig(metadata RepoMetadata) RiskConfig {
       // Language + framework detection
       // Return appropriate threshold config
   }
   ```

5. Config validation on historical data
   ```bash
   # Validate on omnara test repo
   ./crisk init-local test_sandbox/omnara
   # Run scenarios with different configs
   ```

### Week 4-5: Confidence-Driven Investigation

6. Confidence assessment prompt
   ```go
   // internal/agent/prompts/confidence.go
   const ConfidencePrompt = `...`
   ```

7. Breakthrough detection
   ```go
   // internal/agent/investigation.go
   func trackBreakthrough(context, evidence) {
       // Record when risk level changes significantly
   }
   ```

8. Early stopping logic
   ```go
   func InvestigateWithConfidence(ctx Context) RiskAssessment {
       for confidence < 0.85 && iteration < 5 {
           // ...
       }
   }
   ```

### Week 6-9: ARC Recombination (depends on ARC implementation)

9. Complementary pattern detection
   ```go
   // internal/arc/recombination.go
   func FindComplementaryARCs(arcs []ARCMatch) []HybridPattern {
       // ...
   }
   ```

10. Hybrid insight synthesis
    ```go
    func SynthesizeHybridInsight(arcA, arcB ARCMatch, context Context) string {
        // LLM prompt: "Combine these two patterns..."
    }
    ```

---

## Validation Metrics

### Success Criteria

**False Positive Rate:**
- Baseline: 50% (from test results)
- Target: 10-15% (70-80% reduction)
- Measure: User feedback on `crisk check` results

**Insight Quality:**
- Baseline: Generic recommendations ("add tests")
- Target: Specific actionable insights ("Add integration tests for auth + user service interaction")
- Measure: User feedback on recommendation usefulness

**Performance:**
- Baseline: 2,500ms average (all checks)
- Target: 662ms average (weighted by distribution)
- Measure: Latency percentiles (p50, p95, p99)

**Confidence Calibration:**
- Target: When confidence > 0.85, FP rate < 5%
- Measure: Track (confidence, actual_outcome) pairs

### A/B Testing Framework

```python
def assign_investigation_mode(user_id):
    """
    Randomly assign users to control/treatment
    """
    if hash(user_id) % 2 == 0:
        return "control"    # Fixed thresholds, 3-hop investigation
    else:
        return "treatment"  # Adaptive + confidence-driven

def track_outcome(user_id, mode, file, risk_score, user_feedback):
    """
    Track outcomes for A/B test analysis
    """
    analytics.track({
        "user_id": user_id,
        "mode": mode,
        "risk_score": risk_score,
        "false_positive": user_feedback == "incorrect_high_risk",
        "timestamp": now()
    })
```

---

## References

**Internal Documents:**
- [TEST_RESULTS_OCT_2025.md](../../03-implementation/testing/TEST_RESULTS_OCT_2025.md) - Empirical evidence of current gaps
- [MODIFICATION_TYPES_AND_TESTING.md](../../03-implementation/testing/MODIFICATION_TYPES_AND_TESTING.md) - Modification type taxonomy (Phase 0 design)
- [agentic_design.md](../agentic_design.md) - Current investigation architecture
- [arc_intelligence_architecture.md](../arc_intelligence_architecture.md) - ARC database integration strategy

**External Research:**
- "An AI system to help scientists write expert-level empirical software" (Google DeepMind, arXiv:2509.06503v1)
  - Key insights: PUCT tree search with confidence-based stopping, adaptive configuration systems, pattern recombination achieving 44% improvement

**12-Factor Principles Applied:**
- **Factor 3: Own your context window** - Phase 0 reduces unnecessary LLM calls, selective evidence gathering
- **Factor 8: Own your control flow** - Confidence-driven investigation with explicit decision criteria
- **Factor 10: Small, focused agents** - Phase 0 handles pre-analysis, Phase 1 baseline, Phase 2 deep investigation

---

**Status Note:** This ADR is **proposed** pending:
1. Validation of Phase 0 implementation on test scenarios
2. A/B test results comparing adaptive vs fixed thresholds
3. Performance benchmarks on real repositories
