package main

import (
	"context"
	"fmt"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/llm"
	"github.com/rohankatakam/coderisk/internal/risk"
	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "crisk-index-ownership",
	Short: "Calculate ownership signals for CodeBlocks",
	Long: `crisk-index-ownership - Microservice 5: Ownership Risk Indexer

Calculates ownership signals for each CodeBlock in the repository.

Features:
  • Last modifier calculation
  • Staleness (days since last edit)
  • Familiarity map (edit counts per developer)
  • Original author tracking

Output: CodeBlocks with ownership properties populated`,
	Version: Version,
	RunE:    runIndexOwnership,
}

var (
	repoID  int64
	verbose bool
)

func init() {
	rootCmd.Flags().Int64Var(&repoID, "repo-id", 0, "Repository ID from Neo4j (required)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	rootCmd.MarkFlagRequired("repo-id")

	rootCmd.SetVersionTemplate(`crisk-index-ownership {{.Version}}
Build time: ` + BuildTime + `
Git commit: ` + GitCommit + `
`)
}

func runIndexOwnership(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	fmt.Printf("🚀 crisk-index-ownership - Ownership Indexing Service\n")
	fmt.Printf("   Repository ID: %d\n", repoID)
	fmt.Println()

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get Neo4j connection details
	neoURI := os.Getenv("NEO4J_URI")
	neoPassword := os.Getenv("NEO4J_PASSWORD")
	if neoURI == "" {
		neoURI = cfg.Neo4j.URI
	}
	if neoPassword == "" {
		neoPassword = cfg.Neo4j.Password
	}

	// Connect to Neo4j
	fmt.Printf("[1/3] Connecting to Neo4j...\n")
	driver, err := neo4j.NewDriverWithContext(
		neoURI,
		neo4j.BasicAuth(cfg.Neo4j.User, neoPassword, ""),
	)
	if err != nil {
		return fmt.Errorf("failed to create Neo4j driver: %w", err)
	}
	defer driver.Close(ctx)

	if err := driver.VerifyConnectivity(ctx); err != nil {
		return fmt.Errorf("failed to connect to Neo4j: %w", err)
	}
	fmt.Printf("  ✓ Connected to Neo4j\n\n")

	// Create LLM client (optional for ownership calculations)
	fmt.Printf("[2/3] Initializing LLM client...\n")
	llmClient, err := llm.NewClient(ctx, cfg)
	if err != nil {
		fmt.Printf("  ⚠️  LLM client creation failed: %v\n", err)
		fmt.Printf("  → Continuing without LLM (semantic importance will be skipped)\n\n")
		llmClient = nil
	} else if llmClient.IsEnabled() {
		fmt.Printf("  ✓ LLM client ready (%s)\n\n", llmClient.GetProvider())
	} else {
		fmt.Printf("  ⚠️  LLM client disabled (semantic importance will be skipped)\n\n")
		llmClient = nil
	}

	// Check how many blocks exist
	blockCount, err := countCodeBlocks(ctx, driver, cfg.Neo4j.Database, repoID)
	if err != nil {
		return fmt.Errorf("failed to count blocks: %w", err)
	}
	fmt.Printf("  Found %d CodeBlock nodes\n", blockCount)

	if blockCount == 0 {
		return fmt.Errorf("no CodeBlock nodes found for repo_id=%d\n\nRun 'crisk-atomize' first", repoID)
	}

	// Create ownership indexer
	indexer := risk.NewBlockIndexer(driver, cfg.Neo4j.Database, llmClient)

	// Run ownership calculations
	fmt.Printf("\n[3/3] Calculating ownership properties...\n")
	result, err := indexer.IndexOwnership(ctx, repoID)
	if err != nil {
		return fmt.Errorf("ownership indexing failed: %w", err)
	}

	// Print results
	fmt.Printf("\n📊 Ownership Indexing Results:\n")
	fmt.Printf("   Repository ID: %d\n", result.RepoID)
	fmt.Printf("   Total Duration: %v\n", result.TotalDuration)
	fmt.Printf("\n   Phase Results:\n")
	fmt.Printf("     - Original Authors Set: %d blocks\n", result.OriginalAuthors)
	fmt.Printf("     - Last Modifiers Set: %d blocks\n", result.LastModifiers)
	fmt.Printf("     - Familiarity Maps Set: %d blocks\n", result.FamiliarityMaps)
	fmt.Printf("\n   Verification:\n")
	fmt.Printf("     - Incomplete Blocks: %d\n", result.IncompleteBlocks)

	if result.IncompleteBlocks > 0 {
		fmt.Printf("\n⚠️  WARNING: %d blocks are missing ownership properties\n", result.IncompleteBlocks)
	} else {
		fmt.Printf("\n✅ SUCCESS: All %d blocks have complete ownership properties\n", blockCount)
	}

	// Get detailed stats
	stats, err := indexer.GetIndexingStats(ctx, repoID)
	if err != nil {
		fmt.Printf("\n⚠️  Warning: Failed to get stats: %v\n", err)
	} else {
		fmt.Printf("\n📈 Detailed Statistics:\n")
		fmt.Printf("   Total Blocks: %d\n", stats["total_blocks"])
		fmt.Printf("   With Original Author: %d\n", stats["with_original_author"])
		fmt.Printf("   With Last Modifier: %d\n", stats["with_last_modifier"])
		fmt.Printf("   With Familiarity Map: %d\n", stats["with_familiarity_map"])
		if stats["missing_original_author"] > 0 {
			fmt.Printf("   Missing Original Author: %d\n", stats["missing_original_author"])
		}
		if stats["missing_last_modifier"] > 0 {
			fmt.Printf("   Missing Last Modifier: %d\n", stats["missing_last_modifier"])
		}
		if stats["missing_familiarity_map"] > 0 {
			fmt.Printf("   Missing Familiarity Map: %d\n", stats["missing_familiarity_map"])
		}
	}

	fmt.Printf("\n✅ Ownership indexing complete\n")
	fmt.Printf("\n🚀 Next: crisk-index-coupling --repo-id %d\n", repoID)

	return nil
}

func countCodeBlocks(ctx context.Context, driver neo4j.DriverWithContext, database string, repoID int64) (int, error) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: database,
	})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
			MATCH (b:CodeBlock {repo_id: $repoID})
			RETURN count(b) AS count
		`, map[string]any{
			"repoID": repoID,
		})
		if err != nil {
			return nil, err
		}

		record, err := res.Single(ctx)
		if err != nil {
			return nil, err
		}

		count, _ := record.Get("count")
		return count, nil
	})

	if err != nil {
		return 0, err
	}

	return int(result.(int64)), nil
}
