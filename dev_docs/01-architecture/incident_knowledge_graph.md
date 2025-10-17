# Incident Knowledge Graph: CVE for Architecture

**Last Updated:** October 4, 2025
**Status:** Design Phase - Q1 2026 Implementation
**Owner:** Architecture Team

> **📘 Product Context:** See [strategic_moats.md](../00-product/strategic_moats.md) for business rationale (Cornered Resource strategy)

---

## Executive Summary

The **Incident Knowledge Graph** is CodeRisk's foundational data asset—a cross-industry database linking **commits → patterns → incidents** in a privacy-preserving manner. This creates an irreplaceable cornered resource similar to MITRE's CVE database but for architectural risks.

**Strategic Value:**
- **Cornered Resource:** 5-10 year data advantage over competitors
- **Network Effect:** More incidents = better predictions (exponential value)
- **Moat:** Requires 10K+ incidents to build, years to replicate

**Core Components:**
1. **ARC Database** (Architectural Risk Catalog) - Public incident patterns
2. **Causal Graph** - Commit → Deploy → Incident linkage
3. **Pattern Matching** - Similarity search across organizations
4. **Federated Learning** - Privacy-preserving cross-org intelligence

---

## Architecture Overview

### System Diagram

```
┌──────────────────────────────────────────────────────────────┐
│                 Incident Knowledge Graph                      │
│                                                               │
│  ┌─────────────┐   ┌──────────────┐   ┌─────────────────┐  │
│  │  ARC Public │   │ Causal Graph │   │ Pattern         │  │
│  │  Database   │   │  (Private)   │   │ Matcher         │  │
│  │             │   │              │   │ (Federated)     │  │
│  │  - ARC-001  │   │ Commit ────► │   │                 │  │
│  │  - ARC-002  │   │   │          │   │ Graph           │  │
│  │  - ...      │   │   ▼          │   │ Signatures      │  │
│  │             │   │ Deploy ────► │   │ (Hashed)        │  │
│  │ 100+ risks  │   │   │          │   │                 │  │
│  └─────────────┘   │   ▼          │   │ Similarity      │  │
│                     │ Incident     │   │ Search          │  │
│                     └──────────────┘   └─────────────────┘  │
│                                                               │
│  Data Sources:                                                │
│  - Datadog / Sentry (monitoring)                             │
│  - PagerDuty (incident management)                           │
│  - GitHub (commit metadata)                                  │
│  - CodeRisk checks (risk scores)                             │
└──────────────────────────────────────────────────────────────┘

                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                    Storage Layer                              │
│                                                               │
│  ┌─────────────┐   ┌──────────────┐   ┌─────────────────┐  │
│  │  Neptune    │   │  PostgreSQL  │   │  Redis          │  │
│  │  Graph DB   │   │  (Metadata)  │   │  (Cache)        │  │
│  │             │   │              │   │                 │  │
│  │ - Causal    │   │ - ARC        │   │ - Pattern       │  │
│  │   graphs    │   │   entries    │   │   matches       │  │
│  │ - Patterns  │   │ - Incidents  │   │ - Similarity    │  │
│  │             │   │ - Companies  │   │   scores        │  │
│  └─────────────┘   └──────────────┘   └─────────────────┘  │
└──────────────────────────────────────────────────────────────┘

                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                    Public API                                 │
│                                                               │
│  GET  /v1/arc/{arc_id}           # Retrieve ARC entry        │
│  POST /v1/arc/search             # Search by pattern         │
│  POST /v1/incidents/submit       # Submit incident           │
│  GET  /v1/patterns/similar       # Find similar patterns     │
└──────────────────────────────────────────────────────────────┘
```

---

## Component 1: ARC Database (Public Incident Catalog)

### Schema Design

