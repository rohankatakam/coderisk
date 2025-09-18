#!/usr/bin/env python3
"""
Test CodeRisk system on medium-sized repository (topoteretes/cognee)
Following the testing guide for comprehensive evaluation
"""

import asyncio
import os
import sys
import time
import json
from pathlib import Path

# Add the package to the path
sys.path.insert(0, str(Path(__file__).parent))

from coderisk.core.risk_engine import RiskEngine
from coderisk.observability.langfuse_integration import observer
from coderisk.observability.deepeval_integration import tester


async def analyze_repository_structure(repo_path: str):
    """Analyze the structure of the medium repository"""
    print("📁 Analyzing Repository Structure")
    print("-" * 50)

    try:
        repo_path = Path(repo_path)

        # Count files by type
        file_counts = {}
        total_files = 0
        total_size = 0

        for file_path in repo_path.rglob("*"):
            if file_path.is_file() and not file_path.parts[0].startswith('.'):
                suffix = file_path.suffix.lower() or 'no_extension'
                file_counts[suffix] = file_counts.get(suffix, 0) + 1
                total_files += 1
                total_size += file_path.stat().st_size

        # Get top file types
        top_types = sorted(file_counts.items(), key=lambda x: x[1], reverse=True)[:10]

        print(f"📊 Repository Statistics:")
        print(f"   - Total files: {total_files:,}")
        print(f"   - Total size: {total_size / (1024*1024):.1f} MB")
        print(f"   - Top file types:")

        for ext, count in top_types:
            percentage = (count / total_files) * 100
            print(f"     {ext}: {count} files ({percentage:.1f}%)")

        # Log repository analysis
        observer.log_repository_analysis(str(repo_path), {
            "total_files": total_files,
            "total_size_mb": total_size / (1024*1024),
            "file_types": dict(top_types),
            "repository_name": "cognee",
            "analysis_type": "structure"
        })

        return {
            "total_files": total_files,
            "total_size_mb": total_size / (1024*1024),
            "top_file_types": dict(top_types)
        }

    except Exception as e:
        print(f"❌ Repository analysis failed: {e}")
        return {}


async def test_risk_assessment_on_medium_repo(repo_path: str):
    """Test risk assessment on the medium repository"""
    print(f"\n🎯 Testing Risk Assessment on Medium Repository")
    print("-" * 60)

    try:
        # Change to repository directory
        original_dir = os.getcwd()
        os.chdir(repo_path)

        print(f"📂 Working in: {repo_path}")

        # Initialize risk engine
        start_time = time.time()
        engine = RiskEngine(".", use_advanced_calculations=True)
        await engine.initialize()
        init_time = (time.time() - start_time) * 1000

        print(f"✅ RiskEngine initialized ({init_time:.1f}ms)")

        # Perform risk assessment
        start_time = time.time()
        assessment = await engine.assess_worktree_risk()
        assessment_time = (time.time() - start_time) * 1000

        print(f"✅ Risk assessment completed:")
        print(f"   - Risk Tier: {assessment.tier.value}")
        print(f"   - Risk Score: {assessment.score:.1f}/100")
        print(f"   - Assessment Time: {assessment_time:.1f}ms")
        print(f"   - Reported Time: {assessment.assessment_time_ms}ms")

        # Analyze signals
        active_signals = [s for s in assessment.signals if s.score > 0.1]
        print(f"   - Active Signals: {len(active_signals)}")

        for signal in active_signals[:5]:  # Show top 5
            print(f"     • {signal.name}: {signal.score:.3f} (confidence: {signal.confidence:.3f})")

        # Log comprehensive metrics
        medium_repo_metrics = {
            "repository_name": "cognee",
            "repository_type": "medium_python_ai",
            "initialization_time_ms": init_time,
            "assessment_time_ms": assessment_time,
            "risk_tier": assessment.tier.value,
            "risk_score": assessment.score,
            "total_regression_risk": assessment.total_regression_risk,
            "active_signals_count": len(active_signals),
            "files_analyzed": len(assessment.change_context.files_changed),
            "lines_changed": assessment.change_context.lines_added + assessment.change_context.lines_deleted,
            "test_type": "medium_repository_assessment"
        }

        observer.log_performance_metrics(medium_repo_metrics)

        # Create evaluation test case
        if tester.enabled:
            test_case = tester.create_test_case_from_assessment(
                repo_path=repo_path,
                assessment_result={
                    "tier": assessment.tier.value,
                    "score": assessment.score,
                    "signals": [{"name": s.name, "score": s.score, "confidence": s.confidence} for s in assessment.signals],
                    "assessment_time_ms": assessment.assessment_time_ms,
                    "repository_metrics": medium_repo_metrics
                },
                expected_tier="MEDIUM",  # Expected for medium-sized AI repository
                expected_signals=["dependency_risk", "test_gap", "merge_risk"]
            )
            print("✅ Created evaluation test case for medium repository")

        return assessment, medium_repo_metrics

    except Exception as e:
        print(f"❌ Medium repository assessment failed: {e}")
        return None, {}

    finally:
        # Return to original directory
        os.chdir(original_dir)


