# Cloud Migration Readiness Assessment

**Date**: 2025-11-18
**Current State**: Local Docker Infrastructure
**Target State**: AWS + Supabase + Neo4j Aura
**Migration Path**: Clear and Well-Defined

---

## Executive Summary

Your local implementation is **architected for easy cloud migration**. The microservice design, Postgres-first protocol, and stateless workers mean you can migrate to AWS infrastructure with minimal code changes.

**Migration Effort Estimate**: 1-2 weeks (mostly infrastructure setup, minimal code changes)

---

## Current Local Infrastructure ✅

### What You Have
```
┌─────────────────────────────────────────────┐
│  Local Development Environment             │
├─────────────────────────────────────────────┤
│                                             │
│  ┌─────────────┐      ┌─────────────┐     │
│  │ PostgreSQL  │      │   Neo4j     │     │
│  │ (Docker)    │      │  (Docker)   │     │
│  │ :5433       │      │   :7688     │     │
│  └─────────────┘      └─────────────┘     │
│         ▲                     ▲            │
│         │                     │            │
│         └─────────┬───────────┘            │
│                   │                        │
│         ┌─────────▼─────────┐              │
│         │  Go Microservices │              │
│         │  (6 binaries)     │              │
│         └───────────────────┘              │
│                                             │
│  Binaries: crisk-stage, crisk-ingest,      │
│            crisk-atomize, crisk-index-*    │
│                                             │
│  Orchestration: Sequential execution       │
│                 via crisk-init             │
└─────────────────────────────────────────────┘
```

### What's Already Cloud-Ready ✅

1. **Stateless Workers** ✅
   - Each binary is a standalone process
   - No shared state between runs
   - Perfect for container execution

2. **Database-First Design** ✅
   - Postgres as source of truth
   - Neo4j as derived cache
   - Checkpointing via `processed_at` timestamps

3. **Idempotent Operations** ✅
   - `ON CONFLICT` upserts in Postgres
   - `MERGE` operations in Neo4j
   - Can re-run safely after failures

4. **Microservice Separation** ✅
   - Clear input/output contracts
   - Each service reads from Postgres, writes to both DBs
   - No inter-service communication needed

5. **Configuration Externalization** ✅
   - Environment variable driven
   - No hardcoded paths or credentials
   - Easy to inject cloud secrets

---

## Target Cloud Infrastructure (AWS)

### Recommended Architecture
```
┌───────────────────────────────────────────────────────────────┐
│  AWS Cloud Environment                                        │
├───────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐ │
│  │ API Gateway  │────▶│   Lambda     │────▶│     SQS      │ │
│  │              │     │  Authorizer  │     │ "init-queue" │ │
│  └──────────────┘     └──────────────┘     └──────┬───────┘ │
│         ▲                                          │         │
│         │                                          │         │
│         │                                          ▼         │
│  ┌──────┴──────┐                        ┌──────────────────┐ │
│  │   GitHub    │                        │  Step Functions  │ │
│  │  Webhooks   │                        │  (Orchestrator)  │ │
│  └─────────────┘                        └─────────┬────────┘ │
│                                                    │          │
│                                                    ▼          │
│                                         ┌──────────────────┐ │
│                                         │   AWS Batch      │ │
│                                         │   (Job Queue)    │ │
│                                         └─────────┬────────┘ │
│                                                   │          │
│                                                   ▼          │
│                          ┌────────────────────────────────┐ │
│                          │   Fargate Workers (Private)    │ │
│                          │                                │ │
│                          │  ┌───────────────────────┐    │ │
│                          │  │ crisk-worker:latest   │    │ │
│                          │  │ (All 6 microservices) │    │ │
│                          │  └───────────────────────┘    │ │
│                          └────────┬───────────┬──────────┘ │
│                                   │           │            │
│                                   ▼           ▼            │
│                          ┌──────────┐  ┌──────────┐       │
│                          │ Supabase │  │  Neo4j   │       │
│                          │ (Postgres│  │   Aura   │       │
│                          └──────────┘  └──────────┘       │
│                                                            │
│  Check API:                                               │
│  ┌──────────────┐                                         │
│  │   Lambda     │────────────────┐                        │
│  │ (crisk check)│                │                        │
│  └──────────────┘                ▼                        │
│                          ┌──────────┐  ┌──────────┐       │
│                          │ Supabase │  │  Neo4j   │       │
│                          │          │  │   Aura   │       │
│                          └──────────┘  └──────────┘       │
└───────────────────────────────────────────────────────────┘
```

