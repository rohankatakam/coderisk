package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/linking"
	"github.com/spf13/cobra"
)

var (
	repoFullName     string
	groundTruthPath  string
	outputReportPath string
)

var rootCmd = &cobra.Command{
	Use:   "test-linker",
	Short: "Test issue-PR linker against ground truth dataset",
	Long: `test-linker validates the issue-PR linking results against a manually labeled ground truth dataset.

It compares the links found by the linker with expected links and calculates:
  - Precision: True positives / (True positives + False positives)
  - Recall: True positives / (True positives + False negatives)
  - F1 Score: Harmonic mean of precision and recall

Ground truth format: JSON file with expected links for each issue.

Example:
  test-linker --repo omnara-ai/omnara \
              --ground-truth test_data/omnara_ground_truth_expanded.json \
              --output test_report.txt`,
	RunE: runTestLinker,
}

func init() {
	rootCmd.Flags().StringVar(&repoFullName, "repo", "", "Repository full name (owner/repo)")
	rootCmd.Flags().StringVar(&groundTruthPath, "ground-truth", "", "Path to ground truth JSON file")
	rootCmd.Flags().StringVar(&outputReportPath, "output", "test_report.txt", "Path to output report file")

	rootCmd.MarkFlagRequired("repo")
	rootCmd.MarkFlagRequired("ground-truth")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// GroundTruthTestCase represents a test case from ground truth
type GroundTruthTestCase struct {
	IssueNumber    int      `json:"issue_number"`
	Title          string   `json:"title"`
	ExpectedLinks  struct {
		AssociatedPRs []int `json:"associated_prs"`
	} `json:"expected_links"`
	ShouldDetect       bool    `json:"should_detect"`
	ExpectedConfidence float64 `json:"expected_confidence"`
	LinkQuality        string  `json:"link_quality"`
	Difficulty         string  `json:"difficulty"`
	Notes              string  `json:"notes"`
}

// GroundTruthDataset represents the complete ground truth dataset
type GroundTruthDataset struct {
	Repository      string                `json:"repository"`
	TotalCases      int                   `json:"total_cases"`
	TestCases       []GroundTruthTestCase `json:"test_cases"`
	ValidationMetrics struct {
		ExpectedTruePositives  int     `json:"expected_true_positives"`
		ExpectedTrueNegatives  int     `json:"expected_true_negatives"`
		ExpectedFalseNegatives int     `json:"expected_false_negatives"`
		TargetPrecision        float64 `json:"target_precision"`
		TargetRecall           float64 `json:"target_recall"`
		TargetF1               float64 `json:"target_f1"`
	} `json:"validation_metrics"`
}

// TestResult represents the result of testing one issue
type TestResult struct {
	IssueNumber      int
	Title            string
	ExpectedPRs      []int
	ActualPRs        []int
	ShouldDetect     bool
	Difficulty       string
	TruePositives    []int
	FalsePositives   []int
	FalseNegatives   []int
	IsCorrect        bool
	ConfidenceScores map[int]float64
	LinkQualities    map[int]string
}

func runTestLinker(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	log.Printf("========================================")
	log.Printf("Issue-PR Linker Test Suite")
	log.Printf("========================================")
	log.Printf("Repository: %s", repoFullName)
	log.Printf("Ground Truth: %s", groundTruthPath)
	log.Printf("Output Report: %s", outputReportPath)
	log.Printf("")

	// Load ground truth
	groundTruth, err := loadGroundTruth(groundTruthPath)
	if err != nil {
		return fmt.Errorf("failed to load ground truth: %w", err)
	}

	log.Printf("Loaded %d test cases from ground truth", len(groundTruth.TestCases))
	log.Printf("")

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Connect to PostgreSQL
	log.Printf("Connecting to PostgreSQL...")
	stagingDB, err := database.NewStagingClient(
		ctx,
		cfg.Storage.PostgresHost,
		cfg.Storage.PostgresPort,
		cfg.Storage.PostgresDB,
		cfg.Storage.PostgresUser,
		cfg.Storage.PostgresPassword,
	)
	if err != nil {
		return fmt.Errorf("PostgreSQL connection failed: %w", err)
	}
	defer stagingDB.Close()
	log.Printf("  ✓ Connected to PostgreSQL")

	// Get repository ID
	repoID, err := stagingDB.GetRepositoryID(ctx, repoFullName)
	if err != nil {
		return fmt.Errorf("repository not found: %s: %w", repoFullName, err)
	}
	log.Printf("  ✓ Found repository (ID: %d)", repoID)
	log.Printf("")

	// Get actual links from database
	log.Printf("Fetching actual links from database...")
	links, err := stagingDB.GetIssuePRLinks(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to get links: %w", err)
	}
	log.Printf("  ✓ Loaded %d links", len(links))
	log.Printf("")

	// Build map: issue_number -> []pr_number
	actualLinksMap := make(map[int][]int)
	confidenceMap := make(map[int]map[int]float64)
	qualityMap := make(map[int]map[int]string)

	for _, link := range links {
		actualLinksMap[link.IssueNumber] = append(actualLinksMap[link.IssueNumber], link.PRNumber)

		if confidenceMap[link.IssueNumber] == nil {
			confidenceMap[link.IssueNumber] = make(map[int]float64)
			qualityMap[link.IssueNumber] = make(map[int]string)
		}
		confidenceMap[link.IssueNumber][link.PRNumber] = link.FinalConfidence
		qualityMap[link.IssueNumber][link.PRNumber] = string(link.LinkQuality)
	}

	// Run tests
	log.Printf("Running tests...")
	log.Printf("─────────────────────────────────────")
	log.Printf("")

	results := make([]TestResult, 0, len(groundTruth.TestCases))

	for _, testCase := range groundTruth.TestCases {
		result := runTestCase(testCase, actualLinksMap, confidenceMap, qualityMap)
		results = append(results, result)

		// Print result
		status := "✓ PASS"
		if !result.IsCorrect {
			status = "✗ FAIL"
		}

		log.Printf("%s  Issue #%d: %s", status, result.IssueNumber, truncateText(result.Title, 60))
		log.Printf("      Expected PRs: %v", result.ExpectedPRs)
		log.Printf("      Actual PRs:   %v", result.ActualPRs)

		if len(result.FalsePositives) > 0 {
			log.Printf("      ⚠️  False Positives: %v", result.FalsePositives)
		}
		if len(result.FalseNegatives) > 0 {
			log.Printf("      ⚠️  False Negatives: %v", result.FalseNegatives)
		}

		log.Printf("")
	}

	// Calculate metrics
	metrics := calculateMetrics(results)

	// Print summary
	printSummary(results, metrics, groundTruth)

	// Write detailed report
	if err := writeReport(outputReportPath, results, metrics, groundTruth); err != nil {
		log.Printf("⚠️  Failed to write report: %v", err)
	} else {
		log.Printf("✓ Detailed report written to: %s", outputReportPath)
	}

	// Return error if targets not met
	if !meetsTargets(metrics, groundTruth) {
		return fmt.Errorf("test suite failed - targets not met")
	}

	log.Printf("")
	log.Printf("✅ All targets met!")
	return nil
}

func loadGroundTruth(path string) (*GroundTruthDataset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var dataset GroundTruthDataset
	if err := json.Unmarshal(data, &dataset); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &dataset, nil
}

func runTestCase(testCase GroundTruthTestCase, actualLinksMap, confidenceMap, qualityMap map[int]map[int]string) TestResult {
	expectedPRs := testCase.ExpectedLinks.AssociatedPRs
	actualPRs := actualLinksMap[testCase.IssueNumber]

	result := TestResult{
		IssueNumber:      testCase.IssueNumber,
		Title:            testCase.Title,
		ExpectedPRs:      expectedPRs,
		ActualPRs:        actualPRs,
		ShouldDetect:     testCase.ShouldDetect,
		Difficulty:       testCase.Difficulty,
		ConfidenceScores: confidenceMap[testCase.IssueNumber],
		LinkQualities:    qualityMap[testCase.IssueNumber],
	}

	// Calculate TPs, FPs, FNs
	expectedSet := makeSet(expectedPRs)
	actualSet := makeSet(actualPRs)

	for pr := range expectedSet {
		if actualSet[pr] {
			result.TruePositives = append(result.TruePositives, pr)
		} else {
			result.FalseNegatives = append(result.FalseNegatives, pr)
		}
	}

	for pr := range actualSet {
		if !expectedSet[pr] {
			result.FalsePositives = append(result.FalsePositives, pr)
		}
	}

	// Determine if correct
	if testCase.ShouldDetect {
		// Should have found links - check if we got all expected PRs
		result.IsCorrect = len(result.FalseNegatives) == 0 && len(result.FalsePositives) == 0
	} else {
		// Should not have found links
		result.IsCorrect = len(actualPRs) == 0
	}

	return result
}

func calculateMetrics(results []TestResult) map[string]float64 {
	var totalTP, totalFP, totalFN int

	for _, result := range results {
		totalTP += len(result.TruePositives)
		totalFP += len(result.FalsePositives)
		totalFN += len(result.FalseNegatives)
	}

	precision := 0.0
	if totalTP+totalFP > 0 {
		precision = float64(totalTP) / float64(totalTP+totalFP)
	}

	recall := 0.0
	if totalTP+totalFN > 0 {
		recall = float64(totalTP) / float64(totalTP+totalFN)
	}

	f1 := 0.0
	if precision+recall > 0 {
		f1 = 2 * (precision * recall) / (precision + recall)
	}

	return map[string]float64{
		"precision": precision,
		"recall":    recall,
		"f1":        f1,
		"tp":        float64(totalTP),
		"fp":        float64(totalFP),
		"fn":        float64(totalFN),
	}
}

func printSummary(results []TestResult, metrics map[string]float64, groundTruth *GroundTruthDataset) {
	log.Printf("========================================")
	log.Printf("Test Results Summary")
	log.Printf("========================================")
	log.Printf("")

	passed := 0
	for _, result := range results {
		if result.IsCorrect {
			passed++
		}
	}

	log.Printf("Tests Passed: %d/%d (%.1f%%)", passed, len(results), 100.0*float64(passed)/float64(len(results)))
	log.Printf("")

	log.Printf("Confusion Matrix:")
	log.Printf("  True Positives:  %.0f", metrics["tp"])
	log.Printf("  False Positives: %.0f", metrics["fp"])
	log.Printf("  False Negatives: %.0f", metrics["fn"])
	log.Printf("")

	log.Printf("Metrics:")
	log.Printf("  Precision: %.3f (target: %.3f)", metrics["precision"], groundTruth.ValidationMetrics.TargetPrecision)
	log.Printf("  Recall:    %.3f (target: %.3f)", metrics["recall"], groundTruth.ValidationMetrics.TargetRecall)
	log.Printf("  F1 Score:  %.3f (target: %.3f)", metrics["f1"], groundTruth.ValidationMetrics.TargetF1)
	log.Printf("")
}

func writeReport(path string, results []TestResult, metrics map[string]float64, groundTruth *GroundTruthDataset) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "Issue-PR Linker Test Report\n")
	fmt.Fprintf(f, "Repository: %s\n", groundTruth.Repository)
	fmt.Fprintf(f, "Total Test Cases: %d\n\n", len(results))

	// Sort results by issue number
	sort.Slice(results, func(i, j int) bool {
		return results[i].IssueNumber < results[j].IssueNumber
	})

	// Write detailed results
	fmt.Fprintf(f, "Detailed Results:\n")
	fmt.Fprintf(f, "=================\n\n")

	for _, result := range results {
		status := "PASS"
		if !result.IsCorrect {
			status = "FAIL"
		}

		fmt.Fprintf(f, "[%s] Issue #%d: %s\n", status, result.IssueNumber, result.Title)
		fmt.Fprintf(f, "  Difficulty: %s\n", result.Difficulty)
		fmt.Fprintf(f, "  Expected PRs: %v\n", result.ExpectedPRs)
		fmt.Fprintf(f, "  Actual PRs:   %v\n", result.ActualPRs)

		if len(result.TruePositives) > 0 {
			fmt.Fprintf(f, "  True Positives: %v\n", result.TruePositives)
			for _, pr := range result.TruePositives {
				fmt.Fprintf(f, "    PR #%d: confidence=%.2f, quality=%s\n",
					pr, result.ConfidenceScores[pr], result.LinkQualities[pr])
			}
		}

		if len(result.FalsePositives) > 0 {
			fmt.Fprintf(f, "  ⚠️  False Positives: %v\n", result.FalsePositives)
		}

		if len(result.FalseNegatives) > 0 {
			fmt.Fprintf(f, "  ⚠️  False Negatives: %v\n", result.FalseNegatives)
		}

		fmt.Fprintf(f, "\n")
	}

	// Write metrics summary
	fmt.Fprintf(f, "\nMetrics Summary:\n")
	fmt.Fprintf(f, "================\n\n")
	fmt.Fprintf(f, "Precision: %.3f (target: %.3f)\n", metrics["precision"], groundTruth.ValidationMetrics.TargetPrecision)
	fmt.Fprintf(f, "Recall:    %.3f (target: %.3f)\n", metrics["recall"], groundTruth.ValidationMetrics.TargetRecall)
	fmt.Fprintf(f, "F1 Score:  %.3f (target: %.3f)\n", metrics["f1"], groundTruth.ValidationMetrics.TargetF1)
	fmt.Fprintf(f, "\nTrue Positives:  %.0f\n", metrics["tp"])
	fmt.Fprintf(f, "False Positives: %.0f\n", metrics["fp"])
	fmt.Fprintf(f, "False Negatives: %.0f\n", metrics["fn"])

	return nil
}

func meetsTargets(metrics map[string]float64, groundTruth *GroundTruthDataset) bool {
	return metrics["precision"] >= groundTruth.ValidationMetrics.TargetPrecision &&
		metrics["recall"] >= groundTruth.ValidationMetrics.TargetRecall &&
		metrics["f1"] >= groundTruth.ValidationMetrics.TargetF1
}

func makeSet(items []int) map[int]bool {
	set := make(map[int]bool)
	for _, item := range items {
		set[item] = true
	}
	return set
}

func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
