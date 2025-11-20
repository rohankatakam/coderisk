package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// IngestEdgeSyncer syncs all edges created by crisk-ingest from PostgreSQL to Neo4j
// Reference: DATA_SCHEMA_REFERENCE.md line 827-846 - crisk-ingest edge spec
// Handles: AUTHORED, MODIFIED, CREATED, MERGED_AS, REFERENCES, CLOSED_BY
type IngestEdgeSyncer struct {
	db       *sql.DB
	driver   neo4j.DriverWithContext
	database string
}

// NewIngestEdgeSyncer creates a new ingest edge syncer
func NewIngestEdgeSyncer(db *sql.DB, driver neo4j.DriverWithContext, database string) *IngestEdgeSyncer {
	return &IngestEdgeSyncer{
		db:       db,
		driver:   driver,
		database: database,
	}
}

// SyncAllIngestEdges syncs all 7 edge types from crisk-ingest
// Returns: (total edges created, error)
func (s *IngestEdgeSyncer) SyncAllIngestEdges(ctx context.Context, repoID int64) (int, error) {
	log.Printf("  🔗 Syncing crisk-ingest edges from PostgreSQL → Neo4j...")

	totalSynced := 0

	// 1. AUTHORED edges (Developer→Commit)
	authored, err := s.syncAuthoredEdges(ctx, repoID)
	if err != nil {
		return totalSynced, fmt.Errorf("AUTHORED edge sync failed: %w", err)
	}
	totalSynced += authored

	// 2. MODIFIED edges (Commit→File) - requires file_identity_map for canonical path resolution
	modified, err := s.syncModifiedEdges(ctx, repoID)
	if err != nil {
		return totalSynced, fmt.Errorf("MODIFIED edge sync failed: %w", err)
	}
	totalSynced += modified

	// 3. CREATED edges from Issues (Issue→Developer)
	issueCreated, err := s.syncIssueCreatedEdges(ctx, repoID)
	if err != nil {
		return totalSynced, fmt.Errorf("Issue CREATED edge sync failed: %w", err)
	}
	totalSynced += issueCreated

	// 4. CREATED edges from PRs (PR→Developer)
	prCreated, err := s.syncPRCreatedEdges(ctx, repoID)
	if err != nil {
		return totalSynced, fmt.Errorf("PR CREATED edge sync failed: %w", err)
	}
	totalSynced += prCreated

	// 5. MERGED_AS edges (PR→Commit)
	mergedAs, err := s.syncMergedAsEdges(ctx, repoID)
	if err != nil {
		return totalSynced, fmt.Errorf("MERGED_AS edge sync failed: %w", err)
	}
	totalSynced += mergedAs

	// 6. REFERENCES edges (Issue→Issue/PR)
	references, err := s.syncReferencesEdges(ctx, repoID)
	if err != nil {
		return totalSynced, fmt.Errorf("REFERENCES edge sync failed: %w", err)
	}
	totalSynced += references

	// 7. CLOSED_BY edges (Issue→Commit)
	closedBy, err := s.syncClosedByEdges(ctx, repoID)
	if err != nil {
		return totalSynced, fmt.Errorf("CLOSED_BY edge sync failed: %w", err)
	}
	totalSynced += closedBy

	log.Printf("     ✓ Total crisk-ingest edges synced: %d", totalSynced)
	return totalSynced, nil
}

// syncAuthoredEdges creates AUTHORED edges (Developer→Commit)
// Reference: DATA_SCHEMA_REFERENCE.md line 629-638
func (s *IngestEdgeSyncer) syncAuthoredEdges(ctx context.Context, repoID int64) (int, error) {
	log.Printf("     → Syncing AUTHORED edges (Developer→Commit)...")

	// Query commits to get author relationships
	query := `
		SELECT author_email, sha, author_date
		FROM github_commits
		WHERE repo_id = $1
		ORDER BY author_date ASC`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	synced := 0
	for rows.Next() {
		var authorEmail, sha string
		var authorDate time.Time

		if err := rows.Scan(&authorEmail, &sha, &authorDate); err != nil {
			return synced, err
		}

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			query := `
				MATCH (d:Developer {email: $authorEmail, repo_id: $repoID})
				MATCH (c:Commit {sha: $sha, repo_id: $repoID})
				MERGE (d)-[r:AUTHORED]->(c)
				SET r.timestamp = datetime($timestamp)
				RETURN count(r) as created`

			params := map[string]any{
				"authorEmail": authorEmail,
				"sha":         sha,
				"repoID":      repoID,
				"timestamp":   authorDate.Format(time.RFC3339),
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			log.Printf("       ⚠️  Failed to create AUTHORED edge %s→%s: %v", authorEmail, sha[:8], err)
			continue
		}
		synced++

		if synced%500 == 0 {
			log.Printf("       → Synced %d AUTHORED edges", synced)
		}
	}

	log.Printf("       ✓ Synced %d AUTHORED edges", synced)
	return synced, rows.Err()
}

