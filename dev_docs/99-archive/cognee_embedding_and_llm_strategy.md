# Cognee Embedding & LLM Strategy for CodeRisk

This note explains how Cognee decides when to create embeddings, how it invokes LLMs, and how we can constrain those behaviors so CodeRisk scales to large codebases under strict API budgets.

## 1. Pipeline Overview

Cognee processes repository data in two main stages:

1. **`add` (Ingestion)** – Normalizes raw inputs (files, directories, text) into a dataset. No embeddings or LLM calls occur here; content is stored with metadata (see `CogneeKnowledgeProcessor.ingest_repository_data` in `ingestion/cognee_processor.py:97-141`).
2. **`cognify` / task pipelines** – Converts stored items into DataPoints, runs embeddings for the fields marked as searchable, and builds the knowledge graph (code graph, temporal edges, etc.). `CogneeCodeAnalyzer` triggers this stage via `run_code_graph_pipeline` (`core/cognee_integration.py:60-110`).

## 2. What Triggers Embeddings

Cognee’s embedding work is driven by DataPoint metadata:

- Every DataPoint has `metadata.index_fields` (see the DataPoints doc). When `cognify` runs, Cognee embeds only those fields and stores vectors in the configured vector DB (LanceDB by default).
- Built-in types (`DocumentChunk`, `TextSummary`, `Entity`, etc.) ship with sensible defaults, but we can override them when defining custom DataPoints to limit what gets embedded.
- The code graph pipeline that builds AST-level relationships relies on structural parsing (Kuzu graph edges) and does **not** require embeddings.

**Practical levers for large codebases**
- Limit indexing to high-signal fields (e.g., file summaries, function docstrings) instead of raw file bodies.
- Chunk only the top risk hotspots (use commit heuristics in `GitHistoryExtractor` or our Level 1 metrics to decide which files to embed).
- Skip embeddings entirely for Level 1 assessments; rely on local heuristics and previously cached vectors.

## 3. Where LLM Inference Occurs

Cognee itself defers to the configured LLM provider whenever a task or search mode needs generative reasoning:

- **Search operations** – `AdvancedSearchAggregator` calls `cognee.search` with `SearchType` values. Certain modes (e.g., `FEELING_LUCKY`, some `CHUNKS` fusion strategies) may compose summarized answers using the LLM. Pure graph lookups (`GRAPH_COMPLETION`, `CYPHER`) avoid LLM inference.
- **Summarization tasks** – Pipelines can generate summaries (`TextSummary`, `CodeSummary`) or insights, which involve LLM calls. In CodeRisk, we only need these in Level 2/3 analyses.
- **Interactive questions** – Not part of CodeRisk, but relevant if we expose Cognee dashboards.

To keep costs down, we should confine LLM usage to the `crisk check` path and cap the number of calls (e.g., ≤10 per assessment via the upcoming `APIBudgetManager`).

## 4. Provider Configuration & Local Options

Cognee lets us choose different providers for embeddings and LLMs via environment variables (`coderisk/coderisk/core/config.py:81-98`). Relevant combinations include:

| Use Case | Provider | Key Vars |
| --- | --- | --- |
| **OpenAI** (default) | `LLM_PROVIDER="openai"` / `EMBEDDING_PROVIDER="openai"` | `OPENAI_API_KEY`, `EMBEDDING_MODEL` (e.g., `openai/text-embedding-3-large`) |
| **Gemini** | `LLM_PROVIDER="gemini"`, `EMBEDDING_PROVIDER="gemini"` | `GEMINI_API_KEY`, `EMBEDDING_MODEL="gemini/text-embedding-004"` (dim=768) |
| **Ollama local LLM** | `LLM_PROVIDER="ollama"`, `LLM_MODEL="llama3.1:8b"`, `LLM_ENDPOINT="http://localhost:11434/v1"`, `LLM_API_KEY="ollama"` | Requires `HUGGINGFACE_TOKENIZER` until upstream fix |
| **Ollama local embeddings** | `EMBEDDING_PROVIDER="ollama"`, `EMBEDDING_MODEL="avr/sfr-embedding-mistral:latest"`, `EMBEDDING_ENDPOINT="http://localhost:11434/api/embeddings"`, `HUGGINGFACE_TOKENIZER="Salesforce/SFR-Embedding-Mistral"` |
| **Fastembed (CPU)** | `EMBEDDING_PROVIDER="fastembed"`, `EMBEDDING_MODEL="sentence-transformers/all-MiniLM-L6-v2"`, `EMBEDDING_DIMENSIONS="384"` | No GPU required; Python < 3.13
| **Custom endpoints** | `LLM_PROVIDER="custom"` / `EMBEDDING_PROVIDER="custom"` | Supply `LLM_ENDPOINT`, `EMBEDDING_ENDPOINT`, API keys; must be OpenAI-compatible |

**Important:** Mixing Ollama LLM with OpenAI embeddings can raise `NoDataError`; use the same provider across both when possible.

## 5. Limiting Embedding Volume

1. **Index Fields Audit** – Define DataPoints so that only critical fields (e.g., summary text, dependency metadata) have `index_fields`. Omit raw file bodies for large repos when vector search isn’t required.
2. **Chunk Selection** – Control chunking in pipelines; process only recently touched files or files flagged by Level 1 heuristics.
3. **TTL & Cache Eviction** – Use LanceDB metadata to expire embeddings after a rolling window; re-embed only when necessary.
4. **Local Providers** – For evaluations or air-gapped environments, switch to Ollama/Fastembed so API calls drop to zero (in return for local CPU/GPU cost).

## 6. Limiting LLM Calls

- **Assessment Budget** – Integrate call counters in `SearchEnhancedRiskEngine` to enforce ≤10 LLM calls per `crisk check`; fallback to cached or heuristic results when reaching the limit.
- **Search Strategy Selection** – Prefer `GRAPH_COMPLETION`, `CYPHER`, and `CODE` searches that rely on structured data and embeddings rather than on-demand LLM synthesis.
- **Local LLM Usage** – Route Level 2/3 summaries through Ollama or another custom endpoint when network/API budgets are tight.
- **Graceful Degradation** – When budgets are exhausted mid-check, revert to Level 1 outputs and clearly annotate that AI enrichment was skipped.

## 7. Graph Construction Without Embeddings

- The code graph (imports, call graph, co-change edges) is produced by `run_code_graph_pipeline`, `CogneeKnowledgeProcessor`, and our ingestion tasks. These components parse source code, git history, and incidents to generate nodes/edges without requiring vector search.
- Embeddings enhance semantic similarity (e.g., “find files similar to X”), but they are not mandatory for building the structural graph. For massive codebases, we can rely on the graph-only approach plus Level 1 heuristics and selectively enable embeddings for critical modules.

## 8. Recommendations for Large Codebases

1. **Default to Local Heuristics (Level 1)** – Deliver meaningful risk insight without touching external APIs.
2. **Selective Embedding** – Embed only the top N risky files per day or modules recently changed; skip static libraries.
3. **Hybrid Provider Strategy** – Use local Fastembed/Ollama for bulk processing, reserve OpenAI/Gemini Batch for high-precision signals.
4. **LLM Budget Guardrails** – Embed a per-check token budget (e.g., $0.04) and max call count (≤10) enforced centrally.
5. **Monitor Storage** – Track LanceDB size and SQLite/Kuzu performance; promote repositories to a scalable backend only when local storage becomes a bottleneck.

By understanding and controlling how Cognee decides to embed and invoke LLMs, we can keep CodeRisk affordable, responsive, and scalable, even across very large repositories.
