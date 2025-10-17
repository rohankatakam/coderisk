# Graph Construction Integration Guide

**Purpose:** Implementation guide for building the 3-layer CodeRisk knowledge graph from repository data

**Prerequisites:**
- Docker Compose stack running (see [local_deployment.md](local_deployment.md))
- Go 1.23+
- Repository to analyze (local or GitHub)

**Last Updated:** October 2, 2025

**AI Agent Implementation Note:** This guide is designed for AI agents implementing graph construction. Read alongside [scalability_analysis.md](../../01-architecture/scalability_analysis.md) for architectural decisions and [graph_ontology.md](../../01-architecture/graph_ontology.md) for schema details.

---

## Architecture Context

**References:**
- **[scalability_analysis.md](../../01-architecture/scalability_analysis.md)** - Scalability validation and implementation decisions
- **[graph_ontology.md](../../01-architecture/graph_ontology.md)** - 3-layer graph schema
- **[data_volumes.md](../../01-architecture/data_volumes.md)** - Real data volume analysis from omnara and kubernetes
- **[spec.md](../../spec.md)** - System requirements and constraints
- **[GitHub API Test Data](../../../test_data/github_api/README.md)** - Real API response schemas

**Implementation Guides (Priority 6):**
- **[layer_1_treesitter.md](layer_1_treesitter.md)** - Layer 1: Code Structure (Tree-sitter AST parsing)
- **[layers_2_3_github_fetching.md](layers_2_3_github_fetching.md)** - Priority 6A: GitHub API → PostgreSQL (Stage 1)
- **[layers_2_3_graph_construction.md](layers_2_3_graph_construction.md)** - Priority 6B: PostgreSQL → Neo4j/Neptune (Stage 2)
- **[cli_integration.md](cli_integration.md)** - Priority 6C: End-to-end `crisk init` command

**Schema & Scripts:**
- **[PostgreSQL Schema](../../../scripts/schema/postgresql_staging.sql)** - Staging database schema (8 tables)

**Validated Scale (Based on Real Data):**
- **Small repos (omnara-ai/omnara):** 100 commits, 251 issues, 180 PRs, ~23s init, 5.6MB storage
- **Enterprise repos (kubernetes/kubernetes):** 5K commits (90-day), 5K issues, 2K PRs, ~2min init, 168MB storage

**Key Decisions:**
1. ✅ Shallow clone (`--depth 1`) for repository download
2. ✅ Static parsing with Tree-sitter (pre-computed during init)
3. ✅ 90-day window for commits (balances recency with data volume)
4. ✅ Filter incidents by recency (<90 days) and relevance
5. ✅ Skip embeddings in v1.0 (store raw text in PostgreSQL)
6. ✅ Process both open and recent closed issues/PRs
7. ✅ Store diffs for commits and PRs (needed for FIXES edges)

---

## Overview

Graph construction follows a **2-stage pipeline** with **3-layer ontology**:

### Data Pipeline (2 Stages)

```
Stage 1: GitHub API → PostgreSQL (Staging)
   ↓ Fetch & store raw JSON (22s omnara, 2min kubernetes)
Stage 2: PostgreSQL → Neptune (Graph Construction)
   ↓ Parse & transform (40ms omnara, 1.75s kubernetes)
```

