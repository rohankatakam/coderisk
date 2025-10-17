# Reality Gap Analysis: What We Have vs What We Claim + GitHub Mining Strategy

**Created:** October 10, 2025
**Status:** Critical Analysis - Implementation Roadmap
**Owner:** Product + Engineering Team

> **Question:** How do we actually implement our unique features? Do we mine GitHub repos and use LLMs to build training data?

---

## Executive Summary

**REALITY CHECK:** There's a **massive gap** between what we **claim as unique features** and what we've **actually implemented**.

### What We Have (Production-Ready ✅)

| Feature | Status | Implementation |
|---------|--------|----------------|
| **Basic incident tracking** | ✅ Local only | PostgreSQL schema + CRUD operations |
| **Incident linking to files** | ✅ Local only | Manual `crisk incident link` command |
| **Incident search (BM25)** | ✅ Local only | Full-text search in PostgreSQL |
| **Temporal coupling (git history)** | ✅ Working | Parses local git history, creates CO_CHANGED edges |
| **Graph construction (local repo)** | ✅ Working | Tree-sitter AST + git analysis → Neo4j |
| **Phase 2 LLM investigation** | ✅ Working | OpenAI integration, hop-by-hop graph search |

### What We DON'T Have (Pure Vision 🔮)

| Feature | Status | Gap |
|---------|--------|-----|
| **ARC Database (10K+ incidents)** | ❌ Design only | We have 0 public ARC entries, 0 cross-org data |
| **Cross-organization learning** | ❌ Design only | No federated learning, no privacy-preserving hashing |
| **Incident → Commit → Pattern linking** | ❌ Design only | No causal graph, no automatic attribution |
| **Cloud infrastructure** | ❌ Design only | No Neptune, no settings portal, no GitHub OAuth |
| **CVE integration (NVD)** | ❌ Design only | No NVD API client, no dependency scanning |
| **Public ARC API** | ❌ Design only | No `/v1/arc/{arc_id}` endpoints |

**Bottom Line:** We have a **local-only tool** that works on **single repositories**. Everything about "23 companies' learnings," "10K+ incidents," and "cross-org intelligence" is **aspirational**.

---

## Part 1: The Reality Gap (What's Real vs What's Vision)

### 1.1. Incident Knowledge Graph Status

**What We Claim:**
```
"10,000+ incidents linked to commits"
"CVE-like catalog for architecture (ARC-2025-XXX)"
"Cross-organization pattern learning"
"Privacy-preserving federated learning"
```

**Reality:**
```
What we have:
✅ PostgreSQL schema for incidents (tables exist)
✅ CRUD operations (create, read, update, delete)
✅ Manual incident linking (crisk incident link <id> <file>)
✅ BM25 full-text search (search incidents by keywords)
✅ Local-only (single repo, single user)

What we DON'T have:
❌ No public ARC entries (0 incidents cataloged)
❌ No cross-org data collection
❌ No federated learning
❌ No automatic incident attribution
❌ No causal graph (commit → deploy → incident)
❌ No cloud infrastructure
❌ No API for public access
```

**Implementation Gap:** **95%** - We have the local data model but none of the infrastructure, data collection, or cross-org features.

---

### 1.2. Cross-Organization Learning Status

**What We Claim:**
```
"Learn from 100+ companies (privacy-preserving)"
"Federated learning (no code leaves VPC)"
"Graph signature hashing (one-way)"
"'This pattern observed at 23 companies'"
```

**Reality:**
```
What we have:
✅ Nothing - This is 100% design documentation

What we DON'T have:
❌ No pattern extraction algorithm
❌ No graph signature hashing
❌ No privacy-preserving techniques
❌ No federated learning infrastructure
❌ No multi-tenant data model
❌ No company onboarding process
❌ No data sharing agreements
```

**Implementation Gap:** **100%** - Completely unimplemented, pure vision.

---

### 1.3. CVE + ARC Combined Analysis Status

**What We Claim:**
```
"CVE vulnerabilities (from NVD)"
"+ Architectural patterns (from ARC)"
"= Combined risk score with 3x multiplier"
```

