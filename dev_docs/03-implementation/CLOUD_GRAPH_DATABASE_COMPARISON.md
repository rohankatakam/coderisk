# Cloud Graph Database Comparison: AWS Neptune vs Alternatives

**Version:** 1.0
**Last Updated:** October 10, 2025
**Purpose:** Evaluate cloud graph database options for CodeRisk production deployment
**Criteria:** Cost, performance, scalability, compatibility, vendor lock-in

---

## Executive Summary

**Recommendation:** Use **Neo4j Aura** for MVP, migrate to **Neptune Serverless** after reaching 1,000+ users

**Rationale:**
- **Neo4j Aura:** Same Cypher queries as local development, zero migration pain, free tier available
- **Neptune:** 70% cheaper at scale ($450/mo vs $2,000/mo @ 1K users), but requires Gremlin or openCypher translation
- **Embedded SQLite:** Best for fully offline/local mode, no cloud option

**Cost Comparison @ 1,000 users:**
- Neo4j Aura Professional: $2,000/month
- AWS Neptune Serverless: $450/month (**77% savings**)
- Neo4j Aura Free: $0/month (up to 50k nodes, 175k edges) ← **Perfect for MVP!**

---

## Detailed Comparison

### Option 1: Neo4j Aura (Managed Neo4j Cloud)

#### Overview
- **Provider:** Neo4j (official cloud offering)
- **Query Language:** Cypher (native)
- **Deployment:** Fully managed, multi-cloud (AWS, Azure, GCP)
- **Compatibility:** 100% compatible with local Neo4j

#### Tiers

| Tier | Nodes | Edges | Storage | RAM | Cost/Month | Use Case |
|------|-------|-------|---------|-----|------------|----------|
| **Free** | 50,000 | 175,000 | 50MB | 1GB | $0 | **MVP, small repos** |
| **Professional** | 200M | 2B | 10GB | 8GB | $65 | Medium repos |
| **Enterprise** | Unlimited | Unlimited | Unlimited | 64GB+ | $2,000+ | Large scale (1K users) |

**Free tier limits (perfect for MVP):**
- 50,000 nodes (≈ 10-15 small repos or 1 large repo)
- 175,000 relationships
- 50MB storage
- 1GB RAM
- Paused after 7 days of inactivity (restarts in 1-2 min)

#### Pros
✅ **Zero migration** - Same Cypher queries as local development
✅ **Free tier** - Great for MVP (supports ~10 small repos or 1 large repo)
✅ **Proven reliability** - Neo4j powers LinkedIn, NASA, UBS
✅ **Auto-scaling** - Scales vertically (upgrade tier as you grow)
✅ **Built-in monitoring** - Query performance, slow query analysis
✅ **Multi-cloud** - Not locked to AWS (can deploy on GCP, Azure)
✅ **Point-in-time recovery** - Automated backups, 7-day retention
✅ **Apoc procedures** - Advanced graph algorithms available

#### Cons
❌ **Expensive at scale** - $2,000/month @ 1K users (16GB RAM tier)
❌ **Free tier limitations** - Pauses after 7 days inactivity (not production-ready)
❌ **Vertical scaling only** - Can't distribute graph across nodes
❌ **Cold start** - Free tier takes 1-2 minutes to wake up
❌ **No serverless** - Always-on pricing (even when idle)

#### Cost Breakdown

```
Free tier (MVP):
- Up to 50k nodes: $0/month
- Suitable for: 10 small repos (5k nodes each) or 1 large repo (50k nodes)

Professional tier (500 users):
- 200M nodes, 2B edges, 10GB storage, 8GB RAM
- Cost: $65/month
- Suitable for: 100 repos × 2M nodes/repo

Enterprise tier (1,000+ users):
- Unlimited nodes/edges
- 16GB RAM minimum
- Cost: $2,000/month (starts at $690, scales to $2K+)
- Suitable for: 1,000 users × 10 repos × 20k nodes/repo = 200M nodes
```

#### Performance Characteristics
- **Query latency:** 10-50ms (1-hop queries)
- **Write throughput:** 10k nodes/sec
- **Bulk import:** 100k nodes/sec (via neo4j-admin)
- **Cold start:** 1-2 minutes (free tier only)

