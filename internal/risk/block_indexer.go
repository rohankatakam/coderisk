package risk

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rohankatakam/coderisk/internal/llm"
)

// BlockIndexer orchestrates ownership property calculations for CodeBlocks
// Reference: AGENT_P3A_OWNERSHIP.md - Main indexer orchestrator
type BlockIndexer struct {
	calculator *OwnershipCalculator
	logger     *slog.Logger
}

// NewBlockIndexer creates a new block indexer
func NewBlockIndexer(driver neo4j.DriverWithContext, database string, llmClient *llm.Client) *BlockIndexer {
	return &BlockIndexer{
		calculator: NewOwnershipCalculator(driver, database, llmClient),
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
func (idx *BlockIndexer) GetIndexingStats(ctx context.Context, repoID int64) (map[string]int, error) {
	stats := make(map[string]int)

	// Count total blocks
	totalBlocks, err := idx.countBlocksWithProperty(ctx, repoID, "")
	if err != nil {
		return nil, err
	}
	stats["total_blocks"] = totalBlocks

	// Count blocks with original_author
	withOriginalAuthor, err := idx.countBlocksWithProperty(ctx, repoID, "original_author")
	if err != nil {
		return nil, err
	}
	stats["with_original_author"] = withOriginalAuthor

	// Count blocks with last_modifier
	withLastModifier, err := idx.countBlocksWithProperty(ctx, repoID, "last_modifier")
	if err != nil {
		return nil, err
	}
	stats["with_last_modifier"] = withLastModifier

	// Count blocks with familiarity_map
	withFamiliarityMap, err := idx.countBlocksWithProperty(ctx, repoID, "familiarity_map")
	if err != nil {
		return nil, err
	}
	stats["with_familiarity_map"] = withFamiliarityMap

	// Calculate missing
	stats["missing_original_author"] = totalBlocks - withOriginalAuthor
	stats["missing_last_modifier"] = totalBlocks - withLastModifier
	stats["missing_familiarity_map"] = totalBlocks - withFamiliarityMap

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
