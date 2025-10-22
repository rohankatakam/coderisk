# Agentic Investigation Design

**Version:** 5.0 (MVP)
**Last Updated:** January 2025
**Purpose:** LLM-guided regression prevention through automated due diligence for local-first MVP
**Deployment:** Docker + local Neo4j, no cloud infrastructure

---

## Overview

CodeRisk uses an **LLM-guided agent** that automates the due diligence developers should perform manually before committing—but rarely do because it takes 10-15 minutes. The agent prevents regressions by checking blast radius, co-change patterns, ownership context, and incident history in <5 seconds.

**Core Principles:**
- **Regression prevention** - Automates due diligence that prevents regressions (blast radius, co-change, ownership)
- **Selective, not exhaustive** - Only analyze what the LLM needs, when it needs it
- **Evidence-driven** - Combine multiple low-FP metrics instead of single scores
- **Fast baseline** - 80% of checks complete in <500ms without LLM
- **Deep investigation** - 20% of risky changes get full LLM analysis
- **Local-first** - Runs entirely in Docker with local Neo4j graph database

---

## Why Agentic Investigation?

### The Graph Explosion Problem

**Naive approach (pre-compute everything):**
```
10,000-file repository
→ 100,000,000 possible file pairs
→ 500,000,000 metric calculations
→ 58 days of computation
→ INFEASIBLE
```

**Agentic approach (calculate on-demand):**
```
Developer changes 1 file (auth.py)
→ Load 15 neighboring files (1-hop)
→ Calculate ~50 metrics
→ 0.5 seconds
→ 10,000,000x faster
```

**Key insight:** 99% of relationships are irrelevant to any given change. The LLM navigates to the 1% that matters.

### Why Agentic Investigation Prevents Regressions

**Traditional Approach (Manual Due Diligence):**
- Developer manually runs: `git log --follow payment.py` → **5 minutes**
- Developer manually searches: "what imports payment.py?" → **5 minutes**
- Developer manually checks: `git blame` for ownership → **3 minutes**
- Developer manually reviews: past incidents related to payment → **2 minutes**
- **Total: 15+ minutes, often skipped entirely**

**CodeRisk Approach (Automated Due Diligence):**
- Phase 1 loads 1-hop neighbors with CO_CHANGED edges → **200ms**
- Calculates temporal coupling + ownership automatically → **50ms**
- LLM synthesizes: "This usually changes with X, owned by @bob" → **2-5 seconds**
- **Total: 2-5 seconds, always performed**

**Regression Prevention in Practice:**
Without CodeRisk → Developer skips due diligence → Commits incomplete change → PR merged → Production breaks
With CodeRisk → Automated check flags coupled file → Developer updates both → Regression prevented

**Why This Works:**
- **Pre-commit timing:** Catches issues before public PR (private feedback, no sunk cost)
- **Always runs:** Developers won't spend 15 minutes manually, but automated checks always happen
- **Temporal coupling (unique moat):** CO_CHANGED edges reveal hidden dependencies structural analysis misses
- **Ownership context:** Surfaces who to ask for review, reducing knowledge gaps

### Graph as Database, Not Calculator

The graph stores **persistent facts** (code structure, git history, incidents). The agent calculates **ephemeral insights** (risk scores, blast radius) on-demand.

**What's in Neo4j (persistent):**
- Files and their import relationships
- Functions and their call relationships
- Git commits and what files they modified
- Co-change patterns (files that change together)
- Incident history and affected files

**What's NOT in Neo4j (computed on-demand):**
- Risk scores
- Blast radius calculations
- Ownership churn metrics
- Investigation traces

**Analogy:** Neo4j is like Wikipedia (stores facts), the LLM is like a research assistant (navigates facts to answer questions).

---

## Two-Phase Investigation

### Phase 1: Fast Baseline (No LLM, <500ms)

**Purpose:** Quickly filter out low-risk changes without expensive LLM calls.

**Process:**
1. Extract changed files from git diff
2. Calculate Tier 1 metrics in parallel (3 fast queries):
   - **Structural coupling:** How many files import/call this code?
   - **Temporal co-change:** What files frequently change together?
   - **Test coverage ratio:** Does this file have adequate tests?
3. Apply simple threshold heuristics:
   - IF coupling > 10 files → HIGH RISK
   - OR co-change > 70% frequency → HIGH RISK
   - OR test coverage < 30% → HIGH RISK
   - ELSE → LOW RISK

**Result:** 80% of changes are LOW risk and return immediately (~200ms average).

**Why this works:**
- Tier 1 metrics are **factual** (low false positive rate ~1-5%)
- Simple thresholds catch obvious risk signals
- Neo4j queries are fast (<50ms each when run in parallel)
- No LLM needed = cheap and fast

### Phase 2: LLM-Guided Investigation (2-5s)

**Purpose:** Deep dive into risky changes with contextual understanding.

