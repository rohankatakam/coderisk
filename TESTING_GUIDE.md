# CodeRisk - Issue Linking Testing & Integration Guide

## 🎯 Overview

This guide shows you how to:
1. **Clean build** from scratch
2. **Test graph construction** from staging data
3. **Tune and iterate** on the implementation
4. **Integrate** the new linking features

---

## 📋 Prerequisites

Before starting, ensure you have:
- ✅ Docker & Docker Compose installed
- ✅ Go 1.21+ installed
- ✅ `jq` installed (for JSON processing): `brew install jq`
- ✅ PostgreSQL client (`psql`) installed
- ✅ Git cloned repository
- ✅ Environment variables set (or use defaults)

---

## 🚀 Quick Start - Clean Build & Test

### Step 1: Clean Everything
```bash
cd /Users/rohankatakam/Documents/brain/coderisk

# Clean all artifacts and databases
make clean-all

# This will:
# - Remove bin/ directory
# - Stop all Docker containers
# - Delete all Docker volumes (CAUTION: All data lost!)
# - Prune Docker system
```

### Step 2: Build & Start Services
```bash
# Full development setup (builds binary + starts services)
make dev

# This will:
# - Build the crisk binary
# - Start Neo4j, PostgreSQL, Redis
# - Initialize database schemas
# - Show service URLs
```

**Expected Output:**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ Development environment ready!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🚀 Test the workflow:
   cd /tmp
   git clone https://github.com/hashicorp/terraform-exec
   cd terraform-exec
   /Users/rohankatakam/Documents/brain/coderisk/bin/crisk init

💡 Quick checks:
   ./bin/crisk --version          # Check version
   ./bin/crisk help               # See all commands

📌 Binary location: ./bin/crisk (local build)
```

### Step 3: Verify Services
```bash
# Check service status
make status

# Should show:
# NAME                   IMAGE                      STATUS
# coderisk-neo4j         neo4j:5.15-community       Up
# coderisk-postgres      postgres:16-alpine         Up
# coderisk-redis         redis:7-alpine             Up
```

**Service URLs:**
- **Neo4j Browser**: http://localhost:7475 (user: `neo4j`, pass: `CHANGE_THIS_PASSWORD_IN_PRODUCTION_123`)
- **PostgreSQL**: `localhost:5433` (user: `coderisk`, pass: `CHANGE_THIS_PASSWORD_IN_PRODUCTION_123`)
- **Redis**: `localhost:6380`

---

## 🧪 Test Graph Construction from Staging Data

### Phase 1: Ingest GitHub Data into Postgres

1. **Set GitHub Token**:
```bash
export GITHUB_TOKEN="your_github_token_here"
```

2. **Run Ingestion** (example with omnara repo):
```bash
# Navigate to a test repository
cd /tmp
git clone https://github.com/omnara-ai/omnara
cd omnara

# Initialize CodeRisk (ingests into Postgres staging tables)
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk init

# This will:
# - Detect it's a GitHub repo
# - Fetch issues, PRs, commits via GitHub API
# - Store in Postgres staging tables (github_issues, github_pull_requests, etc.)
# - Build the knowledge graph in Neo4j
```

**Check Postgres Data:**
```bash
# Connect to Postgres
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk

# Run queries
SELECT COUNT(*) FROM github_issues WHERE repo_id = (SELECT id FROM github_repositories WHERE full_name = 'omnara-ai/omnara');
SELECT COUNT(*) FROM github_pull_requests WHERE repo_id = (SELECT id FROM github_repositories WHERE full_name = 'omnara-ai/omnara');
SELECT COUNT(*) FROM github_commits WHERE repo_id = (SELECT id FROM github_repositories WHERE full_name = 'omnara-ai/omnara');

# Exit
\q
```

**Check Neo4j Graph:**
```bash
# Open Neo4j Browser: http://localhost:7475
# Run Cypher queries:

