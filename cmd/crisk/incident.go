package main

import (
	"context"
	"fmt"
	"time"

	"github.com/coderisk/coderisk-go/internal/graph"
	"github.com/coderisk/coderisk-go/internal/incidents"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/cobra"
)

var (
	incidentSeverity  string
	incidentResolved  bool
	incidentRootCause string
	incidentImpact    string
	linkLineNumber    int
	linkFunction      string
	searchLimit       int
)

// Incident management commands
var incidentCmd = &cobra.Command{
	Use:   "incident",
	Short: "Manage incidents for risk analysis",
	Long:  `Create, link, and search production incidents to improve risk predictions.`,
}

var createIncidentCmd = &cobra.Command{
	Use:   "create [title] [description]",
	Short: "Create a new incident",
	Long:  `Create a new production incident or bug report for tracking.`,
	Args:  cobra.MinimumNArgs(2),
	RunE:  runCreateIncident,
}

var linkIncidentCmd = &cobra.Command{
	Use:   "link [incident-id] [file-path]",
	Short: "Link an incident to a file",
	Long:  `Create a manual link between an incident and the file that caused it.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runLinkIncident,
}

var unlinkIncidentCmd = &cobra.Command{
	Use:   "unlink [incident-id] [file-path]",
	Short: "Remove incident link from a file",
	Long:  `Remove the link between an incident and a file.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runUnlinkIncident,
}

var searchIncidentsCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search incidents using full-text search",
	Long:  `Search incidents using BM25-style full-text search.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runSearchIncidents,
}

var listIncidentsCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent incidents",
	Long:  `List recent incidents, optionally filtered by severity.`,
	RunE:  runListIncidents,
}

var incidentStatsCmd = &cobra.Command{
	Use:   "stats [file-path]",
	Short: "Show incident statistics for a file",
	Long:  `Display aggregated incident statistics for a specific file.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runIncidentStats,
}

func init() {
	// Create incident command
	createIncidentCmd.Flags().StringVar(&incidentSeverity, "severity", "medium", "Severity level (critical, high, medium, low)")
	createIncidentCmd.Flags().BoolVar(&incidentResolved, "resolved", false, "Mark incident as resolved")
	createIncidentCmd.Flags().StringVar(&incidentRootCause, "root-cause", "", "Root cause description")
	createIncidentCmd.Flags().StringVar(&incidentImpact, "impact", "", "Impact description")

	// Link incident command
	linkIncidentCmd.Flags().IntVar(&linkLineNumber, "line", 0, "Specific line number (0 = entire file)")
	linkIncidentCmd.Flags().StringVar(&linkFunction, "function", "", "Function name that caused the incident")

	// Search incidents command
	searchIncidentsCmd.Flags().IntVar(&searchLimit, "limit", 10, "Maximum number of results")

	// List incidents command
	listIncidentsCmd.Flags().StringVar(&incidentSeverity, "severity", "", "Filter by severity (critical, high, medium, low)")
	listIncidentsCmd.Flags().IntVar(&searchLimit, "limit", 20, "Maximum number of results")

	// Add subcommands
	incidentCmd.AddCommand(createIncidentCmd)
	incidentCmd.AddCommand(linkIncidentCmd)
	incidentCmd.AddCommand(unlinkIncidentCmd)
	incidentCmd.AddCommand(searchIncidentsCmd)
	incidentCmd.AddCommand(listIncidentsCmd)
	incidentCmd.AddCommand(incidentStatsCmd)
}

func runCreateIncident(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	title := args[0]
	description := ""
	if len(args) > 1 {
		for i := 1; i < len(args); i++ {
			if i > 1 {
				description += " "
			}
			description += args[i]
		}
	}

	// Get database connection
	db, err := getPostgresDB()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	incDB := incidents.NewDatabase(db)

	// Validate severity
	severity := incidents.Severity(incidentSeverity)
	if !severity.Validate() {
		return fmt.Errorf("invalid severity: %s (must be: critical, high, medium, low)", incidentSeverity)
	}

	// Create incident
	incident := &incidents.Incident{
		Title:       title,
		Description: description,
		Severity:    severity,
		OccurredAt:  time.Now(),
		RootCause:   incidentRootCause,
		Impact:      incidentImpact,
	}

	if incidentResolved {
		now := time.Now()
		incident.ResolvedAt = &now
	}

	if err := incDB.CreateIncident(ctx, incident); err != nil {
		return fmt.Errorf("failed to create incident: %w", err)
	}

	fmt.Printf("✅ Created incident: %s\n", incident.ID)
	fmt.Printf("   Title: %s\n", incident.Title)
	fmt.Printf("   Severity: %s\n", incident.Severity)
	fmt.Printf("   Occurred: %s\n", incident.OccurredAt.Format("2006-01-02 15:04:05"))

	return nil
}

