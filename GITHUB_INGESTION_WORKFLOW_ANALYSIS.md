# GitHub Ingestion Strategy: Pre-PR Workflow Analysis

**Date:** 2025-10-23
**Purpose:** Align ingestion strategy with actual developer workflow
**Critical Finding:** Current strategy assumes PR-centric analysis, but core use case is PRE-PR local branch analysis

---

## Problem Statement

### **Current Strategy Assumption (WRONG):**
```
Developer workflow:
1. Create branch
2. Make commits
3. Create PR ← We analyze HERE
4. Merge PR

Our ingestion:
- Fetch all PRs from GitHub
- Fetch all commits from GitHub
- Build graph from GitHub data
```

### **Actual Core Use Case (CORRECT):**
```
Developer workflow:
1. Create local branch
2. Make local commits
3. Run `crisk check` ← We analyze HERE (BEFORE PR exists!)
4. Fix issues
5. Create PR (optional)
6. Merge PR

Our need:
- Analyze LOCAL uncommitted changes
- Analyze LOCAL branch (not yet pushed)
- Analyze LOCAL branch (pushed but no PR yet)
- Compare against main/master branch
```

---

## Core Differentiator Analysis

### **What Makes CodeRisk Valuable:**

**Scenario 1: Pre-Commit Analysis (Most Valuable)**
```bash
# Developer is working locally
$ git status
On branch feature/auth-refactor
Changes not staged for commit:
  modified:   src/auth.py
  modified:   src/models/user.py

# Run crisk BEFORE committing
$ crisk check src/auth.py

Expected:
- Analyze uncommitted changes in auth.py
- Compare against main branch version
- Show blast radius (which code depends on auth.py)
- Risk assessment based on changes
```

**Problem:** This file doesn't exist in GitHub yet!

---

**Scenario 2: Pre-Push Analysis (Very Valuable)**
```bash
# Developer has local commits
$ git log --oneline main..HEAD
abc123 Fix authentication bug
def456 Add password validation
ghi789 Update user model

# Run crisk on local branch
$ crisk check

Expected:
- Analyze 3 local commits
- Compare against main branch
- Show which files changed
- Risk assessment
```

**Problem:** These commits don't exist in GitHub yet!

---

**Scenario 3: Pre-PR Analysis (Valuable)**
```bash
# Developer pushed branch to GitHub
$ git push origin feature/auth-refactor

# Run crisk before creating PR
$ crisk check

Expected:
- Analyze branch on GitHub
- Compare against main
- Show risk assessment
```

**Problem:** No PR exists yet, so PR-centric ingestion misses this!

---

**Scenario 4: Post-PR Analysis (Current Strategy)**
```bash
# Developer created PR #123
# Run crisk on PR
$ crisk check --pr 123

Expected:
- Analyze PR changes
- Compare against base branch
```

**This is the ONLY scenario our current strategy handles!**

---

## Workflow Coverage Analysis

| Scenario | Developer State | GitHub State | Current Strategy Handles? |
|----------|----------------|--------------|--------------------------|
| **1. Pre-Commit** | Local changes | Nothing | ❌ NO |
| **2. Pre-Push** | Local commits | Nothing | ❌ NO |
| **3. Pre-PR** | Pushed branch | Branch exists, no PR | ❌ NO |
| **4. Post-PR** | PR created | PR exists | ✅ YES |

**Coverage:** 25% of use cases (only scenario 4)

---

## What We Actually Need

### **Primary Data Source: Local Git Repository**

```bash
# Get current branch divergence from main
$ git log main..HEAD --name-status

Returns:
abc123 Fix authentication bug
  M  src/auth.py         (Modified)
  M  src/models/user.py  (Modified)

def456 Add password validation
  M  src/auth.py         (Modified)
  A  src/validators.py   (Added)

ghi789 Update user model
  M  src/models/user.py  (Modified)
```

**This gives us:**
- ✅ Local commits (not in GitHub)
- ✅ Files changed per commit
- ✅ Change type (Modified/Added/Deleted)
- ✅ Comparison against main branch
- ❌ Patch data (need git show)
- ❌ Remote context (PR links, issues)

---

### **Hybrid Approach: Local Git + GitHub API**

