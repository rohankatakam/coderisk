package graph

import (
	"context"
	"fmt"
	"log"

	"github.com/rohankatakam/coderisk/internal/database"
)

// IssueLinker creates Issue nodes and FIXED_BY edges in the graph
// Reference: REVISED_MVP_STRATEGY.md - Phase 3: Merge and Create Edges
type IssueLinker struct {
	stagingDB *database.StagingClient
	backend   Backend
}

// NewIssueLinker creates an issue linker
func NewIssueLinker(stagingDB *database.StagingClient, backend Backend) *IssueLinker {
	return &IssueLinker{
		stagingDB: stagingDB,
		backend:   backend,
	}
}

// LinkIssues creates FIXED_BY edges based on extracted references
// Note: Issue nodes are already created by processIssues in builder.go
// Reference: REVISED_MVP_STRATEGY.md - Bidirectional confidence boost
func (l *IssueLinker) LinkIssues(ctx context.Context, repoID int64) (*BuildStats, error) {
	log.Printf("🔗 Linking issues to commits and PRs...")
	stats := &BuildStats{}

	// Create FIXED_BY edges from extracted references
	edgeStats, err := l.createFixedByEdges(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("failed to create FIXED_BY edges: %w", err)
	}
	stats.Edges += edgeStats.Edges
	log.Printf("  ✓ Created %d FIXED_BY edges", edgeStats.Edges)

	return stats, nil
}

// createIssueNodes creates Issue nodes in the graph
func (l *IssueLinker) createIssueNodes(ctx context.Context, repoID int64) (*BuildStats, error) {
	stats := &BuildStats{}

	// Fetch all issues (both processed and unprocessed)
	issues, err := l.stagingDB.FetchUnprocessedIssues(ctx, repoID, 10000)
	if err != nil {
		return stats, fmt.Errorf("failed to fetch issues: %w", err)
	}

	if len(issues) == 0 {
		return stats, nil
	}

	// Create nodes in batches
	var nodes []GraphNode
	for _, issue := range issues {
		// Determine if this is a bug/incident
		isBug := l.isBugIssue(issue)

		// Create Issue node
		// Schema: REVISED_MVP_STRATEGY.md - Issue nodes with bug classification
		node := GraphNode{
			Label: "Issue",
			ID:    fmt.Sprintf("issue:%d", issue.Number),
			Properties: map[string]interface{}{
				"number":     issue.Number,
				"title":      issue.Title,
				"state":      issue.State,
				"is_bug":     isBug,
				"created_at": issue.CreatedAt.Unix(),
			},
		}

		if issue.ClosedAt != nil {
			node.Properties["closed_at"] = issue.ClosedAt.Unix()
		}

		nodes = append(nodes, node)
	}

	// Create all nodes
	if len(nodes) > 0 {
		if _, err := l.backend.CreateNodes(ctx, nodes); err != nil {
			return stats, fmt.Errorf("failed to create nodes: %w", err)
		}
		stats.Nodes = len(nodes)
	}

	return stats, nil
}

// createFixedByEdges creates FIXED_BY edges based on extracted references
func (l *IssueLinker) createFixedByEdges(ctx context.Context, repoID int64) (*BuildStats, error) {
	stats := &BuildStats{}

	// Get all extracted references
	refs, err := l.stagingDB.GetIssueCommitRefs(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("failed to get references: %w", err)
	}

	if len(refs) == 0 {
		return stats, nil
	}

	// Merge bidirectional references and boost confidence
	mergedRefs := l.mergeReferences(refs)

	// Create edges
	var edges []GraphEdge
	for _, ref := range mergedRefs {
		// Only create edges for "fixes" action (not "mentions")
		if ref.Action != "fixes" {
			continue
		}

		// Only create edges with sufficient confidence
		if ref.Confidence < 0.75 {
			continue
		}

		issueID := fmt.Sprintf("issue:%d", ref.IssueNumber)

		// Create edge to commit if available
		if ref.CommitSHA != nil {
			commitID := fmt.Sprintf("commit:%s", *ref.CommitSHA)

			edge := GraphEdge{
				Label: "FIXED_BY",
				From:  issueID,
				To:    commitID,
				Properties: map[string]interface{}{
					"confidence":       ref.Confidence,
					"detection_method": ref.DetectionMethod,
					"extracted_from":   ref.ExtractedFrom,
				},
			}

			edges = append(edges, edge)
		}

		// Create edge to PR if available
		if ref.PRNumber != nil {
			prID := fmt.Sprintf("pr:%d", *ref.PRNumber)

			edge := GraphEdge{
				Label: "FIXED_BY",
				From:  issueID,
				To:    prID,
				Properties: map[string]interface{}{
					"confidence":       ref.Confidence,
					"detection_method": ref.DetectionMethod,
					"extracted_from":   ref.ExtractedFrom,
				},
			}

			edges = append(edges, edge)
		}
	}

	// Create all edges
	if len(edges) > 0 {
		if err := l.backend.CreateEdges(ctx, edges); err != nil {
			return stats, fmt.Errorf("failed to create edges: %w", err)
		}
		stats.Edges = len(edges)
	}

	return stats, nil
}

// mergeReferences merges bidirectional references and boosts confidence
// Reference: REVISED_MVP_STRATEGY.md - Bidirectional confidence boost
func (l *IssueLinker) mergeReferences(refs []database.IssueCommitRef) []database.IssueCommitRef {
	// Group by (issue_number, commit_sha, pr_number) tuple
	type refKey struct {
		IssueNumber int
		CommitSHA   string
		PRNumber    int
	}

	refMap := make(map[refKey]*database.IssueCommitRef)

	for _, ref := range refs {
		commitSHA := ""
		if ref.CommitSHA != nil {
			commitSHA = *ref.CommitSHA
		}

		prNumber := 0
		if ref.PRNumber != nil {
			prNumber = *ref.PRNumber
		}

		key := refKey{
			IssueNumber: ref.IssueNumber,
			CommitSHA:   commitSHA,
			PRNumber:    prNumber,
		}

		existing, exists := refMap[key]
		if !exists {
			// First time seeing this reference
			refCopy := ref
			refMap[key] = &refCopy
		} else {
			// Duplicate found - apply bidirectional boost
			// Keep highest confidence + boost
			if ref.Confidence > existing.Confidence {
				existing.Confidence = ref.Confidence
			}

			// If different detection methods, it's bidirectional
			if ref.DetectionMethod != existing.DetectionMethod {
				existing.DetectionMethod = "bidirectional"
				// Boost confidence by 0.05 (capped at 0.95)
				existing.Confidence += 0.05
				if existing.Confidence > 0.95 {
					existing.Confidence = 0.95
				}
			}

			// Prefer "fixes" over "mentions"
			if ref.Action == "fixes" {
				existing.Action = "fixes"
			}
		}
	}

	// Convert map back to slice
	merged := make([]database.IssueCommitRef, 0, len(refMap))
	for _, ref := range refMap {
		merged = append(merged, *ref)
	}

	return merged
}

// isBugIssue determines if an issue is a bug based on labels
func (l *IssueLinker) isBugIssue(issue database.IssueData) bool {
	// Parse labels JSON
	var labels []string
	if err := issue.Labels.UnmarshalJSON(issue.Labels); err == nil {
		for _, label := range labels {
			if label == "bug" || label == "Bug" || label == "defect" || label == "issue" {
				return true
			}
		}
	}

	return false
}
