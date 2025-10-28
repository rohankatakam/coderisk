package graph

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// BatchNodeCreator handles efficient batch node creation with UNWIND
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 2
//
// The UNWIND pattern is the most efficient way to create multiple nodes:
// Instead of: MERGE (n:File {path: "a.js"}) MERGE (n:File {path: "b.js"})...
// We use: UNWIND $nodes AS node MERGE (n:File {path: node.path}) SET n += node
//
// This reduces round trips and allows Neo4j to optimize execution.
type BatchNodeCreator struct {
	driver   neo4j.DriverWithContext
	database string
	config   BatchConfig
}

// NewBatchNodeCreator creates a batch operation handler
func NewBatchNodeCreator(driver neo4j.DriverWithContext, database string, config BatchConfig) *BatchNodeCreator {
	return &BatchNodeCreator{
		driver:   driver,
		database: database,
		config:   config,
	}
}

// CreateFileNodes creates File nodes in optimized batches using UNWIND
func (b *BatchNodeCreator) CreateFileNodes(ctx context.Context, nodes []GraphNode) error {
	if len(nodes) == 0 {
		return nil
	}

	// Convert nodes to parameter format
	nodeParams := make([]map[string]any, len(nodes))
	for i, node := range nodes {
		nodeParams[i] = node.Properties
	}

	// Process in batches
	batchSize := b.config.FileBatchSize
	for i := 0; i < len(nodeParams); i += batchSize {
		end := i + batchSize
		if end > len(nodeParams) {
			end = len(nodeParams)
		}

		batch := nodeParams[i:end]

		// Use UNWIND for efficient batch creation
		// Match on path (Layer 1 key) for idempotent MERGE
		// Reference: dev_docs/01-architecture/simplified_graph_schema.md line 94 (File nodes use 'path' property)
		query := `
			UNWIND $nodes AS node
			MERGE (f:File {path: node.path})
			SET f += node
			RETURN count(f) as created
		`

		_, err := neo4j.ExecuteQuery(ctx, b.driver, query,
			map[string]any{"nodes": batch},
			neo4j.EagerResultTransformer,
			neo4j.ExecuteQueryWithDatabase(b.database))

		if err != nil {
			return fmt.Errorf("batch file creation failed (batch %d-%d): %w", i, end, err)
		}
	}

	return nil
}

// CreateFunctionNodes creates Function nodes efficiently
func (b *BatchNodeCreator) CreateFunctionNodes(ctx context.Context, nodes []GraphNode) error {
	if len(nodes) == 0 {
		return nil
	}

	nodeParams := make([]map[string]any, len(nodes))
	for i, node := range nodes {
		nodeParams[i] = node.Properties
	}

	batchSize := b.config.FunctionBatchSize
	for i := 0; i < len(nodeParams); i += batchSize {
		end := i + batchSize
		if end > len(nodeParams) {
			end = len(nodeParams)
		}

		batch := nodeParams[i:end]

		query := `
			UNWIND $nodes AS node
			MERGE (f:Function {unique_id: node.unique_id})
			SET f += node
			RETURN count(f) as created
		`

		_, err := neo4j.ExecuteQuery(ctx, b.driver, query,
			map[string]any{"nodes": batch},
			neo4j.EagerResultTransformer,
			neo4j.ExecuteQueryWithDatabase(b.database))

		if err != nil {
			return fmt.Errorf("batch function creation failed (batch %d-%d): %w", i, end, err)
		}
	}

	return nil
}

