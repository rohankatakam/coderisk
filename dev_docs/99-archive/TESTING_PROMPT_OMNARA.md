# Testing Prompt: Omnara Repository Validation

**Created:** 2025-10-13
**Purpose:** Comprehensive testing of CodeRisk on omnara repo (medium-sized, ~2.4K stars)
**Session type:** Single Claude Code session, systematic validation

---

## Context

You've already tested CodeRisk on the omnara repository with the following results:

**What worked:**
- Graph ingestion completed (init-local took ~15 minutes)
- Phase 0 security keyword detection worked
- Phase 1 baseline assessment completed
- Phase 2 deep investigation ran successfully
- Detected CRITICAL risk for API key logging

**What needs validation:**
1. All 10 modification types (currently only tested Security type)
2. False positive rate measurement
3. Performance benchmarks across different change sizes
4. Edge cases and error handling

---

## Prerequisites

Before starting this testing session:

```bash
# 1. Verify infrastructure is running
docker compose ps
# Expected: Neo4j, PostgreSQL, Redis all healthy

# 2. Verify API key is set
echo $OPENAI_API_KEY
# Expected: sk-...

# 3. Verify omnara repo is initialized
cd /path/to/omnara
crisk check --version
# Expected: Version number displayed
```

---

## Testing Tasks

### Task 1: Modification Type Coverage (10 types)

Test all 10 modification types from Phase 0 detection:

#### 1.1 Security Changes (Already Tested ✅)

**Goal:** Verify security keyword detection and escalation

```bash
# Already validated - auth code + API key logging detected as CRITICAL
# Confirmation: Phase 0 correctly detected security keywords
```

#### 1.2 Documentation-Only Changes

**Goal:** Verify docs-only changes get LOW risk and skip expensive analysis

```bash
# Create test change
cd /path/to/omnara
git checkout -b test/docs-only

# Modify README
echo "\n## Additional Section\nSome documentation content." >> README.md

# Check without committing (if supported)
git add README.md
crisk check

# Expected output:
# - Phase 0: Detected DOCS_ONLY modification
# - Risk: LOW
# - Phase 2: Skipped (not worth $0.05 LLM call)
```

**Validation criteria:**
- [ ] Phase 0 detects DOCS_ONLY
- [ ] Risk assessment: LOW
- [ ] Phase 2 skipped or minimal investigation
- [ ] Check completes in <1 second

#### 1.3 Configuration Changes

**Goal:** Verify config file changes get HIGH risk and force escalation

```bash
# Create test change
git checkout -b test/config-change

# Modify .env or config file
echo "NEW_API_ENDPOINT=https://api.example.com" >> .env.example

git add .env.example
crisk check

# Expected output:
# - Phase 0: Detected CONFIG modification
# - Risk: HIGH (force escalate)
# - Phase 2: Full investigation even if Phase 1 shows LOW
```

**Validation criteria:**
- [ ] Phase 0 detects CONFIG
- [ ] Risk assessment: HIGH or forced escalation
- [ ] Phase 2 runs even if metrics don't indicate risk

#### 1.4 Structural Changes (Refactoring)

**Goal:** Verify refactoring without logic changes gets LOW-MEDIUM risk

```bash
# Create test change
git checkout -b test/refactor

# Rename a function (no logic change)
# Find a simple utility function and rename it
# Example: findById -> findRecordById

# Update all call sites
git add -A
crisk check

# Expected output:
# - Phase 0: Detected STRUCTURAL modification
# - Risk: LOW-MEDIUM (depends on coupling)
# - Phase 2: Investigates coupling but finds no behavioral change
```

**Validation criteria:**
- [ ] Phase 0 detects STRUCTURAL or REFACTORING
- [ ] Risk assessment considers coupling but not behavior
- [ ] Phase 2 mentions "no logic change" or similar

#### 1.5 Behavioral Changes (Logic)

**Goal:** Verify logic changes get MEDIUM-HIGH risk with thorough investigation

```bash
# Create test change
git checkout -b test/behavioral

# Modify business logic
# Example: Change validation logic in a function
# Find a validation function and change a condition

git add -A
crisk check

# Expected output:
# - Phase 0: Detected BEHAVIORAL modification
# - Risk: MEDIUM-HIGH
# - Phase 2: Full investigation of dependencies and tests
```

**Validation criteria:**
- [ ] Phase 0 detects BEHAVIORAL
- [ ] Risk assessment: MEDIUM-HIGH
- [ ] Phase 2 investigates dependencies and test coverage

#### 1.6 Interface Changes (API/Schema)

**Goal:** Verify API or schema changes get HIGH risk

```bash
# Create test change
git checkout -b test/interface

# Modify API endpoint signature or database schema
# Example: Add a new required parameter to an API route

git add -A
crisk check

# Expected output:
# - Phase 0: Detected INTERFACE modification
# - Risk: HIGH
# - Phase 2: Investigates all callers and consumers
```

**Validation criteria:**
- [ ] Phase 0 detects INTERFACE
- [ ] Risk assessment: HIGH
- [ ] Phase 2 investigates breaking changes

