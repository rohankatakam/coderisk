-- ============================================
-- CodeRisk GitHub API Staging Schema
-- ============================================
-- Purpose: Store raw GitHub API JSON responses for idempotent graph construction
-- Database: PostgreSQL 15+ with JSONB support
-- Date: October 2, 2025
--
-- Design Principles:
-- 1. Store raw JSON in JSONB for flexibility and indexing
-- 2. Track fetch/process timestamps for incremental updates
-- 3. Enforce uniqueness constraints for idempotency
-- 4. Support fast lookup by repo + entity ID
-- 5. Enable partial index on unprocessed records
-- ============================================

-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================
-- Core Repository Table
-- ============================================

CREATE TABLE github_repositories (
    id BIGSERIAL PRIMARY KEY,

    -- Repository identification
    github_id BIGINT UNIQUE NOT NULL,              -- GitHub's numeric ID
    owner VARCHAR(255) NOT NULL,                   -- e.g., "omnara-ai"
    name VARCHAR(255) NOT NULL,                    -- e.g., "omnara"
    full_name VARCHAR(512) NOT NULL UNIQUE,        -- e.g., "omnara-ai/omnara"

    -- Local repository path (for converting relative <-> absolute paths)
    absolute_path TEXT,                             -- e.g., "/Users/.../omnara" (local clone location)

    -- Raw API response
    raw_data JSONB NOT NULL,

    -- Metadata
    fetched_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- Indexing
    CONSTRAINT unique_owner_name UNIQUE(owner, name)
);

CREATE INDEX idx_repositories_github_id ON github_repositories(github_id);
CREATE INDEX idx_repositories_full_name ON github_repositories(full_name);
CREATE INDEX idx_repositories_fetched_at ON github_repositories(fetched_at DESC);

-- ============================================
-- Layer 2: Temporal Data (Git History)
-- ============================================

-- Commits Table
CREATE TABLE github_commits (
    id BIGSERIAL PRIMARY KEY,

    -- Foreign key
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,

    -- Commit identification
    sha VARCHAR(40) NOT NULL,                       -- Git SHA-1 hash

    -- Extracted fields (for fast querying without JSON parsing)
    author_name VARCHAR(255),
    author_email VARCHAR(255),
    author_date TIMESTAMP,
    committer_name VARCHAR(255),
    committer_email VARCHAR(255),
    committer_date TIMESTAMP,
    message TEXT,
    verified BOOLEAN DEFAULT FALSE,

    -- Change statistics
    additions INTEGER DEFAULT 0,
    deletions INTEGER DEFAULT 0,
    total_changes INTEGER DEFAULT 0,
    files_changed INTEGER DEFAULT 0,

    -- Raw API response (includes files[], patch diffs, verification)
    raw_data JSONB NOT NULL,

    -- Processing metadata
    fetched_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,                         -- NULL = not yet processed into graph

    -- Constraints
    CONSTRAINT unique_repo_commit UNIQUE(repo_id, sha)
);

CREATE INDEX idx_commits_repo_id ON github_commits(repo_id);
CREATE INDEX idx_commits_sha ON github_commits(sha);
CREATE INDEX idx_commits_author_date ON github_commits(author_date DESC);
CREATE INDEX idx_commits_author_email ON github_commits(author_email);
CREATE INDEX idx_commits_processed ON github_commits(repo_id, processed_at) WHERE processed_at IS NULL;

-- JSONB index for fast file lookups
CREATE INDEX idx_commits_files ON github_commits USING GIN ((raw_data -> 'files'));

-- Developers Table (extracted from commits)
CREATE TABLE github_developers (
    id BIGSERIAL PRIMARY KEY,

    -- Foreign key
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,

    -- Developer identification
    email VARCHAR(255) NOT NULL,                    -- Primary identifier
    name VARCHAR(255),
    github_login VARCHAR(255),                      -- If available from API

    -- Statistics
    first_commit_at TIMESTAMP,
    last_commit_at TIMESTAMP,
    total_commits INTEGER DEFAULT 0,
    total_additions INTEGER DEFAULT 0,
    total_deletions INTEGER DEFAULT 0,

    -- Metadata
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- Constraints
    CONSTRAINT unique_repo_email UNIQUE(repo_id, email)
);

CREATE INDEX idx_developers_repo_id ON github_developers(repo_id);
CREATE INDEX idx_developers_email ON github_developers(email);
CREATE INDEX idx_developers_github_login ON github_developers(github_login);

