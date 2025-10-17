# Phase 1: Dual-Mode Architecture Implementation

**Version:** 1.0
**Last Updated:** October 10, 2025
**Purpose:** Enable CodeRisk to run in both Cloud and Local modes with unified codebase
**Target:** MVP for production distribution (Homebrew, pip, package managers)

---

## Executive Summary

**Goal:** Allow users to install `crisk` via package managers and choose between:
1. **Cloud Mode** (default) - Zero setup, connects to hosted backend
2. **Local Mode** (advanced) - Docker-based, all data local, privacy-first

**Key Principle:** Same CLI binary, same codebase, mode selected at runtime

**User Experience:**
```bash
# Install via package manager (Homebrew example)
brew install coderisk/tap/crisk

# First run - prompts for mode selection
crisk init-local

# Cloud mode: Works immediately (with API key)
# Local mode: Auto-checks Docker, guides user through setup
```

---

## Architecture Overview

### Current State (Local-Only)
```
┌──────────────┐
│  crisk CLI   │
└──────────────┘
       ↓
┌──────────────────────────────────────┐
│  Docker Compose (Required)           │
│  - Neo4j (bolt://localhost:7687)     │
│  - PostgreSQL (localhost:5432)       │
│  - Redis (localhost:6379)            │
└──────────────────────────────────────┘
```

**Problems:**
- ❌ Requires Docker Desktop installation
- ❌ Manual `make start` before using crisk
- ❌ Not suitable for package manager distribution
- ❌ Poor UX for non-technical users

### Target State (Dual-Mode)
```
┌──────────────────────────────────────┐
│  crisk CLI (Single Binary)           │
│  - Detects mode (cloud/local)        │
│  - Routes to appropriate backend     │
└──────────────────────────────────────┘
       ↓                    ↓
┌──────────────┐   ┌──────────────────┐
│  Cloud Mode  │   │   Local Mode     │
│              │   │                  │
│  REST API    │   │  Docker Compose  │
│  to hosted   │   │  (same as today) │
│  services    │   │                  │
└──────────────┘   └──────────────────┘
       ↓                    ↓
┌──────────────┐   ┌──────────────────┐
│ Your Backend │   │  Local Services  │
│              │   │                  │
│ - Neptune    │   │  - Neo4j         │
│ - PostgreSQL │   │  - PostgreSQL    │
│ - Redis      │   │  - Redis         │
└──────────────┘   └──────────────────┘
```

---

## Implementation Roadmap

### Phase 1.1: Backend Abstraction Layer (Week 1)

**Goal:** Create unified interface for both cloud and local backends

#### Step 1: Define Backend Interface

**File:** `internal/backend/backend.go`

```go
package backend

import "context"

// Backend is the unified interface for graph operations
// Implementations: CloudBackend (REST API), LocalBackend (direct Neo4j)
type Backend interface {
    // Graph operations
    CreateNodes(ctx context.Context, nodes []GraphNode) ([]string, error)
    CreateEdges(ctx context.Context, edges []GraphEdge) error
    Query(ctx context.Context, query string) (interface{}, error)

    // Health check
    HealthCheck(ctx context.Context) error

    // Cleanup
    Close() error
}

// GraphNode represents a node in the knowledge graph
type GraphNode struct {
    Label      string                 `json:"label"`
    ID         string                 `json:"id"`
    Properties map[string]interface{} `json:"properties"`
}

// GraphEdge represents an edge in the knowledge graph
type GraphEdge struct {
    Label      string                 `json:"label"`
    From       string                 `json:"from"`
    To         string                 `json:"to"`
    Properties map[string]interface{} `json:"properties"`
}
```

#### Step 2: Implement Cloud Backend

**File:** `internal/backend/cloud.go`

