# CodeRisk Strategic Architecture Plan
## Removing TreeSitter & Implementing Two-Pipeline LLM System

**Document Version:** 1.0
**Date:** 2025-11-14
**Status:** Active Development Plan

---

## Executive Summary

**Goal:** Transform CodeRisk from a mixed TreeSitter/GitHub system into a pure GitHub-data-driven platform with two specialized LLM pipelines that directly support the paper's thesis on $P_{\text{regression}}$ prediction.

**Key Strategic Changes:**
1. **Remove TreeSitter entirely** - eliminates stale data problems and simplifies cloud architecture
2. **Implement Pipeline 1: Link Resolution** - augments 100% confidence graph by extracting implicit references from text
3. **Implement Pipeline 2: Scenario Detection** - detects firefighting patterns (context gaps, panic spirals, incident clusters) for GTM/research

**Timeline:** 3 months (12 weeks)
**Primary Deliverable:** Cloud-ready system validated against Repo Sniper ground truth

**Core Value Proposition:**
Surface shadow market signals ($R_{\text{temporal}}$, $R_{\text{ownership}}$, $R_{\text{coupling}}$) to predict regression probability at the source, shifting risk assessment left to the "hot context" author pre-commit.

---

## Architecture Overview

### Two Distinct LLM Pipelines

**Pipeline 1: Link Resolution (ASSOCIATED_WITH v2)**
- **Purpose:** Fill gaps in 100% confidence graph when GitHub API data is incomplete
- **Runs:** During `crisk init`, Phase 6 (after timeline edges created)
- **Input:** Orphaned issues (closed but no timeline events linking to PR/Commit)
- **LLM Task:** Extract structured references from unstructured text (comments, descriptions)
- **Output:** ASSOCIATED_WITH edges with 0.7-0.95 confidence scores
- **Value:** Completes the graph for accurate $R_{\text{temporal}}$ calculation

**Pipeline 2: Scenario Detection (Narrative Intelligence)**
- **Purpose:** Detect firefighting patterns for GTM/research/risk assessment
- **Runs:** During `crisk init`, Phases 7-8 (after all edges exist)
- **Input:** All PRs with complete edge context
- **LLM Task:** Classify PRs (FEATURE/HOTFIX/REVERT/CHORE) + Generate narratives
- **Output:** Scenario nodes (context gaps, panic spirals, incident clusters)
- **Value:** GTM intelligence + Repo Sniper validation + temporal risk signals

### Data Flow

```
GitHub API → PostgreSQL Staging → Neo4j 100% Graph → Link Resolution → Complete Graph → Scenario Detection → Risk Signals
```

1. **GitHub API ingestion** creates PostgreSQL staging tables
2. **Graph builder** creates 100% confidence nodes/edges in Neo4j
3. **Link resolver** adds LLM-extracted edges for orphaned issues
4. **Scenario detector** analyzes patterns and creates Scenario nodes
5. **crisk check** queries scenarios to surface $R_{\text{temporal}}$ signals

---

## Phase 1: Foundation (Month 1)

### Week 1: Validate 100% Confidence Graph

**Objective:** Establish baseline quality of current graph construction before adding LLM augmentation

#### Task 1.1: Verify Timeline Edge Implementation
- Run `./bin/test-graph-construction` validation script on omnara-ai/omnara repository
- Confirm zero duplicate edges across all three edge type pairs:
  - MERGED_AS vs ASSOCIATED_WITH (PR → Commit)
  - REFERENCES vs ASSOCIATED_WITH (Issue → PR)
  - CLOSED_BY vs ASSOCIATED_WITH (Issue → Commit)
- Validate that all three timeline edge types are being created correctly from timeline events
- Document any warnings or errors for investigation

#### Task 1.2: Measure Timeline Coverage
- Query Neo4j to calculate percentage of closed issues that have timeline-based edges
- Run coverage query: count issues with REFERENCES or CLOSED_BY edges vs total closed issues
- Document coverage metrics for omnara repository as baseline
- Establish expected improvement targets for link resolution pipeline (target: +30% coverage)

