"""
Cognee processor for GitHub data ingestion

Converts GitHub data into Cognee DataPoints and ingests them into the knowledge graph
"""

import asyncio
from datetime import datetime
from typing import List, Dict, Optional, Any
import cognee
from cognee import SearchType
from cognee.infrastructure.engine.models.DataPoint import DataPoint
from pathlib import Path
import structlog
import json

from .github_extractor import GitHubExtractor, CommitData, PRData, IssueData, DeveloperData

logger = structlog.get_logger(__name__)


class CommitDataPoint(DataPoint):
    """DataPoint for commit information"""
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

    class Config:
        arbitrary_types_allowed = True


class PRDataPoint(DataPoint):
    """DataPoint for pull request information"""
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

    class Config:
        arbitrary_types_allowed = True


class IssueDataPoint(DataPoint):
    """DataPoint for issue information"""
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

    class Config:
        arbitrary_types_allowed = True


class DeveloperDataPoint(DataPoint):
    """DataPoint for developer information"""
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

    class Config:
        arbitrary_types_allowed = True


class CogneeGitHubProcessor:
    """Process and ingest GitHub data into Cognee"""

    def __init__(self, repo_path: str, dataset_name: Optional[str] = None):
        self.repo_path = Path(repo_path).resolve()
        self.dataset_name = dataset_name or f"github_{self.repo_path.name}"
        self.extractor = None
        self._initialized = False

        logger.info("CogneeGitHubProcessor initialized",
                   repo_path=str(self.repo_path),
                   dataset_name=self.dataset_name)

    async def initialize(self, github_token: Optional[str] = None):
        """Initialize the processor and extractor"""
        if self._initialized:
            return

        # Initialize GitHub extractor
        self.extractor = GitHubExtractor(str(self.repo_path), github_token)

        # Set up Cognee configuration if needed
        try:
            # Test Cognee connection
            await cognee.prune.prune_data()  # Clear old data if needed
            logger.info("Cognee connection verified")
        except Exception as e:
            logger.warning(f"Cognee initialization warning: {e}")

        self._initialized = True
        logger.info("Processor initialization complete")

    async def extract_and_ingest_github_data(self,
                                            repo_name: Optional[str] = None,
                                            window_days: int = 90,
                                            include_code: bool = True) -> Dict[str, Any]:
        """Extract GitHub data and ingest into Cognee"""
        if not self._initialized:
            await self.initialize()

        logger.info(f"Starting GitHub data extraction and ingestion",
                   repo_name=repo_name,
                   window_days=window_days)

        # Step 1: Extract GitHub data
        github_data = await self.extractor.extract_all(repo_name, window_days)

        # Step 2: Convert to DataPoints
        datapoints = await self._convert_to_datapoints(github_data["data"])

        # Step 3: Ingest into Cognee
        ingestion_results = await self._ingest_to_cognee(datapoints, include_code)

        # Step 4: Build knowledge graph
        graph_results = await self._build_knowledge_graph()

        # Prepare summary
        summary = {
            "extraction": github_data["statistics"],
            "ingestion": ingestion_results,
            "graph": graph_results,
            "dataset_name": self.dataset_name
        }

        logger.info("GitHub data processing complete", summary=summary)
        return summary

    async def _convert_to_datapoints(self, data: Dict[str, List]) -> Dict[str, List[DataPoint]]:
        """Convert extracted data to Cognee DataPoints"""
        logger.info("Converting data to DataPoints")

        datapoints = {
            "commits": [],
            "pull_requests": [],
            "issues": [],
            "developers": []
        }

        # Convert commits
        for commit_dict in data.get("commits", []):
            commit_dp = CommitDataPoint(
                sha=commit_dict["sha"],
                message=commit_dict["message"],
                author=commit_dict["author"],
                author_email=commit_dict["author_email"],
                timestamp=datetime.fromisoformat(commit_dict["timestamp"]),
                files_changed=commit_dict["files_changed"],
                additions=commit_dict["additions"],
                deletions=commit_dict["deletions"],
                is_merge=commit_dict["is_merge"],
                is_revert=commit_dict["is_revert"],
                is_hotfix=commit_dict["is_hotfix"],
                branch=commit_dict["branch"],
                parent_shas=commit_dict["parent_shas"]
            )
            datapoints["commits"].append(commit_dp)

        # Convert PRs
        for pr_dict in data.get("pull_requests", []):
            pr_dp = PRDataPoint(
                number=pr_dict["number"],
                title=pr_dict["title"],
                description=pr_dict["description"],
                state=pr_dict["state"],
                author=pr_dict["author"],
                created_at=datetime.fromisoformat(pr_dict["created_at"]),
                merged_at=datetime.fromisoformat(pr_dict["merged_at"]) if pr_dict["merged_at"] else None,
                closed_at=datetime.fromisoformat(pr_dict["closed_at"]) if pr_dict["closed_at"] else None,
                files_changed=pr_dict["files_changed"],
                additions=pr_dict["additions"],
                deletions=pr_dict["deletions"],
                review_comments=pr_dict["review_comments"],
                labels=pr_dict["labels"],
                reviewers=pr_dict["reviewers"],
                linked_issues=pr_dict["linked_issues"]
            )
            datapoints["pull_requests"].append(pr_dp)

        # Convert issues
        for issue_dict in data.get("issues", []):
            issue_dp = IssueDataPoint(
                number=issue_dict["number"],
                title=issue_dict["title"],
                description=issue_dict["description"],
                state=issue_dict["state"],
                author=issue_dict["author"],
                created_at=datetime.fromisoformat(issue_dict["created_at"]),
                closed_at=datetime.fromisoformat(issue_dict["closed_at"]) if issue_dict["closed_at"] else None,
                labels=issue_dict["labels"],
                assignees=issue_dict["assignees"],
                linked_prs=issue_dict["linked_prs"],
                comments=issue_dict["comments"],
                is_bug=issue_dict["is_bug"],
                is_incident=issue_dict["is_incident"],
                severity=issue_dict["severity"]
            )
            datapoints["issues"].append(issue_dp)

        # Convert developers
        for dev_dict in data.get("developers", []):
            dev_dp = DeveloperDataPoint(
                username=dev_dict["username"],
                email=dev_dict["email"],
                name=dev_dict["name"],
                commits_count=dev_dict["commits_count"],
                prs_authored=dev_dict["prs_authored"],
                prs_reviewed=dev_dict["prs_reviewed"],
                issues_created=dev_dict["issues_created"],
                issues_resolved=dev_dict["issues_resolved"],
                first_contribution=datetime.fromisoformat(dev_dict["first_contribution"]),
                last_contribution=datetime.fromisoformat(dev_dict["last_contribution"]),
                files_touched=dev_dict["files_touched"],
                expertise_areas=dev_dict["expertise_areas"]
            )
            datapoints["developers"].append(dev_dp)

        logger.info("DataPoint conversion complete",
                   commits=len(datapoints["commits"]),
                   prs=len(datapoints["pull_requests"]),
                   issues=len(datapoints["issues"]),
                   developers=len(datapoints["developers"]))

        return datapoints

    async def _ingest_to_cognee(self, datapoints: Dict[str, List[DataPoint]],
                                include_code: bool = True) -> Dict[str, Any]:
        """Ingest DataPoints into Cognee"""
        logger.info("Starting Cognee ingestion")

        ingestion_stats = {}

        try:
            # Convert DataPoints to JSON-serializable format for Cognee
            # Ingest commits as text with metadata
            if datapoints["commits"]:
                logger.info(f"Ingesting {len(datapoints['commits'])} commits")
                commit_texts = []
                for commit in datapoints["commits"]:
                    # Create a text representation with metadata
                    text = f"Commit {commit.sha[:8]} by {commit.author} on {commit.timestamp}\n"
                    text += f"Message: {commit.message}\n"
                    text += f"Files changed: {', '.join(commit.files_changed[:10])}"
                    if len(commit.files_changed) > 10:
                        text += f" and {len(commit.files_changed) - 10} more"
                    commit_texts.append(text)

                await cognee.add(
                    commit_texts,
                    dataset_name=self.dataset_name,
                    node_set=["github", "commits", "coderisk"]
                )
                ingestion_stats["commits"] = len(datapoints["commits"])

            # Ingest PRs as text
            if datapoints["pull_requests"]:
                logger.info(f"Ingesting {len(datapoints['pull_requests'])} pull requests")
                pr_texts = []
                for pr in datapoints["pull_requests"]:
                    text = f"Pull Request #{pr.number}: {pr.title}\n"
                    text += f"Author: {pr.author}, State: {pr.state}\n"
                    text += f"Description: {pr.description[:500]}\n"
                    text += f"Files changed: {', '.join(pr.files_changed[:10])}"
                    pr_texts.append(text)

                await cognee.add(
                    pr_texts,
                    dataset_name=self.dataset_name,
                    node_set=["github", "pull_requests", "coderisk"]
                )
                ingestion_stats["pull_requests"] = len(datapoints["pull_requests"])

            # Ingest issues as text
            if datapoints["issues"]:
                logger.info(f"Ingesting {len(datapoints['issues'])} issues")
                issue_texts = []
                for issue in datapoints["issues"]:
                    text = f"Issue #{issue.number}: {issue.title}\n"
                    text += f"Author: {issue.author}, State: {issue.state}\n"
                    text += f"Labels: {', '.join(issue.labels)}\n"
                    text += f"Description: {issue.description[:500]}"
                    if issue.is_bug:
                        text += "\n[BUG]"
                    if issue.is_incident:
                        text += f"\n[INCIDENT - Severity: {issue.severity}]"
                    issue_texts.append(text)

                await cognee.add(
                    issue_texts,
                    dataset_name=self.dataset_name,
                    node_set=["github", "issues", "coderisk"]
                )
                ingestion_stats["issues"] = len(datapoints["issues"])

            # Ingest developers as text
            if datapoints["developers"]:
                logger.info(f"Ingesting {len(datapoints['developers'])} developers")
                dev_texts = []
                for dev in datapoints["developers"]:
                    text = f"Developer: {dev.username} ({dev.name or 'N/A'})\n"
                    text += f"Email: {dev.email or 'N/A'}\n"
                    text += f"Stats: {dev.commits_count} commits, {dev.prs_authored} PRs authored, "
                    text += f"{dev.prs_reviewed} PRs reviewed\n"
                    text += f"Expertise: {', '.join(dev.expertise_areas)}"
                    dev_texts.append(text)

                await cognee.add(
                    dev_texts,
                    dataset_name=self.dataset_name,
                    node_set=["github", "developers", "coderisk"]
                )
                ingestion_stats["developers"] = len(datapoints["developers"])

            # Optionally include code graph
            if include_code and self.repo_path.exists():
                logger.info("Adding repository code to dataset")
                from cognee.api.v1.cognify.code_graph_pipeline import run_code_graph_pipeline

                async for result in run_code_graph_pipeline(
                    str(self.repo_path),
                    include_docs=False,
                    excluded_paths=["**/node_modules/**", "**/dist/**", "**/.git/**"]
                ):
                    logger.debug("Code graph progress", result=result)

                ingestion_stats["code_graph"] = True

        except Exception as e:
            logger.error(f"Ingestion failed: {e}")
            ingestion_stats["error"] = str(e)

        return ingestion_stats

    async def _build_knowledge_graph(self) -> Dict[str, Any]:
        """Build knowledge graph with temporal awareness"""
        logger.info("Building knowledge graph with Cognee")

        graph_stats = {}

        try:
            # Get ontology path if available
            ontology_path = self.repo_path.parent / "coderisk" / "ontology" / "coderisk_ontology.owl"

            # Cognify with temporal awareness
            cognify_kwargs = {
                "datasets": [self.dataset_name],  # Fixed: should be "datasets" not "dataset_name"
                "temporal_cognify": True  # Enable temporal awareness for time-based analysis
            }

            if ontology_path.exists():
                cognify_kwargs["ontology_file_path"] = str(ontology_path)
                logger.info("Using CodeRisk ontology for cognification")

            # Process the data with Cognee
            logger.info("Starting cognify process", datasets=[self.dataset_name])
            await cognee.cognify(**cognify_kwargs)
            logger.info("Cognify process completed successfully")

            graph_stats["status"] = "success"
            graph_stats["ontology_used"] = ontology_path.exists()

            logger.info("Knowledge graph built successfully")

        except Exception as e:
            logger.error(f"Graph building failed: {e}")
            graph_stats["status"] = "failed"
            graph_stats["error"] = str(e)

        return graph_stats

    async def query_github_data(self, query: str, search_type: str = "chunks") -> List[Dict[str, Any]]:
        """Query the ingested GitHub data"""
        try:
            if search_type == "code":
                results = await cognee.search(
                    query_type=SearchType.CODE,
                    query_text=query,
                    datasets=[self.dataset_name]  # Added dataset scoping
                )
            elif search_type == "temporal":
                results = await cognee.search(
                    query_type=SearchType.TEMPORAL,
                    query_text=query,
                    datasets=[self.dataset_name]  # Added dataset scoping
                )
            elif search_type == "graph":
                results = await cognee.search(
                    query_type=SearchType.GRAPH_COMPLETION,
                    query_text=query,
                    datasets=[self.dataset_name]  # Added dataset scoping
                )
            else:
                results = await cognee.search(
                    query_type=SearchType.CHUNKS,
                    query_text=query,
                    datasets=[self.dataset_name]  # Added dataset scoping
                )

            return results if isinstance(results, list) else [results]

        except Exception as e:
            logger.error(f"Query failed: {e}")
            return []

    async def get_commit_history(self, file_path: str) -> List[Dict[str, Any]]:
        """Get commit history for a specific file"""
        query = f"Find all commits that changed file {file_path}"
        return await self.query_github_data(query, "temporal")

    async def get_related_issues(self, file_paths: List[str]) -> List[Dict[str, Any]]:
        """Get issues related to specific files"""
        query = f"Find issues and bugs related to files: {', '.join(file_paths)}"
        return await self.query_github_data(query, "chunks")

    async def get_developer_expertise(self, username: str) -> Dict[str, Any]:
        """Get developer expertise information"""
        query = f"Find expertise areas and contributions for developer {username}"
        results = await self.query_github_data(query, "chunks")
        return results[0] if results else {}

    async def get_co_change_patterns(self, files: List[str]) -> List[Dict[str, Any]]:
        """Find files that frequently change together"""
        query = f"Find files that frequently change together with: {', '.join(files)}"
        return await self.query_github_data(query, "temporal")