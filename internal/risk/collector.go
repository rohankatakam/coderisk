package risk

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rohankatakam/coderisk/internal/graph"
)

// Collector orchestrates Phase 1 data collection (graph queries)
type Collector struct {
	graphBackend graph.Backend
}

// NewCollector creates a new Phase 1 data collector
func NewCollector(backend graph.Backend) *Collector {
	return &Collector{
		graphBackend: backend,
	}
}

// CollectPhase1Data executes all Phase 1 queries and returns consolidated data
// Per IMPLEMENTATION_GAP_ANALYSIS.md: Now executes actual graph queries
// Multi-repo safety: Converts RepoID to int64 and passes to all queries
func (c *Collector) CollectPhase1Data(ctx context.Context, req *AnalysisRequest) (*Phase1Data, error) {
	if len(req.FilePaths) == 0 {
		return nil, fmt.Errorf("no file paths provided")
	}

	filePath := req.FilePaths[0] // Primary file being analyzed

	// Convert RepoID from string to int64 for Neo4j queries
	repoID, err := strconv.ParseInt(req.RepoID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid repo_id %q: %w", req.RepoID, err)
	}

	data := &Phase1Data{
		QueryDurations: make(map[string]time.Duration),
		CollectionTime: time.Now(),
	}

	// Query 1: Change Complexity (from git diff - no graph query needed)
	if err := c.analyzeChangeComplexity(req.GitDiff, data); err != nil {
		return nil, fmt.Errorf("change complexity analysis failed: %w", err)
	}

	// Query 2: Ownership (dynamic from AUTHORED + MODIFIED edges)
	if err := c.queryOwnership(ctx, filePath, repoID, data); err != nil {
		// Non-fatal: log warning and continue
		fmt.Printf("  ⚠️  Ownership query failed: %v\n", err)
	}

	// Query 3: Blast Radius (DEPENDS_ON traversal)
	if err := c.queryBlastRadius(ctx, filePath, repoID, data); err != nil {
		fmt.Printf("  ⚠️  Blast radius query failed: %v\n", err)
	}

	// Query 4: Co-Change Partners (dynamic from MODIFIED edges)
	if err := c.queryCoChangePartners(ctx, filePath, repoID, data); err != nil {
		fmt.Printf("  ⚠️  Co-change query failed: %v\n", err)
	}

	// Query 5: Incident History (commit message regex)
	if err := c.queryIncidentHistory(ctx, filePath, repoID, data); err != nil {
		fmt.Printf("  ⚠️  Incident history query failed: %v\n", err)
	}

	// Query 6: Recent Commits (last 5 commits via MODIFIED)
	if err := c.queryRecentCommits(ctx, filePath, repoID, data); err != nil {
		fmt.Printf("  ⚠️  Recent commits query failed: %v\n", err)
	}

	return data, nil
}

func (c *Collector) analyzeChangeComplexity(gitDiff string, data *Phase1Data) error {
	start := time.Now()
	defer func() {
		data.QueryDurations["change_complexity"] = time.Since(start)
	}()

	lines := strings.Split(gitDiff, "\n")
	linesAdded := 0
	linesDeleted := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			linesAdded++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			linesDeleted++
		}
	}

	totalChanges := linesAdded + linesDeleted
	complexityScore := float64(totalChanges) / 100.0
	if complexityScore > 1.0 {
		complexityScore = 1.0
	}

	data.LinesAdded = linesAdded
	data.LinesDeleted = linesDeleted
	data.LinesModified = 0
	data.ComplexityScore = complexityScore

	return nil
}

// queryOwnership executes the ownership query and populates Phase1Data
func (c *Collector) queryOwnership(ctx context.Context, filePath string, repoID int64, data *Phase1Data) error {
	start := time.Now()
	defer func() {
		data.QueryDurations["ownership"] = time.Since(start)
	}()

	results, err := c.graphBackend.QueryWithParams(ctx, QueryOwnership, map[string]interface{}{
		"filePath": filePath,
		"repoId":   repoID,
	})
	if err != nil {
		return fmt.Errorf("ownership query failed: %w", err)
	}

	if len(results) > 0 {
		// Get primary owner (first result)
		if email, ok := results[0]["d.email"].(string); ok {
			data.OwnerEmail = email
		}
		if name, ok := results[0]["d.name"].(string); ok {
			data.FileOwner = name
		}
		if commitCount, ok := results[0]["commit_count"].(int64); ok {
			data.CommitCount = int(commitCount)
		}

		// Build developer history from all results (top 3 owners)
		data.DeveloperHistory = make([]DeveloperActivity, 0, len(results))
		for _, row := range results {
			email, _ := row["d.email"].(string)
			name, _ := row["d.name"].(string)
			commitCount, _ := row["commit_count"].(int64)
			lastCommit, _ := row["last_commit_date"].(int64)

			data.DeveloperHistory = append(data.DeveloperHistory, DeveloperActivity{
				Developer:    name,
				Email:        email,
				CommitCount:  int(commitCount),
				LastCommit:   time.Unix(lastCommit, 0),
				LinesChanged: 0, // Not available in this query
				IsActiveNow:  false,
			})
		}
		data.TeamSize = len(results)
	}

	return nil
}