**Reality:**
```
What we have:
✅ Nothing for CVE integration

What we DON'T have:
❌ No NVD API client
❌ No CVE database
❌ No dependency parsing (requirements.txt, package.json)
❌ No CVE → code mapping
❌ No combined risk scoring
```

**Implementation Gap:** **100%** - Completely unimplemented, pure design.

---

### 1.4. Temporal Coupling Status (PARTIALLY WORKING ✅)

**What We Claim:**
```
"Detects files that change together (85% co-change rate)"
"Real-time check before commit (not dashboard)"
"Integrated with git diff"
```

**Reality:**
```
What we have:
✅ Git history parsing (internal/temporal/)
✅ Co-change detection (CO_CHANGED edges in Neo4j)
✅ Frequency calculation (how often files change together)
✅ Real-time pre-commit integration (crisk check)
✅ Working end-to-end (tested with omnara repo)

What we DON'T have:
❌ No co-change rate display in output (hidden from user)
❌ No "You changed X but not Y" warnings (not surfaced)
❌ Limited git history depth (performance concerns)
```

**Implementation Gap:** **30%** - Core algorithm works, but UX and surfacing needs work.

---

### 1.5. Agentic Graph Search Status (WORKING ✅)

**What We Claim:**
```
"LLM-guided hop-by-hop graph traversal"
"Intelligent metric selection (not brute force)"
"Evidence-based reasoning"
```

**Reality:**
```
What we have:
✅ Phase 2 LLM investigation (internal/agent/)
✅ Hop-by-hop graph navigation (investigator.go)
✅ OpenAI integration (user provides API key)
✅ Evidence collection (temporal + incidents + graph)
✅ Synthesis with LLM reasoning
✅ Working end-to-end (tested Oct 10)

What we DON'T have:
❌ Only works with local Neo4j (not Neptune)
❌ No cloud deployment
❌ No metric validation (all metrics used, not selective)
❌ No false positive tracking
```

**Implementation Gap:** **20%** - Core works locally, needs cloud + validation.

---

## Part 2: GitHub Mining Strategy (Bootstrap the ARC Database)

### 2.1. The Cold Start Problem

**Problem:**
- We claim "10,000+ incidents from 23 companies"
- Reality: We have 0 incidents, 0 companies
- Chicken-and-egg: Can't demonstrate value without data, can't get companies without value

**Solution: Mine GitHub to Bootstrap**
```
Step 1: Mine public repos for incident patterns
Step 2: Use LLMs to extract + structure data
Step 3: Build initial ARC database (100+ entries)
Step 4: Launch with "pre-trained" knowledge
Step 5: Get companies to contribute (network effects)
```

---

### 2.2. What We Can Mine from GitHub

#### 2.2.1. Public Incident Data Sources

**GitHub Issues (Labeled as bugs/incidents):**
```
Query: label:bug OR label:incident OR label:production
Example repos:
- react (Meta)
- kubernetes (CNCF)
- tensorflow (Google)
- django (Django Software Foundation)
- rails (Ruby on Rails)
```

**What we can extract:**
- Issue title + description
- Files mentioned in issue body
- Commits linked to issue (via "Fixes #123")
- Labels (bug, critical, production, outage)
- Timestamps (when reported, when resolved)

**GitHub API:**
```bash
# Get issues labeled as bugs
GET /repos/{owner}/{repo}/issues?labels=bug,incident&state=closed

# Get commits linked to issue
GET /repos/{owner}/{repo}/issues/{issue_number}/events
# Filter for "closed" event with commit_id
```

---

#### 2.2.2. Commit Messages (Bug Fixes)

**Patterns to detect incidents:**
```regex
# Bug fix patterns
"fix|bug|crash|error|failure|broke|broken|regress"

# Production incident patterns
"hotfix|rollback|revert|urgent|critical|outage|down"

# Examples:
"fix: critical bug in payment processing"
"hotfix: auth service crash on startup"
"revert: rollback breaking change in user service"
```

**What we can extract:**
- Commit SHA
- Commit message (incident description)
- Files changed (affected files)
- Timestamp
- Author

