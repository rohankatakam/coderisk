package risk

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rohankatakam/coderisk/internal/graph"
	"github.com/rohankatakam/coderisk/internal/llm"
)

// BlockIndexer orchestrates ownership property calculations for CodeBlocks
// Reference: AGENT_P3A_OWNERSHIP.md - Main indexer orchestrator
// Reference: DATA_SCHEMA_REFERENCE.md lines 947-950 - PostgreSQL + Neo4j dual-write
type BlockIndexer struct {
	calculator *OwnershipCalculator
	logger     *slog.Logger
}

// NewBlockIndexer creates a new block indexer with PostgreSQL + Neo4j support
func NewBlockIndexer(db *sql.DB, neo4jBackend *graph.Neo4jBackend, driver neo4j.DriverWithContext, database string, llmClient *llm.Client) *BlockIndexer {
	return &BlockIndexer{
		calculator: NewOwnershipCalculator(db, neo4jBackend, driver, database, llmClient),
		logger:     slog.Default().With("component", "block_indexer"),
	}
}

// IndexResult represents the complete result of indexing ownership properties
type IndexResult struct {
	RepoID              int64
	TotalBlocks         int
	OriginalAuthors     int
	LastModifiers       int
	FamiliarityMaps     int
	IncompleteBlocks    int
	TotalDuration       time.Duration
	PhaseResults        []OwnershipResult
	Success             bool
	Error               error
}

// IndexOwnership runs all ownership calculations for a repository
// Reference: AGENT_P3A_OWNERSHIP.md - Complete workflow
func (idx *BlockIndexer) IndexOwnership(ctx context.Context, repoID int64) (*IndexResult, error) {
	start := time.Now()

	idx.logger.Info("starting ownership indexing", "repo_id", repoID)

	result := &IndexResult{
		RepoID:       repoID,
		PhaseResults: []OwnershipResult{},
	}

	// Phase 1: Calculate Original Authors
	idx.logger.Info("phase 1: calculating original authors", "repo_id", repoID)
	originalAuthorResult, err := idx.calculator.CalculateOriginalAuthor(ctx, repoID)
	if err != nil {
		idx.logger.Error("failed to calculate original authors", "error", err)
		result.Error = err
		result.Success = false
		result.TotalDuration = time.Since(start)
		return result, err
	}
	result.PhaseResults = append(result.PhaseResults, *originalAuthorResult)
	result.OriginalAuthors = originalAuthorResult.BlocksUpdated

	// Phase 2: Calculate Last Modifier and Staleness
	idx.logger.Info("phase 2: calculating last modifier and staleness", "repo_id", repoID)
	lastModifierResult, err := idx.calculator.CalculateLastModifierAndStaleness(ctx, repoID)
	if err != nil {
		idx.logger.Error("failed to calculate last modifier and staleness", "error", err)
		result.Error = err
		result.Success = false
		result.TotalDuration = time.Since(start)
		return result, err
	}
	result.PhaseResults = append(result.PhaseResults, *lastModifierResult)
	result.LastModifiers = lastModifierResult.BlocksUpdated

	// Phase 3: Calculate Familiarity Maps
	idx.logger.Info("phase 3: calculating familiarity maps", "repo_id", repoID)
	familiarityResult, err := idx.calculator.CalculateFamiliarityMap(ctx, repoID)
	if err != nil {
		idx.logger.Error("failed to calculate familiarity maps", "error", err)
		result.Error = err
		result.Success = false
		result.TotalDuration = time.Since(start)
		return result, err
	}
	result.PhaseResults = append(result.PhaseResults, *familiarityResult)
	result.FamiliarityMaps = familiarityResult.BlocksUpdated

	// Verification: Check all blocks have ownership properties
	idx.logger.Info("verification: checking ownership properties", "repo_id", repoID)
	incompleteBlocks, err := idx.calculator.VerifyOwnershipProperties(ctx, repoID)
	if err != nil {
		idx.logger.Warn("verification check failed", "error", err)
		// Don't fail the entire indexing for verification issues
	} else {
		result.IncompleteBlocks = incompleteBlocks
	}

	result.TotalDuration = time.Since(start)
	result.Success = true

	idx.logger.Info("ownership indexing complete",
		"repo_id", repoID,
		"original_authors", result.OriginalAuthors,
		"last_modifiers", result.LastModifiers,
		"familiarity_maps", result.FamiliarityMaps,
		"incomplete_blocks", result.IncompleteBlocks,
		"duration_ms", result.TotalDuration.Milliseconds(),
		"success", result.Success)

	return result, nil
}

