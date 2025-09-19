"""
Search aggregation system for CodeRisk

This module combines multiple search methods (CYPHER, chunks, graph completion)
to extract risk assessment data from Cognee's multi-modal storage.
"""

import asyncio
from typing import List, Dict, Any, Optional, Tuple
from datetime import datetime, timedelta
import re
import json
import cognee
from cognee import SearchType
import structlog

from .cypher_queries import CypherQueryEngine

logger = structlog.get_logger(__name__)


class SearchAggregator:
    """Aggregates search results from multiple Cognee search types for risk assessment"""

    def __init__(self):
        self.logger = logger
        self.cypher_engine = CypherQueryEngine()

    async def search_commits(self, query: str = "commits", limit: int = 100) -> Dict[str, Any]:
        """Search for commit data using multiple methods"""

        results = {
            'method': 'multi_search_commits',
            'query': query,
            'chunks': [],
            'graph': [],
            'total_found': 0,
            'parsed_commits': []
        }

        try:
            # Method 1: CHUNKS search for commit text data
            try:
                chunk_results = await cognee.search(
                    query_type=SearchType.CHUNKS,
                    query_text=f"commit commits author message SHA changes files {query}"
                )
                if chunk_results:
                    results['chunks'] = chunk_results[:limit]
                    self.logger.info(f"Found {len(results['chunks'])} commit chunks")
            except Exception as e:
                self.logger.debug(f"Chunks search failed: {e}")

            # Method 2: GRAPH search for commit-related data
            try:
                graph_results = await cognee.search(
                    query_type=SearchType.GRAPH_COMPLETION,
                    query_text=f"Find commit information, SHA, author, message, files changed: {query}"
                )
                if graph_results:
                    results['graph'] = [graph_results] if not isinstance(graph_results, list) else graph_results
                    self.logger.info(f"Found {len(results['graph'])} graph results")
            except Exception as e:
                self.logger.debug(f"Graph search failed: {e}")

            # Method 3: Natural language search
            try:
                nl_results = await cognee.search(
                    query_type=SearchType.NATURAL_LANGUAGE,
                    query_text=f"Show me commits with author information and file changes: {query}"
                )
                if nl_results:
                    if isinstance(nl_results, list):
                        results['graph'].extend(nl_results)
                    else:
                        results['graph'].append(nl_results)
            except Exception as e:
                self.logger.debug(f"Natural language search failed: {e}")

            # Parse commit data from results
            results['parsed_commits'] = self._parse_commit_data(results['chunks'] + results['graph'])
            results['total_found'] = len(results['parsed_commits'])

        except Exception as e:
            self.logger.error(f"Commit search failed: {e}")
            results['error'] = str(e)

        return results

    async def search_developers(self, query: str = "developers authors") -> Dict[str, Any]:
        """Search for developer data using multiple methods"""

        results = {
            'method': 'multi_search_developers',
            'query': query,
            'chunks': [],
            'graph': [],
            'total_found': 0,
            'parsed_developers': []
        }

        try:
            # Search for developer/author information
            search_queries = [
                f"developer author contributor programmer coder {query}",
                f"commits by author email username {query}",
                f"developer activity contributions stats {query}"
            ]

            for search_query in search_queries:
                try:
                    # CHUNKS search
                    chunk_results = await cognee.search(
                        query_type=SearchType.CHUNKS,
                        query_text=search_query
                    )
                    if chunk_results:
                        results['chunks'].extend(chunk_results if isinstance(chunk_results, list) else [chunk_results])

                    # GRAPH search
                    graph_results = await cognee.search(
                        query_type=SearchType.GRAPH_COMPLETION,
                        query_text=search_query
                    )
                    if graph_results:
                        results['graph'].extend(graph_results if isinstance(graph_results, list) else [graph_results])

                except Exception as e:
                    self.logger.debug(f"Developer search query failed: {e}")

            # Parse developer data
            results['parsed_developers'] = self._parse_developer_data(results['chunks'] + results['graph'])
            results['total_found'] = len(results['parsed_developers'])

        except Exception as e:
            self.logger.error(f"Developer search failed: {e}")
            results['error'] = str(e)

        return results

    async def search_files_and_changes(self, query: str = "files changed") -> Dict[str, Any]:
        """Search for file change data"""

        results = {
            'method': 'multi_search_files',
            'query': query,
            'chunks': [],
            'graph': [],
            'code': [],
            'total_found': 0,
            'parsed_files': []
        }

        try:
            # Search for file change information
            search_queries = [
                f"files changed modified added deleted {query}",
                f"additions deletions lines changed {query}",
                f"file modifications code changes {query}"
            ]

            for search_query in search_queries:
                try:
                    # CHUNKS search
                    chunk_results = await cognee.search(
                        query_type=SearchType.CHUNKS,
                        query_text=search_query
                    )
                    if chunk_results:
                        results['chunks'].extend(chunk_results if isinstance(chunk_results, list) else [chunk_results])

                    # CODE search for file-related information
                    code_results = await cognee.search(
                        query_type=SearchType.CODE,
                        query_text=search_query
                    )
                    if code_results:
                        results['code'].extend(code_results if isinstance(code_results, list) else [code_results])

                    # GRAPH search
                    graph_results = await cognee.search(
                        query_type=SearchType.GRAPH_COMPLETION,
                        query_text=search_query
                    )
                    if graph_results:
                        results['graph'].extend(graph_results if isinstance(graph_results, list) else [graph_results])

                except Exception as e:
                    self.logger.debug(f"File search query failed: {e}")

            # Parse file change data
            all_results = results['chunks'] + results['graph'] + results['code']
            results['parsed_files'] = self._parse_file_data(all_results)
            results['total_found'] = len(results['parsed_files'])

        except Exception as e:
            self.logger.error(f"File search failed: {e}")
            results['error'] = str(e)

        return results

    async def search_risk_indicators(self, query: str = "hotfix revert bug urgent") -> Dict[str, Any]:
        """Search for risk indicators and patterns"""

        results = {
            'method': 'multi_search_risk',
            'query': query,
            'chunks': [],
            'graph': [],
            'total_found': 0,
            'parsed_risks': []
        }

        try:
            # Search for risk indicators
            risk_queries = [
                "hotfix urgent emergency quick fix critical",
                "revert rollback undo fix mistake error",
                "bug issue problem defect incident failure",
                "breaking change breaking urgent priority high"
            ]

            for risk_query in risk_queries:
                try:
                    # CHUNKS search
                    chunk_results = await cognee.search(
                        query_type=SearchType.CHUNKS,
                        query_text=risk_query
                    )
                    if chunk_results:
                        results['chunks'].extend(chunk_results if isinstance(chunk_results, list) else [chunk_results])

                    # GRAPH search
                    graph_results = await cognee.search(
                        query_type=SearchType.GRAPH_COMPLETION,
                        query_text=risk_query
                    )
                    if graph_results:
                        results['graph'].extend(graph_results if isinstance(graph_results, list) else [graph_results])

                except Exception as e:
                    self.logger.debug(f"Risk search query failed: {e}")

            # Parse risk indicators
            results['parsed_risks'] = self._parse_risk_data(results['chunks'] + results['graph'])
            results['total_found'] = len(results['parsed_risks'])

        except Exception as e:
            self.logger.error(f"Risk search failed: {e}")
            results['error'] = str(e)

        return results

    def _parse_commit_data(self, search_results: List[Any]) -> List[Dict[str, Any]]:
        """Parse commit information from search results"""

        commits = []

        for result in search_results:
            try:
                # Extract text content
                text_content = ""
                if hasattr(result, 'text'):
                    text_content = result.text
                elif isinstance(result, dict) and 'text' in result:
                    text_content = result['text']
                elif isinstance(result, str):
                    text_content = result
                else:
                    text_content = str(result)

                # Parse commit patterns from text
                commit_info = self._extract_commit_patterns(text_content)
                if commit_info:
                    commits.append(commit_info)

            except Exception as e:
                self.logger.debug(f"Failed to parse commit result: {e}")

        return commits

    def _parse_developer_data(self, search_results: List[Any]) -> List[Dict[str, Any]]:
        """Parse developer information from search results"""

        developers = []

        for result in search_results:
            try:
                # Extract text content
                text_content = ""
                if hasattr(result, 'text'):
                    text_content = result.text
                elif isinstance(result, dict) and 'text' in result:
                    text_content = result['text']
                elif isinstance(result, str):
                    text_content = result
                else:
                    text_content = str(result)

                # Parse developer patterns from text
                dev_info = self._extract_developer_patterns(text_content)
                if dev_info:
                    developers.append(dev_info)

            except Exception as e:
                self.logger.debug(f"Failed to parse developer result: {e}")

        return developers

    def _parse_file_data(self, search_results: List[Any]) -> List[Dict[str, Any]]:
        """Parse file change information from search results"""

        files = []

        for result in search_results:
            try:
                # Extract text content
                text_content = ""
                if hasattr(result, 'text'):
                    text_content = result.text
                elif isinstance(result, dict) and 'text' in result:
                    text_content = result['text']
                elif isinstance(result, str):
                    text_content = result
                else:
                    text_content = str(result)

                # Parse file patterns from text
                file_info = self._extract_file_patterns(text_content)
                if file_info:
                    files.append(file_info)

            except Exception as e:
                self.logger.debug(f"Failed to parse file result: {e}")

        return files

    def _parse_risk_data(self, search_results: List[Any]) -> List[Dict[str, Any]]:
        """Parse risk indicators from search results"""

        risks = []

        for result in search_results:
            try:
                # Extract text content
                text_content = ""
                if hasattr(result, 'text'):
                    text_content = result.text
                elif isinstance(result, dict) and 'text' in result:
                    text_content = result['text']
                elif isinstance(result, str):
                    text_content = result
                else:
                    text_content = str(result)

                # Parse risk patterns from text
                risk_info = self._extract_risk_patterns(text_content)
                if risk_info:
                    risks.append(risk_info)

            except Exception as e:
                self.logger.debug(f"Failed to parse risk result: {e}")

        return risks

    def _extract_commit_patterns(self, text: str) -> Optional[Dict[str, Any]]:
        """Extract commit information using regex patterns"""

        commit_info = {}

        # Look for commit SHA patterns
        sha_pattern = r'([a-f0-9]{8,40})'
        sha_matches = re.findall(sha_pattern, text.lower())
        if sha_matches:
            commit_info['sha'] = sha_matches[0]

        # Look for author patterns
        author_patterns = [
            r'by\s+([a-zA-Z\s]+)',
            r'author[:\s]+([a-zA-Z\s]+)',
            r'Author:\s*([^<\n]+)'
        ]

        for pattern in author_patterns:
            matches = re.findall(pattern, text, re.IGNORECASE)
            if matches:
                commit_info['author'] = matches[0].strip()
                break

        # Look for timestamp patterns
        timestamp_patterns = [
            r'on\s+(\d{4}-\d{2}-\d{2})',
            r'(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})',
            r'(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2})'
        ]

        for pattern in timestamp_patterns:
            matches = re.findall(pattern, text)
            if matches:
                commit_info['timestamp'] = matches[0]
                break

        # Look for file change patterns
        file_patterns = [
            r'Files changed:\s*([^,\n]+(?:,\s*[^,\n]+)*)',
            r'files?:\s*([^,\n]+(?:,\s*[^,\n]+)*)'
        ]

        for pattern in file_patterns:
            matches = re.findall(pattern, text, re.IGNORECASE)
            if matches:
                files_str = matches[0]
                commit_info['files_changed'] = [f.strip() for f in files_str.split(',')]
                break

        # Look for message patterns
        message_patterns = [
            r'Message:\s*([^\n]+)',
            r'commit message:\s*([^\n]+)'
        ]

        for pattern in message_patterns:
            matches = re.findall(pattern, text, re.IGNORECASE)
            if matches:
                commit_info['message'] = matches[0].strip()
                break

        return commit_info if commit_info else None

    def _extract_developer_patterns(self, text: str) -> Optional[Dict[str, Any]]:
        """Extract developer information using regex patterns"""

        dev_info = {}

        # Look for developer name patterns
        name_patterns = [
            r'Developer:\s*([^(\n]+)',
            r'Username:\s*([^\s\n]+)',
            r'Author:\s*([^<\n]+)'
        ]

        for pattern in name_patterns:
            matches = re.findall(pattern, text, re.IGNORECASE)
            if matches:
                dev_info['name'] = matches[0].strip()
                break

        # Look for email patterns
        email_pattern = r'([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})'
        email_matches = re.findall(email_pattern, text)
        if email_matches:
            dev_info['email'] = email_matches[0]

        # Look for stats patterns
        stats_patterns = [
            r'(\d+)\s+commits?',
            r'(\d+)\s+PRs?\s+authored',
            r'(\d+)\s+PRs?\s+reviewed'
        ]

        for pattern in stats_patterns:
            matches = re.findall(pattern, text, re.IGNORECASE)
            if matches:
                if 'commits' in pattern:
                    dev_info['commits_count'] = int(matches[0])
                elif 'authored' in pattern:
                    dev_info['prs_authored'] = int(matches[0])
                elif 'reviewed' in pattern:
                    dev_info['prs_reviewed'] = int(matches[0])

        return dev_info if dev_info else None

    def _extract_file_patterns(self, text: str) -> Optional[Dict[str, Any]]:
        """Extract file change information using regex patterns"""

        file_info = {}

        # Look for file path patterns
        file_patterns = [
            r'([a-zA-Z0-9_./\\-]+\.[a-zA-Z]{1,4})',  # File extensions
            r'Files changed:\s*([^,\n]+(?:,\s*[^,\n]+)*)'
        ]

        for pattern in file_patterns:
            matches = re.findall(pattern, text)
            if matches:
                if 'Files changed' in pattern:
                    file_info['files'] = [f.strip() for f in matches[0].split(',')]
                else:
                    file_info['files'] = matches
                break

        # Look for change statistics
        stats_patterns = [
            r'(\d+)\s+additions?',
            r'(\d+)\s+deletions?',
            r'(\d+)\s+lines?\s+changed'
        ]

        for pattern in stats_patterns:
            matches = re.findall(pattern, text, re.IGNORECASE)
            if matches:
                if 'addition' in pattern:
                    file_info['additions'] = int(matches[0])
                elif 'deletion' in pattern:
                    file_info['deletions'] = int(matches[0])
                elif 'changed' in pattern:
                    file_info['lines_changed'] = int(matches[0])

        return file_info if file_info else None

    def _extract_risk_patterns(self, text: str) -> Optional[Dict[str, Any]]:
        """Extract risk indicators using regex patterns"""

        risk_info = {}

        # Look for risk indicator keywords
        risk_keywords = {
            'hotfix': r'\b(hotfix|urgent fix|emergency fix|critical fix)\b',
            'revert': r'\b(revert|rollback|undo)\b',
            'bug': r'\b(bug|defect|issue|problem|error)\b',
            'incident': r'\b(incident|outage|failure|crash)\b'
        }

        for risk_type, pattern in risk_keywords.items():
            matches = re.findall(pattern, text, re.IGNORECASE)
            if matches:
                risk_info[f'is_{risk_type}'] = True
                risk_info[f'{risk_type}_mentions'] = len(matches)

        # Look for severity indicators
        severity_patterns = [
            r'severity:\s*(high|medium|low|critical)',
            r'priority:\s*(high|medium|low|critical)',
            r'\[(critical|high|medium|low)\]'
        ]

        for pattern in severity_patterns:
            matches = re.findall(pattern, text, re.IGNORECASE)
            if matches:
                risk_info['severity'] = matches[0].lower()
                break

        return risk_info if risk_info else None

    async def run_comprehensive_search(self) -> Dict[str, Any]:
        """Run comprehensive search across all data types"""

        self.logger.info("Starting comprehensive search aggregation")

        analysis = {
            'timestamp': datetime.now().isoformat(),
            'commits': {},
            'developers': {},
            'files': {},
            'risks': {},
            'cypher_stats': {},
            'summary': {}
        }

        try:
            # Run all search methods in parallel
            commit_task = self.search_commits()
            developer_task = self.search_developers()
            file_task = self.search_files_and_changes()
            risk_task = self.search_risk_indicators()
            cypher_task = self.cypher_engine.get_basic_stats()

            # Wait for all results
            analysis['commits'] = await commit_task
            analysis['developers'] = await developer_task
            analysis['files'] = await file_task
            analysis['risks'] = await risk_task
            analysis['cypher_stats'] = await cypher_task

            # Create summary
            analysis['summary'] = {
                'total_commits_found': analysis['commits'].get('total_found', 0),
                'total_developers_found': analysis['developers'].get('total_found', 0),
                'total_files_found': analysis['files'].get('total_found', 0),
                'total_risks_found': analysis['risks'].get('total_found', 0),
                'graph_nodes': analysis['cypher_stats'].get('total_nodes', 0),
                'graph_relationships': analysis['cypher_stats'].get('total_relationships', 0)
            }

            self.logger.info("Comprehensive search complete", summary=analysis['summary'])

        except Exception as e:
            self.logger.error(f"Comprehensive search failed: {e}")
            analysis['error'] = str(e)

        return analysis


# Convenience functions

async def run_commit_search(query: str = "commits") -> Dict[str, Any]:
    """Run commit search"""
    aggregator = SearchAggregator()
    return await aggregator.search_commits(query)


async def run_comprehensive_analysis() -> Dict[str, Any]:
    """Run full comprehensive search analysis"""
    aggregator = SearchAggregator()
    return await aggregator.run_comprehensive_search()