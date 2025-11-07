package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Repository info structure
type RepositoryInfo struct {
	ID       int64
	FullName string
}

// CLQS Statistics structure
type CLQSStatistics struct {
	TotalClosed   int
	Eligible      int
	Linked        int
	TotalLinks    int
	AvgConfidence float64
}

// Link Coverage Stats
type LinkCoverageStats struct {
	TotalClosed    int
	EligibleIssues int
	LinkedIssues   int
	UnlinkedIssues int
}

// Confidence Stats
type ConfidenceStats struct {
	HighCount   int
	HighAvg     float64
	MediumCount int
	MediumAvg   float64
	LowCount    int
	LowAvg      float64
	TotalLinks  int
}

// Evidence Diversity Stats
type EvidenceDiversityStats struct {
	AvgEvidenceTypes    float64
	SixTypes            int
	FiveTypes           int
	FourTypes           int
	ThreeTypes          int
	TwoTypes            int
	OneType             int
	ExplicitCount       int
	TimelineCount       int
	BidirectionalCount  int
	SemanticCount       int
	TemporalCount       int
	FileContextCount    int
	TotalLinks          int
}

// Temporal Precision Stats
type TemporalPrecisionStats struct {
	TotalLinks         int
	TightTemporalLinks int
	Under5Min          int
	FiveMinTo1Hr       int
	OneHrTo24Hr        int
	Over24Hr           int
}

// Semantic Strength Stats
type SemanticStrengthStats struct {
	AvgMaxSemantic float64
	HighSemantic   int
	MediumSemantic int
	LowSemantic    int
	AvgTitle       float64
	AvgBody        float64
	AvgComment     float64
	TotalLinks     int
}

// CLQSScore represents a CLQS score to store
type CLQSScore struct {
	RepoID               int64
	CLQS                 float64
	Grade                string
	Rank                 string
	ConfidenceMultiplier float64

	LinkCoverage      float64
	ConfidenceQuality float64
	EvidenceDiversity float64
	TemporalPrecision float64
	SemanticStrength  float64

	LinkCoverageContribution      float64
	ConfidenceQualityContribution float64
	EvidenceDiversityContribution float64
	TemporalPrecisionContribution float64
	SemanticStrengthContribution  float64

	TotalClosedIssues int
	EligibleIssues    int
	LinkedIssues      int
	TotalLinks        int
	AvgConfidence     float64

	ComputedAt time.Time
}

// RepositoryExists checks if a repository exists
func (c *StagingClient) RepositoryExists(ctx context.Context, repoID int64) (bool, error) {
	var exists bool
	err := c.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM github_repositories WHERE id = $1)
	`, repoID).Scan(&exists)
	return exists, err
}

// CountClosedIssues counts the number of closed issues for a repository
func (c *StagingClient) CountClosedIssues(ctx context.Context, repoID int64) (int, error) {
	var count int
	err := c.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM github_issues
		WHERE repo_id = $1 AND state = 'closed' AND closed_at IS NOT NULL
	`, repoID).Scan(&count)
	return count, err
}

