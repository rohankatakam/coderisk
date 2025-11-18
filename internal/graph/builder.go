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
	"github.com/rohankatakam/coderisk/internal/ingestion"
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

// buildCompositeNodeID creates a multi-repo safe node ID with format: <repo_id>:<type>:<identifier>
// This prevents node ID collisions across repositories.
//
// Examples:
//   - buildCompositeNodeID(1, "issue", "175") → "1:issue:175"
//   - buildCompositeNodeID(2, "commit", "abc123") → "2:commit:abc123"
//   - buildCompositeNodeID(1, "file", "src/main.go") → "1:file:src/main.go"
//
// Performance: Uses strings.Builder for efficient string concatenation
func buildCompositeNodeID(repoID int64, nodeType, identifier string) string {
	var buf strings.Builder
	// Pre-allocate capacity: repoID (max 20 chars) + nodeType + identifier + 2 colons
	buf.Grow(len(nodeType) + len(identifier) + 22)
	buf.WriteString(strconv.FormatInt(repoID, 10))
	buf.WriteByte(':')
	buf.WriteString(nodeType)
	buf.WriteByte(':')
	buf.WriteString(identifier)
	return buf.String()
}

// parseCompositeNodeID extracts repo_id, node type, and identifier from a composite ID
// Supports both new 3-part format and legacy 2-part format for backward compatibility.
//
// Examples:
//   - parseCompositeNodeID("1:issue:175") → (1, "issue", "175", nil)
//   - parseCompositeNodeID("issue:175") → (0, "issue", "175", nil) // legacy format
//
// Returns error if the ID format is invalid
func parseCompositeNodeID(compositeID string) (repoID int64, nodeType, identifier string, err error) {
	parts := strings.SplitN(compositeID, ":", 3)

	if len(parts) == 3 {
		// New format: <repo_id>:<type>:<identifier>
		repoID, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, "", "", fmt.Errorf("invalid repo_id in composite ID %q: %w", compositeID, err)
		}
		nodeType = parts[1]
		identifier = parts[2]
		return repoID, nodeType, identifier, nil
	} else if len(parts) == 2 {
		// Legacy format: <type>:<identifier> (backward compatibility, assumes repo_id=0)
		return 0, parts[0], parts[1], nil
	}

	return 0, "", "", fmt.Errorf("invalid composite ID format %q: expected <repo_id>:<type>:<id> or <type>:<id>", compositeID)
}

// Builder orchestrates graph construction from PostgreSQL staging tables
// Reference: dev_docs/03-implementation/integration_guides/layers_2_3_graph_construction.md
type Builder struct {
	stagingDB    *database.StagingClient
	backend      Backend
	identityRepo *database.FileIdentityRepository
}

