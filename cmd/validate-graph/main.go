package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/graph"
)

// ValidationReport contains comprehensive Neo4j graph validation results
type ValidationReport struct {
	RepoID       int64
	RepoFullName string
	IsValid      bool
	Summary      DataSummary
	Completeness CompletenessMetrics
	Edges        EdgeMetrics
	Warnings     []string
	MissingData  []string
}

type DataSummary struct {
	// Node counts
	Files      int
	Commits    int
	PRs        int
	Issues     int
	Developers int

	// Composite property checks
	FilesWithRepoID      int
	CommitsWithRepoID    int
	PRsWithRepoID        int
	IssuesWithRepoID     int
	NodesWithRepoName    int
}

type EdgeMetrics struct {
	ModifiedEdges      int // Commit->File
	AuthoredEdges      int // Developer->Commit
	CreatedEdges       int // Developer->PR
	DependsOnEdges     int // File->File
	AssociatedWithEdges int // Issue->Commit, Issue->PR
	InPREdges          int // Commit->PR
}

type CompletenessMetrics struct {
	FilesWithRepoID   float64
	CommitsWithRepoID float64
	PRsWithRepoID     float64
	IssuesWithRepoID  float64
}

func main() {
	var repoID int64
	var jsonOutput bool

	flag.Int64Var(&repoID, "repo", 1, "Repository ID to validate")
	flag.BoolVar(&jsonOutput, "json", false, "Output as JSON")
	flag.Parse()

	ctx := context.Background()

	// Load config
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to Neo4j
	graphBackend, err := graph.NewNeo4jBackend(
		ctx,
		cfg.Neo4j.URI,
		cfg.Neo4j.User,
		cfg.Neo4j.Password,
		cfg.Neo4j.Database,
	)
	if err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	defer graphBackend.Close(ctx)

	// Validate graph
	report, err := validateGraph(ctx, graphBackend, repoID)
	if err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	// Print report
	printReport(report, jsonOutput)

	if !report.IsValid {
		os.Exit(1)
	}
}

func validateGraph(ctx context.Context, backend graph.Backend, repoID int64) (*ValidationReport, error) {
	report := &ValidationReport{
		RepoID:      repoID,
		IsValid:     true,
		Warnings:    make([]string, 0),
		MissingData: make([]string, 0),
	}

	// Get repo full name
	results, err := backend.QueryWithParams(ctx, `
		MATCH (c:Commit) WHERE c.repo_id = $repoId
		RETURN c.repo_full_name LIMIT 1
	`, map[string]interface{}{"repoId": repoID})
	if err == nil && len(results) > 0 {
		if name, ok := results[0]["c.repo_full_name"].(string); ok {
			report.RepoFullName = name
		}
	}

	// Validate node counts
	if err := validateNodes(ctx, backend, repoID, report); err != nil {
		return nil, err
	}

	// Validate composite properties
	if err := validateCompositeProperties(ctx, backend, repoID, report); err != nil {
		return nil, err
	}

	// Validate edges
	if err := validateEdges(ctx, backend, repoID, report); err != nil {
		return nil, err
	}

	// Validate constraints
	if err := validateConstraints(ctx, backend, report); err != nil {
		return nil, err
	}

	return report, nil
}

