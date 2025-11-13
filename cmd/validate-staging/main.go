package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/database"
)

// ValidationReport contains the validation results for a repository
type ValidationReport struct {
	RepoID       int64             `json:"repo_id"`
	RepoName     string            `json:"repo_name"`
	IsValid      bool              `json:"is_valid"`
	Summary      DataSummary       `json:"summary"`
	MissingData  []string          `json:"missing_data,omitempty"`
	Warnings     []string          `json:"warnings,omitempty"`
	Completeness CompletionMetrics `json:"completeness"`
}

// DataSummary contains counts of all data types
type DataSummary struct {
	Commits              int `json:"commits"`
	CommitsWithFileData  int `json:"commits_with_file_data"`
	Issues               int `json:"issues"`
	IssueComments        int `json:"issue_comments"`
	PullRequests         int `json:"pull_requests"`
	PRFiles              int `json:"pr_files"`
	PRFilesWithPatch     int `json:"pr_files_with_patch"`
	Contributors         int `json:"contributors"`
	IssueTimelineEvents  int `json:"issue_timeline_events"`
}

// CompletionMetrics tracks data completeness percentages
type CompletionMetrics struct {
	CommitsWithFiles float64 `json:"commits_with_files_pct"`
	PRFilesWithPatch float64 `json:"pr_files_with_patch_pct"`
}

func main() {
	repoName := flag.String("repo", "", "Repository name (e.g., 'omnara' or 'supabase')")
	jsonOutput := flag.Bool("json", false, "Output results as JSON")
	flag.Parse()

	if *repoName == "" {
		fmt.Fprintln(os.Stderr, "Usage: validate-staging -repo <repo_name> [-json]")
		fmt.Fprintln(os.Stderr, "Example: validate-staging -repo omnara")
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Connect to PostgreSQL
	ctx := context.Background()
	stagingDB, err := database.NewStagingClient(
		ctx,
		cfg.Storage.PostgresHost,
		cfg.Storage.PostgresPort,
		cfg.Storage.PostgresDB,
		cfg.Storage.PostgresUser,
		cfg.Storage.PostgresPassword,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "PostgreSQL connection failed: %v\n", err)
		os.Exit(1)
	}
	defer stagingDB.Close()

	// Validate the repository
	report, err := validateRepository(ctx, stagingDB, *repoName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Validation failed: %v\n", err)
		os.Exit(1)
	}

	// Output results
	if *jsonOutput {
		jsonBytes, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "JSON marshaling failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonBytes))
	} else {
		printReport(report)
	}

	// Exit code based on validation result
	if !report.IsValid {
		os.Exit(1)
	}
}

func validateRepository(ctx context.Context, db *database.StagingClient, repoName string) (*ValidationReport, error) {
	// Get repository ID
	query := `SELECT id, owner, name FROM github_repositories WHERE name = $1`
	rows, err := db.Query(ctx, query, repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to query repository: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("repository '%s' not found in database", repoName)
	}

	var repoID int64
	var owner, name string
	if err := rows.Scan(&repoID, &owner, &name); err != nil {
		return nil, fmt.Errorf("failed to scan repository: %w", err)
	}
	rows.Close()

	fullName := fmt.Sprintf("%s/%s", owner, name)

	// Initialize report
	report := &ValidationReport{
		RepoID:      repoID,
		RepoName:    fullName,
		IsValid:     true,
		MissingData: []string{},
		Warnings:    []string{},
	}

	// Check commits
	query = `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN raw_data->'files' IS NOT NULL THEN 1 END) as with_files
		FROM github_commits
		WHERE repo_id = $1
	`
	var commitTotal, commitWithFiles int
	err = db.QueryRow(ctx, query, repoID).Scan(&commitTotal, &commitWithFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to query commits: %w", err)
	}
	report.Summary.Commits = commitTotal
	report.Summary.CommitsWithFileData = commitWithFiles

	if commitTotal == 0 {
		report.MissingData = append(report.MissingData, "commits")
		report.IsValid = false
	} else if commitWithFiles < commitTotal {
		pct := float64(commitWithFiles) / float64(commitTotal) * 100
		report.Completeness.CommitsWithFiles = pct
		if pct < 95.0 {
			report.Warnings = append(report.Warnings, fmt.Sprintf("Only %.1f%% of commits have file data", pct))
		}
	} else {
		report.Completeness.CommitsWithFiles = 100.0
	}

	// Check issues
	query = `SELECT COUNT(*) FROM github_issues WHERE repo_id = $1`
	err = db.QueryRow(ctx, query, repoID).Scan(&report.Summary.Issues)
	if err != nil {
		return nil, fmt.Errorf("failed to query issues: %w", err)
	}
	if report.Summary.Issues == 0 {
		report.Warnings = append(report.Warnings, "no issues found (repository may have no issues)")
	}

	// Check issue comments
	query = `SELECT COUNT(*) FROM github_issue_comments WHERE repo_id = $1`
	err = db.QueryRow(ctx, query, repoID).Scan(&report.Summary.IssueComments)
	if err != nil {
		return nil, fmt.Errorf("failed to query issue comments: %w", err)
	}

	// Check pull requests
	query = `SELECT COUNT(*) FROM github_pull_requests WHERE repo_id = $1`
	err = db.QueryRow(ctx, query, repoID).Scan(&report.Summary.PullRequests)
	if err != nil {
		return nil, fmt.Errorf("failed to query pull requests: %w", err)
	}
	if report.Summary.PullRequests == 0 {
		report.MissingData = append(report.MissingData, "pull_requests")
		report.IsValid = false
	}

	// Check PR files with patch data
	query = `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN patch IS NOT NULL AND patch != '' THEN 1 END) as with_patch
		FROM github_pr_files
		WHERE repo_id = $1
	`
	var prFilesTotal, prFilesWithPatch int
	err = db.QueryRow(ctx, query, repoID).Scan(&prFilesTotal, &prFilesWithPatch)
	if err != nil {
		return nil, fmt.Errorf("failed to query PR files: %w", err)
	}
	report.Summary.PRFiles = prFilesTotal
	report.Summary.PRFilesWithPatch = prFilesWithPatch

	if prFilesTotal == 0 {
		report.MissingData = append(report.MissingData, "pr_files")
		report.IsValid = false
	} else {
		pct := float64(prFilesWithPatch) / float64(prFilesTotal) * 100
		report.Completeness.PRFilesWithPatch = pct
		if pct < 80.0 {
			report.Warnings = append(report.Warnings, fmt.Sprintf("Only %.1f%% of PR files have patch data", pct))
		}
	}

	// Check contributors
	query = `SELECT COUNT(*) FROM github_contributors WHERE repo_id = $1`
	err = db.QueryRow(ctx, query, repoID).Scan(&report.Summary.Contributors)
	if err != nil {
		return nil, fmt.Errorf("failed to query contributors: %w", err)
	}
	if report.Summary.Contributors == 0 {
		report.Warnings = append(report.Warnings, "no contributors found")
	}

	// Check issue timeline events
	query = `
		SELECT COUNT(DISTINCT t.id)
		FROM github_issue_timeline t
		JOIN github_issues i ON t.issue_id = i.id
		WHERE i.repo_id = $1
	`
	err = db.QueryRow(ctx, query, repoID).Scan(&report.Summary.IssueTimelineEvents)
	if err != nil {
		// Timeline table might not exist in older schemas
		report.Summary.IssueTimelineEvents = 0
	}

	return report, nil
}

