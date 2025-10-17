# Team Sharing & Multi-Branch Strategy

**Version:** 2.0
**Last Updated:** October 2, 2025
**Purpose:** How teams share graphs and handle multiple branches efficiently

---

## Core Concept

**One graph per repository** (not per user, not per branch)
- Team members share the same base graph (main branch)
- Feature branches stored as lightweight deltas (98% smaller)
- Deltas merged with base at query time

**Benefits:**
- **90% cost reduction** - One 2GB graph vs 10× 2GB per team member
- **Consistent assessment** - Same graph = same risk scores
- **Faster investigations** - Shared cache, no cold starts

---

## Three-Tier Architecture

### Tier 1: Base Graph (Main Branch)
**What:** Full-fidelity graph for main/master branch
**Size:** ~2GB for 5K file repository
**Lifecycle:** Always warm, never scales to zero
**Shared by:** All team members
**Updates:** Incremental on push to main (10-30 seconds)

**Contains:**
- Complete code structure (files, functions, classes)
- Full temporal history (commits, PRs, issues)
- All relationships (calls, co-changes, ownership)
- Pre-computed metrics (centrality, blast radius)

### Tier 2: Branch Deltas (Feature Branches)
**What:** Lightweight graph containing only changes vs main
**Size:** ~10-50MB (98% smaller than full graph)
**Lifecycle:** Created on-demand, deleted after merge
**Shared by:** Team members on same branch
**State:** Cold (scales to zero when unused)

**Contains:**
- Changed files only (git diff vs main)
- Modified functions and classes
- Updated call graph edges (affected relationships)
- **Inherits** from main: temporal data, incidents, ownership

### Tier 3: Merged View (Virtual)
**What:** Runtime merge of base + delta via federated query
**Latency:** +100ms vs base-only query
**Cached:** Redis, 15-min TTL
**Strategy:** Delta overrides base for changed entities

**Query logic:**
1. Check if entity exists in branch delta → use delta version
2. If not in delta → use base graph version
3. Merge relationships on-the-fly
4. Cache result for 15 minutes

---

## Storage Efficiency

### Without Team Sharing (Naive Approach)
**Team of 10 developers, 5 active branches:**
- 10 base graphs: 10× 2GB = 20GB
- 5 branch graphs: 5× 2GB = 10GB
- **Total: 30GB per repository**

### With Team Sharing (Our Approach)
**Same team:**
- 1 base graph (shared): 2GB
- 5 branch deltas: 5× 50MB = 250MB
- **Total: 2.25GB per repository**
- **Reduction: 92%**

---

## Branch Lifecycle Management

### Branch Delta Creation (Incremental Strategy)
**Trigger:** First `crisk check` on feature branch
**Process:**
1. Detect current branch via git
2. Check if delta exists for this branch
3. If not: Use **incremental delta strategy**:
   - Find merge base: `git merge-base main HEAD`
   - Get changed files: `git diff --name-only <merge_base> HEAD`
   - Parse **only changed files** (50 vs 1000 files)
   - Use `git_sha` property to avoid redundant nodes
4. Create delta graph (changed nodes only)
5. Store in separate Neptune database

**Time:** 3-5 seconds (vs 5-10 min for full graph)
**Efficiency:** Parse 50 changed files instead of 1000+ total files (95% reduction)

### Branch Delta Update (Incremental)
**Trigger:** New commits pushed to branch
**Process:**
1. Webhook receives push event
2. Extract **incremental diff** via git:
   - `git diff --name-only <previous_sha> <new_sha>`
   - Parse only newly changed files (5-10 files typically)
3. Update delta graph (add/modify/delete nodes)
4. Update `git_sha` properties on affected nodes
5. Invalidate related Redis caches

**Time:** 2-5 seconds (incremental diff vs full branch scan)
**Efficiency:** Parse 5-10 newly changed files instead of all 50+ branch changes

### Branch Delta Deletion
**Trigger:** Branch merged or deleted
**Process:**
1. Mark delta as "archived" (not immediately deleted)
2. After 7 days: Delete Neptune database
3. Clean up metadata in PostgreSQL

**Rationale:** 7-day retention allows resurrection if branch undeleted

---

## Team Access Control

### Repository-Level Permissions
**Stored in PostgreSQL:**
- `repositories` table: org, repo name, Neptune DB name, visibility
- `repository_access` table: user_id, repo_id, role (owner/admin/member)
- `teams` table: team members and billing info

