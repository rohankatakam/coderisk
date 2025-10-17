# Configuration Management Strategy

**Last Updated:** 2025-10-13
**Owner:** Engineering Team
**Status:** Active - Ready for Implementation
**Phase:** Pre-Launch Setup

> **📘 Cross-reference:** See [packaging_and_distribution.md](packaging_and_distribution.md) for distribution strategy and [open_core_strategy.md](../00-product/open_core_strategy.md) for what's open vs proprietary.

---

## Executive Summary

**CORRECTED:** CodeRisk requires both OpenAI API key AND graph database for full functionality.

**What's Required:**
1. **OpenAI API Key** - LLM reasoning (Phase 1 + Phase 2)
2. **Graph Database** - Code relationships (Phase 1 + Phase 2)
3. **Phase 0 Pre-Filter** - Fast heuristics (<50ms, not standalone)

This document defines how users configure CodeRisk, ensuring clear setup instructions and good developer experience.

**Key Principles:**
- ✅ Phase 0 is a pre-filter only (<50ms), NOT a standalone mode
- ✅ Both API key AND Docker/graph required for risk assessment
- ✅ Clear 17-minute setup flow (one-time per repo)
- ✅ Transparent costs: $0.03-0.05/check (BYOK model)

---

## Configuration Requirements

### Full Setup Required (Both Components)

**What's Required for CodeRisk to Work:**

1. **OpenAI API Key** - LLM for reasoning
   - Synthesizes evidence from graph
   - Decides what to investigate next
   - Assesses confidence levels
   - Cost: $0.03-0.05/check

2. **Graph Database** - Stores relationships
   - Tree-sitter AST (code structure)
   - Git history (CO_CHANGED relationships)
   - Temporal patterns (files changed together)
   - Setup: Docker + 15 min ingestion

3. **Phase 0 Pre-Filter** - Optimization (<50ms)
   - Detects security keywords → escalate
   - Detects docs-only → reduce priority
   - NOT a replacement for agentic investigation
   - NOT a standalone analysis mode

### Complete Setup (17 Minutes One-Time)

**Step-by-step:**

```bash
# 1. Install CLI (30 seconds)
brew install rohankatakam/coderisk/crisk

# 2. Configure API key (30 seconds)
crisk configure
# Or manually:
export OPENAI_API_KEY="sk-..."

# 3. Start Docker infrastructure (2 minutes)
docker compose up -d
# Starts: Neo4j (graph), PostgreSQL (metadata), Redis (cache)

# 4. Initialize repository (10-15 minutes)
cd your-repo
crisk init-local
# Builds graph: Tree-sitter AST + Git history
# Creates: IMPORTS, CALLS, CO_CHANGED relationships

# 5. Check for risks (2-5 seconds)
crisk check
# ✅ Phase 0 + Phase 1 + Phase 2 complete
```

**What you get after setup:**
- ✅ Full agentic graph navigation
- ✅ Temporal coupling detection
- ✅ <3% false positive rate
- ✅ 2-5 second checks
- ✅ Transparent costs ($0.03-0.05/check)

**Output example:**
```
✅ Phase 0: Pre-filter (0.05s)
✅ Phase 1: Baseline assessment (1.2s) - Queries graph
✅ Phase 2: Deep investigation (2.3s) - Navigates graph

Risk: HIGH
Evidence: 12 dependencies + 0.87 co-change + security keywords
Confidence: 87%
```

---

## API Key Management

### Current Implementation (install.sh)

**Status:** ✅ Already implemented, works well

**Features:**
1. Interactive prompt during installation
2. Auto-detects shell (zsh, bash, other)
3. Adds to appropriate config file (~/.zshrc, ~/.bashrc)
4. Checks for existing key, offers to update
5. Exports for current session
6. Graceful skip option

**User Flow:**
```bash
./install.sh

# Prompts:
# 🔑 API Key Setup (Optional but Recommended)
# Do you have an OpenAI API key? (y/n): y
# Enter your OpenAI API key: sk-...
# ✅ Added OPENAI_API_KEY to ~/.zshrc
```

**Edge Cases Handled:**
- Key already exists → offer to update
- No key provided → show instructions for later
- Wrong shell detected → fallback to ~/.profile

### Improvements for GoReleaser Install