**GitHub API:**
```bash
# Search commits
GET /search/commits?q=repo:{owner}/{repo}+fix+bug+crash

# Get commit details
GET /repos/{owner}/{repo}/commits/{sha}
```

---

#### 2.2.3. Pull Request Descriptions

**Incident indicators:**
```
PR title: "Fix critical payment timeout"
PR body: "This fixes the production incident where..."
PR labels: hotfix, urgent, production
```

**What we can extract:**
- PR title + description (incident context)
- Files changed
- Linked issues (via "Closes #456")
- Review comments (root cause discussion)

---

#### 2.2.4. Git History (Temporal Coupling)

**What we can mine:**
- Files that change together (co-change patterns)
- Change frequency
- Authors
- Time windows

**Already implemented:** ✅ We have this in `internal/temporal/`

---

### 2.3. LLM Processing Pipeline

**Step 1: Data Extraction**
```python
# Pseudocode for mining GitHub

for repo in TOP_1000_REPOS:
    # 1. Get issues labeled as bugs
    issues = github.get_issues(repo, labels=["bug", "incident"])

    # 2. Get commits with bug fix keywords
    commits = github.search_commits(repo, query="fix bug crash")

    # 3. Get hotfix PRs
    prs = github.get_prs(repo, labels=["hotfix", "urgent"])

    # Store raw data
    database.store_raw_incident(repo, issues, commits, prs)
```

**Step 2: LLM Structuring**
```python
# Use LLM to structure incident data

def extract_incident_pattern(raw_incident, llm):
    """
    Use LLM to extract structured incident pattern
    """
    prompt = f"""
    Analyze this GitHub incident and extract:
    1. Root cause (architectural issue)
    2. Affected files
    3. Pattern type (temporal coupling, missing tests, etc.)
    4. Severity
    5. Mitigation steps

    GitHub Issue:
    Title: {raw_incident.title}
    Description: {raw_incident.description}
    Files changed: {raw_incident.files}
    Commits: {raw_incident.commits}

    Output JSON format:
    {{
      "pattern_type": "temporal_coupling" | "missing_tests" | "api_breaking_change",
      "severity": "critical" | "high" | "medium" | "low",
      "root_cause": "...",
      "affected_files": ["file1.py", "file2.py"],
      "coupling_files": ["file_a.py", "file_b.py"],
      "mitigation": "...",
      "similar_to": "ARC-2025-001 (if similar pattern exists)"
    }}
    """

    response = llm.complete(prompt)
    return json.loads(response)
```

**Step 3: Pattern Deduplication**
```python
# Deduplicate similar incidents into ARC entries

def create_arc_entry(incidents, llm):
    """
    Cluster similar incidents into single ARC entry
    """
    # Group by pattern_type and affected files
    clusters = cluster_incidents(incidents, similarity_threshold=0.85)

    for cluster in clusters:
        # Use LLM to synthesize ARC entry
        prompt = f"""
        Create an ARC (Architectural Risk Catalog) entry by synthesizing these {len(cluster)} similar incidents:

        {json.dumps(cluster, indent=2)}

        Output format:
        {{
          "arc_id": "ARC-2025-XXX",
          "title": "Short descriptive title",
          "description": "Detailed pattern description",
          "pattern_signature": "sha256:...",
          "severity": "HIGH",
          "incident_count": {len(cluster)},
          "affected_repos": ["react", "vue", "angular"],
          "mitigation_steps": [...]
        }}
        """

        arc_entry = llm.complete(prompt)
        database.store_arc_entry(arc_entry)
```

---

### 2.4. Implementation Roadmap (Bootstrap Phase)

#### Phase 1: Data Collection (Week 1-2)

**Goal:** Collect 10,000+ raw incidents from GitHub

**Steps:**
1. **Select target repositories** (top 1000 by stars)
   ```python
   # GitHub search API
   repos = github.search_repos(
       query="stars:>10000 language:python OR language:javascript",
       limit=1000
   )
   ```

