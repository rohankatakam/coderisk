"""
Implementation of the 7 core risk signals from docs/risk_math.md:

1. Δ-Diffusion Blast Radius (ΔDBR)
2. Hawkes-Decayed Co-Change (HDCC)
3. G² Surprise
4. Ownership Authority Mismatch (OAM)
5. Span-Core & Bridge Risk
6. Incident Adjacency (GB-RRF)
7. JIT Baselines
"""

import asyncio
import math
import numpy as np
from typing import Dict, List, Tuple, Optional, Set, Any
from dataclasses import dataclass
from datetime import datetime, timedelta
import logging

from ..models.risk_assessment import RiskSignal
from .mathematical_models import (
    calculate_delta_diffusion_blast_radius,
    calculate_hawkes_decayed_cochange,
    calculate_g2_surprise,
    calculate_ownership_authority_mismatch,
    GraphEdge
)

logger = logging.getLogger(__name__)


@dataclass
class IncidentData:
    """Represents a historical incident"""
    incident_id: str
    timestamp: datetime
    affected_files: List[str]
    description: str
    keywords: List[str]
    severity: float


@dataclass
class SpanCoreData:
    """Data for span-core analysis"""
    file_path: str
    core_persistence_score: float
    temporal_span_days: int
    centrality_score: float


class RiskSignalCalculator:
    """Base calculator for risk signals with common functionality"""

    def __init__(self, timeout_ms: int = 500):
        self.timeout_ms = timeout_ms

    async def calculate_with_timeout(self, calc_func, *args, **kwargs) -> Any:
        """Execute calculation with timeout"""
        try:
            return await asyncio.wait_for(
                asyncio.create_task(asyncio.coroutine(calc_func)(*args, **kwargs)),
                timeout=self.timeout_ms / 1000
            )
        except asyncio.TimeoutError:
            logger.warning(f"Calculation timeout after {self.timeout_ms}ms")
            return None
        except Exception as e:
            logger.error(f"Calculation failed: {e}")
            return None


class BlastRadiusSignal(RiskSignalCalculator):
    """Calculates Δ-Diffusion Blast Radius (ΔDBR) signal"""

    async def calculate(
        self,
        changed_files: List[str],
        import_edges: List[GraphEdge],
        cochange_edges: List[GraphEdge]
    ) -> RiskSignal:
        """
        Calculate ΔDBR signal.

        Args:
            changed_files: Files being modified
            import_edges: Import dependency edges
            cochange_edges: Co-change relationship edges

        Returns:
            RiskSignal with ΔDBR score
        """
        start_time = datetime.now()

        try:
            score, impact_scores, evidence = calculate_delta_diffusion_blast_radius(
                changed_files, import_edges, cochange_edges,
                max_hops=2, timeout_ms=self.timeout_ms
            )

            # Generate evidence text
            evidence_texts = [
                f"Blast radius affects {len(evidence.get('impacted_files', []))} files",
                f"Maximum impact delta: {evidence.get('max_impact_delta', 0.0):.3f}"
            ]

            if evidence.get('impacted_files'):
                top_files = evidence['impacted_files'][:3]
                evidence_texts.append(f"High-impact files: {', '.join(top_files)}")

        except Exception as e:
            logger.error(f"ΔDBR calculation failed: {e}")
            score = 0.5
            evidence_texts = [f"ΔDBR calculation failed: {str(e)}"]

        end_time = datetime.now()
        response_time = int((end_time - start_time).total_seconds() * 1000)

        return RiskSignal(
            name="delta_diffusion_blast_radius",
            score=score,
            confidence=0.85,
            evidence=evidence_texts,
            response_time_ms=response_time
        )


