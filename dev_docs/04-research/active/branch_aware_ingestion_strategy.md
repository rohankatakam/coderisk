# Branch-Aware Incremental Ingestion Strategy

**Purpose:** Design incremental Layer 1 ingestion with GitHub language detection and branch-specific graphs
**Status:** Active Research → Will become ADR when validated
**Last Updated:** October 3, 2025
**Author:** AI Agent (Claude) responding to user requirements

---

## Problem Statement

### Current Implementation Issues

**1. Upfront Parsing Without Language Detection**
- Current: Parse all supported languages (JS/TS/Python) blindly
- Problem: Wastes resources on languages not in codebase
- Solution: Use GitHub `/repos/{owner}/{repo}/languages` API

**2. Branch-Agnostic Graph Storage**
- Current: Single graph per repository (main branch only)
- Problem: Developers work on feature branches that diverge from main
- Reality: "feature-branch is 2 commits ahead, 43 commits behind main"
- Impact: Risk assessment evaluates wrong codebase version

**3. Full Re-ingestion Per Branch**
- Current: No incremental strategy designed
- Problem: Re-parsing entire repo per branch = wasteful
- Reality: Only 1-5% of files change per branch
- Goal: Incremental updates without data duplication

### User Requirements (from conversation)

1. **Language Detection:** Use GitHub API to know which parsers to load
2. **Selective Parsing:** Only load Tree-sitter grammars for languages actually used
3. **Branch Context:** Evaluate code in context of developer's current branch
4. **Incremental Updates:** Parse only changed files, not entire codebase
5. **Data Integrity:** No redundant graph data, proper branch topology

---

## GitHub Language Detection API

### Endpoint

```
GET /repos/{owner}/{repo}/languages
```

### Response Example

```json
{
  "TypeScript": 532891,
  "JavaScript": 45231,
  "Python": 12345,
  "HTML": 8901,
  "CSS": 3456
}
```

### Usage in CodeRisk

**During `crisk init`:**
1. Fetch language statistics from GitHub API
2. Determine which Tree-sitter grammars to load
3. Skip parsers for unused languages
4. Store language manifest in PostgreSQL

**Benefits:**
- **Memory:** Load only 1-2 parsers instead of all 5
- **Speed:** Faster parser initialization
- **Accuracy:** Know primary language for reporting

**Example:**
```go
// Fetch languages from GitHub
langs, _ := github.GetRepoLanguages(ctx, "omnara-ai", "omnara")
// langs = {"TypeScript": 532891, "JavaScript": 45231}

// Determine parsers needed
parsers := []string{}
if langs["JavaScript"] > 0 || langs["TypeScript"] > 0 {
    parsers = append(parsers, "javascript", "typescript")
}
if langs["Python"] > 0 {
    parsers = append(parsers, "python")
}

// Initialize only needed parsers
for _, lang := range parsers {
    lp, _ := NewLanguageParser(lang)
    defer lp.Close()
}
```

---

## Branch-Aware Graph Architecture

### Core Principle: Git SHA as Source of Truth

**Every node stores:**
- `branch_name`: e.g., "main", "feature/auth-refactor"
- `git_sha`: SHA of commit this data was parsed from
- `last_updated`: Timestamp of last parsing

**Example:**
```cypher
CREATE (f:File {
  path: "src/auth.ts",
  branch: "feature/auth-refactor",
  git_sha: "abc123def456",
  last_updated: "2025-10-03T10:30:00Z",
  language: "typescript",
  loc: 234
})
```

### Layer 1: Branch-Specific Code Structure

**Storage Model:**

```
Graph Database Structure:
├── Base Graph (main branch)
│   ├── File nodes (all files on main)
│   ├── Function nodes
│   ├── Class nodes
│   └── CALLS/IMPORTS edges
│
├── Branch Delta (feature-branch)
│   ├── File nodes (only changed files)
│   ├── Function nodes (only changed functions)
│   ├── Class nodes (only changed classes)
│   └── CALLS/IMPORTS edges (only affected edges)
│
└── Query-Time Merge
    └── Delta overrides base for matching paths
```

**Node Uniqueness:**
- Unique ID: `{path}:{branch}:{git_sha}`
- Example: `src/auth.ts:feature-auth:abc123`

**Query Strategy:**
```cypher
// Get file in context of current branch
MATCH (f:File {path: $path, branch: $branch})
RETURN f

UNION

// Fallback to base if not in branch delta
MATCH (f:File {path: $path, branch: "main"})
WHERE NOT EXISTS {
  MATCH (delta:File {path: $path, branch: $branch})
}
RETURN f
```

### Layer 2 & 3: Branch-Agnostic Temporal Data

**Key Insight:** Git history and issues are repo-level, not branch-specific

**Storage:**
- `Commit`, `Developer`, `Issue`, `PullRequest` nodes: NO branch property
- Stored once, shared across all branches
- No duplication needed

**Relationships:**
```cypher
// MODIFIES edges link commits to files on specific branches
MATCH (c:Commit {sha: "abc123"})-[:MODIFIES]->(f:File {path: $path, branch: $branch})
```

