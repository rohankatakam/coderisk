# ADR 006: Multi-Tenant Neptune with Public Repository Caching

**Date:** 2025-10-12
**Status:** Accepted
**Deciders:** Architecture team
**Tags:** scalability, multi-tenancy, cost-optimization
**Supersedes:** Partial aspects of ADR-004 (deployment strategy)

---

## Context

We need a cloud graph database strategy that supports **multi-tenant SaaS** deployment with:

1. **Public repository caching** - 1,000 users adding `facebook/react` should store ONE graph, not 1,000 copies
2. **Private repository isolation** - Each organization's private repos must be completely isolated
3. **Branch-aware workflows** - Developers run `crisk check` on feature branches (not just main)
4. **Scalable from day 1** - Architecture must handle 1,000+ users without major redesign
5. **Cost-efficient** - Avoid per-user graph duplication (99% storage waste)

**Current local implementation:**
- Docker Compose with Neo4j Community Edition
- Single-user workflow (one graph per repository)
- No multi-tenancy, no access control
- Works perfectly for local development

**Cloud deployment challenge:**
- How do we handle 1,000 users adding the same public repo?
- How do we isolate private repos between organizations?
- How do we support feature branch analysis without duplicating 2GB graphs?
- How do we prevent cost explosion?

---

## Decision

**We will implement a single AWS Neptune Serverless cluster with multi-tenant database namespacing, public repository reference counting, and branch delta architecture.**

### Core Architecture

```
SINGLE NEPTUNE SERVERLESS CLUSTER (auto-scales 0.5-128 NCUs)
├─ Public repo graphs (shared, reference-counted)
│  ├─ neptune_public_repo_facebook_react_main
│  ├─ neptune_public_repo_vercel_nextjs_main
│  └─ ... (~100-200 popular repos)
│
├─ Private org graphs (isolated per organization)
│  ├─ neptune_org_{org_id}_repo_{uuid}_main
│  └─ ... (one per private repo per org)
│
└─ Branch deltas (lightweight, ephemeral)
   ├─ neptune_public_repo_facebook_react_branch_{sha}
   └─ neptune_org_{org_id}_repo_{uuid}_branch_{sha}
```

### Key Mechanisms

1. **Database Namespacing** - Neptune supports multiple named databases within a single cluster
2. **Reference Counting** - Track how many users/orgs use each public cache (PostgreSQL metadata)
3. **Garbage Collection** - Archive unused public caches to S3 after 30 days (ref_count = 0)
4. **Branch Deltas** - Store only file changes vs main branch (50MB vs 2GB full graph)
5. **Access Control** - PostgreSQL row-level security enforces private repo isolation

---

## Options Considered

### Option 1: One Neptune Cluster Per User ❌

**Approach:** Each user gets their own Neptune cluster

**Pros:**
- ✅ Perfect isolation
- ✅ Simple access control

**Cons:**
- ❌ **Cost explosion:** 1,000 users × $450/month = $450,000/month
- ❌ AWS account limits (25 Neptune clusters per region)
- ❌ Management overhead (1,000 clusters to monitor)
- ❌ No public repository sharing

**Verdict:** Economically infeasible

---

### Option 2: Neo4j Aura with Multi-Tenancy ❌

**Approach:** Use Neo4j Aura's multi-database support

**Pros:**
- ✅ Same Cypher queries as local
- ✅ Multi-database support built-in
- ✅ Mature product

