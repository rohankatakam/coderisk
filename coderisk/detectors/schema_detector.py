"""
Schema Risk Detector

Analyzes database migration operations for risky schema changes.
Focuses on DROP, ADD NOT NULL, type changes, and missing backfill strategies.

Time budget: 10-40ms
"""

import re
from typing import Dict, List, Set, Optional, Any
from pathlib import Path

from . import BaseDetector, DetectorResult, ChangeContext, FileChange, detector_registry


@detector_registry.register
class SchemaRiskDetector(BaseDetector):
    """Detects risky database schema changes in migration files"""

    def __init__(self, repo_path: str):
        super().__init__(repo_path)
        self.migration_patterns = [
            r'migrations?/',
            r'migrate/',
            r'alembic/',
            r'db/migrate',
            r'database/migrations',
            r'\.sql$',
            r'schema\.rb$'
        ]

        self.risky_operations = {
            'DROP TABLE': 0.9,
            'DROP COLUMN': 0.8,
            'ALTER COLUMN.*NOT NULL': 0.7,
            'ALTER COLUMN.*TYPE': 0.6,
            'ADD CONSTRAINT.*UNIQUE': 0.5,
            'ADD CONSTRAINT.*NOT NULL': 0.7,
            'CREATE UNIQUE INDEX': 0.4,
            'RENAME TABLE': 0.6,
            'RENAME COLUMN': 0.5
        }

    async def analyze(self, context: ChangeContext) -> DetectorResult:
        """Analyze schema changes for risk"""
        schema_risks = []
        total_risk_score = 0.0
        reasons = []
        anchors = []
        evidence = {
            "risky_operations": [],
            "missing_backfills": [],
            "unsafe_migrations": [],
            "migration_files": []
        }

        for file_change in context.files_changed:
            if not self._is_migration_file(file_change.path):
                continue

            evidence["migration_files"].append(file_change.path)

            # Analyze migration file content
            file_risk = await self._analyze_migration_file(file_change)

            if file_risk['score'] > 0:
                schema_risks.append(file_risk)
                total_risk_score += file_risk['score']
                reasons.extend(file_risk['reasons'])
                anchors.extend(file_risk['anchors'])
                evidence["risky_operations"].extend(file_risk['risky_ops'])

        # Normalize and adjust score
        if schema_risks:
            risk_score = min(total_risk_score / len(schema_risks), 1.0)

            # Check for critical operations
            for op in evidence["risky_operations"]:
                if op.get('severity') == 'critical':
                    risk_score = max(risk_score, 0.9)
        else:
            risk_score = 0.0

        return DetectorResult(
            score=risk_score,
            reasons=reasons[:5],
            anchors=anchors[:10],
            evidence=evidence,
            execution_time_ms=0.0
        )

    async def _analyze_migration_file(self, file_change: FileChange) -> Dict[str, Any]:
        """Analyze a specific migration file"""
        file_risk = {
            'score': 0.0,
            'reasons': [],
            'anchors': [],
            'risky_ops': []
        }

        try:
            file_path = Path(self.repo_path) / file_change.path
            if not file_path.exists():
                return file_risk

            content = file_path.read_text(encoding='utf-8')

            # Analyze migration content
            await self._check_risky_operations(content, file_change, file_risk)
            await self._check_missing_backfills(content, file_change, file_risk)
            await self._check_transaction_safety(content, file_change, file_risk)

        except Exception as e:
            file_risk['reasons'].append(f"Failed to analyze migration {file_change.path}: {str(e)}")

        return file_risk

    async def _check_risky_operations(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Check for risky SQL operations"""
        lines = content.split('\n')

        for line_num, line in enumerate(lines, 1):
            line_upper = line.upper().strip()

            for operation, risk_weight in self.risky_operations.items():
                if re.search(operation, line_upper):
                    severity = 'critical' if risk_weight >= 0.8 else 'high' if risk_weight >= 0.6 else 'medium'

                    file_risk['risky_ops'].append({
                        'operation': operation,
                        'line': line.strip(),
                        'line_number': line_num,
                        'severity': severity,
                        'risk_weight': risk_weight
                    })

                    file_risk['score'] += risk_weight
                    file_risk['reasons'].append(f"Risky operation: {operation}")
                    file_risk['anchors'].append(f"{file_change.path}:{line_num}")

    async def _check_missing_backfills(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Check for missing backfill strategies"""
        content_upper = content.upper()

        # Check for NOT NULL additions without defaults or backfills
        if re.search(r'ADD COLUMN.*NOT NULL', content_upper):
            if not re.search(r'DEFAULT|BACKFILL|UPDATE.*SET', content_upper):
                file_risk['score'] += 0.6
                file_risk['reasons'].append("NOT NULL column added without backfill strategy")
                file_risk['anchors'].append(f"{file_change.path}:not_null_no_backfill")

        # Check for type changes without data migration
        if re.search(r'ALTER COLUMN.*TYPE', content_upper):
            if not re.search(r'USING|CAST|CONVERT', content_upper):
                file_risk['score'] += 0.5
                file_risk['reasons'].append("Column type changed without data conversion")
                file_risk['anchors'].append(f"{file_change.path}:type_change_no_conversion")

    async def _check_transaction_safety(self, content: str, file_change: FileChange, file_risk: Dict[str, Any]):
        """Check for transaction safety issues"""
        content_upper = content.upper()

        # Check for missing transactions around risky operations
        has_transaction = re.search(r'BEGIN|START TRANSACTION', content_upper)
        has_risky_op = any(re.search(op, content_upper) for op in self.risky_operations.keys())

        if has_risky_op and not has_transaction:
            file_risk['score'] += 0.3
            file_risk['reasons'].append("Risky operations not wrapped in transaction")
            file_risk['anchors'].append(f"{file_change.path}:no_transaction")

        # Check for concurrent index creation without CONCURRENTLY
        if re.search(r'CREATE.*INDEX', content_upper) and not re.search(r'CONCURRENTLY', content_upper):
            file_risk['score'] += 0.4
            file_risk['reasons'].append("Index creation may lock table")
            file_risk['anchors'].append(f"{file_change.path}:blocking_index")

    def _is_migration_file(self, file_path: str) -> bool:
        """Check if file is a database migration file"""
        path_lower = file_path.lower()
        return any(re.search(pattern, path_lower) for pattern in self.migration_patterns)