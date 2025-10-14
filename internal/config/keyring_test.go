package config

import (
	"os"
	"testing"
)

func TestKeyringManager_SaveAndGetAPIKey(t *testing.T) {
	km := NewKeyringManager()

	// Check if keychain is available (skip test on CI without keychain)
	if !km.IsAvailable() {
		t.Skip("Keychain not available, skipping test")
	}

	// Clean up before test
	defer km.DeleteAPIKey()

	testKey := "sk-test123456789"

	// Test Save
	err := km.SaveAPIKey(testKey)
	if err != nil {
		t.Fatalf("Failed to save API key: %v", err)
	}

	// Test Get
	retrievedKey, err := km.GetAPIKey()
	if err != nil {
		t.Fatalf("Failed to get API key: %v", err)
	}

	if retrievedKey != testKey {
		t.Errorf("Expected key %s, got %s", testKey, retrievedKey)
	}
}

func TestKeyringManager_DeleteAPIKey(t *testing.T) {
	km := NewKeyringManager()

	if !km.IsAvailable() {
		t.Skip("Keychain not available, skipping test")
	}

	testKey := "sk-test-delete-123"

	// Save a key first
	err := km.SaveAPIKey(testKey)
	if err != nil {
		t.Fatalf("Failed to save API key: %v", err)
	}

	// Delete the key
	err = km.DeleteAPIKey()
	if err != nil {
		t.Fatalf("Failed to delete API key: %v", err)
	}

	// Verify it's deleted
	retrievedKey, err := km.GetAPIKey()
	if err != nil {
		t.Fatalf("Error getting API key after deletion: %v", err)
	}
	if retrievedKey != "" {
		t.Errorf("Expected empty key after deletion, got %s", retrievedKey)
	}
}

func TestKeyringManager_GetAPIKey_NotFound(t *testing.T) {
	km := NewKeyringManager()

	if !km.IsAvailable() {
		t.Skip("Keychain not available, skipping test")
	}

	// Ensure no key exists
	km.DeleteAPIKey()

	// Try to get non-existent key
	retrievedKey, err := km.GetAPIKey()
	if err != nil {
		t.Fatalf("Expected no error for non-existent key, got: %v", err)
	}
	if retrievedKey != "" {
		t.Errorf("Expected empty string for non-existent key, got: %s", retrievedKey)
	}
}

func TestKeyringManager_SaveAPIKey_EmptyKey(t *testing.T) {
	km := NewKeyringManager()

	if !km.IsAvailable() {
		t.Skip("Keychain not available, skipping test")
	}

	// Try to save empty key
	err := km.SaveAPIKey("")
	if err == nil {
		t.Error("Expected error when saving empty API key")
	}
}

func TestKeyringManager_IsAvailable(t *testing.T) {
	km := NewKeyringManager()

	// Just verify the method doesn't panic
	available := km.IsAvailable()

	// We can't assert true/false since it depends on the environment
	// But we can verify it returns a boolean
	if available {
		t.Log("Keychain is available")
	} else {
		t.Log("Keychain is not available (headless system or missing dependencies)")
	}
}

func TestGetAPIKeySource_EnvironmentVariable(t *testing.T) {
	km := NewKeyringManager()
	cfg := Default()

	// Set environment variable
	testKey := "sk-env-test-123"
	os.Setenv("OPENAI_API_KEY", testKey)
	defer os.Unsetenv("OPENAI_API_KEY")

	// Get source info
	sourceInfo := km.GetAPIKeySource(cfg)

	if sourceInfo.Source != "env" {
		t.Errorf("Expected source 'env', got '%s'", sourceInfo.Source)
	}
	if !sourceInfo.Secure {
		t.Error("Expected env var source to be marked as secure")
	}
}

func TestGetAPIKeySource_Keychain(t *testing.T) {
	km := NewKeyringManager()

	if !km.IsAvailable() {
		t.Skip("Keychain not available, skipping test")
	}

	cfg := Default()

	// Ensure no env var
	os.Unsetenv("OPENAI_API_KEY")

	// Save key to keychain
	testKey := "sk-keychain-test-123"
	err := km.SaveAPIKey(testKey)
	if err != nil {
		t.Fatalf("Failed to save API key to keychain: %v", err)
	}
	defer km.DeleteAPIKey()

	// Get source info
	sourceInfo := km.GetAPIKeySource(cfg)

	if sourceInfo.Source != "keychain" {
		t.Errorf("Expected source 'keychain', got '%s'", sourceInfo.Source)
	}
	if !sourceInfo.Secure {
		t.Error("Expected keychain source to be marked as secure")
	}
}

func TestGetAPIKeySource_ConfigFile(t *testing.T) {
	km := NewKeyringManager()

	if !km.IsAvailable() {
		t.Skip("Keychain not available, skipping test")
	}

	cfg := Default()
	cfg.API.OpenAIKey = "sk-config-test-123"

	// Ensure no env var and no keychain key
	os.Unsetenv("OPENAI_API_KEY")
	km.DeleteAPIKey()

	// Get source info
	sourceInfo := km.GetAPIKeySource(cfg)

	if sourceInfo.Source != "config" {
		t.Errorf("Expected source 'config', got '%s'", sourceInfo.Source)
	}
	if sourceInfo.Secure {
		t.Error("Expected config file source to be marked as insecure")
	}
}

