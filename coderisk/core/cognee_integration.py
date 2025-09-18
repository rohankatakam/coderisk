"""
Cognee integration for CodeGraph and risk analysis
Enhanced version with full ingestion pipeline integration
"""

import asyncio
import cognee
from cognee import SearchType
from cognee.api.v1.cognify.code_graph_pipeline import run_code_graph_pipeline
from cognee.modules.code import CodeGraph
from typing import List, Dict, Optional, Any
import os
from pathlib import Path
import structlog

# Import the new ingestion components
from ..ingestion.git_history_extractor import GitHistoryExtractor
from ..ingestion.cognee_processor import CogneeKnowledgeProcessor
from ..ingestion.data_models import (
    CommitDataPoint,
    FileChangeDataPoint,
    DeveloperDataPoint,
    IncidentDataPoint,
)

logger = structlog.get_logger(__name__)


class CogneeCodeAnalyzer:
    """Enhanced Integration with Cognee for comprehensive code analysis and risk assessment"""

    def __init__(self, repo_path: str, enable_full_ingestion: bool = True):
        self.repo_path = Path(repo_path).resolve()
        self.enable_full_ingestion = enable_full_ingestion
        self.code_graph = CodeGraph()
        self._is_initialized = False

        # New ingestion components
        self.history_extractor = GitHistoryExtractor(str(self.repo_path)) if enable_full_ingestion else None
        self.knowledge_processor = CogneeKnowledgeProcessor(str(self.repo_path)) if enable_full_ingestion else None

        # Dataset name for this repository
        self.dataset_name = f"coderisk_{self.repo_path.name}"

        logger.info("CogneeCodeAnalyzer initialized",
                   repo_path=str(self.repo_path),
                   full_ingestion=enable_full_ingestion)

    async def initialize(self, include_docs: bool = False, force_reingest: bool = False) -> None:
        """Initialize the code graph and optionally perform full repository ingestion"""
        if self._is_initialized and not force_reingest:
            return

        logger.info("Initializing Cognee analyzer", include_docs=include_docs, force_reingest=force_reingest)

        if self.enable_full_ingestion:
            # Enhanced initialization with full ingestion pipeline
            await self._initialize_with_full_ingestion(include_docs, force_reingest)
        else:
            # Legacy initialization (backward compatibility)
            await self._initialize_legacy(include_docs)

        self._is_initialized = True
        logger.info("Cognee analyzer initialization complete")

    async def _initialize_with_full_ingestion(self, include_docs: bool, force_reingest: bool) -> None:
        """Initialize with full Cognee ingestion pipeline"""
        logger.info("Starting full repository ingestion pipeline")

        # Step 1: Initialize knowledge processor
        await self.knowledge_processor.initialize()

        # Step 2: Extract repository history if needed
        if force_reingest or await self._should_reingest_data():
            logger.info("Extracting repository history")

            commits, file_changes, developers = await self.history_extractor.extract_repository_history()

            logger.info("Repository extraction complete",
                       commits=len(commits),
                       file_changes=len(file_changes),
                       developers=len(developers))

            # Step 3: Ingest data into Cognee
            await self.knowledge_processor.ingest_repository_data(
                commits=commits,
                file_changes=file_changes,
                developers=developers
            )

        # Step 4: Initialize CodeGraph for current structure
        await self._initialize_code_graph(include_docs)

    async def _initialize_legacy(self, include_docs: bool) -> None:
        """Legacy initialization for backward compatibility"""
        logger.info("Using legacy initialization (SimpleAnalyzer compatibility)")
        await self._initialize_code_graph(include_docs)

    async def _initialize_code_graph(self, include_docs: bool) -> None:
        """Initialize CodeGraph component"""
        logger.info("Initializing CodeGraph")

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

        logger.info("CodeGraph initialization complete")

    async def _should_reingest_data(self) -> bool:
        """Check if we should reingest repository data"""
        # Simple heuristic - could be enhanced with timestamp checking
        try:
            # Check if dataset exists and has recent data
            recent_data = await cognee.search(
                query_text="recent commits",
                query_type=SearchType.CHUNKS,
                dataset_name=self.dataset_name,
                top_k=1
            )
            return len(recent_data) == 0
        except:
            return True  # Reingest if we can't check

    async def analyze_dependencies(self, file_paths: List[str]) -> Dict[str, Any]:
        """Analyze dependencies for given files with enhanced Cognee integration"""
        if not self._is_initialized:
            await self.initialize()

        if self.enable_full_ingestion and self.knowledge_processor:
            # Use enhanced knowledge processor for dependency analysis
            return await self.knowledge_processor.analyze_blast_radius(file_paths)
        else:
            # Legacy dependency analysis
            dependencies = {}
            for file_path in file_paths:
                try:
                    # Use CodeGraph to get dependencies
                    deps = await self.code_graph.get_dependencies([file_path])
                    dependencies[file_path] = deps
                except Exception as e:
                    logger.warning("Could not analyze dependencies", file_path=file_path, error=str(e))
                    dependencies[file_path] = []

            return dependencies

    async def search_code_patterns(self, query: str, search_type: str = "code") -> List[Dict[str, Any]]:
        """Search for code patterns using Cognee with enhanced capabilities"""
        if not self._is_initialized:
            await self.initialize()

        try:
            # Use dataset-specific search if full ingestion is enabled
            search_kwargs = {}
            if self.enable_full_ingestion:
                search_kwargs["dataset_name"] = self.dataset_name

            if search_type == "code":
                results = await cognee.search(
                    query_type=SearchType.CODE,
                    query_text=query,
                    **search_kwargs
                )
            elif search_type == "chunks":
                results = await cognee.search(
                    query_type=SearchType.CHUNKS,
                    query_text=query,
                    **search_kwargs
                )
            elif search_type == "temporal":
                results = await cognee.search(
                    query_type=SearchType.TEMPORAL,
                    query_text=query,
                    **search_kwargs
                )
            elif search_type == "graph":
                results = await cognee.search(
                    query_type=SearchType.GRAPH_COMPLETION,
                    query_text=query,
                    **search_kwargs
                )
            else:
                results = await cognee.search(
                    query_type=SearchType.FEELING_LUCKY,
                    query_text=query,
                    **search_kwargs
                )

            return results if isinstance(results, list) else [results]
        except Exception as e:
            logger.warning("Search failed", query=query, search_type=search_type, error=str(e))
            return []

    async def find_similar_changes(self, files: List[str], change_description: str) -> List[Dict[str, Any]]:
        """Find similar historical changes with enhanced temporal search"""
        if self.enable_full_ingestion and self.knowledge_processor:
            # Use enhanced temporal search
            query = f"Find similar changes to files: {', '.join(files)} with description: {change_description}"
            return await self.search_code_patterns(query, "temporal")
        else:
            # Legacy search
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

    # Enhanced Methods for Full Cognee Integration
    async def analyze_cochange_patterns(self, files: List[str]) -> Dict[str, Any]:
        """Analyze co-change patterns using enhanced Cognee capabilities"""
        if not self.enable_full_ingestion or not self.knowledge_processor:
            logger.warning("Co-change analysis requires full ingestion mode")
            return {"cochange_patterns": [], "risk_score": 0.0}

        return await self.knowledge_processor.analyze_cochange_patterns(files)

    async def find_incident_adjacency(self, changes: Dict[str, Any]) -> Dict[str, Any]:
        """Find similar incidents using multi-modal search"""
        if not self.enable_full_ingestion or not self.knowledge_processor:
            logger.warning("Incident adjacency analysis requires full ingestion mode")
            return {"similar_incidents": [], "fused_results": []}

        return await self.knowledge_processor.find_incident_adjacency(changes)

    async def search_security_patterns(self, code_snippet: str, file_path: str) -> Dict[str, Any]:
        """Search for security vulnerabilities using pattern matching"""
        if not self.enable_full_ingestion or not self.knowledge_processor:
            # Fallback to simple search
            query = f"security vulnerabilities in {file_path}: {code_snippet[:200]}"
            results = await self.search_code_patterns(query, "chunks")
            return {"vulnerabilities": results, "confidence": 0.5}

        return await self.knowledge_processor.search_security_patterns(code_snippet, file_path)

    async def run_full_risk_analysis(self, diff_data: Dict[str, Any]) -> Dict[str, Any]:
        """Run comprehensive risk analysis using the full Cognee pipeline"""
        if not self.enable_full_ingestion or not self.knowledge_processor:
            logger.warning("Full risk analysis requires full ingestion mode")
            return {"error": "Full ingestion mode required"}

        return await self.knowledge_processor.run_risk_analysis_pipeline(diff_data)

    async def provide_feedback(self, query_text: str, feedback_type: str = "positive") -> None:
        """Provide feedback to improve future predictions"""
        if self.enable_full_ingestion and self.knowledge_processor:
            await self.knowledge_processor.provide_feedback(query_text, feedback_type)
        else:
            logger.warning("Feedback requires full ingestion mode")

    async def update_with_new_data(self,
                                  commits: Optional[List[CommitDataPoint]] = None,
                                  incidents: Optional[List[IncidentDataPoint]] = None) -> None:
        """Update knowledge base with new data incrementally"""
        if not self.enable_full_ingestion or not self.knowledge_processor:
            logger.warning("Incremental updates require full ingestion mode")
            return

        if commits or incidents:
            await self.knowledge_processor.ingest_repository_data(
                commits=commits or [],
                file_changes=[],
                developers=[],
                incidents=incidents
            )

    async def get_risk_insights(self, query: str) -> Dict[str, Any]:
        """Get general risk insights using natural language queries"""
        try:
            # Use different search strategies based on query type
            if "incident" in query.lower() or "bug" in query.lower():
                search_type = "temporal"
            elif "pattern" in query.lower() or "similar" in query.lower():
                search_type = "chunks"
            elif "dependency" in query.lower() or "impact" in query.lower():
                search_type = "graph"
            else:
                search_type = "feeling_lucky"

            results = await self.search_code_patterns(query, search_type)

            return {
                "query": query,
                "search_type": search_type,
                "results": results,
                "insights_count": len(results)
            }
        except Exception as e:
            logger.error("Failed to get risk insights", query=query, error=str(e))
            return {"error": str(e), "results": []}

    def get_capabilities(self) -> Dict[str, bool]:
        """Get available capabilities based on initialization mode"""
        return {
            "basic_code_analysis": True,
            "dependency_analysis": True,
            "pattern_search": True,
            "full_ingestion": self.enable_full_ingestion,
            "temporal_analysis": self.enable_full_ingestion,
            "incident_correlation": self.enable_full_ingestion,
            "security_analysis": self.enable_full_ingestion,
            "cochange_analysis": self.enable_full_ingestion,
            "feedback_learning": self.enable_full_ingestion,
            "risk_pipeline": self.enable_full_ingestion,
        }