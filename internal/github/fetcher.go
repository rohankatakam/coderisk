package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/google/go-github/v57/github"
	"golang.org/x/time/rate"
)

// Fetcher handles GitHub API data fetching and storage in PostgreSQL staging tables
// Reference: dev_docs/03-implementation/integration_guides/layers_2_3_github_fetching.md
type Fetcher struct {
	client      *github.Client
	stagingDB   *database.StagingClient
	rateLimiter *rate.Limiter
}

// NewFetcher creates a GitHub API fetcher with PostgreSQL staging storage
func NewFetcher(githubToken string, stagingDB *database.StagingClient) *Fetcher {
	client := github.NewClient(nil).WithAuthToken(githubToken)

	// GitHub allows 5,000 requests/hour = 1.4 req/sec
	// Use conservative 1 req/sec to avoid rate limits
	limiter := rate.NewLimiter(rate.Every(1*time.Second), 1)

	return &Fetcher{
		client:      client,
		stagingDB:   stagingDB,
		rateLimiter: limiter,
	}
}

// FetchStats tracks fetching statistics
type FetchStats struct {
	Commits      int
	Issues       int
	PRs          int
	Branches     int
	Contributors int
}

// FetchAll fetches all GitHub data for a repository and stores in PostgreSQL
// This is Priority 6A: GitHub API → PostgreSQL staging
// Smart checkpointing: skips fetching if data already exists
func (f *Fetcher) FetchAll(ctx context.Context, owner, repo string) (int64, *FetchStats, error) {
	log.Printf("🔍 Fetching GitHub data for %s/%s...", owner, repo)
	stats := &FetchStats{}

	// 1. Fetch repository metadata (always needed to get repoID)
	repoID, err := f.FetchRepository(ctx, owner, repo)
	if err != nil {
		return 0, stats, fmt.Errorf("fetch repository failed: %w", err)
	}
	log.Printf("  ✓ Repository ID: %d", repoID)

	// Check if data already exists (smart checkpointing)
	existingStats, err := f.checkExistingData(ctx, repoID)
	if err != nil {
		log.Printf("  ⚠️  Could not check existing data: %v", err)
	} else if existingStats.hasData() {
		log.Printf("  ℹ️  Data already exists in PostgreSQL (skipping fetch):")
		log.Printf("     Commits: %d | Issues: %d | PRs: %d | Branches: %d",
			existingStats.Commits, existingStats.Issues, existingStats.PRs, existingStats.Branches)
		return repoID, existingStats, nil
	}

	// 2. Fetch commits (90-day window)
	commitCount, err := f.FetchCommits(ctx, repoID, owner, repo)
	if err != nil {
		return repoID, stats, fmt.Errorf("fetch commits failed: %w", err)
	}
	stats.Commits = commitCount
	log.Printf("  ✓ Fetched %d commits", commitCount)

	// 3. Fetch issues (90-day filtered)
	issueCount, err := f.FetchIssues(ctx, repoID, owner, repo)
	if err != nil {
		return repoID, stats, fmt.Errorf("fetch issues failed: %w", err)
	}
	stats.Issues = issueCount
	log.Printf("  ✓ Fetched %d issues", issueCount)

	// 4. Fetch pull requests (90-day filtered)
	prCount, err := f.FetchPullRequests(ctx, repoID, owner, repo)
	if err != nil {
		return repoID, stats, fmt.Errorf("fetch PRs failed: %w", err)
	}
	stats.PRs = prCount
	log.Printf("  ✓ Fetched %d pull requests", prCount)

	// 5. Fetch branches
	branchCount, err := f.FetchBranches(ctx, repoID, owner, repo)
	if err != nil {
		return repoID, stats, fmt.Errorf("fetch branches failed: %w", err)
	}
	stats.Branches = branchCount
	log.Printf("  ✓ Fetched %d branches", branchCount)

	// 6. Fetch languages
	if err := f.FetchLanguages(ctx, repoID, owner, repo); err != nil {
		log.Printf("  ⚠️  Failed to fetch languages: %v", err)
	}

	// 7. Fetch contributors
	contributorCount, err := f.FetchContributors(ctx, repoID, owner, repo)
	if err != nil {
		log.Printf("  ⚠️  Failed to fetch contributors: %v", err)
	} else {
		stats.Contributors = contributorCount
		log.Printf("  ✓ Fetched %d contributors", contributorCount)
	}

	return repoID, stats, nil
}

