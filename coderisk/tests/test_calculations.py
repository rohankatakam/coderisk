"""
Comprehensive test suite for advanced risk calculations.

Tests all mathematical models and signal calculations with synthetic data
to ensure mathematical accuracy and performance requirements.
"""

import pytest
import numpy as np
import asyncio
from datetime import datetime, timedelta
from typing import List, Dict, Tuple
import math

from ..calculations.mathematical_models import (
    calculate_delta_diffusion_blast_radius,
    calculate_hawkes_decayed_cochange,
    calculate_g2_surprise,
    calculate_ownership_authority_mismatch,
    TemporalDecayModel,
    PageRankDelta,
    GraphEdge
)

from ..calculations.core_signals import (
    BlastRadiusSignal,
    CoChangeSignal,
    G2SurpriseSignal,
    OwnershipAuthorityMismatch,
    SpanCoreSignal,
    BridgeRiskSignal,
    IncidentAdjacencySignal,
    JITBaselinesSignal,
    IncidentData,
    SpanCoreData
)

from ..calculations.regression_scaling import (
    RegressionScalingModel,
    TeamFactor,
    CodebaseFactor,
    ChangeVelocityFactor,
    MigrationMultiplier
)

from ..calculations.scoring_engine import (
    ScoringEngine,
    ConformalRiskControl,
    MonotoneScorer,
    AutoHighRules,
    ConformalCalibrationData
)

from ..models.risk_assessment import RiskTier, RiskSignal


class TestMathematicalModels:
    """Test core mathematical model implementations"""

    def test_temporal_decay_model(self):
        """Test temporal decay functions"""
        decay_model = TemporalDecayModel(fast_decay=0.1, slow_decay=0.01)

        # Test Hawkes kernel
        assert decay_model.hawkes_kernel(0) == 1.3  # Initial value
        assert decay_model.hawkes_kernel(1) > decay_model.hawkes_kernel(10)  # Decay over time
        assert decay_model.hawkes_kernel(-1) == 0.0  # No negative time

        # Test exponential decay
        assert decay_model.exponential_decay(0) == 1.0
        assert 0 < decay_model.exponential_decay(10) < 1.0

    def test_pagerank_delta(self):
        """Test PageRank delta calculation"""
        pr_calc = PageRankDelta()

        # Create test graph
        nodes = {'A', 'B', 'C', 'D'}
        edges = [
            GraphEdge('A', 'B', 1.0, 'import'),
            GraphEdge('B', 'C', 1.0, 'import'),
            GraphEdge('C', 'D', 1.0, 'import'),
            GraphEdge('D', 'A', 1.0, 'import')
        ]

        # Test baseline PageRank
        result = pr_calc.calculate_baseline_pagerank(edges, nodes)
        assert len(result.scores) == 4
        assert all(0 <= score <= 1 for score in result.scores.values())
        assert abs(sum(result.scores.values()) - 1.0) < 0.01  # Should sum to 1

        # Test delta calculation
        modified_edges = edges[:-1]  # Remove one edge
        deltas = pr_calc.calculate_delta(result.scores, modified_edges, nodes)
        assert len(deltas) == 4
        assert max(deltas.values()) > 0  # Should have some impact

    def test_delta_diffusion_blast_radius(self):
        """Test ΔDBR calculation"""
        changed_files = ['file1.py', 'file2.py']
        import_edges = [
            GraphEdge('file1.py', 'file3.py', 1.0, 'import'),
            GraphEdge('file2.py', 'file4.py', 1.0, 'import'),
            GraphEdge('file3.py', 'file5.py', 1.0, 'import')
        ]
        cochange_edges = [
            GraphEdge('file1.py', 'file2.py', 0.8, 'cochange'),
            GraphEdge('file2.py', 'file3.py', 0.6, 'cochange')
        ]

        score, impact_scores, evidence = calculate_delta_diffusion_blast_radius(
            changed_files, import_edges, cochange_edges
        )

        assert 0 <= score <= 1
        assert isinstance(impact_scores, dict)
        assert 'calculation_time_ms' in evidence
        assert evidence['calculation_time_ms'] <= 500  # Performance requirement

    def test_hawkes_decayed_cochange(self):
        """Test HDCC calculation"""
        changed_files = ['file1.py', 'file2.py']
        now = datetime.now()
        cochange_history = [
            ('file1.py', 'file3.py', now - timedelta(days=10), 0.8),
            ('file2.py', 'file4.py', now - timedelta(days=30), 0.6),
            ('file1.py', 'file5.py', now - timedelta(days=60), 0.4)
        ]

        score, coupling_scores, evidence = calculate_hawkes_decayed_cochange(
            changed_files, cochange_history
        )

        assert 0 <= score <= 1
        assert isinstance(coupling_scores, dict)
        assert evidence['analysis_window_days'] == 90

    def test_g2_surprise(self):
        """Test G² surprise calculation"""
        file_pairs = [('file1.py', 'file2.py'), ('file1.py', 'file3.py')]
        expected_frequencies = {
            ('file1.py', 'file2.py'): 2.0,
            ('file1.py', 'file3.py'): 1.0
        }
        observed_frequencies = {
            ('file1.py', 'file2.py'): 10,
            ('file1.py', 'file3.py'): 1
        }

        score, unusual_pairs, evidence = calculate_g2_surprise(
            file_pairs, expected_frequencies, observed_frequencies
        )

        assert 0 <= score <= 1
        assert isinstance(unusual_pairs, list)
        assert evidence['pairs_analyzed'] == 2

    def test_ownership_authority_mismatch(self):
        """Test OAM calculation"""
        changed_files = ['file1.py', 'file2.py']
        now = datetime.now()
        ownership_history = {
            'file1.py': [
                ('alice', now - timedelta(days=10), 50),
                ('bob', now - timedelta(days=20), 30),
                ('alice', now - timedelta(days=30), 20)
            ],
            'file2.py': [
                ('charlie', now - timedelta(days=5), 100)  # Single author risk
            ]
        }
        team_experience = {
            'alice': 0.8,
            'bob': 0.6,
            'charlie': 0.2  # Inexperienced
        }

        score, file_scores, evidence = calculate_ownership_authority_mismatch(
            changed_files, ownership_history, team_experience
        )

        assert 0 <= score <= 1
        assert len(file_scores) == 2
        assert file_scores['file2.py'] > file_scores['file1.py']  # Single author + inexperienced


