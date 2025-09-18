"""
Advanced scoring engine with Conformal Risk Control and monotone scoring.

Implements:
- Conformal Risk Control (CRC) for bounded false-escalations
- Monotone scoring with auto-High rules
- Risk tier determination with statistical guarantees
"""

from typing import Dict, List, Tuple, Optional, Set, Any
from dataclasses import dataclass
from enum import Enum
import logging
from collections import defaultdict
import pickle
import json

try:
    import numpy as np
    NUMPY_AVAILABLE = True
except ImportError:
    NUMPY_AVAILABLE = False
    # Fallback for basic array operations
    class np:
        @staticmethod
        def array(data):
            return data

        @staticmethod
        def mean(data, axis=None):
            if axis is None:
                return sum(data) / len(data) if data else 0.0
            # Simplified for axis=0
            return [sum(col) / len(data) for col in zip(*data)]

        @staticmethod
        def unique(data):
            return list(set(data))

        @staticmethod
        def quantile(data, q):
            sorted_data = sorted(data)
            n = len(sorted_data)
            if n == 0:
                return 0.0
            index = int(q * (n - 1))
            return sorted_data[index]

        @staticmethod
        def argmin(data):
            return data.index(min(data))

from ..models.risk_assessment import RiskTier, RiskSignal

logger = logging.getLogger(__name__)


class ConformalMethod(Enum):
    """Conformal prediction methods"""
    SPLIT_CONFORMAL = "split_conformal"
    FULL_CONFORMAL = "full_conformal"
    JACKKNIFE_PLUS = "jackknife_plus"


@dataclass
class ConformalCalibrationData:
    """Data for conformal calibration"""
    features: np.ndarray  # Feature vectors
    labels: np.ndarray    # True risk labels (0=Low, 1=Medium, 2=High, 3=Critical)
    scores: np.ndarray    # Model prediction scores
    segments: np.ndarray  # Segment identifiers (e.g., size buckets)


@dataclass
class ConformalPrediction:
    """Conformal prediction result"""
    prediction_set: Set[RiskTier]
    confidence_level: float
    prediction_intervals: Dict[RiskTier, Tuple[float, float]]
    calibration_score: float


@dataclass
class AutoHighRule:
    """Auto-High escalation rule"""
    name: str
    condition: str
    threshold: float
    signal_names: List[str]
    description: str


