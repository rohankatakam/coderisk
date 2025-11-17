package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rohankatakam/coderisk/internal/atomizer"
	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/llm"
)

func main() {
	ctx := context.Background()

	// Get commit SHAs from args or use default
	commits := []string{
		"aa268b4", // Migrate to PostgreSQL Schema V2
		"4351ba6", // Remove TreeSitter
		"e423285", // Add timeline edge support
		"6d5f19d", // fix: Critical extraction bug
		"56f0d6a", // feat: Improve graph edge semantics
		"a6f0024", // feat: Add validation tooling
		"1b8afa8", // Improve explain mode output
		"e423d49", // feat: Implement YC demo critical fixes
		"8163d86", // feat: add context-aware history pruning
		"64ad48d", // Fix truncation in --explain output
	}

	if len(os.Args) > 1 {
		commits = os.Args[1:]
	}

	// Initialize LLM client
	cfg := config.Default()
	llmClient, err := llm.NewClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create LLM client: %v", err)
	}

	if !llmClient.IsEnabled() {
		log.Fatalf("LLM client not enabled. Set PHASE2_ENABLED=true and configure GEMINI_API_KEY")
	}

	// Initialize atomizer
	extractor := atomizer.NewExtractor(llmClient)

	log.Printf("Testing atomizer on %d commits...\n", len(commits))

	successCount := 0
	for i, sha := range commits {
		log.Printf("\n=== Commit %d/%d: %s ===", i+1, len(commits), sha)

		// Get commit info
		commitData, err := getCommitData(sha)
		if err != nil {
			log.Printf("❌ Failed to get commit data: %v", err)
			continue
		}

		// Extract code blocks
		result, err := extractor.ExtractCodeBlocks(ctx, *commitData)
		if err != nil {
			log.Printf("❌ Extraction failed: %v", err)
			continue
		}

		// Print results
		log.Printf("✓ Extraction successful")
		log.Printf("  Intent: %s", result.LLMIntentSummary)
		log.Printf("  Issues: %v", result.MentionedIssues)
		log.Printf("  Events: %d", len(result.ChangeEvents))

		for j, event := range result.ChangeEvents {
			log.Printf("    %d. %s: %s in %s",
				j+1, event.Behavior, event.TargetBlockName, event.TargetFile)
		}

		// Print JSON for inspection
		jsonData, _ := json.MarshalIndent(result, "", "  ")
		log.Printf("\nJSON Output:\n%s\n", string(jsonData))

		successCount++
	}

	// Calculate success rate
	successRate := float64(successCount) / float64(len(commits)) * 100
	log.Printf("\n=== Summary ===")
	log.Printf("Successful extractions: %d/%d (%.1f%%)", successCount, len(commits), successRate)

	if successRate >= 70.0 {
		log.Printf("✓ PASSED: Achieved %.1f%% success rate (target: 70%%)", successRate)
		os.Exit(0)
	} else {
		log.Printf("❌ FAILED: Only %.1f%% success rate (target: 70%%)", successRate)
		os.Exit(1)
	}
}

// getCommitData fetches commit information from git
func getCommitData(sha string) (*atomizer.CommitData, error) {
	// Get commit message
	msgCmd := exec.Command("git", "log", "-1", "--format=%B", sha)
	msgOut, err := msgCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	// Get author email
	emailCmd := exec.Command("git", "log", "-1", "--format=%ae", sha)
	emailOut, err := emailCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	// Get author date
	dateCmd := exec.Command("git", "log", "-1", "--format=%aI", sha)
	dateOut, err := dateCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get date: %w", err)
	}

	// Get diff
	diffCmd := exec.Command("git", "show", "--format=", sha)
	diffOut, err := diffCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	// Parse timestamp
	timestamp, err := time.Parse(time.RFC3339, strings.TrimSpace(string(dateOut)))
	if err != nil {
		timestamp = time.Now() // Fallback
	}

	return &atomizer.CommitData{
		SHA:         sha,
		Message:     strings.TrimSpace(string(msgOut)),
		DiffContent: string(diffOut),
		AuthorEmail: strings.TrimSpace(string(emailOut)),
		Timestamp:   timestamp,
	}, nil
}
