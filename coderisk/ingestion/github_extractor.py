"""
GitHub data extraction module for CodeRisk

Extracts commits, PRs, issues, and developer information from GitHub repositories
using the GitHub API and git history.
"""

import asyncio
import os
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import List, Dict, Optional, Any, AsyncGenerator
import git
from github import Github, Repository, PullRequest, Issue, Commit
import structlog
from dataclasses import dataclass, field
import json

logger = structlog.get_logger(__name__)


@dataclass
class CommitData:
    """Structured commit data for ingestion"""
    sha: str
    message: str
    author: str
    author_email: str
    timestamp: datetime
    files_changed: List[str]
    additions: int
    deletions: int
    is_merge: bool
    is_revert: bool
    is_hotfix: bool
    branch: str
    parent_shas: List[str]

    def to_dict(self) -> Dict[str, Any]:
        return {
            "sha": self.sha,
            "message": self.message,
            "author": self.author,
            "author_email": self.author_email,
            "timestamp": self.timestamp.isoformat(),
            "files_changed": self.files_changed,
            "additions": self.additions,
            "deletions": self.deletions,
            "is_merge": self.is_merge,
            "is_revert": self.is_revert,
            "is_hotfix": self.is_hotfix,
            "branch": self.branch,
            "parent_shas": self.parent_shas
        }


@dataclass
class PRData:
    """Structured PR data for ingestion"""
    number: int
    title: str
    description: str
    state: str
    author: str
    created_at: datetime
    merged_at: Optional[datetime]
    closed_at: Optional[datetime]
    files_changed: List[str]
    additions: int
    deletions: int
    review_comments: List[Dict[str, str]]
    labels: List[str]
    reviewers: List[str]
    linked_issues: List[int]

    def to_dict(self) -> Dict[str, Any]:
        return {
            "number": self.number,
            "title": self.title,
            "description": self.description,
            "state": self.state,
            "author": self.author,
            "created_at": self.created_at.isoformat(),
            "merged_at": self.merged_at.isoformat() if self.merged_at else None,
            "closed_at": self.closed_at.isoformat() if self.closed_at else None,
            "files_changed": self.files_changed,
            "additions": self.additions,
            "deletions": self.deletions,
            "review_comments": self.review_comments,
            "labels": self.labels,
            "reviewers": self.reviewers,
            "linked_issues": self.linked_issues
        }


@dataclass
class IssueData:
    """Structured issue data for ingestion"""
    number: int
    title: str
    description: str
    state: str
    author: str
    created_at: datetime
    closed_at: Optional[datetime]
    labels: List[str]
    assignees: List[str]
    linked_prs: List[int]
    comments: List[Dict[str, str]]
    is_bug: bool
    is_incident: bool
    severity: Optional[str]

    def to_dict(self) -> Dict[str, Any]:
        return {
            "number": self.number,
            "title": self.title,
            "description": self.description,
            "state": self.state,
            "author": self.author,
            "created_at": self.created_at.isoformat(),
            "closed_at": self.closed_at.isoformat() if self.closed_at else None,
            "labels": self.labels,
            "assignees": self.assignees,
            "linked_prs": self.linked_prs,
            "comments": self.comments,
            "is_bug": self.is_bug,
            "is_incident": self.is_incident,
            "severity": self.severity
        }


@dataclass
class DeveloperData:
    """Structured developer data for ingestion"""
    username: str
    email: Optional[str]
    name: Optional[str]
    commits_count: int
    prs_authored: int
    prs_reviewed: int
    issues_created: int
    issues_resolved: int
    first_contribution: datetime
    last_contribution: datetime
    files_touched: List[str]
    expertise_areas: List[str]

    def to_dict(self) -> Dict[str, Any]:
        return {
            "username": self.username,
            "email": self.email,
            "name": self.name,
            "commits_count": self.commits_count,
            "prs_authored": self.prs_authored,
            "prs_reviewed": self.prs_reviewed,
            "issues_created": self.issues_created,
            "issues_resolved": self.issues_resolved,
            "first_contribution": self.first_contribution.isoformat(),
            "last_contribution": self.last_contribution.isoformat(),
            "files_touched": self.files_touched,
            "expertise_areas": self.expertise_areas
        }


