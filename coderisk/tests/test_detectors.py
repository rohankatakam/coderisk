"""
Test Suite for CodeRisk Micro-Detectors

Comprehensive tests for all 9 micro-detectors using synthetic code examples
and mock file systems.
"""

import unittest
import asyncio
import tempfile
import os
from pathlib import Path
from unittest.mock import Mock, patch, MagicMock

from ..detectors import (
    ChangeContext, FileChange, DetectorResult,
    detector_registry
)
from ..detectors.api_detector import ApiBreakDetector
from ..detectors.schema_detector import SchemaRiskDetector
from ..detectors.dependency_detector import DependencyRiskDetector
from ..detectors.performance_detector import PerformanceRiskDetector
from ..detectors.concurrency_detector import ConcurrencyRiskDetector
from ..detectors.security_detector import SecurityRiskDetector
from ..detectors.config_detector import ConfigRiskDetector
from ..detectors.test_detector import TestGapDetector
from ..detectors.merge_detector import MergeRiskDetector


class TestDetectorBase(unittest.TestCase):
    """Base class for detector tests"""

    def setUp(self):
        """Set up test environment"""
        self.temp_dir = tempfile.mkdtemp()
        self.repo_path = self.temp_dir

    def tearDown(self):
        """Clean up test environment"""
        import shutil
        shutil.rmtree(self.temp_dir, ignore_errors=True)

    def create_test_file(self, path: str, content: str):
        """Create a test file with given content"""
        full_path = Path(self.temp_dir) / path
        full_path.parent.mkdir(parents=True, exist_ok=True)
        full_path.write_text(content)
        return str(full_path)

    def create_change_context(self, files_data: list) -> ChangeContext:
        """Create a ChangeContext with test files"""
        file_changes = []

        for file_data in files_data:
            path = file_data['path']
            content = file_data.get('content', '')
            change_type = file_data.get('change_type', 'modified')
            hunks = file_data.get('hunks', [])

            # Create the file
            if content:
                self.create_test_file(path, content)

            file_change = FileChange(
                path=path,
                change_type=change_type,
                lines_added=file_data.get('lines_added', 10),
                lines_deleted=file_data.get('lines_deleted', 5),
                hunks=hunks
            )
            file_changes.append(file_change)

        return ChangeContext(
            files_changed=file_changes,
            total_lines_added=sum(f.lines_added for f in file_changes),
            total_lines_deleted=sum(f.lines_deleted for f in file_changes)
        )


class TestApiBreakDetector(TestDetectorBase):
    """Test API break risk detector"""

    def setUp(self):
        super().setUp()
        self.detector = ApiBreakDetector(self.repo_path)

    def test_python_function_removal(self):
        """Test detection of removed Python functions"""
        files_data = [{
            'path': 'api.py',
            'content': '''
def public_function():
    pass

def _private_function():
    pass

class PublicClass:
    def public_method(self):
        pass
            ''',
            'hunks': [{
                'lines': [
                    '-def public_function():',
                    '-    pass',
                    '+# Function removed'
                ]
            }]
        }]

        context = self.create_change_context(files_data)
        result = asyncio.run(self.detector.analyze(context))

        self.assertIsInstance(result, DetectorResult)
        self.assertGreater(result.score, 0.0)
        self.assertTrue(any('public_function' in reason for reason in result.reasons))

    def test_javascript_export_changes(self):
        """Test detection of JavaScript export changes"""
        files_data = [{
            'path': 'module.js',
            'content': '''
export function apiFunction() {
    return 'hello';
}

export default class MyClass {
    method() {}
}
            ''',
            'hunks': [{
                'lines': [
                    '-export function apiFunction() {',
                    '+// function removed'
                ]
            }]
        }]

        context = self.create_change_context(files_data)
        result = asyncio.run(self.detector.analyze(context))

        self.assertGreater(result.score, 0.0)
        self.assertTrue(len(result.evidence['breaking_changes']) > 0)

    def test_no_api_changes(self):
        """Test no risk when no API changes detected"""
        files_data = [{
            'path': 'internal.py',
            'content': '''
def _internal_function():
    pass

# Just a comment change
x = 1 + 1
            ''',
            'hunks': [{
                'lines': [
                    '+# Added comment',
                    ' x = 1 + 1'
                ]
            }]
        }]

        context = self.create_change_context(files_data)
        result = asyncio.run(self.detector.analyze(context))

        self.assertEqual(result.score, 0.0)