**Cons:**
- ❌ **Expensive at scale:** $2,000/month @ 1K users (from ADR-004)
- ❌ No true serverless (always-on pricing)
- ❌ Vertical scaling only (can't distribute load)
- ❌ Free tier too limited (50k nodes = ~10 small repos)

**Cost over 24 months:** $48,000 (vs $8,000 with Neptune)

**Verdict:** 6x more expensive than Neptune at scale

---

### Option 3: Multi-Tenant Neptune with Public Caching ✅ **CHOSEN**

**Approach:** Single Neptune cluster, database namespacing, reference-counted public caches

**Pros:**
- ✅ **99% storage savings** - One copy of `react` for 1,000 users
- ✅ **True serverless** - Scales to 0.5 NCU when idle ($4/month minimum)
- ✅ **Cost-efficient** - $317/month @ 1K users vs $2,000 with Neo4j
- ✅ **AWS-native** - IAM integration, VPC security, CloudWatch monitoring
- ✅ **Scales to millions** - No theoretical limits

**Cons:**
- ⚠️ **Query conversion** - Cypher → Gremlin (70-110 hours one-time)
- ⚠️ **Multi-tenancy complexity** - Need access control layer in app
- ⚠️ **Reference counting logic** - Must track public cache usage

**Cost over 24 months:** $8,000 (storage + compute)

**Verdict:** Best economics + scalability

---

## Rationale

### 1. Cost Efficiency (Primary Driver)

**Without public caching (naive approach):**
```
1,000 users each add facebook/react (2GB graph)
→ 1,000 × 2GB = 2,000GB storage
→ 2,000GB × $0.10/GB = $200/month for ONE repo
→ 10 popular repos = $2,000/month storage alone
```

**With public caching (our approach):**
```
First user adds facebook/react → Build once (2GB)
Next 999 users → Increment ref_count, instant access
→ 1 × 2GB = 2GB storage (shared)
→ 2GB × $0.10/GB = $0.20/month for 1,000 users
→ 100 popular repos = $20/month

Savings: $2,000 - $20 = $1,980/month (99% reduction)
```

### 2. Developer Experience

**Local workflow (current):**
```bash
cd my-repo
git checkout feature-branch
crisk check  # Analyzes feature branch changes
```

**Cloud workflow (with branch deltas):**
```bash
cd my-repo
git checkout feature-branch
crisk check  # Same UX, but:
  1. Queries main branch base graph (2GB, shared)
  2. Queries feature branch delta (50MB, ephemeral)
  3. Merges results at query time (<100ms overhead)
```

**Key insight:** Users don't care about infrastructure - they just want `crisk check` to work on any branch instantly.

### 3. Scalability

**Growth trajectory:**
```
Month 1-6:   100 users   → $50/month Neptune
Month 6-12:  500 users   → $150/month Neptune
Month 12-18: 1,000 users → $317/month Neptune
Month 18-24: 5,000 users → $800/month Neptune

Per-user cost decreases with scale:
100 users:   $0.50/user/month
1,000 users: $0.32/user/month
5,000 users: $0.16/user/month
```

**Neptune auto-scaling:**
- Idle (weekends): 0.5 NCUs = $4/month
- Normal (weekdays): 4-8 NCUs = $200-400/month
- Peak (deployments): 16-32 NCUs for 1 hour = $2-4
- **No manual intervention required**

### 4. Multi-Tenancy Best Practices

**Why database namespacing over separate clusters:**

| Aspect | Separate Clusters | Single Cluster + Namespacing |
|--------|------------------|------------------------------|
| **Isolation** | Physical | Logical (sufficient for SaaS) |
| **Cost** | $450 × N clusters | $450 for ALL tenants |
| **Management** | N × complexity | 1 × complexity |
| **Backup** | N separate backups | 1 backup (all tenants) |
| **Monitoring** | N dashboards | 1 dashboard |
| **Scaling** | Manual per cluster | Automatic for all |

**Security:** PostgreSQL row-level security + IAM policies = strong multi-tenant isolation

---

## Consequences

### Positive

✅ **Massive cost savings** - 99% storage reduction via public caching
✅ **Instant onboarding** - Popular repos already cached (0-2s vs 5-10 min build)
✅ **Scalable from day 1** - Architecture handles 10,000+ users without redesign
✅ **Developer experience preserved** - `crisk check` works on any branch
✅ **Efficient branching** - 50MB deltas vs 2GB full graphs (98% reduction)
✅ **Auto-scaling economics** - Pay only for what you use (true serverless)

### Negative

⚠️ **Query conversion effort** - 70-110 hours to convert Cypher → Gremlin (one-time)
⚠️ **Multi-tenancy complexity** - Need access control, reference counting, garbage collection
⚠️ **Operational monitoring** - Must track reference counts, archive/restore cycles
⚠️ **AWS lock-in** - Neptune is AWS-only (can't move to GCP/Azure easily)

### Neutral

- Backend abstraction layer needed (`GraphBackend` interface supports both Neo4j and Neptune)
- PostgreSQL schema for multi-tenant metadata
- S3 for archived public caches
- Cron jobs for garbage collection

### Risks & Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **Public cache thrashing** (frequent add/remove) | Low | Medium | 7-day soft delete before archival |
| **Reference count bugs** (orphaned graphs) | Medium | Low | Weekly audit job to reconcile |
| **Neptune cold start** (0.5 NCU → 8 NCU) | Low | Low | Keep popular repos warm (ref_count > 10) |
| **Gremlin query bugs** (vs Cypher) | Medium | High | Extensive testing, dual-write during migration |
| **Multi-tenant data leak** | Very Low | Critical | RLS in PostgreSQL + IAM policies + audit logs |

---

## Implementation Notes

### Phase 1: Infrastructure Setup (Week 1)

**Terraform resources:**
1. Neptune Serverless cluster (0.5-16 NCUs)
2. RDS PostgreSQL (t4g.micro, multi-tenant metadata)
3. ElastiCache Redis (t4g.micro, Phase 1 cache)
4. ECS Fargate (crisk-api service, 2-10 tasks)
5. ALB with HTTPS (ACM certificate)
6. S3 bucket (archived public caches, lifecycle policies)
7. Secrets Manager (GitHub OAuth secrets, encryption keys)

**PostgreSQL schema:**
```sql
CREATE TABLE repositories (
    id UUID PRIMARY KEY,
    github_owner VARCHAR(255),
    github_repo VARCHAR(255),
    visibility VARCHAR(20),  -- 'public' or 'private'
    neptune_db_name VARCHAR(500),
    reference_count INT DEFAULT 0,  -- Key metric!
    state VARCHAR(20),  -- 'building', 'ready', 'warm', 'archived', 'deleted'
    last_accessed_at TIMESTAMP,
    UNIQUE(github_owner, github_repo)
);

CREATE TABLE repository_access (
    user_id UUID REFERENCES users(id),
    org_id UUID REFERENCES organizations(id),
    repo_id UUID REFERENCES repositories(id),
    role VARCHAR(20),
    UNIQUE(user_id, repo_id)
);
```

### Phase 2: Multi-Tenant Query Router (Week 2)

**Application layer:**
```go
// internal/graph/multi_tenant.go
type MultiTenantBackend struct {
    neptune  *NeptuneClient
    postgres *PostgresClient
    cache    *RedisClient
}

func (m *MultiTenantBackend) Query(ctx context.Context, userID, repoID uuid.UUID, query string) ([]Result, error) {
    // 1. Verify access (PostgreSQL RLS)
    hasAccess := m.postgres.VerifyAccess(ctx, userID, repoID)
    if !hasAccess {
        return nil, ErrUnauthorized
    }

    // 2. Get Neptune database name
    repo := m.postgres.GetRepository(ctx, repoID)

    // 3. Query Neptune with namespaced database
    results := m.neptune.QueryDatabase(ctx, repo.NeptuneDBName, query)

    // 4. Update last_accessed_at
    m.postgres.TouchRepository(ctx, repoID)

    return results, nil
}
```

### Phase 3: Public Cache Lifecycle (Week 3)

**Reference counting logic:**
```go
func (m *MultiTenantBackend) AddRepository(ctx context.Context, userID uuid.UUID, githubURL string) error {
    owner, repo := parseGitHubURL(githubURL)
    visibility := m.github.GetVisibility(owner, repo)

    if visibility == "public" {
        // Check if public cache exists
        existing := m.postgres.FindRepository(ctx, owner, repo)

        if existing != nil {
            // Cache hit! Increment ref_count
            m.postgres.IncrementRefCount(ctx, existing.ID)
            m.postgres.GrantAccess(ctx, userID, existing.ID)
            return nil  // Instant access
        } else {
            // First user - build public cache
            neptuneDB := fmt.Sprintf("neptune_public_repo_%s_%s_main", owner, repo)
            repoID := m.postgres.CreateRepository(ctx, owner, repo, "public", neptuneDB, 1)
            m.postgres.GrantAccess(ctx, userID, repoID)
            m.enqueueGraphBuild(repoID, owner, repo, neptuneDB)
            return nil
        }
    } else {
        // Private repo - isolated graph
        neptuneDB := fmt.Sprintf("neptune_org_%s_repo_%s_main", orgID, uuid.New())
        // ... create isolated graph
    }
}
```

**Garbage collection (cron job):**
```go
// Run daily at 2 AM UTC
func GarbageCollectUnusedCaches(ctx context.Context) {
    // Find repos with ref_count=0 for 30+ days
    candidates := postgres.Query(`
        SELECT id, neptune_db_name FROM repositories
        WHERE reference_count = 0
          AND last_accessed_at < NOW() - INTERVAL '30 days'
          AND state = 'ready'
    `)

    for _, repo := range candidates {
        // Archive to S3
        snapshot := neptune.CreateSnapshot(repo.NeptuneDBName)
        s3.Upload(snapshot, fmt.Sprintf("archives/%s.snapshot", repo.ID))

        // Delete Neptune database
        neptune.DeleteDatabase(repo.NeptuneDBName)

        // Update state
        postgres.UpdateState(repo.ID, "archived")

        log.Info("Archived unused public cache", "repo", repo.NeptuneDBName)
    }
}
```

### Phase 4: Branch Delta Support (Week 4)

**Delta creation on-demand:**
```bash
# User runs crisk check on feature branch
crisk check

# Backend flow:
1. Detect branch: feature/new-auth
2. Check if delta exists: neptune_public_repo_facebook_react_branch_{sha}
3. If not:
   - git diff main..HEAD (find changed files)
   - Parse only changed files (50 files vs 5,000 total)
   - Create delta graph (50MB vs 2GB full)
4. Query: Merge(main_graph, branch_delta) at runtime
5. Cache result in Redis (15-min TTL)
```

**Storage efficiency:**
```
Main branch graph: 2GB (shared across all users)
Feature branch delta: 50MB (10 changed files)
Total per feature branch: 50MB (98% smaller)
```

### Timeline

- **Week 1:** Infrastructure (Terraform, PostgreSQL schema, Neptune cluster)
- **Week 2:** Multi-tenant query router + access control
- **Week 3:** Public cache lifecycle + garbage collection
- **Week 4:** Branch delta support + testing
- **Week 5:** GitHub OAuth integration
- **Week 6:** Monitoring dashboards + alerting
- **Week 7:** Load testing + optimization
- **Week 8:** Production deployment

**Total:** 8 weeks to full production readiness

---

## Cost Projection

### At 1,000 Users (Month 12)

**Neptune Storage:**
- Public caches (100 popular repos): 200GB × $0.10 = $20/month
- Private repos (100 orgs): 200GB × $0.10 = $20/month
- Branch deltas (200 active): 10GB × $0.10 = $1/month
- **Total storage: $41/month**

**Neptune Compute (auto-scales):**
- Peak hours (8 NCUs avg × 8h × 22 days): 1,408 NCU-hours
- Off-peak (2 NCUs avg × 16h × 22 days): 704 NCU-hours
- Idle (0.5 NCUs × 48h × 8 days): 192 NCU-hours
- **Total: 2,304 NCU-hours × $0.12 = $277/month**

**Other AWS Services:**
- RDS PostgreSQL (t4g.micro): $15/month
- ElastiCache Redis (t4g.micro): $12/month
- ECS Fargate (4 tasks avg): $50/month
- ALB: $20/month
- S3 (archived caches): $5/month
- **Total other: $102/month**

**Grand Total: $420/month @ 1,000 users = $0.42/user/month**

**Comparison:**
- Neo4j Aura Enterprise: $2,000/month ($2.00/user)
- Neptune multi-tenant: $420/month ($0.42/user)
- **Savings: 79% cheaper**

---

## Success Metrics

### Performance Targets

- [ ] Public cache hit rate > 80% (popular repos already cached)
- [ ] Branch delta creation < 5s (vs 5-10 min full graph)
- [ ] Query latency with delta < 500ms (base + delta merge)
- [ ] Reference count accuracy > 99.9% (audit job validates)

### Cost Targets

- [ ] Storage per user < $0.10/month @ 1K users
- [ ] Compute per user < $0.30/month @ 1K users
- [ ] Total infrastructure < $500/month @ 1K users

### Operational Targets

- [ ] Garbage collection recovers > 90% of archived storage
- [ ] Zero data leaks between private repos (audit logs)
- [ ] S3 restoration time < 2 minutes (99th percentile)

---

## References

- **ADR-001:** [Neptune over Neo4j](001-neptune-over-neo4j.md) - Original Neptune recommendation
- **ADR-004:** [Neo4j Aura to Neptune Migration](004-neo4j-aura-to-neptune-migration.md) - Phased approach (partially superseded)
- **[cloud_deployment.md](../cloud_deployment.md)** - Will be updated to reflect multi-tenant architecture
- **[public_caching.md](../../02-operations/public_caching.md)** - Public repository caching strategy
- **[team_and_branching.md](../../02-operations/team_and_branching.md)** - Branch delta architecture
- **AWS Neptune Multi-Tenancy:** https://docs.aws.amazon.com/neptune/latest/userguide/feature-overview-db-clustering.html

---

## Decision Log

**2025-10-12:** Accepted multi-tenant Neptune architecture with public caching
- **Rationale:** 99% storage savings + 79% cost reduction vs Neo4j + scalable to 10K+ users
- **Action:** Build Terraform infrastructure, implement multi-tenant query router
- **Review:** Month 6 (evaluate at 500 users), Month 12 (validate cost projections)
