# Graph Database Comparison: Neptune vs Neo4j vs Alternatives

**Version:** 1.1
**Last Updated:** October 10, 2025
**Purpose:** Evaluate graph database options for CodeRisk cloud service
**Decision:** Amazon Neptune Serverless (recommended for scale)
**Status:** Partially superseded by [ADR-004](004-neo4j-aura-to-neptune-migration.md) (phased approach)

> **⚠️ UPDATE (Oct 10, 2025):** This ADR's recommendation (Neptune for long-term) is still valid, but **implementation strategy has changed** to a phased approach:
> - **Phase 1 (MVP):** Neo4j Aura Free ($0/month, 0-100 users)
> - **Phase 2 (Growth):** Neo4j Aura Professional ($65-200/month, 100-500 users)
> - **Phase 3 (Scale):** AWS Neptune Serverless ($450/month, 1,000+ users)
>
> See **[ADR-004: Neo4j Aura to Neptune Migration](004-neo4j-aura-to-neptune-migration.md)** for complete phased strategy and rationale.

---

## 1. Executive Summary

**Recommendation: Amazon Neptune Serverless**

**Why:**
- ✅ **70-85% cost savings** vs Neo4j Aura ($300-600/month vs $2,000/month)
- ✅ **True serverless** - scales to zero when idle, pay per request
- ✅ **Better cost model** for variable workloads (investigations are bursty)
- ✅ **Native AWS integration** (IAM, VPC, KMS, CloudWatch)
- ✅ **Auto-scaling** - handles traffic spikes without provisioning

**Trade-offs:**
- ⚠️ Cypher vs Gremlin/openCypher (Neptune supports openCypher, close enough)
- ⚠️ Cold start latency (100-500ms for first query after idle)
- ⚠️ Less mature ecosystem than Neo4j

**Bottom line:** For our use case (bursty investigations, multi-tenant SaaS), Neptune Serverless saves ~$20K/year per 1,000 users while providing better auto-scaling.

---

## 2. Detailed Comparison

### 2.1. Cost Analysis (1,000 users, 100 repos)

**Scenario:** 1,000 users, 10 investigations/user/day, 100 repositories

**Amazon Neptune Serverless:**
```
Pricing model: NCUs (Neptune Capacity Units)
- 1 NCU = 2 GB RAM + 1 vCPU
- $0.12 per NCU-hour

Workload estimation:
  Investigation pattern:
    - Peak hours (9am-5pm): 80% of traffic = 8,000 investigations/hour = 2.2/sec
    - Off-peak (5pm-9am): 20% of traffic = 2,000 investigations/hour = 0.56/sec
    - Nights/weekends: Near zero

  NCU requirements:
    - Peak (2.2 req/sec): 16 NCUs (can handle 3-5 req/sec per NCU)
    - Off-peak (0.56 req/sec): 4 NCUs
    - Idle: 0.5 NCU (min capacity for fast wake-up)

  Monthly hours:
    - Peak (8 hours × 22 weekdays): 176 hours × 16 NCUs = 2,816 NCU-hours
    - Off-peak (16 hours × 22 weekdays): 352 hours × 4 NCUs = 1,408 NCU-hours
    - Idle (nights, weekends): 192 hours × 0.5 NCUs = 96 NCU-hours

  Total NCU-hours: 2,816 + 1,408 + 96 = 4,320 NCU-hours/month
  Cost: 4,320 × $0.12 = $518/month

  Storage (100 repos, avg 2GB each):
    - 200GB × $0.10/GB-month = $20/month

  I/O (graph queries):
    - ~10M read IOs/month × $0.20/million = $2/month

  Backups:
    - Continuous backup included
    - Snapshots: ~$5/month

Total Neptune Serverless: $545/month
Per-user: $0.55/month
```