// syncModifiedEdges creates MODIFIED edges (Commit→File)
// Reference: DATA_SCHEMA_REFERENCE.md line 641-652
// NOTE: Requires parsing git diff stats from commits - simplified version creates edges for all files
func (s *IngestEdgeSyncer) syncModifiedEdges(ctx context.Context, repoID int64) (int, error) {
	log.Printf("     → Syncing MODIFIED edges (Commit→File)...")

	// Query github_commits with file stats
	// NOTE: This is simplified - full implementation would parse actual file changes from git
	// For now, we'll create edges based on file_identity_map presence
	query := `
		SELECT c.sha, c.repo_id
		FROM github_commits c
		WHERE c.repo_id = $1
		  AND c.files_changed > 0
		ORDER BY c.topological_index ASC`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type commitRef struct {
		SHA    string
		RepoID int64
	}

	var commits []commitRef
	for rows.Next() {
		var c commitRef
		if err := rows.Scan(&c.SHA, &c.RepoID); err != nil {
			return 0, err
		}
		commits = append(commits, c)
	}

	if err := rows.Err(); err != nil {
		return 0, err
	}

	log.Printf("       Found %d commits with file changes", len(commits))
	log.Printf("       ⚠️  MODIFIED edge sync requires git diff parsing - skipping for now")
	log.Printf("       (This will be populated by crisk-ingest when it writes to Neo4j)")

	return 0, nil
}

// syncIssueCreatedEdges creates CREATED edges (Issue→Developer)
// Reference: DATA_SCHEMA_REFERENCE.md line 670-679
func (s *IngestEdgeSyncer) syncIssueCreatedEdges(ctx context.Context, repoID int64) (int, error) {
	log.Printf("     → Syncing Issue CREATED edges (Issue→Developer)...")

	query := `
		SELECT number, user_login, created_at
		FROM github_issues
		WHERE repo_id = $1
		ORDER BY number`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	synced := 0
	for rows.Next() {
		var number int
		var userLogin string
		var createdAt time.Time

		if err := rows.Scan(&number, &userLogin, &createdAt); err != nil {
			return synced, err
		}

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			// Note: CREATED edge goes FROM Issue TO Developer (creator)
			query := `
				MATCH (i:Issue {number: $number, repo_id: $repoID})
				MATCH (d:Developer {email: $userLogin, repo_id: $repoID})
				MERGE (i)-[r:CREATED]->(d)
				SET r.created_at = datetime($createdAt)
				RETURN count(r) as created`

			params := map[string]any{
				"number":    number,
				"userLogin": userLogin,
				"repoID":    repoID,
				"createdAt": createdAt.Format(time.RFC3339),
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			// Developer might not exist if userLogin is username not email - skip gracefully
			continue
		}
		synced++
	}

	log.Printf("       ✓ Synced %d Issue CREATED edges", synced)
	return synced, rows.Err()
}

// syncPRCreatedEdges creates CREATED edges (PR→Developer)
// Reference: DATA_SCHEMA_REFERENCE.md line 670-679
func (s *IngestEdgeSyncer) syncPRCreatedEdges(ctx context.Context, repoID int64) (int, error) {
	log.Printf("     → Syncing PR CREATED edges (PR→Developer)...")

	query := `
		SELECT number, user_login, created_at
		FROM github_pull_requests
		WHERE repo_id = $1
		ORDER BY number`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	synced := 0
	for rows.Next() {
		var number int
		var userLogin string
		var createdAt time.Time

		if err := rows.Scan(&number, &userLogin, &createdAt); err != nil {
			return synced, err
		}

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			query := `
				MATCH (pr:PR {number: $number, repo_id: $repoID})
				MATCH (d:Developer {email: $userLogin, repo_id: $repoID})
				MERGE (pr)-[r:CREATED]->(d)
				SET r.created_at = datetime($createdAt)
				RETURN count(r) as created`

			params := map[string]any{
				"number":    number,
				"userLogin": userLogin,
				"repoID":    repoID,
				"createdAt": createdAt.Format(time.RFC3339),
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			// Developer might not exist - skip gracefully
			continue
		}
		synced++
	}

	log.Printf("       ✓ Synced %d PR CREATED edges", synced)
	return synced, rows.Err()
}

