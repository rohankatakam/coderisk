# Claude Code Prompt: Configuration Management & User Onboarding (PROFESSIONAL SECURITY TIER)

**Session Type:** Backend (coderisk-go repository)
**Estimated Time:** 4-5 hours (includes OS keychain integration)
**Phase:** Professional Security & User Experience
**Can Run:** In parallel with packaging session (but after packaging if modifying CLI structure)
**Reference Docs:**
- [IMPROVED_API_KEY_SETUP.md](IMPROVED_API_KEY_SETUP.md) - Complete API key strategy
- [KEYCHAIN_INTEGRATION_GUIDE.md](KEYCHAIN_INTEGRATION_GUIDE.md) - Keychain implementation details
- [configuration_management.md](configuration_management.md) - Configuration strategy
- [developer_experience.md](../00-product/developer_experience.md) - UX principles

---

## Context

**IMPORTANT CORRECTION:** CodeRisk requires BOTH OpenAI API key AND graph database to function.

**What's Required:**
1. **OpenAI API key** - LLM reasoning (Phase 1 + Phase 2)
2. **Graph database** - Code relationships (Phase 1 + Phase 2)
3. **Phase 0** - Pre-filter only (<50ms, NOT standalone)

**Setup flow (17 minutes one-time per repo):**
1. Install CLI (30 seconds)
2. Configure API key (30 seconds)
3. Start Docker (2 minutes)
4. Initialize repository (10-15 minutes)
5. Check for risks (2-5 seconds)

The LLM needs the graph to navigate - without it, there are no IMPORTS, CALLS, CO_CHANGED relationships to query. Phase 0 is just a pre-filter to detect security keywords, docs-only changes, and configuration selection. It's NOT a replacement for the full agentic investigation.

Read these documents first:
- [CORRECTED_LAUNCH_STRATEGY_V2.md](../../CORRECTED_LAUNCH_STRATEGY_V2.md) - Accurate requirements and setup
- [configuration_management.md](configuration_management.md) - Complete configuration strategy (now corrected)

---

## Objective

Improve user onboarding with **professional-grade security** by implementing:
1. **OS Keychain Integration** - Store API keys securely (no plaintext storage)
2. `crisk configure` - Interactive setup wizard with keychain support (guides through full 17-minute setup)
3. `crisk config` - Get/set configuration values with keychain integration
4. `crisk migrate-to-keychain` - Migrate existing plaintext keys to secure storage
5. Enhanced error messages - Show what's missing with security guidance
6. Configuration file support - YAML via Viper (persistent settings, API key optional)

**Professional Security Features (Tier 3 - Best Practice):**
- ✅ API keys stored in OS keychain (macOS Keychain, Windows Credential Manager, Linux Secret Service)
- ✅ No plaintext storage in config files (except opt-in for CI/CD)
- ✅ Multi-level precedence: env var > keychain > config file > .env
- ✅ Cross-platform support (macOS, Windows, Linux)
- ✅ Migration from old approach (shell config files)

**Key messaging:**
- Both API key AND Docker required (not optional)
- 17-minute one-time setup per repo
- Cost: $0.03-0.05/check (BYOK model)
- **Secure by default**: API key stored in OS keychain, not plaintext

---

## Tasks

### 0. Add Dependencies

**Add go-keyring for OS keychain support:**

```bash
go get github.com/zalando/go-keyring@latest
```

This adds cross-platform keychain support:
- macOS: Keychain Services API
- Windows: Windows Credential Manager API
- Linux: freedesktop.org Secret Service API (requires libsecret)

**Note:** We already have Viper and godotenv in go.mod - use these instead of TOML parser.

### 1. Configuration File Infrastructure (ALREADY EXISTS - EXTEND IT)

**IMPORTANT:** `internal/config/config.go` already exists with comprehensive Viper-based configuration!

**What you need to ADD:**

```
internal/config/
├── config.go       # ALREADY EXISTS - extend with keychain support
├── keyring.go      # NEW - OS keychain interface (see KEYCHAIN_INTEGRATION_GUIDE.md)
```

**Extend existing APIConfig struct in `config.go`:**

```go
type APIConfig struct {
    OpenAIKey     string `yaml:"openai_key"`
    OpenAIModel   string `yaml:"openai_model"`
    UseKeychain   bool   `yaml:"use_keychain"`   // NEW: Prefer keychain over config file
    CustomLLMURL  string `yaml:"custom_llm_url"`
    CustomLLMKey  string `yaml:"custom_llm_key"`
    EmbeddingURL  string `yaml:"embedding_url"`
    EmbeddingKey  string `yaml:"embedding_key"`
}
```

