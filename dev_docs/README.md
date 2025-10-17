# CodeRisk Development Documentation

**Last Updated:** October 4, 2025

---

## 🚨 For AI Agents / Claude Code

### For Documentation Changes

**BEFORE making any documentation changes:**

1. **READ:** [DOCUMENTATION_WORKFLOW.md](DOCUMENTATION_WORKFLOW.md) - Complete guide for AI-assisted updates
2. **CONSULT:** [12-factor-agents-main/README.md](12-factor-agents-main/README.md) - Check relevant principles
3. **FOLLOW:** Decision tree to determine where content goes
4. **CHECK:** Cross-reference checklist before committing

**Key Rules:**
- ✅ **spec.md is ground truth** - Update it first for requirement changes
- ✅ **No redundancy** - Merge into existing docs, don't duplicate
- ✅ **Update cross-references** - Verify all links after changes
- ✅ **High-level only** - Concepts and decisions, not implementation code
- ✅ **Cite 12-factor principles** - Reference applicable factors when making decisions

**Start here:** [DOCUMENTATION_WORKFLOW.md](DOCUMENTATION_WORKFLOW.md)

---

### For Code Implementation

**BEFORE implementing any code changes:**

1. **READ:** [DEVELOPMENT_WORKFLOW.md](DEVELOPMENT_WORKFLOW.md) - Complete guide for AI-assisted development
2. **CHECK:** Security guardrails and architectural constraints
3. **CONSULT:** [12-factor-agents-main/README.md](12-factor-agents-main/README.md) - Apply relevant principles
4. **FOLLOW:** Decision tree to determine where code goes
5. **VERIFY:** Quality gates before committing

**Key Rules:**
- 🔐 **Security first** - Get approval for auth/crypto/permissions changes
- ✅ **Architecture compliance** - Align with spec.md constraints
- ✅ **Test coverage** - 60-80% coverage by component type
- ✅ **Error handling** - Context propagation, descriptive errors
- ✅ **Performance** - Hop limits, caching, timeouts

**Start here:** [DEVELOPMENT_WORKFLOW.md](DEVELOPMENT_WORKFLOW.md)

---

## Quick Start

**New to CodeRisk?** Start here:
1. Read [01-architecture/system_overview_layman.md](01-architecture/system_overview_layman.md) - Accessible explanation for everyone
2. Read [spec.md](spec.md) - The single source of truth
3. Explore [01-architecture/](01-architecture/) - Understand system design
4. Check [03-implementation/status.md](03-implementation/status.md) - See what's built

**Updating documentation?** Read [DOCUMENTATION_WORKFLOW.md](DOCUMENTATION_WORKFLOW.md) first.

**Implementing code?** Read [DEVELOPMENT_WORKFLOW.md](DEVELOPMENT_WORKFLOW.md) first.

---

## Documentation Structure

### 📋 **[spec.md](spec.md)** - Main Specification
**The single source of truth** for requirements, business logic, and architecture.
- All functional & non-functional requirements
- Technology stack
- Constraints and risks
- Glossary

**When to read:** Before any implementation work, when making decisions

---

### 🎯 **[00-product/](00-product/)** - Product & Business
Product vision, user personas, market analysis, competitive positioning.

**Documents:**
- Product vision and mission
- User personas (Ben, Clara)
- Competitive analysis (vs Greptile, Codescene)
- Go-to-market strategy

**When to use:** Product planning, positioning, feature prioritization

---

### 🏗️ **[01-architecture/](01-architecture/)** - System Design
Core system architecture, design decisions, technical specifications.

**Documents:**
- [cloud_deployment.md](01-architecture/cloud_deployment.md) - Cloud infrastructure, BYOK model, costs
- [graph_ontology.md](01-architecture/graph_ontology.md) - Five-layer graph structure (now with branch-aware Layer 1)
- [agentic_design.md](01-architecture/agentic_design.md) - Agent investigation strategy
- [decisions/](01-architecture/decisions/) - Architecture Decision Records (ADRs)
  - [ADR-001: Neptune over Neo4j](01-architecture/decisions/001-neptune-over-neo4j.md)
  - [ADR-002: Branch-Aware Incremental Ingestion](01-architecture/decisions/002-branch-aware-incremental-ingestion.md)