**Problem:** GoReleaser doesn't run install.sh, users get bare binary

**Solution:** Add `crisk configure` command

**Usage:**
```bash
# Install via Homebrew
brew install crisk

# First run without key
crisk check
# ⚠️  Phase 2 unavailable: OPENAI_API_KEY not set
# Run `crisk configure` to set up your API key

# Interactive setup
crisk configure
# 🔑 Let's set up your OpenAI API key
# Enter key: sk-...
# ✅ Saved to ~/.config/crisk/config.toml
# ✅ Added to ~/.zshrc
```

**Config File Format:**
```toml
# ~/.config/crisk/config.toml
[api]
openai_key = "sk-..."

[local]
docker_required = true
neo4j_uri = "bolt://localhost:7687"
postgres_uri = "postgresql://localhost:5432/coderisk"
redis_uri = "redis://localhost:6379"
```

### Alternative: crisk config

**Subcommands:**
```bash
crisk config set openai_key sk-...
crisk config get openai_key
crisk config show
crisk config reset
```

**Benefits:**
- Familiar pattern (like `git config`)
- Can set multiple config values
- Easy to script
- Non-interactive option

---

## Environment Variable Precedence

**Order (highest to lowest):**

1. **Command-line flag** (future)
   ```bash
   crisk check --openai-key=sk-...
   ```

2. **Environment variable**
   ```bash
   export OPENAI_API_KEY="sk-..."
   crisk check
   ```

3. **Config file**
   ```bash
   # ~/.config/crisk/config.toml
   [api]
   openai_key = "sk-..."
   ```

4. **Interactive prompt**
   ```bash
   crisk check
   # ⚠️  OPENAI_API_KEY not set
   # Enter key (or press Enter to skip): sk-...
   ```

**Rationale:**
- CLI flag: Per-command override (CI/CD, testing)
- Env var: Current session override (temporary key)
- Config file: Persistent user preference (default)
- Prompt: Fallback for first-time users

---

## Docker Infrastructure Setup

### Current State

**Files:**
- `docker-compose.yml` - Defines Neo4j, PostgreSQL, Redis
- `.env.example` - Template for environment variables

**Usage:**
```bash
docker compose up -d
```

**Issues:**
1. Requires Docker installed (barrier for some users)
2. 5-15 minute setup time (Neo4j, PostgreSQL, Redis)
3. Resource usage (RAM, disk space)

### Improvement: Optional Docker

**Message Hierarchy:**

**1. Phase 0 (No Docker):**
```bash
crisk check
# ✅ Phase 0 analysis complete
# 💡 Install Docker for full graph-based analysis
```

**2. Docker Not Running:**
```bash
crisk init-local
# ❌ Docker not running
# Run: docker compose up -d
```

**3. Docker Running, No Init:**
```bash
crisk check --explain
# ✅ Phase 0 + Phase 2 (without graph)
# 💡 Run `crisk init-local` to enable graph analysis
```

**4. Fully Initialized:**
```bash
crisk check --explain
# ✅ Full analysis with graph data
```

### Error Scenarios and Handling

**Scenario 1: API Key Not Set**
```bash
crisk check
# Error: OPENAI_API_KEY not set
#
# CodeRisk requires an OpenAI API key for LLM-guided risk assessment.
#
# Setup options:
#   1. Interactive: crisk configure
#   2. Manual: export OPENAI_API_KEY="sk-..."
#   3. Get key: https://platform.openai.com/api-keys
#
# Cost: $0.03-0.05 per check (BYOK model)
```

**Scenario 2: Docker Not Running**
```bash
export OPENAI_API_KEY="sk-..."
crisk check
# Error: Cannot connect to graph database (Neo4j not reachable)
#
# CodeRisk requires a graph database for agentic navigation.
#
# Start infrastructure:
#   docker compose up -d
#   docker compose wait neo4j postgres redis
#
# Then initialize your repository:
#   crisk init-local
```

**Scenario 3: Graph Not Initialized**
```bash
export OPENAI_API_KEY="sk-..."
docker compose up -d
crisk check
# Error: Graph not initialized for this repository
#
# Initialize repository (one-time, 10-15 minutes):
#   crisk init-local
#
# This builds the code graph the LLM will navigate.
```

