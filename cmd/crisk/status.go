package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show CodeRisk status and ingestion information",
	Long:  `Display current CodeRisk configuration and repository ingestion status.`,
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	fmt.Printf("🔍 CodeRisk Status\n")
	fmt.Printf("%s\n", strings.Repeat("═", 50))

	// Configuration info
	fmt.Printf("\n📋 Configuration:\n")
	fmt.Printf("  Storage: %s\n", cfg.Storage.Type)
	if cfg.Neo4j.URI != "" {
		fmt.Printf("  Neo4j URI: %s\n", cfg.Neo4j.URI)
	}
	if cfg.Storage.PostgresHost != "" {
		fmt.Printf("  PostgreSQL: %s:%d/%s\n", cfg.Storage.PostgresHost, cfg.Storage.PostgresPort, cfg.Storage.PostgresDB)
	}

	// Database connectivity check
	fmt.Printf("\n💾 Database Status:\n")

	// Try to connect to PostgreSQL if configured
	if cfg.Storage.PostgresHost != "" {
		fmt.Printf("  PostgreSQL: Configured\n")
		// TODO: Add actual connection test
	} else {
		fmt.Printf("  PostgreSQL: ⚠️ Not configured\n")
	}

	// Try to connect to Neo4j if configured
	if cfg.Neo4j.URI != "" {
		fmt.Printf("  Neo4j: Configured\n")
		// TODO: Add actual connection test
	} else {
		fmt.Printf("  Neo4j: ⚠️ Not configured\n")
	}

	// Repository info
	fmt.Printf("\n🔗 Repository:\n")
	// TODO: Implement git detection
	fmt.Printf("  Status: Not yet implemented\n")

	// LLM configuration
	fmt.Printf("\n🤖 LLM Integration:\n")
	if cfg.API.OpenAIKey != "" {
		fmt.Printf("  OpenAI Model: %s\n", cfg.API.OpenAIModel)
		fmt.Printf("  API Key: ✅ Configured\n")
	} else {
		fmt.Printf("  API Key: ❌ Not configured (Phase 2 disabled)\n")
	}

	fmt.Println("\n💡 Ready! Run 'crisk check <file>' to analyze changes")

	return nil
}
