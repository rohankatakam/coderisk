package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/graph"
)

func main() {
	ctx := context.Background()

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

	fmt.Println("🔍 Investigating Edge: (PR #1)-[:ASSOCIATED_WITH]->(Commit d12fe4a)")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	// 1. Get the actual commit from PostgreSQL
	fmt.Println("📊 PostgreSQL Data:")
	fmt.Println(strings.Repeat("-", 80))

	var commit struct {
		SHA     string
		Message string
		AuthorName  *string
		AuthorEmail *string
		CommittedAt *int64
	}

	err = stagingDB.QueryRow(ctx, `
		SELECT sha, message, author_name, author_email,
		       EXTRACT(EPOCH FROM committer_date)::bigint as committed_at
		FROM github_commits
		WHERE repo_id = 1 AND sha LIKE 'd12fe4a%'
	`).Scan(&commit.SHA, &commit.Message, &commit.AuthorName, &commit.AuthorEmail, &commit.CommittedAt)

	if err != nil {
		log.Fatalf("Failed to get commit from PostgreSQL: %v", err)
	}

	fmt.Printf("Commit SHA: %s\n", commit.SHA)
	fmt.Printf("Commit Message:\n  |%s|\n", commit.Message)
	fmt.Printf("  (Length: %d characters)\n", len(commit.Message))

	authorName := "(null)"
	if commit.AuthorName != nil {
		authorName = *commit.AuthorName
	}
	authorEmail := "(null)"
	if commit.AuthorEmail != nil {
		authorEmail = *commit.AuthorEmail
	}
	fmt.Printf("Author: %s <%s>\n", authorName, authorEmail)

	if commit.CommittedAt != nil {
		fmt.Printf("Timestamp: %d\n", *commit.CommittedAt)
	} else {
		fmt.Printf("Timestamp: (null)\n")
	}
	fmt.Println()

	// 2. Get PR #1 data
	var pr struct {
		Number int
		Title  string
		Body   *string
		State  string
		CreatedAt int64
		MergedAt  *int64
	}

	err = stagingDB.QueryRow(ctx, `
		SELECT number, title, body, state,
		       EXTRACT(EPOCH FROM created_at)::bigint as created_at,
		       EXTRACT(EPOCH FROM merged_at)::bigint as merged_at
		FROM github_pull_requests
		WHERE repo_id = 1 AND number = 1
	`).Scan(&pr.Number, &pr.Title, &pr.Body, &pr.State, &pr.CreatedAt, &pr.MergedAt)

	if err != nil {
		log.Fatalf("Failed to get PR from PostgreSQL: %v", err)
	}

	fmt.Printf("PR #%d: %s\n", pr.Number, pr.Title)
	fmt.Printf("State: %s\n", pr.State)
	fmt.Printf("Created: %d\n", pr.CreatedAt)
	if pr.MergedAt != nil {
		fmt.Printf("Merged: %d\n", *pr.MergedAt)
	}
	if pr.Body != nil {
		fmt.Printf("Body:\n  |%s|\n", *pr.Body)
		fmt.Printf("  (Length: %d characters)\n", len(*pr.Body))
	} else {
		fmt.Println("Body: (null)")
	}
	fmt.Println()

	// 3. Check the extracted references table
	fmt.Println("📋 Extraction Results (github_issue_commit_refs):")
	fmt.Println(strings.Repeat("-", 80))

	rows, err := stagingDB.Query(ctx, `
		SELECT
			issue_number,
			commit_sha,
			pr_number,
			action,
			confidence,
			detection_method,
			extracted_from,
			evidence,
			created_at
		FROM github_issue_commit_refs
		WHERE repo_id = 1
		  AND (issue_number = 1 OR pr_number = 1)
		  AND commit_sha LIKE 'd12fe4a%'
		ORDER BY created_at DESC
	`)
	if err != nil {
		log.Fatalf("Failed to query refs: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var ref struct {
			IssueNumber     int
			CommitSHA       *string
			PRNumber        *int
			Action          string
			Confidence      float64
			DetectionMethod string
			ExtractedFrom   string
			Evidence        []byte
			CreatedAt       int64
		}

		err := rows.Scan(
			&ref.IssueNumber,
			&ref.CommitSHA,
			&ref.PRNumber,
			&ref.Action,
			&ref.Confidence,
			&ref.DetectionMethod,
			&ref.ExtractedFrom,
			&ref.Evidence,
			&ref.CreatedAt,
		)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		count++
		fmt.Printf("\nReference %d:\n", count)
		fmt.Printf("  Issue Number: %d\n", ref.IssueNumber)
		if ref.CommitSHA != nil {
			fmt.Printf("  Commit SHA: %s\n", *ref.CommitSHA)
		}
		if ref.PRNumber != nil {
			fmt.Printf("  PR Number: %d\n", *ref.PRNumber)
		}
		fmt.Printf("  Action: %s\n", ref.Action)
		fmt.Printf("  Confidence: %.2f\n", ref.Confidence)
		fmt.Printf("  Detection Method: %s\n", ref.DetectionMethod)
		fmt.Printf("  Extracted From: %s\n", ref.ExtractedFrom)

		var evidence []interface{}
		if len(ref.Evidence) > 0 {
			json.Unmarshal(ref.Evidence, &evidence)
		}
		fmt.Printf("  Evidence: %v\n", evidence)
		fmt.Printf("  Created At: %d\n", ref.CreatedAt)
	}

	if count == 0 {
		fmt.Println("  ⚠️  No references found in extraction table!")
	}
	fmt.Println()

	// 4. Check Neo4j edge
	fmt.Println("🔗 Neo4j Edge Data:")
	fmt.Println(strings.Repeat("-", 80))

	results, err := backend.QueryWithParams(ctx, `
		MATCH (pr:PR {number: 1, repo_id: 1})-[r:ASSOCIATED_WITH]->(c:Commit)
		WHERE c.sha STARTS WITH 'd12fe4a'
		RETURN pr.number, pr.title, c.sha, c.message,
		       r.relationship_type, r.confidence, r.detected_via, r.rationale
	`, nil)
	if err != nil {
		log.Fatalf("Failed to query Neo4j: %v", err)
	}

	if len(results) == 0 {
		fmt.Println("  ⚠️  No edge found in Neo4j!")
	} else {
		for _, row := range results {
			fmt.Printf("PR #%v → Commit %v\n", row["pr.number"], row["c.sha"])
			fmt.Printf("  Commit Message: |%v|\n", row["c.message"])
			fmt.Printf("  relationship_type: %v\n", row["r.relationship_type"])
			fmt.Printf("  confidence: %v\n", row["r.confidence"])
			fmt.Printf("  detected_via: %v\n", row["r.detected_via"])
			fmt.Printf("  rationale: %v\n", row["r.rationale"])
		}
	}
	fmt.Println()

	// 5. Check if this was actually extracted from commit message
	fmt.Println("🤔 Analysis:")
	fmt.Println(strings.Repeat("-", 80))

	if len(commit.Message) == 0 || commit.Message == "Initial commit" {
		fmt.Println("⚠️  ISSUE DETECTED:")
		fmt.Println("  The commit message is empty or generic ('Initial commit')")
		fmt.Println("  There is NO way the LLM could have extracted PR #1 from this message!")
		fmt.Println()
		fmt.Println("  Possible causes:")
		fmt.Println("  1. Bug in extraction logic (extracted from wrong field)")
		fmt.Println("  2. Temporal correlation mislabeled as 'commit_extraction'")
		fmt.Println("  3. Extraction happened on different data than what's in database")
	} else if !contains(commit.Message, "#1") && !contains(commit.Message, "PR 1") {
		fmt.Println("⚠️  ISSUE DETECTED:")
		fmt.Printf("  The commit message doesn't contain '#1' or 'PR 1'\n")
		fmt.Printf("  Message: |%s|\n", commit.Message)
		fmt.Println("  LLM should NOT have extracted this reference!")
	} else {
		fmt.Println("✅ Commit message contains reference to PR #1")
		fmt.Printf("  Message: %s\n", commit.Message)
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		   (len(s) >= len(substr)) &&
		   (s == substr || len(s) > len(substr) &&
		   (bytes.Contains([]byte(s), []byte(substr))))
}
