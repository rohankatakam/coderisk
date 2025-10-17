# CodeRisk System Overview: A Layman's Guide

**Purpose:** Explain CodeRisk's architecture and agentic search in accessible, non-technical terms
**Last Updated:** October 5, 2025
**Audience:** Non-technical stakeholders, new team members, product managers

> **Cross-reference:** For technical details, see [agentic_design.md](agentic_design.md), [graph_ontology.md](graph_ontology.md), and [developer_experience.md](../00-product/developer_experience.md)

---

## The Big Picture

CodeRisk is like having an experienced senior engineer automatically review your code changes before you commit them. It's designed specifically for the modern development workflow where AI tools (like Claude, Cursor, GitHub Copilot) help you write code incredibly fast—but sometimes too fast for you to fully understand all the implications.

---

## The Problem It Solves

When you write code by hand, you write maybe 50-100 lines per hour and naturally understand what you're building. But with AI assistants, you can generate 500-1000 lines in the same time. The risk? You might commit code you don't fully understand, which could break things or introduce security issues.

CodeRisk acts as your safety net—it automatically analyzes code changes and warns you: "Hey, this looks risky because..." before you commit.

---

## How It Works: The Two-Phase System

### Phase 1: Quick Health Check (200ms)

When you try to commit code, CodeRisk first runs a super-fast baseline check:

**1. Structural Coupling**: How many other files depend on what you changed?
   *Think: "If this breaks, how many things break with it?"*

**2. Temporal Co-Change**: Do certain files always change together?
   *Think: "These files are BFFs—when you change one, you almost always change the other"*

**3. Test Coverage**: Are there tests for this code?

If everything looks good (low coupling, good tests, no weird patterns), it says "✅ Safe to commit" in under 200ms and you're done.

### Phase 2: Deep Investigation (3-5 seconds, only when needed)

If Phase 1 spots potential issues, it escalates to an AI-powered investigation.

This is where **"agentic search"** happens—an LLM (Large Language Model) acts like a detective, deciding what to investigate next based on what it finds.

---

## What is Agentic Search?

