package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/temporal"
)

// Builder orchestrates graph construction from PostgreSQL staging tables
// Reference: dev_docs/03-implementation/integration_guides/layers_2_3_graph_construction.md
type Builder struct {
	stagingDB *database.StagingClient
	backend   Backend
}

// NewBuilder creates a graph builder instance
func NewBuilder(stagingDB *database.StagingClient, backend Backend) *Builder {
	return &Builder{
		stagingDB: stagingDB,
		backend:   backend,
	}
}

// BuildStats tracks graph construction statistics
type BuildStats struct {
	Nodes int
	Edges int
}

// BuildGraph constructs the complete graph for a repository
// This is Priority 6B: PostgreSQL → Neo4j/Neptune
// repoPath is the absolute path to the cloned repository (needed to resolve file paths)
func (b *Builder) BuildGraph(ctx context.Context, repoID int64, repoPath string) (*BuildStats, error) {
	log.Printf("🔨 Building graph for repo %d (path: %s)...", repoID, repoPath)
	stats := &BuildStats{}

	// Process commits (Layer 2: Temporal)
	commitStats, err := b.processCommits(ctx, repoID, repoPath)
	if err != nil {
		return stats, fmt.Errorf("process commits failed: %w", err)
	}
	stats.Nodes += commitStats.Nodes
	stats.Edges += commitStats.Edges
	log.Printf("  ✓ Processed commits: %d nodes, %d edges", commitStats.Nodes, commitStats.Edges)

	// Process issues (Layer 3: Incidents)
	issueStats, err := b.processIssues(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("process issues failed: %w", err)
	}
	stats.Nodes += issueStats.Nodes
	stats.Edges += issueStats.Edges
	log.Printf("  ✓ Processed issues: %d nodes, %d edges", issueStats.Nodes, issueStats.Edges)

	// Process PRs (Layer 3: Incidents)
	prStats, err := b.processPRs(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("process PRs failed: %w", err)
	}
	stats.Nodes += prStats.Nodes
	stats.Edges += prStats.Edges
	log.Printf("  ✓ Processed PRs: %d nodes, %d edges", prStats.Nodes, prStats.Edges)

	// Process branches (Layer 2: Temporal)
	branchStats, err := b.processBranches(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("process branches failed: %w", err)
	}
	stats.Nodes += branchStats.Nodes
	log.Printf("  ✓ Processed branches: %d nodes", branchStats.Nodes)

	// Link commits to default branch (MVP simplification: only default branch)
	// See: dev_docs/01-architecture/simplified_graph_schema.md line 268
	commitBranchStats, err := b.linkCommitsToDefaultBranch(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("link commits to default branch failed: %w", err)
	}
	stats.Edges += commitBranchStats.Edges
	log.Printf("  ✓ Linked commits to default branch: %d edges", commitBranchStats.Edges)

	// Link PRs to branches
	prBranchStats, err := b.linkPRsToBranches(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("link PRs to branches failed: %w", err)
	}
	stats.Edges += prBranchStats.Edges
	log.Printf("  ✓ Linked PRs to branches: %d edges", prBranchStats.Edges)

	// Note: CO_CHANGED edges are now computed dynamically in queries, not pre-calculated
	// This prevents stale data and reduces ingestion time by ~30%
	// See: dev_docs/01-architecture/simplified_graph_schema.md

	return stats, nil
}

// processCommits transforms commits from PostgreSQL to graph nodes/edges
func (b *Builder) processCommits(ctx context.Context, repoID int64, repoPath string) (*BuildStats, error) {
	batchSize := 100
	stats := &BuildStats{}

	for {
		// Fetch unprocessed commits
		commits, err := b.stagingDB.FetchUnprocessedCommits(ctx, repoID, batchSize)
		if err != nil {
			return stats, err
		}

		if len(commits) == 0 {
			break // All processed
		}

		// Transform to graph entities
		var allNodes []GraphNode
		var allEdges []GraphEdge
		var commitIDs []int64

		for _, commit := range commits {
			nodes, edges, err := b.transformCommit(commit, repoPath)
			if err != nil {
				log.Printf("  ⚠️  Failed to transform commit %s: %v", commit.SHA, err)
				continue
			}

			allNodes = append(allNodes, nodes...)
			allEdges = append(allEdges, edges...)
			commitIDs = append(commitIDs, commit.ID)
		}

		// Create nodes
		if len(allNodes) > 0 {
			if _, err := b.backend.CreateNodes(ctx, allNodes); err != nil {
				return stats, fmt.Errorf("failed to create nodes: %w", err)
			}
			stats.Nodes += len(allNodes)
		}

		// Create edges
		if len(allEdges) > 0 {
			if err := b.backend.CreateEdges(ctx, allEdges); err != nil {
				return stats, fmt.Errorf("failed to create edges: %w", err)
			}
			stats.Edges += len(allEdges)
		}

		// Mark as processed
		if len(commitIDs) > 0 {
			if err := b.stagingDB.MarkCommitsProcessed(ctx, commitIDs); err != nil {
				return stats, fmt.Errorf("failed to mark commits as processed: %w", err)
			}
		}
	}

	return stats, nil
}

