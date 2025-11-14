# ASSOCIATED_WITH Edge Creation System

## Overview

The ASSOCIATED_WITH edge system creates connections between Issues/PRs and Commits in the Neo4j knowledge graph. These edges represent relationships discovered through multiple detection methods, combining AI-powered extraction with rule-based temporal correlation to build a comprehensive map of how code changes relate to project incidents and features.

## Purpose and Value

**Why ASSOCIATED_WITH edges matter:**
- Enable risk analysis by linking code changes to bug reports and incidents
- Support impact assessment by connecting PRs to their related commits
- Provide traceability between project management (Issues/PRs) and version control (Commits)
- Power CLQS (Code Line Quality Score) calculations through evidence-based quality metrics

**What they represent:**
- A directional relationship from an Issue or PR to a Commit
- Includes semantic meaning (fixes, mentions, associated_with)
- Carries confidence scores (0.4-1.0) indicating reliability
- Contains metadata about how the relationship was discovered

## Edge Schema

### Node Types (Source)
- **Issue nodes:** Represent GitHub issues (bug reports, feature requests, tasks)
- **PR nodes:** Represent GitHub pull requests (code review units)

### Node Types (Target)
- **Commit nodes:** Represent git commits (atomic code changes)

### Edge Direction
```
(Issue)-[:ASSOCIATED_WITH]->(Commit)
(PR)-[:ASSOCIATED_WITH]->(Commit)
```

### Edge Properties

Every ASSOCIATED_WITH edge contains five metadata fields:

1. **relationship_type** (string)
   - Semantic meaning of the relationship
   - Values: "fixes", "mentions", "associated_with"
   - Determines how strongly the entities are related

2. **confidence** (float: 0.0-1.0)
   - Reliability score of the detected relationship
   - Higher values indicate stronger evidence
   - Minimum threshold: 0.4 (lower confidence edges are filtered out)

3. **detected_via** (string)
   - The detection method that discovered this relationship
   - Values: "commit_extraction", "pr_extraction", "temporal"
   - Indicates data source and extraction technique

4. **rationale** (string)
   - Human-readable explanation of why this edge exists
   - Examples: "Extracted from commit_message", "Extracted from temporal_correlation_temporal_match_1hr"
   - Provides audit trail for debugging and validation

5. **evidence** (array)
   - Tags used for CLQS (Code Line Quality Score) calculation
   - Can include patterns, keywords, or timestamps
   - Supports downstream quality metrics

## Three-Path Detection System

The system uses three complementary methods to discover relationships, each optimized for different scenarios:

### Path 1: Commit Message Extraction (LLM-Based)

**How it works:**
1. Fetches unprocessed commits from PostgreSQL staging database
2. Processes commits in batches of 20
3. Sends commit messages to Gemini Flash 1.5 LLM with structured prompt
4. LLM extracts issue/PR references and classifies relationship type
5. Returns JSON with target IDs, actions, and confidence scores

**Detection capabilities:**
- Explicit references: "#42", "GH-123", "issue 456"
- Action keywords: "fixes #42", "closes #123", "resolves #456"
- Implicit mentions: "see #42 for context", "related to #123"
- Multi-reference extraction: Single commit can reference multiple issues

**Confidence scoring:**
- **0.9:** Explicit "fixes/closes/resolves" with issue number
- **0.7:** Clear mention with issue number but non-fix action
- **0.5:** Implicit or unclear references

**Example input:**
```
Commit: "Fix authentication timeout bug (#127)"
```

**Example output:**
```json
{
  "target_id": 127,
  "action": "fixes",
  "confidence": 0.9
}
```

**Edge created:**
```
(Issue #127)-[:ASSOCIATED_WITH]->(Commit abc123)
{
  relationship_type: "fixes",
  confidence: 0.9,
  detected_via: "commit_extraction",
  rationale: "Extracted from commit_message"
}
```

**Volume:** ~165 edges (67% of total)