2. **Mine incident data**
   - Issues with labels: `bug`, `incident`, `production`, `critical`
   - Commits with keywords: `fix`, `bug`, `crash`, `hotfix`
   - PRs with labels: `hotfix`, `urgent`, `rollback`

3. **Store raw data in PostgreSQL**
   ```sql
   CREATE TABLE raw_github_incidents (
       id SERIAL PRIMARY KEY,
       repo_owner TEXT NOT NULL,
       repo_name TEXT NOT NULL,
       source_type TEXT NOT NULL, -- 'issue', 'commit', 'pr'
       source_id TEXT NOT NULL,   -- issue #, commit SHA, PR #
       title TEXT,
       description TEXT,
       labels TEXT[],
       files_changed TEXT[],
       created_at TIMESTAMPTZ,
       resolved_at TIMESTAMPTZ,
       raw_json JSONB
   );
   ```

4. **Rate limit management**
   - GitHub API: 5,000 requests/hour (authenticated)
   - Use multiple API keys (rotate)
   - Cache responses (avoid re-fetching)

**Deliverable:** 10,000+ raw incidents in database

**Effort:** 1 week (automated script + monitoring)

---

#### Phase 2: LLM Processing (Week 3-4)

**Goal:** Extract structured patterns from raw incidents

**Steps:**
1. **Batch processing with LLM**
   ```python
   # Process in batches of 100
   for batch in batches(raw_incidents, size=100):
       structured = llm.batch_extract_patterns(batch)
       database.store_structured_incidents(structured)
   ```

2. **LLM prompt optimization**
   - Test with different prompts
   - Validate output quality
   - Fine-tune for incident extraction

3. **Pattern classification**
   - Temporal coupling
   - Missing tests
   - API breaking changes
   - Database coupling
   - Auth/Payment coupling
   - etc.

4. **Cost management**
   - OpenAI GPT-4: $0.01/1K tokens
   - Average prompt: 1K tokens input + 500 tokens output = $0.015/incident
   - 10,000 incidents × $0.015 = **$150 total**

**Deliverable:** 10,000 structured incident patterns

**Effort:** 1 week (LLM processing + validation)

---

#### Phase 3: ARC Catalog Creation (Week 5-6)

**Goal:** Deduplicate into 100-500 ARC entries

**Steps:**
1. **Clustering similar incidents**
   ```python
   # Use embeddings for similarity
   embeddings = openai.embeddings(
       [inc.description for inc in structured_incidents]
   )

   clusters = cluster_by_similarity(
       embeddings,
       threshold=0.85,
       method="hierarchical"
   )
   ```

2. **LLM synthesis into ARC entries**
   - For each cluster, synthesize single ARC entry
   - Include: title, description, mitigation, incident_count

3. **Manual review (quality control)**
   - Review top 100 ARC entries
   - Fix incorrect classifications
   - Merge duplicates

4. **ARC numbering**
   ```
   ARC-2025-001: Auth + User Service Temporal Coupling
   ARC-2025-002: Payment + Database Coupling Without Validation
   ARC-2025-003: API Breaking Changes Without Version Bump
   ...
   ARC-2025-100: Missing Circuit Breaker in External API Calls
   ```

**Deliverable:** 100 public ARC entries (validated)

**Effort:** 2 weeks (clustering + review + publishing)

---

#### Phase 4: Public Launch (Week 7-8)

**Goal:** Launch ARC database publicly

**Steps:**
1. **Create public website**
   - `https://coderisk.com/arc`
   - Browse all ARC entries
   - Search by keyword
   - View incident counts

2. **Public API**
   ```bash
   # Get ARC entry
   GET https://api.coderisk.com/v1/arc/ARC-2025-001

   # Search ARCs
   POST https://api.coderisk.com/v1/arc/search
   {
     "query": "temporal coupling",
     "severity": "HIGH"
   }
   ```

3. **Marketing & PR**
   - Blog post: "Introducing ARC: CVE for Architecture"
   - HackerNews launch
   - Tweet storm
   - Email to beta users