---

## Migration Mapping: Local → Cloud

### 1. Databases

| Local | Cloud | Migration Effort |
|-------|-------|------------------|
| PostgreSQL Docker (port 5433) | Supabase Postgres | **LOW** - Change connection string |
| Neo4j Docker (port 7688) | Neo4j Aura | **LOW** - Change connection string |

**Code Changes Required:**
```go
// Before (local):
cfg.Postgres.Host = "localhost"
cfg.Postgres.Port = 5433
cfg.Neo4j.URI = "bolt://localhost:7688"

// After (cloud):
cfg.Postgres.Host = os.Getenv("SUPABASE_HOST")
cfg.Postgres.Port = 5432
cfg.Neo4j.URI = os.Getenv("NEO4J_AURA_URI")
```

**Migration Steps:**
1. Export Postgres data: `pg_dump`
2. Import to Supabase: `psql -h <supabase-host>`
3. Export Neo4j data: `neo4j-admin dump`
4. Import to Neo4j Aura: `neo4j-admin load`

**Estimated Time**: 4 hours

---

### 2. Compute (Microservices)

| Local | Cloud | Migration Effort |
|-------|-------|------------------|
| 6 separate binaries | 1 Docker image with all binaries | **MEDIUM** - Create Dockerfile |
| Sequential execution via `crisk-init` | AWS Step Functions | **MEDIUM** - ASL JSON definition |
| Direct execution | AWS Batch jobs on Fargate | **LOW** - Job definitions |

**Docker Image Strategy:**
```dockerfile
# Single image with all microservices
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN make build

FROM alpine:latest
RUN apk add --no-cache git ca-certificates
COPY --from=builder /app/bin/* /app/
ENTRYPOINT ["/app/entrypoint.sh"]
```

**Entrypoint Strategy:**
```bash
#!/bin/sh
# entrypoint.sh - Runs different services based on env var
case "$SERVICE_NAME" in
    stage)
        /app/crisk-stage "$@"
        ;;
    ingest)
        /app/crisk-ingest "$@"
        ;;
    atomize)
        /app/crisk-atomize "$@"
        ;;
    # ... other services
esac
```

**Step Functions Definition:**
```json
{
  "Comment": "crisk-init pipeline",
  "StartAt": "Stage",
  "States": {
    "Stage": {
      "Type": "Task",
      "Resource": "arn:aws:batch:...",
      "Parameters": {
        "JobDefinition": "crisk-worker",
        "Environment": [
          {"Name": "SERVICE_NAME", "Value": "stage"}
        ]
      },
      "Next": "Ingest"
    },
    "Ingest": {
      "Type": "Task",
      "Resource": "arn:aws:batch:...",
      "Parameters": {
        "JobDefinition": "crisk-worker",
        "Environment": [
          {"Name": "SERVICE_NAME", "Value": "ingest"}
        ]
      },
      "Next": "Atomize"
    }
    // ... continue for all 6 services
  }
}
```

**Estimated Time**: 2 days

---

### 3. Orchestration

| Local | Cloud | Migration Effort |
|-------|-------|------------------|
| `crisk-init` binary | AWS Step Functions | **MEDIUM** - ASL definition |
| Direct function calls | Job submissions | **LOW** - API calls |
| Error handling in code | Step Functions retry | **LOW** - Configuration |

