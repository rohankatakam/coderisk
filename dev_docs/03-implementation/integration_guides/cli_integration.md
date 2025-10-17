# CLI Integration: `crisk init` Command (Priority 6C)

**Purpose:** Wire Priorities 6A and 6B into end-to-end `crisk init` CLI command

**Last Updated:** October 3, 2025

**Prerequisites:**
- Priority 6A complete (GitHub API → PostgreSQL)
- Priority 6B complete (PostgreSQL → Neo4j/Neptune)
- Go 1.23+
- Cobra CLI framework

**Target:** Priority 6C - End-to-end initialization command

---

## Architecture Context

**References:**
- [layers_2_3_github_fetching.md](layers_2_3_github_fetching.md) - Priority 6A (Stage 1)
- [layers_2_3_graph_construction.md](layers_2_3_graph_construction.md) - Priority 6B (Stage 2)
- [layer_1_treesitter.md](layer_1_treesitter.md) - Priority 7 (future)
- [local_deployment.md](local_deployment.md) - Docker Compose setup

**Command Purpose:**
Provide a single command to initialize CodeRisk for a repository, combining:
1. GitHub API fetching (Priority 6A)
2. Graph construction (Priority 6B)
3. Validation and reporting

**Performance Targets:**
- omnara-ai/omnara: ~23s total
- kubernetes/kubernetes: ~2min total

---

## Overview

The `crisk init` command orchestrates the complete data pipeline:

```
crisk init omnara-ai/omnara
   ↓
Stage 1: GitHub API → PostgreSQL (22s omnara, 2min kubernetes)
   ↓ (fetch commits, issues, PRs, branches, trees)
Stage 2: PostgreSQL → Neo4j/Neptune (40ms omnara, 1.75s kubernetes)
   ↓ (parse JSONB, generate queries, batch insert)
Validation & Report
   ↓ (verify node/edge counts, print summary)
✅ Ready for `crisk check` commands
```

---

## CLI Command Structure

### Main Command

```go
// cmd/crisk/init.go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/spf13/cobra"
    "coderisk/internal/github"
    "coderisk/internal/graph"
    "coderisk/internal/db"
)

func initCommand() *cobra.Command {
    var (
        githubToken string
        backend     string // "neo4j" or "neptune"
    )

    cmd := &cobra.Command{
        Use:   "init [repository]",
        Short: "Initialize CodeRisk for a repository",
        Long: `Initialize CodeRisk by fetching GitHub data and building the knowledge graph.

Examples:
  crisk init omnara-ai/omnara
  crisk init omnara-ai/omnara --backend neo4j
  crisk init omnara-ai/omnara --token ghp_xxxxx

The repository can be specified as:
  - owner/repo (fetches from GitHub)
  - /path/to/local/repo (future: local analysis)
`,
        Args: cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return runInit(args[0], githubToken, backend)
        },
    }

    cmd.Flags().StringVarP(&githubToken, "token", "t", "", "GitHub personal access token (or set GITHUB_TOKEN)")
    cmd.Flags().StringVarP(&backend, "backend", "b", "neo4j", "Graph backend: neo4j (local) or neptune (cloud)")

    return cmd
}
```

### Main Initialization Logic

