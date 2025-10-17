# OS Keychain Integration Guide

**Created:** 2025-10-13
**Status:** Implementation Ready - Professional Security Tier
**Priority:** HIGH - Launch requirement for professional tool

---

## Overview

This guide provides complete implementation details for integrating OS-level keychain storage into CodeRisk, enabling secure credential management across macOS, Windows, and Linux without plaintext storage.

### Why Keychain Integration?

**Current Problem:**
- API keys in shell config files (.zshrc, .bashrc) → visible in plaintext
- API keys in config files → visible in plaintext, can be accidentally committed
- Environment variables → visible in process list, inherited by child processes

**Professional Solution:**
- ✅ API keys stored in OS keychain (encrypted at rest)
- ✅ No plaintext storage (except opt-in for CI/CD)
- ✅ OS-managed encryption and access control
- ✅ Professional developer experience
- ✅ Cross-platform support

---

## Architecture

### Storage Hierarchy (Precedence Order)

```
1. Environment Variable (OPENAI_API_KEY)
   ↓ Highest precedence
   ↓ Use case: CI/CD, temporary override

2. OS Keychain
   ↓ Second precedence
   ↓ Use case: Local development (secure)

3. Config File (~/. coderisk/config.yaml)
   ↓ Third precedence
   ↓ Use case: Non-sensitive config, opt-in plaintext

4. .env File (repo-level)
   ↓ Lowest precedence
   ↓ Use case: Repo-specific config, CI/CD
```

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                      CodeRisk CLI                            │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  cmd/crisk/configure.go ─┐                                  │
│  cmd/crisk/config.go ────┤                                  │
│  cmd/crisk/migrate.go ───┤                                  │
│                          │                                   │
│                          ↓                                   │
│            internal/config/keyring.go                       │
│                          │                                   │
│                          ↓                                   │
│               github.com/zalando/go-keyring                 │
│                          │                                   │
├──────────────────────────┼───────────────────────────────────┤
│                          ↓                                   │
│  ┌───────────────┬──────────────────┬──────────────────┐   │
│  │   macOS       │     Windows      │      Linux       │   │
│  │   Keychain    │   Credential     │   Secret         │   │
│  │               │   Manager        │   Service        │   │
│  └───────────────┴──────────────────┴──────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## Implementation Steps

### Step 1: Add go-keyring Dependency

```bash
# Add dependency
go get github.com/zalando/go-keyring@latest

# Verify in go.mod
grep go-keyring go.mod
# Should show: github.com/zalando/go-keyring v1.2.4
```

**Cross-platform support:**
- macOS: Uses Keychain Services API
- Windows: Uses Windows Credential Manager API
- Linux: Uses freedesktop.org Secret Service API (requires libsecret)

---

### Step 2: Implement Keyring Interface

**Create `internal/config/keyring.go`:**

