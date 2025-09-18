"""
Git History Extractor using PyDriller

Extracts comprehensive git repository history and converts it to structured
DataPoints for Cognee ingestion. Processes commits, files, and developer data
with temporal awareness and risk indicator detection.
"""

import re
import asyncio
from typing import List, Dict, Optional, Set, Tuple, Iterator
from datetime import datetime, timedelta
from pathlib import Path
from pydriller import Repository
from pydriller.domain.commit import Commit
import structlog

from .data_models import (
    CommitDataPoint,
    FileChangeDataPoint,
    DeveloperDataPoint,
    DependencyDataPoint,
    SecurityDataPoint,
)

logger = structlog.get_logger(__name__)


class GitHistoryExtractor:
    """
    Extracts Git repository history and converts to Cognee DataPoints

    Features:
    - 90-day rolling window for temporal analysis
    - Risk indicator detection (security, migrations, large changes)
    - Developer expertise and collaboration tracking
    - File dependency extraction
    - Security pattern detection
    """

    def __init__(self, repo_path: str, window_days: int = 90):
        self.repo_path = Path(repo_path).resolve()
        self.window_days = window_days
        self.cutoff_date = datetime.now() - timedelta(days=window_days)

        # Risk pattern detection
        self.security_patterns = self._load_security_patterns()
        self.migration_patterns = self._load_migration_patterns()
        self.config_patterns = self._load_config_patterns()

        # Developer tracking
        self.developer_stats: Dict[str, Dict] = {}
        self.file_experts: Dict[str, List[str]] = {}  # file -> [developers]

        # Dependency tracking
        self.dependencies: List[DependencyDataPoint] = []

        logger.info("GitHistoryExtractor initialized",
                   repo_path=str(self.repo_path),
                   window_days=window_days)

    def _load_security_patterns(self) -> List[Dict[str, str]]:
        """Load patterns that indicate security-related changes"""
        return [
            {"pattern": r"(?i)(auth|login|password|token|jwt|session)", "type": "authentication"},
            {"pattern": r"(?i)(sql|query|database|db)", "type": "database"},
            {"pattern": r"(?i)(crypto|encrypt|decrypt|hash|salt)", "type": "cryptography"},
            {"pattern": r"(?i)(permission|access|role|privilege)", "type": "authorization"},
            {"pattern": r"(?i)(validate|sanitize|escape|xss|csrf)", "type": "input_validation"},
            {"pattern": r"(?i)(security|vulnerability|cve|exploit)", "type": "security_explicit"},
        ]

    def _load_migration_patterns(self) -> List[str]:
        """Load patterns that indicate database migrations"""
        return [
            r"(?i)migration",
            r"(?i)schema",
            r"(?i)alter\s+table",
            r"(?i)create\s+table",
            r"(?i)drop\s+table",
            r"(?i)add\s+column",
            r"(?i)remove\s+column",
            r"migrate/",
            r"migrations/",
            r"\.sql$",
        ]

    def _load_config_patterns(self) -> List[str]:
        """Load patterns that indicate configuration changes"""
        return [
            r"\.env",
            r"config\.",
            r"settings\.",
            r"\.yml$",
            r"\.yaml$",
            r"\.json$",
            r"\.toml$",
            r"\.ini$",
            r"docker",
            r"k8s",
            r"kubernetes",
        ]

    async def extract_repository_history(self) -> Tuple[List[CommitDataPoint], List[FileChangeDataPoint], List[DeveloperDataPoint]]:
        """
        Extract complete repository history within the time window

        Returns:
            Tuple of (commits, file_changes, developers)
        """
        logger.info("Starting repository history extraction")

        commits = []
        file_changes = []

        # Process commits in chronological order
        for commit in Repository(str(self.repo_path), since=self.cutoff_date).traverse_commits():
            try:
                commit_data = await self._process_commit(commit)
                commits.append(commit_data)

                # Process file changes for this commit
                commit_file_changes = await self._process_file_changes(commit)
                file_changes.extend(commit_file_changes)

                # Update developer statistics
                self._update_developer_stats(commit)

            except Exception as e:
                logger.error("Error processing commit",
                           commit_sha=commit.hash,
                           error=str(e))
                continue

        # Generate developer profiles
        developers = await self._generate_developer_profiles()

        logger.info("Repository history extraction complete",
                   commits_count=len(commits),
                   file_changes_count=len(file_changes),
                   developers_count=len(developers))

        return commits, file_changes, developers

    async def _process_commit(self, commit: Commit) -> CommitDataPoint:
        """Process a single commit into a CommitDataPoint"""

        # Detect risk indicators
        is_revert = self._is_revert_commit(commit.msg)
        is_hotfix = self._is_hotfix_commit(commit.msg, commit.branches)
        is_security_fix = self._is_security_commit(commit.msg)
        has_db_migration = self._has_db_migration(commit)
        touches_auth = self._touches_auth(commit)
        touches_config = self._touches_config(commit)

        # Calculate change size
        total_lines = sum(mod.added_lines + mod.deleted_lines for mod in commit.modified_files)
        large_change = total_lines > 200

        return CommitDataPoint(
            sha=commit.hash,
            message=commit.msg,
            timestamp=commit.committer_date,
            author=commit.author.name,
            author_email=commit.author.email,
            files_changed=[mod.filename for mod in commit.modified_files if mod.filename],
            lines_added=sum(mod.added_lines for mod in commit.modified_files),
            lines_deleted=sum(mod.deleted_lines for mod in commit.modified_files),
            is_merge=commit.merge,
            is_revert=is_revert,
            is_hotfix=is_hotfix,
            is_security_fix=is_security_fix,
            branch=commit.branches[0] if commit.branches else None,
            parents=[parent for parent in commit.parents] if commit.parents else [],
            has_db_migration=has_db_migration,
            touches_auth=touches_auth,
            touches_config=touches_config,
            large_change=large_change,
        )

    async def _process_file_changes(self, commit: Commit) -> List[FileChangeDataPoint]:
        """Process file changes for a commit"""
        file_changes = []

        for mod in commit.modified_files:
            if not mod.filename:
                continue

            # Determine change type
            change_type = "modified"
            if mod.change_type.name == "ADD":
                change_type = "added"
            elif mod.change_type.name == "DELETE":
                change_type = "deleted"
            elif mod.change_type.name == "RENAME":
                change_type = "renamed"

            # Extract code structure changes
            functions_added, functions_modified, functions_deleted = self._extract_function_changes(mod)
            classes_added, classes_modified, classes_deleted = self._extract_class_changes(mod)
            imports_added, imports_removed = self._extract_import_changes(mod)

            # Determine file type
            language = self._detect_language(mod.filename)
            is_test_file = self._is_test_file(mod.filename)
            is_config_file = self._is_config_file(mod.filename)
            is_migration_file = self._is_migration_file(mod.filename)

            file_change = FileChangeDataPoint(
                file_path=mod.filename,
                commit_sha=commit.hash,
                change_type=change_type,
                lines_added=mod.added_lines,
                lines_deleted=mod.deleted_lines,
                complexity_score=mod.complexity if hasattr(mod, 'complexity') else None,
                functions_added=functions_added,
                functions_modified=functions_modified,
                functions_deleted=functions_deleted,
                classes_added=classes_added,
                classes_modified=classes_modified,
                classes_deleted=classes_deleted,
                imports_added=imports_added,
                imports_removed=imports_removed,
                language=language,
                is_test_file=is_test_file,
                is_config_file=is_config_file,
                is_migration_file=is_migration_file,
                diff_content=mod.diff[:10000] if mod.diff else "",  # Limit diff size
            )

            file_changes.append(file_change)

        return file_changes

    def _update_developer_stats(self, commit: Commit) -> None:
        """Update developer statistics for collaboration tracking"""
        email = commit.author.email
        name = commit.author.name

        if email not in self.developer_stats:
            self.developer_stats[email] = {
                "name": name,
                "total_commits": 0,
                "files_touched": set(),
                "collaborators": set(),
                "languages": set(),
                "recent_commits": [],
            }

        stats = self.developer_stats[email]
        stats["total_commits"] += 1
        stats["recent_commits"].append(commit.committer_date)

        # Track files and languages
        for mod in commit.modified_files:
            if mod.filename:
                stats["files_touched"].add(mod.filename)
                lang = self._detect_language(mod.filename)
                if lang:
                    stats["languages"].add(lang)

                # Track file expertise
                if mod.filename not in self.file_experts:
                    self.file_experts[mod.filename] = []
                if email not in self.file_experts[mod.filename]:
                    self.file_experts[mod.filename].append(email)

    async def _generate_developer_profiles(self) -> List[DeveloperDataPoint]:
        """Generate developer profiles from collected statistics"""
        developers = []
        now = datetime.now()
        thirty_days_ago = now - timedelta(days=30)
        ninety_days_ago = now - timedelta(days=90)

        for email, stats in self.developer_stats.items():
            # Calculate recent activity
            recent_commits = [d for d in stats["recent_commits"] if d >= thirty_days_ago]
            commits_90_days = [d for d in stats["recent_commits"] if d >= ninety_days_ago]

            # Determine expertise areas (files they've modified frequently)
            file_counts = {}
            for file_path in stats["files_touched"]:
                if file_path in self.file_experts and email in self.file_experts[file_path]:
                    # Simple heuristic: if they're in the top contributors to a file
                    file_counts[file_path] = file_counts.get(file_path, 0) + 1

            expertise_files = sorted(file_counts.keys(),
                                   key=lambda f: file_counts[f],
                                   reverse=True)[:10]

            developer = DeveloperDataPoint(
                email=email,
                name=stats["name"],
                total_commits=stats["total_commits"],
                commits_last_30_days=len(recent_commits),
                commits_last_90_days=len(commits_90_days),
                expertise_files=expertise_files,
                expertise_languages=list(stats["languages"]),
                frequent_collaborators=[],  # Would need PR data for this
                review_participation=0,  # Would need PR review data
                incident_association_count=0,  # Would need incident data
                revert_rate=0.0,  # Would need to track reverts
                bug_introduction_rate=0.0,  # Would need bug tracking
                team=None,  # Would need org data
                seniority_level="unknown",
            )

            developers.append(developer)

        return developers

    # Helper methods for risk detection
    def _is_revert_commit(self, message: str) -> bool:
        """Detect if commit is a revert"""
        return bool(re.search(r"(?i)revert|rollback", message))

    def _is_hotfix_commit(self, message: str, branches: List[str]) -> bool:
        """Detect if commit is a hotfix"""
        hotfix_indicators = [
            re.search(r"(?i)hotfix|urgent|emergency", message),
            any("hotfix" in branch.lower() for branch in branches if branch),
            re.search(r"(?i)prod|production", message),
        ]
        return any(hotfix_indicators)

    def _is_security_commit(self, message: str) -> bool:
        """Detect if commit addresses security issues"""
        for pattern_info in self.security_patterns:
            if re.search(pattern_info["pattern"], message):
                return True
        return False

    def _has_db_migration(self, commit: Commit) -> bool:
        """Detect if commit includes database migrations"""
        for mod in commit.modified_files:
            if not mod.filename:
                continue
            for pattern in self.migration_patterns:
                if re.search(pattern, mod.filename) or re.search(pattern, commit.msg):
                    return True
        return False

    def _touches_auth(self, commit: Commit) -> bool:
        """Detect if commit touches authentication/authorization code"""
        auth_patterns = [p for p in self.security_patterns
                        if p["type"] in ["authentication", "authorization"]]

        for mod in commit.modified_files:
            if not mod.filename:
                continue
            for pattern_info in auth_patterns:
                if re.search(pattern_info["pattern"], mod.filename):
                    return True
                if mod.diff and re.search(pattern_info["pattern"], mod.diff):
                    return True
        return False

    def _touches_config(self, commit: Commit) -> bool:
        """Detect if commit touches configuration files"""
        for mod in commit.modified_files:
            if not mod.filename:
                continue
            for pattern in self.config_patterns:
                if re.search(pattern, mod.filename):
                    return True
        return False

    # Helper methods for code structure extraction
    def _extract_function_changes(self, modification) -> Tuple[List[str], List[str], List[str]]:
        """Extract function-level changes (simplified implementation)"""
        added, modified, deleted = [], [], []

        if not modification.diff:
            return added, modified, deleted

        # Simple regex-based function detection (would need language-specific parsers for production)
        function_patterns = [
            r"def\s+(\w+)",  # Python
            r"function\s+(\w+)",  # JavaScript
            r"(?:public|private|protected)?\s*\w+\s+(\w+)\s*\(",  # Java/C#
        ]

        diff_lines = modification.diff.split('\n')
        for line in diff_lines:
            if line.startswith('+'):
                for pattern in function_patterns:
                    match = re.search(pattern, line)
                    if match:
                        added.append(match.group(1))
            elif line.startswith('-'):
                for pattern in function_patterns:
                    match = re.search(pattern, line)
                    if match:
                        deleted.append(match.group(1))

        # Modified functions are those that appear in both added and deleted
        modified = list(set(added) & set(deleted))
        added = [f for f in added if f not in modified]
        deleted = [f for f in deleted if f not in modified]

        return added, modified, deleted

    def _extract_class_changes(self, modification) -> Tuple[List[str], List[str], List[str]]:
        """Extract class-level changes (simplified implementation)"""
        added, modified, deleted = [], [], []

        if not modification.diff:
            return added, modified, deleted

        class_patterns = [
            r"class\s+(\w+)",  # Python, Java, C#
            r"interface\s+(\w+)",  # Java, TypeScript
        ]

        diff_lines = modification.diff.split('\n')
        for line in diff_lines:
            if line.startswith('+'):
                for pattern in class_patterns:
                    match = re.search(pattern, line)
                    if match:
                        added.append(match.group(1))
            elif line.startswith('-'):
                for pattern in class_patterns:
                    match = re.search(pattern, line)
                    if match:
                        deleted.append(match.group(1))

        modified = list(set(added) & set(deleted))
        added = [c for c in added if c not in modified]
        deleted = [c for c in deleted if c not in modified]

        return added, modified, deleted

    def _extract_import_changes(self, modification) -> Tuple[List[str], List[str]]:
        """Extract import/dependency changes"""
        added, removed = [], []

        if not modification.diff:
            return added, removed

        import_patterns = [
            r"import\s+(.+)",  # Python, Java, JavaScript
            r"from\s+(\S+)\s+import",  # Python
            r"require\s*\(\s*['\"](.+)['\"]\s*\)",  # Node.js
            r"#include\s*<(.+)>",  # C/C++
        ]

        diff_lines = modification.diff.split('\n')
        for line in diff_lines:
            if line.startswith('+'):
                for pattern in import_patterns:
                    match = re.search(pattern, line)
                    if match:
                        added.append(match.group(1).strip())
            elif line.startswith('-'):
                for pattern in import_patterns:
                    match = re.search(pattern, line)
                    if match:
                        removed.append(match.group(1).strip())

        return added, removed

    # File type detection helpers
    def _detect_language(self, filename: str) -> Optional[str]:
        """Detect programming language from filename"""
        ext_map = {
            '.py': 'python',
            '.js': 'javascript',
            '.ts': 'typescript',
            '.jsx': 'javascript',
            '.tsx': 'typescript',
            '.java': 'java',
            '.go': 'go',
            '.rs': 'rust',
            '.cpp': 'cpp',
            '.c': 'c',
            '.cs': 'csharp',
            '.php': 'php',
            '.rb': 'ruby',
            '.kt': 'kotlin',
            '.swift': 'swift',
        }

        ext = Path(filename).suffix.lower()
        return ext_map.get(ext)

    def _is_test_file(self, filename: str) -> bool:
        """Detect if file is a test file"""
        test_indicators = [
            'test_', '_test.', 'tests/', '/test/', 'spec_', '_spec.',
            '.test.', '.spec.', '__tests__/', 'TestCase', 'Test.java'
        ]
        return any(indicator in filename for indicator in test_indicators)

    def _is_config_file(self, filename: str) -> bool:
        """Detect if file is a configuration file"""
        for pattern in self.config_patterns:
            if re.search(pattern, filename):
                return True
        return False

    def _is_migration_file(self, filename: str) -> bool:
        """Detect if file is a database migration"""
        for pattern in self.migration_patterns:
            if re.search(pattern, filename):
                return True
        return False

    async def extract_dependencies(self) -> List[DependencyDataPoint]:
        """Extract file dependency relationships"""
        # This would require more sophisticated static analysis
        # For now, return empty list - can be enhanced with AST parsing
        logger.info("Dependency extraction not yet implemented")
        return []