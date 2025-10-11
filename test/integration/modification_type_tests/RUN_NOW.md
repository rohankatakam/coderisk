# ⚡ Run Tests Now - Quick Guide

## Status: ✅ All Scripts Fixed!

The binary path issues are resolved. Scripts now work with `./bin/crisk`.

---

## 🚀 Step 1: One-Time Setup

Run this **once** to set up the test environment:

```bash
./test/integration/modification_type_tests/setup_tests.sh
```

This will:
1. ✅ Verify `crisk` binary exists (builds if needed)
2. ✅ Start Docker services
3. ✅ Clone omnara repository to `test_sandbox/omnara`
4. ✅ Clean git working directory
5. ✅ Make all test scripts executable

**Expected output:**
```
========================================
Modification Type Tests - Setup
========================================

✅ Running from coderisk-go root
✅ crisk binary exists at ./bin/crisk
✅ Docker services running
✅ omnara repository cloned
✅ omnara working directory clean
✅ Test scripts are executable

========================================
✅ Setup Complete!
========================================
```

---

## 🧪 Step 2: Run All Tests

After setup completes:

```bash
./test/integration/modification_type_tests/run_all_tests.sh
```

**Expected output (success):**
```
========================================
Modification Type Tests - Master Runner
Started: 2025-10-10 XX:XX:XX
========================================

========================================
Running Scenario 7: Security-Sensitive Change (Type 9)
========================================

Using binary: ./bin/crisk
✅ Git working directory clean
Making security-sensitive changes...
✅ Changes applied successfully
Running crisk check...
✅ crisk check completed

[... crisk output ...]

✅ Git working directory restored to clean state
✅ Test completed

[... more scenarios ...]

========================================
TEST EXECUTION SUMMARY
========================================
Total scenarios: 4
Passed: 4  ← Should see 4 passed!
Failed: 0
Success rate: 100%
```

---

## 📊 Step 3: Review Outputs

Check the actual outputs:

```bash
# View all outputs
ls -lh test/integration/modification_type_tests/output_scenario_*.txt

# View specific output
cat test/integration/modification_type_tests/output_scenario_7.txt

# View test report
cat test/integration/modification_type_tests/test_report.txt
```

---

## 🐛 Troubleshooting

### Issue: "Git working directory not clean"

The omnara repo has uncommitted changes.

**Fix:**
```bash
cd test_sandbox/omnara
git restore .
git clean -fd
cd ../..
./test/integration/modification_type_tests/run_all_tests.sh
```

### Issue: "omnara repository not found"

The setup script didn't clone it.

**Fix:**
```bash
mkdir -p test_sandbox
cd test_sandbox
git clone https://github.com/omnara-ai/omnara.git
cd ..
./test/integration/modification_type_tests/run_all_tests.sh
```

### Issue: "crisk binary not found"

The build didn't complete.

**Fix:**
```bash
make build
# Verify
ls -lh ./bin/crisk
./test/integration/modification_type_tests/run_all_tests.sh
```

### Issue: "Docker services not running"

**Fix:**
```bash
docker compose up -d
sleep 5  # Wait for services to start
./test/integration/modification_type_tests/run_all_tests.sh
```

---

## ✅ What Success Looks Like

Each test should:

1. ✅ Find the binary at `./bin/crisk`
2. ✅ Verify git is clean
3. ✅ Make controlled changes
4. ✅ Run `crisk check`
5. ✅ Capture output
6. ✅ Show validation checks
7. ✅ Reset git state

**Output files generated:**
- `output_scenario_5.txt` - Structural refactoring
- `output_scenario_6a.txt` - Production config
- `output_scenario_7.txt` - Security-sensitive
- `output_scenario_10.txt` - Documentation-only
- `test_report.txt` - Summary

---

## 📖 After Tests Pass

1. **Review outputs** against [TEST_PLAN.md](TEST_PLAN.md) expectations

2. **Create expected outputs**:
   ```bash
   cd test/integration/modification_type_tests
   cp output_scenario_7.txt expected_scenario_7.txt
   cp output_scenario_10.txt expected_scenario_10.txt
   cp output_scenario_6a.txt expected_scenario_6a.txt
   cp output_scenario_5.txt expected_scenario_5.txt
   ```

3. **Re-run to compare**:
   ```bash
   ./run_all_tests.sh
   # Now generates diff_scenario_X.txt files
   ```

---

## 🎯 Quick Command Reference

```bash
# Setup (once)
./test/integration/modification_type_tests/setup_tests.sh

# Run all tests
./test/integration/modification_type_tests/run_all_tests.sh

# Run single test
./test/integration/modification_type_tests/scenario_7_security.sh

# Clean omnara if needed
cd test_sandbox/omnara && git restore . && git clean -fd && cd ../..

# View outputs
cat test/integration/modification_type_tests/output_scenario_*.txt

# View report
cat test/integration/modification_type_tests/test_report.txt
```

---

**Ready to run!** Start with the setup script:

```bash
./test/integration/modification_type_tests/setup_tests.sh
```