// IndexOwnershipWithSemanticImportance runs ownership calculations with LLM-based importance classification
// This is a more expensive operation that should be run sparingly
// Reference: AGENT_P3A_OWNERSHIP.md - LLM Semantic Importance
func (idx *BlockIndexer) IndexOwnershipWithSemanticImportance(ctx context.Context, repoID int64) (*IndexResult, error) {
	// First run standard ownership indexing
	result, err := idx.IndexOwnership(ctx, repoID)
	if err != nil {
		return result, err
	}

	// Then calculate semantic importance for each block
	idx.logger.Info("calculating semantic importance with LLM", "repo_id", repoID)

	// This would require fetching all blocks and processing them
	// For now, we'll skip this as it's expensive and optional
	idx.logger.Warn("semantic importance calculation not yet implemented - skipping")

	return result, nil
}

// IndexOwnershipIncremental updates ownership properties for specific blocks
// Useful for incremental updates after new commits are processed
func (idx *BlockIndexer) IndexOwnershipIncremental(ctx context.Context, repoID int64, blockIDs []int64) error {
	idx.logger.Info("starting incremental ownership indexing",
		"repo_id", repoID,
		"block_count", len(blockIDs))

	// For incremental updates, we just re-run the full calculation
	// The queries are idempotent and will update only the blocks that need it
	_, err := idx.IndexOwnership(ctx, repoID)
	if err != nil {
		return fmt.Errorf("incremental indexing failed: %w", err)
	}

	return nil
}

// GetIndexingStats returns statistics about the current indexing state
// Following Postgres-First Write Protocol: Query PostgreSQL (source of truth)
func (idx *BlockIndexer) GetIndexingStats(ctx context.Context, repoID int64) (map[string]int, error) {
	stats := make(map[string]int)

	// Query PostgreSQL for ownership stats (source of truth)
	query := `
		SELECT
			COUNT(*) as total_blocks,
			COUNT(original_author_email) as with_original_author,
			COUNT(last_modifier_email) as with_last_modifier,
			COUNT(familiarity_map) as with_familiarity_map,
			COUNT(*) - COUNT(original_author_email) as missing_original_author,
			COUNT(*) - COUNT(last_modifier_email) as missing_last_modifier,
			COUNT(*) - COUNT(familiarity_map) as missing_familiarity_map
		FROM code_blocks
		WHERE repo_id = $1
	`

	row := idx.calculator.db.QueryRowContext(ctx, query, repoID)

	var totalBlocks, withOriginalAuthor, withLastModifier, withFamiliarityMap int
	var missingOriginalAuthor, missingLastModifier, missingFamiliarityMap int

	err := row.Scan(
		&totalBlocks,
		&withOriginalAuthor,
		&withLastModifier,
		&withFamiliarityMap,
		&missingOriginalAuthor,
		&missingLastModifier,
		&missingFamiliarityMap,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query PostgreSQL stats: %w", err)
	}

	stats["total_blocks"] = totalBlocks
	stats["with_original_author"] = withOriginalAuthor
	stats["with_last_modifier"] = withLastModifier
	stats["with_familiarity_map"] = withFamiliarityMap
	stats["missing_original_author"] = missingOriginalAuthor
	stats["missing_last_modifier"] = missingLastModifier
	stats["missing_familiarity_map"] = missingFamiliarityMap

	return stats, nil
}

// countBlocksWithProperty counts blocks that have a specific property set
func (idx *BlockIndexer) countBlocksWithProperty(ctx context.Context, repoID int64, propertyName string) (int, error) {
	var query string
	if propertyName == "" {
		query = `
			MATCH (b:CodeBlock {repo_id: $repoID})
			RETURN count(b) AS count
		`
	} else {
		query = fmt.Sprintf(`
			MATCH (b:CodeBlock {repo_id: $repoID})
			WHERE b.%s IS NOT NULL
			RETURN count(b) AS count
		`, propertyName)
	}

	session := idx.calculator.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: idx.calculator.database,
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

		count, _ := record.Get("count")
		return count, nil
	})

	if err != nil {
		return 0, err
	}

	return int(result.(int64)), nil
}

// MarkOwnershipIndexed delegates to the underlying calculator
// Updates ownership_indexed_at timestamp for idempotency tracking
func (idx *BlockIndexer) MarkOwnershipIndexed(ctx context.Context, repoID int64) error {
	return idx.calculator.MarkOwnershipIndexed(ctx, repoID)
}
