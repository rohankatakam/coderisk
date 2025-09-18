#!/usr/bin/env python3
"""
Basic usage examples for CodeRisk
"""

import asyncio
import sys
from pathlib import Path

# Add the parent directory to the path to import coderisk
sys.path.insert(0, str(Path(__file__).parent.parent))

from coderisk import RiskEngine


async def main():
    """Basic usage example"""

    # Initialize risk engine for current directory
    engine = RiskEngine(".")
    await engine.initialize()

    # Assess current changes
    print("🔍 Assessing current working tree changes...")
    assessment = await engine.assess_worktree_risk()

    print(f"Risk Level: {assessment.tier.value}")
    print(f"Risk Score: {assessment.score:.1f}/100")
    print(f"Top Concerns: {', '.join(assessment.top_concerns)}")
    print(f"Explanation: {assessment.get_explanation()}")

    # Show scaling factors
    print(f"\nRegression Scaling Factors:")
    print(f"  Team Factor: {assessment.team_factor:.2f}x")
    print(f"  Codebase Factor: {assessment.codebase_factor:.2f}x")
    print(f"  Change Velocity: {assessment.change_velocity:.2f}x")
    print(f"  Migration Multiplier: {assessment.migration_multiplier:.2f}x")
    print(f"  Total Regression Risk: {assessment.total_regression_risk:.2f}x baseline")


if __name__ == "__main__":
    asyncio.run(main())