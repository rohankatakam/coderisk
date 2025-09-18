"""
Core risk assessment engine implementing the regression scaling model
with advanced mathematical calculations
"""

import asyncio
import math
from typing import List, Dict, Optional, Any, Tuple
from datetime import datetime, timedelta
from pathlib import Path
import logging

try:
    import git
    GIT_AVAILABLE = True
except ImportError:
    GIT_AVAILABLE = False
    git = None

try:
    import numpy as np
    NUMPY_AVAILABLE = True
except ImportError:
    NUMPY_AVAILABLE = False
    # Fallback for basic array operations
    class np:
        @staticmethod
        def array(data):
            return data

from ..models.risk_assessment import (
    RiskAssessment, RiskTier, RiskSignal, ChangeContext,
    RiskEvidence, RiskRecommendation
)
from ..models.risk_signals import (
    AdvancedRiskSignal, RiskSignalSuite,
    BlastRadiusSignalData, CoChangeSignalData, JITBaselinesSignalData
)
from ..calculations import (
    RiskSignalCalculator, BlastRadiusSignal, CoChangeSignal,
    OwnershipAuthorityMismatch, SpanCoreSignal, BridgeRiskSignal,
    IncidentAdjacencySignal, JITBaselinesSignal, G2SurpriseSignal,
    RegressionScalingModel, ScoringEngine,
    GraphEdge, IncidentData, SpanCoreData
)

try:
    from .cognee_integration import CogneeCodeAnalyzer
    COGNEE_AVAILABLE = True
except ImportError:
    COGNEE_AVAILABLE = False
    from .simple_analyzer import SimpleCodeAnalyzer

logger = logging.getLogger(__name__)


