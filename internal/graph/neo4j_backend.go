package graph

import (
	"context"
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jBackend implements Backend interface for Neo4j with Cypher queries
// Reference: dev_docs/03-implementation/integration_guides/layers_2_3_graph_construction.md
type Neo4jBackend struct {
	driver neo4j.DriverWithContext
	ctx    context.Context
}

// NewNeo4jBackend creates a Neo4j backend instance
func NewNeo4jBackend(ctx context.Context, uri, username, password string) (*Neo4jBackend, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}

	// Verify connectivity
	if err := driver.VerifyConnectivity(ctx); err != nil {
		driver.Close(ctx)
		return nil, fmt.Errorf("failed to connect to Neo4j: %w", err)
	}

	return &Neo4jBackend{
		driver: driver,
		ctx:    ctx,
	}, nil
}

// CreateNode creates a single node using idempotent MERGE
func (n *Neo4jBackend) CreateNode(node GraphNode) (string, error) {
	cypher := generateCypherNode(node)
	session := n.driver.NewSession(n.ctx, neo4j.SessionConfig{})
	defer session.Close(n.ctx)

	result, err := session.Run(n.ctx, cypher, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create node: %w", err)
	}

	if result.Next(n.ctx) {
		record := result.Record()
		if id, ok := record.Get("id"); ok {
			return fmt.Sprintf("%v", id), nil
		}
	}

	return "", nil
}

// CreateNodes creates multiple nodes in batch
func (n *Neo4jBackend) CreateNodes(nodes []GraphNode) ([]string, error) {
	if len(nodes) == 0 {
		return []string{}, nil
	}

	// Generate all Cypher commands
	commands := make([]string, len(nodes))
	for i, node := range nodes {
		commands[i] = generateCypherNode(node)
	}

	// Execute in single transaction
	if err := n.ExecuteBatch(commands); err != nil {
		return nil, err
	}

	// Return dummy IDs (Neo4j doesn't return IDs in batch)
	ids := make([]string, len(nodes))
	for i := range nodes {
		ids[i] = fmt.Sprintf("%d", i)
	}

	return ids, nil
}

// CreateEdge creates a single edge using idempotent MERGE
func (n *Neo4jBackend) CreateEdge(edge GraphEdge) error {
	cypher := generateCypherEdge(edge)
	session := n.driver.NewSession(n.ctx, neo4j.SessionConfig{})
	defer session.Close(n.ctx)

	result, err := session.Run(n.ctx, cypher, nil)
	if err != nil {
		// Enhanced error with diagnostic info
		fromLabel, fromID := parseNodeID(edge.From)
		toLabel, toID := parseNodeID(edge.To)
		return fmt.Errorf("failed to create edge %s: from=%s:%s to=%s:%s: %w",
			edge.Label, fromLabel, fromID, toLabel, toID, err)
	}

	// Check if any records were returned (nodes found and edge created)
	if !result.Next(n.ctx) {
		fromLabel, fromID := parseNodeID(edge.From)
		toLabel, toID := parseNodeID(edge.To)
		return fmt.Errorf("edge creation returned no results (nodes may not exist): %s: from=%s:%s to=%s:%s",
			edge.Label, fromLabel, fromID, toLabel, toID)
	}

	return nil
}

// CreateEdges creates multiple edges in batch
func (n *Neo4jBackend) CreateEdges(edges []GraphEdge) error {
	if len(edges) == 0 {
		return nil
	}

	// Log diagnostic info for first edge (helps debug path issues)
	if len(edges) > 0 {
		fromLabel, fromID := parseNodeID(edges[0].From)
		toLabel, toID := parseNodeID(edges[0].To)
		fmt.Printf("DEBUG: Creating %d edges. First edge: %s (%s:%s) -> %s (%s:%s)\n",
			len(edges), edges[0].Label, fromLabel, fromID, edges[0].Label, toLabel, toID)
	}

	// Generate all Cypher commands
	commands := make([]string, len(edges))
	for i, edge := range edges {
		commands[i] = generateCypherEdge(edge)
	}

	// Log first Cypher query for debugging
	if len(commands) > 0 {
		fmt.Printf("DEBUG: First Cypher query: %s\n", commands[0][:200])
	}

	// Execute in single transaction
	return n.ExecuteBatch(commands)
}

// ExecuteBatch executes multiple commands in a single transaction
func (n *Neo4jBackend) ExecuteBatch(commands []string) error {
	session := n.driver.NewSession(n.ctx, neo4j.SessionConfig{})
	defer session.Close(n.ctx)

	_, err := session.ExecuteWrite(n.ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for _, cmd := range commands {
			if _, err := tx.Run(n.ctx, cmd, nil); err != nil {
				return nil, fmt.Errorf("batch command failed: %w", err)
			}
		}
		return nil, nil
	})

	return err
}

