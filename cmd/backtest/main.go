package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/rohankatakam/coderisk/internal/backtest"
	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/graph"
)

func main() {
	// Parse command-line flags
	groundTruthPath := flag.String("ground-truth", "test_data/omnara_ground_truth.json", "Path to ground truth JSON file")
	repoID := flag.Int64("repo-id", 1, "Repository ID in database")
	outputDir := flag.String("output", "test_results", "Output directory for reports")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	runTemporal := flag.Bool("temporal", true, "Run temporal validation")
	runSemantic := flag.Bool("semantic", true, "Run semantic validation")
	runCLQS := flag.Bool("clqs", true, "Run CLQS benchmark")

	flag.Parse()

	if *verbose {
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	} else {
		log.SetFlags(log.Ltime)
	}

	log.Printf("🚀 CodeRisk Backtesting Framework")
	log.Printf("═══════════════════════════════════")

	ctx := context.Background()

	// Load configuration from environment or config file
	cfg, err := config.Load(".coderisk.yaml")
	if err != nil {
		// If config file doesn't exist, use environment variables
		log.Printf("  ⚠️  Config file not found, using environment variables")
		cfg = &config.Config{
			Storage: config.StorageConfig{
				PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
				PostgresPort:     getEnvInt("POSTGRES_PORT", 5432),
				PostgresDB:       getEnv("POSTGRES_DB", "coderisk"),
				PostgresUser:     getEnv("POSTGRES_USER", "coderisk"),
				PostgresPassword: getEnv("POSTGRES_PASSWORD", "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"),
			},
			Neo4j: config.Neo4jConfig{
				URI:      getEnv("NEO4J_URI", "bolt://localhost:7688"),
				User:     getEnv("NEO4J_USER", "neo4j"),
				Password: getEnv("NEO4J_PASSWORD", "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"),
			},
		}
	}

	// Initialize database connections
	log.Printf("\n📊 Initializing database connections...")

	stagingDB, err := database.NewStagingClient(
		ctx,
		cfg.Storage.PostgresHost,
		cfg.Storage.PostgresPort,
		cfg.Storage.PostgresDB,
		cfg.Storage.PostgresUser,
		cfg.Storage.PostgresPassword,
	)
	if err != nil {
		log.Fatalf("Failed to create staging DB client: %v", err)
	}
	defer stagingDB.Close()
	log.Printf("  ✓ Connected to PostgreSQL")

	neo4jDB, err := graph.NewClient(ctx, cfg.Neo4j.URI, cfg.Neo4j.User, cfg.Neo4j.Password)
	if err != nil {
		log.Fatalf("Failed to create Neo4j client: %v", err)
	}
	defer neo4jDB.Close(ctx)
	log.Printf("  ✓ Connected to Neo4j")

	// Load ground truth
	log.Printf("\n📖 Loading ground truth data...")
	groundTruth, err := backtest.LoadGroundTruth(*groundTruthPath)
	if err != nil {
		log.Fatalf("Failed to load ground truth: %v", err)
	}
	log.Printf("  ✓ Loaded %d test cases from %s", groundTruth.TotalCases, *groundTruthPath)
	log.Printf("    Repository: %s", groundTruth.Repository)
	log.Printf("    Pattern distribution: %v", groundTruth.PatternDistribution)

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	reportPrefix := filepath.Join(*outputDir, fmt.Sprintf("backtest_%s", timestamp))

	// Run backtesting
	log.Printf("\n🧪 Running Backtesting Suite...")
	log.Printf("═══════════════════════════════════")

	// 1. Full backtesting (all patterns)
	log.Printf("\n[1/4] Running comprehensive backtest...")
	backtester, err := backtest.NewBacktester(stagingDB, neo4jDB, *groundTruthPath)
	if err != nil {
		log.Fatalf("Failed to create backtester: %v", err)
	}

	backtestReport, err := backtester.RunBacktest(ctx, *repoID)
	if err != nil {
		log.Fatalf("Backtesting failed: %v", err)
	}

	// Save comprehensive report
	backtestReportPath := reportPrefix + "_comprehensive.json"
	if err := backtester.SaveReport(backtestReport, backtestReportPath); err != nil {
		log.Printf("  ⚠️  Failed to save comprehensive report: %v", err)
	}

	// 2. Temporal validation
	var temporalReport map[string]interface{}
	if *runTemporal {
		log.Printf("\n[2/4] Running temporal pattern validation...")
		temporalMatcher := backtest.NewTemporalMatcher(stagingDB, neo4jDB)

		temporalResults, err := temporalMatcher.ValidateAllTemporalCases(ctx, groundTruth)
		if err != nil {
			log.Printf("  ⚠️  Temporal validation failed: %v", err)
		} else {
			temporalMetrics := temporalMatcher.CalculateTemporalMetrics(temporalResults)
			temporalReport = map[string]interface{}{
				"timestamp": time.Now().Format(time.RFC3339),
				"results":   temporalResults,
				"metrics":   temporalMetrics,
			}

			// Save temporal report
			temporalReportPath := reportPrefix + "_temporal.json"
			if err := saveJSONReport(temporalReport, temporalReportPath); err != nil {
				log.Printf("  ⚠️  Failed to save temporal report: %v", err)
			}

			// Print temporal metrics
			log.Printf("\n  📊 Temporal Pattern Metrics:")
			log.Printf("    Precision: %.2f%%", temporalMetrics["precision"].(float64)*100)
			log.Printf("    Recall: %.2f%%", temporalMetrics["recall"].(float64)*100)
			log.Printf("    F1 Score: %.2f%%", temporalMetrics["f1_score"].(float64)*100)
			log.Printf("    Avg Confidence: %.2f", temporalMetrics["avg_confidence"].(float64))
		}
	}

	// 3. Semantic validation
	var semanticReport map[string]interface{}
	if *runSemantic {
		log.Printf("\n[3/4] Running semantic pattern validation...")
		semanticMatcher := backtest.NewSemanticMatcher(stagingDB, neo4jDB)

		semanticResults, err := semanticMatcher.ValidateAllSemanticCases(ctx, groundTruth)
		if err != nil {
			log.Printf("  ⚠️  Semantic validation failed: %v", err)
		} else {
			semanticMetrics := semanticMatcher.CalculateSemanticMetrics(semanticResults)
			semanticReport = map[string]interface{}{
				"timestamp": time.Now().Format(time.RFC3339),
				"results":   semanticResults,
				"metrics":   semanticMetrics,
			}

			// Save semantic report
			semanticReportPath := reportPrefix + "_semantic.json"
			if err := saveJSONReport(semanticReport, semanticReportPath); err != nil {
				log.Printf("  ⚠️  Failed to save semantic report: %v", err)
			}

			// Print semantic metrics
			log.Printf("\n  📊 Semantic Pattern Metrics:")
			log.Printf("    Precision: %.2f%%", semanticMetrics["precision"].(float64)*100)
			log.Printf("    Recall: %.2f%%", semanticMetrics["recall"].(float64)*100)
			log.Printf("    F1 Score: %.2f%%", semanticMetrics["f1_score"].(float64)*100)
			log.Printf("    Avg Semantic Score: %.2f", semanticMetrics["avg_semantic_score"].(float64))
			log.Printf("    Avg Confidence: %.2f", semanticMetrics["avg_confidence"].(float64))
		}
	}

	// 4. CLQS Benchmark
	var clqsReport *graph.CLQSReport
	if *runCLQS {
		log.Printf("\n[4/4] Running CLQS (Codebase Linking Quality Score) benchmark...")
		clqsCalculator := graph.NewLinkingQualityScore(stagingDB, neo4jDB)

		clqsReport, err = clqsCalculator.CalculateCLQS(ctx, *repoID, groundTruth.Repository)
		if err != nil {
			log.Printf("  ⚠️  CLQS calculation failed: %v", err)
		} else {
			// Save CLQS report
			clqsReportPath := reportPrefix + "_clqs.json"
			if err := saveCLQSReport(clqsReport, clqsReportPath); err != nil {
				log.Printf("  ⚠️  Failed to save CLQS report: %v", err)
			}

			// Print CLQS summary
			log.Printf("\n  📊 CLQS Benchmark Results:")
			log.Printf("    Overall Score: %.1f/100 (%s - %s)",
				clqsReport.OverallScore, clqsReport.Grade, clqsReport.Rank)
			log.Printf("    Explicit Linking: %.1f%% (weight: %.0f%%)",
				clqsReport.Components.ExplicitLinking.Score,
				clqsReport.Components.ExplicitLinking.Weight*100)
			log.Printf("    Temporal Correlation: %.1f%% (weight: %.0f%%)",
				clqsReport.Components.TemporalCorrelation.Score,
				clqsReport.Components.TemporalCorrelation.Weight*100)
			log.Printf("    Comment Quality: %.1f%% (weight: %.0f%%)",
				clqsReport.Components.CommentQuality.Score,
				clqsReport.Components.CommentQuality.Weight*100)
			log.Printf("    Semantic Consistency: %.1f%% (weight: %.0f%%)",
				clqsReport.Components.SemanticConsistency.Score,
				clqsReport.Components.SemanticConsistency.Weight*100)
			log.Printf("    Bidirectional Refs: %.1f%% (weight: %.0f%%)",
				clqsReport.Components.BidirectionalRefs.Score,
				clqsReport.Components.BidirectionalRefs.Weight*100)
		}
	}

	// Generate summary report
	log.Printf("\n📊 Generating Summary Report...")
	summaryReport := generateSummaryReport(backtestReport, temporalReport, semanticReport, clqsReport, groundTruth)

	summaryReportPath := reportPrefix + "_summary.json"
	if err := saveJSONReport(summaryReport, summaryReportPath); err != nil {
		log.Printf("  ⚠️  Failed to save summary report: %v", err)
	}

	// Print final summary
	log.Printf("\n═══════════════════════════════════")
	log.Printf("✅ Backtesting Complete!")
	log.Printf("═══════════════════════════════════")
	log.Printf("\n📄 Reports saved to:")
	log.Printf("  • Comprehensive: %s", backtestReportPath)
	if *runTemporal && temporalReport != nil {
		log.Printf("  • Temporal: %s", reportPrefix+"_temporal.json")
	}
	if *runSemantic && semanticReport != nil {
		log.Printf("  • Semantic: %s", reportPrefix+"_semantic.json")
	}
	if *runCLQS && clqsReport != nil {
		log.Printf("  • CLQS: %s", reportPrefix+"_clqs.json")
	}
	log.Printf("  • Summary: %s", summaryReportPath)

	// Compare against target metrics from ground truth
	log.Printf("\n🎯 Target Metrics Comparison:")
	log.Printf("  Target Precision: %.2f%% | Actual: %.2f%% | %s",
		groundTruth.ValidationMetrics.TargetPrecision*100,
		backtestReport.Metrics.Precision*100,
		getPassFail(backtestReport.Metrics.Precision >= groundTruth.ValidationMetrics.TargetPrecision))
	log.Printf("  Target Recall: %.2f%% | Actual: %.2f%% | %s",
		groundTruth.ValidationMetrics.TargetRecall*100,
		backtestReport.Metrics.Recall*100,
		getPassFail(backtestReport.Metrics.Recall >= groundTruth.ValidationMetrics.TargetRecall))
	log.Printf("  Target F1: %.2f%% | Actual: %.2f%% | %s",
		groundTruth.ValidationMetrics.TargetF1*100,
		backtestReport.Metrics.F1Score*100,
		getPassFail(backtestReport.Metrics.F1Score >= groundTruth.ValidationMetrics.TargetF1))

	// Exit with appropriate code
	if backtestReport.Metrics.F1Score >= groundTruth.ValidationMetrics.TargetF1 {
		log.Printf("\n✅ All target metrics met!")
		os.Exit(0)
	} else {
		log.Printf("\n❌ Some target metrics not met")
		os.Exit(1)
	}
}

