# Layers 2 & 3: Graph Construction (Priority 6B)

**Purpose:** Implementation guide for transforming PostgreSQL-staged GitHub data into Neo4j (local) or Neptune (cloud) graph

**Last Updated:** October 3, 2025

**Prerequisites:**
- PostgreSQL with GitHub data fetched (Priority 6A complete)
- Neo4j running (local) OR Neptune endpoint (cloud)
- Go 1.23+

**Target:** Priority 6B - Stage 2 of data pipeline (PostgreSQL → Graph Database)

---

## Architecture Context

**References:**
- [graph_ontology.md](../../01-architecture/graph_ontology.md) - Layers 2 & 3 schema
- [data_volumes.md](../../01-architecture/data_volumes.md) - Node/edge counts
- [local_deployment.md](local_deployment.md) - Neo4j configuration
- [PostgreSQL Schema](../../../scripts/schema/postgresql_staging.sql) - Source data structure
- [Test Data](../../../test_data/github_api/) - Sample API responses

**Graph Backends:**
- **Neo4j (Local):** Cypher query language, Docker Compose, free
- **Neptune (Cloud):** Gremlin query language, AWS managed, pay-per-use

**Layer 2 & 3 Entities:**
- **Layer 2 (Temporal):** Commit, Developer, PullRequest nodes + AUTHORED, MODIFIES, MERGED_TO edges
- **Layer 3 (Incidents):** Issue nodes + FIXES edges

**Performance Targets:**
- omnara-ai/omnara: 40ms (250 nodes, 400 edges)
- kubernetes/kubernetes: 1.75s (12,500 nodes, 23,000 edges)

---

## Overview

Graph construction follows a 3-phase transformation pipeline:

```
Phase 1: PostgreSQL → Go Structs
   ↓ (SELECT unprocessed records, parse JSONB)
Phase 2: Go Structs → Graph Entities
   ↓ (Map to ontology: Commit → nodes + edges)
Phase 3: Graph Entities → Neo4j/Neptune
   ↓ (Generate Cypher/Gremlin, batch execute, mark processed)
```

**Key Principles:**
- ✅ **Idempotent:** Re-runs don't create duplicates (MERGE/coalesce pattern)
- ✅ **Resumable:** Uses `processed_at` timestamp for checkpointing
- ✅ **Dual-backend:** Single codebase supports Neo4j + Neptune

**Total Time:** 40ms (omnara), 1.75s (kubernetes)

---

## Graph Backend Abstraction

### Interface Design

```go
package graph

type Backend interface {
    // Node operations
    CreateNode(node GraphNode) (string, error)
    CreateNodes(nodes []GraphNode) ([]string, error)

    // Edge operations
    CreateEdge(edge GraphEdge) error
    CreateEdges(edges []GraphEdge) error

    // Batch operations
    ExecuteBatch(commands []string) error

    // Query operations
    Query(query string) (interface{}, error)

    // Lifecycle
    Close() error
}

type GraphNode struct {
    Label      string                 // "Commit", "Developer", "Issue", etc.
    ID         string                 // Unique identifier
    Properties map[string]interface{} // Node properties
}

type GraphEdge struct {
    Label      string                 // "AUTHORED", "MODIFIES", "FIXES", etc.
    From       string                 // Source node ID
    To         string                 // Target node ID
    Properties map[string]interface{} // Edge properties
}
```

### Neo4j Implementation

```go
package graph

import (
    "fmt"
    "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Neo4jBackend struct {
    driver neo4j.DriverWithContext
}

func NewNeo4jBackend(uri, username, password string) (*Neo4jBackend, error) {
    driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
    if err != nil {
        return nil, err
    }

    return &Neo4jBackend{driver: driver}, nil
}

func (n *Neo4jBackend) CreateNode(node GraphNode) (string, error) {
    cypher := generateCypherNode(node)
    return n.executeCypher(cypher)
}

func (n *Neo4jBackend) CreateEdge(edge GraphEdge) error {
    cypher := generateCypherEdge(edge)
    _, err := n.executeCypher(cypher)
    return err
}

func (n *Neo4jBackend) ExecuteBatch(commands []string) error {
    ctx := context.Background()
    session := n.driver.NewSession(ctx, neo4j.SessionConfig{})
    defer session.Close(ctx)

    // Execute all commands in a transaction
    _, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
        for _, cypher := range commands {
            if _, err := tx.Run(ctx, cypher, nil); err != nil {
                return nil, err
            }
        }
        return nil, nil
    })

    return err
}

func (n *Neo4jBackend) executeCypher(cypher string) (string, error) {
    ctx := context.Background()
    session := n.driver.NewSession(ctx, neo4j.SessionConfig{})
    defer session.Close(ctx)

    result, err := session.Run(ctx, cypher, nil)
    if err != nil {
        return "", err
    }

    if result.Next(ctx) {
        record := result.Record()
        if id, ok := record.Get("id"); ok {
            return fmt.Sprintf("%v", id), nil
        }
    }

    return "", nil
}
```

