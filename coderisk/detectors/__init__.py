"""
CodeRisk Micro-Detectors Module

This module contains 9 specialized micro-detectors that analyze different
categories of risk in code changes. Each detector is designed to run in
50-150ms and provide deterministic risk assessment.

Detectors:
- api_break_risk: AST diff of public surface × importer count
- schema_risk: Migration ops (DROP/NOT NULL) × backfill checks
- dep_risk: Lockfile diff, major bumps, transitive changes
- perf_risk: Loop+I/O/DB detection weighted by centrality
- concurrency_risk: Shared state writes, lock order changes
- security_risk: Mini-SAST (unsafe patterns, secrets detection)
- config_risk: K8s/terraform/yaml risky changes × service fan-out
- test_gap_risk: Test coverage ratio with smoothing
- merge_risk: Overlap detection with upstream hotspots
"""

from abc import ABC, abstractmethod
from dataclasses import dataclass
from typing import List, Dict, Any, Optional
from pathlib import Path
import time
import asyncio


@dataclass
class DetectorResult:
    """Result from a micro-detector analysis"""
    score: float  # Risk score between 0.0 and 1.0
    reasons: List[str]  # Human-readable reasons for the score
    anchors: List[str]  # File paths and line numbers where risk was detected
    evidence: Dict[str, Any]  # Structured evidence data
    execution_time_ms: float  # Time taken to execute in milliseconds

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON serialization"""
        return {
            "score": round(self.score, 3),
            "reasons": self.reasons,
            "anchors": self.anchors,
            "evidence": self.evidence,
            "execution_time_ms": round(self.execution_time_ms, 2)
        }


@dataclass
class FileChange:
    """Represents a file change in the diff"""
    path: str
    change_type: str  # 'added', 'modified', 'deleted', 'renamed'
    lines_added: int
    lines_deleted: int
    hunks: List[Dict[str, Any]]  # Git diff hunks
    old_path: Optional[str] = None  # For renamed files


@dataclass
class ChangeContext:
    """Context about the changes being analyzed"""
    files_changed: List[FileChange]
    total_lines_added: int
    total_lines_deleted: int
    commit_sha: Optional[str] = None
    branch: Optional[str] = None
    author: Optional[str] = None


class BaseDetector(ABC):
    """Base class for all micro-detectors"""

    def __init__(self, repo_path: str):
        self.repo_path = Path(repo_path)
        self.name = self.__class__.__name__.lower().replace('detector', '')

    @abstractmethod
    async def analyze(self, context: ChangeContext) -> DetectorResult:
        """
        Analyze the changes and return a risk assessment.

        Must complete within 150ms (preferably 50-100ms).
        Returns a score between 0.0 (no risk) and 1.0 (maximum risk).
        """
        pass

    async def run_with_timeout(self, context: ChangeContext, timeout_ms: int = 150) -> DetectorResult:
        """Run the detector with a timeout"""
        start_time = time.perf_counter()

        try:
            # Run with timeout
            result = await asyncio.wait_for(
                self.analyze(context),
                timeout=timeout_ms / 1000.0
            )

            # Update execution time
            execution_time = (time.perf_counter() - start_time) * 1000
            result.execution_time_ms = execution_time

            return result

        except asyncio.TimeoutError:
            execution_time = (time.perf_counter() - start_time) * 1000
            return DetectorResult(
                score=0.0,
                reasons=[f"Detector timed out after {timeout_ms}ms"],
                anchors=[],
                evidence={"timeout": True, "timeout_ms": timeout_ms},
                execution_time_ms=execution_time
            )
        except Exception as e:
            execution_time = (time.perf_counter() - start_time) * 1000
            return DetectorResult(
                score=0.0,
                reasons=[f"Detector failed: {str(e)}"],
                anchors=[],
                evidence={"error": str(e)},
                execution_time_ms=execution_time
            )


class DetectorRegistry:
    """Registry for all available detectors"""

    def __init__(self):
        self._detectors = {}

    def register(self, detector_class):
        """Register a detector class"""
        name = detector_class.__name__.lower().replace('detector', '')
        self._detectors[name] = detector_class
        return detector_class

    def get_detector(self, name: str, repo_path: str) -> BaseDetector:
        """Get a detector instance by name"""
        if name not in self._detectors:
            raise ValueError(f"Unknown detector: {name}")
        return self._detectors[name](repo_path)

    def get_all_detectors(self, repo_path: str) -> List[BaseDetector]:
        """Get instances of all registered detectors"""
        return [detector_class(repo_path) for detector_class in self._detectors.values()]

    def list_detectors(self) -> List[str]:
        """List all registered detector names"""
        return list(self._detectors.keys())


# Global detector registry
detector_registry = DetectorRegistry()


# Import all detector implementations
from .api_detector import ApiBreakDetector
from .schema_detector import SchemaRiskDetector
from .dependency_detector import DependencyRiskDetector
from .performance_detector import PerformanceRiskDetector
from .concurrency_detector import ConcurrencyRiskDetector
from .security_detector import SecurityRiskDetector
from .config_detector import ConfigRiskDetector
from .test_detector import TestGapDetector
from .merge_detector import MergeRiskDetector


__all__ = [
    'BaseDetector',
    'DetectorResult',
    'FileChange',
    'ChangeContext',
    'DetectorRegistry',
    'detector_registry',
    'ApiBreakDetector',
    'SchemaRiskDetector',
    'DependencyRiskDetector',
    'PerformanceRiskDetector',
    'ConcurrencyRiskDetector',
    'SecurityRiskDetector',
    'ConfigRiskDetector',
    'TestGapDetector',
    'MergeRiskDetector'
]