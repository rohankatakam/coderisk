package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// IssueSyncer syncs Issue nodes from PostgreSQL to Neo4j
// Reference: DATA_SCHEMA_REFERENCE.md line 548-565 - Issue node spec
type IssueSyncer struct {
	db       *sql.DB
	driver   neo4j.DriverWithContext
	database string
}

// NewIssueSyncer creates a new issue syncer
func NewIssueSyncer(db *sql.DB, driver neo4j.DriverWithContext, database string) *IssueSyncer {
	return &IssueSyncer{
		db:       db,
		driver:   driver,
		database: database,
	}
}

// Issue represents an issue from PostgreSQL
type Issue struct {
	Number    int
	RepoID    int64
	Title     string
	State     string
	CreatedAt time.Time
	ClosedAt  *time.Time
	Labels    []string
	UserLogin string
}

// SyncMissingIssues syncs Issue nodes from PostgreSQL to Neo4j (incremental)
// Returns: (nodes created, error)
func (s *IssueSyncer) SyncMissingIssues(ctx context.Context, repoID int64) (int, error) {
	log.Printf("  🐛 Syncing Issue nodes from PostgreSQL → Neo4j...")

	// Step 1: Get all issues from PostgreSQL (excluding PRs)
	issues, err := s.getAllIssues(ctx, repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to get postgres issues: %w", err)
	}
	log.Printf("     PostgreSQL: %d Issues", len(issues))

	// Step 2: Get existing issue composite IDs from Neo4j
	neo4jIDs, err := s.getNeo4jIssueCompositeIDs(ctx, repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to get neo4j issues: %w", err)
	}
	log.Printf("     Neo4j: %d Issues", len(neo4jIDs))

	// Step 3: Compute delta (issues not in Neo4j)
	neo4jIDSet := make(map[string]bool, len(neo4jIDs))
	for _, id := range neo4jIDs {
		neo4jIDSet[id] = true
	}

	var missingIssues []Issue
	for _, issue := range issues {
		compositeID := fmt.Sprintf("%d:issue:%d", issue.RepoID, issue.Number)
		if !neo4jIDSet[compositeID] {
			missingIssues = append(missingIssues, issue)
		}
	}

	if len(missingIssues) == 0 {
		log.Printf("     ✓ All Issues already in sync")
		return 0, nil
	}
	log.Printf("     Delta: %d Issues need syncing", len(missingIssues))

	// Step 4: Create missing Issue nodes in Neo4j
	synced := 0
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	for _, issue := range missingIssues {
		compositeID := fmt.Sprintf("%d:issue:%d", issue.RepoID, issue.Number)

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			// Create Issue node with MERGE (idempotent)
			// Reference: DATA_SCHEMA_REFERENCE.md line 860 - UNIQUE(Issue.repo_id, Issue.number)
			query := `
				MERGE (i:Issue {id: $compositeID, repo_id: $repoID, number: $number})
				SET i.title = $title,
				    i.state = $state,
				    i.created_at = datetime($createdAt),
				    i.closed_at = CASE WHEN $closedAt IS NOT NULL THEN datetime($closedAt) ELSE NULL END,
				    i.labels = $labels,
				    i.user_login = $userLogin
				RETURN i.id as id`

			// Handle NULL closed_at
			var closedAtStr interface{}
			if issue.ClosedAt != nil {
				closedAtStr = issue.ClosedAt.Format(time.RFC3339)
			}

			params := map[string]any{
				"compositeID": compositeID,
				"repoID":      issue.RepoID,
				"number":      issue.Number,
				"title":       issue.Title,
				"state":       issue.State,
				"createdAt":   issue.CreatedAt.Format(time.RFC3339),
				"closedAt":    closedAtStr,
				"labels":      issue.Labels,
				"userLogin":   issue.UserLogin,
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			log.Printf("     ⚠️  Failed to sync issue #%d: %v", issue.Number, err)
			continue
		}
		synced++

		// Log progress every 100 issues
		if synced%100 == 0 {
			log.Printf("     → Synced %d/%d issues", synced, len(missingIssues))
		}
	}

	log.Printf("     ✓ Synced %d Issues to Neo4j", synced)
	return synced, nil
}

// getAllIssues returns all issues from PostgreSQL (excluding PRs)
func (s *IssueSyncer) getAllIssues(ctx context.Context, repoID int64) ([]Issue, error) {
	query := `
		SELECT
			number,
			repo_id,
			title,
			state,
			created_at,
			closed_at,
			labels,
			user_login
		FROM github_issues
		WHERE repo_id = $1
		ORDER BY number`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []Issue
	for rows.Next() {
		var i Issue
		var labelsJSON []byte

		err := rows.Scan(
			&i.Number,
			&i.RepoID,
			&i.Title,
			&i.State,
			&i.CreatedAt,
			&i.ClosedAt,
			&labelsJSON,
			&i.UserLogin,
		)
		if err != nil {
			return nil, err
		}

		// Parse labels JSONB
		if len(labelsJSON) > 0 {
			if err := json.Unmarshal(labelsJSON, &i.Labels); err != nil {
				log.Printf("     ⚠️  Failed to parse labels for issue #%d: %v", i.Number, err)
				i.Labels = []string{}
			}
		}

		issues = append(issues, i)
	}

	return issues, rows.Err()
}

// getNeo4jIssueCompositeIDs returns existing issue composite IDs from Neo4j
func (s *IssueSyncer) getNeo4jIssueCompositeIDs(ctx context.Context, repoID int64) ([]string, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `MATCH (i:Issue {repo_id: $repoID}) RETURN i.id as id`
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
