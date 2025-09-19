"""
CYPHER query support for CodeRisk risk calculations

This module provides CYPHER queries compatible with Kuzu graph database
used by Cognee for risk assessment calculations.

Updated to use Kuzu-specific syntax and database connection management.
"""

import asyncio
from typing import List, Dict, Any, Optional, Tuple
from datetime import datetime, timedelta
import cognee
from cognee import SearchType
import structlog

from ..core.database_manager import safe_cognee_operation, db_manager

logger = structlog.get_logger(__name__)


class CypherQueryEngine:
    """
    Engine for executing CYPHER queries against Cognee's Kuzu graph database

    Updated for Kuzu-specific syntax and safe concurrent access.
    """

    def __init__(self):
        self.logger = logger

    async def execute_query(self, query: str, dataset_name: Optional[str] = None) -> List[Tuple]:
        """
        Execute a CYPHER query with safe database connection management.

        Args:
            query: Kuzu-compatible CYPHER query
            dataset_name: Optional dataset scope for the query
        """
        # Convert to Kuzu-compatible syntax
        kuzu_query = self._convert_to_kuzu_syntax(query)

        async def _execute():
            try:
                search_params = {
                    "query_type": SearchType.CYPHER,
                    "query_text": kuzu_query
                }

                if dataset_name:
                    search_params["datasets"] = [dataset_name]

                results = await cognee.search(**search_params)

                if isinstance(results, list):
                    return results
                else:
                    return [results] if results else []

            except Exception as e:
                self.logger.error(f"CYPHER query failed: {e}",
                                query=kuzu_query,
                                original_query=query)
                return []

        # Use safe database operation
        return await safe_cognee_operation(
            _execute,
            process_id=f"cypher_{hash(kuzu_query) % 10000}"
        )

    def _convert_to_kuzu_syntax(self, query: str) -> str:
        """
        Convert standard CYPHER to Kuzu-compatible syntax.

        Key differences in Kuzu:
        - No labels() function, use label() instead
        - No toString() function, use CAST as STRING
        - LIST functions have list_ prefix
        - Different function names for some operations
        """
        kuzu_query = query

        # Function replacements for Kuzu
        replacements = {
            # Function name changes
            'labels(': 'label(',
            'toString(': 'CAST(',
            'toInteger(': 'CAST(',
            'toFloat(': 'CAST(',

            # List function prefixes
            'size(': 'list_size(',
            'head(': 'list_first(',
            'tail(': 'list_tail(',
            'reverse(': 'list_reverse(',

            # String functions that need adjustment
            'split(': 'list_split(',

            # Handle CAST syntax for toString conversions
            'CAST(': 'CAST(',  # This will be handled specially below
        }

        for old, new in replacements.items():
            kuzu_query = kuzu_query.replace(old, new)

        # Handle toString() -> CAST(..., STRING) conversion
        import re

        # Pattern for toString(expression) -> CAST(expression AS STRING)
        toString_pattern = r'CAST\(([^)]+)\)'

        def toString_replacement(match):
            expression = match.group(1)
            # Check if this was originally a toString call
            if 'toString(' in query and expression in query:
                return f'CAST({expression} AS STRING)'
            return match.group(0)  # Return unchanged if not a toString conversion

        kuzu_query = re.sub(toString_pattern, toString_replacement, kuzu_query)

        # Add semicolon if missing (required in Kuzu)
        if not kuzu_query.strip().endswith(';'):
            kuzu_query += ';'

        return kuzu_query

    async def get_basic_stats(self, dataset_name: Optional[str] = None) -> Dict[str, int]:
        """Get basic graph statistics using Kuzu-compatible queries"""
        stats = {}

        # Total nodes (Kuzu-compatible)
        try:
            result = await self.execute_query(
                "MATCH (n) RETURN count(n) as total_nodes",
                dataset_name=dataset_name
            )
            stats['total_nodes'] = result[0][0] if result and result[0] else 0
        except Exception as e:
            self.logger.debug(f"Node count query failed: {e}")
            stats['total_nodes'] = 0

        # Total relationships (Kuzu-compatible)
        try:
            result = await self.execute_query(
                "MATCH ()-[r]-() RETURN count(r) as total_relationships",
                dataset_name=dataset_name
            )
            stats['total_relationships'] = result[0][0] if result and result[0] else 0
        except Exception as e:
            self.logger.debug(f"Relationship count query failed: {e}")
            stats['total_relationships'] = 0

        # Node types (Kuzu uses label() function)
        try:
            result = await self.execute_query(
                "MATCH (n) RETURN label(n) as node_type, count(n) as count",
                dataset_name=dataset_name
            )
            stats['node_types'] = {row[0]: row[1] for row in result if row and len(row) >= 2}
        except Exception as e:
            self.logger.debug(f"Node types query failed: {e}")
            stats['node_types'] = {}

        return stats

    async def find_commit_nodes(self) -> List[Dict[str, Any]]:
        """Find nodes related to commits using property search"""

        # Try different approaches to find commit data
        queries = [
            # Look for nodes with commit-related properties
            "MATCH (n) WHERE n.sha IS NOT NULL RETURN n LIMIT 10",
            "MATCH (n) WHERE n.message IS NOT NULL RETURN n LIMIT 10",
            "MATCH (n) WHERE n.author IS NOT NULL RETURN n LIMIT 10",
            "MATCH (n) WHERE n.timestamp IS NOT NULL RETURN n LIMIT 10",
        ]

        all_results = []
        for query in queries:
            try:
                results = await self.execute_query(query)
                if results:
                    all_results.extend(results)
                    self.logger.info(f"Found {len(results)} nodes", query=query)
            except Exception as e:
                self.logger.debug(f"Query failed: {e}", query=query)

        return all_results

    async def find_nodes_by_property(self, property_name: str, limit: int = 10) -> List[Tuple]:
        """Find nodes that have a specific property"""
        query = f"MATCH (n) WHERE n.{property_name} IS NOT NULL RETURN n LIMIT {limit}"
        return await self.execute_query(query)

    async def get_all_node_properties(self, limit: int = 5) -> List[Dict[str, Any]]:
        """Get sample nodes with all their properties"""
        query = f"MATCH (n) RETURN n LIMIT {limit}"
        results = await self.execute_query(query)

        # Parse results to extract property information
        parsed_results = []
        for result in results:
            if result and len(result) > 0:
                node_data = result[0]
                parsed_results.append({
                    'node_data': str(node_data),
                    'type': type(node_data).__name__
                })

        return parsed_results

    # Risk Calculation Queries

    async def calculate_commit_frequency(self, window_days: int = 30) -> Dict[str, Any]:
        """Calculate commit frequency for ΔDBR (Delta Defect Bug Rate)"""

        # Since we can't easily filter by date in Kuzu without knowing exact schema,
        # we'll count all commits and approximate

        stats = {}

        # Try to find commit-related nodes
        commit_queries = [
            "MATCH (n) WHERE n.sha IS NOT NULL RETURN count(n) as commit_count",
            "MATCH (n) WHERE n.message IS NOT NULL RETURN count(n) as message_count",
            "MATCH (n) WHERE n.author IS NOT NULL RETURN count(n) as author_count",
        ]

        for query in commit_queries:
            try:
                result = await self.execute_query(query)
                query_key = query.split("as ")[1] if "as " in query else "count"
                stats[query_key] = result[0][0] if result else 0
            except Exception as e:
                self.logger.debug(f"Commit frequency query failed: {e}")

        return stats

    async def calculate_developer_activity(self) -> Dict[str, Any]:
        """Calculate developer activity metrics"""

        stats = {}

        # Try to find developer-related information
        developer_queries = [
            "MATCH (n) WHERE n.author IS NOT NULL RETURN count(distinct n.author) as unique_authors",
            "MATCH (n) WHERE n.author_email IS NOT NULL RETURN count(distinct n.author_email) as unique_emails",
        ]

        for query in developer_queries:
            try:
                result = await self.execute_query(query)
                query_key = query.split("as ")[1] if "as " in query else "count"
                stats[query_key] = result[0][0] if result else 0
            except Exception as e:
                self.logger.debug(f"Developer activity query failed: {e}")

        return stats

    async def calculate_file_change_patterns(self) -> Dict[str, Any]:
        """Calculate file change patterns for risk assessment"""

        stats = {}

        # Try to find file-related information
        file_queries = [
            "MATCH (n) WHERE n.files_changed IS NOT NULL RETURN count(n) as nodes_with_files",
            "MATCH (n) WHERE n.additions IS NOT NULL RETURN sum(n.additions) as total_additions",
            "MATCH (n) WHERE n.deletions IS NOT NULL RETURN sum(n.deletions) as total_deletions",
        ]

        for query in file_queries:
            try:
                result = await self.execute_query(query)
                query_key = query.split("as ")[1] if "as " in query else "count"
                stats[query_key] = result[0][0] if result else 0
            except Exception as e:
                self.logger.debug(f"File change query failed: {e}")

        return stats

    async def find_hotfix_patterns(self) -> Dict[str, Any]:
        """Find patterns indicating hotfixes or urgent changes"""

        stats = {}

        # Try to find hotfix indicators
        hotfix_queries = [
            "MATCH (n) WHERE n.is_hotfix IS NOT NULL RETURN count(n) as hotfix_count",
            "MATCH (n) WHERE n.is_revert IS NOT NULL RETURN count(n) as revert_count",
            "MATCH (n) WHERE n.is_merge IS NOT NULL RETURN count(n) as merge_count",
        ]

        for query in hotfix_queries:
            try:
                result = await self.execute_query(query)
                query_key = query.split("as ")[1] if "as " in query else "count"
                stats[query_key] = result[0][0] if result else 0
            except Exception as e:
                self.logger.debug(f"Hotfix pattern query failed: {e}")

        return stats

    async def run_comprehensive_risk_analysis(self) -> Dict[str, Any]:
        """Run comprehensive risk analysis using all available CYPHER queries"""

        self.logger.info("Starting comprehensive risk analysis")

        analysis = {
            'timestamp': datetime.now().isoformat(),
            'basic_stats': {},
            'commit_frequency': {},
            'developer_activity': {},
            'file_patterns': {},
            'hotfix_patterns': {},
            'sample_nodes': [],
            'available_properties': []
        }

        try:
            # Basic statistics
            analysis['basic_stats'] = await self.get_basic_stats()

            # Sample node exploration
            analysis['sample_nodes'] = await self.get_all_node_properties(limit=3)

            # Risk calculations
            analysis['commit_frequency'] = await self.calculate_commit_frequency()
            analysis['developer_activity'] = await self.calculate_developer_activity()
            analysis['file_patterns'] = await self.calculate_file_change_patterns()
            analysis['hotfix_patterns'] = await self.find_hotfix_patterns()

            # Find available properties by exploring nodes
            commit_nodes = await self.find_commit_nodes()
            if commit_nodes:
                analysis['available_properties'] = [str(node) for node in commit_nodes[:3]]

            self.logger.info("Risk analysis complete",
                           nodes=analysis['basic_stats'].get('total_nodes', 0),
                           relationships=analysis['basic_stats'].get('total_relationships', 0))

        except Exception as e:
            self.logger.error(f"Risk analysis failed: {e}")
            analysis['error'] = str(e)

        return analysis


# Convenience functions

async def run_basic_cypher_test() -> Dict[str, Any]:
    """Run basic CYPHER functionality test"""
    engine = CypherQueryEngine()
    return await engine.get_basic_stats()


async def run_full_risk_analysis() -> Dict[str, Any]:
    """Run full risk analysis using CYPHER queries"""
    engine = CypherQueryEngine()
    return await engine.run_comprehensive_risk_analysis()