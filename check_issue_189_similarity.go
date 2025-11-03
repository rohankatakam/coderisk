package main

import (
	"fmt"
	"github.com/rohankatakam/coderisk/internal/graph"
)

func main() {
	matcher := graph.NewSemanticMatcher()

	issue189 := "[BUG] Ctrl + Z = Dead"
	pr203 := "handle ctrl z"

	similarity := matcher.CalculateSimilarity(issue189, pr203)
	fmt.Printf("Issue #189 vs PR #203 similarity: %.4f (%.1f%%)\n", similarity, similarity*100)

	if similarity >= 0.10 {
		fmt.Println("✅ Would ACCEPT (similarity >= 10%)")
	} else {
		fmt.Println("❌ Would REJECT (similarity < 10% and delta > 1 hour)")
	}

	// Also check Issue #219 for comparison
	issue219 := "[BUG] prompts from subagents aren't shown"
	pr229 := "Codex 0.36.0"
	pr230 := "feat: Open source the Omnara Frontend"

	sim219_229 := matcher.CalculateSimilarity(issue219, pr229)
	sim219_230 := matcher.CalculateSimilarity(issue219, pr230)

	fmt.Printf("\nIssue #219 vs PR #229 similarity: %.4f (%.1f%%)\n", sim219_229, sim219_229*100)
	fmt.Printf("Issue #219 vs PR #230 similarity: %.4f (%.1f%%)\n", sim219_230, sim219_230*100)
}
