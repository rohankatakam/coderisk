# Agentic Investigation Design

**Version:** 4.0
**Last Updated:** October 10, 2025
**Purpose:** LLM-guided code risk investigation with minimal graph queries and evidence-based reasoning
**Design Philosophy:** Few robust metrics + LLM intelligence, not exhaustive pre-computation *(12-factor: Factors 3, 8, 10)*

**Latest Updates:**
- **Phase 0 Pre-Analysis** - Adaptive security/docs/config detection before baseline (see [ADR-005](decisions/005-confidence-driven-investigation.md))
- **Confidence-Driven Investigation** - Dynamic stopping based on evidence quality, not fixed hop count
- **Adaptive Configuration Selection** - Domain-aware thresholds (Python web vs Go backend)
- **Pattern Recombination** - Hybrid ARC insights from complementary patterns

---

## 1. Core Concept

CodeRisk uses an **LLM-guided agent** that selectively calculates robust metrics based on investigation needs, not exhaustive pre-computation. The agent decides what to calculate and when to stop based on evidence quality.

**Key Principles:**
- **Selective, not exhaustive** - Only calculate metrics the LLM requests (Tier 1 always, Tier 2 on-demand)
- **Evidence-driven** - LLM synthesizes risk from multiple low-FP metrics, not single scores
- **Self-validating** - Metrics track FP rates via user feedback, auto-excluded if >3%
- **Hop-limited** - Max 3 hops (1-hop structure, 2-hop investigation) prevents cost explosion
- **Cache-aware** - Redis stores metric results (15-min TTL), avoid redundant queries

**Contrast with Exhaustive Approaches:**
- ❌ **Pre-compute everything**: Calculate all O(n²) file relationships upfront → Infeasible for 10K+ file repos
- ❌ **Load entire graph**: Pass full graph to LLM → Exceeds 200K token context window, $3/request
- ✅ **Agentic traversal**: Load 1-hop neighbors (15 files), LLM decides next step → <2K tokens, $0.004/request

### 1.1. Why Agentic Traversal? The Graph Explosion Problem

**The Fundamental Issue:**
```
Scenario: Developer changes auth.py in 10,000-file repository

Option 1 (Naive): Pre-compute everything
  - File pairs: 10,000 × 10,000 = 100,000,000 relationships
  - Metrics per pair: 5 (coupling, co-change, blast radius, etc.)
  - Total calculations: 500,000,000 metrics
  - Time: 500M × 0.01s = 58 days
  - Storage: 50GB
  - Result: INFEASIBLE

Option 2 (Agentic): Calculate on-demand, guided by LLM
  - Changed files: 1 (auth.py)
  - 1-hop neighbors: ~15 files (imports + importers)
  - Tier 1 metrics: 15 × 3 = 45 calculations
  - LLM-requested Tier 2: ~5 additional calculations
  - Total: 50 calculations
  - Time: 0.5 seconds
  - Storage: 5KB (cached)
  - Result: ✅ FEASIBLE

Efficiency gain: ~10,000,000x faster
```

**Key Insight:** 99% of relationships are irrelevant to any given code change. The LLM navigates to the 1% that matters.

### 1.2. Graph as Database, Not Calculator

**Critical Distinction:**

| Component | Role | What It Stores | What It Computes |
|-----------|------|----------------|------------------|
| **Neptune Graph** | Knowledge base | Files, functions, commits, relationships (IMPORTS, CALLS, CO_CHANGED) | Nothing (query engine only) |
| **Investigation Agent** | Decision engine | Nothing (stateless) | What to calculate, when to stop |
| **Working Memory** | Temporary context | Small subgraph (~15 files) during investigation | Nothing (just holds data) |
| **Redis Cache** | Performance layer | Calculated metric results (15-min TTL) | Nothing (cache only) |
| **LLM** | Synthesizer | Nothing (stateless API) | Risk assessment from evidence |

**What's in the graph (persistent):**
```cypher
// Factual relationships discovered during ingestion
(auth.py:File)-[:IMPORTS]->(user_service.py:File)
(auth.py)-[:CO_CHANGED {frequency: 0.87}]->(user_service.py)
(login:Function)-[:CALLS]->(validate_token:Function)
(auth.py)-[:AFFECTED_BY]->(INC-453:Incident)
```

