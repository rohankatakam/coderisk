"""
Regression scaling model implementation.

Implements the exact mathematical formulas from docs/regression_scaling_model.md:
- Team Factor: 1 + 0.3 × log₂(team_size)
- Codebase Factor: 1 + 0.2 × (LOC/100k) × coupling_coefficient
- Change Velocity: 1 + (commits_per_week/100)
- Migration Multiplier: 2^(major_versions) × 1.5^(breaking_changes)
"""

import math
from typing import Dict, List, Optional, Tuple
from dataclasses import dataclass
from datetime import datetime, timedelta
from pathlib import Path
import logging

try:
    import git
    GIT_AVAILABLE = True
except ImportError:
    GIT_AVAILABLE = False
    git = None

logger = logging.getLogger(__name__)


@dataclass
class ScalingFactors:
    """Container for all regression scaling factors"""
    team_factor: float
    codebase_factor: float
    change_velocity: float
    migration_multiplier: float

    @property
    def total_multiplier(self) -> float:
        """Calculate total regression risk multiplier"""
        return self.team_factor * self.codebase_factor * self.change_velocity * self.migration_multiplier


@dataclass
class TeamMetrics:
    """Team composition and activity metrics"""
    active_contributors: int
    total_contributors: int
    commits_last_week: int
    commits_last_month: int
    contributor_experience: Dict[str, float]


@dataclass
class CodebaseMetrics:
    """Codebase size and complexity metrics"""
    total_loc: int
    file_count: int
    avg_file_size: float
    coupling_coefficient: float
    import_density: float


@dataclass
class MigrationIndicators:
    """Migration and upgrade indicators"""
    major_version_changes: int
    breaking_api_changes: int
    framework_migrations: List[str]
    dependency_upgrades: List[Tuple[str, str, str]]  # (package, old_ver, new_ver)


class TeamFactor:
    """Calculates team scaling factor: 1 + 0.3 × log₂(team_size)"""

    @staticmethod
    def calculate(team_size: int) -> float:
        """
        Calculate team factor.

        Args:
            team_size: Number of active team members

        Returns:
            Team scaling factor
        """
        if team_size <= 0:
            return 1.0

        # Handle edge case for team_size = 1 (log₂(1) = 0)
        if team_size == 1:
            return 1.0

        return 1 + 0.3 * math.log2(team_size)

    @staticmethod
    def get_risk_level(team_factor: float) -> str:
        """Get risk level description for team factor"""
        if team_factor < 1.3:
            return "Low (small team)"
        elif team_factor < 1.6:
            return "Medium (growing team)"
        elif team_factor < 2.0:
            return "High (large team)"
        else:
            return "Critical (enterprise team)"


class CodebaseFactor:
    """Calculates codebase factor: 1 + 0.2 × (LOC/100k) × coupling_coefficient"""

    @staticmethod
    def calculate(total_loc: int, coupling_coefficient: float) -> float:
        """
        Calculate codebase factor.

        Args:
            total_loc: Total lines of code
            coupling_coefficient: Coupling strength (0.0 to 1.0)

        Returns:
            Codebase scaling factor
        """
        if total_loc <= 0:
            return 1.0

        loc_ratio = total_loc / 100_000
        return 1 + 0.2 * loc_ratio * coupling_coefficient

    @staticmethod
    def estimate_coupling_coefficient(
        import_density: float,
        file_count: int,
        avg_file_size: float
    ) -> float:
        """
        Estimate coupling coefficient from codebase metrics.

        Args:
            import_density: Average imports per file
            file_count: Total number of files
            avg_file_size: Average file size in LOC

        Returns:
            Estimated coupling coefficient (0.0 to 1.0)
        """
        # Heuristic based on file structure and imports
        base_coupling = min(1.0, import_density / 20.0)  # Normalize import density

        # Adjust for file size (larger files often mean more coupling)
        size_factor = min(1.0, avg_file_size / 200.0)

        # Adjust for file count (more files can mean better modularity)
        modularity_factor = max(0.2, 1.0 - (file_count / 1000.0))

        coupling = base_coupling * (0.6 + 0.2 * size_factor + 0.2 * modularity_factor)
        return max(0.2, min(1.0, coupling))  # Clamp to reasonable range

    @staticmethod
    def get_risk_level(codebase_factor: float) -> str:
        """Get risk level description for codebase factor"""
        if codebase_factor < 1.1:
            return "Low (well-modularized)"
        elif codebase_factor < 1.5:
            return "Medium (moderate coupling)"
        elif codebase_factor < 3.0:
            return "High (tightly coupled)"
        else:
            return "Critical (monolithic)"


