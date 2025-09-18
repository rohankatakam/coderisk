"""
CodeRisk Integrations Module

Provides integration with external systems like MCP servers, GitHub Actions,
CI/CD pipelines, and other development tools.
"""

from .mcp_server import MCPServer, MCPTool
from .github_actions import GitHubActionsIntegration

__all__ = [
    'MCPServer',
    'MCPTool',
    'GitHubActionsIntegration'
]