---

## Incremental Ingestion Strategy

### Phase 1: Initial Ingestion (Main Branch)

**Trigger:** `crisk init <repo>`

**Steps:**
1. Fetch language statistics from GitHub API
2. Clone repository (shallow, `--depth 1`)
3. Load only needed Tree-sitter parsers
4. Parse all files on main branch
5. Create Layer 1 graph nodes with `branch: "main"`
6. Store git SHA as `main_git_sha` in PostgreSQL

**Time:** 10s for ~1K files (omnara)

### Phase 2: Branch Delta Creation

**Trigger:** `crisk check` on feature branch

**Detection:**
```bash
# Detect current branch
current_branch=$(git branch --show-current)
# e.g., "feature/auth-refactor"

# Get divergence point
merge_base=$(git merge-base main HEAD)
# e.g., "xyz789" (commit where branch diverged)

# Get changed files
changed_files=$(git diff --name-only $merge_base HEAD)
```

**Steps:**
1. Check if delta exists: `SELECT * FROM branch_deltas WHERE branch = $branch AND repo_id = $repo_id`
2. If exists and `git_sha` matches HEAD → use cached delta
3. If not exists or SHA differs:
   a. Get list of changed files (`git diff $merge_base HEAD`)
   b. Parse only changed files with Tree-sitter
   c. Create delta graph nodes with `branch: $branch`
   d. Store delta metadata in PostgreSQL
4. Query-time merge: Delta overrides base

**Time:** 3-5s for ~50 changed files (typical feature branch)

### Phase 3: Incremental Update (New Commits)

**Trigger:** New commit pushed to branch

**Steps:**
1. Get new commits since last ingestion: `git log $last_sha..HEAD`
2. Extract files changed in new commits only
3. Re-parse changed files
4. Update delta graph nodes
5. Update `git_sha` in PostgreSQL
6. Invalidate Redis caches for affected files

**Time:** 5-10s

---

## Data Model Changes

### PostgreSQL Schema Updates

**New table: `branch_deltas`**
```sql
CREATE TABLE branch_deltas (
  id SERIAL PRIMARY KEY,
  repo_id INT REFERENCES repositories(id),
  branch_name VARCHAR(255) NOT NULL,
  git_sha VARCHAR(40) NOT NULL,
  merge_base_sha VARCHAR(40) NOT NULL,  -- Divergence point from main
  created_at TIMESTAMP DEFAULT NOW(),
  last_updated TIMESTAMP DEFAULT NOW(),
  node_count INT,  -- Number of nodes in delta
  status VARCHAR(50),  -- 'active', 'merged', 'deleted'
  UNIQUE(repo_id, branch_name)
);
```

**Update table: `repositories`**
```sql
ALTER TABLE repositories ADD COLUMN languages JSONB;
-- Stores: {"TypeScript": 532891, "JavaScript": 45231}
```

### Graph Node Property Changes

**File node (new properties):**
```cypher
CREATE (f:File {
  path: "src/auth.ts",
  branch: "feature/auth-refactor",  -- NEW
  git_sha: "abc123",                 -- NEW
  language: "typescript",
  loc: 234,
  last_updated: "2025-10-03T10:30:00Z"  -- NEW
})
```

**Function/Class nodes (new properties):**
```cypher
CREATE (fn:Function {
  name: "authenticateUser",
  file_path: "src/auth.ts",
  branch: "feature/auth-refactor",  -- NEW
  signature: "function authenticateUser(token: string): boolean",
  start_line: 45,
  end_line: 67
})
```

---

## Storage Efficiency Analysis

### Scenario: Team of 10, 5 Active Branches

**Main branch (base graph):**
- 1,000 files
- 10,000 functions
- 2,000 classes
- Total nodes: ~13,000
- Size: ~2GB

**Feature branch (delta):**
- 50 changed files (5% of repo)
- 200 changed functions
- 40 changed classes
- Total nodes: ~290
- Size: ~10-50MB (98% reduction)

**Total storage:**
- Base: 2GB
- 5 deltas: 5 × 50MB = 250MB
- **Total: 2.25GB** (vs 30GB without deltas)
- **Savings: 92%**

---

## Query Performance Impact

### Query Types

**1. File lookup (branch-specific):**
```cypher
MATCH (f:File {path: $path, branch: $branch})
RETURN f
```
**Performance:** <10ms (indexed on path + branch)

**2. Fallback to base:**
```cypher
MATCH (f:File {path: $path})
WHERE f.branch IN [$branch, "main"]
RETURN f
ORDER BY CASE f.branch WHEN $branch THEN 0 ELSE 1 END
LIMIT 1
```
**Performance:** <20ms (delta overrides base)

**3. Cross-file analysis:**
```cypher
MATCH (f:File {path: $changed_file, branch: $branch})-[:IMPORTS]->(dep:File)
WHERE dep.branch IN [$branch, "main"]
RETURN dep
```
**Performance:** <100ms (1-hop traversal with branch filter)