**Process:**
1. **Initialize context:** Load 1-hop neighbor files from Neo4j into working memory
2. **Provide baseline evidence:** Give LLM the Tier 1 metrics + git diff context
3. **Iterative investigation** (max 3 rounds):
   - LLM reviews current evidence
   - LLM decides next action:
     - Calculate Tier 2 metric (ownership churn, incident similarity)
     - Expand to 2-hop neighbors for broader context
     - Finalize assessment (has enough evidence)
   - Execute LLM's request
   - Add new evidence to context
4. **Synthesize assessment:** LLM produces final risk level, confidence, key evidence, and recommendations

**Why this works:**
- LLM sees **full context** (not just numbers)
- LLM can **ask for more data** if needed (like a detective)
- **Hop limit** (max 3) prevents runaway costs
- Only used for **20% of checks** (selective, not exhaustive)

---

## Metric Tiers

### Tier 1: Always Calculate (Fast, Low FP Rate)

Calculated during Phase 1 baseline, cached for 15 minutes:

**1. Structural Coupling**
- Definition: Files that directly import or call changed code
- Query: 1-hop traversal on IMPORTS/CALLS edges
- FP Rate: ~1-2% (dependencies are factual)
- Cost: <50ms
- Evidence: "Function `check_auth()` is called by 12 other functions"
- **Regression Prevention Use Case:** Prevents breaking changes to widely-used functions—developer sees 12 dependents and adds defensive checks or tests all call sites

**2. Temporal Co-Change**
- Definition: Files that frequently change together in git history
- Query: Read CO_CHANGED edge weight (pre-computed during git ingestion)
- FP Rate: ~3-5% (temporal patterns are observable)
- Cost: <20ms
- Evidence: "File A and File B changed together in 15 of last 20 commits (75%)"
- **Regression Prevention Use Case:** Prevents incomplete changes—developer modifies A.py but forgets B.py (which changes together 80% of time), CodeRisk warns, developer updates both, regression avoided

**3. Test Coverage Ratio**
- Definition: Ratio of test code to source code
- Query: Find test files via naming convention + TESTS relationship
- FP Rate: ~5-8% (depends on naming consistency)
- Cost: <50ms
- Evidence: "auth.py has test ratio 0.45 (auth_test.py is 45% the size)"
- **Regression Prevention Use Case:** Prevents untested changes from reaching production—developer sees low test coverage, adds tests before committing, catches bugs pre-commit instead of in production

### Tier 2: Calculate on LLM Request (Context-Dependent)

Only calculated during Phase 2 if LLM asks for them:

**4. Ownership Churn**
- Definition: Primary code owner changed recently (within 90 days)
- Query: Aggregate git commits by developer, detect transitions
- FP Rate: ~5-7% (ownership is factual)
- Cost: <50ms
- Evidence: "Primary owner changed from Bob to Alice 14 days ago"
- When: LLM asks "Who owns this code?"
- **Regression Prevention Use Case:** Prevents knowledge gaps—new owner may lack context on subtle behavior, CodeRisk recommends pinging previous owner @bob for review, prevents regression from misunderstanding original intent

**5. Incident Similarity**
- Definition: Similarity between current change and past incident descriptions
- Query: Text search on incident database
- FP Rate: ~8-12% (noisy but directional)
- Cost: <50ms
- Evidence: "Commit mentions 'timeout' similar to Incident #123"
- When: LLM asks "Are there similar past incidents?"
- **Regression Prevention Use Case:** Prevents repeating past failures—developer modifying timeout logic sees warning about Incident #123 (auth timeout after 30s), adds defensive timeout handling, prevents recurrence of same incident

---

## Working Memory

**Goal:** Load graph data once, navigate in-memory during investigation to provide regression prevention context.

**What gets loaded (Regression Prevention Data):**
- Changed files (from git diff)
- 1-hop neighbor files (imports, importers, **co-changed files via CO_CHANGED edges**)
- Tier 1 metrics (pre-calculated)
- **Git commit history (90-day window) with ownership context**
- **Incident links (if any exist) for failure history**

**Benefits for Regression Prevention:**
- Single Neo4j query per investigation (fast)
- LLM has full context: **ownership + temporal coupling + incident history**
- Can expand to 2-hop if LLM requests (controlled growth)
- **Ownership context:** Surfaces who wrote code, who to ask for review
- **Temporal context:** Reveals which files change together (hidden dependencies)

**Limits:**
- Max 3 investigation rounds (prevents runaway costs)
- Max 2-hop graph traversal (prevents exponential explosion)

---

## Performance Characteristics

### Expected Latency

| Scenario | Frequency | Latency | Why |
|----------|-----------|---------|-----|
| **LOW risk (Phase 1 only)** | 80% of checks | 200ms | Simple metrics, no LLM |
| **HIGH risk (Phase 1 + 2)** | 20% of checks | 3-5s | LLM investigation |
| **Cached baseline** | 85-90% hit rate | 5ms | Redis cache for repeat files |

### Cost Economics

