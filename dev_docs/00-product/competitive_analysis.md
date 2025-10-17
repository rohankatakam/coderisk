# Competitive Analysis (MVP Focus)

**Last Updated:** October 17, 2025
**Status:** Active - Local-First Positioning
**Target Market:** Solo developers + small teams using AI coding assistants

> **📘 Strategic Simplification:** Updated to focus on local-first MVP positioning. Complex trust infrastructure and ARC database positioning archived to [99-archive/00-product-future-vision](../99-archive/00-product-future-vision/) for v2-v4.

---

## Executive Summary

**CodeRisk MVP Positioning:**
> **"Local-first pre-commit risk scanner for AI-generated code"**

**Category:** Pre-commit architectural safety check (NEW workflow moment)

**Key Differentiators:**
1. **Timing:** Pre-commit (before code becomes public) vs post-commit (PR review, CI/CD)
2. **Architecture:** Local-first (Docker + local Neo4j) vs cloud-based
3. **Economics:** BYOK model (~$1-2/month) vs all-inclusive pricing ($30-150/month)
4. **Focus:** Architectural risks for AI-generated code vs general code quality

**Primary Competitors:**
- **Greptile** - Cloud PR review tool (later timing, higher cost)
- **SonarQube** - Rule-based static analysis (high false positives)
- **Codescene** - Health dashboard for managers (batch, not real-time)

**Competitive Advantages:**
- ✅ **Earliest intervention** - Pre-commit, not PR review
- ✅ **Local-first** - Fast, private, no cloud costs
- ✅ **Low cost** - Free BYOK (~$1-2/month LLM costs)
- ✅ **AI-focused** - Designed for AI coding assistant users
- ✅ **Low false positives** - <5% target vs 10-20% industry

---

## Market Positioning Map

### Timing in Developer Workflow

```
┌─────────────────────────────────────────────────────────┐
│                    Timing in Workflow                    │
├──────────────┬──────────────┬──────────────┬────────────┤
│  Pre-Commit  │   Pre-PR     │  PR Review   │ Post-Merge │
│  (PRIVATE)   │   (PRIVATE)  │  (PUBLIC)    │ (DEPLOYED) │
├──────────────┼──────────────┼──────────────┼────────────┤
│  CodeRisk    │              │  Greptile    │  Codescene │
│  (2-5s)      │              │  (30s-2min)  │  (hours)   │
│  LOCAL       │              │  SonarQube   │  Datadog   │
│              │              │  (minutes)   │  (passive) │
└──────────────┴──────────────┴──────────────┴────────────┘

              ↑ CodeRisk owns this moment
              └─ BEFORE developer commits publicly
```

**CodeRisk's Unique Position:**
- **Earliest intervention point** - Before code becomes public
- **Private feedback** - No embarrassment, no sunk cost fallacy
- **Local execution** - Fast, private, no network latency
- **Rapid iteration** - Fix locally, recheck in 5 seconds

---

## Primary Competitors

### 1. Greptile (Cloud PR Co-pilot)

**Company Profile:**
- **Founded:** 2023
- **Funding:** Seed stage (~$2M)
- **Focus:** Conversational AI-powered PR review
- **Architecture:** Cloud-based, vector search + RAG
- **Pricing:** $30-100/user/month (estimated)

**Product Positioning:**
> "Chat with your codebase during PR review"

**Feature Comparison:**

| Feature | CodeRisk (MVP) | Greptile |
|---------|----------------|----------|
| **Timing** | Pre-commit (private) | PR review (public) |
| **Architecture** | Local (Docker + Neo4j) | Cloud (vector DB) |
| **Latency** | 2-5 seconds | 30 seconds - 2 minutes |
| **Use Case** | "Is this AI code safe?" | "How does X work?" |
| **Workflow** | Binary check (pass/fail) | Conversational exploration |
| **Setup** | `brew install` → Done | OAuth + repo indexing (5 min) |
| **Privacy** | 100% local (except LLM call) | Code sent to cloud |
| **Pricing** | Free + BYOK ($1-2/mo LLM) | $30-100/user/month (all-in) |
| **LLM Costs** | User pays directly (transparent) | Included (marked up 2-3x) |
| **Infrastructure** | User's machine (Docker) | Greptile's cloud |

**Strengths:**
- ✅ Strong conversational UX (ChatGPT for code)
- ✅ Good for exploratory questions during PR review
- ✅ Integrates with GitHub PR interface
- ✅ Mature cloud infrastructure

