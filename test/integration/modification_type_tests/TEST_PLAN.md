# Modification Type Tests - Test Plan

**Purpose:** Automated testing plan for 12 modification type scenarios
**Repository:** test_sandbox/omnara
**Last Updated:** October 10, 2025

---

## Test Execution Flow

Each test follows this pattern:

```bash
1. Make controlled change to omnara codebase
2. Verify git status shows expected files changed
3. Run: crisk check <files>
4. Capture output
5. Reset changes (git restore)
6. Compare actual output to expected output
```

---

## Scenario Definitions

### Scenario 5: Structural Refactoring (Type 1)

**Objective:** Multi-file refactoring with import updates

**Target Files:**
- `src/backend/auth/utils.py` (move function to jwt_utils.py)
- `src/backend/auth/jwt_utils.py` (add moved function)
- `src/backend/auth/routes.py` (update import)

**Change Type:** Type 1B - Dependency Changes

**Specific Changes:**
```python
# In auth/utils.py: Comment out (simulate move)
# def update_user_profile(...): ...

# In auth/jwt_utils.py: Add
# def update_user_profile(user, data):
#     """Moved from utils.py"""
#     pass

# In auth/routes.py: Update import
# from .jwt_utils import create_api_key_jwt, get_token_hash, update_user_profile
```

**Expected Risk Assessment:**
```
Risk Level: ⚠️  HIGH

Modification Type: STRUCTURAL (Type 1B - Dependency Changes)
Base Risk: 0.7 (structural changes)

Metrics:
  Coupling: 3 files affected (refactoring pattern detected)
  Co-change frequency: 0.5-0.7 (auth files change together)
  Test coverage: 0.3-0.5 (moderate coverage)

Phase 1 completed in <200ms

Escalating to Phase 2 (multi-file structural change detected)...

Key Evidence:
1. [structural] 3-file refactoring detected
2. [modification_type] Type 1B: Dependency reorganization
3. [coupling] Auth module has multiple dependents
4. [recommendation] Run authentication test suite

Risk Level: HIGH (confidence: 0.85-0.92)

Recommendations:
  Critical:
    1. Run full authentication test suite
    2. Verify all import paths updated correctly
    3. Check for circular dependency introduction

  High Priority:
    1. Add integration test for auth flow
    2. Deploy to staging environment
```

---

### Scenario 6A: Production Configuration Change (Type 3)

**Objective:** High-risk production environment configuration

**Target File:**
- `.env.production` (create if not exists, or modify .env.example)

**Change Type:** Type 3A - Environment Configuration

**Specific Changes:**
```bash
# Change PRODUCTION_DB_URL value
PRODUCTION_DB_URL=postgresql://postgres:NEWPASSWORD@db.example.com:5432/postgres
```

**Expected Risk Assessment:**
```
Risk Level: 🔴 CRITICAL

Modification Type: CONFIGURATION (Type 3A - Environment Configuration)
Base Risk: 0.5 → Escalated to 1.0 (production environment detected)

Configuration Analysis:
  Environment: PRODUCTION (high-risk environment)
  Changed values: PRODUCTION_DB_URL
  Sensitive: YES (database credentials)
  File pattern: .env* file

Metrics:
  Coupling: N/A (configuration file)
  Impact: CRITICAL (affects all database connections)

Recommendations:
  Critical:
    1. Verify connection string syntax
    2. Test database connectivity in staging first
    3. Have rollback plan ready
    4. Notify on-call team before deployment

  High Priority:
    1. Verify database access permissions
    2. Update connection pooler settings if needed
```

---

### Scenario 6B: Development Configuration Change (Type 3)

**Objective:** Low-risk development environment configuration

**Target File:**
- `.env.example` (change DEVELOPMENT_DB_URL)

**Change Type:** Type 3A - Environment Configuration

**Specific Changes:**
```bash
# Change DEVELOPMENT_DB_URL value
DEVELOPMENT_DB_URL=postgresql://user:password@localhost:5433/agent_dashboard
```

**Expected Risk Assessment:**
```
Risk Level: ✅ LOW

Modification Type: CONFIGURATION (Type 3A - Environment Configuration)
Base Risk: 0.5 → Reduced to 0.2 (development environment)

Configuration Analysis:
  Environment: DEVELOPMENT (low-risk environment)
  Changed values: DEVELOPMENT_DB_URL
  Sensitive: NO (local development only)
  Impact: LOCAL (affects only development setup)

Metrics:
  Coupling: N/A (example config file)
  Impact: LOW (documentation/template file)

Recommendations:
  Standard:
    1. Restart development server to apply changes
    2. Update documentation if connection changed
```

