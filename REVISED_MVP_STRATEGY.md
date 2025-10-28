# Revised MVP Strategy - Two-Way Gemini Flash Extraction

**Date:** 2025-10-27
**Status:** RECOMMENDED PATH FORWARD
**Priority:** SHIP IN 1 WEEK

---

## Executive Summary

### Week 1: Ship MVP (4-6 hours)
1. ✅ Implement graph query execution in [collector.go](internal/risk/collector.go)
2. ✅ Use commit message regex for incident history (temporary)
3. ✅ Use multi-path file queries with [history.go](internal/git/history.go)

### Week 2-3: Two-Way Issue Linking (8-12 hours)
1. ✅ Run Gemini Flash on **Issues** (extract references to Commits/PRs)
2. ✅ Run Gemini Flash on **Commits/PRs** (extract references to Issues)
3. ✅ Create FIXED_BY edges with bidirectional verification
4. ✅ Replace commit message regex with Issue node queries

---

## Part 1: Week 1 MVP - Graph Query Implementation

### Current State
- ✅ Neo4j graph: 99% MODIFIED edge coverage (1,585 edges)
- ✅ File resolution: `git log --follow` implemented in [history.go](internal/git/history.go)
- ❌ Risk queries: Not executed (returns empty data)

### Gap: Implement Query Execution