class CoChangeSignal(RiskSignalCalculator):
    """Calculates Hawkes-Decayed Co-Change (HDCC) signal"""

    async def calculate(
        self,
        changed_files: List[str],
        cochange_history: List[Tuple[str, str, datetime, float]]
    ) -> RiskSignal:
        """
        Calculate HDCC signal.

        Args:
            changed_files: Files being modified
            cochange_history: Historical co-change data

        Returns:
            RiskSignal with HDCC score
        """
        start_time = datetime.now()

        try:
            score, coupling_scores, evidence = calculate_hawkes_decayed_cochange(
                changed_files, cochange_history,
                analysis_window_days=90
            )

            # Generate evidence text
            evidence_texts = [
                f"Found {evidence.get('cochange_pairs_found', 0)} co-change pairs in 90-day window",
                f"Highest coupling strength: {evidence.get('highest_coupling_strength', 0.0):.3f}"
            ]

            coupled_files = evidence.get('coupled_files', [])
            if coupled_files:
                top_coupled = coupled_files[:3]
                evidence_texts.append(
                    f"Strongly coupled files: {', '.join([f'{file} ({score:.2f})' for file, score in top_coupled])}"
                )

        except Exception as e:
            logger.error(f"HDCC calculation failed: {e}")
            score = 0.3
            evidence_texts = [f"HDCC calculation failed: {str(e)}"]

        end_time = datetime.now()
        response_time = int((end_time - start_time).total_seconds() * 1000)

        return RiskSignal(
            name="hawkes_decayed_cochange",
            score=score,
            confidence=0.80,
            evidence=evidence_texts,
            response_time_ms=response_time
        )


class G2SurpriseSignal(RiskSignalCalculator):
    """Calculates G² Surprise using Dunning log-likelihood"""

    async def calculate(
        self,
        changed_files: List[str],
        file_pair_frequencies: Dict[Tuple[str, str], Tuple[int, float]]
    ) -> RiskSignal:
        """
        Calculate G² surprise signal.

        Args:
            changed_files: Files being modified
            file_pair_frequencies: (observed_freq, expected_freq) for file pairs

        Returns:
            RiskSignal with G² surprise score
        """
        start_time = datetime.now()

        try:
            # Extract file pairs involving changed files
            relevant_pairs = []
            observed_frequencies = {}
            expected_frequencies = {}

            for (file1, file2), (observed, expected) in file_pair_frequencies.items():
                if file1 in changed_files or file2 in changed_files:
                    relevant_pairs.append((file1, file2))
                    observed_frequencies[(file1, file2)] = observed
                    expected_frequencies[(file1, file2)] = expected

            score, unusual_pairs, evidence = calculate_g2_surprise(
                relevant_pairs, expected_frequencies, observed_frequencies
            )

            # Generate evidence text
            evidence_texts = [
                f"Analyzed {evidence.get('pairs_analyzed', 0)} file pairs",
                f"Found {evidence.get('unusual_pairs_found', 0)} unusual co-change patterns"
            ]

            if unusual_pairs:
                top_unusual = unusual_pairs[:2]
                evidence_texts.append(
                    f"Most unusual pairs: {', '.join([f'{f1}-{f2} (G²={g2:.2f})' for f1, f2, g2 in top_unusual])}"
                )

        except Exception as e:
            logger.error(f"G² surprise calculation failed: {e}")
            score = 0.2
            evidence_texts = [f"G² surprise calculation failed: {str(e)}"]

        end_time = datetime.now()
        response_time = int((end_time - start_time).total_seconds() * 1000)

        return RiskSignal(
            name="g2_surprise",
            score=score,
            confidence=0.70,
            evidence=evidence_texts,
            response_time_ms=response_time
        )


