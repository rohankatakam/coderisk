package risk

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rohankatakam/coderisk/internal/graph"
	"github.com/rohankatakam/coderisk/internal/llm"
)

// OwnershipCalculator calculates ownership properties for CodeBlocks
// Reference: AGENT_P3A_OWNERSHIP.md - Ownership Risk Calculator
// Reference: DATA_SCHEMA_REFERENCE.md lines 262-265, 947-950 - PostgreSQL + Neo4j dual-write
type OwnershipCalculator struct {
	db       *sql.DB                // PostgreSQL connection (source of truth)
	neo4j    *graph.Neo4jBackend    // Neo4j connection (derived cache)
	driver   neo4j.DriverWithContext // Legacy Neo4j driver (for backward compatibility)
	database string
	llm      *llm.Client
	logger   *slog.Logger
}

// NewOwnershipCalculator creates a new ownership calculator with PostgreSQL + Neo4j support
// Following Postgres-First Write Protocol (microservice_arch.md Edge Case 4)
func NewOwnershipCalculator(db *sql.DB, neo4jBackend *graph.Neo4jBackend, driver neo4j.DriverWithContext, database string, llmClient *llm.Client) *OwnershipCalculator {
	return &OwnershipCalculator{
		db:       db,
		neo4j:    neo4jBackend,
		driver:   driver,
		database: database,
		llm:      llmClient,
		logger:   slog.Default().With("component", "ownership_calculator"),
	}
}

// CodeBlock represents a code block with basic properties
type CodeBlock struct {
	ID        string
	Name      string
	BlockType string
	FilePath  string
	RepoID    int64
}

// FamiliarityEntry represents a developer's familiarity with a code block
type FamiliarityEntry struct {
	Dev   string `json:"dev"`
	Edits int    `json:"edits"`
}

// OwnershipResult represents the result of ownership calculations
type OwnershipResult struct {
	BlocksUpdated  int
	Error          error
	Duration       time.Duration
	Phase          string
}

// CalculateOriginalAuthor sets the original_author property for all CodeBlocks
// Reference: AGENT_P3A_OWNERSHIP.md - Original Author query
// Reference: DATA_SCHEMA_REFERENCE.md line 262, 947 - Populate original_author_email in PostgreSQL + Neo4j
// Following Postgres-First Write Protocol (microservice_arch.md Edge Case 4)
func (o *OwnershipCalculator) CalculateOriginalAuthor(ctx context.Context, repoID int64) (*OwnershipResult, error) {
	start := time.Now()
	log.Printf("  🔍 Calculating original authors for code blocks...")

	// STEP 1: Write to PostgreSQL (source of truth)
	// Use GitHub commits to find original authors via first_seen_sha
	postgresQuery := `
		UPDATE code_blocks cb
		SET original_author_email = subq.original_author
		FROM (
			SELECT DISTINCT
				cb_inner.id,
				gc.author_email AS original_author
			FROM code_blocks cb_inner
			JOIN github_commits gc ON gc.sha = cb_inner.first_seen_sha
			WHERE cb_inner.repo_id = $1
			  AND gc.author_email IS NOT NULL
		) subq
		WHERE cb.id = subq.id
	`

	result, err := o.db.ExecContext(ctx, postgresQuery, repoID)
	if err != nil {
		return &OwnershipResult{
			BlocksUpdated: 0,
			Error:         fmt.Errorf("failed to update PostgreSQL original authors: %w", err),
			Duration:      time.Since(start),
			Phase:         "original_author",
		}, err
	}

	postgresCount, err := result.RowsAffected()
	if err != nil {
		return &OwnershipResult{
			BlocksUpdated: 0,
			Error:         fmt.Errorf("failed to get rows affected: %w", err),
			Duration:      time.Since(start),
			Phase:         "original_author",
		}, err
	}

	log.Printf("    → Updated %d blocks in PostgreSQL", postgresCount)

	// STEP 2: Sync to Neo4j (derived cache)
	// Note: CodeBlock nodes use 'original_author' property (not original_author_email for space)
	if o.neo4j != nil {
		neo4jQuery := `
			MATCH (b:CodeBlock {repo_id: $repoID})<-[:CREATED_BLOCK]-(c:Commit)<-[:AUTHORED]-(d:Developer)
			SET b.original_author_email = d.email
			RETURN count(b) AS blocks_updated
		`
		params := map[string]interface{}{
			"repoID": repoID,
		}
		queries := []graph.QueryWithParams{{Query: neo4jQuery, Params: params}}
		if err := o.neo4j.ExecuteBatchWithParams(ctx, queries); err != nil {
			log.Printf("    ⚠️  Warning: Failed to sync original authors to Neo4j: %v", err)
			// Continue - PostgreSQL is source of truth
		} else {
			log.Printf("    → Synced original authors to Neo4j")
		}
	}

	count := int(postgresCount)
	o.logger.Info("calculated original authors",
		"repo_id", repoID,
		"blocks_updated", count,
		"duration_ms", time.Since(start).Milliseconds())

	return &OwnershipResult{
		BlocksUpdated: count,
		Error:         nil,
		Duration:      time.Since(start),
		Phase:         "original_author",
	}, nil
}

