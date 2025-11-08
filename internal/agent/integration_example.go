package agent

import (
	"context"
	"fmt"

	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/git"
	"github.com/rohankatakam/coderisk/internal/graph"
	"github.com/rohankatakam/coderisk/internal/metrics"
)

// IntegrationExample shows how to use KickoffPromptBuilder + RiskInvestigator
// This is the complete flow from file changes to risk assessment
//
// Usage in check.go:
//
//	assessment, err := agent.InvestigateFileChanges(
//	    ctx,
//	    files,
//	    resolver,
//	    neo4jClient,
//	    pgClient,
//	    llmClient,
//	    repoID,
//	    riskConfig,
//	)
func InvestigateFileChanges(
	ctx context.Context,
	files []string,
	resolver *git.FileResolver,
	graphClient *graph.Client,
	pgClient PostgresQueryExecutor,
	llmClient *LLMClient,
	repoID string,
	riskConfig config.AdaptiveRiskConfig,
) (*RiskAssessment, error) {

	// Step 1: Resolve files to historical paths
	resolvedFilesMap, err := resolver.BatchResolve(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("file resolution failed: %w", err)
	}

	// Step 2: Build FileChangeContext for each file
	var fileChanges []FileChangeContext
	for _, file := range files {
		matches := resolvedFilesMap[file]

		// Get git information
		diff, _ := git.GetFileDiff(file)
		linesAdded, linesDeleted := git.CountDiffLines(diff)
		changeStatus, _ := git.DetectChangeStatus(file)

		// Determine query path (use first match or current path)
		queryPath := file
		if len(matches) > 0 {
			queryPath = matches[0].HistoricalPath
		}

		// Run Phase 1 for baseline metrics
		adaptiveResult, err := metrics.CalculatePhase1WithConfig(ctx, graphClient, repoID, queryPath, riskConfig)
		var phase1Result *metrics.Phase1Result
		if err != nil {
			// Continue with zero metrics if Phase 1 fails
			phase1Result = &metrics.Phase1Result{}
		} else {
			phase1Result = adaptiveResult.Phase1Result
		}

		// Build file context
		fileContext := FromPhase1Result(
			file,
			changeStatus,
			linesAdded,
			linesDeleted,
			matches,
			git.TruncateDiffForPrompt(diff, 500),
			phase1Result,
		)

		fileChanges = append(fileChanges, fileContext)
	}

	// Step 3: Build kickoff prompt
	promptBuilder := NewKickoffPromptBuilder(fileChanges)
	kickoffPrompt := promptBuilder.BuildKickoffPrompt()

	// Step 4: Create investigator and run investigation
	// Note: This example doesn't set up hybrid client - would need database.NewHybridClient in real use
	investigator := NewRiskInvestigator(llmClient, graphClient, pgClient, nil)
	assessment, err := investigator.Investigate(ctx, kickoffPrompt)
	if err != nil {
		return nil, fmt.Errorf("investigation failed: %w", err)
	}

	return assessment, nil
}

// InvestigateSingleFile is a convenience function for checking a single file
func InvestigateSingleFile(
	ctx context.Context,
	filePath string,
	resolver *git.FileResolver,
	graphClient *graph.Client,
	pgClient PostgresQueryExecutor,
	llmClient *LLMClient,
	repoID string,
	riskConfig config.AdaptiveRiskConfig,
) (*RiskAssessment, error) {
	return InvestigateFileChanges(
		ctx,
		[]string{filePath},
		resolver,
		graphClient,
		pgClient,
		llmClient,
		repoID,
		riskConfig,
	)
}
