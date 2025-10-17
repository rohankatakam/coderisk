# System Architecture Documentation

**Purpose:** Core system design, technical architecture, design decisions

> **📘 For AI agents:** Before creating/updating architecture docs, read [DOCUMENTATION_WORKFLOW.md](../DOCUMENTATION_WORKFLOW.md) to determine if this is the right location and which document to update.

---

## What Goes Here

**System Design:**
- High-level architecture diagrams
- Component interactions
- Data flows
- Technology stack choices

**Technical Specifications:**
- Graph database schema and ontology
- Agent investigation algorithms
- API contracts
- Integration patterns

**Infrastructure:**
- Cloud deployment architecture
- Scalability strategies
- Security design
- Monitoring and observability

**Design Decisions:**
- Architecture Decision Records (ADRs)
- Trade-off analyses
- Technology evaluations

---

## Current Documents

### Core Architecture
- **[system_overview_layman.md](system_overview_layman.md)** - Accessible explanation of CodeRisk architecture and agentic search for non-technical stakeholders
- **[cloud_deployment.md](cloud_deployment.md)** - Cloud infrastructure, BYOK model, pricing, multi-tenancy
- **[graph_ontology.md](graph_ontology.md)** - Five-layer graph structure, relationships, inference rules
- **[agentic_design.md](agentic_design.md)** - Agent investigation strategy with Phase 0 pre-analysis, confidence-driven investigation, adaptive thresholds
- **[arc_intelligence_architecture.md](arc_intelligence_architecture.md)** - ARC database integration, hybrid GraphRAG, pattern recombination strategy
- **[incident_knowledge_graph.md](incident_knowledge_graph.md)** - Incident attribution pipeline, pattern signature hashing, causal chains
- **[risk_assessment_methodology.md](risk_assessment_methodology.md)** - Risk calculation formulas, thresholds, metric validation
- **[prompt_engineering_design.md](prompt_engineering_design.md)** - LLM prompt architecture, context management, token budgets

### Architecture Decisions
- **[decisions/001-neptune-over-neo4j.md](decisions/001-neptune-over-neo4j.md)** - Why Neptune Serverless (70% cost savings) - See ADR-004 for phased approach
- **[decisions/002-branch-aware-incremental-ingestion.md](decisions/002-branch-aware-incremental-ingestion.md)** - Branch-aware ingestion with language detection (92% storage reduction)
- **[decisions/003-postgresql-fulltext-search.md](decisions/003-postgresql-fulltext-search.md)** - PostgreSQL full-text search for incident similarity (eliminates LanceDB dependency)
- **[decisions/004-neo4j-aura-to-neptune-migration.md](decisions/004-neo4j-aura-to-neptune-migration.md)** - Phased cloud database strategy: Neo4j Aura (MVP) → Neptune (Scale)
- **[decisions/005-confidence-driven-investigation.md](decisions/005-confidence-driven-investigation.md)** - Phase 0 pre-analysis, adaptive thresholds, confidence-based stopping (70-80% FP reduction)

---

## Document Guidelines

### When to Add Here
- System design proposals
- Architecture decision records (ADRs)
- Technical deep-dives
- Component specifications

### When NOT to Add Here
- Implementation guides (goes to 03-implementation/)
- Operational procedures (goes to 02-operations/)
- Business requirements (goes to 00-product/)

### Format
- **High-level** - Concepts and decisions, not code
- **Decision-focused** - What we chose and why
- **Timeless** - Avoid implementation details that change frequently

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

**See:** [decisions/README.md](decisions/README.md) for ADR template

---

## Key Concepts

### Cloud-First Architecture
- BYOK (Bring Your Own API Key) model
- Amazon Neptune Serverless for graphs
- Kubernetes for compute orchestration
- PostgreSQL for metadata
- Redis for caching

### Graph Ontology
- Five layers: Syntactic, Semantic, Behavioral, Risk, Causal
- Dense graphs (50-200 relationships per entity)
- Temporal coupling from git history
- Incident correlation

### Agentic Investigation
- **Three-phase architecture:** Phase 0 pre-analysis → Phase 1 baseline → Phase 2 confidence-driven investigation
- **Adaptive thresholds:** Domain-aware configs (Python web vs Go backend)
- **Confidence-based stopping:** Dynamic hop count (1-5) based on evidence quality
- **Hybrid ARC patterns:** Combine complementary patterns for 3-5x better insights
- **Metric validation:** Auto-exclude metrics with >3% FP rate
- **Evidence-based reasoning:** LLM synthesizes from multiple low-FP signals

---

**Back to:** [dev_docs/README.md](../README.md)
