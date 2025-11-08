# CodeRisk Schema Reference

Complete schema documentation for PostgreSQL staging tables and Neo4j graph database.

## PostgreSQL Staging Schema

### github_issue_pr_links
Validated Issue-PR relationships with multi-signal confidence scoring.

```sql
Column                    | Type                  | Notes
--------------------------|-----------------------|----------------------------------
id                        | bigint                | Primary key
repo_id                   | bigint                | FK to github_repositories
issue_number              | integer               | Issue number (not internal ID)
pr_number                 | integer               | PR number (not internal ID)
detection_method          | varchar(50)           | deep_link_finder | github_timeline_verified | explicit_bidirectional
final_confidence          | numeric(4,3)          | 0.000-1.000, CHECK constraint
link_quality              | varchar(20)           | high | medium | low
confidence_breakdown      | jsonb                 | See breakdown structure below
evidence_sources          | text[]                | Array of evidence identifiers
comprehensive_rationale   | text                  | Human-readable explanation
semantic_analysis         | jsonb                 | LLM analysis results (nullable)
temporal_analysis         | jsonb                 | Timing correlation data (nullable)
flags                     | jsonb                 | Warnings, negative signals
metadata                  | jsonb                 | Phase provenance, timestamps
created_at                | timestamp             | Default: now()
updated_at                | timestamp             | Default: now()
```

**Indexes:**
- Primary key: `id`
- Unique constraint: `(repo_id, issue_number, pr_number)`
- B-tree: `repo_id`, `issue_number`, `pr_number`, `detection_method`
- B-tree DESC: `final_confidence`

**confidence_breakdown JSONB structure:**
```json
{
  "BaseConfidence": 0.850,
  "BidirectionalBoost": 0.100,
  "SemanticBoost": 0.080,
  "TemporalBoost": 0.150,
  "FileContextBoost": 0.050,
  "SharedPRPenalty": 0.000,
  "NegativeSignalPenalty": 0.000
}
```

---

### github_issue_timeline
Timeline events from GitHub Issues Timeline API.

```sql
Column            | Type                  | Notes
------------------|-----------------------|----------------------------------
id                | bigint                | Primary key
issue_id          | bigint                | FK to github_issues (internal ID)
event_type        | varchar(50)           | referenced | cross-referenced | closed | labeled | etc.
created_at        | timestamp             | Event timestamp from GitHub
source_type       | varchar(20)           | pull_request | issue | null
source_number     | integer               | PR/Issue number (nullable)
source_sha        | varchar(40)           | Commit SHA for 'referenced' events
source_title      | text                  | PR/Issue title (nullable)
source_body       | text                  | PR/Issue body (nullable)
source_state      | varchar(20)           | open | closed | merged
source_merged_at  | timestamp             | PR merge time (nullable)
actor_login       | varchar(255)          | GitHub username
actor_id          | bigint                | GitHub user ID
raw_data          | jsonb                 | Full API response
fetched_at        | timestamp             | Default: now()
processed_at      | timestamp             | Linking system processing time
```

**Indexes:**
- Primary key: `id`
- Unique constraint: `(issue_id, event_type, created_at, source_number)`
- B-tree: `issue_id`, `event_type`, `source_type`, `source_number`
- Partial index: `processed_at WHERE processed_at IS NULL`

**Key event types for linking:**
- `cross-referenced`: Direct PR/Issue cross-reference (rare in practice)
- `referenced`: Commit reference (needs resolution to PR via commits API)
- `closed`: Issue closed event

---

### github_issues
GitHub Issues metadata.

```sql
Column              | Type                  | Notes
--------------------|-----------------------|----------------------------------
id                  | bigint                | Primary key (internal)
repo_id             | bigint                | FK to github_repositories
github_id           | bigint                | GitHub's global issue ID
number              | integer               | Issue number (#123)
title               | text                  | Issue title
body                | text                  | Issue description (nullable)
state               | varchar(20)           | open | closed
user_login          | varchar(255)          | Issue author username
user_id             | bigint                | Issue author GitHub ID
author_association  | varchar(50)           | OWNER | CONTRIBUTOR | etc.
labels              | jsonb                 | Array of label objects
assignees           | jsonb                 | Array of assignee objects
created_at          | timestamp             | Issue creation time
updated_at          | timestamp             | Last modified time
closed_at           | timestamp             | Issue close time (nullable)
comments_count      | integer               | Default: 0
reactions_count     | integer               | Default: 0
raw_data            | jsonb                 | Full GitHub API response
fetched_at          | timestamp             | Default: now()
processed_at        | timestamp             | Graph processing timestamp
```

