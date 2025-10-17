# Cognee Search Implementation Strategy for Risk Math Calculations



## Executive Summary



This document outlines the comprehensive search implementation strategy for all risk math calculations in the CodeRisk system using Cognee's search capabilities. Based on analysis of `cognee_mvp_design.md`, `risk_math.md`, and `testing_spec.md`, this strategy maximizes retrieval insight and evidence quality while maintaining performance requirements (≤2s p50, ≤5s p95).



## Core Requirements and Constraints



### Performance Boundaries

- **Graph queries**: ≤2 hops, degree-capped (~200 nodes), ≤500ms per query

- **Overall assessment**: ≤2s p50 / ≤5s p95 (warm cache)

- **Detector execution**: 150-450ms p50 per micro-detector

- **Evidence quality priority**: Multiple searches with result fusion over speed



### Data Sources

- **90-day historical window**: commits, PRs, issues, ownership data

- **Three-database architecture**: RDBMS, vector (LanceDB), graph (Kuzu via Cognee)

- **Real-time inputs**: working tree diffs, PR changes

- **Graph structures**: IMPORTS, CO_CHANGED relationships with temporal metadata



## Search Strategy Matrix by Calculation Type



### 1. Graph-Intensive Calculations



#### Δ-Diffusion Blast Radius (ΔDBR)

**Mathematical Requirement**: Local PPR/HKPR delta from proposed edge changes over IMPORTS/CO_CHANGED graphs



**Primary Search Strategy**:

- **SearchType.GRAPH_COMPLETION** for comprehensive graph traversal

- **Raw CYPHER queries** for precise PPR calculations with degree caps

- **SearchType.TEMPORAL** for historical blast radius patterns



**Evidence Collection**:

1. Direct graph traversal: `IMPORTS` and `CO_CHANGED` relationships within 2-hop limit

2. Historical impact analysis: Files previously affected by similar changes

3. Dependency fan-out: Downstream consumers and their change frequency



**Query Pattern**: Start from changed files → traverse IMPORTS edges → calculate PPR delta → apply temporal decay



#### Hawkes-Decayed Co-Change (HDCC)

**Mathematical Requirement**: Two-timescale (fast/slow) decay modeling for evolutionary coupling



**Primary Search Strategy**:

- **SearchType.TEMPORAL** with time-aware queries for historical co-change patterns

- **Raw CYPHER** for precise co-change frequency calculations

- **SearchType.GRAPH_COMPLETION** for relationship strength analysis



**Evidence Collection**:

1. Historical co-change events: Files that changed together in 90-day window

2. Temporal decay modeling: Recent vs. distant co-change patterns

3. Change coupling strength: Frequency and proximity of joint modifications



**Query Pattern**: Find co-change pairs → calculate temporal distances → apply dual-decay model (fast/slow)



#### Span-Core & Bridge Risk

**Mathematical Requirement**: Temporal core persistence and bridging centrality in IMPORTS subgraph



**Primary Search Strategy**:

- **SearchType.GRAPH_COMPLETION** for centrality calculations

- **Raw CYPHER** for 2-hop IMPORTS subgraph analysis

- **SearchType.TEMPORAL** for core persistence over time



**Evidence Collection**:

1. Bridge detection: Files connecting otherwise disconnected components

2. Core persistence: Files consistently central across time windows

3. Structural importance: Impact on graph connectivity if removed



**Query Pattern**: Extract 2-hop subgraph → calculate betweenness centrality → identify temporal patterns



### 2. Hybrid Search Calculations



#### Incident Adjacency (GB-RRF)

**Mathematical Requirement**: Reciprocal Rank Fusion of BM25 + vector kNN + local PPR ranks



**Multi-Modal Search Strategy**:

- **SearchType.CHUNKS** for BM25 text similarity to incident descriptions

- **Vector similarity search** for semantic matching of code changes

- **SearchType.GRAPH_COMPLETION** for PPR-based structural similarity

- **SearchType.FEELING_LUCKY** for automatic optimal approach selection



**Evidence Collection**:

1. Textual similarity: Incident descriptions matching change descriptions

2. Semantic similarity: Code patterns similar to past incident code

