package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// CommitBlockEdgeSyncer syncs MODIFIED_BLOCK/CREATED_BLOCK edges from PostgreSQL to Neo4j
// This backfills edges that crisk-atomize should have created but failed to persist
type CommitBlockEdgeSyncer struct {
	db       *sql.DB
	driver   neo4j.DriverWithContext
	database string
}

// NewCommitBlockEdgeSyncer creates a new commit-block edge syncer
func NewCommitBlockEdgeSyncer(db *sql.DB, driver neo4j.DriverWithContext, database string) *CommitBlockEdgeSyncer {
	return &CommitBlockEdgeSyncer{
		db:       db,
		driver:   driver,
		database: database,
	}
}

// CodeBlockChange represents a code_block_changes row from PostgreSQL
type CodeBlockChange struct {
	BlockID     int64
	CommitSHA   string
	ChangeType  string
	LinesAdded  int
	LinesDeleted int
	ChangedAt   time.Time
}

// SyncMissingEdges syncs MODIFIED_BLOCK/CREATED_BLOCK edges from PostgreSQL to Neo4j
// Returns: (edges created, error)
func (s *CommitBlockEdgeSyncer) SyncMissingEdges(ctx context.Context, repoID int64) (int, error) {
	log.Printf("  🔗 Syncing Commit→CodeBlock edges from PostgreSQL → Neo4j...")

	// Step 1: Get all code_block_changes from PostgreSQL
	changes, err := s.getAllChanges(ctx, repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to get postgres changes: %w", err)
	}
	log.Printf("     PostgreSQL: %d code_block_changes", len(changes))

	// Step 2: Get existing edges from Neo4j
	existingEdges, err := s.getExistingEdgeCount(ctx, repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to get neo4j edge count: %w", err)
	}
	log.Printf("     Neo4j: %d MODIFIED_BLOCK edges", existingEdges)

	// Step 3: Create missing edges in Neo4j
	// We create all edges from PostgreSQL - MERGE makes it idempotent
	created := 0
	failed := 0
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	for i, change := range changes {
		edgeType := "MODIFIED_BLOCK"
		if change.ChangeType == "created" {
			edgeType = "CREATED_BLOCK"
		} else if change.ChangeType == "deleted" {
			edgeType = "DELETED_BLOCK"
		}

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			// Match Commit and CodeBlock nodes, create edge
			query := fmt.Sprintf(`
				MATCH (c:Commit {sha: $commitSHA, repo_id: $repoID})
				MATCH (b:CodeBlock {db_id: $blockID, repo_id: $repoID})
				MERGE (c)-[r:%s]->(b)
				SET r.repo_id = $repoID,
				    r.timestamp = $timestamp,
				    r.change_type = $changeType,
				    r.lines_added = $linesAdded,
				    r.lines_deleted = $linesDeleted
				RETURN count(r) as edges_created`, edgeType)

			params := map[string]any{
				"commitSHA":    change.CommitSHA,
				"blockID":      change.BlockID,
				"repoID":       repoID,
				"timestamp":    change.ChangedAt.Unix(),
				"changeType":   change.ChangeType,
				"linesAdded":   change.LinesAdded,
				"linesDeleted": change.LinesDeleted,
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			log.Printf("     ⚠️  Failed to create edge for commit %s → block %d: %v",
				change.CommitSHA[:8], change.BlockID, err)
			failed++
			continue
		}
		created++

		// Log progress every 500 edges
		if (i+1)%500 == 0 {
			log.Printf("     → Synced %d/%d edges (%d failed)", i+1-failed, len(changes), failed)
		}
	}

	log.Printf("     ✓ Synced %d edges to Neo4j (%d failed)", created, failed)
	return created, nil
}

// getAllChanges returns all code_block_changes from PostgreSQL
func (s *CommitBlockEdgeSyncer) getAllChanges(ctx context.Context, repoID int64) ([]CodeBlockChange, error) {
	query := `
		SELECT block_id, commit_sha, change_type, lines_added, lines_deleted, created_at
		FROM code_block_changes
		WHERE repo_id = $1
		ORDER BY created_at ASC, block_id ASC`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var changes []CodeBlockChange
	for rows.Next() {
		var c CodeBlockChange
		err := rows.Scan(
			&c.BlockID,
			&c.CommitSHA,
			&c.ChangeType,
			&c.LinesAdded,
			&c.LinesDeleted,
			&c.ChangedAt,
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}

	return changes, rows.Err()
}

// getExistingEdgeCount returns count of MODIFIED_BLOCK edges in Neo4j
func (s *CommitBlockEdgeSyncer) getExistingEdgeCount(ctx context.Context, repoID int64) (int, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Count all Commit→CodeBlock edges (any type)
		query := `
			MATCH (c:Commit {repo_id: $repoID})-[r]->(b:CodeBlock {repo_id: $repoID})
			WHERE type(r) IN ['MODIFIED_BLOCK', 'CREATED_BLOCK', 'DELETED_BLOCK']
			RETURN count(r) as edge_count`

		result, err := tx.Run(ctx, query, map[string]any{"repoID": repoID})
		if err != nil {
			return nil, err
		}

		records, err := result.Collect(ctx)
		if err != nil {
			return nil, err
		}

		if len(records) == 0 {
			return 0, nil
		}

		count, ok := records[0].Get("edge_count")
		if !ok {
			return 0, nil
		}

		return int(count.(int64)), nil
	})

	if err != nil {
		return 0, err
	}

	return result.(int), nil
}
