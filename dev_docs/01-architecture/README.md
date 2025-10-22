# System Architecture Documentation

**Purpose:** Core system design, technical architecture, design decisions for CodeRisk MVP

> **📘 For AI agents:** Before creating/updating architecture docs, read [DOCUMENTATION_WORKFLOW.md](../DOCUMENTATION_WORKFLOW.md) to determine if this is the right location and which document to update.

---

## What Goes Here

**System Design:**
- High-level architecture overview
- Component interactions and data flows
- Technology stack choices
- Local-first deployment strategy
- Regression prevention methodology

**Technical Specifications:**
- Graph database schema (Neo4j)
- LLM-guided investigation algorithms
- Metric calculation and validation (regression detection)
- Integration patterns
- Temporal coupling analysis (CO_CHANGED edges)

**Design Decisions:**
- Architecture Decision Records (ADRs)
- Trade-off analyses
- Technology evaluations
- MVP scope decisions

---

## Current Documents (MVP Focus)

### Core Architecture

**Start Here:**
- **[mvp_architecture_overview.md](mvp_architecture_overview.md)** - **READ THIS FIRST** - High-level overview of MVP architecture, deployment, and design principles

**Detailed Specifications:**
- **[agentic_design.md](agentic_design.md)** - Two-phase investigation flow (fast baseline + LLM investigation), automated due diligence, regression prevention through temporal analysis
- **[graph_ontology.md](graph_ontology.md)** - Three-layer graph structure (Structure, Temporal, Incidents), Neo4j schema, CO_CHANGED edges for temporal coupling
- **[risk_assessment_methodology.md](risk_assessment_methodology.md)** - Regression prevention methodology, five simple metrics, regression detection scenarios, validation framework

### Architecture Decisions

**Relevant to MVP:**
- **[decisions/002-branch-aware-incremental-ingestion.md](decisions/002-branch-aware-incremental-ingestion.md)** - Branch-aware ingestion with language detection
- **[decisions/005-confidence-driven-investigation.md](decisions/005-confidence-driven-investigation.md)** - Confidence-based stopping and adaptive investigation

**Cloud-Related (Deferred to Post-MVP):**
- **[decisions/001-neptune-over-neo4j.md](decisions/001-neptune-over-neo4j.md)** - Cloud database strategy (future)
- **[decisions/003-postgresql-fulltext-search.md](decisions/003-postgresql-fulltext-search.md)** - Full-text search strategy (using Neo4j FTS for MVP)
- **[decisions/004-neo4j-aura-to-neptune-migration.md](decisions/004-neo4j-aura-to-neptune-migration.md)** - Cloud migration path (future)

**Note:** MVP uses local Neo4j in Docker, not cloud databases. Cloud ADRs are preserved for future reference.

---

## Archived Documents (Future Vision)

Documents that discuss cloud infrastructure, enterprise features, or complex optimizations beyond MVP scope have been moved to:

**[../99-archive/01-architecture-future-vision/](../99-archive/01-architecture-future-vision/)**

Archived documents include:
- cloud_deployment.md - AWS Neptune, EKS, Lambda deployment
- arc_intelligence_architecture.md - GitHub mining and ARC database
- scalability_analysis.md - Enterprise-scale validation
- data_volumes.md - Cloud storage calculations
- trust_infrastructure.md - AI code provenance (Q2-Q3 2026)
- incident_knowledge_graph.md - Federated learning (Q1-Q2 2026)
- system_overview_layman.md - References cloud features
- prompt_engineering_design.md - Detailed prompt optimization
- agentic_design_v4.0.md - Complex three-phase version

**Why archived:** These documents describe features outside the 4-6 week MVP timeline. They remain preserved for post-MVP phases.

---

## Document Guidelines

### When to Add Here
- System design proposals for MVP
- Architecture decision records (ADRs)
- Technical deep-dives on core features
- Component specifications

### When NOT to Add Here
- Implementation guides (goes to 03-implementation/)
- Operational procedures (goes to 02-operations/)
- Business requirements (goes to 00-product/)
- Future/cloud features (goes to 99-archive/)