**Phase 1 (baseline):**
- 3 Neo4j queries (~40ms each, parallel)
- No LLM calls
- Cost: ~$0.001 per check (Neo4j compute)

**Phase 2 (LLM investigation):**
- 3-4 LLM calls (decision + synthesis)
- ~2K tokens per call
- Cost: ~$0.01 per investigation

**Daily cost for 100 checks:**
- 80 checks × $0.001 (Phase 1 only) = $0.08
- 20 checks × $0.01 (Phase 2) = $0.20
- **Total: ~$0.30/day** (vs $5-10/day with always-LLM approach)

---

## False Positive Management

### User Feedback Loop

**Process:**
1. User runs `crisk check` and gets HIGH risk assessment
2. User disagrees (thinks it's a false alarm)
3. User runs `crisk feedback --false-positive --reason "intentional coupling"`
4. System records feedback in local database
5. If metric crosses 3% FP rate (with >20 samples), auto-disable

**Why this matters:**
- Builds trust through self-correction
- Learns from user's domain knowledge
- Prevents metric degradation over time

### Metric Validation

Each metric tracks:
- Total uses
- True positives (user confirmed risk)
- False positives (user rejected assessment)
- FP rate (false_positives / total_uses)
- Enabled/disabled status

**Auto-exclusion:** Metrics with >3% FP rate (and >20 samples) are disabled until reviewed.

---

## Evidence Chain

The investigation produces a trace showing how the risk assessment was reached:

**Structure:**
- Phase 1 baseline results (Tier 1 metrics)
- Phase 2 investigation steps (what the LLM requested and found)
- Final synthesis (risk level, confidence, key evidence)
- Recommendations (actionable next steps)

**Purpose:**
- Transparency: User sees why assessment was made
- Debugging: Understand false positives
- Learning: Improve prompts and thresholds over time

**Storage:**
- Investigation traces are cached locally (15-minute TTL)
- User can review trace with `crisk explain <commit-sha>`

---

## Integration with Graph Ontology

See [graph_ontology.md](graph_ontology.md) for full schema details.

**What the agent queries:**
- **Layer 1 (Structure):** File, Function, Class nodes + IMPORTS, CALLS, CONTAINS edges
- **Layer 2 (Temporal):** Commit, Developer nodes + CO_CHANGED, MODIFIES edges
- **Layer 3 (Incidents):** Incident nodes + CAUSED_BY, AFFECTS edges

**Query patterns:**
- 1-hop traversal: Get direct dependencies/dependents
- Co-change lookup: Read pre-computed edge weights
- Incident search: Text similarity on descriptions
- Ownership analysis: Aggregate commits by developer

---

## Design Decisions

### Why Two-Phase Architecture?

**Single-phase (always use LLM):**
- Slow: 3-5s for every check (even trivial changes)
- Expensive: ~$0.05 per check × 100/day = $5/day
- Overkill: 80% of changes don't need deep analysis

**Two-phase (fast baseline + selective LLM):**
- Fast: 200ms for 80% of checks
- Cheap: ~$0.30/day for 100 checks
- Smart: Deep investigation only when needed

### Why Max 3 Investigation Hops?

**Without limits:**
- Exponential explosion: 10 neighbors/hop → 10³ = 1000 nodes
- Unbounded cost: Could exceed $0.50 per check
- Slow: Could take 30+ seconds

**With 3-hop limit:**
- Controlled: ~20-50 nodes explored
- Predictable: $0.01-0.03 per investigation
- Fast: 3-5s typical

### Why On-Demand Calculation?

**Pre-computed scores approach:**
- Expensive: High upfront cost during `crisk init`
- Stale: Scores don't reflect latest context
- High FP: Complex statistical models (15-25% FP rate)

**On-demand calculation:**
- Fresh: Always uses latest graph data
- Contextual: LLM combines multiple signals
- Low FP: Simple metrics (1-5% FP) + LLM reasoning

---

## MVP Implementation Notes

**What's included in MVP:**
- Two-phase investigation (Phase 1 baseline, Phase 2 LLM)
- Tier 1 metrics (coupling, co-change, test coverage)
- Basic Tier 2 metrics (ownership, incidents)
- Working memory (1-hop neighbors)
- Evidence chain generation
- False positive tracking

**What's deferred to post-MVP:**
- Advanced caching strategies (beyond basic 15-min TTL)
- 2-hop graph expansion (stick to 1-hop for now)
- Adaptive thresholds (use fixed thresholds initially)
- Multi-agent verification
- Phase 0 pre-analysis (docs detection, security keywords)

**Timeline:** 4-6 weeks for MVP implementation.

---

## Related Documentation

- **[graph_ontology.md](graph_ontology.md)** - Graph schema and entity relationships
- **[risk_assessment_methodology.md](risk_assessment_methodology.md)** - Metric formulas and thresholds
- **[00-product/developer_experience.md](../00-product/developer_experience.md)** - CLI workflow and user experience
- **[00-product/mvp_vision.md](../00-product/mvp_vision.md)** - Overall MVP scope and goals
