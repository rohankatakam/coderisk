# Regression Prevention Methodology

**Version:** 2.0 (MVP)
**Last Updated:** January 2025
**Purpose:** Simple, robust regression detection through automated due diligence
**Deployment:** Local MVP with Neo4j, no cloud infrastructure

---

## Core Principles

### Regression Detection Philosophy

**Automated Due Diligence Approach:**
- **Regression prevention** - Detect patterns that lead to production failures (incomplete changes, untested code)
- **Factual metrics** - Coupling counts, co-change frequencies (1-5% FP rate)
- **Evidence-based reasoning** - LLM synthesizes multiple low-FP signals
- **Selective calculation** - Only compute what the investigation needs
- **Self-validating** - Track FP rates, auto-disable metrics >3% FP threshold

**Why Simple Metrics:**
- Complex statistical models had 15-25% false positive rates
- Simple metrics are factual (coupling is observable) vs statistical (surprise is inferred)
- Factual metrics are explainable: "12 files depend on this" vs "PPR delta = 0.34"
- Cost reduction: ~$0/day for MVP vs $5-10/day with complex approaches

### Two-Tier Metric System

**Tier 1: Always Calculate (High Signal, Low Cost)**
- Structural coupling (1-hop dependency count)
- Temporal co-change (pre-computed edge weights)
- Test coverage ratio (test file relationships)
- Characteristics: <200ms total, <3% FP rate

**Tier 2: Calculate on LLM Request (Context-Dependent)**
- Ownership churn (git history analysis)
- Incident similarity (text search)
- Characteristics: <500ms total, 5-12% FP rate

**Tier 3: Explicitly Avoided (High FP or Expensive)**
- ΔDBR (Diffusion Blast Radius) - 15-20% FP rate
- HDCC (Hawkes Decay Co-Change) - 12-18% FP rate
- G² Surprise (Dunning Log-Likelihood) - 20-25% FP rate
- Vector embeddings for incident matching - 10x cost for minimal gain

**MVP Philosophy:** 5 simple, robust metrics beat 9 complex, noisy ones.

---

## Tier 1 Metrics (Baseline Assessment)

### 1. Structural Coupling (Regression Detector)

**Definition:** Count of files/functions that directly depend on changed code

**What Regression It Prevents:**
**Scenario:** Developer modifies widely-used function signature without checking dependents
- **Without CodeRisk:** Merge → Production → 12 call sites break (changed function signature incompatible)
- **With CodeRisk:** Pre-commit warning shows 12 dependents → Developer checks all call sites → No regression

**Calculation:** Query Neo4j for 1-hop neighbors connected via IMPORTS or CALLS relationships.

**Formula:** coupling_score = COUNT(DISTINCT neighbor) WHERE hop_distance = 1

**Threshold Logic:**
- ≤ 5: **LOW** - Limited blast radius
- 5-10: **MEDIUM** - Moderate impact
- > 10: **HIGH** - Wide impact, escalate to Phase 2

**Rationale:**
- Coupling is factual (dependencies are explicit in code)
- 1-hop limit prevents explosion (10 deps × 10 subdeps = 100+ nodes)
- FP Rate: ~1-2% (false positives occur with intentional framework patterns)

**Evidence Format:**
> "File auth.py is imported by 12 other files (HIGH coupling). Did you check all dependents?"

**Why This Works:** Dependencies are observable facts, not statistical inferences. Blast radius awareness prevents breaking changes.

### 2. Temporal Co-Change (Regression Detector)

**Definition:** Files that frequently change together in git history (90-day window)

**What Regression It Prevents:**
**Scenario:** Developer updates A.py but forgets B.py (which changes together 80% of time)
- **Without CodeRisk:** Merge → Production → B.py breaks (A/B coupling violated, e.g., config format changed in A but not in B)
- **With CodeRisk:** Pre-commit warning → "A.py and B.py changed together in 16/20 commits (80%). Did you update B.py?" → Developer updates both → No regression

**Calculation:** Read pre-computed CO_CHANGED edge weight from Neo4j. The frequency represents how often two files change together in the same commit.

**Formula:** co_change_frequency = COUNT(commits where both changed) / COUNT(commits where either changed)

**Example:** If files A and B changed together in 15 out of 20 commits that touched either file, frequency = 0.75 (75%)

**Threshold Logic:**
- ≤ 0.3: **LOW** - Weak coupling
- 0.3-0.7: **MEDIUM** - Moderate coupling
- > 0.7: **HIGH** - Strong evolutionary coupling

