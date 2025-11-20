package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// AtomizeEdgeSyncer syncs edges created by crisk-atomize from PostgreSQL to Neo4j
// Reference: DATA_SCHEMA_REFERENCE.md line 863-877 - crisk-atomize edge spec
// Handles: CONTAINS (File→CodeBlock), RENAMED_FROM (CodeBlock→CodeBlock), IMPORTS_FROM (CodeBlock→CodeBlock)
type AtomizeEdgeSyncer struct {
	db       *sql.DB
	driver   neo4j.DriverWithContext
	database string
}

// NewAtomizeEdgeSyncer creates a new atomize edge syncer
func NewAtomizeEdgeSyncer(db *sql.DB, driver neo4j.DriverWithContext, database string) *AtomizeEdgeSyncer {
	return &AtomizeEdgeSyncer{
		db:       db,
		driver:   driver,
		database: database,
	}
}

// SyncAllAtomizeEdges syncs all 3 edge types from crisk-atomize
// Returns: (total edges created, error)
func (s *AtomizeEdgeSyncer) SyncAllAtomizeEdges(ctx context.Context, repoID int64) (int, error) {
	log.Printf("  🔗 Syncing crisk-atomize edges from PostgreSQL → Neo4j...")

	totalSynced := 0

	// 1. CONTAINS edges (File→CodeBlock)
	contains, err := s.syncContainsEdges(ctx, repoID)
	if err != nil {
		return totalSynced, fmt.Errorf("CONTAINS edge sync failed: %w", err)
	}
	totalSynced += contains

	// 2. RENAMED_FROM edges (CodeBlock→CodeBlock)
	renamedFrom, err := s.syncRenamedFromEdges(ctx, repoID)
	if err != nil {
		return totalSynced, fmt.Errorf("RENAMED_FROM edge sync failed: %w", err)
	}
	totalSynced += renamedFrom

	// 3. IMPORTS_FROM edges (CodeBlock→CodeBlock)
	importsFrom, err := s.syncImportsFromEdges(ctx, repoID)
	if err != nil {
		return totalSynced, fmt.Errorf("IMPORTS_FROM edge sync failed: %w", err)
	}
	totalSynced += importsFrom

	log.Printf("     ✓ Total crisk-atomize edges synced: %d", totalSynced)
	return totalSynced, nil
}

// syncContainsEdges creates CONTAINS edges (File→CodeBlock)
// Reference: DATA_SCHEMA_REFERENCE.md line 740-748
func (s *AtomizeEdgeSyncer) syncContainsEdges(ctx context.Context, repoID int64) (int, error) {
	log.Printf("     → Syncing CONTAINS edges (File→CodeBlock)...")

	// Query all code blocks with their canonical file paths
	query := `
		SELECT id, file_path
		FROM code_blocks
		WHERE repo_id = $1
		ORDER BY id`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	synced := 0
	for rows.Next() {
		var blockID int64
		var filePath string

		if err := rows.Scan(&blockID, &filePath); err != nil {
			return synced, err
		}

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			// Match File by canonical_path and CodeBlock by db_id
			query := `
				MATCH (f:File {canonical_path: $filePath, repo_id: $repoID})
				MATCH (b:CodeBlock {db_id: $blockID, repo_id: $repoID})
				MERGE (f)-[r:CONTAINS]->(b)
				RETURN count(r) as created`

			params := map[string]any{
				"filePath": filePath,
				"blockID":  blockID,
				"repoID":   repoID,
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			log.Printf("       ⚠️  Failed to create CONTAINS edge for block %d: %v", blockID, err)
			continue
		}
		synced++

		if synced%200 == 0 {
			log.Printf("       → Synced %d CONTAINS edges", synced)
		}
	}

	log.Printf("       ✓ Synced %d CONTAINS edges", synced)
	return synced, rows.Err()
}

