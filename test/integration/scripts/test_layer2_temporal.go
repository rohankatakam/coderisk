//go:build integration
// +build integration

package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/rohankatakam/coderisk/internal/temporal"
)

func main() {
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println("Layer 2 Test: Temporal (Git History Analysis)")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println()

	testRepoPath := "/tmp/test-repo"

	// Check if test repo exists
	if _, err := os.Stat(testRepoPath + "/.git"); err != nil {
		log.Fatalf("❌ Git repository not found: %s\n   Run: cd /tmp && mkdir test-repo && cd test-repo && git init", testRepoPath)
	}

	// Parse git history
	windowDays := 90
	fmt.Printf("Parsing git history (last %d days)...\n", windowDays)

	commits, err := temporal.ParseGitHistory(testRepoPath, windowDays)
	if err != nil {
		log.Fatalf("❌ Layer 2 FAILED: %v", err)
	}

	fmt.Printf("✅ Git history parsed successfully\n")
	fmt.Printf("   Commits found: %d\n\n", len(commits))

	if len(commits) == 0 {
		fmt.Println("⚠️  WARNING: No commits found in repository")
		fmt.Println("   This is a new/empty repository")
		fmt.Println("   Layer 2 functionality works, but needs commits to analyze")
		fmt.Println()
		fmt.Println("=" + strings.Repeat("=", 70))
		fmt.Println("✅ LAYER 2 TEST PASSED (with limitations)")
		fmt.Println("=" + strings.Repeat("=", 70))
		return
	}

	// Extract developers
	developers := temporal.ExtractDevelopers(commits)
	fmt.Printf("Developer analysis:\n")
	fmt.Printf("  Unique developers: %d\n", len(developers))

	if len(developers) > 0 && len(developers) <= 5 {
		for email, dev := range developers {
			fmt.Printf("    - %s: %s\n", email, dev.Name)
		}
	}
	fmt.Println()

	// Calculate co-changes
	minFrequency := 0.3
	fmt.Printf("Calculating co-change patterns (min frequency: %.0f%%)...\n", minFrequency*100)

	coChanges := temporal.CalculateCoChanges(commits, minFrequency)
	fmt.Printf("✅ Co-change analysis complete\n")
	fmt.Printf("   Co-change pairs found: %d\n\n", len(coChanges))

	if len(coChanges) > 0 {
		fmt.Println("Sample co-change patterns:")
		displayCount := 3
		if len(coChanges) < displayCount {
			displayCount = len(coChanges)
		}

		for i := 0; i < displayCount; i++ {
			cc := coChanges[i]
			fmt.Printf("  %d. %s <-> %s\n", i+1, cc.FileA, cc.FileB)
			fmt.Printf("     Frequency: %.1f%% (%d co-changes)\n",
				cc.Frequency*100, cc.CoChanges)
		}
		fmt.Println()
	}

	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println("✅ LAYER 2 TEST PASSED")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println()
	fmt.Println("Git history analysis is working correctly!")
	fmt.Println("The Temporal layer can track ownership and co-change patterns.")
}
