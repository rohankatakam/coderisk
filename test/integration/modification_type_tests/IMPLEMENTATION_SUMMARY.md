# Automated Modification Type Tests - Implementation Summary

**Created:** October 10, 2025
**Status:** Ready for execution
**Test Repository:** test_sandbox/omnara

---

## What We Built

An automated testing framework that:

1. **Makes controlled changes** to the omnara codebase
2. **Runs `crisk check`** on modified files
3. **Captures output** to files for analysis
4. **Validates results** against expected behavior
5. **Resets changes** to maintain clean git state
6. **Generates reports** comparing actual vs expected outputs

---

## Files Created

### Documentation
- **[README.md](README.md)** - Comprehensive testing guide (7KB)
- **[TEST_PLAN.md](TEST_PLAN.md)** - Detailed specifications for all 12 scenarios (39KB)
- **[IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md)** - This file

### Test Scripts
- **[setup_tests.sh](setup_tests.sh)** - One-time setup script (2KB)
- **[run_all_tests.sh](run_all_tests.sh)** - Master test runner (6KB)
- **[scenario_5_structural.sh](scenario_5_structural.sh)** - Type 1: Structural refactoring (4KB)
- **[scenario_6a_prod_config.sh](scenario_6a_prod_config.sh)** - Type 3: Production config (4.5KB)
- **[scenario_7_security.sh](scenario_7_security.sh)** - Type 9: Security-sensitive (3.8KB)
- **[scenario_10_docs_only.sh](scenario_10_docs_only.sh)** - Type 6: Documentation-only (4.2KB)

### Output Files (Generated During Tests)
- `output_scenario_5.txt` - Actual crisk output for scenario 5
- `output_scenario_6a.txt` - Actual crisk output for scenario 6A
- `output_scenario_7.txt` - Actual crisk output for scenario 7
- `output_scenario_10.txt` - Actual crisk output for scenario 10
- `test_report.txt` - Consolidated test execution report

---

## How It Works

### Individual Test Flow

```
┌─────────────────────────────────────────────────────────┐
│ 1. SETUP PHASE                                          │
├─────────────────────────────────────────────────────────┤
│ • Verify prerequisites (crisk binary, git repo)         │
│ • Check git working tree is clean                       │
│ • Navigate to test_sandbox/omnara                       │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│ 2. MODIFICATION PHASE                                   │
├─────────────────────────────────────────────────────────┤
│ • Make controlled changes to target files               │
│   - Security: Add TODO in auth/routes.py                │
│   - Config: Modify .env.production                      │
│   - Docs: Add section to README.md                      │
│   - Structural: Comment out function + update import    │
│ • Verify changes with git diff                          │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│ 3. EXECUTION PHASE                                      │
├─────────────────────────────────────────────────────────┤
│ • Run: crisk check <modified-files>                     │
│ • Capture stdout/stderr to output_scenario_X.txt        │
│ • Measure execution time                                │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│ 4. VALIDATION PHASE                                     │
├─────────────────────────────────────────────────────────┤
│ • Display output                                        │
│ • Check for key indicators:                             │
│   ✅ Risk Level present                                 │
│   ✅ Expected risk level (HIGH/LOW/etc)                 │
│   ✅ Modification type mentioned (if Phase 0 exists)    │
│   ✅ Phase 2 escalation (for high-risk scenarios)       │
│ • Print validation summary                              │
└─────────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│ 5. CLEANUP PHASE                                        │
├─────────────────────────────────────────────────────────┤
│ • Reset changes: git restore <files>                    │
│ • Remove backups (.bak files)                           │
│ • Verify git working tree clean                         │
└─────────────────────────────────────────────────────────┘
```

### Master Runner Flow

