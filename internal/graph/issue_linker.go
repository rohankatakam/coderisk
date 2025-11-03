package graph

import (
	"context"
	"fmt"
	"log"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/github"
)

// IssueLinker creates Issue nodes and FIXED_BY/ASSOCIATED_WITH edges in the graph
// Reference: REVISED_MVP_STRATEGY.md - Phase 3: Merge and Create Edges
type IssueLinker struct {
	stagingDB      *database.StagingClient
	backend        Backend
	entityResolver *github.EntityResolver
}

// NewIssueLinker creates an issue linker
func NewIssueLinker(stagingDB *database.StagingClient, backend Backend) *IssueLinker {
	return &IssueLinker{
		stagingDB:      stagingDB,
		backend:        backend,
		entityResolver: github.NewEntityResolver(stagingDB),
	}
}

// LinkIssues creates FIXED_BY and ASSOCIATED_WITH edges based on extracted references
// Note: Issue nodes are already created by processIssues in builder.go
// Reference: REVISED_MVP_STRATEGY.md - Bidirectional confidence boost
func (l *IssueLinker) LinkIssues(ctx context.Context, repoID int64) (*BuildStats, error) {
	log.Printf("🔗 Linking issues to commits and PRs...")
	stats := &BuildStats{}

	// Create edges from extracted references (FIXED_BY + ASSOCIATED_WITH)
	edgeStats, err := l.createIncidentEdges(ctx, repoID)
	if err != nil {
		return stats, fmt.Errorf("failed to create incident edges: %w", err)
	}
	stats.Edges += edgeStats.Edges
	log.Printf("  ✓ Created %d incident edges", edgeStats.Edges)

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

// createIncidentEdges creates FIXED_BY and ASSOCIATED_WITH edges with entity resolution
func (l *IssueLinker) createIncidentEdges(ctx context.Context, repoID int64) (*BuildStats, error) {
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

	// Create edges with entity resolution
	var edges []GraphEdge
	skippedLowConf := 0
	skippedNotFound := 0

	for _, ref := range mergedRefs {
		// Skip low-confidence references
		if ref.Confidence < 0.4 {
			skippedLowConf++
			continue
		}

		// Resolve entity type: is ref.IssueNumber actually an Issue or PR?
		entityType, _, err := l.entityResolver.ResolveEntity(ctx, repoID, ref.IssueNumber)
		if err != nil {
			// Entity not found (likely outside 90-day window)
			fmt.Printf("  ⚠️  Entity #%d not found in database: %v\n", ref.IssueNumber, err)
			skippedNotFound++
			continue
		}

		fmt.Printf("  ✓ Entity #%d resolved as %s (action: %s, confidence: %.2f)\n",
			ref.IssueNumber, entityType, ref.Action, ref.Confidence)

		// Determine edge label and source node
		var edgeLabel string
		var sourceID string

		if entityType == github.EntityTypeIssue && ref.Action == "fixes" {
			// Issue resolved by commit/PR → use specific FIXED_BY edge
			edgeLabel = "FIXED_BY"
			sourceID = fmt.Sprintf("issue:%d", ref.IssueNumber)
			fmt.Printf("    → Creating FIXED_BY edge from %s\n", sourceID)
		} else {
			// PR/other entity or non-fix action → use generic ASSOCIATED_WITH edge
			edgeLabel = "ASSOCIATED_WITH"
			if entityType == github.EntityTypePR {
				sourceID = fmt.Sprintf("pr:%d", ref.IssueNumber)
				fmt.Printf("    → Creating ASSOCIATED_WITH edge from PR#%d\n", ref.IssueNumber)
			} else {
				sourceID = fmt.Sprintf("issue:%d", ref.IssueNumber)
				fmt.Printf("    → Creating ASSOCIATED_WITH edge from Issue#%d\n", ref.IssueNumber)
			}
		}

		// Create edge to commit if available
		if ref.CommitSHA != nil {
			commitID := fmt.Sprintf("commit:%s", *ref.CommitSHA)

			edge := GraphEdge{
				Label: edgeLabel,
				From:  sourceID,
				To:    commitID,
				Properties: map[string]interface{}{
					"relationship_type": ref.Action,
					"confidence":        ref.Confidence,
					"detected_via":      ref.DetectionMethod,
					"rationale":         fmt.Sprintf("Extracted from %s", ref.ExtractedFrom),
					"evidence":          ref.Evidence, // Evidence tags for CLQS calculation
				},
			}

			edges = append(edges, edge)
		}

		// Create edge to PR if available
		if ref.PRNumber != nil {
			prID := fmt.Sprintf("pr:%d", *ref.PRNumber)

			edge := GraphEdge{
				Label: edgeLabel,
				From:  sourceID,
				To:    prID,
				Properties: map[string]interface{}{
					"relationship_type": ref.Action,
					"confidence":        ref.Confidence,
					"detected_via":      ref.DetectionMethod,
					"rationale":         fmt.Sprintf("Extracted from %s", ref.ExtractedFrom),
					"evidence":          ref.Evidence, // Evidence tags for CLQS calculation
				},
			}

			edges = append(edges, edge)
		}
	}

	// Create all edges
	fmt.Printf("\n  📊 Summary: %d references processed\n", len(mergedRefs))
	fmt.Printf("    • Skipped (low confidence): %d\n", skippedLowConf)
	fmt.Printf("    • Skipped (not found): %d\n", skippedNotFound)
	fmt.Printf("    • Edges to create: %d\n\n", len(edges))

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