// CalculateLastModifierAndStaleness sets last_modifier and last_modified_date properties
// Reference: AGENT_P3A_OWNERSHIP.md - Last Modifier + Staleness query
// Reference: DATA_SCHEMA_REFERENCE.md lines 304-306, 881-883 - Populate last_modifier_email and last_modified_date (STATIC)
// Reference: microservice_arch.md lines 105-147 - Static vs Dynamic Separation (staleness computed at query time)
// Edge case: Block with no modifications (only creation): last_modifier = original_author
// Following Postgres-First Write Protocol (microservice_arch.md Edge Case 4)
func (o *OwnershipCalculator) CalculateLastModifierAndStaleness(ctx context.Context, repoID int64) (*OwnershipResult, error) {
	start := time.Now()
	log.Printf("  🔍 Calculating last modifiers and last modified timestamps...")

	// STEP 1: Write to PostgreSQL (source of truth)
	// Find last modifier using most recent change from code_block_changes table
	// CRITICAL: Use author_date (always populated) NOT committer_date (may be NULL)
	// CRITICAL: Store last_modified_date (STATIC timestamp) NOT staleness_days (DYNAMIC property)
	query := `
		WITH last_changes AS (
			SELECT DISTINCT ON (cbc.block_id)
				cbc.block_id,
				c.author_email AS last_modifier,
				c.author_date AS last_modified_date
			FROM code_block_changes cbc
			JOIN code_blocks cb_inner ON cb_inner.id = cbc.block_id
			JOIN github_commits c ON c.sha = cbc.commit_sha AND c.repo_id = cbc.repo_id
			WHERE cb_inner.repo_id = $1
			ORDER BY cbc.block_id, c.author_date DESC
		)
		UPDATE code_blocks cb
		SET
			last_modifier_email = lc.last_modifier,
			last_modified_date = lc.last_modified_date
		FROM last_changes lc
		WHERE cb.id = lc.block_id
		  AND cb.repo_id = $1
	`

	result, err := o.db.ExecContext(ctx, query, repoID)
	if err != nil {
		return &OwnershipResult{
			BlocksUpdated: 0,
			Error:         fmt.Errorf("failed to update PostgreSQL last modifiers: %w", err),
			Duration:      time.Since(start),
			Phase:         "last_modifier_staleness",
		}, err
	}

	modifiedCount, _ := result.RowsAffected()

	// Edge case: Blocks with no changes in code_block_changes - use first_seen_sha as fallback
	// Use author_date from creation commit as last_modified_date
	unmodifiedQuery := `
		UPDATE code_blocks cb
		SET
			last_modifier_email = subq.last_modifier,
			last_modified_date = subq.last_modified_date
		FROM (
			SELECT
				cb_inner.id,
				gc.author_email AS last_modifier,
				gc.author_date AS last_modified_date
			FROM code_blocks cb_inner
			JOIN github_commits gc ON gc.sha = cb_inner.first_seen_sha
			WHERE cb_inner.repo_id = $1
			  AND cb_inner.last_modifier_email IS NULL
			  AND gc.author_email IS NOT NULL
		) subq
		WHERE cb.id = subq.id
	`

	unmodifiedResult, err := o.db.ExecContext(ctx, unmodifiedQuery, repoID)
	if err != nil {
		log.Printf("    ⚠️  Warning: Failed to update unmodified blocks: %v", err)
	}

	unmodifiedCount := int64(0)
	if unmodifiedResult != nil {
		unmodifiedCount, _ = unmodifiedResult.RowsAffected()
	}

	totalCount := modifiedCount + unmodifiedCount
	log.Printf("    → Updated %d blocks in PostgreSQL (%d modified, %d unmodified)", totalCount, modifiedCount, unmodifiedCount)

	// STEP 2: Sync to Neo4j (derived cache)
	// Store last_modified_date (STATIC timestamp) in Neo4j
	// Staleness is computed dynamically at query time using duration.between()
	if o.neo4j != nil {
		// Sync modified blocks - store last_modified_date as static property
		neo4jModifiedQuery := `
			MATCH (b:CodeBlock {repo_id: $repoID})<-[:MODIFIED_BLOCK]-(c:Commit)<-[:AUTHORED]-(d:Developer)
			WITH b, d, c ORDER BY c.author_date DESC
			WITH b, head(collect(d)) AS lastDev, head(collect(c)) AS lastCommit
			SET b.last_modifier_email = lastDev.email,
			    b.last_modified_date = lastCommit.author_date
			RETURN count(b) AS blocks_updated
		`
		params := map[string]interface{}{"repoID": repoID}
		queries := []graph.QueryWithParams{{Query: neo4jModifiedQuery, Params: params}}
		if err := o.neo4j.ExecuteBatchWithParams(ctx, queries); err != nil {
			log.Printf("    ⚠️  Warning: Failed to sync modified blocks to Neo4j: %v", err)
		}

		// Sync unmodified blocks - use creation commit's author_date
		neo4jUnmodifiedQuery := `
			MATCH (b:CodeBlock {repo_id: $repoID})<-[:CREATED_BLOCK]-(c:Commit)<-[:AUTHORED]-(d:Developer)
			WHERE NOT EXISTS((b)<-[:MODIFIED_BLOCK]-())
			SET b.last_modifier_email = d.email,
			    b.last_modified_date = c.author_date
			RETURN count(b) AS blocks_updated
		`
		queries = []graph.QueryWithParams{{Query: neo4jUnmodifiedQuery, Params: params}}
		if err := o.neo4j.ExecuteBatchWithParams(ctx, queries); err != nil {
			log.Printf("    ⚠️  Warning: Failed to sync unmodified blocks to Neo4j: %v", err)
		} else {
			log.Printf("    → Synced last modifiers and last_modified_date to Neo4j")
		}
	}

	count := int(totalCount)
	o.logger.Info("calculated last modifier and staleness",
		"repo_id", repoID,
		"modified_blocks", modifiedCount,
		"unmodified_blocks", unmodifiedCount,
		"total_blocks", totalCount,
		"duration_ms", time.Since(start).Milliseconds())

	return &OwnershipResult{
		BlocksUpdated: count,
		Error:         nil,
		Duration:      time.Since(start),
		Phase:         "last_modifier_staleness",
	}, nil
}

