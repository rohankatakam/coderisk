package clqs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rohankatakam/coderisk/internal/database"
)

// Calculator orchestrates CLQS score calculation
type Calculator struct {
	db *database.StagingClient
}

// NewCalculator creates a new CLQS calculator
func NewCalculator(db *database.StagingClient) *Calculator {
	return &Calculator{
		db: db,
	}
}

// Calculate computes the complete CLQS score for a repository
func (c *Calculator) Calculate(ctx context.Context, repoID int64) (*CLQSReport, error) {
	log.Printf("Calculating CLQS for repository ID %d", repoID)

	// Validate prerequisites
	if err := c.validatePrerequisites(ctx, repoID); err != nil {
		return nil, fmt.Errorf("prerequisite check failed: %w", err)
	}

	// Calculate all components
	comp1, err := c.CalculateLinkCoverage(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("component 1 (Link Coverage) failed: %w", err)
	}

	comp2, err := c.CalculateConfidenceQuality(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("component 2 (Confidence Quality) failed: %w", err)
	}

	comp3, err := c.CalculateEvidenceDiversity(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("component 3 (Evidence Diversity) failed: %w", err)
	}

	comp4, err := c.CalculateTemporalPrecision(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("component 4 (Temporal Precision) failed: %w", err)
	}

	comp5, err := c.CalculateSemanticStrength(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("component 5 (Semantic Strength) failed: %w", err)
	}

	// Calculate composite CLQS
	clqs := comp1.Contribution + comp2.Contribution + comp3.Contribution +
		comp4.Contribution + comp5.Contribution

	// Assign grade and rank
	grade, rank := assignGradeAndRank(clqs)
	confidenceMultiplier := calculateConfidenceMultiplier(clqs)

	// Get repository info
	repoInfo, err := c.db.GetRepositoryInfo(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository info: %w", err)
	}

	// Generate statistics
	stats, err := c.generateStatistics(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate statistics: %w", err)
	}

	// Generate recommendations
	recommendations := generateRecommendations([]*ComponentBreakdown{
		comp1, comp2, comp3, comp4, comp5,
	}, stats)

	// Calculate labeling opportunities
	labelingOpps := calculateLabelingOpportunities(comp2, clqs)

	// Build report
	report := &CLQSReport{
		Repository: RepositoryInfo{
			ID:         repoID,
			FullName:   repoInfo.FullName,
			AnalyzedAt: time.Now(),
		},
		Overall: OverallScore{
			CLQS:                 clqs,
			Grade:                grade,
			Rank:                 rank,
			ConfidenceMultiplier: confidenceMultiplier,
		},
		Components:            []*ComponentBreakdown{comp1, comp2, comp3, comp4, comp5},
		Statistics:            *stats,
		Recommendations:       recommendations,
		LabelingOpportunities: labelingOpps,
	}

	// Store results
	if err := c.storeResults(ctx, repoID, report); err != nil {
		log.Printf("Warning: failed to store CLQS results: %v", err)
	}

	log.Printf("CLQS calculation complete: %.1f (%s - %s)", clqs, grade, rank)

	return report, nil
}

// validatePrerequisites checks if the repository has sufficient data for CLQS calculation
func (c *Calculator) validatePrerequisites(ctx context.Context, repoID int64) error {
	// Check repository exists
	exists, err := c.db.RepositoryExists(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to check repository: %w", err)
	}
	if !exists {
		return fmt.Errorf("repository %d not found", repoID)
	}

	// Check for closed issues
	count, err := c.db.CountClosedIssues(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to count closed issues: %w", err)
	}
	if count < 10 {
		return fmt.Errorf("insufficient data: repository has only %d closed issues (minimum 10 required)", count)
	}

	return nil
}

// generateStatistics generates overall statistics
func (c *Calculator) generateStatistics(ctx context.Context, repoID int64) (*Statistics, error) {
	stats, err := c.db.QueryCLQSStatistics(ctx, repoID)
	if err != nil {
		return nil, err
	}

	return &Statistics{
		TotalClosedIssues: stats.TotalClosed,
		EligibleIssues:    stats.Eligible,
		LinkedIssues:      stats.Linked,
		TotalLinks:        stats.TotalLinks,
		AvgConfidence:     stats.AvgConfidence,
	}, nil
}

// storeResults stores CLQS results to the database
func (c *Calculator) storeResults(ctx context.Context, repoID int64, report *CLQSReport) error {
	score := &database.CLQSScore{
		RepoID:               repoID,
		CLQS:                 report.Overall.CLQS,
		Grade:                report.Overall.Grade,
		Rank:                 report.Overall.Rank,
		ConfidenceMultiplier: report.Overall.ConfidenceMultiplier,

		LinkCoverage:      report.Components[0].Score,
		ConfidenceQuality: report.Components[1].Score,
		EvidenceDiversity: report.Components[2].Score,
		TemporalPrecision: report.Components[3].Score,
		SemanticStrength:  report.Components[4].Score,

		LinkCoverageContribution:      report.Components[0].Contribution,
		ConfidenceQualityContribution: report.Components[1].Contribution,
		EvidenceDiversityContribution: report.Components[2].Contribution,
		TemporalPrecisionContribution: report.Components[3].Contribution,
		SemanticStrengthContribution:  report.Components[4].Contribution,

		TotalClosedIssues: report.Statistics.TotalClosedIssues,
		EligibleIssues:    report.Statistics.EligibleIssues,
		LinkedIssues:      report.Statistics.LinkedIssues,
		TotalLinks:        report.Statistics.TotalLinks,
		AvgConfidence:     report.Statistics.AvgConfidence,

		ComputedAt: time.Now(),
	}

	return c.db.StoreCLQSScore(ctx, score, report.Components, report.Recommendations, report.LabelingOpportunities)
}

// assignGradeAndRank assigns a letter grade and quality rank based on CLQS score
func assignGradeAndRank(clqs float64) (string, string) {
	switch {
	case clqs >= 97:
		return "A+", "World-Class"
	case clqs >= 93:
		return "A", "World-Class"
	case clqs >= 90:
		return "A-", "World-Class"
	case clqs >= 87:
		return "B+", "High Quality"
	case clqs >= 83:
		return "B", "High Quality"
	case clqs >= 80:
		return "B-", "High Quality"
	case clqs >= 77:
		return "C+", "Moderate Quality"
	case clqs >= 73:
		return "C", "Moderate Quality"
	case clqs >= 70:
		return "C-", "Moderate Quality"
	case clqs >= 67:
		return "D+", "Below Average"
	case clqs >= 63:
		return "D", "Below Average"
	case clqs >= 60:
		return "D-", "Below Average"
	default:
		return "F", "Poor Quality"
	}
}

// calculateConfidenceMultiplier calculates the confidence multiplier for risk scoring
func calculateConfidenceMultiplier(clqs float64) float64 {
	switch {
	case clqs >= 90:
		return 1.00
	case clqs >= 80:
		return 0.90
	case clqs >= 70:
		return 0.75
	case clqs >= 60:
		return 0.50
	default:
		return 0.25
	}
}