```go
package backend

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

// CloudBackend connects to CodeRisk hosted services via REST API
type CloudBackend struct {
    apiKey   string
    endpoint string // e.g., "https://api.coderisk.dev"
    client   *http.Client
}

// NewCloudBackend creates a cloud backend instance
func NewCloudBackend(apiKey, endpoint string) (*CloudBackend, error) {
    if apiKey == "" {
        return nil, fmt.Errorf("API key required for cloud mode. Get yours at: https://coderisk.dev/signup")
    }

    return &CloudBackend{
        apiKey:   apiKey,
        endpoint: endpoint,
        client:   &http.Client{Timeout: 30 * time.Second},
    }, nil
}

// CreateNodes sends graph nodes to cloud backend
func (c *CloudBackend) CreateNodes(ctx context.Context, nodes []GraphNode) ([]string, error) {
    url := fmt.Sprintf("%s/v1/graph/nodes", c.endpoint)

    payload, _ := json.Marshal(map[string]interface{}{
        "nodes": nodes,
    })

    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
    if err != nil {
        return nil, err
    }

    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("cloud backend unavailable: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("cloud backend error: %s", resp.Status)
    }

    var result struct {
        IDs []string `json:"ids"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return result.IDs, nil
}

// CreateEdges sends graph edges to cloud backend
func (c *CloudBackend) CreateEdges(ctx context.Context, edges []GraphEdge) error {
    url := fmt.Sprintf("%s/v1/graph/edges", c.endpoint)

    payload, _ := json.Marshal(map[string]interface{}{
        "edges": edges,
    })

    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
    if err != nil {
        return err
    }

    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return fmt.Errorf("cloud backend unavailable: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("cloud backend error: %s", resp.Status)
    }

    return nil
}

// Query executes a graph query via cloud API
func (c *CloudBackend) Query(ctx context.Context, query string) (interface{}, error) {
    url := fmt.Sprintf("%s/v1/graph/query", c.endpoint)

    payload, _ := json.Marshal(map[string]interface{}{
        "query": query,
    })

    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
    if err != nil {
        return nil, err
    }

    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("cloud backend unavailable: %w", err)
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return result["data"], nil
}

// HealthCheck pings cloud backend
func (c *CloudBackend) HealthCheck(ctx context.Context) error {
    url := fmt.Sprintf("%s/health", c.endpoint)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return err
    }

    resp, err := c.client.Do(req)
    if err != nil {
        return fmt.Errorf("cloud backend unreachable: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("cloud backend unhealthy: %s", resp.Status)
    }

    return nil
}

// Close is a no-op for cloud backend (stateless HTTP client)
func (c *CloudBackend) Close() error {
    return nil
}
```

#### Step 3: Refactor Local Backend

**File:** `internal/backend/local.go`

```go
package backend

import (
    "context"

    "github.com/coderisk/coderisk-go/internal/graph"
)

// LocalBackend wraps existing Neo4j backend
type LocalBackend struct {
    neo4j *graph.Neo4jBackend
}

// NewLocalBackend creates a local backend instance
func NewLocalBackend(ctx context.Context, neo4jURI, username, password string) (*LocalBackend, error) {
    neo4j, err := graph.NewNeo4jBackend(ctx, neo4jURI, username, password)
    if err != nil {
        return nil, fmt.Errorf("local Neo4j unavailable: %w\n"+
            "Hint: Start services with: make start", err)
    }

    return &LocalBackend{
        neo4j: neo4j,
    }, nil
}

// CreateNodes delegates to Neo4j backend
func (l *LocalBackend) CreateNodes(ctx context.Context, nodes []GraphNode) ([]string, error) {
    // Convert backend.GraphNode to graph.GraphNode
    graphNodes := make([]graph.GraphNode, len(nodes))
    for i, node := range nodes {
        graphNodes[i] = graph.GraphNode{
            Label:      node.Label,
            ID:         node.ID,
            Properties: node.Properties,
        }
    }

    return l.neo4j.CreateNodes(graphNodes)
}

// CreateEdges delegates to Neo4j backend
func (l *LocalBackend) CreateEdges(ctx context.Context, edges []GraphEdge) error {
    graphEdges := make([]graph.GraphEdge, len(edges))
    for i, edge := range edges {
        graphEdges[i] = graph.GraphEdge{
            Label:      edge.Label,
            From:       edge.From,
            To:         edge.To,
            Properties: edge.Properties,
        }
    }

    return l.neo4j.CreateEdges(graphEdges)
}

// Query delegates to Neo4j backend
func (l *LocalBackend) Query(ctx context.Context, query string) (interface{}, error) {
    return l.neo4j.Query(query)
}

// HealthCheck verifies Neo4j connectivity
func (l *LocalBackend) HealthCheck(ctx context.Context) error {
    _, err := l.neo4j.Query("RETURN 1")
    return err
}