4. **Integration into `crisk check`**
   ```bash
   crisk check auth.py

   ⚠️  HIGH risk detected:

   Pattern matches known architectural risks:
     - ARC-2025-001: Auth + User Service Coupling (47 incidents from 23 repos)

   Your change is 91% similar to ARC-2025-001
   Historical outcome: 89% incident rate within 7 days
   ```

**Deliverable:** Public ARC database live

**Effort:** 2 weeks (website + API + marketing)

---

### 2.5. GitHub Mining - Technical Implementation

#### 2.5.1. Data Collection Script

**`scripts/mine_github_incidents.py`:**
```python
#!/usr/bin/env python3

import os
import json
import time
from github import Github
from sqlalchemy import create_engine

# GitHub API client
g = Github(os.environ["GITHUB_TOKEN"])

# Database connection
engine = create_engine(os.environ["DATABASE_URL"])

# Target repositories (top 1000 by stars)
TARGET_REPOS = [
    "facebook/react",
    "kubernetes/kubernetes",
    "tensorflow/tensorflow",
    "django/django",
    "rails/rails",
    # ... (1000 repos)
]

def mine_repo_incidents(repo_name):
    """
    Mine incidents from a single repository
    """
    repo = g.get_repo(repo_name)

    # 1. Get issues labeled as bugs
    issues = repo.get_issues(
        state="closed",
        labels=["bug", "incident", "production"]
    )

    for issue in issues:
        # Extract incident data
        incident = {
            "repo_owner": repo.owner.login,
            "repo_name": repo.name,
            "source_type": "issue",
            "source_id": str(issue.number),
            "title": issue.title,
            "description": issue.body,
            "labels": [l.name for l in issue.labels],
            "created_at": issue.created_at,
            "closed_at": issue.closed_at,
            "raw_json": {
                "url": issue.html_url,
                "comments": issue.comments,
                "reactions": issue.get_reactions().totalCount
            }
        }

        # Get files mentioned in issue
        # (parse issue body for file references)
        incident["files_changed"] = extract_file_references(issue.body)

        # Get linked commits (via "Fixes #123")
        events = issue.get_events()
        linked_commits = [
            e.commit_id for e in events
            if e.event == "closed" and e.commit_id
        ]
        incident["raw_json"]["linked_commits"] = linked_commits

        # Store in database
        store_incident(engine, incident)

    # 2. Get commits with bug fix keywords
    commits = repo.get_commits()
    for commit in commits:
        if matches_bug_pattern(commit.commit.message):
            incident = {
                "repo_owner": repo.owner.login,
                "repo_name": repo.name,
                "source_type": "commit",
                "source_id": commit.sha,
                "title": commit.commit.message.split("\n")[0],
                "description": commit.commit.message,
                "files_changed": [f.filename for f in commit.files],
                "created_at": commit.commit.author.date,
                "raw_json": {
                    "url": commit.html_url,
                    "additions": commit.stats.additions,
                    "deletions": commit.stats.deletions
                }
            }

            store_incident(engine, incident)

    # 3. Get hotfix PRs
    prs = repo.get_pulls(
        state="closed",
        sort="updated",
        direction="desc"
    )

    for pr in prs:
        if has_hotfix_label(pr) or matches_hotfix_title(pr.title):
            incident = {
                "repo_owner": repo.owner.login,
                "repo_name": repo.name,
                "source_type": "pr",
                "source_id": str(pr.number),
                "title": pr.title,
                "description": pr.body,
                "labels": [l.name for l in pr.labels],
                "files_changed": [f.filename for f in pr.get_files()],
                "created_at": pr.created_at,
                "closed_at": pr.closed_at,
                "raw_json": {
                    "url": pr.html_url,
                    "merged": pr.merged,
                    "merge_commit_sha": pr.merge_commit_sha
                }
            }

            store_incident(engine, incident)

def matches_bug_pattern(message):
    """Check if commit message indicates bug fix"""
    keywords = ["fix", "bug", "crash", "error", "failure", "hotfix", "rollback"]
    return any(kw in message.lower() for kw in keywords)

def extract_file_references(text):
    """Extract file paths mentioned in text"""
    import re
    # Match patterns like: src/file.py, lib/utils.js
    pattern = r'[\w/.-]+\.\w+'
    matches = re.findall(pattern, text)
    return [m for m in matches if "/" in m]  # Only file paths

def store_incident(engine, incident):
    """Store incident in PostgreSQL"""
    with engine.connect() as conn:
        conn.execute("""
            INSERT INTO raw_github_incidents
            (repo_owner, repo_name, source_type, source_id, title, description,
             labels, files_changed, created_at, resolved_at, raw_json)
            VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
        """, (
            incident["repo_owner"],
            incident["repo_name"],
            incident["source_type"],
            incident["source_id"],
            incident["title"],
            incident["description"],
            incident.get("labels", []),
            incident.get("files_changed", []),
            incident["created_at"],
            incident.get("closed_at"),
            json.dumps(incident["raw_json"])
        ))

# Main execution
if __name__ == "__main__":
    for repo_name in TARGET_REPOS:
        print(f"Mining {repo_name}...")
        try:
            mine_repo_incidents(repo_name)
            print(f"✓ {repo_name} complete")
        except Exception as e:
            print(f"✗ {repo_name} failed: {e}")

        # Rate limit management
        time.sleep(1)  # 1 second between repos
```

