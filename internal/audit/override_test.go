package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogOverride(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Test 1: Log first override
	event1 := OverrideEvent{
		Timestamp:  time.Now(),
		Author:     "test@example.com",
		CommitSHA:  "abc123",
		RiskLevel:  "HIGH",
		FilesCount: 3,
		Issues:     []string{"no_tests", "high_coupling"},
		Reason:     "urgent_hotfix",
	}

	if err := LogOverride(event1); err != nil {
		t.Fatalf("LogOverride() error = %v", err)
	}

	// Verify .coderisk directory was created
	if _, err := os.Stat(".coderisk"); os.IsNotExist(err) {
		t.Error(".coderisk directory was not created")
	}

	// Verify log file was created
	logPath := filepath.Join(".coderisk", "hook_log.jsonl")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("hook_log.jsonl was not created")
	}

	// Test 2: Log second override (append)
	event2 := OverrideEvent{
		Timestamp:  time.Now(),
		Author:     "test@example.com",
		CommitSHA:  "def456",
		RiskLevel:  "MEDIUM",
		FilesCount: 1,
		Issues:     []string{"low_coverage"},
		Reason:     "will_fix_later",
	}

	if err := LogOverride(event2); err != nil {
		t.Fatalf("LogOverride() second call error = %v", err)
	}

	// Test 3: Read and verify log contents
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	// Parse JSONL (each line is a JSON object)
	lines := 0
	decoder := json.NewDecoder(os.Stdin)
	for decoder.More() {
		var event OverrideEvent
		if err := decoder.Decode(&event); err != nil {
			break
		}
		lines++
	}

	// Should have 2 events (count newlines)
	eventCount := 0
	for _, c := range content {
		if c == '\n' {
			eventCount++
		}
	}

	if eventCount != 2 {
		t.Errorf("Expected 2 events in log, found %d", eventCount)
	}

	// Verify first event can be parsed
	var firstEvent OverrideEvent
	firstLine := content[:len(content)]
	for i, c := range content {
		if c == '\n' {
			firstLine = content[:i]
			break
		}
	}

	if err := json.Unmarshal(firstLine, &firstEvent); err != nil {
		t.Errorf("Failed to parse first event: %v", err)
	}

	if firstEvent.Author != event1.Author {
		t.Errorf("Expected author %s, got %s", event1.Author, firstEvent.Author)
	}
	if firstEvent.RiskLevel != event1.RiskLevel {
		t.Errorf("Expected risk level %s, got %s", event1.RiskLevel, firstEvent.RiskLevel)
	}
	if firstEvent.FilesCount != event1.FilesCount {
		t.Errorf("Expected files count %d, got %d", event1.FilesCount, firstEvent.FilesCount)
	}
}

func TestLogOverride_DirectoryCreation(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Verify .coderisk doesn't exist yet
	if _, err := os.Stat(".coderisk"); !os.IsNotExist(err) {
		t.Fatal(".coderisk should not exist yet")
	}

	// Log override should create directory
	event := OverrideEvent{
		Timestamp:  time.Now(),
		Author:     "test@example.com",
		RiskLevel:  "HIGH",
		FilesCount: 1,
		Issues:     []string{"test"},
	}

	if err := LogOverride(event); err != nil {
		t.Fatalf("LogOverride() error = %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(".coderisk")
	if err != nil {
		t.Fatalf("Failed to stat .coderisk: %v", err)
	}
	if !info.IsDir() {
		t.Error(".coderisk is not a directory")
	}
}
