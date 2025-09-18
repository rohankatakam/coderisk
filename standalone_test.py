#!/usr/bin/env python3
"""
Standalone test for Cognee ingestion pipeline
"""

import sys
import tempfile
import shutil
from pathlib import Path
from datetime import datetime
import subprocess
import asyncio

# Import modules directly to avoid package import issues
sys.path.insert(0, str(Path(__file__).parent))

# Import Pydantic for models
from pydantic import BaseModel
from typing import List, Optional, Dict, Any

# Import PyDriller for git extraction
from pydriller import Repository


# Define DataPoint models inline to avoid import issues
class DataPoint(BaseModel):
    """Base DataPoint class for Cognee compatibility"""
    metadata: Dict[str, Any] = {}


class CommitDataPoint(DataPoint):
    """Structured commit data for Cognee ingestion"""
    sha: str
    message: str
    timestamp: datetime
    author: str
    author_email: str
    files_changed: List[str]
    lines_added: int
    lines_deleted: int
    is_merge: bool = False
    is_revert: bool = False
    is_hotfix: bool = False
    is_security_fix: bool = False
    branch: Optional[str] = None
    parents: List[str] = []

    # Risk indicators
    has_db_migration: bool = False
    touches_auth: bool = False
    touches_config: bool = False
    large_change: bool = False

    metadata: Dict[str, Any] = {
        "index_fields": ["message", "author", "files_changed"],
        "vector_fields": ["message"],
        "temporal_field": "timestamp"
    }


def create_test_repository():
    """Create a test git repository"""
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
        ("auth.py", "def authenticate(user, password):\n    return check_credentials(user, password)\n", "Add authentication system"),
        ("utils.py", "def helper():\n    return 'help'\n", "Add utility functions"),
        ("config.yaml", "database:\n  host: localhost\n  port: 5432\n", "Add database configuration"),
        ("test_auth.py", "def test_auth():\n    assert authenticate('user', 'pass')\n", "Add authentication tests"),
        ("migrations/001_users.sql", "CREATE TABLE users (id INT, name VARCHAR);", "Add user table migration"),
    ]

    for filename, content, message in files_and_commits:
        file_path = repo_path / filename
        file_path.parent.mkdir(parents=True, exist_ok=True)
        file_path.write_text(content)
        subprocess.run(["git", "add", "."], cwd=repo_path, check=True, capture_output=True)
        subprocess.run(["git", "commit", "-m", message], cwd=repo_path, check=True, capture_output=True)

    print(f"  ✅ Created test repo with {len(files_and_commits)} commits at {repo_path}")
    return str(repo_path), temp_dir


def extract_commits_with_pydriller(repo_path):
    """Extract commits using PyDriller directly"""
    print("📊 Extracting commits with PyDriller...")

    commits = []

    for commit in Repository(repo_path).traverse_commits():
        # Detect risk indicators
        is_security = any(word in commit.msg.lower() for word in ['auth', 'security', 'password', 'login'])
        has_migration = any('migration' in file.filename.lower() if file.filename else False
                           for file in commit.modified_files)
        touches_auth = any('auth' in file.filename.lower() if file.filename else False
                          for file in commit.modified_files)
        touches_config = any(file.filename.endswith(('.yaml', '.yml', '.json', '.ini')) if file.filename else False
                            for file in commit.modified_files)

        total_lines = sum(mod.added_lines + mod.deleted_lines for mod in commit.modified_files)
        large_change = total_lines > 50  # Lower threshold for test

        commit_data = CommitDataPoint(
            sha=commit.hash,
            message=commit.msg,
            timestamp=commit.committer_date,
            author=commit.author.name,
            author_email=commit.author.email,
            files_changed=[mod.filename for mod in commit.modified_files if mod.filename],
            lines_added=sum(mod.added_lines for mod in commit.modified_files),
            lines_deleted=sum(mod.deleted_lines for mod in commit.modified_files),
            is_merge=commit.merge,
            is_security_fix=is_security,
            has_db_migration=has_migration,
            touches_auth=touches_auth,
            touches_config=touches_config,
            large_change=large_change,
            parents=[parent for parent in commit.parents] if commit.parents else [],
        )

        commits.append(commit_data)

    print(f"  ✅ Extracted {len(commits)} commits")

    # Analyze the commits
    security_commits = [c for c in commits if c.is_security_fix]
    auth_commits = [c for c in commits if c.touches_auth]
    migration_commits = [c for c in commits if c.has_db_migration]
    config_commits = [c for c in commits if c.touches_config]
    large_commits = [c for c in commits if c.large_change]

    print(f"     Security-related: {len(security_commits)}")
    print(f"     Auth-related: {len(auth_commits)}")
    print(f"     With migrations: {len(migration_commits)}")
    print(f"     Config changes: {len(config_commits)}")
    print(f"     Large changes: {len(large_commits)}")

    # Show commit details
    for i, commit in enumerate(commits):
        print(f"     [{i+1}] {commit.sha[:8]} - {commit.message}")
        if commit.is_security_fix:
            print(f"         🔒 Security-related")
        if commit.touches_auth:
            print(f"         🔑 Touches authentication")
        if commit.has_db_migration:
            print(f"         🗃️  Database migration")
        if commit.touches_config:
            print(f"         ⚙️  Configuration change")
        print(f"         📝 Files: {commit.files_changed}")

    return commits


