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
	query := `
		INSERT INTO code_blocks (
			repo_id, file_path, block_name, block_type,
			language, first_seen_sha, current_status,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, 'active', NOW(), NOW())
		ON CONFLICT (repo_id, file_path, block_name)
		DO UPDATE SET
			updated_at = NOW()
		RETURNING id
	`

	var blockID int64
	err := w.db.QueryRowContext(ctx, query,
		repoID,
		event.TargetFile,
		event.TargetBlockName,
		event.BlockType,
		detectLanguage(event.TargetFile),
		commitSHA,
	).Scan(&blockID)

	if err != nil {
		return 0, fmt.Errorf("failed to create code block %s:%s: %w", event.TargetFile, event.TargetBlockName, err)
	}

	return blockID, nil
}

// CreateModification records a modification event for a code block
// Reference: migrations/001_code_block_schema.sql - code_block_modifications table
func (w *DBWriter) CreateModification(ctx context.Context, blockID int64, repoID int64, commitSHA, authorEmail string, timestamp time.Time, modType string) error {
	// Parse additions/deletions from old_version and new_version if available
	additions := 0
	deletions := 0

	query := `
		INSERT INTO code_block_modifications (
			code_block_id, commit_sha, developer_email,
			change_type, modified_at, additions, deletions, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (code_block_id, commit_sha) DO NOTHING
	`

	_, err := w.db.ExecContext(ctx, query,
		blockID,
		commitSHA,
		authorEmail,
		modType,
		timestamp,
		additions,
		deletions,
	)

	if err != nil {
		return fmt.Errorf("failed to create modification for block %d: %w", blockID, err)
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
func (w *DBWriter) GetCodeBlockID(ctx context.Context, repoID int64, filePath, blockName string) (int64, error) {
	query := `
		SELECT id FROM code_blocks
		WHERE repo_id = $1 AND file_path = $2 AND block_name = $3
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