**ARC Entry Structure:**
```sql
-- PostgreSQL schema
CREATE TABLE arc_entries (
    arc_id VARCHAR(20) PRIMARY KEY,  -- e.g., "ARC-2025-001"
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    pattern_signature VARCHAR(64) NOT NULL,  -- SHA256 hash
    severity VARCHAR(10) CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    first_reported TIMESTAMPTZ NOT NULL,
    last_updated TIMESTAMPTZ NOT NULL,
    incident_count INTEGER DEFAULT 0,
    company_count INTEGER DEFAULT 0,
    avg_downtime_hours FLOAT,
    avg_impact_cost FLOAT,
    mitigation_steps JSONB,  -- Array of mitigation strategies
    affected_patterns JSONB,  -- Graph pattern details
    related_arcs VARCHAR[] DEFAULT '{}',  -- Related ARC IDs
    verified_by VARCHAR(50) DEFAULT 'coderisk',
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'deprecated', 'superseded'))
);

CREATE INDEX idx_arc_pattern ON arc_entries(pattern_signature);
CREATE INDEX idx_arc_severity ON arc_entries(severity);
CREATE INDEX idx_arc_updated ON arc_entries(last_updated DESC);
```

**Example ARC Entry:**
```json
{
    "arc_id": "ARC-2025-001",
    "title": "Auth + User Service Temporal Coupling Without Integration Tests",
    "description": "Authentication service changes without corresponding user service integration tests cause cascade failures due to temporal coupling.",
    "pattern_signature": "sha256:a1b2c3...",
    "severity": "HIGH",
    "first_reported": "2025-03-15T10:00:00Z",
    "last_updated": "2025-10-04T14:30:00Z",
    "incident_count": 47,
    "company_count": 23,
    "avg_downtime_hours": 4.2,
    "avg_impact_cost": 12000,
    "mitigation_steps": [
        {
            "step": 1,
            "action": "Add integration tests for auth + user service interactions",
            "effectiveness": 0.92,
            "implementation_time_hours": 4
        },
        {
            "step": 2,
            "action": "Implement circuit breaker pattern for auth service calls",
            "effectiveness": 0.87,
            "implementation_time_hours": 8
        },
        {
            "step": 3,
            "action": "Use contract testing (Pact, Spring Cloud Contract)",
            "effectiveness": 0.78,
            "implementation_time_hours": 12
        }
    ],
    "affected_patterns": {
        "language": "python",
        "services": ["auth", "user"],
        "coupling_type": "temporal",
        "co_change_rate": 0.85,
        "graph_structure": {
            "nodes": ["auth.py", "user_service.py"],
            "edges": [{"from": "auth.py", "to": "user_service.py", "type": "calls", "frequency": "high"}]
        }
    },
    "related_arcs": ["ARC-2025-034", "ARC-2025-045"],
    "verified_by": "coderisk-v1.0",
    "status": "active"
}
```

### ARC Numbering Convention

**Format:** `ARC-{YEAR}-{SEQUENCE}`

**Examples:**
- `ARC-2025-001` - First architectural risk identified in 2025
- `ARC-2025-034` - 34th risk in 2025
- `ARC-2026-001` - First risk in 2026

**Sequence Management:**
```sql
-- Auto-increment sequence per year
CREATE SEQUENCE arc_2025_seq START 1;
CREATE SEQUENCE arc_2026_seq START 1;

-- Function to generate next ARC ID
CREATE OR REPLACE FUNCTION generate_arc_id(year INT)
RETURNS VARCHAR AS $$
DECLARE
    seq_name VARCHAR := 'arc_' || year || '_seq';
    next_num INT;
BEGIN
    EXECUTE 'SELECT nextval(''' || seq_name || ''')' INTO next_num;
    RETURN 'ARC-' || year || '-' || LPAD(next_num::TEXT, 3, '0');
END;
$$ LANGUAGE plpgsql;
```

### Public API Endpoints

**GET /v1/arc/{arc_id}**
```bash
curl https://api.coderisk.com/v1/arc/ARC-2025-001

# Response:
{
    "arc_id": "ARC-2025-001",
    "title": "Auth + User Service Temporal Coupling...",
    "severity": "HIGH",
    "incident_count": 47,
    "company_count": 23,
    "mitigation_steps": [...],
    "url": "https://coderisk.com/arc/ARC-2025-001"
}
```