```
run_all_tests.sh
    ↓
┌─────────────────────────────┐
│ Initialize test report      │
└─────────────────────────────┘
    ↓
┌─────────────────────────────┐
│ For each scenario:          │
│  • scenario_5_structural    │
│  • scenario_6a_prod_config  │
│  • scenario_7_security      │
│  • scenario_10_docs_only    │
└─────────────────────────────┘
    ↓
┌─────────────────────────────┐
│ Run test script             │
│ Record PASS/FAIL            │
│ Capture timing              │
└─────────────────────────────┘
    ↓
┌─────────────────────────────┐
│ Analyze all outputs         │
│  • Extract risk levels      │
│  • Check for Phase 2        │
│  • Count recommendations    │
└─────────────────────────────┘
    ↓
┌─────────────────────────────┐
│ Compare vs expected (if     │
│ expected_scenario_X.txt     │
│ files exist)                │
└─────────────────────────────┘
    ↓
┌─────────────────────────────┐
│ Generate final report       │
│ Exit with status code       │
└─────────────────────────────┘
```

---

## Quick Start

### 1. One-Time Setup

```bash
# From coderisk-go root
./test/integration/modification_type_tests/setup_tests.sh
```

This will:
- Build crisk binary if needed
- Start Docker services
- Clone omnara repository
- Make scripts executable
- Optionally initialize graph

### 2. Run All Tests

```bash
./test/integration/modification_type_tests/run_all_tests.sh
```

Expected output:
```
========================================
Modification Type Tests - Master Runner
Started: 2025-10-10 12:00:00
========================================

========================================
Running Scenario 7: Security-Sensitive Change (Type 9)
========================================
[... test output ...]
✅ Scenario 7: PASSED

[... more scenarios ...]

========================================
TEST EXECUTION SUMMARY
========================================
Total scenarios: 4
Passed: 4
Failed: 0
Success rate: 100%
```

### 3. Run Individual Test

```bash
./test/integration/modification_type_tests/scenario_7_security.sh
```

---

## Test Scenarios Implemented

### ✅ Scenario 5: Structural Refactoring (Type 1)

**Files Modified:**
- `src/backend/auth/utils.py`
- `src/backend/auth/routes.py`

**Changes:**
- Comments out `update_user_profile` function (simulates move)
- Updates import statement in routes.py

**Expected Outcome:**
- Risk Level: **HIGH**
- Type: Structural (Type 1B - Dependency Changes)
- Multi-file refactoring detected
- Recommendations for testing and verification

---

### ✅ Scenario 6A: Production Configuration (Type 3)

**Files Modified:**
- `.env.production` (created)

**Changes:**
- Changes `PRODUCTION_DB_URL` password

**Expected Outcome:**
- Risk Level: **CRITICAL**
- Type: Configuration (Type 3A - Environment)
- Production environment detected
- Sensitive credential warning
- Rollback plan recommendation

---

### ✅ Scenario 7: Security-Sensitive (Type 9)

**Files Modified:**
- `src/backend/auth/routes.py`

**Changes:**
- Adds TODO comment in `sync_user` function about session timeout

**Expected Outcome:**
- Risk Level: **CRITICAL**
- Type: Security (Type 9A - Authentication)
- Security keywords detected
- Forced Phase 2 escalation
- Security review recommendations

**Key Validation:**
- Should detect security context (auth file, user session)
- Should recommend security team review
- Should suggest bypass vulnerability testing

---

### ✅ Scenario 10: Documentation-Only (Type 6)

**Files Modified:**
- `README.md`

**Changes:**
- Adds "Development Setup" section with installation instructions

**Expected Outcome:**
- Risk Level: **LOW**
- Type: Documentation (Type 6B - External Docs)
- Fast execution (<50ms if Phase 0 implemented)
- No Phase 2 escalation
- Safe to commit immediately

**Key Validation:**
- Fastest execution time (should skip risk analysis)
- Zero runtime impact acknowledged
- No graph queries executed

---

## Expected vs Actual Workflow

### Current State (Without Phase 0)

All scenarios run through **Phase 1** baseline assessment:

