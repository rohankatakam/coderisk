package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/rohankatakam/coderisk/internal/linking/types"
)

// StoreLinkOutput stores a validated issue-PR link
func (c *StagingClient) StoreLinkOutput(ctx context.Context, repoID int64, link types.LinkOutput) error {
	query := `
		INSERT INTO github_issue_pr_links (
			repo_id, issue_number, pr_number, detection_method,
			final_confidence, link_quality, confidence_breakdown,
			evidence_sources, comprehensive_rationale,
			semantic_analysis, temporal_analysis, flags, metadata,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
		ON CONFLICT (repo_id, issue_number, pr_number)
		DO UPDATE SET
			final_confidence = GREATEST(github_issue_pr_links.final_confidence, EXCLUDED.final_confidence),
			confidence_breakdown = EXCLUDED.confidence_breakdown,
			evidence_sources = EXCLUDED.evidence_sources,
			comprehensive_rationale = EXCLUDED.comprehensive_rationale,
			semantic_analysis = EXCLUDED.semantic_analysis,
			temporal_analysis = EXCLUDED.temporal_analysis,
			flags = EXCLUDED.flags,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`

	// Marshal JSON fields
	confidenceBreakdown, _ := json.Marshal(link.ConfidenceBreakdown)
	semanticAnalysis, _ := json.Marshal(link.SemanticAnalysis)
	temporalAnalysis, _ := json.Marshal(link.TemporalAnalysis)
	flags, _ := json.Marshal(link.Flags)
	metadata, _ := json.Marshal(link.Metadata)

	_, err := c.db.ExecContext(ctx, query,
		repoID,
		link.IssueNumber,
		link.PRNumber,
		string(link.DetectionMethod),
		link.FinalConfidence,
		string(link.LinkQuality),
		confidenceBreakdown,
		pq.Array(link.EvidenceSources),
		link.ComprehensiveRationale,
		semanticAnalysis,
		temporalAnalysis,
		flags,
		metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to store link output: %w", err)
	}

	return nil
}

// StoreNoLinkOutput stores an issue with no PR links found
func (c *StagingClient) StoreNoLinkOutput(ctx context.Context, repoID int64, noLink types.NoLinkOutput) error {
	query := `
		INSERT INTO github_issue_no_links (
			repo_id, issue_number, no_links_reason, classification,
			classification_confidence, classification_rationale,
			conversation_summary, candidates_evaluated, best_candidate_score,
			safety_brake_reason, issue_closed_at, analyzed_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		ON CONFLICT (repo_id, issue_number)
		DO UPDATE SET
			no_links_reason = EXCLUDED.no_links_reason,
			classification = EXCLUDED.classification,
			classification_confidence = EXCLUDED.classification_confidence,
			classification_rationale = EXCLUDED.classification_rationale,
			conversation_summary = EXCLUDED.conversation_summary,
			candidates_evaluated = EXCLUDED.candidates_evaluated,
			best_candidate_score = EXCLUDED.best_candidate_score,
			safety_brake_reason = EXCLUDED.safety_brake_reason,
			analyzed_at = EXCLUDED.analyzed_at,
			updated_at = NOW()
	`

	_, err := c.db.ExecContext(ctx, query,
		repoID,
		noLink.IssueNumber,
		string(noLink.NoLinksReason),
		string(noLink.Classification),
		noLink.ClassificationConfidence,
		noLink.ClassificationRationale,
		noLink.ConversationSummary,
		noLink.CandidatesEvaluated,
		noLink.BestCandidateScore,
		noLink.SafetyBrakeReason,
		noLink.IssueClosedAt,
		noLink.AnalyzedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store no-link output: %w", err)
	}

	return nil
}