func TestGetAPIKeySource_None(t *testing.T) {
	km := NewKeyringManager()

	if !km.IsAvailable() {
		t.Skip("Keychain not available, skipping test")
	}

	cfg := Default()

	// Ensure no API key anywhere
	os.Unsetenv("OPENAI_API_KEY")
	km.DeleteAPIKey()
	cfg.API.OpenAIKey = ""

	// Get source info
	sourceInfo := km.GetAPIKeySource(cfg)

	if sourceInfo.Source != "none" {
		t.Errorf("Expected source 'none', got '%s'", sourceInfo.Source)
	}
	if sourceInfo.Secure {
		t.Error("Expected none source to be marked as insecure")
	}
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Standard API key",
			input:    "sk-proj-1234567890abcdefg",
			expected: "sk-proj...defg",
		},
		{
			name:     "Empty key",
			input:    "",
			expected: "(not set)",
		},
		{
			name:     "Short key",
			input:    "sk-test",
			expected: "***",
		},
		{
			name:     "Exact 12 chars",
			input:    "sk-test12345",
			expected: "sk-test...2345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskAPIKey(tt.input)
			if result != tt.expected {
				t.Errorf("MaskAPIKey(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestKeyringManager_RoundTrip(t *testing.T) {
	km := NewKeyringManager()

	if !km.IsAvailable() {
		t.Skip("Keychain not available, skipping test")
	}

	// Clean slate
	km.DeleteAPIKey()

	// Test multiple save/get cycles
	keys := []string{
		"sk-test-round-trip-1",
		"sk-test-round-trip-2",
		"sk-test-round-trip-3",
	}

	for _, key := range keys {
		// Save
		if err := km.SaveAPIKey(key); err != nil {
			t.Fatalf("Failed to save key %s: %v", key, err)
		}

		// Retrieve
		retrieved, err := km.GetAPIKey()
		if err != nil {
			t.Fatalf("Failed to get key: %v", err)
		}

		if retrieved != key {
			t.Errorf("Round trip failed: expected %s, got %s", key, retrieved)
		}
	}

	// Clean up
	km.DeleteAPIKey()
}

func TestKeyringManager_DeleteNonExistentKey(t *testing.T) {
	km := NewKeyringManager()

	if !km.IsAvailable() {
		t.Skip("Keychain not available, skipping test")
	}

	// Ensure key doesn't exist
	km.DeleteAPIKey()

	// Delete again (should not error)
	err := km.DeleteAPIKey()
	if err != nil {
		t.Errorf("Expected no error when deleting non-existent key, got: %v", err)
	}
}

// TestKeyringIntegration is a comprehensive integration test
func TestKeyringIntegration(t *testing.T) {
	km := NewKeyringManager()

	if !km.IsAvailable() {
		t.Skip("Keychain not available, skipping integration test")
	}

	// Clean slate - remove all sources
	oldEnv := os.Getenv("OPENAI_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")
	defer func() {
		if oldEnv != "" {
			os.Setenv("OPENAI_API_KEY", oldEnv)
		}
	}()

	km.DeleteAPIKey()
	defer km.DeleteAPIKey()

	cfg := Default()

	// Step 1: No key configured
	sourceInfo := km.GetAPIKeySource(cfg)
	if sourceInfo.Source != "none" {
		t.Errorf("Step 1: Expected source 'none', got '%s'", sourceInfo.Source)
	}

	// Step 2: Save to keychain
	testKey := "sk-integration-test-key"
	if err := km.SaveAPIKey(testKey); err != nil {
		t.Fatalf("Step 2: Failed to save key: %v", err)
	}

	// Step 3: Verify keychain is now the source
	sourceInfo = km.GetAPIKeySource(cfg)
	if sourceInfo.Source != "keychain" {
		t.Errorf("Step 3: Expected source 'keychain', got '%s'", sourceInfo.Source)
	}

	// Step 4: Set environment variable (should take precedence)
	os.Setenv("OPENAI_API_KEY", "sk-env-override")
	defer os.Unsetenv("OPENAI_API_KEY")

	sourceInfo = km.GetAPIKeySource(cfg)
	if sourceInfo.Source != "env" {
		t.Errorf("Step 4: Expected source 'env', got '%s'", sourceInfo.Source)
	}

	// Step 5: Remove env var, back to keychain
	os.Unsetenv("OPENAI_API_KEY")
	sourceInfo = km.GetAPIKeySource(cfg)
	if sourceInfo.Source != "keychain" {
		t.Errorf("Step 5: Expected source 'keychain', got '%s'", sourceInfo.Source)
	}

	// Step 6: Retrieve key from keychain
	retrieved, err := km.GetAPIKey()
	if err != nil {
		t.Fatalf("Step 6: Failed to get key: %v", err)
	}
	if retrieved != testKey {
		t.Errorf("Step 6: Expected key %s, got %s", testKey, retrieved)
	}

	// Step 7: Delete from keychain
	if err := km.DeleteAPIKey(); err != nil {
		t.Fatalf("Step 7: Failed to delete key: %v", err)
	}

	// Step 8: Verify key is gone
	sourceInfo = km.GetAPIKeySource(cfg)
	if sourceInfo.Source != "none" {
		t.Errorf("Step 8: Expected source 'none', got '%s'", sourceInfo.Source)
	}
}
