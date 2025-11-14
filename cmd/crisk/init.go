package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/rohankatakam/coderisk/internal/auth"
	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/github"
	"github.com/rohankatakam/coderisk/internal/graph"
	"github.com/rohankatakam/coderisk/internal/linking"
	"github.com/rohankatakam/coderisk/internal/llm"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize CodeRisk analysis for current repository",
	Long: `Initialize CodeRisk by building the 100% confidence knowledge graph from GitHub data.

This command must be run inside a cloned GitHub repository.

What it does:
  • Fetch GitHub commit history, ownership, and temporal data
  • Fetch GitHub issues and PRs
  • Build 100% confidence graph with REFERENCES and CLOSED_BY edges from timeline events

Usage:
  git clone https://github.com/owner/repo
  cd repo
  crisk init [--days N] [--llm]

Examples:
  cd ~/projects/my-repo
  crisk init                  # Default: last 90 days, 100% confidence graph only
  crisk init --days 30        # Last 30 days
  crisk init --days 180       # Last 180 days
  crisk init --all            # Full repository history
  crisk init --llm            # Include LLM-based ASSOCIATED_WITH extraction (requires API key)

Requirements:
  • Must be run inside a cloned GitHub repository
  • GitHub Personal Access Token (for fetching issues, commits)
  • LLM API key (optional, only needed with --llm flag)
  • Docker with Neo4j and PostgreSQL running

Configuration:
  Development: Use .env file (copy from .env.example)
  Production: Run 'crisk login' for cloud authentication`,
	Args: cobra.NoArgs,
	RunE: runInit,
}

func init() {
	// Add time window flags per IMPLEMENTATION_GAP_ANALYSIS.md
	initCmd.Flags().Int("days", 90, "Ingest PRs merged in last N days (default: 90)")
	initCmd.Flags().Bool("all", false, "Ingest entire repository history")
	initCmd.Flags().Bool("llm", false, "Enable LLM-based ASSOCIATED_WITH edge extraction (requires API key)")
}

