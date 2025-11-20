package diffanalyzer

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rohankatakam/coderisk/internal/llm"
	"github.com/rohankatakam/coderisk/internal/mcp"
	"github.com/rohankatakam/coderisk/internal/mcp/tools"
)

// Analyzer orchestrates the diff analysis pipeline
type Analyzer struct {
	neo4jDriver     neo4j.DriverWithContext
	db              *sql.DB
	atomizer        *mcp.DiffAtomizer
	fileResolver    *FileResolver
	graphMatcher    *GraphMatcher
	neo4jAggregator *Neo4jRiskAggregator
	logger          *log.Logger
}

// NewAnalyzer creates a new diff analyzer
func NewAnalyzer(neo4jDriver neo4j.DriverWithContext, db *sql.DB, llmClient *llm.Client, logger *log.Logger) *Analyzer {
	return &Analyzer{
		neo4jDriver:     neo4jDriver,
		db:              db,
		atomizer:        mcp.NewDiffAtomizer(llmClient),
		fileResolver:    NewFileResolver(db, logger),
		graphMatcher:    NewGraphMatcher(neo4jDriver, logger),
		neo4jAggregator: NewNeo4jRiskAggregator(neo4jDriver, logger),
		logger:          logger,
	}
}

// Analyze performs complete diff analysis pipeline
func (a *Analyzer) Analyze(ctx context.Context, repoID int64, diff string) (*RiskEvidenceJSON, error) {
	a.logger.Printf("=== Starting Diff Analysis for repo_id=%d ===", repoID)
	a.logger.Printf("Diff size: %d bytes", len(diff))

	// STEP 1: Extract block references from diff (LLM)
	a.logger.Println("[STEP 1] Extracting blocks from diff using LLM...")
	blockRefs, err := a.atomizer.ExtractBlocksFromDiff(ctx, diff)
	if err != nil {
		a.logger.Printf("[STEP 1] ERROR: Block extraction failed: %v", err)
		return nil, fmt.Errorf("block extraction failed: %w", err)
	}
	a.logger.Printf("[STEP 1] SUCCESS: Extracted %d blocks from diff", len(blockRefs))

	if len(blockRefs) == 0 {
		a.logger.Println("[STEP 1] No blocks found in diff, returning empty result")
		return &RiskEvidenceJSON{
			RiskSummary: "LOW",
			Blocks:      []BlockRisk{},
		}, nil
	}

	// STEP 2: Resolve file paths to canonical paths (PostgreSQL)
	a.logger.Println("[STEP 2] Resolving file paths to canonical paths...")
	filePaths := extractFilePaths(blockRefs)
	a.logger.Printf("[STEP 2] Found %d unique files in diff", len(filePaths))

	canonicalPaths, err := a.fileResolver.BatchResolve(ctx, repoID, filePaths)
	if err != nil {
		a.logger.Printf("[STEP 2] ERROR: File resolution failed: %v", err)
		return nil, fmt.Errorf("file resolution failed: %w", err)
	}
	a.logger.Printf("[STEP 2] SUCCESS: Resolved %d file paths", len(canonicalPaths))

	// STEP 3: Match blocks to Neo4j graph (3-tier fuzzy matching)
	a.logger.Println("[STEP 3] Matching blocks to Neo4j graph...")
	var blockMatches []BlockMatch

	for i, ref := range blockRefs {
		canonicalPath, ok := canonicalPaths[ref.FilePath]
		if !ok {
			canonicalPath = ref.FilePath // Fallback to original path
			a.logger.Printf("[STEP 3] WARN: No canonical path for %s, using original", ref.FilePath)
		}

		a.logger.Printf("[STEP 3] Matching block %d/%d: %s::%s", i+1, len(blockRefs), canonicalPath, ref.BlockName)

		match, err := a.graphMatcher.MatchBlock(ctx, repoID, canonicalPath, ref.BlockName, ref.Signature)
		if err != nil {
			a.logger.Printf("[STEP 3] ERROR: Graph match failed for %s::%s: %v", canonicalPath, ref.BlockName, err)
			continue
		}

		blockMatches = append(blockMatches, BlockMatch{
			DiffBlock:     convertToBlockRef(ref),
			GraphBlock:    match,
			CanonicalPath: canonicalPath,
		})
	}
	a.logger.Printf("[STEP 3] SUCCESS: Matched %d/%d blocks to graph", len(blockMatches), len(blockRefs))

	// STEP 4: Query risk data from Neo4j (parallel)
	a.logger.Println("[STEP 4] Querying risk data from Neo4j...")
	riskData := make([]BlockRiskData, len(blockMatches))
	var wg sync.WaitGroup
	var mutex sync.Mutex
	errorCount := 0

	for i, match := range blockMatches {
		if match.GraphBlock.MatchType == "new_function" {
			// Skip risk queries for new functions
			a.logger.Printf("[STEP 4] Skipping risk query for new function: %s", match.DiffBlock.BlockName)
			riskData[i] = BlockRiskData{Status: "new", Confidence: "low"}
			continue
		}

		wg.Add(1)
		go func(idx int, blockID int64, blockName string) {
			defer wg.Done()

			a.logger.Printf("[STEP 4] Querying risk for block_id=%d (%s)", blockID, blockName)

			// Query all 4 dimensions in parallel
			var innerWg sync.WaitGroup
			var temporal *TemporalRisk
			var ownership *OwnershipRisk
			var coupling *CouplingRisk
			var history *ChangeHistory

			innerWg.Add(4)

			go func() {
				defer innerWg.Done()
				var err error
				temporal, err = a.neo4jAggregator.QueryTemporal(ctx, blockID)
				if err != nil {
					a.logger.Printf("[STEP 4] WARN: Temporal query failed for block_id=%d: %v", blockID, err)
					mutex.Lock()
					errorCount++
					mutex.Unlock()
				}
			}()

			go func() {
				defer innerWg.Done()
				var err error
				ownership, err = a.neo4jAggregator.QueryOwnership(ctx, blockID)
				if err != nil {
					a.logger.Printf("[STEP 4] WARN: Ownership query failed for block_id=%d: %v", blockID, err)
					mutex.Lock()
					errorCount++
					mutex.Unlock()
				}
			}()

			go func() {
				defer innerWg.Done()
				var err error
				coupling, err = a.neo4jAggregator.QueryCoupling(ctx, blockID)
				if err != nil {
					a.logger.Printf("[STEP 4] WARN: Coupling query failed for block_id=%d: %v", blockID, err)
					mutex.Lock()
					errorCount++
					mutex.Unlock()
				}
			}()

			go func() {
				defer innerWg.Done()
				var err error
				history, err = a.neo4jAggregator.QueryHistory(ctx, blockID)
				if err != nil {
					a.logger.Printf("[STEP 4] WARN: History query failed for block_id=%d: %v", blockID, err)
					mutex.Lock()
					errorCount++
					mutex.Unlock()
				}
			}()

			innerWg.Wait()

			riskData[idx] = BlockRiskData{
				Temporal:      temporal,
				Ownership:     ownership,
				Coupling:      coupling,
				ChangeHistory: history,
			}

			a.logger.Printf("[STEP 4] Completed risk queries for block_id=%d", blockID)
		}(i, match.GraphBlock.ID, match.DiffBlock.BlockName)
	}

	wg.Wait()
	a.logger.Printf("[STEP 4] SUCCESS: Completed risk queries (%d errors)", errorCount)

	// STEP 5: Build RiskEvidenceJSON
	a.logger.Println("[STEP 5] Building final risk evidence JSON...")
	evidence := a.buildEvidence(blockMatches, riskData)
	a.logger.Printf("[STEP 5] SUCCESS: Risk summary = %s, %d blocks analyzed", evidence.RiskSummary, len(evidence.Blocks))

	a.logger.Println("=== Diff Analysis Complete ===")
	return evidence, nil
}

