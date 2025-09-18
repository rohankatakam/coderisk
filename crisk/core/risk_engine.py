"""
Core risk assessment engine implementing the regression scaling model
"""

import asyncio
import math
from typing import List, Dict, Optional, Any
from datetime import datetime
from pathlib import Path
import git

from ..models.risk_assessment import (
    RiskAssessment, RiskTier, RiskSignal, ChangeContext,
    RiskEvidence, RiskRecommendation
)
try:
    from .cognee_integration import CogneeCodeAnalyzer
    COGNEE_AVAILABLE = True
except ImportError:
    COGNEE_AVAILABLE = False
    from .simple_analyzer import SimpleCodeAnalyzer


class RiskEngine:
    """Main risk assessment engine"""

    def __init__(self, repo_path: str):
        self.repo_path = Path(repo_path).resolve()
        if COGNEE_AVAILABLE:
            self.analyzer = CogneeCodeAnalyzer(repo_path)
        else:
            self.analyzer = SimpleCodeAnalyzer(repo_path)
        self._repo = None

    async def initialize(self) -> None:
        """Initialize the risk engine"""
        await self.analyzer.initialize()
        try:
            self._repo = git.Repo(self.repo_path)
        except git.InvalidGitRepositoryError:
            print(f"Warning: {self.repo_path} is not a git repository")
            self._repo = None

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
        """Core risk assessment logic"""
        start_time = datetime.now()

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