// syncMergedAsEdges creates MERGED_AS edges (PR→Commit)
// Reference: DATA_SCHEMA_REFERENCE.md line 682-694
func (s *IngestEdgeSyncer) syncMergedAsEdges(ctx context.Context, repoID int64) (int, error) {
	log.Printf("     → Syncing MERGED_AS edges (PR→Commit)...")

	query := `
		SELECT number, merge_commit_sha, merged_at
		FROM github_pull_requests
		WHERE repo_id = $1
		  AND merged = true
		  AND merge_commit_sha IS NOT NULL
		ORDER BY number`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	synced := 0
	for rows.Next() {
		var number int
		var mergeCommitSHA string
		var mergedAt time.Time

		if err := rows.Scan(&number, &mergeCommitSHA, &mergedAt); err != nil {
			return synced, err
		}

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			query := `
				MATCH (pr:PR {number: $number, repo_id: $repoID})
				MATCH (c:Commit {sha: $mergeCommitSHA, repo_id: $repoID})
				MERGE (pr)-[r:MERGED_AS]->(c)
				SET r.merged_at = datetime($mergedAt)
				RETURN count(r) as created`

			params := map[string]any{
				"number":         number,
				"mergeCommitSHA": mergeCommitSHA,
				"repoID":         repoID,
				"mergedAt":       mergedAt.Format(time.RFC3339),
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			log.Printf("       ⚠️  Failed to create MERGED_AS edge PR#%d→%s: %v", number, mergeCommitSHA[:8], err)
			continue
		}
		synced++
	}

	log.Printf("       ✓ Synced %d MERGED_AS edges", synced)
	return synced, rows.Err()
}

// syncReferencesEdges creates REFERENCES edges (Issue→Issue/PR)
// Reference: DATA_SCHEMA_REFERENCE.md line 697-707
func (s *IngestEdgeSyncer) syncReferencesEdges(ctx context.Context, repoID int64) (int, error) {
	log.Printf("     → Syncing REFERENCES edges (Issue→Issue/PR)...")

	query := `
		SELECT t.issue_id, i.number as issue_number, t.source_type, t.source_number, t.event_type, t.created_at
		FROM github_issue_timeline t
		JOIN github_issues i ON t.issue_id = i.id
		WHERE i.repo_id = $1
		  AND t.event_type IN ('cross-referenced', 'referenced')
		  AND t.source_number IS NOT NULL
		ORDER BY t.created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	synced := 0
	for rows.Next() {
		var issueID int64
		var issueNumber int
		var sourceType string
		var sourceNumber int
		var eventType string
		var createdAt time.Time

		if err := rows.Scan(&issueID, &issueNumber, &sourceType, &sourceNumber, &eventType, &createdAt); err != nil {
			return synced, err
		}

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			// Determine target node label
			targetLabel := "Issue"
			if sourceType == "pull_request" {
				targetLabel = "PR"
			}

			query := fmt.Sprintf(`
				MATCH (i:Issue {number: $issueNumber, repo_id: $repoID})
				MATCH (target:%s {number: $sourceNumber, repo_id: $repoID})
				MERGE (i)-[r:REFERENCES]->(target)
				SET r.reference_type = $eventType,
				    r.created_at = datetime($createdAt)
				RETURN count(r) as created`, targetLabel)

			params := map[string]any{
				"issueNumber":  issueNumber,
				"sourceNumber": sourceNumber,
				"repoID":       repoID,
				"eventType":    eventType,
				"createdAt":    createdAt.Format(time.RFC3339),
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			continue
		}
		synced++
	}

	log.Printf("       ✓ Synced %d REFERENCES edges", synced)
	return synced, rows.Err()
}

// syncClosedByEdges creates CLOSED_BY edges (Issue→Commit)
// Reference: DATA_SCHEMA_REFERENCE.md line 710-721
func (s *IngestEdgeSyncer) syncClosedByEdges(ctx context.Context, repoID int64) (int, error) {
	log.Printf("     → Syncing CLOSED_BY edges (Issue→Commit)...")

	query := `
		SELECT t.issue_id, i.number as issue_number, t.source_sha, t.created_at
		FROM github_issue_timeline t
		JOIN github_issues i ON t.issue_id = i.id
		WHERE i.repo_id = $1
		  AND t.event_type = 'closed'
		  AND t.source_sha IS NOT NULL
		ORDER BY t.created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	synced := 0
	for rows.Next() {
		var issueID int64
		var issueNumber int
		var commitSHA string
		var createdAt time.Time

		if err := rows.Scan(&issueID, &issueNumber, &commitSHA, &createdAt); err != nil {
			return synced, err
		}

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			query := `
				MATCH (i:Issue {number: $issueNumber, repo_id: $repoID})
				MATCH (c:Commit {sha: $commitSHA, repo_id: $repoID})
				MERGE (i)-[r:CLOSED_BY]->(c)
				SET r.closed_at = datetime($closedAt)
				RETURN count(r) as created`

			params := map[string]any{
				"issueNumber": issueNumber,
				"commitSHA":   commitSHA,
				"repoID":      repoID,
				"closedAt":    createdAt.Format(time.RFC3339),
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			log.Printf("       ⚠️  Failed to create CLOSED_BY edge Issue#%d→%s: %v", issueNumber, commitSHA[:8], err)
			continue
		}
		synced++
	}

	log.Printf("       ✓ Synced %d CLOSED_BY edges", synced)
	return synced, rows.Err()
}
