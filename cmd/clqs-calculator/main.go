package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/rohankatakam/coderisk/internal/clqs"
	"github.com/rohankatakam/coderisk/internal/database"
)

func main() {
	// Command-line flags
	repo := flag.String("repo", "", "Repository full name (e.g., 'omnara-ai/omnara')")
	format := flag.String("format", "text", "Output format: 'text' or 'json'")
	output := flag.String("output", "", "Output file path (default: stdout)")
	recalculate := flag.Bool("recalculate", false, "Force recalculation even if cached")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	if *repo == "" {
		fmt.Fprintf(os.Stderr, "Error: --repo flag is required\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s --repo OWNER/REPO [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Disable log output unless verbose
	if !*verbose {
		log.SetOutput(io.Discard)
	}

	// Get database connection parameters from environment
	pgHost := os.Getenv("POSTGRES_HOST")
	if pgHost == "" {
		pgHost = "localhost"
	}

	pgPortStr := os.Getenv("POSTGRES_PORT")
	if pgPortStr == "" {
		pgPortStr = "5433"
	}
	pgPort, err := strconv.Atoi(pgPortStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid POSTGRES_PORT: %v\n", err)
		os.Exit(1)
	}

	pgDB := os.Getenv("POSTGRES_DB")
	if pgDB == "" {
		pgDB = "coderisk"
	}

	pgUser := os.Getenv("POSTGRES_USER")
	if pgUser == "" {
		pgUser = "coderisk_user"
	}

	pgPassword := os.Getenv("POSTGRES_PASSWORD")
	if pgPassword == "" {
		fmt.Fprintf(os.Stderr, "Error: POSTGRES_PASSWORD environment variable is required\n")
		os.Exit(1)
	}

	// Initialize database connection
	ctx := context.Background()
	db, err := database.NewStagingClient(ctx, pgHost, pgPort, pgDB, pgUser, pgPassword)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Get repository ID
	repoID, err := db.GetRepositoryIDByFullName(ctx, *repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Repository '%s' not found: %v\n", *repo, err)
		os.Exit(1)
	}

	var report *clqs.CLQSReport

	// Check if we have a cached CLQS score
	if !*recalculate {
		cached, err := db.GetLatestCLQSScore(ctx, repoID)
		if err == nil && cached != nil {
			if *verbose {
				fmt.Fprintf(os.Stderr, "Using cached CLQS score from %s (use --recalculate to force)\n",
					cached.ComputedAt.Format("2006-01-02 15:04:05"))
			}
			// Build minimal report from cached score
			report = buildReportFromCached(ctx, db, cached)
		}
	}

	// Calculate CLQS if no cached score or recalculate requested
	if report == nil {
		if *verbose {
			fmt.Fprintf(os.Stderr, "Calculating CLQS for %s...\n", *repo)
		}

		calculator := clqs.NewCalculator(db)
		report, err = calculator.Calculate(ctx, repoID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "CLQS calculation failed: %v\n", err)
			os.Exit(1)
		}
	}

	// Output report
	if err := outputReport(report, *format, *output); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to output report: %v\n", err)
		os.Exit(1)
	}
}

// buildReportFromCached creates a minimal report from a cached CLQS score
func buildReportFromCached(ctx context.Context, db *database.StagingClient, cached *database.CLQSScore) *clqs.CLQSReport {
	repoInfo, _ := db.GetRepositoryInfo(ctx, cached.RepoID)

	return &clqs.CLQSReport{
		Repository: clqs.RepositoryInfo{
			ID:         cached.RepoID,
			FullName:   repoInfo.FullName,
			AnalyzedAt: cached.ComputedAt,
		},
		Overall: clqs.OverallScore{
			CLQS:                 cached.CLQS,
			Grade:                cached.Grade,
			Rank:                 cached.Rank,
			ConfidenceMultiplier: cached.ConfidenceMultiplier,
		},
		Components: []*clqs.ComponentBreakdown{
			{
				Name:         "Link Coverage",
				Score:        cached.LinkCoverage,
				Weight:       clqs.WeightLinkCoverage,
				Contribution: cached.LinkCoverageContribution,
				Status:       getStatus(cached.LinkCoverage),
			},
			{
				Name:         "Confidence Quality",
				Score:        cached.ConfidenceQuality,
				Weight:       clqs.WeightConfidenceQuality,
				Contribution: cached.ConfidenceQualityContribution,
				Status:       getStatus(cached.ConfidenceQuality),
			},
			{
				Name:         "Evidence Diversity",
				Score:        cached.EvidenceDiversity,
				Weight:       clqs.WeightEvidenceDiversity,
				Contribution: cached.EvidenceDiversityContribution,
				Status:       getStatus(cached.EvidenceDiversity),
			},
			{
				Name:         "Temporal Precision",
				Score:        cached.TemporalPrecision,
				Weight:       clqs.WeightTemporalPrecision,
				Contribution: cached.TemporalPrecisionContribution,
				Status:       getStatus(cached.TemporalPrecision),
			},
			{
				Name:         "Semantic Strength",
				Score:        cached.SemanticStrength,
				Weight:       clqs.WeightSemanticStrength,
				Contribution: cached.SemanticStrengthContribution,
				Status:       getStatus(cached.SemanticStrength),
			},
		},
		Statistics: clqs.Statistics{
			TotalClosedIssues: cached.TotalClosedIssues,
			EligibleIssues:    cached.EligibleIssues,
			LinkedIssues:      cached.LinkedIssues,
			TotalLinks:        cached.TotalLinks,
			AvgConfidence:     cached.AvgConfidence,
		},
		Recommendations:       []clqs.Recommendation{},
		LabelingOpportunities: clqs.LabelingOpportunities{},
	}
}

// getStatus determines status from score
func getStatus(score float64) string {
	switch {
	case score >= 85:
		return "Excellent"
	case score >= 75:
		return "Good"
	case score >= 60:
		return "Fair"
	default:
		return "Poor"
	}
}

// outputReport writes the report to the specified destination
func outputReport(report *clqs.CLQSReport, format, outputPath string) error {
	var w *os.File
	var err error

	if outputPath == "" {
		w = os.Stdout
	} else {
		w, err = os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer w.Close()
	}

	switch format {
	case "json":
		return clqs.FormatJSON(report, w)
	case "text":
		return clqs.FormatReport(report, w)
	default:
		return fmt.Errorf("unknown format: %s (use 'text' or 'json')", format)
	}
}