def test_cognee_import():
    """Test Cognee import and basic functionality"""
    print("🔍 Testing Cognee import...")

    try:
        import cognee
        print(f"  ✅ Cognee imported successfully")

        # Check available functions
        functions = ['add', 'cognify', 'memify', 'search', 'config', 'SearchType']
        available = []
        missing = []

        for func in functions:
            if hasattr(cognee, func):
                available.append(func)
            else:
                missing.append(func)

        print(f"  ✅ Available functions: {available}")
        if missing:
            print(f"  ⚠️  Missing functions: {missing}")

        # Test SearchType import
        try:
            from cognee import SearchType
            search_types = [attr for attr in dir(SearchType) if not attr.startswith('_')]
            print(f"  ✅ SearchType available: {search_types}")
        except ImportError as e:
            print(f"  ❌ SearchType import failed: {e}")

        return True

    except ImportError as e:
        print(f"  ❌ Cognee import failed: {e}")
        return False


async def test_cognee_basic_operations():
    """Test basic Cognee operations with sample data"""
    print("🧪 Testing Cognee basic operations...")

    try:
        import cognee

        # Test configuration
        cognee.config()  # config is not async in this version
        print("  ✅ Cognee configured")

        # Create sample data
        sample_commits = [
            CommitDataPoint(
                sha="abc123",
                message="Add authentication system",
                timestamp=datetime.now(),
                author="John Doe",
                author_email="john@example.com",
                files_changed=["auth.py"],
                lines_added=50,
                lines_deleted=0,
                is_security_fix=True,
                touches_auth=True
            )
        ]

        # Convert to dict for Cognee
        commit_dicts = [commit.model_dump() for commit in sample_commits]

        # Test add operation (simplified for this version)
        await cognee.add(commit_dicts, dataset_name="test_coderisk")
        print("  ✅ Data added to Cognee")

        # Test cognify
        await cognee.cognify(datasets=["test_coderisk"])
        print("  ✅ Data cognified")

        # Test search
        from cognee import SearchType
        results = await cognee.search(
            query_text="authentication",
            query_type=SearchType.CHUNKS,
            dataset_name="test_coderisk"
        )
        print(f"  ✅ Search completed, found {len(results) if results else 0} results")

        return True

    except Exception as e:
        print(f"  ❌ Cognee operations failed: {e}")
        import traceback
        traceback.print_exc()
        return False


async def main():
    """Main test function"""
    print("🚀 Cognee Integration Standalone Test")
    print("=" * 50)

    success = True

    try:
        # Test 1: Test Cognee import
        cognee_available = test_cognee_import()
        print()

        # Test 2: Create test repository and extract data
        repo_path, temp_dir = create_test_repository()
        print()

        try:
            # Test 3: Extract commits with PyDriller
            commits = extract_commits_with_pydriller(repo_path)
            print()

            # Test 4: Test Cognee operations if available
            if cognee_available:
                cognee_ops_success = await test_cognee_basic_operations()
                print()
            else:
                cognee_ops_success = False
                print("  ⚠️  Skipping Cognee operations test (import failed)")
                print()

            # Summary
            print("📋 Test Results Summary")
            print("=" * 30)
            print(f"✅ PyDriller extraction: {len(commits)} commits processed")
            print(f"{'✅' if cognee_available else '❌'} Cognee import: {'Success' if cognee_available else 'Failed'}")
            print(f"{'✅' if cognee_ops_success else '❌'} Cognee operations: {'Success' if cognee_ops_success else 'Failed'}")

            # Detailed analysis
            if commits:
                security_count = sum(1 for c in commits if c.is_security_fix)
                auth_count = sum(1 for c in commits if c.touches_auth)
                migration_count = sum(1 for c in commits if c.has_db_migration)

                print("\n📊 Data Analysis:")
                print(f"   - Total commits: {len(commits)}")
                print(f"   - Security-related: {security_count}")
                print(f"   - Authentication: {auth_count}")
                print(f"   - Database migrations: {migration_count}")

                # Show risk indicators working
                if security_count > 0 or auth_count > 0 or migration_count > 0:
                    print("   ✅ Risk indicator detection working")
                else:
                    print("   ⚠️  Risk indicators may need tuning")

            # Final verdict
            if cognee_available and cognee_ops_success and commits:
                print("\n🎉 SUCCESS: Cognee integration pipeline is fully functional!")
                print("\n🔄 Ready for:")
                print("   1. Full repository ingestion")
                print("   2. Knowledge graph construction")
                print("   3. Risk analysis queries")
                print("   4. Temporal pattern detection")
            elif cognee_available and commits:
                print("\n✅ PARTIAL SUCCESS: Core components working")
                print("   - Git extraction: ✅")
                print("   - Cognee import: ✅")
                print("   - Basic operations: ⚠️  (needs investigation)")
            else:
                print("\n⚠️  ISSUES DETECTED:")
                if not cognee_available:
                    print("   - Cognee not properly installed/configured")
                if not commits:
                    print("   - Git extraction failed")
                success = False

        finally:
            # Cleanup
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