### Neptune (Gremlin) Implementation

```go
package graph

import (
    gremlingo "github.com/apache/tinkerpop/gremlin-go/v3/driver"
)

type NeptuneBackend struct {
    client *gremlingo.DriverRemoteConnection
}

func NewNeptuneBackend(endpoint string) (*NeptuneBackend, error) {
    client, err := gremlingo.NewDriverRemoteConnection(endpoint)
    if err != nil {
        return nil, err
    }

    return &NeptuneBackend{client: client}, nil
}

func (n *NeptuneBackend) CreateNode(node GraphNode) (string, error) {
    gremlin := generateGremlinNode(node)
    return n.executeGremlin(gremlin)
}

func (n *NeptuneBackend) CreateEdge(edge GraphEdge) error {
    gremlin := generateGremlinEdge(edge)
    _, err := n.executeGremlin(gremlin)
    return err
}

func (n *NeptuneBackend) ExecuteBatch(commands []string) error {
    // Gremlin allows chaining with semicolons
    script := strings.Join(commands, "; ")
    _, err := n.executeGremlin(script)
    return err
}

func (n *NeptuneBackend) executeGremlin(script string) (string, error) {
    g := gremlingo.Traversal_().WithRemote(n.client)
    result, err := g.Inject(script).ToList()
    if err != nil {
        return "", err
    }

    if len(result) > 0 {
        return fmt.Sprintf("%v", result[0]), nil
    }

    return "", nil
}
```

---

## Phase 1: Fetch Unprocessed Data from PostgreSQL

### Commit Data

```go
type CommitData struct {
    ID          int64           `db:"id"`
    RepoID      int64           `db:"repo_id"`
    SHA         string          `db:"sha"`
    AuthorEmail string          `db:"author_email"`
    AuthorName  string          `db:"author_name"`
    AuthorDate  time.Time       `db:"author_date"`
    Message     string          `db:"message"`
    RawData     json.RawMessage `db:"raw_data"`
}

// FetchUnprocessedCommits retrieves commits ready for graph construction
func FetchUnprocessedCommits(db *sql.DB, repoID int64, limit int) ([]CommitData, error) {
    query := `
        SELECT id, repo_id, sha, author_email, author_name, author_date, message, raw_data
        FROM v_unprocessed_commits
        WHERE repo_id = $1
        LIMIT $2
    `

    rows, err := db.Query(query, repoID, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var commits []CommitData
    for rows.Next() {
        var c CommitData
        if err := rows.Scan(&c.ID, &c.RepoID, &c.SHA, &c.AuthorEmail, &c.AuthorName, &c.AuthorDate, &c.Message, &c.RawData); err != nil {
            return nil, err
        }
        commits = append(commits, c)
    }

    return commits, nil
}
```

### Issue and PR Data

```go
type IssueData struct {
    ID        int64           `db:"id"`
    RepoID    int64           `db:"repo_id"`
    Number    int             `db:"number"`
    Title     string          `db:"title"`
    Body      string          `db:"body"`
    State     string          `db:"state"`
    Labels    json.RawMessage `db:"labels"`
    CreatedAt time.Time       `db:"created_at"`
    ClosedAt  *time.Time      `db:"closed_at"`
    RawData   json.RawMessage `db:"raw_data"`
}

type PRData struct {
    ID             int64           `db:"id"`
    RepoID         int64           `db:"repo_id"`
    Number         int             `db:"number"`
    Title          string          `db:"title"`
    Body           string          `db:"body"`
    State          string          `db:"state"`
    Merged         bool            `db:"merged"`
    MergeCommitSHA *string         `db:"merge_commit_sha"`
    MergedAt       *time.Time      `db:"merged_at"`
    RawData        json.RawMessage `db:"raw_data"`
}

// Similar fetch functions for issues and PRs...
```

