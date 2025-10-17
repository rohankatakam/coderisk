# Clear: Streamlined MVP Architecture for Rapid Launch

## Executive Summary

This document presents a dramatically simplified architecture that cuts 70% of the complexity from the original plans while maintaining the core value proposition. We focus on **what ships in 30 days**, using Render for hosting, avoiding Kubernetes entirely, and leveraging LLMs strategically to reduce development time.

**Key Decisions:**
- **Render.com** for all hosting (no K8s needed)
- **Single PostgreSQL** database (no multi-database complexity)
- **OpenAI/Gemini APIs** with smart caching (no local LLMs)
- **Monolithic Go API** (no microservices)
- **React SPA** with minimal dependencies

## Core Value Proposition (Unchanged)

**Clear provides requirements quality scoring and compliance gap analysis through a simple CSV upload interface.**

Target: Systems engineers in regulated industries who need quick requirement quality assessment.

## Simplified Tech Stack

### What We're KEEPING:
```
Frontend:       React + TypeScript (simple SPA)
Backend:        Go + Gin (monolithic API)
Database:       PostgreSQL with JSONB (single DB)
LLM:           OpenAI GPT-4o-mini / Gemini Flash (cached)
Hosting:       Render.com (Web + DB + Redis)
Storage:       Render Disks / S3 for files
```

### What We're CUTTING:
```
❌ Neo4j (graph database) - Use PostgreSQL recursive CTEs
❌ DuckDB (analytics) - PostgreSQL handles this fine at MVP scale
❌ Qdrant (vector DB) - PostgreSQL pgvector extension
❌ Kubernetes - Render handles scaling
❌ Multiple LLM providers - Start with one, add later
❌ Complex caching layers - Simple Redis key-value
❌ Microservices - Monolith is faster to build
❌ Event-driven architecture - Direct function calls
```

## MVP Architecture (30 Days)

```
┌─────────────────────────────────────────────┐
│         React Frontend (Render Static)       │
│                                              │
│  • Upload CSV                                │
│  • View Dashboard                            │
│  • Export Reports                            │
└─────────────────────────────────────────────┘
                        ↓ HTTPS
┌─────────────────────────────────────────────┐
│      Go Monolithic API (Render Web)          │
│                                              │
│  /api/upload    - Process CSV                │
│  /api/analyze   - Run quality checks         │
│  /api/dashboard - Get results                │
│  /api/export    - Generate reports           │
└─────────────────────────────────────────────┘
                        ↓
┌─────────────────┬───────────────────────────┐
│   PostgreSQL    │     Redis Cache           │
│   (Render DB)   │    (Render Redis)         │
│                 │                            │
│ • requirements  │ • LLM responses           │
│ • projects      │ • Analysis results        │
│ • analysis_runs │ • Session data            │
└─────────────────┴───────────────────────────┘
                        ↓
┌─────────────────────────────────────────────┐
│         OpenAI API / Gemini API              │
│                                              │
│  Smart usage:                                │
│  • Clarity analysis only                     │
│  • Cached aggressively                       │
│  • $0.02 per requirement MAX                 │
└─────────────────────────────────────────────┘
```

## Database Schema (PostgreSQL Only)

```sql
-- Single database handles everything
CREATE DATABASE clear_mvp;

-- Core tables
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    industry VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW(),
    metadata JSONB
);

CREATE TABLE requirements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id),
    external_id VARCHAR(255), -- From CSV
    text TEXT NOT NULL,
    type VARCHAR(50),
    level VARCHAR(50), -- UR, SR, DR, TC
    parent_id UUID REFERENCES requirements(id),

    -- Quality scores (computed)
    quality_score FLOAT,
    quality_breakdown JSONB,

    -- Compliance mapping
    compliance_mappings JSONB,

    -- For graph operations
    path_array UUID[], -- Materialized path for hierarchy

    created_at TIMESTAMP DEFAULT NOW(),
    metadata JSONB
);

-- Analysis results
CREATE TABLE analysis_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id),
    status VARCHAR(50),
    results JSONB, -- Store everything as JSON
    created_at TIMESTAMP DEFAULT NOW()
);

-- Simple caching for LLM
CREATE TABLE llm_cache (
    hash VARCHAR(64) PRIMARY KEY, -- SHA256 of input
    response JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_requirements_project ON requirements(project_id);
CREATE INDEX idx_requirements_parent ON requirements(parent_id);
CREATE INDEX idx_requirements_path ON requirements USING GIN(path_array);
CREATE INDEX idx_requirements_quality ON requirements(quality_score);
```