**Current crisk-init Logic:**
```go
// Sequential execution (local)
func runInit() error {
    repoID := runStage()
    runIngest(repoID)
    runAtomize(repoID)
    runIndexIncident(repoID)
    runIndexOwnership(repoID)
    runIndexCoupling(repoID)
    return nil
}
```

**Step Functions Equivalent:**
```json
{
  "States": {
    "Stage": { "Next": "Ingest" },
    "Ingest": { "Next": "Atomize" },
    "Atomize": {
      "Type": "Parallel",
      "Branches": [
        {"StartAt": "IndexIncident"},
        {"StartAt": "IndexOwnership"},
        {"StartAt": "IndexCoupling"}
      ]
    }
  }
}
```

**Benefits:**
- Built-in retry logic
- Visual workflow monitoring
- Automatic error catching
- Branching/parallel execution

**Estimated Time**: 1 day

---

### 4. API Gateway (Check Endpoint)

| Local | Cloud | Migration Effort |
|-------|-------|------------------|
| `crisk check` CLI | Lambda function | **LOW** - Wrap existing logic |
| Direct execution | API Gateway REST endpoint | **LOW** - Configuration |
| No auth | Lambda Authorizer (Clerk JWT) | **MEDIUM** - Implement authorizer |

**Lambda Implementation:**
```go
// cmd/lambda-check/main.go
package main

import (
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/coderisk/internal/risk"
)

type CheckRequest struct {
    FilePath  string `json:"file_path"`
    BlockName string `json:"block_name"`
    RepoID    int64  `json:"repo_id"`
}

func handleCheck(req CheckRequest) (risk.RiskReport, error) {
    // Same logic as cmd/crisk/check.go
    return risk.GetCodeBlockRisk(req.RepoID, req.FilePath, req.BlockName)
}

func main() {
    lambda.Start(handleCheck)
}
```

**API Gateway Configuration:**
```
POST /check
  ├─ Lambda Authorizer (validates Clerk JWT)
  ├─ Lambda Integration (crisk-check)
  └─ Response: RiskReport JSON
```

**Estimated Time**: 1 day

---

### 5. Webhooks (Incremental Updates)

| Local | Cloud | Migration Effort |
|-------|-------|------------------|
| Manual `crisk init` | GitHub Webhook → Lambda → SQS | **MEDIUM** - New feature |
| N/A | Incremental stage worker | **LOW** - Flag existing code |

**Webhook Flow:**
```
GitHub Push Event
    │
    ▼
API Gateway (/webhook)
    │
    ▼
Lambda (verify signature)
    │
    ▼
SQS (webhook-queue)
    │
    ▼
Step Functions (incremental)
    │
    ▼
AWS Batch (crisk-stage --incremental)
```

**Code Changes:**
```go
// cmd/crisk-stage/main.go
incremental := flag.Bool("incremental", false, "Incremental mode")
commitSHA := flag.String("commit", "", "Specific commit to process")

if *incremental {
    // Only fetch new commits since MAX(created_at)
    // Only fetch specific commit if commitSHA provided
}
```

**Estimated Time**: 2 days

---

## Code Changes Summary

### Minimal Changes Required ✅

**1. Configuration (Environment Variables)**
```go
// internal/config/config.go
type Config struct {
    Postgres struct {
        Host     string // Read from ENV
        Port     int
        User     string
        Password string // AWS Secrets Manager
        DB       string
    }
    Neo4j struct {
        URI      string // Read from ENV
        Password string // AWS Secrets Manager
    }
    LLM struct {
        Provider string
        APIKey   string // AWS Secrets Manager
    }
}

func LoadConfig() (*Config, error) {
    // Use viper or similar to load from ENV
    return &Config{
        Postgres: PostgresConfig{
            Host:     os.Getenv("POSTGRES_HOST"),
            Password: getSecret("POSTGRES_PASSWORD"),
        },
    }
}
```

