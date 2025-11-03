package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// GroundTruth represents the expected test results
type GroundTruth struct {
	Repository          string                  `json:"repository"`
	GitHubURL           string                  `json:"github_url"`
	ValidationDate      string                  `json:"validation_date"`
	Validator           string                  `json:"validator"`
	TotalCases          int                     `json:"total_cases"`
	PatternDistribution map[string]int          `json:"pattern_distribution"`
	TestCases           []GroundTruthTestCase   `json:"test_cases"`
	ValidationMetrics   ValidationMetrics       `json:"validation_metrics"`
	Notes               string                  `json:"notes"`
}

type GroundTruthTestCase struct {
	IssueNumber       int           `json:"issue_number"`
	Title             string        `json:"title"`
	IssueURL          string        `json:"issue_url"`
	ExpectedLinks     ExpectedLinks `json:"expected_links"`
	LinkingPatterns   []string      `json:"linking_patterns"`
	PrimaryEvidence   interface{}   `json:"primary_evidence"`
	LinkQuality       string        `json:"link_quality"`
	Difficulty        string        `json:"difficulty"`
	Notes             string        `json:"notes"`
	ExpectedConfidence *float64     `json:"expected_confidence"`
	ShouldDetect      bool          `json:"should_detect"`
	ExpectedMiss      bool          `json:"expected_miss"`
}

type ExpectedLinks struct {
	FixedByCommits   []string `json:"fixed_by_commits"`
	AssociatedPRs    []int    `json:"associated_prs"`
	AssociatedIssues []int    `json:"associated_issues"`
}

type ValidationMetrics struct {
	ExpectedTruePositives  int     `json:"expected_true_positives"`
	ExpectedTrueNegatives  int     `json:"expected_true_negatives"`
	ExpectedFalseNegatives int     `json:"expected_false_negatives"`
	ExpectedFalsePositives int     `json:"expected_false_positives"`
	TargetPrecision        float64 `json:"target_precision"`
	TargetRecall           float64 `json:"target_recall"`
	TargetF1               float64 `json:"target_f1"`
}

// Layer validation results
type Layer1Results struct {
	FilesCreated     int       `json:"files_created"`
	DependenciesFound int      `json:"dependencies_found"`
	TreesitterErrors int       `json:"treesitter_errors"`
	Status           string    `json:"status"`
	Duration         time.Duration `json:"duration"`
}

type Layer2Results struct {
	CommitsCreated    int       `json:"commits_created"`
	PRsCreated        int       `json:"prs_created"`
	DevelopersCreated int       `json:"developers_created"`
	ModifiedEdges     int       `json:"modified_edges"`
	Status            string    `json:"status"`
	Duration          time.Duration `json:"duration"`
}

type Layer3Results struct {
	TestCases []TestCaseResult `json:"test_cases"`
	TotalTP   int              `json:"total_tp"`
	TotalFP   int              `json:"total_fp"`
	TotalFN   int              `json:"total_fn"`
	TotalTN   int              `json:"total_tn"`
	Precision float64          `json:"precision"`
	Recall    float64          `json:"recall"`
	F1Score   float64          `json:"f1_score"`
	Status    string           `json:"status"`
	Duration  time.Duration    `json:"duration"`
}

type TestCaseResult struct {
	IssueNumber      int                `json:"issue_number"`
	Title            string             `json:"title"`
	ExpectedLinks    ExpectedLinks      `json:"expected_links"`
	ActualLinks      []ActualLink       `json:"actual_links"`
	PatternsUsed     []string           `json:"patterns_used"`
	TP               int                `json:"tp"`
	FP               int                `json:"fp"`
	FN               int                `json:"fn"`
	Status           string             `json:"status"` // "PASS", "FAIL", "PARTIAL"
	MissedPatterns   []string           `json:"missed_patterns"`
	ConfidenceScores map[string]float64 `json:"confidence_scores"`
}

type ActualLink struct {
	Type       string   `json:"type"` // "pr", "commit", "issue"
	ID         string   `json:"id"`
	Confidence float64  `json:"confidence"`
	Evidence   []string `json:"evidence"`
}

type PatternCoverage struct {
	ByPattern map[string]PatternStats `json:"by_pattern"`
}

type PatternStats struct {
	TotalCases    int     `json:"total_cases"`
	Detected      int     `json:"detected"`
	Missed        int     `json:"missed"`
	DetectionRate float64 `json:"detection_rate"`
}

