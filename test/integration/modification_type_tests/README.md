# Modification Type Tests

**Purpose:** Automated testing suite for validating CodeRisk's detection and assessment of different modification types
**Repository:** test_sandbox/omnara
**Last Updated:** October 10, 2025

---

## Overview

This directory contains automated tests for the 12 modification type scenarios defined in [MODIFICATION_TYPES_AND_TESTING.md](../../../dev_docs/03-implementation/testing/MODIFICATION_TYPES_AND_TESTING.md).

Each test:
1. Makes a controlled change to the omnara codebase
2. Runs `crisk check` on the modified files
3. Captures and validates the output
4. Resets the changes to maintain a clean git state

---

## Quick Start

### Run All Tests

```bash
# From coderisk-go root directory
./test/integration/modification_type_tests/run_all_tests.sh
```

### Run Individual Test

```bash
# Make script executable
chmod +x test/integration/modification_type_tests/scenario_7_security.sh

# Run test
./test/integration/modification_type_tests/scenario_7_security.sh
```

---

## Test Scenarios

### Implemented Tests

| Scenario | Script | Type | Description | Status |
|----------|--------|------|-------------|--------|
| 5 | `scenario_5_structural.sh` | Type 1 | Multi-file refactoring | ✅ Implemented |
| 6A | `scenario_6a_prod_config.sh` | Type 3 | Production config change | ✅ Implemented |
| 7 | `scenario_7_security.sh` | Type 9 | Security-sensitive change | ✅ Implemented |
| 10 | `scenario_10_docs_only.sh` | Type 6 | Documentation-only | ✅ Implemented |

### Planned Tests (To Be Implemented)

| Scenario | Script | Type | Description | Priority |
|----------|--------|------|-------------|----------|
| 6B | `scenario_6b_dev_config.sh` | Type 3 | Development config change | P1 |
| 8 | `scenario_8_performance.sh` | Type 10 | Performance optimization | P1 |
| 9 | `scenario_9_multi_type.sh` | Combined | Multi-type change | P1 |
| 11 | `scenario_11_ownership.sh` | Type 8 | New contributor change | P2 |
| 12 | `scenario_12_hotspot.sh` | Type 7 | Temporal hotspot | P2 |

---

## File Structure

```
modification_type_tests/
├── README.md                      # This file
├── TEST_PLAN.md                   # Detailed test specifications
├── run_all_tests.sh               # Master test runner
├── scenario_5_structural.sh       # Type 1: Structural refactoring
├── scenario_6a_prod_config.sh     # Type 3: Production config
├── scenario_7_security.sh         # Type 9: Security-sensitive
├── scenario_10_docs_only.sh       # Type 6: Documentation-only
├── output_scenario_*.txt          # Actual test outputs (generated)
├── expected_scenario_*.txt        # Expected outputs (to be created)
├── diff_scenario_*.txt            # Comparison results (generated)
└── test_report.txt                # Consolidated test report (generated)
```

---

## Prerequisites

### 1. CodeRisk Setup

```bash
# Build crisk binary
make build

# Verify crisk works
./crisk --version

# Start Docker services
docker compose up -d
```

### 2. Test Repository Setup

```bash
# Clone omnara to test_sandbox/
mkdir -p test_sandbox
cd test_sandbox
git clone https://github.com/omnara-ai/omnara.git

# Verify clean git state
cd omnara
git status  # Should show "working tree clean"
```

### 3. Initialize CodeRisk Graph (Optional)

For more realistic test results, initialize the omnara graph:

```bash
# From coderisk-go root
./crisk init-local test_sandbox/omnara
```

This enables:
- Coupling analysis (Layer 1)
- Co-change frequency detection (Layer 2)
- Incident linkage (Layer 3 - requires manual incident creation)

---

## Test Execution Flow

### Individual Test Flow

Each test script follows this pattern:

```bash
1. Verify prerequisites (crisk binary, git repo)
2. Check git working tree is clean
3. Make controlled changes to specific files
4. Verify changes with git diff
5. Run: crisk check <files>
6. Capture output to output_scenario_X.txt
7. Validate output (risk level, keywords, etc.)
8. Reset git changes (git restore)
9. Verify clean state restored
```

### Master Runner Flow

The `run_all_tests.sh` script:

```bash
1. Initialize test report
2. For each scenario:
   a. Run test script
   b. Record PASS/FAIL
   c. Capture timing
3. Analyze all outputs
4. Generate comparison report
5. Exit with status code (0 = all passed)
```

---

## Validation Criteria

Each test validates different aspects:

### All Tests Check

- ✅ Risk level present in output
- ✅ Output is well-formed (not empty/error)
- ✅ Git state restored to clean

### Type-Specific Checks

**Security (Scenario 7):**
- ✅ CRITICAL or HIGH risk detected
- ✅ Security keywords mentioned
- ✅ Phase 2 escalation occurred

**Documentation (Scenario 10):**
- ✅ LOW risk detected
- ✅ Fast execution (<50ms if Phase 0 implemented)
- ✅ No Phase 2 escalation

**Production Config (Scenario 6A):**
- ✅ CRITICAL risk detected
- ✅ Environment context (production)
- ✅ Sensitive value warnings

