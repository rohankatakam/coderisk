# CodeRisk: Vision & Mission

**Last Updated:** October 4, 2025
**Owner:** Product Team
**Status:** Active - Strategic Evolution to Trust Infrastructure

> **📘 Cross-reference:** See [spec.md](../spec.md) Section 1 for complete technical overview and requirements
> **📘 Strategic Context:** See [strategic_moats.md](strategic_moats.md) for 7 Powers implementation plan

---

## Vision Statement (Updated October 2025)

**CodeRisk is the trust infrastructure for AI-generated code—the authoritative standard that makes AI coding safe, measurable, and insurable.**

We envision a world where:
- Every AI-generated code block carries a trust certificate
- Architectural risks are catalogued like security vulnerabilities (CVE for Architecture)
- Companies learn from each other's incidents without sharing code
- "CodeRisk Verified" becomes the quality seal for AI code

### Strategic Evolution: From Tool to Infrastructure

**What Changed:**
- **Old:** Pre-flight check tool (analysis product)
- **New:** Trust infrastructure for AI code (platform + standard)

**Why This Matters:**
- Tools can be copied → Infrastructure becomes required
- Analysis is a feature → Trust is a business model
- Pre-commit timing → Trust layer spans entire SDLC

### 3-5 Year Vision (Revised)

**2025 (V1.0 - Pre-Flight Check Foundation):**
- Launch: Pre-commit risk assessment tool
- Own the pre-commit workflow moment
- 1,000+ teams using CodeRisk daily
- <3% false positive rate through intelligent investigation
- **Foundation for trust infrastructure**

**2026 (V2.0 - Trust Infrastructure):**
- Launch: CVE for Architecture (ARC database)
- Launch: AI code provenance certificates
- Launch: Privacy-preserving cross-org pattern learning
- Launch: AI code insurance (underwrite the risk)
- 10,000+ teams, 100 companies in pattern learning network
- **Become required infrastructure (not optional tool)**

**2027 (V3.0 - Trust Standard):**
- CodeRisk Trust Framework becomes industry standard (OWASP/CNCF)
- 10,000+ certified "Trust Engineers" globally
- AI tool vendors integrate for "CodeRisk Verified" badges
- State of AI Code Trust annual report (industry authority)
- **Own the narrative, define the category**

**2028+ (V4.0 - Trust Monopoly):**
- CodeRisk = Switzerland (neutral arbiter for all AI code)
- Trust insurance required for enterprise AI deployments
- Cross-industry incident knowledge graph (100K+ incidents)
- Platform: All dev tools integrate CodeRisk Trust API
- **Winner-take-most market position**

---

## Mission Statement (Updated)

**Our mission is to make AI-generated code trustworthy through privacy-preserving intelligence, open standards, and underwritten guarantees.**

We believe the AI coding revolution requires:
- ✅ **Trust infrastructure** - Not just analysis, but verifiable guarantees
- ✅ **Shared learning** - Cross-org pattern learning (privacy-preserving)
- ✅ **Open standards** - Industry framework owned by community, not vendor
- ✅ **Skin in the game** - Insurance that puts money behind predictions

### What We're Solving (Expanded)

**Original Problem:** Pre-commit uncertainty
- "Is this change safe to proceed?"
- Solved by: Pre-flight check tool

**Expanded Problem:** AI code trust crisis
- "Can we trust AI-generated code in production?"
- "How do we learn from incidents without sharing code?"
- "Who underwrites the risk of AI coding at scale?"

**Our Solution:**
1. **Trust Certificates** - Cryptographic proof of AI code safety
2. **Incident Knowledge Graph** - CVE for Architecture (10K+ incidents)
3. **Federated Learning** - Cross-org learning (no code leaves VPC)
4. **Trust Insurance** - Underwrite AI code risk ($0.10/check guarantees)
5. **Open Standard** - CodeRisk Trust Framework (industry-owned)

### Core Values (Updated)

**1. Developer-First (Unchanged)**
- Design for individual contributors, not management
- Optimize for daily workflow, not quarterly reports
- Respect developers' time (fast, accurate, low friction)

**2. Transparency (Enhanced)**
- BYOK model (see LLM costs)
- Explainable results (investigation traces)
- Open source core + **open trust standard**
- **NEW:** Public incident database (ARC catalog)

**3. Continuous Learning (Amplified)**
- Metric validation (learn from user overrides)
- Incident feedback → improve detection
- **NEW:** Cross-org pattern learning (privacy-preserving)
- **NEW:** Network effects (more users = smarter tool)

**4. Trust Through Verification (New)**
- Provenance certificates (cryptographic signing)
- Public audit trails (immutable trust records)
- Insurance underwriting (skin in the game)
- Open standards (community governance)

