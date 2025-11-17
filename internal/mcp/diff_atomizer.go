package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/rohankatakam/coderisk/internal/atomizer"
	"github.com/rohankatakam/coderisk/internal/llm"
	"github.com/rohankatakam/coderisk/internal/mcp/tools"
)

// DiffAtomizer extracts code block references from git diffs using LLM
// This is a thin wrapper around the existing atomizer.Extractor
type DiffAtomizer struct {
	extractor *atomizer.Extractor
}

// NewDiffAtomizer creates a new diff atomizer
// llmClient should be a concrete *llm.GeminiClient or wrapped Client
func NewDiffAtomizer(llmClient *llm.Client) *DiffAtomizer {
	return &DiffAtomizer{
		extractor: atomizer.NewExtractor(llmClient),
	}
}

// ExtractBlocksFromDiff converts a git diff into a list of code block references
// This performs "meta-ingestion" - using the same LLM process as ingestion,
// but applied to uncommitted diffs to identify which blocks to query in the graph
func (d *DiffAtomizer) ExtractBlocksFromDiff(ctx context.Context, diff string) ([]tools.BlockReference, error) {
	// 1. Parse diff to get file paths (for validation)
	filePaths := ExtractFilePathsFromDiff(diff)
	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no valid file paths found in diff")
	}

	// 2. Create synthetic CommitData for the diff
	// We use a synthetic commit because the atomizer expects this format
	commit := atomizer.CommitData{
		SHA:         "uncommitted", // Special marker for uncommitted changes
		Message:     "Analyzing uncommitted changes", // Generic message
		DiffContent: diff,
		AuthorEmail: "local-user@local", // Placeholder
		Timestamp:   time.Now(),
	}

	// 3. Call the existing LLM extractor to get change events
	eventLog, err := d.extractor.ExtractCodeBlocks(ctx, commit)
	if err != nil {
		return nil, fmt.Errorf("failed to extract code blocks from diff: %w", err)
	}

	// 4. Convert ChangeEvents to BlockReferences
	var blockRefs []tools.BlockReference
	for _, event := range eventLog.ChangeEvents {
		// Only process block-level events (not imports or file-level events)
		if event.TargetBlockName == "" {
			continue
		}

		// Filter to relevant behaviors
		if event.Behavior != "CREATE_BLOCK" &&
		   event.Behavior != "MODIFY_BLOCK" &&
		   event.Behavior != "DELETE_BLOCK" {
			continue
		}

		blockRefs = append(blockRefs, tools.BlockReference{
			FilePath:  event.TargetFile,
			BlockName: event.TargetBlockName,
			BlockType: event.BlockType,
			Behavior:  event.Behavior,
		})
	}

	if len(blockRefs) == 0 {
		return nil, fmt.Errorf("no code blocks found in diff (LLM extracted %d events but none were block-level changes)", len(eventLog.ChangeEvents))
	}

	return blockRefs, nil
}

// ExtractBlocksFromDiffWithContext provides additional context for better extraction
// This version allows passing commit message and author info if available
func (d *DiffAtomizer) ExtractBlocksFromDiffWithContext(
	ctx context.Context,
	diff string,
	commitMessage string,
	authorEmail string,
) ([]tools.BlockReference, error) {
	// Create CommitData with provided context
	commit := atomizer.CommitData{
		SHA:         "uncommitted",
		Message:     commitMessage,
		DiffContent: diff,
		AuthorEmail: authorEmail,
		Timestamp:   time.Now(),
	}

	// Call extractor
	eventLog, err := d.extractor.ExtractCodeBlocks(ctx, commit)
	if err != nil {
		return nil, fmt.Errorf("failed to extract code blocks from diff: %w", err)
	}

	// Convert to BlockReferences
	var blockRefs []tools.BlockReference
	for _, event := range eventLog.ChangeEvents {
		if event.TargetBlockName == "" {
			continue
		}

		if event.Behavior != "CREATE_BLOCK" &&
		   event.Behavior != "MODIFY_BLOCK" &&
		   event.Behavior != "DELETE_BLOCK" {
			continue
		}

		blockRefs = append(blockRefs, tools.BlockReference{
			FilePath:  event.TargetFile,
			BlockName: event.TargetBlockName,
			BlockType: event.BlockType,
			Behavior:  event.Behavior,
		})
	}

	return blockRefs, nil
}
