# Known Risk Test Fixtures

This directory contains test files with **known risk levels** for validating the Phase 1 risk calculation engine.

## Files

### 1. `low_risk.go` + `low_risk_test.go`

**Expected Risk:** LOW

**Metrics:**
- **Coupling:** 0 dependencies → LOW (≤5 threshold)
- **Co-change:** No pattern detected → LOW (≤0.3 threshold)
- **Test ratio:** ~0.75 (3 test functions, comprehensive coverage) → LOW (≥0.8 threshold)

**Rationale:** Simple utility function with excellent test coverage and no dependencies.

---

### 2. `medium_risk.go` + `medium_risk_test.go`

**Expected Risk:** MEDIUM

**Metrics:**
- **Coupling:** 2 imports (context, models) → LOW (≤5 threshold)
- **Co-change:** No pattern detected → LOW (≤0.3 threshold)
- **Test ratio:** ~0.4 (1 test function, minimal coverage) → MEDIUM (0.3-0.8 range)

**Rationale:** Moderate coupling with minimal test coverage. Not enough to trigger HIGH risk, but not ideal.

---

### 3. `high_risk.go` (NO TEST FILE)

**Expected Risk:** HIGH

**Metrics:**
- **Coupling:** 15+ imports → **HIGH** (>10 threshold) ✅ **Escalates to Phase 2**
- **Co-change:** No pattern detected → LOW (but coupling already triggers HIGH)
- **Test ratio:** 0 (no test file exists) → **HIGH** (<0.3 threshold) ✅ **Escalates to Phase 2**

**Rationale:** Violates multiple thresholds:
1. High structural coupling (15+ imports, including network, database, file system)
2. Zero test coverage (no test file exists)
3. Complex logic mixing multiple concerns
4. External dependencies (HTTP, SQL, file I/O)

**Phase 1 Decision:** Should escalate to Phase 2 due to `coupling > 10` OR `test_ratio < 0.3`

---

## Usage

These fixtures are used by:

1. **Unit tests** (`internal/metrics/*_test.go`) - Validate individual metric calculations
2. **Integration tests** (`test/integration/test_check_e2e.sh`) - End-to-end CLI testing
3. **Manual validation** - Developers can run `crisk check` on these files to verify output

## Expected Outputs

### Quiet Mode (`--quiet`)

```bash
$ crisk check --quiet test/fixtures/known_risk/low_risk.go
✅ LOW risk

$ crisk check --quiet test/fixtures/known_risk/medium_risk.go
⚠️  MEDIUM risk

$ crisk check --quiet test/fixtures/known_risk/high_risk.go
❌ HIGH risk
```

### Standard Mode (default)

```bash
$ crisk check test/fixtures/known_risk/high_risk.go

Risk Assessment: HIGH

Issues:
  • COUPLING_HIGH: File is connected to 15 other files (HIGH coupling)
  • TEST_COVERAGE_LOW: No test files found (0% coverage - HIGH)

Recommendations:
  • Review and reduce structural coupling
  • Add test coverage for this file

⚠️  HIGH RISK - Would escalate to Phase 2 (LLM investigation)
```

### Explain Mode (`--explain`)

```bash
$ crisk check --explain test/fixtures/known_risk/high_risk.go

=== Phase 1 Risk Assessment ===

File: test/fixtures/known_risk/high_risk.go
Overall Risk: HIGH
Phase 2 Escalation: true

Evidence (Tier 1 Metrics):
  • Coupling: File is connected to 15 other files (HIGH coupling)
  • Co-Change: No co-change pattern detected
  • Test Coverage: No test files found (0% coverage - HIGH)

Duration: 187ms

Decision Tree:
  1. Coupling count (15) > threshold (10) → HIGH risk
  2. Test ratio (0.0) < threshold (0.3) → HIGH risk
  3. Any HIGH signal → Escalate to Phase 2 ✅

Next Steps (Phase 2):
  - Expand graph to 2-hop neighbors
  - Calculate ownership churn (Tier 2 metric)
  - Check incident similarity (BM25 search)
  - LLM synthesis of evidence chain
```

### AI Mode (`--ai-mode`)

```bash
$ crisk check --ai-mode test/fixtures/known_risk/high_risk.go | jq .

{
  "risk_level": "HIGH",
  "risk_score": 8.0,
  "confidence": 0.8,
  "should_block_commit": true,
  "block_reason": "high_risk_detected",
  "override_allowed": true,
  "override_requires_justification": true,
  "files": [
    {
      "path": "test/fixtures/known_risk/high_risk.go",
      "risk_score": 8.0,
      "metrics": {
        "coupling": {
          "name": "Structural Coupling",
          "value": 15,
          "threshold": 10
        },
        "test_coverage": {
          "name": "Test Coverage",
          "value": 0.0,
          "threshold": 0.3
        }
      }
    }
  ],
  "issues": [
    {
      "id": "COUPLING_HIGH",
      "severity": "HIGH",
      "category": "coupling",
      "file": "test/fixtures/known_risk/high_risk.go",
      "message": "File is connected to 15 other files (HIGH coupling)"
    },
    {
      "id": "TEST_COVERAGE_LOW",
      "severity": "HIGH",
      "category": "quality",
      "file": "test/fixtures/known_risk/high_risk.go",
      "message": "No test files found (0% coverage - HIGH)"
    }
  ]
}
```

---

## Validation Criteria

✅ **Accurate Risk Levels**
- Low risk file → LOW (no false positives)
- Medium risk file → MEDIUM (correct boundary case)
- High risk file → HIGH (no false negatives)

✅ **Correct Escalation Logic**
- Only HIGH risk files trigger `should_escalate: true`
- Phase 2 escalation message displayed only for HIGH risk

✅ **Metric Accuracy**
- Coupling counts match actual imports
- Test ratio matches actual file presence
- Co-change calculated from git history (if available)

✅ **Performance**
- Phase 1 completes in <500ms per file
- Cache hits reduce latency to <50ms

---

## References

- **Specification:** `dev_docs/01-architecture/risk_assessment_methodology.md` (§2 - Tier 1 Metrics)
- **Implementation:** `internal/metrics/` (coupling.go, co_change.go, test_ratio.go)
- **Session Prompt:** `dev_docs/03-implementation/session_guides/SESSION_6_PROMPT.md`
