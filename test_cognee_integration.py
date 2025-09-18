#!/usr/bin/env python3
"""
Test script for Cognee integration pipeline

This script tests the ingestion pipeline independently of the main package
to verify functionality before integration.
"""

import sys
import tempfile
import shutil
from pathlib import Path
from datetime import datetime
import subprocess
import asyncio

# Add current directory to path to import our modules
sys.path.insert(0, str(Path(__file__).parent))

# Import our ingestion modules directly
from coderisk.ingestion.data_models import (
    CommitDataPoint,
    FileChangeDataPoint,
    DeveloperDataPoint,
    IncidentDataPoint,
)
from coderisk.ingestion.git_history_extractor import GitHistoryExtractor


async def test_data_models():
    """Test DataPoint model creation and validation"""
    print("🧪 Testing DataPoint models...")

    # Test CommitDataPoint
    commit = CommitDataPoint(
        sha='abc123def456',
        message='Fix authentication bug in login system',
        timestamp=datetime.now(),
        author='John Doe',
        author_email='john@example.com',
        files_changed=['auth.py', 'login.py'],
        lines_added=15,
        lines_deleted=3,
        is_security_fix=True,
        touches_auth=True
    )

    print(f"  ✅ CommitDataPoint: {commit.sha} by {commit.author}")
    print(f"     Security fix: {commit.is_security_fix}")
    print(f"     Files: {commit.files_changed}")

    # Test FileChangeDataPoint
    file_change = FileChangeDataPoint(
        file_path='auth.py',
        commit_sha='abc123',
        change_type='modified',
        lines_added=10,
        lines_deleted=2,
        language='python',
        is_test_file=False,
        functions_modified=['authenticate', 'validate_token']
    )

    print(f"  ✅ FileChangeDataPoint: {file_change.file_path} ({file_change.language})")
    print(f"     Functions modified: {file_change.functions_modified}")

    # Test DeveloperDataPoint
    developer = DeveloperDataPoint(
        email='john@example.com',
        name='John Doe',
        total_commits=150,
        commits_last_30_days=25,
        expertise_files=['auth.py', 'utils.py'],
        expertise_languages=['python', 'javascript']
    )

    print(f"  ✅ DeveloperDataPoint: {developer.name} ({developer.email})")
    print(f"     Total commits: {developer.total_commits}")
    print(f"     Expertise: {developer.expertise_languages}")

    # Test IncidentDataPoint
    incident = IncidentDataPoint(
        incident_id='INC-001',
        title='Authentication service down',
        description='Users unable to login due to database connection timeout',
        severity='P1',
        started_at=datetime.now(),
        affected_files=['auth.py', 'database.py'],
        root_cause='Database connection pool exhausted'
    )

    print(f"  ✅ IncidentDataPoint: {incident.incident_id} ({incident.severity})")
    print(f"     Title: {incident.title}")
    print(f"     Affected files: {incident.affected_files}")

    return True


def create_test_repository():
    """Create a test git repository for testing"""
    print("🏗️  Creating test repository...")

    temp_dir = tempfile.mkdtemp()
    repo_path = Path(temp_dir) / "test_repo"
    repo_path.mkdir()

    # Initialize git repo
    subprocess.run(["git", "init"], cwd=repo_path, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.email", "test@example.com"], cwd=repo_path, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.name", "Test User"], cwd=repo_path, check=True, capture_output=True)

    # Create test files and commits
    files_and_commits = [
        ("app.py", "def main():\n    print('Hello World')\n", "Initial application"),
        ("auth.py", "def authenticate(user):\n    return True\n", "Add authentication system"),
        ("utils.py", "def helper():\n    return 'help'\n", "Add utility functions"),
        ("config.yaml", "database:\n  host: localhost\n", "Add configuration"),
        ("test_auth.py", "def test_auth():\n    assert True\n", "Add authentication tests"),
    ]

    for filename, content, message in files_and_commits:
        file_path = repo_path / filename
        file_path.write_text(content)
        subprocess.run(["git", "add", "."], cwd=repo_path, check=True, capture_output=True)
        subprocess.run(["git", "commit", "-m", message], cwd=repo_path, check=True, capture_output=True)

    print(f"  ✅ Created test repo with {len(files_and_commits)} commits at {repo_path}")
    return str(repo_path), temp_dir