**5. Privacy-First (Enhanced)**
- Private repos stay private
- **NEW:** Federated learning (no code transmitted)
- **NEW:** Graph signatures (one-way hashes)
- **NEW:** Differential privacy (prevent re-identification)

### What We're Solving

**The Core Problem:**
Developers face acute uncertainty at the moment of creation: **"Is it safe to proceed?"**

Traditional tools fail to address:
- **Architectural regressions** in established patterns (invisible to linters)
- **Temporal coupling** and hidden dependencies (not in code structure)
- **Unknown unknowns** in blast radius and downstream impact (exponential complexity)
- **High false positive rates** from static rule-based systems (10-20% FP rate)
- **Pre-commit context** - most tools only work at PR review time (too late)

**Our Solution:**
Agentic graph search that combines:
1. **Hybrid graph** (tree-sitter AST + GitHub temporal data)
2. **LLM reasoning** (intelligent investigation, not brute force)
3. **Validated metrics** (only low-FP metrics, self-improving)
4. **Explainable evidence** (investigation traces, hop-by-hop decisions)

---

## Guiding Principles

### 1. Intelligent Performance Over Speed

**Principle:** Fast enough to be useful, smart enough to be accurate

- ✅ 2-5 second response via agentic graph search with hop limits
- ✅ NOT brute-force analysis (too slow) or shallow heuristics (too inaccurate)
- ✅ Hop-limited investigation (max 3-5 hops) prevents exponential cost

**Why this matters:**
- Developers won't use a tool that takes 30+ seconds
- Developers won't trust a tool with 10-20% false positives
- Sweet spot: Fast enough for inner loop, smart enough to find real risks

### 2. Zero-Friction Adoption

**Principle:** Works immediately, no complex setup

- ✅ Public repos: Instant access via shared cache (0-2s, auto-discovery)
- ✅ Private repos: GitHub OAuth + auto-discover team graph (<1 min)
- ✅ No manual configuration (no config files, no training period)
- ✅ Works after `git clone` (leverages existing repo structure)

**Why this matters:**
- Adoption dies with complex onboarding (see: many static analysis tools)
- Developers evaluate tools in first 30 seconds of use
- Network effects: Shared public cache benefits all users instantly

### 3. Explainable Intelligence

**Principle:** Every risk score includes clear evidence and reasoning

- ✅ Investigation trace: Shows agent's path (nodes visited, metrics calculated)
- ✅ Evidence-backed: Every finding links to graph data (not black-box ML)
- ✅ Actionable recommendations: "Add tests for X", "Review coupled file Y"
- ✅ Reproducible: Same change → same investigation → same result

**Why this matters:**
- Developers won't trust unexplainable "AI magic"
- Debugging false positives requires understanding why the tool flagged it
- Learning opportunity: Developers understand their codebase better

### 4. Shared Cache Model

**Principle:** Build once, benefit many (99% storage reduction for public repos)

- ✅ Public repos cached once (React, Kubernetes, Next.js)
- ✅ First user triggers build (5-10 min wait)
- ✅ Subsequent users: Instant access (0-2s)
- ✅ Reference counting: Archive when unused (30 days)

**Why this matters:**
- Economics: Can't afford to build graph for every user
- User experience: Instant access to popular repos
- Network effects: More users → more pre-built caches → better UX

### 5. Cloud-First, BYOK

**Principle:** We host infrastructure, user controls LLM costs

- ✅ User provides OpenAI/Anthropic API key (transparency, choice)
- ✅ We cover infrastructure ($2.30/user/month at 1K users)
- ✅ No LLM markup (user pays provider directly)
- ✅ Better unit economics (70-85% cost savings vs competitors)

**Why this matters:**
- Trust: Users see exactly what they pay for LLM calls
- Flexibility: Users can switch LLM providers
- Economics: Sustainable business model without markup

### 6. Privacy-First

**Principle:** Private repos stay private, customer data never leaves their control

- ✅ GitHub OAuth verification for private repo access
- ✅ Isolated Neptune databases per organization
- ✅ Enterprise: Self-hosted in customer VPC (100% data residency)
- ✅ Never log source code or API keys

**Why this matters:**
- Enterprise requirement: Cannot send proprietary code to third parties
- Compliance: GDPR, HIPAA, SOC2 requirements
- Trust: Security incident could destroy entire business

### 7. High Signal, Low Noise

**Principle:** Fewer robust metrics > many inaccurate metrics

- ✅ Target: <3% false positive rate (vs 10-20% industry standard)
- ✅ Metric validation: Track FP rates, auto-disable metrics >3% FP
- ✅ User feedback loop: Overrides improve metric selection
- ✅ Agent-selected: LLM picks best metrics for context (not static ruleset)

