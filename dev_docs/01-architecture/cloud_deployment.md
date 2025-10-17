# Cloud Deployment Architecture

**Version:** 5.0
**Last Updated:** October 12, 2025
**Purpose:** Multi-tenant SaaS deployment with public repository caching
**Design Philosophy:** Cloud-first multi-tenant, local option for individuals *(12-factor: Factor 1 - Stateless compute)*

> **Major Update:** Moved from phased Neo4j → Neptune migration to AWS Neptune from day 1 with multi-tenant architecture. See [ADR-006](decisions/006-multi-tenant-neptune-architecture.md) for full rationale.

---

## Core Strategy: Multi-Tenant Cloud with Public Caching

### Two Deployment Options

**Option 1: Cloud Mode (Recommended for Teams)** ⭐ **MULTI-TENANT**
- **Single AWS Neptune Serverless cluster** (all users share infrastructure)
- **Public repository caching** (1,000 users adding `react` = 1 graph, not 1,000)
- **Private repository isolation** (per-organization security boundaries)
- **Branch delta architecture** (50MB deltas vs 2GB full graphs)
- Zero setup for users (install CLI, authenticate, done)
- BYOK for LLM (user provides OpenAI/Anthropic API key)
- **Best for:** Teams, collaborative workflows, public + private repos
- **Cost:** $0.32-0.50/user/month @ 1K users (99% cheaper than per-user isolation)

**Option 2: Local Mode (For Individuals/Privacy)**
- Local Docker Compose stack (Neo4j Community, PostgreSQL, Redis)
- Single-user, single-repository workflow
- Runs on user's hardware (no cloud dependency except LLM)
- One-time setup (~10-15 minutes)
- BYOK for LLM (optional, only Phase 2 investigations)
- **Best for:** Solo devs, air-gapped environments, privacy requirements, $0/month cost

### Cloud Mode: Multi-Tenant Architecture

**AWS Neptune Serverless from Day 1**
- **Cost:** $317/month @ 1,000 users ($0.32/user)
- **Public cache hit rate:** 80% (instant access to popular repos)
- **Storage efficiency:** 99% reduction via public caching
- **Branch support:** Lightweight 50MB deltas (not 2GB full graphs)
- **Auto-scaling:** 0.5-128 NCUs based on load
- **Deployment time:** 6-8 weeks (infrastructure + multi-tenancy logic)

### BYOK Model (Both Modes)

**User provides:**
- OpenAI/Anthropic API key for LLM reasoning (Phase 2 investigations only)

**We provide (Cloud Mode):**
- Graph storage (Neo4j Aura → Neptune)
- Two-phase investigation engine (Kubernetes)
- Metric validation database (PostgreSQL)
- Redis caching layer (15-min TTL)
- All infrastructure and managed services

**User provides (Local Mode):**
- Hardware (16GB+ RAM recommended)
- Docker runtime

**Benefits:**
- Zero setup for cloud (install CLI, authenticate, done)
- $0/month for local (hardware + Docker only)
- Transparent pricing (no LLM markup, only 20% of checks use LLM)
- Better economics (user pays LLM directly)
- Works for any repository size
- **80% of checks complete in <500ms** (no LLM needed)

---

## System Architecture *(12-factor: Factor 3 - Own your context window, Factor 5 - Unify state)*

