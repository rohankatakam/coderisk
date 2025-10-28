# Issue Linking Implementation - Two-Way Extraction with LLM

**Date:** 2025-10-27
**Status:** Implementation Complete - Ready for Testing
**Reference:** REVISED_MVP_STRATEGY.md, GITHUB_API_ANALYSIS.md

---

## Overview

This document describes the implementation of two-way issue linking using OpenAI GPT-4o-mini for structured extraction of relationships between Issues, Commits, and Pull Requests.

## Architecture

### Data Flow

```
GitHub API
    ↓
1. Fetch Issues, Commits, PRs, Timeline Events → PostgreSQL
    ↓
2. Extract References using OpenAI GPT-4o-mini
    ├── Issue → Commit/PR (issue_extraction)
    ├── Commit → Issue (commit_extraction)
    ├── PR → Issue (pr_extraction)
    └── Timeline → Cross-references (timeline_extraction)
    ↓
3. Merge Bidirectional References (confidence boost)
    ↓
4. Store in github_issue_commit_refs table
    ↓
5. Create Neo4j Graph
    ├── Issue nodes
    ├── Commit nodes (already exists)
    ├── PR nodes (already exists)
    └── FIXED_BY edges (Issue → Commit/PR)
```

## Implementation Components

### 1. Database Schema

**File:** `scripts/schema/postgresql_staging.sql`

**New Tables:**

#### `github_issue_timeline`
Stores GitHub Timeline API events (cross-references, comments, etc.)

```sql
CREATE TABLE github_issue_timeline (
    id BIGSERIAL PRIMARY KEY,
    issue_id BIGINT NOT NULL REFERENCES github_issues(id),
    event_type VARCHAR(50) NOT NULL,
    source_type VARCHAR(20),              -- 'pr', 'issue', 'commit'
    source_number INTEGER,                -- PR or Issue number
    source_body TEXT,                     -- PR/Issue body text
    ...
);
```

#### `github_issue_commit_refs`
Stores extracted references with confidence scores

```sql
CREATE TABLE github_issue_commit_refs (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL,
    issue_number INTEGER NOT NULL,
    commit_sha VARCHAR(40),
    pr_number INTEGER,
    action VARCHAR(20) NOT NULL,          -- 'fixes', 'mentions'
    confidence FLOAT NOT NULL,            -- 0.0 to 1.0
    detection_method VARCHAR(50),         -- 'bidirectional', 'issue_extraction', etc.
    extracted_from VARCHAR(50),
    ...
);
```

### 2. GitHub API Fetching

**File:** `internal/github/fetcher.go`

**New Methods:**

- `FetchIssueTimelines()` - Fetches timeline events for closed issues
- `fetchIssueTimeline()` - Fetches timeline for a single issue
- `storeTimelineEvent()` - Stores timeline event with cross-reference data

**Integration:** Called from `FetchAll()` after fetching issues

### 3. LLM Integration

**File:** `internal/llm/client.go`

**New Method:**

- `CompleteJSON()` - Sends prompts with `response_format: json_object` for structured outputs

**Model:** GPT-4o-mini (`gpt-4o-mini`)
**Temperature:** 0.1 (for consistency)
**Max Tokens:** 2000

### 4. Issue Extraction

**File:** `internal/github/issue_extractor.go`

**Class:** `IssueExtractor`

**Methods:**
- `ExtractReferences()` - Processes all issues in batches
- `processBatch()` - Sends batch to LLM and stores results
- `ExtractTimelineReferences()` - Extracts from timeline cross-references

**Batch Size:** 20 issues per LLM call

**Prompt Template:**
```
You are a GitHub issue analyzer. Extract commit and PR references from issue text.

Rules:
- type: "commit" or "pr"
- action: "fixes" (for closes/resolves/fixes), "mentions" (for related/see)
- confidence: 0.9-1.0 for explicit fixes, 0.7-0.9 for "fixed by", 0.5-0.7 for mentions
```

### 5. Commit/PR Extraction

**File:** `internal/github/commit_extractor.go`

**Class:** `CommitExtractor`

**Methods:**
- `ExtractCommitReferences()` - Processes all commits
- `ExtractPRReferences()` - Processes all PRs
- `processCommitBatch()` - Batch processes commits
- `processPRBatch()` - Batch processes PRs

**Batch Size:** 20 commits/PRs per LLM call

**Prompt Template:**
```
You are a Git commit analyzer. Extract issue references from commit messages.

Rules:
- Look for patterns: "Fixes #123", "Closes #456", "Resolves #789"
- Ignore negations: "Don't fix #123"
- action: "fixes" for closes/resolves/fixes, "mentions" for related/see
- confidence: 0.9-1.0 for "Fixes #123", 0.7-0.9 for "fix #123"
```

### 6. Reference Merging & Graph Construction

**File:** `internal/graph/issue_linker.go`

**Class:** `IssueLinker`

**Methods:**
- `LinkIssues()` - Main orchestrator
- `createIssueNodes()` - Creates Issue nodes in Neo4j
- `createFixedByEdges()` - Creates FIXED_BY edges
- `mergeReferences()` - Merges bidirectional references with confidence boost

**Bidirectional Logic:**
```go
// If same (issue, commit) found from multiple detection methods:
if different_detection_methods {
    detection_method = "bidirectional"
    confidence += 0.05  // Boost by 5%
    if confidence > 0.95 {
        confidence = 0.95  // Cap at 95%
    }
}
```

**Edge Filtering:**
- Only create edges for `action == "fixes"` (not "mentions")
- Only create edges with `confidence >= 0.75`

### 7. Graph Builder Integration

**File:** `internal/graph/builder.go`

**Updated:** `BuildGraph()` method

