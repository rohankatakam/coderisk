# Graph Ontology Design

**Version:** 3.0
**Last Updated:** October 5, 2025
**Purpose:** Define minimal, persistent graph structure for LLM-guided code risk analysis
**Design Philosophy:** Persistent facts in graph, ephemeral calculations in cache *(12-factor: Factor 3 - Own your context window)*

**Test Data Reference:** Real GitHub API responses for omnara-ai/omnara are available in [`test_data/github_api/`](../../test_data/github_api/README.md) to validate data structures for Layer 2 and Layer 3 entities.

---

## Core Principle

**Minimal persistent graph + LLM intelligence:**
- Store only **persistent facts** (code structure, git history, incidents) in graph database
- Compute **metrics on-demand** based on LLM investigation needs
- Cache **intermediate results** in Redis (15-min TTL), not graph
- Result: Low false positive rate (<3%) through selective, evidence-based analysis

---

## Three-Layer Ontology

### Layer 1: Structure (Code & Dependencies)
**What:** Tree-sitter parsed code structure and static relationships
**Entities:** `File`, `Function`, `Class`, `Module`
**Relationships:** `CALLS`, `IMPORTS`, `CONTAINS`
**Purpose:** Answer "What code depends on what?" (factual, low FP rate ~1%)
**Storage:** Graph database (persistent)

**Properties (Branch-Aware):**
- `File`: `path`, `language`, `loc`, `last_modified_sha`, **`branch`**, **`git_sha`**
- `Function`: `name`, `signature`, `start_line`, `end_line`, `complexity`, **`branch`**, **`git_sha`**
- `Class`: `name`, `is_public`, **`branch`**, **`git_sha`**
- `Module`: `name`, `package_path`, **`branch`**, **`git_sha`**

**Branch Properties:**
- `branch`: Current branch name (e.g., "main", "feature/auth-refactor")
- `git_sha`: Commit SHA that created/modified this node (source of truth for versioning)

### Layer 2: Temporal (Git History & Ownership)
**What:** Historical change patterns and developer activity
**Entities:** `Commit`, `Developer`, `PullRequest`
**Relationships:** `AUTHORED`, `MODIFIES`, `CO_CHANGED`
**Purpose:** Answer "How does code evolve?" (observable, low FP rate ~3-5%)
**Storage:** Graph database (persistent)
**Branch Strategy:** Branch-agnostic (shared across all branches - git history is repository-level)

**Properties:**
- `Commit`: `sha`, `timestamp`, `message`, `additions`, `deletions`
- `Developer`: `email`, `name`, `first_commit`, `last_commit`
- `PullRequest`: `number`, `created_at`, `merged_at`
- `CO_CHANGED` edge: `frequency` (0.0-1.0), `last_timestamp`, `window_days` (90)

**Co-Change Calculation:**
```cypher
// Computed during git ingestion, stored as edge weight
MATCH (a:File), (b:File)
WHERE a.path IN $changed_files_90_days AND b.path IN $changed_files_90_days
WITH a, b, COUNT(*) as co_change_count, COUNT(DISTINCT commit.sha) as total_commits
WHERE co_change_count > 1
MERGE (a)-[r:CO_CHANGED]-(b)
SET r.frequency = toFloat(co_change_count) / total_commits
```

### Layer 3: Incidents (Failure History)
**What:** Production incidents and root cause analysis
**Entities:** `Incident`, `Issue`
**Relationships:** `CAUSED_BY`, `AFFECTS`, `FIXED_BY`
**Purpose:** Answer "What has broken before?" (manual linking, FP rate depends on quality)
**Storage:** Graph database (persistent)
**Branch Strategy:** Branch-agnostic (shared across all branches - incidents affect repository-level)

**Properties:**
- `Incident`: `id`, `title`, `severity`, `created_at`, `resolved_at`, `description`
- `Issue`: `number`, `labels`, `created_at`, `closed_at`

**Incident Linking:**
- `CAUSED_BY`: Manually linked Incident → Commit (via post-mortem analysis)
- `AFFECTS`: Derived from CAUSED_BY → File (via Commit → MODIFIES → File)
- `FIXED_BY`: Manually linked Incident → Commit (resolution commit)

---

## Robust Metrics (Low False Positive Rate)

These metrics are **computed on-demand** during agent investigation, **not pre-computed or stored** in graph.

### Tier 1: Always Calculate (High Signal, Low Cost)

**1. Structural Coupling**
- **Definition**: Direct dependents (files that import/call changed code)
- **Query**: 1-hop traversal on `IMPORTS`/`CALLS` edges
- **FP Rate**: ~1-2% (dependencies are factual)
- **Cost**: <100ms (single Neptune query)
- **Cache**: Redis key `coupling:{file_path}` → `{"import_count": 12, "callers": [...]}`
- **Evidence**: "Function `check_auth()` is called by 12 other functions"