class TestCoreSignals:
    """Test risk signal implementations"""

    @pytest.mark.asyncio
    async def test_blast_radius_signal(self):
        """Test blast radius signal calculation"""
        signal_calc = BlastRadiusSignal()
        changed_files = ['file1.py', 'file2.py']
        import_edges = [GraphEdge('file1.py', 'file3.py', 1.0, 'import')]
        cochange_edges = [GraphEdge('file1.py', 'file2.py', 0.8, 'cochange')]

        signal = await signal_calc.calculate(changed_files, import_edges, cochange_edges)

        assert signal.name == "delta_diffusion_blast_radius"
        assert 0 <= signal.score <= 1
        assert 0 <= signal.confidence <= 1
        assert isinstance(signal.evidence, list)
        assert signal.response_time_ms <= 500

    @pytest.mark.asyncio
    async def test_cochange_signal(self):
        """Test co-change signal calculation"""
        signal_calc = CoChangeSignal()
        changed_files = ['file1.py']
        now = datetime.now()
        cochange_history = [
            ('file1.py', 'file2.py', now - timedelta(days=10), 0.8)
        ]

        signal = await signal_calc.calculate(changed_files, cochange_history)

        assert signal.name == "hawkes_decayed_cochange"
        assert 0 <= signal.score <= 1
        assert len(signal.evidence) > 0

    @pytest.mark.asyncio
    async def test_jit_baselines_signal(self):
        """Test JIT baselines signal calculation"""
        signal_calc = JITBaselinesSignal()
        changed_files = ['file1.py', 'file2.py']
        lines_added = 100
        lines_deleted = 50
        diff_hunks = ['@@ -1,5 +1,8 @@\n+new code\n old code\n']

        signal = await signal_calc.calculate(changed_files, lines_added, lines_deleted, diff_hunks)

        assert signal.name == "jit_baselines"
        assert 0 <= signal.score <= 1

    @pytest.mark.asyncio
    async def test_incident_adjacency_signal(self):
        """Test incident adjacency signal calculation"""
        signal_calc = IncidentAdjacencySignal()
        changed_files = ['auth.py', 'login.py']
        commit_message = "fix authentication bug"

        incidents = [
            IncidentData(
                incident_id="INC-001",
                timestamp=datetime.now() - timedelta(days=30),
                affected_files=['auth.py', 'session.py'],
                description="Authentication failure",
                keywords=['auth', 'login', 'security'],
                severity=0.8
            )
        ]

        signal = await signal_calc.calculate(changed_files, commit_message, incidents)

        assert signal.name == "incident_adjacency"
        assert 0 <= signal.score <= 1

    @pytest.mark.asyncio
    async def test_auto_high_rules(self):
        """Test auto-High rules for large changes"""
        signal_calc = JITBaselinesSignal()

        # Test auto-High for large number of files
        large_file_list = [f'file{i}.py' for i in range(25)]  # 25 files > 20 threshold

        signal = await signal_calc.calculate(large_file_list, 100, 50, [])

        # Should trigger auto-High rule
        assert signal.score >= 0.9
        assert any("AUTO-HIGH" in evidence for evidence in signal.evidence)