**When to use:** System design, architecture decisions, technical deep-dives

**Subdirectories:**
- `decisions/` - ADRs documenting key technical choices with rationale

---

### ⚙️ **[02-operations/](02-operations/)** - Operational Strategies
How the system operates at scale, cost optimization, multi-tenancy.

**Documents:**
- [team_and_branching.md](02-operations/team_and_branching.md) - Team sharing, branch deltas
- [public_caching.md](02-operations/public_caching.md) - Public repo cache, garbage collection
- Cost modeling and optimization
- Scaling strategies

**When to use:** Implementing multi-user features, cost optimization, ops planning

---

### 🔨 **[03-implementation/](03-implementation/)** - Implementation Guides
Current status, implementation roadmap, integration guides.

**Documents:**
- [status.md](03-implementation/status.md) - Current implementation status, MVP blockers
- [DX_FOUNDATION_COMPLETE.md](03-implementation/DX_FOUNDATION_COMPLETE.md) - ✅ DX Foundation phase complete!
- `phases/` - Detailed phase roadmaps (MVP, Multi-branch, Public cache)
- `integration_guides/` - How to integrate specific components

**Recent Completion (Oct 4, 2025):**
- ✅ **DX Foundation Phase** - Pre-commit hook, 4 verbosity levels, AI Mode
  - [PARALLEL_SESSION_PLAN.md](03-implementation/PARALLEL_SESSION_PLAN.md) - 3 parallel session strategy
  - [THREE_SESSIONS_SUMMARY.md](03-implementation/THREE_SESSIONS_SUMMARY.md) - Quick reference

**When to use:** Before starting implementation, tracking progress, integration work

**Subdirectories:**
- `phases/` - Detailed roadmaps for each development phase
- `integration_guides/` - Step-by-step integration instructions for components

---

### 🔬 **[04-research/](04-research/)** - Research & Experiments
Explorations, experiments, prototypes, and research findings.

**Subdirectories:**
- `active/` - Current research and experiments
- `archive/` - Completed research (successful or failed)

**When to use:** Exploring new ideas, documenting experiments, research

**Guidelines:**
- Document hypothesis, experiment, and results
- Move to `archive/` when complete (tag as success/failure/inconclusive)
- Reference from architecture docs if experiment informs decision

---

### 🗄️ **[99-archive/](99-archive/)** - Archived Documentation
Deprecated documents, legacy designs, superseded research.

**Contents:** Previous architecture attempts, old designs, outdated specs

**When to use:** Historical reference, understanding evolution

**Note:** Do not reference archive docs in active documentation

---

### 📚 **[12-factor-agents-main/](12-factor-agents-main/)** - AI Agent Development Framework
Principles for building reliable LLM applications. *(Reference material, not CodeRisk-specific)*

**Purpose:** Proven practices for AI agent development that guide our documentation and architecture decisions.

**How to use:**
1. **Entry point:** [12-factor-agents-main/README.md](12-factor-agents-main/README.md) - Overview and table of contents
2. **Individual factors:** [12-factor-agents-main/content/](12-factor-agents-main/content/) - Detailed principles
3. **Consult when:** Making decisions about context engineering, prompts, tools, control flow, etc.
4. **Cite in docs:** Reference relevant factors when documenting architecture decisions

**Key factors for documentation:**
- **Factor 3:** [Own your context window](12-factor-agents-main/content/factor-03-own-your-context-window.md) - Context engineering principles
- **Factor 8:** [Own your control flow](12-factor-agents-main/content/factor-08-own-your-control-flow.md) - Workflow and decision tree design
- **Factor 10:** [Small, Focused Agents](12-factor-agents-main/content/factor-10-small-focused-agents.md) - Agent decomposition

**Note:** Use as **guidance and structure**, not rigid requirements. See [DOCUMENTATION_WORKFLOW.md](DOCUMENTATION_WORKFLOW.md) for factor-to-topic mapping.

---

## Documentation Workflow

### Adding New Documentation

**Product documents** → `00-product/`
- Market research, competitive analysis
- User feedback, persona updates
- Feature proposals with business justification

**Architecture decisions** → `01-architecture/decisions/`
- Use ADR format (see template in decisions/README.md)
- Number sequentially: `001-title.md`, `002-title.md`
- Include context, options, decision, consequences

