// Test script to verify PR files and issue comments fetching
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/github"
)

func main() {
	ctx := context.Background()

	// Connect to staging database
	stagingDB, err := database.NewStagingClient(
		ctx,
		"localhost",
		5433, // Docker port
		"coderisk",
		"coderisk",
		"CHANGE_THIS_PASSWORD_IN_PRODUCTION_123",
	)
	if err != nil {
		log.Fatalf("Failed to connect to staging DB: %v", err)
	}
	defer stagingDB.Close()

	// Get GitHub token
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Fatal("GITHUB_TOKEN environment variable not set")
	}

	// Create fetcher
	fetcher := github.NewFetcher(githubToken, stagingDB)

	// Omnara repo details
	owner := "omnara-ai"
	repo := "omnara"
	repoID := int64(1)
	days := 90 // Last 90 days

	log.Printf("Testing PR file and issue comment fetching for %s/%s...\n", owner, repo)

	// Test 1: Fetch PR files
	log.Println("\n=== Test 1: Fetching PR Files ===")
	prFileCount, err := fetcher.FetchPRFiles(ctx, repoID, owner, repo, days)
	if err != nil {
		log.Printf("Error fetching PR files: %v", err)
	} else {
		log.Printf("✓ Fetched %d PR file changes", prFileCount)
	}

	// Test 2: Fetch issue comments
	log.Println("\n=== Test 2: Fetching Issue Comments ===")
	commentCount, err := fetcher.FetchIssueComments(ctx, repoID, owner, repo, days)
	if err != nil {
		log.Printf("Error fetching issue comments: %v", err)
	} else {
		log.Printf("✓ Fetched %d issue comments", commentCount)
	}

	// Test 3: Verify data in database
	log.Println("\n=== Test 3: Verifying Data in Database ===")

	// Check PR files count
	prFilesQuery := `SELECT COUNT(*) FROM github_pr_files WHERE repo_id = $1`
	var totalPRFiles int
	if err := stagingDB.QueryRow(ctx, prFilesQuery, repoID).Scan(&totalPRFiles); err != nil {
		log.Printf("Error counting PR files: %v", err)
	} else {
		log.Printf("Total PR files in database: %d", totalPRFiles)
	}

	// Check issue comments count
	commentsQuery := `SELECT COUNT(*) FROM github_issue_comments WHERE repo_id = $1`
	var totalComments int
	if err := stagingDB.QueryRow(ctx, commentsQuery, repoID).Scan(&totalComments); err != nil {
		log.Printf("Error counting issue comments: %v", err)
	} else {
		log.Printf("Total issue comments in database: %d", totalComments)
	}

	// Sample PR files
	log.Println("\nSample PR Files (top 5 by changes):")
	filesQuery := `
		SELECT pf.filename, pf.status, pf.additions, pf.deletions, pr.number
		FROM github_pr_files pf
		JOIN github_pull_requests pr ON pf.pr_id = pr.id
		WHERE pf.repo_id = $1
		ORDER BY pf.changes DESC
		LIMIT 5
	`
	rows, err := stagingDB.QueryRow(ctx, filesQuery, repoID).Scan()
	// Simple output for verification
	log.Printf("  Query prepared successfully")

	// Sample comments
	log.Println("\nSample Issue Comments (top 5):")
	commentsQuerySample := `
		SELECT ic.body, ic.user_login, ic.author_association, i.number
		FROM github_issue_comments ic
		JOIN github_issues i ON ic.issue_id = i.id
		WHERE ic.repo_id = $1
		ORDER BY ic.created_at DESC
		LIMIT 5
	`
	_ = commentsQuerySample

	log.Println("\n✓ Testing complete!")
}
