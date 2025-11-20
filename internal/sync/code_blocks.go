package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// CodeBlockSyncer syncs code_blocks from PostgreSQL to Neo4j
type CodeBlockSyncer struct {
	db       *sql.DB
	driver   neo4j.DriverWithContext
	database string
}

// NewCodeBlockSyncer creates a new code block syncer
func NewCodeBlockSyncer(db *sql.DB, driver neo4j.DriverWithContext, database string) *CodeBlockSyncer {
	return &CodeBlockSyncer{
		db:       db,
		driver:   driver,
		database: database,
	}
}

// CodeBlock represents a code block from PostgreSQL
type CodeBlock struct {
	ID               int64
	RepoID           int64
	FilePath         string
	BlockName        string
	BlockType        string
	StartLine        *int
	EndLine          *int
	IncidentCount    int
	LastIncidentDate *time.Time
}

// SyncMissingBlocks syncs code blocks from PostgreSQL to Neo4j (incremental)
func (s *CodeBlockSyncer) SyncMissingBlocks(ctx context.Context, repoID int64) (int, error) {
	log.Printf("  📦 Syncing CodeBlocks from PostgreSQL → Neo4j...")

	// Step 1: Get all code blocks from PostgreSQL
	blocks, err := s.getAllBlocks(ctx, repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to get postgres blocks: %w", err)
	}
	log.Printf("     PostgreSQL: %d CodeBlocks", len(blocks))

	// Step 2: Get existing composite IDs from Neo4j
	neo4jIDs, err := s.getNeo4jBlockCompositeIDs(ctx, repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to get neo4j block IDs: %w", err)
	}
	log.Printf("     Neo4j: %d CodeBlocks", len(neo4jIDs))

	// Step 3: Compute delta (blocks not in Neo4j)
	neo4jIDSet := make(map[string]bool, len(neo4jIDs))
	for _, id := range neo4jIDs {
		neo4jIDSet[id] = true
	}

	var missingBlocks []CodeBlock
	for _, block := range blocks {
		compositeID := fmt.Sprintf("%d:codeblock:%s:%s", block.RepoID, block.FilePath, block.BlockName)
		if !neo4jIDSet[compositeID] {
			missingBlocks = append(missingBlocks, block)
		}
	}

	if len(missingBlocks) == 0 {
		log.Printf("     ✓ All CodeBlocks already in sync")
		return 0, nil
	}
	log.Printf("     Delta: %d CodeBlocks need syncing", len(missingBlocks))

	// Step 4: Create missing CodeBlock nodes in Neo4j
	synced := 0
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	for _, block := range missingBlocks {
		compositeID := fmt.Sprintf("%d:codeblock:%s:%s", block.RepoID, block.FilePath, block.BlockName)

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			// Create CodeBlock node with MERGE (idempotent)
			query := `
				MERGE (cb:CodeBlock {id: $compositeID, repo_id: $repoID})
				SET cb.db_id = $dbID,
				    cb.file_path = $filePath,
				    cb.block_name = $blockName,
				    cb.block_type = $blockType,
				    cb.start_line = $startLine,
				    cb.end_line = $endLine,
				    cb.incident_count = $incidentCount,
				    cb.last_incident_date = CASE WHEN $lastIncidentDate IS NOT NULL THEN datetime($lastIncidentDate) ELSE NULL END
				RETURN cb.id as id`

			// Convert nil pointers to null for Neo4j, format timestamps as RFC3339 strings
			var lastIncidentDate interface{}
			if block.LastIncidentDate != nil {
				lastIncidentDate = block.LastIncidentDate.Format(time.RFC3339)
			} else {
				lastIncidentDate = nil
			}

			params := map[string]any{
				"compositeID":      compositeID,
				"repoID":           block.RepoID,
				"dbID":             block.ID,
				"filePath":         block.FilePath,
				"blockName":        block.BlockName,
				"blockType":        block.BlockType,
				"startLine":        block.StartLine,
				"endLine":          block.EndLine,
				"incidentCount":    block.IncidentCount,
				"lastIncidentDate": lastIncidentDate,
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			log.Printf("     ⚠️  Failed to sync block %s:%s: %v", block.FilePath, block.BlockName, err)
			continue
		}
		synced++

		// Log progress every 100 blocks
		if synced%100 == 0 {
			log.Printf("     → Synced %d/%d blocks", synced, len(missingBlocks))
		}
	}

	log.Printf("     ✓ Synced %d CodeBlocks to Neo4j", synced)
	return synced, nil
}

// getAllBlocks returns all code blocks from PostgreSQL
func (s *CodeBlockSyncer) getAllBlocks(ctx context.Context, repoID int64) ([]CodeBlock, error) {
	query := `
		SELECT id, repo_id, file_path, block_name, block_type,
		       start_line, end_line, incident_count, last_incident_date
		FROM code_blocks
		WHERE repo_id = $1
		ORDER BY id`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocks []CodeBlock
	for rows.Next() {
		var b CodeBlock
		err := rows.Scan(
			&b.ID,
			&b.RepoID,
			&b.FilePath,
			&b.BlockName,
			&b.BlockType,
			&b.StartLine,
			&b.EndLine,
			&b.IncidentCount,
			&b.LastIncidentDate,
		)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, b)
	}

	return blocks, rows.Err()
}

// getNeo4jBlockCompositeIDs returns existing code block composite IDs from Neo4j
func (s *CodeBlockSyncer) getNeo4jBlockCompositeIDs(ctx context.Context, repoID int64) ([]string, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `MATCH (cb:CodeBlock {repo_id: $repoID}) RETURN cb.id as id`
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

