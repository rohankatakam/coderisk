package risk

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rohankatakam/coderisk/internal/graph"
	"github.com/rohankatakam/coderisk/internal/llm"
)

// CouplingCalculator analyzes co-change relationships and import dependencies for CodeBlocks
// Reference: AGENT-P3B Coupling Risk Calculator
// Implements R_coupling calculation: co-change relationships and import dependencies
// Following Postgres-First Write Protocol (microservice_arch.md Edge Case 4)
type CouplingCalculator struct {
	db       *sql.DB                // PostgreSQL connection (source of truth)
	neo4j    *graph.Neo4jBackend    // Neo4j connection (derived cache)
	driver   neo4j.DriverWithContext // Legacy Neo4j driver (for backward compatibility)
	database string
	llm      *llm.Client
	repoID   int64
}

// NewCouplingCalculator creates a new coupling calculator instance with PostgreSQL + Neo4j support
func NewCouplingCalculator(db *sql.DB, neo4jBackend *graph.Neo4jBackend, driver neo4j.DriverWithContext, database string, llmClient *llm.Client, repoID int64) *CouplingCalculator {
	return &CouplingCalculator{
		db:       db,
		neo4j:    neo4jBackend,
		driver:   driver,
		database: database,
		llm:      llmClient,
		repoID:   repoID,
	}
}

// CodeBlockInfo represents a code block with its metadata
type CodeBlockInfo struct {
	ID       int64
	FilePath string
	Name     string
	Type     string
	Language string
}

// CoChangePair represents two code blocks that frequently change together
type CoChangePair struct {
	BlockA       CodeBlockInfo
	BlockB       CodeBlockInfo
	CoChangeRate float64
	CoChangeCount int
	TotalChangesA int
	LastCoChange  time.Time
	CommitSHA     string
	Explanation   string // LLM-generated explanation
}

