package backtest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/graph"
)

// Backtester validates graph construction and linking against ground truth
type Backtester struct {
	stagingDB   *database.StagingClient
	neo4jDB     *graph.Client
	groundTruth *GroundTruth
}

// NewBacktester creates a new backtester
func NewBacktester(stagingDB *database.StagingClient, neo4jDB *graph.Client, groundTruthPath string) (*Backtester, error) {
	gt, err := LoadGroundTruth(groundTruthPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load ground truth: %w", err)
	}

	return &Backtester{
		stagingDB:   stagingDB,
		neo4jDB:     neo4jDB,
		groundTruth: gt,
	}, nil
}

// GroundTruth represents validated test cases from ground truth JSON
type GroundTruth struct {
	Repository          string                 `json:"repository"`
	GithubURL           string                 `json:"github_url"`
	ValidationDate      string                 `json:"validation_date"`
	Validator           string                 `json:"validator"`
	TotalCases          int                    `json:"total_cases"`
	PatternDistribution map[string]int         `json:"pattern_distribution"`
	TestCases           []GroundTruthTestCase  `json:"test_cases"`
	ValidationMetrics   ValidationMetrics      `json:"validation_metrics"`
	Notes               string                 `json:"notes"`
}

// GroundTruthTestCase represents a single test case
type GroundTruthTestCase struct {
	IssueNumber    int                    `json:"issue_number"`
	Title          string                 `json:"title"`
	IssueURL       string                 `json:"issue_url"`
	ExpectedLinks  ExpectedLinks          `json:"expected_links"`
	LinkingPatterns []string              `json:"linking_patterns"`
	PrimaryEvidence map[string]interface{} `json:"primary_evidence"`
	LinkQuality    string                 `json:"link_quality"`
	Difficulty     string                 `json:"difficulty"`
	Notes          string                 `json:"notes"`
	ExpectedConfidence float64             `json:"expected_confidence"`
	ShouldDetect   bool                   `json:"should_detect"`
	ExpectedMiss   bool                   `json:"expected_miss,omitempty"`
	GithubVerification GithubVerification  `json:"github_verification"`
}

// ExpectedLinks defines what links should be found
type ExpectedLinks struct {
	FixedByCommits   []string `json:"fixed_by_commits"`
	AssociatedPRs    []int    `json:"associated_prs"`
	AssociatedIssues []int    `json:"associated_issues"`
}

// GithubVerification tracks manual verification
type GithubVerification struct {
	IssueState       string `json:"issue_state"`
	PRState          *string `json:"pr_state"`
	VerifiedManually bool   `json:"verified_manually"`
}

// ValidationMetrics defines expected test outcomes
type ValidationMetrics struct {
	ExpectedTruePositives  int     `json:"expected_true_positives"`
	ExpectedTrueNegatives  int     `json:"expected_true_negatives"`
	ExpectedFalseNegatives int     `json:"expected_false_negatives"`
	ExpectedFalsePositives int     `json:"expected_false_positives"`
	TargetPrecision        float64 `json:"target_precision"`
	TargetRecall           float64 `json:"target_recall"`
	TargetF1               float64 `json:"target_f1"`
}

// BacktestResult represents the results of backtesting
type BacktestResult struct {
	TestCase        GroundTruthTestCase `json:"test_case"`
	Actual          ActualResult        `json:"actual"`
	Status          string              `json:"status"` // "PASS", "FAIL", "EXPECTED_MISS"
	ConfidenceDelta float64             `json:"confidence_delta"`
	Errors          []string            `json:"errors,omitempty"`
}

// ActualResult represents what the system actually found
type ActualResult struct {
	Found           bool     `json:"found"`
	PRLinks         []int    `json:"pr_links"`
	CommitLinks     []string `json:"commit_links"`
	Confidence      float64  `json:"confidence"`
	Evidence        []string `json:"evidence"`
	DetectionMethod string   `json:"detection_method"`
}

// BacktestReport contains the complete backtesting analysis
type BacktestReport struct {
	Repository       string            `json:"repository"`
	TestDate         string            `json:"test_date"`
	TotalCases       int               `json:"total_cases"`
	Results          []BacktestResult  `json:"results"`
	Metrics          PerformanceMetrics `json:"metrics"`
	PatternAnalysis  PatternPerformance `json:"pattern_analysis"`
	Summary          string            `json:"summary"`
}

// PerformanceMetrics contains precision, recall, F1
type PerformanceMetrics struct {
	TruePositives  int     `json:"true_positives"`
	TrueNegatives  int     `json:"true_negatives"`
	FalsePositives int     `json:"false_positives"`
	FalseNegatives int     `json:"false_negatives"`
	Precision      float64 `json:"precision"`
	Recall         float64 `json:"recall"`
	F1Score        float64 `json:"f1_score"`
	Accuracy       float64 `json:"accuracy"`
}

