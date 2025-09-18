"""
Cognee Knowledge Processor

Handles the core integration with Cognee for building knowledge graphs,
vector stores, and performing risk analysis queries. Implements the full
Cognee pipeline for code risk assessment.
"""

import asyncio
import cognee
from cognee import SearchType, Task, Pipeline
from cognee.modules.code import CodeGraph
from cognee.infrastructure.data.models.data_point import DataPoint
from typing import List, Dict, Optional, Any, Tuple
from datetime import datetime, timedelta
from pathlib import Path
import structlog

from .data_models import (
    CommitDataPoint,
    FileChangeDataPoint,
    DeveloperDataPoint,
    IncidentDataPoint,
    SecurityDataPoint,
    RiskPatternDataPoint,
)

logger = structlog.get_logger(__name__)


class CogneeKnowledgeProcessor:
    """
    Core Cognee integration for knowledge processing and risk analysis

    Features:
    - Repository data ingestion with temporal awareness
    - Knowledge graph construction with custom pipelines
    - Vector similarity search for incident patterns
    - Custom risk analysis tasks and pipelines
    - Feedback learning for continuous improvement
    """

    def __init__(self, repo_path: str, dataset_name: str = "coderisk_repo"):
        self.repo_path = Path(repo_path).resolve()
        self.dataset_name = dataset_name
        self.code_graph = CodeGraph()
        self.is_initialized = False

        # Custom pipelines for risk analysis
        self.risk_pipeline = None
        self.ingestion_pipeline = None

        logger.info("CogneeKnowledgeProcessor initialized",
                   repo_path=str(self.repo_path),
                   dataset_name=dataset_name)

    async def initialize(self) -> None:
        """Initialize Cognee and set up custom pipelines"""
        if self.is_initialized:
            return

        logger.info("Initializing Cognee knowledge processor")

        # Initialize Cognee configuration
        await cognee.config()

        # Set up custom pipelines
        await self._setup_risk_analysis_pipeline()
        await self._setup_ingestion_pipeline()

        self.is_initialized = True
        logger.info("Cognee knowledge processor initialized successfully")

    async def ingest_repository_data(
        self,
        commits: List[CommitDataPoint],
        file_changes: List[FileChangeDataPoint],
        developers: List[DeveloperDataPoint],
        incidents: Optional[List[IncidentDataPoint]] = None,
        security_data: Optional[List[SecurityDataPoint]] = None
    ) -> None:
        """
        Ingest all repository data into Cognee with temporal awareness
        """
        if not self.is_initialized:
            await self.initialize()

        logger.info("Starting repository data ingestion",
                   commits_count=len(commits),
                   file_changes_count=len(file_changes),
                   developers_count=len(developers))

        try:
            # Phase 1: Ingest structured data points
            await self._ingest_commits(commits)
            await self._ingest_file_changes(file_changes)
            await self._ingest_developers(developers)

            if incidents:
                await self._ingest_incidents(incidents)

            if security_data:
                await self._ingest_security_data(security_data)

            # Phase 2: Process with CodeGraph for code structure
            await self._process_code_structure()

            # Phase 3: Build knowledge graph with temporal awareness
            await self._build_knowledge_graph()

            # Phase 4: Extract patterns and enrich relationships
            await self._extract_risk_patterns()

            logger.info("Repository data ingestion completed successfully")

        except Exception as e:
            logger.error("Error during repository data ingestion", error=str(e))
            raise

    async def _ingest_commits(self, commits: List[CommitDataPoint]) -> None:
        """Ingest commit data with temporal metadata"""
        logger.info("Ingesting commits", count=len(commits))

        await cognee.add(
            commits,
            dataset_name=self.dataset_name,
            metadata={
                "data_type": "commits",
                "window_days": 90,
                "temporal_field": "timestamp",
                "ingestion_time": datetime.now().isoformat()
            }
        )

    async def _ingest_file_changes(self, file_changes: List[FileChangeDataPoint]) -> None:
        """Ingest file change data"""
        logger.info("Ingesting file changes", count=len(file_changes))

        await cognee.add(
            file_changes,
            dataset_name=self.dataset_name,
            metadata={
                "data_type": "file_changes",
                "ingestion_time": datetime.now().isoformat()
            }
        )

    async def _ingest_developers(self, developers: List[DeveloperDataPoint]) -> None:
        """Ingest developer profiles"""
        logger.info("Ingesting developer profiles", count=len(developers))

        await cognee.add(
            developers,
            dataset_name=self.dataset_name,
            metadata={
                "data_type": "developers",
                "ingestion_time": datetime.now().isoformat()
            }
        )

    async def _ingest_incidents(self, incidents: List[IncidentDataPoint]) -> None:
        """Ingest incident data with temporal metadata"""
        logger.info("Ingesting incidents", count=len(incidents))

        await cognee.add(
            incidents,
            dataset_name=self.dataset_name,
            metadata={
                "data_type": "incidents",
                "temporal_field": "started_at",
                "ingestion_time": datetime.now().isoformat()
            }
        )

    async def _ingest_security_data(self, security_data: List[SecurityDataPoint]) -> None:
        """Ingest security vulnerability data"""
        logger.info("Ingesting security data", count=len(security_data))

        await cognee.add(
            security_data,
            dataset_name=self.dataset_name,
            metadata={
                "data_type": "security",
                "temporal_field": "discovered_at",
                "ingestion_time": datetime.now().isoformat()
            }
        )

    async def _process_code_structure(self) -> None:
        """Process repository with CodeGraph for structural analysis"""
        logger.info("Processing code structure with CodeGraph")

        # Use CodeGraph to analyze repository structure
        await self.code_graph.process_repository(str(self.repo_path))

        # Add CodeGraph results to the dataset
        code_structure_data = await self._extract_code_graph_data()
        if code_structure_data:
            await cognee.add(
                code_structure_data,
                dataset_name=self.dataset_name,
                metadata={
                    "data_type": "code_structure",
                    "source": "code_graph",
                    "ingestion_time": datetime.now().isoformat()
                }
            )

    async def _extract_code_graph_data(self) -> List[Dict[str, Any]]:
        """Extract structured data from CodeGraph analysis"""
        # This would extract entities and relationships from CodeGraph
        # For now, return empty list - would need to implement based on CodeGraph API
        logger.info("CodeGraph data extraction not yet implemented")
        return []

    async def _build_knowledge_graph(self) -> None:
        """Build knowledge graph with temporal awareness"""
        logger.info("Building knowledge graph with temporal awareness")

        await cognee.cognify(
            datasets=[self.dataset_name],
            temporal_cognify=True  # Enable temporal awareness
        )

    async def _extract_risk_patterns(self) -> None:
        """Extract risk patterns using memify"""
        logger.info("Extracting risk patterns")

        # Use memify to extract coding rules and risk patterns
        await cognee.memify(
            dataset=self.dataset_name,
            extract_coding_rules=True
        )

    async def _setup_risk_analysis_pipeline(self) -> None:
        """Set up custom pipeline for risk analysis"""

        @Task
        async def extract_change_context(diff_data: Dict[str, Any]) -> Dict[str, Any]:
            """Extract context from code changes"""
            return {
                "files": diff_data.get("files", []),
                "functions": diff_data.get("functions", []),
                "blast_radius": await self._calculate_initial_blast_radius(diff_data)
            }

        @Task
        async def compute_temporal_risk(change_context: Dict[str, Any]) -> Dict[str, Any]:
            """Compute time-based risk factors"""
            files = change_context.get("files", [])

            # Search for historical incidents in these files
            incident_query = f"Find incidents in files {', '.join(files)} in last 90 days"
            historical_risks = await cognee.search(
                query_text=incident_query,
                query_type=SearchType.TEMPORAL,
                dataset_name=self.dataset_name,
                top_k=20
            )

            return {
                "change_context": change_context,
                "historical_risks": historical_risks,
                "temporal_score": await self._apply_temporal_decay(historical_risks)
            }

        @Task
        async def derive_risk_patterns(temporal_risk: Dict[str, Any]) -> Dict[str, Any]:
            """Use memify to extract risk patterns"""
            # Search for derived coding rules that indicate risk
            risk_rules = await cognee.search(
                query_type=SearchType.CODING_RULES,
                query_text="List risk-related coding patterns",
                dataset_name=self.dataset_name
            )

            return {
                "temporal_risk": temporal_risk,
                "risk_patterns": risk_rules,
                "final_score": await self._calculate_composite_risk_score(temporal_risk, risk_rules)
            }

        # Create the pipeline
        self.risk_pipeline = Pipeline([
            extract_change_context,
            compute_temporal_risk,
            derive_risk_patterns
        ])

    async def _setup_ingestion_pipeline(self) -> None:
        """Set up custom pipeline for data ingestion"""

        @Task
        async def validate_data_points(data_points: List[DataPoint]) -> List[DataPoint]:
            """Validate and clean data points"""
            validated = []
            for dp in data_points:
                try:
                    # Validate required fields
                    if hasattr(dp, 'metadata') and dp.metadata:
                        validated.append(dp)
                except Exception as e:
                    logger.warning("Skipping invalid data point", error=str(e))
            return validated

        @Task
        async def enrich_with_metadata(data_points: List[DataPoint]) -> List[DataPoint]:
            """Enrich data points with additional metadata"""
            for dp in data_points:
                if hasattr(dp, 'metadata'):
                    dp.metadata.update({
                        "processed_at": datetime.now().isoformat(),
                        "processor_version": "1.0.0"
                    })
            return data_points

        @Task
        async def batch_ingest(data_points: List[DataPoint]) -> Dict[str, Any]:
            """Batch ingest data points"""
            await cognee.add(
                data_points,
                dataset_name=self.dataset_name,
                incremental=True
            )
            return {"ingested_count": len(data_points)}

        self.ingestion_pipeline = Pipeline([
            validate_data_points,
            enrich_with_metadata,
            batch_ingest
        ])

    # Risk Analysis Methods
    async def analyze_blast_radius(self, file_changes: List[str], time_window: str = "30 days") -> Dict[str, Any]:
        """Calculate blast radius using temporal graph queries"""
        if not self.is_initialized:
            await self.initialize()

        # Query impact graph with temporal constraints
        impact_query = f"Find files impacted by {', '.join(file_changes)} in last {time_window}"
        impact_graph = await cognee.search(
            query_text=impact_query,
            query_type=SearchType.TEMPORAL,
            dataset_name=self.dataset_name
        )

        # Use CodeGraph for deeper dependency analysis
        dependencies = await self.code_graph.get_dependencies(file_changes)

        return {
            "impact_graph": impact_graph,
            "dependencies": dependencies,
            "blast_radius_score": await self._compute_ppr_delta(impact_graph, dependencies)
        }

    async def analyze_cochange_patterns(self, files: List[str]) -> Dict[str, Any]:
        """Analyze co-change patterns with feedback learning"""
        if not self.is_initialized:
            await self.initialize()

        # Query for co-change patterns
        cochange_query = f"Find files that frequently change together with {', '.join(files)}"
        cochange_result = await cognee.search(
            query_text=cochange_query,
            query_type=SearchType.GRAPH_COMPLETION,
            dataset_name=self.dataset_name,
            save_interaction=True  # Enable feedback
        )

        # Apply Hawkes decay model
        risk_score = await self._apply_hawkes_decay(cochange_result)

        # Provide feedback for high-confidence detections
        if risk_score > 0.7:
            await cognee.search(
                query_text="Important co-change pattern detected",
                query_type=SearchType.FEEDBACK,
                last_k=1
            )

        return {
            "cochange_patterns": cochange_result,
            "risk_score": risk_score
        }

    async def find_incident_adjacency(self, changes: Dict[str, Any]) -> Dict[str, Any]:
        """Find similar incidents using multi-modal search"""
        if not self.is_initialized:
            await self.initialize()

        change_description = changes.get("description", "")
        affected_files = changes.get("files", [])

        # Vector similarity for semantic matching
        similar_incidents = await cognee.search(
            query_text=change_description,
            query_type=SearchType.CHUNKS,
            dataset_name=self.dataset_name,
            top_k=10
        )

        # Graph traversal for structural proximity
        graph_query = f"Find incidents linked to files: {', '.join(affected_files)}"
        graph_incidents = await cognee.search(
            query_text=graph_query,
            query_type=SearchType.GRAPH_COMPLETION,
            dataset_name=self.dataset_name
        )

        # Use "Feeling Lucky" for automatic best approach
        auto_incidents = await cognee.search(
            query_text=f"Risk analysis for {change_description}",
            query_type=SearchType.FEELING_LUCKY,
            dataset_name=self.dataset_name
        )

        # Combine results using reciprocal rank fusion
        fused_results = await self._fuse_rankings([similar_incidents, graph_incidents, auto_incidents])

        return {
            "similar_incidents": similar_incidents,
            "graph_incidents": graph_incidents,
            "auto_incidents": auto_incidents,
            "fused_results": fused_results
        }

    async def search_security_patterns(self, code_snippet: str, file_path: str) -> Dict[str, Any]:
        """Search for security vulnerabilities using pattern matching"""
        if not self.is_initialized:
            await self.initialize()

        # Multi-modal security search
        vulnerabilities = await cognee.search(
            query_text=code_snippet,
            query_type=SearchType.FEELING_LUCKY,  # Auto-selects best approach
            dataset_name=self.dataset_name
        )

        # Apply feedback if high-confidence detection
        if hasattr(vulnerabilities, 'confidence') and vulnerabilities.confidence > 0.9:
            await cognee.search(
                query_text="Critical security pattern detected",
                query_type=SearchType.FEEDBACK,
                last_k=1
            )

        return {
            "vulnerabilities": vulnerabilities,
            "confidence": getattr(vulnerabilities, 'confidence', 0.0)
        }

    async def run_risk_analysis_pipeline(self, diff_data: Dict[str, Any]) -> Dict[str, Any]:
        """Run the complete risk analysis pipeline"""
        if not self.risk_pipeline:
            await self._setup_risk_analysis_pipeline()

        return await self.risk_pipeline.run(diff_data)

    # Helper methods
    async def _calculate_initial_blast_radius(self, diff_data: Dict[str, Any]) -> float:
        """Calculate initial blast radius estimate"""
        files = diff_data.get("files", [])
        if not files:
            return 0.0

        # Simple heuristic - would be enhanced with actual dependency analysis
        return min(len(files) * 0.1, 1.0)

    async def _apply_temporal_decay(self, historical_risks: List[Any]) -> float:
        """Apply temporal decay to historical risk data"""
        if not historical_risks:
            return 0.0

        # Simple temporal decay - more recent incidents have higher weight
        now = datetime.now()
        total_score = 0.0

        for risk in historical_risks:
            if hasattr(risk, 'timestamp'):
                days_ago = (now - risk.timestamp).days
                decay_factor = max(0.1, 1.0 - (days_ago / 90.0))  # 90-day decay
                total_score += decay_factor

        return min(total_score / len(historical_risks), 1.0)

    async def _apply_hawkes_decay(self, cochange_result: Any, fast_decay: float = 0.1, slow_decay: float = 0.01) -> float:
        """Apply Hawkes process decay model for co-change analysis"""
        # Simplified Hawkes model implementation
        # In practice, this would use the actual mathematical model
        if not cochange_result:
            return 0.0

        # Placeholder implementation
        return 0.5  # Would implement actual Hawkes process

    async def _compute_ppr_delta(self, impact_graph: Any, dependencies: Any) -> float:
        """Compute PageRank delta for blast radius calculation"""
        # Placeholder for PageRank calculation
        # Would implement actual graph algorithms
        return 0.3

    async def _calculate_composite_risk_score(self, temporal_risk: Dict[str, Any], risk_patterns: Any) -> float:
        """Calculate composite risk score from multiple factors"""
        temporal_score = temporal_risk.get("temporal_score", 0.0)
        pattern_score = 0.5 if risk_patterns else 0.0  # Simplified

        return (temporal_score * 0.7) + (pattern_score * 0.3)

    async def _fuse_rankings(self, result_lists: List[List[Any]]) -> List[Any]:
        """Implement reciprocal rank fusion for combining search results"""
        # Simplified RRF implementation
        if not result_lists or not any(result_lists):
            return []

        # Return the first non-empty result list for now
        for results in result_lists:
            if results:
                return results

        return []

    # Feedback and Learning
    async def provide_feedback(self, query_text: str, feedback_type: str = "positive") -> None:
        """Provide feedback to improve future predictions"""
        if not self.is_initialized:
            await self.initialize()

        await cognee.search(
            query_text=f"{feedback_type} feedback: {query_text}",
            query_type=SearchType.FEEDBACK,
            last_k=1
        )

    async def update_risk_patterns(self, new_patterns: List[RiskPatternDataPoint]) -> None:
        """Update risk patterns based on new learning"""
        if not self.is_initialized:
            await self.initialize()

        await cognee.add(
            new_patterns,
            dataset_name=self.dataset_name,
            incremental=True,
            metadata={
                "data_type": "risk_patterns",
                "update_time": datetime.now().isoformat()
            }
        )

        # Re-run memify to incorporate new patterns
        await cognee.memify(
            dataset=self.dataset_name,
            filter_by_type="RiskSignal"
        )