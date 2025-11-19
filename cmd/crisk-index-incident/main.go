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
	Use:   "crisk-index-incident",
	Short: "Link issues to CodeBlocks and calculate temporal risk",
	Long: `crisk-index-incident - Microservice 4: Temporal Risk Indexer

Links issues to CodeBlocks that were modified to fix them.
Calculates temporal risk signals based on incident history.

Features:
  • File-level linking: Issue → Commit → File
  • Function-level linking: Issue → Commit → CodeBlock
  • LLM summarization of incident history per block
  • Stores temporal_summary property on CodeBlock nodes

Output: CodeBlocks with temporal_summary property`,
	Version: Version,
	RunE:    runIndexIncident,
}

var (
	repoID  int64
	verbose bool
	force   bool
)

func init() {
	rootCmd.Flags().Int64Var(&repoID, "repo-id", 0, "Repository ID from PostgreSQL (required)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.Flags().BoolVar(&force, "force", false, "Force reprocessing of all blocks (ignore temporal_indexed_at)")

	rootCmd.MarkFlagRequired("repo-id")

	rootCmd.SetVersionTemplate(`crisk-index-incident {{.Version}}
Build time: ` + BuildTime + `
Git commit: ` + GitCommit + `
`)
}

func runIndexIncident(cmd *cobra.Command, args []string) error {
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

	fmt.Printf("🚀 crisk-index-incident - Temporal Risk Indexing Service\n")
	fmt.Printf("   Repository ID: %d\n", repoID)
	fmt.Printf("   Timestamp: %s\n", startTime.Format(time.RFC3339))
	fmt.Println()

	// Get database URL from config or environment
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	dbURL := os.Getenv("POSTGRES_DSN")
	if dbURL == "" {
		// Build from config
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			cfg.Storage.PostgresUser,
			cfg.Storage.PostgresPassword,
			cfg.Storage.PostgresHost,
			cfg.Storage.PostgresPort,
			cfg.Storage.PostgresDB,
		)
	}

	// Connect to PostgreSQL
	fmt.Printf("[1/4] Connecting to PostgreSQL...\n")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	fmt.Printf("  ✓ Connected to PostgreSQL\n\n")

	// Connect to Neo4j for graph updates
	fmt.Printf("[2/5] Connecting to Neo4j...\n")
	neo4jBackend, err := graph.NewNeo4jBackend(
		ctx,
		cfg.Neo4j.URI,
		cfg.Neo4j.User,
		cfg.Neo4j.Password,
		cfg.Neo4j.Database,
	)
	if err != nil {
		return fmt.Errorf("failed to connect to Neo4j: %w", err)
	}
	defer neo4jBackend.Close(ctx)
	fmt.Printf("  ✓ Connected to Neo4j\n\n")

	// Create LLM client for summarization
	fmt.Printf("[3/5] Initializing LLM client...\n")
	llmClient, err := llm.NewClient(ctx, cfg)
	if err != nil {
		fmt.Printf("  ⚠️  LLM client creation failed: %v\n", err)
		fmt.Printf("  → Continuing without LLM summaries\n\n")
	} else if llmClient.IsEnabled() {
		fmt.Printf("  ✓ LLM client ready (%s)\n\n", llmClient.GetProvider())
	} else {
		fmt.Printf("  ⚠️  LLM client not enabled, summaries will be skipped\n\n")
	}

	// Create temporal calculator with Neo4j support
	calc := risk.NewTemporalCalculator(db, neo4jBackend, llmClient, repoID)

	// Step 1: Link incidents to blocks via commits
	fmt.Printf("[4/5] Linking incidents to CodeBlocks...\n")
	fmt.Printf("  Strategy: Issue → (closed by) → Commit → (modified) → CodeBlock\n")
	fmt.Printf("  Creating FIXED_BY_BLOCK edges in Neo4j graph\n")

	linkedCount, err := calc.LinkIssuesViaCommits(ctx)
	if err != nil {
		return fmt.Errorf("failed to link incidents: %w", err)
	}
	fmt.Printf("  ✓ Created %d incident links (PostgreSQL + Neo4j)\n\n", linkedCount)

	// Step 2: Calculate incident counts and sync to Neo4j
	fmt.Printf("  Calculating incident counts and syncing to Neo4j...\n")
	blocksUpdated, err := calc.CalculateIncidentCounts(ctx)
	if err != nil {
		return fmt.Errorf("failed to calculate counts: %w", err)
	}
	fmt.Printf("  ✓ Updated %d blocks with incident counts (PostgreSQL + Neo4j)\n\n", blocksUpdated)

	// Step 3: Generate temporal summaries using LLM and sync to Neo4j
	fmt.Printf("[5/5] Generating temporal summaries...\n")
	if llmClient != nil && llmClient.IsEnabled() {
		summaryCount, err := calc.GenerateTemporalSummaries(ctx, force)
		if err != nil {
			fmt.Printf("  ⚠️  Warning: Failed to generate summaries: %v\n", err)
		} else {
			fmt.Printf("  ✓ Generated %d temporal summaries (PostgreSQL + Neo4j)\n\n", summaryCount)
		}
	} else {
		fmt.Printf("  ⚠️  Skipping temporal summaries (LLM not enabled)\n\n")
	}

	// Show statistics
	fmt.Printf("📊 Incident Statistics:\n")
	stats, err := calc.GetIncidentStatistics(ctx)
	if err != nil {
		return fmt.Errorf("failed to get statistics: %w", err)
	}

	fmt.Printf("   Total blocks:              %d\n", stats["total_blocks"])
	fmt.Printf("   Blocks with incidents:     %d\n", stats["blocks_with_incidents"])
	fmt.Printf("   Blocks without incidents:  %d\n", stats["blocks_without_incidents"])
	fmt.Printf("   Total unique issues:       %d\n", stats["total_unique_issues"])
	fmt.Printf("   Total incident links:      %d\n", stats["total_incident_links"])
	blocksWithIncidents, ok := stats["blocks_with_incidents"].(int)
	if !ok {
		// Try int64 if int fails
		blocksWithIncidentsInt64, ok := stats["blocks_with_incidents"].(int64)
		if ok {
			blocksWithIncidents = int(blocksWithIncidentsInt64)
		}
	}
	if blocksWithIncidents > 0 {
		fmt.Printf("   Average incidents/block:   %.2f\n", stats["avg_incidents_per_block"])
		fmt.Printf("   Max incidents/block:       %.0f\n", stats["max_incidents_per_block"])
	}

	// Show top hotspots
	fmt.Printf("\n📍 Top 5 Incident Hotspots:\n")
	topBlocks, err := calc.GetTopIncidentBlocks(ctx, 5)
	if err != nil {
		return fmt.Errorf("failed to get top blocks: %w", err)
	}

	if len(topBlocks) == 0 {
		fmt.Printf("   No incident hotspots found\n")
	} else {
		for i, block := range topBlocks {
			fmt.Printf("   %d. %s (%s) - %d incidents\n",
				i+1,
				block["block_name"],
				block["file_path"],
				block["incident_count"])
		}
	}

	// Summary
	totalDuration := time.Since(startTime)
	fmt.Printf("\n✅ Temporal indexing complete\n")
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("   Total time: %v\n", totalDuration)
	fmt.Printf("   Incident links created: %d\n", linkedCount)
	fmt.Printf("   Blocks updated: %d\n", blocksUpdated)
	fmt.Printf("   Blocks with incidents: %d/%d (%.1f%%)\n",
		stats["blocks_with_incidents"],
		stats["total_blocks"],
		float64(blocksWithIncidents)*100.0/float64(stats["total_blocks"].(int)))
	fmt.Printf("\n🚀 Next: crisk-index-ownership --repo-id %d\n", repoID)

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
	logPath := filepath.Join(logDir, fmt.Sprintf("crisk-index-incident_%s.log", timestamp))

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
