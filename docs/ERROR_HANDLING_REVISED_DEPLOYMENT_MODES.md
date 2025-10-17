# Error Handling - Revised Deployment Modes

**Date:** October 2025
**Status:** 🎯 UPDATED STRATEGY

---

## Clarification: Deployment Modes

### What We Actually Mean

**Development Mode** (git clone):
- Developer clones the repo
- Uses `make dev` workflow
- Has `.env` file with passwords auto-configured by Docker Compose
- Only needs to set: `OPENAI_API_KEY`, `GITHUB_TOKEN`

**Production Mode** (Homebrew/GoReleaser):
- User installs via `brew install crisk`
- No `.env` file (single binary distribution)
- No Docker by default (must be started separately)
- Credentials via: environment variables, keychain, or interactive config

---

## How Popular Go CLI Tools Handle This

### 1. GitHub CLI (`gh`)
```bash
# Installation
brew install gh

# First time setup - interactive auth
gh auth login
# Stores token in keychain (macOS) or encrypted file (Linux)

# Usage
gh repo clone owner/repo
# No need to set GITHUB_TOKEN in shell
```

**Pattern:**
- Interactive first-run setup
- Keychain integration (macOS)
- Encrypted credential storage (Linux/Windows)
- Environment variable override: `GH_TOKEN`

### 2. Stripe CLI (`stripe`)
```bash
# Installation
brew install stripe/stripe-cli/stripe

# First time setup
stripe login
# Opens browser for OAuth, stores API key locally

# Or manual setup
stripe config --set api_key sk_test_...

# Usage
stripe listen --forward-to localhost:3000
```

**Pattern:**
- Interactive login (OAuth preferred)
- Manual API key config fallback
- Stored in `~/.config/stripe/config.toml`
- Environment variable override: `STRIPE_API_KEY`

### 3. Railway CLI (`railway`)
```bash
# Installation
brew install railway

# First time setup
railway login
# Opens browser for auth, stores session

# Usage
railway run npm start
```

**Pattern:**
- Browser-based OAuth
- Session stored in `~/.railway/`
- No manual token needed

### 4. Vercel CLI (`vercel`)
```bash
# Installation
npm i -g vercel

# First time setup
vercel login
# Email-based login, stores token

# Or
vercel --token TOKEN_HERE

# Config stored in ~/.vercel/
```

---

## Recommended Strategy for CodeRisk

### Deployment Modes (REVISED)

```go
type DeploymentMode string

const (
    // ModeDevelopment: git clone + make dev
    // - Uses .env file
    // - Docker Compose managed
    // - Passwords from .env are fine
    ModeDevelopment DeploymentMode = "development"

    // ModePackaged: brew install crisk
    // - Single binary, no .env
    // - Docker user-managed
    // - Interactive config or keychain
    ModePackaged DeploymentMode = "packaged"

    // ModeCI: GitHub Actions, GitLab CI, etc.
    // - All from environment variables
    // - No interactive prompts
    ModeCI DeploymentMode = "ci"
)
```

### Detection Logic

```go
func DetectMode() DeploymentMode {
    // CI environment
    if isCI() {
        return ModeCI
    }

    // Check if .env file exists (development mode)
    if _, err := os.Stat(".env"); err == nil {
        return ModeDevelopment
    }

    // Check if in git repository (development mode)
    if isGitRepo() {
        return ModeDevelopment
    }

    // Otherwise, packaged installation
    return ModePackaged
}
```

---

## Credential Management Strategy

### For Development Mode (git clone)

**What works now - KEEP IT:**
```bash
# .env file
OPENAI_API_KEY=sk-...
GITHUB_TOKEN=ghp_...
NEO4J_PASSWORD=CHANGE_THIS_PASSWORD_IN_PRODUCTION_123  # Fine for local Docker
POSTGRES_PASSWORD=CHANGE_THIS_PASSWORD_IN_PRODUCTION_123  # Fine for local Docker
```

**No changes needed!** This is perfect for development.

### For Packaged Mode (brew install)