**Update applyEnvOverrides() function:**

```go
// applyEnvOverrides in config.go - UPDATE THIS FUNCTION
func applyEnvOverrides(cfg *Config) {
    // ... existing GitHub config code ...

    // API configuration - UPDATED FOR KEYCHAIN
    // Precedence: 1. Env var (highest) 2. Keychain 3. Config file (lowest)

    if key := os.Getenv("OPENAI_API_KEY"); key != "" {
        // Environment variable has highest precedence (for CI/CD)
        cfg.API.OpenAIKey = key
    } else {
        // Try keychain if no env var
        km := NewKeyringManager()
        if km.IsAvailable() {
            if keychainKey, err := km.GetAPIKey(); err == nil && keychainKey != "" {
                cfg.API.OpenAIKey = keychainKey
            }
        }
        // If still empty, config file value is used (loaded earlier)
    }

    // ... rest of function unchanged ...
}
```

**File locations (already supported by existing config.go):**
1. `.env.local` (repo-specific overrides, highest precedence for files)
2. `.env` (repo-level)
3. `.coderisk/config.yaml` (repo-specific config)
4. `~/.coderisk/config.yaml` (user-wide config)
5. `~/.coderisk/.env` (user-wide .env)

**Precedence order (with keychain):**
1. Environment variable `OPENAI_API_KEY` (highest, for CI/CD)
2. OS Keychain (second, for local dev secure storage)
3. Config file `~/.coderisk/config.yaml` (third, opt-in plaintext)
4. .env files (lowest, CI/CD or repo-specific)

See KEYCHAIN_INTEGRATION_GUIDE.md for complete implementation details.

### 2. Implement OS Keychain Interface

**Create `internal/config/keyring.go`:**

See [KEYCHAIN_INTEGRATION_GUIDE.md](KEYCHAIN_INTEGRATION_GUIDE.md) Section "Step 2: Implement Keyring Interface" for complete implementation.

**Key functions to implement:**
- `SaveAPIKey(apiKey string) error` - Store in OS keychain
- `GetAPIKey() (string, error)` - Retrieve from OS keychain
- `DeleteAPIKey() error` - Remove from OS keychain
- `IsAvailable() bool` - Check if keychain is available (false on headless systems)
- `Get APIKeySource(cfg *Config) KeySourceInfo` - Determine where API key is stored
- `MaskAPIKey(apiKey string) string` - Mask for display

###3. crisk configure Command (WITH KEYCHAIN SUPPORT)

**Add new subcommand: `cmd/crisk/configure.go`**

See [KEYCHAIN_INTEGRATION_GUIDE.md](KEYCHAIN_INTEGRATION_GUIDE.md) Section "Step 4: Implement `crisk configure` Command" for complete code.

**Key features:**
- Interactive 4-step wizard
- Keychain storage option (if available)
- Validates API key format (starts with sk-)
- Falls back to config file on headless systems
- Shows where key is stored (platform-specific paths)

**Interactive wizard flow (WITH KEYCHAIN):**

```
$ crisk configure

🔧 CodeRisk Configuration Wizard
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

CodeRisk Setup (17 minutes one-time per repo)

CodeRisk requires:
  ✅ OpenAI API key - LLM reasoning ($0.03-0.05/check)
  ✅ Graph database - Code relationships (Docker + Neo4j)
  ✅ Phase 0 pre-filter - Fast heuristics (<50ms)

Without both requirements, CodeRisk cannot perform risk assessment.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Step 1/4: API Key Setup (30 seconds)

Enter your OpenAI API key (starts with sk-...): sk-...

🔒 Secure Storage Options:
  1. OS Keychain (recommended, encrypted, secure)
  2. Config file (plaintext, not recommended for local dev)

Choice (1/2): 1
✅ API key saved to OS keychain (secure)
   📍 macOS Keychain Access.app → 'CodeRisk'
   (or Windows Credential Manager / Linux Secret Service)

Step 2/4: Docker Infrastructure (2 minutes)

CodeRisk requires Docker for the graph database.
The LLM needs the graph to navigate code relationships.

Current Docker status: Not running ❌

Start Docker now? (y/n): y
⏳ Starting Docker services...
   docker compose up -d
   docker compose wait neo4j postgres redis
✅ Docker services ready!

Step 3/4: Repository Initialization (10-15 minutes)

Initialize graph for this repository?
  • Builds code graph (Tree-sitter AST)
  • Analyzes git history (CO_CHANGED relationships)
  • Creates IMPORTS, CALLS, CO_CHANGED edges

Current directory: /Users/you/my-repo

Initialize now? (y/n): y
⏳ Initializing repository...
   crisk init-local
   Progress: Parsing files → Building graph → Analyzing history
✅ Graph initialized! (took 12 minutes)

Step 4/4: Default Settings

Verbosity level (quiet/standard/verbose) [standard]: standard
Color output (y/n) [y]: y
Enable pre-commit hook (y/n) [y]: y

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ Setup complete!

What you get:
  ✅ <3% false positive rate
  ✅ 2-5 second checks (after setup)
  ✅ Agentic graph navigation
  ✅ Transparent costs ($0.03-0.05/check)

Next steps:
  1. Run a check: crisk check
  2. Detailed investigation: crisk check --explain
  3. Pre-commit hook: Already installed!

Cost: ~$3-5/month for 100 checks (just OpenAI API)
📚 Documentation: https://github.com/rohankatakam/coderisk-go
```

