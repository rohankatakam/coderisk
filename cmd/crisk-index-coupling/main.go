package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/graph"
	"github.com/rohankatakam/coderisk/internal/llm"
	"github.com/rohankatakam/coderisk/internal/risk"
	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "crisk-index-coupling",
	Short: "Calculate coupling signals for CodeBlocks",
	Long: `crisk-index-coupling - Microservice 6: Coupling Risk Indexer

Calculates coupling signals for CodeBlocks in the repository.

Features:
  • Implicit coupling: Co-change analysis (statistical)
  • Explicit coupling: Dependency graph from IMPORTS_FROM
  • Creates CO_CHANGES_WITH edges between CodeBlocks
  • Calculates coupling strength based on co-change frequency

Output: Coupling edges and risk scores`,
	Version: Version,
	RunE:    runIndexCoupling,
}

var (
	repoID  int64
	verbose bool
	force   bool
)

func init() {
	rootCmd.Flags().Int64Var(&repoID, "repo-id", 0, "Repository ID from PostgreSQL (required)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.Flags().BoolVar(&force, "force", false, "Force recalculation of all coupling data")

	rootCmd.MarkFlagRequired("repo-id")

	rootCmd.SetVersionTemplate(`crisk-index-coupling {{.Version}}
Build time: ` + BuildTime + `
Git commit: ` + GitCommit + `
`)
}

func runIndexCoupling(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := context.Background()

	// Setup logging to file
	logFile, err := setupLogging()
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to setup log file: %v\n", err)
	} else {
		defer logFile.Close()
		fmt.Printf("📝 Logging to: %s\n", logFile.Name())
	}

	fmt.Printf("🚀 crisk-index-coupling - Coupling Indexing Service\n")
	fmt.Printf("   Repository ID: %d\n", repoID)
	fmt.Printf("   Timestamp: %s\n", startTime.Format(time.RFC3339))
	fmt.Println()

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get database URL
	dbURL := os.Getenv("POSTGRES_DSN")
	if dbURL == "" {
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			cfg.Storage.PostgresUser,
			cfg.Storage.PostgresPassword,
			cfg.Storage.PostgresHost,
			cfg.Storage.PostgresPort,
			cfg.Storage.PostgresDB,
		)
	}

	// Connect to PostgreSQL
	fmt.Printf("[1/5] Connecting to PostgreSQL...\n")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	fmt.Printf("  ✓ Connected to PostgreSQL\n\n")

	// Connect to Neo4j using graph backend
	fmt.Printf("[2/5] Connecting to Neo4j...\n")
	neo4jBackend, err := graph.NewNeo4jBackend(
		ctx,
		cfg.Neo4j.URI,
		cfg.Neo4j.User,
		cfg.Neo4j.Password,
		cfg.Neo4j.Database,
	)
	if err != nil {
		return fmt.Errorf("failed to connect to Neo4j backend: %w", err)
	}
	defer neo4jBackend.Close(ctx)
	fmt.Printf("  ✓ Connected to Neo4j\n\n")

	// Also connect legacy Neo4j driver for backward compatibility
	driver, err := neo4j.NewDriverWithContext(
		cfg.Neo4j.URI,
		neo4j.BasicAuth(cfg.Neo4j.User, cfg.Neo4j.Password, ""),
	)
	if err != nil {
		return fmt.Errorf("failed to create Neo4j driver: %w", err)
	}
	defer driver.Close(ctx)

	if err := driver.VerifyConnectivity(ctx); err != nil {
		return fmt.Errorf("failed to verify Neo4j connectivity: %w", err)
	}

	// Create LLM client (optional)
	fmt.Printf("[3/5] Initializing LLM client...\n")
	var llmClient *llm.Client
	if os.Getenv("GEMINI_API_KEY") != "" {
		llmClient, err = llm.NewClient(ctx, cfg)
		if err != nil {
			fmt.Printf("  ⚠️  LLM client creation failed: %v\n", err)
			fmt.Printf("  → Continuing without LLM explanations\n\n")
			llmClient = nil
		} else if llmClient.IsEnabled() {
			fmt.Printf("  ✓ LLM client ready (%s)\n\n", llmClient.GetProvider())
		} else {
			llmClient = nil
		}
	} else {
		fmt.Printf("  ⚠️  LLM client disabled (GEMINI_API_KEY not set)\n\n")
	}

	// Create coupling calculator with Neo4j support
	calc := risk.NewCouplingCalculator(db, neo4jBackend, driver, cfg.Neo4j.Database, llmClient, repoID)

	// Phase 1: Calculate co-changes
	fmt.Printf("[4/5] Calculating co-change relationships...\n")
	fmt.Printf("  Strategy: ULTRA-STRICT (Thesis-Aligned Blast Radius Warnings)\n")
	fmt.Printf("  Filters:\n")
	fmt.Printf("    • Co-change rate ≥95%% (blocks changed together 95%%+ of time)\n")
	fmt.Printf("    • Minimum 10 absolute co-changes (statistical significance)\n")
	fmt.Printf("    • BOTH blocks have incident_count ≥1 (only incident-prone blocks)\n")
	fmt.Printf("    • 12-month rolling window (exclude stale relationships >365 days)\n")
	fmt.Printf("  Storage: Static facts only (co_change_count, co_change_percentage)\n")
	fmt.Printf("  Dynamic: Incident weights, recency, Top-K computed at query time\n\n")

	edgesCreated, err := calc.CalculateCoChanges(ctx)
	if err != nil {
		return fmt.Errorf("failed to calculate co-changes: %w", err)
	}
	fmt.Printf("  ✓ Created %d co-change edges in PostgreSQL\n", edgesCreated)

	// Phase 2: Update coupling aggregates
	fmt.Printf("\n[5/5] Updating coupling aggregates and risk scores...\n\n")

	blocksUpdated, err := calc.UpdateCouplingAggregates(ctx)
	if err != nil {
		return fmt.Errorf("failed to update coupling aggregates: %w", err)
	}
	fmt.Printf("  ✓ Updated %d blocks with coupling aggregates\n\n", blocksUpdated)

	// Phase 3: Calculate final risk scores
	scoresCalculated, err := calc.CalculateRiskScores(ctx)
	if err != nil {
		return fmt.Errorf("failed to calculate risk scores: %w", err)
	}
	fmt.Printf("  ✓ Calculated risk scores for %d blocks\n\n", scoresCalculated)

	// Mark all blocks as coupling-indexed (idempotency tracking)
	if force {
		fmt.Printf("  ⚠️  Force mode: Recalculated all coupling data\n")
	}
	fmt.Printf("  Updating coupling_indexed_at timestamps...\n")
	if err := calc.MarkCouplingIndexed(ctx); err != nil {
		fmt.Printf("  ⚠️  Warning: Failed to update timestamps: %v\n", err)
	}
	fmt.Println()

	// Get statistics
	fmt.Printf("📊 Co-Change Statistics:\n")
	stats, err := calc.GetCoChangeStatistics(ctx)
	if err != nil {
		return fmt.Errorf("failed to get statistics: %w", err)
	}

	fmt.Printf("   Total Edges: %d\n", stats["total_edges"])
	if stats["total_edges"].(int) > 0 {
		fmt.Printf("   Min Coupling Rate: %.1f%%\n", stats["min_rate"].(float64)*100)
		fmt.Printf("   Max Coupling Rate: %.1f%%\n", stats["max_rate"].(float64)*100)
		fmt.Printf("   Avg Coupling Rate: %.1f%%\n", stats["avg_rate"].(float64)*100)
		fmt.Printf("\n   Coupling Distribution:\n")
		fmt.Printf("     High (≥75%%): %d edges\n", stats["high_coupling_count"])
		fmt.Printf("     Medium (50-75%%): %d edges\n", stats["medium_coupling_count"])
	}

	// Get top coupled blocks
	fmt.Printf("\n📍 Top 10 Most Coupled Blocks:\n")
	topBlocks, err := calc.GetTopCoupledBlocks(ctx, 10)
	if err != nil {
		return fmt.Errorf("failed to get top coupled blocks: %w", err)
	}

	if len(topBlocks) == 0 {
		fmt.Printf("   No coupled blocks found\n")
	} else {
		for i, block := range topBlocks {
			fmt.Printf("   %d. %s (%s)\n", i+1, block["block_name"], block["block_type"])
			fmt.Printf("      File: %s\n", block["file_path"])
			fmt.Printf("      Couplings: %d edges | Avg Rate: %.1f%%\n",
				block["total_couplings"],
				block["avg_coupling_rate"].(float64)*100)
		}
	}

	fmt.Printf("\n✅ Coupling indexing complete\n")
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("   Total time: %v\n", time.Since(startTime))
	fmt.Printf("   Co-change edges: %d\n", edgesCreated)
	fmt.Printf("   Blocks with coupling data: %d\n", blocksUpdated)
	fmt.Printf("   Risk scores calculated: %d\n", scoresCalculated)
	fmt.Printf("\n🚀 All indexing services complete! Repository is ready for risk analysis.\n")

	return nil
}

func setupLogging() (*os.File, error) {
	// Create logs directory if it doesn't exist
	logDir := "/tmp/coderisk-logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("20060102_150405")
	logPath := filepath.Join(logDir, fmt.Sprintf("crisk-index-coupling_%s.log", timestamp))

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Setup multi-writer to write to both stdout and file
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	return logFile, nil
}
