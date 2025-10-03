package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coderisk/coderisk-go/internal/config"
	"github.com/coderisk/coderisk-go/internal/database"
	"github.com/coderisk/coderisk-go/internal/github"
	"github.com/coderisk/coderisk-go/internal/graph"
	"github.com/spf13/cobra"
)

var (
	backendType string
)

var initCmd = &cobra.Command{
	Use:   "init [repository]",
	Short: "Initialize CodeRisk for a repository",
	Long: `Initialize CodeRisk by fetching GitHub data and building the knowledge graph.

Examples:
  crisk init omnara-ai/omnara
  crisk init omnara-ai/omnara --backend neo4j

Configuration:
  All credentials are loaded from .env file in project root.
  Run 'cp .env.example .env' and fill in your GITHUB_TOKEN.

The repository must be specified as owner/repo format.`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVarP(&backendType, "backend", "b", "neo4j", "Graph backend: neo4j (local) or neptune (cloud)")
}

func runInit(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := context.Background()

	// Load environment variables from .env file
	envLoader := config.NewEnvLoader()
	envLoader.MustLoad()

	// Validate required variables including GitHub token
	if err := envLoader.ValidateWithGitHub(); err != nil {
		return err
	}

	// Parse repository name
	repoName := args[0]
	owner, repo, err := parseRepoName(repoName)
	if err != nil {
		return fmt.Errorf("invalid repository name: %w", err)
	}

	fmt.Printf("🚀 Initializing CodeRisk for %s/%s...\n", owner, repo)
	fmt.Printf("   Backend: %s\n", backendType)
	fmt.Printf("   Config: %s\n", envLoader.GetPath())

	// Connect to PostgreSQL
	fmt.Printf("\n[0/3] Connecting to databases...\n")
	stagingDB, err := database.NewStagingClient(
		ctx,
		config.GetString("POSTGRES_HOST", "localhost"),
		config.GetInt("POSTGRES_PORT_EXTERNAL", 5433),
		config.MustGetString("POSTGRES_DB"),
		config.MustGetString("POSTGRES_USER"),
		config.MustGetString("POSTGRES_PASSWORD"),
	)
	if err != nil {
		return fmt.Errorf("PostgreSQL connection failed: %w", err)
	}
	defer stagingDB.Close()

	// Connect to graph backend
	var graphBackend graph.Backend
	switch backendType {
	case "neo4j":
		// Construct Neo4j URI from components
		neo4jHost := config.GetString("NEO4J_HOST", "localhost")
		neo4jPort := config.GetInt("NEO4J_BOLT_PORT", 7688)
		neo4jURI := fmt.Sprintf("bolt://%s:%d", neo4jHost, neo4jPort)

		graphBackend, err = graph.NewNeo4jBackend(
			ctx,
			neo4jURI,
			config.MustGetString("NEO4J_USER"),
			config.MustGetString("NEO4J_PASSWORD"),
		)
		if err != nil {
			return fmt.Errorf("Neo4j connection failed: %w", err)
		}
	case "neptune":
		return fmt.Errorf("Neptune backend not yet implemented")
	default:
		return fmt.Errorf("unsupported backend: %s (use 'neo4j' or 'neptune')", backendType)
	}
	defer graphBackend.Close()

	fmt.Printf("  ✓ Connected to PostgreSQL\n")
	fmt.Printf("  ✓ Connected to %s\n", backendType)

	// Stage 1: Fetch GitHub data → PostgreSQL
	fmt.Printf("\n[1/3] Fetching GitHub API data...\n")
	fetchStart := time.Now()

	fetcher := github.NewFetcher(config.MustGetString("GITHUB_TOKEN"), stagingDB)
	repoID, stats, err := fetcher.FetchAll(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}

	fetchDuration := time.Since(fetchStart)
	fmt.Printf("  ✓ Fetched in %v\n", fetchDuration)
	fmt.Printf("    Commits: %d | Issues: %d | PRs: %d | Branches: %d\n",
		stats.Commits, stats.Issues, stats.PRs, stats.Branches)

	// Stage 2: Build graph from PostgreSQL → Neo4j/Neptune
	fmt.Printf("\n[2/3] Building knowledge graph...\n")
	graphStart := time.Now()

	builder := graph.NewBuilder(stagingDB, graphBackend)
	buildStats, err := builder.BuildGraph(ctx, repoID)
	if err != nil {
		return fmt.Errorf("graph construction failed: %w", err)
	}

	graphDuration := time.Since(graphStart)
	fmt.Printf("  ✓ Graph built in %v\n", graphDuration)
	fmt.Printf("    Nodes: %d | Edges: %d\n", buildStats.Nodes, buildStats.Edges)

	// Stage 3: Validate
	fmt.Printf("\n[3/3] Validating...\n")
	if err := validateGraph(ctx, graphBackend, buildStats); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	fmt.Printf("  ✓ Validation passed\n")

	// Summary
	totalDuration := time.Since(startTime)
	fmt.Printf("\n✅ CodeRisk initialized for %s/%s\n", owner, repo)
	fmt.Printf("   Total time: %v (fetch: %v, graph: %v)\n",
		totalDuration, fetchDuration, graphDuration)
	fmt.Printf("\n💡 Try: crisk check <file>\n")

	return nil
}

// Helper functions

// parseRepoName splits "owner/repo" into components
func parseRepoName(repoName string) (owner, repo string, err error) {
	parts := strings.Split(repoName, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("expected format: owner/repo, got: %s", repoName)
	}
	return parts[0], parts[1], nil
}

// validateGraph performs basic sanity checks on the constructed graph
func validateGraph(ctx context.Context, backend graph.Backend, stats *graph.BuildStats) error {
	if stats.Nodes == 0 {
		return fmt.Errorf("no nodes created in graph")
	}
	if stats.Edges == 0 {
		return fmt.Errorf("no edges created in graph")
	}

	// Verify required node types exist
	requiredLabels := []string{"Commit", "Developer"}

	for _, label := range requiredLabels {
		query := fmt.Sprintf("MATCH (n:%s) RETURN count(n) as count", label)
		result, err := backend.Query(query)
		if err != nil {
			return fmt.Errorf("failed to query %s nodes: %w", label, err)
		}

		count := result.(int64)
		if count == 0 {
			fmt.Printf("  ⚠️  Warning: No %s nodes found\n", label)
		}
	}

	return nil
}
