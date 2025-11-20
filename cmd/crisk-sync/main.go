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

	// Phase 1: Sync Nodes (foundation - referenced before referencing)
	fmt.Printf("\n📦 Phase 1: Syncing Nodes\n")

	// 1.1 Developers
	devSyncer := sync.NewDeveloperSyncer(db, driver, cfg.Neo4j.Database)
	devsSynced, err := devSyncer.SyncMissingDevelopers(ctx, repoID)
	if err != nil {
		return 2, fmt.Errorf("Developer sync failed: %w", err)
	}

	// 1.2 Commits
	commitSyncer := sync.NewCommitSyncer(db, driver, cfg.Neo4j.Database)
	commitsSynced, err := commitSyncer.SyncMissingCommits(ctx, repoID)
	if err != nil {
		return 2, fmt.Errorf("Commit sync failed: %w", err)
	}

	// 1.3 Files
	fileSyncer := sync.NewFileSyncer(db, driver, cfg.Neo4j.Database)
	filesSynced, err := fileSyncer.SyncMissingFiles(ctx, repoID)
	if err != nil {
		return 2, fmt.Errorf("File sync failed: %w", err)
	}

	// 1.4 Issues
	issueSyncer := sync.NewIssueSyncer(db, driver, cfg.Neo4j.Database)
	issuesSynced, err := issueSyncer.SyncMissingIssues(ctx, repoID)
	if err != nil {
		return 2, fmt.Errorf("Issue sync failed: %w", err)
	}

	// 1.5 PRs
	prSyncer := sync.NewPRSyncer(db, driver, cfg.Neo4j.Database)
	prsSynced, err := prSyncer.SyncMissingPRs(ctx, repoID)
	if err != nil {
		return 2, fmt.Errorf("PR sync failed: %w", err)
	}

	// 1.6 CodeBlocks
	blockSyncer := sync.NewCodeBlockSyncer(db, driver, cfg.Neo4j.Database)
	blocksSynced, err := blockSyncer.SyncMissingBlocks(ctx, repoID)
	if err != nil {
		return 2, fmt.Errorf("CodeBlock sync failed: %w", err)
	}

	// Phase 2: Sync Edges (after all nodes exist)
	fmt.Printf("\n🔗 Phase 2: Syncing Edges\n")

	// 2.1 crisk-ingest edges (AUTHORED, MODIFIED, CREATED, MERGED_AS, REFERENCES, CLOSED_BY)
	ingestEdgeSyncer := sync.NewIngestEdgeSyncer(db, driver, cfg.Neo4j.Database)
	ingestEdgesSynced, err := ingestEdgeSyncer.SyncAllIngestEdges(ctx, repoID)
	if err != nil {
		return 2, fmt.Errorf("crisk-ingest edge sync failed: %w", err)
	}

	// 2.2 crisk-atomize commit→block edges (MODIFIED_BLOCK, CREATED_BLOCK, DELETED_BLOCK)
	commitBlockEdgeSyncer := sync.NewCommitBlockEdgeSyncer(db, driver, cfg.Neo4j.Database)
	commitBlockEdgesSynced, err := commitBlockEdgeSyncer.SyncMissingEdges(ctx, repoID)
	if err != nil {
		return 2, fmt.Errorf("Commit→CodeBlock edge sync failed: %w", err)
	}

	// 2.3 crisk-atomize semantic edges (CONTAINS, RENAMED_FROM, IMPORTS_FROM)
	atomizeEdgeSyncer := sync.NewAtomizeEdgeSyncer(db, driver, cfg.Neo4j.Database)
	atomizeEdgesSynced, err := atomizeEdgeSyncer.SyncAllAtomizeEdges(ctx, repoID)
	if err != nil {
		return 2, fmt.Errorf("crisk-atomize edge sync failed: %w", err)
	}

	// 2.4 crisk-index-incident edges (FIXED_BY_BLOCK)
	incidentEdgeSyncer := sync.NewIncidentEdgeSyncer(db, driver, cfg.Neo4j.Database)
	incidentEdgesSynced, err := incidentEdgeSyncer.SyncFixedByBlockEdges(ctx, repoID)
	if err != nil {
		return 2, fmt.Errorf("FIXED_BY_BLOCK edge sync failed: %w", err)
	}

	// 2.5 crisk-index-coupling edges (CO_CHANGES_WITH)
	couplingEdgeSyncer := sync.NewCouplingEdgeSyncer(db, driver, cfg.Neo4j.Database)
	couplingEdgesSynced, err := couplingEdgeSyncer.SyncCoChangesWithEdges(ctx, repoID)
	if err != nil {
		return 2, fmt.Errorf("CO_CHANGES_WITH edge sync failed: %w", err)
	}

	// Phase 3: Sync Properties (enrichment)
	fmt.Printf("\n📊 Phase 3: Syncing Properties\n")

	// 3.1 Incident properties on CodeBlock
	incidentPropSyncer := sync.NewIncidentPropSyncer(db, driver, cfg.Neo4j.Database)
	incidentPropsSynced, err := incidentPropSyncer.SyncIncidentProperties(ctx, repoID)
	if err != nil {
		return 2, fmt.Errorf("Incident property sync failed: %w", err)
	}

	// 3.2 Ownership properties on CodeBlock
	ownershipPropSyncer := sync.NewOwnershipPropSyncer(db, driver, cfg.Neo4j.Database)
	ownershipPropsSynced, err := ownershipPropSyncer.SyncOwnershipProperties(ctx, repoID)
	if err != nil {
		return 2, fmt.Errorf("Ownership property sync failed: %w", err)
	}

	// Run validation again to verify
	fmt.Printf("\n📊 Validating post-sync state...\n")
	finalResults, err := validator.ValidateAfterIngest(ctx, repoID)
	if err != nil {
		return 2, fmt.Errorf("post-sync validation failed: %w", err)
	}

	validation.LogResults(finalResults)

	// Summary
	totalNodesSynced := devsSynced + commitsSynced + filesSynced + issuesSynced + prsSynced + blocksSynced
	totalEdgesSynced := ingestEdgesSynced + commitBlockEdgesSynced + atomizeEdgesSynced + incidentEdgesSynced + couplingEdgesSynced
	totalPropsSynced := incidentPropsSynced + ownershipPropsSynced

	fmt.Printf("\n✅ Incremental sync complete:\n")
	fmt.Printf("\n📦 Nodes: %d total\n", totalNodesSynced)
	fmt.Printf("   Developers: %d\n", devsSynced)
	fmt.Printf("   Commits: %d\n", commitsSynced)
	fmt.Printf("   Files: %d\n", filesSynced)
	fmt.Printf("   Issues: %d\n", issuesSynced)
	fmt.Printf("   PRs: %d\n", prsSynced)
	fmt.Printf("   CodeBlocks: %d\n", blocksSynced)
	fmt.Printf("\n🔗 Edges: %d total\n", totalEdgesSynced)
	fmt.Printf("   crisk-ingest edges: %d\n", ingestEdgesSynced)
	fmt.Printf("   Commit→CodeBlock edges: %d\n", commitBlockEdgesSynced)
	fmt.Printf("   crisk-atomize edges: %d\n", atomizeEdgesSynced)
	fmt.Printf("   FIXED_BY_BLOCK edges: %d\n", incidentEdgesSynced)
	fmt.Printf("   CO_CHANGES_WITH edges: %d\n", couplingEdgesSynced)
	fmt.Printf("\n📊 Properties: %d blocks updated\n", totalPropsSynced)
	fmt.Printf("   Incident properties: %d blocks\n", incidentPropsSynced)
	fmt.Printf("   Ownership properties: %d blocks\n", ownershipPropsSynced)

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
