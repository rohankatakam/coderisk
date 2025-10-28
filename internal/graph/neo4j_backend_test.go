package graph

import (
	"testing"
)

func TestParseNodeID_IntegerConversion(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedLabel string
		expectedID    interface{}
		expectedType  string
	}{
		// PR numbers should be integers
		{
			name:          "PR with number",
			input:         "pr:42",
			expectedLabel: "PR",
			expectedID:    42,
			expectedType:  "int",
		},
		{
			name:          "PR with large number",
			input:         "pr:1234",
			expectedLabel: "PR",
			expectedID:    1234,
			expectedType:  "int",
		},
		// Issue numbers should be integers
		{
			name:          "Issue with number",
			input:         "issue:123",
			expectedLabel: "Issue",
			expectedID:    123,
			expectedType:  "int",
		},
		// Commits should remain strings (SHA hashes)
		{
			name:          "Commit with SHA",
			input:         "commit:abc123def456",
			expectedLabel: "Commit",
			expectedID:    "abc123def456",
			expectedType:  "string",
		},
		{
			name:          "Commit with full SHA",
			input:         "commit:1a3b331c039106696d6af6e10942e3e71f6180f2",
			expectedLabel: "Commit",
			expectedID:    "1a3b331c039106696d6af6e10942e3e71f6180f2",
			expectedType:  "string",
		},
		// Files should remain strings
		{
			name:          "File with path",
			input:         "file:src/main.go",
			expectedLabel: "File",
			expectedID:    "src/main.go",
			expectedType:  "string",
		},
		{
			name:          "File with complex path",
			input:         "file:apps/web/src/components/Dashboard.tsx",
			expectedLabel: "File",
			expectedID:    "apps/web/src/components/Dashboard.tsx",
			expectedType:  "string",
		},
		// Developers should remain strings (emails)
		{
			name:          "Developer with email",
			input:         "developer:john@example.com",
			expectedLabel: "Developer",
			expectedID:    "john@example.com",
			expectedType:  "string",
		},
		{
			name:          "Developer with noreply email",
			input:         "developer:john@users.noreply.github.com",
			expectedLabel: "Developer",
			expectedID:    "john@users.noreply.github.com",
			expectedType:  "string",
		},
		// Edge cases
		{
			name:          "File path without prefix (backwards compatibility)",
			input:         "src/utils/helper.ts",
			expectedLabel: "File",
			expectedID:    "src/utils/helper.ts",
			expectedType:  "string",
		},
		{
			name:          "PR with invalid number falls back to string",
			input:         "pr:invalid",
			expectedLabel: "PR",
			expectedID:    "invalid",
			expectedType:  "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			label, id := parseNodeID(tt.input)

			// Check label
			if label != tt.expectedLabel {
				t.Errorf("parseNodeID(%s) label = %s; want %s", tt.input, label, tt.expectedLabel)
			}

			// Check ID value
			if id != tt.expectedID {
				t.Errorf("parseNodeID(%s) id = %v; want %v", tt.input, id, tt.expectedID)
			}

			// Check ID type
			switch tt.expectedType {
			case "int":
				if _, ok := id.(int); !ok {
					t.Errorf("parseNodeID(%s) id type = %T; want int", tt.input, id)
				}
			case "string":
				if _, ok := id.(string); !ok {
					t.Errorf("parseNodeID(%s) id type = %T; want string", tt.input, id)
				}
			}
		})
	}
}

func TestParseNodeID_TypeMatching(t *testing.T) {
	// This test verifies the type mismatch fix
	// PR/Issue numbers must be integers to match Neo4j storage

	prLabel, prID := parseNodeID("pr:42")

	if prLabel != "PR" {
		t.Errorf("Expected label PR, got %s", prLabel)
	}

	// Critical assertion: ID must be an integer, not string
	prNumber, ok := prID.(int)
	if !ok {
		t.Fatalf("Expected PR ID to be int, got %T. This will cause type mismatch in Neo4j queries!", prID)
	}

	if prNumber != 42 {
		t.Errorf("Expected PR number 42, got %d", prNumber)
	}

	// Verify that string "42" would NOT match (this was the bug)
	if prID == "42" {
		t.Error("PR ID should be int 42, not string \"42\". Type mismatch bug still exists!")
	}
}

func TestGetUniqueKey(t *testing.T) {
	tests := []struct {
		label       string
		expectedKey string
	}{
		{"File", "path"},
		{"file", "path"},
		{"Developer", "email"},
		{"developer", "email"},
		{"Commit", "sha"},
		{"commit", "sha"},
		{"PR", "number"},
		{"pr", "number"},
		{"Issue", "number"},
		{"issue", "number"},
		{"Unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			key := getUniqueKey(tt.label)
			if key != tt.expectedKey {
				t.Errorf("getUniqueKey(%s) = %s; want %s", tt.label, key, tt.expectedKey)
			}
		})
	}
}
