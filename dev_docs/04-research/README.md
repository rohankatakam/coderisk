# Research & Experiments Documentation

**Purpose:** Document explorations, experiments, prototypes, and research findings

> **📘 For AI agents:** Before creating/updating research docs, read [DOCUMENTATION_WORKFLOW.md](../DOCUMENTATION_WORKFLOW.md) to determine if this is the right location and which document to update.

---

## What Goes Here

**Research Documents:**
- Hypothesis and research questions
- Experiment designs
- Prototype results
- Performance benchmarks
- Algorithm explorations

**Experiments:**
- Proof-of-concepts
- A/B test designs
- Performance comparisons
- Alternative approaches

**Findings:**
- Research conclusions
- Lessons learned
- Recommendations (implement vs abandon)

---

## Subdirectories

### `active/`
Current research and experiments in progress.

**Current Research:**
- ✅ [branch_aware_ingestion_strategy.md](active/branch_aware_ingestion_strategy.md) - **COMPLETED** (moved to ADR-002)
  - Result: Success ✅ - Accepted as ADR-002
  - Outcome: 92% storage reduction, incremental delta strategy

**What goes here:**
- Ongoing experiments
- Research not yet concluded
- Prototypes under evaluation

**Move to archive when:**
- Experiment completed (success or failure)
- Decision made (implement or abandon)
- Research question answered

### `archive/`
Completed research, both successful and failed.

**What goes here:**
- Concluded experiments
- Validated or invalidated hypotheses
- Abandoned prototypes with learnings

**Tag documents:**
- ✅ SUCCESS - Led to implementation
- ❌ FAILURE - Did not work, documented why
- ⚠️ INCONCLUSIVE - Needs more research

---

## Document Guidelines

### When to Add Here
- Exploring new algorithms or approaches
- Performance experiments and benchmarks
- Technology evaluations
- Proof-of-concept implementations
- "What if?" investigations

### When NOT to Add Here
- Decided architecture (goes to 01-architecture/)
- Implementation guides (goes to 03-implementation/)
- Product ideas (goes to 00-product/)

### Format
- **Hypothesis-driven** - Start with clear question
- **Experiment design** - How you'll test it
- **Results** - What you found (include data)
- **Conclusion** - What it means (implement/abandon/needs more work)

---

## Research Document Template

```markdown
# Research: [Title]

**Author:** [Name]
**Date:** YYYY-MM-DD
**Status:** [Active | Complete]
**Result:** [Success ✅ | Failure ❌ | Inconclusive ⚠️]

---

## Hypothesis

[What are you trying to prove or disprove?]

**Research Question:**
[Clear, specific question]

---

## Context

**Background:**
- [Why are we exploring this?]
- [What problem does it solve?]

**Constraints:**
- [Time, resources, scope]

---

## Experiment Design

**Approach:**
[How you'll test the hypothesis]

**Methodology:**
1. [Step 1]
2. [Step 2]

**Success Criteria:**
- [What defines success?]
- [What metrics to measure?]

---

## Results

**Data:**
[Quantitative results, benchmarks, measurements]

**Observations:**
[Qualitative findings, unexpected behaviors]

**Artifacts:**
- [Link to prototype code]
- [Link to benchmarks]

---

## Analysis

**Findings:**
[What the data tells us]

**Limitations:**
[What we couldn't test or prove]

**Threats to Validity:**
[Assumptions, biases, confounding factors]

---

## Conclusion

**Recommendation:**
[Implement | Abandon | Needs more research]

**Rationale:**
[Why this recommendation?]

**Next Steps:**
- [If implement: what changes to architecture?]
- [If abandon: what to try instead?]
- [If inconclusive: what additional experiments?]

---

## References

- [Related ADRs]
- [External research, papers]
- [Related experiments]
```

---

## Research Lifecycle

### 1. Start Research
- Create document in `active/`
- Define hypothesis and experiment design
- Get feedback from team

### 2. Run Experiment
- Execute experiment
- Document results as you go
- Update status

### 3. Conclude Research
- Analyze results
- Make recommendation
- Update status to "Complete"

### 4. Archive
- Move to `archive/`
- Tag with result (Success ✅ / Failure ❌ / Inconclusive ⚠️)
- If success → create ADR in 01-architecture/decisions/

### 5. Implement (if successful)
- Update architecture docs
- Add to implementation roadmap
- Reference research in ADR

---

## Example Research Topics

**Algorithm Research:**
- "Can we reduce hop count to 2 without losing accuracy?"
- "Does caching spatial context improve latency by >30%?"
- "Alternative graph traversal strategies"

**Technology Evaluation:**
- "TigerGraph vs Neptune performance comparison"
- "OpenSearch vs PostgreSQL for full-text search"
- "Rust vs Go for graph traversal performance"

**Performance Optimization:**
- "Impact of materialized views on cache hit rate"
- "Optimal Redis cache TTL values"
- "Batch vs streaming graph updates"

---

## Best Practices

1. **Document failures** - Failed experiments are valuable learning
2. **Include data** - Benchmarks, metrics, graphs
3. **Be specific** - Precise hypothesis, clear success criteria
4. **Time-box** - Set deadline for experiments
5. **Review results** - Get team feedback before concluding
6. **Archive promptly** - Move to archive when done

---

**Back to:** [dev_docs/README.md](../README.md)
