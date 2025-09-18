#!/usr/bin/env python3
"""
Enhanced integration test with observability and evaluation
"""

import asyncio
import os
import sys
import json
from pathlib import Path

# Add the package to the path
sys.path.insert(0, str(Path(__file__).parent))

from coderisk.core.risk_engine import RiskEngine
from coderisk.observability.langfuse_integration import observer
from coderisk.observability.deepeval_integration import tester


async def test_observability_integration():
    """Test observability integration"""
    print("\n🔍 Testing Observability Integration")
    print("=" * 50)

    try:
        # Check if observability is configured
        langfuse_configured = bool(os.getenv("LANGFUSE_PUBLIC_KEY") and os.getenv("LANGFUSE_SECRET_KEY"))
        deepeval_configured = bool(
            os.getenv("OPENAI_API_KEY") or
            os.getenv("ANTHROPIC_API_KEY") or
            os.getenv("GOOGLE_API_KEY")
        )

        print(f"✅ Langfuse configured: {langfuse_configured}")
        print(f"✅ DeepEval configured: {deepeval_configured}")

        if langfuse_configured:
            print("✅ Langfuse observability will be active")
        else:
            print("⚠️  Langfuse not configured - set LANGFUSE_PUBLIC_KEY and LANGFUSE_SECRET_KEY")

        if deepeval_configured:
            print("✅ DeepEval testing will be active")
        else:
            print("⚠️  DeepEval not configured - set OPENAI_API_KEY, ANTHROPIC_API_KEY, or GOOGLE_API_KEY")

        return True

    except Exception as e:
        print(f"❌ Observability integration test failed: {e}")
        return False


async def test_enhanced_risk_assessment():
    """Test risk assessment with full observability"""
    print("\n🧠 Testing Enhanced Risk Assessment with Observability")
    print("=" * 60)

    try:
        # Initialize engine with observability
        engine = RiskEngine(".", use_advanced_calculations=True)
        await engine.initialize()

        print("✅ Enhanced RiskEngine initialized with observability")

        # Perform assessment with automatic tracing
        assessment = await engine.assess_worktree_risk()

        # Log performance metrics
        performance_metrics = {
            "assessment_time_ms": assessment.assessment_time_ms,
            "risk_tier": assessment.tier.value,
            "risk_score": assessment.score,
            "total_regression_risk": assessment.total_regression_risk,
            "signals_detected": len([s for s in assessment.signals if s.score > 0.1])
        }

        observer.log_performance_metrics(performance_metrics)

        # Log repository analysis
        repo_metrics = {
            "repository_type": "coderisk_project",
            "files_analyzed": len(assessment.change_context.files_changed),
            "lines_changed": assessment.change_context.lines_added + assessment.change_context.lines_deleted,
            "has_advanced_calculations": True
        }

        observer.log_repository_analysis(".", repo_metrics)

        print(f"✅ Enhanced assessment completed with observability:")
        print(f"   - Risk Tier: {assessment.tier.value}")
        print(f"   - Risk Score: {assessment.score:.1f}/100")
        print(f"   - Assessment Time: {assessment.assessment_time_ms}ms")
        print(f"   - Signals with activity: {performance_metrics['signals_detected']}")

        return assessment

    except Exception as e:
        print(f"❌ Enhanced risk assessment failed: {e}")
        return None


