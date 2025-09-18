"""
Simplified analyzer for MVP testing without Cognee dependency
"""

import os
from typing import List, Dict, Optional, Any
from pathlib import Path


class SimpleCodeAnalyzer:
    """Simplified code analyzer for MVP testing"""

    def __init__(self, repo_path: str):
        self.repo_path = Path(repo_path).resolve()
        self._is_initialized = False

    async def initialize(self, include_docs: bool = False) -> None:
        """Initialize the analyzer"""
        print(f"Initializing simple analyzer for {self.repo_path}...")
        self._is_initialized = True
        print("Simple analyzer initialization complete!")

    async def analyze_dependencies(self, file_paths: List[str]) -> Dict[str, Any]:
        """Simple dependency analysis based on file patterns"""
        dependencies = {}
        for file_path in file_paths:
            deps = []
            try:
                # Simple heuristic: find imports in Python files
                if file_path.endswith('.py'):
                    full_path = self.repo_path / file_path
                    if full_path.exists():
                        with open(full_path, 'r', encoding='utf-8') as f:
                            for line in f:
                                line = line.strip()
                                if line.startswith('import ') or line.startswith('from '):
                                    deps.append(line)
                dependencies[file_path] = deps[:10]  # Limit to first 10
            except Exception as e:
                print(f"Warning: Could not analyze dependencies for {file_path}: {e}")
                dependencies[file_path] = []

        return dependencies

    async def search_code_patterns(self, query: str, search_type: str = "code") -> List[Dict[str, Any]]:
        """Simple pattern search based on keywords"""
        results = []

        # Simple keyword-based search
        keywords = query.lower().split()

        try:
            for file_path in self.repo_path.rglob("*.py"):
                if file_path.is_file():
                    try:
                        content = file_path.read_text(encoding='utf-8')
                        content_lower = content.lower()

                        # Check if any keywords match
                        if any(keyword in content_lower for keyword in keywords):
                            results.append({
                                "file_path": str(file_path.relative_to(self.repo_path)),
                                "content_snippet": content[:200],
                                "match_type": search_type
                            })

                            # Limit results
                            if len(results) >= 10:
                                break
                    except Exception:
                        continue
        except Exception as e:
            print(f"Warning: Search failed for '{query}': {e}")

        return results

    async def find_similar_changes(self, files: List[str], change_description: str) -> List[Dict[str, Any]]:
        """Find similar historical changes (simplified)"""
        # For MVP, return mock similar changes
        return [
            {"file": f, "similarity": 0.8, "description": "Similar change pattern"}
            for f in files[:3]
        ]

    async def get_file_impact_radius(self, file_paths: List[str]) -> Dict[str, List[str]]:
        """Calculate impact radius for changed files (simplified)"""
        impact_map = {}

        for file_path in file_paths:
            # Simple heuristic: files in same directory are related
            impacted = []
            try:
                file_dir = Path(file_path).parent
                related_files = list(self.repo_path.glob(f"{file_dir}/*.py"))

                for related_file in related_files[:5]:  # Limit to 5
                    rel_path = str(related_file.relative_to(self.repo_path))
                    if rel_path != file_path:
                        impacted.append(rel_path)

            except Exception:
                pass

            impact_map[file_path] = impacted

        return impact_map

    async def analyze_api_surface(self, file_paths: List[str]) -> Dict[str, Any]:
        """Analyze API surface changes (simplified)"""
        api_analysis = {
            "public_functions": [],
            "exported_classes": [],
            "breaking_changes": [],
            "importers": {}
        }

        for file_path in file_paths:
            # Simple heuristic: find def and class statements
            public_elements = []
            try:
                full_path = self.repo_path / file_path
                if full_path.exists() and file_path.endswith('.py'):
                    with open(full_path, 'r', encoding='utf-8') as f:
                        for line_num, line in enumerate(f, 1):
                            line = line.strip()
                            if line.startswith('def ') and not line.startswith('def _'):
                                public_elements.append(f"function at line {line_num}")
                            elif line.startswith('class '):
                                public_elements.append(f"class at line {line_num}")

            except Exception:
                pass

            api_analysis["importers"][file_path] = public_elements[:5]

        return api_analysis

    async def detect_incident_patterns(self, change_context: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Detect patterns similar to past incidents (simplified)"""
        patterns = []

        # Simple heuristics for risky patterns
        files = change_context.get('files', [])
        message = change_context.get('message', '').lower()

        if any('auth' in f.lower() for f in files):
            patterns.append({"type": "auth_change", "risk": "high"})

        if any('config' in f.lower() for f in files):
            patterns.append({"type": "config_change", "risk": "medium"})

        if 'fix' in message or 'bug' in message:
            patterns.append({"type": "bug_fix", "risk": "medium"})

        if 'migration' in message or 'upgrade' in message:
            patterns.append({"type": "migration", "risk": "high"})

        return patterns