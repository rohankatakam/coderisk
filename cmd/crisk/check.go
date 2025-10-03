package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/coderisk/coderisk-go/internal/cache"
	"github.com/coderisk/coderisk-go/internal/database"
	"github.com/coderisk/coderisk-go/internal/graph"
	"github.com/coderisk/coderisk-go/internal/metrics"
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

func runCheck(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get files to check (from args or git status)
	files := args
	if len(files) == 0 {
		// TODO: Auto-detect from git status
		return fmt.Errorf("no files specified. Usage: crisk check <file>")
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

	// Assess each file
	repoID := "local" // TODO: Get from git repo
	for _, file := range files {
		fmt.Printf("\n=== Analyzing %s ===\n\n", file)

		result, err := registry.CalculatePhase1(ctx, repoID, file)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// Display results
		fmt.Println(result.FormatSummary())

		if result.ShouldEscalate {
			fmt.Println("\n⚠️  HIGH RISK - Would escalate to Phase 2 (LLM investigation)")
			fmt.Println("    Phase 2 requires LLM API key (set PHASE2_ENABLED=true)")
		}
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
