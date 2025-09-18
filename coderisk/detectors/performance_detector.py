"""
Performance Risk Detector

Detects performance anti-patterns like loops with I/O, inefficient queries,
string concatenation in loops, and other performance bottlenecks.

Time budget: 20-60ms
"""

import re
import ast
from typing import Dict, List, Set, Optional, Any
from pathlib import Path

from . import BaseDetector, DetectorResult, ChangeContext, FileChange, detector_registry


@detector_registry.register
class PerformanceRiskDetector(BaseDetector):
    """Detects performance anti-patterns and bottlenecks in code changes"""

    def __init__(self, repo_path: str):
        super().__init__(repo_path)

        # Performance anti-patterns by language
        self.patterns = {
            'python': {
                'loop_with_io': [
                    r'for\s+.*:\s*\n.*?(requests\.|urllib\.|http|database|db\.|session\.)',
                    r'while\s+.*:\s*\n.*?(requests\.|urllib\.|http|database|db\.|session\.)',
                ],
                'string_concat_loop': [
                    r'for\s+.*:\s*\n.*?\+?=.*str\(',
                    r'while\s+.*:\s*\n.*?\+?=.*["\']',
                ],
                'n_plus_one': [
                    r'for\s+.*:\s*\n.*?(\.get\(|\.filter\(|\.query\()',
                ],
                'inefficient_data_structures': [
                    r'for\s+.*in\s+.*:\s*\n.*?\.append\(',  # List comprehension candidate
                    r'dict\(\[\(.*for.*in.*\)\]\)',  # Dict comprehension candidate
                ]
            },
            'javascript': {
                'loop_with_io': [
                    r'for\s*\(.*\)\s*\{[^}]*?(fetch\(|axios\.|request\(|http)',
                    r'while\s*\(.*\)\s*\{[^}]*?(fetch\(|axios\.|request\(|http)',
                ],
                'string_concat_loop': [
                    r'for\s*\(.*\)\s*\{[^}]*?\+=.*["\']',
                    r'while\s*\(.*\)\s*\{[^}]*?\+=.*["\']',
                ],
                'dom_manipulation_loop': [
                    r'for\s*\(.*\)\s*\{[^}]*?(document\.|querySelector|getElementById)',
                ],
                'async_in_loop': [
                    r'for\s*\(.*\)\s*\{[^}]*?await\s+',
                    r'while\s*\(.*\)\s*\{[^}]*?await\s+',
                ]
            },
            'sql': {
                'missing_indexes': [
                    r'WHERE.*=.*AND.*=',  # Multiple conditions without indexes
                    r'ORDER BY.*,.*,',  # Multiple column sorts
                ],
                'select_star': [
                    r'SELECT\s+\*\s+FROM',
                ],
                'like_prefix': [
                    r'LIKE\s+["\']%.*["\']',  # Leading wildcard
                ]
            }
        }

        self.io_operations = [
            'requests', 'urllib', 'http', 'fetch', 'axios',
            'database', 'db.', 'session.', 'query', 'cursor',
            'file.', 'open(', 'read(', 'write(',
            'socket', 'connection', 'client.'
        ]

    async def analyze(self, context: ChangeContext) -> DetectorResult:
        """Analyze performance risks in code changes"""
        perf_risks = []
        total_risk_score = 0.0
        reasons = []
        anchors = []
        evidence = {
            "anti_patterns": [],
            "loops_with_io": [],
            "inefficient_queries": [],
            "string_operations": [],
            "hot_paths": []
        }

        for file_change in context.files_changed:
            if file_change.change_type == 'deleted':
                continue

            file_path = Path(self.repo_path) / file_change.path

            if not file_path.exists() or not self._is_code_file(file_path):
                continue

            # Analyze performance issues in this file
            file_risk = await self._analyze_file_performance(file_change, file_path)

            if file_risk['score'] > 0:
                perf_risks.append(file_risk)
                total_risk_score += file_risk['score']
                reasons.extend(file_risk['reasons'])
                anchors.extend(file_risk['anchors'])

                # Merge evidence
                for key in ['anti_patterns', 'loops_with_io', 'inefficient_queries']:
                    if key in file_risk:
                        evidence[key].extend(file_risk[key])

        # Normalize score
        if perf_risks:
            risk_score = min(total_risk_score / len(context.files_changed), 1.0)

            # Weight by file importance (hot paths get higher risk)
            for file_change in context.files_changed:
                if self._is_hot_path(file_change.path):
                    risk_score *= 1.5
                    evidence["hot_paths"].append(file_change.path)

            risk_score = min(risk_score, 1.0)
        else:
            risk_score = 0.0

        return DetectorResult(
            score=risk_score,
            reasons=reasons[:5],
            anchors=anchors[:10],
            evidence=evidence,
            execution_time_ms=0.0
        )

    async def _analyze_file_performance(self, file_change: FileChange, file_path: Path) -> Dict[str, Any]:
        """Analyze performance issues in a specific file"""
        file_risk = {
            'score': 0.0,
            'reasons': [],
            'anchors': [],
            'anti_patterns': [],
            'loops_with_io': [],
            'inefficient_queries': []
        }

        try:
            content = file_path.read_text(encoding='utf-8')
            language = self._detect_language(file_path)

            if language == 'python':
                await self._analyze_python_performance(content, file_change, file_risk)
            elif language in ['javascript', 'typescript']:
                await self._analyze_js_performance(content, file_change, file_risk)
            elif language == 'sql':
                await self._analyze_sql_performance(content, file_change, file_risk)
            else:
                await self._analyze_generic_performance(content, file_change, file_risk)

        except Exception as e:
            file_risk['reasons'].append(f"Failed to analyze {file_path.name}: {str(e)}")

        return file_risk

    async def _analyze_python_performance(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Analyze Python-specific performance issues"""
        try:
            tree = ast.parse(content)

            # Look for loops containing I/O operations
            for node in ast.walk(tree):
                if isinstance(node, (ast.For, ast.While)):
                    loop_body = ast.get_source_segment(content, node) or ""

                    # Check for I/O in loop
                    if any(io_op in loop_body for io_op in self.io_operations):
                        file_risk['loops_with_io'].append({
                            'line': node.lineno,
                            'type': 'io_in_loop',
                            'code_snippet': loop_body[:100] + '...' if len(loop_body) > 100 else loop_body
                        })
                        file_risk['score'] += 0.4
                        file_risk['reasons'].append(f"I/O operation in loop at line {node.lineno}")
                        file_risk['anchors'].append(f"{file_change.path}:{node.lineno}")

                    # Check for string concatenation in loop
                    if re.search(r'\+=.*["\']', loop_body):
                        file_risk['anti_patterns'].append({
                            'line': node.lineno,
                            'type': 'string_concat_loop',
                            'suggestion': 'Use list.join() or f-strings instead'
                        })
                        file_risk['score'] += 0.2
                        file_risk['reasons'].append(f"String concatenation in loop at line {node.lineno}")
                        file_risk['anchors'].append(f"{file_change.path}:{node.lineno}")

        except SyntaxError:
            # Fall back to regex analysis
            await self._analyze_with_patterns(content, file_change, file_risk, 'python')

    async def _analyze_js_performance(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Analyze JavaScript/TypeScript performance issues"""
        await self._analyze_with_patterns(content, file_change, file_risk, 'javascript')

        # Additional JS-specific checks
        lines = content.split('\n')

        for line_num, line in enumerate(lines, 1):
            # Check for inefficient DOM operations
            if re.search(r'document\.getElementById.*for.*\(', line):
                file_risk['score'] += 0.3
                file_risk['reasons'].append(f"DOM query in loop at line {line_num}")
                file_risk['anchors'].append(f"{file_change.path}:{line_num}")

            # Check for synchronous async operations
            if re.search(r'for.*await\s+', line):
                file_risk['score'] += 0.3
                file_risk['reasons'].append(f"Sequential async operations at line {line_num}")
                file_risk['anchors'].append(f"{file_change.path}:{line_num}")

    async def _analyze_sql_performance(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Analyze SQL performance issues"""
        lines = content.split('\n')

        for line_num, line in enumerate(lines, 1):
            line_upper = line.upper().strip()

            # Check for SELECT *
            if re.search(r'SELECT\s+\*', line_upper):
                file_risk['inefficient_queries'].append({
                    'line': line_num,
                    'issue': 'SELECT *',
                    'suggestion': 'Select only needed columns'
                })
                file_risk['score'] += 0.2
                file_risk['reasons'].append(f"SELECT * query at line {line_num}")

            # Check for LIKE with leading wildcard
            if re.search(r'LIKE\s+["\']%', line_upper):
                file_risk['score'] += 0.3
                file_risk['reasons'].append(f"LIKE with leading wildcard at line {line_num}")

            # Check for missing WHERE clauses on large operations
            if re.search(r'(UPDATE|DELETE).*(?!WHERE)', line_upper):
                file_risk['score'] += 0.5
                file_risk['reasons'].append(f"Potentially unsafe operation at line {line_num}")

    async def _analyze_generic_performance(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Generic performance analysis for any language"""
        lines = content.split('\n')

        for line_num, line in enumerate(lines, 1):
            # Look for obvious performance issues
            if re.search(r'(for|while).*\{[^}]*?(http|request|query|database)', line, re.IGNORECASE):
                file_risk['score'] += 0.3
                file_risk['reasons'].append(f"Potential I/O in loop at line {line_num}")

    async def _analyze_with_patterns(self, content: str, file_change: FileChange, file_risk: Dict[str, Any], language: str):
        """Analyze using regex patterns for a specific language"""
        patterns = self.patterns.get(language, {})

        for pattern_type, pattern_list in patterns.items():
            for pattern in pattern_list:
                matches = re.finditer(pattern, content, re.MULTILINE | re.DOTALL)

                for match in matches:
                    # Calculate line number
                    line_num = content[:match.start()].count('\n') + 1

                    risk_weight = self._get_pattern_risk_weight(pattern_type)
                    file_risk['score'] += risk_weight
                    file_risk['reasons'].append(f"{pattern_type.replace('_', ' ').title()} at line {line_num}")
                    file_risk['anchors'].append(f"{file_change.path}:{line_num}")

    def _get_pattern_risk_weight(self, pattern_type: str) -> float:
        """Get risk weight for different pattern types"""
        weights = {
            'loop_with_io': 0.4,
            'string_concat_loop': 0.2,
            'n_plus_one': 0.5,
            'inefficient_data_structures': 0.1,
            'dom_manipulation_loop': 0.3,
            'async_in_loop': 0.3,
            'missing_indexes': 0.4,
            'select_star': 0.2,
            'like_prefix': 0.3
        }
        return weights.get(pattern_type, 0.1)

    def _is_hot_path(self, file_path: str) -> bool:
        """Check if file is likely to be a performance-critical hot path"""
        hot_path_indicators = [
            'handler', 'controller', 'api', 'service',
            'middleware', 'router', 'process', 'worker',
            'core', 'engine', 'main', 'index'
        ]

        path_lower = file_path.lower()
        return any(indicator in path_lower for indicator in hot_path_indicators)

    def _detect_language(self, file_path: Path) -> Optional[str]:
        """Detect programming language"""
        suffix = file_path.suffix.lower()
        language_map = {
            '.py': 'python',
            '.js': 'javascript',
            '.jsx': 'javascript',
            '.ts': 'typescript',
            '.tsx': 'typescript',
            '.sql': 'sql',
            '.java': 'java',
            '.go': 'go',
            '.rs': 'rust'
        }
        return language_map.get(suffix)

    def _is_code_file(self, file_path: Path) -> bool:
        """Check if file is analyzable code"""
        code_extensions = {'.py', '.js', '.jsx', '.ts', '.tsx', '.java', '.go', '.rs', '.sql'}
        return file_path.suffix.lower() in code_extensions