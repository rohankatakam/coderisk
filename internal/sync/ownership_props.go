package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// OwnershipPropSyncer syncs ownership-related properties on CodeBlock nodes
// Reference: DATA_SCHEMA_REFERENCE.md line 611-615 - ownership properties
type OwnershipPropSyncer struct {
	db       *sql.DB
	driver   neo4j.DriverWithContext
	database string
}

// NewOwnershipPropSyncer creates a new ownership property syncer
func NewOwnershipPropSyncer(db *sql.DB, driver neo4j.DriverWithContext, database string) *OwnershipPropSyncer {
	return &OwnershipPropSyncer{
		db:       db,
		driver:   driver,
		database: database,
	}
}

// SyncOwnershipProperties syncs ownership properties to CodeBlock nodes
// Properties: original_author_email, last_modifier_email, last_modified_date, familiarity_map
// Returns: (blocks updated, error)
func (s *OwnershipPropSyncer) SyncOwnershipProperties(ctx context.Context, repoID int64) (int, error) {
	log.Printf("  👤 Syncing ownership properties to CodeBlock nodes...")

	// Query code_blocks for ownership fields
	query := `
		SELECT
			id,
			original_author_email,
			last_modifier_email,
			last_modified_date,
			familiarity_map
		FROM code_blocks
		WHERE repo_id = $1
		  AND (original_author_email IS NOT NULL OR last_modifier_email IS NOT NULL)
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
		var originalAuthor, lastModifier *string
		var lastModifiedDate *time.Time
		var familiarityMapJSON []byte

		if err := rows.Scan(&blockID, &originalAuthor, &lastModifier, &lastModifiedDate, &familiarityMapJSON); err != nil {
			return synced, err
		}

		// Convert familiarity_map JSONB to JSON string for Neo4j storage
		// Neo4j doesn't support map properties - must store as JSON string
		var familiarityMapStr *string
		if len(familiarityMapJSON) > 0 {
			// Validate it's valid JSON (could be array or object)
			var testJSON interface{}
			if err := json.Unmarshal(familiarityMapJSON, &testJSON); err != nil {
				log.Printf("     ⚠️  Failed to parse familiarity_map for block %d: %v", blockID, err)
			} else {
				// Store as JSON string
				jsonStr := string(familiarityMapJSON)
				familiarityMapStr = &jsonStr
			}
		}

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			// Update CodeBlock properties
			// Note: familiarity_map stored as JSON string since Neo4j doesn't support maps
			query := `
				MATCH (b:CodeBlock {db_id: $blockID, repo_id: $repoID})
				SET b.original_author_email = $originalAuthor,
				    b.last_modifier_email = $lastModifier,
				    b.last_modified_date = CASE WHEN $lastModifiedDate IS NOT NULL THEN datetime($lastModifiedDate) ELSE NULL END,
				    b.familiarity_map_json = $familiarityMapJSON
				RETURN count(b) as updated`

			var lastModifiedDateStr interface{}
			if lastModifiedDate != nil {
				lastModifiedDateStr = lastModifiedDate.Format(time.RFC3339)
			}

			params := map[string]any{
				"blockID":            blockID,
				"repoID":             repoID,
				"originalAuthor":     originalAuthor,
				"lastModifier":       lastModifier,
				"lastModifiedDate":   lastModifiedDateStr,
				"familiarityMapJSON": familiarityMapStr,
			}

			result, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, err
			}
			return result.Collect(ctx)
		})

		if err != nil {
			log.Printf("     ⚠️  Failed to update ownership properties for block %d: %v", blockID, err)
			continue
		}
		synced++

		if synced%100 == 0 {
			log.Printf("     → Updated %d blocks", synced)
		}
	}

	log.Printf("     ✓ Updated ownership properties on %d CodeBlock nodes", synced)
	return synced, rows.Err()
}
