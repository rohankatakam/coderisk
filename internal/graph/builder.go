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
	"github.com/rohankatakam/coderisk/internal/linking/types"
)

// normalizeGitHubEmail normalizes GitHub email formats to ensure consistency
// between commit authors and PR authors.
//
// GitHub uses two email formats:
//   - Commits: "{GitHub_ID}+{login}@users.noreply.github.com" (e.g., "12345+user@...")
//   - PRs: "{login}@users.noreply.github.com" (e.g., "user@...")
//
// This function strips the GitHub ID prefix to create a canonical email.
//
// Examples:
//   - "38349974+EddyDavies@users.noreply.github.com" → "EddyDavies@users.noreply.github.com"
//   - "EddyDavies@users.noreply.github.com" → "EddyDavies@users.noreply.github.com"
//   - "real.email@example.com" → "real.email@example.com" (unchanged)
func normalizeGitHubEmail(email string) string {
	// Check if this is a GitHub noreply email with ID prefix
	if strings.Contains(email, "+") && strings.Contains(email, "@users.noreply.github.com") {
		parts := strings.Split(email, "+")
		if len(parts) == 2 {
			// Return "login@users.noreply.github.com"
			return parts[1]
		}
	}
	// Return email unchanged if not a GitHub noreply format
	return email
}

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
// Schema: PRE_COMMIT_GRAPH_SPEC.md - 4 nodes (File, Developer, Commit, PR), 7 edges
func (b *Builder) BuildGraph(ctx context.Context, repoID int64, repoPath string) (*BuildStats, error) {
	log.Printf("🔨 Building graph for repo %d (path: %s)...", repoID, repoPath)
	stats := &BuildStats{}

	// Process commits (creates Commit, Developer nodes + AUTHORED, MODIFIED edges)
	commitStats, err := b.processCommits(ctx, repoID, repoPath)
	if err != nil {
		return stats, fmt.Errorf("process commits failed: %w", err)
	}
	stats.Nodes += commitStats.Nodes
	stats.Edges += commitStats.Edges
	log.Printf("  ✓ Processed commits: %d nodes, %d edges", commitStats.Nodes, commitStats.Edges)

	// Process PRs (creates PR nodes + CREATED, MERGED_AS edges)
	prStats, err := b.processPRs(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("process PRs failed: %w", err)
	}
	stats.Nodes += prStats.Nodes
	stats.Edges += prStats.Edges
	log.Printf("  ✓ Processed PRs: %d nodes, %d edges", prStats.Nodes, prStats.Edges)

	// NOTE: OWNS edges removed - ownership is now computed dynamically via queries
	// See: IMPLEMENTATION_GAP_ANALYSIS.md - Ownership should be dynamic from AUTHORED + MODIFIED edges
	// No pre-computed OWNS edges in MVP spec

	// Link commits to PRs via IN_PR edges
	commitPRStats, err := b.linkCommitsToPRs(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("link commits to PRs failed: %w", err)
	}
	stats.Edges += commitPRStats.Edges
	log.Printf("  ✓ Linked commits to PRs: %d edges", commitPRStats.Edges)

	// Process Issues (creates Issue nodes)
	// Phase 2 of issue_ingestion_implementation_plan.md
	issueStats, err := b.processIssues(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("process issues failed: %w", err)
	}
	stats.Nodes += issueStats.Nodes
	log.Printf("  ✓ Processed issues: %d nodes", issueStats.Nodes)

	// Run Temporal Correlation (finds temporal matches and stores as IssueCommitRefs)
	// Reference: LINKING_PATTERNS.md - Pattern 2: Temporal (20% of real-world cases)
	temporalStats, err := b.runTemporalCorrelation(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("temporal correlation failed: %w", err)
	}
	log.Printf("  ✓ Temporal correlation: found %d matches", temporalStats.Edges)

	// Link Issues to Commits/PRs (creates FIXED_BY/ASSOCIATED_WITH edges)
	// Reference: REVISED_MVP_STRATEGY.md - Two-way issue linking
	// This now includes both explicit references AND temporal matches
	linkStats, err := b.linkIssues(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("link issues failed: %w", err)
	}
	stats.Nodes += linkStats.Nodes
	stats.Edges += linkStats.Edges
	log.Printf("  ✓ Linked issues: %d nodes, %d edges", linkStats.Nodes, linkStats.Edges)

	// Load validated issue-PR links from PostgreSQL into Neo4j
	// Uses multi-signal ground truth classification for FIXED_BY vs ASSOCIATED_WITH edges
	// Reference: Gap closure implementation - Multi-Signal Ground Truth Classification
	log.Printf("  Loading validated issue-PR links...")
	validatedLinkStats, err := b.loadIssuePRLinksIntoGraph(ctx, repoID)
	if err != nil {
		log.Printf("  ⚠️  Warning: Failed to load validated links: %v", err)
		// Don't fail entire build, continue with old system links
	} else {
		stats.Edges += validatedLinkStats.Edges
		log.Printf("  ✓ Loaded %d validated issue-PR links", validatedLinkStats.Edges)
	}

	return stats, nil
}

