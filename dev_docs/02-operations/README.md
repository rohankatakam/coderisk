# Operational Strategies Documentation (MVP Focus)

**Last Updated:** October 17, 2025
**Purpose:** Operational patterns for local-first Docker deployment

> **📘 Strategic Shift:** Simplified to local-first MVP operations. Cloud multi-tenancy, Neptune, and team sharing docs moved to [../99-archive/02-operations-future-vision](../99-archive/02-operations-future-vision/) for post-MVP phases.

---

## What Goes Here

**Local Deployment:**
- Docker resource management
- Neo4j optimization patterns
- Local caching strategies
- Development workflows

**MVP Operations:**
- Single-user deployment patterns
- Performance optimization for local graphs
- Troubleshooting common issues
- Backup and recovery procedures

**Development Best Practices:**
- Contributor workflows
- User workflows (pre-commit checks)
- Configuration management
- Multi-repository management

---

## Current Documents (MVP Focus)

### Core Operations

**Start Here:**
- **[local_deployment_operations.md](local_deployment_operations.md)** - Local Docker deployment, Neo4j resource management, caching strategies, monitoring, troubleshooting

**Development:**
- **[development_best_practices.md](development_best_practices.md)** - Development workflows for contributors and users, configuration management, performance optimization

---

## Archived Documents (Future Vision)

Documents discussing cloud infrastructure, multi-tenancy, and team collaboration have been moved to:

**[../99-archive/02-operations-future-vision/](../99-archive/02-operations-future-vision/)**

Archived documents:
- **team_and_branching.md** - Team sharing (90% cost reduction), branch delta strategy, Neptune multi-tenancy, GitHub OAuth (Q2-Q3 2026)
- **public_caching.md** - Public repo cache (99% storage reduction), reference counting, S3 archival, garbage collection (Q3-Q4 2026)

**Why archived:** These documents describe cloud operational patterns outside the 4-6 week MVP timeline. They remain preserved for post-MVP phases when user demand validates cloud features.

**When to revisit:** After MVP validates demand (100+ users, users request team features)

---

## Document Guidelines

### When to Add Here
- Local deployment optimizations
- Docker resource management
- Development workflow improvements
- Troubleshooting patterns
- Performance tuning for local graphs

### When NOT to Add Here
- Cloud infrastructure (goes to 99-archive until validated)
- Architecture decisions (goes to 01-architecture/)
- Implementation guides (goes to 03-implementation/)
- Product requirements (goes to 00-product/)

### Format
- **Practical** - Focus on operational procedures
- **Concise** - High-level patterns, not code
- **Actionable** - Step-by-step instructions
- **No code** - Commands/config examples OK, but minimal

---

## Key Concepts (MVP Focus)

### Local Docker Deployment

**Components:**
- Neo4j container (512MB-2GB RAM)
- CodeRisk CLI (50-100MB RAM)
- Data volumes (`~/.coderisk/data/`)

**Benefits:**
- Zero infrastructure cost
- No network latency
- Full data control
- Sufficient for solo/small teams (<10K files)

### Resource Optimization

**Auto-shutdown pattern:**
- Neo4j shuts down after 30 min idle
- Saves 500MB-2GB RAM
- Auto-starts on next check (5-10s cold start)

**Multi-repository management:**
- Enable auto-shutdown for inactive repos
- Selective initialization (only frequently-developed repos)
- Prune temporal data (commits >30 days old)

### Local Caching Strategies

**Three-layer cache:**
1. **In-memory** (CLI process) - 15 min TTL, 30-40% hit rate
2. **Filesystem** (`~/.coderisk/cache/`) - 1 hour TTL, 10-15% hit rate
3. **Neo4j query cache** - 60 min TTL, 50-60% hit rate

**Surgical invalidation:**
- Only invalidate cache entries for modified files
- Retain 60-70% of cache after commit

### Development Workflows

**For contributors:**
- Branch from main
- Run tests before PR
- Use conventional commits
- Integration test with real repos

**For users:**
- Install pre-commit hook (`crisk hook install`)
- Adjust thresholds for team needs
- Provide feedback on false positives
- Link incidents to code

---

## MVP Scope Summary

### ✅ In MVP (4-6 weeks)
- Local Docker deployment (Neo4j + CLI)
- Single-user workflows
- Filesystem caching
- Resource optimization (auto-shutdown)
- Development best practices
- Troubleshooting guides

### ❌ Out of MVP (Post-MVP)
- Cloud deployment (Neptune, K8s, Lambda)
- Multi-tenancy patterns
- Team collaboration (branch deltas, shared cache)
- Public repository caching
- GitHub OAuth integration
- Row-level security
- Reference counting and garbage collection

---

## How to Navigate This Documentation

**1. Start with local operations:**
   - Read [local_deployment_operations.md](local_deployment_operations.md) for deployment patterns

**2. Learn development workflows:**
   - Review [development_best_practices.md](development_best_practices.md) for contributor and user workflows

**3. Optimize for your use case:**
   - Adjust Neo4j memory (small/medium/large repos)
   - Enable auto-shutdown for resource savings
   - Configure caching strategies
   - Tune thresholds for acceptable false positive rate

**4. Future reference:**
   - Explore archived docs in [../99-archive/02-operations-future-vision/](../99-archive/02-operations-future-vision/) for post-MVP cloud patterns

---

## Related Documentation

**Product Vision:**
- [../00-product/mvp_vision.md](../00-product/mvp_vision.md) - MVP scope and goals
- [../00-product/developer_experience.md](../00-product/developer_experience.md) - UX design

**Architecture:**
- [../01-architecture/mvp_architecture_overview.md](../01-architecture/mvp_architecture_overview.md) - System architecture
- [../01-architecture/agentic_design.md](../01-architecture/agentic_design.md) - Investigation flow

**Implementation:**
- [../03-implementation/DEVELOPMENT_AND_DEPLOYMENT.md](../03-implementation/DEVELOPMENT_AND_DEPLOYMENT.md) - Development workflow
- [../03-implementation/integration_guides/local_deployment.md](../03-implementation/integration_guides/local_deployment.md) - Docker setup guide

---

## Migration Path to Cloud (If Validated)

**Phase 1 (Weeks 1-6): MVP - Local Only**
- Documents: Current 02-operations docs
- Features: Local Neo4j, Docker, single-user
- Infrastructure: Zero cloud cost

**Phase 2 (Months 3-6): Team Features**
- Restore: team_and_branching.md (if users request team features)
- Features: Branch deltas, team access control
- Infrastructure: PostgreSQL metadata, still local Neo4j per user

**Phase 3 (Months 6-12): Scale**
- Restore: public_caching.md, cloud_deployment.md
- Features: Shared graph cache, cloud backend
- Infrastructure: AWS Neptune, S3, Lambda

---

**Back to:** [dev_docs/README.md](../README.md)

---

**Last Updated:** October 17, 2025
**Next Review:** After MVP launch (Week 7-8)