---

### Path 2: PR Body Extraction (LLM-Based)

**How it works:**
1. Fetches unprocessed pull requests from staging database
2. Processes PR descriptions/bodies in batches
3. Sends PR text to Gemini Flash 1.5 LLM
4. LLM extracts issue references from PR description
5. Returns structured references with relationship types

**Detection capabilities:**
- PR templates: "Fixes #42", "Closes #123"
- Related issues: "Related to #456", "Depends on #789"
- Cross-references: "Supersedes #321", "Continues #654"
- GitHub auto-link syntax parsing

**Confidence scoring:**
- **0.9:** Explicit fix/close keywords in PR body
- **0.8:** Clear reference without fix keywords
- **0.6:** Weak or ambiguous mention

**Example input:**
```
PR #45 body: "This PR fixes the race condition reported in #127 and #128"
```

**Example output:**
```json
[
  {"target_id": 127, "action": "fixes", "confidence": 0.9},
  {"target_id": 128, "action": "fixes", "confidence": 0.9}
]
```

**Edge created:**
```
(PR #45)-[:ASSOCIATED_WITH]->(Commit xyz789)
{
  relationship_type: "fixes",
  confidence: 0.9,
  detected_via: "pr_extraction",
  rationale: "Extracted from pr_body"
}
```

**Volume:** ~18 edges (7% of total)

---

### Path 3: Temporal Correlation (Rule-Based)

**How it works:**
1. Queries staging database for issues and their state transitions
2. Finds commits that occurred near issue closure/update events
3. Applies time-window rules to determine correlation strength
4. Creates edges for temporally proximate events without explicit references

**Detection rules:**

| Time Window | Confidence | Rationale |
|-------------|------------|-----------|
| Within 5 minutes | 0.8 | Very likely related (developer closed issue right after commit) |
| Within 1 hour | 0.7 | Likely related (same work session) |
| Within 24 hours | 0.5 | Possibly related (same day) |

**When it activates:**
- Issue closed without explicit commit reference
- PR merged near commit timestamp
- Issue updated shortly after commit
- Commit has no message or ambiguous message

**Example scenario:**
```
Issue #127: "Authentication timeout" (State: Open)
  ↓
Commit abc123: "Update session handler" (timestamp: 2024-01-15 10:00:00)
  ↓
Issue #127: State changed to Closed (timestamp: 2024-01-15 10:05:00)
```

**Edge created:**
```
(Issue #127)-[:ASSOCIATED_WITH]->(Commit abc123)
{
  relationship_type: "associated_with",
  confidence: 0.8,
  detected_via: "temporal",
  rationale: "Extracted from temporal_correlation_temporal_match_5min"
}
```

**Advantages:**
- Catches relationships missed by text extraction
- No dependency on commit message quality
- Works across languages and conventions
- Handles silent fixes (commits without issue references)

**Volume:** ~62 edges (25% of total)

---

## Bidirectional Confidence Boosting

When the same relationship is detected from multiple sources, the system merges references and increases confidence:

**Scenario:** Commit message mentions "#42" AND Issue #42's timeline has a commit reference event

**Process:**
1. Commit extraction finds: Issue #42, confidence 0.7
2. Timeline extraction finds: Issue #42 → Commit abc123
3. System merges both references
4. Boosts confidence: 0.7 → 0.9 (bidirectional evidence)

**Result:** Single high-confidence edge instead of multiple weak edges

---

## Entity Resolution

The system handles GitHub's ambiguous numbering scheme where Issues and PRs share the same number space:

**Problem:** "#42" could be Issue #42 OR PR #42

**Solution:**
1. Extract raw number from text (e.g., "42")
2. Query staging database: "Is #42 an Issue or PR?"
3. Resolve entity type using `EntityResolver`
4. Create appropriate source node ID: `issue:42` or `pr:42`

