package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/github"
	"github.com/rohankatakam/coderisk/internal/llm"
	"github.com/spf13/cobra"
)

var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract issue-commit-PR relationships using LLM",
	Long: `Extract relationships between issues, commits, and PRs using OpenAI GPT-4o-mini.

This command runs two-way extraction:
  1. Extract issue references from commits and PRs
  2. Extract commit/PR references from issues
  3. Extract cross-references from issue timeline events

Results are stored in PostgreSQL for graph construction.

Requires: OPENAI_API_KEY environment variable or configured API key.`,
	RunE: runExtract,
}

func init() {
	rootCmd.AddCommand(extractCmd)
}

func runExtract(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load config with environment variable overrides
	cfg := config.Default()

	// Apply environment variable overrides for database connection
	if host := os.Getenv("POSTGRES_HOST"); host != "" {
		cfg.Storage.PostgresHost = host
	} else {
		cfg.Storage.PostgresHost = "localhost"
	}

	if port := os.Getenv("POSTGRES_PORT_EXTERNAL"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Storage.PostgresPort)
	} else {
		cfg.Storage.PostgresPort = 5433
	}

	if db := os.Getenv("POSTGRES_DB"); db != "" {
		cfg.Storage.PostgresDB = db
	} else {
		cfg.Storage.PostgresDB = "coderisk"
	}

	if user := os.Getenv("POSTGRES_USER"); user != "" {
		cfg.Storage.PostgresUser = user
	} else {
		cfg.Storage.PostgresUser = "coderisk"
	}

	if pass := os.Getenv("POSTGRES_PASSWORD"); pass != "" {
		cfg.Storage.PostgresPassword = pass
	} else {
		cfg.Storage.PostgresPassword = "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
	}

	// Set OpenAI API key from environment if available
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		cfg.API.OpenAIKey = apiKey
	}

	// Check for OpenAI API key
	if cfg.API.OpenAIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY not configured. Set via environment or run 'crisk configure'")
	}

	log.Printf("🤖 Starting issue-commit-PR extraction...")
	log.Printf("   Model: GPT-4o-mini")
	log.Printf("   Batch size: 20 per API call")

	// Connect to PostgreSQL
	stagingDB, err := database.NewStagingClient(
		ctx,
		cfg.Storage.PostgresHost,
		cfg.Storage.PostgresPort,
		cfg.Storage.PostgresDB,
		cfg.Storage.PostgresUser,
		cfg.Storage.PostgresPassword,
	)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer stagingDB.Close()

	// Get repository ID
	repoID, err := getCurrentRepoID(ctx, stagingDB)
	if err != nil {
		return fmt.Errorf("failed to get repository ID: %w", err)
	}

	log.Printf("   Repository ID: %d", repoID)

	// Create LLM client
	llmClient, err := llm.NewClient(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}

	if !llmClient.IsEnabled() {
		return fmt.Errorf("LLM client not enabled. Set PHASE2_ENABLED=true and configure OPENAI_API_KEY")
	}

	// Create extractors
	issueExtractor := github.NewIssueExtractor(llmClient, stagingDB)
	commitExtractor := github.NewCommitExtractor(llmClient, stagingDB)

	// Phase 1: Extract from Issues
	log.Printf("\n[1/3] Extracting references from issues...")
	issueRefs, err := issueExtractor.ExtractReferences(ctx, repoID)
	if err != nil {
		return fmt.Errorf("issue extraction failed: %w", err)
	}
	log.Printf("   ✓ Extracted %d references from issues", issueRefs)

	// Phase 2: Extract from Commits
	log.Printf("\n[2/3] Extracting references from commits...")
	commitRefs, err := commitExtractor.ExtractCommitReferences(ctx, repoID)
	if err != nil {
		return fmt.Errorf("commit extraction failed: %w", err)
	}
	log.Printf("   ✓ Extracted %d references from commits", commitRefs)

	// Phase 3: Extract from PRs
	log.Printf("\n[3/3] Extracting references from pull requests...")
	prRefs, err := commitExtractor.ExtractPRReferences(ctx, repoID)
	if err != nil {
		return fmt.Errorf("PR extraction failed: %w", err)
	}
	log.Printf("   ✓ Extracted %d references from PRs", prRefs)

	// Summary
	totalRefs := issueRefs + commitRefs + prRefs
	log.Printf("\n✅ Extraction complete!")
	log.Printf("   Total references: %d", totalRefs)
	log.Printf("   - From issues: %d", issueRefs)
	log.Printf("   - From commits: %d", commitRefs)
	log.Printf("   - From PRs: %d", prRefs)
	log.Printf("\n💡 Next: Run 'crisk init' to rebuild graph with FIXED_BY edges")

	return nil
}

// getCurrentRepoID gets the repository ID from the database
// For now, we assume repo_id = 1 (first repository)
// TODO: Support multiple repositories
func getCurrentRepoID(ctx context.Context, stagingDB *database.StagingClient) (int64, error) {
	// For now, hardcode to 1 since we only have one repo
	// In production, we'd detect from git or pass as parameter
	return 1, nil
}
