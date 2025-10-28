package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

// StagingClient provides access to GitHub API staging tables
// Reference: scripts/schema/postgresql_staging.sql
// Reference: dev_docs/03-implementation/integration_guides/layers_2_3_github_fetching.md
type StagingClient struct {
	db *sql.DB
}

// NewStagingClient creates a PostgreSQL client for GitHub staging tables
func NewStagingClient(ctx context.Context, host string, port int, database, user, password string) (*StagingClient, error) {
	connString := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		host, port, database, user, password,
	)

	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	// Verify connectivity
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &StagingClient{db: db}, nil
}

// Close closes the database connection
func (c *StagingClient) Close() error {
	return c.db.Close()
}

// DataCounts represents counts of existing data for a repository
type DataCounts struct {
	Commits      int
	Issues       int
	PRs          int
	Branches     int
	Contributors int
}

// GetDataCounts returns counts of existing data for a repository
func (c *StagingClient) GetDataCounts(ctx context.Context, repoID int64) (*DataCounts, error) {
	query := `
		SELECT
			(SELECT COUNT(*) FROM github_commits WHERE repo_id = $1) as commits,
			(SELECT COUNT(*) FROM github_issues WHERE repo_id = $1) as issues,
			(SELECT COUNT(*) FROM github_pull_requests WHERE repo_id = $1) as prs,
			(SELECT COUNT(*) FROM github_branches WHERE repo_id = $1) as branches,
			(SELECT COUNT(*) FROM github_contributors WHERE repo_id = $1) as contributors
	`

	counts := &DataCounts{}
	err := c.db.QueryRowContext(ctx, query, repoID).Scan(
		&counts.Commits,
		&counts.Issues,
		&counts.PRs,
		&counts.Branches,
		&counts.Contributors,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get data counts: %w", err)
	}

	return counts, nil
}

// ===================================
// Repository Operations
// ===================================

// StoreRepository stores repository metadata with raw JSON and absolute path
func (c *StagingClient) StoreRepository(ctx context.Context, githubID int64, owner, name, fullName, absolutePath string, rawData json.RawMessage) (int64, error) {
	query := `
		INSERT INTO github_repositories (github_id, owner, name, full_name, absolute_path, raw_data, fetched_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (full_name)
		DO UPDATE SET raw_data = EXCLUDED.raw_data, absolute_path = EXCLUDED.absolute_path, fetched_at = NOW(), updated_at = NOW()
		RETURNING id
	`

	var repoID int64
	err := c.db.QueryRowContext(ctx, query, githubID, owner, name, fullName, absolutePath, rawData).Scan(&repoID)
	if err != nil {
		return 0, fmt.Errorf("failed to store repository: %w", err)
	}

	return repoID, nil
}

// GetRepositoryID returns the internal ID for a repository
func (c *StagingClient) GetRepositoryID(ctx context.Context, fullName string) (int64, error) {
	var repoID int64
	query := "SELECT id FROM github_repositories WHERE full_name = $1"
	err := c.db.QueryRowContext(ctx, query, fullName).Scan(&repoID)
	if err != nil {
		return 0, fmt.Errorf("repository not found: %s: %w", fullName, err)
	}
	return repoID, nil
}

// ===================================
// Commit Operations
// ===================================

// StoreCommit stores commit data with raw JSON
func (c *StagingClient) StoreCommit(ctx context.Context, repoID int64, sha string, authorName, authorEmail string, authorDate time.Time, message string, additions, deletions, totalChanges, filesChanged int, rawData json.RawMessage) error {
	query := `
		INSERT INTO github_commits (
			repo_id, sha, author_name, author_email, author_date,
			message, additions, deletions, total_changes, files_changed,
			raw_data, fetched_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		ON CONFLICT (repo_id, sha)
		DO UPDATE SET raw_data = EXCLUDED.raw_data, fetched_at = NOW()
	`

	_, err := c.db.ExecContext(ctx, query,
		repoID, sha, authorName, authorEmail, authorDate,
		message, additions, deletions, totalChanges, filesChanged,
		rawData,
	)

	if err != nil {
		return fmt.Errorf("failed to store commit %s: %w", sha, err)
	}

	return nil
}

