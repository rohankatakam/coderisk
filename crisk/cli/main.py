"""
CodeRisk CLI - Main command interface
"""

import asyncio
import click
import json
from pathlib import Path
from rich.console import Console
from rich.table import Table
from rich.panel import Panel
from rich.text import Text
from rich.progress import Progress, SpinnerColumn, TextColumn

from ..core.risk_engine import RiskEngine
from ..models.risk_assessment import RiskTier

console = Console()


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
def check(repo: str, output_json: bool, verbose: bool):
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