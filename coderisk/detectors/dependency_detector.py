"""
Dependency Risk Detector

Analyzes changes to lockfiles and dependency manifests for risky updates.
Focuses on major version bumps, transitive dependency changes, and security issues.

Time budget: 10-30ms
"""

import json
import re
from typing import Dict, List, Set, Optional, Any, Tuple
from pathlib import Path
import semver

from . import BaseDetector, DetectorResult, ChangeContext, FileChange, detector_registry


@detector_registry.register
class DependencyRiskDetector(BaseDetector):
    """Detects risky dependency changes in lockfiles and manifests"""

    def __init__(self, repo_path: str):
        super().__init__(repo_path)
        self.lockfile_patterns = [
            'package-lock.json',
            'yarn.lock',
            'pnpm-lock.yaml',
            'poetry.lock',
            'Pipfile.lock',
            'Gemfile.lock',
            'composer.lock',
            'go.sum',
            'Cargo.lock'
        ]

        self.manifest_patterns = [
            'package.json',
            'pyproject.toml',
            'requirements.txt',
            'Pipfile',
            'Gemfile',
            'composer.json',
            'go.mod',
            'Cargo.toml'
        ]

    async def analyze(self, context: ChangeContext) -> DetectorResult:
        """Analyze dependency changes for risk"""
        dependency_risks = []
        total_risk_score = 0.0
        reasons = []
        anchors = []
        evidence = {
            "major_updates": [],
            "new_dependencies": [],
            "removed_dependencies": [],
            "transitive_changes": [],
            "security_concerns": []
        }

        for file_change in context.files_changed:
            if not self._is_dependency_file(file_change.path):
                continue

            # Analyze dependency file
            file_risk = await self._analyze_dependency_file(file_change)

            if file_risk['score'] > 0:
                dependency_risks.append(file_risk)
                total_risk_score += file_risk['score']
                reasons.extend(file_risk['reasons'])
                anchors.extend(file_risk['anchors'])

                # Merge evidence
                for key in evidence:
                    if key in file_risk:
                        evidence[key].extend(file_risk[key])

        # Normalize score
        if dependency_risks:
            risk_score = min(total_risk_score / len(dependency_risks), 1.0)

            # Auto-escalate for critical dependency issues
            if any(dep.get('severity') == 'critical' for dep in evidence.get('major_updates', [])):
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

    async def _analyze_dependency_file(self, file_change: FileChange) -> Dict[str, Any]:
        """Analyze a specific dependency file"""
        file_risk = {
            'score': 0.0,
            'reasons': [],
            'anchors': [],
            'major_updates': [],
            'new_dependencies': [],
            'removed_dependencies': []
        }

        try:
            file_path = Path(self.repo_path) / file_change.path

            if file_change.path.endswith('.json'):
                await self._analyze_json_dependencies(file_change, file_risk)
            elif file_change.path.endswith('.lock'):
                await self._analyze_lockfile(file_change, file_risk)
            elif 'requirements' in file_change.path:
                await self._analyze_requirements_txt(file_change, file_risk)
            else:
                await self._analyze_generic_dependencies(file_change, file_risk)

        except Exception as e:
            file_risk['reasons'].append(f"Failed to analyze {file_change.path}: {str(e)}")

        return file_risk

    async def _analyze_json_dependencies(self, file_change: FileChange, file_risk: Dict[str, Any]):
        """Analyze package.json or similar JSON dependency files"""
        for hunk in file_change.hunks:
            for line in hunk.get('lines', []):
                if line.startswith('+') or line.startswith('-'):
                    # Parse dependency changes
                    dep_match = re.search(r'"([^"]+)":\s*"([^"]+)"', line[1:])
                    if dep_match:
                        package_name = dep_match.group(1)
                        version = dep_match.group(2)

                        if line.startswith('+'):
                            await self._process_dependency_addition(package_name, version, file_change, file_risk)
                        elif line.startswith('-'):
                            await self._process_dependency_removal(package_name, version, file_change, file_risk)

    async def _analyze_lockfile(self, file_change: FileChange, file_risk: Dict[str, Any]):
        """Analyze lockfile changes"""
        # Look for version changes in lockfiles
        version_changes = 0
        major_version_changes = 0

        for hunk in file_change.hunks:
            for line in hunk.get('lines', []):
                if line.startswith('+') or line.startswith('-'):
                    # Count version changes
                    if re.search(r'"version":|version\s*[:=]', line):
                        version_changes += 1

                        # Check for major version changes
                        version_match = re.search(r'(\d+)\.(\d+)\.(\d+)', line)
                        if version_match and line.startswith('-'):
                            old_version = version_match.group(0)
                            # Look for corresponding + line with new version
                            for add_line in hunk.get('lines', []):
                                if add_line.startswith('+') and old_version not in add_line:
                                    new_version_match = re.search(r'(\d+)\.(\d+)\.(\d+)', add_line)
                                    if new_version_match:
                                        new_version = new_version_match.group(0)
                                        if self._is_major_version_change(old_version, new_version):
                                            major_version_changes += 1

        if version_changes > 10:
            file_risk['score'] += 0.4
            file_risk['reasons'].append(f"Large number of dependency updates ({version_changes})")
            file_risk['anchors'].append(f"{file_change.path}:mass_update")

        if major_version_changes > 0:
            file_risk['score'] += 0.3 * major_version_changes
            file_risk['reasons'].append(f"Major version changes detected ({major_version_changes})")
            file_risk['anchors'].append(f"{file_change.path}:major_versions")

    async def _analyze_requirements_txt(self, file_change: FileChange, file_risk: Dict[str, Any]):
        """Analyze requirements.txt changes"""
        for hunk in file_change.hunks:
            for line in hunk.get('lines', []):
                if line.startswith('+') or line.startswith('-'):
                    # Parse requirement lines
                    req_match = re.search(r'^[+-]([a-zA-Z0-9_-]+)([>=<~!]+)([0-9.]+)', line)
                    if req_match:
                        package_name = req_match.group(1)
                        operator = req_match.group(2)
                        version = req_match.group(3)

                        if line.startswith('+'):
                            await self._process_dependency_addition(package_name, version, file_change, file_risk)

    async def _analyze_generic_dependencies(self, file_change: FileChange, file_risk: Dict[str, Any]):
        """Generic analysis for other dependency files"""
        # Count total lines changed as a proxy for dependency churn
        lines_changed = file_change.lines_added + file_change.lines_deleted

        if lines_changed > 50:
            file_risk['score'] += 0.3
            file_risk['reasons'].append(f"Large dependency file changes ({lines_changed} lines)")
            file_risk['anchors'].append(f"{file_change.path}:large_changes")

    async def _process_dependency_addition(self, package_name: str, version: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Process a new dependency addition"""
        risk_score = 0.0

        # Check for potentially risky packages
        risky_patterns = [
            r'beta|alpha|rc|dev|snapshot',  # Pre-release versions
            r'^0\.',  # Major version 0.x.x
            r'latest|master|main'  # Unpinned versions
        ]

        for pattern in risky_patterns:
            if re.search(pattern, version, re.IGNORECASE):
                risk_score += 0.2
                file_risk['reasons'].append(f"Potentially unstable version: {package_name}@{version}")

        # Check for suspicious package names
        suspicious_patterns = [
            r'\d+',  # Numbers in package names can be typosquatting
            r'[_-]{2,}',  # Multiple consecutive separators
        ]

        for pattern in suspicious_patterns:
            if re.search(pattern, package_name):
                risk_score += 0.1
                file_risk['reasons'].append(f"Suspicious package name: {package_name}")

        if risk_score > 0:
            file_risk['score'] += risk_score
            file_risk['anchors'].append(f"{file_change.path}:new_dep:{package_name}")

        file_risk['new_dependencies'].append({
            'name': package_name,
            'version': version,
            'risk_score': risk_score
        })

    async def _process_dependency_removal(self, package_name: str, version: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Process a dependency removal"""
        # Dependency removals are generally lower risk but worth noting
        file_risk['removed_dependencies'].append({
            'name': package_name,
            'version': version
        })

    def _is_major_version_change(self, old_version: str, new_version: str) -> bool:
        """Check if version change is a major version bump"""
        try:
            old_parts = old_version.split('.')
            new_parts = new_version.split('.')

            if len(old_parts) >= 1 and len(new_parts) >= 1:
                return int(old_parts[0]) != int(new_parts[0])

        except (ValueError, IndexError):
            # If we can't parse versions, assume it might be major
            return True

        return False

    def _is_dependency_file(self, file_path: str) -> bool:
        """Check if file is a dependency-related file"""
        filename = Path(file_path).name
        return (filename in self.lockfile_patterns or
                filename in self.manifest_patterns or
                'requirements' in filename.lower())