**Indexes:**
- Primary key: `id`
- Unique constraint: `(repo_id, number)`
- B-tree: `repo_id`, `github_id`, `state`
- B-tree DESC: `created_at`, `closed_at`
- GIN: `labels`, `assignees`, full-text on `title || body`
- Partial index: `(repo_id, processed_at) WHERE processed_at IS NULL`

---

### github_pull_requests
GitHub Pull Request metadata.

```sql
Column              | Type                  | Notes
--------------------|-----------------------|----------------------------------
id                  | bigint                | Primary key (internal)
repo_id             | bigint                | FK to github_repositories
github_id           | bigint                | GitHub's global PR ID
number              | integer               | PR number (#456)
title               | text                  | PR title
body                | text                  | PR description (nullable)
state               | varchar(20)           | open | closed
user_login          | varchar(255)          | PR author username
user_id             | bigint                | PR author GitHub ID
author_association  | varchar(50)           | OWNER | CONTRIBUTOR | etc.
head_ref            | varchar(255)          | Source branch name
head_sha            | varchar(40)           | Source branch commit SHA
base_ref            | varchar(255)          | Target branch name (usually main)
base_sha            | varchar(40)           | Target branch commit SHA
merged              | boolean               | Default: false
merged_at           | timestamp             | PR merge time (nullable)
merge_commit_sha    | varchar(40)           | Merge commit SHA (nullable)
draft               | boolean               | Default: false
labels              | jsonb                 | Array of label objects
created_at          | timestamp             | PR creation time
updated_at          | timestamp             | Last modified time
closed_at           | timestamp             | PR close time (nullable)
raw_data            | jsonb                 | Full GitHub API response
fetched_at          | timestamp             | Default: now()
processed_at        | timestamp             | Graph processing timestamp
```

**Indexes:**
- Primary key: `id`
- Unique constraint: `(repo_id, number)`
- B-tree: `repo_id`, `github_id`, `state`, `merged`
- B-tree DESC: `created_at`, `merged_at`
- B-tree partial: `merge_commit_sha WHERE merge_commit_sha IS NOT NULL`
- GIN: `labels`, full-text on `title || body`
- Partial index: `(repo_id, processed_at) WHERE processed_at IS NULL`

---

### github_commits
Git commit metadata.

```sql
Column            | Type                  | Notes
------------------|-----------------------|----------------------------------
id                | bigint                | Primary key
repo_id           | bigint                | FK to github_repositories
sha               | varchar(40)           | Git commit SHA (unique)
author_name       | varchar(255)          | Commit author name
author_email      | varchar(255)          | Commit author email
author_date       | timestamp             | Authorship timestamp
committer_name    | varchar(255)          | Committer name
committer_email   | varchar(255)          | Committer email
committer_date    | timestamp             | Commit timestamp
message           | text                  | Commit message
verified          | boolean               | GPG signature verification
additions         | integer               | Lines added
deletions         | integer               | Lines deleted
total_changes     | integer               | additions + deletions
files_changed     | integer               | Number of files modified
raw_data          | jsonb                 | Full GitHub API response
fetched_at        | timestamp             | Default: now()
processed_at      | timestamp             | Graph processing timestamp
```

**Indexes:**
- Primary key: `id`
- Unique constraint: `(repo_id, sha)`
- B-tree: `repo_id`, `sha`, `author_email`, `committer_email`
- B-tree DESC: `author_date`, `committer_date`

---

## Neo4j Graph Schema

### Node Labels

#### `:File`
Source code files.

```cypher
Properties:
  path: string              // Relative path from repo root
  historical: boolean       // false = current, true = past version
```

**Unique constraint**: `path` (for current files only)

---

#### `:Commit`
Git commits.

```cypher
Properties:
  sha: string               // Git commit SHA (unique)
  message: string           // Commit message
  author_email: string      // Commit author email
  committed_at: datetime    // Commit timestamp (ISO 8601)
  additions: integer        // Lines added
  deletions: integer        // Lines deleted
  on_default_branch: boolean // true = on main/master
```