**Edge label logic:**
```
IF (entity_type == Issue AND action == "fixes"):
    Create FIXED_BY edge  // Specific edge for bug fixes
ELSE:
    Create ASSOCIATED_WITH edge  // Generic association
```

**Example:**
- "#42" in commit message could be PR #42 (code review) or Issue #42 (bug report)
- System queries database, finds PR #42 exists
- Creates: `(PR #42)-[:ASSOCIATED_WITH]->(Commit)`
- If Issue #42 existed instead and action was "fixes"
- Would create: `(Issue #42)-[:FIXED_BY]->(Commit)` instead

---

## Data Flow Pipeline

### Phase 1: GitHub Ingestion
```
GitHub API → PostgreSQL Staging Tables
```

**Tables populated:**
- `github_commits` (commit metadata and messages)
- `github_pull_requests` (PR metadata and bodies)
- `github_issues` (issue metadata)
- `github_issue_timeline` (state transitions and events)

**Timing:** During `crisk init` command, fetches last N days of history

---

### Phase 2: Reference Extraction
```
PostgreSQL → LLM Extraction → `github_issue_commit_refs` table
```

**Process:**
1. Commit extractor processes unprocessed commits
2. PR extractor processes unprocessed PRs
3. LLM returns structured references
4. References stored in `github_issue_commit_refs` with metadata

**Table schema:**
- `issue_number` (int) - Target issue/PR number
- `commit_sha` (string) - Target commit hash
- `pr_number` (int) - Source PR if applicable
- `action` (string) - Relationship type (fixes, mentions, etc.)
- `confidence` (float) - Reliability score
- `detection_method` (string) - How it was found
- `extracted_from` (string) - Source context

---

### Phase 3: Temporal Correlation
```
PostgreSQL Issue Timeline → Rule Engine → `github_issue_commit_refs` table
```

**Process:**
1. Temporal correlator queries issue state changes
2. Finds commits within time windows
3. Applies confidence scoring rules
4. Inserts temporal references into same table

**Deduplication:** Same table as LLM extraction allows merge with bidirectional boost

---

### Phase 4: Graph Edge Creation
```
`github_issue_commit_refs` → Entity Resolution → Neo4j ASSOCIATED_WITH edges
```

**Process:**
1. IssueLinker reads all references from staging table
2. Merges bidirectional references (boost confidence)
3. Filters low-confidence edges (< 0.4 threshold)
4. Resolves entity types (Issue vs PR disambiguation)
5. Creates Neo4j edges with full metadata
6. Logs skipped references (low confidence, entity not found)

**Timing:** During `crisk init` after nodes are created, or during incremental updates

---

## Quality Filters

### Minimum Confidence Threshold
- **Cutoff:** 0.4
- **Purpose:** Eliminate noise from weak correlations
- **Impact:** ~10-15% of raw references filtered out

### Entity Existence Validation
- **Check:** Target Issue/PR/Commit must exist in graph
- **Purpose:** Prevent dangling edges to non-existent nodes
- **Common cause:** Reference to entity outside ingestion time window

### Duplicate Prevention
- **Method:** Merge bidirectional references before edge creation
- **Purpose:** Avoid multiple edges for same relationship
- **Benefit:** Cleaner graph, higher confidence scores

---

## Statistical Breakdown (Omnara Repository)

### Overall Metrics
- **Total edges:** 245
- **PR → Commit:** 176 edges (72%)
- **Issue → Commit:** 69 edges (28%)

### By Detection Method
| Method | Count | Percentage | LLM Used |
|--------|-------|------------|----------|
| commit_extraction | 165 | 67% | ✅ Gemini Flash 1.5 |
| temporal | 62 | 25% | ❌ Rule-based |
| pr_extraction | 18 | 7% | ✅ Gemini Flash 1.5 |

**LLM extraction:** 183 edges (75%)
**Rule-based:** 62 edges (25%)

