# Product & Business Documentation (MVP Focus)

**Last Updated:** January 2025
**Purpose:** Product vision, user research, and market positioning for CodeRisk MVP

> **📘 Strategic Shift:** Simplified to local-first MVP based on market research. Complex cloud features moved to [99-archive/00-product-future-vision](../99-archive/00-product-future-vision/) for v2-v4.

---

## Strategic Positioning (January 2025)

**CodeRisk = Automated Due Diligence Layer**

Prevents regressions by automating what developers should check before committing:
- **Blast radius:** What depends on my changes?
- **Co-change patterns:** What files change together? (temporal coupling)
- **Ownership context:** Who wrote this? Who should review?
- **Incident history:** Has this pattern failed before?

**Complementary to PR Review Tools:**
- **Layer 1 (Pre-commit):** CodeRisk prevents regressions through temporal analysis
- **Layer 2 (PR review):** Greptile/CodeRabbit review code quality (bugs, style, best practices)

**Unique Moat:** Temporal coupling analysis (CO_CHANGED edges)—no other tool tracks which files evolve together.

---

## MVP Strategy (4-6 Weeks to Launch)

**Core Insight from Market:** Cursor ($500M ARR in 15 months) won by doing ONE thing exceptionally well, not by building complex infrastructure first.

**Our Approach:**
1. **Ship local-first tool** in 4-6 weeks
2. **Validate demand** with 100 active users
3. **Add cloud features** only if users request them

**Not building for MVP:**
- ❌ Cloud infrastructure (Neptune, K8s, portal)
- ❌ Multi-deployment modes
- ❌ Trust infrastructure / ARC database
- ❌ Complex pricing tiers

---

## What Goes Here

**Product Strategy:**
- MVP vision and mission (regression prevention focus, local-first)
- User personas (solo developers + small teams)
- Competitive positioning (complementary to Greptile/CodeRabbit, regression prevention positioning)

**User Experience:**
- Developer workflows (local git-based)
- UX design (CLI-first, pre-commit hooks)

**Business Model:**
- Simplified pricing (free + optional pro)
- MVP success metrics (100 users, 5 testimonials)

---

## Current Documents

### Core Product Vision

1. **[mvp_vision.md](mvp_vision.md)** - Regression prevention positioning for local-first MVP
   - **When to read:** Understanding CodeRisk's MVP strategy and regression prevention positioning
   - **Key topics:** Regression prevention, automated due diligence, temporal coupling moat, complementary to Greptile/CodeRabbit
   - **Changed from:** `vision_and_mission.md` (cloud platform vision archived)
   - **Updated:** January 2025 with regression prevention messaging

### User Research

2. **[user_personas.md](user_personas.md)** - Target users for MVP (solo devs + small teams)
   - **When to read:** Understanding who we're building for in MVP phase
   - **Key topics:** Ben (solo dev), Clara (small team lead), pain points, adoption triggers
   - **Changed:** Removed enterprise personas (Alex, Maya, Sam) - archived for v2

### Market Positioning

3. **[competitive_analysis.md](competitive_analysis.md)** - Regression prevention positioning vs competitors
   - **When to read:** Understanding competitive landscape and differentiation
   - **Key topics:** Complementary to Greptile/CodeRabbit (not competitive), temporal coupling unique moat, regression prevention focus
   - **Changed:** Updated positioning for regression prevention, complementary to PR review tools

### User Experience

4. **[developer_experience.md](developer_experience.md)** - UX design for local-first tool
   - **When to read:** Understanding optimal UX for pre-commit checks
   - **Key topics:** Pre-commit hooks, adaptive verbosity, local Neo4j setup
   - **Changed:** Removed cloud sync features, focused on local Docker setup

5. **[developer_workflows.md](developer_workflows.md)** - Git workflows for local tool
   - **When to read:** Understanding how CodeRisk fits into daily git workflows
   - **Key topics:** Local git patterns, solo workflows, small team collaboration
   - **Changed:** Removed enterprise workflows, focused on local-first

### Business Model

6. **[simplified_pricing.md](simplified_pricing.md)** - Free BYOK + optional pro tier
   - **When to read:** Understanding MVP pricing strategy
   - **Key topics:** Free unlimited (BYOK), optional Pro ($10/month for teams)
   - **Replaces:** `pricing_strategy.md` (complex multi-tier cloud pricing archived)

---

## Archived Documents (Future Vision)

