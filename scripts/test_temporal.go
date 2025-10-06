package main

import (
	"fmt"
	"log"

	"github.com/coderisk/coderisk-go/internal/temporal"
)

func main() {
	// Test on current repository
	repoPath := "."

	fmt.Println("🔍 Parsing git history...")
	commits, err := temporal.ParseGitHistory(repoPath, 90)
	if err != nil {
		log.Fatalf("Failed to parse git history: %v", err)
	}

	fmt.Printf("✅ Parsed %d commits from last 90 days\n\n", len(commits))

	// Extract developers
	developers := temporal.ExtractDevelopers(commits)
	fmt.Printf("👥 Found %d developers:\n", len(developers))
	for i, dev := range developers {
		if i < 5 { // Show first 5
			fmt.Printf("   - %s (%s): %d commits\n", dev.Name, dev.Email, dev.TotalCommits)
		}
	}
	if len(developers) > 5 {
		fmt.Printf("   ... and %d more\n", len(developers)-5)
	}
	fmt.Println()

	// Calculate co-changes
	coChanges := temporal.CalculateCoChanges(commits, 0.3)
	fmt.Printf("🔗 Found %d co-change pairs (frequency >= 0.3):\n", len(coChanges))
	for i, cc := range coChanges {
		if i < 5 { // Show top 5
			fmt.Printf("   - %s <-> %s (%.2f frequency, %d co-changes)\n",
				cc.FileA, cc.FileB, cc.Frequency, cc.CoChanges)
		}
	}
	if len(coChanges) > 5 {
		fmt.Printf("   ... and %d more\n", len(coChanges)-5)
	}
	fmt.Println()

	// Calculate ownership
	ownership := temporal.CalculateOwnership(commits)
	fmt.Printf("📁 Calculated ownership for %d files\n", len(ownership))

	// Sample a few ownership records
	count := 0
	for filePath, own := range ownership {
		if count < 3 {
			fmt.Printf("   - %s: owned by %s", filePath, own.CurrentOwner)
			if own.PreviousOwner != "" {
				fmt.Printf(" (previously: %s, %d days ago)", own.PreviousOwner, own.DaysSince)
			}
			fmt.Println()
		}
		count++
	}
	if len(ownership) > 3 {
		fmt.Printf("   ... and %d more\n", len(ownership)-3)
	}

	fmt.Println("\n✨ Temporal analysis complete!")
}