// ===================================
// Issue Operations
// ===================================

// StoreIssue stores issue data with raw JSON
func (c *StagingClient) StoreIssue(ctx context.Context, repoID int64, githubID int64, number int, title, body, state, userLogin string, userID int64, labels json.RawMessage, createdAt time.Time, closedAt *time.Time, rawData json.RawMessage) error {
	query := `
		INSERT INTO github_issues (
			repo_id, github_id, number, title, body, state,
			user_login, user_id, labels, created_at, closed_at,
			raw_data, fetched_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		ON CONFLICT (repo_id, number)
		DO UPDATE SET raw_data = EXCLUDED.raw_data, fetched_at = NOW()
	`

	_, err := c.db.ExecContext(ctx, query,
		repoID, githubID, number, title, body, state,
		userLogin, userID, labels, createdAt, closedAt,
		rawData,
	)

	if err != nil {
		return fmt.Errorf("failed to store issue #%d: %w", number, err)
	}

	return nil
}

// ===================================
// Pull Request Operations
// ===================================

// StorePullRequest stores PR data with raw JSON
func (c *StagingClient) StorePullRequest(ctx context.Context, repoID int64, githubID int64, number int, title, body, state, userLogin string, userID int64, headRef, headSHA, baseRef, baseSHA string, merged bool, mergedAt *time.Time, mergeCommitSHA *string, labels json.RawMessage, createdAt time.Time, closedAt *time.Time, rawData json.RawMessage) error {
	query := `
		INSERT INTO github_pull_requests (
			repo_id, github_id, number, title, body, state,
			user_login, user_id, head_ref, head_sha, base_ref, base_sha,
			merged, merged_at, merge_commit_sha, labels,
			created_at, closed_at, raw_data, fetched_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, NOW())
		ON CONFLICT (repo_id, number)
		DO UPDATE SET raw_data = EXCLUDED.raw_data, fetched_at = NOW()
	`

	_, err := c.db.ExecContext(ctx, query,
		repoID, githubID, number, title, body, state,
		userLogin, userID, headRef, headSHA, baseRef, baseSHA,
		merged, mergedAt, mergeCommitSHA, labels,
		createdAt, closedAt, rawData,
	)

	if err != nil {
		return fmt.Errorf("failed to store PR #%d: %w", number, err)
	}

	return nil
}

// ===================================
// Branch Operations
// ===================================

// StoreBranch stores branch data with raw JSON
func (c *StagingClient) StoreBranch(ctx context.Context, repoID int64, name, commitSHA string, protected bool, rawData json.RawMessage) error {
	query := `
		INSERT INTO github_branches (
			repo_id, name, commit_sha, protected, raw_data, fetched_at
		)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (repo_id, name)
		DO UPDATE SET commit_sha = EXCLUDED.commit_sha, raw_data = EXCLUDED.raw_data, fetched_at = NOW()
	`

	_, err := c.db.ExecContext(ctx, query, repoID, name, commitSHA, protected, rawData)
	if err != nil {
		return fmt.Errorf("failed to store branch %s: %w", name, err)
	}

	return nil
}

// ===================================
// Tree Operations
// ===================================

// StoreTree stores file tree data with raw JSON
func (c *StagingClient) StoreTree(ctx context.Context, repoID int64, sha, ref string, truncated bool, treeCount int, rawData json.RawMessage) error {
	query := `
		INSERT INTO github_trees (
			repo_id, sha, ref, truncated, tree_count, raw_data, fetched_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (repo_id, sha)
		DO UPDATE SET raw_data = EXCLUDED.raw_data, fetched_at = NOW()
	`

	_, err := c.db.ExecContext(ctx, query, repoID, sha, ref, truncated, treeCount, rawData)
	if err != nil {
		return fmt.Errorf("failed to store tree %s: %w", sha, err)
	}

	return nil
}

// ===================================
// Metadata Operations
// ===================================

