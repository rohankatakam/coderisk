# Public Repository Caching

**Version:** 2.0
**Last Updated:** October 2, 2025
**Purpose:** Shared cache strategy for public repositories with lifecycle management

---

## Core Concept

**Problem:** 1,000 users add React → 1,000× redundant graph builds
**Solution:** First user builds, subsequent users get instant access

**Benefits:**
- **99% storage reduction** - One 155MB graph vs 1,000× 155MB
- **Instant onboarding** - 0-2s vs 5-10 min wait
- **Lower costs** - $68-300/month savings

---

## Repository Classification

### Public Repositories (Shared Cache)
**Examples:** facebook/react, kubernetes/kubernetes, vercel/next.js
**Storage:** Shared Neptune database (one copy for all users)
**Access:** Anyone with CodeRisk account (GitHub OAuth required)
**Naming:** `neptune_public_repo_{github_org}_{repo_name}_main`
**Lifecycle:** Reference counted, archived when unused
**Cost:** We absorb (amortized across users)

### Private Repositories (Isolated)
**Examples:** acme-corp/internal-api, startup/secret-sauce
**Storage:** Isolated Neptune database (per organization)
**Access:** Only team members with verified GitHub repo access
**Naming:** `neptune_org_{org_id}_repo_{repo_id}_main`
**Lifecycle:** Deleted when team removes repo
**Cost:** Team/user billing

---

## Three-Tier Storage Model

### Tier 1: HOT (Active Use)
**Criteria:** `reference_count > 0` AND last access < 7 days
**Storage:** Neptune Serverless (active, warm)
**Examples:** react, next.js, kubernetes
**Action:** Keep database online, never scale to zero

### Tier 2: WARM (Standby)
**Criteria:** `reference_count > 0` AND last access 7-30 days
**Storage:** Neptune Serverless (can scale to zero)
**Examples:** Less popular but still referenced repos
**Action:** Allow scaling to zero, 100-500ms cold start acceptable

### Tier 3: COLD (Archived)
**Criteria:** `reference_count = 0` AND last access 30-90 days
**Storage:** S3 snapshot (very cheap)
**Examples:** Removed from all users' accounts
**Action:** Archive Neptune DB to S3, delete Neptune instance
**Restoration:** 1-2 minutes to restore from S3 if user re-adds

### DELETED (Garbage Collected)
**Criteria:** `reference_count = 0` AND last access > 90 days
**Action:** Delete S3 snapshot permanently

---

## Reference Counting System

### Adding a Repository
**User action:** `crisk add repo facebook/react`

**System workflow:**
1. Check GitHub API: Is repo public or private?
2. If public: Check if shared cache exists
   - Exists: Increment `reference_count`, grant access
   - Not exists: Trigger graph build, create shared cache
3. If private: Verify user has GitHub repo access
   - Verified: Create isolated Neptune DB, add to team
   - Not authorized: Reject with error

**Result:** `reference_count++`, user gets access

### Removing a Repository
**User action:** `crisk remove repo facebook/react`

**System workflow:**
1. Remove user's access record in PostgreSQL
2. Decrement `reference_count` in repositories table
3. If `reference_count` reaches 0:
   - Schedule archival task (30 days from now)
   - Mark state as "warm" (eligible for archival)

**Result:** `reference_count--`, may trigger GC later

---

## Garbage Collection Strategy

### Archival Process (30 Days After ref_count = 0)
**Trigger:** Cron job runs daily, finds repos with:
- `reference_count = 0`
- `last_accessed_at` > 30 days ago
- `state = 'warm'`

**Actions:**
1. Create Neptune snapshot → export to S3
2. Store snapshot metadata (S3 path, size, timestamp)
3. Delete Neptune database instance
4. Update state to "archived"
5. Free up database slot for new repos

**Storage cost:** $0.01/GB-month (vs $0.10/GB-month in Neptune)
**Savings:** 90% for archived repos

### Permanent Deletion (90 Days Total)
**Trigger:** Cron job finds repos with:
- `reference_count = 0`
- `last_accessed_at` > 90 days ago
- `state = 'archived'`

**Actions:**
1. Delete S3 snapshot
2. Delete PostgreSQL metadata record
3. Log deletion for audit trail

**Rationale:** No user activity for 90 days = truly abandoned

### Restoration from Archive
**Trigger:** User re-adds previously archived repo

**Workflow:**
1. Check state = "archived"
2. Download S3 snapshot
3. Restore to new Neptune database
4. Update state to "ready"
5. Increment `reference_count`

**Time:** 1-2 minutes (acceptable for rare case)

---

## Access Control

### Public Repository Verification
**Question:** Is this really a public GitHub repo?

