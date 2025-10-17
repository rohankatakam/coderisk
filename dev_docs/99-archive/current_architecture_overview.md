# Current CodeRisk Architecture (January 2025)

## Executive Summary
CodeRisk runs as a local-first CLI that orchestrates Cognee-backed graph, vector, and relational stores to evaluate regression risk. The CLI provisions Cognee datasets during `crisk init`, then reuses them during `crisk check` to execute an advanced risk engine that fuses statistical detectors with multi-modal search evidence (coderisk/coderisk/cli/main.py:200, coderisk/coderisk/cli/main.py:697). Core execution depends on Cognee’s Kuzu graph, LanceDB vector store, and SQLite metadata databases configured for local development by default (coderisk/coderisk/core/config.py:47, coderisk/coderisk/core/config.py:81-98). OpenAI-powered operations are invoked through Cognee without an internal budget governor, so API spend is gated only by ad-hoc retries (coderisk/coderisk/core/retry_handler.py:17-107).

## Developer Entry Points
- `crisk init` validates git state, collects API credentials, and builds the Cognee dataset by instantiating either the base `RiskEngine` or the search-enhanced variant (coderisk/coderisk/cli/main.py:697-742).
- `crisk check` performs a lightweight initialization, loads the chosen risk engine, and executes a worktree assessment while streaming progress to the console (coderisk/coderisk/cli/main.py:200-338).
- The CLI ensures repositories are pre-initialized (`.coderisk/initialized`) before allowing checks and surfaces the log file that captures full diagnostics (coderisk/coderisk/cli/main.py:207-262).

## Risk Engines and Scoring
- `RiskEngine` is the baseline assessor: it always wires in `CogneeCodeAnalyzer`, advanced mathematical signal calculators, regression scaling, and a conformal scoring engine (coderisk/coderisk/core/risk_engine.py:63-195). Assessments derive git diff context and compute multiple risk signals, then aggregate them into `RiskAssessment` objects (coderisk/coderisk/core/risk_engine.py:133-228).
- `SearchEnhancedRiskEngine` subclasses the base engine to replace signal computation with a `SearchEnhancedRiskSignalEngine`, a Cognee `CogneeKnowledgeProcessor`, and an assessment semaphore that serializes requests to avoid SQLite contention (coderisk/coderisk/core/search_enhanced_risk_engine.py:39-199).
- For every change, the search-enhanced engine converts git context to an `EnhancedRiskContext`, invokes search-enhanced signal calculators, optionally merges pipeline outputs, and produces enriched evidence and recommendations (coderisk/coderisk/core/search_enhanced_risk_engine.py:204-320, coderisk/coderisk/core/search_enhanced_risk_engine.py:487-555).
- Both engines rely on the regression scaling model and scoring engine to adjust raw signals for team, codebase, and change velocity factors before emitting final tiers (coderisk/coderisk/core/risk_engine.py:188-200).

## Knowledge & Search Infrastructure
- `CogneeCodeAnalyzer` orchestrates dataset construction: it runs Cognee’s code graph pipeline, adds repository data to a dataset, applies the CodeRisk ontology, and validates the dataset using Cognee searches (coderisk/coderisk/core/cognee_integration.py:20-190).
- `CogneeKnowledgeProcessor` sets up custom Cognee pipelines, ingests commits, file changes, developers, incidents, and security datapoints, then triggers code graph extraction and graph enrichment phases (coderisk/coderisk/ingestion/cognee_processor.py:35-141).
- `AdvancedSearchAggregator` maps each risk calculation to a specific combination of Cognee `SearchType`s, runs those searches sequentially through an internal semaphore, fuses evidence with reciprocal-rank fusion (RRF), and tracks performance metrics (coderisk/coderisk/queries/advanced_search_aggregator.py:156-246). It falls back gracefully when the Kuzu graph adapter cannot be reached.
- `SearchEnhancedRiskSignalEngine` consumes the aggregator to produce signal-level scores and textual evidence for ΔDBR, HDCC, incident adjacency, and other detectors while capturing the search plan used (coderisk/coderisk/calculations/search_enhanced_signals.py:1-198).
- `CypherQueryEngine` funnels raw graph lookups through Cognee’s CYPHER interface wrapped in a safe connection context to work around Kuzu’s single-writer limitations (coderisk/coderisk/queries/cypher_queries.py:1-116).

## Data Sources & Ingestion
- `GitHistoryExtractor` collects a 90-day commit window with PyDriller, annotates change metadata (security, migrations, config changes), and produces structured DataPoints for downstream ingestion (coderisk/coderisk/ingestion/git_history_extractor.py:1-200).
- DataPoint schemas in `ingestion/data_models.py` encode commits, file changes, developers, incidents, and security findings for Cognee’s vector and graph layers.
- Static analysis helpers (`analysis/static_analyzer.py`) and rule-based detectors (`detectors/*`) remain available for complementary signals, though the search-enhanced path increasingly centralizes risk scoring.

## Storage & Configuration
- `CodeRiskConfig` loads environment variables and defaults to local storage: SQLite for relational persistence, LanceDB for vectors, and Kuzu for graph queries (coderisk/coderisk/core/config.py:47-98). Selecting PostgreSQL or alternative providers requires explicit overrides.
- `DatabaseConnectionManager` serializes Cognee/Kuzu access by toggling `COGNEE_KUZU_SINGLE_CONNECTION` and tracking active handles, mitigating lock faults at the cost of throughput (coderisk/coderisk/core/database_manager.py:18-115).
- `CodeRiskDatabase` offers a PostgreSQL/SQLite abstraction for storing assessments and processing status, though most runtime state currently lives in Cognee-managed stores (coderisk/coderisk/core/sql_database.py:250-409).

## External Dependencies & Controls
- OpenAI API keys and GitHub tokens are required for initialization; configuration helpers prompt for and persist credentials under the user’s config directory (coderisk/coderisk/config.py:62-112).
- `RetryHandler` supplies exponential backoff for OpenAI and GitHub calls but does not enforce hard budget ceilings or daily quotas (coderisk/coderisk/core/retry_handler.py:17-207).
- Observability integrates with Cognee’s telemetry to track search execution, but there is no in-process accounting of API spend or rate-limit states beyond retries.

## Observed Constraints & Bottlenecks
- Search operations are serialized to sidestep SQLite and Kuzu concurrency limits, constraining throughput for large assessments (coderisk/coderisk/core/search_enhanced_risk_engine.py:189-201, coderisk/coderisk/queries/advanced_search_aggregator.py:183-231).
- Every `crisk check` depends on Cognee searches that can trigger multiple OpenAI embedding or completion calls per signal without tier-based throttling, leading to unpredictable costs.
- Dataset creation and re-ingestion remain heavyweight operations tied to Cognee pipelines, making `crisk init` slow on large repositories.
- There is no caching layer for search results or signal computations across assessments, so repeated checks often replay the same external calls.
- Graceful degradation tiers described in the financial architecture doc are not yet represented in code; all assessments attempt the deepest search-enhanced path.

This snapshot reflects the current architecture that future work must optimize for predictable cost, constrained OpenAI usage, and improved local performance while maintaining Cognee-backed intelligence.
