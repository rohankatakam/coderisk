"""
Risk assessment models and data structures
"""

from enum import Enum
from typing import List, Dict, Optional, Any
from pydantic import BaseModel
from datetime import datetime


class RiskTier(str, Enum):
    """Risk tier classification"""
    LOW = "LOW"
    MEDIUM = "MEDIUM"
    HIGH = "HIGH"
    CRITICAL = "CRITICAL"


class RiskSignal(BaseModel):
    """Individual risk signal"""
    name: str
    score: float  # 0.0 to 1.0
    confidence: float  # 0.0 to 1.0
    evidence: List[str]
    response_time_ms: Optional[int] = None


class ChangeContext(BaseModel):
    """Context about the code changes being assessed"""
    files_changed: List[str]
    lines_added: int
    lines_deleted: int
    functions_changed: List[str]
    is_migration: bool = False
    commit_message: Optional[str] = None
    author: Optional[str] = None
    timestamp: Optional[datetime] = None


class RiskEvidence(BaseModel):
    """Evidence supporting a risk assessment"""
    type: str  # "file_path", "incident", "pattern", etc.
    description: str
    file_path: Optional[str] = None
    line_number: Optional[int] = None
    confidence: float = 0.0


class RiskRecommendation(BaseModel):
    """Actionable recommendation to mitigate risk"""
    action: str
    priority: str  # "high", "medium", "low"
    description: str
    estimated_effort: Optional[str] = None


class RiskAssessment(BaseModel):
    """Complete risk assessment result"""

    # Core risk metrics
    tier: RiskTier
    score: float  # 0-100
    confidence: float  # 0.0 to 1.0

    # Risk breakdown
    signals: List[RiskSignal]
    categories: Dict[str, float]  # blast_radius, co_change, incident_adjacency, etc.

    # Context
    change_context: ChangeContext

    # Evidence and explanations
    evidence: List[RiskEvidence]
    recommendations: List[RiskRecommendation]

    # Regression scaling factors
    team_factor: float = 1.0
    codebase_factor: float = 1.0
    change_velocity: float = 1.0
    migration_multiplier: float = 1.0

    # Metadata
    assessment_time_ms: Optional[int] = None
    created_at: datetime

    @property
    def total_regression_risk(self) -> float:
        """Calculate total regression risk using scaling formula"""
        base_risk = self.score / 100.0
        return base_risk * self.team_factor * self.codebase_factor * self.change_velocity * self.migration_multiplier

    @property
    def top_concerns(self) -> List[str]:
        """Get top 3 risk concerns"""
        sorted_signals = sorted(self.signals, key=lambda x: x.score * x.confidence, reverse=True)
        return [signal.name for signal in sorted_signals[:3]]

    def get_explanation(self) -> str:
        """Get human-readable risk explanation"""
        if self.tier == RiskTier.LOW:
            return f"Low risk change ({self.score:.0f}/100). Primary concerns: {', '.join(self.top_concerns[:2])}"
        elif self.tier == RiskTier.MEDIUM:
            return f"Medium risk change ({self.score:.0f}/100). Key issues: {', '.join(self.top_concerns)}"
        elif self.tier == RiskTier.HIGH:
            return f"High risk change ({self.score:.0f}/100). Major concerns: {', '.join(self.top_concerns)}"
        else:  # CRITICAL
            return f"CRITICAL risk change ({self.score:.0f}/100). Immediate issues: {', '.join(self.top_concerns)}"