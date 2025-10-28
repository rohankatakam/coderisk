# CodeRisk Implementation Gap Analysis

**Date:** 2025-01-24
**Status:** Current State vs MVP Documentation
**Purpose:** Identify what needs to change to align with MVP specifications

---

## Executive Summary

The current implementation is **approximately 60% complete** toward the MVP vision. There are significant architectural gaps between what's documented in the MVP specs and what's actually implemented.

**Key Findings:**
- ✅ Basic infrastructure is solid (Neo4j, PostgreSQL, GitHub API integration)
- ❌ Ingestion strategy doesn't match MVP spec (missing PR-based approach)
- ❌ Graph schema has mismatches (edges referenced in queries don't exist)
- ❌ Risk assessment queries are defined but not implemented
- ❌ No end-to-end risk analysis pipeline

---

## Part 1: Ingestion Strategy Gaps

### What MVP Documentation Says (ingestion.md)

**Philosophy:** PR-centric ingestion from main branch
```
1. Fetch PRs merged in last N days (default: 90)
2. For each PR, fetch commit list
3. For each commit, fetch commit details (files[], patch)
4. Store in PostgreSQL with raw_data
5. Run TreeSitter on HEAD (local)
6. Merge file nodes (TreeSitter + commit files)
7. Build Neo4j graph with all edges
```

**Key Points:**
- Default: `crisk init` = last 90 days of merged PRs
- Flexible time windows: `--days 7`, `--days 90`, `--all`
- Main branch only (feature branches ignored by design)
- Zero API calls during `crisk check` (all pre-fetched)
- Store patches in PostgreSQL for LLM context

### What Current Implementation Does

**File:** [internal/ingestion/processor.go](internal/ingestion/processor.go)

```go
// Current approach: Two separate systems
ProcessRepository()           // Clone + TreeSitter parse (Layer 1)
  ├─ Shallow clone (--depth 1)
  ├─ Walk files, parse with TreeSitter
  └─ Create File nodes + DEPENDS_ON edges

ProcessRepositoryFromPath()   // Local-only parsing (no GitHub)
  └─ Same as above but for pre-cloned repos
```

**File:** [internal/github/fetcher.go](internal/github/fetcher.go)

```go
// GitHub data fetching (separate system)
FetchAll(owner, repo)
  ├─ FetchCommits() [90-day window] → PostgreSQL
  ├─ FetchIssues() [90-day window]  → PostgreSQL
  ├─ FetchPRs() [90-day window]     → PostgreSQL
  └─ FetchBranches()                → PostgreSQL
```

**File:** [internal/graph/builder.go](internal/graph/builder.go)

```go
// Graph building (only processes GitHub data)
BuildGraph(repoID, repoPath)
  ├─ processCommits()      // GitHub commits → Commit nodes
  ├─ processPRs()          // GitHub PRs → PR nodes
  ├─ calculateOwnership()  // Creates OWNS edges
  └─ linkCommitsToPRs()    // Creates IN_PR edges
```

### Critical Gaps

| Issue | Current State | MVP Spec | Impact |
|-------|--------------|----------|---------|
| **PR-based fetching** | Fetches commits directly (not via PRs) | Fetch PRs first, then commits per PR | Missing PR context for commits |
| **File merge** | TreeSitter files and commit files separate | Union of both with path resolution | File nodes incomplete |
| **Patch storage** | Not storing patches in PostgreSQL | Store full patch data in `raw_data` column | LLM analysis won't work |
| **MODIFIED edges** | Created from local git only (Layer 2) | Created from GitHub commit `files[]` data | Graph incomplete for `crisk init` |
| **Time window flags** | Hardcoded 90 days | `--days N` and `--all` flags | No flexibility |
| **Main branch filtering** | No filtering by branch | Only main branch commits | Feature branch pollution |

### What Needs to Change

**File:** [cmd/crisk/init.go](cmd/crisk/init.go) (likely missing or incomplete)

```go
// NEEDED: CLI flags for time windows
initCmd.Flags().Int("days", 90, "Ingest PRs merged in last N days")
initCmd.Flags().Bool("all", false, "Ingest entire repository history")
```

**File:** [internal/github/fetcher.go](internal/github/fetcher.go)

```go
// NEEDED: New method for PR-centric fetching
func (f *Fetcher) FetchMergedPRs(ctx context.Context, since time.Time) ([]*PR, error) {
    // GET /repos/{owner}/{repo}/pulls?state=closed&sort=updated&since={since}
    // Filter for merged_at != null
}

func (f *Fetcher) FetchPRCommits(ctx context.Context, prNumber int) ([]*Commit, error) {
    // GET /repos/{owner}/{repo}/pulls/{number}/commits
}

func (f *Fetcher) FetchCommitDetails(ctx context.Context, sha string) (*CommitDetail, error) {
    // GET /repos/{owner}/{repo}/commits/{sha}
    // MUST include files[] array with patch data
}
```

**File:** [internal/database/staging.go](internal/database/staging.go)

```go
// NEEDED: Store patches in github_commits.raw_data
func (db *Database) StoreCommitWithPatch(commit *Commit) error {
    // INSERT github_commits with full files[] + patch in raw_data JSONB
}
```

**File:** [internal/ingestion/processor.go](internal/ingestion/processor.go)

```go
// NEEDED: Unified ingestion orchestrator
func (p *Processor) IngestRepository(ctx context.Context, owner, repo string, days int, all bool) error {
    // 1. Determine time window
    since := calculateSince(days, all)

    // 2. Fetch PRs merged since date
    prs := github.FetchMergedPRs(ctx, since)

    // 3. For each PR, fetch commits
    commits := github.FetchPRCommits(ctx, prs)

    // 4. Fetch commit details with files[] and patch
    commitDetails := github.FetchCommitDetails(ctx, commits)

    // 5. Store in PostgreSQL with patches
    database.StoreCommitsWithPatches(commitDetails)

    // 6. Run TreeSitter on HEAD
    treeSitterFiles := treesitter.AnalyzeRepository(repoPath)

    // 7. Merge file lists (union + path resolution)
    allFiles := mergeFileLists(commitDetails.Files, treeSitterFiles)

    // 8. Build Neo4j graph
    graph.BuildGraph(commits, prs, allFiles)
}
```

---

## Part 2: Graph Schema Gaps

### What MVP Documentation Says (schema.md)

**4 Core Nodes:**
- File (path, language, loc, last_modified)
- Developer (email, name, github_login)
- Commit (sha, message, author_email, committed_at, additions, deletions)
- PR (number, title, state, base_branch, head_branch, author_email, merged_at, merge_commit_sha)

**6 Core Edges:**
1. `MODIFIED` (Commit → File) - additions, deletions, status, previous_filename
2. `DEPENDS_ON` (File → File) - import_type
3. `CREATED` (Developer → PR)
4. `AUTHORED` (Developer → Commit)
5. `IN_PR` (Commit → PR)
6. `MERGED_AS` (PR → Commit)

**Explicitly Deferred (Post-MVP):**
- Issue node
- `FIXED_BY` edge (Issue → Commit)
- `CAUSED` edge (Commit → Issue)
- `CALLS` edge (File → File)
- `OWNS` edge (pre-computed ownership)

**Key Note from schema.md:**
> "Ownership computed dynamically from [:AUTHORED] + [:MODIFIED] edges (no pre-computed [:OWNS] edge)"

### What Current Implementation Has

**File:** [internal/graph/builder.go](internal/graph/builder.go)

**Nodes Created:**
- ✅ File (from TreeSitter)
- ✅ Commit (from GitHub commits)
- ✅ Developer (from commit authors)
- ✅ PR (from GitHub PRs)

**Edges Created:**
- ✅ `DEPENDS_ON` (File → File) - from TreeSitter imports
- ✅ `AUTHORED` (Developer → Commit)
- ✅ `MODIFIED` (Commit → File) - **BUT only in Layer 2 (local git), not from GitHub**
- ✅ `OWNS` (Developer → File) - **CONTRADICTS MVP spec (should be dynamic)**
- ✅ `CREATED` (Developer → PR)
- ✅ `MERGED_AS` (PR → Commit)
- ✅ `IN_PR` (Commit → PR)
- ⚠️ `CO_CHANGED` (File - File) - **DEPRECATED but still created in Layer 2**

### Critical Schema Mismatches

| Issue | Current State | MVP Spec | Fix Required |
|-------|--------------|----------|--------------|
| **OWNS edge** | Pre-computed and stored | Should be dynamic query only | Remove edge creation, update queries |
| **CO_CHANGED edge** | Created in Layer 2 (deprecated) | Should be dynamic query only | Remove edge creation entirely |
| **MODIFIED source** | Only from local git (Layer 2) | From GitHub commit `files[]` data | Add MODIFIED creation in BuildGraph |
| **Issue nodes** | Not created (correct per MVP) | Confirmed deferred | None (matches spec) |
| **LINKED_TO edge** | Referenced in queries but never created | Not in MVP schema | Remove from queries |
| **CALLS edge** | Referenced in queries but never created | Deferred to post-MVP | Remove from queries |

### What Needs to Change

**File:** [internal/graph/builder.go](internal/graph/builder.go)

```go
// REMOVE: Pre-computed ownership edges
func (b *Builder) calculateOwnership() error {
    // DELETE THIS METHOD - ownership should be dynamic
}

// REMOVE: From BuildGraph()
// b.calculateOwnership()  // DELETE THIS LINE

// ADD: Process MODIFIED edges from GitHub commit data
func (b *Builder) processCommitFiles(commits []*Commit) error {
    // For each commit in github_commits table:
    //   Extract files[] from raw_data JSONB
    //   For each file:
    //     Create MODIFIED edge (Commit → File)
    //     Properties: additions, deletions, status, previous_filename
}

// UPDATE: BuildGraph() to call processCommitFiles
func (b *Builder) BuildGraph(repoID string, repoPath string) error {
    b.processCommits()
    b.processCommitFiles()  // NEW: Create MODIFIED edges from GitHub data
    b.processPRs()
    b.linkCommitsToPRs()
    // b.calculateOwnership()  // REMOVE
}
```

**File:** [internal/ingestion/processor.go](internal/ingestion/processor.go)

```go
// REMOVE: AddLayer2CoChangedEdges (already marked deprecated)
// DELETE THIS METHOD ENTIRELY

// REMOVE: All CO_CHANGED edge creation code
```

---

## Part 3: Query Implementation Gaps

### What MVP Documentation Says (schema.md)

**5 Critical Queries** that must work for `crisk check`:

1. **File Ownership** (dynamic from AUTHORED + MODIFIED)
2. **Blast Radius** (DEPENDS_ON traversal)
3. **Co-Change Partners** (dynamic from MODIFIED + time window)
4. **Incident History** (commit message regex on bug keywords)
5. **Recent Activity** (last 5 commits via MODIFIED)

**Key Point:** All queries operate on main branch data only (no branch filtering needed because ingestion only fetches main branch).

### What Current Implementation Has

**File:** [internal/risk/queries.go](internal/risk/queries.go)

**7 Fixed Queries Defined:**
1. Recent Incidents (uses `LINKED_TO` edge - doesn't exist)
2. Dependency Count (uses `IMPORTS` + `CALLS` edges - CALLS doesn't exist)
3. Co-Change Partners (uses `CO_CHANGED` edge - deprecated)
4. Ownership (filters by default branch - correct approach)
5. Blast Radius (uses variable-length `DEPENDS_ON` - correct)
6. Incident History (uses `LINKED_TO` edge - doesn't exist)
7. Recent Commits (filters by default branch - correct)

**File:** [internal/risk/collector.go](internal/risk/collector.go)

```go
func (c *Collector) CollectPhase1Data(ctx context.Context, changeset *types.Changeset) (*types.Phase1Data, error) {
    // CURRENT: Only implements change_complexity analysis (from git diff)
    // MISSING: All 7 graph queries

    return &types.Phase1Data{
        ChangeComplexity: complexity,
        // All other fields: zero values (not populated)
        RecentIncidents: nil,
        DependencyCount: 0,
        CoChangePartners: nil,
        Ownership: nil,
        BlastRadius: nil,
        IncidentHistory: nil,
        RecentCommits: nil,
    }, nil
}
```

### Critical Query Gaps

| Query | Status | Blocker | Fix Required |
|-------|--------|---------|--------------|
| **Ownership** | Defined but not executed | None (schema correct) | Implement in Collector |
| **Blast Radius** | Defined but not executed | DEPENDS_ON edges incomplete (only relative imports) | Fix TreeSitter + implement in Collector |
| **Co-Change** | Defined but wrong edge type | Uses CO_CHANGED (deprecated) | Rewrite to dynamic query |
| **Incident History** | Defined but uses wrong edge | Uses LINKED_TO (doesn't exist) | Rewrite to commit message regex |
| **Recent Activity** | Defined but not executed | None (schema correct) | Implement in Collector |
| **Recent Incidents** | Defined but uses wrong edge | Uses LINKED_TO (doesn't exist) | Remove or defer to post-MVP |
| **Dependency Count** | Defined but uses wrong edge | Uses CALLS (doesn't exist) | Rewrite to only use DEPENDS_ON |

### What Needs to Change

**File:** [internal/risk/queries.go](internal/risk/queries.go)

```go
// UPDATE: Query 3 - Co-Change Partners (remove CO_CHANGED edge reference)
const CoChangePartnersQuery = `
MATCH (f:File {path: $file_path})<-[:MODIFIED]-(c:Commit)
WHERE c.committed_at > datetime() - duration('P90D')
WITH f, collect(c) as target_commits, count(c) as total_commits

UNWIND target_commits as commit
MATCH (commit)-[:MODIFIED]->(other:File)
WHERE other.path <> $file_path
WITH other, count(commit) as co_changes, total_commits
WITH other, co_changes, toFloat(co_changes)/toFloat(total_commits) as frequency
WHERE frequency > 0.5
RETURN other.path, co_changes, frequency
ORDER BY frequency DESC
LIMIT 10
`

// UPDATE: Query 4 - Incident History (use commit message regex, not LINKED_TO)
const IncidentHistoryQuery = `
MATCH (f:File {path: $file_path})<-[:MODIFIED]-(c:Commit)
WHERE c.message =~ '(?i).*(fix|bug|hotfix|patch).*'
  AND c.committed_at > datetime() - duration('P180D')
RETURN c.sha, c.message, c.committed_at
ORDER BY c.committed_at DESC
LIMIT 5
`

// REMOVE: Query 1 (Recent Incidents) - requires Issue nodes, deferred to post-MVP
// DELETE OR MARK AS DISABLED

// UPDATE: Query 2 (Dependency Count) - remove CALLS edge reference
const DependencyCountQuery = `
MATCH (f:File {path: $file_path})<-[:DEPENDS_ON*1..2]-(dependent:File)
RETURN count(distinct dependent) as dependency_count
`

// UPDATE: Query 4 (Ownership) - remove OWNS edge, make fully dynamic
const OwnershipQuery = `
MATCH (d:Developer)-[:AUTHORED]->(c:Commit)-[:MODIFIED]->(f:File {path: $file_path})
WITH d, count(c) as commit_count, max(c.committed_at) as last_commit_date
WITH d, commit_count, last_commit_date, sum(commit_count) OVER () as total_file_commits
RETURN d.email, d.name, commit_count, last_commit_date,
       toFloat(commit_count)/toFloat(total_file_commits) as ownership_percentage
ORDER BY commit_count DESC
LIMIT 3
`
```

**File:** [internal/risk/collector.go](internal/risk/collector.go)

```go
// IMPLEMENT: Actually execute the queries
func (c *Collector) CollectPhase1Data(ctx context.Context, changeset *types.Changeset) (*types.Phase1Data, error) {
    data := &types.Phase1Data{}

    // 1. Change complexity (already implemented)
    data.ChangeComplexity = c.analyzeChangeComplexity(changeset)

    // 2. Execute ownership query
    ownershipResult, err := c.graphBackend.Query(ctx, OwnershipQuery, map[string]any{
        "file_path": changeset.FilePath,
    })
    if err != nil {
        return nil, err
    }
    data.Ownership = parseOwnershipResult(ownershipResult)

    // 3. Execute blast radius query
    blastResult, err := c.graphBackend.Query(ctx, BlastRadiusQuery, map[string]any{
        "file_path": changeset.FilePath,
    })
    if err != nil {
        return nil, err
    }
    data.BlastRadius = parseBlastRadiusResult(blastResult)

    // 4. Execute co-change query
    coChangeResult, err := c.graphBackend.Query(ctx, CoChangePartnersQuery, map[string]any{
        "file_path": changeset.FilePath,
    })
    if err != nil {
        return nil, err
    }
    data.CoChangePartners = parseCoChangeResult(coChangeResult)

    // 5. Execute incident history query
    incidentResult, err := c.graphBackend.Query(ctx, IncidentHistoryQuery, map[string]any{
        "file_path": changeset.FilePath,
    })
    if err != nil {
        return nil, err
    }
    data.IncidentHistory = parseIncidentResult(incidentResult)

    // 6. Execute recent commits query
    recentResult, err := c.graphBackend.Query(ctx, RecentCommitsQuery, map[string]any{
        "file_path": changeset.FilePath,
    })
    if err != nil {
        return nil, err
    }
    data.RecentCommits = parseRecentCommitsResult(recentResult)

    return data, nil
}
```

---

## Part 4: Architectural Issues

### Issue 1: Two Disconnected Systems

**Current State:**

```
System A: Local Parsing (Layer 1)
  ├─ ProcessRepository() / ProcessRepositoryFromPath()
  ├─ TreeSitter parsing → File nodes
  └─ Local git history → CO_CHANGED edges (deprecated)

System B: GitHub Data (Layers 2-3)
  ├─ FetchAll() → PostgreSQL staging
  ├─ BuildGraph() → Commit, Developer, PR nodes
  └─ Never touches File nodes from System A
```

**Problem:** File nodes created by System A are isolated from Commit nodes created by System B. MODIFIED edges only created if Layer 2 (local git) is run, but `crisk init` (GitHub-only) doesn't create MODIFIED edges.

**MVP Vision:** Unified system where:
1. GitHub API fetches commit data with `files[]` arrays
2. TreeSitter analyzes HEAD for file structure
3. File nodes are union of both sources (merged by path)
4. MODIFIED edges created from GitHub commit `files[]` data
5. DEPENDS_ON edges created from TreeSitter import analysis

**Fix:** Merge systems in a new unified ingestion flow.

---

### Issue 2: No Main Branch Filtering

**Current State:**

```go
// github/fetcher.go
func (f *Fetcher) FetchCommits(ctx context.Context) error {
    // Fetches commits from ALL branches (no filtering)
    // GET /repos/{owner}/{repo}/commits (defaults to default branch, but includes merge commits from all branches)
}
```

**Problem:** Queries reference `:ON_BRANCH` filtering, but:
- No Branch nodes created
- No ON_BRANCH edges created
- Commits from feature branches pollute ownership/co-change data

**MVP Vision (ingestion.md):**
> "All commits in graph are from main branch (ingested via merged PRs) → No branch filtering needed!"

**Fix:**
1. Fetch only PRs merged to main branch
2. Fetch commits per PR (these are the pre-merge commits)
3. Link commits to PRs (IN_PR edge)
4. All commits in graph are automatically main-branch-related (via PR merge)
5. Remove `:ON_BRANCH` filtering from queries (not needed)

---

### Issue 3: Deprecated Code Not Removed

**Files with deprecated/unused code:**

1. **[internal/temporal/co_change.go](internal/temporal/co_change.go)**
   - Pre-computed co-change edges (marked DEPRECATED in comments)
   - Should be dynamic query only

2. **[internal/ingestion/processor.go](internal/ingestion/processor.go)**
   - `AddLayer2CoChangedEdges()` (marked deprecated)
   - `AddLayer3IncidentNodes()` (commented out)

3. **[internal/graph/builder.go](internal/graph/builder.go)**
   - `calculateOwnership()` - creates OWNS edges (should be dynamic per MVP spec)

4. **Throughout codebase:**
   - References to Function, Class nodes (not in current schema)
   - References to CALLS edges (deferred to post-MVP)

**Fix:** Clean sweep to remove deprecated code and comments referencing removed features.

---

### Issue 4: PostgreSQL Underutilized

**Current State:**

```go
// database/staging.go - Store raw JSON
func (db *Database) StoreCommit(commit *Commit) error {
    // Stores commit with raw_data JSONB
}

// graph/builder.go - Read metadata only
func (b *Builder) processCommits() error {
    // Reads sha, message, author, date
    // Ignores raw_data (files[], patch)
}
```

**Problem:**
- Commit patches fetched from GitHub API
- Stored in PostgreSQL `raw_data` column
- Never extracted or used in graph
- MODIFIED edges not created from this data

**MVP Vision (ingestion.md):**
> "Store in PostgreSQL `commits` table with `raw_data` (files[], patch)"
> "Build Neo4j graph: [:MODIFIED] edges (600 from commit files[])"

**Fix:**
1. Extract `files[]` array from `github_commits.raw_data`
2. For each file in array:
   - Create/merge File node
   - Create MODIFIED edge (Commit → File)
   - Store additions, deletions, status, previous_filename as edge properties
3. Keep patch data in PostgreSQL for LLM analysis during `crisk check`

---

### Issue 5: TreeSitter Import Detection Incomplete

**Current State:**

**File:** [internal/treesitter/parser.go](internal/treesitter/parser.go)

```go
// Documented limitation: ~60% success rate on imports
// Only handles relative paths (./utils, ../lib)
// Fails on:
//   - Aliases (@/components, ~/utils)
//   - External packages (react, lodash)
//   - Dynamic imports (require(variable))
```

**Problem:** DEPENDS_ON edges incomplete → Blast radius queries unreliable

**MVP Vision:** Focus on internal dependencies for blast radius

**Fix Options:**
1. **Short-term:** Document limitation, accept 60% coverage for MVP
2. **Medium-term:** Add alias resolution (tsconfig.json, package.json parsing)
3. **Long-term:** Full semantic analysis (requires language servers)

**Recommendation for MVP:** Accept current limitation, improve post-MVP

---

## Part 5: End-to-End Flow Gaps

### What Should Happen (`crisk init`)

```
User runs: crisk init --days 90

1. Determine time window (90 days ago)
2. Fetch PRs merged in last 90 days → ~150 PRs
3. For each PR, fetch commit list → ~200 unique commits
4. For each commit, fetch details with files[] and patch → ~200 API calls
5. Store all data in PostgreSQL with raw_data
6. Run TreeSitter on HEAD → ~2000 files analyzed
7. Merge file lists (union of commit files + TreeSitter files)
8. Build Neo4j graph:
   - Create Commit, Developer, PR, File nodes
   - Create AUTHORED, MODIFIED, CREATED, MERGED_AS, IN_PR, DEPENDS_ON edges
9. Report statistics

Total: ~352 API calls, 3-4 minutes
```

### What Currently Happens

**Command:** `crisk init` (no time window flags implemented)

```
1. FetchAll(owner, repo)
   ├─ FetchCommits() → PostgreSQL (all commits, 90-day window hardcoded)
   ├─ FetchIssues() → PostgreSQL (90-day window)
   ├─ FetchPRs() → PostgreSQL (90-day window)
   └─ (no commit details with files[] fetched)

2. BuildGraph(repoID, repoPath)
   ├─ processCommits() → Commit + Developer nodes
   ├─ processPRs() → PR nodes
   ├─ calculateOwnership() → OWNS edges (should be dynamic)
   └─ linkCommitsToPRs() → IN_PR edges
   (MODIFIED edges NOT created from GitHub data)

3. (TreeSitter parsing only if ProcessRepository() called separately)
```

**Problems:**
- Not PR-centric (fetches commits directly)
- No commit details with `files[]` arrays
- No MODIFIED edges from GitHub data
- TreeSitter and GitHub systems disconnected
- No file list merging
- No time window flags

---

### What Should Happen (`crisk check`)

```
User runs: crisk check src/payment.py

1. Git diff (local) → Get changed lines
2. Query Neo4j:
   a. Ownership (AUTHORED + MODIFIED traversal)
   b. Blast radius (DEPENDS_ON traversal)
   c. Co-change partners (dynamic from MODIFIED + time window)
   d. Incident history (commit message regex)
   e. Recent activity (last 5 commits via MODIFIED)
3. Calculate risk score
4. If high risk:
   - Fetch patches from PostgreSQL for relevant commits
   - Send to LLM with context
5. Display results

Total: 0 API calls, <3 seconds
```

### What Currently Happens

**Command:** `crisk check src/payment.py`

```
1. Git diff (local) → Get changed lines ✅
2. Collector.CollectPhase1Data()
   └─ Only calculates change_complexity ❌
   └─ Returns zero values for all graph queries ❌
3. (Risk scoring not implemented)
4. (LLM analysis not implemented)
5. (Output formatting exists but no data to format)
```

**Problems:**
- Graph queries not executed
- No risk scoring
- No LLM integration for high-risk files
- End-to-end pipeline incomplete

---

## Part 6: Priority Fixes for MVP

### Critical Path (Must Fix for MVP)

**Priority 1: Unified Ingestion (Week 1-2)**

1. Implement PR-centric fetching in `github/fetcher.go`:
   - `FetchMergedPRs(since time.Time)`
   - `FetchPRCommits(prNumber int)`
   - `FetchCommitDetails(sha string)` with `files[]` and `patch`

2. Add CLI flags in `cmd/crisk/init.go`:
   - `--days N` (default: 90)
   - `--all`

3. Create unified ingestion orchestrator in `ingestion/processor.go`:
   - Combine GitHub fetching + TreeSitter parsing
   - Merge file lists (union + path resolution)
   - Call BuildGraph with merged data

4. Update `graph/builder.go`:
   - Add `processCommitFiles()` to create MODIFIED edges from GitHub data
   - Remove `calculateOwnership()` (make dynamic)
   - Process merged file list (TreeSitter + GitHub commits)

**Priority 2: Fix Graph Schema (Week 2)**

1. Remove pre-computed edges:
   - Delete `calculateOwnership()` method
   - Remove all CO_CHANGED edge creation code
   - Delete deprecated methods (AddLayer2CoChangedEdges, etc.)

2. Add MODIFIED edge creation from GitHub commit `files[]` data

3. Clean up references to removed features:
   - Remove Function/Class node references
   - Remove CALLS edge references
   - Remove LINKED_TO edge references

**Priority 3: Implement Queries (Week 3)**

1. Update queries in `risk/queries.go`:
   - Rewrite Co-Change query (dynamic from MODIFIED)
   - Rewrite Incident History query (commit message regex)
   - Rewrite Ownership query (fully dynamic from AUTHORED + MODIFIED)
   - Remove/disable queries that need deferred features (Recent Incidents)

2. Implement query execution in `risk/collector.go`:
   - Execute all 5 core queries
   - Parse results into Phase1Data structure
   - Add error handling

**Priority 4: End-to-End Integration (Week 4)**

1. Complete risk scoring logic
2. Integrate LLM analysis for high-risk files (fetch patches from PostgreSQL)
3. Wire up output formatting
4. Add comprehensive error handling
5. Add progress indicators for long ingestions

---

### Nice-to-Have (Post-MVP)

**Improve TreeSitter Import Detection:**
- Add alias resolution (tsconfig.json, package.json parsing)
- Handle more import patterns
- Improve success rate from 60% to 85%+

**Issue Linking:**
- Ingest GitHub Issues
- Create LINKED_TO edges (Issue → File)
- Implement full incident queries

**Incremental Updates:**
- Delta-only ingestion (track last sync timestamp)
- Reduce API calls by 93% (352 → 25 per run)
- Enable daily/hourly sync

**Advanced Features:**
- Call graph analysis (CALLS edges)
- Function/Class node support
- Branch delta graphs
- Temporal hotspots

---

## Part 7: File-by-File Change Checklist

### Files to Create

- [ ] `cmd/crisk/init.go` - CLI command with `--days` and `--all` flags
- [ ] `internal/ingestion/orchestrator.go` - Unified ingestion flow
- [ ] Tests for new ingestion flow

### Files to Modify (Critical)

- [ ] **`internal/github/fetcher.go`**
  - Add `FetchMergedPRs(since time.Time)`
  - Add `FetchPRCommits(prNumber int)`
  - Update `FetchCommitDetails(sha string)` to include `files[]` and `patch`

- [ ] **`internal/database/staging.go`**
  - Ensure `StoreCommit()` stores full `files[]` and `patch` in `raw_data`
  - Add method to extract `files[]` from stored commits

- [ ] **`internal/graph/builder.go`**
  - Add `processCommitFiles()` to create MODIFIED edges from GitHub data
  - Remove `calculateOwnership()` method entirely
  - Update `BuildGraph()` to accept merged file list
  - Remove CO_CHANGED edge creation

- [ ] **`internal/risk/queries.go`**
  - Update Co-Change query (dynamic from MODIFIED, remove CO_CHANGED edge)
  - Update Incident History query (commit message regex, remove LINKED_TO)
  - Update Ownership query (remove OWNS edge reference)
  - Update Dependency Count query (remove CALLS edge)
  - Remove or disable Recent Incidents query (needs Issue nodes)

- [ ] **`internal/risk/collector.go`**
  - Implement full `CollectPhase1Data()` method
  - Execute all 5 core queries
  - Parse results into Phase1Data
  - Add error handling and logging

### Files to Modify (Cleanup)

- [ ] **`internal/ingestion/processor.go`**
  - Remove `AddLayer2CoChangedEdges()` (deprecated)
  - Remove `AddLayer3IncidentNodes()` (commented out)
  - Clean up references to removed features

- [ ] **`internal/temporal/co_change.go`**
  - Mark entire file as deprecated or remove
  - Co-change now computed dynamically via queries

- [ ] **`internal/graph/builder.go`**
  - Remove deprecated code comments
  - Clean up references to Function/Class nodes

### Files to Review (Lower Priority)

- [ ] **`internal/treesitter/parser.go`**
  - Document current limitations (60% import success rate)
  - Plan improvements for post-MVP (alias resolution)

- [ ] **`internal/risk/agents/`**
  - Review agent implementations
  - Ensure they work with updated Phase1Data

- [ ] **`internal/output/`**
  - Review formatters
  - Ensure they handle new query results

---

## Part 8: Testing Strategy

### Integration Tests Needed

1. **Ingestion Flow:**
   - Test PR-centric fetching with different time windows (7, 30, 90 days, all)
   - Verify file list merging (TreeSitter + GitHub commits)
   - Verify MODIFIED edges created from GitHub `files[]` data

2. **Graph Schema:**
   - Verify no OWNS edges created (ownership is dynamic)
   - Verify no CO_CHANGED edges created (co-change is dynamic)
   - Verify MODIFIED edges have correct properties (additions, deletions, status)

3. **Query Execution:**
   - Test each of 5 core queries independently
   - Verify results match expected schema
   - Test edge cases (no commits, no dependencies, etc.)

4. **End-to-End:**
   - Full flow: `crisk init --days 30` → `crisk check file.py`
   - Verify 0 API calls during `crisk check`
   - Verify results within performance targets (<3s)

### Unit Tests Needed

1. **File list merging** (TreeSitter files + GitHub commit files)
2. **Time window calculation** (--days N, --all)
3. **Query result parsing** (Neo4j results → Phase1Data)
4. **Ownership calculation** (dynamic from edges)
5. **Co-change frequency** (dynamic computation)

---

## Summary

The implementation has a **solid foundation** (Neo4j, PostgreSQL, GitHub API, TreeSitter) but needs **significant refactoring** to align with MVP specs:

**Biggest Gaps:**
1. ❌ No PR-centric ingestion (fetches commits directly)
2. ❌ MODIFIED edges not created from GitHub data
3. ❌ Pre-computed edges (OWNS, CO_CHANGED) contradict dynamic query philosophy
4. ❌ Risk queries defined but not executed (Collector is a stub)
5. ❌ Two disconnected systems (TreeSitter vs GitHub)

**Estimated Effort:**
- Priority 1 fixes: 2-3 weeks (1 developer)
- Full MVP completion: 4-5 weeks
- Testing + polish: 1-2 weeks
- **Total: 5-7 weeks to production-ready MVP**

**Key Success Metrics:**
- `crisk init` completes in 3-4 minutes (90-day window)
- `crisk check` completes in <3 seconds with 0 API calls
- All 5 core queries return valid results
- Graph schema matches documentation exactly

---

**Next Steps:**
1. Review this analysis with team
2. Prioritize fixes (start with Priority 1)
3. Create detailed implementation plan per component
4. Begin refactoring with comprehensive tests
