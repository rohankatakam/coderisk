// Test semantic filtering in temporal correlation
package main

import (
	"context"
	"log"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/graph"
)

func main() {
	ctx := context.Background()

	log.Println("🧪 Testing Semantic Filtering in Temporal Correlation")
	log.Println("══════════════════════════════════════════════════════")

	// Connect to PostgreSQL
	stagingDB, err := database.NewStagingClient(ctx, "localhost", 5433, "coderisk", "coderisk", "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123")
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer stagingDB.Close()
	log.Println("  ✓ Connected to PostgreSQL")

	// Get repository ID
	repoID, err := stagingDB.GetRepositoryID(ctx, "omnara-ai/omnara")
	if err != nil {
		log.Fatalf("Failed to get repository ID: %v", err)
	}
	log.Printf("  ✓ Found repository ID: %d", repoID)

	// Clear existing temporal matches
	log.Println("\n🗑️  Clearing existing temporal matches...")
	_, err = stagingDB.Query(ctx, "DELETE FROM github_issue_commit_refs WHERE repo_id = $1 AND detection_method = 'temporal'", repoID)
	if err != nil {
		log.Fatalf("Failed to clear temporal matches: %v", err)
	}

	// Create temporal correlator (now with semantic filtering!)
	correlator := graph.NewTemporalCorrelator(stagingDB, nil)

	// Find temporal matches with semantic filtering
	log.Println("\n🔍 Finding temporal matches with semantic filtering...")
	matches, err := correlator.FindTemporalMatches(ctx, repoID)
	if err != nil {
		log.Fatalf("Failed to find temporal matches: %v", err)
	}

	log.Printf("\n✓ Found %d temporal matches after semantic filtering", len(matches))

	// Store matches in database
	log.Println("\n💾 Storing filtered temporal matches...")
	if err := correlator.StoreTemporalMatches(ctx, repoID, matches); err != nil {
		log.Fatalf("Failed to store temporal matches: %v", err)
	}

	// Show statistics for key test cases
	log.Println("\n📊 Checking key test cases:")

	testCases := []int{221, 189, 187, 219}
	for _, issueNum := range testCases {
		rows, err := stagingDB.Query(ctx, "SELECT COUNT(*) FROM github_issue_commit_refs WHERE repo_id = $1 AND issue_number = $2 AND detection_method = 'temporal'", repoID, issueNum)
		if err != nil {
			log.Printf("  ⚠️  Failed to query issue #%d: %v", issueNum, err)
			continue
		}

		var count int
		if rows.Next() {
			rows.Scan(&count)
		}
		rows.Close()

		if issueNum == 219 {
			if count == 0 {
				log.Printf("  ✅ Issue #219: %d matches (CORRECTLY REJECTED!)", count)
			} else {
				log.Printf("  ❌ Issue #219: %d matches (should be 0 - false positive!)", count)
			}
		} else {
			log.Printf("  ✓ Issue #%d: %d matches", issueNum, count)
		}
	}

	log.Println("\n✅ Semantic filtering test complete!")
	log.Println("══════════════════════════════════════════════════════")
}