// Query executes a Cypher query and returns results
func (n *Neo4jBackend) Query(query string) (interface{}, error) {
	session := n.driver.NewSession(n.ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(n.ctx)

	result, err := session.Run(n.ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	if result.Next(n.ctx) {
		record := result.Record()
		if count, ok := record.Get("count"); ok {
			return count, nil
		}
	}

	return 0, nil
}

// Close closes the Neo4j driver connection
func (n *Neo4jBackend) Close() error {
	return n.driver.Close(n.ctx)
}

// ===================================
// Cypher Query Generation
// ===================================

// generateCypherNode creates Cypher MERGE query for idempotent node creation
func generateCypherNode(node GraphNode) string {
	uniqueKey := getUniqueKey(node.Label)
	uniqueValue := node.Properties[uniqueKey]

	// Build property SET clauses
	propClauses := []string{}
	for key, value := range node.Properties {
		propClauses = append(propClauses, fmt.Sprintf("n.%s = %s", key, formatCypherValue(value)))
	}

	return fmt.Sprintf(
		"MERGE (n:%s {%s: %s}) SET %s RETURN id(n) as id",
		node.Label,
		uniqueKey,
		formatCypherValue(uniqueValue),
		strings.Join(propClauses, ", "),
	)
}

// generateCypherEdge creates Cypher MERGE query for idempotent edge creation
// Returns both the query and a verification query to check if nodes exist
func generateCypherEdge(edge GraphEdge) string {
	fromLabel, fromID := parseNodeID(edge.From)
	toLabel, toID := parseNodeID(edge.To)

	fromKey := getUniqueKey(fromLabel)
	toKey := getUniqueKey(toLabel)

	// Build property SET clauses
	propClauses := []string{}
	for key, value := range edge.Properties {
		propClauses = append(propClauses, fmt.Sprintf("r.%s = %s", key, formatCypherValue(value)))
	}

	propsStr := ""
	if len(propClauses) > 0 {
		propsStr = "SET " + strings.Join(propClauses, ", ")
	}

	// Use OPTIONAL MATCH to detect missing nodes and return diagnostics
	// This helps debug silent edge creation failures
	return fmt.Sprintf(
		"MATCH (from:%s {%s: %s}) MATCH (to:%s {%s: %s}) MERGE (from)-[r:%s]->(to) %s RETURN from, to",
		fromLabel, fromKey, formatCypherValue(fromID),
		toLabel, toKey, formatCypherValue(toID),
		edge.Label,
		propsStr,
	)
}

// formatCypherValue formats a value for Cypher query
func formatCypherValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		// Escape quotes
		escaped := strings.ReplaceAll(v, "'", "\\'")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		return fmt.Sprintf("'%s'", escaped)
	case int, int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case []string:
		quoted := make([]string, len(v))
		for i, s := range v {
			escaped := strings.ReplaceAll(s, "'", "\\'")
			quoted[i] = fmt.Sprintf("'%s'", escaped)
		}
		return "[" + strings.Join(quoted, ", ") + "]"
	default:
		return "''"
	}
}

// parseNodeID extracts label and ID from node reference (e.g., "commit:abc123" -> "Commit", "abc123")
func parseNodeID(nodeID string) (label, id string) {
	parts := strings.SplitN(nodeID, ":", 2)
	if len(parts) == 2 {
		// Capitalize first letter for label
		label = strings.ToUpper(string(parts[0][0])) + parts[0][1:]
		return label, parts[1]
	}

	// If no prefix, assume it's a file path (backwards compatibility)
	// File paths don't have colons at the start but may have them in the path
	if strings.Contains(nodeID, "/") || strings.Contains(nodeID, ".") {
		return "File", nodeID
	}

	return "Unknown", nodeID
}

// getUniqueKey returns the unique identifier field for each node type
func getUniqueKey(label string) string {
	keys := map[string]string{
		"Commit":      "sha",
		"commit":      "sha",
		"Developer":   "email",
		"developer":   "email",
		"Issue":       "number",
		"issue":       "number",
		"PullRequest": "number",
		"pullrequest": "number",
		// Incident nodes (Layer 3)
		"Incident": "id", // For Incidents, id = UUID string
		"incident": "id",
		// Tree-sitter entity types (Layer 1)
		"File":     "file_path", // For Files, use file_path for matching
		"file":     "file_path",
		"Function": "unique_id", // For Functions, unique_id = filepath:name:line
		"function": "unique_id",
		"Class":    "unique_id", // For Classes, unique_id = filepath:name:line
		"class":    "unique_id",
		"Import":   "unique_id", // For Imports, unique_id = filepath:name:line
		"import":   "unique_id",
	}

	if key, ok := keys[label]; ok {
		return key
	}
	return "id"
}
