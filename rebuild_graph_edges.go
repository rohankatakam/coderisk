// Rebuild Neo4j graph edges from PostgreSQL IssueCommitRefs
package main

import (
	"context"
	"log"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/graph"
)

func main() {
	ctx := context.Background()

	log.Println("🔄 Rebuilding Neo4j Graph Edges")
	log.Println("═══════════════════════════════")

	// Connect to PostgreSQL
	stagingDB, err := database.NewStagingClient(ctx, "localhost", 5433, "coderisk", "coderisk", "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123")
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer stagingDB.Close()
	log.Println("  ✓ Connected to PostgreSQL")

	// Connect to Neo4j
	neo4jDB, err := graph.NewNeo4jBackend(ctx, "bolt://localhost:7688", "neo4j", "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123", "neo4j")
	if err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	defer neo4jDB.Close(ctx)
	log.Println("  ✓ Connected to Neo4j")

	// Get repository ID
	repoID := int64(1)
	log.Printf("\n📊 Rebuilding edges for repo ID: %d\n", repoID)

	// Use IssueLinker to create edges from PostgreSQL refs
	linker := graph.NewIssueLinker(stagingDB, neo4jDB)
	stats, err := linker.LinkIssues(ctx, repoID)
	if err != nil {
		log.Fatalf("Failed to link issues: %v", err)
	}

	log.Printf("\n✅ Graph Rebuild Complete!")
	log.Printf("  • Edges created: %d", stats.Edges)
	log.Printf("  • Nodes created: %d", stats.Nodes)
	log.Println("═══════════════════════════════")
}