class TestSchemaRiskDetector(TestDetectorBase):
    """Test schema risk detector"""

    def setUp(self):
        super().setUp()
        self.detector = SchemaRiskDetector(self.repo_path)

    def test_dangerous_migration(self):
        """Test detection of dangerous migration operations"""
        files_data = [{
            'path': 'migrations/001_dangerous.sql',
            'content': '''
DROP TABLE old_table;
ALTER TABLE users ADD COLUMN email VARCHAR(255) NOT NULL;
ALTER COLUMN age TYPE INTEGER;
            ''',
            'hunks': [{
                'lines': [
                    '+DROP TABLE old_table;',
                    '+ALTER TABLE users ADD COLUMN email VARCHAR(255) NOT NULL;'
                ]
            }]
        }]

        context = self.create_change_context(files_data)
        result = asyncio.run(self.detector.analyze(context))

        self.assertGreater(result.score, 0.5)
        self.assertTrue(any('DROP' in reason for reason in result.reasons))
        self.assertTrue(any('NOT NULL' in reason for reason in result.reasons))

    def test_safe_migration(self):
        """Test safe migration operations"""
        files_data = [{
            'path': 'migrations/002_safe.sql',
            'content': '''
CREATE TABLE new_table (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255)
);

ALTER TABLE users ADD COLUMN phone VARCHAR(20);
CREATE INDEX idx_users_email ON users(email);
            ''',
            'hunks': [{
                'lines': [
                    '+CREATE TABLE new_table (',
                    '+    id SERIAL PRIMARY KEY,',
                    '+    name VARCHAR(255)',
                    '+);'
                ]
            }]
        }]

        context = self.create_change_context(files_data)
        result = asyncio.run(self.detector.analyze(context))

        self.assertLess(result.score, 0.3)


class TestDependencyRiskDetector(TestDetectorBase):
    """Test dependency risk detector"""

    def setUp(self):
        super().setUp()
        self.detector = DependencyRiskDetector(self.repo_path)

    def test_major_version_updates(self):
        """Test detection of major version updates"""
        files_data = [{
            'path': 'package.json',
            'content': '''
{
  "dependencies": {
    "react": "^18.0.0",
    "lodash": "^4.17.21"
  }
}
            ''',
            'hunks': [{
                'lines': [
                    '-    "react": "^17.0.0",',
                    '+    "react": "^18.0.0",',
                    '-    "lodash": "^3.10.1",',
                    '+    "lodash": "^4.17.21"'
                ]
            }]
        }]

        context = self.create_change_context(files_data)
        result = asyncio.run(self.detector.analyze(context))

        self.assertGreater(result.score, 0.0)

    def test_new_dependency_addition(self):
        """Test new dependency addition"""
        files_data = [{
            'path': 'requirements.txt',
            'content': '''
requests==2.28.1
numpy==1.21.0
suspicious-package==0.1.0-beta
            ''',
            'hunks': [{
                'lines': [
                    '+suspicious-package==0.1.0-beta'
                ]
            }]
        }]

        context = self.create_change_context(files_data)
        result = asyncio.run(self.detector.analyze(context))

        self.assertGreater(result.score, 0.0)
        self.assertTrue(any('unstable' in reason for reason in result.reasons))


