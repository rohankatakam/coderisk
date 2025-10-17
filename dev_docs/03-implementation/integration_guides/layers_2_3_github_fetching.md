# Layers 2 & 3: GitHub API Fetching (Priority 6A)

**Purpose:** Implementation guide for fetching GitHub API data and storing in PostgreSQL staging database

**Last Updated:** October 3, 2025

**Prerequisites:**
- PostgreSQL 15+ running (Docker Compose or standalone)
- GitHub personal access token (for authenticated requests)
- Go 1.23+

**Target:** Priority 6A - Stage 1 of data pipeline (GitHub API → PostgreSQL)

---

## Architecture Context

**References:**
- [data_volumes.md](../../01-architecture/data_volumes.md) - Accurate volume calculations from real API data
- [graph_ontology.md](../../01-architecture/graph_ontology.md) - Layers 2 & 3 schema
- [local_deployment.md](local_deployment.md) - PostgreSQL configuration
- [PostgreSQL Schema](../../../scripts/schema/postgresql_staging.sql) - Staging database schema
- [Test Data Samples](../../../test_data/github_api/README.md) - Real API response examples

**Layer 2 & 3 Purpose:**
- **Layer 2 (Temporal):** Git history, developers, change patterns
- **Layer 3 (Incidents):** Issues, PRs, bug tracking
- **Benefits:** Idempotent, resumable, auditable, testable

**Performance Targets:**
- omnara-ai/omnara: 22s (110 API requests)
- kubernetes/kubernetes: 2min (628 API requests)

---

## Overview

GitHub API fetching follows a staged approach with checkpointing:

```
GitHub API (REST)
   ↓ HTTP GET with pagination
PostgreSQL JSONB tables
   ↓ Raw JSON storage with metadata
Ready for graph construction (Priority 6B)
```

**Benefits of Staging Layer:**
- ✅ **Idempotent:** Re-runs don't duplicate data
- ✅ **Resumable:** Failures tracked via `fetched_at` timestamp
- ✅ **Auditable:** Know what was fetched and when
- ✅ **Testable:** Can test graph construction offline without API calls

**Total Time:** 22s (omnara), 2min (kubernetes)
**Storage:** 3.6 MB (omnara), 108 MB (kubernetes)

---

## PostgreSQL Schema Deployment

### Step 1: Deploy Schema

**Schema Location:** [scripts/schema/postgresql_staging.sql](../../../scripts/schema/postgresql_staging.sql)

```bash
# Deploy schema
psql -h localhost -U postgres -d coderisk -f scripts/schema/postgresql_staging.sql
```

**Key Tables:**
- `github_repositories` - Repository metadata (1 per repo)
- `github_commits` - Commit history with raw JSONB
- `github_developers` - Developer profiles (extracted from commits)
- `github_issues` - Issue tracking
- `github_pull_requests` - PR metadata with merge info
- `github_branches` - Branch metadata
- `github_trees` - File structure snapshots
- `github_languages` - Language statistics
- `github_contributors` - Contributor activity

**Key Features:**
- JSONB columns for raw API responses
- `fetched_at` + `processed_at` timestamps for checkpointing
- Partial indexes on `WHERE processed_at IS NULL`
- GIN indexes on JSONB for fast querying
- Views for graph construction workers

### Step 2: Verify Deployment

```sql
-- Check tables exist
\dt github_*

-- Check views exist
\dv v_unprocessed_*

-- Test insert
INSERT INTO github_repositories (github_id, owner, name, full_name, raw_data)
VALUES (123, 'test', 'repo', 'test/repo', '{}');

SELECT * FROM github_repositories WHERE full_name = 'test/repo';
```

---

## GitHub API Client Implementation

### HTTP Client with Rate Limiting

**Implementation: `internal/github/client.go`**

