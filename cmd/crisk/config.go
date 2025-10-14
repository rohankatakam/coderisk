package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coderisk/coderisk-go/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CodeRisk configuration",
	Long:  `View and modify CodeRisk configuration settings.`,
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get configuration value",
	Long: `Get a configuration value, optionally showing where it's stored.

Examples:
  # Get API key value
  crisk config get api.openai_key

  # Show where the API key is stored
  crisk config get api.openai_key --show-source`,
	Args: cobra.MaximumNArgs(1),
	RunE: runConfigGet,
}

var (
	useKeychain bool
	noKeychain  bool
	showSource  bool
)

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set configuration value",
	Long: `Set a configuration value with optional keychain storage.

Examples:
  # Store API key in OS keychain (secure, recommended)
  crisk config set api.openai_key sk-... --use-keychain

  # Store API key in config file (plaintext, for CI/CD)
  crisk config set api.openai_key sk-... --no-keychain

  # If neither flag specified, will prompt user`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	RunE:  runConfigList,
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit configuration in default editor",
	RunE:  runConfigEdit,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	RunE:  runConfigInit,
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configInitCmd)

	// Add flags
	configSetCmd.Flags().BoolVar(&useKeychain, "use-keychain", false, "Store API key in OS keychain (secure)")
	configSetCmd.Flags().BoolVar(&noKeychain, "no-keychain", false, "Store API key in config file (plaintext)")
	configGetCmd.Flags().BoolVar(&showSource, "show-source", false, "Show where the value is stored")
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return runConfigList(cmd, args)
	}

	key := args[0]

	// Special handling for api.openai_key with source info
	if key == "api.openai_key" && showSource {
		km := config.NewKeyringManager()
		sourceInfo := km.GetAPIKeySource(cfg)

		fmt.Printf("%s\n", config.MaskAPIKey(cfg.API.OpenAIKey))
		fmt.Printf("Source: %s\n", sourceInfo.Source)
		if sourceInfo.Secure {
			fmt.Println("Security: ✅ Secure")
		} else {
			fmt.Println("Security: ⚠️  Plaintext")
		}
		return nil
	}

	value := getConfigValue(cfg, key)
	if value == nil {
		fmt.Printf("Configuration key '%s' not found\n", key)
		return nil
	}

	fmt.Printf("%s = %v\n", key, value)
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
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

		// Save config file
		configPath := getConfigPath()
		if err := cfg.Save(configPath); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		return nil
	}

	// Handle other config keys normally
	if err := setConfigValue(cfg, key, value); err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	// Save configuration
	configPath := getConfigPath()
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✅ Set %s = %s\n", key, value)
	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	fmt.Println("📋 CodeRisk Configuration")
	fmt.Println("════════════════════════")

	fmt.Printf("\n🏗️  General:\n")
	fmt.Printf("  mode = %s\n", cfg.Mode)

	fmt.Printf("\n💾 Storage:\n")
	fmt.Printf("  storage.type = %s\n", cfg.Storage.Type)
	fmt.Printf("  storage.local_path = %s\n", cfg.Storage.LocalPath)
	if cfg.Storage.PostgresDSN != "" {
		fmt.Printf("  storage.postgres_dsn = %s\n", maskDSN(cfg.Storage.PostgresDSN))
	}

	fmt.Printf("\n🐙 GitHub:\n")
	if cfg.GitHub.Token != "" {
		fmt.Printf("  github.token = %s\n", maskToken(cfg.GitHub.Token))
	} else {
		fmt.Printf("  github.token = (not set)\n")
	}
	fmt.Printf("  github.rate_limit = %d\n", cfg.GitHub.RateLimit)

	fmt.Printf("\n🗂️  Cache:\n")
	fmt.Printf("  cache.directory = %s\n", cfg.Cache.Directory)
	fmt.Printf("  cache.ttl = %s\n", cfg.Cache.TTL)
	fmt.Printf("  cache.max_size = %d\n", cfg.Cache.MaxSize)
	if cfg.Cache.SharedCacheURL != "" {
		fmt.Printf("  cache.shared_cache_url = %s\n", cfg.Cache.SharedCacheURL)
	}

	fmt.Printf("\n🤖 API:\n")
	if cfg.API.OpenAIKey != "" {
		km := config.NewKeyringManager()
		sourceInfo := km.GetAPIKeySource(cfg)
		fmt.Printf("  api.openai_key = %s\n", config.MaskAPIKey(cfg.API.OpenAIKey))
		fmt.Printf("    Source: %s\n", sourceInfo.Recommended)
	} else {
		fmt.Printf("  api.openai_key = (not set)\n")
	}
	fmt.Printf("  api.openai_model = %s\n", cfg.API.OpenAIModel)
	if cfg.API.CustomLLMURL != "" {
		fmt.Printf("  api.custom_llm_url = %s\n", cfg.API.CustomLLMURL)
	}

	fmt.Printf("\n⚠️  Risk:\n")
	fmt.Printf("  risk.default_level = %d\n", cfg.Risk.DefaultLevel)
	fmt.Printf("  risk.low_threshold = %.2f\n", cfg.Risk.LowThreshold)
	fmt.Printf("  risk.medium_threshold = %.2f\n", cfg.Risk.MediumThreshold)
	fmt.Printf("  risk.high_threshold = %.2f\n", cfg.Risk.HighThreshold)

	fmt.Printf("\n🔄 Sync:\n")
	fmt.Printf("  sync.auto_sync = %v\n", cfg.Sync.AutoSync)
	fmt.Printf("  sync.fresh_threshold = %s\n", cfg.Sync.FreshThreshold)
	fmt.Printf("  sync.stale_threshold = %s\n", cfg.Sync.StaleThreshold)

	fmt.Printf("\n💰 Budget:\n")
	fmt.Printf("  budget.daily_limit = $%.2f\n", cfg.Budget.DailyLimit)
	fmt.Printf("  budget.monthly_limit = $%.2f\n", cfg.Budget.MonthlyLimit)
	fmt.Printf("  budget.per_check_limit = $%.2f\n", cfg.Budget.PerCheckLimit)

	return nil
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	configPath := getConfigPath()

	// Ensure config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := runConfigInit(cmd, args); err != nil {
			return err
		}
	}

	// Get editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano" // Default fallback
	}

	fmt.Printf("Opening config file in %s...\n", editor)
	fmt.Printf("Config file: %s\n", configPath)

	// Note: In a real implementation, you'd use os/exec to open the editor
	fmt.Println("(Note: Editor opening not implemented in this demo)")

	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	configPath := getConfigPath()

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Configuration file already exists at %s\n", configPath)
		fmt.Print("Overwrite? [y/N]: ")

		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Initialization cancelled")
			return nil
		}
	}

	// Create default config
	defaultCfg := config.Default()

	// Save config file
	if err := defaultCfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✅ Created configuration file: %s\n", configPath)
	fmt.Println("\n💡 Next steps:")
	fmt.Println("  1. Set your GitHub token: crisk config set github.token <your-token>")
	fmt.Println("  2. Optionally set OpenAI API key: crisk config set api.openai_key <your-key>")
	fmt.Println("  3. Run 'crisk init' to initialize your repository")

	return nil
}