func validateNodes(ctx context.Context, backend graph.Backend, repoID int64, report *ValidationReport) error {
	// Count Files
	results, err := backend.QueryWithParams(ctx, `
		MATCH (f:File) WHERE f.repo_id = $repoId RETURN count(f) as count
	`, map[string]interface{}{"repoId": repoID})
	if err != nil {
		return fmt.Errorf("failed to count Files: %w", err)
	}
	if len(results) > 0 {
		if count, ok := results[0]["count"].(int64); ok {
			report.Summary.Files = int(count)
		}
	}
	if report.Summary.Files == 0 {
		report.MissingData = append(report.MissingData, "files")
		report.IsValid = false
	}

	// Count Commits
	results, err = backend.QueryWithParams(ctx, `
		MATCH (c:Commit) WHERE c.repo_id = $repoId RETURN count(c) as count
	`, map[string]interface{}{"repoId": repoID})
	if err != nil {
		return fmt.Errorf("failed to count Commits: %w", err)
	}
	if len(results) > 0 {
		if count, ok := results[0]["count"].(int64); ok {
			report.Summary.Commits = int(count)
		}
	}
	if report.Summary.Commits == 0 {
		report.MissingData = append(report.MissingData, "commits")
		report.IsValid = false
	}

	// Count PRs
	results, err = backend.QueryWithParams(ctx, `
		MATCH (p:PR) WHERE p.repo_id = $repoId RETURN count(p) as count
	`, map[string]interface{}{"repoId": repoID})
	if err != nil {
		return fmt.Errorf("failed to count PRs: %w", err)
	}
	if len(results) > 0 {
		if count, ok := results[0]["count"].(int64); ok {
			report.Summary.PRs = int(count)
		}
	}
	if report.Summary.PRs == 0 {
		report.Warnings = append(report.Warnings, "no PRs found")
	}

	// Count Issues
	results, err = backend.QueryWithParams(ctx, `
		MATCH (i:Issue) WHERE i.repo_id = $repoId RETURN count(i) as count
	`, map[string]interface{}{"repoId": repoID})
	if err != nil {
		return fmt.Errorf("failed to count Issues: %w", err)
	}
	if len(results) > 0 {
		if count, ok := results[0]["count"].(int64); ok {
			report.Summary.Issues = int(count)
		}
	}
	if report.Summary.Issues == 0 {
		report.Warnings = append(report.Warnings, "no issues found")
	}

	// Count Developers (global, no repo_id)
	results, err = backend.QueryWithParams(ctx, `
		MATCH (d:Developer) RETURN count(d) as count
	`, nil)
	if err != nil {
		return fmt.Errorf("failed to count Developers: %w", err)
	}
	if len(results) > 0 {
		if count, ok := results[0]["count"].(int64); ok {
			report.Summary.Developers = int(count)
		}
	}
	if report.Summary.Developers == 0 {
		report.Warnings = append(report.Warnings, "no developers found")
	}

	return nil
}