## Core Services (Simplified)

### 1. CSV Import Service (Week 1)

```go
type ImportService struct {
    db *sql.DB
}

func (s *ImportService) ProcessCSV(file io.Reader, projectID string) error {
    // Simple CSV parsing
    records, _ := csv.NewReader(file).ReadAll()

    // Auto-detect format (80% of formats follow patterns)
    format := s.detectFormat(records[0])

    // Bulk insert into PostgreSQL
    batch := []Requirement{}
    for _, record := range records[1:] {
        req := s.parseRequirement(record, format)
        batch = append(batch, req)

        if len(batch) >= 100 {
            s.bulkInsert(batch)
            batch = []Requirement{}
        }
    }

    // Build hierarchy using parent references
    s.buildHierarchy(projectID)

    return nil
}
```

### 2. Quality Analysis (Week 2)

```go
type QualityAnalyzer struct {
    db       *sql.DB
    llm      *LLMClient
    cache    *redis.Client
}

func (qa *QualityAnalyzer) AnalyzeRequirement(req Requirement) QualityScore {
    score := QualityScore{}

    // Rule-based checks (fast, free)
    score.Completeness = qa.checkCompleteness(req)  // Has all fields
    score.Atomicity = qa.checkAtomicity(req)        // Single concept
    score.Testability = qa.checkTestability(req)    // Has criteria

    // LLM only for clarity (cached)
    cacheKey := fmt.Sprintf("clarity:%s", hash(req.Text))
    if cached := qa.cache.Get(cacheKey); cached != nil {
        score.Clarity = cached.Value
    } else {
        score.Clarity = qa.llm.AnalyzeClarity(req.Text)
        qa.cache.Set(cacheKey, score.Clarity, 7*24*time.Hour)
    }

    // Simple weighted average
    score.Overall = (score.Completeness*0.2 +
                    score.Clarity*0.3 +
                    score.Testability*0.3 +
                    score.Atomicity*0.2)

    return score
}
```

### 3. Gap Detection (Week 2)

```go
func (gd *GapDetector) FindGaps(projectID string) []Gap {
    gaps := []Gap{}

    // Use PostgreSQL recursive CTEs for hierarchy traversal
    query := `
        WITH RECURSIVE req_tree AS (
            SELECT id, parent_id, level, text
            FROM requirements
            WHERE project_id = $1 AND parent_id IS NULL

            UNION ALL

            SELECT r.id, r.parent_id, r.level, r.text
            FROM requirements r
            JOIN req_tree rt ON r.parent_id = rt.id
        )
        SELECT * FROM req_tree;
    `

    requirements := db.Query(query, projectID)

    // Simple gap rules
    for _, req := range requirements {
        // UR without SR
        if req.Level == "UR" && !hasChild(req, "SR") {
            gaps = append(gaps, Gap{
                Type: "missing_sr",
                Requirement: req.ID,
                Message: "User requirement lacks system requirement",
                Severity: "high",
            })
        }

        // SR without test
        if req.Level == "SR" && !hasVerification(req) {
            gaps = append(gaps, Gap{
                Type: "untested",
                Requirement: req.ID,
                Message: "System requirement has no test case",
                Severity: "medium",
            })
        }
    }

    return gaps
}
```

### 4. Compliance Mapping (Week 3)