#### Best For
- ✅ **MVP** (use free tier, upgrade later)
- ✅ **<500 users** (Professional tier at $65-200/mo)
- ✅ **Rapid development** (zero migration, same queries)
- ❌ **1,000+ users** (too expensive vs Neptune)

---

### Option 2: AWS Neptune Serverless

#### Overview
- **Provider:** Amazon Web Services
- **Query Language:** Gremlin (Apache TinkerPop) or openCypher (limited)
- **Deployment:** AWS-only, fully managed
- **Compatibility:** Requires query translation from Cypher

#### Pricing Model (Serverless)
**NCUs (Neptune Capacity Units):**
- 1 NCU = 2GB RAM + equivalent vCPU
- Scales from 0.5 NCU to 128 NCUs
- **Auto-scales** based on load (scales to 0.5 NCU when idle)

**Cost Formula:**
```
Monthly cost = (NCU-hours × $0.12) + (Storage GB × $0.10) + (I/O millions × $0.20)
```

#### Cost Examples

| Users | Graph Size | Avg NCUs | Storage | I/O | Cost/Month |
|-------|-----------|----------|---------|-----|------------|
| 100 | 10M nodes, 100M edges | 2 NCUs | 100GB | 50M | $45 |
| 1,000 | 100M nodes, 1B edges | 8 NCUs | 200GB | 500M | $450 |
| 10,000 | 1B nodes, 10B edges | 32 NCUs | 2TB | 5B | $1,100 |

**Idle cost:** 0.5 NCUs × 24h × 30d × $0.12 = **$4.32/month** (when idle!)

#### Pros
✅ **70-80% cheaper** than Neo4j Aura at scale ($450 vs $2,000 @ 1K users)
✅ **True serverless** - Scales to 0.5 NCUs when idle (saves money)
✅ **AWS integration** - Easy to connect to RDS, S3, Lambda, etc.
✅ **High availability** - Multi-AZ replication included
✅ **Global databases** - Read replicas in multiple regions
✅ **ACID transactions** - Full transactional support
✅ **Fast bulk loading** - S3 import for large graphs