**Run script:**
```bash
export GITHUB_TOKEN="ghp_xxxxx"
export DATABASE_URL="postgresql://localhost/coderisk"

python3 scripts/mine_github_incidents.py
```

---

#### 2.5.2. LLM Pattern Extraction

**`scripts/extract_patterns.py`:**
```python
#!/usr/bin/env python3

import os
import json
from openai import OpenAI
from sqlalchemy import create_engine

client = OpenAI(api_key=os.environ["OPENAI_API_KEY"])
engine = create_engine(os.environ["DATABASE_URL"])

EXTRACTION_PROMPT = """
Analyze this GitHub incident and extract a structured architectural risk pattern.

GitHub Incident:
Title: {title}
Description: {description}
Repository: {repo}
Files Changed: {files}
Labels: {labels}

Extract:
1. Pattern type (temporal_coupling, missing_tests, api_breaking_change, database_coupling, auth_coupling, etc.)
2. Severity (critical, high, medium, low)
3. Root cause (architectural issue, not specific bug)
4. Affected file types (e.g., "auth service + user service")
5. Coupling files (if temporal coupling pattern)
6. Mitigation steps

Output JSON only:
{{
  "pattern_type": "...",
  "severity": "...",
  "root_cause": "...",
  "affected_file_types": [...],
  "coupling_files": [...],
  "mitigation": "..."
}}
"""

def extract_pattern(incident):
    """
    Use LLM to extract structured pattern from incident
    """
    prompt = EXTRACTION_PROMPT.format(
        title=incident["title"],
        description=incident["description"][:1000],  # Truncate long descriptions
        repo=f"{incident['repo_owner']}/{incident['repo_name']}",
        files=", ".join(incident.get("files_changed", [])),
        labels=", ".join(incident.get("labels", []))
    )

    response = client.chat.completions.create(
        model="gpt-4",
        messages=[{"role": "user", "content": prompt}],
        temperature=0.2,  # Lower for more consistent extraction
        response_format={"type": "json_object"}
    )

    pattern = json.loads(response.choices[0].message.content)
    return pattern

def process_incidents():
    """
    Process all raw incidents and extract patterns
    """
    with engine.connect() as conn:
        # Get unprocessed incidents
        result = conn.execute("""
            SELECT * FROM raw_github_incidents
            WHERE id NOT IN (SELECT incident_id FROM structured_patterns)
            LIMIT 100
        """)

        for row in result:
            incident = dict(row)

            try:
                # Extract pattern
                pattern = extract_pattern(incident)

                # Store structured pattern
                conn.execute("""
                    INSERT INTO structured_patterns
                    (incident_id, pattern_type, severity, root_cause,
                     affected_file_types, coupling_files, mitigation, extracted_at)
                    VALUES (%s, %s, %s, %s, %s, %s, %s, NOW())
                """, (
                    incident["id"],
                    pattern["pattern_type"],
                    pattern["severity"],
                    pattern["root_cause"],
                    pattern.get("affected_file_types", []),
                    pattern.get("coupling_files", []),
                    pattern["mitigation"]
                ))

                print(f"✓ Processed incident {incident['id']}")

            except Exception as e:
                print(f"✗ Failed incident {incident['id']}: {e}")

if __name__ == "__main__":
    process_incidents()
```

