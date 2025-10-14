# Changelog

All notable changes to CodeRisk CLI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of CodeRisk CLI
- Lightning-fast AI-powered code risk assessment
- Phase 0: Pre-filter for security keywords and docs-only changes
- Phase 1: Baseline risk assessment with graph analysis
- Phase 2: Deep LLM-guided agentic investigation
- Support for local mode (self-hosted with Docker)
- Interactive API key setup during installation
- Pre-commit hook integration
- Incident database for learning from past incidents
- Comprehensive testing suite

### Features
- <3% false positive rate (vs 10-20% industry standard)
- 2-5 second risk checks after initial setup
- Multi-platform support: macOS (Intel/ARM), Linux (x64/ARM64), Windows
- Docker support for containerized workflows
- Transparent BYOK pricing: $0.03-0.05 per check
- Graph-based code analysis using Neo4j
- Tree-sitter AST parsing for accurate code understanding
- Git history analysis for temporal coupling detection
- Confidence-driven investigation (stops at 85% confidence)

### Installation Methods
- Homebrew formula (macOS/Linux)
- Universal install script (one-liner)
- Docker images (multi-arch)
- Direct binary downloads from GitHub Releases

## Release Process

This changelog is automatically updated by GoReleaser when creating releases.
Releases are triggered by pushing tags in the format `v*` (e.g., `v1.0.0`).

### Version Format
- **Major** (v2.0.0): Breaking changes
- **Minor** (v1.1.0): New features, backward compatible
- **Patch** (v1.0.1): Bug fixes, backward compatible

---

**Note:** This is a template. Actual releases and their changes will appear above this line.
