# ADR-003: PostgreSQL Full-Text Search for Incident Similarity

**Date:** 2025-10-05
**Status:** Accepted
**Deciders:** Architecture team
**Tags:** technology, storage, simplification

---

## Context

CodeRisk needs to perform keyword-based similarity search to match commit messages and code changes against historical incident descriptions. This helps identify when a current change is similar to past failures.

**Background:**
- V1.0 design explicitly excludes vector embeddings (too expensive, marginal accuracy gains)
- We only need BM25-style keyword matching, not semantic similarity
- Initial documentation incorrectly suggested LanceDB for text search
- We already have PostgreSQL in our tech stack for metadata storage
- Need <100ms search latency for incident similarity metric

**Constraints:**
- Must support keyword search with ranking (BM25-like)
- Should integrate with existing PostgreSQL metadata queries
- Need to minimize infrastructure complexity
- Must handle ~1,000-10,000 incidents per repository

---

## Decision

**We will:** Use PostgreSQL's built-in full-text search (FTS) for incident similarity matching instead of adding LanceDB or another search engine.

---

## Options Considered

### Option 1: PostgreSQL Full-Text Search (CHOSEN)

**Pros:**
- Already in our tech stack (no new infrastructure)
- Built-in `tsvector` and GIN indexing for fast text search
- Excellent for BM25-style keyword matching via `ts_rank_cd()`
- Can join with other metadata queries efficiently
- Native support for stemming, stop words, and language-specific search
- <50ms query time for 10K incidents (with GIN index)
- Zero additional cost

**Cons:**
- Slightly less flexible than dedicated search engines
- Ranking algorithm is PostgreSQL's default (not pure BM25, but similar)
- Not designed for massive-scale text search (but we only have 1K-10K incidents)

**Cost:** $0 (already using PostgreSQL)

### Option 2: LanceDB (Vector + Text Search)

**Pros:**
- Supports both vector embeddings and BM25 search
- Optimized for similarity search at scale
- Could support future semantic search if needed

**Cons:**
- **Requires vector embeddings** (we explicitly excluded this in v1.0)
- Adds new infrastructure dependency
- Increased operational complexity (another service to manage)
- ~$50-100/month for hosting (AWS ECS or separate instance)
- Overkill for simple keyword matching
- Would need to sync data from PostgreSQL (data duplication)

**Cost:** ~$50-100/month + operational overhead

### Option 3: Elasticsearch / OpenSearch

**Pros:**
- Industry-standard text search
- Excellent BM25 implementation
- Very fast for large-scale search

**Cons:**
- Massive overkill for 1K-10K incidents
- Requires separate cluster (~$200-500/month minimum)
- High operational complexity
- Data duplication with PostgreSQL
- Need to maintain sync between PostgreSQL and ES

**Cost:** ~$200-500/month + operational overhead

### Option 4: In-Memory BM25 (Python libraries)

**Pros:**
- Simple implementation
- No additional infrastructure
- Can use libraries like `rank-bm25`

**Cons:**
- Requires loading all incidents into memory on every search
- Slow for 10K+ incidents (~500ms)
- No persistent indexing
- Doesn't scale as incident count grows

**Cost:** $0 (but poor performance)

---

## Rationale

**Key factors:**

1. **Simplicity over features**: We need keyword search, not semantic search. PostgreSQL FTS is built for this exact use case.

2. **Infrastructure minimization** *(12-factor: Factor 5 - Unify execution state)*: Every additional service adds operational burden. PostgreSQL is already required for metadata—using it for text search eliminates a dependency.

3. **Performance is sufficient**: PostgreSQL GIN indexes provide <50ms search time for 10K incidents, well within our <100ms target.

4. **Cost efficiency**: $0 vs $50-500/month for dedicated search infrastructure. Over 3 years, this saves $1,800-18,000.

