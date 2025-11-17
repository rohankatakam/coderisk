package atomizer

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Processor orchestrates the chronological processing of commits
// Reference: AGENT_P2B_PROCESSOR.md - Main event processing loop
type Processor struct {
	extractor   *Extractor
	dbWriter    *DBWriter
	graphWriter *GraphWriter
	db          *sql.DB
}

// NewProcessor creates a new event processor
func NewProcessor(extractor *Extractor, db *sql.DB, neoDriver neo4j.DriverWithContext, neoDatabase string) *Processor {
	return &Processor{
		extractor:   extractor,
		dbWriter:    NewDBWriter(db),
		graphWriter: NewGraphWriter(neoDriver, neoDatabase),
		db:          db,
	}
}

// ProcessCommitsChronologically processes all commits in chronological order
// Reference: AGENT_P2B_PROCESSOR.md - Core processing logic
func (p *Processor) ProcessCommitsChronologically(ctx context.Context, commits []CommitData, repoID int64) error {
	log.Printf("Starting chronological processing of %d commits for repo %d", len(commits), repoID)

	// 1. Initialize state tracker
	state := NewStateTracker()

	// 2. Load existing code blocks from database (if any)
	if err := p.dbWriter.LoadExistingBlocks(ctx, repoID, state); err != nil {
		log.Printf("WARNING: Failed to load existing blocks: %v", err)
		// Continue anyway - this is just for incremental processing
	}
	log.Printf("Loaded %d existing code blocks into state", state.GetBlockCount())

	// 3. Process each commit in chronological order
	successCount := 0
	errorCount := 0

	for i, commit := range commits {
		log.Printf("Processing commit %d/%d: %s (%s)", i+1, len(commits), commit.SHA, commit.Message[:min(50, len(commit.Message))])

		// 3a. Extract events via LLM
		eventLog, err := p.extractor.ExtractCodeBlocks(ctx, commit)
		if err != nil {
			log.Printf("WARNING: Failed to extract blocks from %s: %v", commit.SHA, err)
			errorCount++
			continue // Skip failed commits
		}

		// 3b. Process each event
		for j, event := range eventLog.ChangeEvents {
			if err := p.processEvent(ctx, &event, commit, state, repoID); err != nil {
				log.Printf("WARNING: Failed to process event %d in commit %s: %v", j, commit.SHA, err)
				errorCount++
			} else {
				successCount++
			}
		}
	}

	log.Printf("Chronological processing complete: %d events processed, %d errors", successCount, errorCount)
	return nil
}

// processEvent handles a single ChangeEvent
// Reference: AGENT_P2B_PROCESSOR.md - Event type dispatch
func (p *Processor) processEvent(ctx context.Context, event *ChangeEvent, commit CommitData, state *StateTracker, repoID int64) error {
	switch event.Behavior {
	case "CREATE_BLOCK":
		return p.handleCreateBlock(ctx, event, commit, state, repoID)
	case "MODIFY_BLOCK":
		return p.handleModifyBlock(ctx, event, commit, state, repoID)
	case "DELETE_BLOCK":
		return p.handleDeleteBlock(ctx, event, commit, state, repoID)
	case "ADD_IMPORT":
		return p.handleAddImport(ctx, event, commit, state, repoID)
	case "REMOVE_IMPORT":
		return p.handleRemoveImport(ctx, event, commit, state, repoID)
	default:
		return fmt.Errorf("unknown behavior: %s", event.Behavior)
	}
}

// handleCreateBlock processes a CREATE_BLOCK event
// Reference: AGENT_P2B_PROCESSOR.md - CREATE_BLOCK handling
// Edge case: If block already exists, treat as MODIFY
func (p *Processor) handleCreateBlock(ctx context.Context, event *ChangeEvent, commit CommitData, state *StateTracker, repoID int64) error {
	// Check if block already exists (edge case)
	if blockID, exists := state.GetBlockID(event.TargetFile, event.TargetBlockName); exists {
		log.Printf("WARNING: CREATE_BLOCK for existing block %s:%s (treating as MODIFY)", event.TargetFile, event.TargetBlockName)

		// Update existing block
		if err := p.dbWriter.UpdateCodeBlock(ctx, blockID, commit.SHA, commit.AuthorEmail, commit.Timestamp); err != nil {
			return fmt.Errorf("failed to update existing block: %w", err)
		}

		// Create modification record
		if err := p.dbWriter.CreateModification(ctx, blockID, repoID, commit.SHA, commit.AuthorEmail, commit.Timestamp, "modify"); err != nil {
			return fmt.Errorf("failed to create modification: %w", err)
		}

		// Create MODIFIED_BLOCK edge in Neo4j
		if err := p.graphWriter.CreateModifiedBlockEdge(ctx, commit.SHA, blockID, repoID, commit.Timestamp); err != nil {
			log.Printf("WARNING: Failed to create MODIFIED_BLOCK edge: %v", err)
		}

		return nil
	}

	// Create new code block in PostgreSQL
	blockID, err := p.dbWriter.CreateCodeBlock(ctx, event, commit.SHA, commit.AuthorEmail, commit.Timestamp, repoID)
	if err != nil {
		return fmt.Errorf("failed to create code block: %w", err)
	}

	// Track in state
	state.SetBlockID(event.TargetFile, event.TargetBlockName, blockID)

	// Create modification record (creation is a modification)
	if err := p.dbWriter.CreateModification(ctx, blockID, repoID, commit.SHA, commit.AuthorEmail, commit.Timestamp, "create"); err != nil {
		return fmt.Errorf("failed to create modification record: %w", err)
	}

	// Create CodeBlock node in Neo4j
	if err := p.graphWriter.CreateCodeBlockNode(ctx, blockID, event, repoID); err != nil {
		log.Printf("WARNING: Failed to create CodeBlock node: %v", err)
		// Continue - PostgreSQL is source of truth
	}

	// Create CREATED_BLOCK edge in Neo4j
	if err := p.graphWriter.CreateCreatedBlockEdge(ctx, commit.SHA, blockID, event, repoID, commit.Timestamp); err != nil {
		log.Printf("WARNING: Failed to create CREATED_BLOCK edge: %v", err)
	}

	// Create CONTAINS edge from File to CodeBlock
	if err := p.graphWriter.CreateContainsEdge(ctx, event.TargetFile, blockID, repoID); err != nil {
		log.Printf("WARNING: Failed to create CONTAINS edge: %v", err)
	}

	return nil
}