// Close closes Neo4j connection
func (l *LocalBackend) Close() error {
    return l.neo4j.Close()
}
```

#### Step 4: Backend Factory

**File:** `internal/backend/factory.go`

```go
package backend

import (
    "context"
    "fmt"
    "os"

    "github.com/coderisk/coderisk-go/internal/config"
)

// NewBackend creates the appropriate backend based on configuration
func NewBackend(ctx context.Context, cfg *config.Config) (Backend, error) {
    // Check for explicit mode override
    mode := os.Getenv("CODERISK_MODE")
    if mode == "" {
        mode = cfg.Mode // From config file
    }

    switch mode {
    case "cloud":
        return newCloudBackend(cfg)
    case "local":
        return newLocalBackend(ctx, cfg)
    case "":
        // No mode set - prompt user or use smart defaults
        return promptUserForMode(ctx, cfg)
    default:
        return nil, fmt.Errorf("unknown mode: %s (valid: cloud, local)", mode)
    }
}

// newCloudBackend creates cloud backend from config
func newCloudBackend(cfg *config.Config) (Backend, error) {
    apiKey := os.Getenv("CODERISK_API_KEY")
    if apiKey == "" {
        apiKey = cfg.Cloud.APIKey
    }

    endpoint := cfg.Cloud.Endpoint
    if endpoint == "" {
        endpoint = "https://api.coderisk.dev" // Default production endpoint
    }

    return NewCloudBackend(apiKey, endpoint)
}

// newLocalBackend creates local backend from config
func newLocalBackend(ctx context.Context, cfg *config.Config) (Backend, error) {
    neo4jURI := cfg.Local.Neo4jURI
    if neo4jURI == "" {
        neo4jURI = "bolt://localhost:7687"
    }

    username := cfg.Local.Neo4jUser
    if username == "" {
        username = "neo4j"
    }

    password := cfg.Local.Neo4jPassword
    if password == "" {
        password = os.Getenv("NEO4J_PASSWORD")
    }

    return NewLocalBackend(ctx, neo4jURI, username, password)
}

// promptUserForMode guides user through mode selection
func promptUserForMode(ctx context.Context, cfg *config.Config) (Backend, error) {
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Println("Welcome to CodeRisk!")
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Println("")
    fmt.Println("Choose how you want to run CodeRisk:")
    fmt.Println("")
    fmt.Println("1. Cloud Mode (recommended)")
    fmt.Println("   ✓ No setup required")
    fmt.Println("   ✓ Instant start")
    fmt.Println("   ✓ Free for open source projects")
    fmt.Println("   → Get API key: https://coderisk.dev/signup")
    fmt.Println("")
    fmt.Println("2. Local Mode (advanced)")
    fmt.Println("   ✓ All data on your machine")
    fmt.Println("   ✓ No cloud dependency")
    fmt.Println("   ✗ Requires Docker Desktop")
    fmt.Println("   → Run: make start")
    fmt.Println("")
    fmt.Print("Choice (1/2): ")

    var choice string
    fmt.Scanln(&choice)

    switch choice {
    case "1":
        fmt.Println("")
        fmt.Print("Enter your API key (or press Enter to visit signup page): ")
        var apiKey string
        fmt.Scanln(&apiKey)

        if apiKey == "" {
            fmt.Println("Opening browser to: https://coderisk.dev/signup")
            // TODO: Open browser
            return nil, fmt.Errorf("API key required. Get yours at: https://coderisk.dev/signup")
        }

        // Save to config
        cfg.Mode = "cloud"
        cfg.Cloud.APIKey = apiKey
        cfg.Save()

        return NewCloudBackend(apiKey, "https://api.coderisk.dev")

    case "2":
        fmt.Println("")
        fmt.Println("Checking Docker...")

        // Check if Docker is running
        if !isDockerRunning() {
            return nil, fmt.Errorf("Docker not found. Install: https://docker.com/get-started")
        }

        fmt.Println("Docker found ✓")
        fmt.Println("")
        fmt.Println("Starting services...")
        fmt.Println("Run: make start")
        fmt.Println("")

        // Save to config
        cfg.Mode = "local"
        cfg.Save()

        return nil, fmt.Errorf("run 'make start' to start local services, then retry")

    default:
        return nil, fmt.Errorf("invalid choice: %s", choice)
    }
}