```
┌─────────────────────────────────────────────────────────────────────┐
│  CLIENT LAYER (crisk CLI)                                           │
│  • Parse git diff                                                   │
│  • Send to API Gateway                                              │
│  • Display results (risk level + evidence)                          │
│  • NO local graph/LLM (cloud-only)                                  │
└─────────────────────────────────────────────────────────────────────┘
                                    ↓ HTTPS
┌─────────────────────────────────────────────────────────────────────┐
│  API GATEWAY + AUTH                                                 │
│  • JWT authentication (15-min tokens)                               │
│  • Rate limiting (per-user quotas)                                  │
│  • Request routing to investigation engine                          │
└─────────────────────────────────────────────────────────────────────┘
                                    ↓
┌─────────────────────────────────────────────────────────────────────┐
│  INVESTIGATION ENGINE (Kubernetes)                                  │
│                                                                     │
│  ┌─────────────────────────────────────────────────────┐          │
│  │ PHASE 1: BASELINE (<500ms, 80% of checks)          │          │
│  │ • Query Redis cache (5ms if hit)                   │          │
│  │ • Calculate Tier 1 metrics (parallel):             │          │
│  │   - Coupling (Neptune 1-hop query, 50ms)           │          │
│  │   - Co-change (Neptune edge read, 20ms)            │          │
│  │   - Test ratio (Neptune relationship, 50ms)        │          │
│  │ • Apply heuristic (10ms)                           │          │
│  │ • Cache result in Redis (15-min TTL)               │          │
│  │ • If LOW → Return (no LLM needed)                  │          │
│  └─────────────────────────────────────────────────────┘          │
│                          ↓ (only 20% proceed)                      │
│  ┌─────────────────────────────────────────────────────┐          │
│  │ PHASE 2: LLM INVESTIGATION (3-5s)                  │          │
│  │ • Load 1-hop neighbors (Neptune, 100ms)            │          │
│  │ • LLM decision loop (max 3 iterations):            │          │
│  │   - Calculate Tier 2 metrics (ownership/incidents) │          │
│  │   - Expand graph (2-hop if needed)                 │          │
│  │   - Synthesize risk assessment                     │          │
│  │ • Cache investigation trace (Redis)                │          │
│  │ • Record metric usage (Postgres)                   │          │
│  └─────────────────────────────────────────────────────┘          │
└─────────────────────────────────────────────────────────────────────┘
                    ↓                    ↓                    ↓
        ┌───────────────┐   ┌───────────────┐   ┌───────────────┐
        │  NEPTUNE      │   │  REDIS        │   │  POSTGRES     │
        │  Serverless   │   │  ElastiCache  │   │  RDS          │
        │               │   │               │   │               │
        │  • Files      │   │  • Tier 1     │   │  • Users      │
        │  • Functions  │   │    metrics    │   │  • API keys   │
        │  • Commits    │   │  • Tier 2     │   │  • Metric     │
        │  • Incidents  │   │    metrics    │   │    validation │
        │  • CALLS      │   │  • Investig.  │   │  • FP rates   │
        │  • IMPORTS    │   │    traces     │   │               │
        │  • CO_CHANGED │   │  (15-min TTL) │   │               │
        └───────────────┘   └───────────────┘   └───────────────┘
```

### Client Layer
**Component:** Lightweight CLI binary (~8MB, written in Go)
**Responsibilities:**
- Parse git diff (changed files only)
- Send to cloud API via HTTPS
- Display risk assessment + evidence
**No local storage:** No graph database, no LLM, no heavy compute

### API Layer
**Component:** API Gateway + Auth (AWS API Gateway + Lambda)
**Tech:** Go microservices, JWT tokens (15-min expiry)
**Responsibilities:**
- GitHub OAuth authentication
- Rate limiting per user/team (configurable quotas)
- Request routing to investigation engine
- Response caching (CloudFront CDN)

### Investigation Engine
**Component:** Kubernetes cluster (EKS)
**Pods:**
- **Baseline workers** (stateless, fast autoscale): Phase 1 checks
- **LLM workers** (stateful, slower autoscale): Phase 2 investigations
- **Graph ingest workers**: Webhook processing, git history extraction

