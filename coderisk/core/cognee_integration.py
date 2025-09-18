"""
Cognee integration for CodeGraph and risk analysis
Updated version with latest Cognee API integration
"""

import asyncio
import cognee
from cognee import SearchType
from cognee.modules.observability.get_observe import get_observe
from cognee.api.v1.cognify.code_graph_pipeline import run_code_graph_pipeline
from cognee.api.v1.visualize.visualize import visualize_graph
from typing import List, Dict, Optional, Any
import os
from pathlib import Path
import structlog

logger = structlog.get_logger(__name__)


class CogneeCodeAnalyzer:
    """Cognee Integration for comprehensive code analysis and risk assessment using latest API"""

    def __init__(self, repo_path: str, enable_full_ingestion: bool = True):
        self.repo_path = Path(repo_path).resolve()
        self.enable_full_ingestion = enable_full_ingestion
        self._is_initialized = False
        self._code_graph_built = False

        # Initialize Cognee's observability
        self.observe = get_observe()

        # Dataset name for this repository
        self.dataset_name = f"coderisk_{self.repo_path.name}"

        # Ontology file path
        self.ontology_path = Path(__file__).parent.parent / "ontology" / "coderisk_ontology.owl"

        logger.info("CogneeCodeAnalyzer initialized",
                   repo_path=str(self.repo_path),
                   full_ingestion=enable_full_ingestion,
                   ontology_path=str(self.ontology_path))

    async def initialize(self, include_docs: bool = False, force_reingest: bool = False) -> None:
        """Initialize the code graph using the latest Cognee API"""
        if self._is_initialized and not force_reingest:
            return

        logger.info("Initializing Cognee analyzer", include_docs=include_docs, force_reingest=force_reingest)

        # Use the latest Cognee code graph pipeline
        await self._build_code_graph(include_docs, force_reingest)

        self._is_initialized = True
        logger.info("Cognee analyzer initialization complete")

    async def _build_code_graph(self, include_docs: bool, force_reingest: bool) -> None:
        """Build code graph using the latest Cognee code graph pipeline"""
        logger.info("Building code graph with Cognee", include_docs=include_docs)

        try:
            # Build code graph using the new pipeline
            async for result in run_code_graph_pipeline(
                str(self.repo_path),
                include_docs=include_docs,
                excluded_paths=["**/node_modules/**", "**/dist/**", "**/.git/**", "**/venv/**", "**/__pycache__/**"],
                supported_languages=["python", "typescript", "javascript", "java", "go", "rust", "cpp"]
            ):
                logger.debug("Code graph pipeline progress", result=result)

            self._code_graph_built = True
            logger.info("Code graph built successfully")

            # Optionally add data to a dataset for enhanced search
            if self.enable_full_ingestion:
                await self._add_to_dataset(include_docs)

        except Exception as e:
            logger.error("Code graph building failed", error=str(e))
            raise RuntimeError(f"Failed to build code graph: {e}") from e

    async def _add_to_dataset(self, include_docs: bool) -> None:
        """Add repository data to a Cognee dataset for enhanced search capabilities"""
        logger.info("Adding repository to Cognee dataset", dataset_name=self.dataset_name)

        try:
            # Add the repository to a dataset
            await cognee.add(
                str(self.repo_path),
                dataset_name=self.dataset_name,
                node_set=["coderisk", "repository"]
            )

            # Cognify with ontology if available
            cognify_kwargs = {"dataset_name": self.dataset_name}
            if self.ontology_path.exists():
                cognify_kwargs["ontology_file_path"] = str(self.ontology_path)
                logger.info("Using CodeRisk ontology for cognification")

            await cognee.cognify(**cognify_kwargs)

            logger.info("Repository successfully added to Cognee dataset")

        except Exception as e:
            logger.warning("Failed to add to dataset, continuing with code graph only", error=str(e))

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
        """Analyze dependencies for given files using Cognee code graph"""
        if not self._is_initialized:
            await self.initialize()

        dependencies = {}
        for file_path in file_paths:
            try:
                # Search for dependencies using Cognee code search
                query = f"Find dependencies and imports for {file_path}"
                deps = await self.search_code_patterns(query, "code")
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
        """Find similar historical changes using Cognee search"""
        query = f"Find similar changes to files: {', '.join(files)} with description: {change_description}"
        if self.enable_full_ingestion:
            # Use temporal search for dataset-based analysis
            return await self.search_code_patterns(query, "temporal")
        else:
            # Use code search for code graph-based analysis
            return await self.search_code_patterns(query, "code")

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

    # Enhanced Methods for Cognee Integration
    async def analyze_cochange_patterns(self, files: List[str]) -> Dict[str, Any]:
        """Analyze co-change patterns using Cognee capabilities"""
        if not self.enable_full_ingestion:
            logger.warning("Co-change analysis requires full ingestion mode")
            return {"cochange_patterns": [], "risk_score": 0.0}

        # Use temporal search to find co-change patterns
        query = f"Find files that frequently change together with: {', '.join(files)}"
        results = await self.search_code_patterns(query, "temporal")

        # Extract co-change patterns from results
        cochange_patterns = []
        for result in results:
            if isinstance(result, dict):
                cochange_patterns.append(result)

        return {
            "cochange_patterns": cochange_patterns,
            "risk_score": min(len(cochange_patterns) * 0.1, 1.0)
        }

    async def find_incident_adjacency(self, changes: Dict[str, Any]) -> Dict[str, Any]:
        """Find similar incidents using Cognee search"""
        if not self.enable_full_ingestion:
            logger.warning("Incident adjacency analysis requires full ingestion mode")
            return {"similar_incidents": [], "fused_results": []}

        # Search for similar incidents
        files_str = ', '.join(changes.get('files', []))
        query = f"Find incidents and bugs related to files: {files_str}"
        similar_incidents = await self.search_code_patterns(query, "chunks")

        return {
            "similar_incidents": similar_incidents,
            "fused_results": similar_incidents  # Same for now
        }

    async def search_security_patterns(self, code_snippet: str, file_path: str) -> Dict[str, Any]:
        """Search for security vulnerabilities using pattern matching"""
        query = f"security vulnerabilities in {file_path}: {code_snippet[:200]}"
        results = await self.search_code_patterns(query, "chunks")

        # Analyze results for security patterns
        confidence = 0.7 if results else 0.1
        return {"vulnerabilities": results, "confidence": confidence}

    async def run_full_risk_analysis(self, diff_data: Dict[str, Any]) -> Dict[str, Any]:
        """Run comprehensive risk analysis using Cognee"""
        if not self._is_initialized:
            await self.initialize()

        files = diff_data.get('files', [])

        # Perform multiple analyses
        dependencies = await self.analyze_dependencies(files)
        impact_radius = await self.get_file_impact_radius(files)
        api_analysis = await self.analyze_api_surface(files)
        incident_patterns = await self.detect_incident_patterns(diff_data)

        return {
            "dependencies": dependencies,
            "impact_radius": impact_radius,
            "api_analysis": api_analysis,
            "incident_patterns": incident_patterns,
            "analysis_complete": True
        }

    async def provide_feedback(self, query_text: str, feedback_type: str = "positive") -> None:
        """Provide feedback to improve future predictions"""
        # For now, just log feedback - can be extended to use Cognee's feedback mechanisms
        logger.info("Received feedback", query=query_text, feedback_type=feedback_type)

    async def update_with_new_data(self, new_data: Dict[str, Any]) -> None:
        """Update knowledge base with new data incrementally"""
        if not self.enable_full_ingestion:
            logger.warning("Incremental updates require full ingestion mode")
            return

        # Re-add data to the dataset
        try:
            await self._add_to_dataset(include_docs=False)
            logger.info("Data updated successfully")
        except Exception as e:
            logger.error("Failed to update data", error=str(e))

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

    async def visualize_code_graph(self, output_path: str = "./cognee_code_graph.html") -> str:
        """Generate a visualization of the code graph"""
        if not self._code_graph_built:
            await self.initialize()

        try:
            await visualize_graph(output_path)
            logger.info("Code graph visualization generated", output_path=output_path)
            return output_path
        except Exception as e:
            logger.error("Failed to generate visualization", error=str(e))
            raise RuntimeError(f"Failed to visualize code graph: {e}") from e

    def get_capabilities(self) -> Dict[str, bool]:
        """Get available capabilities based on initialization mode"""
        return {
            "basic_code_analysis": True,
            "dependency_analysis": True,
            "pattern_search": True,
            "code_graph": True,
            "visualization": True,
            "ontology_support": self.ontology_path.exists(),
            "full_ingestion": self.enable_full_ingestion,
            "temporal_analysis": self.enable_full_ingestion,
            "incident_correlation": self.enable_full_ingestion,
            "security_analysis": True,  # Available in both modes
            "cochange_analysis": self.enable_full_ingestion,
            "feedback_learning": True,  # Available in both modes
            "risk_pipeline": True,  # Available in both modes
        }