// CreateClassNodes creates Class nodes efficiently
func (b *BatchNodeCreator) CreateClassNodes(ctx context.Context, nodes []GraphNode) error {
	if len(nodes) == 0 {
		return nil
	}

	nodeParams := make([]map[string]any, len(nodes))
	for i, node := range nodes {
		nodeParams[i] = node.Properties
	}

	batchSize := b.config.ClassBatchSize
	for i := 0; i < len(nodeParams); i += batchSize {
		end := i + batchSize
		if end > len(nodeParams) {
			end = len(nodeParams)
		}

		batch := nodeParams[i:end]

		query := `
			UNWIND $nodes AS node
			MERGE (c:Class {unique_id: node.unique_id})
			SET c += node
			RETURN count(c) as created
		`

		_, err := neo4j.ExecuteQuery(ctx, b.driver, query,
			map[string]any{"nodes": batch},
			neo4j.EagerResultTransformer,
			neo4j.ExecuteQueryWithDatabase(b.database))

		if err != nil {
			return fmt.Errorf("batch class creation failed (batch %d-%d): %w", i, end, err)
		}
	}

	return nil
}

// CreateCommitNodes creates Commit nodes efficiently (Layer 2)
func (b *BatchNodeCreator) CreateCommitNodes(ctx context.Context, nodes []GraphNode) error {
	if len(nodes) == 0 {
		return nil
	}

	nodeParams := make([]map[string]any, len(nodes))
	for i, node := range nodes {
		nodeParams[i] = node.Properties
	}

	batchSize := b.config.CommitBatchSize
	for i := 0; i < len(nodeParams); i += batchSize {
		end := i + batchSize
		if end > len(nodeParams) {
			end = len(nodeParams)
		}

		batch := nodeParams[i:end]

		query := `
			UNWIND $nodes AS node
			MERGE (c:Commit {sha: node.sha})
			SET c += node
			RETURN count(c) as created
		`

		_, err := neo4j.ExecuteQuery(ctx, b.driver, query,
			map[string]any{"nodes": batch},
			neo4j.EagerResultTransformer,
			neo4j.ExecuteQueryWithDatabase(b.database))

		if err != nil {
			return fmt.Errorf("batch commit creation failed (batch %d-%d): %w", i, end, err)
		}
	}

	return nil
}

// CreateDeveloperNodes creates Developer nodes efficiently (Layer 2)
func (b *BatchNodeCreator) CreateDeveloperNodes(ctx context.Context, nodes []GraphNode) error {
	if len(nodes) == 0 {
		return nil
	}

	nodeParams := make([]map[string]any, len(nodes))
	for i, node := range nodes {
		nodeParams[i] = node.Properties
	}

	batchSize := b.config.DeveloperBatchSize
	for i := 0; i < len(nodeParams); i += batchSize {
		end := i + batchSize
		if end > len(nodeParams) {
			end = len(nodeParams)
		}

		batch := nodeParams[i:end]

		query := `
			UNWIND $nodes AS node
			MERGE (d:Developer {email: node.email})
			SET d += node
			RETURN count(d) as created
		`

		_, err := neo4j.ExecuteQuery(ctx, b.driver, query,
			map[string]any{"nodes": batch},
			neo4j.EagerResultTransformer,
			neo4j.ExecuteQueryWithDatabase(b.database))

		if err != nil {
			return fmt.Errorf("batch developer creation failed (batch %d-%d): %w", i, end, err)
		}
	}

	return nil
}

// CreateIssueNodes creates Issue nodes efficiently (Layer 3)
func (b *BatchNodeCreator) CreateIssueNodes(ctx context.Context, nodes []GraphNode) error {
	if len(nodes) == 0 {
		return nil
	}

	// Determine label from first node (all nodes in batch should have same label)
	label := "Issue"
	if len(nodes) > 0 && nodes[0].Label != "" {
		label = nodes[0].Label
	}

	nodeParams := make([]map[string]any, len(nodes))
	for i, node := range nodes {
		nodeParams[i] = node.Properties
	}

	batchSize := b.config.IncidentBatchSize
	for i := 0; i < len(nodeParams); i += batchSize {
		end := i + batchSize
		if end > len(nodeParams) {
			end = len(nodeParams)
		}

		batch := nodeParams[i:end]

		// Use dynamic label for Issue/PR/Incident nodes (all use 'number' as unique key)
		query := fmt.Sprintf(`
			UNWIND $nodes AS node
			MERGE (n:%s {number: node.number})
			SET n += node
			RETURN count(n) as created
		`, label)

		_, err := neo4j.ExecuteQuery(ctx, b.driver, query,
			map[string]any{"nodes": batch},
			neo4j.EagerResultTransformer,
			neo4j.ExecuteQueryWithDatabase(b.database))

		if err != nil {
			return fmt.Errorf("batch %s creation failed (batch %d-%d): %w", label, i, end, err)
		}
	}

	return nil
}