**Operational guides** → `02-operations/`
- Scaling strategies, cost optimization
- Multi-tenancy patterns
- Operational runbooks

**Implementation guides** → `03-implementation/`
- Integration tutorials
- Setup instructions
- Troubleshooting guides

**Research & experiments** → `04-research/active/`
- Hypothesis and experiment design
- Prototype results
- Move to `archive/` when complete

### Deprecating Documents

1. Move to `99-archive/`
2. Add deprecation note to top of file
3. Update spec.md to remove references
4. Do NOT delete (preserve history)

---

## Document Templates

### Architecture Decision Record (ADR)
See `01-architecture/decisions/README.md` for template.

### Research Document
See `04-research/README.md` for template.

---

## Best Practices

### Writing Guidelines
- **Concise** - Focus on decisions and concepts, not implementation details
- **No code** - High-level descriptions only (code goes in codebase)
- **Cross-reference** - Link related documents
- **Version** - Include "Last Updated" date
- **Single source of truth** - spec.md is authoritative, others support

### Document Naming
- Use lowercase with hyphens: `team-and-branching.md`
- Be descriptive: `neptune-over-neo4j.md` not `decision.md`
- Number ADRs: `001-topic.md`, `002-topic.md`

### Maintaining spec.md
- Update spec.md for any requirement changes
- Keep spec.md high-level, details in subdirectories
- Review spec.md in pull requests for accuracy

---

## Quick Reference

| Need | Document | Location |
|------|----------|----------|
| Requirements | spec.md | Root |
| Infrastructure | cloud_deployment.md | 01-architecture/ |
| Graph model | graph_ontology.md | 01-architecture/ |
| Agent design | agentic_design.md | 01-architecture/ |
| Team sharing | team_and_branching.md | 02-operations/ |
| Public cache | public_caching.md | 02-operations/ |
| Current status | status.md | 03-implementation/ |
| Database choice | 004-neo4j-aura-to-neptune-migration.md | 01-architecture/decisions/ |
| Testing edge fixes | testing_edge_fixes.md | 03-implementation/integration_guides/ |

---

## Documentation Update Workflow

### For AI Agents / Automated Updates

**CRITICAL:** Read [DOCUMENTATION_WORKFLOW.md](DOCUMENTATION_WORKFLOW.md) for complete guidance.

**Quick checklist:**
1. ✅ Determine where content goes (use decision tree)
2. ✅ Read spec.md and relevant docs first
3. ✅ Update spec.md if requirements changed
4. ✅ Update or create supporting docs
5. ✅ Update all cross-references
6. ✅ Update parent README files
7. ✅ Update "Last Updated" dates

**Guardrails:**
- 🚫 **Never duplicate** content across docs
- 🚫 **Never skip** spec.md for requirement changes
- 🚫 **Never delete** docs (move to 99-archive/)
- 🚫 **Never add** implementation code (concepts only)
- ✅ **Always verify** cross-references work
- ✅ **Always update** parent README files

### For Humans

**Adding new content:**
1. Check [DOCUMENTATION_WORKFLOW.md](DOCUMENTATION_WORKFLOW.md) decision tree
2. See if existing doc covers topic (avoid redundancy)
3. Follow template from relevant README
4. Add to parent directory README
5. Link from spec.md if architectural

**Updating existing content:**
1. Update "Last Updated" date
2. Check if spec.md needs updating
3. Verify cross-references still accurate
4. Update parent README if major changes

**Deprecating content:**
1. Add deprecation notice to top
2. Move to 99-archive/
3. Remove from active doc references
4. Update spec.md

---

## Questions?

- **Unclear requirements?** Read spec.md first, then relevant architecture doc
- **Implementation question?** Check 03-implementation/
- **Historical context?** Check 99-archive/ (but don't build from it)
- **New idea?** Document in 04-research/active/
- **How to update docs?** Read [DOCUMENTATION_WORKFLOW.md](DOCUMENTATION_WORKFLOW.md)

---

**Remember:**
- **spec.md is the single source of truth** - All other docs support it
- **DOCUMENTATION_WORKFLOW.md is the update guide** - Follow it for changes
