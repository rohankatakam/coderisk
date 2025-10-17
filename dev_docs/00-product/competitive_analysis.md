# Competitive Analysis

**Last Updated:** October 4, 2025
**Owner:** Product Team
**Status:** Active - Counter-Positioning Strategy Deployed

> **📘 Cross-reference:** See [spec.md](../spec.md) Section 1.4 for technical comparison details
> **📘 Strategic Context:** See [strategic_moats.md](strategic_moats.md) for 7 Powers counter-positioning

---

## Executive Summary (Updated October 2025)

**Strategic Repositioning:** CodeRisk has evolved from "Pre-Flight Check tool" to **"Trust Infrastructure for AI-Generated Code"** —a fundamentally different market position that competitors cannot adopt without destroying their existing business models.

**Previous Positioning (Vulnerable):**
- Category: Pre-flight check tool (new workflow moment)
- Differentiation: Timing (pre-commit) + intelligence (agentic)
- **Problem:** Competitors could add pre-commit mode easily

**New Positioning (Defensible):**
- Category: **The ARC Database** (CVE for Architecture) + Trust infrastructure
- Business Model: Platform + insurance + certification (not just analysis)
- **Moat:** Competitors cannot copy without cannibalizing existing revenue
- **Bootstrap:** GitHub mining strategy provides 100 ARC entries in 8 weeks (10x faster than organic)

**Key Competitive Advantages (Updated):**
1. **Cornered Resource**: The ARC Database - 100 patterns from 10,000 GitHub incidents (8-week bootstrap)
2. **Counter-Positioning**: Trust infrastructure business model they can't adopt
3. **Network Effects**: Cross-org learning bootstrapped by GitHub mining (perceived Day 1 value)
4. **Brand**: "First ARC Database" category creation (first-mover advantage)
5. **Economics**: BYOK model + insurance revenue (70-85% cost advantage)

> **📊 Strategic Update:** GitHub mining analysis complete - see [github_mining_7_powers_alignment.md](../04-research/active/github_mining_7_powers_alignment.md) for 85% alignment score with 7 Powers framework

---

## Competitive Landscape

### Market Segmentation

```
┌─────────────────────────────────────────────────────────┐
│                    Timing in Workflow                    │
├──────────────┬──────────────┬──────────────┬────────────┤
│  Pre-Commit  │   Pre-PR     │  PR Review   │ Post-Merge │
├──────────────┼──────────────┼──────────────┼────────────┤
│  CodeRisk    │  CodeRisk    │  Greptile    │  Codescene │
│  (2-5s)      │  (2-5s)      │  (30s-2min)  │  (hours)   │
│              │              │  SonarQube   │  Datadog   │
│              │              │  (minutes)   │  (passive) │
└──────────────┴──────────────┴──────────────┴────────────┘
```

**CodeRisk's Unique Position:**
- **Earliest intervention point** (before developer commits publicly)
- **Private feedback loop** (no embarrassment, no sunk cost fallacy)
- **Rapid iteration** (fix locally, recheck in 5 seconds)

---

## Direct Competitors

### 1. Greptile (Conversational PR Co-pilot)

**Company Profile:**
- **Founded:** 2023
- **Funding:** Seed stage (~$2M)
- **Focus:** AI-powered PR review assistant
- **Customer Base:** 100+ teams (estimate)

**Product Positioning:**
> "Chat with your codebase during PR review"

**Feature Comparison:**

| Feature | CodeRisk | Greptile |
|---------|----------|----------|
| **Timing** | Pre-commit | PR review |
| **Format** | Decisive check (binary) | Conversational (open-ended) |
| **Latency** | 2-5 seconds | 30 seconds - 2 minutes |
| **Use Case** | "Is this safe?" | "How does X work?" |
| **Data Model** | Hybrid graph (AST + temporal) | Code embeddings + RAG |
| **Investigation** | Agentic graph search | Vector similarity search |
| **Setup** | 0-1 min (OAuth) | 2-5 min (repo indexing) |
| **Pricing** | $10-50/user/month | $30-100/user/month (estimate) |
| **LLM Cost** | User BYOK (transparent) | Included (markup) |