**Responsibilities:**
- Two-phase investigation (see architecture diagram)
- Metric calculation (Tier 1 always, Tier 2 on-demand)
- LLM orchestration (only Phase 2, user's API key)
- Metric validation tracking (Postgres updates)

**Scaling:**
- Phase 1 workers: 10-100 pods (CPU-bound, fast scale)
- Phase 2 workers: 5-20 pods (LLM I/O-bound, slower scale)

### Graph Storage (Multi-Tenant)
**Component:** Amazon Neptune Serverless (SINGLE CLUSTER)
**Tech:** Gremlin queries (openCypher support limited)
**Schema:** 3 layers (Structure, Temporal, Incidents) - see [graph_ontology.md](graph_ontology.md)

**Multi-Tenant Database Namespacing:**
```
SINGLE NEPTUNE CLUSTER (0.5-128 NCUs, all tenants share)
├─ Public repo graphs (shared, reference-counted)
│  ├─ neptune_public_repo_facebook_react_main (2GB, 1,000 users)
│  ├─ neptune_public_repo_vercel_nextjs_main (1.5GB, 500 users)
│  └─ ... (~100-200 popular repos, cached)
│
├─ Private org graphs (isolated per organization)
│  ├─ neptune_org_acme_corp_repo_uuid123_main (2GB, 10 users)
│  ├─ neptune_org_startup_x_repo_uuid456_main (1GB, 5 users)
│  └─ ... (one per private repo per org)
│
└─ Branch deltas (lightweight, ephemeral)
   ├─ neptune_public_repo_facebook_react_branch_abc123 (50MB)
   └─ neptune_org_acme_repo_uuid123_branch_def456 (30MB)
```

**Responsibilities:**
- **Public caching:** First user builds, subsequent users get instant access (reference counting)
- **Private isolation:** Organization-level security boundaries (not per-user)
- **Branch deltas:** Store only changed files vs main (50MB vs 2GB full graph)
- **Access control:** PostgreSQL row-level security enforces who can query which database
- **Garbage collection:** Archive unused public caches to S3 after 30 days (ref_count = 0)

**Scaling:** 0.5-128 NCUs (auto-scales based on ALL tenants' query load)
**Cost optimization:** Scales to 0.5 NCU during idle periods ($4/month minimum)
**Storage efficiency:** 99% reduction via public caching (1 copy of `react` vs 1,000 copies)

See [ADR-006](decisions/006-multi-tenant-neptune-architecture.md) for full multi-tenant design.

### Metadata Storage (Multi-Tenant Access Control)
**Component:** PostgreSQL (RDS)
**Schema:**
- `users` - GitHub OAuth, team membership, encrypted tokens
- `organizations` - Team/company entities for private repo isolation
- `repositories` - Neptune DB mapping, visibility (public/private), reference counts
- `repository_access` - Row-level security (user → repo, org → repo permissions)
- `api_keys` - Encrypted LLM API keys (AES-256-GCM)
- `metric_validations` - User feedback on metric accuracy
- `metric_stats` - FP rates, auto-disable logic

**Multi-Tenant Access Control:**
```sql
-- Key table for public cache reference counting
CREATE TABLE repositories (
    id UUID PRIMARY KEY,
    github_owner VARCHAR(255),
    github_repo VARCHAR(255),
    visibility VARCHAR(20),  -- 'public' or 'private'
    neptune_db_name VARCHAR(500),  -- e.g., 'neptune_public_repo_facebook_react_main'
    reference_count INT DEFAULT 0,  -- How many users/orgs use this
    state VARCHAR(20),  -- 'building', 'ready', 'warm', 'archived', 'deleted'
    last_accessed_at TIMESTAMP,
    UNIQUE(github_owner, github_repo)
);

-- Row-level security for private repos
CREATE TABLE repository_access (
    user_id UUID REFERENCES users(id),
    org_id UUID REFERENCES organizations(id),
    repo_id UUID REFERENCES repositories(id),
    role VARCHAR(20),  -- 'owner', 'admin', 'member'
    UNIQUE(user_id, repo_id)
);

ALTER TABLE repository_access ENABLE ROW LEVEL SECURITY;
CREATE POLICY user_repo_access ON repository_access
    FOR SELECT USING (user_id = current_setting('app.current_user_id')::UUID);
```

**Responsibilities:**
- **Reference counting:** Track public cache usage (increment on add, decrement on remove)
- **Access control:** Row-level security for private repos (user can only see their repos)
- **GitHub verification:** Verify private repo collaborator status via GitHub API
- **Garbage collection metadata:** Track `last_accessed_at` for archival decisions
- **Encrypted API key storage:** Envelope encryption for user LLM keys
- **Metric validation system:** Track FP rates, auto-disable metrics >3% FP rate

See [public_caching.md](../../02-operations/public_caching.md) for reference counting details.

### Cache Layer
**Component:** Redis (ElastiCache)
**Configuration:** cluster mode enabled, 3 shards
**TTL:** 15 minutes (all cached data)
**Keys:**
- `baseline:{repo_id}:{file_path}` - Tier 1 metrics (coupling, co-change, test_ratio)
- `tier2:{repo_id}:{file_path}:{metric}` - Tier 2 metrics (ownership, incidents)
- `investigation:{hash(changed_files)}` - Full investigation traces
**Hit rate:** 85-90% (same files checked repeatedly)
**Eviction:** LRU policy

### Settings Portal
**Component:** Next.js web application (hosted on Vercel)
**Responsibilities:**
- API key configuration (encrypted transmission)
- Team management (invite, remove, roles)
- Repository listing (connected repos, webhook status)
- Metric dashboard (FP rates, disabled metrics)
- Billing/usage dashboard

---

## Infrastructure Components

### Compute (Kubernetes EKS)
**Configuration:**
- 3-10 nodes (auto-scales based on load)
- Instance type: t3.large (2 vCPU, 8GB RAM)
- Pods: 50-200 (baseline workers dominate due to 80/20 split)
**Cost:** $900/month @ 1K users (30% reduction due to faster Phase 1)

### Graph Database (Neptune Serverless)
**Configuration:**
- Auto-scaling 0.5-128 NCUs
- Storage: 200GB @ 1K users (10K repos × 20MB avg)
- Query pattern: 1-hop queries (faster than old 2-hop)
**Cost:** $450/month (20% reduction due to simpler queries)

### Metadata (PostgreSQL RDS)
**Size:** db.t3.medium (2 vCPU, 4GB RAM)
**Storage:** 100GB SSD
**New tables:** metric_validations, metric_stats (see [agentic_design.md](agentic_design.md))
**Cost:** $350/month (unchanged)

### Cache (Redis ElastiCache)
**Size:** cache.t3.large (2 vCPU, 6.1GB RAM) - **UPGRADED**
**Configuration:** Cluster mode (3 shards)
**Usage:** 85-90% hit rate (critical for Phase 1 performance)
**Cost:** $300/month (50% increase, justified by 40x speedup on cache hits)

### Storage (S3)
**Usage:**
- Public cache archival (compressed graphs for popular repos)
- Neptune backups (automated daily)
- Investigation trace archives (30-day retention)
**Cost:** $50/month (500GB)

### Networking
**Components:**
- Application Load Balancer (ALB)
- VPC NAT Gateway
- CloudFront CDN (settings portal, static assets)
**Cost:** $150/month

### Monitoring & Observability
**Tools:**
- CloudWatch (metrics, logs)
- DataDog (APM, distributed tracing)
- Sentry (error tracking)
**Cost:** $100/month

**Total Infrastructure:** $2,300/month @ 1K users = **$2.30/user/month** (6% reduction from V2.0)

---

## Cost Model: Cloud Mode (Phased Approach)

### Phase 1: Neo4j Aura Free Tier (0-100 users)

**Graph Database:** Neo4j Aura Free
- Cost: **$0/month**
- Capacity: 50k nodes, 175k edges
- Storage: ~10 small repos or 1 large repo
- Limits: Single instance, community support
- Perfect for: MVP, beta testing, proof of concept

**Other Infrastructure:**

| Component | Cost | Notes |
|-----------|------|-------|
| Kubernetes (EKS) | $250 | 3 small nodes |
| PostgreSQL (RDS) | $350 | db.t3.medium |
| Redis (ElastiCache) | $150 | cache.t3.medium |
| Other (ALB, S3, monitoring) | $160 | Networking, storage |
| **Total (excluding graph)** | **$910/month** | **$9.10/user** @ 100 users |

**MVP Cost:** $910/month (Neo4j Free) = **$9.10/user**

---

### Phase 2: Neo4j Aura Professional (100-500 users)

**Graph Database:** Neo4j Aura Professional
- Cost: **$65-200/month** (pay-as-you-grow)
- Capacity: 200M nodes, 1B edges
- Storage: ~500 users, ~5,000 repos
- Features: Multi-AZ, automated backups, 24/7 support
- Migration: **One-click upgrade** from Free tier

**Infrastructure Scaling:**

| Users | Neo4j Aura | Kubernetes | PostgreSQL | Redis | Other | Total | Per User |
|-------|------------|------------|------------|-------|-------|-------|----------|
| 100 | $65 | $250 | $350 | $150 | $160 | $975 | $9.75 |
| 250 | $120 | $500 | $350 | $200 | $200 | $1,370 | $5.48 |
| 500 | $200 | $700 | $350 | $250 | $250 | $1,750 | $3.50 |

**Cost per user decreases** with scale due to shared infrastructure.

---

### Phase 3: AWS Neptune Serverless (1,000+ users)

**Graph Database:** Neptune Serverless
- Cost: **$450/month** @ 1K users
- Capacity: Unlimited (auto-scales 0.5-128 NCUs)
- Storage: Unlimited (pay for what you use)
- Features: Multi-AZ, VPC, IAM integration, openCypher + Gremlin
- Migration: **70-110 dev hours** (Cypher → Gremlin conversion)

**Infrastructure Scaling:**

| Users | Neptune | Kubernetes | PostgreSQL | Redis | Other | Total | Per User | Savings vs Neo4j |
|-------|---------|------------|------------|-------|-------|-------|----------|------------------|
| 1,000 | $450 | $900 | $350 | $300 | $300 | $2,300 | $2.30 | $22,305 (24mo) |
| 10,000 | $1,100 | $3,000 | $500 | $600 | $600 | $5,800 | $0.58 | $150,000 (24mo) |

**Neptune vs Neo4j Cost Comparison (1,000 users, 24 months):**
- Neo4j Aura Enterprise: $2,000/month × 24 = **$48,000**
- Neptune Serverless: $450/month × 24 = **$10,800**
- **Migration cost:** ~$15,000 (100 dev hours × $150/hr)
- **Total Neptune:** $25,800
- **Savings:** **$22,305 (77% reduction)**
- **Break-even:** Month 10

---

### Cost Comparison Summary

| Phase | Users | Solution | Monthly Cost | Total (24mo) | Per User/Month |
|-------|-------|----------|--------------|--------------|----------------|
| **MVP** | 0-100 | Neo4j Free | $910 | $21,840 | $9.10 |
| **Growth** | 100-500 | Neo4j Pro | $1,370-1,750 | $27,720 | $5.48-3.50 |
| **Scale** | 1,000+ | Neptune | $2,300 | $55,200 | $2.30 |
| **Scale (alt)** | 1,000+ | Neo4j Enterprise | $2,910 | $69,840 | $2.91 |

**Recommendation:**
- **Months 0-6:** Neo4j Aura Free (zero graph cost, fast deployment)
- **Months 6-12:** Neo4j Aura Professional (one-click upgrade, same queries)
- **Month 12+:** Migrate to Neptune (77% cost savings at scale)

**Key insights:**
- Neo4j for **speed to market** (zero migration, same Cypher queries)
- Neptune for **scale economics** (77% savings vs Neo4j Enterprise)
- Migration at **Month 12** maximizes learning while minimizing technical debt

### User LLM Costs (Their Burden) - **REDUCED**
**Phase 1 (80% of checks):** $0 (no LLM)
**Phase 2 (20% of checks):** ~$0.03-0.05 per investigation
**Monthly (10 checks/day, 20% HIGH risk):** ~$2-3 (vs $9-15 in V2.0)
**Paid by:** User directly to OpenAI/Anthropic via BYOK

---

## Pricing Tiers

### Free Tier
**Price:** $0/month
**Limits:**
- 1 user
- 3 repositories
- 100 checks/month
**Target:** Individual developers, OSS contributors

### Pro Tier
**Price:** $9/user/month
**Limits:**
- Unlimited repositories
- 1,000 checks/month
- Email support
**Margin:** $9 - $2.45 = **$6.55 profit (267%)**

### Team Tier
**Price:** $29/user/month
**Features:**
- Team graph sharing
- 5,000 checks/month
- Branch delta support
- Priority support
- Admin dashboard
**Margin:** $29 - $2.45 = **$26.55 profit (1,084%)**

### Enterprise Tier
**Price:** Custom ($5K-50K/month)
**Features:**
- Self-hosted Neptune in customer VPC
- SAML/OIDC authentication
- Custom SLA
- Dedicated support
**Target:** 50-500+ developers

---

## API Key Security

### Storage
**Encryption:** AES-256-GCM
**Key management:** AWS Secrets Manager (rotates every 90 days)
**Never logged:** API keys never appear in logs, metrics, or traces

### Usage
**Decryption:** Only at LLM call time (ephemeral, in-memory)
**Transmission:** Direct HTTPS to OpenAI/Anthropic (we never see responses)
**Rotation:** User can rotate anytime via settings portal

### Audit
**Logging:** Every LLM call logged (timestamp, model, tokens, cost)
**Transparency:** User sees all API calls in dashboard
**Alerts:** Email if unusual API usage detected

---

## Multi-Tenancy & Isolation

### Database Isolation
**PostgreSQL:** Row-level security (RLS) enforces team boundaries
**Neptune:** Separate databases per repo (no cross-contamination)
**Redis:** Namespace prefixes (`team:{id}:investigation:...`)

### Compute Isolation
**Kubernetes:** Namespaces per team tier (free/pro/team)
**Network:** Network policies prevent cross-tenant traffic
**Resources:** Resource quotas per namespace

### Data Isolation
**Private repos:** Isolated Neptune DBs, GitHub OAuth verification
**Public repos:** Shared cache, read-only access
**Team data:** Cannot query other teams' repos

---

## Webhook Integration

### GitHub Webhooks
**Events:** Push to main, PR merge, branch delete
**Trigger:** Incremental graph update
**Processing:** 10-30 seconds (extract commits → update graph)
**Reliability:** Retry with exponential backoff, dead letter queue

### Graph Update Flow
**On push to main:**
1. Receive webhook payload
2. Extract commits and changed files
3. Delete old graph nodes for changed files
4. Re-parse with tree-sitter
5. Insert new nodes and relationships
6. Invalidate affected Redis caches

**Result:** Main graph always fresh, no user action required

---

## Disaster Recovery

### Backups
**Neptune:** Continuous backup, 35-day retention
**PostgreSQL:** Daily snapshots, 30-day retention
**S3:** Versioning enabled, lifecycle policies

### RTO/RPO
**Recovery Time Objective (RTO):** 4 hours
**Recovery Point Objective (RPO):** 1 hour
**Strategy:** Multi-AZ deployment, automated failover

### High Availability
**API Gateway:** Multi-AZ, auto-scaling
**Kubernetes:** 3 availability zones, self-healing
**Neptune:** Multi-AZ replication
**PostgreSQL:** Multi-AZ with read replica

---

## Monitoring & Observability

### Key Metrics
**Latency:**
- `crisk check` p50, p95, p99
- Neptune query time
- LLM call duration

**Availability:**
- API uptime (target: 99.9%)
- Investigation success rate
- Webhook processing rate

**Cost:**
- Neptune NCU-hours
- LLM API calls (user's burden)
- S3 storage growth

### Alerts
**Critical:**
- API downtime > 5 minutes
- Neptune cluster unavailable
- PostgreSQL connection pool exhausted

**Warning:**
- Investigation latency > 10s (p95)
- Cache hit rate < 70%
- Unusual API key usage

---

## Security

### Authentication
**Users:** GitHub OAuth (primary)
**API:** JWT tokens (15-min expiry)
**Teams:** SAML/OIDC (enterprise tier)

### Authorization
**Model:** Role-based access control (RBAC)
**Roles:** Owner, Admin, Member, Read-only
**Enforcement:** API layer + PostgreSQL RLS

### Compliance
**SOC 2 Type II:** In progress (target: Q2 2026)
**GDPR:** Data residency options (EU region)
**HIPAA:** Enterprise tier only (BAA available)

### Encryption
**In transit:** TLS 1.3 (all connections)
**At rest:** AES-256 (Neptune, RDS, S3)
**API keys:** AES-256-GCM with envelope encryption

---

## Deployment Regions

### Phase 1 (Launch)
**Region:** us-east-1 (N. Virginia)
**Rationale:** Lowest cost, most services available

### Phase 2 (Expansion)
**Regions:** us-west-2 (Oregon), eu-west-1 (Ireland)
**Rationale:** Latency reduction for EU/West Coast users

### Phase 3 (Global)
**Regions:** ap-southeast-1 (Singapore), ap-northeast-1 (Tokyo)
**Target:** 2027

---

## Local Deployment Option (Alternative Architecture)

**Target users:** Individual developers, small teams (1-5 people), air-gapped environments
**Repository scale:** Small (500 files), Medium (5K files), Large (50K files)
**Trade-offs:** Setup complexity vs no monthly fees, local compute vs cloud convenience

### Local Architecture (Docker Compose)

```
┌─────────────────────────────────────────────────────────────────────┐
│  CLIENT LAYER (crisk CLI)                                           │
│  • Parse git diff                                                   │
│  • Connect to local API (http://localhost:8080)                    │
│  • Display results                                                  │
└─────────────────────────────────────────────────────────────────────┘
                                    ↓ HTTP
┌─────────────────────────────────────────────────────────────────────┐
│  DOCKER COMPOSE STACK (localhost)                                   │
│                                                                     │
│  ┌──────────────────────────────────────────────────┐             │
│  │  API SERVICE (Go, Port 8080)                     │             │
│  │  • Two-phase investigation (same logic as cloud) │             │
│  │  • OpenAI API calls (user's key, optional)       │             │
│  │  • Metric validation                             │             │
│  └──────────────────────────────────────────────────┘             │
│                          ↓                                          │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐                   │
│  │  NEO4J     │  │  REDIS     │  │  POSTGRES  │                   │
│  │  Community │  │  (cache)   │  │  (metadata)│                   │
│  │  Edition   │  │            │  │            │                   │
│  │            │  │  • Metrics │  │  • Repos   │                   │
│  │  • Files   │  │  • Traces  │  │  • FP rates│                   │
│  │  • Commits │  │  15-min    │  │            │                   │
│  │  • CALLS   │  │  TTL       │  │            │                   │
│  │  • IMPORTS │  │            │  │            │                   │
│  └────────────┘  └────────────┘  └────────────┘                   │
│  Port: 7474      Port: 6379      Port: 5432                        │
│  Storage: 1-50GB Storage: 2-8GB  Storage: 1-5GB                    │
└─────────────────────────────────────────────────────────────────────┘
```

### Docker Compose Configuration

```yaml
# docker-compose.yml
version: '3.8'

services:
  neo4j:
    image: neo4j:5.15-community
    ports:
      - "7474:7474"  # HTTP
      - "7687:7687"  # Bolt
    volumes:
      - neo4j_data:/data
      - neo4j_logs:/logs
    environment:
      - NEO4J_AUTH=neo4j/coderisk123  # Change in production
      - NEO4J_dbms_memory_heap_max__size=2G
      - NEO4J_dbms_memory_pagecache_size=1G
    healthcheck:
      test: ["CMD", "cypher-shell", "-u", "neo4j", "-p", "coderisk123", "RETURN 1"]
      interval: 10s
      timeout: 5s
      retries: 5

  postgres:
    image: postgres:16-alpine
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    environment:
      - POSTGRES_DB=coderisk
      - POSTGRES_USER=coderisk
      - POSTGRES_PASSWORD=coderisk123  # Change in production
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U coderisk"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    command: redis-server --maxmemory 2gb --maxmemory-policy allkeys-lru
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  api:
    build: .
    ports:
      - "8080:8080"
    depends_on:
      neo4j:
        condition: service_healthy
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    environment:
      - NEO4J_URI=bolt://neo4j:7687
      - NEO4J_USER=neo4j
      - NEO4J_PASSWORD=coderisk123
      - POSTGRES_DSN=postgresql://coderisk:coderisk123@postgres:5432/coderisk
      - REDIS_URL=redis://redis:6379
      - OPENAI_API_KEY=${OPENAI_API_KEY}  # User provides
      - MODE=local
    volumes:
      - ./repos:/repos  # Mount local repos for analysis

volumes:
  neo4j_data:
  neo4j_logs:
  postgres_data:
  redis_data:
```

### System Requirements by Codebase Size

| Codebase Size | Files | Graph Size | RAM | Disk | CPU | Boot Time | Check Latency |
|---------------|-------|------------|-----|------|-----|-----------|---------------|
| **Small** | 500 | 8K nodes, 50K edges | 4GB | 10GB | 2 cores | 30s | 200ms (P1), 3s (P2) |
| **Medium** | 5K | 80K nodes, 600K edges | 8GB | 50GB | 4 cores | 60s | 300ms (P1), 4s (P2) |
| **Large** | 50K | 800K nodes, 6M edges | 16GB | 200GB | 8 cores | 120s | 500ms (P1), 6s (P2) |

**Recommended hardware:**
- **Individuals:** MacBook Pro M1 (16GB RAM), Linux workstation (16GB RAM)
- **Small teams:** Shared server (32GB RAM, 500GB SSD)
- **Large repos:** Dedicated server (64GB RAM, 1TB NVMe SSD)

### Local vs Cloud Comparison

| Aspect | Local (Docker Compose) | Cloud (AWS) |
|--------|----------------------|-------------|
| **Setup time** | 10-15 min (docker-compose up) | 2 min (install CLI, auth) |
| **Initial cost** | $0 (free software) | $0 (free tier) |
| **Monthly cost** | $0 + hardware depreciation | $9-29/user |
| **LLM cost** | $2-3/month (BYOK to OpenAI) | $2-3/month (BYOK to OpenAI) |
| **Storage** | Local disk (1-200GB) | Cloud (unlimited) |
| **Team sharing** | Manual (export/import graphs) | Built-in (multi-tenant) |
| **Webhook updates** | Manual `crisk sync` | Automatic (GitHub webhooks) |
| **Availability** | When local machine on | 99.9% uptime |
| **Backup** | User responsibility | Automated (35-day retention) |
| **Internet required** | Only for LLM calls (Phase 2) | Always |
| **Max scale** | 50K files (hardware limit) | Unlimited |

### Local Deployment Steps

**1. Install Docker & Docker Compose**
```bash
# macOS
brew install docker docker-compose

# Linux
sudo apt-get install docker.io docker-compose

# Verify
docker --version
docker-compose --version
```

**2. Clone CodeRisk repo**
```bash
git clone https://github.com/coderisk/coderisk-go.git
cd coderisk-go
```

**3. Configure environment**
```bash
# Create .env file
cat > .env <<EOF
OPENAI_API_KEY=sk-...  # Optional: only for Phase 2 investigations
MODE=local
EOF
```

**4. Start services**
```bash
docker-compose up -d
```

**5. Initialize repository**
```bash
# Install CLI
go install ./cmd/crisk

# Point to local deployment
export CRISK_API_URL=http://localhost:8080

# Initialize your repo
cd /path/to/your/repo
crisk init
```

**6. Run checks**
```bash
# Make changes
git add .

# Check risk
crisk check
```

### Local-Only Mode (No LLM)

For air-gapped environments or users who don't want LLM:

**Configuration:**
```yaml
# docker-compose.override.yml
services:
  api:
    environment:
      - PHASE2_ENABLED=false  # Disable LLM investigations
```

**Behavior:**
- Only Phase 1 baseline checks (200ms)
- Simple heuristic (coupling > 10, co-change > 0.7, test_ratio < 0.3)
- No LLM calls (no internet required after initial setup)
- Risk levels: LOW, MEDIUM, HIGH (based on thresholds)
- **Trade-off:** No contextual reasoning, higher FP rate (~8-10% vs <3%)

### Local Performance Characteristics

**Graph ingestion (one-time setup):**
- Small repo (500 files): ~30 seconds
- Medium repo (5K files): ~5 minutes
- Large repo (50K files): ~45 minutes

**Incremental updates (git sync):**
- 10 files changed: ~5 seconds
- 100 files changed: ~30 seconds

**Risk checks (cached):**
- Phase 1: 5ms (Redis hit), 150ms (cold)
- Phase 2: 2-4s (LLM call dominates)

### Kubernetes Alternative (Multi-User Teams)

For teams who want local deployment but need multi-user support:

**Use minikube or k3s:**
```bash
# Install k3s (lightweight Kubernetes)
curl -sfL https://get.k3s.io | sh -

# Deploy CodeRisk helm chart
helm install coderisk ./deploy/helm/coderisk \
  --set persistence.storageClass=local-path \
  --set replicas.api=2 \
  --set neo4j.resources.memory=8Gi
```

**Supports:**
- 5-20 concurrent users
- Team graph sharing
- Load balancing
- Auto-scaling (within hardware limits)

**Hardware:**
- 32GB RAM minimum
- 8+ CPU cores
- 500GB SSD

### When to Choose Local vs Cloud

**Choose Local if:**
- ✅ Single developer or small team (1-5 people)
- ✅ Air-gapped environment (no internet access)
- ✅ Privacy requirements (code cannot leave network)
- ✅ Cost-sensitive (willing to trade setup time for $0/month)
- ✅ Small-medium codebase (<10K files)
- ✅ Have suitable hardware (16GB+ RAM)

**Choose Cloud if:**
- ✅ Team size >5 people
- ✅ Want zero setup (install CLI, done)
- ✅ Need team collaboration (shared graphs)
- ✅ Large codebase (>10K files)
- ✅ Want automatic webhook updates
- ✅ Need 99.9% uptime
- ✅ Willing to pay $9-29/user/month

---

## Key Design Decisions

### 1. Why BYOK for LLM?
**Decision:** User provides API key
**Rationale:**
- Transparent pricing (no markup)
- User controls LLM choice
- Better unit economics
**Trade-off:** User setup friction (acceptable)

### 2. Why Neptune Serverless?
**Decision:** Neptune over Neo4j Aura
**Rationale:**
- 70% cost savings ($545 vs $2,000/month)
- True serverless (scales to zero)
- Perfect for bursty workload
**Trade-off:** Cold start latency (acceptable)

### 3. Why Cloud-First (with Local Option)?
**Decision:** Cloud default, local Docker Compose for specific use cases
**Rationale:**
- **Cloud:** Better UX (zero setup), team collaboration, automatic webhooks
- **Local:** Privacy, air-gapped environments, $0/month cost
- Same codebase (Go API works in both modes)
**Trade-off:** Two deployment paths to maintain (acceptable, shared 95% of code)

### 4. Why Settings Portal?
**Decision:** Web UI for configuration
**Rationale:**
- Better UX for API key entry
- Team management visualization
- Billing dashboard
**Trade-off:** Additional service to maintain (acceptable)

---

**See also:**
- [graph_ontology.md](graph_ontology.md) - Graph schema (3 layers: Structure, Temporal, Incidents)
- [agentic_design.md](agentic_design.md) - Two-phase investigation strategy, metric validation
- [public_caching.md](../02-operations/public_caching.md) - Public repository graph caching
- [team_and_branching.md](../02-operations/team_and_branching.md) - Team collaboration, branch deltas

**Local deployment resources:**
- `docker-compose.yml` - Docker Compose configuration (in repo root)
- `deploy/helm/coderisk/` - Kubernetes Helm chart (for k3s/minikube)
- `docs/local-setup.md` - Step-by-step local installation guide