-- Branches Table
CREATE TABLE github_branches (
    id BIGSERIAL PRIMARY KEY,

    -- Foreign key
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,

    -- Branch identification
    name VARCHAR(255) NOT NULL,
    commit_sha VARCHAR(40) NOT NULL,

    -- Protection status
    protected BOOLEAN DEFAULT FALSE,

    -- Raw API response
    raw_data JSONB NOT NULL,

    -- Metadata
    fetched_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,

    -- Constraints
    CONSTRAINT unique_repo_branch UNIQUE(repo_id, name)
);

CREATE INDEX idx_branches_repo_id ON github_branches(repo_id);
CREATE INDEX idx_branches_name ON github_branches(name);
CREATE INDEX idx_branches_commit_sha ON github_branches(commit_sha);

-- Trees Table (file structure snapshots)
CREATE TABLE github_trees (
    id BIGSERIAL PRIMARY KEY,

    -- Foreign key
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,

    -- Tree identification
    sha VARCHAR(40) NOT NULL,
    ref VARCHAR(255),                                -- Branch name or commit SHA

    -- Tree metadata
    truncated BOOLEAN DEFAULT FALSE,                 -- True if tree has >100k entries
    tree_count INTEGER DEFAULT 0,                    -- Number of entries

    -- Raw API response (includes tree[] array)
    raw_data JSONB NOT NULL,

    -- Metadata
    fetched_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,

    -- Constraints
    CONSTRAINT unique_repo_tree UNIQUE(repo_id, sha)
);

CREATE INDEX idx_trees_repo_id ON github_trees(repo_id);
CREATE INDEX idx_trees_sha ON github_trees(sha);
CREATE INDEX idx_trees_ref ON github_trees(ref);

-- JSONB index for fast file path lookups
CREATE INDEX idx_trees_entries ON github_trees USING GIN ((raw_data -> 'tree'));

-- ============================================
-- Layer 3: Incidents (Issues & Pull Requests)
-- ============================================

