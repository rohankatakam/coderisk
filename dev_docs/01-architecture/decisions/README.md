# Architecture Decision Records (ADRs)

**Purpose:** Document significant architecture decisions with context and rationale

> **📘 For AI agents:** Before creating an ADR, read [DOCUMENTATION_WORKFLOW.md](../../DOCUMENTATION_WORKFLOW.md) to ensure this is an architecture decision (not a requirement change or implementation detail).

---

## What is an ADR?

An Architecture Decision Record captures an important architectural decision along with its context and consequences. ADRs help:
- Preserve reasoning behind decisions
- Onboard new team members
- Avoid revisiting settled decisions
- Learn from past choices

---

## When to Write an ADR

Write an ADR for:
- ✅ Technology selection (database, framework, language)
- ✅ Architectural patterns (microservices, monolith, event-driven)
- ✅ Security or compliance decisions
- ✅ Scalability strategies
- ✅ Trade-offs with significant impact

Don't write an ADR for:
- ❌ Minor implementation details
- ❌ Temporary workarounds
- ❌ Obvious choices with no alternatives
- ❌ Decisions that can be easily reversed

---

## ADR Format

Use this template for all ADRs:

```markdown
# ADR-NNN: [Short Title]

**Date:** YYYY-MM-DD
**Status:** [Proposed | Accepted | Deprecated | Superseded by ADR-XXX]
**Deciders:** [Names of people involved in decision]
**Tags:** [technology, scalability, security, etc.]

---

## Context

[Describe the issue or situation that requires a decision]

**Background:**
- What is the problem we're trying to solve?
- What constraints do we have?
- What's driving this decision now?

---

## Decision

[State the decision clearly]

**We will:** [Clear statement of what we decided]

---

## Options Considered

### Option 1: [Name]
**Pros:**
- [Advantage 1]
- [Advantage 2]

**Cons:**
- [Disadvantage 1]
- [Disadvantage 2]

**Cost:** [If applicable]

### Option 2: [Name]
[Repeat structure]

### Option 3: [Name]
[Repeat structure]

---

## Rationale

[Explain why this decision was made]

**Key factors:**
1. [Factor 1 and why it mattered]
2. [Factor 2 and why it mattered]

**Data supporting decision:**
- [Benchmark results, cost analysis, etc.]

---

## Consequences

**Positive:**
- [Benefit 1]
- [Benefit 2]

**Negative:**
- [Trade-off 1]
- [Trade-off 2]

**Neutral:**
- [Change 1]

**Risks:**
- [Risk 1 and mitigation]
- [Risk 2 and mitigation]

---

## Implementation Notes

[High-level guidance on implementing this decision]

**Timeline:** [If applicable]
**Dependencies:** [Other components affected]
**Migration path:** [If changing from previous approach]

---

## References

- [Link to spec.md section]
- [Link to related ADRs]
- [External resources, benchmarks, papers]
```

---

## Current ADRs

| # | Title | Status | Date |
|---|-------|--------|------|
| 001 | [Neptune over Neo4j](001-neptune-over-neo4j.md) | Accepted | 2025-10-01 |
| 002 | [Branch-Aware Incremental Ingestion](002-branch-aware-incremental-ingestion.md) | Accepted | 2025-10-03 |
| 003 | [PostgreSQL Full-Text Search for Incident Similarity](003-postgresql-fulltext-search.md) | Accepted | 2025-10-05 |
| 004 | [Neo4j Aura to Neptune Migration](004-neo4j-aura-to-neptune-migration.md) | Superseded by ADR-006 | 2025-10-05 |
| 005 | [Confidence-Driven Investigation with Adaptive Thresholds](005-confidence-driven-investigation.md) | Proposed | 2025-10-10 |
| 006 | [Multi-Tenant Neptune with Public Repository Caching](006-multi-tenant-neptune-architecture.md) | Accepted | 2025-10-12 |

---

## ADR Lifecycle

### Proposed
- ADR written, under review
- Seeking feedback from team

### Accepted
- Decision made and implemented
- This is the current approach

### Deprecated
- No longer recommended
- Kept for historical reference

### Superseded
- Replaced by a newer ADR
- Reference the superseding ADR number

---

## Best Practices

1. **Write ADRs early** - Document decisions when made, not after implementation
2. **Be concise** - 1-2 pages max, focus on decision and rationale
3. **Include data** - Benchmarks, cost analysis, user research
4. **Update status** - Mark as deprecated if decision changes
5. **Link to spec.md** - Connect ADRs to requirements
6. **Number sequentially** - Never reuse numbers, even for deprecated ADRs

---

## Review Process

1. Author creates ADR in "Proposed" status
2. Team reviews (async or in meeting)
3. Decision made → status updated to "Accepted"
4. ADR committed to repo

**If decision changes later:**
- Create new ADR superseding the old one
- Update old ADR status to "Superseded by ADR-XXX"
- Update spec.md and architecture docs

---

**Back to:** [01-architecture/README.md](../README.md) | [dev_docs/README.md](../../README.md)
