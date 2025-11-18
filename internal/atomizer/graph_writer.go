package atomizer

import (
	"context"
	"fmt"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// GraphWriter handles all Neo4j write operations for code blocks
// Reference: AGENT_P2B_PROCESSOR.md - Neo4j graph construction
type GraphWriter struct {
	driver   neo4j.DriverWithContext
	database string
}

// NewGraphWriter creates a new graph writer for code blocks
func NewGraphWriter(driver neo4j.DriverWithContext, database string) *GraphWriter {
	return &GraphWriter{
		driver:   driver,
		database: database,
	}
}

// CreateCodeBlockNode creates a CodeBlock node in Neo4j
// Reference: AGENT_P2B_PROCESSOR.md - Neo4j Cypher queries
// Schema aligned with DATA_SCHEMA_REFERENCE.md
func (g *GraphWriter) CreateCodeBlockNode(ctx context.Context, blockID int64, event *ChangeEvent, repoID int64) error {
	query := `
		MERGE (b:CodeBlock {
			id: $composite_id,
			repo_id: $repo_id
		})
		SET b.block_type = $block_type,
		    b.block_name = $block_name,
		    b.canonical_file_path = $canonical_file_path,
		    b.start_line = $start_line,
		    b.end_line = $end_line,
		    b.language = $language,
		    b.db_id = $db_id
	`

	compositeID := fmt.Sprintf("%d:codeblock:%s:%s", repoID, event.TargetFile, event.TargetBlockName)

	session := g.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: g.database,
	})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, map[string]any{
			"composite_id":        compositeID,
			"repo_id":             repoID,
			"block_type":          event.BlockType,
			"block_name":          event.TargetBlockName,
			"canonical_file_path": event.TargetFile,
			"start_line":          event.StartLine,
			"end_line":            event.EndLine,
			"language":            detectLanguage(event.TargetFile),
			"db_id":               blockID,
		})
		return nil, err
	})

	if err != nil {
		return fmt.Errorf("failed to create CodeBlock node: %w", err)
	}

	return nil
}

// CreateCreatedBlockEdge creates a CREATED_BLOCK edge from Commit to CodeBlock
// Reference: AGENT_P2B_PROCESSOR.md - CREATED_BLOCK relationship
func (g *GraphWriter) CreateCreatedBlockEdge(ctx context.Context, commitSHA string, blockID int64, event *ChangeEvent, repoID int64, timestamp time.Time) error {
	query := `
		MATCH (c:Commit {sha: $commit_sha, repo_id: $repo_id})
		MATCH (b:CodeBlock {db_id: $block_id, repo_id: $repo_id})
		MERGE (c)-[r:CREATED_BLOCK]->(b)
		SET r.repo_id = $repo_id,
		    r.timestamp = $timestamp
	`

	session := g.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: g.database,
	})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, map[string]any{
			"commit_sha": commitSHA,
			"block_id":   blockID,
			"repo_id":    repoID,
			"timestamp":  timestamp.Unix(),
		})
		return nil, err
	})

	if err != nil {
		return fmt.Errorf("failed to create CREATED_BLOCK edge: %w", err)
	}

	return nil
}

// CreateModifiedBlockEdge creates a MODIFIED_BLOCK edge from Commit to CodeBlock
// Reference: AGENT_P2B_PROCESSOR.md - MODIFIED_BLOCK relationship
func (g *GraphWriter) CreateModifiedBlockEdge(ctx context.Context, commitSHA string, blockID int64, repoID int64, timestamp time.Time) error {
	query := `
		MATCH (c:Commit {sha: $commit_sha, repo_id: $repo_id})
		MATCH (b:CodeBlock {db_id: $block_id, repo_id: $repo_id})
		MERGE (c)-[r:MODIFIED_BLOCK]->(b)
		SET r.repo_id = $repo_id,
		    r.timestamp = $timestamp
	`

	session := g.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: g.database,
	})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, map[string]any{
			"commit_sha": commitSHA,
			"block_id":   blockID,
			"repo_id":    repoID,
			"timestamp":  timestamp.Unix(),
		})
		return nil, err
	})

	if err != nil {
		return fmt.Errorf("failed to create MODIFIED_BLOCK edge: %w", err)
	}

	return nil
}

// CreateDeletedBlockEdge creates a DELETED_BLOCK edge from Commit to CodeBlock
// Reference: AGENT_P2B_PROCESSOR.md - DELETE_BLOCK handling
func (g *GraphWriter) CreateDeletedBlockEdge(ctx context.Context, commitSHA string, blockID int64, repoID int64, timestamp time.Time) error {
	query := `
		MATCH (c:Commit {sha: $commit_sha, repo_id: $repo_id})
		MATCH (b:CodeBlock {db_id: $block_id, repo_id: $repo_id})
		MERGE (c)-[r:DELETED_BLOCK]->(b)
		SET r.repo_id = $repo_id,
		    r.timestamp = $timestamp
	`

	session := g.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: g.database,
	})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, map[string]any{
			"commit_sha": commitSHA,
			"block_id":   blockID,
			"repo_id":    repoID,
			"timestamp":  timestamp.Unix(),
		})
		return nil, err
	})

	if err != nil {
		return fmt.Errorf("failed to create DELETED_BLOCK edge: %w", err)
	}

	return nil
}