MATCH (i:Issue) WHERE i.repo_name = 'omnara-ai/omnara' RETURN count(i);
MATCH (pr:PullRequest) WHERE pr.repo_name = 'omnara-ai/omnara' RETURN count(pr);
MATCH (i:Issue)-[r:FIXES_ISSUE]->(pr:PullRequest) RETURN count(r);
```

---

### Phase 2: Test Issue Linking Accuracy

Now that you have data in Postgres and Neo4j, test the linking quality:

```bash
cd /Users/rohankatakam/Documents/brain/coderisk

# Run full pipeline test (currently stubbed - will implement integration)
./scripts/test_full_pipeline.sh omnara

# Currently outputs:
# SKIP status because graph construction from staging is not yet integrated
```

**Manual Validation (for now):**
```bash
# Check if issues are linked to PRs in Neo4j
MATCH (i:Issue {number: 221})-[r:FIXES_ISSUE]->(pr:PullRequest)
RETURN i.number, pr.number, r.confidence, r.evidence

# Expected: Should find temporal correlation (PR #222)
```

---

## 🔧 Integration Steps (TODO)

The following integration work is needed to make the test_full_graph runner work:

### 1. Add Helper Methods to StagingClient

**File:** `internal/database/staging.go`

Add these methods:
```go
// GetClosedIssuesWithTimestamps retrieves all closed issues with timestamps
func (c *StagingClient) GetClosedIssuesWithTimestamps(ctx context.Context, repoID int64) ([]IssueWithTimestamp, error) {
	query := `
		SELECT number, title, state, closed_at, body
		FROM github_issues
		WHERE repo_id = $1 AND state = 'closed' AND closed_at IS NOT NULL
		ORDER BY closed_at DESC
	`

	rows, err := c.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []IssueWithTimestamp
	for rows.Next() {
		var issue IssueWithTimestamp
		if err := rows.Scan(&issue.Number, &issue.Title, &issue.State, &issue.ClosedAt, &issue.Body); err != nil {
			return nil, err
		}
		issues = append(issues, issue)
	}

	return issues, nil
}

