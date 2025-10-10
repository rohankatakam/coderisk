# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added - October 8, 2025 (P0-P2 Completion Milestone)

#### Phase 2 LLM Investigation (P0)
- **Phase 2 Integration**: Full LLM investigation now integrated into `crisk check` command
- **Escalation Logic**: Automatic escalation when coupling >10, co-change >0.7, or incidents >0
- **Investigator Engine**: Hop-by-hop graph navigation with evidence collection
- **LLM Synthesis**: OpenAI integration for final risk assessment
- **Display Functions**: 
  - `--explain` mode shows full hop-by-hop trace
  - `--ai-mode` includes investigation_trace[] in JSON
  - Standard mode shows summary with recommendations
- **Graceful Degradation**: Helpful message when OPENAI_API_KEY not set

#### Edge Creation Fixes (P1)
- **CAUSED_BY Edges Fixed**: 
  - Added `Incident` unique key mapping to Neo4j backend
  - Fixed File node matching to use `path` field instead of `unique_id`
  - Enhanced `parseNodeID()` to handle file paths without prefix
- **CO_CHANGED Verification**: Confirmed existing timeout/verification implementation
- **AI Mode Complete**: Verified all AI Mode files functional:
  - `ai_actions.go` - AI prompt generation with 4 fix types
  - `ai_converter.go` - JSON conversion logic
  - `ai_mode.go` - Main formatter
  - `graph_analysis.go` - Blast radius, hotspots, temporal coupling
  - `types.go` - Complete type definitions
- **NULL Handling**: Confirmed `sql.NullString` implementation in incident search

#### Integration Tests & CLI (P2)
- **Performance Benchmarks**: Added `test_performance_benchmarks.sh`
  - Co-change query validation (<20ms target)
  - Incident BM25 search (<50ms target)
  - Structural queries (<50ms target)
- **Version Flag**: Added `--version` flag with build info
  - Shows version, build time, git commit
  - Makefile integration for automatic version injection
- **Makefile Targets**: 
  - `make test-layer2` - CO_CHANGED edge validation
  - `make test-layer3` - CAUSED_BY edge validation
  - `make test-performance` - Performance benchmarks
  - `make test-integration` - Run all integration tests

### Fixed
- **Neo4j Backend**: Fixed Incident node edge creation (missing unique key mapping)
- **File Matching**: Fixed File node matching in edge creation queries
- **Node ID Parsing**: Enhanced to handle file paths without label prefix

### Changed
- **Status Documentation**: Updated to reflect 100% core feature completion
- **README**: Updated feature list to show all layers complete
- **Build System**: Version information now injected at build time

---

## [0.1.0] - 2025-10-05

### Added - Graph Construction Fix
- **File Node Fix**: Corrected unique_id generation (was causing 1-node collision)
- **Edge Creation**: Implemented CONTAINS and IMPORTS relationship creation
- **Verification**: Added graph construction verification (5,524 nodes + 5,103 edges)

---

## [0.0.1] - 2025-10-04

### Added - Week 1 Foundation
- **Git Integration**: Implemented all git utility functions
- **Init Flow**: Complete `init-local` orchestration
- **Tree-sitter Parsing**: Multi-language AST parsing
- **Neo4j Integration**: Graph database with Layer 1 entities
- **PostgreSQL**: Metadata and incident storage
- **CLI Framework**: Cobra-based command structure
- **Pre-commit Hook**: Automatic risk checks on commit
- **Verbosity Levels**: 4 output modes (quiet, standard, explain, AI)

### Infrastructure
- Docker Compose setup (Neo4j, PostgreSQL, Redis)
- Environment configuration with .env support
- Makefile build system

---

## Version History

- **October 8, 2025**: P0-P2 Complete (100% core features)
- **October 5, 2025**: Graph construction bug fix
- **October 4, 2025**: Week 1 foundation complete
- **September 2025**: Initial project setup

---

**For detailed implementation status, see [dev_docs/03-implementation/status.md](dev_docs/03-implementation/status.md)**
