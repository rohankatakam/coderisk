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

func main() {
	ctx := context.Background()

	log.Println("=== AGENT-P3C: Temporal Risk Calculator ===")
	log.Println("")

	// Get database URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	// Connect to PostgreSQL
	log.Println("Connecting to PostgreSQL...")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("✅ Connected to PostgreSQL")

	// Create LLM client
	log.Println("Initializing LLM client...")
	cfg := &config.Config{
		API: config.APIConfig{
			GeminiKey: os.Getenv("GEMINI_API_KEY"),
		},
	}

	// Enable Phase 2 for LLM
	os.Setenv("PHASE2_ENABLED", "true")

	llmClient, err := llm.NewClient(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create LLM client: %v", err)
	}

	if llmClient.IsEnabled() {
		log.Printf("✅ LLM client initialized (provider: %s)", llmClient.GetProvider())
	} else {
		log.Println("⚠️  LLM client not enabled, summaries will be skipped")
	}

	// Determine repo ID (default to 4 as specified by user)
	repoID := int64(4)
	if envRepoID := os.Getenv("REPO_ID"); envRepoID != "" {
		fmt.Sscanf(envRepoID, "%d", &repoID)
	}

	log.Printf("Processing repository ID: %d\n", repoID)
	log.Println("")

	// Create temporal calculator
	calc := risk.NewTemporalCalculator(db, nil, llmClient, repoID)

	// Step 1: Link incidents to blocks
	log.Println("Step 1: Linking incidents to code blocks via commits...")
	log.Println("  Strategy: Issue → (closed by) → Commit → (modified) → CodeBlock")
	linkedCount, err := calc.LinkIssuesViaCommits(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to link incidents: %v", err)
	}
	log.Printf("✅ Created %d incident links\n", linkedCount)
	log.Println("")

	// Step 2: Calculate incident counts
	log.Println("Step 2: Calculating incident counts for all blocks...")
	blocksUpdated, err := calc.CalculateIncidentCounts(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to calculate counts: %v", err)
	}
	log.Printf("✅ Updated %d blocks with incident counts\n", blocksUpdated)
	log.Println("")

	// Step 3: Generate temporal summaries for top blocks
	if llmClient.IsEnabled() {
		log.Println("Step 3: Generating LLM temporal summaries for high-incident blocks...")
		summaryCount, err := calc.GenerateTemporalSummaries(ctx)
		if err != nil {
			log.Printf("⚠️  Warning: Failed to generate summaries: %v", err)
		} else {
			log.Printf("✅ Generated %d temporal summaries\n", summaryCount)
		}
		log.Println("")
	} else {
		log.Println("Step 3: Skipping temporal summaries (LLM not enabled)")
		log.Println("")
	}

	// Show statistics
	log.Println("=== Final Statistics ===")
	stats, err := calc.GetIncidentStatistics(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to get statistics: %v", err)
	}

	log.Printf("Total blocks:              %d", stats["total_blocks"])
	log.Printf("Blocks with incidents:     %d", stats["blocks_with_incidents"])
	log.Printf("Blocks without incidents:  %d", stats["blocks_without_incidents"])
	log.Printf("Total unique issues:       %d", stats["total_unique_issues"])
	log.Printf("Total incident links:      %d", stats["total_incident_links"])
	log.Printf("Average incidents/block:   %.2f", stats["avg_incidents_per_block"])
	log.Printf("Max incidents/block:       %.0f", stats["max_incidents_per_block"])
	log.Println("")

	// Show top hotspots
	log.Println("=== Top 5 Incident Hotspots ===")
	topBlocks, err := calc.GetTopIncidentBlocks(ctx, 5)
	if err != nil {
		log.Fatalf("❌ Failed to get top blocks: %v", err)
	}

	if len(topBlocks) == 0 {
		log.Println("No incident hotspots found")
	} else {
		for i, block := range topBlocks {
			log.Printf("%d. %s (%s) - %d incidents",
				i+1,
				block["block_name"],
				block["file_path"],
				block["incident_count"])
		}
	}
	log.Println("")

	log.Println("=== AGENT-P3C: Complete ===")
	log.Println("Next: Run verification script to validate results")
	log.Println("  psql $DATABASE_URL -f scripts/verify_temporal_calculator.sql")
}
