"""
Static Code Analyzer

Provides static analysis capabilities for detecting patterns, complexity,
dependencies, and other code characteristics.
"""

import re
from typing import Dict, List, Optional, Any, Set, Tuple
from dataclasses import dataclass
from pathlib import Path
from collections import defaultdict, Counter

from .ast_parser import ASTParser, ParseResult, Language, ASTNode


@dataclass
class ComplexityMetrics:
    """Code complexity metrics"""
    cyclomatic_complexity: int
    cognitive_complexity: int
    lines_of_code: int
    lines_of_comments: int
    maintainability_index: float


@dataclass
class DependencyInfo:
    """Information about code dependencies"""
    imports: List[str]
    exports: List[str]
    internal_calls: List[str]
    external_calls: List[str]


@dataclass
class PatternMatch:
    """Represents a matched code pattern"""
    pattern_name: str
    line_number: int
    column: Optional[int]
    matched_text: str
    severity: str  # 'low', 'medium', 'high', 'critical'
    description: str
    suggestion: Optional[str]


@dataclass
class AnalysisResult:
    """Result of static code analysis"""
    file_path: str
    language: Language
    success: bool
    complexity: Optional[ComplexityMetrics]
    dependencies: Optional[DependencyInfo]
    patterns: List[PatternMatch]
    security_issues: List[PatternMatch]
    performance_issues: List[PatternMatch]
    maintainability_issues: List[PatternMatch]
    error_message: Optional[str]
    analysis_time_ms: float


