"""
Merge Risk Detector

Detects overlap with upstream changes and hotspots for potential merge conflicts.
Analyzes files with high co-change frequency and recent modifications.

Time budget: 10-20ms
"""

import re
from typing import Dict, List, Set, Optional, Any, Tuple
from pathlib import Path
from datetime import datetime, timedelta

from . import BaseDetector, DetectorResult, ChangeContext, FileChange, detector_registry


@detector_registry.register
class MergeRiskDetector(BaseDetector):
    """Detects potential merge conflicts and overlap with upstream changes"""

    def __init__(self, repo_path: str):
        super().__init__(repo_path)

        # Files that are prone to merge conflicts
        self.conflict_prone_patterns = [
            r'package\.json$',
            r'requirements\.txt$',
            r'Cargo\.toml$',
            r'go\.mod$',
            r'pom\.xml$',
            r'.*\.lock$',
            r'CHANGELOG',
            r'VERSION',
            r'.*\.md$',  # Documentation files
            r'config/.*',
            r'settings.*',
        ]

        # Hotspot indicators (files that change frequently)
        self.hotspot_indicators = [
            'index', 'main', 'core', 'common', 'shared',
            'utils', 'helpers', 'constants', 'config'
        ]

    async def analyze(self, context: ChangeContext) -> DetectorResult:
        """Analyze merge conflict risk"""
        merge_risks = []
        total_risk_score = 0.0
        reasons = []
        anchors = []
        evidence = {
            "conflict_prone_files": [],
            "hotspot_overlaps": [],
            "recent_upstream_changes": [],
            "high_churn_files": [],
            "potential_conflicts": []
        }

        # Get recent commits for context
        recent_commits = await self._get_recent_commits()

        for file_change in context.files_changed:
            # Analyze merge risk for this file
            file_risk = await self._analyze_file_merge_risk(file_change, recent_commits)

            if file_risk['score'] > 0:
                merge_risks.append(file_risk)
                total_risk_score += file_risk['score']
                reasons.extend(file_risk['reasons'])
                anchors.extend(file_risk['anchors'])

                # Collect evidence
                if file_risk.get('is_conflict_prone'):
                    evidence["conflict_prone_files"].append(file_change.path)

                if file_risk.get('is_hotspot'):
                    evidence["hotspot_overlaps"].append(file_change.path)

                if file_risk.get('recent_changes'):
                    evidence["recent_upstream_changes"].extend(file_risk['recent_changes'])

        # Check for cross-file dependencies and conflicts
        await self._check_cross_file_conflicts(context.files_changed, evidence, total_risk_score, reasons)

        # Normalize score
        if merge_risks:
            risk_score = min(total_risk_score / len(context.files_changed), 1.0)

            # Weight by number of conflict-prone files
            conflict_factor = len(evidence["conflict_prone_files"]) / max(len(context.files_changed), 1)
            risk_score *= (1 + conflict_factor)

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

    async def _analyze_file_merge_risk(self, file_change: FileChange, recent_commits: List[Dict]) -> Dict[str, Any]:
        """Analyze merge risk for a specific file"""
        file_risk = {
            'score': 0.0,
            'reasons': [],
            'anchors': [],
            'is_conflict_prone': False,
            'is_hotspot': False,
            'recent_changes': []
        }

        # Check if file is conflict-prone
        if self._is_conflict_prone_file(file_change.path):
            file_risk['is_conflict_prone'] = True
            file_risk['score'] += 0.4
            file_risk['reasons'].append(f"Conflict-prone file: {file_change.path}")
            file_risk['anchors'].append(f"{file_change.path}:conflict_prone")

        # Check if file is a hotspot
        if self._is_hotspot_file(file_change.path):
            file_risk['is_hotspot'] = True
            file_risk['score'] += 0.3
            file_risk['reasons'].append(f"Hotspot file: {file_change.path}")

        # Check for recent upstream changes to same file
        recent_file_changes = [
            commit for commit in recent_commits
            if file_change.path in commit.get('files', [])
        ]

        if recent_file_changes:
            file_risk['recent_changes'] = recent_file_changes[:3]  # Keep top 3
            file_risk['score'] += min(0.5, len(recent_file_changes) * 0.1)
            file_risk['reasons'].append(f"Recent upstream changes to {file_change.path}")

        # Analyze change patterns for conflict likelihood
        conflict_risk = await self._assess_change_conflict_risk(file_change)
        file_risk['score'] += conflict_risk['score']
        file_risk['reasons'].extend(conflict_risk['reasons'])

        return file_risk

    async def _assess_change_conflict_risk(self, file_change: FileChange) -> Dict[str, Any]:
        """Assess conflict risk based on the nature of changes"""
        conflict_risk = {
            'score': 0.0,
            'reasons': []
        }

        try:
            file_path = Path(self.repo_path) / file_change.path

            if not file_path.exists():
                return conflict_risk

            # For specific file types, check for high-conflict patterns
            if file_change.path.endswith('.json'):
                await self._check_json_conflict_risk(file_change, conflict_risk)
            elif file_change.path.endswith(('.yml', '.yaml')):
                await self._check_yaml_conflict_risk(file_change, conflict_risk)
            elif any(file_change.path.endswith(ext) for ext in ['.py', '.js', '.java', '.go']):
                await self._check_code_conflict_risk(file_change, conflict_risk)

        except Exception as e:
            conflict_risk['reasons'].append(f"Failed to assess conflict risk for {file_change.path}: {str(e)}")

        return conflict_risk

    async def _check_json_conflict_risk(self, file_change: FileChange, conflict_risk: Dict[str, Any]):
        """Check JSON files for merge conflict risk"""
        # JSON files are particularly prone to conflicts in certain sections
        high_conflict_sections = [
            'dependencies', 'devDependencies', 'scripts',
            'version', 'name', 'description'
        ]

        for hunk in file_change.hunks:
            for line in hunk.get('lines', []):
                if line.startswith(('+', '-')):
                    for section in high_conflict_sections:
                        if f'"{section}"' in line:
                            conflict_risk['score'] += 0.2
                            conflict_risk['reasons'].append(f"Change to {section} in JSON file")
                            break

    async def _check_yaml_conflict_risk(self, file_change: FileChange, conflict_risk: Dict[str, Any]):
        """Check YAML files for merge conflict risk"""
        # YAML indentation makes it conflict-prone
        for hunk in file_change.hunks:
            lines_with_indentation_changes = 0

            for line in hunk.get('lines', []):
                if line.startswith(('+', '-')):
                    # Check for indentation changes
                    stripped = line[1:].lstrip()
                    if len(line[1:]) - len(stripped) != len(line[1:]) - len(line[1:].lstrip()):
                        lines_with_indentation_changes += 1

            if lines_with_indentation_changes > 2:
                conflict_risk['score'] += 0.3
                conflict_risk['reasons'].append("Multiple indentation changes in YAML")

    async def _check_code_conflict_risk(self, file_change: FileChange, conflict_risk: Dict[str, Any]):
        """Check code files for merge conflict risk"""
        # Import statements and function signatures are conflict-prone
        risky_patterns = [
            r'^(import|from|#include|using)',  # Import statements
            r'^(class|def|function|interface)',  # Declarations
            r'^(export|module\.exports)',  # Exports
        ]

        for hunk in file_change.hunks:
            for line in hunk.get('lines', []):
                if line.startswith(('+', '-')):
                    line_content = line[1:].strip()

                    for pattern in risky_patterns:
                        if re.match(pattern, line_content):
                            conflict_risk['score'] += 0.1
                            break

    async def _check_cross_file_conflicts(self, files_changed: List[FileChange], evidence: Dict[str, Any],
                                        total_risk_score: float, reasons: List[str]):
        """Check for potential conflicts across multiple files"""
        # Group files by type for cross-dependency analysis
        dependency_files = []
        config_files = []
        code_files = []

        for file_change in files_changed:
            if self._is_dependency_file(file_change.path):
                dependency_files.append(file_change.path)
            elif self._is_config_file(file_change.path):
                config_files.append(file_change.path)
            else:
                code_files.append(file_change.path)

        # Multiple dependency files = higher conflict risk
        if len(dependency_files) > 1:
            evidence["potential_conflicts"].append({
                'type': 'multiple_dependency_files',
                'files': dependency_files,
                'risk': 'high'
            })
            reasons.append(f"Multiple dependency files changed: {len(dependency_files)}")

        # Mixed config and code changes
        if config_files and code_files:
            evidence["potential_conflicts"].append({
                'type': 'mixed_config_code_changes',
                'config_files': config_files,
                'code_files': code_files[:3],  # Limit for readability
                'risk': 'medium'
            })

    async def _get_recent_commits(self) -> List[Dict]:
        """Get recent commits for conflict analysis"""
        # This is a simplified implementation
        # In practice, would use git log to get recent commits
        return []

    def _is_conflict_prone_file(self, file_path: str) -> bool:
        """Check if file is prone to merge conflicts"""
        return any(re.search(pattern, file_path, re.IGNORECASE) for pattern in self.conflict_prone_patterns)

    def _is_hotspot_file(self, file_path: str) -> bool:
        """Check if file is likely to be a hotspot (frequently changed)"""
        path_lower = file_path.lower()
        return any(indicator in path_lower for indicator in self.hotspot_indicators)

    def _is_dependency_file(self, file_path: str) -> bool:
        """Check if file is a dependency management file"""
        dependency_files = [
            'package.json', 'requirements.txt', 'Pipfile', 'poetry.lock',
            'Cargo.toml', 'go.mod', 'pom.xml', 'build.gradle'
        ]
        filename = Path(file_path).name
        return filename in dependency_files or filename.endswith('.lock')

    def _is_config_file(self, file_path: str) -> bool:
        """Check if file is a configuration file"""
        config_patterns = [
            r'config/', r'settings/', r'\.env', r'\.ini$', r'\.conf$',
            r'\.yml$', r'\.yaml$', r'\.toml$'
        ]
        return any(re.search(pattern, file_path, re.IGNORECASE) for pattern in config_patterns)