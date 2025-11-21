package cli

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// CodeBlock represents a resolved code block
type CodeBlock struct {
	ID                   int64
	BlockName            string
	BlockType            string
	Signature            string
	CanonicalFilePath    string
	HistoricalBlockNames []string
	StartLine            int
	EndLine              int
}

// ResolveBlockByName finds code blocks matching the given name and file path
func ResolveBlockByName(ctx context.Context, db *sql.DB, repoID int64, blockName, filePath string) ([]CodeBlock, error) {
	query := `
		SELECT
			id,
			block_name,
			block_type,
			signature,
			canonical_file_path,
			COALESCE(historical_block_names, '[]'::jsonb),
			start_line,
			end_line
		FROM code_blocks
		WHERE repo_id = $1
			AND block_name = $2
			AND canonical_file_path = $3
		ORDER BY start_line ASC
	`

	rows, err := db.QueryContext(ctx, query, repoID, blockName, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to query code blocks: %w", err)
	}
	defer rows.Close()

	var blocks []CodeBlock
	for rows.Next() {
		var block CodeBlock
		var historicalNames string
		err := rows.Scan(
			&block.ID,
			&block.BlockName,
			&block.BlockType,
			&block.Signature,
			&block.CanonicalFilePath,
			&historicalNames,
			&block.StartLine,
			&block.EndLine,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan code block: %w", err)
		}

		// Parse historical names (simplified - just store as string for now)
		block.HistoricalBlockNames = parseHistoricalNames(historicalNames)
		blocks = append(blocks, block)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating code blocks: %w", err)
	}

	return blocks, nil
}

// ResolveBlockByID finds a code block by its ID
func ResolveBlockByID(ctx context.Context, db *sql.DB, blockID int64) (*CodeBlock, error) {
	query := `
		SELECT
			id,
			block_name,
			block_type,
			signature,
			canonical_file_path,
			COALESCE(historical_block_names, '[]'::jsonb),
			start_line,
			end_line
		FROM code_blocks
		WHERE id = $1
	`

	var block CodeBlock
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
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("code block not found: %s", blockID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query code block: %w", err)
	}

	block.HistoricalBlockNames = parseHistoricalNames(historicalNames)
	return &block, nil
}

// FindSimilarBlocks performs fuzzy search for similar block names
func FindSimilarBlocks(ctx context.Context, db *sql.DB, repoID int64, blockName, filePath string) ([]CodeBlock, error) {
	// Use PostgreSQL similarity search with trigrams or simple ILIKE
	query := `
		SELECT
			id,
			block_name,
			block_type,
			signature,
			canonical_file_path,
			COALESCE(historical_block_names, '[]'::jsonb),
			start_line,
			end_line
		FROM code_blocks
		WHERE repo_id = $1
			AND canonical_file_path = $2
			AND (
				block_name ILIKE $3
				OR block_name ILIKE '%' || $4 || '%'
			)
		ORDER BY
			CASE
				WHEN block_name ILIKE $3 THEN 1
				ELSE 2
			END,
			block_name
		LIMIT 10
	`

	likePattern := blockName + "%"
	rows, err := db.QueryContext(ctx, query, repoID, filePath, likePattern, blockName)
	if err != nil {
		return nil, fmt.Errorf("failed to query similar blocks: %w", err)
	}
	defer rows.Close()

	var blocks []CodeBlock
	for rows.Next() {
		var block CodeBlock
		var historicalNames string
		err := rows.Scan(
			&block.ID,
			&block.BlockName,
			&block.BlockType,
			&block.Signature,
			&block.CanonicalFilePath,
			&historicalNames,
			&block.StartLine,
			&block.EndLine,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan code block: %w", err)
		}

		block.HistoricalBlockNames = parseHistoricalNames(historicalNames)
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// parseHistoricalNames extracts block names from JSONB array
func parseHistoricalNames(jsonbStr string) []string {
	// Simplified parsing - just remove brackets and quotes
	cleaned := strings.Trim(jsonbStr, "[]")
	if cleaned == "" {
		return []string{}
	}

	parts := strings.Split(cleaned, ",")
	names := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.Trim(strings.TrimSpace(part), "\"")
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}