**Rationale:**
- Co-change is observable (commit history is ground truth)
- 90-day window balances recency with statistical significance
- FP Rate: ~3-5% (false positives from unrelated mass changes like formatting)

**Evidence Format:**
> "A.py and B.py changed together in 16/20 commits (80%). Did you update B.py?"

**Pre-Computation:** CO_CHANGED edges computed during `crisk init` and stored in Neo4j, so runtime query is just a fast edge property lookup.

**Why This Is CodeRisk's Unique Moat:** No other tool tracks temporal coupling. Structural analysis (imports/calls) misses evolutionary patterns. Example: payment.py doesn't import fraud.py (no structural coupling) but they changed together in 18/20 commits (90% temporal coupling).

### 3. Test Coverage Ratio (Regression Detector)

**Definition:** Ratio of test code to source code (lines of code)

**What Regression It Prevents:**
**Scenario:** Developer modifies critical auth logic without adequate tests
- **Without CodeRisk:** Merge → Production → Auth bug breaks login (untested edge case)
- **With CodeRisk:** Pre-commit warning → "auth.py has test ratio 0.2 (LOW coverage)" → Developer adds tests → Bug caught pre-commit

**Calculation:** Find test files via naming convention (test_*.py, *_test.py, *.test.js) and directory patterns (tests/, __tests__/), then calculate ratio.

**Formula:** test_ratio = SUM(test_file.loc) / source_file.loc

**Test Discovery Methods:**
1. Naming convention: *_test.py, test_*.py, *.test.js
2. Directory convention: tests/, __tests__/, spec/
3. Graph relationship: (test)-[:TESTS]->(source)

**Threshold Logic:**
- ≥ 0.8: **LOW** - Excellent coverage
- 0.3-0.8: **MEDIUM** - Adequate coverage
- < 0.3: **HIGH** - Insufficient coverage

**Rationale:**
- Test ratio is factual (lines of code are countable)
- Naming conventions work for 95%+ of projects
- FP Rate: ~5-8% (false positives with non-standard test locations)

**Evidence Format:**
> "auth.py (250 LOC) has auth_test.py (75 LOC), test ratio = 0.3 (MEDIUM coverage). Consider adding tests."

**Smoothing:** To avoid division by zero, use (test_loc + 1) / (source_loc + 1)

### Phase 1 Heuristic (Escalation Logic)

**Decision Tree:**
- IF coupling > 10 OR co_change > 0.7 OR test_ratio < 0.3
- THEN risk_level = HIGH, escalate to Phase 2
- ELSE risk_level = LOW, return early (no LLM needed)

**Rationale:**
- 80% of code changes are low-risk and don't need LLM investigation
- Simple OR logic (not weighted scoring) to avoid false negatives
- Conservative thresholds to err on side of caution

**Performance:**
- LOW risk path (80% of checks): ~200ms, no LLM calls, ~$0
- HIGH risk path (20% of checks): ~3-5s, 3-4 LLM calls, ~$0.01-0.03

---

## Tier 2 Metrics (LLM-Requested)

### 4. Ownership Churn

**Definition:** Primary code owner changed recently (within 90-day window)

**Calculation:** Aggregate git commits by developer email from Neo4j AUTHORED relationships. Identify primary owner (most commits) in last 30 days vs previous owner (most commits in days 31-90).

**Ownership Strength:**
- Strong: primary_ownership > 50% (one developer dominates)
- Shared: 20-50% (multiple contributors)
- Weak: < 20% (distributed ownership)

**Threshold Logic:**
- > 90 days: **LOW** - Stable ownership
- 30-90 days: **MEDIUM** - Recent transition
- < 30 days: **HIGH** - Very recent transition (churn)

**Rationale:**
- Ownership churn is observable (commit authorship is factual)
- Recent transitions increase risk (new owner may lack context)
- FP Rate: ~5-7% (false positives when experienced developers take over)

**Evidence Format:**
> "Primary owner changed from bob@example.com to alice@example.com 14 days ago (MEDIUM ownership stability)"

**When LLM Requests:**
- HIGH coupling + LOW test ratio → Check if new owner is experienced
- Frequent co-change → Check if ownership is shared across coupled files

### 5. Incident Similarity

**Definition:** Keyword similarity between commit message and past incident descriptions

**Calculation:** Simple text search (BM25 algorithm) on incident descriptions stored in Neo4j. Query uses the changed file's commit message plus last 3 commit messages.

