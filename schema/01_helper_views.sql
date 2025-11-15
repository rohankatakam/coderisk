-- ============================================================================
-- Helper Views for Graph Construction
-- ============================================================================
-- Purpose: Provide compatibility with existing StagingClient query methods
-- Version: 2.0
-- Date: 2025-11-14
--
-- These views enable the graph builder to query unprocessed data from staging
-- ============================================================================

-- ============================================================================
-- View 1: v_unprocessed_commits
-- ============================================================================
-- Purpose: Retrieve commits that haven't been processed into Neo4j yet
-- Used by: StagingClient.FetchUnprocessedCommits()
-- Used by: graph.Builder.BuildGraph()

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

COMMENT ON VIEW v_unprocessed_commits IS 'Commits awaiting graph construction (processed_at IS NULL)';

-- ============================================================================
-- View 2: v_unprocessed_issues
-- ============================================================================
-- Purpose: Retrieve issues that haven't been processed into Neo4j yet
-- Used by: StagingClient.FetchUnprocessedIssues()
-- Used by: graph.Builder.BuildGraph()

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

COMMENT ON VIEW v_unprocessed_issues IS 'Issues awaiting graph construction (processed_at IS NULL)';

-- ============================================================================
-- View 3: v_unprocessed_prs
-- ============================================================================
-- Purpose: Retrieve PRs that haven't been processed into Neo4j yet
-- Used by: StagingClient.FetchUnprocessedPRs()
-- Used by: graph.Builder.BuildGraph()

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

COMMENT ON VIEW v_unprocessed_prs IS 'Pull requests awaiting graph construction (processed_at IS NULL)';

-- ============================================================================
-- View 4: v_unprocessed_issue_comments (Optional)
-- ============================================================================
-- Purpose: Retrieve issue comments that haven't been processed yet
-- Note: Currently not actively used by graph builder, but included for completeness

CREATE OR REPLACE VIEW v_unprocessed_issue_comments AS
SELECT
    ic.id,
    ic.repo_id,
    ic.issue_id,
    ic.github_id,
    ic.body,
    ic.user_login,
    ic.created_at,
    ic.raw_data
FROM github_issue_comments ic
WHERE ic.processed_at IS NULL
ORDER BY ic.created_at ASC;

COMMENT ON VIEW v_unprocessed_issue_comments IS 'Issue comments awaiting processing (optional, for future use)';

-- ============================================================================
-- View 5: v_pr_file_summary (Optional)
-- ============================================================================
-- Purpose: Aggregate file change statistics for each PR
-- Note: Currently not actively used, but useful for analytics

CREATE OR REPLACE VIEW v_pr_file_summary AS
SELECT
    pf.pr_id,
    pf.repo_id,
    COUNT(*) AS file_count,
    SUM(pf.additions) AS total_additions,
    SUM(pf.deletions) AS total_deletions,
    SUM(pf.changes) AS total_changes,
    ARRAY_AGG(pf.filename ORDER BY pf.changes DESC) AS files_changed
FROM github_pr_files pf
GROUP BY pf.pr_id, pf.repo_id;

COMMENT ON VIEW v_pr_file_summary IS 'Aggregated file change statistics per PR (analytics/debugging)';

-- ============================================================================
-- VALIDATION
-- ============================================================================

DO $$
DECLARE
    required_views TEXT[] := ARRAY[
        'v_unprocessed_commits',
        'v_unprocessed_issues',
        'v_unprocessed_prs',
        'v_unprocessed_issue_comments',
        'v_pr_file_summary'
    ];
    view_name TEXT;
    missing_views TEXT[] := '{}';
BEGIN
    FOREACH view_name IN ARRAY required_views
    LOOP
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.views
            WHERE table_schema = 'public' AND table_name = view_name
        ) THEN
            missing_views := array_append(missing_views, view_name);
        END IF;
    END LOOP;

    IF array_length(missing_views, 1) > 0 THEN
        RAISE EXCEPTION 'View creation failed. Missing views: %', array_to_string(missing_views, ', ');
    ELSE
        RAISE NOTICE '✅ Helper views created successfully!';
        RAISE NOTICE '   Total views: %', array_length(required_views, 1);
        RAISE NOTICE '';
        RAISE NOTICE '📋 Views available:';
        RAISE NOTICE '   - v_unprocessed_commits (for graph.Builder)';
        RAISE NOTICE '   - v_unprocessed_issues (for graph.Builder)';
        RAISE NOTICE '   - v_unprocessed_prs (for graph.Builder)';
        RAISE NOTICE '   - v_unprocessed_issue_comments (optional)';
        RAISE NOTICE '   - v_pr_file_summary (analytics)';
        RAISE NOTICE '';
        RAISE NOTICE '✅ Schema now 100%% compatible with GitHub API downloaders!';
    END IF;
END $$;