**Weaknesses:**
- ❌ Too late in workflow (developer already committed)
- ❌ Cloud-based (privacy concerns, network latency)
- ❌ Expensive (includes LLM with markup)
- ❌ Requires conversation (not instant feedback)

**When Greptile Wins:**
- PR reviewer needs deep code exploration
- Team comfortable with cloud-based tools
- Budget allows $30-100/user/month

**When CodeRisk Wins:**
- Developer needs instant pre-commit check
- Privacy-conscious teams (local-first)
- Budget-conscious ($1-2/month vs $30-100/month)
- Using AI coding assistants (Claude Code, Cursor)

**Positioning:**
> **"Use CodeRisk *before* you commit (private), use Greptile *during* PR review (public)"**
>
> Not competitive—teams can use both. CodeRisk for pre-commit safety, Greptile for PR exploration.

---

### 2. SonarQube (Static Analysis)

**Company Profile:**
- **Founded:** 2008
- **Market Leader:** 7M+ developers, 400K+ organizations
- **Focus:** Code quality and security scanning
- **Architecture:** Rule-based static analysis
- **Pricing:** Free (Community) → $10-150/user/month (Enterprise)

**Product Positioning:**
> "Clean Code for developers, Safe Code for organizations"

**Feature Comparison:**

| Feature | CodeRisk (MVP) | SonarQube |
|---------|----------------|-----------|
| **Intelligence** | LLM-guided + graph analysis | Rule-based patterns |
| **False Positive Rate** | <5% (target) | 10-20% (industry standard) |
| **Focus** | Architectural risks, coupling | Security, bugs, code smells |
| **Context** | Temporal coupling + git history | Syntax + known vulnerabilities |
| **Evolution** | Learns from codebase | Fixed rulesets (manual updates) |
| **Timing** | Pre-commit (local) | CI/CD (cloud) |
| **Setup** | `brew install` (1 min) | CI/CD integration (hours) |
| **Privacy** | 100% local | Code sent to SonarCloud OR self-hosted |

**Strengths:**
- ✅ Industry standard (7M+ users)
- ✅ Comprehensive security rules (OWASP, CWE)
- ✅ Free tier (Community Edition)
- ✅ Mature product (16+ years)

**Weaknesses:**
- ❌ High false positive rate (10-20%)
- ❌ Rule-based (misses architectural coupling)
- ❌ Alert fatigue (developers ignore warnings)
- ❌ No temporal coupling analysis
- ❌ Not optimized for AI-generated code

**When SonarQube Wins:**
- Team needs security vulnerability scanning
- Compliance requires OWASP coverage
- Already invested in SonarQube infrastructure

**When CodeRisk Wins:**
- Team frustrated with SonarQube's false positives
- Need architectural risk analysis (not just security)
- Want local-first tool (privacy, speed)
- Using AI coding assistants

**Positioning:**
> **"SonarQube finds *security bugs*, CodeRisk finds *architectural risks* in AI code"**
>
> Complementary—run both. SonarQube for security, CodeRisk for architecture.

---

### 3. Codescene (Health Dashboard)

**Company Profile:**
- **Founded:** 2015
- **Market Position:** 500+ enterprise teams
- **Focus:** Codebase health monitoring for managers
- **Architecture:** Cloud dashboard, batch analysis
- **Pricing:** $50-150/user/month

**Product Positioning:**
> "X-ray for your codebase: Find hotspots and technical debt"

**Feature Comparison:**

| Feature | CodeRisk (MVP) | Codescene |
|---------|----------------|-----------|
| **Timing** | Pre-commit (real-time) | Post-merge (batch, daily) |
| **Target User** | Individual developer | Tech lead / manager |
| **Workflow** | Developer inner loop (CLI) | Management dashboard (web) |
| **Latency** | 2-5 seconds | Hours (daily batch analysis) |
| **Scope** | Specific change impact | Codebase-wide trends |
| **Intelligence** | LLM + graph analysis | Statistical models |
| **Actionability** | "Fix before commit" | "Prioritize refactoring" |
| **Setup** | `brew install` (1 min) | Historical data ingestion (days) |

**Strengths:**
- ✅ Mature product (9+ years)
- ✅ Strong visualization (hotspot maps)
- ✅ Good for technical debt prioritization
- ✅ Predictive analytics

**Weaknesses:**
- ❌ Not real-time (batch processing)
- ❌ Optimized for managers, not developers
- ❌ Dashboard-centric (not CLI-friendly)
- ❌ Expensive ($50-150/user/month)