**Unique constraint**: `sha`

---

#### `:Developer`
Contributors to the codebase.

```cypher
Properties:
  email: string             // Primary identifier (unique)
  name: string              // Developer name
  last_active: datetime     // Most recent commit timestamp
```

**Unique constraint**: `email`

---

#### `:PR` (Pull Request)
GitHub Pull Requests.

```cypher
Properties:
  number: integer           // PR number (#456)
  title: string             // PR title
  body: string              // PR description
  state: string             // "open" | "closed"
  author_email: string      // PR author email
  head_branch: string       // Source branch
  base_branch: string       // Target branch (usually "main")
  merge_commit_sha: string  // Merge commit SHA (nullable)
  merged_at: datetime       // Merge timestamp (nullable, ISO 8601)
  created_at: datetime      // PR creation timestamp (ISO 8601)
```

**Unique constraint**: `number`

---

#### `:Issue`
GitHub Issues.

```cypher
Properties:
  number: integer           // Issue number (#123)
  title: string             // Issue title
  body: string              // Issue description
  state: string             // "open" | "closed"
  labels: string[]          // Array of label names
  created_at: datetime      // Issue creation timestamp (ISO 8601)
  closed_at: datetime       // Issue close timestamp (nullable, ISO 8601)
```

**Unique constraint**: `number`

---

### Relationship Types

#### `(:File)-[:DEPENDS_ON]->(:File)`
File-level dependencies (import/require statements).

```cypher
Properties: none
```

---

#### `(:Developer)-[:AUTHORED {timestamp}]->(:Commit)`
Developer created a commit.

```cypher
Properties:
  timestamp: datetime       // Same as commit.committed_at
```

---

#### `(:Commit)-[:MODIFIED {additions, deletions, status}]->(:File)`
Commit modified a file.

```cypher
Properties:
  additions: integer        // Lines added to this file
  deletions: integer        // Lines deleted from this file
  status: string            // "added" | "modified" | "removed"
```

---

#### `(:Commit)-[:IN_PR]->(:PR)`
Commit belongs to a pull request.

```cypher
Properties: none
```

---

#### `(:Developer)-[:CREATED]->(:Issue)`
Developer created an issue.

```cypher
Properties: none
```

---

#### `(:Issue)-[:FIXED_BY]->(:PR)`
Issue was definitively fixed by PR (high confidence, multi-signal verification).

**Multi-Signal Ground Truth Criteria (ALL must be true):**
1. Detection method: `github_timeline_verified` OR `explicit_bidirectional`
2. Base confidence: ≥ 0.85
3. No negative signals: `negative_penalty = 0.0`
4. At least ONE ground truth signal:
   - Temporal boost ≥ 0.12 (closed within 1 hour of PR merge)
   - Bidirectional boost > 0.0 (cross-referenced in both directions)
   - Semantic boost ≥ 0.10 + explicit fixes keyword

```cypher
Properties:
  confidence: float                 // 0.850-0.999 (final confidence)
  detection_method: string          // "github_timeline_verified" | "explicit_bidirectional"
  link_quality: string              // "high" | "medium"
  evidence_sources: string[]        // ["temporal_normal", "semantic_title_match"]

  // Confidence breakdown (transparency for future review)
  base_confidence: float            // 0.850-0.950
  temporal_boost: float             // 0.000-0.150
  bidirectional_boost: float        // 0.000-0.100
  semantic_boost: float             // 0.000-0.150
  negative_penalty: float           // 0.000 (always zero for FIXED_BY)

  created_from: "validated_link"    // System identifier
```

---

#### `(:Issue)-[:ASSOCIATED_WITH]->(:PR)`
Issue is associated with PR but not definitively a fix (mentions, weak temporal correlation, or negative signals present).

**Created when:**
- Confidence ≥ 0.70 but doesn't meet FIXED_BY criteria
- Any detection method allowed
- May have negative signals or weak ground truth

```cypher
Properties:
  confidence: float                 // 0.700-0.999
  detection_method: string          // Any: "deep_link_finder" | "github_timeline_verified" | "explicit_bidirectional"
  link_quality: string              // "high" | "medium" | "low"
  evidence_sources: string[]        // Full evidence array

  // Confidence breakdown
  base_confidence: float            // Varies
  temporal_boost: float             // May be low/zero
  bidirectional_boost: float        // May be zero
  semantic_boost: float             // Varies
  negative_penalty: float           // May be non-zero (e.g., -0.050)

  created_from: "validated_link"
```