3. Structural similarity: Graph proximity to previous incident locations

4. Fusion ranking: Combined RRF score with evidence traceability



**Query Pattern**: Parallel execution of three search types → rank fusion → evidence aggregation



#### Ownership Authority Mismatch (OAM)

**Mathematical Requirement**: owner_churn, minor_edit_ratio, experience_gap, bus_entropy calculations



**Primary Search Strategy**:

- **SearchType.TEMPORAL** for ownership history analysis

- **SearchType.CODE** for file modification patterns

- **Raw CYPHER** for ownership relationship graphs



**Evidence Collection**:

1. Ownership transitions: Historical CODEOWNERS changes for affected files

2. Edit pattern analysis: Ratio of minor vs. major edits by author

3. Experience metrics: Author familiarity with codebase areas

4. Bus factor: Distribution of knowledge across team members



**Query Pattern**: Aggregate ownership data → calculate churn metrics → assess authority gaps



### 3. Pattern Detection Calculations



#### G² Surprise (Dunning Log-Likelihood)

**Mathematical Requirement**: Statistical analysis of unusual file pair relationships



**Primary Search Strategy**:

- **Raw CYPHER** for precise co-occurrence statistics

- **SearchType.GRAPH_COMPLETION** for relationship pattern analysis

- **SearchType.CODING_RULES** for extracted statistical patterns



**Evidence Collection**:

1. Co-occurrence frequencies: Expected vs. observed file pair changes

2. Statistical significance: Log-likelihood ratios for unusual patterns

3. Context patterns: Surrounding files that explain or contradict surprises



**Query Pattern**: Calculate expected co-change frequencies → compare with observed → identify anomalies



#### JIT Baselines

**Mathematical Requirement**: Size, churn, diff entropy calculations



**Primary Search Strategy**:

- **SearchType.CODE** for file size and change metrics

- **SearchType.TEMPORAL** for historical churn analysis

- **Raw queries** for entropy calculations



**Evidence Collection**:

1. Size metrics: Files touched, lines changed, complexity measures

2. Churn patterns: Historical modification frequency and intensity

3. Entropy analysis: Information content of changes relative to file history



**Query Pattern**: Aggregate file metrics → calculate historical baselines → compute relative risk



## Micro-Detector Search Strategies



### API Break Risk Detection

**Search Requirements**: Public surface analysis and importer impact



**Implementation Strategy**:

- **SearchType.CODE** for API surface extraction

- **CodeGraph.get_dependencies()** for importer analysis

- **SearchType.GRAPH_COMPLETION** for impact propagation



**Evidence Sources**: Public function signatures, import relationships, downstream usage patterns



### Schema Risk Detection

**Search Requirements**: Migration analysis and table impact assessment



**Implementation Strategy**:

- **SearchType.CODE** for migration file analysis

- **Raw CYPHER** for table relationship graphs

- **SearchType.TEMPORAL** for schema evolution patterns



**Evidence Sources**: DDL statements, table dependency graphs, historical migration patterns



### Security Risk Detection

**Search Requirements**: Pattern matching and vulnerability correlation



**Implementation Strategy**:

- **SearchType.CHUNKS** for code pattern similarity

- **SearchType.FEELING_LUCKY** for automatic vulnerability detection

- **Custom DataPoints** with security-specific indexing



**Evidence Sources**: Security patterns, vulnerability databases, code similarity matches



### Performance Risk Detection

**Search Requirements**: Hotpath analysis and complexity assessment



**Implementation Strategy**:

- **SearchType.CODE** for loop and I/O pattern detection

- **ΔDBR calculations** for centrality weighting

- **SearchType.TEMPORAL** for performance regression history



**Evidence Sources**: Code complexity metrics, execution path analysis, historical performance data



### Test Gap Risk Detection

**Search Requirements**: Test coverage correlation and semantic mapping



**Implementation Strategy**:

- **SearchType.CODE** for test file discovery

- **SearchType.CHUNKS** for semantic test-code mapping

- **Ratio calculations** with smoothing for coverage gaps



**Evidence Sources**: Test file locations, semantic similarity scores, coverage statistics



## Search Optimization Strategies



### Performance Optimization

