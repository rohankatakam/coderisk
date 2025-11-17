# CodeRisk: Critical Value Analysis

## What It Does (Frank Description)

CodeRisk is a **code risk aggregator** that analyzes uncommitted code changes against a repository's historical incident, ownership, and coupling data. It surfaces this information through an MCP (Model Context Protocol) server that integrates with AI coding assistants like Claude Code and Cursor.

**Core Functionality:**
1. **Incident Linking**: Shows which code blocks have caused bugs in the past
2. **Coupling Analysis**: Identifies code that frequently changes together
3. **Ownership Tracking**: Shows who wrote/modified code and when
4. **Risk Scoring**: Combines all signals into a weighted risk score

**Technical Architecture:**
- **Ingestion** (`crisk init`): One-time 30-60 min process to analyze git history → PostgreSQL + Neo4j
- **Query** (MCP server): Real-time queries via Claude Code (~7 sec response)
- **Databases**: Hybrid PostgreSQL (incidents, coupling) + Neo4j (ownership, graph structure)

---

## The Workflow It Emulates

### Manual Workflow (Without CodeRisk)

When a developer wants to assess risk before committing code:

1. **Check Git History** (~2-3 min)
   ```bash
   git log --follow path/to/file.py
   git blame path/to/file.py
   ```
   - See who last touched it
   - See when it was modified

2. **Search for Related Bugs** (~5-10 min)
   - Open GitHub Issues tab
   - Search for file name: `mcpagent.py`
   - Read through issues manually
   - Click into each issue to understand context

3. **Check Method-Level History** (~3-5 min)
   ```bash
   git log -L :connect:path/to/file.py
   ```
   - Repeat for each method of concern
   - Manually correlate commits to issues

4. **Assess Team Knowledge** (~2-3 min)
   ```bash
   git shortlog -sn path/to/file.py
   ```
   - Check if original author still active
   - Identify bus factor risk

**Total Manual Time: 12-21 minutes per file**

**Reality Check:** Developers only do this for 10-20% of changes (high-risk PRs). Most commits skip this research entirely.

### CodeRisk Workflow

Ask Claude Code:
```
What are the risk factors for path/to/file.py?
```

**Response time: ~7 seconds**

Returns:
- All historical incidents linked to specific methods
- Coupling relationships (what else breaks when this changes)
- Ownership metadata (staleness, familiarity distribution)
- Risk score with weighted factors

---

## Time Saved vs Pain Saved

### Time Saved (Quantitative)

| Scenario | Manual | CodeRisk | Time Saved |
|----------|--------|----------|------------|
| Single file risk check | 12-21 min | 7 sec | **99% reduction** |
| Pre-commit review (5 files) | 60-105 min | 35 sec | **99% reduction** |
| Weekly usage (10 queries) | 2-3.5 hours | ~1 min | **98% reduction** |

**BUT**: This assumes developers were doing the manual research. **Most don't.**

**Realistic Time Saved:**
- Teams that actually do risk research: **~90 min/week saved**
- Teams that skip risk research: **0 min saved** (but risk reduction gained)

### Pain Saved (Qualitative)

**High Pain Eliminated:**
- ✅ **Context switching**: No more jumping between terminal → GitHub → IDE
- ✅ **Partial data**: Manual searches miss incidents that don't mention file name
- ✅ **Method-level granularity**: `git log -L` is extremely tedious for multiple methods
- ✅ **Inconsistent research**: Junior devs don't know what to look for; tool equalizes knowledge

**Medium Pain Eliminated:**
- ✅ **Onboarding friction**: New devs get instant historical context
- ✅ **PR anxiety**: "Is this safe to merge?" answered instantly

**Low Pain / No Change:**
- ❌ **Experienced devs on stable codebases**: They already know risky files
- ❌ **Solo developers**: No knowledge transfer problem to solve

---

## Practical Value: Honest Assessment

### What Makes This Valuable

1. **Method-Level Incident Linking** ⭐⭐⭐⭐⭐
   - **Novel**: Git/GitHub don't link issues to specific methods
   - **High signal**: Knowing `MCPAgent.connect` caused Issue #353 is actionable
   - **Cannot be done manually**: Would require reading every issue + commit message

2. **Coupling Data at Code-Block Level** ⭐⭐⭐⭐
   - **Novel**: Git shows co-changed files, not co-changed methods
   - **Actionable**: "This method always changes with these tests" helps predict impact
   - **Can be approximated manually**: Experienced devs know coupling patterns

3. **Risk Scoring** ⭐⭐⭐
   - **Marginally novel**: Simple weighted formula (incidents × 10 + coupling × 2 + staleness)
   - **Useful for triage**: Sort by risk score to prioritize reviews
   - **Could be gamed**: Doesn't understand code complexity, just historical patterns

4. **Ownership/Staleness** ⭐⭐
   - **Not novel**: `git blame` and `git log` provide this
   - **Convenience**: Aggregated in one place
   - **Low differentiation**: Competitors (CodeClimate, SonarQube) also show this

### What Doesn't Make This Valuable

