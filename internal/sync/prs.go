package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// PRSyncer syncs PR nodes from PostgreSQL to Neo4j
// Reference: DATA_SCHEMA_REFERENCE.md line 568-586 - PR node spec
type PRSyncer struct {
	db       *sql.DB
	driver   neo4j.DriverWithContext
	database string
}

// NewPRSyncer creates a new PR syncer
func NewPRSyncer(db *sql.DB, driver neo4j.DriverWithContext, database string) *PRSyncer {
	return &PRSyncer{
		db:       db,
		driver:   driver,
		database: database,
	}
}

// PR represents a pull request from PostgreSQL
type PR struct {
	Number         int
	RepoID         int64
	Title          string
	State          string
	Merged         bool
	CreatedAt      time.Time
	MergedAt       *time.Time
	MergeCommitSHA *string
	UserLogin      string
}

// SyncMissingPRs syncs PR nodes from PostgreSQL to Neo4j (incremental)
// Returns: (nodes created, error)
func (s *PRSyncer) SyncMissingPRs(ctx context.Context, repoID int64) (int, error) {
	log.Printf("  🔀 Syncing PR nodes from PostgreSQL → Neo4j...")

	// Step 1: Get all PRs from PostgreSQL
	prs, err := s.getAllPRs(ctx, repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to get postgres PRs: %w", err)
	}
	log.Printf("     PostgreSQL: %d PRs", len(prs))

	// Step 2: Get existing PR composite IDs from Neo4j
	neo4jIDs, err := s.getNeo4jPRCompositeIDs(ctx, repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to get neo4j PRs: %w", err)
	}
	log.Printf("     Neo4j: %d PRs", len(neo4jIDs))

	// Step 3: Compute delta (PRs not in Neo4j)
	neo4jIDSet := make(map[string]bool, len(neo4jIDs))
	for _, id := range neo4jIDs {
		neo4jIDSet[id] = true
	}

	var missingPRs []PR
	for _, pr := range prs {
		compositeID := fmt.Sprintf("%d:pr:%d", pr.RepoID, pr.Number)
		if !neo4jIDSet[compositeID] {
			missingPRs = append(missingPRs, pr)
		}
	}

	if len(missingPRs) == 0 {
		log.Printf("     ✓ All PRs already in sync")
		return 0, nil
	}
	log.Printf("     Delta: %d PRs need syncing", len(missingPRs))

	// Step 4: Create missing PR nodes in Neo4j
	synced := 0
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	for _, pr := range missingPRs {
		compositeID := fmt.Sprintf("%d:pr:%d", pr.RepoID, pr.Number)

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			// Create PR node with MERGE (idempotent)
			// Reference: DATA_SCHEMA_REFERENCE.md line 861 - UNIQUE(PR.repo_id, PR.number)
			query := `
				MERGE (pr:PR {id: $compositeID, repo_id: $repoID, number: $number})
				SET pr.title = $title,
				    pr.state = $state,
				    pr.merged = $merged,
				    pr.created_at = datetime($createdAt),
				    pr.merged_at = CASE WHEN $mergedAt IS NOT NULL THEN datetime($mergedAt) ELSE NULL END,
				    pr.merge_commit_sha = $mergeCommitSHA,
				    pr.user_login = $userLogin
				RETURN pr.id as id`

			// Handle NULL fields
			var mergedAtStr interface{}
			if pr.MergedAt != nil {
				mergedAtStr = pr.MergedAt.Format(time.RFC3339)
			}

			params := map[string]any{
				"compositeID":    compositeID,
				"repoID":         pr.RepoID,
				"number":         pr.Number,
				"title":          pr.Title,
				"state":          pr.State,
				"merged":         pr.Merged,
				"createdAt":      pr.CreatedAt.Format(time.RFC3339),
				"mergedAt":       mergedAtStr,
				"mergeCommitSHA": pr.MergeCommitSHA,
				"userLogin":      pr.UserLogin,
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			log.Printf("     ⚠️  Failed to sync PR #%d: %v", pr.Number, err)
			continue
		}
		synced++

		// Log progress every 100 PRs
		if synced%100 == 0 {
			log.Printf("     → Synced %d/%d PRs", synced, len(missingPRs))
		}
	}

	log.Printf("     ✓ Synced %d PRs to Neo4j", synced)
	return synced, nil
}

// getAllPRs returns all PRs from PostgreSQL
func (s *PRSyncer) getAllPRs(ctx context.Context, repoID int64) ([]PR, error) {
	query := `
		SELECT
			number,
			repo_id,
			title,
			state,
			merged,
			created_at,
			merged_at,
			merge_commit_sha,
			user_login
		FROM github_pull_requests
		WHERE repo_id = $1
		ORDER BY number`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []PR
	for rows.Next() {
		var p PR

		err := rows.Scan(
			&p.Number,
			&p.RepoID,
			&p.Title,
			&p.State,
			&p.Merged,
			&p.CreatedAt,
			&p.MergedAt,
			&p.MergeCommitSHA,
			&p.UserLogin,
		)
		if err != nil {
			return nil, err
		}

		prs = append(prs, p)
	}

	return prs, rows.Err()
}

// getNeo4jPRCompositeIDs returns existing PR composite IDs from Neo4j
func (s *PRSyncer) getNeo4jPRCompositeIDs(ctx context.Context, repoID int64) ([]string, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `MATCH (pr:PR {repo_id: $repoID}) RETURN pr.id as id`
		result, err := tx.Run(ctx, query, map[string]any{"repoID": repoID})
		if err != nil {
			return nil, err
		}

		return result.Collect(ctx)
	})

	if err != nil {
		return nil, err
	}

	records := result.([]*neo4j.Record)
	var ids []string
	for _, record := range records {
		id, ok := record.Get("id")
		if !ok {
			continue
		}
		ids = append(ids, id.(string))
	}

	return ids, nil
}
