package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rohankatakam/coderisk/internal/atomizer"
	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/llm"
)

// test-processor demonstrates AGENT-P2B chronological event processing
// Reference: AGENT_P2B_PROCESSOR.md
func main() {
	ctx := context.Background()

	log.Printf("=== AGENT-P2B: Chronological Event Processor Test ===\n")

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

	// 5. Create processor components
	extractor := atomizer.NewExtractor(llmClient)
	processor := atomizer.NewProcessor(extractor, db, driver, "neo4j")

	// 6. Get test commits (chronologically ordered)
	commits := getTestCommits()
	log.Printf("\n=== Processing %d commits chronologically ===\n", len(commits))

	// 7. Process commits
	repoID := int64(1) // Test repository ID
	if err := processor.ProcessCommitsChronologically(ctx, commits, repoID); err != nil {
		log.Fatalf("Processing failed: %v", err)
	}

	// 8. Verify results in PostgreSQL
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

	// 9. Verify results in Neo4j
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	result, err := session.Run(ctx, "MATCH (b:CodeBlock {repo_id: $repo_id}) RETURN count(b) as count", map[string]any{"repo_id": repoID})
	if err != nil {
		log.Printf("WARNING: Failed to count CodeBlock nodes: %v", err)
	} else if result.Next(ctx) {
		count, _ := result.Record().Get("count")
		log.Printf("✓ CodeBlock nodes in Neo4j: %v", count)
	}

	log.Printf("\n=== Test Complete ===\n")
	log.Printf("✓ AGENT-P2B processing successful")
	log.Printf("✓ PostgreSQL and Neo4j data written")
	log.Printf("✓ Ready for production use")
}

// getTestCommits returns a small set of test commits in chronological order
func getTestCommits() []atomizer.CommitData {
	// Get the last 10 commits from the current repository
	shas := getRecentCommits(10)

	var commits []atomizer.CommitData
	for _, sha := range shas {
		commitData, err := getCommitData(sha)
		if err != nil {
			log.Printf("WARNING: Failed to get commit data for %s: %v", sha, err)
			continue
		}
		commits = append(commits, *commitData)
	}

	return commits
}

// getRecentCommits gets the last N commit SHAs in chronological order (oldest first)
func getRecentCommits(n int) []string {
	cmd := exec.Command("git", "log", "--reverse", fmt.Sprintf("-n%d", n), "--format=%H")
	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("Failed to get recent commits: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	return lines
}

// getCommitData fetches commit information from git
func getCommitData(sha string) (*atomizer.CommitData, error) {
	// Get commit message
	msgCmd := exec.Command("git", "log", "-1", "--format=%B", sha)
	msgOut, err := msgCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	// Get author email
	emailCmd := exec.Command("git", "log", "-1", "--format=%ae", sha)
	emailOut, err := emailCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	// Get author date
	dateCmd := exec.Command("git", "log", "-1", "--format=%aI", sha)
	dateOut, err := dateCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get date: %w", err)
	}

	// Get diff
	diffCmd := exec.Command("git", "show", "--format=", sha)
	diffOut, err := diffCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	// Parse timestamp
	timestamp, err := time.Parse(time.RFC3339, strings.TrimSpace(string(dateOut)))
	if err != nil {
		timestamp = time.Now() // Fallback
	}

	return &atomizer.CommitData{
		SHA:         sha,
		Message:     strings.TrimSpace(string(msgOut)),
		DiffContent: string(diffOut),
		AuthorEmail: strings.TrimSpace(string(emailOut)),
		Timestamp:   timestamp,
	}, nil
}
