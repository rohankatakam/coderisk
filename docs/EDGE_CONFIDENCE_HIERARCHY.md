# Edge Confidence Hierarchy

**Purpose:** Define the confidence levels and priority rules for graph edges to prevent collisions and ensure data quality.

**Date Created:** 2025-11-13
**Status:** Production System Design

---

## Overview

CodeRisk uses two edge systems with different confidence levels:

1. **100% Confidence Edges** - Definitive relationships from GitHub/Git metadata
2. **Probabilistic Edges** - Inferred relationships from LLM extraction and temporal correlation (40-95% confidence)

**Priority Rule:** Never create a probabilistic edge if a 100% confidence edge already exists for the same relationship.

---

## 100% Confidence Edges (GitHub/Git Metadata)

These edges are created from definitive data sources and have implicit 100% confidence.

### AUTHORED
- **Direction:** Developer → Commit
- **Source:** `github_commits.author_email` (Git metadata)
- **Meaning:** Developer authored this commit
- **Properties:** `timestamp` (author date)
- **Created in:** `builder.go:processCommits()`

### MODIFIED
- **Direction:** Commit → File
- **Source:** `github_commits.raw_data.files[]` (Git diff)
- **Meaning:** Commit modified this file
- **Properties:** `additions`, `deletions`, `status` (added/modified/deleted)
- **Created in:** `builder.go:processCommits()`

### CREATED
- **Direction:** Developer → PR
- **Source:** `github_pull_requests.user_login` (GitHub API)
- **Meaning:** Developer created this pull request
- **Properties:** None
- **Created in:** `builder.go:processPRs()`

### MERGED_AS
- **Direction:** PR → Commit
- **Source:** `github_pull_requests.merge_commit_sha` (GitHub API)
- **Meaning:** PR was merged as this commit
- **Properties:** None
- **Created in:** `builder.go:linkPRsToMergeCommits()`
- **Note:** This is the definitive PR→Commit relationship

### REFERENCES
- **Direction:** Issue → PR
- **Source:** `github_issue_timeline` (event_type = 'cross-referenced', source_type = 'pull_request')
- **Meaning:** Issue was cross-referenced in this PR (GitHub verified)
- **Properties:** `source: "github_timeline"`, `confidence: 1.0`, `event_type: "cross-referenced"`
- **Created in:** `builder.go:createTimelineEdges()`
- **Added:** 2025-11-13 (Gap closure)

### CLOSED_BY
- **Direction:** Issue → Commit
- **Source:** `github_issue_timeline` (event_type = 'closed', source_sha present)
- **Meaning:** Issue was closed by this commit (GitHub verified)
- **Properties:** `source: "github_timeline"`, `confidence: 1.0`, `event_type: "closed"`
- **Created in:** `builder.go:createTimelineEdges()`
- **Added:** 2025-11-13 (Gap closure)

---

## Probabilistic Edges (LLM + Temporal Correlation)

These edges are inferred and have confidence scores between 0.4 and 0.95.

### ASSOCIATED_WITH
- **Direction:** Issue/PR → Commit  or  Issue → PR
- **Source:** LLM extraction from commit messages/PR bodies, or temporal correlation
- **Meaning:** Issue/PR is associated with this commit (inferred, not definitive)
- **Confidence Range:** 0.4 - 0.95
- **Properties:**
  - `relationship_type`: "fixes", "mentions", "associated_with"
  - `confidence`: 0.4 - 0.95
  - `detected_via`: "commit_extraction", "pr_extraction", "temporal"
  - `rationale`: Description of how link was detected
  - `evidence`: Array of tags for CLQS calculation
- **Created in:** `issue_linker.go:createIncidentEdges()`

**Confidence Score Meanings:**
- **0.9 - 0.95:** "Fixes #123" with validation
- **0.7 - 0.89:** Clear reference ("fix #123", "related to #123") with validation
- **0.55 - 0.75:** Temporal correlation (Issue closed within time window of commit/PR)
- **0.4 - 0.69:** Weak reference or failed validation
- **< 0.4:** Not created (filtered out)

### FIXED_BY
- **Direction:** Issue → Commit
- **Source:** LLM extraction with action="fixes"
- **Meaning:** Issue was likely fixed by this commit (inferred)
- **Confidence Range:** 0.7 - 0.95
- **Properties:** Same as ASSOCIATED_WITH
- **Created in:** `issue_linker.go:createIncidentEdges()`
- **Note:** Special case of ASSOCIATED_WITH when action="fixes" and entity is Issue

---

## Edge Priority Rules

### Rule 1: 100% Confidence Takes Priority
If a 100% confidence edge exists for relationship (A, B), **do not create** a probabilistic edge for the same relationship.

**Examples:**

#### Example 1: PR Merge Commit
```
Given:
  - PR #1 merged as commit abc123 (GitHub API data)

Correct:
  ✅ MERGED_AS edge: PR #1 → Commit abc123 (100% confidence)
  ❌ ASSOCIATED_WITH edge: PR #1 → Commit abc123 (DO NOT CREATE)

Reason: Commit message says "Merge pull request #1" but MERGED_AS already exists
```

#### Example 2: Issue Cross-Reference
```
Given:
  - Issue #42 cross-referenced in PR #45 (Timeline event from GitHub)

Correct:
  ✅ REFERENCES edge: Issue #42 → PR #45 (100% confidence)
  ❌ ASSOCIATED_WITH edge: Issue #42 → PR #45 (DO NOT CREATE)

Reason: PR body says "Fixes #42" but REFERENCES already exists
```

