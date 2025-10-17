# ADR 004: Phased Cloud Graph Database Strategy (Neo4j Aura → Neptune)

**Status:** Superseded by [ADR-006](006-multi-tenant-neptune-architecture.md)
**Date:** October 10, 2025
**Superseded Date:** October 12, 2025
**Deciders:** Architecture team
**Supersedes:** [ADR-001: Neptune over Neo4j](001-neptune-over-neo4j.md) (partial)

> **⚠️ UPDATE (Oct 12, 2025):** This ADR has been superseded by [ADR-006: Multi-Tenant Neptune Architecture](006-multi-tenant-neptune-architecture.md). The analysis showed that Neptune's serverless pricing ($50/month @ idle) + multi-tenant public caching (99% storage savings) makes it more cost-effective to deploy Neptune from day 1 rather than migrate from Neo4j Aura. The phased approach added unnecessary migration overhead ($15k) without providing meaningful cost savings during MVP phase.

---

## Context and Problem Statement

We need a cloud graph database for the MVP deployment, but face competing priorities:

1. **Speed to market** - Deploy within 1-2 weeks with minimal setup
2. **Cost efficiency** - Minimize infrastructure costs during beta/early growth
3. **Scale economics** - Prepare for 1,000+ users with sustainable unit economics
4. **Migration risk** - Avoid premature optimization while preserving future optionality

**Original Decision (ADR-001):** Recommended AWS Neptune for cost savings vs Neo4j Aura
**Problem:** Neptune requires upfront setup, query conversion, and delays MVP by 2-4 weeks

**Question:** Should we stick with Neptune (long-term winner) or start with Neo4j Aura (faster MVP)?

---

## Decision Drivers

### Speed to Market
- ✅ **Critical:** MVP deployment within 1-2 weeks
- ✅ **Zero migration effort:** Local Docker uses Neo4j Community → same Cypher queries in cloud
- ❌ **Neptune delay:** 2-4 weeks for setup, query conversion, testing

### Cost Efficiency
- ✅ **0-100 users:** Neo4j Aura **Free tier** = $0/month (vs Neptune $450)
- ✅ **100-500 users:** Neo4j Aura Pro = $65-200/month (acceptable for growth phase)
- ❌ **1,000+ users:** Neo4j Enterprise = $2,000/month (Neptune @ $450 is 77% cheaper)

### Technical Risk
- ✅ **Neo4j:** Same queries as local, proven in production, community support
- ❌ **Neptune:** Gremlin learning curve, query conversion risk, cold start latency

### Learning Opportunity
- ✅ **Validate product-market fit** with Neo4j Aura before committing to Neptune
- ✅ **Real-world query patterns** inform Neptune migration (avoid premature optimization)
- ✅ **User feedback** on graph size/performance before locking in database

---

## Considered Options

### Option 1: Neptune from Day 1 (Original ADR-001)
**Pros:**
- ✅ Best long-term economics (77% cost savings at scale)
- ✅ Unlimited scale (auto-scales 0.5-128 NCUs)
- ✅ No future migration needed

**Cons:**
- ❌ 2-4 week delay to MVP
- ❌ Upfront migration cost (~$15k dev time)
- ❌ Risk of over-engineering before product-market fit
- ❌ Gremlin learning curve for team

**Total Cost (24 months, 1K users):** $55,200 (Neptune) + $15,000 (migration) = **$70,200**

---

### Option 2: Neo4j Aura Only (No Migration Plan)
**Pros:**
- ✅ Fastest MVP (deploy this week)
- ✅ Zero migration from local Docker
- ✅ Simple Cypher queries (team already knows)
- ✅ Free tier for MVP ($0 graph cost)

**Cons:**
- ❌ Expensive at scale ($2,000/month @ 1K users)
- ❌ No migration path planned (risk of lock-in)
- ❌ Higher long-term costs

**Total Cost (24 months, 1K users):** $21,840 (Free, 6mo) + $48,000 (Enterprise, 18mo) = **$69,840**

---

