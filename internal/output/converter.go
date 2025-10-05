package output

import (
	"time"

	"github.com/coderisk/coderisk-go/internal/metrics"
	"github.com/coderisk/coderisk-go/internal/models"
)

// ConvertPhase1ToRiskResult converts Phase1Result to RiskResult for formatting
func ConvertPhase1ToRiskResult(phase1 *metrics.Phase1Result) *models.RiskResult {
	result := &models.RiskResult{
		Branch:       "main", // TODO: Get from git
		FilesChanged: 1,
		RiskLevel:    string(phase1.OverallRisk),
		RiskScore:    calculateRiskScore(phase1),
		Confidence:   0.8, // Phase 1 has fixed confidence
		StartTime:    time.Now().Add(-time.Duration(phase1.DurationMS) * time.Millisecond),
		EndTime:      time.Now(),
		Duration:     time.Duration(phase1.DurationMS) * time.Millisecond,
		CacheHit:     false,
	}

	// Convert to FileRisk
	result.Files = []models.FileRisk{
		{
			Path:         phase1.FilePath,
			Language:     "unknown", // TODO: Detect from file
			LinesChanged: 0,         // TODO: Get from git diff
			RiskScore:    calculateRiskScore(phase1),
			Metrics:      convertMetrics(phase1),
		},
	}

	// Convert to issues
	result.Issues = convertToIssues(phase1)

	// Generate recommendations
	result.Recommendations = generateRecommendations(phase1)

	return result
}

func calculateRiskScore(phase1 *metrics.Phase1Result) float64 {
	switch phase1.OverallRisk {
	case metrics.RiskLevelHigh:
		return 8.0
	case metrics.RiskLevelMedium:
		return 5.0
	case metrics.RiskLevelLow:
		return 2.0
	default:
		return 0.0
	}
}

func convertMetrics(phase1 *metrics.Phase1Result) map[string]models.Metric {
	metrics := make(map[string]models.Metric)

	if phase1.Coupling != nil {
		threshold := 10.0
		metrics["coupling"] = models.Metric{
			Name:      "Structural Coupling",
			Value:     float64(phase1.Coupling.Count),
			Threshold: &threshold,
		}
	}

	if phase1.CoChange != nil {
		threshold := 0.7
		metrics["co_change"] = models.Metric{
			Name:      "Temporal Co-Change",
			Value:     phase1.CoChange.MaxFrequency,
			Threshold: &threshold,
		}
	}

	if phase1.TestRatio != nil {
		threshold := 0.3
		metrics["test_coverage"] = models.Metric{
			Name:      "Test Coverage",
			Value:     phase1.TestRatio.Ratio,
			Threshold: &threshold,
		}
	}

	return metrics
}

func convertToIssues(phase1 *metrics.Phase1Result) []models.RiskIssue {
	var issues []models.RiskIssue

	if phase1.Coupling != nil && phase1.Coupling.ShouldEscalate() {
		issues = append(issues, models.RiskIssue{
			ID:       "COUPLING_HIGH",
			Severity: string(phase1.Coupling.RiskLevel),
			Category: "coupling",
			File:     phase1.FilePath,
			Message:  phase1.Coupling.FormatEvidence(),
		})
	}

	if phase1.CoChange != nil && phase1.CoChange.ShouldEscalate() {
		issues = append(issues, models.RiskIssue{
			ID:       "COCHANGE_HIGH",
			Severity: string(phase1.CoChange.RiskLevel),
			Category: "temporal",
			File:     phase1.FilePath,
			Message:  phase1.CoChange.FormatEvidence(),
		})
	}

	if phase1.TestRatio != nil && phase1.TestRatio.ShouldEscalate() {
		issues = append(issues, models.RiskIssue{
			ID:       "TEST_COVERAGE_LOW",
			Severity: string(phase1.TestRatio.RiskLevel),
			Category: "quality",
			File:     phase1.FilePath,
			Message:  phase1.TestRatio.FormatEvidence(),
		})
	}

	return issues
}

func generateRecommendations(phase1 *metrics.Phase1Result) []string {
	var recs []string

	if phase1.Coupling != nil && phase1.Coupling.ShouldEscalate() {
		recs = append(recs, "Review and reduce structural coupling")
	}

	if phase1.CoChange != nil && phase1.CoChange.ShouldEscalate() {
		recs = append(recs, "Investigate temporal coupling patterns")
	}

	if phase1.TestRatio != nil && phase1.TestRatio.ShouldEscalate() {
		recs = append(recs, "Add test coverage for this file")
	}

	return recs
}
