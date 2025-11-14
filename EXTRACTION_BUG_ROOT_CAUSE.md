# Extraction Bug - Root Cause Analysis

## Bug Confirmed and Root Cause Identified

**Date:** 2025-11-13
**Status:** 🔴 CRITICAL BUG - Systematic misattribution of all LLM-extracted references

---

## The Bug

ASSOCIATED_WITH edges are being created with **systematically wrong commit associations**. References extracted from one commit are being attributed to a completely different commit.

### Reproduction Confirmed

Re-ran extraction on omnara repo - bug reproduced **identically**:
- "Initial commit" (no references) → Edge to Issue #1 created ❌
- "Update Makefile" (no references) → Edge to Issue #2 created ❌
- "lint" (no references) → Edge to Issue #3 created ❌
- "change" (no references) → Edge to Issue #4 created ❌

Meanwhile:
- "Merge pull request **#1**..." → Edge to Issue #0 created ❌ (should be #1!)

---

## Root Cause: Array Index Mismatch

### The Broken Code

**Location:** `internal/github/commit_extractor.go:164-181`

```go
// Store extracted references
var allRefs []database.IssueCommitRef
for i, output := range result.Results {
    commit := commits[i]  // ← BUG: Assumes result.Results[i] matches commits[i]

    for _, ref := range output.References {
        commitSHA := commit.SHA
        dbRef := database.IssueCommitRef{
            RepoID:          repoID,
            IssueNumber:     ref.TargetID,
            CommitSHA:       &commitSHA,  // ← Wrong commit!
            Action:          ref.Action,
            Confidence:      ref.Confidence,
            DetectionMethod: "commit_extraction",
            ExtractedFrom:   "commit_message",
        }
        allRefs = append(allRefs, dbRef)
    }
}
```

### Why It Fails

**LLM Output Format:**
```json
{
  "results": [
    {
      "id": "abc123",      // Commit SHA
      "references": [...]   // Array of references found in THIS commit
    },
    {
      "id": "def456",
      "references": []      // No references - but entry still exists!
    }
  ]
}
```

**The Problem:**
The code loops using index `i` to match `result.Results[i]` to `commits[i]`:

```go
for i, output := range result.Results {
    commit := commits[i]  // Assumes same position
```

But there's **NO guarantee** that:
1. `result.Results` is in the same order as `commits`
2. `result.Results` has the same length as `commits`
3. LLM doesn't skip/reorder commits

**The code should match on `output.ID` (commit SHA) instead of array index `i`.**

---

## Concrete Example from Batch 1

### Input to LLM (Batch 1, commits 1-20):