---

### Scenario 7: Security-Sensitive Change (Type 9)

**Objective:** Authentication logic modification with security keywords

**Target File:**
- `src/backend/auth/routes.py`

**Change Type:** Type 9A - Authentication Changes

**Specific Changes:**
```python
# In sync_user function, add new validation
@router.post("/sync-user")
async def sync_user(
    request: SyncUserRequest,
    current_user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
):
    """Sync user from Supabase to our database"""
    # ADD THIS LINE:
    # TODO: Add session timeout validation here

    # Verify the requesting user matches the user being synced
    if str(current_user.id) != request.id:
        raise HTTPException(status_code=403, detail="Cannot sync different user")
```

**Keywords Detected:** `sync`, `user`, `auth`, `current_user`, `HTTPException` (security context)

**Expected Risk Assessment:**
```
Risk Level: 🔴 CRITICAL

⚠️  Security-sensitive keywords detected: auth, user, session, validate

Phase 0: Pre-Analysis
  Modification Type: SECURITY (Type 9A - Authentication)
  Base Risk: 1.0 (CRITICAL)
  Force Escalate: YES (security changes require LLM review)
  File path indicates auth module: src/backend/auth/routes.py

Proceeding to Phase 2 (skipping Phase 1 baseline)...

Key Evidence:
1. [security] Authentication route modification
2. [file_location] Critical auth module: backend/auth/routes.py
3. [coupling] Auth routes serve all authenticated users
4. [modification] Function: sync_user (user session management)

Risk Level: CRITICAL (confidence: 0.95)

Recommendations:
  Critical (must do before commit):
    1. Conduct security review with security team
    2. Test authentication flow with multiple user roles
    3. Verify session timeout behavior
    4. Check for authentication bypass vulnerabilities
    5. Add security regression tests for sync_user endpoint

  High Priority:
    1. Deploy to staging with monitoring
    2. Add rate limiting to prevent abuse
    3. Log authentication attempts
```

---

### Scenario 8: Performance Optimization (Type 10)

**Objective:** Add caching to database query

**Target File:**
- `src/backend/db/queries.py` (if exists) or `src/shared/database/models.py`

**Change Type:** Type 10A - Algorithm Optimization

**Specific Changes:**
```python
# Add caching decorator (simulated)
# from functools import lru_cache
#
# @lru_cache(maxsize=100)
# def get_user_by_id(db: Session, user_id: str) -> User:
#     """Get user by ID with caching"""
#     return db.query(User).filter(User.id == user_id).first()
```

**Keywords Detected:** `cache`, `lru_cache`, `query`, `performance`

**Expected Risk Assessment:**
```
Risk Level: ⚠️  HIGH

Modification Type: PERFORMANCE (Type 10A - Algorithm Optimization)
Base Risk: 0.7 (performance-critical)

Performance Analysis:
  Changed function: get_user_by_id (database query)
  Optimization: Added LRU caching layer
  Potential issues: Cache invalidation, stale data, memory usage

Metrics:
  Coupling: 5-8 files call get_user_by_id
  Test coverage: 0.4-0.6 (needs cache-specific tests)

Key Evidence:
1. [performance] Caching added to database query
2. [keywords] lru_cache, maxsize detected
3. [risk] Stale data if user updated elsewhere
4. [recommendation] Cache invalidation strategy needed

Recommendations:
  Critical:
    1. Add tests for cache hit/miss scenarios
    2. Add tests for cache invalidation logic
    3. Verify cache TTL is appropriate
    4. Test with concurrent user updates

  High Priority:
    1. Load test to verify performance improvement
    2. Monitor cache hit rate in staging
    3. Add cache metrics/observability
    4. Document cache invalidation strategy
```

---

### Scenario 9: Multi-Type Change (Type 2+5+6+9)

**Objective:** Combined security + behavioral + testing + docs change

**Target Files:**
- `src/backend/auth/routes.py` (behavioral + security)
- `src/backend/tests/test_auth.py` (create if needed - testing)
- `README.md` (documentation)

**Change Types:** Type 9 (Security) + Type 2 (Behavioral) + Type 5 (Testing) + Type 6 (Documentation)

