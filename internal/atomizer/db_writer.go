package atomizer

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// DBWriter handles all PostgreSQL write operations for code blocks
// Reference: AGENT_P2B_PROCESSOR.md - Database writes
// Reference: migrations/001_code_block_schema.sql - Schema definition
type DBWriter struct {
	db *sql.DB
}

// NewDBWriter creates a new database writer for code blocks
func NewDBWriter(db *sql.DB) *DBWriter {
	return &DBWriter{db: db}
}

// CreateCodeBlock inserts a new code block into PostgreSQL
// Returns the PostgreSQL ID of the created block
// Reference: AGENT_P2B_PROCESSOR.md - CREATE_BLOCK handling
func (w *DBWriter) CreateCodeBlock(ctx context.Context, event *ChangeEvent, commitSHA, authorEmail string, timestamp time.Time, repoID int64) (int64, error) {
	// Use TargetFile as both canonical_file_path and path_at_creation
	// canonical_file_path will be updated by file identity resolution later
	query := `
		INSERT INTO code_blocks (
			repo_id, file_path, block_name, block_type,
			language, first_seen_sha, current_status,
			canonical_file_path, path_at_creation,
			start_line, end_line,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, 'active', $7, $8, $9, $10, NOW(), NOW())
		ON CONFLICT (repo_id, canonical_file_path, block_name)
		DO UPDATE SET
			updated_at = NOW(),
			file_path = EXCLUDED.file_path,
			start_line = EXCLUDED.start_line,
			end_line = EXCLUDED.end_line
		RETURNING id
	`

	var blockID int64
	err := w.db.QueryRowContext(ctx, query,
		repoID,
		event.TargetFile,        // file_path (current path)
		event.TargetBlockName,
		event.BlockType,
		detectLanguage(event.TargetFile),
		commitSHA,
		event.TargetFile,        // canonical_file_path (will be resolved later)
		event.TargetFile,        // path_at_creation
		event.StartLine,         // start_line
		event.EndLine,           // end_line
	).Scan(&blockID)

	if err != nil {
		return 0, fmt.Errorf("failed to create code block %s:%s: %w", event.TargetFile, event.TargetBlockName, err)
	}

	return blockID, nil
}

// CreateModification records a change event to code_block_changes table
// Reference: DATA_SCHEMA_REFERENCE.md lines 282-301 - code_block_changes table schema
// Reference: AGENT_P2B_PROCESSOR.md - Change event tracking
func (w *DBWriter) CreateModification(ctx context.Context, blockID int64, repoID int64, event *ChangeEvent, commitSHA, authorEmail string, timestamp time.Time, changeSummary string) error {
	// Calculate lines added/deleted from old_version and new_version
	linesAdded := 0
	linesDeleted := 0

	// TODO: Parse old_version and new_version to calculate line changes
	// For now, use basic heuristics
	if event.NewVersion != "" && event.OldVersion != "" {
		oldLines := strings.Count(event.OldVersion, "\n") + 1
		newLines := strings.Count(event.NewVersion, "\n") + 1
		if newLines > oldLines {
			linesAdded = newLines - oldLines
		} else {
			linesDeleted = oldLines - newLines
		}
	}

	// canonical_file_path = commit_time_path for now (will be resolved by identity mapper later)
	canonicalFilePath := event.TargetFile
	commitTimePath := event.TargetFile

	// Convert behavior to change_type per DATA_SCHEMA_REFERENCE.md
	changeType := ""
	switch event.Behavior {
	case "CREATE_BLOCK":
		changeType = "created"
	case "MODIFY_BLOCK":
		changeType = "modified"
	case "DELETE_BLOCK":
		changeType = "deleted"
	case "RENAME_BLOCK":
		changeType = "renamed"
	default:
		changeType = "modified" // fallback
	}

	query := `
		INSERT INTO code_block_changes (
			repo_id, commit_sha, block_id,
			canonical_file_path, commit_time_path,
			block_type, block_name, change_type,
			old_name, lines_added, lines_deleted,
			complexity_delta, change_summary, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
		ON CONFLICT (repo_id, commit_sha, COALESCE(block_id, 0)) DO UPDATE SET
			canonical_file_path = EXCLUDED.canonical_file_path,
			commit_time_path = EXCLUDED.commit_time_path,
			block_type = EXCLUDED.block_type,
			block_name = EXCLUDED.block_name,
			change_type = EXCLUDED.change_type,
			lines_added = EXCLUDED.lines_added,
			lines_deleted = EXCLUDED.lines_deleted,
			change_summary = EXCLUDED.change_summary
	`

	_, err := w.db.ExecContext(ctx, query,
		repoID,
		commitSHA,
		blockID,
		canonicalFilePath,
		commitTimePath,
		event.BlockType,
		event.TargetBlockName,
		changeType, // converted from CREATE_BLOCK -> created, etc.
		nil,        // old_name (for renames - not yet implemented)
		linesAdded,
		linesDeleted,
		0,              // complexity_delta (not yet calculated)
		changeSummary,  // LLM summary of the change
	)

	if err != nil {
		return fmt.Errorf("failed to create change event for block %d: %w", blockID, err)
	}

	return nil
}

