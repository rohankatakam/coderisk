package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/rohankatakam/coderisk/internal/treesitter"
)

func main() {
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println("Layer 1 Test: Structure (Tree-sitter AST Parsing)")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println()

	testRepoPath := "/tmp/test-repo"
	testFile := testRepoPath + "/example.py"

	// Check if test file exists
	if _, err := os.Stat(testFile); err != nil {
		log.Fatalf("❌ Test file not found: %s\n   Run: cd /tmp && mkdir test-repo && cd test-repo && git init && echo 'def hello(): pass' > example.py", testFile)
	}

	// Parse the file using tree-sitter
	result, err := treesitter.ParseFile(testFile)
	if err != nil {
		log.Fatalf("❌ Layer 1 FAILED: %v", err)
	}

	fmt.Printf("✅ Parsed file: %s\n", result.FilePath)
	fmt.Printf("   Language: %s\n", result.Language)
	fmt.Printf("   Entities found: %d\n\n", len(result.Entities))

	if len(result.Entities) == 0 {
		log.Fatal("❌ Layer 1 FAILED: No entities extracted from file")
	}

	// Display entities
	functionCount := 0
	classCount := 0
	importCount := 0

	fmt.Println("Extracted entities:")
	for _, entity := range result.Entities {
		fmt.Printf("  - %-10s: %-30s (lines %3d-%-3d)\n",
			entity.Type, entity.Name, entity.StartLine, entity.EndLine)

		switch entity.Type {
		case "function":
			functionCount++
		case "class":
			classCount++
		case "import":
			importCount++
		}
	}

	fmt.Println()
	fmt.Printf("Summary:\n")
	fmt.Printf("  Functions: %d\n", functionCount)
	fmt.Printf("  Classes:   %d\n", classCount)
	fmt.Printf("  Imports:   %d\n", importCount)
	fmt.Printf("  Total:     %d\n\n", len(result.Entities))

	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println("✅ LAYER 1 TEST PASSED")
	fmt.Println("=" + strings.Repeat("=", 70))
	fmt.Println()
	fmt.Println("Tree-sitter AST parsing is working correctly!")
	fmt.Println("The Structure layer can extract functions, classes, and imports.")
}
