# 🔧 Quick Fix - Clone Omnara Repository

## Issue

The test scripts expect the **full omnara repository** at `test_sandbox/omnara`, but it's not there yet.

The `src/omnara/` directory is just a Python package, not the full repo.

## Solution

Run these commands to clone the full repository:

```bash
# From coderisk-go root
mkdir -p test_sandbox
cd test_sandbox
git clone https://github.com/omnara-ai/omnara.git
cd omnara

# Force clean any existing changes
git fetch origin
git reset --hard origin/main
git clean -fdx

# Go back to root
cd ../..
```

## Verify

```bash
# Check the repo exists
ls -la test_sandbox/omnara/

# Check it's clean
cd test_sandbox/omnara && git status

# You should see:
# On branch main
# Your branch is up to date with 'origin/main'.
#
# nothing to commit, working tree clean
```

## Then Run Tests

```bash
# From coderisk-go root
./test/integration/modification_type_tests/run_all_tests.sh
```

---

## Alternative: Update Test Scripts to Use Existing Omnara

If you have the omnara repo cloned elsewhere, you can update the test scripts:

**Option 1: Use existing omnara clone**

If you already have omnara cloned somewhere:

```bash
# Create symlink
ln -s /path/to/your/omnara test_sandbox/omnara
```

**Option 2: Update TEST_DIR in all scripts**

Edit each `scenario_*.sh` file and change:
```bash
TEST_DIR="test_sandbox/omnara"
```

To wherever your omnara repo actually is.

---

## Fastest Solution (Copy-Paste)

```bash
mkdir -p test_sandbox && \
cd test_sandbox && \
git clone https://github.com/omnara-ai/omnara.git && \
cd omnara && \
git reset --hard origin/main && \
git clean -fdx && \
cd ../.. && \
./test/integration/modification_type_tests/run_all_tests.sh
```

This will:
1. Create test_sandbox directory
2. Clone omnara repository
3. Reset to clean state
4. Run all tests

**Expected time:** ~30 seconds (depending on network speed)