**Moved to:** [99-archive/00-product-future-vision](../99-archive/00-product-future-vision/)

These documents represent v2-v4 vision (cloud platform, trust infrastructure, ARC database) that should only be built AFTER validating MVP demand:

- ❌ `strategic_moats.md` - Trust infrastructure, 7 Powers (v3-v4)
- ❌ `open_core_strategy.md` - Complex licensing (not needed for free tool)
- ❌ `pricing_strategy.md` - Multi-tier cloud pricing (replaced by simplified)
- ❌ `market_sizing.md` - TAM/SAM/SOM (premature for MVP)
- ❌ `go_to_market.md` - Stub (not developed yet)
- ❌ `customer_acquisition.md` - Stub (not developed yet)
- ❌ `success_metrics.md` - Multi-year OKRs (too complex for MVP)

**When to revisit:** After MVP validates demand (100+ users, 10+ paying teams)

---

## MVP Success Criteria (4-6 Weeks)

**Launch Goals:**
- 100 GitHub stars
- 50 active weekly users
- 5 "saved me from incident" testimonials
- <5% false positive rate
- <3s average check time

**Validation Signals:**
- Users request team features
- Users willing to pay for cloud sync
- Low churn (<10% monthly)
- Organic growth via word-of-mouth

**Decision Point:**
- ✅ If validated → Add cloud features (revisit archived docs)
- ❌ If not validated → Pivot or improve local tool

---

## Document Guidelines

### When to Add Here
- MVP product features with user validation
- Local-first workflow improvements
- User research from beta users
- Competitive positioning updates

### When NOT to Add Here
- Technical architecture (goes to 01-architecture/)
- Implementation details (goes to 03-implementation/)
- Future vision (goes to 99-archive/ until validated)
- Cloud features (wait for user demand)

### Format
- **Concise** - High-level strategy, not implementation
- **User-focused** - Why we're building, who it's for
- **Evidence-based** - User quotes, market data, metrics
- **No code** - Conceptual only, implementation in codebase

---

## Reading Order

**New Team Members:**
1. Start with `mvp_vision.md` - Understand MVP strategy
2. Read `user_personas.md` - Know who we're building for
3. Review `competitive_analysis.md` - Understand market position
4. Read `developer_experience.md` - Understand UX goals

**Before Feature Work:**
1. Check if feature is in MVP scope (mvp_vision.md)
2. Validate with user personas (user_personas.md)
3. Ensure fits developer workflows (developer_workflows.md)
4. Review UX impact (developer_experience.md)

**Before Marketing:**
1. Read `competitive_analysis.md` - Positioning
2. Review `user_personas.md` - Target audience
3. Check `simplified_pricing.md` - Pricing messaging

---

## Migration Path to Cloud (If Validated)

**Phase 1 (Weeks 1-6): MVP - Local Tool**
- Documents: Current 00-product docs
- Features: Local Neo4j, Docker setup, free BYOK

**Phase 2 (Months 3-6): Team Features**
- Add: Cloud sync docs (if users request it)
- Features: Shared team configs, cloud pattern library

**Phase 3 (Months 6-12): Scale**
- Revisit: Archived docs in 99-archive/00-product-future-vision/
- Features: Cloud infrastructure, multi-deployment, advanced pricing

**Phase 4 (Year 2+): Platform**
- Restore: strategic_moats.md, open_core_strategy.md
- Features: Trust infrastructure, ARC database, insurance

---

## Key Principles (From Market Research)

1. **Ship Fast, Validate Early**
   - Cursor: $500M ARR in 15 months by shipping simple IDE integration first
   - Don't build infrastructure before proving demand

2. **Local-First When Possible**
   - Faster (no network latency)
   - Cheaper (no cloud costs)
   - Privacy-friendly (data stays local)

3. **One Thing Exceptionally Well**
   - Own pre-commit risk assessment
   - Don't try to be everything (dashboard, PR review, etc)

4. **User-Driven Roadmap**
   - Build cloud features only if users request them
   - Don't assume users want complex infrastructure

---

**Related Documentation:**
- [SIMPLIFIED_MVP_PROPOSAL.md](../SIMPLIFIED_MVP_PROPOSAL.md) - Full MVP rationale
- [spec.md](../spec.md) - Technical requirements (being updated)
- [01-architecture/](../01-architecture/) - Technical architecture

---

**Last Updated:** January 2025
**Next Review:** After MVP launch (Week 7-8)
