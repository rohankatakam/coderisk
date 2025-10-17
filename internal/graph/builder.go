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
	stagingDB       *database.StagingClient
	backend         Backend
	processedCommits []temporal.Commit // Track commits for co-change calculation
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

	// Initialize commit tracking for co-change calculation
	b.processedCommits = make([]temporal.Commit, 0)

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

	// Calculate CO_CHANGED edges from commit patterns
	log.Printf("  ℹ️  Calculating co-change patterns from commits...")
	coChangeStats, err := b.calculateCoChangedEdges(ctx, repoID, repoPath)
	if err != nil {
		// Don't fail the entire build if co-change calculation fails
		log.Printf("  ⚠️  CO_CHANGED edge calculation failed (non-fatal): %v", err)
	} else {
		stats.Edges += coChangeStats.Edges
		log.Printf("  ✓ Created CO_CHANGED edges: %d pairs", coChangeStats.Edges/2) // Bidirectional, so divide by 2
	}

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
	fileChanges := make([]temporal.FileChange, 0, len(fullCommit.Files))
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

		// Track for co-change calculation
		fileChanges = append(fileChanges, temporal.FileChange{
			Path:      absolutePath,
			Additions: file.Additions,
			Deletions: file.Deletions,
		})
	}

	// Store commit for later co-change calculation
	b.processedCommits = append(b.processedCommits, temporal.Commit{
		SHA:          commit.SHA,
		Author:       commit.AuthorName,
		Email:        commit.AuthorEmail,
		Timestamp:    commit.AuthorDate,
		Message:      commit.Message,
		FilesChanged: fileChanges,
	})

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

// AddLayer2CoChangedEdges creates CO_CHANGED edges from temporal analysis
// 12-Factor Principle - Factor 8: Concurrency via process model
// Use batching to avoid overwhelming the database with large transactions
func (b *Builder) AddLayer2CoChangedEdges(ctx context.Context, coChanges []temporal.CoChangeResult) (*BuildStats, error) {
	stats := &BuildStats{}
	var edges []GraphEdge

	// Build all edges (forward + reverse for bidirectional relationship)
	for _, cc := range coChanges {
		// Forward edge
		edge := GraphEdge{
			Label: "CO_CHANGED",
			From:  fmt.Sprintf("file:%s", cc.FileA),
			To:    fmt.Sprintf("file:%s", cc.FileB),
			Properties: map[string]interface{}{
				"frequency":   cc.Frequency,
				"co_changes":  cc.CoChanges,
				"window_days": cc.WindowDays,
			},
		}
		edges = append(edges, edge)

		// Reverse edge (CO_CHANGED is bidirectional)
		reverseEdge := GraphEdge{
			Label: "CO_CHANGED",
			From:  fmt.Sprintf("file:%s", cc.FileB),
			To:    fmt.Sprintf("file:%s", cc.FileA),
			Properties: map[string]interface{}{
				"frequency":   cc.Frequency,
				"co_changes":  cc.CoChanges,
				"window_days": cc.WindowDays,
			},
		}
		edges = append(edges, reverseEdge)
	}

	// Batch create edges to avoid large transactions
	if len(edges) > 0 {
		log.Printf("Creating %d CO_CHANGED edges in batches...", len(edges))

		// Create in batches of 100 to avoid large transactions
		batchSize := 100
		for i := 0; i < len(edges); i += batchSize {
			end := i + batchSize
			if end > len(edges) {
				end = len(edges)
			}

			batch := edges[i:end]
			if err := b.backend.CreateEdges(ctx, batch); err != nil {
				return stats, fmt.Errorf("failed to create CO_CHANGED edges (batch %d-%d): %w", i, end, err)
			}

			log.Printf("  ✓ Created batch %d-%d (%d edges)", i, end, len(batch))
		}

		stats.Edges = len(edges)
		log.Printf("  ✓ Created %d CO_CHANGED edges total", len(edges))
	}

	return stats, nil
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

// calculateCoChangedEdges calculates co-change patterns from processed commits
func (b *Builder) calculateCoChangedEdges(ctx context.Context, repoID int64, repoPath string) (*BuildStats, error) {
	if len(b.processedCommits) == 0 {
		log.Printf("    No commits to analyze for co-change patterns")
		return &BuildStats{}, nil
	}

	log.Printf("    Analyzing %d commits for co-change patterns...", len(b.processedCommits))

	// Calculate co-changes with 30% frequency threshold
	coChanges := temporal.CalculateCoChanges(b.processedCommits, 0.3)
	log.Printf("    Found %d co-change pairs (frequency >= 30%%)", len(coChanges))

	// Create CO_CHANGED edges
	return b.AddLayer2CoChangedEdges(ctx, coChanges)
}