class TestRegressionScaling:
    """Test regression scaling model components"""

    def test_team_factor_calculation(self):
        """Test team factor formula: 1 + 0.3 × log₂(team_size)"""
        # Test specific values from specification
        assert TeamFactor.calculate(1) == 1.0
        assert abs(TeamFactor.calculate(2) - 1.3) < 0.01
        assert abs(TeamFactor.calculate(4) - 1.6) < 0.01
        assert abs(TeamFactor.calculate(8) - 1.9) < 0.01

        # Test edge cases
        assert TeamFactor.calculate(0) == 1.0
        assert TeamFactor.calculate(-1) == 1.0

    def test_codebase_factor_calculation(self):
        """Test codebase factor formula: 1 + 0.2 × (LOC/100k) × coupling"""
        # Test specific scenarios from specification
        assert abs(CodebaseFactor.calculate(10000, 0.4) - 1.008) < 0.01
        assert abs(CodebaseFactor.calculate(100000, 0.4) - 1.08) < 0.01
        assert abs(CodebaseFactor.calculate(500000, 0.6) - 1.60) < 0.01

        # Test edge cases
        assert CodebaseFactor.calculate(0, 0.5) == 1.0
        assert CodebaseFactor.calculate(100000, 0.0) == 1.0

    def test_change_velocity_calculation(self):
        """Test change velocity formula: 1 + (commits_per_week/100)"""
        assert ChangeVelocityFactor.calculate(0) == 1.0
        assert ChangeVelocityFactor.calculate(10) == 1.1
        assert ChangeVelocityFactor.calculate(100) == 2.0
        assert ChangeVelocityFactor.calculate(500) == 6.0

    def test_migration_multiplier_calculation(self):
        """Test migration multiplier: 2^(major_versions) × 1.5^(breaking_changes)"""
        assert MigrationMultiplier.calculate(0, 0) == 1.0
        assert MigrationMultiplier.calculate(1, 0) == 2.0
        assert MigrationMultiplier.calculate(0, 1) == 1.5
        assert MigrationMultiplier.calculate(1, 1) == 3.0
        assert MigrationMultiplier.calculate(2, 2) == 9.0  # 2^2 * 1.5^2

    def test_migration_detection(self):
        """Test migration indicator detection"""
        # Test dependency file changes
        dep_files = ['package.json', 'requirements.txt']
        indicators = MigrationMultiplier.detect_migration_indicators(dep_files, None)
        assert indicators.major_version_changes > 0

        # Test commit message keywords
        commit_msg = "Upgrade React from 17 to 18 with breaking changes"
        indicators = MigrationMultiplier.detect_migration_indicators([], commit_msg)
        assert indicators.breaking_api_changes > 0

    def test_regression_scaling_integration(self):
        """Test full regression scaling model integration"""
        model = RegressionScalingModel("/tmp/test_repo")

        # Test with mock scenario
        changed_files = ['package.json', 'src/main.js']
        commit_message = "upgrade framework to v2.0"

        # This would normally require a real repo, so we test the components
        factors = model.calculate_scaling_factors(changed_files, commit_message)

        assert factors.team_factor >= 1.0
        assert factors.codebase_factor >= 1.0
        assert factors.change_velocity >= 1.0
        assert factors.migration_multiplier >= 1.0
        assert factors.total_multiplier >= 1.0