func runLinkIncident(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	incidentID := args[0]
	filePath := args[1]

	// Get database connection
	db, err := getPostgresDB()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Get graph client
	graphClient, err := getGraphBackend()
	if err != nil {
		return fmt.Errorf("failed to connect to graph database: %w", err)
	}
	defer graphClient.Close()

	// Create linker
	incDB := incidents.NewDatabase(db)
	linker := incidents.NewLinker(incDB, &graphClientAdapter{backend: graphClient})

	// Create link
	if err := linker.LinkIncident(ctx, incidentID, filePath, linkLineNumber, linkFunction); err != nil {
		return fmt.Errorf("failed to link incident: %w", err)
	}

	fmt.Printf("✅ Linked incident %s to %s\n", incidentID, filePath)
	if linkLineNumber > 0 {
		fmt.Printf("   Line: %d\n", linkLineNumber)
	}
	if linkFunction != "" {
		fmt.Printf("   Function: %s\n", linkFunction)
	}

	return nil
}

func runUnlinkIncident(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	incidentID := args[0]
	filePath := args[1]

	// Get database connection
	db, err := getPostgresDB()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Get graph client
	graphClient, err := getGraphBackend()
	if err != nil {
		return fmt.Errorf("failed to connect to graph database: %w", err)
	}
	defer graphClient.Close()

	// Create linker
	incDB := incidents.NewDatabase(db)
	linker := incidents.NewLinker(incDB, &graphClientAdapter{backend: graphClient})

	// Remove link
	if err := linker.UnlinkIncident(ctx, incidentID, filePath); err != nil {
		return fmt.Errorf("failed to unlink incident: %w", err)
	}

	fmt.Printf("✅ Removed link between incident %s and %s\n", incidentID, filePath)

	return nil
}

func runSearchIncidents(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	query := args[0]

	// Get database connection
	db, err := getPostgresDB()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	incDB := incidents.NewDatabase(db)

	// Search incidents
	results, err := incDB.SearchIncidents(ctx, query, searchLimit)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Printf("No incidents found matching '%s'\n", query)
		return nil
	}

	fmt.Printf("🔍 Found %d incident(s) matching '%s':\n\n", len(results), query)

	for i, result := range results {
		fmt.Printf("%d. [%s] %s (ID: %s)\n", i+1, result.Incident.Severity, result.Incident.Title, result.Incident.ID)
		fmt.Printf("   Occurred: %s\n", result.Incident.OccurredAt.Format("2006-01-02"))
		fmt.Printf("   Relevance: %s (score: %.3f)\n", result.Relevance, result.Rank)
		if result.Incident.RootCause != "" {
			fmt.Printf("   Root Cause: %s\n", truncate(result.Incident.RootCause, 80))
		}
		fmt.Println()
	}

	return nil
}

