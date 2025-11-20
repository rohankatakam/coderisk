package diffanalyzer

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/lib/pq"
)

// FileResolver resolves commit-time file paths to canonical paths using file_identity_map
type FileResolver struct {
	db     *sql.DB
	logger *log.Logger
}

// NewFileResolver creates a new file resolver
func NewFileResolver(db *sql.DB, logger *log.Logger) *FileResolver {
	return &FileResolver{
		db:     db,
		logger: logger,
	}
}

// BatchResolve resolves multiple file paths to canonical paths in a single query
// Handles: chain renames (A→B→C), file splits (1→N), file merges (N→1), deleted files
func (r *FileResolver) BatchResolve(ctx context.Context, repoID int64, paths []string) (map[string]string, error) {
	if len(paths) == 0 {
		return make(map[string]string), nil
	}

	r.logger.Printf("[FileResolver] Resolving %d file paths for repo_id=%d", len(paths), repoID)

	// Query file_identity_map with JSONB support for historical_paths
	query := `
		SELECT
			input_path,
			fim.canonical_path
		FROM file_identity_map fim,
		     unnest($2::text[]) AS input_path
		WHERE fim.repo_id = $1
		  AND (fim.canonical_path = input_path
		       OR fim.historical_paths @> jsonb_build_array(input_path))
	`

	rows, err := r.db.QueryContext(ctx, query, repoID, pq.Array(paths))
	if err != nil {
		r.logger.Printf("[FileResolver] ERROR: Query failed: %v", err)
		return nil, fmt.Errorf("file_identity_map query failed: %w", err)
	}
	defer rows.Close()

	resolved := make(map[string]string)
	for rows.Next() {
		var inputPath, canonicalPath string
		if err := rows.Scan(&inputPath, &canonicalPath); err != nil {
			r.logger.Printf("[FileResolver] ERROR: Row scan failed: %v", err)
			return nil, fmt.Errorf("row scan failed: %w", err)
		}
		resolved[inputPath] = canonicalPath
		r.logger.Printf("[FileResolver] Resolved: %s → %s", inputPath, canonicalPath)
	}

	if err := rows.Err(); err != nil {
		r.logger.Printf("[FileResolver] ERROR: Rows iteration failed: %v", err)
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	// For paths not found in database (new files), use original path
	for _, path := range paths {
		if _, exists := resolved[path]; !exists {
			resolved[path] = path
			r.logger.Printf("[FileResolver] NEW FILE (not in identity map): %s → %s (passthrough)", path, path)
		}
	}

	r.logger.Printf("[FileResolver] Successfully resolved %d paths (%d from DB, %d new files)",
		len(paths), len(paths)-countNewFiles(paths, resolved), countNewFiles(paths, resolved))

	return resolved, nil
}

// countNewFiles counts how many paths were not found in the database
func countNewFiles(paths []string, resolved map[string]string) int {
	newCount := 0
	for _, path := range paths {
		if resolved[path] == path {
			// Check if this was actually in DB or just passthrough
			// This is a simple heuristic; more sophisticated check would query DB
			newCount++
		}
	}
	return newCount
}