---

## Phase 2: Map to Graph Entities

### Commit → Graph Nodes + Edges

```go
// ProcessCommit transforms a commit into graph nodes and edges
func ProcessCommit(commit CommitData) ([]GraphNode, []GraphEdge, error) {
    nodes := []GraphNode{}
    edges := []GraphEdge{}

    // 1. Create Commit node
    commitNode := GraphNode{
        Label: "Commit",
        ID:    fmt.Sprintf("commit:%s", commit.SHA),
        Properties: map[string]interface{}{
            "sha":          commit.SHA,
            "message":      commit.Message,
            "author_email": commit.AuthorEmail,
            "author_name":  commit.AuthorName,
            "author_date":  commit.AuthorDate.Unix(),
        },
    }
    nodes = append(nodes, commitNode)

    // 2. Create Developer node
    developerNode := GraphNode{
        Label: "Developer",
        ID:    fmt.Sprintf("developer:%s", commit.AuthorEmail),
        Properties: map[string]interface{}{
            "email": commit.AuthorEmail,
            "name":  commit.AuthorName,
        },
    }
    nodes = append(nodes, developerNode)

    // 3. Create AUTHORED edge
    authoredEdge := GraphEdge{
        Label: "AUTHORED",
        From:  developerNode.ID,
        To:    commitNode.ID,
        Properties: map[string]interface{}{
            "timestamp": commit.AuthorDate.Unix(),
        },
    }
    edges = append(edges, authoredEdge)

    // 4. Parse raw_data to get files[] for MODIFIES edges
    var fullCommit struct {
        Files []struct {
            Filename  string `json:"filename"`
            Status    string `json:"status"`
            Additions int    `json:"additions"`
            Deletions int    `json:"deletions"`
        } `json:"files"`
        Stats struct {
            Additions int `json:"additions"`
            Deletions int `json:"deletions"`
            Total     int `json:"total"`
        } `json:"stats"`
    }

    if err := json.Unmarshal(commit.RawData, &fullCommit); err != nil {
        return nil, nil, err
    }

    // Update commit node with stats
    commitNode.Properties["additions"] = fullCommit.Stats.Additions
    commitNode.Properties["deletions"] = fullCommit.Stats.Deletions

    // 5. Create MODIFIES edges for each file
    for _, file := range fullCommit.Files {
        modifiesEdge := GraphEdge{
            Label: "MODIFIES",
            From:  commitNode.ID,
            To:    fmt.Sprintf("file:%s", file.Filename),
            Properties: map[string]interface{}{
                "status":    file.Status,
                "additions": file.Additions,
                "deletions": file.Deletions,
                "timestamp": commit.AuthorDate.Unix(),
            },
        }
        edges = append(edges, modifiesEdge)
    }

    return nodes, edges, nil
}
```

### Issue → Graph Node

```go
// ProcessIssue transforms an issue into a graph node
func ProcessIssue(issue IssueData) (GraphNode, error) {
    // Parse labels
    var labels []string
    if err := json.Unmarshal(issue.Labels, &labels); err != nil {
        return GraphNode{}, err
    }

    node := GraphNode{
        Label: "Issue",
        ID:    fmt.Sprintf("issue:%d", issue.Number),
        Properties: map[string]interface{}{
            "number":     issue.Number,
            "title":      issue.Title,
            "body":       issue.Body,
            "state":      issue.State,
            "labels":     labels,
            "created_at": issue.CreatedAt.Unix(),
        },
    }

    if issue.ClosedAt != nil {
        node.Properties["closed_at"] = issue.ClosedAt.Unix()
    }

    return node, nil
}
```

### PR → Graph Node + MERGED_TO + FIXES Edges

