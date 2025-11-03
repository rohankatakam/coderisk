package backtest

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/graph"
)

// TemporalMatcher validates temporal linking patterns
type TemporalMatcher struct {
	stagingDB *database.StagingClient
	neo4jDB   *graph.Client
}

// NewTemporalMatcher creates a temporal matcher
func NewTemporalMatcher(stagingDB *database.StagingClient, neo4jDB *graph.Client) *TemporalMatcher {
	return &TemporalMatcher{
		stagingDB: stagingDB,
		neo4jDB:   neo4jDB,
	}
}

// TemporalValidationResult represents the validation of temporal links
type TemporalValidationResult struct {
	IssueNumber       int                    `json:"issue_number"`
	ExpectedPRs       []int                  `json:"expected_prs"`
	DetectedPRs       []TemporalPRMatch      `json:"detected_prs"`
	Matched           bool                   `json:"matched"`
	MatchedPRs        []int                  `json:"matched_prs"`
	MissedPRs         []int                  `json:"missed_prs"`
	FalsePositives    []int                  `json:"false_positives"`
	TimingAnalysis    map[string]interface{} `json:"timing_analysis"`
	ConfidenceScores  map[int]float64        `json:"confidence_scores"`
	EvidenceFound     []string               `json:"evidence_found"`
}

// TemporalPRMatch represents a temporal match between issue and PR
type TemporalPRMatch struct {
	PRNumber       int           `json:"pr_number"`
	PRTitle        string        `json:"pr_title"`
	PRMergedAt     time.Time     `json:"pr_merged_at"`
	IssueClosedAt  time.Time     `json:"issue_closed_at"`
	TimeDelta      time.Duration `json:"time_delta"`
	DeltaSeconds   int64         `json:"delta_seconds"`
	Confidence     float64       `json:"confidence"`
	Evidence       []string      `json:"evidence"`
	SemanticScore  float64       `json:"semantic_score,omitempty"`
}

