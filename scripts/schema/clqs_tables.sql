-- ============================================
-- CLQS (Code Lineage Quality Score) Tables
-- ============================================

-- Repository-level CLQS scores
CREATE TABLE IF NOT EXISTS clqs_scores (
    id BIGSERIAL PRIMARY KEY,

    -- Foreign key
    repo_id BIGINT NOT NULL REFERENCES github_repositories(id) ON DELETE CASCADE,

    -- Overall CLQS
    clqs NUMERIC(5,2) NOT NULL CHECK (clqs >= 0 AND clqs <= 100),
    clqs_grade VARCHAR(3) NOT NULL,  -- "A+", "A", "A-", "B+", etc.
    clqs_rank VARCHAR(50) NOT NULL,   -- "World-Class", "High Quality", etc.
    confidence_multiplier NUMERIC(3,2) NOT NULL,

    -- Component scores (0-100)
    link_coverage NUMERIC(5,2) NOT NULL,
    confidence_quality NUMERIC(5,2) NOT NULL,
    evidence_diversity NUMERIC(5,2) NOT NULL,
    temporal_precision NUMERIC(5,2) NOT NULL,
    semantic_strength NUMERIC(5,2) NOT NULL,

    -- Component contributions (weighted)
    link_coverage_contribution NUMERIC(5,2) NOT NULL,
    confidence_quality_contribution NUMERIC(5,2) NOT NULL,
    evidence_diversity_contribution NUMERIC(5,2) NOT NULL,
    temporal_precision_contribution NUMERIC(5,2) NOT NULL,
    semantic_strength_contribution NUMERIC(5,2) NOT NULL,

    -- Statistics
    total_closed_issues INTEGER NOT NULL,
    eligible_issues INTEGER NOT NULL,
    linked_issues INTEGER NOT NULL,
    total_links INTEGER NOT NULL,
    avg_confidence NUMERIC(4,3) NOT NULL,

    -- Metadata
    clqs_version VARCHAR(10) DEFAULT 'v2.1',
    computed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- Constraints
    CONSTRAINT unique_repo_clqs_latest UNIQUE(repo_id, computed_at)
);

CREATE INDEX IF NOT EXISTS idx_clqs_scores_repo_id ON clqs_scores(repo_id);
CREATE INDEX IF NOT EXISTS idx_clqs_scores_clqs ON clqs_scores(clqs DESC);
CREATE INDEX IF NOT EXISTS idx_clqs_scores_computed_at ON clqs_scores(computed_at DESC);

-- Component details (for deep-dive analysis)
CREATE TABLE IF NOT EXISTS clqs_component_details (
    id BIGSERIAL PRIMARY KEY,

    -- Foreign key
    clqs_score_id BIGINT NOT NULL REFERENCES clqs_scores(id) ON DELETE CASCADE,

    -- Component breakdown (JSONB)
    link_coverage_details JSONB NOT NULL,
    confidence_quality_details JSONB NOT NULL,
    evidence_diversity_details JSONB NOT NULL,
    temporal_precision_details JSONB NOT NULL,
    semantic_strength_details JSONB NOT NULL,

    -- Recommendations
    recommendations JSONB NOT NULL,

    -- Labeling opportunities
    labeling_opportunities JSONB NOT NULL,

    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_clqs_component_details_score_id ON clqs_component_details(clqs_score_id);

-- Comments
COMMENT ON TABLE clqs_scores IS 'CLQS (Code Lineage Quality Score) - composite quality scores for issue-PR linking';
COMMENT ON COLUMN clqs_scores.clqs IS 'Overall CLQS score (0-100)';
COMMENT ON COLUMN clqs_scores.confidence_multiplier IS 'Confidence multiplier for risk scoring (0.25-1.00)';
COMMENT ON COLUMN clqs_scores.link_coverage IS 'Component 1: Link Coverage score (0-100)';
COMMENT ON COLUMN clqs_scores.confidence_quality IS 'Component 2: Confidence Quality score (0-100)';
COMMENT ON COLUMN clqs_scores.evidence_diversity IS 'Component 3: Evidence Diversity score (0-100)';
COMMENT ON COLUMN clqs_scores.temporal_precision IS 'Component 4: Temporal Precision score (0-100)';
COMMENT ON COLUMN clqs_scores.semantic_strength IS 'Component 5: Semantic Strength score (0-100)';