**What's NOT in the graph (calculated on-demand):**
```
❌ Risk scores (LLM synthesizes from evidence)
❌ Blast radius (calculated when LLM requests it)
❌ Ownership churn (Tier 2 metric, on-demand only)
❌ "Is this change risky?" (LLM decision, not stored)
```

**Analogy:** The graph is like Wikipedia (stores facts), the LLM is like a research assistant (navigates facts to answer questions).

### 1.3. Context Window Economics *(12-factor: Factor 3)*

**Why we can't load the entire graph:**

```
Full graph approach:
  - 10,000 files × 100 tokens each = 1,000,000 tokens
  - Claude 3.5 Sonnet context limit: 200,000 tokens
  - Over limit by 5x → Cannot fit
  - Even if it fit: Cost = $3/request (GPT-4 Turbo pricing)

Agentic approach:
  - 1-hop neighbors: 15 files × 100 tokens = 1,500 tokens
  - Evidence chain: ~500 tokens
  - Total context: ~2,000 tokens
  - Cost: $0.004/request (750x cheaper)
  - Latency: 2s vs 30s (faster response time)
```

**The agent is our intelligent navigator:** It loads minimal data, explores selectively, and stops when confident.

**Related Documentation:**
- **[risk_assessment_methodology.md](risk_assessment_methodology.md)** - Detailed metric formulas, thresholds, and validation framework
- **[prompt_engineering_design.md](prompt_engineering_design.md)** - LLM prompt architecture and context management
- **[graph_ontology.md](graph_ontology.md)** - Graph schema and data sources for metrics

---

## 2. Investigation Flow *(12-factor: Factor 8 - Own your control flow)*

### 2.1. Three-Phase Architecture (Updated Oct 2025)

> **See [ADR-005](decisions/005-confidence-driven-investigation.md)** for full rationale behind Phase 0 addition and confidence-driven investigation.

```
┌─────────────────────────────────────────────────────────────┐
│  PHASE 0: ADAPTIVE PRE-ANALYSIS (No LLM, <50ms) ✨NEW      │
│                                                             │
│  git diff → Extract changed files + modifications          │
│     ↓                                                       │
│  Fast heuristics (keyword/path matching):                  │
│     • Security keyword detection → Force CRITICAL          │
│     • Documentation-only → Skip Phase 1/2 (return LOW)     │
│     • Production config → Force HIGH                       │
│     • Select domain-aware config (Python web vs Go)        │
│     ↓                                                       │
│  Decision:                                                 │
│     IF force_escalate → Skip to Phase 2                   │
│     IF skip_analysis → Return LOW immediately              │
│     ELSE → Proceed to Phase 1 with adaptive config         │
└─────────────────────────────────────────────────────────────┘
                                  ↓
┌─────────────────────────────────────────────────────────────┐
│  PHASE 1: BASELINE ASSESSMENT (No LLM, <500ms)             │
│                                                             │
│  Calculate Tier 1 metrics (parallel):                      │
│     • Structural coupling (1-hop IMPORTS/CALLS)            │
│     • Temporal co-change (CO_CHANGED edge weight)          │
│     • Test coverage ratio (TESTS relationship)             │
│     ↓                                                       │
│  Adaptive heuristic (using config from Phase 0):           │
│     IF coupling > threshold OR co_change > threshold       │
│        OR test_ratio < threshold                           │
│        → HIGH RISK (proceed to Phase 2)                    │
│     ELSE → LOW RISK (return early, no LLM needed)          │
└─────────────────────────────────────────────────────────────┘
                                  ↓ (only if HIGH)
┌─────────────────────────────────────────────────────────────┐
│  PHASE 2: CONFIDENCE-DRIVEN INVESTIGATION (2-5s) ✨UPDATED │
│                                                             │
│  1. INITIALIZE CONTEXT                                     │
│     - Load 1-hop neighbors into working memory (once)      │
│     - Prepare Tier 1 evidence + modification type for LLM  │
│     - Check for ARC pattern matches (hybrid matching)      │
│     ↓                                                       │
│  2. CONFIDENCE-BASED LOOP (max 5 iterations, adaptive)     │
│     For each iteration:                                    │
│       • LLM reviews current evidence                       │
│       • LLM assesses confidence (0.0-1.0)                  │
│       • IF confidence >= 0.85 → FINALIZE (stop early)      │
│       • ELSE LLM decides next action:                      │
│           a) Calculate Tier 2 metric (ownership/incidents) │
│           b) Expand to 2-hop neighbors                     │
│           c) Request ARC pattern analysis                  │
│       • Execute LLM request                                │
│       • Track breakthroughs (risk level changes)           │
│       • Add evidence to context                            │
│     ↓                                                       │
│  3. SYNTHESIZE RISK ASSESSMENT                             │
│     - LLM combines all evidence (including hybrid ARCs)    │
│     - Generate risk level + confidence + reasoning         │
│     - Explain breakthrough points in investigation         │
│     - Cache investigation trace (Redis, 15-min TTL)        │
└─────────────────────────────────────────────────────────────┘
```