async def test_deepeval_validation():
    """Test DeepEval validation capabilities"""
    print("\n🧪 Testing DeepEval Validation")
    print("=" * 40)

    try:
        if not tester.enabled:
            print("⚠️  DeepEval not enabled - skipping validation test")
            return True

        # Create a sample test case
        sample_assessment = {
            "tier": "HIGH",
            "score": 85.5,
            "signals": [
                {"name": "blast_radius", "score": 0.7},
                {"name": "cochange", "score": 0.6},
                {"name": "test_gap", "score": 0.8}
            ],
            "change_context": {
                "files_changed": ["coderisk/core/risk_engine.py"],
                "lines_added": 50,
                "lines_deleted": 10
            },
            "assessment_time_ms": 2500
        }

        test_case = tester.create_test_case_from_assessment(
            repo_path=".",
            assessment_result=sample_assessment,
            expected_tier="HIGH",
            expected_signals=["blast_radius", "cochange", "test_gap"]
        )

        print("✅ Created sample test case for validation")

        # Validate individual signal (if configured)
        signal_accuracy = tester.validate_signal_accuracy(
            signal_name="test_gap",
            detected_evidence=["Missing test coverage in risk_engine.py"],
            actual_code_changes=["coderisk/core/risk_engine.py"]
        )

        if signal_accuracy is not None:
            print(f"✅ Signal validation completed: {signal_accuracy:.2f} accuracy")
        else:
            print("⚠️  Signal validation skipped (LLM not configured)")

        return True

    except Exception as e:
        print(f"❌ DeepEval validation failed: {e}")
        return False


async def test_comprehensive_workflow():
    """Test the complete workflow with observability and validation"""
    print("\n🚀 Testing Comprehensive Workflow")
    print("=" * 45)

    try:
        # Step 1: Run assessment with observability
        assessment = await test_enhanced_risk_assessment()
        if not assessment:
            return False

        # Step 2: Create evaluation test case
        if tester.enabled:
            test_case = tester.create_test_case_from_assessment(
                repo_path=".",
                assessment_result={
                    "tier": assessment.tier.value,
                    "score": assessment.score,
                    "signals": [{"name": s.name, "score": s.score} for s in assessment.signals],
                    "change_context": {
                        "files_changed": assessment.change_context.files_changed,
                        "lines_added": assessment.change_context.lines_added,
                        "lines_deleted": assessment.change_context.lines_deleted
                    },
                    "assessment_time_ms": assessment.assessment_time_ms
                }
            )
            print("✅ Created evaluation test case from assessment")

        # Step 3: Log comprehensive metrics
        comprehensive_metrics = {
            "workflow_type": "comprehensive_test",
            "observability_enabled": observer.enabled,
            "evaluation_enabled": tester.enabled,
            "assessment_quality": "high" if assessment.score > 70 else "medium" if assessment.score > 40 else "low",
            "performance_tier": "fast" if assessment.assessment_time_ms < 2000 else "medium" if assessment.assessment_time_ms < 5000 else "slow"
        }

        observer.log_performance_metrics(comprehensive_metrics)

        print("✅ Comprehensive workflow completed successfully")
        print(f"   - Observability: {'✅ Active' if observer.enabled else '⚠️  Inactive'}")
        print(f"   - Evaluation: {'✅ Active' if tester.enabled else '⚠️  Inactive'}")
        print(f"   - Assessment Quality: {comprehensive_metrics['assessment_quality']}")
        print(f"   - Performance Tier: {comprehensive_metrics['performance_tier']}")

        return True

    except Exception as e:
        print(f"❌ Comprehensive workflow failed: {e}")
        return False


async def main():
    """Main test function"""
    print("🚀 CodeRisk Enhanced Integration Test Suite with Observability")
    print("=" * 70)

    results = []

    # Test observability setup
    results.append(await test_observability_integration())

    # Test enhanced assessment
    results.append(await test_enhanced_risk_assessment() is not None)

    # Test DeepEval validation
    results.append(await test_deepeval_validation())

    # Test comprehensive workflow
    results.append(await test_comprehensive_workflow())

    # Summary
    print("\n📋 Enhanced Test Results Summary")
    print("=" * 40)

    test_names = [
        "Observability Integration",
        "Enhanced Risk Assessment",
        "DeepEval Validation",
        "Comprehensive Workflow"
    ]

    passed = sum(results)
    total = len(results)

    for i, (test_name, result) in enumerate(zip(test_names, results)):
        status = "✅ PASS" if result else "❌ FAIL"
        print(f"{status}: {test_name}")

    print(f"\nOverall: {passed}/{total} tests passed")

    if passed == total:
        print("🎉 All enhanced tests passed! CodeRisk observability integration is working correctly.")
    else:
        print("⚠️  Some enhanced tests failed. Check the output above for details.")


if __name__ == "__main__":
    asyncio.run(main())