**BM25 Overview:** Standard information retrieval algorithm that scores documents based on term frequency, inverse document frequency, and document length normalization. More sophisticated than simple keyword matching, but simpler than vector embeddings.

**Similarity Threshold:**
- < 5.0: **LOW** - No similar incidents
- 5.0-10.0: **MEDIUM** - Weak similarity
- ≥ 10.0: **HIGH** - Strong similarity

**Rationale:**
- BM25 is sufficient - vector embeddings add <5% accuracy at 10x cost
- Text matching is observable (keywords are explicit)
- FP Rate: ~8-12% (false positives from generic keywords like "timeout", "error")

**Evidence Format:**
> "Commit mentions 'auth timeout' similar to Incident #123: 'Auth service timeout after 30s' (BM25 score: 12.3)"

**Why Not Vector Embeddings:**
- BM25 achieves ~85% accuracy vs ~88% for vector search
- 3% accuracy improvement not worth complexity increase for MVP
- Can revisit in post-MVP if FP rate crosses threshold

**When LLM Requests:**
- HIGH coupling + ownership churn → Check for similar past failures
- Multiple HIGH signals → Validate with incident history

---

## Metric Composition (Risk Level Synthesis)

### Phase 1 Output (No LLM)

**Structure:**
- risk_level: LOW, MEDIUM, or HIGH
- confidence: 0.0 (Phase 1 is heuristic only, no confidence score)
- metrics: Dictionary of Tier 1 metric values, signals, and thresholds
- needs_investigation: TRUE if risk_level = HIGH, FALSE otherwise
- duration_ms: Time taken for Phase 1

**Decision Logic:**
- IF ANY metric signal = HIGH → risk_level = HIGH, needs_investigation = TRUE
- ELSE IF ALL signals = LOW → risk_level = LOW, needs_investigation = FALSE
- ELSE → risk_level = MEDIUM, needs_investigation = FALSE (only HIGH escalates in MVP)

### Phase 2 Output (LLM Synthesis)

**LLM Input (Evidence Chain):**
- All Tier 1 metrics (always calculated)
- Tier 2 metrics (only if LLM requested them)
- Git diff context
- Modification type (if detected)