class OwnershipAuthorityMismatch(RiskSignalCalculator):
    """Calculates Ownership Authority Mismatch (OAM) signal"""

    async def calculate(
        self,
        changed_files: List[str],
        file_ownership_history: Dict[str, List[Tuple[str, datetime, int]]],
        team_experience: Dict[str, float]
    ) -> RiskSignal:
        """
        Calculate OAM signal.

        Args:
            changed_files: Files being modified
            file_ownership_history: File ownership history
            team_experience: Team member experience scores

        Returns:
            RiskSignal with OAM score
        """
        start_time = datetime.now()

        try:
            score, file_scores, evidence = calculate_ownership_authority_mismatch(
                changed_files, file_ownership_history, team_experience
            )

            # Generate evidence text
            evidence_texts = [
                f"Analyzed ownership for {evidence.get('files_analyzed', 0)} files"
            ]

            ownership_mismatches = evidence.get('ownership_mismatches', [])
            if ownership_mismatches:
                evidence_texts.append(f"Found {len(ownership_mismatches)} ownership mismatches")
                top_mismatch = ownership_mismatches[0]
                evidence_texts.append(
                    f"Highest mismatch: {top_mismatch['file']} (score: {top_mismatch['mismatch_score']:.2f})"
                )

        except Exception as e:
            logger.error(f"OAM calculation failed: {e}")
            score = 0.4
            evidence_texts = [f"OAM calculation failed: {str(e)}"]

        end_time = datetime.now()
        response_time = int((end_time - start_time).total_seconds() * 1000)

        return RiskSignal(
            name="ownership_authority_mismatch",
            score=score,
            confidence=0.75,
            evidence=evidence_texts,
            response_time_ms=response_time
        )


class SpanCoreSignal(RiskSignalCalculator):
    """Calculates Span-Core temporal persistence signal"""

    async def calculate(
        self,
        changed_files: List[str],
        span_core_data: List[SpanCoreData]
    ) -> RiskSignal:
        """
        Calculate span-core signal.

        Args:
            changed_files: Files being modified
            span_core_data: Span-core analysis data

        Returns:
            RiskSignal with span-core score
        """
        start_time = datetime.now()

        try:
            # Find span-core data for changed files
            relevant_data = [data for data in span_core_data if data.file_path in changed_files]

            if not relevant_data:
                score = 0.2
                evidence_texts = ["No span-core data available for changed files"]
            else:
                # Calculate weighted score based on core persistence and centrality
                weighted_scores = []
                for data in relevant_data:
                    # Core persistence: how consistently the file appears in temporal cores
                    core_weight = data.core_persistence_score

                    # Centrality: importance in dependency graph
                    centrality_weight = data.centrality_score

                    # Combined score
                    file_score = 0.6 * core_weight + 0.4 * centrality_weight
                    weighted_scores.append(file_score)

                # Overall span-core risk
                if weighted_scores:
                    score = max(weighted_scores)  # Use max as most critical
                else:
                    score = 0.2

                # Generate evidence
                evidence_texts = [
                    f"Analyzed {len(relevant_data)} files with span-core data",
                    f"Average core persistence: {np.mean([d.core_persistence_score for d in relevant_data]):.3f}",
                    f"Average centrality: {np.mean([d.centrality_score for d in relevant_data]):.3f}"
                ]

                # Highlight high-risk files
                high_risk_files = [
                    data.file_path for data, score in zip(relevant_data, weighted_scores)
                    if score > 0.7
                ]
                if high_risk_files:
                    evidence_texts.append(f"High span-core risk files: {', '.join(high_risk_files)}")

        except Exception as e:
            logger.error(f"Span-core calculation failed: {e}")
            score = 0.3
            evidence_texts = [f"Span-core calculation failed: {str(e)}"]

        end_time = datetime.now()
        response_time = int((end_time - start_time).total_seconds() * 1000)

        return RiskSignal(
            name="span_core_risk",
            score=score,
            confidence=0.65,
            evidence=evidence_texts,
            response_time_ms=response_time
        )


