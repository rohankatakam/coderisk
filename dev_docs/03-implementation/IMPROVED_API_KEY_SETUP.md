# Improved API Key Setup Strategy

**Created:** 2025-10-13
**Status:** Recommended improvements
**Issue:** Current shell config editing approach is not developer-friendly

---

## Current State Analysis

### What We Have ✅

1. **Sophisticated config system** (`internal/config/config.go`):
   - ✅ Viper for config management (v1.18.2)
   - ✅ godotenv for .env files (v1.5.1)
   - ✅ Multi-source config loading (in order of precedence):
     - `.env.local` (highest precedence, repo-specific overrides)
     - `.env` (repo-level)
     - `.env.example` (template)
     - `~/.coderisk/.env` (user-level)
     - `~/.coderisk/config.yaml` (user-level)
     - Environment variables (OPENAI_API_KEY)
   - ✅ `Save()` method to persist config to file

2. **Current install.sh approach**:
   - ❌ Manually edits shell config files (.zshrc, .bashrc, .profile)
   - ❌ Requires shell restart
   - ❌ Not discoverable (hidden in shell config)
   - ❌ Hard to update/remove

### The Problem ⚠️

**LLM client doesn't use the config system!**

```go
// internal/llm/client.go:51
openaiKey := os.Getenv("OPENAI_API_KEY")  // ❌ Only reads from env var
```

The config system already supports reading from `.env` and `config.yaml`, but the LLM client bypasses it and reads directly from environment variables.

---

## Recommended Solution: 3-Tier Approach

### Tier 1: Quick Fix (30 minutes) - Use Config System

**Update LLM client to read from config:**

```go
// internal/llm/client.go
func NewClient(ctx context.Context, cfg *config.Config) (*Client, error) {
    logger := slog.Default().With("component", "llm")

    // Check if Phase 2 is enabled
    phase2Enabled := os.Getenv("PHASE2_ENABLED") == "true"
    if !phase2Enabled {
        logger.Info("phase 2 disabled, LLM client not initialized")
        return &Client{
            provider: ProviderNone,
            logger:   logger,
            enabled:  false,
        }, nil
    }

    // Check for OpenAI API key (from config OR environment)
    openaiKey := cfg.API.OpenAIKey
    if openaiKey == "" {
        openaiKey = os.Getenv("OPENAI_API_KEY") // Fallback to env var
    }

    if openaiKey != "" {
        client := openai.NewClient(openaiKey)
        logger.Info("openai client initialized")
        return &Client{
            provider:     ProviderOpenAI,
            openaiClient: client,
            logger:       logger,
            enabled:      true,
        }, nil
    }

    // ... rest of function
}
```

**Update install.sh to create config file:**

```bash
# Instead of editing shell config:
# OLD:
# echo "export OPENAI_API_KEY=\"$OPENAI_KEY\"" >> ~/.zshrc

# NEW: Create config file
mkdir -p ~/.coderisk
cat > ~/.coderisk/config.yaml <<EOF
api:
  openai_key: "$OPENAI_KEY"
  openai_model: "gpt-4o-mini"
EOF

echo "✅ API key saved to ~/.coderisk/config.yaml"
```

**Benefits:**
- ✅ No shell restart required
- ✅ Easy to find and edit (~/.coderisk/config.yaml)
- ✅ Works with existing config system
- ✅ Can be version controlled (per-repo .env)

---

### Tier 2: Better DX (2-3 hours) - Interactive Configuration

**Add `crisk configure` command:**

```go
// cmd/crisk/configure.go
package main

import (
    "bufio"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/coderisk/coderisk-go/internal/config"
    "github.com/spf13/cobra"
)

func newConfigureCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "configure",
        Short: "Interactive setup wizard for CodeRisk",
        Long: `Walk through CodeRisk configuration step-by-step.

