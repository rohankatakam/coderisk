"""
Core mathematical models for risk calculation.

Implements the advanced mathematical models from docs/risk_math.md:
- Δ-Diffusion Blast Radius (ΔDBR)
- Hawkes-Decayed Co-Change (HDCC)
- G² Surprise (Dunning log-likelihood)
"""

import math
from typing import Dict, List, Tuple, Optional, Set
from dataclasses import dataclass
from datetime import datetime, timedelta
import logging

try:
    import numpy as np
    NUMPY_AVAILABLE = True
except ImportError:
    NUMPY_AVAILABLE = False
    # Fallback for basic array operations
    class np:
        @staticmethod
        def zeros(shape):
            if isinstance(shape, tuple):
                return [[0.0 for _ in range(shape[1])] for _ in range(shape[0])]
            return [0.0 for _ in range(shape)]

        @staticmethod
        def ones(n):
            return [1.0 for _ in range(n)]

        @staticmethod
        def array(data):
            return data

        class linalg:
            @staticmethod
            def norm(arr, ord=None):
                if ord == 1:
                    return sum(abs(x) for x in arr)
                return math.sqrt(sum(x*x for x in arr))

logger = logging.getLogger(__name__)


@dataclass
class GraphEdge:
    """Represents an edge in the dependency or co-change graph"""
    source: str
    target: str
    weight: float
    edge_type: str  # 'import', 'cochange'
    timestamp: Optional[datetime] = None


@dataclass
class PageRankResult:
    """Result of PageRank calculation"""
    scores: Dict[str, float]
    iterations: int
    convergence_time_ms: float


class TemporalDecayModel:
    """Implements temporal decay functions for time-weighted analysis"""

    def __init__(self, fast_decay: float = 0.1, slow_decay: float = 0.01):
        """
        Initialize decay model.

        Args:
            fast_decay: Fast decay rate for recent changes (λ_fast)
            slow_decay: Slow decay rate for long-term patterns (λ_slow)
        """
        self.fast_decay = fast_decay
        self.slow_decay = slow_decay

    def hawkes_kernel(self, time_delta_days: float) -> float:
        """
        Hawkes process kernel for temporal decay.

        Args:
            time_delta_days: Days since the event

        Returns:
            Decay weight
        """
        if time_delta_days < 0:
            return 0.0

        # Two-timescale Hawkes kernel
        fast_component = math.exp(-self.fast_decay * time_delta_days)
        slow_component = 0.3 * math.exp(-self.slow_decay * time_delta_days)

        return fast_component + slow_component

    def exponential_decay(self, time_delta_days: float, decay_rate: float = None) -> float:
        """
        Simple exponential decay.

        Args:
            time_delta_days: Days since the event
            decay_rate: Override decay rate

        Returns:
            Decay weight
        """
        rate = decay_rate if decay_rate is not None else self.fast_decay
        return math.exp(-rate * time_delta_days)


class PageRankDelta:
    """Calculates PageRank deltas for impact assessment"""

    def __init__(self, damping_factor: float = 0.85, max_iterations: int = 100,
                 tolerance: float = 1e-6):
        self.damping_factor = damping_factor
        self.max_iterations = max_iterations
        self.tolerance = tolerance

    def calculate_baseline_pagerank(self, edges: List[GraphEdge],
                                  nodes: Set[str]) -> PageRankResult:
        """
        Calculate baseline PageRank scores.

        Args:
            edges: Graph edges
            nodes: All nodes in the graph

        Returns:
            PageRank results
        """
        start_time = datetime.now()
        node_list = list(nodes)
        n = len(node_list)

        if n == 0:
            return PageRankResult({}, 0, 0.0)

        # Build adjacency matrix
        node_to_idx = {node: i for i, node in enumerate(node_list)}
        adj_matrix = np.zeros((n, n))

        for edge in edges:
            if edge.source in node_to_idx and edge.target in node_to_idx:
                i, j = node_to_idx[edge.source], node_to_idx[edge.target]
                adj_matrix[i][j] = edge.weight

        # Normalize by outgoing degree
        row_sums = adj_matrix.sum(axis=1)
        for i in range(n):
            if row_sums[i] > 0:
                adj_matrix[i] /= row_sums[i]

        # Initialize PageRank vector
        pr = np.ones(n) / n

        # Power iteration
        for iteration in range(self.max_iterations):
            prev_pr = pr.copy()
            pr = (1 - self.damping_factor) / n + self.damping_factor * adj_matrix.T @ pr

            # Check convergence
            if np.linalg.norm(pr - prev_pr, 1) < self.tolerance:
                break

        # Convert back to dictionary
        scores = {node_list[i]: pr[i] for i in range(n)}

        end_time = datetime.now()
        calc_time = (end_time - start_time).total_seconds() * 1000

        return PageRankResult(scores, iteration + 1, calc_time)

    def calculate_delta(self, baseline_scores: Dict[str, float],
                       modified_edges: List[GraphEdge],
                       nodes: Set[str]) -> Dict[str, float]:
        """
        Calculate PageRank delta after edge modifications.

        Args:
            baseline_scores: Original PageRank scores
            modified_edges: Modified edge set
            nodes: All nodes

        Returns:
            Dictionary of PageRank deltas
        """
        modified_result = self.calculate_baseline_pagerank(modified_edges, nodes)
        modified_scores = modified_result.scores

        deltas = {}
        for node in nodes:
            baseline = baseline_scores.get(node, 0.0)
            modified = modified_scores.get(node, 0.0)
            deltas[node] = abs(modified - baseline)

        return deltas


