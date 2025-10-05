package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/coderisk/coderisk-go/internal/ai"
	"github.com/coderisk/coderisk-go/internal/cache"
	"github.com/coderisk/coderisk-go/internal/database"
	"github.com/coderisk/coderisk-go/internal/git"
	"github.com/coderisk/coderisk-go/internal/graph"
	"github.com/coderisk/coderisk-go/internal/metrics"
	"github.com/coderisk/coderisk-go/internal/models"
	"github.com/coderisk/coderisk-go/internal/output"
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

	// Assess each file
	repoID := "local" // TODO: Get from git repo
	hasHighRisk := false

	for _, file := range files {
		result, err := registry.CalculatePhase1(ctx, repoID, file)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// Convert to RiskResult and format
		riskResult := output.ConvertPhase1ToRiskResult(result)

		// If AI mode, enhance with AI prompts and confidence scores
		if aiMode {
			enrichWithAIData(riskResult)
		}

		if err := formatter.Format(riskResult, os.Stdout); err != nil {
			return fmt.Errorf("formatting error: %w", err)
		}

		if result.ShouldEscalate {
			hasHighRisk = true
			if !preCommit {
				fmt.Println("\n⚠️  HIGH RISK - Would escalate to Phase 2 (LLM investigation)")
				fmt.Println("    Phase 2 requires LLM API key (set PHASE2_ENABLED=true)")
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