// CalculateCoChanges detects and creates co-change relationships between code blocks
// Reference: AGENT-P3B §1 - Co-Change Detection
// Only creates edges when co-change rate >= 0.5
func (c *CouplingCalculator) CalculateCoChanges(ctx context.Context) (int, error) {
	// Query to find commits that modified multiple blocks
	// We'll find all pairs of blocks modified by the same commit
	query := `
		WITH block_commits AS (
			SELECT
				cbc.block_id AS code_block_id,
				cbc.commit_sha,
				gc.author_date AS modified_at
			FROM code_block_changes cbc
			JOIN code_blocks cb ON cb.id = cbc.block_id
			JOIN github_commits gc ON gc.sha = cbc.commit_sha AND gc.repo_id = cbc.repo_id
			WHERE cb.repo_id = $1 AND cbc.repo_id = $1
		),
		co_change_pairs AS (
			SELECT
				bc1.code_block_id AS block_a_id,
				bc2.code_block_id AS block_b_id,
				COUNT(DISTINCT bc1.commit_sha) AS co_change_count,
				MAX(bc1.modified_at) AS last_co_changed_at,
				MAX(bc1.commit_sha) AS last_co_change_commit_sha
			FROM block_commits bc1
			JOIN block_commits bc2 ON bc1.commit_sha = bc2.commit_sha
			WHERE bc1.code_block_id < bc2.code_block_id  -- Prevent duplicates and self-pairs
			GROUP BY bc1.code_block_id, bc2.code_block_id
		),
		block_total_changes AS (
			SELECT
				cbc.block_id AS code_block_id,
				COUNT(*) AS total_changes
			FROM code_block_changes cbc
			JOIN code_blocks cb ON cb.id = cbc.block_id
			WHERE cb.repo_id = $1 AND cbc.repo_id = $1
			GROUP BY cbc.block_id
		)
		SELECT
			ccp.block_a_id,
			ccp.block_b_id,
			ccp.co_change_count,
			btc.total_changes AS total_a_changes,
			CAST(ccp.co_change_count AS DECIMAL) / btc.total_changes AS co_change_rate,
			ccp.last_co_changed_at,
			ccp.last_co_change_commit_sha
		FROM co_change_pairs ccp
		JOIN block_total_changes btc ON btc.code_block_id = ccp.block_a_id
		WHERE CAST(ccp.co_change_count AS DECIMAL) / btc.total_changes >= 0.5  -- Only significant coupling
		ORDER BY co_change_rate DESC
	`

	rows, err := c.db.QueryContext(ctx, query, c.repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to query co-changes: %w", err)
	}
	defer rows.Close()

	edgesCreated := 0
	var coChangePairs []CoChangePair

	// Collect all co-change pairs
	for rows.Next() {
		var blockAID, blockBID int64
		var coChangeCount, totalAChanges int
		var coChangeRate float64
		var lastCoChange time.Time
		var commitSHA string

		if err := rows.Scan(&blockAID, &blockBID, &coChangeCount, &totalAChanges, &coChangeRate, &lastCoChange, &commitSHA); err != nil {
			return edgesCreated, fmt.Errorf("failed to scan co-change row: %w", err)
		}

		// Get block info
		blockA, err := c.getBlockInfo(ctx, blockAID)
		if err != nil {
			return edgesCreated, fmt.Errorf("failed to get block A info: %w", err)
		}

		blockB, err := c.getBlockInfo(ctx, blockBID)
		if err != nil {
			return edgesCreated, fmt.Errorf("failed to get block B info: %w", err)
		}

		pair := CoChangePair{
			BlockA:        blockA,
			BlockB:        blockB,
			CoChangeRate:  coChangeRate,
			CoChangeCount: coChangeCount,
			TotalChangesA: totalAChanges,
			LastCoChange:  lastCoChange,
			CommitSHA:     commitSHA,
		}

		coChangePairs = append(coChangePairs, pair)

		// Insert into code_block_coupling table
		insertQuery := `
			INSERT INTO code_block_coupling (
				block_a_id, block_b_id,
				co_change_count, co_change_rate,
				last_co_changed_at,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
			ON CONFLICT (block_a_id, block_b_id)
			DO UPDATE SET
				co_change_count = EXCLUDED.co_change_count,
				co_change_rate = EXCLUDED.co_change_rate,
				last_co_changed_at = EXCLUDED.last_co_changed_at,
				updated_at = NOW()
		`

		_, err = c.db.ExecContext(ctx, insertQuery,
			blockAID, blockBID,
			coChangeCount, coChangeRate,
			lastCoChange,
		)
		if err != nil {
			return edgesCreated, fmt.Errorf("failed to insert co-change edge: %w", err)
		}

		edgesCreated++
	}

	if err := rows.Err(); err != nil {
		return edgesCreated, fmt.Errorf("error iterating co-change rows: %w", err)
	}

	log.Printf("    → Created %d co-change edges in PostgreSQL", edgesCreated)

	// STEP 2: Sync to Neo4j (derived cache)
	// Create CO_CHANGES_WITH edges in Neo4j from PostgreSQL coupling data
	if c.neo4j != nil && edgesCreated > 0 {
		log.Printf("    → Syncing %d CO_CHANGES_WITH edges to Neo4j...", edgesCreated)

		// Query PostgreSQL for all coupling edges for this repo
		syncQuery := `
			SELECT
				cbc.block_a_id,
				cbc.block_b_id,
				cbc.co_change_count,
				cbc.co_change_rate,
				cbc.last_co_changed_at
			FROM code_block_coupling cbc
			JOIN code_blocks cba ON cba.id = cbc.block_a_id
			WHERE cba.repo_id = $1
		`

		syncRows, err := c.db.QueryContext(ctx, syncQuery, c.repoID)
		if err != nil {
			log.Printf("    ⚠️  Warning: Failed to query coupling data for Neo4j sync: %v", err)
		} else {
			defer syncRows.Close()

			neo4jEdgesCreated := 0
			for syncRows.Next() {
				var blockAID, blockBID int64
				var coChangeCount int
				var coChangeRate float64
				var lastCoChange time.Time

				if err := syncRows.Scan(&blockAID, &blockBID, &coChangeCount, &coChangeRate, &lastCoChange); err != nil {
					log.Printf("    ⚠️  Warning: Failed to scan coupling row: %v", err)
					continue
				}

				// Create CO_CHANGES_WITH edge in Neo4j
				// Using db_id to link to CodeBlock nodes (which have db_id from PostgreSQL id)
				neo4jQuery := `
					MATCH (a:CodeBlock {db_id: $blockAID, repo_id: $repoID})
					MATCH (b:CodeBlock {db_id: $blockBID, repo_id: $repoID})
					MERGE (a)-[r:CO_CHANGES_WITH]-(b)
					SET r.co_change_count = $coChangeCount,
					    r.coupling_rate = $couplingRate,
					    r.first_co_change = datetime($lastCoChange),
					    r.last_co_change = datetime($lastCoChange)
				`

				params := map[string]interface{}{
					"blockAID":      blockAID,
					"blockBID":      blockBID,
					"repoID":        c.repoID,
					"coChangeCount": coChangeCount,
					"couplingRate":  coChangeRate,
					"lastCoChange":  lastCoChange.Format(time.RFC3339),
				}

				queries := []graph.QueryWithParams{{Query: neo4jQuery, Params: params}}
				if err := c.neo4j.ExecuteBatchWithParams(ctx, queries); err != nil {
					log.Printf("    ⚠️  Warning: Failed to create CO_CHANGES_WITH edge for blocks %d-%d: %v", blockAID, blockBID, err)
					continue
				}

				neo4jEdgesCreated++
			}

			if err := syncRows.Err(); err != nil {
				log.Printf("    ⚠️  Warning: Error iterating coupling rows for Neo4j sync: %v", err)
			} else {
				log.Printf("    ✓ Synced %d CO_CHANGES_WITH edges to Neo4j", neo4jEdgesCreated)
			}
		}
	}

	// Generate LLM explanations for top 10 coupled pairs
	if c.llm != nil && c.llm.IsEnabled() && len(coChangePairs) > 0 {
		topN := 10
		if len(coChangePairs) < topN {
			topN = len(coChangePairs)
		}

		// Rate limiting: Add 2-second delay between LLM calls
		// Gemini Tier 1: 2,000 RPM (~33 req/sec), 10 calls over 20 seconds = safe
		const delayBetweenCallsSeconds = 2

		for i := 0; i < topN; i++ {
			explanation, err := c.ExplainCoupling(ctx, coChangePairs[i])
			if err != nil {
				// Log error but don't fail - explanations are optional
				fmt.Printf("Warning: Failed to generate explanation for pair %d: %v\n", i, err)
				continue
			}
			fmt.Printf("Co-change pair #%d (rate: %.2f%%): %s\n", i+1, coChangePairs[i].CoChangeRate*100, explanation)

			// Rate limiting: Sleep between calls to avoid burst traffic
			if i < topN-1 {
				time.Sleep(time.Duration(delayBetweenCallsSeconds) * time.Second)
			}
		}
	}

	return edgesCreated, nil
}