**Specific Changes:**
```python
# auth/routes.py: Add error handling (Type 2 + Type 9)
try:
    if str(current_user.id) != request.id:
        raise HTTPException(status_code=403, detail="Cannot sync different user")
except Exception as e:
    logger.error(f"Auth error: {e}")
    raise

# tests/test_auth.py: Add test (Type 5)
# def test_sync_user_unauthorized():
#     """Test unauthorized sync attempt"""
#     pass

# README.md: Add note (Type 6)
# ## Authentication
# The sync-user endpoint validates user identity before syncing.
```

**Expected Risk Assessment:**
```
Risk Level: 🔴 CRITICAL

Multi-Type Change Detected:
  Primary: Type 9 (Security) - auth/routes.py
  Secondary: Type 2 (Behavioral) - auth/routes.py
  Supporting: Type 5 (Testing) - tests/test_auth.py
  Supporting: Type 6 (Documentation) - README.md

Risk Calculation:
  auth/routes.py: 1.0 (Type 9 security + Type 2 behavioral)
  test_auth.py: 0.2 (Type 5 test addition) × 0.3 = 0.06
  README.md: 0.1 (Type 6 docs) × 0.3 = 0.03

  Final Risk = MAX(1.0, 0.2, 0.1) + (0.06 + 0.03) = 1.09 → capped at 1.0

Overall Assessment:
  - Security change dominates risk profile
  - Error handling adds behavioral complexity
  - Test additions reduce risk slightly (good practice ✅)
  - Documentation additions are positive (✅)

Risk Level: CRITICAL (confidence: 0.95)

Recommendations:
  [Same as Scenario 7 - Security-sensitive, plus:]
  - Verify error handling doesn't leak sensitive info
  - Ensure test coverage for new error paths
```

---

### Scenario 10: Documentation-Only Change (Type 6)

**Objective:** Zero-runtime-impact documentation update

**Target Files:**
- `README.md` (update docs)
- `docs/guides/authentication.md` (if exists, create minimal change)

**Change Type:** Type 6B - External Documentation

**Specific Changes:**
```markdown
# README.md: Add section
## Development Setup
1. Install dependencies: `pip install -r requirements-dev.txt`
2. Set up environment: Copy `.env.example` to `.env`
3. Run tests: `pytest tests/`
```

**Expected Risk Assessment:**
```
Phase 0: Pre-Analysis

Documentation-Only Change Detected (Type 6)
  Files: README.md
  Content: Markdown only, no code changes
  Runtime Impact: ZERO

Skipping Phase 1 and Phase 2 (no risk analysis needed)

Risk Level: ✅ LOW

Assessment: Safe to commit (documentation improvements)

Performance: <10ms total (no graph queries executed)
```

---

### Scenario 11: Ownership Risk (Type 8)

**Objective:** New contributor's first change to complex file

**Target File:**
- `src/backend/auth/jwt_utils.py`

**Change Type:** Type 8A - New Contributor Changes

**Specific Changes:**
```python
# Add comment simulating new contributor work
# Author: alice@example.com (NEW CONTRIBUTOR)
# TODO: Refactor token expiration logic for better performance
```

**Git Author Override:**
```bash
GIT_AUTHOR_NAME="Alice Newbie" GIT_AUTHOR_EMAIL="alice@example.com"
```

**Expected Risk Assessment:**
```
Risk Level: ⚠️  MODERATE → HIGH (ownership factor)

Modification Type: BEHAVIORAL (Type 2) + OWNERSHIP (Type 8A)

Ownership Analysis:
  Current author: Alice Newbie (alice@example.com)
  File history: 0 prior commits by this author
  Primary owner: (Unknown - would query git log)
  File: jwt_utils.py (authentication utilities)
  Complexity: Moderate to High (JWT, tokens, security)

Risk Factors:
  - New contributor to this file (ownership risk)
  - Security-sensitive file (JWT authentication)
  - No prior familiarity with auth system

Metrics:
  Coupling: 3-5 files import from jwt_utils
  Test coverage: 0.4-0.6 (moderate)
  Incidents: (would check incident database)

Recommendations:
  Critical:
    1. Request review from auth module owner
    2. Pair program or knowledge transfer session
    3. Review JWT security best practices
    4. Add extra test coverage for changed sections

  High Priority:
    1. Review file documentation
    2. Ensure understanding of token lifecycle
    3. Verify error handling for token operations
```

---

### Scenario 12: Temporal Hotspot (Type 7)

**Objective:** High-churn file with incident history

**Target File:**
- `apps/web/src/integrations/supabase/client.ts` (or similar frequently changed file)

