package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/sync"
	"github.com/rohankatakam/coderisk/internal/validation"
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
	Use:   "crisk-sync",
	Short: "Database consistency recovery and validation tool",
	Long: `crisk-sync - PostgreSQL/Neo4j Consistency Recovery

Validates and repairs data consistency between PostgreSQL (source of truth)
and Neo4j (derived cache). Implements the Postgres-First Write Protocol.

Modes:
  incremental   Sync delta only (missing entities) - fast, minutes
  full          Rebuild entire Neo4j graph from Postgres - slow, hours
  validate-only Report discrepancies without modification - fast, seconds

Exit Codes:
  0 - Success (all entities synced, variance ≥95%)
  1 - Warning (variance 90-95% or minor issues)
  2 - Failure (variance <90% or critical errors)

Examples:
  crisk-sync --repo-id 11 --mode incremental
  crisk-sync --repo-id 11 --mode full --dry-run
  crisk-sync --repo-id 11 --mode validate-only`,
	Version: Version,
	RunE:    runSync,
}

var (
	repoID       int64
	mode         string
	dryRun       bool
	validateOnly bool
)

func init() {
	rootCmd.Flags().Int64Var(&repoID, "repo-id", 0, "Repository ID to sync (required)")
	rootCmd.Flags().StringVar(&mode, "mode", "incremental", "Sync mode: incremental, full, validate-only")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Report actions without executing")
	rootCmd.Flags().BoolVar(&validateOnly, "validate-only", false, "Validate only (alias for --mode validate-only)")

	rootCmd.MarkFlagRequired("repo-id")

	rootCmd.SetVersionTemplate(`crisk-sync {{.Version}}
Build time: ` + BuildTime + `
Git commit: ` + GitCommit + `
`)
}

func runSync(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := context.Background()

	fmt.Printf("🔄 crisk-sync - Database Consistency Recovery\n")
	fmt.Printf("   Repository ID: %d\n", repoID)

	// Validate-only flag overrides mode
	if validateOnly {
		mode = "validate-only"
	}

	fmt.Printf("   Mode: %s\n", mode)
	if dryRun {
		fmt.Printf("   Dry Run: true (no changes will be made)\n")
	}
	fmt.Println()

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Connect to PostgreSQL
	fmt.Printf("[1/3] Connecting to PostgreSQL...\n")
	stagingDB, err := database.NewStagingClient(
		ctx,
		cfg.Storage.PostgresHost,
		cfg.Storage.PostgresPort,
		cfg.Storage.PostgresDB,
		cfg.Storage.PostgresUser,
		cfg.Storage.PostgresPassword,
	)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer stagingDB.Close()
	fmt.Printf("  ✓ Connected\n\n")

	// Connect to Neo4j
	fmt.Printf("[2/3] Connecting to Neo4j...\n")

	// Build Neo4j URI from config
	neoURI := cfg.Neo4j.URI
	neoPassword := cfg.Neo4j.Password

	// Fallback to environment variables if config is empty
	if neoURI == "" {
		envURI := os.Getenv("NEO4J_URI")
		if envURI != "" {
			neoURI = envURI
		} else {
			neoURI = "bolt://localhost:7688"
		}
	}
	if neoPassword == "" {
		neoPassword = os.Getenv("NEO4J_PASSWORD")
	}

	neoDriver, err := neo4j.NewDriverWithContext(
		neoURI,
		neo4j.BasicAuth("neo4j", neoPassword, ""),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to Neo4j: %w", err)
	}
	defer neoDriver.Close(ctx)

	// Verify connection
	if err := neoDriver.VerifyConnectivity(ctx); err != nil {
		return fmt.Errorf("neo4j connection verification failed: %w", err)
	}
	fmt.Printf("  ✓ Connected\n\n")

	// Execute sync based on mode
	fmt.Printf("[3/3] Executing %s sync...\n", mode)

	var exitCode int
	switch mode {
	case "incremental":
		exitCode, err = syncIncremental(ctx, stagingDB.DB(), neoDriver, repoID, dryRun)
	case "full":
		exitCode, err = syncFull(ctx, stagingDB.DB(), neoDriver, repoID, dryRun)
	case "validate-only":
		exitCode, err = validateOnlyMode(ctx, stagingDB.DB(), neoDriver, repoID)
	default:
		return fmt.Errorf("invalid mode: %s (must be incremental, full, or validate-only)", mode)
	}

	if err != nil {
		fmt.Printf("\n❌ Sync failed: %v\n", err)
		return err
	}

	duration := time.Since(startTime)
	fmt.Printf("\n✅ Sync completed in %v\n", duration.Round(time.Second))

	os.Exit(exitCode)
	return nil
}