**Scenario 4: Everything Set Up**
```bash
export OPENAI_API_KEY="sk-..."
docker compose up -d
crisk init-local  # First time only
crisk check
# ✅ Phase 0 + Phase 1 + Phase 2 complete (3.5s)
# Risk: HIGH - 12 dependencies + 0.87 co-change
```

### Docker Compose Improvements

**Add health checks:**
```yaml
services:
  neo4j:
    healthcheck:
      test: ["CMD", "cypher-shell", "RETURN 1"]
      interval: 10s
      timeout: 5s
      retries: 5

  postgres:
    healthcheck:
      test: ["CMD", "pg_isready"]
      interval: 10s
      timeout: 5s
      retries: 5
```

**Add startup script:**
```bash
#!/bin/bash
# wait-for-services.sh
echo "⏳ Waiting for services..."
docker compose up -d
docker compose wait neo4j postgres redis
echo "✅ Services ready! Run: crisk init-local"
```

---

## Configuration File Structure

### Location

**Precedence:**
1. `./.crisk/config.toml` (repository-specific, git-ignored)
2. `~/.config/crisk/config.toml` (user-wide)
3. `/etc/crisk/config.toml` (system-wide, rare)

### Format (TOML)

```toml
# ~/.config/crisk/config.toml

[api]
# OpenAI API key for Phase 2 LLM analysis
openai_key = "sk-..."

# Optional: Anthropic for alternative LLM (future)
# anthropic_key = "sk-ant-..."

[local]
# Docker infrastructure
docker_required = true
neo4j_uri = "bolt://localhost:7687"
neo4j_user = "neo4j"
neo4j_password = "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
postgres_uri = "postgresql://coderisk:coderisk_password@localhost:5432/coderisk"
redis_uri = "redis://localhost:6379"

[cloud]
# Cloud platform (future)
# api_endpoint = "https://api.coderisk.dev"
# api_key = "crsk_..."

[ui]
# Verbosity level: quiet, standard, verbose, debug
verbosity = "standard"

# Color output
color = true

# AI mode (JSON output)
ai_mode = false

[hooks]
# Pre-commit hook enabled
enabled = true

# Fail commit on CRITICAL risk
fail_on_critical = true

# Fail commit on HIGH risk (optional, default: false)
fail_on_high = false
```

### YAML Alternative (Optional)

Some users prefer YAML:

```yaml
# ~/.config/crisk/config.yaml
api:
  openai_key: "sk-..."

local:
  docker_required: true
  neo4j_uri: "bolt://localhost:7687"
  postgres_uri: "postgresql://localhost:5432/coderisk"
  redis_uri: "redis://localhost:6379"

ui:
  verbosity: standard
  color: true
```

**Recommendation:** Support both TOML and YAML, prefer TOML (simpler parsing).

---

## First-Run Experience

### Scenario 1: Brew Install (No Setup Yet)

```bash
brew install crisk
cd my-repo
crisk check
```

**Output:**
```
❌ Error: OPENAI_API_KEY not set

CodeRisk requires an OpenAI API key for LLM-guided risk assessment.

Setup (takes 17 minutes one-time per repo):

Step 1: Configure API key (30 seconds)
  crisk configure
  # Or: export OPENAI_API_KEY="sk-..."
  # Get key: https://platform.openai.com/api-keys

Step 2: Start infrastructure (2 minutes)
  docker compose up -d

Step 3: Initialize repository (10-15 minutes)
  crisk init-local

Step 4: Check for risks (2-5 seconds)
  crisk check

Cost: $0.03-0.05 per check (BYOK model)
Learn more: https://github.com/rohankatakam/coderisk-go
```

**User thinks:** "Clear instructions! I need to set up API key + Docker. 17 minutes one-time is reasonable."

### Scenario 2: API Key Set, Docker Not Running

```bash
export OPENAI_API_KEY="sk-..."
crisk check
```

**Output:**
```
❌ Error: Cannot connect to graph database (Neo4j not reachable)

CodeRisk requires a graph database for agentic navigation.
The LLM needs the graph to find coupling, temporal patterns, and dependencies.

Next steps:

1. Start Docker infrastructure (2 minutes):
   docker compose up -d
   docker compose wait neo4j postgres redis

2. Initialize repository (10-15 minutes, one-time):
   crisk init-local

Then you can run:
   crisk check  # 2-5 seconds per check
```

