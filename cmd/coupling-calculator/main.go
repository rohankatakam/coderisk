package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/llm"
	"github.com/rohankatakam/coderisk/internal/risk"
)

// Example usage of the Coupling Risk Calculator
// Reference: AGENT-P3B Coupling Risk Calculator
//
// Usage:
//   export DATABASE_URL="postgresql://user:pass@localhost:5432/coderisk"
//   export GEMINI_API_KEY="your-api-key"  # Optional, for explanations
//   export PHASE2_ENABLED="true"          # Optional, for LLM features
//   go run cmd/coupling-calculator/main.go <repo_id>

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: coupling-calculator <repo_id>")
		fmt.Println()
		fmt.Println("Environment variables:")
		fmt.Println("  DATABASE_URL  - PostgreSQL connection string (required)")
		fmt.Println("  GEMINI_API_KEY - API key for LLM explanations (optional)")
		fmt.Println("  PHASE2_ENABLED - Set to 'true' to enable LLM features (optional)")
		os.Exit(1)
	}

	repoIDStr := os.Args[1]
	var repoID int64
	if _, err := fmt.Sscanf(repoIDStr, "%d", &repoID); err != nil {
		log.Fatalf("Invalid repo_id: %s", repoIDStr)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	ctx := context.Background()

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Printf("✓ Connected to database\n")

	// Initialize LLM client (optional)
	var llmClient *llm.Client
	if os.Getenv("PHASE2_ENABLED") == "true" {
		cfg := &config.Config{
			API: config.APIConfig{
				GeminiKey: os.Getenv("GEMINI_API_KEY"),
			},
		}
		llmClient, err = llm.NewClient(ctx, cfg)
		if err != nil {
			log.Printf("Warning: Failed to initialize LLM client: %v", err)
			log.Printf("Continuing without LLM explanations...")
		} else if llmClient.IsEnabled() {
			fmt.Printf("✓ LLM client enabled (%s)\n", llmClient.GetProvider())
		}
	}

	// Create coupling calculator
	calc := risk.NewCouplingCalculator(db, llmClient, repoID)
	fmt.Printf("✓ Coupling calculator initialized for repo_id=%d\n\n", repoID)

	// Step 1: Calculate co-changes
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("STEP 1: Calculating Co-Change Relationships")
	fmt.Println("═══════════════════════════════════════════════════════")
	edgesCreated, err := calc.CalculateCoChanges(ctx)
	if err != nil {
		log.Fatalf("Failed to calculate co-changes: %v", err)
	}
	fmt.Printf("\n✓ Created %d co-change edges (rate >= 50%%)\n\n", edgesCreated)

	// Step 2: Get statistics
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("STEP 2: Co-Change Statistics")
	fmt.Println("═══════════════════════════════════════════════════════")
	stats, err := calc.GetCoChangeStatistics(ctx)
	if err != nil {
		log.Fatalf("Failed to get statistics: %v", err)
	}

	fmt.Printf("Total Edges:         %d\n", stats["total_edges"])
	if stats["total_edges"].(int) > 0 {
		fmt.Printf("Min Coupling Rate:   %.1f%%\n", stats["min_rate"].(float64)*100)
		fmt.Printf("Max Coupling Rate:   %.1f%%\n", stats["max_rate"].(float64)*100)
		fmt.Printf("Avg Coupling Rate:   %.1f%%\n", stats["avg_rate"].(float64)*100)
		fmt.Printf("\nCoupling Distribution:\n")
		fmt.Printf("  High (≥75%%):       %d edges\n", stats["high_coupling_count"])
		fmt.Printf("  Medium (50-75%%):   %d edges\n", stats["medium_coupling_count"])
	}

	// Step 3: Get top coupled blocks
	fmt.Println("\n═══════════════════════════════════════════════════════")
	fmt.Println("STEP 3: Top 10 Most Coupled Blocks")
	fmt.Println("═══════════════════════════════════════════════════════")
	topBlocks, err := calc.GetTopCoupledBlocks(ctx, 10)
	if err != nil {
		log.Fatalf("Failed to get top coupled blocks: %v", err)
	}

	if len(topBlocks) == 0 {
		fmt.Println("No coupled blocks found.")
	} else {
		for i, block := range topBlocks {
			fmt.Printf("\n%d. %s (%s)\n", i+1, block["block_name"], block["block_type"])
			fmt.Printf("   File: %s\n", block["file_path"])
			fmt.Printf("   Couplings: %d edges\n", block["total_couplings"])
			fmt.Printf("   Avg Rate: %.1f%%\n", block["avg_coupling_rate"].(float64)*100)
		}
	}

	fmt.Println("\n═══════════════════════════════════════════════════════")
	fmt.Println("✓ Analysis Complete")
	fmt.Println("═══════════════════════════════════════════════════════")
}
