# Advanced Multi-Hop Investigation System

**Status:** Archived for post-MVP evaluation
**Date Archived:** October 21, 2025
**Reason:** Over-engineered for MVP scope

## What's Here

This directory contains the sophisticated multi-hop agent navigation system that was replaced by the simpler single-call investigator for MVP.

### Files

- **hop_navigator.go** - 5-hop navigation with confidence loops
- **breakthroughs.go** - Breakthrough detection tracking risk level changes
- **confidence_loop_test.go** - Tests for confidence-driven navigation

### Why It Was Replaced

The MVP requirement (FR-3) calls for:
- Single LLM call with due diligence context
- <5s latency target (p95)
- Graceful degradation if LLM fails

The multi-hop system:
- Made 5+ LLM calls per investigation (expensive, slow)
- Had ~1,500 LOC of complexity
- Was sophisticated but over-engineered for initial launch

### When to Restore

**Decision Point:** Week 4 beta validation

Restore if:
- False positive rate >15% with simple investigator
- Complex cases need multi-step reasoning
- Users request more detailed investigation trails

Keep simple if:
- False positive rate <15%
- Performance targets met
- Prompt engineering can handle edge cases

### How to Restore

1. Move files back to `internal/agent/`:
   ```bash
   git mv internal/agent/_advanced/*.go internal/agent/
   ```

2. Update `cmd/crisk/check.go`:
   ```go
   investigator := agent.NewInvestigator(llmClient, temporalClient, incidentsClient, nil)
   ```

3. Consider hybrid approach: simple for MEDIUM, complex for HIGH/CRITICAL

## References

- [REFACTORING_PLAN.md](../../../dev_docs/03-implementation/REFACTORING_PLAN.md) - Full refactoring rationale
- [mvp_development_plan.md](../../../dev_docs/00-product/mvp_development_plan.md#L173-L206) - MVP Phase 2 requirements
- [agentic_design.md](../../../dev_docs/01-architecture/agentic_design.md) - Original multi-hop design

## Lessons Learned

1. **Start simple** - Single LLM call may be sufficient
2. **Measure first** - False positive rate determines if complexity needed
3. **Preserve work** - Don't delete, archive for future evaluation
4. **MVP focus** - Ship fast, iterate based on real feedback