### Option 3: Phased Migration (Neo4j → Neptune) ✅ **CHOSEN**
**Pros:**
- ✅ **Week 1 MVP:** Neo4j Aura Free tier ($0/month, zero setup)
- ✅ **Months 1-12:** Neo4j Aura Pro ($65-200/month, one-click upgrade)
- ✅ **Month 12+:** Migrate to Neptune (77% cost savings validated)
- ✅ Validates product-market fit before major migration
- ✅ Real query patterns inform Neptune optimization
- ✅ Same codebase supports both (abstraction layer)

**Cons:**
- ⚠️ One-time migration effort (70-110 dev hours)
- ⚠️ Dual backend support (acceptable, <5% code overhead)

**Total Cost (24 months, 1K users):**
- Months 0-6 (Neo4j Free): $5,460 (infrastructure only)
- Months 6-12 (Neo4j Pro @ $150): $10,200
- Months 12-24 (Neptune @ $450): $27,600
- Migration cost: $15,000
- **Total: $58,260** (vs $69,840 Neo4j-only, vs $70,200 Neptune-only)

**ROI:** Saves $11,580 vs Neo4j-only, while being faster than Neptune-only by 2-4 weeks

---

## Decision Outcome

**Chosen option:** **Option 3 - Phased Migration (Neo4j Aura → Neptune)**

### Rationale

1. **Speed to market wins** - Deploy MVP this week with Neo4j Aura Free (zero setup)
2. **De-risk product** - Validate PMF before investing $15k in Neptune migration
3. **Best economics** - Free tier (0-100 users), affordable growth (Pro @ $65-200), then migrate to Neptune for scale
4. **Learning period** - Months 0-12 inform Neptune query optimization (avoid premature design)
5. **Backend abstraction** - Same codebase supports both (GraphBackend interface already exists)

### Implementation Phases

#### Phase 1: MVP (Weeks 1-2, 0-100 users)
**Graph Database:** Neo4j Aura Free Tier
- Cost: **$0/month** (50k nodes, 175k edges)
- Setup: Create Aura account, copy connection string to env vars
- Migration: **Zero** (same Cypher queries as local Docker)
- Deployment time: **2-3 hours** (vs 2-4 weeks for Neptune)
- Supports: ~10 small repos or 1 large repo

**Action Items:**
- [ ] Create Neo4j Aura Free account
- [ ] Update `.env.production` with Aura URI
- [ ] Deploy Kubernetes manifests (no code changes)
- [ ] Verify graph construction works end-to-end

#### Phase 2: Growth (Months 4-12, 100-500 users)
**Graph Database:** Neo4j Aura Professional
- Cost: **$65-200/month** (pay-as-you-grow)
- Capacity: 200M nodes (perfect for 500 users)
- Migration: **One-click upgrade** in Aura console
- No code changes required

**Trigger:** When Free tier hits capacity (~100 users or 50k nodes)

**Action Items:**
- [ ] Monitor node count in Aura dashboard
- [ ] Upgrade to Pro when approaching limits
- [ ] Update billing/cost tracking

#### Phase 3: Scale (Month 12+, 1,000+ users)
**Graph Database:** AWS Neptune Serverless
- Cost: **$450/month** @ 1K users (77% savings vs Neo4j Enterprise)
- Migration effort: **70-110 dev hours**
- Break-even: **Month 10** of Neptune operation

**Trigger:** When reaching 500+ users OR Neo4j Aura cost >$200/month

**Migration Plan:**
1. **Weeks 1-2:** Implement Gremlin backend (parallel to Cypher)
2. **Week 3:** Convert 20 core queries (Cypher → Gremlin)
3. **Week 4:** Dual-write pattern (writes go to both, reads from Cypher)
4. **Week 5:** Backfill Neptune from Neo4j (full data copy)
5. **Week 6:** Gradual traffic shift (10% → 50% → 100% reads from Neptune)
6. **Week 7:** Cutover complete, Neo4j deprecated
7. **Week 8:** Monitor, optimize, celebrate 77% cost savings

