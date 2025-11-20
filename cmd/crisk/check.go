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
	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/auth"
	appconfig "github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/git"
	"github.com/rohankatakam/coderisk/internal/graph"
	"github.com/rohankatakam/coderisk/internal/incidents"
	"github.com/rohankatakam/coderisk/internal/llm"
	"github.com/rohankatakam/coderisk/internal/metrics"
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
	checkCmd.Flags().Bool("no-ai", false, "Skip Phase 2 LLM investigation (Phase 1 quantitative metrics only)")

	// Mutually exclusive flags
	checkCmd.MarkFlagsMutuallyExclusive("quiet", "explain", "ai-mode")
}

func runCheck(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Detect deployment mode
	mode := appconfig.DetectMode()

	var geminiAPIKey string
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
			return fmt.Errorf("LLM API key not configured.\nVisit: https://coderisk.dev/dashboard/settings")
		}

		// Use LLM API key (OpenAI field repurposed for LLM in cloud mode)
		geminiAPIKey = os.Getenv("GEMINI_API_KEY")
		if geminiAPIKey == "" {
			geminiAPIKey = creds.OpenAIAPIKey // Legacy: repurpose OpenAI field for Gemini
		}

	} else {
		// Development mode: Use Gemini from environment
		geminiAPIKey = os.Getenv("GEMINI_API_KEY")

		// Try to load auth manager for telemetry (optional in dev mode)
		authManager, _ = auth.NewManager()
		if authManager != nil {
			authManager.LoadSession() // Ignore errors in dev mode
		}
	}

	// Get pre-commit and no-ai flags
	preCommit, _ := cmd.Flags().GetBool("pre-commit")
	noAI, _ := cmd.Flags().GetBool("no-ai")

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

	pgClient, err := initPostgres(ctx)
	if err != nil {
		return fmt.Errorf("postgres initialization failed: %w", err)
	}
	defer pgClient.Close()

	// Create staging client for GitHub data queries
	stagingClient, err := initStagingClient(ctx)
	if err != nil {
		return fmt.Errorf("staging client initialization failed: %w", err)
	}
	defer stagingClient.Close()

	// Create sqlx connection for incidents database (Phase 2)
	// Note: Using same connection config as pgClient but with sqlx for incidents API
	sqlxDB, err := initPostgresSQLX()
	if err != nil {
		return fmt.Errorf("sqlx postgres initialization failed: %w", err)
	}
	defer sqlxDB.Close()

	// Create incidents database for Phase 2
	_ = incidents.NewDatabase(sqlxDB) // Not used in new agent system

	// Create hybrid client for combined Neo4j + Postgres queries
	hybridClient := database.NewHybridClient(neo4jClient, stagingClient)

	// Load CLQS score for the repository (for confidence display)
	// Note: repoID would need to be determined from the repository
	// For now, we'll load it later when we have the repoID
	var clqsScore *output.CLQSInfo

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

	// Load CLQS score for confidence display (if repository is in database)
	if repoID != "local" && stagingClient != nil {
		// Query database to get numeric repo ID
		dbRepoID, err := stagingClient.GetRepositoryID(ctx, repoID)
		if err == nil {
			// Load CLQS score (gracefully handles missing scores)
			clqsScore, _ = loadCLQSScore(ctx, stagingClient, dbRepoID)
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

	// Get repository root for file resolution
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("failed to get repository root: %w", err)
	}

	// Create file resolver to bridge current paths to historical graph data
	// Uses 2-level strategy: exact match (100% confidence) -> git log --follow (95% confidence)
	slog.Info("=== FILE RESOLUTION STAGE ===", "file_count", len(files))
	resolver := git.NewFileResolver(repoRoot, neo4jClient)

	// Batch resolve all files in parallel
	resolveStart := time.Now()
	resolvedFilesMap, err := resolver.BatchResolve(ctx, files)
	resolveDuration := time.Since(resolveStart)
	if err != nil {
		slog.Error("file resolution failed", "error", err, "duration", resolveDuration)
		return fmt.Errorf("file resolution failed: %w", err)
	}
	slog.Info("file resolution complete",
		"duration", resolveDuration,
		"files_resolved", len(resolvedFilesMap))

	for _, file := range files {
		matches := resolvedFilesMap[file]
		slog.Info("processing file", "file", file, "matches", len(matches))

		// Collect ALL historical paths for this file (current + renames)
		var queryPaths []string
		var resolutionMethod string
		var resolutionConfidence float64

		if len(matches) == 0 {
			// New file - no historical data in graph
			queryPaths = []string{file}
			resolutionMethod = "new-file"
			resolutionConfidence = 0.0

			if !quiet && !preCommit {
				fmt.Printf("\nℹ️  %s: New file (no historical data)\n", file)
			}
		} else {
			// Use ALL matched paths to capture full historical data
			for _, match := range matches {
				queryPaths = append(queryPaths, match.HistoricalPath)
			}

			// Report highest confidence match for logging
			resolutionMethod = matches[0].Method
			resolutionConfidence = matches[0].Confidence

			if !quiet && !preCommit && len(matches) > 1 {
				fmt.Printf("\n🔍 %s: Found %d historical paths (merging data)\n", file, len(matches))
				for _, match := range matches {
					fmt.Printf("   - %s (%s, %.0f%% confidence)\n", match.HistoricalPath, match.Method, match.Confidence*100)
				}
			}
		}

		// Phase 1: Baseline Assessment with Adaptive Config
		// CRITICAL: Pass ALL historical paths to capture full history across renames
		slog.Info("=== PHASE 1 METRICS STAGE ===",
			"file", file,
			"query_paths", queryPaths,
			"resolution_method", resolutionMethod,
			"resolution_confidence", resolutionConfidence)

		phase1Start := time.Now()
		adaptiveResult, err := metrics.CalculatePhase1WithMultiplePaths(ctx, neo4jClient, repoID, queryPaths, riskConfig)
		phase1Duration := time.Since(phase1Start)

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			slog.Error("phase 1 failed", "error", err, "duration", phase1Duration)
			continue
		}

		slog.Info("phase 1 complete",
			"duration", phase1Duration,
			"overall_risk", adaptiveResult.OverallRisk,
			"should_escalate", adaptiveResult.ShouldEscalate,
			"coupling_count", adaptiveResult.Phase1Result.Coupling.Count,
			"cochange_max_freq", adaptiveResult.Phase1Result.CoChange.MaxFrequency)

		// Phase 0 escalation removed - rely solely on Phase 1 metrics

		// Convert to RiskResult and format
		riskResult := output.ConvertPhase1ToRiskResult(adaptiveResult.Phase1Result)

		// If AI mode, pass Phase 1 result to formatter for enhanced analysis
		if aiMode {
			if aiFormatter, ok := formatter.(*output.AIFormatter); ok {
				aiFormatter.SetPhase1Result(adaptiveResult.Phase1Result)
			}
		}

		if err := formatter.Format(riskResult, os.Stdout); err != nil {
			return fmt.Errorf("formatting error: %w", err)
		}

		if adaptiveResult.ShouldEscalate {
			hasHighRisk = true

			// Skip Phase 2 if --no-ai flag is set
			if noAI {
				if !preCommit {
					fmt.Printf("\n⚠️  HIGH RISK detected (Phase 1 metrics only, skipping Phase 2 investigation)\n")
					fmt.Printf("    Risk Level: %s\n", adaptiveResult.OverallRisk)
					fmt.Printf("    💡 Remove --no-ai flag to enable detailed LLM investigation\n")
				}
				continue
			}

			// Use Gemini API key from cloud or environment
			// 12-factor: Factor 3 - Configuration from environment
			if geminiAPIKey == "" {
				// No API key - show message and continue
				if !preCommit {
					fmt.Println("\n⚠️  HIGH RISK detected")
					fmt.Println("    Enable Phase 2 LLM investigation by either:")
					fmt.Println("      1. Running 'crisk login' to use cloud credentials")
					fmt.Println("      2. Setting GEMINI_API_KEY environment variable")
					fmt.Println("      3. Using --no-ai flag to skip Phase 2")
				}
				continue
			}

			// Phase 2: Agent-Based Investigation with Complete Pipeline
			// 12-factor: Factor 8 - Own your control flow (selective investigation)
			if !preCommit {
				fmt.Println("\n🔍 Escalating to Phase 2 (Agent-based investigation)...")
			}

			slog.Info("=== PHASE 2 PIPELINE START ===",
				"file", file,
				"phase1_risk", adaptiveResult.OverallRisk)

			// STEP 1: Get git information
			slog.Info("STEP 1: Extracting git information")
			diff, diffErr := git.GetFileDiff(file)
			if diffErr != nil {
				slog.Warn("failed to get diff", "error", diffErr)
				diff = ""
			}
			linesAdded, linesDeleted := git.CountDiffLines(diff)
			changeStatus, statusErr := git.DetectChangeStatus(file)
			if statusErr != nil {
				slog.Warn("failed to detect change status", "error", statusErr)
				changeStatus = "MODIFIED"
			}
			slog.Info("STEP 1 complete",
				"lines_added", linesAdded,
				"lines_deleted", linesDeleted,
				"change_status", changeStatus,
				"diff_size", len(diff))

			// STEP 2: Build FileChangeContext
			slog.Info("STEP 2: Building file change context")
			fileContext := agent.FromPhase1Result(
				file,
				changeStatus,
				linesAdded,
				linesDeleted,
				matches,
				git.TruncateDiffForPrompt(diff, 500),
				adaptiveResult.Phase1Result,
			)
			slog.Info("STEP 2 complete",
				"resolution_matches", len(fileContext.ResolutionMatches),
				"coupling_score", fileContext.CouplingScore,
				"cochange_freq", fileContext.CoChangeFrequency)

			// STEP 3: Build kickoff prompt
			slog.Info("STEP 3: Building kickoff prompt")
			promptBuilder := agent.NewKickoffPromptBuilder([]agent.FileChangeContext{fileContext})

			// Add CLQS context if available
			if clqsScore != nil {
				agentCLQS := &agent.CLQSInfo{
					Score:              clqsScore.Score,
					Grade:              clqsScore.Grade,
					Rank:               clqsScore.Rank,
					LinkCoverage:       clqsScore.LinkCoverage,
					ConfidenceQuality:  clqsScore.ConfidenceQuality,
					EvidenceDiversity:  clqsScore.EvidenceDiversity,
					TemporalPrecision:  clqsScore.TemporalPrecision,
					SemanticStrength:   clqsScore.SemanticStrength,
				}
				promptBuilder.WithCLQS(agentCLQS)
			}

			kickoffPrompt := promptBuilder.BuildKickoffPrompt()
			slog.Info("STEP 3 complete",
				"prompt_length", len(kickoffPrompt),
				"estimated_tokens", len(kickoffPrompt)/4)

			// STEP 4: Create LLM client and investigator
			slog.Info("STEP 4: Creating LLM investigator")
			// Use Redis address from environment or default to docker-compose setup
			redisAddr := os.Getenv("REDIS_ADDR")
			if redisAddr == "" {
				redisAddr = "localhost:6380" // Default from docker-compose.yml
			}
			geminiClient, err := llm.NewGeminiClient(ctx, geminiAPIKey, "gemini-2.0-flash", redisAddr)
			if err != nil {
				fmt.Printf("❌ LLM client error: %v\n", err)
				slog.Error("failed to create LLM client", "error", err)
				continue
			}

			// Create Postgres adapter
			pgAdapter := agent.NewPostgresAdapter(stagingClient)

			// Create GeminiInvestigator with graph, postgres, and hybrid clients
			investigator := agent.NewGeminiInvestigator(geminiClient, neo4jClient, pgAdapter, hybridClient)
			slog.Info("STEP 4 complete", "investigator_created", true)

			// STEP 5: Run agent investigation
			slog.Info("STEP 5: Running agent investigation")
			invCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()

			investigationStart := time.Now()
			assessment, err := investigator.Investigate(invCtx, kickoffPrompt)
			investigationDuration := time.Since(investigationStart)

			if err != nil {
				// Graceful degradation: If Phase 2 fails, fall back to Phase 1 results
				// Reference: YC_DEMO_GAP_ANALYSIS.md Bug 3.1 - Graceful degradation
				errMsg := err.Error()
				isRateLimited := strings.Contains(errMsg, "RATE_LIMIT_EXHAUSTED") ||
								strings.Contains(errMsg, "429") ||
								strings.Contains(errMsg, "Resource exhausted")

				if isRateLimited {
					fmt.Printf("\n⚠️  Phase 2 investigation rate limited. Showing Phase 1 results:\n")
					fmt.Printf("    Risk Level: %s (based on quantitative metrics)\n", adaptiveResult.OverallRisk)
					fmt.Printf("    💡 Tip: Wait a few minutes and try again, or use --no-ai flag\n")
					slog.Warn("phase 2 rate limited, using phase 1 results",
						"duration", investigationDuration,
						"phase1_risk", adaptiveResult.OverallRisk)
				} else {
					fmt.Printf("⚠️  Investigation failed: %v\n", err)
					slog.Error("investigation failed", "error", err, "duration", investigationDuration)
				}
				continue
			}

			slog.Info("STEP 5 complete",
				"duration", investigationDuration,
				"final_risk", assessment.RiskLevel,
				"confidence", assessment.Confidence,
				"hops", len(assessment.Investigation.Hops),
				"total_tokens", assessment.Investigation.TotalTokens)

			// STEP 6: Check for directive decision points (12-factor agent: Factor #6 and #7)
			// This implements human-in-the-loop workflow for uncertain or high-risk cases
			slog.Info("STEP 6: Checking for directive decision points")

			// Check if any tools failed (indicates missing data)
			missingData := make(map[string]bool)
			// TODO: Track tool failures during investigation to detect missing co-change data
			// For now, we check if confidence is low which might indicate data issues

			directiveDecision := agent.CheckForDirectiveNeeded(assessment, file, missingData)

			if directiveDecision.ShouldPause {
				slog.Info("directive decision point identified",
					"type", directiveDecision.Directive.Action.Type,
					"reason", directiveDecision.Directive.Reasoning)

				// In non-interactive modes (preCommit, quiet), log but don't pause
				if !preCommit && !quiet {
					// Display directive to user
					agent.DisplayDirective(directiveDecision.Directive, directiveDecision.DecisionNum)

					// Prompt for user choice
					shortcuts := make([]string, len(directiveDecision.Directive.UserOptions))
					for i, opt := range directiveDecision.Directive.UserOptions {
						shortcuts[i] = opt.Shortcut
					}

					var userChoice string
					for {
						userChoice = agent.PromptForChoice("a", shortcuts)
						shouldContinue, shouldAbort := agent.HandleUserChoice(userChoice, directiveDecision.Directive)

						if shouldAbort {
							fmt.Println("\n❌ Investigation aborted by user")
							return nil
						}

						if shouldContinue {
							break
						}

						// Invalid choice - loop again
						fmt.Println("Invalid choice. Please try again.")
					}

					slog.Info("user decision recorded", "choice", userChoice)

					// If user chose to skip or abort high risk, warn them
					if (directiveDecision.Directive.Action.Type == agent.DirectiveTypeEscalate) && userChoice != "a" {
						fmt.Println("\n⚠️  WARNING: Proceeding with HIGH/CRITICAL risk change without addressing concerns")
					}
				}
			} else {
				slog.Info("no directive needed - investigation complete")
			}

			slog.Info("=== PHASE 2 PIPELINE COMPLETE ===",
				"file", file,
				"risk", assessment.RiskLevel,
				"total_duration", investigationDuration)

			// Display results based on verbosity mode
			if aiMode {
				// AI Mode: Include investigation trace in JSON
				output.DisplayPhase2JSON(*assessment)
			} else if explain {
				// Explain Mode: Show Manager + Developer views + detailed investigation trace
				// Query hybrid client for enriched context data
				incidents, _ := hybridClient.GetIncidentHistoryForFiles(ctx, queryPaths, 180)
				ownership, _ := hybridClient.GetOwnershipHistoryForFiles(ctx, queryPaths)
				cochange, _ := hybridClient.GetCoChangePartnersWithContext(ctx, queryPaths, 0.3)
				blastRadius, _ := hybridClient.GetBlastRadiusWithIncidents(ctx, file)

				// Build demo output data structure
				demoData := &output.DemoOutputData{
					Assessment:  assessment,
					Incidents:   incidents,
					Ownership:   ownership,
					CoChange:    cochange,
					BlastRadius: blastRadius,
					CLQSScore:   clqsScore,
					FilePath:    file,
				}

				// Display Manager View (business impact)
				output.DisplayManagerView(demoData)

				// Display Developer View (actionable insights)
				output.DisplayDeveloperView(demoData)

				// Display CLQS Confidence (data quality)
				if clqsScore != nil {
					output.DisplayCLQSConfidence(demoData)
				}

				// Show detailed hop-by-hop investigation trace
				output.DisplayPhase2Trace(*assessment)
			} else {
				// Standard Mode: Query hybrid data and show Manager + Developer views
				// Query hybrid client for enriched context data
				incidents, _ := hybridClient.GetIncidentHistoryForFiles(ctx, queryPaths, 180)
				ownership, _ := hybridClient.GetOwnershipHistoryForFiles(ctx, queryPaths)
				cochange, _ := hybridClient.GetCoChangePartnersWithContext(ctx, queryPaths, 0.3)
				blastRadius, _ := hybridClient.GetBlastRadiusWithIncidents(ctx, file)

				// Build demo output data structure
				demoData := &output.DemoOutputData{
					Assessment:  assessment,
					Incidents:   incidents,
					Ownership:   ownership,
					CoChange:    cochange,
					BlastRadius: blastRadius,
					CLQSScore:   clqsScore,
					FilePath:    file,
				}

				// Display Manager View (business impact)
				output.DisplayManagerView(demoData)

				// Display Developer View (actionable insights)
				output.DisplayDeveloperView(demoData)

				// Display CLQS Confidence (data quality)
				if clqsScore != nil {
					output.DisplayCLQSConfidence(demoData)
				}
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

		if hasHighRisk && geminiAPIKey != "" {
			// Rough estimate: 1000 tokens per high-risk file with Phase 2
			estimatedTokens = 1000 * totalFiles
			// Estimate cost: Much cheaper with Gemini (~$0.001 per 1000 tokens)
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

// initStagingClient creates a PostgreSQL staging client for GitHub data queries
func initStagingClient(ctx context.Context) (*database.StagingClient, error) {
	slog.Debug("initializing PostgreSQL staging client")

	cfg, err := appconfig.Load("")
	if err != nil {
		slog.Error("failed to load config for PostgreSQL staging", "error", err)
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	slog.Info("connecting to PostgreSQL staging", "host", cfg.Storage.PostgresHost, "port", cfg.Storage.PostgresPort, "database", cfg.Storage.PostgresDB)

	// Create staging client with same connection parameters
	stagingClient, err := database.NewStagingClient(
		ctx,
		cfg.Storage.PostgresHost,
		cfg.Storage.PostgresPort,
		cfg.Storage.PostgresDB,
		cfg.Storage.PostgresUser,
		cfg.Storage.PostgresPassword,
	)
	if err != nil {
		slog.Error("failed to connect to PostgreSQL staging", "error", err)
		return nil, err
	}

	slog.Info("successfully connected to PostgreSQL staging")
	return stagingClient, nil
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
// DEPRECATED: Replaced by FileResolver for better file history tracking
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

// loadCLQSScore loads the CLQS score for the repository from database
func loadCLQSScore(ctx context.Context, stagingClient *database.StagingClient, repoID int64) (*output.CLQSInfo, error) {
	query := `
		SELECT
			final_score,
			grade,
			rank,
			link_coverage_score,
			confidence_quality_score,
			evidence_diversity_score,
			temporal_precision_score,
			semantic_strength_score
		FROM clqs_scores
		WHERE repo_id = $1
		ORDER BY calculated_at DESC
		LIMIT 1
	`

	var (
		score              int
		grade              string
		rank               string
		linkCoverage       int
		confidenceQuality  int
		evidenceDiversity  int
		temporalPrecision  int
		semanticStrength   int
	)

	err := stagingClient.QueryRow(ctx, query, repoID).Scan(
		&score,
		&grade,
		&rank,
		&linkCoverage,
		&confidenceQuality,
		&evidenceDiversity,
		&temporalPrecision,
		&semanticStrength,
	)

	if err != nil {
		// CLQS score not found - this is okay, just means it hasn't been calculated yet
		return nil, nil
	}

	return &output.CLQSInfo{
		Score:              score,
		Grade:              grade,
		Rank:               rank,
		LinkCoverage:       linkCoverage,
		ConfidenceQuality:  confidenceQuality,
		EvidenceDiversity:  evidenceDiversity,
		TemporalPrecision:  temporalPrecision,
		SemanticStrength:   semanticStrength,
	}, nil
}