// getBlockInfo retrieves metadata for a code block
func (c *CouplingCalculator) getBlockInfo(ctx context.Context, blockID int64) (CodeBlockInfo, error) {
	query := `
		SELECT id, file_path, block_name, block_type, language
		FROM code_blocks
		WHERE id = $1
	`

	var info CodeBlockInfo
	err := c.db.QueryRowContext(ctx, query, blockID).Scan(
		&info.ID,
		&info.FilePath,
		&info.Name,
		&info.Type,
		&info.Language,
	)
	if err != nil {
		return info, fmt.Errorf("failed to query block info: %w", err)
	}

	return info, nil
}

// ExplainCoupling generates an LLM explanation for why two blocks are coupled
// Reference: AGENT-P3B §3 - LLM Coupling Explanation
func (c *CouplingCalculator) ExplainCoupling(ctx context.Context, pair CoChangePair) (string, error) {
	if c.llm == nil || !c.llm.IsEnabled() {
		return "", fmt.Errorf("llm client not enabled")
	}

	systemPrompt := "You are a code analysis expert. Explain why two code blocks might be coupled based on their metadata."

	userPrompt := fmt.Sprintf(`These two code blocks changed together %d%% of the time (%d out of %d commits):

Block 1: %s (%s in %s)
Block 2: %s (%s in %s)

Explain WHY they might be coupled in 1-2 sentences. Consider:
- Are they in the same file or related files?
- Do their names suggest related functionality?
- What architectural relationship might cause them to change together?`,
		int(pair.CoChangeRate*100),
		pair.CoChangeCount,
		pair.TotalChangesA,
		pair.BlockA.Name, pair.BlockA.Type, pair.BlockA.FilePath,
		pair.BlockB.Name, pair.BlockB.Type, pair.BlockB.FilePath,
	)

	response, err := c.llm.Complete(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", fmt.Errorf("llm completion failed: %w", err)
	}

	return strings.TrimSpace(response), nil
}