**Non-interactive mode:**
```bash
# Full setup in one command
crisk configure --key=sk-... --start-docker --init-repo --yes

# Skip repo init (do it later)
crisk configure --key=sk-... --start-docker --skip-init --yes
```

### 4. crisk config Command (WITH KEYCHAIN SUPPORT)

**Add new subcommand: `cmd/crisk/config.go`**

See [KEYCHAIN_INTEGRATION_GUIDE.md](KEYCHAIN_INTEGRATION_GUIDE.md) Section "Step 5: Implement `crisk config` Commands" for complete code.

**Key features:**
- `crisk config show` - Shows config with security indicators
- `crisk config get <key> --show-source` - Shows where value is stored
- `crisk config set <key> <value> --use-keychain` - Store in keychain
- `crisk config set <key> <value> --no-keychain` - Store in config file

**Subcommands:**

```bash
# Show all configuration (WITH SECURITY INDICATORS)
crisk config show
# Current Configuration:
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# API Key: sk-...abc123
#   Source: Stored securely in OS keychain ✅
#   Security: ✅ Secure storage
# Model: gpt-4o-mini
# Daily Limit: $2.00
# Monthly Limit: $60.00

# Get specific value with source info
crisk config get api.openai_key --show-source
# sk-proj...abc123
# Source: keychain
# Security: ✅ Secure

# Set value in KEYCHAIN (secure, recommended)
crisk config set api.openai_key sk-new-key --use-keychain
# ✅ API key saved to OS keychain (secure)

# Set value in CONFIG FILE (plaintext, for CI/CD)
crisk config set api.openai_key sk-new-key --no-keychain
# ✅ API key saved to config file (plaintext)
#    💡 For better security, use: --use-keychain flag
```

### 5. crisk migrate-to-keychain Command

**Add new subcommand: `cmd/crisk/migrate.go`**

See [KEYCHAIN_INTEGRATION_GUIDE.md](KEYCHAIN_INTEGRATION_GUIDE.md) Section "Step 6: Implement Migration Command" for complete code.

**Purpose:** Migrate existing API keys from plaintext storage (shell config, config file, .env) to secure OS keychain.

**Features:**
- Detects where API key is currently stored
- Migrates to OS keychain
- Optionally cleans up plaintext storage
- Updates config to use keychain

**Example usage:**
```bash
crisk migrate-to-keychain

# 🔄 Migrate to OS Keychain
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#
# Current Status:
#   API Key: sk-...abc123
#   Source: env
#   Security: ⚠️  Plaintext in shell config
#
# This will:
#   1. Store API key securely in OS keychain
#   2. Optionally remove from shell config
#
# Proceed with migration? (Y/n): y
# ✅ API key saved to OS keychain
# ✅ Removed from ~/.zshrc
#
# Your API key is now:
#   🔒 Encrypted by your OS
#   🔒 Accessible only to your user account
#   🔒 Not visible in plaintext anywhere
```

### 6. Update LLM Client to Use Config

**Update `internal/llm/client.go`:**

See [KEYCHAIN_INTEGRATION_GUIDE.md](KEYCHAIN_INTEGRATION_GUIDE.md) Section "Step 7: Update LLM Client" for complete code.

**Change `NewClient()` signature:**
```go
// OLD:
func NewClient(ctx context.Context) (*Client, error)

// NEW (with config):
func NewClient(ctx context.Context, cfg *config.Config) (*Client, error)
```