#### Cons
❌ **Query translation required** - Cypher → Gremlin or openCypher
❌ **openCypher limitations** - Not all Cypher features supported
❌ **Vendor lock-in** - AWS-only (can't move to GCP/Azure)
❌ **Cold start penalty** - 30-60 second delay when scaling from 0.5 NCUs
❌ **Complexity** - More moving parts (VPC, security groups, endpoints)
❌ **No free tier** - Minimum $4.32/month (0.5 NCUs idle)

#### Cypher vs Gremlin Example

**Cypher (what we use today):**
```cypher
MATCH (f:File)-[r:CO_CHANGED]->(other:File)
WHERE f.file_path = '/path/to/file.ts' AND r.frequency >= 0.7
RETURN other.name, r.frequency
ORDER BY r.frequency DESC
LIMIT 10
```

**Gremlin (Neptune equivalent):**
```groovy
g.V().has('File', 'file_path', '/path/to/file.ts')
  .outE('CO_CHANGED').has('frequency', gte(0.7))
  .order().by('frequency', desc)
  .limit(10)
  .project('name', 'frequency')
    .by(inV().values('name'))
    .by('frequency')
```

**openCypher (Neptune limited support):**
```cypher
// Same as Cypher, but some features missing:
// - No APOC procedures
// - Limited path algorithms
// - Some aggregations not supported
```

#### Migration Effort

**High-level estimate:**
- **Query rewrite:** 40-60 hours (100+ Cypher queries → Gremlin/openCypher)
- **Testing:** 20-30 hours (validate query results match)
- **Performance tuning:** 10-20 hours (optimize Gremlin traversals)
- **Total:** 70-110 hours (~$10,500-16,500 @ $150/hr)

**ROI calculation:**
- Dev cost: $10,500-16,500 (one-time)
- Monthly savings: $1,550/month ($2,000 - $450)
- Break-even: 7-11 months
- **Worth it if you expect 1,000+ users for >1 year**

#### Best For
- ✅ **1,000+ users** (cost savings justify migration effort)
- ✅ **AWS-committed** (already using AWS infrastructure)
- ✅ **High scale** (10K+ users, 1B+ nodes)
- ❌ **MVP** (migration overhead not worth it yet)

---

### Option 3: Embedded SQLite with Graph Extensions

#### Overview
- **Provider:** Self-hosted (no cloud vendor)
- **Query Language:** SQL + custom graph functions
- **Deployment:** Embedded in crisk binary (~/.coderisk/data.db)
- **Compatibility:** Requires complete rewrite

#### Extensions
- **[sqlite-graph](https://github.com/dpapathanasiou/simple-graph)** - Simple graph layer on SQLite
- **[SQLite FTS5](https://www.sqlite.org/fts5.html)** - Full-text search (replace Neo4j text indexes)
- **[SQLite JSON1](https://www.sqlite.org/json1.html)** - JSON property storage

#### Schema Example

```sql
-- Nodes table
CREATE TABLE nodes (
    id TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    properties JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_nodes_label ON nodes(label);

-- Edges table
CREATE TABLE edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    label TEXT NOT NULL,
    from_id TEXT NOT NULL,
    to_id TEXT NOT NULL,
    properties JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_id) REFERENCES nodes(id),
    FOREIGN KEY (to_id) REFERENCES nodes(id)
);

CREATE INDEX idx_edges_from ON edges(from_id);
CREATE INDEX idx_edges_to ON edges(to_id);
CREATE INDEX idx_edges_label ON edges(label);

-- Co-change query example (replaces Cypher)
SELECT
    n2.id,
    json_extract(n2.properties, '$.name') as name,
    json_extract(e.properties, '$.frequency') as frequency
FROM edges e
JOIN nodes n1 ON e.from_id = n1.id
JOIN nodes n2 ON e.to_id = n2.id
WHERE n1.id = 'file:/path/to/file.ts'
  AND e.label = 'CO_CHANGED'
  AND json_extract(e.properties, '$.frequency') >= 0.7
ORDER BY frequency DESC
LIMIT 10;
```

#### Pros
✅ **Zero cloud cost** - All data local
✅ **Zero dependencies** - No Docker, no services, just SQLite
✅ **Instant startup** - No cold start delays
✅ **Simple deployment** - Single binary + single DB file
✅ **Privacy-friendly** - Code never leaves user's machine
✅ **Works offline** - No internet required (except LLM calls)
✅ **Cross-platform** - Works on macOS, Linux, Windows, WASM

#### Cons
❌ **Complete rewrite** - 200+ hours of development
❌ **No graph algorithms** - Manual implementation of path finding, etc.
❌ **Slower queries** - 2-10x slower than Neo4j for graph traversals
❌ **No ACID across files** - SQLite is single-file (limits parallelism)
❌ **Limited scale** - Practical limit ~10M nodes (performance degrades)
❌ **No team sharing** - Single-user only (no multi-tenancy)

#### Cost Comparison

| Aspect | SQLite | Neo4j | Neptune |
|--------|--------|-------|---------|
| **Dev cost** | $30k | $0 | $15k |
| **Monthly cost** | $0 | $2,000 | $450 |
| **Break-even** | Never (upfront cost) | Immediate | 7 months |

**When it makes sense:**
- If 100% of users use local mode (zero cloud adoption)
- If privacy/air-gapped is primary requirement
- If scale is <100 repos per user

#### Performance Comparison

| Operation | SQLite | Neo4j | Neptune |
|-----------|--------|-------|---------|
| 1-hop query | 50-100ms | 10-20ms | 20-40ms |
| 2-hop query | 200-500ms | 30-50ms | 50-100ms |
| Path finding | 1-5s | 100-300ms | 200-500ms |
| Bulk import | 10k nodes/sec | 100k nodes/sec | 50k nodes/sec |

#### Best For
- ✅ **Local-only mode** (no cloud backend)
- ✅ **Single developers** (no team features)
- ✅ **Air-gapped environments** (no internet access)
- ❌ **Cloud mode** (not applicable)
- ❌ **Large teams** (no sharing capabilities)

---

## Recommended Architecture

### Phase 1 (MVP): Neo4j Aura Free Tier

**Timeline:** Weeks 1-12 (launch to 100 users)

**Infrastructure:**
```
┌─────────────────┐
│   crisk CLI     │
└─────────────────┘
        ↓
┌─────────────────┐
│  REST API       │
│  (Railway/Fly)  │
│  $5/month       │
└─────────────────┘
        ↓
┌─────────────────────────────────────┐
│  Neo4j Aura Free                    │
│  - 50k nodes, 175k edges            │
│  - Supports ~10 small repos         │
│  - $0/month                         │
│  - Pauses after 7 days (auto-wake) │
└─────────────────────────────────────┘
```

**Total cost:** $5/month (API hosting only)

**Limitations:**
- Max 50k nodes (≈ 10 small repos or 1 large repo)
- Pauses after 7 days inactivity (1-2 min wake time)
- Single region (us-east-1)

**When to migrate:** >50k nodes OR >100 active users OR >7 days without pause acceptable

---

### Phase 2 (Growth): Neo4j Aura Professional

**Timeline:** Month 4-12 (100-500 users)

**Infrastructure:**
```
┌─────────────────┐
│   crisk CLI     │
└─────────────────┘
        ↓
┌─────────────────┐
│  REST API       │
│  (AWS ECS)      │
│  $50/month      │
└─────────────────┘
        ↓
┌─────────────────────────────────┐
│  Neo4j Aura Professional        │
│  - 200M nodes, 2B edges         │
│  - 8GB RAM, 10GB storage        │
│  - $65-200/month (scales)       │
│  - Always-on                    │
│  - Point-in-time recovery       │
└─────────────────────────────────┘
```

**Total cost:** $115-250/month

**Capacity:**
- 200M nodes (≈ 500 users × 10 repos × 40k nodes/repo)
- Always-on (no pausing)
- Multi-region support

**When to migrate:** >500 users OR monthly bill >$500

---

### Phase 3 (Scale): AWS Neptune Serverless

**Timeline:** Month 12+ (1,000+ users)

**Infrastructure:**
```
┌─────────────────┐
│   crisk CLI     │
└─────────────────┘
        ↓
┌─────────────────────────────┐
│  API Gateway + Lambda       │
│  (AWS)                      │
│  $150/month                 │
└─────────────────────────────┘
        ↓
┌─────────────────────────────────┐
│  AWS Neptune Serverless         │
│  - Unlimited nodes/edges        │
│  - Auto-scales 0.5-128 NCUs     │
│  - $450/month @ 1K users        │
│  - Multi-AZ HA                  │
└─────────────────────────────────┘
```

**Total cost:** $600/month (vs $2,000 with Neo4j Aura)

**Migration effort:**
- 70-110 hours dev time ($10,500-16,500)
- 2-3 weeks timeline
- Zero downtime migration possible (dual-write pattern)

**Capacity:**
- Unlimited scale (tested to 1B+ nodes)
- True serverless (pays only for what you use)
- Global replication

---

## Migration Path: Neo4j → Neptune

### Strategy: Dual-Write Pattern (Zero Downtime)

**Phase 1:** Add Neptune alongside Neo4j (2 weeks)
```go
type DualBackend struct {
    neo4j   *Neo4jBackend
    neptune *NeptuneBackend
}

func (d *DualBackend) CreateNodes(ctx context.Context, nodes []GraphNode) error {
    // Write to both (Neptune in background)
    errChan := make(chan error, 1)
    go func() {
        errChan <- d.neptune.CreateNodes(ctx, nodes)
    }()

    // Primary write (Neo4j)
    if err := d.neo4j.CreateNodes(ctx, nodes); err != nil {
        return err
    }

    // Wait for Neptune (log errors, don't fail)
    if err := <-errChan; err != nil {
        log.Warn("Neptune write failed (non-fatal)", err)
    }

    return nil
}
```

**Phase 2:** Backfill historical data (1 week)
```bash
# Export from Neo4j Aura
neo4j-admin dump --database=neo4j --to=backup.dump

# Convert to Neptune format (Gremlin or openCypher)
./scripts/convert_neo4j_to_neptune.sh backup.dump

# Import to Neptune
aws neptune-db start-loader --source s3://bucket/neptune-data/
```

**Phase 3:** Validate data parity (1 week)
```bash
# Run validation queries on both
./scripts/validate_migration.sh

# Compare results
diff <(query_neo4j.sh) <(query_neptune.sh)
```

**Phase 4:** Flip traffic to Neptune (1 day)
```go
type Backend interface {
    // ...
}

func NewBackend(ctx context.Context, cfg *config.Config) Backend {
    if cfg.UseNeptune {
        return NewNeptuneBackend(ctx, cfg)
    }
    return NewNeo4jBackend(ctx, cfg)
}
```

**Phase 5:** Decommission Neo4j (1 day)
```bash
# Delete Neo4j Aura instance
# Celebrate $1,550/month savings!
```

---

## Final Recommendation

### For MVP (0-100 users): Neo4j Aura Free
- **Cost:** $0/month (plus $5/month API hosting)
- **Effort:** Zero migration (already using Neo4j locally)
- **Timeline:** Immediate (can deploy today)

### For Growth (100-500 users): Neo4j Aura Professional
- **Cost:** $65-200/month
- **Effort:** Zero migration (upgrade button click)
- **Timeline:** Immediate (5 minute upgrade)

### For Scale (1,000+ users): AWS Neptune Serverless
- **Cost:** $450/month (77% savings vs Neo4j)
- **Effort:** 70-110 hours dev time + 4-5 weeks
- **Timeline:** Plan migration at 500 users, execute at 800-900 users

### For Local-Only: Embedded SQLite
- **Cost:** $0/month (no cloud)
- **Effort:** 200+ hours complete rewrite
- **Timeline:** 6-8 weeks
- **Use case:** Privacy-first, air-gapped, single developer

---

## Cost Projection (24 Months)

| Month | Users | Graph DB | Cost/Month | Cumulative |
|-------|-------|----------|------------|------------|
| 1-3 | 0-100 | Neo4j Free | $0 | $0 |
| 4-6 | 100-300 | Neo4j Pro | $65 | $195 |
| 7-12 | 300-800 | Neo4j Pro | $150 | $1,095 |
| 13 | 900 | **Migration** | $150 + $450 | $1,695 |
| 14-24 | 1,000-5,000 | Neptune | $450-800 | $7,695 |

**Total 24-month cost:** $7,695
**vs staying on Neo4j:** $30,000+ (4x more expensive)

**Migration ROI:**
- Dev cost: $15,000 (one-time, month 13)
- Savings: $1,550/month (starting month 14)
- Break-even: 10 months after migration (month 23)
- **Net savings over 24 months: $15,000**

---

## Decision Matrix

| Criteria | Neo4j Aura | AWS Neptune | SQLite |
|----------|-----------|-------------|--------|
| **MVP speed** | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐ |
| **Cost @ 100 users** | ⭐⭐⭐⭐⭐ ($0) | ⭐⭐⭐ ($45) | ⭐⭐⭐⭐⭐ ($0) |
| **Cost @ 1K users** | ⭐ ($2,000) | ⭐⭐⭐⭐⭐ ($450) | ⭐⭐⭐⭐⭐ ($0) |
| **Scalability** | ⭐⭐⭐ (16GB limit) | ⭐⭐⭐⭐⭐ (unlimited) | ⭐⭐ (10M nodes) |
| **Query performance** | ⭐⭐⭐⭐⭐ (native) | ⭐⭐⭐⭐ (Gremlin) | ⭐⭐⭐ (SQL) |
| **Dev effort** | ⭐⭐⭐⭐⭐ (0 hours) | ⭐⭐⭐ (100 hours) | ⭐ (200 hours) |
| **Vendor lock-in** | ⭐⭐⭐⭐ (multi-cloud) | ⭐⭐ (AWS-only) | ⭐⭐⭐⭐⭐ (none) |
| **Team sharing** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐ (local only) |

---

## Next Steps

1. **Week 1-4:** Deploy MVP with Neo4j Aura Free tier
2. **Month 4:** Upgrade to Neo4j Aura Professional when exceeding free tier
3. **Month 10:** Start Neptune migration planning (design queries, test openCypher)
4. **Month 12:** Execute Neptune migration (dual-write → validate → flip traffic)
5. **Month 13+:** Run on Neptune, save $1,550/month

---

**Related Documents:**
- [PHASE_1_DUAL_MODE_ARCHITECTURE.md](PHASE_1_DUAL_MODE_ARCHITECTURE.md)
- [cloud_deployment.md](../01-architecture/cloud_deployment.md)

**Status:** Ready for review
**Decision needed:** Approve Neo4j Aura Free for MVP
