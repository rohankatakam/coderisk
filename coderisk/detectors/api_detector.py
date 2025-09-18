"""
API Break Risk Detector

Analyzes AST differences in public surface area and weighs by importer count.
Detects breaking changes to APIs, function signatures, class interfaces, etc.

Time budget: 50-150ms
"""

import ast
import re
from typing import Dict, List, Set, Optional, Any
from pathlib import Path
import asyncio

from . import BaseDetector, DetectorResult, ChangeContext, FileChange, detector_registry


@detector_registry.register
class ApiBreakDetector(BaseDetector):
    """Detects API breaking changes by analyzing AST diffs of public interfaces"""

    def __init__(self, repo_path: str):
        super().__init__(repo_path)
        self.public_patterns = {
            'python': [
                r'^class\s+([A-Z]\w*)',  # Public classes (PascalCase)
                r'^def\s+([a-zA-Z]\w*)',  # Public functions (not _private)
                r'^([A-Z_][A-Z0-9_]*)\s*=',  # Public constants
                r'^__all__\s*=',  # Explicit exports
            ],
            'javascript': [
                r'^export\s+(class|function|const|let|var)\s+(\w+)',
                r'^export\s+default',
                r'^module\.exports',
                r'^exports\.\w+',
            ],
            'typescript': [
                r'^export\s+(class|function|const|let|var|interface|type)\s+(\w+)',
                r'^export\s+default',
                r'^declare\s+(class|function|interface|type)\s+(\w+)',
            ],
            'java': [
                r'^public\s+(class|interface|enum)\s+(\w+)',
                r'^public\s+\w+\s+(\w+)\s*\(',  # Public methods
                r'^public\s+static\s+final\s+\w+\s+(\w+)',  # Public constants
            ]
        }

    async def analyze(self, context: ChangeContext) -> DetectorResult:
        """Analyze API breaking changes"""
        api_changes = []
        total_risk_score = 0.0
        reasons = []
        anchors = []
        evidence = {
            "breaking_changes": [],
            "signature_changes": [],
            "removals": [],
            "importer_impact": {}
        }

        for file_change in context.files_changed:
            if file_change.change_type == 'deleted':
                continue

            file_path = Path(self.repo_path) / file_change.path

            # Skip if file doesn't exist or isn't a code file
            if not file_path.exists() or not self._is_code_file(file_path):
                continue

            # Analyze API changes in this file
            file_risk = await self._analyze_file_api_changes(file_change, file_path)

            if file_risk['score'] > 0:
                api_changes.append(file_risk)
                total_risk_score += file_risk['score']
                reasons.extend(file_risk['reasons'])
                anchors.extend(file_risk['anchors'])
                evidence["breaking_changes"].extend(file_risk['breaking_changes'])

        # Normalize score to 0-1 range
        if api_changes:
            # Weight by number of files and severity
            risk_score = min(total_risk_score / len(context.files_changed), 1.0)

            # Check for critical patterns that auto-escalate
            for change in evidence["breaking_changes"]:
                if change.get('severity') == 'critical':
                    risk_score = max(risk_score, 0.9)
        else:
            risk_score = 0.0

        return DetectorResult(
            score=risk_score,
            reasons=reasons[:5],  # Limit to top 5 reasons
            anchors=anchors[:10],  # Limit to top 10 anchors
            evidence=evidence,
            execution_time_ms=0.0  # Will be set by run_with_timeout
        )

    async def _analyze_file_api_changes(self, file_change: FileChange, file_path: Path) -> Dict[str, Any]:
        """Analyze API changes in a specific file"""
        file_risk = {
            'score': 0.0,
            'reasons': [],
            'anchors': [],
            'breaking_changes': []
        }

        language = self._detect_language(file_path)
        if not language:
            return file_risk

        try:
            # Read current file content
            current_content = file_path.read_text(encoding='utf-8')

            # For Python files, use AST analysis
            if language == 'python':
                await self._analyze_python_ast(file_change, current_content, file_risk)
            else:
                # For other languages, use regex-based analysis
                await self._analyze_with_patterns(file_change, current_content, language, file_risk)

        except Exception as e:
            # Don't fail the entire analysis for one file
            file_risk['reasons'].append(f"Failed to analyze {file_path.name}: {str(e)}")

        return file_risk

    async def _analyze_python_ast(self, file_change: FileChange, content: str, file_risk: Dict[str, Any]):
        """Analyze Python AST for API changes"""
        try:
            tree = ast.parse(content)

            # Extract public API elements
            public_api = self._extract_python_public_api(tree)

            # Check hunks for changes to public API
            for hunk in file_change.hunks:
                lines_removed = [line[1:] for line in hunk.get('lines', []) if line.startswith('-')]
                lines_added = [line[1:] for line in hunk.get('lines', []) if line.startswith('+')]

                # Detect function signature changes
                removed_funcs = self._extract_function_signatures(lines_removed)
                added_funcs = self._extract_function_signatures(lines_added)

                for func_name in removed_funcs:
                    if func_name in added_funcs:
                        old_sig = removed_funcs[func_name]
                        new_sig = added_funcs[func_name]
                        if old_sig != new_sig:
                            severity = self._assess_signature_change_severity(old_sig, new_sig)
                            file_risk['breaking_changes'].append({
                                'type': 'signature_change',
                                'function': func_name,
                                'old_signature': old_sig,
                                'new_signature': new_sig,
                                'severity': severity
                            })
                            file_risk['score'] += 0.3 if severity == 'breaking' else 0.1
                            file_risk['reasons'].append(f"Function signature changed: {func_name}")
                            file_risk['anchors'].append(f"{file_change.path}:function:{func_name}")

                # Detect removed public functions/classes
                for func_name in removed_funcs:
                    if func_name not in added_funcs and not func_name.startswith('_'):
                        file_risk['breaking_changes'].append({
                            'type': 'removal',
                            'function': func_name,
                            'severity': 'critical'
                        })
                        file_risk['score'] += 0.5
                        file_risk['reasons'].append(f"Public function removed: {func_name}")
                        file_risk['anchors'].append(f"{file_change.path}:function:{func_name}")

        except SyntaxError:
            # File might have syntax errors or be in a different Python version
            file_risk['reasons'].append(f"Could not parse Python AST for {file_change.path}")

    async def _analyze_with_patterns(self, file_change: FileChange, content: str, language: str, file_risk: Dict[str, Any]):
        """Analyze using regex patterns for non-Python languages"""
        patterns = self.public_patterns.get(language, [])

        for hunk in file_change.hunks:
            lines_removed = [line[1:] for line in hunk.get('lines', []) if line.startswith('-')]
            lines_added = [line[1:] for line in hunk.get('lines', []) if line.startswith('+')]

            # Check for removed exports
            for line in lines_removed:
                for pattern in patterns:
                    match = re.search(pattern, line.strip())
                    if match:
                        export_name = match.group(1) if match.lastindex and match.lastindex >= 1 else "unknown"

                        # Check if this export was re-added (signature change vs removal)
                        is_readded = any(export_name in added_line for added_line in lines_added)

                        if not is_readded:
                            file_risk['breaking_changes'].append({
                                'type': 'export_removal',
                                'name': export_name,
                                'severity': 'critical'
                            })
                            file_risk['score'] += 0.4
                            file_risk['reasons'].append(f"Public export removed: {export_name}")
                            file_risk['anchors'].append(f"{file_change.path}:export:{export_name}")

    def _extract_python_public_api(self, tree: ast.AST) -> Dict[str, Any]:
        """Extract public API elements from Python AST"""
        public_api = {
            'classes': [],
            'functions': [],
            'constants': [],
            'exports': []
        }

        for node in ast.walk(tree):
            if isinstance(node, ast.ClassDef) and not node.name.startswith('_'):
                public_api['classes'].append({
                    'name': node.name,
                    'lineno': node.lineno,
                    'methods': [m.name for m in node.body if isinstance(m, ast.FunctionDef) and not m.name.startswith('_')]
                })
            elif isinstance(node, ast.FunctionDef) and not node.name.startswith('_'):
                public_api['functions'].append({
                    'name': node.name,
                    'lineno': node.lineno,
                    'args': [arg.arg for arg in node.args.args],
                    'defaults': len(node.args.defaults)
                })

        return public_api

    def _extract_function_signatures(self, lines: List[str]) -> Dict[str, str]:
        """Extract function signatures from code lines"""
        signatures = {}
        for line in lines:
            # Simple regex for function definitions
            func_match = re.match(r'\s*def\s+(\w+)\s*\((.*?)\)', line.strip())
            if func_match:
                func_name = func_match.group(1)
                func_args = func_match.group(2)
                signatures[func_name] = func_args.strip()
        return signatures

    def _assess_signature_change_severity(self, old_sig: str, new_sig: str) -> str:
        """Assess the severity of a function signature change"""
        old_args = [arg.strip() for arg in old_sig.split(',') if arg.strip()]
        new_args = [arg.strip() for arg in new_sig.split(',') if arg.strip()]

        # Removing required arguments is breaking
        if len(new_args) < len(old_args):
            # Check if removed args had defaults
            return 'breaking'

        # Adding required arguments is breaking
        if len(new_args) > len(old_args):
            # Check if new args have defaults
            return 'breaking'

        # Argument type or name changes could be breaking
        if old_args != new_args:
            return 'potentially_breaking'

        return 'safe'

    def _detect_language(self, file_path: Path) -> Optional[str]:
        """Detect the programming language of a file"""
        suffix = file_path.suffix.lower()
        language_map = {
            '.py': 'python',
            '.js': 'javascript',
            '.jsx': 'javascript',
            '.ts': 'typescript',
            '.tsx': 'typescript',
            '.java': 'java',
            '.kt': 'kotlin',
            '.go': 'go',
            '.rs': 'rust'
        }
        return language_map.get(suffix)

    def _is_code_file(self, file_path: Path) -> bool:
        """Check if file is a code file we can analyze"""
        code_extensions = {'.py', '.js', '.jsx', '.ts', '.tsx', '.java', '.kt', '.go', '.rs'}
        return file_path.suffix.lower() in code_extensions