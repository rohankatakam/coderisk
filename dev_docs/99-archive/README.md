# Archived Documentation

**Purpose:** Historical reference for deprecated designs, superseded research, legacy documentation

---

## What's in Here

This directory contains:
- **Legacy architecture designs** - Previous approaches that were replaced
- **Deprecated specifications** - Outdated requirements and specs
- **Old research** - Early explorations and experiments
- **Superseded decisions** - Architecture choices we moved away from

---

## Why We Keep Archives

**Historical Context:**
- Understand why decisions were made
- Learn from past attempts
- Avoid repeating mistakes

**Institutional Memory:**
- Preserve reasoning behind pivots
- Document evolution of thinking
- Onboard new team members to history

---

## Important Notes

### ⚠️ Do NOT Build From Archive

**Archive documents are NOT current:**
- May contain outdated information
- Decisions may have been reversed
- Technology choices may have changed
- Requirements may have evolved

**Always refer to:**
- [spec.md](../spec.md) for current requirements
- [01-architecture/](../01-architecture/) for current design
- [03-implementation/status.md](../03-implementation/status.md) for current status

### Do NOT Reference in Active Docs

Active documentation should NOT link to archive:
- ❌ Don't reference archive docs in spec.md
- ❌ Don't reference archive in architecture docs
- ❌ Don't use archive for implementation guidance

**Exception:** ADRs may reference archived alternatives for context

---

## Contents

### Legacy Designs (Pre-Cloud Pivot)
Documents from when CodeRisk was designed as local-first tool:
- `ideal_architecture.md`
- `STREAMLINED_MVP_ARCHITECTURE.md`
- `architecture_pathways.md`
- `architecture_tradeoff_analysis.md`

### Deprecated Technologies
Explorations of technologies we chose not to use:
- `cognee_design.md` - Cognee embedding strategy (removed for performance)
- `cognee_embedding_and_llm_strategy.md`
- `ENHANCED_TREESITTER_HCGS_STRATEGY.md`

### Early Product Thinking
Initial product exploration and positioning:
- `product_design.md`
- `functional_business_requirements.md`
- `dev_experience.md`

### Competitive Research
Early competitive analysis:
- `greptile_analysis.md`
- `greptile_vs_coderisk_analysis.md`

### Legacy Technical Specs
Old technical documentation:
- `risk_math.md` - Early risk calculation approaches
- `risk_math_optimized.md`
- `language_performance_analysis.md`
- `go_migration_timing_analysis.md`
- `GITHUB_DATA_INGESTION_STRATEGY.md`
- `TESTING_GUIDE.md`

### Reference Material
Pricing and cost references (may still be useful):
- `llm_pricing_reference.md`

### Launch Strategy Evolution (2025-10-14)
Earlier iterations of launch and production readiness planning:
- `CORRECTED_LAUNCH_STRATEGY.md` - Initial launch strategy (superseded)
- `CORRECTED_LAUNCH_STRATEGY_V2.md` - Revised launch strategy (superseded)
- `LAUNCH_SETUP_SUMMARY.md` - Early setup summary (superseded)
- `PROFESSIONAL_SECURITY_TIER_SUMMARY.md` - Security tier planning (superseded)
- **Superseded by:** [dev_docs/03-implementation/PRODUCTION_READINESS.md](../03-implementation/PRODUCTION_READINESS.md)

### Testing Artifacts (2025-10-14)
Historical testing documentation from Omnara integration:
- `TESTING_PROMPT_OMNARA.md` - Test prompts for Omnara integration
- `TESTING_REPORT_OMNARA.md` - Test results from Omnara integration

---

## How Documents End Up Here

### From Active Research
1. Experiment completed in 04-research/active/
2. Results documented
3. Decision: NOT to implement
4. Moved to 04-research/archive/ first
5. After 6 months: Consider moving to 99-archive/

### From Architecture Changes
1. New ADR created superseding old approach
2. Old architecture docs marked as deprecated
3. Moved to 99-archive/
4. Deprecation note added to top of file

### From Requirement Changes
1. spec.md updated with new requirements
2. Old requirement docs marked as superseded
3. Moved to 99-archive/

---

## Deprecation Notice Template

When moving a document to archive, add this to the top:

```markdown
# ⚠️ DEPRECATED - [Document Title]

**Status:** Deprecated
**Deprecated Date:** YYYY-MM-DD
**Reason:** [Why this is no longer current]
**Superseded By:** [Link to current documentation]

---

**DO NOT USE THIS DOCUMENT FOR IMPLEMENTATION**

This document is archived for historical reference only.

For current information, see:
- [spec.md](../spec.md)
- [Relevant current doc]

---

[Original document content below]
```

---

## Accessing Archive

### When to Read Archive
- Understanding evolution of the project
- Learning why certain approaches were abandoned
- Historical context for current decisions
- Onboarding to project history

### When NOT to Read Archive
- Planning new features (use spec.md)
- Implementing components (use 03-implementation/)
- Making architecture decisions (use 01-architecture/)

---

## Archive Maintenance

**Periodic Cleanup (Annually):**
- Review archive for documents with no historical value
- Consider permanent deletion after 2+ years
- Keep documents that explain major pivots
- Keep documents referenced in current ADRs

**Do NOT Delete:**
- Documents explaining why we pivoted
- Failed experiments with valuable learnings
- Major architecture alternatives we considered

---

**Back to:** [dev_docs/README.md](../README.md)
