# GitHub Data Extraction Specification

## Overview

This document specifies exactly what data CodeRisk extracts from GitHub repositories for risk assessment, including scope, timeframes, and API limitations.

## Data Extraction Scope

### 1. Commits

**What we extract:**
- Commits from **ALL branches** (not just main/master)
- Time window: Configurable, default **90 days** (as specified in risk_math.md)
- Maximum commits: No hard limit, but shallow clones may limit history

**Data points captured:**
- SHA (commit hash)
- Commit message
- Author name and email
- Timestamp
- Files changed (list of file paths)
- Lines added/deleted
- Parent commits (for merge detection)
- Flags: is_merge, is_revert, is_hotfix

**Current Implementation:**
```python
# From github_extractor.py
async def extract_commits(self, window_days: int = 90, branch: str = "main")
```

**ISSUE IDENTIFIED:** Currently only extracting from main branch!

**REQUIRED FIX:** Should iterate through all branches:
```python
for branch in repo.branches:
    commits = await extract_commits(window_days=90, branch=branch.name)
```

### 2. Pull Requests

**What we SHOULD extract:**
- **BOTH open AND closed** pull requests
- Time window: Created or updated within last **90 days**
- Required: GitHub API token (PyGithub library requirement)

**Data points captured:**
- PR number
- Title and description
- State (open/closed/merged)
- Author
- Created/merged/closed timestamps
- Files changed with additions/deletions
- Review comments
- Reviewers
- Linked issues
- Labels

**API Method:**
```python
repo.get_pulls(state="all", sort="updated", direction="desc")
```

**Current Implementation Status:**
- ✅ Code exists and works with token
- ❌ Requires GitHub token (not working without it)
- ✅ Correctly fetches both open and closed

### 3. Issues

**What we SHOULD extract:**
- **BOTH open AND closed** issues
- Time window: Created or updated within last **90 days**
- Excludes: Pull requests (GitHub API returns PRs as issues too)
- Required: GitHub API token

**Data points captured:**
- Issue number
- Title and description
- State (open/closed)
- Author
- Created/closed timestamps
- Labels (including bug, incident, severity indicators)
- Assignees
- Linked PRs (extracted from description/comments)
- Comments with authors and timestamps
- Flags: is_bug, is_incident
- Severity (extracted from labels)

**API Method:**
```python
repo.get_issues(state="all", since=since_date)
# Filter out PRs: if issue.pull_request: continue
```

**Current Implementation Status:**
- ✅ Code exists and works with token
- ❌ Requires GitHub token
- ✅ Correctly filters out PRs
- ✅ Extracts severity from labels

### 4. Developers

**What we extract:**
- All unique contributors from commits, PRs, and issues
- Aggregated statistics per developer
- Time window: Based on activity within the 90-day window

**Data points captured:**
- Username
- Email (from commits)
- Name
- Commit count
- PRs authored/reviewed
- Issues created/resolved
- First/last contribution dates
- Files touched
- Expertise areas (inferred from file types)

**Current Implementation Status:**
- ✅ Correctly aggregates from all sources
- ✅ Works partially without token (commits only)

## Time Window Configuration

### Default: 90 Days
As specified in `risk_math.md`:
> "90-day sample (or chosen window W): commits/PRs/issues/owners"

### Configurable Window
The extraction window is configurable via the `window_days` parameter:
- Minimum: 7 days (for meaningful data)
- Default: 90 days
- Maximum: 365 days (GitHub API limitations)

### Time Window Application
- **Commits**: Commits within last `window_days`
- **PRs**: Created OR updated within last `window_days`
- **Issues**: Created OR updated within last `window_days`
- **Developers**: Activity within the time window

## API Limitations & Requirements

### PyGithub Library Limitations

1. **Authentication Required for:**
   - Pull Requests data
   - Issues data
   - Private repositories
   - Rate limit increase (60/hour → 5000/hour with token)

2. **Works Without Token:**
   - Local git commit history
   - Public repository cloning
   - Basic developer info from commits

