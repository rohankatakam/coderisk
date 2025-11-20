package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
	ctx := context.Background()

	// Connect to Neo4j
	uri := os.Getenv("NEO4J_URI")
	if uri == "" {
		uri = "bolt://localhost:7688"
	}

	password := os.Getenv("NEO4J_PASSWORD")
	if password == "" {
		log.Fatal("NEO4J_PASSWORD environment variable must be set")
	}

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth("neo4j", password, ""))
	if err != nil {
		log.Fatalf("Failed to create driver: %v", err)
	}
	defer driver.Close(ctx)

	// Verify connection
	if err := driver.VerifyConnectivity(ctx); err != nil {
		log.Fatalf("Failed to verify connectivity: %v", err)
	}

	log.Println("✅ Connected to Neo4j")

	// Get repo_id from args or default to 18
	repoID := int64(18)
	if len(os.Args) > 1 {
		fmt.Sscanf(os.Args[1], "%d", &repoID)
	}

	log.Printf("🗑️  Deleting all Neo4j data for repo_id=%d...", repoID)

	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	// Delete all nodes and relationships for this repo
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (n {repo_id: $repoID})
			DETACH DELETE n
			RETURN count(n) as deleted_count
		`
		result, err := tx.Run(ctx, query, map[string]any{"repoID": repoID})
		if err != nil {
			return nil, err
		}

		record, err := result.Single(ctx)
		if err != nil {
			return nil, err
		}

		count, _ := record.Get("deleted_count")
		return count, nil
	})

	if err != nil {
		log.Fatalf("Failed to delete nodes: %v", err)
	}

	log.Printf("✅ Successfully wiped Neo4j data for repo_id=%d", repoID)

	// Verify deletion
	count, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `MATCH (n {repo_id: $repoID}) RETURN count(n) as count`
		result, err := tx.Run(ctx, query, map[string]any{"repoID": repoID})
		if err != nil {
			return nil, err
		}

		record, err := result.Single(ctx)
		if err != nil {
			return nil, err
		}

		count, _ := record.Get("count")
		return count, nil
	})

	if err != nil {
		log.Fatalf("Failed to verify deletion: %v", err)
	}

	log.Printf("📊 Remaining nodes for repo_id=%d: %v", repoID, count)
}