class StaticAnalyzer:
    """Static code analysis engine"""

    def __init__(self):
        self.ast_parser = ASTParser()
        self._init_patterns()

    def _init_patterns(self):
        """Initialize analysis patterns"""
        # Security patterns
        self.security_patterns = {
            'sql_injection': {
                'patterns': [
                    r'execute\s*\(\s*["\'][^"\']*\+',
                    r'query\s*\(\s*["\'][^"\']*\+',
                    r'SELECT\s+.*\+.*FROM',
                ],
                'severity': 'high',
                'description': 'Potential SQL injection vulnerability',
                'suggestion': 'Use parameterized queries instead of string concatenation'
            },
            'xss': {
                'patterns': [
                    r'innerHTML\s*=.*\+',
                    r'document\.write\s*\(.*\+',
                    r'\.html\s*\(.*\+',
                ],
                'severity': 'medium',
                'description': 'Potential XSS vulnerability',
                'suggestion': 'Sanitize user input before rendering'
            },
            'hardcoded_secrets': {
                'patterns': [
                    r'password\s*=\s*["\'][^"\']{8,}["\']',
                    r'api[_-]?key\s*=\s*["\'][^"\']{10,}["\']',
                    r'secret\s*=\s*["\'][^"\']{10,}["\']',
                ],
                'severity': 'critical',
                'description': 'Hardcoded secret detected',
                'suggestion': 'Move secrets to environment variables or secure configuration'
            }
        }

        # Performance patterns
        self.performance_patterns = {
            'loop_with_io': {
                'patterns': [
                    r'for\s+.*:\s*\n.*?(requests\.|urllib\.|http)',
                    r'while\s+.*:\s*\n.*?(requests\.|urllib\.|http)',
                ],
                'severity': 'high',
                'description': 'I/O operation inside loop',
                'suggestion': 'Consider batching operations or using async programming'
            },
            'string_concat_loop': {
                'patterns': [
                    r'for\s+.*:\s*\n.*?\+=.*["\']',
                    r'while\s+.*:\s*\n.*?\+=.*["\']',
                ],
                'severity': 'medium',
                'description': 'String concatenation in loop',
                'suggestion': 'Use list.join() or StringBuilder for better performance'
            },
            'nested_loops': {
                'patterns': [
                    r'for\s+.*:\s*\n\s*for\s+.*:\s*\n\s*for',
                ],
                'severity': 'medium',
                'description': 'Deeply nested loops detected',
                'suggestion': 'Consider algorithm optimization or breaking into smaller functions'
            }
        }

        # Maintainability patterns
        self.maintainability_patterns = {
            'long_function': {
                'patterns': [],  # Handled by complexity analysis
                'severity': 'medium',
                'description': 'Function is too long',
                'suggestion': 'Break into smaller, more focused functions'
            },
            'magic_numbers': {
                'patterns': [
                    r'\b(?:0x[0-9a-fA-F]+|[0-9]+)\b(?!\s*[.)])',  # Numbers not in common contexts
                ],
                'severity': 'low',
                'description': 'Magic number detected',
                'suggestion': 'Replace with named constant'
            },
            'todo_comments': {
                'patterns': [
                    r'#\s*TODO',
                    r'//\s*TODO',
                    r'/\*\s*TODO',
                ],
                'severity': 'low',
                'description': 'TODO comment found',
                'suggestion': 'Address TODO items or create proper issues'
            }
        }

    def analyze_file(self, file_path: str) -> AnalysisResult:
        """Analyze a single file"""
        import time
        start_time = time.perf_counter()

        try:
            # Parse the file
            parse_result = self.ast_parser.parse_file(file_path)

            if not parse_result.success:
                return AnalysisResult(
                    file_path=file_path,
                    language=parse_result.language,
                    success=False,
                    complexity=None,
                    dependencies=None,
                    patterns=[],
                    security_issues=[],
                    performance_issues=[],
                    maintainability_issues=[],
                    error_message=parse_result.error_message,
                    analysis_time_ms=(time.perf_counter() - start_time) * 1000
                )

            # Read file content for pattern matching
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read()

            # Perform analysis
            complexity = self._analyze_complexity(parse_result, content)
            dependencies = self._analyze_dependencies(parse_result, content)
            patterns = self._find_patterns(content, parse_result.language)

            # Categorize patterns
            security_issues = []
            performance_issues = []
            maintainability_issues = []

            for pattern in patterns:
                if any(sec_name in pattern.pattern_name for sec_name in self.security_patterns.keys()):
                    security_issues.append(pattern)
                elif any(perf_name in pattern.pattern_name for perf_name in self.performance_patterns.keys()):
                    performance_issues.append(pattern)
                else:
                    maintainability_issues.append(pattern)

            analysis_time_ms = (time.perf_counter() - start_time) * 1000

            return AnalysisResult(
                file_path=file_path,
                language=parse_result.language,
                success=True,
                complexity=complexity,
                dependencies=dependencies,
                patterns=patterns,
                security_issues=security_issues,
                performance_issues=performance_issues,
                maintainability_issues=maintainability_issues,
                error_message=None,
                analysis_time_ms=analysis_time_ms
            )

        except Exception as e:
            return AnalysisResult(
                file_path=file_path,
                language=Language.UNKNOWN,
                success=False,
                complexity=None,
                dependencies=None,
                patterns=[],
                security_issues=[],
                performance_issues=[],
                maintainability_issues=[],
                error_message=str(e),
                analysis_time_ms=(time.perf_counter() - start_time) * 1000
            )

    def _analyze_complexity(self, parse_result: ParseResult, content: str) -> ComplexityMetrics:
        """Analyze code complexity"""
        lines = content.split('\n')
        lines_of_code = len([line for line in lines if line.strip() and not line.strip().startswith('#')])
        lines_of_comments = len([line for line in lines if line.strip().startswith('#') or line.strip().startswith('//')])

        # Calculate cyclomatic complexity
        cyclomatic_complexity = self._calculate_cyclomatic_complexity(content)

        # Calculate cognitive complexity (simplified)
        cognitive_complexity = self._calculate_cognitive_complexity(content)

        # Calculate maintainability index (simplified version)
        maintainability_index = self._calculate_maintainability_index(
            lines_of_code, cyclomatic_complexity, lines_of_comments
        )

        return ComplexityMetrics(
            cyclomatic_complexity=cyclomatic_complexity,
            cognitive_complexity=cognitive_complexity,
            lines_of_code=lines_of_code,
            lines_of_comments=lines_of_comments,
            maintainability_index=maintainability_index
        )

    def _analyze_dependencies(self, parse_result: ParseResult, content: str) -> DependencyInfo:
        """Analyze code dependencies"""
        imports = []
        exports = []
        internal_calls = []
        external_calls = []

        if parse_result.root_node:
            # Extract imports from AST
            import_nodes = parse_result.root_node.find_children_by_type('import')
            import_from_nodes = parse_result.root_node.find_children_by_type('import_from')

            for import_node in import_nodes + import_from_nodes:
                if 'module' in import_node.metadata:
                    imports.append(import_node.metadata['module'])
                if 'names' in import_node.metadata:
                    for name, alias in import_node.metadata['names']:
                        imports.append(name)

        # Extract exports (language-specific patterns)
        export_patterns = [
            r'export\s+(?:default\s+)?(\w+)',
            r'module\.exports\s*=\s*(\w+)',
            r'exports\.(\w+)',
        ]

        for pattern in export_patterns:
            matches = re.finditer(pattern, content)
            for match in matches:
                exports.append(match.group(1))

        # Simple function call detection
        function_call_pattern = r'(\w+)\s*\('
        matches = re.finditer(function_call_pattern, content)

        for match in matches:
            func_name = match.group(1)
            # Classify as internal vs external (simplified heuristic)
            if func_name in imports or '.' in func_name:
                external_calls.append(func_name)
            else:
                internal_calls.append(func_name)

        return DependencyInfo(
            imports=list(set(imports)),
            exports=list(set(exports)),
            internal_calls=list(set(internal_calls)),
            external_calls=list(set(external_calls))
        )

    def _find_patterns(self, content: str, language: Language) -> List[PatternMatch]:
        """Find patterns in code content"""
        patterns = []
        all_pattern_sets = [
            self.security_patterns,
            self.performance_patterns,
            self.maintainability_patterns
        ]

        for pattern_set in all_pattern_sets:
            for pattern_name, pattern_config in pattern_set.items():
                for pattern in pattern_config['patterns']:
                    matches = re.finditer(pattern, content, re.MULTILINE | re.IGNORECASE)

                    for match in matches:
                        line_number = content[:match.start()].count('\n') + 1

                        patterns.append(PatternMatch(
                            pattern_name=pattern_name,
                            line_number=line_number,
                            column=match.start() - content.rfind('\n', 0, match.start()) - 1,
                            matched_text=match.group(),
                            severity=pattern_config['severity'],
                            description=pattern_config['description'],
                            suggestion=pattern_config.get('suggestion')
                        ))

        return patterns

    def _calculate_cyclomatic_complexity(self, content: str) -> int:
        """Calculate cyclomatic complexity"""
        # Count decision points
        decision_patterns = [
            r'\bif\b', r'\belif\b', r'\belse\b',
            r'\bfor\b', r'\bwhile\b',
            r'\btry\b', r'\bcatch\b', r'\bexcept\b',
            r'\bcase\b', r'\bdefault\b',
            r'\b&&\b', r'\b\|\|\b', r'\band\b', r'\bor\b',
            r'\?.*:', # Ternary operator
        ]

        complexity = 1  # Base complexity
        for pattern in decision_patterns:
            matches = re.findall(pattern, content, re.IGNORECASE)
            complexity += len(matches)

        return complexity

    def _calculate_cognitive_complexity(self, content: str) -> int:
        """Calculate cognitive complexity (simplified)"""
        # Similar to cyclomatic but with nesting weights
        complexity = 0
        nesting_level = 0

        lines = content.split('\n')
        for line in lines:
            stripped = line.strip()

            # Increase nesting for certain constructs
            if re.search(r'\b(if|for|while|try|def|class)\b', stripped):
                nesting_level += 1
                complexity += nesting_level

            # Decrease nesting
            if stripped in ['end', '}'] or (stripped.startswith('except') or stripped.startswith('finally')):
                nesting_level = max(0, nesting_level - 1)

        return complexity

    def _calculate_maintainability_index(self, loc: int, cc: int, comments: int) -> float:
        """Calculate maintainability index (simplified)"""
        import math

        if loc == 0:
            return 100.0

        # Simplified version of the maintainability index formula
        mi = 171 - 5.2 * math.log(loc) - 0.23 * cc - 16.2 * math.log(loc + comments + 1)

        # Normalize to 0-100 scale
        return max(0.0, min(100.0, mi))

    def analyze_diff(self, old_content: str, new_content: str, language: Language) -> Dict[str, Any]:
        """Analyze differences between two versions of code"""
        old_patterns = self._find_patterns(old_content, language)
        new_patterns = self._find_patterns(new_content, language)

        added_patterns = []
        removed_patterns = []

        # Simple diff - in practice would need more sophisticated matching
        old_pattern_sigs = {(p.pattern_name, p.line_number): p for p in old_patterns}
        new_pattern_sigs = {(p.pattern_name, p.line_number): p for p in new_patterns}

        for sig, pattern in new_pattern_sigs.items():
            if sig not in old_pattern_sigs:
                added_patterns.append(pattern)

        for sig, pattern in old_pattern_sigs.items():
            if sig not in new_pattern_sigs:
                removed_patterns.append(pattern)

        return {
            'added_patterns': added_patterns,
            'removed_patterns': removed_patterns,
            'net_pattern_change': len(added_patterns) - len(removed_patterns)
        }