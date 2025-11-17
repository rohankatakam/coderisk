package risk

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rohankatakam/coderisk/internal/llm"
)

// OwnershipCalculator calculates ownership properties for CodeBlocks
// Reference: AGENT_P3A_OWNERSHIP.md - Ownership Risk Calculator
type OwnershipCalculator struct {
	driver   neo4j.DriverWithContext
	database string
	llm      *llm.Client
	logger   *slog.Logger
}

// NewOwnershipCalculator creates a new ownership calculator
func NewOwnershipCalculator(driver neo4j.DriverWithContext, database string, llmClient *llm.Client) *OwnershipCalculator {
	return &OwnershipCalculator{
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
func (o *OwnershipCalculator) CalculateOriginalAuthor(ctx context.Context, repoID int64) (*OwnershipResult, error) {
	start := time.Now()

	query := `
		MATCH (b:CodeBlock {repo_id: $repoID})<-[:CREATED_BLOCK]-(c:Commit)<-[:AUTHORED]-(d:Developer)
		SET b.original_author = d.email
		RETURN count(b) AS blocks_updated
	`

	session := o.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: o.database,
	})
	defer session.Close(ctx)

	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
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

		blocksUpdated, _ := record.Get("blocks_updated")
		return blocksUpdated, nil
	})

	if err != nil {
		return &OwnershipResult{
			BlocksUpdated: 0,
			Error:         fmt.Errorf("failed to calculate original author: %w", err),
			Duration:      time.Since(start),
			Phase:         "original_author",
		}, err
	}

	count := int(result.(int64))
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

// CalculateLastModifierAndStaleness sets last_modifier and staleness_days properties
// Reference: AGENT_P3A_OWNERSHIP.md - Last Modifier + Staleness query
// Edge case: Block with no modifications (only creation): last_modifier = original_author
func (o *OwnershipCalculator) CalculateLastModifierAndStaleness(ctx context.Context, repoID int64) (*OwnershipResult, error) {
	start := time.Now()

	// First, set last_modifier and staleness for blocks with modifications
	modifiedQuery := `
		MATCH (b:CodeBlock {repo_id: $repoID})<-[:MODIFIED_BLOCK]-(c:Commit)<-[:AUTHORED]-(d:Developer)
		WITH b, d, c ORDER BY c.timestamp DESC
		WITH b, head(collect(d)) AS lastDev, head(collect(c)) AS lastCommit
		SET b.last_modifier = lastDev.email,
		    b.staleness_days = duration.between(datetime({epochSeconds: lastCommit.committed_at}), datetime()).days
		RETURN count(b) AS blocks_updated
	`

	session := o.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: o.database,
	})
	defer session.Close(ctx)

	modifiedResult, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, modifiedQuery, map[string]any{
			"repoID": repoID,
		})
		if err != nil {
			return nil, err
		}

		record, err := res.Single(ctx)
		if err != nil {
			return nil, err
		}

		blocksUpdated, _ := record.Get("blocks_updated")
		return blocksUpdated, nil
	})

	if err != nil {
		return &OwnershipResult{
			BlocksUpdated: 0,
			Error:         fmt.Errorf("failed to calculate last modifier: %w", err),
			Duration:      time.Since(start),
			Phase:         "last_modifier_staleness",
		}, err
	}

	modifiedCount := int(modifiedResult.(int64))

	// Edge case: Blocks with no modifications - use original author as last modifier
	unmodifiedQuery := `
		MATCH (b:CodeBlock {repo_id: $repoID})<-[:CREATED_BLOCK]-(c:Commit)<-[:AUTHORED]-(d:Developer)
		WHERE NOT EXISTS((b)<-[:MODIFIED_BLOCK]-())
		WITH b, d, c
		SET b.last_modifier = d.email,
		    b.staleness_days = duration.between(datetime({epochSeconds: c.committed_at}), datetime()).days
		RETURN count(b) AS blocks_updated
	`

	unmodifiedResult, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, unmodifiedQuery, map[string]any{
			"repoID": repoID,
		})
		if err != nil {
			return nil, err
		}

		record, err := res.Single(ctx)
		if err != nil {
			return nil, err
		}

		blocksUpdated, _ := record.Get("blocks_updated")
		return blocksUpdated, nil
	})

	if err != nil {
		o.logger.Warn("failed to handle unmodified blocks", "error", err)
	}

	unmodifiedCount := 0
	if unmodifiedResult != nil {
		unmodifiedCount = int(unmodifiedResult.(int64))
	}

	totalCount := modifiedCount + unmodifiedCount

	o.logger.Info("calculated last modifier and staleness",
		"repo_id", repoID,
		"modified_blocks", modifiedCount,
		"unmodified_blocks", unmodifiedCount,
		"total_blocks", totalCount,
		"duration_ms", time.Since(start).Milliseconds())

	return &OwnershipResult{
		BlocksUpdated: totalCount,
		Error:         nil,
		Duration:      time.Since(start),
		Phase:         "last_modifier_staleness",
	}, nil
}

// CalculateFamiliarityMap builds a familiarity map showing which developers have edited each block
// Reference: AGENT_P3A_OWNERSHIP.md - Familiarity Map query
// Edge case: Block modified 100+ times - limit to top 10 contributors
func (o *OwnershipCalculator) CalculateFamiliarityMap(ctx context.Context, repoID int64) (*OwnershipResult, error) {
	start := time.Now()

	// Check if APOC is available
	hasAPOC, err := o.checkAPOCAvailable(ctx)
	if err != nil {
		o.logger.Warn("failed to check APOC availability", "error", err)
		hasAPOC = false
	}

	var query string
	if hasAPOC {
		// Use APOC for JSON conversion
		query = `
			MATCH (b:CodeBlock {repo_id: $repoID})<-[:CREATED_BLOCK|MODIFIED_BLOCK]-(c:Commit)<-[:AUTHORED]-(d:Developer)
			WITH b, d.email AS dev, count(c) AS edits
			ORDER BY edits DESC
			WITH b, collect({dev: dev, edits: edits})[0..10] AS famMap
			SET b.familiarity_map = apoc.convert.toJson(famMap)
			RETURN count(b) AS blocks_updated
		`
	} else {
		// Manual JSON construction (APOC not available)
		// We'll process blocks one by one using a different approach
		o.logger.Warn("APOC not available, using manual JSON construction")
		return o.calculateFamiliarityMapManual(ctx, repoID)
	}

	session := o.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: o.database,
	})
	defer session.Close(ctx)

	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
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

		blocksUpdated, _ := record.Get("blocks_updated")
		return blocksUpdated, nil
	})

	if err != nil {
		return &OwnershipResult{
			BlocksUpdated: 0,
			Error:         fmt.Errorf("failed to calculate familiarity map: %w", err),
			Duration:      time.Since(start),
			Phase:         "familiarity_map",
		}, err
	}

	count := int(result.(int64))
	o.logger.Info("calculated familiarity maps",
		"repo_id", repoID,
		"blocks_updated", count,
		"apoc_used", hasAPOC,
		"duration_ms", time.Since(start).Milliseconds())

	return &OwnershipResult{
		BlocksUpdated: count,
		Error:         nil,
		Duration:      time.Since(start),
		Phase:         "familiarity_map",
	}, nil
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
func (o *OwnershipCalculator) VerifyOwnershipProperties(ctx context.Context, repoID int64) (int, error) {
	query := `
		MATCH (b:CodeBlock {repo_id: $repoID})
		WHERE b.original_author IS NULL
		   OR b.last_modifier IS NULL
		   OR b.staleness_days IS NULL
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
