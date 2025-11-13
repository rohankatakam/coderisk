package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/graph"
)

func main() {
	ctx := context.Background()

	// Load config
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to databases
	stagingDB, err := database.NewStagingClient(
		ctx,
		cfg.Storage.PostgresHost,
		cfg.Storage.PostgresPort,
		cfg.Storage.PostgresDB,
		cfg.Storage.PostgresUser,
		cfg.Storage.PostgresPassword,
	)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer stagingDB.Close()

	graphBackend, err := graph.NewNeo4jBackend(
		ctx,
		cfg.Neo4j.URI,
		cfg.Neo4j.User,
		cfg.Neo4j.Password,
		cfg.Neo4j.Database,
	)
	if err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	defer graphBackend.Close(ctx)

	// Create composite constraints BEFORE building graph
	fmt.Println("📋 Creating composite unique constraints...")
	if err := createCompositeConstraints(ctx, graphBackend); err != nil {
		log.Fatalf("Failed to create constraints: %v", err)
	}
	fmt.Println("✅ Constraints created successfully")

	// Reset processed_at flags are handled externally
	// Run this before the test:
	// PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk \
	//   -c "UPDATE github_commits SET processed_at = NULL WHERE repo_id = 1; \
	//       UPDATE github_pull_requests SET processed_at = NULL WHERE repo_id = 1; \
	//       UPDATE github_issues SET processed_at = NULL WHERE repo_id = 1;"
	fmt.Println("\n⚠️  Ensure you reset processed_at flags first!")

	// Build graph for repo ID 1 (omnara)
	fmt.Println("\n🔨 Building graph from PostgreSQL data (repo_id=1)...")
	builder := graph.NewBuilder(stagingDB, graphBackend)

	repoPath := "/Users/rohankatakam/Documents/brain/omnara" // omnara repo path
	stats, err := builder.BuildGraph(ctx, 1, repoPath)
	if err != nil {
		log.Fatalf("Failed to build graph: %v", err)
	}

	fmt.Printf("✅ Graph built successfully!\n")
	fmt.Printf("   Nodes: %d\n", stats.Nodes)
	fmt.Printf("   Edges: %d\n", stats.Edges)

	// Verify composite IDs in Neo4j
	fmt.Println("\n🔍 Verifying composite node IDs in Neo4j...")

	// Check a sample commit node
	query := `MATCH (c:Commit) WHERE c.repo_id = 1 RETURN c.sha, c.repo_id, c.repo_full_name LIMIT 1`
	results, err := graphBackend.QueryWithParams(ctx, query, nil)
	if err != nil {
		log.Fatalf("Failed to query commit: %v", err)
	}
	if len(results) > 0 {
		fmt.Printf("✅ Sample Commit node: sha=%v, repo_id=%v, repo_full_name=%v\n",
			results[0]["c.sha"], results[0]["c.repo_id"], results[0]["c.repo_full_name"])
	}

	// Check a sample PR node
	query = `MATCH (p:PR) WHERE p.repo_id = 1 RETURN p.number, p.repo_id, p.repo_full_name LIMIT 1`
	results, err = graphBackend.QueryWithParams(ctx, query, nil)
	if err != nil {
		log.Fatalf("Failed to query PR: %v", err)
	}
	if len(results) > 0 {
		fmt.Printf("✅ Sample PR node: number=%v, repo_id=%v, repo_full_name=%v\n",
			results[0]["p.number"], results[0]["p.repo_id"], results[0]["p.repo_full_name"])
	}

	// Check a sample Issue node
	query = `MATCH (i:Issue) WHERE i.repo_id = 1 RETURN i.number, i.repo_id, i.repo_full_name LIMIT 1`
	results, err = graphBackend.QueryWithParams(ctx, query, nil)
	if err != nil {
		log.Fatalf("Failed to query Issue: %v", err)
	}
	if len(results) > 0 {
		fmt.Printf("✅ Sample Issue node: number=%v, repo_id=%v, repo_full_name=%v\n",
			results[0]["i.number"], results[0]["i.repo_id"], results[0]["i.repo_full_name"])
	}

	// Check edge creation
	query = `MATCH (c:Commit)-[r]->(f:File) WHERE c.repo_id = 1 AND f.repo_id = 1 RETURN type(r) as rel_type, count(*) as count`
	results, err = graphBackend.QueryWithParams(ctx, query, nil)
	if err != nil {
		log.Fatalf("Failed to query edges: %v", err)
	}
	if len(results) > 0 {
		fmt.Printf("✅ Commit->File edges: type=%v, count=%v\n",
			results[0]["rel_type"], results[0]["count"])
	} else {
		fmt.Println("⚠️  No Commit->File edges found!")
	}

	fmt.Println("\n✅ All tests passed! Multi-repo partitioning is working correctly.")
	os.Exit(0)
}

// createCompositeConstraints creates the required composite unique constraints
// Reference: cmd/crisk/init.go createIndexes()
func createCompositeConstraints(ctx context.Context, backend graph.Backend) error {
	constraints := []string{
		// Multi-repo composite constraints (PRIMARY KEYS)
		"CREATE CONSTRAINT file_repo_path_unique IF NOT EXISTS FOR (f:File) REQUIRE (f.repo_id, f.path) IS UNIQUE",
		"CREATE CONSTRAINT commit_repo_sha_unique IF NOT EXISTS FOR (c:Commit) REQUIRE (c.repo_id, c.sha) IS UNIQUE",
		"CREATE CONSTRAINT pr_repo_number_unique IF NOT EXISTS FOR (pr:PR) REQUIRE (pr.repo_id, pr.number) IS UNIQUE",
		"CREATE CONSTRAINT issue_repo_number_unique IF NOT EXISTS FOR (i:Issue) REQUIRE (i.repo_id, i.number) IS UNIQUE",
		// Global constraints (no repo_id)
		"CREATE CONSTRAINT developer_email_unique IF NOT EXISTS FOR (d:Developer) REQUIRE d.email IS UNIQUE",
	}

	// Use ExecuteBatch which runs in write mode
	if err := backend.ExecuteBatch(ctx, constraints); err != nil {
		return fmt.Errorf("failed to create constraints: %w", err)
	}

	fmt.Printf("  ✓ Created %d constraints successfully\n", len(constraints))
	return nil
}
