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
	Use:   "crisk-index-ownership",
	Short: "Calculate ownership signals for CodeBlocks",
	Long: `crisk-index-ownership - Microservice 5: Ownership Risk Indexer

Calculates ownership signals for each CodeBlock in the repository.

Features:
  • Last modifier calculation
  • Staleness (days since last edit)
  • Familiarity map (edit counts per developer)
  • Original author tracking

Output: CodeBlocks with ownership properties populated`,
	Version: Version,
	RunE:    runIndexOwnership,
}

var (
	repoID  int64
	verbose bool
	force   bool
)

func init() {
	rootCmd.Flags().Int64Var(&repoID, "repo-id", 0, "Repository ID from Neo4j (required)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.Flags().BoolVar(&force, "force", false, "Force recalculation of all ownership data")

	rootCmd.MarkFlagRequired("repo-id")

	rootCmd.SetVersionTemplate(`crisk-index-ownership {{.Version}}
Build time: ` + BuildTime + `
Git commit: ` + GitCommit + `
`)
}

func runIndexOwnership(cmd *cobra.Command, args []string) error {
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

	fmt.Printf("🚀 crisk-index-ownership - Ownership Indexing Service\n")
	fmt.Printf("   Repository ID: %d\n", repoID)
	fmt.Printf("   Timestamp: %s\n", startTime.Format(time.RFC3339))
	fmt.Println()

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Connect to PostgreSQL
	fmt.Printf("[1/4] Connecting to PostgreSQL...\n")
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
	fmt.Printf("[2/4] Connecting to Neo4j...\n")
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

	// Create LLM client (optional for ownership calculations)
	fmt.Printf("[3/4] Initializing LLM client...\n")
	llmClient, err := llm.NewClient(ctx, cfg)
	if err != nil {
		fmt.Printf("  ⚠️  LLM client creation failed: %v\n", err)
		fmt.Printf("  → Continuing without LLM (semantic importance will be skipped)\n\n")
		llmClient = nil
	} else if llmClient.IsEnabled() {
		fmt.Printf("  ✓ LLM client ready (%s)\n\n", llmClient.GetProvider())
	} else {
		fmt.Printf("  ⚠️  LLM client disabled (semantic importance will be skipped)\n\n")
		llmClient = nil
	}

	// Check how many blocks exist
	blockCount, err := countCodeBlocks(ctx, driver, cfg.Neo4j.Database, repoID)
	if err != nil {
		return fmt.Errorf("failed to count blocks: %w", err)
	}
	fmt.Printf("  Found %d CodeBlock nodes\n", blockCount)

	if blockCount == 0 {
		return fmt.Errorf("no CodeBlock nodes found for repo_id=%d\n\nRun 'crisk-atomize' first", repoID)
	}

	// Create ownership indexer with PostgreSQL + Neo4j support
	indexer := risk.NewBlockIndexer(db, neo4jBackend, driver, cfg.Neo4j.Database, llmClient)

	// Run ownership calculations
	fmt.Printf("\n[4/4] Calculating ownership properties...\n")
	if force {
		fmt.Printf("  ⚠️  Force mode: Recalculating all ownership data\n")
	} else {
		fmt.Printf("  ℹ️  Normal mode: Calculating ownership data\n")
	}
	result, err := indexer.IndexOwnership(ctx, repoID)
	if err != nil {
		return fmt.Errorf("ownership indexing failed: %w", err)
	}

	// Mark all blocks as ownership-indexed (idempotency tracking)
	fmt.Printf("\n  Updating ownership_indexed_at timestamps...\n")
	if err := indexer.MarkOwnershipIndexed(ctx, repoID); err != nil {
		fmt.Printf("  ⚠️  Warning: Failed to update timestamps: %v\n", err)
	}

	// Print results
	fmt.Printf("\n📊 Ownership Indexing Results:\n")
	fmt.Printf("   Repository ID: %d\n", result.RepoID)
	fmt.Printf("   Total Duration: %v\n", result.TotalDuration)
	fmt.Printf("\n   Phase Results:\n")
	fmt.Printf("     - Original Authors Set: %d blocks\n", result.OriginalAuthors)
	fmt.Printf("     - Last Modifiers Set: %d blocks\n", result.LastModifiers)
	fmt.Printf("     - Familiarity Maps Set: %d blocks\n", result.FamiliarityMaps)
	fmt.Printf("\n   Verification:\n")
	fmt.Printf("     - Incomplete Blocks: %d\n", result.IncompleteBlocks)

	if result.IncompleteBlocks > 0 {
		fmt.Printf("\n⚠️  WARNING: %d blocks are missing ownership properties\n", result.IncompleteBlocks)
	} else {
		fmt.Printf("\n✅ SUCCESS: All %d blocks have complete ownership properties\n", blockCount)
	}

	// Get detailed stats
	stats, err := indexer.GetIndexingStats(ctx, repoID)
	if err != nil {
		fmt.Printf("\n⚠️  Warning: Failed to get stats: %v\n", err)
	} else {
		fmt.Printf("\n📈 Detailed Statistics:\n")
		fmt.Printf("   Total Blocks: %d\n", stats["total_blocks"])
		fmt.Printf("   With Original Author: %d\n", stats["with_original_author"])
		fmt.Printf("   With Last Modifier: %d\n", stats["with_last_modifier"])
		fmt.Printf("   With Familiarity Map: %d\n", stats["with_familiarity_map"])
		if stats["missing_original_author"] > 0 {
			fmt.Printf("   Missing Original Author: %d\n", stats["missing_original_author"])
		}
		if stats["missing_last_modifier"] > 0 {
			fmt.Printf("   Missing Last Modifier: %d\n", stats["missing_last_modifier"])
		}
		if stats["missing_familiarity_map"] > 0 {
			fmt.Printf("   Missing Familiarity Map: %d\n", stats["missing_familiarity_map"])
		}
	}

	fmt.Printf("\n✅ Ownership indexing complete\n")
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("   Total time: %v\n", time.Since(startTime))
	fmt.Printf("   Blocks with ownership data: %d/%d\n", result.OriginalAuthors, blockCount)
	fmt.Printf("\n🚀 Next: crisk-index-coupling --repo-id %d\n", repoID)

	return nil
}

func countCodeBlocks(ctx context.Context, driver neo4j.DriverWithContext, database string, repoID int64) (int, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: database,
	})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
			MATCH (b:CodeBlock {repo_id: $repoID})
			RETURN count(b) AS count
		`, map[string]any{
			"repoID": repoID,
		})
		if err != nil {
			return nil, err
		}

		record, err := res.Single(ctx)
		if err != nil {
			return nil, err
		}

		count, _ := record.Get("count")
		return count, nil
	})

	if err != nil {
		return 0, err
	}

	return int(result.(int64)), nil
}

func setupLogging() (*os.File, error) {
	// Create logs directory if it doesn't exist
	logDir := "/tmp/coderisk-logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("20060102_150405")
	logPath := filepath.Join(logDir, fmt.Sprintf("crisk-index-ownership_%s.log", timestamp))

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
