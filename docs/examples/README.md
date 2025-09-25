# CodeRisk Examples

This directory contains practical examples of using CodeRisk in different scenarios.

## 🎯 Available Examples

### Basic Usage
- `basic-setup.md` - Simple local setup and first analysis
- `team-workflow.md` - Setting up team collaboration
- `ci-integration.md` - Integrating with CI/CD pipelines

### Advanced Scenarios
- `enterprise-deployment.md` - Large-scale enterprise setup
- `custom-policies.md` - Defining custom risk policies
- `performance-optimization.md` - Optimizing for large repositories

### Integration Examples
- `github-actions.md` - GitHub Actions workflow integration
- `pre-commit-hooks.md` - Git pre-commit hook setup
- `slack-notifications.md` - Risk alerts via Slack

## 🚀 Quick Start Example

```bash
# Clone and setup
git clone https://github.com/rohankatakam/coderisk.git
cd coderisk
make setup && make build

# Navigate to your project
cd /path/to/your/project

# Initialize and check
/path/to/coderisk/bin/crisk init --local-only
/path/to/coderisk/bin/crisk check
```

## 💡 Contributing Examples

Have a useful CodeRisk integration or workflow? Please contribute by:

1. Creating a new example file
2. Following the existing format
3. Including clear setup instructions
4. Adding it to this README
5. Submitting a pull request

---

**All examples are community contributions and maintained by the CodeRisk community.**