```go
// ProcessPR transforms a PR into graph node and edges
func ProcessPR(pr PRData) (GraphNode, []GraphEdge, error) {
    edges := []GraphEdge{}

    // Create PR node
    node := GraphNode{
        Label: "PullRequest",
        ID:    fmt.Sprintf("pr:%d", pr.Number),
        Properties: map[string]interface{}{
            "number":     pr.Number,
            "title":      pr.Title,
            "body":       pr.Body,
            "state":      pr.State,
            "merged":     pr.Merged,
            "created_at": pr.CreatedAt.Unix(),
        },
    }

    if pr.MergedAt != nil {
        node.Properties["merged_at"] = pr.MergedAt.Unix()
    }

    // Create MERGED_TO edge if merged
    if pr.Merged && pr.MergeCommitSHA != nil {
        mergedToEdge := GraphEdge{
            Label: "MERGED_TO",
            From:  node.ID,
            To:    fmt.Sprintf("commit:%s", *pr.MergeCommitSHA),
            Properties: map[string]interface{}{
                "merged_at": pr.MergedAt.Unix(),
            },
        }
        edges = append(edges, mergedToEdge)
    }

    // Extract issue references from title/body for FIXES edges
    issueNumbers := extractIssueReferences(pr.Title, pr.Body)
    for _, issueNum := range issueNumbers {
        fixesEdge := GraphEdge{
            Label: "FIXES",
            From:  node.ID,
            To:    fmt.Sprintf("issue:%d", issueNum),
            Properties: map[string]interface{}{
                "detected_from": "pr_body",
                "timestamp":     pr.CreatedAt.Unix(),
            },
        }
        edges = append(edges, fixesEdge)
    }

    return node, edges, nil
}

// extractIssueReferences parses text for "Fixes #123", "Closes #456" patterns
func extractIssueReferences(title, body string) []int {
    text := strings.ToLower(title + " " + body)
    re := regexp.MustCompile(`(?:fix|fixes|fixed|close|closes|closed|resolve|resolves|resolved)\s+#(\d+)`)
    matches := re.FindAllStringSubmatch(text, -1)

    issueNumbers := []int{}
    seen := make(map[int]bool)

    for _, match := range matches {
        if len(match) > 1 {
            if num, err := strconv.Atoi(match[1]); err == nil {
                if !seen[num] {
                    issueNumbers = append(issueNumbers, num)
                    seen[num] = true
                }
            }
        }
    }

    return issueNumbers
}
```

---

## Phase 3: Generate Backend-Specific Queries

### Neo4j Cypher Generation

```go
// generateCypherNode creates Cypher MERGE query for idempotent node creation
func generateCypherNode(node GraphNode) string {
    // Build property string
    props := []string{}
    for key, value := range node.Properties {
        props = append(props, fmt.Sprintf("%s: %s", key, formatCypherValue(value)))
    }

    // Use first property as unique key (e.g., sha for Commit, email for Developer)
    var uniqueKey string
    switch node.Label {
    case "Commit":
        uniqueKey = "sha"
    case "Developer":
        uniqueKey = "email"
    case "Issue":
        uniqueKey = "number"
    case "PullRequest":
        uniqueKey = "number"
    default:
        uniqueKey = "id"
    }

    return fmt.Sprintf(`
        MERGE (n:%s {%s: %s})
        SET %s
        RETURN id(n) as id
    `, node.Label, uniqueKey, formatCypherValue(node.Properties[uniqueKey]),
        strings.Join(propsToSet(node.Properties, "n"), ", "))
}

// generateCypherEdge creates Cypher MERGE query for idempotent edge creation
func generateCypherEdge(edge GraphEdge) string {
    // Extract label and ID from node references
    fromLabel, fromID := parseNodeID(edge.From)
    toLabel, toID := parseNodeID(edge.To)

    // Build property string
    props := []string{}
    for key, value := range edge.Properties {
        props = append(props, fmt.Sprintf("r.%s = %s", key, formatCypherValue(value)))
    }

    propsStr := ""
    if len(props) > 0 {
        propsStr = "SET " + strings.Join(props, ", ")
    }

    return fmt.Sprintf(`
        MATCH (from:%s {%s: %s})
        MATCH (to:%s {%s: %s})
        MERGE (from)-[r:%s]->(to)
        %s
    `, fromLabel, getUniqueKey(fromLabel), formatCypherValue(fromID),
       toLabel, getUniqueKey(toLabel), formatCypherValue(toID),
       edge.Label, propsStr)
}