**Neo4j Aura Enterprise:**
```
Pricing model: Fixed instance pricing
- 8GB RAM, 32GB storage instance = $200/month
- Need 10 instances (one per 10 repos) = $2,000/month

OR

- 64GB RAM, 256GB storage instance = $1,200/month
- Need 1 instance (all repos) = $1,200/month
- But can't scale down during off-peak

Cost: $1,200-2,000/month
Per-user: $1.20-2.00/month

Savings with Neptune: 55-73%
```

**Neo4j Aura Professional:**
```
Pricing model: Fixed instance pricing
- 4GB RAM, 16GB storage = $65/month
- Need 20 instances (one per 5 repos) = $1,300/month

Cost: $1,300/month
Per-user: $1.30/month

Savings with Neptune: 58%
```

### 2.2. Scaling to 10,000 Users

**Amazon Neptune Serverless:**
```
Workload: 100,000 investigations/day = 4,167/hour peak = 1.16/sec

NCU requirements:
  - Peak (11.6 req/sec): 32 NCUs (Neptune auto-scales)
  - Off-peak (2.9 req/sec): 8 NCUs
  - Idle: 0.5 NCU

Monthly cost:
  - Peak: 176h × 32 NCUs = 5,632 NCU-hours × $0.12 = $676
  - Off-peak: 352h × 8 NCUs = 2,816 NCU-hours × $0.12 = $338
  - Idle: 192h × 0.5 NCUs = 96 NCU-hours × $0.12 = $12
  - Storage (1,000 repos, 2GB avg): 2TB × $0.10 = $200
  - I/O: 100M reads × $0.20/million = $20

Total: $1,246/month
Per-user: $0.12/month (economies of scale!)
```

**Neo4j Aura:**
```
Fixed instances don't benefit from scale:
  - Still need 100 instances (one per 10 repos)
  - 100 × $200 = $20,000/month

OR

  - 10× 64GB instances = $12,000/month

Cost: $12,000-20,000/month
Per-user: $1.20-2.00/month

Savings with Neptune: 90-94% at scale!
```

### 2.3. Feature Comparison

| Feature | Neptune Serverless | Neo4j Aura | Winner |
|---------|-------------------|------------|--------|
| **Query Language** | Gremlin, openCypher | Cypher | Neo4j (better Cypher) |
| **Serverless** | ✅ True serverless | ❌ Fixed instances | Neptune |
| **Auto-scaling** | ✅ Automatic (0.5-128 NCUs) | ❌ Manual resize | Neptune |
| **Cold start** | ~100-500ms | N/A (always on) | Neo4j |
| **Multi-tenancy** | ✅ Single cluster, multiple DBs | ❌ Need multiple instances | Neptune |
| **Backup** | ✅ Continuous, point-in-time | ✅ Automatic snapshots | Tie |
| **HA/Failover** | ✅ Multi-AZ by default | ✅ HA clusters available | Tie |
| **IAM integration** | ✅ Native AWS IAM | ❌ Separate auth | Neptune |
| **VPC integration** | ✅ Native VPC | ✅ VPC peering | Tie |
| **Monitoring** | ✅ CloudWatch native | ⚠️ Third-party + CloudWatch | Neptune |
| **Ecosystem** | ⚠️ Smaller | ✅ Larger (Neo4j plugins) | Neo4j |
| **Client libraries** | ✅ Gremlin (all languages) | ✅ Neo4j drivers | Tie |
| **Graph algorithms** | ⚠️ Manual implementation | ✅ Graph Data Science lib | Neo4j |
| **ACID transactions** | ✅ Full ACID | ✅ Full ACID | Tie |
| **Cost** | ✅ $545/month (1K users) | ❌ $1,200-2,000/month | Neptune |

### 2.4. Query Language: openCypher vs Cypher

**Neo4j Cypher:**
```cypher
// Find co-changed files
MATCH (f1:File)-[:CO_CHANGED_WITH {frequency: > 0.5}]->(f2:File)
WHERE f1.path = 'auth.py'
RETURN f2.path, f2.co_change_frequency
ORDER BY f2.co_change_frequency DESC
LIMIT 10
```