```
scenario_7_security.sh
    ↓
crisk check auth/routes.py
    ↓
Phase 1: Calculate Tier 1 metrics
  • Coupling: Check dependencies
  • Co-change: Check temporal patterns
  • Test coverage: Check test ratio
    ↓
Heuristic: Coupling > 10? → Escalate
    ↓
Phase 2: LLM Investigation (if HIGH)
    ↓
Output: Risk assessment
```

**Expected Behavior:**
- Scenario 7 (security): Should trigger Phase 2 due to auth file coupling
- Scenario 10 (docs): Might still run Phase 1 (200ms), but report LOW risk
- Scenario 6A (prod config): May or may not detect "production" context

### Future State (With Phase 0)

Phase 0 pre-filters based on modification type:

```
scenario_7_security.sh
    ↓
crisk check auth/routes.py
    ↓
Phase 0: Quick heuristics
  • Security keywords? YES → Force escalate
  • Docs-only? NO
  • Config file? NO
    ↓
SKIP Phase 1 (not needed)
    ↓
Phase 2: LLM Investigation (forced)
    ↓
Output: CRITICAL risk

scenario_10_docs_only.sh
    ↓
crisk check README.md
    ↓
Phase 0: Quick heuristics
  • Security keywords? NO
  • Docs-only? YES → Skip analysis
  • Config file? NO
    ↓
SKIP Phase 1 and Phase 2
    ↓
Output: LOW risk (<10ms)
```

**Improved Behavior:**
- Security changes **always** escalate (no false negatives)
- Documentation changes **never** trigger analysis (instant LOW)
- Configuration changes get environment-aware assessment

---

## Validation Strategy

### Automated Checks (Built Into Scripts)

Each test script validates:

```bash
✅ Risk level present in output
✅ Risk level matches expected (HIGH/CRITICAL/LOW)
✅ Key indicators found (security, configuration, documentation)
✅ Phase 2 triggered/skipped as expected
✅ Execution time reasonable
```

### Manual Review Process

After running tests:

1. **Review actual outputs**
   ```bash
   cat test/integration/modification_type_tests/output_scenario_7.txt
   ```

2. **Compare to TEST_PLAN.md**
   - Does risk level match expected?
   - Are recommendations present and actionable?
   - Is modification type mentioned (if Phase 0 exists)?

3. **Create expected outputs**
   ```bash
   # If actual output is correct:
   cp output_scenario_7.txt expected_scenario_7.txt

   # Or manually create ideal output:
   vim expected_scenario_7.txt
   ```

4. **Re-run tests to compare**
   ```bash
   ./run_all_tests.sh
   # Generates diff_scenario_7.txt if differences found
   ```

---

## Next Steps

### Immediate (Complete 4 Implemented Tests)

1. **Run tests** to generate actual outputs
   ```bash
   ./test/integration/modification_type_tests/setup_tests.sh
   ./test/integration/modification_type_tests/run_all_tests.sh
   ```

2. **Review outputs** against TEST_PLAN.md expectations
   ```bash
   for i in 5 6a 7 10; do
       echo "=== Scenario $i ==="
       cat test/integration/modification_type_tests/output_scenario_${i}.txt
       echo ""
   done
   ```

3. **Create expected outputs** for validated scenarios
   ```bash
   # After validation, save as expected
   cp output_scenario_7.txt expected_scenario_7.txt
   # ... repeat for others
   ```

4. **Document findings** - Note any discrepancies between expected and actual

### Short-Term (Complete Remaining 8 Tests)

Implement scenarios:
- 6B: Development config change
- 8: Performance optimization
- 9: Multi-type change
- 11: Ownership risk
- 12: Temporal hotspot

Use existing scripts as templates.

### Medium-Term (Phase 0 Implementation)

1. **Implement Phase 0 in crisk** (see [MODIFICATION_TYPES_AND_TESTING.md](../../../dev_docs/03-implementation/testing/MODIFICATION_TYPES_AND_TESTING.md) §5)

