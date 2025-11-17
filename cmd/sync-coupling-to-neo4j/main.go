package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// CouplingRecord represents a coupling relationship from PostgreSQL
type CouplingRecord struct {
	BlockAID        int64
	BlockBID        int64
	CoChangeCount   int
	CoChangeRate    float64
	LastCoChangedAt time.Time
}

func main() {
	ctx := context.Background()

	// Connect to PostgreSQL
	pgDSN := os.Getenv("POSTGRES_DSN")
	if pgDSN == "" {
		log.Fatal("POSTGRES_DSN environment variable is required")
	}

	pgPool, err := pgxpool.New(ctx, pgDSN)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgPool.Close()

	// Connect to Neo4j
	neo4jURI := os.Getenv("NEO4J_URI")
	neo4jPassword := os.Getenv("NEO4J_PASSWORD")
	if neo4jURI == "" || neo4jPassword == "" {
		log.Fatal("NEO4J_URI and NEO4J_PASSWORD environment variables are required")
	}

	neo4jDriver, err := neo4j.NewDriverWithContext(neo4jURI, neo4j.BasicAuth("neo4j", neo4jPassword, ""))
	if err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	defer neo4jDriver.Close(ctx)

	// Verify Neo4j connection
	if err := neo4jDriver.VerifyConnectivity(ctx); err != nil {
		log.Fatalf("Failed to verify Neo4j connectivity: %v", err)
	}

	log.Println("✅ Connected to PostgreSQL and Neo4j")

	// Read all coupling data from PostgreSQL
	log.Println("📊 Reading coupling data from PostgreSQL...")
	query := `
		SELECT
			block_a_id,
			block_b_id,
			co_change_count,
			co_change_rate,
			COALESCE(last_co_changed_at, NOW()) as last_co_changed_at
		FROM code_block_coupling
		ORDER BY co_change_rate DESC
	`

	rows, err := pgPool.Query(ctx, query)
	if err != nil {
		log.Fatalf("Failed to query coupling data: %v", err)
	}
	defer rows.Close()

	var couplings []CouplingRecord
	for rows.Next() {
		var c CouplingRecord
		err := rows.Scan(
			&c.BlockAID,
			&c.BlockBID,
			&c.CoChangeCount,
			&c.CoChangeRate,
			&c.LastCoChangedAt,
		)
		if err != nil {
			log.Fatalf("Failed to scan coupling row: %v", err)
		}
		couplings = append(couplings, c)
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating coupling rows: %v", err)
	}

	log.Printf("📊 Found %d coupling relationships in PostgreSQL\n", len(couplings))

	if len(couplings) == 0 {
		log.Println("⚠️  No coupling data found. Nothing to sync.")
		return
	}

	// Sync to Neo4j
	log.Println("🔄 Syncing coupling data to Neo4j...")

	session := neo4jDriver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: "neo4j",
	})
	defer session.Close(ctx)

	successCount := 0
	errorCount := 0
	notFoundCount := 0

	for i, c := range couplings {
		// Create CO_CHANGES_WITH edge in Neo4j
		// IMPORTANT: Use db_id property to match CodeBlock nodes
		cypherQuery := `
			MATCH (a:CodeBlock {db_id: $blockAID})
			MATCH (b:CodeBlock {db_id: $blockBID})
			MERGE (a)-[r:CO_CHANGES_WITH]->(b)
			SET r.rate = $rate,
			    r.co_change_count = $count,
			    r.last_co_changed_at = datetime($lastChanged)
			RETURN a.db_id, b.db_id
		`

		result, err := session.Run(ctx, cypherQuery, map[string]interface{}{
			"blockAID":    c.BlockAID,
			"blockBID":    c.BlockBID,
			"rate":        c.CoChangeRate,
			"count":       c.CoChangeCount,
			"lastChanged": c.LastCoChangedAt.Format(time.RFC3339),
		})

		if err != nil {
			log.Printf("❌ Error syncing coupling %d->%d: %v", c.BlockAID, c.BlockBID, err)
			errorCount++
		} else if result.Next(ctx) {
			// Successfully created edge
			successCount++
		} else {
			// No edge created - likely one or both blocks don't exist in Neo4j
			notFoundCount++
			if notFoundCount <= 5 {
				log.Printf("⚠️  Blocks not found in Neo4j: %d -> %d", c.BlockAID, c.BlockBID)
			}
		}

		// Progress indicator
		if (i+1)%100 == 0 {
			log.Printf("   Processed %d/%d relationships...", i+1, len(couplings))
		}
	}

	log.Printf("\n✅ Sync complete!")
	log.Printf("   Success: %d relationships", successCount)
	log.Printf("   Errors: %d relationships", errorCount)
	log.Printf("   Not found: %d relationships (blocks don't exist in Neo4j)", notFoundCount)

	// Verify sync
	log.Println("\n🔍 Verifying Neo4j coupling edges...")
	verifyQuery := `
		MATCH ()-[r:CO_CHANGES_WITH]->()
		RETURN count(r) as edge_count
	`

	result, err := session.Run(ctx, verifyQuery, nil)
	if err != nil {
		log.Printf("❌ Failed to verify: %v", err)
	} else if result.Next(ctx) {
		edgeCount := result.Record().Values[0].(int64)
		log.Printf("✅ Neo4j now has %d CO_CHANGES_WITH edges", edgeCount)

		if edgeCount > 0 {
			// Show sample
			sampleQuery := `
				MATCH (a:CodeBlock)-[r:CO_CHANGES_WITH]->(b:CodeBlock)
				RETURN a.name, b.name, r.rate, r.co_change_count
				LIMIT 3
			`
			sampleResult, err := session.Run(ctx, sampleQuery, nil)
			if err == nil {
				log.Println("\n📋 Sample coupling edges:")
				for sampleResult.Next(ctx) {
					record := sampleResult.Record()
					aName := record.Values[0]
					bName := record.Values[1]
					rate := record.Values[2]
					count := record.Values[3]
					log.Printf("   %s <-> %s (rate: %.2f, count: %d)", aName, bName, rate, count)
				}
			}
		}
	}

	log.Println("\n✨ Migration complete!")
	if successCount > 0 {
		log.Println("   Next step: Update GetCouplingData in local_graph_client.go to query Neo4j")
		log.Println("   You can now test the MCP server with Neo4j coupling queries")
	} else {
		log.Println("   ⚠️  No edges were created. Check that CodeBlock nodes have db_id property set")
	}
}