This will configure:
1. OpenAI API key (required for LLM-guided risk assessment)
2. Model selection (gpt-4o-mini recommended)
3. Budget limits (optional, for cost control)
4. Storage location (default: ~/.coderisk/local.db)`,
        RunE: runConfigure,
    }
}

func runConfigure(cmd *cobra.Command, args []string) error {
    fmt.Println("🔧 CodeRisk Configuration Wizard")
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Println()

    reader := bufio.NewReader(os.Stdin)

    // Load existing config if it exists
    homeDir, _ := os.UserHomeDir()
    configPath := filepath.Join(homeDir, ".coderisk", "config.yaml")
    cfg, err := config.Load(configPath)
    if err != nil {
        cfg = config.Default()
    }

    // Step 1: OpenAI API Key
    fmt.Println("Step 1/4: OpenAI API Key")
    fmt.Println()
    if cfg.API.OpenAIKey != "" {
        fmt.Printf("Current: %s...%s\n",
            cfg.API.OpenAIKey[:7],
            cfg.API.OpenAIKey[len(cfg.API.OpenAIKey)-4:])
        fmt.Print("Keep existing key? (Y/n): ")
    } else {
        fmt.Println("CodeRisk requires an OpenAI API key for LLM-guided risk assessment.")
        fmt.Println("Get your key at: https://platform.openai.com/api-keys")
        fmt.Println("Cost: $0.03-0.05 per check (~$3-5/month for 100 checks)")
        fmt.Println()
        fmt.Print("Enter your OpenAI API key (starts with sk-...): ")
    }

    response, _ := reader.ReadString('\n')
    response = strings.TrimSpace(response)

    if cfg.API.OpenAIKey == "" || (response != "" && strings.ToLower(response) != "y") {
        if strings.HasPrefix(response, "sk-") {
            cfg.API.OpenAIKey = response
            fmt.Println("✅ API key saved")
        } else {
            fmt.Println("⚠️  Invalid API key format (should start with sk-)")
            fmt.Println("You can add it later with: crisk config set api.openai_key sk-...")
        }
    }
    fmt.Println()

    // Step 2: Model Selection
    fmt.Println("Step 2/4: LLM Model")
    fmt.Println()
    fmt.Println("Available models:")
    fmt.Println("  1. gpt-4o-mini (recommended, fast, $0.03-0.05/check)")
    fmt.Println("  2. gpt-4o (slower, higher quality, $0.15-0.20/check)")
    fmt.Printf("Current: %s\n", cfg.API.OpenAIModel)
    fmt.Print("Select model (1-2) or press Enter to keep current: ")

    response, _ = reader.ReadString('\n')
    response = strings.TrimSpace(response)

    switch response {
    case "1":
        cfg.API.OpenAIModel = "gpt-4o-mini"
        fmt.Println("✅ Using gpt-4o-mini")
    case "2":
        cfg.API.OpenAIModel = "gpt-4o"
        fmt.Println("✅ Using gpt-4o")
    case "":
        fmt.Printf("✅ Keeping %s\n", cfg.API.OpenAIModel)
    }
    fmt.Println()

    // Step 3: Budget Limits
    fmt.Println("Step 3/4: Budget Limits (Optional)")
    fmt.Println()
    fmt.Println("Set spending limits to control costs:")
    fmt.Printf("Current daily limit: $%.2f\n", cfg.Budget.DailyLimit)
    fmt.Printf("Current monthly limit: $%.2f\n", cfg.Budget.MonthlyLimit)
    fmt.Print("Change budget limits? (y/N): ")

    response, _ = reader.ReadString('\n')
    response = strings.TrimSpace(response)

    if strings.ToLower(response) == "y" {
        fmt.Print("Daily limit ($): ")
        var daily float64
        fmt.Scanln(&daily)
        if daily > 0 {
            cfg.Budget.DailyLimit = daily
        }

        fmt.Print("Monthly limit ($): ")
        var monthly float64
        fmt.Scanln(&monthly)
        if monthly > 0 {
            cfg.Budget.MonthlyLimit = monthly
        }
        fmt.Println("✅ Budget limits updated")
    } else {
        fmt.Println("✅ Keeping current limits")
    }
    fmt.Println()

    // Step 4: Save Configuration
    fmt.Println("Step 4/4: Save Configuration")
    fmt.Println()
    fmt.Printf("Save to: %s\n", configPath)
    fmt.Print("Confirm? (Y/n): ")

    response, _ = reader.ReadString('\n')
    response = strings.TrimSpace(response)

    if response == "" || strings.ToLower(response) == "y" {
        if err := cfg.Save(configPath); err != nil {
            return fmt.Errorf("failed to save config: %w", err)
        }
        fmt.Println("✅ Configuration saved!")
        fmt.Println()
        fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        fmt.Println("🎯 Next Steps:")
        fmt.Println()
        fmt.Println("1. Start infrastructure:")
        fmt.Println("   docker compose up -d")
        fmt.Println()
        fmt.Println("2. Initialize a repository:")
        fmt.Println("   cd /path/to/your/repo")
        fmt.Println("   crisk init-local")
        fmt.Println()
        fmt.Println("3. Check for risks:")
        fmt.Println("   crisk check")
        fmt.Println()
    } else {
        fmt.Println("⏭️  Configuration not saved")
    }

    return nil
}
```

