# Local Deployment Integration Guide

**Purpose:** Comprehensive implementation guide for running CodeRisk locally with Docker Compose, Neo4j, Redis, and PostgreSQL - designed for AI agent implementation

**Prerequisites:** Docker Desktop (macOS/Windows) or Docker Engine + Docker Compose (Linux), 8GB+ RAM, 50GB+ disk

**Last Updated:** October 2, 2025

**Target Users:** Individual developers, small teams (1-5 people), air-gapped environments

**AI Agent Implementation Note:** This document is intended to be read by AI agents implementing the local deployment alongside [DEVELOPMENT_WORKFLOW.md](../../DEVELOPMENT_WORKFLOW.md) for guardrails and best practices.

---

## Architecture Context & References

> **📐 12-Factor Principles Applied:**
> - [Factor 1: Natural Language to Tool Calls](../../12-factor-agents-main/content/factor-01-natural-language-to-tool-calls.md) - CLI uses same API as cloud
> - [Factor 5: Unify Execution State](../../12-factor-agents-main/content/factor-05-unify-execution-state.md) - Separate graph (persistent) from cache (ephemeral)
> - [Factor 8: Own Your Control Flow](../../12-factor-agents-main/content/factor-08-own-your-control-flow.md) - Two-phase investigation (deterministic + LLM)
> - [Factor 12: Stateless Reducer](../../12-factor-agents-main/content/factor-12-stateless-reducer.md) - API service is stateless, scales horizontally

**Required Reading Before Implementation:**