// StoreLanguages stores repository language statistics
func (c *StagingClient) StoreLanguages(ctx context.Context, repoID int64, languages json.RawMessage) error {
	query := `
		INSERT INTO github_languages (repo_id, languages, fetched_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (repo_id)
		DO UPDATE SET languages = EXCLUDED.languages, fetched_at = NOW()
	`

	_, err := c.db.ExecContext(ctx, query, repoID, languages)
	if err != nil {
		return fmt.Errorf("failed to store languages: %w", err)
	}

	return nil
}

// StoreContributor stores contributor data
func (c *StagingClient) StoreContributor(ctx context.Context, repoID int64, githubID int64, login string, contributions int, rawData json.RawMessage) error {
	query := `
		INSERT INTO github_contributors (
			repo_id, github_id, login, contributions, raw_data, fetched_at
		)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (repo_id, github_id)
		DO UPDATE SET contributions = EXCLUDED.contributions, raw_data = EXCLUDED.raw_data, fetched_at = NOW()
	`

	_, err := c.db.ExecContext(ctx, query, repoID, githubID, login, contributions, rawData)
	if err != nil {
		return fmt.Errorf("failed to store contributor %s: %w", login, err)
	}

	return nil
}

// ===================================
// Query Operations for Graph Construction
// ===================================

// CommitData represents data fetched from v_unprocessed_commits view
type CommitData struct {
	ID          int64
	RepoID      int64
	SHA         string
	AuthorEmail string
	AuthorName  string
	AuthorDate  time.Time
	Message     string
	RawData     json.RawMessage
}

// FetchUnprocessedCommits retrieves commits ready for graph construction
func (c *StagingClient) FetchUnprocessedCommits(ctx context.Context, repoID int64, limit int) ([]CommitData, error) {
	query := `
		SELECT id, repo_id, sha, author_email, author_name, author_date, message, raw_data
		FROM v_unprocessed_commits
		WHERE repo_id = $1
		LIMIT $2
	`

	rows, err := c.db.QueryContext(ctx, query, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch unprocessed commits: %w", err)
	}
	defer rows.Close()

	var commits []CommitData
	for rows.Next() {
		var c CommitData
		if err := rows.Scan(&c.ID, &c.RepoID, &c.SHA, &c.AuthorEmail, &c.AuthorName, &c.AuthorDate, &c.Message, &c.RawData); err != nil {
			return nil, fmt.Errorf("failed to scan commit: %w", err)
		}
		commits = append(commits, c)
	}

	return commits, rows.Err()
}

// MarkCommitsProcessed updates processed_at timestamp for commits
func (c *StagingClient) MarkCommitsProcessed(ctx context.Context, commitIDs []int64) error {
	query := `
		UPDATE github_commits
		SET processed_at = NOW()
		WHERE id = ANY($1)
	`

	_, err := c.db.ExecContext(ctx, query, pq.Array(commitIDs))
	if err != nil {
		return fmt.Errorf("failed to mark commits as processed: %w", err)
	}

	return nil
}

// IssueData represents data from v_unprocessed_issues view
type IssueData struct {
	ID        int64
	RepoID    int64
	Number    int
	Title     string
	Body      string
	State     string
	Labels    json.RawMessage
	CreatedAt time.Time
	ClosedAt  *time.Time
	RawData   json.RawMessage
}

// FetchUnprocessedIssues retrieves issues ready for graph construction
func (c *StagingClient) FetchUnprocessedIssues(ctx context.Context, repoID int64, limit int) ([]IssueData, error) {
	query := `
		SELECT id, repo_id, number, title, body, state, labels, created_at, closed_at, raw_data
		FROM v_unprocessed_issues
		WHERE repo_id = $1
		LIMIT $2
	`

	rows, err := c.db.QueryContext(ctx, query, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch unprocessed issues: %w", err)
	}
	defer rows.Close()

	var issues []IssueData
	for rows.Next() {
		var i IssueData
		if err := rows.Scan(&i.ID, &i.RepoID, &i.Number, &i.Title, &i.Body, &i.State, &i.Labels, &i.CreatedAt, &i.ClosedAt, &i.RawData); err != nil {
			return nil, fmt.Errorf("failed to scan issue: %w", err)
		}
		issues = append(issues, i)
	}

	return issues, rows.Err()
}