func saveJSONReport(report interface{}, path string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	log.Printf("  ✓ Saved: %s", path)
	return nil
}

func saveCLQSReport(report *graph.CLQSReport, path string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal CLQS report: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write CLQS report: %w", err)
	}

	log.Printf("  ✓ Saved: %s", path)
	return nil
}

func generateSummaryReport(
	backtestReport *backtest.BacktestReport,
	temporalReport map[string]interface{},
	semanticReport map[string]interface{},
	clqsReport *graph.CLQSReport,
	groundTruth *backtest.GroundTruth,
) map[string]interface{} {
	summary := map[string]interface{}{
		"timestamp":   time.Now().Format(time.RFC3339),
		"repository":  groundTruth.Repository,
		"test_date":   backtestReport.TestDate,
		"total_cases": backtestReport.TotalCases,
	}

	// Overall metrics
	summary["overall_metrics"] = map[string]interface{}{
		"precision": backtestReport.Metrics.Precision,
		"recall":    backtestReport.Metrics.Recall,
		"f1_score":  backtestReport.Metrics.F1Score,
		"accuracy":  backtestReport.Metrics.Accuracy,
	}

	// Target comparison
	summary["target_comparison"] = map[string]interface{}{
		"target_precision": groundTruth.ValidationMetrics.TargetPrecision,
		"target_recall":    groundTruth.ValidationMetrics.TargetRecall,
		"target_f1":        groundTruth.ValidationMetrics.TargetF1,
		"meets_targets": backtestReport.Metrics.Precision >= groundTruth.ValidationMetrics.TargetPrecision &&
			backtestReport.Metrics.Recall >= groundTruth.ValidationMetrics.TargetRecall &&
			backtestReport.Metrics.F1Score >= groundTruth.ValidationMetrics.TargetF1,
	}

	// Pattern-specific metrics
	if temporalReport != nil {
		summary["temporal_metrics"] = temporalReport["metrics"]
	}
	if semanticReport != nil {
		summary["semantic_metrics"] = semanticReport["metrics"]
	}

	// CLQS scores
	if clqsReport != nil {
		summary["clqs"] = map[string]interface{}{
			"overall_score": clqsReport.OverallScore,
			"grade":         clqsReport.Grade,
			"rank":          clqsReport.Rank,
			"components": map[string]float64{
				"explicit_linking":      clqsReport.Components.ExplicitLinking.Score,
				"temporal_correlation":  clqsReport.Components.TemporalCorrelation.Score,
				"comment_quality":       clqsReport.Components.CommentQuality.Score,
				"semantic_consistency":  clqsReport.Components.SemanticConsistency.Score,
				"bidirectional_refs":    clqsReport.Components.BidirectionalRefs.Score,
			},
		}
	}

	return summary
}

func getPassFail(pass bool) string {
	if pass {
		return "✅ PASS"
	}
	return "❌ FAIL"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
