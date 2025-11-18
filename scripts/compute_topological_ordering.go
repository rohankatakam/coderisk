package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/database"
	gitops "github.com/rohankatakam/coderisk/internal/git"
)

// Script to compute and store topological ordering for existing repository data
// Usage: go run scripts/compute_topological_ordering.go <repo_id>

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <repo_id>\n", os.Args[0])
		os.Exit(1)
	}

	repoIDStr := os.Args[1]
	var repoID int64
	if _, err := fmt.Sscanf(repoIDStr, "%d", &repoID); err != nil {
		log.Fatalf("Invalid repo_id: %v", err)
	}

	ctx := context.Background()

	// Load config
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to PostgreSQL
	fmt.Printf("Connecting to PostgreSQL...\n")
	stagingDB, err := database.NewStagingClient(
		ctx,
		cfg.Storage.PostgresHost,
		cfg.Storage.PostgresPort,
		cfg.Storage.PostgresDB,
		cfg.Storage.PostgresUser,
		cfg.Storage.PostgresPassword,
	)
	if err != nil {
		log.Fatalf("PostgreSQL connection failed: %v", err)
	}
	defer stagingDB.Close()

	// Get repository info
	var fullName, repoPath string
	row := stagingDB.QueryRow(ctx, "SELECT full_name, absolute_path FROM github_repositories WHERE id = $1", repoID)
	if err := row.Scan(&fullName, &repoPath); err != nil {
		log.Fatalf("Failed to get repository info: %v", err)
	}

	fmt.Printf("Repository: %s (ID: %d)\n", fullName, repoID)
	fmt.Printf("Path: %s\n\n", repoPath)

	// Compute topological ordering
	fmt.Printf("[1/4] Computing topological ordering...\n")
	sorter := gitops.NewTopologicalSorter(repoPath)

	topoOrder, err := sorter.ComputeTopologicalOrder(ctx)
	if err != nil {
		log.Fatalf("Failed to compute topological order: %v", err)
	}
	fmt.Printf("  ✓ Computed ordering for %d commits\n\n", len(topoOrder))

	// Compute parent SHAs hash for force-push detection
	fmt.Printf("[2/4] Computing parent SHAs hash...\n")
	parentHash, err := sorter.ComputeParentSHAsHash(ctx)
	if err != nil {
		log.Fatalf("Failed to compute parent hash: %v", err)
	}
	fmt.Printf("  ✓ Parent hash: %s\n\n", parentHash[:16]+"...")

	// Update database: topological_index for commits
	fmt.Printf("[3/4] Updating commit topological indexes...\n")
	db := stagingDB.DB()
	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	updateCount := 0
	for sha, index := range topoOrder {
		result, err := tx.Exec(
			"UPDATE github_commits SET topological_index = $1 WHERE repo_id = $2 AND sha = $3",
			index, repoID, sha)
		if err != nil {
			log.Fatalf("Failed to update commit %s: %v", sha, err)
		}

		rows, _ := result.RowsAffected()
		updateCount += int(rows)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}
	fmt.Printf("  ✓ Updated %d commits\n\n", updateCount)

	// Update repository with parent_shas_hash
	fmt.Printf("[4/4] Updating repository metadata...\n")
	_, err = db.Exec(
		"UPDATE github_repositories SET parent_shas_hash = $1 WHERE id = $2",
		parentHash, repoID)
	if err != nil {
		log.Fatalf("Failed to update repository: %v", err)
	}
	fmt.Printf("  ✓ Repository metadata updated\n\n")

	// Validation
	fmt.Printf("📊 Validation:\n")

	// Count commits with topological_index
	var indexedCount int
	row = stagingDB.QueryRow(ctx,
		"SELECT COUNT(*) FROM github_commits WHERE repo_id = $1 AND topological_index IS NOT NULL",
		repoID)
	row.Scan(&indexedCount)

	// Total commits
	var totalCount int
	row = stagingDB.QueryRow(ctx,
		"SELECT COUNT(*) FROM github_commits WHERE repo_id = $1",
		repoID)
	row.Scan(&totalCount)

	fmt.Printf("  Commits with topological_index: %d / %d\n", indexedCount, totalCount)

	if indexedCount == totalCount {
		fmt.Printf("\n✅ Topological ordering complete!\n")
	} else {
		fmt.Printf("\n⚠️  Warning: Some commits missing topological_index\n")
	}

	// Update parent_shas for commits (if missing)
	fmt.Printf("\n[Optional] Populating parent_shas column...\n")
	commits, err := stagingDB.Query(ctx,
		"SELECT sha FROM github_commits WHERE repo_id = $1 AND (parent_shas IS NULL OR array_length(parent_shas, 1) IS NULL)",
		repoID)
	if err != nil {
		log.Printf("  Failed to query commits: %v", err)
		return
	}
	defer commits.Close()

	var shasToUpdate []string
	for commits.Next() {
		var sha string
		if err := commits.Scan(&sha); err != nil {
			continue
		}
		shasToUpdate = append(shasToUpdate, sha)
	}

	if len(shasToUpdate) > 0 {
		fmt.Printf("  Found %d commits without parent_shas, updating...\n", len(shasToUpdate))

		tx2, err := db.Begin()
		if err != nil {
			log.Printf("  Failed to start transaction: %v", err)
			return
		}
		defer tx2.Rollback()

		for i, sha := range shasToUpdate {
			parents, err := sorter.GetCommitParents(ctx, sha)
			if err != nil {
				log.Printf("  Warning: Failed to get parents for %s: %v", sha, err)
				continue
			}

			// Convert []string to PostgreSQL array format
			_, err = tx2.Exec(
				"UPDATE github_commits SET parent_shas = $1 WHERE repo_id = $2 AND sha = $3",
				parents, repoID, sha)
			if err != nil {
				log.Printf("  Failed to update parent_shas for %s: %v", sha, err)
				continue
			}

			if (i+1)%100 == 0 {
				fmt.Printf("  Progress: %d/%d commits\n", i+1, len(shasToUpdate))
			}
		}

		if err := tx2.Commit(); err != nil {
			log.Printf("  Failed to commit: %v", err)
			return
		}
		fmt.Printf("  ✓ Updated parent_shas for %d commits\n", len(shasToUpdate))
	} else {
		fmt.Printf("  All commits already have parent_shas populated\n")
	}

	fmt.Printf("\n✅ Complete! Repository %d is ready for semantic processing.\n", repoID)
}