func printReport(report *ValidationReport) {
	fmt.Println("================================================================================")
	fmt.Printf("GitHub Data Staging Validation Report: %s\n", report.RepoName)
	fmt.Println("================================================================================")
	fmt.Printf("Repository ID: %d\n", report.RepoID)
	fmt.Println()

	// Overall status
	if report.IsValid {
		fmt.Println("✅ Status: VALID - All critical data present")
	} else {
		fmt.Println("❌ Status: INVALID - Missing critical data")
	}
	fmt.Println()

	// Data summary
	fmt.Println("📊 Data Summary:")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────")
	fmt.Printf("  Commits:              %6d", report.Summary.Commits)
	if report.Summary.Commits > 0 {
		fmt.Printf(" (%.1f%% with file data)\n", report.Completeness.CommitsWithFiles)
	} else {
		fmt.Println(" ❌")
	}

	fmt.Printf("  Issues:               %6d\n", report.Summary.Issues)
	fmt.Printf("  Issue Comments:       %6d\n", report.Summary.IssueComments)
	fmt.Printf("  Pull Requests:        %6d", report.Summary.PullRequests)
	if report.Summary.PullRequests == 0 {
		fmt.Println(" ❌")
	} else {
		fmt.Println()
	}

	fmt.Printf("  PR Files:             %6d", report.Summary.PRFiles)
	if report.Summary.PRFiles > 0 {
		fmt.Printf(" (%.1f%% with patches)\n", report.Completeness.PRFilesWithPatch)
	} else {
		fmt.Println(" ❌")
	}

	fmt.Printf("  Contributors:         %6d\n", report.Summary.Contributors)
	fmt.Printf("  Timeline Events:      %6d\n", report.Summary.IssueTimelineEvents)
	fmt.Println()

	// Missing data
	if len(report.MissingData) > 0 {
		fmt.Println("❌ Missing Critical Data:")
		fmt.Println("────────────────────────────────────────────────────────────────────────────────")
		for _, missing := range report.MissingData {
			fmt.Printf("  • %s\n", missing)
		}
		fmt.Println()
	}

	// Warnings
	if len(report.Warnings) > 0 {
		fmt.Println("⚠️  Warnings:")
		fmt.Println("────────────────────────────────────────────────────────────────────────────────")
		for _, warning := range report.Warnings {
			fmt.Printf("  • %s\n", warning)
		}
		fmt.Println()
	}

	// Recommendation
	fmt.Println("💡 Recommendation:")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────")
	if report.IsValid && len(report.Warnings) == 0 {
		fmt.Println("  ✅ Data is complete and ready for LLM analysis!")
		fmt.Println("  → You can safely run: crisk init --days 365")
		fmt.Println("  → GitHub API fetching will be skipped (data already staged)")
	} else if report.IsValid {
		fmt.Println("  ⚠️  Data is present but has some warnings")
		fmt.Println("  → You can still run: crisk init --days 365")
		fmt.Println("  → Some features may have incomplete data")
	} else {
		fmt.Println("  ❌ Critical data is missing")
		fmt.Println("  → Run: crisk init --days 365")
		fmt.Println("  → This will fetch missing data from GitHub API")
	}
	fmt.Println("================================================================================")
}
