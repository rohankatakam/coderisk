package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// This script fixes the timeline extraction bug by reprocessing "referenced" events
// to extract commit_id fields that were previously ignored

func main() {
	ctx := context.Background()

	// Connect to PostgreSQL
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	log.Println("🔧 Fixing timeline reference extraction...")

	// Get all "referenced" events with NULL source data
	query := `
		SELECT id, raw_data
		FROM github_issue_timeline
		WHERE event_type = 'referenced'
		  AND source_sha IS NULL
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		log.Fatalf("Failed to query timeline events: %v", err)
	}
	defer rows.Close()

	var eventIDs []int64
	var rawDataList [][]byte
	for rows.Next() {
		var id int64
		var rawData []byte
		if err := rows.Scan(&id, &rawData); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		eventIDs = append(eventIDs, id)
		rawDataList = append(rawDataList, rawData)
	}

	log.Printf("  Found %d referenced events to fix", len(eventIDs))

	if len(eventIDs) == 0 {
		log.Println("  ✓ No events need fixing")
		return
	}

	// Process each event
	fixedCount := 0
	for i, id := range eventIDs {
		var rawEvent map[string]interface{}
		if err := json.Unmarshal(rawDataList[i], &rawEvent); err != nil {
			log.Printf("  ⚠️  Failed to parse raw_data for event %d: %v", id, err)
			continue
		}

		// Extract commit_id if present
		commitID, ok := rawEvent["commit_id"].(string)
		if !ok || commitID == "" {
			continue
		}

		// Update the event with source_sha and source_type
		updateQuery := `
			UPDATE github_issue_timeline
			SET source_sha = $1,
			    source_type = 'commit'
			WHERE id = $2
		`

		_, err := pool.Exec(ctx, updateQuery, commitID, id)
		if err != nil {
			log.Printf("  ⚠️  Failed to update event %d: %v", id, err)
			continue
		}

		fixedCount++
	}

	log.Printf("  ✅ Fixed %d/%d referenced events", fixedCount, len(eventIDs))

	// Show statistics
	var stats struct {
		TotalReferenced   int
		WithCommitSHA     int
		UniqueCommits     int
		AffectedIssues    int
	}

	err = pool.QueryRow(ctx, `
		SELECT
			COUNT(*) as total_referenced,
			COUNT(source_sha) as with_commit_sha,
			COUNT(DISTINCT source_sha) as unique_commits,
			COUNT(DISTINCT issue_id) as affected_issues
		FROM github_issue_timeline
		WHERE event_type = 'referenced'
	`).Scan(&stats.TotalReferenced, &stats.WithCommitSHA, &stats.UniqueCommits, &stats.AffectedIssues)
	if err != nil {
		log.Fatalf("Failed to get statistics: %v", err)
	}

	fmt.Println("\n📊 Timeline Statistics:")
	fmt.Printf("  Total 'referenced' events: %d\n", stats.TotalReferenced)
	fmt.Printf("  Events with commit SHA: %d (%.1f%%)\n", stats.WithCommitSHA,
		float64(stats.WithCommitSHA)/float64(stats.TotalReferenced)*100)
	fmt.Printf("  Unique commits referenced: %d\n", stats.UniqueCommits)
	fmt.Printf("  Issues with references: %d\n", stats.AffectedIssues)
}