// isDockerRunning checks if Docker daemon is accessible
func isDockerRunning() bool {
    cmd := exec.Command("docker", "info")
    return cmd.Run() == nil
}
```

---

### Phase 1.2: Configuration Management (Week 1)

**Goal:** Support both cloud and local configuration

#### Configuration File Format

**File:** `~/.coderisk/config.yaml`

```yaml
# CodeRisk Configuration
# Mode: cloud (default) or local
mode: cloud

# Cloud mode configuration
cloud:
  api_key: sk_live_abc123xyz  # Get from https://coderisk.dev/signup
  endpoint: https://api.coderisk.dev

# Local mode configuration
local:
  neo4j_uri: bolt://localhost:7687
  neo4j_user: neo4j
  neo4j_password: ""  # Leave empty to use NEO4J_PASSWORD env var

  postgres_uri: postgresql://localhost:5432/coderisk
  postgres_user: coderisk
  postgres_password: ""

  redis_uri: redis://localhost:6379

# LLM configuration (optional, only for Phase 2 investigations)
llm:
  provider: openai  # openai or anthropic
  api_key: sk-...   # Your LLM API key (BYOK)
  model: gpt-4o-mini
```

#### Update config package

**File:** `internal/config/config.go`

```go
type Config struct {
    Mode  string       `yaml:"mode"`  // "cloud" or "local"
    Cloud CloudConfig  `yaml:"cloud"`
    Local LocalConfig  `yaml:"local"`
    LLM   LLMConfig    `yaml:"llm"`
}

type CloudConfig struct {
    APIKey   string `yaml:"api_key"`
    Endpoint string `yaml:"endpoint"`
}

type LocalConfig struct {
    Neo4jURI        string `yaml:"neo4j_uri"`
    Neo4jUser       string `yaml:"neo4j_user"`
    Neo4jPassword   string `yaml:"neo4j_password"`
    PostgresURI     string `yaml:"postgres_uri"`
    PostgresUser    string `yaml:"postgres_user"`
    PostgresPassword string `yaml:"postgres_password"`
    RedisURI        string `yaml:"redis_uri"`
}

type LLMConfig struct {
    Provider string `yaml:"provider"`
    APIKey   string `yaml:"api_key"`
    Model    string `yaml:"model"`
}
```

---

### Phase 1.3: CLI Integration (Week 2)

**Goal:** Update all commands to use backend abstraction

#### Update init-local command

**File:** `cmd/crisk/init_local.go`

```go
func runInitLocal(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Load configuration
    cfg, err := config.Load("")
    if err != nil {
        cfg = config.Default()
    }

    // Create backend (cloud or local)
    backend, err := backend.NewBackend(ctx, cfg)
    if err != nil {
        return err
    }
    defer backend.Close()

    // Health check
    if err := backend.HealthCheck(ctx); err != nil {
        return fmt.Errorf("backend health check failed: %w", err)
    }

    // Rest of init-local logic remains the same
    // ...
}
```

#### Update check command

**File:** `cmd/crisk/check.go`

```go
func runCheck(cmd *cobra.Command, args []string) error {
    ctx := context.Background()

    // Load configuration
    cfg, err := config.Load("")
    if err != nil {
        return err
    }

    // Create backend
    backend, err := backend.NewBackend(ctx, cfg)
    if err != nil {
        return err
    }
    defer backend.Close()

    // Query graph for risk metrics
    // Uses backend.Query() instead of direct Neo4j calls
    // ...
}
```

---

### Phase 1.4: Cloud API Server (Week 2)

**Goal:** Implement REST API that wraps Neptune/PostgreSQL/Redis

#### API Endpoints

**File:** `cmd/api/handlers/graph.go`

```go
package handlers

import (
    "encoding/json"
    "net/http"

    "github.com/coderisk/coderisk-go/internal/graph"
)

type GraphHandler struct {
    neo4j *graph.Neo4jBackend  // Replace with Neptune client
}