```go
package github

import (
    "context"
    "fmt"
    "net/http"
    "time"
    "golang.org/x/time/rate"
)

type Client struct {
    httpClient *http.Client
    token      string
    baseURL    string
    limiter    *rate.Limiter
}

// NewClient creates GitHub API client with rate limiting
func NewClient(token string) *Client {
    // GitHub allows 5,000 requests/hour authenticated
    // = 83.3 req/min = 1.4 req/sec
    // Use conservative limit: 1 req/sec
    limiter := rate.NewLimiter(rate.Every(1*time.Second), 1)

    return &Client{
        httpClient: &http.Client{Timeout: 30 * time.Second},
        token:      token,
        baseURL:    "https://api.github.com",
        limiter:    limiter,
    }
}

// Get performs GET request with rate limiting and auth
func (c *Client) Get(ctx context.Context, path string) (*http.Response, error) {
    // Wait for rate limit
    if err := c.limiter.Wait(ctx); err != nil {
        return nil, err
    }

    url := c.baseURL + path
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    // Add authentication
    req.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
    req.Header.Set("Accept", "application/vnd.github+json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }

    // Check rate limit headers
    if resp.Header.Get("X-RateLimit-Remaining") == "0" {
        resetTime := resp.Header.Get("X-RateLimit-Reset")
        return nil, fmt.Errorf("rate limit exceeded, resets at %s", resetTime)
    }

    return resp, nil
}
```

### Pagination Handler

```go
// GetPaginated fetches all pages for a paginated endpoint
func (c *Client) GetPaginated(ctx context.Context, path string, perPage int) ([][]byte, error) {
    var allPages [][]byte
    page := 1

    for {
        // Build URL with pagination params
        url := fmt.Sprintf("%s?per_page=%d&page=%d", path, perPage, page)

        resp, err := c.Get(ctx, url)
        if err != nil {
            return nil, err
        }
        defer resp.Body.Close()

        // Read response body
        body, err := io.ReadAll(resp.Body)
        if err != nil {
            return nil, err
        }

        // Check if empty (no more pages)
        if string(body) == "[]" {
            break
        }

        allPages = append(allPages, body)

        // Check for Link header (GitHub pagination)
        linkHeader := resp.Header.Get("Link")
        if !strings.Contains(linkHeader, `rel="next"`) {
            break // No more pages
        }

        page++
    }

    return allPages, nil
}
```

---

## Fetching Endpoints

### 1. Repository Metadata

**Endpoint:** `GET /repos/{owner}/{repo}`

```go
// FetchRepository fetches and stores repository metadata
func FetchRepository(ctx context.Context, client *Client, owner, name string) (int64, error) {
    path := fmt.Sprintf("/repos/%s/%s", owner, name)
    resp, err := client.Get(ctx, path)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return 0, err
    }

    // Parse to extract GitHub ID
    var repo struct {
        ID       int64  `json:"id"`
        FullName string `json:"full_name"`
    }
    if err := json.Unmarshal(body, &repo); err != nil {
        return 0, err
    }

    // Store in PostgreSQL
    query := `
        INSERT INTO github_repositories (github_id, owner, name, full_name, raw_data, fetched_at)
        VALUES ($1, $2, $3, $4, $5, NOW())
        ON CONFLICT (full_name)
        DO UPDATE SET raw_data = EXCLUDED.raw_data, fetched_at = NOW()
        RETURNING id
    `

    var repoID int64
    err = db.QueryRow(query, repo.ID, owner, name, repo.FullName, body).Scan(&repoID)
    if err != nil {
        return 0, err
    }

    return repoID, nil
}
```

### 2. Commits (90-Day Window)

**Endpoint:** `GET /repos/{owner}/{repo}/commits?since={date}&per_page=100`