3. **Rate Limits:**
   - Unauthenticated: 60 requests/hour
   - Authenticated: 5,000 requests/hour
   - GraphQL: 5,000 points/hour

### GitPython Library Capabilities

**What it CAN do:**
- Extract full commit history from local clone
- Access all branches
- Get file changes and diffs
- Work with shallow clones (limited history)

**What it CANNOT do:**
- Access GitHub-specific data (PRs, issues)
- Get review comments
- Access labels or assignees

## Required Fixes

### 1. Multi-Branch Commit Extraction
**Current:** Only extracts from main branch
**Required:** Extract from ALL branches

```python
# Fix needed in github_extractor.py
async def extract_commits(self, window_days: int = 90) -> List[CommitData]:
    all_commits = []
    for branch in self.git_repo.branches:
        try:
            commits = self._extract_branch_commits(branch.name, window_days)
            all_commits.extend(commits)
        except Exception as e:
            logger.warning(f"Failed to extract from branch {branch.name}: {e}")

    # Deduplicate commits by SHA
    unique_commits = {c.sha: c for c in all_commits}
    return list(unique_commits.values())
```

### 2. Token Validation & Warnings
**Current:** Silent failure when no token
**Required:** Clear warnings about missing data

```python
def __init__(self, repo_path: str, github_token: Optional[str] = None):
    if not github_token:
        logger.warning(
            "No GitHub token provided. The following data will NOT be available:\n"
            "- Pull Requests (0 PRs will be extracted)\n"
            "- Issues (0 issues will be extracted)\n"
            "- Review comments and reviewers\n"
            "To get full data, set GITHUB_TOKEN environment variable"
        )
```

### 3. Data Completeness Check
**Required:** Report what data is missing

```python
def validate_extraction_completeness(self, data: Dict) -> Dict[str, bool]:
    return {
        "commits": len(data["commits"]) > 0,
        "pull_requests": len(data["pull_requests"]) > 0 or not self.github_token,
        "issues": len(data["issues"]) > 0 or not self.github_token,
        "has_all_branches": self._checked_all_branches,
        "has_full_history": not self._is_shallow_clone
    }
```

## Usage Examples

### Full Extraction (with GitHub token)
```python
import os
from coderisk.ingestion.github_extractor import GitHubExtractor

# Set token
os.environ["GITHUB_TOKEN"] = "ghp_xxxxx"

# Extract all data
extractor = GitHubExtractor("/path/to/repo", os.getenv("GITHUB_TOKEN"))
data = await extractor.extract_all(
    repo_name="omnara-ai/omnara",
    window_days=90
)

# Results should include:
# - Commits from ALL branches
# - ALL pull requests (open & closed)
# - ALL issues (open & closed)
# - Complete developer profiles
```

### Limited Extraction (without token)
```python
# Without token - only local git data
extractor = GitHubExtractor("/path/to/repo")
data = await extractor.extract_all(window_days=90)

# Results will include:
# - Commits from local branches only
# - 0 pull requests (requires API)
# - 0 issues (requires API)
# - Developer info from commits only
```

## Validation Checklist

- [ ] Commits extracted from ALL branches, not just main
- [ ] Both OPEN and CLOSED pull requests included
- [ ] Both OPEN and CLOSED issues included
- [ ] Time window properly applied (90 days default)
- [ ] Deduplication of commits across branches
- [ ] Clear warnings when GitHub token missing
- [ ] Proper error handling for API rate limits
- [ ] Extraction summary shows what data was/wasn't collected

## Testing Requirements

1. **With Token Test**: Should extract PRs and issues
2. **Without Token Test**: Should extract commits only with warning
3. **Multi-Branch Test**: Verify all branches are processed
4. **Time Window Test**: Verify 90-day filter works correctly
5. **Deduplication Test**: Same commit on multiple branches counted once

## Conclusion

Our current implementation has a critical bug: we're only extracting commits from the main branch when we should extract from ALL branches. Additionally, without a GitHub token, we're missing crucial PR and issue data that's required for proper risk assessment as specified in risk_math.md. These issues must be fixed for accurate risk calculation.