// UpdateCodeBlock updates the last_modified metadata for a code block
// Used during MODIFY_BLOCK operations
func (w *DBWriter) UpdateCodeBlock(ctx context.Context, blockID int64, commitSHA, authorEmail string, timestamp time.Time) error {
	query := `
		UPDATE code_blocks
		SET updated_at = NOW()
		WHERE id = $1
	`

	result, err := w.db.ExecContext(ctx, query, blockID)
	if err != nil {
		return fmt.Errorf("failed to update code block %d: %w", blockID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("code block %d not found", blockID)
	}

	return nil
}

// MarkCodeBlockDeleted marks a code block as deleted (soft delete)
// Reference: AGENT_P2B_PROCESSOR.md - DELETE_BLOCK edge case
func (w *DBWriter) MarkCodeBlockDeleted(ctx context.Context, blockID int64, commitSHA string, timestamp time.Time) error {
	query := `
		UPDATE code_blocks
		SET current_status = 'deleted',
		    updated_at = NOW()
		WHERE id = $1
	`

	result, err := w.db.ExecContext(ctx, query, blockID)
	if err != nil {
		return fmt.Errorf("failed to mark code block %d as deleted: %w", blockID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("code block %d not found", blockID)
	}

	return nil
}

// GetCodeBlockID retrieves the PostgreSQL ID for a code block by file path and name
// Used to initialize state tracker from existing data
// Uses canonical_file_path for matching per DATA_SCHEMA_REFERENCE.md
func (w *DBWriter) GetCodeBlockID(ctx context.Context, repoID int64, filePath, blockName string) (int64, error) {
	query := `
		SELECT id FROM code_blocks
		WHERE repo_id = $1 AND canonical_file_path = $2 AND block_name = $3
	`

	var blockID int64
	err := w.db.QueryRowContext(ctx, query, repoID, filePath, blockName).Scan(&blockID)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("code block not found: %s:%s", filePath, blockName)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get code block ID: %w", err)
	}

	return blockID, nil
}

// LoadExistingBlocks loads all existing code blocks for a repository into the state tracker
// Used to initialize state before processing commits
func (w *DBWriter) LoadExistingBlocks(ctx context.Context, repoID int64, state *StateTracker) error {
	query := `
		SELECT id, file_path, block_name
		FROM code_blocks
		WHERE repo_id = $1
	`

	rows, err := w.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return fmt.Errorf("failed to load existing blocks: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int64
		var filePath, blockName string

		if err := rows.Scan(&id, &filePath, &blockName); err != nil {
			return fmt.Errorf("failed to scan block: %w", err)
		}

		state.SetBlockID(filePath, blockName, id)
		count++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating blocks: %w", err)
	}

	return nil
}

// detectLanguage infers programming language from file extension
// Reference: AGENT_P2B_PROCESSOR.md - Language detection
func detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	languageMap := map[string]string{
		".go":   "go",
		".py":   "python",
		".js":   "javascript",
		".ts":   "typescript",
		".tsx":  "typescript",
		".jsx":  "javascript",
		".java": "java",
		".c":    "c",
		".cpp":  "cpp",
		".cs":   "csharp",
		".rb":   "ruby",
		".php":  "php",
		".rs":   "rust",
		".kt":   "kotlin",
		".swift": "swift",
	}

	if lang, ok := languageMap[ext]; ok {
		return lang
	}

	return "unknown"
}

// MarkCommitAtomized updates the atomized_at timestamp for a commit
// This enables idempotency - the commit won't be reprocessed on subsequent runs
// Reference: Migration 009 - Idempotency tracking
func (w *DBWriter) MarkCommitAtomized(ctx context.Context, commitSHA string, repoID int64) error {
	query := `
		UPDATE github_commits
		SET atomized_at = NOW()
		WHERE repo_id = $1 AND sha = $2
	`

	result, err := w.db.ExecContext(ctx, query, repoID, commitSHA)
	if err != nil {
		return fmt.Errorf("failed to update atomized_at for commit %s: %w", commitSHA, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("commit %s not found in repo %d", commitSHA, repoID)
	}

	return nil
}