```go
func runInit(repoName, githubToken, backendType string) error {
    startTime := time.Now()

    // Parse repository name
    owner, name, err := parseRepoName(repoName)
    if err != nil {
        return fmt.Errorf("invalid repository name: %w", err)
    }

    log.Printf("🚀 Initializing CodeRisk for %s/%s...\n", owner, name)
    log.Printf("   Backend: %s\n", backendType)

    // Get GitHub token
    if githubToken == "" {
        githubToken = os.Getenv("GITHUB_TOKEN")
    }
    if githubToken == "" {
        return fmt.Errorf("GitHub token required: use --token or set GITHUB_TOKEN")
    }

    // Connect to PostgreSQL
    log.Printf("\n[0/3] Connecting to databases...")
    dbConn, err := db.Connect()
    if err != nil {
        return fmt.Errorf("PostgreSQL connection failed: %w", err)
    }
    defer dbConn.Close()

    // Connect to graph backend
    var graphBackend graph.Backend
    switch backendType {
    case "neo4j":
        graphBackend, err = graph.NewNeo4jBackend(
            os.Getenv("NEO4J_URI"),
            os.Getenv("NEO4J_USER"),
            os.Getenv("NEO4J_PASSWORD"),
        )
    case "neptune":
        graphBackend, err = graph.NewNeptuneBackend(
            os.Getenv("NEPTUNE_ENDPOINT"),
        )
    default:
        return fmt.Errorf("unsupported backend: %s (use 'neo4j' or 'neptune')", backendType)
    }
    if err != nil {
        return fmt.Errorf("graph backend connection failed: %w", err)
    }
    defer graphBackend.Close()

    log.Printf("  ✓ Connected to PostgreSQL")
    log.Printf("  ✓ Connected to %s", backendType)

    ctx := context.Background()

    // Stage 1: Fetch GitHub data → PostgreSQL
    log.Printf("\n[1/3] Fetching GitHub API data...")
    fetchStart := time.Now()

    repoID, stats, err := fetchGitHubData(ctx, dbConn, githubToken, owner, name)
    if err != nil {
        return fmt.Errorf("fetch failed: %w", err)
    }

    fetchDuration := time.Since(fetchStart)
    log.Printf("  ✓ Fetched in %v", fetchDuration)
    log.Printf("    Commits: %d | Issues: %d | PRs: %d | Branches: %d",
        stats.Commits, stats.Issues, stats.PRs, stats.Branches)

    // Stage 2: Build graph from PostgreSQL → Neo4j/Neptune
    log.Printf("\n[2/3] Building knowledge graph...")
    graphStart := time.Now()

    graphStats, err := buildGraph(ctx, dbConn, graphBackend, repoID)
    if err != nil {
        return fmt.Errorf("graph construction failed: %w", err)
    }

    graphDuration := time.Since(graphStart)
    log.Printf("  ✓ Graph built in %v", graphDuration)
    log.Printf("    Nodes: %d | Edges: %d", graphStats.Nodes, graphStats.Edges)

    // Stage 3: Validate
    log.Printf("\n[3/3] Validating...")
    if err := validateGraph(ctx, graphBackend, graphStats); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    log.Printf("  ✓ Validation passed")

    // Summary
    totalDuration := time.Since(startTime)
    log.Printf("\n✅ CodeRisk initialized for %s/%s", owner, name)
    log.Printf("   Total time: %v (fetch: %v, graph: %v)",
        totalDuration, fetchDuration, graphDuration)
    log.Printf("\n💡 Try: crisk check <file>")

    return nil
}
```

### Helper Functions

```go
// parseRepoName splits "owner/repo" into components
func parseRepoName(repoName string) (owner, name string, err error) {
    parts := strings.Split(repoName, "/")
    if len(parts) != 2 {
        return "", "", fmt.Errorf("expected format: owner/repo")
    }
    return parts[0], parts[1], nil
}

type FetchStats struct {
    Commits  int
    Issues   int
    PRs      int
    Branches int
}

// fetchGitHubData orchestrates Priority 6A
func fetchGitHubData(ctx context.Context, db *sql.DB, token, owner, name string) (int64, FetchStats, error) {
    client := github.NewClient(token)
    stats := FetchStats{}

    // 1. Fetch repository metadata
    repoID, err := github.FetchRepository(ctx, client, owner, name)
    if err != nil {
        return 0, stats, err
    }

    // 2. Fetch commits (90-day window)
    commitCount, err := github.FetchCommits(ctx, client, repoID, owner, name)
    if err != nil {
        return 0, stats, err
    }
    stats.Commits = commitCount

    // 3. Fetch issues (filtered)
    issueCount, err := github.FetchIssues(ctx, client, repoID, owner, name)
    if err != nil {
        return 0, stats, err
    }
    stats.Issues = issueCount

    // 4. Fetch PRs (filtered)
    prCount, err := github.FetchPullRequests(ctx, client, repoID, owner, name)
    if err != nil {
        return 0, stats, err
    }
    stats.PRs = prCount

    // 5. Fetch branches
    branchCount, err := github.FetchBranches(ctx, client, repoID, owner, name)
    if err != nil {
        return 0, stats, err
    }
    stats.Branches = branchCount

    return repoID, stats, nil
}

type GraphStats struct {
    Nodes int
    Edges int
}

// buildGraph orchestrates Priority 6B
func buildGraph(ctx context.Context, db *sql.DB, backend graph.Backend, repoID int64) (GraphStats, error) {
    if err := graph.BuildGraph(db, backend, repoID); err != nil {
        return GraphStats{}, err
    }

    // Query graph to get counts
    stats := GraphStats{}

    // Count nodes
    nodeResult, err := backend.Query(getNodeCountQuery(backend))
    if err != nil {
        return stats, err
    }
    stats.Nodes = parseCount(nodeResult)

    // Count edges
    edgeResult, err := backend.Query(getEdgeCountQuery(backend))
    if err != nil {
        return stats, err
    }
    stats.Edges = parseCount(edgeResult)

    return stats, nil
}

func getNodeCountQuery(backend graph.Backend) string {
    switch backend.(type) {
    case *graph.Neo4jBackend:
        return "MATCH (n) RETURN count(n) as count"
    case *graph.NeptuneBackend:
        return "g.V().count()"
    default:
        return ""
    }
}

func getEdgeCountQuery(backend graph.Backend) string {
    switch backend.(type) {
    case *graph.Neo4jBackend:
        return "MATCH ()-[r]->() RETURN count(r) as count"
    case *graph.NeptuneBackend:
        return "g.E().count()"
    default:
        return ""
    }
}

// validateGraph performs basic sanity checks
func validateGraph(ctx context.Context, backend graph.Backend, stats GraphStats) error {
    if stats.Nodes == 0 {
        return fmt.Errorf("no nodes created in graph")
    }
    if stats.Edges == 0 {
        return fmt.Errorf("no edges created in graph")
    }

    // Verify required node types exist
    requiredLabels := []string{"Commit", "Developer", "Issue", "PullRequest"}

    for _, label := range requiredLabels {
        query := getLabelCountQuery(backend, label)
        result, err := backend.Query(query)
        if err != nil {
            return fmt.Errorf("failed to query %s nodes: %w", label, err)
        }

        count := parseCount(result)
        if count == 0 {
            log.Printf("  ⚠️  Warning: No %s nodes found", label)
        }
    }

    return nil
}

func getLabelCountQuery(backend graph.Backend, label string) string {
    switch backend.(type) {
    case *graph.Neo4jBackend:
        return fmt.Sprintf("MATCH (n:%s) RETURN count(n) as count", label)
    case *graph.NeptuneBackend:
        return fmt.Sprintf("g.V().hasLabel('%s').count()", label)
    default:
        return ""
    }
}
```