**Update to use config system:**
- Read API key from `cfg.API.OpenAIKey` (which already checked env var and keychain)
- Log the key source for debugging
- Show helpful message if key is missing

### 7. Enhanced crisk check Messaging

**Update `cmd/crisk/check.go` output:**

**Current output (no API key):**
```
⚠️  Phase 2 unavailable: OPENAI_API_KEY not set
```

**New output (tiered messaging):**
```
✅ Phase 0: Static analysis complete (0.2s)
   9 files: LOW risk, 1 file: MEDIUM risk

💡 Want deeper analysis?
   Phase 2 (LLM investigation): Set OPENAI_API_KEY
   Full mode (graph analysis): Run `docker compose up -d && crisk init-local`

   Quick setup: crisk configure
```

**With API key, no Docker:**
```
✅ Phase 0: Static analysis (0.2s)
✅ Phase 2: LLM investigation (2.1s)

💡 Want full graph-based analysis?
   Run: docker compose up -d && crisk init-local
```

**Fully configured:**
```
✅ Phase 0: Static analysis (0.2s)
✅ Phase 1: Graph analysis (1.5s)
✅ Phase 2: LLM investigation (2.3s)

📊 Full analysis complete with graph data
```

See configuration_management.md Section "First-Run Experience" for complete UX flows.

### 5. crisk doctor Command (Diagnostic)

**Add new subcommand: `cmd/crisk/doctor.go`**

Diagnoses configuration and environment:

```bash
$ crisk doctor

🩺 CodeRisk Health Check
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✅ crisk binary: v1.0.0 (latest)
✅ Go version: 1.21.3
✅ Config file: ~/.config/crisk/config.toml

Phase 0 (Static Analysis):
  ✅ Always available - no dependencies

Phase 2 (LLM Investigation):
  ✅ OPENAI_API_KEY: Set (sk-...***...abc)
  ✅ API key valid: Verified with OpenAI
  💰 API usage: $1.23 this month

Phase 1 (Graph Analysis):
  ❌ Docker: Not running
     Fix: docker compose up -d
  ❌ Neo4j: Unreachable (bolt://localhost:7687)
  ❌ PostgreSQL: Unreachable (localhost:5432)
  ❌ Redis: Unreachable (localhost:6379)

Repository Status:
  ❌ Not initialized
     Fix: crisk init-local

Overall: 2/3 phases available ✅

Recommendations:
  1. Start Docker: docker compose up -d
  2. Initialize repo: crisk init-local
  3. Or use Phase 0 + Phase 2 without Docker
```

### 6. Update install.sh

**Current install.sh is good!** It already:
- ✅ Prompts for API key interactively
- ✅ Adds to shell config (~/.zshrc, ~/.bashrc)
- ✅ Handles existing keys
- ✅ Shows Docker setup instructions

**Minor improvement:** Mention Phase 0 works immediately:

```bash
echo "✅ CodeRisk installed successfully!"
echo ""
echo "🎯 You can use it right now (no configuration needed):"
echo "   cd /path/to/your/repo"
echo "   crisk check                    # Phase 0: Static analysis (works now!)"
echo ""
echo "📚 For deeper analysis, add your API key:"
echo "   crisk configure"
echo ""
```

### 7. Update README.md

**Add tiered instructions:**

```markdown
## Quick Start

### Tier 1: Phase 0 Only (30 seconds, zero config)

**Works immediately after install!**

```bash
brew install crisk
cd your-repo
crisk check
# ✅ Phase 0 analysis complete
# 💡 Tip: Add API key for Phase 2 (deeper analysis)
```

### Tier 2: Phase 0 + Phase 2 (1 minute)

**Add API key for LLM-powered investigation:**

```bash
crisk configure
# Or manually:
export OPENAI_API_KEY="sk-..."

crisk check --explain
# ✅ Phase 0 + Phase 2 analysis
```

[Get OpenAI API key →](https://platform.openai.com/api-keys)

### Tier 3: Full Mode (10 minutes)

**Set up Docker for graph-based analysis:**

```bash
docker compose up -d
crisk init-local            # Takes 5-15 min for medium repos
crisk check --explain       # Full analysis with graph data
```

[Docker installation guide →](https://docs.docker.com/get-docker/)

---

### What You Get Per Tier

| Feature | Phase 0 | + Phase 2 | + Full |
|---------|---------|-----------|--------|
| Modification type detection | ✅ | ✅ | ✅ |
| Security pattern detection | ✅ | ✅ | ✅ |
| Risk level assignment | ✅ | ✅ | ✅ |
| LLM investigation | ❌ | ✅ | ✅ |
| Detailed explanations | ❌ | ✅ | ✅ |
| Graph analysis | ❌ | ❌ | ✅ |
| Temporal coupling | ❌ | ❌ | ✅ |
| Incident correlation | ❌ | ❌ | ✅ |

**Recommendation:** Start with Phase 0 (works now!), add API key when you want more detail.
```