#### 1.7 Testing Changes

**Goal:** Verify test additions get LOW risk and skip investigation

```bash
# Create test change
git checkout -b test/add-tests

# Add new test cases
# Find a test file and add a new test

git add -A
crisk check

# Expected output:
# - Phase 0: Detected TESTING modification
# - Risk: LOW
# - Phase 2: Skipped or minimal investigation
```

**Validation criteria:**
- [ ] Phase 0 detects TESTING
- [ ] Risk assessment: LOW
- [ ] Check completes quickly (<2 seconds)

#### 1.8 Temporal Patterns (High Churn)

**Goal:** Verify files with high churn get elevated risk

```bash
# Query graph for high-churn files
# (May need to use Neo4j directly)

# Modify a file that has high churn history
# Check git log to find frequently changed files

git log --name-only --pretty=format: | sort | uniq -c | sort -rn | head -10

# Pick a high-churn file and make a change
git add -A
crisk check

# Expected output:
# - Phase 0: Detected TEMPORAL risk (high churn)
# - Risk: MEDIUM-HIGH (elevated due to churn)
# - Phase 2: Mentions churn pattern
```

**Validation criteria:**
- [ ] Phase 0 or Phase 1 detects high churn
- [ ] Risk elevated due to churn pattern
- [ ] Explanation mentions churn frequency

#### 1.9 Ownership Patterns (New Contributor)

**Goal:** Verify changes from new contributors get different assessment

```bash
# This may require testing with a different git author
# Check if CodeRisk considers author history

git config user.name "New Contributor"
git config user.email "new@example.com"

# Make a change
# ... modify a file ...

git add -A
crisk check

# Expected output:
# - Risk may be elevated for unfamiliar contributor
# - Or Phase 2 mentions authorship in context
```

**Validation criteria:**
- [ ] CodeRisk considers authorship (if implemented)
- [ ] New contributors may trigger additional checks

#### 1.10 Performance Changes (Caching, Concurrency)

**Goal:** Verify performance optimizations get appropriate risk level

```bash
# Create test change
git checkout -b test/performance

# Add caching or concurrency changes
# Example: Wrap a function with caching logic

git add -A
crisk check

# Expected output:
# - Phase 0: May detect PERFORMANCE or BEHAVIORAL
# - Risk: MEDIUM (concurrency risks, cache invalidation)
# - Phase 2: Investigates race conditions or cache correctness
```

**Validation criteria:**
- [ ] Risk assessment considers concurrency/caching risks
- [ ] Phase 2 mentions relevant concerns

---

### Task 2: False Positive Rate Measurement

**Goal:** Measure false positive rate on known-safe changes

#### 2.1 Pure Refactoring (No Logic Change)

```bash
# Make 5 different refactoring changes:
# 1. Rename variable
# 2. Extract function (no behavior change)
# 3. Reorder imports
# 4. Format code (add newlines, fix indentation)
# 5. Add type annotations

# For each change:
git checkout -b test/fp-refactor-N
# ... make change ...
git add -A
crisk check
# Record: Did it flag as HIGH risk? (False positive if yes)
```

**Expected:** All should be LOW or MEDIUM, not HIGH or CRITICAL

#### 2.2 Documentation Updates

```bash
# Make 5 different documentation changes:
# 1. Update README
# 2. Add code comments
# 3. Update JSDoc/docstrings
# 4. Fix typos in comments
# 5. Add inline explanations

# For each change:
# ... make change ...
crisk check
# Record: Did it flag as MEDIUM+ risk? (False positive if yes)
```

**Expected:** All should be LOW

#### 2.3 Test Additions (No Production Code Change)

```bash
# Add 5 new test cases in different test files
# Make sure NO production code changes

# For each test addition:
crisk check
# Record: Did it flag as MEDIUM+ risk? (False positive if yes)
```

**Expected:** All should be LOW

#### 2.4 False Positive Rate Calculation

```bash
# Total safe changes: 15 (5 refactors + 5 docs + 5 tests)
# False positives: Count of MEDIUM+ flags on safe changes
# FP Rate = (False Positives / Total Safe Changes) * 100%

# Target: <3% FP rate
# Expected: 0-1 false positives out of 15 tests
```

---

### Task 3: Performance Benchmarks

**Goal:** Measure check performance across different change sizes

#### 3.1 Single File Change

```bash
# Modify 1 file
git checkout -b test/perf-single
# ... modify 1 file ...
git add -A
time crisk check

# Record:
# - Total time: ___ seconds
# - Phase 0: ___ ms
# - Phase 1: ___ seconds
# - Phase 2: ___ seconds
```

**Expected:** 2-5 seconds total

#### 3.2 Small Change (3-5 files)

```bash
# Modify 3-5 related files
git checkout -b test/perf-small
# ... modify 3-5 files ...
git add -A
time crisk check

# Record timings
```

**Expected:** 3-7 seconds total

#### 3.3 Medium Change (10-15 files)

```bash
# Modify 10-15 files
git checkout -b test/perf-medium
# ... modify 10-15 files ...
git add -A
time crisk check

# Record timings
```