---

## Error Handling

### Graceful Degradation

```go
// If fetch fails partway through, graph construction can still proceed with partial data
func runInitWithRecovery(repoName, githubToken, backendType string) error {
    // ... setup code ...

    // Stage 1: Fetch (with error recovery)
    log.Printf("\n[1/3] Fetching GitHub API data...")
    repoID, stats, fetchErr := fetchGitHubData(ctx, dbConn, githubToken, owner, name)

    if fetchErr != nil {
        log.Printf("  ⚠️  Fetch incomplete: %v", fetchErr)
        log.Printf("  Proceeding with partial data...")
    }

    // Check if we have minimum data to proceed
    if stats.Commits == 0 {
        return fmt.Errorf("no commits fetched, cannot proceed")
    }

    // Stage 2: Build graph (always proceed if we have any data)
    log.Printf("\n[2/3] Building knowledge graph...")
    graphStats, err := buildGraph(ctx, dbConn, graphBackend, repoID)
    if err != nil {
        return fmt.Errorf("graph construction failed: %w", err)
    }

    // ... rest of init ...
}
```

### Retry on Failure

```go
// Allow re-running init without re-fetching (idempotent)
func runInit(repoName, githubToken, backendType string) error {
    // Check if already initialized
    repoID, exists := checkExistingRepo(dbConn, owner, name)

    if exists {
        log.Printf("ℹ️  Repository already initialized (ID: %d)", repoID)
        log.Printf("   Checking for new data...")

        // Only fetch if data is stale (> 1 day old)
        if isStale(dbConn, repoID) {
            log.Printf("   Data is stale, re-fetching...")
            // Proceed with fetch
        } else {
            log.Printf("   Data is fresh, skipping fetch")
            log.Printf("   Re-building graph...")
            // Skip to graph construction
        }
    }

    // ... rest of init ...
}

func isStale(db *sql.DB, repoID int64) bool {
    var fetchedAt time.Time
    query := "SELECT fetched_at FROM github_repositories WHERE id = $1"
    db.QueryRow(query, repoID).Scan(&fetchedAt)

    return time.Since(fetchedAt) > 24*time.Hour
}
```

---

## Environment Configuration

### Required Environment Variables

```bash
# .env file for local development

# GitHub API
GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=coderisk

# Neo4j (local)
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=password

# Neptune (cloud) - optional
NEPTUNE_ENDPOINT=wss://your-neptune-cluster.region.neptune.amazonaws.com:8182/gremlin
```

### Loading Environment

