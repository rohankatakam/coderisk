package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/linking"
	"github.com/rohankatakam/coderisk/internal/llm"
	"github.com/spf13/cobra"
)

var (
	repoFullName string
	days         int
	dryRun       bool
)

var rootCmd = &cobra.Command{
	Use:   "issue-pr-linker",
	Short: "Link GitHub issues to pull requests using multi-phase analysis",
	Long: `issue-pr-linker is a standalone tool that links GitHub issues to pull requests
using a comprehensive multi-phase approach:

Phase 0: Pre-processing (DORA metrics + Timeline API verification)
Phase 1: Explicit reference extraction (LLM-based)
Phase 2: Issue processing loop
  - Path A: Validate explicit links (bidirectional + semantic + temporal)
  - Path B: Deep link finder (bug classification + temporal-semantic search)

Reference: test_data/docs/linking/Issue_Flow.md

Requirements:
  - PostgreSQL database with staged GitHub data (issues, PRs, commits, timeline events)
  - OpenAI API key (for LLM-based analysis)
  - Repository must be already ingested via 'crisk init'

Example:
  issue-pr-linker --repo omnara-ai/omnara --days 90
  issue-pr-linker --repo myorg/myrepo --days 180 --dry-run`,
	RunE: runLinker,
}

func init() {
	rootCmd.Flags().StringVar(&repoFullName, "repo", "", "Repository full name (owner/repo)")
	rootCmd.Flags().IntVar(&days, "days", 90, "Time window for DORA metrics (0 = all history)")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Dry run mode (don't write to database)")

	rootCmd.MarkFlagRequired("repo")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runLinker(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	log.Printf("Issue-PR Linker")
	log.Printf("Repository: %s", repoFullName)
	log.Printf("Time window: %d days", days)
	log.Printf("Dry run: %v", dryRun)
	log.Printf("")

	// Load configuration
	cfg, err := loadConfig()
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
		return fmt.Errorf("repository not found: %s (run 'crisk init' first): %w", repoFullName, err)
	}
	log.Printf("  ✓ Found repository (ID: %d)", repoID)
	log.Printf("")

	// Check data availability
	if err := checkDataAvailability(ctx, stagingDB, repoID); err != nil {
		return err
	}

	// Initialize LLM client
	log.Printf("Initializing LLM client...")
	llmClient, err := llm.NewClient(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM client: %w", err)
	}

	if !llmClient.IsEnabled() {
		return fmt.Errorf("LLM client not enabled - OPENAI_API_KEY must be set")
	}
	log.Printf("  ✓ LLM client initialized")
	log.Printf("")

	if dryRun {
		log.Printf("⚠️  DRY RUN MODE: No data will be written to database")
		log.Printf("")
	}

	// Create orchestrator
	orchestrator := linking.NewOrchestrator(stagingDB, llmClient, repoID, days)

	// Run the linking pipeline
	if err := orchestrator.Run(ctx); err != nil {
		return fmt.Errorf("linking pipeline failed: %w", err)
	}

	log.Printf("✅ Issue-PR linking complete!")
	return nil
}

// loadConfig loads configuration from environment
func loadConfig() (*config.Config, error) {
	// Check for required environment variables
	if os.Getenv("OPENAI_API_KEY") == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	// Load config (will use environment variables)
	cfg, err := config.Load("")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Validate database configuration
	if cfg.Storage.PostgresHost == "" {
		return nil, fmt.Errorf("POSTGRES_HOST not configured")
	}

	return cfg, nil
}

// checkDataAvailability verifies required data exists
func checkDataAvailability(ctx context.Context, stagingDB *database.StagingClient, repoID int64) error {
	log.Printf("Checking data availability...")

	counts, err := stagingDB.GetDataCounts(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to get data counts: %w", err)
	}

	log.Printf("  Issues: %d", counts.Issues)
	log.Printf("  PRs: %d", counts.PRs)
	log.Printf("  Commits: %d", counts.Commits)

	if counts.Issues == 0 {
		return fmt.Errorf("no issues found - run 'crisk init' to ingest data")
	}

	if counts.PRs == 0 {
		return fmt.Errorf("no pull requests found - run 'crisk init' to ingest data")
	}

	log.Printf("  ✓ Data availability check passed")
	log.Printf("")
	return nil
}
