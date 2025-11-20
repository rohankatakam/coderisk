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
	log.Printf("🔨 Starting chronological processing of %d commits for repo %d", len(commits), repoID)

	// 1. Initialize state tracker
	state := NewStateTracker()

	// 2. Load existing code blocks from database (if any)
	if err := p.dbWriter.LoadExistingBlocks(ctx, repoID, state); err != nil {
		log.Printf("⚠️  WARNING: Failed to load existing blocks: %v", err)
		// Continue anyway - this is just for incremental processing
	}
	log.Printf("  ✓ Loaded %d existing code blocks into state", state.GetBlockCount())

	// 3. Process each commit in chronological order
	successCount := 0
	errorCount := 0
	blocksCreated := 0
	blocksModified := 0
	blocksDeleted := 0
	importsAdded := 0
	importsRemoved := 0

	// Track progress every 10 commits
	batchSize := 10
	for i, commit := range commits {
		if i%batchSize == 0 || i == len(commits)-1 {
			log.Printf("  📥 Processing commit %d/%d: %s", i+1, len(commits), commit.SHA[:8])
		}

		// 3a. Extract events via LLM
		eventLog, err := p.extractor.ExtractCodeBlocks(ctx, commit)
		if err != nil {
			log.Printf("  ⚠️  WARNING: Failed to extract blocks from %s: %v", commit.SHA[:8], err)
			errorCount++
			continue // Skip failed commits
		}

		if i%batchSize == 0 || i == len(commits)-1 {
			log.Printf("    → Extracted %d events (summary: %s)",
				len(eventLog.ChangeEvents),
				eventLog.LLMIntentSummary[:min(60, len(eventLog.LLMIntentSummary))])
		}

		// 3b. Process each event
		commitHasErrors := false
		for j, event := range eventLog.ChangeEvents {
			if err := p.processEvent(ctx, &event, eventLog, commit, state, repoID); err != nil {
				log.Printf("  ⚠️  WARNING: Failed to process event %d in commit %s: %v", j, commit.SHA[:8], err)
				errorCount++
				commitHasErrors = true
			} else {
				successCount++
				// Track event types
				switch event.Behavior {
				case "CREATE_BLOCK":
					blocksCreated++
				case "MODIFY_BLOCK":
					blocksModified++
				case "DELETE_BLOCK":
					blocksDeleted++
				case "RENAME_BLOCK":
					blocksModified++ // Count renames as modifications
				case "ADD_IMPORT":
					importsAdded++
				case "REMOVE_IMPORT":
					importsRemoved++
				}
			}
		}

		// 3c. Mark commit as atomized (idempotency tracking)
		// Only mark as atomized if commit processed successfully (even if some events had warnings)
		if !commitHasErrors {
			if err := p.dbWriter.MarkCommitAtomized(ctx, commit.SHA, repoID); err != nil {
				log.Printf("  ⚠️  WARNING: Failed to mark commit %s as atomized: %v", commit.SHA[:8], err)
				// Continue anyway - this is just for idempotency tracking
			}
		}

		// Log cumulative progress every batchSize commits
		if (i+1)%batchSize == 0 || i == len(commits)-1 {
			log.Printf("  ✓ Progress: %d/%d commits | %d events | %d blocks created | %d modified",
				i+1, len(commits), successCount, blocksCreated, blocksModified)
		}
	}

	log.Printf("🎉 Chronological processing complete!")
	log.Printf("  📊 Summary:")
	log.Printf("     Total commits: %d", len(commits))
	log.Printf("     Total events: %d (errors: %d)", successCount, errorCount)
	log.Printf("     Blocks created: %d", blocksCreated)
	log.Printf("     Blocks modified: %d", blocksModified)
	log.Printf("     Blocks deleted: %d", blocksDeleted)
	log.Printf("     Imports added: %d", importsAdded)
	log.Printf("     Imports removed: %d", importsRemoved)
	log.Printf("     Final block count: %d", state.GetBlockCount())
	return nil
}