**Structural (Scenario 5):**
- ✅ HIGH or MEDIUM risk
- ✅ Multi-file impact noted
- ✅ Refactoring recommendations

---

## Creating Expected Outputs

After running tests and validating actual outputs:

### 1. Review Actual Output

```bash
cat test/integration/modification_type_tests/output_scenario_7.txt
```

### 2. Verify Against TEST_PLAN.md

Compare actual output to expected output in [TEST_PLAN.md](TEST_PLAN.md).

### 3. Create Expected Output File

If actual output is correct:

```bash
cp output_scenario_7.txt expected_scenario_7.txt
```

If actual output needs tuning, manually create `expected_scenario_7.txt` with the ideal output.

### 4. Re-run Tests to Compare

```bash
./run_all_tests.sh
# Now generates diff_scenario_7.txt comparing actual vs expected
```

---

## Troubleshooting

### Test Fails: "Git working directory not clean"

**Problem:** Previous test didn't restore git state

**Solution:**
```bash
cd test_sandbox/omnara
git status
git restore .  # Reset all changes
git clean -fd  # Remove untracked files
```

### Test Fails: "crisk binary not found"

**Problem:** Running from wrong directory or crisk not built

**Solution:**
```bash
# Verify you're in coderisk-go root
pwd  # Should end with /coderisk-go

# Build crisk
make build
```

### Test Output Shows "No files to check"

**Problem:** File path incorrect or file not tracked by git

**Solution:**
```bash
cd test_sandbox/omnara
git status --short  # Verify file is modified
git diff <file>     # Verify changes exist
```

### Crisk Check Returns Error

**Problem:** Docker services not running or Neo4j not initialized

**Solution:**
```bash
# Start services
docker compose up -d

# Verify services
docker ps

# Initialize graph (if not done)
./crisk init-local test_sandbox/omnara
```

---

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Modification Type Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Start Docker services
        run: docker compose up -d

      - name: Build crisk
        run: make build

      - name: Clone omnara test repo
        run: |
          mkdir -p test_sandbox
          git clone https://github.com/omnara-ai/omnara.git test_sandbox/omnara

      - name: Run modification type tests
        run: ./test/integration/modification_type_tests/run_all_tests.sh

      - name: Upload test outputs
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: test-outputs
          path: test/integration/modification_type_tests/output_scenario_*.txt

      - name: Upload test report
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: test-report
          path: test/integration/modification_type_tests/test_report.txt
```

---

## Extending Tests

### Adding a New Scenario

1. **Create test script:** `scenario_N_description.sh`

```bash
#!/bin/bash
set -e

SCENARIO_NAME="Scenario N: Description"
TEST_DIR="test_sandbox/omnara"
CRISK_BIN="./crisk"
TARGET_FILE="path/to/file"
OUTPUT_FILE="test/integration/modification_type_tests/output_scenario_N.txt"

# Follow standard test flow (see existing scripts)
```

2. **Document in TEST_PLAN.md**

Add expected output specification.

3. **Update run_all_tests.sh**

Add scenario to `SCENARIOS` array:

```bash
SCENARIOS=(
    # ... existing ...
    "N:scenario_N_description.sh:Your Description"
)
```

4. **Test and validate**

```bash
./scenario_N_description.sh
# Review output
# Create expected_scenario_N.txt if correct
```

---

## Performance Benchmarks

Target execution times:

| Scenario | Target Time | Notes |
|----------|-------------|-------|
| Docs-only (10) | <50ms | Should skip Phase 1/2 with Phase 0 |
| Standard (5, 7) | <200ms | Phase 1 baseline |
| High-risk (7) | 3-5s | Phase 2 investigation |

Track actual times with:

```bash
time ./scenario_7_security.sh
```

---

## Related Documentation

- **[MODIFICATION_TYPES_AND_TESTING.md](../../../dev_docs/03-implementation/testing/MODIFICATION_TYPES_AND_TESTING.md)** - Full modification type taxonomy
- **[TESTING_EXPANSION_SUMMARY.md](../../../dev_docs/03-implementation/testing/TESTING_EXPANSION_SUMMARY.md)** - Executive summary
- **[INTEGRATION_TEST_STRATEGY.md](../../../dev_docs/03-implementation/testing/INTEGRATION_TEST_STRATEGY.md)** - Overall testing strategy
- **[TEST_PLAN.md](TEST_PLAN.md)** - Detailed expected outputs for each scenario

---

## Contributing

### Before Committing Tests

1. ✅ Verify test script is idempotent (can run multiple times)
2. ✅ Ensure git state is fully restored after test
3. ✅ Add validation checks for key output elements
4. ✅ Document expected output in TEST_PLAN.md
5. ✅ Update README.md if adding new scenario

### Test Quality Checklist

- [ ] Script has proper error handling (`set -e`)
- [ ] Git state verified before and after
- [ ] Output file named correctly (`output_scenario_X.txt`)
- [ ] Validation checks print clear ✅/❌/ℹ️ indicators
- [ ] Script is executable (`chmod +x`)
- [ ] Added to master runner (`run_all_tests.sh`)

---

**Last Updated:** October 10, 2025
**Maintainer:** CodeRisk QA Team
