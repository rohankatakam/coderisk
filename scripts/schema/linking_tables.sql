-- ============================================
-- CodeRisk Issue-PR Linking Tables
-- ============================================
-- Purpose: Store validated issue-PR links with confidence scores
-- Reference: test_data/docs/linking/Issue_Flow.md
-- ============================================

-- Issue-PR Links Table
CREATE TABLE IF NOT EXISTS github_issue_pr_links (
    id BIGSERIAL PRIMARY KEY,

    -- Foreign key
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,

    -- Link identification
    issue_number INTEGER NOT NULL,
    pr_number INTEGER NOT NULL,

    -- Detection metadata
    detection_method VARCHAR(50) NOT NULL, -- "github_timeline_verified", "explicit", "explicit_bidirectional", "deep_link_finder"
    final_confidence NUMERIC(4,3) NOT NULL CHECK (final_confidence >= 0 AND final_confidence <= 1),
    link_quality VARCHAR(20) NOT NULL, -- "high", "medium", "low"

    -- Confidence breakdown (JSONB)
    confidence_breakdown JSONB NOT NULL,

    -- Evidence sources (array)
    evidence_sources TEXT[] NOT NULL,

    -- Rationale
    comprehensive_rationale TEXT NOT NULL,

    -- Analysis results (JSONB)
    semantic_analysis JSONB,
    temporal_analysis JSONB,

    -- Flags (JSONB)
    flags JSONB NOT NULL,

    -- Metadata (JSONB)
    metadata JSONB NOT NULL,

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- Constraints
    CONSTRAINT unique_repo_issue_pr UNIQUE(repo_id, issue_number, pr_number)
);

CREATE INDEX idx_issue_pr_links_repo_id ON github_issue_pr_links(repo_id);
CREATE INDEX idx_issue_pr_links_issue_number ON github_issue_pr_links(issue_number);
CREATE INDEX idx_issue_pr_links_pr_number ON github_issue_pr_links(pr_number);
CREATE INDEX idx_issue_pr_links_confidence ON github_issue_pr_links(final_confidence DESC);
CREATE INDEX idx_issue_pr_links_detection_method ON github_issue_pr_links(detection_method);

-- Issue No-Links Table (for true negatives)
CREATE TABLE IF NOT EXISTS github_issue_no_links (
    id BIGSERIAL PRIMARY KEY,

    -- Foreign key
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,

    -- Issue identification
    issue_number INTEGER NOT NULL,

    -- Classification
    no_links_reason VARCHAR(50) NOT NULL,
    classification VARCHAR(50) NOT NULL,
    classification_confidence NUMERIC(4,3) NOT NULL,
    classification_rationale TEXT NOT NULL,
    conversation_summary TEXT,

    -- Deep finder results (if applicable)
    candidates_evaluated INTEGER,
    best_candidate_score NUMERIC(4,3),
    safety_brake_reason TEXT,

    -- Timestamps
    issue_closed_at TIMESTAMP,
    analyzed_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- Constraints
    CONSTRAINT unique_repo_issue_no_link UNIQUE(repo_id, issue_number)
);

CREATE INDEX idx_issue_no_links_repo_id ON github_issue_no_links(repo_id);
CREATE INDEX idx_issue_no_links_issue_number ON github_issue_no_links(issue_number);
CREATE INDEX idx_issue_no_links_classification ON github_issue_no_links(classification);

-- DORA Metrics Table (repository-level)
CREATE TABLE IF NOT EXISTS github_dora_metrics (
    id BIGSERIAL PRIMARY KEY,

    -- Foreign key
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,

    -- Metrics
    median_lead_time_hours NUMERIC(10,2) NOT NULL,
    median_pr_lifespan_hours NUMERIC(10,2) NOT NULL,
    sample_size INTEGER NOT NULL,
    insufficient_history BOOLEAN DEFAULT FALSE,

    -- Timeline data
    timeline_events_fetched INTEGER DEFAULT 0,
    cross_reference_links_found INTEGER DEFAULT 0,

    -- Timestamps
    computed_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),

    -- Constraints
    CONSTRAINT unique_repo_dora UNIQUE(repo_id, computed_at)
);

CREATE INDEX idx_dora_metrics_repo_id ON github_dora_metrics(repo_id);
CREATE INDEX idx_dora_metrics_computed_at ON github_dora_metrics(computed_at DESC);