// UpdateCouplingAggregates updates code_blocks table with aggregated coupling metrics
// Reference: DATA_SCHEMA_REFERENCE.md lines 266-267, 851-853
// Populates: co_change_count, avg_coupling_rate in code_blocks table
// Following Postgres-First Write Protocol
func (c *CouplingCalculator) UpdateCouplingAggregates(ctx context.Context) (int, error) {
	log.Printf("  🔍 Updating coupling aggregates for code blocks...")

	// STEP 1: Write to PostgreSQL (source of truth)
	// Calculate aggregated coupling metrics per block
	query := `
		WITH block_coupling_stats AS (
			SELECT
				cbc.block_a_id AS block_id,
				COUNT(*) AS coupling_count,
				AVG(cbc.co_change_rate) AS avg_rate
			FROM code_block_coupling cbc
			JOIN code_blocks cb ON cb.id = cbc.block_a_id
			WHERE cb.repo_id = $1
			GROUP BY cbc.block_a_id
			UNION ALL
			SELECT
				cbc.block_b_id AS block_id,
				COUNT(*) AS coupling_count,
				AVG(cbc.co_change_rate) AS avg_rate
			FROM code_block_coupling cbc
			JOIN code_blocks cb ON cb.id = cbc.block_b_id
			WHERE cb.repo_id = $1
			GROUP BY cbc.block_b_id
		),
		aggregated AS (
			SELECT
				block_id,
				SUM(coupling_count)::INTEGER AS total_couplings,
				AVG(avg_rate)::FLOAT AS avg_coupling_rate
			FROM block_coupling_stats
			GROUP BY block_id
		)
		UPDATE code_blocks cb
		SET
			co_change_count = a.total_couplings,
			avg_coupling_rate = a.avg_coupling_rate
		FROM aggregated a
		WHERE cb.id = a.block_id
		  AND cb.repo_id = $1
	`

	result, err := c.db.ExecContext(ctx, query, c.repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to update coupling aggregates in PostgreSQL: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("    → Updated %d blocks with coupling aggregates in PostgreSQL", rowsAffected)

	// STEP 2: Sync to Neo4j (derived cache)
	if c.neo4j != nil && rowsAffected > 0 {
		log.Printf("    → Syncing coupling aggregates to Neo4j...")

		// Sync using similar aggregation logic in Neo4j
		neo4jQuery := `
			MATCH (b:CodeBlock {repo_id: $repoID})-[r:CO_CHANGES_WITH]-(other:CodeBlock)
			WITH b, count(r) AS coupling_count, avg(r.coupling_rate) AS avg_rate
			SET b.co_change_count = coupling_count,
			    b.avg_coupling_rate = avg_rate
			RETURN count(b) AS updated_blocks
		`

		params := map[string]interface{}{"repoID": c.repoID}
		queries := []graph.QueryWithParams{{Query: neo4jQuery, Params: params}}
		if err := c.neo4j.ExecuteBatchWithParams(ctx, queries); err != nil {
			log.Printf("    ⚠️  Warning: Failed to sync coupling aggregates to Neo4j: %v", err)
		} else {
			log.Printf("    ✓ Synced coupling aggregates to Neo4j")
		}
	}

	return int(rowsAffected), nil
}

// CalculateRiskScores calculates final risk scores for all code blocks
// Reference: DATA_SCHEMA_REFERENCE.md lines 268, 977-998
// Formula: Temporal (40%) + Ownership (30%) + Coupling (30%)
// Following Postgres-First Write Protocol
func (c *CouplingCalculator) CalculateRiskScores(ctx context.Context) (int, error) {
	log.Printf("  🔍 Calculating final risk scores...")

	// STEP 1: Write to PostgreSQL (source of truth)
	// Calculate composite risk score using weighted formula
	query := `
		WITH risk_components AS (
			SELECT
				id,
				repo_id,
				-- Temporal component (40% weight)
				-- incident_count × recency_multiplier
				CASE
					WHEN last_incident_date IS NULL THEN 0
					WHEN last_incident_date > NOW() - INTERVAL '90 days' THEN incident_count * 2.0
					WHEN last_incident_date > NOW() - INTERVAL '180 days' THEN incident_count * 1.0
					ELSE incident_count * 0.5
				END AS temporal_score,

				-- Ownership component (30% weight)
				-- staleness_days / 365 × complexity_estimate
				COALESCE(staleness_days, 0)::FLOAT / 365.0 * COALESCE(complexity_estimate, 1) AS ownership_score,

				-- Coupling component (30% weight)
				-- co_change_count × avg_coupling_rate
				COALESCE(co_change_count, 0)::FLOAT * COALESCE(avg_coupling_rate, 0.0) AS coupling_score
			FROM code_blocks
			WHERE repo_id = $1
		),
		normalized_scores AS (
			SELECT
				id,
				-- Weighted combination
				(temporal_score * 0.4 + ownership_score * 0.3 + coupling_score * 0.3) AS raw_score,
				-- Calculate max for normalization
				MAX(temporal_score * 0.4 + ownership_score * 0.3 + coupling_score * 0.3) OVER () AS max_score
			FROM risk_components
		)
		UPDATE code_blocks cb
		SET risk_score = CASE
			WHEN ns.max_score > 0 THEN (ns.raw_score / ns.max_score * 100.0)::FLOAT
			ELSE 0.0
		END
		FROM normalized_scores ns
		WHERE cb.id = ns.id
		  AND cb.repo_id = $1
	`

	result, err := c.db.ExecContext(ctx, query, c.repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate risk scores in PostgreSQL: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("    → Calculated risk scores for %d blocks in PostgreSQL", rowsAffected)

	// STEP 2: Sync to Neo4j (derived cache)
	if c.neo4j != nil && rowsAffected > 0 {
		log.Printf("    → Syncing risk scores to Neo4j...")

		// Query PostgreSQL for all risk scores and sync to Neo4j
		syncQuery := `
			SELECT id, risk_score
			FROM code_blocks
			WHERE repo_id = $1 AND risk_score IS NOT NULL
		`

		rows, err := c.db.QueryContext(ctx, syncQuery, c.repoID)
		if err != nil {
			log.Printf("    ⚠️  Warning: Failed to query risk scores for Neo4j sync: %v", err)
		} else {
			defer rows.Close()

			syncedCount := 0
			for rows.Next() {
				var blockID int64
				var riskScore float64

				if err := rows.Scan(&blockID, &riskScore); err != nil {
					log.Printf("    ⚠️  Warning: Failed to scan risk score row: %v", err)
					continue
				}

				// Update Neo4j CodeBlock node with risk_score
				neo4jQuery := `
					MATCH (b:CodeBlock {db_id: $blockID, repo_id: $repoID})
					SET b.risk_score = $riskScore
				`

				params := map[string]interface{}{
					"blockID":   blockID,
					"repoID":    c.repoID,
					"riskScore": riskScore,
				}

				queries := []graph.QueryWithParams{{Query: neo4jQuery, Params: params}}
				if err := c.neo4j.ExecuteBatchWithParams(ctx, queries); err != nil {
					log.Printf("    ⚠️  Warning: Failed to update risk_score for block %d: %v", blockID, err)
					continue
				}

				syncedCount++
			}

			if err := rows.Err(); err != nil {
				log.Printf("    ⚠️  Warning: Error iterating risk score rows: %v", err)
			} else {
				log.Printf("    ✓ Synced %d risk scores to Neo4j", syncedCount)
			}
		}
	}

	return int(rowsAffected), nil
}

// CalculateDependencies maps file-level import dependencies to block-level dependencies
// Reference: AGENT-P3B §2 - Import Dependency Mapping
// Note: This is a simplified version that would need IMPORTS_FROM edges in the graph
func (c *CouplingCalculator) CalculateDependencies(ctx context.Context) (int, error) {
	// For now, this is a placeholder since we need to integrate with the graph database
	// to get IMPORTS_FROM edges from the file-level graph.
	// In a full implementation, we would:
	// 1. Query Neo4j for File-level IMPORTS_FROM edges
	// 2. Map them to code blocks using file paths
	// 3. Create block-level dependency relationships

	// TODO: Implement when Neo4j integration is ready
	// This requires:
	// - Neo4j driver connection
	// - Cypher query to find IMPORTS_FROM edges
	// - Mapping logic from files to blocks

	return 0, nil
}

// GetCoChangeStatistics returns summary statistics about co-change relationships
func (c *CouplingCalculator) GetCoChangeStatistics(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) AS total_edges,
			MIN(co_change_rate) AS min_rate,
			MAX(co_change_rate) AS max_rate,
			AVG(co_change_rate) AS avg_rate,
			COUNT(*) FILTER (WHERE co_change_rate >= 0.75) AS high_coupling_count,
			COUNT(*) FILTER (WHERE co_change_rate >= 0.5 AND co_change_rate < 0.75) AS medium_coupling_count
		FROM code_block_coupling cbc
		JOIN code_blocks cba ON cba.id = cbc.block_a_id
		WHERE cba.repo_id = $1
	`

	var totalEdges, highCoupling, mediumCoupling int
	var minRate, maxRate, avgRate sql.NullFloat64

	err := c.db.QueryRowContext(ctx, query, c.repoID).Scan(
		&totalEdges,
		&minRate,
		&maxRate,
		&avgRate,
		&highCoupling,
		&mediumCoupling,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query statistics: %w", err)
	}

	stats := map[string]interface{}{
		"total_edges":          totalEdges,
		"min_rate":             minRate.Float64,
		"max_rate":             maxRate.Float64,
		"avg_rate":             avgRate.Float64,
		"high_coupling_count":  highCoupling,  // rate >= 0.75
		"medium_coupling_count": mediumCoupling, // 0.5 <= rate < 0.75
	}

	return stats, nil
}

// GetTopCoupledBlocks returns the code blocks with the most co-change relationships
func (c *CouplingCalculator) GetTopCoupledBlocks(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	query := `
		WITH block_coupling_counts AS (
			SELECT
				cbc.block_a_id AS block_id,
				COUNT(*) AS coupling_count,
				AVG(cbc.co_change_rate) AS avg_rate
			FROM code_block_coupling cbc
			JOIN code_blocks cb ON cb.id = cbc.block_a_id
			WHERE cb.repo_id = $1
			GROUP BY cbc.block_a_id
			UNION ALL
			SELECT
				cbc.block_b_id AS block_id,
				COUNT(*) AS coupling_count,
				AVG(cbc.co_change_rate) AS avg_rate
			FROM code_block_coupling cbc
			JOIN code_blocks cb ON cb.id = cbc.block_b_id
			WHERE cb.repo_id = $1
			GROUP BY cbc.block_b_id
		),
		aggregated AS (
			SELECT
				block_id,
				SUM(coupling_count) AS total_couplings,
				AVG(avg_rate) AS avg_coupling_rate
			FROM block_coupling_counts
			GROUP BY block_id
		)
		SELECT
			cb.id,
			cb.file_path,
			cb.block_name,
			cb.block_type,
			a.total_couplings,
			a.avg_coupling_rate
		FROM aggregated a
		JOIN code_blocks cb ON cb.id = a.block_id
		ORDER BY a.total_couplings DESC, a.avg_coupling_rate DESC
		LIMIT $2
	`

	rows, err := c.db.QueryContext(ctx, query, c.repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top coupled blocks: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id int64
		var filePath, blockName, blockType string
		var totalCouplings int
		var avgRate float64

		if err := rows.Scan(&id, &filePath, &blockName, &blockType, &totalCouplings, &avgRate); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		results = append(results, map[string]interface{}{
			"id":               id,
			"file_path":        filePath,
			"block_name":       blockName,
			"block_type":       blockType,
			"total_couplings":  totalCouplings,
			"avg_coupling_rate": avgRate,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

// MarkCouplingIndexed updates the coupling_indexed_at timestamp for all blocks in a repo
// This enables tracking when coupling calculations were last performed
// Reference: Migration 009 - Idempotency tracking
func (c *CouplingCalculator) MarkCouplingIndexed(ctx context.Context) error {
	query := `
		UPDATE code_blocks
		SET coupling_indexed_at = NOW()
		WHERE repo_id = $1
	`

	result, err := c.db.ExecContext(ctx, query, c.repoID)
	if err != nil {
		return fmt.Errorf("failed to update coupling_indexed_at: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	log.Printf("    → Marked %d blocks as coupling-indexed", rows)
	return nil
}
