"""
Concurrency Risk Detector

Detects concurrency-related risks like race conditions, deadlocks,
shared state modifications, and missing synchronization.

Time budget: 20-60ms
"""

import re
import ast
from typing import Dict, List, Set, Optional, Any
from pathlib import Path

from . import BaseDetector, DetectorResult, ChangeContext, FileChange, detector_registry


@detector_registry.register
class ConcurrencyRiskDetector(BaseDetector):
    """Detects concurrency and thread safety issues in code changes"""

    def __init__(self, repo_path: str):
        super().__init__(repo_path)

        # Concurrency patterns by language
        self.patterns = {
            'python': {
                'shared_state_writes': [
                    r'global\s+\w+.*=',
                    r'self\.\w+\s*=.*(?!self\.)',  # Instance variable modification
                    r'class\s+\w+:.*\n.*\w+\s*=',  # Class variable modification
                ],
                'missing_locks': [
                    r'threading\.',
                    r'multiprocessing\.',
                    r'concurrent\.futures',
                ],
                'async_issues': [
                    r'async\s+def.*(?!await)',
                    r'await.*(?!async)',
                ]
            },
            'javascript': {
                'shared_state_writes': [
                    r'window\.\w+\s*=',
                    r'global\.\w+\s*=',
                    r'this\.\w+\s*=.*(?!this\.)',
                ],
                'race_conditions': [
                    r'setTimeout.*\n.*=',
                    r'setInterval.*\n.*=',
                    r'Promise\.all.*\n.*=',
                ],
                'async_issues': [
                    r'async\s+function.*(?!await)',
                    r'\.then\(.*\)\.catch\(',
                ]
            },
            'java': {
                'shared_state_writes': [
                    r'static\s+\w+.*=',
                    r'public\s+\w+.*=',
                    r'volatile\s+\w+.*=',
                ],
                'missing_synchronization': [
                    r'public\s+void\s+\w+.*\{(?!.*synchronized)',
                    r'static\s+void\s+\w+.*\{(?!.*synchronized)',
                ],
                'lock_ordering': [
                    r'synchronized\s*\(',
                    r'Lock\s+\w+',
                ]
            }
        }

        # Shared state indicators
        self.shared_state_keywords = [
            'global', 'static', 'class', 'singleton',
            'cache', 'registry', 'pool', 'queue',
            'shared', 'common', 'state'
        ]

        # Synchronization primitives
        self.sync_primitives = [
            'lock', 'mutex', 'semaphore', 'barrier',
            'synchronized', 'atomic', 'volatile',
            'concurrent', 'thread-safe'
        ]

    async def analyze(self, context: ChangeContext) -> DetectorResult:
        """Analyze concurrency risks in code changes"""
        concurrency_risks = []
        total_risk_score = 0.0
        reasons = []
        anchors = []
        evidence = {
            "shared_state_modifications": [],
            "missing_synchronization": [],
            "potential_race_conditions": [],
            "deadlock_risks": [],
            "async_await_issues": []
        }

        for file_change in context.files_changed:
            if file_change.change_type == 'deleted':
                continue

            file_path = Path(self.repo_path) / file_change.path

            if not file_path.exists() or not self._is_code_file(file_path):
                continue

            # Analyze concurrency issues in this file
            file_risk = await self._analyze_file_concurrency(file_change, file_path)

            if file_risk['score'] > 0:
                concurrency_risks.append(file_risk)
                total_risk_score += file_risk['score']
                reasons.extend(file_risk['reasons'])
                anchors.extend(file_risk['anchors'])

                # Merge evidence
                for key in evidence:
                    if key in file_risk:
                        evidence[key].extend(file_risk[key])

        # Normalize score
        if concurrency_risks:
            risk_score = min(total_risk_score / len(context.files_changed), 1.0)

            # Weight by potential impact
            if any(risk.get('severity') == 'critical' for risk in evidence.get('deadlock_risks', [])):
                risk_score = max(risk_score, 0.8)
        else:
            risk_score = 0.0

        return DetectorResult(
            score=risk_score,
            reasons=reasons[:5],
            anchors=anchors[:10],
            evidence=evidence,
            execution_time_ms=0.0
        )

    async def _analyze_file_concurrency(self, file_change: FileChange, file_path: Path) -> Dict[str, Any]:
        """Analyze concurrency issues in a specific file"""
        file_risk = {
            'score': 0.0,
            'reasons': [],
            'anchors': [],
            'shared_state_modifications': [],
            'missing_synchronization': [],
            'potential_race_conditions': [],
            'deadlock_risks': [],
            'async_await_issues': []
        }

        try:
            content = file_path.read_text(encoding='utf-8')
            language = self._detect_language(file_path)

            if language == 'python':
                await self._analyze_python_concurrency(content, file_change, file_risk)
            elif language in ['javascript', 'typescript']:
                await self._analyze_js_concurrency(content, file_change, file_risk)
            elif language == 'java':
                await self._analyze_java_concurrency(content, file_change, file_risk)
            else:
                await self._analyze_generic_concurrency(content, file_change, file_risk)

            # Common analysis for all languages
            await self._check_shared_state_access(content, file_change, file_risk)
            await self._check_lock_ordering(content, file_change, file_risk)

        except Exception as e:
            file_risk['reasons'].append(f"Failed to analyze {file_path.name}: {str(e)}")

        return file_risk

    async def _analyze_python_concurrency(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Analyze Python-specific concurrency issues"""
        try:
            tree = ast.parse(content)

            # Check for global variable modifications
            for node in ast.walk(tree):
                if isinstance(node, ast.Global):
                    for name in node.names:
                        file_risk['shared_state_modifications'].append({
                            'line': node.lineno,
                            'variable': name,
                            'type': 'global_modification',
                            'severity': 'medium'
                        })
                        file_risk['score'] += 0.3
                        file_risk['reasons'].append(f"Global variable modification: {name}")

                # Check for class variable assignments
                elif isinstance(node, ast.Assign):
                    for target in node.targets:
                        if isinstance(target, ast.Attribute) and isinstance(target.value, ast.Name):
                            if target.value.id in ['self', 'cls']:
                                file_risk['shared_state_modifications'].append({
                                    'line': node.lineno,
                                    'variable': target.attr,
                                    'type': 'instance_variable',
                                    'severity': 'low'
                                })
                                file_risk['score'] += 0.1

        except SyntaxError:
            # Fall back to regex analysis
            await self._analyze_with_patterns(content, file_change, file_risk, 'python')

        # Check for threading without proper synchronization
        if re.search(r'threading\.|Thread\(', content) and not re.search(r'Lock\(|RLock\(|Semaphore\(', content):
            file_risk['missing_synchronization'].append({
                'type': 'threading_without_locks',
                'severity': 'high'
            })
            file_risk['score'] += 0.4
            file_risk['reasons'].append("Threading used without synchronization primitives")

    async def _analyze_js_concurrency(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Analyze JavaScript/TypeScript concurrency issues"""
        lines = content.split('\n')

        for line_num, line in enumerate(lines, 1):
            # Check for async/await issues
            if re.search(r'async\s+function', line) and 'await' not in content[content.find(line):]:
                file_risk['async_await_issues'].append({
                    'line': line_num,
                    'issue': 'async_without_await',
                    'severity': 'medium'
                })
                file_risk['score'] += 0.2
                file_risk['reasons'].append(f"Async function without await at line {line_num}")

            # Check for Promise race conditions
            if re.search(r'Promise\.all\(', line) and re.search(r'\.then\(', line):
                file_risk['potential_race_conditions'].append({
                    'line': line_num,
                    'type': 'promise_race',
                    'severity': 'medium'
                })
                file_risk['score'] += 0.3

            # Check for shared state modifications
            if re.search(r'(window|global)\.\w+\s*=', line):
                file_risk['shared_state_modifications'].append({
                    'line': line_num,
                    'type': 'global_state_write',
                    'severity': 'high'
                })
                file_risk['score'] += 0.4

    async def _analyze_java_concurrency(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Analyze Java concurrency issues"""
        lines = content.split('\n')

        synchronized_blocks = []
        lock_acquisitions = []

        for line_num, line in enumerate(lines, 1):
            # Track synchronized blocks and locks for deadlock analysis
            if re.search(r'synchronized\s*\(', line):
                synchronized_blocks.append(line_num)

            if re.search(r'\.lock\(\)', line):
                lock_acquisitions.append(line_num)

            # Check for static variable modifications
            if re.search(r'static\s+\w+.*=', line):
                file_risk['shared_state_modifications'].append({
                    'line': line_num,
                    'type': 'static_variable',
                    'severity': 'high'
                })
                file_risk['score'] += 0.4

            # Check for volatile variables
            if 'volatile' in line:
                file_risk['shared_state_modifications'].append({
                    'line': line_num,
                    'type': 'volatile_variable',
                    'severity': 'medium'
                })
                file_risk['score'] += 0.2

        # Analyze potential deadlock scenarios
        if len(synchronized_blocks) > 1 or len(lock_acquisitions) > 1:
            file_risk['deadlock_risks'].append({
                'type': 'multiple_locks',
                'locations': synchronized_blocks + lock_acquisitions,
                'severity': 'high'
            })
            file_risk['score'] += 0.5
            file_risk['reasons'].append("Multiple synchronization points detected")

    async def _analyze_generic_concurrency(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Generic concurrency analysis"""
        lines = content.split('\n')

        for line_num, line in enumerate(lines, 1):
            # Look for common concurrency keywords
            for keyword in self.shared_state_keywords:
                if keyword in line.lower() and '=' in line:
                    file_risk['score'] += 0.1
                    file_risk['reasons'].append(f"Potential shared state modification at line {line_num}")

    async def _check_shared_state_access(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Check for unsafe shared state access patterns"""
        # Look for read-modify-write operations
        rmw_patterns = [
            r'\w+\s*\+=',
            r'\w+\s*-=',
            r'\w+\s*\*=',
            r'\w+\s*/=',
            r'\+\+\w+',
            r'\w+\+\+',
        ]

        for pattern in rmw_patterns:
            matches = re.finditer(pattern, content)
            for match in matches:
                line_num = content[:match.start()].count('\n') + 1
                file_risk['potential_race_conditions'].append({
                    'line': line_num,
                    'type': 'read_modify_write',
                    'pattern': match.group(),
                    'severity': 'medium'
                })
                file_risk['score'] += 0.2

    async def _check_lock_ordering(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Check for potential deadlock due to lock ordering"""
        # This is a simplified check - in practice, would need more sophisticated analysis
        lock_patterns = [
            r'synchronized\s*\(',
            r'\.lock\(\)',
            r'with\s+.*lock',
            r'mutex\.',
        ]

        lock_lines = []
        for pattern in lock_patterns:
            matches = re.finditer(pattern, content)
            for match in matches:
                line_num = content[:match.start()].count('\n') + 1
                lock_lines.append(line_num)

        # If multiple locks in close proximity, flag as potential deadlock risk
        if len(lock_lines) > 1:
            for i in range(len(lock_lines) - 1):
                if lock_lines[i + 1] - lock_lines[i] < 10:  # Within 10 lines
                    file_risk['deadlock_risks'].append({
                        'lines': [lock_lines[i], lock_lines[i + 1]],
                        'type': 'close_proximity_locks',
                        'severity': 'medium'
                    })
                    file_risk['score'] += 0.3

    async def _analyze_with_patterns(self, content: str, file_change: FileChange, file_risk: Dict[str, Any], language: str):
        """Analyze using regex patterns for a specific language"""
        patterns = self.patterns.get(language, {})

        for pattern_type, pattern_list in patterns.items():
            for pattern in pattern_list:
                matches = re.finditer(pattern, content, re.MULTILINE | re.DOTALL)

                for match in matches:
                    line_num = content[:match.start()].count('\n') + 1
                    risk_weight = self._get_pattern_risk_weight(pattern_type)

                    file_risk['score'] += risk_weight
                    file_risk['reasons'].append(f"{pattern_type.replace('_', ' ').title()} at line {line_num}")
                    file_risk['anchors'].append(f"{file_change.path}:{line_num}")

    def _get_pattern_risk_weight(self, pattern_type: str) -> float:
        """Get risk weight for different pattern types"""
        weights = {
            'shared_state_writes': 0.3,
            'missing_locks': 0.4,
            'async_issues': 0.2,
            'race_conditions': 0.4,
            'missing_synchronization': 0.5,
            'lock_ordering': 0.3
        }
        return weights.get(pattern_type, 0.1)

    def _detect_language(self, file_path: Path) -> Optional[str]:
        """Detect programming language"""
        suffix = file_path.suffix.lower()
        language_map = {
            '.py': 'python',
            '.js': 'javascript',
            '.jsx': 'javascript',
            '.ts': 'typescript',
            '.tsx': 'typescript',
            '.java': 'java',
            '.go': 'go',
            '.rs': 'rust'
        }
        return language_map.get(suffix)

    def _is_code_file(self, file_path: Path) -> bool:
        """Check if file is analyzable code"""
        code_extensions = {'.py', '.js', '.jsx', '.ts', '.tsx', '.java', '.go', '.rs', '.c', '.cpp', '.h'}
        return file_path.suffix.lower() in code_extensions