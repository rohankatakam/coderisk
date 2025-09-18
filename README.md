# CodeRisk 🛡️

**Know your risk before you ship**

CodeRisk is an AI-powered tool that predicts code regression risk before you commit, with a focus on migration safety and team scaling scenarios.

## Quick Start

```bash
# Install directly from GitHub
pip install git+https://github.com/rohankatakam/coderisk.git

# Or clone and install locally
git clone https://github.com/rohankatakam/coderisk.git
cd coderisk
pip install -e .

# Check risk of your current changes
crisk check

# Check risk of a specific commit
crisk commit abc123

# Get detailed analysis
crisk check --verbose
```

## What It Does

CodeRisk analyzes your code changes and provides:

- **Risk Score** (0-100) with tier classification (LOW/MEDIUM/HIGH/CRITICAL)
- **Blast Radius** - Which files and functions are impacted
- **Migration Risk** - Special detection for framework/dependency changes
- **Breaking Change Detection** - Will this break something downstream?
- **Regression Probability** - Based on historical patterns and team scaling factors

## The Problem

After 3 years at Oracle watching regression bugs destroy productivity, we built CodeRisk to prevent the "Oracle nightmare":

- **70% of developer time** spent fixing regressions instead of building features
- **One migration** creating a 6-month cascade of bugs
- **Customer escalations** consuming entire sprints
- **Codebases feeling totaled** - unfixable but too valuable to abandon

## How It Works

CodeRisk uses the **Regression Scaling Formula**:

```
Risk = Base Risk × Team Factor × Codebase Factor × Change Velocity × Migration Multiplier
```

### Risk Signals Analyzed

1. **Blast Radius (ΔDBR)** - Impact scope via dependency graph
2. **Co-Change Analysis (HDCC)** - Historical change coupling patterns
3. **Incident Adjacency** - Proximity to past incidents
4. **API Breaking Changes** - Public interface modifications
5. **Performance Risk** - Complexity and loop detection
6. **Test Coverage Gap** - Missing test coverage analysis
7. **Security Patterns** - Vulnerability pattern detection

## Example Output

```
╔═══════════════════════════════════════╗
║              CodeRisk                 ║
║        Know your risk before          ║
║            you ship                   ║
╚═══════════════════════════════════════╝

🔍 Analyzing repository: /path/to/your/repo

┌─ Risk Assessment ─────────────────────────┐
│ 🔶 Risk Level: HIGH (73.2/100)           │
│                                           │
│ High risk change (73/100). Major         │
│ concerns: blast_radius, api_breaking,     │
│ incident_adjacency                        │
│                                           │
│ Assessment completed in 1,847ms           │
└───────────────────────────────────────────┘

🎯 Top Concerns:
  1. Blast Radius
  2. Api Breaking
  3. Incident Adjacency

📊 Change Summary:
  • Files changed: 12
  • Lines added: 340
  • Lines deleted: 180

💡 Recommendations:
  • [HIGH] Add additional reviewer
    Given the high risk score, have a senior team member review this change
  • [HIGH] API compatibility check
    Verify backward compatibility of API changes
  • [MEDIUM] Add tests
    Improve test coverage for the modified code
```

## CLI Commands

### `crisk check`
Analyze uncommitted changes in your working tree.

**Options:**
- `--repo, -r` - Repository path (default: current directory)
- `--json` - Output results as JSON
- `--verbose, -v` - Show detailed signal analysis

### `crisk commit <sha>`
Analyze a specific commit.

### `crisk init`
Initialize CodeRisk for a repository (builds CodeGraph).

## JSON Output

Use `--json` for programmatic integration:

```json
{
  "tier": "HIGH",
  "score": 73.2,
  "confidence": 0.82,
  "total_regression_risk": 15.7,
  "top_concerns": ["blast_radius", "api_breaking", "incident_adjacency"],
  "explanation": "High risk change (73/100). Major concerns: blast_radius, api_breaking, incident_adjacency",
  "scaling_factors": {
    "team_factor": 1.6,
    "codebase_factor": 1.8,
    "change_velocity": 1.3,
    "migration_multiplier": 2.0
  }
}
```

## Integration

### Git Hooks

Add to `.git/hooks/pre-commit`:

```bash
#!/bin/bash
crisk check --json > /tmp/risk.json
RISK_SCORE=$(cat /tmp/risk.json | jq -r '.score')

if (( $(echo "$RISK_SCORE > 80" | bc -l) )); then
    echo "🔴 HIGH RISK COMMIT (Score: $RISK_SCORE)"
    echo "Consider additional review before pushing"
    exit 1
fi
```

### GitHub Actions

```yaml
name: Risk Assessment
on: [pull_request]

jobs:
  risk-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Install CodeRisk
        run: pip install crisk
      - name: Assess Risk
        run: crisk check --json
```

## Architecture

CodeRisk is built on [Cognee](https://cognee.ai), leveraging:

- **CodeGraph** - Repository analysis and dependency tracking
- **Temporal Awareness** - Historical pattern analysis
- **Custom Pipelines** - Risk assessment workflows
- **DataPoints** - Structured knowledge representation
- **Feedback Systems** - Continuous learning

## Target Users

### Primary: High Regression Risk Teams
- Team size: 8-30 developers
- Codebase: 200K-2M LOC
- Change velocity: 50+ commits/week
- **Annual Revenue Impact**: $100K-10M per incident

### Secondary: Migration Teams
- Any team size planning major migration
- Timeline pressure (quarterly deadlines)
- Business-critical applications
- Previous migration failures/delays

### Tertiary: Fast-Growing Startups
- Team doubled in last year
- Series A/B funded
- Moving from "move fast" to "don't break things"
- First platform stability hire

## Development

```bash
# Clone and install for development
git clone https://github.com/rohankatakam/coderisk.git
cd coderisk
pip install -e .

# Install development dependencies
pip install -e ".[dev]"

# Run tests (coming soon)
pytest

# Install with Cognee integration
pip install -e ".[cognee]"
```

## Requirements

- Python 3.8+
- Git repository
- Cognee API access (for advanced features)

## License

MIT License - see [LICENSE](LICENSE) file.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## Support

- 📧 Email: rohan@coderisk.dev
- 🐛 Issues: [GitHub Issues](https://github.com/rohankatakam/coderisk/issues)
- 💬 Discord: [CodeRisk Community](https://discord.gg/coderisk)

---

**Remember**: If just 10 developers say "holy shit this is real" in Week 1, we've found product-market fit.

**Start using tonight. Prevent the Oracle nightmare.**