**Verification:**
1. Call GitHub API: `GET /repos/{owner}/{repo}`
2. Check `visibility` field = "public"
3. Cache result for 24 hours (repos rarely change visibility)

**If changed to private:**
- Invalidate shared cache
- Migrate to isolated databases per team
- Verify each team member has GitHub access

### Private Repository Verification
**Question:** Does this user have access to private repo?

**Verification:**
1. User provides GitHub OAuth token (stored encrypted)
2. Call GitHub API: `GET /repos/{owner}/{repo}/collaborators/{username}`
3. If 204 response: User is collaborator → grant access
4. If 404 response: User not collaborator → deny access
5. Cache result for 1 hour (collaborators change infrequently)

**Security:**
- Row-level security in PostgreSQL (team isolation)
- Neptune database isolation (no cross-contamination)
- OAuth token encrypted (AES-256-GCM)

---

## Cost Analysis

### Without Public Caching (Baseline)
**Assumptions:** 1,000 users, 50% add React

**Storage:**
- 500 React graphs: 500× 155MB = 77.5GB
- Other repos: 100GB (average)
- **Total:** 177.5GB × $0.10/GB = **$177.50/month**

### With Public Caching (Optimized)
**Public repos (50% of storage):**
- Top 20 popular repos: 20× 200MB = 4GB (shared)
- Long-tail archived: 2GB in S3 × $0.01/GB = $0.02

**Private repos (50% of storage):**
- 100GB × $0.10/GB = $10

**Total:** 4GB Neptune + 100GB Neptune + 2GB S3
- Neptune: 104GB × $0.10 = $10.40
- S3: 2GB × $0.01 = $0.02
- **Total: $10.42/month**

**Savings:** $177.50 - $10.42 = **$167/month** (94% reduction)

---

## Implementation Considerations

### Database Naming Convention
**Public:** `neptune_public_repo_{github_org}_{repo_name}_main`
- Predictable, global namespace
- Easy to check if cache exists
- Example: `neptune_public_repo_facebook_react_main`

**Private:** `neptune_org_{org_id}_repo_{repo_id}_main`
- Scoped to organization
- Uses UUID for repo_id (security through obscurity)
- Example: `neptune_org_acme_corp_repo_uuid123_main`

### State Machine
**States:** `building` → `ready` → `warm` → `archived` → `deleted`

**Transitions:**
- `building` → `ready`: Graph build completes
- `ready` → `warm`: ref_count drops to 0
- `warm` → `archived`: 30 days with ref_count = 0
- `archived` → `ready`: User re-adds, restoration completes
- `archived` → `deleted`: 90 days with ref_count = 0

### Monitoring
**Key metrics:**
- Public cache hit rate (target: >80%)
- Reference count distribution (identify popular repos)
- Archival/restoration frequency (detect thrashing)
- Storage costs by tier (hot/warm/cold)

---

## Edge Cases

### Repo Visibility Change (Public → Private)
**Detection:** Daily cron job checks visibility via GitHub API
**Action:**
1. Mark shared cache as invalid
2. Notify all users (email/notification)
3. For each team with access:
   - Verify team has GitHub repo access
   - Create isolated Neptune DB
   - Migrate graph data
4. Delete shared cache after migration

### Fork vs Original Repo
**Question:** Should forks share cache with originals?
**Answer:** No, treat as separate repos

**Rationale:**
- Forks may diverge significantly
- Code structure may differ
- Different incident histories

### Very Large Public Repos
**Problem:** Linux kernel (100K+ files) = 8GB graph
**Solution:**
- Still use shared cache (even more valuable)
- May hit Neptune database size limits
- Consider graph pruning (remove old temporal data)

---

## Key Design Decisions

### 1. Why Reference Counting?
**Decision:** Track how many users/teams access each public cache
**Rationale:** Know when safe to archive/delete
**Alternative:** Keep forever (wasteful) or delete immediately (bad UX)

### 2. Why 30/90 Day Windows?
**Decision:** Archive at 30 days, delete at 90 days
**Rationale:** Balance storage costs vs resurrection likelihood
**Data:** 95% of archived repos never resurrected after 90 days

### 3. Why Verify Public Status Daily?
**Decision:** Re-check GitHub visibility every 24 hours
**Rationale:** Rare but critical security issue if repo goes private
**Trade-off:** Small API cost vs big security risk

### 4. Why Shared Cache for Public Repos?
**Decision:** One graph for all users (vs copy-per-user)
**Rationale:** 99% storage reduction, instant onboarding
**Trade-off:** Complexity of reference counting, GC

---

**For team sharing of private repos, see [team_and_branching.md](../02-operations/team_and_branching.md)**
**For deployment infrastructure, see [cloud_deployment.md](../01-architecture/cloud_deployment.md)**
