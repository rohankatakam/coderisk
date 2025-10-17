package github

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rohankatakam/coderisk/internal/models"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// Extractor orchestrates GitHub data extraction
type Extractor struct {
	client *Client
	logger *logrus.Logger
}

// NewExtractor creates a new GitHub extractor
func NewExtractor(client *Client, logger *logrus.Logger) *Extractor {
	return &Extractor{
		client: client,
		logger: logger,
	}
}

// ExtractResult contains all extracted data from a repository
type ExtractResult struct {
	Repository   *models.Repository
	Commits      []*models.Commit
	Files        []*models.File
	PullRequests []*models.PullRequest
	Issues       []*models.Issue
	ExtractedAt  time.Time
}

// ExtractRepository performs complete repository extraction
func (e *Extractor) ExtractRepository(ctx context.Context, owner, name string) (*ExtractResult, error) {
	startTime := time.Now()
	e.logger.WithFields(logrus.Fields{
		"owner": owner,
		"name":  name,
	}).Info("Starting repository extraction")

	result := &ExtractResult{
		ExtractedAt: startTime,
	}

	// Use errgroup for parallel extraction
	g, ctx := errgroup.WithContext(ctx)

	// Extract repository metadata
	g.Go(func() error {
		repo, err := e.client.FetchRepository(ctx, owner, name)
		if err != nil {
			return fmt.Errorf("extract repository: %w", err)
		}
		result.Repository = repo
		e.logger.Info("Repository metadata extracted")
		return nil
	})

	// Extract commits (last 90 days by default)
	g.Go(func() error {
		since := time.Now().AddDate(0, -3, 0)
		commits, err := e.client.FetchCommits(ctx, owner, name, since)
		if err != nil {
			return fmt.Errorf("extract commits: %w", err)
		}
		result.Commits = commits
		e.logger.WithField("count", len(commits)).Info("Commits extracted")
		return nil
	})

	// Extract files
	g.Go(func() error {
		// Wait for repository to be fetched first
		time.Sleep(100 * time.Millisecond)
		files, err := e.client.FetchFiles(ctx, owner, name, "HEAD")
		if err != nil {
			return fmt.Errorf("extract files: %w", err)
		}
		result.Files = files
		e.logger.WithField("count", len(files)).Info("Files extracted")
		return nil
	})

	// Extract pull requests
	g.Go(func() error {
		prs, err := e.client.FetchPullRequests(ctx, owner, name, "all")
		if err != nil {
			return fmt.Errorf("extract pull requests: %w", err)
		}
		result.PullRequests = prs
		e.logger.WithField("count", len(prs)).Info("Pull requests extracted")
		return nil
	})

	// Extract issues
	g.Go(func() error {
		issues, err := e.client.FetchIssues(ctx, owner, name, nil)
		if err != nil {
			return fmt.Errorf("extract issues: %w", err)
		}
		result.Issues = issues
		e.logger.WithField("count", len(issues)).Info("Issues extracted")
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	duration := time.Since(startTime)
	e.logger.WithFields(logrus.Fields{
		"duration": duration.String(),
		"commits":  len(result.Commits),
		"files":    len(result.Files),
		"prs":      len(result.PullRequests),
		"issues":   len(result.Issues),
	}).Info("Repository extraction completed")

	return result, nil
}

// IncrementalExtract performs incremental extraction since last update
func (e *Extractor) IncrementalExtract(ctx context.Context, owner, name string, since time.Time) (*ExtractResult, error) {
	e.logger.WithFields(logrus.Fields{
		"owner": owner,
		"name":  name,
		"since": since,
	}).Info("Starting incremental extraction")

	result := &ExtractResult{
		ExtractedAt: time.Now(),
	}

	g, ctx := errgroup.WithContext(ctx)

	// Only extract new commits
	g.Go(func() error {
		commits, err := e.client.FetchCommits(ctx, owner, name, since)
		if err != nil {
			return fmt.Errorf("extract commits: %w", err)
		}
		result.Commits = commits
		e.logger.WithField("count", len(commits)).Info("New commits extracted")
		return nil
	})

	// Extract recently updated PRs
	g.Go(func() error {
		prs, err := e.client.FetchPullRequests(ctx, owner, name, "all")
		if err != nil {
			return fmt.Errorf("extract pull requests: %w", err)
		}

		// Filter for recently updated PRs
		var recentPRs []*models.PullRequest
		for _, pr := range prs {
			if pr.CreatedAt.After(since) ||
				(pr.MergedAt != nil && pr.MergedAt.After(since)) ||
				(pr.ClosedAt != nil && pr.ClosedAt.After(since)) {
				recentPRs = append(recentPRs, pr)
			}
		}
		result.PullRequests = recentPRs
		e.logger.WithField("count", len(recentPRs)).Info("Recent PRs extracted")
		return nil
	})

	// Extract recently updated issues
	g.Go(func() error {
		issues, err := e.client.FetchIssues(ctx, owner, name, nil)
		if err != nil {
			return fmt.Errorf("extract issues: %w", err)
		}

		// Filter for recently updated issues
		var recentIssues []*models.Issue
		for _, issue := range issues {
			if issue.CreatedAt.After(since) ||
				(issue.ClosedAt != nil && issue.ClosedAt.After(since)) {
				recentIssues = append(recentIssues, issue)
			}
		}
		result.Issues = recentIssues
		e.logger.WithField("count", len(recentIssues)).Info("Recent issues extracted")
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return result, nil
}

// ExtractFileChanges extracts changed files from commits
func (e *Extractor) ExtractFileChanges(ctx context.Context, commits []*models.Commit) (map[string][]*models.File, error) {
	fileChanges := make(map[string][]*models.File)
	mu := sync.Mutex{}

	g, ctx := errgroup.WithContext(ctx)

	for _, commit := range commits {
		commit := commit // Capture for goroutine
		g.Go(func() error {
			// In production, this would fetch actual file changes from GitHub API
			// For now, we'll store the reference
			mu.Lock()
			fileChanges[commit.SHA] = []*models.File{} // Initialize empty for now
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return fileChanges, nil
}