**Why this matters:**
- Developers ignore noisy tools (alert fatigue)
- One good signal > ten noisy signals
- Self-improving system builds trust over time

---

## Strategic Positioning: "Pre-Flight Check" Category

### The Workflow Moat

**Own the rapid, local, pre-commit sanity check workflow:**

```bash
git commit -am "wip" && crisk check  # 5s response, decisive answer
```

**Timing is everything:**
- **Pre-commit** (CodeRisk) - Developer hasn't pushed yet, can fix locally
- **Pre-PR** (still CodeRisk) - Developer hasn't requested review, no public commitment
- **PR review** (Greptile, others) - Too late, developer already invested, sunk cost fallacy
- **Post-merge** (Codescene, dashboards) - Way too late, incident already happened

**This addresses developer uncertainty in the chaotic inner loop, long before polished PRs.**

### Why This is Defensible

**Workflow habits are sticky:**
- Once `crisk check` is in muscle memory (like `git status`), hard to switch
- First-mover advantage in "pre-flight check" category
- Competitors anchored to later workflow stages (PR review, dashboards)

**Network effects:**
- Shared public cache (more users → more pre-built repos → faster onboarding)
- Team graphs (team builds graph once, all members benefit)
- Metric validation (more usage → better FP detection → higher accuracy)

**Technical moat:**
- Agentic graph search (not just static analysis)
- Hybrid graph (code structure + temporal data)
- Validated metrics (self-improving system)

---

## Differentiation from Competitors

### vs. Greptile (Conversational PR Co-pilot)

**Greptile positioning:** AI assistant that answers questions about your codebase during PR review

**CodeRisk differentiation:**
- ✅ **Timing:** Pre-commit (earlier) vs PR review (later)
- ✅ **Format:** Decisive check (binary answer) vs conversational (open-ended)
- ✅ **Speed:** 2-5s (instant) vs 30s-2min (interactive)
- ✅ **Use case:** Pre-flight safety check vs PR review assistant

**When to use which:**
- CodeRisk: "Is this change safe?" (before committing)
- Greptile: "How does this authentication flow work?" (during PR review)

### vs. Codescene (Health Monitor)

**Codescene positioning:** Long-term codebase health dashboard with technical debt tracking

**CodeRisk differentiation:**
- ✅ **Timing:** Immediate pre-commit vs batch analysis (daily/weekly)
- ✅ **Scope:** Specific change impact vs overall codebase trends
- ✅ **Latency:** 2-5s (real-time) vs hours (batch processing)
- ✅ **Workflow:** Developer inner loop vs management dashboard

**When to use which:**
- CodeRisk: "Is this specific change risky?" (developer tool)
- Codescene: "Which parts of our codebase are most problematic?" (manager tool)

### vs. SonarQube (Static Analysis)

**SonarQube positioning:** Rule-based static analysis for code quality and security

**CodeRisk differentiation:**
- ✅ **Intelligence:** Agentic graph search vs rule-based patterns
- ✅ **FP rate:** <3% (intelligent) vs 10-20% (rules)
- ✅ **Context:** Temporal coupling + graph vs syntax only
- ✅ **Evolution:** Self-improving metrics vs fixed rulesets

**When to use which:**
- CodeRisk: Architectural risk, hidden coupling, temporal patterns
- SonarQube: Code quality, security vulnerabilities, style issues

### Unique Value Proposition

**Only CodeRisk offers:**
1. **Cloud-first SaaS with shared public caching** (instant access to React, Kubernetes, etc.)
2. **Agentic graph search** (LLM-guided investigation, not brute-force)
3. **Hybrid graph** (tree-sitter AST + GitHub temporal data)
4. **BYOK model** (user provides API key, 70-85% cost savings)
5. **Pre-commit timing** (earliest possible intervention)

---

## Company Values

### 1. Developer-First

**Every decision starts with:** "Does this make developers' lives better?"

- Design for the individual contributor, not management
- Optimize for daily workflow, not quarterly reports
- Respect developers' time (fast, accurate, low friction)

### 2. Transparency

**No hidden costs, no black boxes, no surprises**

- BYOK model (see exactly what you pay for LLM calls)
- Explainable results (investigation traces, evidence)
- Open source core (CLI + agent framework)

### 3. Continuous Learning

**Build systems that get smarter over time**

- Metric validation (learn from user overrides)
- Incident feedback (link incidents → improve detection)
- Agent improvement (better metric selection over time)

### 4. Pragmatic Innovation

**Use cutting-edge tech when it solves real problems**

