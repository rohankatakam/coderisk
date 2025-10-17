# Cost-Efficient Risk Math & Search Plan

This note describes a heavy `crisk init` + lightweight `crisk check` strategy that maximizes reuse of graph/embedding work while honoring the $60/repo budget cap. It supplements `ideal_architecture.md`, `cognee_embedding_and_llm_strategy.md`, and `financial_constraints_architecture.md`.

## 1. Lifecycle Overview

| Stage | What happens | Cost profile |
| --- | --- | --- |
| `crisk init` (day 0) | Full repository ingest → chunking → embeddings (using low-cost Batch pricing) → graph build → memify enrichment → risk sketch precomputation | Expensive but one-time (budget up to $15) |
| Continuous ingestion (new commits/issues) | Incremental updates: add/cognify for deltas, daily memify pass, nightly Batch embedding refresh | Spread across daily $2 allowance |
| `crisk check` | Fast, budget-aware risk assessment using cached data + selective API calls | ≤ $0.04 per check |

## 2. Heavy Init Strategy (High Cost Allowed)

1. **Add & Cognify Everything Once**
   - Walk the repository with `GitHistoryExtractor` to capture 90-day history, ownership, incidents.
   - Use Cognee `.add` followed by `.cognify` to build equivalent DataPoints.

2. **Compute Embeddings in Batch Mode**
   - Partition the corpus (unchanged hot files, tests, config) and call the provider’s Batch API.
   - Store embeddings in LanceDB collections keyed by file hash + chunk index; annotate with TTL (14–30 days).
   - Compute derived embeddings (function summaries, incident narratives) while rates are cheap.

3. **Graph Construction & Memify Enrichment**
   - Run `run_code_graph_pipeline` to extract IMPORTS, CALLS, CO_CHANGED edges (Kuzu).
   - Apply memify pipelines to encode coding rules, incident patterns, and cross-file associations.

4. **Precompute Risk Sketches**
   - Offline algorithms produce cached values consumed by `crisk check`:
     * **ΔDBR surrogate**: approximate Personalized PageRank deltas per file using cached graph.
     * **HDCC metrics**: two-timescale coupling stored in SQLite.
     * **Incident adjacency vectors**: store top-K incidents per file using fused results.
     * **Ownership/test metrics**: commit counts, test coverage ratios, etc.

5. **Persist Forecasted Budgets**
   - Log the tokens and dollars burned; ensure remaining monthly budget accommodates daily updates and checks.

## 3. Incremental Updates

1. **Lightweight ingestion** triggered by commits/PRs/issues.
2. **Graph/embedding update logic**
   - Embeddings: only for changed files (daily Batch job). Others reuse cached vectors.
   - Graph edges: update via incremental pipeline (follows same 2-hop cap assumptions).
3. **Memify** runs daily, focusing on changed subgraphs to refresh derived facts.
4. **Risk sketch updates** adjust only the impacted files’ cached metrics.

## 4. `crisk check` Algorithm

1. **Budget Gate**
   - `APIBudgetManager` verifies remaining daily budget; otherwise, run Level 1 only.

2. **Local Risk Core (always)**
   - Load cached risk sketches: ΔDBR surrogate, HDCC scores, incident adjacency cache, ownership/test metrics.
   - Compute raw signal scores; escalate to Level 2 only if the result is high-risk or stale.

3. **Selective Refresh (optional)**
   - If cached data older than threshold or is missing, run targeted queries:
     * ≤ 2 graph queries (Kuzu) per critical file for ΔDBR and HDCC refinement.
     * ≤ 1 search (vector + BM25 RRF) for incident adjacency updates.
   - No more than 3 API calls; rely on cached embeddings first, falling back to Batch-stored vectors.

4. **LLM Inference**
   - Use local Ollama to summarize evidence; call external LLM only for high-risk results after budget approval.

5. **Output**
   - Combine cached metrics and refreshed signals into final `RiskAssessment`. Log which pieces were reused vs. recomputed.

## 5. Mathematical Optimizations

1. **Graph Analytics**
   - Precompute approximate personalized PageRank and betweenness centrality using power iteration offline; store vectors per file.
   - For checks, update PPR contributions only within the neighbor set of changed files (≤2 hops).

2. **Co-Change Modeling**
   - Maintain sliding-window statistics (fast/slow decay) in SQLite; updates only require the latest commit info.
   - HDCC score for a file pair = `w_fast * recent_count + w_slow * historic_count` with weights set offline.

3. **Incident Similarity**
   - Keep per-file incident templates (BM25 + vector centroid) precomputed.
   - At check time, compare diff summary to top-K incidents using cached embeddings.

4. **Ownership/Test Metrics**
   - Pre-load per-file developer scores, churn rates, and coverage delta; update incrementally.

5. **Scoring Engine**
   - The conformal regression model can run offline; at check time, pass updated features for final risk score.

## 6. Storage Discussion: SQLite vs. Postgres

| Criterion | SQLite | Postgres |
| --- | --- | --- |
| Setup | Zero install (default) | Requires local server (Docker) |
| Concurrency | Single writer; fine for CLI usage | Handles concurrent checks/background tasks |
| Analytics | Basic SQL | Advanced window functions, materialized views |
| Recommended | ✅ For single-user local workflows | ✅ If we run background jobs + telemetry |

**Recommendation:** Ship SQLite by default; expose a `--profile postgres` for teams that want richer analytics and multi-user concurrency. The math pipeline works either way.

## 7. Provider Budget Model

- **Init:** Provision up to $15 (OpenAI/Gemini Batch) per repo.
- **Daily Refresh:** Cap at $2 (embeddings + targeted LLM calls).
- **Check:** ≤ $0.04 per check; bypass external providers otherwise.
- Mix local (Ollama/Fastembed) and external calls to stretch the budget.
- Monitor actual spend and adjust chunking/caching thresholds accordingly.

## 8. Fairness vs. `search_strategy.md`

- Heavy precomputation recovers most signals defined in `search_strategy.md` without live multi-modal searches each time.
- Selective refresh ensures high-risk situations still re-run the deep analysis within budget.
- Level 3 preview (full fusion) can be exposed to premium plans without rewriting code.

## 9. Next Steps

1. Implement `RiskSketch` cache schema (SQLite/Postgres) with versioned features.
2. Build daily Batch job that re-embeds only changed chunks.
3. Integrate memify-based enrichment to maintain coding rules and incident relationships.
4. Wire Level 2 selective refresh into `SearchEnhancedRiskEngine` with budget checks.
5. Provide instrumentation dashboards to verify spend vs. latency.

This strategy delivers a single, uniform risk pipeline that feels as powerful as the original vision, while keeping day-to-day checks well within our financial limits.