**For Pre-PR Analysis (Scenarios 1-3):**
```bash
# Use LOCAL git data
git log main..HEAD              # Get local commits
git diff main..HEAD --stat      # Get file changes
git show <commit-sha>           # Get patch data

# Use GITHUB for context (historical data)
GET /repos/{owner}/{repo}/commits?per_page=100  # Historical commits
GET /repos/{owner}/{repo}/issues                # Related issues
GET /repos/{owner}/{repo}/pulls                 # Historical PRs
```

**For Post-PR Analysis (Scenario 4):**
```bash
# Use GitHub API
GET /repos/{owner}/{repo}/pulls/{number}/commits
GET /repos/{owner}/{repo}/pulls/{number}/files
GET /repos/{owner}/{repo}/commits/{sha}  # With patches
```

---

## Revised Strategy: Git-First, GitHub-Second

### **Phase 1: Local Git Ingestion (0 API calls)**

```bash
# Detect current branch
current_branch=$(git rev-parse --abbrev-ref HEAD)

# Get divergence from main
commits=$(git log main..HEAD --format="%H|%s|%an|%ae|%at")

# For each commit, get file changes
for commit in $commits; do
  git show $commit --stat --format=""
  # Returns: filename, additions, deletions, change type
done

# Build graph
(Commit)-[:MODIFIES {additions, deletions, status}]->(File)

# Get patches on-demand
git show $commit -- <filename>  # Returns patch data
```

**What this gives us:**
- ✅ Works for pre-commit (git diff)
- ✅ Works for pre-push (local commits)
- ✅ Works for pre-PR (pushed branch)
- ✅ Zero API calls
- ✅ Instant ingestion (<1 second)

---

### **Phase 2: GitHub Context (163 API calls, one-time)**

```bash
# Metadata for historical context
GET /repos/{owner}/{repo}/pulls?per_page=100          # 2 calls
GET /repos/{owner}/{repo}/commits?per_page=100        # 5 calls
GET /repos/{owner}/{repo}/issues?per_page=100         # 1 call

# PR→Commit mapping for historical PRs
GET /repos/{owner}/{repo}/pulls/{number}/commits      # 155 calls

# Build historical graph
(PR)-[:CONTAINS_COMMIT]->(Commit)
```

**Purpose:**
- Historical context: "Has this file been risky before?"
- PR patterns: "Which PRs touched this module?"
- Issue links: "What bugs were fixed in this area?"

---

### **Phase 3: On-Demand Patch Fetching**

**For Local Commits:**
```bash
git show <commit-sha>  # No API calls!
```

**For Historical Commits (from GitHub):**
```bash
GET /repos/{owner}/{repo}/commits/{sha}  # 1 API call per commit
```

---

## File Metadata Without Patches: Analysis

### **Question:** Can we get Commit→File metadata without fetching patches?

**Answer:** Depends on data source!

### **Option A: Local Git (RECOMMENDED)**

```bash
# Get file changes WITHOUT patches
git log main..HEAD --name-status --format="%H"

Output:
abc123
M       src/auth.py
M       src/models/user.py

def456
M       src/auth.py
A       src/validators.py

# Get additions/deletions (still no patch)
git log main..HEAD --stat --format="%H"

Output:
abc123
 src/auth.py        | 15 ++++++++-------
 src/models/user.py |  3 +--
 2 files changed, 9 insertions(+), 8 deletions(-)

def456
 src/auth.py        | 42 ++++++++++++++++++++++++++++++++++++++++++
 src/validators.py  | 28 ++++++++++++++++++++++++++++
 2 files changed, 70 insertions(+)
```

**API Calls:** 0
**Speed:** <100ms for 1000 commits
**What you get:**
- ✅ Filename
- ✅ Status (M/A/D/R)
- ✅ Additions/deletions count
- ❌ NO patch data (defer to on-demand)

**Perfect for ingestion!**

---

### **Option B: GitHub REST API**

**Can we get file metadata from PR→Commits endpoint?**

```bash
GET /repos/{owner}/{repo}/pulls/{number}/commits

Response:
[
  {
    "sha": "abc123",
    "commit": {
      "message": "Fix auth bug",
      "author": {...}
    }
    // ❌ NO files[] array!
  }
]
```