**Neptune openCypher (99% compatible):**
```cypher
// Same query works!
MATCH (f1:File)-[:CO_CHANGED_WITH]->(f2:File)
WHERE f1.path = 'auth.py' AND r.frequency > 0.5
RETURN f2.path, f2.co_change_frequency
ORDER BY f2.co_change_frequency DESC
LIMIT 10
```

**Differences (minor):**
- Neptune doesn't support some advanced Cypher features (e.g., APOC procedures)
- No Graph Data Science library (have to implement algorithms ourselves)
- 95% of our queries will work identically

**Mitigation:**
- Write Cypher-compatible queries
- Use common subset of openCypher
- If we ever migrate to Neo4j, minimal code changes

### 2.5. Gremlin Alternative (if we want Neptune-native)

**Gremlin query:**
```groovy
// Find co-changed files
g.V().has('File', 'path', 'auth.py')
  .outE('CO_CHANGED_WITH').has('frequency', gt(0.5))
  .inV()
  .order().by('co_change_frequency', desc)
  .limit(10)
  .values('path', 'co_change_frequency')
```

**Pros:**
- Native to Neptune (better optimized)
- More expressive for complex graph traversals
- Better performance (10-30% faster)

**Cons:**
- Less familiar syntax (learning curve)
- Harder to read (not SQL-like)
- Migration to Neo4j harder if needed

**Recommendation:** Start with openCypher (familiar, portable), optimize with Gremlin later if needed

---

## 3. Cold Start Analysis

### 3.1. What is Cold Start?

**Neptune Serverless scales to 0.5 NCU minimum when idle:**
- Below 0.5 NCU: cluster pauses (no cost)
- First query after pause: 100-500ms latency (cold start)
- Subsequent queries: normal latency (10-50ms)

**When does it happen?**
- After 10-15 minutes of no queries
- Typically: nights, weekends, holidays

**Impact:**
```
User flow:
  1. User runs: crisk check (first query of the day)
  2. API → Neptune (cluster waking up)
  3. Cold start: 100-500ms extra latency
  4. Total investigation: 2.5s (vs 2.0s warm)
  5. User sees: "Risk: HIGH (2.5s)"

Next investigation (within 15 min):
  1. User runs: crisk check
  2. API → Neptune (already warm)
  3. No cold start
  4. Total investigation: 2.0s
  5. User sees: "Risk: HIGH (2.0s)"
```

**Is this acceptable?**
✅ Yes! Users won't notice 0.5s difference
✅ First check of the day being 2.5s vs 2.0s is fine
✅ Saves $1,400+/month vs always-on

### 3.2. Mitigation Strategies

**Option 1: Keep-alive ping (simple)**
```python
# Cron job every 10 minutes during work hours
# Runs lightweight query to keep cluster warm

@cron("*/10 * * * *")  # Every 10 minutes
def keep_neptune_warm():
    if is_work_hours():  # 6am-8pm UTC (covers US + Europe)
        neptune.execute("MATCH (n) RETURN count(n) LIMIT 1")

Cost: Minimal (0.5 NCU × 14 hours × 30 days = 210 NCU-hours × $0.12 = $25/month)
Benefit: No cold starts during work hours
```

**Option 2: Predictive pre-warming**
```python
# Warm up cluster before first expected usage
# Based on historical patterns

@cron("0 6 * * 1-5")  # 6am weekdays
def prewarm_neptune():
    neptune.set_min_capacity(4)  # Scale to 4 NCUs
    time.sleep(60)  # Keep warm for 1 min
    neptune.set_min_capacity(0.5)  # Back to auto-scale
```

**Option 3: Accept cold starts (recommended)**
- 0.5s extra latency for first query is acceptable
- Saves $1,400+/month
- Most users won't notice
- Those who do won't care (still <3s total)