**Add `crisk config` command for get/set:**

```go
// cmd/crisk/config.go
package main

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/coderisk/coderisk-go/internal/config"
    "github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "config",
        Short: "Manage CodeRisk configuration",
        Long: `Get, set, or show configuration values.

Examples:
  # Show all configuration
  crisk config show

  # Get a specific value
  crisk config get api.openai_key

  # Set a value
  crisk config set api.openai_key sk-...
  crisk config set budget.monthly_limit 50.00`,
    }

    cmd.AddCommand(newConfigShowCmd())
    cmd.AddCommand(newConfigGetCmd())
    cmd.AddCommand(newConfigSetCmd())

    return cmd
}

func newConfigShowCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "show",
        Short: "Show current configuration",
        RunE: func(cmd *cobra.Command, args []string) error {
            homeDir, _ := os.UserHomeDir()
            configPath := filepath.Join(homeDir, ".coderisk", "config.yaml")

            cfg, err := config.Load(configPath)
            if err != nil {
                return fmt.Errorf("failed to load config: %w", err)
            }

            fmt.Println("Current Configuration:")
            fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
            fmt.Printf("API Key: %s...%s\n",
                cfg.API.OpenAIKey[:7],
                cfg.API.OpenAIKey[len(cfg.API.OpenAIKey)-4:])
            fmt.Printf("Model: %s\n", cfg.API.OpenAIModel)
            fmt.Printf("Daily Limit: $%.2f\n", cfg.Budget.DailyLimit)
            fmt.Printf("Monthly Limit: $%.2f\n", cfg.Budget.MonthlyLimit)
            fmt.Printf("Storage: %s\n", cfg.Storage.LocalPath)

            return nil
        },
    }
}

func newConfigGetCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "get <key>",
        Short: "Get a configuration value",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            homeDir, _ := os.UserHomeDir()
            configPath := filepath.Join(homeDir, ".coderisk", "config.yaml")

            cfg, err := config.Load(configPath)
            if err != nil {
                return fmt.Errorf("failed to load config: %w", err)
            }

            key := args[0]

            // Simple key lookup
            switch key {
            case "api.openai_key":
                fmt.Println(cfg.API.OpenAIKey)
            case "api.openai_model":
                fmt.Println(cfg.API.OpenAIModel)
            case "budget.daily_limit":
                fmt.Printf("%.2f\n", cfg.Budget.DailyLimit)
            case "budget.monthly_limit":
                fmt.Printf("%.2f\n", cfg.Budget.MonthlyLimit)
            case "storage.local_path":
                fmt.Println(cfg.Storage.LocalPath)
            default:
                return fmt.Errorf("unknown config key: %s", key)
            }

            return nil
        },
    }
}

func newConfigSetCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "set <key> <value>",
        Short: "Set a configuration value",
        Args:  cobra.ExactArgs(2),
        RunE: func(cmd *cobra.Command, args []string) error {
            homeDir, _ := os.UserHomeDir()
            configPath := filepath.Join(homeDir, ".coderisk", "config.yaml")

            cfg, err := config.Load(configPath)
            if err != nil {
                cfg = config.Default()
            }

            key := args[0]
            value := args[1]

            // Simple key setting
            switch key {
            case "api.openai_key":
                cfg.API.OpenAIKey = value
            case "api.openai_model":
                cfg.API.OpenAIModel = value
            case "budget.daily_limit":
                var limit float64
                fmt.Sscanf(value, "%f", &limit)
                cfg.Budget.DailyLimit = limit
            case "budget.monthly_limit":
                var limit float64
                fmt.Sscanf(value, "%f", &limit)
                cfg.Budget.MonthlyLimit = limit
            case "storage.local_path":
                cfg.Storage.LocalPath = value
            default:
                return fmt.Errorf("unknown config key: %s", key)
            }

            if err := cfg.Save(configPath); err != nil {
                return fmt.Errorf("failed to save config: %w", err)
            }

            fmt.Printf("✅ Set %s = %s\n", key, value)

            return nil
        },
    }
}
```

