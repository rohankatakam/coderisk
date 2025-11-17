package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/llm"
	"github.com/rohankatakam/coderisk/internal/risk"
)

func main() {
	ctx := context.Background()

	log.Println("=== AGENT-P3A: Ownership Risk Calculator ===")
	log.Println("Starting ownership indexing...")

	// Create minimal config for environment-based execution
	cfg := &config.Config{
		API: config.APIConfig{
			GeminiKey:   os.Getenv("GEMINI_API_KEY"),
			GeminiModel: "gemini-2.0-flash",
		},
	}

	// Connect to Neo4j
	neoURI := os.Getenv("NEO4J_URI")
	neoPassword := os.Getenv("NEO4J_PASSWORD")
	if neoURI == "" {
		neoURI = "bolt://localhost:7688"
	}
	if neoPassword == "" {
		neoPassword = "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
	}

	log.Printf("Connecting to Neo4j at %s...", neoURI)
	driver, err := neo4j.NewDriverWithContext(
		neoURI,
		neo4j.BasicAuth("neo4j", neoPassword, ""),
	)
	if err != nil {
		log.Fatalf("Failed to create Neo4j driver: %v", err)
	}
	defer driver.Close(ctx)

	// Verify Neo4j connectivity
	if err := driver.VerifyConnectivity(ctx); err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	log.Println("✓ Neo4j connected")

	// Create LLM client (optional for ownership calculations)
	log.Println("Creating LLM client...")
	llmClient, err := llm.NewClient(ctx, cfg)
	if err != nil {
		log.Printf("Warning: Failed to create LLM client: %v", err)
		log.Println("Continuing without LLM (semantic importance will be skipped)")
	} else if llmClient.IsEnabled() {
		log.Printf("✓ LLM client ready (%s)", llmClient.GetProvider())
	} else {
		log.Println("LLM client disabled (semantic importance will be skipped)")
	}

	// Create ownership indexer
	indexer := risk.NewBlockIndexer(driver, "neo4j", llmClient)
	log.Println("✓ Ownership indexer created")

	// Get repo ID from args or use default
	repoID := int64(4)
	if len(os.Args) > 1 {
		log.Printf("Note: Using default repo_id=4 (args ignored)")
	}

	// Check how many blocks exist in Neo4j
	log.Printf("\nChecking CodeBlock nodes in Neo4j for repo_id=%d...", repoID)
	blockCount, err := countCodeBlocks(ctx, driver, repoID)
	if err != nil {
		log.Fatalf("Failed to count blocks: %v", err)
	}
	log.Printf("Found %d CodeBlock nodes in Neo4j", blockCount)

	if blockCount == 0 {
		log.Fatalf("ERROR: No CodeBlock nodes found for repo_id=%d. Has AGENT-P2C completed?", repoID)
	}

	// Run ownership calculations
	log.Printf("\n=== Phase 1: Original Authors ===")
	start := time.Now()
	result, err := indexer.IndexOwnership(ctx, repoID)
	if err != nil {
		log.Fatalf("Ownership indexing failed: %v", err)
	}

	// Print results
	log.Printf("\n=== OWNERSHIP INDEXING COMPLETE ===")
	log.Printf("Repository ID: %d", result.RepoID)
	log.Printf("Success: %v", result.Success)
	log.Printf("Total Duration: %v", result.TotalDuration)
	log.Printf("\nPhase Results:")
	log.Printf("  - Original Authors Set: %d blocks", result.OriginalAuthors)
	log.Printf("  - Last Modifiers Set: %d blocks", result.LastModifiers)
	log.Printf("  - Familiarity Maps Set: %d blocks", result.FamiliarityMaps)
	log.Printf("\nVerification:")
	log.Printf("  - Incomplete Blocks: %d", result.IncompleteBlocks)

	if result.IncompleteBlocks > 0 {
		log.Printf("\n⚠️  WARNING: %d blocks are missing ownership properties", result.IncompleteBlocks)
	} else {
		log.Printf("\n✅ SUCCESS: All %d blocks have complete ownership properties", blockCount)
	}

	// Get detailed stats
	log.Printf("\n=== Indexing Statistics ===")
	stats, err := indexer.GetIndexingStats(ctx, repoID)
	if err != nil {
		log.Printf("Warning: Failed to get stats: %v", err)
	} else {
		log.Printf("Total Blocks: %d", stats["total_blocks"])
		log.Printf("With Original Author: %d", stats["with_original_author"])
		log.Printf("With Last Modifier: %d", stats["with_last_modifier"])
		log.Printf("With Familiarity Map: %d", stats["with_familiarity_map"])
		log.Printf("Missing Original Author: %d", stats["missing_original_author"])
		log.Printf("Missing Last Modifier: %d", stats["missing_last_modifier"])
		log.Printf("Missing Familiarity Map: %d", stats["missing_familiarity_map"])
	}

	log.Printf("\n=== Total Execution Time: %v ===", time.Since(start))
	log.Println("\nAGENT-P3A: OWNERSHIP INDEXING COMPLETE ✅")
}

func countCodeBlocks(ctx context.Context, driver neo4j.DriverWithContext, repoID int64) (int, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: "neo4j",
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