### By Relationship Type
| Type | Count | Percentage | Meaning |
|------|-------|------------|---------|
| mentions | 149 | 61% | Generic reference, not a fix |
| associated_with | 62 | 25% | Temporal correlation |
| fixes | 34 | 14% | Explicit bug fix |

### By Confidence Score
| Confidence | Count | Percentage | Quality |
|------------|-------|------------|---------|
| 0.9 | 41 | 17% | High - Explicit fix keywords |
| 0.8 | 38 | 16% | Medium-high - Clear reference |
| 0.7 | 113 | 46% | Medium - Mention or 1hr window |
| 0.5-0.6 | 53 | 22% | Lower - Weak temporal match |

### By Extraction Source
| Source | Count | Description |
|--------|-------|-------------|
| commit_message | 165 | LLM parsed commit text |
| temporal_match_1hr | 45 | Issue closed within 1 hour |
| pr_body | 18 | LLM parsed PR description |
| temporal_match_5min | 16 | Issue closed within 5 minutes |
| temporal_match_24hr | 1 | Issue closed within 24 hours |

---

## LLM Configuration

### Model Used
**Gemini Flash 1.5**

**Why this model:**
- Fast inference (batch processing 20 commits/PRs at once)
- Cost-effective for high-volume extraction
- Strong structured output reliability (JSON parsing)
- Multilingual support (handles non-English commits)
- Good at understanding GitHub conventions

### Prompt Engineering

**Commit extraction prompt:**
```
Extract all issue/PR references from this commit message.
Identify the relationship type (fixes, closes, resolves, mentions).
Assign confidence scores based on explicitness.
Return JSON array with: target_id, action, confidence.
```

**PR extraction prompt:**
```
Extract all issue references from this PR description.
Identify fix/close relationships vs general mentions.
Handle GitHub auto-linking syntax (#123, GH-456).
Return JSON array with: target_id, action, confidence.
```

**Output validation:**
- JSON schema validation
- Range check on confidence (0.0-1.0)
- Integer validation on target IDs
- Enum validation on action types

### Error Handling
- **LLM timeout:** Skip batch, mark as unprocessed, retry later
- **Invalid JSON:** Log warning, skip reference, continue processing
- **Rate limiting:** Exponential backoff with retry
- **API errors:** Fail gracefully, preserve partial results

---

## Integration Points

### With Graph Construction
- **When:** During `BuildGraph()` in `internal/graph/builder.go`
- **Order:** After nodes created, before validation
- **Function:** `IssueLinker.LinkIssues(ctx, repoID)`

### With Risk Analysis
- **Usage:** `crisk check` queries ASSOCIATED_WITH edges
- **Purpose:** Find incidents related to changed files
- **Query pattern:** `(File)<-[:MODIFIED]-(Commit)<-[:ASSOCIATED_WITH]-(Issue {is_bug: true})`

### With CLQS Calculation
- **Role:** Evidence field provides quality signals
- **Metric:** Incident density = incidents / total commits
- **Impact:** Files with many FIXED_BY edges have lower quality scores

---

## Operational Characteristics

### Performance
- **Extraction speed:** ~100 commits/minute (LLM-limited)
- **Graph creation:** ~1000 edges/second (Neo4j batch insert)
- **Memory usage:** <100MB for 1000 references (staging table)

### Idempotency
- **Safe re-runs:** `processed_at` timestamps prevent duplicate extraction
- **Incremental updates:** Only new commits/PRs/issues processed
- **Graph merges:** Neo4j MERGE ensures no duplicate edges

### Monitoring
- **Logs:** Extraction counts, skipped references, confidence distribution
- **Validation:** `cmd/validate-graph` checks edge counts and schemas
- **Alerts:** Low extraction rate, high skip rate, LLM errors

---

## Edge Cases and Limitations

### Multiple References in One Commit
✅ **Handled:** LLM extracts all references, creates multiple edges
```
"Fixes #42 and #43" → Two ASSOCIATED_WITH edges
```

