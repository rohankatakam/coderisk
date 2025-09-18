#!/usr/bin/env python3
"""
Integration test for the complete CodeRisk system
"""

import asyncio
import os
import sys
from pathlib import Path

# Add the package to the path
sys.path.insert(0, str(Path(__file__).parent))

from coderisk.core.risk_engine import RiskEngine




async def test_advanced_integration():
    """Test full integration with Cognee and mathematical models"""
    print("🧪 Testing Full CodeRisk Integration (Cognee + Mathematical Models)")
    print("=" * 50)

    try:
        # Test with full Cognee integration and advanced calculations
        engine = RiskEngine(".", use_advanced_calculations=True)
        await engine.initialize()

        print("✅ Full RiskEngine initialized successfully")

        # Test assessment
        assessment = await engine.assess_worktree_risk()

        print(f"✅ Full risk assessment completed:")
        print(f"   - Risk Tier: {assessment.tier.value}")
        print(f"   - Risk Score: {assessment.score:.1f}/100")
        print(f"   - Total Regression Risk: {assessment.total_regression_risk:.2f}x baseline")
        print(f"   - Team Factor: {assessment.team_factor:.2f}x")
        print(f"   - Codebase Factor: {assessment.codebase_factor:.2f}x")
        print(f"   - Change Velocity: {assessment.change_velocity:.2f}x")
        print(f"   - Migration Multiplier: {assessment.migration_multiplier:.2f}x")
        print(f"   - Assessment Time: {assessment.assessment_time_ms}ms")

        return True

    except Exception as e:
        print(f"❌ Full integration test failed: {e}")
        return False


async def test_cognee_integration():
    """Test Cognee integration if available"""
    print("\n🧪 Testing Cognee Integration")
    print("=" * 50)

    try:
        # Check if LLM_API_KEY is set
        if not os.getenv('LLM_API_KEY'):
            print("⚠️  LLM_API_KEY not set - skipping Cognee integration test")
            return True

        # Import Cognee modules
        from coderisk.core.cognee_integration import CogneeCodeAnalyzer
        from coderisk.ingestion.git_history_extractor import GitHistoryExtractor
        from coderisk.ingestion.cognee_processor import CogneeKnowledgeProcessor

        print("✅ Cognee modules imported successfully")

        # Test analyzer initialization
        analyzer = CogneeCodeAnalyzer(".", enable_full_ingestion=False)
        await analyzer.initialize()

        print("✅ CogneeCodeAnalyzer initialized successfully")

        # Test capabilities
        capabilities = analyzer.get_capabilities()
        print(f"✅ Capabilities: {capabilities}")

        return True

    except ImportError as e:
        print(f"⚠️  Cognee not available (expected): {e}")
        return True
    except Exception as e:
        print(f"❌ Cognee integration test failed: {e}")
        return False


async def test_detectors():
    """Test micro-detectors"""
    print("\n🧪 Testing Micro-Detectors")
    print("=" * 50)

    try:
        from coderisk.detectors import detector_registry, ChangeContext, FileChange

        # List available detectors
        detectors = detector_registry.list_detectors()
        print(f"✅ Available detectors: {detectors}")

        # Create test context
        context = ChangeContext(
            files_changed=[],
            total_lines_added=0,
            total_lines_deleted=0
        )

        # Test a single detector
        if detectors:
            detector = detector_registry.get_detector(detectors[0], ".")
            result = await detector.run_with_timeout(context)
            print(f"✅ Detector '{detectors[0]}' executed successfully:")
            print(f"   - Score: {result.score:.3f}")
            print(f"   - Execution Time: {result.execution_time_ms:.1f}ms")

        return True

    except Exception as e:
        print(f"❌ Detectors test failed: {e}")
        return False


async def test_calculations():
    """Test calculation engine"""
    print("\n🧪 Testing Calculation Engine")
    print("=" * 50)

    try:
        from coderisk.calculations import (
            RegressionScalingModel,
            BlastRadiusSignal,
            ScoringEngine
        )

        print("✅ Calculation modules imported successfully")

        # Test regression scaling
        scaling_model = RegressionScalingModel(".")
        scaling_model.initialize()

        factors = scaling_model.calculate_scaling_factors(["test.py"], "test commit")
        print(f"✅ Regression scaling calculated:")
        print(f"   - Team Factor: {factors.team_factor:.2f}")
        print(f"   - Codebase Factor: {factors.codebase_factor:.2f}")
        print(f"   - Change Velocity: {factors.change_velocity:.2f}")
        print(f"   - Migration Multiplier: {factors.migration_multiplier:.2f}")

        return True

    except Exception as e:
        print(f"❌ Calculations test failed: {e}")
        return False


async def main():
    """Run all integration tests"""
    print("🚀 CodeRisk Integration Test Suite")
    print("=" * 50)

    tests = [
        ("Full Integration", test_advanced_integration),
        ("Micro-Detectors", test_detectors),
        ("Calculation Engine", test_calculations),
        ("Cognee Integration", test_cognee_integration),
    ]

    results = []

    for test_name, test_func in tests:
        try:
            success = await test_func()
            results.append((test_name, success))
        except Exception as e:
            print(f"❌ Test '{test_name}' crashed: {e}")
            results.append((test_name, False))

    # Summary
    print("\n📋 Test Results Summary")
    print("=" * 50)

    passed = 0
    for test_name, success in results:
        status = "✅ PASS" if success else "❌ FAIL"
        print(f"{status}: {test_name}")
        if success:
            passed += 1

    print(f"\nOverall: {passed}/{len(results)} tests passed")

    if passed == len(results):
        print("🎉 All tests passed! CodeRisk integration is working correctly.")
    else:
        print("⚠️  Some tests failed. Check the output above for details.")

    return passed == len(results)


if __name__ == "__main__":
    success = asyncio.run(main())
    sys.exit(0 if success else 1)