**Answer:** ❌ NO - PR commits endpoint does NOT include file data

**To get files, you need:**
```bash
GET /repos/{owner}/{repo}/commits/{sha}  # Individual commit details

Response:
{
  "sha": "abc123",
  "files": [
    {
      "filename": "src/auth.py",
      "status": "modified",
      "additions": 15,
      "deletions": 3,
      "patch": "..." // ✅ Includes patch (even if we don't want it)
    }
  ]
}
```

**API Calls Required:** 1 per commit (465 calls for 465 commits!)

---

## Modified Strategy 5B: Git-First Hybrid

### **Ingestion Strategy:**

```bash
# Step 1: Local Git Analysis (0 API calls, <1 second)
# Get commits in current branch vs main
git log main..HEAD --stat --name-status --format="%H|%s|%an|%ae|%at"

# Build graph
for each commit:
  CREATE (c:Commit {sha, message, author_email, author_date})
  CREATE (d:Developer {email, name})
  CREATE (d)-[:AUTHORED]->(c)

  for each file in commit:
    CREATE (f:File {path})
    CREATE (c)-[:MODIFIES {
      status: "modified",
      additions: 15,
      deletions: 3
      // ❌ NO patch property yet
    }]->(f)

# Step 2: GitHub Historical Context (163 API calls, one-time)
GET /repos/{owner}/{repo}/pulls?per_page=100          # 2 calls
GET /repos/{owner}/{repo}/commits?per_page=100        # 5 calls
GET /repos/{owner}/{repo}/issues?per_page=100         # 1 call
GET /repos/{owner}/{repo}/pulls/{number}/commits      # 155 calls

# Build historical graph
(PR)-[:CONTAINS_COMMIT]->(Commit)

# Step 3: On-Demand Patch Fetching
# For local commits:
git show <sha>  # 0 API calls

# For historical commits:
GET /repos/{owner}/{repo}/commits/{sha}  # 1 API call
```

---

### **Graph Structure:**

```cypher
// Ingestion (Local Git + GitHub)
(Developer)-[:AUTHORED]->(Commit)
(Commit)-[:MODIFIES {
  status: "modified",
  additions: 15,
  deletions: 3
  // ❌ NO patch yet
}]->(File)
(PR)-[:CONTAINS_COMMIT]->(Commit)  // Historical only
(Commit)-[:ON_BRANCH]->(Branch)

// On-demand (During Analysis)
MATCH (c:Commit)-[r:MODIFIES]->(f:File)
SET r.patch = <fetch_from_git_or_github>
// ✅ Add patch property to existing edge
```

---

## API Call Analysis

### **Scenario 1: Pre-Commit Analysis (Local Changes)**

```bash
# User has uncommitted changes
$ crisk check src/auth.py

API Calls:
  - Local git diff: 0 API calls
  - Historical context: 0 API calls (already cached)
  - Patch data: 0 API calls (local git show)

Total: 0 API calls ✅
```

---

### **Scenario 2: Pre-Push Analysis (Local Branch)**

```bash
# User has 3 local commits
$ crisk check

API Calls:
  - Local git log: 0 API calls
  - File metadata: 0 API calls (git log --stat)
  - Patch data (on-demand): 0 API calls (git show)
  - Historical context: 0 API calls (already cached)

Total: 0 API calls ✅
```

---

### **Scenario 3: Pre-PR Analysis (Pushed Branch)**

```bash
# User pushed branch to GitHub, no PR yet
$ crisk check

Option A: Use local git (FAST)
  - Same as Scenario 2
  - Total: 0 API calls ✅

Option B: Fetch from GitHub
  - Check if branch exists: 1 API call
  - Get branch commits: 1 API call
  - Total: 2 API calls
```

---

### **Scenario 4: Post-PR Analysis**

```bash
# User created PR #123
$ crisk check --pr 123

API Calls:
  - Get PR commits: 1 API call (already cached if recently created)
  - Get commit patches (on-demand): 5-20 API calls

Total: 5-20 API calls
```

---

## Implementation Recommendation

### **Strategy 5B-Git (Modified for Local-First)**