**When Codescene Wins:**
- Management needs codebase health dashboard
- Team wants to prioritize technical debt
- Large enterprise with budget for premium tools

**When CodeRisk Wins:**
- Developer needs instant feedback before committing
- Team wants developer-first tool (not manager dashboard)
- Small team or solo developer
- Using AI coding assistants

**Positioning:**
> **"Codescene is for your *manager*, CodeRisk is for *you*"**
>
> Codescene tells you where debt is (dashboard). CodeRisk prevents new debt (pre-commit).

---

## Indirect Competitors

### GitHub Advanced Security (GHAS)

**Focus:** Security scanning (not architectural analysis)
**Overlap:** CI/CD code scanning
**Differentiation:** CodeRisk focuses on architectural coupling, not security vulnerabilities

**When GHAS Wins:**
- Already using GitHub Enterprise
- Need Dependabot, secret scanning

**When CodeRisk Wins:**
- Need architectural coupling analysis
- Want pre-commit feedback (GHAS is CI/CD only)
- Solo developer (GHAS requires Enterprise)

---

## Competitive Advantages Matrix

| Capability | CodeRisk | Greptile | Codescene | SonarQube |
|------------|----------|----------|-----------|-----------|
| **Pre-Commit Timing** | ✅ Core | ❌ No | ❌ No | ⚠️ IDE only |
| **Local-First** | ✅ Core | ❌ Cloud | ❌ Cloud | ⚠️ Optional |
| **Low Cost (<$5/month)** | ✅ BYOK | ❌ $30-100 | ❌ $50-150 | ⚠️ Free tier |
| **AI Code Focus** | ✅ Core | ⚠️ General | ❌ No | ❌ No |
| **Temporal Coupling** | ✅ Core | ❌ No | ✅ Yes | ❌ No |
| **Low FP Rate (<5%)** | ✅ Target | ⚠️ Varies | ⚠️ ~5-10% | ❌ ~10-20% |
| **Real-Time (<5s)** | ✅ Core | ⚠️ 30s-2min | ❌ Batch | ⚠️ Varies |
| **CLI-First** | ✅ Core | ❌ Web | ❌ Dashboard | ⚠️ CI/CD |
| **Privacy (100% Local)** | ✅ Core | ❌ Cloud | ❌ Cloud | ⚠️ Optional |
| **Graph Database** | ✅ Local Neo4j | ⚠️ Vector DB | ❌ SQL | ❌ Files |
| **LLM Integration** | ✅ BYOK | ✅ Included | ❌ No | ❌ No |

**Legend:**
- ✅ Strong capability
- ⚠️ Partial capability
- ❌ Not supported

---

## Pricing Comparison

### Solo Developer Scenario (100 checks/month)

| Product | Monthly Cost | What's Included |
|---------|-------------|-----------------|
| **CodeRisk (MVP)** | **$1-2** | Free tool + BYOK LLM (~$0.01/check) |
| Greptile | $30-70 | All-inclusive (LLM marked up 2-3x) |
| Codescene | $50-100 | Cloud dashboard |
| SonarQube | $0-10 | Community free, Developer $10/month |

**CodeRisk Savings:** 80-98% vs competitors

### Small Team Scenario (5 developers, 500 checks/month)

| Product | Monthly Cost | What's Included |
|---------|-------------|-----------------|
| **CodeRisk (MVP)** | **$5-10** | Free tool + BYOK LLM (~$5 total) |
| Greptile | $150-350 | 5 users × $30-70/user |
| Codescene | $250-500 | 5 users × $50-100/user |
| SonarQube | $50-750 | $10-150/user (depends on tier) |

**CodeRisk Savings:** 95-98% vs competitors

### Cost Breakdown (Transparent)

**CodeRisk Total Cost:**
```
Tool: $0 (free, open source)
Infrastructure: $0 (runs locally on user's Docker)
LLM API: ~$0.01/check × 100 checks = $1/month (user's OpenAI/Anthropic)

Total: $1-2/month
```

**Greptile Total Cost:**
```
Subscription: $30-70/month (includes LLM with 2-3x markup)
Infrastructure: Included (cloud-hosted)

Total: $30-70/month
```

**Why CodeRisk is 95%+ cheaper:**
- User runs locally (no cloud infrastructure costs for us)
- User provides API key (no LLM markup)
- We don't pay for compute (user's Docker container)

---

## Positioning Map