// CalculateFamiliarityMap builds a familiarity map showing which developers have edited each block
// Reference: AGENT_P3A_OWNERSHIP.md - Familiarity Map query
// Reference: DATA_SCHEMA_REFERENCE.md line 265, 950 - Populate familiarity_map JSONB
// Edge case: Block modified 100+ times - limit to top 10 contributors
// Following Postgres-First Write Protocol (microservice_arch.md Edge Case 4)
func (o *OwnershipCalculator) CalculateFamiliarityMap(ctx context.Context, repoID int64) (*OwnershipResult, error) {
	start := time.Now()
	log.Printf("  🔍 Calculating familiarity maps...")

	// STEP 1: Write to PostgreSQL (source of truth)
	// Build familiarity map as JSON in PostgreSQL using jsonb_agg
	postgresQuery := `
		WITH developer_edits AS (
			SELECT
				cbc.block_id AS block_id,
				c.author_email AS email,
				COUNT(DISTINCT cbc.commit_sha) AS edit_count
			FROM code_block_changes cbc
			JOIN code_blocks cb ON cb.id = cbc.block_id
			JOIN github_commits c ON c.sha = cbc.commit_sha AND c.repo_id = cbc.repo_id
			WHERE cb.repo_id = $1
			  AND c.author_email IS NOT NULL
			GROUP BY cbc.block_id, c.author_email
		),
		ranked_developers AS (
			SELECT
				block_id,
				email,
				edit_count,
				ROW_NUMBER() OVER (PARTITION BY block_id ORDER BY edit_count DESC) AS rank
			FROM developer_edits
		),
		familiarity_json AS (
			SELECT
				block_id,
				jsonb_agg(
					jsonb_build_object('dev', email, 'edits', edit_count)
					ORDER BY edit_count DESC
				) AS familiarity_map
			FROM ranked_developers
			WHERE rank <= 10
			GROUP BY block_id
		)
		UPDATE code_blocks cb
		SET familiarity_map = fj.familiarity_map
		FROM familiarity_json fj
		WHERE cb.id = fj.block_id
	`

	result, err := o.db.ExecContext(ctx, postgresQuery, repoID)
	if err != nil {
		return &OwnershipResult{
			BlocksUpdated: 0,
			Error:         fmt.Errorf("failed to update PostgreSQL familiarity maps: %w", err),
			Duration:      time.Since(start),
			Phase:         "familiarity_map",
		}, err
	}

	postgresCount, _ := result.RowsAffected()

	// Edge case: Blocks with no code_block_changes - create minimal familiarity map from first_seen_sha
	if postgresCount == 0 {
		fallbackQuery := `
			UPDATE code_blocks cb
			SET familiarity_map = subq.familiarity_map
			FROM (
				SELECT
					cb_inner.id,
					jsonb_build_array(
						jsonb_build_object('dev', gc.author_email, 'edits', 1)
					) AS familiarity_map
				FROM code_blocks cb_inner
				JOIN github_commits gc ON gc.sha = cb_inner.first_seen_sha
				WHERE cb_inner.repo_id = $1
				  AND cb_inner.familiarity_map IS NULL
				  AND gc.author_email IS NOT NULL
			) subq
			WHERE cb.id = subq.id
		`
		fallbackResult, err := o.db.ExecContext(ctx, fallbackQuery, repoID)
		if err != nil {
			log.Printf("    ⚠️  Warning: Failed to create fallback familiarity maps: %v", err)
		} else {
			fallbackCount, _ := fallbackResult.RowsAffected()
			postgresCount = fallbackCount
			log.Printf("    → Created %d fallback familiarity maps from first_seen_sha", fallbackCount)
		}
	}

	log.Printf("    → Updated %d blocks in PostgreSQL", postgresCount)

	// STEP 2: Sync to Neo4j (derived cache)
	if o.neo4j != nil {
		// Check if APOC is available for Neo4j JSON conversion
		hasAPOC, _ := o.checkAPOCAvailable(ctx)

		if hasAPOC {
			// Use APOC for efficient JSON conversion in Neo4j
			neo4jQuery := `
				MATCH (b:CodeBlock {repo_id: $repoID})<-[:CREATED_BLOCK|MODIFIED_BLOCK]-(c:Commit)<-[:AUTHORED]-(d:Developer)
				WITH b, d.email AS dev, count(c) AS edits
				ORDER BY edits DESC
				WITH b, collect({dev: dev, edits: edits})[0..10] AS famMap
				SET b.familiarity_map = apoc.convert.toJson(famMap)
				RETURN count(b) AS blocks_updated
			`
			params := map[string]interface{}{"repoID": repoID}
			queries := []graph.QueryWithParams{{Query: neo4jQuery, Params: params}}
			if err := o.neo4j.ExecuteBatchWithParams(ctx, queries); err != nil {
				log.Printf("    ⚠️  Warning: Failed to sync familiarity maps to Neo4j: %v", err)
			} else {
				log.Printf("    → Synced familiarity maps to Neo4j (APOC)")
			}
		} else {
			// Manual sync without APOC (slower but works)
			log.Printf("    ℹ️  APOC not available, using manual Neo4j sync...")
			if err := o.syncFamiliarityMapToNeo4jManual(ctx, repoID); err != nil {
				log.Printf("    ⚠️  Warning: Failed to manually sync to Neo4j: %v", err)
			} else {
				log.Printf("    → Synced familiarity maps to Neo4j (manual)")
			}
		}
	}

	count := int(postgresCount)
	o.logger.Info("calculated familiarity maps",
		"repo_id", repoID,
		"blocks_updated", count,
		"duration_ms", time.Since(start).Milliseconds())

	return &OwnershipResult{
		BlocksUpdated: count,
		Error:         nil,
		Duration:      time.Since(start),
		Phase:         "familiarity_map",
	}, nil
}

