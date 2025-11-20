package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/sync"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: crisk-sync-edges <repo-id>")
		os.Exit(1)
	}

	repoID, err := strconv.ParseInt(os.Args[1], 10, 64)
	if err != nil {
		log.Fatalf("Invalid repo-id: %v", err)
	}

	ctx := context.Background()

	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Printf("🔗 Commit→CodeBlock Edge Backfill Tool\n")
	fmt.Printf("   Repository ID: %d\n\n", repoID)

	// Connect to PostgreSQL
	fmt.Println("[1/2] Connecting to PostgreSQL...")
	stagingDB, err := database.NewStagingClient(
		ctx,
		cfg.Storage.PostgresHost,
		cfg.Storage.PostgresPort,
		cfg.Storage.PostgresDB,
		cfg.Storage.PostgresUser,
		cfg.Storage.PostgresPassword,
	)
	if err != nil {
		log.Fatalf("PostgreSQL connection failed: %v", err)
	}
	defer stagingDB.Close()
	fmt.Println("  ✓ Connected")

	// Connect to Neo4j
	fmt.Println("\n[2/2] Connecting to Neo4j...")
	neoDriver, err := neo4j.NewDriverWithContext(
		cfg.Neo4j.URI,
		neo4j.BasicAuth("neo4j", cfg.Neo4j.Password, ""),
	)
	if err != nil {
		log.Fatalf("Neo4j connection failed: %v", err)
	}
	defer neoDriver.Close(ctx)

	if err := neoDriver.VerifyConnectivity(ctx); err != nil {
		log.Fatalf("Neo4j connection verification failed: %v", err)
	}
	fmt.Println("  ✓ Connected")

	// Run edge syncer
	fmt.Println("\n🚀 Starting edge sync...")
	edgeSyncer := sync.NewCommitBlockEdgeSyncer(stagingDB.DB(), neoDriver, cfg.Neo4j.Database)
	edgesSynced, err := edgeSyncer.SyncMissingEdges(ctx, repoID)
	if err != nil {
		log.Fatalf("Edge sync failed: %v", err)
	}

	fmt.Printf("\n✅ Successfully synced %d Commit→CodeBlock edges\n", edgesSynced)
}