func validateCompositeProperties(ctx context.Context, backend graph.Backend, repoID int64, report *ValidationReport) error {
	// Check Files with repo_id
	results, err := backend.QueryWithParams(ctx, `
		MATCH (f:File) WHERE f.repo_id = $repoId AND f.repo_id IS NOT NULL
		RETURN count(f) as count
	`, map[string]interface{}{"repoId": repoID})
	if err != nil {
		return fmt.Errorf("failed to check File repo_id: %w", err)
	}
	if len(results) > 0 {
		if count, ok := results[0]["count"].(int64); ok {
			report.Summary.FilesWithRepoID = int(count)
		}
	}
	if report.Summary.Files > 0 {
		report.Completeness.FilesWithRepoID = float64(report.Summary.FilesWithRepoID) / float64(report.Summary.Files) * 100
		if report.Completeness.FilesWithRepoID < 100 {
			report.Warnings = append(report.Warnings, fmt.Sprintf("Only %.1f%% of Files have repo_id", report.Completeness.FilesWithRepoID))
		}
	}

	// Check Commits with repo_id
	results, err = backend.QueryWithParams(ctx, `
		MATCH (c:Commit) WHERE c.repo_id = $repoId AND c.repo_id IS NOT NULL
		RETURN count(c) as count
	`, map[string]interface{}{"repoId": repoID})
	if err != nil {
		return fmt.Errorf("failed to check Commit repo_id: %w", err)
	}
	if len(results) > 0 {
		if count, ok := results[0]["count"].(int64); ok {
			report.Summary.CommitsWithRepoID = int(count)
		}
	}
	if report.Summary.Commits > 0 {
		report.Completeness.CommitsWithRepoID = float64(report.Summary.CommitsWithRepoID) / float64(report.Summary.Commits) * 100
		if report.Completeness.CommitsWithRepoID < 100 {
			report.Warnings = append(report.Warnings, fmt.Sprintf("Only %.1f%% of Commits have repo_id", report.Completeness.CommitsWithRepoID))
		}
	}

	// Check PRs with repo_id
	if report.Summary.PRs > 0 {
		results, err = backend.QueryWithParams(ctx, `
			MATCH (p:PR) WHERE p.repo_id = $repoId AND p.repo_id IS NOT NULL
			RETURN count(p) as count
		`, map[string]interface{}{"repoId": repoID})
		if err != nil {
			return fmt.Errorf("failed to check PR repo_id: %w", err)
		}
		if len(results) > 0 {
			if count, ok := results[0]["count"].(int64); ok {
				report.Summary.PRsWithRepoID = int(count)
			}
		}
		report.Completeness.PRsWithRepoID = float64(report.Summary.PRsWithRepoID) / float64(report.Summary.PRs) * 100
		if report.Completeness.PRsWithRepoID < 100 {
			report.Warnings = append(report.Warnings, fmt.Sprintf("Only %.1f%% of PRs have repo_id", report.Completeness.PRsWithRepoID))
		}
	}

	// Check Issues with repo_id
	if report.Summary.Issues > 0 {
		results, err = backend.QueryWithParams(ctx, `
			MATCH (i:Issue) WHERE i.repo_id = $repoId AND i.repo_id IS NOT NULL
			RETURN count(i) as count
		`, map[string]interface{}{"repoId": repoID})
		if err != nil {
			return fmt.Errorf("failed to check Issue repo_id: %w", err)
		}
		if len(results) > 0 {
			if count, ok := results[0]["count"].(int64); ok {
				report.Summary.IssuesWithRepoID = int(count)
			}
		}
		report.Completeness.IssuesWithRepoID = float64(report.Summary.IssuesWithRepoID) / float64(report.Summary.Issues) * 100
		if report.Completeness.IssuesWithRepoID < 100 {
			report.Warnings = append(report.Warnings, fmt.Sprintf("Only %.1f%% of Issues have repo_id", report.Completeness.IssuesWithRepoID))
		}
	}

	// Check nodes with repo_full_name
	results, err = backend.QueryWithParams(ctx, `
		MATCH (n) WHERE n.repo_id = $repoId AND n.repo_full_name IS NOT NULL
		RETURN count(n) as count
	`, map[string]interface{}{"repoId": repoID})
	if err != nil {
		return fmt.Errorf("failed to check repo_full_name: %w", err)
	}
	if len(results) > 0 {
		if count, ok := results[0]["count"].(int64); ok {
			report.Summary.NodesWithRepoName = int(count)
		}
	}

	return nil
}