```go
type ComplianceMapper struct {
    standards map[string][]Clause // Preloaded
    llm       *LLMClient
    cache     *redis.Client
}

func (cm *ComplianceMapper) MapToStandard(req Requirement, standard string) []Mapping {
    // Start with keyword matching (free, fast)
    keywordMatches := cm.findKeywordMatches(req, standard)

    // Only use LLM for ambiguous cases
    if len(keywordMatches) == 0 || keywordMatches[0].Confidence < 0.5 {
        // Cached LLM call
        cacheKey := fmt.Sprintf("compliance:%s:%s", hash(req.Text), standard)
        if cached := cm.cache.Get(cacheKey); cached != nil {
            return cached.Mappings
        }

        mappings := cm.llm.MapToStandard(req.Text, standard)
        cm.cache.Set(cacheKey, mappings, 30*24*time.Hour)
        return mappings
    }

    return keywordMatches
}
```

## LLM Cost Optimization Strategy

### Smart LLM Usage (Target: $0.02 per requirement)

```go
type LLMClient struct {
    client *openai.Client
    cache  *redis.Client
}

func (lc *LLMClient) AnalyzeClarity(text string) float64 {
    // Use cheapest model that works
    model := "gpt-4o-mini" // $0.15 per 1M input tokens

    // Minimal prompt
    prompt := fmt.Sprintf(
        "Rate clarity 0-100: '%s'. Return only number.",
        text[:min(500, len(text))], // Truncate long requirements
    )

    response := lc.client.Complete(prompt, model)
    return parseFloat(response)
}

// Batch processing for efficiency
func (lc *LLMClient) BatchAnalyze(requirements []string) []float64 {
    // Send 10 requirements in one call
    prompt := "Rate each requirement's clarity 0-100. Return CSV of scores.\n"
    for i, req := range requirements[:min(10, len(requirements))] {
        prompt += fmt.Sprintf("%d. %s\n", i+1, req[:200])
    }

    response := lc.client.Complete(prompt, "gpt-4o-mini")
    return parseCSV(response)
}
```

### Caching Strategy

```go
// Three-tier caching
type CacheManager struct {
    memory *lru.Cache      // In-memory (instant)
    redis  *redis.Client   // Redis (milliseconds)
    db     *sql.DB        // PostgreSQL (backup)
}

func (cm *CacheManager) Get(key string) interface{} {
    // L1: Memory
    if val := cm.memory.Get(key); val != nil {
        return val
    }

    // L2: Redis
    if val := cm.redis.Get(key); val != nil {
        cm.memory.Set(key, val)
        return val
    }

    // L3: Database
    var val interface{}
    cm.db.QueryRow("SELECT response FROM llm_cache WHERE hash = $1", key).Scan(&val)
    if val != nil {
        cm.redis.Set(key, val, 24*time.Hour)
        cm.memory.Set(key, val)
    }

    return val
}
```

## Render.com Deployment Configuration

### Services Setup

```yaml
# render.yaml
services:
  # Go API
  - type: web
    name: clear-api
    env: go
    buildCommand: go build -o bin/server cmd/server/main.go
    startCommand: ./bin/server
    envVars:
      - key: DATABASE_URL
        fromDatabase:
          name: clear-db
          property: connectionString
      - key: REDIS_URL
        fromService:
          name: clear-redis
          property: connectionString
      - key: OPENAI_API_KEY
        sync: false # Set manually

  # React Frontend
  - type: web
    name: clear-frontend
    env: static
    buildCommand: npm run build
    staticPublishPath: ./build
    routes:
      - type: rewrite
        source: /*
        destination: /index.html

# Database
databases:
  - name: clear-db
    plan: starter # $7/month

# Redis
services:
  - type: redis
    name: clear-redis
    plan: starter # $10/month
```

### Environment Variables

```bash
# Production (.env.production)
DATABASE_URL=postgresql://user:pass@host:5432/clear
REDIS_URL=redis://host:6379
OPENAI_API_KEY=sk-...
JWT_SECRET=...
FRONTEND_URL=https://clear.onrender.com
```

## Scaling Strategy (When Needed)

### Phase 1: MVP (0-10 customers)
- Single Render web service
- Starter PostgreSQL
- Basic Redis cache
- **Cost: ~$50/month**

