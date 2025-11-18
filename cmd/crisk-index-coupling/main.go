package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"github.com/rohankatakam/coderisk/internal/config"
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
)

func init() {
	rootCmd.Flags().Int64Var(&repoID, "repo-id", 0, "Repository ID from PostgreSQL (required)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	rootCmd.MarkFlagRequired("repo-id")

	rootCmd.SetVersionTemplate(`crisk-index-coupling {{.Version}}
Build time: ` + BuildTime + `
Git commit: ` + GitCommit + `
`)
}

func runIndexCoupling(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	fmt.Printf("🚀 crisk-index-coupling - Coupling Indexing Service\n")
	fmt.Printf("   Repository ID: %d\n", repoID)
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
	fmt.Printf("[1/3] Connecting to PostgreSQL...\n")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	fmt.Printf("  ✓ Connected to PostgreSQL\n\n")

	// Create LLM client (optional)
	fmt.Printf("[2/3] Initializing LLM client...\n")
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

	// Create coupling calculator
	calc := risk.NewCouplingCalculator(db, llmClient, repoID)

	// Calculate co-changes
	fmt.Printf("[3/3] Calculating co-change relationships...\n")
	fmt.Printf("  Strategy: Find CodeBlocks modified together in commits\n")
	fmt.Printf("  Threshold: ≥50%% co-change rate\n\n")

	edgesCreated, err := calc.CalculateCoChanges(ctx)
	if err != nil {
		return fmt.Errorf("failed to calculate co-changes: %w", err)
	}
	fmt.Printf("  ✓ Created %d co-change edges\n\n", edgesCreated)

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
	fmt.Printf("\n🚀 All indexing services complete! Repository is ready for risk analysis.\n")

	return nil
}
