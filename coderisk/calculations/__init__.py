"""
Advanced risk calculation engine with mathematical models

This module implements the core mathematical models and signals
from the CodeRisk v4 specification.
"""

from .mathematical_models import (
    calculate_delta_diffusion_blast_radius,
    calculate_hawkes_decayed_cochange,
    calculate_g2_surprise,
    TemporalDecayModel,
    PageRankDelta
)

from .core_signals import (
    RiskSignalCalculator,
    BlastRadiusSignal,
    CoChangeSignal,
    G2SurpriseSignal,
    OwnershipAuthorityMismatch,
    SpanCoreSignal,
    BridgeRiskSignal,
    IncidentAdjacencySignal,
    JITBaselinesSignal,
    GraphEdge,
    IncidentData,
    SpanCoreData
)

from .regression_scaling import (
    RegressionScalingModel,
    TeamFactor,
    CodebaseFactor,
    ChangeVelocityFactor,
    MigrationMultiplier
)

from .scoring_engine import (
    ScoringEngine,
    ConformalRiskControl,
    MonotoneScorer,
    AutoHighRules
)

__all__ = [
    # Mathematical models
    'calculate_delta_diffusion_blast_radius',
    'calculate_hawkes_decayed_cochange',
    'calculate_g2_surprise',
    'TemporalDecayModel',
    'PageRankDelta',

    # Core signals
    'RiskSignalCalculator',
    'BlastRadiusSignal',
    'CoChangeSignal',
    'G2SurpriseSignal',
    'OwnershipAuthorityMismatch',
    'SpanCoreSignal',
    'BridgeRiskSignal',
    'IncidentAdjacencySignal',
    'JITBaselinesSignal',
    'GraphEdge',
    'IncidentData',
    'SpanCoreData',

    # Regression scaling
    'RegressionScalingModel',
    'TeamFactor',
    'CodebaseFactor',
    'ChangeVelocityFactor',
    'MigrationMultiplier',

    # Scoring engine
    'ScoringEngine',
    'ConformalRiskControl',
    'MonotoneScorer',
    'AutoHighRules'
]