// FetchRepository fetches repository metadata and stores in PostgreSQL
func (f *Fetcher) FetchRepository(ctx context.Context, owner, repo string) (int64, error) {
	if err := f.rateLimiter.Wait(ctx); err != nil {
		return 0, err
	}

	repository, resp, err := f.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return 0, fmt.Errorf("GitHub API error: %w", err)
	}

	// Marshal to JSON for raw storage
	rawData, err := json.Marshal(repository)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal repository: %w", err)
	}

	// Store in PostgreSQL
	repoID, err := f.stagingDB.StoreRepository(
		ctx,
		repository.GetID(),
		owner,
		repo,
		repository.GetFullName(),
		rawData,
	)
	if err != nil {
		return 0, err
	}

	f.logRateLimit(resp)
	return repoID, nil
}

// FetchCommits fetches commits from the last 90 days
func (f *Fetcher) FetchCommits(ctx context.Context, repoID int64, owner, repo string) (int, error) {
	// 90-day window for temporal analysis
	since := time.Now().AddDate(0, 0, -90)

	opts := &github.CommitsListOptions{
		Since: since,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	count := 0
	for {
		if err := f.rateLimiter.Wait(ctx); err != nil {
			return count, err
		}

		commits, resp, err := f.client.Repositories.ListCommits(ctx, owner, repo, opts)
		if err != nil {
			return count, fmt.Errorf("list commits failed: %w", err)
		}

		for _, commit := range commits {
			// Fetch full commit details (includes files and stats)
			if err := f.fetchFullCommit(ctx, repoID, owner, repo, commit.GetSHA()); err != nil {
				log.Printf("  ⚠️  Failed to fetch commit %s: %v", commit.GetSHA(), err)
				continue
			}
			count++
		}

		f.logRateLimit(resp)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return count, nil
}

// fetchFullCommit fetches detailed commit info with files[] and stats
func (f *Fetcher) fetchFullCommit(ctx context.Context, repoID int64, owner, repo, sha string) error {
	if err := f.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	commit, _, err := f.client.Repositories.GetCommit(ctx, owner, repo, sha, nil)
	if err != nil {
		return fmt.Errorf("get commit failed: %w", err)
	}

	// Marshal to JSON for raw storage
	rawData, err := json.Marshal(commit)
	if err != nil {
		return fmt.Errorf("failed to marshal commit: %w", err)
	}

	// Extract fields for fast querying
	authorDate := commit.GetCommit().GetAuthor().GetDate().Time
	message := commit.GetCommit().GetMessage()
	authorName := commit.GetCommit().GetAuthor().GetName()
	authorEmail := commit.GetCommit().GetAuthor().GetEmail()

	stats := commit.GetStats()
	additions := stats.GetAdditions()
	deletions := stats.GetDeletions()
	total := stats.GetTotal()
	filesChanged := len(commit.Files)

	// Store in PostgreSQL
	return f.stagingDB.StoreCommit(
		ctx,
		repoID,
		sha,
		authorName,
		authorEmail,
		authorDate,
		message,
		additions,
		deletions,
		total,
		filesChanged,
		rawData,
	)
}

// FetchIssues fetches issues with 90-day filtering
func (f *Fetcher) FetchIssues(ctx context.Context, repoID int64, owner, repo string) (int, error) {
	opts := &github.IssueListByRepoOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	count := 0
	cutoff := time.Now().AddDate(0, 0, -90)

	for {
		if err := f.rateLimiter.Wait(ctx); err != nil {
			return count, err
		}

		issues, resp, err := f.client.Issues.ListByRepo(ctx, owner, repo, opts)
		if err != nil {
			return count, fmt.Errorf("list issues failed: %w", err)
		}

		for _, issue := range issues {
			// Skip pull requests (they're in a separate table)
			if issue.IsPullRequest() {
				continue
			}

			// Apply 90-day filter: open OR closed within 90 days
			if issue.GetState() == "open" ||
				(issue.ClosedAt != nil && issue.GetClosedAt().Time.After(cutoff)) {
				if err := f.storeIssue(ctx, repoID, issue); err != nil {
					log.Printf("  ⚠️  Failed to store issue #%d: %v", issue.GetNumber(), err)
					continue
				}
				count++
			}
		}

		f.logRateLimit(resp)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return count, nil
}

// storeIssue stores a single issue in PostgreSQL
func (f *Fetcher) storeIssue(ctx context.Context, repoID int64, issue *github.Issue) error {
	// Marshal to JSON
	rawData, err := json.Marshal(issue)
	if err != nil {
		return fmt.Errorf("failed to marshal issue: %w", err)
	}

	// Extract labels as JSON array
	labelNames := []string{}
	for _, label := range issue.Labels {
		labelNames = append(labelNames, label.GetName())
	}
	labelsJSON, _ := json.Marshal(labelNames)

	var closedAt *time.Time
	if issue.ClosedAt != nil {
		t := issue.GetClosedAt().Time
		closedAt = &t
	}

	return f.stagingDB.StoreIssue(
		ctx,
		repoID,
		issue.GetID(),
		issue.GetNumber(),
		issue.GetTitle(),
		issue.GetBody(),
		issue.GetState(),
		issue.GetUser().GetLogin(),
		issue.GetUser().GetID(),
		labelsJSON,
		issue.GetCreatedAt().Time,
		closedAt,
		rawData,
	)
}

// FetchPullRequests fetches PRs with 90-day filtering
func (f *Fetcher) FetchPullRequests(ctx context.Context, repoID int64, owner, repo string) (int, error) {
	opts := &github.PullRequestListOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	count := 0
	cutoff := time.Now().AddDate(0, 0, -90)

	for {
		if err := f.rateLimiter.Wait(ctx); err != nil {
			return count, err
		}

		prs, resp, err := f.client.PullRequests.List(ctx, owner, repo, opts)
		if err != nil {
			return count, fmt.Errorf("list PRs failed: %w", err)
		}

		for _, pr := range prs {
			// Apply 90-day filter: open OR merged/closed within 90 days
			if pr.GetState() == "open" ||
				(pr.MergedAt != nil && pr.GetMergedAt().Time.After(cutoff)) ||
				(pr.ClosedAt != nil && pr.GetClosedAt().Time.After(cutoff)) {
				if err := f.storePR(ctx, repoID, pr); err != nil {
					log.Printf("  ⚠️  Failed to store PR #%d: %v", pr.GetNumber(), err)
					continue
				}
				count++
			}
		}

		f.logRateLimit(resp)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return count, nil
}

// storePR stores a single pull request in PostgreSQL
func (f *Fetcher) storePR(ctx context.Context, repoID int64, pr *github.PullRequest) error {
	// Marshal to JSON
	rawData, err := json.Marshal(pr)
	if err != nil {
		return fmt.Errorf("failed to marshal PR: %w", err)
	}

	// Extract labels as JSON array
	labelNames := []string{}
	for _, label := range pr.Labels {
		labelNames = append(labelNames, label.GetName())
	}
	labelsJSON, _ := json.Marshal(labelNames)

	var mergedAt *time.Time
	if pr.MergedAt != nil {
		t := pr.GetMergedAt().Time
		mergedAt = &t
	}

	var closedAt *time.Time
	if pr.ClosedAt != nil {
		t := pr.GetClosedAt().Time
		closedAt = &t
	}

	var mergeCommitSHA *string
	if pr.MergeCommitSHA != nil {
		mergeCommitSHA = pr.MergeCommitSHA
	}

	return f.stagingDB.StorePullRequest(
		ctx,
		repoID,
		pr.GetID(),
		pr.GetNumber(),
		pr.GetTitle(),
		pr.GetBody(),
		pr.GetState(),
		pr.GetUser().GetLogin(),
		pr.GetUser().GetID(),
		pr.GetHead().GetRef(),
		pr.GetHead().GetSHA(),
		pr.GetBase().GetRef(),
		pr.GetBase().GetSHA(),
		pr.GetMerged(),
		mergedAt,
		mergeCommitSHA,
		labelsJSON,
		pr.GetCreatedAt().Time,
		closedAt,
		rawData,
	)
}

// FetchBranches fetches all branches
func (f *Fetcher) FetchBranches(ctx context.Context, repoID int64, owner, repo string) (int, error) {
	opts := &github.BranchListOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	count := 0
	for {
		if err := f.rateLimiter.Wait(ctx); err != nil {
			return count, err
		}

		branches, resp, err := f.client.Repositories.ListBranches(ctx, owner, repo, opts)
		if err != nil {
			return count, fmt.Errorf("list branches failed: %w", err)
		}

		for _, branch := range branches {
			rawData, err := json.Marshal(branch)
			if err != nil {
				continue
			}

			if err := f.stagingDB.StoreBranch(
				ctx,
				repoID,
				branch.GetName(),
				branch.GetCommit().GetSHA(),
				branch.GetProtected(),
				rawData,
			); err != nil {
				log.Printf("  ⚠️  Failed to store branch %s: %v", branch.GetName(), err)
				continue
			}
			count++
		}

		f.logRateLimit(resp)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return count, nil
}

// FetchLanguages fetches repository language statistics
func (f *Fetcher) FetchLanguages(ctx context.Context, repoID int64, owner, repo string) error {
	if err := f.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	languages, _, err := f.client.Repositories.ListLanguages(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("list languages failed: %w", err)
	}

	languagesJSON, err := json.Marshal(languages)
	if err != nil {
		return fmt.Errorf("failed to marshal languages: %w", err)
	}

	return f.stagingDB.StoreLanguages(ctx, repoID, languagesJSON)
}

// FetchContributors fetches repository contributors
func (f *Fetcher) FetchContributors(ctx context.Context, repoID int64, owner, repo string) (int, error) {
	opts := &github.ListContributorsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	count := 0
	for {
		if err := f.rateLimiter.Wait(ctx); err != nil {
			return count, err
		}

		contributors, resp, err := f.client.Repositories.ListContributors(ctx, owner, repo, opts)
		if err != nil {
			return count, fmt.Errorf("list contributors failed: %w", err)
		}

		for _, contributor := range contributors {
			rawData, err := json.Marshal(contributor)
			if err != nil {
				continue
			}

			if err := f.stagingDB.StoreContributor(
				ctx,
				repoID,
				contributor.GetID(),
				contributor.GetLogin(),
				contributor.GetContributions(),
				rawData,
			); err != nil {
				log.Printf("  ⚠️  Failed to store contributor %s: %v", contributor.GetLogin(), err)
				continue
			}
			count++
		}

		f.logRateLimit(resp)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return count, nil
}

// logRateLimit logs GitHub API rate limit info
func (f *Fetcher) logRateLimit(resp *github.Response) {
	if resp == nil {
		return
	}

	remaining := resp.Rate.Remaining
	limit := resp.Rate.Limit

	// Warn if getting low
	if remaining < 100 {
		log.Printf("  ⚠️  Rate limit low: %d/%d remaining", remaining, limit)
	}
}

// checkExistingData checks if GitHub data already exists in PostgreSQL
func (f *Fetcher) checkExistingData(ctx context.Context, repoID int64) (*FetchStats, error) {
	counts, err := f.stagingDB.GetDataCounts(ctx, repoID)
	if err != nil {
		return nil, err
	}

	stats := &FetchStats{
		Commits:      counts.Commits,
		Issues:       counts.Issues,
		PRs:          counts.PRs,
		Branches:     counts.Branches,
		Contributors: counts.Contributors,
	}

	return stats, nil
}

// hasData returns true if any data exists
func (s *FetchStats) hasData() bool {
	return s.Commits > 0 || s.Issues > 0 || s.PRs > 0 || s.Branches > 0
}