// runTemporalCorrelation finds temporal correlations between issues and PRs/commits
// and stores them as IssueCommitRef entries in the database
func (b *Builder) runTemporalCorrelation(ctx context.Context, repoID int64) (*BuildStats, error) {
	stats := &BuildStats{}

	// Create temporal correlator (only needs staging DB, not Neo4j)
	correlator := NewTemporalCorrelator(b.stagingDB, nil)

	// Find temporal matches
	matches, err := correlator.FindTemporalMatches(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("failed to find temporal matches: %w", err)
	}

	// Store matches in database as IssueCommitRefs
	if err := correlator.StoreTemporalMatches(ctx, repoID, matches); err != nil {
		return stats, fmt.Errorf("failed to store temporal matches: %w", err)
	}

	stats.Edges = len(matches)
	return stats, nil
}

// linkIssues creates Issue nodes and FIXED_BY/ASSOCIATED_WITH edges
// based on both explicit LLM extraction AND temporal correlation
func (b *Builder) linkIssues(ctx context.Context, repoID int64) (*BuildStats, error) {
	linker := NewIssueLinker(b.stagingDB, b.backend)
	return linker.LinkIssues(ctx, repoID)
}

// TestLoadIssuePRLinks is a public wrapper for testing the link loading functionality
func (b *Builder) TestLoadIssuePRLinks(ctx context.Context, repoID int64) (*BuildStats, error) {
	return b.loadIssuePRLinksIntoGraph(ctx, repoID)
}

// hasExplicitFixesKeyword checks evidence for explicit fixing keywords
func hasExplicitFixesKeyword(evidenceSources []string) bool {
	fixKeywords := []string{"ref_fixes", "ref_closes", "ref_resolves"}
	for _, evidence := range evidenceSources {
		for _, keyword := range fixKeywords {
			if strings.Contains(evidence, keyword) {
				return true
			}
		}
	}
	return false
}

// loadIssuePRLinksIntoGraph loads validated issue-PR links from PostgreSQL into Neo4j
// Uses multi-signal ground truth classification to determine FIXED_BY vs ASSOCIATED_WITH edges
// Reference: Multi-Signal Ground Truth Classification strategy (gap closure implementation)
func (b *Builder) loadIssuePRLinksIntoGraph(ctx context.Context, repoID int64) (*BuildStats, error) {
	stats := &BuildStats{}

	// Query all links from the new linking system
	links, err := b.stagingDB.GetIssuePRLinks(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("failed to get issue-PR links: %w", err)
	}

	if len(links) == 0 {
		log.Printf("  No issue-PR links found")
		return stats, nil
	}

	log.Printf("  Found %d issue-PR links to process", len(links))

	var edges []GraphEdge
	fixedByCount := 0
	associatedCount := 0
	filteredCount := 0

	for _, link := range links {
		// Skip low-quality links (confidence < 0.70)
		if link.FinalConfidence < 0.70 {
			filteredCount++
			continue
		}

		// Determine edge type using multi-signal criteria
		edgeLabel := determineEdgeTypeMultiSignal(&link)

		if edgeLabel == "FIXED_BY" {
			fixedByCount++
		} else {
			associatedCount++
		}

		// Create edge with comprehensive properties
		edge := GraphEdge{
			Label: edgeLabel,
			From:  fmt.Sprintf("issue:%d", link.IssueNumber),
			To:    fmt.Sprintf("pr:%d", link.PRNumber),
			Properties: map[string]interface{}{
				"confidence":          link.FinalConfidence,
				"detection_method":    string(link.DetectionMethod),
				"link_quality":        string(link.LinkQuality),
				"evidence_sources":    link.EvidenceSources,
				// Store breakdown for transparency and future human review
				"base_confidence":     link.ConfidenceBreakdown.BaseConfidence,
				"temporal_boost":      link.ConfidenceBreakdown.TemporalBoost,
				"bidirectional_boost": link.ConfidenceBreakdown.BidirectionalBoost,
				"semantic_boost":      link.ConfidenceBreakdown.SemanticBoost,
				"negative_penalty":    link.ConfidenceBreakdown.NegativeSignalPenalty,
				"created_from":        "validated_link", // Distinguish from old system
			},
		}

		edges = append(edges, edge)
	}

	// Delete old Issue-PR edges before creating new validated ones
	// This ensures we replace legacy temporal/extraction edges with Multi-Signal classified edges
	if len(edges) > 0 {
		log.Printf("  Removing old Issue-PR edges to avoid duplicates...")
		deleteQuery := `
			MATCH (i:Issue)-[r]->(pr:PR)
			WHERE r.created_from IS NULL OR r.created_from <> 'validated_link'
			DELETE r
		`
		if err := b.backend.ExecuteBatch(ctx, []string{deleteQuery}); err != nil {
			log.Printf("  ⚠️  Warning: Failed to delete old edges: %v", err)
			// Continue anyway - CreateEdges will create new edges
		}

		// Batch create edges in Neo4j
		if err := b.backend.CreateEdges(ctx, edges); err != nil {
			return stats, fmt.Errorf("failed to create edges: %w", err)
		}
		stats.Edges = len(edges)

		log.Printf("  ✓ Created %d FIXED_BY edges", fixedByCount)
		log.Printf("  ✓ Created %d ASSOCIATED_WITH edges", associatedCount)
		if filteredCount > 0 {
			log.Printf("  ℹ Filtered %d low-confidence links (< 0.70)", filteredCount)
		}
	}

	return stats, nil
}