- LLMs for reasoning (not just hype)
- Graph databases for relationships (not just trendy)
- Proven low-FP metrics (not experimental)

---

## Product Principles

### Why We Make Certain Design Choices

**1. Why agentic investigation instead of static metrics?**
- Static metrics: High FP rate (10-20%), developers ignore them
- Agentic: Selective calculation (<3% FP), developers trust results
- Trade-off: Slightly slower (2-5s) but much more accurate

**2. Why cloud-first instead of local-first?**
- Cloud: Zero setup, shared cache, team collaboration
- Local: Complex setup, no sharing, isolated
- Trade-off: Requires internet, but 99% of developers have it

**3. Why BYOK instead of including LLM costs?**
- BYOK: Transparent pricing, user choice, better economics
- Included: Markup needed (2-3x), hidden costs
- Trade-off: Setup friction (paste API key), but worth it

**4. Why pre-commit instead of PR review?**
- Pre-commit: Earliest intervention, private (no embarrassment)
- PR review: Too late, public (sunk cost fallacy)
- Trade-off: Less context (no review discussion), but faster feedback

**5. Why hop-limited search instead of exhaustive?**
- Hop-limited: Fast (2-5s), controlled cost ($0.03-0.05/check)
- Exhaustive: Slow (30s+), expensive ($0.50+/check)
- Trade-off: Might miss deep relationships, but 3-5 hops covers 95% of cases

---

## Success Criteria

### What "Success" Looks Like

**Adoption metrics:**
- ✅ 1,000+ teams using CodeRisk daily within 12 months
- ✅ >10 checks/day/developer (habit formation)
- ✅ >80% public cache hit rate (network effects working)

**Quality metrics:**
- ✅ <3% false positive rate (better than 10-20% industry standard)
- ✅ >90% incident prediction accuracy (catch issues before production)
- ✅ >4.5/5 NPS (user satisfaction)

**Business metrics:**
- ✅ Enterprise contracts: >$5K/month each
- ✅ Sustainable unit economics: >70% gross margin
- ✅ Annual recurring revenue (ARR): $1M+ within 18 months

**Category creation:**
- ✅ "Pre-flight check" becomes recognized developer workflow category
- ✅ Competitors reposition to differentiate from CodeRisk
- ✅ Developers say "I crisk check before committing" (verb usage)

---

## What We're NOT Building (Out of Scope)

### V1.0 Exclusions

❌ **Web dashboard for graph visualization**
- Focus: CLI-first, settings portal only
- Rationale: Developers live in terminal, not browser

❌ **Real-time collaborative features**
- Focus: Individual developer workflow
- Rationale: Pre-commit is solo activity

❌ **IDE extensions**
- Focus: CLI tool first
- Rationale: Build foundation before integrations

❌ **Static predetermined metric sets**
- Focus: Agent-selected metrics
- Rationale: Intelligent selection beats fixed rules

❌ **Conversational interface**
- Focus: Decisive binary answer
- Rationale: Developers need action, not discussion

### Why These Boundaries Matter

**Focus:**
- Trying to do everything = doing nothing well
- Own one workflow moment (pre-commit) exceptionally well

**Differentiation:**
- By NOT building chat (Greptile does that)
- By NOT building dashboards (Codescene does that)
- We own "pre-flight check" exclusively

**Resource allocation:**
- Small team can't build everything
- Must deliver exceptional experience in narrow scope

---

## Future Evolution

### Phase 1 (2025): Own Pre-Commit Workflow
- Single developer, single repo
- CLI-first, cloud-hosted
- Shared public cache for OSS

### Phase 2 (2026): Team Intelligence
- Team-wide learning (shared metric validation)
- Cross-repo patterns (auth issues common across repos)
- CI/CD integration (automated gates)

### Phase 3 (2027-2028): Platform
- API for other tools (IDE extensions, review tools)
- Architectural quality score (team health metrics)
- Real-time collaboration (shared investigations)

---

## Related Documents

**Product & Business:**
- [user_personas.md](user_personas.md) - Detailed user profiles
- [competitive_analysis.md](competitive_analysis.md) - Market positioning
- [pricing_strategy.md](pricing_strategy.md) - Pricing tiers and rationale
- [success_metrics.md](success_metrics.md) - OKRs and tracking

**Technical:**
- [spec.md](../spec.md) - Complete requirements specification
- [01-architecture/agentic_design.md](../01-architecture/agentic_design.md) - Agent investigation strategy
- [01-architecture/graph_ontology.md](../01-architecture/graph_ontology.md) - Graph schema

---

**Last Updated:** October 2, 2025
**Next Review:** January 2026 (quarterly review)