class GitHubExtractor:
    """Extract GitHub repository data for risk analysis"""

    def __init__(self, repo_path: str, github_token: Optional[str] = None):
        self.repo_path = Path(repo_path)
        self.github_token = github_token or os.getenv("GITHUB_TOKEN")
        self.github_client = None
        self.github_repo = None
        self.git_repo = None

        # Initialize if token available
        if self.github_token:
            self.github_client = Github(self.github_token)
            logger.info("GitHub API client initialized")
        else:
            logger.warning(
                "\n" + "="*60 + "\n"
                "⚠️  NO GITHUB TOKEN PROVIDED\n"
                "="*60 + "\n"
                "The following data will NOT be available:\n"
                "  ❌ Pull Requests (0 PRs will be extracted)\n"
                "  ❌ Issues (0 issues will be extracted)\n"
                "  ❌ Review comments and reviewers\n"
                "  ❌ Issue labels and assignees\n"
                "\n"
                "Only local git commit history will be extracted.\n"
                "\n"
                "To get full data for risk assessment:\n"
                "  1. Create a GitHub token at: https://github.com/settings/tokens\n"
                "  2. Set environment variable: export GITHUB_TOKEN='ghp_xxxxx'\n"
                "  3. Run the extraction again\n"
                + "="*60
            )

    def _init_local_repo(self):
        """Initialize local git repository"""
        try:
            self.git_repo = git.Repo(self.repo_path)
            logger.info("Local git repository initialized", path=str(self.repo_path))
        except Exception as e:
            logger.error(f"Failed to initialize git repository: {e}")
            raise

    def _init_github_repo(self, repo_name: str):
        """Initialize GitHub repository connection"""
        if not self.github_client:
            logger.warning("GitHub client not available")
            return

        try:
            self.github_repo = self.github_client.get_repo(repo_name)
            logger.info("GitHub repository connected", repo=repo_name)
        except Exception as e:
            logger.error(f"Failed to connect to GitHub repository: {e}")

    async def extract_commits(self, window_days: int = 90, branch: Optional[str] = None) -> List[CommitData]:
        """Extract commit history from git - from ALL branches or specific branch"""
        if not self.git_repo:
            self._init_local_repo()

        commits_data = []
        commits_seen = set()  # Track unique commits by SHA
        since_date = datetime.now(timezone.utc) - timedelta(days=window_days)

        try:
            # Determine which branches to process
            if branch:
                # Specific branch requested
                branches_to_process = [branch]
            else:
                # Extract from ALL branches
                branches_to_process = [b.name for b in self.git_repo.branches]
                logger.info(f"Processing {len(branches_to_process)} branches: {branches_to_process[:5]}...")

            # Process each branch
            for branch_name in branches_to_process:
                try:
                    branch_commits = 0
                    # Get commits from this branch
                    for commit in self.git_repo.iter_commits(branch_name, since=since_date):
                        # Skip if we've already seen this commit (deduplication)
                        if commit.hexsha in commits_seen:
                            continue
                        commits_seen.add(commit.hexsha)

                        # Determine commit type
                        is_merge = len(commit.parents) > 1
                        is_revert = commit.message.lower().startswith("revert")
                        is_hotfix = "hotfix" in commit.message.lower() or "urgent" in commit.message.lower()

                        # Get changed files
                        files_changed = []
                        additions = 0
                        deletions = 0

                        for diff in commit.diff(commit.parents[0] if commit.parents else None):
                            if diff.a_path:
                                files_changed.append(diff.a_path)
                            if diff.b_path and diff.b_path not in files_changed:
                                files_changed.append(diff.b_path)

                        # Get stats if available
                        try:
                            stats = commit.stats.total
                            additions = stats.get("insertions", 0)
                            deletions = stats.get("deletions", 0)
                        except:
                            pass

                        commit_data = CommitData(
                            sha=commit.hexsha,
                            message=commit.message.strip(),
                            author=commit.author.name,
                            author_email=commit.author.email,
                            timestamp=datetime.fromtimestamp(commit.committed_date, tz=timezone.utc),
                            files_changed=files_changed,
                            additions=additions,
                            deletions=deletions,
                            is_merge=is_merge,
                            is_revert=is_revert,
                            is_hotfix=is_hotfix,
                            branch=branch_name,
                            parent_shas=[p.hexsha for p in commit.parents]
                        )

                        commits_data.append(commit_data)
                        branch_commits += 1

                    if branch_commits > 0:
                        logger.debug(f"Branch {branch_name}: {branch_commits} commits")

                except Exception as e:
                    logger.warning(f"Failed to extract from branch {branch_name}: {e}")

            # Sort commits by timestamp (newest first)
            commits_data.sort(key=lambda c: c.timestamp, reverse=True)

            logger.info(f"Extracted {len(commits_data)} unique commits from {len(branches_to_process)} branches (last {window_days} days)")

        except Exception as e:
            logger.error(f"Failed to extract commits: {e}")

        return commits_data

    async def extract_pull_requests(self, window_days: int = 90) -> List[PRData]:
        """Extract pull requests from GitHub API"""
        if not self.github_repo:
            logger.warning("GitHub repository not initialized, skipping PR extraction")
            return []

        prs_data = []
        since_date = datetime.now(timezone.utc) - timedelta(days=window_days)

        try:
            # Get PRs (both open and closed)
            for pr in self.github_repo.get_pulls(state="all", sort="updated", direction="desc"):
                # Skip if outside window
                if pr.created_at < since_date:
                    break

                # Extract review comments
                review_comments = []
                for comment in pr.get_review_comments():
                    review_comments.append({
                        "author": comment.user.login if comment.user else "unknown",
                        "body": comment.body,
                        "path": comment.path,
                        "line": comment.line
                    })

                # Extract reviewers
                reviewers = []
                for review in pr.get_reviews():
                    if review.user and review.user.login not in reviewers:
                        reviewers.append(review.user.login)

                # Extract linked issues from description
                linked_issues = self._extract_issue_numbers(pr.body or "")

                # Get changed files
                files_changed = []
                additions = 0
                deletions = 0
                for file in pr.get_files():
                    files_changed.append(file.filename)
                    additions += file.additions
                    deletions += file.deletions

                pr_data = PRData(
                    number=pr.number,
                    title=pr.title,
                    description=pr.body or "",
                    state=pr.state,
                    author=pr.user.login if pr.user else "unknown",
                    created_at=pr.created_at,
                    merged_at=pr.merged_at,
                    closed_at=pr.closed_at,
                    files_changed=files_changed,
                    additions=additions,
                    deletions=deletions,
                    review_comments=review_comments,
                    labels=[label.name for label in pr.labels],
                    reviewers=reviewers,
                    linked_issues=linked_issues
                )

                prs_data.append(pr_data)

            logger.info(f"Extracted {len(prs_data)} pull requests")

        except Exception as e:
            logger.error(f"Failed to extract pull requests: {e}")

        return prs_data

    async def extract_issues(self, window_days: int = 90) -> List[IssueData]:
        """Extract issues from GitHub API"""
        if not self.github_repo:
            logger.warning("GitHub repository not initialized, skipping issue extraction")
            return []

        issues_data = []
        since_date = datetime.now(timezone.utc) - timedelta(days=window_days)

        try:
            # Get issues (both open and closed)
            for issue in self.github_repo.get_issues(state="all", since=since_date):
                # Skip pull requests (they also appear as issues)
                if issue.pull_request:
                    continue

                # Determine issue type
                labels_lower = [label.name.lower() for label in issue.labels]
                is_bug = any(label in labels_lower for label in ["bug", "defect", "error"])
                is_incident = any(label in labels_lower for label in ["incident", "outage", "production"])

                # Determine severity
                severity = None
                for label in labels_lower:
                    if "critical" in label or "p0" in label:
                        severity = "critical"
                        break
                    elif "high" in label or "p1" in label:
                        severity = "high"
                        break
                    elif "medium" in label or "p2" in label:
                        severity = "medium"
                        break
                    elif "low" in label or "p3" in label:
                        severity = "low"
                        break

                # Extract comments
                comments = []
                for comment in issue.get_comments():
                    comments.append({
                        "author": comment.user.login if comment.user else "unknown",
                        "body": comment.body,
                        "created_at": comment.created_at.isoformat()
                    })

                # Extract linked PRs from comments and description
                linked_prs = self._extract_pr_numbers(issue.body or "")
                for comment in comments:
                    linked_prs.extend(self._extract_pr_numbers(comment["body"]))

                issue_data = IssueData(
                    number=issue.number,
                    title=issue.title,
                    description=issue.body or "",
                    state=issue.state,
                    author=issue.user.login if issue.user else "unknown",
                    created_at=issue.created_at,
                    closed_at=issue.closed_at,
                    labels=[label.name for label in issue.labels],
                    assignees=[assignee.login for assignee in issue.assignees],
                    linked_prs=list(set(linked_prs)),
                    comments=comments,
                    is_bug=is_bug,
                    is_incident=is_incident,
                    severity=severity
                )

                issues_data.append(issue_data)

            logger.info(f"Extracted {len(issues_data)} issues")

        except Exception as e:
            logger.error(f"Failed to extract issues: {e}")

        return issues_data

    async def extract_developers(self, commits: List[CommitData],
                                prs: List[PRData],
                                issues: List[IssueData]) -> List[DeveloperData]:
        """Extract developer information from commits, PRs, and issues"""
        developers = {}

        # Process commits
        for commit in commits:
            key = commit.author_email or commit.author
            if key not in developers:
                developers[key] = {
                    "username": commit.author,
                    "email": commit.author_email,
                    "name": commit.author,
                    "commits_count": 0,
                    "prs_authored": 0,
                    "prs_reviewed": 0,
                    "issues_created": 0,
                    "issues_resolved": 0,
                    "first_contribution": commit.timestamp,
                    "last_contribution": commit.timestamp,
                    "files_touched": set(),
                    "expertise_areas": set()
                }

            dev = developers[key]
            dev["commits_count"] += 1
            dev["files_touched"].update(commit.files_changed)

            # Update contribution dates
            if commit.timestamp < dev["first_contribution"]:
                dev["first_contribution"] = commit.timestamp
            if commit.timestamp > dev["last_contribution"]:
                dev["last_contribution"] = commit.timestamp

            # Determine expertise areas from files
            for file in commit.files_changed:
                if file.endswith(".py"):
                    dev["expertise_areas"].add("python")
                elif file.endswith((".js", ".ts", ".jsx", ".tsx")):
                    dev["expertise_areas"].add("javascript")
                elif file.endswith((".java", ".kt")):
                    dev["expertise_areas"].add("java")
                elif file.endswith((".go")):
                    dev["expertise_areas"].add("golang")
                elif file.endswith((".rs")):
                    dev["expertise_areas"].add("rust")
                elif file.endswith((".sql")):
                    dev["expertise_areas"].add("database")
                elif "test" in file.lower():
                    dev["expertise_areas"].add("testing")
                elif "doc" in file.lower():
                    dev["expertise_areas"].add("documentation")

        # Process PRs
        for pr in prs:
            key = pr.author
            if key not in developers:
                developers[key] = {
                    "username": pr.author,
                    "email": None,
                    "name": pr.author,
                    "commits_count": 0,
                    "prs_authored": 0,
                    "prs_reviewed": 0,
                    "issues_created": 0,
                    "issues_resolved": 0,
                    "first_contribution": pr.created_at,
                    "last_contribution": pr.created_at,
                    "files_touched": set(),
                    "expertise_areas": set()
                }

            developers[key]["prs_authored"] += 1
            developers[key]["files_touched"].update(pr.files_changed)

            # Process reviewers
            for reviewer in pr.reviewers:
                if reviewer not in developers:
                    developers[reviewer] = {
                        "username": reviewer,
                        "email": None,
                        "name": reviewer,
                        "commits_count": 0,
                        "prs_authored": 0,
                        "prs_reviewed": 0,
                        "issues_created": 0,
                        "issues_resolved": 0,
                        "first_contribution": pr.created_at,
                        "last_contribution": pr.created_at,
                        "files_touched": set(),
                        "expertise_areas": set()
                    }
                developers[reviewer]["prs_reviewed"] += 1

        # Process issues
        for issue in issues:
            key = issue.author
            if key not in developers:
                developers[key] = {
                    "username": issue.author,
                    "email": None,
                    "name": issue.author,
                    "commits_count": 0,
                    "prs_authored": 0,
                    "prs_reviewed": 0,
                    "issues_created": 0,
                    "issues_resolved": 0,
                    "first_contribution": issue.created_at,
                    "last_contribution": issue.created_at,
                    "files_touched": set(),
                    "expertise_areas": set()
                }

            developers[key]["issues_created"] += 1

            # Process assignees (resolved issues)
            if issue.state == "closed":
                for assignee in issue.assignees:
                    if assignee in developers:
                        developers[assignee]["issues_resolved"] += 1

        # Convert to DeveloperData objects
        developers_data = []
        for key, dev in developers.items():
            developers_data.append(DeveloperData(
                username=dev["username"],
                email=dev["email"],
                name=dev["name"],
                commits_count=dev["commits_count"],
                prs_authored=dev["prs_authored"],
                prs_reviewed=dev["prs_reviewed"],
                issues_created=dev["issues_created"],
                issues_resolved=dev["issues_resolved"],
                first_contribution=dev["first_contribution"],
                last_contribution=dev["last_contribution"],
                files_touched=list(dev["files_touched"]),
                expertise_areas=list(dev["expertise_areas"])
            ))

        logger.info(f"Extracted {len(developers_data)} developers")
        return developers_data

    def _extract_issue_numbers(self, text: str) -> List[int]:
        """Extract issue numbers from text (e.g., #123, fixes #456)"""
        import re
        pattern = r'#(\d+)'
        matches = re.findall(pattern, text)
        return [int(match) for match in matches]

    def _extract_pr_numbers(self, text: str) -> List[int]:
        """Extract PR numbers from text"""
        import re
        pattern = r'(?:PR|pr|pull request)\s*#?(\d+)'
        matches = re.findall(pattern, text, re.IGNORECASE)
        return [int(match) for match in matches]

    async def extract_all(self, repo_name: Optional[str] = None,
                          window_days: int = 90) -> Dict[str, Any]:
        """Extract all GitHub data for the repository"""
        logger.info(f"Starting GitHub data extraction for {repo_name or self.repo_path}")

        # Initialize GitHub repo if name provided
        if repo_name:
            self._init_github_repo(repo_name)

        # Extract data
        commits = await self.extract_commits(window_days)
        prs = await self.extract_pull_requests(window_days)
        issues = await self.extract_issues(window_days)
        developers = await self.extract_developers(commits, prs, issues)

        # Prepare summary
        summary = {
            "repository": repo_name or str(self.repo_path),
            "extraction_date": datetime.now(timezone.utc).isoformat(),
            "window_days": window_days,
            "statistics": {
                "commits": len(commits),
                "pull_requests": len(prs),
                "issues": len(issues),
                "developers": len(developers),
                "total_files_changed": len(set(f for c in commits for f in c.files_changed)),
                "total_additions": sum(c.additions for c in commits),
                "total_deletions": sum(c.deletions for c in commits)
            },
            "data": {
                "commits": [c.to_dict() for c in commits],
                "pull_requests": [pr.to_dict() for pr in prs],
                "issues": [i.to_dict() for i in issues],
                "developers": [d.to_dict() for d in developers]
            }
        }

        logger.info("GitHub data extraction complete", stats=summary["statistics"])
        return summary