// transformCommit converts a commit into graph nodes and edges
// repoPath is used to convert relative file paths from GitHub to absolute paths matching File nodes
func (b *Builder) transformCommit(commit database.CommitData, repoPath string) ([]GraphNode, []GraphEdge, error) {
	var nodes []GraphNode
	var edges []GraphEdge

	// Parse raw data to extract files
	var fullCommit struct {
		Files []struct {
			Filename  string `json:"filename"`
			Status    string `json:"status"`
			Additions int    `json:"additions"`
			Deletions int    `json:"deletions"`
		} `json:"files"`
		Stats struct {
			Additions int `json:"additions"`
			Deletions int `json:"deletions"`
			Total     int `json:"total"`
		} `json:"stats"`
	}

	if err := json.Unmarshal(commit.RawData, &fullCommit); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal commit: %w", err)
	}

	// 1. Create Commit node
	commitNode := GraphNode{
		Label: "Commit",
		ID:    fmt.Sprintf("commit:%s", commit.SHA),
		Properties: map[string]interface{}{
			"sha":          commit.SHA,
			"message":      commit.Message,
			"author_email": commit.AuthorEmail,
			"author_name":  commit.AuthorName,
			"author_date":  commit.AuthorDate.Unix(),
			"additions":    fullCommit.Stats.Additions,
			"deletions":    fullCommit.Stats.Deletions,
		},
	}
	nodes = append(nodes, commitNode)

	// 2. Create Developer node
	developerNode := GraphNode{
		Label: "Developer",
		ID:    fmt.Sprintf("developer:%s", commit.AuthorEmail),
		Properties: map[string]interface{}{
			"email": commit.AuthorEmail,
			"name":  commit.AuthorName,
		},
	}
	nodes = append(nodes, developerNode)

	// 3. Create AUTHORED edge
	authoredEdge := GraphEdge{
		Label: "AUTHORED",
		From:  developerNode.ID,
		To:    commitNode.ID,
		Properties: map[string]interface{}{
			"timestamp": commit.AuthorDate.Unix(),
		},
	}
	edges = append(edges, authoredEdge)

	// 4. Create MODIFIES edges for each file
	// Convert relative GitHub paths to absolute paths matching File nodes
	for _, file := range fullCommit.Files {
		// Convert relative path (e.g., "src/main.go") to absolute (e.g., "/path/to/repo/src/main.go")
		absolutePath := fmt.Sprintf("%s/%s", repoPath, file.Filename)

		modifiesEdge := GraphEdge{
			Label: "MODIFIES",
			From:  commitNode.ID,
			To:    fmt.Sprintf("file:%s", absolutePath),
			Properties: map[string]interface{}{
				"status":    file.Status,
				"additions": file.Additions,
				"deletions": file.Deletions,
				"timestamp": commit.AuthorDate.Unix(),
			},
		}
		edges = append(edges, modifiesEdge)
	}

	return nodes, edges, nil
}

// processIssues transforms issues from PostgreSQL to graph nodes
func (b *Builder) processIssues(ctx context.Context, repoID int64) (*BuildStats, error) {
	batchSize := 100
	stats := &BuildStats{}

	for {
		// Fetch unprocessed issues
		issues, err := b.stagingDB.FetchUnprocessedIssues(ctx, repoID, batchSize)
		if err != nil {
			return stats, err
		}

		if len(issues) == 0 {
			break
		}

		// Transform to graph entities
		var allNodes []GraphNode
		var issueIDs []int64

		for _, issue := range issues {
			node, err := b.transformIssue(issue)
			if err != nil {
				log.Printf("  ⚠️  Failed to transform issue #%d: %v", issue.Number, err)
				continue
			}

			allNodes = append(allNodes, node)
			issueIDs = append(issueIDs, issue.ID)
		}

		// Create nodes
		if len(allNodes) > 0 {
			if _, err := b.backend.CreateNodes(ctx, allNodes); err != nil {
				return stats, fmt.Errorf("failed to create issue nodes: %w", err)
			}
			stats.Nodes += len(allNodes)
		}

		// Mark as processed
		if len(issueIDs) > 0 {
			if err := b.stagingDB.MarkIssuesProcessed(ctx, issueIDs); err != nil {
				return stats, fmt.Errorf("failed to mark issues as processed: %w", err)
			}
		}
	}

	return stats, nil
}