**Strengths:**
- ✅ Strong conversational UX (feels like ChatGPT for code)
- ✅ Good for exploratory questions during PR review
- ✅ Integrates with GitHub PR interface
- ✅ Handles "How does X work?" questions well

**Weaknesses:**
- ❌ Too late in workflow (developer already committed, sunk cost)
- ❌ Not optimized for rapid yes/no decisions
- ❌ Expensive (includes LLM costs with markup)
- ❌ Requires back-and-forth conversation (not instant)

**When Greptile Wins:**
- PR reviewer needs to understand complex code flow
- Team wants conversational debugging experience
- Budget allows for higher per-user costs

**When CodeRisk Wins:**
- Developer needs instant pre-commit risk check
- Team values early feedback (before PR)
- Budget-conscious teams (BYOK model)

**Differentiation Strategy:**
- **Timing**: "Use CodeRisk *before* you commit, use Greptile *during* PR review"
- **Workflow**: CodeRisk is reflexive (like `git status`), Greptile is investigative
- **Integration**: Not competitive—teams can use both

---

### 2. Codescene (Health Monitor & Technical Debt Dashboard)

**Company Profile:**
- **Founded:** 2015
- **Funding:** Bootstrapped → Acquired (2023)
- **Focus:** Long-term codebase health monitoring
- **Customer Base:** 500+ enterprise teams

**Product Positioning:**
> "X-ray for your codebase: Find hotspots and technical debt"

**Feature Comparison:**

| Feature | CodeRisk | Codescene |
|---------|----------|-----------|
| **Timing** | Pre-commit (real-time) | Post-merge (batch) |
| **Scope** | Specific change impact | Codebase-wide trends |
| **Latency** | 2-5 seconds | Hours (daily batch) |
| **User** | Individual developer | Tech lead / manager |
| **Workflow** | Inner loop (local) | Management dashboard |
| **Intelligence** | LLM-guided investigation | Statistical analysis |
| **Data Model** | Graph (3 hops) | Time-series metrics |
| **Actionability** | "Fix before commit" | "Prioritize refactoring" |
| **Pricing** | $10-50/user/month | $50-150/user/month |

**Strengths:**
- ✅ Mature product (9+ years in market)
- ✅ Strong visualization (hotspot maps)
- ✅ Good for technical debt prioritization
- ✅ Enterprise-ready (compliance, security)
- ✅ Predictive analytics (which code will cause issues)

**Weaknesses:**
- ❌ Not real-time (batch processing, not inner loop)
- ❌ Optimized for managers, not developers
- ❌ Expensive setup (requires historical data ingestion)
- ❌ Dashboard-centric (not CLI-friendly)

**When Codescene Wins:**
- Management needs codebase health dashboard
- Team wants to prioritize technical debt
- Large enterprise with budget for premium tools

**When CodeRisk Wins:**
- Developer needs instant feedback before committing
- Team wants developer-first tool (not manager dashboard)
- Startup/mid-size company prioritizing speed

**Differentiation Strategy:**
- **User**: "Codescene is for your *manager*, CodeRisk is for *you*"
- **Timing**: "Codescene tells you where debt is, CodeRisk prevents new debt"
- **Integration**: Complementary—CodeRisk prevents issues Codescene would later flag

---

### 3. SonarQube (Static Analysis & Code Quality)

**Company Profile:**
- **Founded:** 2008
- **Funding:** Private equity backed
- **Focus:** Code quality and security scanning
- **Customer Base:** 7M+ developers, 400K+ organizations

**Product Positioning:**
> "Clean Code for developers, Safe Code for organizations"

**Feature Comparison:**