// ValidateTemporalLinks validates temporal linking for a ground truth test case
func (tm *TemporalMatcher) ValidateTemporalLinks(ctx context.Context, testCase GroundTruthTestCase) (*TemporalValidationResult, error) {
	result := &TemporalValidationResult{
		IssueNumber:      testCase.IssueNumber,
		ExpectedPRs:      testCase.ExpectedLinks.AssociatedPRs,
		DetectedPRs:      []TemporalPRMatch{},
		MatchedPRs:       []int{},
		MissedPRs:        []int{},
		FalsePositives:   []int{},
		TimingAnalysis:   make(map[string]interface{}),
		ConfidenceScores: make(map[int]float64),
		EvidenceFound:    []string{},
	}

	// Get issue timing from primary evidence
	var issueClosedAt time.Time
	if closedAtStr, ok := testCase.PrimaryEvidence["issue_closed_at"].(string); ok {
		var err error
		issueClosedAt, err = time.Parse(time.RFC3339, closedAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse issue_closed_at: %w", err)
		}
	} else {
		return nil, fmt.Errorf("issue_closed_at not found in primary_evidence")
	}

	result.TimingAnalysis["issue_closed_at"] = issueClosedAt.Format(time.RFC3339)

	// Get temporal delta from evidence if available
	expectedDelta := int64(0)
	if delta, ok := testCase.PrimaryEvidence["temporal_delta_seconds"].(float64); ok {
		expectedDelta = int64(delta)
		result.TimingAnalysis["expected_delta_seconds"] = expectedDelta
	}

	// Query all PRs that could temporally match
	query := `
		SELECT
			pr.number,
			pr.title,
			pr.merged_at,
			pr.state
		FROM github_pull_requests pr
		WHERE pr.repo_id = (
			SELECT repo_id FROM github_issues WHERE number = $1 LIMIT 1
		)
		AND pr.merged_at IS NOT NULL
		AND ABS(EXTRACT(EPOCH FROM (pr.merged_at - $2::timestamp))) <= 86400
		ORDER BY ABS(EXTRACT(EPOCH FROM (pr.merged_at - $2::timestamp)))
	`

	rows, err := tm.stagingDB.Query(ctx, query, testCase.IssueNumber, issueClosedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to query temporal PRs: %w", err)
	}
	defer rows.Close()

	// Process each PR
	for rows.Next() {
		var prNumber int
		var prTitle string
		var prMergedAt time.Time
		var prState string

		if err := rows.Scan(&prNumber, &prTitle, &prMergedAt, &prState); err != nil {
			log.Printf("  ⚠️  Failed to scan PR row: %v", err)
			continue
		}

		// Calculate temporal match
		match := tm.calculateTemporalMatch(testCase.IssueNumber, testCase.Title, issueClosedAt, prNumber, prTitle, prMergedAt)
		result.DetectedPRs = append(result.DetectedPRs, match)

		// Track confidence
		result.ConfidenceScores[prNumber] = match.Confidence
		result.EvidenceFound = append(result.EvidenceFound, match.Evidence...)
	}

	// Validate against expected PRs
	for _, expectedPR := range testCase.ExpectedLinks.AssociatedPRs {
		found := false
		for _, detected := range result.DetectedPRs {
			if detected.PRNumber == expectedPR {
				result.MatchedPRs = append(result.MatchedPRs, expectedPR)
				found = true
				break
			}
		}
		if !found {
			result.MissedPRs = append(result.MissedPRs, expectedPR)
		}
	}

	// Find false positives (detected but not expected)
	for _, detected := range result.DetectedPRs {
		expected := false
		for _, expectedPR := range testCase.ExpectedLinks.AssociatedPRs {
			if detected.PRNumber == expectedPR {
				expected = true
				break
			}
		}
		if !expected && detected.Confidence >= 0.60 {
			result.FalsePositives = append(result.FalsePositives, detected.PRNumber)
		}
	}

	result.Matched = len(result.MissedPRs) == 0 && len(result.FalsePositives) == 0

	return result, nil
}

// calculateTemporalMatch calculates confidence and evidence for a temporal match
func (tm *TemporalMatcher) calculateTemporalMatch(
	issueNumber int,
	issueTitle string,
	issueClosedAt time.Time,
	prNumber int,
	prTitle string,
	prMergedAt time.Time,
) TemporalPRMatch {
	match := TemporalPRMatch{
		PRNumber:      prNumber,
		PRTitle:       prTitle,
		PRMergedAt:    prMergedAt,
		IssueClosedAt: issueClosedAt,
		Evidence:      []string{},
	}

	// Calculate time delta
	delta := issueClosedAt.Sub(prMergedAt)
	if delta < 0 {
		delta = -delta
	}
	match.TimeDelta = delta
	match.DeltaSeconds = int64(delta.Seconds())

	// Apply temporal scoring based on LINKING_PATTERNS.md
	baseConfidence := 0.50 // Default baseline

	if delta < 5*time.Minute {
		baseConfidence = 0.75 // Strong temporal correlation
		match.Evidence = append(match.Evidence, "temporal_match_5min")
	} else if delta < 1*time.Hour {
		baseConfidence = 0.65 // Moderate temporal correlation
		match.Evidence = append(match.Evidence, "temporal_match_1hr")
	} else if delta < 24*time.Hour {
		baseConfidence = 0.55 // Weak temporal correlation
		match.Evidence = append(match.Evidence, "temporal_match_24hr")
	}

	// Apply semantic boost if titles match
	semanticScore := graph.ComputeSemanticSimilarity(issueTitle, prTitle)
	match.SemanticScore = semanticScore

	if semanticScore >= 0.70 {
		baseConfidence += 0.15
		match.Evidence = append(match.Evidence, "semantic_high")
	} else if semanticScore >= 0.50 {
		baseConfidence += 0.10
		match.Evidence = append(match.Evidence, "semantic_medium")
	} else if semanticScore >= 0.30 {
		baseConfidence += 0.05
		match.Evidence = append(match.Evidence, "semantic_low")
	}

	// Cap at 0.98 as per LINKING_PATTERNS.md
	if baseConfidence > 0.98 {
		baseConfidence = 0.98
	}

	match.Confidence = baseConfidence

	return match
}

