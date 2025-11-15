# GitHub API Optimization Analysis
## Fetcher Strategy for Rate Limits, Pagination, and Efficiency

**Date:** 2025-11-14
**Status:** Analysis Complete with Recommendations

---

## Executive Summary

Based on GitHub API documentation analysis, our current **REST API-only approach is optimal** for CodeRisk's ingestion requirements. While GraphQL could theoretically reduce API calls, **patch data is only available via REST API**, making a hybrid approach unnecessarily complex.

**Recommendation:** Optimize existing REST fetcher with better rate limit handling and smart pagination.

---

## Rate Limit Analysis

### Current Limits (Personal Access Token)

| Resource | Limit | Notes |
|----------|-------|-------|
| **Primary Rate Limit** | **5,000 requests/hour** | Our main constraint |
| **Secondary Rate Limit (concurrent)** | 100 concurrent requests | Shared across REST + GraphQL |
| **Secondary Rate Limit (per endpoint)** | 900 points/minute | Most GET requests = 1 point |
| **CPU Time Limit** | 90 seconds/minute | Total response time budget |

### Current Implementation Analysis

**Current Rate Limiter** ([fetcher.go:29](internal/github/fetcher.go#L29)):
```go
limiter := rate.NewLimiter(rate.Every(1*time.Second), 1)  // 1 req/sec
```

**Analysis:**
- ❌ **Too conservative**: 1 req/sec = 3,600 req/hour (uses only 72% of available 5,000/hour)
- ✅ **Safe for secondary limits**: Avoids 900 points/minute limit
- ❌ **Wasteful**: Leaves ~1,400 requests/hour unused

**Optimal Rate:** 1.2 req/sec (4,320 req/hour, 86% utilization, safe margin for bursts)

---

## Pagination Strategy Analysis

### Current Pagination

**Implementation** ([fetcher.go:200-243](internal/github/fetcher.go#L200-L243)):
```go
opts := &github.CommitsListOptions{
    ListOptions: github.ListOptions{
        PerPage: 100,  // Maximum allowed
    },
}

for {
    commits, resp, err := f.client.Repositories.ListCommits(ctx, owner, repo, opts)
    // ... process commits

    if resp.NextPage == 0 {
        break
    }
    opts.Page = resp.NextPage
}
```

**Analysis:**
- ✅ **Optimal page size**: 100 items (maximum allowed)
- ✅ **Correct pagination**: Uses `Link` header via `resp.NextPage`
- ✅ **No issues**: Implementation is efficient

---

## GraphQL vs REST Analysis

### REST API (Current)

**✅ Advantages:**
1. **Patch data available**: Only way to get `files[].patch` for atomization
2. **Simple pagination**: Link headers handle everything
3. **Predictable rate limits**: 1 request = 1 point
4. **Already implemented**: Working, tested, stable

**❌ Disadvantages:**
1. **More API calls**: Separate calls for commits, issues, PRs, timeline, comments
2. **Over-fetching**: Gets full objects even if we only need subset

### GraphQL API (Alternative)

**✅ Advantages:**
1. **Batching**: Could fetch issues + timeline + comments in one query
2. **Under-fetching**: Request only needed fields
3. **Potential savings**: ~30-50% fewer API calls for issues/PRs

**❌ Disadvantages:**
1. ⚠️ **NO PATCH DATA**: Cannot retrieve `files[].patch` for commits via GraphQL
2. **Complex queries**: Node limits (500k nodes), point calculations tricky
3. **Rate limit complexity**: Points calculated per query depth (hard to predict)
4. **Requires hybrid**: Would still need REST for commit patches

**Verdict:** GraphQL adds complexity without solving our core need (patch data).

---

## Critical Constraint: Patch Data

### Why Patch Data is Essential

From [POSTGRES_STAGING_VALIDATION_REPORT.md](POSTGRES_STAGING_VALIDATION_REPORT.md):
> **Pipeline 2 (Code-Block Atomization):**
> - Reads `raw_data->'files'[n]->>'patch'` to extract diffs
> - Feeds patches to LLM to identify modified functions/methods

**Example Patch Data:**
```json
{
  "files": [{
    "filename": "src/TableEditor.tsx",
    "patch": "@@ -42,7 +42,10 @@ export function updateTableEditor() {\n-  old line\n+  new line",
    "additions": 10,
    "deletions": 3
  }]
}
```

### Patch Data Availability

| API | Endpoint | Patch Data? |
|-----|----------|-------------|
| **REST** | `GET /repos/{owner}/{repo}/commits/{sha}` | ✅ **Yes** (`files[].patch`) |
| **GraphQL** | `repository { ... commits { ... } }` | ❌ **No** (only stats, not diffs) |

**Conclusion:** REST API is **required** for our use case.

---

## Current Fetcher Performance Analysis

### API Call Breakdown (90-day ingestion)

**Example: omnara-ai/omnara (hypothetical 90 days)**

| Operation | Endpoint | API Calls | Rate Limit Impact |
|-----------|----------|-----------|-------------------|
| Repository metadata | `GET /repos/{owner}/{repo}` | 1 | Negligible |
| Commit listing | `GET /repos/{owner}/{repo}/commits` | ~5 pages (50 commits/page × 100/page) | ~5 calls |
| **Commit details** | `GET /repos/{owner}/{repo}/commits/{sha}` | **500 calls** | **10%** of hourly limit |
| Issue listing | `GET /repos/{owner}/{repo}/issues` | ~10 pages (1,000 issues) | ~10 calls |
| PR listing | `GET /repos/{owner}/{repo}/pulls` | ~5 pages (500 PRs) | ~5 calls |
| Timeline events | `GET /repos/{owner}/{repo}/issues/{issue_number}/timeline` | **1,000 calls** | **20%** of hourly limit |
| Issue comments | `GET /repos/{owner}/{repo}/issues/{issue_number}/comments` | 1,000 issues × ~5 comments avg | ~1,000 calls (20%) |
| PR files | `GET /repos/{owner}/{repo}/pulls/{pr_number}/files` | 500 PRs × ~10 files avg | ~500 calls (10%) |
| **Total** | - | **~3,000+ calls** | **60% of hourly limit** |

**Time Required:**
- Current rate: 1 req/sec = 3,000 seconds = **50 minutes**
- Optimal rate: 1.2 req/sec = 2,500 seconds = **42 minutes**
- **Improvement:** 8 minutes saved (16%)

---

## Optimization Opportunities

### 1. ✅ **Increase Rate Limit Utilization** (High Impact)

**Current:** 1 req/sec (72% utilization)
**Proposed:** 1.2 req/sec (86% utilization)

**Implementation:**
```go
// Old:
limiter := rate.NewLimiter(rate.Every(1*time.Second), 1)

// New:
limiter := rate.NewLimiter(rate.Every(850*time.Millisecond), 1)  // 1.18 req/sec ≈ 4,248/hour
```

**Benefits:**
- ✅ 16% faster ingestion
- ✅ Still safe from secondary limits (900 points/minute = 15 points/sec >> 1.18 points/sec)
- ✅ Minimal code change

**Risk:** Low (well under secondary limits)

---

### 2. ✅ **Smart Checkpointing** (Already Implemented!)

**Current:** [fetcher.go:64-80](internal/github/fetcher.go#L64-L80)
```go
existingStats, err := f.checkExistingData(ctx, repoID)
if existingStats.Commits > 0 {
    log.Printf("  ℹ️  Commits already exist (%d), skipping fetch", existingStats.Commits)
    stats.Commits = existingStats.Commits
} else {
    // Fetch commits
}
```

**Analysis:**
- ✅ **Excellent**: Avoids re-fetching existing data
- ✅ **Idempotent**: Can run `crisk init` multiple times safely
- ⚠️ **Improvement opportunity**: Add incremental fetch (fetch only new commits since last run)

**Proposed Enhancement:**
```go
// Instead of all-or-nothing, fetch incrementally:
lastCommitDate, err := f.stagingDB.GetLastCommitDate(ctx, repoID)
if lastCommitDate != nil {
    opts.Since = *lastCommitDate  // Only fetch commits after this date
}
```

**Benefits:**
- ✅ Reduces API calls on subsequent runs
- ✅ Keeps repo up-to-date with minimal cost

---

### 3. ⚠️ **Batch Issue Timeline Fetching** (Medium Impact)

**Current:** [fetcher.go:653-675](internal/github/fetcher.go#L653-L675)
```go
// Fetches timeline for each issue individually (1 API call per issue)
for _, issue := range issues {
    f.fetchIssueTimeline(ctx, repoID, owner, repo, issue.Number)
}
```

**Problem:** For 1,000 issues = 1,000 API calls (20% of hourly limit)

**Potential Optimization:** GraphQL timeline batching
```graphql
query {
  repository(owner: "omnara-ai", name: "omnara") {
    issues(first: 100) {
      nodes {
        number
        timelineItems(first: 100) {
          nodes {
            __typename
            ... on CrossReferencedEvent {
              source {
                ... on PullRequest { number }
              }
            }
            ... on ClosedEvent {
              closer {
                ... on Commit { oid }
              }
            }
          }
        }
      }
    }
  }
}
```

**Benefits:**
- ✅ Could reduce 1,000 API calls → ~100 API calls (10× reduction)
- ✅ Significant time savings (15-20 minutes)

**Challenges:**
- ❌ Complex GraphQL query construction
- ❌ Point calculation complexity (nested connections)
- ❌ Still need REST for commit patches (hybrid approach)

**Recommendation:** **Low priority** - Current approach works, complexity not worth it yet.

---

### 4. ✅ **Parallel Fetching** (High Impact, Already Partially Implemented)

**Current:** Sequential fetching within each entity type

**Opportunity:** Fetch different entity types in parallel
```go
// Current (sequential):
f.FetchCommits(...)   // Wait
f.FetchIssues(...)    // Wait
f.FetchPullRequests() // Wait

// Proposed (parallel):
errGroup, ctx := errgroup.WithContext(ctx)
errGroup.Go(func() error { return f.FetchCommits(...) })
errGroup.Go(func() error { return f.FetchIssues(...) })
errGroup.Go(func() error { return f.FetchPullRequests(...) })
err := errGroup.Wait()
```

**Benefits:**
- ✅ No rate limit impact (same total API calls)
- ✅ Better CPU utilization while waiting for network I/O
- ✅ Faster wall-clock time (but not API call time)

**Challenges:**
- ⚠️ Must respect 100 concurrent request limit (secondary rate limit)
- ⚠️ Shared rate limiter needs mutex protection

**Recommendation:** **Medium priority** - Nice-to-have but not critical.

---

### 5. ✅ **Response Header Monitoring** (Already Implemented!)

**Current:** [fetcher.go:982-994](internal/github/fetcher.go#L982-L994)
```go
func (f *Fetcher) logRateLimit(resp *github.Response) {
    remaining := resp.Rate.Remaining
    limit := resp.Rate.Limit
    if remaining < 100 {
        log.Printf("  ⚠️  Rate limit low: %d/%d remaining", remaining, limit)
    }
}
```

**Analysis:**
- ✅ **Good**: Monitors rate limit headers
- ⚠️ **Enhancement opportunity**: Implement adaptive throttling

**Proposed Enhancement:**
```go
func (f *Fetcher) adjustRateLimit(resp *github.Response) {
    remaining := resp.Rate.Remaining
    resetTime := time.Unix(resp.Rate.Reset.Unix(), 0)
    timeUntilReset := time.Until(resetTime)

    if remaining < 100 {
        // Slow down significantly
        f.rateLimiter = rate.NewLimiter(rate.Every(2*time.Second), 1)
    } else if remaining < 500 {
        // Slow down moderately
        f.rateLimiter = rate.NewLimiter(rate.Every(1200*time.Millisecond), 1)
    }
    // Otherwise keep optimal rate
}
```

**Benefits:**
- ✅ Adaptive to actual rate limit consumption
- ✅ Prevents hitting rate limit ceiling
- ✅ Auto-recovers when rate limit resets

---

## Recommended Optimization Plan

### Phase 1: Quick Wins (15 minutes implementation)

1. **✅ Increase base rate limit to 1.2 req/sec**
   - Change: `rate.Every(850*time.Millisecond)`
   - Benefit: 16% faster ingestion
   - Risk: Very low

2. **✅ Add adaptive rate limiting**
   - Monitor `x-ratelimit-remaining` header
   - Slow down when `remaining < 500`
   - Benefit: Prevents rate limit exhaustion

### Phase 2: Incremental Improvements (1 hour implementation)

3. **✅ Implement incremental commit fetching**
   - Add `GetLastCommitDate()` to StagingClient
   - Use `opts.Since` for subsequent runs
   - Benefit: Massively reduces re-ingestion cost

4. **✅ Add progress reporting**
   - Show "X/Y pages processed" every 10 pages
   - Benefit: Better user experience

### Phase 3: Advanced (Future, if needed)

5. **⚠️ Hybrid GraphQL for timeline events** (optional)
   - Only if timeline fetching becomes bottleneck
   - Complex implementation
   - Benefit: ~10× reduction in timeline API calls

6. **⚠️ Parallel entity fetching** (optional)
   - Use `errgroup` for concurrent fetches
   - Benefit: Better CPU utilization

---

## Performance Projections

### Current Performance

| Metric | Value |
|--------|-------|
| Rate limit | 5,000 req/hour |
| Current utilization | 72% (3,600 req/hour at 1 req/sec) |
| 90-day ingestion time | ~50 minutes |
| Re-ingestion cost | 100% (re-fetches everything) |

### After Phase 1 Optimizations

| Metric | Value | Change |
|--------|-------|--------|
| Rate limit | 5,000 req/hour | - |
| Optimized utilization | 86% (4,248 req/hour at 1.18 req/sec) | +14% |
| 90-day ingestion time | **~42 minutes** | **-16%** |
| Re-ingestion cost | 100% | - |

### After Phase 2 Optimizations

| Metric | Value | Change |
|--------|-------|--------|
| 90-day ingestion time | ~42 minutes | - |
| **Re-ingestion cost** | **~5-10%** (only new commits) | **-90%** |
| Daily update time | **~2-3 minutes** | **-94%** |

---

## Critical Insights

### 1. REST API is Required

**Reason:** Patch data (`files[].patch`) is ONLY available via REST API.
**Implication:** Any GraphQL optimization must be hybrid (GraphQL for metadata, REST for patches).
**Decision:** Stick with REST-only for simplicity unless timeline fetching becomes unbearable.

### 2. Current Implementation is Good

**Strengths:**
- ✅ Correct pagination (uses Link headers)
- ✅ Optimal page sizes (100 items)
- ✅ Smart checkpointing (avoids re-fetch)
- ✅ Rate limit monitoring

**Minor Issues:**
- ⚠️ Conservative rate limit (72% utilization)
- ⚠️ No incremental fetching (re-fetches on every run)

### 3. Timeline Events are the Bottleneck

**Problem:** 1,000 issues × 1 API call each = 20% of rate limit

**Solutions:**
1. ✅ **Short-term:** Accept the cost (timeline events are critical for 100% confidence edges)
2. ⚠️ **Long-term:** Consider GraphQL hybrid if >10K issues (but adds complexity)

---

## Implementation Priority

| Priority | Optimization | Effort | Impact | Implement? |
|----------|-------------|--------|--------|-----------|
| **1** | Increase rate to 1.2 req/sec | 5 min | 16% faster | ✅ **Yes** |
| **2** | Adaptive rate limiting | 10 min | Prevents exhaustion | ✅ **Yes** |
| **3** | Incremental commit fetching | 1 hour | 90% re-ingestion savings | ✅ **Yes** |
| **4** | Progress reporting | 15 min | Better UX | ✅ **Yes** |
| 5 | Parallel fetching | 2 hours | Marginal (I/O bound) | ⚠️ **Maybe** |
| 6 | GraphQL timeline batching | 4 hours | 10× timeline reduction | ⚠️ **Future** |

---

## Conclusion

**Current fetcher is well-designed** but leaves room for easy optimization:

1. **Immediate wins:** Increase rate to 1.2 req/sec, add adaptive throttling (15 min implementation, 16% faster)
2. **High-value add:** Incremental fetching (1 hour implementation, 90% re-ingestion savings)
3. **GraphQL not worth it:** Patch data requirement makes hybrid approach too complex for marginal gains

**Recommendation:** Implement Phase 1 + Phase 2 optimizations (~1.25 hours total), defer GraphQL indefinitely.

---

**Analysis Date:** 2025-11-14
**Analyzed By:** GitHub API Optimization Tool
**References:**
- [github_rate_limits_rest_api.md](../api_docs/github/github_rate_limits_rest_api.md)
- [github_pagination_rest_api.md](../api_docs/github/github_pagination_rest_api.md)
- [github_commit_rest_api.md](../api_docs/github/github_commit_rest_api.md)
