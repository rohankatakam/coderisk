package tools

import (
	"context"
	"fmt"
)

// GraphClient interface for querying risk data
type GraphClient interface {
	GetCodeBlocksForFile(ctx context.Context, filePath string, historicalPaths []string, repoID int) ([]CodeBlock, error)
	GetCodeBlocksByNames(ctx context.Context, filePathsWithHistorical map[string][]string, blockNames []string, repoID int) ([]CodeBlock, error)
	GetCouplingData(ctx context.Context, blockID string) (*CouplingData, error)
	GetTemporalData(ctx context.Context, blockID string) (*TemporalData, error)
}

// IdentityResolver interface for resolving file renames
type IdentityResolver interface {
	ResolveHistoricalPaths(ctx context.Context, currentPath string) ([]string, error)
	ResolveHistoricalPathsWithRoot(ctx context.Context, currentPath string, repoRoot string) ([]string, error)
}

// DiffAtomizer interface for extracting code blocks from diffs
type DiffAtomizer interface {
	ExtractBlocksFromDiff(ctx context.Context, diff string) ([]BlockReference, error)
}

// BlockReference represents a code block extracted from a diff
type BlockReference struct {
	FilePath  string
	BlockName string
	BlockType string
	Behavior  string
}

// GetRiskSummaryTool implements the crisk.get_risk_summary tool
type GetRiskSummaryTool struct {
	graphClient      GraphClient
	identityResolver IdentityResolver
	diffAtomizer     DiffAtomizer // Optional: for diff-based analysis
}

// NewGetRiskSummaryTool creates a new GetRiskSummaryTool
func NewGetRiskSummaryTool(graphClient GraphClient, identityResolver IdentityResolver, diffAtomizer DiffAtomizer) *GetRiskSummaryTool {
	return &GetRiskSummaryTool{
		graphClient:      graphClient,
		identityResolver: identityResolver,
		diffAtomizer:     diffAtomizer,
	}
}