// POST /v1/graph/nodes
func (h *GraphHandler) CreateNodes(w http.ResponseWriter, r *http.Request) {
    // Authenticate request
    apiKey := extractAPIKey(r)
    if !h.validateAPIKey(apiKey) {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Parse request
    var req struct {
        Nodes []graph.GraphNode `json:"nodes"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    // Create nodes in Neptune
    ids, err := h.neo4j.CreateNodes(req.Nodes)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Return response
    json.NewEncoder(w).Encode(map[string]interface{}{
        "ids": ids,
    })
}

// POST /v1/graph/edges
func (h *GraphHandler) CreateEdges(w http.ResponseWriter, r *http.Request) {
    // Similar to CreateNodes...
}

// POST /v1/graph/query
func (h *GraphHandler) Query(w http.ResponseWriter, r *http.Request) {
    // Authenticate
    apiKey := extractAPIKey(r)
    if !h.validateAPIKey(apiKey) {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Parse query
    var req struct {
        Query string `json:"query"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    // Execute query
    result, err := h.neo4j.Query(req.Query)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Return results
    json.NewEncoder(w).Encode(map[string]interface{}{
        "data": result,
    })
}
```

---

### Phase 1.5: Package Distribution (Week 3)

**Goal:** Publish to Homebrew, pip, uv

#### Homebrew Formula

**File:** `Formula/crisk.rb` (in separate tap repo)

```ruby
class Crisk < Formula
  desc "Lightning-fast risk assessment for code changes"
  homepage "https://coderisk.dev"
  url "https://github.com/coderisk/crisk/releases/download/v0.1.0/crisk-darwin-amd64.tar.gz"
  sha256 "..."
  version "0.1.0"

  def install
    bin.install "crisk"
  end

  def caveats
    <<~EOS
      🎉 CodeRisk installed successfully!

      Choose your mode:

      1. Cloud Mode (recommended):
         Get API key: https://coderisk.dev/signup
         Run: crisk init-local

      2. Local Mode (advanced):
         Install Docker: https://docker.com
         Start services: make start
         Run: crisk init-local
    EOS
  end

  test do
    system "#{bin}/crisk", "--version"
  end
end
```

**Installation:**
```bash
brew tap coderisk/tap
brew install crisk
```

---

## User Experience Flow

### First-Time Cloud User

```bash
# Install
brew install coderisk/tap/crisk

# Run for first time
crisk init-local

# Output:
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Welcome to CodeRisk!
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#
# Choose how you want to run CodeRisk:
# 1. Cloud Mode (recommended)
# 2. Local Mode (advanced)
#
# Choice (1/2): 1
#
# Enter your API key: sk_live_abc123
#
# ✅ Connected to CodeRisk Cloud
# 🔄 Syncing repository...
# ✅ Repository indexed (421 files, 2,563 functions)
#
# Ready! Run: crisk check <file>

# Use immediately
git add src/auth/session.ts
crisk check src/auth/session.ts

# Output:
# ⚠️  MEDIUM risk: src/auth/session.ts
# Evidence:
#   - High coupling (12 imports, 8 dependents)
#   - Co-changes with auth/route.ts (85% frequency)
#   - Low test coverage (0.2 ratio)
```

### First-Time Local User

```bash
# Install
brew install coderisk/tap/crisk

# Run for first time
crisk init-local

# Output:
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Welcome to CodeRisk!
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#
# Choose how you want to run CodeRisk:
# 1. Cloud Mode (recommended)
# 2. Local Mode (advanced)
#
# Choice (1/2): 2
#
# Checking Docker...
# Docker found ✓
#
# Starting services...
# Run: make start

# User runs make start
make start

# Output:
# 🐳 Starting CodeRisk services...
# ⏳ Waiting for services to initialize...
# ✅ Services started!
#
# 📋 Service URLs:
#    Neo4j: http://localhost:7474
#    PostgreSQL: localhost:5432
#    Redis: localhost:6379

# Now run init-local again
crisk init-local

# Output:
# ✅ Connected to local services
# 🔄 Parsing repository...
# ✅ Repository indexed (421 files, 2,563 functions)
#
# Ready! Run: crisk check <file>
```

---

## Testing Strategy

### Unit Tests

```go
// internal/backend/cloud_test.go
func TestCloudBackend_CreateNodes(t *testing.T) {
    // Mock HTTP server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify auth header
        assert.Equal(t, "Bearer sk_test_123", r.Header.Get("Authorization"))

        // Return mock response
        json.NewEncoder(w).Encode(map[string]interface{}{
            "ids": []string{"node1", "node2"},
        })
    }))
    defer server.Close()

    // Create cloud backend
    backend, err := NewCloudBackend("sk_test_123", server.URL)
    require.NoError(t, err)

    // Test create nodes
    nodes := []GraphNode{
        {Label: "File", ID: "file:test.ts", Properties: map[string]interface{}{"name": "test.ts"}},
    }

    ids, err := backend.CreateNodes(context.Background(), nodes)
    require.NoError(t, err)
    assert.Len(t, ids, 2)
}