**2. Docker Entrypoint**
```bash
# scripts/entrypoint.sh (NEW FILE)
#!/bin/sh
set -e

case "$SERVICE_NAME" in
    stage)     exec /app/crisk-stage "$@" ;;
    ingest)    exec /app/crisk-ingest "$@" ;;
    atomize)   exec /app/crisk-atomize "$@" ;;
    incident)  exec /app/crisk-index-incident "$@" ;;
    ownership) exec /app/crisk-index-ownership "$@" ;;
    coupling)  exec /app/crisk-index-coupling "$@" ;;
    *)         echo "Unknown service: $SERVICE_NAME"; exit 1 ;;
esac
```

**3. Lambda Wrapper (Check Endpoint)**
```go
// cmd/lambda-check/main.go (NEW FILE)
package main

import (
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/coderisk/cmd/crisk"
)

func handler(req CheckRequest) (CheckResponse, error) {
    // Reuse existing check logic from cmd/crisk/check.go
    return crisk.RunCheck(req.RepoID, req.FilePath, req.BlockName)
}

func main() {
    lambda.Start(handler)
}
```

**Total New Code**: ~200 lines
**Total Modified Code**: ~50 lines (config loading)

---

## Infrastructure as Code (IaC)

### Recommended: Terraform

**Modules to Create:**

1. **VPC & Networking** (~100 lines)
   - Private subnets for Fargate
   - NAT gateway for outbound access
   - Security groups

2. **AWS Batch** (~150 lines)
   - Compute environment (Fargate Spot)
   - Job queue
   - Job definition (crisk-worker container)

3. **Step Functions** (~200 lines)
   - State machine definition (ASL)
   - IAM roles
   - CloudWatch logs

4. **API Gateway** (~150 lines)
   - REST API
   - Lambda integrations
   - Authorizer

5. **Secrets Manager** (~50 lines)
   - Supabase credentials
   - Neo4j Aura credentials
   - Gemini API key
   - GitHub token

6. **ECR** (~30 lines)
   - Docker image repository

**Total Terraform**: ~680 lines

**Estimated Time**: 3 days

---

## Migration Checklist

### Phase 1: Database Migration (1 day)
- [ ] Create Supabase project
- [ ] Create Neo4j Aura instance
- [ ] Export local Postgres data
- [ ] Import to Supabase
- [ ] Export local Neo4j data
- [ ] Import to Neo4j Aura
- [ ] Test connections from local machine

### Phase 2: Containerization (1 day)
- [ ] Create Dockerfile
- [ ] Create entrypoint.sh
- [ ] Build Docker image locally
- [ ] Test all 6 services in container
- [ ] Push to AWS ECR

### Phase 3: AWS Infrastructure (3 days)
- [ ] Write Terraform modules
- [ ] Create VPC and networking
- [ ] Set up AWS Batch (Fargate)
- [ ] Create Step Functions
- [ ] Set up API Gateway
- [ ] Configure Secrets Manager
- [ ] Deploy and test

### Phase 4: Lambda Functions (1 day)
- [ ] Create Lambda authorizer (Clerk JWT)
- [ ] Create Lambda check endpoint
- [ ] Test API Gateway integration
- [ ] Add CloudWatch logging

### Phase 5: Webhook Integration (2 days)
- [ ] Create webhook receiver Lambda
- [ ] Set up SQS queue
- [ ] Configure GitHub webhooks
- [ ] Test incremental updates
- [ ] Add error handling

### Phase 6: Testing & Monitoring (2 days)
- [ ] End-to-end testing
- [ ] Load testing
- [ ] Set up CloudWatch dashboards
- [ ] Configure alarms
- [ ] Document runbooks

**Total Migration Time**: 10 business days (2 weeks)

---

## Cost Estimates (Monthly, 100 repos)

### Current (Local)
- Development machine: $0 (your laptop)
- Electricity: ~$5

**Total**: ~$5/month

### Cloud (AWS + Supabase + Neo4j Aura)

**Compute (AWS Batch on Fargate Spot):**
- Ingestion: ~100 repos × 30 min × $0.04/hour = $2/month
- Indexing: ~100 repos × 10 min × $0.04/hour = $0.67/month
- Check API (Lambda): ~10,000 requests × $0.20/1M = $0.002/month

