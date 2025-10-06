package incidents

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// GraphClient interface for Neo4j operations (implemented by graph.Backend)
type GraphClient interface {
	CreateNode(node GraphNode) (string, error)
	CreateEdge(edge GraphEdge) error
}

// GraphNode represents a node in the graph
type GraphNode struct {
	Label      string
	ID         string
	Properties map[string]interface{}
}

// GraphEdge represents an edge in the graph
type GraphEdge struct {
	Label      string
	From       string
	To         string
	Properties map[string]interface{}
}

// Linker handles manual incident-to-file linking
type Linker struct {
	db    *Database
	graph GraphClient
}

// NewLinker creates a new incident linker
func NewLinker(db *Database, graph GraphClient) *Linker {
	return &Linker{
		db:    db,
		graph: graph,
	}
}

// LinkIncident creates link between incident and file (CLI command)
func (l *Linker) LinkIncident(ctx context.Context, incidentID string, filePath string, lineNumber int, function string) error {
	// Parse incident ID
	id, err := uuid.Parse(incidentID)
	if err != nil {
		return fmt.Errorf("invalid incident ID: %w", err)
	}

	// Validate incident exists
	incident, err := l.db.GetIncident(ctx, id)
	if err != nil {
		return fmt.Errorf("incident not found: %w", err)
	}

	// Create link in PostgreSQL (incident_files table)
	link := &IncidentFile{
		IncidentID:     id,
		FilePath:       filePath,
		LineNumber:     lineNumber,
		BlamedFunction: function,
		Confidence:     1.0, // Manual link = 100% confidence
	}

	if err := l.db.LinkIncidentToFile(ctx, link); err != nil {
		return fmt.Errorf("create database link: %w", err)
	}

	// Create Incident node in Neo4j if it doesn't exist
	incidentNode := GraphNode{
		Label: "Incident",
		ID:    incident.ID.String(),
		Properties: map[string]interface{}{
			"id":         incident.ID.String(),
			"title":      incident.Title,
			"severity":   string(incident.Severity),
			"occurred_at": incident.OccurredAt.Unix(),
		},
	}

	if incident.ResolvedAt != nil {
		incidentNode.Properties["resolved_at"] = incident.ResolvedAt.Unix()
	}
	if incident.RootCause != "" {
		incidentNode.Properties["root_cause"] = incident.RootCause
	}

	// Create node (idempotent operation - will be ignored if exists)
	if _, err := l.graph.CreateNode(incidentNode); err != nil {
		return fmt.Errorf("create incident node: %w", err)
	}

	// Create CAUSED_BY edge in Neo4j: (Incident)-[:CAUSED_BY]->(File)
	edgeProps := map[string]interface{}{
		"confidence": link.Confidence,
	}
	if lineNumber > 0 {
		edgeProps["line_number"] = lineNumber
	}
	if function != "" {
		edgeProps["blamed_function"] = function
	}

	edge := GraphEdge{
		Label:      "CAUSED_BY",
		From:       incident.ID.String(),
		To:         filePath,
		Properties: edgeProps,
	}

	if err := l.graph.CreateEdge(edge); err != nil {
		return fmt.Errorf("create CAUSED_BY edge: %w", err)
	}

	return nil
}

// UnlinkIncident removes link between incident and file
func (l *Linker) UnlinkIncident(ctx context.Context, incidentID string, filePath string) error {
	// Parse incident ID
	id, err := uuid.Parse(incidentID)
	if err != nil {
		return fmt.Errorf("invalid incident ID: %w", err)
	}

	// Delete from PostgreSQL
	if err := l.db.UnlinkIncidentFromFile(ctx, id, filePath); err != nil {
		return fmt.Errorf("remove database link: %w", err)
	}

	// Note: We don't remove the Neo4j edge here as it requires a more complex query
	// The edge will be cleaned up during next full graph rebuild
	// For manual cleanup, use: MATCH (i:Incident {id: $id})-[r:CAUSED_BY]->(f:File {path: $path}) DELETE r

	return nil
}

// SuggestLinks uses BM25 search to suggest file links for an incident
func (l *Linker) SuggestLinks(ctx context.Context, incidentID string, threshold float64) ([]SuggestionResult, error) {
	// Parse incident ID
	id, err := uuid.Parse(incidentID)
	if err != nil {
		return nil, fmt.Errorf("invalid incident ID: %w", err)
	}

	// Get incident
	incident, err := l.db.GetIncident(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get incident: %w", err)
	}

	// Extract file paths from description and root cause
	filePaths := l.extractFilePaths(incident.Description + " " + incident.RootCause)

	var suggestions []SuggestionResult

	for _, filePath := range filePaths {
		// Simple confidence based on where it was mentioned
		confidence := 0.6 // Default confidence

		// Higher confidence if in root cause
		if strings.Contains(incident.RootCause, filePath) {
			confidence = 0.8
		}

		// Even higher if mentioned multiple times
		mentions := strings.Count(incident.Description+" "+incident.RootCause, filePath)
		if mentions > 1 {
			confidence = 0.9
		}

		if confidence >= threshold {
			suggestions = append(suggestions, SuggestionResult{
				FilePath:   filePath,
				Confidence: confidence,
				Reason:     fmt.Sprintf("Mentioned %d time(s) in incident", mentions),
			})
		}
	}

	return suggestions, nil
}

// extractFilePaths extracts file paths from text using simple heuristics
func (l *Linker) extractFilePaths(text string) []string {
	var paths []string
	seen := make(map[string]bool)

	// Common file extensions to look for
	extensions := []string{".go", ".py", ".js", ".ts", ".java", ".rb", ".php", ".c", ".cpp", ".h", ".rs", ".sql"}

	words := strings.Fields(text)
	for _, word := range words {
		// Clean up punctuation
		word = strings.Trim(word, ".,;:()[]{}\"'")

		// Check if it looks like a file path
		for _, ext := range extensions {
			if strings.HasSuffix(word, ext) {
				if !seen[word] {
					paths = append(paths, word)
					seen[word] = true
				}
				break
			}
		}
	}

	return paths
}

// SuggestionResult represents a suggested file link
type SuggestionResult struct {
	FilePath   string
	Confidence float64
	Reason     string
}

// CreateIncidentNode creates an Incident node in Neo4j
func (l *Linker) CreateIncidentNode(ctx context.Context, incident *Incident) error {
	node := GraphNode{
		Label: "Incident",
		ID:    incident.ID.String(),
		Properties: map[string]interface{}{
			"id":         incident.ID.String(),
			"title":      incident.Title,
			"severity":   string(incident.Severity),
			"occurred_at": incident.OccurredAt.Unix(),
		},
	}

	if incident.ResolvedAt != nil {
		node.Properties["resolved_at"] = incident.ResolvedAt.Unix()
	}
	if incident.RootCause != "" {
		node.Properties["root_cause"] = incident.RootCause
	}
	if incident.Impact != "" {
		node.Properties["impact"] = incident.Impact
	}

	_, err := l.graph.CreateNode(node)
	if err != nil {
		return fmt.Errorf("create incident node: %w", err)
	}

	return nil
}