// ValidateAllTemporalCases validates all temporal test cases from ground truth
func (tm *TemporalMatcher) ValidateAllTemporalCases(ctx context.Context, groundTruth *GroundTruth) ([]TemporalValidationResult, error) {
	log.Printf("🕐 Validating temporal linking patterns...")

	results := []TemporalValidationResult{}

	for _, testCase := range groundTruth.TestCases {
		// Only validate temporal cases
		hasTemporal := false
		for _, pattern := range testCase.LinkingPatterns {
			if pattern == "temporal" {
				hasTemporal = true
				break
			}
		}

		if !hasTemporal {
			continue
		}

		log.Printf("  Testing Issue #%d (temporal)", testCase.IssueNumber)

		result, err := tm.ValidateTemporalLinks(ctx, testCase)
		if err != nil {
			log.Printf("    ❌ Error: %v", err)
			continue
		}

		if result.Matched {
			confidence := 0.0
			if len(result.MatchedPRs) > 0 && result.ConfidenceScores[result.MatchedPRs[0]] > 0 {
				confidence = result.ConfidenceScores[result.MatchedPRs[0]]
			}
			log.Printf("    ✅ Matched: %v (confidence: %.2f)",
				result.MatchedPRs,
				confidence)
		} else {
			log.Printf("    ❌ Failed:")
			if len(result.MissedPRs) > 0 {
				log.Printf("       Missed PRs: %v", result.MissedPRs)
			}
			if len(result.FalsePositives) > 0 {
				log.Printf("       False Positives: %v", result.FalsePositives)
			}
		}

		results = append(results, *result)
	}

	return results, nil
}

// CalculateTemporalMetrics computes metrics for temporal linking
func (tm *TemporalMatcher) CalculateTemporalMetrics(results []TemporalValidationResult) map[string]interface{} {
	metrics := make(map[string]interface{})

	totalCases := len(results)
	matched := 0
	totalExpected := 0
	totalDetected := 0
	totalMissed := 0
	totalFP := 0

	var confidenceSum float64
	confidenceCount := 0

	for _, result := range results {
		if result.Matched {
			matched++
		}

		totalExpected += len(result.ExpectedPRs)
		totalDetected += len(result.DetectedPRs)
		totalMissed += len(result.MissedPRs)
		totalFP += len(result.FalsePositives)

		// Average confidence across all detected links
		for _, score := range result.ConfidenceScores {
			confidenceSum += score
			confidenceCount++
		}
	}

	metrics["total_cases"] = totalCases
	metrics["matched_cases"] = matched
	metrics["match_rate"] = float64(matched) / float64(totalCases)

	metrics["total_expected_links"] = totalExpected
	metrics["total_detected_links"] = totalDetected
	metrics["total_missed_links"] = totalMissed
	metrics["total_false_positives"] = totalFP

	if totalDetected > 0 {
		metrics["precision"] = float64(totalExpected-totalMissed) / float64(totalDetected)
	} else {
		metrics["precision"] = 0.0
	}

	if totalExpected > 0 {
		metrics["recall"] = float64(totalExpected-totalMissed) / float64(totalExpected)
	} else {
		metrics["recall"] = 0.0
	}

	if confidenceCount > 0 {
		metrics["avg_confidence"] = confidenceSum / float64(confidenceCount)
	} else {
		metrics["avg_confidence"] = 0.0
	}

	precision := metrics["precision"].(float64)
	recall := metrics["recall"].(float64)
	if precision+recall > 0 {
		metrics["f1_score"] = 2 * (precision * recall) / (precision + recall)
	} else {
		metrics["f1_score"] = 0.0
	}

	return metrics
}