**POST /v1/arc/search**
```bash
curl -X POST https://api.coderisk.com/v1/arc/search \
  -H "Content-Type: application/json" \
  -d '{
    "pattern_signature": "sha256:a1b2c3...",
    "similarity_threshold": 0.85
  }'

# Response:
{
    "matches": [
        {
            "arc_id": "ARC-2025-001",
            "similarity": 0.91,
            "title": "Auth + User Service Temporal Coupling...",
            "incident_count": 47
        },
        {
            "arc_id": "ARC-2025-034",
            "similarity": 0.87,
            "title": "Payment + Database Temporal Coupling...",
            "incident_count": 23
        }
    ],
    "total": 2
}
```

---

## Component 2: Causal Graph (Commit → Incident Linkage)

### Graph Schema (Neptune)

**Node Types:**
```gremlin
// Commit node
g.addV('commit')
  .property('sha', 'abc123')
  .property('timestamp', '2025-10-04T14:00:00Z')
  .property('author', 'alice@example.com')
  .property('message', 'Update auth logic')
  .property('files_changed', ['auth.py', 'user_service.py'])
  .property('risk_score', 7.8)

// Deploy node
g.addV('deploy')
  .property('id', 'deploy-456')
  .property('timestamp', '2025-10-04T14:30:00Z')
  .property('environment', 'production')
  .property('service', 'auth-service')

// Incident node
g.addV('incident')
  .property('id', 'INC-2025-1043')
  .property('timestamp', '2025-10-04T15:00:00Z')
  .property('severity', 'high')
  .property('downtime_minutes', 47)
  .property('error_type', 'timeout')
  .property('service', 'auth-service')
```

**Edge Types:**
```gremlin
// Commit → Deploy
g.V().has('commit', 'sha', 'abc123')
  .addE('deployed_in')
  .to(g.V().has('deploy', 'id', 'deploy-456'))
  .property('timestamp', '2025-10-04T14:30:00Z')

// Deploy → Incident
g.V().has('deploy', 'id', 'deploy-456')
  .addE('caused')
  .to(g.V().has('incident', 'id', 'INC-2025-1043'))
  .property('confidence', 0.94)  // Automated detection confidence
  .property('method', 'automatic')  // or 'manual'

// Incident → ARC
g.V().has('incident', 'id', 'INC-2025-1043')
  .addE('matches')
  .to(g.V().has('arc', 'arc_id', 'ARC-2025-001'))
  .property('similarity', 0.91)
```

### Causal Chain Query

**Query: Find all incidents caused by a commit**
```gremlin
g.V().has('commit', 'sha', 'abc123')
  .out('deployed_in')
  .out('caused')
  .hasLabel('incident')
  .valueMap()
```

**Query: Find root cause commit for incident**
```gremlin
g.V().has('incident', 'id', 'INC-2025-1043')
  .in('caused')
  .in('deployed_in')
  .hasLabel('commit')
  .valueMap()
```

**Query: Find similar past incidents**
```gremlin
g.V().has('incident', 'id', 'INC-2025-1043')
  .out('matches')
  .hasLabel('arc')
  .in('matches')
  .hasLabel('incident')
  .where(neq('INC-2025-1043'))
  .order().by('timestamp', desc)
  .limit(10)
  .valueMap()
```

---

## Component 3: Automatic Incident Attribution

### Integration Architecture

**Monitoring Tool Integrations:**
```yaml
# Supported integrations
integrations:
  - datadog:
      api_key: encrypted
      webhooks:
        - incident.detected
        - incident.resolved
      metrics:
        - error_rate
        - latency_p95
        - downtime

  - sentry:
      dsn: encrypted
      webhooks:
        - error.created
        - issue.assigned
      capture:
        - stack_traces
        - error_context

  - pagerduty:
      api_key: encrypted
      webhooks:
        - incident.triggered
        - incident.acknowledged
      correlation:
        - service_map
        - on_call_schedule
```

### Attribution Pipeline

