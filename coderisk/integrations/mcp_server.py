"""
MCP (Model Context Protocol) Server Integration

Provides tools and server implementation for integrating CodeRisk with
Claude Code and other MCP-compatible AI systems.
"""

import json
import asyncio
from typing import Dict, List, Optional, Any, Union
from dataclasses import dataclass
from pathlib import Path

from ..core.risk_engine import RiskEngine
from ..detectors import detector_registry


@dataclass
class MCPTool:
    """MCP tool definition"""
    name: str
    description: str
    parameters: Dict[str, Any]
    handler: callable


class MCPServer:
    """MCP server for CodeRisk integration"""

    def __init__(self, repo_path: str):
        self.repo_path = repo_path
        self.risk_engine = None
        self.tools = {}
        self._register_tools()

    def _register_tools(self):
        """Register all available MCP tools"""

        # Risk assessment tool
        self.tools['assess_worktree'] = MCPTool(
            name='assess_worktree',
            description='Assess risk of uncommitted changes in the working tree',
            parameters={
                'type': 'object',
                'properties': {
                    'repo_path': {
                        'type': 'string',
                        'description': 'Path to the repository (optional, uses current if not specified)'
                    },
                    'categories': {
                        'type': 'boolean',
                        'description': 'Include category breakdown in results',
                        'default': False
                    },
                    'explain': {
                        'type': 'boolean',
                        'description': 'Include detailed explanations',
                        'default': False
                    }
                }
            },
            handler=self._assess_worktree
        )

        # Commit risk assessment tool
        self.tools['score_pr'] = MCPTool(
            name='score_pr',
            description='Score risk for a specific pull request or commit',
            parameters={
                'type': 'object',
                'properties': {
                    'commit_sha': {
                        'type': 'string',
                        'description': 'Commit SHA or PR identifier to analyze'
                    },
                    'repo_path': {
                        'type': 'string',
                        'description': 'Path to the repository (optional)'
                    }
                },
                'required': ['commit_sha']
            },
            handler=self._score_pr
        )

        # Explain high risk tool
        self.tools['explain_high'] = MCPTool(
            name='explain_high',
            description='Get detailed explanation for high-risk changes',
            parameters={
                'type': 'object',
                'properties': {
                    'file_path': {
                        'type': 'string',
                        'description': 'Specific file to analyze (optional)'
                    },
                    'category': {
                        'type': 'string',
                        'description': 'Specific risk category to explain',
                        'enum': ['api', 'schema', 'deps', 'perf', 'concurrency', 'security', 'config', 'tests', 'merge']
                    }
                }
            },
            handler=self._explain_high
        )

        # Graph neighborhood viewer
        self.tools['graph_neighborhood'] = MCPTool(
            name='graph_neighborhood',
            description='View graph neighborhood for a file or symbol',
            parameters={
                'type': 'object',
                'properties': {
                    'target': {
                        'type': 'string',
                        'description': 'File path or symbol to explore'
                    },
                    'hops': {
                        'type': 'integer',
                        'description': 'Number of hops to traverse (default: 2)',
                        'default': 2,
                        'minimum': 1,
                        'maximum': 5
                    },
                    'relationship_type': {
                        'type': 'string',
                        'description': 'Type of relationships to follow',
                        'enum': ['imports', 'co_changed', 'both'],
                        'default': 'both'
                    }
                },
                'required': ['target']
            },
            handler=self._graph_neighborhood
        )

        # Detector runner tool
        self.tools['run_detectors'] = MCPTool(
            name='run_detectors',
            description='Run specific micro-detectors on current changes',
            parameters={
                'type': 'object',
                'properties': {
                    'detectors': {
                        'type': 'array',
                        'items': {
                            'type': 'string',
                            'enum': ['api_break', 'schema_risk', 'dependency_risk', 'performance_risk',
                                   'concurrency_risk', 'security_risk', 'config_risk', 'test_gap', 'merge_risk']
                        },
                        'description': 'List of detectors to run (all if not specified)'
                    },
                    'timeout_ms': {
                        'type': 'integer',
                        'description': 'Timeout per detector in milliseconds',
                        'default': 150,
                        'minimum': 50,
                        'maximum': 1000
                    }
                }
            },
            handler=self._run_detectors
        )

    async def initialize(self):
        """Initialize the MCP server"""
        self.risk_engine = RiskEngine(self.repo_path)
        await self.risk_engine.initialize()

    async def handle_request(self, method: str, params: Dict[str, Any]) -> Dict[str, Any]:
        """Handle MCP request"""
        if method == 'tools/list':
            return await self._list_tools()
        elif method == 'tools/call':
            return await self._call_tool(params)
        else:
            return {
                'error': {
                    'code': -32601,
                    'message': f'Method not found: {method}'
                }
            }

    async def _list_tools(self) -> Dict[str, Any]:
        """List all available tools"""
        tools = []
        for tool_name, tool in self.tools.items():
            tools.append({
                'name': tool.name,
                'description': tool.description,
                'inputSchema': tool.parameters
            })

        return {'tools': tools}

    async def _call_tool(self, params: Dict[str, Any]) -> Dict[str, Any]:
        """Call a specific tool"""
        tool_name = params.get('name')
        arguments = params.get('arguments', {})

        if tool_name not in self.tools:
            return {
                'error': {
                    'code': -32602,
                    'message': f'Unknown tool: {tool_name}'
                }
            }

        tool = self.tools[tool_name]

        try:
            result = await tool.handler(arguments)
            return {
                'content': [
                    {
                        'type': 'text',
                        'text': json.dumps(result, indent=2)
                    }
                ]
            }
        except Exception as e:
            return {
                'error': {
                    'code': -32603,
                    'message': f'Tool execution failed: {str(e)}'
                }
            }

    async def _assess_worktree(self, args: Dict[str, Any]) -> Dict[str, Any]:
        """Assess worktree risk"""
        repo_path = args.get('repo_path', self.repo_path)
        categories = args.get('categories', False)
        explain = args.get('explain', False)

        if not self.risk_engine:
            await self.initialize()

        # Get basic assessment
        assessment = await self.risk_engine.assess_worktree_risk()

        result = {
            'tier': assessment.tier.value,
            'score': round(assessment.score, 1),
            'confidence': round(assessment.confidence, 2),
            'explanation': assessment.get_explanation(),
            'assessment_time_ms': assessment.assessment_time_ms,
            'change_context': {
                'files_changed': len(assessment.change_context.files_changed),
                'lines_added': assessment.change_context.lines_added,
                'lines_deleted': assessment.change_context.lines_deleted
            }
        }

        # Add categories if requested
        if categories or explain:
            detector_results = await self._run_all_detectors()

            if categories:
                result['categories'] = self._format_categories(detector_results)

            if explain:
                result['explanations'] = detector_results

        return result

    async def _score_pr(self, args: Dict[str, Any]) -> Dict[str, Any]:
        """Score a pull request or commit"""
        commit_sha = args['commit_sha']
        repo_path = args.get('repo_path', self.repo_path)

        if not self.risk_engine:
            await self.initialize()

        assessment = await self.risk_engine.assess_commit_risk(commit_sha)

        return {
            'commit_sha': commit_sha,
            'tier': assessment.tier.value,
            'score': round(assessment.score, 1),
            'explanation': assessment.get_explanation(),
            'assessment_time_ms': assessment.assessment_time_ms
        }

    async def _explain_high(self, args: Dict[str, Any]) -> Dict[str, Any]:
        """Explain high-risk findings"""
        file_path = args.get('file_path')
        category = args.get('category')

        detector_results = await self._run_all_detectors()

        explanations = {}

        # Filter by category if specified
        if category:
            detector_name = f"{category}_risk" if category != 'api' else 'api_break'
            if detector_name in detector_results:
                explanations[category] = detector_results[detector_name]
        else:
            # Return all high-risk findings
            for detector_name, result in detector_results.items():
                if result.get('score', 0) >= 0.6:  # High risk threshold
                    category_name = detector_name.replace('_risk', '').replace('api_break', 'api')
                    explanations[category_name] = result

        return {
            'explanations': explanations,
            'high_risk_count': len(explanations)
        }

    async def _graph_neighborhood(self, args: Dict[str, Any]) -> Dict[str, Any]:
        """Get graph neighborhood for a target"""
        target = args['target']
        hops = args.get('hops', 2)
        relationship_type = args.get('relationship_type', 'both')

        # This would integrate with the graph system
        # For now, return a mock structure
        return {
            'target': target,
            'hops': hops,
            'relationship_type': relationship_type,
            'nodes': [],
            'edges': [],
            'note': 'Graph integration not yet implemented'
        }

    async def _run_detectors(self, args: Dict[str, Any]) -> Dict[str, Any]:
        """Run specific detectors"""
        detector_names = args.get('detectors', [])
        timeout_ms = args.get('timeout_ms', 150)

        # Mock change context - would integrate with git
        from ..detectors import ChangeContext
        context = ChangeContext(
            files_changed=[],
            total_lines_added=0,
            total_lines_deleted=0
        )

        results = {}

        if not detector_names:
            # Run all detectors
            detectors = detector_registry.get_all_detectors(self.repo_path)
        else:
            # Run specific detectors
            detectors = []
            for name in detector_names:
                try:
                    detector = detector_registry.get_detector(name, self.repo_path)
                    detectors.append(detector)
                except ValueError:
                    results[name] = {'error': f'Unknown detector: {name}'}

        for detector in detectors:
            try:
                result = await detector.run_with_timeout(context, timeout_ms)
                results[detector.name] = result.to_dict()
            except Exception as e:
                results[detector.name] = {'error': str(e)}

        return results

    async def _run_all_detectors(self) -> Dict[str, Any]:
        """Run all detectors and return results"""
        return await self._run_detectors({})

    def _format_categories(self, detector_results: Dict[str, Any]) -> Dict[str, Any]:
        """Format detector results as categories"""
        categories = {}

        category_mapping = {
            'api_break': 'api',
            'schema_risk': 'schema',
            'dependency_risk': 'deps',
            'performance_risk': 'perf',
            'concurrency_risk': 'concurrency',
            'security_risk': 'security',
            'config_risk': 'config',
            'test_gap': 'tests',
            'merge_risk': 'merge'
        }

        for detector_name, category_name in category_mapping.items():
            result = detector_results.get(detector_name, {})
            categories[category_name] = {
                'score': result.get('score', 0.0),
                'reasons': result.get('reasons', []),
                'anchors': result.get('anchors', [])
            }

        return categories

    def get_server_info(self) -> Dict[str, Any]:
        """Get MCP server information"""
        return {
            'name': 'CodeRisk MCP Server',
            'version': '0.1.0',
            'description': 'AI-powered code regression risk assessment',
            'tools_count': len(self.tools),
            'supported_protocols': ['MCP 1.0'],
            'repository': self.repo_path
        }