```go
// FetchCommits fetches commits from last 90 days
func FetchCommits(ctx context.Context, client *Client, repoID int64, owner, name string) (int, error) {
    since := time.Now().AddDate(0, 0, -90).Format(time.RFC3339)
    path := fmt.Sprintf("/repos/%s/%s/commits?since=%s", owner, name, since)

    pages, err := client.GetPaginated(ctx, path, 100)
    if err != nil {
        return 0, err
    }

    count := 0
    for _, page := range pages {
        // Parse commits array
        var commits []struct {
            SHA    string          `json:"sha"`
            Commit json.RawMessage `json:"commit"`
        }

        if err := json.Unmarshal(page, &commits); err != nil {
            return count, err
        }

        // Insert each commit
        for _, commit := range commits {
            // Fetch full commit details (includes files[] and patch)
            fullCommit, err := fetchFullCommit(ctx, client, owner, name, commit.SHA)
            if err != nil {
                log.Printf("Warning: failed to fetch commit %s: %v", commit.SHA, err)
                continue
            }

            // Store in PostgreSQL
            if err := storeCommit(repoID, fullCommit); err != nil {
                return count, err
            }

            count++
        }
    }

    return count, nil
}

// fetchFullCommit gets detailed commit info with files[] and diff
func fetchFullCommit(ctx context.Context, client *Client, owner, name, sha string) ([]byte, error) {
    path := fmt.Sprintf("/repos/%s/%s/commits/%s", owner, name, sha)
    resp, err := client.Get(ctx, path)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    return io.ReadAll(resp.Body)
}

// storeCommit inserts commit into PostgreSQL
func storeCommit(repoID int64, rawData []byte) error {
    // Parse to extract fields for fast querying
    var commit struct {
        SHA    string `json:"sha"`
        Commit struct {
            Author struct {
                Name  string    `json:"name"`
                Email string    `json:"email"`
                Date  time.Time `json:"date"`
            } `json:"author"`
            Message string `json:"message"`
        } `json:"commit"`
        Stats struct {
            Additions int `json:"additions"`
            Deletions int `json:"deletions"`
            Total     int `json:"total"`
        } `json:"stats"`
        Files []json.RawMessage `json:"files"`
    }

    if err := json.Unmarshal(rawData, &commit); err != nil {
        return err
    }

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

    _, err := db.Exec(query,
        repoID, commit.SHA,
        commit.Commit.Author.Name, commit.Commit.Author.Email, commit.Commit.Author.Date,
        commit.Commit.Message,
        commit.Stats.Additions, commit.Stats.Deletions, commit.Stats.Total,
        len(commit.Files),
        rawData,
    )

    return err
}
```

### 3. Issues (Filtered)

**Endpoint:** `GET /repos/{owner}/{repo}/issues?state=all&per_page=100`