// NewBuilder creates a graph builder instance
func NewBuilder(stagingDB *database.StagingClient, backend Backend) *Builder {
	return &Builder{
		stagingDB:    stagingDB,
		backend:      backend,
		identityRepo: database.NewFileIdentityRepository(stagingDB.DB()),
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

	// Fetch repository metadata for composite node IDs
	repoInfo, err := b.stagingDB.GetRepositoryInfo(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("failed to fetch repository info: %w", err)
	}
	repoFullName := repoInfo.FullName

	// Process commits (creates Commit, Developer nodes + AUTHORED, MODIFIED edges)
	commitStats, err := b.processCommits(ctx, repoID, repoFullName, repoPath)
	if err != nil {
		return stats, fmt.Errorf("process commits failed: %w", err)
	}
	stats.Nodes += commitStats.Nodes
	stats.Edges += commitStats.Edges
	log.Printf("  ✓ Processed commits: %d nodes, %d edges", commitStats.Nodes, commitStats.Edges)

	// Process PRs (creates PR nodes + CREATED, MERGED_AS edges)
	prStats, err := b.processPRs(ctx, repoID, repoFullName)
	if err != nil {
		return stats, fmt.Errorf("process PRs failed: %w", err)
	}
	stats.Nodes += prStats.Nodes
	stats.Edges += prStats.Edges
	log.Printf("  ✓ Processed PRs: %d nodes, %d edges", prStats.Nodes, prStats.Edges)

	// NOTE: OWNS edges removed - ownership is now computed dynamically via queries
	// See: IMPLEMENTATION_GAP_ANALYSIS.md - Ownership should be dynamic from AUTHORED + MODIFIED edges
	// No pre-computed OWNS edges in MVP spec

	// Link PRs to merge commits via MERGED_AS edges
	commitPRStats, err := b.linkPRsToMergeCommits(ctx, repoID, repoFullName)
	if err != nil {
		return stats, fmt.Errorf("link PRs to merge commits failed: %w", err)
	}
	stats.Edges += commitPRStats.Edges
	log.Printf("  ✓ Linked PRs to merge commits: %d edges", commitPRStats.Edges)

	// Process Issues (creates Issue nodes)
	// Phase 2 of issue_ingestion_implementation_plan.md
	issueStats, err := b.processIssues(ctx, repoID, repoFullName)
	if err != nil {
		return stats, fmt.Errorf("process issues failed: %w", err)
	}
	stats.Nodes += issueStats.Nodes
	log.Printf("  ✓ Processed issues: %d nodes", issueStats.Nodes)

	// Create 100% confidence edges from timeline events (REFERENCES, CLOSED_BY)
	// This must run BEFORE ASSOCIATED_WITH to prevent collisions
	// Reference: Gap analysis 2025-11-13 - Timeline edges take priority
	timelineStats, err := b.createTimelineEdges(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("create timeline edges failed: %w", err)
	}
	stats.Edges += timelineStats.Edges
	log.Printf("  ✓ Created timeline edges: %d edges (100%% confidence)", timelineStats.Edges)

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
	validatedLinkStats, err := b.loadIssuePRLinksIntoGraph(ctx, repoID, repoFullName)
	if err != nil {
		log.Printf("  ⚠️  Warning: Failed to load validated links: %v", err)
		// Don't fail entire build, continue with old system links
	} else {
		stats.Edges += validatedLinkStats.Edges
		log.Printf("  ✓ Loaded %d validated issue-PR links", validatedLinkStats.Edges)
	}

	return stats, nil
}

// RunPipeline2 executes Pipeline 2 code-block atomization for a repository
// This is an optional enhancement that extracts fine-grained code blocks from commits
// Reference: AGENT-P2A and AGENT-P2B implementation
//
// Requirements:
//   - repoID: Repository identifier
//   - repoPath: Absolute path to cloned repository (for git diff access)
//   - llmClient: LLM client for semantic extraction (must be enabled)
//   - rawDB: Raw SQL database connection (stagingDB.DB() accessor)
//   - neoDriver: Neo4j driver for graph operations
//   - neoDatabase: Neo4j database name
func (b *Builder) RunPipeline2(ctx context.Context, repoID int64, repoPath string, llmClient interface{}, rawDB interface{}, neoDriver interface{}, neoDatabase string) error {
	log.Printf("🔬 Running Pipeline 2: Code-Block Atomization...")

	// Import atomizer package (will be resolved by Go compiler)
	// We use interface{} types here to avoid circular dependencies
	// The actual implementation will type-assert these at runtime

	// Note: This is a placeholder for the actual implementation
	// The real integration will happen in the init command
	log.Printf("  ⚠️  Pipeline 2 not yet fully integrated into Builder")
	log.Printf("  → Use standalone process-repo-v2 command for now")

	return nil
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
	// Fetch repository metadata for composite node IDs
	repoInfo, err := b.stagingDB.GetRepositoryInfo(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository info: %w", err)
	}
	return b.loadIssuePRLinksIntoGraph(ctx, repoID, repoInfo.FullName)
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
func (b *Builder) loadIssuePRLinksIntoGraph(ctx context.Context, repoID int64, repoFullName string) (*BuildStats, error) {
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

		// Create edge with comprehensive properties using composite IDs
		edge := GraphEdge{
			Label: edgeLabel,
			From:  buildCompositeNodeID(repoID, "issue", strconv.Itoa(link.IssueNumber)),
			To:    buildCompositeNodeID(repoID, "pr", strconv.Itoa(link.PRNumber)),
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
		deleteQuery := fmt.Sprintf(`
			MATCH (i:Issue)-[r]->(pr:PR)
			WHERE i.repo_id = %d AND pr.repo_id = %d
			AND (r.created_from IS NULL OR r.created_from <> 'validated_link')
			DELETE r
		`, repoID, repoID)
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
func (b *Builder) processCommits(ctx context.Context, repoID int64, repoFullName, repoPath string) (*BuildStats, error) {
	batchSize := 100
	stats := &BuildStats{}

	// Load file identity mapping once for efficient path resolution
	// This maps historical_path -> canonical_path for all files in the repository
	log.Printf("  Loading file identity map for repo %d...", repoID)
	identityMap, err := b.identityRepo.GetByRepoID(ctx, repoID)
	if err != nil {
		log.Printf("  ⚠️  Failed to load file identity map: %v (continuing without canonical paths)", err)
		identityMap = make(map[string]*ingestion.FileIdentity) // Empty map, fall back to historical paths
	} else {
		log.Printf("  ✓ Loaded %d file identity mappings", len(identityMap))
	}

	// Build reverse lookup: historical_path -> canonical_path
	pathResolutionMap := make(map[string]string)
	for canonicalPath, identity := range identityMap {
		for _, historicalPath := range identity.HistoricalPaths {
			pathResolutionMap[historicalPath] = canonicalPath
		}
	}

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
			nodes, edges, err := b.transformCommit(commit, repoID, repoFullName, repoPath, pathResolutionMap)
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
func (b *Builder) transformCommit(commit database.CommitData, repoID int64, repoFullName, repoPath string, pathResolutionMap map[string]string) ([]GraphNode, []GraphEdge, error) {
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

	// 1. Create Commit node with composite ID and repo properties
	// Schema: PRE_COMMIT_GRAPH_SPEC.md - Commit with on_default_branch property
	commitNode := GraphNode{
		Label: "Commit",
		ID:    buildCompositeNodeID(repoID, "commit", commit.SHA),
		Properties: map[string]interface{}{
			"repo_id":            repoID,
			"repo_full_name":     repoFullName,
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

	// 2. Create Developer node with composite ID and repo properties
	// Schema: PRE_COMMIT_GRAPH_SPEC.md - Developer with email (PRIMARY KEY), name
	// Normalize email to ensure consistency with PR authors
	normalizedEmail := normalizeGitHubEmail(commit.AuthorEmail)
	developerNode := GraphNode{
		Label: "Developer",
		ID:    buildCompositeNodeID(repoID, "developer", normalizedEmail),
		Properties: map[string]interface{}{
			"repo_id":        repoID,
			"repo_full_name": repoFullName,
			"email":          normalizedEmail,
			"name":           commit.AuthorName,
			"last_active":    commit.AuthorDate.Unix(), // Most recent commit timestamp
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
	// NEW: Resolve historical paths to canonical paths using file_identity_map
	// Reference: ingestion_aws.md Pipeline 1.0 - File Identity Resolution
	for _, file := range fullCommit.Files {
		// Historical path from GitHub API (path at commit time)
		historicalPath := file.Filename

		// Resolve to canonical path (current path at HEAD)
		canonicalPath := historicalPath // Default: use historical path if no mapping found
		if resolvedPath, exists := pathResolutionMap[historicalPath]; exists {
			canonicalPath = resolvedPath
		}

		// Create File node with CANONICAL path as the node ID
		// This ensures all historical instances of the same logical file link to one canonical node
		fileID := buildCompositeNodeID(repoID, "file", canonicalPath)
		fileNode := GraphNode{
			Label: "File",
			ID:    fileID,
			Properties: map[string]interface{}{
				"repo_id":         repoID,
				"repo_full_name":  repoFullName,
				"path":            canonicalPath,         // Required by Neo4j constraint
				"canonical_path":  canonicalPath,         // Current path at HEAD
				"path_at_commit":  historicalPath,        // Path at this commit (may differ due to renames)
				"is_renamed":      historicalPath != canonicalPath, // True if file was renamed since this commit
			},
		}
		nodes = append(nodes, fileNode)

		// Create MODIFIED edge
		modifiedEdge := GraphEdge{
			Label: "MODIFIED",
			From:  commitNode.ID,
			To:    fileID,
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
func (b *Builder) processPRs(ctx context.Context, repoID int64, repoFullName string) (*BuildStats, error) {
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
			node, edges, err := b.transformPR(pr, repoID, repoFullName)
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
				// Normalize email to match Developer node IDs
				authorEmail = normalizeGitHubEmail(authorEmail)

				// Create Developer node for PR author (will be merged with existing if present)
				developerNode := GraphNode{
					Label: "Developer",
					ID:    buildCompositeNodeID(repoID, "developer", authorEmail),
					Properties: map[string]interface{}{
						"email":          authorEmail,
						"name":           fullPR.User.Login, // Use login as name initially
						"last_active":    pr.CreatedAt.Unix(),
					},
				}
				allNodes = append(allNodes, developerNode)

				createdEdge := GraphEdge{
					Label: "CREATED",
					From:  developerNode.ID,
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
func (b *Builder) transformPR(pr database.PRData, repoID int64, repoFullName string) (GraphNode, []GraphEdge, error) {
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

	// Create PR node with composite ID and repo properties
	// Schema: PRE_COMMIT_GRAPH_SPEC.md - PR with author_email, base_branch, head_branch, merge_commit_sha
	node := GraphNode{
		Label: "PR",
		ID:    buildCompositeNodeID(repoID, "pr", strconv.Itoa(pr.Number)),
		Properties: map[string]interface{}{
			"repo_id":        repoID,
			"repo_full_name": repoFullName,
			"number":         pr.Number,
			"title":          pr.Title,
			"body":           pr.Body,
			"state":          pr.State,
			"base_branch":    fullPRData.Base.Ref,
			"head_branch":    fullPRData.Head.Ref,
			"author_email":   authorEmail, // Links to Developer (normalized)
			"created_at":     pr.CreatedAt.Unix(),
		},
	}

	if pr.MergedAt != nil {
		node.Properties["merged_at"] = pr.MergedAt.Unix()
	}

	if pr.MergeCommitSHA != nil {
		node.Properties["merge_commit_sha"] = *pr.MergeCommitSHA
	}

	// CREATED edge will be added in processPRs
	// MERGED_AS edge is now created in linkPRsToMergeCommits() function

	return node, edges, nil
}

// processIssues transforms Issues from PostgreSQL to graph nodes
// Phase 2 of issue_ingestion_implementation_plan.md
// Creates Issue nodes with all properties from schema.md
func (b *Builder) processIssues(ctx context.Context, repoID int64, repoFullName string) (*BuildStats, error) {
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
			node, err := b.transformIssue(issue, repoID, repoFullName)
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
func (b *Builder) transformIssue(issue database.IssueData, repoID int64, repoFullName string) (GraphNode, error) {
	// Parse labels from JSON array
	var labels []string
	if err := json.Unmarshal(issue.Labels, &labels); err != nil {
		log.Printf("  ⚠️  Failed to parse labels for issue #%d: %v", issue.Number, err)
		labels = []string{} // Default to empty array
	}

	// Create Issue node with composite ID and repo properties
	node := GraphNode{
		Label: "Issue",
		ID:    buildCompositeNodeID(repoID, "issue", strconv.Itoa(issue.Number)),
		Properties: map[string]interface{}{
			"repo_id":        repoID,
			"repo_full_name": repoFullName,
			"number":         issue.Number,
			"title":          issue.Title,
			"body":           issue.Body,
			"state":          issue.State,
			"labels":         labels,
			"created_at":     issue.CreatedAt.Unix(),
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

// linkPRsToMergeCommits creates MERGED_AS edges (PR → Commit)
// This links pull requests to their merge commits, showing which commit a PR was merged as.
// Direction: PR → Commit (clearer than the old IN_PR naming)
func (b *Builder) linkPRsToMergeCommits(ctx context.Context, repoID int64, repoFullName string) (*BuildStats, error) {
	stats := &BuildStats{}

	// Get all processed PRs with their commit data
	prs, err := b.stagingDB.GetProcessedPRBranchData(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("failed to get PR data: %w", err)
	}

	// For each PR with a merge commit, create MERGED_AS edge
	var edges []GraphEdge
	for _, pr := range prs {
		if pr.MergeCommitSHA != nil {
			// Link the PR to its merge commit using composite IDs
			mergedAsEdge := GraphEdge{
				Label: "MERGED_AS",
				From:  buildCompositeNodeID(repoID, "pr", strconv.Itoa(pr.Number)),
				To:    buildCompositeNodeID(repoID, "commit", *pr.MergeCommitSHA),
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
				return stats, fmt.Errorf("failed to create IN_PR edges (batch %d-%d): %w", i, end, err)
			}
		}

		stats.Edges = len(edges)
	}

	return stats, nil
}

// createTimelineEdges creates 100% confidence edges from GitHub timeline events
// This implements the missing timeline edge system identified in the gap analysis.
//
// Creates two types of edges with 100% confidence:
// 1. REFERENCES: Issue → PR (from cross-referenced events)
// 2. CLOSED_BY: Issue → Commit (from closed events with source_sha)
//
// These edges have higher confidence than ASSOCIATED_WITH edges because they come
// directly from GitHub's timeline API, not from LLM extraction or temporal correlation.
//
// Reference: Gap analysis 2025-11-13 - Timeline events are fetched but never converted to edges
func (b *Builder) createTimelineEdges(ctx context.Context, repoID int64) (*BuildStats, error) {
	stats := &BuildStats{}

	log.Printf("  Creating 100% confidence edges from timeline events...")

	// Query timeline events for cross-references and closures
	// We need to query the database directly since there's no method to get all timeline events
	query := `
		SELECT
			gi.number as issue_number,
			gte.event_type,
			gte.source_type,
			gte.source_number,
			gte.source_sha
		FROM github_issue_timeline gte
		JOIN github_issues gi ON gte.issue_id = gi.id
		WHERE gi.repo_id = $1
		  AND (
		    (gte.event_type = 'cross-referenced' AND gte.source_type = 'pull_request' AND gte.source_number IS NOT NULL)
		    OR
		    (gte.event_type = 'closed' AND gte.source_sha IS NOT NULL)
		  )
	`

	rows, err := b.stagingDB.Query(ctx, query, repoID)
	if err != nil {
		return stats, fmt.Errorf("failed to query timeline events: %w", err)
	}
	defer rows.Close()

	var edges []GraphEdge
	referencesCount := 0
	closedByCount := 0

	for rows.Next() {
		var issueNumber int
		var eventType string
		var sourceType *string
		var sourceNumber *int
		var sourceSHA *string

		err := rows.Scan(&issueNumber, &eventType, &sourceType, &sourceNumber, &sourceSHA)
		if err != nil {
			log.Printf("  ⚠️  Failed to scan timeline event: %v", err)
			continue
		}

		if eventType == "cross-referenced" && sourceType != nil && *sourceType == "pull_request" && sourceNumber != nil {
			// Create REFERENCES edge: Issue → PR
			edge := GraphEdge{
				Label: "REFERENCES",
				From:  buildCompositeNodeID(repoID, "issue", strconv.Itoa(issueNumber)),
				To:    buildCompositeNodeID(repoID, "pr", strconv.Itoa(*sourceNumber)),
				Properties: map[string]interface{}{
					"source":     "github_timeline",
					"confidence": 1.0, // 100% confidence - from GitHub API
					"event_type": "cross-referenced",
				},
			}
			edges = append(edges, edge)
			referencesCount++
		} else if eventType == "closed" && sourceSHA != nil {
			// Create CLOSED_BY edge: Issue → Commit
			edge := GraphEdge{
				Label: "CLOSED_BY",
				From:  buildCompositeNodeID(repoID, "issue", strconv.Itoa(issueNumber)),
				To:    buildCompositeNodeID(repoID, "commit", *sourceSHA),
				Properties: map[string]interface{}{
					"source":     "github_timeline",
					"confidence": 1.0, // 100% confidence - from GitHub API
					"event_type": "closed",
				},
			}
			edges = append(edges, edge)
			closedByCount++
		}
	}

	if err = rows.Err(); err != nil {
		return stats, fmt.Errorf("error iterating timeline events: %w", err)
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
				return stats, fmt.Errorf("failed to create timeline edges (batch %d-%d): %w", i, end, err)
			}
		}

		stats.Edges = len(edges)
		log.Printf("    ✓ Created %d REFERENCES edges (Issue → PR)", referencesCount)
		log.Printf("    ✓ Created %d CLOSED_BY edges (Issue → Commit)", closedByCount)
	} else {
		log.Printf("    ℹ️  No timeline events found to create edges")
	}

	return stats, nil
}

// REMOVED: Deprecated methods (AddLayer2CoChangedEdges, AddLayer3IncidentNodes, AddLayer3CausedByEdges)
// These methods were marked deprecated and are now removed per IMPLEMENTATION_GAP_ANALYSIS.md
// - CO_CHANGED edges: Now computed dynamically via queries (see internal/risk/queries.go)
// - Incident nodes: Not part of MVP schema (deferred to post-MVP)
// - CAUSED_BY edges: Not part of MVP schema (deferred to post-MVP)