func validateEdges(ctx context.Context, backend graph.Backend, repoID int64, report *ValidationReport) error {
	// MODIFIED edges (Commit->File)
	results, err := backend.QueryWithParams(ctx, `
		MATCH (c:Commit)-[r:MODIFIED]->(f:File)
		WHERE c.repo_id = $repoId AND f.repo_id = $repoId
		RETURN count(r) as count
	`, map[string]interface{}{"repoId": repoID})
	if err != nil {
		return fmt.Errorf("failed to count MODIFIED edges: %w", err)
	}
	if len(results) > 0 {
		if count, ok := results[0]["count"].(int64); ok {
			report.Edges.ModifiedEdges = int(count)
		}
	}
	if report.Edges.ModifiedEdges == 0 {
		report.MissingData = append(report.MissingData, "MODIFIED edges")
		report.IsValid = false
	}

	// AUTHORED edges (Developer->Commit)
	results, err = backend.QueryWithParams(ctx, `
		MATCH (d:Developer)-[r:AUTHORED]->(c:Commit)
		WHERE c.repo_id = $repoId
		RETURN count(r) as count
	`, map[string]interface{}{"repoId": repoID})
	if err != nil {
		return fmt.Errorf("failed to count AUTHORED edges: %w", err)
	}
	if len(results) > 0 {
		if count, ok := results[0]["count"].(int64); ok {
			report.Edges.AuthoredEdges = int(count)
		}
	}
	if report.Edges.AuthoredEdges == 0 {
		report.MissingData = append(report.MissingData, "AUTHORED edges")
		report.IsValid = false
	}

	// CREATED edges (Developer->PR)
	results, err = backend.QueryWithParams(ctx, `
		MATCH (d:Developer)-[r:CREATED]->(p:PR)
		WHERE p.repo_id = $repoId
		RETURN count(r) as count
	`, map[string]interface{}{"repoId": repoID})
	if err != nil {
		return fmt.Errorf("failed to count CREATED edges: %w", err)
	}
	if len(results) > 0 {
		if count, ok := results[0]["count"].(int64); ok {
			report.Edges.CreatedEdges = int(count)
		}
	}
	if report.Edges.CreatedEdges == 0 {
		report.Warnings = append(report.Warnings, "no CREATED edges (Developer->PR)")
	}

	// DEPENDS_ON edges (File->File)
	results, err = backend.QueryWithParams(ctx, `
		MATCH (f1:File)-[r:DEPENDS_ON]->(f2:File)
		WHERE f1.repo_id = $repoId AND f2.repo_id = $repoId
		RETURN count(r) as count
	`, map[string]interface{}{"repoId": repoID})
	if err != nil {
		return fmt.Errorf("failed to count DEPENDS_ON edges: %w", err)
	}
	if len(results) > 0 {
		if count, ok := results[0]["count"].(int64); ok {
			report.Edges.DependsOnEdges = int(count)
		}
	}
	if report.Edges.DependsOnEdges == 0 {
		report.Warnings = append(report.Warnings, "no DEPENDS_ON edges (dependency analysis not run or no dependencies found)")
	}

	// ASSOCIATED_WITH edges (Issue/PR -> Commit)
	results, err = backend.QueryWithParams(ctx, `
		MATCH (n)-[r:ASSOCIATED_WITH]->(c:Commit)
		WHERE n.repo_id = $repoId AND c.repo_id = $repoId
		RETURN count(r) as count
	`, map[string]interface{}{"repoId": repoID})
	if err != nil {
		return fmt.Errorf("failed to count ASSOCIATED_WITH edges: %w", err)
	}
	if len(results) > 0 {
		if count, ok := results[0]["count"].(int64); ok {
			report.Edges.AssociatedWithEdges = int(count)
		}
	}

	// IN_PR edges (Commit->PR)
	results, err = backend.QueryWithParams(ctx, `
		MATCH (c:Commit)-[r:IN_PR]->(p:PR)
		WHERE c.repo_id = $repoId AND p.repo_id = $repoId
		RETURN count(r) as count
	`, map[string]interface{}{"repoId": repoID})
	if err != nil {
		return fmt.Errorf("failed to count IN_PR edges: %w", err)
	}
	if len(results) > 0 {
		if count, ok := results[0]["count"].(int64); ok {
			report.Edges.InPREdges = int(count)
		}
	}

	return nil
}

func validateConstraints(ctx context.Context, backend graph.Backend, report *ValidationReport) error {
	// Check for composite unique constraints
	requiredConstraints := []string{
		"file_repo_path_unique",
		"commit_repo_sha_unique",
		"pr_repo_number_unique",
		"issue_repo_number_unique",
		"developer_email_unique",
	}

	results, err := backend.QueryWithParams(ctx, `SHOW CONSTRAINTS`, nil)
	if err != nil {
		return fmt.Errorf("failed to check constraints: %w", err)
	}

	foundConstraints := make(map[string]bool)
	for _, row := range results {
		if name, ok := row["name"].(string); ok {
			foundConstraints[name] = true
		}
	}

	for _, constraint := range requiredConstraints {
		if !foundConstraints[constraint] {
			report.Warnings = append(report.Warnings, fmt.Sprintf("missing constraint: %s", constraint))
			report.IsValid = false
		}
	}

	return nil
}