// MarkIssuesProcessed updates processed_at timestamp for issues
func (c *StagingClient) MarkIssuesProcessed(ctx context.Context, issueIDs []int64) error {
	query := `
		UPDATE github_issues
		SET processed_at = NOW()
		WHERE id = ANY($1)
	`

	_, err := c.db.ExecContext(ctx, query, pq.Array(issueIDs))
	if err != nil {
		return fmt.Errorf("failed to mark issues as processed: %w", err)
	}

	return nil
}

// PRData represents data from v_unprocessed_prs view
type PRData struct {
	ID             int64
	RepoID         int64
	Number         int
	Title          string
	Body           string
	State          string
	Merged         bool
	MergeCommitSHA *string
	CreatedAt      time.Time
	MergedAt       *time.Time
	RawData        json.RawMessage
}

// FetchUnprocessedPRs retrieves PRs ready for graph construction
func (c *StagingClient) FetchUnprocessedPRs(ctx context.Context, repoID int64, limit int) ([]PRData, error) {
	query := `
		SELECT id, repo_id, number, title, body, state, merged, merge_commit_sha, created_at, merged_at, raw_data
		FROM v_unprocessed_prs
		WHERE repo_id = $1
		LIMIT $2
	`

	rows, err := c.db.QueryContext(ctx, query, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch unprocessed PRs: %w", err)
	}
	defer rows.Close()

	var prs []PRData
	for rows.Next() {
		var p PRData
		if err := rows.Scan(&p.ID, &p.RepoID, &p.Number, &p.Title, &p.Body, &p.State, &p.Merged, &p.MergeCommitSHA, &p.CreatedAt, &p.MergedAt, &p.RawData); err != nil {
			return nil, fmt.Errorf("failed to scan PR: %w", err)
		}
		prs = append(prs, p)
	}

	return prs, rows.Err()
}

// MarkPRsProcessed updates processed_at timestamp for PRs
func (c *StagingClient) MarkPRsProcessed(ctx context.Context, prIDs []int64) error {
	query := `
		UPDATE github_pull_requests
		SET processed_at = NOW()
		WHERE id = ANY($1)
	`

	_, err := c.db.ExecContext(ctx, query, pq.Array(prIDs))
	if err != nil {
		return fmt.Errorf("failed to mark PRs as processed: %w", err)
	}

	return nil
}

// BranchData represents data from github_branches table
type BranchData struct {
	ID        int64
	RepoID    int64
	Name      string
	CommitSHA string
	Protected bool
	RawData   json.RawMessage
}

// FetchUnprocessedBranches retrieves branches ready for graph construction
func (c *StagingClient) FetchUnprocessedBranches(ctx context.Context, repoID int64, limit int) ([]BranchData, error) {
	query := `
		SELECT id, repo_id, name, commit_sha, protected, raw_data
		FROM github_branches
		WHERE repo_id = $1 AND processed_at IS NULL
		LIMIT $2
	`

	rows, err := c.db.QueryContext(ctx, query, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch unprocessed branches: %w", err)
	}
	defer rows.Close()

	var branches []BranchData
	for rows.Next() {
		var b BranchData
		if err := rows.Scan(&b.ID, &b.RepoID, &b.Name, &b.CommitSHA, &b.Protected, &b.RawData); err != nil {
			return nil, fmt.Errorf("failed to scan branch: %w", err)
		}
		branches = append(branches, b)
	}

	return branches, rows.Err()
}

// MarkBranchesProcessed updates processed_at timestamp for branches
func (c *StagingClient) MarkBranchesProcessed(ctx context.Context, branchIDs []int64) error {
	query := `
		UPDATE github_branches
		SET processed_at = NOW()
		WHERE id = ANY($1)
	`

	_, err := c.db.ExecContext(ctx, query, pq.Array(branchIDs))
	if err != nil {
		return fmt.Errorf("failed to mark branches as processed: %w", err)
	}

	return nil
}

