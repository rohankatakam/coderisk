package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/rohankatakam/coderisk/internal/agent"
	"github.com/rohankatakam/coderisk/internal/ai"
	"github.com/rohankatakam/coderisk/internal/analysis/config"
	"github.com/rohankatakam/coderisk/internal/analysis/phase0"
	"github.com/rohankatakam/coderisk/internal/auth"
	"github.com/rohankatakam/coderisk/internal/cache"
	appconfig "github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/git"
	"github.com/rohankatakam/coderisk/internal/graph"
	"github.com/rohankatakam/coderisk/internal/incidents"
	"github.com/rohankatakam/coderisk/internal/metrics"
	"github.com/rohankatakam/coderisk/internal/models"
	"github.com/rohankatakam/coderisk/internal/output"
	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver for sqlx
	"github.com/jmoiron/sqlx"
	"github.com/spf13/cobra"
)

// checkCmd performs Phase 1 risk assessment on specified files
// Reference: risk_assessment_methodology.md §2 - Tier 1 Metrics
// 12-factor: Factor 1 - Natural language to tool calls (CLI → metric calculation)
var checkCmd = &cobra.Command{
	Use:   "check [file...]",
	Short: "Assess risk for changed files using Phase 1 baseline metrics",
	Long: `Runs Phase 1 baseline assessment using Tier 1 metrics:
  - Structural Coupling (dependency count)
  - Temporal Co-Change (commit patterns)
  - Test Coverage Ratio

Completes in <500ms (no LLM needed for low-risk files).
Reference: risk_assessment_methodology.md §2`,
	RunE: runCheck,
}

func init() {
	checkCmd.Flags().Bool("quiet", false, "Output one-line summary (for pre-commit hooks)")
	checkCmd.Flags().Bool("explain", false, "Show full investigation trace")
	checkCmd.Flags().Bool("ai-mode", false, "Output machine-readable JSON for AI assistants")
	checkCmd.Flags().Bool("pre-commit", false, "Run in pre-commit hook mode (checks staged files)")

	// Mutually exclusive flags
	checkCmd.MarkFlagsMutuallyExclusive("quiet", "explain", "ai-mode")
}

