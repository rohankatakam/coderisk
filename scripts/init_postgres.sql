-- ========================================
-- PostgreSQL Schema Initialization
-- Reference: risk_assessment_methodology.md §5.2
-- ========================================

-- Metric validation tracking (user feedback)
CREATE TABLE IF NOT EXISTS metric_validations (
    id SERIAL PRIMARY KEY,
    metric_name VARCHAR(50) NOT NULL,  -- "coupling", "co_change", etc.
    file_path VARCHAR(500) NOT NULL,
    metric_value JSONB NOT NULL,       -- Full metric output
    user_feedback VARCHAR(20),         -- "true_positive", "false_positive", null
    feedback_reason TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Aggregate statistics per metric (auto-calculated)
CREATE TABLE IF NOT EXISTS metric_stats (
    metric_name VARCHAR(50) PRIMARY KEY,
    total_uses INT DEFAULT 0,
    false_positives INT DEFAULT 0,
    true_positives INT DEFAULT 0,
    fp_rate FLOAT GENERATED ALWAYS AS (
        CASE WHEN total_uses > 0
        THEN false_positives::FLOAT / total_uses
        ELSE 0.0 END
    ) STORED,
    is_enabled BOOLEAN DEFAULT TRUE,
    last_updated TIMESTAMP DEFAULT NOW()
);

-- Repository metadata (for multi-repo support)
CREATE TABLE IF NOT EXISTS repositories (
    id SERIAL PRIMARY KEY,
    repo_path VARCHAR(500) UNIQUE NOT NULL,
    last_sync TIMESTAMP,
    graph_node_count INT DEFAULT 0,
    graph_edge_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Investigation traces (for debugging and analytics)
CREATE TABLE IF NOT EXISTS investigation_traces (
    id SERIAL PRIMARY KEY,
    file_path VARCHAR(500) NOT NULL,
    phase INT NOT NULL,  -- 1 or 2
    hop_count INT,
    metrics_calculated JSONB,
    llm_decisions JSONB,
    final_risk_level VARCHAR(20),
    duration_ms INT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_metric_validations_name ON metric_validations(metric_name);
CREATE INDEX IF NOT EXISTS idx_metric_validations_file ON metric_validations(file_path);
CREATE INDEX IF NOT EXISTS idx_investigation_traces_file ON investigation_traces(file_path);
CREATE INDEX IF NOT EXISTS idx_investigation_traces_created ON investigation_traces(created_at DESC);

-- Initialize default metric stats (from risk_assessment_methodology.md §2)
INSERT INTO metric_stats (metric_name, is_enabled) VALUES
    ('coupling', TRUE),
    ('co_change', TRUE),
    ('test_ratio', TRUE),
    ('ownership_churn', TRUE),
    ('incident_similarity', TRUE)
ON CONFLICT (metric_name) DO NOTHING;

-- ========================================
-- Layer 3: Incident Database (Session B)
-- Reference: ADR-003, graph_ontology.md Layer 3
-- ========================================

-- Incidents table with full-text search support
CREATE TABLE IF NOT EXISTS incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('critical', 'high', 'medium', 'low')),
    occurred_at TIMESTAMP NOT NULL,
    resolved_at TIMESTAMP,
    root_cause TEXT,
    impact TEXT,

    -- Full-text search columns (BM25-style ranking)
    search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(root_cause, '')), 'C')
    ) STORED,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- GIN index for fast full-text search (BM25 ranking)
CREATE INDEX IF NOT EXISTS idx_incidents_search ON incidents USING GIN(search_vector);

-- Index for time-based queries (recent incidents)
CREATE INDEX IF NOT EXISTS idx_incidents_occurred_at ON incidents(occurred_at DESC);

-- Index for severity-based queries
CREATE INDEX IF NOT EXISTS idx_incidents_severity ON incidents(severity);

-- Incident-to-file links (many-to-many)
CREATE TABLE IF NOT EXISTS incident_files (
    incident_id UUID REFERENCES incidents(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    line_number INT DEFAULT 0,
    blamed_function TEXT DEFAULT '',
    confidence FLOAT DEFAULT 1.0 CHECK (confidence >= 0.0 AND confidence <= 1.0),

    PRIMARY KEY (incident_id, file_path)
);

-- Index for file-based queries (get incidents for a file)
CREATE INDEX IF NOT EXISTS idx_incident_files_path ON incident_files(file_path);

-- Index for incident-based queries (get files for an incident)
CREATE INDEX IF NOT EXISTS idx_incident_files_incident ON incident_files(incident_id);