### Format
- **High-level** - Concepts and decisions, not code
- **Decision-focused** - What we chose and why
- **MVP-scoped** - Focus on 4-6 week deliverable
- **No code** - Explain concepts without implementation details

---

## Architecture Decision Records (ADRs)

ADRs are stored in the `decisions/` subdirectory and numbered sequentially.

**When to write an ADR:**
- Significant technology choice (database, framework, language)
- Architectural pattern adoption
- Security or scalability decision
- Trade-off between competing approaches

**ADR Naming:**
- Format: `NNN-short-title.md` (e.g., `001-neptune-over-neo4j.md`)
- Number sequentially starting from 001
- Use lowercase with hyphens

**ADR Status:**
- Some ADRs discuss cloud features (Neptune, PostgreSQL) not in MVP
- These remain valid for future phases
- MVP uses simplified local alternatives (Neo4j, SQLite)

**See:** [decisions/README.md](decisions/README.md) for ADR template

---

## Key Concepts

### Local-First Architecture (MVP)

**Deployment:**
- Docker Compose stack (Neo4j + CodeRisk CLI)
- Local Neo4j graph database (not cloud)
- SQLite for validation data (not PostgreSQL)
- Filesystem cache (not Redis)
- Zero cloud infrastructure

**Philosophy:**
- Free BYOK model (~$1-2/month in LLM costs)
- No network latency (all local)
- Full data control (no vendor lock-in)
- Sufficient for solo/small teams (<10K files)

### Three-Layer Graph Ontology

**Layer 1: Structure (Code & Dependencies)**
- Entities: File, Function, Class, Module
- Relationships: CALLS, IMPORTS, CONTAINS
- Purpose: "What code depends on what?" (~1% FP rate)

**Layer 2: Temporal (Git History & Ownership)**
- Entities: Commit, Developer, PullRequest
- Relationships: AUTHORED, MODIFIES, CO_CHANGED
- Purpose: "How does code evolve?" (~3-5% FP rate)

**Layer 3: Incidents (Failure History)**
- Entities: Incident, Issue
- Relationships: CAUSED_BY, AFFECTS, FIXED_BY
- Purpose: "What has broken before?" (~5% FP rate with manual linking)

**Why three layers:** Structure and temporal are factual, incidents are manually curated. Semantic and risk layers are computed on-demand, not stored.

### Two-Phase Investigation

**Phase 1: Fast Baseline (80% of checks, <200ms)**
- Calculate Tier 1 metrics in parallel (coupling, co-change, test coverage)
- Apply simple heuristic (OR logic)
- If ANY metric HIGH → escalate to Phase 2
- Else return LOW risk immediately (no LLM needed)

**Phase 2: LLM-Guided Investigation (20% of checks, 3-5s)**
- Load 1-hop neighbors into working memory
- LLM reviews evidence, decides next action:
  - Calculate Tier 2 metric (ownership churn, incident similarity)
  - Expand to 2-hop neighbors (rare)
  - Finalize assessment (has enough evidence)
- Max 3 investigation rounds (prevents runaway costs)
- LLM synthesizes final risk level, confidence, evidence, recommendations

**Why two-phase:** 10x cost savings ($0.60/day vs $6/day), 5x latency improvement (800ms avg vs 4s avg)

### Temporal Coupling: The Regression Prevention Core

**Why CO_CHANGED Edges Prevent Regressions:**

Traditional tools analyze code structure (imports, calls) but miss evolutionary coupling.

**Example:**
- `payment.py` doesn't import `fraud.py` (no structural coupling)
- But they changed together in 18/20 commits (90% temporal coupling)
- Developer modifies `payment.py` → CodeRisk warns about `fraud.py`
- Without warning: Merge → Production → Fraud detection breaks

**This is CodeRisk's unique moat** - no other tool tracks temporal coupling.

**How It Works:**
- During `crisk init`: Analyze 90-day git history → Create CO_CHANGED edges with frequency weights
- At pre-commit time: Query CO_CHANGED edges (< 20ms) → Warn about coupled files
- Developer sees: "payment.py and fraud.py changed together 18/20 times (90%). Did you update fraud.py?"