// GetRepositoryInfo retrieves repository information
func (c *StagingClient) GetRepositoryInfo(ctx context.Context, repoID int64) (*RepositoryInfo, error) {
	var info RepositoryInfo
	err := c.db.QueryRowContext(ctx, `
		SELECT id, full_name
		FROM github_repositories
		WHERE id = $1
	`, repoID).Scan(&info.ID, &info.FullName)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// QueryLinkCoverageStats queries statistics for Link Coverage component
func (c *StagingClient) QueryLinkCoverageStats(ctx context.Context, repoID int64) (*LinkCoverageStats, error) {
	query := `
		WITH eligible_issues AS (
			SELECT i.number
			FROM github_issues i
			WHERE i.repo_id = $1
			  AND i.state = 'closed'
			  AND i.closed_at IS NOT NULL
			  AND NOT EXISTS (
				  SELECT 1 FROM github_issue_no_links n
				  WHERE n.repo_id = i.repo_id
					AND n.issue_number = i.number
					AND n.classification IN ('not_a_bug', 'duplicate', 'wontfix', 'user_action_required')
			  )
		),
		linked_issues AS (
			SELECT DISTINCT issue_number
			FROM github_issue_pr_links
			WHERE repo_id = $1
			  AND final_confidence >= 0.50
		)
		SELECT
			(SELECT COUNT(*) FROM github_issues WHERE repo_id = $1 AND state = 'closed') as total_closed,
			(SELECT COUNT(*) FROM eligible_issues) as eligible,
			(SELECT COUNT(*) FROM linked_issues) as linked,
			(SELECT COUNT(*) FROM eligible_issues) - (SELECT COUNT(*) FROM linked_issues) as unlinked
	`

	var stats LinkCoverageStats
	err := c.db.QueryRowContext(ctx, query, repoID).Scan(
		&stats.TotalClosed,
		&stats.EligibleIssues,
		&stats.LinkedIssues,
		&stats.UnlinkedIssues,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query link coverage stats: %w", err)
	}

	return &stats, nil
}

// QueryConfidenceStats queries statistics for Confidence Quality component
func (c *StagingClient) QueryConfidenceStats(ctx context.Context, repoID int64) (*ConfidenceStats, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE final_confidence >= 0.85) as high_count,
			COALESCE(AVG(final_confidence) FILTER (WHERE final_confidence >= 0.85), 0) as high_avg,
			COUNT(*) FILTER (WHERE final_confidence >= 0.70 AND final_confidence < 0.85) as medium_count,
			COALESCE(AVG(final_confidence) FILTER (WHERE final_confidence >= 0.70 AND final_confidence < 0.85), 0) as medium_avg,
			COUNT(*) FILTER (WHERE final_confidence < 0.70) as low_count,
			COALESCE(AVG(final_confidence) FILTER (WHERE final_confidence < 0.70), 0) as low_avg,
			COUNT(*) as total_links
		FROM github_issue_pr_links
		WHERE repo_id = $1
	`

	var stats ConfidenceStats
	err := c.db.QueryRowContext(ctx, query, repoID).Scan(
		&stats.HighCount,
		&stats.HighAvg,
		&stats.MediumCount,
		&stats.MediumAvg,
		&stats.LowCount,
		&stats.LowAvg,
		&stats.TotalLinks,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query confidence stats: %w", err)
	}

	return &stats, nil
}

// QueryEvidenceDiversityStats queries statistics for Evidence Diversity component
func (c *StagingClient) QueryEvidenceDiversityStats(ctx context.Context, repoID int64) (*EvidenceDiversityStats, error) {
	query := `
		WITH evidence_counts AS (
			SELECT
				id,
				issue_number,
				pr_number,
				evidence_sources,
				-- Count distinct evidence categories
				(CASE WHEN 'explicit' = ANY(evidence_sources) THEN 1 ELSE 0 END +
				 CASE WHEN 'github_timeline_verified' = ANY(evidence_sources) THEN 1 ELSE 0 END +
				 CASE WHEN 'bidirectional' = ANY(evidence_sources) THEN 1 ELSE 0 END +
				 CASE WHEN ('semantic_title' = ANY(evidence_sources) OR
							'semantic_body' = ANY(evidence_sources) OR
							'semantic_comment' = ANY(evidence_sources)) THEN 1 ELSE 0 END +
				 CASE WHEN 'temporal' = ANY(evidence_sources) THEN 1 ELSE 0 END +
				 CASE WHEN 'file_context' = ANY(evidence_sources) THEN 1 ELSE 0 END
				) as evidence_count
			FROM github_issue_pr_links
			WHERE repo_id = $1
		)
		SELECT
			COALESCE(AVG(evidence_count), 0) as avg_evidence_types,
			COUNT(*) FILTER (WHERE evidence_count = 6) as six_types,
			COUNT(*) FILTER (WHERE evidence_count = 5) as five_types,
			COUNT(*) FILTER (WHERE evidence_count = 4) as four_types,
			COUNT(*) FILTER (WHERE evidence_count = 3) as three_types,
			COUNT(*) FILTER (WHERE evidence_count = 2) as two_types,
			COUNT(*) FILTER (WHERE evidence_count = 1) as one_type,
			-- Count usage of each evidence type
			COUNT(*) FILTER (WHERE 'explicit' = ANY(evidence_sources)) as explicit_count,
			COUNT(*) FILTER (WHERE 'github_timeline_verified' = ANY(evidence_sources)) as timeline_count,
			COUNT(*) FILTER (WHERE 'bidirectional' = ANY(evidence_sources)) as bidirectional_count,
			COUNT(*) FILTER (WHERE 'semantic_title' = ANY(evidence_sources) OR
								   'semantic_body' = ANY(evidence_sources) OR
								   'semantic_comment' = ANY(evidence_sources)) as semantic_count,
			COUNT(*) FILTER (WHERE 'temporal' = ANY(evidence_sources)) as temporal_count,
			COUNT(*) FILTER (WHERE 'file_context' = ANY(evidence_sources)) as file_context_count,
			COUNT(*) as total_links
		FROM evidence_counts
	`

	var stats EvidenceDiversityStats
	err := c.db.QueryRowContext(ctx, query, repoID).Scan(
		&stats.AvgEvidenceTypes,
		&stats.SixTypes,
		&stats.FiveTypes,
		&stats.FourTypes,
		&stats.ThreeTypes,
		&stats.TwoTypes,
		&stats.OneType,
		&stats.ExplicitCount,
		&stats.TimelineCount,
		&stats.BidirectionalCount,
		&stats.SemanticCount,
		&stats.TemporalCount,
		&stats.FileContextCount,
		&stats.TotalLinks,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query evidence diversity stats: %w", err)
	}

	return &stats, nil
}

// QueryTemporalPrecisionStats queries statistics for Temporal Precision component
func (c *StagingClient) QueryTemporalPrecisionStats(ctx context.Context, repoID int64) (*TemporalPrecisionStats, error) {
	query := `
		WITH temporal_stats AS (
			SELECT
				CAST(temporal_analysis->>'temporal_delta_seconds' AS INTEGER) as delta_seconds
			FROM github_issue_pr_links
			WHERE repo_id = $1
			  AND temporal_analysis IS NOT NULL
			  AND temporal_analysis->>'temporal_delta_seconds' IS NOT NULL
		)
		SELECT
			COUNT(*) as total_links,
			COUNT(*) FILTER (WHERE ABS(delta_seconds) < 3600) as tight_temporal,
			COUNT(*) FILTER (WHERE ABS(delta_seconds) < 300) as under_5min,
			COUNT(*) FILTER (WHERE ABS(delta_seconds) >= 300
						   AND ABS(delta_seconds) < 3600) as five_min_to_1hr,
			COUNT(*) FILTER (WHERE ABS(delta_seconds) >= 3600
						   AND ABS(delta_seconds) < 86400) as one_hr_to_24hr,
			COUNT(*) FILTER (WHERE ABS(delta_seconds) >= 86400) as over_24hr
		FROM temporal_stats
	`

	var stats TemporalPrecisionStats
	err := c.db.QueryRowContext(ctx, query, repoID).Scan(
		&stats.TotalLinks,
		&stats.TightTemporalLinks,
		&stats.Under5Min,
		&stats.FiveMinTo1Hr,
		&stats.OneHrTo24Hr,
		&stats.Over24Hr,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query temporal precision stats: %w", err)
	}

	return &stats, nil
}

// QuerySemanticStrengthStats queries statistics for Semantic Strength component
func (c *StagingClient) QuerySemanticStrengthStats(ctx context.Context, repoID int64) (*SemanticStrengthStats, error) {
	query := `
		WITH semantic_scores AS (
			SELECT
				issue_number,
				pr_number,
				GREATEST(
					COALESCE(CAST(semantic_analysis->>'title_score' AS NUMERIC), 0.0),
					COALESCE(CAST(semantic_analysis->>'body_score' AS NUMERIC), 0.0),
					COALESCE(CAST(semantic_analysis->>'comment_score' AS NUMERIC), 0.0)
				) as max_semantic,
				COALESCE(CAST(semantic_analysis->>'title_score' AS NUMERIC), 0) as title_score,
				COALESCE(CAST(semantic_analysis->>'body_score' AS NUMERIC), 0) as body_score,
				COALESCE(CAST(semantic_analysis->>'comment_score' AS NUMERIC), 0) as comment_score
			FROM github_issue_pr_links
			WHERE repo_id = $1
			  AND semantic_analysis IS NOT NULL
		)
		SELECT
			COALESCE(AVG(max_semantic), 0) as avg_max_semantic,
			COUNT(*) FILTER (WHERE max_semantic >= 0.70) as high_semantic,
			COUNT(*) FILTER (WHERE max_semantic >= 0.50 AND max_semantic < 0.70) as medium_semantic,
			COUNT(*) FILTER (WHERE max_semantic < 0.50) as low_semantic,
			COALESCE(AVG(title_score), 0) as avg_title,
			COALESCE(AVG(body_score), 0) as avg_body,
			COALESCE(AVG(comment_score), 0) as avg_comment,
			COUNT(*) as total_links
		FROM semantic_scores
	`

	var stats SemanticStrengthStats
	err := c.db.QueryRowContext(ctx, query, repoID).Scan(
		&stats.AvgMaxSemantic,
		&stats.HighSemantic,
		&stats.MediumSemantic,
		&stats.LowSemantic,
		&stats.AvgTitle,
		&stats.AvgBody,
		&stats.AvgComment,
		&stats.TotalLinks,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query semantic strength stats: %w", err)
	}

	return &stats, nil
}

// QueryCLQSStatistics queries overall CLQS statistics
func (c *StagingClient) QueryCLQSStatistics(ctx context.Context, repoID int64) (*CLQSStatistics, error) {
	query := `
		WITH eligible_issues AS (
			SELECT i.number
			FROM github_issues i
			WHERE i.repo_id = $1
			  AND i.state = 'closed'
			  AND i.closed_at IS NOT NULL
			  AND NOT EXISTS (
				  SELECT 1 FROM github_issue_no_links n
				  WHERE n.repo_id = i.repo_id
					AND n.issue_number = i.number
					AND n.classification IN ('not_a_bug', 'duplicate', 'wontfix', 'user_action_required')
			  )
		),
		linked_issues AS (
			SELECT DISTINCT issue_number
			FROM github_issue_pr_links
			WHERE repo_id = $1
		)
		SELECT
			(SELECT COUNT(*) FROM github_issues WHERE repo_id = $1 AND state = 'closed') as total_closed,
			(SELECT COUNT(*) FROM eligible_issues) as eligible,
			(SELECT COUNT(*) FROM linked_issues) as linked,
			(SELECT COUNT(*) FROM github_issue_pr_links WHERE repo_id = $1) as total_links,
			COALESCE((SELECT AVG(final_confidence) FROM github_issue_pr_links WHERE repo_id = $1), 0) as avg_confidence
	`

	var stats CLQSStatistics
	err := c.db.QueryRowContext(ctx, query, repoID).Scan(
		&stats.TotalClosed,
		&stats.Eligible,
		&stats.Linked,
		&stats.TotalLinks,
		&stats.AvgConfidence,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query CLQS statistics: %w", err)
	}

	return &stats, nil
}

// StoreCLQSScore stores the CLQS score and details to the database
func (c *StagingClient) StoreCLQSScore(ctx context.Context, score *CLQSScore, components interface{}, recommendations interface{}, labelingOpps interface{}) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Store main CLQS score
	var scoreID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO clqs_scores (
			repo_id, clqs, clqs_grade, clqs_rank, confidence_multiplier,
			link_coverage, confidence_quality, evidence_diversity, temporal_precision, semantic_strength,
			link_coverage_contribution, confidence_quality_contribution, evidence_diversity_contribution,
			temporal_precision_contribution, semantic_strength_contribution,
			total_closed_issues, eligible_issues, linked_issues, total_links, avg_confidence,
			computed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
		RETURNING id
	`,
		score.RepoID, score.CLQS, score.Grade, score.Rank, score.ConfidenceMultiplier,
		score.LinkCoverage, score.ConfidenceQuality, score.EvidenceDiversity, score.TemporalPrecision, score.SemanticStrength,
		score.LinkCoverageContribution, score.ConfidenceQualityContribution, score.EvidenceDiversityContribution,
		score.TemporalPrecisionContribution, score.SemanticStrengthContribution,
		score.TotalClosedIssues, score.EligibleIssues, score.LinkedIssues, score.TotalLinks, score.AvgConfidence,
		score.ComputedAt,
	).Scan(&scoreID)
	if err != nil {
		return fmt.Errorf("failed to insert CLQS score: %w", err)
	}

	// Convert component details to JSONB
	linkCovJSON, _ := json.Marshal(map[string]interface{}{})
	confQualJSON, _ := json.Marshal(map[string]interface{}{})
	evidDivJSON, _ := json.Marshal(map[string]interface{}{})
	tempPrecJSON, _ := json.Marshal(map[string]interface{}{})
	semStrJSON, _ := json.Marshal(map[string]interface{}{})
	recsJSON, _ := json.Marshal(recommendations)
	labelingJSON, _ := json.Marshal(labelingOpps)

	// Store component details
	_, err = tx.ExecContext(ctx, `
		INSERT INTO clqs_component_details (
			clqs_score_id,
			link_coverage_details, confidence_quality_details, evidence_diversity_details,
			temporal_precision_details, semantic_strength_details,
			recommendations, labeling_opportunities
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		scoreID,
		linkCovJSON, confQualJSON, evidDivJSON, tempPrecJSON, semStrJSON,
		recsJSON, labelingJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to insert component details: %w", err)
	}

	return tx.Commit()
}

// GetLatestCLQSScore retrieves the most recent CLQS score for a repository
func (c *StagingClient) GetLatestCLQSScore(ctx context.Context, repoID int64) (*CLQSScore, error) {
	query := `
		SELECT
			repo_id, clqs, clqs_grade, clqs_rank, confidence_multiplier,
			link_coverage, confidence_quality, evidence_diversity, temporal_precision, semantic_strength,
			link_coverage_contribution, confidence_quality_contribution, evidence_diversity_contribution,
			temporal_precision_contribution, semantic_strength_contribution,
			total_closed_issues, eligible_issues, linked_issues, total_links, avg_confidence,
			computed_at
		FROM clqs_scores
		WHERE repo_id = $1
		ORDER BY computed_at DESC
		LIMIT 1
	`

	var score CLQSScore
	err := c.db.QueryRowContext(ctx, query, repoID).Scan(
		&score.RepoID, &score.CLQS, &score.Grade, &score.Rank, &score.ConfidenceMultiplier,
		&score.LinkCoverage, &score.ConfidenceQuality, &score.EvidenceDiversity, &score.TemporalPrecision, &score.SemanticStrength,
		&score.LinkCoverageContribution, &score.ConfidenceQualityContribution, &score.EvidenceDiversityContribution,
		&score.TemporalPrecisionContribution, &score.SemanticStrengthContribution,
		&score.TotalClosedIssues, &score.EligibleIssues, &score.LinkedIssues, &score.TotalLinks, &score.AvgConfidence,
		&score.ComputedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // No cached score
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query latest CLQS score: %w", err)
	}

	return &score, nil
}

// GetRepositoryIDByFullName retrieves the repository ID by full name
func (c *StagingClient) GetRepositoryIDByFullName(ctx context.Context, fullName string) (int64, error) {
	var id int64
	err := c.db.QueryRowContext(ctx, `
		SELECT id FROM github_repositories WHERE full_name = $1
	`, fullName).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("repository '%s' not found: %w", fullName, err)
	}
	return id, nil
}
