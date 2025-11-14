package main

import (
	"context"
	"log"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/graph"
)

func main() {
	ctx := context.Background()

	log.Println("🧪 Testing graph construction with timeline edges...")

	// Connect to PostgreSQL
	stagingDB, err := database.NewStagingClient(
		ctx,
		"localhost",
		5433,
		"coderisk",
		"coderisk",
		"CHANGE_THIS_PASSWORD_IN_PRODUCTION_123",
	)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer stagingDB.Close()

	// Connect to Neo4j
	backend, err := graph.NewNeo4jBackend(
		ctx,
		"bolt://localhost:7688",
		"neo4j",
		"CHANGE_THIS_PASSWORD_IN_PRODUCTION_123",
		"neo4j",
	)
	if err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	defer backend.Close(ctx)

	// Check if graph already has data
	log.Println("📊 Checking existing graph data...")
	countQuery := "MATCH (n) RETURN COUNT(n) as count"
	countResults, err := backend.QueryWithParams(ctx, countQuery, nil)
	if err != nil {
		log.Fatalf("Failed to count nodes: %v", err)
	}

	var nodeCount int64
	if len(countResults) > 0 {
		if count, ok := countResults[0]["count"].(int64); ok {
			nodeCount = count
		}
	}

	if nodeCount > 0 {
		log.Printf("  ℹ️  Graph already contains %d nodes - skipping rebuild", nodeCount)
		log.Println("  ℹ️  To rebuild, manually clear the graph first")
	} else {
		// Build graph for repo_id=6 (omnara-ai/omnara)
		builder := graph.NewBuilder(stagingDB, backend)

		log.Println("🔨 Building graph from PostgreSQL data...")
		stats, err := builder.BuildGraph(ctx, 6, "/tmp/omnara")
		if err != nil {
			log.Fatalf("Failed to build graph: %v", err)
		}

		log.Printf("\n✅ Graph construction complete!")
		log.Printf("  Nodes created: %d", stats.Nodes)
		log.Printf("  Edges created: %d", stats.Edges)
	}

	// Run validation queries
	log.Println("\n🔍 Running validation queries...")

	// Check for duplicate PR→Commit edges
	dupPRCommit := `
		MATCH (pr:PR)-[m:MERGED_AS]->(c:Commit)
		MATCH (pr)-[a:ASSOCIATED_WITH]->(c)
		RETURN COUNT(*) as duplicates
	`
	results, err := backend.QueryWithParams(ctx, dupPRCommit, nil)
	if err != nil {
		log.Printf("  ⚠️  Failed to check PR→Commit duplicates: %v", err)
	} else if len(results) > 0 {
		if count, ok := results[0]["duplicates"].(int64); ok {
			if count == 0 {
				log.Printf("  ✅ No duplicate PR→Commit edges found")
			} else {
				log.Printf("  ❌ Found %d duplicate PR→Commit edges", count)
			}
		}
	}

	// Check for duplicate Issue→PR edges
	dupIssuePR := `
		MATCH (i:Issue)-[r:REFERENCES]->(pr:PR)
		MATCH (i)-[a:ASSOCIATED_WITH]->(pr)
		RETURN COUNT(*) as duplicates
	`
	results, err = backend.QueryWithParams(ctx, dupIssuePR, nil)
	if err != nil {
		log.Printf("  ⚠️  Failed to check Issue→PR duplicates: %v", err)
	} else if len(results) > 0 {
		if count, ok := results[0]["duplicates"].(int64); ok {
			if count == 0 {
				log.Printf("  ✅ No duplicate Issue→PR edges found")
			} else {
				log.Printf("  ❌ Found %d duplicate Issue→PR edges", count)
			}
		}
	}

	// Check for duplicate Issue→Commit edges
	dupIssueCommit := `
		MATCH (i:Issue)-[c:CLOSED_BY]->(commit:Commit)
		MATCH (i)-[a:ASSOCIATED_WITH]->(commit)
		RETURN COUNT(*) as duplicates
	`
	results, err = backend.QueryWithParams(ctx, dupIssueCommit, nil)
	if err != nil {
		log.Printf("  ⚠️  Failed to check Issue→Commit duplicates: %v", err)
	} else if len(results) > 0 {
		if count, ok := results[0]["duplicates"].(int64); ok {
			if count == 0 {
				log.Printf("  ✅ No duplicate Issue→Commit edges found")
			} else {
				log.Printf("  ❌ Found %d duplicate Issue→Commit edges", count)
			}
		}
	}

	// Count edges by type
	log.Println("\n📊 Edge statistics:")
	edgeStats := `
		MATCH ()-[r]->()
		RETURN type(r) as edge_type, COUNT(r) as count
		ORDER BY count DESC
	`
	results, err = backend.QueryWithParams(ctx, edgeStats, nil)
	if err != nil {
		log.Printf("  ⚠️  Failed to get edge statistics: %v", err)
	} else {
		for _, result := range results {
			edgeType := result["edge_type"].(string)
			count := result["count"].(int64)
			log.Printf("  %s: %d", edgeType, count)
		}
	}

	log.Println("\n✅ All tests complete!")
}