class ConformalRiskControl:
    """Implements Conformal Risk Control for bounded false-escalations"""

    def __init__(self, target_coverage: float = 0.95, method: ConformalMethod = ConformalMethod.SPLIT_CONFORMAL):
        """
        Initialize CRC.

        Args:
            target_coverage: Target coverage probability (e.g., 0.95 for 95% coverage)
            method: Conformal prediction method
        """
        self.target_coverage = target_coverage
        self.method = method
        self.calibration_data = None
        self.quantiles = {}
        self.segment_quantiles = {}

    def calibrate(self, calibration_data: ConformalCalibrationData) -> None:
        """
        Calibrate the conformal predictor.

        Args:
            calibration_data: Historical data for calibration
        """
        self.calibration_data = calibration_data

        # Calculate conformity scores (nonconformity measures)
        conformity_scores = self._calculate_conformity_scores(
            calibration_data.scores, calibration_data.labels
        )

        # Calculate quantiles for overall dataset
        alpha = 1 - self.target_coverage
        self.quantiles = self._calculate_quantiles(conformity_scores, alpha)

        # Calculate segment-specific quantiles
        unique_segments = np.unique(calibration_data.segments)
        for segment in unique_segments:
            segment_mask = calibration_data.segments == segment
            segment_scores = conformity_scores[segment_mask]
            self.segment_quantiles[segment] = self._calculate_quantiles(segment_scores, alpha)

        logger.info(f"CRC calibrated with {len(calibration_data.features)} samples")

    def predict(self, feature_vector: np.ndarray, score: float, segment: Optional[int] = None) -> ConformalPrediction:
        """
        Make conformal prediction.

        Args:
            feature_vector: Feature vector for the change
            score: Model prediction score
            segment: Segment identifier

        Returns:
            Conformal prediction with uncertainty quantification
        """
        if self.calibration_data is None:
            raise ValueError("CRC not calibrated. Call calibrate() first.")

        # Choose quantiles (segment-specific if available)
        quantiles = self.segment_quantiles.get(segment, self.quantiles) if segment is not None else self.quantiles

        # Calculate prediction set
        prediction_set = set()
        prediction_intervals = {}

        for tier_idx, tier in enumerate([RiskTier.LOW, RiskTier.MEDIUM, RiskTier.HIGH, RiskTier.CRITICAL]):
            # Calculate conformity score for this tier
            conformity_score = abs(score - tier_idx / 3.0)  # Normalize tier to [0, 1]

            # Check if score is within conformal interval
            if conformity_score <= quantiles.get(tier_idx, float('inf')):
                prediction_set.add(tier)

            # Calculate prediction interval
            lower_bound = max(0.0, score - quantiles.get(tier_idx, 1.0))
            upper_bound = min(1.0, score + quantiles.get(tier_idx, 1.0))
            prediction_intervals[tier] = (lower_bound, upper_bound)

        # If no tier in prediction set, add the most likely one
        if not prediction_set:
            tier_scores = [abs(score - i / 3.0) for i in range(4)]
            best_tier_idx = np.argmin(tier_scores)
            best_tier = [RiskTier.LOW, RiskTier.MEDIUM, RiskTier.HIGH, RiskTier.CRITICAL][best_tier_idx]
            prediction_set.add(best_tier)

        # Calculate calibration score (efficiency measure)
        calibration_score = len(prediction_set) / 4.0  # Smaller is better

        return ConformalPrediction(
            prediction_set=prediction_set,
            confidence_level=self.target_coverage,
            prediction_intervals=prediction_intervals,
            calibration_score=calibration_score
        )

    def _calculate_conformity_scores(self, scores: np.ndarray, labels: np.ndarray) -> np.ndarray:
        """Calculate conformity scores (lower is more conforming)"""
        return np.abs(scores - labels / 3.0)  # Normalize labels to [0, 1] scale

    def _calculate_quantiles(self, conformity_scores: np.ndarray, alpha: float) -> Dict[int, float]:
        """Calculate quantiles for each risk tier"""
        quantiles = {}
        for tier_idx in range(4):  # 4 risk tiers
            tier_scores = conformity_scores  # Use all scores for now
            if len(tier_scores) > 0:
                quantile = np.quantile(tier_scores, 1 - alpha)
                quantiles[tier_idx] = quantile
            else:
                quantiles[tier_idx] = 1.0  # Conservative fallback

        return quantiles


class MonotoneScorer:
    """Implements monotone scoring where all features are non-decreasing w.r.t. risk"""

    def __init__(self, feature_weights: Optional[Dict[str, float]] = None):
        """
        Initialize monotone scorer.

        Args:
            feature_weights: Weights for each feature/signal
        """
        self.feature_weights = feature_weights or self._default_weights()

    def _default_weights(self) -> Dict[str, float]:
        """Default feature weights based on risk math specification"""
        return {
            'delta_diffusion_blast_radius': 0.25,
            'hawkes_decayed_cochange': 0.15,
            'g2_surprise': 0.10,
            'ownership_authority_mismatch': 0.15,
            'span_core_risk': 0.10,
            'bridge_risk': 0.10,
            'incident_adjacency': 0.20,
            'jit_baselines': 0.15,

            # Micro-detectors (lower weights)
            'api_break_risk': 0.05,
            'schema_risk': 0.05,
            'security_risk': 0.05,
            'performance_risk': 0.03,
            'test_gap_risk': 0.07
        }

    def calculate_score(self, signals: List[RiskSignal]) -> Tuple[float, Dict[str, float]]:
        """
        Calculate monotone risk score.

        Args:
            signals: List of risk signals

        Returns:
            Tuple of (total_score, signal_contributions)
        """
        signal_contributions = {}
        total_weighted_score = 0.0
        total_weight = 0.0

        for signal in signals:
            weight = self.feature_weights.get(signal.name, 0.1)

            # Ensure monotonicity: higher signal scores → higher risk
            monotone_score = self._ensure_monotonicity(signal.score, signal.name)

            # Weight by confidence
            contribution = monotone_score * signal.confidence * weight
            signal_contributions[signal.name] = contribution

            total_weighted_score += contribution
            total_weight += weight

        # Normalize to [0, 1] scale
        if total_weight > 0:
            final_score = min(1.0, total_weighted_score / total_weight)
        else:
            final_score = 0.5  # Default moderate risk

        return final_score, signal_contributions

    def _ensure_monotonicity(self, score: float, signal_name: str) -> float:
        """Ensure signal score is monotonic w.r.t. risk"""
        # All signals should already be monotonic (higher value = higher risk)
        # This is a placeholder for any necessary transformations
        return max(0.0, min(1.0, score))


