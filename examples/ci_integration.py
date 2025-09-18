#!/usr/bin/env python3
"""
CI/CD integration example for CodeRisk
"""

import asyncio
import json
import sys
from pathlib import Path

# Add the parent directory to the path to import coderisk
sys.path.insert(0, str(Path(__file__).parent.parent))

from coderisk import RiskEngine


async def ci_risk_check(repo_path: str, max_score: float = 75.0):
    """
    CI/CD integration example that fails the build on high risk

    Args:
        repo_path: Path to the repository
        max_score: Maximum allowed risk score (0-100)

    Returns:
        0 if risk is acceptable, 1 if risk is too high
    """

    try:
        engine = RiskEngine(repo_path)
        await engine.initialize()

        assessment = await engine.assess_worktree_risk()

        # Output results in a format suitable for CI
        result = {
            "risk_score": assessment.score,
            "risk_tier": assessment.tier.value,
            "passed": assessment.score <= max_score,
            "total_regression_risk": assessment.total_regression_risk,
            "top_concerns": assessment.top_concerns,
            "recommendations": [
                {
                    "action": rec.action,
                    "priority": rec.priority,
                    "description": rec.description
                } for rec in assessment.recommendations
            ]
        }

        print(json.dumps(result, indent=2))

        if assessment.score > max_score:
            print(f"\n❌ RISK CHECK FAILED: Score {assessment.score:.1f} exceeds threshold {max_score}", file=sys.stderr)
            print(f"Top concerns: {', '.join(assessment.top_concerns)}", file=sys.stderr)
            return 1
        else:
            print(f"\n✅ RISK CHECK PASSED: Score {assessment.score:.1f} is within threshold {max_score}", file=sys.stderr)
            return 0

    except Exception as e:
        print(f"❌ ERROR: {e}", file=sys.stderr)
        return 1


async def main():
    """Main entry point"""
    repo_path = sys.argv[1] if len(sys.argv) > 1 else "."
    max_score = float(sys.argv[2]) if len(sys.argv) > 2 else 75.0

    exit_code = await ci_risk_check(repo_path, max_score)
    sys.exit(exit_code)


if __name__ == "__main__":
    asyncio.run(main())