**Regression Prevention in Practice:**
- **Without temporal analysis:** Developer updates A.py, forgets B.py → Production breaks
- **With temporal analysis:** CodeRisk flags B.py as coupled → Developer updates both → Regression prevented

### Five Simple Metrics (Not 9+ Complex Ones)

**Tier 1 (always calculated, factual):**
1. Structural coupling - 1-2% FP rate
2. Temporal co-change - 3-5% FP rate
3. Test coverage ratio - 5-8% FP rate

**Tier 2 (LLM-requested, context-dependent):**
4. Ownership churn - 5-7% FP rate
5. Incident similarity - 8-12% FP rate

**Explicitly avoided (high FP, expensive):**
- ΔDBR (Diffusion Blast Radius) - 15-20% FP rate
- HDCC (Hawkes Decay Co-Change) - 12-18% FP rate
- G² Surprise - 20-25% FP rate
- Vector embeddings - Marginal improvement at 10x cost

**Why simple:** Factual metrics are explainable, cheap, and accurate. Complex statistical models add marginal value at high cost.

### Self-Validating Metrics

**Feedback Loop:**
- User provides feedback: `crisk feedback --false-positive --reason "..."`
- System tracks FP rate per metric in SQLite
- Auto-disable if fp_rate > 3% (with >20 samples)
- Admin reviews and adjusts thresholds
- Tracks which metrics best prevent regressions (based on user feedback)

**Why this matters:** Builds trust through self-correction, learns from user's domain knowledge, prevents metric degradation, continuously improves regression detection accuracy.

---

## MVP Scope Summary

### ✅ In MVP (4-6 weeks)
- Two-phase risk assessment (baseline + LLM)
- Three-layer graph ontology (Neo4j local)
- Five simple metrics (no complex models)
- Docker Compose deployment
- CLI: `crisk init`, `crisk check`, `crisk incident`
- Pre-commit hook integration
- False positive tracking
- Supported languages: Python, JavaScript, TypeScript, Go

### ❌ Out of MVP (Post-MVP)
- Cloud deployment (Neptune, K8s, Lambda)
- Enterprise features (RBAC, SSO, audit logs)
- Complex statistical models (ΔDBR, HDCC, G²)
- Vector embeddings for semantic search
- Web portal for settings/dashboards
- ARC database (industry risk patterns)
- Multi-file PR risk assessment
- Adaptive thresholds (learning from feedback)

---

## How to Navigate This Documentation

**1. Start with the overview:**
   - Read [mvp_architecture_overview.md](mvp_architecture_overview.md) for high-level understanding

**2. Dive into specific areas:**
   - Investigation flow → [agentic_design.md](agentic_design.md)
   - Graph schema → [graph_ontology.md](graph_ontology.md)
   - Metrics and validation → [risk_assessment_methodology.md](risk_assessment_methodology.md)

**3. Understand decisions:**
   - Review relevant ADRs in [decisions/](decisions/)
   - Note which ADRs are MVP vs future

**4. See the big picture:**
   - Connect to product vision in [../00-product/mvp_vision.md](../00-product/mvp_vision.md)
   - Understand user workflow in [../00-product/developer_experience.md](../00-product/developer_experience.md)

**5. Future reference:**
   - Explore archived docs in [../99-archive/01-architecture-future-vision/](../99-archive/01-architecture-future-vision/) for post-MVP ideas

---

## Related Documentation

**Product Documentation:**
- [../00-product/mvp_vision.md](../00-product/mvp_vision.md) - MVP scope and goals
- [../00-product/developer_experience.md](../00-product/developer_experience.md) - CLI workflow and UX
- [../00-product/simplified_pricing.md](../00-product/simplified_pricing.md) - BYOK pricing model

**Implementation (Future):**
- [../03-implementation/](../03-implementation/) - Code structure and implementation guides (TBD)

**Operations (Future):**
- [../02-operations/](../02-operations/) - Deployment and operational procedures (TBD)

---

**Back to:** [dev_docs/README.md](../README.md)
