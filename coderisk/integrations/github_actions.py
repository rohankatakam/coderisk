"""
GitHub Actions Integration

Provides tools and templates for integrating CodeRisk with GitHub Actions
workflows, including PR checks, status updates, and automated comments.
"""

import json
import os
import subprocess
from typing import Dict, List, Optional, Any
from pathlib import Path
from dataclasses import dataclass

from ..core.risk_engine import RiskEngine


@dataclass
class GitHubContext:
    """GitHub Actions context information"""
    event_name: str
    repository: str
    ref: str
    sha: str
    pull_request_number: Optional[int]
    actor: str
    workflow: str


@dataclass
class PRComment:
    """PR comment structure"""
    body: str
    path: Optional[str] = None
    line: Optional[int] = None
    side: str = "RIGHT"  # "LEFT" or "RIGHT"


class GitHubActionsIntegration:
    """GitHub Actions integration for CodeRisk"""

    def __init__(self, repo_path: str):
        self.repo_path = repo_path
        self.github_context = self._get_github_context()

    def _get_github_context(self) -> Optional[GitHubContext]:
        """Extract GitHub Actions context from environment"""
        if not os.getenv('GITHUB_ACTIONS'):
            return None

        return GitHubContext(
            event_name=os.getenv('GITHUB_EVENT_NAME', ''),
            repository=os.getenv('GITHUB_REPOSITORY', ''),
            ref=os.getenv('GITHUB_REF', ''),
            sha=os.getenv('GITHUB_SHA', ''),
            pull_request_number=self._get_pr_number(),
            actor=os.getenv('GITHUB_ACTOR', ''),
            workflow=os.getenv('GITHUB_WORKFLOW', '')
        )

    def _get_pr_number(self) -> Optional[int]:
        """Extract PR number from GitHub context"""
        if os.getenv('GITHUB_EVENT_NAME') == 'pull_request':
            event_path = os.getenv('GITHUB_EVENT_PATH')
            if event_path and os.path.exists(event_path):
                try:
                    with open(event_path, 'r') as f:
                        event_data = json.load(f)
                    return event_data.get('pull_request', {}).get('number')
                except Exception:
                    pass
        return None

    async def run_pr_check(self) -> Dict[str, Any]:
        """Run CodeRisk check for a pull request"""
        if not self.github_context or self.github_context.event_name != 'pull_request':
            raise ValueError("This function can only be run in a pull request context")

        risk_engine = RiskEngine(self.repo_path)
        await risk_engine.initialize()

        # Get the assessment
        assessment = await risk_engine.assess_worktree_risk()

        # Create check result
        result = {
            'conclusion': self._get_check_conclusion(assessment),
            'title': f"CodeRisk Assessment: {assessment.tier.value.upper()}",
            'summary': self._generate_check_summary(assessment),
            'annotations': self._generate_annotations(assessment),
            'actions': self._generate_actions(assessment)
        }

        return result

    async def generate_pr_comment(self, include_categories: bool = True) -> str:
        """Generate a PR comment with risk assessment"""
        risk_engine = RiskEngine(self.repo_path)
        await risk_engine.initialize()

        assessment = await risk_engine.assess_worktree_risk()

        # Get detector results if categories requested
        detector_results = {}
        if include_categories:
            # This would run the detectors - simplified for now
            pass

        comment_body = self._format_pr_comment(assessment, detector_results if include_categories else None)
        return comment_body

    def _get_check_conclusion(self, assessment) -> str:
        """Get GitHub check conclusion based on assessment"""
        tier_to_conclusion = {
            'LOW': 'success',
            'MEDIUM': 'neutral',
            'HIGH': 'failure',
            'CRITICAL': 'failure'
        }
        return tier_to_conclusion.get(assessment.tier.value, 'neutral')

    def _generate_check_summary(self, assessment) -> str:
        """Generate check summary"""
        return f"""
## CodeRisk Assessment Summary

**Risk Level**: {assessment.tier.value} ({assessment.score:.1f}/100)

{assessment.get_explanation()}

**Changes**:
- Files changed: {len(assessment.change_context.files_changed)}
- Lines added: {assessment.change_context.lines_added}
- Lines deleted: {assessment.change_context.lines_deleted}

**Assessment completed in {assessment.assessment_time_ms}ms**
        """.strip()

    def _generate_annotations(self, assessment) -> List[Dict[str, Any]]:
        """Generate GitHub check annotations"""
        annotations = []

        # Add annotations for top concerns
        for i, concern in enumerate(assessment.top_concerns[:5]):
            annotations.append({
                'path': 'README.md',  # Default path
                'start_line': 1,
                'end_line': 1,
                'annotation_level': 'warning',
                'title': 'Risk Factor',
                'message': concern.replace('_', ' ').title()
            })

        return annotations

    def _generate_actions(self, assessment) -> List[Dict[str, Any]]:
        """Generate suggested actions"""
        actions = []

        if assessment.tier.value in ['HIGH', 'CRITICAL']:
            actions.append({
                'label': 'Request Review',
                'description': 'Request additional review due to high risk',
                'identifier': 'request_review'
            })

        if assessment.recommendations:
            actions.append({
                'label': 'View Recommendations',
                'description': 'View detailed recommendations for risk mitigation',
                'identifier': 'view_recommendations'
            })

        return actions

    def _format_pr_comment(self, assessment, detector_results: Optional[Dict] = None) -> str:
        """Format PR comment with risk assessment"""
        tier_emoji = {
            'LOW': '✅',
            'MEDIUM': '⚠️',
            'HIGH': '🔶',
            'CRITICAL': '🔴'
        }

        comment = f"""
## {tier_emoji.get(assessment.tier.value, '⚪')} CodeRisk Assessment

**Risk Level**: {assessment.tier.value} ({assessment.score:.1f}/100)

{assessment.get_explanation()}

### 📊 Change Summary
- **Files changed**: {len(assessment.change_context.files_changed)}
- **Lines added**: {assessment.change_context.lines_added}
- **Lines deleted**: {assessment.change_context.lines_deleted}
        """

        # Add top concerns
        if assessment.top_concerns:
            comment += "\n### 🎯 Top Concerns\n"
            for i, concern in enumerate(assessment.top_concerns[:3], 1):
                comment += f"{i}. {concern.replace('_', ' ').title()}\n"

        # Add category breakdown if provided
        if detector_results:
            comment += "\n### 📋 Risk Categories\n"
            comment += "| Category | Score | Status |\n"
            comment += "|----------|-------|--------|\n"

            for detector_name, result in detector_results.items():
                score = result.get('score', 0.0)
                category_name = detector_name.replace('_', ' ').title()

                if score >= 0.8:
                    status = "🔴 Critical"
                elif score >= 0.6:
                    status = "🔶 High"
                elif score >= 0.3:
                    status = "⚠️ Medium"
                elif score > 0:
                    status = "💡 Low"
                else:
                    status = "✅ Good"

                comment += f"| {category_name} | {score:.2f} | {status} |\n"

        # Add recommendations
        if assessment.recommendations:
            comment += "\n### 💡 Recommendations\n"
            for rec in assessment.recommendations[:3]:
                priority_emoji = {"high": "🔴", "medium": "🟡", "low": "🟢"}.get(rec.priority, "⚪")
                comment += f"- {priority_emoji} **{rec.action}**: {rec.description}\n"

        comment += f"\n---\n*Assessment completed in {assessment.assessment_time_ms}ms*\n"
        comment += "*🤖 Generated by [CodeRisk](https://github.com/rohankatakam/coderisk)*"

        return comment

    def create_status_check(self, assessment) -> Dict[str, Any]:
        """Create GitHub status check payload"""
        return {
            'state': self._get_status_state(assessment),
            'description': f"Risk: {assessment.tier.value} ({assessment.score:.1f}/100)",
            'context': 'CodeRisk Assessment',
            'target_url': None  # Could link to detailed report
        }

    def _get_status_state(self, assessment) -> str:
        """Get GitHub status state"""
        state_mapping = {
            'LOW': 'success',
            'MEDIUM': 'success',
            'HIGH': 'failure',
            'CRITICAL': 'failure'
        }
        return state_mapping.get(assessment.tier.value, 'pending')

    def generate_workflow_template(self, workflow_name: str = "coderisk") -> str:
        """Generate GitHub Actions workflow template"""
        template = f"""
name: {workflow_name.title()}

on:
  pull_request:
    types: [opened, synchronize]
  push:
    branches: [main, develop]

jobs:
  risk-assessment:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write
      checks: write

    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history for better analysis

      - name: Set up Python
        uses: actions/setup-python@v4
        with:
          python-version: '3.11'

      - name: Install CodeRisk
        run: |
          pip install coderisk

      - name: Run CodeRisk Assessment
        id: coderisk
        run: |
          crisk check --json --categories --explain > assessment.json
          echo "assessment<<EOF" >> $GITHUB_OUTPUT
          cat assessment.json >> $GITHUB_OUTPUT
          echo "EOF" >> $GITHUB_OUTPUT

      - name: Create PR Comment
        if: github.event_name == 'pull_request'
        uses: actions/github-script@v7
        with:
          script: |
            const assessment = JSON.parse(`${{{{ steps.coderisk.outputs.assessment }}}}`);

            let comment = `## 🔍 CodeRisk Assessment\\n\\n`;
            comment += `**Risk Level**: ${{assessment.tier}} (${{assessment.score}}/100)\\n\\n`;
            comment += `${{assessment.explanation}}\\n\\n`;

            if (assessment.categories) {{
              comment += `### 📋 Risk Categories\\n\\n`;
              comment += `| Category | Score | Status |\\n`;
              comment += `|----------|-------|--------|\\n`;

              for (const [category, data] of Object.entries(assessment.categories)) {{
                const score = data.score;
                let status = '✅ Good';
                if (score >= 0.8) status = '🔴 Critical';
                else if (score >= 0.6) status = '🔶 High';
                else if (score >= 0.3) status = '⚠️ Medium';
                else if (score > 0) status = '💡 Low';

                comment += `| ${{category.charAt(0).toUpperCase() + category.slice(1)}} | ${{score.toFixed(2)}} | ${{status}} |\\n`;
              }}
            }}

            comment += `\\n---\\n*Assessment completed in ${{assessment.assessment_time_ms}}ms*`;

            github.rest.issues.createComment({{
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: comment
            }});

      - name: Update Check Status
        if: always()
        uses: actions/github-script@v7
        with:
          script: |
            const assessment = JSON.parse(`${{{{ steps.coderisk.outputs.assessment }}}}`);

            let conclusion = 'success';
            if (assessment.tier === 'HIGH' || assessment.tier === 'CRITICAL') {{
              conclusion = 'failure';
            }} else if (assessment.tier === 'MEDIUM') {{
              conclusion = 'neutral';
            }}

            await github.rest.checks.create({{
              owner: context.repo.owner,
              repo: context.repo.repo,
              name: 'CodeRisk Assessment',
              head_sha: context.sha,
              status: 'completed',
              conclusion: conclusion,
              output: {{
                title: `Risk Assessment: ${{assessment.tier}}`,
                summary: `Risk score: ${{assessment.score}}/100\\n\\n${{assessment.explanation}}`,
              }}
            }});
        """.strip()

        return template

    def get_setup_instructions(self) -> str:
        """Get setup instructions for GitHub Actions integration"""
        return """
# CodeRisk GitHub Actions Setup

## 1. Add Workflow File

Create `.github/workflows/coderisk.yml` in your repository with the generated template.

## 2. Required Permissions

Make sure your workflow has these permissions:
- `contents: read` - To checkout code
- `pull-requests: write` - To comment on PRs
- `checks: write` - To create check runs

## 3. Environment Variables (Optional)

You can customize CodeRisk behavior with these environment variables:
- `CODERISK_RISK_THRESHOLD` - Risk threshold for failing checks (default: 70)
- `CODERISK_TIMEOUT_MS` - Timeout for analysis in milliseconds (default: 30000)
- `CODERISK_CATEGORIES` - Always include category breakdown (default: true)

## 4. Integration with Branch Protection

You can add CodeRisk as a required check in your branch protection rules:
1. Go to Settings > Branches
2. Add "CodeRisk Assessment" to required status checks
3. Configure rules based on your team's risk tolerance

## 5. Advanced Configuration

For advanced setups, you can:
- Use matrix builds to test multiple configurations
- Integrate with external reporting tools
- Set up scheduled assessments for dependency monitoring
- Configure different thresholds for different file types

## 6. Troubleshooting

Common issues:
- **Analysis timeout**: Increase `CODERISK_TIMEOUT_MS` for large repositories
- **Missing permissions**: Ensure workflow has proper GitHub token permissions
- **False positives**: Use `.coderiskignore` file to exclude specific patterns

For more help, see the CodeRisk documentation.
        """.strip()