class BridgeRiskSignal(RiskSignalCalculator):
    """Calculates bridging centrality risk signal"""

    async def calculate(
        self,
        changed_files: List[str],
        import_graph: List[GraphEdge],
        max_hops: int = 2
    ) -> RiskSignal:
        """
        Calculate bridge risk signal.

        Args:
            changed_files: Files being modified
            import_graph: Import dependency graph
            max_hops: Maximum hops for subgraph analysis

        Returns:
            RiskSignal with bridge risk score
        """
        start_time = datetime.now()

        try:
            # Build subgraph around changed files
            relevant_nodes = set(changed_files)
            for _ in range(max_hops):
                new_nodes = set()
                for edge in import_graph:
                    if edge.source in relevant_nodes:
                        new_nodes.add(edge.target)
                    if edge.target in relevant_nodes:
                        new_nodes.add(edge.source)
                relevant_nodes.update(new_nodes)

            # Calculate betweenness centrality approximation
            bridging_scores = {}
            for file in changed_files:
                if file in relevant_nodes:
                    # Count paths that go through this file
                    paths_through_file = 0
                    total_paths = 0

                    # Simple approximation: count edges where file is intermediate
                    for edge in import_graph:
                        if edge.source in relevant_nodes and edge.target in relevant_nodes:
                            total_paths += 1
                            # Check if file could be on path (simplified)
                            if file != edge.source and file != edge.target:
                                # This is a very simplified approximation
                                paths_through_file += 0.1

                    if total_paths > 0:
                        bridging_scores[file] = paths_through_file / total_paths
                    else:
                        bridging_scores[file] = 0.0

            # Calculate overall bridge risk
            if bridging_scores:
                score = max(bridging_scores.values())
            else:
                score = 0.2

            # Generate evidence
            evidence_texts = [
                f"Analyzed bridging centrality for {len(changed_files)} files",
                f"Subgraph contains {len(relevant_nodes)} nodes"
            ]

            high_bridge_files = [
                file for file, bridge_score in bridging_scores.items()
                if bridge_score > 0.5
            ]
            if high_bridge_files:
                evidence_texts.append(f"High bridging centrality files: {', '.join(high_bridge_files)}")

        except Exception as e:
            logger.error(f"Bridge risk calculation failed: {e}")
            score = 0.3
            evidence_texts = [f"Bridge risk calculation failed: {str(e)}"]

        end_time = datetime.now()
        response_time = int((end_time - start_time).total_seconds() * 1000)

        return RiskSignal(
            name="bridge_risk",
            score=score,
            confidence=0.60,
            evidence=evidence_texts,
            response_time_ms=response_time
        )