```go
// internal/config/config.go
package config

import (
    "os"
    "github.com/joho/godotenv"
)

type Config struct {
    GitHub   GitHubConfig
    Postgres PostgresConfig
    Neo4j    Neo4jConfig
    Neptune  NeptuneConfig
}

type GitHubConfig struct {
    Token string
}

type PostgresConfig struct {
    Host     string
    Port     string
    User     string
    Password string
    DB       string
}

type Neo4jConfig struct {
    URI      string
    User     string
    Password string
}

type NeptuneConfig struct {
    Endpoint string
}

func Load() (*Config, error) {
    // Load .env if exists
    godotenv.Load()

    return &Config{
        GitHub: GitHubConfig{
            Token: os.Getenv("GITHUB_TOKEN"),
        },
        Postgres: PostgresConfig{
            Host:     getEnvOrDefault("POSTGRES_HOST", "localhost"),
            Port:     getEnvOrDefault("POSTGRES_PORT", "5432"),
            User:     getEnvOrDefault("POSTGRES_USER", "postgres"),
            Password: getEnvOrDefault("POSTGRES_PASSWORD", "postgres"),
            DB:       getEnvOrDefault("POSTGRES_DB", "coderisk"),
        },
        Neo4j: Neo4jConfig{
            URI:      getEnvOrDefault("NEO4J_URI", "bolt://localhost:7687"),
            User:     getEnvOrDefault("NEO4J_USER", "neo4j"),
            Password: getEnvOrDefault("NEO4J_PASSWORD", "password"),
        },
        Neptune: NeptuneConfig{
            Endpoint: os.Getenv("NEPTUNE_ENDPOINT"),
        },
    }, nil
}

func getEnvOrDefault(key, defaultVal string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return defaultVal
}
```

---

## Usage Examples

### Basic Usage

```bash
# Initialize omnara-ai/omnara with default Neo4j backend
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
crisk init omnara-ai/omnara
```

**Output:**
```
🚀 Initializing CodeRisk for omnara-ai/omnara...
   Backend: neo4j

[0/3] Connecting to databases...
  ✓ Connected to PostgreSQL
  ✓ Connected to neo4j

[1/3] Fetching GitHub API data...
  ✓ Fetched in 22.3s
    Commits: 100 | Issues: 75 | PRs: 72 | Branches: 100

[2/3] Building knowledge graph...
  ✓ Graph built in 41ms
    Nodes: 250 | Edges: 402

[3/3] Validating...
  ✓ Validation passed

✅ CodeRisk initialized for omnara-ai/omnara
   Total time: 22.4s (fetch: 22.3s, graph: 41ms)

💡 Try: crisk check <file>
```

### With Custom Backend

```bash
# Use Neptune (cloud) instead of Neo4j
export NEPTUNE_ENDPOINT=wss://your-cluster.neptune.amazonaws.com:8182/gremlin
crisk init omnara-ai/omnara --backend neptune
```

### Re-run (Idempotent)

```bash
# Re-running is safe - will skip if data is fresh
crisk init omnara-ai/omnara

# Output:
# ℹ️  Repository already initialized (ID: 1)
#    Checking for new data...
#    Data is fresh, skipping fetch
#    Re-building graph...
#  ✓ Graph built in 38ms
```

---

## Testing Strategy

### Integration Tests

```go
func TestInitCommandE2E(t *testing.T) {
    // Requires Docker Compose running (PostgreSQL + Neo4j)
    // Requires GITHUB_TOKEN set

    token := os.Getenv("GITHUB_TOKEN")
    if token == "" {
        t.Skip("GITHUB_TOKEN not set")
    }

    // Run init command
    err := runInit("omnara-ai/omnara", token, "neo4j")
    require.NoError(t, err)

    // Verify PostgreSQL has data
    db := connectDB(t)
    defer db.Close()

    var commitCount int
    db.QueryRow("SELECT COUNT(*) FROM github_commits").Scan(&commitCount)
    assert.Greater(t, commitCount, 0)

    // Verify Neo4j has data
    neo4j := connectNeo4j(t)
    defer neo4j.Close()

    result, err := neo4j.Query("MATCH (c:Commit) RETURN count(c) as count")
    require.NoError(t, err)
    assert.Greater(t, result, 0)
}
```

---

## Performance Summary

**From [data_volumes.md](../../01-architecture/data_volumes.md):**

| Repository | Stage 1 (Fetch) | Stage 2 (Graph) | Total | Status |
|------------|-----------------|-----------------|-------|--------|
| omnara | 22s | 40ms | ~23s | ✅ <30s |
| kubernetes | 2min | 1.75s | ~2min | ✅ <10min |

---

## Next Steps

1. ✅ **Design complete** - Full CLI workflow documented
2. ⏭️ **Implement config** - `internal/config/config.go`
3. ⏭️ **Implement init command** - `cmd/crisk/init.go`
4. ⏭️ **Test with omnara** - End-to-end validation
5. ⏭️ **Add progress bar** - Enhance UX (optional)
6. ⏭️ **Add `--force` flag** - Force re-fetch even if fresh

---

## References

- **Priority 6A:** [layers_2_3_github_fetching.md](layers_2_3_github_fetching.md)
- **Priority 6B:** [layers_2_3_graph_construction.md](layers_2_3_graph_construction.md)
- **Local Deployment:** [local_deployment.md](local_deployment.md)
- **Cobra CLI:** https://github.com/spf13/cobra
