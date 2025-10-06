package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coderisk/coderisk-go/internal/config"
	"github.com/coderisk/coderisk-go/internal/git"
	"github.com/coderisk/coderisk-go/internal/graph"
	"github.com/coderisk/coderisk-go/internal/ingestion"
	"github.com/spf13/cobra"
)

// initLocalCmd implements Week 1 Task 2: Local repository cloning + Tree-sitter parsing + Graph construction
// Reference: SESSION_5_PROMPT.md - Init Flow Orchestration
//
// Flow:
// Phase 1: Git Detection (auto-detect or --url)
// Phase 2: Layer 1 - Clone & AST Parse (Tree-sitter)
// Phase 3: Layer 2 - GitHub Data (optional, with --skip-github)
// Phase 4: Layer 3 - Graph Construction (optional, with --skip-graph)
// Phase 5: Completion Summary
var initLocalCmd = &cobra.Command{
	Use:   "init-local [url]",
	Short: "Initialize CodeRisk with local repository analysis (Week 1)",
	Long: `Initialize CodeRisk by cloning a repository, analyzing its structure with Tree-sitter,
and building the dependency graph in Neo4j.

Examples:
  crisk init-local                                    # Auto-detect from current git repo
  crisk init-local github.com/owner/repo              # Clone and analyze specific repo
  crisk init-local https://github.com/owner/repo.git  # Full GitHub URL
  crisk init-local --skip-github                      # Skip GitHub API fetching
  crisk init-local --skip-graph                       # Skip graph construction (parse only)

This implements the Week 1 Task 2 flow:
  1. Git Detection (auto or explicit URL)
  2. Repository Cloning (shallow clone)
  3. Tree-sitter Parsing (functions, classes, imports)
  4. Graph Construction (Neo4j)
`,
	RunE: runInitLocal,
}

func init() {
	initLocalCmd.Flags().String("url", "", "Repository URL (overrides auto-detection)")
	initLocalCmd.Flags().Bool("skip-github", false, "Skip GitHub API fetching (Layer 2)")
	initLocalCmd.Flags().Bool("skip-graph", false, "Skip graph construction (Layer 3)")
}

func runInitLocal(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := context.Background()

	// ========== Phase 1: Git Detection ==========
	var repoURL string
	urlFlag, _ := cmd.Flags().GetString("url")

	if urlFlag != "" {
		// Explicit URL from flag
		repoURL = urlFlag
	} else if len(args) > 0 {
		// URL from positional argument
		repoURL = args[0]
	} else {
		// Auto-detect from current directory
		remoteURL, err := git.GetRemoteURL()
		if err != nil {
			return fmt.Errorf("❌ Not a git repository. Use: crisk init-local <url>")
		}
		repoURL = remoteURL
	}

	// Parse repository URL to extract org/repo
	org, repo, err := git.ParseRepoURL(repoURL)
	if err != nil {
		return fmt.Errorf("❌ Invalid repository URL: %w", err)
	}

	fmt.Printf("✅ Repository detected: %s/%s\n", org, repo)
	fmt.Printf("   URL: %s\n\n", repoURL)

	// ========== Phase 2: Layer 1 - Clone & Parse ==========
	reportProgress("Cloning repository", "start")

	// Clone repository (or use cached copy)
	repoPath, err := ingestion.CloneRepository(ctx, repoURL)
	if err != nil {
		return fmt.Errorf("❌ Clone failed: %w", err)
	}

	reportProgress(fmt.Sprintf("Repository cloned to %s", repoPath), "done")

	// Count files before parsing
	reportProgress("Analyzing source files", "start")
	fileStats, err := ingestion.CountFiles(repoPath)
	if err != nil {
		return fmt.Errorf("❌ File analysis failed: %w", err)
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

	reportProgress(fmt.Sprintf("Found %d source files: %s",
		fileStats.JavaScript+fileStats.TypeScript+fileStats.Python,
		strings.Join(languages, ", ")), "done")

	// ========== Phase 3: Tree-sitter Parsing ==========
	skipGraph, _ := cmd.Flags().GetBool("skip-graph")
	var graphBackend graph.Backend

	if !skipGraph {
		// Connect to Neo4j for graph construction
		envLoader := config.NewEnvLoader()
		envLoader.MustLoad()

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
			return fmt.Errorf("❌ Neo4j connection failed: %w\n"+
				"   Hint: Start Neo4j with: docker-compose up -d", err)
		}
		defer graphBackend.Close()

		reportProgress("Connected to Neo4j", "done")
	}

	// Parse repository with Tree-sitter
	reportProgress("Parsing source code with Tree-sitter", "start")

	processorConfig := ingestion.DefaultProcessorConfig()
	var graphBuilder *graph.Builder
	if graphBackend != nil {
		// Create graph builder for Layer 2 temporal analysis
		graphBuilder = graph.NewBuilder(nil, graphBackend)
	}
	processor := ingestion.NewProcessor(processorConfig, graphBackend, graphBuilder)

	parseStart := time.Now()
	result, err := processor.ProcessRepository(ctx, repoURL)
	if err != nil {
		return fmt.Errorf("❌ Repository processing failed: %w", err)
	}
	parseDuration := time.Since(parseStart)

	reportProgress(fmt.Sprintf("Parsed %d files in %v (%d functions, %d classes, %d imports)",
		result.FilesParsed,
		parseDuration,
		result.Functions,
		result.Classes,
		result.Imports), "done")

	if result.FilesFailed > 0 {
		fmt.Printf("   ⚠️  %d files failed to parse (errors logged)\n", result.FilesFailed)
	}

	// ========== Phase 4: Layer 3 - Graph Construction ==========
	// (Already done by ProcessRepository if graphBackend != nil)
	if !skipGraph {
		reportProgress(fmt.Sprintf("Graph construction complete: %d entities stored",
			result.EntitiesTotal), "done")
	}

	// ========== Phase 5: Completion Summary ==========
	totalDuration := time.Since(startTime)

	fmt.Println("\n🎉 Initialization complete!\n")
	fmt.Println("📊 Statistics:")
	fmt.Printf("   Repository:   %s/%s\n", org, repo)
	fmt.Printf("   Files parsed: %d\n", result.FilesParsed)
	fmt.Printf("   Functions:    %d\n", result.Functions)
	fmt.Printf("   Classes:      %d\n", result.Classes)
	fmt.Printf("   Imports:      %d\n", result.Imports)
	fmt.Printf("   Total time:   %v (clone + parse: %v)\n", totalDuration, parseDuration)

	if !skipGraph {
		fmt.Println("\n🚀 Next steps:")
		fmt.Println("   • Run 'crisk check <file>' to analyze risk")
		fmt.Println("   • Install pre-commit hook: 'crisk hook install'")
	} else {
		fmt.Println("\n📝 Note: Graph construction skipped (--skip-graph)")
		fmt.Println("   Run without --skip-graph to enable risk analysis")
	}

	return nil
}

// reportProgress prints a progress message with emoji
func reportProgress(message string, status string) {
	if status == "start" {
		fmt.Printf("⏳ %s...\n", message)
	} else if status == "done" {
		fmt.Printf("✅ %s\n", message)
	}
}
