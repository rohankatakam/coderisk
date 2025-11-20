package sync

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// IncidentEdgeSyncer syncs FIXED_BY_BLOCK edges from PostgreSQL to Neo4j
// Reference: DATA_SCHEMA_REFERENCE.md line 724-737 - FIXED_BY_BLOCK edge spec
type IncidentEdgeSyncer struct {
	db       *sql.DB
	driver   neo4j.DriverWithContext
	database string
}

// NewIncidentEdgeSyncer creates a new incident edge syncer
func NewIncidentEdgeSyncer(db *sql.DB, driver neo4j.DriverWithContext, database string) *IncidentEdgeSyncer {
	return &IncidentEdgeSyncer{
		db:       db,
		driver:   driver,
		database: database,
	}
}

// SyncFixedByBlockEdges syncs FIXED_BY_BLOCK edges from code_block_incidents table
// Returns: (edges created, error)
func (s *IncidentEdgeSyncer) SyncFixedByBlockEdges(ctx context.Context, repoID int64) (int, error) {
	log.Printf("  🐛 Syncing FIXED_BY_BLOCK edges from PostgreSQL → Neo4j...")

	// Query code_block_incidents for issue→block relationships
	query := `
		SELECT
			cbi.issue_id,
			cbi.block_id,
			cbi.incident_date,
			cbi.incident_type,
			gi.number as issue_number
		FROM code_block_incidents cbi
		JOIN github_issues gi ON cbi.issue_id = gi.id
		WHERE cbi.repo_id = $1
		ORDER BY cbi.incident_date ASC`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	synced := 0
	for rows.Next() {
		var issueID, blockID int64
		var incidentDate time.Time
		var incidentType *string
		var issueNumber int

		if err := rows.Scan(&issueID, &blockID, &incidentDate, &incidentType, &issueNumber); err != nil {
			return synced, err
		}

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			// Create FIXED_BY_BLOCK edge from Issue to CodeBlock
			query := `
				MATCH (i:Issue {number: $issueNumber, repo_id: $repoID})
				MATCH (b:CodeBlock {db_id: $blockID, repo_id: $repoID})
				MERGE (i)-[r:FIXED_BY_BLOCK]->(b)
				SET r.incident_date = datetime($incidentDate),
				    r.incident_type = $incidentType
				RETURN count(r) as created`

			var incidentTypeStr interface{}
			if incidentType != nil {
				incidentTypeStr = *incidentType
			}

			params := map[string]any{
				"issueNumber":  issueNumber,
				"blockID":      blockID,
				"repoID":       repoID,
				"incidentDate": incidentDate.Format(time.RFC3339),
				"incidentType": incidentTypeStr,
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			log.Printf("     ⚠️  Failed to create FIXED_BY_BLOCK edge Issue#%d→Block %d: %v", issueNumber, blockID, err)
			continue
		}
		synced++

		if synced%100 == 0 {
			log.Printf("     → Synced %d FIXED_BY_BLOCK edges", synced)
		}
	}

	log.Printf("     ✓ Synced %d FIXED_BY_BLOCK edges", synced)
	return synced, rows.Err()
}