**Update install.sh to run wizard:**

```bash
# After crisk is installed, offer to run configure
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔑 API Key Setup"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
read -p "Run configuration wizard now? (Y/n): " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
    ~/.local/bin/crisk configure
else
    echo "⏭️  Skipping configuration."
    echo ""
    echo "Run later with: crisk configure"
fi
```

**Benefits:**
- ✅ Interactive, guided setup
- ✅ No shell editing required
- ✅ Easy to reconfigure (just run `crisk configure` again)
- ✅ Simple get/set for automation
- ✅ Validates input
- ✅ Shows current values

---

### Tier 3: Best Practice (Future, 4-5 hours) - OS Keychain Integration

**Use go-keyring for secure storage:**

```bash
# Add to go.mod
go get github.com/zalando/go-keyring
```

```go
// internal/config/keyring.go
package config

import (
    "github.com/zalando/go-keyring"
)

const (
    keyringService = "CodeRisk"
    keyringUser    = "default"
)

// SaveAPIKeyToKeyring stores API key in OS keychain
func SaveAPIKeyToKeyring(apiKey string) error {
    return keyring.Set(keyringService, keyringUser, apiKey)
}

// GetAPIKeyFromKeyring retrieves API key from OS keychain
func GetAPIKeyFromKeyring() (string, error) {
    return keyring.Get(keyringService, keyringUser)
}

// DeleteAPIKeyFromKeyring removes API key from OS keychain
func DeleteAPIKeyFromKeyring() error {
    return keyring.Delete(keyringService, keyringUser)
}
```

**Update config loading to check keyring first:**

```go
// applyEnvOverrides in config.go
func applyEnvOverrides(cfg *Config) {
    // API configuration - check in order of precedence:
    // 1. Environment variable (highest, for CI/CD)
    // 2. Keyring (most secure, for local dev)
    // 3. Config file (fallback)

    if key := os.Getenv("OPENAI_API_KEY"); key != "" {
        cfg.API.OpenAIKey = key
    } else if key, err := GetAPIKeyFromKeyring(); err == nil && key != "" {
        cfg.API.OpenAIKey = key
    }

    // ... rest of function
}
```

**Update `crisk configure` to use keyring:**

```go
fmt.Print("Store API key in OS keychain (more secure)? (Y/n): ")
response, _ := reader.ReadString('\n')

if strings.TrimSpace(response) == "" || strings.ToLower(strings.TrimSpace(response)) == "y" {
    if err := config.SaveAPIKeyToKeyring(cfg.API.OpenAIKey); err != nil {
        fmt.Printf("⚠️  Failed to save to keychain: %v\n", err)
        fmt.Println("Saving to config file instead...")
    } else {
        fmt.Println("✅ API key saved to OS keychain")
        cfg.API.OpenAIKey = "" // Don't save in config file
    }
}
```

**Benefits:**
- ✅ Most secure (OS-level encryption)
- ✅ No plaintext storage
- ✅ Cross-platform (macOS Keychain, Windows Credential Manager, Linux Secret Service)
- ✅ Seamless for developers
- ❌ Complexity for CI/CD (still need env vars there)

---

## Comparison Matrix

