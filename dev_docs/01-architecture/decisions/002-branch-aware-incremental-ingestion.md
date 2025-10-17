# ADR 002: Branch-Aware Incremental Ingestion with Language Detection

**Status:** Accepted
**Date:** October 3, 2025
**Decision Makers:** AI Agent (Claude) + User Requirements
**Supersedes:** N/A
**Related:** ADR-001 (Neptune), [team_and_branching.md](../../02-operations/team_and_branching.md), [graph_ontology.md](../graph_ontology.md)

---

## Context

### Current State

**Layer 1 (Code Structure) Implementation:**
- Tree-sitter parsers for JavaScript/TypeScript/Python
- Upfront parsing of entire codebase
- Single graph per repository (main branch only)
- No language detection - all parsers loaded regardless of codebase

**Problems Identified:**

1. **Resource Waste:** Loading all Tree-sitter grammars when repo might only use one language
2. **Branch Blindness:** Graph represents main branch, but developers work on feature branches
3. **Wrong Context:** `crisk check` on feature branch evaluates main branch code (stale)
4. **Full Re-ingestion:** No incremental strategy for branch updates

### User Requirements

From conversation on October 3, 2025:

> "With the GitHub API, we can actually see what languages a codebase uses. Should we decide which language parsers to use instead of running all of them?"

> "How do we deal with branches? A branch can be '2 commits ahead of, 43 commits behind main'. How do we create our 3 layers without reingesting each branch in entirety but ensuring data integrity?"

> "For CodeRisk as the pre-commit stage, it's important we evaluate code based on the current branch the developer is in."

### Constraints from spec.md

- **C-1 (Fast Checks):** Phase 1 checks must complete in <500ms
- **C-6 (Hop Limits):** Max 3-5 graph hops per investigation
- **R-6 (Construction Time):** Graph construction must be <5-10 min for initial, <10s for incremental
- **Scope:** Multi-branch support explicitly in scope (§1.5)

---

## Decision

We will implement **Branch-Aware Incremental Ingestion** with **GitHub Language Detection** using the following architecture:

### Part 1: GitHub Language Detection API

**What:** Query GitHub `/repos/{owner}/{repo}/languages` API during `crisk init`