class RiskEngine:
    """Main risk assessment engine with advanced mathematical calculations"""

    def __init__(self, repo_path: str, use_advanced_calculations: bool = True):
        self.repo_path = Path(repo_path).resolve()
        self.use_advanced_calculations = use_advanced_calculations

        # Initialize analyzers
        if COGNEE_AVAILABLE:
            self.analyzer = CogneeCodeAnalyzer(repo_path)
        else:
            self.analyzer = SimpleCodeAnalyzer(repo_path)
        self._repo = None

        # Initialize advanced calculation components
        if use_advanced_calculations:
            self.regression_scaling = RegressionScalingModel(str(repo_path))
            self.scoring_engine = ScoringEngine(use_conformal=True)

            # Initialize signal calculators
            self.blast_radius_calc = BlastRadiusSignal()
            self.cochange_calc = CoChangeSignal()
            self.ownership_calc = OwnershipAuthorityMismatch()
            self.span_core_calc = SpanCoreSignal()
            self.bridge_calc = BridgeRiskSignal()
            self.incident_calc = IncidentAdjacencySignal()
            self.jit_calc = JITBaselinesSignal()
            self.g2_calc = G2SurpriseSignal()

            # Cache for expensive computations
            self._graph_cache = {}
            self._cochange_cache = {}
        else:
            self.regression_scaling = None
            self.scoring_engine = None

    async def initialize(self) -> None:
        """Initialize the risk engine"""
        await self.analyzer.initialize()
        if GIT_AVAILABLE:
            try:
                self._repo = git.Repo(self.repo_path)
            except git.InvalidGitRepositoryError:
                logger.warning(f"{self.repo_path} is not a git repository")
                self._repo = None
        else:
            logger.warning("Git not available - repository features disabled")
            self._repo = None

        # Initialize regression scaling model
        if self.use_advanced_calculations and self.regression_scaling:
            self.regression_scaling.initialize()

    async def assess_worktree_risk(self) -> RiskAssessment:
        """Assess risk of uncommitted changes in the working tree"""
        if not self._repo:
            raise ValueError("Not a git repository")

        # Get git diff information
        diff_info = self._get_git_diff_info()
        change_context = ChangeContext(
            files_changed=diff_info["files"],
            lines_added=diff_info["additions"],
            lines_deleted=diff_info["deletions"],
            functions_changed=diff_info["functions"],
            timestamp=datetime.now()
        )

        return await self._assess_change_risk(change_context)

    async def assess_commit_risk(self, commit_sha: str) -> RiskAssessment:
        """Assess risk of a specific commit"""
        if not self._repo:
            raise ValueError("Not a git repository")

        commit = self._repo.commit(commit_sha)
        diff_info = self._get_commit_diff_info(commit)

        change_context = ChangeContext(
            files_changed=diff_info["files"],
            lines_added=diff_info["additions"],
            lines_deleted=diff_info["deletions"],
            functions_changed=diff_info["functions"],
            commit_message=commit.message,
            author=commit.author.name,
            timestamp=commit.committed_datetime
        )

        return await self._assess_change_risk(change_context)

    async def _assess_change_risk(self, change_context: ChangeContext) -> RiskAssessment:
        """Core risk assessment logic with advanced mathematical calculations"""
        start_time = datetime.now()

        if self.use_advanced_calculations:
            return await self._assess_change_risk_advanced(change_context, start_time)
        else:
            return await self._assess_change_risk_basic(change_context, start_time)

    async def _assess_change_risk_advanced(self, change_context: ChangeContext, start_time: datetime) -> RiskAssessment:
        """Advanced risk assessment using mathematical models"""
        # Calculate advanced risk signals
        signal_suite = await self._calculate_advanced_signals(change_context)

        # Get all signals for compatibility
        all_signals = signal_suite.get_all_signals()
        signals = [RiskSignal(s.name, s.score, s.confidence, s.evidence, s.response_time_ms) for s in all_signals]

        # Calculate regression scaling factors
        scaling_factors = self.regression_scaling.calculate_scaling_factors(
            change_context.files_changed,
            change_context.commit_message
        )

        # Prepare metadata for scoring engine
        metadata = {
            'files_changed': len(change_context.files_changed),
            'lines_added': change_context.lines_added,
            'lines_deleted': change_context.lines_deleted,
            'segment': self._determine_size_segment(change_context),
            'has_secret': False,  # TODO: Get from security signal
            'has_critical_sink': False,  # TODO: Get from security signal
            'importer_count': 0,  # TODO: Get from API signal
            'importer_p95': 10  # TODO: Calculate from repository data
        }

        # Use scoring engine if available
        if self.scoring_engine:
            try:
                # Create feature vector for conformal prediction
                feature_vector = self._create_feature_vector(all_signals, scaling_factors)

                # Score the change
                tier, final_score, scoring_details = self.scoring_engine.score_change(
                    all_signals, metadata, feature_vector
                )
            except Exception as e:
                logger.warning(f"Advanced scoring failed, falling back to basic: {e}")
                tier, final_score, scoring_details = self._basic_scoring_fallback(all_signals)
        else:
            tier, final_score, scoring_details = self._basic_scoring_fallback(all_signals)

        # Apply regression scaling to base score
        base_score = scoring_details.get('base_score', final_score)
        scaled_score = min(100.0, base_score * 100 * scaling_factors.total_multiplier)

        # Generate evidence and recommendations
        evidence = self._generate_evidence_advanced(all_signals, change_context, scaling_factors)
        recommendations = self._generate_recommendations_advanced(all_signals, tier, scaling_factors)

        # Create risk categories breakdown
        categories = self._create_categories_breakdown(signal_suite)

        end_time = datetime.now()
        assessment_time_ms = int((end_time - start_time).total_seconds() * 1000)

        return RiskAssessment(
            tier=tier,
            score=scaled_score,
            confidence=self._calculate_confidence_advanced(all_signals),
            signals=signals,
            categories=categories,
            change_context=change_context,
            evidence=evidence,
            recommendations=recommendations,
            team_factor=scaling_factors.team_factor,
            codebase_factor=scaling_factors.codebase_factor,
            change_velocity=scaling_factors.change_velocity,
            migration_multiplier=scaling_factors.migration_multiplier,
            assessment_time_ms=assessment_time_ms,
            created_at=datetime.now()
        )

    async def _assess_change_risk_basic(self, change_context: ChangeContext, start_time: datetime) -> RiskAssessment:
        """Basic risk assessment (original logic)"""
        # Initialize risk signals
        signals = []

        # Core risk signals (parallel execution)
        tasks = [
            self._calculate_blast_radius(change_context),
            self._analyze_cochange_risk(change_context),
            self._detect_incident_adjacency(change_context),
            self._assess_api_breaking_changes(change_context),
            self._detect_performance_risk(change_context),
            self._analyze_test_coverage_gap(change_context),
            self._detect_security_patterns(change_context)
        ]

        signal_results = await asyncio.gather(*tasks, return_exceptions=True)

        # Process signal results
        signal_names = [
            "blast_radius", "cochange_risk", "incident_adjacency",
            "api_breaking", "performance_risk", "test_coverage", "security_risk"
        ]

        for i, result in enumerate(signal_results):
            if isinstance(result, Exception):
                print(f"Warning: Signal {signal_names[i]} failed: {result}")
                continue
            if result:
                signals.append(result)

        # Calculate regression scaling factors
        team_factor = self._calculate_team_factor()
        codebase_factor = self._calculate_codebase_factor()
        change_velocity = self._calculate_change_velocity()
        migration_multiplier = self._calculate_migration_multiplier(change_context)

        # Calculate base risk score
        base_score = self._calculate_base_risk_score(signals)

        # Apply regression scaling formula
        scaled_score = min(100.0, base_score * team_factor * codebase_factor * change_velocity * migration_multiplier)

        # Determine risk tier
        tier = self._determine_risk_tier(scaled_score)

        # Generate evidence and recommendations
        evidence = self._generate_evidence(signals, change_context)
        recommendations = self._generate_recommendations(signals, tier)

        # Create risk categories breakdown
        categories = {
            "blast_radius": next((s.score for s in signals if s.name == "blast_radius"), 0.0),
            "cochange_risk": next((s.score for s in signals if s.name == "cochange_risk"), 0.0),
            "incident_adjacency": next((s.score for s in signals if s.name == "incident_adjacency"), 0.0),
            "api_breaking": next((s.score for s in signals if s.name == "api_breaking"), 0.0),
            "performance_risk": next((s.score for s in signals if s.name == "performance_risk"), 0.0),
            "test_coverage": next((s.score for s in signals if s.name == "test_coverage"), 0.0),
            "security_risk": next((s.score for s in signals if s.name == "security_risk"), 0.0)
        }

        end_time = datetime.now()
        assessment_time_ms = int((end_time - start_time).total_seconds() * 1000)

        return RiskAssessment(
            tier=tier,
            score=scaled_score,
            confidence=self._calculate_confidence(signals),
            signals=signals,
            categories=categories,
            change_context=change_context,
            evidence=evidence,
            recommendations=recommendations,
            team_factor=team_factor,
            codebase_factor=codebase_factor,
            change_velocity=change_velocity,
            migration_multiplier=migration_multiplier,
            assessment_time_ms=assessment_time_ms,
            created_at=datetime.now()
        )

    def _get_git_diff_info(self) -> Dict[str, Any]:
        """Extract information from git diff (uncommitted changes)"""
        if not self._repo:
            return {"files": [], "additions": 0, "deletions": 0, "functions": []}

        try:
            # Get staged and unstaged changes
            staged_diff = self._repo.index.diff("HEAD")
            unstaged_diff = self._repo.index.diff(None)

            files = set()
            additions = 0
            deletions = 0

            for diff in staged_diff + unstaged_diff:
                if diff.a_path:
                    files.add(diff.a_path)
                if diff.b_path:
                    files.add(diff.b_path)

                # Try to get line counts (may not always be available)
                try:
                    if hasattr(diff, 'change_type') and diff.change_type != 'D':
                        # Rough estimation - in real implementation, parse diff properly
                        additions += 10  # Placeholder
                        deletions += 5   # Placeholder
                except:
                    pass

            return {
                "files": list(files),
                "additions": additions,
                "deletions": deletions,
                "functions": []  # Would need AST parsing to extract function names
            }
        except Exception as e:
            print(f"Warning: Could not get git diff info: {e}")
            return {"files": [], "additions": 0, "deletions": 0, "functions": []}

    def _get_commit_diff_info(self, commit) -> Dict[str, Any]:
        """Extract information from a specific commit"""
        try:
            files = list(commit.stats.files.keys())
            total_stats = commit.stats.total
            additions = total_stats['insertions']
            deletions = total_stats['deletions']

            return {
                "files": files,
                "additions": additions,
                "deletions": deletions,
                "functions": []  # Would need AST parsing
            }
        except Exception as e:
            print(f"Warning: Could not get commit diff info: {e}")
            return {"files": [], "additions": 0, "deletions": 0, "functions": []}

    async def _calculate_blast_radius(self, change_context: ChangeContext) -> RiskSignal:
        """Calculate blast radius using dependency analysis"""
        start_time = datetime.now()

        try:
            # Get impact radius from Cognee
            impact_map = await self.analyzer.get_file_impact_radius(change_context.files_changed)

            # Calculate total impacted files
            all_impacted = set()
            for file_path, impacted_files in impact_map.items():
                all_impacted.update(impacted_files)
                all_impacted.add(file_path)

            # Risk score based on number of impacted files
            impact_count = len(all_impacted)
            if impact_count <= 3:
                score = 0.1
            elif impact_count <= 10:
                score = 0.3
            elif impact_count <= 25:
                score = 0.6
            else:
                score = 0.9

            evidence = [f"Change impacts {impact_count} files directly or indirectly"]
            if impact_count > 10:
                evidence.append(f"High-impact files: {list(all_impacted)[:5]}...")

        except Exception as e:
            print(f"Warning: Blast radius calculation failed: {e}")
            score = 0.5  # Default moderate risk
            evidence = ["Could not calculate blast radius - using default risk"]

        end_time = datetime.now()
        response_time = int((end_time - start_time).total_seconds() * 1000)

        return RiskSignal(
            name="blast_radius",
            score=score,
            confidence=0.8,
            evidence=evidence,
            response_time_ms=response_time
        )

    async def _analyze_cochange_risk(self, change_context: ChangeContext) -> RiskSignal:
        """Analyze co-change patterns using historical data"""
        start_time = datetime.now()

        try:
            # Find similar historical changes
            change_desc = change_context.commit_message or "File changes"
            similar_changes = await self.analyzer.find_similar_changes(
                change_context.files_changed, change_desc
            )

            # Simple heuristic: more similar changes = higher risk
            similarity_count = len(similar_changes)
            if similarity_count == 0:
                score = 0.2  # Unknown pattern
            elif similarity_count <= 3:
                score = 0.3  # Some precedent
            elif similarity_count <= 10:
                score = 0.6  # Common pattern
            else:
                score = 0.8  # Very common, potentially risky

            evidence = [f"Found {similarity_count} similar change patterns in history"]

        except Exception as e:
            print(f"Warning: Co-change analysis failed: {e}")
            score = 0.3
            evidence = ["Could not analyze co-change patterns"]

        end_time = datetime.now()
        response_time = int((end_time - start_time).total_seconds() * 1000)

        return RiskSignal(
            name="cochange_risk",
            score=score,
            confidence=0.6,
            evidence=evidence,
            response_time_ms=response_time
        )

    async def _detect_incident_adjacency(self, change_context: ChangeContext) -> RiskSignal:
        """Detect proximity to past incidents"""
        start_time = datetime.now()

        try:
            incident_patterns = await self.analyzer.detect_incident_patterns({
                "files": change_context.files_changed,
                "message": change_context.commit_message or "",
                "functions": change_context.functions_changed
            })

            incident_count = len(incident_patterns)
            if incident_count == 0:
                score = 0.1
            elif incident_count <= 2:
                score = 0.4
            elif incident_count <= 5:
                score = 0.7
            else:
                score = 0.9

            evidence = [f"Found {incident_count} incident-related patterns"]

        except Exception as e:
            print(f"Warning: Incident adjacency detection failed: {e}")
            score = 0.2
            evidence = ["Could not detect incident patterns"]

        end_time = datetime.now()
        response_time = int((end_time - start_time).total_seconds() * 1000)

        return RiskSignal(
            name="incident_adjacency",
            score=score,
            confidence=0.7,
            evidence=evidence,
            response_time_ms=response_time
        )

    async def _assess_api_breaking_changes(self, change_context: ChangeContext) -> RiskSignal:
        """Detect potential API breaking changes"""
        start_time = datetime.now()

        try:
            api_analysis = await self.analyzer.analyze_api_surface(change_context.files_changed)

            # Simple heuristics for breaking changes
            score = 0.0
            evidence = []

            # Check for files with many importers
            high_impact_files = []
            for file_path, importers in api_analysis.get("importers", {}).items():
                if len(importers) > 5:
                    high_impact_files.append(file_path)
                    score = max(score, 0.7)

            if high_impact_files:
                evidence.append(f"Modified files with many importers: {high_impact_files}")

            # Check for specific file patterns that often contain APIs
            api_files = [f for f in change_context.files_changed
                        if any(pattern in f.lower() for pattern in ['api', 'interface', 'service', 'client'])]
            if api_files:
                score = max(score, 0.6)
                evidence.append(f"Modified API-related files: {api_files}")

        except Exception as e:
            print(f"Warning: API breaking change detection failed: {e}")
            score = 0.3
            evidence = ["Could not analyze API surface"]

        end_time = datetime.now()
        response_time = int((end_time - start_time).total_seconds() * 1000)

        return RiskSignal(
            name="api_breaking",
            score=score,
            confidence=0.6,
            evidence=evidence,
            response_time_ms=response_time
        )

    async def _detect_performance_risk(self, change_context: ChangeContext) -> RiskSignal:
        """Detect potential performance issues"""
        # Simple heuristics based on change patterns
        score = 0.0
        evidence = []

        # Large file changes
        if change_context.lines_added > 500:
            score = max(score, 0.4)
            evidence.append(f"Large change: {change_context.lines_added} lines added")

        # Many files changed
        if len(change_context.files_changed) > 20:
            score = max(score, 0.5)
            evidence.append(f"Wide change: {len(change_context.files_changed)} files modified")

        # Performance-sensitive file patterns
        perf_files = [f for f in change_context.files_changed
                     if any(pattern in f.lower() for pattern in ['loop', 'query', 'db', 'cache', 'process'])]
        if perf_files:
            score = max(score, 0.6)
            evidence.append(f"Performance-sensitive files: {perf_files}")

        return RiskSignal(
            name="performance_risk",
            score=score,
            confidence=0.5,
            evidence=evidence or ["No performance risk indicators detected"],
            response_time_ms=50
        )

    async def _analyze_test_coverage_gap(self, change_context: ChangeContext) -> RiskSignal:
        """Analyze test coverage gaps"""
        # Simple heuristics
        score = 0.0
        evidence = []

        # Check if test files are being modified alongside code
        test_files = [f for f in change_context.files_changed
                     if any(pattern in f.lower() for pattern in ['test', 'spec', '__test__'])]
        code_files = [f for f in change_context.files_changed if f not in test_files]

        if code_files and not test_files:
            score = 0.8
            evidence.append("Code changes without corresponding test updates")
        elif len(test_files) < len(code_files) * 0.3:
            score = 0.6
            evidence.append("Insufficient test coverage for code changes")
        else:
            score = 0.2
            evidence.append("Good test coverage ratio")

        return RiskSignal(
            name="test_coverage",
            score=score,
            confidence=0.7,
            evidence=evidence,
            response_time_ms=30
        )

    async def _detect_security_patterns(self, change_context: ChangeContext) -> RiskSignal:
        """Detect security-related risk patterns"""
        score = 0.0
        evidence = []

        # Security-sensitive file patterns
        security_files = [f for f in change_context.files_changed
                         if any(pattern in f.lower() for pattern in [
                             'auth', 'login', 'password', 'token', 'security', 'crypto',
                             'permission', 'role', 'admin', 'config'
                         ])]

        if security_files:
            score = 0.7
            evidence.append(f"Security-sensitive files modified: {security_files}")

        # Check commit message for security keywords
        if change_context.commit_message:
            security_keywords = ['fix', 'security', 'vulnerability', 'patch', 'cve']
            if any(keyword in change_context.commit_message.lower() for keyword in security_keywords):
                score = max(score, 0.5)
                evidence.append("Security-related keywords in commit message")

        return RiskSignal(
            name="security_risk",
            score=score,
            confidence=0.6,
            evidence=evidence or ["No security risk indicators detected"],
            response_time_ms=40
        )

    def _calculate_team_factor(self) -> float:
        """Calculate team scaling factor (simplified for MVP)"""
        # For MVP, assume small team (would integrate with git log analysis later)
        team_size = 4  # Default assumption
        return 1 + 0.3 * math.log2(team_size)

    def _calculate_codebase_factor(self) -> float:
        """Calculate codebase scaling factor"""
        try:
            # Count lines of code (simplified)
            total_loc = 0
            for file_path in self.repo_path.rglob("*.py"):
                try:
                    total_loc += sum(1 for line in file_path.read_text().splitlines() if line.strip())
                except:
                    pass

            # Simple coupling coefficient (would be more sophisticated in full version)
            coupling_coefficient = 0.6  # Assume moderate coupling

            return 1 + 0.2 * (total_loc / 100000) * coupling_coefficient
        except:
            return 1.2  # Default moderate factor

    def _calculate_change_velocity(self) -> float:
        """Calculate change velocity factor"""
        if not self._repo:
            return 1.1  # Default

        try:
            # Count commits in last week (simplified)
            commits_last_week = len(list(self._repo.iter_commits('--since="1 week ago"')))
            return 1 + (commits_last_week / 100)
        except:
            return 1.1

    def _calculate_migration_multiplier(self, change_context: ChangeContext) -> float:
        """Calculate migration risk multiplier"""
        multiplier = 1.0

        # Check for migration indicators
        migration_indicators = [
            'migration', 'upgrade', 'version', 'framework', 'dependency',
            'refactor', 'modernize', 'rewrite'
        ]

        commit_msg = (change_context.commit_message or "").lower()
        if any(indicator in commit_msg for indicator in migration_indicators):
            multiplier = 2.0  # Basic migration multiplier

        # Check for package.json, requirements.txt changes
        config_files = [f for f in change_context.files_changed
                       if any(name in f.lower() for name in [
                           'package.json', 'requirements.txt', 'pom.xml', 'cargo.toml',
                           'go.mod', 'gemfile'
                       ])]

        if config_files:
            multiplier = max(multiplier, 1.5)

        return multiplier

    def _calculate_base_risk_score(self, signals: List[RiskSignal]) -> float:
        """Calculate base risk score from signals"""
        if not signals:
            return 20.0  # Default low risk

        # Weighted average of signals
        total_weighted_score = 0.0
        total_weight = 0.0

        signal_weights = {
            "blast_radius": 0.25,
            "cochange_risk": 0.15,
            "incident_adjacency": 0.20,
            "api_breaking": 0.20,
            "performance_risk": 0.05,
            "test_coverage": 0.10,
            "security_risk": 0.05
        }

        for signal in signals:
            weight = signal_weights.get(signal.name, 0.1)
            weighted_score = signal.score * signal.confidence * weight * 100
            total_weighted_score += weighted_score
            total_weight += weight

        if total_weight == 0:
            return 20.0

        return total_weighted_score / total_weight

    def _determine_risk_tier(self, score: float) -> RiskTier:
        """Determine risk tier from score"""
        if score < 25:
            return RiskTier.LOW
        elif score < 50:
            return RiskTier.MEDIUM
        elif score < 75:
            return RiskTier.HIGH
        else:
            return RiskTier.CRITICAL

    def _calculate_confidence(self, signals: List[RiskSignal]) -> float:
        """Calculate overall confidence in assessment"""
        if not signals:
            return 0.5

        return sum(s.confidence for s in signals) / len(signals)

    def _generate_evidence(self, signals: List[RiskSignal], change_context: ChangeContext) -> List[RiskEvidence]:
        """Generate evidence for the assessment"""
        evidence = []

        for signal in signals:
            for evidence_text in signal.evidence:
                evidence.append(RiskEvidence(
                    type="signal_evidence",
                    description=evidence_text,
                    confidence=signal.confidence
                ))

        return evidence

    def _generate_recommendations(self, signals: List[RiskSignal], tier: RiskTier) -> List[RiskRecommendation]:
        """Generate actionable recommendations"""
        recommendations = []

        if tier in [RiskTier.HIGH, RiskTier.CRITICAL]:
            recommendations.append(RiskRecommendation(
                action="Add additional reviewer",
                priority="high",
                description="Given the high risk score, have a senior team member review this change"
            ))

        # Signal-specific recommendations
        for signal in signals:
            if signal.name == "test_coverage" and signal.score > 0.6:
                recommendations.append(RiskRecommendation(
                    action="Add tests",
                    priority="medium",
                    description="Improve test coverage for the modified code"
                ))

            if signal.name == "api_breaking" and signal.score > 0.6:
                recommendations.append(RiskRecommendation(
                    action="API compatibility check",
                    priority="high",
                    description="Verify backward compatibility of API changes"
                ))

        return recommendations

    # Advanced calculation methods

    async def _calculate_advanced_signals(self, change_context: ChangeContext) -> RiskSignalSuite:
        """Calculate all advanced risk signals"""
        suite = RiskSignalSuite()

        try:
            # Prepare graph data
            import_edges, cochange_edges = await self._get_graph_data(change_context.files_changed)
            cochange_history = await self._get_cochange_history(change_context.files_changed)

            # Parallel calculation of mathematical signals
            math_tasks = [
                self._calculate_blast_radius_advanced(change_context, import_edges, cochange_edges),
                self._calculate_cochange_advanced(change_context, cochange_history),
                self._calculate_ownership_advanced(change_context),
                self._calculate_jit_baselines_advanced(change_context)
            ]

            # Execute mathematical signals in parallel
            math_results = await asyncio.gather(*math_tasks, return_exceptions=True)

            # Process results
            signal_names = ['blast_radius', 'cochange', 'ownership', 'jit_baselines']
            for i, result in enumerate(math_results):
                if isinstance(result, Exception):
                    logger.error(f"Advanced signal {signal_names[i]} failed: {result}")
                    continue

                if signal_names[i] == 'blast_radius':
                    suite.blast_radius = result
                elif signal_names[i] == 'cochange':
                    suite.cochange = result
                elif signal_names[i] == 'ownership':
                    suite.ownership = result
                elif signal_names[i] == 'jit_baselines':
                    suite.jit_baselines = result

            # Add simpler signals that don't require complex graph data
            suite.span_core = await self._calculate_span_core_advanced(change_context)
            suite.bridge_risk = await self._calculate_bridge_risk_advanced(change_context, import_edges)
            suite.incident_adjacency = await self._calculate_incident_adjacency_advanced(change_context)
            suite.g2_surprise = await self._calculate_g2_surprise_advanced(change_context)

        except Exception as e:
            logger.error(f"Advanced signal calculation failed: {e}")

        return suite

    async def _calculate_blast_radius_advanced(
        self,
        change_context: ChangeContext,
        import_edges: List[GraphEdge],
        cochange_edges: List[GraphEdge]
    ) -> AdvancedRiskSignal:
        """Calculate advanced blast radius using ΔDBR"""
        signal = await self.blast_radius_calc.calculate(
            change_context.files_changed,
            import_edges,
            cochange_edges
        )

        # Create detailed data
        detailed_data = BlastRadiusSignalData(
            delta_dbr_score=signal.score,
            impacted_files=[],  # Will be populated by calculation
            impact_scores={},
            max_impact_delta=0.0,
            graph_nodes_analyzed=len(set(e.source for e in import_edges + cochange_edges)),
            calculation_time_ms=signal.response_time_ms
        )

        return AdvancedRiskSignal(
            signal.name, signal.score, signal.confidence,
            signal.evidence, signal.response_time_ms,
            detailed_data=detailed_data
        )

    async def _calculate_cochange_advanced(
        self,
        change_context: ChangeContext,
        cochange_history: List[Tuple[str, str, datetime, float]]
    ) -> AdvancedRiskSignal:
        """Calculate advanced co-change using HDCC"""
        signal = await self.cochange_calc.calculate(
            change_context.files_changed,
            cochange_history
        )

        # Create detailed data
        detailed_data = CoChangeSignalData(
            hdcc_score=signal.score,
            coupled_files=[],  # Will be populated by calculation
            analysis_window_days=90,
            cochange_pairs_found=len(cochange_history),
            hawkes_decay_params=(0.1, 0.01)
        )

        return AdvancedRiskSignal(
            signal.name, signal.score, signal.confidence,
            signal.evidence, signal.response_time_ms,
            detailed_data=detailed_data
        )

    async def _calculate_ownership_advanced(self, change_context: ChangeContext) -> AdvancedRiskSignal:
        """Calculate ownership authority mismatch"""
        # Get ownership data (simplified for MVP)
        ownership_history = await self._get_ownership_history(change_context.files_changed)
        team_experience = await self._get_team_experience()

        signal = await self.ownership_calc.calculate(
            change_context.files_changed,
            ownership_history,
            team_experience
        )

        return AdvancedRiskSignal(
            signal.name, signal.score, signal.confidence,
            signal.evidence, signal.response_time_ms
        )

    async def _calculate_jit_baselines_advanced(self, change_context: ChangeContext) -> AdvancedRiskSignal:
        """Calculate JIT baselines signal"""
        # Get diff hunks (simplified)
        diff_hunks = await self._get_diff_hunks(change_context)

        signal = await self.jit_calc.calculate(
            change_context.files_changed,
            change_context.lines_added,
            change_context.lines_deleted,
            diff_hunks
        )

        # Create detailed data
        detailed_data = JITBaselinesSignalData(
            jit_score=signal.score,
            size_risk=0.0,  # Will be populated by calculation
            churn_risk=0.0,
            entropy_risk=0.0,
            auto_high_triggered=len(change_context.files_changed) >= 20,
            diff_complexity_metrics={}
        )

        return AdvancedRiskSignal(
            signal.name, signal.score, signal.confidence,
            signal.evidence, signal.response_time_ms,
            detailed_data=detailed_data
        )

    async def _calculate_span_core_advanced(self, change_context: ChangeContext) -> AdvancedRiskSignal:
        """Calculate span-core temporal persistence"""
        # Get span-core data (simplified)
        span_core_data = await self._get_span_core_data(change_context.files_changed)

        signal = await self.span_core_calc.calculate(
            change_context.files_changed,
            span_core_data
        )

        return AdvancedRiskSignal(
            signal.name, signal.score, signal.confidence,
            signal.evidence, signal.response_time_ms
        )

    async def _calculate_bridge_risk_advanced(
        self,
        change_context: ChangeContext,
        import_edges: List[GraphEdge]
    ) -> AdvancedRiskSignal:
        """Calculate bridging centrality risk"""
        signal = await self.bridge_calc.calculate(
            change_context.files_changed,
            import_edges
        )

        return AdvancedRiskSignal(
            signal.name, signal.score, signal.confidence,
            signal.evidence, signal.response_time_ms
        )

    async def _calculate_incident_adjacency_advanced(self, change_context: ChangeContext) -> AdvancedRiskSignal:
        """Calculate incident adjacency using GB-RRF"""
        # Get historical incidents (simplified)
        incidents = await self._get_historical_incidents()

        signal = await self.incident_calc.calculate(
            change_context.files_changed,
            change_context.commit_message,
            incidents
        )

        return AdvancedRiskSignal(
            signal.name, signal.score, signal.confidence,
            signal.evidence, signal.response_time_ms
        )

    async def _calculate_g2_surprise_advanced(self, change_context: ChangeContext) -> AdvancedRiskSignal:
        """Calculate G² surprise"""
        # Get file pair frequencies (simplified)
        file_pairs = await self._get_file_pair_frequencies(change_context.files_changed)

        signal = await self.g2_calc.calculate(
            change_context.files_changed,
            file_pairs
        )

        return AdvancedRiskSignal(
            signal.name, signal.score, signal.confidence,
            signal.evidence, signal.response_time_ms
        )

    # Helper methods for advanced calculations

    async def _get_graph_data(self, changed_files: List[str]) -> Tuple[List[GraphEdge], List[GraphEdge]]:
        """Get import and co-change graph data"""
        # This would integrate with Cognee or use cached data
        # For now, return mock data for testing
        import_edges = []
        cochange_edges = []

        try:
            if hasattr(self.analyzer, 'get_import_graph'):
                import_data = await self.analyzer.get_import_graph(changed_files)
                import_edges = [
                    GraphEdge(src, dst, weight, 'import')
                    for src, dst, weight in import_data
                ]

            if hasattr(self.analyzer, 'get_cochange_graph'):
                cochange_data = await self.analyzer.get_cochange_graph(changed_files)
                cochange_edges = [
                    GraphEdge(src, dst, weight, 'cochange')
                    for src, dst, weight in cochange_data
                ]
        except Exception as e:
            logger.warning(f"Failed to get graph data: {e}")

        return import_edges, cochange_edges

    async def _get_cochange_history(self, changed_files: List[str]) -> List[Tuple[str, str, datetime, float]]:
        """Get co-change history data"""
        # This would query historical co-change patterns
        # For now, return mock data
        history = []

        try:
            if hasattr(self.analyzer, 'get_cochange_history'):
                history = await self.analyzer.get_cochange_history(changed_files)
        except Exception as e:
            logger.warning(f"Failed to get co-change history: {e}")

        return history

    async def _get_ownership_history(self, changed_files: List[str]) -> Dict[str, List[Tuple[str, datetime, int]]]:
        """Get file ownership history"""
        ownership = {}

        try:
            if self._repo:
                for file_path in changed_files:
                    file_history = []
                    try:
                        # Get commits for this file
                        commits = list(self._repo.iter_commits(paths=file_path, max_count=50))
                        for commit in commits:
                            author = commit.author.name
                            timestamp = commit.committed_datetime
                            # Simplified: assume 50 lines changed per commit
                            lines_changed = 50
                            file_history.append((author, timestamp, lines_changed))
                    except Exception:
                        pass
                    ownership[file_path] = file_history
        except Exception as e:
            logger.warning(f"Failed to get ownership history: {e}")

        return ownership

    async def _get_team_experience(self) -> Dict[str, float]:
        """Get team member experience scores"""
        experience = {}

        try:
            if self._repo:
                # Get all contributors
                contributors = set()
                for commit in self._repo.iter_commits(max_count=1000):
                    contributors.add(commit.author.name)

                # Simple experience score based on commit count
                for contributor in contributors:
                    commit_count = len(list(self._repo.iter_commits(author=contributor, max_count=100)))
                    experience[contributor] = min(1.0, commit_count / 100.0)
        except Exception as e:
            logger.warning(f"Failed to get team experience: {e}")

        return experience

    async def _get_diff_hunks(self, change_context: ChangeContext) -> List[str]:
        """Get diff hunks for entropy calculation"""
        hunks = []

        try:
            if self._repo:
                # Get diff
                diff = self._repo.git.diff('--cached', '--unified=3')
                if not diff:
                    diff = self._repo.git.diff('HEAD', '--unified=3')

                # Split into hunks (simplified)
                hunks = diff.split('@@')[1::2]  # Every other element after @@
        except Exception as e:
            logger.warning(f"Failed to get diff hunks: {e}")

        return hunks

    async def _get_span_core_data(self, changed_files: List[str]) -> List[SpanCoreData]:
        """Get span-core data for files"""
        data = []

        # Mock data for testing
        for file_path in changed_files:
            data.append(SpanCoreData(
                file_path=file_path,
                core_persistence_score=0.5,
                temporal_span_days=30,
                centrality_score=0.3
            ))

        return data

    async def _get_historical_incidents(self) -> List[IncidentData]:
        """Get historical incident data"""
        incidents = []

        try:
            if hasattr(self.analyzer, 'get_incidents'):
                incidents = await self.analyzer.get_incidents()
        except Exception as e:
            logger.warning(f"Failed to get incidents: {e}")

        return incidents

    async def _get_file_pair_frequencies(self, changed_files: List[str]) -> Dict[Tuple[str, str], Tuple[int, float]]:
        """Get file pair co-change frequencies"""
        frequencies = {}

        # Mock data for testing
        for i, file1 in enumerate(changed_files):
            for file2 in changed_files[i+1:]:
                observed = 5  # Mock observed frequency
                expected = 2.0  # Mock expected frequency
                frequencies[(file1, file2)] = (observed, expected)

        return frequencies

    def _create_feature_vector(self, signals: List[AdvancedRiskSignal], scaling_factors) -> np.ndarray:
        """Create feature vector for conformal prediction"""
        features = []

        # Signal scores
        signal_names = [
            'delta_diffusion_blast_radius', 'hawkes_decayed_cochange', 'g2_surprise',
            'ownership_authority_mismatch', 'span_core_risk', 'bridge_risk',
            'incident_adjacency', 'jit_baselines'
        ]

        signal_dict = {s.name: s for s in signals}
        for name in signal_names:
            if name in signal_dict:
                features.append(signal_dict[name].score)
            else:
                features.append(0.0)

        # Scaling factors
        features.extend([
            scaling_factors.team_factor,
            scaling_factors.codebase_factor,
            scaling_factors.change_velocity,
            scaling_factors.migration_multiplier
        ])

        return np.array(features)

    def _basic_scoring_fallback(self, signals: List[AdvancedRiskSignal]) -> Tuple[RiskTier, float, Dict[str, Any]]:
        """Fallback scoring when advanced scoring fails"""
        if not signals:
            return RiskTier.LOW, 0.2, {'base_score': 0.2}

        # Simple weighted average
        total_score = 0.0
        total_weight = 0.0

        for signal in signals:
            weight = 1.0  # Equal weights for fallback
            total_score += signal.score * signal.confidence * weight
            total_weight += weight

        if total_weight > 0:
            base_score = total_score / total_weight
        else:
            base_score = 0.2

        # Determine tier
        if base_score < 0.25:
            tier = RiskTier.LOW
        elif base_score < 0.50:
            tier = RiskTier.MEDIUM
        elif base_score < 0.75:
            tier = RiskTier.HIGH
        else:
            tier = RiskTier.CRITICAL

        return tier, base_score, {'base_score': base_score}

    def _determine_size_segment(self, change_context: ChangeContext) -> int:
        """Determine size segment for conformal prediction"""
        file_count = len(change_context.files_changed)

        if file_count <= 2:
            return 0  # Small
        elif file_count <= 10:
            return 1  # Medium
        elif file_count <= 20:
            return 2  # Large
        else:
            return 3  # Very large

    def _generate_evidence_advanced(
        self,
        signals: List[AdvancedRiskSignal],
        change_context: ChangeContext,
        scaling_factors
    ) -> List[RiskEvidence]:
        """Generate evidence for advanced assessment"""
        evidence = []

        # Add signal evidence
        for signal in signals:
            for evidence_text in signal.evidence:
                evidence.append(RiskEvidence(
                    type="mathematical_signal",
                    description=evidence_text,
                    confidence=signal.confidence
                ))

        # Add scaling factor evidence
        evidence.append(RiskEvidence(
            type="regression_scaling",
            description=f"Team factor: {scaling_factors.team_factor:.2f}, "
                       f"Codebase factor: {scaling_factors.codebase_factor:.2f}, "
                       f"Velocity factor: {scaling_factors.change_velocity:.2f}, "
                       f"Migration multiplier: {scaling_factors.migration_multiplier:.2f}",
            confidence=0.9
        ))

        return evidence

    def _generate_recommendations_advanced(
        self,
        signals: List[AdvancedRiskSignal],
        tier: RiskTier,
        scaling_factors
    ) -> List[RiskRecommendation]:
        """Generate recommendations for advanced assessment"""
        recommendations = []

        # High-risk recommendations
        if tier in [RiskTier.HIGH, RiskTier.CRITICAL]:
            recommendations.append(RiskRecommendation(
                action="Require additional reviewer",
                priority="high",
                description="High risk detected by mathematical models - require senior review"
            ))

        # Migration-specific recommendations
        if scaling_factors.migration_multiplier > 2.0:
            recommendations.append(RiskRecommendation(
                action="Migration risk mitigation",
                priority="high",
                description="Migration detected - consider phased rollout and additional testing"
            ))

        # Signal-specific recommendations
        signal_dict = {s.name: s for s in signals}

        if 'delta_diffusion_blast_radius' in signal_dict and signal_dict['delta_diffusion_blast_radius'].score > 0.7:
            recommendations.append(RiskRecommendation(
                action="Impact analysis",
                priority="medium",
                description="High blast radius - analyze downstream impact and notify affected teams"
            ))

        if 'jit_baselines' in signal_dict and signal_dict['jit_baselines'].score > 0.8:
            recommendations.append(RiskRecommendation(
                action="Test coverage review",
                priority="medium",
                description="Large change detected - ensure comprehensive test coverage"
            ))

        return recommendations

    def _create_categories_breakdown(self, signal_suite: RiskSignalSuite) -> Dict[str, float]:
        """Create categories breakdown from signal suite"""
        categories = {
            "regression": 0.0,
            "blast_radius": 0.0,
            "cochange": 0.0,
            "ownership": 0.0,
            "incident_adjacency": 0.0,
            "api": 0.0,
            "security": 0.0,
            "performance": 0.0,
            "tests": 0.0
        }

        # Map signals to categories
        if signal_suite.blast_radius:
            categories["blast_radius"] = signal_suite.blast_radius.score
            categories["regression"] = max(categories["regression"], signal_suite.blast_radius.score)

        if signal_suite.cochange:
            categories["cochange"] = signal_suite.cochange.score

        if signal_suite.ownership:
            categories["ownership"] = signal_suite.ownership.score

        if signal_suite.incident_adjacency:
            categories["incident_adjacency"] = signal_suite.incident_adjacency.score

        if signal_suite.api_break:
            categories["api"] = signal_suite.api_break.score

        if signal_suite.security_risk:
            categories["security"] = signal_suite.security_risk.score

        if signal_suite.performance_risk:
            categories["performance"] = signal_suite.performance_risk.score

        if signal_suite.test_gap:
            categories["tests"] = signal_suite.test_gap.score

        return categories

    def _calculate_confidence_advanced(self, signals: List[AdvancedRiskSignal]) -> float:
        """Calculate overall confidence for advanced assessment"""
        if not signals:
            return 0.5

        # Weight by mathematical vs heuristic signals
        total_confidence = 0.0
        total_weight = 0.0

        for signal in signals:
            weight = 1.5 if signal.is_mathematical_signal() else 1.0
            total_confidence += signal.confidence * weight
            total_weight += weight

        return total_confidence / total_weight if total_weight > 0 else 0.5