### Phase 2: Growth (10-50 customers)
- Scale to Standard plan on Render
- Add read replicas for PostgreSQL
- Implement job queues with BullMQ
- **Cost: ~$200/month**

### Phase 3: Scale (50+ customers)
- Add CDN for frontend (Cloudflare)
- PostgreSQL with pgbouncer
- Consider adding specialized databases
- **Cost: ~$500/month**

### When to Add Complexity

**Add Neo4j when:**
- Graph traversals take >2 seconds
- Need complex relationship queries
- Have >100K requirements

**Add Vector DB when:**
- Semantic search becomes critical
- Need similarity matching at scale
- Have >1M requirements

**Add Kubernetes when:**
- Need multi-region deployment
- Require 99.99% uptime SLA
- Have >$100K MRR

## Implementation Timeline (30 Days)

### Week 1: Foundation
- [ ] Day 1-2: Set up Go project, PostgreSQL schema
- [ ] Day 3-4: CSV import with format detection
- [ ] Day 5: Basic CRUD API endpoints

### Week 2: Core Analysis
- [ ] Day 6-7: Quality scoring engine (rules-based)
- [ ] Day 8-9: Gap detection algorithms
- [ ] Day 10: LLM integration for clarity

### Week 3: Intelligence Layer
- [ ] Day 11-12: Compliance mapping (keyword + LLM)
- [ ] Day 13-14: Caching layer implementation
- [ ] Day 15: Report generation

### Week 4: Frontend & Polish
- [ ] Day 16-18: React dashboard
- [ ] Day 19-20: Upload flow and progress tracking
- [ ] Day 21: Testing with sample data

### Week 5-6: Launch Preparation
- [ ] Deploy to Render
- [ ] Load testing
- [ ] Documentation
- [ ] First customer demos

## Cost Analysis

### Development Costs (Monthly)
```
Render Hosting:     $50  (Web + DB + Redis)
OpenAI API:         $100 (5000 requirements analyzed)
Domain/SSL:         $10
Total:              $160/month
```

### Per-Customer Economics
```
Average Requirements: 1000 per project
LLM Cost:           $20 (with caching)
Hosting Allocation: $5
Margin at $500/mo:  95%
```

## Key Simplifications That Save Time

1. **One Database**: PostgreSQL handles documents (JSONB), relationships (foreign keys), and even vectors (pgvector) if needed later.

2. **Monolithic API**: Faster to develop, debug, and deploy. Microservices can wait until $1M ARR.

3. **Smart LLM Usage**: Only for clarity analysis, everything else is rules-based. Aggressive caching keeps costs under $0.02/requirement.

4. **No Kubernetes**: Render handles scaling automatically. K8s complexity isn't needed until much later.

5. **Simple Frontend**: Basic React SPA with minimal dependencies. No state management library until needed.

6. **CSV-Only**: No integrations in MVP. This eliminates 50% of complexity while still delivering value.

## Success Metrics

### Technical
- Process 1000 requirements in <30 seconds
- LLM costs <$0.02 per requirement
- 95% cache hit rate after first analysis
- <2 second page load time

### Business
- Ship in 30 days
- First paying customer in Week 5
- $10K MRR by Month 3
- 80% gross margin

## Risk Mitigation

### Technical Risks
- **PostgreSQL performance**: Add indexes, use materialized views
- **LLM costs**: Aggressive caching, batch processing
- **Render limitations**: Can migrate to AWS later if needed

### Business Risks
- **Feature creep**: Stay focused on CSV upload → quality score
- **Integration requests**: Promise for Phase 2, not MVP
- **Enterprise requirements**: Start with SMBs, grow upmarket

## Conclusion

This streamlined architecture delivers 90% of the value with 30% of the complexity. By using Render, PostgreSQL, and smart LLM caching, we can ship in 30 days instead of 90. The architecture is simple enough for one developer to maintain but scalable enough to handle our first 50 customers without major rewrites.

**Next Step**: Start building the CSV import service and basic API endpoints. Everything else builds on that foundation.