**Step 1: Incident Detection**
```python
# Webhook handler
@app.post("/webhooks/datadog/incident")
async def handle_incident(incident: DatadogIncident):
    """
    Receives incident alert from Datadog
    """
    # Parse incident data
    incident_data = {
        "id": f"INC-{datetime.now().year}-{incident.id}",
        "timestamp": incident.created_at,
        "service": incident.tags.get("service"),
        "error": incident.message,
        "severity": map_severity(incident.priority),
        "duration_minutes": None,  # Updated when resolved
        "source": "datadog"
    }

    # Store in database
    await store_incident(incident_data)

    # Trigger attribution analysis
    await attribute_incident_to_commits.delay(incident_data["id"])

    return {"status": "received"}
```

**Step 2: Deployment Correlation**
```python
async def find_recent_deploys(incident_timestamp, service, lookback_hours=24):
    """
    Find deployments that occurred before incident
    """
    query = """
        SELECT * FROM deploys
        WHERE service = :service
          AND timestamp <= :incident_time
          AND timestamp >= :incident_time - INTERVAL ':lookback hours'
        ORDER BY timestamp DESC
    """

    deploys = await db.fetch_all(
        query,
        service=service,
        incident_time=incident_timestamp,
        lookback=lookback_hours
    )

    return deploys
```

**Step 3: Commit Analysis**
```python
async def analyze_commit_risk(commit_sha):
    """
    Run CodeRisk analysis on commit
    """
    # Fetch commit from GitHub
    commit = await github.get_commit(commit_sha)

    # Run CodeRisk analysis (cached if already analyzed)
    risk_result = await coderisk_check(commit.files_changed)

    return {
        "commit_sha": commit_sha,
        "risk_score": risk_result.score,
        "risk_level": risk_result.level,
        "files_changed": commit.files_changed,
        "patterns_detected": risk_result.patterns
    }
```

**Step 4: Pattern Matching**
```python
async def match_to_arc_patterns(commit_analysis, incident):
    """
    Find similar ARC patterns
    """
    # Extract graph signature from commit
    pattern_signature = hash_graph_structure(commit_analysis["files_changed"])

    # Search ARC database
    arc_matches = await db.fetch_all("""
        SELECT arc_id, title, pattern_signature, incident_count
        FROM arc_entries
        WHERE similarity(pattern_signature, :signature) > 0.85
        ORDER BY similarity(pattern_signature, :signature) DESC
        LIMIT 5
    """, signature=pattern_signature)

    return arc_matches
```

**Step 5: Causal Link Creation**
```python
async def create_causal_link(commit, deploy, incident, arc_matches, confidence):
    """
    Store causal relationship in Neptune graph
    """
    # Create nodes if not exist
    commit_node = await graph.add_vertex("commit", sha=commit.sha, ...)
    deploy_node = await graph.add_vertex("deploy", id=deploy.id, ...)
    incident_node = await graph.add_vertex("incident", id=incident.id, ...)

    # Create edges
    await graph.add_edge(commit_node, "deployed_in", deploy_node)
    await graph.add_edge(deploy_node, "caused", incident_node, confidence=confidence)

    # Link to ARC if strong match
    if arc_matches and arc_matches[0].similarity > 0.85:
        arc_node = await graph.get_vertex("arc", arc_id=arc_matches[0].arc_id)
        await graph.add_edge(incident_node, "matches", arc_node, similarity=arc_matches[0].similarity)

        # Increment ARC incident count
        await db.execute("""
            UPDATE arc_entries
            SET incident_count = incident_count + 1,
                last_updated = NOW()
            WHERE arc_id = :arc_id
        """, arc_id=arc_matches[0].arc_id)
```

### Full Attribution Workflow