// syncFamiliarityMapToNeo4jManual syncs familiarity maps from PostgreSQL to Neo4j without APOC
func (o *OwnershipCalculator) syncFamiliarityMapToNeo4jManual(ctx context.Context, repoID int64) error {
	// Query PostgreSQL for all blocks with familiarity maps
	query := `
		SELECT id, familiarity_map
		FROM code_blocks
		WHERE repo_id = $1 AND familiarity_map IS NOT NULL
	`

	rows, err := o.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return fmt.Errorf("failed to query PostgreSQL: %w", err)
	}
	defer rows.Close()

	updatedCount := 0
	for rows.Next() {
		var blockID int64
		var famMapJSON []byte

		if err := rows.Scan(&blockID, &famMapJSON); err != nil {
			log.Printf("      Warning: Failed to scan row: %v", err)
			continue
		}

		// Update Neo4j with the JSON string
		neo4jQuery := `
			MATCH (b:CodeBlock {db_id: $blockID, repo_id: $repoID})
			SET b.familiarity_map = $famJson
		`
		params := map[string]interface{}{
			"blockID": blockID,
			"repoID":  repoID,
			"famJson": string(famMapJSON),
		}
		queries := []graph.QueryWithParams{{Query: neo4jQuery, Params: params}}
		if err := o.neo4j.ExecuteBatchWithParams(ctx, queries); err != nil {
			log.Printf("      Warning: Failed to update block %d: %v", blockID, err)
			continue
		}
		updatedCount++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	log.Printf("      Synced %d familiarity maps to Neo4j", updatedCount)
	return nil
}

