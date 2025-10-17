# Graph Ontology Design

**Version:** 4.0 (MVP)
**Last Updated:** October 17, 2025
**Purpose:** Define minimal, persistent graph structure for LLM-guided code risk analysis
**Deployment:** Local Neo4j in Docker, no cloud infrastructure

---

## Core Principle

**Minimal persistent graph + LLM intelligence:**
- Store only **persistent facts** (code structure, git history, incidents) in Neo4j
- Compute **metrics on-demand** based on LLM investigation needs
- Cache **intermediate results** locally (15-min TTL), not in graph
- Result: Low false positive rate (<3%) through selective, evidence-based analysis

---

## Three-Layer Ontology

### Layer 1: Structure (Code & Dependencies)

**What:** Tree-sitter parsed code structure and static relationships

**Entities:** `File`, `Function`, `Class`, `Module`

**Relationships:** `CALLS`, `IMPORTS`, `CONTAINS`

**Purpose:** Answer "What code depends on what?" (factual, low FP rate ~1%)

**Storage:** Neo4j graph database (persistent)

**Properties (Branch-Aware):**
- `File`: path, language, loc (lines of code), last_modified_sha, branch, git_sha
- `Function`: name, signature, start_line, end_line, complexity, branch, git_sha
- `Class`: name, is_public, branch, git_sha
- `Module`: name, package_path, branch, git_sha

**Branch Properties:**
- `branch`: Current branch name (e.g., "main", "feature/auth-refactor")
- `git_sha`: Commit SHA that created/modified this node (source of truth for versioning)

**Why branch-aware:** Allows analyzing feature branches before merge, comparing risk across branches.

### Layer 2: Temporal (Git History & Ownership)

**What:** Historical change patterns and developer activity

**Entities:** `Commit`, `Developer`, `PullRequest`

**Relationships:** `AUTHORED`, `MODIFIES`, `CO_CHANGED`

**Purpose:** Answer "How does code evolve?" (observable, low FP rate ~3-5%)

**Storage:** Neo4j graph database (persistent)

**Branch Strategy:** Branch-agnostic (shared across all branches - git history is repository-level)

**Properties:**
- `Commit`: sha, timestamp, message, additions, deletions
- `Developer`: email, name, first_commit, last_commit
- `PullRequest`: number, created_at, merged_at
- `CO_CHANGED` edge: frequency (0.0-1.0), last_timestamp, window_days (90)

**Co-Change Calculation:**
Computed during git ingestion and stored as edge weight. The frequency represents how often two files change together in the same commit within the 90-day window.

Example: If files A and B changed together in 15 out of 20 commits that touched either file, the CO_CHANGED edge has frequency = 0.75 (75%).

### Layer 3: Incidents (Failure History)

**What:** Production incidents and root cause analysis

**Entities:** `Incident`, `Issue`

**Relationships:** `CAUSED_BY`, `AFFECTS`, `FIXED_BY`

**Purpose:** Answer "What has broken before?" (manual linking, FP rate depends on quality)

**Storage:** Neo4j graph database (persistent)

**Branch Strategy:** Branch-agnostic (shared across all branches - incidents affect repository-level)

**Properties:**
- `Incident`: id, title, severity, created_at, resolved_at, description
- `Issue`: number, labels, created_at, closed_at

**Incident Linking:**
- `CAUSED_BY`: Manually linked Incident → Commit (via post-mortem analysis using `crisk incident link`)
- `AFFECTS`: Derived from CAUSED_BY → File (via Commit → MODIFIES → File chain)
- `FIXED_BY`: Manually linked Incident → Commit (resolution commit)

---

## Robust Metrics (Low False Positive Rate)

These metrics are **computed on-demand** during agent investigation, **not pre-computed or stored** in graph.

### Tier 1: Always Calculate (High Signal, Low Cost)

**1. Structural Coupling**
- **Definition**: Direct dependents (files that import/call changed code)
- **Query**: 1-hop traversal on IMPORTS/CALLS edges in Neo4j
- **FP Rate**: ~1-2% (dependencies are factual)
- **Cost**: <100ms
- **Evidence**: "Function `check_auth()` is called by 12 other functions"

**2. Temporal Co-Change**
- **Definition**: Files that frequently change together
- **Query**: Read CO_CHANGED edge weight (pre-computed during git ingestion)
- **FP Rate**: ~3-5% (temporal coupling is observable)
- **Cost**: <50ms (edge property lookup)
- **Evidence**: "File A and File B changed together in 15 of last 20 commits (75%)"

**3. Test Coverage Ratio**
- **Definition**: Ratio of test code to source code
- **Query**: Find test files via naming convention + TESTS relationship
- **FP Rate**: ~5-8% (depends on naming consistency)
- **Cost**: <50ms
- **Evidence**: "auth.py has test ratio 0.45 (auth_test.py is 45% the size)"

### Tier 2: Calculate on LLM Request (Context-Dependent)