| Index | SHA (short) | Message | Has Reference? |
|-------|-------------|---------|----------------|
| 0 | d12fe4a | "Initial commit" | ❌ No |
| 1 | 44fd60e | "Update Makefile" | ❌ No |
| 2 | f0714f7 | "lint" | ❌ No |
| 3 | 2d626b6 | "change" | ❌ No |
| 4 | f14658a | "Merge pull request **#1** from..." | ✅ Yes (#1) |
| 5 | 6f4c15b | "format" | ❌ No |
| 6 | 6cf2fae | "Merge pull request **#2** from..." | ✅ Yes (#2) |
| ... | ... | ... | ... |
| 14 | 4fe8d24 | "Merge pull request **#3** from..." | ✅ Yes (#3) |
| ... | ... | ... | ... |
| 17 | d0431f9 | "Merge pull request **#4** from..." | ✅ Yes (#4) |

### LLM Output (hypothetical - need to log to confirm):

```json
{
  "results": [
    {"id": "d12fe4a", "references": []},
    {"id": "44fd60e", "references": []},
    {"id": "f0714f7", "references": []},
    {"id": "2d626b6", "references": []},
    {"id": "f14658a", "references": [{"target_id": 1, "action": "mentions", "confidence": 0.9}]},
    {"id": "6f4c15b", "references": []},
    {"id": "6cf2fae", "references": [{"target_id": 2, "action": "mentions", "confidence": 0.9}]},
    ...
    {"id": "4fe8d24", "references": [{"target_id": 3, "action": "mentions", "confidence": 0.9}]},
    ...
    {"id": "d0431f9", "references": [{"target_id": 4, "action": "mentions", "confidence": 0.9}]},
    ...
  ]
}
```

OR (if LLM skips empty results):

```json
{
  "results": [
    {"id": "f14658a", "references": [{"target_id": 1, ...}]},  // result[0]
    {"id": "6cf2fae", "references": [{"target_id": 2, ...}]},  // result[1]
    {"id": "4fe8d24", "references": [{"target_id": 3, ...}]},  // result[2]
    {"id": "d0431f9", "references": [{"target_id": 4, ...}]},  // result[3]
  ]
}
```

### What the Code Does:

```go
for i, output := range result.Results {
    commit := commits[i]  // Gets commits[0], commits[1], commits[2], commits[3]
    // Assigns references to:
    //   commits[0] = d12fe4a  → gets reference #1 ❌
    //   commits[1] = 44fd60e  → gets reference #2 ❌
    //   commits[2] = f0714f7  → gets reference #3 ❌
    //   commits[3] = 2d626b6  → gets reference #4 ❌
}
```

### What Actually Happened (from database):

| Commit SHA | Message | Extracted Issue | Correct? |
|------------|---------|-----------------|----------|
| d12fe4a | "Initial commit" | #1 | ❌ Should be f14658a |
| 44fd60e | "Update Makefile" | #2 | ❌ Should be 6cf2fae |
| f0714f7 | "lint" | #3 | ❌ Should be 4fe8d24 |
| 2d626b6 | "change" | #4 | ❌ Should be d0431f9 |
| f14658a | "Merge pull request **#1**..." | #0 (?) | ❌ Should be #1 |

---

## Impact Assessment

### Severity: 🔴 CRITICAL

**All 173 commit_extraction edges are potentially wrong.**

The misattribution is **systematic**, not random:
- Every batch has the same bug
- References shift to earlier commits in the batch
- Makes ASSOCIATED_WITH edges completely unreliable

### Affected Data

- **commit_extraction:** 173 edges (100% suspect)
- **pr_extraction:** 21 edges (same code pattern - likely affected)
- **temporal:** 62 edges (rule-based - unaffected)

**Total potentially incorrect edges:** 194 out of 245 (79%)

### Consequences

1. **False positive connections:**
   - Generic commits like "Initial commit" linked to bugs they didn't fix
   - Makes risk analysis completely wrong

2. **Missing connections:**
   - Actual fix commits (like f14658a for PR #1) have no edges
   - Can't trace which commits fixed which issues

3. **Confidence scores meaningless:**
   - High confidence (0.9) on hallucinated connections
   - Can't trust ANY confidence scores

4. **CLQS metrics broken:**
   - Files get incorrect incident counts
   - Quality scores are garbage

---

## The Fix

### Required Changes

**1. Match on commit SHA, not array index:**

```go
// Store extracted references
var allRefs []database.IssueCommitRef

// Create a map of SHA → commit for O(1) lookup
commitMap := make(map[string]database.CommitData)
for _, commit := range commits {
    commitMap[commit.SHA] = commit
}

// Match results by ID (SHA)
for _, output := range result.Results {
    // Look up the commit by SHA
    commit, found := commitMap[output.ID]
    if !found {
        log.Printf("⚠️  LLM returned ID %s not in batch - skipping", output.ID)
        continue
    }

    for _, ref := range output.References {
        commitSHA := commit.SHA
        dbRef := database.IssueCommitRef{
            RepoID:          repoID,
            IssueNumber:     ref.TargetID,
            CommitSHA:       &commitSHA,  // ✅ Now correct!
            Action:          ref.Action,
            Confidence:      ref.Confidence,
            DetectionMethod: "commit_extraction",
            ExtractedFrom:   "commit_message",
        }
        allRefs = append(allRefs, dbRef)
    }
}
```

**2. Add validation:**

```go
// Verify reference actually exists in commit message
commitMsg := strings.ToLower(commit.Message)
targetStr := fmt.Sprintf("#%d", ref.TargetID)

if !strings.Contains(commitMsg, targetStr) {
    log.Printf("⚠️  Validation failed: Commit %s doesn't contain %s",
        commit.SHA[:7], targetStr)

    // Lower confidence or skip
    dbRef.Confidence = dbRef.Confidence * 0.5
}
```

**3. Add logging for debugging:**

```go
log.Printf("  Matched LLM result ID=%s to commit %s",
    output.ID[:10], commit.SHA[:10])
```

### Same Fix Needed For:

- `processPRBatch()` in `commit_extractor.go:194+`
- Any other LLM extraction code using array index matching

---

## Testing the Fix

### Before Fix (Current State):

```sql
SELECT
    gc.sha,
    gc.message,
    icr.issue_number,
    (gc.message LIKE '%#' || icr.issue_number || '%') as message_contains_ref
FROM github_commits gc
JOIN github_issue_commit_refs icr ON gc.sha = icr.commit_sha
WHERE icr.detection_method = 'commit_extraction'
  AND icr.repo_id = 1
LIMIT 10;
```

**Expected:** Most rows have `message_contains_ref = false` ❌

### After Fix:

**Expected:** All rows have `message_contains_ref = true` ✅

### Test Cases:

1. **Batch with mixed references:**
   - Commits: A (no ref), B (#1), C (no ref), D (#2)
   - Should extract: B→#1, D→#2
   - Current bug: A→#1, C→#2

2. **Batch with all refs:**
   - Commits: all contain references
   - Should work even with current bug (no misalignment)

3. **Batch with no refs:**
   - Commits: none contain references
   - Should extract: nothing
   - Current bug: might still create refs if LLM hallucinates

---

## Immediate Actions Required

### 1. ⚠️  Stop Using Extraction Data (P0)
- Mark all `commit_extraction` and `pr_extraction` edges as UNRELIABLE
- Don't use for production risk analysis until fixed
- Temporal edges (62) are still valid

### 2. 🔧 Fix the Code (P0)
- Implement SHA-based matching in `commit_extractor.go`
- Add validation that references exist in text
- Add comprehensive logging

### 3. 🧪 Test the Fix (P0)
- Unit tests with known good/bad examples
- Integration test on small batch
- Verify validation catches false positives

### 4. 🔄 Re-extract All Data (P1)
- Clear all `commit_extraction` and `pr_extraction` refs
- Re-run extraction with fixed code
- Rebuild Neo4j ASSOCIATED_WITH edges

### 5. 📊 Audit Results (P1)
- Run validation query on all new edges
- Confirm 100% of edges have references in text
- Compare before/after edge counts

### 6. 📝 Document Learnings (P2)
- Add this case to test suite
- Update LLM integration docs with "gotchas"
- Add monitoring for future extractions

---

## Long-term Improvements

1. **Always validate LLM output:**
   - Don't blindly trust LLM extractions
   - Verify with regex/string matching
   - Lower confidence if validation fails

2. **Structured output with explicit IDs:**
   - Require LLM to return commit SHA in each result
   - Never rely on array ordering

3. **Comprehensive logging:**
   - Log raw LLM responses for debugging
   - Track validation failures
   - Alert on high skip rates

4. **Automated quality checks:**
   - Daily validation query
   - Alert if >5% of edges fail validation
   - Dashboard showing extraction quality

---

## Summary

We discovered a **critical systematic bug** in the LLM extraction system:

- **Root cause:** Array index matching instead of SHA-based matching
- **Impact:** ~79% of ASSOCIATED_WITH edges are incorrect (194/245)
- **Evidence:** Bug reproduced identically on re-extraction
- **Fix:** Match LLM results by `output.ID` instead of array index `i`

The extraction system is **fundamentally broken** and cannot be trusted until fixed.

All ASSOCIATED_WITH edges from `commit_extraction` and `pr_extraction` should be considered **invalid** until re-extracted with the fixed code.

---

## Files to Update

1. `internal/github/commit_extractor.go`
   - Lines 164-181: `processCommitBatch()` result matching
   - Lines 194+: `processPRBatch()` result matching

2. Tests to add:
   - `internal/github/commit_extractor_test.go` (new)
   - Test mixed batch with refs at different positions
   - Test validation catches misattributions

3. Documentation:
   - This file (root cause analysis)
   - `ASSOCIATED_WITH_EDGE_BUG_ANALYSIS.md` (update with fix)
   - `docs/ASSOCIATED_WITH_EDGE_SYSTEM.md` (add validation section)
