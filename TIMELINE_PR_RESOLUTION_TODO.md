# Timeline PR Resolution Enhancement - TODO

## Problem Statement

GitHub's web UI displays "closed this as completed in #PR" when an issue is closed by a merged pull request. However, this information is NOT directly available in the GitHub Timeline API as a structured `cross-referenced` event.

Instead, GitHub provides only:
1. A `closed` event showing when the issue closed
2. A `referenced` event with the **merge commit SHA** (not the PR number)

To replicate GitHub's UI behavior and get high-confidence Issue-PR links for FIXED_BY classification, we need to:
1. Detect `referenced` timeline events with commit IDs
2. Call GitHub's `/repos/{owner}/{repo}/commits/{sha}/pulls` API to resolve commits to their source PRs
3. Store the PR information (number, title, body, state, merged_at) instead of just the commit SHA

## Example Case Study

**Issue**: https://github.com/supabase/supabase/issues/39456
**PR**: https://github.com/supabase/supabase/pull/39457

**Timeline of Events:**
- 2025-11-04T17:16:43Z: PR #39457 merged (merge commit: `4f4497e`)
- 2025-11-04T17:16:44Z: Issue #39456 closed (auto-close, 1 sec later)
- 2025-11-04T17:16:45Z: `referenced` event added with `commit_id: "4f4497e"`

**GitHub's UI Rendering:**
Shows "stylessh closed this as completed in #39457" by:
1. Detecting the `referenced` event with merge commit
2. Calling `/commits/4f4497e/pulls` → returns PR #39457
3. Rendering as "closed in #39457" instead of "closed + commit 4f4497e"

**What We Get from Timeline API:**
```json
{
  "event": "referenced",
  "commit_id": "4f4497e3555a7ab4df4574788d15131943ce7d30",
  "created_at": "2025-11-04T17:16:45Z"
}
```

**What We Need:**
```json
{
  "event": "referenced",
  "source_type": "pull_request",
  "source_number": 39457,
  "source_title": "Rename SvelteKit setup guide's environment variables name in docs",
  "source_body": "Fix #39456",
  "source_state": "closed",
  "source_merged_at": "2025-11-04T17:16:43Z"
}
```

## Implementation Status

### ✅ Code Changes Made

**File**: [internal/github/fetcher.go](internal/github/fetcher.go)

- **Added `PRInfo` struct** (line ~710): Holds PR metadata for timeline events
- **Added `resolveCommitToPR()` method** (line ~718):
  - Calls `/repos/{owner}/{repo}/commits/{sha}/pulls` API
  - Returns PR number, title, body, state, merged_at
  - Rate-limited to avoid API abuse
- **Modified `storeTimelineEvent()` signature** (line ~773):
  - Added `owner` and `repo` parameters
  - Added logic to detect `referenced` events with `commit_id`
  - Calls `resolveCommitToPR()` to convert commit refs to PR refs
- **Updated call site** (line ~680): Pass owner/repo to `storeTimelineEvent()`

**File**: [internal/linking/phase0_preprocessing.go](internal/linking/phase0_preprocessing.go)

- **Updated query** (line ~74): Changed from `event_type = 'cross-referenced'` to `(event_type = 'cross-referenced' OR event_type = 'referenced')` to include resolved PR references

### ⚠️ Not Yet Tested

**Why**: Testing requires fetching GitHub data for a large repo (supabase/supabase) which hits API pagination limits and rate limits.

**Test Case**: Issue #39456 from supabase/supabase should show:
- Timeline event with `source_type = 'pull_request'` and `source_number = 39457`
- Detection method: `github_timeline_verified`
- Edge type: `FIXED_BY` (meets all 4 criteria for Multi-Signal Ground Truth)

## Impact on Linking System

### Current State (Without This Enhancement)
- **Zero** `github_timeline_verified` links in omnara-ai/omnara repo
- All 27 existing links use `deep_link_finder` detection method
- **Zero** `FIXED_BY` edges (all are `ASSOCIATED_WITH`)
- Reason: Multi-Signal classification requires high-quality detection methods

### Expected State (With This Enhancement)
- Issues closed by PR merges will have `github_timeline_verified` links
- These links have base confidence = 0.95 (highest tier)
- Meet Criterion 1 of Multi-Signal Ground Truth (high-quality detection)
- Likely to be classified as `FIXED_BY` edges (if other criteria met)
- Enables accurate risk assessment and CLQS calculations

## Files Modified

1. `internal/github/fetcher.go` - Timeline event processing
2. `internal/linking/phase0_preprocessing.go` - Timeline link extraction query

## Files NOT Modified (May Need Updates)

1. `internal/database/staging.go` - `github_issue_timeline` schema already supports all needed fields
2. `scripts/schema/postgresql_staging.sql` - Schema already correct

## Testing TODO

### Option 1: Use Smaller Test Repo
Find a small open-source repo where:
- Issues are regularly closed by PR merges
- Has fewer than 1000 total issues (avoids pagination limits)
- Example: A small bug-tracking repo or a well-maintained library

### Option 2: Mock Testing
Create unit tests that:
- Mock the GitHub `/commits/{sha}/pulls` API response
- Test `resolveCommitToPR()` function directly
- Verify PR metadata is correctly extracted

### Option 3: Defer Testing
- Keep the enhancement uncommitted but saved
- Focus on `crisk check` implementation first
- Return to this when we have better test infrastructure

## Recommended Next Steps

1. **Option 3 (Defer)** - Revert changes, document the approach, move forward with `crisk check`
2. Create tracking issue: "Implement Timeline PR Resolution for High-Confidence FIXED_BY Links"
3. Return to this after `crisk check` is working and we have test infrastructure

## API Rate Limit Considerations

The `/commits/{sha}/pulls` endpoint adds one API call per `referenced` event with a commit. For repos with many issues:
- Each closed issue typically has 1-2 `referenced` events
- This could add 100-200 API calls for 100 closed issues
- With rate limit of 5000/hour, this is manageable
- Rate limiter (1 req/sec) prevents abuse

## Related Documentation

- [GitHub Timeline API](https://docs.github.com/en/rest/issues/timeline)
- [GitHub Commits API - List PRs for Commit](https://docs.github.com/en/rest/commits/commits#list-pull-requests-associated-with-a-commit)
- `ISSUE_PR_LINKING_README.md` - Multi-Signal Ground Truth Classification
- `internal/clqs/component2.go` - Confidence Quality calculation (benefits from this enhancement)