---

## Implementation Phases

### Phase A: Language Detection (Priority 1)

**Files to update:**
- `internal/ingestion/processor.go` - Add GitHub language API call
- `internal/github/languages.go` - New file for language API client
- `internal/treesitter/parser.go` - Accept language list parameter

**Testing:**
- Unit test: Mock GitHub API response
- Integration test: Parse omnara-ai/omnara with detected languages

**Time:** 2-3 hours

### Phase B: Branch-Aware Parsing (Priority 2)

**Files to update:**
- `internal/treesitter/types.go` - Add `Branch` and `GitSHA` fields
- `internal/treesitter/parser.go` - Accept branch parameter
- `internal/graph/neo4j_backend.go` - Update node creation with branch properties

**Testing:**
- Unit test: Parse same file on different branches
- Integration test: Create delta, verify node uniqueness

**Time:** 4-6 hours

### Phase C: Incremental Delta Creation (Priority 3)

**Files to create:**
- `internal/git/diff.go` - Git diff extraction
- `internal/ingestion/delta.go` - Delta creation logic
- `cmd/crisk/check.go` - Detect branch, load delta

**Testing:**
- Integration test: Create branch, modify files, create delta
- Verify: Delta contains only changed files

**Time:** 6-8 hours

### Phase D: Query-Time Merge (Priority 4)

**Files to update:**
- `internal/graph/query_builder.go` - Branch-aware queries
- `internal/graph/neo4j_backend.go` - UNION query for delta + base

**Testing:**
- Integration test: Query file on feature branch
- Verify: Delta overrides base, fallback works

**Time:** 4-6 hours

---

## Success Criteria

✅ **Language Detection:**
- GitHub API call during `crisk init`
- Only needed parsers loaded
- Language manifest stored in PostgreSQL

✅ **Branch-Aware Graphs:**
- Nodes have `branch` and `git_sha` properties
- Delta graphs created for feature branches
- Base graph remains unchanged

✅ **Incremental Updates:**
- `git diff` used to detect changed files
- Only changed files re-parsed
- Delta graphs updated, not replaced

✅ **Data Integrity:**
- No duplicate nodes for same file on different branches
- Query-time merge works correctly
- Temporal data (Commits, Issues) shared across branches

✅ **Storage Efficiency:**
- Feature branch deltas <5% of base graph size
- Total storage <3GB for team of 10 with 5 branches

✅ **Performance:**
- `crisk check` on feature branch: <5s for cold start
- Incremental update: <10s for new commits
- Query latency: <100ms with branch filter

---

## Open Questions & Risks

### Q1: How to handle merge conflicts in delta graphs?

**Scenario:** Base graph updated while feature branch active

**Option A:** Invalidate delta, force re-parse
- ✅ Simple, guaranteed consistency
- ❌ Slow, user waits for re-parse

**Option B:** Update delta incrementally
- ✅ Fast, no user wait
- ❌ Complex, risk of inconsistency

**Recommendation:** Start with Option A, optimize to B later

### Q2: When to delete branch deltas?

**Trigger:** Branch merged to main

**Strategy:**
1. Mark as `status: 'merged'` immediately
2. Keep for 7 days (allow branch resurrection)
3. Delete after 7 days + no access

**Rationale:** Balances storage costs with user experience

### Q3: How to handle branch renames?

**Detection:** Git doesn't track renames, need manual update

**Strategy:**
1. Detect via git remote metadata
2. Update `branch_name` in PostgreSQL
3. Update `branch` property in graph nodes (expensive)

**Recommendation:** Document limitation, don't auto-handle renames

### R1: Graph query complexity increases

**Risk:** Branch filter adds overhead to every query

**Mitigation:**
- Index on (`path`, `branch`) composite key
- Cache results aggressively (Redis, 15min TTL)
- Monitor query performance

### R2: Delta graph divergence over time

**Risk:** Long-lived feature branches accumulate large deltas

**Mitigation:**
- Warn if delta >10% of base graph
- Suggest rebasing or merging
- Periodically re-baseline delta

---

## Next Steps

1. **Validate with ADR:** Convert this research to formal ADR
2. **Update spec.md:** Add language detection and branch requirements
3. **Update graph_ontology.md:** Document branch properties
4. **Implement Phase A:** Language detection first (lowest risk)
5. **Test with omnara-ai/omnara:** Real-world validation

---

## References

- [spec.md §1.5](../spec.md#15-scope-v10) - Multi-branch support in scope
- [team_and_branching.md](../../02-operations/team_and_branching.md) - Existing branch delta strategy
- [graph_ontology.md](../../01-architecture/graph_ontology.md) - Layer 1 schema
- [layer_1_treesitter.md](../03-implementation/integration_guides/layer_1_treesitter.md) - Current implementation
- [DEVELOPMENT_WORKFLOW.md](../DEVELOPMENT_WORKFLOW.md) - Implementation guardrails
- [DOCUMENTATION_WORKFLOW.md](../DOCUMENTATION_WORKFLOW.md) - Documentation process