**Recommendation:** Option 3 (accept cold starts), add Option 1 (keep-alive) if users complain

---

## 4. Alternative: Azure Cosmos DB (Gremlin API)

### 4.1. Overview

**Cosmos DB with Gremlin API:**
- Microsoft's globally distributed database
- Gremlin API (same as Neptune)
- Serverless option available

**Pricing (serverless):**
```
Request Units (RUs):
  - Read: 1 RU per KB
  - Write: 5 RU per KB
  - Query: varies (10-100 RU typical)

Typical investigation:
  - 5 graph queries × 20 RU each = 100 RU
  - $0.25 per million RU

Cost (1,000 users, 220K investigations/month):
  - 220K × 100 RU = 22M RU
  - 22M × $0.25/million = $5.50/month

Storage (200GB):
  - $0.25/GB-month = $50/month

Total: $55.50/month
```

**Looks cheaper! But...**

**Hidden costs:**
- ❌ Egress fees: $0.087/GB (add $200+/month for query results)
- ❌ Geo-replication required for HA: 2-3× cost
- ❌ Dedicated gateway for low latency: $100+/month
- ❌ RU estimation is hard (over-provisioning common)

**Actual cost:** $300-500/month (still cheaper than Neo4j, but less predictable)

### 4.2. Cosmos DB vs Neptune

| Feature | Cosmos DB Gremlin | Neptune Serverless | Winner |
|---------|------------------|-------------------|--------|
| **Cost (predictable)** | ⚠️ Variable (RU-based) | ✅ NCU-based (easier) | Neptune |
| **Cold start** | ❌ None (always-on dedicated gateway) | ~100-500ms | Cosmos |
| **Multi-region** | ✅ Global distribution | ⚠️ Manual setup | Cosmos |
| **Latency** | ✅ <10ms (SSD-backed) | ✅ 10-50ms | Cosmos |
| **Graph algorithms** | ❌ Manual | ❌ Manual | Tie |
| **Ecosystem** | ⚠️ Smaller | ⚠️ Small | Tie |
| **AWS integration** | ❌ None | ✅ Native | Neptune |
| **Azure integration** | ✅ Native | ❌ None | Cosmos |

**When to use Cosmos DB:**
- Need global distribution (multi-region writes)
- Already on Azure (not AWS)
- Need guaranteed <10ms latency

**Why Neptune is better for us:**
- On AWS already (RDS, EKS, S3)
- More predictable costs
- Better monitoring (CloudWatch)
- Lower egress fees

---

## 5. Alternative: ArangoDB (Multi-model)

### 5.1. Overview

**ArangoDB:**
- Multi-model: Graph + Document + Key-Value
- AQL query language (similar to SQL)
- Self-managed or ArangoGraph (managed cloud)

**ArangoGraph Cloud (managed):**
```
Pricing (AWS deployment):
  - 8GB RAM, 32GB storage = $200/month (similar to Neo4j)
  - No serverless option

Cost (1,000 users, 100 repos):
  - 10 instances × $200 = $2,000/month

Same cost as Neo4j Aura, but:
  - ✅ Multi-model (can store documents + graph)
  - ✅ Single query language for both
  - ❌ Smaller ecosystem than Neo4j
  - ❌ Not serverless
```

**When to use ArangoDB:**
- Need document DB + graph DB in one
- Want SQL-like query language
- Don't need serverless

**Why Neptune is better:**
- 70% cheaper (serverless)
- Pure graph database (simpler)
- Better AWS integration

---

## 6. Alternative: TigerGraph (High-performance)

### 6.1. Overview

**TigerGraph:**
- Fastest graph database (MPP architecture)
- GSQL query language
- TigerGraph Cloud (managed)