**4. Incident Similarity**
- **Definition**: Keyword similarity between commit message and past incident descriptions
- **Query**: Simple text search on incident descriptions stored in Neo4j
- **FP Rate**: ~8-12% (noisy but directional)
- **Cost**: <50ms
- **Evidence**: "Commit mentions 'timeout' similar to Incident #123"
- **When**: LLM asks "Are there similar past incidents?"

**5. Ownership Churn**
- **Definition**: Primary code owner changed recently (within 90 days)
- **Query**: Aggregate git commits by developer from Neo4j, detect transitions
- **FP Rate**: ~5-7% (ownership is factual)
- **Cost**: <50ms
- **Evidence**: "Primary owner changed from Bob to Alice 14 days ago"
- **When**: LLM asks "Who owns this code?"

### Tier 3: Avoid (High FP Rate or Expensive)

These complex metrics are explicitly **not included in MVP**:

❌ **ΔDBR (Diffusion Blast Radius)** - PPR delta calculations, FP rate ~15-20%, expensive
❌ **HDCC (Hawkes Decay Co-Change)** - Two-timescale modeling, FP rate ~12-18%, complex
❌ **G² Surprise** - Statistical anomaly detection, FP rate ~20-25%, noisy
❌ **GB-RRF (Graph-BM25-Vector Fusion)** - Multi-modal search, expensive, marginal gains
❌ **Betweenness Centrality** - Bridge node analysis, FP rate ~10-15%, expensive

**MVP Philosophy:** 5 simple, robust metrics beat 9 complex, noisy ones.

---

## What Does NOT Belong in Graph

**Ephemeral calculations (computed on-demand, optionally cached locally):**
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

**Design Philosophy:** If it can be calculated from the graph in <100ms, don't store it. Keep the graph clean.

---

## Storage Architecture (Local MVP)

### Data Layer Separation

| Layer | Storage | What | Persistence | Example |
|-------|---------|------|-------------|---------|
| **Persistent Graph** | Neo4j (Docker) | Code structure, git history, incidents | Indefinite | `(:File)-[:IMPORTS]->(:File)` |
| **Ephemeral Cache** | Local filesystem | Metric results, investigation context | 15 min TTL | `coupling:auth.py → {"count": 12}` |
| **Validation Data** | SQLite (Docker) | Metric validation, user feedback, FP rates | Indefinite | `metrics.fp_rate WHERE name='coupling'` |

**MVP Simplification:**
- Single Neo4j container for graph
- Simple filesystem cache (or in-memory) for 15-minute TTL
- SQLite for validation data (no separate PostgreSQL)
- Text search on incident descriptions using Neo4j full-text indexes

### Expected Repository Sizes

| Repo Size | Files | Nodes | Edges | Storage | Init Time |
|-----------|-------|-------|-------|---------|-----------|
| **Small** | ~500 | 8K | 50K | ~200MB | 30-60s |
| **Medium** | ~5K | 80K | 600K | ~2GB | 2-5 min |
| **Large** | ~10K+ | 150K+ | 1M+ | ~5GB | 5-10 min |

**MVP Target:** Optimized for small to medium repos (most solo/small team projects).

### Query Performance Targets

- **1-hop structural query** (coupling): <50ms
- **Co-change lookup**: <20ms (edge property read)
- **Incident similarity** (text search): <50ms
- **Total Tier 1 metrics**: <200ms (3 queries in parallel)

---

## Graph Construction Pipeline

### Phase 1: Structural Extraction (Tree-sitter)

**Input:** Local source code repository

**Process:** Parse files → extract AST → create nodes/edges in Neo4j

**Output:** `File`, `Function`, `Class` nodes + `IMPORTS`, `CALLS`, `CONTAINS` edges

**Time:** 1-5 minutes (depends on repo size)

**Storage:** ~60% of final graph

**Implementation:** Tree-sitter parsers for major languages (Python, JavaScript, TypeScript, Go, Java, etc.)

### Phase 2: Temporal Ingestion (Git History)

**Input:** Local git history (90-day window)

**Process:** Parse commits → link to files → calculate co-change frequencies

**Output:** `Commit`, `Developer` nodes + `AUTHORED`, `MODIFIES`, `CO_CHANGED` edges

**Time:** 1-3 minutes (depends on commit count)

**Storage:** ~35% of final graph

**Implementation:**
- Use `git log --since="90 days ago" --numstat` for commit extraction
- Filter by 90-day recency window (balances relevance with data volume)
- Calculate CO_CHANGED edge weights during this phase

### Phase 3: Incident Linking (Manual)

**Input:** GitHub Issues, manual incident reports

**Process:** User manually links incidents to commits using CLI

**Output:** `Incident` nodes + `CAUSED_BY`, `AFFECTS`, `FIXED_BY` edges

**Time:** Ongoing (user adds as incidents occur)

**Storage:** ~5% of final graph