class ChangeVelocityFactor:
    """Calculates change velocity: 1 + (commits_per_week/100)"""

    @staticmethod
    def calculate(commits_per_week: float) -> float:
        """
        Calculate change velocity factor.

        Args:
            commits_per_week: Average commits per week

        Returns:
            Change velocity scaling factor
        """
        if commits_per_week < 0:
            return 1.0

        return 1 + (commits_per_week / 100)

    @staticmethod
    def get_risk_level(velocity_factor: float) -> str:
        """Get risk level description for velocity factor"""
        if velocity_factor < 1.2:
            return "Low (stable pace)"
        elif velocity_factor < 1.5:
            return "Medium (active development)"
        elif velocity_factor < 3.0:
            return "High (rapid development)"
        else:
            return "Critical (extreme velocity)"


class MigrationMultiplier:
    """Calculates migration multiplier: 2^(major_versions) × 1.5^(breaking_changes)"""

    @staticmethod
    def calculate(major_version_changes: int, breaking_api_changes: int) -> float:
        """
        Calculate migration multiplier.

        Args:
            major_version_changes: Number of major version upgrades
            breaking_api_changes: Number of breaking API changes

        Returns:
            Migration risk multiplier
        """
        if major_version_changes < 0:
            major_version_changes = 0
        if breaking_api_changes < 0:
            breaking_api_changes = 0

        major_component = 2 ** major_version_changes
        breaking_component = 1.5 ** breaking_api_changes

        return major_component * breaking_component

    @staticmethod
    def detect_migration_indicators(
        changed_files: List[str],
        commit_message: Optional[str] = None
    ) -> MigrationIndicators:
        """
        Detect migration indicators from changed files and commit message.

        Args:
            changed_files: List of changed file paths
            commit_message: Commit message text

        Returns:
            Migration indicators
        """
        major_versions = 0
        breaking_changes = 0
        frameworks = []
        dependencies = []

        # Check for dependency files
        dependency_files = [
            f for f in changed_files
            if any(name in f.lower() for name in [
                'package.json', 'requirements.txt', 'pom.xml', 'cargo.toml',
                'go.mod', 'gemfile', 'composer.json', 'package-lock.json',
                'yarn.lock', 'pipfile', 'poetry.lock'
            ])
        ]

        if dependency_files:
            # Simple heuristic: dependency file changes suggest potential migration
            major_versions = min(len(dependency_files), 2)  # Cap at 2

        # Check commit message for migration keywords
        if commit_message:
            msg_lower = commit_message.lower()
            migration_keywords = [
                'upgrade', 'migration', 'migrate', 'update', 'bump',
                'framework', 'version', 'breaking', 'refactor', 'modernize'
            ]

            keyword_count = sum(1 for keyword in migration_keywords if keyword in msg_lower)
            if keyword_count >= 2:
                breaking_changes = min(keyword_count - 1, 3)  # Cap at 3

            # Detect framework mentions
            framework_keywords = [
                'react', 'angular', 'vue', 'django', 'flask', 'spring',
                'express', 'rails', 'laravel', 'symfony'
            ]
            frameworks = [fw for fw in framework_keywords if fw in msg_lower]

        return MigrationIndicators(
            major_version_changes=major_versions,
            breaking_api_changes=breaking_changes,
            framework_migrations=frameworks,
            dependency_upgrades=dependencies
        )

    @staticmethod
    def get_risk_level(migration_multiplier: float) -> str:
        """Get risk level description for migration multiplier"""
        if migration_multiplier < 1.5:
            return "Low (minor changes)"
        elif migration_multiplier < 3.0:
            return "Medium (moderate migration)"
        elif migration_multiplier < 10.0:
            return "High (major migration)"
        else:
            return "Critical (framework migration)"


