package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Token represents the stored authentication token
type Token struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
	User        User      `json:"user"`
}

// User represents the authenticated user
type User struct {
	ID    string `json:"id"`     // Clerk user ID
	Email string `json:"email"`
}

// GetTokenPath returns the path to the auth token file
func GetTokenPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %w", err)
	}

	tokenDir := filepath.Join(configDir, "coderisk")
	if err := os.MkdirAll(tokenDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(tokenDir, "auth.json"), nil
}

// SaveToken saves the authentication token to disk
func SaveToken(token *Token) error {
	tokenPath, err := GetTokenPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := os.WriteFile(tokenPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// LoadToken loads the authentication token from disk
func LoadToken() (*Token, error) {
	tokenPath, err := GetTokenPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not authenticated. Run: crisk login")
		}
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token file: %w", err)
	}

	return &token, nil
}

// DeleteToken deletes the authentication token from disk
func DeleteToken() error {
	tokenPath, err := GetTokenPath()
	if err != nil {
		return err
	}

	if err := os.Remove(tokenPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete token file: %w", err)
	}

	return nil
}

// IsAuthenticated checks if the user is authenticated and token is valid
func IsAuthenticated() bool {
	token, err := LoadToken()
	if err != nil {
		return false
	}

	// Check if token is expired
	if time.Now().After(token.ExpiresAt) {
		return false
	}

	return true
}

// GetAccessToken returns the current access token if valid
func GetAccessToken() (string, error) {
	token, err := LoadToken()
	if err != nil {
		return "", err
	}

	// Check if token is expired
	if time.Now().After(token.ExpiresAt) {
		return "", fmt.Errorf("token expired. Run: crisk login")
	}

	return token.AccessToken, nil
}