// GetDefaultBranchName retrieves the default branch name for a repository
func (c *StagingClient) GetDefaultBranchName(ctx context.Context, repoID int64) (string, error) {
	query := `
		SELECT raw_data->>'default_branch' as default_branch
		FROM github_repositories
		WHERE id = $1
	`

	var defaultBranch string
	err := c.db.QueryRowContext(ctx, query, repoID).Scan(&defaultBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get default branch: %w", err)
	}

	return defaultBranch, nil
}

// GetProcessedCommitSHAs retrieves all commit SHAs that have been processed into the graph
func (c *StagingClient) GetProcessedCommitSHAs(ctx context.Context, repoID int64) ([]string, error) {
	query := `
		SELECT sha
		FROM github_commits
		WHERE repo_id = $1 AND processed_at IS NOT NULL
	`

	rows, err := c.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query commits: %w", err)
	}
	defer rows.Close()

	var shas []string
	for rows.Next() {
		var sha string
		if err := rows.Scan(&sha); err != nil {
			return nil, fmt.Errorf("failed to scan commit SHA: %w", err)
		}
		shas = append(shas, sha)
	}

	return shas, rows.Err()
}

// PRBranchData represents PR branch information for graph linking
type PRBranchData struct {
	Number         int
	HeadRef        string
	BaseRef        string
	Merged         bool
	MergeCommitSHA *string
}

// GetProcessedPRBranchData retrieves PR branch information for all processed PRs
func (c *StagingClient) GetProcessedPRBranchData(ctx context.Context, repoID int64) ([]PRBranchData, error) {
	query := `
		SELECT number, head_ref, base_ref, merged, merge_commit_sha
		FROM github_pull_requests
		WHERE repo_id = $1 AND processed_at IS NOT NULL
	`

	rows, err := c.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query PRs: %w", err)
	}
	defer rows.Close()

	var prs []PRBranchData
	for rows.Next() {
		var pr PRBranchData
		if err := rows.Scan(&pr.Number, &pr.HeadRef, &pr.BaseRef, &pr.Merged, &pr.MergeCommitSHA); err != nil {
			return nil, fmt.Errorf("failed to scan PR: %w", err)
		}
		prs = append(prs, pr)
	}

	return prs, rows.Err()
}

// ===================================
// Issue Timeline Operations
// ===================================

// TimelineEventData represents a timeline event
type TimelineEventData struct {
	IssueID         int64
	EventType       string
	CreatedAt       time.Time
	SourceType      *string
	SourceNumber    *int
	SourceSHA       *string
	SourceTitle     *string
	SourceBody      *string
	SourceState     *string
	SourceMergedAt  *time.Time
	ActorLogin      *string
	ActorID         *int64
	RawData         json.RawMessage
}

// StoreTimelineEvent stores a timeline event
func (c *StagingClient) StoreTimelineEvent(ctx context.Context, event TimelineEventData) error {
	query := `
		INSERT INTO github_issue_timeline (
			issue_id, event_type, created_at, source_type, source_number, source_sha,
			source_title, source_body, source_state, source_merged_at,
			actor_login, actor_id, raw_data, fetched_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
		ON CONFLICT (issue_id, event_type, created_at, source_number) DO NOTHING
	`

	_, err := c.db.ExecContext(ctx, query,
		event.IssueID, event.EventType, event.CreatedAt,
		event.SourceType, event.SourceNumber, event.SourceSHA,
		event.SourceTitle, event.SourceBody, event.SourceState, event.SourceMergedAt,
		event.ActorLogin, event.ActorID, event.RawData,
	)

	if err != nil {
		return fmt.Errorf("failed to store timeline event: %w", err)
	}

	return nil
}

