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
	// NOTE: Ownership data (original_author, staleness_days, etc.) must be fetched from PostgreSQL,
	//       NOT from Neo4j. Neo4j only stores: id, name, block_type, file_path, language, db_id, repo_id
	query := `
		MATCH (b:CodeBlock)
		WHERE b.file_path IN $paths AND b.repo_id = $repoID
		RETURN b.db_id AS id,
		       b.name AS name,
		       b.block_type AS type,
		       b.file_path AS path
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"paths":  allPaths,
		"repoID": repoID,
	})
	if err != nil {
		return nil, err
	}

	var blocks []tools.CodeBlock
	var blockIDs []string // Track IDs for batch PostgreSQL lookup

	for result.Next(ctx) {
		record := result.Record()

		// Handle NULL values gracefully
		var id, name, blockType, path string

		// Required fields from Neo4j (only 4 values now)
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

		blocks = append(blocks, tools.CodeBlock{
			ID:   id,
			Name: name,
			Type: blockType,
			Path: path,
			// Ownership data will be fetched from PostgreSQL below
		})
		blockIDs = append(blockIDs, id)
	}

	if err := result.Err(); err != nil {
		return nil, err
	}

	// Fetch ownership data from PostgreSQL for all blocks in one query
	if len(blockIDs) > 0 {
		ownership, err := c.getOwnershipDataBatch(ctx, blockIDs)
		if err != nil {
			// Non-fatal: continue with blocks but log the error
			fmt.Printf("Warning: failed to fetch ownership data: %v\n", err)
		} else {
			// Merge ownership data into blocks
			for i := range blocks {
				if ownerData, ok := ownership[blocks[i].ID]; ok {
					blocks[i].OwnershipData = ownerData
				}
			}
		}
	}

	return blocks, nil
}

// ExecuteQuery implements git.GraphQueryer interface for Neo4j queries
// This allows LocalGraphClient to be used with git.FileResolver
func (c *LocalGraphClient) ExecuteQuery(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	session := c.neo4jDriver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, err
	}

	var results []map[string]any
	for result.Next(ctx) {
		record := result.Record()

		// Convert neo4j.Record to map[string]any
		recordMap := make(map[string]any)
		for _, key := range record.Keys {
			value, ok := record.Get(key)
			if ok {
				recordMap[key] = value
			}
		}
		results = append(results, recordMap)
	}

	return results, result.Err()
}

// GetCouplingData queries Neo4j for coupling relationships
// blockID is the PostgreSQL integer ID (as string)
func (c *LocalGraphClient) GetCouplingData(ctx context.Context, blockID string) (*tools.CouplingData, error) {
	// Coupling data is now stored in Neo4j as CO_CHANGES_WITH edges
	// Query Neo4j for blocks coupled with the given blockID

	// Convert string blockID to int64 for Neo4j query
	// db_id property in Neo4j is int64, not string
	var blockIDVal int64
	_, err := fmt.Sscanf(blockID, "%d", &blockIDVal)
	if err != nil {
		return nil, fmt.Errorf("failed to parse blockID '%s': %w", blockID, err)
	}

	session := c.neo4jDriver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (b:CodeBlock {db_id: $blockID})-[r:CO_CHANGES_WITH]-(coupled:CodeBlock)
		WHERE r.rate >= 0.5
		RETURN coupled.db_id AS id,
		       coupled.name AS name,
		       coupled.file_path AS path,
		       r.rate AS rate,
		       r.co_change_count AS count
		ORDER BY r.rate DESC
		LIMIT 20
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"blockID": blockIDVal,  // Use int64, not string
	})
	if err != nil {
		return nil, err
	}

	var coupledBlocks []tools.CoupledBlock
	for result.Next(ctx) {
		record := result.Record()

		// Handle NULL values gracefully
		var coupledID int64
		var coupledName, coupledPath string
		var rate float64
		var coChangeCount int64

		if record.Values[0] != nil {
			coupledID = record.Values[0].(int64)
		}
		if record.Values[1] != nil {
			coupledName = record.Values[1].(string)
		}
		if record.Values[2] != nil {
			coupledPath = record.Values[2].(string)
		}
		if record.Values[3] != nil {
			rate = record.Values[3].(float64)
		}
		if record.Values[4] != nil {
			coChangeCount = record.Values[4].(int64)
		}

		coupledBlocks = append(coupledBlocks, tools.CoupledBlock{
			ID:            fmt.Sprintf("%d", coupledID),
			Name:          coupledName,
			Path:          coupledPath,
			Rate:          rate,
			CoChangeCount: int(coChangeCount),
		})
	}

	return &tools.CouplingData{
		CoupledWith: coupledBlocks,
	}, result.Err()
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
	// NOTE: Ownership data must be fetched from PostgreSQL separately
	query := `
		MATCH (b:CodeBlock)
		WHERE b.file_path IN $paths
		  AND b.name IN $blockNames
		  AND b.repo_id = $repoID
		RETURN b.db_id AS id,
		       b.name AS name,
		       b.block_type AS type,
		       b.file_path AS path
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
	var blockIDs []string

	for result.Next(ctx) {
		record := result.Record()

		// Handle NULL values gracefully
		var id, name, blockType, path string

		// Required fields from Neo4j (only 4 values now)
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

		blocks = append(blocks, tools.CodeBlock{
			ID:   id,
			Name: name,
			Type: blockType,
			Path: path,
			// Ownership data will be fetched from PostgreSQL below
		})
		blockIDs = append(blockIDs, id)
	}

	if err := result.Err(); err != nil {
		return nil, err
	}

	// Fetch ownership data from PostgreSQL for all blocks in one query
	if len(blockIDs) > 0 {
		ownership, err := c.getOwnershipDataBatch(ctx, blockIDs)
		if err != nil {
			// Non-fatal: continue with blocks but log the error
			fmt.Printf("Warning: failed to fetch ownership data: %v\n", err)
		} else {
			// Merge ownership data into blocks
			for i := range blocks {
				if ownerData, ok := ownership[blocks[i].ID]; ok {
					blocks[i].OwnershipData = ownerData
				}
			}
		}
	}

	return blocks, nil
}