1. **[spec.md](../../spec.md)** - System requirements, scope, constraints, and business logic
   - Section 1.5: Scope (what's in/out for v1.0)
   - Section 4: Non-functional requirements (performance, security)
   - Section 5: Technical requirements (architecture, tech stack)

2. **[DEVELOPMENT_WORKFLOW.md](../../DEVELOPMENT_WORKFLOW.md)** - Implementation guardrails and safety protocols
   - Security constraints (no hardcoded credentials, input validation)
   - Code quality standards (error handling, logging, testing)
   - Quality gates (build, test, lint checks)

3. **[01-architecture/agentic_design.md](../../01-architecture/agentic_design.md)** - Two-phase investigation flow
   - Phase 1: Baseline assessment (<500ms, no LLM)
   - Phase 2: LLM-guided investigation (3-5s, selective metrics)
   - Working memory and hop limits

4. **[01-architecture/graph_ontology.md](../../01-architecture/graph_ontology.md)** - Graph schema and data model
   - Three layers: Structure, Temporal, Incidents
   - Persistent vs ephemeral data (what goes in graph vs Redis)
   - Query performance targets

5. **[01-architecture/risk_assessment_methodology.md](../../01-architecture/risk_assessment_methodology.md)** - Metric calculations
   - Tier 1 metrics (coupling, co-change, test ratio)
   - Tier 2 metrics (ownership churn, incident similarity)
   - Threshold logic and validation framework

---

## Overview

This guide provides end-to-end implementation instructions for local deployment of CodeRisk using Docker Compose. This deployment mode is ideal for:

- **Individual developers** - No monthly fees, full privacy
- **Small teams (1-5 people)** - Shared server deployment
- **Air-gapped environments** - Defense, finance, regulated industries
- **Privacy-sensitive code** - Proprietary algorithms, confidential codebases
- **Cost-conscious users** - $0/month infrastructure (only LLM API costs if using Phase 2)

**Trade-offs vs Cloud:**
- ✅ $0/month infrastructure cost
- ✅ Full data privacy (code never leaves your network)
- ✅ Works offline (Phase 1 baseline checks)
- ❌ Manual setup (10-15 minutes)
- ❌ Manual graph updates (no automatic webhooks)
- ❌ Hardware limits (~50K files max)

---

## Architecture Overview

```
┌────────────────────────────────────────────────────────────┐
│  YOUR MACHINE (localhost)                                  │
│                                                            │
│  ┌──────────────────────────────────────────────┐         │
│  │  crisk CLI (Go binary)                       │         │
│  │  • git diff → http://localhost:8080          │         │
│  │  • Two-phase investigation (spec.md §5.1)    │         │
│  └──────────────────────────────────────────────┘         │
│                        ↓                                   │
│  ┌──────────────────────────────────────────────┐         │
│  │  Docker Compose Stack                        │         │
│  │                                              │         │
│  │  ┌─────────────────────────────────┐        │         │
│  │  │ API Service (Go) :8080          │        │         │
│  │  │ • Phase 1: Baseline (<500ms)    │        │         │
│  │  │ • Phase 2: LLM investigation    │        │         │
│  │  │ • Metric calculation            │        │         │
│  │  │ • OpenAI/Anthropic calls (BYOK) │        │         │
│  │  └─────────────────────────────────┘        │         │
│  │         ↓           ↓           ↓            │         │
│  │  ┌────────┐  ┌────────┐  ┌────────┐         │         │
│  │  │ Neo4j  │  │ Redis  │  │Postgres│         │         │
│  │  │ :7474  │  │ :6379  │  │ :5432  │         │         │
│  │  │ Graph  │  │ Cache  │  │Metadata│         │         │
│  │  │(Layer  │  │(15-min │  │(Metric │         │         │
│  │  │ 1,2,3) │  │  TTL)  │  │  FP)   │         │         │
│  │  └────────┘  └────────┘  └────────┘         │         │
│  └──────────────────────────────────────────────┘         │
└────────────────────────────────────────────────────────────┘
```

**Component Details (see [spec.md](../../spec.md) §5.4):**

- **Neo4j Graph Database** (local equivalent of Neptune Serverless)
  - Stores 3-layer ontology: Structure (tree-sitter AST), Temporal (git history), Incidents
  - See [graph_ontology.md](../../01-architecture/graph_ontology.md) for full schema
  - Local deployment uses Neo4j Community Edition (Cypher syntax)

- **Redis Cache** (ephemeral state, Factor 5)
  - Baseline metric results (15-min TTL)
  - Investigation traces (session cache)
  - See [agentic_design.md](../../01-architecture/agentic_design.md) §4.1 for caching strategy

- **PostgreSQL** (metadata and metric validation)
  - Metric validation tracking (FP rates)
  - User feedback for metric auto-disablement
  - See [risk_assessment_methodology.md](../../01-architecture/risk_assessment_methodology.md) §5 for schema

- **API Service** (stateless, Factor 12)
  - Two-phase investigation orchestration
  - LLM client (user's BYOK API key)
  - Graph query execution

**Storage locations:**
- Neo4j data: `./volumes/neo4j_data` (~1-50GB depending on repo size)
- Redis cache: `./volumes/redis_data` (~2-8GB)
- Postgres metadata: `./volumes/postgres_data` (~1-5GB)

---

## System Requirements

### Minimum Requirements by Codebase Size

| Codebase | Files | Graph Size | RAM | Disk | CPU | Boot Time | Check Latency |
|----------|-------|------------|-----|------|-----|-----------|---------------|
| **Small** | 500 | 8K nodes<br>50K edges | 4GB | 10GB | 2 cores | 30s | 200ms (P1)<br>3s (P2) |
| **Medium** | 5K | 80K nodes<br>600K edges | 8GB | 50GB | 4 cores | 60s | 300ms (P1)<br>4s (P2) |
| **Large** | 50K | 800K nodes<br>6M edges | 16GB | 200GB | 8 cores | 120s | 500ms (P1)<br>6s (P2) |

**Performance targets from [spec.md](../../spec.md) §2.4:**
- Phase 1 (baseline): <500ms (NFR-1)
- Phase 2 (LLM investigation): 3-5s (NFR-1)
- Cache hit rate: >90% (NFR-7)

**Recommended hardware:**
- **MacBook Pro M1/M2** (16GB RAM) - Good for medium repos
- **Linux workstation** (16GB+ RAM) - Good for medium-large repos
- **Shared server** (32GB RAM) - Good for team deployments
- **High-end server** (64GB RAM, NVMe SSD) - Required for large repos (50K files)

### Software Requirements

**Required:**
- Docker Engine 20.10+ or Docker Desktop 4.0+
- Docker Compose 2.0+
- Git 2.30+
- Go 1.21+ (for building CLI from source)

**Optional (for Phase 2 investigations):**
- OpenAI API key or Anthropic API key (for LLM-guided investigations)
- Internet connection (only for LLM calls, not for graph storage)

---

## Installation

### Step 1: Install Docker

**macOS:**
```bash
# Option 1: Docker Desktop (GUI)
# Download from https://www.docker.com/products/docker-desktop/

# Option 2: Homebrew
brew install --cask docker

# Verify
docker --version
docker-compose --version
```

**Linux (Ubuntu/Debian):**
```bash
# Install Docker Engine
sudo apt-get update
sudo apt-get install -y docker.io docker-compose-plugin

# Add your user to docker group (avoid sudo)
sudo usermod -aG docker $USER
newgrp docker

# Verify
docker --version
docker compose version
```

**Windows:**
```powershell
# Install Docker Desktop
# Download from https://www.docker.com/products/docker-desktop/

# Verify in PowerShell
docker --version
docker-compose --version
```

### Step 2: Clone CodeRisk Repository

```bash
# Clone the repository
git clone https://github.com/coderisk/coderisk-go.git
cd coderisk-go

# Verify you're on the main branch
git branch

# Note: Requires Go 1.23+ for dependencies (Anthropic SDK requirement)
# Docker handles this automatically via golang:1.23-alpine base image
```

### Step 3: Create Directory Structure

```bash
# Create volumes directory for persistent storage
# (Security: Use chmod to prevent unauthorized access)
mkdir -p volumes/{neo4j_data,neo4j_logs,postgres_data,redis_data}

# Set permissions (Linux/macOS)
chmod -R 755 volumes/

# Verify structure
tree volumes/
```

**Error Handling:**
- If `tree` not installed: `ls -R volumes/`
- If permission denied: `sudo chown -R $USER:$USER volumes/`

### Step 4: Configure Environment Variables

**CRITICAL SECURITY NOTE ([DEVELOPMENT_WORKFLOW.md](../../DEVELOPMENT_WORKFLOW.md) §3.3):**
- NEVER commit `.env` file to git
- Change default passwords in production
- Use strong passwords (16+ characters, mixed case, symbols)

```bash
# Create .env file for local deployment
# (Follows spec.md §5.2 BYOK model)
cat > .env <<'EOF'
# ========================================
# Deployment Configuration
# ========================================
MODE=local

# ========================================
# Neo4j Configuration (Graph Database)
# Reference: graph_ontology.md - Three-layer graph structure
# ========================================
NEO4J_AUTH=neo4j/CHANGE_THIS_PASSWORD_IN_PRODUCTION_123
NEO4J_MAX_HEAP=2G
NEO4J_PAGECACHE=1G

# Memory tuning by repo size (from system requirements table):
# Small (<500 files):  NEO4J_MAX_HEAP=2G, NEO4J_PAGECACHE=1G
# Medium (<5K files):  NEO4J_MAX_HEAP=4G, NEO4J_PAGECACHE=2G
# Large (<50K files):  NEO4J_MAX_HEAP=8G, NEO4J_PAGECACHE=4G

# ========================================
# PostgreSQL Configuration (Metadata)
# Reference: risk_assessment_methodology.md §5.1 - Validation schema
# ========================================
POSTGRES_DB=coderisk
POSTGRES_USER=coderisk
POSTGRES_PASSWORD=CHANGE_THIS_PASSWORD_IN_PRODUCTION_123

# ========================================
# Redis Configuration (Ephemeral Cache)
# Reference: agentic_design.md §4.1 - Cache strategy
# ========================================
REDIS_MAXMEMORY=2gb
REDIS_MAXMEMORY_POLICY=allkeys-lru

# ========================================
# API Service Configuration
# ========================================
API_PORT=8080
LOG_LEVEL=info

# ========================================
# LLM Configuration (BYOK - Bring Your Own Key)
# Reference: spec.md §1.3 - BYOK model
# ========================================
# IMPORTANT: Phase 2 requires user's LLM API key
# Leave commented if you only want Phase 1 (baseline checks)

# Option 1: OpenAI
# OPENAI_API_KEY=sk-proj-...

# Option 2: Anthropic
# ANTHROPIC_API_KEY=sk-ant-...

# Enable Phase 2 investigations (requires API key above)
PHASE2_ENABLED=false

# ========================================
# Performance Tuning
# Reference: spec.md §2.4 - Performance targets
# ========================================
# Max graph traversal hops (spec.md §6.2 constraint C-6)
MAX_HOPS=5

# False positive rate threshold (spec.md §6.2 constraint C-10)
FP_THRESHOLD=0.03

# Investigation timeout (Phase 2)
INVESTIGATION_TIMEOUT_SECONDS=30
EOF

# Secure the .env file (prevent unauthorized reading)
chmod 600 .env

# Add to .gitignore if not already there
echo ".env" >> .gitignore
```

**Validation:**
```bash
# Verify .env file was created correctly
cat .env | grep -v "^#" | grep -v "^$"

# Should show:
# MODE=local
# NEO4J_AUTH=neo4j/...
# POSTGRES_PASSWORD=...
# etc.
```

### Step 5: Create Docker Compose Configuration

**⚠️ PORT CONFLICT TROUBLESHOOTING:**

If you have existing services running on the default ports, add these variables to your `.env` file:

```bash
# Port Mappings (adjust if ports are in use)
NEO4J_HTTP_PORT=7475      # Default: 7474
NEO4J_BOLT_PORT=7688      # Default: 7687
POSTGRES_PORT_EXTERNAL=5433  # Default: 5432
REDIS_PORT_EXTERNAL=6380     # Default: 6379
```

Then use `${NEO4J_HTTP_PORT}` instead of hardcoded ports in docker-compose.yml. See [IMPLEMENTATION_LOG.md](../IMPLEMENTATION_LOG.md) for details.

**Create `docker-compose.yml` (if not exists):**

```bash
cat > docker-compose.yml <<'EOF'
version: '3.8'

# ========================================
# CodeRisk Local Deployment Stack
# Reference: spec.md §5.6 - Deployment models
# ========================================

services:
  # ========================================
  # Neo4j Graph Database
  # Reference: graph_ontology.md - Three-layer ontology
  # ========================================
  neo4j:
    image: neo4j:5.15-community
    container_name: coderisk-neo4j
    ports:
      - "7474:7474"  # HTTP (browser UI)
      - "7687:7687"  # Bolt (Cypher queries)
    volumes:
      - ./volumes/neo4j_data:/data
      - ./volumes/neo4j_logs:/logs
    environment:
      - NEO4J_AUTH=${NEO4J_AUTH}
      - NEO4J_dbms_memory_heap_max__size=${NEO4J_MAX_HEAP}
      - NEO4J_dbms_memory_pagecache_size=${NEO4J_PAGECACHE}
      # Performance optimizations
      - NEO4J_dbms_query__cache__size=1000
      - NEO4J_dbms_tx__log__rotation__retention__policy=2 days
    healthcheck:
      test: ["CMD-SHELL", "cypher-shell -u neo4j -p ${NEO4J_AUTH#neo4j/} 'RETURN 1'"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  # ========================================
  # PostgreSQL Metadata Database
  # Reference: risk_assessment_methodology.md §5.2 - Validation schema
  # ========================================
  postgres:
    image: postgres:16-alpine
    container_name: coderisk-postgres
    ports:
      - "5432:5432"
    volumes:
      - ./volumes/postgres_data:/var/lib/postgresql/data
      # Initialize schema on first run
      - ./scripts/init_postgres.sql:/docker-entrypoint-initdb.d/01_schema.sql
    environment:
      - POSTGRES_DB=${POSTGRES_DB}
      - POSTGRES_USER=${POSTGRES_USER}
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB}"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  # ========================================
  # Redis Cache (Ephemeral State)
  # Reference: agentic_design.md §4.1 - Caching strategy
  # ========================================
  redis:
    image: redis:7-alpine
    container_name: coderisk-redis
    ports:
      - "6379:6379"
    volumes:
      - ./volumes/redis_data:/data
    command: >
      redis-server
      --maxmemory ${REDIS_MAXMEMORY}
      --maxmemory-policy ${REDIS_MAXMEMORY_POLICY}
      --appendonly yes
      --save 900 1
      --save 300 10
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  # ========================================
  # API Service (Go Application)
  # Reference: agentic_design.md §2 - Two-phase investigation
  # ========================================
  api:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: coderisk-api
    ports:
      - "${API_PORT}:8080"
    environment:
      # Database connections
      - NEO4J_URI=bolt://neo4j:7687
      - NEO4J_USER=neo4j
      - NEO4J_PASSWORD=${NEO4J_AUTH#neo4j/}
      - POSTGRES_HOST=postgres
      - POSTGRES_PORT=5432
      - POSTGRES_DB=${POSTGRES_DB}
      - POSTGRES_USER=${POSTGRES_USER}
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      # LLM configuration (BYOK)
      - OPENAI_API_KEY=${OPENAI_API_KEY:-}
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY:-}
      - PHASE2_ENABLED=${PHASE2_ENABLED}
      # Performance tuning
      - MAX_HOPS=${MAX_HOPS}
      - FP_THRESHOLD=${FP_THRESHOLD}
      - INVESTIGATION_TIMEOUT=${INVESTIGATION_TIMEOUT_SECONDS}
      - LOG_LEVEL=${LOG_LEVEL}
    depends_on:
      neo4j:
        condition: service_healthy
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

networks:
  default:
    name: coderisk-network
EOF
```

**Create PostgreSQL initialization script:**

```bash
mkdir -p scripts
cat > scripts/init_postgres.sql <<'EOF'
-- ========================================
-- PostgreSQL Schema Initialization
-- Reference: risk_assessment_methodology.md §5.2
-- ========================================

-- Metric validation tracking (user feedback)
CREATE TABLE IF NOT EXISTS metric_validations (
    id SERIAL PRIMARY KEY,
    metric_name VARCHAR(50) NOT NULL,  -- "coupling", "co_change", etc.
    file_path VARCHAR(500) NOT NULL,
    metric_value JSONB NOT NULL,       -- Full metric output
    user_feedback VARCHAR(20),         -- "true_positive", "false_positive", null
    feedback_reason TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Aggregate statistics per metric (auto-calculated)
CREATE TABLE IF NOT EXISTS metric_stats (
    metric_name VARCHAR(50) PRIMARY KEY,
    total_uses INT DEFAULT 0,
    false_positives INT DEFAULT 0,
    true_positives INT DEFAULT 0,
    fp_rate FLOAT GENERATED ALWAYS AS (
        CASE WHEN total_uses > 0
        THEN false_positives::FLOAT / total_uses
        ELSE 0.0 END
    ) STORED,
    is_enabled BOOLEAN DEFAULT TRUE,
    last_updated TIMESTAMP DEFAULT NOW()
);

-- Repository metadata (for multi-repo support)
CREATE TABLE IF NOT EXISTS repositories (
    id SERIAL PRIMARY KEY,
    repo_path VARCHAR(500) UNIQUE NOT NULL,
    last_sync TIMESTAMP,
    graph_node_count INT DEFAULT 0,
    graph_edge_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Investigation traces (for debugging and analytics)
CREATE TABLE IF NOT EXISTS investigation_traces (
    id SERIAL PRIMARY KEY,
    file_path VARCHAR(500) NOT NULL,
    phase INT NOT NULL,  -- 1 or 2
    hop_count INT,
    metrics_calculated JSONB,
    llm_decisions JSONB,
    final_risk_level VARCHAR(20),
    duration_ms INT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_metric_validations_name ON metric_validations(metric_name);
CREATE INDEX IF NOT EXISTS idx_metric_validations_file ON metric_validations(file_path);
CREATE INDEX IF NOT EXISTS idx_investigation_traces_file ON investigation_traces(file_path);
CREATE INDEX IF NOT EXISTS idx_investigation_traces_created ON investigation_traces(created_at DESC);

-- Initialize default metric stats (from risk_assessment_methodology.md §2)
INSERT INTO metric_stats (metric_name, is_enabled) VALUES
    ('coupling', TRUE),
    ('co_change', TRUE),
    ('test_ratio', TRUE),
    ('ownership_churn', TRUE),
    ('incident_similarity', TRUE)
ON CONFLICT (metric_name) DO NOTHING;
EOF
```

**Create Dockerfile for API service:**

```bash
cat > Dockerfile <<'EOF'
# ========================================
# CodeRisk API Service Dockerfile
# Reference: spec.md §5.2 - Technology stack
# ========================================

FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy Go modules first (caching optimization)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/crisk-api ./cmd/api

# ========================================
# Runtime stage (minimal image)
# ========================================
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/crisk-api .

# Expose API port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

# Run API service
CMD ["./crisk-api"]
EOF
```

### Step 6: Start Services

```bash
# Start all services in background
# (This will build the API service Docker image first)
docker compose up -d --build

# View logs (follow mode)
docker compose logs -f

# Check service health
docker compose ps

# Expected output:
# NAME                STATUS              PORTS
# coderisk-api-1      Up (healthy)        0.0.0.0:8080->8080/tcp
# coderisk-neo4j-1    Up (healthy)        0.0.0.0:7474->7474/tcp, 7687->7687/tcp
# coderisk-postgres-1 Up (healthy)        0.0.0.0:5432->5432/tcp
# coderisk-redis-1    Up (healthy)        0.0.0.0:6379->6379/tcp
```

**Troubleshooting startup (error handling):**

```bash
# If services fail to start, check logs individually
docker compose logs neo4j
docker compose logs postgres
docker compose logs redis
docker compose logs api

# Common issues and solutions:
# 1. Port conflicts (7474, 7687, 5432, 6379, 8080 already in use)
#    Check what's using the port:
lsof -i :7474
#    Solution: Stop conflicting service or change ports in docker-compose.yml

# 2. Insufficient memory
#    Check Docker memory limit:
docker info | grep Memory
#    Solution: Increase Docker Desktop memory limit (Preferences > Resources)
#    Minimum: 4GB for small repos, 8GB for medium, 16GB for large

# 3. Permission errors (Linux)
#    Fix ownership:
sudo chown -R $USER:$USER volumes/

# 4. Neo4j fails to start (heap size too large)
#    Reduce NEO4J_MAX_HEAP in .env:
#    NEO4J_MAX_HEAP=1G  (for small repos)

# 5. API service health check fails
#    Check if API is listening:
docker compose exec api netstat -tulpn | grep 8080
#    Check API logs for errors:
docker compose logs api | tail -50
```

### Step 7: Verify Installation

```bash
# Test Neo4j connection (browser UI)
curl http://localhost:7474
# Expected: HTML response with Neo4j Browser

# Test API health endpoint
curl http://localhost:8080/health
# Expected: {"status":"healthy","version":"0.1.0","services":{"neo4j":"ok","postgres":"ok","redis":"ok","llm":"disabled"}}

# Test Neo4j Cypher query (verify graph is ready)
# Password is from .env: NEO4J_AUTH=neo4j/PASSWORD
docker compose exec neo4j cypher-shell -u neo4j -p "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" "RETURN 'Graph ready' AS status"
# Expected: │ "Graph ready" │

# Test PostgreSQL connection
docker compose exec postgres psql -U coderisk -d coderisk -c "SELECT count(*) FROM metric_stats;"
# Expected: │ count │
#           │     5 │  (5 default metrics initialized)

# Test Redis connection
docker compose exec redis redis-cli ping
# Expected: PONG
```

**Validation checklist:**
- [ ] All 4 containers healthy (green in `docker compose ps`)
- [ ] Neo4j browser accessible at http://localhost:7474
- [ ] API health endpoint returns `{"status":"healthy"}`
- [ ] PostgreSQL has 5 initialized metrics
- [ ] Redis responds to PING

---

## Configuration

### Neo4j Graph Database

**Access Neo4j Browser:**
```bash
# Open in browser
open http://localhost:7474

# Login credentials (from .env)
# Username: neo4j
# Password: CHANGE_THIS_PASSWORD_IN_PRODUCTION_123
```

**Configure memory for your repo size:**

Edit `.env` file:
```bash
# Small repos (<500 files)
NEO4J_MAX_HEAP=2G
NEO4J_PAGECACHE=1G

# Medium repos (<5K files)
NEO4J_MAX_HEAP=4G
NEO4J_PAGECACHE=2G

# Large repos (<50K files)
NEO4J_MAX_HEAP=8G
NEO4J_PAGECACHE=4G
```

Apply changes:
```bash
docker compose down
docker compose up -d
```

**Create required indexes (performance optimization):**

```cypher
// Run in Neo4j Browser (http://localhost:7474)
// Reference: graph_ontology.md - Query performance targets

// Layer 1: Structure indexes
CREATE INDEX file_path IF NOT EXISTS FOR (f:File) ON (f.path);
CREATE INDEX function_name IF NOT EXISTS FOR (fn:Function) ON (fn.name);
CREATE INDEX class_name IF NOT EXISTS FOR (c:Class) ON (c.name);

// Layer 2: Temporal indexes
CREATE INDEX commit_sha IF NOT EXISTS FOR (c:Commit) ON (c.sha);
CREATE INDEX developer_email IF NOT EXISTS FOR (d:Developer) ON (d.email);

// Layer 3: Incident indexes
CREATE INDEX incident_id IF NOT EXISTS FOR (i:Incident) ON (i.id);

// Verify indexes created
SHOW INDEXES;
```

### Redis Cache Configuration

**Monitor cache usage:**
```bash
# Connect to Redis CLI
docker compose exec redis redis-cli

# Check memory usage
INFO memory

# Check cache hit rate
INFO stats

# View cached keys (by pattern)
KEYS baseline:*
KEYS investigation:*

# Check specific cache entry
GET "coupling:repo123:src/auth.py"

# Clear all cache (if needed)
FLUSHALL
```

**Adjust cache size:**

Edit `.env` file:
```bash
# Small repos: 1gb
# Medium repos: 2gb  (default)
# Large repos: 4-8gb
REDIS_MAXMEMORY=2gb
```

Apply changes:
```bash
docker compose restart redis
```

### PostgreSQL Metadata

**Connect to database:**
```bash
# PostgreSQL CLI
docker compose exec postgres psql -U coderisk -d coderisk

# View tables
\dt

# Expected tables:
# - metric_validations
# - metric_stats
# - repositories
# - investigation_traces

# Query metric stats
SELECT metric_name, total_uses, fp_rate, is_enabled FROM metric_stats;

# Exit
\q
```

**Backup metadata:**
```bash
# Export database
docker compose exec postgres pg_dump -U coderisk coderisk > backup.sql

# Restore database
docker compose exec -T postgres psql -U coderisk coderisk < backup.sql
```

### OpenAI/Anthropic API Key (Phase 2)

**Enable Phase 2 investigations:**

1. Edit `.env` file:
```bash
nano .env

# Add your API key:
OPENAI_API_KEY=sk-proj-YOUR_KEY_HERE

# Or for Anthropic:
# ANTHROPIC_API_KEY=sk-ant-YOUR_KEY_HERE

# Enable Phase 2
PHASE2_ENABLED=true
```

2. Restart API service:
```bash
docker compose restart api
```

3. Verify Phase 2 is enabled:
```bash
curl http://localhost:8080/health | jq .features.phase2
# Expected: true
```

**Security note ([DEVELOPMENT_WORKFLOW.md](../../DEVELOPMENT_WORKFLOW.md) §3.3):**
- API keys are loaded from environment variables (never hardcoded)
- Never commit `.env` file to git
- Use `.env.example` for templates

---

## Usage

### Build CLI Tool

```bash
# Navigate to project root
cd /path/to/coderisk-go

# Build CLI binary
go build -o bin/crisk ./cmd/crisk

# Verify build
./bin/crisk --version

# Add to PATH (optional, for global access)
export PATH=$PATH:$(pwd)/bin
echo 'export PATH=$PATH:'$(pwd)'/bin' >> ~/.bashrc  # or ~/.zshrc
```

### Initialize Your Repository

**IMPORTANT: Read before running init**
- Reference: [agentic_design.md](../../01-architecture/agentic_design.md) §9 - Graph construction pipeline
- This builds the full 3-layer graph in Neo4j (2-5 min for 5K files)

```bash
# Configure CLI to use local deployment
export CRISK_API_URL=http://localhost:8080

# Navigate to your repository
cd /path/to/your/repo

# Initialize (builds initial graph)
# Phase 1: Tree-sitter AST parsing
# Phase 2: Git history extraction (90-day window)
# Phase 3: Co-change pattern calculation
crisk init

# Progress output (with error handling):
# Parsing files: 1234/5000 (24%)
# Extracting commits: 456/1000 (45%)
# Building graph: inserting nodes...
# Calculating co-change patterns...
# ✓ Initialization complete (5m 23s)
```

**Monitor initialization progress:**
```bash
# In another terminal, watch Neo4j graph size
watch -n 5 'curl -s http://localhost:8080/stats | jq'

# Expected output (updates every 5 seconds):
# {
#   "nodes": 82450,
#   "edges": 615234,
#   "repos": 1,
#   "storage_mb": 4823,
#   "layer_counts": {
#     "structure": 65000,
#     "temporal": 15000,
#     "incidents": 2450
#   }
# }
```

**Verification queries (Neo4j Browser):**
```cypher
// Count nodes by layer
MATCH (n)
RETURN labels(n)[0] as layer, count(n) as count
ORDER BY count DESC;

// Verify Layer 1: Structure (tree-sitter)
MATCH (f:File)
RETURN count(f) as file_count, sum(f.loc) as total_loc;

// Verify Layer 2: Temporal (git history)
MATCH (c:Commit)
RETURN count(c) as commit_count, min(c.timestamp) as oldest, max(c.timestamp) as newest;

// Verify Layer 2: Co-change relationships
MATCH ()-[r:CO_CHANGED]->()
WHERE r.frequency > 0.7
RETURN count(r) as high_coupling_pairs;

// Verify Layer 3: Incidents (if linked)
MATCH (i:Incident)
RETURN count(i) as incident_count;
```

### Run Risk Checks

**Phase 1: Baseline Assessment (no LLM needed)**

Reference: [risk_assessment_methodology.md](../../01-architecture/risk_assessment_methodology.md) §2

```bash
# Make some changes
echo "console.log('test');" >> src/index.js
git add src/index.js

# Check risk (Phase 1 baseline - fast, no LLM)
crisk check

# Output:
# Analyzing 1 changed file(s)...
#
# src/index.js:
#   Risk Level: HIGH
#   Confidence: 0.85
#
#   Evidence (Tier 1 Metrics):
#   • Coupling: 12 files import this module (HIGH)
#     └─ Threshold: >10 (risk_assessment_methodology.md §2.1)
#   • Co-change: 0.75 frequency with src/app.js (HIGH)
#     └─ Threshold: >0.7 (risk_assessment_methodology.md §2.2)
#   • Test ratio: 0.3 (auth_test.js is 30% the size) (MEDIUM)
#     └─ Threshold: <0.3 (risk_assessment_methodology.md §2.3)
#
#   Recommendations:
#   • Review files that depend on this module
#   • Consider adding more test coverage
#
# Phase: 1 (Baseline)
# Duration: 187ms
# LLM calls: 0
# Cost: $0.00
```

**Phase 2: LLM Investigation (if OpenAI API key configured)**

Reference: [agentic_design.md](../../01-architecture/agentic_design.md) §2.2

```bash
# For HIGH risk files, Phase 2 runs automatically
# (Only if PHASE2_ENABLED=true and API key configured)
crisk check

# Output shows LLM investigation:
# Phase: 2 (LLM Investigation)
# Hop 1: Calculating ownership churn...
#   • Current owner: alice@example.com
#   • Previous owner: bob@example.com
#   • Transition: 14 days ago (MEDIUM ownership stability)
#     └─ Reference: risk_assessment_methodology.md §3.1
# Hop 2: Checking incident similarity...
#   • Similar to Incident #123: "Auth timeout" (BM25 score: 12.3, HIGH)
#     └─ Reference: risk_assessment_methodology.md §3.2
# Hop 3: Finalizing assessment...
#
# Final Risk Assessment:
#   Risk Level: HIGH
#   Confidence: 0.90
#
# Key Evidence:
#   1. 12 files directly depend on this code (high coupling)
#   2. 75% co-change frequency with auth.py (tight temporal coupling)
#   3. Code owner changed 14 days ago (ownership instability)
#   4. Similar to Incident #123: "Auth timeout after permission check"
#
# Recommendations:
#   1. Add integration tests for permission check timeout scenarios
#   2. Review auth module coupling (consider facade pattern)
#   3. Ensure new owner (alice@) is familiar with incident #123
#
# Duration: 4.2s
# LLM calls: 4
# Cost: ~$0.04 (from your API key)
```

**Explain mode (show investigation trace):**
```bash
# Show detailed investigation process
crisk check --explain

# Output includes:
# - Graph nodes visited
# - Metrics calculated per hop
# - LLM decision reasoning
# - Evidence chain construction
```

### Sync Graph (Manual Webhook Alternative)

**Context:** Local deployment doesn't have automatic GitHub webhooks

Reference: [graph_ontology.md](../../01-architecture/graph_ontology.md) - Update strategy

```bash
# After committing to main branch
git commit -m "Add new feature"

# Sync graph with latest changes (incremental update)
crisk sync

# This will:
# 1. Detect new commits since last sync
# 2. Extract changed files
# 3. Update graph (incremental update, ~10-30s)
# 4. Invalidate affected Redis caches
# 5. Recalculate CO_CHANGED edges (90-day window)

# Output:
# Syncing repository...
# Found 3 new commits
# Updating 12 changed files
# Re-parsing: src/auth.py, src/permissions.py, ...
# Updating CO_CHANGED relationships...
# Invalidating cache keys: coupling:*, co_change:*
# ✓ Graph updated (15.2s)
```

**Automatic sync (cron job):**
```bash
# Add to crontab (sync every hour)
crontab -e

# Add line:
0 * * * * cd /path/to/your/repo && /path/to/crisk sync >> /tmp/crisk-sync.log 2>&1

# Verify cron job:
crontab -l
```

### User Feedback (Metric Validation)

**Context:** False positive tracking for metric auto-disablement

Reference: [risk_assessment_methodology.md](../../01-architecture/risk_assessment_methodology.md) §5

```bash
# If you see a false positive
crisk check
# Output: Risk Level: HIGH (but you know it's safe)

# Provide feedback (updates metric_validations table)
crisk feedback --false-positive --reason "Intentional coupling in framework code"

# System records feedback in PostgreSQL:
# INSERT INTO metric_validations (metric_name, user_feedback, feedback_reason)
# VALUES ('coupling', 'false_positive', 'Intentional coupling in framework code')

# Updates aggregate statistics:
# UPDATE metric_stats
# SET false_positives = false_positives + 1, total_uses = total_uses + 1
# WHERE metric_name = 'coupling'

# If metric crosses 3% FP rate threshold, it's auto-disabled
# (spec.md §6.2 constraint C-10)

# View disabled metrics
crisk metrics --disabled

# Output:
# Disabled Metrics:
#   • coupling (FP rate: 4.2%, n=25)
#     Top reasons:
#     - Intentional coupling in framework code (40%)
#     - Test files inflating count (30%)
#     - Generated code (20%)
```

---

## Testing & Validation

### Verify Graph Construction

**Neo4j Browser Queries:**

```cypher
// Open Neo4j Browser: http://localhost:7474

// 1. Count nodes by type (verify all 3 layers)
MATCH (n)
RETURN labels(n) as type, count(n) as count
ORDER BY count DESC;

// Expected output (for 5K file repo):
// │ type          │ count  │
// │ File          │ 5,234  │  Layer 1: Structure
// │ Function      │ 12,456 │  Layer 1: Structure
// │ Commit        │ 892    │  Layer 2: Temporal
// │ Developer     │ 23     │  Layer 2: Temporal
// │ Incident      │ 5      │  Layer 3: Incidents

// 2. Find files with most dependencies (coupling metric)
MATCH (f:File)-[:IMPORTS]->(dep)
RETURN f.path, count(dep) as dependency_count
ORDER BY dependency_count DESC
LIMIT 10;

// 3. Check co-change patterns (temporal coupling)
MATCH (a:File)-[r:CO_CHANGED]->(b:File)
WHERE r.frequency > 0.7
RETURN a.path, b.path, r.frequency
ORDER BY r.frequency DESC
LIMIT 10;

// 4. Verify test coverage relationships
MATCH (test:File)-[:TESTS]->(source:File)
RETURN source.path, test.path, source.loc, test.loc,
       (test.loc * 1.0 / source.loc) as test_ratio
ORDER BY test_ratio DESC
LIMIT 10;

// 5. Check git history coverage (90-day window)
MATCH (c:Commit)
WITH min(c.timestamp) as oldest, max(c.timestamp) as newest,
     count(c) as total_commits
RETURN oldest, newest,
       duration.between(oldest, newest).days as days_covered,
       total_commits;
```

### Test Phase 1 Baseline

**Performance validation (spec.md §2.4 targets):**

```bash
# Test with known file
echo "test" >> src/known-file.js
git add src/known-file.js

# Run check (should complete in <500ms - NFR-1)
time crisk check

# Expected output:
# real    0m0.187s  ✓ (< 500ms target)
# user    0m0.042s
# sys     0m0.023s

# Verify Redis cache is working
# First run (cold cache)
time crisk check src/test.js  # ~187ms

# Second run (warm cache)
time crisk check src/test.js  # ~5ms (40x faster)

# Check cache hit in Redis
docker compose exec redis redis-cli KEYS "baseline:*"
# Expected: baseline:repo123:src/test.js
```

### Test Phase 2 LLM (if enabled)

**Prerequisites:**
- PHASE2_ENABLED=true
- OPENAI_API_KEY or ANTHROPIC_API_KEY set

```bash
# Test with high-risk file (triggers Phase 2)
echo "critical change" >> src/auth/permissions.py
git add src/auth/permissions.py

# Run check with debug logging
CRISK_LOG_LEVEL=debug crisk check

# Expected debug output (agentic_design.md §2.2):
# [DEBUG] Phase 1: coupling=12, co_change=0.8, test_ratio=0.2
# [DEBUG] Heuristic: HIGH risk, escalating to Phase 2
# [DEBUG] Loading 1-hop neighbors (23 nodes)
# [DEBUG] LLM decision: CALCULATE_METRIC (ownership_churn)
# [DEBUG] Hop 1 complete: ownership changed 14 days ago
# [DEBUG] LLM decision: CALCULATE_METRIC (incident_similarity)
# [DEBUG] Hop 2 complete: similar to Incident #123 (score: 12.3)
# [DEBUG] LLM decision: FINALIZE (confidence: 0.90)
# [DEBUG] Phase 2 complete (3.8s, 4 LLM calls, cost: $0.04)
```

### Load Testing (Optional)

```bash
# Test multiple concurrent checks
for i in {1..10}; do
  crisk check src/file$i.js &
done
wait

# Monitor resource usage during load test
docker stats

# Expected (medium repo):
# CONTAINER            CPU %    MEM USAGE / LIMIT
# coderisk-neo4j-1     25%      2.1GB / 4GB     ✓
# coderisk-redis-1     5%       512MB / 2GB     ✓
# coderisk-postgres-1  3%       256MB / 2GB     ✓
# coderisk-api-1       15%      128MB / 512MB   ✓
```

---

## Troubleshooting

### Issue: Neo4j out of memory

**Symptom:**
```
Error: Java heap space
Neo4j service crashes
docker compose logs neo4j | grep "OutOfMemoryError"
```

**Cause:** Insufficient heap memory for graph size

**Solution:**
```bash
# 1. Check current graph size
docker compose exec neo4j cypher-shell -u neo4j -p "PASSWORD" "
MATCH (n)
RETURN count(n) as node_count,
       count{(a)-->(b)} as edge_count
"

# 2. Increase heap size in .env
nano .env
# Change:
NEO4J_MAX_HEAP=8G  # Increase from 2G
NEO4J_PAGECACHE=4G  # Increase from 1G

# 3. Restart Neo4j
docker compose down neo4j
docker compose up -d neo4j

# 4. Verify new settings
docker compose exec neo4j cypher-shell -u neo4j -p "PASSWORD" "
CALL dbms.listConfig() YIELD name, value
WHERE name CONTAINS 'memory'
RETURN name, value
"
```

### Issue: Slow queries

**Symptom:**
```
crisk check takes >5 seconds
Phase 1 timeout errors
```

**Cause:** Missing Neo4j indexes

**Solution:**
```cypher
// Run in Neo4j Browser (http://localhost:7474)

// Check existing indexes
SHOW INDEXES;

// Create missing indexes (from graph_ontology.md)
CREATE INDEX file_path IF NOT EXISTS FOR (f:File) ON (f.path);
CREATE INDEX function_name IF NOT EXISTS FOR (fn:Function) ON (fn.name);
CREATE INDEX commit_sha IF NOT EXISTS FOR (c:Commit) ON (c.sha);

// Verify index usage on slow query
PROFILE MATCH (f:File {path: $path})-[:IMPORTS]->(dep)
RETURN count(dep);
// Should show "NodeIndexSeek" instead of "NodeByLabelScan"
```

### Issue: Redis cache not persisting

**Symptom:**
```
Cache always cold after restart
100% cache misses
```

**Cause:** Redis AOF persistence disabled

**Solution:**
```bash
# 1. Check Redis config
docker compose exec redis redis-cli CONFIG GET appendonly
# Expected: appendonly yes

# 2. If "no", update docker-compose.yml
# Already configured in our compose file:
# command: redis-server --appendonly yes

# 3. Verify AOF file exists
docker compose exec redis ls -lh /data/*.aof
# Expected: appendonly.aof

# 4. Restart Redis
docker compose restart redis
```

### Issue: Phase 2 investigations failing

**Symptom:**
```
Error: OpenAI API call failed
Phase 2 timeout
HTTP 401 Unauthorized
```

**Diagnosis & Solutions:**

```bash
# 1. Verify API key is set
echo $OPENAI_API_KEY
# Should show: sk-proj-...

# 2. Test API key directly
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"
# Expected: JSON list of models

# 3. Check API service logs
docker compose logs api | grep -i openai
# Look for "API key invalid" or "Rate limit exceeded"

# 4. Common solutions:

# Solution A: API key not loaded in container
# Re-export and restart:
export OPENAI_API_KEY=sk-proj-YOUR_KEY
docker compose restart api

# Solution B: Rate limit exceeded
# Reduce max hops in .env:
MAX_HOPS=3  # Instead of 5
docker compose restart api

# Solution C: Network timeout
# Increase timeout in .env:
INVESTIGATION_TIMEOUT_SECONDS=60  # Instead of 30
docker compose restart api

# Solution D: Disable Phase 2 (use baseline only)
nano .env
# Set: PHASE2_ENABLED=false
docker compose restart api
```

### Issue: Port conflicts

**Symptom:**
```
Error: Bind for 0.0.0.0:7474 failed: port is already allocated
```

**Solution:**
```bash
# 1. Find conflicting process
lsof -i :7474
# Shows: neo4j or other process

# 2. Option A: Stop conflicting service
sudo systemctl stop neo4j  # If system Neo4j running

# 3. Option B: Change ports in docker-compose.yml
nano docker-compose.yml
# Change:
services:
  neo4j:
    ports:
      - "17474:7474"  # Use different external port
      - "17687:7687"

# 4. Update API service connection
nano .env
# Add: NEO4J_URI=bolt://localhost:17687

# 5. Restart
docker compose down
docker compose up -d
```

---

## Advanced Configuration

### Multi-Repository Setup

**Option 1: Separate Neo4j databases (isolated)**

```yaml
# docker-compose.yml - duplicate neo4j service
services:
  neo4j-repo1:
    image: neo4j:5.15-community
    ports:
      - "7474:7474"
    volumes:
      - neo4j_repo1:/data
    environment:
      - NEO4J_AUTH=neo4j/password1

  neo4j-repo2:
    image: neo4j:5.15-community
    ports:
      - "7475:7474"  # Different external port
    volumes:
      - neo4j_repo2:/data
    environment:
      - NEO4J_AUTH=neo4j/password2
```

**Option 2: Shared Neo4j with labels**

```cypher
// All repos in same Neo4j instance
// Differentiate by repo_id property

// Add repo_id to all nodes during init
MATCH (n)
WHERE NOT n.repo_id IS NULL
SET n.repo_id = 'repo123'

// Query specific repo
MATCH (f:File {repo_id: 'repo123'})-[:IMPORTS]->(dep)
WHERE dep.repo_id = 'repo123'
RETURN count(dep)
```

### Backup and Restore

**Automated backups (cron):**

```bash
# Create backup script
cat > backup-coderisk.sh <<'EOF'
#!/bin/bash
BACKUP_DIR=/backups/coderisk
DATE=$(date +%Y%m%d_%H%M%S)

# Backup Neo4j (requires Neo4j admin tools)
docker compose exec -T neo4j neo4j-admin database dump neo4j \
  --to-path=/dumps/neo4j_$DATE.dump

# Backup PostgreSQL
docker compose exec -T postgres pg_dump -U coderisk coderisk \
  > $BACKUP_DIR/postgres_$DATE.sql

# Backup Redis (optional, cache only)
docker compose exec -T redis redis-cli --rdb /data/dump_$DATE.rdb

# Cleanup old backups (keep 7 days)
find $BACKUP_DIR -name "*.dump" -mtime +7 -delete
find $BACKUP_DIR -name "*.sql" -mtime +7 -delete
EOF

chmod +x backup-coderisk.sh

# Schedule daily backups (2 AM)
crontab -e
# Add: 0 2 * * * /path/to/backup-coderisk.sh
```

**Restore from backup:**
```bash
# Stop services
docker compose down

# Restore Neo4j
docker compose run --rm neo4j neo4j-admin database load neo4j \
  --from-path=/dumps/neo4j_20251002_020000.dump

# Restore PostgreSQL
docker compose exec -T postgres psql -U coderisk coderisk \
  < /backups/coderisk/postgres_20251002_020000.sql

# Start services
docker compose up -d
```

---

## Performance Tuning

### Neo4j Query Optimization

```cypher
// In Neo4j Browser (localhost:7474)

// 1. Analyze query performance
EXPLAIN MATCH (f:File)-[:IMPORTS]->(dep)
RETURN f.path, count(dep);

// 2. Create composite indexes for common queries
CREATE INDEX file_lang_path IF NOT EXISTS
FOR (f:File) ON (f.language, f.path);

// 3. Monitor slow queries (queries >1 second)
CALL dbms.listQueries()
YIELD query, elapsedTimeMillis
WHERE elapsedTimeMillis > 1000
RETURN query, elapsedTimeMillis
ORDER BY elapsedTimeMillis DESC;

// 4. Optimize CO_CHANGED query (used in Phase 1)
// Before optimization:
MATCH (f:File {path: $path})-[r:CO_CHANGED]-(other)
RETURN other.path, r.frequency
ORDER BY r.frequency DESC;

// After optimization (with index):
MATCH (f:File {path: $path})-[r:CO_CHANGED]-(other)
WHERE r.frequency > 0.3  // Filter before sort
RETURN other.path, r.frequency
ORDER BY r.frequency DESC
LIMIT 10;  // Limit results
```

### Redis Memory Optimization

```bash
# Connect to Redis
docker compose exec redis redis-cli

# Check memory stats
INFO memory
# Look for: used_memory_human, maxmemory_human

# Analyze key distribution
MEMORY STATS

# Check key count by pattern
KEYS baseline:* | wc -l
KEYS investigation:* | wc -l

# Set shorter TTL if memory pressure
CONFIG SET maxmemory-policy allkeys-lru

# Monitor evictions
INFO stats | grep evicted
# If evicted_keys is high, increase REDIS_MAXMEMORY
```

---

## Migration to Cloud

**When to migrate:**
- Team grows beyond 5 developers
- Need automatic webhooks for graph updates
- Want centralized management
- Hardware limits reached

```bash
# 1. Export graph from local Neo4j
docker compose exec neo4j cypher-shell -u neo4j -p PASSWORD \
  "CALL apoc.export.cypher.all('/exports/full-graph.cypher', {})"

# 2. Export metadata from PostgreSQL
docker compose exec postgres pg_dump -U coderisk coderisk > metadata.sql

# 3. Sign up for cloud account (when available)
crisk signup

# 4. Initialize cloud repo (imports graph)
crisk init --import-from /exports/full-graph.cypher \
           --import-metadata metadata.sql

# 5. Verify cloud deployment
export CRISK_API_URL=https://api.coderisk.ai
crisk check

# 6. Decommission local (optional)
docker compose down -v
rm -rf volumes/
```

---

## References

**Architecture Documentation:**
- [spec.md](../../spec.md) - Requirements, scope, constraints
- [cloud_deployment.md](../../01-architecture/cloud_deployment.md) - Local vs cloud comparison
- [graph_ontology.md](../../01-architecture/graph_ontology.md) - 3-layer graph schema
- [agentic_design.md](../../01-architecture/agentic_design.md) - Two-phase investigation flow
- [risk_assessment_methodology.md](../../01-architecture/risk_assessment_methodology.md) - Metric formulas and thresholds
- [prompt_engineering_design.md](../../01-architecture/prompt_engineering_design.md) - LLM prompt architecture

**Development Workflow:**
- [DEVELOPMENT_WORKFLOW.md](../../DEVELOPMENT_WORKFLOW.md) - Implementation guardrails and safety protocols
- [README.md](../../README.md) - Documentation navigation

**12-Factor Agent Principles:**
- [Factor 1: Natural Language to Tool Calls](../../12-factor-agents-main/content/factor-01-natural-language-to-tool-calls.md)
- [Factor 5: Unify Execution State](../../12-factor-agents-main/content/factor-05-unify-execution-state.md)
- [Factor 8: Own Your Control Flow](../../12-factor-agents-main/content/factor-08-own-your-control-flow.md)
- [Factor 12: Stateless Reducer](../../12-factor-agents-main/content/factor-12-stateless-reducer.md)

**External Documentation:**
- [Neo4j Operations Manual](https://neo4j.com/docs/operations-manual/current/)
- [Redis Documentation](https://redis.io/docs/)
- [Docker Compose Specification](https://docs.docker.com/compose/compose-file/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)

**Support:**
- GitHub Issues: https://github.com/coderisk/coderisk-go/issues
- Community Forum: https://forum.coderisk.ai/c/local-deployment

---

## Implementation Checklist for AI Agents

**Before implementing components, verify:**

- [ ] Read [spec.md](../../spec.md) sections 1.5 (scope), 4 (NFRs), 5 (technical requirements)
- [ ] Read [DEVELOPMENT_WORKFLOW.md](../../DEVELOPMENT_WORKFLOW.md) §3 (security guardrails)
- [ ] Read relevant architecture docs for component being implemented
- [ ] Understand error handling requirements (all inputs validated, no hardcoded credentials)
- [ ] Know performance targets (Phase 1 <500ms, Phase 2 3-5s)
- [ ] Understand metric validation framework (auto-disable >3% FP rate)

**Implementation priorities:**

1. **Docker Compose stack** (this guide) - Infrastructure foundation
2. **CLI commands** (`init`, `check`, `sync`, `feedback`) - User interface
3. **Phase 1 baseline** - Tier 1 metrics (coupling, co-change, test ratio)
4. **Graph construction** - Tree-sitter AST + git history → Neo4j
5. **Redis caching** - 15-min TTL, cache invalidation
6. **Phase 2 LLM** - Agent investigation with hop limits
7. **Metric validation** - FP tracking in PostgreSQL

**Testing requirements:**

- [ ] All services healthy in `docker compose ps`
- [ ] Phase 1 completes in <500ms (spec.md §2.4 NFR-1)
- [ ] Redis cache hit rate >90% (spec.md §2.4 NFR-7)
- [ ] Neo4j indexes created (query performance <50ms)
- [ ] PostgreSQL schema initialized (5 default metrics)
- [ ] Error messages are actionable (no stack traces to users)
- [ ] All credentials from environment variables (no hardcoding)

---

## Implementation Status (October 3, 2025)

**✅ COMPLETED - Priorities 1-6:**
- Priority 1: Docker Compose Stack (Neo4j, PostgreSQL, Redis, API)
- Priority 2: Go Dependencies (Neo4j driver, Redis client, LLM SDKs)
- Priority 3: API Service (Health endpoints, service integration)
- Priority 4: CLI Commands (`crisk check` functional with Phase 1 metrics)
- Priority 5: Phase 1 Baseline (Tier 1 metrics: coupling, co-change, test_ratio)
- **Priority 6: Graph Construction Layers 2 & 3** ✅
  - GitHub API → PostgreSQL staging (4m48s first run, 429ms subsequent with smart checkpointing)
  - PostgreSQL → Neo4j graph construction (8.8s for 1,247 nodes, 2,891 edges)
  - `crisk init` command fully functional
  - Validated on omnara-ai/omnara (276 commits, 71 issues, 180 PRs)

**🚧 PENDING - Priority 7-9:**
- Priority 7: Layer 1 - Code Structure (Tree-sitter for AST parsing)
- Priority 8: Phase 2 LLM Investigation (requires Phase 1 metrics)
- Priority 9: Production Hardening (Neptune backend, incremental sync)

**Current State:**
- `crisk init <owner>/<repo>` works end-to-end with real GitHub data
- `crisk check <file>` works with Phase 1 metrics
- All Tier 1 metrics implemented per risk_assessment_methodology.md
- Redis caching functional (15-min TTL)
- PostgreSQL staging layer operational with checkpointing
- Neo4j graph populated with real temporal and incident data
- Performance: 9.3s total init time (70% faster than 30s target)
- **Production validated:** Tested on omnara-ai/omnara with 1,247 nodes, 2,891 edges

**Next Steps:**
- Implement graph construction per [graph_construction.md](graph_construction.md)
- Architectural decisions documented in [scalability_analysis.md](../../01-architecture/scalability_analysis.md)

See [IMPLEMENTATION_LOG.md](../IMPLEMENTATION_LOG.md) for full details.

---

**Last Updated:** October 2, 2025

**Feedback:** If you encounter issues not covered here, please open a GitHub issue with the `local-deployment` label.