#### Task 1.3: Audit Existing 100% Confidence Edges
- **AUTHORED edges:** Verify Developer → Commit relationships using git author metadata
- **MODIFIED edges:** Verify Commit → File relationships from commit file change data
- **CREATED edges:** Verify Developer → PR relationships from PR author field
- **MERGED_AS edges:** Verify PR → Commit from merge_commit_sha field (GitHub's definitive merge signal)
- **REFERENCES edges:** Verify Issue → PR from timeline cross-reference events
- **CLOSED_BY edges:** Verify Issue → Commit from timeline closed events with commit_id
- Run statistical queries to validate edge counts match expected patterns
- Identify any anomalies or missing data for remediation

**Deliverable:** Comprehensive validation report documenting:
- Current edge coverage percentages
- Identified gaps to be filled by link resolution
- Baseline metrics for comparison after implementation

---

### Week 2: Implement Link Resolution Pipeline (ASSOCIATED_WITH v2)

**Objective:** Build LLM-powered system to extract implicit Issue→PR/Commit links from text when timeline data is missing

#### Task 2.1: Create Link Resolution Infrastructure
- Create new file `internal/github/link_resolver.go` with core resolution logic
- Implement orphaned issue identification query (issues closed without timeline events)
- Design LinkCandidate struct to hold extraction results with evidence
- Create multi-signal validation framework supporting temporal, semantic, and file overlap checks
- Build batch processing system (20 issues per LLM call for cost efficiency)

#### Task 2.2: Implement LLM Extraction Logic
- Design extraction prompt for finding PR/Commit references in issue comments and descriptions
- Handle common edge cases:
  - Version numbers: "fixed in v2.1.0" → query releases/tags for associated commit
  - Vague references: "fixed in latest" → find most recent PR/commit before closure
  - Partial references: "see #145" → extract PR number 145
  - Commit SHAs: "fixed in abc123" → resolve short SHA to full commit
- Implement batch processing with structured JSON response parsing
- Extract evidence strings for transparency and debugging

#### Task 2.3: Build Validation Framework
- **Temporal Validation:**
  - Check if candidate PR merged or commit pushed within 7 days of issue closure
  - Boost confidence by 1.2x if temporal proximity is strong
- **Semantic Validation:**
  - Verify issue number appears in PR title, PR body, or commit message
  - Boost confidence by 1.3x if explicit mention found
- **File Overlap Validation:**
  - Extract file references from issue description (e.g., "bug in src/auth.ts")
  - Compare with files modified in candidate PR/Commit
  - Boost confidence by 1.2x if significant overlap (>30%)
- Implement confidence scoring: base LLM confidence × validation multipliers, capped at 0.95
- Set minimum confidence threshold at 0.7 for edge creation
- Log all validation results for audit and CLQS evidence tracking

#### Task 2.4: Create Edge Storage Logic
- Store ASSOCIATED_WITH edges in Neo4j with comprehensive properties:
  - `confidence`: 0.7-0.95 (never 1.0, reserved for timeline edges)
  - `detected_via`: "llm_link_resolution"
  - `evidence`: original text that triggered extraction (e.g., "Comment: see PR #145")
  - `temporal_validation`: boolean flag
  - `semantic_validation`: boolean flag
  - `file_overlap_validation`: boolean flag
  - `validated_at`: timestamp
- Implement collision prevention: skip edge creation if REFERENCES or CLOSED_BY already exists
- Store audit trail in PostgreSQL `llm_link_audit` table for debugging and validation
- Track extraction source (which comment, which field) for evidence traceability

#### Task 2.5: Integrate into `crisk init`
- Add Phase 6 to initialization pipeline (runs immediately after Phase 5: BuildGraph)
- Add command flag `--enable-link-resolution` defaulting to enabled when PHASE2_ENABLED=true
- Log progress: "Processing X orphaned issues", "Resolved Y candidate links", "Created Z ASSOCIATED_WITH edges"
- Display coverage improvement metrics: orphaned issues before → resolved links after
- Store resolution statistics in postgres for historical tracking

**Deliverable:** Working link resolution pipeline producing measurable coverage improvements with transparent confidence scoring and evidence tracking

---

### Week 3: Implement PR Classification Pipeline

**Objective:** Port Repo Sniper's PR classification system to categorize all PRs for pattern detection

#### Task 3.1: Port Repo Sniper Classification Logic
- Create new file `internal/github/pr_classifier.go` with classification engine
- Replicate Repo Sniper's four-category taxonomy:
  - **FEATURE:** New functionality, major enhancements, new components
  - **HOTFIX:** Urgent fixes for recent bugs (keywords: fix, bug, broken, regression, critical, hotfix)
  - **REVERT:** Explicit rollbacks (keywords: Revert, Undo, Rollback)
  - **CHORE:** Dependencies updates, documentation, minor styling, refactoring
- Define classification rules in system prompt:
  - Keyword patterns for each category
  - Semantic indicators (e.g., rapid succession of fixes = HOTFIX)
  - File change patterns (e.g., package.json only = CHORE)
- Include reasoning requirement in prompt for transparency and debugging

#### Task 3.2: Build Batch Classification System
- Fetch all merged PRs from PostgreSQL staging for target repository
- Extract PR metadata for LLM analysis:
  - Title and body text
  - Labels (bug, enhancement, documentation, etc.)
  - Files changed and lines changed counts
  - Merge date and creation date (for velocity analysis)
  - Author information
- Batch PRs in groups of 100 for single LLM call (cost optimization via batching)
- Design JSON response format: `[{"pr_number": 123, "type": "HOTFIX", "confidence": 0.92, "reasoning": "..."}]`
- Parse structured response and validate all PRs received classification
- Handle partial failures (if LLM misses PRs in batch, retry individually)

#### Task 3.3: Create Classification Storage
- Design PostgreSQL table `pr_classifications`:
  - `id` (primary key)
  - `repo_id` (foreign key to repositories)
  - `pr_number`
  - `classification` (FEATURE | HOTFIX | REVERT | CHORE)
  - `confidence` (0.0-1.0)
  - `reasoning` (text field with LLM explanation)
  - `classified_at` (timestamp)
  - Unique constraint on (repo_id, pr_number)
- Add index on (repo_id, pr_number, classification) for fast scenario detection queries
- Implement upsert logic (update classification if PR re-analyzed)
- Store classification distribution stats (counts per category) for metrics calculation

#### Task 3.4: Integrate into `crisk init`
- Add Phase 7 to initialization pipeline (runs after link resolution completes)
- Display classification progress: "Classifying X PRs in batches of 100"
- Show classification distribution in output:
  - Features: X (Y%)
  - Hotfixes: X (Y%)
  - Reverts: X (Y%)
  - Chores: X (Y%)
- Calculate and log initial firefighting indicators:
  - Hotfix rate: (HOTFIX count / total PR count) × 100
  - Revert rate: (REVERT count / total PR count) × 100
- Flag repositories with hotfix rate > 30% or revert rate > 10% as high-priority targets

**Deliverable:** PR classification system producing Repo Sniper-compatible categorizations stored in PostgreSQL for scenario detection

---

### Week 4: Implement Scenario Detection Pipeline

**Objective:** Build pattern recognition system to detect firefighting scenarios and create Scenario nodes in graph

#### Task 4.1: Build Context Gap Detector
- Query pr_classifications table for all FEATURE PRs in repository
- For each FEATURE, find PRs merged within 48 hours after it (tight time window)
- Filter subsequent PRs for HOTFIX classification (indicates reactive bug fixing)
- Calculate file overlap percentage:
  - Get files modified in FEATURE PR from postgres
  - Get files modified in each HOTFIX PR
  - Overlap = (intersection / union) × 100
- Create Scenario node if overlap > 30% (indicates fix addressed FEATURE's changes)
- Set severity based on fix count:
  - 1 fix = P3 (low severity)
  - 2-3 fixes = P2 (medium severity)
  - 4+ fixes = P1 (high severity)
- Store evidence: FEATURE PR number, HOTFIX PR numbers, overlapping files, time delta

#### Task 4.2: Build Panic Spiral Detector
- Scan PR timeline for consecutive sequences of HOTFIX or REVERT PRs
- Detection criteria: 2+ HOTFIXes or REVERTs with no intervening FEATURE or CHORE
- Calculate spiral characteristics:
  - Length: number of consecutive fixes (2 = P2, 3+ = P1)
  - Duration: time between first and last fix in spiral
  - File churn: total lines changed across all fixes in spiral
- Identify cascading failures where fixes introduce new bugs (HOTFIX → HOTFIX → HOTFIX pattern)
- Create Scenario node with type="panic_spiral"
- Store evidence: all PR numbers in spiral, affected files, total time duration

#### Task 4.3: Build Incident Cluster Detector
- For each FEATURE PR, find all issues linked via ASSOCIATED_WITH or REFERENCES edges
- For each linked issue, find all HOTFIX PRs that also link to same issue
- Count total follow-up fixes per FEATURE to identify "smoking gun" features
- Rank by severity:
  - 1-2 fixes = P3
  - 3-5 fixes = P2
  - 6+ fixes = P1 (like PostHog's PR #40814 → 11 follow-up fixes)
- Calculate firefighting cost:
  - Reviewer tax: sum of lines changed in all HOTFIX PRs
  - MTTR estimate: average time from issue creation to final fix
  - Engineers involved: count of unique authors across all fixes
- Create Scenario node with type="incident_cluster"
- Store evidence: FEATURE PR, linked issues, all HOTFIX PRs, cost breakdown

#### Task 4.4: Implement Narrative Generation
- Design LLM prompt for generating human-readable scenario explanations
- Input context:
  - FEATURE PR: title, description, files changed, author
  - HOTFIX PRs: titles, descriptions, files changed
  - Linked issues: titles, descriptions, closure reasons
  - File changes: diff summaries, affected modules
- Output requirements:
  - Concise narrative (2-3 sentences) explaining what happened and why
  - Example: "Feature PR #500 OAuth refactor introduced login crash requiring 2 hotfixes within 24 hours affecting auth flow"
  - Causal reasoning, not just description
  - Actionable insight where possible
- Store narrative in Scenario node `narrative` property
- Include confidence assessment in narrative quality

#### Task 4.5: Create Scenario Graph Schema
- Design Scenario node with comprehensive properties:
  - `type`: "context_gap" | "panic_spiral" | "incident_cluster"
  - `severity`: "P1" | "P2" | "P3"
  - `narrative`: human-readable explanation
  - `confidence`: 0.85-0.95 (based on evidence quality)
  - `evidence_quality`: map showing which links are 100% vs LLM-resolved
  - `firefighting_cost`: object with reviewer_tax, mttr_estimate, engineers_involved
  - `detected_at`: timestamp
  - `files_affected`: array of file paths
  - `trigger_pr`: PR number that initiated the scenario
- Create edges to connect Scenario to graph entities:
  - `TRIGGERED_BY`: Scenario → Feature PR (the initiating change)
  - `RESULTED_IN`: Scenario → Hotfix PRs (the reactive fixes)
  - `LINKED_TO`: Scenario → Issues (connected problems)
  - `AFFECTED`: Scenario → Files (impacted file nodes)
  - `INVOLVED`: Scenario → Developers (authors of fixes)
- Store evidence quality breakdown to support CLQS confidence scoring
- Index Scenario nodes by severity and detection date for fast querying

#### Task 4.6: Integrate into `crisk init`
- Add Phase 8 to initialization pipeline (runs after PR classification)
- Run all three detectors in sequence:
  1. Context gaps (FEATURE → HOTFIX patterns)
  2. Panic spirals (consecutive HOTFIX chains)
  3. Incident clusters (FEATURE → multiple fixes)
- Display scenario summary:
  - "Detected X context gaps (Y P1, Z P2)"
  - "Detected X panic spirals (Y P1, Z P2)"
  - "Detected X incident clusters (Y P1, Z P2)"
- Output top 3 highest-severity scenarios with narratives for immediate visibility
- Store total scenario count and severity distribution in postgres metrics table
- Log scenario detection statistics for validation against Repo Sniper

**Deliverable:** Complete scenario detection system creating graph-based firefighting intelligence with narrative explanations

---

## Phase 2: Validation & Metrics (Month 2)

### Week 5: Calculate Firefighting Metrics

**Objective:** Implement Repo Sniper's firefighting metrics to validate scenario detection accuracy

#### Task 5.1: Implement Revert Rate Calculation
- Query pr_classifications table: `SELECT COUNT(*) FROM pr_classifications WHERE classification = 'REVERT'`
- Query total PR count: `SELECT COUNT(*) FROM pr_classifications WHERE repo_id = X`
- Calculate percentage: (REVERT count / total PR count) × 100
- Store in postgres table `firefighting_metrics` with repo_id, metric_name, metric_value, calculated_at
- Compare against DORA baseline: Elite teams < 5%, High performers < 15%
- Flag repositories with revert rate > 15% as concerning

#### Task 5.2: Implement Hotfix Rate Calculation
- Query pr_classifications table: `SELECT COUNT(*) FROM pr_classifications WHERE classification = 'HOTFIX'`
- Calculate percentage: (HOTFIX count / total PR count) × 100
- Identify threshold for reactive development: > 20% indicates firefighting culture
- Repo Sniper validation targets (from actual data):
  - Strapi: 48%
  - Windmill: 34.7%
  - Plane: 39%
  - Formbricks: 51.4%
- Store metric and flag high-firefighting repositories

#### Task 5.3: Implement Regression Ratio Calculation
- Query FEATURE count: `SELECT COUNT(*) WHERE classification = 'FEATURE'`
- Query HOTFIX count: `SELECT COUNT(*) WHERE classification = 'HOTFIX'`
- Calculate ratio: HOTFIX count / FEATURE count
- Interpretation:
  - < 0.1 (1 fix per 10 features) = healthy velocity
  - 0.1-0.5 = moderate context issues
  - > 0.5 = severe context collapse
- Store ratio and classify repository health tier

#### Task 5.4: Implement Reviewer Tax Calculation
- Sum lines changed in all HOTFIX and REVERT PRs from postgres
- Sum lines changed in all FEATURE PRs
- Calculate ratio: firefighting lines / feature lines
- Interpretation: percentage of code review effort spent on reactive work vs productive work
- Example: 51% reviewer tax means majority of review capacity consumed by firefighting
- Store metric and highlight in reports as key pain indicator

#### Task 5.5: Calculate Firefighting Score
- Implement Repo Sniper's weighted formula:
  - `Score = (5.0 × RevertRate) + (2.0 × HotfixRate) + (10.0 × PanicClusterCount)`
- Weights rationale:
  - Reverts are severe (5.0 weight) - production failures requiring rollback
  - Hotfixes indicate reactive work (2.0 weight) - unplanned bug fixing
  - Panic spirals are highest severity (10.0 weight) - cascading failures
- Calculate total score and classify repositories:
  - Priority 1 (The Bleeding): score ≥ 50
  - Priority 2 (The Worried): score ≥ 20
  - Healthy: score < 20
- Store score, priority classification, and component breakdown in postgres
- Validate calculation against Repo Sniper ground truth (PostHog: 158.0 expected)

**Deliverable:** Complete metrics calculation system aligned with Repo Sniper algorithm producing comparable scores

---

### Week 6: Integrate Metrics into `crisk check`

**Objective:** Surface scenario-based temporal risk signals to developers during pre-commit checks

#### Task 6.1: Build File Scenario History Query
- Query Neo4j for all Scenarios connected to target file via AFFECTED edge
- Filter by recency: scenarios detected within last 90 days (configurable)
- Filter by severity: only P1 and P2 scenarios (ignore P3 for signal-to-noise ratio)
- Sort by detection date descending (most recent first)
- Limit to 5 scenarios for display (prevent overwhelming output)
- Return scenario metadata: type, severity, narrative, trigger_pr, follow_up_prs, detection_date

#### Task 6.2: Display Scenarios in `crisk check` Output
- Before Phase 1 metrics, check if file has scenario history
- If scenarios exist, display warning section:
  - Header: "⚠️ {filename} has firefighting history"
  - For each scenario: "   {type}: {narrative}"
  - Example: "   context_gap: Feature PR #500 OAuth refactor caused 2 hotfixes within 24hrs"
- Provide actionable guidance based on scenario type:
  - Context gap: "Consider reviewing PR #500 and fixes #510, #515 before making changes"
  - Panic spiral: "This file has triggered cascading failures - extra caution recommended"
  - Incident cluster: "High bug density area - recommend consulting {original_author}"
- Show total scenario count: "{count} high-severity incidents in last 90 days"

#### Task 6.3: Build Developer Ownership Query
- Query Neo4j for developer's historical commits to target file:
  - `MATCH (d:Developer {login: $author})-[:AUTHORED]->(c:Commit)-[:MODIFIED]->(f:File {path: $file})`
- Calculate ownership metrics:
  - Commit count: total commits by developer to this file
  - Staleness: days since last commit (datetime.now - max(commit.authored_date))
  - Familiarity score: commits / total commits to file (percentage ownership)
- Identify special cases:
  - Original author: developer with first commit to file
  - Never touched: developer has zero commits to this file
  - Recent contributor: developer's last commit was < 30 days ago
- Display ownership context: "⚠️ You have never modified this file before" or "✓ You are the original author"

#### Task 6.4: Integrate $R_{\text{temporal}}$ Signals
- Calculate file-level incident density:
  - Total scenarios affecting file in last 90 days
  - Average severity (weighted: P1=3, P2=2, P3=1)
  - Most recent high-severity scenario date
- Calculate file-level hotfix rate:
  - Count HOTFIXes that modified this file (from pr_classifications + file changes)
  - Count all PRs that modified this file
  - Ratio = HOTFIX count / total PR count for this specific file
- Display temporal risk score: "High incident rate (5 scenarios, 40% hotfix rate)"
- Compare to repository average to show if file is outlier

#### Task 6.5: Update Risk Scoring
- Incorporate scenario density into Phase 1 escalation decision:
  - Current threshold: coupling > X, co-change > Y
  - New threshold: scenarios ≥ 2 P1 in 90 days → auto-escalate to Phase 2
- Pass scenario context to Gemini investigator:
  - Include scenario narratives in kickoff prompt
  - Provide linked PR numbers for agent to investigate
  - Context enables richer Phase 2 analysis with historical incident knowledge
- Display escalation reason: "Escalating due to high incident history (3 P1 scenarios)"

**Deliverable:** `crisk check` surfacing scenario-based temporal risk signals with actionable developer guidance

---

### Week 7: Validate Against Repo Sniper Ground Truth

**Objective:** Prove CodeRisk scenario detection accuracy by comparing to Repo Sniper's validated results

#### Task 7.1: Build Validation Test Suite
- Create validation script `scripts/validate_scenario_detection.py`
- Target repositories: 14 successfully analyzed Repo Sniper repos:
  - Strapi, Windmill, Plane, Formbricks, Directus
  - Supabase, Meilisearch, Cal.com, Appwrite, n8n
  - Twenty, Lago, Excalidraw, Airbyte
- For each repository:
  1. Run `crisk init` to generate scenarios and metrics
  2. Extract firefighting metrics from postgres
  3. Extract scenarios from Neo4j
  4. Load Repo Sniper ground truth from JSON files
  5. Run comparison analysis

#### Task 7.2: Compare Firefighting Scores
- Calculate percentage difference: `|coderisk_score - repo_sniper_score| / repo_sniper_score × 100`
- Target accuracy: ±10% for overall score
- Track score components separately:
  - Revert rate comparison
  - Hotfix rate comparison
  - Panic cluster count comparison
- Investigate outliers with > 20% deviation:
  - Examine classification differences (did we classify different PRs as HOTFIX?)
  - Review panic spiral detection logic (are we missing patterns?)
  - Check for data quality issues (missing PRs in our ingestion?)
- Document root causes of deviations and implement fixes

#### Task 7.3: Compare Incident Cluster Detection
- Load Repo Sniper's "smoking gun" PRs (top incident clusters)
- Example: Strapi PR #24706 → 13 follow-up fixes
- Match against CodeRisk's incident clusters:
  - Does our detection find the same trigger PR?
  - Do we count the same number of follow-up fixes?
  - Are the linked issues/PRs matching?
- Calculate overlap percentage: (matched clusters / total ground truth clusters) × 100
- Target: ≥70% overlap in top 5 clusters
- Analyze false negatives:
  - Clusters Repo Sniper found but CodeRisk missed
  - Potential causes: missing link resolution, classification errors, different time windows
- Analyze false positives:
  - Clusters CodeRisk found but Repo Sniper didn't flag
  - Validate if these are genuine scenarios or detection errors

#### Task 7.4: Compare Panic Spiral Detection
- Load Repo Sniper's panic spiral count per repository
- Example: Windmill 10 panic spirals, Strapi 8 spirals
- Compare against CodeRisk's detected spirals:
  - Total count match?
  - Same PR sequences identified?
  - Same severity classifications?
- Calculate detection rate: (CodeRisk spirals / Repo Sniper spirals) × 100
- Target: ≥80% detection rate
- Identify false negatives:
  - Spirals we missed: why? (too strict time window? classification error?)
  - Examine edge cases: single REVERT vs HOTFIX sequences
- Document detection algorithm improvements needed

#### Task 7.5: Measure Link Resolution Effectiveness
- For repositories with low timeline coverage (< 60%), measure impact of link resolution:
  - Coverage before link resolution: X%
  - Coverage after link resolution: Y%
  - Improvement: Y - X percentage points
- Calculate link quality metrics:
  - Confidence score distribution (how many at 0.7-0.8 vs 0.8-0.9 vs 0.9-0.95)
  - Validation success rates (temporal, semantic, file overlap)
  - False positive rate (manual audit of 100 random ASSOCIATED_WITH edges)
- Measure incident cluster accuracy improvement:
  - Clusters detected without link resolution: X
  - Clusters detected with link resolution: Y
  - Additional clusters found: Y - X
- Document which repositories benefited most (those with poor timeline coverage)

**Deliverable:** Comprehensive validation report proving CodeRisk matches or exceeds Repo Sniper accuracy with documented deviations and remediation plans

---

### Week 8: Optimize LLM Performance

**Objective:** Reduce operational costs and improve pipeline efficiency through optimization

#### Task 8.1: Measure Current Token Usage
- Instrument all LLM calls with token counting:
  - Track input tokens and output tokens separately
  - Log per-call token usage to postgres for analysis
- Calculate token usage breakdown per repository:
  - Link resolution: average tokens per orphaned issue
  - PR classification: average tokens per PR batch
  - Narrative generation: average tokens per scenario
- Calculate total cost per repository at Gemini Flash pricing:
  - Input: $0.01 per 1M tokens
  - Output: $0.04 per 1M tokens
- Identify highest-cost operations for optimization focus
- Establish baseline metrics: tokens per repo, cost per repo, processing time per repo

#### Task 8.2: Optimize Prompt Engineering
- **Link resolution optimization:**
  - Remove verbose instructions, keep only essential classification criteria
  - Test few-shot examples vs zero-shot (few-shot may increase accuracy with fewer retries)
  - Compress context: send only last 3 comments instead of all comments
- **PR classification optimization:**
  - Test batch size impact: 50 vs 100 vs 150 PRs per call
  - Shorter system prompt with bullet points instead of paragraphs
  - Remove reasoning requirement for classifications above 0.9 confidence
- **Narrative generation optimization:**
  - Limit context to essential fields only (titles, key file changes)
  - Use template-based narratives for common patterns (reduce generation cost)
  - Only generate narratives for P1 and P2 scenarios (skip P3)
- Measure token reduction: target 30-50% reduction while maintaining accuracy
- A/B test optimized prompts vs original on 5 test repositories

#### Task 8.3: Implement Caching Strategy
- **PR classification caching:**
  - Store classifications in postgres with creation timestamp
  - On subsequent `crisk init` runs, skip PRs already classified if < 30 days old
  - Invalidation: force refresh with `--force-refresh` flag or if PR metadata changed
- **Link resolution caching:**
  - Cache extracted links with confidence scores and evidence
  - Revalidate cached links if issue was updated (new comments added)
  - Expire cache after 30 days to account for new references
- **Scenario detection caching:**
  - Scenarios are derived data, recompute if underlying classifications change
  - Cache scenario narratives to avoid regeneration on unchanged data
- Measure cache hit rate: target 60-80% on subsequent runs
- Calculate cost savings: cached calls × average cost per call

#### Task 8.4: Batch Optimization
- **Experiment with batch sizes:**
  - Test PR classification with batches of 20, 50, 100, 150, 200
  - Measure: cost (larger = fewer API calls), quality (smaller = better accuracy), latency (larger = slower)
- **Find optimal trade-offs:**
  - Link resolution: 20 issues per batch (current) vs 50 (test)
  - PR classification: 100 PRs per batch (current) vs 150 (test)
  - Narrative generation: generate per-scenario (current) vs batch 10 scenarios (test)
- Measure quality degradation at larger batch sizes:
  - Classification accuracy drop-off at 200+ PRs per batch?
  - Link extraction quality issues with 50+ issues per batch?
- Document recommended batch sizes:
  - Small repos (< 500 PRs): larger batches acceptable
  - Large repos (> 2000 PRs): optimize for quality with smaller batches
- Implement adaptive batching based on repository size

**Deliverable:** Cost-optimized LLM pipeline with:
- 30-50% token reduction through prompt optimization
- 60-80% cache hit rate on repeated runs
- Documented batch size recommendations
- Cost per repository reduced to < $0.30 (from current ~$0.50)

---

## Phase 3: TreeSitter Removal & Cloud Migration (Month 3)

### Week 9: Remove TreeSitter System

**Objective:** Eliminate TreeSitter dependency to simplify architecture and enable cloud migration

#### Task 9.1: Delete TreeSitter Code
- Remove entire directory: `internal/treesitter/`
- Remove TreeSitter-specific ingestion logic: `internal/ingestion/processor.go`
- Remove local repository ingestion command: `cmd/crisk/init-local.go`
- Update all import statements in files that referenced treesitter packages
- Remove TreeSitter dependencies from `go.mod`:
  - `github.com/tree-sitter/go-tree-sitter`
  - `github.com/tree-sitter/tree-sitter-javascript`
  - `github.com/tree-sitter/tree-sitter-python`
  - `github.com/tree-sitter/tree-sitter-typescript`
- Run `go mod tidy` to clean unused dependencies

#### Task 9.2: Remove TreeSitter from Graph Construction
- Update `internal/graph/builder.go`:
  - Remove TreeSitter file node creation logic (nodes with `current: true`)
  - Keep only GitHub-sourced File nodes (from commit file changes with `historical: true`)
  - Remove DEPENDS_ON edge creation from import statements
  - Verify Function and Class node creation is also removed (marked deprecated)
- Update graph schema documentation to reflect File nodes are GitHub-only
- Simplify `BuildGraph` function to focus on GitHub data processing
- Remove any TreeSitter-related configuration flags or environment variables

#### Task 9.3: Replace TreeSitter Blast Radius with Co-Change Patterns
- Update `cmd/crisk/check.go` blast radius calculation:
  - Remove TreeSitter parsing calls
  - Remove DEPENDS_ON edge queries
- Implement new co-change pattern query:
  - Find files modified in same commits as target file
  - Filter for co-change count ≥ 3 (changed together at least 3 times)
  - Order by co-change frequency descending
  - Limit to top 20 coupled files
- Query logic (Cypher):
  ```
  MATCH (f1:File {path: $path})<-[:MODIFIED]-(c:Commit)-[:MODIFIED]->(f2:File)
  WITH f2, count(c) as co_change_count
  WHERE co_change_count >= 3
  RETURN f2.path as coupled_file, co_change_count
  ORDER BY co_change_count DESC
  LIMIT 20
  ```
- Validate results are meaningful: co-changed files should be logically related
- Update output format to explain blast radius source: "Files historically changed together"

#### Task 9.4: Update Documentation
- Remove TreeSitter references from:
  - README.md
  - Architecture overview docs
  - DEVELOPMENT_WORKFLOW.md
- Update EDGE_CONFIDENCE_HIERARCHY.md:
  - Remove DEPENDS_ON edge documentation
  - Remove File node `current: true` marker explanation
- Create migration guide: `docs/TREESITTER_REMOVAL_MIGRATION.md`
  - Explain shift from structural to temporal coupling analysis
  - Document benefits: simpler cloud architecture, no stale data issues
  - Provide query examples for new co-change pattern analysis
- Update API documentation if any public interfaces changed

#### Task 9.5: Test `crisk check` Without TreeSitter
- Run `crisk check` on diverse sample files from omnara repository:
  - Frequently changed files (expect rich co-change data)
  - Rarely changed files (expect sparse blast radius)
  - New files (expect no historical co-change patterns)
- Verify blast radius results:
  - Are coupled files logically related?
  - Is output actionable for developers?
  - Any errors or missing functionality?
- Compare blast radius quality to TreeSitter DEPENDS_ON approach:
  - Temporal coupling may be more accurate (reflects actual change patterns)
  - Structural coupling was theoretically precise but became stale
- Document any edge cases or limitations discovered
- Confirm no regression in core functionality

**Deliverable:** Fully functional system with TreeSitter removed, validated on test repositories, with updated documentation

---

### Week 10: Migrate PostgreSQL to Cloud

**Objective:** Move PostgreSQL staging database to cloud provider for production readiness

#### Task 10.1: Choose Cloud Provider
- Evaluate cloud postgres options:
  - **Neon:** Serverless, auto-scaling, branch-based workflows, excellent for development
  - **Supabase:** Built-in connection pooling, realtime features, good DX
  - **AWS RDS:** Enterprise-grade, high availability, broad region support
- Selection criteria:
  - Cost for projected data volume (100GB staging data)
  - Latency (proximity to Neo4j Aura region)
  - Built-in connection pooling (critical for LLM pipeline concurrency)
  - Backup/restore capabilities (automated daily backups required)
  - Team familiarity and operational overhead
- Recommended: Neon or Supabase for faster iteration, RDS for enterprise deployment
- Select region matching Neo4j Aura region (e.g., us-east-1) for optimal latency

#### Task 10.2: Export Staging Schema
- Dump current PostgreSQL schema using `pg_dump`:
  - All GitHub staging tables: repositories, commits, pull_requests, issues, issue_timeline, developers
  - New tables: pr_classifications, firefighting_metrics, issue_commit_refs, llm_link_audit
  - All indexes for performance
  - All foreign key constraints for data integrity
- Export schema only (no data) for initial setup: `pg_dump --schema-only`
- Export data separately for large tables: `pg_dump --data-only --table=github_commits`
- Validate dump files are complete and not corrupted
- Store dump files in secure location with version control

#### Task 10.3: Provision Cloud Database
- Create production database instance:
  - Choose appropriate tier (minimum: 4 vCPU, 16GB RAM for LLM pipeline concurrency)
  - Enable connection pooling (pgBouncer or provider's built-in)
  - Configure connection limits (recommend: 100 max connections)
- Configure security:
  - Enable SSL/TLS for connections
  - Set up IP allowlist (restrict to application server IPs)
  - Rotate credentials and store in environment variables
- Set up automated backups:
  - Daily snapshots with 30-day retention
  - Point-in-time recovery enabled
  - Test restore procedure
- Configure monitoring and alerts:
  - CPU and memory usage alerts (> 80%)
  - Connection pool exhaustion alerts
  - Slow query logging enabled

#### Task 10.4: Migrate Data
- Restore schema to cloud database: `psql -h <cloud-host> -U <user> -d <db> < schema.sql`
- Migrate data in batches (avoid timeout on large tables):
  - Start with small tables: repositories, developers
  - Use parallel export/import for large tables: commits, pull_requests, issues
  - Monitor progress and estimate total migration time
- Verify data integrity after migration:
  - Row count match: `SELECT COUNT(*) FROM github_commits` (local vs cloud)
  - Constraint validation: check all foreign keys resolve correctly
  - Index validity: `REINDEX DATABASE` and verify no errors
- Run sample queries to test performance:
  - LLM pipeline queries (orphaned issues, PR classifications)
  - Graph builder queries (commit file changes)
  - Scenario detection queries (temporal patterns)

#### Task 10.5: Update Application Configuration
- Update `.env` with cloud postgres connection string:
  - POSTGRES_HOST, POSTGRES_PORT, POSTGRES_DB, POSTGRES_USER, POSTGRES_PASSWORD
  - Or single POSTGRES_DSN connection string
- Update connection pooling settings in application:
  - Set max_connections based on cloud tier limits
  - Configure connection timeout and retry logic
- Test `crisk init` end-to-end with cloud postgres:
  - Run on small test repository (< 100 PRs)
  - Verify all phases complete successfully
  - Check no connection errors or timeouts
- Measure latency impact:
  - Compare query times: local postgres vs cloud postgres
  - Optimize slow queries with EXPLAIN ANALYZE
  - Add indexes if needed (e.g., on pr_classifications.classification)
- Load test with concurrent operations:
  - Simulate multiple `crisk init` runs
  - Verify connection pool handles concurrency
  - Monitor database metrics during load test

**Deliverable:** Cloud PostgreSQL operational with all data migrated, application updated and tested, monitoring configured

---

### Week 11: Migrate Neo4j to Aura

**Objective:** Move Neo4j graph database to Aura cloud service for scalability and reliability

#### Task 11.1: Provision Neo4j Aura Instance
- Choose appropriate Aura tier based on projections:
  - Estimate node count: ~50K nodes per repo (Commits, PRs, Issues, Files, Developers, Scenarios)
  - Estimate edge count: ~200K edges per repo (AUTHORED, MODIFIED, MERGED_AS, etc.)
  - Recommended: Professional tier (8GB RAM, sufficient for 5-10 repositories)
- Select region matching PostgreSQL cloud region for low latency
- Configure instance settings:
  - Database name: coderisk-production
  - Enable automatic backups (daily)
  - Configure connection limits
- Obtain credentials:
  - Connection URI (bolt+s:// protocol for secure connection)
  - Username and password
  - Store in environment variables securely

#### Task 11.2: Export Local Graph
- Stop local Neo4j instance to ensure consistent dump
- Use `neo4j-admin database dump` to export database:
  - Command: `neo4j-admin database dump neo4j --to=/tmp/neo4j-backup.dump`
  - Verify dump file created and check file size
- Document pre-migration statistics:
  - Total node count: `MATCH (n) RETURN count(n)`
  - Total edge count: `MATCH ()-[r]->() RETURN count(r)`
  - Node counts by label: `MATCH (n) RETURN labels(n), count(n)`
  - Edge counts by type: `MATCH ()-[r]->() RETURN type(r), count(r)`
- Verify dump file integrity:
  - Check file is not corrupted (can be read by neo4j-admin)
  - Backup dump file to cloud storage for disaster recovery

#### Task 11.3: Import to Aura
- Upload dump file to Aura instance:
  - Use Aura console UI for upload (< 2GB files)
  - For larger files, use neo4j-admin import command
- Restore database from dump:
  - Aura will process dump and restore all nodes/edges
  - Monitor restore progress in Aura console
  - Restore may take 10-30 minutes for 50K nodes
- Verify import success:
  - Check node counts match pre-migration: `MATCH (n) RETURN count(n)`
  - Check edge counts match: `MATCH ()-[r]->() RETURN count(r)`
  - Sample query verification: retrieve specific nodes by ID
- Test query performance:
  - Run scenario detection queries
  - Run blast radius queries
  - Measure query times: should be comparable to local Neo4j

#### Task 11.4: Update Application Configuration
- Update `.env` with Aura connection details:
  - NEO4J_URI: bolt+s://<aura-instance>.databases.neo4j.io
  - NEO4J_USERNAME: neo4j
  - NEO4J_PASSWORD: <generated-password>
- Update connection driver settings:
  - Enable TLS/SSL (required for Aura)
  - Configure connection pool size (recommend: 50)
  - Set connection timeout and retry policies
- Test `crisk check` with cloud Neo4j:
  - Run on sample files from test repository
  - Verify scenario queries return expected results
  - Check file resolution queries work correctly
  - Confirm no connection errors or TLS issues
- Measure performance:
  - Compare query latency: local Neo4j vs Aura
  - Should be within 10-20% (network overhead acceptable)
  - Optimize slow queries if needed (add indexes on node properties)

#### Task 11.5: Configure Aura Security and Monitoring
- Security configuration:
  - Enable IP allowlist if needed (restrict to application server IPs)
  - Rotate default password to strong generated password
  - Configure SSL certificate pinning for enhanced security
- Set up monitoring and alerts:
  - Query performance monitoring (slow query alerts > 1 second)
  - Connection pool usage alerts (> 80% utilization)
  - Memory usage monitoring (alert if approaching limits)
- Configure backup schedule:
  - Aura provides automated daily backups (enabled by default)
  - Verify backups are being created
  - Test restore procedure from backup snapshot
  - Document backup retention policy (Aura: 7-day retention)

**Deliverable:** Cloud Neo4j Aura operational with graph data migrated, application connected and tested, security configured

---

### Week 12: Implement Scheduled Scenario Detection

**Objective:** Automate daily scenario detection to keep firefighting intelligence up-to-date

#### Task 12.1: Create Standalone Scenario Detector Binary
- Extract scenario detection from `crisk init` into new command:
  - Create `cmd/scenario-detector/main.go`
  - Import scenario detection, PR classification, link resolution logic
  - Accept CLI parameters: repo_id, neo4j_uri, postgres_dsn, gemini_api_key, lookback_days
- Implement focused workflow:
  1. Fetch new PRs merged in last N days (incremental update)
  2. Classify only new PRs (skip already classified)
  3. Detect new scenarios (compare with existing scenarios)
  4. Update firefighting metrics
  5. Generate JSON report
- Output structured JSON report:
  - New scenarios detected (count, severity distribution)
  - Updated firefighting score
  - Top 3 new high-severity scenarios with narratives
  - Metrics trends (score change, hotfix rate change)
- Add --dry-run flag for testing without database writes
- Implement comprehensive logging for debugging

#### Task 12.2: Design Cloud Execution Strategy
- Evaluate execution options:
  - **Option A: GitHub Actions**
    - Pros: Free for public repos, easy setup, GitHub-native
    - Cons: 6-hour max runtime, no persistent infrastructure
  - **Option B: Cloud Function (AWS Lambda, GCP Cloud Run)**
    - Pros: Pay-per-execution, auto-scaling, persistent infrastructure optional
    - Cons: Setup complexity, cold start latency, vendor lock-in
  - **Option C: Kubernetes CronJob**
    - Pros: Full control, predictable performance, existing k8s infrastructure
    - Cons: Operational overhead, requires k8s cluster
- Recommended: **Option A (GitHub Actions)** for MVP, transition to Option B at scale
- Selection criteria:
  - Cost: GitHub Actions free tier is sufficient for 5-10 repos
  - Ease of maintenance: minimal DevOps overhead
  - Existing infrastructure: no new services to manage

#### Task 12.3: Implement GitHub Actions Workflow
- Create workflow file: `.github/workflows/scenario_detection.yml`
- Configure schedule trigger:
  - Cron: `0 0 * * *` (daily at midnight UTC)
  - Manual trigger: workflow_dispatch for on-demand runs
- Define job steps:
  1. Checkout repository code
  2. Set up Go environment (version 1.21+)
  3. Build scenario-detector binary
  4. Run detector with secrets:
     - NEO4J_AURA_URI, NEON_DSN, GEMINI_API_KEY from GitHub secrets
     - REPO_ID from workflow input or configuration file
  5. Upload results artifact (JSON report) for historical tracking
  6. Send notification on completion (success or failure)
- Configure secrets in GitHub repository settings:
  - NEO4J_AURA_URI: Aura bolt+s:// connection string
  - NEO4J_PASSWORD: Aura password
  - NEON_DSN: Postgres connection string
  - GEMINI_API_KEY: Gemini API key
- Add error handling:
  - Retry logic for transient failures (network timeouts)
  - Failure notifications via GitHub Actions status
  - Detailed error logs in workflow output

#### Task 12.4: Build Notification System
- Detect new high-severity scenarios:
  - Query for scenarios created in last 24 hours
  - Filter for P1 and P2 severity only
  - Compare with previous run's scenario count (track in postgres)
- Implement notification channels:
  - **Slack webhook:**
    - Send message to #coderisk-alerts channel
    - Include: new scenario count, top scenario narrative, link to report
  - **Email:**
    - Send to team distribution list
    - Subject: "New P1 Scenario Detected in {repo_name}"
  - **GitHub issue creation:**
    - Create issue in repository with scenario details
    - Assign to relevant developers (based on file ownership)
    - Label: "firefighting", "automated-alert"
- Notification content:
  - Scenario narrative (human-readable explanation)
  - Trigger PR and follow-up PRs (linked for easy navigation)
  - Affected files and suggested actions
  - Comparison to previous metrics (score trend)
- Implement notification throttling:
  - Only notify on new P1 scenarios (prevent alert fatigue)
  - Daily digest for P2 scenarios instead of per-scenario alerts
  - Configurable thresholds per repository

#### Task 12.5: Create Historical Tracking
- Design postgres table `scenario_detection_runs`:
  - id, repo_id, run_date, scenarios_detected, firefighting_score
  - new_scenarios (JSON array of new scenario IDs)
  - metrics_snapshot (JSON with revert_rate, hotfix_rate, etc.)
  - runtime_seconds, status (success/failed), error_message
- Store daily detection results:
  - After each run, insert row with metrics snapshot
  - Enable time-series analysis of firefighting trends
- Build trend analysis queries:
  - Firefighting score over time (30-day rolling average)
  - New scenarios per day (detect spikes in firefighting activity)
  - Resolved scenarios (scenarios not detected in recent runs)
- Create simple dashboard query:
  - Top 5 repositories by firefighting score
  - Score change week-over-week (identify improving/degrading repos)
  - Total scenarios by severity (P1/P2/P3 distribution)
- Implement retention policy:
  - Keep detailed runs for 90 days
  - Archive summary stats for 1 year
  - Delete raw data after 1 year

**Deliverable:** Automated daily scenario detection running in cloud with notifications, historical tracking, and trend analysis

---

## Phase 4: Beta Testing & Iteration (Ongoing)

### Task 13.1: Select Beta Test Targets
- Choose 5 repositories from Repo Sniper's Priority 1 list:
  - **Strapi** - Score: 176.0, 48% hotfix rate, 8 panic spirals
  - **Windmill** - Score: 169.4, 34.7% hotfix rate, 10 panic spirals
  - **Plane** - Score: 138.0, 39% hotfix rate, 6 panic spirals
  - **Formbricks** - Score: 137.2, 51.4% hotfix rate, 2.9% revert rate
  - **Directus** - Score: 110.0, 20% hotfix rate, 7 panic spirals
- Selection criteria:
  - High firefighting scores (clear pain points)
  - Active development (recent commits/PRs)
  - Responsive maintainers (based on issue response times)
  - Open source with public GitHub repos (easy onboarding)
- Prepare beta outreach:
  - Personalized email using Repo Sniper template
  - Include specific smoking gun evidence (e.g., "PR #24706 caused 13 fixes")
  - Offer free beta access in exchange for feedback

### Task 13.2: Deploy for Beta Testers
- **Option A: CLI Installation**
  - Provide installation instructions: `curl -sSL https://coderisk.dev/install.sh | sh`
  - Guide through `crisk init` setup for their repository
  - Explain configuration: API keys, database connections
- **Option B: Hosted Web Interface**
  - Deploy web UI for viewing scenarios without local installation
  - GitHub OAuth integration for easy repo access
  - Pre-computed scenarios displayed in dashboard
- Run initial `crisk init` for each beta repository:
  - Generate baseline scenarios and metrics
  - Create firefighting report (PDF or markdown)
  - Highlight top 3 scenarios with specific PR evidence
- Share actionable insights:
  - "Your reviewer tax is 51% - here's how to reduce it"
  - "File X has 5 P1 incidents - recommend refactoring"
  - "Developer Y is the bus factor for module Z"
- Schedule onboarding call:
  - Walk through report findings
  - Explain how to use `crisk check` in workflow
  - Demonstrate pre-commit hook integration

### Task 13.3: Collect Feedback
- Weekly check-ins with beta testers via:
  - Email surveys (structured questions + open feedback)
  - Video calls (30-minute demos and discussions)
  - Slack channel (#coderisk-beta) for async communication
- Key feedback questions:
  - **Scenario accuracy:** Are narratives describing real incidents?
  - **Confidence scores:** Do you trust the 0.7-0.95 confidence levels?
  - **Missing patterns:** Any firefighting issues we didn't detect?
  - **Output format:** Is the report actionable? Too verbose? Missing context?
  - **Integration:** Does `crisk check` fit into your workflow?
  - **Performance:** Is `crisk init` fast enough? Any timeouts?
- Track adoption metrics:
  - Number of `crisk check` runs per day
  - Number of developers using the tool
  - Pre-commit hook installation rate
  - Scenario report views (web UI)
- Identify friction points:
  - Installation difficulties (API key setup, database configuration)
  - False positive scenarios causing alert fatigue
  - Missing integrations (Jira, Slack, CI/CD tools)

### Task 13.4: Iterate on Narrative Quality
- Refine LLM prompts based on feedback:
  - If narratives are too vague: add more context (file changes, code snippets)
  - If narratives are too verbose: simplify to 1-2 sentences
  - If narratives miss causality: emphasize "why" over "what"
- Add more context to narratives:
  - **Specific file changes:** "Modified authentication logic in src/auth.ts lines 45-67"
  - **Code snippets:** Show diff of problematic change
  - **Stack traces from issues:** Include error messages that triggered hotfixes
  - **Timeline visualization:** "24 hours: feature merged → login crashes → 2 hotfixes deployed"
- Improve scenario severity classification:
  - Calibrate P1/P2/P3 thresholds based on feedback
  - Example: If 3-fix cluster feels like P2 not P1, adjust algorithm
  - Add severity justification to narrative: "P1 severity due to production outage"
- Test narrative improvements:
  - A/B test new prompts vs old on 5 repositories
  - Survey beta testers on narrative quality improvement
  - Measure: "How often are narratives accurate?" (target: 90%+)

### Task 13.5: Measure Impact
- Track behavioral changes after 30/60/90 days of usage:
  - **Hotfix rate reduction:** Did teams reduce reactive bug fixing?
  - **Revert rate reduction:** Fewer production rollbacks?
  - **Scenario recurrence:** Same files triggering new scenarios?
  - **Pre-commit usage:** How often do developers check files before committing?
- Collect testimonials and case studies:
  - "CodeRisk helped us catch a critical bug before production" - Developer X, Company Y
  - "Our hotfix rate dropped from 40% to 28% in 60 days" - Engineering Manager, Company Z
  - "The panic spiral detection saved us from a cascading failure" - SRE, Company W
- Measure quantitative impact:
  - Average MTTR reduction (if incidents still occur, are they resolved faster?)
  - MTBF improvement (longer periods between incidents?)
  - Developer velocity (are teams shipping features faster with fewer regressions?)
- Document success stories:
  - Specific scenarios that prevented incidents
  - Files that were refactored due to high scenario density
  - Teams that changed development practices based on insights
- Calculate ROI for beta customers:
  - Estimated cost of prevented incidents (based on MTTR and engineer costs)
  - Time saved in code review (reviewer tax reduction)
  - Revenue protection from avoided downtime

**Deliverable:** Validated product-market fit with 5 beta customers providing positive feedback, testimonials, and measurable impact

---

## Success Criteria

### Technical Metrics

**Link Resolution Pipeline:**
- ≥30% improvement in Issue→PR/Commit link coverage compared to timeline events alone
- ≥90% precision on ASSOCIATED_WITH edges (validated by manual audit of 100 random samples)
- Confidence scores correlate with validation success rate (0.9 confidence = 90%+ precision)

**Scenario Detection Pipeline:**
- ≥80% accuracy vs Repo Sniper ground truth on:
  - Overall firefighting score (±10% deviation)
  - Incident cluster identification (70%+ overlap in top 5 clusters)
  - Panic spiral detection (detect 80%+ of Repo Sniper's spirals)
- ≥85% narrative quality rating from beta testers ("useful" or "very useful")

**Performance:**
- `crisk init` completes in < 10 minutes for repositories with 1000 PRs
- `crisk check` returns results in < 5 seconds for single file analysis
- Scenario detection pipeline processes 100 PRs in < 2 minutes

**Cost:**
- < $0.30 per repository analysis using Gemini Flash (reduced from $0.50 baseline)
- 60-80% cache hit rate on repeated `crisk init` runs
- Total monthly LLM cost < $50 for 10 active repositories

**Cloud Uptime:**
- 99.5% availability for PostgreSQL (Neon/Supabase SLA)
- 99.9% availability for Neo4j Aura
- < 100ms average query latency to cloud databases (p95 < 200ms)

---

### Product Metrics

**Validation:**
- Match Repo Sniper results within ±10% on firefighting score for all 14 test repositories
- Detect ≥90% of Repo Sniper's identified "smoking gun" incident clusters
- Zero critical bugs in production (P0 issues blocking core functionality)

**Coverage:**
- Successfully analyze ≥90% of targeted COSS repositories (minimize failures like Repo Sniper's 30% failure rate)
- Support for repositories with 100-10,000 PRs (wide range)
- Handle multi-language repositories (JavaScript, TypeScript, Python, Go)

**Insight Quality:**
- Beta testers rate scenario narratives as "useful" or "very useful" ≥80% of the time
- < 10% false positive rate on high-severity scenarios (P1 and P2)
- ≥70% of scenarios lead to actionable developer behavior (extra review, refactoring, expert consultation)

---

### Business Metrics

**GTM Validation:**
- 5 beta customers actively using CodeRisk for pre-commit risk assessment
- ≥3 testimonials from beta customers citing specific prevented incidents
- ≥2 case studies showing measurable hotfix rate reduction (target: 20%+ improvement)

**Paper Support:**
- Scenario detection directly demonstrates $R_{\text{temporal}}$ prediction from paper thesis
- Empirical validation of "Default-to-Trust" equilibrium (hotfix rates 20-50% prove reactive culture)
- Evidence of Reviewer Tax metric matching paper's cost model (51% firefighting effort)

**Competitive Differentiation:**
- Clear moat vs CodeRabbit/Greptile: they analyze content (syntax, style), we analyze context (temporal patterns, ownership, incidents)
- Unique value proposition validated: "We don't find bugs, we predict which changes will cause bugs"
- Positioning confirmed: Preparation phase tool (pre-commit) vs Detection phase (post-commit review) vs Recovery phase (incident response)

---

## Risk Mitigation

### Risk 1: LLM Hallucinations in Link Resolution

**Impact:** False positive ASSOCIATED_WITH edges corrupt graph quality and reduce trust
**Probability:** Medium (LLMs can hallucinate non-existent references)

**Mitigation Strategies:**
- **Multi-signal validation:** Require at least 2 of 3 validations (temporal, semantic, file overlap) to pass
- **Confidence threshold:** Set higher threshold at 0.8 instead of 0.7 if precision issues arise
- **Manual audit:** Randomly sample 100 ASSOCIATED_WITH edges per repository and verify accuracy
- **Evidence transparency:** Always store original text that triggered extraction for debugging
- **Comparison baseline:** Compare link resolution accuracy against timeline event coverage

**Monitoring:**
- Track false positive rate by manually auditing random samples weekly
- Alert if validation success rate drops below 80%
- User feedback mechanism: allow developers to report incorrect links

**Fallback Plan:**
- If precision < 80%, raise confidence threshold to 0.85
- If still problematic, disable link resolution for specific repositories
- Investigate prompt engineering improvements or switch LLM provider

---

### Risk 2: Scenario Detection Missing Patterns

**Impact:** Incomplete firefighting intelligence, missed high-risk areas
**Probability:** Medium (pattern detection depends on heuristics and thresholds)

**Mitigation Strategies:**
- **Conservative detection rules:** Start with high overlap (30%), short time windows (48hrs) to minimize false positives
- **Gradual threshold relaxation:** Based on beta feedback, carefully adjust thresholds
- **Ground truth validation:** Continuously compare against Repo Sniper's validated results
- **Multiple detection methods:** Run 3 independent detectors (context gaps, panic spirals, incident clusters) to catch different patterns
- **Transparency:** Always show confidence scores and evidence so users can judge validity

**Iteration Plan:**
- Week 1-2: Run with strict thresholds, measure recall (what % of Repo Sniper patterns we catch)
- Week 3-4: If recall < 80%, relax thresholds (25% overlap, 72hr window) and re-measure
- Week 5-8: A/B test threshold variations on 10 repositories, find optimal balance
- Ongoing: Add new detection patterns based on beta feedback (e.g., "flaky test spiral")

**Monitoring:**
- Track detection recall vs Repo Sniper ground truth weekly
- Survey beta testers: "Did we miss any major firefighting issues?"
- Log scenarios that don't trigger notifications (potential false negatives)

**Fallback Plan:**
- If recall < 70%, expand detection patterns (add new scenario types)
- If precision < 70%, tighten thresholds or add validation steps
- Provide manual scenario creation API for expert users

---

### Risk 3: Cloud Migration Downtime

**Impact:** Service interruption during postgres/Neo4j migration, data loss risk
**Probability:** Low (well-documented migration procedures exist)

**Mitigation Strategies:**
- **Phased migration:** Migrate postgres and Neo4j separately (keep one local while testing other)
- **Parallel systems:** Run local and cloud databases in parallel for 1 week before cutover
- **Rollback readiness:** Maintain local database backups for 30 days after migration
- **Testing period:** Extensive testing on cloud systems before deprecating local
- **Incremental cutover:** Migrate low-traffic repositories first, then scale to high-traffic

**Migration Plan:**
- Week 10: Migrate postgres to Neon, keep Neo4j local, test for 3 days
- Week 11: Migrate Neo4j to Aura, keep postgres cloud, test for 3 days
- Week 12: Full cloud operation, monitor for 1 week before deprecating local
- Week 13+: Delete local databases only after 30-day retention period

**Monitoring:**
- Real-time latency monitoring during migration (alert if p95 > 500ms)
- Data integrity checks after each migration step (row counts, constraint validation)
- User impact tracking (any errors reported during migration window?)

**Rollback Plan:**
- If critical issues arise, revert to local databases using 30-day backups
- Update application config to point back to local connection strings
- Restore from cloud backups if local backups are corrupted
- Document rollback procedure and test quarterly

---

### Risk 4: Cost Overruns on LLM Usage

**Impact:** Unexpected expenses exceed budget, force service degradation
**Probability:** Medium (token usage can spike with large repositories or frequent re-runs)

**Mitigation Strategies:**
- **Aggressive caching:** 30-day TTL on classifications and links (60-80% cache hit rate reduces cost)
- **Budget alerts:** Set up Gemini API spending alerts at $50, $100, $200 monthly thresholds
- **Prompt compression:** Optimize prompts to reduce token usage by 30-50%
- **Batch processing:** Larger batches reduce API call overhead (100 PRs per call vs 20)
- **Tiered service:** Offer free tier with limited analyses per month, paid tier for unlimited

**Cost Monitoring:**
- Daily tracking of token usage per repository in postgres
- Weekly cost reports broken down by: link resolution, classification, narrative generation
- Monthly review of top 10 most expensive repositories (identify outliers)

**Optimization Plan:**
- Week 1-4: Baseline measurement (current cost per repo: ~$0.50)
- Week 5-6: Implement caching (target: $0.35 per repo)
- Week 7-8: Optimize prompts (target: $0.25 per repo)
- Ongoing: Monitor and adjust batch sizes, prompt length, cache TTLs

**Fallback Plan:**
- If monthly costs > $200, pause analyses for low-priority repositories
- If costs > $500, disable narrative generation (most expensive operation) and use templates
- If costs > $1000, switch to smaller LLM model (Gemini Flash 1.5 vs 2.0) or reduce batch sizes
- Emergency: Disable link resolution and rely only on timeline events (eliminates major cost)

---

## Deliverables Summary

1. **Week 1:** Validation report on 100% confidence graph coverage (baseline: X% timeline event coverage, Y nodes, Z edges)
2. **Week 2:** Link resolution pipeline integrated into `crisk init` (improvement: +30% coverage, confidence scores 0.7-0.95)
3. **Week 3:** PR classification pipeline producing Repo Sniper-compatible categorizations (FEATURE/HOTFIX/REVERT/CHORE taxonomy)
4. **Week 4:** Scenario detection pipeline creating graph-based firefighting intelligence (3 detection patterns: context gaps, panic spirals, incident clusters)
5. **Week 5:** Firefighting metrics calculation aligned with Repo Sniper (revert rate, hotfix rate, regression ratio, reviewer tax, firefighting score)
6. **Week 6:** `crisk check` surfacing scenario-based temporal risk signals (file incident history, developer ownership, actionable guidance)
7. **Week 7:** Validation report proving accuracy vs Repo Sniper ground truth (14 repositories, ±10% score accuracy, 70%+ cluster overlap)
8. **Week 8:** Cost-optimized LLM pipeline with performance documentation (30-50% token reduction, 60-80% cache hit rate, < $0.30 per repo)
9. **Week 9:** TreeSitter-free system validated on test repositories (co-change blast radius, simplified architecture, updated docs)
10. **Week 10:** Cloud PostgreSQL operational with data migrated (Neon/Supabase, automated backups, monitoring configured)
11. **Week 11:** Cloud Neo4j Aura operational with graph migrated (bolt+s:// connection, performance validated, security configured)
12. **Week 12:** Automated daily scenario detection with notifications (GitHub Actions workflow, Slack/email alerts, historical tracking)
13. **Ongoing:** Beta testing with 5 customers and iterative improvements (testimonials, impact measurement, narrative quality optimization)

---

## Next Steps After Plan Approval

1. **Week 1 Kickoff:** Run validation queries on current omnara-ai/omnara graph, document baseline metrics
2. **Set up tracking:** Create project board with tasks for each week, assign owners, set deadlines
3. **Communicate plan:** Share this document with engineering team, align on priorities and timeline
4. **Begin execution:** Start with Week 1 Task 1.1 (verify timeline edge implementation)
5. **Weekly reviews:** Every Friday, review completed tasks, identify blockers, adjust timeline if needed

---

**Document Control:**
- **Version:** 1.0
- **Last Updated:** 2025-11-14
- **Owner:** Rohan Katakam
- **Status:** Approved for Execution
- **Review Cycle:** Monthly (re-evaluate timeline and priorities)
