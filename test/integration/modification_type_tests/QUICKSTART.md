# Quick Start Guide

**Fixed:** Binary path detection now works with both `./bin/crisk` and `./crisk`

---

## Run Tests Now

From the **coderisk-go root** directory:

```bash
# Run all tests
./test/integration/modification_type_tests/run_all_tests.sh
```

---

## What This Does

1. **Scenario 7** (Security): Adds TODO comment to `auth/routes.py`
2. **Scenario 10** (Docs): Adds section to `README.md`
3. **Scenario 6A** (Config): Creates `.env.production` and changes DB password
4. **Scenario 5** (Structural): Comments out function and updates import

Each test:
- ✅ Makes the change
- ✅ Runs `crisk check`
- ✅ Captures output
- ✅ Validates results
- ✅ Resets git state

---

## Expected Results

### Best Case (All Features Working)

```
========================================
TEST EXECUTION SUMMARY
========================================
Total scenarios: 4
Passed: 4
Failed: 0
Success rate: 100%
```

### Current State (Basic Risk Assessment)

Even without Phase 0 implemented, you should see:

**Scenario 7 (Security):**
- Risk Level: HIGH or MEDIUM (auth file has coupling)
- Phase 2 may trigger if coupling > 10
- Output file: `output_scenario_7.txt`

**Scenario 10 (Docs):**
- Risk Level: LOW
- Fast execution (~200ms)
- Output file: `output_scenario_10.txt`

**Scenario 6A (Production Config):**
- Risk Level: Depends on detection (LOW to HIGH)
- May not detect "production" context yet
- Output file: `output_scenario_6a.txt`

**Scenario 5 (Structural):**
- Risk Level: MEDIUM or HIGH
- Multi-file change detected
- Output file: `output_scenario_5.txt`

---

## View Test Outputs

```bash
# View all outputs
cat test/integration/modification_type_tests/output_scenario_*.txt

# View specific output
cat test/integration/modification_type_tests/output_scenario_7.txt

# View test report
cat test/integration/modification_type_tests/test_report.txt
```

---

## If Tests Fail

### Error: "crisk binary not found"

Already fixed! Scripts now check both `./bin/crisk` and `./crisk`

### Error: "Git working directory not clean"

```bash
cd test_sandbox/omnara
git restore .
git clean -fd
cd ../..
```

### Error: "No files to check"

This means `crisk check` didn't find modified files. Check:
1. Omnara repo exists at `test_sandbox/omnara`
2. Test script's modifications were applied
3. Git is tracking the files

### Output shows "cannot connect to Neo4j"

```bash
# Start Docker services
docker compose up -d

# Wait a few seconds
sleep 5

# Re-run tests
./test/integration/modification_type_tests/run_all_tests.sh
```

---

## What Success Looks Like

Each test should show:

```
========================================
Scenario 7: Security-Sensitive Change (Type 9)
========================================

Using binary: ./bin/crisk
✅ Git working directory clean

Making security-sensitive changes to src/backend/auth/routes.py...
✅ Changes applied successfully

Running crisk check...
✅ crisk check completed

========================================
ACTUAL OUTPUT:
========================================
[... crisk output ...]

========================================
VALIDATION CHECKS:
========================================
✅ Risk level found: Risk Level: HIGH
✅ HIGH or CRITICAL risk level detected
...

✅ Git working directory restored to clean state

========================================
✅ Test completed
========================================
```

---

## Next Steps After Tests Run

1. **Review outputs** - Check if risk levels match expectations from TEST_PLAN.md

2. **Create expected outputs** (if actual outputs are correct):
   ```bash
   cd test/integration/modification_type_tests
   cp output_scenario_7.txt expected_scenario_7.txt
   cp output_scenario_10.txt expected_scenario_10.txt
   # etc.
   ```

3. **Re-run to compare**:
   ```bash
   ./run_all_tests.sh
   # Now generates diff files comparing actual vs expected
   ```

4. **Implement Phase 0** (optional enhancement):
   - See [MODIFICATION_TYPES_AND_TESTING.md](../../../dev_docs/03-implementation/testing/MODIFICATION_TYPES_AND_TESTING.md) §5
   - After implementation, re-run tests to validate improvements

---

## Useful Commands

```bash
# Run single test
./test/integration/modification_type_tests/scenario_7_security.sh

# Check omnara git status
cd test_sandbox/omnara && git status && cd ../..

# View crisk version
./bin/crisk --version

# Check Docker services
docker ps

# View test logs
cat test/integration/modification_type_tests/test_report.txt
```

---

**All tests are ready to run!** 🚀

The binary path issue is fixed. Just run:
```bash
./test/integration/modification_type_tests/run_all_tests.sh
```
