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

// FileSyncer syncs File nodes from PostgreSQL to Neo4j
// Reference: DATA_SCHEMA_REFERENCE.md line 526-545 - File node spec
type FileSyncer struct {
	db       *sql.DB
	driver   neo4j.DriverWithContext
	database string
}

// NewFileSyncer creates a new file syncer
func NewFileSyncer(db *sql.DB, driver neo4j.DriverWithContext, database string) *FileSyncer {
	return &FileSyncer{
		db:       db,
		driver:   driver,
		database: database,
	}
}

// File represents a file from PostgreSQL file_identity_map
type File struct {
	CanonicalPath   string
	RepoID          int64
	HistoricalPaths []string
	Language        *string
	Status          string
	FileIdentityID  int64
	CreatedAt       *time.Time
	LastModifiedAt  *time.Time
}

// SyncMissingFiles syncs File nodes from PostgreSQL to Neo4j (incremental)
// Returns: (nodes created, error)
func (s *FileSyncer) SyncMissingFiles(ctx context.Context, repoID int64) (int, error) {
	log.Printf("  📁 Syncing File nodes from PostgreSQL → Neo4j...")

	// Step 1: Get all files from PostgreSQL file_identity_map
	files, err := s.getAllFiles(ctx, repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to get postgres files: %w", err)
	}
	log.Printf("     PostgreSQL: %d Files", len(files))

	// Step 2: Get existing file composite IDs from Neo4j
	neo4jIDs, err := s.getNeo4jFileCompositeIDs(ctx, repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to get neo4j files: %w", err)
	}
	log.Printf("     Neo4j: %d Files", len(neo4jIDs))

	// Step 3: Compute delta (files not in Neo4j)
	neo4jIDSet := make(map[string]bool, len(neo4jIDs))
	for _, id := range neo4jIDs {
		neo4jIDSet[id] = true
	}

	var missingFiles []File
	for _, file := range files {
		compositeID := fmt.Sprintf("%d:file:%s", file.RepoID, file.CanonicalPath)
		if !neo4jIDSet[compositeID] {
			missingFiles = append(missingFiles, file)
		}
	}

	if len(missingFiles) == 0 {
		log.Printf("     ✓ All Files already in sync")
		return 0, nil
	}
	log.Printf("     Delta: %d Files need syncing", len(missingFiles))

	// Step 4: Create missing File nodes in Neo4j
	synced := 0
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	for _, file := range missingFiles {
		compositeID := fmt.Sprintf("%d:file:%s", file.RepoID, file.CanonicalPath)

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			// Create File node with MERGE (idempotent)
			// Reference: DATA_SCHEMA_REFERENCE.md line 859 - UNIQUE(File.repo_id, File.canonical_path)
			query := `
				MERGE (f:File {id: $compositeID, repo_id: $repoID, canonical_path: $canonicalPath})
				SET f.historical_paths = $historicalPaths,
				    f.language = $language,
				    f.status = $status,
				    f.file_identity_id = $fileIdentityID,
				    f.created_at = CASE WHEN $createdAt IS NOT NULL THEN datetime($createdAt) ELSE NULL END,
				    f.last_modified_at = CASE WHEN $lastModifiedAt IS NOT NULL THEN datetime($lastModifiedAt) ELSE NULL END
				RETURN f.id as id`

			// Handle NULL timestamps
			var createdAtStr, lastModifiedAtStr interface{}
			if file.CreatedAt != nil {
				createdAtStr = file.CreatedAt.Format(time.RFC3339)
			}
			if file.LastModifiedAt != nil {
				lastModifiedAtStr = file.LastModifiedAt.Format(time.RFC3339)
			}

			params := map[string]any{
				"compositeID":     compositeID,
				"repoID":          file.RepoID,
				"canonicalPath":   file.CanonicalPath,
				"historicalPaths": file.HistoricalPaths,
				"language":        file.Language,
				"status":          file.Status,
				"fileIdentityID":  file.FileIdentityID,
				"createdAt":       createdAtStr,
				"lastModifiedAt":  lastModifiedAtStr,
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			log.Printf("     ⚠️  Failed to sync file %s: %v", file.CanonicalPath, err)
			continue
		}
		synced++

		// Log progress every 200 files
		if synced%200 == 0 {
			log.Printf("     → Synced %d/%d files", synced, len(missingFiles))
		}
	}

	log.Printf("     ✓ Synced %d Files to Neo4j", synced)
	return synced, nil
}

// getAllFiles returns all files from PostgreSQL file_identity_map
func (s *FileSyncer) getAllFiles(ctx context.Context, repoID int64) ([]File, error) {
	query := `
		SELECT
			canonical_path,
			repo_id,
			historical_paths,
			language,
			status,
			id,
			created_at,
			last_modified_at
		FROM file_identity_map
		WHERE repo_id = $1
		ORDER BY canonical_path`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var f File
		var historicalPathsJSON []byte

		err := rows.Scan(
			&f.CanonicalPath,
			&f.RepoID,
			&historicalPathsJSON,
			&f.Language,
			&f.Status,
			&f.FileIdentityID,
			&f.CreatedAt,
			&f.LastModifiedAt,
		)
		if err != nil {
			return nil, err
		}

		// Parse JSONB historical_paths array
		if len(historicalPathsJSON) > 0 {
			if err := json.Unmarshal(historicalPathsJSON, &f.HistoricalPaths); err != nil {
				return nil, fmt.Errorf("failed to parse historical_paths for %s: %w", f.CanonicalPath, err)
			}
		}

		files = append(files, f)
	}

	return files, rows.Err()
}

// getNeo4jFileCompositeIDs returns existing file composite IDs from Neo4j
func (s *FileSyncer) getNeo4jFileCompositeIDs(ctx context.Context, repoID int64) ([]string, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `MATCH (f:File {repo_id: $repoID}) RETURN f.id as id`
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