### 8. Testing

**Test all configuration scenarios:**

```bash
# Clean slate
rm -rf ~/.config/crisk

# Test Phase 0 only
crisk check
# Should work, show tip about API key

# Test configure wizard
crisk configure
# Test interactive flow

# Test config commands
crisk config set api.openai_key sk-test
crisk config get api.openai_key
crisk config show

# Test doctor
crisk doctor
# Should diagnose missing Docker

# Test with API key
export OPENAI_API_KEY="sk-..."
crisk check --explain
# Should work with Phase 2

# Test with Docker
docker compose up -d
crisk init-local
crisk check --explain
# Should work with full analysis
```

---

## Success Criteria

- [ ] Config file created at `~/.config/crisk/config.toml`
- [ ] `crisk configure` wizard works interactively
- [ ] `crisk configure --key=sk-... --yes` works non-interactively
- [ ] `crisk config get/set/show/reset` all work
- [ ] `crisk check` shows tiered messaging (Phase 0 vs Phase 2 vs Full)
- [ ] `crisk doctor` diagnoses configuration issues
- [ ] README.md explains tiers clearly
- [ ] Phase 0 works with ZERO configuration
- [ ] Environment variables override config file
- [ ] Secure API key storage (don't log in plaintext)

---

## Key Files to Create/Modify

**New packages:**
- `internal/config/` (config loading, TOML parsing)

**New commands:**
- `cmd/crisk/configure.go` (interactive wizard)
- `cmd/crisk/config.go` (get/set/show/reset)
- `cmd/crisk/doctor.go` (health check)

**Modified files:**
- `cmd/crisk/check.go` (enhanced messaging)
- `cmd/crisk/main.go` (wire up new commands)
- `README.md` (tiered instructions)
- `install.sh` (mention Phase 0 works immediately)

**Config files:**
- `~/.config/crisk/config.toml` (created by configure)

---

## Design Principles

**From developer_experience.md:**

1. **Progressive Disclosure** - Show Phase 0 first, mention Phase 2/Full as optional
2. **Zero Friction** - Phase 0 works immediately, no setup required
3. **Clear Value Prop** - Explain what each tier provides
4. **Helpful Tips** - Suggest next steps without being annoying
5. **Non-Blocking** - Never error out for missing optional config

**Messaging Tone:**
- ✅ Works: "Phase 0 complete" (celebrate what's working)
- 💡 Optional: "Want deeper analysis? Add API key" (suggest upgrade)
- ⚠️ Error: "Docker not running" (only if user explicitly tried to use it)

---

## Dependencies

**Go Libraries:**
- `github.com/pelletier/go-toml/v2` - TOML parsing
- `github.com/spf13/cobra` - CLI framework (already using?)
- `github.com/fatih/color` - Colored output

**Optional:**
- `github.com/AlecAivazis/survey/v2` - Interactive prompts (nicer UX)

---

## Notes

- Config file should be .gitignore'd (contains API keys)
- API keys should be masked in `crisk config show` (show `sk-...***...abc`)
- `crisk configure` should detect existing install.sh setup (don't duplicate)
- Consider `crisk config edit` to open config file in $EDITOR
- Environment variables take precedence over config file
- `crisk doctor` should check API key validity (test OpenAI API)
- **Phase 0 is the hero** - Emphasize it works immediately!

---

## Reference Documentation

**Required Reading:**
- [configuration_management.md](configuration_management.md) - Complete strategy
- [developer_experience.md](../00-product/developer_experience.md) - UX principles

**External:**
- [go-toml documentation](https://github.com/pelletier/go-toml)
- [Cobra CLI framework](https://github.com/spf13/cobra)

---

## Questions to Ask if Unclear

1. Should config file be TOML or YAML? (Recommend TOML, simpler)
2. Should API keys be encrypted in config file? (Optional, nice-to-have)
3. Should `crisk configure` be idempotent (safe to re-run)? (Yes)
4. Should we support multiple API keys (OpenAI + Anthropic)? (Future, not now)
5. Should `crisk doctor` test actual API connectivity? (Yes, but make it fast)

---

**Good luck! This will make CodeRisk feel magical - works immediately, upgrades easily!**