Traditional approaches would try to calculate everything about your entire codebase upfront (imagine analyzing every possible relationship between 10,000 files—that's 100 million combinations!). This is:
- **Impossibly slow** (would take days)
- **Wasteful** (99% of it is irrelevant to your change)
- **Expensive** (massive computation and storage)

**Agentic search** is different:

### How It Works

**1. Start small**: Load only the immediate neighbors of your changed file (maybe 15 files that import it or call it)

**2. LLM decides**: The AI looks at the evidence and decides:
   - "Hmm, high coupling detected. Let me check ownership history..."
   - "I see temporal coupling, let me look for past incidents..."

**3. Selective expansion**: It only calculates what it needs, when it needs it. Maximum 3 "hops" to prevent runaway analysis.

**4. Synthesis**: After gathering evidence (usually 2-3 investigation steps), the LLM puts it all together and explains:
   > "This is MEDIUM risk because: coupling is high (12 files depend on this), the code owner just changed 14 days ago, and there was a similar incident 3 weeks ago"

### Why It's Efficient

- Instead of 100 million calculations, you do ~50
- Cost: $0.004 per check instead of $3
- Time: 3-5 seconds instead of hours
- **10 million times faster** than exhaustive analysis

---

## The Data Layer

### What's Stored in the Graph Database

CodeRisk uses a graph database (Amazon Neptune) to store **persistent facts**:

#### Layer 1: Code Structure
- Files, functions, classes
- Who imports what, who calls what
- Source: Tree-sitter (parses your actual code)

#### Layer 2: Git History
- Commits, developers, pull requests
- Co-change patterns: "File A and File B changed together in 15 of the last 20 commits"
- Source: Git logs from last 90 days

#### Layer 3: Incident History
- Production failures, bugs, outages
- Which commits caused which incidents
- Source: GitHub Issues, Jira, Sentry (manually linked for accuracy)

### What's NOT Stored (Calculated On-Demand)

- Risk scores (computed fresh each time based on current context)
- Blast radius (how many files affected—calculated when needed)
- Investigation traces (ephemeral, cached for 15 minutes in Redis)

**Analogy**: The graph is like Wikipedia (stores facts). The LLM is like a researcher who reads those facts and answers "Is this change risky?"

---

## State Management & Caching *(12-factor: Factor 5)*

### Working Memory (During Investigation)

The agent loads a small "working context" into RAM:
- Your changed files
- 1-hop neighbors (~15 files)
- Metrics as they're calculated
- Evidence chain (what the LLM has discovered)

This stays small (under 2,000 tokens) so the LLM can reason quickly.

### Redis Cache (15-Minute TTL)

Calculated metrics are cached:
- `coupling:auth.py → {"count": 12, "files": [...]}`
- `incidents:auth.py → [{"id": 123, "similarity": 0.82}]`

If you check the same file twice in 15 minutes, it's instant (5ms instead of 200ms).

### Why This Matters

1. **Speed**: 85-90% cache hit rate means most checks are blazing fast
2. **Cost**: Avoid recalculating the same thing repeatedly
3. **Freshness**: 15-minute TTL ensures data isn't stale

---

## Real Example: Agentic Search in Action

### Scenario: Developer Changes `payment_processor.py`

```bash
git add src/payments/payment_processor.py
git commit -m "Add retry logic for Stripe API calls"
```

### Phase 1: Baseline Assessment (180ms)

CodeRisk runs 3 parallel graph queries:

**Query 1: Structural Coupling (50ms)**
```
Result: 18 files (imports or is imported by 18 other files)
```

**Query 2: Temporal Co-Change (20ms)**
```
Result:
- database/transactions.py → 0.85 (changed together 85% of the time)
- api/checkout.py → 0.72
- models/order.py → 0.68
```

**Query 3: Test Coverage (50ms)**
```
Result: test_loc: 150, source_loc: 450 → ratio: 0.33 (33% coverage)
```

**Baseline Decision:**
```
✅ Coupling: 18 (>10) → HIGH
✅ Co-change: 0.85 (>0.7) → HIGH
⚠️ Test coverage: 0.33 (borderline)

Decision: Escalate to Phase 2 (LLM investigation)
```

---

### Phase 2: LLM-Guided Investigation (3.2s total)

#### Step 1: Initialize Context (100ms)

Load 1-hop neighbors into working memory (15 files).

#### Hop 1: LLM Decision (1.0s LLM call + 100ms query)

**LLM sees:**
```
Changed file: payment_processor.py
- Coupling: 18 files depend on this (HIGH)
- Co-change: 85% with transactions.py (HIGH)
- Test coverage: 33%
```

**LLM decides:**
```json
{
  "action": "CALCULATE_METRIC",
  "metric": "ownership_churn",
  "reasoning": "High coupling + high co-change suggests checking if code owner is stable. Ownership transitions increase risk when changing critical payment code."
}
```

**Result:**
```json
{
  "current_owner": "alice@company.com",
  "previous_owner": "bob@company.com",
  "transition_date": "2025-09-20",
  "days_since_transition": 14
}
```

**Evidence Added**: "Code owner changed from Bob to Alice 14 days ago"

---

#### Hop 2: LLM Decision (1.0s LLM call + 150ms search)

**LLM sees:**
```
Evidence so far:
1. Coupling: 18 files (HIGH)
2. Co-change: 85% with transactions.py (HIGH)
3. Ownership: Changed from Bob to Alice 14 days ago (RECENT TRANSITION)
```

**LLM decides:**
```json
{
  "action": "CALCULATE_METRIC",
  "metric": "incident_similarity",
  "reasoning": "Recent ownership change + high coupling is a red flag. Let's check if this file has incident history. Commit message mentions 'retry logic' and 'Stripe API' - search for similar past failures."
}
```

**Result (PostgreSQL full-text search, see [ADR-003](decisions/003-postgresql-fulltext-search.md)):**
```json
{
  "incident_id": "INC-892",
  "title": "Stripe API timeout causing checkout failures",
  "severity": "HIGH",
  "created_at": "2025-09-10",
  "similarity_score": 0.89,
  "description": "Payment processor failed to handle Stripe timeout, cascaded to transactions.py, caused 500 errors in checkout flow"
}
```

**Evidence Added**: "Similar to Incident #892 (89% match): Stripe timeout that affected transactions.py"

---

#### Hop 3: LLM Decision (0.8s LLM call)

**LLM sees complete evidence:**
```
1. Coupling: 18 files (HIGH)
2. Co-change: 85% with transactions.py (HIGH)
3. Ownership: Bob → Alice 14 days ago (RECENT)
4. Incident: INC-892 "Stripe timeout" (89% similar, affected transactions.py)

Pattern detected: You're adding retry logic to the EXACT code that caused INC-892, which cascaded through transactions.py due to tight coupling. The new owner (Alice) may not be familiar with this incident history.
```

**LLM decides:**
```json
{
  "action": "FINALIZE",
  "reasoning": "Strong evidence chain: high coupling + recent ownership change + similar past incident + temporal coupling with the same file (transactions.py) that was affected in INC-892. Adding retry logic is good, but needs integration tests covering the cascade scenario."
}
```

---

#### Step 4: Synthesis (0.2s LLM call)

**Final LLM output:**
```json
{
  "risk_level": "HIGH",
  "confidence": 0.87,
  "key_factors": [
    "18 files depend on this code - high blast radius if retry logic fails",
    "85% co-change with transactions.py - changes will likely ripple there too",
    "Code owner changed 14 days ago - new owner may not know INC-892 history",
    "89% similarity to INC-892 (Stripe timeout cascade through transactions.py)",
    "Low test coverage (33%) for critical payment code"
  ],
  "recommendations": [
    {
      "priority": 1,
      "action": "Add integration tests covering Stripe timeout + retry + transaction rollback scenario (the exact failure from INC-892)",
      "estimated_time": "45 minutes"
    },
    {
      "priority": 2,
      "action": "Have Alice review INC-892 post-mortem before merging",
      "estimated_time": "15 minutes"
    },
    {
      "priority": 3,
      "action": "Consider circuit breaker pattern to prevent cascade to transactions.py",
      "estimated_time": "2 hours (future improvement)"
    }
  ]
}
```

---

### User Sees This Output

```bash
git commit -m "Add retry logic for Stripe API calls"

🔍 CodeRisk: Analyzing... (3.4s)

🔴 HIGH risk detected (confidence: 87%)

Investigation Summary:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Hop 1: payment_processor.py
  ✓ Calculated coupling: 18 files (HIGH)
  → Decision: Check ownership stability

Hop 2: Ownership analysis
  ⚠ Code owner: Bob → Alice (14 days ago)
  → Decision: Search for similar incidents

Hop 3: Incident search
  🚨 Found INC-892 (89% match): "Stripe timeout cascade"
     - Affected transactions.py (same 85% co-change pattern)
     - Occurred 25 days ago
  → Decision: FINALIZE (strong evidence)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Key Evidence:
1. 18 files depend on payment_processor.py (blast radius)
2. Changes with transactions.py 85% of the time
3. New code owner (14 days) unfamiliar with incident history
4. Similar to INC-892: Stripe timeout → transaction cascade
5. Only 33% test coverage for critical payment code

Recommendations:
1. [45min] Add integration tests: Stripe timeout + retry + rollback
   (Cover exact INC-892 scenario)
2. [15min] Alice: Review INC-892 post-mortem before merging
3. [Future] Consider circuit breaker to prevent transactions.py cascade

❌ Commit blocked.

Options:
  • Fix issues: crisk fix-with-ai --tests
  • Override: git commit --no-verify (will be logged)
  • Details: crisk check --explain
```

---

## Why This Was Efficient

### What We DIDN'T Do:
❌ Analyze all 10,000 files in the repo (would take hours)
❌ Calculate risk scores for every file (wasteful)
❌ Pre-compute all relationships (100 million combinations)
❌ Load entire graph into LLM (exceeds context window)

### What We DID Do:
✅ Loaded only 18 relevant neighbors (1-hop from changed file)
✅ Asked LLM to decide what to investigate (2 targeted calculations)
✅ Stopped after 3 hops (sufficient evidence)
✅ Used small context (< 2K tokens per LLM call)

### Performance Breakdown:

| Step | Time | Cost |
|------|------|------|
| Phase 1 baseline | 180ms | ~$0 (cached 90% of time) |
| Load 1-hop context | 100ms | ~$0 (graph query) |
| Hop 1 (ownership) | 1.1s | ~$0.001 |
| Hop 2 (incidents) | 1.15s | ~$0.002 |
| Hop 3 (synthesis) | 1.0s | ~$0.001 |
| **Total** | **3.4s** | **~$0.004** |

Compare to exhaustive approach:
- **Time**: 3.4s vs hours
- **Cost**: $0.004 vs $3+
- **Accuracy**: HIGH (caught real risk from INC-892)

---

## Key Insights

### The LLM as Detective

The LLM acted like a detective:
1. Started with obvious clues (coupling)
2. Followed the trail (ownership change)
3. Connected the dots (similar incident)
4. Delivered actionable advice

All in under 4 seconds.

### The 99% Insight *(12-factor: Factor 3)*

**Most relationships are irrelevant to any given change.** The LLM intelligently navigates to what actually matters.

This is why agentic search is 10 million times faster than exhaustive approaches—it focuses on the 1% that matters.

### Context Window Economics

**Full graph approach:**
- 10,000 files × 100 tokens each = 1,000,000 tokens
- Exceeds Claude's 200,000 token limit by 5x
- Cost: $3/request (if it fit)

**Agentic approach:**
- 1-hop neighbors: 15 files × 100 tokens = 1,500 tokens
- Evidence chain: ~500 tokens
- Total: ~2,000 tokens
- Cost: $0.004/request
- **750x cheaper**

---

## Storage Architecture *(12-factor: Factor 5)*

### Data Layer Separation

| Layer | Storage | What | TTL | Example |
|-------|---------|------|-----|---------|
| **Persistent Graph** | Neptune | Code structure, git history, incidents | Indefinite | `(:File)-[:IMPORTS]->(:File)` |
| **Ephemeral Cache** | Redis | Metric results, investigation context | 15 min | `coupling:auth.py → {"count": 12}` |
| **Structured Data** | Postgres | Incidents, metric validation, user overrides, FP rates | Indefinite | `metrics.fp_rate WHERE name='coupling'` |
| **Full-Text Search** | Postgres | Incident descriptions (`tsvector` + GIN index) | Indefinite | `"auth timeout" → [Incident#123]` (see [ADR-003](decisions/003-postgresql-fulltext-search.md)) |

### Node Counts (Repository Examples)

- **Small repo** (~500 files): 8K nodes, 50K edges, ~200MB graph
- **Medium repo** (~5K files): 80K nodes, 600K edges, ~2GB graph
- **Large repo** (~50K files): 800K nodes, 6M edges, ~20GB graph

### Query Performance Targets

- **1-hop structural query** (coupling): <50ms
- **Co-change lookup**: <20ms (edge property read)
- **Incident similarity** (PostgreSQL FTS): <50ms (with GIN index, see [ADR-003](decisions/003-postgresql-fulltext-search.md))
- **Total Tier 1 metrics**: <200ms (3 queries in parallel)

---

## Design Principles

### Why Two-Phase Architecture? *(12-factor: Factor 8)*

**Single-phase (always LLM):**
- ❌ Slow: 3-5s for every check (even trivial ones)
- ❌ Expensive: ~$0.05 per check × 100 checks/day = $5/day
- ❌ Overkill: 80% of changes are low-risk, don't need deep analysis

**Two-phase (baseline + selective LLM):**
- ✅ Fast: 200ms for 80% of checks (LOW risk, no LLM)
- ✅ Cheap: $0.01 per Phase 1 check × 100 = $1/day
- ✅ Smart: Deep investigation only when needed

### Why No Pre-Computed Risk Scores?

**Legacy approach:**
- ❌ Expensive: $15 `crisk init` per repo
- ❌ Stale: Pre-computed scores don't reflect latest context
- ❌ High FP: Complex models (15-25% FP rate)

**On-demand calculation:**
- ✅ Fresh: Always uses latest graph data
- ✅ Contextual: LLM combines multiple signals
- ✅ Low FP: Simple metrics (1-5% FP rate), LLM reasoning reduces noise

### Why Max 3 Hops? *(12-factor: Factor 10)*

**Without limits:**
- Risk: Exponential explosion (10 neighbors/hop → 10³ = 1000 nodes)
- Cost: Unbounded LLM calls ($0.50+)
- Latency: Could take 30+ seconds

**With 3-hop limit:**
- ✅ Controlled: ~20-50 nodes explored
- ✅ Predictable: $0.03-0.05 per investigation
- ✅ Fast: 3-5s typical

---

## Summary

CodeRisk uses **agentic search** to provide intelligent code risk assessment:

1. **Fast baseline** (200ms): Simple metrics catch 80% of cases
2. **LLM investigation** (3-5s): Deep analysis only when needed
3. **Selective calculation**: Only compute what matters
4. **Evidence-based**: Combine multiple low-FP signals
5. **Actionable advice**: Tell developers exactly what to fix

**Result**: 10 million times faster than exhaustive approaches, with <3% false positive rate and sub-$0.01 cost per check.

---

**See also:**
- [agentic_design.md](agentic_design.md) - Technical details of agent investigation *(12-factor: Factor 8, 10)*
- [graph_ontology.md](graph_ontology.md) - Graph schema and data sources *(12-factor: Factor 3)*
- [developer_experience.md](../00-product/developer_experience.md) - User-facing UX and workflows
- [risk_assessment_methodology.md](risk_assessment_methodology.md) - Metric formulas and validation *(12-factor: Factor 7)*
