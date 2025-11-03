package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/graph"
)

func main() {
	ctx := context.Background()

	// Connect to PostgreSQL
	stagingDB, err := database.NewStagingClient(ctx, "localhost", 5433, "coderisk", "coderisk", "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123")
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer stagingDB.Close()

	// Get Issue #189
	repoID := int64(1)
	issues, err := stagingDB.GetClosedIssues(ctx, repoID)
	if err != nil {
		log.Fatalf("Failed to get issues: %v", err)
	}

	var issue189 *database.IssueData
	for i := range issues {
		if issues[i].Number == 189 {
			issue189 = &issues[i]
			break
		}
	}

	if issue189 == nil {
		log.Fatal("Issue #189 not found")
	}

	fmt.Printf("Issue #189:\n")
	fmt.Printf("  Title: %s\n", issue189.Title)
	fmt.Printf("  Body: %s\n", issue189.Body)
	fmt.Printf("  Closed: %v\n", issue189.ClosedAt)

	// Get PRs merged near issue close time
	prs, err := stagingDB.GetPRsMergedNear(ctx, repoID, *issue189.ClosedAt, 24*time.Hour)
	if err != nil {
		log.Fatalf("Failed to get PRs: %v", err)
	}

	fmt.Printf("\nPRs merged within 24 hours:\n")
	matcher := graph.NewSemanticMatcher()
	issueText := issue189.Title + " " + issue189.Body

	for _, pr := range prs {
		if pr.MergedAt == nil {
			continue
		}

		delta := issue189.ClosedAt.Sub(*pr.MergedAt)
		if delta < 0 {
			delta = -delta
		}

		prText := pr.Title + " " + pr.Body
		similarity := matcher.CalculateSimilarity(issueText, prText)

		if pr.Number == 203 {
			fmt.Printf("\n🎯 PR #%d: %s\n", pr.Number, pr.Title)
			fmt.Printf("  Body: %q\n", pr.Body)
			fmt.Printf("  Merged: %v\n", *pr.MergedAt)
			fmt.Printf("  Delta: %v (%.1f hours)\n", delta, delta.Hours())
			fmt.Printf("  Semantic similarity: %.4f (%.1f%%)\n", similarity, similarity*100)

			if similarity < 0.10 && delta >= 1*time.Hour {
				fmt.Printf("  ❌ REJECTED: similarity < 10%% and delta >= 1 hour\n")
			} else {
				fmt.Printf("  ✅ ACCEPTED\n")
			}
		}
	}
}