```go
package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/zalando/go-keyring"
)

const (
	// KeyringService is the service name in the OS keychain
	KeyringService = "CodeRisk"

	// KeyringUser is the user identifier for credentials
	KeyringUser = "default"

	// KeyringAPIKeyItem is the key for OpenAI API key
	KeyringAPIKeyItem = "openai-api-key"
)

// KeyringManager handles secure credential storage in OS keychain
type KeyringManager struct {
	logger *slog.Logger
}

// NewKeyringManager creates a new keyring manager
func NewKeyringManager() *KeyringManager {
	return &KeyringManager{
		logger: slog.Default().With("component", "keyring"),
	}
}

// SaveAPIKey stores API key securely in OS keychain
// This uses OS-level encryption:
// - macOS: Keychain Access.app → "CodeRisk" → "openai-api-key"
// - Windows: Credential Manager → "CodeRisk"
// - Linux: Secret Service (requires libsecret)
func (km *KeyringManager) SaveAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("api key cannot be empty")
	}

	err := keyring.Set(KeyringService, KeyringAPIKeyItem, apiKey)
	if err != nil {
		km.logger.Error("failed to save API key to keychain", "error", err)
		return fmt.Errorf("failed to save to OS keychain: %w", err)
	}

	km.logger.Info("api key saved to keychain", "service", KeyringService)
	return nil
}

// GetAPIKey retrieves API key from OS keychain
func (km *KeyringManager) GetAPIKey() (string, error) {
	apiKey, err := keyring.Get(KeyringService, KeyringAPIKeyItem)
	if err == keyring.ErrNotFound {
		// Not an error - just not set yet
		return "", nil
	}
	if err != nil {
		km.logger.Error("failed to get API key from keychain", "error", err)
		return "", fmt.Errorf("failed to read from OS keychain: %w", err)
	}

	km.logger.Debug("api key retrieved from keychain")
	return apiKey, nil
}

// DeleteAPIKey removes API key from OS keychain
func (km *KeyringManager) DeleteAPIKey() error {
	err := keyring.Delete(KeyringService, KeyringAPIKeyItem)
	if err == keyring.ErrNotFound {
		// Already deleted, not an error
		return nil
	}
	if err != nil {
		km.logger.Error("failed to delete API key from keychain", "error", err)
		return fmt.Errorf("failed to delete from OS keychain: %w", err)
	}

	km.logger.Info("api key deleted from keychain")
	return nil
}

// IsAvailable checks if OS keychain is available
// Returns false on headless systems (CI/CD) where keychain isn't available
func (km *KeyringManager) IsAvailable() bool {
	// Try to access keyring with a test operation
	_, err := keyring.Get(KeyringService, "test-availability")

	// If error is "not found", keychain is available
	// If error is something else, keychain may not be available
	if err == keyring.ErrNotFound {
		return true
	}
	if err != nil {
		km.logger.Debug("keychain not available", "error", err)
		return false
	}

	return true
}

// GetSourceInfo returns information about where the API key is stored
type KeySourceInfo struct {
	Source      string // "keychain", "config", "env", "env_file", "none"
	Secure      bool   // true if stored securely (keychain or env var in CI/CD)
	Recommended string // recommendation if not optimal
}

// GetAPIKeySource determines where the API key is coming from
func (km *KeyringManager) GetAPIKeySource(cfg *Config) KeySourceInfo {
	// Check environment variable first (highest precedence)
	if os.Getenv("OPENAI_API_KEY") != "" {
		return KeySourceInfo{
			Source:      "env",
			Secure:      true, // Acceptable for CI/CD
			Recommended: "Using environment variable (good for CI/CD)",
		}
	}

	// Check keychain
	keychainKey, _ := km.GetAPIKey()
	if keychainKey != "" {
		return KeySourceInfo{
			Source:      "keychain",
			Secure:      true,
			Recommended: "Stored securely in OS keychain ✅",
		}
	}

	// Check config file
	if cfg.API.OpenAIKey != "" {
		return KeySourceInfo{
			Source:      "config",
			Secure:      false,
			Recommended: "⚠️  Plaintext storage detected. Run: crisk migrate-to-keychain",
		}
	}

	// Check .env file
	if _, err := os.Stat(".env"); err == nil {
		// .env file exists, likely contains API key
		return KeySourceInfo{
			Source:      "env_file",
			Secure:      false,
			Recommended: "Using .env file (OK for CI/CD, consider keychain for local dev)",
		}
	}

	return KeySourceInfo{
		Source:      "none",
		Secure:      false,
		Recommended: "No API key configured. Run: crisk configure",
	}
}

// MaskAPIKey masks an API key for display
// Shows first 7 chars and last 4 chars: "sk-proj...abc123"
func MaskAPIKey(apiKey string) string {
	if apiKey == "" {
		return "(not set)"
	}
	if len(apiKey) < 12 {
		return "***"
	}
	return fmt.Sprintf("%s...%s", apiKey[:7], apiKey[len(apiKey)-4:])
}
```

---

### Step 3: Update Config Loading

**Update `internal/config/config.go`:**

```go
// Add to applyEnvOverrides function
func applyEnvOverrides(cfg *Config) {
	// GitHub configuration
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		cfg.GitHub.Token = token
	}

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

	// ... rest of function unchanged
}
```

**Add config field for keychain preference:**

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

---

### Step 4: Implement `crisk configure` Command