**Expected:** 5-10 seconds total

#### 3.4 Large Change (30+ files)

```bash
# Modify 30+ files (e.g., rename across project)
git checkout -b test/perf-large
# ... modify 30+ files ...
git add -A
time crisk check

# Record timings
```

**Expected:** 8-15 seconds total

---

### Task 4: Edge Cases and Error Handling

#### 4.1 Empty Diff (No Changes)

```bash
git checkout main
crisk check

# Expected:
# - Should detect no changes
# - Should not call LLM (no cost)
# - Should complete in <1 second
```

#### 4.2 Binary File Changes

```bash
git checkout -b test/binary
# Add or modify an image file
cp /path/to/image.png ./assets/
git add assets/image.png
crisk check

# Expected:
# - Should skip binary files or handle gracefully
# - Should not crash
```

#### 4.3 Very Large File (10,000+ lines)

```bash
# Modify a very large file (if one exists)
# Or create a large test file
crisk check

# Expected:
# - Should handle large diffs
# - Should not timeout
# - Should not exceed token limits
```

#### 4.4 Git Not Initialized

```bash
cd /tmp
mkdir test-no-git
cd test-no-git
crisk check

# Expected:
# - Should show clear error: "Not a git repository"
# - Should not crash
```

#### 4.5 Graph Not Initialized

```bash
cd /tmp
git clone https://github.com/some/repo
cd repo
crisk check

# Expected:
# - Should show error: "Graph not initialized. Run: crisk init-local"
# - Should not crash
```

#### 4.6 API Key Not Set

```bash
unset OPENAI_API_KEY
crisk check

# Expected:
# - Phase 0: Should still run (<50ms)
# - Phase 1: Should fail with clear error: "OPENAI_API_KEY not set"
# - Should show instructions to set API key
```

#### 4.7 Neo4j Not Running

```bash
docker compose stop neo4j
crisk check

# Expected:
# - Phase 0: Should still run
# - Phase 1: Should fail with error: "Cannot connect to graph database"
# - Should show instructions: "Run: docker compose up -d"
```

---

## Validation Checklist

After completing all tasks:

- [ ] All 10 modification types tested and working
- [ ] False positive rate measured and <3%
- [ ] Performance benchmarks recorded and within expectations
- [ ] All edge cases handled gracefully (no crashes)
- [ ] Error messages are clear and actionable
- [ ] Documentation matches actual behavior

---

## Reporting

### Create Testing Report

After completing all tasks, create a testing report:

**File:** `TESTING_REPORT_OMNARA.md`

**Contents:**
1. Summary of results
2. Modification type coverage table (10 types, pass/fail)
3. False positive rate calculation
4. Performance benchmarks table
5. Edge cases tested and results
6. Issues found (if any)
7. Recommended fixes or improvements

---

## Next Steps After Omnara Testing

Based on results:

1. **If all tests pass:**
   - Proceed with small repo test (100-500 files)
   - Proceed with large repo test (5K-10K files)
   - Update documentation with validated performance numbers

2. **If issues found:**
   - Create GitHub issues for each bug
   - Prioritize fixes (critical vs nice-to-have)
   - Re-test after fixes

3. **Update launch strategy:**
   - Confirm 17-minute setup time
   - Confirm $0.03-0.05/check cost
   - Confirm <3% FP rate

---

## Claude Code Session Instructions

**Prompt for testing session:**

```
I need you to systematically test CodeRisk on the omnara repository.

Context:
- CodeRisk is already installed and working
- Graph has been initialized (init-local completed)
- Docker infrastructure is running (Neo4j, PostgreSQL, Redis)
- OpenAI API key is set

Tasks:
1. Test all 10 modification types (see TESTING_PROMPT_OMNARA.md)
2. Measure false positive rate on 15 known-safe changes
3. Benchmark performance across different change sizes
4. Test edge cases and error handling

Instructions:
1. Read TESTING_PROMPT_OMNARA.md for full details
2. Work through each task systematically
3. Record all results in TESTING_REPORT_OMNARA.md
4. Flag any issues or unexpected behavior

Expected duration: 2-3 hours

Please start with Task 1: Modification Type Coverage.
```

---

## Reference Files

**Architecture:**
- [dev_docs/01-architecture/decisions/005-confidence-driven-investigation.md](dev_docs/01-architecture/decisions/005-confidence-driven-investigation.md) - Phase 0 pre-analysis details
- [dev_docs/01-architecture/agentic_design.md](dev_docs/01-architecture/agentic_design.md) - Graph structure and LLM navigation

**Testing Strategy:**
- [dev_docs/03-implementation/testing/MODIFICATION_TYPES_AND_TESTING.md](dev_docs/03-implementation/testing/MODIFICATION_TYPES_AND_TESTING.md) - Full taxonomy of modification types

**Current Status:**
- [CORRECTED_LAUNCH_STRATEGY_V2.md](CORRECTED_LAUNCH_STRATEGY_V2.md) - Current understanding of requirements and setup

---

**Ready to execute?** Run the testing session with omnara repo and report back findings.