// CreateEdgesBatch creates edges in optimized batches using UNWIND
// Groups edges by type for efficiency
func (b *BatchNodeCreator) CreateEdgesBatch(ctx context.Context, edges []GraphEdge) error {
	if len(edges) == 0 {
		return nil
	}

	// Group edges by type for efficiency
	edgesByType := make(map[string][]GraphEdge)
	for _, edge := range edges {
		edgesByType[edge.Label] = append(edgesByType[edge.Label], edge)
	}

	// Process each edge type in batches
	for edgeType, edgeList := range edgesByType {
		if err := b.createEdgesBatchByType(ctx, edgeType, edgeList); err != nil {
			return err
		}
	}

	return nil
}

// createEdgesBatchByType processes a batch of edges of the same type
func (b *BatchNodeCreator) createEdgesBatchByType(ctx context.Context, edgeType string, edges []GraphEdge) error {
	batchSize := b.config.EdgeBatchSize

	for i := 0; i < len(edges); i += batchSize {
		end := i + batchSize
		if end > len(edges) {
			end = len(edges)
		}

		batch := edges[i:end]

		// Convert to parameter format
		edgeParams := make([]map[string]any, len(batch))
		for j, edge := range batch {
			fromLabel, fromID := parseNodeID(edge.From)
			toLabel, toID := parseNodeID(edge.To)

			// Get unique keys for node matching
			fromKey := getUniqueKey(fromLabel)
			toKey := getUniqueKey(toLabel)

			edgeParams[j] = map[string]any{
				"from_key":   fromKey,
				"from_id":    fromID,
				"from_label": fromLabel,
				"to_key":     toKey,
				"to_id":      toID,
				"to_label":   toLabel,
				"props":      edge.Properties,
			}
		}

		// Build dynamic UNWIND query
		// Note: We can't use dynamic labels in Cypher, so we use WHERE clause
		query := fmt.Sprintf(`
			UNWIND $edges AS edge
			MATCH (from)
			WHERE edge.from_label IN labels(from) AND from[edge.from_key] = edge.from_id
			MATCH (to)
			WHERE edge.to_label IN labels(to) AND to[edge.to_key] = edge.to_id
			MERGE (from)-[r:%s]->(to)
			SET r += edge.props
			RETURN count(r) as created
		`, sanitizeLabel(edgeType))

		result, err := neo4j.ExecuteQuery(ctx, b.driver, query,
			map[string]any{"edges": edgeParams},
			neo4j.EagerResultTransformer,
			neo4j.ExecuteQueryWithDatabase(b.database))

		if err != nil {
			return fmt.Errorf("batch edge creation failed for %s (batch %d-%d): %w",
				edgeType, i, end, err)
		}

		// Log edge creation count for debugging
		if len(result.Records) > 0 {
			if created, ok := result.Records[0].Get("created"); ok {
				createdCount := created.(int64)
				if createdCount < int64(len(batch)) {
					log.Printf("⚠️  WARNING: Only created %d/%d %s edges (batch %d-%d). Some nodes may not exist.",
						createdCount, len(batch), edgeType, i, end)
				}
			}
		}
	}

	return nil
}

// sanitizeLabel ensures label is safe for Cypher (already validated by CypherBuilder, but extra safety)
func sanitizeLabel(label string) string {
	// Only allow alphanumeric and underscore
	result := strings.Builder{}
	for _, r := range label {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		}
	}
	return result.String()
}