**Phase 1: One-Time GitHub Ingestion (163 calls)**
```bash
# Historical context only
GET /repos/{owner}/{repo}/pulls?per_page=100          # 2 calls
GET /repos/{owner}/{repo}/commits?per_page=100        # 5 calls
GET /repos/{owner}/{repo}/issues?per_page=100         # 1 call
GET /repos/{owner}/{repo}/pulls/{number}/commits      # 155 calls

# Build: (PR)-[:CONTAINS_COMMIT]->(Commit)
```

**Phase 2: Every `crisk check` Invocation (0 API calls)**
```bash
# Use local git for current work
git log main..HEAD --stat --name-status

# Build:
#   (Commit)-[:MODIFIES {status, additions, deletions}]->(File)
# NO patch property yet
```

**Phase 3: On-Demand During Analysis (0-20 API calls)**
```bash
# For local commits: git show
# For historical GitHub commits: GET /commits/{sha}

# Add patch property to existing edges
```

---

## Benefits of Git-First Approach

### **1. Zero API Calls for Core Use Case**
- Pre-commit analysis: 0 calls
- Pre-push analysis: 0 calls
- Pre-PR analysis: 0 calls
- Only post-PR needs GitHub API

### **2. Instant Ingestion**
- Local git operations: <100ms
- No waiting for API responses
- No rate limit concerns

### **3. Works Offline**
- Analyze local changes without internet
- Perfect for laptop development
- Only need GitHub for historical context

### **4. Accurate Local State**
- Git is source of truth for current work
- No sync issues with GitHub
- Reflects actual developer changes

### **5. Smaller API Footprint**
```
Current strategy: 163 calls per repo
Git-first strategy: 163 calls one-time, then 0 calls for daily use
Savings: 100% for core workflow
```

---

## Graph Schema Alignment

### **Modified Strategy 5B-Git:**

**Nodes:**
```
Developer (from git + GitHub)
Commit (from git + GitHub)
Branch (from git + GitHub)
Issue (from GitHub only)
PR (from GitHub only)
File (from git + GitHub)
```

**Edges - Ingestion:**
```
// From local git (0 API calls)
(Developer)-[:AUTHORED]->(Commit)
(Commit)-[:MODIFIES {status, additions, deletions}]->(File)  // NO patch
(Commit)-[:ON_BRANCH]->(Branch)

// From GitHub (163 API calls, one-time)
(PR)-[:CONTAINS_COMMIT]->(Commit)
(PR)-[:FROM_BRANCH]->(Branch)
(PR)-[:TO_BRANCH]->(Branch)
(PR)-[:MERGED_AS]->(Commit)
```

**Edges - On-Demand:**
```
// During analysis (add patch to existing edge)
MATCH (c:Commit)-[r:MODIFIES]->(f:File)
SET r.patch = get_patch_data(c.sha)  // From git or GitHub
```

---

## Scaling Analysis

| Scenario | API Calls | Time | Use Case Coverage |
|----------|-----------|------|-------------------|
| **Git-First (5B-Git)** | 163 one-time | <1s per analysis | 100% (all 4 scenarios) |
| **PR-First (5B)** | 163 per repo | ~3min per analysis | 25% (only post-PR) |
| **PR+File (4)** | 318 per repo | ~5min per analysis | 25% (only post-PR) |

**Winner:** Git-First (5B-Git) ✅

---

## Recommendation

**Use Modified Strategy 5B-Git:**

1. **One-time GitHub ingestion**: 163 API calls for historical context
   - (PR)-[:CONTAINS_COMMIT]->(Commit)

2. **Every analysis**: 0 API calls, use local git
   - (Commit)-[:MODIFIES {no patch}]->(File)

3. **On-demand patches**: From git (local) or GitHub (historical)
   - Add patch property to existing edges

**This gives you:**
- ✅ Zero API calls for 75% of use cases
- ✅ Instant analysis (<1 second ingestion)
- ✅ Works offline for local development
- ✅ Historical context from GitHub when needed
- ✅ Aligns with actual developer workflow

---

## Next Steps

1. Update GITHUB_GRAPH_SPEC.md to reflect Git-First strategy
2. Document git command patterns for file metadata extraction
3. Design git+GitHub hybrid ingestion flow
4. Clarify when to use git vs GitHub API for patch data
