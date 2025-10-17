# Implementation Documentation (MVP Focus)

**Last Updated:** October 17, 2025
**Purpose:** Implementation guides, status tracking, and development resources for MVP

> **📘 Strategic Shift:** Focused on local-first MVP implementation. Cloud features, completed sessions, and obsolete planning docs moved to [../99-archive/03-implementation-obsolete](../99-archive/03-implementation-obsolete/).

---

## What Goes Here

**Implementation Guides:**
- Step-by-step integration instructions
- Configuration examples
- Component setup procedures
- Testing strategies

**Status Tracking:**
- Current implementation progress
- MVP blockers and next steps
- Component completion status

**Development Resources:**
- Development workflows
- Deployment procedures
- Performance optimization guides
- Distribution strategies

---

## Current Documents (MVP Focus)

### 🚀 Quick Start

**New to CodeRisk development? Start here:**
1. [DEVELOPMENT_AND_DEPLOYMENT.md](DEVELOPMENT_AND_DEPLOYMENT.md) - Development workflow, local setup, testing
2. [integration_guides/local_deployment.md](integration_guides/local_deployment.md) - Docker Compose setup
3. [status.md](status.md) - Current implementation status

---

### 📋 Status & Roadmap

**Track progress:**
- **[status.md](status.md)** - ⭐ Implementation status, MVP completion tracking, blockers
- **[NEXT_STEPS.md](NEXT_STEPS.md)** - Immediate next actions and priorities
- **[PRODUCTION_READINESS.md](PRODUCTION_READINESS.md)** - Production readiness checklist

---

### 🛠 Development & Deployment

**Development workflow:**
- **[DEVELOPMENT_AND_DEPLOYMENT.md](DEVELOPMENT_AND_DEPLOYMENT.md)** - ⭐ Local development setup, testing, contribution guide
- **[DEPLOYMENT_STRATEGY.md](DEPLOYMENT_STRATEGY.md)** - Git workflow, releases, CI/CD pipeline, Homebrew distribution

**Distribution:**
- **[packaging_and_distribution.md](packaging_and_distribution.md)** - GoReleaser, Homebrew, install script, Docker Hub
- **[website_messaging.md](website_messaging.md)** - Website content, messaging framework, open source positioning
- **[configuration_management.md](configuration_management.md)** - API key setup, config files, user experience tiers

**Release management:**
- **[CHANGELOG.md](CHANGELOG.md)** - Version history and release notes

---

### 📚 Integration Guides

**Graph construction:**
- **[integration_guides/graph_construction.md](integration_guides/graph_construction.md)** - ⭐ 3-layer graph overview
- **[integration_guides/layer_1_treesitter.md](integration_guides/layer_1_treesitter.md)** - Layer 1 (Code Structure) with tree-sitter AST parsing
- **[integration_guides/layers_2_3_github_fetching.md](integration_guides/layers_2_3_github_fetching.md)** - Layers 2-3 GitHub API data fetching
- **[integration_guides/layers_2_3_graph_construction.md](integration_guides/layers_2_3_graph_construction.md)** - Layers 2-3 graph construction (Temporal + Incidents)
- **[integration_guides/cli_integration.md](integration_guides/cli_integration.md)** - End-to-end `crisk init` orchestration

**Local deployment:**
- **[integration_guides/local_deployment.md](integration_guides/local_deployment.md)** - ⭐ Docker Compose setup with Neo4j
- **[NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md](NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md)** - Neo4j performance tuning
- **[NEO4J_MODERNIZATION_GUIDE.md](NEO4J_MODERNIZATION_GUIDE.md)** - Neo4j best practices and modernization

**Developer experience:**
- **[integration_guides/ux_adaptive_verbosity.md](integration_guides/ux_adaptive_verbosity.md)** - 4-level verbosity system (quiet, standard, explain, AI mode)
- **[integration_guides/ux_pre_commit_hook.md](integration_guides/ux_pre_commit_hook.md)** - Pre-commit hook integration
- **[integration_guides/ux_ai_mode_output.md](integration_guides/ux_ai_mode_output.md)** - AI Mode JSON schema for IDE integration