```go
// FetchIssues fetches all issues with 90-day filtering
func FetchIssues(ctx context.Context, client *Client, repoID int64, owner, name string) (int, error) {
    path := fmt.Sprintf("/repos/%s/%s/issues?state=all", owner, name)

    pages, err := client.GetPaginated(ctx, path, 100)
    if err != nil {
        return 0, err
    }

    count := 0
    cutoff := time.Now().AddDate(0, 0, -90)

    for _, page := range pages {
        var issues []json.RawMessage
        if err := json.Unmarshal(page, &issues); err != nil {
            return count, err
        }

        for _, issueData := range issues {
            // Parse to check if within 90-day window
            var issue struct {
                Number    int       `json:"number"`
                State     string    `json:"state"`
                CreatedAt time.Time `json:"created_at"`
                ClosedAt  *time.Time `json:"closed_at"`
            }

            if err := json.Unmarshal(issueData, &issue); err != nil {
                continue
            }

            // Apply 90-day filter
            if issue.State == "open" || (issue.ClosedAt != nil && issue.ClosedAt.After(cutoff)) {
                if err := storeIssue(repoID, issueData); err != nil {
                    return count, err
                }
                count++
            }
        }
    }

    return count, nil
}

// storeIssue inserts issue into PostgreSQL
func storeIssue(repoID int64, rawData []byte) error {
    var issue struct {
        ID        int64           `json:"id"`
        Number    int             `json:"number"`
        Title     string          `json:"title"`
        Body      string          `json:"body"`
        State     string          `json:"state"`
        Labels    json.RawMessage `json:"labels"`
        CreatedAt time.Time       `json:"created_at"`
        ClosedAt  *time.Time      `json:"closed_at"`
    }

    if err := json.Unmarshal(rawData, &issue); err != nil {
        return err
    }

    query := `
        INSERT INTO github_issues (
            repo_id, github_id, number, title, body, state,
            labels, created_at, closed_at, raw_data, fetched_at
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
        ON CONFLICT (repo_id, number)
        DO UPDATE SET raw_data = EXCLUDED.raw_data, fetched_at = NOW()
    `

    _, err := db.Exec(query,
        repoID, issue.ID, issue.Number, issue.Title, issue.Body, issue.State,
        issue.Labels, issue.CreatedAt, issue.ClosedAt, rawData,
    )

    return err
}
```

### 4. Pull Requests (Filtered)

**Endpoint:** `GET /repos/{owner}/{repo}/pulls?state=all&per_page=100`

```go
// FetchPullRequests fetches PRs with 90-day filtering
func FetchPullRequests(ctx context.Context, client *Client, repoID int64, owner, name string) (int, error) {
    path := fmt.Sprintf("/repos/%s/%s/pulls?state=all", owner, name)

    pages, err := client.GetPaginated(ctx, path, 100)
    if err != nil {
        return 0, err
    }

    count := 0
    cutoff := time.Now().AddDate(0, 0, -90)

    for _, page := range pages {
        var prs []json.RawMessage
        if err := json.Unmarshal(page, &prs); err != nil {
            return count, err
        }

        for _, prData := range prs {
            var pr struct {
                Number   int        `json:"number"`
                State    string     `json:"state"`
                Merged   bool       `json:"merged"`
                MergedAt *time.Time `json:"merged_at"`
            }

            if err := json.Unmarshal(prData, &pr); err != nil {
                continue
            }

            // Apply 90-day filter
            if pr.State == "open" || (pr.MergedAt != nil && pr.MergedAt.After(cutoff)) {
                if err := storePullRequest(repoID, prData); err != nil {
                    return count, err
                }
                count++
            }
        }
    }

    return count, nil
}
```

---

## Error Handling & Retry Logic

### Exponential Backoff

```go
// FetchWithRetry retries failed requests with exponential backoff
func FetchWithRetry(ctx context.Context, client *Client, path string, maxRetries int) (*http.Response, error) {
    var lastErr error

    for attempt := 0; attempt < maxRetries; attempt++ {
        resp, err := client.Get(ctx, path)
        if err == nil && resp.StatusCode < 500 {
            return resp, nil
        }

        lastErr = err
        if resp != nil && resp.StatusCode == 404 {
            return nil, fmt.Errorf("not found: %s", path) // Don't retry 404
        }

        // Exponential backoff: 1s, 2s, 4s, 8s
        backoff := time.Duration(1<<uint(attempt)) * time.Second
        log.Printf("Retry %d/%d after %v for %s", attempt+1, maxRetries, backoff, path)

        select {
        case <-time.After(backoff):
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }

    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

---

## CLI Command: `crisk fetch`

**Implementation: `cmd/crisk/fetch.go`**

```go
func fetchCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "fetch [repository]",
        Short: "Fetch GitHub API data for a repository",
        Args:  cobra.ExactArgs(1),
        RunE:  runFetch,
    }

    cmd.Flags().StringP("token", "t", "", "GitHub personal access token (or set GITHUB_TOKEN env)")
    return cmd
}