**Migration Cost:** ~100 hours × $150/hr = **$15,000**

---

## Consequences

### Positive

✅ **Immediate MVP deployment** - Launch this week with Neo4j Aura Free
✅ **Zero upfront migration cost** - Delay Neptune work until after PMF validation
✅ **Flexible growth path** - One-click upgrade to Pro, planned migration to Neptune
✅ **Risk mitigation** - Avoid over-engineering before user feedback
✅ **Cost optimization** - Free → $65-200 → $450 (vs $450 from day 1)
✅ **Informed migration** - Real query patterns guide Neptune optimization

### Negative

⚠️ **One-time migration effort** - 70-110 dev hours at Month 12 (acceptable)
⚠️ **Dual backend code** - GraphBackend interface supports both (minimal overhead)
⚠️ **Temporary higher cost** - Months 6-12 pay $65-200 vs $0 (but still validates PMF)

### Neutral

- Backend abstraction already exists (`internal/graph/backend.go`)
- Same Cypher queries work in local Docker + Neo4j Aura
- Migration can be accelerated if hitting Neo4j limits early

---

## Detailed Cost Comparison

### 24-Month Total Cost (1,000 users)

| Phase | Duration | Database | DB Cost | Infra Cost | Total | Cumulative |
|-------|----------|----------|---------|------------|-------|------------|
| **MVP** | Months 0-6 | Neo4j Free | $0 | $910/mo | $5,460 | $5,460 |
| **Growth** | Months 6-12 | Neo4j Pro | $150/mo | $1,220/mo | $8,220 | $13,680 |
| **Migration** | Month 12 | — | — | — | $15,000 | $28,680 |
| **Scale** | Months 12-24 | Neptune | $450/mo | $1,850/mo | $27,600 | **$56,280** |

**Comparison:**
- **Phased (Neo4j → Neptune):** $56,280
- **Neptune from Day 1:** $70,200 ($15k migration + $55,200 infra)
- **Neo4j Aura only:** $69,840 ($21,840 Free + $48,000 Enterprise)

**Savings:** $11,580 vs Neptune-only, $13,560 vs Neo4j-only

---

## Database Feature Comparison

| Feature | Neo4j Aura Free | Neo4j Aura Pro | Neptune Serverless |
|---------|-----------------|----------------|--------------------|
| **Pricing** | $0/month | $65-200/month | $450/month @ 1K users |
| **Capacity** | 50k nodes, 175k edges | 200M nodes, 1B edges | Unlimited |
| **Query Language** | Cypher | Cypher | Gremlin + openCypher |
| **Auto-scaling** | No | Yes (manual) | Yes (automatic) |
| **Multi-AZ** | No | Yes | Yes |
| **Backup** | Community support | Automated (7-day) | Automated (35-day) |
| **Support** | Community | 24/7 email/chat | AWS Premium Support |
| **Migration from local** | Zero (same DB) | Zero (same queries) | 70-110 dev hours |
| **Best for** | MVP, <100 users | Growth, 100-500 users | Scale, 1,000+ users |

---

## Migration Strategy Details

### Backend Abstraction (Already Implemented)

```go
// internal/graph/backend.go
type GraphBackend interface {
    CreateNodes(ctx context.Context, nodes []Node) error
    CreateEdges(ctx context.Context, edges []Edge) error
    Query(ctx context.Context, query string, params map[string]interface{}) ([]Result, error)
}

type Neo4jBackend struct { /* Cypher implementation */ }
type NeptuneBackend struct { /* Gremlin implementation */ }
```

**Current state:** Neo4jBackend fully implemented for local + Aura
**Future work:** Implement NeptuneBackend at Month 12

### Query Conversion Examples

**Cypher (Neo4j):**
```cypher
MATCH (f:File {file_path: $path})-[:CALLS]->(target:Function)
RETURN target.name, target.file_path
LIMIT 10
```

