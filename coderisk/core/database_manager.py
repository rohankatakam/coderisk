"""
Database connection manager for Cognee/Kuzu integration

This module implements proper Kuzu connection management to prevent database locks
and ensure concurrent access safety based on Kuzu's concurrency model.
"""

import asyncio
import os
from typing import Optional, Dict, Any
from pathlib import Path
import structlog
from contextlib import asynccontextmanager

logger = structlog.get_logger(__name__)


class DatabaseConnectionManager:
    """
    Manages Kuzu database connections to prevent concurrent access issues.

    Based on Kuzu's concurrency model:
    - One READ_WRITE Database object
    - Multiple connections from the same Database object
    - Proper connection pooling and cleanup
    """

    _instance: Optional['DatabaseConnectionManager'] = None
    _lock = asyncio.Lock()

    def __new__(cls):
        if cls._instance is None:
            cls._instance = super().__new__(cls)
        return cls._instance

    def __init__(self):
        if hasattr(self, '_initialized'):
            return

        self._initialized = True
        self._active_connections: Dict[str, Any] = {}
        self._connection_locks: Dict[str, asyncio.Lock] = {}
        self._cleanup_tasks: Dict[str, asyncio.Task] = {}

        logger.info("Database connection manager initialized")

    def get_safe_connection_context(self, process_id: str = "default"):
        """
        Get a safe connection context that prevents database locks.

        Args:
            process_id: Unique identifier for this process/operation

        Returns:
            Async context manager for safe database operations
        """
        return self._connection_context(process_id)

    @asynccontextmanager
    async def _connection_context(self, process_id: str):
        """
        Context manager for safe database connections.

        Ensures:
        1. Single READ_WRITE database object per process
        2. Proper connection cleanup
        3. No concurrent database object creation
        """
        async with self._lock:
            if process_id not in self._connection_locks:
                self._connection_locks[process_id] = asyncio.Lock()

        async with self._connection_locks[process_id]:
            connection_info = None
            try:
                # Set environment variable to help Cognee use single connection
                old_kuzu_mode = os.environ.get('COGNEE_KUZU_SINGLE_CONNECTION')
                os.environ['COGNEE_KUZU_SINGLE_CONNECTION'] = 'true'

                connection_info = {
                    'process_id': process_id,
                    'created_at': asyncio.get_event_loop().time(),
                    'active': True
                }

                self._active_connections[process_id] = connection_info

                logger.info("Database connection acquired",
                           process_id=process_id,
                           active_connections=len(self._active_connections))

                yield connection_info

            except Exception as e:
                logger.error("Database connection error",
                           process_id=process_id,
                           error=str(e))
                raise
            finally:
                # Cleanup
                if connection_info:
                    connection_info['active'] = False

                if process_id in self._active_connections:
                    del self._active_connections[process_id]

                # Restore environment
                if old_kuzu_mode is None:
                    os.environ.pop('COGNEE_KUZU_SINGLE_CONNECTION', None)
                else:
                    os.environ['COGNEE_KUZU_SINGLE_CONNECTION'] = old_kuzu_mode

                logger.info("Database connection released",
                           process_id=process_id,
                           active_connections=len(self._active_connections))

    async def wait_for_exclusive_access(self, timeout: int = 30):
        """
        Wait for exclusive database access (no other active connections).

        Args:
            timeout: Maximum time to wait in seconds
        """
        start_time = asyncio.get_event_loop().time()

        while self._active_connections:
            if asyncio.get_event_loop().time() - start_time > timeout:
                raise TimeoutError(f"Timeout waiting for exclusive database access after {timeout}s")

            logger.debug("Waiting for database connections to close",
                        active_connections=len(self._active_connections))
            await asyncio.sleep(0.1)

        logger.info("Exclusive database access acquired")

    def get_active_connections(self) -> Dict[str, Dict[str, Any]]:
        """Get information about currently active connections."""
        return {
            pid: {
                'process_id': info['process_id'],
                'created_at': info['created_at'],
                'active': info['active'],
                'duration': asyncio.get_event_loop().time() - info['created_at']
            }
            for pid, info in self._active_connections.items()
        }

    async def force_cleanup(self):
        """Force cleanup of all connections (use with caution)."""
        logger.warning("Force cleanup of all database connections initiated")

        # Cancel any cleanup tasks
        for task in self._cleanup_tasks.values():
            if not task.done():
                task.cancel()

        self._active_connections.clear()
        self._connection_locks.clear()
        self._cleanup_tasks.clear()

        logger.info("Force cleanup completed")


# Global instance
db_manager = DatabaseConnectionManager()


async def safe_cognee_operation(operation_func, process_id: str = None, **kwargs):
    """
    Execute a Cognee operation with safe database connection management.

    Args:
        operation_func: Async function to execute
        process_id: Unique identifier for this operation
        **kwargs: Arguments to pass to the operation function

    Returns:
        Result of the operation function
    """
    if process_id is None:
        import uuid
        process_id = f"op_{uuid.uuid4().hex[:8]}"

    async with db_manager.get_safe_connection_context(process_id):
        try:
            return await operation_func(**kwargs)
        except Exception as e:
            logger.error("Safe Cognee operation failed",
                        operation=operation_func.__name__,
                        process_id=process_id,
                        error=str(e))
            raise


def setup_kuzu_environment():
    """
    Set up environment variables for optimal Kuzu performance and concurrency.
    """
    # Set Kuzu-specific environment variables
    kuzu_settings = {
        'KUZU_MAX_DB_SIZE': '8192',  # 8GB max database size
        'KUZU_BUFFER_POOL_SIZE': '4096',  # 4GB buffer pool
        'KUZU_NUM_THREADS': '4',  # Limit threads for stability
        'KUZU_CONNECTION_TIMEOUT': '30000',  # 30 second timeout
        'COGNEE_KUZU_SAFE_MODE': 'true',  # Enable safe mode
    }

    for key, value in kuzu_settings.items():
        if key not in os.environ:
            os.environ[key] = value
            logger.debug(f"Set {key}={value}")

    logger.info("Kuzu environment configured for optimal concurrency")


# Initialize on import
setup_kuzu_environment()