func printReport(report *ValidationReport, jsonOutput bool) {
	if jsonOutput {
		// TODO: Implement JSON output
		return
	}

	fmt.Println("================================================================================")
	fmt.Printf("Neo4j Graph Validation Report: %s (repo_id=%d)\n", report.RepoFullName, report.RepoID)
	fmt.Println("================================================================================")

	// Status
	if report.IsValid {
		fmt.Println("✅ Status: VALID - Graph is complete and ready for analysis")
	} else {
		fmt.Println("❌ Status: INVALID - Critical data missing")
	}
	fmt.Println()

	// Node summary
	fmt.Println("📊 Node Summary:")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────")
	fmt.Printf("  Files:                   %d (%.1f%% with repo_id)\n",
		report.Summary.Files, report.Completeness.FilesWithRepoID)
	fmt.Printf("  Commits:                 %d (%.1f%% with repo_id)\n",
		report.Summary.Commits, report.Completeness.CommitsWithRepoID)
	fmt.Printf("  Pull Requests:           %d (%.1f%% with repo_id)\n",
		report.Summary.PRs, report.Completeness.PRsWithRepoID)
	fmt.Printf("  Issues:                  %d (%.1f%% with repo_id)\n",
		report.Summary.Issues, report.Completeness.IssuesWithRepoID)
	fmt.Printf("  Developers:              %d (global)\n", report.Summary.Developers)
	fmt.Printf("  Nodes with repo_full_name: %d\n", report.Summary.NodesWithRepoName)
	fmt.Println()

	// Edge summary
	fmt.Println("🔗 Edge Summary:")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────")
	fmt.Printf("  MODIFIED (Commit->File):        %d\n", report.Edges.ModifiedEdges)
	fmt.Printf("  AUTHORED (Developer->Commit):   %d\n", report.Edges.AuthoredEdges)
	fmt.Printf("  CREATED (Developer->PR):        %d\n", report.Edges.CreatedEdges)
	fmt.Printf("  IN_PR (Commit->PR):             %d\n", report.Edges.InPREdges)
	fmt.Printf("  ASSOCIATED_WITH (Issue/PR->Commit): %d\n", report.Edges.AssociatedWithEdges)
	fmt.Printf("  DEPENDS_ON (File->File):        %d\n", report.Edges.DependsOnEdges)
	fmt.Println()

	// Warnings
	if len(report.Warnings) > 0 {
		fmt.Println("⚠️  Warnings:")
		fmt.Println("────────────────────────────────────────────────────────────────────────────────")
		for _, warning := range report.Warnings {
			fmt.Printf("  • %s\n", warning)
		}
		fmt.Println()
	}

	// Missing data
	if len(report.MissingData) > 0 {
		fmt.Println("❌ Missing Critical Data:")
		fmt.Println("────────────────────────────────────────────────────────────────────────────────")
		for _, missing := range report.MissingData {
			fmt.Printf("  • %s\n", missing)
		}
		fmt.Println()
	}

	// Recommendation
	fmt.Println("💡 Recommendation:")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────")
	if report.IsValid {
		fmt.Println("  ✅ Graph is ready for risk analysis!")
		fmt.Printf("  → You can safely run: crisk check <file>\n")
	} else {
		fmt.Println("  ❌ Graph has critical issues. Please rebuild:")
		fmt.Printf("  → Run: crisk init --days 365\n")
	}
	fmt.Println("================================================================================")
}