**New Step:**
```go
// Link Issues to Commits/PRs (creates FIXED_BY edges)
linkStats, err := b.linkIssues(ctx, repoID)
```

**Neo4j Schema:**

**Nodes:**
- `Issue` - Properties: `number`, `title`, `state`, `is_bug`, `created_at`, `closed_at`

**Edges:**
- `FIXED_BY` - From `Issue` to `Commit` or `PR`
  - Properties: `confidence`, `detection_method`, `extracted_from`

## Usage

### Prerequisites

1. OpenAI API key configured (via `OPENAI_API_KEY` environment variable)
2. Phase 2 enabled: `export PHASE2_ENABLED=true`
3. PostgreSQL with new schema tables
4. Neo4j running

### Running the Pipeline

```bash
# 1. Fetch GitHub data (includes timeline events)
./bin/crisk init

# This will:
# - Fetch commits, issues, PRs from GitHub
# - Fetch timeline events for closed issues
# - Store in PostgreSQL

# 2. Extract references using LLM
# (Manual step for now - to be integrated)
# Run extraction scripts (to be created)

# 3. Build graph (includes issue linking)
# Graph builder will:
# - Create Issue nodes
# - Create FIXED_BY edges based on extracted references
```

### Database Queries

**Check extracted references:**
```sql
SELECT issue_number, commit_sha, pr_number, action, confidence, detection_method
FROM github_issue_commit_refs
WHERE repo_id = 1
ORDER BY confidence DESC;
```

**Count by detection method:**
```sql
SELECT detection_method, COUNT(*) as count, AVG(confidence) as avg_confidence
FROM github_issue_commit_refs
WHERE repo_id = 1
GROUP BY detection_method;
```

### Neo4j Queries

**Check FIXED_BY edges:**
```cypher
MATCH (i:Issue)-[r:FIXED_BY]->(c:Commit)
RETURN i.number, i.title, c.sha, r.confidence, r.detection_method
LIMIT 20;
```

**Count edges by detection method:**
```cypher
MATCH (i:Issue)-[r:FIXED_BY]->(c:Commit)
RETURN r.detection_method, count(*) as count, avg(r.confidence) as avg_conf
ORDER BY count DESC;
```

## Expected Results

### For omnara-ai/omnara Repository

**Inputs:**
- ~80 issues (closed within 90 days)
- ~200 commits
- ~150 PRs

**Expected Outputs:**
- ~100-150 references extracted
- ~80-120 FIXED_BY edges created
- Detection methods distribution:
  - pr_extraction: ~60% (most PRs use "Fixes #123")
  - bidirectional: ~20-30% (high confidence matches)
  - commit_extraction: ~10-20%
  - issue_extraction/timeline_extraction: ~10%

**Cost:**
- Token usage: ~25,000 tokens
- Cost: ~$0.01 (less than 1 cent!)

## Testing Checklist

- [ ] Database schema migration applied successfully
- [ ] Timeline events fetched for closed issues
- [ ] Issue extraction runs without errors
- [ ] Commit/PR extraction runs without errors
- [ ] References stored in `github_issue_commit_refs` table
- [ ] Issue nodes created in Neo4j
- [ ] FIXED_BY edges created with correct properties
- [ ] Bidirectional references have confidence boost
- [ ] Edge count matches expected distribution
- [ ] Manual validation of 10-20 random edges

## Next Steps

### Integration Tasks (High Priority)

1. **Create extraction orchestrator script**
   - Coordinates issue, commit, PR extraction
   - Handles errors and retries
   - Logs progress and stats

2. **Integrate into `crisk init` command**
   - Run extractions after GitHub fetch
   - Before graph construction
   - Add `--skip-llm` flag for testing without LLM

3. **Add extraction status tracking**
   - Track which issues/commits have been processed
   - Enable incremental extraction
   - Avoid re-processing same data

### Testing Tasks

1. **Unit tests**
   - Test extraction prompts
   - Test reference merging logic
   - Test edge filtering

2. **Integration test**
   - Fresh db → fetch → extract → build graph
   - Verify node/edge counts
   - Check bidirectional rate

3. **Manual validation**
   - Sample 20 random FIXED_BY edges
   - Verify on GitHub web UI
   - Calculate accuracy

### Performance Optimization (Future)

1. **Parallel processing**
   - Process multiple batches concurrently
   - Respect rate limits

2. **Caching**
   - Cache LLM responses
   - Reuse for similar inputs

3. **Smart batching**
   - Group similar issues together
   - Vary batch size based on content length

## Files Created/Modified

### New Files
- `scripts/schema/postgresql_staging.sql` - Added timeline and refs tables
- `internal/github/issue_extractor.go` - Issue extraction logic
- `internal/github/commit_extractor.go` - Commit/PR extraction logic
- `internal/graph/issue_linker.go` - Reference merging and graph construction
- `ISSUE_LINKING_IMPLEMENTATION.md` - This document

### Modified Files
- `internal/database/staging.go` - Added timeline and refs data methods
- `internal/github/fetcher.go` - Added timeline fetching
- `internal/llm/client.go` - Added CompleteJSON method
- `internal/graph/builder.go` - Integrated issue linking

## References

- [REVISED_MVP_STRATEGY.md](REVISED_MVP_STRATEGY.md) - Two-way extraction strategy
- [GITHUB_API_ANALYSIS.md](GITHUB_API_ANALYSIS.md) - Timeline API analysis
- [PRE_COMMIT_GRAPH_SPEC.md](dev_docs/02-specifications/PRE_COMMIT_GRAPH_SPEC.md) - Graph schema

---

**Status:** ✅ Implementation complete, ready for testing on omnara repository

**Next Action:** Create extraction orchestrator and integrate into init command
