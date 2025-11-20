package sync

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// CouplingEdgeSyncer syncs CO_CHANGES_WITH edges from PostgreSQL to Neo4j
// Reference: DATA_SCHEMA_REFERENCE.md line 781-797 - CO_CHANGES_WITH edge spec
type CouplingEdgeSyncer struct {
	db       *sql.DB
	driver   neo4j.DriverWithContext
	database string
}

// NewCouplingEdgeSyncer creates a new coupling edge syncer
func NewCouplingEdgeSyncer(db *sql.DB, driver neo4j.DriverWithContext, database string) *CouplingEdgeSyncer {
	return &CouplingEdgeSyncer{
		db:       db,
		driver:   driver,
		database: database,
	}
}

// SyncCoChangesWithEdges syncs CO_CHANGES_WITH edges from code_block_coupling table
// Returns: (edges created, error)
func (s *CouplingEdgeSyncer) SyncCoChangesWithEdges(ctx context.Context, repoID int64) (int, error) {
	log.Printf("  🔗 Syncing CO_CHANGES_WITH edges from PostgreSQL → Neo4j...")

	// Query code_block_coupling for co-change relationships
	query := `
		SELECT
			block_a_id,
			block_b_id,
			co_change_count,
			co_change_percentage,
			computed_at,
			window_start,
			window_end
		FROM code_block_coupling
		WHERE repo_id = $1
		ORDER BY co_change_percentage DESC`

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: s.database})
	defer session.Close(ctx)

	synced := 0
	for rows.Next() {
		var blockAID, blockBID int64
		var coChangeCount int
		var coChangePercentage float64
		var computedAt time.Time
		var windowStart, windowEnd *time.Time

		if err := rows.Scan(&blockAID, &blockBID, &coChangeCount, &coChangePercentage, &computedAt, &windowStart, &windowEnd); err != nil {
			return synced, err
		}

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			// Create bidirectional CO_CHANGES_WITH edge
			// Note: Creating as single edge but logically bidirectional
			query := `
				MATCH (a:CodeBlock {db_id: $blockAID, repo_id: $repoID})
				MATCH (b:CodeBlock {db_id: $blockBID, repo_id: $repoID})
				MERGE (a)-[r:CO_CHANGES_WITH]-(b)
				SET r.co_change_count = $coChangeCount,
				    r.coupling_rate = $coChangePercentage,
				    r.computed_at = datetime($computedAt),
				    r.first_co_change = CASE WHEN $windowStart IS NOT NULL THEN datetime($windowStart) ELSE NULL END,
				    r.last_co_change = CASE WHEN $windowEnd IS NOT NULL THEN datetime($windowEnd) ELSE NULL END
				RETURN count(r) as created`

			var windowStartStr, windowEndStr interface{}
			if windowStart != nil {
				windowStartStr = windowStart.Format(time.RFC3339)
			}
			if windowEnd != nil {
				windowEndStr = windowEnd.Format(time.RFC3339)
			}

			params := map[string]any{
				"blockAID":           blockAID,
				"blockBID":           blockBID,
				"repoID":             repoID,
				"coChangeCount":      coChangeCount,
				"coChangePercentage": coChangePercentage,
				"computedAt":         computedAt.Format(time.RFC3339),
				"windowStart":        windowStartStr,
				"windowEnd":          windowEndStr,
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			log.Printf("     ⚠️  Failed to create CO_CHANGES_WITH edge %d↔%d: %v", blockAID, blockBID, err)
			continue
		}
		synced++

		if synced%100 == 0 {
			log.Printf("     → Synced %d CO_CHANGES_WITH edges", synced)
		}
	}

	log.Printf("     ✓ Synced %d CO_CHANGES_WITH edges", synced)
	return synced, rows.Err()
}