**Implementation:**
- `crisk incident create "title" "description" --severity critical`
- `crisk incident link <incident-id> <commit-sha> --type caused_by`
- Optionally import GitHub Issues with `crisk incident import-github`

**Total Init Time:** 2-10 minutes for typical repos (depending on size)

---

## Update Strategy

### Incremental Updates (Main Branch)

**Trigger:** User runs `crisk check` (local pre-commit hook)

**Process:**
1. Extract changed files from git diff
2. Re-parse changed files with tree-sitter
3. Update `File`/`Function` nodes in Neo4j
4. Create new `Commit` node and `MODIFIES` edges
5. Recalculate `CO_CHANGED` edges for affected files (90-day window)
6. Invalidate local cache for affected files

**Time:** 5-15 seconds (vs 2-10 min full rebuild)

**Optimization:** Only update nodes/edges directly affected by the change.

### Branch Analysis (Feature Branches)

**Approach:** Re-analyze branch on every `crisk check` (acceptable for <100 changed files)

**Why:** Simpler than maintaining separate branch graphs, sufficient for MVP.

**Future:** Store branch-specific changes in separate graph namespace, merge at query time.

---

## Key Design Decisions

### 1. Why Only 3 Layers?

**Decision:** Structure, Temporal, Incidents (removed Semantic and Risk layers from original design)

**Rationale:**
- Semantic patterns are LLM-inferred, not graph-stored
- Risk is computed on-demand, not pre-computed

**Trade-off:** Less pre-computation, but more LLM reliance (acceptable with good prompts)

### 2. Why No Pre-Computed Risk Scores?

**Decision:** Calculate metrics during investigation, not during ingestion

**Rationale:**
- Avoids stale data (risk context changes with each commit)
- Reduces false positives from over-indexing on historical patterns

**Trade-off:** Slightly higher latency (3-5s vs 1-2s), but <3% FP rate vs ~10-15%

### 3. Why Temporal Coupling Only?

**Decision:** Store simple `CO_CHANGED` edge weights, skip complex decay models (HDCC)

**Rationale:** Simple frequency ratio has ~3-5% FP rate, complex decay models add 2% improvement at 10x implementation cost

**Trade-off:** Misses nuanced temporal patterns, but acceptable for MVP

### 4. Why Manual Incident Linking?

**Decision:** Require human to link Incident → Commit (not automatic pattern matching)

**Rationale:**
- Automatic linking has ~20-30% FP rate
- Manual linking has ~5% FP rate
- Builds high-quality training data for future ML models

**Trade-off:** More upfront work, but better accuracy

### 5. Why Local Neo4j (Not Cloud)?

**Decision:** Run Neo4j in Docker container locally, not AWS Neptune

**Rationale:**
- Zero infrastructure cost (free for BYOK model)
- No network latency (graph queries are local)
- Full control over data (no cloud vendor lock-in)
- Sufficient for small/medium repos (MVP target)

**Trade-off:** Limited to single machine resources, but acceptable for target users

---

## Integration Points

**Agent Investigation:** Queries Neo4j for 1-hop context, LLM decides next steps
*(see [agentic_design.md](agentic_design.md))*

**Metric Calculation:** On-demand computation with optional local caching

**Storage:** Local Neo4j (Docker) + SQLite (validation) + filesystem (cache)

**User Interface:** CLI commands (`crisk init`, `crisk check`, `crisk incident`)
*(see [../00-product/developer_experience.md](../00-product/developer_experience.md))*

---

## MVP Implementation Checklist

**Phase 1: Core Graph (Week 1-2)**
- [ ] Neo4j Docker setup with volume persistence
- [ ] Tree-sitter integration for major languages
- [ ] File/Function/Class node creation
- [ ] IMPORTS/CALLS/CONTAINS edge creation

**Phase 2: Git Integration (Week 2-3)**
- [ ] Git log parsing (90-day window)
- [ ] Commit/Developer node creation
- [ ] MODIFIES edge creation
- [ ] CO_CHANGED edge weight calculation

**Phase 3: Incident Tracking (Week 3-4)**
- [ ] Incident node creation CLI
- [ ] Manual linking CLI (CAUSED_BY, FIXED_BY)
- [ ] AFFECTS edge derivation
- [ ] GitHub Issue import (optional)

**Phase 4: Incremental Updates (Week 4)**
- [ ] Git diff extraction
- [ ] Selective node/edge updates
- [ ] Cache invalidation
- [ ] Performance optimization (<15s updates)

---

## Related Documentation

- **[agentic_design.md](agentic_design.md)** - How the LLM investigates using this graph
- **[risk_assessment_methodology.md](risk_assessment_methodology.md)** - How metrics are calculated and validated
- **[../00-product/mvp_vision.md](../00-product/mvp_vision.md)** - Overall MVP scope and goals
- **[../00-product/developer_experience.md](../00-product/developer_experience.md)** - CLI workflow and user experience
