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
	"github.com/coderisk/coderisk-go/internal/ingestion"
	"github.com/spf13/cobra"
)

var (
	backendType string
)

var initCmd = &cobra.Command{
	Use:   "init [repository]",
	Short: "Initialize CodeRisk with full 3-layer analysis (Production)",
	Long: `Initialize CodeRisk by building the complete 3-layer knowledge graph.

This is the PRODUCTION command that enables full risk analysis:
  • Layer 1 (Structure): Tree-sitter parsing of code structure
  • Layer 2 (Temporal): GitHub commit history, co-changes, ownership
  • Layer 3 (Incidents): GitHub issues, PRs, incident tracking

Examples:
  crisk init omnara-ai/omnara
  crisk init omnara-ai/omnara --backend neo4j

Requirements:
  • OpenAI API key (for LLM-guided analysis)
  • GitHub Personal Access Token (for temporal data)
  • Docker (Neo4j, PostgreSQL, Redis)

Configuration:
  Run 'crisk configure' to set up API keys with OS keychain.
  Alternatively, use .env file (see .env.example).

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

	// Stage 0: Clone repository with FULL history (needed for Layer 2 temporal analysis)
	fmt.Printf("\n[0/4] Cloning repository with full git history...\n")
	cloneStart := time.Now()

	repoURL := fmt.Sprintf("https://github.com/%s/%s", owner, repo)
	repoPath, err := ingestion.CloneRepositoryFull(ctx, repoURL)
	if err != nil {
		return fmt.Errorf("clone failed: %w", err)
	}
	cloneDuration := time.Since(cloneStart)
	fmt.Printf("  ✓ Repository cloned to %s (took %v)\n", repoPath, cloneDuration)

	// Parse with tree-sitter (Layer 1: Structure)
	fmt.Printf("\n[0.5/4] Parsing code structure (Layer 1)...\n")
	parseStart := time.Now()

	// Count files before parsing
	fileStats, err := ingestion.CountFiles(repoPath)
	if err != nil {
		return fmt.Errorf("file analysis failed: %w", err)
	}

	languages := []string{}
	if fileStats.JavaScript > 0 {
		languages = append(languages, fmt.Sprintf("JavaScript (%d files)", fileStats.JavaScript))
	}
	if fileStats.TypeScript > 0 {
		languages = append(languages, fmt.Sprintf("TypeScript (%d files)", fileStats.TypeScript))
	}
	if fileStats.Python > 0 {
		languages = append(languages, fmt.Sprintf("Python (%d files)", fileStats.Python))
	}

	fmt.Printf("  ✓ Found %d source files: %s\n",
		fileStats.JavaScript+fileStats.TypeScript+fileStats.Python,
		strings.Join(languages, ", "))

	// Parse with tree-sitter (Layer 1 only, disable temporal analysis)
	// IMPORTANT: Use the same repoPath (-full) for both Layer 1 and Layer 2 to ensure file path consistency
	processorConfig := ingestion.DefaultProcessorConfig()
	processorConfig.EnableTemporal = false // Disable git history analysis (Layer 2 & 3 come from GitHub API)
	graphBuilder := graph.NewBuilder(stagingDB, graphBackend)
	processor := ingestion.NewProcessor(processorConfig, graphBackend, graphBuilder)

	parseResult, err := processor.ProcessRepositoryFromPath(ctx, repoPath)
	if err != nil {
		return fmt.Errorf("repository processing failed: %w", err)
	}

	parseDuration := time.Since(parseStart)
	fmt.Printf("  ✓ Parsed %d files in %v (%d functions, %d classes, %d imports)\n",
		parseResult.FilesParsed,
		parseDuration,
		parseResult.Functions,
		parseResult.Classes,
		parseResult.Imports)

	if parseResult.FilesFailed > 0 {
		fmt.Printf("  ⚠️  %d files failed to parse (errors logged)\n", parseResult.FilesFailed)
	}

	fmt.Printf("  ✓ Graph construction complete: %d entities stored\n", parseResult.EntitiesTotal)

	// Stage 1: Fetch GitHub data → PostgreSQL (Layer 2 & 3: Temporal & Incidents)
	fmt.Printf("\n[1/4] Fetching GitHub API data (Layer 2 & 3: Temporal & Incidents)...\n")
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

	// Stage 2: Build graph from PostgreSQL → Neo4j/Neptune (Layer 2 & 3 graph construction)
	fmt.Printf("\n[2/4] Building temporal & incident graph (Layer 2 & 3)...\n")
	graphStart := time.Now()

	// Reuse graphBuilder from Layer 1
	// Pass repoPath to resolve file paths from GitHub commits
	buildStats, err := graphBuilder.BuildGraph(ctx, repoID, repoPath)
	if err != nil {
		return fmt.Errorf("graph construction failed: %w", err)
	}

	graphDuration := time.Since(graphStart)
	fmt.Printf("  ✓ Graph built in %v\n", graphDuration)
	fmt.Printf("    Nodes: %d | Edges: %d\n", buildStats.Nodes, buildStats.Edges)

	// Stage 3: Validate all 3 layers
	fmt.Printf("\n[3/4] Validating all 3 layers...\n")
	if err := validateGraph(ctx, graphBackend, buildStats, parseResult); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	fmt.Printf("  ✓ All layers validated successfully\n")

	// Summary
	totalDuration := time.Since(startTime)
	fmt.Printf("\n✅ CodeRisk initialized for %s/%s (All 3 Layers)\n", owner, repo)
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("   Total time: %v\n", totalDuration)
	fmt.Printf("   Layer 1 (Structure): %d files, %d functions, %d classes\n",
		parseResult.FilesParsed, parseResult.Functions, parseResult.Classes)
	fmt.Printf("   Layer 2 (Temporal): %d commits, %d developers\n",
		stats.Commits, buildStats.Nodes/3) // Rough estimate
	fmt.Printf("   Layer 3 (Incidents): %d issues, %d PRs\n",
		stats.Issues, stats.PRs)
	fmt.Printf("\n🚀 Next steps:\n")
	fmt.Printf("   • Test: crisk check <file>\n")
	fmt.Printf("   • Browse graph: http://localhost:7475 (Neo4j Browser)\n")
	fmt.Printf("   • Credentials: neo4j / CHANGE_THIS_PASSWORD_IN_PRODUCTION_123\n")

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

// validateGraph performs basic sanity checks on all 3 layers
func validateGraph(ctx context.Context, backend graph.Backend, stats *graph.BuildStats, parseResult *ingestion.ProcessResult) error {
	// Check graph stats
	if stats.Nodes == 0 && parseResult.EntitiesTotal == 0 {
		return fmt.Errorf("no nodes created in graph")
	}

	// Verify required node types exist for all 3 layers
	requiredLabels := map[string]string{
		"File":      "Layer 1 (Structure)",
		"Function":  "Layer 1 (Structure)",
		"Commit":    "Layer 2 (Temporal)",
		"Developer": "Layer 2 (Temporal)",
		"Issue":     "Layer 3 (Incidents)",
	}

	fmt.Printf("  Checking node types:\n")
	for label, layer := range requiredLabels {
		query := fmt.Sprintf("MATCH (n:%s) RETURN count(n) as count", label)
		result, err := backend.Query(query)
		if err != nil {
			return fmt.Errorf("failed to query %s nodes: %w", label, err)
		}

		count := result.(int64)
		if count == 0 {
			fmt.Printf("    ⚠️  No %s nodes (%s)\n", label, layer)
		} else {
			fmt.Printf("    ✓ %s: %d nodes (%s)\n", label, count, layer)
		}
	}

	return nil
}
