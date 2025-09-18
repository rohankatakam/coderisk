"""
CodeRisk CLI - Main command interface
"""

import asyncio
import click
import json
from pathlib import Path
from typing import Dict, Any
from rich.console import Console
from rich.table import Table
from rich.panel import Panel
from rich.text import Text
from rich.progress import Progress, SpinnerColumn, TextColumn

from ..core.risk_engine import RiskEngine
from ..models.risk_assessment import RiskTier
from ..detectors import detector_registry, ChangeContext, FileChange
from ..detectors.api_detector import ApiBreakDetector

console = Console()


async def run_micro_detectors(repo_path: str, progress=None) -> Dict[str, Any]:
    """Run all micro-detectors and return results"""
    # This is a simplified implementation - in practice would need to:
    # 1. Get git diff to create ChangeContext
    # 2. Parse file changes into FileChange objects
    # 3. Run all detectors in parallel

    if progress:
        progress.update(progress.task_ids[0], description="Running micro-detectors...")

    # Mock implementation for now - would integrate with git diff
    context = ChangeContext(
        files_changed=[],
        total_lines_added=0,
        total_lines_deleted=0
    )

    detector_results = {}
    detectors = detector_registry.get_all_detectors(repo_path)

    for detector in detectors:
        try:
            result = await detector.run_with_timeout(context)
            detector_results[detector.name] = result.to_dict()
        except Exception as e:
            detector_results[detector.name] = {
                "score": 0.0,
                "reasons": [f"Detector failed: {str(e)}"],
                "anchors": [],
                "evidence": {"error": str(e)},
                "execution_time_ms": 0.0
            }

    return detector_results


def print_banner():
    """Print CodeRisk banner"""
    banner = """
╔═══════════════════════════════════════╗
║              CodeRisk                 ║
║        Know your risk before          ║
║            you ship                   ║
╚═══════════════════════════════════════╝
    """
    console.print(banner, style="bold blue")


def get_tier_color(tier: RiskTier) -> str:
    """Get color for risk tier"""
    colors = {
        RiskTier.LOW: "green",
        RiskTier.MEDIUM: "yellow",
        RiskTier.HIGH: "orange",
        RiskTier.CRITICAL: "red"
    }
    return colors.get(tier, "white")


def get_tier_emoji(tier: RiskTier) -> str:
    """Get emoji for risk tier"""
    emojis = {
        RiskTier.LOW: "✅",
        RiskTier.MEDIUM: "⚠️",
        RiskTier.HIGH: "🔶",
        RiskTier.CRITICAL: "🔴"
    }
    return emojis.get(tier, "⚪")


@click.group()
@click.version_option(version="0.1.0")
def cli():
    """CodeRisk - AI-powered code regression risk assessment"""
    pass