---

### 2.6. Cost Analysis

**GitHub API:**
- Free: 5,000 requests/hour (authenticated)
- Cost: $0 (use free tier)

**Data Storage:**
- PostgreSQL: ~100MB for 10K incidents
- Cost: ~$10/month (AWS RDS)

**LLM Processing:**
- OpenAI GPT-4: $0.01/1K tokens input, $0.03/1K tokens output
- Average: 1K input + 500 output = $0.025/incident
- 10,000 incidents × $0.025 = **$250 total** (one-time)

**Total Bootstrap Cost:** **$260** (one-time) + $10/month (storage)

---

## Part 3: Immediate Action Plan

### 3.1. What to Build First (Priority Order)

**P0: Bootstrap ARC Database (8 weeks)**
1. Week 1-2: GitHub mining script (collect 10K incidents)
2. Week 3-4: LLM pattern extraction (structure 10K incidents)
3. Week 5-6: Clustering + ARC catalog creation (100 entries)
4. Week 7-8: Public website + API + marketing

**P1: CVE Integration (2 weeks)** - See [nvd_integration_analysis.md](nvd_integration_analysis.md)
1. Week 1: NVD API client + dependency scanning
2. Week 2: Combined CVE + ARC risk scoring

**P2: Cloud Infrastructure (4 weeks)**
1. Week 1: Neptune Serverless setup
2. Week 2: Settings portal (API key config)
3. Week 3: GitHub OAuth authentication
4. Week 4: Multi-tenant data model

**P3: Cross-Org Learning (8 weeks)**
1. Week 1-2: Pattern extraction (graph signatures)
2. Week 3-4: Federated learning (privacy-preserving)
3. Week 5-6: Multi-company onboarding
4. Week 7-8: Cross-org intelligence display

**Total Timeline:** 22 weeks (5.5 months) to reach claimed capabilities

---

### 3.2. Quick Wins (Demonstrate Value Now)

**Week 1: Demo ARC Database (Fake it till you make it)**
```
Goal: Show "ARC-2025-001" in crisk check output

Implementation:
1. Manually create 10 ARC entries (hand-curated)
2. Store in PostgreSQL
3. Add search function to crisk check
4. Display: "This matches ARC-2025-001 (47 incidents from react, vue, angular)"

Effort: 1 week
Value: Demonstrates concept, gets user feedback
```

**Week 2: GitHub Incident Mining PoC**
```
Goal: Collect 1,000 incidents from top 10 repos

Implementation:
1. Write mining script (scripts/mine_github_incidents.py)
2. Target: react, kubernetes, django, rails, vue, angular, tensorflow, pytorch, redis, postgres
3. Store raw data in PostgreSQL

Effort: 1 week
Value: Proves data collection feasibility
```

**Week 3: LLM Pattern Extraction PoC**
```
Goal: Extract 100 structured patterns

Implementation:
1. Write extraction script (scripts/extract_patterns.py)
2. Process 100 incidents with GPT-4
3. Review output quality

Effort: 1 week
Value: Validates LLM approach, estimates costs
```

**Week 4: Public Launch (MVP)**
```
Goal: Launch ARC database with 100 entries

Implementation:
1. Create simple website (Next.js)
2. Display 100 ARC entries
3. Add search + browse
4. Integrate into crisk check

Effort: 1 week
Value: First public milestone, marketing opportunity
```

---

## Part 4: Strategic Recommendations

### 4.1. Be Honest About Status

**Update marketing materials:**
```
Before: "CodeRisk learns from 10,000+ incidents across 23 companies"

After (honest): "CodeRisk is building the first public Architectural Risk Catalog (ARC),
starting with 100+ patterns mined from popular open-source projects"
```

