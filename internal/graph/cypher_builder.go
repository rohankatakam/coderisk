package graph

import (
	"fmt"
	"regexp"
	"strings"
)

// CypherBuilder builds safe, parameterized Cypher queries
// Security: Prevents Cypher injection by using parameters for ALL values
// Reference: DEVELOPMENT_WORKFLOW.md §3.1 - Input validation
// Reference: NEO4J_MODERNIZATION_GUIDE.md §Phase 4 - Cypher injection prevention
type CypherBuilder struct {
	params  map[string]any
	counter int
}

// NewCypherBuilder creates a query builder
func NewCypherBuilder() *CypherBuilder {
	return &CypherBuilder{
		params:  make(map[string]any),
		counter: 0,
	}
}

// AddParam adds a parameter and returns its placeholder
func (b *CypherBuilder) AddParam(value any) string {
	paramName := fmt.Sprintf("p%d", b.counter)
	b.counter++
	b.params[paramName] = value
	return "$" + paramName
}

// Params returns all parameters for the query
func (b *CypherBuilder) Params() map[string]any {
	return b.params
}

// BuildMergeNode creates a safe MERGE query for node creation
// Uses parameters for ALL values, including properties
// Returns: (query string, error)
func (b *CypherBuilder) BuildMergeNode(label string, uniqueKey string, uniqueValue any, properties map[string]any) (string, error) {
	// Validate label (only allow alphanumeric + underscore)
	if !isValidIdentifier(label) {
		return "", fmt.Errorf("invalid node label: %s (must be alphanumeric + underscore)", label)
	}

	// Validate unique key
	if !isValidIdentifier(uniqueKey) {
		return "", fmt.Errorf("invalid unique key: %s (must be alphanumeric + underscore)", uniqueKey)
	}

	// Add unique value as parameter
	uniqueParam := b.AddParam(uniqueValue)

	// Build SET clauses using parameters
	setClauses := []string{}
	for key, value := range properties {
		if !isValidIdentifier(key) {
			return "", fmt.Errorf("invalid property key: %s (must be alphanumeric + underscore)", key)
		}
		paramName := b.AddParam(value)
		setClauses = append(setClauses, fmt.Sprintf("n.%s = %s", key, paramName))
	}

	setClause := strings.Join(setClauses, ", ")

	return fmt.Sprintf(
		"MERGE (n:%s {%s: %s}) SET %s RETURN id(n) as id",
		label,
		uniqueKey,
		uniqueParam,
		setClause,
	), nil
}

// BuildMergeEdge creates a safe MERGE query for edge creation
// Returns: (query string, error)
func (b *CypherBuilder) BuildMergeEdge(
	fromLabel, fromKey string, fromValue any,
	toLabel, toKey string, toValue any,
	edgeLabel string,
	properties map[string]any,
) (string, error) {
	// Validate all identifiers
	if !isValidIdentifier(fromLabel) {
		return "", fmt.Errorf("invalid from label: %s", fromLabel)
	}
	if !isValidIdentifier(fromKey) {
		return "", fmt.Errorf("invalid from key: %s", fromKey)
	}
	if !isValidIdentifier(toLabel) {
		return "", fmt.Errorf("invalid to label: %s", toLabel)
	}
	if !isValidIdentifier(toKey) {
		return "", fmt.Errorf("invalid to key: %s", toKey)
	}
	if !isValidIdentifier(edgeLabel) {
		return "", fmt.Errorf("invalid edge label: %s", edgeLabel)
	}

	// Add parameters for node matching
	fromParam := b.AddParam(fromValue)
	toParam := b.AddParam(toValue)

	// Build property SET clauses using parameters
	var propsStr string
	if len(properties) > 0 {
		propClauses := []string{}
		for key, value := range properties {
			if !isValidIdentifier(key) {
				return "", fmt.Errorf("invalid edge property key: %s", key)
			}
			paramName := b.AddParam(value)
			propClauses = append(propClauses, fmt.Sprintf("r.%s = %s", key, paramName))
		}
		propsStr = "SET " + strings.Join(propClauses, ", ")
	}

	return fmt.Sprintf(
		"MATCH (from:%s {%s: %s}) MATCH (to:%s {%s: %s}) MERGE (from)-[r:%s]->(to) %s RETURN from, to",
		fromLabel, fromKey, fromParam,
		toLabel, toKey, toParam,
		edgeLabel,
		propsStr,
	), nil
}

// isValidIdentifier validates that a string can be safely used as a Cypher identifier
// Only allows alphanumeric characters and underscores (prevents injection)
// Reference: Neo4j naming rules - https://neo4j.com/docs/cypher-manual/current/syntax/naming/
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	// Must start with letter or underscore, then alphanumeric or underscore
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, s)
	return matched
}