func formatCypherValue(value interface{}) string {
    switch v := value.(type) {
    case string:
        // Escape quotes
        escaped := strings.ReplaceAll(v, "'", "\\'")
        return fmt.Sprintf("'%s'", escaped)
    case int, int64, float64:
        return fmt.Sprintf("%v", v)
    case bool:
        return fmt.Sprintf("%t", v)
    case []string:
        quoted := make([]string, len(v))
        for i, s := range v {
            quoted[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(s, "'", "\\'"))
        }
        return "[" + strings.Join(quoted, ", ") + "]"
    default:
        return "''"
    }
}

func parseNodeID(nodeID string) (label, id string) {
    parts := strings.SplitN(nodeID, ":", 2)
    if len(parts) == 2 {
        return parts[0], parts[1]
    }
    return "Unknown", nodeID
}

func getUniqueKey(label string) string {
    keys := map[string]string{
        "Commit":      "sha",
        "Developer":   "email",
        "Issue":       "number",
        "PullRequest": "number",
        "File":        "path",
    }
    if key, ok := keys[label]; ok {
        return key
    }
    return "id"
}
```

### Neptune Gremlin Generation

```go
// generateGremlinNode creates Gremlin query with coalesce for idempotency
func generateGremlinNode(node GraphNode) string {
    uniqueKey := getUniqueKey(node.Label)
    uniqueValue := node.Properties[uniqueKey]

    // Build property list
    props := []string{}
    for key, value := range node.Properties {
        props = append(props, fmt.Sprintf(".property('%s', %s)", key, formatGremlinValue(value)))
    }

    return fmt.Sprintf(`
        g.V().has('%s', '%s', %s).
          fold().
          coalesce(
            unfold(),
            addV('%s')%s
          )
    `, node.Label, uniqueKey, formatGremlinValue(uniqueValue),
       node.Label, strings.Join(props, ""))
}

// generateGremlinEdge creates Gremlin query with coalesce for idempotency
func generateGremlinEdge(edge GraphEdge) string {
    fromLabel, fromID := parseNodeID(edge.From)
    toLabel, toID := parseNodeID(edge.To)

    fromKey := getUniqueKey(fromLabel)
    toKey := getUniqueKey(toLabel)

    // Build property list
    props := []string{}
    for key, value := range edge.Properties {
        props = append(props, fmt.Sprintf(".property('%s', %s)", key, formatGremlinValue(value)))
    }

    return fmt.Sprintf(`
        g.V().has('%s', '%s', %s).as('from').
          V().has('%s', '%s', %s).as('to').
          coalesce(
            __.outE('%s').where(inV().as('to')),
            addE('%s').from('from')%s
          )
    `, fromLabel, fromKey, formatGremlinValue(fromID),
       toLabel, toKey, formatGremlinValue(toID),
       edge.Label, edge.Label, strings.Join(props, ""))
}

func formatGremlinValue(value interface{}) string {
    switch v := value.(type) {
    case string:
        // Escape quotes
        escaped := strings.ReplaceAll(v, "'", "\\'")
        return fmt.Sprintf("'%s'", escaped)
    case int, int64:
        return fmt.Sprintf("%d", v)
    case float64:
        return fmt.Sprintf("%f", v)
    case bool:
        return fmt.Sprintf("%t", v)
    case []string:
        quoted := make([]string, len(v))
        for i, s := range v {
            quoted[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(s, "'", "\\'"))
        }
        return "[" + strings.Join(quoted, ", ") + "]"
    default:
        return "''"
    }
}
```

---

## Batch Execution & Checkpointing

### Main Construction Loop

```go
// BuildGraph constructs graph for a repository
func BuildGraph(db *sql.DB, backend Backend, repoID int64) error {
    log.Printf("Building graph for repo %d...", repoID)

    // Process commits
    if err := processCommits(db, backend, repoID); err != nil {
        return fmt.Errorf("process commits failed: %w", err)
    }

    // Process issues
    if err := processIssues(db, backend, repoID); err != nil {
        return fmt.Errorf("process issues failed: %w", err)
    }

    // Process PRs
    if err := processPRs(db, backend, repoID); err != nil {
        return fmt.Errorf("process PRs failed: %w", err)
    }

    log.Printf("✅ Graph construction complete")
    return nil
}