async def test_performance_on_medium_repo(repo_path: str):
    """Test performance characteristics on medium repository"""
    print(f"\n⚡ Testing Performance on Medium Repository")
    print("-" * 55)

    try:
        original_dir = os.getcwd()
        os.chdir(repo_path)

        performance_runs = []

        # Run multiple assessments to get performance statistics
        for run in range(3):
            print(f"🏃 Performance run {run + 1}/3...")

            start_time = time.time()
            engine = RiskEngine(".", use_advanced_calculations=True)
            await engine.initialize()
            init_time = time.time() - start_time

            start_time = time.time()
            assessment = await engine.assess_worktree_risk()
            assess_time = time.time() - start_time

            performance_runs.append({
                "run": run + 1,
                "init_time_s": init_time,
                "assessment_time_s": assess_time,
                "total_time_s": init_time + assess_time,
                "risk_score": assessment.score
            })

            print(f"   - Run {run + 1}: {assess_time:.2f}s assessment, {assessment.score:.1f} score")

        # Calculate statistics
        avg_init = sum(r["init_time_s"] for r in performance_runs) / len(performance_runs)
        avg_assess = sum(r["assessment_time_s"] for r in performance_runs) / len(performance_runs)
        avg_total = sum(r["total_time_s"] for r in performance_runs) / len(performance_runs)

        print(f"📊 Performance Statistics:")
        print(f"   - Average initialization: {avg_init:.2f}s")
        print(f"   - Average assessment: {avg_assess:.2f}s")
        print(f"   - Average total: {avg_total:.2f}s")

        # Performance assessment
        performance_grade = "EXCELLENT" if avg_assess < 2.0 else "GOOD" if avg_assess < 5.0 else "NEEDS_IMPROVEMENT"
        print(f"   - Performance Grade: {performance_grade}")

        # Log performance statistics
        observer.log_performance_metrics({
            "repository_name": "cognee",
            "performance_runs": len(performance_runs),
            "avg_initialization_time_s": avg_init,
            "avg_assessment_time_s": avg_assess,
            "avg_total_time_s": avg_total,
            "performance_grade": performance_grade,
            "test_type": "medium_repository_performance"
        })

        return performance_runs

    except Exception as e:
        print(f"❌ Performance testing failed: {e}")
        return []

    finally:
        os.chdir(original_dir)


async def test_database_operations_on_medium_repo(repo_path: str):
    """Test database operations with medium repository"""
    print(f"\n💾 Testing Database Operations on Medium Repository")
    print("-" * 60)

    try:
        original_dir = os.getcwd()
        os.chdir(repo_path)

        # Test database initialization with larger dataset
        start_time = time.time()
        engine = RiskEngine(".", use_advanced_calculations=True)
        await engine.initialize()
        db_init_time = (time.time() - start_time) * 1000

        print(f"✅ Database initialized for medium repo ({db_init_time:.1f}ms)")

        # Check database files after medium repo processing
        db_path = os.path.expanduser('~/.pyenv/versions/3.11.9/lib/python3.11/site-packages/cognee/.cognee_system/databases')
        if os.path.exists(db_path):
            db_files = []
            total_db_size = 0
            for db_file in os.listdir(db_path):
                if db_file.endswith('.db'):
                    file_path = os.path.join(db_path, db_file)
                    size = os.path.getsize(file_path) / 1024  # KB
                    total_db_size += size
                    db_files.append({"name": db_file, "size_kb": size})
                    print(f'  📁 {db_file}: {size:.1f} KB')

            print(f"📊 Database metrics:")
            print(f"   - Total database files: {len(db_files)}")
            print(f"   - Total database size: {total_db_size:.1f} KB")

            # Log database metrics
            observer.log_performance_metrics({
                "repository_name": "cognee",
                "database_files_count": len(db_files),
                "total_database_size_kb": total_db_size,
                "database_init_time_ms": db_init_time,
                "test_type": "medium_repository_database"
            })

        return True

    except Exception as e:
        print(f"❌ Database operations test failed: {e}")
        return False

    finally:
        os.chdir(original_dir)