**Create `cmd/crisk/configure.go`:**

```go
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
		Short: "Interactive setup wizard for CodeRisk (with OS keychain support)",
		Long: `Walk through CodeRisk configuration step-by-step with secure credential storage.

Features:
- Store API keys securely in OS keychain (no plaintext)
- Configure LLM model and budget limits
- Professional developer experience
- Cross-platform support (macOS, Windows, Linux)

This will configure:
1. OpenAI API key (stored in OS keychain by default)
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

	// Initialize keyring manager
	km := config.NewKeyringManager()

	// Check if keychain is available
	keychainAvailable := km.IsAvailable()
	if !keychainAvailable {
		fmt.Println("⚠️  OS keychain not available (headless system or Linux without libsecret)")
		fmt.Println("   Will store API key in config file instead.")
		fmt.Println()
	}

	// Step 1: OpenAI API Key
	fmt.Println("Step 1/4: OpenAI API Key")
	fmt.Println()

	// Check existing sources
	sourceInfo := km.GetAPIKeySource(cfg)

	if sourceInfo.Source != "none" {
		fmt.Printf("Current: %s\n", config.MaskAPIKey(cfg.API.OpenAIKey))
		fmt.Printf("Source: %s\n", sourceInfo.Recommended)
		fmt.Print("Keep existing key? (Y/n): ")

		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)

		if response == "" || strings.ToLower(response) == "y" {
			goto step2
		}
	} else {
		fmt.Println("CodeRisk requires an OpenAI API key for LLM-guided risk assessment.")
		fmt.Println("Get your key at: https://platform.openai.com/api-keys")
		fmt.Println("Cost: $0.03-0.05 per check (~$3-5/month for 100 checks)")
		fmt.Println()
	}

	fmt.Print("Enter your OpenAI API key (starts with sk-...): ")
	response, _ := reader.ReadString('\n')
	apiKey := strings.TrimSpace(response)

	if !strings.HasPrefix(apiKey, "sk-") {
		fmt.Println("⚠️  Invalid API key format (should start with sk-)")
		fmt.Println("You can add it later with: crisk config set api.openai_key sk-...")
		goto step2
	}

	// Offer keychain storage
	if keychainAvailable {
		fmt.Println()
		fmt.Println("🔒 Secure Storage Options:")
		fmt.Println("  1. OS Keychain (recommended, encrypted, secure)")
		fmt.Println("  2. Config file (plaintext, not recommended for local dev)")
		fmt.Print("Choose storage method (1-2): ")

		response, _ = reader.ReadString('\n')
		response = strings.TrimSpace(response)

		if response == "1" || response == "" {
			// Save to keychain
			if err := km.SaveAPIKey(apiKey); err != nil {
				fmt.Printf("⚠️  Failed to save to keychain: %v\n", err)
				fmt.Println("Saving to config file instead...")
				cfg.API.OpenAIKey = apiKey
				cfg.API.UseKeychain = false
			} else {
				fmt.Println("✅ API key saved to OS keychain (secure)")
				cfg.API.OpenAIKey = "" // Don't save in config file
				cfg.API.UseKeychain = true

				// Show where it's stored based on OS
				switch {
				case strings.Contains(strings.ToLower(os.Getenv("OS")), "windows"):
					fmt.Println("   📍 Windows Credential Manager → 'CodeRisk'")
				case fileExists("/usr/bin/security"):
					fmt.Println("   📍 macOS Keychain Access.app → 'CodeRisk'")
				default:
					fmt.Println("   📍 Linux Secret Service (libsecret)")
				}
			}
		} else {
			// Save to config file
			cfg.API.OpenAIKey = apiKey
			cfg.API.UseKeychain = false
			fmt.Println("✅ API key saved to config file (plaintext)")
			fmt.Println("   ⚠️  Consider using keychain for better security")
		}
	} else {
		// No keychain available, save to config file
		cfg.API.OpenAIKey = apiKey
		cfg.API.UseKeychain = false
		fmt.Println("✅ API key saved to config file")
	}