// GetIssuePRLinks retrieves all links for a repository
func (c *StagingClient) GetIssuePRLinks(ctx context.Context, repoID int64) ([]types.LinkOutput, error) {
	query := `
		SELECT issue_number, pr_number, detection_method, final_confidence,
			   link_quality, confidence_breakdown, evidence_sources,
			   comprehensive_rationale, semantic_analysis, temporal_analysis,
			   flags, metadata
		FROM github_issue_pr_links
		WHERE repo_id = $1
		ORDER BY final_confidence DESC, issue_number, pr_number
	`

	rows, err := c.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query links: %w", err)
	}
	defer rows.Close()

	var links []types.LinkOutput
	for rows.Next() {
		var link types.LinkOutput
		var confidenceBreakdown, semanticAnalysis, temporalAnalysis, flags, metadata []byte
		var evidenceSources []string

		if err := rows.Scan(
			&link.IssueNumber,
			&link.PRNumber,
			&link.DetectionMethod,
			&link.FinalConfidence,
			&link.LinkQuality,
			&confidenceBreakdown,
			pq.Array(&evidenceSources),
			&link.ComprehensiveRationale,
			&semanticAnalysis,
			&temporalAnalysis,
			&flags,
			&metadata,
		); err != nil {
			return nil, fmt.Errorf("failed to scan link: %w", err)
		}

		link.EvidenceSources = evidenceSources

		// Unmarshal JSON fields
		json.Unmarshal(confidenceBreakdown, &link.ConfidenceBreakdown)
		json.Unmarshal(semanticAnalysis, &link.SemanticAnalysis)
		json.Unmarshal(temporalAnalysis, &link.TemporalAnalysis)
		json.Unmarshal(flags, &link.Flags)
		json.Unmarshal(metadata, &link.Metadata)

		links = append(links, link)
	}

	return links, rows.Err()
}

// GetIssuesWithoutLinks retrieves issues that haven't been processed for linking yet
func (c *StagingClient) GetIssuesWithoutLinks(ctx context.Context, repoID int64) ([]int, error) {
	query := `
		SELECT i.number
		FROM github_issues i
		WHERE i.repo_id = $1
		  AND i.state = 'closed'
		  AND i.closed_at IS NOT NULL
		  AND NOT EXISTS (
		      SELECT 1 FROM github_issue_pr_links l WHERE l.repo_id = i.repo_id AND l.issue_number = i.number
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM github_issue_no_links n WHERE n.repo_id = i.repo_id AND n.issue_number = i.number
		  )
		ORDER BY i.closed_at DESC
	`

	rows, err := c.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query unlinked issues: %w", err)
	}
	defer rows.Close()

	var issueNumbers []int
	for rows.Next() {
		var num int
		if err := rows.Scan(&num); err != nil {
			return nil, fmt.Errorf("failed to scan issue number: %w", err)
		}
		issueNumbers = append(issueNumbers, num)
	}

	return issueNumbers, rows.Err()
}

// GetIssueByNumber retrieves full issue data for linking
func (c *StagingClient) GetIssueByNumber(ctx context.Context, repoID int64, issueNumber int) (*types.IssueData, error) {
	// Get issue metadata
	query := `
		SELECT number, title, body, state, labels, created_at, closed_at
		FROM github_issues
		WHERE repo_id = $1 AND number = $2
	`

	var issue types.IssueData
	var labelsJSON []byte
	err := c.db.QueryRowContext(ctx, query, repoID, issueNumber).Scan(
		&issue.IssueNumber,
		&issue.Title,
		&issue.Body,
		&issue.State,
		&labelsJSON,
		&issue.CreatedAt,
		&issue.ClosedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("issue #%d not found", issueNumber)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query issue: %w", err)
	}

	// Parse labels
	if len(labelsJSON) > 0 {
		var labels []map[string]interface{}
		if err := json.Unmarshal(labelsJSON, &labels); err == nil {
			for _, label := range labels {
				if name, ok := label["name"].(string); ok {
					issue.Labels = append(issue.Labels, name)
				}
			}
		}
	}

	// Get comments
	commentsQuery := `
		SELECT body, user_login, author_association, created_at
		FROM github_issue_comments
		WHERE issue_id = (SELECT id FROM github_issues WHERE repo_id = $1 AND number = $2)
		ORDER BY created_at ASC
	`

	rows, err := c.db.QueryContext(ctx, commentsQuery, repoID, issueNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to query comments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var comment types.CommentData
		if err := rows.Scan(&comment.Body, &comment.Author, &comment.AuthorRole, &comment.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}

		// Apply truncation if needed (1000 first + 500 last if > 2000)
		if len(comment.Body) > 2000 {
			comment.Body = comment.Body[:1000] + "..." + comment.Body[len(comment.Body)-500:]
			comment.WasTruncated = true
		}

		issue.Comments = append(issue.Comments, comment)
	}

	return &issue, rows.Err()
}