### 2.2. Working Memory (Minimal Context)

**Goal:** Load graph data once, navigate in-memory during investigation

```python
class InvestigationContext:
    changed_files: List[str]                    # User's diff
    tier1_metrics: Dict[str, MetricResult]      # Baseline (always calculated)
    working_memory: Dict[str, Node]             # 1-hop neighbors (loaded once)
    evidence_chain: List[EvidenceItem]          # LLM reasoning trace
    hop_count: int                              # Current iteration (max 3)

class MetricResult:
    name: str                                   # "coupling", "co_change", etc.
    value: Any                                  # 12, 0.75, etc.
    evidence: str                               # "Called by 12 functions"
    fp_rate: float                              # 0.02 (2% FP rate)
    cached: bool                                # True if from Redis
```

**Benefits:**
- ✅ Single graph query per investigation (1-hop load)
- ✅ Redis caching reduces repeat calculations
- ✅ LLM context stays small (<2K tokens for evidence)

### 2.3. Expected Performance Distribution (With Phase 0)

**From empirical analysis** ([TEST_RESULTS_OCT_2025.md](../../03-implementation/testing/TEST_RESULTS_OCT_2025.md)):

```
Distribution of checks after Phase 0 implementation:

Phase 0 skip (docs-only):     20% → <10ms   (vs 13.5s current)
Phase 1 only (LOW risk):      60% → 50-200ms (current)
Phase 2 (HIGH risk):          20% → 2-4s    (vs 3-5s current, confidence stops early)

Weighted average latency:
  Before: 100% × 2,500ms = 2,500ms per check
  After:  20%×10ms + 60%×100ms + 20%×3,000ms = 662ms per check

Improvement: 3.8x faster on average
```

**False Positive Reduction:**
- Current baseline: ~50% FP rate (from test scenarios)
- Phase 0 security detection: Fixes auth.py false negative (currently LOW, should be CRITICAL)
- Phase 0 docs skip: Fixes README.md false positive (currently HIGH, should be LOW)
- Adaptive thresholds: 20-30% FP reduction on domain-specific repos
- **Target: 10-15% FP rate** (70-80% reduction)

**Insight Quality Improvements:**
- **Modification type awareness:** LLM knows if change is security/docs/config → specific recommendations
- **Hybrid ARC patterns:** "Auth coupling + no tests" vs generic "high coupling"
- **Breakthrough tracking:** Explain which evidence caused risk level changes
- **Expected: 3-5x more actionable recommendations**

---

## 3. Graph Schema (Persistent Only)

See [graph_ontology.md](graph_ontology.md) for full schema. Key entities:

### 3.1. Persistent Nodes (Stored in Neptune)

**Layer 1: Structure**
- `File` - {path, language, loc, last_modified_sha}
- `Function` - {name, signature, start_line, end_line, complexity}
- `Class` - {name, is_public}
- `Module` - {name, package_path}

**Layer 2: Temporal**
- `Commit` - {sha, timestamp, message, additions, deletions}
- `Developer` - {email, name, first_commit, last_commit}
- `PullRequest` - {number, created_at, merged_at}

**Layer 3: Incidents**
- `Incident` - {id, title, severity, created_at, resolved_at, description}

### 3.2. Persistent Relationships (Stored in Neptune)

**Structural:**
- `CALLS` - Function → Function
- `IMPORTS` - File → File
- `CONTAINS` - File → Function/Class

**Temporal:**
- `CO_CHANGED` - File ↔ File {frequency: 0.75, last_timestamp, window_days: 90}
- `AUTHORED` - Developer → Commit
- `MODIFIES` - Commit → File

