# Ideal Architecture for CodeRisk Under $60/Repo Budget and ICP Expectations

## 1. Can We Deliver the Desired Developer Experience?
- **ICP Expectations (dev_experience.md):** Instant, trustworthy risk feedback in local workflows (Ben); authoritative architectural insight for reviewers (Clara); offline-friendly guardrail for solo developers (David).
- **Financial Guardrails (financial_constraints_architecture.md):** Small-team budget target $50–$100/month; we choose $60/repo/month as the upper bound.
- **Functional Requirements (functional_business_requirements.md):** <2s P95 latency, 7 core signals + 9 micro-detectors, coverage of top languages, optional deep AI insights.

**Conclusion:** Yes, the ideal developer experience is achievable within the $60/repo cap if we restructure the system around local-first heuristics, selective embeddings, and tightly budgeted LLM usage. The current architecture (current_architecture_overview.md) does **not** meet these constraints because it attempts full search-enhanced analysis on every check with uncontrolled API calls and serialized storage, leading to high latency and runaway costs.

## 2. Target Architecture Overview

### 2.1 Architecture Layers
1. **Level 1 Engine (Zero-Cost Local Analysis) – Default Path**
   - Runs within the CLI, leveraging Git history, static analysis, and cached metrics.
   - Sources: `analysis/static_analyzer.py`, detectors under `detectors/`, Git diff parsing in `RiskEngine`.
   - Maintains a local SQLite store of risk summaries and trends.
   - Delivers <1s responses without Cognee or API dependencies.

2. **Level 2 Engine (Budgeted Search & Embeddings) – Optional Enhancements**
   - Uses Cognee search only when the budget manager approves (<3 API calls per check).
   - Embeddings produced in scheduled batch jobs; cached in LanceDB with TTL.
   - Graph traversal (Kuzu) limited to cached subgraphs; no full re-ingestion during checks.
   - Outputs enriched insights (e.g., ΔDBR) within 2s while respecting daily token caps.

3. **Level 3 Engine (Premium / Opt-In)**
   - Full search-enhanced capability (Graph + Temporal + LLM synthesis) triggered only by explicit opt-in or pay-as-you-go usage.
   - Uses provider mix (OpenAI Priority, Gemini Live) only when budgets allow or enterprise profile is active.

### 2.2 Data Infrastructure
- **Default Storage:** SQLite (metadata), LanceDB (vectors), Kuzu (graph) – all local.
- **Selective Embeddings:** Only index high-signal DataPoints (summaries, hotspots). Graph construction does not require embeddings.
- **Batch Scheduler:** Runs at most once daily to update embeddings using low-cost tiers (OpenAI Batch or Gemini Flash Batch) or local providers (Ollama/Fastembed).
- **Cache Management:** LanceDB entries tagged with source, cost, and TTL; expire unused vectors to control disk usage.

### 2.3 LLM Usage
- LLM inference restricted to `crisk check` pathway.
- Hard cap: ≤10 LLM calls per assessment; default provider = local Ollama; fall back to OpenAI/Gemini only when budget allows.
- All pipeline tasks log token estimates via `APIBudgetManager` before execution; failure to authorize = skip to Level 1 result.

### 2.4 Budget Governance
- Implement `APIBudgetManager` providing per-day and per-month ceilings aligned with $60 cap.
- Integrate with `RiskEngine`, `SearchEnhancedRiskEngine`, and `AdvancedSearchAggregator`.
- Record actual spend in SQLite; export daily summaries for auditing.

### 2.5 Developer Experience Enhancements
- CI/CLI responses clearly state which level ran and why (e.g., “Level 2 skipped (budget exhausted)”).
- Provide local dashboards showing budget usage, cache hit rate, and risk trends.
- Offline mode remains fully functional (Level 1).

## 3. Alignment with Requirements
| Requirement | Meets with Ideal Architecture? | Notes |
| --- | --- | --- |
| <2s risk assessments | ✅ Level 1 <1s; Level 2 <2s with cache; Level 3 opt-in |
| 7 core signals + 9 detectors | ✅ All Level 1 or cached/precomputed outputs |
| Multi-language support | ✅ Leveraging existing AST/analysis modules |
| Budget ≤$60/month | ✅ Batch embeddings + capped live calls (documented in llm_pricing_reference.md) |
| Instant feedback for Ben/David | ✅ Level 1 always available offline |
| Deep insights for Clara | ✅ Level 2/3 when budget allows |

## 4. Migration Plan from Current Architecture
1. **Phase 1 (Weeks 1–2): Level 1 Guarantee**
   - Refactor `RiskEngine` to run Level 1 pipeline independent of Cognee.
   - Persist Level 1 outputs to SQLite; expose CLI flags to force Level 1 only.
   - Add messaging around budget status.

2. **Phase 2 (Weeks 3–5): Budget Manager + Caching**
   - Implement `APIBudgetManager` and integrate with `RetryHandler`.
   - Schedule batch embedding jobs; tag LanceDB vectors with TTL.
   - Modify Cognee ingestion to process only deltas and index limited fields.

3. **Phase 3 (Weeks 6–8): Optional Level 3 Profiles**
   - Introduce enterprise profile toggles for Postgres/ChromeDB/Neo4j.
   - Add dashboards and configuration UI for spend monitoring.

## 5. Gaps in Current Architecture
- **Uncontrolled API Usage:** Current `SearchEnhancedRiskEngine` always attempts search-enhanced signals without cost checks (current_architecture_overview.md).
- **Serialization Bottleneck:** Kuzu/SQLite locking serialized check operations; need selective caching and asynchronous batch jobs.
- **Missing Level 1 Fallback:** No code-level guarantee of zero-cost assessments.
- **Absence of Budget Manager:** `RetryHandler` lacks token awareness; no mechanism to enforce $0.04/check limit.
- **LLM Invocation:** Sociates directly with Cognee search; needs explicit call caps and provider selection.

## 6. Feasibility Summary
- **Possible?** Yes, with disciplined caching, local-first computation, and strict budget enforcement, the ideal developer experience is feasible.
- **Current Architecture Suitability?** No. In its present form, it overspends, lacks necessary fallbacks, and suffers from storage contention.
- **Key Actions:** Implement Level 1-first design, integrate `APIBudgetManager`, constrain embeddings, and manage provider usage as outlined.

This ideal architecture provides a roadmap to deliver meaningful, fast risk assessments (meeting our ICP expectations) while honoring the $60/repo financial constraint.
