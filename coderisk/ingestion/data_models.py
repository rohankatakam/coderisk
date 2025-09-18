"""
Cognee DataPoint models for structured knowledge ingestion

These models define the structure of data that will be ingested into Cognee
for building the knowledge graph and enabling semantic search.
"""

from typing import List, Optional, Dict, Any
from datetime import datetime
from pydantic import BaseModel


# Base DataPoint class for Cognee integration
class DataPoint(BaseModel):
    """Base DataPoint class for Cognee compatibility"""
    metadata: Dict[str, Any] = {}


class CommitDataPoint(DataPoint):
    """Structured commit data for Cognee ingestion"""

    sha: str
    message: str
    timestamp: datetime
    author: str
    author_email: str
    files_changed: List[str]
    lines_added: int
    lines_deleted: int
    is_merge: bool = False
    is_revert: bool = False
    is_hotfix: bool = False
    is_security_fix: bool = False
    branch: Optional[str] = None
    parents: List[str] = []

    # Risk indicators extracted from commit
    has_db_migration: bool = False
    touches_auth: bool = False
    touches_config: bool = False
    large_change: bool = False  # >200 lines

    metadata: Dict[str, Any] = {
        "index_fields": ["message", "author", "files_changed"],
        "vector_fields": ["message"],
        "temporal_field": "timestamp"
    }


class PRDataPoint(DataPoint):
    """Pull request data for Cognee ingestion"""

    pr_id: int
    title: str
    description: str
    state: str  # "open", "closed", "merged"
    created_at: datetime
    updated_at: datetime
    merged_at: Optional[datetime] = None
    closed_at: Optional[datetime] = None

    author: str
    author_email: str
    reviewers: List[str] = []
    assignees: List[str] = []

    files_touched: int
    lines_added: int
    lines_deleted: int
    commits_count: int

    review_comments: List[str] = []
    review_thread_count: int = 0
    approval_count: int = 0
    change_request_count: int = 0

    labels: List[str] = []

    # Risk indicators
    is_breaking_change: bool = False
    has_security_review: bool = False
    emergency_merge: bool = False

    metadata: Dict[str, Any] = {
        "index_fields": ["title", "description", "author", "review_comments"],
        "vector_fields": ["title", "description"],
        "temporal_field": "created_at"
    }


class IssueDataPoint(DataPoint):
    """Issue/incident data for Cognee ingestion"""

    issue_id: int
    title: str
    description: str
    state: str  # "open", "closed"
    severity: str  # "P0", "P1", "P2", "P3", "LOW", "MEDIUM", "HIGH", "CRITICAL"
    issue_type: str  # "bug", "incident", "feature", "task"

    created_at: datetime
    updated_at: datetime
    closed_at: Optional[datetime] = None

    author: str
    assignees: List[str] = []
    labels: List[str] = []

    # Incident-specific fields
    is_production_incident: bool = False
    downtime_minutes: Optional[int] = None
    affected_users: Optional[int] = None
    root_cause: Optional[str] = None

    # Related entities
    related_commits: List[str] = []  # SHA references
    related_prs: List[int] = []
    affected_files: List[str] = []

    metadata: Dict[str, Any] = {
        "index_fields": ["title", "description", "root_cause", "affected_files"],
        "vector_fields": ["title", "description", "root_cause"],
        "temporal_field": "created_at"
    }


class FileChangeDataPoint(DataPoint):
    """Individual file change data for detailed analysis"""

    file_path: str
    commit_sha: str
    change_type: str  # "added", "modified", "deleted", "renamed"
    lines_added: int
    lines_deleted: int
    complexity_score: Optional[float] = None

    # Code structure changes
    functions_added: List[str] = []
    functions_modified: List[str] = []
    functions_deleted: List[str] = []
    classes_added: List[str] = []
    classes_modified: List[str] = []
    classes_deleted: List[str] = []
    imports_added: List[str] = []
    imports_removed: List[str] = []

    # File metadata
    language: Optional[str] = None
    is_test_file: bool = False
    is_config_file: bool = False
    is_migration_file: bool = False

    # Content diff for semantic analysis
    diff_content: str = ""

    metadata: Dict[str, Any] = {
        "index_fields": ["file_path", "functions_modified", "classes_modified"],
        "vector_fields": ["diff_content"],
        "entity_type": "file_change"
    }


