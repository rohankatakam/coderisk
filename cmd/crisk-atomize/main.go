package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rohankatakam/coderisk/internal/atomizer"
	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/llm"
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
	Use:   "crisk-atomize",
	Short: "Transform commits into semantic CodeBlock nodes",
	Long: `crisk-atomize - Microservice 3: Semantic Layer (THE MOAT)

Transforms commits into semantic CodeBlock nodes using LLM analysis.
This is the core differentiator that enables function-level risk assessment.

Features:
  • Topologically-ordered processing (ORDER BY topological_index ASC)
  • LLM diff analysis → CommitChangeEventLog JSON
  • Creates CodeBlock nodes with semantic relationships
  • Handles renames via RENAMED_FROM edges
  • Extracts IMPORTS_FROM edges

This service is MANDATORY in the microservice architecture.

Output: Semantic graph with CodeBlock granularity`,
	Version: Version,
	RunE:    runAtomize,
}

var (
	repoID   int64
	repoPath string
	verbose  bool
)

func init() {
	rootCmd.Flags().Int64Var(&repoID, "repo-id", 0, "Repository ID from PostgreSQL (required)")
	rootCmd.Flags().StringVar(&repoPath, "repo-path", "", "Local repository path (required)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	rootCmd.MarkFlagRequired("repo-id")
	rootCmd.MarkFlagRequired("repo-path")

	rootCmd.SetVersionTemplate(`crisk-atomize {{.Version}}
Build time: ` + BuildTime + `
Git commit: ` + GitCommit + `
`)
}

func runAtomize(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := context.Background()

	fmt.Printf("🚀 crisk-atomize - Code Block Atomization Service\n")
	fmt.Printf("   Repository ID: %d\n", repoID)
	fmt.Printf("   Repository Path: %s\n", repoPath)
	fmt.Println()

	// Check for LLM API key
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	if geminiAPIKey == "" {
		return fmt.Errorf("GEMINI_API_KEY environment variable not set\n\nAtomization requires LLM for semantic analysis")
	}

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Connect to PostgreSQL
	fmt.Printf("[1/4] Connecting to databases...\n")
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

	// Create LLM client
	fmt.Printf("[2/4] Initializing LLM client...\n")
	llmClient, err := llm.NewClient(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}

	if !llmClient.IsEnabled() {
		return fmt.Errorf("LLM client not enabled (check GEMINI_API_KEY)")
	}

	fmt.Printf("  ✓ LLM client ready (%s)\n\n", llmClient.GetProvider())

	// Create Neo4j driver
	fmt.Printf("[3/4] Connecting to Neo4j...\n")
	neoDriver, err := createNeo4jDriver(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create Neo4j driver: %w", err)
	}
	defer neoDriver.Close(ctx)

	fmt.Printf("  ✓ Connected to Neo4j\n\n")

	// Fetch commits chronologically
	fmt.Printf("[4/4] Processing commits chronologically...\n")
	processStart := time.Now()

	rows, err := stagingDB.Query(ctx, `
		SELECT sha, message, author_email, author_date, topological_index
		FROM github_commits
		WHERE repo_id = $1
		ORDER BY topological_index ASC NULLS LAST
	`, repoID)
	if err != nil {
		return fmt.Errorf("failed to fetch commits: %w", err)
	}
	defer rows.Close()

	var commits []atomizer.CommitData
	for rows.Next() {
		var sha, message, authorEmail string
		var authorDate time.Time
		var topoIndex *int

		if err := rows.Scan(&sha, &message, &authorEmail, &authorDate, &topoIndex); err != nil {
			fmt.Printf("  ⚠️  Failed to scan commit: %v\n", err)
			continue
		}

		// Fetch git diff for this commit
		diff, err := getCommitDiff(repoPath, sha)
		if err != nil {
			if verbose {
				fmt.Printf("  ⚠️  Failed to get diff for %s: %v\n", sha[:8], err)
			}
			continue
		}

		commits = append(commits, atomizer.CommitData{
			SHA:         sha,
			Message:     message,
			DiffContent: diff,
			AuthorEmail: authorEmail,
			Timestamp:   authorDate,
		})
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating commits: %w", err)
	}

	fmt.Printf("  ✓ Fetched %d commits with diffs\n", len(commits))

	// Create atomizer processor
	extractor := atomizer.NewExtractor(llmClient)
	processor := atomizer.NewProcessor(extractor, stagingDB.DB(), neoDriver, cfg.Neo4j.Database)

	// Run atomization
	fmt.Printf("  Processing %d commits...\n", len(commits))
	if err := processor.ProcessCommitsChronologically(ctx, commits, repoID); err != nil {
		return fmt.Errorf("chronological processing failed: %w", err)
	}

	processDuration := time.Since(processStart)

	// Verify results
	var blockCount int
	err = stagingDB.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM code_blocks WHERE repo_id = $1", repoID).Scan(&blockCount)
	if err != nil {
		fmt.Printf("  ⚠️  Failed to count code blocks: %v\n", err)
	} else {
		fmt.Printf("  ✓ Created %d code blocks\n", blockCount)
	}

	// Summary
	totalDuration := time.Since(startTime)
	fmt.Printf("\n✅ Atomization complete\n")
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("   Total time: %v\n", totalDuration)
	fmt.Printf("   Processing time: %v\n", processDuration)
	fmt.Printf("   Commits processed: %d\n", len(commits))
	fmt.Printf("   CodeBlocks created: %d\n", blockCount)
	fmt.Printf("\n🚀 Next: crisk-index-incident --repo-id %d\n", repoID)

	return nil
}

func getCommitDiff(repoPath, sha string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "show", "--format=", sha)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git show failed: %w", err)
	}
	return string(output), nil
}

func createNeo4jDriver(ctx context.Context, cfg *config.Config) (neo4j.DriverWithContext, error) {
	driver, err := neo4j.NewDriverWithContext(
		cfg.Neo4j.URI,
		neo4j.BasicAuth(cfg.Neo4j.User, cfg.Neo4j.Password, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	// Verify connectivity
	if err := driver.VerifyConnectivity(ctx); err != nil {
		driver.Close(ctx)
		return nil, fmt.Errorf("failed to connect to Neo4j: %w", err)
	}

	return driver, nil
}
