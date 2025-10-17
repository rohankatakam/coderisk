# Architecture Pathways Under a $60/Repository Monthly Budget Cap

## 1. Pricing Constraint & ICP Rationale
- **ICP:** Hands-on developers with limited safety nets—Ben (junior on a fast-growing startup team), David (solo maintainer), and Clara’s small platform team before enterprise rollout.
- **Willingness to Pay:** From user stories, they adopt CodeRisk as a personal code-review safety net. Comparable tooling (linting/SAST) averages $30-50 per seat monthly. Financial guardrails (`financial_constraints_architecture.md`) target $50-100 for small teams.
- **Firm Constraint:** We will enforce **$60/month per repository** (≈$2/day) covering all OpenAI/Gemini usage plus local compute. This translates to:
  - ≤$15 per `crisk init` (Batch processing only).
  - ≤$0.04 per `crisk check` at 1,500 checks/month.
  - Mandatory Level 1 (zero-cost) fallback to ensure value even when budgets are depleted.

## 2. Provider Cost Envelope
Derived from `llm_pricing_reference.md` and the $60 cap:
- **Embeddings (60% of budget ≈ $36):** Use OpenAI `text-embedding-3-small` Batch ($0.01/M) or Gemini 2.5 Flash Batch ($0.15/M) for refresh; cap monthly embedding tokens at 3.6B for OpenAI Batch or 240M for Gemini.
- **LLM Completion (40% ≈ $24):** Prefer OpenAI `gpt-4o-mini` Batch ($0.075 input/$0.30 output per 1M) or Gemini 2.5 Flash standard ($0.30 input / $2.50 output). Live checks must stay within 160M input and 32M output tokens (OpenAI) or 80M/9.6M (Gemini) monthly.
- **Allocation Rules:**
  1. Batch jobs only during scheduled ingestion windows.
  2. Flex/Standard calls allowed only when per-day spend < $2.
  3. Priority tiers forbidden under the default plan.

## 3. Shift Plan from Current Architecture (`current_architecture_overview.md`)
### Path A – Zero-Cost Baseline (Week 1)
**Objective:** Guarantee every `crisk check` delivers a Level 1 assessment without any external API usage.
- **Actions:**
  - Embed Level 1 signal pipeline (complexity, churn, coverage, hotspot proximity, ownership stability) directly into `RiskEngine` (`core/risk_engine.py:170-228`).
  - Persist Level 1 outputs to SQLite (`core/sql_database.py:250-409`) for trend tracking without Cognee access.
  - Ensure CLI surfaces Level 1 findings with clear messaging when budgets block higher tiers (`cli/main.py:200-337`).
- **Resource Impact:** CPU-bound Git analytics, negligible token cost.
- **KPIs:** 100% of checks complete <1s offline; budget utilization unaffected.

### Path B – Budgeted Enhanced Intelligence (Weeks 2-4)
**Objective:** Layer API-driven insights only within the $60/month cap.
- **Budget Manager:** Implement `APIBudgetManager` used by `SearchEnhancedRiskEngine` and `AdvancedSearchAggregator` (`core/search_enhanced_risk_engine.py:204-320`, `queries/advanced_search_aggregator.py:156-246`). Track spend, rate limits, provider mix.
- **Token Accounting:**
  - Enrich `RetryHandler` (`core/retry_handler.py:17-207`) with token estimation per call and budget checks before retries.
  - Log provider/tier per query in LanceDB metadata to drive repeat-use caching (`core/config.py:81-98`).
- **Caching Strategy:**
  - Precompute embeddings via Batch (OpenAI or Gemini) during quiet hours; store vectors in LanceDB. Limit refresh to daily diff output from `GitHistoryExtractor` (`ingestion/git_history_extractor.py:29-200`).
  - Cache search results with 30-day TTL; fallback to cache when API unavailable.
- **Graceful Degradation:** Implement 3-tier logic from financial doc so Level 2 (≤3 API calls) and Level 3 (≤10 calls) execute only when `APIBudgetManager` authorizes spend.
- **KPIs:**
  - Per-check API usage ≤ $0.04 (log per assessment in SQLite).
  - Batch jobs consume ≤ $15/init (record tokens processed).
  - 80% cache hit rate for repeated queries.

### Path C – Controlled Parallelism & Optional Upgrade (Weeks 5-8)
**Objective:** Offer scalability while keeping base plan cost-neutral.
- **Default (Local) Mode:** Keep SQLite/LanceDB/Kuzu (single-writer) for costless infra. Serialized semaphore (`core/database_manager.py:18-170`) remains default.
- **Enterprise Profile (Opt-in):**
  - Provide documented profile to switch to Postgres/ChromeDB/Neo4j with cost warnings; only available for customers exceeding $60 cap and willing to pay infrastructure fees.
  - Throttle assessments to stay within negotiated plan budgets; remove throttle only when a higher-priced plan is purchased.
- **Live Cost Telemetry:**
  - For enterprise installs, expose dashboards showing token burn vs. plan limit to avoid surprise bills.
  - Auto-downgrade to Level 1 when spend exceeds contracted amount.
- **KPIs:** Zero change to base plan costs; enterprise add-ons billed separately with explicit consent.

## 4. Implementation Milestones
| Week | Milestone | Deliverables |
| --- | --- | --- |
| 1 | Path A complete | Offline Level 1 pipeline, CLI messaging, nightly budget report scaffold |
| 2 | Budget manager MVP | Token estimator, spend ledger in SQLite, fail-closed enforcement |
| 3 | Provider-aware caching | Batch ingestion scheduler, LanceDB cache tagging, retry guardrails |
| 4 | Graceful degradation GA | Level 2/3 gating, per-tier telemetry, budget exhaustion alerts |
| 5-6 | Enterprise profile beta | Config toggles, connection pooling updates, cost dashboards |
| 7-8 | Optimization & audit | Cache eviction policy, documentation, audit of per-check costs |

## 5. Success Metrics
- **Cost:** 95% of repositories stay at or below $60/month; no single check exceeds $0.04 API spend.
- **Performance:** Level 1 checks <1s; Level 2 checks <2s; Level 3 checks <5s while under budget.
- **Adoption:** Ben/David personas use Level 1 daily without API keys; Clara upgrades to budgeted Level 2/3 as needed.
- **Sustainability:** Batch embedding refresh jobs run at most once daily; LanceDB footprint stays <5 GB per repo on local machines.

By anchoring every architectural decision to the $60/month constraint, we protect affordability for our ICP while still offering optional intelligence when budgets permit. This roadmap transforms the current architecture into a predictable, cost-aware system without sacrificing developer trust.
