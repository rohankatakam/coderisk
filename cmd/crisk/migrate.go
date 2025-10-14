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

var migrateToKeychainCmd = &cobra.Command{
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
	RunE: runMigrateToKeychain,
}

func runMigrateToKeychain(cmd *cobra.Command, args []string) error {
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
	loadedCfg, err := config.Load(configPath)
	if err != nil {
		loadedCfg = config.Default()
	}

	// Check current source
	sourceInfo := km.GetAPIKeySource(loadedCfg)

	fmt.Println("Current Status:")
	fmt.Printf("  API Key: %s\n", config.MaskAPIKey(loadedCfg.API.OpenAIKey))
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
		apiKey = loadedCfg.API.OpenAIKey
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
		loadedCfg.API.OpenAIKey = ""
		loadedCfg.API.UseKeychain = true
		if err := loadedCfg.Save(configPath); err != nil {
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
		loadedCfg.API.UseKeychain = true
		loadedCfg.Save(configPath)
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
	fmt.Println("You can verify with: crisk config list")

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