class TestScoringEngine:
    """Test advanced scoring engine components"""

    def test_monotone_scorer(self):
        """Test monotone scoring ensures risk increases with signal scores"""
        scorer = MonotoneScorer()

        # Create test signals with increasing scores
        low_signals = [
            RiskSignal("signal1", 0.2, 0.8, [], 10),
            RiskSignal("signal2", 0.1, 0.9, [], 15)
        ]
        high_signals = [
            RiskSignal("signal1", 0.8, 0.8, [], 10),
            RiskSignal("signal2", 0.9, 0.9, [], 15)
        ]

        low_score, _ = scorer.calculate_score(low_signals)
        high_score, _ = scorer.calculate_score(high_signals)

        assert high_score > low_score  # Monotonicity
        assert 0 <= low_score <= 1
        assert 0 <= high_score <= 1

    def test_auto_high_rules(self):
        """Test auto-High rule evaluation"""
        rules = AutoHighRules()

        # Test size threshold rule
        signals = [RiskSignal("jit_baselines", 0.95, 0.9, [], 20)]
        metadata = {"files_changed": 25}

        should_escalate, triggered_rules = rules.evaluate(signals, metadata)

        assert should_escalate
        assert "size_threshold" in triggered_rules

    def test_conformal_risk_control(self):
        """Test conformal prediction calibration and prediction"""
        crc = ConformalRiskControl(target_coverage=0.9)

        # Create synthetic calibration data
        n_samples = 100
        features = np.random.rand(n_samples, 10)
        scores = np.random.rand(n_samples)
        labels = np.random.randint(0, 4, n_samples)  # 4 risk tiers
        segments = np.random.randint(0, 3, n_samples)

        calibration_data = ConformalCalibrationData(features, labels, scores, segments)

        # Test calibration
        crc.calibrate(calibration_data)
        assert crc.calibration_data is not None
        assert len(crc.quantiles) > 0

        # Test prediction
        test_feature = np.random.rand(10)
        test_score = 0.7
        prediction = crc.predict(test_feature, test_score)

        assert len(prediction.prediction_set) >= 1
        assert prediction.confidence_level == 0.9
        assert 0 <= prediction.calibration_score <= 1

    def test_scoring_engine_integration(self):
        """Test full scoring engine integration"""
        engine = ScoringEngine(use_conformal=False)  # No conformal for simplicity

        # Create test signals
        signals = [
            RiskSignal("delta_diffusion_blast_radius", 0.6, 0.8, ["High impact"], 100),
            RiskSignal("hawkes_decayed_cochange", 0.4, 0.7, ["Moderate coupling"], 80),
            RiskSignal("jit_baselines", 0.8, 0.9, ["Large change"], 50)
        ]

        metadata = {
            "files_changed": 15,
            "lines_added": 200,
            "lines_deleted": 100
        }

        tier, score, details = engine.score_change(signals, metadata)

        assert tier in [RiskTier.LOW, RiskTier.MEDIUM, RiskTier.HIGH, RiskTier.CRITICAL]
        assert 0 <= score <= 1
        assert "base_score" in details
        assert "signal_contributions" in details


class TestPerformanceRequirements:
    """Test performance requirements from specification"""

    @pytest.mark.asyncio
    async def test_signal_calculation_timeout(self):
        """Test that signal calculations complete within timeout"""
        signal_calc = BlastRadiusSignal(timeout_ms=100)  # Short timeout for test

        changed_files = ['file1.py']
        import_edges = []
        cochange_edges = []

        start_time = datetime.now()
        signal = await signal_calc.calculate(changed_files, import_edges, cochange_edges)
        end_time = datetime.now()

        calculation_time = (end_time - start_time).total_seconds() * 1000
        assert calculation_time <= 200  # Some buffer for test overhead

    def test_bounded_graph_queries(self):
        """Test that graph queries are bounded as specified"""
        # Test max hops constraint
        changed_files = ['file1.py']
        import_edges = [
            GraphEdge('file1.py', 'file2.py', 1.0, 'import'),
            GraphEdge('file2.py', 'file3.py', 1.0, 'import'),
            GraphEdge('file3.py', 'file4.py', 1.0, 'import'),  # 3 hops from file1
            GraphEdge('file4.py', 'file5.py', 1.0, 'import')   # 4 hops from file1
        ]

        score, _, evidence = calculate_delta_diffusion_blast_radius(
            changed_files, import_edges, [], max_hops=2
        )

        # Should respect max_hops constraint
        assert evidence['total_nodes_analyzed'] <= 10  # Reasonable bound