**Incidents:**
- `CAUSED_BY` - Incident → Commit (manual link)
- `AFFECTS` - Incident → File (derived)
- `FIXED_BY` - Incident → Commit (manual link)

### 3.3. Ephemeral Data (NOT in Graph, Cached in Redis)

❌ Risk scores
❌ Blast radius calculations
❌ Ownership churn metrics
❌ Investigation traces
❌ Centrality scores

---

## 4. Implementation: Phase 1 (Baseline) *(12-factor: Factor 10 - Small, focused agents)*

### 4.1. Fast Baseline (No LLM)

```python
def baseline_risk_assessment(changed_files: List[str]) -> BaselineResult:
    """
    Calculate Tier 1 metrics in parallel, return risk level.
    Target: <500ms total
    """
    metrics = {}

    # Parallel queries (3 concurrent Neptune queries)
    with ThreadPoolExecutor(max_workers=3) as executor:
        for file_path in changed_files:
            # Check Redis cache first
            cached = redis.get(f"baseline:{file_path}")
            if cached and not_stale(cached):
                metrics[file_path] = json.loads(cached)
                continue

            # Query graph (parallel)
            coupling_future = executor.submit(
                query_coupling, file_path
            )
            co_change_future = executor.submit(
                query_co_change, file_path
            )
            test_ratio_future = executor.submit(
                query_test_coverage, file_path
            )

            # Collect results
            metrics[file_path] = {
                "coupling": coupling_future.result(),      # <50ms
                "co_change": co_change_future.result(),    # <20ms
                "test_ratio": test_ratio_future.result(),  # <50ms
            }

            # Cache in Redis (15-min TTL)
            redis.setex(
                f"baseline:{file_path}",
                900,  # 15 minutes
                json.dumps(metrics[file_path])
            )

    # Simple heuristic (no ML model, just thresholds)
    risk_level = "LOW"
    for file_path, m in metrics.items():
        if (m["coupling"]["value"] > 10 or
            m["co_change"]["value"] > 0.7 or
            m["test_ratio"]["value"] < 0.3):
            risk_level = "HIGH"
            break

    return BaselineResult(
        risk_level=risk_level,
        metrics=metrics,
        needs_investigation=(risk_level == "HIGH")
    )

def query_coupling(file_path: str) -> MetricResult:
    """1-hop IMPORTS/CALLS query"""
    query = """
    MATCH (f:File {path: $path})-[:IMPORTS|CALLS]-(neighbor)
    RETURN count(DISTINCT neighbor) as count
    """
    result = neptune.execute(query, path=file_path)
    return MetricResult(
        name="coupling",
        value=result["count"],
        evidence=f"File has {result['count']} direct dependencies",
        fp_rate=0.02  # 2% FP rate (factual)
    )

def query_co_change(file_path: str) -> MetricResult:
    """Read CO_CHANGED edge weight"""
    query = """
    MATCH (f:File {path: $path})-[r:CO_CHANGED]-(other)
    RETURN other.path as file, r.frequency as freq
    ORDER BY r.frequency DESC
    LIMIT 5
    """
    results = neptune.execute(query, path=file_path)
    max_freq = max([r["freq"] for r in results], default=0.0)
    return MetricResult(
        name="co_change",
        value=max_freq,
        evidence=f"Max co-change frequency: {max_freq:.0%}",
        fp_rate=0.05  # 5% FP rate (observable)
    )
```

**Performance:**
- Tier 1 metrics: ~120ms (3 queries in parallel, avg 40ms each)
- Redis cache hit: ~5ms
- Total baseline: <200ms (90% of cases cached)

### 4.2. LLM Decision Loop (Only if HIGH Risk)

