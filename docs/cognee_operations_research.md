# Cognee Operations Pipeline Research

## Overview

This document provides comprehensive research on Cognee's main operations pipeline based on official documentation and source code analysis, specifically for implementing CodeRisk's risk assessment system.

## Key Finding: Current Setup Issue

**CRITICAL ISSUE IDENTIFIED**: Our GitHub data ingestion is incomplete. We're using `cognee.add()` to store text data, but we're **NOT running `cognee.cognify()`** which is required to process the data into chunks and enable vector search.

This is why our chunk searches are failing with "DocumentChunk_text collection not found in vector database".

## Cognee Main Operations Pipeline

### 1. Add Operation

**Purpose**: Ingest raw data into Cognee storage

**Function Signature**:
```python
cognee.add(
    data: Union[BinaryIO, list[BinaryIO], str, list[str]],
    dataset_name: str = 'main_dataset',
    user: cognee.modules.users.models.User.User = None,
    node_set: Optional[List[str]] = None,
    vector_db_config: dict = None,
    graph_db_config: dict = None,
    dataset_id: Optional[uuid.UUID] = None,
    preferred_loaders: List[str] = None,
    incremental_loading: bool = True
)
```

**What it does**:
- Normalizes content into plain text
- Records into datasets with content hash deduplication
- **Does NOT create embeddings or graph structure yet**
- Supports: text, files, directories, S3 URIs
- Creates database records in SQLite

**Current CodeRisk Usage**:
```python
# ✅ We are doing this correctly
await cognee.add(
    commit_texts,
    dataset_name=self.dataset_name,
    node_set=["github", "commits", "coderisk"]
)
```

### 2. Cognify Operation

**Purpose**: Transform ingested data into searchable knowledge graph

**Function Signature**:
```python
cognee.cognify(
    datasets: Union[str, list[str], list[uuid.UUID]] = None,
    user: cognee.modules.users.models.User.User = None,
    graph_model: pydantic.main.BaseModel = <class 'cognee.shared.data_models.KnowledgeGraph'>,
    chunker=<class 'cognee.modules.chunking.TextChunker.TextChunker'>,
    chunk_size: int = None,
    ontology_file_path: Optional[str] = None,
    vector_db_config: dict = None,
    graph_db_config: dict = None,
    run_in_background: bool = False,
    incremental_loading: bool = True,
    custom_prompt: Optional[str] = None,
    temporal_cognify: bool = False
)
```

**Six-Step Pipeline**:
1. **Document Classification** - Convert to Document objects
2. **Permission Check** - Verify write access
3. **Chunk Extraction** - Split into DocumentChunks (creates DocumentChunk_text collection)
4. **Graph Extraction** - Extract entities/relationships with LLMs
5. **Text Summarization** - Generate summaries per chunk
6. **Data Point Addition** - Embed in vector store, persist in graph

**Current CodeRisk Issue**:
```python
# ❌ We are NOT running this step!
# This is why chunk searches fail
await cognee.cognify(
    datasets=[self.dataset_name],
    ontology_file_path=str(ontology_path) if ontology_path.exists() else None
)
```

### 3. Search Operation

**Purpose**: Query the processed knowledge graph

**Function Signature**:
```python
cognee.search(
    query_text: str,
    query_type: SearchType = SearchType.GRAPH_COMPLETION,
    user: Optional[User] = None,
    datasets: Union[list[str], str, NoneType] = None,
    dataset_ids: Union[list[uuid.UUID], uuid.UUID, NoneType] = None,
    system_prompt_path: str = 'answer_simple_question.txt',
    system_prompt: Optional[str] = None,
    top_k: int = 10,
    node_type: Optional[Type] = NodeSet,
    node_name: Optional[List[str]] = None,
    save_interaction: bool = False,
    last_k: Optional[int] = None,
    only_context: bool = False,
    use_combined_context: bool = False
) -> Union[List[SearchResult], CombinedSearchResult]
```

**Available Search Types**:
1. **CHUNKS** - Returns most similar text chunks via vector search
2. **CODE** - Code-specific search
3. **CODING_RULES** - Rules extraction search
4. **CYPHER** - Direct graph database queries
5. **FEEDBACK** - Feedback-based search
6. **FEELING_LUCKY** - Simplified search
7. **GRAPH_COMPLETION** - (Default) Combines vector + graph + LLM
8. **GRAPH_COMPLETION_CONTEXT_EXTENSION** - Extended context version
9. **GRAPH_COMPLETION_COT** - Chain-of-thought reasoning
10. **GRAPH_SUMMARY_COMPLETION** - Summary-based completion
11. **INSIGHTS** - Insight extraction
12. **NATURAL_LANGUAGE** - Natural language processing
13. **RAG_COMPLETION** - Pure vector RAG without graph
14. **SUMMARIES** - Vector search on TextSummary content
15. **TEMPORAL** - Time-aware search

**Correct Usage Pattern**:
```python
# ✅ This is the correct format
results = await cognee.search(
    query_type=SearchType.GRAPH_COMPLETION,
    query_text="Find commits by author with file changes",
    datasets=["test_github_omnara"]
)
```

### 4. Memify Operation (Optional)