// syncRenamedFromEdges creates RENAMED_FROM edges (CodeBlock→CodeBlock)
// Reference: DATA_SCHEMA_REFERENCE.md line 751-763
func (s *AtomizeEdgeSyncer) syncRenamedFromEdges(ctx context.Context, repoID int64) (int, error) {
	log.Printf("     → Syncing RENAMED_FROM edges (CodeBlock→CodeBlock)...")

	// Query function_identity_map for rename relationships
	// NOTE: Current schema doesn't support explicit old_block_id tracking
	// This feature will be implemented when rename detection is added
	query := `
		SELECT block_id, old_block_id, rename_commit_sha
		FROM function_identity_map
		WHERE repo_id = $1
		  AND old_block_id IS NOT NULL
		ORDER BY rename_date ASC`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		// Table might not exist or schema doesn't match
		errMsg := err.Error()
		if errMsg == `pq: relation "function_identity_map" does not exist` ||
		   errMsg == `pq: column "old_block_id" does not exist` {
			log.Printf("       ℹ️  Rename tracking not available - skipping RENAMED_FROM edges")
			return 0, nil
		}
		return 0, err
	}
	defer rows.Close()

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	synced := 0
	for rows.Next() {
		var newBlockID, oldBlockID int64
		var renameCommitSHA string

		if err := rows.Scan(&newBlockID, &oldBlockID, &renameCommitSHA); err != nil {
			return synced, err
		}

		// Get old block name for edge property
		var oldBlockName string
		err := s.db.QueryRowContext(ctx, "SELECT block_name FROM code_blocks WHERE id = $1", oldBlockID).Scan(&oldBlockName)
		if err != nil {
			continue
		}

		_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			query := `
				MATCH (new:CodeBlock {db_id: $newBlockID, repo_id: $repoID})
				MATCH (old:CodeBlock {db_id: $oldBlockID, repo_id: $repoID})
				MERGE (new)-[r:RENAMED_FROM]->(old)
				SET r.old_name = $oldBlockName,
				    r.rename_commit = $renameCommitSHA
				RETURN count(r) as created`

			params := map[string]any{
				"newBlockID":      newBlockID,
				"oldBlockID":      oldBlockID,
				"repoID":          repoID,
				"oldBlockName":    oldBlockName,
				"renameCommitSHA": renameCommitSHA,
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			log.Printf("       ⚠️  Failed to create RENAMED_FROM edge %d→%d: %v", newBlockID, oldBlockID, err)
			continue
		}
		synced++
	}

	log.Printf("       ✓ Synced %d RENAMED_FROM edges", synced)
	return synced, rows.Err()
}

// syncImportsFromEdges creates IMPORTS_FROM edges (CodeBlock→CodeBlock)
// Reference: DATA_SCHEMA_REFERENCE.md line 766-777
func (s *AtomizeEdgeSyncer) syncImportsFromEdges(ctx context.Context, repoID int64) (int, error) {
	log.Printf("     → Syncing IMPORTS_FROM edges (CodeBlock→CodeBlock)...")

	// Query code_block_imports for import relationships
	// NOTE: Import tracking table may not exist yet
	query := `
		SELECT source_block_id, target_block_id, import_type, symbol
		FROM code_block_imports
		WHERE repo_id = $1
		ORDER BY source_block_id`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		// Table might not exist or schema doesn't match
		errMsg := err.Error()
		if errMsg == `pq: relation "code_block_imports" does not exist` ||
		   errMsg == `pq: column "target_block_id" does not exist` ||
		   errMsg == `pq: column "source_block_id" does not exist` {
			log.Printf("       ℹ️  Import tracking not available - skipping IMPORTS_FROM edges")
			return 0, nil
		}
		return 0, err
	}
	defer rows.Close()

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	synced := 0
	for rows.Next() {
		var sourceBlockID, targetBlockID int64
		var importType, symbol string

		if err := rows.Scan(&sourceBlockID, &targetBlockID, &importType, &symbol); err != nil {
			return synced, err
		}

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			query := `
				MATCH (source:CodeBlock {db_id: $sourceBlockID, repo_id: $repoID})
				MATCH (target:CodeBlock {db_id: $targetBlockID, repo_id: $repoID})
				MERGE (source)-[r:IMPORTS_FROM]->(target)
				SET r.import_type = $importType,
				    r.symbol = $symbol
				RETURN count(r) as created`

			params := map[string]any{
				"sourceBlockID": sourceBlockID,
				"targetBlockID": targetBlockID,
				"repoID":        repoID,
				"importType":    importType,
				"symbol":        symbol,
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			log.Printf("       ⚠️  Failed to create IMPORTS_FROM edge %d→%d: %v", sourceBlockID, targetBlockID, err)
			continue
		}
		synced++

		if synced%100 == 0 {
			log.Printf("       → Synced %d IMPORTS_FROM edges", synced)
		}
	}

	log.Printf("       ✓ Synced %d IMPORTS_FROM edges", synced)
	return synced, rows.Err()
}