func runListIncidents(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get database connection
	db, err := getPostgresDB()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	incDB := incidents.NewDatabase(db)

	// List incidents
	severity := incidents.Severity(incidentSeverity)
	if incidentSeverity != "" && !severity.Validate() {
		return fmt.Errorf("invalid severity: %s (must be: critical, high, medium, low)", incidentSeverity)
	}

	incidentsList, err := incDB.ListIncidents(ctx, severity, searchLimit)
	if err != nil {
		return fmt.Errorf("failed to list incidents: %w", err)
	}

	if len(incidentsList) == 0 {
		if incidentSeverity != "" {
			fmt.Printf("No %s severity incidents found\n", incidentSeverity)
		} else {
			fmt.Printf("No incidents found\n")
		}
		return nil
	}

	if incidentSeverity != "" {
		fmt.Printf("📋 Recent %s incidents:\n\n", incidentSeverity)
	} else {
		fmt.Printf("📋 Recent incidents:\n\n")
	}

	for i, inc := range incidentsList {
		resolvedStr := ""
		if inc.ResolvedAt != nil {
			resolvedStr = " ✅"
		}

		fmt.Printf("%d. [%s] %s%s\n", i+1, inc.Severity, inc.Title, resolvedStr)
		fmt.Printf("   ID: %s\n", inc.ID)
		fmt.Printf("   Occurred: %s\n", inc.OccurredAt.Format("2006-01-02 15:04"))

		if inc.ResolvedAt != nil {
			resolution := inc.ResolvedAt.Sub(inc.OccurredAt)
			fmt.Printf("   Resolved: %s (took %s)\n", inc.ResolvedAt.Format("2006-01-02 15:04"), formatResolution(resolution))
		}

		if len(inc.LinkedFiles) > 0 {
			fmt.Printf("   Linked files: %d\n", len(inc.LinkedFiles))
		}

		fmt.Println()
	}

	return nil
}

func runIncidentStats(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	filePath := args[0]

	// Get database connection
	db, err := getPostgresDB()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	incDB := incidents.NewDatabase(db)

	// Get stats
	stats, err := incDB.GetIncidentStats(ctx, filePath)
	if err != nil {
		return fmt.Errorf("failed to get stats: %w", err)
	}

	fmt.Printf("📊 Incident Statistics for: %s\n\n", filePath)

	if stats.TotalIncidents == 0 {
		fmt.Printf("No incidents linked to this file.\n")
		return nil
	}

	fmt.Printf("Total incidents: %d\n", stats.TotalIncidents)
	fmt.Printf("Last 30 days: %d\n", stats.Last30Days)
	fmt.Printf("Last 90 days: %d\n", stats.Last90Days)
	fmt.Printf("\nBy severity:\n")
	fmt.Printf("  Critical: %d\n", stats.CriticalCount)
	fmt.Printf("  High: %d\n", stats.HighCount)

	if stats.LastIncident != nil {
		fmt.Printf("\nLast incident: %s (%s ago)\n",
			stats.LastIncident.Format("2006-01-02"),
			formatDuration(time.Since(*stats.LastIncident)))
	}

	if stats.AvgResolution > 0 {
		fmt.Printf("Average resolution time: %s\n", formatResolution(stats.AvgResolution))
	}

	return nil
}

// Helper functions

func getPostgresDB() (*sqlx.DB, error) {
	// Use hardcoded DSN for now (will be configurable in future)
	dsn := "postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable"

	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func getGraphBackend() (graph.Backend, error) {
	// Use hardcoded Neo4j configuration for now (will be configurable in future)
	uri := "bolt://localhost:7688" // Note: Uses port 7688 based on docker ps output
	username := "neo4j"
	password := "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"

	ctx := context.Background()
	backend, err := graph.NewNeo4jBackend(ctx, uri, username, password)
	if err != nil {
		return nil, err
	}

	return backend, nil
}

// Adapter to convert graph.Backend to incidents.GraphClient
type graphClientAdapter struct {
	backend graph.Backend
}

func (a *graphClientAdapter) CreateNode(node incidents.GraphNode) (string, error) {
	graphNode := graph.GraphNode{
		Label:      node.Label,
		ID:         node.ID,
		Properties: node.Properties,
	}
	return a.backend.CreateNode(graphNode)
}

func (a *graphClientAdapter) CreateEdge(edge incidents.GraphEdge) error {
	graphEdge := graph.GraphEdge{
		Label:      edge.Label,
		From:       edge.From,
		To:         edge.To,
		Properties: edge.Properties,
	}
	return a.backend.CreateEdge(graphEdge)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatResolution(d time.Duration) string {
	hours := int(d.Hours())
	if hours < 24 {
		return fmt.Sprintf("%dh", hours)
	}
	days := hours / 24
	if days < 7 {
		return fmt.Sprintf("%dd", days)
	}
	weeks := days / 7
	return fmt.Sprintf("%dw", weeks)
}
