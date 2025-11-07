package clqs

import (
	"context"
	"fmt"
)

// CalculateLinkCoverage computes Component 1: Link Coverage (35% weight)
//
// Measures what percentage of eligible closed issues have been linked to PRs.
// Excludes issues that don't require code fixes (not_a_bug, duplicate, wontfix, user_action_required).
//
// Score = (linked_issues / eligible_issues) × 100
func (c *Calculator) CalculateLinkCoverage(ctx context.Context, repoID int64) (*ComponentBreakdown, error) {
	// Query eligible issues and linked issues
	stats, err := c.db.QueryLinkCoverageStats(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query link coverage stats: %w", err)
	}

	// Handle boundary case
	if stats.EligibleIssues == 0 {
		return nil, fmt.Errorf("no eligible issues requiring code fixes")
	}

	// Calculate coverage rate
	coverageRate := float64(stats.LinkedIssues) / float64(stats.EligibleIssues)
	score := coverageRate * 100.0
	contribution := score * WeightLinkCoverage

	// Determine status
	status := determineStatus(score, 90, 80, 70)

	// Build details
	details := LinkCoverageDetails{
		TotalClosedIssues: stats.TotalClosed,
		ExcludedIssues:    stats.TotalClosed - stats.EligibleIssues,
		EligibleIssues:    stats.EligibleIssues,
		LinkedIssues:      stats.LinkedIssues,
		UnlinkedIssues:    stats.UnlinkedIssues,
		CoverageRate:      coverageRate,
	}

	return &ComponentBreakdown{
		Name:         "Link Coverage",
		Score:        score,
		Weight:       WeightLinkCoverage,
		Contribution: contribution,
		Status:       status,
		Details:      details,
	}, nil
}

// determineStatus determines status based on thresholds
func determineStatus(score, excellentThreshold, goodThreshold, fairThreshold float64) string {
	switch {
	case score >= excellentThreshold:
		return "Excellent"
	case score >= goodThreshold:
		return "Good"
	case score >= fairThreshold:
		return "Fair"
	default:
		return "Poor"
	}
}
