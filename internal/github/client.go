package github

import (
	"context"
	"fmt"
	"time"

	"github.com/rohankatakam/coderisk/internal/types"
	"github.com/google/go-github/v57/github"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

// Client wraps the GitHub API client with rate limiting and concurrency
type Client struct {
	client      *github.Client
	rateLimiter *rate.Limiter
	maxWorkers  int
}

// NewClient creates a new GitHub client with rate limiting
func NewClient(token string, rateLimit int) *Client {
	client := github.NewClient(nil).WithAuthToken(token)

	return &Client{
		client:      client,
		rateLimiter: rate.NewLimiter(rate.Limit(rateLimit), 1),
		maxWorkers:  20, // Concurrent API calls
	}
}

// FetchRepository gets repository metadata
func (c *Client) FetchRepository(ctx context.Context, owner, name string) (*types.Repository, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	repo, _, err := c.client.Repositories.Get(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("fetch repository: %w", err)
	}

	return &types.Repository{
		ID:            fmt.Sprintf("%s/%s", owner, name),
		Owner:         owner,
		Name:          name,
		FullName:      repo.GetFullName(),
		URL:           repo.GetHTMLURL(),
		DefaultBranch: repo.GetDefaultBranch(),
		Language:      repo.GetLanguage(),
		Size:          int64(repo.GetSize()),
		StarCount:     repo.GetStargazersCount(),
		CreatedAt:     repo.GetCreatedAt().Time,
		UpdatedAt:     repo.GetUpdatedAt().Time,
	}, nil
}

// FetchCommits retrieves commits from the repository
func (c *Client) FetchCommits(ctx context.Context, owner, name string, since time.Time) ([]*types.Commit, error) {
	opts := &github.CommitsListOptions{
		Since: since,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var allCommits []*types.Commit

	for {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}

		commits, resp, err := c.client.Repositories.ListCommits(ctx, owner, name, opts)
		if err != nil {
			return nil, fmt.Errorf("fetch commits: %w", err)
		}

		for _, commit := range commits {
			modelCommit := &types.Commit{
				SHA:         commit.GetSHA(),
				RepoID:      fmt.Sprintf("%s/%s", owner, name),
				Author:      commit.GetAuthor().GetLogin(),
				AuthorEmail: commit.GetCommit().GetAuthor().GetEmail(),
				Message:     commit.GetCommit().GetMessage(),
				Timestamp:   commit.GetCommit().GetAuthor().GetDate().Time,
			}

			// Get parent SHAs
			for _, parent := range commit.Parents {
				modelCommit.ParentSHAs = append(modelCommit.ParentSHAs, parent.GetSHA())
			}

			allCommits = append(allCommits, modelCommit)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allCommits, nil
}

// FetchFiles retrieves all files from the repository
func (c *Client) FetchFiles(ctx context.Context, owner, name, ref string) ([]*types.File, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	tree, _, err := c.client.Git.GetTree(ctx, owner, name, ref, true)
	if err != nil {
		return nil, fmt.Errorf("fetch tree: %w", err)
	}

	var files []*types.File
	repoID := fmt.Sprintf("%s/%s", owner, name)

	// Process files in parallel
	g, ctx := errgroup.WithContext(ctx)
	fileChan := make(chan *types.File, len(tree.Entries))

	// Worker pool
	workerCount := min(c.maxWorkers, len(tree.Entries))
	for i := 0; i < workerCount; i++ {
		g.Go(func() error {
			for entry := range getEntryChannel(tree.Entries) {
				if entry.GetType() != "blob" {
					continue
				}

				file := &types.File{
					ID:     fmt.Sprintf("%s:%s", repoID, entry.GetPath()),
					RepoID: repoID,
					Path:   entry.GetPath(),
					SHA:    entry.GetSHA(),
					Size:   int64(entry.GetSize()),
				}

				// Determine language from extension
				file.Language = detectLanguage(entry.GetPath())

				fileChan <- file
			}
			return nil
		})
	}

	// Collect results
	go func() {
		g.Wait()
		close(fileChan)
	}()

	for file := range fileChan {
		files = append(files, file)
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return files, nil
}

// FetchPullRequests retrieves pull requests
func (c *Client) FetchPullRequests(ctx context.Context, owner, name string, state string) ([]*types.PullRequest, error) {
	opts := &github.PullRequestListOptions{
		State: state,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var allPRs []*types.PullRequest
	repoID := fmt.Sprintf("%s/%s", owner, name)

	for {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}

		prs, resp, err := c.client.PullRequests.List(ctx, owner, name, opts)
		if err != nil {
			return nil, fmt.Errorf("fetch pull requests: %w", err)
		}

		for _, pr := range prs {
			modelPR := &types.PullRequest{
				ID:          int(pr.GetID()),
				RepoID:      repoID,
				Number:      pr.GetNumber(),
				Title:       pr.GetTitle(),
				Description: pr.GetBody(),
				State:       pr.GetState(),
				Author:      pr.GetUser().GetLogin(),
				BaseBranch:  pr.GetBase().GetRef(),
				HeadBranch:  pr.GetHead().GetRef(),
				CreatedAt:   pr.GetCreatedAt().Time,
			}

			if pr.MergedAt != nil {
				t := pr.MergedAt.Time
				modelPR.MergedAt = &t
			}
			if pr.ClosedAt != nil {
				t := pr.ClosedAt.Time
				modelPR.ClosedAt = &t
			}

			allPRs = append(allPRs, modelPR)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allPRs, nil
}

// FetchIssues retrieves issues from the repository
func (c *Client) FetchIssues(ctx context.Context, owner, name string, labels []string) ([]*types.Issue, error) {
	opts := &github.IssueListByRepoOptions{
		Labels: labels,
		State:  "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var allIssues []*types.Issue
	repoID := fmt.Sprintf("%s/%s", owner, name)

	for {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}

		issues, resp, err := c.client.Issues.ListByRepo(ctx, owner, name, opts)
		if err != nil {
			return nil, fmt.Errorf("fetch issues: %w", err)
		}

		for _, issue := range issues {
			// Skip pull requests
			if issue.IsPullRequest() {
				continue
			}

			modelIssue := &types.Issue{
				ID:        int(issue.GetID()),
				RepoID:    repoID,
				Number:    issue.GetNumber(),
				Title:     issue.GetTitle(),
				Body:      issue.GetBody(),
				State:     issue.GetState(),
				Author:    issue.GetUser().GetLogin(),
				CreatedAt: issue.GetCreatedAt().Time,
			}

			// Extract labels
			for _, label := range issue.Labels {
				modelIssue.Labels = append(modelIssue.Labels, label.GetName())
			}

			// Check if it's an incident
			modelIssue.IsIncident = isIncident(modelIssue.Labels)

			if issue.ClosedAt != nil {
				t := issue.ClosedAt.Time
				modelIssue.ClosedAt = &t
			}

			allIssues = append(allIssues, modelIssue)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allIssues, nil
}

// Helper functions

func getEntryChannel(entries []*github.TreeEntry) <-chan *github.TreeEntry {
	ch := make(chan *github.TreeEntry)
	go func() {
		for _, entry := range entries {
			ch <- entry
		}
		close(ch)
	}()
	return ch
}

func detectLanguage(path string) string {
	// Simple language detection based on file extension
	extensions := map[string]string{
		".go":   "Go",
		".py":   "Python",
		".js":   "JavaScript",
		".ts":   "TypeScript",
		".java": "Java",
		".rb":   "Ruby",
		".rs":   "Rust",
		".cpp":  "C++",
		".c":    "C",
		".cs":   "C#",
	}

	for ext, lang := range extensions {
		if len(path) > len(ext) && path[len(path)-len(ext):] == ext {
			return lang
		}
	}
	return "Unknown"
}

func isIncident(labels []string) bool {
	incidentLabels := []string{"incident", "bug", "outage", "production-issue", "hotfix"}
	for _, label := range labels {
		for _, incidentLabel := range incidentLabels {
			if label == incidentLabel {
				return true
			}
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