class IncidentAdjacencySignal(RiskSignalCalculator):
    """Calculates Incident Adjacency using GB-RRF (BM25 + vector kNN + PPR fusion)"""

    async def calculate(
        self,
        changed_files: List[str],
        commit_message: Optional[str],
        historical_incidents: List[IncidentData],
        file_embeddings: Optional[Dict[str, np.ndarray]] = None
    ) -> RiskSignal:
        """
        Calculate incident adjacency signal.

        Args:
            changed_files: Files being modified
            commit_message: Commit message text
            historical_incidents: Historical incident data
            file_embeddings: File embeddings for semantic similarity

        Returns:
            RiskSignal with incident adjacency score
        """
        start_time = datetime.now()

        try:
            # BM25 scoring for textual similarity
            bm25_scores = self._calculate_bm25_scores(
                changed_files, commit_message or "", historical_incidents
            )

            # Vector similarity scoring (if embeddings available)
            vector_scores = {}
            if file_embeddings:
                vector_scores = self._calculate_vector_similarity(
                    changed_files, historical_incidents, file_embeddings
                )

            # File overlap scoring
            overlap_scores = self._calculate_file_overlap_scores(
                changed_files, historical_incidents
            )

            # Reciprocal Rank Fusion (RRF)
            fused_scores = self._reciprocal_rank_fusion(
                [bm25_scores, vector_scores, overlap_scores]
            )

            # Calculate overall score
            if fused_scores:
                score = max(fused_scores.values())
            else:
                score = 0.1

            # Generate evidence
            evidence_texts = [
                f"Analyzed {len(historical_incidents)} historical incidents",
                f"Found {len([s for s in fused_scores.values() if s > 0.3])} potential incident patterns"
            ]

            # Highlight high-risk incidents
            high_risk_incidents = [
                incident.incident_id for incident, risk_score in fused_scores.items()
                if risk_score > 0.5
            ]
            if high_risk_incidents:
                evidence_texts.append(f"Similar to incidents: {', '.join(high_risk_incidents[:3])}")

        except Exception as e:
            logger.error(f"Incident adjacency calculation failed: {e}")
            score = 0.2
            evidence_texts = [f"Incident adjacency calculation failed: {str(e)}"]

        end_time = datetime.now()
        response_time = int((end_time - start_time).total_seconds() * 1000)

        return RiskSignal(
            name="incident_adjacency",
            score=score,
            confidence=0.75,
            evidence=evidence_texts,
            response_time_ms=response_time
        )

    def _calculate_bm25_scores(
        self,
        changed_files: List[str],
        commit_message: str,
        incidents: List[IncidentData]
    ) -> Dict[IncidentData, float]:
        """Calculate BM25 scores for textual similarity"""
        scores = {}
        query_terms = set(commit_message.lower().split() + changed_files)

        for incident in incidents:
            doc_terms = set(
                incident.description.lower().split() +
                incident.keywords +
                incident.affected_files
            )

            # Simple BM25 approximation
            common_terms = query_terms.intersection(doc_terms)
            if doc_terms:
                score = len(common_terms) / len(doc_terms)
            else:
                score = 0.0

            scores[incident] = score

        return scores

    def _calculate_vector_similarity(
        self,
        changed_files: List[str],
        incidents: List[IncidentData],
        embeddings: Dict[str, np.ndarray]
    ) -> Dict[IncidentData, float]:
        """Calculate vector similarity scores"""
        scores = {}

        # Get embeddings for changed files
        changed_embeddings = [
            embeddings[file] for file in changed_files
            if file in embeddings
        ]

        if not changed_embeddings:
            return {}

        # Average embedding for changed files
        query_embedding = np.mean(changed_embeddings, axis=0)

        for incident in incidents:
            # Get embeddings for incident files
            incident_embeddings = [
                embeddings[file] for file in incident.affected_files
                if file in embeddings
            ]

            if incident_embeddings:
                incident_embedding = np.mean(incident_embeddings, axis=0)
                # Cosine similarity
                similarity = np.dot(query_embedding, incident_embedding) / (
                    np.linalg.norm(query_embedding) * np.linalg.norm(incident_embedding)
                )
                scores[incident] = max(0.0, similarity)
            else:
                scores[incident] = 0.0

        return scores

    def _calculate_file_overlap_scores(
        self,
        changed_files: List[str],
        incidents: List[IncidentData]
    ) -> Dict[IncidentData, float]:
        """Calculate file overlap scores"""
        scores = {}
        changed_set = set(changed_files)

        for incident in incidents:
            incident_set = set(incident.affected_files)
            overlap = len(changed_set.intersection(incident_set))
            union = len(changed_set.union(incident_set))

            if union > 0:
                jaccard_score = overlap / union
            else:
                jaccard_score = 0.0

            scores[incident] = jaccard_score

        return scores

    def _reciprocal_rank_fusion(
        self,
        score_lists: List[Dict[IncidentData, float]]
    ) -> Dict[IncidentData, float]:
        """Perform Reciprocal Rank Fusion on multiple score lists"""
        all_incidents = set()
        for scores in score_lists:
            all_incidents.update(scores.keys())

        fused_scores = {}
        k = 60  # RRF parameter

        for incident in all_incidents:
            rrf_score = 0.0

            for scores in score_lists:
                if scores:  # Skip empty score dictionaries
                    # Sort incidents by score (descending)
                    sorted_incidents = sorted(scores.items(), key=lambda x: x[1], reverse=True)

                    # Find rank of this incident
                    rank = next(
                        (i + 1 for i, (inc, _) in enumerate(sorted_incidents) if inc == incident),
                        len(sorted_incidents) + 1
                    )

                    # Add RRF contribution
                    rrf_score += 1.0 / (k + rank)

            fused_scores[incident] = rrf_score

        return fused_scores


