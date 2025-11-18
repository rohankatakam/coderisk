package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/graph"
	"github.com/spf13/cobra"
)

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Build Neo4j graph from staged PostgreSQL data",
	Long: `Build the Neo4j knowledge graph from data already staged in PostgreSQL.

This command transforms staged GitHub data (commits, PRs, issues) into a Neo4j graph
with optional code-block atomization (Pipeline 2) and ownership indexing (Pipeline 3).

Usage:
  crisk ingest --repo-id <id>             # Build graph for specific repo
  crisk ingest --repo-id 11 --llm         # Enable LLM-based link extraction
  crisk ingest --repo-id 11 --atomize     # Enable code-block atomization
  crisk ingest --repo-id 11 --all         # Enable all features

Examples:
  # Rebuild graph for repo_id=11 (basic graph only)
  crisk ingest --repo-id 11

  # Full ingestion with code blocks
  crisk ingest --repo-id 11 --llm --atomize

Requirements:
  • PostgreSQL with staged data (run 'crisk stage' first)
  • Neo4j running locally
  • GEMINI_API_KEY for --llm and --atomize flags`,
	RunE: runIngest,
}

func init() {
	ingestCmd.Flags().Int64("repo-id", 0, "Repository ID from PostgreSQL (required)")
	ingestCmd.Flags().Bool("llm", false, "Enable LLM-based ASSOCIATED_WITH edge extraction")
	ingestCmd.Flags().Bool("atomize", false, "Enable Pipeline 2 code-block atomization (requires --llm)")
	ingestCmd.Flags().Bool("all", false, "Enable all features (--llm --atomize)")
	ingestCmd.Flags().Bool("verify", false, "Verify staged data before ingestion")
	ingestCmd.MarkFlagRequired("repo-id")
}