// handleModifyBlock processes a MODIFY_BLOCK event
// Reference: AGENT_P2B_PROCESSOR.md - MODIFY_BLOCK handling
// Edge case: If block doesn't exist, create it (late detection)
func (p *Processor) handleModifyBlock(ctx context.Context, event *ChangeEvent, commit CommitData, state *StateTracker, repoID int64) error {
	// Check if block exists
	blockID, exists := state.GetBlockID(event.TargetFile, event.TargetBlockName)

	if !exists {
		log.Printf("WARNING: MODIFY_BLOCK for non-existent block %s:%s (creating it)", event.TargetFile, event.TargetBlockName)
		// Treat as CREATE
		return p.handleCreateBlock(ctx, event, commit, state, repoID)
	}

	// Update code block metadata
	if err := p.dbWriter.UpdateCodeBlock(ctx, blockID, commit.SHA, commit.AuthorEmail, commit.Timestamp); err != nil {
		return fmt.Errorf("failed to update code block: %w", err)
	}

	// Create modification record
	if err := p.dbWriter.CreateModification(ctx, blockID, repoID, commit.SHA, commit.AuthorEmail, commit.Timestamp, "modify"); err != nil {
		return fmt.Errorf("failed to create modification: %w", err)
	}

	// Create MODIFIED_BLOCK edge in Neo4j
	if err := p.graphWriter.CreateModifiedBlockEdge(ctx, commit.SHA, blockID, repoID, commit.Timestamp); err != nil {
		log.Printf("WARNING: Failed to create MODIFIED_BLOCK edge: %v", err)
	}

	return nil
}

// handleDeleteBlock processes a DELETE_BLOCK event
// Reference: AGENT_P2B_PROCESSOR.md - DELETE_BLOCK edge case
// Soft delete: Mark as deleted but keep in graph
func (p *Processor) handleDeleteBlock(ctx context.Context, event *ChangeEvent, commit CommitData, state *StateTracker, repoID int64) error {
	blockID, exists := state.GetBlockID(event.TargetFile, event.TargetBlockName)

	if !exists {
		log.Printf("WARNING: DELETE_BLOCK for non-existent block %s:%s (ignoring)", event.TargetFile, event.TargetBlockName)
		return nil
	}

	// Mark as deleted in PostgreSQL
	if err := p.dbWriter.MarkCodeBlockDeleted(ctx, blockID, commit.SHA, commit.Timestamp); err != nil {
		return fmt.Errorf("failed to mark block as deleted: %w", err)
	}

	// Create modification record for deletion
	if err := p.dbWriter.CreateModification(ctx, blockID, repoID, commit.SHA, commit.AuthorEmail, commit.Timestamp, "delete"); err != nil {
		return fmt.Errorf("failed to create deletion modification: %w", err)
	}

	// Create DELETED_BLOCK edge in Neo4j
	if err := p.graphWriter.CreateDeletedBlockEdge(ctx, commit.SHA, blockID, repoID, commit.Timestamp); err != nil {
		log.Printf("WARNING: Failed to create DELETED_BLOCK edge: %v", err)
	}

	// Remove from state tracker
	state.DeleteBlock(event.TargetFile, event.TargetBlockName)

	return nil
}

// handleAddImport processes an ADD_IMPORT event
// Reference: AGENT_P2B_PROCESSOR.md - Import dependency tracking
func (p *Processor) handleAddImport(ctx context.Context, event *ChangeEvent, commit CommitData, state *StateTracker, repoID int64) error {
	// For now, we skip import tracking as it requires more context
	// This would need to resolve dependency_path to actual code blocks
	log.Printf("INFO: ADD_IMPORT event (not yet implemented): %s imports %s", event.TargetFile, event.DependencyPath)
	return nil
}

// handleRemoveImport processes a REMOVE_IMPORT event
// Reference: AGENT_P2B_PROCESSOR.md - Import dependency tracking
func (p *Processor) handleRemoveImport(ctx context.Context, event *ChangeEvent, commit CommitData, state *StateTracker, repoID int64) error {
	// For now, we skip import tracking as it requires more context
	log.Printf("INFO: REMOVE_IMPORT event (not yet implemented): %s removes import %s", event.TargetFile, event.DependencyPath)
	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