**Change Type:** Type 7A + 7B - Hotspot + Incident-Prone

**Specific Changes:**
```typescript
// Add connection retry logic
export const supabase = createClient(
  supabaseUrl,
  supabaseAnonKey,
  {
    // Add this:
    // auth: { autoRefreshToken: true, persistSession: true }
  }
)
```

**Simulation:** File would need artificial git history (15 commits in 30 days) - we'll simulate by documenting expected state

**Expected Risk Assessment:**
```
Risk Level: ⚠️  HIGH

Modification Type: TEMPORAL HOTSPOT (Type 7A + 7B)

Temporal Analysis:
  Churn rate: 15+ changes in last 30 days (HIGH) [simulated]
  Recent changes: File modified frequently
  Integration point: Supabase client (external dependency)
  Impact: All authenticated users

Key Evidence:
1. [temporal] High churn indicates ongoing issues [would be from git log]
2. [integration] External service integration (Supabase)
3. [coupling] Client used across entire web app
4. [modification] Authentication configuration change

Metrics:
  Coupling: 10+ files import supabase client
  Co-change frequency: 0.6-0.8 (changes with other integration files)
  Test coverage: 0.3-0.5 (integration points often undertested)

Recommendations:
  Critical:
    1. Verify autoRefreshToken doesn't conflict with session management
    2. Test session persistence across page reloads
    3. Add integration tests for auth flow

  High Priority:
    1. Monitor for authentication errors in staging
    2. Check if this addresses previous issues
    3. Add client-side error logging
    4. Consider stabilization refactor if churn continues
```

---

## Test Script Template

Each test script follows this structure:

```bash
#!/bin/bash
# test_scenario_X.sh

set -e  # Exit on error

SCENARIO_NAME="Scenario X: Description"
TEST_DIR="test_sandbox/omnara"
CRISK_BIN="./crisk"

echo "========================================"
echo "$SCENARIO_NAME"
echo "========================================"

# 1. Verify we're in correct directory
cd "$TEST_DIR" || exit 1

# 2. Ensure clean git state
git status --short
if [ -n "$(git status --porcelain)" ]; then
    echo "ERROR: Git working directory not clean"
    exit 1
fi

# 3. Make controlled changes
echo "Making changes..."
# [Specific changes for scenario]

# 4. Verify changes
echo "Verifying git diff..."
git status --short
git diff --stat

# 5. Run crisk check
echo "Running crisk check..."
cd ../.. # Back to coderisk-go root
"$CRISK_BIN" check "test_sandbox/omnara/<file>" > "test/integration/modification_type_tests/output_scenario_X.txt" 2>&1 || true

# 6. Display output
echo "Actual Output:"
cat "test/integration/modification_type_tests/output_scenario_X.txt"

# 7. Reset changes
echo "Resetting changes..."
cd "$TEST_DIR"
git restore .

# 8. Verify clean state
if [ -n "$(git status --porcelain)" ]; then
    echo "WARNING: Git working directory not fully restored"
fi

echo "✅ Test completed"
echo ""
```

---

## Expected Output Files

Each test will generate:
1. `output_scenario_X.txt` - Actual crisk output
2. `expected_scenario_X.txt` - Expected output (created manually first)
3. `diff_scenario_X.txt` - Comparison results

---

## Validation Strategy

After running all tests:

```bash
# Compare actual vs expected
for i in {5..12}; do
    echo "Validating Scenario $i..."
    diff -u expected_scenario_$i.txt output_scenario_$i.txt > diff_scenario_$i.txt || true

    # Check key phrases
    grep -q "Risk Level:" output_scenario_$i.txt && echo "✅ Risk level found" || echo "❌ Risk level missing"
    grep -q "Modification Type:" output_scenario_$i.txt && echo "✅ Mod type found" || echo "❌ Mod type missing"
done
```

---

## Next Steps

1. Create expected output files for each scenario
2. Implement test scripts (scenario_5.sh through scenario_12.sh)
3. Create master runner script (run_all_modification_tests.sh)
4. Run tests and compare outputs
5. Tune expected outputs based on actual system behavior
6. Document any discrepancies and system improvements needed

---

**Related Files:**
- [MODIFICATION_TYPES_AND_TESTING.md](../../../dev_docs/03-implementation/testing/MODIFICATION_TYPES_AND_TESTING.md)
- [TESTING_EXPANSION_SUMMARY.md](../../../dev_docs/03-implementation/testing/TESTING_EXPANSION_SUMMARY.md)
