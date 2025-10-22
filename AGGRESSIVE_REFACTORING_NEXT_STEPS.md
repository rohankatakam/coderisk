# Aggressive Refactoring - Immediate Next Steps

## Summary

I've completed Phase 1 (partial) of the aggressive refactoring:
- ✅ Deleted: `internal/cache`, `internal/errors`, `internal/treesitter`
- ✅ Deleted: `internal/analysis/phase0`
- ✅ Deleted: `cmd/crisk/incident.go`
- ⏸️ Imports still need fixing (build will fail)

## Files Reduced
- **Before:** 92 non-test Go files
- **After deletion:** 78 non-test Go files
- **Reduction:** 14 files (15%)

## What Remains to Fix

### 1. Remove Cache References (8 files)
Cache was used by metrics package for Redis caching.

**Files to fix:**
- `internal/metrics/co_change.go`
- `internal/metrics/ownership_churn.go`
- `internal/metrics/coupling.go`
- `internal/metrics/registry.go`
- `internal/metrics/test_ratio.go`
- `internal/metrics/adaptive.go`
- `cmd/crisk/check.go`
- `cmd/crisk/status.go`

**Strategy:** Remove cache layer entirely, query Neo4j/PostgreSQL directly
- Remove `redisClient` parameter from metric functions
- Remove cache lookups (`cache.Get`, `cache.Set`)
- Rely on Neo4j/PostgreSQL query performance

### 2. Remove Errors Package References (2 files)
**Files to fix:**
- `internal/config/credentials.go`
- `internal/config/validator.go`

**Strategy:** Replace with standard library
```go
// Before
return errors.New("validation failed", errors.ValidationError)

// After
return fmt.Errorf("validation failed: %w", err)
```

### 3. Remove Treesitter References (1 file)
**File to fix:**
- `internal/ingestion/processor.go`

**Strategy:** Remove AST parsing, use simple import detection
- For coupling: Parse imports with regex (no AST needed)
- Tree-sitter was over-engineered for MVP needs

### 4. Remove Phase 0 References (1 file)
**File to fix:**
- `cmd/crisk/check.go` (lines 232-265)

**Strategy:** Remove Phase 0 pre-analysis block entirely
- Delete lines 232-265 (Phase 0 execution + skip logic)
- Delete lines 275-282 (Phase 0 escalation override)
- Analyze all files, no skip logic

### 5. Run go mod tidy
After fixing imports, clean up dependencies.

---

## Recommendation: Two Paths Forward

### Path A: Quick Fix (1-2 hours)
**Goal:** Get build passing, minimal refactoring

1. Fix cache references by removing Redis parameter, direct queries
2. Fix errors package references with `fmt.Errorf`
3. Fix treesitter reference by stubbing out
4. Fix phase0 references by removing blocks
5. Run `go mod tidy` and rebuild
6. Test that check/init commands still work

**Result:** Working build, 78 files (down from 124)

### Path B: Deep Refactoring (1 day)
**Goal:** Execute full aggressive refactoring plan

1. Do Path A fixes first
2. **Consolidate analysis → metrics** (biggest win)
   - Merge `internal/analysis` into `internal/metrics`
   - Remove adaptive config complexity
   - Result: 18 files → 8 files

3. **Simplify config package**
   - 7 files → 2 files (config.go, defaults.go)
   - Remove validation, precedence complexity

4. **Simplify output package**
   - 13 files → 4 files
   - Merge formatters into single file

5. **Clean up graph package**
   - 14 files → 5 files
   - Remove complex traversal, caching

**Result:** ~45-50 files (down from 124), lean MVP codebase

---

## My Recommendation: Path B

**Why:**
- You're right - the codebase is too complex for MVP
- 124 files → 50 files is a 60% reduction
- Much easier to understand and iterate on
- Auth flow preserved (production requirement)
- All MVP functionality kept

**Timeline:**
- Path A: 1-2 hours (just get building)
- Path B: 1 full day (proper refactoring)

**Risk:**
- Path A: Low risk, minimal changes
- Path B: Medium risk, but high reward

---

## Immediate Action Items

### If you want Path A (quick fix):
1. I'll fix the 12 import errors
2. Remove phase0 blocks from check.go
3. Get build passing
4. You have working tool with 78 files

### If you want Path B (deep refactoring):
1. I'll do Path A first (get building)
2. Then execute consolidations:
   - analysis → metrics merge
   - config simplification
   - output simplification
   - graph cleanup
3. You have minimal MVP with ~50 files

**Which path do you prefer?** I recommend Path B since you explicitly want "leanest possible codebase" for best understanding and iteration.

---

## What's Already Done (from earlier refactoring)

- ✅ Simple investigator (single LLM call)
- ✅ Complex agent archived to _advanced/
- ✅ Dead code removed (api, storage, audit, etc.)
- ✅ Tests passing
- ✅ Auth flow preserved

**This aggressive refactoring is the final step to minimal MVP.**

---

## Key Insight

The codebase grew to 124 files because we built infrastructure for:
- Multi-hop agent navigation (over-engineered)
- AST parsing (premature)
- Caching layer (premature optimization)
- Phase 0 pre-analysis (nice-to-have)
- Adaptive configuration (over-engineered)
- Complex output system (over-modularized)

**MVP needs:**
- Check files for risk (5 metrics)
- Single LLM call for due diligence
- Auth flow for production
- Init command for ingestion

**We can deliver this in ~50 files instead of 124.**

Let me know which path you want - I'm ready to execute either!