// internal/backend/local_test.go
func TestLocalBackend_CreateNodes(t *testing.T) {
    // Start test Neo4j container
    neo4j := startTestNeo4j(t)
    defer neo4j.Close()

    // Create local backend
    backend, err := NewLocalBackend(context.Background(), neo4j.URI, "neo4j", "test")
    require.NoError(t, err)
    defer backend.Close()

    // Test create nodes
    nodes := []GraphNode{
        {Label: "File", ID: "file:test.ts", Properties: map[string]interface{}{"name": "test.ts"}},
    }

    ids, err := backend.CreateNodes(context.Background(), nodes)
    require.NoError(t, err)
    assert.Len(t, ids, 1)
}
```

### Integration Tests

```bash
# Test cloud mode
export CODERISK_MODE=cloud
export CODERISK_API_KEY=sk_test_123
crisk init-local

# Test local mode
export CODERISK_MODE=local
make start
crisk init-local

# Verify both produce same results
diff <(crisk check file.ts --mode=cloud) <(crisk check file.ts --mode=local)
```

---

## Cost Comparison

### Development Costs

| Task | Estimated Time | Developer Cost (@$150/hr) |
|------|----------------|---------------------------|
| Backend abstraction | 20 hours | $3,000 |
| Cloud backend impl | 16 hours | $2,400 |
| Config management | 8 hours | $1,200 |
| CLI integration | 12 hours | $1,800 |
| Cloud API server | 24 hours | $3,600 |
| Package distribution | 8 hours | $1,200 |
| Testing | 16 hours | $2,400 |
| **Total** | **104 hours** | **$15,600** |

### Operating Costs (Month 1)

| Mode | Users | Infrastructure Cost | Per-User Cost |
|------|-------|---------------------|---------------|
| Cloud | 100 | $955 | $9.55 |
| Cloud | 1,000 | $2,300 | $2.30 |
| Local | N/A | $0 | $0 (user hardware) |

### Revenue Potential (Month 1, assuming 70% cloud adoption)

| Tier | Users | Price | Revenue | Cost | Profit |
|------|-------|-------|---------|------|--------|
| Free | 300 | $0 | $0 | $686 | -$686 |
| Pro | 500 | $9 | $4,500 | $1,150 | $3,350 |
| Team | 200 | $29 | $5,800 | $460 | $5,340 |
| **Total** | **1,000** | - | **$10,300** | **$2,296** | **$8,004** |

**ROI:** $8,004/month profit - $15,600 dev cost = break-even in 2 months

---

## Success Metrics

### Phase 1.1 (Week 1)
- ✅ Backend interface defined
- ✅ Cloud backend implemented
- ✅ Local backend refactored
- ✅ Unit tests passing

### Phase 1.2 (Week 1-2)
- ✅ Config management updated
- ✅ Mode selection working
- ✅ Health checks functional

### Phase 1.3 (Week 2)
- ✅ All CLI commands use backend abstraction
- ✅ init-local works in both modes
- ✅ check works in both modes

### Phase 1.4 (Week 2-3)
- ✅ Cloud API server deployed
- ✅ API authentication working
- ✅ Neptune integration complete

### Phase 1.5 (Week 3)
- ✅ Homebrew formula published
- ✅ Installation works on macOS/Linux
- ✅ First-run UX validated

---

## Next Steps After Phase 1

1. **Phase 2:** Free tier + API key management
2. **Phase 3:** Billing integration (Stripe)
3. **Phase 4:** Settings portal (web UI)
4. **Phase 5:** Team features (shared graphs)

---

## Related Documents

- [cloud_deployment.md](../01-architecture/cloud_deployment.md) - Full cloud architecture
- [graph_ontology.md](../01-architecture/graph_ontology.md) - Graph schema
- [agentic_design.md](../01-architecture/agentic_design.md) - Two-phase investigation

---

**Status:** Ready for implementation
**Owner:** Engineering team
**Review:** Product team
**Approval:** Required before starting Week 1
