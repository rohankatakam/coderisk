# Current PostgreSQL Schema Overview
## CodeRisk Staging Database - Complete Reference

**Date:** 2025-11-14
**Database:** `coderisk`
**Purpose:** Staging layer for GitHub data → Neo4j graph construction

---

## 📊 Schema Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    CORE REPOSITORY DATA                          │
└─────────────────────────────────────────────────────────────────┘
                              ↓
        ┌─────────────────────┴─────────────────────┐
        │                                           │
┌───────▼────────┐                      ┌───────────▼──────────┐
│   COMMIT DATA   │                      │    INCIDENT DATA     │
│  (Temporal)     │                      │  (Issues + PRs)      │
└────────┬────────┘                      └──────────┬───────────┘
         │                                           │
         │                    ┌──────────────────────┤
         │                    │                      │
    ┌────▼─────┐        ┌─────▼──────┐      ┌───────▼──────┐
    │ Files &  │        │  Timeline  │      │   Linking    │
    │ Patches  │        │   Events   │      │   Tables     │
    └──────────┘        └────────────┘      └──────────────┘
```

---

## 🗃️ Table Reference

### 1. **github_repositories** (Root Table)
**Purpose:** Master list of all repositories being tracked

| Column | Type | Purpose |
|--------|------|---------|
| `id` | BIGSERIAL | Internal primary key |
| `github_id` | BIGINT | GitHub's internal repo ID |
| `owner` | VARCHAR(255) | Repository owner (e.g., "omnara-ai") |
| `name` | VARCHAR(255) | Repository name (e.g., "omnara") |
| `full_name` | VARCHAR(512) | "owner/name" format |
| `absolute_path` | TEXT | Local path if cloned |
| `raw_data` | JSONB | **Full GitHub API response** |
| `fetched_at` | TIMESTAMP | When data was pulled from GitHub |
| `updated_at` | TIMESTAMP | Last update timestamp |

**Key Relationships:**
- Parent to ALL other tables via `repo_id` foreign keys
- One repository → Many commits, issues, PRs, etc.

---

### 2. **github_commits** ⭐ CRITICAL FOR CODE-BLOCK PIPELINE
**Purpose:** Complete commit history with **full patch/diff data**

| Column | Type | Purpose |
|--------|------|---------|
| `id` | BIGSERIAL | Internal primary key |
| `repo_id` | BIGINT FK | → github_repositories.id |
| `sha` | VARCHAR(40) | Git commit SHA (unique) |
| `author_name` | VARCHAR(255) | Commit author name |
| `author_email` | VARCHAR(255) | **Developer identity** |
| `author_date` | TIMESTAMP | When commit was authored |
| `committer_name` | VARCHAR(255) | Who committed (if different) |
| `committer_email` | VARCHAR(255) | Committer email |
| `committer_date` | TIMESTAMP | When commit was pushed |
| `message` | TEXT | Commit message |
| `verified` | BOOLEAN | GPG signature status |
| `additions` | INTEGER | Lines added across all files |
| `deletions` | INTEGER | Lines deleted across all files |
| `total_changes` | INTEGER | additions + deletions |
| `files_changed` | INTEGER | Number of files modified |
| `raw_data` | JSONB | **⭐ CONTAINS PATCH DATA** |
| `fetched_at` | TIMESTAMP | When fetched from GitHub |
| `processed_at` | TIMESTAMP | When processed into Neo4j |

**Critical JSONB Structure (`raw_data`):**
```json
{
  "sha": "a255b601da5...",
  "files": [
    {
      "filename": "src/TableEditor.tsx",
      "status": "modified",  // or "added", "deleted", "renamed"
      "additions": 10,
      "deletions": 3,
      "changes": 13,
      "patch": "@@ -42,7 +42,10 @@ export function updateTableEditor() {\n-  old line\n+  new line",
      "sha": "27b9522b80f...",
      "blob_url": "https://github.com/.../blob/...",
      "raw_url": "https://github.com/.../raw/..."
    }
  ],
  "stats": {
    "additions": 10,
    "deletions": 3,
    "total": 13
  }
}
```

**🎯 Pipeline Usage:**
- **Pipeline 2 (Code-Block Atomization):**
  - Reads `raw_data->'files'[n]->>'patch'` to extract diffs
  - Feeds patches to LLM to identify modified functions/methods
  - Creates `CodeBlock` entities from this data

---

### 3. **github_issues** (Incidents)
**Purpose:** All GitHub issues (bugs, feature requests, incidents)

| Column | Type | Purpose |
|--------|------|---------|
| `id` | BIGSERIAL | Internal primary key |
| `repo_id` | BIGINT FK | → github_repositories.id |
| `github_id` | BIGINT | GitHub's issue ID |
| `number` | INTEGER | Issue number (#123) |
| `title` | TEXT | Issue title |
| `body` | TEXT | **Full issue description** |
| `state` | VARCHAR(20) | "open" or "closed" |
| `user_login` | VARCHAR(255) | Who created the issue |
| `user_id` | BIGINT | GitHub user ID |
| `author_association` | VARCHAR(50) | Role (OWNER, CONTRIBUTOR, etc.) |
| `labels` | JSONB | Array of labels (["bug", "critical"]) |
| `assignees` | JSONB | Array of assigned users |
| `created_at` | TIMESTAMP | When issue was opened |
| `updated_at` | TIMESTAMP | Last update |
| `closed_at` | TIMESTAMP | **When incident was resolved** |
| `comments_count` | INTEGER | Number of comments |
| `reactions_count` | INTEGER | Reactions count |
| `raw_data` | JSONB | Full GitHub API response |
| `fetched_at` | TIMESTAMP | When fetched |
| `processed_at` | TIMESTAMP | When processed to Neo4j |

**🎯 Pipeline Usage:**
- **Pipeline 1 (Link Resolution):**
  - LLM reads `body` and comments to find commit/PR references
  - Creates Issue→Commit and Issue→PR links
- **Pipeline 3 (Risk Calculation):**
  - Links issues to code blocks that caused them
  - Powers the `WAS_ROOT_CAUSE_IN` relationship

---

### 4. **github_pull_requests** (PRs)
**Purpose:** All pull requests with merge data

| Column | Type | Purpose |
|--------|------|---------|
| `id` | BIGSERIAL | Internal primary key |
| `repo_id` | BIGINT FK | → github_repositories.id |
| `github_id` | BIGINT | GitHub's PR ID |
| `number` | INTEGER | PR number (#456) |
| `title` | TEXT | PR title |
| `body` | TEXT | **PR description** |
| `state` | VARCHAR(20) | "open", "closed" |
| `user_login` | VARCHAR(255) | PR author |
| `user_id` | BIGINT | GitHub user ID |
| `author_association` | VARCHAR(50) | Role |
| `head_ref` | VARCHAR(255) | Source branch name |
| `head_sha` | VARCHAR(40) | Source branch SHA |
| `base_ref` | VARCHAR(255) | Target branch name |
| `base_sha` | VARCHAR(40) | Target branch SHA |
| `merged` | BOOLEAN | Was PR merged? |
| `merged_at` | TIMESTAMP | **When PR was merged** |
| `merge_commit_sha` | VARCHAR(40) | **⭐ Links PR to Commit** |
| `draft` | BOOLEAN | Draft PR? |
| `labels` | JSONB | Array of labels |
| `created_at` | TIMESTAMP | PR creation time |
| `updated_at` | TIMESTAMP | Last update |
| `closed_at` | TIMESTAMP | When PR was closed |
| `raw_data` | JSONB | Full GitHub API response |
| `fetched_at` | TIMESTAMP | When fetched |
| `processed_at` | TIMESTAMP | When processed |

**🎯 Pipeline Usage:**
- **Link Resolution:**
  - `merge_commit_sha` creates 100% confidence PR→Commit edge
  - `body` scanned by LLM for issue references

---

### 5. **github_issue_timeline** (Timeline Events)
**Purpose:** All events in an issue's lifecycle (comments, cross-refs, closures)

| Column | Type | Purpose |
|--------|------|---------|
| `id` | BIGSERIAL | Internal primary key |
| `issue_id` | BIGINT FK | → github_issues.id |
| `event_type` | VARCHAR(50) | "cross-referenced", "closed", "commented", "labeled" |
| `created_at` | TIMESTAMP | When event occurred |
| `source_type` | VARCHAR(20) | "pull_request", "commit", "issue" |
| `source_number` | INTEGER | **PR/Issue number that referenced this** |
| `source_sha` | VARCHAR(40) | **Commit SHA that closed this** |
| `source_title` | TEXT | Title of source |
| `source_body` | TEXT | Body of source |
| `source_state` | VARCHAR(20) | State of source |
| `source_merged_at` | TIMESTAMP | If source was a merged PR |
| `actor_login` | VARCHAR(255) | Who performed the action |
| `actor_id` | BIGINT | GitHub user ID |
| `raw_data` | JSONB | Full event data |
| `fetched_at` | TIMESTAMP | When fetched |
| `processed_at` | TIMESTAMP | When processed |

**🎯 Pipeline Usage:**
- **100% Confidence Edges:**
  - `event_type='cross-referenced' AND source_type='pull_request'` → REFERENCES edge (Issue→PR)
  - `event_type='closed' AND source_sha IS NOT NULL` → CLOSED_BY edge (Issue→Commit)
- **Note:** Omnara data currently lacks `source_*` fields (GitHub API limitation)

---

### 6. **github_issue_comments** (Comment History)
**Purpose:** All comments on issues and PRs

| Column | Type | Purpose |
|--------|------|---------|
| `id` | BIGSERIAL | Internal primary key |
| `repo_id` | BIGINT FK | → github_repositories.id |
| `issue_id` | BIGINT FK | → github_issues.id |
| `github_id` | BIGINT | GitHub's comment ID |
| `body` | TEXT | **Comment text** |
| `user_login` | VARCHAR(255) | Commenter |
| `user_id` | BIGINT | GitHub user ID |
| `author_association` | VARCHAR(50) | Role |
| `created_at` | TIMESTAMP | When posted |
| `updated_at` | TIMESTAMP | Last edit |
| `raw_data` | JSONB | Full GitHub response |
| `fetched_at` | TIMESTAMP | When fetched |
| `processed_at` | TIMESTAMP | When processed |

**🎯 Pipeline Usage:**
- **Link Resolution:**
  - LLM scans `body` for references like "fixed in #123" or "see PR #456"
  - Supplements missing timeline data

---

### 7. **github_issue_commit_refs** (LLM-Extracted References)
**Purpose:** Issue→Commit/PR references found by LLM extraction

| Column | Type | Purpose |
|--------|------|---------|
| `id` | BIGSERIAL | Internal primary key |
| `repo_id` | BIGINT FK | → github_repositories.id |
| `issue_number` | INTEGER | Issue number |
| `commit_sha` | VARCHAR(40) | Referenced commit SHA |
| `pr_number` | INTEGER | Referenced PR number |
| `action` | VARCHAR(20) | "fixes", "mentions", "closes", "resolves" |
| `confidence` | DOUBLE | 0.0-1.0 confidence score |
| `detection_method` | VARCHAR(50) | "llm", "regex", "timeline" |
| `extracted_from` | VARCHAR(50) | "issue_body", "comment", "pr_body" |
| `extracted_at` | TIMESTAMP | When extracted |
| `created_at` | TIMESTAMP | Row creation |
| `evidence` | ARRAY | Text snippets as evidence |

**🎯 Pipeline Usage:**
- **Link Resolution Output:** This is where LLM extractions are stored
- **Graph Construction:** Read this to create ASSOCIATED_WITH edges

---

### 8. **github_issue_pr_links** (Validated Issue↔PR Links)
**Purpose:** Multi-signal validated links between issues and PRs

| Column | Type | Purpose |
|--------|------|---------|
| `id` | BIGSERIAL | Internal primary key |
| `repo_id` | BIGINT FK | → github_repositories.id |
| `issue_number` | INTEGER | Issue number |
| `pr_number` | INTEGER | PR number |
| `detection_method` | VARCHAR(50) | "github_timeline", "explicit_bidir", etc. |
| `final_confidence` | NUMERIC | **Final confidence score (0.70-1.00)** |
| `link_quality` | VARCHAR(20) | "high", "medium", "low" |
| `confidence_breakdown` | JSONB | Detailed scoring breakdown |
| `evidence_sources` | ARRAY | List of evidence sources |
| `comprehensive_rationale` | TEXT | **Why this link exists** |
| `semantic_analysis` | JSONB | LLM semantic analysis |
| `temporal_analysis` | JSONB | Temporal correlation data |
| `flags` | JSONB | Special flags |
| `metadata` | JSONB | Additional metadata |
| `created_at` | TIMESTAMP | Row creation |
| `updated_at` | TIMESTAMP | Last update |

**🎯 Pipeline Usage:**
- **Graph Construction:**
  - Used to create high-confidence FIXED_BY vs ASSOCIATED_WITH edges
  - Multi-signal ground truth classification

---

### 9. **github_issue_no_links** (Orphaned Issues)
**Purpose:** Issues that couldn't be linked to any commits/PRs

| Column | Type | Purpose |
|--------|------|---------|
| `id` | BIGSERIAL | Internal primary key |
| `repo_id` | BIGINT FK | → github_repositories.id |
| `issue_number` | INTEGER | Issue number |
| `no_links_reason` | VARCHAR(50) | Why no links found |
| `classification` | VARCHAR(50) | Issue type classification |
| `classification_confidence` | NUMERIC | Confidence in classification |
| `classification_rationale` | TEXT | Why this classification |
| `conversation_summary` | TEXT | LLM summary of discussion |
| `candidates_evaluated` | INTEGER | How many potential links checked |
| `best_candidate_score` | NUMERIC | Best match score (if any) |
| `safety_brake_reason` | TEXT | Why link was rejected |
| `issue_closed_at` | TIMESTAMP | When issue closed |
| `analyzed_at` | TIMESTAMP | When analysis run |
| `created_at` | TIMESTAMP | Row creation |
| `updated_at` | TIMESTAMP | Last update |

**🎯 Pipeline Usage:**
- **Link Resolution:** Tracks issues that couldn't be linked
- **Quality Assurance:** Helps identify gaps in linking

---

### 10. **github_pr_files** (PR File Changes)
**Purpose:** Individual files modified in each PR

| Column | Type | Purpose |
|--------|------|---------|
| `id` | BIGSERIAL | Internal primary key |
| `repo_id` | BIGINT FK | → github_repositories.id |
| `pr_id` | BIGINT FK | → github_pull_requests.id |
| `filename` | TEXT | File path |
| `status` | VARCHAR(20) | "modified", "added", "deleted", "renamed" |
| `additions` | INTEGER | Lines added |
| `deletions` | INTEGER | Lines deleted |
| `changes` | INTEGER | Total changes |
| `previous_filename` | TEXT | If renamed |
| `patch` | TEXT | **⭐ File-level diff** |
| `blob_url` | TEXT | GitHub blob URL |
| `raw_url` | TEXT | Raw file URL |
| `raw_data` | JSONB | Full GitHub response |
| `fetched_at` | TIMESTAMP | When fetched |

**🎯 Pipeline Usage:**
- **Code-Block Extraction:** Alternative source of patch data
- **PR Analysis:** Understand what files changed in a PR

---

### 11-15. **Supporting Tables**

| Table | Purpose |
|-------|---------|
| `github_branches` | Branch tracking (for multi-branch repos) |
| `github_contributors` | Contributor statistics |
| `github_developers` | Aggregated developer activity metrics |
| `github_dora_metrics` | DORA metrics (lead time, deployment frequency) |
| `github_languages` | Language breakdown for repo |
| `github_trees` | Git tree objects (for full repo snapshots) |

---

## 🔗 Relationship Diagram

```
github_repositories (1)
    ├─→ github_commits (N) ← github_issue_commit_refs
    ├─→ github_issues (N)
    │       ├─→ github_issue_timeline (N)
    │       ├─→ github_issue_comments (N)
    │       ├─→ github_issue_pr_links (N)
    │       └─→ github_issue_no_links (N)
    ├─→ github_pull_requests (N)
    │       └─→ github_pr_files (N)
    ├─→ github_branches (N)
    ├─→ github_contributors (N)
    ├─→ github_developers (N)
    ├─→ github_dora_metrics (N)
    ├─→ github_languages (N)
    └─→ github_trees (N)