// determineEdgeTypeMultiSignal uses multi-dimensional ground truth signals
// to determine if a link should be FIXED_BY or ASSOCIATED_WITH
// Reference: Multi-Signal Ground Truth Classification strategy
func determineEdgeTypeMultiSignal(link *types.LinkOutput) string {
	// Criterion 1: Must be high-quality detection method
	if link.DetectionMethod != types.DetectionGitHubTimeline &&
	   link.DetectionMethod != types.DetectionExplicitBidir {
		return "ASSOCIATED_WITH"
	}

	// Criterion 2: Must have high base confidence from reliable source
	if link.ConfidenceBreakdown.BaseConfidence < 0.85 {
		return "ASSOCIATED_WITH"
	}

	// Criterion 3: Must have no conflicting evidence
	if link.ConfidenceBreakdown.NegativeSignalPenalty != 0.0 {
		return "ASSOCIATED_WITH"
	}

	// Criterion 4: Must have at least ONE ground truth signal
	// Signal A: Strong temporal correlation (GitHub auto-close behavior)
	signalA := link.ConfidenceBreakdown.TemporalBoost >= 0.12
	// Signal B: Bidirectional verification
	signalB := link.ConfidenceBreakdown.BidirectionalBoost > 0.0
	// Signal C: High semantic + explicit fixes keyword
	signalC := link.ConfidenceBreakdown.SemanticBoost >= 0.10 && hasExplicitFixesKeyword(link.EvidenceSources)

	hasGroundTruthSignal := signalA || signalB || signalC

	if hasGroundTruthSignal {
		return "FIXED_BY"
	}

	return "ASSOCIATED_WITH"
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
	// Schema: PRE_COMMIT_GRAPH_SPEC.md - Commit with on_default_branch property
	commitNode := GraphNode{
		Label: "Commit",
		ID:    fmt.Sprintf("commit:%s", commit.SHA),
		Properties: map[string]interface{}{
			"sha":                commit.SHA,
			"message":            commit.Message,
			"author_email":       commit.AuthorEmail,
			"committed_at":       commit.AuthorDate.Unix(),
			"additions":          fullCommit.Stats.Additions,
			"deletions":          fullCommit.Stats.Deletions,
			"on_default_branch":  true, // All commits from GitHub API are on default branch
		},
	}
	nodes = append(nodes, commitNode)

	// 2. Create Developer node
	// Schema: PRE_COMMIT_GRAPH_SPEC.md - Developer with email (PRIMARY KEY), name
	// Normalize email to ensure consistency with PR authors
	normalizedEmail := normalizeGitHubEmail(commit.AuthorEmail)
	developerNode := GraphNode{
		Label: "Developer",
		ID:    fmt.Sprintf("developer:%s", normalizedEmail),
		Properties: map[string]interface{}{
			"email":        normalizedEmail,
			"name":         commit.AuthorName,
			"last_active":  commit.AuthorDate.Unix(), // Most recent commit timestamp
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

	// 4. Create File nodes + MODIFIED edges for each file (Commit → File)
	// Schema: PRE_COMMIT_GRAPH_SPEC.md - MODIFIED edge with additions, deletions, status
	// GitHub API provides relative paths which now match File node paths exactly!
	// Reference: issue_ingestion_implementation_plan.md Phase 1 - File Resolution
	for _, file := range fullCommit.Files {
		// Use GitHub API path directly - it's relative from repo root
		relativePath := file.Filename

		// Create File node with historical flag
		// These represent files as they existed at commit time (historical paths)
		fileNode := GraphNode{
			Label: "File",
			ID:    fmt.Sprintf("file:%s", relativePath),
			Properties: map[string]interface{}{
				"path":       relativePath,
				"historical": true, // Mark as historical (from GitHub commits)
			},
		}
		nodes = append(nodes, fileNode)

		// Create MODIFIED edge
		modifiedEdge := GraphEdge{
			Label: "MODIFIED",
			From:  commitNode.ID,
			To:    fmt.Sprintf("file:%s", relativePath),
			Properties: map[string]interface{}{
				"additions": file.Additions,
				"deletions": file.Deletions,
				"status":    file.Status, // "added", "modified", "deleted", "renamed"
			},
		}
		edges = append(edges, modifiedEdge)
	}

	return nodes, edges, nil
}

// NOTE: Issue processing removed - Issues are not part of PRE_COMMIT_GRAPH_SPEC.md
// Issues will still be fetched from GitHub API for future use, but not stored in graph

// processPRs transforms PRs from PostgreSQL to graph nodes/edges
// Creates PR nodes + CREATED (Developer → PR) and MERGED_AS (PR → Commit) edges
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

			// Add CREATED edge (Developer → PR)
			// Parse raw data to get author email
			var fullPR struct {
				User struct {
					Login string `json:"login"`
					Email string `json:"email"`
				} `json:"user"`
			}
			if err := json.Unmarshal(pr.RawData, &fullPR); err == nil {
				// GitHub API doesn't always return email, construct from login
				authorEmail := fullPR.User.Email
				if authorEmail == "" {
					// Fallback: use login@users.noreply.github.com
					authorEmail = fmt.Sprintf("%s@users.noreply.github.com", fullPR.User.Login)
				}

				createdEdge := GraphEdge{
					Label: "CREATED",
					From:  fmt.Sprintf("developer:%s", authorEmail),
					To:    node.ID,
					Properties: map[string]interface{}{},
				}
				allEdges = append(allEdges, createdEdge)
			}

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
// Schema: PRE_COMMIT_GRAPH_SPEC.md - PR node with CREATED and MERGED_AS edges
func (b *Builder) transformPR(pr database.PRData) (GraphNode, []GraphEdge, error) {
	var edges []GraphEdge

	// Parse raw data to extract author information
	var fullPR struct {
		User struct {
			Login string `json:"login"`
			Email string `json:"email"`
		} `json:"user"`
	}
	if err := json.Unmarshal(pr.RawData, &fullPR); err != nil {
		log.Printf("  ⚠️  Failed to parse PR author from raw data: %v", err)
	}

	// Parse raw data to extract branch information
	var fullPRData struct {
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
		Head struct {
			Ref string `json:"ref"`
		} `json:"head"`
	}
	if err := json.Unmarshal(pr.RawData, &fullPRData); err != nil {
		log.Printf("  ⚠️  Failed to parse PR branches from raw data: %v", err)
	}

	// Extract author email - GitHub API returns login, may not have email
	authorEmail := fullPR.User.Email
	if authorEmail == "" {
		// Fallback: use login@users.noreply.github.com
		authorEmail = fmt.Sprintf("%s@users.noreply.github.com", fullPR.User.Login)
	}
	// Normalize email to match Developer nodes
	authorEmail = normalizeGitHubEmail(authorEmail)

	// Create PR node
	// Schema: PRE_COMMIT_GRAPH_SPEC.md - PR with author_email, base_branch, head_branch, merge_commit_sha
	node := GraphNode{
		Label: "PR",
		ID:    fmt.Sprintf("pr:%d", pr.Number),
		Properties: map[string]interface{}{
			"number":       pr.Number,
			"title":        pr.Title,
			"body":         pr.Body,
			"state":        pr.State,
			"base_branch":  fullPRData.Base.Ref,
			"head_branch":  fullPRData.Head.Ref,
			"author_email": authorEmail, // Links to Developer (normalized)
			"created_at":   pr.CreatedAt.Unix(),
		},
	}

	if pr.MergedAt != nil {
		node.Properties["merged_at"] = pr.MergedAt.Unix()
	}

	if pr.MergeCommitSHA != nil {
		node.Properties["merge_commit_sha"] = *pr.MergeCommitSHA
	}

	// CREATED edge will be added in processPRs

	// Create MERGED_AS edge if merged (PR → Commit)
	// Schema: PRE_COMMIT_GRAPH_SPEC.md - MERGED_AS relationship
	if pr.Merged && pr.MergeCommitSHA != nil {
		mergedAsEdge := GraphEdge{
			Label:      "MERGED_AS",
			From:       node.ID,
			To:         fmt.Sprintf("commit:%s", *pr.MergeCommitSHA),
			Properties: map[string]interface{}{},
		}
		edges = append(edges, mergedAsEdge)
	}

	// NOTE: FIXES edges removed - Issue nodes not in PRE_COMMIT_GRAPH_SPEC.md

	return node, edges, nil
}

// processIssues transforms Issues from PostgreSQL to graph nodes
// Phase 2 of issue_ingestion_implementation_plan.md
// Creates Issue nodes with all properties from schema.md
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

// transformIssue converts an Issue into a graph node
// Schema reference: dev_docs/mvp/issue_ingestion_implementation_plan.md Phase 4.2
func (b *Builder) transformIssue(issue database.IssueData) (GraphNode, error) {
	// Parse labels from JSON array
	var labels []string
	if err := json.Unmarshal(issue.Labels, &labels); err != nil {
		log.Printf("  ⚠️  Failed to parse labels for issue #%d: %v", issue.Number, err)
		labels = []string{} // Default to empty array
	}

	// Create Issue node
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

	// Add closed_at if issue is closed
	if issue.ClosedAt != nil {
		node.Properties["closed_at"] = issue.ClosedAt.Unix()
	}

	return node, nil
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

// REMOVED: calculateOwnership - OWNS edges are now computed dynamically via queries
// Per IMPLEMENTATION_GAP_ANALYSIS.md:
// "Pre-computed OWNS edges contradict dynamic query philosophy"
// Ownership is now calculated on-demand using:
//   MATCH (d:Developer)-[:AUTHORED]->(c:Commit)-[:MODIFIED]->(f:File {path: $file_path})
// See: internal/risk/queries.go - QueryOwnership for the dynamic query

// linkCommitsToPRs creates IN_PR edges (Commit → PR)
// Schema: PRE_COMMIT_GRAPH_SPEC.md - IN_PR relationship
func (b *Builder) linkCommitsToPRs(ctx context.Context, repoID int64) (*BuildStats, error) {
	stats := &BuildStats{}

	// Get all processed PRs with their commit data
	prs, err := b.stagingDB.GetProcessedPRBranchData(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("failed to get PR data: %w", err)
	}

	// For each PR, we need to find all commits that are part of it
	// This is complex - for MVP, we'll link the merge commit via IN_PR
	// Full implementation would require fetching PR commits from GitHub API

	var edges []GraphEdge
	for _, pr := range prs {
		if pr.MergeCommitSHA != nil {
			// Link the merge commit to the PR
			inPREdge := GraphEdge{
				Label: "IN_PR",
				From:  fmt.Sprintf("commit:%s", *pr.MergeCommitSHA),
				To:    fmt.Sprintf("pr:%d", pr.Number),
				Properties: map[string]interface{}{},
			}
			edges = append(edges, inPREdge)
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
				return stats, fmt.Errorf("failed to create IN_PR edges (batch %d-%d): %w", i, end, err)
			}
		}

		stats.Edges = len(edges)
	}

	return stats, nil
}

// REMOVED: Deprecated methods (AddLayer2CoChangedEdges, AddLayer3IncidentNodes, AddLayer3CausedByEdges)
// These methods were marked deprecated and are now removed per IMPLEMENTATION_GAP_ANALYSIS.md
// - CO_CHANGED edges: Now computed dynamically via queries (see internal/risk/queries.go)
// - Incident nodes: Not part of MVP schema (deferred to post-MVP)
// - CAUSED_BY edges: Not part of MVP schema (deferred to post-MVP)
