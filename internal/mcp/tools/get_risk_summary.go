package tools

import (
	"context"
	"fmt"
)

// GraphClient interface for querying risk data
type GraphClient interface {
	GetCodeBlocksForFile(ctx context.Context, filePath string, historicalPaths []string, repoID int) ([]CodeBlock, error)
	GetCouplingData(ctx context.Context, blockID string) (*CouplingData, error)
	GetTemporalData(ctx context.Context, blockID string) (*TemporalData, error)
}

// IdentityResolver interface for resolving file renames
type IdentityResolver interface {
	ResolveHistoricalPaths(ctx context.Context, currentPath string) ([]string, error)
}

// GetRiskSummaryTool implements the crisk.get_risk_summary tool
type GetRiskSummaryTool struct {
	graphClient      GraphClient
	identityResolver IdentityResolver
}

// NewGetRiskSummaryTool creates a new GetRiskSummaryTool
func NewGetRiskSummaryTool(graphClient GraphClient, identityResolver IdentityResolver) *GetRiskSummaryTool {
	return &GetRiskSummaryTool{
		graphClient:      graphClient,
		identityResolver: identityResolver,
	}
}

// Execute executes the get_risk_summary tool
func (t *GetRiskSummaryTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// 1. Parse arguments
	filePath, ok := args["file_path"].(string)
	if !ok || filePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}

	// Optional diff_content for future use
	// diffContent, _ := args["diff_content"].(string)

	// Use repo_id from actual implementation
	repoID := 4 // mcp-use repo_id

	// 2. Resolve file identity (handle renames)
	historicalPaths, err := t.identityResolver.ResolveHistoricalPaths(ctx, filePath)
	if err != nil {
		// Non-fatal: continue with just current path
		historicalPaths = []string{}
	}

	// 3. Query Neo4j for CodeBlocks in this file
	// IMPORTANT: This now returns ownership data inline
	blocks, err := t.graphClient.GetCodeBlocksForFile(ctx, filePath, historicalPaths, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get code blocks: %w", err)
	}

	// Edge case: No blocks found
	if len(blocks) == 0 {
		return map[string]interface{}{
			"file_path":     filePath,
			"risk_evidence": []BlockEvidence{},
			"warning":       "No code blocks found for this file",
		}, nil
	}

	// 4. For each block, fetch coupling and temporal data
	var evidence []BlockEvidence
	for _, block := range blocks {
		// Ownership is already in block.OwnershipData

		// Query coupling (from Neo4j CO_CHANGES_WITH edges)
		coupling, err := t.graphClient.GetCouplingData(ctx, block.ID)
		if err != nil {
			// Non-fatal: block may have no coupling
			coupling = &CouplingData{CoupledWith: []CoupledBlock{}}
		}

		// Query temporal (from PostgreSQL code_block_incidents)
		temporal, err := t.graphClient.GetTemporalData(ctx, block.ID)
		if err != nil {
			// Non-fatal: block may have no incidents
			temporal = &TemporalData{IncidentCount: 0, Incidents: []TemporalIncident{}}
		}

		evidence = append(evidence, BlockEvidence{
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
			CoupledBlocks: coupling.CoupledWith,
			// Temporal (from PostgreSQL code_block_incidents)
			IncidentCount: temporal.IncidentCount,
			Incidents:     temporal.Incidents,
		})
	}

	// 5. Return structured response
	return map[string]interface{}{
		"file_path":     filePath,
		"total_blocks":  len(blocks),
		"risk_evidence": evidence,
	}, nil
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