2. **Add Phase 0 detection** to test scripts
   ```bash
   if grep -q "Phase 0:" "$OUTPUT_FILE"; then
       echo "✅ Phase 0 pre-analysis detected"
   fi
   ```

3. **Validate performance improvements**
   - Docs-only: <10ms (vs 200ms)
   - Security: Same time, but better recommendations

### Long-Term (CI/CD Integration)

1. **Add to GitHub Actions** workflow
2. **Automate on every PR**
3. **Track test flakiness** and performance regressions
4. **Expand test coverage** to more omnara files

---

## Troubleshooting Guide

### Setup Issues

**Problem:** `crisk` binary not found
```bash
# Solution:
make build
```

**Problem:** Docker services not running
```bash
# Solution:
docker compose up -d
```

**Problem:** omnara repo not found
```bash
# Solution:
mkdir -p test_sandbox
cd test_sandbox
git clone https://github.com/omnara-ai/omnara.git
```

### Test Execution Issues

**Problem:** "Git working directory not clean"
```bash
# Solution:
cd test_sandbox/omnara
git restore .
git clean -fd
```

**Problem:** Test script permission denied
```bash
# Solution:
chmod +x test/integration/modification_type_tests/*.sh
```

**Problem:** Output file shows "No files to check"
```bash
# Cause: File path incorrect or changes not applied
# Solution: Check test script's file paths and modification logic
```

### Validation Issues

**Problem:** Risk level doesn't match expected
```bash
# This is not necessarily a failure!
# Document actual behavior and adjust expectations if system is correct
# Or file bug if system behavior is wrong
```

**Problem:** Modification type not detected
```bash
# Expected if Phase 0 not yet implemented
# Tests are designed to work with or without Phase 0
```

---

## Success Metrics

### Test Execution Metrics

- ✅ All tests complete without errors
- ✅ Git state restored after each test
- ✅ Outputs generated for all scenarios
- ✅ Validation checks pass (risk level present, etc.)

### System Validation Metrics

**Scenario 7 (Security):**
- Should detect HIGH or CRITICAL risk
- Should mention security context
- Should provide security-specific recommendations

**Scenario 10 (Docs):**
- Should detect LOW risk
- Should complete quickly (<200ms, ideally <50ms)
- Should not trigger Phase 2

**Scenario 6A (Production Config):**
- Should detect HIGH or CRITICAL risk
- Should acknowledge production environment (ideally)
- Should warn about sensitive values (ideally)

**Scenario 5 (Structural):**
- Should detect HIGH or MEDIUM risk
- Should note multi-file impact
- Should recommend integration testing

---

## Contributing

### Adding New Test Scenario

1. **Copy template** from existing scenario
2. **Modify for your scenario:**
   - Change TARGET_FILE(S)
   - Update modification logic
   - Adjust validation checks
3. **Document in TEST_PLAN.md**
4. **Update run_all_tests.sh** SCENARIOS array
5. **Test thoroughly** before committing

### Improving Existing Tests

- Add more validation checks
- Improve error handling
- Enhance output formatting
- Add performance benchmarks

---

## References

- **[MODIFICATION_TYPES_AND_TESTING.md](../../../dev_docs/03-implementation/testing/MODIFICATION_TYPES_AND_TESTING.md)** - Full taxonomy and Phase 0 design
- **[TESTING_EXPANSION_SUMMARY.md](../../../dev_docs/03-implementation/testing/TESTING_EXPANSION_SUMMARY.md)** - Executive summary
- **[INTEGRATION_TEST_STRATEGY.md](../../../dev_docs/03-implementation/testing/INTEGRATION_TEST_STRATEGY.md)** - Overall testing strategy
- **[TEST_PLAN.md](TEST_PLAN.md)** - Detailed specifications for all 12 scenarios
- **[README.md](README.md)** - Testing guide and troubleshooting

---

**Status:** ✅ Ready for execution
**Last Updated:** October 10, 2025
**Maintainer:** CodeRisk QA Team