func runCheck(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Detect deployment mode
	mode := appconfig.DetectMode()

	var openaiAPIKey string
	var authManager *auth.Manager
	var err error

	// Production mode: REQUIRE cloud authentication
	if !mode.AllowsDevelopmentDefaults() {
		// Running from packaged binary - must use cloud auth
		authManager, err = auth.NewManager()
		if err != nil {
			return fmt.Errorf("failed to initialize authentication: %w", err)
		}

		if err := authManager.LoadSession(); err != nil {
			return fmt.Errorf("❌ Not authenticated.\n\nRun: crisk login\n")
		}

		// Fetch credentials from cloud
		creds, err := authManager.GetCredentials()
		if err != nil {
			return fmt.Errorf("failed to fetch credentials: %w", err)
		}

		if creds.OpenAIAPIKey == "" {
			return fmt.Errorf("OpenAI API key not configured.\nVisit: https://coderisk.dev/dashboard/settings")
		}

		openaiAPIKey = creds.OpenAIAPIKey

	} else {
		// Development mode: Allow .env file
		openaiAPIKey = os.Getenv("OPENAI_API_KEY")

		// Try to load auth manager for telemetry (optional in dev mode)
		authManager, _ = auth.NewManager()
		if authManager != nil {
			authManager.LoadSession() // Ignore errors in dev mode
		}
	}

	// Get pre-commit flag
	preCommit, _ := cmd.Flags().GetBool("pre-commit")

	// Get files to check (from args, staged files, or git status)
	var files []string

	if preCommit {
		// Pre-commit mode: check staged files
		files, err = git.GetStagedFiles()
		if err != nil {
			return fmt.Errorf("failed to get staged files: %w", err)
		}

		if len(files) == 0 {
			fmt.Println("✅ No files to check")
			return nil
		}
	} else if len(args) > 0 {
		// Files specified as arguments
		files = args
	} else {
		// Auto-detect from git status
		files, err = git.GetChangedFiles()
		if err != nil {
			return fmt.Errorf("failed to get changed files: %w", err)
		}

		if len(files) == 0 {
			fmt.Println("✅ No changed files to check")
			return nil
		}
	}

	// Initialize dependencies from environment
	// Reference: DEVELOPMENT_WORKFLOW.md §3.3 - Load from environment variables
	neo4jClient, err := initNeo4j(ctx)
	if err != nil {
		return fmt.Errorf("neo4j initialization failed: %w", err)
	}
	defer neo4jClient.Close(ctx)

	redisClient, err := initRedis(ctx)
	if err != nil {
		return fmt.Errorf("redis initialization failed: %w", err)
	}
	defer redisClient.Close()

	pgClient, err := initPostgres(ctx)
	if err != nil {
		return fmt.Errorf("postgres initialization failed: %w", err)
	}
	defer pgClient.Close()

	// Create sqlx connection for incidents database (Phase 2)
	// Note: Using same connection config as pgClient but with sqlx for incidents API
	sqlxDB, err := initPostgresSQLX()
	if err != nil {
		return fmt.Errorf("sqlx postgres initialization failed: %w", err)
	}
	defer sqlxDB.Close()

	// Create incidents database for Phase 2
	incidentsDB := incidents.NewDatabase(sqlxDB)

	// Note: No longer using metrics.Registry - using adaptive config directly
	// registry := metrics.NewRegistry(neo4jClient, redisClient, pgClient)

	// Determine verbosity level
	var level output.VerbosityLevel
	quiet, _ := cmd.Flags().GetBool("quiet")
	explain, _ := cmd.Flags().GetBool("explain")
	aiMode, _ := cmd.Flags().GetBool("ai-mode")

	// Pre-commit mode implies quiet
	if preCommit {
		quiet = true
	}

	if quiet {
		level = output.VerbosityQuiet
	} else if explain {
		level = output.VerbosityExplain
	} else if aiMode {
		level = output.VerbosityAIMode
	} else {
		level = output.GetDefaultVerbosity()
	}

	// Create formatter
	formatter := output.NewFormatter(level)

	// If AI mode, configure formatter with additional context
	// 12-factor: Factor 4 - Tools are structured outputs
	if aiMode {
		if aiFormatter, ok := formatter.(*output.AIFormatter); ok {
			aiFormatter.SetGraphClient(neo4jClient)
		}
	}

	// Assess each file
	repoID, err := git.GetRepoID()
	if err != nil {
		// Fallback to "local" if not in a git repo
		repoID = "local"
		if !quiet {
			slog.Warn("could not detect repository ID, using 'local'", "error", err)
		}
	}
	hasHighRisk := false

	// Select adaptive configuration based on repository characteristics
	// Reference: ADR-005 §2 - Adaptive Configuration Selection
	repoMetadata := collectRepoMetadata()
	riskConfig, configReason := config.SelectConfigWithReason(repoMetadata)

	if !quiet && !preCommit {
		inferredDomain := config.InferDomain(repoMetadata)
		slog.Info("adaptive config selected",
			"config", riskConfig.ConfigKey,
			"language", repoMetadata.PrimaryLanguage,
			"domain", string(inferredDomain),
			"reason", configReason)
	}

	// Resolve relative paths to absolute paths in cloned repo
	// This is needed because the graph stores absolute paths from init-local
	resolvedFiles, err := resolveFilePaths(files)
	if err != nil {
		return fmt.Errorf("failed to resolve file paths: %w", err)
	}

	for i, file := range files {
		resolvedPath := resolvedFiles[i]

		// Phase 0: Pre-Analysis (security, docs, config detection)
		// Reference: ADR-005 §1 - Phase 0 Adaptive Pre-Analysis
		phase0Start := time.Now()

		// TODO: Get actual file diff for more accurate Phase 0 analysis
		// For now, Phase 0 works with just file path and extension
		phase0Result := phase0.RunPhase0(file, "")
		phase0Duration := time.Since(phase0Start)

		if !quiet && !preCommit {
			slog.Info("phase 0 complete",
				"file", file,
				"duration_us", phase0Duration.Microseconds(),
				"modification_types", phase0Result.ModificationTypes,
				"skip_analysis", phase0Result.SkipAnalysis,
				"force_escalate", phase0Result.ForceEscalate)
		}

		// If Phase 0 says skip all analysis (e.g., documentation-only)
		if phase0Result.SkipAnalysis {
			// Create simple result for documentation skip
			skipResult := &metrics.Phase1Result{
				FilePath:       file,
				OverallRisk:    metrics.RiskLevel(phase0Result.AggregatedRisk),
				ShouldEscalate: false,
				DurationMS:     phase0Duration.Milliseconds(),
			}
			riskResult := output.ConvertPhase1ToRiskResult(skipResult)

			if err := formatter.Format(riskResult, os.Stdout); err != nil {
				return fmt.Errorf("formatting error: %w", err)
			}
			continue
		}

		// Phase 1: Baseline Assessment with Adaptive Config
		// Reference: ADR-005 §2 - Adaptive Configuration Selection
		adaptiveResult, err := metrics.CalculatePhase1WithConfig(ctx, neo4jClient, redisClient, repoID, resolvedPath, riskConfig)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// If Phase 0 forces escalation, override Phase 1 decision
		if phase0Result.ForceEscalate {
			adaptiveResult.ShouldEscalate = true
			// Override risk level if Phase 0 detected higher risk
			if phase0Result.AggregatedRisk == "CRITICAL" || phase0Result.AggregatedRisk == "HIGH" {
				adaptiveResult.OverallRisk = metrics.RiskLevel(phase0Result.AggregatedRisk)
			}
		}

		// Convert to RiskResult and format
		riskResult := output.ConvertPhase1ToRiskResult(adaptiveResult.Phase1Result)

		// If AI mode, enhance with AI prompts and confidence scores
		if aiMode {
			enrichWithAIData(riskResult)
			// Pass Phase 1 result to formatter for enhanced analysis
			if aiFormatter, ok := formatter.(*output.AIFormatter); ok {
				aiFormatter.SetPhase1Result(adaptiveResult.Phase1Result)
			}
		}

		if err := formatter.Format(riskResult, os.Stdout); err != nil {
			return fmt.Errorf("formatting error: %w", err)
		}

		if adaptiveResult.ShouldEscalate {
			hasHighRisk = true

			// Use OpenAI API key from cloud or environment
			// 12-factor: Factor 3 - Configuration from environment
			if openaiAPIKey == "" {
				// No API key - show message and continue
				if !preCommit {
					fmt.Println("\n⚠️  HIGH RISK detected")
					fmt.Println("    Enable Phase 2 LLM investigation by either:")
					fmt.Println("      1. Running 'crisk login' to use cloud credentials")
					fmt.Println("      2. Setting OPENAI_API_KEY environment variable")
				}
				continue
			}

			// Phase 2: LLM Investigation
			// 12-factor: Factor 8 - Own your control flow (selective investigation)
			if !preCommit {
				fmt.Println("\n🔍 Escalating to Phase 2 (LLM investigation)...")
			}

			// Get repository path for temporal analysis
			repoPath := getRepoPath()

			// Create real clients for evidence collection
			temporalClient, err := agent.NewRealTemporalClient(repoPath)
			if err != nil {
				slog.Warn("temporal client creation failed", "error", err)
				temporalClient = nil // Continue without temporal data
			}

			// Create incidents client
			var incidentsClient *agent.RealIncidentsClient
			if incidentsDB != nil {
				incidentsClient = agent.NewRealIncidentsClient(incidentsDB)
			}

			// Create LLM client
			llmClient, err := agent.NewLLMClient(openaiAPIKey)
			if err != nil {
				fmt.Printf("❌ LLM client error: %v\n", err)
				continue
			}

			// Create investigator
			// 12-factor: Factor 10 - Small, focused agents (investigator is specialized)
			// Note: Passing nil for graph client as GetNeighbors not yet implemented in graph.Client
			investigator := agent.NewInvestigator(llmClient, temporalClient, incidentsClient, nil)

			// Build investigation request from Phase 1 result
			// Timeout increased to 60s to accommodate complex file analysis and API latency
			invCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()

			slog.Info("phase 2 escalation",
				"file", file,
				"phase1_risk", adaptiveResult.OverallRisk,
				"api_key_present", true)

			// Create investigation request
			invReq := agent.InvestigationRequest{
				FilePath:   file,
				ChangeType: "modify", // TODO: detect from git diff
				Baseline: agent.BaselineMetrics{
					CouplingScore:     getCouplingScore(adaptiveResult.Phase1Result),
					CoChangeFrequency: getCoChangeScore(adaptiveResult.Phase1Result),
					IncidentCount:     0, // TODO: extract from Phase 1
				},
			}

			// Run Phase 2 investigation
			assessment, err := investigator.Investigate(invCtx, invReq)
			if err != nil {
				fmt.Printf("⚠️  Investigation failed: %v\n", err)
				continue
			}

			slog.Info("phase 2 complete",
				"file", file,
				"final_risk", assessment.RiskLevel,
				"confidence", assessment.Confidence)

			// Display results based on verbosity mode
			if aiMode {
				// AI Mode: Include investigation trace in JSON
				output.DisplayPhase2JSON(assessment)
			} else if explain {
				// Explain Mode: Show full hop-by-hop trace
				output.DisplayPhase2Trace(assessment)
			} else {
				// Standard Mode: Show summary with recommendations
				output.DisplayPhase2Summary(assessment)
			}
		}
	}

	// Post usage telemetry if authenticated
	if authManager != nil {
		// TODO: Track actual OpenAI tokens used during Phase 2
		// For now, we'll estimate based on whether Phase 2 was triggered
		totalFiles := len(files)
		estimatedTokens := 0
		estimatedCost := 0.0

		if hasHighRisk && openaiAPIKey != "" {
			// Rough estimate: 1000 tokens per high-risk file with Phase 2
			estimatedTokens = 1000 * totalFiles
			// Estimate cost: $0.01 per 1000 tokens (rough GPT-4 pricing)
			estimatedCost = float64(estimatedTokens) * 0.00001
		}

		if err := authManager.PostUsage("check", totalFiles, 0, estimatedTokens, estimatedCost); err != nil {
			// Silently fail - telemetry shouldn't block the command
		}
	}

	// In pre-commit mode, exit with code 1 to block commit on HIGH/CRITICAL risk
	// Reference: ux_pre_commit_hook.md - Exit code strategy
	if preCommit && hasHighRisk {
		os.Exit(1)
	}

	return nil
}

