# E2E Test Suite for Phase 0 + Adaptive System

This directory contains end-to-end validation tests for the Phase 0 pre-analysis, adaptive configuration, and confidence-driven investigation implementations.

## Test Structure

```
test/e2e/
├── README.md                  # This file
├── test_helpers.sh            # Shared test utility functions
├── phase0_validation.sh       # Phase 0 pre-analysis tests
├── adaptive_config_test.sh    # Adaptive configuration validation (TODO)
├── confidence_loop_test.sh    # Confidence-driven investigation (TODO)
├── run_all_e2e_tests.sh       # Master test runner (TODO)
└── output/                    # Test outputs and reports
```

## Test Scenarios

### Phase 0 Validation (`phase0_validation.sh`)

Tests the Phase 0 pre-analysis layer implementation:

1. **Security-Sensitive Change** - Validates security keyword detection
   - File: `src/backend/auth/routes.py`
   - Expected: CRITICAL/HIGH risk, Phase 2 escalation
   - Target latency: <200ms

2. **Documentation-Only Change** - Validates skip logic
   - File: `README.md`
   - Expected: LOW risk, Phase 1/2 skipped, <50ms latency
   - Target latency: <50ms (with Phase 0 skip)

3. **Production Config Change** - Validates environment detection
   - File: `.env.production`
   - Expected: CRITICAL/HIGH risk, production flag detected
   - Target latency: <200ms

4. **Comment-Only Change** - Validates low-impact detection
   - File: `src/omnara/cli/commands.py`
   - Expected: LOW risk, minimal analysis
   - Target latency: <200ms

### Adaptive Config Validation (TODO)

Tests domain-aware threshold selection:

- Python web app threshold selection
- Go backend threshold comparison
- Domain inference accuracy

### Confidence Loop Validation (TODO)

Tests confidence-driven investigation:

- Early stopping for obvious LOW risk
- Extended investigation for ambiguous cases
- Breakthrough detection
- Average latency improvement

## Running Tests

### Run Individual Test Suites

```bash
# Phase 0 validation
./test/e2e/phase0_validation.sh

# Adaptive config validation (when ready)
./test/e2e/adaptive_config_test.sh

# Confidence loop validation (when ready)
./test/e2e/confidence_loop_test.sh
```

### Run All E2E Tests

```bash
# Run complete E2E test suite
./test/e2e/run_all_e2e_tests.sh
```

## Prerequisites

1. **Built crisk binary**: Run `make build` or ensure `./bin/crisk` exists
2. **Clean git state**: Test sandbox must have clean working directory
3. **Test repository**: `test_sandbox/omnara` must be initialized

## Output

Tests generate outputs in `test/e2e/output/`:

- `phase0_*.txt` - Individual test outputs
- `*_report.txt` - Summary reports with pass/fail metrics
- Performance metrics (latency, FP rate)

## Success Criteria

### Phase 0 Tests
- ✅ Security detection: 100% accuracy
- ✅ Documentation skip: <50ms latency
- ✅ Production config: 100% detection
- ✅ All tests pass

### Adaptive Config Tests (TODO)
- ✅ Domain inference: 90%+ accuracy
- ✅ Threshold selection: Appropriate for domain
- ✅ FP reduction: 20-30% vs fixed thresholds

### Confidence Loop Tests (TODO)
- ✅ Early stopping: 40%+ of LOW risk cases
- ✅ Latency improvement: 30%+ average
- ✅ Confidence calibration: >85% confidence = <5% FP

## Integration with CI/CD

These tests can be integrated into CI pipelines:

```yaml
# Example GitHub Actions
- name: Run E2E Tests
  run: |
    make build
    ./test/e2e/run_all_e2e_tests.sh
```

## Troubleshooting

**Tests fail with "crisk binary not found":**
```bash
make build
# Or: go build -o bin/crisk ./cmd/crisk
```

**Tests fail with "Git working directory not clean":**
```bash
cd test_sandbox/omnara
git status
git restore .
```

**Phase 0 features show "NOT YET IMPLEMENTED":**
- This is expected if Agent 1 hasn't completed Phase 0 implementation
- Tests will show warnings but track baseline behavior
- Re-run tests after Phase 0 is implemented

## Developer Notes

### Adding New Tests

1. Create test function in appropriate script
2. Follow existing pattern:
   - Use helper functions from `test_helpers.sh`
   - Make controlled changes with `sed`
   - Restore git state after test
   - Record results with `record_test_result`

3. Update this README with new test description

### Test Helper Functions

See `test_helpers.sh` for available utilities:
- `print_*` - Colored output (success, error, warning, info)
- `validate_*` - Validation functions (risk level, latency, etc.)
- `record_test_result` - Track pass/fail
- `generate_test_report` - Generate summary

## Related Documentation

- [PHASE_0_PARALLEL_IMPLEMENTATION_PLAN.md](../../dev_docs/03-implementation/PHASE_0_PARALLEL_IMPLEMENTATION_PLAN.md) - Implementation plan
- [TEST_RESULTS_OCT_2025.md](../../dev_docs/03-implementation/testing/TEST_RESULTS_OCT_2025.md) - Baseline test results
- [005-confidence-driven-investigation.md](../../dev_docs/01-architecture/decisions/005-confidence-driven-investigation.md) - Architecture decision

## Status

- ✅ **Phase 0 Validation**: Ready for testing (waiting for Agent 1)
- ⏳ **Adaptive Config**: TODO (waiting for Agent 2)
- ⏳ **Confidence Loop**: TODO (waiting for Agent 3)
- ⏳ **Performance Benchmarks**: TODO (waiting for all agents)
- ⏳ **Master Runner**: TODO (integration phase)