// transformIssue converts an issue into a graph node
func (b *Builder) transformIssue(issue database.IssueData) (GraphNode, error) {
	// Parse labels
	var labels []string
	if err := json.Unmarshal(issue.Labels, &labels); err != nil {
		return GraphNode{}, fmt.Errorf("failed to unmarshal labels: %w", err)
	}

	node := GraphNode{
		Label: "Issue",
		ID:    fmt.Sprintf("issue:%d", issue.Number),
		Properties: map[string]interface{}{
			"number":     issue.Number,
			"title":      issue.Title,
			"body":       issue.Body,
			"state":      issue.State,
			"labels":     labels,
			"created_at": issue.CreatedAt.Unix(),
		},
	}

	if issue.ClosedAt != nil {
		node.Properties["closed_at"] = issue.ClosedAt.Unix()
	}

	return node, nil
}

// processPRs transforms PRs from PostgreSQL to graph nodes/edges
func (b *Builder) processPRs(ctx context.Context, repoID int64) (*BuildStats, error) {
	batchSize := 100
	stats := &BuildStats{}

	for {
		// Fetch unprocessed PRs
		prs, err := b.stagingDB.FetchUnprocessedPRs(ctx, repoID, batchSize)
		if err != nil {
			return stats, err
		}

		if len(prs) == 0 {
			break
		}

		// Transform to graph entities
		var allNodes []GraphNode
		var allEdges []GraphEdge
		var prIDs []int64

		for _, pr := range prs {
			node, edges, err := b.transformPR(pr)
			if err != nil {
				log.Printf("  ⚠️  Failed to transform PR #%d: %v", pr.Number, err)
				continue
			}

			allNodes = append(allNodes, node)
			allEdges = append(allEdges, edges...)
			prIDs = append(prIDs, pr.ID)
		}

		// Create nodes
		if len(allNodes) > 0 {
			if _, err := b.backend.CreateNodes(ctx, allNodes); err != nil {
				return stats, fmt.Errorf("failed to create PR nodes: %w", err)
			}
			stats.Nodes += len(allNodes)
		}

		// Create edges
		if len(allEdges) > 0 {
			if err := b.backend.CreateEdges(ctx, allEdges); err != nil {
				return stats, fmt.Errorf("failed to create PR edges: %w", err)
			}
			stats.Edges += len(allEdges)
		}

		// Mark as processed
		if len(prIDs) > 0 {
			if err := b.stagingDB.MarkPRsProcessed(ctx, prIDs); err != nil {
				return stats, fmt.Errorf("failed to mark PRs as processed: %w", err)
			}
		}
	}

	return stats, nil
}

// transformPR converts a PR into graph node and edges
func (b *Builder) transformPR(pr database.PRData) (GraphNode, []GraphEdge, error) {
	var edges []GraphEdge

	// Create PR node
	node := GraphNode{
		Label: "PullRequest",
		ID:    fmt.Sprintf("pr:%d", pr.Number),
		Properties: map[string]interface{}{
			"number":     pr.Number,
			"title":      pr.Title,
			"body":       pr.Body,
			"state":      pr.State,
			"merged":     pr.Merged,
			"created_at": pr.CreatedAt.Unix(),
		},
	}

	if pr.MergedAt != nil {
		node.Properties["merged_at"] = pr.MergedAt.Unix()
	}

	// Create MERGED_TO edge if merged
	if pr.Merged && pr.MergeCommitSHA != nil {
		mergedToEdge := GraphEdge{
			Label:      "MERGED_TO",
			From:       node.ID,
			To:         fmt.Sprintf("commit:%s", *pr.MergeCommitSHA),
			Properties: map[string]interface{}{},
		}
		if pr.MergedAt != nil {
			mergedToEdge.Properties["merged_at"] = pr.MergedAt.Unix()
		}
		edges = append(edges, mergedToEdge)
	}

	// Extract issue references from title and body for FIXES edges
	issueNumbers := extractIssueReferences(pr.Title, pr.Body)
	for _, issueNum := range issueNumbers {
		fixesEdge := GraphEdge{
			Label: "FIXES",
			From:  node.ID,
			To:    fmt.Sprintf("issue:%d", issueNum),
			Properties: map[string]interface{}{
				"detected_from": "pr_body",
				"timestamp":     pr.CreatedAt.Unix(),
			},
		}
		edges = append(edges, fixesEdge)
	}

	return node, edges, nil
}

