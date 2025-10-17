# Market Sizing & TAM/SAM/SOM Analysis

**Last Updated:** October 2, 2025
**Owner:** Product Team
**Status:** Stub (Needs Development)

> **📘 Note:** This document is a stub. Content needs to be developed based on market research and industry data.

---

## Purpose

This document will provide a comprehensive market sizing analysis for CodeRisk, including:

- **TAM (Total Addressable Market)**: All developers who could benefit from pre-commit risk checking
- **SAM (Serviceable Available Market)**: Developers at companies that match our target profile
- **SOM (Serviceable Obtainable Market)**: Realistic market share we can capture in 3-5 years

---

## Placeholder Sections

### 1. Total Addressable Market (TAM)

**To Be Defined:**

**Global Developer Population:**
- Total professional developers worldwide: ~30M (source: Stack Overflow, GitHub)
- Developers at tech companies (not hobbyists): ~15M
- Developers writing production code (not just scripts): ~10M
- Developers using Git (not SVN, other VCS): ~9M

**Pricing Assumption:**
- Average price per developer: $20/month (blend of Starter $10, Pro $25, Enterprise $50)

**TAM Calculation:**
```
9M developers × $20/month × 12 months = $2.16B annual TAM
```

**Market Comparables:**
- GitHub Copilot: ~1M users × $10/month = $120M ARR
- SonarQube: 7M users, ~50K paid = $50-100M ARR (estimate)
- Datadog: ~25K customers, $2B+ ARR (enterprise-heavy)

---

### 2. Serviceable Available Market (SAM)

**To Be Defined:**

**Target Company Profiles:**
- Mid-size tech companies (50-500 engineers): ~50K companies globally
- Enterprise tech companies (500+ engineers): ~5K companies
- High-growth startups (10-50 engineers): ~200K companies

**Developer Count:**
- Mid-size: 50K companies × 150 avg engineers = 7.5M developers
- Enterprise: 5K companies × 2,000 avg engineers = 10M developers (but slower adoption)
- Startups: 200K companies × 20 avg engineers = 4M developers (but price-sensitive)

**Realistic SAM (filtering for fit):**
- Companies with CI/CD (mature dev practices): ~60% of above
- Companies using GitHub/GitLab (not Bitbucket/other): ~70%
- Companies willing to pay for dev tools: ~40%

**SAM Calculation:**
```
(7.5M + 4M) developers × 60% × 70% × 40% = 1.9M developers
1.9M developers × $20/month × 12 months = $456M annual SAM
```

---

### 3. Serviceable Obtainable Market (SOM)

**To Be Defined:**

**Market Share Assumptions (5-year horizon):**
- Year 1: 0.01% of SAM (1,900 developers) = $456K ARR
- Year 2: 0.05% of SAM (9,500 developers) = $2.3M ARR
- Year 3: 0.2% of SAM (38K developers) = $9.1M ARR
- Year 5: 1% of SAM (190K developers) = $45.6M ARR (aggressive but achievable)

**Comparable Penetration Rates:**
- SonarQube: ~0.7% of all developers (50K paid / 7M total)
- GitHub Copilot: ~3% of GitHub users (1M / 30M+)
- ESLint: ~20% of JS developers (ubiquitous, open source)

**SOM Assumptions:**
- Year 1-2: Land & expand with early adopters (bottom-up)
- Year 3-4: Enterprise sales motion kicks in (top-down)
- Year 5+: Category leader, network effects compound

---

### 4. Competitive Landscape Impact

**To Be Defined:**

**Existing Market Share:**
- SonarQube: $50-100M ARR (estimate), dominant in code quality
- Codescene: $10-20M ARR (estimate), niche in tech debt analysis
- Greptile: $2-5M ARR (estimate, early stage), growing in AI code search

**CodeRisk Differentiation:**
- Different workflow moment (pre-commit vs PR review/post-merge)
- Different technology (agentic graph search vs rules/dashboards)
- Different pricing (BYOK, 50-85% cheaper)

**Market Expansion vs Displacement:**
- 60% displacement (SonarQube users switching/adding CodeRisk)
- 40% expansion (new budget, previously no solution)

---

### 5. Growth Drivers

**To Be Defined:**

**Technology Trends:**
- AI/LLM adoption in dev tools (tailwind for agentic approach)
- Graph databases maturing (Neptune Serverless, Neo4j Aura)
- Shift-left movement (catch issues earlier in SDLC)

**Market Trends:**
- Rising cost of production incidents (downtime = lost revenue)
- Developer productivity focus (ship faster without breaking things)
- Tool consolidation fatigue (developers want fewer, better tools)

**Network Effects:**
- Shared public cache (React, Next.js) → instant value → viral growth
- Team adoption (one user → whole team via shared graph)
- Community (open source contributors → paid users)

---

### 6. Risk Factors & Market Headwinds

**To Be Defined:**

**Adoption Barriers:**
- Developer tool fatigue (yet another CLI tool?)
- Integration friction (adding to existing CI/CD pipelines)
- Enterprise sales cycles (6-12 months)

**Competitive Threats:**
- GitHub builds native risk analysis (distribution advantage)
- SonarQube adds LLM intelligence (incumbency advantage)
- New entrants (agentic dev tools are hot category)

**Economic Headwinds:**
- Tech layoffs → reduced budgets
- SaaS consolidation → fewer new tool purchases
- Open source alternatives (free competition)

---

## Placeholder Data Sources

**To Be Collected:**
- Stack Overflow Developer Survey (annual)
- GitHub State of the Octoverse report
- Gartner Application Security Testing market size
- Forrester Software Composition Analysis wave
- Private equity comps (Codescene acquisition, SonarQube funding)
- VC landscape (competitor funding rounds)

---

## Next Steps

1. **Primary Research**: Survey 100+ developers (willingness to pay, current tools)
2. **Industry Reports**: Purchase Gartner, Forrester reports on dev tools market
3. **Competitive Intel**: Research SonarQube, Codescene, Greptile revenue (via investor backchannel)
4. **TAM/SAM/SOM Model**: Build detailed Excel model with assumptions
5. **Investor Pitch Deck**: Synthesize into 2-slide market sizing section

---

## Related Documents

**Product & Business:**
- [vision_and_mission.md](vision_and_mission.md) - 3-5 year vision and market positioning
- [competitive_analysis.md](competitive_analysis.md) - Competitor market share estimates
- [pricing_strategy.md](pricing_strategy.md) - ARPU assumptions for TAM calculation
- [success_metrics.md](success_metrics.md) - Growth targets and OKRs

---

**Last Updated:** October 2, 2025
**Next Review:** TBD (after market research phase)
