package agent

import (
	"context"

	"github.com/coderisk/coderisk-go/internal/incidents"
	"github.com/coderisk/coderisk-go/internal/temporal"
)

// RealTemporalClient implements TemporalClient using git history parsing
type RealTemporalClient struct {
	repoPath string
	commits  []temporal.Commit
}

// NewRealTemporalClient creates a temporal client for a repository
func NewRealTemporalClient(repoPath string) (*RealTemporalClient, error) {
	// Parse last 90 days of git history
	commits, err := temporal.ParseGitHistory(repoPath, 90)
	if err != nil {
		return nil, err
	}

	return &RealTemporalClient{
		repoPath: repoPath,
		commits:  commits,
	}, nil
}

// GetCoChangedFiles returns files that frequently change together
func (r *RealTemporalClient) GetCoChangedFiles(ctx context.Context, filePath string, minFrequency float64) ([]temporal.CoChangeResult, error) {
	return temporal.GetCoChangedFiles(ctx, filePath, minFrequency, r.commits)
}

// GetOwnershipHistory returns ownership information for a file
func (r *RealTemporalClient) GetOwnershipHistory(ctx context.Context, filePath string) (*temporal.OwnershipHistory, error) {
	return temporal.GetOwnershipHistory(ctx, filePath, r.commits)
}

// RealIncidentsClient implements IncidentsClient using PostgreSQL database
type RealIncidentsClient struct {
	db *incidents.Database
}

// NewRealIncidentsClient creates an incidents client
func NewRealIncidentsClient(db *incidents.Database) *RealIncidentsClient {
	return &RealIncidentsClient{db: db}
}

// GetIncidentStats returns incident statistics for a file
func (r *RealIncidentsClient) GetIncidentStats(ctx context.Context, filePath string) (*incidents.IncidentStats, error) {
	return r.db.GetIncidentStats(ctx, filePath)
}

// SearchIncidents performs BM25 full-text search on incidents
func (r *RealIncidentsClient) SearchIncidents(ctx context.Context, query string, limit int) ([]incidents.SearchResult, error) {
	return r.db.SearchIncidents(ctx, query, limit)
}