1. **Degree Capping**: Limit graph traversals to ~200 nodes maximum

2. **Hop Limiting**: Restrict graph searches to 2-hop maximum depth

3. **Batch Processing**: Group similar calculations to reduce query overhead

4. **NodeSet Caching**: Cache frequently accessed subgraphs using Cognee NodeSets

5. **Parallel Execution**: Run independent searches concurrently



### Quality Optimization

1. **Multi-Search Fusion**: Combine multiple search types via Reciprocal Rank Fusion

2. **Evidence Traceability**: Maintain links to source files, lines, and commits

3. **Temporal Context**: Include time-aware decay in all historical analyses

4. **Feedback Integration**: Use SearchType.FEEDBACK for continuous learning

5. **Memify Enhancement**: Leverage extracted patterns from cognee.memify()



### Search Type Selection Matrix



| Calculation | Primary Search | Secondary Search | Evidence Fusion |

|-------------|---------------|------------------|-----------------|

| ΔDBR | GRAPH_COMPLETION | TEMPORAL | Graph + Historical |

| HDCC | TEMPORAL | GRAPH_COMPLETION | Time + Structure |

| G² Surprise | Raw CYPHER | CODING_RULES | Statistical + Patterns |

| OAM | TEMPORAL | CODE | Time + Ownership |

| Bridge Risk | GRAPH_COMPLETION | Raw CYPHER | Structure + Precision |

| Incident Adjacency | CHUNKS + Vector + GRAPH_COMPLETION | FEELING_LUCKY | Triple RRF |

| API Break | CODE | GRAPH_COMPLETION | Surface + Impact |

| Schema Risk | CODE | TEMPORAL | DDL + History |

| Security Risk | CHUNKS | FEELING_LUCKY | Pattern + Auto |

| Performance Risk | CODE | TEMPORAL | Complexity + History |

| Test Gap | CODE | CHUNKS | Coverage + Semantic |



## Implementation Phases



### Phase 1: Core Graph Calculations

- Implement ΔDBR with GRAPH_COMPLETION and raw CYPHER

- Set up HDCC with temporal decay modeling

- Configure bridge risk with centrality calculations



### Phase 2: Hybrid Search Integration

- Implement incident adjacency with triple RRF fusion

- Set up OAM with temporal ownership analysis

- Configure G² surprise with statistical pattern detection



### Phase 3: Micro-Detector Implementation

- Deploy all 9 micro-detectors with appropriate search strategies

- Implement parallel execution with timeout handling

- Set up evidence aggregation and traceability



### Phase 4: Optimization and Learning

- Enable feedback system for continuous improvement

- Implement memify-based pattern extraction

- Optimize query performance with caching and batching



## Success Metrics



### Performance Targets

- **Overall assessment time**: ≤2s p50, ≤5s p95

- **Individual search queries**: ≤500ms for graph operations

- **Micro-detector execution**: 150-450ms p50 per detector

- **Evidence collection**: Complete traceability for all high-risk assessments



### Quality Targets

- **Evidence depth**: ≥3 evidence sources per high-risk calculation

- **Search coverage**: ≥6/9 micro-detectors active per assessment

- **False positive control**: ≤5% through Conformal Risk Control

- **Path discovery**: ≥60% of high-risk assessments show causal paths



## Integration with Cognee Features



### Temporal Cognify

- Enable `temporal_cognify=True` for all historical analyses

- Use time-aware queries for decay modeling and trend analysis

- Leverage temporal DataPoints for chronological evidence



### Custom Pipelines and Tasks

- Create reusable Task definitions for each calculation type

- Implement Pipeline compositions for complex multi-step analyses

- Use async execution for parallel search operations



### DataPoint Optimization

- Design calculation-specific DataPoint structures

- Optimize `index_fields` for search performance

- Use `metadata` for search filtering and context



### Feedback Integration

- Implement `save_interaction=True` for learning from assessments

- Use SearchType.FEEDBACK for prediction accuracy improvement

- Track and learn from risk assessment outcomes



This strategy provides the foundation for implementing all risk math calculations with optimal search performance, comprehensive evidence collection, and continuous learning capabilities within Cognee's architecture.