type FullPipelineReport struct {
	Repository      string          `json:"repository"`
	TestDate        string          `json:"test_date"`
	Layer1          Layer1Results   `json:"layer1"`
	Layer2          Layer2Results   `json:"layer2"`
	Layer3          Layer3Results   `json:"layer3"`
	PatternCoverage PatternCoverage `json:"pattern_coverage"`
	OverallStatus   string          `json:"overall_status"`
	Recommendation  string          `json:"recommendation"`
}

func main() {
	repoFlag := flag.String("repo", "omnara", "Repository to test (omnara, supabase, stagehand)")
	skipRebuildFlag := flag.Bool("skip-rebuild", false, "Skip graph rebuild")
	outputDirFlag := flag.String("output", "test_results", "Output directory for results")
	flag.Parse()

	ctx := context.Background()
	log := logrus.New()
	log.SetLevel(logrus.InfoLevel)

	fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  FULL GRAPH CONSTRUCTION PIPELINE TEST                       ║\n")
	fmt.Printf("║  Repository: %-47s ║\n", *repoFlag)
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n\n")

	// Create output directory
	if err := os.MkdirAll(*outputDirFlag, 0755); err != nil {
		log.Error("Failed to create output directory", "error", err)
		os.Exit(1)
	}

	// PHASE 1: Rebuild graph from Postgres (if needed)
	var layer1Results Layer1Results
	var layer2Results Layer2Results

	if !*skipRebuildFlag {
		fmt.Printf("🔄 PHASE 1: Rebuilding entire graph from Postgres...\n")
		if err := rebuildFullGraph(ctx, *repoFlag); err != nil {
			log.Error("Graph rebuild failed", "error", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Graph rebuilt successfully\n\n")

		// Validate Layer 1
		fmt.Printf("🌲 PHASE 2: Validating Layer 1 (Code Structure)...\n")
		layer1Results = validateLayer1(ctx, *repoFlag)
		printLayerResults("Layer 1", layer1Results.Status, layer1Results.Duration)

		// Validate Layer 2
		fmt.Printf("📝 PHASE 3: Validating Layer 2 (Temporal Data)...\n")
		layer2Results = validateLayer2(ctx, *repoFlag)
		printLayerResults("Layer 2", layer2Results.Status, layer2Results.Duration)
	} else {
		fmt.Printf("⏭️  Skipping graph rebuild (using existing graph)\n\n")
	}

	// PHASE 2: Load ground truth
	fmt.Printf("📖 PHASE 4: Loading ground truth dataset...\n")
	groundTruth, err := loadGroundTruth(*repoFlag)
	if err != nil {
		log.Error("Failed to load ground truth", "error", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Loaded %d test cases\n\n", len(groundTruth.TestCases))

	// PHASE 3: Validate Layer 3 (Issue Linking)
	fmt.Printf("🔗 PHASE 5: Validating Layer 3 (Issue Links)...\n")
	layer3Results := validateLayer3(ctx, *repoFlag, groundTruth)
	printLayerResults("Layer 3", layer3Results.Status, layer3Results.Duration)

	// PHASE 4: Pattern Coverage Analysis
	fmt.Printf("📊 PHASE 6: Analyzing pattern coverage...\n")
	patternCoverage := analyzePatternCoverage(groundTruth, layer3Results)
	printPatternCoverage(patternCoverage)

	// PHASE 5: Generate comprehensive report
	fmt.Printf("\n📝 PHASE 7: Generating comprehensive report...\n")
	report := generateReport(*repoFlag, layer1Results, layer2Results, layer3Results, patternCoverage)

	reportPath := filepath.Join(*outputDirFlag, fmt.Sprintf("%s_full_pipeline_report.json", *repoFlag))
	if err := saveReport(report, reportPath); err != nil {
		log.Error("Failed to save report", "error", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Report saved to %s\n", reportPath)

	// Print summary
	fmt.Printf("\n")
	fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  TEST SUMMARY                                                ║\n")
	fmt.Printf("╠══════════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║ F1 Score:        %-44.2f ║\n", layer3Results.F1Score)
	fmt.Printf("║ Precision:       %-44.2f ║\n", layer3Results.Precision)
	fmt.Printf("║ Recall:          %-44.2f ║\n", layer3Results.Recall)
	fmt.Printf("║ Status:          %-44s ║\n", report.OverallStatus)
	fmt.Printf("║ Recommendation:  %-44s ║\n", report.Recommendation)
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n")

	// Exit with appropriate code
	if report.OverallStatus == "FAIL" {
		os.Exit(1)
	}
}

func rebuildFullGraph(ctx context.Context, repo string) error {
	log := logrus.New()

	// 1. Wipe Neo4j (all nodes/edges for this repo)
	log.Info("Wiping Neo4j for repo", "repo", repo)
	if err := wipeNeo4jRepo(ctx, repo); err != nil {
		return fmt.Errorf("wipe failed: %w", err)
	}

	// 2. Rebuild Layer 1 (TreeSitter - fast, ~10-30s)
	log.Info("Rebuilding Layer 1 (TreeSitter)")
	if err := rebuildLayer1(ctx, repo); err != nil {
		return fmt.Errorf("Layer 1 failed: %w", err)
	}

	// 3. Rebuild Layer 2 (Commits/PRs from Postgres - fast, ~30-60s)
	log.Info("Rebuilding Layer 2 (Commits/PRs)")
	if err := rebuildLayer2(ctx, repo); err != nil {
		return fmt.Errorf("Layer 2 failed: %w", err)
	}

	// 4. Rebuild Layer 3 (Issue linking with LLM - slow, ~2-5 min)
	log.Info("Rebuilding Layer 3 (Issue linking)")
	if err := rebuildLayer3(ctx, repo); err != nil {
		return fmt.Errorf("Layer 3 failed: %w", err)
	}

	return nil
}

func wipeNeo4jRepo(ctx context.Context, repo string) error {
	// TODO: Implement - get neo4j connection from environment
	// For now, this is a stub
	fmt.Printf("  [TODO] Wipe Neo4j for repo: %s\n", repo)
	return nil
}

func rebuildLayer1(ctx context.Context, repo string) error {
	// TODO: Call existing TreeSitter ingestion
	// This should parse code and create File/Function/Dependency nodes
	return nil
}

func rebuildLayer2(ctx context.Context, repo string) error {
	// TODO: Call existing Git/GitHub ingestion
	// This should create Commit/PR/Developer nodes and edges
	return nil
}

func rebuildLayer3(ctx context.Context, repo string) error {
	// TODO: Call issue linking logic
	// This should create FIXES_ISSUE edges
	return nil
}

func validateLayer1(ctx context.Context, repo string) Layer1Results {
	start := time.Now()
	// TODO: Implement Layer 1 validation when integrating
	return Layer1Results{
		FilesCreated:      0,
		DependenciesFound: 0,
		TreesitterErrors:  0,
		Status:            "SKIP",
		Duration:          time.Since(start),
	}
}

func validateLayer2(ctx context.Context, repo string) Layer2Results {
	start := time.Now()
	// TODO: Implement Layer 2 validation when integrating
	return Layer2Results{
		CommitsCreated:    0,
		PRsCreated:        0,
		DevelopersCreated: 0,
		ModifiedEdges:     0,
		Status:            "SKIP",
		Duration:          time.Since(start),
	}
}

func validateLayer3(ctx context.Context, repo string, groundTruth *GroundTruth) Layer3Results {
	start := time.Now()
	// TODO: Implement Layer 3 validation when integrating
	// This is the most important part - will validate issue linking accuracy
	return Layer3Results{
		TestCases: make([]TestCaseResult, 0),
		TotalTP:   0,
		TotalFP:   0,
		TotalFN:   0,
		Precision: 0.0,
		Recall:    0.0,
		F1Score:   0.0,
		Status:    "SKIP",
		Duration:  time.Since(start),
	}
}

func validateTestCase(ctx context.Context, db interface{}, repo string, testCase GroundTruthTestCase) TestCaseResult {
	// TODO: Implement when integrating - stub for now
	return TestCaseResult{}
}


func calculateMetrics(expected ExpectedLinks, actual []ActualLink) (tp, fp, fn int) {
	// Build set of expected PR numbers
	expectedPRs := make(map[int]bool)
	for _, pr := range expected.AssociatedPRs {
		expectedPRs[pr] = true
	}

	// Build set of expected commit SHAs
	expectedCommits := make(map[string]bool)
	for _, commit := range expected.FixedByCommits {
		expectedCommits[commit] = true
	}

	// Count matches
	foundPRs := make(map[int]bool)
	foundCommits := make(map[string]bool)

	for _, link := range actual {
		if link.Type == "PullRequest" {
			var num int
			if _, err := fmt.Sscanf(link.ID, "%d", &num); err == nil {
				if expectedPRs[num] {
					tp++
					foundPRs[num] = true
				} else {
					fp++
				}
			}
		} else if link.Type == "Commit" {
			if expectedCommits[link.ID] {
				tp++
				foundCommits[link.ID] = true
			} else {
				fp++
			}
		}
	}

	// Count false negatives
	for pr := range expectedPRs {
		if !foundPRs[pr] {
			fn++
		}
	}
	for commit := range expectedCommits {
		if !foundCommits[commit] {
			fn++
		}
	}

	return tp, fp, fn
}

func analyzePatternCoverage(groundTruth *GroundTruth, results Layer3Results) PatternCoverage {
	coverage := PatternCoverage{
		ByPattern: make(map[string]PatternStats),
	}

	for i, testCase := range groundTruth.TestCases {
		for _, pattern := range testCase.LinkingPatterns {
			stats := coverage.ByPattern[pattern]
			stats.TotalCases++

			if i < len(results.TestCases) {
				if results.TestCases[i].Status == "PASS" || results.TestCases[i].Status == "PARTIAL" {
					stats.Detected++
				} else {
					stats.Missed++
				}
			}

			coverage.ByPattern[pattern] = stats
		}
	}

	// Calculate detection rates
	for pattern, stats := range coverage.ByPattern {
		if stats.TotalCases > 0 {
			stats.DetectionRate = float64(stats.Detected) / float64(stats.TotalCases) * 100
		}
		coverage.ByPattern[pattern] = stats
	}

	return coverage
}

func loadGroundTruth(repo string) (*GroundTruth, error) {
	filename := fmt.Sprintf("test_data/%s_ground_truth.json", repo)
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read ground truth file: %w", err)
	}

	var gt GroundTruth
	if err := json.Unmarshal(data, &gt); err != nil {
		return nil, fmt.Errorf("failed to parse ground truth JSON: %w", err)
	}

	return &gt, nil
}

func generateReport(repo string, l1 Layer1Results, l2 Layer2Results, l3 Layer3Results, pc PatternCoverage) FullPipelineReport {
	status := "FAIL"
	recommendation := "RED LIGHT - Do not proceed to Supabase backtests"

	if l3.F1Score >= 75 {
		status = "PASS"
		recommendation = "GREEN LIGHT - Proceed to Supabase backtests"
	} else if l3.F1Score >= 60 {
		status = "ACCEPTABLE"
		recommendation = "YELLOW LIGHT - More tuning needed"
	}

	return FullPipelineReport{
		Repository:      repo,
		TestDate:        time.Now().Format(time.RFC3339),
		Layer1:          l1,
		Layer2:          l2,
		Layer3:          l3,
		PatternCoverage: pc,
		OverallStatus:   status,
		Recommendation:  recommendation,
	}
}

func saveReport(report FullPipelineReport, path string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func printLayerResults(name, status string, duration time.Duration) {
	statusIcon := "✅"
	if status == "FAIL" {
		statusIcon = "❌"
	} else if status == "ACCEPTABLE" {
		statusIcon = "🟡"
	}
	fmt.Printf("%s %s: %s (%.2fs)\n", statusIcon, name, status, duration.Seconds())
}

func printPatternCoverage(coverage PatternCoverage) {
	fmt.Printf("\n📊 PATTERN COVERAGE ANALYSIS\n")
	fmt.Printf("╔══════════════════════╦═══════╦═════════╦═════════╦═══════════╗\n")
	fmt.Printf("║ Pattern              ║ Total ║ Detected║ Missed  ║ Rate      ║\n")
	fmt.Printf("╠══════════════════════╬═══════╬═════════╬═════════╬═══════════╣\n")

	patterns := []string{"explicit", "temporal", "semantic", "comment", "cross_ref", "merge", "bidirectional", "none"}
	for _, pattern := range patterns {
		stats, exists := coverage.ByPattern[pattern]
		if !exists {
			continue
		}

		status := "✅"
		if stats.DetectionRate < 70 {
			status = "❌"
		} else if stats.DetectionRate < 85 {
			status = "🟡"
		}

		fmt.Printf("║ %-20s ║ %5d ║ %7d ║ %7d ║ %s %5.1f%% ║\n",
			pattern, stats.TotalCases, stats.Detected, stats.Missed,
			status, stats.DetectionRate)
	}

	fmt.Printf("╚══════════════════════╩═══════╩═════════╩═════════╩═══════════╝\n")
}