def calculate_delta_diffusion_blast_radius(
    changed_files: List[str],
    import_edges: List[GraphEdge],
    cochange_edges: List[GraphEdge],
    max_hops: int = 2,
    timeout_ms: int = 500
) -> Tuple[float, Dict[str, float], Dict[str, any]]:
    """
    Calculate Δ-Diffusion Blast Radius (ΔDBR).

    Measures the PageRank delta from dependency changes using both
    import relationships and co-change patterns.

    Args:
        changed_files: List of files being modified
        import_edges: Import dependency edges
        cochange_edges: Co-change relationship edges
        max_hops: Maximum hops for graph traversal (≤2)
        timeout_ms: Timeout for calculation

    Returns:
        Tuple of (blast_radius_score, file_impact_scores, evidence)
    """
    start_time = datetime.now()
    evidence = {
        'changed_files': changed_files,
        'calculation_time_ms': 0,
        'impacted_files': [],
        'max_impact_delta': 0.0
    }

    try:
        # Combine import and co-change edges
        all_edges = import_edges + cochange_edges

        # Extract all nodes within max_hops
        all_nodes = set()
        for edge in all_edges:
            all_nodes.add(edge.source)
            all_nodes.add(edge.target)

        # Filter to nodes within max_hops of changed files
        relevant_nodes = set(changed_files)
        for _ in range(max_hops):
            new_nodes = set()
            for edge in all_edges:
                if edge.source in relevant_nodes:
                    new_nodes.add(edge.target)
                if edge.target in relevant_nodes:
                    new_nodes.add(edge.source)
            relevant_nodes.update(new_nodes)

        # Calculate baseline PageRank
        pr_calculator = PageRankDelta()
        baseline_result = pr_calculator.calculate_baseline_pagerank(all_edges, relevant_nodes)

        # Check timeout
        elapsed = (datetime.now() - start_time).total_seconds() * 1000
        if elapsed > timeout_ms:
            logger.warning(f"ΔDBR calculation timeout after {elapsed}ms")
            return 0.5, {}, evidence

        # Simulate edge modifications by removing changed files
        modified_edges = [edge for edge in all_edges
                         if edge.source not in changed_files and edge.target not in changed_files]

        # Calculate delta
        deltas = pr_calculator.calculate_delta(baseline_result.scores, modified_edges, relevant_nodes)

        # Calculate blast radius score
        max_delta = max(deltas.values()) if deltas else 0.0
        total_impact = sum(deltas.values())

        # Normalize to [0, 1] scale
        blast_radius_score = min(1.0, total_impact * 10)  # Scale factor

        # Identify highly impacted files
        impacted_files = [file for file, delta in deltas.items()
                         if delta > 0.01 and file not in changed_files]

        evidence.update({
            'calculation_time_ms': elapsed,
            'impacted_files': impacted_files[:10],  # Top 10
            'max_impact_delta': max_delta,
            'total_nodes_analyzed': len(relevant_nodes)
        })

        return blast_radius_score, deltas, evidence

    except Exception as e:
        logger.error(f"ΔDBR calculation failed: {e}")
        elapsed = (datetime.now() - start_time).total_seconds() * 1000
        evidence['calculation_time_ms'] = elapsed
        evidence['error'] = str(e)
        return 0.5, {}, evidence