#### Example 3: Issue Closure
```
Given:
  - Issue #127 closed by commit abc123 (Timeline event from GitHub)

Correct:
  ✅ CLOSED_BY edge: Issue #127 → Commit abc123 (100% confidence)
  ❌ ASSOCIATED_WITH edge: Issue #127 → Commit abc123 (DO NOT CREATE)

Reason: Temporal correlation finds match but CLOSED_BY already exists
```

### Rule 2: Collision Prevention Implementation

**In `commit_extractor.go`:**
```go
// Skip merge commits - they already have MERGED_AS edges
if strings.HasPrefix(commit.Message, "Merge pull request") {
    continue // Don't process with LLM
}
```

**In `issue_linker.go`:**
```go
// Before creating ASSOCIATED_WITH edge, check if 100% edge exists
checkQuery := `
    MATCH (source)-[r]-(target)
    WHERE id(source) = $sourceID AND id(target) = $targetID
      AND type(r) IN ['MERGED_AS', 'REFERENCES', 'CLOSED_BY']
    RETURN COUNT(r) > 0 as exists
`
if exists {
    continue // Skip creating ASSOCIATED_WITH
}
```

### Rule 3: Build Order Ensures Priority

Graph construction order in `builder.go:BuildGraph()`:

1. **Process commits** → Creates AUTHORED, MODIFIED edges
2. **Process PRs** → Creates CREATED edges
3. **Link PRs to merge commits** → Creates MERGED_AS edges (100%)
4. **Process issues** → Creates Issue nodes
5. **Create timeline edges** → Creates REFERENCES, CLOSED_BY edges (100%) ← **BEFORE ASSOCIATED_WITH**
6. **Run temporal correlation** → Finds temporal matches (stored in PostgreSQL)
7. **Link issues** → Creates ASSOCIATED_WITH edges (with deduplication checks)
8. **Load validated links** → Creates/updates ASSOCIATED_WITH edges from multi-signal ground truth

This order ensures 100% confidence edges exist before probabilistic edges are created.

---

## Query Filtering Best Practices

### For Production Queries

Always filter out low-confidence edges:

```cypher
// Get Issue → Commit links
MATCH (i:Issue)-[r:ASSOCIATED_WITH]->(c:Commit)
WHERE r.confidence >= 0.4  // Filter low confidence
RETURN i, r, c
```

### Prefer 100% Confidence Edges

When both edge types might exist, prefer the definitive one:

```cypher
// Get PR → Commit relationship
MATCH (pr:PR)-[r]->(c:Commit)
WHERE type(r) IN ['MERGED_AS', 'ASSOCIATED_WITH']
WITH pr, c,
     CASE type(r)
       WHEN 'MERGED_AS' THEN 1.0
       ELSE r.confidence
     END as confidence,
     r
ORDER BY confidence DESC
RETURN pr, r, c
LIMIT 1  // Take highest confidence
```

---

## Monitoring and Validation

### Health Check Queries

**Check for Duplicate Edges (Should Return 0):**

```cypher
// Find PRs with both MERGED_AS and ASSOCIATED_WITH to same commit
MATCH (pr:PR)-[m:MERGED_AS]->(c:Commit)
MATCH (pr)-[a:ASSOCIATED_WITH]->(c)
RETURN pr.number, c.sha, 'DUPLICATE PR→Commit' as issue
```

```cypher
// Find Issues with both REFERENCES and ASSOCIATED_WITH to same PR
MATCH (i:Issue)-[r:REFERENCES]->(pr:PR)
MATCH (i)-[a:ASSOCIATED_WITH]->(pr)
RETURN i.number, pr.number, 'DUPLICATE Issue→PR' as issue
```

```cypher
// Find Issues with both CLOSED_BY and ASSOCIATED_WITH to same commit
MATCH (i:Issue)-[c:CLOSED_BY]->(commit:Commit)
MATCH (i)-[a:ASSOCIATED_WITH]->(commit)
RETURN i.number, commit.sha, 'DUPLICATE Issue→Commit' as issue
```

**Expected Result:** 0 rows for all queries

### Edge Count Statistics

```cypher
// Count edges by type
MATCH ()-[r]->()
RETURN type(r) as edge_type, COUNT(r) as count
ORDER BY count DESC
```

**Expected Ratios (Approximate):**
- MODIFIED: ~40-50% (dominant, many files per commit)
- AUTHORED: ~15-20%
- ASSOCIATED_WITH: ~15-20%
- TEMPORAL: ~10-15%
- CREATED: ~5-10%
- MERGED_AS: ~3-5%
- REFERENCES: ~2-5%
- CLOSED_BY: ~1-3%

---

## Change Log

### 2025-11-13 - Initial Version
- Added REFERENCES and CLOSED_BY edges (timeline system implementation)
- Defined collision prevention rules
- Documented priority hierarchy
- Added merge commit filter
- Added deduplication checks in issue linker

---

## References

- **Gap Analysis:** Gap analysis 2025-11-13 - Timeline edges missing
- **Bug Fix:** EXTRACTION_BUG_FIX_VERIFICATION.md - SHA-based matching
- **Architecture:** ASSOCIATED_WITH_EDGE_SYSTEM.md - Probabilistic edge system
- **Implementation:**
  - `internal/graph/builder.go` - 100% confidence edge creation
  - `internal/graph/issue_linker.go` - Probabilistic edge creation
  - `internal/github/commit_extractor.go` - LLM extraction
