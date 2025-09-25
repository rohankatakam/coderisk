package storage

import (
	"context"
	"errors"

	"github.com/coderisk/coderisk-go/internal/models"
)

// Common errors
var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

// Store defines the storage interface
type Store interface {
	// Repository operations
	SaveRepository(ctx context.Context, repo *models.Repository) error
	GetRepository(ctx context.Context, repoID string) (*models.Repository, error)

	// Commit operations
	SaveCommits(ctx context.Context, commits []*models.Commit) error
	GetCommits(ctx context.Context, repoID string, limit int) ([]*models.Commit, error)

	// File operations
	SaveFiles(ctx context.Context, files []*models.File) error
	GetFiles(ctx context.Context, repoID string) ([]*models.File, error)

	// Pull request operations
	SavePullRequests(ctx context.Context, prs []*models.PullRequest) error

	// Issue operations
	SaveIssues(ctx context.Context, issues []*models.Issue) error

	// Risk operations
	SaveRiskSketches(ctx context.Context, sketches []*models.RiskSketch) error
	GetRiskSketches(ctx context.Context, repoID string) ([]*models.RiskSketch, error)
	SaveRiskAssessment(ctx context.Context, assessment *models.RiskAssessment) error
	GetRiskAssessment(ctx context.Context, id string) (*models.RiskAssessment, error)

	// Cache operations
	SaveCacheMetadata(ctx context.Context, metadata *models.CacheMetadata) error
	GetCacheMetadata(ctx context.Context, repoID string) (*models.CacheMetadata, error)

	// Close connection
	Close() error
}