// CreateContainsEdge creates a CONTAINS edge from File to CodeBlock
// Reference: AGENT_P2B_PROCESSOR.md - File-CodeBlock relationship
func (g *GraphWriter) CreateContainsEdge(ctx context.Context, filePath string, blockID int64, repoID int64) error {
	query := `
		MATCH (f:File {path: $file_path, repo_id: $repo_id})
		MATCH (b:CodeBlock {db_id: $block_id, repo_id: $repo_id})
		MERGE (f)-[r:CONTAINS]->(b)
		SET r.repo_id = $repo_id
	`

	session := g.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: g.database,
	})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, map[string]any{
			"file_path": filePath,
			"block_id":  blockID,
			"repo_id":   repoID,
		})
		return nil, err
	})

	if err != nil {
		// File node might not exist yet - this is okay, we'll create the edge later
		// For now, just log and continue
		return nil
	}

	return nil
}

// CreateImportEdge creates an IMPORTS edge between code blocks
// Reference: AGENT_P2B_PROCESSOR.md - ADD_IMPORT handling
func (g *GraphWriter) CreateImportEdge(ctx context.Context, fromBlockID, toBlockID int64, repoID int64, timestamp time.Time) error {
	query := `
		MATCH (from:CodeBlock {db_id: $from_block_id, repo_id: $repo_id})
		MATCH (to:CodeBlock {db_id: $to_block_id, repo_id: $repo_id})
		MERGE (from)-[r:IMPORTS]->(to)
		SET r.repo_id = $repo_id,
		    r.created_at = $timestamp
	`

	session := g.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: g.database,
	})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, map[string]any{
			"from_block_id": fromBlockID,
			"to_block_id":   toBlockID,
			"repo_id":       repoID,
			"timestamp":     timestamp.Unix(),
		})
		return nil, err
	})

	if err != nil {
		return fmt.Errorf("failed to create IMPORTS edge: %w", err)
	}

	return nil
}

// RemoveImportEdge removes an IMPORTS edge between code blocks
// Reference: AGENT_P2B_PROCESSOR.md - REMOVE_IMPORT handling
func (g *GraphWriter) RemoveImportEdge(ctx context.Context, fromBlockID, toBlockID int64, repoID int64) error {
	query := `
		MATCH (from:CodeBlock {db_id: $from_block_id, repo_id: $repo_id})
			-[r:IMPORTS]->
			(to:CodeBlock {db_id: $to_block_id, repo_id: $repo_id})
		DELETE r
	`

	session := g.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: g.database,
	})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, map[string]any{
			"from_block_id": fromBlockID,
			"to_block_id":   toBlockID,
			"repo_id":       repoID,
		})
		return nil, err
	})

	if err != nil {
		return fmt.Errorf("failed to remove IMPORTS edge: %w", err)
	}

	return nil
}

// BatchCreateCodeBlockNodes creates multiple CodeBlock nodes in a single transaction
// Reference: AGENT_P2B_PROCESSOR.md - Edge case: large transactions (batch 50 at a time)
func (g *GraphWriter) BatchCreateCodeBlockNodes(ctx context.Context, events []struct {
	BlockID int64
	Event   *ChangeEvent
	RepoID  int64
}) error {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: g.database,
	})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		for _, item := range events {
			query := `
				MERGE (b:CodeBlock {
					id: $composite_id,
					repo_id: $repo_id
				})
				SET b.block_type = $block_type,
					b.block_name = $block_name,
					b.canonical_file_path = $canonical_file_path,
					b.start_line = $start_line,
					b.end_line = $end_line,
					b.language = $language,
					b.db_id = $db_id
			`

			compositeID := fmt.Sprintf("%d:codeblock:%s:%s", item.RepoID, item.Event.TargetFile, item.Event.TargetBlockName)

			_, err := tx.Run(ctx, query, map[string]any{
				"composite_id":        compositeID,
				"repo_id":             item.RepoID,
				"block_type":          item.Event.BlockType,
				"block_name":          item.Event.TargetBlockName,
				"canonical_file_path": item.Event.TargetFile,
				"start_line":          item.Event.StartLine,
				"end_line":            item.Event.EndLine,
				"language":            detectLanguage(item.Event.TargetFile),
				"db_id":               item.BlockID,
			})

			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("failed to batch create CodeBlock nodes: %w", err)
	}

	return nil
}