**Platform integrations:**
- **[KEYCHAIN_INTEGRATION_GUIDE.md](KEYCHAIN_INTEGRATION_GUIDE.md)** - macOS Keychain for secure API key storage
- **[GRAPH_SEARCH_INTEGRATION_GUIDE.md](GRAPH_SEARCH_INTEGRATION_GUIDE.md)** - Graph search strategies and query optimization

**Testing helpers:**
- **[integration_guides/quick_test_commands.md](integration_guides/quick_test_commands.md)** - Quick reference for testing
- **[integration_guides/testing_edge_fixes.md](integration_guides/testing_edge_fixes.md)** - Testing CO_CHANGED/CAUSED_BY edge creation

---

### 🧪 Testing Strategy

**Strategy documents:**
- **[testing/INTEGRATION_TEST_STRATEGY.md](testing/INTEGRATION_TEST_STRATEGY.md)** - ⭐ High-level testing approach
- **[testing/MODIFICATION_TYPES_AND_TESTING.md](testing/MODIFICATION_TYPES_AND_TESTING.md)** - Modification taxonomy and test scenarios
- **[testing/TESTING_EXPANSION_SUMMARY.md](testing/TESTING_EXPANSION_SUMMARY.md)** - Testing expansion plan
- **[testing/E2E_TEST_SUMMARY.md](testing/E2E_TEST_SUMMARY.md)** - E2E test execution summary

**Data verification:**
- **[testing/DATA_VERIFICATION_QUERIES.md](testing/DATA_VERIFICATION_QUERIES.md)** - Neo4j/PostgreSQL verification queries

---

### 📖 Phase Documentation

**Completed phases:**
- **[phases/phase_dx_foundation.md](phases/phase_dx_foundation.md)** - ✅ Developer Experience foundation (pre-commit hook, verbosity, AI mode) - **COMPLETE**

**Note:** Future phase plans (Q1 2026, Q2-Q3 2026) archived to [../99-archive/03-implementation-obsolete/](../99-archive/03-implementation-obsolete/) as they're beyond MVP scope.

---

## Document Guidelines

### When to Add Here
- Implementation step-by-step guides
- Testing strategies and procedures
- Development workflow improvements
- Status updates and roadmaps
- Integration tutorials

