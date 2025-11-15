-- ============================================================================
-- CodeRisk PostgreSQL Schema - Initial Creation
-- ============================================================================
-- Purpose: Complete schema for multi-repo risk analysis system
-- Version: 2.0 (Aligned with crisk init pipelines)
-- Date: 2025-11-14
--
-- Architecture:
--   - PostgreSQL: "Kitchen" - Source of truth for raw data + LLM outputs
--   - Neo4j: "Restaurant" - Fast query index for crisk check
--
-- Design Principles:
--   1. Multi-repo safe (all tables have repo_id foreign keys)
--   2. No data collision between repositories
--   3. Reproducible (can rebuild Neo4j from Postgres alone)
--   4. Minimal redundancy (only essential denormalization)
--
-- ============================================================================

-- ============================================================================
-- PART 1: CORE REPOSITORY DATA (Root Table)
-- ============================================================================

CREATE TABLE IF NOT EXISTS github_repositories (
    id BIGSERIAL PRIMARY KEY,
    github_id BIGINT UNIQUE NOT NULL,
    owner VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    full_name VARCHAR(512) UNIQUE NOT NULL,  -- "owner/name" format
    absolute_path TEXT,
    description TEXT,
    default_branch VARCHAR(255) DEFAULT 'main',
    raw_data JSONB,                           -- Full GitHub API response
    fetched_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_repos_owner_name ON github_repositories(owner, name);
CREATE INDEX idx_repos_github_id ON github_repositories(github_id);

COMMENT ON TABLE github_repositories IS 'Master list of all tracked repositories';

-- ============================================================================
-- PART 2: PIPELINE 1 - LINK RESOLUTION (100% Confidence Fact Layer)
-- ============================================================================

-- Commits: Core temporal data with full patch/diff information
CREATE TABLE IF NOT EXISTS github_commits (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    sha VARCHAR(40) NOT NULL,
    author_name VARCHAR(255),
    author_email VARCHAR(255) NOT NULL,
    author_date TIMESTAMP NOT NULL,
    committer_name VARCHAR(255),
    committer_email VARCHAR(255),
    committer_date TIMESTAMP,
    message TEXT,
    verified BOOLEAN DEFAULT false,
    additions INTEGER DEFAULT 0,
    deletions INTEGER DEFAULT 0,
    total_changes INTEGER DEFAULT 0,
    files_changed INTEGER DEFAULT 0,
    raw_data JSONB NOT NULL,                  -- Contains 'files' array with patch data
    fetched_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,                   -- When indexed into Neo4j

    CONSTRAINT unique_repo_commit UNIQUE(repo_id, sha)
);

CREATE INDEX idx_commits_repo ON github_commits(repo_id);
CREATE INDEX idx_commits_sha ON github_commits(sha);
CREATE INDEX idx_commits_author_email ON github_commits(author_email);
CREATE INDEX idx_commits_author_date ON github_commits(author_date DESC);
CREATE INDEX idx_commits_processed ON github_commits(processed_at) WHERE processed_at IS NULL;

COMMENT ON TABLE github_commits IS 'All commits with full patch/diff data for atomization';
COMMENT ON COLUMN github_commits.raw_data IS 'CRITICAL: Contains files[].patch for Pipeline 2 atomization';

-- Issues: Incidents, bugs, feature requests
CREATE TABLE IF NOT EXISTS github_issues (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    github_id BIGINT NOT NULL,
    number INTEGER NOT NULL,
    title TEXT NOT NULL,
    body TEXT,
    state VARCHAR(20) NOT NULL,
    user_login VARCHAR(255),
    user_id BIGINT,
    author_association VARCHAR(50),
    labels JSONB DEFAULT '[]'::jsonb,
    assignees JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP,
    closed_at TIMESTAMP,
    comments_count INTEGER DEFAULT 0,
    reactions_count INTEGER DEFAULT 0,
    raw_data JSONB,
    fetched_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,

    CONSTRAINT unique_repo_issue UNIQUE(repo_id, number)
);

CREATE INDEX idx_issues_repo ON github_issues(repo_id);
CREATE INDEX idx_issues_number ON github_issues(number);
CREATE INDEX idx_issues_state ON github_issues(state);
CREATE INDEX idx_issues_closed_at ON github_issues(closed_at) WHERE closed_at IS NOT NULL;
CREATE INDEX idx_issues_labels ON github_issues USING gin(labels);

COMMENT ON TABLE github_issues IS 'All GitHub issues (incidents, bugs, features)';

-- Pull Requests: Code change proposals with merge data
CREATE TABLE IF NOT EXISTS github_pull_requests (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    github_id BIGINT NOT NULL,
    number INTEGER NOT NULL,
    title TEXT NOT NULL,
    body TEXT,
    state VARCHAR(20) NOT NULL,
    user_login VARCHAR(255),
    user_id BIGINT,
    author_association VARCHAR(50),
    head_ref VARCHAR(255),
    head_sha VARCHAR(40),
    base_ref VARCHAR(255),
    base_sha VARCHAR(40),
    merged BOOLEAN DEFAULT false,
    merged_at TIMESTAMP,
    merge_commit_sha VARCHAR(40),                -- CRITICAL: Links PR to Commit (100% confidence)
    draft BOOLEAN DEFAULT false,
    labels JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP,
    closed_at TIMESTAMP,
    raw_data JSONB,
    fetched_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,

    CONSTRAINT unique_repo_pr UNIQUE(repo_id, number)
);

CREATE INDEX idx_prs_repo ON github_pull_requests(repo_id);
CREATE INDEX idx_prs_number ON github_pull_requests(number);
CREATE INDEX idx_prs_merged ON github_pull_requests(merged);
CREATE INDEX idx_prs_merge_commit ON github_pull_requests(merge_commit_sha) WHERE merge_commit_sha IS NOT NULL;

COMMENT ON TABLE github_pull_requests IS 'All pull requests with merge commit linkage';
COMMENT ON COLUMN github_pull_requests.merge_commit_sha IS '100% confidence link: PR→Commit';

-- Timeline Events: Issue/PR lifecycle events (comments, cross-refs, closures)
CREATE TABLE IF NOT EXISTS github_issue_timeline (
    id BIGSERIAL PRIMARY KEY,
    issue_id BIGINT NOT NULL REFERENCES github_issues(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    source_type VARCHAR(20),                      -- "pull_request", "commit", "issue"
    source_number INTEGER,                        -- PR/Issue number that referenced this
    source_sha VARCHAR(40),                       -- Commit SHA that closed this
    source_title TEXT,
    source_body TEXT,
    source_state VARCHAR(20),
    source_merged_at TIMESTAMP,
    actor_login VARCHAR(255),
    actor_id BIGINT,
    raw_data JSONB,
    fetched_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP
);

CREATE INDEX idx_timeline_issue ON github_issue_timeline(issue_id);
CREATE INDEX idx_timeline_event_type ON github_issue_timeline(event_type);
CREATE INDEX idx_timeline_source_sha ON github_issue_timeline(source_sha) WHERE source_sha IS NOT NULL;

COMMENT ON TABLE github_issue_timeline IS 'Timeline events for REFERENCES and CLOSED_BY edges';

-- Issue Comments: All discussion threads
CREATE TABLE IF NOT EXISTS github_issue_comments (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    issue_id BIGINT NOT NULL REFERENCES github_issues(id) ON DELETE CASCADE,
    github_id BIGINT NOT NULL,
    body TEXT NOT NULL,
    user_login VARCHAR(255),
    user_id BIGINT,
    author_association VARCHAR(50),
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP,
    raw_data JSONB,
    fetched_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,

    CONSTRAINT unique_repo_comment UNIQUE(repo_id, github_id)
);

CREATE INDEX idx_comments_issue ON github_issue_comments(issue_id);
CREATE INDEX idx_comments_repo ON github_issue_comments(repo_id);

COMMENT ON TABLE github_issue_comments IS 'All issue/PR comments for LLM link extraction';

-- LLM-Extracted References: Output of Pipeline 1 LLM extraction
CREATE TABLE IF NOT EXISTS github_issue_commit_refs (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    issue_number INTEGER NOT NULL,
    commit_sha VARCHAR(40),
    pr_number INTEGER,
    action VARCHAR(20),                           -- "fixes", "mentions", "closes", "resolves"
    confidence DOUBLE PRECISION,                  -- 0.0-1.0
    detection_method VARCHAR(50),                 -- "llm", "regex", "timeline"
    extracted_from VARCHAR(50),                   -- "issue_body", "comment", "pr_body"
    extracted_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    evidence TEXT[]
);

CREATE INDEX idx_refs_repo ON github_issue_commit_refs(repo_id);
CREATE INDEX idx_refs_issue ON github_issue_commit_refs(issue_number);
CREATE INDEX idx_refs_commit ON github_issue_commit_refs(commit_sha);

COMMENT ON TABLE github_issue_commit_refs IS 'LLM-extracted issue→commit/PR references';

-- Validated Issue-PR Links: Multi-signal ground truth from Linker Service
CREATE TABLE IF NOT EXISTS github_issue_pr_links (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    issue_number INTEGER NOT NULL,
    pr_number INTEGER NOT NULL,
    detection_method VARCHAR(50) NOT NULL,
    final_confidence NUMERIC(4,2) NOT NULL,      -- 0.70-1.00
    link_quality VARCHAR(20) NOT NULL,           -- "high", "medium", "low"
    confidence_breakdown JSONB,
    evidence_sources TEXT[],
    comprehensive_rationale TEXT,
    semantic_analysis JSONB,
    temporal_analysis JSONB,
    flags JSONB,
    metadata JSONB,
    user_verified BOOLEAN DEFAULT false,         -- Set true when user approves in Linker Service
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT unique_repo_issue_pr_link UNIQUE(repo_id, issue_number, pr_number)
);

CREATE INDEX idx_pr_links_repo ON github_issue_pr_links(repo_id);
CREATE INDEX idx_pr_links_issue ON github_issue_pr_links(issue_number);
CREATE INDEX idx_pr_links_confidence ON github_issue_pr_links(final_confidence DESC);
CREATE INDEX idx_pr_links_user_verified ON github_issue_pr_links(user_verified);

COMMENT ON TABLE github_issue_pr_links IS 'Validated Issue↔PR links for FIXED_BY edges';
COMMENT ON COLUMN github_issue_pr_links.user_verified IS 'True when user approved in Linker Service';

-- Orphaned Issues: Issues that couldn't be linked
CREATE TABLE IF NOT EXISTS github_issue_no_links (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    issue_number INTEGER NOT NULL,
    no_links_reason VARCHAR(50),
    classification VARCHAR(50),
    classification_confidence NUMERIC(4,2),
    classification_rationale TEXT,
    conversation_summary TEXT,
    candidates_evaluated INTEGER DEFAULT 0,
    best_candidate_score NUMERIC(4,2),
    safety_brake_reason TEXT,
    issue_closed_at TIMESTAMP,
    analyzed_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT unique_repo_no_link_issue UNIQUE(repo_id, issue_number)
);

CREATE INDEX idx_no_links_repo ON github_issue_no_links(repo_id);

COMMENT ON TABLE github_issue_no_links IS 'Issues that could not be linked to commits/PRs';

-- PR File Changes: Individual files modified in PRs
CREATE TABLE IF NOT EXISTS github_pr_files (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    pr_id BIGINT NOT NULL REFERENCES github_pull_requests(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    status VARCHAR(20) NOT NULL,                 -- "modified", "added", "deleted", "renamed"
    additions INTEGER DEFAULT 0,
    deletions INTEGER DEFAULT 0,
    changes INTEGER DEFAULT 0,
    previous_filename TEXT,
    patch TEXT,                                   -- File-level diff
    blob_url TEXT,
    raw_url TEXT,
    raw_data JSONB,
    fetched_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_pr_files_pr ON github_pr_files(pr_id);
CREATE INDEX idx_pr_files_filename ON github_pr_files(filename);

COMMENT ON TABLE github_pr_files IS 'Individual file changes within PRs';

-- ============================================================================
-- PART 3: PIPELINE 2 - CODE-BLOCK ATOMIZATION (Atomic Unit Layer)
-- ============================================================================

-- Code Blocks: Individual functions, methods, classes
CREATE TABLE IF NOT EXISTS code_blocks (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    block_name TEXT NOT NULL,                    -- e.g., "updateTableEditor", "TableEditor::render"
    block_type VARCHAR(50) NOT NULL,             -- "function", "method", "class", "component"
    first_seen_sha VARCHAR(40) NOT NULL,
    current_status VARCHAR(20) DEFAULT 'active', -- "active", "deleted", "renamed"
    evolved_from_id BIGINT REFERENCES code_blocks(id) ON DELETE SET NULL,
    language VARCHAR(50),                        -- "typescript", "python", "go", "javascript"
    start_line INTEGER,
    end_line INTEGER,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT unique_repo_file_block UNIQUE(repo_id, file_path, block_name)
);

CREATE INDEX idx_blocks_repo ON code_blocks(repo_id);
CREATE INDEX idx_blocks_file ON code_blocks(file_path);
CREATE INDEX idx_blocks_name ON code_blocks(block_name);
CREATE INDEX idx_blocks_status ON code_blocks(current_status);
CREATE INDEX idx_blocks_first_seen ON code_blocks(first_seen_sha);

COMMENT ON TABLE code_blocks IS 'Atomic code units (functions, methods) extracted from commits';
COMMENT ON COLUMN code_blocks.evolved_from_id IS 'Tracks function renames to preserve risk history';

-- Code Block Modifications: Transaction log of block changes
CREATE TABLE IF NOT EXISTS code_block_modifications (
    id BIGSERIAL PRIMARY KEY,
    code_block_id BIGINT NOT NULL REFERENCES code_blocks(id) ON DELETE CASCADE,
    commit_sha VARCHAR(40) NOT NULL,
    developer_email VARCHAR(255) NOT NULL,
    change_type VARCHAR(20) NOT NULL,            -- "added", "modified", "deleted"
    additions INTEGER DEFAULT 0,
    deletions INTEGER DEFAULT 0,
    modified_at TIMESTAMP NOT NULL,
    raw_llm_output JSONB,                        -- Full LLM analysis of the change
    is_refactor_only BOOLEAN DEFAULT false,      -- True if LLM classified as style-only
    created_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT unique_block_commit UNIQUE(code_block_id, commit_sha)
);

CREATE INDEX idx_block_mods_block ON code_block_modifications(code_block_id);
CREATE INDEX idx_block_mods_commit ON code_block_modifications(commit_sha);
CREATE INDEX idx_block_mods_developer ON code_block_modifications(developer_email);
CREATE INDEX idx_block_mods_date ON code_block_modifications(modified_at DESC);

COMMENT ON TABLE code_block_modifications IS 'History of all changes to each code block (Pipeline 2 output)';
COMMENT ON COLUMN code_block_modifications.raw_llm_output IS 'Direct output from Pipeline 2 LLM Atomizer';

-- ============================================================================
-- PART 4: PIPELINE 3 - CONTEXTUAL RISK INDEXING (Intelligence Layer)
-- ============================================================================

-- Code Block Risk Index: Pre-computed risk properties for each block
CREATE TABLE IF NOT EXISTS code_block_risk_index (
    code_block_id BIGINT PRIMARY KEY REFERENCES code_blocks(id) ON DELETE CASCADE,

    -- R_temporal (Incident Risk)
    incident_count INTEGER DEFAULT 0,
    temporal_summary TEXT,                       -- LLM-generated summary of incident history
    last_incident_at TIMESTAMP,

    -- R_ownership (Knowledge Risk)
    semantic_importance VARCHAR(20),             -- LLM-classified: "P0", "P1", "P2"
    original_author_email VARCHAR(255),
    last_modifier_email VARCHAR(255),
    staleness_days INTEGER,
    familiarity_map JSONB DEFAULT '{}'::jsonb,   -- {dev_email: edit_count}

    -- Metadata
    last_indexed_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_risk_incidents ON code_block_risk_index(incident_count DESC) WHERE incident_count > 0;
CREATE INDEX idx_risk_staleness ON code_block_risk_index(staleness_days DESC);
CREATE INDEX idx_risk_importance ON code_block_risk_index(semantic_importance);

COMMENT ON TABLE code_block_risk_index IS 'Pre-computed R_temporal and R_ownership risk signals (Pipeline 3 output)';
COMMENT ON COLUMN code_block_risk_index.temporal_summary IS 'LLM summary of incident patterns for this block';
COMMENT ON COLUMN code_block_risk_index.semantic_importance IS 'LLM-classified business criticality';

-- Code Block Coupling: Co-change risk between blocks
CREATE TABLE IF NOT EXISTS code_block_coupling (
    id BIGSERIAL PRIMARY KEY,
    block_a_id BIGINT NOT NULL REFERENCES code_blocks(id) ON DELETE CASCADE,
    block_b_id BIGINT NOT NULL REFERENCES code_blocks(id) ON DELETE CASCADE,
    co_change_rate NUMERIC(4,2) NOT NULL,        -- 0.00-1.00
    co_change_count INTEGER NOT NULL,
    reason TEXT,                                  -- LLM-generated explanation
    last_co_changed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT unique_block_pair UNIQUE(block_a_id, block_b_id),
    CONSTRAINT ordered_pair CHECK(block_a_id < block_b_id)  -- Ensure (A,B) not (B,A)
);

CREATE INDEX idx_coupling_a ON code_block_coupling(block_a_id);
CREATE INDEX idx_coupling_b ON code_block_coupling(block_b_id);
CREATE INDEX idx_coupling_rate ON code_block_coupling(co_change_rate DESC) WHERE co_change_rate >= 0.50;

COMMENT ON TABLE code_block_coupling IS 'Pre-computed R_coupling (co-change) risk (Pipeline 3 output)';
COMMENT ON COLUMN code_block_coupling.reason IS 'LLM explanation of why these blocks change together';

-- ============================================================================
-- PART 5: SUPPORTING TABLES (Metadata & Analytics)
-- ============================================================================

-- Branches
CREATE TABLE IF NOT EXISTS github_branches (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    commit_sha VARCHAR(40),
    protected BOOLEAN DEFAULT false,
    raw_data JSONB,
    fetched_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT unique_repo_branch UNIQUE(repo_id, name)
);

CREATE INDEX idx_branches_repo ON github_branches(repo_id);

-- Contributors
CREATE TABLE IF NOT EXISTS github_contributors (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    github_id BIGINT NOT NULL,
    login VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    contributions INTEGER DEFAULT 0,
    raw_data JSONB,
    fetched_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT unique_repo_contributor UNIQUE(repo_id, login)
);

CREATE INDEX idx_contributors_repo ON github_contributors(repo_id);

-- Developers (Aggregated Activity)
CREATE TABLE IF NOT EXISTS github_developers (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    commit_count INTEGER DEFAULT 0,
    first_commit_at TIMESTAMP,
    last_commit_at TIMESTAMP,
    active_days INTEGER DEFAULT 0,
    raw_data JSONB,
    fetched_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT unique_repo_developer UNIQUE(repo_id, email)
);

CREATE INDEX idx_developers_repo ON github_developers(repo_id);
CREATE INDEX idx_developers_email ON github_developers(email);

-- Languages
CREATE TABLE IF NOT EXISTS github_languages (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    language VARCHAR(100) NOT NULL,
    bytes BIGINT DEFAULT 0,
    percentage NUMERIC(5,2),
    fetched_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT unique_repo_language UNIQUE(repo_id, language)
);

CREATE INDEX idx_languages_repo ON github_languages(repo_id);

-- DORA Metrics
CREATE TABLE IF NOT EXISTS github_dora_metrics (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    metric_type VARCHAR(50) NOT NULL,
    value NUMERIC(10,2),
    period_start TIMESTAMP,
    period_end TIMESTAMP,
    raw_data JSONB,
    calculated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT unique_repo_dora_metric UNIQUE(repo_id, metric_type, period_start)
);

CREATE INDEX idx_dora_repo ON github_dora_metrics(repo_id);

-- Git Trees (Full repo snapshots)
CREATE TABLE IF NOT EXISTS github_trees (
    id BIGSERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,
    sha VARCHAR(40) NOT NULL,
    path TEXT,
    mode VARCHAR(10),
    type VARCHAR(20),
    size BIGINT,
    raw_data JSONB,
    fetched_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT unique_repo_tree UNIQUE(repo_id, sha, path)
);

CREATE INDEX idx_trees_repo ON github_trees(repo_id);

-- ============================================================================
-- PART 6: TRIGGERS & HELPER FUNCTIONS
-- ============================================================================

-- Auto-update updated_at timestamps
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_repos_timestamp
    BEFORE UPDATE ON github_repositories
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_blocks_timestamp
    BEFORE UPDATE ON code_blocks
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_risk_index_timestamp
    BEFORE UPDATE ON code_block_risk_index
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_coupling_timestamp
    BEFORE UPDATE ON code_block_coupling
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER update_pr_links_timestamp
    BEFORE UPDATE ON github_issue_pr_links
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- ============================================================================
-- PART 7: VALIDATION
-- ============================================================================

DO $$
DECLARE
    required_tables TEXT[] := ARRAY[
        'github_repositories',
        'github_commits',
        'github_issues',
        'github_pull_requests',
        'github_issue_timeline',
        'github_issue_comments',
        'github_issue_commit_refs',
        'github_issue_pr_links',
        'github_issue_no_links',
        'github_pr_files',
        'code_blocks',
        'code_block_modifications',
        'code_block_risk_index',
        'code_block_coupling',
        'github_branches',
        'github_contributors',
        'github_developers',
        'github_languages',
        'github_dora_metrics',
        'github_trees'
    ];
    table_name TEXT;
    missing_tables TEXT[] := '{}';
BEGIN
    FOREACH table_name IN ARRAY required_tables
    LOOP
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.tables
            WHERE table_schema = 'public' AND table_name = table_name
        ) THEN
            missing_tables := array_append(missing_tables, table_name);
        END IF;
    END LOOP;

    IF array_length(missing_tables, 1) > 0 THEN
        RAISE EXCEPTION 'Schema initialization failed. Missing tables: %', array_to_string(missing_tables, ', ');
    ELSE
        RAISE NOTICE '✅ Schema initialization complete!';
        RAISE NOTICE '   Total tables created: %', array_length(required_tables, 1);
        RAISE NOTICE '';
        RAISE NOTICE '📊 Pipeline Table Mapping:';
        RAISE NOTICE '   Pipeline 1 (Link Resolution): 9 tables';
        RAISE NOTICE '   Pipeline 2 (Atomization): 2 tables';
        RAISE NOTICE '   Pipeline 3 (Risk Indexing): 2 tables';
        RAISE NOTICE '   Supporting tables: 7 tables';
        RAISE NOTICE '';
        RAISE NOTICE '✅ Ready for multi-repo ingestion!';
    END IF;
END $$;