class JITBaselinesSignal(RiskSignalCalculator):
    """Calculates Just-In-Time (JIT) baseline signals"""

    async def calculate(
        self,
        changed_files: List[str],
        lines_added: int,
        lines_deleted: int,
        diff_hunks: List[str]
    ) -> RiskSignal:
        """
        Calculate JIT baselines signal.

        Args:
            changed_files: Files being modified
            lines_added: Number of lines added
            lines_deleted: Number of lines deleted
            diff_hunks: List of diff hunks

        Returns:
            RiskSignal with JIT baselines score
        """
        start_time = datetime.now()

        try:
            # Size risk (touched files with auto-High cutoff)
            size_score = self._calculate_size_risk(changed_files)

            # Churn risk
            churn_score = self._calculate_churn_risk(lines_added, lines_deleted)

            # Diff entropy
            entropy_score = self._calculate_diff_entropy(diff_hunks)

            # Combined JIT score (weighted average)
            score = 0.4 * size_score + 0.3 * churn_score + 0.3 * entropy_score

            # Generate evidence
            evidence_texts = [
                f"Files changed: {len(changed_files)} (size risk: {size_score:.2f})",
                f"Lines changed: +{lines_added}/-{lines_deleted} (churn risk: {churn_score:.2f})",
                f"Diff entropy: {entropy_score:.2f}"
            ]

            # Auto-High rule for size
            if len(changed_files) >= 20:  # Threshold for auto-High
                evidence_texts.append("AUTO-HIGH: Large number of files changed")
                score = max(score, 0.9)

        except Exception as e:
            logger.error(f"JIT baselines calculation failed: {e}")
            score = 0.3
            evidence_texts = [f"JIT baselines calculation failed: {str(e)}"]

        end_time = datetime.now()
        response_time = int((end_time - start_time).total_seconds() * 1000)

        return RiskSignal(
            name="jit_baselines",
            score=score,
            confidence=0.90,
            evidence=evidence_texts,
            response_time_ms=response_time
        )

    def _calculate_size_risk(self, changed_files: List[str]) -> float:
        """Calculate risk based on number of files changed"""
        file_count = len(changed_files)

        if file_count <= 2:
            return 0.1
        elif file_count <= 5:
            return 0.3
        elif file_count <= 10:
            return 0.5
        elif file_count <= 20:
            return 0.7
        else:
            return 0.9  # Auto-High threshold

    def _calculate_churn_risk(self, lines_added: int, lines_deleted: int) -> float:
        """Calculate risk based on code churn"""
        total_churn = lines_added + lines_deleted

        if total_churn <= 10:
            return 0.1
        elif total_churn <= 50:
            return 0.3
        elif total_churn <= 200:
            return 0.5
        elif total_churn <= 500:
            return 0.7
        else:
            return 0.9

    def _calculate_diff_entropy(self, diff_hunks: List[str]) -> float:
        """Calculate entropy of diff hunks"""
        if not diff_hunks:
            return 0.1

        try:
            # Calculate character frequency distribution
            char_counts = {}
            total_chars = 0

            for hunk in diff_hunks:
                for char in hunk:
                    char_counts[char] = char_counts.get(char, 0) + 1
                    total_chars += 1

            if total_chars == 0:
                return 0.1

            # Calculate Shannon entropy
            entropy = 0.0
            for count in char_counts.values():
                probability = count / total_chars
                if probability > 0:
                    entropy -= probability * math.log2(probability)

            # Normalize to [0, 1] scale (max entropy for ASCII is ~6.6)
            normalized_entropy = min(1.0, entropy / 6.6)

            return normalized_entropy

        except Exception:
            return 0.3  # Default moderate entropy