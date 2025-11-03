-- ============================================
-- Migration: Add PR Files and Issue Comments Tables
-- ============================================
-- Purpose: Support temporal-semantic issue linking
-- Date: November 2, 2025
-- Required for: LLM-based semantic matching using file context
--
-- Background:
-- The temporal-semantic linker needs:
-- 1. PR file changes for semantic matching (which files were modified)
-- 2. Issue comments for comment-based linking (maintainer links in comments)
-- ============================================

-- ============================================
-- PR Files Table
-- ============================================
-- Stores file-level changes for each pull request
-- Fetched via: GET /repos/{owner}/{repo}/pulls/{number}/files
-- Reference: TEMPORAL_SEMANTIC_LINKING.md lines 88-113

CREATE TABLE IF NOT EXISTS github_pr_files (
    id BIGSERIAL PRIMARY KEY,

    -- Foreign keys
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    pr_id BIGINT NOT NULL REFERENCES github_pull_requests(id) ON DELETE CASCADE,

    -- File identification
    filename TEXT NOT NULL,                      -- e.g., "src/lib/agents/AgentManager.ts"
    status VARCHAR(20) NOT NULL,                 -- "added", "modified", "removed", "renamed"

    -- Change statistics
    additions INTEGER DEFAULT 0,
    deletions INTEGER DEFAULT 0,
    changes INTEGER DEFAULT 0,

    -- Rename handling
    previous_filename TEXT,                      -- For renamed files

    -- Optional patch data (can be large, NULLABLE)
    patch TEXT,                                  -- Actual diff patch (optional, may be NULL for large files)

    -- Blob URLs
    blob_url TEXT,
    raw_url TEXT,

    -- Raw API response
    raw_data JSONB NOT NULL,

    -- Metadata
    fetched_at TIMESTAMP DEFAULT NOW(),

    -- Constraints
    CONSTRAINT unique_pr_file UNIQUE(pr_id, filename)
);

CREATE INDEX idx_pr_files_repo_id ON github_pr_files(repo_id);
CREATE INDEX idx_pr_files_pr_id ON github_pr_files(pr_id);
CREATE INDEX idx_pr_files_filename ON github_pr_files(filename);
CREATE INDEX idx_pr_files_status ON github_pr_files(status);
CREATE INDEX idx_pr_files_changes ON github_pr_files(changes DESC);

-- GIN index for fast filename pattern matching
CREATE INDEX idx_pr_files_filename_trgm ON github_pr_files USING GIN (filename gin_trgm_ops);

-- ============================================
-- Issue Comments Table
-- ============================================
-- Stores comments on issues (for comment-based linking)
-- Fetched via: GET /repos/{owner}/{repo}/issues/{number}/comments
-- Reference: Stagehand ground truth test case #1060

CREATE TABLE IF NOT EXISTS github_issue_comments (
    id BIGSERIAL PRIMARY KEY,

    -- Foreign keys
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    issue_id BIGINT NOT NULL REFERENCES github_issues(id) ON DELETE CASCADE,

    -- Comment identification
    github_id BIGINT NOT NULL,                   -- GitHub's comment ID

    -- Comment content
    body TEXT NOT NULL,

    -- Author information
    user_login VARCHAR(255),
    user_id BIGINT,
    author_association VARCHAR(50),              -- OWNER/MAINTAINER/CONTRIBUTOR/NONE

    -- Temporal data
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP,

    -- Raw API response
    raw_data JSONB NOT NULL,

    -- Processing metadata
    fetched_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,                      -- For LLM extraction tracking

    -- Constraints
    CONSTRAINT unique_issue_comment UNIQUE(repo_id, github_id)
);

CREATE INDEX idx_issue_comments_repo_id ON github_issue_comments(repo_id);
CREATE INDEX idx_issue_comments_issue_id ON github_issue_comments(issue_id);
CREATE INDEX idx_issue_comments_github_id ON github_issue_comments(github_id);
CREATE INDEX idx_issue_comments_user_login ON github_issue_comments(user_login);
CREATE INDEX idx_issue_comments_author_association ON github_issue_comments(author_association);
CREATE INDEX idx_issue_comments_created_at ON github_issue_comments(created_at DESC);
CREATE INDEX idx_issue_comments_processed ON github_issue_comments(processed_at) WHERE processed_at IS NULL;

-- Full-text search on comment body
CREATE INDEX idx_issue_comments_text ON github_issue_comments USING GIN (to_tsvector('english', body));

-- ============================================
-- Helpful Views
-- ============================================

-- View: Unprocessed issue comments (for LLM extraction)
CREATE OR REPLACE VIEW v_unprocessed_issue_comments AS
SELECT
    c.id,
    c.repo_id,
    c.issue_id,
    i.number AS issue_number,
    c.body,
    c.user_login,
    c.author_association,
    c.created_at
FROM github_issue_comments c
JOIN github_issues i ON c.issue_id = i.id
WHERE c.processed_at IS NULL
ORDER BY c.created_at ASC;

-- View: PR file change summary (for semantic analysis)
CREATE OR REPLACE VIEW v_pr_file_summary AS
SELECT
    pf.pr_id,
    pr.number AS pr_number,
    pr.title AS pr_title,
    COUNT(*) AS files_changed,
    SUM(pf.additions) AS total_additions,
    SUM(pf.deletions) AS total_deletions,
    SUM(pf.changes) AS total_changes,
    array_agg(pf.filename ORDER BY pf.changes DESC) AS top_files
FROM github_pr_files pf
JOIN github_pull_requests pr ON pf.pr_id = pr.id
GROUP BY pf.pr_id, pr.number, pr.title;

-- ============================================
-- Enable pg_trgm extension for filename matching
-- ============================================
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ============================================
-- Sample Queries
-- ============================================

-- Get all files changed in a PR (for temporal-semantic matching)
-- SELECT filename, status, additions, deletions
-- FROM github_pr_files
-- WHERE pr_id = 123
-- ORDER BY additions + deletions DESC
-- LIMIT 50;

-- Find comments that mention PR numbers (for comment-based linking)
-- SELECT ic.issue_id, i.number, ic.body, ic.user_login, ic.author_association
-- FROM github_issue_comments ic
-- JOIN github_issues i ON ic.issue_id = i.id
-- WHERE ic.body ~* '#[0-9]+'
-- AND ic.author_association IN ('OWNER', 'MAINTAINER', 'MEMBER');

-- Get PRs that modified a specific file path pattern
-- SELECT pr.number, pr.title, pr.merged_at, pf.filename, pf.changes
-- FROM github_pr_files pf
-- JOIN github_pull_requests pr ON pf.pr_id = pr.id
-- WHERE pf.filename LIKE 'src/lib/agents/%'
-- ORDER BY pr.merged_at DESC;

-- Find maintainer comments on open issues
-- SELECT i.number, i.title, ic.body, ic.user_login, ic.created_at
-- FROM github_issue_comments ic
-- JOIN github_issues i ON ic.issue_id = i.id
-- WHERE i.state = 'open'
-- AND ic.author_association IN ('OWNER', 'MAINTAINER', 'MEMBER')
-- ORDER BY ic.created_at DESC;