@cli.command()
@click.option("--repo", "-r", default=".", help="Repository path (default: current directory)")
@click.option("--json", "output_json", is_flag=True, help="Output results as JSON")
@click.option("--verbose", "-v", is_flag=True, help="Verbose output")
@click.option("--explain", is_flag=True, help="Show detailed explanations for risk factors")
@click.option("--categories", is_flag=True, help="Show risk breakdown by category")
def check(repo: str, output_json: bool, verbose: bool, explain: bool, categories: bool):
    """Check risk of uncommitted changes in working tree"""

    if not output_json:
        print_banner()

    async def run_assessment():
        repo_path = Path(repo).resolve()

        if not repo_path.exists():
            console.print(f"❌ Repository path does not exist: {repo_path}", style="red")
            return

        if not output_json:
            console.print(f"🔍 Analyzing repository: {repo_path}")

        # Initialize risk engine with progress
        with Progress(
            SpinnerColumn(),
            TextColumn("[progress.description]{task.description}"),
            console=console if not output_json else None,
            transient=True,
        ) as progress:
            if not output_json:
                task = progress.add_task("Initializing CodeGraph...", total=None)

            risk_engine = RiskEngine(str(repo_path))

            try:
                await risk_engine.initialize()

                if not output_json:
                    progress.update(task, description="Analyzing changes...")

                # Assess risk
                assessment = await risk_engine.assess_worktree_risk()

                # Run micro-detectors if categories or explain flags are used
                detector_results = {}
                if categories or explain:
                    detector_results = await run_micro_detectors(str(repo_path), progress)

                if not output_json:
                    progress.update(task, description="Assessment complete!", completed=True)

            except Exception as e:
                if output_json:
                    console.print(json.dumps({"error": str(e)}))
                else:
                    console.print(f"❌ Error during assessment: {e}", style="red")
                return

        # Output results
        if output_json:
            # JSON output
            result = {
                "tier": assessment.tier.value,
                "score": round(assessment.score, 1),
                "confidence": round(assessment.confidence, 2),
                "total_regression_risk": round(assessment.total_regression_risk, 2),
                "top_concerns": assessment.top_concerns,
                "explanation": assessment.get_explanation(),
                "assessment_time_ms": assessment.assessment_time_ms,
                "scaling_factors": {
                    "team_factor": round(assessment.team_factor, 2),
                    "codebase_factor": round(assessment.codebase_factor, 2),
                    "change_velocity": round(assessment.change_velocity, 2),
                    "migration_multiplier": round(assessment.migration_multiplier, 2)
                },
                "change_context": {
                    "files_changed": len(assessment.change_context.files_changed),
                    "lines_added": assessment.change_context.lines_added,
                    "lines_deleted": assessment.change_context.lines_deleted
                }
            }

            # Add category results if requested
            if categories and detector_results:
                result["categories"] = {
                    "api": {
                        "score": detector_results.get("api_break", {}).get("score", 0.0),
                        "reasons": detector_results.get("api_break", {}).get("reasons", []),
                        "anchors": detector_results.get("api_break", {}).get("anchors", [])
                    },
                    "schema": {
                        "score": detector_results.get("schema_risk", {}).get("score", 0.0),
                        "reasons": detector_results.get("schema_risk", {}).get("reasons", []),
                        "anchors": detector_results.get("schema_risk", {}).get("anchors", [])
                    },
                    "deps": {
                        "score": detector_results.get("dependency_risk", {}).get("score", 0.0),
                        "reasons": detector_results.get("dependency_risk", {}).get("reasons", []),
                        "anchors": detector_results.get("dependency_risk", {}).get("anchors", [])
                    },
                    "perf": {
                        "score": detector_results.get("performance_risk", {}).get("score", 0.0),
                        "reasons": detector_results.get("performance_risk", {}).get("reasons", []),
                        "anchors": detector_results.get("performance_risk", {}).get("anchors", [])
                    },
                    "concurrency": {
                        "score": detector_results.get("concurrency_risk", {}).get("score", 0.0),
                        "reasons": detector_results.get("concurrency_risk", {}).get("reasons", []),
                        "anchors": detector_results.get("concurrency_risk", {}).get("anchors", [])
                    },
                    "security": {
                        "score": detector_results.get("security_risk", {}).get("score", 0.0),
                        "reasons": detector_results.get("security_risk", {}).get("reasons", []),
                        "anchors": detector_results.get("security_risk", {}).get("anchors", [])
                    },
                    "config": {
                        "score": detector_results.get("config_risk", {}).get("score", 0.0),
                        "reasons": detector_results.get("config_risk", {}).get("reasons", []),
                        "anchors": detector_results.get("config_risk", {}).get("anchors", [])
                    },
                    "tests": {
                        "score": detector_results.get("test_gap", {}).get("score", 0.0),
                        "reasons": detector_results.get("test_gap", {}).get("reasons", []),
                        "anchors": detector_results.get("test_gap", {}).get("anchors", [])
                    },
                    "merge": {
                        "score": detector_results.get("merge_risk", {}).get("score", 0.0),
                        "reasons": detector_results.get("merge_risk", {}).get("reasons", []),
                        "anchors": detector_results.get("merge_risk", {}).get("anchors", [])
                    }
                }

            if verbose:
                result["signals"] = [
                    {
                        "name": signal.name,
                        "score": round(signal.score, 2),
                        "confidence": round(signal.confidence, 2),
                        "evidence": signal.evidence
                    } for signal in assessment.signals
                ]
                result["recommendations"] = [
                    {
                        "action": rec.action,
                        "priority": rec.priority,
                        "description": rec.description
                    } for rec in assessment.recommendations
                ]

            # Add detailed explanations if requested
            if explain and detector_results:
                result["explanations"] = {
                    "detectors": detector_results
                }

            console.print(json.dumps(result, indent=2))
        else:
            # Rich formatted output
            console.print()

            # Main risk panel
            tier_emoji = get_tier_emoji(assessment.tier)
            tier_color = get_tier_color(assessment.tier)

            risk_text = Text()
            risk_text.append(f"{tier_emoji} Risk Level: ", style="bold")
            risk_text.append(f"{assessment.tier.value}", style=f"bold {tier_color}")
            risk_text.append(f" ({assessment.score:.1f}/100)", style="bold")

            panel_content = f"""
{risk_text}

{assessment.get_explanation()}

Assessment completed in {assessment.assessment_time_ms}ms
            """.strip()

            console.print(Panel(panel_content, title="Risk Assessment", border_style=tier_color))

            # Top concerns
            if assessment.top_concerns:
                console.print("\n🎯 Top Concerns:", style="bold")
                for i, concern in enumerate(assessment.top_concerns[:3], 1):
                    console.print(f"  {i}. {concern.replace('_', ' ').title()}")

            # Change summary
            console.print(f"\n📊 Change Summary:", style="bold")
            console.print(f"  • Files changed: {len(assessment.change_context.files_changed)}")
            console.print(f"  • Lines added: {assessment.change_context.lines_added}")
            console.print(f"  • Lines deleted: {assessment.change_context.lines_deleted}")

            # Regression scaling factors (if verbose)
            if verbose:
                console.print(f"\n⚖️  Regression Scaling Factors:", style="bold")
                console.print(f"  • Team factor: {assessment.team_factor:.2f}x")
                console.print(f"  • Codebase factor: {assessment.codebase_factor:.2f}x")
                console.print(f"  • Change velocity: {assessment.change_velocity:.2f}x")
                console.print(f"  • Migration multiplier: {assessment.migration_multiplier:.2f}x")
                console.print(f"  • Total regression risk: {assessment.total_regression_risk:.2f}x baseline")

            # Recommendations
            if assessment.recommendations:
                console.print(f"\n💡 Recommendations:", style="bold")
                for rec in assessment.recommendations[:3]:
                    priority_style = "red" if rec.priority == "high" else "yellow" if rec.priority == "medium" else "green"
                    console.print(f"  • [{rec.priority.upper()}] {rec.action}", style=priority_style)
                    console.print(f"    {rec.description}", style="dim")

            # Signal details (if verbose)
            if verbose and assessment.signals:
                console.print(f"\n🔍 Signal Details:", style="bold")

                table = Table(show_header=True, header_style="bold magenta")
                table.add_column("Signal", style="cyan")
                table.add_column("Score", justify="center")
                table.add_column("Confidence", justify="center")
                table.add_column("Evidence", style="dim")

                for signal in assessment.signals:
                    evidence_text = "; ".join(signal.evidence[:2])
                    if len(signal.evidence) > 2:
                        evidence_text += "..."

                    score_color = "red" if signal.score > 0.7 else "yellow" if signal.score > 0.4 else "green"

                    table.add_row(
                        signal.name.replace("_", " ").title(),
                        f"[{score_color}]{signal.score:.2f}[/]",
                        f"{signal.confidence:.2f}",
                        evidence_text
                    )

                console.print(table)

            # Category breakdown (if requested)
            if categories and detector_results:
                console.print(f"\n📋 Risk Categories:", style="bold")

                categories_table = Table(show_header=True, header_style="bold magenta")
                categories_table.add_column("Category", style="cyan")
                categories_table.add_column("Score", justify="center")
                categories_table.add_column("Status", justify="center")
                categories_table.add_column("Top Issue", style="dim")

                category_mapping = {
                    "api_break": ("API Breaking", "api_break"),
                    "schema_risk": ("Schema Changes", "schema_risk"),
                    "dependency_risk": ("Dependencies", "dependency_risk"),
                    "performance_risk": ("Performance", "performance_risk"),
                    "concurrency_risk": ("Concurrency", "concurrency_risk"),
                    "security_risk": ("Security", "security_risk"),
                    "config_risk": ("Configuration", "config_risk"),
                    "test_gap": ("Test Coverage", "test_gap"),
                    "merge_risk": ("Merge Conflicts", "merge_risk")
                }

                for detector_name, (category_name, key) in category_mapping.items():
                    result = detector_results.get(detector_name, {})
                    score = result.get("score", 0.0)
                    reasons = result.get("reasons", [])

                    # Determine status and color
                    if score >= 0.8:
                        status = "🔴 Critical"
                        score_color = "red"
                    elif score >= 0.6:
                        status = "🔶 High"
                        score_color = "orange"
                    elif score >= 0.3:
                        status = "⚠️ Medium"
                        score_color = "yellow"
                    elif score > 0:
                        status = "💡 Low"
                        score_color = "blue"
                    else:
                        status = "✅ Good"
                        score_color = "green"

                    top_issue = reasons[0] if reasons else "No issues detected"
                    if len(top_issue) > 40:
                        top_issue = top_issue[:37] + "..."

                    categories_table.add_row(
                        category_name,
                        f"[{score_color}]{score:.2f}[/]",
                        status,
                        top_issue
                    )

                console.print(categories_table)

            # Detailed explanations (if requested)
            if explain and detector_results:
                console.print(f"\n🔍 Detailed Explanations:", style="bold")

                for detector_name, result in detector_results.items():
                    if result.get("score", 0) > 0 or result.get("reasons"):
                        category_name = detector_name.replace("_", " ").title()
                        console.print(f"\n{category_name}:", style="bold cyan")

                        score = result.get("score", 0.0)
                        console.print(f"  Score: {score:.2f}")

                        reasons = result.get("reasons", [])
                        if reasons:
                            console.print("  Issues:")
                            for reason in reasons[:3]:  # Show top 3 reasons
                                console.print(f"    • {reason}", style="dim")

                        anchors = result.get("anchors", [])
                        if anchors:
                            console.print("  Locations:")
                            for anchor in anchors[:3]:  # Show top 3 locations
                                console.print(f"    📍 {anchor}", style="dim blue")

                        execution_time = result.get("execution_time_ms", 0)
                        console.print(f"  Analysis time: {execution_time:.1f}ms", style="dim")

    # Run the async assessment
    try:
        asyncio.run(run_assessment())
    except KeyboardInterrupt:
        if not output_json:
            console.print("\n❌ Assessment cancelled by user", style="red")
    except Exception as e:
        if output_json:
            console.print(json.dumps({"error": str(e)}))
        else:
            console.print(f"❌ Unexpected error: {e}", style="red")