// initNeo4j creates Neo4j client from config
func initNeo4j(ctx context.Context) (*graph.Client, error) {
	slog.Debug("initializing Neo4j connection")

	cfg, err := appconfig.Load("")
	if err != nil {
		slog.Error("failed to load config for Neo4j", "error", err)
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Validate Neo4j configuration
	mode := appconfig.DetectMode()
	slog.Debug("detected deployment mode", "mode", mode)

	result := cfg.ValidateWithMode(appconfig.ValidationContextCheck, mode)
	if result.HasErrors() {
		slog.Error("Neo4j configuration validation failed", "errors", result.Error())
		return nil, fmt.Errorf("configuration validation failed:\n%s", result.Error())
	}

	slog.Info("connecting to Neo4j", "uri", cfg.Neo4j.URI, "database", cfg.Neo4j.Database)

	client, err := graph.NewClient(
		ctx,
		cfg.Neo4j.URI,
		cfg.Neo4j.User,
		cfg.Neo4j.Password,
	)
	if err != nil {
		slog.Error("failed to connect to Neo4j", "error", err, "uri", cfg.Neo4j.URI)
		return nil, err
	}

	slog.Info("successfully connected to Neo4j")
	return client, nil
}

// initRedis creates Redis client from config
func initRedis(ctx context.Context) (*cache.Client, error) {
	slog.Debug("initializing Redis connection")

	cfg, err := appconfig.Load("")
	if err != nil {
		slog.Error("failed to load config for Redis", "error", err)
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Validate Redis configuration
	if cfg.Cache.RedisHost == "" {
		slog.Error("Redis configuration missing", "field", "REDIS_HOST")
		return nil, fmt.Errorf("REDIS_HOST is required in .env or environment variable")
	}

	if cfg.Cache.RedisPort == 0 {
		slog.Error("Redis configuration missing", "field", "REDIS_PORT_EXTERNAL")
		return nil, fmt.Errorf("REDIS_PORT_EXTERNAL is required in .env or environment variable")
	}

	slog.Info("connecting to Redis", "host", cfg.Cache.RedisHost, "port", cfg.Cache.RedisPort)

	client, err := cache.NewClient(ctx, cfg.Cache.RedisHost, cfg.Cache.RedisPort, cfg.Cache.RedisPassword)
	if err != nil {
		slog.Error("failed to connect to Redis", "error", err, "host", cfg.Cache.RedisHost, "port", cfg.Cache.RedisPort)
		return nil, err
	}

	slog.Info("successfully connected to Redis")
	return client, nil
}

// initPostgres creates PostgreSQL client from config
func initPostgres(ctx context.Context) (*database.Client, error) {
	slog.Debug("initializing PostgreSQL connection")

	cfg, err := appconfig.Load("")
	if err != nil {
		slog.Error("failed to load config for PostgreSQL", "error", err)
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Validate PostgreSQL configuration
	mode := appconfig.DetectMode()
	slog.Debug("detected deployment mode", "mode", mode)

	result := cfg.ValidateWithMode(appconfig.ValidationContextCheck, mode)
	if result.HasErrors() {
		slog.Error("PostgreSQL configuration validation failed", "errors", result.Error())
		return nil, fmt.Errorf("configuration validation failed:\n%s", result.Error())
	}

	slog.Info("connecting to PostgreSQL", "host", cfg.Storage.PostgresHost, "port", cfg.Storage.PostgresPort, "database", cfg.Storage.PostgresDB)

	// Use individual connection details from config
	// Config loading handles defaults and environment variable overrides
	client, err := database.NewClient(
		ctx,
		cfg.Storage.PostgresHost,
		cfg.Storage.PostgresPort,
		cfg.Storage.PostgresDB,
		cfg.Storage.PostgresUser,
		cfg.Storage.PostgresPassword,
	)
	if err != nil {
		slog.Error("failed to connect to PostgreSQL", "error", err, "host", cfg.Storage.PostgresHost, "port", cfg.Storage.PostgresPort)
		return nil, err
	}

	slog.Info("successfully connected to PostgreSQL")
	return client, nil
}

// initPostgresSQLX creates a sqlx Postgres connection for incidents database
func initPostgresSQLX() (*sqlx.DB, error) {
	cfg, err := appconfig.Load("")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Use DSN from config
	dsn := cfg.Storage.PostgresDSN
	if dsn == "" {
		return nil, fmt.Errorf("POSTGRES_DSN is required in .env or environment variable")
	}

	return sqlx.Connect("pgx", dsn)
}

// getRepoPath retrieves the git repository root path
func getRepoPath() string {
	// For now, use current working directory
	// TODO: Add git.GetRepoRoot() to find actual repo root
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}

// resolveFilePaths converts relative file paths to absolute paths in the cloned repo
// This matches the absolute paths stored in the graph during init-local
func resolveFilePaths(files []string) ([]string, error) {
	// Get the current git remote URL
	remoteURL, err := git.GetRemoteURL()
	if err != nil {
		// Not in a git repo - paths might already be absolute or this will fail later
		return files, nil
	}

	// Generate the repo hash (same logic as clone.go)
	repoHash := generateRepoHashForCheck(remoteURL)

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Build the cloned repo path
	clonedRepoPath := fmt.Sprintf("%s/.coderisk/repos/%s", homeDir, repoHash)

	// Resolve each file path
	resolved := make([]string, len(files))
	for i, file := range files {
		// Convert to absolute path in cloned repo
		resolved[i] = fmt.Sprintf("%s/%s", clonedRepoPath, file)
	}

	return resolved, nil
}

// generateRepoHashForCheck creates repo hash (duplicates logic from clone.go)
func generateRepoHashForCheck(url string) string {
	// Normalize URL
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimSuffix(url, "/")

	// Generate SHA256 hash
	h := sha256.New()
	h.Write([]byte(url))
	hashBytes := h.Sum(nil)

	// Use first 16 characters of hex
	return fmt.Sprintf("%x", hashBytes)[:16]
}

// getCouplingScore extracts coupling score from Phase 1 result
func getCouplingScore(result *metrics.Phase1Result) float64 {
	if result.Coupling == nil {
		return 0.0
	}
	// Normalize count to 0-1 score (max 20 dependencies = 1.0)
	score := float64(result.Coupling.Count) / 20.0
	if score > 1.0 {
		score = 1.0
	}
	return score
}

// getCoChangeScore extracts co-change frequency from Phase 1 result
func getCoChangeScore(result *metrics.Phase1Result) float64 {
	if result.CoChange == nil {
		return 0.0
	}
	// Use the max co-change frequency (already 0-1)
	return result.CoChange.MaxFrequency
}

// enrichWithAIData adds AI-specific fields to RiskResult for AI Mode
func enrichWithAIData(result *models.RiskResult) {
	promptGen := ai.NewPromptGenerator()
	confCalc := ai.NewConfidenceCalculator()

	// Enhance each issue with AI data
	for i := range result.Issues {
		issue := &result.Issues[i]

		// Find corresponding file
		var fileRisk models.FileRisk
		for _, f := range result.Files {
			if f.Path == issue.File {
				fileRisk = f
				break
			}
		}

		// Determine fix type if not set
		if issue.FixType == "" {
			issue.FixType = promptGen.DetermineFixType(*issue, fileRisk)
		}

		// Calculate confidence
		issue.FixConfidence = confCalc.Calculate(*issue, fileRisk)

		// Generate AI prompt
		issue.AIPromptTemplate = promptGen.GeneratePrompt(*issue, fileRisk)

		// Mark as auto-fixable if prompt exists and confidence is high
		issue.AutoFixable = issue.AIPromptTemplate != "" && confCalc.ShouldAutoFix(issue.FixConfidence)

		// Estimate fix time and lines
		complexity := 0.0
		if comp, ok := fileRisk.Metrics["complexity"]; ok {
			complexity = comp.Value
		}
		issue.EstimatedFixTimeMin = confCalc.EstimateFixTime(issue.FixType, complexity)
		issue.EstimatedLines = confCalc.EstimateLines(issue.FixType)

		// Determine expected files
		issue.ExpectedFiles = confCalc.DetermineExpectedFiles(issue.FixType, issue.File, fileRisk.Language)

		// Generate fix command
		issue.FixCommand = fmt.Sprintf("crisk fix-with-ai --%s %s:%d-%d", issue.FixType, issue.File, issue.LineStart, issue.LineEnd)
	}

	// Set commit control flags based on risk level
	switch result.RiskLevel {
	case "CRITICAL":
		result.ShouldBlock = true
		result.BlockReason = "critical_risk_detected"
		result.OverrideAllowed = false
		result.OverrideRequiresJustification = true
	case "HIGH":
		result.ShouldBlock = true
		result.BlockReason = "high_risk_detected"
		result.OverrideAllowed = true
		result.OverrideRequiresJustification = true
	case "MEDIUM":
		result.ShouldBlock = false
		result.BlockReason = ""
		result.OverrideAllowed = true
		result.OverrideRequiresJustification = false
	default:
		result.ShouldBlock = false
		result.OverrideAllowed = true
	}

	// Set performance metrics (simplified - would be calculated during actual analysis)
	result.Performance = models.Performance{
		TotalDurationMS: int(result.Duration.Milliseconds()),
		Breakdown: map[string]int{
			"phase1_metrics": int(result.Duration.Milliseconds()),
		},
		CacheEfficiency: map[string]interface{}{
			"cache_hit": result.CacheHit,
		},
	}
}

// collectRepoMetadata gathers repository characteristics for adaptive config selection
// Reference: ADR-005 §2 - Adaptive Configuration Selection
func collectRepoMetadata() config.RepoMetadata {
	repoPath := getRepoPath()

	// Detect primary language from file extensions
	language := detectPrimaryLanguage(repoPath)

	// Detect dependencies (as map for version tracking)
	depsMap := make(map[string]string)
	deps := detectDependencies(repoPath)
	for _, dep := range deps {
		depsMap[dep] = "detected" // Version info not available in simple scan
	}

	// Get directory names
	dirNames := getTopLevelDirectories(repoPath)

	// Read requirements.txt if exists
	var reqLines []string
	if data, err := os.ReadFile(repoPath + "/requirements.txt"); err == nil {
		reqLines = strings.Split(string(data), "\n")
	}

	// Read go.mod if exists
	var goModLines []string
	if data, err := os.ReadFile(repoPath + "/go.mod"); err == nil {
		goModLines = strings.Split(string(data), "\n")
	}

	return config.RepoMetadata{
		PrimaryLanguage: language,
		Dependencies:    depsMap,
		DirectoryNames:  dirNames,
		RequirementsTxt: reqLines,
		GoMod:           goModLines,
	}
}

// detectPrimaryLanguage determines the primary programming language
func detectPrimaryLanguage(repoPath string) string {
	// Check for language-specific files in priority order
	languageMarkers := map[string][]string{
		"python":     {"requirements.txt", "setup.py", "pyproject.toml", "Pipfile"},
		"go":         {"go.mod", "go.sum"},
		"javascript": {"package.json"},
		"typescript": {"tsconfig.json"},
		"rust":       {"Cargo.toml"},
		"java":       {"pom.xml", "build.gradle"},
	}

	for lang, markers := range languageMarkers {
		for _, marker := range markers {
			if _, err := os.Stat(repoPath + "/" + marker); err == nil {
				return lang
			}
		}
	}

	return "unknown"
}

// detectDependencies extracts framework and library information
func detectDependencies(repoPath string) []string {
	var deps []string

	// Python: read requirements.txt or pyproject.toml
	if data, err := os.ReadFile(repoPath + "/requirements.txt"); err == nil {
		content := string(data)
		if strings.Contains(content, "flask") || strings.Contains(content, "Flask") {
			deps = append(deps, "flask")
		}
		if strings.Contains(content, "django") || strings.Contains(content, "Django") {
			deps = append(deps, "django")
		}
		if strings.Contains(content, "fastapi") || strings.Contains(content, "FastAPI") {
			deps = append(deps, "fastapi")
		}
		if strings.Contains(content, "pandas") || strings.Contains(content, "numpy") {
			deps = append(deps, "pandas")
		}
	}

	// Go: read go.mod
	if data, err := os.ReadFile(repoPath + "/go.mod"); err == nil {
		content := string(data)
		if strings.Contains(content, "gin-gonic/gin") {
			deps = append(deps, "gin")
		}
		if strings.Contains(content, "labstack/echo") {
			deps = append(deps, "echo")
		}
	}

	// JavaScript/TypeScript: read package.json
	if data, err := os.ReadFile(repoPath + "/package.json"); err == nil {
		content := string(data)
		if strings.Contains(content, "\"react\"") {
			deps = append(deps, "react")
		}
		if strings.Contains(content, "\"vue\"") {
			deps = append(deps, "vue")
		}
		if strings.Contains(content, "\"express\"") {
			deps = append(deps, "express")
		}
	}

	return deps
}

// getTopLevelDirectories gets the top-level directory names in the repository
func getTopLevelDirectories(repoPath string) []string {
	var dirs []string

	entries, err := os.ReadDir(repoPath)
	if err != nil {
		return dirs
	}

	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			dirs = append(dirs, entry.Name())
		}
	}

	return dirs
}