// calculateFamiliarityMapManual manually builds JSON when APOC is not available
// Edge case: APOC not available - build JSON manually with string concatenation
func (o *OwnershipCalculator) calculateFamiliarityMapManual(ctx context.Context, repoID int64) (*OwnershipResult, error) {
	start := time.Now()

	// First, get all blocks
	blocksQuery := `
		MATCH (b:CodeBlock {repo_id: $repoID})
		RETURN b.db_id AS block_id
	`

	session := o.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: o.database,
	})
	defer session.Close(ctx)

	blockIDs := []int64{}
	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, blocksQuery, map[string]any{
			"repoID": repoID,
		})
		if err != nil {
			return nil, err
		}

		records, err := res.Collect(ctx)
		if err != nil {
			return nil, err
		}

		for _, record := range records {
			blockID, _ := record.Get("block_id")
			blockIDs = append(blockIDs, blockID.(int64))
		}

		return nil, nil
	})

	if err != nil {
		return &OwnershipResult{
			BlocksUpdated: 0,
			Error:         fmt.Errorf("failed to get block IDs: %w", err),
			Duration:      time.Since(start),
			Phase:         "familiarity_map_manual",
		}, err
	}

	// Process each block
	updatedCount := 0
	for _, blockID := range blockIDs {
		// Get developer edits for this block
		editsQuery := `
			MATCH (b:CodeBlock {db_id: $blockID, repo_id: $repoID})<-[:CREATED_BLOCK|MODIFIED_BLOCK]-(c:Commit)<-[:AUTHORED]-(d:Developer)
			WITH d.email AS dev, count(c) AS edits
			ORDER BY edits DESC
			LIMIT 10
			RETURN collect({dev: dev, edits: edits}) AS famMap
		`

		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			res, err := tx.Run(ctx, editsQuery, map[string]any{
				"blockID": blockID,
				"repoID":  repoID,
			})
			if err != nil {
				return nil, err
			}

			record, err := res.Single(ctx)
			if err != nil {
				return nil, err
			}

			famMapRaw, _ := record.Get("famMap")
			famMapList := famMapRaw.([]any)

			// Convert to JSON manually
			entries := []FamiliarityEntry{}
			for _, item := range famMapList {
				itemMap := item.(map[string]any)
				entries = append(entries, FamiliarityEntry{
					Dev:   itemMap["dev"].(string),
					Edits: int(itemMap["edits"].(int64)),
				})
			}

			jsonBytes, err := json.Marshal(entries)
			if err != nil {
				return nil, err
			}

			// Update the block with JSON string
			updateQuery := `
				MATCH (b:CodeBlock {db_id: $blockID, repo_id: $repoID})
				SET b.familiarity_map = $famJson
			`

			_, err = tx.Run(ctx, updateQuery, map[string]any{
				"blockID": blockID,
				"repoID":  repoID,
				"famJson": string(jsonBytes),
			})

			return nil, err
		})

		if err != nil {
			o.logger.Warn("failed to update familiarity map for block", "block_id", blockID, "error", err)
		} else {
			updatedCount++
		}
	}

	o.logger.Info("calculated familiarity maps manually",
		"repo_id", repoID,
		"blocks_updated", updatedCount,
		"total_blocks", len(blockIDs),
		"duration_ms", time.Since(start).Milliseconds())

	return &OwnershipResult{
		BlocksUpdated: updatedCount,
		Error:         nil,
		Duration:      time.Since(start),
		Phase:         "familiarity_map_manual",
	}, nil
}