**Databases:**
- Supabase (Free tier): $0 (up to 500MB)
- Supabase (Pro): $25/month (2GB)
- Neo4j Aura (Professional): $65/month (8GB)

**Storage:**
- S3 (git repos): ~10GB × $0.023/GB = $0.23/month
- ECR (Docker images): ~2GB × $0.10/GB = $0.20/month

**Data Transfer:**
- Outbound: ~50GB × $0.09/GB = $4.50/month

**LLM API (Gemini):**
- With dual-pipeline: ~$500-1,000/month
- Without dual-pipeline: ~$5,000-10,000/month

**Total (with dual-pipeline)**: ~$600-1,100/month
**Total (without dual-pipeline)**: ~$5,100-10,100/month

**ROI of Dual-Pipeline**: $4,500-9,000/month savings (90% reduction)

---

## Advantages of Cloud Migration

### 1. Scalability ✅
- **Local**: Single machine, sequential processing
- **Cloud**: Parallel Fargate tasks, process 100s of repos simultaneously

### 2. Availability ✅
- **Local**: Downtime during laptop sleep/restart
- **Cloud**: 99.9% uptime SLA

### 3. Webhooks ✅
- **Local**: Manual `crisk init` runs
- **Cloud**: Automatic incremental updates on every push

### 4. Collaboration ✅
- **Local**: Single developer
- **Cloud**: Multiple developers, shared data

### 5. Monitoring ✅
- **Local**: Local logs only
- **Cloud**: CloudWatch dashboards, alarms, distributed tracing

### 6. Cost Optimization ✅
- **Local**: Fixed cost (laptop)
- **Cloud**: Pay-per-use (Spot instances)

---

## Recommended Migration Path

### Option 1: Gradual Migration (Lower Risk)
1. **Week 1**: Migrate databases only (Supabase + Neo4j Aura)
   - Keep running microservices locally
   - Point to cloud databases
   - Validate data sync

2. **Week 2**: Containerize and deploy workers
   - Build Docker image
   - Deploy to AWS Batch
   - Test with small repos

3. **Week 3**: Add API Gateway and Lambda
   - Deploy check endpoint
   - Test from production clients
   - Monitor performance

4. **Week 4**: Add webhooks
   - Set up incremental updates
   - Test with real repositories
   - Full production cutover

**Total**: 4 weeks, minimal risk

---

### Option 2: Full Migration (Faster)
1. **Days 1-3**: Infrastructure setup (Terraform)
2. **Days 4-5**: Containerization and deployment
3. **Days 6-7**: API Gateway and Lambda
4. **Days 8-9**: Webhooks and testing
5. **Day 10**: Production cutover

**Total**: 2 weeks, higher risk

---

## Recommendation

**For Your Use Case**: Option 1 (Gradual Migration)

**Rationale:**
- Current local setup works perfectly
- No production users yet
- Can test cloud incrementally
- Lower risk of data loss

**Start With**:
1. Migrate databases (Supabase + Neo4j Aura) - 1 day
2. Test existing code with cloud databases - 1 day
3. Validate performance and cost - 1 week

Then decide if full migration is needed based on:
- Performance requirements
- Cost constraints
- User demand
- Collaboration needs

---

## Conclusion

Your local implementation is **exceptionally well-architected for cloud migration**:

✅ Stateless microservices (container-ready)
✅ Database-first design (cloud database compatible)
✅ Idempotent operations (retry-safe)
✅ Environment-driven config (secrets-ready)
✅ Clear service boundaries (Step Functions compatible)

**Migration Complexity**: **LOW to MEDIUM**
**Migration Time**: **2-4 weeks**
**Code Changes**: **<300 lines**
**Infrastructure Code**: **~680 lines Terraform**

You can migrate incrementally with minimal risk, starting with just the databases and gradually adding cloud compute as needed.

**Ready to migrate**: ✅ **YES**
**Recommended start date**: After local validation is complete (this week)
