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

	// KeyringGitHubTokenItem is the key for GitHub token
	KeyringGitHubTokenItem = "github-token"
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

// SetAPIKey is an alias for SaveAPIKey for consistency with credentials.go
func (km *KeyringManager) SetAPIKey(apiKey string) error {
	return km.SaveAPIKey(apiKey)
}

// GetGitHubToken retrieves GitHub token from OS keychain
func (km *KeyringManager) GetGitHubToken() (string, error) {
	token, err := keyring.Get(KeyringService, KeyringGitHubTokenItem)
	if err == keyring.ErrNotFound {
		// Not an error - just not set yet
		return "", nil
	}
	if err != nil {
		km.logger.Error("failed to get GitHub token from keychain", "error", err)
		return "", fmt.Errorf("failed to read from OS keychain: %w", err)
	}

	km.logger.Debug("github token retrieved from keychain")
	return token, nil
}

// SetGitHubToken stores GitHub token securely in OS keychain
func (km *KeyringManager) SetGitHubToken(token string) error {
	if token == "" {
		return fmt.Errorf("github token cannot be empty")
	}

	err := keyring.Set(KeyringService, KeyringGitHubTokenItem, token)
	if err != nil {
		km.logger.Error("failed to save GitHub token to keychain", "error", err)
		return fmt.Errorf("failed to save to OS keychain: %w", err)
	}

	km.logger.Info("github token saved to keychain", "service", KeyringService)
	return nil
}

// DeleteGitHubToken removes GitHub token from OS keychain
func (km *KeyringManager) DeleteGitHubToken() error {
	err := keyring.Delete(KeyringService, KeyringGitHubTokenItem)
	if err == keyring.ErrNotFound {
		// Already deleted, not an error
		return nil
	}
	if err != nil {
		km.logger.Error("failed to delete GitHub token from keychain", "error", err)
		return fmt.Errorf("failed to delete from OS keychain: %w", err)
	}

	km.logger.Info("github token deleted from keychain")
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

// KeySourceInfo returns information about where the API key is stored
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
