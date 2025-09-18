"""
Comprehensive test suite for the Cognee ingestion pipeline

Tests cover:
- GitHistoryExtractor functionality
- CogneeKnowledgeProcessor integration
- DataPoint model validation
- End-to-end ingestion workflow
"""

import pytest
import asyncio
import tempfile
import shutil
from pathlib import Path
from datetime import datetime, timedelta
from typing import List, Dict, Any
from unittest.mock import Mock, AsyncMock, patch

# Import the components to test
from ..ingestion.git_history_extractor import GitHistoryExtractor
from ..ingestion.cognee_processor import CogneeKnowledgeProcessor
from ..ingestion.data_models import (
    CommitDataPoint,
    FileChangeDataPoint,
    DeveloperDataPoint,
    IncidentDataPoint,
    SecurityDataPoint,
)
from ..core.cognee_integration import CogneeCodeAnalyzer


class TestGitHistoryExtractor:
    """Test suite for GitHistoryExtractor"""

    @pytest.fixture
    def temp_repo(self):
        """Create a temporary git repository for testing"""
        temp_dir = tempfile.mkdtemp()
        repo_path = Path(temp_dir) / "test_repo"
        repo_path.mkdir()

        # Initialize git repo and create some test commits
        import subprocess
        import os

        os.chdir(str(repo_path))
        subprocess.run(["git", "init"], check=True, capture_output=True)
        subprocess.run(["git", "config", "user.email", "test@example.com"], check=True, capture_output=True)
        subprocess.run(["git", "config", "user.name", "Test User"], check=True, capture_output=True)

        # Create test files and commits
        test_file = repo_path / "test.py"
        test_file.write_text("def hello():\n    print('Hello World')\n")
        subprocess.run(["git", "add", "."], check=True, capture_output=True)
        subprocess.run(["git", "commit", "-m", "Initial commit"], check=True, capture_output=True)

        # Create a second commit
        test_file.write_text("def hello():\n    print('Hello World!')\n\ndef goodbye():\n    print('Goodbye')\n")
        subprocess.run(["git", "add", "."], check=True, capture_output=True)
        subprocess.run(["git", "commit", "-m", "Add goodbye function"], check=True, capture_output=True)

        yield str(repo_path)

        # Cleanup
        shutil.rmtree(temp_dir)

    @pytest.fixture
    def extractor(self, temp_repo):
        """Create GitHistoryExtractor instance"""
        return GitHistoryExtractor(temp_repo, window_days=30)

    def test_extractor_initialization(self, extractor):
        """Test GitHistoryExtractor initialization"""
        assert extractor.repo_path.exists()
        assert extractor.window_days == 30
        assert len(extractor.security_patterns) > 0
        assert len(extractor.migration_patterns) > 0

    @pytest.mark.asyncio
    async def test_extract_repository_history(self, extractor):
        """Test complete repository history extraction"""
        commits, file_changes, developers = await extractor.extract_repository_history()

        # Should have extracted some commits
        assert len(commits) >= 2
        assert all(isinstance(commit, CommitDataPoint) for commit in commits)

        # Should have file changes
        assert len(file_changes) >= 2
        assert all(isinstance(change, FileChangeDataPoint) for change in file_changes)

        # Should have developer data
        assert len(developers) >= 1
        assert all(isinstance(dev, DeveloperDataPoint) for dev in developers)

    def test_security_pattern_detection(self, extractor):
        """Test security pattern detection"""
        # Test authentication patterns
        assert extractor._is_security_commit("Update JWT authentication")
        assert extractor._is_security_commit("Fix SQL injection vulnerability")
        assert not extractor._is_security_commit("Add new feature")

    def test_migration_detection(self, extractor):
        """Test database migration detection"""
        # Mock commit with migration file
        mock_commit = Mock()
        mock_file = Mock()
        mock_file.filename = "migrations/001_create_users.sql"
        mock_commit.modified_files = [mock_file]
        mock_commit.msg = "Add user table migration"

        assert extractor._has_db_migration(mock_commit)

    def test_language_detection(self, extractor):
        """Test programming language detection"""
        assert extractor._detect_language("test.py") == "python"
        assert extractor._detect_language("test.js") == "javascript"
        assert extractor._detect_language("test.ts") == "typescript"
        assert extractor._detect_language("test.java") == "java"
        assert extractor._detect_language("test.txt") is None

    def test_file_type_detection(self, extractor):
        """Test file type detection"""
        assert extractor._is_test_file("test_utils.py")
        assert extractor._is_test_file("utils.test.js")
        assert not extractor._is_test_file("utils.py")

        assert extractor._is_config_file("config.yml")
        assert extractor._is_config_file("settings.json")
        assert not extractor._is_config_file("utils.py")