// GetPRsMergedNear finds PRs merged within a time window
func (c *StagingClient) GetPRsMergedNear(ctx context.Context, repoID int64, targetTime time.Time, window time.Duration) ([]PRWithTimestamp, error) {
	query := `
		SELECT number, title, state, merged_at, body
		FROM github_pull_requests
		WHERE repo_id = $1
		  AND state = 'closed'
		  AND merged_at IS NOT NULL
		  AND merged_at BETWEEN $2 AND $3
		ORDER BY merged_at DESC
	`

	startTime := targetTime.Add(-window)
	endTime := targetTime.Add(window)

	rows, err := c.db.QueryContext(ctx, query, repoID, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []PRWithTimestamp
	for rows.Next() {
		var pr PRWithTimestamp
		if err := rows.Scan(&pr.Number, &pr.Title, &pr.State, &pr.MergedAt, &pr.Body); err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}

	return prs, nil
}

// Helper types
type IssueWithTimestamp struct {
	Number    int
	Title     string
	State     string
	ClosedAt  *time.Time
	Body      string
}

type PRWithTimestamp struct {
	Number    int
	Title     string
	State     string
	MergedAt  time.Time
	Body      string
}
```

### 2. Integrate Temporal Correlator into Issue Linker

**File:** `internal/incidents/linker.go` (or wherever issue linking happens)

Add temporal boost logic:
```go
import "github.com/rohankatakam/coderisk/internal/graph"

func (il *IssueLinker) CreateIssueLinks(ctx context.Context, repoID int64) error {
	// ... existing code to create edges ...

	// NEW: Apply temporal boost
	correlator := graph.NewTemporalCorrelator(il.stagingDB, il.neo4jClient)
	matches, err := correlator.FindTemporalMatches(ctx, repoID)
	if err != nil {
		log.Warn("Temporal correlation failed", "error", err)
		// Don't fail - just skip temporal boost
	} else {
		for _, match := range matches {
			// Update existing edge or create new one
			il.applyTemporalBoost(ctx, match)
		}
	}

	return nil
}
```

### 3. Integrate Comment Analysis into Issue Extractor

**File:** `internal/github/issue_extractor.go`

Update to fetch and analyze comments:
```go
import "github.com/rohankatakam/coderisk/internal/llm"

func (e *IssueExtractor) ExtractReferences(ctx context.Context, repoID int64) (int, error) {
	// ... existing code ...

	// NEW: Fetch and analyze comments
	comments, err := e.fetchIssueComments(ctx, issue.ID)
	if err != nil {
		log.Warn("Failed to fetch comments", "issue", issue.Number, "error", err)
		comments = []llm.Comment{} // Continue without comments
	}

	// Create comment analyzer
	analyzer := llm.NewCommentAnalyzer(e.llmClient)

	// Extract references from comments
	commentRefs, err := analyzer.ExtractCommentReferences(
		ctx,
		issue.Number,
		issue.Title,
		issue.Body,
		issue.ClosedAt,
		comments,
		repoOwner,
		collaborators,
	)

	if err != nil {
		log.Warn("Comment analysis failed", "issue", issue.Number, "error", err)
		commentRefs = []llm.Reference{} // Continue without comment refs
	}

	// Merge comment refs with body refs
	allRefs = append(allRefs, commentRefs...)

	// ... store references ...
}
```

### 4. Add CLQS Calculation to Init Command

**File:** `cmd/crisk/init.go` (or wherever graph construction completes)

Add CLQS reporting at the end:
```go
import "github.com/rohankatakam/coderisk/internal/graph"

func runInit(cmd *cobra.Command, args []string) error {
	// ... existing graph construction ...

	// NEW: Calculate and display CLQS
	lqs := graph.NewLinkingQualityScore(stagingDB, neo4jClient)
	report, err := lqs.CalculateCLQS(ctx, repoID, repoFullName)
	if err != nil {
		log.Warn("CLQS calculation failed", "error", err)
	} else {
		fmt.Printf("\n")
		fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
		fmt.Printf("║  CODEBASE LINKING QUALITY SCORE                              ║\n")
		fmt.Printf("╠══════════════════════════════════════════════════════════════╣\n")
		fmt.Printf("║ Overall Score:   %.1f/100 (%s - %s)%-18s║\n",
			report.OverallScore, report.Grade, report.Rank, "")
		fmt.Printf("║                                                              ║\n")
		fmt.Printf("║ Components:                                                  ║\n")
		fmt.Printf("║   • Explicit Linking:      %.1f%% %-29s║\n",
			report.Components.ExplicitLinking.Score, "")
		fmt.Printf("║   • Temporal Correlation:  %.1f%% %-29s║\n",
			report.Components.TemporalCorrelation.Score, "")
		fmt.Printf("║   • Comment Quality:       %.1f%% %-29s║\n",
			report.Components.CommentQuality.Score, "")
		fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n")

		// Print recommendations
		if len(report.Recommendations) > 0 {
			fmt.Printf("\n💡 Recommendations:\n")
			for _, rec := range report.Recommendations {
				fmt.Printf("  • %s\n", rec)
			}
		}
	}

	return nil
}
```

---

## 🔄 Iterative Tuning Workflow

Once integrated, use this workflow to tune and improve:

### 1. Run Full Pipeline Test
```bash
./scripts/test_full_pipeline.sh omnara
```

**Output:**
```
╔══════════════════════════════════════════════════════════════╗
║  TEST RESULTS SUMMARY                                        ║
╠══════════════════════════════════════════════════════════════╣
║ Repository:      omnara                                      ║
║ F1 Score:        68.5%                                       ║
║ Precision:       82.3%                                       ║
║ Recall:          58.2%                                       ║
║ Status:          YELLOW LIGHT                                ║
╚══════════════════════════════════════════════════════════════╝

💡 Recommendations:
  • Temporal Correlation is low (45.0%) - implement temporal boost
  • Comment Quality is low (20.0%) - fetch and analyze issue comments
```

### 2. Analyze Failures
```bash
# Check which test cases failed
jq '.layer3.test_cases[] | select(.status == "FAIL")' test_results/omnara_full_pipeline_report.json

# Output:
# {
#   "issue_number": 221,
#   "title": "[FEATURE] allow user to set default agent",
#   "expected_links": {"associated_prs": [222]},
#   "actual_links": [],
#   "status": "FAIL"
# }
```

### 3. Fix the Issue
```bash
# Example: Issue #221 needs temporal correlation
# 1. Integrate temporal_correlator.go (see Integration Steps above)
# 2. Rebuild
make rebuild

# 3. Re-ingest to rebuild graph
cd /tmp/omnara
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk init --force
```

### 4. Re-test
```bash
./scripts/test_full_pipeline.sh omnara
```

**Expected Improvement:**
```
F1 Score: 68.5% → 75.2% ✅ GREEN LIGHT!
```

### 5. Iterate
Repeat steps 1-4 until F1 ≥ 75%:
- Fix failing patterns
- Tune confidence thresholds
- Improve prompts
- Add missing features

---

## 📊 Checking Results

### Manual Neo4j Queries
```cypher
// Count issues with links
MATCH (i:Issue {repo_name: 'omnara-ai/omnara'})-[r:FIXES_ISSUE]->()
RETURN count(DISTINCT i)

// Show confidence distribution
MATCH (i:Issue {repo_name: 'omnara-ai/omnara'})-[r:FIXES_ISSUE]->()
RETURN
  count(CASE WHEN r.confidence >= 0.85 THEN 1 END) as high,
  count(CASE WHEN r.confidence >= 0.70 AND r.confidence < 0.85 THEN 1 END) as medium,
  count(CASE WHEN r.confidence < 0.70 THEN 1 END) as low

// Show evidence types
MATCH (i:Issue {repo_name: 'omnara-ai/omnara'})-[r:FIXES_ISSUE]->()
UNWIND r.evidence as evidence_type
RETURN evidence_type, count(*) as count
ORDER BY count DESC
```

### Check CLQS Over Time
```bash
# Store CLQS in Postgres for tracking
# TODO: Add table to store historical CLQS scores
CREATE TABLE repository_clqs_history (
    id SERIAL PRIMARY KEY,
    repo_id BIGINT REFERENCES github_repositories(id),
    score FLOAT,
    precision FLOAT,
    recall FLOAT,
    f1_score FLOAT,
    calculated_at TIMESTAMP DEFAULT NOW()
);
```

---

## 🐛 Troubleshooting

### Issue: "Connection refused" to Neo4j
```bash
# Check if Neo4j is running
docker ps | grep neo4j

# If not running:
make start

# Check logs:
docker logs coderisk-neo4j
```

### Issue: "Permission denied" on psql
```bash
# Use docker exec instead:
docker exec -it coderisk-postgres psql -U coderisk -d coderisk
```

### Issue: Test fails with "ground truth not found"
```bash
# Ensure test_data files exist:
ls -la test_data/*.json

# Expected:
# omnara_ground_truth.json
# supabase_ground_truth.json
# stagehand_ground_truth.json
```

### Issue: Binary not found
```bash
# Rebuild:
make rebuild

# Check binary:
ls -la bin/crisk
./bin/crisk --version
```

---

## 📝 Summary

**Current Status:**
- ✅ All code compiles
- ✅ Infrastructure ready (Docker, Make, schemas)
- ✅ Ground truth datasets created
- ✅ Test scripts ready
- ⏳ Integration needed (4 steps above)

**Next Steps:**
1. Integrate temporal correlator (2-3 hours)
2. Integrate comment analyzer (2-3 hours)
3. Add CLQS to init command (1 hour)
4. Run full pipeline tests
5. Iterate until F1 ≥ 75%

**Testing Workflow:**
```bash
# 1. Clean build
make clean-all
make dev

# 2. Ingest test data
cd /tmp/omnara
/path/to/crisk init

# 3. Run tests (once integrated)
./scripts/test_full_pipeline.sh omnara

# 4. Tune and re-test
# ... make changes ...
make rebuild
cd /tmp/omnara && crisk init --force
./scripts/test_full_pipeline.sh omnara
```

You're now ready to integrate and test! 🚀
