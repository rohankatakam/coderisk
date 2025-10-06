package main

import (
	"context"
	"fmt"
	"time"

	"github.com/coderisk/coderisk-go/internal/config"
	"github.com/coderisk/coderisk-go/internal/graph"
	"github.com/coderisk/coderisk-go/internal/ingestion"
	"github.com/spf13/cobra"
)

var (
	parseBackendType string
	parseWorkers     int
)

var parseCmd = &cobra.Command{
	Use:   "parse [repository]",
	Short: "Parse repository source code and build Layer 1 graph",
	Long: `Parse source code using Tree-sitter AST parsing to extract code structure.

This command:
1. Clones the repository (if not already cloned)
2. Discovers source files (JavaScript/TypeScript)
3. Parses files and extracts functions, classes, imports
4. Builds Layer 1 graph (File, Function, Class nodes + CALLS, IMPORTS edges)

Examples:
  crisk parse omnara-ai/omnara
  crisk parse https://github.com/omnara-ai/omnara
  crisk parse omnara-ai/omnara --backend neo4j
  crisk parse omnara-ai/omnara --workers 40

Repository Storage:
  Repositories are cloned to ~/.coderisk/repos/<hash>/ for reuse.

Performance:
  - omnara-ai/omnara (~1K files): ~10s
  - kubernetes/kubernetes (~50K files): ~5min`,
	Args: cobra.ExactArgs(1),
	RunE: runParse,
}

func init() {
	parseCmd.Flags().StringVarP(&parseBackendType, "backend", "b", "neo4j", "Graph backend: neo4j (local) or neptune (cloud)")
	parseCmd.Flags().IntVarP(&parseWorkers, "workers", "w", 20, "Number of concurrent parsers")
}

func runParse(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := context.Background()

	// Load environment variables from .env file
	envLoader := config.NewEnvLoader()
	envLoader.MustLoad()

	// Parse repository argument (supports org/repo or full URL)
	repoArg := args[0]
	repoURL, err := normalizeRepoURL(repoArg)
	if err != nil {
		return fmt.Errorf("invalid repository: %w", err)
	}

	fmt.Printf("🚀 CodeRisk Layer 1: AST Parsing\n")
	fmt.Printf("Repository: %s\n", repoURL)
	fmt.Printf("Backend: %s\n", parseBackendType)
	fmt.Printf("Workers: %d\n\n", parseWorkers)

	// Initialize graph backend
	var graphClient graph.Backend
	switch parseBackendType {
	case "neo4j":
		neo4jClient, err := initializeNeo4jBackend(ctx)
		if err != nil {
			return fmt.Errorf("failed to initialize Neo4j: %w", err)
		}
		defer neo4jClient.Close()
		graphClient = neo4jClient

	case "neptune":
		return fmt.Errorf("Neptune backend not yet implemented for Layer 1")

	default:
		return fmt.Errorf("unsupported backend: %s", parseBackendType)
	}

	// Create processor
	processorConfig := &ingestion.ProcessorConfig{
		Workers:    parseWorkers,
		Timeout:    30 * time.Second,
		GraphBatch: 100,
	}
	var graphBuilder *graph.Builder
	if graphClient != nil {
		graphBuilder = graph.NewBuilder(nil, graphClient)
	}
	processor := ingestion.NewProcessor(processorConfig, graphClient, graphBuilder)

	// Process repository
	fmt.Printf("⏳ Processing repository...\n\n")
	result, err := processor.ProcessRepository(ctx, repoURL)
	if err != nil {
		return fmt.Errorf("processing failed: %w", err)
	}

	// Print results
	fmt.Printf("✅ Processing complete!\n\n")
	fmt.Printf("📊 Statistics:\n")
	fmt.Printf("  Repository:  %s\n", result.RepoPath)
	fmt.Printf("  Duration:    %v\n", result.Duration)
	fmt.Printf("  Files:       %d total (%d parsed, %d failed)\n",
		result.FilesTotal, result.FilesParsed, result.FilesFailed)
	fmt.Printf("  Entities:    %d total\n", result.EntitiesTotal)
	fmt.Printf("    Functions: %d\n", result.Functions)
	fmt.Printf("    Classes:   %d\n", result.Classes)
	fmt.Printf("    Imports:   %d\n\n", result.Imports)

	// Print errors if any
	if len(result.Errors) > 0 {
		fmt.Printf("⚠️  Warnings (%d):\n", len(result.Errors))
		for i, err := range result.Errors {
			if i < 10 { // Only show first 10 errors
				fmt.Printf("  - %v\n", err)
			}
		}
		if len(result.Errors) > 10 {
			fmt.Printf("  ... and %d more errors\n", len(result.Errors)-10)
		}
		fmt.Printf("\n")
	}

	// Print next steps
	fmt.Printf("🎯 Next Steps:\n")
	fmt.Printf("  1. Verify graph: cypher-shell -u neo4j -p coderisk123 \"MATCH (n) RETURN labels(n), count(n)\"\n")
	fmt.Printf("  2. Query functions: \"MATCH (f:Function) RETURN f.name LIMIT 10\"\n")
	fmt.Printf("  3. Check imports: \"MATCH (f:File)-[:IMPORTS]->(i:Import) RETURN f.file_path, i.import_path LIMIT 10\"\n")
	fmt.Printf("\n")

	fmt.Printf("⏱️  Total time: %v\n", time.Since(startTime))

	return nil
}

// normalizeRepoURL converts various input formats to full GitHub URL
func normalizeRepoURL(input string) (string, error) {
	// Already a full URL
	if len(input) > 8 && input[:8] == "https://" {
		return input, nil
	}

	// git@ format
	if len(input) > 4 && input[:4] == "git@" {
		// Convert to https
		org, repo, err := ingestion.ParseRepoURL(input)
		if err != nil {
			return "", err
		}
		return ingestion.BuildGitHubURL(org, repo), nil
	}

	// org/repo format
	org, repo, err := ingestion.ParseRepoURL(input)
	if err != nil {
		return "", err
	}
	return ingestion.BuildGitHubURL(org, repo), nil
}

// initializeNeo4jBackend creates Neo4j graph backend
func initializeNeo4jBackend(ctx context.Context) (*graph.Neo4jBackend, error) {
	neo4jURI := config.GetString("NEO4J_URI", "bolt://localhost:7687")
	neo4jUser := config.GetString("NEO4J_USER", "neo4j")
	neo4jPassword := config.GetString("NEO4J_PASSWORD", "coderisk123")

	neo4jClient, err := graph.NewNeo4jBackend(ctx, neo4jURI, neo4jUser, neo4jPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j client: %w", err)
	}

	return neo4jClient, nil
}
