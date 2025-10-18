package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

// EnvLoader handles loading environment variables from .env file
// Reference: DEVELOPMENT_WORKFLOW.md §3.3 - Security guardrails
type EnvLoader struct {
	loaded bool
	path   string
}

// NewEnvLoader creates an environment loader
func NewEnvLoader() *EnvLoader {
	return &EnvLoader{}
}

// Load loads environment variables from .env file in project root
// This ensures all secrets come from a single source
func (e *EnvLoader) Load() error {
	if e.loaded {
		return nil // Already loaded
	}

	// Try to find .env file in current directory or parent directories
	envPath, err := findEnvFile()
	if err != nil {
		return fmt.Errorf("failed to find .env file: %w\nPlease create .env from .env.example", err)
	}

	e.path = envPath

	// Load .env file
	if err := godotenv.Load(envPath); err != nil {
		return fmt.Errorf("failed to load %s: %w", envPath, err)
	}

	// In verbose mode, log the loaded .env path
	if os.Getenv("VERBOSE") != "" || os.Getenv("DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "📝 Loaded .env from: %s\n", envPath)
	}

	e.loaded = true
	return nil
}

// MustLoad loads .env or panics (use for CLI commands)
func (e *EnvLoader) MustLoad() {
	if err := e.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nQuick setup:\n")
		fmt.Fprintf(os.Stderr, "  1. cp .env.example .env\n")
		fmt.Fprintf(os.Stderr, "  2. Edit .env and add your GITHUB_TOKEN\n")
		fmt.Fprintf(os.Stderr, "  3. Verify .env is in .gitignore\n")
		os.Exit(1)
	}
}

// GetPath returns the path to the loaded .env file
func (e *EnvLoader) GetPath() string {
	return e.path
}

// Validate checks that all required environment variables are set
func (e *EnvLoader) Validate() error {
	required := []string{
		"POSTGRES_DB",
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"NEO4J_USER",
		"NEO4J_PASSWORD",
	}

	missing := []string{}
	for _, key := range required {
		if os.Getenv(key) == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missing)
	}

	return nil
}

// ValidateWithGitHub validates including GitHub token (for init command)
func (e *EnvLoader) ValidateWithGitHub() error {
	if err := e.Validate(); err != nil {
		return err
	}

	if os.Getenv("GITHUB_TOKEN") == "" {
		return fmt.Errorf("GITHUB_TOKEN is required for fetching repository data.\nCreate a token at: https://github.com/settings/tokens")
	}

	return nil
}

// findEnvFile searches for .env file based on deployment mode
func findEnvFile() (string, error) {
	// In development mode, look for .env relative to the binary location
	// This allows running ./bin/crisk from anywhere and finding the coderisk repo's .env
	if IsDevelopment() {
		execPath, err := os.Executable()
		if err == nil {
			// Get the directory containing the binary
			// For ./bin/crisk, this gives us /path/to/coderisk/bin
			binDir := filepath.Dir(execPath)

			// Go up one level to get the repo root
			// /path/to/coderisk/bin -> /path/to/coderisk
			repoRoot := filepath.Dir(binDir)

			// Check if .env exists in repo root
			envPath := filepath.Join(repoRoot, ".env")
			if _, err := os.Stat(envPath); err == nil {
				return envPath, nil
			}
		}
	}

	// Fall back to searching from current directory (for other modes or if binary path fails)
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Search up the directory tree (max 5 levels)
	searchPath := cwd
	for i := 0; i < 5; i++ {
		envPath := filepath.Join(searchPath, ".env")
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}

		// Move up one directory
		parent := filepath.Dir(searchPath)
		if parent == searchPath {
			break // Reached root
		}
		searchPath = parent
	}

	return "", fmt.Errorf(".env file not found in %s or parent directories", cwd)
}

// Helper functions for type-safe environment variable access

// GetString returns string value or default
func GetString(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// GetInt returns int value or default
func GetInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

// GetBool returns bool value or default
func GetBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return boolVal
		}
	}
	return defaultVal
}

// MustGetString returns string value or panics
func MustGetString(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return val
}
