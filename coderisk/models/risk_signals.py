"""
Extended risk signal models for advanced mathematical calculations.

This module extends the existing RiskSignal model with additional
signal types specific to the advanced risk calculation engine.
"""

from dataclasses import dataclass
from typing import Dict, List, Optional, Tuple, Any
from datetime import datetime
import numpy as np

from .risk_assessment import RiskSignal


@dataclass
class BlastRadiusSignalData:
    """Detailed data for blast radius signal"""
    delta_dbr_score: float
    impacted_files: List[str]
    impact_scores: Dict[str, float]
    max_impact_delta: float
    graph_nodes_analyzed: int
    calculation_time_ms: float


@dataclass
class CoChangeSignalData:
    """Detailed data for co-change signal"""
    hdcc_score: float
    coupled_files: List[Tuple[str, float]]  # (file, coupling_strength)
    analysis_window_days: int
    cochange_pairs_found: int
    hawkes_decay_params: Tuple[float, float]  # (fast_decay, slow_decay)


@dataclass
class G2SurpriseSignalData:
    """Detailed data for G² surprise signal"""
    g2_score: float
    unusual_pairs: List[Tuple[str, str, float]]  # (file1, file2, g2_value)
    pairs_analyzed: int
    max_g2_value: float
    significance_threshold: float = 3.84  # χ² critical value


@dataclass
class OwnershipSignalData:
    """Detailed data for ownership authority mismatch"""
    oam_score: float
    file_mismatch_scores: Dict[str, float]
    ownership_mismatches: List[Dict[str, Any]]
    bus_factor_risks: List[str]
    experience_gaps: Dict[str, float]


@dataclass
class SpanCoreSignalData:
    """Detailed data for span-core temporal persistence"""
    span_core_score: float
    core_persistence_scores: Dict[str, float]
    temporal_spans: Dict[str, int]  # file -> span_days
    centrality_scores: Dict[str, float]
    high_risk_files: List[str]


@dataclass
class BridgeSignalData:
    """Detailed data for bridging centrality risk"""
    bridge_score: float
    bridging_centrality: Dict[str, float]
    subgraph_size: int
    critical_bridge_files: List[str]
    max_hops_analyzed: int


@dataclass
class IncidentAdjacencySignalData:
    """Detailed data for incident adjacency (GB-RRF)"""
    incident_score: float
    similar_incidents: List[Dict[str, Any]]
    bm25_scores: Dict[str, float]
    vector_similarity_scores: Dict[str, float]
    file_overlap_scores: Dict[str, float]
    rrf_fusion_weights: Tuple[float, float, float]  # (bm25, vector, overlap)


@dataclass
class JITBaselinesSignalData:
    """Detailed data for JIT baselines"""
    jit_score: float
    size_risk: float
    churn_risk: float
    entropy_risk: float
    auto_high_triggered: bool
    diff_complexity_metrics: Dict[str, float]


@dataclass
class MicroDetectorSignalData:
    """Base class for micro-detector signal data"""
    detector_name: str
    execution_time_ms: float
    anchors: List[str]  # File/line anchors for evidence
    confidence: float


@dataclass
class APIBreakSignalData(MicroDetectorSignalData):
    """API breaking change detector data"""
    breaking_changes: List[Dict[str, Any]]
    importer_count: int
    severity_score: float
    public_surface_changes: List[str]


@dataclass
class SchemaRiskSignalData(MicroDetectorSignalData):
    """Schema risk detector data"""
    risky_operations: List[str]  # DROP, NOT NULL, type changes
    affected_tables: List[str]
    has_backfill: bool
    has_rollback: bool
    table_fanout_proxy: int


@dataclass
class SecurityRiskSignalData(MicroDetectorSignalData):
    """Security risk detector data"""
    vulnerabilities_found: List[Dict[str, Any]]
    secret_patterns: List[str]
    unsafe_patterns: List[str]
    severity_levels: Dict[str, int]  # vulnerability -> severity (1-5)


@dataclass
class PerformanceRiskSignalData(MicroDetectorSignalData):
    """Performance risk detector data"""
    performance_issues: List[Dict[str, Any]]
    complexity_increases: Dict[str, float]
    hot_path_modifications: List[str]
    algorithmic_complexity_delta: str