async def generate_medium_repo_report(repo_structure, assessment_result, performance_data):
    """Generate comprehensive report for medium repository testing"""
    print(f"\n📋 Generating Comprehensive Medium Repository Report")
    print("-" * 65)

    try:
        assessment, metrics = assessment_result if assessment_result else (None, {})

        report = {
            "repository_info": {
                "name": "cognee (topoteretes/cognee)",
                "type": "Medium-sized AI/ML Python project",
                "size_category": "Medium",
                **repo_structure
            },
            "risk_assessment": {
                "tier": assessment.tier.value if assessment else "UNKNOWN",
                "score": assessment.score if assessment else 0,
                "total_regression_risk": assessment.total_regression_risk if assessment else 0,
                "assessment_time_ms": metrics.get("assessment_time_ms", 0),
                "signals_detected": metrics.get("active_signals_count", 0)
            },
            "performance_analysis": {
                "runs_completed": len(performance_data),
                "avg_assessment_time": sum(r["assessment_time_s"] for r in performance_data) / len(performance_data) if performance_data else 0,
                "performance_consistent": len(set(r["risk_score"] for r in performance_data)) <= 2 if performance_data else False
            },
            "system_capabilities": {
                "observability_enabled": observer.enabled,
                "evaluation_enabled": tester.enabled,
                "advanced_calculations": True,
                "cognee_integration": metrics.get("files_analyzed", 0) > 0
            }
        }

        print("📊 Medium Repository Test Summary:")
        print(f"   Repository: {report['repository_info']['name']}")
        print(f"   Files: {report['repository_info'].get('total_files', 0):,}")
        print(f"   Size: {report['repository_info'].get('total_size_mb', 0):.1f} MB")
        print(f"   Risk Assessment: {report['risk_assessment']['tier']} ({report['risk_assessment']['score']:.1f}/100)")
        print(f"   Performance: {report['performance_analysis']['avg_assessment_time']:.2f}s average")
        print(f"   Observability: {'✅ Active' if report['system_capabilities']['observability_enabled'] else '⚠️ Inactive'}")

        # Log final report
        observer.log_performance_metrics({
            **report,
            "test_type": "medium_repository_final_report"
        })

        # Create final evaluation if enabled
        if tester.enabled and assessment:
            final_evaluation = tester.create_test_case_from_assessment(
                repo_path="/tmp/test_repos/cognee",
                assessment_result=report,
                expected_tier="MEDIUM",
                expected_signals=["performance_risk", "dependency_risk", "test_gap"]
            )
            print("✅ Created final evaluation test case")

        return report

    except Exception as e:
        print(f"❌ Report generation failed: {e}")
        return {}


async def main():
    """Main test function for medium repository"""
    print("🚀 CodeRisk Medium Repository Testing (topoteretes/cognee)")
    print("=" * 70)

    repo_path = "/tmp/test_repos/cognee"

    # Verify repository exists
    if not os.path.exists(repo_path):
        print(f"❌ Repository not found at {repo_path}")
        print("Please ensure the repository is cloned first.")
        return

    try:
        # Test sequence
        print("🔍 Phase 1: Repository Analysis")
        repo_structure = await analyze_repository_structure(repo_path)

        print("\n🎯 Phase 2: Risk Assessment")
        assessment_result = await test_risk_assessment_on_medium_repo(repo_path)

        print("\n⚡ Phase 3: Performance Testing")
        performance_data = await test_performance_on_medium_repo(repo_path)

        print("\n💾 Phase 4: Database Operations")
        db_result = await test_database_operations_on_medium_repo(repo_path)

        print("\n📋 Phase 5: Comprehensive Report")
        final_report = await generate_medium_repo_report(repo_structure, assessment_result, performance_data)

        # Final summary
        print("\n🏁 Medium Repository Testing Complete")
        print("=" * 50)

        success_count = sum([
            bool(repo_structure),
            bool(assessment_result[0] if assessment_result else False),
            bool(performance_data),
            bool(db_result),
            bool(final_report)
        ])

        print(f"✅ Completed phases: {success_count}/5")

        if success_count >= 4:
            print("🎉 Medium repository testing successful!")
            print("💡 CodeRisk system handles medium-sized repositories effectively.")
        else:
            print("⚠️ Some phases had issues. Review the detailed output above.")

        print("\n📚 Observability Summary:")
        print(f"   - Langfuse tracing: {'✅ Active' if observer.enabled else '⚠️ Configure LANGFUSE_PUBLIC_KEY & LANGFUSE_SECRET_KEY'}")
        print(f"   - DeepEval testing: {'✅ Active' if tester.enabled else '⚠️ Configure OPENAI_API_KEY or similar'}")

    except Exception as e:
        print(f"❌ Medium repository testing failed: {e}")


if __name__ == "__main__":
    asyncio.run(main())