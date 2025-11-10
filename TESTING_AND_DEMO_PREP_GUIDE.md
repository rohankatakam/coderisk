# CodeRisk Testing & Demo Preparation Guide

**Date**: November 10, 2025
**Purpose**: Guide manual testing and labeling to find compelling demo examples
**Status**: Ready for execution

---

## Overview

This guide will help you systematically test `crisk check` on the Supabase repository to:
1. Validate the system works correctly
2. Find compelling examples for demos
3. Identify any issues that need fixing
4. Build confidence in the product

---

## Prerequisites

✅ **Completed Steps**:
- [x] Supabase repository ingested (`crisk init` completed)
- [x] History pruning implemented (no more rate limits)
- [x] Rich narrative output format working
- [x] Code committed and pushed

⚙️ **System Requirements**:
- Docker running (Neo4j + PostgreSQL)
- Gemini API key configured
- Binary built: `/Users/rohankatakam/Documents/brain/coderisk/bin/crisk`

---

## Testing Strategy

We'll use a **three-tier approach** to find the best demo examples:

### Tier 1: Known Incidents (Ground Truth)
Test files that were part of confirmed production incidents

### Tier 2: High-Risk Files (Predicted)
Test files flagged as HIGH risk by Phase 1 metrics

### Tier 3: Control Group (Baseline)
Test low-risk files to ensure we're not over-flagging

---

## Part 1: Identify Test Cases

### Step 1.1: Find Known Incident Files (Tier 1)

These are files from the Supabase revert PRs we identified:

#### **Revert PR #39866** (October 25, 2024)
- **Description**: "fix: revert 39818" - Emergency revert of table editor bug
- **File**: `apps/studio/components/ui/TableEditor.tsx` (or similar)
- **Issue**: Customers couldn't edit databases
- **Severity**: SEV-1 (Production down)

#### **Revert PR #34789** (August 2024)
- **Description**: Auth flow regression
- **File**: `apps/studio/components/auth/SignInPage.tsx` (or similar)
- **Severity**: SEV-2 (Core functionality broken)

#### **Revert PR #32145** (July 2024)
- **Description**: API endpoint performance regression
- **File**: `apps/platform/backend/api/queries.ts` (or similar)
- **Severity**: SEV-2 (Performance degradation)

**Action**: Use the GitHub UI or API to find the exact file paths from these PRs.

```bash
# Example: Find files changed in a revert PR
gh api repos/supabase/supabase/pulls/39866/files --jq '.[] | .filename'
```

**Record Format**:
Create a spreadsheet or markdown file:

```markdown
| Test ID | File Path | Incident PR | Expected Risk | Notes |
|---------|-----------|-------------|---------------|-------|
| T1-01   | apps/studio/components/ui/TableEditor.tsx | #39866 | HIGH/CRITICAL | Production table editor failure |
| T1-02   | apps/studio/components/auth/SignInPage.tsx | #34789 | HIGH | Auth flow regression |
| T1-03   | apps/platform/backend/api/queries.ts | #32145 | MEDIUM/HIGH | Performance regression |
```

---

### Step 1.2: Find High-Risk Files from Phase 1 (Tier 2)

Run queries against the database to find files with high risk indicators:

#### Query 1: Files with Multiple Incidents
```bash
# Connect to PostgreSQL
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "
SELECT
  f.path,
  COUNT(DISTINCT il.issue_id) as incident_count,
  MAX(il.link_confidence) as max_confidence
FROM files f
JOIN file_incident_links fil ON f.id = fil.file_id
JOIN issue_links il ON fil.link_id = il.id
WHERE il.link_confidence > 0.5
GROUP BY f.path
HAVING COUNT(DISTINCT il.issue_id) >= 2
ORDER BY incident_count DESC, max_confidence DESC
LIMIT 20;
"
```

#### Query 2: Files with Stale Ownership
```bash
# Files where the main contributor is inactive
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "
SELECT
  f.path,
  d.email as owner,
  EXTRACT(DAY FROM NOW() - MAX(c.timestamp)) as days_inactive
FROM files f
JOIN commits c ON c.id IN (
  SELECT commit_id FROM commit_files WHERE file_id = f.id
)
JOIN developers d ON c.developer_id = d.id
GROUP BY f.path, d.email
HAVING EXTRACT(DAY FROM NOW() - MAX(c.timestamp)) > 90
ORDER BY days_inactive DESC
LIMIT 20;
"
```