**Pricing:**
```
TigerGraph Cloud:
  - 16GB RAM, 100GB storage = $400/month
  - No serverless

Cost (1,000 users, 100 repos):
  - 5 instances × $400 = $2,000/month

Same cost as Neo4j, but:
  - ✅ 10-100× faster for complex graph algorithms (PageRank, shortest path)
  - ❌ Proprietary query language (GSQL, learning curve)
  - ❌ Smaller ecosystem
  - ❌ Not serverless
```

**When to use TigerGraph:**
- Need real-time graph analytics (fraud detection, recommendations)
- Complex graph algorithms are primary use case
- Can afford fixed costs

**Why Neptune is better:**
- 70% cheaper
- We don't need ultra-fast graph algorithms (pre-compute works)
- Standard query language (Gremlin/openCypher)

---

## 7. Alternative: Dgraph (Open Source)

### 7.1. Overview

**Dgraph:**
- Open-source graph database
- GraphQL native
- Dgraph Cloud (managed)

**Pricing (self-hosted on EKS):**
```
Self-hosted:
  - 3× m5.large instances (HA): ~$300/month
  - Storage: 200GB EBS = $20/month
  - Total: ~$320/month

Cheaper than Neptune!

But:
  - ❌ We manage updates, backups, scaling
  - ❌ No auto-scaling
  - ❌ Operational overhead
```

**Dgraph Cloud (managed):**
```
Pricing:
  - Dedicated: $500-1,000/month
  - Shared: $200/month (not suitable for prod)

More expensive than Neptune
```

**When to use Dgraph:**
- Want GraphQL-native queries
- Have ops team to manage self-hosted
- Open-source requirement

**Why Neptune is better:**
- Fully managed (less ops)
- Auto-scaling
- Better SLA/support

---

## 8. Final Recommendation

### 8.1. Decision Matrix

| Database | Cost (1K users) | Serverless | AWS Native | Query Lang | Ease | Score |
|----------|----------------|-----------|-----------|-----------|------|-------|
| **Neptune Serverless** | **$545** | ✅ | ✅ | openCypher | ✅ | **9/10** |
| Neo4j Aura | $1,200-2,000 | ❌ | ⚠️ | Cypher | ✅ | 7/10 |
| Cosmos DB Gremlin | $300-500 | ✅ | ❌ | Gremlin | ⚠️ | 6/10 |
| ArangoDB | $2,000 | ❌ | ⚠️ | AQL | ✅ | 5/10 |
| TigerGraph | $2,000 | ❌ | ⚠️ | GSQL | ⚠️ | 5/10 |
| Dgraph (self-hosted) | $320 | ❌ | ⚠️ | GraphQL | ❌ | 6/10 |

### 8.2. Final Decision: Amazon Neptune Serverless

**Why:**
1. **Cost**: 70-85% cheaper than Neo4j ($545 vs $1,200-2,000/month)
2. **Serverless**: Scales to zero, pay per request (perfect for bursty workload)
3. **AWS integration**: Native IAM, VPC, KMS, CloudWatch
4. **Auto-scaling**: Handles traffic spikes (0.5-128 NCUs)
5. **Multi-tenancy**: Single cluster, multiple databases (one per repo or org)

**Trade-offs we accept:**
- Cold start latency (100-500ms for first query after idle)
  - Mitigation: Keep-alive ping during work hours if needed
- openCypher vs full Cypher (95% compatible)
  - Mitigation: Use Cypher subset, portable to Neo4j later
- Smaller ecosystem than Neo4j
  - Mitigation: Implement graph algorithms ourselves (PageRank, centrality)

**Implementation plan:**
1. Start with openCypher (familiar, portable)
2. Use Gremlin for complex traversals if needed (better performance)
3. Implement custom graph algorithms (one-time cost)
4. Monitor cold start impact, add keep-alive if users complain
5. Re-evaluate after 6 months (can migrate to Neo4j if needed)

### 8.3. Cost Summary