**Purpose**: Semantic enrichment of existing knowledge graphs

**Usage**:
```python
await cognee.memify(dataset="dataset_name")
```

**What it does**:
- Adds derived facts and semantic associations
- Enhances searchability
- Enables specialized search types
- Can be run multiple times incrementally

## Datasets and NodeSets

### Datasets
- **Purpose**: Named containers for documents and metadata
- **Functions**: Organization, permissions, processing tracking
- **Usage**: All operations are scoped by dataset
- **Example**: `"test_github_omnara"`, `"coderisk_main"`

### NodeSets
- **Purpose**: Semantic tags within datasets
- **Functions**: Grouping, filtering, graph traversal
- **Usage**: Added during ingestion with `node_set` parameter
- **Example**: `["github", "commits", "coderisk"]`

## Current CodeRisk Pipeline Issues

### Issue 1: Missing Cognify Step
**Problem**: We add data but never run `cognify()`, so:
- No DocumentChunk_text collection created
- No vector embeddings generated
- Chunk searches fail with 404 errors

**Solution**: Add cognify step after data ingestion:
```python
# After all cognee.add() calls
await cognee.cognify(
    datasets=[self.dataset_name],
    ontology_file_path=str(ontology_path) if ontology_path.exists() else None
)
```

### Issue 2: Incomplete Search Implementation
**Problem**: We're only using basic search types, missing:
- Dataset-scoped searches
- Proper error handling for missing data
- Search type optimization for different data types

**Solution**: Implement proper search patterns:
```python
# For commit data
commit_results = await cognee.search(
    query_type=SearchType.CHUNKS,
    query_text="commits with author and file changes",
    datasets=[dataset_name],
    top_k=50
)

# For code analysis
code_results = await cognee.search(
    query_type=SearchType.CODE,
    query_text="function definitions and imports",
    datasets=[dataset_name]
)

# For risk assessment
risk_results = await cognee.search(
    query_type=SearchType.GRAPH_COMPLETION,
    query_text="hotfix commits and bug incidents",
    datasets=[dataset_name]
)
```

### Issue 3: Ontology Not Being Used
**Problem**: We have an ontology file but it's not being applied during cognify.

**Solution**: Pass ontology path to cognify:
```python
ontology_path = Path("coderisk/ontology/coderisk_ontology.owl")
await cognee.cognify(
    datasets=[dataset_name],
    ontology_file_path=str(ontology_path) if ontology_path.exists() else None
)
```

## Recommended Implementation Plan

### Step 1: Fix Data Processing Pipeline
1. Update `CogneeGitHubProcessor` to run cognify after add operations
2. Ensure ontology is loaded during cognify
3. Verify chunk creation with database inspection

### Step 2: Implement Proper Search Strategy
1. Use CHUNKS for raw data retrieval
2. Use GRAPH_COMPLETION for complex reasoning
3. Use CYPHER for precise calculations
4. Use CODE for code-specific analysis

### Step 3: Optimize for Risk Assessment
1. Create specialized search functions for each risk metric
2. Implement search result aggregation
3. Add proper error handling and fallbacks

## Code Examples for Implementation

### Complete Data Processing Pipeline
```python
async def process_github_data_correctly(self, repo_name: str, window_days: int = 90):
    # Step 1: Extract GitHub data
    github_data = await self.extractor.extract_all(repo_name, window_days)

    # Step 2: Add to Cognee (we're already doing this)
    await self._add_to_cognee(github_data)

    # Step 3: Cognify (WE'RE MISSING THIS!)
    await cognee.cognify(
        datasets=[self.dataset_name],
        ontology_file_path=str(self.ontology_path) if self.ontology_path.exists() else None,
        temporal_cognify=True  # Enable for time-based analysis
    )

    # Step 4: Optional memify for enhanced search
    await cognee.memify(dataset=self.dataset_name)

    return {"status": "complete", "dataset": self.dataset_name}
```

### Proper Search Implementation
```python
async def search_commits_properly(self, query: str, dataset_name: str):
    # Try multiple search types and aggregate results
    search_strategies = [
        (SearchType.CHUNKS, "Raw chunk retrieval"),
        (SearchType.GRAPH_COMPLETION, "Graph-enhanced search"),
        (SearchType.TEMPORAL, "Time-aware search")
    ]

    results = {}
    for search_type, description in search_strategies:
        try:
            result = await cognee.search(
                query_type=search_type,
                query_text=query,
                datasets=[dataset_name],
                top_k=20
            )
            results[search_type.value] = {
                "description": description,
                "count": len(result) if isinstance(result, list) else 1,
                "data": result
            }
        except Exception as e:
            results[search_type.value] = {"error": str(e)}

    return results
```

## Next Steps

1. **IMMEDIATE**: Fix the missing cognify step in our GitHub processor
2. **SHORT-TERM**: Implement proper search patterns with error handling
3. **MEDIUM-TERM**: Optimize search strategies for each risk calculation
4. **LONG-TERM**: Implement memify for enhanced semantic understanding

This research shows that our core architecture is sound, but we're missing the crucial `cognify` step that processes our ingested data into a searchable knowledge graph.