package github

import (
	"context"
	"fmt"

	"github.com/rohankatakam/coderisk/internal/database"
)

// EntityType represents the type of GitHub entity
type EntityType string

const (
	EntityTypeIssue   EntityType = "Issue"
	EntityTypePR      EntityType = "PR"
	EntityTypeCommit  EntityType = "Commit"
	EntityTypeUnknown EntityType = "Unknown"
)

// EntityResolver resolves entity references to their types
type EntityResolver struct {
	stagingDB *database.StagingClient
}

// NewEntityResolver creates an entity resolver
func NewEntityResolver(stagingDB *database.StagingClient) *EntityResolver {
	return &EntityResolver{
		stagingDB: stagingDB,
	}
}

// ResolveEntity determines if a reference (e.g., "#42") is an Issue, PR, or Commit
// Returns the entity type and internal database ID
func (r *EntityResolver) ResolveEntity(ctx context.Context, repoID int64, referenceNumber int) (EntityType, int64, error) {
	// Try Issue first (most common for "Fixes #N")
	issueID, err := r.resolveIssue(ctx, repoID, referenceNumber)
	if err == nil {
		return EntityTypeIssue, issueID, nil
	}

	// Try PR second
	prID, err := r.resolvePR(ctx, repoID, referenceNumber)
	if err == nil {
		return EntityTypePR, prID, nil
	}

	// Not found in either table
	return EntityTypeUnknown, 0, fmt.Errorf("entity #%d not found in repo %d", referenceNumber, repoID)
}

// resolveIssue checks if the reference number exists in github_issues
func (r *EntityResolver) resolveIssue(ctx context.Context, repoID int64, number int) (int64, error) {
	// Query directly from github_issues table (not the unprocessed view)
	query := `
		SELECT id
		FROM github_issues
		WHERE repo_id = $1 AND number = $2
		LIMIT 1
	`

	var issueID int64
	err := r.stagingDB.QueryRow(ctx, query, repoID, number).Scan(&issueID)
	if err != nil {
		return 0, fmt.Errorf("issue #%d not found in repo %d: %w", number, repoID, err)
	}

	return issueID, nil
}

// resolvePR checks if the reference number exists in github_pull_requests
func (r *EntityResolver) resolvePR(ctx context.Context, repoID int64, number int) (int64, error) {
	// Query directly from github_pull_requests table (not the unprocessed view)
	query := `
		SELECT id
		FROM github_pull_requests
		WHERE repo_id = $1 AND number = $2
		LIMIT 1
	`

	var prID int64
	err := r.stagingDB.QueryRow(ctx, query, repoID, number).Scan(&prID)
	if err != nil {
		return 0, fmt.Errorf("PR #%d not found in repo %d: %w", number, repoID, err)
	}

	return prID, nil
}