**Neptune Serverless (recommended):**
```
1,000 users:   $545/month  ($0.55/user)
10,000 users:  $1,246/month ($0.12/user)

Annual savings vs Neo4j:
  - 1K users: ($1,200 - $545) × 12 = $7,860/year
  - 10K users: ($12,000 - $1,246) × 12 = $129,048/year
```

**ROI:**
- Initial setup cost: ~$10K (implement graph algorithms, migration scripts)
- Break-even: 1.3 months
- 5-year savings: $32K (1K users) to $645K (10K users)

**Decision: Use Amazon Neptune Serverless**

---

## 9. Migration Path (If Needed)

### 9.1. Neptune → Neo4j (if we outgrow Neptune)

**When to migrate:**
- Need advanced Cypher features (APOC, GDS library)
- Cold start becomes unacceptable (>1s hurting UX)
- Need better ecosystem (third-party integrations)

**Migration effort:**
- Query translation: Low (95% compatible)
- Data export/import: Medium (CSV export → Neo4j import)
- Testing: Medium (validate all queries)
- Downtime: ~2-4 hours (blue-green deployment)

**Cost:**
- Migration project: ~$20K (2 weeks engineering)
- Ongoing: +$700-1,400/month (Neo4j premium)

**Likelihood:** Low (Neptune should scale to 100K+ users)

### 9.2. Self-Hosted → Neptune (if we start self-hosted)

**Migration effort:**
- Very easy (just change connection string)
- No schema changes needed

**Downtime:** <1 hour

---

## 10. Action Items

### 10.1. Immediate (Phase 1)

- [x] Choose Amazon Neptune Serverless
- [ ] Set up Neptune cluster (one per AWS region)
- [ ] Implement openCypher graph schema
- [ ] Build graph construction pipeline (tree-sitter → Neptune)
- [ ] Implement core graph queries (co-change, blast radius, ownership)

### 10.2. Short-term (Phase 2)

- [ ] Implement graph algorithms (PageRank, betweenness centrality)
- [ ] Set up monitoring (CloudWatch dashboards)
- [ ] Configure auto-scaling policies (0.5-128 NCUs)
- [ ] Load test with production data

### 10.3. Long-term (Phase 3)

- [ ] Evaluate cold start impact (add keep-alive if needed)
- [ ] Optimize queries (use Gremlin for hot paths)
- [ ] Multi-region deployment (DR)
- [ ] Re-evaluate at 10K users (migrate to Neo4j if needed)

---

## 11. Updated Cost Model

### 11.1. Infrastructure Costs (1,000 users)

**Before (with Neo4j Aura):**
```
Neo4j Aura:                      $2,000/month
Kubernetes (EKS):                $1,050/month
PostgreSQL (RDS):                  $350/month
Redis (ElastiCache):               $200/month
S3 + Secrets + WAF:                $105/month
────────────────────────────────────────────
Total:                           $3,705/month
Per-user:                          $3.71/month
```

**After (with Neptune Serverless):**
```
Neptune Serverless:                $545/month
Kubernetes (EKS):                $1,050/month
PostgreSQL (RDS):                  $350/month
Redis (ElastiCache):               $200/month
S3 + Secrets + WAF:                $105/month
────────────────────────────────────────────
Total:                           $2,250/month
Per-user:                          $2.25/month
────────────────────────────────────────────
Savings:                         $1,455/month (39% reduction!)
```

### 11.2. Pricing Impact

**New margins with Neptune:**

| Tier | Price | Infrastructure Cost | Profit | Margin |
|------|-------|-------------------|--------|--------|
| Free | $0 | $2.25 | -$2.25 | Loss leader |
| Pro | $9 | $2.25 | $6.75 | 300% |
| Team | $29 | $2.25 | $26.75 | 1,189% |

**Even better unit economics!**

This is a **no-brainer decision**. Neptune Serverless saves $17K/year at 1K users, $129K/year at 10K users, with minimal trade-offs.