| Feature | CodeRisk | SonarQube |
|---------|----------|-----------|
| **Intelligence** | Agentic graph search | Rule-based patterns |
| **False Positive Rate** | <3% (LLM-validated) | 10-20% (industry standard) |
| **Context** | Temporal coupling + graph | Syntax + known vulnerabilities |
| **Evolution** | Self-improving (metric validation) | Fixed rulesets (manual updates) |
| **Risk Types** | Architectural, coupling, incidents | Security, bugs, code smells |
| **Integration** | CLI + CI/CD | CI/CD + IDE plugins |
| **Pricing** | $10-50/user/month | Free (Community) → $150+/user/month (Enterprise) |

**Strengths:**
- ✅ Industry standard (7M+ users)
- ✅ Comprehensive rules (1000+ patterns)
- ✅ Security focus (OWASP, CWE coverage)
- ✅ IDE integration (real-time linting)
- ✅ Free tier (Community Edition)

**Weaknesses:**
- ❌ High false positive rate (10-20%)
- ❌ Rule-based (misses architectural issues)
- ❌ No temporal coupling analysis
- ❌ Alert fatigue (developers ignore warnings)

**When SonarQube Wins:**
- Team needs security vulnerability scanning
- Compliance requires OWASP coverage
- Budget allows for free tier (small teams)

**When CodeRisk Wins:**
- Team frustrated with SonarQube's false positives
- Need architectural risk analysis (not just code quality)
- Want intelligent investigation (not rigid rules)

**Differentiation Strategy:**
- **Focus**: "SonarQube finds *bugs*, CodeRisk finds *architectural risks*"
- **Accuracy**: "SonarQube has 10-20% FP, CodeRisk has <3% FP"
- **Integration**: Complementary—run both (SonarQube for security, CodeRisk for architecture)

---

## Indirect Competitors

### 4. GitHub Advanced Security (GHAS)

**Positioning:** Native GitHub security scanning
**Overlap:** Code scanning, dependency alerts
**Differentiation:** CodeRisk focuses on architectural risk, not security vulnerabilities

**When GHAS Wins:**
- Already using GitHub Enterprise
- Need Dependabot, secret scanning

**When CodeRisk Wins:**
- Need architectural coupling analysis
- Want pre-commit feedback (GHAS is CI/CD)

---

### 5. vFunction (Application Modernization & Architectural Intelligence)

**Company Profile:**
- **Founded:** 2017
- **Funding:** $60M+ (Series B)
- **Focus:** Monolith-to-microservices transformation
- **Customer Base:** Enterprise (Fortune 500 focus)

**Product Positioning:**
> "Architectural intelligence for legacy application modernization"

**Feature Comparison:**