```python
@celery.task
async def attribute_incident_to_commits(incident_id):
    """
    Complete incident attribution pipeline
    """
    # 1. Fetch incident
    incident = await db.fetch_one("SELECT * FROM incidents WHERE id = :id", id=incident_id)

    # 2. Find recent deploys (last 24 hours)
    deploys = await find_recent_deploys(incident.timestamp, incident.service, lookback_hours=24)

    if not deploys:
        logger.warning(f"No deploys found for incident {incident_id}")
        return

    # 3. Analyze each deploy's commits
    attribution_results = []
    for deploy in deploys:
        commits = await get_deploy_commits(deploy.id)

        for commit in commits:
            # Run risk analysis
            analysis = await analyze_commit_risk(commit.sha)

            # Match to ARC patterns
            arc_matches = await match_to_arc_patterns(analysis, incident)

            # Calculate attribution confidence
            confidence = calculate_attribution_confidence(
                time_delta=incident.timestamp - deploy.timestamp,
                risk_score=analysis["risk_score"],
                arc_similarity=arc_matches[0].similarity if arc_matches else 0
            )

            attribution_results.append({
                "commit": commit,
                "deploy": deploy,
                "analysis": analysis,
                "arc_matches": arc_matches,
                "confidence": confidence
            })

    # 4. Select most likely root cause (highest confidence)
    root_cause = max(attribution_results, key=lambda x: x["confidence"])

    if root_cause["confidence"] > 0.7:  # High confidence threshold
        # Create causal links in graph
        await create_causal_link(
            root_cause["commit"],
            root_cause["deploy"],
            incident,
            root_cause["arc_matches"],
            root_cause["confidence"]
        )

        # Notify team
        await notify_team(incident_id, root_cause)

    return {
        "incident_id": incident_id,
        "root_cause_commit": root_cause["commit"].sha if root_cause["confidence"] > 0.7 else None,
        "confidence": root_cause["confidence"],
        "arc_matches": root_cause["arc_matches"]
    }
```

---

## Component 4: Privacy-Preserving Pattern Learning (Federated)

### Graph Signature Hashing

**Problem:** Need to match patterns across companies without sharing code.

**Solution:** One-way hash of graph structure

```python
def extract_graph_signature(files_changed, temporal_coupling):
    """
    Extract privacy-preserving pattern signature
    """
    # Build graph structure (no file names, just structure)
    graph_structure = {
        "nodes": len(files_changed),
        "edges": [],
        "coupling_scores": []
    }

    # Add coupling edges (anonymized)
    for (file_a, file_b), coupling_strength in temporal_coupling.items():
        # Use file type, not name
        type_a = get_file_type(file_a)  # e.g., "service", "model", "controller"
        type_b = get_file_type(file_b)

        graph_structure["edges"].append({
            "from_type": type_a,
            "to_type": type_b,
            "coupling": round(coupling_strength, 2)
        })

    # Sort for deterministic hashing
    graph_structure["edges"] = sorted(graph_structure["edges"], key=lambda x: (x["from_type"], x["to_type"]))

    # Hash structure
    structure_json = json.dumps(graph_structure, sort_keys=True)
    signature = hashlib.sha256(structure_json.encode()).hexdigest()

    return signature, graph_structure
```

**Example:**
```python
# Company A (Finance):
files_changed = ["src/auth/login.py", "src/user/profile.py"]
temporal_coupling = {("auth/login.py", "user/profile.py"): 0.87}

signature_a, _ = extract_graph_signature(files_changed, temporal_coupling)
# → "sha256:a1b2c3d4e5f6..."

# Company B (E-commerce):
files_changed = ["app/authentication.py", "app/customer.py"]
temporal_coupling = {("authentication.py", "customer.py"): 0.89}

signature_b, _ = extract_graph_signature(files_changed, temporal_coupling)
# → "sha256:a1b2c3d4e5f6..."  (SAME! Pattern matched despite different code)
```

### Federated Learning Architecture