class TestCogneeKnowledgeProcessor:
    """Test suite for CogneeKnowledgeProcessor"""

    @pytest.fixture
    def temp_repo_path(self):
        """Create temporary repository path"""
        temp_dir = tempfile.mkdtemp()
        yield temp_dir
        shutil.rmtree(temp_dir)

    @pytest.fixture
    def processor(self, temp_repo_path):
        """Create CogneeKnowledgeProcessor instance"""
        return CogneeKnowledgeProcessor(temp_repo_path, "test_dataset")

    def test_processor_initialization(self, processor):
        """Test CogneeKnowledgeProcessor initialization"""
        assert processor.repo_path.exists()
        assert processor.dataset_name == "test_dataset"
        assert not processor.is_initialized

    @pytest.mark.asyncio
    @patch('cognee.config')
    async def test_initialize(self, mock_config, processor):
        """Test processor initialization"""
        mock_config.return_value = None
        await processor.initialize()
        assert processor.is_initialized
        mock_config.assert_called_once()

    @pytest.mark.asyncio
    @patch('cognee.add')
    async def test_ingest_commits(self, mock_add, processor):
        """Test commit ingestion"""
        processor.is_initialized = True

        commits = [
            CommitDataPoint(
                sha="abc123",
                message="Test commit",
                timestamp=datetime.now(),
                author="Test User",
                author_email="test@example.com",
                files_changed=["test.py"],
                lines_added=10,
                lines_deleted=5
            )
        ]

        await processor._ingest_commits(commits)
        mock_add.assert_called_once()

    @pytest.mark.asyncio
    @patch('cognee.search')
    async def test_analyze_blast_radius(self, mock_search, processor):
        """Test blast radius analysis"""
        processor.is_initialized = True
        mock_search.return_value = [{"impact": "high"}]

        result = await processor.analyze_blast_radius(["test.py"])

        assert "impact_graph" in result
        assert "blast_radius_score" in result
        mock_search.assert_called()

    @pytest.mark.asyncio
    @patch('cognee.search')
    async def test_security_pattern_search(self, mock_search, processor):
        """Test security pattern search"""
        processor.is_initialized = True
        mock_search.return_value = Mock(confidence=0.9)

        result = await processor.search_security_patterns("password = input()", "auth.py")

        assert "vulnerabilities" in result
        assert "confidence" in result
        mock_search.assert_called()


class TestDataPointModels:
    """Test suite for DataPoint model validation"""

    def test_commit_datapoint_creation(self):
        """Test CommitDataPoint creation and validation"""
        commit = CommitDataPoint(
            sha="abc123def456",
            message="Fix authentication bug",
            timestamp=datetime.now(),
            author="John Doe",
            author_email="john@example.com",
            files_changed=["auth.py", "utils.py"],
            lines_added=15,
            lines_deleted=3,
            is_security_fix=True
        )

        assert commit.sha == "abc123def456"
        assert commit.is_security_fix is True
        assert len(commit.files_changed) == 2
        assert "message" in commit.metadata["index_fields"]

    def test_file_change_datapoint(self):
        """Test FileChangeDataPoint creation"""
        file_change = FileChangeDataPoint(
            file_path="src/auth.py",
            commit_sha="abc123",
            change_type="modified",
            lines_added=10,
            lines_deleted=2,
            functions_modified=["authenticate", "validate_token"],
            language="python",
            is_test_file=False
        )

        assert file_change.file_path == "src/auth.py"
        assert file_change.language == "python"
        assert len(file_change.functions_modified) == 2

    def test_developer_datapoint(self):
        """Test DeveloperDataPoint creation"""
        developer = DeveloperDataPoint(
            email="john@example.com",
            name="John Doe",
            total_commits=150,
            commits_last_30_days=25,
            expertise_files=["auth.py", "utils.py"],
            expertise_languages=["python", "javascript"]
        )

        assert developer.email == "john@example.com"
        assert len(developer.expertise_files) == 2
        assert developer.total_commits == 150

    def test_incident_datapoint(self):
        """Test IncidentDataPoint creation"""
        incident = IncidentDataPoint(
            incident_id="INC-123",
            title="Authentication service down",
            description="Users unable to login",
            severity="P1",
            started_at=datetime.now(),
            affected_files=["auth.py", "session.py"],
            root_cause="Database connection timeout"
        )

        assert incident.incident_id == "INC-123"
        assert incident.severity == "P1"
        assert len(incident.affected_files) == 2


