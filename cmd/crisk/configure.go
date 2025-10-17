package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
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
3. Budget limits (optional, for cost control)`,
	RunE: runConfigure,
}

func runConfigure(cmd *cobra.Command, args []string) error {
	fmt.Println("🔧 CodeRisk Configuration Wizard")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Load existing config if it exists
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".coderisk", "config.yaml")
	loadedCfg, err := config.Load(configPath)
	if err != nil {
		loadedCfg = config.Default()
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

	// Declare variables before potential goto
	var apiKey string
	var response string

	// Check existing sources
	sourceInfo := km.GetAPIKeySource(loadedCfg)

	if sourceInfo.Source != "none" {
		fmt.Printf("Current: %s\n", config.MaskAPIKey(loadedCfg.API.OpenAIKey))
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
	response, _ = reader.ReadString('\n')
	apiKey = strings.TrimSpace(response)

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
				loadedCfg.API.OpenAIKey = apiKey
				loadedCfg.API.UseKeychain = false
			} else {
				fmt.Println("✅ API key saved to OS keychain (secure)")
				loadedCfg.API.OpenAIKey = "" // Don't save in config file
				loadedCfg.API.UseKeychain = true

				// Show where it's stored based on OS
				fmt.Printf("   📍 %s\n", getKeychainLocation())
			}
		} else {
			// Save to config file
			loadedCfg.API.OpenAIKey = apiKey
			loadedCfg.API.UseKeychain = false
			fmt.Println("✅ API key saved to config file (plaintext)")
			fmt.Println("   ⚠️  Consider using keychain for better security")
		}
	} else {
		// No keychain available, save to config file
		loadedCfg.API.OpenAIKey = apiKey
		loadedCfg.API.UseKeychain = false
		fmt.Println("✅ API key saved to config file")
	}

step2:
	fmt.Println()
	fmt.Println("Step 2/4: LLM Model")
	fmt.Println()
	fmt.Println("Available models:")
	fmt.Println("  1. gpt-4o-mini (recommended, fast, $0.03-0.05/check)")
	fmt.Println("  2. gpt-4o (slower, higher quality, $0.15-0.20/check)")
	fmt.Printf("Current: %s\n", loadedCfg.API.OpenAIModel)
	fmt.Print("Select model (1-2) or press Enter to keep current: ")

	response, _ = reader.ReadString('\n')
	response = strings.TrimSpace(response)

	switch response {
	case "1":
		loadedCfg.API.OpenAIModel = "gpt-4o-mini"
		fmt.Println("✅ Using gpt-4o-mini")
	case "2":
		loadedCfg.API.OpenAIModel = "gpt-4o"
		fmt.Println("✅ Using gpt-4o")
	case "":
		fmt.Printf("✅ Keeping %s\n", loadedCfg.API.OpenAIModel)
	}
	fmt.Println()

	// Step 3: Budget Limits
	fmt.Println("Step 3/4: Budget Limits (Optional)")
	fmt.Println()
	fmt.Println("Set spending limits to control costs:")
	fmt.Printf("Current daily limit: $%.2f\n", loadedCfg.Budget.DailyLimit)
	fmt.Printf("Current monthly limit: $%.2f\n", loadedCfg.Budget.MonthlyLimit)
	fmt.Print("Change budget limits? (y/N): ")

	response, _ = reader.ReadString('\n')
	response = strings.TrimSpace(response)

	if strings.ToLower(response) == "y" {
		fmt.Print("Daily limit ($): ")
		var daily float64
		fmt.Scanln(&daily)
		if daily > 0 {
			loadedCfg.Budget.DailyLimit = daily
		}

		fmt.Print("Monthly limit ($): ")
		var monthly float64
		fmt.Scanln(&monthly)
		if monthly > 0 {
			loadedCfg.Budget.MonthlyLimit = monthly
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
		if err := loadedCfg.Save(configPath); err != nil {
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

		if loadedCfg.API.UseKeychain {
			fmt.Println("🔒 Security: API key stored securely in OS keychain")
		}
	} else {
		fmt.Println("⏭️  Configuration not saved")
	}

	return nil
}

func getKeychainLocation() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS Keychain Access.app → 'CodeRisk'"
	case "windows":
		return "Windows Credential Manager → 'CodeRisk'"
	case "linux":
		return "Linux Secret Service (libsecret)"
	default:
		return "OS Keychain"
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