func runFetch(cmd *cobra.Command, args []string) error {
    repoName := args[0]
    owner, name := parseRepoName(repoName)

    // Get GitHub token
    token := cmd.Flag("token").Value.String()
    if token == "" {
        token = os.Getenv("GITHUB_TOKEN")
    }
    if token == "" {
        return fmt.Errorf("GitHub token required (--token or GITHUB_TOKEN)")
    }

    client := github.NewClient(token)
    ctx := context.Background()

    log.Printf("Fetching %s...", repoName)

    // 1. Fetch repository metadata
    log.Printf("[1/5] Fetching repository metadata...")
    repoID, err := github.FetchRepository(ctx, client, owner, name)
    if err != nil {
        return err
    }
    log.Printf("  ✓ Repository ID: %d", repoID)

    // 2. Fetch commits
    log.Printf("[2/5] Fetching commits (90-day window)...")
    commitCount, err := github.FetchCommits(ctx, client, repoID, owner, name)
    if err != nil {
        return err
    }
    log.Printf("  ✓ Fetched %d commits", commitCount)

    // 3. Fetch issues
    log.Printf("[3/5] Fetching issues...")
    issueCount, err := github.FetchIssues(ctx, client, repoID, owner, name)
    if err != nil {
        return err
    }
    log.Printf("  ✓ Fetched %d issues", issueCount)

    // 4. Fetch PRs
    log.Printf("[4/5] Fetching pull requests...")
    prCount, err := github.FetchPullRequests(ctx, client, repoID, owner, name)
    if err != nil {
        return err
    }
    log.Printf("  ✓ Fetched %d pull requests", prCount)

    // 5. Fetch metadata
    log.Printf("[5/5] Fetching metadata (branches, languages, contributors)...")
    // ... (similar pattern)

    log.Printf("✅ Fetch complete for %s", repoName)
    log.Printf("   Commits: %d | Issues: %d | PRs: %d", commitCount, issueCount, prCount)

    return nil
}
```

---

## Testing Strategy

### Unit Tests

```go
func TestFetchRepository(t *testing.T) {
    // Use test data from test_data/github_api/
    repoJSON, _ := os.ReadFile("../../../test_data/github_api/omnara-ai-omnara/repository.json")

    // Mock HTTP client
    mockClient := &MockClient{
        responses: map[string][]byte{
            "/repos/omnara-ai/omnara": repoJSON,
        },
    }

    repoID, err := FetchRepository(context.Background(), mockClient, "omnara-ai", "omnara")
    assert.NoError(t, err)
    assert.Greater(t, repoID, int64(0))
}
```

### Integration Tests

```go
func TestFetchOmnara(t *testing.T) {
    // Requires PostgreSQL running
    // Requires GITHUB_TOKEN set
    token := os.Getenv("GITHUB_TOKEN")
    if token == "" {
        t.Skip("GITHUB_TOKEN not set")
    }

    client := NewClient(token)
    ctx := context.Background()

    // Fetch omnara-ai/omnara
    repoID, err := FetchRepository(ctx, client, "omnara-ai", "omnara")
    require.NoError(t, err)

    // Verify stored in PostgreSQL
    var count int
    db.QueryRow("SELECT COUNT(*) FROM github_commits WHERE repo_id = $1", repoID).Scan(&count)
    assert.Greater(t, count, 0)
}
```

---

## Performance Summary

**From [data_volumes.md](../../01-architecture/data_volumes.md):**

| Repository | API Requests | Fetch Time | Storage |
|------------|--------------|------------|---------|
| omnara | 110 | 22s | 3.6 MB |
| kubernetes | 628 | 2min | 108 MB |

**Both well within GitHub API rate limits (5,000 req/hour) ✅**

---

## Next Steps

1. ✅ **Schema deployed** - PostgreSQL tables ready
2. ✅ **Design complete** - All endpoints documented
3. ⏭️ **Implement client** - `internal/github/client.go`
4. ⏭️ **Implement fetchers** - `internal/github/fetch.go`
5. ⏭️ **Wire CLI** - `cmd/crisk/fetch.go`
6. ⏭️ **Test with omnara** - Validate end-to-end
7. ⏭️ **Proceed to Priority 6B** - Graph construction

---

## References

- **PostgreSQL Schema:** [scripts/schema/postgresql_staging.sql](../../../scripts/schema/postgresql_staging.sql)
- **Data Volumes:** [data_volumes.md](../../01-architecture/data_volumes.md)
- **Test Data:** [test_data/github_api/](../../../test_data/github_api/)
- **GitHub API Docs:** https://docs.github.com/en/rest
