"""
File Analyzer

High-level file analysis that combines AST parsing and static analysis
to provide comprehensive insights about code files.
"""

import os
from typing import Dict, List, Optional, Any, Set
from dataclasses import dataclass
from pathlib import Path
from collections import defaultdict

from .ast_parser import ASTParser, Language
from .static_analyzer import StaticAnalyzer, AnalysisResult


@dataclass
class FileMetrics:
    """Basic file metrics"""
    size_bytes: int
    line_count: int
    function_count: int
    class_count: int
    import_count: int
    last_modified: float


@dataclass
class FileAnalysisResult:
    """Comprehensive file analysis result"""
    file_path: str
    language: Language
    metrics: FileMetrics
    static_analysis: AnalysisResult
    risk_factors: List[str]
    recommendations: List[str]
    importance_score: float  # 0.0 to 1.0
    change_frequency: Optional[float]  # If git history available


class FileAnalyzer:
    """High-level file analyzer combining multiple analysis techniques"""

    def __init__(self):
        self.ast_parser = ASTParser()
        self.static_analyzer = StaticAnalyzer()

        # File importance indicators
        self.importance_keywords = {
            'critical': ['main', 'core', 'engine', 'auth', 'security', 'payment'],
            'high': ['api', 'service', 'handler', 'controller', 'model', 'database'],
            'medium': ['util', 'helper', 'common', 'shared', 'config'],
            'low': ['test', 'example', 'demo', 'sample', 'mock']
        }

    def analyze_file(self, file_path: str, include_git_history: bool = False) -> FileAnalysisResult:
        """Perform comprehensive file analysis"""
        path = Path(file_path)

        # Get basic file metrics
        metrics = self._get_file_metrics(path)

        # Perform static analysis
        static_analysis = self.static_analyzer.analyze_file(file_path)

        # Calculate importance score
        importance_score = self._calculate_importance_score(path, static_analysis)

        # Identify risk factors
        risk_factors = self._identify_risk_factors(static_analysis, metrics)

        # Generate recommendations
        recommendations = self._generate_recommendations(static_analysis, risk_factors)

        # Get change frequency from git if requested
        change_frequency = None
        if include_git_history:
            change_frequency = self._get_change_frequency(file_path)

        return FileAnalysisResult(
            file_path=file_path,
            language=static_analysis.language,
            metrics=metrics,
            static_analysis=static_analysis,
            risk_factors=risk_factors,
            recommendations=recommendations,
            importance_score=importance_score,
            change_frequency=change_frequency
        )

    def analyze_directory(self, directory_path: str, extensions: Optional[Set[str]] = None) -> List[FileAnalysisResult]:
        """Analyze all files in a directory"""
        if extensions is None:
            extensions = {'.py', '.js', '.jsx', '.ts', '.tsx', '.java', '.go', '.rs'}

        results = []
        directory = Path(directory_path)

        for file_path in directory.rglob('*'):
            if file_path.is_file() and file_path.suffix in extensions:
                # Skip test files and build artifacts
                if self._should_skip_file(file_path):
                    continue

                try:
                    result = self.analyze_file(str(file_path))
                    results.append(result)
                except Exception as e:
                    # Log error but continue with other files
                    print(f"Failed to analyze {file_path}: {e}")

        return results

    def get_project_summary(self, analysis_results: List[FileAnalysisResult]) -> Dict[str, Any]:
        """Generate project-level summary from file analysis results"""
        if not analysis_results:
            return {}

        # Language distribution
        language_counts = defaultdict(int)
        for result in analysis_results:
            language_counts[result.language.value] += 1

        # Risk distribution
        risk_levels = {'low': 0, 'medium': 0, 'high': 0, 'critical': 0}
        for result in analysis_results:
            if result.static_analysis.success:
                total_issues = (
                    len(result.static_analysis.security_issues) +
                    len(result.static_analysis.performance_issues) +
                    len(result.static_analysis.maintainability_issues)
                )

                if total_issues == 0:
                    risk_levels['low'] += 1
                elif total_issues <= 3:
                    risk_levels['medium'] += 1
                elif total_issues <= 10:
                    risk_levels['high'] += 1
                else:
                    risk_levels['critical'] += 1

        # Top risk factors
        all_risk_factors = []
        for result in analysis_results:
            all_risk_factors.extend(result.risk_factors)

        risk_factor_counts = defaultdict(int)
        for factor in all_risk_factors:
            risk_factor_counts[factor] += 1

        top_risk_factors = sorted(risk_factor_counts.items(), key=lambda x: x[1], reverse=True)[:10]

        # Most important files
        important_files = sorted(
            analysis_results,
            key=lambda x: x.importance_score,
            reverse=True
        )[:10]

        # Complexity distribution
        complexity_stats = {
            'avg_cyclomatic': 0,
            'avg_cognitive': 0,
            'avg_maintainability': 0,
            'files_with_metrics': 0
        }

        for result in analysis_results:
            if result.static_analysis.complexity:
                complexity_stats['avg_cyclomatic'] += result.static_analysis.complexity.cyclomatic_complexity
                complexity_stats['avg_cognitive'] += result.static_analysis.complexity.cognitive_complexity
                complexity_stats['avg_maintainability'] += result.static_analysis.complexity.maintainability_index
                complexity_stats['files_with_metrics'] += 1

        if complexity_stats['files_with_metrics'] > 0:
            complexity_stats['avg_cyclomatic'] /= complexity_stats['files_with_metrics']
            complexity_stats['avg_cognitive'] /= complexity_stats['files_with_metrics']
            complexity_stats['avg_maintainability'] /= complexity_stats['files_with_metrics']

        return {
            'total_files': len(analysis_results),
            'languages': dict(language_counts),
            'risk_distribution': risk_levels,
            'top_risk_factors': top_risk_factors,
            'most_important_files': [f.file_path for f in important_files],
            'complexity_stats': complexity_stats,
            'total_lines_of_code': sum(r.metrics.line_count for r in analysis_results),
            'total_functions': sum(r.metrics.function_count for r in analysis_results),
            'total_classes': sum(r.metrics.class_count for r in analysis_results)
        }

    def _get_file_metrics(self, file_path: Path) -> FileMetrics:
        """Get basic file metrics"""
        try:
            stat = file_path.stat()
            size_bytes = stat.st_size
            last_modified = stat.st_mtime

            # Count lines
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read()
                lines = content.split('\n')

            line_count = len(lines)

            # Parse for structural metrics
            parse_result = self.ast_parser.parse_content(content, self.ast_parser.detect_language(str(file_path)))

            function_count = 0
            class_count = 0
            import_count = 0

            if parse_result.success and parse_result.root_node:
                function_count = len(parse_result.find_functions())
                class_count = len(parse_result.find_classes())
                import_count = len(parse_result.find_imports())

            return FileMetrics(
                size_bytes=size_bytes,
                line_count=line_count,
                function_count=function_count,
                class_count=class_count,
                import_count=import_count,
                last_modified=last_modified
            )

        except Exception:
            return FileMetrics(
                size_bytes=0,
                line_count=0,
                function_count=0,
                class_count=0,
                import_count=0,
                last_modified=0.0
            )

    def _calculate_importance_score(self, file_path: Path, static_analysis: AnalysisResult) -> float:
        """Calculate file importance score (0.0 to 1.0)"""
        score = 0.0
        file_name = file_path.name.lower()
        file_path_str = str(file_path).lower()

        # Check importance keywords
        for level, keywords in self.importance_keywords.items():
            for keyword in keywords:
                if keyword in file_name or keyword in file_path_str:
                    if level == 'critical':
                        score += 0.4
                    elif level == 'high':
                        score += 0.3
                    elif level == 'medium':
                        score += 0.2
                    elif level == 'low':
                        score -= 0.1  # Test files are less important
                    break

        # Size and complexity factors
        if static_analysis.success and static_analysis.complexity:
            # Larger files with more functions are often more important
            if static_analysis.complexity.lines_of_code > 500:
                score += 0.2
            elif static_analysis.complexity.lines_of_code > 100:
                score += 0.1

        # Dependency factors
        if static_analysis.success and static_analysis.dependencies:
            # Files with many exports are often important
            if len(static_analysis.dependencies.exports) > 10:
                score += 0.2
            elif len(static_analysis.dependencies.exports) > 5:
                score += 0.1

        # Path-based importance
        path_parts = file_path.parts
        if 'src' in path_parts or 'lib' in path_parts:
            score += 0.1
        if 'test' in path_parts or 'tests' in path_parts:
            score -= 0.2

        return max(0.0, min(1.0, score))

    def _identify_risk_factors(self, static_analysis: AnalysisResult, metrics: FileMetrics) -> List[str]:
        """Identify risk factors for the file"""
        risk_factors = []

        if not static_analysis.success:
            risk_factors.append('parse_failure')
            return risk_factors

        # Complexity risks
        if static_analysis.complexity:
            if static_analysis.complexity.cyclomatic_complexity > 20:
                risk_factors.append('high_cyclomatic_complexity')
            if static_analysis.complexity.cognitive_complexity > 30:
                risk_factors.append('high_cognitive_complexity')
            if static_analysis.complexity.maintainability_index < 20:
                risk_factors.append('low_maintainability')

        # Size risks
        if metrics.line_count > 1000:
            risk_factors.append('large_file')
        if metrics.function_count > 50:
            risk_factors.append('many_functions')

        # Security risks
        if static_analysis.security_issues:
            critical_security = [i for i in static_analysis.security_issues if i.severity == 'critical']
            if critical_security:
                risk_factors.append('critical_security_issues')
            elif static_analysis.security_issues:
                risk_factors.append('security_issues')

        # Performance risks
        if static_analysis.performance_issues:
            high_perf = [i for i in static_analysis.performance_issues if i.severity in ['high', 'critical']]
            if high_perf:
                risk_factors.append('performance_issues')

        # Documentation risks
        if static_analysis.complexity and static_analysis.complexity.lines_of_comments == 0:
            if metrics.line_count > 100:
                risk_factors.append('no_documentation')

        return risk_factors

    def _generate_recommendations(self, static_analysis: AnalysisResult, risk_factors: List[str]) -> List[str]:
        """Generate recommendations based on analysis"""
        recommendations = []

        # Complexity recommendations
        if 'high_cyclomatic_complexity' in risk_factors:
            recommendations.append('Break down complex functions into smaller, more focused units')

        if 'high_cognitive_complexity' in risk_factors:
            recommendations.append('Reduce nesting and simplify control flow')

        if 'low_maintainability' in risk_factors:
            recommendations.append('Refactor to improve code maintainability')

        # Size recommendations
        if 'large_file' in risk_factors:
            recommendations.append('Consider splitting this file into smaller modules')

        if 'many_functions' in risk_factors:
            recommendations.append('Group related functions into classes or separate modules')

        # Security recommendations
        if 'critical_security_issues' in risk_factors:
            recommendations.append('Address critical security vulnerabilities immediately')

        if 'security_issues' in risk_factors:
            recommendations.append('Review and fix security-related issues')

        # Performance recommendations
        if 'performance_issues' in risk_factors:
            recommendations.append('Optimize performance bottlenecks')

        # Documentation recommendations
        if 'no_documentation' in risk_factors:
            recommendations.append('Add comments and documentation for complex logic')

        # General recommendations based on issues
        if static_analysis.success:
            if static_analysis.patterns:
                recommendations.append('Review flagged code patterns and consider refactoring')

        return recommendations

    def _get_change_frequency(self, file_path: str) -> Optional[float]:
        """Get change frequency from git history (if available)"""
        try:
            import subprocess
            result = subprocess.run(
                ['git', 'log', '--oneline', '--follow', file_path],
                capture_output=True,
                text=True,
                cwd=Path(file_path).parent
            )

            if result.returncode == 0:
                commit_count = len(result.stdout.strip().split('\n'))
                return float(commit_count)

        except Exception:
            pass

        return None

    def _should_skip_file(self, file_path: Path) -> bool:
        """Check if file should be skipped during analysis"""
        skip_patterns = [
            'test', 'tests', '__pycache__', 'node_modules',
            'dist', 'build', 'target', '.git'
        ]

        path_str = str(file_path).lower()
        return any(pattern in path_str for pattern in skip_patterns)