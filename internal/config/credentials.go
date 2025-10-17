package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/rohankatakam/coderisk/internal/errors"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

// CredentialManager handles credential retrieval with priority chain
// Priority: Environment Variables → Keychain → Config File → Interactive Prompt
type CredentialManager struct {
	mode       DeploymentMode
	keyring    *KeyringManager
	configPath string
}

// Credentials holds all user credentials
type Credentials struct {
	OpenAIAPIKey string `yaml:"openai_api_key"`
	GitHubToken  string `yaml:"github_token"`
}

// NewCredentialManager creates a new credential manager
func NewCredentialManager() *CredentialManager {
	mode := DetectMode()
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".config", "coderisk", "config.yaml")

	return &CredentialManager{
		mode:       mode,
		keyring:    NewKeyringManager(),
		configPath: configPath,
	}
}

// GetOpenAIAPIKey retrieves the OpenAI API key using priority chain
func (cm *CredentialManager) GetOpenAIAPIKey() (string, error) {
	// 1. Environment variable (highest priority)
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return key, nil
	}

	// 2. Keychain (macOS/Linux)
	if cm.keyring.IsAvailable() {
		if key, err := cm.keyring.GetAPIKey(); err == nil && key != "" {
			return key, nil
		}
	}

	// 3. Config file (~/.config/coderisk/config.yaml)
	if creds, err := cm.loadConfigFile(); err == nil && creds.OpenAIAPIKey != "" {
		return creds.OpenAIAPIKey, nil
	}

	// 4. Interactive prompt (only in packaged mode, not in CI)
	if cm.mode.AllowsInteractivePrompts() && isInteractive() {
		fmt.Println("\n⚠️  OpenAI API Key not found.")
		fmt.Println("   Create one at: https://platform.openai.com/api-keys")
		fmt.Println()
		return cm.promptForAPIKey()
	}

	// Not found anywhere
	return "", errors.ConfigErrorf(
		"OPENAI_API_KEY not found. Set it via:\n"+
			"  1. Environment variable: export OPENAI_API_KEY=sk-...\n"+
			"  2. Run: crisk configure (to set up keychain)\n"+
			"  3. Config file: %s", cm.configPath)
}

// GetGitHubToken retrieves the GitHub token using priority chain
func (cm *CredentialManager) GetGitHubToken() (string, error) {
	// 1. Environment variable (highest priority)
	for _, envVar := range []string{"GITHUB_TOKEN", "GH_TOKEN"} {
		if token := os.Getenv(envVar); token != "" {
			return token, nil
		}
	}

	// 2. Keychain (macOS/Linux)
	if cm.keyring.IsAvailable() {
		if token, err := cm.keyring.GetGitHubToken(); err == nil && token != "" {
			return token, nil
		}
	}

	// 3. Config file
	if creds, err := cm.loadConfigFile(); err == nil && creds.GitHubToken != "" {
		return creds.GitHubToken, nil
	}

	// 4. Interactive prompt (optional credential)
	if cm.mode.AllowsInteractivePrompts() && isInteractive() {
		fmt.Println("\n⚠️  GitHub Token not found (optional).")
		fmt.Println("   Required for: private repos, higher rate limits")
		fmt.Println("   Create one at: https://github.com/settings/tokens")
		fmt.Println()
		fmt.Print("Enter GitHub Token (or press Enter to skip): ")

		token, _ := cm.readSecurely()
		if token != "" {
			// Save to keychain if available
			if cm.keyring.IsAvailable() {
				cm.keyring.SetGitHubToken(token)
			}
			return token, nil
		}
		return "", nil // Optional, return empty
	}

	// GitHub token is optional for public repos
	return "", nil
}

// SaveCredentials saves credentials to keychain (preferred) or config file (fallback)
func (cm *CredentialManager) SaveCredentials(creds Credentials) error {
	// Try keychain first (macOS/Linux)
	if cm.keyring.IsAvailable() {
		if creds.OpenAIAPIKey != "" {
			if err := cm.keyring.SetAPIKey(creds.OpenAIAPIKey); err != nil {
				return errors.Wrap(err, errors.ErrorTypeConfig, errors.SeverityHigh,
					"failed to save OpenAI API key to keychain")
			}
		}
		if creds.GitHubToken != "" {
			if err := cm.keyring.SetGitHubToken(creds.GitHubToken); err != nil {
				return errors.Wrap(err, errors.ErrorTypeConfig, errors.SeverityHigh,
					"failed to save GitHub token to keychain")
			}
		}
		return nil
	}

	// Fallback: Save to config file
	return cm.saveConfigFile(creds)
}

// loadConfigFile loads credentials from config file
func (cm *CredentialManager) loadConfigFile() (*Credentials, error) {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return nil, err
	}

	var creds Credentials
	if err := yaml.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	return &creds, nil
}

// saveConfigFile saves credentials to config file
func (cm *CredentialManager) saveConfigFile(creds Credentials) error {
	// Ensure directory exists
	dir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Marshal to YAML
	data, err := yaml.Marshal(creds)
	if err != nil {
		return err
	}

	// Write file with restrictive permissions (user-only read/write)
	if err := os.WriteFile(cm.configPath, data, 0600); err != nil {
		return err
	}

	return nil
}

// promptForAPIKey prompts user for OpenAI API key
func (cm *CredentialManager) promptForAPIKey() (string, error) {
	fmt.Print("Enter OpenAI API Key: ")
	key, err := cm.readSecurely()
	if err != nil {
		return "", err
	}

	if key == "" {
		return "", errors.ConfigError("OpenAI API key is required")
	}

	// Validate format (starts with sk-)
	if !strings.HasPrefix(key, "sk-") {
		return "", errors.ValidationError("OpenAI API key should start with 'sk-'")
	}

	// Save to keychain if available
	if cm.keyring.IsAvailable() {
		if err := cm.keyring.SetAPIKey(key); err == nil {
			fmt.Println("✓ Saved to keychain")
		}
	} else {
		// Save to config file as fallback
		creds := Credentials{OpenAIAPIKey: key}
		if err := cm.saveConfigFile(creds); err == nil {
			fmt.Printf("✓ Saved to %s\n", cm.configPath)
		}
	}

	return key, nil
}

// readSecurely reads a password/token from stdin without echoing
func (cm *CredentialManager) readSecurely() (string, error) {
	// Try to read from terminal (supports password masking)
	if term.IsTerminal(int(syscall.Stdin)) {
		bytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println() // New line after password input
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(bytes)), nil
	}

	// Fallback: Read from stdin (piped input)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// isInteractive returns true if stdin is a terminal (not piped)
func isInteractive() bool {
	return term.IsTerminal(int(syscall.Stdin))
}

// GetMode returns the current deployment mode
func (cm *CredentialManager) GetMode() DeploymentMode {
	return cm.mode
}

// GetConfigPath returns the path to the config file
func (cm *CredentialManager) GetConfigPath() string {
	return cm.configPath
}

// HasCredentials checks if credentials are configured
func (cm *CredentialManager) HasCredentials() bool {
	// Check environment
	if os.Getenv("OPENAI_API_KEY") != "" {
		return true
	}

	// Check keychain
	if cm.keyring.IsAvailable() {
		if key, err := cm.keyring.GetAPIKey(); err == nil && key != "" {
			return true
		}
	}

	// Check config file
	if creds, err := cm.loadConfigFile(); err == nil && creds.OpenAIAPIKey != "" {
		return true
	}

	return false
}