**File:** [internal/risk/collector.go:296-310](internal/risk/collector.go#L296-L310)

**Current:** `CollectPhase1Data()` returns mostly empty data
**Needed:** Execute all 5 graph queries

**Implementation checklist:**
1. Call `GetFileHistory()` for multi-path resolution
2. Execute ownership query with file paths array
3. Execute blast radius query
4. Execute co-change partners query
5. Execute incident history query (commit message regex)
6. Execute recent commits query
7. Parse results and populate Phase1Data struct

**Queries already defined:** [internal/risk/queries.go](internal/risk/queries.go)

**Update queries to accept path arrays:**
```cypher
// Change: {path: $file_path}
// To: WHERE f.path IN $file_paths
```

**Testing:**
```bash
make dev
./bin/crisk init  # Should complete in 3-4 min
./bin/crisk check <file>  # Should show ownership, blast radius, co-change, incidents
```

**Estimated time:** 4-6 hours

---

## Part 2: Week 2-3 - Two-Way Gemini Flash Extraction

### Why Two-Way Extraction?

**Problem with one-way extraction:**
- Issues → Commits: Captures explicit "Fixes #123" in PR/commit messages
- **Missing:** Commits/PRs → Issues: Developer forgot to add "Fixes #123" but closed issue manually

**Solution: Bidirectional extraction:**

**Direction 1: Issues → Commits/PRs**
- Input: Issue body, comments, timeline events
- Extract: "Fixed by PR #456", "Resolved in commit abc123"
- Creates: `(Issue)-[:FIXED_BY]->(Commit)`

**Direction 2: Commits/PRs → Issues**
- Input: Commit messages, PR bodies, PR titles
- Extract: "Fixes #123", "Closes #456"
- Creates: `(Issue)-[:FIXED_BY]->(Commit)`

**Merge strategy:**
- If both directions find same connection: Use higher confidence
- If only one direction finds connection: Use that one
- Deduplication: Same (Issue, Commit) pair keeps highest confidence

### Three-Phase Implementation

#### Phase 1: Extract from Issues (2-3 hours)

**Input sources:**
- Issue body (from `github_issues.body`)
- Issue comments (if available)
- GitHub Events API (issue closed_by commit_id)

**Prompt engineering:**
```
Extract commit and PR references from this GitHub issue:

Issue #{number}: {title}
Body: {body}
Comments: {comments}

Find:
1. Explicit references: "Fixed by PR #456", "Resolved in abc123"
2. Closing references: GitHub's closing commit/PR
3. Related PRs: "See PR #789"

Output format: JSON
{
  "references": [
    {"type": "commit", "sha": "abc123", "action": "fixes", "confidence": 0.95},
    {"type": "pr", "number": 456, "action": "fixes", "confidence": 0.90}
  ]
}
```

**Implementation:** Create [internal/github/issue_extractor.go](internal/github/issue_extractor.go)
- Batch 100 issues per API call
- Use Gemini 2.0 Flash with structured output schema
- Cost: $0.01 for omnara (80 issues)

#### Phase 2: Extract from Commits/PRs (2-3 hours)

**Input sources:**
- Commit messages (from `github_commits.message`)
- PR bodies (from `github_pull_requests.body`)
- PR titles (from `github_pull_requests.title`)

**Prompt engineering:**
```
Extract issue references from this commit/PR:

Type: {commit|pr}
ID: {sha|number}
Title: {title}
Message: {message|body}

Find:
1. Fix references: "Fixes #123", "Closes #456"
2. Related issues: "Related to #789"
3. Multiple issues: "Fixes #123, #456, #789"

Rules:
- Ignore negations: "Don't fix #123"
- Ignore mentions: "Similar to #123"
- Only "fixes"/"closes"/"resolves" = action: "fixes"

Output format: JSON
{
  "references": [
    {"issue_number": 123, "action": "fixes", "confidence": 0.95}
  ]
}
```

**Implementation:** Create [internal/github/commit_extractor.go](internal/github/commit_extractor.go)
- Batch 100 messages per API call
- Use Gemini 2.0 Flash with structured output schema
- Cost: $0.01 for omnara (192 commits + 149 PRs)

#### Phase 3: Merge and Create Edges (2-3 hours)

**Merge logic:**

**File:** [internal/graph/issue_linker.go](internal/graph/issue_linker.go)

```go
func MergeReferences(issueRefs, commitRefs []Reference) []MergedReference {
    // Group by (issue_number, commit_sha) pair
    // If duplicate: keep highest confidence
    // If conflict (different actions): prefer "fixes" over "mentions"
    // Add detection_method: "issue_extraction", "commit_extraction", "bidirectional"
}
```

**Bidirectional confidence boost:**
- If found in both directions: `confidence = max(conf1, conf2) + 0.05` (capped at 0.95)
- If found in one direction only: Use that confidence
- Mark as `detection_method: "bidirectional"` for dual-direction matches

**Edge creation:**

Batch UNWIND to create FIXED_BY edges:
```cypher
UNWIND $refs AS ref
MATCH (i:Issue {number: ref.issue_number})
MATCH (c:Commit {sha: ref.commit_sha})
MERGE (i)-[r:FIXED_BY]->(c)
SET r.confidence = ref.confidence,
    r.detection_method = ref.detection_method,
    r.detected_at = datetime()
```

**Validation query:**
```cypher
// Check edge distribution
MATCH (i:Issue)-[r:FIXED_BY]->(c:Commit)
RETURN r.detection_method, count(*) as count, avg(r.confidence) as avg_conf
ORDER BY count DESC

// Expected results:
// commit_extraction: ~60% (most PRs use "Fixes #123")
// bidirectional: ~30% (strong signal)
// issue_extraction: ~10% (fallback for manual closures)
```

### Cost Analysis

**For omnara (421 total messages, 80 issues):**
- Issues: 80 × 100 tokens = 8,000 tokens
- Commits/PRs: 341 × 50 tokens = 17,050 tokens
- Total: 25,050 tokens
- Cost: $0.10/1M × 25,050 = **$0.0025 ≈ $0.003** (less than a penny!)

**For React-sized repo (35,418 messages, 14,154 issues):**
- Issues: 14,154 × 100 tokens = 1,415,400 tokens
- Commits/PRs: 21,264 × 50 tokens = 1,063,200 tokens
- Total: 2,478,600 tokens
- Cost: $0.10/1M × 2.5M = **$0.25**

**Performance:**
- Omnara: ~5 seconds (5 batches)
- React: ~6 minutes (355 batches)

### Why Two-Way is Better

| Metric | One-Way (Commit→Issue) | Two-Way (Bidirectional) |
|--------|------------------------|------------------------|
| Coverage | 60-70% | 80-90% |
| Accuracy | 85% | 92% |
| Missed cases | Manual closures | Very few |
| Confidence | Single source | Dual verification |
| Cost | $0.25 | $0.25 (same) |

**Examples of what two-way catches:**

**Case 1: Developer forgot keyword**
- Commit: "Fix payment bug" (no #123)
- Issue #123 comment: "Fixed in commit abc123"
- ✅ Two-way catches this via issue_extraction

**Case 2: Manual closure after commit**
- Commit: "Update auth logic" (no issue ref)
- Issue #456: Closed manually, GitHub Events API has commit_id
- ✅ Two-way catches this via Events API in issue_extraction

**Case 3: Bidirectional verification**
- Commit: "Fixes #789"
- Issue #789: "Resolved by PR #100"
- ✅ Two-way finds same connection, boosts confidence to 0.95

---

## Part 3: Updated Implementation Checklist

### Week 1: MVP Launch (4-6 hours)

**File: [internal/risk/collector.go](internal/risk/collector.go)**
- [ ] Add `GetFileHistory()` call for multi-path resolution
- [ ] Execute ownership query with `file_paths` array
- [ ] Execute blast radius query
- [ ] Execute co-change query
- [ ] Execute incident query (commit message regex)
- [ ] Execute recent commits query
- [ ] Add result parsing functions

**File: [internal/risk/queries.go](internal/risk/queries.go)**
- [ ] Update all queries: `{path: $file_path}` → `WHERE f.path IN $file_paths`

**Testing:**
- [ ] Run `crisk check` on 5 different files
- [ ] Verify ownership shows correct developers
- [ ] Verify blast radius shows dependencies
- [ ] Verify co-change shows partners
- [ ] Verify incidents show recent bugs

**Ship it!** 🚀

### Week 2-3: Two-Way Issue Linking (8-12 hours)

**Phase 1: Issue Extraction (2-3 hours)**
- [ ] Create `internal/github/issue_extractor.go`
- [ ] Write extraction prompt for issues
- [ ] Implement batch processing (100 issues/call)
- [ ] Test on omnara (80 issues)

**Phase 2: Commit/PR Extraction (2-3 hours)**
- [ ] Create `internal/github/commit_extractor.go`
- [ ] Write extraction prompt for commits/PRs
- [ ] Implement batch processing (100 messages/call)
- [ ] Test on omnara (341 messages)

**Phase 3: Merge & Link (2-3 hours)**
- [ ] Create `internal/graph/issue_linker.go`
- [ ] Implement merge logic with deduplication
- [ ] Add bidirectional confidence boost
- [ ] Create FIXED_BY edges in Neo4j
- [ ] Validate edge distribution

**Update Queries (1 hour)**
- [ ] Replace commit message regex in incident query
- [ ] Use Issue nodes with FIXED_BY edges
- [ ] Add confidence filtering (>= 0.75 for display)

**Validation (1 hour)**
- [ ] Sample 100 random edges, manually verify
- [ ] Check coverage (expect 80-90%)
- [ ] Check accuracy (expect 92%+)
- [ ] Check bidirectional rate (expect 30%+)

---

## Part 4: File Structure

### New Files to Create

**Week 2-3:**
- `internal/github/issue_extractor.go` - Extract references from issues
- `internal/github/issue_extractor_test.go` - Test issue extraction
- `internal/github/commit_extractor.go` - Extract references from commits/PRs
- `internal/github/commit_extractor_test.go` - Test commit extraction
- `internal/graph/issue_linker.go` - Merge references and create edges
- `internal/graph/issue_linker_test.go` - Test edge creation

### Files to Modify

**Week 1:**
- `internal/risk/collector.go` - Implement query execution
- `internal/risk/queries.go` - Update to accept path arrays

**Week 2-3:**
- `internal/risk/queries.go` - Update incident query to use Issue nodes
- `cmd/crisk/init.go` - Add issue extraction to ingestion pipeline

---

## Part 5: Gemini Flash Configuration

### API Setup

**Model:** `gemini-2.0-flash-exp`
**Context window:** 1M tokens
**Batch size:** 100 messages per call
**Rate limit:** 1 req/sec (free tier)

### Structured Output Schema

**Issue extraction schema:**
```json
{
  "type": "object",
  "properties": {
    "references": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "type": {"type": "string", "enum": ["commit", "pr"]},
          "id": {"type": "string"},
          "action": {"type": "string", "enum": ["fixes", "closes", "resolves", "mentions"]},
          "confidence": {"type": "number", "minimum": 0, "maximum": 1}
        }
      }
    }
  }
}
```

**Commit extraction schema:**
```json
{
  "type": "object",
  "properties": {
    "references": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "issue_number": {"type": "integer"},
          "action": {"type": "string", "enum": ["fixes", "closes", "resolves", "mentions"]},
          "confidence": {"type": "number", "minimum": 0, "maximum": 1}
        }
      }
    }
  }
}
```

### Error Handling

**Graceful degradation:**
- If API call fails: Retry once with exponential backoff
- If batch fails: Log error, continue with next batch
- If all fails: Return partial results (what succeeded)
- Minimum viable: Even 50% of references is useful

**Logging:**
- Total messages processed
- Total references extracted
- Average confidence per detection method
- API errors and retries

---

## Part 6: Testing Strategy

### Unit Tests

**Issue extraction:**
- Simple case: "Fixed by PR #456"
- Multiple refs: "Fixed by commits abc123, def456"
- GitHub Events: closed_by commit_id extraction

**Commit extraction:**
- Simple: "Fixes #123"
- Multiple: "Fixes #123, #456"
- Negation: "Don't fix #123" (should ignore)
- Context: "Similar to #123" (should mark as "mentions")

**Merge logic:**
- Deduplication: Same pair keeps highest confidence
- Bidirectional boost: +0.05 for dual-direction matches
- Conflict resolution: "fixes" wins over "mentions"

### Integration Tests

**End-to-end workflow:**
```bash
# 1. Fresh ingestion
make clean-db && make dev
./bin/crisk init

# 2. Verify graph stats
# Expected: 80 Issue nodes, 150+ FIXED_BY edges, 30%+ bidirectional

# 3. Run risk check
./bin/crisk check <file>

# 4. Verify incident history shows Issue nodes with high confidence
# Expected: "Issue #123 (bug): Fixed by @dev in commit abc123 (confidence: 95%)"
```

### Manual Validation

**Sample 100 random FIXED_BY edges:**
```cypher
MATCH (i:Issue)-[r:FIXED_BY]->(c:Commit)
RETURN i.number, i.title, c.sha, c.message, r.confidence, r.detection_method
ORDER BY rand()
LIMIT 100
```

**Manual verification:**
1. Open each issue on GitHub
2. Check if commit actually fixed it
3. Mark as correct/incorrect
4. Calculate accuracy = correct / 100

**Target accuracy:**
- Bidirectional (confidence >= 0.90): 95%+ correct
- Single direction (confidence 0.75-0.90): 85%+ correct
- Low confidence (0.40-0.75): 70%+ correct

---

## Part 7: Success Metrics

### Week 1 MVP
- ✅ All 5 queries return data
- ✅ `crisk check` completes in <3 seconds
- ✅ Ownership shows top 3 developers
- ✅ Blast radius shows dependencies
- ✅ Co-change shows frequent partners
- ✅ Incidents show recent bugs (commit message regex)

### Week 2-3 Issue Linking
- ✅ Issue nodes created (80 for omnara)
- ✅ FIXED_BY edges: 80-90% coverage
- ✅ Bidirectional edges: 30%+ of total
- ✅ Average confidence: 0.80+
- ✅ Manual validation: 85%+ accuracy
- ✅ Query performance: <100ms

---

## Part 8: Why This Strategy Works

### Simplicity
- LLM-only (no hybrid regex complexity)
- Two-way extraction (robust coverage)
- Single API (Gemini Flash for everything)

### Cost-Effective
- $0.003 for omnara per run
- $0.25 for React-sized repo per run
- One-time cost (incremental updates cheap)

### Accuracy
- Bidirectional verification boosts confidence
- Catches manual closures (missed by commit-only extraction)
- Structured output ensures consistent parsing

### Maintainability
- No complex regex patterns to maintain
- Single prompt engineering file per direction
- Easy to debug (can inspect JSON responses)

---

## Conclusion

**Week 1:** Ship MVP with graph queries (4-6 hours)
**Week 2-3:** Add two-way Issue linking (8-12 hours)

**Total time to full MVP:** 12-18 hours
**Total cost:** ~$0.01 for omnara, ~$0.50 for React-sized repos

**Key insight:** Two-way extraction catches what one-way misses (manual closures, forgotten keywords) for the same cost.

---

**Last Updated:** 2025-10-27
**Status:** Ready for implementation
**Next Action:** Implement graph queries in collector.go (Week 1)
