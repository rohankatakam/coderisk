"""
DeepEval integration for CodeRisk testing validation

Provides automated evaluation and testing capabilities for risk assessment
accuracy using DeepEval's LLM-based evaluation framework.
"""

import os
import json
from typing import Dict, Any, List, Optional, Tuple
from dataclasses import dataclass
import structlog

try:
    from deepeval import evaluate
    from deepeval.metrics import GEval, FaithfulnessMetric, ContextualRelevancyMetric
    from deepeval.test_case import LLMTestCase
    from deepeval.dataset import EvaluationDataset
    DEEPEVAL_AVAILABLE = True
except ImportError:
    DEEPEVAL_AVAILABLE = False

logger = structlog.get_logger(__name__)


@dataclass
class RiskAssessmentTestCase:
    """Test case for risk assessment evaluation"""
    repository_path: str
    expected_risk_tier: str
    expected_signals: List[str]
    actual_assessment: Dict[str, Any]
    context: Dict[str, Any]


class DeepEvalTester:
    """DeepEval integration for automated testing validation"""

    def __init__(self):
        self.enabled = DEEPEVAL_AVAILABLE and self._check_config()
        if self.enabled:
            logger.info("DeepEval testing enabled")
        else:
            if not DEEPEVAL_AVAILABLE:
                logger.warning("DeepEval not available - install with: pip install deepeval")
            else:
                logger.warning("DeepEval configuration missing - set OPENAI_API_KEY or similar LLM provider")

    def _check_config(self) -> bool:
        """Check if DeepEval is properly configured"""
        return bool(
            os.getenv("OPENAI_API_KEY") or
            os.getenv("ANTHROPIC_API_KEY") or
            os.getenv("GOOGLE_API_KEY")
        )

    def create_risk_assessment_metric(self) -> Optional[Any]:
        """Create a custom metric for risk assessment accuracy"""
        if not self.enabled:
            return None

        return GEval(
            name="RiskAssessmentAccuracy",
            criteria="Evaluate the accuracy and appropriateness of the risk assessment based on the code changes and repository context.",
            evaluation_params=[
                "Risk tier appropriateness (LOW/MEDIUM/HIGH)",
                "Relevance of detected risk signals",
                "Accuracy of confidence scores",
                "Completeness of risk evidence"
            ],
            threshold=0.7
        )

    def create_contextual_relevancy_metric(self) -> Optional[Any]:
        """Create a metric for contextual relevancy of risk signals"""
        if not self.enabled:
            return None

        return ContextualRelevancyMetric(
            threshold=0.8,
            model="gpt-3.5-turbo",
            include_reason=True
        )

    def evaluate_risk_assessment(
        self,
        test_cases: List[RiskAssessmentTestCase]
    ) -> Optional[Dict[str, Any]]:
        """Evaluate multiple risk assessment test cases"""
        if not self.enabled:
            logger.warning("DeepEval not enabled, skipping evaluation")
            return None

        try:
            llm_test_cases = []
            risk_metric = self.create_risk_assessment_metric()
            relevancy_metric = self.create_contextual_relevancy_metric()

            for test_case in test_cases:
                # Convert to DeepEval test case format
                llm_test_case = LLMTestCase(
                    input=f"Repository: {test_case.repository_path}\\nContext: {json.dumps(test_case.context)}",
                    actual_output=json.dumps(test_case.actual_assessment),
                    expected_output=f"Risk Tier: {test_case.expected_risk_tier}\\nExpected Signals: {test_case.expected_signals}",
                    context=[
                        f"Repository path: {test_case.repository_path}",
                        f"File changes: {test_case.context.get('files_changed', [])}",
                        f"Lines changed: {test_case.context.get('lines_changed', 0)}"
                    ]
                )
                llm_test_cases.append(llm_test_case)

            # Create dataset and evaluate
            dataset = EvaluationDataset(test_cases=llm_test_cases)
            metrics = [risk_metric, relevancy_metric] if risk_metric and relevancy_metric else []

            if metrics:
                results = evaluate(dataset, metrics)
                return self._process_evaluation_results(results)
            else:
                logger.warning("No metrics available for evaluation")
                return None

        except Exception as e:
            logger.error(f"DeepEval evaluation failed: {e}")
            return None

    def _process_evaluation_results(self, results: Any) -> Dict[str, Any]:
        """Process and summarize evaluation results"""
        summary = {
            "total_test_cases": 0,
            "passed_test_cases": 0,
            "failed_test_cases": 0,
            "average_scores": {},
            "detailed_results": []
        }

        try:
            if hasattr(results, 'test_results'):
                summary["total_test_cases"] = len(results.test_results)
                passed = sum(1 for result in results.test_results if result.success)
                summary["passed_test_cases"] = passed
                summary["failed_test_cases"] = summary["total_test_cases"] - passed

                # Calculate average scores by metric
                metric_scores = {}
                for result in results.test_results:
                    if hasattr(result, 'metrics_metadata'):
                        for metric_data in result.metrics_metadata:
                            metric_name = metric_data.get('metric', 'unknown')
                            score = metric_data.get('score', 0)
                            if metric_name not in metric_scores:
                                metric_scores[metric_name] = []
                            metric_scores[metric_name].append(score)

                for metric_name, scores in metric_scores.items():
                    summary["average_scores"][metric_name] = sum(scores) / len(scores) if scores else 0

        except Exception as e:
            logger.error(f"Failed to process evaluation results: {e}")

        return summary

    def create_test_case_from_assessment(
        self,
        repo_path: str,
        assessment_result: Dict[str, Any],
        expected_tier: str = None,
        expected_signals: List[str] = None
    ) -> RiskAssessmentTestCase:
        """Create a test case from an actual assessment result"""
        return RiskAssessmentTestCase(
            repository_path=repo_path,
            expected_risk_tier=expected_tier or "UNKNOWN",
            expected_signals=expected_signals or [],
            actual_assessment=assessment_result,
            context={
                "files_changed": assessment_result.get("change_context", {}).get("files_changed", []),
                "lines_changed": (
                    assessment_result.get("change_context", {}).get("lines_added", 0) +
                    assessment_result.get("change_context", {}).get("lines_deleted", 0)
                ),
                "assessment_time_ms": assessment_result.get("assessment_time_ms", 0)
            }
        )

    def benchmark_repository_types(self) -> List[RiskAssessmentTestCase]:
        """Create benchmark test cases for different repository types"""
        benchmark_cases = [
            {
                "repo_type": "small_python",
                "expected_tier": "LOW",
                "expected_signals": ["test_gap", "merge_risk"],
                "description": "Small Python project with good test coverage"
            },
            {
                "repo_type": "medium_javascript",
                "expected_tier": "MEDIUM",
                "expected_signals": ["dep_risk", "security_risk", "performance_risk"],
                "description": "Medium-sized JavaScript project with dependencies"
            },
            {
                "repo_type": "large_enterprise",
                "expected_tier": "HIGH",
                "expected_signals": ["api_break_risk", "schema_risk", "config_risk"],
                "description": "Large enterprise application with multiple services"
            }
        ]

        # This would be implemented to create actual test cases
        # For now, return empty list as placeholder
        return []

    def validate_signal_accuracy(
        self,
        signal_name: str,
        detected_evidence: List[str],
        actual_code_changes: List[str]
    ) -> Optional[float]:
        """Validate the accuracy of a specific risk signal"""
        if not self.enabled:
            return None

        try:
            # Create a focused test case for this signal
            test_case = LLMTestCase(
                input=f"Signal: {signal_name}\\nCode Changes: {actual_code_changes}",
                actual_output=f"Evidence: {detected_evidence}",
                expected_output=f"Accurate detection of {signal_name} risks",
                context=actual_code_changes
            )

            relevancy_metric = self.create_contextual_relevancy_metric()
            if relevancy_metric:
                result = evaluate([test_case], [relevancy_metric])
                return self._extract_score_from_result(result)

        except Exception as e:
            logger.error(f"Signal validation failed for {signal_name}: {e}")

        return None

    def _extract_score_from_result(self, result: Any) -> float:
        """Extract numerical score from evaluation result"""
        try:
            if hasattr(result, 'test_results') and result.test_results:
                first_result = result.test_results[0]
                if hasattr(first_result, 'metrics_metadata') and first_result.metrics_metadata:
                    return first_result.metrics_metadata[0].get('score', 0.0)
        except Exception:
            pass
        return 0.0


# Global tester instance
tester = DeepEvalTester()