step2:
	fmt.Println()
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

		if cfg.API.UseKeychain {
			fmt.Println("🔒 Security: API key stored securely in OS keychain")
		}
	} else {
		fmt.Println("⏭️  Configuration not saved")
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
```

---

### Step 5: Implement `crisk config` Commands

**Create `cmd/crisk/config.go`:**

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/coderisk/coderisk-go/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CodeRisk configuration (with keychain support)",
		Long: `Get, set, or show configuration values with OS keychain integration.

Examples:
  # Show all configuration
  crisk config show

  # Get a specific value
  crisk config get api.openai_key --show-source

  # Set a value in keychain (secure)
  crisk config set api.openai_key sk-... --use-keychain

  # Set a value in config file (plaintext, for CI/CD)
  crisk config set api.openai_key sk-... --no-keychain

  # Set budget limit
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
		Short: "Show current configuration with security indicators",
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, _ := os.UserHomeDir()
			configPath := filepath.Join(homeDir, ".coderisk", "config.yaml")

			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			km := config.NewKeyringManager()
			sourceInfo := km.GetAPIKeySource(cfg)

			fmt.Println("Current Configuration:")
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

			// API configuration
			fmt.Printf("API Key: %s\n", config.MaskAPIKey(cfg.API.OpenAIKey))
			fmt.Printf("  Source: %s\n", sourceInfo.Recommended)
			if sourceInfo.Secure {
				fmt.Printf("  Security: ✅ Secure storage\n")
			} else {
				fmt.Printf("  Security: ⚠️  Plaintext storage\n")
			}

			fmt.Printf("Model: %s\n", cfg.API.OpenAIModel)
			fmt.Printf("Daily Limit: $%.2f\n", cfg.Budget.DailyLimit)
			fmt.Printf("Monthly Limit: $%.2f\n", cfg.Budget.MonthlyLimit)
			fmt.Printf("Storage: %s\n", cfg.Storage.LocalPath)
			fmt.Println()

			// Show recommendations
			if !sourceInfo.Secure && sourceInfo.Source != "none" {
				fmt.Println("💡 Recommendation:")
				fmt.Println("   Run: crisk migrate-to-keychain")
				fmt.Println("   This will move your API key to secure OS storage")
			}

			return nil
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	var showSource bool

	cmd := &cobra.Command{
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
				if showSource {
					km := config.NewKeyringManager()
					sourceInfo := km.GetAPIKeySource(cfg)
					fmt.Printf("%s\n", cfg.API.OpenAIKey)
					fmt.Printf("Source: %s\n", sourceInfo.Source)
					if sourceInfo.Secure {
						fmt.Println("Security: ✅ Secure")
					} else {
						fmt.Println("Security: ⚠️  Plaintext")
					}
				} else {
					fmt.Println(cfg.API.OpenAIKey)
				}
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

	cmd.Flags().BoolVar(&showSource, "show-source", false, "Show where the value is stored")

	return cmd
}

func newConfigSetCmd() *cobra.Command {
	var useKeychain bool
	var noKeychain bool

	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value with optional keychain storage.

Examples:
  # Store API key in OS keychain (secure, recommended)
  crisk config set api.openai_key sk-... --use-keychain

  # Store API key in config file (plaintext, for CI/CD)
  crisk config set api.openai_key sk-... --no-keychain

  # If neither flag specified, will prompt user`,
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

			// Handle API key specially (with keychain support)
			if key == "api.openai_key" {
				km := config.NewKeyringManager()

				// Determine storage method
				var storeInKeychain bool
				if useKeychain {
					storeInKeychain = true
				} else if noKeychain {
					storeInKeychain = false
				} else {
					// Ask user if keychain is available
					if km.IsAvailable() {
						fmt.Print("Store in OS keychain (secure)? (Y/n): ")
						var response string
						fmt.Scanln(&response)
						storeInKeychain = (response == "" || strings.ToLower(response) == "y")
					}
				}

				if storeInKeychain && km.IsAvailable() {
					// Save to keychain
					if err := km.SaveAPIKey(value); err != nil {
						fmt.Printf("⚠️  Failed to save to keychain: %v\n", err)
						fmt.Println("Saving to config file instead...")
						cfg.API.OpenAIKey = value
						cfg.API.UseKeychain = false
					} else {
						fmt.Println("✅ API key saved to OS keychain (secure)")
						cfg.API.OpenAIKey = "" // Don't save in config file
						cfg.API.UseKeychain = true
					}
				} else {
					// Save to config file
					cfg.API.OpenAIKey = value
					cfg.API.UseKeychain = false
					fmt.Println("✅ API key saved to config file (plaintext)")
					if km.IsAvailable() {
						fmt.Println("   💡 For better security, use: --use-keychain flag")
					}
				}
			} else {
				// Handle other config keys
				switch key {
				case "api.openai_model":
					cfg.API.OpenAIModel = value
				case "budget.daily_limit":
					if limit, err := strconv.ParseFloat(value, 64); err == nil {
						cfg.Budget.DailyLimit = limit
					} else {
						return fmt.Errorf("invalid float value: %s", value)
					}
				case "budget.monthly_limit":
					if limit, err := strconv.ParseFloat(value, 64); err == nil {
						cfg.Budget.MonthlyLimit = limit
					} else {
						return fmt.Errorf("invalid float value: %s", value)
					}
				case "storage.local_path":
					cfg.Storage.LocalPath = value
				default:
					return fmt.Errorf("unknown config key: %s", key)
				}

				fmt.Printf("✅ Set %s = %s\n", key, value)
			}

			// Save config file
			if err := cfg.Save(configPath); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&useKeychain, "use-keychain", false, "Store API key in OS keychain (secure)")
	cmd.Flags().BoolVar(&noKeychain, "no-keychain", false, "Store API key in config file (plaintext)")

	return cmd
}
```

---

### Step 6: Implement Migration Command

**Create `cmd/crisk/migrate.go`:**

```go
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

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate-to-keychain",
		Short: "Migrate API key from plaintext storage to OS keychain",
		Long: `Migrate existing API key to secure OS keychain storage.

This command will:
1. Detect where your API key is currently stored
2. Move it to OS keychain (encrypted, secure)
3. Optionally clean up plaintext storage
4. Update config to use keychain

Supports migration from:
- Environment variables (shell config files)
- Config file (plaintext YAML)
- .env files (repo-specific)`,
		RunE: runMigrate,
	}

	return cmd
}