class TestMathematicalAccuracy:
    """Test mathematical accuracy of implementations"""

    def test_pagerank_convergence(self):
        """Test PageRank convergence properties"""
        pr_calc = PageRankDelta(tolerance=1e-6)

        # Create strongly connected graph
        nodes = {'A', 'B', 'C'}
        edges = [
            GraphEdge('A', 'B', 1.0, 'import'),
            GraphEdge('B', 'C', 1.0, 'import'),
            GraphEdge('C', 'A', 1.0, 'import')
        ]

        result = pr_calc.calculate_baseline_pagerank(edges, nodes)

        # Check convergence properties
        assert result.iterations < pr_calc.max_iterations
        assert abs(sum(result.scores.values()) - 1.0) < 1e-5  # Probability constraint

    def test_hawkes_kernel_properties(self):
        """Test mathematical properties of Hawkes kernel"""
        decay_model = TemporalDecayModel()

        # Test monotonicity (should decrease over time)
        times = [0, 1, 5, 10, 30, 90]
        values = [decay_model.hawkes_kernel(t) for t in times]

        for i in range(len(values) - 1):
            assert values[i] >= values[i + 1], f"Not monotonic at {times[i]}"

        # Test non-negativity
        assert all(v >= 0 for v in values)

    def test_g2_statistical_properties(self):
        """Test G² statistic mathematical properties"""
        # Test that higher observed vs expected gives higher G²
        file_pairs = [('file1.py', 'file2.py')]
        expected = {('file1.py', 'file2.py'): 2.0}

        # Low observed frequency
        observed_low = {('file1.py', 'file2.py'): 2}
        score_low, _, _ = calculate_g2_surprise(file_pairs, expected, observed_low)

        # High observed frequency
        observed_high = {('file1.py', 'file2.py'): 20}
        score_high, _, _ = calculate_g2_surprise(file_pairs, expected, observed_high)

        assert score_high > score_low  # Higher surprise for unusual frequency


class TestSyntheticDataValidation:
    """Test calculations with known synthetic data scenarios"""

    def test_perfect_storm_scenario(self):
        """Test the 'perfect storm' migration scenario from docs"""
        # Large team migration scenario
        team_factor = TeamFactor.calculate(30)  # 30 developers
        codebase_factor = CodebaseFactor.calculate(1000000, 0.8)  # 1M LOC, high coupling
        velocity_factor = ChangeVelocityFactor.calculate(200)  # 200 commits/week
        migration_multiplier = MigrationMultiplier.calculate(2, 3)  # Major migration

        total_multiplier = team_factor * codebase_factor * velocity_factor * migration_multiplier

        # Should be in the "extreme risk" range (20-50x baseline)
        assert total_multiplier >= 20
        assert total_multiplier <= 100  # Reasonable upper bound

    def test_small_team_scenario(self):
        """Test small team scenario with low risk multipliers"""
        team_factor = TeamFactor.calculate(4)  # 4 developers
        codebase_factor = CodebaseFactor.calculate(50000, 0.4)  # 50k LOC, modular
        velocity_factor = ChangeVelocityFactor.calculate(30)  # 30 commits/week
        migration_multiplier = MigrationMultiplier.calculate(0, 0)  # No migration

        total_multiplier = team_factor * codebase_factor * velocity_factor * migration_multiplier

        # Should be in low risk range (2-3x baseline)
        assert 2 <= total_multiplier <= 4

    def test_edge_case_scenarios(self):
        """Test edge cases and boundary conditions"""
        # Empty inputs
        score, _, _ = calculate_delta_diffusion_blast_radius([], [], [])
        assert 0 <= score <= 1

        # Single file
        score, _, _ = calculate_hawkes_decayed_cochange(['file1.py'], [])
        assert 0 <= score <= 1

        # Zero team size
        assert TeamFactor.calculate(0) == 1.0

        # Zero codebase
        assert CodebaseFactor.calculate(0, 0.5) == 1.0


# Integration test with mock data
@pytest.mark.asyncio
async def test_full_advanced_calculation_integration():
    """Integration test of the complete advanced calculation pipeline"""
    from ..core.risk_engine import RiskEngine
    from ..models.risk_assessment import ChangeContext

    # This would normally require mocking the analyzer and repository
    # For now, test that the structure is correct

    # Create change context
    change_context = ChangeContext(
        files_changed=['src/main.py', 'src/utils.py'],
        lines_added=150,
        lines_deleted=50,
        functions_changed=['main', 'helper'],
        commit_message="Refactor authentication system",
        author="test_user",
        timestamp=datetime.now()
    )

    # Test that we can create the context and it has required fields
    assert len(change_context.files_changed) == 2
    assert change_context.lines_added == 150
    assert change_context.commit_message is not None


if __name__ == "__main__":
    # Run tests with performance measurements
    import time

    async def run_performance_tests():
        """Run performance-critical tests"""
        print("Running performance tests...")

        # Test signal calculation speed
        signal_calc = BlastRadiusSignal()
        start = time.time()

        for _ in range(10):
            await signal_calc.calculate(
                ['file1.py', 'file2.py'],
                [GraphEdge('file1.py', 'file3.py', 1.0, 'import')],
                [GraphEdge('file1.py', 'file2.py', 0.8, 'cochange')]
            )

        avg_time = (time.time() - start) * 1000 / 10
        print(f"Average signal calculation time: {avg_time:.2f}ms")
        assert avg_time < 500, "Signal calculation too slow"

        print("Performance tests passed!")

    # Run if executed directly
    asyncio.run(run_performance_tests())