**Gremlin (Neptune):**
```gremlin
g.V().has('File', 'file_path', path)
  .out('CALLS')
  .hasLabel('Function')
  .valueMap('name', 'file_path')
  .limit(10)
```

**Conversion effort:** ~20 core queries, 2-3 hours each = **40-60 hours**

### Dual-Write Pattern (Week 4 of migration)

```go
func (s *MigrationBackend) CreateNodes(ctx context.Context, nodes []Node) error {
    // Write to both Neo4j and Neptune during migration
    errNeo4j := s.neo4jBackend.CreateNodes(ctx, nodes)
    errNeptune := s.neptuneBackend.CreateNodes(ctx, nodes)

    if errNeo4j != nil {
        return fmt.Errorf("neo4j write failed: %w", errNeo4j)
    }
    if errNeptune != nil {
        log.Warn("neptune write failed (non-blocking): %v", errNeptune)
    }
    return nil
}

func (s *MigrationBackend) Query(ctx context.Context, query string, params map[string]interface{}) ([]Result, error) {
    // Read from Neo4j (primary) with fallback to Neptune
    results, err := s.neo4jBackend.Query(ctx, query, params)
    if err != nil {
        log.Warn("neo4j query failed, trying neptune: %v", err)
        return s.neptuneBackend.Query(ctx, convertCypherToGremlin(query), params)
    }
    return results, nil
}
```

**Duration:** 2 weeks dual-write, then gradual traffic shift

---

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| **Neo4j Free capacity exceeded early** | Medium | Medium | Upgrade to Pro (1-click), or accelerate Neptune migration |
| **Neptune migration takes longer** | Low | Medium | Budget 110 hours (worst-case), hire consultant if needed |
| **Query conversion introduces bugs** | Medium | High | Dual-write pattern, gradual traffic shift, comprehensive testing |
| **Neo4j vendor lock-in** | Low | Low | Backend abstraction prevents lock-in, migration is planned |
| **Neptune cold start latency** | Low | Low | Keep 0.5 NCU warm, pre-warm on traffic spikes |

**Overall risk:** **Low-Medium** - Mitigated by phased approach, backend abstraction, and planned migration

---

## Monitoring & Success Criteria

### Phase 1 (Neo4j Free) Success Criteria
- [ ] Graph construction completes in <10 min for 500-file repo
- [ ] Query latency p95 <200ms (cached)
- [ ] Zero downtime during Aura Free → Pro upgrade
- [ ] Node count stays <45k (5k buffer before limit)

### Phase 2 (Neo4j Pro) Success Criteria
- [ ] Support 500 users with <$200/month graph cost
- [ ] Query latency p95 <300ms (at scale)
- [ ] Backup/restore tested and documented
- [ ] Multi-AZ failover tested (simulate AZ outage)

### Phase 3 (Neptune) Success Criteria
- [ ] Migration completes in <8 weeks
- [ ] Zero data loss during migration
- [ ] Query latency matches or beats Neo4j
- [ ] Monthly cost <$500 @ 1K users
- [ ] Auto-scaling works (0.5 → 128 NCUs)

---

## References

- [ADR-001: Neptune over Neo4j](001-neptune-over-neo4j.md) - Original Neptune recommendation
- [cloud_deployment.md](../cloud_deployment.md) - Updated with phased strategy
- [Neo4j Aura Pricing](https://neo4j.com/pricing/) - Free and Professional tiers
- [AWS Neptune Pricing](https://aws.amazon.com/neptune/pricing/) - Serverless pricing calculator
- [Neo4j to Neptune Migration Guide](https://docs.aws.amazon.com/neptune/latest/userguide/migrate-neo4j.html)

---

## Decision Log

**October 10, 2025:** Accepted Option 3 (Phased Migration)
- Rationale: Speed to market + cost efficiency + de-risked migration
- Action: Deploy Neo4j Aura Free tier for MVP launch
- Review: Month 6 (assess upgrade to Pro), Month 12 (plan Neptune migration)

**Supersedes:** ADR-001 (partial) - Neptune is still the long-term choice, but phased approach is better