// extractFilePaths extracts unique file paths from block references
func extractFilePaths(blockRefs []tools.BlockReference) []string {
	pathMap := make(map[string]bool)
	var paths []string

	for _, ref := range blockRefs {
		if !pathMap[ref.FilePath] {
			paths = append(paths, ref.FilePath)
			pathMap[ref.FilePath] = true
		}
	}

	return paths
}

// convertToBlockRef converts tools.BlockReference to DiffBlockRef
func convertToBlockRef(ref tools.BlockReference) DiffBlockRef {
	return DiffBlockRef{
		FilePath:  ref.FilePath,
		BlockName: ref.BlockName,
		BlockType: ref.BlockType,
		Behavior:  ref.Behavior,
		Signature: ref.Signature,
	}
}

// buildEvidence constructs the final RiskEvidenceJSON from matched blocks and risk data
func (a *Analyzer) buildEvidence(blockMatches []BlockMatch, riskData []BlockRiskData) *RiskEvidenceJSON {
	var blocks []BlockRisk
	criticalCount := 0
	highCount := 0
	mediumCount := 0

	for i, match := range blockMatches {
		risk := riskData[i]

		blockRisk := BlockRisk{
			Name:       match.DiffBlock.BlockName,
			File:       match.CanonicalPath,
			ChangeType: match.DiffBlock.Behavior,
			MatchType:  match.GraphBlock.MatchType,
			Risks: RiskDimensions{
				Temporal:      risk.Temporal,
				Ownership:     risk.Ownership,
				Coupling:      risk.Coupling,
				ChangeHistory: risk.ChangeHistory,
			},
		}

		// Calculate risk level
		riskLevel := calculateRiskLevel(risk)
		switch riskLevel {
		case "CRITICAL":
			criticalCount++
		case "HIGH":
			highCount++
		case "MEDIUM":
			mediumCount++
		}

		blocks = append(blocks, blockRisk)
	}

	// Determine overall risk summary
	summary := "LOW"
	if criticalCount > 0 {
		summary = "CRITICAL"
	} else if highCount > 0 {
		summary = "HIGH"
	} else if mediumCount > 0 {
		summary = "MEDIUM"
	}

	a.logger.Printf("[buildEvidence] Risk distribution: CRITICAL=%d, HIGH=%d, MEDIUM=%d, summary=%s",
		criticalCount, highCount, mediumCount, summary)

	return &RiskEvidenceJSON{
		RiskSummary: summary,
		Blocks:      blocks,
	}
}

// calculateRiskLevel determines risk level based on risk data
func calculateRiskLevel(risk BlockRiskData) string {
	if risk.Status == "new" {
		return "LOW"
	}

	score := 0.0

	// Temporal scoring (40%)
	if risk.Temporal != nil {
		if risk.Temporal.IncidentCount >= 5 {
			score += 40
		} else if risk.Temporal.IncidentCount >= 3 {
			score += 30
		} else if risk.Temporal.IncidentCount >= 1 {
			score += 15
		}
	}

	// Ownership scoring (30%)
	if risk.Ownership != nil {
		if risk.Ownership.Status == "STALE" {
			score += 20
		}
		if risk.Ownership.BusFactorWarning {
			score += 10
		}
	}

	// Coupling scoring (30%)
	if risk.Coupling != nil {
		if risk.Coupling.Score > 10 {
			score += 30
		} else if risk.Coupling.Score > 5 {
			score += 20
		} else if risk.Coupling.Score > 0 {
			score += 10
		}
	}

	// Classify risk level
	if score >= 70 {
		return "CRITICAL"
	} else if score >= 50 {
		return "HIGH"
	} else if score >= 30 {
		return "MEDIUM"
	}
	return "LOW"
}