**Priority 1: Keychain (macOS)**
```bash
# First time after brew install
crisk configure

# Interactive prompts:
Enter OpenAI API Key: sk-...
Enter GitHub Token (optional): ghp-...

# Stores in macOS Keychain
# service: "coderisk"
# account: "openai_api_key"
```

**Priority 2: Config File**
```bash
# Alternative: Manual config file
mkdir -p ~/.config/coderisk
cat > ~/.config/coderisk/config.yaml <<EOF
openai_api_key: sk-...
github_token: ghp-...
EOF
```

**Priority 3: Environment Variables (Override)**
```bash
# Always works, overrides keychain/config
export OPENAI_API_KEY=sk-...
export GITHUB_TOKEN=ghp-...
crisk check
```

### Credential Priority (Packaged Mode)

```
1. Environment Variables (highest priority)
   ↓
2. Keychain (macOS) / Secret Service (Linux)
   ↓
3. Config file (~/.config/coderisk/config.yaml)
   ↓
4. Interactive prompt (if missing)
```

---

## Database Passwords (Neo4j, Postgres)

### Development Mode
- Use `.env` defaults
- Docker Compose manages services
- No user interaction needed

### Packaged Mode
- User starts Docker themselves: `docker compose up -d`
- Passwords auto-generated on first run OR
- User provides their own via environment variables

**Smart approach:**
```bash
# When user runs: crisk init-local
# 1. Check if Docker is running
# 2. Check if Neo4j/Postgres containers exist
# 3. If not, offer to start them:

Would you like crisk to start the database services? (Y/n)
> y

Starting Neo4j and PostgreSQL with Docker Compose...
Generated secure passwords (stored in ~/.config/coderisk/docker.env)

✓ Neo4j running on bolt://localhost:7687
✓ PostgreSQL running on localhost:5432
```

---

## Implementation Plan (REVISED)

### Phase 1: Mode Detection (1 hour)

Create `internal/config/mode.go`:

```go
package config

import (
    "os"
    "path/filepath"
)

type DeploymentMode string

const (
    ModeDevelopment DeploymentMode = "development"
    ModePackaged    DeploymentMode = "packaged"
    ModeCI          DeploymentMode = "ci"
)

func DetectMode() DeploymentMode {
    // Explicit override
    if mode := os.Getenv("CRISK_MODE"); mode != "" {
        return DeploymentMode(mode)
    }

    // CI detection
    if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
        return ModeCI
    }

    // Development mode: .env file exists
    if _, err := os.Stat(".env"); err == nil {
        return ModeDevelopment
    }

    // Development mode: inside git repo
    if _, err := os.Stat(".git"); err == nil {
        return ModeDevelopment
    }

    // Check if running from source (go.mod exists)
    if _, err := os.Stat("go.mod"); err == nil {
        return ModeDevelopment
    }

    // Otherwise: packaged installation
    return ModePackaged
}

func (m DeploymentMode) IsDevelopment() bool {
    return m == ModeDevelopment
}

func (m DeploymentMode) IsPackaged() bool {
    return m == ModePackaged
}

func (m DeploymentMode) IsCI() bool {
    return m == ModeCI
}
```

### Phase 2: Keychain Integration (2 hours)

The existing `internal/config/keyring.go` already handles this! Just need to enhance it:

```go
// internal/config/credentials.go
package config

type CredentialManager struct {
    mode     DeploymentMode
    keyring  *KeyringManager
    configFile string
}

func NewCredentialManager() *CredentialManager {
    mode := DetectMode()
    return &CredentialManager{
        mode:       mode,
        keyring:    NewKeyringManager(),
        configFile: filepath.Join(os.UserHomeDir(), ".config", "coderisk", "config.yaml"),
    }
}

func (cm *CredentialManager) GetAPIKey() (string, error) {
    // 1. Environment variable (highest priority)
    if key := os.Getenv("OPENAI_API_KEY"); key != "" {
        return key, nil
    }

    // 2. Keychain (if available)
    if cm.keyring.IsAvailable() {
        if key, err := cm.keyring.GetAPIKey(); err == nil && key != "" {
            return key, nil
        }
    }

    // 3. Config file
    if key := cm.readFromConfigFile("openai_api_key"); key != "" {
        return key, nil
    }

    // 4. Interactive prompt (only in packaged mode)
    if cm.mode.IsPackaged() && isInteractive() {
        return cm.promptForAPIKey()
    }

    return "", errors.ConfigError("OPENAI_API_KEY not found. Run 'crisk configure' to set it up.")
}
```