func runIngest(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := context.Background()

	repoID, _ := cmd.Flags().GetInt64("repo-id")
	enableLLM, _ := cmd.Flags().GetBool("llm")
	enableAtomize, _ := cmd.Flags().GetBool("atomize")
	enableAll, _ := cmd.Flags().GetBool("all")
	verifyOnly, _ := cmd.Flags().GetBool("verify")

	if enableAll {
		enableLLM = true
		enableAtomize = true
	}

	if enableAtomize && !enableLLM {
		return fmt.Errorf("--atomize requires --llm flag")
	}

	fmt.Printf("🔄 CodeRisk Ingest (repo_id=%d)\n", repoID)
	fmt.Printf("   Mode: Graph Construction from Staged Data\n")
	if enableAtomize {
		fmt.Printf("   Features: Basic Graph + LLM Links + Code-Block Atomization\n")
	} else if enableLLM {
		fmt.Printf("   Features: Basic Graph + LLM Links\n")
	} else {
		fmt.Printf("   Features: Basic Graph Only\n")
	}

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

	// Connect to PostgreSQL
	fmt.Printf("\n[1/5] Connecting to databases...\n")
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

	// Connect to Neo4j
	graphBackend, err := graph.NewNeo4jBackend(
		ctx,
		cfg.Neo4j.URI,
		cfg.Neo4j.User,
		cfg.Neo4j.Password,
		cfg.Neo4j.Database,
	)
	if err != nil {
		return fmt.Errorf("Neo4j connection failed: %w", err)
	}
	defer graphBackend.Close(ctx)

	fmt.Printf("  ✓ Connected to PostgreSQL\n")
	fmt.Printf("  ✓ Connected to Neo4j\n")

	// Verify staged data exists
	fmt.Printf("\n[2/5] Verifying staged data for repo_id=%d...\n", repoID)
	stats, err := verifyStagedData(ctx, stagingDB, repoID)
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	fmt.Printf("  ✓ Repository: %s\n", stats.FullName)
	fmt.Printf("  ✓ Staged Data:\n")
	fmt.Printf("    - Commits: %d\n", stats.Commits)
	fmt.Printf("    - Issues: %d\n", stats.Issues)
	fmt.Printf("    - PRs: %d\n", stats.PRs)
	fmt.Printf("    - File Identity Map: %d files\n", stats.Files)

	if verifyOnly {
		fmt.Printf("\n✅ Verification complete (--verify flag, skipping ingestion)\n")
		return nil
	}

	// Get repository path (needed for git operations during atomization)
	var repoPath string
	if enableAtomize {
		// Check if we're in the repository directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Verify it's a git repo
		if _, err := os.Stat(filepath.Join(cwd, ".git")); err == nil {
			repoPath = cwd
			fmt.Printf("  ✓ Using repository at: %s\n", repoPath)
		} else {
			return fmt.Errorf("--atomize requires running from repository directory\n\ncd to %s and run again", stats.FullName)
		}
	}

	// Build basic graph (Pipeline 1)
	fmt.Printf("\n[3/5] Building knowledge graph...\n")
	buildStart := time.Now()

	// Get repository path for graph building (use absolute_path if available, else current dir)
	row := stagingDB.QueryRow(ctx, "SELECT absolute_path FROM github_repositories WHERE id = $1", repoID)
	var dbPath *string
	if err := row.Scan(&dbPath); err != nil || dbPath == nil || *dbPath == "" {
		// Fall back to current directory if not in database
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		repoPath = cwd
		fmt.Printf("  ℹ️  Using current directory: %s\n", repoPath)
	} else {
		repoPath = *dbPath
		fmt.Printf("  ℹ️  Using database path: %s\n", repoPath)
	}

	builder := graph.NewBuilder(stagingDB, graphBackend)
	buildStats, err := builder.BuildGraph(ctx, repoID, repoPath)
	if err != nil {
		return fmt.Errorf("graph build failed: %w", err)
	}

	buildDuration := time.Since(buildStart)
	fmt.Printf("  ✓ Graph built in %v\n", buildDuration)
	fmt.Printf("    Nodes: %d | Edges: %d\n", buildStats.Nodes, buildStats.Edges)

	// Pipeline 2: Code-Block Atomization (optional)
	if enableAtomize {
		fmt.Printf("\n[4/5] Pipeline 2: Code-block atomization...\n")
		atomizationStart := time.Now()

		// Use existing runPipeline2 from init.go
		if err := runPipeline2(ctx, stagingDB, graphBackend, repoID, repoPath); err != nil {
			// Don't fail entire pipeline, just log warning
			fmt.Printf("  ⚠️  Pipeline 2 failed: %v\n", err)
			fmt.Printf("  → Continuing without code-block atomization\n")
		} else {
			atomizationDuration := time.Since(atomizationStart)
			fmt.Printf("  ✓ Pipeline 2 complete in %v\n", atomizationDuration)
		}
	} else {
		fmt.Printf("\n[4/5] Pipeline 2: Skipped (use --atomize to enable)\n")
	}

	// Create indexes
	fmt.Printf("\n[5/5] Creating database indexes...\n")
	indexStart := time.Now()

	if err := createIndexes(ctx, graphBackend); err != nil {
		fmt.Printf("  ⚠️  Index creation failed (non-fatal): %v\n", err)
	} else {
		indexDuration := time.Since(indexStart)
		fmt.Printf("  ✓ Indexes created in %v\n", indexDuration)
	}

	// Summary
	totalDuration := time.Since(startTime)

	// Query Neo4j for CodeBlock count
	codeBlockCount := int64(0)
	if result, err := graphBackend.Query(ctx, fmt.Sprintf("MATCH (b:CodeBlock {repo_id: %d}) RETURN count(b) as count", repoID)); err == nil {
		codeBlockCount = result.(int64)
	}

	fmt.Printf("\n✅ Ingestion complete for repo_id=%d\n", repoID)
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("   Total time: %v\n", totalDuration)
	fmt.Printf("   Graph: %d nodes, %d edges\n", buildStats.Nodes, buildStats.Edges)
	if enableAtomize {
		fmt.Printf("   CodeBlocks: %d\n", codeBlockCount)
	}
	fmt.Printf("\n🚀 Next steps:\n")
	fmt.Printf("   • Test MCP server: restart crisk-check-server\n")
	fmt.Printf("   • Query graph: http://localhost:7475 (Neo4j Browser)\n")

	return nil
}

// verifyStagedData checks that required data exists in PostgreSQL
func verifyStagedData(ctx context.Context, stagingDB *database.StagingClient, repoID int64) (*StagedDataStats, error) {
	stats := &StagedDataStats{}

	// Check repository exists
	row := stagingDB.QueryRow(ctx, "SELECT full_name FROM github_repositories WHERE id = $1", repoID)
	if err := row.Scan(&stats.FullName); err != nil {
		return nil, fmt.Errorf("repo_id=%d not found in github_repositories table\n\nRun 'crisk stage' first to download GitHub data", repoID)
	}

	// Count staged entities
	row = stagingDB.QueryRow(ctx, "SELECT COUNT(*) FROM github_commits WHERE repo_id = $1", repoID)
	row.Scan(&stats.Commits)

	row = stagingDB.QueryRow(ctx, "SELECT COUNT(*) FROM github_issues WHERE repo_id = $1", repoID)
	row.Scan(&stats.Issues)

	row = stagingDB.QueryRow(ctx, "SELECT COUNT(*) FROM github_pull_requests WHERE repo_id = $1", repoID)
	row.Scan(&stats.PRs)

	row = stagingDB.QueryRow(ctx, "SELECT COUNT(*) FROM file_identity_map WHERE repo_id = $1", repoID)
	row.Scan(&stats.Files)

	if stats.Commits == 0 {
		return nil, fmt.Errorf("no commits found for repo_id=%d\n\nRun 'crisk stage' to download GitHub data", repoID)
	}

	return stats, nil
}

type StagedDataStats struct {
	FullName string
	Commits  int
	Issues   int
	PRs      int
	Files    int
}