**Update README:**
```
Status: MVP - Local-only analysis
Roadmap: Cloud infrastructure (Q1 2026), Cross-org learning (Q2 2026)
```

### 4.2. Focus on Quick Wins

**What works TODAY:**
- ✅ Temporal coupling (real, tested, working)
- ✅ Incident tracking (local, manual)
- ✅ LLM investigation (Phase 2)
- ✅ Pre-commit checks

**What to market:**
- "Real-time temporal coupling detection"
- "AI-powered architectural risk analysis"
- "Pre-commit safety checks"
- "Local incident tracking"

**What NOT to claim yet:**
- ❌ "10,000+ incidents" (we have 0)
- ❌ "23 companies" (we have 0)
- ❌ "Cross-org learning" (not implemented)

### 4.3. Bootstrap Strategy

**Phase 1: Solo Launch (Today)**
- Market what works: temporal coupling + LLM investigation
- Honest about status: "MVP, local-only"
- Focus on individual developers

**Phase 2: ARC Database (8 weeks)**
- Mine GitHub for 10K incidents
- Create 100 ARC entries
- Launch public catalog
- Market: "First public ARC database"

**Phase 3: Company Onboarding (Months 3-6)**
- Get 5 companies to contribute incidents
- Build cross-org learning
- Federated learning
- Market: "Learn from X companies"

**Phase 4: Network Effects (Months 7-12)**
- Get 20+ companies
- True cross-org intelligence
- Market: "Industry-wide patterns"

---

## Part 5: Conclusion

### 5.1. Reality Check Summary

**What We Have:**
- ✅ Working local tool (temporal coupling + LLM investigation)
- ✅ Solid foundation (PostgreSQL + Neo4j + tree-sitter)
- ✅ Good architecture (graph-based, agentic)

**What We DON'T Have:**
- ❌ No incident database (0 entries)
- ❌ No cross-org data (0 companies)
- ❌ No cloud infrastructure
- ❌ No CVE integration
- ❌ No public ARC catalog

**Gap:** **80%** of claimed features are unimplemented

### 5.2. Path Forward

**GitHub Mining = Bootstrap Solution**
- Mine 10K incidents from public repos
- Use LLMs to structure data
- Create 100 ARC entries (manually reviewed)
- Launch public catalog
- **Timeline:** 8 weeks, **Cost:** $260

**Why This Works:**
- ✅ Demonstrates concept
- ✅ Real data (not fake)
- ✅ Marketing opportunity ("First ARC database")
- ✅ Network effects (get companies to contribute)
- ✅ Low cost (bootstrappable)

### 5.3. Final Recommendation

**DO THIS NOW:**
1. **Week 1:** Create 10 manual ARC entries (proof of concept)
2. **Week 2-3:** Build GitHub mining script + mine 1K incidents (PoC)
3. **Week 4:** LLM extraction (100 patterns)
4. **Week 5-8:** Scale to 10K incidents → 100 ARC entries
5. **Week 9:** Launch public ARC database + marketing

**STOP CLAIMING:**
- "10,000+ incidents" (say "building ARC database")
- "23 companies" (say "bootstrapping from OSS")
- "Cross-org learning" (say "roadmap Q2 2026")

**BE HONEST:**
- "MVP: Local-only temporal coupling analysis"
- "Roadmap: ARC database (8 weeks), Cloud (12 weeks), Cross-org (20 weeks)"

---

## Related Documents

**Product:**
- [vision_and_mission.md](../../00-product/vision_and_mission.md) - Long-term vision (aspirational)
- [competitive_analysis_detailed_breakdown.md](competitive_analysis_detailed_breakdown.md) - What competitors have

**Implementation:**
- [status.md](../../03-implementation/status.md) - Current implementation status
- [incident_knowledge_graph.md](../../01-architecture/incident_knowledge_graph.md) - Design (not implemented)

**Research:**
- [nvd_integration_analysis.md](nvd_integration_analysis.md) - CVE integration strategy

---

**Last Updated:** October 10, 2025
**Next Review:** November 2025 (post-ARC bootstrap)
