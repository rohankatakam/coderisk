package mcp

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rohankatakam/coderisk/internal/mcp/tools"
)

// LocalGraphClient queries Neo4j and PostgreSQL directly
type LocalGraphClient struct {
	neo4jDriver neo4j.DriverWithContext
	pgPool      *pgxpool.Pool
}

// NewLocalGraphClient creates a new local graph client
func NewLocalGraphClient(driver neo4j.DriverWithContext, pgPool *pgxpool.Pool) *LocalGraphClient {
	return &LocalGraphClient{
		neo4jDriver: driver,
		pgPool:      pgPool,
	}
}

// GetCodeBlocksForFile queries Neo4j for code blocks in a file
func (c *LocalGraphClient) GetCodeBlocksForFile(ctx context.Context, filePath string, historicalPaths []string, repoID int) ([]tools.CodeBlock, error) {
	// Query Neo4j directly (no HTTP, local connection)
	session := c.neo4jDriver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	allPaths := append([]string{filePath}, historicalPaths...)

	// CRITICAL: CodeBlock nodes are not linked to File nodes in current implementation
	// Query CodeBlocks directly by file_path property
	// IMPORTANT: b.db_id is the PostgreSQL integer ID used for temporal queries
	query := `
		MATCH (b:CodeBlock)
		WHERE b.file_path IN $paths AND b.repo_id = $repoID
		RETURN b.db_id AS id,
		       b.name AS name,
		       b.block_type AS type,
		       b.file_path AS path,
		       b.original_author AS original_author,
		       b.last_modifier AS last_modifier,
		       b.staleness_days AS staleness_days,
		       b.familiarity_map AS familiarity_map,
		       b.semantic_importance AS semantic_importance
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"paths":  allPaths,
		"repoID": repoID,
	})
	if err != nil {
		return nil, err
	}

	var blocks []tools.CodeBlock
	for result.Next(ctx) {
		record := result.Record()

		// Handle NULL values gracefully
		var id, name, blockType, path string
		var originalAuthor, lastModifier, familiarityMap, semanticImportance string
		var stalenessDays int64

		// Required fields
		// db_id is int64 from PostgreSQL, convert to string for consistency
		if record.Values[0] != nil {
			dbID := record.Values[0].(int64)
			id = fmt.Sprintf("%d", dbID)
		}
		if record.Values[1] != nil {
			name = record.Values[1].(string)
		}
		if record.Values[2] != nil {
			blockType = record.Values[2].(string)
		}
		if record.Values[3] != nil {
			path = record.Values[3].(string)
		}

		// Optional ownership fields
		if record.Values[4] != nil {
			originalAuthor = record.Values[4].(string)
		}
		if record.Values[5] != nil {
			lastModifier = record.Values[5].(string)
		}
		if record.Values[6] != nil {
			stalenessDays = record.Values[6].(int64)
		}
		if record.Values[7] != nil {
			familiarityMap = record.Values[7].(string)
		}
		if record.Values[8] != nil {
			semanticImportance = record.Values[8].(string)
		}

		blocks = append(blocks, tools.CodeBlock{
			ID:   id,
			Name: name,
			Type: blockType,
			Path: path,
			OwnershipData: tools.OwnershipData{
				OriginalAuthor:     originalAuthor,
				LastModifier:       lastModifier,
				StaleDays:          int(stalenessDays),
				FamiliarityMap:     familiarityMap,
				SemanticImportance: semanticImportance,
			},
		})
	}

	return blocks, result.Err()
}

// GetCouplingData queries PostgreSQL for coupling relationships
// blockID is the PostgreSQL integer ID (as string)
func (c *LocalGraphClient) GetCouplingData(ctx context.Context, blockID string) (*tools.CouplingData, error) {
	// Coupling data is stored in PostgreSQL code_block_coupling table
	// Find all blocks coupled with the given blockID (both as block_a and block_b)
	query := `
		SELECT
			CASE
				WHEN cbc.block_a_id = $1 THEN cbc.block_b_id
				ELSE cbc.block_a_id
			END AS coupled_block_id,
			cb.block_name,
			cb.file_path,
			cbc.co_change_rate,
			cbc.co_change_count
		FROM code_block_coupling cbc
		JOIN code_blocks cb ON cb.id = CASE
			WHEN cbc.block_a_id = $1 THEN cbc.block_b_id
			ELSE cbc.block_a_id
		END
		WHERE (cbc.block_a_id = $1 OR cbc.block_b_id = $1)
		  AND cbc.co_change_rate >= 0.5
		ORDER BY cbc.co_change_rate DESC
		LIMIT 20
	`

	rows, err := c.pgPool.Query(ctx, query, blockID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var coupledBlocks []tools.CoupledBlock
	for rows.Next() {
		var coupledID int64
		var coupledName, coupledPath string
		var rate float64
		var coChangeCount int

		err := rows.Scan(&coupledID, &coupledName, &coupledPath, &rate, &coChangeCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan coupling row: %w", err)
		}

		coupledBlocks = append(coupledBlocks, tools.CoupledBlock{
			ID:            fmt.Sprintf("%d", coupledID),
			Name:          coupledName,
			Path:          coupledPath,
			Rate:          rate,
			CoChangeCount: coChangeCount,
		})
	}

	return &tools.CouplingData{
		CoupledWith: coupledBlocks,
	}, nil
}

// GetTemporalData queries PostgreSQL for temporal incident data
func (c *LocalGraphClient) GetTemporalData(ctx context.Context, blockID string) (*tools.TemporalData, error) {
	// Query PostgreSQL code_block_incidents table
	query := `
		SELECT
			cbi.issue_id,
			cbi.confidence,
			gi.title AS issue_title,
			gi.state AS issue_state
		FROM code_block_incidents cbi
		LEFT JOIN github_issues gi ON cbi.issue_id = gi.id
		WHERE cbi.code_block_id = $1
		ORDER BY cbi.confidence DESC
		LIMIT 10
	`

	rows, err := c.pgPool.Query(ctx, query, blockID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []tools.TemporalIncident
	for rows.Next() {
		var incident tools.TemporalIncident
		var issueID sql.NullInt64
		var issueTitle, issueState sql.NullString

		err := rows.Scan(
			&issueID,
			&incident.ConfidenceScore,
			&issueTitle,
			&issueState,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan temporal incident: %w", err)
		}

		if issueID.Valid {
			incident.IssueID = int(issueID.Int64)
			incident.IssueTitle = issueTitle.String
			incident.IssueState = issueState.String
		}

		incidents = append(incidents, incident)
	}

	return &tools.TemporalData{
		IncidentCount: len(incidents),
		Incidents:     incidents,
	}, nil
}

// GetCodeBlocksByNames queries Neo4j for specific code blocks by name
// This is used for diff-based analysis where we know the block names from LLM extraction
func (c *LocalGraphClient) GetCodeBlocksByNames(ctx context.Context, filePathsWithHistorical map[string][]string, blockNames []string, repoID int) ([]tools.CodeBlock, error) {
	if len(blockNames) == 0 {
		return nil, fmt.Errorf("no block names provided")
	}

	session := c.neo4jDriver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Build list of all paths to search (current + historical for each file)
	var allPaths []string
	for _, historical := range filePathsWithHistorical {
		allPaths = append(allPaths, historical...)
	}

	// Query for blocks that match any of the block names in any of the file paths
	// IMPORTANT: b.db_id is the PostgreSQL integer ID used for temporal queries
	query := `
		MATCH (b:CodeBlock)
		WHERE b.file_path IN $paths
		  AND b.name IN $blockNames
		  AND b.repo_id = $repoID
		RETURN b.db_id AS id,
		       b.name AS name,
		       b.block_type AS type,
		       b.file_path AS path,
		       b.original_author AS original_author,
		       b.last_modifier AS last_modifier,
		       b.staleness_days AS staleness_days,
		       b.familiarity_map AS familiarity_map,
		       b.semantic_importance AS semantic_importance
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"paths":      allPaths,
		"blockNames": blockNames,
		"repoID":     repoID,
	})
	if err != nil {
		return nil, err
	}

	var blocks []tools.CodeBlock
	for result.Next(ctx) {
		record := result.Record()

		// Handle NULL values gracefully
		var id, name, blockType, path string
		var originalAuthor, lastModifier, familiarityMap, semanticImportance string
		var stalenessDays int64

		// Required fields
		// db_id is int64 from PostgreSQL, convert to string for consistency
		if record.Values[0] != nil {
			dbID := record.Values[0].(int64)
			id = fmt.Sprintf("%d", dbID)
		}
		if record.Values[1] != nil {
			name = record.Values[1].(string)
		}
		if record.Values[2] != nil {
			blockType = record.Values[2].(string)
		}
		if record.Values[3] != nil {
			path = record.Values[3].(string)
		}

		// Optional ownership fields
		if record.Values[4] != nil {
			originalAuthor = record.Values[4].(string)
		}
		if record.Values[5] != nil {
			lastModifier = record.Values[5].(string)
		}
		if record.Values[6] != nil {
			stalenessDays = record.Values[6].(int64)
		}
		if record.Values[7] != nil {
			familiarityMap = record.Values[7].(string)
		}
		if record.Values[8] != nil {
			semanticImportance = record.Values[8].(string)
		}

		blocks = append(blocks, tools.CodeBlock{
			ID:   id,
			Name: name,
			Type: blockType,
			Path: path,
			OwnershipData: tools.OwnershipData{
				OriginalAuthor:     originalAuthor,
				LastModifier:       lastModifier,
				StaleDays:          int(stalenessDays),
				FamiliarityMap:     familiarityMap,
				SemanticImportance: semanticImportance,
			},
		})
	}

	return blocks, result.Err()
}
