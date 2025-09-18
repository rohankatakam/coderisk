"""
Langfuse integration for CodeRisk observability

Provides detailed tracing and monitoring for risk assessment operations
using Langfuse's observability platform.
"""

import os
import time
from typing import Dict, Any, Optional, List
from functools import wraps
import structlog

try:
    from langfuse.decorators import observe
    from langfuse import Langfuse
    LANGFUSE_AVAILABLE = True
except ImportError:
    LANGFUSE_AVAILABLE = False
    # Fallback decorator that does nothing
    def observe(*args, **kwargs):
        def decorator(func):
            return func
        return decorator

logger = structlog.get_logger(__name__)


class LangfuseObserver:
    """Langfuse integration for CodeRisk observability"""

    def __init__(self):
        self.enabled = LANGFUSE_AVAILABLE and self._check_config()
        if self.enabled:
            self.client = Langfuse()
            logger.info("Langfuse observability enabled")
        else:
            self.client = None
            if not LANGFUSE_AVAILABLE:
                logger.warning("Langfuse not available - install with: pip install langfuse")
            else:
                logger.warning("Langfuse configuration missing - set LANGFUSE_PUBLIC_KEY and LANGFUSE_SECRET_KEY")

    def _check_config(self) -> bool:
        """Check if Langfuse is properly configured"""
        return bool(
            os.getenv("LANGFUSE_PUBLIC_KEY") and
            os.getenv("LANGFUSE_SECRET_KEY")
        )

    def observe_risk_assessment(self, func):
        """Decorator for risk assessment functions"""
        if not self.enabled:
            return func

        @observe(as_type="generation", name=f"risk_assessment_{func.__name__}")
        @wraps(func)
        async def wrapper(*args, **kwargs):
            start_time = time.time()
            try:
                result = await func(*args, **kwargs)
                execution_time = (time.time() - start_time) * 1000

                # Add metadata
                if self.client:
                    self.client.trace(
                        name=f"risk_assessment_{func.__name__}",
                        metadata={
                            "execution_time_ms": execution_time,
                            "function": func.__name__,
                            "status": "success"
                        }
                    )

                return result
            except Exception as e:
                execution_time = (time.time() - start_time) * 1000
                if self.client:
                    self.client.trace(
                        name=f"risk_assessment_{func.__name__}",
                        metadata={
                            "execution_time_ms": execution_time,
                            "function": func.__name__,
                            "status": "error",
                            "error": str(e)
                        }
                    )
                raise
        return wrapper

    def observe_signal_calculation(self, signal_name: str):
        """Decorator for individual risk signal calculations"""
        def decorator(func):
            if not self.enabled:
                return func

            @observe(as_type="generation", name=f"signal_{signal_name}")
            @wraps(func)
            async def wrapper(*args, **kwargs):
                start_time = time.time()
                try:
                    result = await func(*args, **kwargs)
                    execution_time = (time.time() - start_time) * 1000

                    # Extract signal metadata if available
                    metadata = {
                        "signal_name": signal_name,
                        "execution_time_ms": execution_time,
                        "status": "success"
                    }

                    if hasattr(result, 'score'):
                        metadata["signal_score"] = result.score
                    if hasattr(result, 'confidence'):
                        metadata["signal_confidence"] = result.confidence
                    if hasattr(result, 'response_time_ms'):
                        metadata["reported_time_ms"] = result.response_time_ms

                    if self.client:
                        self.client.trace(name=f"signal_{signal_name}", metadata=metadata)

                    return result
                except Exception as e:
                    execution_time = (time.time() - start_time) * 1000
                    if self.client:
                        self.client.trace(
                            name=f"signal_{signal_name}",
                            metadata={
                                "signal_name": signal_name,
                                "execution_time_ms": execution_time,
                                "status": "error",
                                "error": str(e)
                            }
                        )
                    raise
            return wrapper
        return decorator

    def observe_cognee_operation(self, operation_name: str):
        """Decorator for Cognee operations"""
        def decorator(func):
            if not self.enabled:
                return func

            @observe(as_type="generation", name=f"cognee_{operation_name}")
            @wraps(func)
            async def wrapper(*args, **kwargs):
                start_time = time.time()
                try:
                    result = await func(*args, **kwargs)
                    execution_time = (time.time() - start_time) * 1000

                    metadata = {
                        "operation": operation_name,
                        "execution_time_ms": execution_time,
                        "status": "success"
                    }

                    # Add result metadata if available
                    if isinstance(result, (list, tuple)):
                        metadata["result_count"] = len(result)
                    elif isinstance(result, dict):
                        metadata["result_keys"] = list(result.keys())

                    if self.client:
                        self.client.trace(name=f"cognee_{operation_name}", metadata=metadata)

                    return result
                except Exception as e:
                    execution_time = (time.time() - start_time) * 1000
                    if self.client:
                        self.client.trace(
                            name=f"cognee_{operation_name}",
                            metadata={
                                "operation": operation_name,
                                "execution_time_ms": execution_time,
                                "status": "error",
                                "error": str(e)
                            }
                        )
                    raise
            return wrapper
        return decorator

    def log_performance_metrics(self, metrics: Dict[str, Any]):
        """Log custom performance metrics"""
        if not self.enabled or not self.client:
            return

        self.client.trace(
            name="performance_metrics",
            metadata={
                "type": "performance",
                **metrics
            }
        )

    def log_repository_analysis(self, repo_path: str, metrics: Dict[str, Any]):
        """Log repository analysis metrics"""
        if not self.enabled or not self.client:
            return

        self.client.trace(
            name="repository_analysis",
            metadata={
                "type": "repository_analysis",
                "repository_path": repo_path,
                **metrics
            }
        )


# Global observer instance
observer = LangfuseObserver()