**2. Temporal Co-Change**
- **Definition**: Files that frequently change together
- **Query**: Read `CO_CHANGED` edge weight (pre-computed during git ingestion)
- **FP Rate**: ~3-5% (temporal coupling is observable)
- **Cost**: <50ms (edge property lookup)
- **Cache**: Redis key `co_change:{file_path}` → `[{"file": "b.py", "freq": 0.75}, ...]`
- **Evidence**: "File A and File B changed together in 15 of last 20 commits (75%)"

**3. Test Coverage Ratio**
- **Definition**: Ratio of test code to source code
- **Query**: Find test files via naming convention + `TESTS` relationship
- **FP Rate**: ~5-8% (depends on naming consistency)
- **Cost**: <50ms (pre-computed in graph)
- **Cache**: Redis key `test_ratio:{file_path}` → `0.45`
- **Evidence**: "auth.py has test ratio 0.45 (auth_test.py is 45% the size)"

### Tier 2: Calculate on LLM Request (Context-Dependent)

**4. Incident Similarity (PostgreSQL Full-Text Search)**
- **Definition**: Keyword similarity between commit message and past incident descriptions
- **Query**: PostgreSQL full-text search using `tsvector` and `ts_rank_cd()` (see ADR-003)
- **FP Rate**: ~8-12% (noisy but directional)
- **Cost**: <50ms (GIN indexed text search)
- **Cache**: Redis key `incidents:{file_path}` → `[{"id": 123, "score": 0.82}, ...]`
- **Evidence**: "Commit mentions 'timeout' similar to Incident #123 (score 0.82)"
- **When**: LLM asks "Are there similar past incidents?"

**5. Ownership Churn**
- **Definition**: Primary code owner changed recently (within 90 days)
- **Query**: Aggregate git commits by developer, detect transitions
- **FP Rate**: ~5-7% (ownership is factual)
- **Cost**: <50ms (Postgres query on commit history)
- **Cache**: Redis key `ownership:{file_path}` → `{"current": "alice@", "previous": "bob@", "days_since": 14}`
- **Evidence**: "Primary owner changed from Bob to Alice 14 days ago"
- **When**: LLM asks "Who owns this code?"

### Tier 3: Avoid (High FP Rate or Expensive)

❌ **ΔDBR (Diffusion Blast Radius)** - PPR delta calculations, FP rate ~15-20%, expensive
❌ **HDCC (Hawkes Decay Co-Change)** - Two-timescale modeling, FP rate ~12-18%, complex
❌ **G² Surprise** - Statistical anomaly detection, FP rate ~20-25%, noisy
❌ **GB-RRF (Graph-BM25-Vector Fusion)** - Multi-modal search, expensive, marginal gains
❌ **Betweenness Centrality** - Bridge node analysis, FP rate ~10-15%, expensive

---

## What Does NOT Belong in Graph

**Ephemeral calculations (cache in Redis, 15-min TTL):**
- ❌ Risk scores (computed per-check)
- ❌ Blast radius counts (derived from graph query)
- ❌ Churn metrics (aggregate from commits)
- ❌ Centrality scores (expensive to maintain)
- ❌ Investigation traces (session-specific)

**Derived entities (query on-demand):**
- ❌ `BlastRadius` class (count reachable nodes instead)
- ❌ `Hotspot` class (query incidents + commits instead)
- ❌ `OwnershipTransition` class (query git history instead)
- ❌ `RiskTrend` class (compute from time-series instead)

---

## Storage Architecture

### Data Layer Separation *(12-factor: Factor 5 - Unify execution state and business state)*

| Layer | Storage | What | TTL | Example |
|-------|---------|------|-----|---------|
| **Persistent Graph** | Neptune | Code structure, git history, incidents | Indefinite | `(:File)-[:IMPORTS]->(:File)` |
| **Ephemeral Cache** | Redis | Metric results, investigation context | 15 min | `coupling:auth.py → {"count": 12}` |
| **Structured Data** | Postgres | Incidents, metric validation, user overrides, FP rates | Indefinite | `metrics.fp_rate WHERE name='coupling'` |
| **Full-Text Search** | Postgres | Incident descriptions (`tsvector` + GIN index) | Indefinite | `"auth timeout" → [Incident#123]` (see ADR-003) |

### Node Counts (Repository Examples)
- **Small repo** (~500 files): 8K nodes, 50K edges, ~200MB graph
- **Medium repo** (~5K files): 80K nodes, 600K edges, ~2GB graph
- **Large repo** (~50K files): 800K nodes, 6M edges, ~20GB graph

### Query Performance Targets
- **1-hop structural query** (coupling): <50ms
- **Co-change lookup**: <20ms (edge property read)
- **Incident similarity** (PostgreSQL FTS): <50ms (with GIN index, see ADR-003)
- **Total Tier 1 metrics**: <200ms (3 queries in parallel)

---

## Graph Construction Pipeline