// GetPRByNumber retrieves full PR data for linking
func (c *StagingClient) GetPRByNumber(ctx context.Context, repoID int64, prNumber int) (*types.PRData, error) {
	// Get PR metadata
	query := `
		SELECT number, title, body, state, merged, merged_at, created_at, merge_commit_sha
		FROM github_pull_requests
		WHERE repo_id = $1 AND number = $2
	`

	var pr types.PRData
	err := c.db.QueryRowContext(ctx, query, repoID, prNumber).Scan(
		&pr.PRNumber,
		&pr.Title,
		&pr.Body,
		&pr.State,
		&pr.Merged,
		&pr.MergedAt,
		&pr.CreatedAt,
		&pr.MergeCommitSHA,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("PR #%d not found", prNumber)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query PR: %w", err)
	}

	// Get PR files
	filesQuery := `
		SELECT filename, status, additions, deletions, previous_filename
		FROM github_pr_files
		WHERE pr_id = (SELECT id FROM github_pull_requests WHERE repo_id = $1 AND number = $2)
		ORDER BY changes DESC
	`

	rows, err := c.db.QueryContext(ctx, filesQuery, repoID, prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to query PR files: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var file types.PRFileData
		if err := rows.Scan(&file.Filename, &file.Status, &file.Additions, &file.Deletions, &file.PreviousFilename); err != nil {
			return nil, fmt.Errorf("failed to scan PR file: %w", err)
		}
		pr.Files = append(pr.Files, file)
	}

	return &pr, rows.Err()
}

// GetAllClosedIssues retrieves all closed issues for a repository
func (c *StagingClient) GetAllClosedIssues(ctx context.Context, repoID int64) ([]types.IssueData, error) {
	query := `
		SELECT number, title, body, state, labels, created_at, closed_at
		FROM github_issues
		WHERE repo_id = $1 AND state = 'closed' AND closed_at IS NOT NULL
		ORDER BY closed_at DESC
	`

	rows, err := c.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query closed issues: %w", err)
	}
	defer rows.Close()

	var issues []types.IssueData
	for rows.Next() {
		var issue types.IssueData
		var labelsJSON []byte

		if err := rows.Scan(
			&issue.IssueNumber,
			&issue.Title,
			&issue.Body,
			&issue.State,
			&labelsJSON,
			&issue.CreatedAt,
			&issue.ClosedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan issue: %w", err)
		}

		// Parse labels
		if len(labelsJSON) > 0 {
			var labels []map[string]interface{}
			if err := json.Unmarshal(labelsJSON, &labels); err == nil {
				for _, label := range labels {
					if name, ok := label["name"].(string); ok {
						issue.Labels = append(issue.Labels, name)
					}
				}
			}
		}

		issues = append(issues, issue)
	}

	return issues, rows.Err()
}

// ComputeDORAMetrics calculates repository-level DORA metrics
func (c *StagingClient) ComputeDORAMetrics(ctx context.Context, repoID int64, days int) (*types.DORAMetrics, error) {
	// Get cutoff time (default 90 days)
	cutoff := time.Now().AddDate(0, 0, -days)
	if days == 0 {
		// Use a very old date to get all data
		cutoff = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	query := `
		SELECT
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (merged_at - created_at))/3600.0), 0) as median_lead_time_hours,
			COUNT(*) as sample_size
		FROM github_pull_requests
		WHERE repo_id = $1
		  AND merged_at IS NOT NULL
		  AND merged_at >= $2
	`

	var metrics types.DORAMetrics
	err := c.db.QueryRowContext(ctx, query, repoID, cutoff).Scan(
		&metrics.MedianLeadTimeHours,
		&metrics.SampleSize,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to compute DORA metrics: %w", err)
	}

	metrics.MedianPRLifespanHours = metrics.MedianLeadTimeHours // Same calculation
	metrics.ComputedAt = time.Now()
	metrics.InsufficientHistory = metrics.SampleSize < 10

	return &metrics, nil
}

// StoreDORAMetrics stores DORA metrics in the database
func (c *StagingClient) StoreDORAMetrics(ctx context.Context, repoID int64, metrics *types.DORAMetrics) error {
	query := `
		INSERT INTO github_dora_metrics (
			repo_id, median_lead_time_hours, median_pr_lifespan_hours,
			sample_size, insufficient_history, timeline_events_fetched,
			cross_reference_links_found, computed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (repo_id, computed_at) DO NOTHING
	`

	_, err := c.db.ExecContext(ctx, query,
		repoID,
		metrics.MedianLeadTimeHours,
		metrics.MedianPRLifespanHours,
		metrics.SampleSize,
		metrics.InsufficientHistory,
		metrics.TimelineEventsFetched,
		metrics.CrossReferenceLinksFound,
		metrics.ComputedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store DORA metrics: %w", err)
	}

	return nil
}