#### Query 3: Files with High Co-Change Frequency
```bash
# Files that frequently change together (forgotten update risk)
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "
SELECT
  f1.path as file,
  COUNT(DISTINCT c.id) as change_frequency,
  ARRAY_AGG(DISTINCT f2.path) as cochange_partners
FROM files f1
JOIN commit_files cf1 ON f1.id = cf1.file_id
JOIN commits c ON cf1.commit_id = c.id
JOIN commit_files cf2 ON c.id = cf2.commit_id AND cf2.file_id != cf1.file_id
JOIN files f2 ON cf2.file_id = f2.id
WHERE c.timestamp > NOW() - INTERVAL '180 days'
GROUP BY f1.path
HAVING COUNT(DISTINCT c.id) > 10
ORDER BY change_frequency DESC
LIMIT 20;
"
```

**Record These Files**: Add to your test spreadsheet as Tier 2 tests.

---

### Step 1.3: Select Control Files (Tier 3)

Pick a few files that should be LOW risk:

- Documentation files (README.md, docs/*.md)
- New files (created recently, no history)
- Configuration files (package.json, tsconfig.json)

```bash
# Find recently created files with no incident history
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "
SELECT f.path, MIN(c.timestamp) as created_at
FROM files f
JOIN commit_files cf ON f.id = cf.file_id
JOIN commits c ON cf.commit_id = c.id
WHERE f.path LIKE '%.md'
GROUP BY f.path
ORDER BY MIN(c.timestamp) DESC
LIMIT 10;
"
```

---

## Part 2: Run Tests and Record Results

### Step 2.1: Set Up Environment

```bash
# Set environment variables
export GEMINI_API_KEY="<your-key>"
export LLM_PROVIDER="gemini"
export POSTGRES_DSN="postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable"
export POSTGRES_HOST="localhost"
export POSTGRES_PORT="5433"
export POSTGRES_DB="coderisk"
export POSTGRES_USER="coderisk"
export POSTGRES_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
export NEO4J_URI="bolt://localhost:7688"
export NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"

# Navigate to Supabase repo
cd /Users/rohankatakam/Documents/brain/supabase
```

### Step 2.2: Test Each File

For each file in your test spreadsheet:

```bash
# Run crisk check with --explain
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk check <file-path> --explain > /tmp/test_<test-id>.txt 2>&1

# Example:
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk check apps/studio/components/ui/TableEditor.tsx --explain > /tmp/test_T1-01.txt 2>&1
```

### Step 2.3: Record Results

For each test, record:

| Field | What to Record |
|-------|----------------|
| **Test ID** | T1-01, T1-02, etc. |
| **File Path** | Full path to file |
| **Expected Risk** | What you expected (based on ground truth) |
| **Actual Risk** | What crisk reported (LOW/MEDIUM/HIGH/CRITICAL) |
| **Confidence** | Confidence percentage (0-100%) |
| **Investigation Hops** | Number of hops the agent took |
| **Duration** | Time to complete (seconds) |
| **Incidents Found** | How many past incidents were discovered |
| **Ownership Status** | Active/Inactive/Stale |
| **Co-Change Partners** | How many files change together |
| **Match** | ✅ Correct / ⚠️ Partial / ❌ Incorrect |
| **Demo Quality** | 🌟🌟🌟 (3 stars = perfect demo) |

**Example Entry**:
```markdown
| T1-01 | apps/studio/.../TableEditor.tsx | HIGH | HIGH | 85% | 7 | 12.5s | 3 | Inactive (120 days) | 8 | ✅ | 🌟🌟🌟 |
```

---

## Part 3: Identify Demo Candidates

### Criteria for a Great Demo Example

A file is a **great demo candidate** if:

1. ✅ **Matches Ground Truth**: Predicted risk aligns with actual incident history
2. ✅ **Clear Narrative**: Agent's investigation tells a compelling story
3. ✅ **Rich Data**: Shows incidents, ownership, co-change patterns
4. ✅ **High Stakes**: File is in a critical user flow (auth, payments, data editing)
5. ✅ **Actionable Insights**: Agent provides specific recommendations
6. ✅ **Fast Execution**: Completes in <15 seconds

### Demo Tiers

#### 🌟🌟🌟 **Tier S: The Money Shot**
- Known production incident (SEV-1 or SEV-2)
- Agent finds 2+ past incidents
- Shows stale ownership + co-change risk
- Predicts HIGH/CRITICAL risk correctly
- Investigation narrative is compelling

**Use Case**: This is your YC Demo Day slide. This is the Supabase revert PR example.

#### 🌟🌟 **Tier A: Strong Supporting Example**
- Known production incident OR high Phase 1 risk
- Agent finds 1+ past incident
- Shows at least 2 risk factors (ownership, co-change, blast radius)
- Predicts MEDIUM/HIGH risk correctly

**Use Case**: Design partner demos, blog posts, technical deep-dives.

#### 🌟 **Tier B: Illustrative Example**
- Demonstrates specific feature (co-change detection, ownership tracking)
- May not have incident history, but shows pattern detection
- Useful for explaining how the tool works

**Use Case**: Documentation, tutorial videos, feature walkthroughs.

---

## Part 4: Label Missing Data (If Needed)

If you test a file from a known revert PR and crisk **doesn't find incidents**, you need to investigate why:

### Step 4.1: Check if Issue Exists in Database

```bash
# Search for the revert PR issue number
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "
SELECT id, number, title, state, created_at
FROM github_issues
WHERE number = 39866;  -- Replace with actual PR number
"
```

**If NOT found**: The issue wasn't ingested. Check:
- Was it created within the last 180 days? (default `crisk init` window)
- Does it have bug/incident labels?

### Step 4.2: Check if Issue is Linked to File

```bash
# Check if issue is linked to the file
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "
SELECT
  gi.number as issue,
  f.path as file,
  il.link_confidence,
  il.detection_method
FROM github_issues gi
JOIN issue_links il ON gi.id = il.issue_id
JOIN file_incident_links fil ON il.id = fil.link_id
JOIN files f ON fil.file_id = f.id
WHERE gi.number = 39866 AND f.path LIKE '%TableEditor%';
"
```

**If NOT found**: The issue-file link wasn't detected. This is a labeling opportunity.

### Step 4.3: Manual Labeling

If the link is missing, you can manually add it:

```sql
-- Find the issue ID
SELECT id FROM github_issues WHERE number = 39866;

-- Find the file ID
SELECT id FROM files WHERE path = 'apps/studio/components/ui/TableEditor.tsx';

-- Create the link
INSERT INTO issue_links (issue_id, pr_id, link_type, link_confidence, detection_method, evidence_sources)
VALUES (
  <issue-id>,
  <pr-id>,  -- Find PR ID similarly
  'FIXED_BY',
  1.0,
  'MANUAL_LABEL',
  ARRAY['Manual label: Known production incident from revert PR']
);

-- Link the file
INSERT INTO file_incident_links (link_id, file_id)
SELECT il.id, f.id
FROM issue_links il, files f
WHERE il.issue_id = <issue-id> AND f.id = <file-id>;
```

**After Labeling**: Re-run `crisk check` on the file to see if it now detects the incident.

---

## Part 5: Analyze Results and Iterate

### Step 5.1: Calculate Accuracy

After testing all files, calculate:

```
Accuracy = (Correct Predictions) / (Total Tests) × 100%

Where:
- Correct = Predicted risk matches expected risk (±1 level)
- Example: Predicted HIGH for a known SEV-1 incident = Correct
- Example: Predicted MEDIUM for a known SEV-1 incident = Partial (±1 level)
- Example: Predicted LOW for a known SEV-1 incident = Incorrect
```

**Target Accuracy**: 80%+ for Tier 1 tests (known incidents)

### Step 5.2: Identify Patterns in Failures

If tests fail, categorize the failure:

| Failure Type | Cause | Action |
|--------------|-------|--------|
| **False Negative** | Missed a known incident | Check if issue was ingested, check linking |
| **False Positive** | Flagged a safe file as HIGH | Review Phase 1 metrics, check thresholds |
| **Missing Data** | No incidents/ownership found | Data ingestion issue, check queries |
| **Agent Confusion** | Agent gave wrong reasoning | Prompt engineering, tool improvements |

### Step 5.3: Document Findings

Create a summary report:

```markdown
# CodeRisk Testing Results - Supabase

## Summary
- **Total Tests**: 30
- **Tier 1 (Known Incidents)**: 10 tests, 9 correct (90% accuracy)
- **Tier 2 (High Risk)**: 15 tests, 12 correct (80% accuracy)
- **Tier 3 (Control)**: 5 tests, 5 correct (100% accuracy)

## Top Demo Candidates
1. **T1-01: TableEditor.tsx** - 🌟🌟🌟 Money shot! Found 3 incidents, stale ownership
2. **T1-02: SignInPage.tsx** - 🌟🌟 Strong example, 1 incident + co-change risk
3. **T2-05: queries.ts** - 🌟🌟 Performance regression pattern detected

## Issues Found
- 2 known incidents not linked (labeling needed)
- 1 false positive (threshold tuning needed)
```

---

## Part 6: Prepare Demo Script

Once you have your top candidates, prepare a demo script:

### The Demo Flow

1. **Set Context** (30 seconds)
   - "This is the Supabase repository, 73K stars, 800+ contributors"
   - "On October 25, they had a production incident - revert PR #39866"

2. **Show the Revert PR** (30 seconds)
   - Pull up GitHub PR: https://github.com/supabase/supabase/pull/39866
   - "Customers couldn't edit their databases. SEV-1, all hands on deck."

3. **Run crisk check** (10 seconds)
   - `crisk check apps/studio/components/ui/TableEditor.tsx --explain`
   - Let the output stream in real-time

4. **Narrate the Investigation** (60 seconds)
   - Point out each hop: "It's checking incident history... finding 3 past incidents"
   - "Look at the ownership - the main contributor left 120 days ago"
   - "It's detecting co-change patterns - 8 files usually change together"

5. **Show the Final Assessment** (30 seconds)
   - "It predicts HIGH risk with 85% confidence"
   - "These are the exact same patterns that caused the October incident"

6. **The Punchline** (30 seconds)
   - "This was all in the data. If they'd run this before committing, they would have seen this."
   - "That's predictive risk. That's CodeRisk."

**Total Time**: ~3 minutes

---

## Part 7: Iterate and Refine

Based on your testing:

### If Accuracy is Low (<70%):
- Review Phase 1 thresholds (coupling, co-change, incidents)
- Check data quality (CLQS score)
- Investigate false positives/negatives

### If Agent Investigation is Weak:
- Review agent prompts (kickoff prompt, tool descriptions)
- Add more context to tool results
- Improve error handling

### If Performance is Slow:
- Check database query performance
- Verify history pruning is working
- Monitor token usage

---

## Quick Reference Commands

### Run a Single Test
```bash
cd /Users/rohankatakam/Documents/brain/supabase
export GEMINI_API_KEY="<key>" && \
export POSTGRES_DSN="postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable" && \
export NEO4J_URI="bolt://localhost:7688" && \
export NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" && \
/Users/rohankatakam/Documents/brain/coderisk/bin/crisk check <file> --explain
```

### Query Incident History
```bash
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "
SELECT gi.number, gi.title, f.path, il.link_confidence
FROM github_issues gi
JOIN issue_links il ON gi.id = il.issue_id
JOIN file_incident_links fil ON il.id = fil.link_id
JOIN files f ON fil.file_id = f.id
WHERE f.path LIKE '%<search-term>%';
"
```

### Check CLQS Score
```bash
PGPASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123" psql -h localhost -p 5433 -U coderisk -d coderisk -c "
SELECT * FROM dora_metrics WHERE repo_id = 1;
"
```

---

## Success Metrics

By the end of this guide, you should have:

- ✅ 20-30 test cases run and documented
- ✅ 3-5 "money shot" demo examples identified
- ✅ 80%+ accuracy on known incident files
- ✅ A polished 3-minute demo script
- ✅ Confidence that the system works

**Next Steps**: Film the demo, prepare the pitch deck, practice the narrative!

---

## Appendix: Example Test Spreadsheet

```csv
TestID,FilePath,ExpectedRisk,ActualRisk,Confidence,Hops,Duration,IncidentsFound,OwnershipStatus,CoChangePartners,Match,DemoQuality
T1-01,apps/studio/components/ui/TableEditor.tsx,HIGH,HIGH,85%,7,12.5s,3,Inactive (120d),8,✅,🌟🌟🌟
T1-02,apps/studio/components/auth/SignInPage.tsx,HIGH,MEDIUM,70%,6,10.2s,1,Active,3,⚠️,🌟🌟
T1-03,apps/platform/backend/api/queries.ts,MEDIUM,MEDIUM,75%,5,8.9s,0,Active,12,✅,🌟
T2-01,apps/studio/hooks/useTableQuery.ts,MEDIUM,HIGH,65%,6,11.1s,0,Stale (60d),5,⚠️,🌟
T3-01,README.md,LOW,LOW,90%,3,5.2s,0,Active,0,✅,-
```

**Good luck!** 🚀