```python
def llm_guided_investigation(context: InvestigationContext) -> RiskAssessment:
    """
    LLM decides what to investigate next.
    Target: 3-5s total (3 iterations max)
    """
    # Initialize: Load 1-hop neighbors (once)
    context.working_memory = load_1_hop_neighbors(context.changed_files)

    for iteration in range(3):  # Max 3 hops
        context.hop_count = iteration

        # LLM decides next action
        decision = llm_decide_action(context)

        if decision.action == "CALCULATE_METRIC":
            # LLM requested Tier 2 metric (ownership or incidents)
            metric_result = calculate_tier2_metric(
                metric_name=decision.metric_name,
                file_path=decision.target_file,
                context=context.working_memory
            )
            context.evidence_chain.append(EvidenceItem(
                type="metric",
                data=metric_result,
                hop=iteration
            ))

        elif decision.action == "EXPAND_GRAPH":
            # LLM wants 2-hop neighbors
            expanded = load_2_hop_neighbors(decision.target_file)
            context.working_memory.update(expanded)
            context.evidence_chain.append(EvidenceItem(
                type="expansion",
                data=f"Loaded {len(expanded)} 2-hop neighbors",
                hop=iteration
            ))

        elif decision.action == "FINALIZE":
            # LLM has enough evidence
            break

    # LLM synthesizes final risk assessment
    assessment = llm_synthesize_risk(context)

    # Cache investigation trace
    redis.setex(
        f"investigation:{hash(context.changed_files)}",
        900,  # 15 minutes
        json.dumps(assessment.to_dict())
    )

    return assessment
```

**LLM Prompts:**

**Decision Prompt:**
```
You are investigating a code change for risk. Current evidence:

Tier 1 Metrics (always calculated):
- Structural coupling: {coupling} files depend on this
- Co-change frequency: {co_change}
- Test coverage ratio: {test_ratio}

{evidence_chain}

Based on this evidence, decide your next action:
1. CALCULATE_METRIC: Request ownership or incident data
2. EXPAND_GRAPH: Load 2-hop neighbors for broader context
3. FINALIZE: Enough evidence to assess risk

Respond with JSON: {"action": "...", "reasoning": "...", "target": "..."}
```

**Synthesis Prompt:**
```
Synthesize a risk assessment based on all gathered evidence:

Evidence:
{evidence_chain}

Provide:
1. Risk level: LOW, MEDIUM, HIGH
2. Confidence: 0.0-1.0
3. Key factors: List 3-5 evidence points supporting your conclusion
4. Recommendations: Actionable next steps for developer

Respond with JSON.
```

---

## 5. Metric Validation System *(12-factor: Factor 7 - Contact humans with tool calls)*

### 5.1. False Positive Tracking (Postgres Schema)

```sql
CREATE TABLE metric_validations (
    id SERIAL PRIMARY KEY,
    metric_name VARCHAR(50) NOT NULL,  -- "coupling", "co_change", etc.
    file_path VARCHAR(500) NOT NULL,
    metric_value JSONB NOT NULL,       -- {"value": 12, "evidence": "..."}
    user_feedback VARCHAR(20),         -- "true_positive", "false_positive", null
    feedback_reason TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE metric_stats (
    metric_name VARCHAR(50) PRIMARY KEY,
    total_uses INT DEFAULT 0,
    false_positives INT DEFAULT 0,
    true_positives INT DEFAULT 0,
    fp_rate FLOAT GENERATED ALWAYS AS (
        CASE WHEN total_uses > 0
        THEN false_positives::FLOAT / total_uses
        ELSE 0.0 END
    ) STORED,
    is_enabled BOOLEAN DEFAULT TRUE,
    last_updated TIMESTAMP DEFAULT NOW()
);
```

**Auto-exclusion logic:**
```python
def record_feedback(metric_name: str, is_false_positive: bool):
    """User provides feedback on metric accuracy"""
    if is_false_positive:
        db.execute("""
            UPDATE metric_stats
            SET false_positives = false_positives + 1,
                total_uses = total_uses + 1,
                last_updated = NOW()
            WHERE metric_name = $1
        """, metric_name)
    else:
        db.execute("""
            UPDATE metric_stats
            SET true_positives = true_positives + 1,
                total_uses = total_uses + 1,
                last_updated = NOW()
            WHERE metric_name = $1
        """, metric_name)

    # Auto-disable if FP rate > 3% and enough samples
    stats = db.query("""
        SELECT fp_rate, total_uses FROM metric_stats
        WHERE metric_name = $1
    """, metric_name)

    if stats.total_uses >= 20 and stats.fp_rate > 0.03:
        db.execute("""
            UPDATE metric_stats
            SET is_enabled = FALSE
            WHERE metric_name = $1
        """, metric_name)
        logger.warn(f"Metric {metric_name} auto-disabled: "
                   f"FP rate {stats.fp_rate:.1%} (n={stats.total_uses})")
```

### 5.2. User Override Workflow