**Legacy ASSOCIATED_WITH edges** (from old system, pre-validation):
```cypher
Properties:
  confidence: float                 // 0.500-0.999
  detected_via: string              // "issue_commit_ref" | "temporal_semantic"
  relationship_type: string         // "fixes" | "mentions" | "discusses"
  evidence: string                  // Textual evidence
  rationale: string                 // Human-readable explanation
```

These will be replaced by validated links over time.

---

## Detection Method Taxonomy

| Method                    | Source                           | Base Confidence | Eligible for FIXED_BY? |
|---------------------------|----------------------------------|-----------------|------------------------|
| `github_timeline_verified`| Timeline API cross-reference     | 0.95            | ✅ Yes                 |
| `explicit_bidirectional`  | Bidirectional text references    | 0.90            | ✅ Yes                 |
| `deep_link_finder`        | LLM semantic analysis            | 0.70-0.85       | ❌ No                  |

---

## Link Quality Tiers

| Quality | Final Confidence Range | Description                                  |
|---------|------------------------|----------------------------------------------|
| High    | 0.85 - 1.00            | Multiple ground truth signals, no negatives  |
| Medium  | 0.70 - 0.84            | Some evidence, may have weak signals         |
| Low     | 0.50 - 0.69            | Filtered out from graph (too noisy)          |

---

## Evidence Source Identifiers

Used in `evidence_sources` array:

| Identifier               | Meaning                                           |
|--------------------------|---------------------------------------------------|
| `temporal_immediate`     | Closed within 5 minutes of PR merge (boost 0.15)  |
| `temporal_normal`        | Closed within 1 hour of PR merge (boost 0.12)     |
| `temporal_correlation`   | Closed within 24 hours (boost 0.05)               |
| `bidirectional_verified` | Cross-referenced in both issue and PR             |
| `semantic_title_match`   | Issue/PR titles highly similar                    |
| `semantic_body_match`    | Issue/PR bodies reference each other              |
| `ref_fixes`              | Explicit "fixes #123" keyword                     |
| `ref_closes`             | Explicit "closes #123" keyword                    |
| `ref_resolves`           | Explicit "resolves #123" keyword                  |
| `file_context_shared`    | Issue and PR modify same files                    |

---

## Negative Signals (Penalties)

| Signal                   | Penalty | Detected When                                    |
|--------------------------|---------|--------------------------------------------------|
| `shared_pr_ambiguity`    | -0.05   | Multiple issues claim same PR                    |
| `reopened_issue`         | -0.10   | Issue reopened after PR merge                    |
| `conflicting_evidence`   | -0.05   | Comments suggest different root cause            |

---

## Data Flow Summary

```
1. GitHub API → PostgreSQL Staging Tables
   ├─ github_issues
   ├─ github_pull_requests
   ├─ github_commits
   └─ github_issue_timeline

2. Issue-PR Linking Pipeline → github_issue_pr_links
   ├─ Phase 0: Timeline + DORA metrics
   ├─ Phase 1: Explicit reference extraction (LLM)
   └─ Phase 2: Semantic + temporal analysis

3. Graph Builder → Neo4j
   ├─ Nodes: File, Commit, Developer, PR, Issue
   ├─ Validated Links: FIXED_BY, ASSOCIATED_WITH (from github_issue_pr_links)
   └─ Other Edges: DEPENDS_ON, AUTHORED, MODIFIED, IN_PR, CREATED
```

---

## Key Constraints & Indexes

**PostgreSQL:**
- All timestamps are `timestamp without time zone` (assumes UTC)
- Foreign keys use `ON DELETE CASCADE` for clean repo deletion
- JSONB fields use GIN indexes for performance
- Unique constraints prevent duplicate API data

**Neo4j:**
- Unique constraints on primary identifiers (`sha`, `email`, `number`, `path`)
- No indexes on relationship properties (small graph size)
- All timestamps stored as ISO 8601 datetime strings

---

## Version Information

- PostgreSQL Schema Version: 1.2 (includes linking tables)
- Neo4j Schema Version: 1.1 (includes validated Issue-PR edges)
- Last Updated: 2025-11-07
