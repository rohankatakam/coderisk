package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rohankatakam/coderisk/internal/ingestion"
)

// FileIdentityRepository handles database operations for file identity mappings
type FileIdentityRepository struct {
	db *sql.DB
}

// NewFileIdentityRepository creates a new file identity repository
func NewFileIdentityRepository(db *sql.DB) *FileIdentityRepository {
	return &FileIdentityRepository{db: db}
}

// BatchInsert inserts multiple file identity mappings in a single transaction
func (r *FileIdentityRepository) BatchInsert(ctx context.Context, repoID int64, identities map[string]*ingestion.FileIdentity) error {
	if len(identities) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO file_identity_map (
			repo_id,
			canonical_path,
			historical_paths,
			first_seen_commit_sha,
			last_modified_commit_sha,
			last_modified_at,
			language,
			file_type,
			status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (repo_id, canonical_path)
		DO UPDATE SET
			historical_paths = EXCLUDED.historical_paths,
			last_modified_commit_sha = EXCLUDED.last_modified_commit_sha,
			last_modified_at = EXCLUDED.last_modified_at,
			last_updated_at = NOW()
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	count := 0
	for canonicalPath, identity := range identities {
		// Convert historical paths to JSONB
		historicalPathsJSON, err := ingestion.HistoricalPathsToJSON(identity.HistoricalPaths)
		if err != nil {
			return fmt.Errorf("failed to convert historical paths to JSON for %s: %w", canonicalPath, err)
		}

		// Handle zero time values (PostgreSQL doesn't like zero timestamps)
		var lastModifiedAt interface{}
		if identity.LastModifiedAt.IsZero() {
			lastModifiedAt = nil
		} else {
			lastModifiedAt = identity.LastModifiedAt
		}

		_, err = stmt.ExecContext(ctx,
			repoID,
			canonicalPath,
			historicalPathsJSON,
			identity.FirstSeenCommitSHA,
			identity.LastModifiedCommitSHA,
			lastModifiedAt,
			identity.Language,
			identity.FileType,
			identity.Status,
		)
		if err != nil {
			return fmt.Errorf("failed to insert file identity for %s: %w", canonicalPath, err)
		}

		count++
		if count%1000 == 0 {
			fmt.Printf("  ✓ Inserted %d/%d file identities...\n", count, len(identities))
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("✅ Successfully inserted %d file identity mappings\n", count)
	return nil
}

// GetByRepoID retrieves all file identity mappings for a repository
func (r *FileIdentityRepository) GetByRepoID(ctx context.Context, repoID int64) (map[string]*ingestion.FileIdentity, error) {
	query := `
		SELECT
			canonical_path,
			historical_paths,
			first_seen_commit_sha,
			last_modified_commit_sha,
			last_modified_at,
			language,
			file_type,
			status
		FROM file_identity_map
		WHERE repo_id = $1
		  AND status = 'active'
	`

	rows, err := r.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query file identities: %w", err)
	}
	defer rows.Close()

	identities := make(map[string]*ingestion.FileIdentity)

	for rows.Next() {
		var identity ingestion.FileIdentity
		var historicalPathsJSON []byte
		var lastModifiedAt sql.NullTime

		err := rows.Scan(
			&identity.CanonicalPath,
			&historicalPathsJSON,
			&identity.FirstSeenCommitSHA,
			&identity.LastModifiedCommitSHA,
			&lastModifiedAt,
			&identity.Language,
			&identity.FileType,
			&identity.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Parse historical paths JSONB
		if err := json.Unmarshal(historicalPathsJSON, &identity.HistoricalPaths); err != nil {
			return nil, fmt.Errorf("failed to unmarshal historical paths for %s: %w", identity.CanonicalPath, err)
		}

		if lastModifiedAt.Valid {
			identity.LastModifiedAt = lastModifiedAt.Time
		}

		identities[identity.CanonicalPath] = &identity
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return identities, nil
}

// ResolveCanonicalPath resolves a historical path to its canonical path
func (r *FileIdentityRepository) ResolveCanonicalPath(ctx context.Context, repoID int64, historicalPath string) (string, error) {
	var canonicalPath sql.NullString

	query := `SELECT resolve_canonical_path($1, $2)`
	err := r.db.QueryRowContext(ctx, query, repoID, historicalPath).Scan(&canonicalPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve canonical path: %w", err)
	}

	if !canonicalPath.Valid {
		return "", fmt.Errorf("no canonical path found for %s", historicalPath)
	}

	return canonicalPath.String, nil
}

// BatchResolveCanonicalPaths resolves multiple historical paths to canonical paths
func (r *FileIdentityRepository) BatchResolveCanonicalPaths(ctx context.Context, repoID int64, historicalPaths []string) (map[string]string, error) {
	if len(historicalPaths) == 0 {
		return make(map[string]string), nil
	}

	var resultJSON []byte
	query := `SELECT batch_resolve_canonical_paths($1, $2)`

	// Convert string slice to PostgreSQL text array format
	err := r.db.QueryRowContext(ctx, query, repoID, historicalPaths).Scan(&resultJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to batch resolve canonical paths: %w", err)
	}

	// Parse JSON result
	var pathMap map[string]string
	if err := json.Unmarshal(resultJSON, &pathMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal path mapping: %w", err)
	}

	return pathMap, nil
}

// GetCount returns the number of file identities for a repository
func (r *FileIdentityRepository) GetCount(ctx context.Context, repoID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM file_identity_map WHERE repo_id = $1 AND status = 'active'`
	err := r.db.QueryRowContext(ctx, query, repoID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get file identity count: %w", err)
	}
	return count, nil
}

// DeleteByRepoID deletes all file identity mappings for a repository
func (r *FileIdentityRepository) DeleteByRepoID(ctx context.Context, repoID int64) error {
	query := `DELETE FROM file_identity_map WHERE repo_id = $1`
	result, err := r.db.ExecContext(ctx, query, repoID)
	if err != nil {
		return fmt.Errorf("failed to delete file identities: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	fmt.Printf("🗑️  Deleted %d file identity mappings for repo %d\n", rowsAffected, repoID)
	return nil
}

// UpdateIncrementalRenames updates file identity map based on new commits
// This is used for incremental updates after the initial ingestion
func (r *FileIdentityRepository) UpdateIncrementalRenames(ctx context.Context, repoID int64, renames map[string]string, commitSHA string, commitTime time.Time) error {
	if len(renames) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// For each rename: old_path -> new_path
	// 1. Find the file identity by searching historical_paths for old_path
	// 2. Update canonical_path to new_path
	// 3. Append new_path to historical_paths if not already present
	for oldPath, newPath := range renames {
		// Find existing identity by searching historical paths
		var id int64
		var historicalPathsJSON []byte

		query := `
			SELECT id, historical_paths
			FROM file_identity_map
			WHERE repo_id = $1
			  AND (canonical_path = $2 OR historical_paths @> jsonb_build_array($2))
			LIMIT 1
		`
		err := tx.QueryRowContext(ctx, query, repoID, oldPath).Scan(&id, &historicalPathsJSON)
		if err != nil {
			if err == sql.ErrNoRows {
				// File not found - might be a new file, skip
				continue
			}
			return fmt.Errorf("failed to find file identity for %s: %w", oldPath, err)
		}

		// Parse historical paths
		var historicalPaths []string
		if err := json.Unmarshal(historicalPathsJSON, &historicalPaths); err != nil {
			return fmt.Errorf("failed to unmarshal historical paths: %w", err)
		}

		// Append new path if not already present
		found := false
		for _, p := range historicalPaths {
			if p == newPath {
				found = true
				break
			}
		}
		if !found {
			historicalPaths = append(historicalPaths, newPath)
		}

		// Update record
		updatedHistoricalJSON, err := ingestion.HistoricalPathsToJSON(historicalPaths)
		if err != nil {
			return fmt.Errorf("failed to marshal updated historical paths: %w", err)
		}

		updateQuery := `
			UPDATE file_identity_map
			SET canonical_path = $1,
			    historical_paths = $2,
			    last_modified_commit_sha = $3,
			    last_modified_at = $4,
			    last_updated_at = NOW()
			WHERE id = $5
		`
		_, err = tx.ExecContext(ctx, updateQuery, newPath, updatedHistoricalJSON, commitSHA, commitTime, id)
		if err != nil {
			return fmt.Errorf("failed to update file identity for %s -> %s: %w", oldPath, newPath, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	fmt.Printf("✅ Updated %d file renames incrementally\n", len(renames))
	return nil
}
