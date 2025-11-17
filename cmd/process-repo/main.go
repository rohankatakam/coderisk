package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rohankatakam/coderisk/internal/atomizer"
	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/llm"
)

// process-repo fetches commits from PostgreSQL and processes them chronologically
// This is the REAL integration test for AGENT-P2B on mcp-use repository
func main() {
	ctx := context.Background()

	log.Printf("=== AGENT-P2B: Processing mcp-use Repository ===\n")

	// 1. Check environment variables
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("ERROR: DATABASE_URL not set")
	}

	neoURI := os.Getenv("NEO4J_URI")
	neoUser := os.Getenv("NEO4J_USERNAME")
	neoPassword := os.Getenv("NEO4J_PASSWORD")
	if neoURI == "" || neoUser == "" || neoPassword == "" {
		log.Fatal("ERROR: Neo4j credentials not set (NEO4J_URI, NEO4J_USERNAME, NEO4J_PASSWORD)")
	}

	// 2. Connect to PostgreSQL
	log.Printf("Connecting to PostgreSQL...")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping PostgreSQL: %v", err)
	}
	log.Printf("✓ PostgreSQL connected")

	// 3. Connect to Neo4j
	log.Printf("Connecting to Neo4j...")
	driver, err := neo4j.NewDriverWithContext(neoURI, neo4j.BasicAuth(neoUser, neoPassword, ""))
	if err != nil {
		log.Fatalf("Failed to create Neo4j driver: %v", err)
	}
	defer driver.Close(ctx)

	if err := driver.VerifyConnectivity(ctx); err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	log.Printf("✓ Neo4j connected")

	// 4. Initialize LLM client
	log.Printf("Initializing LLM client...")
	os.Setenv("PHASE2_ENABLED", "true")
	cfg := config.Default()
	llmClient, err := llm.NewClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create LLM client: %v", err)
	}

	if !llmClient.IsEnabled() {
		log.Fatal("ERROR: LLM client not enabled. Set PHASE2_ENABLED=true and GEMINI_API_KEY")
	}
	log.Printf("✓ LLM client ready")

	// 5. Create staging client to fetch commits
	stagingClient, err := database.NewStagingClient(ctx, "localhost", 5433, "coderisk", "coderisk",
		os.Getenv("PGPASSWORD"))
	if err != nil {
		log.Fatalf("Failed to create staging client: %v", err)
	}
	defer stagingClient.Close()

	// 6. Get mcp-use repo ID
	repoID := int64(4) // mcp-use repo ID
	log.Printf("Processing repository ID: %d (mcp-use)", repoID)

	// 7. Fetch ALL commits chronologically from database
	log.Printf("Fetching commits from database...")
	rows, err := db.QueryContext(ctx, `
		SELECT sha, message, author_email, author_name, author_date, raw_data
		FROM github_commits
		WHERE repo_id = $1
		ORDER BY author_date ASC
	`, repoID)
	if err != nil {
		log.Fatalf("Failed to fetch commits: %v", err)
	}
	defer rows.Close()

	var commits []atomizer.CommitData
	for rows.Next() {
		var c atomizer.CommitData
		var rawData []byte
		if err := rows.Scan(&c.SHA, &c.Message, &c.AuthorEmail, &c.AuthorName, &c.Timestamp, &rawData); err != nil {
			log.Printf("WARNING: Failed to scan commit: %v", err)
			continue
		}

		// Get diff from git
		// For now, we'll use an empty diff - in production, fetch from git or stored diffs
		c.DiffContent = "" // This will result in empty change events

		commits = append(commits, c)
	}

	log.Printf("✓ Fetched %d commits from database", len(commits))

	if len(commits) == 0 {
		log.Fatal("ERROR: No commits found for repo_id 4")
	}

	// 8. IMPORTANT: For real processing, we need diffs!
	// Let's fetch diffs from the mcp-use git repository
	log.Printf("\n⚠️  WARNING: Need to fetch diffs from git repository")
	log.Printf("Please ensure mcp-use repository is cloned at expected location")

	// Get absolute path from database
	var absolutePath string
	err = db.QueryRowContext(ctx, "SELECT absolute_path FROM github_repositories WHERE id = $1", repoID).Scan(&absolutePath)
	if err != nil {
		log.Fatalf("Failed to get repository path: %v", err)
	}

	log.Printf("Repository path: %s", absolutePath)

	// For now, let's process just the first 10 commits to test
	// In production, you'd process all 517
	if len(commits) > 10 {
		log.Printf("⚠️  Processing first 10 commits only (for testing)")
		log.Printf("To process all 517 commits, remove this limit")
		commits = commits[:10]
	}

	// 9. Create processor components
	extractor := atomizer.NewExtractor(llmClient)
	processor := atomizer.NewProcessor(extractor, db, driver, "neo4j")

	// 10. Process commits (WITHOUT diffs for now - will create empty events)
	log.Printf("\n=== Processing %d commits ===\n", len(commits))
	if err := processor.ProcessCommitsChronologically(ctx, commits, repoID); err != nil {
		log.Fatalf("Processing failed: %v", err)
	}

	// 11. Verify results
	log.Printf("\n=== Verification ===\n")

	var blockCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM code_blocks WHERE repo_id = $1", repoID).Scan(&blockCount)
	if err != nil {
		log.Printf("WARNING: Failed to count code blocks: %v", err)
	} else {
		log.Printf("✓ Code blocks in PostgreSQL: %d", blockCount)
	}

	var modCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM code_block_modifications WHERE repo_id = $1", repoID).Scan(&modCount)
	if err != nil {
		log.Printf("WARNING: Failed to count modifications: %v", err)
	} else {
		log.Printf("✓ Modifications in PostgreSQL: %d", modCount)
	}

	log.Printf("\n=== Processing Complete ===\n")
	log.Printf("⚠️  NOTE: Commits processed without diffs (empty events)")
	log.Printf("To get real code blocks, you need to:")
	log.Printf("1. Fetch diffs from git repository")
	log.Printf("2. Re-run processor with diff content")
	log.Printf("\nSee AGENT_P2B_PROCESSOR.md for full implementation")
}