**Access verification:**
1. User runs `crisk check`
2. CLI reads git remote URL → identifies repo
3. API call to check if user has access
4. For private repos: verify GitHub OAuth token
5. Grant access to Neptune database

### GitHub OAuth Integration
**For private repositories:**
- User authenticates via GitHub OAuth
- Store encrypted OAuth token
- Verify collaborator status via GitHub API before granting graph access
- Prevents unauthorized access to private repo graphs

**For public repositories:**
- Shared cache (no access control needed)
- Read-only access for all authenticated users

---

## Caching Strategy

### Layer 1: Redis (Investigation Cache)
**What:** Cached investigation results
**TTL:** 15 minutes
**Key format:** `investigation:{repo_id}:{file_hash}:{branch}`
**Hit rate:** 35% (same dev, same diff within 15 min)

### Layer 2: Neptune Query Cache
**What:** Built-in Neptune query result caching
**TTL:** Managed by Neptune (typically 60 minutes)
**Hit rate:** 55% (team members querying similar regions)

### Layer 3: Materialized Views
**What:** Pre-computed graph metrics
**Update:** Daily at 2 AM UTC
**Hit rate:** 95% (metrics change slowly)

**Examples:**
- Function complexity (changes only when function modified)
- File blast radius (changes only when call graph modified)
- Ownership information (changes only when commits added)

### Cache Invalidation
**Smart invalidation on branch update:**
1. Identify changed files from commits
2. Find investigations touching those files
3. Invalidate only affected investigations (surgical)
4. Keep unaffected investigations cached

**Result:** 60-70% cache retention after branch update (vs 0% naive invalidation)

---

## Performance Characteristics

### Latency Targets

| Operation | Target | Typical | Notes |
|-----------|--------|---------|-------|
| Main branch check | <2s | 1.0s | Base graph always warm |
| Feature branch check (warm) | <3s | 1.1s | Base + delta merge, cached |
| Feature branch check (cold) | <5s | 4.1s | Includes delta creation |
| Branch switch | <1s | 0.5s | Delta swap in memory |

### Cost Analysis (1,000 Users, 100 Repos)

**Neptune Storage:**
- Base graphs: 100× 2GB = 200GB × $0.10/GB = $20/month
- Branch deltas: 200× 50MB = 10GB × $0.10/GB = $1/month
- **Total storage: $21/month**

**Neptune Compute:**
- Peak hours: 16 NCUs × 176h = 2,816 NCU-hours × $0.12 = $338/month
- Off-peak: 4 NCUs × 352h = 1,408 NCU-hours × $0.12 = $169/month
- Idle: 0.5 NCUs × 192h = 96 NCU-hours × $0.12 = $12/month
- **Total compute: $519/month**

**Total Neptune cost: $540/month ($0.54/user)**

---

## Key Design Decisions

### 1. Why One Graph Per Repo (Not Per User)?
**Decision:** Team shares single base graph
**Rationale:**
- 90% cost reduction
- Consistent risk scores across team
- Shared cache = faster investigations

**Trade-off:** Requires access control layer

### 2. Why Branch Deltas (Not Full Graphs)?
**Decision:** Store only changes vs main, merge at query time
**Rationale:**
- 98% storage reduction (10-50MB vs 2GB)
- Faster branch creation (3-5s vs 5-10 min)
- Most entities unchanged between branches

**Trade-off:** +100ms query latency (acceptable)

### 3. Why 7-Day Delta Retention?
**Decision:** Keep deleted branch deltas for 7 days
**Rationale:**
- Allows branch resurrection
- Debugging historical issues
- Minimal storage cost

**Trade-off:** Slight storage overhead (acceptable)

### 4. Why Three-Layer Cache?
**Decision:** Redis (15min) + Neptune (60min) + Materialized views (daily)
**Rationale:**
- Different TTLs for different data freshness needs
- 90%+ cache hit rate overall
- Sub-2s latency for cached investigations

**Trade-off:** Cache invalidation complexity

---

## Integration Points

**Agent Investigation:** Queries merged view (base + delta)
**Settings Portal:** Manages team membership and access
**Webhooks:** Trigger delta updates on push
**GitHub OAuth:** Verifies private repo access

---

**For graph structure, see [graph_ontology.md](../01-architecture/graph_ontology.md)**
**For public cache strategy, see [public_caching.md](public_caching.md)**