```

---

## ⭐ Key Data for Code-Block Pipeline

| What You Need | Where It Lives | Coverage |
|---------------|----------------|----------|
| **Commit patches/diffs** | `github_commits.raw_data->'files'[n]->>'patch'` | **100%** ✅ |
| **File paths** | `github_commits.raw_data->'files'[n]->>'filename'` | **100%** ✅ |
| **Commit SHAs** | `github_commits.sha` | **100%** ✅ |
| **Issue descriptions** | `github_issues.body` | **100%** ✅ |
| **Issue→Commit links** | `github_issue_commit_refs` | Via LLM extraction |
| **Issue→PR links** | `github_issue_pr_links` | Multi-signal validation |
| **PR→Commit links** | `github_pull_requests.merge_commit_sha` | **43.8%** ✅ |

---

## 🎯 What's Missing for Code-Block Pipeline

You need **4 new tables** to store the atomized code-block data:

1. **`code_blocks`** - Individual functions/methods extracted from commits
2. **`code_block_modifications`** - History of changes to each block
3. **`code_block_incidents`** - Links blocks to incidents they caused
4. **`code_block_co_changes`** - Coupling risk between blocks

**See:** `migrations/001_code_block_schema.sql` for the proposed schema

---

## 📈 Current Data Quality

- **Patch Coverage:** 100% (all 3 commits have full diffs) ✅
- **Issue Metadata:** 100% (all 41 issues have descriptions) ✅
- **Timeline Events:** Limited (no cross-reference metadata) ⚠️
- **PR Merge Links:** 43.8% (7/16 PRs have merge commits) ✅

**Verdict:** Schema is **production-ready** for code-block pipeline once new tables are added.

---

**Document Version:** 1.0
**Last Updated:** 2025-11-14