// extractIssueReferences parses text for "Fixes #123", "Closes #456" patterns
func extractIssueReferences(title, body string) []int {
	text := strings.ToLower(title + " " + body)
	re := regexp.MustCompile(`(?:fix|fixes|fixed|close|closes|closed|resolve|resolves|resolved)\s+#(\d+)`)
	matches := re.FindAllStringSubmatch(text, -1)

	issueNumbers := []int{}
	seen := make(map[int]bool)

	for _, match := range matches {
		if len(match) > 1 {
			if num, err := strconv.Atoi(match[1]); err == nil {
				if !seen[num] {
					issueNumbers = append(issueNumbers, num)
					seen[num] = true
				}
			}
		}
	}

	return issueNumbers
}

// AddLayer2CoChangedEdges is DEPRECATED - CO_CHANGED edges are now computed dynamically
// This method is kept for backwards compatibility but does nothing
// See: dev_docs/01-architecture/simplified_graph_schema.md
func (b *Builder) AddLayer2CoChangedEdges(ctx context.Context, coChanges []temporal.CoChangeResult) (*BuildStats, error) {
	log.Printf("⚠️  AddLayer2CoChangedEdges called but CO_CHANGED edges are now computed dynamically")
	log.Printf("    See: dev_docs/01-architecture/simplified_graph_schema.md")
	return &BuildStats{}, nil
}

// AddLayer3IncidentNodes creates Incident nodes in the graph
// This is called from the incidents package when creating manual incident links
func (b *Builder) AddLayer3IncidentNodes(ctx context.Context, incidents []GraphNode) (*BuildStats, error) {
	stats := &BuildStats{}

	if len(incidents) == 0 {
		return stats, nil
	}

	// Create incident nodes
	if _, err := b.backend.CreateNodes(ctx, incidents); err != nil {
		return stats, fmt.Errorf("failed to create incident nodes: %w", err)
	}

	stats.Nodes = len(incidents)
	return stats, nil
}

// AddLayer3CausedByEdges creates CAUSED_BY edges between incidents and files
// This is called from the incidents package when creating manual incident links
func (b *Builder) AddLayer3CausedByEdges(ctx context.Context, edges []GraphEdge) (*BuildStats, error) {
	stats := &BuildStats{}

	if len(edges) == 0 {
		return stats, nil
	}

	// Create CAUSED_BY edges
	if err := b.backend.CreateEdges(ctx, edges); err != nil {
		return stats, fmt.Errorf("failed to create CAUSED_BY edges: %w", err)
	}

	stats.Edges = len(edges)
	return stats, nil
}

// processBranches transforms branches from PostgreSQL to graph nodes
func (b *Builder) processBranches(ctx context.Context, repoID int64) (*BuildStats, error) {
	batchSize := 100
	stats := &BuildStats{}

	// Get default branch name
	defaultBranchName, err := b.stagingDB.GetDefaultBranchName(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("failed to get default branch name: %w", err)
	}

	for {
		// Fetch unprocessed branches
		branches, err := b.stagingDB.FetchUnprocessedBranches(ctx, repoID, batchSize)
		if err != nil {
			return stats, err
		}

		if len(branches) == 0 {
			break
		}

		// Transform to graph entities
		var allNodes []GraphNode
		var branchIDs []int64

		for _, branch := range branches {
			node := b.transformBranch(branch, defaultBranchName)
			allNodes = append(allNodes, node)
			branchIDs = append(branchIDs, branch.ID)
		}

		// Create nodes
		if len(allNodes) > 0 {
			if _, err := b.backend.CreateNodes(ctx, allNodes); err != nil {
				return stats, fmt.Errorf("failed to create branch nodes: %w", err)
			}
			stats.Nodes += len(allNodes)
		}

		// Mark as processed
		if len(branchIDs) > 0 {
			if err := b.stagingDB.MarkBranchesProcessed(ctx, branchIDs); err != nil {
				return stats, fmt.Errorf("failed to mark branches as processed: %w", err)
			}
		}
	}

	return stats, nil
}

