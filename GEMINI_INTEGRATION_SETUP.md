# CodeRisk Gemini Integration & Directive Agent System - Setup Guide

## Table of Contents
1. [Overview](#overview)
2. [What We've Built](#what-weve-built)
3. [Current State](#current-state)
4. [Environment Setup](#environment-setup)
5. [Database Setup](#database-setup)
6. [Running CodeRisk](#running-coderisk)
7. [Testing Workflow](#testing-workflow)
8. [What's Working](#whats-working)
9. [Critical Gaps to Fix](#critical-gaps-to-fix)
10. [Goal & Intent](#goal--intent)
11. [Test Cases](#test-cases)
12. [Next Steps](#next-steps)

---

## Overview

This project integrates **Google Gemini API** as the primary LLM provider for CodeRisk's Phase 2 agent-based risk investigation system. We're building toward a **12-factor agent architecture** with human-in-the-loop workflows, but currently have a fully autonomous agent-to-agent system.

**Key Repository:** Testing on [omnara-ai/omnara](https://github.com/omnara-ai/omnara) with ground truth data

---

## What We've Built

### 1. Multi-Provider LLM Architecture
- **Files Created:**
  - `internal/llm/gemini_client.go` - Gemini API client (Complete, CompleteJSON, CompleteWithTools)
  - Modified `internal/llm/client.go` - Multi-provider support (Gemini/OpenAI)
  - Modified `internal/config/config.go` - Added GeminiKey and GeminiModel config

- **Features:**
  - Provider selection: env var > config > default to Gemini
  - Fast model: `gemini-2.0-flash` (2K RPM, production-ready)
  - Deep model: `gemini-1.5-pro` (for complex synthesis)
  - Native JSON mode: `ResponseMIMEType: "application/json"`

### 2. Directive Agent Infrastructure (UNUSED - Critical Gap)
- **Files Created:**
  - `internal/agent/directive_types.go` - Decision point types (DirectiveMessage, DirectiveAction, UserOption)
  - `internal/agent/directive_builder.go` - Helper functions (BuildContactHumanDirective, BuildDeepInvestigationDirective, BuildEscalationDirective)
  - `internal/agent/directive_display.go` - Terminal UI for displaying directives
  - `internal/agent/conversation_state.go` - State management (DirectiveInvestigation, terminal states)
  - `internal/agent/checkpoint_store.go` - PostgreSQL CRUD for investigation checkpoints

- **Terminal States:**
  - `SafeToCommit` - Low risk, safe to proceed
  - `RisksUnresolved` - Risks identified, needs attention
  - `BlockedWaiting` - Waiting for human input
  - `InvestigationIncomplete` - Investigation could not complete
  - `InvestigationAborted` - User aborted investigation

### 3. PostgreSQL Checkpoint Storage (UNUSED - Critical Gap)
- **Migration:** `scripts/schema/migrations/003_add_investigation_checkpoints.sql`
- **Table:** `investigations` with JSONB fields for flexible state storage

---

## Current State

### ✅ Working:
1. **Gemini Integration:** All LLM calls use gemini-2.0-flash successfully
2. **Phase 2 Investigations:** Agent-based risk analysis runs end-to-end
3. **Risk Stratification:** Correctly classifies LOW (70-90% confidence) vs MEDIUM (70% confidence)
4. **Adaptive Investigation:** 3-5 hops based on complexity
5. **Zero Rate Limits:** gemini-2.0-flash has 2K RPM (vs 10 RPM for experimental)

### ❌ Not Working:
1. **Co-change Partner Queries:** All investigations fail to retrieve co-change data (critical gap)
2. **Human Decision Points:** Agents never pause for user input (violates 12-factor Factor #6)
3. **Directive System:** Built infrastructure is completely unused (violates 12-factor Factor #7)
4. **Checkpoint Storage:** PostgreSQL investigations table never written to
5. **Agent-to-Human Communication:** System is purely agent-to-agent (should be agent-to-human)

---

## Environment Setup

### Prerequisites
- Go 1.21+
- Docker & Docker Compose (for Neo4j and PostgreSQL)
- Git

### 1. Clone and Build

```bash
cd /Users/rohankatakam/Documents/brain/coderisk

# Build the binary
make build

# Binary location
ls -la /Users/rohankatakam/Documents/brain/coderisk/bin/crisk
```

### 2. Start Docker Services

```bash
# Start Neo4j and PostgreSQL
docker compose up -d

# Verify services are running
docker ps

# Expected output:
# - neo4j on ports 7475 (HTTP), 7688 (Bolt)
# - postgres on port 5433
```

### 3. API Keys

**Gemini API Key** (Required):
```
AIzaSyAnkF7s3RLV5wVYLhxCRVnI2HrxVUK7zzU
```

**GitHub Token** (Required for repo ingestion):
```
GITHUB_TOKEN_HERE
```

**Rate Limits:**
- gemini-2.0-flash: **2,000 requests/minute** (paid tier, $100 credit available)
- gemini-2.0-flash-exp: 10 requests/minute (free tier, DO NOT USE)

---

## Database Setup

### PostgreSQL

**Connection Details:**
- Host: `localhost`
- Port: `5433`
- Database: `coderisk`
- User: `coderisk`
- Password: `CHANGE_THIS_PASSWORD_IN_PRODUCTION_123`
- DSN: `postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable`

**Key Tables:**
1. **repositories** - Stores repository metadata
2. **github_issues** - Issue data with LLM classifications
3. **github_prs** - Pull request data
4. **github_commits** - Commit history
5. **issue_pr_links** - Links between issues and PRs (ground truth)
6. **investigations** ⚠️ **UNUSED** - Checkpoint storage for directive investigations

**Connect to PostgreSQL:**
```bash
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk
```

**Useful Queries:**
```sql
-- Check repository ingestion
SELECT id, full_name, default_branch, updated_at FROM repositories;

-- Check issue count and classifications
SELECT
    COUNT(*) as total_issues,
    COUNT(*) FILTER (WHERE state = 'closed') as closed_issues,
    COUNT(*) FILTER (WHERE llm_classification IS NOT NULL) as classified_issues
FROM github_issues;

-- Check issue-PR links
SELECT
    i.number as issue_num,
    i.title as issue_title,
    p.number as pr_num,
    p.title as pr_title,
    l.confidence
FROM issue_pr_links l
JOIN github_issues i ON l.issue_id = i.id
JOIN github_prs p ON l.pr_id = p.id
ORDER BY l.confidence DESC
LIMIT 10;

-- Check investigations table (should be empty currently)
SELECT COUNT(*) FROM investigations;
```

### Neo4j

**Connection Details:**
- URI: `bolt://localhost:7688`
- Username: `neo4j`
- Password: `CHANGE_THIS_PASSWORD_IN_PRODUCTION_123`
- HTTP: `http://localhost:7475`

**Key Node Types:**
- `File` - Source code files
- `Developer` - Contributors
- `Commit` - Git commits
- `Issue` - GitHub issues
- `PullRequest` - GitHub PRs

**Connect to Neo4j:**
```bash
# Via HTTP API
curl -s -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  -H "Content-Type: application/json" \
  -X POST http://localhost:7475/db/neo4j/tx/commit \
  -d '{"statements":[{"statement":"MATCH (f:File) RETURN count(f) as count"}]}'
```

**Useful Queries:**
```cypher
// Count nodes by type
MATCH (n)
RETURN labels(n)[0] as type, count(*) as count
ORDER BY count DESC

// Check file nodes
MATCH (f:File)
RETURN f.path, f.language
LIMIT 10

// Check developer ownership
MATCH (d:Developer)-[:AUTHORED]->(c:Commit)-[:MODIFIED]->(f:File)
WHERE f.path CONTAINS 'ChatMessage.tsx'
RETURN d.email, count(c) as commits
ORDER BY commits DESC
```

---

## Running CodeRisk

### Complete Environment Setup (Copy-Paste Ready)

```bash
# Export all required environment variables
export GEMINI_API_KEY="AIzaSyAnkF7s3RLV5wVYLhxCRVnI2HrxVUK7zzU"
export LLM_PROVIDER="gemini"
export PHASE2_ENABLED="true"
export GITHUB_TOKEN="GITHUB_TOKEN_HERE"

# Neo4j configuration
export NEO4J_URI="bolt://localhost:7688"
export NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"

# PostgreSQL configuration
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="5433"
export POSTGRES_DB="coderisk"
export POSTGRES_USER="coderisk"
export POSTGRES_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
export POSTGRES_DSN="postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable"
```

### crisk init (Repository Ingestion)

**Purpose:** Ingest a GitHub repository into Neo4j and PostgreSQL for analysis

**Test Repository:** https://github.com/omnara-ai/omnara

```bash
# Navigate to test repository
cd /tmp/omnara

# Run crisk init (takes ~5-10 minutes for Omnara)
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk init

# Expected output:
# - Ingests 421 files, 156 commits, 81 issues, 128 PRs
# - LLM classifies 43 closed issues
# - Creates 19 deep links with confidence scores
# - Exit code 0 on success
```

**Key Steps in crisk init:**
1. **GitHub Data Fetch:** Clone repo, fetch issues/PRs/commits via GitHub API
2. **Neo4j Ingestion:** Build code graph (files, commits, developers)
3. **PostgreSQL Ingestion:** Store issues, PRs, commits in relational format
4. **LLM Classification:** Use Gemini to classify issue types (fixed_with_code, user_action_required, wontfix, duplicate, unclear, not_a_bug)
5. **Issue-PR Linking:** Multi-signal ground truth classification

**Logs to Monitor:**
```bash
# Watch for Gemini initialization
grep "gemini client initialized"

# Check issue processing
grep "processing issue"

# Verify completion
echo $?  # Should be 0
```

### crisk check (Risk Analysis)

**Purpose:** Analyze risk of file changes using agent-based investigation

```bash
# Single file check with explanation
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk check \
  apps/web/src/components/dashboard/chat/ChatMessage.tsx \
  --explain

# Multiple files check
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk check \
  apps/web/src/components/dashboard/CommandPalette.tsx \
  apps/web/src/components/dashboard/SidebarDashboardLayout.tsx \
  --explain
```

**Expected Output Structure:**
```
🔍 File Resolution: Found 2 historical paths
   - exact match (100% confidence)
   - git-follow rename (95% confidence)

🔍 Phase 1 Metrics:
   Risk Level: HIGH
   Recommendations: Add test coverage, Investigate temporal coupling

🔍 Escalating to Phase 2 (Agent-based investigation)...

🔍 Phase 2 Investigation:
   Started: 2025-11-09 19:15:00
   Completed: 2025-11-09 19:15:10 (10.0s)
   Agent hops: 3-5

   Hop 1: query_incident_history, query_ownership, query_cochange_partners, query_blast_radius
   Hop 2: query_recent_commits, get_blast_radius_analysis
   Hop 3: [synthesis]

📊 Final Assessment:
   Risk Level: LOW (confidence: 90%)
   Summary: [Detailed reasoning with evidence]

   Investigation completed in 10.0s (3 hops, 6556 tokens)
```

---

## Testing Workflow

### Test Repository Setup

```bash
# Clone fresh Omnara repository for testing
cd /tmp
rm -rf omnara
git clone https://github.com/omnara-ai/omnara.git
cd omnara

# Set environment (copy-paste from "Complete Environment Setup" above)
export GEMINI_API_KEY="AIzaSyAnkF7s3RLV5wVYLhxCRVnI2HrxVUK7zzU"
export LLM_PROVIDER="gemini"
export PHASE2_ENABLED="true"
export GITHUB_TOKEN="GITHUB_TOKEN_HERE"
export NEO4J_URI="bolt://localhost:7688"
export NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="5433"
export POSTGRES_DB="coderisk"
export POSTGRES_USER="coderisk"
export POSTGRES_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
export POSTGRES_DSN="postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable"

# Run init
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk init
```

### Clean Database for Fresh Test

```bash
# Stop services
cd /Users/rohankatakam/Documents/brain/coderisk
docker compose down

# Remove volumes (WARNING: Deletes all data)
docker volume rm coderisk_neo4j_data coderisk_postgres_data

# Restart services
docker compose up -d

# Wait for services to be ready (~10 seconds)
sleep 10

# Verify services
docker ps
```

---

## What's Working

### 1. Gemini Integration ✅
- **Model:** gemini-2.0-flash (2K RPM, production-ready)
- **Zero Rate Limits:** No 429 errors in any test
- **Logs Confirm:**
  ```
  INFO gemini client initialized component=gemini model=gemini-2.0-flash
  ```

### 2. Phase 2 Investigation Flow ✅
- **Autonomous Agent Execution:** RiskInvestigator runs 3-5 hops
- **Node Visits:** query_incident_history, query_ownership, query_cochange_partners, query_blast_radius, query_recent_commits, get_blast_radius_analysis
- **Token Usage:** 6,353 - 11,355 tokens per investigation
- **Duration:** 9.5s - 14.9s per file

### 3. Risk Assessment ✅
- **Stratification:** LOW (80-90% confidence) vs MEDIUM (70% confidence)
- **Evidence-Based:** Checks incident history, ownership, co-change patterns, blast radius
- **Adaptive:** More hops (5) for complex cases like SidebarDashboardLayout

### 4. Test Cases Executed ✅

| Test Case | File | Change Type | Risk | Confidence | Hops | Tokens |
|-----------|------|-------------|------|------------|------|--------|
| #122 - JSON Parsing Bug | ChatMessage.tsx | Bug fix | LOW | 90% | 3 | 6,556 |
| #115 - Command Handler | CommandPalette.tsx | Bug fix | MEDIUM | 70% | 3 | 6,353 |
| #187 - Mobile Sync | SidebarDashboardLayout.tsx | Bug fix | LOW | 80% | 5 | 11,355 |
| #160 - Agent Naming | LaunchAgentModal.tsx | Feature | LOW | 90% | 4 | 8,398 |

---

## Critical Gaps to Fix

### 🔴 Priority 1: Co-change Partner Query Failures

**Problem:** All 4 test cases show:
```
error when checking co-change partners
```

**Impact:** Missing critical risk signal - files that frequently change together

**Investigation Needed:**
1. Check Neo4j schema for co-change relationships
2. Review query in `internal/agent/investigator.go` (query_cochange_partners node)
3. Verify data exists:
   ```cypher
   MATCH (f1:File)-[:CO_CHANGED_WITH]->(f2:File)
   RETURN count(*) as cochange_count
   ```

**Fix Location:** Likely in `internal/agent/investigator.go` or `internal/graph/queries.go`

### 🔴 Priority 2: No Human Decision Points

**Problem:** Agents never pause for human input

**12-Factor Violation:** Factor #6 (Launch/Pause/Resume) and Factor #7 (Contact Humans with Tools)

**Expected Behavior:**
- MEDIUM risk (70% confidence) → Should trigger `BuildDeepInvestigationDirective`
- Missing co-change data → Should trigger `BuildContactHumanDirective`
- Recent incidents → Should trigger `BuildEscalationDirective`

**Implementation Needed:**
1. Add decision logic in `internal/agent/investigator.go` after final risk assessment
2. Generate DirectiveMessage when conditions met
3. Display directive using `directive_display.go`
4. Save checkpoint using `checkpoint_store.go`
5. Wait for user input (CLI prompt)
6. Resume investigation based on user choice

**Example Decision Logic (Pseudocode):**
```go
// In investigator.go after final risk calculation
if riskLevel == "MEDIUM" || confidence < 0.75 {
    directive := BuildDeepInvestigationDirective(
        estimatedTime: "2-3 minutes",
        estimatedCost: 0.05,
        reasoning: "Risk level is MEDIUM with uncertainty in co-change patterns",
        evidence: gatherEvidence(),
    )

    // Display directive to user
    DisplayDirective(directive)

    // Save checkpoint
    checkpoint := NewDirectiveInvestigation(files, repoID)
    checkpoint.AddDecision(directive, userChoice, userResponse)
    checkpointStore.Save(ctx, checkpoint)

    // Wait for user input
    userChoice := promptUser(directive.UserOptions)

    if userChoice == "ABORT" {
        checkpoint.SetTerminalState(InvestigationAborted, riskLevel, confidence, summary)
        return
    }
}
```

### 🔴 Priority 3: Checkpoint Storage Unused

**Problem:** PostgreSQL `investigations` table never written to

**Impact:** No pause/resume capability

**Verification:**
```sql
-- Should be 0 currently
SELECT COUNT(*) FROM investigations;
```

**Fix:** Integrate checkpoint_store.go into investigator.go workflow

### 🟡 Priority 4: Directive System Unused

**Problem:** All directive infrastructure exists but is never called

**Files Affected:**
- `internal/agent/directive_types.go` ✅ Created (unused)
- `internal/agent/directive_builder.go` ✅ Created (unused)
- `internal/agent/directive_display.go` ✅ Created (unused)
- `internal/agent/conversation_state.go` ✅ Created (unused)

**Fix:** Wire directive system into Phase 2 pipeline in `internal/linking/phase2.go`

---

## Goal & Intent

### Ultimate Goal: 12-Factor Agent Architecture

We are building a **human-in-the-loop risk assessment system** that aligns with [12-factor agent principles](https://github.com/humanlayer/12-factor-agents):

#### Current State (Autonomous):
```
User → crisk check → Phase 1 → Phase 2 Agent → Final Risk → User
                         ↓
                   (fully autonomous)
```

#### Target State (Human-in-the-Loop):
```
User → crisk check → Phase 1 → Phase 2 Agent → Decision Point
                                       ↓
                                 [PAUSE - Contact Human]
                                       ↓
                              Display Directive
                                       ↓
                              User Makes Choice
                                       ↓
                              Save Checkpoint
                                       ↓
                              Resume Investigation
                                       ↓
                              Final Risk → User
```

### Key Principles to Implement:

**Factor #2: Own Your Prompts** ✅ Already implemented
- Explicit prompts in agent kickoff (Step 3 logs)

**Factor #3: Own Your Context Window** ✅ Already implemented
- Structured context with file paths, metrics, git info

**Factor #6: Launch/Pause/Resume** ❌ **NOT IMPLEMENTED**
- Agents should pause for human decisions
- Investigations should be resumable via checkpoints
- Async workflow: agent → pause → human → resume

**Factor #7: Contact Humans with Tools** ❌ **NOT IMPLEMENTED**
- Agents should generate directives for user action
- Multiple channels: CLI prompts, Slack messages, email
- Decision points: APPROVE, MODIFY, SKIP, ABORT

### Use Cases for Human-in-the-Loop:

1. **Uncertain Risk (70% confidence)**
   - Directive: "Should I proceed with deep investigation? (Estimated cost: $0.05, time: 2-3 min)"
   - User options: APPROVE, SKIP, ABORT

2. **Missing Co-change Data**
   - Directive: "Unable to verify co-change patterns. Contact @ishaan to confirm related files?"
   - User options: CONTACT_HUMAN (generates Slack message), SKIP, PROCEED_WITH_CAUTION

3. **Recent Incidents**
   - Directive: "File caused 3 incidents in last 30 days. Escalate to @team-lead for review?"
   - User options: ESCALATE (creates Jira ticket), MODIFY (adjust thresholds), PROCEED

4. **MEDIUM/HIGH Risk**
   - Directive: "Change to CommandPalette.tsx is MEDIUM risk due to recent restructuring. Manual review recommended."
   - User options: MANUAL_REVIEW (pauses for human), AUTO_PROCEED (trust agent), ABORT

---

## Test Cases

### Ground Truth Data

**Source:** `/Users/rohankatakam/Documents/brain/coderisk/test_data/omnara_ground_truth_expanded.json`

**Key Test Cases:**
- Issue #122: JSON parsing bug (explicit link)
- Issue #115: Command handler bug (bidirectional link)
- Issue #187: Mobile interface sync (temporal + semantic)
- Issue #160: Agent naming feature (explicit link)
- Issue #189: Ctrl+Z bug (temporal + semantic, hard case)
- Issue #227: True negative (not a bug, user action required)

### Current Test Results

**Test Case 1: ChatMessage.tsx (Issue #122)**
```bash
cd /tmp/omnara
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk check \
  apps/web/src/components/dashboard/chat/ChatMessage.tsx --explain
```

**Result:**
- Risk: LOW (90% confidence)
- Duration: 9.5s, 3 hops, 6,556 tokens
- Findings: No incidents, active ownership, zero blast radius
- ⚠️ Co-change query failed

**Test Case 2: CommandPalette.tsx (Issue #115)**
```bash
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk check \
  apps/web/src/components/dashboard/CommandPalette.tsx --explain
```

**Result:**
- Risk: MEDIUM (70% confidence) ⚠️ **Should trigger directive**
- Duration: 14.9s, 3 hops, 6,353 tokens
- Findings: No incidents, single-dev ownership, recent restructuring noted
- ⚠️ Co-change query failed
- ❌ **No directive generated** (expected BuildDeepInvestigationDirective)

**Test Case 3 & 4: Multiple Files**
```bash
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk check \
  apps/web/src/components/dashboard/SidebarDashboardLayout.tsx \
  apps/web/src/components/dashboard/LaunchAgentModal.tsx \
  --explain
```

**Results:**
- SidebarDashboardLayout: LOW (80%), 10.0s, 5 hops, 11,355 tokens
- LaunchAgentModal: LOW (90%), 9.8s, 4 hops, 8,398 tokens
- ⚠️ Both co-change queries failed

### Test Modifications in Omnara Repo

**Files Modified for Testing:**
1. `apps/web/src/components/dashboard/chat/ChatMessage.tsx` - JSON parsing bug fix
2. `apps/web/src/components/dashboard/CommandPalette.tsx` - /clear and /reset command fix
3. `apps/web/src/components/dashboard/SidebarDashboardLayout.tsx` - Mobile sync fix
4. `apps/web/src/components/dashboard/LaunchAgentModal.tsx` - Agent naming feature

**Note:** These are test modifications to simulate the ground truth issue fixes. Revert before committing.

---

## Next Steps

### Immediate Actions (For Next Person)

#### 1. Fix Co-change Partner Query (Priority 1)
```bash
# Check if co-change relationships exist
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "SELECT COUNT(*) FROM file_cochanges;"

# Or in Neo4j
curl -s -u neo4j:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
  -H "Content-Type: application/json" \
  -X POST http://localhost:7475/db/neo4j/tx/commit \
  -d '{"statements":[{"statement":"MATCH (f1:File)-[:CO_CHANGED_WITH]->(f2:File) RETURN count(*) as count"}]}'
```

**Investigation Files:**
- `internal/agent/investigator.go` - Search for "query_cochange_partners"
- `internal/graph/queries.go` - Check co-change query implementation
- `internal/metrics/cochange.go` - Verify co-change calculation

**Expected Fix:** Add proper error handling or fix Cypher query

#### 2. Implement Human Decision Points (Priority 2)

**Target File:** `internal/linking/phase2.go` or `internal/agent/investigator.go`

**Add After Risk Calculation:**
```go
// After final risk level is determined
if riskLevel == "MEDIUM" || riskLevel == "HIGH" || confidence < 0.75 {
    // Build directive based on risk type
    var directive *agent.DirectiveMessage

    if riskLevel == "MEDIUM" {
        directive = agent.BuildDeepInvestigationDirective(
            "2-3 minutes",
            0.05,
            fmt.Sprintf("Risk is MEDIUM with %d%% confidence. Recent restructuring noted.", int(confidence*100)),
            gatherEvidence(inv),
        )
    } else if missingCoChangeData {
        directive = agent.BuildContactHumanDirective(
            ownerEmail,
            "Code Owner",
            filePath,
            "N/A",
            "Unable to verify co-change patterns. Please confirm related files were updated.",
            gatherEvidence(inv),
        )
    }

    // Display directive
    agent.DisplayDirective(directive)

    // Save checkpoint
    checkpointInv := agent.NewDirectiveInvestigation([]string{filePath}, repoID)
    checkpointInv.AddDecision(directive, userChoice, userResponse)
    checkpointStore.Save(ctx, checkpointInv)

    // Prompt user
    userChoice := promptUserForChoice(directive.UserOptions)

    if userChoice == "ABORT" {
        checkpointInv.SetTerminalState(agent.InvestigationAborted, riskLevel, confidence, summary)
        return investigationResult
    }
}
```

#### 3. Wire Checkpoint Storage (Priority 3)

**Add to main.go or check command:**
```go
// Initialize checkpoint store
checkpointStore, err := agent.NewCheckpointStore(ctx, postgresPool)
if err != nil {
    return fmt.Errorf("failed to create checkpoint store: %w", err)
}

// Pass to investigator
investigator := NewRiskInvestigator(llmClient, neo4jClient, checkpointStore)
```

#### 4. Test and Verify

**After Each Fix, Run:**
```bash
# Clean test
cd /tmp
rm -rf omnara
git clone https://github.com/omnara-ai/omnara.git
cd omnara

# Set env
export GEMINI_API_KEY="AIzaSyAnkF7s3RLV5wVYLhxCRVnI2HrxVUK7zzU"
export LLM_PROVIDER="gemini"
export PHASE2_ENABLED="true"
export GITHUB_TOKEN="GITHUB_TOKEN_HERE"
export NEO4J_URI="bolt://localhost:7688"
export NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="5433"
export POSTGRES_DB="coderisk"
export POSTGRES_USER="coderisk"
export POSTGRES_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
export POSTGRES_DSN="postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable"

# Init
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk init

# Test CommandPalette (should trigger directive)
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk check \
  apps/web/src/components/dashboard/CommandPalette.tsx --explain

# Expected: Directive displayed, user prompted, checkpoint saved
```

**Success Criteria:**
- ✅ Co-change query returns data (not error)
- ✅ MEDIUM risk triggers directive display
- ✅ User prompted with options (APPROVE, SKIP, ABORT)
- ✅ Checkpoint saved to PostgreSQL investigations table
- ✅ Investigation can be resumed with `crisk check --resume <id>`

#### 5. Iterate Until Passing

**Goal:** All 4 test cases should:
1. ✅ Return accurate risk assessment
2. ✅ Display all co-change data
3. ✅ Trigger directives when appropriate (MEDIUM risk, missing data, incidents)
4. ✅ Save checkpoints for resumable investigations
5. ✅ Provide clear user options at decision points
6. ✅ Complete 12-factor agent alignment

---

## Useful Commands

### Quick Test Cycle
```bash
# 1. Rebuild
cd /Users/rohankatakam/Documents/brain/coderisk
make build

# 2. Set env (one-liner)
export GEMINI_API_KEY="AIzaSyAnkF7s3RLV5wVYLhxCRVnI2HrxVUK7zzU" LLM_PROVIDER="gemini" PHASE2_ENABLED="true" GITHUB_TOKEN="GITHUB_TOKEN_HERE" NEO4J_URI="bolt://localhost:7688" NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" POSTGRES_HOST="localhost" POSTGRES_PORT="5433" POSTGRES_DB="coderisk" POSTGRES_USER="coderisk" POSTGRES_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" POSTGRES_DSN="postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable"

# 3. Test
cd /tmp/omnara
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk check \
  apps/web/src/components/dashboard/CommandPalette.tsx --explain
```

### Debugging
```bash
# Check logs for Gemini calls
grep "gemini" /path/to/logs

# Verify co-change data exists
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "SELECT file_path, cochange_freq FROM file_cochanges LIMIT 10;"

# Check investigations table
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "SELECT id, phase, terminal_state, can_resume FROM investigations;"

# Watch Phase 2 in real-time
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk check <file> --explain 2>&1 | tee test_output.txt
```

---

## References

- **12-Factor Agents:** https://github.com/humanlayer/12-factor-agents
- **Omnara Repository:** https://github.com/omnara-ai/omnara
- **Ground Truth Data:** `/Users/rohankatakam/Documents/brain/coderisk/test_data/omnara_ground_truth_expanded.json`
- **Gemini API Docs:** https://ai.google.dev/docs
- **Commit:** a28db47 - "feat: Add Gemini API support with multi-provider LLM architecture and directive agent system"

---

## Contact & Handoff

**Current State:** Gemini integration working, Phase 2 autonomous, missing human-in-the-loop

**Critical Path:**
1. Fix co-change query → Unblocks risk assessment
2. Add directive triggers → Enables human decisions
3. Wire checkpoint storage → Enables pause/resume
4. Test & iterate → Validate 12-factor alignment

**Next Person:** You have everything needed to continue. Start with Priority 1 (co-change fix), then Priority 2 (directive triggers). Test after each change with CommandPalette.tsx (MEDIUM risk case).

Good luck! 🚀