class RegressionScalingModel:
    """Main regression scaling model calculator"""

    def __init__(self, repo_path: str):
        self.repo_path = Path(repo_path)
        self._repo = None
        self._cached_metrics = {}

    def initialize(self) -> bool:
        """Initialize the model with repository access"""
        if not GIT_AVAILABLE:
            logger.warning("Git not available - using fallback metrics")
            return False

        try:
            self._repo = git.Repo(self.repo_path)
            return True
        except git.InvalidGitRepositoryError:
            logger.warning(f"Not a git repository: {self.repo_path}")
            return False

    def calculate_scaling_factors(
        self,
        changed_files: List[str],
        commit_message: Optional[str] = None,
        force_refresh: bool = False
    ) -> ScalingFactors:
        """
        Calculate all regression scaling factors.

        Args:
            changed_files: List of changed file paths
            commit_message: Commit message for migration detection
            force_refresh: Force refresh of cached metrics

        Returns:
            Complete scaling factors
        """
        # Get team metrics
        team_metrics = self._get_team_metrics(force_refresh)
        team_factor = TeamFactor.calculate(team_metrics.active_contributors)

        # Get codebase metrics
        codebase_metrics = self._get_codebase_metrics(force_refresh)
        codebase_factor = CodebaseFactor.calculate(
            codebase_metrics.total_loc,
            codebase_metrics.coupling_coefficient
        )

        # Calculate change velocity
        velocity_factor = ChangeVelocityFactor.calculate(team_metrics.commits_last_week)

        # Detect migration indicators
        migration_indicators = MigrationMultiplier.detect_migration_indicators(
            changed_files, commit_message
        )
        migration_multiplier = MigrationMultiplier.calculate(
            migration_indicators.major_version_changes,
            migration_indicators.breaking_api_changes
        )

        return ScalingFactors(
            team_factor=team_factor,
            codebase_factor=codebase_factor,
            change_velocity=velocity_factor,
            migration_multiplier=migration_multiplier
        )

    def _get_team_metrics(self, force_refresh: bool = False) -> TeamMetrics:
        """Get team composition and activity metrics"""
        cache_key = 'team_metrics'
        if not force_refresh and cache_key in self._cached_metrics:
            return self._cached_metrics[cache_key]

        if not self._repo:
            # Fallback values when no git repository
            metrics = TeamMetrics(
                active_contributors=4,
                total_contributors=6,
                commits_last_week=15,
                commits_last_month=60,
                contributor_experience={}
            )
            self._cached_metrics[cache_key] = metrics
            return metrics

        try:
            # Get contributors from last 30 days
            since_date = datetime.now() - timedelta(days=30)
            commits_last_month = list(self._repo.iter_commits(since=since_date))

            # Get contributors from last 7 days
            since_week = datetime.now() - timedelta(days=7)
            commits_last_week = list(self._repo.iter_commits(since=since_week))

            # Count unique contributors
            monthly_contributors = set(commit.author.name for commit in commits_last_month)
            weekly_contributors = set(commit.author.name for commit in commits_last_week)

            # Calculate contributor experience (simplified)
            contributor_experience = {}
            all_commits = list(self._repo.iter_commits(max_count=1000))
            for commit in all_commits:
                author = commit.author.name
                if author not in contributor_experience:
                    contributor_experience[author] = 0.0
                contributor_experience[author] += 1.0

            # Normalize experience scores
            if contributor_experience:
                max_commits = max(contributor_experience.values())
                for author in contributor_experience:
                    contributor_experience[author] = min(1.0, contributor_experience[author] / max_commits)

            metrics = TeamMetrics(
                active_contributors=len(weekly_contributors),
                total_contributors=len(monthly_contributors),
                commits_last_week=len(commits_last_week),
                commits_last_month=len(commits_last_month),
                contributor_experience=contributor_experience
            )

        except Exception as e:
            logger.warning(f"Failed to get team metrics: {e}")
            # Fallback values
            metrics = TeamMetrics(
                active_contributors=4,
                total_contributors=6,
                commits_last_week=15,
                commits_last_month=60,
                contributor_experience={}
            )

        self._cached_metrics[cache_key] = metrics
        return metrics

    def _get_codebase_metrics(self, force_refresh: bool = False) -> CodebaseMetrics:
        """Get codebase size and complexity metrics"""
        cache_key = 'codebase_metrics'
        if not force_refresh and cache_key in self._cached_metrics:
            return self._cached_metrics[cache_key]

        try:
            total_loc = 0
            file_count = 0
            file_sizes = []
            import_counts = []

            # Count lines of code and analyze imports
            code_extensions = {'.py', '.js', '.ts', '.java', '.cpp', '.c', '.go', '.rs', '.rb', '.php'}

            for file_path in self.repo_path.rglob('*'):
                if file_path.is_file() and file_path.suffix in code_extensions:
                    try:
                        content = file_path.read_text(encoding='utf-8', errors='ignore')
                        lines = [line.strip() for line in content.splitlines() if line.strip()]
                        file_loc = len(lines)

                        if file_loc > 0:
                            total_loc += file_loc
                            file_count += 1
                            file_sizes.append(file_loc)

                            # Count imports (simplified heuristic)
                            import_count = sum(1 for line in lines if any(
                                keyword in line for keyword in ['import', 'require', 'include', '#include']
                            ))
                            import_counts.append(import_count)

                    except Exception:
                        continue

            # Calculate metrics
            avg_file_size = sum(file_sizes) / len(file_sizes) if file_sizes else 100
            avg_imports = sum(import_counts) / len(import_counts) if import_counts else 5

            # Estimate coupling coefficient
            coupling_coefficient = CodebaseFactor.estimate_coupling_coefficient(
                import_density=avg_imports,
                file_count=file_count,
                avg_file_size=avg_file_size
            )

            metrics = CodebaseMetrics(
                total_loc=total_loc,
                file_count=file_count,
                avg_file_size=avg_file_size,
                coupling_coefficient=coupling_coefficient,
                import_density=avg_imports
            )

        except Exception as e:
            logger.warning(f"Failed to get codebase metrics: {e}")
            # Fallback values
            metrics = CodebaseMetrics(
                total_loc=50000,
                file_count=200,
                avg_file_size=250,
                coupling_coefficient=0.6,
                import_density=8.0
            )

        self._cached_metrics[cache_key] = metrics
        return metrics

    def get_risk_assessment(self, scaling_factors: ScalingFactors) -> Dict[str, str]:
        """Get risk level assessment for each factor"""
        return {
            'team': TeamFactor.get_risk_level(scaling_factors.team_factor),
            'codebase': CodebaseFactor.get_risk_level(scaling_factors.codebase_factor),
            'velocity': ChangeVelocityFactor.get_risk_level(scaling_factors.change_velocity),
            'migration': MigrationMultiplier.get_risk_level(scaling_factors.migration_multiplier),
            'overall': self._get_overall_risk_level(scaling_factors.total_multiplier)
        }

    def _get_overall_risk_level(self, total_multiplier: float) -> str:
        """Get overall risk level from total multiplier"""
        if total_multiplier < 3.0:
            return "Low (manageable risk)"
        elif total_multiplier < 10.0:
            return "Medium (elevated risk)"
        elif total_multiplier < 50.0:
            return "High (significant risk)"
        else:
            return "Critical (extreme risk)"