**User thinks:** "Got it - Docker is required, not optional. Let me start it."

### Scenario 3: Full Setup Complete

```bash
export OPENAI_API_KEY="sk-..."
docker compose up -d
crisk init-local
# ... 10 minutes later ...
crisk check --explain
```

**Output:**
```
🔍 CodeRisk Investigation Report

✅ Phase 0: Static analysis (0.2s)
✅ Phase 1: Graph analysis (1.5s)
✅ Phase 2: LLM investigation (2.3s)

📊 Graph Analysis:
   - Temporal coupling: auth.ts + session.ts (changed together 15 times)
   - High coupling: auth.ts → 8 dependencies
   - Past incidents: 2 similar auth bugs in last 6 months

📝 Risk Assessment: apps/web/src/auth.ts

HIGH RISK: Authentication change in high-coupling area

This file has caused 2 incidents historically and is
temporally coupled with session.ts (which wasn't changed).
Recommendation: Also review session.ts for consistency.

Critical patterns detected:
- JWT token validation logic modified
- Session expiry handling changed
- Missing corresponding database migration

⚠️  Action required:
   1. Add unit tests for token validation
   2. Review session.ts for consistency
   3. Create database migration for schema changes

📈 Confidence: 87% (based on graph + LLM analysis)
```

**User thinks:** "Wow, that's really helpful! Worth the Docker setup."

---

## Configuration Commands (New)

### crisk configure

**Interactive setup wizard:**

```bash
crisk configure
```

**Flow:**
```
🔧 CodeRisk Configuration Wizard
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Step 1/3: API Key Setup

CodeRisk uses OpenAI for deep risk investigation (Phase 2).
Phase 0 works without it, but Phase 2 provides much richer insights.

Do you have an OpenAI API key? (y/n): y
Enter your API key (sk-...): sk-...
✅ API key saved to ~/.config/crisk/config.toml

Step 2/3: Docker Infrastructure

For full graph-based analysis, you'll need Docker.
This enables temporal coupling detection and incident correlation.

Install Docker? (y/n/skip): skip
⏭️  Skipped. You can run `docker compose up -d` later.

Step 3/3: Default Settings

Verbosity level (quiet/standard/verbose): standard
Color output (y/n): y
Pre-commit hook (y/n): y

✅ Configuration complete!

Next steps:
1. Run: crisk check (Phase 0 + Phase 2)
2. For full analysis: docker compose up -d && crisk init-local
```

### crisk config

**Get/set individual values:**

```bash
# Show all config
crisk config show

# Get specific value
crisk config get api.openai_key
# sk-...

# Set value
crisk config set api.openai_key sk-new-key
# ✅ Updated api.openai_key

# Reset to defaults
crisk config reset
# ⚠️  This will delete your config file. Continue? (y/n): y
# ✅ Config reset to defaults
```

---

## Documentation Updates Required

### README.md

**Update Quick Start:**

```markdown
## Quick Start

### Installation (17 minutes one-time per repo)

#### Step 1: Install CLI (30 seconds)
```bash
brew install rohankatakam/coderisk/crisk
```

#### Step 2: Configure API key (30 seconds)
```bash
crisk configure
# Or manually:
export OPENAI_API_KEY="sk-..."
```

[Get API key →](https://platform.openai.com/api-keys)

#### Step 3: Start infrastructure (2 minutes)
```bash
docker compose up -d
```

#### Step 4: Initialize repository (10-15 minutes)
```bash
cd your-repo
crisk init-local
```

#### Step 5: Check for risks (2-5 seconds)
```bash
crisk check                    # Quick baseline assessment
crisk check --explain          # Detailed LLM investigation report
crisk check --ai-mode          # JSON output for AI tools
```

### What You Get

After setup:
- ✅ <3% false positive rate (vs 10-20% industry standard)
- ✅ 2-5 second checks (agentic graph search)
- ✅ Temporal coupling detection (files changed together)
- ✅ Transparent costs ($0.03-0.05/check, BYOK)
- ✅ Self-hosted privacy (all data stays local)

**Cost:** ~$3-5/month for 100 checks (just OpenAI API costs)
```

### Website (coderisk.dev)

**Update installation section:**