@dataclass
class ConcurrencyRiskSignalData(MicroDetectorSignalData):
    """Concurrency risk detector data"""
    race_conditions: List[Dict[str, Any]]
    deadlock_risks: List[str]
    shared_state_modifications: List[str]
    lock_order_changes: List[str]


class AdvancedRiskSignal(RiskSignal):
    """Extended risk signal with detailed mathematical data"""

    def __init__(
        self,
        name: str,
        score: float,
        confidence: float,
        evidence: List[str],
        response_time_ms: int,
        detailed_data: Optional[Any] = None,
        mathematical_metadata: Optional[Dict[str, Any]] = None
    ):
        super().__init__(name, score, confidence, evidence, response_time_ms)
        self.detailed_data = detailed_data
        self.mathematical_metadata = mathematical_metadata or {}

    def get_blast_radius_data(self) -> Optional[BlastRadiusSignalData]:
        """Get blast radius detailed data if available"""
        if isinstance(self.detailed_data, BlastRadiusSignalData):
            return self.detailed_data
        return None

    def get_cochange_data(self) -> Optional[CoChangeSignalData]:
        """Get co-change detailed data if available"""
        if isinstance(self.detailed_data, CoChangeSignalData):
            return self.detailed_data
        return None

    def get_g2_surprise_data(self) -> Optional[G2SurpriseSignalData]:
        """Get G² surprise detailed data if available"""
        if isinstance(self.detailed_data, G2SurpriseSignalData):
            return self.detailed_data
        return None

    def get_ownership_data(self) -> Optional[OwnershipSignalData]:
        """Get ownership detailed data if available"""
        if isinstance(self.detailed_data, OwnershipSignalData):
            return self.detailed_data
        return None

    def get_span_core_data(self) -> Optional[SpanCoreSignalData]:
        """Get span-core detailed data if available"""
        if isinstance(self.detailed_data, SpanCoreSignalData):
            return self.detailed_data
        return None

    def get_bridge_data(self) -> Optional[BridgeSignalData]:
        """Get bridge risk detailed data if available"""
        if isinstance(self.detailed_data, BridgeSignalData):
            return self.detailed_data
        return None

    def get_incident_data(self) -> Optional[IncidentAdjacencySignalData]:
        """Get incident adjacency detailed data if available"""
        if isinstance(self.detailed_data, IncidentAdjacencySignalData):
            return self.detailed_data
        return None

    def get_jit_data(self) -> Optional[JITBaselinesSignalData]:
        """Get JIT baselines detailed data if available"""
        if isinstance(self.detailed_data, JITBaselinesSignalData):
            return self.detailed_data
        return None

    def is_mathematical_signal(self) -> bool:
        """Check if this is a mathematical signal (vs. heuristic)"""
        mathematical_signals = {
            'delta_diffusion_blast_radius',
            'hawkes_decayed_cochange',
            'g2_surprise',
            'ownership_authority_mismatch',
            'span_core_risk',
            'bridge_risk',
            'incident_adjacency',
            'jit_baselines'
        }
        return self.name in mathematical_signals

    def get_mathematical_complexity(self) -> str:
        """Get the mathematical complexity level of the signal"""
        complexity_map = {
            'delta_diffusion_blast_radius': 'High (PageRank computation)',
            'hawkes_decayed_cochange': 'Medium (Temporal decay models)',
            'g2_surprise': 'Medium (Statistical likelihood)',
            'ownership_authority_mismatch': 'Low (Aggregation functions)',
            'span_core_risk': 'Medium (Graph centrality)',
            'bridge_risk': 'High (Betweenness centrality)',
            'incident_adjacency': 'High (Multi-modal fusion)',
            'jit_baselines': 'Low (Statistical aggregation)'
        }
        return complexity_map.get(self.name, 'Low (Heuristic)')

    def get_graph_complexity(self) -> Dict[str, Any]:
        """Get graph computational complexity metrics"""
        if 'graph_metrics' in self.mathematical_metadata:
            return self.mathematical_metadata['graph_metrics']

        # Default complexity metrics
        return {
            'nodes_analyzed': 0,
            'edges_analyzed': 0,
            'max_hops': 2,
            'time_complexity': 'O(V + E)',
            'space_complexity': 'O(V)',
            'bounded_query': True
        }

    def get_temporal_complexity(self) -> Dict[str, Any]:
        """Get temporal analysis complexity metrics"""
        if 'temporal_metrics' in self.mathematical_metadata:
            return self.mathematical_metadata['temporal_metrics']

        # Default temporal metrics
        return {
            'analysis_window_days': 90,
            'decay_functions': [],
            'time_series_length': 0,
            'temporal_resolution': 'daily'
        }

    def to_detailed_dict(self) -> Dict[str, Any]:
        """Convert to detailed dictionary including mathematical metadata"""
        base_dict = {
            'name': self.name,
            'score': self.score,
            'confidence': self.confidence,
            'evidence': self.evidence,
            'response_time_ms': self.response_time_ms,
            'is_mathematical': self.is_mathematical_signal(),
            'complexity': self.get_mathematical_complexity()
        }

        # Add detailed data if available
        if self.detailed_data:
            base_dict['detailed_data'] = self.detailed_data.__dict__

        # Add mathematical metadata
        base_dict['mathematical_metadata'] = self.mathematical_metadata

        # Add complexity metrics
        base_dict['graph_complexity'] = self.get_graph_complexity()
        base_dict['temporal_complexity'] = self.get_temporal_complexity()

        return base_dict