```
                    High Intelligence (LLM/AI)
                              ▲
                              │
                         CodeRisk
                         (Pre-commit)
                      LOCAL + FAST
                              │
                         Greptile
                         (PR Review)
                      CLOUD + CONVERSATIONAL
                              │
    Fast ◄────────────────────┼────────────────────► Slow
  (Real-time)                 │                  (Batch)
   LOCAL                      │                  CLOUD
                              │
                         SonarQube              Codescene
                         (CI/CD)                (Dashboard)
                      RULES-BASED           STATISTICAL
                              │
                              ▼
                    Low Intelligence (Rules/Stats)
```

**CodeRisk Positioning:**
- **Top-Left Quadrant**: High intelligence + Fast + Local (UNIQUE)
- **Workflow Moat**: Earliest intervention point (pre-commit)
- **Economic Moat**: BYOK model (95%+ cheaper)
- **Privacy Moat**: 100% local (except LLM API call)

---

## Competitive Moats (MVP)

### 1. Workflow Timing (Pre-Commit)

**Thesis:** Once developers adopt pre-commit checks, they won't switch to post-commit tools

**Evidence:**
- Pre-commit = private feedback (no embarrassment)
- Post-commit = public commitment (sunk cost fallacy)
- Developers prefer catching issues early

**Defensibility:**
- First-mover advantage in "pre-commit for AI code" category
- Competitors anchored to later workflow stages
- Habit formation (muscle memory)

### 2. Local-First Architecture

**Thesis:** Privacy-conscious developers prefer local tools over cloud SaaS

**Evidence:**
- Code never leaves developer's machine (except LLM API)
- Fast (no network latency)
- Works offline (no internet required for graph analysis)