| Approach | Setup Time | Security | Discoverability | CI/CD Friendly | Cross-Platform |
|----------|-----------|----------|-----------------|----------------|----------------|
| **Current (shell config)** | 1 min | Medium | Low | Medium | Yes |
| **Tier 1 (config file)** | 30 sec | Medium | High | High | Yes |
| **Tier 2 (wizard)** | 2 min | Medium | Very High | High | Yes |
| **Tier 3 (keyring)** | 2 min | Very High | Very High | Medium | Yes |

---

## Recommended Implementation Plan

### Phase 1: Immediate (30 minutes)
1. ✅ Update `llm.NewClient()` to accept and use config
2. ✅ Update install.sh to create `~/.coderisk/config.yaml`
3. ✅ Test that API key is read from config file

### Phase 2: Enhanced UX (2-3 hours)
4. ✅ Implement `crisk configure` command
5. ✅ Implement `crisk config` command (get/set/show)
6. ✅ Update install.sh to offer wizard
7. ✅ Add validation and error messages

### Phase 3: Security (Future, 4-5 hours)
8. ⏳ Add go-keyring dependency
9. ⏳ Implement keyring integration
10. ⏳ Update `crisk configure` to offer keyring storage
11. ⏳ Add `crisk config migrate-to-keyring` command

---

## Updated Documentation

### README.md Installation Section

**Before:**
```bash
# Current approach
export OPENAI_API_KEY="sk-..."
echo 'export OPENAI_API_KEY="sk-..."' >> ~/.zshrc
```

**After (Tier 1):**
```bash
# One-command setup
./install.sh

# Or manual setup
mkdir -p ~/.coderisk
cat > ~/.coderisk/config.yaml <<EOF
api:
  openai_key: "sk-..."
  openai_model: "gpt-4o-mini"
EOF
```

**After (Tier 2):**
```bash
# Interactive wizard
./install.sh  # Runs crisk configure automatically

# Or run wizard later
crisk configure

# Or set directly
crisk config set api.openai_key sk-...
```

### AGENT_ORCHESTRATION_GUIDE.md Updates

Update lines 591-612 to reflect implemented commands:

```markdown
### After Config Session

```bash
# 1. Test crisk configure (interactive wizard)
crisk configure
# ✅ Should walk through 4-step setup
# ✅ Should validate API key format
# ✅ Should save to ~/.coderisk/config.yaml

# 2. Test crisk config commands
crisk config show
# ✅ Should display all config values

crisk config get api.openai_key
# ✅ Should show API key

crisk config set budget.daily_limit 5.00
# ✅ Should update config file

# 3. Test config file
cat ~/.coderisk/config.yaml
# ✅ Should have proper YAML format
# ✅ Should contain API key

# 4. Test that crisk check uses config
unset OPENAI_API_KEY  # Clear env var
crisk check
# ✅ Should still work (reads from config file)
```

---

## Migration Guide for Existing Users

For users who already have `OPENAI_API_KEY` in their shell config:

```bash
# Option 1: Use crisk configure to migrate
crisk configure
# (It will detect existing env var and offer to save to config)

# Option 2: Manual migration
crisk config set api.openai_key "$OPENAI_API_KEY"

# Option 3: Keep using environment variable (still works)
# No changes needed - env var has highest precedence
```

---

## Summary

**Current Problem:**
- Manual shell config editing is not developer-friendly
- Hard to discover, update, or remove

**Recommended Solution:**
1. **Quick fix** (30 min): Use existing config system, create `~/.coderisk/config.yaml`
2. **Better UX** (2-3 hours): Add `crisk configure` and `crisk config` commands
3. **Best practice** (future): OS keychain integration with go-keyring

**Priority:**
- Implement Tier 1 (config file) immediately
- Implement Tier 2 (wizard) before launch
- Consider Tier 3 (keyring) for v1.1+

**All libraries already available:**
- ✅ Viper v1.18.2 (in go.mod)
- ✅ godotenv v1.5.1 (in go.mod)
- ⏳ go-keyring (need to add)

---

**Next Steps:**
1. Update `internal/llm/client.go` to use config
2. Update `install.sh` to create config file
3. Implement `cmd/crisk/configure.go`
4. Implement `cmd/crisk/config.go`
5. Test on fresh install
6. Update documentation