class TestPerformanceRiskDetector(TestDetectorBase):
    """Test performance risk detector"""

    def setUp(self):
        super().setUp()
        self.detector = PerformanceRiskDetector(self.repo_path)

    def test_io_in_loop(self):
        """Test detection of I/O operations in loops"""
        files_data = [{
            'path': 'slow_code.py',
            'content': '''
import requests

def fetch_data(urls):
    results = []
    for url in urls:
        response = requests.get(url)
        results.append(response.json())
    return results

def string_concat_loop(items):
    result = ""
    for item in items:
        result += str(item)
    return result
            ''',
            'hunks': [{
                'lines': [
                    '+    for url in urls:',
                    '+        response = requests.get(url)',
                    '+    for item in items:',
                    '+        result += str(item)'
                ]
            }]
        }]

        context = self.create_change_context(files_data)
        result = asyncio.run(self.detector.analyze(context))

        self.assertGreater(result.score, 0.0)
        self.assertTrue(len(result.evidence['loops_with_io']) > 0)

    def test_efficient_code(self):
        """Test efficient code patterns"""
        files_data = [{
            'path': 'efficient_code.py',
            'content': '''
def efficient_processing(items):
    # Pre-fetch all data
    all_data = batch_fetch(items)

    # Use list comprehension
    results = [process_item(item, all_data) for item in items]

    return results
            ''',
            'hunks': [{
                'lines': [
                    '+def efficient_processing(items):',
                    '+    all_data = batch_fetch(items)',
                    '+    results = [process_item(item, all_data) for item in items]'
                ]
            }]
        }]

        context = self.create_change_context(files_data)
        result = asyncio.run(self.detector.analyze(context))

        self.assertEqual(result.score, 0.0)


class TestSecurityRiskDetector(TestDetectorBase):
    """Test security risk detector"""

    def setUp(self):
        super().setUp()
        self.detector = SecurityRiskDetector(self.repo_path)

    def test_sql_injection_detection(self):
        """Test SQL injection vulnerability detection"""
        files_data = [{
            'path': 'vulnerable.py',
            'content': '''
def get_user(user_id):
    query = "SELECT * FROM users WHERE id = " + user_id
    cursor.execute(query)
    return cursor.fetchone()

def another_vulnerability(name):
    sql = f"DELETE FROM users WHERE name = '{name}'"
    execute(sql)
            ''',
            'hunks': [{
                'lines': [
                    '+    query = "SELECT * FROM users WHERE id = " + user_id',
                    '+    sql = f"DELETE FROM users WHERE name = \'{name}\'"'
                ]
            }]
        }]

        context = self.create_change_context(files_data)
        result = asyncio.run(self.detector.analyze(context))

        self.assertGreater(result.score, 0.0)
        self.assertTrue(len(result.evidence['vulnerabilities']) > 0)

    def test_hardcoded_secrets(self):
        """Test hardcoded secrets detection"""
        files_data = [{
            'path': 'config.py',
            'content': '''
API_KEY = "sk-1234567890abcdef1234567890abcdef"
PASSWORD = "supersecretpassword123"
DATABASE_URL = "postgres://user:password@localhost/db"
            ''',
            'hunks': [{
                'lines': [
                    '+API_KEY = "sk-1234567890abcdef1234567890abcdef"',
                    '+PASSWORD = "supersecretpassword123"'
                ]
            }]
        }]

        context = self.create_change_context(files_data)
        result = asyncio.run(self.detector.analyze(context))

        self.assertGreater(result.score, 0.0)
        self.assertTrue(len(result.evidence['secrets_detected']) > 0)