// PatternPerformance tracks performance by pattern type
type PatternPerformance struct {
	Temporal PatternMetrics `json:"temporal"`
	Semantic PatternMetrics `json:"semantic"`
	Explicit PatternMetrics `json:"explicit"`
	Comment  PatternMetrics `json:"comment"`
}

// PatternMetrics tracks metrics for a specific pattern
type PatternMetrics struct {
	TotalCases     int     `json:"total_cases"`
	Detected       int     `json:"detected"`
	Missed         int     `json:"missed"`
	DetectionRate  float64 `json:"detection_rate"`
	AvgConfidence  float64 `json:"avg_confidence"`
}

// LoadGroundTruth loads ground truth data from JSON file
func LoadGroundTruth(path string) (*GroundTruth, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read ground truth file: %w", err)
	}

	var gt GroundTruth
	if err := json.Unmarshal(data, &gt); err != nil {
		return nil, fmt.Errorf("failed to parse ground truth JSON: %w", err)
	}

	return &gt, nil
}

// RunBacktest executes the backtesting suite
func (bt *Backtester) RunBacktest(ctx context.Context, repoID int64) (*BacktestReport, error) {
	log.Printf("🧪 Starting backtesting against ground truth...")
	log.Printf("  Repository: %s", bt.groundTruth.Repository)
	log.Printf("  Test Cases: %d", bt.groundTruth.TotalCases)

	report := &BacktestReport{
		Repository: bt.groundTruth.Repository,
		TestDate:   time.Now().Format(time.RFC3339),
		TotalCases: bt.groundTruth.TotalCases,
		Results:    []BacktestResult{},
		Metrics: PerformanceMetrics{},
		PatternAnalysis: PatternPerformance{},
	}

	// Test each case
	for i, testCase := range bt.groundTruth.TestCases {
		log.Printf("\n  [%d/%d] Testing Issue #%d: %s",
			i+1, bt.groundTruth.TotalCases, testCase.IssueNumber, testCase.Title)

		result, err := bt.testCase(ctx, repoID, testCase)
		if err != nil {
			log.Printf("    ❌ Error: %v", err)
			result.Errors = append(result.Errors, err.Error())
		}

		report.Results = append(report.Results, result)

		// Log result
		switch result.Status {
		case "PASS":
			log.Printf("    ✅ PASS (confidence: %.2f, delta: %.2f)",
				result.Actual.Confidence, result.ConfidenceDelta)
		case "EXPECTED_MISS":
			log.Printf("    ⏭️  EXPECTED_MISS (internal fix)")
		case "FAIL":
			log.Printf("    ❌ FAIL")
			for _, err := range result.Errors {
				log.Printf("       - %s", err)
			}
		}
	}

	// Calculate metrics
	report.Metrics = bt.calculateMetrics(report.Results)
	report.PatternAnalysis = bt.analyzePatterns(report.Results)
	report.Summary = bt.generateSummary(report)

	log.Printf("\n📊 Backtesting Complete:")
	log.Printf("  Precision: %.2f%%", report.Metrics.Precision*100)
	log.Printf("  Recall: %.2f%%", report.Metrics.Recall*100)
	log.Printf("  F1 Score: %.2f%%", report.Metrics.F1Score*100)
	log.Printf("  Accuracy: %.2f%%", report.Metrics.Accuracy*100)

	return report, nil
}