### When NOT to Add Here
- Architecture decisions (goes to 01-architecture/)
- Operational patterns (goes to 02-operations/)
- Product requirements (goes to 00-product/)
- One-time session prompts (complete work, don't document)
- Research experiments (goes to 04-research/)

### Format
- **Actionable** - Clear steps, commands, examples
- **Current** - Keep status docs updated weekly
- **Concise** - High-level guidance, not code dumps
- **No code** - Minimal examples OK, link to codebase for details

---

## MVP Implementation Status

### ✅ Completed (Ready for MVP)

**Core functionality:**
- Pre-commit hook integration
- Adaptive verbosity (quiet, standard, explain, AI)
- 3-layer graph ingestion (structure, temporal, incidents)
- Tree-sitter AST parsing (Python, JS, TS, Go)
- Git history analysis
- Docker Compose deployment
- Authentication (login/logout/whoami)
- Incident tracking (create, link, search)

**Development infrastructure:**
- GoReleaser configuration
- GitHub Actions CI/CD
- Homebrew distribution
- Installation script
- Docker Hub publishing

### 🚧 In Progress (Current Sprint)

- Risk assessment engine (Phase 1 baseline + Phase 2 LLM)
- Neo4j query optimization
- False positive tracking
- Configuration management improvements

### 📅 Planned (Next 2-4 Weeks)

- Complete risk assessment integration
- Performance optimization (<200ms Phase 1, <5s Phase 2)
- Production hardening
- Beta user onboarding
- Documentation polish

**See [NEXT_STEPS.md](NEXT_STEPS.md) and [status.md](status.md) for detailed roadmap**

---

## Archived Documents (Obsolete)

Documents that are completed, obsolete, or out of MVP scope have been moved to:

**[../99-archive/03-implementation-obsolete/](../99-archive/03-implementation-obsolete/)**

**Archived categories:**
- Session prompts (SESSION_*.md) - Completed work
- Coordination docs (PARALLEL_SESSION_PLAN*.md) - Sessions complete
- Completion reports (DX_FOUNDATION_COMPLETE.md, etc.) - Historical milestones
- Cloud research (CLOUD_GRAPH_DATABASE_COMPARISON.md) - Deferred to post-MVP
- Future phase plans (Q1 2026, Q2-Q3 2026) - Beyond MVP scope
- Test results (TEST_RESULTS_OCT_2025.md) - One-time reports

**Why archived:** These docs were useful for specific work sessions or represent future vision outside 4-6 week MVP timeline.

**See:** [../99-archive/03-implementation-obsolete/README.md](../99-archive/03-implementation-obsolete/README.md) for full archive index

---

## How to Navigate This Documentation

**1. New contributors - Start here:**
   - Read [DEVELOPMENT_AND_DEPLOYMENT.md](DEVELOPMENT_AND_DEPLOYMENT.md) for setup
   - Check [status.md](status.md) to understand current progress
   - Review [NEXT_STEPS.md](NEXT_STEPS.md) for priorities

**2. Implementing features:**
   - Find relevant integration guide in [integration_guides/](integration_guides/)
   - Check [testing/](testing/) for testing strategy
   - Update [status.md](status.md) when complete

**3. Deploying/releasing:**
   - Follow [DEPLOYMENT_STRATEGY.md](DEPLOYMENT_STRATEGY.md) for releases
   - Check [PRODUCTION_READINESS.md](PRODUCTION_READINESS.md) before launch
   - Update [CHANGELOG.md](CHANGELOG.md) with changes

**4. Optimizing:**
   - Review [NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md](NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md) for graph performance
   - Check [integration_guides/graph_construction.md](integration_guides/graph_construction.md) for optimization strategies

**5. Future reference:**
   - Explore archived docs in [../99-archive/03-implementation-obsolete/](../99-archive/03-implementation-obsolete/) for historical context
   - Check [../99-archive/01-architecture-future-vision/](../99-archive/01-architecture-future-vision/) for cloud features

---

## Related Documentation

**Product Vision:**
- [../00-product/mvp_vision.md](../00-product/mvp_vision.md) - MVP scope and goals
- [../00-product/developer_experience.md](../00-product/developer_experience.md) - UX design principles

**Architecture:**
- [../01-architecture/mvp_architecture_overview.md](../01-architecture/mvp_architecture_overview.md) - System architecture
- [../01-architecture/agentic_design.md](../01-architecture/agentic_design.md) - Investigation flow
- [../01-architecture/graph_ontology.md](../01-architecture/graph_ontology.md) - Graph schema

**Operations:**
- [../02-operations/local_deployment_operations.md](../02-operations/local_deployment_operations.md) - Operational patterns
- [../02-operations/development_best_practices.md](../02-operations/development_best_practices.md) - Best practices

---

## Integration Guide Template

When adding new integration guides, use this structure:

```markdown
# [Component] Integration Guide

**Purpose:** [What this component does]
**Prerequisites:** [What you need before starting]
**Last Updated:** YYYY-MM-DD

---

## Overview

[High-level description]

---

## Installation

[Step-by-step installation]

---

## Configuration

[Required configuration, examples]

---

## Usage

[Common patterns, examples]

---

## Testing

[How to test the integration]

---

## Troubleshooting

### Issue: [Common problem]
**Symptom:** [What you see]
**Cause:** [Why it happens]
**Solution:** [How to fix]

---

## Related Documentation

- [Architecture doc](../01-architecture/...)
- [Operations doc](../02-operations/...)
```

---

## Status Update Frequency

**Keep these docs current:**
- **status.md** - Update weekly (or after major milestones)
- **NEXT_STEPS.md** - Update when priorities change
- **PRODUCTION_READINESS.md** - Update before release
- **CHANGELOG.md** - Update with every release

---

**Back to:** [dev_docs/README.md](../README.md)

---

**Last Updated:** October 17, 2025
**Next Review:** After MVP beta launch (Week 7-8)
