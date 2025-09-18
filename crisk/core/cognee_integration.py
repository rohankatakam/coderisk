"""
Cognee integration for CodeGraph and risk analysis
"""

import asyncio
import cognee
from cognee import SearchType
from cognee.api.v1.cognify.code_graph_pipeline import run_code_graph_pipeline
from cognee.modules.code import CodeGraph
from typing import List, Dict, Optional, Any
import os
from pathlib import Path


class CogneeCodeAnalyzer:
    """Integration with Cognee for code analysis and risk assessment"""

    def __init__(self, repo_path: str):
        self.repo_path = Path(repo_path).resolve()
        self.code_graph = CodeGraph()
        self._is_initialized = False

    async def initialize(self, include_docs: bool = False) -> None:
        """Initialize the code graph for the repository"""
        if self._is_initialized:
            return

        print(f"Initializing CodeGraph for {self.repo_path}...")

        # Build the code graph
        async for _ in run_code_graph_pipeline(
            str(self.repo_path),
            include_docs=include_docs,
            excluded_paths=[
                "**/node_modules/**",
                "**/dist/**",
                "**/build/**",
                "**/.git/**",
                "**/venv/**",
                "**/__pycache__/**",
                "**/coverage/**",
                "**/test_results/**"
            ],
            supported_languages=["python", "typescript", "javascript", "java", "go"]
        ):
            pass

        self._is_initialized = True
        print("CodeGraph initialization complete!")

    async def analyze_dependencies(self, file_paths: List[str]) -> Dict[str, Any]:
        """Analyze dependencies for given files"""
        if not self._is_initialized:
            await self.initialize()

        dependencies = {}
        for file_path in file_paths:
            try:
                # Use CodeGraph to get dependencies
                deps = await self.code_graph.get_dependencies([file_path])
                dependencies[file_path] = deps
            except Exception as e:
                print(f"Warning: Could not analyze dependencies for {file_path}: {e}")
                dependencies[file_path] = []

        return dependencies

    async def search_code_patterns(self, query: str, search_type: str = "code") -> List[Dict[str, Any]]:
        """Search for code patterns using Cognee"""
        if not self._is_initialized:
            await self.initialize()

        try:
            if search_type == "code":
                results = await cognee.search(
                    query_type=SearchType.CODE,
                    query_text=query
                )
            elif search_type == "chunks":
                results = await cognee.search(
                    query_type=SearchType.CHUNKS,
                    query_text=query
                )
            else:
                results = await cognee.search(
                    query_type=SearchType.FEELING_LUCKY,
                    query_text=query
                )

            return results if isinstance(results, list) else [results]
        except Exception as e:
            print(f"Warning: Search failed for '{query}': {e}")
            return []

    async def find_similar_changes(self, files: List[str], change_description: str) -> List[Dict[str, Any]]:
        """Find similar historical changes"""
        query = f"Find similar changes to files: {', '.join(files)} with description: {change_description}"
        return await self.search_code_patterns(query, "chunks")

    async def get_file_impact_radius(self, file_paths: List[str]) -> Dict[str, List[str]]:
        """Calculate impact radius for changed files"""
        impact_map = {}

        for file_path in file_paths:
            # Search for files that import or depend on this file
            query = f"Find files that import or depend on {file_path}"
            impacted_files = await self.search_code_patterns(query, "code")

            # Extract file paths from results
            file_list = []
            for result in impacted_files:
                if isinstance(result, dict) and 'file_path' in result:
                    file_list.append(result['file_path'])
                elif isinstance(result, str):
                    file_list.append(result)

            impact_map[file_path] = file_list

        return impact_map

    async def analyze_api_surface(self, file_paths: List[str]) -> Dict[str, Any]:
        """Analyze API surface changes"""
        api_analysis = {
            "public_functions": [],
            "exported_classes": [],
            "breaking_changes": [],
            "importers": {}
        }

        for file_path in file_paths:
            # Find public functions and classes
            query = f"Find all public functions and classes in {file_path}"
            api_elements = await self.search_code_patterns(query, "code")

            # Find who imports from this file
            import_query = f"Find all files that import from {file_path}"
            importers = await self.search_code_patterns(import_query, "code")

            api_analysis["importers"][file_path] = importers

        return api_analysis

    async def detect_incident_patterns(self, change_context: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Detect patterns similar to past incidents"""
        # Search for incident-related patterns
        files_query = f"Find incidents related to files: {', '.join(change_context.get('files', []))}"
        incident_results = await self.search_code_patterns(files_query, "chunks")

        # Look for specific risk patterns
        risk_patterns = [
            "authentication changes",
            "database migration",
            "configuration changes",
            "error handling modifications",
            "concurrent code changes"
        ]

        pattern_results = []
        for pattern in risk_patterns:
            if any(pattern.lower() in str(change_context).lower() for pattern in risk_patterns):
                query = f"Find past issues with {pattern}"
                results = await self.search_code_patterns(query, "chunks")
                pattern_results.extend(results)

        return incident_results + pattern_results