### Ambiguous Actions
✅ **Handled:** LLM assigns most likely relationship type
```
"See #42 for details" → action: "mentions", confidence: 0.7
```

### False Positives (Temporal)
⚠️ **Limitation:** Commit and issue closure may be coincidental
```
Developer closes unrelated issue shortly after commit
→ Creates low-confidence edge (0.5-0.7)
```

### Missing References
⚠️ **Limitation:** Developer doesn't mention issue in commit/PR
```
Commit fixes bug but no "#42" reference
→ Only temporal correlation can detect (if timing matches)
```

### Cross-Repository References
❌ **Not supported:** References to issues in other repositories filtered out
```
"Fixes org/other-repo#42" → Skipped (entity not found in current repo)
```

### Deleted/Private Issues
⚠️ **Skipped:** Reference extracted but entity resolution fails
```
"Fixes #42" but Issue #42 deleted or outside time window
→ Logged as "entity not found", no edge created
```

---

## Configuration Options

### Ingestion Scope
- `--days N` flag controls history depth (default: 90 days)
- Affects which commits/PRs/issues are fetched
- Older references may fail entity resolution

### Confidence Threshold
- Hardcoded at 0.4 in `internal/graph/issue_linker.go`
- Can be adjusted to trade precision/recall
- Lower threshold: More edges, more noise
- Higher threshold: Fewer edges, miss weak signals

### LLM Provider
- Environment variable: `LLM_PROVIDER=gemini`
- Alternatives: OpenAI GPT-4, Anthropic Claude (via adapter)
- Requires corresponding API key

### Batch Sizes
- Commit extraction: 20 commits/batch
- PR extraction: 20 PRs/batch
- Graph edge creation: 100 edges/batch

---

## Troubleshooting

### No Edges Created
**Symptom:** 0 ASSOCIATED_WITH edges in graph
**Causes:**
1. No commits/PRs ingested (check `github_commits`, `github_pull_requests` tables)
2. LLM extraction failed (check logs for API errors)
3. All references below confidence threshold
4. Entity resolution failed (references to non-existent issues)

**Debug:**
```sql
SELECT COUNT(*) FROM github_issue_commit_refs WHERE repo_id = 1;
```

---

### Low Edge Count
**Symptom:** Fewer edges than expected
**Causes:**
1. Short ingestion window (--days too small)
2. Developers don't reference issues in commits
3. High skip rate (check logs for "skipped" messages)

**Debug:**
```
Check logs for: "Skipped low confidence: X references"
```

---

### High False Positive Rate
**Symptom:** Unrelated issues linked to commits
**Causes:**
1. Temporal window too wide (captures coincidental closures)
2. LLM misinterprets mentions as fixes
3. Cross-talk in batch processing

**Solutions:**
- Reduce temporal window (5min instead of 1hr)
- Increase confidence threshold
- Review LLM prompt for clarity

---

## Future Enhancements

### Planned Improvements
1. **File-level linking:** Connect issues to specific changed files
2. **Multi-hop reasoning:** "PR #45 fixes Issue #42 which caused by Commit xyz"
3. **Confidence learning:** Train model to improve scoring over time
4. **Cross-repository support:** Handle references to external repos

### Experimental Features
1. **Semantic similarity:** Use embeddings to find related issues without explicit refs
2. **Developer patterns:** Learn individual developer's commit message styles
3. **Auto-tagging:** Classify relationship types beyond fixes/mentions

---

## Summary

The ASSOCIATED_WITH edge system creates a rich knowledge graph connecting project management entities (Issues, PRs) to code changes (Commits). By combining AI-powered text extraction with rule-based temporal correlation, it achieves 75% coverage through LLM methods while capturing silent relationships through time-based heuristics. The three-path detection system provides redundancy and comprehensive coverage, while confidence scoring and entity resolution ensure high-quality edges. This foundation enables advanced risk analysis, incident tracking, and code quality metrics throughout the CodeRisk platform.
