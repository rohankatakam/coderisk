"""
Cognee data ingestion pipeline for CodeRisk

This module provides the data ingestion pipeline that feeds Git repository
history, commits, PRs, and issues into Cognee for knowledge graph construction.
"""

from .git_history_extractor import GitHistoryExtractor
from .cognee_processor import CogneeKnowledgeProcessor
from .data_models import (
    CommitDataPoint,
    PRDataPoint,
    IssueDataPoint,
    IncidentDataPoint,
    FileChangeDataPoint,
    DeveloperDataPoint,
)

__all__ = [
    "GitHistoryExtractor",
    "CogneeKnowledgeProcessor",
    "CommitDataPoint",
    "PRDataPoint",
    "IssueDataPoint",
    "IncidentDataPoint",
    "FileChangeDataPoint",
    "DeveloperDataPoint",
]