// processEvent handles a single ChangeEvent
// Reference: AGENT_P2B_PROCESSOR.md - Event type dispatch
func (p *Processor) processEvent(ctx context.Context, event *ChangeEvent, eventLog *CommitChangeEventLog, commit CommitData, state *StateTracker, repoID int64) error {
	switch event.Behavior {
	case "CREATE_BLOCK":
		return p.handleCreateBlock(ctx, event, eventLog, commit, state, repoID)
	case "MODIFY_BLOCK":
		return p.handleModifyBlock(ctx, event, eventLog, commit, state, repoID)
	case "DELETE_BLOCK":
		return p.handleDeleteBlock(ctx, event, eventLog, commit, state, repoID)
	case "RENAME_BLOCK":
		return p.handleRenameBlock(ctx, event, eventLog, commit, state, repoID)
	case "ADD_IMPORT":
		return p.handleAddImport(ctx, event, eventLog, commit, state, repoID)
	case "REMOVE_IMPORT":
		return p.handleRemoveImport(ctx, event, eventLog, commit, state, repoID)
	default:
		return fmt.Errorf("unknown behavior: %s", event.Behavior)
	}
}

// handleCreateBlock processes a CREATE_BLOCK event
// Reference: AGENT_P2B_PROCESSOR.md - CREATE_BLOCK handling
// Edge case: If block already exists, treat as MODIFY
func (p *Processor) handleCreateBlock(ctx context.Context, event *ChangeEvent, eventLog *CommitChangeEventLog, commit CommitData, state *StateTracker, repoID int64) error {
	// Check if block already exists in state (edge case)
	if blockID, exists := state.GetBlockID(event.TargetFile, event.TargetBlockName); exists {
		log.Printf("WARNING: CREATE_BLOCK for existing block %s:%s in state (treating as MODIFY)", event.TargetFile, event.TargetBlockName)

		// Update existing block
		if err := p.dbWriter.UpdateCodeBlock(ctx, blockID, commit.SHA, commit.AuthorEmail, commit.Timestamp); err != nil {
			return fmt.Errorf("failed to update existing block: %w", err)
		}

		// Create change record using LLM summary
		if err := p.dbWriter.CreateModification(ctx, blockID, repoID, event, commit.SHA, commit.AuthorEmail, commit.Timestamp, eventLog.LLMIntentSummary); err != nil {
			return fmt.Errorf("failed to create modification: %w", err)
		}

		// Create MODIFIED_BLOCK edge in Neo4j
		if err := p.graphWriter.CreateModifiedBlockEdge(ctx, commit.SHA, blockID, repoID, commit.Timestamp); err != nil {
			log.Printf("WARNING: Failed to create MODIFIED_BLOCK edge: %v", err)
		}

		return nil
	}

	// Check if block already exists in database (for idempotency)
	dbBlockID, err := p.dbWriter.FindCodeBlock(ctx, repoID, event.TargetFile, event.TargetBlockName)
	if err == nil && dbBlockID > 0 {
		log.Printf("WARNING: CREATE_BLOCK for existing block %s:%s in database (treating as MODIFY)", event.TargetFile, event.TargetBlockName)

		// Add to state tracker
		state.SetBlockID(event.TargetFile, event.TargetBlockName, dbBlockID)

		// Update existing block
		if err := p.dbWriter.UpdateCodeBlock(ctx, dbBlockID, commit.SHA, commit.AuthorEmail, commit.Timestamp); err != nil {
			return fmt.Errorf("failed to update existing block: %w", err)
		}

		// Create change record using LLM summary
		if err := p.dbWriter.CreateModification(ctx, dbBlockID, repoID, event, commit.SHA, commit.AuthorEmail, commit.Timestamp, eventLog.LLMIntentSummary); err != nil {
			return fmt.Errorf("failed to create modification: %w", err)
		}

		// Create MODIFIED_BLOCK edge in Neo4j
		if err := p.graphWriter.CreateModifiedBlockEdge(ctx, commit.SHA, dbBlockID, repoID, commit.Timestamp); err != nil {
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

	// Create change record (creation is a change event)
	if err := p.dbWriter.CreateModification(ctx, blockID, repoID, event, commit.SHA, commit.AuthorEmail, commit.Timestamp, eventLog.LLMIntentSummary); err != nil {
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
func (p *Processor) handleModifyBlock(ctx context.Context, event *ChangeEvent, eventLog *CommitChangeEventLog, commit CommitData, state *StateTracker, repoID int64) error {
	// Check if block exists
	blockID, exists := state.GetBlockID(event.TargetFile, event.TargetBlockName)

	if !exists {
		log.Printf("WARNING: MODIFY_BLOCK for non-existent block %s:%s (creating it)", event.TargetFile, event.TargetBlockName)
		// Treat as CREATE
		return p.handleCreateBlock(ctx, event, eventLog, commit, state, repoID)
	}

	// Update code block metadata
	if err := p.dbWriter.UpdateCodeBlock(ctx, blockID, commit.SHA, commit.AuthorEmail, commit.Timestamp); err != nil {
		return fmt.Errorf("failed to update code block: %w", err)
	}

	// Create change record
	if err := p.dbWriter.CreateModification(ctx, blockID, repoID, event, commit.SHA, commit.AuthorEmail, commit.Timestamp, eventLog.LLMIntentSummary); err != nil {
		return fmt.Errorf("failed to create modification: %w", err)
	}

	// Create MODIFIED_BLOCK edge in Neo4j
	if err := p.graphWriter.CreateModifiedBlockEdge(ctx, commit.SHA, blockID, repoID, commit.Timestamp); err != nil {
		log.Printf("WARNING: Failed to create MODIFIED_BLOCK edge: %v", err)
	}

	return nil
}

// handleRenameBlock processes a RENAME_BLOCK event
// Tracks function renames in function_identity_map for git log --follow functionality
func (p *Processor) handleRenameBlock(ctx context.Context, event *ChangeEvent, eventLog *CommitChangeEventLog, commit CommitData, state *StateTracker, repoID int64) error {
	// Validate event
	if err := event.ValidateEvent(); err != nil {
		return err
	}

	// Look up old block using state tracker first
	oldBlockID, exists := state.GetBlockID(event.TargetFile, event.OldBlockName)

	if !exists {
		// Try database lookup as fallback
		query := `
			SELECT id, signature
			FROM code_blocks
			WHERE repo_id = $1
			  AND canonical_file_path = $2
			  AND block_name = $3
			  AND current_status = 'active'
		`

		var oldSignature string
		err := p.db.QueryRowContext(ctx, query,
			repoID,
			event.TargetFile,
			event.OldBlockName,
		).Scan(&oldBlockID, &oldSignature)

		if err == sql.ErrNoRows {
			// Old block not found - fallback to DELETE + CREATE
			log.Printf("⚠️  WARNING: Rename target not found, treating as DELETE+CREATE - old_name: %s, new_name: %s, file: %s",
				event.OldBlockName, event.TargetBlockName, event.TargetFile)

			// Delete old (if exists in state)
			deleteEvent := &ChangeEvent{
				Behavior:        "DELETE_BLOCK",
				TargetFile:      event.TargetFile,
				TargetBlockName: event.OldBlockName,
			}
			_ = p.handleDeleteBlock(ctx, deleteEvent, eventLog, commit, state, repoID)

			// Create new
			createEvent := &ChangeEvent{
				Behavior:        "CREATE_BLOCK",
				TargetFile:      event.TargetFile,
				TargetBlockName: event.TargetBlockName,
				Signature:       event.Signature,
				BlockType:       event.BlockType,
				NewVersion:      event.NewVersion,
			}
			return p.handleCreateBlock(ctx, createEvent, eventLog, commit, state, repoID)
		} else if err != nil {
			return fmt.Errorf("failed to look up old block: %w", err)
		}
	}

	// Normalize new signature
	normalizedSig := NormalizeSignature(event.Signature)

	// Update code_blocks table
	updateQuery := `
		UPDATE code_blocks
		SET block_name = $1,
			signature = $2,
			historical_block_names = COALESCE(historical_block_names, '[]'::jsonb) || jsonb_build_array($3),
			last_modified_commit = $4,
			updated_at = NOW()
		WHERE id = $5
	`

	_, err := p.db.ExecContext(ctx, updateQuery,
		event.TargetBlockName,
		normalizedSig,
		event.OldBlockName,
		commit.SHA,
		oldBlockID,
	)
	if err != nil {
		return fmt.Errorf("failed to update block: %w", err)
	}

	// Update state tracker with new name
	state.DeleteBlock(event.TargetFile, event.OldBlockName)
	state.SetBlockID(event.TargetFile, event.TargetBlockName, oldBlockID)

	// Insert into function_identity_map
	identityQuery := `
		INSERT INTO function_identity_map
			(repo_id, block_id, historical_name, signature, commit_sha, rename_date)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	// Use commit timestamp as rename_date
	_, err = p.db.ExecContext(ctx, identityQuery,
		repoID,
		oldBlockID,
		event.OldBlockName,
		event.Signature, // Use original signature (not normalized) for historical tracking
		commit.SHA,
		commit.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to insert identity map: %w", err)
	}

	// Create code_block_changes entry
	changeQuery := `
		INSERT INTO code_block_changes
			(block_id, commit_sha, change_type, old_version, new_version)
		VALUES ($1, $2, 'renamed', $3, $4)
	`

	_, err = p.db.ExecContext(ctx, changeQuery,
		oldBlockID,
		commit.SHA,
		event.OldVersion,
		event.NewVersion,
	)
	if err != nil {
		return fmt.Errorf("failed to create change entry: %w", err)
	}

	// Create RENAMED_BLOCK edge in Neo4j (if graphWriter supports it)
	// Note: This may need to be implemented in graphWriter
	if err := p.graphWriter.CreateModifiedBlockEdge(ctx, commit.SHA, oldBlockID, repoID, commit.Timestamp); err != nil {
		log.Printf("⚠️  WARNING: Failed to create RENAMED_BLOCK edge: %v", err)
	}

	log.Printf("✓ Function renamed: %s → %s (block_id: %d, commit: %s)",
		event.OldBlockName, event.TargetBlockName, oldBlockID, commit.SHA[:8])

	return nil
}

// handleDeleteBlock processes a DELETE_BLOCK event
// Reference: AGENT_P2B_PROCESSOR.md - DELETE_BLOCK edge case
// Soft delete: Mark as deleted but keep in graph
func (p *Processor) handleDeleteBlock(ctx context.Context, event *ChangeEvent, eventLog *CommitChangeEventLog, commit CommitData, state *StateTracker, repoID int64) error {
	blockID, exists := state.GetBlockID(event.TargetFile, event.TargetBlockName)

	if !exists {
		log.Printf("WARNING: DELETE_BLOCK for non-existent block %s:%s (ignoring)", event.TargetFile, event.TargetBlockName)
		return nil
	}

	// Mark as deleted in PostgreSQL
	if err := p.dbWriter.MarkCodeBlockDeleted(ctx, blockID, commit.SHA, commit.Timestamp); err != nil {
		return fmt.Errorf("failed to mark block as deleted: %w", err)
	}

	// Create change record for deletion
	if err := p.dbWriter.CreateModification(ctx, blockID, repoID, event, commit.SHA, commit.AuthorEmail, commit.Timestamp, eventLog.LLMIntentSummary); err != nil {
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
func (p *Processor) handleAddImport(ctx context.Context, event *ChangeEvent, eventLog *CommitChangeEventLog, commit CommitData, state *StateTracker, repoID int64) error {
	// For now, we skip import tracking as it requires more context
	// This would need to resolve dependency_path to actual code blocks
	log.Printf("INFO: ADD_IMPORT event (not yet implemented): %s imports %s", event.TargetFile, event.DependencyPath)
	return nil
}

// handleRemoveImport processes a REMOVE_IMPORT event
// Reference: AGENT_P2B_PROCESSOR.md - Import dependency tracking
func (p *Processor) handleRemoveImport(ctx context.Context, event *ChangeEvent, eventLog *CommitChangeEventLog, commit CommitData, state *StateTracker, repoID int64) error {
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
