package sync

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// IncidentPropSyncer syncs incident-related properties on CodeBlock nodes
// Reference: DATA_SCHEMA_REFERENCE.md line 609-610 - incident_count, temporal_summary properties
type IncidentPropSyncer struct {
	db       *sql.DB
	driver   neo4j.DriverWithContext
	database string
}

// NewIncidentPropSyncer creates a new incident property syncer
func NewIncidentPropSyncer(db *sql.DB, driver neo4j.DriverWithContext, database string) *IncidentPropSyncer {
	return &IncidentPropSyncer{
		db:       db,
		driver:   driver,
		database: database,
	}
}

// SyncIncidentProperties syncs incident_count, last_incident_date, and temporal_summary to CodeBlock nodes
// Returns: (blocks updated, error)
func (s *IncidentPropSyncer) SyncIncidentProperties(ctx context.Context, repoID int64) (int, error) {
	log.Printf("  📊 Syncing incident properties to CodeBlock nodes...")

	// Query code_blocks for incident-related fields
	query := `
		SELECT id, incident_count, last_incident_date, temporal_summary
		FROM code_blocks
		WHERE repo_id = $1
		  AND (incident_count > 0 OR temporal_summary IS NOT NULL)
		ORDER BY id`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	synced := 0
	for rows.Next() {
		var blockID int64
		var incidentCount int
		var lastIncidentDate *time.Time
		var temporalSummary *string

		if err := rows.Scan(&blockID, &incidentCount, &lastIncidentDate, &temporalSummary); err != nil {
			return synced, err
		}

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			// Update CodeBlock properties
			query := `
				MATCH (b:CodeBlock {db_id: $blockID, repo_id: $repoID})
				SET b.incident_count = $incidentCount,
				    b.last_incident_date = CASE WHEN $lastIncidentDate IS NOT NULL THEN datetime($lastIncidentDate) ELSE NULL END,
				    b.temporal_summary = $temporalSummary
				RETURN count(b) as updated`

			var lastIncidentDateStr interface{}
			if lastIncidentDate != nil {
				lastIncidentDateStr = lastIncidentDate.Format(time.RFC3339)
			}

			params := map[string]any{
				"blockID":          blockID,
				"repoID":           repoID,
				"incidentCount":    incidentCount,
				"lastIncidentDate": lastIncidentDateStr,
				"temporalSummary":  temporalSummary,
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			log.Printf("     ⚠️  Failed to update incident properties for block %d: %v", blockID, err)
			continue
		}
		synced++

		if synced%100 == 0 {
			log.Printf("     → Updated %d blocks", synced)
		}
	}

	log.Printf("     ✓ Updated incident properties on %d CodeBlock nodes", synced)
	return synced, rows.Err()
}
