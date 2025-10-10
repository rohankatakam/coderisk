package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/coderisk/coderisk-go/internal/agent"
	"github.com/coderisk/coderisk-go/internal/ai"
	"github.com/coderisk/coderisk-go/internal/cache"
	"github.com/coderisk/coderisk-go/internal/database"
	"github.com/coderisk/coderisk-go/internal/git"
	"github.com/coderisk/coderisk-go/internal/graph"
	"github.com/coderisk/coderisk-go/internal/incidents"
	"github.com/coderisk/coderisk-go/internal/metrics"
	"github.com/coderisk/coderisk-go/internal/models"
	"github.com/coderisk/coderisk-go/internal/output"
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

	// Get pre-commit flag
	preCommit, _ := cmd.Flags().GetBool("pre-commit")

	// Get files to check (from args, staged files, or git status)
	var files []string
	var err error

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

	// Create metrics registry
	registry := metrics.NewRegistry(neo4jClient, redisClient, pgClient)

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
	repoID := "local" // TODO: Get from git repo
	hasHighRisk := false

	// Resolve relative paths to absolute paths in cloned repo
	// This is needed because the graph stores absolute paths from init-local
	resolvedFiles, err := resolveFilePaths(files)
	if err != nil {
		return fmt.Errorf("failed to resolve file paths: %w", err)
	}

	for i, file := range files {
		resolvedPath := resolvedFiles[i]
		result, err := registry.CalculatePhase1(ctx, repoID, resolvedPath)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// Convert to RiskResult and format
		riskResult := output.ConvertPhase1ToRiskResult(result)

		// If AI mode, enhance with AI prompts and confidence scores
		if aiMode {
			enrichWithAIData(riskResult)
			// Pass Phase 1 result to formatter for enhanced analysis
			if aiFormatter, ok := formatter.(*output.AIFormatter); ok {
				aiFormatter.SetPhase1Result(result)
			}
		}

		if err := formatter.Format(riskResult, os.Stdout); err != nil {
			return fmt.Errorf("formatting error: %w", err)
		}

		if result.ShouldEscalate {
			hasHighRisk = true

			// Get OpenAI API key from environment
			// 12-factor: Factor 3 - Configuration from environment
			apiKey := os.Getenv("OPENAI_API_KEY")

			if apiKey == "" {
				// No API key - show message and continue
				if !preCommit {
					fmt.Println("\n⚠️  HIGH RISK detected")
					fmt.Println("    Set OPENAI_API_KEY to enable Phase 2 LLM investigation")
					fmt.Println("    Example: export OPENAI_API_KEY=sk-...")
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
			llmClient, err := agent.NewLLMClient(apiKey)
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
				"phase1_risk", result.OverallRisk,
				"api_key_present", true)

			// Create investigation request
			invReq := agent.InvestigationRequest{
				FilePath:   file,
				ChangeType: "modify", // TODO: detect from git diff
				Baseline: agent.BaselineMetrics{
					CouplingScore:     getCouplingScore(result),
					CoChangeFrequency: getCoChangeScore(result),
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

	// In pre-commit mode, exit with code 1 to block commit on HIGH/CRITICAL risk
	// Reference: ux_pre_commit_hook.md - Exit code strategy
	if preCommit && hasHighRisk {
		os.Exit(1)
	}

	return nil
}

// initNeo4j creates Neo4j client from environment
func initNeo4j(ctx context.Context) (*graph.Client, error) {
	return graph.NewClient(
		ctx,
		getEnvOrDefault("NEO4J_URI", "bolt://localhost:7688"),
		getEnvOrDefault("NEO4J_USER", "neo4j"),
		getEnvOrDefault("NEO4J_PASSWORD", "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"),
	)
}

// initRedis creates Redis client from environment
func initRedis(ctx context.Context) (*cache.Client, error) {
	port, _ := strconv.Atoi(getEnvOrDefault("REDIS_PORT_EXTERNAL", "6380"))
	return cache.NewClient(
		ctx,
		getEnvOrDefault("REDIS_HOST", "localhost"),
		port,
		getEnvOrDefault("REDIS_PASSWORD", ""),
	)
}

// initPostgres creates PostgreSQL client from environment
func initPostgres(ctx context.Context) (*database.Client, error) {
	port, _ := strconv.Atoi(getEnvOrDefault("POSTGRES_PORT_EXTERNAL", "5433"))
	return database.NewClient(
		ctx,
		getEnvOrDefault("POSTGRES_HOST", "localhost"),
		port,
		getEnvOrDefault("POSTGRES_DB", "coderisk"),
		getEnvOrDefault("POSTGRES_USER", "coderisk"),
		getEnvOrDefault("POSTGRES_PASSWORD", "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"),
	)
}

// getEnvOrDefault retrieves environment variable with fallback
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// initPostgresSQLX creates a sqlx Postgres connection for incidents database
func initPostgresSQLX() (*sqlx.DB, error) {
	port, _ := strconv.Atoi(getEnvOrDefault("POSTGRES_PORT_EXTERNAL", "5433"))
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		getEnvOrDefault("POSTGRES_USER", "coderisk"),
		getEnvOrDefault("POSTGRES_PASSWORD", "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"),
		getEnvOrDefault("POSTGRES_HOST", "localhost"),
		port,
		getEnvOrDefault("POSTGRES_DB", "coderisk"),
	)

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