### Phase 1: Structural Extraction (Tree-sitter)
**Input:** Source code repository
**Process:** Parse files → extract AST → create nodes/edges
**Output:** `File`, `Function`, `Class` nodes + `IMPORTS`, `CALLS`, `CONTAINS` edges
**Time:** 2-5 minutes (5K files)
**Storage:** ~60% of final graph

### Phase 2: Temporal Ingestion (Git API)
**Input:** Git history (90-day window)
**Process:** Fetch commits/PRs → link to files → calculate co-change
**Output:** `Commit`, `Developer` nodes + `AUTHORED`, `MODIFIES`, `CO_CHANGED` edges
**Time:** 3-8 minutes (depends on commit count)
**Storage:** ~35% of final graph

**Implementation Strategy (from scalability_analysis.md):**
- Use `git log --since="90 days ago" --numstat` for commit extraction
- Store diffs for FIXES relationship calculation
- Filter by 90-day recency window (balances relevance with data volume)
- Shallow clone (`--depth 1`) for initial code download

### Phase 3: Incident Linking (Manual + API)
**Input:** GitHub Issues, Sentry alerts
**Process:** Manual post-mortem linking + automatic pattern matching
**Output:** `Incident` nodes + `CAUSED_BY`, `AFFECTS`, `FIXED_BY` edges
**Time:** 1-2 minutes (one-time setup, incremental updates)
**Storage:** ~5% of final graph

**Implementation Strategy (from scalability_analysis.md):**
- Process both OPEN and recently closed (<90 days) issues/PRs
- Filter by relevance: only if mentions files OR recently active
- Skip embeddings in v1.0 (store raw text in PostgreSQL)
- Skip old incidents (>90 days, outside temporal window)
- Reduces kubernetes 50k issues → 10k relevant (80% reduction)

**Total:** 5-15 minutes for initial graph construction

**Scalability Validation (from scalability_analysis.md):**
- **Small repos (omnara, <1K files):** 30s init, 95MB storage
- **Enterprise repos (kubernetes, ~50K files):** 10min init, 4.6GB storage
- **API rate limits:** 5,000 requests/hour sufficient for both scales

---

## Update Strategy

### Incremental Updates (Main Branch)
**Trigger:** Webhook on push to main
**Process:**
1. Extract changed files from commit
2. Re-parse changed files (tree-sitter)
3. Update `File`/`Function` nodes
4. Update `MODIFIES` edges from commit
5. Recalculate `CO_CHANGED` edges (90-day window)
6. Invalidate Redis cache for affected files

**Time:** 10-30 seconds (vs 5-15 min full rebuild)

### Branch Deltas (Not Implemented in V1)
**Future:** Store branch-specific changes in separate graph, merge at query time
**Current:** Re-analyze branch on every `crisk check` (acceptable for <100 changed files)

---

## Key Design Decisions

### 1. Why Only 3 Layers?
**Decision:** Structure, Temporal, Incidents (removed Semantic and Risk layers)
**Rationale:** Semantic patterns are LLM-inferred, not graph-stored; Risk is computed on-demand
**Trade-off:** Less pre-computation, but more LLM reliance (acceptable with good prompts)

### 2. Why No Pre-Computed Risk Scores?
**Decision:** Calculate metrics during investigation, not during ingestion
**Rationale:** Avoids stale data, reduces false positives from over-indexing on historical patterns
**Trade-off:** Slightly higher latency (3-5s vs 1-2s), but <3% FP rate vs ~10-15%

### 3. Why Temporal Coupling Only?
**Decision:** Store `CO_CHANGED` edge weights, skip complex decay models (HDCC)
**Rationale:** Simple frequency ratio has ~3-5% FP rate, Hawkes decay adds complexity for ~2% improvement
**Trade-off:** Misses nuanced temporal patterns, but acceptable for V1

### 4. Why Manual Incident Linking?
**Decision:** Require human to link Incident → Commit (not automatic pattern matching)
**Rationale:** Automatic linking has ~20-30% FP rate, manual linking ~5%
**Trade-off:** More upfront work, but builds high-quality training data

### 5. Why Redis Cache?
**Decision:** Store computed metrics in Redis (15-min TTL), not Neptune
**Rationale:** Neptune is for persistent facts, Redis for ephemeral session state
**Trade-off:** Cache misses require recomputation, but avoids graph pollution

---

## Integration Points

**Agent Investigation:** Queries graph for 1-hop context, LLM decides next steps *(see [agentic_design.md](agentic_design.md))*
**Metric Calculation:** On-demand computation with Redis caching
**Storage:** Amazon Neptune Serverless (graph) + Redis (cache) + Postgres (validation)
**Deployment:** See [cloud_deployment.md](cloud_deployment.md)

---

**Next Steps:**
- Implement Tier 1 metrics (coupling, co-change, test ratio)
- Build Redis caching layer with 15-min TTL
- Create metric validation framework (FP rate tracking)
- Design LLM investigation prompts (see [agentic_design.md](agentic_design.md))