// Execute executes the get_risk_summary tool
func (t *GetRiskSummaryTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// 1. Parse arguments
	filePath, _ := args["file_path"].(string)
	diffContent, _ := args["diff_content"].(string)
	repoRoot, _ := args["repo_root"].(string) // Optional: repository root for absolute path resolution

	// Either file_path OR diff_content must be provided
	if filePath == "" && diffContent == "" {
		return nil, fmt.Errorf("either file_path or diff_content is required")
	}

	// Parse limits with defaults
	maxCoupledBlocks := 1
	if val, ok := args["max_coupled_blocks"].(int); ok && val > 0 {
		maxCoupledBlocks = val
	}
	maxIncidents := 1
	if val, ok := args["max_incidents"].(int); ok && val > 0 {
		maxIncidents = val
	}
	maxBlocks := 10
	if val, ok := args["max_blocks"].(int); ok {
		maxBlocks = val // 0 means return all
	}

	// Parse filters
	var blockTypes map[string]bool
	if types, ok := args["block_types"].([]interface{}); ok && len(types) > 0 {
		blockTypes = make(map[string]bool)
		for _, t := range types {
			if typeStr, ok := t.(string); ok {
				blockTypes[typeStr] = true
			}
		}
	}

	summaryOnly, _ := args["summary_only"].(bool)
	minStaleness, _ := args["min_staleness"].(int)
	minIncidents, _ := args["min_incidents"].(int)
	minRiskScore, _ := args["min_risk_score"].(float64)
	includeRiskScore, _ := args["include_risk_score"].(bool)
	prioritizeRecent, _ := args["prioritize_recent"].(bool)

	// Use repo_id from actual implementation
	repoID := 4 // mcp-use repo_id

	var blocks []CodeBlock

	// 2. BRANCH: Diff-based analysis OR file-based analysis
	if diffContent != "" && t.diffAtomizer != nil {
		// === DIFF-BASED FLOW (NEW) ===
		// Extract code blocks from diff using LLM (meta-ingestion)
		blockRefs, err := t.diffAtomizer.ExtractBlocksFromDiff(ctx, diffContent)
		if err != nil {
			return nil, fmt.Errorf("failed to extract blocks from diff: %w", err)
		}

		// Build map of file paths to historical paths
		filePathsWithHistorical := make(map[string][]string)
		blockNamesSet := make(map[string]bool)

		for _, ref := range blockRefs {
			// Resolve historical paths for each file in the diff
			var historical []string
			var err error
			if repoRoot != "" {
				historical, err = t.identityResolver.ResolveHistoricalPathsWithRoot(ctx, ref.FilePath, repoRoot)
			} else {
				historical, err = t.identityResolver.ResolveHistoricalPaths(ctx, ref.FilePath)
			}
			if err != nil {
				// Non-fatal: use just current path
				historical = []string{}
			}
			allPaths := append([]string{ref.FilePath}, historical...)
			filePathsWithHistorical[ref.FilePath] = allPaths

			// Track unique block names
			blockNamesSet[ref.BlockName] = true
		}

		// Convert set to slice
		var blockNames []string
		for name := range blockNamesSet {
			blockNames = append(blockNames, name)
		}

		// Query graph for all extracted blocks
		blocks, err = t.graphClient.GetCodeBlocksByNames(ctx, filePathsWithHistorical, blockNames, repoID)
		if err != nil {
			return nil, fmt.Errorf("failed to get code blocks by names: %w", err)
		}
	} else {
		// === FILE-BASED FLOW (EXISTING) ===
		// Resolve file identity (handle renames)
		var historicalPaths []string
		var err error
		if repoRoot != "" {
			historicalPaths, err = t.identityResolver.ResolveHistoricalPathsWithRoot(ctx, filePath, repoRoot)
		} else {
			historicalPaths, err = t.identityResolver.ResolveHistoricalPaths(ctx, filePath)
		}
		if err != nil {
			// Non-fatal: continue with just current path
			historicalPaths = []string{}
		}

		// Query Neo4j for CodeBlocks in this file
		blocks, err = t.graphClient.GetCodeBlocksForFile(ctx, filePath, historicalPaths, repoID)
		if err != nil {
			return nil, fmt.Errorf("failed to get code blocks: %w", err)
		}
	}

	// Edge case: No blocks found
	if len(blocks) == 0 {
		return map[string]interface{}{
			"file_path":     filePath,
			"risk_evidence": []BlockEvidence{},
			"warning":       "No code blocks found for this file",
		}, nil
	}

	// 4. First, build all evidence with risk scores (before filtering/limiting)
	type BlockWithScore struct {
		evidence  BlockEvidence
		riskScore float64
	}

	var allBlocks []BlockWithScore
	var totalIncidentCount int
	var maxStalenessSeen int
	blockTypeStats := make(map[string]int)

	for _, block := range blocks {
		// Apply block type filter
		if blockTypes != nil && !blockTypes[block.Type] {
			continue
		}

		// Query temporal data first for filtering
		temporal, err := t.graphClient.GetTemporalData(ctx, block.ID)
		incidentCount := 0
		incidents := []TemporalIncident{}
		if err == nil && temporal != nil {
			incidentCount = temporal.IncidentCount
			if temporal.Incidents != nil {
				incidents = temporal.Incidents
			}
		}

		// Apply incident filter
		if incidentCount < minIncidents {
			continue
		}

		// Apply staleness filter
		if block.OwnershipData.StaleDays < minStaleness {
			continue
		}

		// Track stats for summary mode
		totalIncidentCount += incidentCount
		if block.OwnershipData.StaleDays > maxStalenessSeen {
			maxStalenessSeen = block.OwnershipData.StaleDays
		}
		blockTypeStats[block.Type]++

		// Query coupling (from Neo4j CO_CHANGES_WITH edges)
		coupling, err := t.graphClient.GetCouplingData(ctx, block.ID)
		coupledBlocks := []CoupledBlock{}
		couplingScore := 0.0
		if err == nil && coupling != nil && coupling.CoupledWith != nil {
			coupledBlocks = coupling.CoupledWith
			// Calculate coupling score (number of coupled blocks + avg coupling rate)
			for _, cb := range coupledBlocks {
				couplingScore += cb.Rate
			}
			if len(coupledBlocks) > 0 {
				couplingScore = couplingScore / float64(len(coupledBlocks)) * float64(len(coupledBlocks))
			}
		}

		// Calculate risk score (weighted combination of factors)
		// Higher score = higher risk
		riskScore := 0.0

		// Temporal risk: incidents are the strongest signal
		riskScore += float64(incidentCount) * 10.0

		// Coupling risk: highly coupled code is risky
		riskScore += couplingScore * 2.0

		// Staleness risk: old code might have knowledge issues
		// But cap it - very fresh code (0 days) isn't necessarily low risk
		stalenessScore := float64(block.OwnershipData.StaleDays) / 30.0
		if stalenessScore > 3.0 {
			stalenessScore = 3.0 // Cap at 90 days worth of risk
		}
		riskScore += stalenessScore

		// Block type risk: classes are more risky than individual methods
		if block.Type == "class" {
			riskScore += 2.0
		}

		// Recency boost: if prioritize_recent is enabled, boost score for recently changed code
		// This helps focus on active development areas where bugs are more likely to be introduced
		if prioritizeRecent && block.OwnershipData.StaleDays < 30 {
			// Inverse staleness boost: fresher code gets higher boost
			recencyBoost := (30.0 - float64(block.OwnershipData.StaleDays)) / 30.0 * 5.0
			riskScore += recencyBoost
		}

		// Apply risk score filter (after calculating the full score)
		if riskScore < minRiskScore {
			continue
		}

		// Limit coupled blocks and incidents for output
		if len(coupledBlocks) > maxCoupledBlocks {
			coupledBlocks = coupledBlocks[:maxCoupledBlocks]
		}
		if len(incidents) > maxIncidents {
			incidents = incidents[:maxIncidents]
		}

		blockEvidence := BlockEvidence{
			BlockName:          block.Name,
			BlockType:          block.Type,
			FilePath:           block.Path,
			// Ownership (from Neo4j CodeBlock node)
			OriginalAuthor:     block.OwnershipData.OriginalAuthor,
			LastModifier:       block.OwnershipData.LastModifier,
			StaleDays:          block.OwnershipData.StaleDays,
			FamiliarityMap:     block.OwnershipData.FamiliarityMap,
			SemanticImportance: block.OwnershipData.SemanticImportance,
			// Coupling (from Neo4j CO_CHANGES_WITH edges)
			CoupledBlocks: coupledBlocks,
			// Temporal (from PostgreSQL code_block_incidents)
			IncidentCount: incidentCount,
			Incidents:     incidents,
		}

		// Include risk score if requested (for debugging/verification)
		if includeRiskScore {
			blockEvidence.RiskScore = &riskScore
		}

		// Store block with its risk score
		if !summaryOnly {
			allBlocks = append(allBlocks, BlockWithScore{
				evidence:  blockEvidence,
				riskScore: riskScore,
			})
		}
	}

	// 5. Sort blocks by risk score (highest first) and apply limit
	var evidence []BlockEvidence
	if !summaryOnly && len(allBlocks) > 0 {
		// Sort by risk score descending (highest risk first)
		for i := 0; i < len(allBlocks); i++ {
			for j := i + 1; j < len(allBlocks); j++ {
				if allBlocks[j].riskScore > allBlocks[i].riskScore {
					allBlocks[i], allBlocks[j] = allBlocks[j], allBlocks[i]
				}
			}
		}

		// Extract top N blocks
		limit := maxBlocks
		if limit <= 0 || limit > len(allBlocks) {
			limit = len(allBlocks)
		}
		for i := 0; i < limit; i++ {
			evidence = append(evidence, allBlocks[i].evidence)
		}
	}

	// 6. Return structured response
	response := map[string]interface{}{
		"total_blocks":  len(blocks),
		"risk_evidence": evidence,
	}

	// Add file_path for file-based queries, or indicate diff-based analysis
	if diffContent != "" {
		response["analysis_type"] = "diff-based"
		response["diff_provided"] = true
	} else {
		response["file_path"] = filePath
		response["analysis_type"] = "file-based"
	}

	// Add summary stats if requested
	if summaryOnly {
		response["summary"] = map[string]interface{}{
			"total_blocks_analyzed":  len(blocks),
			"blocks_matching_filter": len(blockTypeStats),
			"total_incidents":        totalIncidentCount,
			"max_staleness_days":     maxStalenessSeen,
			"block_type_counts":      blockTypeStats,
		}
		// Clear evidence array in summary mode
		response["risk_evidence"] = []BlockEvidence{}
	}

	return response, nil
}

// GetSchema returns the JSON schema for the tool
func (t *GetRiskSummaryTool) GetSchema() map[string]interface{} {
	return map[string]interface{}{
		"description": "Get risk evidence for a file including ownership, coupling, and temporal incident data",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file to analyze",
				},
				"diff_content": map[string]interface{}{
					"type":        "string",
					"description": "Optional diff content for uncommitted changes",
				},
			},
			"required": []string{"file_path"},
		},
	}
}