// detectCurrentRepo detects the git repository in the current directory
func detectCurrentRepo() (owner, repo, repoPath string, err error) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find git root directory
	gitRoot := cwd
	for {
		if _, err := os.Stat(filepath.Join(gitRoot, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(gitRoot)
		if parent == gitRoot {
			return "", "", "", fmt.Errorf("not a git repository\n\nRun this command inside a cloned git repository, or specify:\n  crisk init owner/repo")
		}
		gitRoot = parent
	}

	// Get git remote URL
	cmd := exec.Command("git", "-C", gitRoot, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get git remote: %w\n\nMake sure the repository has an 'origin' remote set.", err)
	}

	remoteURL := strings.TrimSpace(string(output))

	// Parse owner/repo from remote URL
	// Supports: https://github.com/owner/repo.git or git@github.com:owner/repo.git
	re := regexp.MustCompile(`github\.com[:/]([^/]+)/(.+?)(?:\.git)?$`)
	matches := re.FindStringSubmatch(remoteURL)
	if matches == nil || len(matches) < 3 {
		return "", "", "", fmt.Errorf("could not parse GitHub owner/repo from remote URL: %s\n\nRemote URL must be a GitHub repository.", remoteURL)
	}

	owner = matches[1]
	repo = matches[2]
	repoPath = gitRoot

	return owner, repo, repoPath, nil
}

func runInit(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := context.Background()

	// Detect deployment mode
	mode := config.DetectMode()

	var githubToken, geminiAPIKey string
	var authManager *auth.Manager

	// Production mode: REQUIRE cloud authentication
	if !mode.AllowsDevelopmentDefaults() {
		// Running from packaged binary (brew install, etc.)
		// MUST authenticate with cloud
		authManager, err := auth.NewManager()
		if err != nil {
			return fmt.Errorf("failed to initialize authentication: %w", err)
		}

		if err := authManager.LoadSession(); err != nil {
			return fmt.Errorf("❌ Not authenticated.\n\n" +
				"To use CodeRisk, you must authenticate:\n" +
				"  → Run: crisk login\n\n" +
				"Visit https://coderisk.dev to create an account.\n")
		}

		// Fetch credentials from cloud
		fmt.Println("🔐 Fetching credentials from CodeRisk Cloud...")
		creds, err := authManager.GetCredentials()
		if err != nil {
			return fmt.Errorf("failed to fetch cloud credentials: %w\n\nTry running 'crisk logout' then 'crisk login' again.", err)
		}

		if creds.GitHubToken == "" {
			return fmt.Errorf("❌ Missing GitHub token.\n\n" +
				"Please configure your API keys at:\n" +
				"  → https://coderisk.dev/dashboard/settings\n")
		}

		githubToken = creds.GitHubToken
		// LLM key is optional - llm.NewClient will use Gemini by default if configured
		geminiAPIKey = creds.OpenAIAPIKey // Legacy: repurpose OpenAI field for Gemini
		fmt.Println("  ✓ Using credentials from cloud")

	} else {
		// Development mode: Allow .env file
		fmt.Printf("🔧 Development mode detected (%s)\n", mode.Description())

		// Load environment variables from .env file
		envLoader := config.NewEnvLoader()
		if err := envLoader.Load(); err != nil {
			return fmt.Errorf("failed to load .env file: %w\n\nCopy .env.example to .env and configure your credentials.", err)
		}

		githubToken = os.Getenv("GITHUB_TOKEN")
		geminiAPIKey = os.Getenv("GEMINI_API_KEY")
		if geminiAPIKey == "" {
			geminiAPIKey = os.Getenv("OPENAI_API_KEY") // Fallback for legacy configs
		}

		if githubToken == "" {
			return fmt.Errorf("GITHUB_TOKEN not set in .env file")
		}
		// LLM API key is optional - Phase 2 will be disabled if not provided

		fmt.Println("  ✓ Using credentials from .env file")

		// Validate required variables
		if err := envLoader.ValidateWithGitHub(); err != nil {
			return err
		}
	}

	// Set environment variables for downstream code that expects them
	os.Setenv("GITHUB_TOKEN", githubToken)
	os.Setenv("GEMINI_API_KEY", geminiAPIKey)

	// Detect current repository
	fmt.Println("📁 Detecting repository from current directory...")
	owner, repo, repoPath, err := detectCurrentRepo()
	if err != nil {
		return err
	}
	fmt.Printf("  ✓ Detected: %s/%s\n", owner, repo)
	fmt.Printf("  ✓ Path: %s\n", repoPath)

	fmt.Printf("\n🚀 Initializing CodeRisk for %s/%s...\n", owner, repo)
	fmt.Printf("   Backend: Neo4j (local)\n")
	fmt.Printf("   Mode: %s\n", mode.Description())

	// Load and validate configuration
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	result := cfg.ValidateWithMode(config.ValidationContextInit, mode)
	if result.HasErrors() {
		return fmt.Errorf("configuration validation failed:\n%s", result.Error())
	}

	// Connect to PostgreSQL
	fmt.Printf("\n[0/5] Connecting to databases...\n")
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

	// Connect to Neo4j (only supported backend for MVP)
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

	// Use the already-cloned repository
	fmt.Printf("\n[1/4] Using repository at %s...\n", repoPath)
	fmt.Printf("  ✓ Skipping clone (using existing repository)\n")

	// Fetch GitHub data → PostgreSQL
	fmt.Printf("\n[2/4] Fetching GitHub API data...\n")
	fetchStart := time.Now()

	// Get time window flags
	days, _ := cmd.Flags().GetInt("days")
	allHistory, _ := cmd.Flags().GetBool("all")

	// If --all flag is set, use 0 days (meaning no time limit)
	if allHistory {
		days = 0
		fmt.Printf("  ℹ️  Fetching entire repository history (no time limit)\n")
	} else {
		fmt.Printf("  ℹ️  Fetching last %d days of history\n", days)
	}

	fetcher := github.NewFetcher(config.MustGetString("GITHUB_TOKEN"), stagingDB)
	repoID, stats, err := fetcher.FetchAll(ctx, owner, repo, repoPath, days)
	if err != nil {
		return fmt.Errorf("fetch failed: %w", err)
	}

	fetchDuration := time.Since(fetchStart)
	fmt.Printf("  ✓ Fetched in %v\n", fetchDuration)
	fmt.Printf("    Commits: %d | Issues: %d | PRs: %d | Branches: %d\n",
		stats.Commits, stats.Issues, stats.PRs, stats.Branches)

	// Check if --llm flag is enabled
	enableLLM, _ := cmd.Flags().GetBool("llm")

	// Stage: Extract issue-commit-PR relationships using LLM (only if --llm flag is set)
	if enableLLM {
		fmt.Printf("\n[3/4] Extracting issue-commit-PR relationships (LLM analysis)...\n")
		extractStart := time.Now()

		// Create LLM client
		llmClient, err := llm.NewClient(ctx, cfg)
		if err != nil {
			fmt.Printf("  ⚠️  LLM client creation failed: %v\n", err)
			fmt.Printf("  → Continuing without LLM extraction\n")
		} else if llmClient.IsEnabled() {
			// Create extractors
			issueExtractor := github.NewIssueExtractor(llmClient, stagingDB)
			commitExtractor := github.NewCommitExtractor(llmClient, stagingDB)

			// Extract from Issues
			issueRefs, err := issueExtractor.ExtractReferences(ctx, repoID)
			if err != nil {
				fmt.Printf("  ⚠️  Issue extraction failed: %v\n", err)
			} else {
				fmt.Printf("  ✓ Extracted %d references from issues\n", issueRefs)
			}

			// Extract from Commits
			commitRefs, err := commitExtractor.ExtractCommitReferences(ctx, repoID)
			if err != nil {
				fmt.Printf("  ⚠️  Commit extraction failed: %v\n", err)
			} else {
				fmt.Printf("  ✓ Extracted %d references from commits\n", commitRefs)
			}

			// Extract from PRs
			prRefs, err := commitExtractor.ExtractPRReferences(ctx, repoID)
			if err != nil {
				fmt.Printf("  ⚠️  PR extraction failed: %v\n", err)
			} else {
				fmt.Printf("  ✓ Extracted %d references from PRs\n", prRefs)
			}

			extractDuration := time.Since(extractStart)
			totalRefs := issueRefs + commitRefs + prRefs
			fmt.Printf("  ✓ Extracted %d total references in %v\n", totalRefs, extractDuration)

			// Stage: Issue-PR Linking
			fmt.Printf("\n  Linking issues to pull requests...\n")
			linkStart := time.Now()

			// Create linking orchestrator with the time window
			orchestrator := linking.NewOrchestrator(stagingDB, llmClient, repoID, days)

			// Run multi-phase linking pipeline
			if err := orchestrator.Run(ctx); err != nil {
				fmt.Printf("  ⚠️  Issue-PR linking failed: %v\n", err)
				fmt.Printf("  → Graph will use fallback linking methods\n")
			} else {
				linkDuration := time.Since(linkStart)
				fmt.Printf("  ✓ Issue-PR linking completed in %v\n", linkDuration)
				fmt.Printf("    (Links will be loaded into graph during graph construction)\n")
			}
		} else {
			fmt.Printf("  ⚠️  LLM extraction skipped (API key not configured)\n")
		}
	} else {
		fmt.Printf("\n[3/4] LLM extraction skipped (use --llm flag to enable)\n")
	}

	// Stage: Build graph from PostgreSQL → Neo4j (100% confidence graph)
	fmt.Printf("\n[4/4] Building 100%% confidence graph from GitHub data...\n")
	graphStart := time.Now()

	// Create graph builder
	graphBuilder := graph.NewBuilder(stagingDB, graphBackend)

	// Pass repoPath to resolve file paths from GitHub commits
	buildStats, err := graphBuilder.BuildGraph(ctx, repoID, repoPath)
	if err != nil {
		return fmt.Errorf("graph construction failed: %w", err)
	}

	graphDuration := time.Since(graphStart)
	fmt.Printf("  ✓ Graph built in %v\n", graphDuration)
	fmt.Printf("    Nodes: %d | Edges: %d\n", buildStats.Nodes, buildStats.Edges)

	// Stage: Validate graph
	fmt.Printf("\n  Validating graph structure...\n")
	if err := validateGraph(ctx, graphBackend, buildStats); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	fmt.Printf("  ✓ Graph validated successfully\n")

	// Stage: Apply indexes for optimal query performance
	fmt.Printf("\n  Creating database indexes...\n")
	indexStart := time.Now()

	if err := createIndexes(ctx, graphBackend); err != nil {
		// Non-fatal: indexes improve performance but aren't required
		fmt.Printf("  ⚠️  Index creation failed (non-fatal): %v\n", err)
	} else {
		indexDuration := time.Since(indexStart)
		fmt.Printf("  ✓ Indexes created in %v\n", indexDuration)
	}

	// Summary
	totalDuration := time.Since(startTime)

	// Query Neo4j for actual Developer count (not rough estimate)
	developerCount := int64(0)
	if result, err := graphBackend.Query(ctx, "MATCH (d:Developer) RETURN count(d) as count"); err == nil {
		developerCount = result.(int64)
	}

	fmt.Printf("\n✅ CodeRisk initialized for %s/%s (100%% Confidence Graph)\n", owner, repo)
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("   Total time: %v\n", totalDuration)
	fmt.Printf("   GitHub Data: %d commits, %d issues, %d PRs\n",
		stats.Commits, stats.Issues, stats.PRs)
	fmt.Printf("   Graph: %d nodes, %d edges (%d developers)\n",
		buildStats.Nodes, buildStats.Edges, developerCount)
	if enableLLM {
		fmt.Printf("   LLM Extraction: Enabled\n")
	} else {
		fmt.Printf("   LLM Extraction: Disabled (use --llm to enable)\n")
	}
	fmt.Printf("\n🚀 Next steps:\n")
	fmt.Printf("   • Test: crisk check <file>\n")
	fmt.Printf("   • Browse graph: http://localhost:7475 (Neo4j Browser)\n")
	fmt.Printf("   • Credentials: %s / <from .env file>\n", cfg.Neo4j.User)

	// Post usage telemetry if authenticated
	if authManager != nil {
		estimatedCost := 0.0 // No LLM usage in init without --llm flag
		totalNodes := buildStats.Nodes

		if err := authManager.PostUsage("init", 0, totalNodes, 0, estimatedCost); err != nil {
			// Silently fail - telemetry shouldn't block the command
			fmt.Printf("   (telemetry skipped: %v)\n", err)
		}
	}

	return nil
}

// Helper functions

// createIndexes applies all necessary indexes for optimal query performance
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 1
func createIndexes(ctx context.Context, backend graph.Backend) error {
	indexes := []struct {
		name  string
		query string
	}{
		// ===== Multi-repo composite constraints (PRIMARY KEYS) =====
		// These ensure uniqueness within a repository (repo_id + unique property)
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

		// ===== Multi-repo composite indexes (PERFORMANCE) =====
		// These speed up lookups filtered by repo_id
		{
			"file_repo_path_idx",
			"CREATE INDEX file_repo_path_idx IF NOT EXISTS FOR (f:File) ON (f.repo_id, f.path)",
		},
		{
			"commit_repo_sha_idx",
			"CREATE INDEX commit_repo_sha_idx IF NOT EXISTS FOR (c:Commit) ON (c.repo_id, c.sha)",
		},
		{
			"commit_repo_date_idx",
			"CREATE INDEX commit_repo_date_idx IF NOT EXISTS FOR (c:Commit) ON (c.repo_id, c.committed_at)",
		},

		// ===== Global constraints (no repo_id) =====
		// Developer emails are shared across repositories
		{
			"developer_email_unique",
			"CREATE CONSTRAINT developer_email_unique IF NOT EXISTS FOR (d:Developer) REQUIRE d.email IS UNIQUE",
		},

		// ===== Deprecated node types (backward compatibility) =====
		{
			"function_unique_id_unique",
			"CREATE CONSTRAINT function_unique_id_unique IF NOT EXISTS FOR (f:Function) REQUIRE f.unique_id IS UNIQUE",
		},
		{
			"class_unique_id_unique",
			"CREATE CONSTRAINT class_unique_id_unique IF NOT EXISTS FOR (c:Class) REQUIRE c.unique_id IS UNIQUE",
		},
		{
			"incident_id_unique",
			"CREATE CONSTRAINT incident_id_unique IF NOT EXISTS FOR (i:Incident) REQUIRE i.id IS UNIQUE",
		},
	}

	for _, idx := range indexes {
		if _, err := backend.Query(ctx, idx.query); err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.name, err)
		}
	}

	return nil
}

// parseRepoName splits "owner/repo" into components
func parseRepoName(repoName string) (owner, repo string, err error) {
	parts := strings.Split(repoName, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("expected format: owner/repo, got: %s", repoName)
	}
	return parts[0], parts[1], nil
}

// validateGraph performs basic sanity checks on the 100% confidence graph
func validateGraph(ctx context.Context, backend graph.Backend, stats *graph.BuildStats) error {
	// Check graph stats
	if stats.Nodes == 0 {
		return fmt.Errorf("no nodes created in graph")
	}

	// Verify required node types exist
	requiredLabels := map[string]string{
		"File":      "GitHub commits",
		"Commit":    "GitHub commits",
		"Developer": "GitHub commits",
		"Issue":     "GitHub issues",
		"PR":        "GitHub pull requests",
	}

	fmt.Printf("  Checking node types:\n")
	for label, layer := range requiredLabels {
		query := fmt.Sprintf("MATCH (n:%s) RETURN count(n) as count", label)
		result, err := backend.Query(ctx, query)
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
