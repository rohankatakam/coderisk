package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/graph"
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
	Use:   "crisk-ingest",
	Short: "Build Neo4j graph from staged data",
	Long: `crisk-ingest - Microservice 2: Fact Graph Builder

Builds the 100% confidence graph skeleton from staged PostgreSQL data.
Creates nodes and edges with NO LLM involvement - only verifiable facts.

Features:
  • Creates nodes: Developer, Commit, File, Issue, PR
  • Creates edges: AUTHORED, MODIFIED, CREATED, MERGED_AS, REFERENCES, CLOSED_BY
  • Uses file identity map for canonical paths
  • Batch processing with Neo4j transactions
  • Timeline event processing (100% confidence)

Output: Base Neo4j graph with verifiable connections`,
	Version: Version,
	RunE:    runIngest,
}

var (
	repoID  int64
	repoPath string
	verbose bool
)

func init() {
	rootCmd.Flags().Int64Var(&repoID, "repo-id", 0, "Repository ID from PostgreSQL (required)")
	rootCmd.Flags().StringVar(&repoPath, "repo-path", "", "Repository path for file resolution (optional)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	rootCmd.MarkFlagRequired("repo-id")

	rootCmd.SetVersionTemplate(`crisk-ingest {{.Version}}
Build time: ` + BuildTime + `
Git commit: ` + GitCommit + `
`)
}

func runIngest(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := context.Background()

	fmt.Printf("🚀 crisk-ingest - Graph Construction Service\n")
	fmt.Printf("   Repository ID: %d\n", repoID)
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
	fmt.Printf("  ✓ Connected to Neo4j\n\n")

	// Verify staged data
	fmt.Printf("[2/4] Verifying staged data...\n")
	stats, err := verifyStagedData(ctx, stagingDB, repoID)
	if err != nil {
		return err
	}

	fmt.Printf("  ✓ Repository: %s\n", stats.FullName)
	fmt.Printf("  ✓ Commits: %d | Issues: %d | PRs: %d\n\n",
		stats.Commits, stats.Issues, stats.PRs)

	// Get repository path from database if not provided
	if repoPath == "" {
		row := stagingDB.QueryRow(ctx, "SELECT absolute_path FROM github_repositories WHERE id = $1", repoID)
		var dbPath *string
		if err := row.Scan(&dbPath); err == nil && dbPath != nil && *dbPath != "" {
			repoPath = *dbPath
		} else {
			// Fall back to current directory
			repoPath, _ = os.Getwd()
		}
	}
	fmt.Printf("  Using repository path: %s\n\n", repoPath)

	// Build graph
	fmt.Printf("[3/4] Building 100%% confidence graph...\n")
	buildStart := time.Now()

	builder := graph.NewBuilder(stagingDB, graphBackend)
	buildStats, err := builder.BuildGraph(ctx, repoID, repoPath)
	if err != nil {
		return fmt.Errorf("graph build failed: %w", err)
	}

	buildDuration := time.Since(buildStart)
	fmt.Printf("  ✓ Graph built in %v\n", buildDuration)
	fmt.Printf("    Nodes: %d | Edges: %d\n\n", buildStats.Nodes, buildStats.Edges)

	// Create indexes
	fmt.Printf("[4/4] Creating database indexes...\n")
	indexStart := time.Now()

	if err := createIndexes(ctx, graphBackend); err != nil {
		fmt.Printf("  ⚠️  Index creation failed (non-fatal): %v\n", err)
	} else {
		indexDuration := time.Since(indexStart)
		fmt.Printf("  ✓ Indexes created in %v\n\n", indexDuration)
	}

	// Summary
	totalDuration := time.Since(startTime)
	fmt.Printf("✅ Graph construction complete\n")
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("   Total time: %v\n", totalDuration)
	fmt.Printf("   Nodes: %d | Edges: %d\n", buildStats.Nodes, buildStats.Edges)
	fmt.Printf("\n🚀 Next: crisk-atomize --repo-id %d --repo-path %s\n", repoID, repoPath)

	return nil
}

func verifyStagedData(ctx context.Context, stagingDB *database.StagingClient, repoID int64) (*StagedDataStats, error) {
	stats := &StagedDataStats{}

	// Check repository exists
	row := stagingDB.QueryRow(ctx, "SELECT full_name FROM github_repositories WHERE id = $1", repoID)
	if err := row.Scan(&stats.FullName); err != nil {
		return nil, fmt.Errorf("repo_id=%d not found\n\nRun 'crisk-stage' first", repoID)
	}

	// Count staged entities
	row = stagingDB.QueryRow(ctx, "SELECT COUNT(*) FROM github_commits WHERE repo_id = $1", repoID)
	row.Scan(&stats.Commits)

	row = stagingDB.QueryRow(ctx, "SELECT COUNT(*) FROM github_issues WHERE repo_id = $1", repoID)
	row.Scan(&stats.Issues)

	row = stagingDB.QueryRow(ctx, "SELECT COUNT(*) FROM github_pull_requests WHERE repo_id = $1", repoID)
	row.Scan(&stats.PRs)

	if stats.Commits == 0 {
		return nil, fmt.Errorf("no commits found for repo_id=%d\n\nRun 'crisk-stage' first", repoID)
	}

	return stats, nil
}

type StagedDataStats struct {
	FullName string
	Commits  int
	Issues   int
	PRs      int
}

func createIndexes(ctx context.Context, backend graph.Backend) error {
	indexes := []struct {
		name  string
		query string
	}{
		{
			"file_repo_path_unique",
			"CREATE CONSTRAINT file_repo_path_unique IF NOT EXISTS FOR (f:File) REQUIRE (f.repo_id, f.path) IS UNIQUE",
		},
		{
			"commit_repo_sha_unique",
			"CREATE CONSTRAINT commit_repo_sha_unique IF NOT EXISTS FOR (c:Commit) REQUIRE (c.repo_id, c.sha) IS UNIQUE",
		},
		{
			"pr_repo_number_unique",
			"CREATE CONSTRAINT pr_repo_number_unique IF NOT EXISTS FOR (pr:PR) REQUIRE (pr.repo_id, pr.number) IS UNIQUE",
		},
		{
			"issue_repo_number_unique",
			"CREATE CONSTRAINT issue_repo_number_unique IF NOT EXISTS FOR (i:Issue) REQUIRE (i.repo_id, i.number) IS UNIQUE",
		},
		{
			"developer_email_unique",
			"CREATE CONSTRAINT developer_email_unique IF NOT EXISTS FOR (d:Developer) REQUIRE d.email IS UNIQUE",
		},
	}

	for _, idx := range indexes {
		if _, err := backend.Query(ctx, idx.query); err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.name, err)
		}
	}

	return nil
}