// transformBranch converts a branch into a graph node
func (b *Builder) transformBranch(branch database.BranchData, defaultBranchName string) GraphNode {
	isDefault := branch.Name == defaultBranchName

	node := GraphNode{
		Label: "Branch",
		ID:    fmt.Sprintf("branch:%s", branch.Name),
		Properties: map[string]interface{}{
			"name":       branch.Name,
			"is_default": isDefault,
		},
	}

	return node
}

// linkCommitsToDefaultBranch creates [:ON_BRANCH] edges from all commits to the default branch
// MVP simplification: only link to default branch (see simplified_graph_schema.md line 268)
func (b *Builder) linkCommitsToDefaultBranch(ctx context.Context, repoID int64) (*BuildStats, error) {
	stats := &BuildStats{}

	// Get default branch name
	defaultBranchName, err := b.stagingDB.GetDefaultBranchName(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("failed to get default branch name: %w", err)
	}

	// Get all processed commit SHAs
	shas, err := b.stagingDB.GetProcessedCommitSHAs(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("failed to get commit SHAs: %w", err)
	}

	// Build ON_BRANCH edges
	var edges []GraphEdge
	for _, sha := range shas {
		edge := GraphEdge{
			Label:      "ON_BRANCH",
			From:       fmt.Sprintf("commit:%s", sha),
			To:         fmt.Sprintf("branch:%s", defaultBranchName),
			Properties: map[string]interface{}{},
		}
		edges = append(edges, edge)
	}

	// Create edges in batches
	if len(edges) > 0 {
		batchSize := 100
		for i := 0; i < len(edges); i += batchSize {
			end := i + batchSize
			if end > len(edges) {
				end = len(edges)
			}

			batch := edges[i:end]
			if err := b.backend.CreateEdges(ctx, batch); err != nil {
				return stats, fmt.Errorf("failed to create ON_BRANCH edges (batch %d-%d): %w", i, end, err)
			}
		}

		stats.Edges = len(edges)
	}

	return stats, nil
}

// linkPRsToBranches creates [:FROM_BRANCH], [:TO_BRANCH], [:MERGED_AS] edges for PRs
func (b *Builder) linkPRsToBranches(ctx context.Context, repoID int64) (*BuildStats, error) {
	stats := &BuildStats{}

	// Get all processed PRs with branch data
	prs, err := b.stagingDB.GetProcessedPRBranchData(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("failed to get PR branch data: %w", err)
	}

	// Build PR-branch edges
	var edges []GraphEdge
	for _, pr := range prs {
		prID := fmt.Sprintf("pr:%d", pr.Number)

		// Create FROM_BRANCH edge (source branch)
		fromBranchEdge := GraphEdge{
			Label:      "FROM_BRANCH",
			From:       prID,
			To:         fmt.Sprintf("branch:%s", pr.HeadRef),
			Properties: map[string]interface{}{},
		}
		edges = append(edges, fromBranchEdge)

		// Create TO_BRANCH edge (target branch)
		toBranchEdge := GraphEdge{
			Label:      "TO_BRANCH",
			From:       prID,
			To:         fmt.Sprintf("branch:%s", pr.BaseRef),
			Properties: map[string]interface{}{},
		}
		edges = append(edges, toBranchEdge)

		// Create MERGED_AS edge if PR was merged
		if pr.Merged && pr.MergeCommitSHA != nil {
			mergedAsEdge := GraphEdge{
				Label:      "MERGED_AS",
				From:       prID,
				To:         fmt.Sprintf("commit:%s", *pr.MergeCommitSHA),
				Properties: map[string]interface{}{},
			}
			edges = append(edges, mergedAsEdge)
		}
	}

	// Create edges in batches
	if len(edges) > 0 {
		batchSize := 100
		for i := 0; i < len(edges); i += batchSize {
			end := i + batchSize
			if end > len(edges) {
				end = len(edges)
			}

			batch := edges[i:end]
			if err := b.backend.CreateEdges(ctx, batch); err != nil {
				return stats, fmt.Errorf("failed to create PR-branch edges (batch %d-%d): %w", i, end, err)
			}
		}

		stats.Edges = len(edges)
	}

	return stats, nil
}