**Why:**
- Reduce memory footprint (load 1-2 parsers instead of 5)
- Faster initialization (don't initialize unused grammars)
- Accurate reporting (know primary language)

**How:**
```go
// During crisk init
langs, _ := github.GetRepoLanguages(ctx, org, repo)
// Response: {"TypeScript": 532891, "JavaScript": 45231}

// Load only needed parsers
parsers := determineNeededParsers(langs)
// parsers = ["javascript", "typescript"]
```

**Storage:** Store language manifest in PostgreSQL `repositories.languages` (JSONB column)

### Part 2: Branch-Aware Graph Nodes

**What:** Add `branch` and `git_sha` properties to all Layer 1 nodes

**Why:**
- Enable context-specific risk assessment (evaluate actual code developer is changing)
- Support incremental updates (detect what changed via git diff)
- Maintain data integrity (no duplicate nodes, proper version tracking)

**How:**
```cypher
CREATE (f:File {
  path: "src/auth.ts",
  branch: "feature/auth-refactor",  -- NEW
  git_sha: "abc123def456",           -- NEW
  language: "typescript",
  loc: 234,
  last_updated: "2025-10-03T10:30:00Z"
})
```

**Node Uniqueness:** Composite ID = `{path}:{branch}:{git_sha}`

### Part 3: Delta Graph Architecture

**What:** Store feature branches as lightweight delta graphs (5% of base size)

**Why:**
- **Storage:** 92% reduction (2.25GB vs 30GB for team of 10)
- **Speed:** 3-5s to create delta vs 5-10min for full graph
- **Correctness:** Evaluate code in branch context, not stale main

**Structure:**
```
Base Graph (main branch):
├── All files, functions, classes on main
├── Size: ~2GB for 5K file repo
└── Shared by all team members

Delta Graph (feature-branch):
├── Only changed files/functions/classes
├── Size: ~50MB (98% smaller)
└── Shared by developers on same branch

Query-Time Merge:
└── Delta overrides base for matching paths
```

**Query Strategy:**
```cypher
// Priority 1: Check delta
MATCH (f:File {path: $path, branch: $branch})
RETURN f

UNION

// Priority 2: Fallback to base
MATCH (f:File {path: $path, branch: "main"})
WHERE NOT EXISTS {
  MATCH (delta:File {path: $path, branch: $branch})
}
RETURN f
LIMIT 1
```

### Part 4: Incremental Update Strategy

**What:** Use `git diff` to detect changed files, parse only those

**Why:**
- Typical feature branch changes 1-5% of files
- Parsing 50 files takes 3-5s vs 10s for 1,000 files
- Incremental updates <10s (meets spec.md constraint R-6)

**Triggers:**

1. **Initial Delta Creation:** First `crisk check` on feature branch
   ```bash
   merge_base=$(git merge-base main HEAD)
   changed_files=$(git diff --name-only $merge_base HEAD)
   # Parse only $changed_files
   ```

2. **Incremental Update:** New commits pushed
   ```bash
   new_commits=$(git log $last_sha..HEAD)
   changed_files=$(git diff $last_sha HEAD --name-only)
   # Parse only $changed_files, update delta
   ```

3. **Branch Merge:** Delete delta after 7-day grace period

**Storage:**
```sql
CREATE TABLE branch_deltas (
  id SERIAL PRIMARY KEY,
  repo_id INT REFERENCES repositories(id),
  branch_name VARCHAR(255) NOT NULL,
  git_sha VARCHAR(40) NOT NULL,
  merge_base_sha VARCHAR(40) NOT NULL,  -- Divergence point
  created_at TIMESTAMP DEFAULT NOW(),
  last_updated TIMESTAMP DEFAULT NOW(),
  node_count INT,
  status VARCHAR(50),  -- 'active', 'merged', 'deleted'
  UNIQUE(repo_id, branch_name)
);
```

### Part 5: Layer Separation

**Layer 1 (Code Structure):** Branch-specific
- File, Function, Class nodes have `branch` property
- Delta graphs per branch
- Incremental updates

**Layer 2 & 3 (Temporal/Incidents):** Branch-agnostic
- Commit, Developer, Issue, PullRequest nodes: NO branch property
- Stored once, shared across branches
- No duplication needed

**Rationale:** Git history and issues are repo-level facts, not branch-specific

---

## Consequences

### Positive

✅ **Accurate Risk Assessment:** Evaluate code in branch context, not stale main
✅ **Storage Efficiency:** 92% reduction (2.25GB vs 30GB for team of 10)
✅ **Fast Updates:** 3-5s for delta creation, <10s for incremental updates
✅ **Memory Efficiency:** Load only 1-2 parsers instead of 5
✅ **Cost Reduction:** Less parsing = less CPU = lower infrastructure costs
✅ **User Experience:** `crisk check` evaluates actual code developer is changing
✅ **Data Integrity:** No duplicate nodes, proper version tracking via git SHA

### Negative

❌ **Query Complexity:** Every query needs branch filter (mitigated by indexing)
❌ **Implementation Complexity:** More logic for delta management
❌ **Delta Divergence:** Long-lived branches may accumulate large deltas (mitigated by rebase recommendations)
❌ **Merge Conflicts:** Base graph updates while feature branch active (mitigated by invalidation + re-parse)

### Risks

**R1: Query performance degradation**
- **Mitigation:** Index on (`path`, `branch`), aggressive Redis caching (15min TTL)

**R2: Delta graph divergence over time**
- **Mitigation:** Warn if delta >10% of base, suggest rebase

**R3: Branch rename handling**
- **Mitigation:** Document limitation, don't auto-handle (manual update required)

---

## Implementation Plan

### Phase A: Language Detection (Week 1)

**Files:**
- `internal/github/languages.go` - GitHub API client
- `internal/ingestion/processor.go` - Call API during init
- `internal/treesitter/parser.go` - Accept language list param

**Testing:**
- Unit: Mock GitHub API response
- Integration: Parse omnara-ai/omnara with detected languages

**Success Criteria:**
- GitHub API call during `crisk init`
- Only needed parsers loaded
- Language manifest stored in PostgreSQL

### Phase B: Branch-Aware Parsing (Week 2)

**Files:**
- `internal/treesitter/types.go` - Add Branch, GitSHA fields
- `internal/graph/neo4j_backend.go` - Update node creation
- Database migration: Add branch, git_sha columns

**Testing:**
- Unit: Parse same file on different branches
- Integration: Verify node uniqueness

**Success Criteria:**
- Nodes have `branch` and `git_sha` properties
- No duplicate nodes for same file on different branches

### Phase C: Incremental Delta Creation (Week 3-4)

**Files:**
- `internal/git/diff.go` - Git diff extraction
- `internal/ingestion/delta.go` - Delta creation logic
- `cmd/crisk/check.go` - Detect branch, load delta

**Testing:**
- Integration: Create branch, modify files, create delta
- Verify: Delta contains only changed files

**Success Criteria:**
- `git diff` detects changed files
- Only changed files parsed
- Delta graph created in <5s

### Phase D: Query-Time Merge (Week 5)

**Files:**
- `internal/graph/query_builder.go` - Branch-aware queries
- `internal/graph/neo4j_backend.go` - UNION query logic

**Testing:**
- Integration: Query file on feature branch
- Verify: Delta overrides base, fallback works

**Success Criteria:**
- Queries return correct branch version
- Fallback to base works
- Query latency <100ms with branch filter

---

## Alternatives Considered

### Alternative 1: Parse On-Demand (No Upfront Parsing)

**Approach:** Parse files only when `crisk check` is run

**Pros:**
- Faster `crisk init` (no parsing)
- Always up-to-date

**Cons:**
- ❌ Can't meet <500ms check time (spec.md C-1)
- ❌ Can't do cross-file analysis (no dependency graph)
- ❌ Poor UX (user waits on every check)

**Verdict:** Rejected - violates performance constraints

### Alternative 2: Full Graph Per Branch

**Approach:** Create complete graph for every branch

**Pros:**
- Simple query logic
- No merge complexity

**Cons:**
- ❌ 10× storage cost (20GB vs 2.25GB)
- ❌ 10× parsing time per branch (5-10min each)
- ❌ Unsustainable for teams with many branches

**Verdict:** Rejected - violates storage/cost constraints

### Alternative 3: No Branch Support (Main Only)

**Approach:** Only support main branch analysis

**Pros:**
- Simplest implementation
- No delta complexity

**Cons:**
- ❌ Violates spec.md §1.5 (multi-branch explicitly in scope)
- ❌ Poor UX for pre-commit checks (wrong context)
- ❌ Unusable for feature branch development

**Verdict:** Rejected - violates requirements

---

## Validation Criteria

### Functional Requirements

- [ ] GitHub language API called during `crisk init`
- [ ] Only needed Tree-sitter parsers loaded
- [ ] Graph nodes have `branch` and `git_sha` properties
- [ ] Delta graphs created for feature branches
- [ ] `git diff` detects changed files correctly
- [ ] Only changed files re-parsed on incremental update
- [ ] Query-time merge returns correct branch version
- [ ] Fallback to base graph works when file not in delta

### Performance Requirements

- [ ] Language detection API call: <1s
- [ ] Initial delta creation: <5s for 50 files
- [ ] Incremental update: <10s for 10 files
- [ ] Query with branch filter: <100ms
- [ ] Memory usage: <500MB per parser

### Storage Requirements

- [ ] Delta graph size <5% of base graph
- [ ] Total storage for team of 10 with 5 branches: <3GB
- [ ] No duplicate nodes for same file on different branches

### Data Integrity

- [ ] No redundant nodes (proper uniqueness via composite ID)
- [ ] Temporal data (Commits, Issues) not duplicated per branch
- [ ] Git SHA tracking enables cache invalidation

---

## References

- **Research:** [branch_aware_ingestion_strategy.md](../../04-research/active/branch_aware_ingestion_strategy.md)
- **Requirements:** [spec.md §1.5](../../spec.md#15-scope-v10) - Multi-branch support
- **Architecture:** [graph_ontology.md](../graph_ontology.md) - Layer 1 schema
- **Operations:** [team_and_branching.md](../../02-operations/team_and_branching.md) - Branch delta strategy
- **Implementation:** [layer_1_treesitter.md](../../03-implementation/integration_guides/layer_1_treesitter.md)
- **Constraints:** [spec.md §6.2](../../spec.md#62-resource-constraints) - Performance/cost limits
- **GitHub API:** https://docs.github.com/en/rest/repos/repos#list-repository-languages

---

**Accepted by:** AI Agent (Claude) based on user requirements
**Next Actions:**
1. Update spec.md with language detection requirement (§3.X functional requirements)
2. Update graph_ontology.md with branch property specifications
3. Update team_and_branching.md with incremental ingestion details
4. Create implementation tickets for Phases A-D