5. **Query integration**: Can join incident search with other metadata queries in a single SQL statement:
   ```sql
   SELECT i.*, ts_rank_cd(i.search_vector, query) AS score,
          c.sha, c.timestamp
   FROM incidents i
   JOIN caused_by cb ON i.id = cb.incident_id
   JOIN commits c ON cb.commit_sha = c.sha
   WHERE i.search_vector @@ to_tsquery('stripe & timeout')
   ORDER BY score DESC;
   ```

**Data supporting decision:**
- PostgreSQL FTS benchmark: 47ms for 10K incidents with GIN index (tested on `db.t3.medium`)
- 95th percentile: <80ms
- Index size: ~5MB for 10K incidents
- Memory overhead: Minimal (indexes cached in shared buffers)

---

## Consequences

**Positive:**
- Simpler architecture (fewer moving parts)
- Lower operational complexity (one less service to manage)
- Zero additional cost
- Fast enough for our use case (<50ms)
- Can leverage existing PostgreSQL expertise
- Tight integration with metadata queries

**Negative:**
- Limited to keyword search (can't add semantic search without embeddings)
- PostgreSQL's ranking isn't pure BM25 (uses `ts_rank_cd()` which is similar but not identical)
- If we ever need >100K incidents, may need to revisit (but unlikely per repo)

**Neutral:**
- Incident data stored in PostgreSQL instead of separate search engine
- Need to create and maintain `tsvector` columns and GIN indexes

**Risks:**
- **Risk**: PostgreSQL FTS doesn't scale to millions of incidents
  - **Mitigation**: We only store 90 days of incidents per repo (~1K-10K). If we hit 100K+, we can partition or move to dedicated search.

- **Risk**: Query performance degrades with complex searches
  - **Mitigation**: Benchmark shows <50ms for realistic queries. Can add more indexes if needed.

---

## Implementation Notes

**Database schema:**
```sql
CREATE TABLE incidents (
    id SERIAL PRIMARY KEY,
    external_id VARCHAR(100) UNIQUE,  -- GitHub issue #, Jira ticket, etc.
    title TEXT NOT NULL,
    description TEXT,
    severity VARCHAR(20),
    created_at TIMESTAMP NOT NULL,
    resolved_at TIMESTAMP,
    repository_id INTEGER REFERENCES repositories(id),
    search_vector TSVECTOR  -- Auto-maintained full-text search index
);

-- GIN index for fast text search
CREATE INDEX incidents_search_idx ON incidents USING GIN(search_vector);

-- Auto-update search vector on insert/update
CREATE TRIGGER incidents_search_update
BEFORE INSERT OR UPDATE ON incidents
FOR EACH ROW EXECUTE FUNCTION
tsvector_update_trigger(search_vector, 'pg_catalog.english', title, description);
```

**Query pattern:**
```sql
-- Find similar incidents (BM25-like ranking)
SELECT id, title,
       ts_rank_cd(search_vector, query) AS similarity_score
FROM incidents,
     to_tsquery('english', 'stripe & timeout & payment') AS query
WHERE search_vector @@ query
  AND repository_id = $repo_id
  AND created_at > NOW() - INTERVAL '90 days'
ORDER BY similarity_score DESC
LIMIT 5;
```

**Performance targets:**
- <50ms p50 latency
- <100ms p95 latency
- <5MB index size per 10K incidents

**Timeline:** Immediate (v1.0)

**Dependencies:**
- PostgreSQL 12+ (for improved FTS features)
- Existing `incidents` table schema

**Migration path:**
- N/A (no previous implementation to migrate from)
- Remove LanceDB references from documentation

---

## References

- [graph_ontology.md](../graph_ontology.md) - Storage architecture (Layer 3: Incidents)
- [agentic_design.md](../agentic_design.md) - Tier 2 metrics (incident similarity)
- [PostgreSQL Full-Text Search Documentation](https://www.postgresql.org/docs/current/textsearch.html)
- [spec.md](../../spec.md) - NFR-22: Incident similarity search <100ms