// FetchUnprocessedTimelineEvents retrieves timeline events ready for LLM extraction
func (c *StagingClient) FetchUnprocessedTimelineEvents(ctx context.Context, repoID int64, limit int) ([]TimelineEventData, error) {
	query := `
		SELECT t.id as issue_id, t.event_type, t.created_at, t.source_type, t.source_number,
			   t.source_sha, t.source_title, t.source_body, t.source_state, t.source_merged_at,
			   t.actor_login, t.actor_id, t.raw_data
		FROM github_issue_timeline t
		JOIN github_issues i ON t.issue_id = i.id
		WHERE i.repo_id = $1 AND t.processed_at IS NULL
		LIMIT $2
	`

	rows, err := c.db.QueryContext(ctx, query, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch unprocessed timeline events: %w", err)
	}
	defer rows.Close()

	var events []TimelineEventData
	for rows.Next() {
		var e TimelineEventData
		if err := rows.Scan(&e.IssueID, &e.EventType, &e.CreatedAt, &e.SourceType, &e.SourceNumber,
			&e.SourceSHA, &e.SourceTitle, &e.SourceBody, &e.SourceState, &e.SourceMergedAt,
			&e.ActorLogin, &e.ActorID, &e.RawData); err != nil {
			return nil, fmt.Errorf("failed to scan timeline event: %w", err)
		}
		events = append(events, e)
	}

	return events, rows.Err()
}

// MarkTimelineEventsProcessed updates processed_at timestamp for timeline events
func (c *StagingClient) MarkTimelineEventsProcessed(ctx context.Context, eventIDs []int64) error {
	query := `
		UPDATE github_issue_timeline
		SET processed_at = NOW()
		WHERE id = ANY($1)
	`

	_, err := c.db.ExecContext(ctx, query, pq.Array(eventIDs))
	if err != nil {
		return fmt.Errorf("failed to mark timeline events as processed: %w", err)
	}

	return nil
}

// ===================================
// Issue-Commit Reference Operations
// ===================================

// IssueCommitRef represents a reference between an issue and a commit/PR
type IssueCommitRef struct {
	RepoID          int64
	IssueNumber     int
	CommitSHA       *string
	PRNumber        *int
	Action          string
	Confidence      float64
	DetectionMethod string
	ExtractedFrom   string
}

// StoreIssueCommitRefs stores multiple issue-commit references in a batch
func (c *StagingClient) StoreIssueCommitRefs(ctx context.Context, refs []IssueCommitRef) error {
	if len(refs) == 0 {
		return nil
	}

	query := `
		INSERT INTO github_issue_commit_refs (
			repo_id, issue_number, commit_sha, pr_number,
			action, confidence, detection_method, extracted_from,
			extracted_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		ON CONFLICT (repo_id, issue_number, commit_sha, pr_number, detection_method)
		DO UPDATE SET
			confidence = GREATEST(github_issue_commit_refs.confidence, EXCLUDED.confidence),
			extracted_at = NOW()
	`

	// Use a transaction for batch insert
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, ref := range refs {
		_, err := stmt.ExecContext(ctx,
			ref.RepoID, ref.IssueNumber, ref.CommitSHA, ref.PRNumber,
			ref.Action, ref.Confidence, ref.DetectionMethod, ref.ExtractedFrom,
		)
		if err != nil {
			return fmt.Errorf("failed to insert reference: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetIssueCommitRefs retrieves all issue-commit references for a repository
func (c *StagingClient) GetIssueCommitRefs(ctx context.Context, repoID int64) ([]IssueCommitRef, error) {
	query := `
		SELECT repo_id, issue_number, commit_sha, pr_number,
			   action, confidence, detection_method, extracted_from
		FROM github_issue_commit_refs
		WHERE repo_id = $1
		ORDER BY confidence DESC, issue_number, commit_sha
	`

	rows, err := c.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query references: %w", err)
	}
	defer rows.Close()

	var refs []IssueCommitRef
	for rows.Next() {
		var ref IssueCommitRef
		if err := rows.Scan(&ref.RepoID, &ref.IssueNumber, &ref.CommitSHA, &ref.PRNumber,
			&ref.Action, &ref.Confidence, &ref.DetectionMethod, &ref.ExtractedFrom); err != nil {
			return nil, fmt.Errorf("failed to scan reference: %w", err)
		}
		refs = append(refs, ref)
	}

	return refs, rows.Err()
}

// ===================================
// Helper Methods for Entity Resolution
// ===================================

// QueryRow executes a query that returns a single row
func (c *StagingClient) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return c.db.QueryRowContext(ctx, query, args...)
}