@dataclass
class RiskSignalSuite:
    """Complete suite of risk signals for a change"""
    # Core mathematical signals
    blast_radius: Optional[AdvancedRiskSignal] = None
    cochange: Optional[AdvancedRiskSignal] = None
    g2_surprise: Optional[AdvancedRiskSignal] = None
    ownership: Optional[AdvancedRiskSignal] = None
    span_core: Optional[AdvancedRiskSignal] = None
    bridge_risk: Optional[AdvancedRiskSignal] = None
    incident_adjacency: Optional[AdvancedRiskSignal] = None
    jit_baselines: Optional[AdvancedRiskSignal] = None

    # Micro-detectors
    api_break: Optional[AdvancedRiskSignal] = None
    schema_risk: Optional[AdvancedRiskSignal] = None
    security_risk: Optional[AdvancedRiskSignal] = None
    performance_risk: Optional[AdvancedRiskSignal] = None
    concurrency_risk: Optional[AdvancedRiskSignal] = None
    config_risk: Optional[AdvancedRiskSignal] = None
    test_gap: Optional[AdvancedRiskSignal] = None
    merge_risk: Optional[AdvancedRiskSignal] = None

    def get_all_signals(self) -> List[AdvancedRiskSignal]:
        """Get all non-None signals"""
        signals = []
        for field_name in self.__dataclass_fields__:
            signal = getattr(self, field_name)
            if signal is not None:
                signals.append(signal)
        return signals

    def get_mathematical_signals(self) -> List[AdvancedRiskSignal]:
        """Get only mathematical signals"""
        return [signal for signal in self.get_all_signals() if signal.is_mathematical_signal()]

    def get_micro_detector_signals(self) -> List[AdvancedRiskSignal]:
        """Get only micro-detector signals"""
        return [signal for signal in self.get_all_signals() if not signal.is_mathematical_signal()]

    def get_total_execution_time(self) -> int:
        """Get total execution time for all signals"""
        return sum(signal.response_time_ms for signal in self.get_all_signals())

    def get_highest_risk_signal(self) -> Optional[AdvancedRiskSignal]:
        """Get the signal with highest risk score"""
        signals = self.get_all_signals()
        if not signals:
            return None
        return max(signals, key=lambda s: s.score * s.confidence)

    def get_signal_summary(self) -> Dict[str, Any]:
        """Get summary statistics for the signal suite"""
        all_signals = self.get_all_signals()
        if not all_signals:
            return {}

        mathematical_signals = self.get_mathematical_signals()
        micro_detector_signals = self.get_micro_detector_signals()

        return {
            'total_signals': len(all_signals),
            'mathematical_signals': len(mathematical_signals),
            'micro_detector_signals': len(micro_detector_signals),
            'total_execution_time_ms': self.get_total_execution_time(),
            'average_score': sum(s.score for s in all_signals) / len(all_signals),
            'average_confidence': sum(s.confidence for s in all_signals) / len(all_signals),
            'highest_risk_signal': self.get_highest_risk_signal().name if self.get_highest_risk_signal() else None,
            'signals_above_threshold': len([s for s in all_signals if s.score > 0.6])
        }