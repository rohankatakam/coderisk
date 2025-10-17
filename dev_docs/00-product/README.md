# Product & Business Documentation

**Purpose:** Product vision, user research, market analysis, competitive positioning

> **📘 For AI agents:** Before creating/updating product docs, read [DOCUMENTATION_WORKFLOW.md](../DOCUMENTATION_WORKFLOW.md) to determine if this is the right location and which document to update.

---

## What Goes Here

**Product Strategy:**
- Product vision and mission statements
- Value propositions
- Market analysis and sizing
- Go-to-market strategy

**User Research:**
- User personas (Ben the Developer, Clara the Tech Lead)
- User journey maps
- Pain points and jobs-to-be-done
- User feedback and interviews

**Competitive Analysis:**
- Competitor comparisons (vs Greptile, Codescene, etc.)
- Feature matrices
- Pricing analysis
- Market positioning

**Business Planning:**
- Pricing strategy and tiers
- Revenue models
- Customer acquisition strategy
- Growth metrics and targets

---

## Document Guidelines

### When to Add Here
- Product feature proposals with business justification
- Market research findings
- User persona updates
- Competitive intelligence

### When NOT to Add Here
- Technical architecture (goes to 01-architecture/)
- Implementation details (goes to 03-implementation/)
- Research experiments (goes to 04-research/)

### Format
- **Be data-driven** - Include market data, user quotes, metrics
- **Business-focused** - Why we're building, who it's for, market opportunity
- **User-centric** - Focused on user needs and pain points

---

## Current Documents

### Core Product Documentation

1. **[vision_and_mission.md](vision_and_mission.md)** - Product vision, mission, guiding principles, strategic positioning
   - **When to read:** Understanding CodeRisk's strategic direction and "pre-flight check" category creation
   - **Key topics:** 3-5 year vision, differentiation from competitors, guiding principles, company values

2. **[user_personas.md](user_personas.md)** - Detailed user profiles (Ben, Clara, Alex, Maya, Sam)
   - **When to read:** Understanding target users, pain points, and value propositions
   - **Key topics:** User goals, daily workflows, pain points, adoption triggers, success metrics

3. **[developer_workflows.md](developer_workflows.md)** - Git workflows, vibe coding integration, team-size adoption patterns
   - **When to read:** Understanding how CodeRisk integrates into real developer workflows (manual coding + AI-assisted)
   - **Key topics:** Git patterns, vibe coding (Claude Code/Cursor), team size workflows (solo → enterprise), OSS workflows, workflow injection points

4. **[developer_experience.md](developer_experience.md)** - UX design for seamless automatic risk assessment
   - **When to read:** Understanding the optimal user experience for vibe coding and automatic pre-commit checks
   - **Key topics:** Pre-commit hooks, adaptive verbosity, team-size UX, vibe coding feedback loops, error messages, performance targets, CLI patterns

5. **[competitive_analysis.md](competitive_analysis.md)** - Market positioning vs Greptile, Codescene, SonarQube
   - **When to read:** Understanding competitive landscape and differentiation strategy
   - **Key topics:** Feature comparison, pricing comparison, competitive moats, win/loss analysis

6. **[pricing_strategy.md](pricing_strategy.md)** - Pricing tiers (Free, Starter, Pro, Enterprise), BYOK model
   - **When to read:** Understanding pricing rationale and unit economics
   - **Key topics:** BYOK model, cost breakdown, competitive pricing, discount strategies

7. **[success_metrics.md](success_metrics.md)** - OKRs, KPIs, performance targets
   - **When to read:** Understanding success criteria and measurement strategy
   - **Key topics:** Performance metrics, quality metrics (FP rate), business metrics (MRR, churn), OKRs by quarter

8. **[strategic_moats.md](strategic_moats.md)** - 7 Powers framework, competitive moats, ARC database strategy
   - **When to read:** Understanding long-term competitive advantages and strategic positioning
   - **Key topics:** Cornered Resource (ARC database), Counter-Positioning (trust infrastructure), Network Effects, Brand building

9. **[open_core_strategy.md](open_core_strategy.md)** - Open source vs proprietary licensing, community strategy
   - **When to read:** Understanding what's open source (MIT License) vs proprietary (cloud platform)
   - **Key topics:** License structure, open core rationale, community strategy, revenue model, risk mitigation

### Future Documentation (Stubs)

10. **[go_to_market.md](go_to_market.md)** - Launch strategy, distribution channels, marketing campaigns *(Stub)*
   - **Status:** Needs development based on customer interviews

11. **[market_sizing.md](market_sizing.md)** - TAM/SAM/SOM analysis, market opportunity *(Stub)*
   - **Status:** Needs development based on market research

12. **[customer_acquisition.md](customer_acquisition.md)** - Acquisition channels, CAC, LTV, growth loops *(Stub)*
   - **Status:** Needs development based on channel testing

---

## Template: User Persona

```markdown
# User Persona: [Name]

**Role:** [Title/Role]
**Company Type:** [Startup/Mid-size/Enterprise]
**Team Size:** [Number]

## Background
[Brief background about this persona]

## Goals
- [Primary goal 1]
- [Primary goal 2]

## Pain Points
- [Pain point 1]
- [Pain point 2]

## Current Workflow
[How they currently solve this problem]

## CodeRisk Value
[How CodeRisk helps this persona]

## Quotes
> "[Direct quote from user research]"
```

---

**Back to:** [dev_docs/README.md](../README.md)
