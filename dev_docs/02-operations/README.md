# Operational Strategies Documentation

**Purpose:** How the system operates at scale, cost optimization, multi-tenancy patterns

> **📘 For AI agents:** Before creating/updating operational docs, read [DOCUMENTATION_WORKFLOW.md](../DOCUMENTATION_WORKFLOW.md) to determine if this is the right location and which document to update.

---

## What Goes Here

**Scaling Strategies:**
- Team graph sharing
- Multi-branch support
- Public repository caching
- Load balancing and auto-scaling

**Cost Optimization:**
- Infrastructure cost modeling
- Caching strategies
- Resource utilization
- Garbage collection

**Multi-Tenancy:**
- Data isolation patterns
- Access control
- Resource quotas
- Billing attribution

**Lifecycle Management:**
- Graph lifecycle (creation, updates, deletion)
- Cache eviction policies
- Backup and archival
- Reference counting

---

## Current Documents

- **[team_and_branching.md](team_and_branching.md)** - Team sharing (90% cost reduction), branch delta strategy
- **[public_caching.md](public_caching.md)** - Public repo cache (99% storage reduction), reference counting, GC

---

## Document Guidelines

### When to Add Here
- Scaling patterns and strategies
- Cost optimization approaches
- Operational runbooks
- Multi-tenant isolation patterns
- Cache management strategies

### When NOT to Add Here
- Initial architecture design (goes to 01-architecture/)
- Implementation tutorials (goes to 03-implementation/)
- Product requirements (goes to 00-product/)

### Format
- **Operations-focused** - How things work at scale
- **Data-driven** - Include cost models, scaling metrics
- **Practical** - Real-world operational considerations

---

## Key Concepts

### Team Sharing
- One graph per repository (not per user)
- 90% cost reduction vs per-user graphs
- Shared base graph + individual branch deltas
- Row-level security for access control

### Branch Deltas
- 98% smaller than full graphs (10-50MB vs 2GB)
- Created on-demand, deleted after merge
- Merged with base at query time (+100ms latency)
- Cached in Redis for 15 minutes

### Public Caching
- Shared cache for public repos (React, Kubernetes, etc.)
- Reference counting for lifecycle management
- 99% storage reduction for popular repos
- Three-tier storage: Hot (Neptune) → Warm (Neptune) → Cold (S3)

### Garbage Collection
- Archive when ref_count = 0 for 30 days
- Delete when ref_count = 0 for 90 days
- Restore from S3 in 1-2 minutes if needed

---

## Cost Analysis Framework

When documenting cost optimizations, include:

**Baseline Cost:**
- Infrastructure costs without optimization
- Per-user costs at different scales

**Optimized Cost:**
- Infrastructure costs with optimization
- Savings percentage and absolute dollars

**Break-even Analysis:**
- At what scale does optimization pay off?
- Implementation cost vs ongoing savings

**Monitoring:**
- Key metrics to track
- Alerts for cost anomalies

---

**Back to:** [dev_docs/README.md](../README.md)