-- Issues Table
CREATE TABLE github_issues (
    id BIGSERIAL PRIMARY KEY,

    -- Foreign key
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,

    -- Issue identification
    github_id BIGINT NOT NULL,                       -- GitHub's issue ID
    number INTEGER NOT NULL,                         -- Issue number (#123)

    -- Extracted fields
    title TEXT,
    body TEXT,
    state VARCHAR(20),                               -- "open" or "closed"

    -- User information
    user_login VARCHAR(255),
    user_id BIGINT,
    author_association VARCHAR(50),                  -- OWNER/CONTRIBUTOR/NONE

    -- Labels (extracted for fast querying)
    labels JSONB,                                    -- Array of label names

    -- Assignment
    assignees JSONB,                                 -- Array of assignee logins

    -- Temporal data
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    closed_at TIMESTAMP,

    -- Engagement metrics
    comments_count INTEGER DEFAULT 0,
    reactions_count INTEGER DEFAULT 0,

    -- Raw API response
    raw_data JSONB NOT NULL,

    -- Processing metadata
    fetched_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,

    -- Constraints
    CONSTRAINT unique_repo_issue UNIQUE(repo_id, number)
);

CREATE INDEX idx_issues_repo_id ON github_issues(repo_id);
CREATE INDEX idx_issues_number ON github_issues(repo_id, number);
CREATE INDEX idx_issues_github_id ON github_issues(github_id);
CREATE INDEX idx_issues_state ON github_issues(state);
CREATE INDEX idx_issues_created_at ON github_issues(created_at DESC);
CREATE INDEX idx_issues_closed_at ON github_issues(closed_at DESC) WHERE closed_at IS NOT NULL;
CREATE INDEX idx_issues_processed ON github_issues(repo_id, processed_at) WHERE processed_at IS NULL;

-- JSONB indexes for label and assignee filtering
CREATE INDEX idx_issues_labels ON github_issues USING GIN (labels);
CREATE INDEX idx_issues_assignees ON github_issues USING GIN (assignees);

-- Full-text search on title and body
CREATE INDEX idx_issues_text ON github_issues USING GIN (to_tsvector('english', COALESCE(title, '') || ' ' || COALESCE(body, '')));

-- Pull Requests Table
CREATE TABLE github_pull_requests (
    id BIGSERIAL PRIMARY KEY,

    -- Foreign key
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,

    -- PR identification
    github_id BIGINT NOT NULL,                       -- GitHub's PR ID
    number INTEGER NOT NULL,                         -- PR number (#123)

    -- Extracted fields
    title TEXT,
    body TEXT,
    state VARCHAR(20),                               -- "open" or "closed"

    -- User information
    user_login VARCHAR(255),
    user_id BIGINT,
    author_association VARCHAR(50),

    -- Branch information
    head_ref VARCHAR(255),                           -- Source branch
    head_sha VARCHAR(40),                            -- Source commit
    base_ref VARCHAR(255),                           -- Target branch (usually "main")
    base_sha VARCHAR(40),                            -- Target commit

    -- Merge information
    merged BOOLEAN DEFAULT FALSE,
    merged_at TIMESTAMP,
    merge_commit_sha VARCHAR(40),                    -- CRITICAL: Links PR to commit

    -- Draft status
    draft BOOLEAN DEFAULT FALSE,

    -- Labels
    labels JSONB,

    -- Temporal data
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    closed_at TIMESTAMP,

    -- Raw API response
    raw_data JSONB NOT NULL,

    -- Processing metadata
    fetched_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,

    -- Constraints
    CONSTRAINT unique_repo_pr UNIQUE(repo_id, number)
);

CREATE INDEX idx_prs_repo_id ON github_pull_requests(repo_id);
CREATE INDEX idx_prs_number ON github_pull_requests(repo_id, number);
CREATE INDEX idx_prs_github_id ON github_pull_requests(github_id);
CREATE INDEX idx_prs_state ON github_pull_requests(state);
CREATE INDEX idx_prs_merged ON github_pull_requests(merged);
CREATE INDEX idx_prs_merge_commit_sha ON github_pull_requests(merge_commit_sha) WHERE merge_commit_sha IS NOT NULL;
CREATE INDEX idx_prs_created_at ON github_pull_requests(created_at DESC);
CREATE INDEX idx_prs_merged_at ON github_pull_requests(merged_at DESC) WHERE merged_at IS NOT NULL;
CREATE INDEX idx_prs_processed ON github_pull_requests(repo_id, processed_at) WHERE processed_at IS NULL;

-- JSONB index for labels
CREATE INDEX idx_prs_labels ON github_pull_requests USING GIN (labels);

-- Full-text search on title and body
CREATE INDEX idx_prs_text ON github_pull_requests USING GIN (to_tsvector('english', COALESCE(title, '') || ' ' || COALESCE(body, '')));

-- ============================================
-- Metadata Tables
-- ============================================

-- Languages Table (per repository)
CREATE TABLE github_languages (
    id BIGSERIAL PRIMARY KEY,

    -- Foreign key
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,

    -- Language data
    languages JSONB NOT NULL,                        -- { "TypeScript": 1463990, "Python": 977372, ... }

    -- Metadata
    fetched_at TIMESTAMP DEFAULT NOW(),

    -- Constraints
    CONSTRAINT unique_repo_languages UNIQUE(repo_id)
);

CREATE INDEX idx_languages_repo_id ON github_languages(repo_id);
CREATE INDEX idx_languages_data ON github_languages USING GIN (languages);

-- Contributors Table
CREATE TABLE github_contributors (
    id BIGSERIAL PRIMARY KEY,

    -- Foreign key
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,

    -- Contributor identification
    github_id BIGINT NOT NULL,
    login VARCHAR(255) NOT NULL,

    -- Statistics
    contributions INTEGER DEFAULT 0,

    -- Raw API response
    raw_data JSONB NOT NULL,

    -- Metadata
    fetched_at TIMESTAMP DEFAULT NOW(),

    -- Constraints
    CONSTRAINT unique_repo_contributor UNIQUE(repo_id, github_id)
);

CREATE INDEX idx_contributors_repo_id ON github_contributors(repo_id);
CREATE INDEX idx_contributors_login ON github_contributors(login);
CREATE INDEX idx_contributors_contributions ON github_contributors(contributions DESC);

-- ============================================
-- Views for Graph Construction
-- ============================================

-- View: Unprocessed commits for graph ingestion
CREATE OR REPLACE VIEW v_unprocessed_commits AS
SELECT
    c.id,
    c.repo_id,
    r.full_name AS repo_full_name,
    c.sha,
    c.author_email,
    c.author_name,
    c.author_date,
    c.message,
    c.raw_data
FROM github_commits c
JOIN github_repositories r ON c.repo_id = r.id
WHERE c.processed_at IS NULL
ORDER BY c.author_date ASC;

-- View: Unprocessed issues for graph ingestion
CREATE OR REPLACE VIEW v_unprocessed_issues AS
SELECT
    i.id,
    i.repo_id,
    r.full_name AS repo_full_name,
    i.number,
    i.title,
    i.body,
    i.state,
    i.labels,
    i.created_at,
    i.closed_at,
    i.raw_data
FROM github_issues i
JOIN github_repositories r ON i.repo_id = r.id
WHERE i.processed_at IS NULL
ORDER BY i.created_at ASC;

-- View: Unprocessed PRs for graph ingestion
CREATE OR REPLACE VIEW v_unprocessed_prs AS
SELECT
    p.id,
    p.repo_id,
    r.full_name AS repo_full_name,
    p.number,
    p.title,
    p.body,
    p.state,
    p.merged,
    p.merge_commit_sha,
    p.created_at,
    p.merged_at,
    p.raw_data
FROM github_pull_requests p
JOIN github_repositories r ON p.repo_id = r.id
WHERE p.processed_at IS NULL
ORDER BY p.created_at ASC;

-- ============================================
-- Helper Functions
-- ============================================

-- Function: Mark commit as processed
CREATE OR REPLACE FUNCTION mark_commit_processed(commit_id BIGINT)
RETURNS VOID AS $$
BEGIN
    UPDATE github_commits
    SET processed_at = NOW()
    WHERE id = commit_id;
END;
$$ LANGUAGE plpgsql;

-- Function: Mark issue as processed
CREATE OR REPLACE FUNCTION mark_issue_processed(issue_id BIGINT)
RETURNS VOID AS $$
BEGIN
    UPDATE github_issues
    SET processed_at = NOW()
    WHERE id = issue_id;
END;
$$ LANGUAGE plpgsql;

-- Function: Mark PR as processed
CREATE OR REPLACE FUNCTION mark_pr_processed(pr_id BIGINT)
RETURNS VOID AS $$
BEGIN
    UPDATE github_pull_requests
    SET processed_at = NOW()
    WHERE id = pr_id;
END;
$$ LANGUAGE plpgsql;

-- Function: Get repository by full name
CREATE OR REPLACE FUNCTION get_repository_id(repo_full_name VARCHAR)
RETURNS BIGINT AS $$
DECLARE
    repo_id BIGINT;
BEGIN
    SELECT id INTO repo_id
    FROM github_repositories
    WHERE full_name = repo_full_name;

    RETURN repo_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- Sample Queries
-- ============================================

-- Get all unprocessed commits for a repository
-- SELECT * FROM v_unprocessed_commits WHERE repo_full_name = 'omnara-ai/omnara' LIMIT 100;

-- Get commits that modify a specific file
-- SELECT sha, author_email, author_date, message
-- FROM github_commits
-- WHERE repo_id = 1
-- AND raw_data -> 'files' @> '[{"filename": "src/relay_server/config.py"}]';

-- Get all open issues with "bug" label
-- SELECT number, title, created_at
-- FROM github_issues
-- WHERE repo_id = 1
-- AND state = 'open'
-- AND labels @> '["bug"]'::jsonb;

-- Find PRs that were merged in the last 90 days
-- SELECT number, title, merged_at, merge_commit_sha
-- FROM github_pull_requests
-- WHERE repo_id = 1
-- AND merged = TRUE
-- AND merged_at > NOW() - INTERVAL '90 days';

-- Find commits by a specific developer
-- SELECT sha, author_date, message
-- FROM github_commits
-- WHERE repo_id = 1
-- AND author_email = 'kartiksarangmath@gmail.com'
-- ORDER BY author_date DESC;

-- ============================================
-- Performance Notes
-- ============================================

-- 1. JSONB indexes allow fast querying without parsing JSON
-- 2. Partial indexes on processed_at reduce index size for active queries
-- 3. GIN indexes on JSONB arrays enable fast containment searches
-- 4. Foreign key CASCADE ensures data integrity when deleting repositories
-- 5. Views provide clean interface for graph construction workers

-- ============================================
-- Storage Estimates (from DATA_VOLUME_ANALYSIS.md)
-- ============================================

-- omnara-ai/omnara:
--   - Total records: ~650 (100 commits, 251 issues, 180 PRs, etc.)
--   - JSON data: ~2.8 MB
--   - Indexes: ~0.8 MB
--   - Total: ~3.6 MB

-- kubernetes/kubernetes:
--   - Total records: ~12,500 (5k commits, 5k issues, 2k PRs, etc.)
--   - JSON data: ~83 MB
--   - Indexes: ~25 MB
--   - Total: ~108 MB