```
┌─────────────────────────────────────────────────────┐
│              Company A (Private VPC)                │
│                                                     │
│  ┌──────────────┐         ┌─────────────────────┐ │
│  │  CodeRisk    │         │ Pattern Extractor   │ │
│  │  Agent       │ ──────► │ (Local)             │ │
│  │              │         │                     │ │
│  │  Analyzes    │         │ - Graph structure   │ │
│  │  commits     │         │ - One-way hash      │ │
│  │              │         │ - NO code sent      │ │
│  └──────────────┘         └─────────────────────┘ │
│                                     │              │
│                                     ▼              │
│                          ┌──────────────────────┐ │
│                          │ Pattern Signature    │ │
│                          │ sha256:a1b2c3...     │ │
│                          └──────────────────────┘ │
└─────────────────────────────────────────────────┴─┘
                                     │
                                     │ (signature only)
                                     ▼
┌─────────────────────────────────────────────────────┐
│           CodeRisk Central (Cloud)                  │
│                                                     │
│  ┌──────────────────────────────────────────────┐  │
│  │  Pattern Aggregation Database                │  │
│  │                                              │  │
│  │  Signature: sha256:a1b2c3...                │  │
│  │  - Observed at: 23 companies (anonymous)    │  │
│  │  - Incident count: 47                       │  │
│  │  - Avg severity: HIGH                       │  │
│  │  - Success rate (mitigation): 92%          │  │
│  └──────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
                                     │
                                     │ (aggregated insights only)
                                     ▼
┌─────────────────────────────────────────────────────┐
│              Company D (Private VPC)                │
│                                                     │
│  Developer makes similar change                     │
│  → Pattern extracted locally (sha256:a1b2c3...)   │
│  → Sent to CodeRisk Central                        │
│  → Match found: 23 companies, 47 incidents         │
│                                                     │
│  crisk check output:                                │
│  ⚠️ This pattern caused incidents at 23 companies  │
│     (Your code never left your network)            │
└─────────────────────────────────────────────────────┘
```

### Privacy Guarantees

**What Gets Shared:**
```json
{
    "pattern_signature": "sha256:a1b2c3d4e5f6...",
    "language": "python",
    "node_count": 2,
    "coupling_strength": 0.87,
    "incident_occurred": true,
    "severity": "HIGH",
    "mitigation_applied": "integration_tests",
    "mitigation_success": true
}
```

**What NEVER Gets Shared:**
- ❌ Source code
- ❌ File names
- ❌ Company name
- ❌ Developer names
- ❌ Specific error messages
- ❌ Infrastructure details

**Technical Guarantees:**
- One-way hashing (cannot reverse to get code)
- Differential privacy (noise added to counts)
- k-anonymity (require minimum 3 companies before showing pattern)
- Homomorphic encryption (for similarity matching)

---

## Data Models & Schemas

### PostgreSQL Schema (ARC + Incidents)

```sql
-- Complete database schema

-- ARC entries (public)
CREATE TABLE arc_entries (
    -- ... (schema shown earlier)
);

-- Incidents (private, per-company)
CREATE TABLE incidents (
    id VARCHAR(50) PRIMARY KEY,
    company_id VARCHAR(50) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    service VARCHAR(100),
    error_type VARCHAR(100),
    error_message TEXT,
    severity VARCHAR(20),
    downtime_minutes INTEGER,
    impact_cost FLOAT,
    root_cause_commit VARCHAR(64),  -- git SHA
    arc_match VARCHAR(20),  -- ARC ID if matched
    arc_similarity FLOAT,
    status VARCHAR(20) DEFAULT 'open',
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_incidents_company ON incidents(company_id);
CREATE INDEX idx_incidents_timestamp ON incidents(timestamp DESC);
CREATE INDEX idx_incidents_arc ON incidents(arc_match);

-- Pattern signatures (aggregated, privacy-preserving)
CREATE TABLE pattern_signatures (
    signature VARCHAR(64) PRIMARY KEY,
    language VARCHAR(20),
    node_count INTEGER,
    edge_count INTEGER,
    coupling_strength_avg FLOAT,
    incident_count INTEGER DEFAULT 0,
    company_count INTEGER DEFAULT 0,  -- k-anonymity (min 3)
    severity_distribution JSONB,
    first_seen TIMESTAMPTZ NOT NULL,
    last_seen TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_patterns_incident_count ON pattern_signatures(incident_count DESC);

-- Company participation (opt-in tracking)
CREATE TABLE company_pattern_sharing (
    company_id VARCHAR(50) PRIMARY KEY,
    opted_in BOOLEAN DEFAULT false,
    patterns_shared INTEGER DEFAULT 0,
    incidents_contributed INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### Neptune Graph Schema

```gremlin
// Vertex labels
g.addV('commit')
g.addV('deploy')
g.addV('incident')
g.addV('arc')
g.addV('pattern')
g.addV('file')
g.addV('function')