// CalculateSemanticImportance uses LLM to classify code block importance
// Reference: AGENT_P3A_OWNERSHIP.md - LLM Semantic Importance
func (o *OwnershipCalculator) CalculateSemanticImportance(ctx context.Context, block CodeBlock) (string, error) {
	if o.llm == nil || !o.llm.IsEnabled() {
		o.logger.Warn("LLM not enabled, skipping semantic importance")
		return "P2", nil // Default to medium priority
	}

	systemPrompt := "You are a code risk analyzer. Classify the importance of code blocks based on their name, type, and file path."

	userPrompt := fmt.Sprintf(`Classify the importance of this code block:

Name: %s
Type: %s
File: %s

Options:
- P0 (Critical): Core business logic, auth, payments, security-critical code
- P1 (High): Important features, data processing, API endpoints
- P2 (Medium): UI components, helpers, utilities, test code

Return ONLY: P0, P1, or P2`, block.Name, block.BlockType, block.FilePath)

	response, err := o.llm.Complete(ctx, systemPrompt, userPrompt)
	if err != nil {
		o.logger.Warn("LLM completion failed", "error", err, "block", block.Name)
		return "P2", err // Default to lowest priority on error
	}

	// Parse response
	importance := strings.TrimSpace(response)
	importance = strings.ToUpper(importance)

	if importance != "P0" && importance != "P1" && importance != "P2" {
		o.logger.Warn("invalid importance response", "response", importance, "block", block.Name)
		return "P2", fmt.Errorf("invalid importance: %s", importance)
	}

	return importance, nil
}

// checkAPOCAvailable checks if APOC plugin is available in Neo4j
func (o *OwnershipCalculator) checkAPOCAvailable(ctx context.Context) (bool, error) {
	session := o.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: o.database,
	})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, "RETURN apoc.version() AS version", nil)
		if err != nil {
			return false, err
		}

		_, err = res.Single(ctx)
		if err != nil {
			return false, err
		}

		return true, nil
	})

	if err != nil {
		return false, nil // APOC not available
	}

	return result.(bool), nil
}

// VerifyOwnershipProperties checks that all CodeBlocks have ownership properties set
// Reference: AGENT_P3A_OWNERSHIP.md - Verification query
// Note: Validates STATIC properties only (staleness_days is computed dynamically)
func (o *OwnershipCalculator) VerifyOwnershipProperties(ctx context.Context, repoID int64) (int, error) {
	query := `
		MATCH (b:CodeBlock {repo_id: $repoID})
		WHERE b.original_author IS NULL
		   OR b.last_modifier IS NULL
		   OR b.last_modified_date IS NULL
		RETURN count(b) AS incomplete_blocks
	`

	session := o.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: o.database,
	})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, query, map[string]any{
			"repoID": repoID,
		})
		if err != nil {
			return nil, err
		}

		record, err := res.Single(ctx)
		if err != nil {
			return nil, err
		}

		incompleteBlocks, _ := record.Get("incomplete_blocks")
		return incompleteBlocks, nil
	})

	if err != nil {
		return -1, fmt.Errorf("verification query failed: %w", err)
	}

	incompleteCount := int(result.(int64))

	if incompleteCount == 0 {
		o.logger.Info("ownership verification passed", "repo_id", repoID)
	} else {
		o.logger.Warn("ownership verification found incomplete blocks",
			"repo_id", repoID,
			"incomplete_blocks", incompleteCount)
	}

	return incompleteCount, nil
}

// MarkOwnershipIndexed updates the ownership_indexed_at timestamp for all blocks in a repo
// This enables tracking when ownership calculations were last performed
// Reference: Migration 009 - Idempotency tracking
func (o *OwnershipCalculator) MarkOwnershipIndexed(ctx context.Context, repoID int64) error {
	query := `
		UPDATE code_blocks
		SET ownership_indexed_at = NOW()
		WHERE repo_id = $1
	`

	result, err := o.db.ExecContext(ctx, query, repoID)
	if err != nil {
		return fmt.Errorf("failed to update ownership_indexed_at: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	log.Printf("    → Marked %d blocks as ownership-indexed", rows)
	return nil
}
