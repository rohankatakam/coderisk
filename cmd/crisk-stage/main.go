package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/github"
	"github.com/rohankatakam/coderisk/internal/ingestion"
	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "crisk-stage",
	Short: "Stage GitHub data into PostgreSQL",
	Long: `crisk-stage - Microservice 1: Fact Collector

Downloads raw GitHub data and stores it in PostgreSQL staging tables.
Builds file identity map using git log --follow for canonical path resolution.

This service is idempotent and supports checkpointing for resume capability.

Features:
  • API rate limiting with exponential backoff
  • Idempotent storage (ON CONFLICT handling)
  • Checkpointing via processed_at column
  • File identity map for rename tracking

Output: repo_id and staged data ready for crisk-ingest`,
	Version: Version,
	RunE:    runStage,
}

var (
	repoOwner string
	repoName  string
	repoPath  string
	days      int
	verbose   bool
)

func init() {
	rootCmd.Flags().StringVar(&repoOwner, "owner", "", "GitHub repository owner (required)")
	rootCmd.Flags().StringVar(&repoName, "repo", "", "GitHub repository name (required)")
	rootCmd.Flags().StringVar(&repoPath, "path", "", "Local repository path (required)")
	rootCmd.Flags().IntVar(&days, "days", 0, "Fetch last N days only (0 = all history)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	rootCmd.MarkFlagRequired("owner")
	rootCmd.MarkFlagRequired("repo")
	rootCmd.MarkFlagRequired("path")

	rootCmd.SetVersionTemplate(`crisk-stage {{.Version}}
Build time: ` + BuildTime + `
Git commit: ` + GitCommit + `
`)
}

func runStage(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := context.Background()

	fmt.Printf("🚀 crisk-stage - GitHub Data Staging Service\n")
	fmt.Printf("   Repository: %s/%s\n", repoOwner, repoName)
	fmt.Printf("   Path: %s\n", repoPath)
	if days > 0 {
		fmt.Printf("   Time window: Last %d days\n", days)
	} else {
		fmt.Printf("   Time window: Full history\n")
	}
	fmt.Println()

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	mode := config.DetectMode()
	result := cfg.ValidateWithMode(config.ValidationContextInit, mode)
	if result.HasErrors() {
		return fmt.Errorf("configuration validation failed:\n%s", result.Error())
	}

	// Get GitHub token
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return fmt.Errorf("GITHUB_TOKEN environment variable not set")
	}

	// Connect to PostgreSQL
	fmt.Printf("[1/3] Connecting to PostgreSQL...\n")
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
	fmt.Printf("  ✓ Connected to PostgreSQL\n\n")

	// Fetch GitHub data
	fmt.Printf("[2/3] Fetching GitHub API data...\n")
	fetchStart := time.Now()

	fetcher := github.NewFetcher(githubToken, stagingDB)
	repoID, stats, err := fetcher.FetchAll(ctx, repoOwner, repoName, repoPath, days)
	if err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}

	fetchDuration := time.Since(fetchStart)
	fmt.Printf("  ✓ Fetched in %v\n", fetchDuration)
	fmt.Printf("    Commits: %d | Issues: %d | PRs: %d | Branches: %d\n\n",
		stats.Commits, stats.Issues, stats.PRs, stats.Branches)

	// Build file identity map (check if already exists)
	fmt.Printf("[3/3] Building file identity map...\n")
	identityStart := time.Now()

	identityRepo := database.NewFileIdentityRepository(stagingDB.DB())
	existingCount, err := identityRepo.GetCount(ctx, repoID)
	if err != nil {
		fmt.Printf("  ⚠️  Could not check existing identity map: %v\n", err)
	}

	var identityMapSize int
	if existingCount > 0 {
		fmt.Printf("  ℹ️  File identity map already exists (%d files), skipping rebuild\n", existingCount)
		identityMapSize = existingCount
	} else {
		identityMapper := ingestion.NewFileIdentityMapper(repoPath, repoID)
		identityMap, err := identityMapper.BuildIdentityMap(ctx)
		if err != nil {
			return fmt.Errorf("failed to build file identity map: %w", err)
		}

		// Store identity map
		if err := identityRepo.DeleteByRepoID(ctx, repoID); err != nil {
			fmt.Printf("  ⚠️  Failed to clean existing identity mappings: %v\n", err)
		}

		if err := identityRepo.BatchInsert(ctx, repoID, identityMap); err != nil {
			return fmt.Errorf("failed to store file identity mappings: %w", err)
		}

		identityMapSize = len(identityMap)
		fmt.Printf("  ✓ File identity map built in %v\n", time.Since(identityStart))
		fmt.Printf("    Traced %d source files\n", identityMapSize)
	}

	identityDuration := time.Since(identityStart)
	if existingCount > 0 {
		fmt.Printf("  ✓ File identity check completed in %v\n", identityDuration)
	}
	fmt.Println()

	// Summary
	totalDuration := time.Since(startTime)
	fmt.Printf("✅ Staging complete\n")
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("   Repository ID: %d\n", repoID)
	fmt.Printf("   Total time: %v\n", totalDuration)
	fmt.Printf("   GitHub entities: %d commits, %d issues, %d PRs\n",
		stats.Commits, stats.Issues, stats.PRs)
	fmt.Printf("   File identity map: %d files\n", identityMapSize)
	fmt.Printf("\n🚀 Next: crisk-ingest --repo-id %d\n", repoID)

	// Output repo_id for orchestrator (machine-readable format on last line)
	fmt.Printf("\nREPO_ID=%d\n", repoID)

	return nil
}