// Edge labels
g.addE('deployed_in')  // commit → deploy
g.addE('caused')       // deploy → incident
g.addE('matches')      // incident → arc
g.addE('has_pattern')  // commit → pattern
g.addE('modifies')     // commit → file
g.addE('calls')        // function → function
g.addE('couples_with') // file → file (temporal)
```

---

## API Specifications

### Public API (External)

**Endpoint:** `https://api.coderisk.com/v1/`

```yaml
openapi: 3.0.0
info:
  title: CodeRisk Incident Knowledge Graph API
  version: 1.0.0

paths:
  /arc/{arc_id}:
    get:
      summary: Get ARC entry by ID
      parameters:
        - name: arc_id
          in: path
          required: true
          schema:
            type: string
            example: "ARC-2025-001"
      responses:
        '200':
          description: ARC entry found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ARCEntry'

  /arc/search:
    post:
      summary: Search ARC database by pattern
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                pattern_signature:
                  type: string
                  example: "sha256:a1b2c3..."
                similarity_threshold:
                  type: number
                  minimum: 0
                  maximum: 1
                  example: 0.85
      responses:
        '200':
          description: Matching ARCs found
          content:
            application/json:
              schema:
                type: object
                properties:
                  matches:
                    type: array
                    items:
                      $ref: '#/components/schemas/ARCMatch'

  /incidents/submit:
    post:
      summary: Submit incident (authenticated)
      security:
        - ApiKeyAuth: []
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/IncidentSubmission'
      responses:
        '201':
          description: Incident created
          content:
            application/json:
              schema:
                type: object
                properties:
                  incident_id:
                    type: string
                  arc_match:
                    $ref: '#/components/schemas/ARCMatch'

components:
  schemas:
    ARCEntry:
      type: object
      properties:
        arc_id:
          type: string
        title:
          type: string
        description:
          type: string
        severity:
          type: string
          enum: [LOW, MEDIUM, HIGH, CRITICAL]
        incident_count:
          type: integer
        company_count:
          type: integer
        mitigation_steps:
          type: array
          items:
            type: object

  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      in: header
      name: X-API-Key
```

---

## Implementation Plan

### Phase 1: Foundation (Q1 2026)

**Week 1-2: Database Setup**
- ✅ PostgreSQL schema deployment
- ✅ Neptune graph database provisioning
- ✅ First 20 ARC entries (manually curated)

**Week 3-4: Monitoring Integrations**
- ✅ Datadog webhook integration
- ✅ Sentry webhook integration
- ✅ PagerDuty webhook integration

**Week 5-6: Attribution Pipeline**
- ✅ Incident detection webhook handlers
- ✅ Deployment correlation logic
- ✅ Commit analysis integration
- ✅ Causal graph construction

**Week 7-8: Public API**
- ✅ ARC search endpoint
- ✅ Incident submission endpoint
- ✅ Pattern matching endpoint

**Week 9-10: CLI Integration**
- ✅ `crisk check` shows ARC matches
- ✅ `crisk incident link` manual attribution
- ✅ Pre-commit hook updates

**Week 11-12: Initial Data Collection**
- ✅ Partner with 10 friendly companies
- ✅ Collect first 1,000 incidents
- ✅ Curate first 100 ARC entries

**Q1 Success Criteria:**
- ✅ 100 ARC entries published
- ✅ 1,000 incidents attributed to commits
- ✅ 10 companies contributing data
- ✅ 80% attribution accuracy

---

### Phase 2: Federated Learning (Q2 2026)

**Week 1-2: Pattern Extraction**
- ✅ Graph signature hashing implementation
- ✅ Privacy-preserving pattern extraction
- ✅ Differential privacy implementation