// queryBlastRadius executes the blast radius query
func (c *Collector) queryBlastRadius(ctx context.Context, filePath string, repoID int64, data *Phase1Data) error {
	start := time.Now()
	defer func() {
		data.QueryDurations["blast_radius"] = time.Since(start)
	}()

	results, err := c.graphBackend.QueryWithParams(ctx, QueryBlastRadius, map[string]interface{}{
		"filePath": filePath,
		"repoId":   repoID,
	})
	if err != nil {
		return fmt.Errorf("blast radius query failed: %w", err)
	}

	if len(results) > 0 {
		if count, ok := results[0]["dependent_count"].(int64); ok {
			data.BlastRadius = int(count)
		}
		if dependents, ok := results[0]["sample_dependents"].([]interface{}); ok {
			data.DependentFiles = make([]string, 0, len(dependents))
			for _, dep := range dependents {
				if depStr, ok := dep.(string); ok {
					data.DependentFiles = append(data.DependentFiles, depStr)
				}
			}
		}
	}

	return nil
}

// queryCoChangePartners executes the co-change partners query
func (c *Collector) queryCoChangePartners(ctx context.Context, filePath string, repoID int64, data *Phase1Data) error {
	start := time.Now()
	defer func() {
		data.QueryDurations["co_change"] = time.Since(start)
	}()

	results, err := c.graphBackend.QueryWithParams(ctx, QueryCoChangePartners, map[string]interface{}{
		"filePath": filePath,
		"repoId":   repoID,
	})
	if err != nil {
		return fmt.Errorf("co-change query failed: %w", err)
	}

	data.CoChangePartners = make([]CoChangePartner, 0, len(results))
	for _, row := range results {
		path, _ := row["other.path"].(string)
		frequency, _ := row["frequency"].(float64)
		coChanges, _ := row["co_changes"].(int64)

		data.CoChangePartners = append(data.CoChangePartners, CoChangePartner{
			FilePath:       path,
			Frequency:      frequency,
			TotalCoChanges: int(coChanges),
			IsUpdated:      false, // Would need to check against current changeset
		})
	}

	return nil
}

// queryIncidentHistory executes the incident history query
func (c *Collector) queryIncidentHistory(ctx context.Context, filePath string, repoID int64, data *Phase1Data) error {
	start := time.Now()
	defer func() {
		data.QueryDurations["incident_history"] = time.Since(start)
	}()

	results, err := c.graphBackend.QueryWithParams(ctx, QueryIncidentHistory, map[string]interface{}{
		"filePath": filePath,
		"repoId":   repoID,
	})
	if err != nil {
		return fmt.Errorf("incident history query failed: %w", err)
	}

	data.IncidentCount = len(results)
	data.Incidents = make([]string, 0, len(results))
	for _, row := range results {
		sha, _ := row["c.sha"].(string)
		message, _ := row["c.message"].(string)
		data.Incidents = append(data.Incidents, fmt.Sprintf("%s: %s", sha[:7], message))
	}

	// Calculate incident density (incidents per 100 commits)
	if data.CommitCount > 0 {
		data.IncidentDensity = float64(data.IncidentCount) / float64(data.CommitCount) * 100.0
	}

	return nil
}

// queryRecentCommits executes the recent commits query
func (c *Collector) queryRecentCommits(ctx context.Context, filePath string, repoID int64, data *Phase1Data) error {
	start := time.Now()
	defer func() {
		data.QueryDurations["recent_commits"] = time.Since(start)
	}()

	results, err := c.graphBackend.QueryWithParams(ctx, QueryRecentCommits, map[string]interface{}{
		"filePath": filePath,
		"repoId":   repoID,
	})
	if err != nil {
		return fmt.Errorf("recent commits query failed: %w", err)
	}

	// Extract last modified date and modifier from most recent commit
	if len(results) > 0 {
		if timestamp, ok := results[0]["c.committed_at"].(int64); ok {
			data.LastModified = time.Unix(timestamp, 0)
		}
		if email, ok := results[0]["d.email"].(string); ok {
			data.LastModifier = email
		}

		// Calculate churn score based on recent additions/deletions
		totalChurn := 0
		for _, row := range results {
			additions, _ := row["c.additions"].(int64)
			deletions, _ := row["c.deletions"].(int64)
			totalChurn += int(additions) + int(deletions)
		}
		// Normalize churn score (0-1 scale, capped at 500 lines = max)
		data.ChurnScore = float64(totalChurn) / 500.0
		if data.ChurnScore > 1.0 {
			data.ChurnScore = 1.0
		}
	}

	return nil
}

// QueryCommitsForFilePaths executes the multi-path query to find commits that
// modified any of the given file paths. This is used for on-demand file resolution
// where historical paths are discovered via git log --follow.
//
// Parameters:
//   - ctx: Context for cancellation
//   - filePaths: Array of file paths (current + historical)
//   - repoID: Repository ID for multi-repo safety
//   - limit: Maximum number of commits to return
//
// Returns:
//   - []map[string]interface{}: Array of commit records with developer info
//   - error: If query fails
//
// Reference: issue_ingestion_implementation_plan.md Phase 1.3
func (c *Collector) QueryCommitsForFilePaths(ctx context.Context, filePaths []string, repoID int64, limit int) ([]map[string]interface{}, error) {
	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no file paths provided")
	}

	results, err := c.graphBackend.QueryWithParams(ctx, QueryCommitsForPaths, map[string]interface{}{
		"paths":  filePaths,
		"repoId": repoID,
		"limit":  limit,
	})
	if err != nil {
		return nil, fmt.Errorf("commits for paths query failed: %w", err)
	}

	return results, nil
}
