package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// BlockChangeEvent represents a single change event in a block's history
type BlockChangeEvent struct {
	CommitSHA        string
	CommitMessage    string
	CommitDate       time.Time
	AuthorEmail      string
	AuthorName       string
	ChangeType       string  // CREATED, MODIFIED, DELETED, RENAMED
	ComplexityDelta  int
	LinesAdded       int
	LinesDeleted     int
	OldBlockName     string  // For renames
	IssueNumber      *int    // Linked incident (if any)
	IssueTitle       *string
	IssueSeverity    *string
}

// GetBlockHistory retrieves the chronological history of changes to a code block
func GetBlockHistory(ctx context.Context, db *sqlx.DB, blockID int64, limit int) ([]BlockChangeEvent, error) {
	query := `
		SELECT
			gc.sha,
			gc.message,
			gc.author_date,
			gc.author_email,
			gc.author_name,
			cbc.change_type,
			COALESCE(cbc.complexity_delta, 0),
			COALESCE(cbc.lines_added, 0),
			COALESCE(cbc.lines_deleted, 0),
			cbc.old_name,
			gi.number,
			gi.title,
			gi.state
		FROM code_block_changes cbc
		JOIN github_commits gc ON gc.sha = cbc.commit_sha AND gc.repo_id = cbc.repo_id
		LEFT JOIN code_block_incidents cbi
			ON cbi.block_id = cbc.block_id
			AND cbi.incident_date BETWEEN gc.author_date - INTERVAL '7 days'
			AND gc.author_date + INTERVAL '7 days'
		LEFT JOIN github_issues gi
			ON gi.id = cbi.issue_id
		WHERE cbc.block_id = $1
		ORDER BY gc.topological_index DESC
		LIMIT $2
	`

	rows, err := db.QueryContext(ctx, query, blockID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query block history: %w", err)
	}
	defer rows.Close()

	var events []BlockChangeEvent
	for rows.Next() {
		var event BlockChangeEvent
		var oldBlockName sql.NullString
		var issueNumber sql.NullInt64
		var issueTitle, issueSeverity sql.NullString

		err := rows.Scan(
			&event.CommitSHA,
			&event.CommitMessage,
			&event.CommitDate,
			&event.AuthorEmail,
			&event.AuthorName,
			&event.ChangeType,
			&event.ComplexityDelta,
			&event.LinesAdded,
			&event.LinesDeleted,
			&oldBlockName,
			&issueNumber,
			&issueTitle,
			&issueSeverity,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan block change event: %w", err)
		}

		if oldBlockName.Valid {
			event.OldBlockName = oldBlockName.String
		}
		if issueNumber.Valid {
			num := int(issueNumber.Int64)
			event.IssueNumber = &num
		}
		if issueTitle.Valid {
			event.IssueTitle = &issueTitle.String
		}
		if issueSeverity.Valid {
			event.IssueSeverity = &issueSeverity.String
		}

		events = append(events, event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating block history: %w", err)
	}

	return events, nil
}

// GetBlockWithRenameChain retrieves a block with its complete rename history
func GetBlockWithRenameChain(ctx context.Context, db *sqlx.DB, blockID int64) (*BlockWithHistory, error) {
	query := `
		SELECT
			id,
			block_name,
			block_type,
			signature,
			canonical_file_path,
			COALESCE(historical_block_names, '[]'::jsonb)::text,
			start_line,
			end_line,
			repo_id
		FROM code_blocks
		WHERE id = $1
	`

	var block BlockWithHistory
	var historicalNames string

	err := db.QueryRowContext(ctx, query, blockID).Scan(
		&block.ID,
		&block.BlockName,
		&block.BlockType,
		&block.Signature,
		&block.CanonicalFilePath,
		&historicalNames,
		&block.StartLine,
		&block.EndLine,
		&block.RepoID,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("code block not found: %s", blockID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query code block: %w", err)
	}

	// Parse historical names
	block.RenameChain = parseRenameChain(historicalNames)

	return &block, nil
}

// BlockWithHistory represents a code block with its rename chain
type BlockWithHistory struct {
	ID                int64
	BlockName         string
	BlockType         string
	Signature         string
	CanonicalFilePath string
	StartLine         int
	EndLine           int
	RepoID            int64
	RenameChain       []string // Historical names in order
}

// parseRenameChain extracts the rename chain from JSONB
func parseRenameChain(jsonbStr string) []string {
	// Simplified JSON parsing - just extract strings between quotes
	// In production, use encoding/json for proper parsing
	chain := []string{}

	// Remove brackets
	jsonbStr = jsonbStr[1 : len(jsonbStr)-1]

	if jsonbStr == "" {
		return chain
	}

	// Split by commas and clean up
	parts := splitJSON(jsonbStr)
	for _, part := range parts {
		// Remove quotes
		name := part[1 : len(part)-1]
		chain = append(chain, name)
	}

	return chain
}

// splitJSON splits a JSON array string by commas (simplified)
func splitJSON(s string) []string {
	var result []string
	var current string
	inQuotes := false

	for _, char := range s {
		if char == '"' {
			inQuotes = !inQuotes
			current += string(char)
		} else if char == ',' && !inQuotes {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}