```bash
# During crisk check, user sees HIGH risk assessment
$ crisk check
→ Risk: HIGH
→ Coupling: 12 files depend on this (HIGH)
→ Co-change: 0.8 frequency with auth.py (HIGH)

# User disagrees, provides feedback
$ crisk feedback --false-positive --reason "This coupling is intentional, well-tested"

# System records feedback
→ Updates metric_stats table
→ Stores feedback_reason for future analysis
→ If fp_rate crosses 3%, metric is auto-disabled

# Admin reviews disabled metrics
$ crisk metrics --disabled
→ Metric "coupling" disabled (FP rate: 4.2%, n=25)
→ Common false positive reasons:
   - "Intentional coupling in framework code" (40%)
   - "Test files inflating count" (30%)
```

---

## 6. Evidence Chain Construction

### 6.1. Investigation Trace Format (Cached in Redis)

```json
{
  "risk_level": "HIGH",
  "confidence": 0.85,
  "investigation_summary": {
    "phase1_baseline": {
      "duration_ms": 180,
      "metrics": {
        "coupling": {"value": 12, "threshold": 10, "signal": "HIGH"},
        "co_change": {"value": 0.75, "threshold": 0.7, "signal": "HIGH"},
        "test_ratio": {"value": 0.3, "threshold": 0.3, "signal": "MEDIUM"}
      },
      "decision": "ESCALATE_TO_PHASE2"
    },
    "phase2_investigation": [
      {
        "hop": 1,
        "llm_action": "CALCULATE_METRIC",
        "metric_requested": "ownership_churn",
        "file": "auth/permissions.py",
        "result": {
          "current_owner": "alice@example.com",
          "previous_owner": "bob@example.com",
          "days_since_transition": 14
        },
        "llm_reasoning": "High coupling suggests checking ownership stability"
      },
      {
        "hop": 2,
        "llm_action": "CALCULATE_METRIC",
        "metric_requested": "incident_similarity",
        "file": "auth/permissions.py",
        "result": [
          {"incident_id": 123, "similarity": 0.82, "title": "Auth timeout after permission check"}
        ],
        "llm_reasoning": "Recent ownership change + past incident suggests elevated risk"
      },
      {
        "hop": 3,
        "llm_action": "FINALIZE",
        "llm_reasoning": "Sufficient evidence: high coupling, owner transition, similar incident"
      }
    ]
  },
  "key_evidence": [
    "12 files directly depend on this code (high coupling)",
    "75% co-change frequency with auth.py (tight temporal coupling)",
    "Code owner changed 14 days ago (ownership instability)",
    "Similar to Incident #123: 'Auth timeout after permission check'"
  ],
  "recommendations": [
    "Add integration tests for permission check timeout scenarios",
    "Review auth module coupling (consider facade pattern)",
    "Ensure new owner (alice@) is familiar with incident #123"
  ],
  "metadata": {
    "total_duration_ms": 4200,
    "llm_calls": 4,
    "graph_queries": 5,
    "cache_hits": 2
  }
}
```

---

## 7. Performance Characteristics

### 7.1. Latency Breakdown (Revised for Two-Phase)

| Phase | Typical Time | Cache Hit | Optimizations |
|-------|-------------|-----------|---------------|
| **Phase 1: Baseline** | **180ms** | **5ms** | **Parallel queries, Redis cache** |
| - Coupling query | 50ms | - | 1-hop IMPORTS/CALLS |
| - Co-change query | 20ms | - | Edge property read |
| - Test ratio query | 50ms | - | TESTS relationship |
| - Heuristic eval | 10ms | - | Simple thresholds |
| **Phase 2: LLM (only 20% of checks)** | **3-5s** | **N/A** | **Selective metric calculation** |
| - Load 1-hop neighbors | 100ms | - | Single Neptune query |
| - LLM hop 1 | 1.2s | - | Ownership/incident metric |
| - LLM hop 2 | 1.0s | - | Cached graph data |
| - LLM hop 3 | 0.8s | - | Synthesis |
| **Total (LOW risk, 80%)** | **200ms** | **5ms** | **No LLM needed** |
| **Total (HIGH risk, 20%)** | **3.5s** | **N/A** | **LLM investigation** |

### 7.2. Cache Hit Rates

**Redis caching (15-min TTL):**
- Baseline metrics: **85-90%** (same files checked frequently)
- Incident similarity: **60-70%** (static incident DB)
- Ownership data: **70-80%** (infrequent git commits)

