# Implementation Documentation

**Purpose:** Implementation status, roadmaps, integration guides, how-to tutorials

> **📘 For AI agents:** Before creating/updating implementation docs, read [DOCUMENTATION_WORKFLOW.md](../DOCUMENTATION_WORKFLOW.md) to determine if this is the right location and which document to update.

---

## What Goes Here

**Status Tracking:**
- Current implementation progress
- MVP blockers and dependencies
- Technology stack status
- Component completion tracking

**Phase Roadmaps:**
- Detailed phase plans (MVP, Multi-branch, Public cache)
- Timeline estimates
- Success criteria
- Deliverables

**Integration Guides:**
- Step-by-step integration instructions
- Configuration examples
- Troubleshooting guides
- Best practices

**How-To Tutorials:**
- Setting up development environment
- Running tests
- Deploying components
- Debugging common issues

---

## Current Documents

**🎯 Development & Deployment:**
- **[DEVELOPMENT_AND_DEPLOYMENT.md](DEVELOPMENT_AND_DEPLOYMENT.md)** - ⭐ High-level development workflow and deployment strategy
- **[DEPLOYMENT_STRATEGY.md](DEPLOYMENT_STRATEGY.md)** - Detailed release process, CI/CD, and branching strategy

**🎯 Current Status:**
- **[CURRENT_STATUS_AND_CLOUD_READINESS.md](CURRENT_STATUS_AND_CLOUD_READINESS.md)** - What's working locally vs. what's needed for cloud multi-tenant deployment
- **[status.md](status.md)** - Implementation status, MVP blockers, phase roadmap
- **[NEXT_STEPS.md](NEXT_STEPS.md)** - Immediate next actions (Week 1-8 roadmap)
- **[PHASE_0_PARALLEL_IMPLEMENTATION_PLAN.md](PHASE_0_PARALLEL_IMPLEMENTATION_PLAN.md)** - Phase 0 + Adaptive System (4 parallel agents, 2-3 weeks)
- **[IMPLEMENTATION_LOG.md](IMPLEMENTATION_LOG.md)** - Historical implementation log with decisions
- **[DX_FOUNDATION_COMPLETE.md](DX_FOUNDATION_COMPLETE.md)** - DX Foundation phase summary

**Coordination Documents (Parallel Sessions):**

**DX Foundation (Complete ✅):**
- **[PARALLEL_SESSION_PLAN.md](PARALLEL_SESSION_PLAN.md)** - File ownership map and checkpoints
- **[THREE_SESSIONS_SUMMARY.md](THREE_SESSIONS_SUMMARY.md)** - Quick reference for managing sessions
- **[SESSION_1_PROMPT.md](SESSION_1_PROMPT.md)** - Session 1: Pre-commit hook (complete)
- **[SESSION_2_PROMPT.md](SESSION_2_PROMPT.md)** - Session 2: Adaptive verbosity (complete)
- **[SESSION_3_PROMPT.md](SESSION_3_PROMPT.md)** - Session 3: AI Mode (complete)

**Week 1 Core Functionality (Ready to Execute):**
- **[../../WEEK1_QUICK_START.md](../../WEEK1_QUICK_START.md)** - 🚀 **START HERE** - Quick start guide for Week 1 parallel sessions
- **[PARALLEL_SESSION_PLAN_WEEK1.md](PARALLEL_SESSION_PLAN_WEEK1.md)** - File ownership map and coordination protocol
- **[SESSION_4_PROMPT.md](SESSION_4_PROMPT.md)** - Session 4: Git Integration Functions
- **[SESSION_5_PROMPT.md](SESSION_5_PROMPT.md)** - Session 5: Init Flow Orchestration
- **[SESSION_6_PROMPT.md](SESSION_6_PROMPT.md)** - Session 6: Risk Calculation & Validation

---

## Subdirectories

### `phases/`
Detailed roadmaps for each development phase:
- **[phase_dx_foundation.md](phases/phase_dx_foundation.md)** - ✅ Developer Experience foundation (pre-commit hook, verbosity, AI mode) **COMPLETE**
- **[phase_cornered_resource_q1_2026.md](phases/phase_cornered_resource_q1_2026.md)** - Q1 2026: Cornered resource strategy
- **[phase_trust_layer_q2_q3_2026.md](phases/phase_trust_layer_q2_q3_2026.md)** - Q2-Q3 2026: Trust and collaboration

**Next Phase:** Complete core functionality (git integration, crisk init) - See [NEXT_STEPS.md](NEXT_STEPS.md)

**Template:**
- Scope and goals
- Deliverables and success criteria
- Timeline and milestones
- Dependencies and blockers
- Testing strategy

### `integration_guides/`
Step-by-step guides for integrating components:

**Graph Construction:**
- **[local_deployment.md](integration_guides/local_deployment.md)** - Docker Compose setup with Neo4j, Redis, Postgres
- **[graph_construction.md](integration_guides/graph_construction.md)** - 3-layer graph construction overview (Tree-sitter, Git, GitHub API)
- **[layer_1_treesitter.md](integration_guides/layer_1_treesitter.md)** - Layer 1 (Code Structure) with tree-sitter AST parsing
- **[layers_2_3_github_fetching.md](integration_guides/layers_2_3_github_fetching.md)** - Priority 6A: GitHub API → PostgreSQL (Stage 1)
- **[layers_2_3_graph_construction.md](integration_guides/layers_2_3_graph_construction.md)** - Priority 6B: PostgreSQL → Neo4j/Neptune (Stage 2)
- **[cli_integration.md](integration_guides/cli_integration.md)** - Priority 6C: End-to-end `crisk init` orchestration