### Phase 3: Interactive Configuration (1 hour)

```bash
# New command: crisk configure
$ crisk configure

CodeRisk Configuration
======================

OpenAI API Key (required for Phase 2 analysis):
  Create at: https://platform.openai.com/api-keys
  Enter key (or press Enter to skip): sk-...
  ✓ Saved to keychain

GitHub Token (optional, for fetching private repos):
  Create at: https://github.com/settings/tokens
  Enter token (or press Enter to skip): ghp-...
  ✓ Saved to keychain

Configuration complete!
Run 'crisk init-local' to set up your first repository.
```

### Phase 4: Database Auto-Setup (2 hours)

Smart Docker detection and setup:

```go
// cmd/crisk/init_local.go
func runInitLocal(cmd *cobra.Command, args []string) error {
    // Check if Docker is available
    if !dockerAvailable() {
        logging.Fatal("Docker is required but not running. Please start Docker Desktop.")
    }

    // Check if containers exist
    if !containersRunning() {
        if isInteractive() {
            fmt.Println("Database services are not running.")
            fmt.Print("Start them now? (Y/n): ")

            var response string
            fmt.Scanln(&response)

            if response == "" || strings.ToLower(response) == "y" {
                if err := startDatabases(); err != nil {
                    logging.Fatal("failed to start databases", "error", err)
                }
                fmt.Println("✓ Databases started")
            } else {
                logging.Fatal("Databases required. Run: docker compose up -d")
            }
        } else {
            // Non-interactive: try to start automatically
            logging.Info("starting database services")
            if err := startDatabases(); err != nil {
                logging.Fatal("failed to start databases", "error", err)
            }
        }
    }

    // Continue with init...
}
```

---

## Revised Error Handling Strategy

### Development Mode
- Allow `.env` defaults for passwords ✅
- Require `OPENAI_API_KEY` and `GITHUB_TOKEN` explicitly set
- No warnings about "insecure" passwords for local Docker

### Packaged Mode
- Use keychain/config file for API keys
- Auto-generate Docker passwords if needed
- Interactive prompts for missing credentials
- Environment variable overrides always work

### CI Mode
- Everything from environment variables
- No interactive prompts
- Fail fast if credentials missing

---

## What Changes vs Original Plan

### KEEP (Good as-is):
- ✅ Centralized logging infrastructure
- ✅ Error handling framework
- ✅ Configuration validation (but update logic)
- ✅ Fix ignored errors
- ✅ Replace fallback patterns
- ✅ Dynamic metadata extraction

### CHANGE:
- ❌ Don't distinguish "production" as "cloud"
- ✅ Use: Development (git clone) vs Packaged (brew install) vs CI
- ✅ Focus on keychain integration for packaged mode
- ✅ Keep `.env` defaults for development mode
- ✅ Add `crisk configure` command for easy setup

### NEW:
- ✅ Credential manager with priority chain
- ✅ Interactive configuration wizard
- ✅ Smart Docker auto-start
- ✅ Config file support (~/.config/crisk/)

---

## Next Steps

1. **Update mode.go** - Use Development/Packaged/CI distinction
2. **Enhance credential manager** - Priority chain (env → keychain → config → prompt)
3. **Add `crisk configure` command** - Interactive setup
4. **Smart Docker detection** - Auto-start with user consent
5. **Keep logging/error handling work** - This is still critical

---

## Files to Create/Modify

**New:**
- `internal/config/mode.go` (revised)
- `internal/config/credentials.go` (credential manager)
- `cmd/crisk/configure.go` (interactive setup)
- `internal/docker/manager.go` (smart Docker handling)

**Modify:**
- `internal/config/validator.go` (use new modes)
- `cmd/crisk/init_local.go` (smart Docker setup)
- All commands (use credential manager)

---

**This aligns with your packaging workflow and follows Go CLI best practices!**
