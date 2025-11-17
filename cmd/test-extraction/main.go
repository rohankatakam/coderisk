package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/github"
	"github.com/rohankatakam/coderisk/internal/llm"
)

func main() {
	ctx := context.Background()

	// Get API key
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY not set")
	}

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

	// Create LLM client
	llmClient := llm.NewClient(apiKey, "gemini")

	// Create extractor
	extractor := github.NewCommitExtractor(llmClient, stagingDB)

	// Run extraction
	fmt.Println("🔍 Running commit extraction for repo_id=1 (omnara-ai/omnara)...")
	fmt.Println("This will process commits in batches of 20 and log each batch.")
	fmt.Println()

	refsCount, err := extractor.ExtractCommitReferences(ctx, 1)
	if err != nil {
		log.Fatalf("Extraction failed: %v", err)
	}

	fmt.Printf("\n✅ Extraction complete: %d references extracted\n", refsCount)

	// Now check the specific commit we're investigating
	fmt.Println("\n" + "="*80)
	fmt.Println("🔍 Checking PR #1 → Commit d12fe4a edge...")
	fmt.Println("="*80)

	var result struct {
		IssueNumber    int
		CommitSHA      string
		Action         string
		Confidence     float64
		DetectionMethod string
		ExtractedFrom  string
	}

	err = stagingDB.QueryRow(ctx, `
		SELECT
			issue_number,
			commit_sha,
			action,
			confidence,
			detection_method,
			extracted_from
		FROM github_issue_commit_refs
		WHERE repo_id = 1
		  AND issue_number = 1
		  AND commit_sha LIKE 'd12fe4a%'
		  AND detection_method = 'commit_extraction'
	`).Scan(
		&result.IssueNumber,
		&result.CommitSHA,
		&result.Action,
		&result.Confidence,
		&result.DetectionMethod,
		&result.ExtractedFrom,
	)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			fmt.Println("✅ GOOD: No edge found (bug did not reproduce)")
		} else {
			log.Printf("Query error: %v", err)
		}
	} else {
		fmt.Println("⚠️  BAD: Edge still exists (bug reproduced)")
		fmt.Printf("\nExtracted reference:\n")
		fmt.Printf("  Issue/PR: #%d\n", result.IssueNumber)
		fmt.Printf("  Commit: %s\n", result.CommitSHA)
		fmt.Printf("  Action: %s\n", result.Action)
		fmt.Printf("  Confidence: %.2f\n", result.Confidence)
		fmt.Printf("  Method: %s\n", result.DetectionMethod)
		fmt.Printf("  From: %s\n", result.ExtractedFrom)

		// Get the actual commit message
		var commitMsg string
		err = stagingDB.QueryRow(ctx, `
			SELECT message
			FROM github_commits
			WHERE repo_id = 1 AND sha LIKE 'd12fe4a%'
		`).Scan(&commitMsg)

		if err == nil {
			fmt.Printf("\nActual commit message:\n")
			fmt.Printf("  |%s|\n", commitMsg)
			fmt.Printf("\n⚠️  Bug reproduced! LLM extracted #%d from commit with message '%s'\n",
				result.IssueNumber, commitMsg)
		}
	}
}