```markdown
## Installation

Open Source AI Code Risk Assessment
LLM-powered agentic graph search - <3% false positives

### Setup (17 minutes one-time per repo)

#### Prerequisites
- Docker Desktop installed
- OpenAI API key ([get one here](https://platform.openai.com/api-keys))

#### Step 1: Install CLI (30 seconds)
```bash
brew install rohankatakam/coderisk/crisk
```

#### Step 2: Configure API Key (30 seconds)
```bash
crisk configure
# Enter your OpenAI API key: sk-...
```

Or manually:
```bash
export OPENAI_API_KEY="sk-..."
# Add to ~/.zshrc or ~/.bashrc for persistence
```

#### Step 3: Start Infrastructure (2 minutes)
```bash
docker compose up -d
# Starts: Neo4j (graph), PostgreSQL (metadata), Redis (cache)
```

Wait for services to be ready:
```bash
docker compose ps
# All services should show "running (healthy)"
```

#### Step 4: Initialize Repository (10-15 minutes)
```bash
cd your-repo
crisk init-local
# Progress: Parsing files → Building graph → Analyzing git history
# This builds the graph the LLM will navigate
```

#### Step 5: Start Checking (2-5 seconds)
```bash
crisk check
# ✅ Phase 0 + Phase 1 + Phase 2 complete
# Risk assessment with <3% FP rate
```

### What You Get

After setup, every `crisk check`:
- Analyzes your changes in 2-5 seconds
- LLM navigates your code graph
- Finds coupling, temporal patterns, risks
- Costs $0.03-0.05 (your OpenAI account)

**Setup time breakdown:**
- Install CLI: 30 sec
- Configure API key: 30 sec
- Start Docker: 2 min
- Graph ingestion: 10-15 min (depends on repo size)

**Total: ~17 minutes** (one-time per repo)
```

---

## Implementation Checklist

### Phase 1: Config File Support (Week 1)

- [ ] Create `internal/config/` package
- [ ] Support TOML parsing (`github.com/pelletier/go-toml`)
- [ ] Support YAML parsing (optional, `gopkg.in/yaml.v3`)
- [ ] Config file precedence (repo → user → system)
- [ ] Environment variable override
- [ ] Test config loading with various edge cases

### Phase 2: crisk configure Command (Week 1)

- [ ] Add `configure` subcommand to CLI
- [ ] Interactive wizard (API key, Docker, settings)
- [ ] Shell detection and config file updates
- [ ] Non-interactive mode: `crisk configure --key=sk-...`
- [ ] Validation (check API key format, Docker status)

### Phase 3: crisk config Command (Week 1)

- [ ] Add `config` subcommand with `get`, `set`, `show`, `reset`
- [ ] Dot notation for nested keys (`api.openai_key`)
- [ ] Secure storage (encrypt API keys in config file?)
- [ ] Config migration (if format changes)

### Phase 4: Enhanced Messaging (Week 2)

- [ ] Phase 0 output shows "API key not set" tip
- [ ] Phase 2 output shows "Docker not running" tip
- [ ] `crisk check` detects missing components, suggests setup
- [ ] Color-coded messages (green=good, yellow=optional, red=error)

### Phase 5: Docker Integration (Week 2)

- [ ] Auto-detect Docker running (`docker ps` check)
- [ ] Health check before `init-local`
- [ ] Better error messages (Docker not installed vs not running)
- [ ] Add `crisk doctor` command to diagnose issues

### Phase 6: Documentation (Week 2)

- [ ] Update README.md with tier-based instructions
- [ ] Update website with progressive setup guide
- [ ] Add troubleshooting guide (common config issues)
- [ ] Create video walkthrough (setup in 3 minutes)

---

## Related Documents

**Implementation:**
- [packaging_and_distribution.md](packaging_and_distribution.md) - Distribution strategy
- [BACKEND_PACKAGING_PROMPT.md](BACKEND_PACKAGING_PROMPT.md) - Packaging Claude Code prompt

**Product:**
- [open_core_strategy.md](../00-product/open_core_strategy.md) - Open source positioning
- [developer_experience.md](../00-product/developer_experience.md) - UX design

---

**Last Updated:** 2025-10-13
**Next Steps:** Implement `crisk configure` and `crisk config` commands