1. **For Small Teams (<5 devs)** ❌
   - They already know which code is risky
   - Time saved doesn't justify ingestion cost (30-60 min setup)

2. **For Stable Codebases** ❌
   - Few incidents = low signal value
   - Coupling is stable = already understood

3. **No Actionable Recommendations** ❌
   - Tool says "high risk" but doesn't say **why** or **what to do**
   - Next step: Add LLM-generated recommendations based on incident patterns

4. **Friction in Current Form** ❌
   - **30-60 min ingestion** is a high barrier to trial
   - **Two databases** (PostgreSQL + Neo4j) is complex deployment
   - **Claude Code only** limits distribution

---

## Conversion to Paid Value

### Would Someone Pay For This?

**Target Customer:**
- **Engineering teams: 20-100 developers**
- **High incident rate: >50 bugs/quarter**
- **Fast-growing: Onboarding 2+ devs/month**
- **AI-generated code: >30% of commits from Copilot/Cursor**

**Pricing Model:**
- **Free Tier**: 10 queries/day (individual trial)
- **Team Tier**: $50/dev/month (unlimited queries, Slack integration, CI/CD webhooks)
- **Enterprise**: $$$$ (Custom ingestion, on-prem deployment, SSO)

### ROI Calculation (Team of 30 Devs)

**Cost:**
- $50/dev/month × 30 = **$1,500/month**

**Value (Optimistic):**
- **Time saved**: 90 min/week/dev × 30 devs = 45 hours/week
- **At $100/hour**: **$4,500/week** = **$18,000/month**
- **ROI**: 12x return

**Value (Pessimistic - What Actually Happens):**
- Only 10% of devs use it regularly (3 active users)
- They each save 90 min/week = 4.5 hours/week total
- **At $100/hour**: **$450/week** = **$1,800/month**
- **ROI**: 1.2x return (marginal)

**Reality Check:**
- **Adoption is the killer**. If <30% of team uses it, value collapses.
- **Incident linking is the only unique feature**. Without incidents, it's just faster `git blame`.

### Competitive Comparison

| Feature | CodeRisk | SonarQube | CodeClimate | GitHub Advanced Security |
|---------|----------|-----------|-------------|-------------------------|
| Static analysis | ❌ | ✅ | ✅ | ✅ |
| Incident linking | ✅ | ❌ | ❌ | ❌ |
| Method-level coupling | ✅ | ❌ | ❌ | ❌ |
| Ownership/staleness | ✅ | ✅ | ✅ | ✅ |
| AI integration | ✅ | ❌ | ❌ | ❌ |
| Price | TBD | $10-30/dev | $50/dev | $49/committer |

**Differentiation:** CodeRisk is the **only tool** that links historical incidents to specific methods. This is the wedge.

---

## Raw Novelty Score

### 1. Method-Level Incident Linking: **9/10 Novel**
- **Cannot be done with git commands**
- **Requires LLM + graph traversal**
- **Unique in market**

### 2. Code-Block Coupling: **7/10 Novel**
- Git shows file-level co-change
- This shows **method-level** co-change
- Useful but not revolutionary

### 3. Risk Scoring: **4/10 Novel**
- Simple weighted formula
- Many tools do similar scoring
- Not defensible IP

### 4. Ownership/Staleness: **2/10 Novel**
- Standard `git blame` output
- Presented nicely but not unique

**Overall Novelty: 6.5/10**

---

## Critical Take: Should This Exist?

### ✅ **YES, if:**
1. **Incident linking works reliably** - This is the killer feature. If 80%+ of incidents link correctly, the tool is worth building.
2. **You target the right customer** - Fast-growing teams with high incident rates will pay. Stable teams won't.
3. **You reduce friction** - 30-60 min ingestion kills trials. Make it <5 min or offer hosted service.
4. **You add actionable insights** - "High risk" → "High risk because last 3 incidents were null pointer errors in this method. Consider adding validation."

### ❌ **NO, if:**
1. **Incident linking is <50% accurate** - The entire value prop collapses. Users will stop trusting it.
2. **You can't reduce deployment complexity** - PostgreSQL + Neo4j is enterprise-grade friction for a dev tool.
3. **Adoption stays <20%** - Tools only used by 1-2 team members don't get renewed.

---

## Bottom Line

**CodeRisk is a 70th percentile tool solving an 80th percentile problem.**

The **problem** (understanding code risk) is real and painful for growing teams.
The **solution** (aggregating incidents + coupling + ownership) is genuinely useful.
The **execution** (PostgreSQL + Neo4j + 30min setup) has too much friction for mass adoption.

**To succeed:**
1. **Nail incident linking** - Make it >90% accurate with confidence scores
2. **Kill setup friction** - Hosted SaaS with <5 min onboarding
3. **Add recommendations** - Don't just say "risky", say "here's why + what to do"
4. **Integrate everywhere** - Claude Code is great, but also need: VS Code, Cursor, GitHub PR comments, Slack bots

**Current state:** Impressive technical demo. Not yet a must-have product.

**Path to must-have:** Focus on **incident linking accuracy** and **zero-setup deployment**. Everything else is table stakes.