func processCommits(db *sql.DB, backend Backend, repoID int64) error {
    batchSize := 100
    totalProcessed := 0

    for {
        // Fetch unprocessed commits
        commits, err := FetchUnprocessedCommits(db, repoID, batchSize)
        if err != nil {
            return err
        }

        if len(commits) == 0 {
            break // All processed
        }

        // Transform to graph entities
        var allNodes []GraphNode
        var allEdges []GraphEdge
        var commitIDs []int64

        for _, commit := range commits {
            nodes, edges, err := ProcessCommit(commit)
            if err != nil {
                log.Printf("Warning: failed to process commit %s: %v", commit.SHA, err)
                continue
            }

            allNodes = append(allNodes, nodes...)
            allEdges = append(allEdges, edges...)
            commitIDs = append(commitIDs, commit.ID)
        }

        // Create nodes
        if err := backend.CreateNodes(allNodes); err != nil {
            return err
        }

        // Create edges
        if err := backend.CreateEdges(allEdges); err != nil {
            return err
        }

        // Mark as processed in PostgreSQL
        if err := markCommitsProcessed(db, commitIDs); err != nil {
            return err
        }

        totalProcessed += len(commits)
        log.Printf("  Processed %d commits (total: %d)", len(commits), totalProcessed)
    }

    return nil
}

// markCommitsProcessed updates processed_at timestamp
func markCommitsProcessed(db *sql.DB, ids []int64) error {
    query := `
        UPDATE github_commits
        SET processed_at = NOW()
        WHERE id = ANY($1)
    `

    _, err := db.Exec(query, pq.Array(ids))
    return err
}
```

---

## Testing Strategy

### Unit Tests

```go
func TestProcessCommit(t *testing.T) {
    // Load test data from test_data/github_api/
    commitJSON, _ := os.ReadFile("../../../test_data/github_api/omnara-ai-omnara/commits/6f8be10.json")

    commit := CommitData{
        SHA:         "6f8be10...",
        AuthorEmail: "dev@example.com",
        AuthorName:  "Developer",
        AuthorDate:  time.Now(),
        Message:     "fix: update config",
        RawData:     commitJSON,
    }

    nodes, edges, err := ProcessCommit(commit)
    assert.NoError(t, err)

    // Verify nodes
    assert.Len(t, nodes, 2) // Commit + Developer
    assert.Equal(t, "Commit", nodes[0].Label)
    assert.Equal(t, "Developer", nodes[1].Label)

    // Verify edges
    assert.Greater(t, len(edges), 1) // AUTHORED + MODIFIES edges
}
```

### Integration Tests

```go
func TestBuildGraphNeo4j(t *testing.T) {
    // Requires Neo4j running
    backend, err := NewNeo4jBackend("bolt://localhost:7687", "neo4j", "password")
    require.NoError(t, err)
    defer backend.Close()

    // Requires PostgreSQL with test data
    db := setupTestDB(t)
    defer db.Close()

    // Build graph
    err = BuildGraph(db, backend, 1)
    require.NoError(t, err)

    // Verify nodes created
    result, err := backend.Query("MATCH (c:Commit) RETURN count(c) as count")
    require.NoError(t, err)
    assert.Greater(t, result, 0)
}
```

---

## Performance Summary

**From [data_volumes.md](../../01-architecture/data_volumes.md):**

| Repository | Nodes | Edges | Graph Time | Storage |
|------------|-------|-------|------------|---------|
| omnara | 250 | 400 | 40ms | 2 MB |
| kubernetes | 12,500 | 23,000 | 1.75s | 60 MB |

**Both meet performance targets (<2s for kubernetes) ✅**

---

## Next Steps

1. ✅ **Design complete** - Dual backend support validated
2. ✅ **Query generation** - Cypher + Gremlin documented
3. ⏭️ **Implement backend abstraction** - `internal/graph/backend.go`
4. ⏭️ **Implement builders** - `internal/graph/builder.go`
5. ⏭️ **Test with Neo4j** - Local validation
6. ⏭️ **Test with Neptune** - Cloud validation (optional)
7. ⏭️ **Proceed to Priority 6C** - CLI integration

---

## References

- **Graph Ontology:** [graph_ontology.md](../../01-architecture/graph_ontology.md)
- **Data Volumes:** [data_volumes.md](../../01-architecture/data_volumes.md)
- **Neo4j Docs:** https://neo4j.com/docs/cypher-manual/current/
- **Gremlin Docs:** https://tinkerpop.apache.org/docs/current/reference/
- **PostgreSQL Schema:** [scripts/schema/postgresql_staging.sql](../../../scripts/schema/postgresql_staging.sql)
