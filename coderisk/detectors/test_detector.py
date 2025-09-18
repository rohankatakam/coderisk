"""
Test Gap Risk Detector

Analyzes test coverage and identifies gaps in testing for changed code.
Calculates risk based on ratio of tests to code changes with smoothing.

Time budget: 10-20ms
"""

import re
from typing import Dict, List, Set, Optional, Any
from pathlib import Path

from . import BaseDetector, DetectorResult, ChangeContext, FileChange, detector_registry


@detector_registry.register
class TestGapDetector(BaseDetector):
    """Detects test coverage gaps for code changes"""

    def __init__(self, repo_path: str):
        super().__init__(repo_path)

        # Test file patterns by language/framework
        self.test_patterns = {
            'python': [
                r'test_.*\.py$',
                r'.*_test\.py$',
                r'tests/.*\.py$',
                r'.*test.*\.py$',
            ],
            'javascript': [
                r'.*\.test\.(js|ts|jsx|tsx)$',
                r'.*\.spec\.(js|ts|jsx|tsx)$',
                r'__tests__/.*\.(js|ts|jsx|tsx)$',
                r'tests?/.*\.(js|ts|jsx|tsx)$',
            ],
            'java': [
                r'.*Test\.java$',
                r'.*Tests\.java$',
                r'test/.*\.java$',
                r'src/test/.*\.java$',
            ],
            'go': [
                r'.*_test\.go$',
            ],
            'rust': [
                r'.*test.*\.rs$',
                r'tests/.*\.rs$',
            ]
        }

        # Test framework indicators
        self.test_frameworks = {
            'python': ['pytest', 'unittest', 'nose', 'testify'],
            'javascript': ['jest', 'mocha', 'jasmine', 'cypress', 'playwright'],
            'java': ['junit', 'testng', 'mockito'],
            'go': ['testing', 'testify'],
            'rust': ['test', 'proptest'],
        }

        # Risky code patterns that need tests
        self.risky_code_patterns = [
            r'class\s+\w+.*Exception',  # Custom exceptions
            r'def\s+.*parse.*\(',  # Parsing functions
            r'def\s+.*validate.*\(',  # Validation functions
            r'def\s+.*serialize.*\(',  # Serialization
            r'async\s+def',  # Async functions
            r'@.*route\(',  # Web routes
            r'@.*api\(',  # API endpoints
        ]

    async def analyze(self, context: ChangeContext) -> DetectorResult:
        """Analyze test coverage gaps"""
        test_risks = []
        total_risk_score = 0.0
        reasons = []
        anchors = []
        evidence = {
            "changed_code_files": [],
            "related_test_files": [],
            "missing_tests": [],
            "test_coverage_ratio": 0.0,
            "risky_code_without_tests": []
        }

        # Separate code files from test files
        code_files = []
        test_files = []

        for file_change in context.files_changed:
            if self._is_test_file(file_change.path):
                test_files.append(file_change)
                evidence["related_test_files"].append(file_change.path)
            elif self._is_code_file(file_change.path):
                code_files.append(file_change)
                evidence["changed_code_files"].append(file_change.path)

        # Calculate test coverage ratio
        if code_files:
            coverage_ratio = len(test_files) / len(code_files)
            evidence["test_coverage_ratio"] = round(coverage_ratio, 2)

            # Base risk from coverage ratio (inverted - low coverage = high risk)
            coverage_risk = max(0, (1.0 - coverage_ratio) * 0.8)
            total_risk_score += coverage_risk

            if coverage_ratio < 0.3:
                reasons.append(f"Low test coverage ratio: {coverage_ratio:.2f}")
            elif coverage_ratio < 0.6:
                reasons.append(f"Moderate test coverage ratio: {coverage_ratio:.2f}")

        # Analyze individual code files for test gaps
        for file_change in code_files:
            file_risk = await self._analyze_code_file_test_gap(file_change)

            if file_risk['score'] > 0:
                test_risks.append(file_risk)
                total_risk_score += file_risk['score']
                reasons.extend(file_risk['reasons'])
                anchors.extend(file_risk['anchors'])

                if file_risk.get('risky_patterns'):
                    evidence["risky_code_without_tests"].extend(file_risk['risky_patterns'])

        # Check for missing test files for critical modules
        await self._check_missing_critical_tests(code_files, test_files, evidence, total_risk_score, reasons)

        # Normalize score with smoothing
        if code_files:
            # Apply smoothing based on change size
            total_lines_changed = sum(f.lines_added + f.lines_deleted for f in code_files)
            smoothing_factor = min(1.0, total_lines_changed / 100)  # Reduce risk for small changes

            risk_score = min(total_risk_score / len(code_files), 1.0) * smoothing_factor
        else:
            risk_score = 0.0

        return DetectorResult(
            score=risk_score,
            reasons=reasons[:5],
            anchors=anchors[:10],
            evidence=evidence,
            execution_time_ms=0.0
        )

    async def _analyze_code_file_test_gap(self, file_change: FileChange) -> Dict[str, Any]:
        """Analyze a code file for test coverage gaps"""
        file_risk = {
            'score': 0.0,
            'reasons': [],
            'anchors': [],
            'risky_patterns': []
        }

        try:
            file_path = Path(self.repo_path) / file_change.path

            if not file_path.exists():
                return file_risk

            content = file_path.read_text(encoding='utf-8')

            # Check for corresponding test file
            if not self._has_corresponding_test_file(file_change.path):
                file_risk['score'] += 0.4
                file_risk['reasons'].append(f"No corresponding test file for {file_change.path}")
                file_risk['anchors'].append(f"{file_change.path}:missing_test_file")

            # Check for risky code patterns without tests
            await self._check_risky_patterns(content, file_change, file_risk)

            # Weight by file importance
            if self._is_critical_file(file_change.path):
                file_risk['score'] *= 1.5
                file_risk['reasons'].append(f"Critical file without adequate testing: {file_change.path}")

        except Exception as e:
            file_risk['reasons'].append(f"Failed to analyze {file_change.path}: {str(e)}")

        return file_risk

    async def _check_risky_patterns(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Check for risky code patterns that need testing"""
        lines = content.split('\n')

        for line_num, line in enumerate(lines, 1):
            for pattern in self.risky_code_patterns:
                match = re.search(pattern, line, re.IGNORECASE)
                if match:
                    # Check if this line was added in the change
                    line_was_added = self._line_was_added(line_num, file_change)

                    if line_was_added:
                        file_risk['risky_patterns'].append({
                            'line': line_num,
                            'pattern': pattern,
                            'code': line.strip(),
                            'type': self._classify_risky_pattern(pattern)
                        })

                        file_risk['score'] += 0.3
                        file_risk['reasons'].append(f"Risky code pattern at line {line_num}: {self._classify_risky_pattern(pattern)}")
                        file_risk['anchors'].append(f"{file_change.path}:{line_num}")

    async def _check_missing_critical_tests(self, code_files: List[FileChange], test_files: List[FileChange],
                                          evidence: Dict[str, Any], total_risk_score: float, reasons: List[str]):
        """Check for missing tests for critical code modules"""
        critical_modules = []

        for file_change in code_files:
            if self._is_critical_file(file_change.path):
                critical_modules.append(file_change.path)

        missing_critical_tests = []
        for critical_file in critical_modules:
            if not any(self._files_are_related(critical_file, test_file.path) for test_file in test_files):
                missing_critical_tests.append(critical_file)

        if missing_critical_tests:
            evidence["missing_tests"] = missing_critical_tests
            reasons.append(f"Critical files without tests: {len(missing_critical_tests)}")

    def _has_corresponding_test_file(self, code_file_path: str) -> bool:
        """Check if a code file has a corresponding test file"""
        code_path = Path(self.repo_path) / code_file_path
        code_dir = code_path.parent
        code_name = code_path.stem

        # Common test file patterns
        test_patterns = [
            f"test_{code_name}.py",
            f"{code_name}_test.py",
            f"{code_name}.test.js",
            f"{code_name}.spec.js",
            f"{code_name}Test.java",
            f"{code_name}_test.go",
        ]

        # Check in same directory
        for pattern in test_patterns:
            if (code_dir / pattern).exists():
                return True

        # Check in tests directory
        tests_dir = code_dir / "tests"
        if tests_dir.exists():
            for pattern in test_patterns:
                if (tests_dir / pattern).exists():
                    return True

        # Check in __tests__ directory (JavaScript)
        tests_dir = code_dir / "__tests__"
        if tests_dir.exists():
            for pattern in test_patterns:
                if (tests_dir / pattern).exists():
                    return True

        return False

    def _files_are_related(self, code_file: str, test_file: str) -> bool:
        """Check if a code file and test file are related"""
        code_name = Path(code_file).stem.lower()
        test_name = Path(test_file).stem.lower()

        # Remove common test prefixes/suffixes
        test_name_clean = test_name.replace('test_', '').replace('_test', '').replace('.test', '').replace('.spec', '')

        return code_name in test_name_clean or test_name_clean in code_name

    def _line_was_added(self, line_num: int, file_change: FileChange) -> bool:
        """Check if a line was added in the file change"""
        # This is a simplified check - in practice would need to parse git hunks properly
        for hunk in file_change.hunks:
            for line in hunk.get('lines', []):
                if line.startswith('+') and not line.startswith('+++'):
                    return True
        return False

    def _classify_risky_pattern(self, pattern: str) -> str:
        """Classify the type of risky pattern"""
        if 'exception' in pattern.lower():
            return 'exception_handling'
        elif 'parse' in pattern.lower():
            return 'parsing_logic'
        elif 'validate' in pattern.lower():
            return 'validation_logic'
        elif 'serialize' in pattern.lower():
            return 'serialization'
        elif 'async' in pattern.lower():
            return 'async_operation'
        elif 'route' in pattern.lower() or 'api' in pattern.lower():
            return 'api_endpoint'
        else:
            return 'complex_logic'

    def _is_critical_file(self, file_path: str) -> bool:
        """Check if file is critical and needs comprehensive testing"""
        critical_indicators = [
            'api', 'service', 'handler', 'controller',
            'auth', 'security', 'payment', 'billing',
            'core', 'main', 'engine', 'parser',
            'database', 'model', 'schema'
        ]

        path_lower = file_path.lower()
        return any(indicator in path_lower for indicator in critical_indicators)

    def _is_test_file(self, file_path: str) -> bool:
        """Check if file is a test file"""
        for language, patterns in self.test_patterns.items():
            for pattern in patterns:
                if re.search(pattern, file_path, re.IGNORECASE):
                    return True

        # Additional checks for test indicators
        test_indicators = ['test', 'spec', '__tests__', 'tests/']
        path_lower = file_path.lower()
        return any(indicator in path_lower for indicator in test_indicators)

    def _is_code_file(self, file_path: str) -> bool:
        """Check if file is a code file (not test, config, or docs)"""
        code_extensions = {'.py', '.js', '.jsx', '.ts', '.tsx', '.java', '.go', '.rs', '.c', '.cpp', '.h'}

        # Must have code extension
        if not any(file_path.endswith(ext) for ext in code_extensions):
            return False

        # Must not be a test file
        if self._is_test_file(file_path):
            return False

        # Must not be in excluded directories
        excluded_dirs = ['docs/', 'documentation/', 'examples/', 'scripts/', 'tools/']
        return not any(excluded in file_path.lower() for excluded in excluded_dirs)