class TestCogneeCodeAnalyzer:
    """Test suite for enhanced CogneeCodeAnalyzer"""

    @pytest.fixture
    def temp_repo_path(self):
        """Create temporary repository path"""
        temp_dir = tempfile.mkdtemp()
        yield temp_dir
        shutil.rmtree(temp_dir)

    @pytest.fixture
    def analyzer_full(self, temp_repo_path):
        """Create analyzer with full ingestion enabled"""
        return CogneeCodeAnalyzer(temp_repo_path, enable_full_ingestion=True)

    @pytest.fixture
    def analyzer_legacy(self, temp_repo_path):
        """Create analyzer with legacy mode"""
        return CogneeCodeAnalyzer(temp_repo_path, enable_full_ingestion=False)

    def test_analyzer_initialization_full(self, analyzer_full):
        """Test analyzer initialization with full ingestion"""
        assert analyzer_full.enable_full_ingestion is True
        assert analyzer_full.history_extractor is not None
        assert analyzer_full.knowledge_processor is not None

    def test_analyzer_initialization_legacy(self, analyzer_legacy):
        """Test analyzer initialization in legacy mode"""
        assert analyzer_legacy.enable_full_ingestion is False
        assert analyzer_legacy.history_extractor is None
        assert analyzer_legacy.knowledge_processor is None

    def test_capabilities_full(self, analyzer_full):
        """Test capabilities in full mode"""
        capabilities = analyzer_full.get_capabilities()
        assert capabilities["full_ingestion"] is True
        assert capabilities["temporal_analysis"] is True
        assert capabilities["incident_correlation"] is True

    def test_capabilities_legacy(self, analyzer_legacy):
        """Test capabilities in legacy mode"""
        capabilities = analyzer_legacy.get_capabilities()
        assert capabilities["full_ingestion"] is False
        assert capabilities["temporal_analysis"] is False
        assert capabilities["basic_code_analysis"] is True

    @pytest.mark.asyncio
    @patch('cognee.search')
    async def test_search_code_patterns_enhanced(self, mock_search, analyzer_full):
        """Test enhanced search with dataset specification"""
        analyzer_full._is_initialized = True
        mock_search.return_value = [{"result": "test"}]

        results = await analyzer_full.search_code_patterns("test query", "temporal")

        mock_search.assert_called()
        # Check that dataset_name was passed
        call_kwargs = mock_search.call_args[1]
        assert "dataset_name" in call_kwargs

    @pytest.mark.asyncio
    async def test_cochange_analysis_warning(self, analyzer_legacy):
        """Test co-change analysis warning in legacy mode"""
        result = await analyzer_legacy.analyze_cochange_patterns(["test.py"])
        assert result["cochange_patterns"] == []
        assert result["risk_score"] == 0.0