**Defensibility:**
- Competitors built for cloud (can't easily pivot to local)
- Network effects favor cloud (they won't cannibalize)
- We own privacy-first positioning

### 3. BYOK Economics

**Thesis:** Developers prefer transparent costs over all-inclusive pricing with markup

**Evidence:**
- Reddit discussions: "Greptile's pricing is opaque"
- Developers want to see exact LLM costs
- BYOK model proven by Cursor, other AI dev tools

**Defensibility:**
- Competitors rely on LLM markup for margins
- Can't pivot to BYOK without losing revenue
- We have 95%+ cost advantage

### 4. AI Coding Assistant Focus

**Thesis:** AI-generated code has unique risks (coupling, patterns) that general tools miss

**Evidence:**
- AI generates code 5-10x faster (can't manually review all)
- AI doesn't understand codebase architecture
- Developers using Claude Code, Cursor need safety check

**Defensibility:**
- Competitors optimize for general code quality
- We optimize for AI-generated code specifically
- Pattern library focused on AI coding risks

---

## Win/Loss Scenarios

### When CodeRisk Wins

**Scenario 1: Solo Developer Using AI**
- **Profile:** Solo dev or small team (2-5 people)
- **Trigger:** Using Claude Code, Cursor, Copilot daily
- **Pain Point:** Uncertain if AI-generated code is safe
- **Why CodeRisk:** Free, local, instant pre-commit check

**Scenario 2: Privacy-Conscious Team**
- **Profile:** Startup handling sensitive data (health, finance)
- **Trigger:** Can't send code to cloud (compliance, privacy)
- **Pain Point:** Cloud tools like Greptile not allowed
- **Why CodeRisk:** 100% local (except LLM API call)

**Scenario 3: Budget-Conscious Startup**
- **Profile:** Bootstrapped startup, 5-10 developers
- **Trigger:** Sticker shock from Greptile ($150-700/month for team)
- **Pain Point:** Can't afford $30-100/user/month tools
- **Why CodeRisk:** $5-10/month total (BYOK model)

### When CodeRisk Loses

**Scenario 1: Enterprise Compliance**
- **Lost To:** SonarQube
- **Reason:** Must satisfy OWASP/CWE security compliance
- **Learning:** Integrate with SonarQube (complementary)

**Scenario 2: Management Dashboards**
- **Lost To:** Codescene
- **Reason:** VP Engineering wants codebase health dashboard
- **Learning:** Not our MVP target (defer to v2)

**Scenario 3: Conversational UX Preference**
- **Lost To:** Greptile
- **Reason:** Team prefers chat-based exploration
- **Learning:** Not our target (we optimize for speed, not conversation)

---

## Competitive Response Strategies

### If Greptile Adds Pre-Commit Mode

**Threat Level:** Medium

**Response:**
1. **Timing Advantage:** Already own pre-commit habit (first-mover)
2. **Architecture:** We're local (faster), they're cloud (network latency)
3. **Economics:** BYOK is 95% cheaper (they can't match)
4. **Privacy:** We're 100% local (they need cloud for business model)

### If SonarQube Adds LLM Intelligence

**Threat Level:** Medium (large user base)

**Response:**
1. **False Positives:** Emphasize our <5% FP rate vs their 10-20% history
2. **Timing:** We're pre-commit (private), they're CI/CD (public)
3. **Cost:** BYOK vs their all-inclusive (we're 90%+ cheaper)
4. **Focus:** We're for AI code, they're for general security

### If Codescene Adds Real-Time Mode

**Threat Level:** Low (different target user)

**Response:**
1. **Target User:** We're for developers (CLI), they're for managers (dashboard)
2. **Cost:** $1-2/month vs $50-150/month (98% cheaper)
3. **Setup:** `brew install` vs historical data ingestion

### If GitHub Builds Native Pre-Commit Analysis

**Threat Level:** High (distribution advantage)

**Response:**
1. **Local-First:** We run locally (private, fast), GitHub would be cloud
2. **BYOK:** User controls LLM costs, GitHub would bundle/markup
3. **AI Focus:** We're optimized for AI code, GitHub would be general
4. **Privacy:** We're 100% local, GitHub requires cloud

---

## Market Positioning Statements

### Primary Message (Developers)
> **"CodeRisk: Pre-commit safety check for AI-generated code"**
>
> Run `crisk check` before committing—instant risk analysis in 5 seconds.
> Free, local-first, privacy-friendly. Just pay your own LLM costs (~$1-2/month).

### Against Greptile
> **"Use CodeRisk *before* you commit, Greptile *during* PR review"**
>
> CodeRisk: Pre-commit (private, local, $1-2/month)
> Greptile: PR review (public, cloud, $30-100/month)
>
> Not competitive—use both.

### Against SonarQube
> **"SonarQube finds security bugs. CodeRisk finds architectural risks in AI code."**
>
> SonarQube: Security focus, 10-20% false positives, CI/CD timing
> CodeRisk: Architecture focus, <5% false positives, pre-commit timing
>
> Complementary—run both.

### Against Codescene
> **"Codescene is for your manager. CodeRisk is for you."**
>
> Codescene: Health dashboard (batch, hours, $50-150/month)
> CodeRisk: Pre-commit check (real-time, 5 seconds, $1-2/month)

---

## MVP Go-to-Market Focus

### Target Audience (Prioritized)

**Tier 1: Solo Developers Using AI**
- 1-5 person teams
- Using Claude Code, Cursor, Copilot daily
- Budget-conscious ($1-2/month acceptable)
- Privacy-conscious (prefer local tools)

**Tier 2: Small Teams (2-10 people)**
- Startup or small company
- Team uses AI coding assistants
- Tech lead wants quality gates
- Budget constraints ($30-100/user/month too expensive)

**NOT Targeting for MVP:**
- ❌ Enterprise (50-500+ people) - v2
- ❌ Managers wanting dashboards - v2
- ❌ Security/compliance teams - different tool

### Positioning by Use Case

**Use Case 1: "Is my AI code safe to commit?"**
- **Target:** Solo developer using Claude Code daily
- **Message:** "Pre-commit safety check for AI code. 5 seconds. Free."
- **Competitor:** Manual review (slow) or blind commit (risky)

**Use Case 2: "My team generates too much AI code to review"**
- **Target:** Tech lead managing 2-10 AI users
- **Message:** "Automated pre-commit checks. Catch issues before PR."
- **Competitor:** Manual PR review (bottleneck)

**Use Case 3: "I can't afford $30-100/month per developer"**
- **Target:** Budget-conscious startup
- **Message:** "$1-2/month total. BYOK model. 95% cheaper."
- **Competitor:** Greptile, Codescene (too expensive)

---

## Related Documents

**Product:**
- [mvp_vision.md](mvp_vision.md) - MVP vision and scope
- [user_personas.md](user_personas.md) - Ben (solo dev), Clara (small team)
- [simplified_pricing.md](simplified_pricing.md) - Free BYOK model

**User Experience:**
- [developer_experience.md](developer_experience.md) - Local tool UX
- [developer_workflows.md](developer_workflows.md) - Git workflows

**Archived (Future):**
- [../99-archive/00-product-future-vision/](../99-archive/00-product-future-vision/) - Complex positioning (v2-v4)

---

**Last Updated:** October 17, 2025
**Next Review:** After MVP launch (Week 7-8), after 50+ user feedback
