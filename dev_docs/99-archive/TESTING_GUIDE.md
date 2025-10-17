# CodeRisk System Testing Guide

This comprehensive guide provides high-level testing procedures to validate the complete CodeRisk system across various repository types and edge cases.

## 🎯 Testing Overview

CodeRisk integrates multiple components requiring systematic validation:
- **Multi-Database Architecture**: RDBMS (SQLite/PostgreSQL), Vector DB (LanceDB/PGVector), Graph DB (Kuzu/Neo4j)
- **Cognee Integration**: Full repository ingestion with code graph creation
- **GitHub Data Pipeline**: Commits, PRs, Issues extraction and processing
- **Risk Assessment Engine**: Mathematical models and micro-detectors
- **Observability**: Structured logging and performance monitoring

## 🚀 Environment Setup

### Prerequisites
- Python 3.8+ with CodeRisk installed
- OpenAI API key for Cognee integration
- Git access to test repositories
- Sufficient disk space for database files (~1-5GB per large repository)

### Configuration Files
- Configure `.env` file with production-ready settings
- Verify `test_integration.py` exists in repository root
- Ensure CLI commands are accessible (`crisk --help`)

## 📊 Test Repository Matrix

Test CodeRisk across repositories of varying sizes and characteristics to validate scalability and accuracy:

### Small Repository (1-100 commits)
**Repository**: `https://github.com/rohankatakam/coderisk` (Self-test)
- **Purpose**: Baseline functionality validation
- **Expected Data**: ~10 commits, ~50 files, 2-3 developers
- **Test Focus**: Basic pipeline functionality, database initialization

### Medium Repository (100-1000 commits)
**Repository**: `https://github.com/topoteretes/cognee`
- **Purpose**: Cognee framework integration validation
- **Expected Data**: ~500 commits, ~200 files, 5-10 developers
- **Test Focus**: Cognee's built-in code graph capabilities, temporal analysis

### Large Python Repository (1000+ commits)
**Repository**: `https://github.com/psf/requests`
- **Purpose**: Large-scale Python project analysis
- **Expected Data**: ~2000+ commits, ~100 files, 300+ developers
- **Test Focus**: Performance under load, memory management

### Large JavaScript Repository (1000+ commits)
**Repository**: `https://github.com/lodash/lodash`
- **Purpose**: Multi-language support validation
- **Expected Data**: ~1500+ commits, ~1000 files, 200+ developers
- **Test Focus**: Tree-sitter parsing, language-specific detectors

### Enterprise-Scale Repository (5000+ commits)
**Repository**: `https://github.com/microsoft/vscode`
- **Purpose**: Enterprise-scale validation
- **Expected Data**: ~100k+ commits, ~10k files, 1000+ developers
- **Test Focus**: Extreme scale handling, rate limiting, chunked processing

## 🧪 Test Execution Framework

### Phase 1: System Integration Tests
Execute the integrated test suite to validate core functionality:

**Test File**: `test_integration.py`
**Expected Results**: 4/4 tests passing
- Full Integration (Cognee + Mathematical Models)
- Micro-Detectors (9 specialized detectors)
- Calculation Engine (Advanced mathematical models)
- Cognee Integration (Knowledge graph operations)

### Phase 2: Database Validation
Verify multi-database architecture initialization and health:

**RDBMS Validation**:
- Database file creation in Cognee system directory
- Table schema validation for metadata storage
- Connection pooling and transaction handling

**Vector Database Validation**:
- LanceDB initialization with proper schemas
- Embedding storage and retrieval operations
- Vector similarity search functionality

**Graph Database Validation**:
- Kuzu database initialization
- Node and edge creation for code dependencies
- Graph traversal and path finding operations

### Phase 3: Repository Ingestion Tests

For each test repository, validate complete data pipeline:

**Git History Extraction**:
- All commits captured with proper metadata
- File change deltas accurately recorded
- Developer attribution and timestamps preserved

**GitHub API Integration** (if available):
- Pull requests with merge information
- Issue tracking and link correlation
- Branch and tag metadata extraction

**Cognee Knowledge Graph Construction**:
- Code structure parsing via tree-sitter
- Dependency relationship mapping
- Function/class relationship graphs
- Import/export dependency chains

**Data Volume Validation**:
- Verify extracted commit counts match repository
- Confirm file change counts align with git log
- Validate developer count accuracy

### Phase 4: Performance Benchmarking

Execute performance tests across repository size categories:

**Timing Targets**:
- Small repos: <10 seconds end-to-end
- Medium repos: <30 seconds end-to-end
- Large repos: <2 minutes end-to-end
- Risk assessment: <2 seconds (any repo size)
- Individual detectors: 50-150ms each