class TestConfigRiskDetector(TestDetectorBase):
    """Test configuration risk detector"""

    def setUp(self):
        super().setUp()
        self.detector = ConfigRiskDetector(self.repo_path)

    def test_kubernetes_security_issues(self):
        """Test Kubernetes security configuration issues"""
        files_data = [{
            'path': 'k8s/deployment.yaml',
            'content': '''
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      containers:
      - name: app
        image: app:latest
        securityContext:
          privileged: true
          runAsRoot: true
          allowPrivilegeEscalation: true
            ''',
            'hunks': [{
                'lines': [
                    '+        securityContext:',
                    '+          privileged: true',
                    '+          runAsRoot: true'
                ]
            }]
        }]

        context = self.create_change_context(files_data)
        result = asyncio.run(self.detector.analyze(context))

        self.assertGreater(result.score, 0.5)
        self.assertTrue(len(result.evidence['k8s_security_issues']) > 0)

    def test_terraform_public_access(self):
        """Test Terraform public access detection"""
        files_data = [{
            'path': 'infrastructure/main.tf',
            'content': '''
resource "aws_security_group" "web" {
  ingress {
    from_port   = 0
    to_port     = 65535
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
            ''',
            'hunks': [{
                'lines': [
                    '+    cidr_blocks = ["0.0.0.0/0"]'
                ]
            }]
        }]

        context = self.create_change_context(files_data)
        result = asyncio.run(self.detector.analyze(context))

        self.assertGreater(result.score, 0.0)


class TestTestGapDetector(TestDetectorBase):
    """Test test gap detector"""

    def setUp(self):
        super().setUp()
        self.detector = TestGapDetector(self.repo_path)

    def test_code_without_tests(self):
        """Test detection of code changes without corresponding tests"""
        files_data = [
            {
                'path': 'src/important_module.py',
                'content': '''
class CriticalClass:
    def important_method(self):
        # Complex business logic
        pass

    def parse_data(self, data):
        # Parsing logic that needs testing
        return processed_data
                ''',
                'hunks': [{
                    'lines': [
                        '+class CriticalClass:',
                        '+    def important_method(self):',
                        '+        # Complex business logic'
                    ]
                }]
            }
        ]

        context = self.create_change_context(files_data)
        result = asyncio.run(self.detector.analyze(context))

        self.assertGreater(result.score, 0.0)
        self.assertTrue(len(result.evidence['changed_code_files']) > 0)
        self.assertEqual(len(result.evidence['related_test_files']), 0)

    def test_code_with_tests(self):
        """Test code changes with corresponding tests"""
        files_data = [
            {
                'path': 'src/module.py',
                'content': 'def function(): pass',
                'hunks': [{'lines': ['+def function(): pass']}]
            },
            {
                'path': 'tests/test_module.py',
                'content': 'def test_function(): assert True',
                'hunks': [{'lines': ['+def test_function(): assert True']}]
            }
        ]

        context = self.create_change_context(files_data)
        result = asyncio.run(self.detector.analyze(context))

        self.assertLess(result.score, 0.5)  # Lower risk when tests are present


class TestMergeRiskDetector(TestDetectorBase):
    """Test merge risk detector"""

    def setUp(self):
        super().setUp()
        self.detector = MergeRiskDetector(self.repo_path)

    def test_conflict_prone_files(self):
        """Test detection of conflict-prone file changes"""
        files_data = [
            {
                'path': 'package.json',
                'content': '{"dependencies": {"react": "^18.0.0"}}',
                'hunks': [{'lines': ['+  "react": "^18.0.0"']}]
            },
            {
                'path': 'requirements.txt',
                'content': 'django==4.0.0',
                'hunks': [{'lines': ['+django==4.0.0']}]
            }
        ]

        context = self.create_change_context(files_data)
        result = asyncio.run(self.detector.analyze(context))

        self.assertGreater(result.score, 0.0)
        self.assertTrue(len(result.evidence['conflict_prone_files']) > 0)

    def test_multiple_dependency_files(self):
        """Test risk from multiple dependency file changes"""
        files_data = [
            {'path': 'package.json', 'content': '{}'},
            {'path': 'yarn.lock', 'content': ''},
            {'path': 'requirements.txt', 'content': ''}
        ]

        context = self.create_change_context(files_data)
        result = asyncio.run(self.detector.analyze(context))

        self.assertGreater(result.score, 0.0)