| Feature | CodeRisk | vFunction |
|---------|----------|-----------|
| **Timing** | Pre-commit (developer) | Post-deployment (architect) |
| **Target User** | Individual developer | Architect / CTO |
| **Use Case** | "Is this change safe?" | "How do we modernize this monolith?" |
| **Scope** | Single file/change | Entire application |
| **Intelligence** | Pre-commit risk assessment | Architecture refactoring recommendations |
| **Integration** | CLI + pre-commit hook | Platform + GenAI assistant integration |
| **Latency** | 2-5 seconds | Hours-days (full analysis) |
| **Pricing** | $10-50/user/month | Enterprise (6-7 figures/year estimate) |
| **AI Integration** | BYOK (user's LLM) | Platform-provided (architectural context) |

**Strengths:**
- ✅ Deep architectural analysis (static + dynamic code analysis)
- ✅ Automated service extraction (monolith → microservices)
- ✅ Enterprise-proven (15x faster modernization claim)
- ✅ Framework upgrade capabilities (Java, .NET)
- ✅ Architectural observability (drift detection)

**Weaknesses:**
- ❌ Not real-time (batch processing, hours-days)
- ❌ Enterprise-only (not accessible to individual developers)
- ❌ Platform-dependent (not CLI-first)
- ❌ Expensive (6-7 figure annual contracts)
- ❌ Modernization-focused (not day-to-day development)

**When vFunction Wins:**
- Large enterprise modernizing legacy monoliths
- Multi-year cloud migration projects
- Need architectural refactoring roadmap
- Budget for 6-7 figure platform

**When CodeRisk Wins:**
- Daily development workflow (not one-time modernization)
- Individual developers need instant feedback
- Startup/mid-size company budget
- Pre-commit safety checks (not architectural transformation)

**Differentiation Strategy:**
- **Workflow Stage**: "vFunction is for *planning* modernization (months), CodeRisk is for *daily* development (seconds)"
- **User**: "vFunction is for your *architect*, CodeRisk is for your *developer*"
- **Scope**: "vFunction analyzes entire applications, CodeRisk analyzes individual changes"
- **Integration**: Not competitive—vFunction modernization + CodeRisk daily checks complement each other

**Key Insight:**
vFunction and CodeRisk target different workflow moments and users. vFunction helps enterprises plan multi-year modernization projects (top-down, architect-driven). CodeRisk helps developers make safe changes daily (bottom-up, developer-driven). A company could use vFunction to plan microservices extraction, then use CodeRisk to ensure developers don't introduce new coupling during the transition.

---

### 6. Datadog / New Relic (Observability Dashboards)

**Positioning:** Post-production monitoring and alerting
**Overlap:** Incident detection (reactive, not proactive)
**Differentiation:** CodeRisk prevents incidents *before* they happen

**When Observability Tools Win:**
- Need runtime metrics, APM, log aggregation
- Team already invested in observability stack

**When CodeRisk Wins:**
- Want to prevent incidents (not just detect them)
- Shift-left strategy (catch issues pre-commit)

---

## Competitive Matrix

| Capability | CodeRisk | Greptile | Codescene | SonarQube | vFunction |
|------------|----------|----------|-----------|-----------|-----------|
| **Pre-Commit Timing** | ✅ Core | ❌ No | ❌ No | ⚠️ IDE plugin only | ❌ No |
| **Agentic Investigation** | ✅ Core | ⚠️ RAG only | ❌ No | ❌ No | ⚠️ Architectural only |
| **Temporal Coupling** | ✅ Core | ❌ No | ✅ Yes | ❌ No | ⚠️ Via analysis |
| **Incident Prediction** | ✅ Core | ❌ No | ⚠️ Hotspots only | ❌ No | ❌ No |
| **Low FP Rate (<3%)** | ✅ Target | ⚠️ Varies | ⚠️ ~5-10% | ❌ ~10-20% | ⚠️ Unknown |
| **Shared Public Cache** | ✅ Core | ❌ No | ❌ No | ❌ No | ❌ No |
| **BYOK Model** | ✅ Core | ❌ No | ❌ No | N/A | ❌ No |
| **CLI-First** | ✅ Core | ⚠️ Web-first | ❌ Dashboard | ⚠️ CI/CD | ❌ Platform |
| **Graph Database** | ✅ Neptune | ⚠️ Vector DB | ❌ SQL | ❌ Files | ✅ Yes |
| **LLM Integration** | ✅ Phase 2 | ✅ Core | ❌ No | ❌ No | ✅ GenAI |
| **Real-Time (<5s)** | ✅ Core | ⚠️ 30s-2min | ❌ Batch | ⚠️ Varies | ❌ Hours-days |
| **Modernization Focus** | ❌ No | ❌ No | ❌ No | ❌ No | ✅ Core |
| **Target User** | Developer | Developer | Tech Lead | Developer | Architect/CTO |
| **Price Range** | $10-50/user/mo | $30-100/user/mo | $50-150/user/mo | Free-$150/user/mo | $100K-500K+/yr |

**Legend:**
- ✅ Strong capability
- ⚠️ Partial capability
- ❌ Not supported

---

## Pricing Comparison

| Product | Free Tier | Starter | Pro | Enterprise |
|---------|-----------|---------|-----|------------|
| **CodeRisk** | Public repos only | $10/user/month | $25/user/month | $50/user/month |
| **Greptile** | Limited (10 queries) | $30/user/month (est) | $70/user/month (est) | Custom |
| **Codescene** | No | $50/user/month | $100/user/month | $150+/user/month |
| **SonarQube** | Community (free) | N/A | $10/user/month | $150/user/month |

**CodeRisk Cost Advantage:**
- **BYOK Model**: User pays OpenAI/Anthropic directly ($0.03-0.05/check)
- **No LLM Markup**: Competitors include LLM costs with 2-3x markup
- **Shared Cache**: First user pays graph build cost, subsequent users get instant access
- **Infrastructure Only**: We charge $2.30/user/month infrastructure, user controls LLM spend

**Example Total Cost (100 checks/month):**
- **CodeRisk**: $10/month (plan) + $3-5/month (LLM) = **$13-15/month**
- **Greptile**: $30-70/month (all-inclusive, but LLM marked up) = **$30-70/month**
- **Codescene**: $50-100/month (no LLM) = **$50-100/month**

**Savings: 50-85% vs competitors**

---

## Market Positioning Map

```
                    High Intelligence (LLM/Agentic)
                              ▲
                              │
                         CodeRisk
                         (Pre-commit)
                              │
                         Greptile
                         (PR Review)
                              │
    Fast ◄────────────────────┼────────────────────► Slow
    (Real-time)               │                  (Batch)
                              │
                         SonarQube              Codescene
                         (CI/CD)                (Dashboard)
                              │
                              ▼
                    Low Intelligence (Rules/Stats)
```

**CodeRisk Positioning:**
- **Top-Left Quadrant**: High intelligence + Fast (unique position)
- **Workflow Moat**: Earliest intervention point
- **Technical Moat**: Only agentic graph search in market

---

## Competitive Moats

### 1. Workflow Habit Formation

**Thesis:** Once `crisk check` becomes muscle memory (like `git status`), developers won't switch

**Evidence:**
- Git displaced SVN by owning local workflow
- ESLint/Prettier displaced JSLint by being faster
- Pre-commit hooks are sticky (run automatically)

**Defensibility:**
- First-mover advantage in "pre-flight check" category
- Competitors anchored to later workflow stages (PR review, dashboards)
- Switching cost: Retrain muscle memory + lose shared cache

### 2. Network Effects (Shared Public Cache)

**Thesis:** More users → More pre-built repos → Faster onboarding → More users

**Mechanics:**
1. First user of React triggers graph build (5-10 min wait)
2. Subsequent users get instant access (0-2s)
3. Popular repos (React, Next.js, Kubernetes) cached by Day 1
4. Long tail repos cached over time (organic growth)

**Defensibility:**
- 99% storage reduction (only build once per repo)
- Instant access to popular OSS repos (competitive UX advantage)
- New entrants must rebuild cache from scratch

### 3. Technical Moat (Agentic Graph Search)

**Thesis:** Agentic graph search is fundamentally different from competitors' approaches

**Key Differences:**
- **vs Rule-Based (SonarQube)**: Selective calculation, not brute force
- **vs Vector Search (Greptile)**: Graph relationships, not embedding similarity
- **vs Statistical (Codescene)**: LLM reasoning, not fixed models

**Defensibility:**
- Requires hybrid graph (AST + temporal data) infrastructure
- Requires metric validation framework (self-improving system)
- Requires agentic reasoning (not just prompting)

### 4. Data Moat (Incident Linking)

**Thesis:** Manual incident → commit linking creates high-quality training data

**Data Flywheel:**
1. User links Incident #123 → Commit abc123 (manual, high quality)
2. System learns patterns (auth issues → timeout incidents)
3. Future incidents auto-suggest similar commits (semi-automated)
4. More usage → Better patterns → Higher accuracy

**Defensibility:**
- Manual linking is tedious (competitors won't do it)
- High-quality data beats quantity (low FP rate)
- Accumulated over time (new entrants start from zero)

---

## Win/Loss Analysis

### When CodeRisk Wins

**Scenario 1: Developer-First Culture**
- **Customer Profile**: Startup/mid-size tech companies (50-500 engineers)
- **Decision Maker**: Engineering manager or tech lead (bottom-up adoption)
- **Trigger**: Developer frustration with existing tools (SonarQube noise, slow PR reviews)
- **Objection Handled**: "We already use SonarQube" → "CodeRisk complements SonarQube (architecture vs security)"

**Scenario 2: High Incident Rate**
- **Customer Profile**: Fast-growing companies with production incidents
- **Decision Maker**: VP Engineering or CTO (top-down mandate)
- **Trigger**: Recent major outage, post-mortem identified "unknown coupling"
- **Objection Handled**: "We have observability tools" → "CodeRisk prevents incidents, not just detects them"

**Scenario 3: Budget-Conscious Teams**
- **Customer Profile**: Bootstrapped startups, small teams
- **Decision Maker**: Founder or lead developer
- **Trigger**: Sticker shock from Greptile/Codescene pricing
- **Objection Handled**: "LLM costs are too high" → "You control LLM spend with BYOK"

### When CodeRisk Loses

**Scenario 1: Management-Driven Dashboard Needs**
- **Lost To**: Codescene
- **Reason**: VP Engineering wants codebase health dashboard, not developer CLI tool
- **Learning**: Better positioning for management (add lightweight dashboard view)

**Scenario 2: Security Compliance Requirements**
- **Lost To**: SonarQube + GHAS
- **Reason**: Must satisfy OWASP/CWE compliance (CodeRisk doesn't scan for CVEs)
- **Learning**: Integrate with SonarQube (complementary, not competitive)

**Scenario 3: Conversational UX Preference**
- **Lost To**: Greptile
- **Reason**: Team prefers chat-based exploration over binary risk checks
- **Learning**: Not our target user (we optimize for speed, not conversation)

---

## Competitive Response Strategies

### If Greptile Adds Pre-Commit Mode

**Threat Level**: Medium
**Response**:
1. **Workflow Moat**: Already own pre-commit habit (first-mover advantage)
2. **Speed**: Our 2-5s is faster than their 30s-2min (architectural difference)
3. **Economics**: BYOK model is 50-70% cheaper

### If SonarQube Adds LLM Intelligence

**Threat Level**: High (large user base)
**Response**:
1. **False Positive Rate**: Emphasize our <3% FP vs their historical 10-20%
2. **Graph Intelligence**: We have temporal coupling + incident data (they don't)
3. **Shared Cache**: Our network effects (they rebuild for every org)

### If Codescene Adds Real-Time Mode

**Threat Level**: Low (different target user)
**Response**:
1. **Developer-First**: We're CLI-first, they're dashboard-first (UX mismatch)
2. **Cost**: Our $10-25/user vs their $50-150/user
3. **Setup**: Our 0-1 min vs their historical data requirements

### If GitHub Builds Native Graph Analysis

**Threat Level**: High (distribution advantage)
**Response**:
1. **Agentic Intelligence**: We have LLM-guided investigation (they'd have rules)
2. **Shared Cache**: We support multi-repo (GitHub is repo-siloed)
3. **BYOK**: User controls LLM costs (GitHub would bundle/markup)
4. **Enterprise**: Offer on-prem (GitHub Cloud-only for GHAS)

### If vFunction Adds Real-Time Developer Checks

**Threat Level**: Very Low (different market)
**Response**:
1. **Target User**: We're for developers (daily), they're for architects (multi-year projects)
2. **Price Point**: We're $10-50/user/month, they're $100K-500K+/year (100x difference)
3. **Workflow**: We're pre-commit (seconds), they're planning (months)
4. **Integration**: Not competitive—companies use vFunction for modernization planning, then use CodeRisk to protect daily changes during transformation

---

## Market Opportunities

### Whitespace (No Good Solution Today)

**1. Temporal Coupling Detection**
- **Problem**: Files that change together but have no structural dependency
- **Current Solutions**: None (Codescene has basic hotspots, but not real-time)
- **CodeRisk Advantage**: Only tool with graph-based temporal analysis

**2. Pre-Commit Architectural Guardrails**
- **Problem**: Architectural patterns regress over time (no enforcement)
- **Current Solutions**: Manual PR reviews (inconsistent, slow)
- **CodeRisk Advantage**: Only tool at pre-commit workflow moment

**3. Incident-Driven Risk Assessment**
- **Problem**: Past incidents don't inform future risk checks
- **Current Solutions**: Post-mortem docs (unused, siloed)
- **CodeRisk Advantage**: Only tool linking incidents → code changes

---

## Recommended Positioning Statements

### Primary Message (Developers)
> **"CodeRisk: The ARC Database (CVE for Architecture)"**
>
> Pre-commit risk checking backed by 100 architectural patterns from analyzing 10,000 real production incidents.
> Run `crisk check` before committing—instant risk analysis in 2-5 seconds.

### Secondary Message (Engineering Managers)
> **"Learn from 10,000 production incidents before they happen to you"**
>
> CodeRisk's ARC database catalogs architectural risks from 1,000 top GitHub repos, preventing known incident patterns before they reach production.

### Against Greptile
> **"Use CodeRisk before you commit, Greptile during PR review"**
>
> CodeRisk is your pre-flight safety check. Greptile is your in-flight assistant. Use both.

### Against Codescene
> **"Codescene is for your manager. CodeRisk is for you."**
>
> Codescene tells you where technical debt is (dashboard). CodeRisk prevents new debt (CLI).

### Against SonarQube
> **"SonarQube finds bugs. CodeRisk finds architectural risks."**
>
> SonarQube: Security + code quality (10-20% FP rate)
> CodeRisk: Coupling + blast radius + incidents (<3% FP rate)

### Against vFunction
> **"vFunction modernizes applications. CodeRisk protects daily changes."**
>
> vFunction is for multi-year modernization projects (architect-driven, 6-7 figures)
> CodeRisk is for daily development (developer-driven, $10-50/user/month)
>
> Use both: vFunction plans the transformation, CodeRisk prevents new coupling during it

---

## Analyst & Media Positioning

### Category Creation: "Pre-Flight Check"

**Thesis:** Create new category between "Static Analysis" and "PR Review"

**Talking Points:**
- ✅ "Pre-flight check" is a new developer workflow category
- ✅ Analogous to `git status` (reflexive, habitual, fast)
- ✅ Addresses "moment of uncertainty" before committing
- ✅ Complements existing tools (not replacement)

**Target Analysts:**
- Gartner: "Application Security Testing" category
- Forrester: "Software Composition Analysis" wave
- RedMonk: Developer tools coverage

**Target Media:**
- TechCrunch: "Y Combinator-backed CodeRisk launches pre-commit risk checker"
- The New Stack: "How CodeRisk Uses Graph Databases to Prevent Production Incidents"
- InfoQ: "Agentic Code Analysis: LLMs Meet Graph Databases"

---

## Related Documents

**Product & Business:**
- [vision_and_mission.md](vision_and_mission.md) - Product vision and differentiation
- [user_personas.md](user_personas.md) - Target users (Ben, Clara, Alex)
- [pricing_strategy.md](pricing_strategy.md) - Pricing tiers and rationale
- [success_metrics.md](success_metrics.md) - OKRs and tracking

**Technical:**
- [spec.md](../spec.md) - Complete requirements specification (Section 1.4)
- [01-architecture/agentic_design.md](../01-architecture/agentic_design.md) - Technical differentiation

---

**Last Updated:** October 10, 2025
**Next Review:** January 2026 (quarterly competitive scan)
**Recent Changes:**
- Added GitHub mining bootstrap strategy and updated positioning (ARC Database first-mover)
- Added vFunction (Application Modernization) to competitive analysis - non-competitive, different market (architect vs developer, multi-year vs daily)