func syncIncremental(ctx context.Context, db *sql.DB, driver neo4j.DriverWithContext, repoID int64, dryRun bool) (int, error) {
	validator := validation.NewConsistencyValidator(db, driver)

	// Run validation first
	results, err := validator.ValidateAfterIngest(ctx, repoID)
	if err != nil {
		return 2, fmt.Errorf("validation failed: %w", err)
	}

	validation.LogResults(results)

	// Check if sync needed
	needsSync := false
	for _, r := range results {
		if !r.PassedThreshold {
			needsSync = true
			break
		}
	}

	if !needsSync {
		fmt.Printf("\n✓ All entities already in sync (≥95%% threshold)\n")
		return 0, nil // Success
	}

	if dryRun {
		fmt.Printf("\n[DRY RUN] Would sync the following:\n")
		for _, r := range results {
			if !r.PassedThreshold {
				delta := r.PostgresCount - r.Neo4jCount
				fmt.Printf("  %s: %d missing entities\n", r.EntityType, delta)
			}
		}
		return 0, nil // Success (dry run)
	}

	// Perform incremental sync
	fmt.Printf("\n🔄 Starting incremental sync...\n")

	// Get Neo4j database name from config
	cfg, err := config.Load("")
	if err != nil {
		return 2, fmt.Errorf("failed to load config for Neo4j database: %w", err)
	}

	// Sync CodeBlocks
	blockSyncer := sync.NewCodeBlockSyncer(db, driver, cfg.Neo4j.Database)
	blocksSynced, err := blockSyncer.SyncMissingBlocks(ctx, repoID)
	if err != nil {
		return 2, fmt.Errorf("CodeBlock sync failed: %w", err)
	}

	// Sync Commit→CodeBlock edges (MODIFIED_BLOCK, CREATED_BLOCK, DELETED_BLOCK)
	edgeSyncer := sync.NewCommitBlockEdgeSyncer(db, driver, cfg.Neo4j.Database)
	edgesSynced, err := edgeSyncer.SyncMissingEdges(ctx, repoID)
	if err != nil {
		return 2, fmt.Errorf("Commit→CodeBlock edge sync failed: %w", err)
	}

	// TODO: Add more syncers for other entities (Commits, Files, etc.)

	// Run validation again to verify
	fmt.Printf("\n📊 Validating post-sync state...\n")
	finalResults, err := validator.ValidateAfterIngest(ctx, repoID)
	if err != nil {
		return 2, fmt.Errorf("post-sync validation failed: %w", err)
	}

	validation.LogResults(finalResults)

	// Summary
	fmt.Printf("\n✅ Incremental sync complete:\n")
	fmt.Printf("   CodeBlocks synced: %d\n", blocksSynced)
	fmt.Printf("   Commit→CodeBlock edges synced: %d\n", edgesSynced)

	// Determine exit code based on final variance
	minVariance := 100.0
	for _, r := range finalResults {
		if r.VariancePercent < minVariance {
			minVariance = r.VariancePercent
		}
	}

	if minVariance >= 95.0 {
		return 0, nil // Success
	} else if minVariance >= 90.0 {
		return 1, nil // Warning
	} else {
		return 2, nil // Failure
	}
}

func syncFull(ctx context.Context, db *sql.DB, driver neo4j.DriverWithContext, repoID int64, dryRun bool) (int, error) {
	fmt.Printf("⚠️  FULL REBUILD: This will delete all Neo4j data for repo_id=%d\n", repoID)

	if dryRun {
		fmt.Printf("\n[DRY RUN] Would execute:\n")
		fmt.Printf("  1. DELETE all Neo4j nodes/edges for repo_id=%d\n", repoID)
		fmt.Printf("  2. Rebuild from Postgres (commits, files, blocks, etc.)\n")
		fmt.Printf("  3. Validate final state\n")
		return 0, nil // Success (dry run)
	}

	// Step 1: Clear Neo4j data for this repo
	fmt.Printf("\n  Clearing Neo4j data for repo_id=%d...\n", repoID)
	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	result, err := session.Run(ctx,
		"MATCH (n {repo_id: $repoID}) DETACH DELETE n RETURN count(n) as deleted",
		map[string]any{"repoID": repoID},
	)
	if err != nil {
		return 2, fmt.Errorf("failed to clear Neo4j data: %w", err)
	}

	if result.Next(ctx) {
		deleted := result.Record().Values[0].(int64)
		fmt.Printf("  ✓ Deleted %d nodes\n", deleted)
	}

	// Step 2: Rebuild would go here
	// TODO: Re-run crisk-ingest, crisk-atomize, indexers
	fmt.Printf("\n⚠️  Full rebuild requires re-running microservices:\n")
	fmt.Printf("   1. crisk-ingest --repo-id %d\n", repoID)
	fmt.Printf("   2. crisk-atomize --repo-id %d\n", repoID)
	fmt.Printf("   3. crisk-index-* --repo-id %d\n", repoID)
	fmt.Printf("\n  Automated full rebuild not yet implemented.\n")

	return 1, nil // Warning
}

func validateOnlyMode(ctx context.Context, db *sql.DB, driver neo4j.DriverWithContext, repoID int64) (int, error) {
	validator := validation.NewConsistencyValidator(db, driver)

	// Run validation
	results, err := validator.ValidateAfterIngest(ctx, repoID)
	if err != nil {
		return 2, fmt.Errorf("validation failed: %w", err)
	}

	validation.LogResults(results)

	// Determine exit code based on variance thresholds
	minVariance := 100.0
	for _, r := range results {
		if r.VariancePercent < minVariance {
			minVariance = r.VariancePercent
		}
	}

	if minVariance >= 95.0 {
		return 0, nil // Success: all entities ≥95%
	} else if minVariance >= 90.0 {
		return 1, nil // Warning: some entities 90-95%
	} else {
		return 2, nil // Failure: some entities <90%
	}
}