// getOwnershipDataBatch fetches ownership data from PostgreSQL for multiple blocks in one query
// Computes ownership from code_block_modifications table
// Returns a map of blockID -> OwnershipData
func (c *LocalGraphClient) getOwnershipDataBatch(ctx context.Context, blockIDs []string) (map[string]tools.OwnershipData, error) {
	if len(blockIDs) == 0 {
		return nil, nil
	}

	// Convert string IDs to int64 for PostgreSQL query
	var intIDs []interface{}
	for _, id := range blockIDs {
		var intID int64
		_, err := fmt.Sscanf(id, "%d", &intID)
		if err != nil {
			continue // Skip invalid IDs
		}
		intIDs = append(intIDs, intID)
	}

	if len(intIDs) == 0 {
		return nil, nil
	}

	// Query code_block_modifications to compute ownership metrics
	// - Original author: first developer to modify the block (earliest modified_at)
	// - Last modifier: most recent developer to modify the block
	// - Staleness: days since last modification
	query := `
		WITH block_stats AS (
			SELECT
				code_block_id,
				FIRST_VALUE(developer_email) OVER (PARTITION BY code_block_id ORDER BY modified_at ASC) as first_author,
				FIRST_VALUE(developer_email) OVER (PARTITION BY code_block_id ORDER BY modified_at DESC) as last_author,
				MAX(modified_at) OVER (PARTITION BY code_block_id) as last_modified
			FROM code_block_modifications
			WHERE code_block_id = ANY($1)
		)
		SELECT DISTINCT
			code_block_id,
			first_author,
			last_author,
			EXTRACT(EPOCH FROM (NOW() - last_modified)) / 86400 as staleness_days
		FROM block_stats
	`

	rows, err := c.pgPool.Query(ctx, query, intIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to query code_block_modifications: %w", err)
	}
	defer rows.Close()

	result := make(map[string]tools.OwnershipData)
	for rows.Next() {
		var id int64
		var originalAuthor, lastModifier string
		var stalenessDays float64

		if err := rows.Scan(&id, &originalAuthor, &lastModifier, &stalenessDays); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		result[fmt.Sprintf("%d", id)] = tools.OwnershipData{
			OriginalAuthor: originalAuthor,
			LastModifier:   lastModifier,
			StaleDays:      int(stalenessDays),
			// FamiliarityMap and SemanticImportance would require additional computation
			// from modification history - can be added later if needed
		}
	}

	return result, rows.Err()
}