**Memory Targets**:
- Small repos: <200MB peak usage
- Medium repos: <500MB peak usage
- Large repos: <2GB peak usage
- No memory leaks during repeated assessments

**Database Growth**:
- Monitor database file sizes post-ingestion
- Verify reasonable storage efficiency
- Check index creation and query performance

### Phase 5: Edge Case Validation

Test system resilience against problematic scenarios:

**Empty/Minimal Repositories**:
- Newly initialized git repositories
- Single-commit repositories
- Repositories with no code files

**Large File Handling**:
- Repositories with binary assets
- Files exceeding size limits
- Unusual file types and encodings

**Git History Anomalies**:
- Merge conflicts and resolution patterns
- Force pushes and history rewrites
- Large refactoring commits

**Network and API Failures**:
- Rate limiting scenarios
- Temporary API unavailability
- Partial data retrieval

## 🔍 Observability Validation

### Structured Logging Assessment
Monitor log output during test execution for:

**Initialization Logs**:
- Database connection establishment
- Cognee system startup
- Configuration validation

**Processing Logs**:
- Git extraction progress and metrics
- Cognee ingestion status and timing
- Risk calculation performance data

**Error Handling Logs**:
- Graceful failure modes
- Retry logic execution
- Fallback mechanism activation

### Database Health Monitoring
Track database operations and file system changes:

**File System Validation**:
- Database files created in expected locations
- File size growth patterns during ingestion
- Cleanup and maintenance operations

**Query Performance**:
- Database query execution times
- Index usage and optimization
- Connection pool utilization

## 📈 Success Criteria Matrix

### Functional Requirements
- [ ] All repository types successfully ingest without critical errors
- [ ] Cognee code graph generation completes for all test cases
- [ ] Risk assessments produce reasonable scores across repository types
- [ ] CLI commands function correctly with all test repositories

### Performance Requirements
- [ ] Processing times meet targets for each repository size category
- [ ] Memory usage stays within defined bounds
- [ ] Database storage efficiency meets expectations
- [ ] No memory leaks detected during extended testing

### Data Quality Requirements
- [ ] Commit extraction accuracy >99% (verified via git log comparison)
- [ ] Developer attribution correctly mapped
- [ ] File dependency graphs logically consistent
- [ ] Risk signal calculations produce non-zero meaningful results

### Observability Requirements
- [ ] Structured logs provide clear pipeline visibility
- [ ] Error messages are actionable and descriptive
- [ ] Performance metrics available for optimization
- [ ] Database health monitoring functions correctly

## 🚨 Failure Scenarios and Recovery

### Common Failure Patterns
**API Rate Limiting**: Implement exponential backoff and request throttling
**Large Repository Timeouts**: Enable chunked processing and progress checkpoints
**Database Lock Contention**: Verify connection pooling and transaction isolation
**Memory Exhaustion**: Implement streaming processing for large datasets

### Recovery Procedures
**Partial Ingestion Failures**: Resume from last successful checkpoint
**Database Corruption**: Automatic backup and restore mechanisms
**Configuration Errors**: Validation and helpful error messaging
**Network Interruptions**: Retry logic with persistent state

## 🎯 Test Execution Checklist

### Pre-Test Setup
- [ ] Environment variables configured correctly
- [ ] All dependencies installed and versions verified
- [ ] Database directories have sufficient disk space
- [ ] Network connectivity to GitHub APIs confirmed

### During Test Execution
- [ ] Monitor system resource utilization
- [ ] Capture and review structured log outputs
- [ ] Document any unexpected behaviors or errors
- [ ] Record actual vs. expected performance metrics

### Post-Test Validation
- [ ] Compare extracted data counts with repository ground truth
- [ ] Verify database file integrity and reasonable sizes
- [ ] Validate risk assessment outputs for logical consistency
- [ ] Review log files for any warning or error patterns

### Regression Testing
- [ ] Re-run tests after any significant code changes
- [ ] Validate backwards compatibility with existing datasets
- [ ] Ensure performance characteristics remain stable
- [ ] Verify new features don't break existing functionality

## 📞 Troubleshooting Reference

### Configuration Issues
Review `.env` file settings and ensure API keys are valid and have sufficient quotas.

### Performance Problems
Monitor system resources and adjust processing parameters in configuration.

### Data Quality Issues
Validate git repository accessibility and verify GitHub API permissions if using extended features.

### Database Problems
Check file permissions and disk space in Cognee system directories.

This testing framework ensures comprehensive validation of CodeRisk's multi-database architecture, Cognee integration, and risk assessment capabilities across diverse repository types and scales.