@cli.command()
@click.argument("commit_sha")
@click.option("--repo", "-r", default=".", help="Repository path (default: current directory)")
@click.option("--json", "output_json", is_flag=True, help="Output results as JSON")
def commit(commit_sha: str, repo: str, output_json: bool):
    """Assess risk of a specific commit"""

    async def run_commit_assessment():
        repo_path = Path(repo).resolve()

        if not repo_path.exists():
            console.print(f"❌ Repository path does not exist: {repo_path}", style="red")
            return

        risk_engine = RiskEngine(str(repo_path))

        try:
            await risk_engine.initialize()
            assessment = await risk_engine.assess_commit_risk(commit_sha)

            if output_json:
                result = {
                    "commit_sha": commit_sha,
                    "tier": assessment.tier.value,
                    "score": round(assessment.score, 1),
                    "explanation": assessment.get_explanation()
                }
                console.print(json.dumps(result, indent=2))
            else:
                print_banner()
                console.print(f"\n🔍 Commit Assessment: {commit_sha[:8]}")
                console.print(f"📊 Risk Score: {assessment.score:.1f}/100 ({assessment.tier.value})")
                console.print(f"📝 {assessment.get_explanation()}")

        except Exception as e:
            if output_json:
                console.print(json.dumps({"error": str(e)}))
            else:
                console.print(f"❌ Error assessing commit {commit_sha}: {e}", style="red")

    try:
        asyncio.run(run_commit_assessment())
    except KeyboardInterrupt:
        if not output_json:
            console.print("\n❌ Assessment cancelled by user", style="red")


@cli.command()
@click.option("--repo", "-r", default=".", help="Repository path (default: current directory)")
def init(repo: str):
    """Initialize CodeRisk for a repository"""

    print_banner()
    console.print(f"🚀 Initializing CodeRisk for repository: {repo}")

    async def run_init():
        repo_path = Path(repo).resolve()

        if not repo_path.exists():
            console.print(f"❌ Repository path does not exist: {repo_path}", style="red")
            return

        with Progress(
            SpinnerColumn(),
            TextColumn("[progress.description]{task.description}"),
            console=console,
            transient=True,
        ) as progress:
            task = progress.add_task("Building CodeGraph...", total=None)

            risk_engine = RiskEngine(str(repo_path))

            try:
                await risk_engine.initialize()
                progress.update(task, description="Initialization complete!", completed=True)

                console.print("✅ CodeRisk initialized successfully!", style="green")
                console.print("💡 You can now run 'crisk check' to assess your changes")

            except Exception as e:
                console.print(f"❌ Initialization failed: {e}", style="red")

    try:
        asyncio.run(run_init())
    except KeyboardInterrupt:
        console.print("\n❌ Initialization cancelled by user", style="red")


def main():
    """Main CLI entry point"""
    cli()


if __name__ == "__main__":
    main()