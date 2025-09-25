# CodeRisk Go

High-performance risk assessment engine for code changes, delivering sub-5-second analysis for enterprise teams.

[![Go Report Card](https://goreportcard.com/badge/github.com/rohankatakam/coderisk)](https://goreportcard.com/report/github.com/rohankatakam/coderisk)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-blue)](https://golang.org/)

## 🚀 Quick Start

### Prerequisites
- Go 1.21 or higher
- Git repository to analyze
- GitHub Personal Access Token ([get one here](https://github.com/settings/tokens))

### Installation

```bash
# Clone the repository
git clone https://github.com/rohankatakam/coderisk.git
cd coderisk

# Set up development environment (installs tools)
make setup

# Build the CLI
make build-cli

# Set up environment
./scripts/setup-env.sh

# Test on any Git repository
cd /path/to/your/repo
/path/to/coderisk/bin/crisk init --local-only
/path/to/coderisk/bin/crisk check
```

### Global Installation

```bash
make install
# Now use 'crisk' from anywhere
crisk --help
```

## 🏗️ Architecture

CodeRisk analyzes code changes using a multi-level approach:

- **Level 1**: Local cache-based analysis (<2s)
- **Level 2**: Selective API calls for context (<5s)
- **Level 3**: Full LLM-powered analysis (<10s)

```
coderisk/
├── cmd/                    # Application entrypoints
│   ├── crisk/             # CLI tool
│   └── crisk-server/      # API server (planned)
├── internal/              # Private application code
│   ├── github/           # GitHub API integration
│   ├── risk/             # Risk calculation engine
│   ├── storage/          # Database abstractions
│   ├── cache/            # Caching layer
│   ├── config/           # Configuration management
│   └── models/           # Domain models
├── scripts/              # Development and setup scripts
└── test/                 # Test files
```

## 💡 Usage

### Initialize Repository
```bash
# Auto-detect and connect to team cache
crisk init

# Use local-only mode (free, basic features)
crisk init --local-only

# Force full ingestion (creates team cache)
crisk init --repo https://github.com/owner/repo
```

### Analyze Risk
```bash
# Check current changes
crisk check

# Check specific commit
crisk check --commit abc123

# Detailed analysis
crisk check --level 3
```

### Configuration
```bash
# View current settings
crisk config list

# Set GitHub token
crisk config set github.token ghp_your_token

# View status
crisk status
```

## 🔧 Development

### Prerequisites
- Go 1.21+
- Make
- Git

### Setup Development Environment
```bash
# Install development tools and dependencies
make setup

# Run all quality checks and tests
make ci

# Quick development build
make dev

# Watch for changes (requires air)
make watch
```

### Development Workflow

1. **Make changes** to the code
2. **Test locally**: `make dev` and test with a repository
3. **Run quality checks**: `make ci`
4. **Clean for commit**: `make pre-push`
5. **Commit and push**

### Available Make Targets

| Target | Description |
|--------|-------------|
| `make setup` | Install development tools |
| `make build` | Build CLI and server |
| `make test` | Run all tests |
| `make lint` | Run linters |
| `make clean` | Remove build artifacts |
| `make clean-all` | Deep clean for fresh state |
| `make pre-push` | Validate before pushing |
| `make ci` | Run complete CI pipeline |

## 🧪 Testing

```bash
# Run all tests
make test

# Run with coverage
make coverage

# Run integration tests
make test-integration

# Run benchmarks
make benchmark
```

## 📦 Deployment Models

### Enterprise
- Self-hosted with custom LLM endpoints
- Full data control and security
- Custom risk policies

### Team (Recommended)
- Cloud SaaS with shared ingestion
- Cost-effective for teams
- Managed infrastructure

### OSS
- Public cache with sponsor funding
- Community-driven
- Open source projects

## ⚙️ Configuration

CodeRisk uses `.env` files for configuration:

```bash
# Copy example configuration
cp .env.example .env

# Edit configuration
vim .env
```

Key configuration options:
- `GITHUB_TOKEN`: GitHub Personal Access Token
- `OPENAI_API_KEY`: OpenAI API key (optional, for Level 3)
- `CODERISK_MODE`: `local`, `team`, `enterprise`, `oss`
- `STORAGE_TYPE`: `sqlite`, `postgres`

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Quick Contributor Setup

```bash
# Fork the repository on GitHub
git clone https://github.com/your-username/coderisk.git
cd coderisk

# Set up development environment
make setup

# Create feature branch
git checkout -b feature/your-feature

# Make changes and test
make ci

# Clean and validate before pushing
make pre-push

# Create pull request
```

### Code Quality Standards

- **Go fmt**: All code must be formatted
- **Go vet**: No vet warnings allowed
- **Linting**: Pass golangci-lint checks
- **Tests**: Maintain >80% coverage
- **Documentation**: Update README and docs

## 📊 Performance Targets

- **Level 1 Check**: <2 seconds (local cache)
- **Level 2 Check**: <5 seconds (API calls)
- **Level 3 Check**: <10 seconds (full analysis)

## 🔒 Security

CodeRisk handles sensitive information:
- GitHub tokens are never logged
- Local databases use secure permissions
- API calls use rate limiting
- No code is transmitted without explicit consent

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- GitHub API for repository data
- SQLite for local storage
- Cobra CLI framework
- The open source Go community

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/rohankatakam/coderisk/issues)
- **Discussions**: [GitHub Discussions](https://github.com/rohankatakam/coderisk/discussions)
- **Documentation**: [Wiki](https://github.com/rohankatakam/coderisk/wiki)

---

**Made with ❤️ for developers who care about code quality and risk management.**