class AutoHighRules:
    """Implements auto-High escalation rules for critical conditions"""

    def __init__(self):
        self.rules = self._initialize_rules()

    def _initialize_rules(self) -> List[AutoHighRule]:
        """Initialize auto-High rules from specification"""
        return [
            AutoHighRule(
                name="api_breaking_critical",
                condition="api_break_risk >= 0.9 AND importer_count >= P95",
                threshold=0.9,
                signal_names=["api_break_risk"],
                description="Critical API breaking change with many importers"
            ),
            AutoHighRule(
                name="schema_critical",
                condition="schema_risk >= 0.9 AND (DROP OR NOT_NULL) AND no_backfill",
                threshold=0.9,
                signal_names=["schema_risk"],
                description="Critical schema change without backfill"
            ),
            AutoHighRule(
                name="security_critical",
                condition="security_risk >= 0.9 AND (secret OR critical_sink)",
                threshold=0.9,
                signal_names=["security_risk"],
                description="Critical security vulnerability"
            ),
            AutoHighRule(
                name="massive_blast_radius",
                condition="delta_diffusion_blast_radius >= 0.95",
                threshold=0.95,
                signal_names=["delta_diffusion_blast_radius"],
                description="Massive blast radius affecting critical infrastructure"
            ),
            AutoHighRule(
                name="size_threshold",
                condition="jit_baselines >= 0.9 AND files_changed >= 20",
                threshold=0.9,
                signal_names=["jit_baselines"],
                description="Auto-High for large number of files changed"
            )
        ]

    def evaluate(self, signals: List[RiskSignal], metadata: Dict[str, Any]) -> Tuple[bool, List[str]]:
        """
        Evaluate auto-High rules.

        Args:
            signals: Risk signals
            metadata: Additional metadata for rule evaluation

        Returns:
            Tuple of (should_escalate, triggered_rules)
        """
        signal_dict = {signal.name: signal for signal in signals}
        triggered_rules = []

        for rule in self.rules:
            if self._evaluate_rule(rule, signal_dict, metadata):
                triggered_rules.append(rule.name)

        should_escalate = len(triggered_rules) > 0

        if should_escalate:
            logger.info(f"Auto-High triggered by rules: {triggered_rules}")

        return should_escalate, triggered_rules

    def _evaluate_rule(self, rule: AutoHighRule, signals: Dict[str, RiskSignal], metadata: Dict[str, Any]) -> bool:
        """Evaluate a specific auto-High rule"""
        try:
            # Check if required signals are present
            for signal_name in rule.signal_names:
                if signal_name not in signals:
                    return False

                signal = signals[signal_name]
                if signal.score < rule.threshold:
                    return False

            # Rule-specific additional conditions
            if rule.name == "api_breaking_critical":
                importer_count = metadata.get("importer_count", 0)
                p95_threshold = metadata.get("importer_p95", 10)
                return importer_count >= p95_threshold

            elif rule.name == "schema_critical":
                has_drop_or_not_null = metadata.get("has_drop_or_not_null", False)
                has_backfill = metadata.get("has_backfill", True)
                return has_drop_or_not_null and not has_backfill

            elif rule.name == "security_critical":
                has_secret = metadata.get("has_secret", False)
                has_critical_sink = metadata.get("has_critical_sink", False)
                return has_secret or has_critical_sink

            elif rule.name == "size_threshold":
                files_changed = metadata.get("files_changed", 0)
                return files_changed >= 20

            # Default: just check threshold
            return True

        except Exception as e:
            logger.error(f"Error evaluating rule {rule.name}: {e}")
            return False