class DeveloperDataPoint(DataPoint):
    """Developer activity and expertise data"""

    email: str
    name: str

    # Activity metrics
    total_commits: int = 0
    commits_last_30_days: int = 0
    commits_last_90_days: int = 0

    # Expertise areas (file paths they frequently modify)
    expertise_files: List[str] = []
    expertise_languages: List[str] = []

    # Collaboration metrics
    frequent_collaborators: List[str] = []
    review_participation: int = 0  # Number of PRs reviewed

    # Risk indicators
    incident_association_count: int = 0  # How many incidents linked to their commits
    revert_rate: float = 0.0  # Percentage of commits that get reverted
    bug_introduction_rate: float = 0.0  # Rate of bug-introducing commits

    # Team membership
    team: Optional[str] = None
    seniority_level: str = "unknown"  # "junior", "mid", "senior", "staff"

    metadata: Dict[str, Any] = {
        "index_fields": ["name", "email", "expertise_files", "team"],
        "entity_type": "developer"
    }


class SecurityDataPoint(DataPoint):
    """Security-related data for vulnerability analysis"""

    vulnerability_id: str
    severity: str  # "LOW", "MEDIUM", "HIGH", "CRITICAL"
    vulnerability_type: str  # "SQL_INJECTION", "XSS", "AUTH_BYPASS", etc.

    description: str
    affected_files: List[str] = []
    affected_functions: List[str] = []

    discovered_at: datetime
    fixed_at: Optional[datetime] = None
    fix_commit_sha: Optional[str] = None

    # CVE information if applicable
    cve_id: Optional[str] = None
    cvss_score: Optional[float] = None

    # Pattern for similarity matching
    vulnerable_pattern: str = ""
    fix_pattern: str = ""

    metadata: Dict[str, Any] = {
        "index_fields": ["vulnerability_type", "affected_files", "affected_functions"],
        "vector_fields": ["description", "vulnerable_pattern", "fix_pattern"],
        "temporal_field": "discovered_at"
    }


class IncidentDataPoint(DataPoint):
    """Production incident data for historical pattern analysis"""

    incident_id: str
    title: str
    description: str
    severity: str  # "P0", "P1", "P2", "P3"

    started_at: datetime
    resolved_at: Optional[datetime] = None
    detection_time_minutes: Optional[int] = None
    resolution_time_minutes: Optional[int] = None

    # Impact metrics
    affected_users: Optional[int] = None
    revenue_impact: Optional[float] = None
    downtime_minutes: Optional[int] = None

    # Root cause analysis
    root_cause: Optional[str] = None
    contributing_factors: List[str] = []

    # Related entities
    trigger_commit_sha: Optional[str] = None
    related_commits: List[str] = []
    affected_services: List[str] = []
    affected_files: List[str] = []

    # Response team
    oncall_engineer: Optional[str] = None
    incident_commander: Optional[str] = None
    responders: List[str] = []

    # Post-incident data
    post_mortem_url: Optional[str] = None
    action_items: List[str] = []

    metadata: Dict[str, Any] = {
        "index_fields": ["title", "root_cause", "affected_files", "affected_services"],
        "vector_fields": ["description", "root_cause"],
        "temporal_field": "started_at"
    }


class DependencyDataPoint(DataPoint):
    """Code dependency relationships for impact analysis"""

    source_file: str
    target_file: str
    dependency_type: str  # "import", "function_call", "class_inheritance", "data_dependency"

    source_function: Optional[str] = None
    target_function: Optional[str] = None
    source_class: Optional[str] = None
    target_class: Optional[str] = None

    # Dependency strength metrics
    call_frequency: int = 1  # How often this dependency is used
    is_critical: bool = False  # Whether this is a critical dependency

    # Change co-occurrence
    co_change_frequency: int = 0  # How often these files change together
    last_co_change: Optional[datetime] = None

    metadata: Dict[str, Any] = {
        "index_fields": ["source_file", "target_file", "dependency_type"],
        "entity_type": "dependency"
    }


# Risk-specific DataPoint for pattern learning
class RiskPatternDataPoint(DataPoint):
    """Risk patterns learned from historical data"""

    pattern_id: str
    pattern_type: str  # "file_pattern", "commit_pattern", "change_pattern"
    description: str

    # Pattern definition
    trigger_conditions: Dict[str, Any] = {}
    risk_indicators: List[str] = []

    # Statistical data
    occurrences: int = 0
    incident_correlation: float = 0.0  # 0.0 to 1.0
    false_positive_rate: float = 0.0

    # Example cases
    positive_examples: List[str] = []  # Commit SHAs where this pattern led to incidents
    negative_examples: List[str] = []  # Commit SHAs where pattern was benign

    created_at: datetime
    last_updated: datetime

    metadata: Dict[str, Any] = {
        "index_fields": ["pattern_type", "description", "risk_indicators"],
        "vector_fields": ["description"],
        "entity_type": "risk_pattern"
    }