def calculate_hawkes_decayed_cochange(
    changed_files: List[str],
    cochange_history: List[Tuple[str, str, datetime, float]],
    analysis_window_days: int = 90,
    fast_decay: float = 0.1,
    slow_decay: float = 0.01
) -> Tuple[float, Dict[str, float], Dict[str, any]]:
    """
    Calculate Hawkes-Decayed Co-Change (HDCC) risk.

    Uses two-timescale evolutionary coupling with temporal decay.

    Args:
        changed_files: Files being modified
        cochange_history: List of (file1, file2, timestamp, strength) tuples
        analysis_window_days: Analysis window
        fast_decay: Fast decay rate
        slow_decay: Slow decay rate

    Returns:
        Tuple of (hdcc_score, file_coupling_scores, evidence)
    """
    current_time = datetime.now()
    cutoff_time = current_time - timedelta(days=analysis_window_days)

    decay_model = TemporalDecayModel(fast_decay, slow_decay)
    evidence = {
        'analysis_window_days': analysis_window_days,
        'cochange_pairs_found': 0,
        'highest_coupling_strength': 0.0,
        'coupled_files': []
    }

    try:
        # Filter history to analysis window
        recent_history = [
            (f1, f2, ts, strength) for f1, f2, ts, strength in cochange_history
            if ts >= cutoff_time
        ]

        # Calculate coupling strengths with temporal decay
        coupling_scores = {}

        for file1, file2, timestamp, base_strength in recent_history:
            # Check if either file is being changed
            if file1 in changed_files or file2 in changed_files:
                time_delta = (current_time - timestamp).days
                decay_weight = decay_model.hawkes_kernel(time_delta)

                # Calculate weighted coupling strength
                weighted_strength = base_strength * decay_weight

                # Store coupling for both directions
                other_file = file2 if file1 in changed_files else file1
                if other_file not in coupling_scores:
                    coupling_scores[other_file] = 0.0
                coupling_scores[other_file] += weighted_strength

        # Calculate overall HDCC score
        if not coupling_scores:
            hdcc_score = 0.1  # Low risk if no coupling history
        else:
            max_coupling = max(coupling_scores.values())
            avg_coupling = sum(coupling_scores.values()) / len(coupling_scores)

            # Combine max and average for final score
            hdcc_score = min(1.0, 0.7 * max_coupling + 0.3 * avg_coupling)

        # Prepare evidence
        coupled_files = sorted(coupling_scores.items(), key=lambda x: x[1], reverse=True)
        evidence.update({
            'cochange_pairs_found': len(recent_history),
            'highest_coupling_strength': max(coupling_scores.values()) if coupling_scores else 0.0,
            'coupled_files': coupled_files[:5]  # Top 5
        })

        return hdcc_score, coupling_scores, evidence

    except Exception as e:
        logger.error(f"HDCC calculation failed: {e}")
        evidence['error'] = str(e)
        return 0.3, {}, evidence


def calculate_g2_surprise(
    file_pairs: List[Tuple[str, str]],
    expected_frequencies: Dict[Tuple[str, str], float],
    observed_frequencies: Dict[Tuple[str, str], int],
    min_frequency: int = 2
) -> Tuple[float, List[Tuple[str, str, float]], Dict[str, any]]:
    """
    Calculate G² Surprise using Dunning log-likelihood.

    Identifies unusual file pairs based on co-change patterns.

    Args:
        file_pairs: Pairs of files to analyze
        expected_frequencies: Expected co-change frequencies
        observed_frequencies: Observed co-change frequencies
        min_frequency: Minimum frequency threshold

    Returns:
        Tuple of (surprise_score, unusual_pairs, evidence)
    """
    evidence = {
        'pairs_analyzed': len(file_pairs),
        'unusual_pairs_found': 0,
        'max_g2_value': 0.0
    }

    try:
        unusual_pairs = []
        g2_values = []

        for file1, file2 in file_pairs:
            pair = (file1, file2)
            reverse_pair = (file2, file1)

            # Get observed and expected frequencies
            observed = observed_frequencies.get(pair, 0) + observed_frequencies.get(reverse_pair, 0)
            expected = expected_frequencies.get(pair, 1.0) + expected_frequencies.get(reverse_pair, 1.0)

            if observed < min_frequency:
                continue

            # Calculate G² (log-likelihood ratio)
            if expected > 0:
                g2 = 2 * observed * math.log(observed / expected)
                g2_values.append(g2)

                # Store unusual pairs (high G² values)
                if g2 > 3.84:  # χ² critical value for p < 0.05
                    unusual_pairs.append((file1, file2, g2))

        # Calculate surprise score
        if not g2_values:
            surprise_score = 0.1
        else:
            max_g2 = max(g2_values)
            avg_g2 = sum(g2_values) / len(g2_values)

            # Normalize G² to [0, 1] scale
            surprise_score = min(1.0, (max_g2 + avg_g2) / 20.0)

        # Sort unusual pairs by G² value
        unusual_pairs.sort(key=lambda x: x[2], reverse=True)

        evidence.update({
            'unusual_pairs_found': len(unusual_pairs),
            'max_g2_value': max(g2_values) if g2_values else 0.0
        })

        return surprise_score, unusual_pairs[:10], evidence  # Top 10 unusual pairs

    except Exception as e:
        logger.error(f"G² surprise calculation failed: {e}")
        evidence['error'] = str(e)
        return 0.2, [], evidence


