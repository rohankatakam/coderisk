# Contributing to CodeRisk

Thank you for your interest in contributing to CodeRisk! We're building the trust infrastructure for AI-generated code, and we welcome contributions from developers of all experience levels.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
- [Development Setup](#development-setup)
- [Contribution Guidelines](#contribution-guidelines)
- [Areas Accepting Contributions](#areas-accepting-contributions)
- [Areas Not Accepting Contributions](#areas-not-accepting-contributions)
- [Pull Request Process](#pull-request-process)
- [Coding Standards](#coding-standards)
- [Community](#community)

## Code of Conduct

This project adheres to a code of conduct that we expect all contributors to follow:

- **Be respectful:** Treat everyone with respect and consideration
- **Be constructive:** Provide helpful feedback and suggestions
- **Be collaborative:** Work together towards common goals
- **Be inclusive:** Welcome diverse perspectives and backgrounds

## How Can I Contribute?

### Reporting Bugs

Before creating a bug report:
- Check the [existing issues](https://github.com/rohankatakam/coderisk-go/issues) to avoid duplicates
- Ensure you're using the latest version of CodeRisk

When creating a bug report, include:
- **Clear title and description**
- **Steps to reproduce** the issue
- **Expected vs actual behavior**
- **Environment details** (OS, Go version, Docker version)
- **Relevant logs or error messages**
- **Code samples** if applicable

### Suggesting Features

We love feature suggestions! Before submitting:
- Check [existing feature requests](https://github.com/rohankatakam/coderisk-go/discussions)
- Consider if it fits CodeRisk's [vision and mission](dev_docs/00-product/vision_and_mission.md)

When suggesting a feature:
- **Describe the problem** you're trying to solve
- **Propose a solution** with specific examples
- **Explain the benefits** to CodeRisk users
- **Consider alternatives** you've evaluated

### Improving Documentation

Documentation contributions are highly valued! You can help by:
- Fixing typos or clarifying explanations
- Adding examples or use cases
- Improving architecture documentation
- Writing tutorials or guides

See [dev_docs/DOCUMENTATION_WORKFLOW.md](dev_docs/DOCUMENTATION_WORKFLOW.md) for documentation contribution guidelines.

## Development Setup

### Prerequisites

- **Go 1.21+** ([Installation guide](https://golang.org/doc/install))
- **Docker & Docker Compose** ([Installation guide](https://docs.docker.com/get-docker/))
- **Git** ([Installation guide](https://git-scm.com/downloads))

### Getting Started

1. **Fork the repository** on GitHub

2. **Clone your fork:**
   ```bash
   git clone https://github.com/YOUR_USERNAME/coderisk-go.git
   cd coderisk-go
   ```

3. **Add upstream remote:**
   ```bash
   git remote add upstream https://github.com/rohankatakam/coderisk-go.git
   ```

4. **Install dependencies:**
   ```bash
   go mod download
   ```

5. **Start local infrastructure:**
   ```bash
   docker compose up -d
   ```

6. **Build the CLI:**
   ```bash
   go build -o crisk ./cmd/crisk
   ```

7. **Run tests:**
   ```bash
   go test ./...
   ```

### Development Workflow

1. **Create a feature branch:**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** and commit:
   ```bash
   git add .
   git commit -m "feat: Add your feature description"
   ```

3. **Keep your branch updated:**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

4. **Push to your fork:**
   ```bash
   git push origin feature/your-feature-name
   ```

5. **Create a Pull Request** on GitHub

## Contribution Guidelines

### Commit Message Convention

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Adding or updating tests
- `refactor:` Code refactoring
- `perf:` Performance improvements
- `chore:` Build process or auxiliary tool changes

**Examples:**
```
feat: Add temporal coupling metric for co-change detection
fix: Handle nil pointer in graph traversal
docs: Update architecture decision for Neptune migration
test: Add integration tests for init-local command
```

### Code Review Process

All contributions require code review before merging:

1. **Automated checks** must pass (tests, linting)
2. **At least one maintainer approval** required
3. **Address feedback** constructively
4. **Squash commits** if requested before merge

## Areas Accepting Contributions

We welcome contributions in the following areas:

### ✅ CLI Tool & Commands

- New CLI commands or flags
- Improved error messages and user feedback
- CLI performance optimizations
- Cross-platform compatibility fixes

**Example PRs:**
- Add `crisk diff` command for comparing risk between branches
- Improve `crisk check` output formatting
- Add progress bars for long-running operations

### ✅ Metrics & Analysis

- New risk metrics (coupling, complexity, etc.)
- Improvements to existing metrics
- Metric validation and testing
- False positive reduction

**Example PRs:**
- Add cyclomatic complexity metric
- Improve co-change detection accuracy
- Add test coverage depth analysis

### ✅ Graph Parsers

- New language support (tree-sitter grammars)
- Parser improvements for existing languages
- Edge case handling in parsing
- Performance optimizations

**Example PRs:**
- Add Rust language support
- Improve TypeScript JSX parsing
- Fix Python decorator parsing

### ✅ Local Mode Infrastructure

- Docker Compose improvements
- Local deployment optimizations
- Database schema migrations
- Local setup automation

**Example PRs:**
- Reduce Docker image size
- Add health checks for services
- Improve init-local performance

### ✅ Testing & Quality

- Unit tests for core functionality
- Integration tests for CLI commands
- End-to-end tests for workflows
- Performance benchmarks

**Example PRs:**
- Add tests for graph ingestion
- Create integration tests for hooks
- Add benchmark suite for metrics

### ✅ Documentation

- Architecture documentation
- API documentation
- User guides and tutorials
- Code comments and examples

**Example PRs:**
- Document graph ontology design
- Create metric contribution guide
- Add troubleshooting guide

### ✅ Developer Experience

- Build process improvements
- Development tooling
- CI/CD pipeline enhancements
- Installation scripts

**Example PRs:**
- Add Makefile for common tasks
- Improve install.sh error handling
- Add pre-commit hooks for contributors

## Areas Not Accepting Contributions

The following areas are **closed to external contributions** as they contain proprietary business logic and competitive advantages:

### ❌ Cloud Infrastructure

- Multi-tenant architecture
- AWS Neptune integration
- Public repository caching system
- Branch delta optimization
- Reference counting and garbage collection

**Reason:** Core infrastructure for commercial cloud platform.

### ❌ ARC Database

- Incident catalog contents
- Architectural risk patterns
- Pattern fingerprinting algorithms
- Cross-organization learning

**Reason:** Proprietary dataset representing years of research and data collection.

### ❌ Phase 2 LLM Investigation

- Agentic graph navigation algorithms
- LLM prompt engineering and optimization
- Metric selection AI
- Investigation trace generation

**Reason:** Competitive advantage in intelligent risk assessment.

### ❌ Trust Infrastructure

- AI code provenance certificates
- Insurance underwriting algorithms
- Trust score calculation
- Certificate validation system

**Reason:** Core intellectual property for trust platform business model.

### ❌ Enterprise Features

- SSO and SAML integration
- Audit logging system
- Team management platform
- Billing and subscription management

**Reason:** Commercial features for enterprise customers.

**Note:** If you have ideas for these areas, please open a [Discussion](https://github.com/rohankatakam/coderisk-go/discussions) to share your thoughts. We may incorporate community feedback into our roadmap.

## Pull Request Process

### Before Submitting

- [ ] Code builds successfully (`go build ./...`)
- [ ] All tests pass (`go test ./...`)
- [ ] Code is formatted (`go fmt ./...`)
- [ ] Documentation is updated if needed
- [ ] Commit messages follow conventions
- [ ] Branch is rebased on latest main

### PR Checklist

- [ ] **Title** clearly describes the change
- [ ] **Description** explains what and why
- [ ] **Tests** are added or updated
- [ ] **Documentation** is updated
- [ ] **Breaking changes** are documented
- [ ] **Related issues** are linked

### PR Template

```markdown
## Summary
Brief description of changes

## Motivation
Why is this change needed?

## Changes
- List of changes made
- Another change

## Testing
How was this tested?

## Related Issues
Fixes #123
Closes #456
```

## Coding Standards

### Go Style Guide

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` for formatting
- Run `go vet` for static analysis

### Code Organization

```
coderisk-go/
├── cmd/crisk/           # CLI entry point
├── internal/            # Internal packages
│   ├── graph/           # Graph operations
│   ├── metrics/         # Risk metrics
│   ├── ingestion/       # Code parsing
│   └── hooks/           # Git hooks
├── test/                # Integration tests
└── dev_docs/            # Documentation
```

### Testing Standards

- **Unit tests:** Test individual functions/methods
- **Integration tests:** Test component interactions
- **E2E tests:** Test complete workflows

**Test naming:**
```go
func TestMetricCalculation(t *testing.T) { }
func TestMetricCalculation_WithNilInput_ReturnsError(t *testing.T) { }
```

**Coverage expectations:**
- New code: >80% coverage
- Critical paths: >90% coverage
- Use `go test -cover` to verify

### Error Handling

```go
// Good: Clear error messages with context
if err != nil {
    return fmt.Errorf("failed to parse file %s: %w", path, err)
}

// Bad: Generic error messages
if err != nil {
    return err
}
```

### Logging

```go
// Use structured logging
log.Info("Analyzing repository", "repo", repoPath, "files", fileCount)

// Not
log.Info(fmt.Sprintf("Analyzing repository %s with %d files", repoPath, fileCount))
```

## Community

### Communication Channels

- **GitHub Issues:** Bug reports and feature requests
- **GitHub Discussions:** Questions, ideas, and community chat
- **Pull Requests:** Code contributions and reviews

### Getting Help

- **Documentation:** Start with [dev_docs/](dev_docs/)
- **Issues:** Search existing issues for similar problems
- **Discussions:** Ask questions in GitHub Discussions

### Recognition

Contributors are recognized in:
- GitHub contributors list
- Release notes for significant contributions
- Special thanks in major milestones

## License

By contributing to CodeRisk, you agree that your contributions will be licensed under the [MIT License](LICENSE) for the open source components.

All contributions must:
- Be your original work or properly attributed
- Not violate any third-party licenses or intellectual property
- Include appropriate license headers if adding new files

---

**Thank you for contributing to CodeRisk!**

We're building the trust infrastructure for AI-generated code, and every contribution helps make AI coding safer for developers everywhere.

Questions? Open a [Discussion](https://github.com/rohankatakam/coderisk-go/discussions) or reach out via [GitHub Issues](https://github.com/rohankatakam/coderisk-go/issues).