class TestDetectorRegistry(TestDetectorBase):
    """Test detector registry functionality"""

    def test_registry_contains_all_detectors(self):
        """Test that registry contains all expected detectors"""
        expected_detectors = [
            'api_break', 'schema_risk', 'dependency_risk', 'performance_risk',
            'concurrency_risk', 'security_risk', 'config_risk', 'test_gap', 'merge_risk'
        ]

        available_detectors = detector_registry.list_detectors()

        for detector_name in expected_detectors:
            self.assertIn(detector_name, available_detectors)

    def test_detector_instantiation(self):
        """Test that all detectors can be instantiated"""
        for detector_name in detector_registry.list_detectors():
            detector = detector_registry.get_detector(detector_name, self.repo_path)
            self.assertIsNotNone(detector)
            self.assertEqual(detector.repo_path.name, Path(self.repo_path).name)

    def test_detector_timeout_handling(self):
        """Test that detectors handle timeouts properly"""
        detector = detector_registry.get_detector('api_break', self.repo_path)
        context = ChangeContext(files_changed=[], total_lines_added=0, total_lines_deleted=0)

        # Test with very short timeout
        result = asyncio.run(detector.run_with_timeout(context, timeout_ms=1))

        # Should complete without error (either success or timeout)
        self.assertIsInstance(result, DetectorResult)
        self.assertGreaterEqual(result.execution_time_ms, 0)


class TestDetectorIntegration(TestDetectorBase):
    """Integration tests for multiple detectors"""

    def test_comprehensive_analysis(self):
        """Test running multiple detectors on a complex change"""
        files_data = [
            {
                'path': 'src/api.py',
                'content': '''
import requests

def get_user(user_id):
    # SQL injection vulnerability
    query = "SELECT * FROM users WHERE id = " + str(user_id)
    return execute_query(query)

def fetch_user_data(user_ids):
    results = []
    for user_id in user_ids:  # Performance issue: I/O in loop
        response = requests.get(f"/api/users/{user_id}")
        results.append(response.json())
    return results
                ''',
                'hunks': [{
                    'lines': [
                        '+def get_user(user_id):',
                        '+    query = "SELECT * FROM users WHERE id = " + str(user_id)',
                        '+    for user_id in user_ids:',
                        '+        response = requests.get(f"/api/users/{user_id}")'
                    ]
                }]
            },
            {
                'path': 'migrations/001_add_users.sql',
                'content': 'DROP TABLE old_users; ALTER TABLE users ADD COLUMN email VARCHAR(255) NOT NULL;',
                'hunks': [{'lines': ['+DROP TABLE old_users;']}]
            },
            {
                'path': 'package.json',
                'content': '{"dependencies": {"react": "^18.0.0"}}',
                'hunks': [{'lines': ['+  "react": "^18.0.0"']}]
            }
        ]

        context = self.create_change_context(files_data)

        # Run multiple detectors
        detectors = [
            ('security_risk', SecurityRiskDetector),
            ('performance_risk', PerformanceRiskDetector),
            ('schema_risk', SchemaRiskDetector),
            ('dependency_risk', DependencyRiskDetector)
        ]

        results = {}
        for detector_name, detector_class in detectors:
            detector = detector_class(self.repo_path)
            result = asyncio.run(detector.analyze(context))
            results[detector_name] = result

        # Verify multiple risks detected
        high_risk_detectors = [name for name, result in results.items() if result.score > 0.5]
        self.assertGreater(len(high_risk_detectors), 1)

        # Security detector should find SQL injection
        self.assertGreater(results['security_risk'].score, 0.0)

        # Performance detector should find I/O in loop
        self.assertGreater(results['performance_risk'].score, 0.0)

        # Schema detector should find risky migration
        self.assertGreater(results['schema_risk'].score, 0.0)


if __name__ == '__main__':
    # Custom test runner that handles async tests
    unittest.main(verbosity=2)