**Benefits of Staging Layer:**
- ✅ Idempotent (can re-run without re-fetching)
- ✅ Resumable (failures don't lose progress)
- ✅ Auditable (full history of fetches)
- ✅ Testable (can test graph construction offline)

### Graph Ontology (3 Layers)

```
Layer 1: Structure (Tree-sitter AST)
   ↓ File, Function, Class nodes + CALLS, IMPORTS edges
Layer 2: Temporal (Git History)
   ↓ Commit, Developer nodes + AUTHORED, MODIFIES edges
Layer 3: Incidents (Issues/PRs)
   ↓ Issue, PR nodes + FIXES, MERGED_TO edges
```

**Total Time:** ~23s (omnara) to ~2min (kubernetes)

**See:** Priority 6 implementation guides above for detailed Stage 1 (fetching) and Stage 2 (graph construction) instructions.

---

## Phase 1: Repository Cloning & Structure Extraction

### Step 1: Clone Repository

**Implementation: `internal/ingestion/clone.go`**

```go
package ingestion

import (
    "context"
    "crypto/sha256"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
)

// CloneRepository performs shallow clone for fast initial download
// Reference: scalability_analysis.md - Git Clone Strategy
func CloneRepository(ctx context.Context, url string) (*Repository, error) {
    // Generate unique hash for repo storage
    hash := fmt.Sprintf("%x", sha256.Sum256([]byte(url)))[:12]
    repoPath := filepath.Join(os.Getenv("HOME"), ".coderisk", "repos", hash)

    // Check if already cloned
    if _, err := os.Stat(repoPath); err == nil {
        return &Repository{Path: repoPath, URL: url}, nil
    }

    // Shallow clone (--depth 1) for speed
    // Reference: scalability_analysis.md - Shallow clone reduces omnara from 74MB to ~20MB
    cmd := exec.CommandContext(ctx, "git", "clone",
        "--depth", "1",
        "--single-branch",
        url,
        repoPath,
    )

    if output, err := cmd.CombinedOutput(); err != nil {
        return nil, fmt.Errorf("git clone failed: %w, output: %s", err, output)
    }

    return &Repository{Path: repoPath, URL: url}, nil
}

type Repository struct {
    Path string
    URL  string
}
```

**Usage:**
```bash
# Test with omnara-ai/omnara (baseline)
repo, err := CloneRepository(ctx, "https://github.com/omnara-ai/omnara.git")
# Expected: ~5-10s download, ~20MB disk usage

# Test with kubernetes/kubernetes (stress test)
repo, err := CloneRepository(ctx, "https://github.com/kubernetes/kubernetes.git")
# Expected: ~60s download, ~400MB disk usage (vs 1.4GB full clone)
```

**Performance Validation:**
- **omnara:** 5-10s clone time
- **kubernetes:** 60s clone time (90% faster than full clone)

### Step 2: Tree-sitter Parsing (Layer 1)

**Implementation: `internal/ingestion/treesitter.go`**

```go
package ingestion

import (
    "context"
    "os"
    "path/filepath"

    sitter "github.com/smacker/go-tree-sitter"
    "github.com/smacker/go-tree-sitter/golang"
    "github.com/smacker/go-tree-sitter/javascript"
    "github.com/smacker/go-tree-sitter/python"
    "github.com/smacker/go-tree-sitter/typescript"
)

// ParseRepository extracts code structure for Layer 1
// Reference: graph_ontology.md - Layer 1: Structure
func ParseRepository(ctx context.Context, repoPath string, neo4j *graph.Client) (*StructureGraph, error) {
    stats := &StructureGraph{
        Files:     0,
        Functions: 0,
        Classes:   0,
    }

    // Language parsers (v1.0 support: Go, TypeScript, JavaScript, Python)
    // Reference: spec.md §5.2 - Technology stack
    parsers := map[string]*sitter.Language{
        ".go":   golang.GetLanguage(),
        ".js":   javascript.GetLanguage(),
        ".ts":   typescript.GetLanguage(),
        ".tsx":  typescript.GetLanguage(),
        ".py":   python.GetLanguage(),
    }

    // Walk repository files
    err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
        if err != nil || info.IsDir() {
            return err
        }

        ext := filepath.Ext(path)
        lang, supported := parsers[ext]
        if !supported {
            return nil // Skip unsupported files
        }

        // Parse file
        content, _ := os.ReadFile(path)
        parser := sitter.NewParser()
        parser.SetLanguage(lang)
        tree := parser.Parse(nil, content)

        // Extract nodes from AST
        relPath, _ := filepath.Rel(repoPath, path)
        nodes := extractNodes(tree.RootNode(), relPath, content)

        // Insert into Neo4j
        if err := insertStructureNodes(ctx, neo4j, nodes); err != nil {
            return err
        }

        stats.Files++
        stats.Functions += nodes.FunctionCount
        stats.Classes += nodes.ClassCount

        return nil
    })

    return stats, err
}

type StructureGraph struct {
    Files     int
    Functions int
    Classes   int
}

// extractNodes parses AST and creates graph nodes
func extractNodes(root *sitter.Node, filePath string, source []byte) *ASTNodes {
    nodes := &ASTNodes{
        FilePath: filePath,
    }

    // Traverse AST (language-specific queries)
    cursor := sitter.NewTreeCursor(root)
    defer cursor.Close()

    // Example: Extract functions (Go)
    // function_declaration @function
    if cursor.CurrentNode().Type() == "function_declaration" {
        funcNode := cursor.CurrentNode()
        nameNode := funcNode.ChildByFieldName("name")

        nodes.Functions = append(nodes.Functions, Function{
            Name:      string(source[nameNode.StartByte():nameNode.EndByte()]),
            StartLine: funcNode.StartPoint().Row + 1,
            EndLine:   funcNode.EndPoint().Row + 1,
        })
        nodes.FunctionCount++
    }

    // Extract imports, classes, etc.
    // ... (similar logic for each node type)

    return nodes
}

type ASTNodes struct {
    FilePath      string
    Functions     []Function
    Classes       []Class
    Imports       []Import
    FunctionCount int
    ClassCount    int
}

type Function struct {
    Name      string
    StartLine int
    EndLine   int
}

type Class struct {
    Name     string
    IsPublic bool
}

type Import struct {
    Path string
}
```

**Neo4j Insertion:**

```go
// insertStructureNodes creates Layer 1 nodes and edges
// Reference: graph_ontology.md - Layer 1 schema
func insertStructureNodes(ctx context.Context, neo4j *graph.Client, nodes *ASTNodes) error {
    session := neo4j.NewSession()
    defer session.Close(ctx)

    // Create File node
    _, err := session.Run(ctx, `
        MERGE (f:File {path: $path})
        SET f.language = $language,
            f.loc = $loc,
            f.last_modified_sha = $sha
    `, map[string]interface{}{
        "path":     nodes.FilePath,
        "language": detectLanguage(nodes.FilePath),
        "loc":      calculateLOC(nodes),
        "sha":      "", // Will be set in Phase 2
    })
    if err != nil {
        return err
    }

    // Create Function nodes and CONTAINS edges
    for _, fn := range nodes.Functions {
        _, err := session.Run(ctx, `
            MATCH (f:File {path: $file_path})
            MERGE (fn:Function {name: $name, file_path: $file_path})
            SET fn.start_line = $start_line,
                fn.end_line = $end_line
            MERGE (f)-[:CONTAINS]->(fn)
        `, map[string]interface{}{
            "file_path":  nodes.FilePath,
            "name":       fn.Name,
            "start_line": fn.StartLine,
            "end_line":   fn.EndLine,
        })
        if err != nil {
            return err
        }
    }

    // Create IMPORTS edges
    for _, imp := range nodes.Imports {
        _, err := session.Run(ctx, `
            MATCH (f:File {path: $from_path})
            MERGE (to:File {path: $to_path})
            MERGE (f)-[:IMPORTS]->(to)
        `, map[string]interface{}{
            "from_path": nodes.FilePath,
            "to_path":   imp.Path,
        })
        if err != nil {
            return err
        }
    }

    return nil
}
```

**Performance Targets (from scalability_analysis.md):**
- **omnara (1K files):** ~15-20s parsing
- **kubernetes (50K files):** ~5-8min parsing

---

## Phase 2: Git History Extraction (Layer 2)

### Step 3: Extract Commit History (90-day window)

**Implementation: `internal/ingestion/git_history.go`**

```go
package ingestion

import (
    "bufio"
    "context"
    "os/exec"
    "strconv"
    "strings"
    "time"
)

// ExtractCommitHistory builds Layer 2 from git log
// Reference: scalability_analysis.md - 90-day window strategy
func ExtractCommitHistory(ctx context.Context, repoPath string, neo4j *graph.Client) (*TemporalGraph, error) {
    stats := &TemporalGraph{
        Commits:   0,
        CoChanged: 0,
    }

    // Fetch commits from last 90 days with file stats
    // Reference: scalability_analysis.md - Use git log --numstat
    since := time.Now().AddDate(0, 0, -90).Format("2006-01-02")
    cmd := exec.CommandContext(ctx, "git", "log",
        "--since="+since,
        "--numstat",
        "--pretty=format:%H|%an|%ae|%at|%s",
    )
    cmd.Dir = repoPath

    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("git log failed: %w", err)
    }

    // Parse git log output
    commits := parseGitLog(string(output))

    // Insert commits into Neo4j
    for _, commit := range commits {
        if err := insertCommit(ctx, neo4j, commit); err != nil {
            return nil, err
        }
        stats.Commits++
    }

    // Calculate CO_CHANGED edges
    // Reference: graph_ontology.md - Co-change calculation
    coChangeCount, err := calculateCoChange(ctx, neo4j)
    if err != nil {
        return nil, err
    }
    stats.CoChanged = coChangeCount

    return stats, nil
}

type TemporalGraph struct {
    Commits   int
    CoChanged int
}

type Commit struct {
    SHA       string
    Author    string
    Email     string
    Timestamp time.Time
    Message   string
    Files     []FileChange
}

type FileChange struct {
    Path      string
    Additions int
    Deletions int
}

// parseGitLog parses git log --numstat output
func parseGitLog(output string) []Commit {
    var commits []Commit
    var current *Commit

    scanner := bufio.NewScanner(strings.NewReader(output))
    for scanner.Scan() {
        line := scanner.Text()

        if strings.Contains(line, "|") {
            // Commit line: SHA|author|email|timestamp|message
            parts := strings.Split(line, "|")
            timestamp, _ := strconv.ParseInt(parts[3], 10, 64)

            current = &Commit{
                SHA:       parts[0],
                Author:    parts[1],
                Email:     parts[2],
                Timestamp: time.Unix(timestamp, 0),
                Message:   parts[4],
                Files:     []FileChange{},
            }
            commits = append(commits, *current)
        } else if line != "" && current != nil {
            // File change line: additions deletions path
            parts := strings.Fields(line)
            if len(parts) == 3 {
                adds, _ := strconv.Atoi(parts[0])
                dels, _ := strconv.Atoi(parts[1])
                current.Files = append(current.Files, FileChange{
                    Path:      parts[2],
                    Additions: adds,
                    Deletions: dels,
                })
            }
        }
    }

    return commits
}

// insertCommit creates Layer 2 nodes and edges
func insertCommit(ctx context.Context, neo4j *graph.Client, commit Commit) error {
    session := neo4j.NewSession()
    defer session.Close(ctx)

    // Create Commit node and Developer node
    _, err := session.Run(ctx, `
        MERGE (d:Developer {email: $email})
        SET d.name = $author,
            d.last_commit = $timestamp

        MERGE (c:Commit {sha: $sha})
        SET c.timestamp = datetime($timestamp),
            c.message = $message,
            c.additions = $additions,
            c.deletions = $deletions

        MERGE (d)-[:AUTHORED]->(c)
    `, map[string]interface{}{
        "email":     commit.Email,
        "author":    commit.Author,
        "sha":       commit.SHA,
        "timestamp": commit.Timestamp.Format(time.RFC3339),
        "message":   commit.Message,
        "additions": sumAdditions(commit.Files),
        "deletions": sumDeletions(commit.Files),
    })
    if err != nil {
        return err
    }

    // Create MODIFIES edges to files
    for _, fc := range commit.Files {
        _, err := session.Run(ctx, `
            MATCH (c:Commit {sha: $sha})
            MERGE (f:File {path: $path})
            MERGE (c)-[m:MODIFIES]->(f)
            SET m.additions = $additions,
                m.deletions = $deletions
        `, map[string]interface{}{
            "sha":       commit.SHA,
            "path":      fc.Path,
            "additions": fc.Additions,
            "deletions": fc.Deletions,
        })
        if err != nil {
            return err
        }
    }

    return nil
}

// calculateCoChange creates CO_CHANGED edges based on co-occurrence
// Reference: graph_ontology.md - Co-change frequency calculation
func calculateCoChange(ctx context.Context, neo4j *graph.Client) (int, error) {
    session := neo4j.NewSession()
    defer session.Close(ctx)

    // Find files that changed together in same commits
    result, err := session.Run(ctx, `
        MATCH (a:File)<-[:MODIFIES]-(c:Commit)-[:MODIFIES]->(b:File)
        WHERE a.path < b.path  // Avoid duplicates
        WITH a, b, count(c) as co_change_count
        WHERE co_change_count > 1

        MATCH (a)<-[:MODIFIES]-(ac:Commit)
        WITH a, b, co_change_count, count(DISTINCT ac) as total_a_commits

        MATCH (b)<-[:MODIFIES]-(bc:Commit)
        WITH a, b, co_change_count, total_a_commits, count(DISTINCT bc) as total_b_commits

        WITH a, b,
             toFloat(co_change_count) / toFloat(total_a_commits + total_b_commits - co_change_count) as frequency
        WHERE frequency > 0.3  // Only store meaningful co-changes

        MERGE (a)-[r:CO_CHANGED]-(b)
        SET r.frequency = frequency,
            r.last_timestamp = datetime(),
            r.window_days = 90

        RETURN count(r) as co_change_edges
    `, nil)

    if err != nil {
        return 0, err
    }

    record := result.Next(ctx)
    count := record.Values[0].(int64)

    return int(count), nil
}
```

**Performance Targets (from scalability_analysis.md):**
- **omnara (300 commits in 90 days):** 3-5s
- **kubernetes (5,000 commits in 90 days):** 50-100s

---

## Phase 3: GitHub Issues/PRs (Layer 3)

### Step 4: Fetch Incidents (Optional, filtered by relevance)

**Implementation: `internal/ingestion/github.go`**

```go
package ingestion

import (
    "context"
    "time"

    "github.com/google/go-github/v57/github"
    "golang.org/x/oauth2"
)

// FetchIncidents retrieves issues and PRs from GitHub
// Reference: scalability_analysis.md - Filter by 90-day recency
func FetchIncidents(ctx context.Context, owner, repo string, neo4j *graph.Client, pg *database.Client) (*IncidentGraph, error) {
    // Initialize GitHub client (requires GITHUB_TOKEN)
    ts := oauth2.StaticTokenSource(
        &oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
    )
    tc := oauth2.NewClient(ctx, ts)
    client := github.NewClient(tc)

    stats := &IncidentGraph{
        Issues: 0,
        PRs:    0,
    }

    // Filter: Only issues from last 90 days OR currently open
    // Reference: scalability_analysis.md - Reduces kubernetes 50k → 10k (80%)
    since := time.Now().AddDate(0, 0, -90)

    opts := &github.IssueListByRepoOptions{
        State: "all",
        Since: since,
        ListOptions: github.ListOptions{PerPage: 100},
    }

    for {
        issues, resp, err := client.Issues.ListByRepo(ctx, owner, repo, opts)
        if err != nil {
            return nil, err
        }

        for _, issue := range issues {
            // Filter: Skip if closed >90 days ago
            if issue.ClosedAt != nil && issue.ClosedAt.Before(since) {
                continue
            }

            // Store in PostgreSQL (no embeddings in v1.0)
            // Reference: scalability_analysis.md - Hybrid approach
            if err := storeIncident(ctx, pg, issue); err != nil {
                return nil, err
            }

            // Create Neo4j node and relationships
            if err := linkIncident(ctx, neo4j, issue); err != nil {
                return nil, err
            }

            if issue.IsPullRequest() {
                stats.PRs++
            } else {
                stats.Issues++
            }
        }

        if resp.NextPage == 0 {
            break
        }
        opts.Page = resp.NextPage
    }

    return stats, nil
}

type IncidentGraph struct {
    Issues int
    PRs    int
}

// storeIncident saves incident text in PostgreSQL
// Reference: scalability_analysis.md - No embeddings, store raw text
func storeIncident(ctx context.Context, pg *database.Client, issue *github.Issue) error {
    query := `
        INSERT INTO incidents (
            number, title, body, state, created_at, closed_at, is_pr
        ) VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT (number) DO UPDATE
        SET title = EXCLUDED.title,
            body = EXCLUDED.body,
            state = EXCLUDED.state,
            closed_at = EXCLUDED.closed_at
    `

    _, err := pg.Exec(ctx, query,
        issue.GetNumber(),
        issue.GetTitle(),
        issue.GetBody(),
        issue.GetState(),
        issue.GetCreatedAt(),
        issue.ClosedAt,
        issue.IsPullRequest(),
    )

    return err
}

// linkIncident creates Neo4j relationships to affected files
func linkIncident(ctx context.Context, neo4j *graph.Client, issue *github.Issue) error {
    session := neo4j.NewSession()
    defer session.Close(ctx)

    // Create Issue/PR node
    nodeType := "Issue"
    if issue.IsPullRequest() {
        nodeType = "PullRequest"
    }

    _, err := session.Run(ctx, fmt.Sprintf(`
        MERGE (i:%s {number: $number})
        SET i.title = $title,
            i.state = $state,
            i.created_at = datetime($created_at),
            i.closed_at = datetime($closed_at)
    `, nodeType), map[string]interface{}{
        "number":     issue.GetNumber(),
        "title":      issue.GetTitle(),
        "state":      issue.GetState(),
        "created_at": issue.GetCreatedAt().Format(time.RFC3339),
        "closed_at":  formatNullTime(issue.ClosedAt),
    })

    if err != nil {
        return err
    }

    // Create MENTIONS edges (if issue body mentions file paths)
    // Pattern: Extract "src/file.go" from issue body
    files := extractFilePaths(issue.GetBody())
    for _, file := range files {
        _, err := session.Run(ctx, fmt.Sprintf(`
            MATCH (i:%s {number: $number})
            MERGE (f:File {path: $path})
            MERGE (i)-[:MENTIONS]->(f)
        `, nodeType), map[string]interface{}{
            "number": issue.GetNumber(),
            "path":   file,
        })
        if err != nil {
            return err
        }
    }

    return nil
}
```

**GitHub API Rate Limiting:**
- **Authenticated:** 5,000 requests/hour
- **omnara:** ~2 requests (100 issues in 90 days)
- **kubernetes:** ~100 requests (10k issues in 90 days)

**Storage Strategy:**
- **PostgreSQL:** Raw text (JSONB), full-text search (tsvector)
- **Neo4j:** Nodes and MENTIONS/FIXES edges only
- **No embeddings** in v1.0 (added on-demand in Phase 2)

---

## Performance Validation

### Test with omnara-ai/omnara (Baseline)

```bash
# Expected results from scalability_analysis.md

# Phase 1: Clone + Parse
time crisk init https://github.com/omnara-ai/omnara.git
# Expected: 30s total
#   - Clone: 5-10s
#   - Parse: 15-20s
#   - Storage: 95MB

# Verify Neo4j graph
docker compose exec neo4j cypher-shell -u neo4j -p PASSWORD "
MATCH (n)
RETURN labels(n) as type, count(n) as count
ORDER BY count DESC
"
# Expected:
#   File: ~1,000
#   Function: ~5,000
#   Commit: ~300
#   Developer: ~10
```

### Test with kubernetes/kubernetes (Stress Test)

```bash
# Expected results from scalability_analysis.md

# Phase 1: Clone + Parse
time crisk init https://github.com/kubernetes/kubernetes.git
# Expected: 10min total
#   - Clone: 60s
#   - Parse: 5-8min
#   - Commit extraction: 50-100s
#   - Storage: 4.6GB

# Verify graph size
docker compose exec neo4j cypher-shell -u neo4j -p PASSWORD "
MATCH (n)
RETURN count(n) as nodes
"
# Expected: ~300,000 nodes

docker compose exec neo4j cypher-shell -u neo4j -p PASSWORD "
MATCH ()-[r]->()
RETURN count(r) as edges
"
# Expected: ~1,000,000 edges
```

---

## Storage Requirements

**From scalability_analysis.md:**

| Repository | Files | Neo4j Graph | PostgreSQL | Redis Cache | **Total** |
|-----------|-------|-------------|------------|-------------| ---------|
| omnara    | 1K    | 35MB        | 10MB       | 50MB        | **95MB**  |
| kubernetes| 50K   | 3.6GB       | 500MB      | 500MB       | **4.6GB** |

**Conclusion:** Architecture scales from small to enterprise repos.

---

## Troubleshooting

### Issue: Tree-sitter parse errors

**Symptom:**
```
Error: invalid syntax at line 42
Failed to parse src/file.go
```

**Cause:** Unsupported language version or syntax

**Solution:**
```bash
# Update tree-sitter grammar
go get -u github.com/smacker/go-tree-sitter/golang

# Skip files with parse errors (non-blocking)
# Add to error handler:
if err := parseFile(file); err != nil {
    log.Warn("Skipping file due to parse error", "file", file, "error", err)
    return nil  // Continue with next file
}
```

### Issue: GitHub API rate limit exceeded

**Symptom:**
```
Error: API rate limit exceeded
403 Forbidden
```

**Cause:** 5,000 requests/hour limit reached

**Solution:**
```bash
# Check remaining rate limit
curl -H "Authorization: token $GITHUB_TOKEN" \
  https://api.github.com/rate_limit

# Wait or reduce scope:
# Option 1: Process only open issues
opts.State = "open"

# Option 2: Increase time window
since := time.Now().AddDate(0, 0, -30)  // 30 days instead of 90
```

### Issue: Neo4j out of memory during init

**Symptom:**
```
Error: Java heap space
Neo4j crashes during graph construction
```

**Cause:** Insufficient heap for large repos

**Solution:**
```bash
# Increase Neo4j heap in .env
NEO4J_MAX_HEAP=8G  # Increase from 2G
NEO4J_PAGECACHE=4G  # Increase from 1G

# Batch inserts (reduce memory pressure)
# Process files in batches of 100:
for i := 0; i < len(files); i += 100 {
    batch := files[i:min(i+100, len(files))]
    insertBatch(ctx, neo4j, batch)
}
```

---

## Next Steps

After graph construction is complete:

1. **Test Tier 1 metrics** - Run `crisk check` to verify coupling, co-change, test_ratio calculations
2. **Validate performance** - Ensure Phase 1 completes in <500ms
3. **Verify cache hit rate** - Check Redis cache >90% hit rate after warm-up
4. **Implement incremental updates** - Add `crisk sync` for graph updates without full rebuild

---

## References

**Architecture:**
- [scalability_analysis.md](../../01-architecture/scalability_analysis.md) - Scalability validation and decisions
- [graph_ontology.md](../../01-architecture/graph_ontology.md) - Graph schema
- [spec.md](../../spec.md) - System requirements

**Implementation:**
- [local_deployment.md](local_deployment.md) - Docker Compose setup
- [DEVELOPMENT_WORKFLOW.md](../../DEVELOPMENT_WORKFLOW.md) - Implementation guardrails

**External:**
- [go-tree-sitter documentation](https://github.com/smacker/go-tree-sitter)
- [GitHub API v3 documentation](https://docs.github.com/en/rest)
- [Neo4j Cypher documentation](https://neo4j.com/docs/cypher-manual/)

---

**Last Updated:** October 2, 2025