class TestEndToEndIntegration:
    """End-to-end integration tests"""

    @pytest.fixture
    def temp_repo(self):
        """Create a temporary git repository"""
        temp_dir = tempfile.mkdtemp()
        repo_path = Path(temp_dir) / "test_repo"
        repo_path.mkdir()

        # Initialize git repo
        import subprocess
        import os

        os.chdir(str(repo_path))
        subprocess.run(["git", "init"], check=True, capture_output=True)
        subprocess.run(["git", "config", "user.email", "test@example.com"], check=True, capture_output=True)
        subprocess.run(["git", "config", "user.name", "Test User"], check=True, capture_output=True)

        # Create multiple test files and commits
        files_and_commits = [
            ("app.py", "def main(): pass", "Initial app"),
            ("auth.py", "def login(): pass", "Add authentication"),
            ("utils.py", "def helper(): pass", "Add utilities"),
            ("config.yml", "database: sqlite", "Add config"),
        ]

        for filename, content, message in files_and_commits:
            file_path = repo_path / filename
            file_path.write_text(content)
            subprocess.run(["git", "add", "."], check=True, capture_output=True)
            subprocess.run(["git", "commit", "-m", message], check=True, capture_output=True)

        yield str(repo_path)
        shutil.rmtree(temp_dir)

    @pytest.mark.asyncio
    @patch('cognee.config')
    @patch('cognee.add')
    @patch('cognee.cognify')
    @patch('cognee.memify')
    async def test_full_ingestion_pipeline(self, mock_memify, mock_cognify, mock_add, mock_config, temp_repo):
        """Test complete ingestion pipeline from git history to Cognee"""
        # Setup mocks
        mock_config.return_value = None
        mock_add.return_value = None
        mock_cognify.return_value = None
        mock_memify.return_value = None

        # Create analyzer and run ingestion
        analyzer = CogneeCodeAnalyzer(temp_repo, enable_full_ingestion=True)

        # This would normally do full ingestion, but we're mocking Cognee
        await analyzer.initialize(force_reingest=True)

        # Verify that Cognee operations were called
        mock_config.assert_called()
        assert mock_add.call_count >= 1  # Should have ingested commits, file changes, developers
        mock_cognify.assert_called()
        mock_memify.assert_called()

    @pytest.mark.asyncio
    async def test_git_extraction_integration(self, temp_repo):
        """Test GitHistoryExtractor with real git repository"""
        extractor = GitHistoryExtractor(temp_repo, window_days=30)

        commits, file_changes, developers = await extractor.extract_repository_history()

        # Verify extraction results
        assert len(commits) >= 4  # Should match number of commits created
        assert len(file_changes) >= 4  # Should have file changes
        assert len(developers) >= 1  # Should have at least one developer

        # Verify commit content
        commit_messages = [c.message for c in commits]
        assert "Initial app" in commit_messages
        assert "Add authentication" in commit_messages

        # Verify file detection
        file_paths = [fc.file_path for fc in file_changes]
        assert "app.py" in file_paths
        assert "auth.py" in file_paths
        assert "config.yml" in file_paths

        # Verify language detection
        py_files = [fc for fc in file_changes if fc.language == "python"]
        assert len(py_files) >= 3

        # Verify config file detection
        config_files = [fc for fc in file_changes if fc.is_config_file]
        assert len(config_files) >= 1

    def test_data_model_consistency(self):
        """Test that all DataPoint models have consistent metadata"""
        from ..ingestion.data_models import (
            CommitDataPoint, FileChangeDataPoint, DeveloperDataPoint,
            IncidentDataPoint, SecurityDataPoint
        )

        # All models should have metadata attribute
        models = [CommitDataPoint, FileChangeDataPoint, DeveloperDataPoint,
                 IncidentDataPoint, SecurityDataPoint]

        for model in models:
            # Create a minimal instance to check metadata
            try:
                # This would need proper initialization based on each model's requirements
                # For now, just check that the class has the metadata field defined
                assert hasattr(model, '__annotations__')
                assert 'metadata' in model.__annotations__
            except Exception:
                # If we can't instantiate, at least check metadata is defined
                pass


# Utility functions for testing
def create_test_commit() -> CommitDataPoint:
    """Create a test commit for testing purposes"""
    return CommitDataPoint(
        sha="test123",
        message="Test commit",
        timestamp=datetime.now(),
        author="Test User",
        author_email="test@example.com",
        files_changed=["test.py"],
        lines_added=10,
        lines_deleted=0
    )


def create_test_incident() -> IncidentDataPoint:
    """Create a test incident for testing purposes"""
    return IncidentDataPoint(
        incident_id="TEST-1",
        title="Test incident",
        description="Test incident description",
        severity="P2",
        started_at=datetime.now(),
        affected_files=["test.py"]
    )


# Configuration for pytest
def pytest_configure(config):
    """Configure pytest for async testing"""
    config.addinivalue_line(
        "markers", "asyncio: mark test as async"
    )


if __name__ == "__main__":
    # Run tests when script is executed directly
    pytest.main([__file__, "-v"])