// testCase tests a single ground truth case
func (bt *Backtester) testCase(ctx context.Context, repoID int64, testCase GroundTruthTestCase) (BacktestResult, error) {
	result := BacktestResult{
		TestCase: testCase,
		Actual: ActualResult{
			Found:      false,
			PRLinks:    []int{},
			CommitLinks: []string{},
			Evidence:   []string{},
		},
		Status: "FAIL",
		Errors: []string{},
	}

	// Handle expected misses (true negatives)
	if testCase.ExpectedMiss {
		result.Status = "EXPECTED_MISS"
		return result, nil
	}

	// Query Neo4j for actual links
	query := `
		MATCH (i:Issue {number: $issue_number})
		OPTIONAL MATCH (i)-[r:FIXES_ISSUE|ASSOCIATED_WITH|MENTIONS]->(target)
		WHERE target:PR OR target:Commit
		RETURN
			labels(target) as target_type,
			CASE WHEN target:PR THEN target.number ELSE null END as pr_number,
			CASE WHEN target:Commit THEN target.sha ELSE null END as commit_sha,
			r.confidence as confidence,
			r.evidence as evidence,
			r.detection_method as detection_method
		ORDER BY r.confidence DESC
	`

	records, err := bt.neo4jDB.ExecuteQuery(ctx, query, map[string]any{
		"issue_number": testCase.IssueNumber,
	})
	if err != nil {
		return result, fmt.Errorf("query failed: %w", err)
	}

	// Process results
	maxConfidence := 0.0
	for _, record := range records {
		// Skip if no target (OPTIONAL MATCH returned null)
		if record["target_type"] == nil {
			continue
		}

		result.Actual.Found = true

		// Extract target type
		targetType := ""
		if labels, ok := record["target_type"].([]interface{}); ok && len(labels) > 0 {
			if label, ok := labels[0].(string); ok {
				targetType = label
			}
		}

		// Extract confidence
		if confidence, ok := record["confidence"].(float64); ok {
			if confidence > maxConfidence {
				maxConfidence = confidence
			}
		}

		// Extract evidence
		if evidence, ok := record["evidence"].([]interface{}); ok {
			for _, ev := range evidence {
				if evStr, ok := ev.(string); ok {
					result.Actual.Evidence = append(result.Actual.Evidence, evStr)
				}
			}
		}

		// Extract detection method
		if method, ok := record["detection_method"].(string); ok {
			result.Actual.DetectionMethod = method
		}

		// Extract PR number or Commit SHA
		if targetType == "PR" && record["pr_number"] != nil {
			if prNum, ok := record["pr_number"].(int64); ok {
				result.Actual.PRLinks = append(result.Actual.PRLinks, int(prNum))
			}
		} else if targetType == "Commit" && record["commit_sha"] != nil {
			if sha, ok := record["commit_sha"].(string); ok {
				result.Actual.CommitLinks = append(result.Actual.CommitLinks, sha)
			}
		}
	}

	result.Actual.Confidence = maxConfidence

	// Validate against expected links
	if testCase.ShouldDetect {
		if !result.Actual.Found {
			result.Status = "FAIL"
			result.Errors = append(result.Errors, "Expected link not found")
		} else if !bt.validateLinks(testCase.ExpectedLinks, result.Actual) {
			result.Status = "FAIL"
			result.Errors = append(result.Errors, "Found links don't match expected links")
		} else {
			result.Status = "PASS"
		}

		// Check confidence delta
		if testCase.ExpectedConfidence > 0 {
			result.ConfidenceDelta = result.Actual.Confidence - testCase.ExpectedConfidence
		}
	} else {
		// Should NOT detect (true negative)
		if result.Actual.Found {
			result.Status = "FAIL"
			result.Errors = append(result.Errors, "False positive: found link when none expected")
		} else {
			result.Status = "PASS"
		}
	}

	return result, nil
}