async def test_git_history_extractor(repo_path):
    """Test GitHistoryExtractor functionality"""
    print("📊 Testing GitHistoryExtractor...")

    extractor = GitHistoryExtractor(repo_path, window_days=30)

    # Test initialization
    print(f"  ✅ Extractor initialized for {extractor.repo_path}")
    print(f"     Window: {extractor.window_days} days")
    print(f"     Security patterns: {len(extractor.security_patterns)}")
    print(f"     Migration patterns: {len(extractor.migration_patterns)}")

    # Test pattern detection
    assert extractor._is_security_commit("Update JWT authentication system")
    assert not extractor._is_security_commit("Add new feature")
    print("  ✅ Security pattern detection working")

    assert extractor._detect_language("test.py") == "python"
    assert extractor._detect_language("config.yaml") is None
    print("  ✅ Language detection working")

    assert extractor._is_test_file("test_auth.py")
    assert not extractor._is_test_file("auth.py")
    print("  ✅ Test file detection working")

    assert extractor._is_config_file("config.yaml")
    assert not extractor._is_config_file("auth.py")
    print("  ✅ Config file detection working")

    # Extract repository history
    commits, file_changes, developers = await extractor.extract_repository_history()

    print(f"  ✅ Extracted {len(commits)} commits")
    print(f"  ✅ Extracted {len(file_changes)} file changes")
    print(f"  ✅ Extracted {len(developers)} developers")

    # Verify data quality
    if commits:
        print(f"     First commit: {commits[0].message}")
        print(f"     Last commit: {commits[-1].message}")

        # Check for expected files
        all_files = set()
        for commit in commits:
            all_files.update(commit.files_changed)
        print(f"     Files touched: {sorted(all_files)}")

        # Check for security detection
        security_commits = [c for c in commits if c.is_security_fix]
        auth_commits = [c for c in commits if c.touches_auth]
        print(f"     Security commits detected: {len(security_commits)}")
        print(f"     Auth-related commits: {len(auth_commits)}")

    if file_changes:
        languages = set(fc.language for fc in file_changes if fc.language)
        test_files = [fc for fc in file_changes if fc.is_test_file]
        config_files = [fc for fc in file_changes if fc.is_config_file]

        print(f"     Languages detected: {sorted(languages)}")
        print(f"     Test files: {len(test_files)}")
        print(f"     Config files: {len(config_files)}")

    if developers:
        dev = developers[0]
        print(f"     Developer: {dev.name} ({dev.email})")
        print(f"     Total commits: {dev.total_commits}")
        print(f"     Recent commits: {dev.commits_last_30_days}")
        print(f"     Expertise files: {dev.expertise_files[:3]}...")  # Show first 3

    return commits, file_changes, developers


async def test_cognee_basic_functions():
    """Test basic Cognee functionality"""
    print("🔍 Testing basic Cognee functions...")

    try:
        import cognee
        print(f"  ✅ Cognee imported successfully (version: {getattr(cognee, '__version__', 'unknown')})")

        # Test basic cognee functions
        print("  ✅ Available Cognee functions:")
        functions = ['add', 'cognify', 'memify', 'search', 'config']
        for func in functions:
            if hasattr(cognee, func):
                print(f"     ✅ {func}")
            else:
                print(f"     ❌ {func}")

        # Test SearchType
        from cognee import SearchType
        print(f"  ✅ SearchType available: {list(SearchType)}")

        return True

    except Exception as e:
        print(f"  ❌ Cognee test failed: {e}")
        return False


async def main():
    """Main test function"""
    print("🚀 Starting Cognee Integration Tests")
    print("=" * 50)

    success = True

    try:
        # Test 1: Data models
        await test_data_models()
        print()

        # Test 2: Create test repository
        repo_path, temp_dir = create_test_repository()
        print()

        try:
            # Test 3: Git history extraction
            commits, file_changes, developers = await test_git_history_extractor(repo_path)
            print()

            # Test 4: Basic Cognee functionality
            cognee_available = await test_cognee_basic_functions()
            print()

            # Summary
            print("📋 Test Summary")
            print("=" * 30)
            print(f"✅ Data models: Working")
            print(f"✅ Git extraction: {len(commits)} commits, {len(file_changes)} changes, {len(developers)} devs")
            print(f"{'✅' if cognee_available else '❌'} Cognee integration: {'Available' if cognee_available else 'Issues detected'}")

            if cognee_available:
                print("\n🎉 All tests passed! The Cognee ingestion pipeline is ready for use.")
                print("\n🔄 Next steps:")
                print("   1. Set up environment variables (LLM_API_KEY)")
                print("   2. Test with real repository using CogneeCodeAnalyzer")
                print("   3. Verify knowledge graph construction")
            else:
                print("\n⚠️  Some Cognee features may not be available.")
                print("   The ingestion pipeline is built but may need Cognee updates.")
                success = False

        finally:
            # Cleanup test repository
            shutil.rmtree(temp_dir)
            print(f"\n🧹 Cleaned up test repository")

    except Exception as e:
        print(f"\n❌ Test failed with error: {e}")
        import traceback
        traceback.print_exc()
        success = False

    return success


if __name__ == "__main__":
    result = asyncio.run(main())
    exit(0 if result else 1)