// Helper functions

func getConfigPath() string {
	if cfgFile != "" {
		return cfgFile
	}

	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".coderisk", "config.yaml")
}

func getConfigValue(cfg *config.Config, key string) interface{} {
	switch key {
	case "mode":
		return cfg.Mode
	case "storage.type":
		return cfg.Storage.Type
	case "storage.local_path":
		return cfg.Storage.LocalPath
	case "storage.postgres_dsn":
		return maskDSN(cfg.Storage.PostgresDSN)
	case "github.token":
		return maskToken(cfg.GitHub.Token)
	case "github.rate_limit":
		return cfg.GitHub.RateLimit
	case "cache.directory":
		return cfg.Cache.Directory
	case "api.openai_model":
		return cfg.API.OpenAIModel
	case "risk.default_level":
		return cfg.Risk.DefaultLevel
	case "sync.auto_sync":
		return cfg.Sync.AutoSync
	case "budget.daily_limit":
		return cfg.Budget.DailyLimit
	default:
		return nil
	}
}

func setConfigValue(cfg *config.Config, key, value string) error {
	switch key {
	case "mode":
		cfg.Mode = value
	case "storage.type":
		cfg.Storage.Type = value
	case "storage.local_path":
		cfg.Storage.LocalPath = value
	case "github.token":
		cfg.GitHub.Token = value
	case "api.openai_key":
		cfg.API.OpenAIKey = value
	case "api.openai_model":
		cfg.API.OpenAIModel = value
	case "sync.auto_sync":
		cfg.Sync.AutoSync = value == "true"
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}
	return nil
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

func maskDSN(dsn string) string {
	if dsn == "" {
		return ""
	}
	return "postgres://***:***@host/db"
}