**Developer Experience (UX):**
- **[ux_adaptive_verbosity.md](integration_guides/ux_adaptive_verbosity.md)** - 4-level verbosity system (quiet, standard, explain, AI mode)
- **[ux_pre_commit_hook.md](integration_guides/ux_pre_commit_hook.md)** - Pre-commit hook integration (`crisk hook install`)
- **[ux_ai_mode_output.md](integration_guides/ux_ai_mode_output.md)** - AI Mode JSON schema for Claude Code/Cursor integration

**Testing & Validation:**
- **[END_TO_END_TESTING_RESULTS.md](END_TO_END_TESTING_RESULTS.md)** - ✅ Complete E2E testing results, critical bugs found/fixed
- **[testing_edge_fixes.md](integration_guides/testing_edge_fixes.md)** - Complete testing instructions for CO_CHANGED/CAUSED_BY edge fixes
- **[quick_test_commands.md](integration_guides/quick_test_commands.md)** - Quick reference commands and expected results

**Testing Strategy Documentation:** (in `testing/`)
- **[TESTING_EXPANSION_SUMMARY.md](testing/TESTING_EXPANSION_SUMMARY.md)** - ✨ **NEW** - Executive summary: 8 new test scenarios and Phase 0 recommendation
- **[MODIFICATION_TYPES_AND_TESTING.md](testing/MODIFICATION_TYPES_AND_TESTING.md)** - ✨ **NEW** - Full taxonomy: 10 modification types, impact framework, detailed test scenarios
- **[INTEGRATION_TEST_STRATEGY.md](testing/INTEGRATION_TEST_STRATEGY.md)** - High-level testing strategy for Claude Code + CodeRisk validation
- **[E2E_TEST_SUMMARY.md](testing/E2E_TEST_SUMMARY.md)** - Final E2E test execution results (all tests passing)
- **[DATA_VERIFICATION_QUERIES.md](testing/DATA_VERIFICATION_QUERIES.md)** - Neo4j/PostgreSQL queries for verifying graph construction

**Distribution & Launch (Pre-Launch Phase):**
- **[packaging_and_distribution.md](packaging_and_distribution.md)** - ✨ **NEW** - GoReleaser, Homebrew, install script, Docker distribution strategy
- **[website_messaging.md](website_messaging.md)** - ✨ **NEW** - Website content, messaging framework, open source positioning
- **[configuration_management.md](configuration_management.md)** - ✨ **NEW** - API key setup, config file, tiered user experience (Phase 0/1/2)

**Claude Code Prompts (Ready to Execute):**
- **[BACKEND_PACKAGING_PROMPT.md](BACKEND_PACKAGING_PROMPT.md)** - 🚀 **Session 1** - GoReleaser, Homebrew, GitHub Actions setup
- **[FRONTEND_WEBSITE_PROMPT.md](FRONTEND_WEBSITE_PROMPT.md)** - 🚀 **Session 2** - Website updates for open source messaging
- **[CONFIG_MANAGEMENT_PROMPT.md](CONFIG_MANAGEMENT_PROMPT.md)** - 🚀 **Session 3** - `crisk configure`, tiered UX, config file support

**Future Guides (Planned):**
- Neptune integration (openCypher queries, connection pooling)
- GitHub OAuth (authentication flow, token management)
- Settings portal (API key config, team management)
- Redis caching (investigation cache, spatial context)
- Webhook handlers (graph updates, branch lifecycle)

**Template:**
- Prerequisites
- Installation and setup
- Configuration
- Usage examples
- Troubleshooting
- Testing

---

## Document Guidelines

### When to Add Here
- Implementation status updates
- Integration tutorials
- Troubleshooting guides
- Development workflows
- Testing strategies

### When NOT to Add Here
- Architecture decisions (goes to 01-architecture/)
- Product requirements (goes to 00-product/)
- Research experiments (goes to 04-research/)

### Format
- **Actionable** - Clear steps, examples, commands
- **Current** - Keep status docs updated weekly
- **Practical** - Focus on "how to" not "what is"

---

## Status Tracking

### What to Track
- ✅ Completed components
- ⚠️ Partial implementations (what's missing)
- ❌ Not started components
- 🚧 Blockers and dependencies

### Update Frequency
- **status.md** - Weekly updates
- **Phase docs** - At phase boundaries
- **Integration guides** - As components evolve

---

## Phase Documentation Template

```markdown
# Phase N: [Phase Name]

**Timeline:** [Start] - [End]
**Status:** [Proposed | In Progress | Completed]

---

## Goals

[High-level objectives for this phase]

---

## Scope

**In Scope:**
- [ ] Feature 1
- [ ] Feature 2

**Out of Scope:**
- [Deferred to next phase]

---

## Deliverables

| Deliverable | Owner | Status | Notes |
|-------------|-------|--------|-------|
| Component A | [Name] | Done | |
| Component B | [Name] | In Progress | |

---

## Success Criteria

- [ ] Criterion 1
- [ ] Criterion 2

---

## Dependencies

- [External dependency]
- [Blocker from previous phase]

---

## Testing Strategy

[How this phase will be tested]

---

## Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Risk 1 | High | Medium | Strategy |

---

## Timeline

Week 1-2: [Milestone]
Week 3-4: [Milestone]
```

---

## Integration Guide Template

```markdown
# [Component] Integration Guide

**Purpose:** [What this component does]
**Prerequisites:** [What you need before starting]

---

## Installation

[Step-by-step installation instructions]

---

## Configuration

[Required configuration, examples]

---

## Usage

[Code examples, common patterns]

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

## References

- [Link to architecture doc]
- [Link to external docs]
```

---

**Back to:** [dev_docs/README.md](../README.md)