class ScoringEngine:
    """Main scoring engine combining all components"""

    def __init__(self, use_conformal: bool = True, target_coverage: float = 0.95):
        """
        Initialize scoring engine.

        Args:
            use_conformal: Whether to use conformal risk control
            target_coverage: Target coverage for conformal prediction
        """
        self.use_conformal = use_conformal
        self.monotone_scorer = MonotoneScorer()
        self.auto_high_rules = AutoHighRules()

        if use_conformal:
            self.conformal_predictor = ConformalRiskControl(target_coverage)
        else:
            self.conformal_predictor = None

        self.is_calibrated = False

    def calibrate(self, calibration_data: ConformalCalibrationData) -> None:
        """Calibrate the scoring engine with historical data"""
        if self.conformal_predictor:
            self.conformal_predictor.calibrate(calibration_data)
            self.is_calibrated = True
            logger.info("Scoring engine calibrated with conformal prediction")

    def score_change(
        self,
        signals: List[RiskSignal],
        metadata: Dict[str, Any],
        feature_vector: Optional[np.ndarray] = None
    ) -> Tuple[RiskTier, float, Dict[str, Any]]:
        """
        Score a change and determine risk tier.

        Args:
            signals: Risk signals
            metadata: Additional metadata
            feature_vector: Feature vector for conformal prediction

        Returns:
            Tuple of (risk_tier, score, scoring_details)
        """
        # Calculate monotone score
        base_score, signal_contributions = self.monotone_scorer.calculate_score(signals)

        # Check auto-High rules
        auto_high_triggered, triggered_rules = self.auto_high_rules.evaluate(signals, metadata)

        # Apply conformal prediction if available
        conformal_result = None
        if self.conformal_predictor and self.is_calibrated and feature_vector is not None:
            try:
                segment = metadata.get("segment")
                conformal_result = self.conformal_predictor.predict(feature_vector, base_score, segment)
            except Exception as e:
                logger.warning(f"Conformal prediction failed: {e}")

        # Determine final tier
        if auto_high_triggered:
            final_tier = RiskTier.HIGH
            final_score = max(base_score, 0.9)  # Ensure high score for auto-High
        elif conformal_result and len(conformal_result.prediction_set) == 1:
            # Use conformal prediction if confident
            final_tier = list(conformal_result.prediction_set)[0]
            final_score = base_score
        else:
            # Use deterministic thresholds
            final_tier = self._determine_tier_from_score(base_score)
            final_score = base_score

        # Prepare scoring details
        scoring_details = {
            "base_score": base_score,
            "final_score": final_score,
            "signal_contributions": signal_contributions,
            "auto_high_triggered": auto_high_triggered,
            "triggered_rules": triggered_rules,
            "conformal_prediction": conformal_result.__dict__ if conformal_result else None,
            "tier_determination": "auto_high" if auto_high_triggered else (
                "conformal" if conformal_result and len(conformal_result.prediction_set) == 1 else "threshold"
            )
        }

        return final_tier, final_score, scoring_details

    def _determine_tier_from_score(self, score: float) -> RiskTier:
        """Determine risk tier from score using standard thresholds"""
        if score < 0.25:
            return RiskTier.LOW
        elif score < 0.50:
            return RiskTier.MEDIUM
        elif score < 0.75:
            return RiskTier.HIGH
        else:
            return RiskTier.CRITICAL

    def get_feature_importance(self) -> Dict[str, float]:
        """Get feature importance weights"""
        return self.monotone_scorer.feature_weights.copy()

    def update_feature_weights(self, new_weights: Dict[str, float]) -> None:
        """Update feature weights"""
        self.monotone_scorer.feature_weights.update(new_weights)
        logger.info("Feature weights updated")

    def save_model(self, filepath: str) -> None:
        """Save calibrated model to file"""
        model_data = {
            "feature_weights": self.monotone_scorer.feature_weights,
            "conformal_calibration": self.conformal_predictor.calibration_data if self.conformal_predictor else None,
            "conformal_quantiles": self.conformal_predictor.quantiles if self.conformal_predictor else None,
            "is_calibrated": self.is_calibrated
        }

        with open(filepath, 'wb') as f:
            pickle.dump(model_data, f)

        logger.info(f"Model saved to {filepath}")

    def load_model(self, filepath: str) -> None:
        """Load calibrated model from file"""
        try:
            with open(filepath, 'rb') as f:
                model_data = pickle.load(f)

            self.monotone_scorer.feature_weights = model_data["feature_weights"]
            self.is_calibrated = model_data["is_calibrated"]

            if self.conformal_predictor and model_data["conformal_calibration"]:
                self.conformal_predictor.calibration_data = model_data["conformal_calibration"]
                self.conformal_predictor.quantiles = model_data["conformal_quantiles"]

            logger.info(f"Model loaded from {filepath}")

        except Exception as e:
            logger.error(f"Failed to load model: {e}")
            raise