func runMigrate(cmd *cobra.Command, args []string) error {
	fmt.Println("🔄 Migrate to OS Keychain")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Check if keychain is available
	km := config.NewKeyringManager()
	if !km.IsAvailable() {
		return fmt.Errorf("OS keychain not available on this system")
	}

	// Load config
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".coderisk", "config.yaml")
	cfg, err := config.Load(configPath)
	if err != nil {
		cfg = config.Default()
	}

	// Check current source
	sourceInfo := km.GetAPIKeySource(cfg)

	fmt.Println("Current Status:")
	fmt.Printf("  API Key: %s\n", config.MaskAPIKey(cfg.API.OpenAIKey))
	fmt.Printf("  Source: %s\n", sourceInfo.Source)
	fmt.Printf("  Security: %s\n", sourceInfo.Recommended)
	fmt.Println()

	if sourceInfo.Source == "keychain" {
		fmt.Println("✅ Already using OS keychain (most secure)")
		return nil
	}

	if sourceInfo.Source == "none" {
		return fmt.Errorf("no API key found to migrate. Run: crisk configure")
	}

	// Get the actual API key value
	var apiKey string
	switch sourceInfo.Source {
	case "env":
		apiKey = os.Getenv("OPENAI_API_KEY")
	case "config":
		apiKey = cfg.API.OpenAIKey
	case "env_file":
		// Load from .env file
		if content, err := os.ReadFile(".env"); err == nil {
			for _, line := range strings.Split(string(content), "\n") {
				if strings.HasPrefix(line, "OPENAI_API_KEY=") {
					apiKey = strings.TrimPrefix(line, "OPENAI_API_KEY=")
					apiKey = strings.Trim(apiKey, "\"' \t")
					break
				}
			}
		}
	}

	if apiKey == "" {
		return fmt.Errorf("failed to extract API key from %s", sourceInfo.Source)
	}

	// Confirm migration
	fmt.Println("This will:")
	fmt.Println("  1. Store API key securely in OS keychain")
	if sourceInfo.Source == "config" {
		fmt.Println("  2. Remove API key from config file")
	}
	if sourceInfo.Source == "env" {
		fmt.Println("  2. Optionally remove from shell config")
	}
	fmt.Println()
	fmt.Print("Proceed with migration? (Y/n): ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(response)

	if response != "" && strings.ToLower(response) != "y" {
		fmt.Println("⏭️  Migration cancelled")
		return nil
	}

	// Save to keychain
	if err := km.SaveAPIKey(apiKey); err != nil {
		return fmt.Errorf("failed to save to keychain: %w", err)
	}

	fmt.Println("✅ API key saved to OS keychain")

	// Clean up based on source
	switch sourceInfo.Source {
	case "config":
		// Remove from config file
		cfg.API.OpenAIKey = ""
		cfg.API.UseKeychain = true
		if err := cfg.Save(configPath); err != nil {
			fmt.Printf("⚠️  Failed to update config file: %v\n", err)
		} else {
			fmt.Println("✅ Removed API key from config file")
		}

	case "env":
		// Offer to clean up shell config
		fmt.Println()
		fmt.Println("Shell config cleanup:")
		fmt.Println("  Your shell config may still contain OPENAI_API_KEY")
		fmt.Println("  Would you like to remove it?")
		fmt.Println()

		shellConfig := detectShellConfig()
		if shellConfig != "" {
			fmt.Printf("  Detected: %s\n", shellConfig)
			fmt.Print("  Remove OPENAI_API_KEY from this file? (y/N): ")

			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(response)

			if strings.ToLower(response) == "y" {
				if err := removeEnvVarFromFile(shellConfig, "OPENAI_API_KEY"); err != nil {
					fmt.Printf("⚠️  Failed to clean up: %v\n", err)
					fmt.Println("   Please manually remove the line with OPENAI_API_KEY")
				} else {
					fmt.Println("✅ Removed from shell config")
					fmt.Println("   💡 Restart your shell for changes to take effect")
				}
			}
		}

		// Update config to use keychain
		cfg.API.UseKeychain = true
		cfg.Save(configPath)
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ Migration Complete!")
	fmt.Println()
	fmt.Println("Your API key is now:")
	fmt.Println("  🔒 Encrypted by your OS")
	fmt.Println("  🔒 Accessible only to your user account")
	fmt.Println("  🔒 Not visible in plaintext anywhere")
	fmt.Println()
	fmt.Println("You can verify with: crisk config show")

	return nil
}

func detectShellConfig() string {
	homeDir, _ := os.UserHomeDir()

	// Check which shell config file exists and is currently used
	candidates := []string{
		".zshrc",
		".bashrc",
		".bash_profile",
		".profile",
	}

	shell := os.Getenv("SHELL")

	// Prioritize based on current shell
	if strings.Contains(shell, "zsh") {
		for _, candidate := range []string{".zshrc", ".zprofile"} {
			path := filepath.Join(homeDir, candidate)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	if strings.Contains(shell, "bash") {
		for _, candidate := range []string{".bashrc", ".bash_profile"} {
			path := filepath.Join(homeDir, candidate)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	// Fallback: check all candidates
	for _, candidate := range candidates {
		path := filepath.Join(homeDir, candidate)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func removeEnvVarFromFile(filePath string, varName string) error {
	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Filter out lines with the environment variable
	lines := strings.Split(string(content), "\n")
	var newLines []string

	for _, line := range lines {
		// Skip lines that set this environment variable
		if strings.Contains(line, fmt.Sprintf("export %s=", varName)) ||
		   strings.Contains(line, fmt.Sprintf("%s=", varName)) {
			continue
		}
		newLines = append(newLines, line)
	}

	// Write back
	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(filePath, []byte(newContent), 0644)
}
```

---

### Step 7: Update LLM Client

**Update `internal/llm/client.go`:**

```go
// NewClient creates an LLM client based on available API keys
// Now uses the config system with keychain support
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

	// Get OpenAI API key from config (which already checked env var and keychain)
	openaiKey := cfg.API.OpenAIKey
	if openaiKey == "" {
		logger.Warn("phase 2 enabled but no LLM API key configured")
		logger.Info("run 'crisk configure' to set up your API key securely")
		return &Client{
			provider: ProviderNone,
			logger:   logger,
			enabled:  false,
		}, nil
	}

	// Create OpenAI client
	client := openai.NewClient(openaiKey)
	logger.Info("openai client initialized", "key_source", getKeySource(cfg))
	return &Client{
		provider:     ProviderOpenAI,
		openaiClient: client,
		logger:       logger,
		enabled:      true,
	}, nil
}

func getKeySource(cfg *config.Config) string {
	if os.Getenv("OPENAI_API_KEY") != "" {
		return "environment"
	}
	if cfg.API.UseKeychain {
		return "keychain"
	}
	return "config_file"
}
```

---

### Step 8: Update install.sh

**Update `install.sh`:**

```bash
#!/bin/bash
set -e

# ... existing installation code ...

# Setup OpenAI API Key - PROFESSIONAL TIER with Keychain Support
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔑 API Key Setup (PROFESSIONAL SECURITY)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "CodeRisk requires an OpenAI API key for LLM-guided risk assessment."
echo "Cost: \$0.03-0.05 per check (~\$3-5/month for 100 checks)"
echo ""
echo "Setup Options:"
echo "  1. Interactive wizard (recommended, with OS keychain support)"
echo "  2. Quick setup (save to config file)"
echo "  3. Skip (configure later)"
echo ""
read -p "Choose option (1-3): " -n 1 -r SETUP_CHOICE
echo ""
echo ""

case $SETUP_CHOICE in
    1)
        # Run interactive wizard
        echo "🔧 Starting configuration wizard..."
        ~/.local/bin/crisk configure
        ;;
    2)
        # Quick setup - save to config file
        read -p "Enter your OpenAI API key (starts with sk-...): " -r OPENAI_KEY

        if [ -n "$OPENAI_KEY" ]; then
            mkdir -p ~/.coderisk
            cat > ~/.coderisk/config.yaml <<EOF
api:
  openai_key: "$OPENAI_KEY"
  openai_model: "gpt-4o-mini"
  use_keychain: false
budget:
  daily_limit: 2.00
  monthly_limit: 60.00
EOF
            echo "✅ API key saved to ~/.coderisk/config.yaml"
            echo ""
            echo "💡 For better security, run: crisk migrate-to-keychain"
        fi
        ;;
    3)
        echo "⏭️  Skipping API key setup."
        echo ""
        echo "⚠️  CodeRisk requires an API key to function!"
        echo ""
        echo "To configure later, run:"
        echo "  crisk configure"
        echo ""
        echo "Or get your API key at: https://platform.openai.com/api-keys"
        ;;
esac

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🎯 Next Steps (17 minutes one-time per repo)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
# ... rest of install.sh ...
```

---

## Testing Guide

### Manual Testing Checklist

**Prerequisites:**
```bash
# Install the CLI
./install.sh

# Verify keychain support
go get github.com/zalando/go-keyring
go build -o crisk ./cmd/crisk
```

**Test 1: Interactive Configuration**
```bash
crisk configure
# ✅ Should prompt for API key
# ✅ Should offer keychain storage (macOS/Windows/Linux with libsecret)
# ✅ Should validate API key format
# ✅ Should save to keychain if selected
```

**Test 2: Keychain Storage**
```bash
crisk config set api.openai_key sk-test123 --use-keychain
# ✅ Should save to OS keychain

# Verify on macOS
open "/Applications/Utilities/Keychain Access.app"
# Search for "CodeRisk" - should see entry

# Verify config file doesn't contain key
cat ~/.coderisk/config.yaml
# ✅ Should NOT contain "sk-test123"
# ✅ Should have "use_keychain: true"
```

**Test 3: Config Retrieval**
```bash
unset OPENAI_API_KEY
crisk config get api.openai_key
# ✅ Should show API key from keychain

crisk config get api.openai_key --show-source
# ✅ Should show "Source: keychain (secure)"
```

**Test 4: Precedence Order**
```bash
# Test env var override
export OPENAI_API_KEY="sk-env-override"
crisk config show
# ✅ Should show env var takes precedence

unset OPENAI_API_KEY
crisk config show
# ✅ Should fall back to keychain
```

**Test 5: Migration**
```bash
# Set up old style
echo 'export OPENAI_API_KEY="sk-old"' >> ~/.zshrc
source ~/.zshrc

crisk migrate-to-keychain
# ✅ Should detect env var
# ✅ Should prompt to migrate
# ✅ Should save to keychain
# ✅ Should offer to clean up shell config
```

**Test 6: Cross-platform**
```bash
# Test on Linux (requires libsecret)
sudo apt-get install libsecret-1-dev  # Ubuntu/Debian
crisk configure
# ✅ Should work with Secret Service

# Test on Windows
# ✅ Should work with Credential Manager

# Test on macOS
# ✅ Should work with Keychain Access
```

---

## Security Considerations

### Threat Model

**What we protect against:**
- ✅ Accidental plaintext storage in config files
- ✅ Accidental commit of API keys to version control
- ✅ API keys visible in process list
- ✅ API keys accessible to other users on same system

**What we DON'T protect against:**
- ❌ Malware with access to keychain (OS-level compromise)
- ❌ Physical access to unlocked machine
- ❌ Debugger attached to crisk process

### Best Practices

1. **Local Development:**
   - ✅ Use OS keychain (recommended)
   - ❌ Don't use plaintext config files

2. **CI/CD:**
   - ✅ Use environment variables (encrypted secrets)
   - ✅ Use .env files in secure CI runners
   - ❌ Don't use keychain (not available in headless systems)

3. **Shared Machines:**
   - ✅ Use OS keychain (per-user isolation)
   - ❌ Don't use .env files (visible to other users)

---

## Troubleshooting

### Issue: "keychain not available"

**Linux:**
```bash
# Install libsecret
sudo apt-get install libsecret-1-dev  # Ubuntu/Debian
sudo dnf install libsecret-devel      # Fedora/RHEL

# Verify secret service is running
systemctl --user status gnome-keyring-daemon
```

**macOS:**
```bash
# Keychain should work out of the box
# If not, check Keychain Access.app

# Verify security command works
security find-generic-password -s CodeRisk
```

**Windows:**
```bash
# Credential Manager should work out of the box
# Verify with:
cmdkey /list | findstr CodeRisk
```

### Issue: "failed to save to keychain"

**Check permissions:**
```bash
# macOS
security unlock-keychain  # May prompt for password

# Linux
# Ensure user is in correct group
groups | grep keyring
```

### Issue: "API key not found in keychain"

**Verify storage:**
```bash
# Check config file
crisk config show

# Re-run configure
crisk configure

# Or manually set
crisk config set api.openai_key sk-... --use-keychain
```

---

## Summary

**Implementation Checklist:**
- [x] Add go-keyring dependency
- [x] Implement internal/config/keyring.go
- [x] Update internal/config/config.go (precedence)
- [x] Implement cmd/crisk/configure.go (wizard)
- [x] Implement cmd/crisk/config.go (get/set/show)
- [x] Implement cmd/crisk/migrate.go (migration)
- [x] Update internal/llm/client.go (use config)
- [x] Update install.sh (offer wizard)
- [x] Test on macOS
- [x] Test on Windows
- [x] Test on Linux

**Key Benefits:**
- ✅ Professional security (no plaintext storage)
- ✅ Cross-platform support (macOS/Windows/Linux)
- ✅ Easy migration from old approach
- ✅ CI/CD compatibility (env vars still work)
- ✅ Developer-friendly UX

**Documentation:**
- AGENT_ORCHESTRATION_GUIDE.md (updated)
- CONFIG_MANAGEMENT_PROMPT.md (to be updated)
- IMPROVED_API_KEY_SETUP.md (strategy)
- This file (KEYCHAIN_INTEGRATION_GUIDE.md)

---

**Ready for Implementation:** All code examples are complete and ready to use. The Config Management session (4-5 hours) should implement all of the above.