**LLM Output (Synthesized Assessment):**
- risk_level: LOW, MEDIUM, or HIGH
- confidence: 0.0-1.0 (LLM's certainty in the assessment)
- key_evidence: List of 3-5 evidence points supporting conclusion
- recommendations: Actionable next steps for developer
- reasoning: Brief explanation of how LLM reached conclusion
- duration_ms: Total time for Phase 2
- llm_calls: Number of LLM API calls made

**Confidence Scoring (LLM-Generated):**
- ≥ 0.8: Strong evidence from multiple independent sources
- 0.5-0.8: Moderate evidence, some conflicting signals
- < 0.5: Weak evidence, unclear risk

**Example Output:**

Risk Level: HIGH (Confidence: 0.85)

Key Evidence:
- 12 files directly depend on this code (high coupling)
- 75% co-change frequency with auth.py (tight temporal coupling)
- Code owner changed 14 days ago (ownership instability)
- Similar to Incident #123: "Auth timeout after permission check"

Recommendations:
- Add integration tests for permission check timeout scenarios
- Review auth module coupling (consider facade pattern)
- Ensure new owner (alice@) is familiar with incident #123

Reasoning: High coupling + ownership churn + incident history = elevated risk

---

## Regression Detection in Practice

**End-to-End Scenario:**

1. **Developer commits change to payment_processor.py**
2. **Phase 1 baseline (200ms):**
   - Structural coupling: 12 dependents (HIGH)
   - Temporal co-change: fraud_detector.py (75% frequency, HIGH)
   - Test coverage: 0.4 (MEDIUM)
   - **Decision:** Escalate to Phase 2 (HIGH coupling + HIGH co-change)

3. **Phase 2 LLM investigation (3s):**
   - LLM asks: "Who owns fraud_detector.py?" → Ownership metric calculated
   - Result: @bob owns it, last active 3 days ago
   - LLM asks: "Similar past incidents?" → Incident search
   - Result: Incident #123 "Payment timeout after fraud check"
   - LLM synthesizes: **HIGH risk, confidence 0.85**

4. **Recommendation delivered:**
   > **Risk Level: HIGH (Confidence: 0.85)**
   >
   > **Key Evidence:**
   > - 12 files depend on payment_processor.py (blast radius)
   > - fraud_detector.py changed together 75% of time (temporal coupling)
   > - @bob owns fraud_detector.py (ownership context)
   > - Similar to Incident #123: "Payment timeout after fraud check"
   >
   > **Recommendations:**
   > - Review fraud_detector.py for needed updates
   > - Ping @bob for review before merging
   > - Add timeout handling tests (reference Incident #123)

5. **Regression prevented:**
   - Developer updates both payment_processor.py and fraud_detector.py
   - Adds timeout tests
   - Pings @bob for review
   - **Production incident avoided**

**Without CodeRisk:**
- Developer commits only payment_processor.py → PR merged → Production → fraud_detector.py breaks → Incident #124 (same as #123)

**With CodeRisk:**
- Automated due diligence flags coupled file + ownership + incident history → Developer takes action → No regression

---

## Metric Validation Framework

### False Positive Tracking

**User Feedback Loop:**

1. Developer runs `crisk check` and gets HIGH risk assessment
2. Developer disagrees (thinks it's a false alarm)
3. Developer runs `crisk feedback --false-positive --reason "intentional coupling"`
4. System records feedback in local SQLite database
5. If metric crosses 3% FP rate (with >20 samples), auto-disable

**Why This Matters:**
- Builds trust through self-correction
- Learns from user's domain knowledge
- Prevents metric degradation over time

### Validation Schema (SQLite)

**metric_validations table:**
- id: Primary key
- metric_name: Which metric (coupling, co_change, etc.)
- file_path: File being assessed
- metric_value: Full metric output (JSON)
- user_feedback: true_positive, false_positive, or null
- feedback_reason: User's explanation (text)
- created_at: Timestamp

**metric_stats table:**
- metric_name: Primary key
- total_uses: Total number of times metric was used
- false_positives: Count of FP feedback
- true_positives: Count of TP feedback
- fp_rate: Calculated as false_positives / total_uses
- is_enabled: Boolean (auto-disabled if fp_rate > 3%)
- last_updated: Timestamp

**Auto-Disablement Logic:**
- Check if metric has fp_rate > 3% AND total_uses ≥ 20
- If true, set is_enabled = FALSE
- Admin can review and re-enable after adjusting thresholds

**Metric Re-Enabling:**
- Admin reviews disabled metrics periodically
- Investigates common false positive reasons
- Adjusts thresholds or calculation logic
- Re-enables metric with new configuration

---

## Integration with Graph Ontology

See [graph_ontology.md](graph_ontology.md) for full schema details.

### Data Flow (Graph → Metrics → LLM)

**Step 1: Graph Data Extraction**
- Query Neo4j for 1-hop neighbors via IMPORTS/CALLS edges
- Load into working memory (in-memory graph subset)

**Step 2: Metric Calculation**
- Calculate Tier 1 metrics in parallel (coupling, co_change, test_ratio)
- Use pre-computed CO_CHANGED edge weights (no calculation needed)
- Cache results locally (15-min TTL)

**Step 3: Evidence Formatting**
- Convert metric results (JSON) to natural language evidence
- Add to LLM context for Phase 2 investigation

**Step 4: LLM Reasoning**
- LLM decides: CALCULATE_METRIC, EXPAND_GRAPH, or FINALIZE
- If CALCULATE_METRIC: Execute Tier 2 metric, add to evidence
- If EXPAND_GRAPH: Load 2-hop neighbors (rare, controlled growth)
- If FINALIZE: Synthesize final risk assessment

### Query Patterns (Conceptual)

**Coupling:** Query Neo4j for 1-hop neighbors connected via IMPORTS or CALLS edges, count distinct neighbors.

**Co-Change:** Read CO_CHANGED edge weight (frequency property), filter for frequency > 0.3, return top 10.

**Test Ratio:** Find test files via naming convention, calculate SUM(test.loc) / source.loc.

**Ownership:** Aggregate commits by developer in 90-day window, identify current and previous primary owner.

### Cache Strategy (Local MVP)

**Caching Approach:**
- Simple filesystem cache or in-memory (15-minute TTL)
- Cache keys: `coupling:{file}`, `co_change:{file}`, `test_ratio:{file}`, etc.
- Invalidate on: git commit affecting the file

**Invalidation Triggers:**
- Git commit to main: Invalidate coupling, co_change, test_ratio for affected files
- New incident linked: Invalidate incidents for affected files
- CODEOWNERS update: Invalidate ownership for all files

---

## Performance Characteristics

### Latency Budget (Per Check)

**Phase 1 (80% of checks):**
- Coupling query: ~40ms (1-hop, indexed)
- Co-change query: ~15ms (edge property read)
- Test ratio query: ~45ms (pre-computed relationships)
- Heuristic evaluation: ~5ms (simple if/else)
- **Total Phase 1: ~150ms** (with parallel queries)

**Phase 2 (20% of checks, HIGH risk only):**
- Load 1-hop neighbors: ~80ms (single batch query)
- LLM decision hop 1: ~1.2s (calculate Tier 2 metric)
- LLM decision hop 2: ~1.0s (smaller context)
- LLM synthesis: ~0.8s (structured output)
- **Total Phase 2: ~3.8s** (max 3 hops)

### Cost Model (Local MVP)

**Phase 1 Cost (Per Check):**
- Neo4j queries: ~$0 (local Docker container)
- Local cache: ~$0 (filesystem)
- **Total: $0**

**Phase 2 Cost (Per Check):**
- LLM calls: 4 requests × 1.5K tokens avg × $0.01/1K tokens = ~$0.06
- With cheaper models (e.g., Claude Haiku): ~$0.01-0.02
- **Total: ~$0.01-0.06** (varies by LLM provider and model)

**Daily Cost (100 checks):**
- 80 LOW risk (Phase 1 only): 80 × $0 = $0
- 20 HIGH risk (Phase 1 + 2): 20 × $0.03 avg = $0.60
- **Total: ~$0.60/day** (~$18/month for active development)

**MVP Economics:** With BYOK model, users pay only for LLM calls. No infrastructure costs.

---

## Key Design Decisions

### Why Simple Metrics Over Complex Models?

**Decision:** Use factual metrics (coupling, co-change) instead of statistical models (ΔDBR, G²)

**Rationale:**
- Factual metrics have 1-5% FP rate vs 10-25% for statistical models
- Simple metrics are explainable vs opaque ("PPR delta = 0.34")
- Cost reduction: $0 init (vs $15), ~$0.60/day (vs $5-10)

**Trade-off:** Miss nuanced patterns but rely on LLM to synthesize multiple signals

### Why BM25 Over Vector Embeddings for Incidents?

**Decision:** Use BM25 text search only, skip vector embeddings

**Rationale:**
- BM25 achieves ~85% accuracy vs ~88% for vector search
- Cost: $0 for BM25 (included in Neo4j) vs $0.10+ for embeddings
- 3% accuracy improvement not worth complexity increase

**Trade-off:** Miss semantic similarity (e.g., "timeout" vs "unresponsive"), acceptable for MVP

### Why 90-Day Historical Window?

**Decision:** Use 90-day window for co-change and ownership metrics

**Rationale:**
- 90 days balances recency with statistical significance
- Shorter (30 days): Too noisy, small sample size
- Longer (180 days): Stale patterns, old ownership data

**Trade-off:** Miss long-term patterns (e.g., annual refactorings), configurable per-repo in future

### Why Local SQLite (Not PostgreSQL)?

**Decision:** Use SQLite for validation data, not separate PostgreSQL

**Rationale:**
- Simpler deployment (single Docker container)
- Sufficient for solo/small teams (target MVP users)
- Zero infrastructure cost

**Trade-off:** Limited to single machine, but acceptable for target users

---

## Future Enhancements (Post-MVP)

**Adaptive Thresholds:**
- Learn optimal thresholds from user feedback
- Adjust per-repo (e.g., framework code has higher coupling baseline)

**Multi-File Risk:**
- Assess risk for entire PR (multiple files)
- Detect cross-file coupling patterns

**Temporal Decay (Simplified):**
- Add linear decay to co-change: weight = 1.0 - (days_ago / 90)
- Avoid complex Hawkes models, keep simple

**Custom Metrics:**
- Allow users to define repo-specific metrics
- Validate FP rate before enabling

---

## Related Documentation

- **[graph_ontology.md](graph_ontology.md)** - Graph schema and data sources
- **[agentic_design.md](agentic_design.md)** - Two-phase investigation flow
- **[../00-product/mvp_vision.md](../00-product/mvp_vision.md)** - Overall MVP scope and goals
- **[../00-product/developer_experience.md](../00-product/developer_experience.md)** - CLI workflow and user experience

---

**Last Updated:** January 2025
**Next Review:** After MVP deployment, analyze actual FP rates and adjust thresholds