**Performance impact:**
- Cold start (no cache): ~200ms Phase 1
- Warm cache: ~5ms Phase 1
- **Speedup: 40x** for repeat checks on same files

---

## 8. Key Design Decisions

### 8.1. Why Two-Phase Architecture?

**Single-phase (always LLM):**
- ❌ Slow: 3-5s for every check (even trivial ones)
- ❌ Expensive: ~$0.05 per check × 100 checks/day = $5/day
- ❌ Overkill: 80% of changes are low-risk, don't need deep analysis

**Two-phase (baseline + selective LLM):**
- ✅ Fast: 200ms for 80% of checks (LOW risk, no LLM)
- ✅ Cheap: $0.01 per Phase 1 check × 100 = $1/day
- ✅ Smart: Deep investigation only when needed

### 8.2. Why No Pre-Computed Risk Scores?

**Legacy approach (ΔDBR, HDCC, G², etc.):**
- ❌ Expensive: $15 `crisk init` per repo
- ❌ Stale: Pre-computed scores don't reflect latest context
- ❌ High FP: Complex models (15-25% FP rate)

**On-demand calculation:**
- ✅ Fresh: Always uses latest graph data
- ✅ Contextual: LLM combines multiple signals
- ✅ Low FP: Simple metrics (1-5% FP rate), LLM reasoning reduces noise

### 8.3. Why Max 3 Hops?

**Without limits:**
- Risk: Exponential explosion (10 neighbors/hop → 10³ = 1000 nodes)
- Cost: Unbounded LLM calls ($0.50+)
- Latency: Could take 30+ seconds

**With 3-hop limit:**
- ✅ Controlled: ~20-50 nodes explored
- ✅ Predictable: $0.03-0.05 per investigation
- ✅ Fast: 3-5s typical

---

## 9. Implementation Checklist

**Phase 1: Baseline (V1)**
- [ ] Implement Tier 1 metrics (coupling, co-change, test_ratio)
- [ ] Neptune query functions (1-hop traversal)
- [ ] Redis caching layer (15-min TTL)
- [ ] Simple heuristic (threshold-based risk level)

**Phase 2: LLM Investigation (V1)**
- [ ] LLM decision prompt (CALCULATE_METRIC, EXPAND_GRAPH, FINALIZE)
- [ ] LLM synthesis prompt (risk level + evidence + recommendations)
- [ ] Tier 2 metrics (ownership_churn, incident_similarity)
- [ ] Working memory management (1-hop neighbors)

**Metric Validation (V1)**
- [ ] Postgres schema (metric_validations, metric_stats)
- [ ] User feedback CLI (`crisk feedback --false-positive`)
- [ ] Auto-disable logic (>3% FP rate)

**Future (V2):**
- [ ] 2-hop graph expansion (if LLM requests)
- [ ] Multi-agent verification (parallel investigations)
- [ ] Adaptive thresholds (learn from user feedback)

---

## 10. Comparison: Legacy vs. New Architecture

| Aspect | Legacy (search_strategy.md) | New (This Doc) |
|--------|---------------------------|----------------|
| **Init cost** | $15 (heavy batch processing) | $0 (no pre-computation) |
| **Check latency** | 2-5s (always LLM) | 200ms (80%), 3-5s (20%) |
| **Metrics** | 9 complex (ΔDBR, HDCC, G², OAM, Bridge, etc.) | 5 simple (coupling, co-change, test, ownership, incidents) |
| **FP rate** | 10-20% (statistical models) | <3% (factual + LLM reasoning) |
| **Graph storage** | Risk scores, centrality, blast radius | Only structure + history |
| **Caching** | Spatial context, region maps | Redis metrics (15-min) |
| **LLM usage** | Every check (expensive) | Only HIGH risk (selective) |
| **Total daily cost** | $5-10 (100 checks) | $1-2 (100 checks) |

---

**See also:**
- [graph_ontology.md](graph_ontology.md) - Graph schema and persistent entities
- [risk_assessment_methodology.md](risk_assessment_methodology.md) - Metric formulas, thresholds, and validation
- [prompt_engineering_design.md](prompt_engineering_design.md) - LLM prompt design and context management
- [cloud_deployment.md](cloud_deployment.md) - Infrastructure and scaling