**Week 3-4: Pattern Aggregation**
- ✅ Central pattern database
- ✅ Similarity matching algorithm
- ✅ k-anonymity enforcement (min 3 companies)

**Week 5-6: Federated Deployment**
- ✅ On-device pattern extraction (in customer VPC)
- ✅ Secure pattern submission (encrypted channel)
- ✅ Pattern matching API

**Week 7-8: CLI Integration**
- ✅ Federated learning opt-in
- ✅ `crisk check` uses cross-org patterns
- ✅ "X companies observed this pattern" messaging

**Week 9-10: Privacy Audits**
- ✅ Security audit (third-party)
- ✅ Privacy review (legal)
- ✅ Compliance certification (GDPR, SOC2)

**Week 11-12: Scale to 100 Companies**
- ✅ Onboard 90 additional companies
- ✅ Collect 5,000 more patterns
- ✅ Improve prediction accuracy to 85%

**Q2 Success Criteria:**
- ✅ 100 companies opted in to federated learning
- ✅ 10,000 incidents in knowledge graph
- ✅ 500 unique pattern signatures
- ✅ 85% prediction accuracy
- ✅ Zero privacy incidents

---

## Success Metrics

### Data Quality Metrics

**ARC Database:**
- Total ARC entries: Target 100 (Q1), 500 (Q2)
- Incident coverage: % of incidents matching ARC patterns (target: 80%)
- ARC accuracy: % of ARC patterns that reliably predict incidents (target: 90%)

**Causal Graph:**
- Total incidents attributed: Target 10,000 (Q2)
- Attribution confidence: Avg confidence score (target: 0.85)
- False attribution rate: % of incorrect attributions (target: <5%)

**Pattern Learning:**
- Total patterns identified: Target 500 (Q2)
- Cross-org pattern matches: Target 10,000 (Q2)
- Prediction improvement: Accuracy gain from federated learning (target: +15%)

### Business Impact Metrics

**Network Effects:**
- Companies contributing data: Target 100 (Q2)
- Incidents prevented: # of incidents caught before production (target: 5,000)
- Value created: Estimated $ saved from prevented incidents (target: $10M)

**API Usage:**
- ARC API queries: Target 100K/month (Q2)
- Pattern matching requests: Target 50K/month (Q2)
- Public ARC page views: Target 500K/month (Q2)

---

## Security & Privacy Considerations

### Data Classification

**Public Data (ARC Database):**
- ✅ Pattern signatures (hashed)
- ✅ Aggregated statistics (>3 companies)
- ✅ Mitigation strategies
- ✅ Incident counts (anonymized)

**Private Data (Never Shared):**
- ❌ Source code
- ❌ File names
- ❌ Company names
- ❌ Developer identities
- ❌ Infrastructure details

### Compliance

**GDPR:**
- Right to erasure: Companies can delete all contributed patterns
- Data minimization: Only collect pattern signatures, not code
- Consent: Explicit opt-in for pattern sharing

**SOC2:**
- Access controls: RBAC for ARC database
- Encryption: At-rest (AES-256), in-transit (TLS 1.3)
- Audit logs: All API access logged

**HIPAA (for healthcare customers):**
- No PHI in patterns (architecture only)
- BAA available for enterprise
- Self-hosted option (customer VPC)

---

## Related Documents

**Product:**
- [strategic_moats.md](../00-product/strategic_moats.md) - Business strategy (Cornered Resource)
- [vision_and_mission.md](../00-product/vision_and_mission.md) - Trust infrastructure vision

**Architecture:**
- [graph_ontology.md](graph_ontology.md) - Graph schema details
- [trust_infrastructure.md](trust_infrastructure.md) - Trust certificates (next component)
- [cloud_deployment.md](cloud_deployment.md) - Infrastructure requirements

**Implementation:**
- [phases/phase-cornered-resource.md](../03-implementation/phases/phase-cornered-resource.md) - Q1 2026 roadmap

---

**Last Updated:** October 4, 2025
**Next Review:** January 2026 (post Q1 implementation)
