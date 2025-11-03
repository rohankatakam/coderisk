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
// Applies semantic validation to filter out false positives
func (tc *TemporalCorrelator) FindTemporalMatches(ctx context.Context, repoID int64) ([]TemporalMatch, error) {
	log.Printf("🕐 Finding temporal correlations with semantic validation...")

	matches := []TemporalMatch{}
	semanticMatcher := NewSemanticMatcher()

	// Counters for logging
	totalCandidates := 0
	semanticRejections := 0
	semanticBoosts := 0

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
			totalCandidates++

			// Create temporal match
			match := tc.createTemporalMatch(issue.Number, *issue.ClosedAt, "pr", fmt.Sprintf("%d", pr.Number), pr.MergedAt)

			// Skip if temporal confidence is zero (outside time windows)
			if match.Confidence == 0.0 {
				semanticRejections++
				continue
			}

			// Validate with semantic similarity (using improved title+body matching)
			similarity := semanticMatcher.CalculateIssueToPRSimilarity(issue.Title, issue.Body, pr.Title, pr.Body)

			// Adaptive filtering based on semantic relevance:
			// - High similarity (≥10%): Accept all temporal matches
			// - Low similarity (<10%): Only accept if very close in time (<1 hour)
			if similarity < 0.10 && match.Delta >= 1*time.Hour {
				semanticRejections++
				continue // Reject: low relevance and not very close in time
			}

			// Apply semantic boost if similarity is high
			if similarity >= 0.20 {
				match.Confidence = min(match.Confidence+0.10, 0.98)
				match.Evidence = append(match.Evidence, "semantic_boost")
				semanticBoosts++
			} else if similarity >= 0.10 {
				match.Confidence = min(match.Confidence+0.05, 0.98)
				match.Evidence = append(match.Evidence, "semantic_boost")
				semanticBoosts++
			}

			matches = append(matches, match)
		}

		// Find commits around the same time
		commits, err := tc.getCommitsNear(ctx, repoID, *issue.ClosedAt, 24*time.Hour)
		if err != nil {
			log.Printf("  ⚠️  Failed to find commits for issue #%d: %v", issue.Number, err)
			continue
		}

		for _, commit := range commits {
			totalCandidates++

			// Create temporal match
			match := tc.createTemporalMatch(issue.Number, *issue.ClosedAt, "commit", commit.SHA, commit.AuthorDate)

			// Skip if temporal confidence is zero (outside time windows)
			if match.Confidence == 0.0 {
				semanticRejections++
				continue
			}

			// Validate with semantic similarity (using improved title+body matching)
			similarity := semanticMatcher.CalculateIssueToCommitSimilarity(issue.Title, issue.Body, commit.Message)

			// Adaptive filtering based on semantic relevance:
			// - High similarity (≥10%): Accept all temporal matches
			// - Low similarity (<10%): Only accept if very close in time (<1 hour)
			if similarity < 0.10 && match.Delta >= 1*time.Hour {
				semanticRejections++
				continue // Reject: low relevance and not very close in time
			}

			// Apply semantic boost if similarity is high
			if similarity >= 0.20 {
				match.Confidence = min(match.Confidence+0.10, 0.98)
				match.Evidence = append(match.Evidence, "semantic_boost")
				semanticBoosts++
			} else if similarity >= 0.10 {
				match.Confidence = min(match.Confidence+0.05, 0.98)
				match.Evidence = append(match.Evidence, "semantic_boost")
				semanticBoosts++
			}

			matches = append(matches, match)
		}
	}

	log.Printf("  ✓ Found %d temporal matches (%d candidates, %d rejected by semantic filter, %d boosted)",
		len(matches), totalCandidates, semanticRejections, semanticBoosts)
	return matches, nil
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// StoreTemporalMatches stores temporal matches as IssueCommitRef entries in the database
func (tc *TemporalCorrelator) StoreTemporalMatches(ctx context.Context, repoID int64, matches []TemporalMatch) error {
	if len(matches) == 0 {
		return nil
	}

	// Convert temporal matches to IssueCommitRef entries
	refs := make([]database.IssueCommitRef, 0, len(matches))
	for _, match := range matches {
		ref := database.IssueCommitRef{
			RepoID:          repoID,
			IssueNumber:     match.IssueNumber,
			Action:          "associated_with", // Temporal matches are associations, not explicit fixes
			Confidence:      match.Confidence,
			DetectionMethod: "temporal",
			ExtractedFrom:   fmt.Sprintf("temporal_correlation_%s", match.Evidence[0]),
			Evidence:        match.Evidence, // Store evidence tags (e.g., ["temporal_match_5min"])
		}

		// Set target based on type
		if match.TargetType == "pr" {
			prNum := 0
			if _, err := fmt.Sscanf(match.TargetID, "%d", &prNum); err == nil {
				ref.PRNumber = &prNum
			}
		} else if match.TargetType == "commit" {
			ref.CommitSHA = &match.TargetID
		}

		refs = append(refs, ref)
	}

	// Store in database
	if err := tc.stagingDB.StoreIssueCommitRefs(ctx, refs); err != nil {
		return fmt.Errorf("failed to store temporal matches: %w", err)
	}

	log.Printf("  ✓ Stored %d temporal matches to database", len(refs))
	return nil
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
	issues, err := tc.stagingDB.GetClosedIssues(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get closed issues: %w", err)
	}

	result := make([]IssueInfo, len(issues))
	for i, issue := range issues {
		result[i] = IssueInfo{
			Number:   issue.Number,
			Title:    issue.Title,
			Body:     issue.Body,
			State:    issue.State,
			ClosedAt: issue.ClosedAt,
		}
	}
	return result, nil
}

// getPRsMergedNear finds PRs merged within a time window
func (tc *TemporalCorrelator) getPRsMergedNear(ctx context.Context, repoID int64, targetTime time.Time, window time.Duration) ([]PRInfo, error) {
	prs, err := tc.stagingDB.GetPRsMergedNear(ctx, repoID, targetTime, window)
	if err != nil {
		return nil, fmt.Errorf("failed to get PRs merged near time: %w", err)
	}

	result := make([]PRInfo, len(prs))
	for i, pr := range prs {
		mergedAt := time.Time{}
		if pr.MergedAt != nil {
			mergedAt = *pr.MergedAt
		}
		result[i] = PRInfo{
			Number:   pr.Number,
			Title:    pr.Title,
			Body:     pr.Body,
			State:    pr.State,
			MergedAt: mergedAt,
		}
	}
	return result, nil
}

// getCommitsNear finds commits created within a time window
func (tc *TemporalCorrelator) getCommitsNear(ctx context.Context, repoID int64, targetTime time.Time, window time.Duration) ([]CommitInfo, error) {
	commits, err := tc.stagingDB.GetCommitsNear(ctx, repoID, targetTime, window)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits near time: %w", err)
	}

	result := make([]CommitInfo, len(commits))
	for i, commit := range commits {
		result[i] = CommitInfo{
			SHA:        commit.SHA,
			Message:    commit.Message,
			AuthorDate: commit.AuthorDate,
		}
	}
	return result, nil
}

// Helper types
type IssueInfo struct {
	Number    int
	Title     string
	Body      string // Added for semantic matching
	State     string
	ClosedAt  *time.Time
}

type PRInfo struct {
	Number    int
	Title     string
	Body      string // Added for semantic matching
	State     string
	MergedAt  time.Time
}

type CommitInfo struct {
	SHA        string
	Message    string
	AuthorDate time.Time
}
