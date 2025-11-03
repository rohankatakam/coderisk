package graph

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rohankatakam/coderisk/internal/database"
)

// TemporalCorrelator finds temporal correlations between issues, PRs, and commits
type TemporalCorrelator struct {
	stagingDB *database.StagingClient
	neo4jDB   *Client
}

// NewTemporalCorrelator creates a temporal correlator
func NewTemporalCorrelator(stagingDB *database.StagingClient, neo4jDB *Client) *TemporalCorrelator {
	return &TemporalCorrelator{
		stagingDB: stagingDB,
		neo4jDB:   neo4jDB,
	}
}

// TemporalMatch represents a temporal correlation between an issue and PR/commit
type TemporalMatch struct {
	IssueNumber   int
	IssueClosedAt time.Time
	TargetType    string // "pr" or "commit"
	TargetID      string // PR number or commit SHA
	TargetTime    time.Time
	Delta         time.Duration
	Confidence    float64
	Evidence      []string
}

// FindTemporalMatches finds PRs/commits that occurred near the time an issue was closed
func (tc *TemporalCorrelator) FindTemporalMatches(ctx context.Context, repoID int64) ([]TemporalMatch, error) {
	log.Printf("🕐 Finding temporal correlations...")

	matches := []TemporalMatch{}

	// Get all closed issues with timestamps
	issues, err := tc.getClosedIssues(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get closed issues: %w", err)
	}

	log.Printf("  Found %d closed issues", len(issues))

	// For each issue, find PRs merged around the same time
	for _, issue := range issues {
		if issue.ClosedAt == nil {
			continue
		}

		// Find PRs merged within 24 hours of issue close
		prs, err := tc.getPRsMergedNear(ctx, repoID, *issue.ClosedAt, 24*time.Hour)
		if err != nil {
			log.Printf("  ⚠️  Failed to find PRs for issue #%d: %v", issue.Number, err)
			continue
		}

		for _, pr := range prs {
			match := tc.createTemporalMatch(issue.Number, *issue.ClosedAt, "pr", fmt.Sprintf("%d", pr.Number), pr.MergedAt)
			if match.Confidence > 0.0 {
				matches = append(matches, match)
			}
		}

		// Find commits around the same time
		commits, err := tc.getCommitsNear(ctx, repoID, *issue.ClosedAt, 24*time.Hour)
		if err != nil {
			log.Printf("  ⚠️  Failed to find commits for issue #%d: %v", issue.Number, err)
			continue
		}

		for _, commit := range commits {
			match := tc.createTemporalMatch(issue.Number, *issue.ClosedAt, "commit", commit.SHA, commit.AuthorDate)
			if match.Confidence > 0.0 {
				matches = append(matches, match)
			}
		}
	}

	log.Printf("  ✓ Found %d temporal matches", len(matches))
	return matches, nil
}

// createTemporalMatch calculates confidence and evidence for a temporal match
func (tc *TemporalCorrelator) createTemporalMatch(
	issueNumber int,
	issueClosedAt time.Time,
	targetType string,
	targetID string,
	targetTime time.Time,
) TemporalMatch {

	delta := issueClosedAt.Sub(targetTime)
	if delta < 0 {
		delta = -delta // Absolute value
	}

	match := TemporalMatch{
		IssueNumber:   issueNumber,
		IssueClosedAt: issueClosedAt,
		TargetType:    targetType,
		TargetID:      targetID,
		TargetTime:    targetTime,
		Delta:         delta,
		Confidence:    0.0,
		Evidence:      []string{},
	}

	// Apply temporal scoring
	if delta < 5*time.Minute {
		match.Confidence = 0.75 // High confidence for <5 min
		match.Evidence = append(match.Evidence, "temporal_match_5min")
	} else if delta < 1*time.Hour {
		match.Confidence = 0.65 // Medium confidence for <1 hr
		match.Evidence = append(match.Evidence, "temporal_match_1hr")
	} else if delta < 24*time.Hour {
		match.Confidence = 0.55 // Low confidence for <24 hr
		match.Evidence = append(match.Evidence, "temporal_match_24hr")
	}

	return match
}

// ApplyTemporalBoost adds temporal boost to an existing reference
func ApplyTemporalBoost(confidence float64, evidence []string, issueClosedAt, targetTime time.Time) (float64, []string) {
	if issueClosedAt.IsZero() || targetTime.IsZero() {
		return confidence, evidence
	}

	delta := issueClosedAt.Sub(targetTime)
	if delta < 0 {
		delta = -delta
	}

	if delta < 5*time.Minute {
		confidence += 0.15
		evidence = append(evidence, "temporal_match_5min")
	} else if delta < 1*time.Hour {
		confidence += 0.10
		evidence = append(evidence, "temporal_match_1hr")
	} else if delta < 24*time.Hour {
		confidence += 0.05
		evidence = append(evidence, "temporal_match_24hr")
	}

	// Cap at 0.98
	if confidence > 0.98 {
		confidence = 0.98
	}

	return confidence, evidence
}

// getClosedIssues retrieves all closed issues with timestamps
func (tc *TemporalCorrelator) getClosedIssues(ctx context.Context, repoID int64) ([]IssueInfo, error) {
	// Use FetchUnprocessedIssues as a template - we need all closed issues
	// For now, return empty slice - this will be implemented when integrating
	return []IssueInfo{}, nil
}

// getPRsMergedNear finds PRs merged within a time window
func (tc *TemporalCorrelator) getPRsMergedNear(ctx context.Context, repoID int64, targetTime time.Time, window time.Duration) ([]PRInfo, error) {
	// TODO: Implement when integrating - need to add Query method to StagingClient
	return []PRInfo{}, nil
}

// getCommitsNear finds commits created within a time window
func (tc *TemporalCorrelator) getCommitsNear(ctx context.Context, repoID int64, targetTime time.Time, window time.Duration) ([]CommitInfo, error) {
	// TODO: Implement when integrating - need to add Query method to StagingClient
	return []CommitInfo{}, nil
}

// Helper types
type IssueInfo struct {
	Number    int
	Title     string
	State     string
	ClosedAt  *time.Time
}

type PRInfo struct {
	Number    int
	Title     string
	State     string
	MergedAt  time.Time
}

type CommitInfo struct {
	SHA        string
	Message    string
	AuthorDate time.Time
}