def calculate_ownership_authority_mismatch(
    changed_files: List[str],
    file_ownership_history: Dict[str, List[Tuple[str, datetime, int]]],
    team_experience: Dict[str, float],
    analysis_window_days: int = 180
) -> Tuple[float, Dict[str, float], Dict[str, any]]:
    """
    Calculate Ownership Authority Mismatch (OAM).

    Measures misalignment between file ownership patterns and change authority.

    Args:
        changed_files: Files being modified
        file_ownership_history: File -> [(author, timestamp, lines_changed)]
        team_experience: Author -> experience_score
        analysis_window_days: Analysis window

    Returns:
        Tuple of (oam_score, file_mismatch_scores, evidence)
    """
    current_time = datetime.now()
    cutoff_time = current_time - timedelta(days=analysis_window_days)

    evidence = {
        'files_analyzed': len(changed_files),
        'ownership_mismatches': [],
        'bus_factor_risks': []
    }

    try:
        file_mismatch_scores = {}

        for file_path in changed_files:
            if file_path not in file_ownership_history:
                file_mismatch_scores[file_path] = 0.5  # Unknown ownership
                continue

            # Filter to recent history
            recent_changes = [
                (author, ts, lines) for author, ts, lines in file_ownership_history[file_path]
                if ts >= cutoff_time
            ]

            if not recent_changes:
                file_mismatch_scores[file_path] = 0.3
                continue

            # Calculate ownership metrics
            total_lines = sum(lines for _, _, lines in recent_changes)
            author_contributions = {}

            for author, _, lines in recent_changes:
                if author not in author_contributions:
                    author_contributions[author] = 0
                author_contributions[author] += lines

            # Calculate ownership concentration (bus factor)
            sorted_contributions = sorted(author_contributions.values(), reverse=True)
            if len(sorted_contributions) == 1:
                bus_factor = 0.9  # Single author risk
            else:
                top_author_ratio = sorted_contributions[0] / total_lines
                bus_factor = min(0.9, top_author_ratio)

            # Calculate experience mismatch
            weighted_experience = 0.0
            for author, contribution in author_contributions.items():
                author_exp = team_experience.get(author, 0.5)  # Default mid experience
                weight = contribution / total_lines
                weighted_experience += weight * author_exp

            # Authority mismatch: low experience + high ownership concentration
            experience_gap = max(0.0, 0.8 - weighted_experience)
            authority_mismatch = (bus_factor + experience_gap) / 2

            file_mismatch_scores[file_path] = authority_mismatch

            # Track evidence
            if authority_mismatch > 0.6:
                evidence['ownership_mismatches'].append({
                    'file': file_path,
                    'mismatch_score': authority_mismatch,
                    'bus_factor': bus_factor,
                    'experience_gap': experience_gap
                })

        # Calculate overall OAM score
        if not file_mismatch_scores:
            oam_score = 0.3
        else:
            oam_score = sum(file_mismatch_scores.values()) / len(file_mismatch_scores)

        return oam_score, file_mismatch_scores, evidence

    except Exception as e:
        logger.error(f"OAM calculation failed: {e}")
        evidence['error'] = str(e)
        return 0.4, {}, evidence