// validateLinks checks if actual links match expected links
func (bt *Backtester) validateLinks(expected ExpectedLinks, actual ActualResult) bool {
	// Check PR links
	if len(expected.AssociatedPRs) > 0 {
		found := false
		for _, expectedPR := range expected.AssociatedPRs {
			for _, actualPR := range actual.PRLinks {
				if expectedPR == actualPR {
					found = true
					break
				}
			}
		}
		if !found {
			return false
		}
	}

	// Check commit links
	if len(expected.FixedByCommits) > 0 {
		found := false
		for _, expectedCommit := range expected.FixedByCommits {
			for _, actualCommit := range actual.CommitLinks {
				// Match by prefix (short SHA vs full SHA)
				if len(expectedCommit) >= 7 && len(actualCommit) >= 7 {
					if expectedCommit[:7] == actualCommit[:7] {
						found = true
						break
					}
				}
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// calculateMetrics computes precision, recall, F1
func (bt *Backtester) calculateMetrics(results []BacktestResult) PerformanceMetrics {
	metrics := PerformanceMetrics{}

	for _, result := range results {
		switch {
		case result.Status == "PASS" && result.TestCase.ShouldDetect:
			metrics.TruePositives++
		case result.Status == "PASS" && !result.TestCase.ShouldDetect:
			metrics.TrueNegatives++
		case result.Status == "FAIL" && result.TestCase.ShouldDetect:
			metrics.FalseNegatives++
		case result.Status == "FAIL" && !result.TestCase.ShouldDetect:
			metrics.FalsePositives++
		case result.Status == "EXPECTED_MISS":
			metrics.FalseNegatives++ // Count as FN for metrics
		}
	}

	total := float64(metrics.TruePositives + metrics.TrueNegatives +
		metrics.FalsePositives + metrics.FalseNegatives)

	if total > 0 {
		metrics.Accuracy = float64(metrics.TruePositives+metrics.TrueNegatives) / total
	}

	if metrics.TruePositives+metrics.FalsePositives > 0 {
		metrics.Precision = float64(metrics.TruePositives) /
			float64(metrics.TruePositives+metrics.FalsePositives)
	}

	if metrics.TruePositives+metrics.FalseNegatives > 0 {
		metrics.Recall = float64(metrics.TruePositives) /
			float64(metrics.TruePositives+metrics.FalseNegatives)
	}

	if metrics.Precision+metrics.Recall > 0 {
		metrics.F1Score = 2 * (metrics.Precision * metrics.Recall) /
			(metrics.Precision + metrics.Recall)
	}

	return metrics
}

// analyzePatterns breaks down performance by pattern type
func (bt *Backtester) analyzePatterns(results []BacktestResult) PatternPerformance {
	analysis := PatternPerformance{}

	// Track pattern-specific metrics
	temporalCases := []BacktestResult{}
	semanticCases := []BacktestResult{}
	explicitCases := []BacktestResult{}
	commentCases := []BacktestResult{}

	for _, result := range results {
		for _, pattern := range result.TestCase.LinkingPatterns {
			switch pattern {
			case "temporal":
				temporalCases = append(temporalCases, result)
			case "semantic":
				semanticCases = append(semanticCases, result)
			case "explicit":
				explicitCases = append(explicitCases, result)
			case "comment":
				commentCases = append(commentCases, result)
			}
		}
	}

	analysis.Temporal = bt.calculatePatternMetrics(temporalCases)
	analysis.Semantic = bt.calculatePatternMetrics(semanticCases)
	analysis.Explicit = bt.calculatePatternMetrics(explicitCases)
	analysis.Comment = bt.calculatePatternMetrics(commentCases)

	return analysis
}

// calculatePatternMetrics computes metrics for a specific pattern
func (bt *Backtester) calculatePatternMetrics(cases []BacktestResult) PatternMetrics {
	metrics := PatternMetrics{
		TotalCases: len(cases),
	}

	if len(cases) == 0 {
		return metrics
	}

	totalConfidence := 0.0
	for _, result := range cases {
		if result.Status == "PASS" || result.Actual.Found {
			metrics.Detected++
			totalConfidence += result.Actual.Confidence
		} else {
			metrics.Missed++
		}
	}

	metrics.DetectionRate = float64(metrics.Detected) / float64(metrics.TotalCases)
	if metrics.Detected > 0 {
		metrics.AvgConfidence = totalConfidence / float64(metrics.Detected)
	}

	return metrics
}

// generateSummary creates a human-readable summary
func (bt *Backtester) generateSummary(report *BacktestReport) string {
	summary := fmt.Sprintf("Backtesting Results for %s\n", report.Repository)
	summary += fmt.Sprintf("Test Date: %s\n", report.TestDate)
	summary += fmt.Sprintf("Total Cases: %d\n\n", report.TotalCases)

	summary += "Overall Performance:\n"
	summary += fmt.Sprintf("  Precision: %.2f%%\n", report.Metrics.Precision*100)
	summary += fmt.Sprintf("  Recall: %.2f%%\n", report.Metrics.Recall*100)
	summary += fmt.Sprintf("  F1 Score: %.2f%%\n", report.Metrics.F1Score*100)
	summary += fmt.Sprintf("  Accuracy: %.2f%%\n\n", report.Metrics.Accuracy*100)

	summary += "Pattern Analysis:\n"
	summary += fmt.Sprintf("  Temporal: %d/%d detected (%.1f%%), avg confidence: %.2f\n",
		report.PatternAnalysis.Temporal.Detected,
		report.PatternAnalysis.Temporal.TotalCases,
		report.PatternAnalysis.Temporal.DetectionRate*100,
		report.PatternAnalysis.Temporal.AvgConfidence)
	summary += fmt.Sprintf("  Semantic: %d/%d detected (%.1f%%), avg confidence: %.2f\n",
		report.PatternAnalysis.Semantic.Detected,
		report.PatternAnalysis.Semantic.TotalCases,
		report.PatternAnalysis.Semantic.DetectionRate*100,
		report.PatternAnalysis.Semantic.AvgConfidence)
	summary += fmt.Sprintf("  Explicit: %d/%d detected (%.1f%%), avg confidence: %.2f\n",
		report.PatternAnalysis.Explicit.Detected,
		report.PatternAnalysis.Explicit.TotalCases,
		report.PatternAnalysis.Explicit.DetectionRate*100,
		report.PatternAnalysis.Explicit.AvgConfidence)
	summary += fmt.Sprintf("  Comment: %d/%d detected (%.1f%%), avg confidence: %.2f\n",
		report.PatternAnalysis.Comment.Detected,
		report.PatternAnalysis.Comment.TotalCases,
		report.PatternAnalysis.Comment.DetectionRate*100,
		report.PatternAnalysis.Comment.AvgConfidence)

	return summary
}

// SaveReport saves the backtesting report to a JSON file
func (bt *Backtester) SaveReport(report *BacktestReport, outputPath string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	log.Printf("📄 Report saved to: %s", outputPath)
	return nil
}
