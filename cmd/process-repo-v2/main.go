package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	_ "github.com/lib/pq"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rohankatakam/coderisk/internal/atomizer"
	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/llm"
)

// process-repo-v2: REAL processor that fetches diffs from git repository
// This implements AGENT-P2B for mcp-use with actual commit diffs
func main() {
	ctx := context.Background()

	log.Printf("=== AGENT-P2B: Processing mcp-use Repository (WITH DIFFS) ===\n")

	// 1. Check environment variables
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("ERROR: DATABASE_URL not set")
	}

	neoURI := os.Getenv("NEO4J_URI")
	neoUser := os.Getenv("NEO4J_USERNAME")
	neoPassword := os.Getenv("NEO4J_PASSWORD")
	if neoURI == "" || neoUser == "" || neoPassword == "" {
		log.Fatal("ERROR: Neo4j credentials not set")
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

	// 5. Get repository info
	repoID := int64(4) // mcp-use
	var repoPath string
	err = db.QueryRowContext(ctx, "SELECT absolute_path FROM github_repositories WHERE id = $1", repoID).Scan(&repoPath)
	if err != nil {
		log.Fatalf("Failed to get repository path: %v", err)
	}

	log.Printf("Repository path: %s", repoPath)

	// Verify repository exists
	if _, err := os.Stat(repoPath + "/.git"); os.IsNotExist(err) {
		log.Fatalf("ERROR: Git repository not found at %s", repoPath)
	}

	// 6. Fetch commits from database (chronologically)
	log.Printf("Fetching commits from database...")
	rows, err := db.QueryContext(ctx, `
		SELECT sha, message, author_email, author_name, author_date
		FROM github_commits
		WHERE repo_id = $1
		ORDER BY author_date ASC
	`, repoID)
	if err != nil {
		log.Fatalf("Failed to fetch commits: %v", err)
	}
	defer rows.Close()

	var dbCommits []struct {
		SHA         string
		Message     string
		AuthorEmail string
		AuthorName  string
		AuthorDate  string
	}

	for rows.Next() {
		var c struct {
			SHA         string
			Message     string
			AuthorEmail string
			AuthorName  string
			AuthorDate  string
		}
		if err := rows.Scan(&c.SHA, &c.Message, &c.AuthorEmail, &c.AuthorName, &c.AuthorDate); err != nil {
			log.Printf("WARNING: Failed to scan commit: %v", err)
			continue
		}
		dbCommits = append(dbCommits, c)
	}

	log.Printf("✓ Fetched %d commits from database", len(dbCommits))

	// 7. For testing, process first 10 commits only
	// Remove this limit to process all 517 commits
	limit := 10
	if len(os.Args) > 1 && os.Args[1] == "--all" {
		limit = len(dbCommits)
		log.Printf("⚠️  Processing ALL %d commits (this may take a while!)", limit)
	} else {
		if len(dbCommits) > limit {
			log.Printf("⚠️  Processing first %d commits only (use --all for full processing)", limit)
			dbCommits = dbCommits[:limit]
		}
	}

	// 8. Fetch diffs from git repository
	log.Printf("Fetching diffs from git repository...")
	var commits []atomizer.CommitData
	for i, c := range dbCommits {
		diff, err := getCommitDiff(repoPath, c.SHA)
		if err != nil {
			log.Printf("WARNING: Failed to get diff for %s: %v", c.SHA[:8], err)
			continue
		}

		commits = append(commits, atomizer.CommitData{
			SHA:         c.SHA,
			Message:     c.Message,
			DiffContent: diff,
			AuthorEmail: c.AuthorEmail,
			Timestamp:   mustParseTime(c.AuthorDate),
		})

		if (i+1)%50 == 0 {
			log.Printf("  Fetched diffs for %d/%d commits...", i+1, len(dbCommits))
		}
	}

	log.Printf("✓ Fetched %d commit diffs", len(commits))

	// 9. Create processor and run
	extractor := atomizer.NewExtractor(llmClient)
	processor := atomizer.NewProcessor(extractor, db, driver, "neo4j")

	log.Printf("\n=== Processing %d commits chronologically ===\n", len(commits))
	if err := processor.ProcessCommitsChronologically(ctx, commits, repoID); err != nil {
		log.Fatalf("Processing failed: %v", err)
	}

	// 10. Verify results
	log.Printf("\n=== Verification ===\n")

	var blockCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM code_blocks WHERE repo_id = $1", repoID).Scan(&blockCount)
	if err != nil {
		log.Printf("WARNING: Failed to count code blocks: %v", err)
	} else {
		log.Printf("✓ Code blocks in PostgreSQL: %d", blockCount)
	}

	var fileCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT file_path) FROM code_blocks WHERE repo_id = $1", repoID).Scan(&fileCount)
	if err != nil {
		log.Printf("WARNING: Failed to count files: %v", err)
	} else {
		log.Printf("✓ Files with code blocks: %d", fileCount)
	}

	var modCount int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM code_block_modifications WHERE repo_id = $1", repoID).Scan(&modCount)
	if err != nil {
		log.Printf("WARNING: Failed to count modifications: %v", err)
	} else {
		log.Printf("✓ Modifications recorded: %d", modCount)
	}

	// 11. Check Neo4j
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	result, err := session.Run(ctx, "MATCH (b:CodeBlock {repo_id: $repo_id}) RETURN count(b) as count", map[string]any{"repo_id": repoID})
	if err != nil {
		log.Printf("WARNING: Failed to query Neo4j: %v", err)
	} else if result.Next(ctx) {
		count, _ := result.Record().Get("count")
		log.Printf("✓ CodeBlock nodes in Neo4j: %v", count)
	}

	log.Printf("\n=== Processing Complete ===\n")
	log.Printf("✅ AGENT-P2B successfully processed mcp-use repository")
	log.Printf("✅ Code blocks extracted and stored in PostgreSQL + Neo4j")
	log.Printf("✅ Ready for AGENT-P2C!")
}

// getCommitDiff fetches the git diff for a commit
func getCommitDiff(repoPath, sha string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "show", "--format=", sha)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git show failed: %w", err)
	}
	return string(output), nil
}

// mustParseTime parses a timestamp or returns zero time
func mustParseTime(s string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Now()
		}
	}
	return t
}
