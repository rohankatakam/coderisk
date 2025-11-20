package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/diffanalyzer"
	"github.com/rohankatakam/coderisk/internal/llm"
)

func main() {
	// Parse command-line flags
	var (
		repoID      int64
		diffContent string
		diffFile    string
		outputJSON  bool
		logFile     string
	)

	flag.Int64Var(&repoID, "repo-id", 0, "Repository ID (required)")
	flag.StringVar(&diffContent, "diff", "", "Git diff content (or use --diff-file)")
	flag.StringVar(&diffFile, "diff-file", "", "Path to diff file")
	flag.BoolVar(&outputJSON, "json", false, "Output JSON instead of human-readable")
	flag.StringVar(&logFile, "log-file", "/tmp/crisk-analyze-diff.log", "Path to log file")
	flag.Parse()

	// Setup logging
	logFileHandle, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to open log file %s: %v\n", logFile, err)
		os.Exit(1)
	}
	defer logFileHandle.Close()

	logger := log.New(logFileHandle, "", log.LstdFlags|log.Lshortfile)
	logger.Println("========================================")
	logger.Printf("crisk-analyze-diff started at %s", time.Now().Format(time.RFC3339))
	logger.Println("========================================")

	// Validate required flags
	if repoID == 0 {
		fmt.Fprintln(os.Stderr, "ERROR: --repo-id is required")
		logger.Println("ERROR: --repo-id not provided")
		flag.Usage()
		os.Exit(1)
	}

	// Read diff
	var diff string
	if diffFile != "" {
		logger.Printf("Reading diff from file: %s", diffFile)
		content, err := os.ReadFile(diffFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to read diff file %s: %v\n", diffFile, err)
			logger.Printf("ERROR: Failed to read diff file: %v", err)
			os.Exit(1)
		}
		diff = string(content)
		logger.Printf("Successfully read %d bytes from diff file", len(diff))
	} else if diffContent != "" {
		diff = diffContent
		logger.Printf("Using diff from command line (%d bytes)", len(diff))
	} else {
		fmt.Fprintln(os.Stderr, "ERROR: No diff provided. Use --diff or --diff-file")
		logger.Println("ERROR: No diff provided")
		flag.Usage()
		os.Exit(1)
	}

	// Initialize clients
	ctx := context.Background()

	// PostgreSQL
	postgresDSN := os.Getenv("POSTGRES_DSN")
	if postgresDSN == "" {
		fmt.Fprintln(os.Stderr, "ERROR: POSTGRES_DSN environment variable not set")
		logger.Println("ERROR: POSTGRES_DSN not set")
		os.Exit(1)
	}
	logger.Printf("Connecting to PostgreSQL: %s", maskDSN(postgresDSN))

	db, err := sql.Open("postgres", postgresDSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to connect to PostgreSQL: %v\n", err)
		logger.Printf("ERROR: Failed to connect to PostgreSQL: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	// Test PostgreSQL connection
	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: PostgreSQL ping failed: %v\n", err)
		logger.Printf("ERROR: PostgreSQL ping failed: %v", err)
		os.Exit(1)
	}
	logger.Println("PostgreSQL connection successful")

	// Neo4j
	neo4jURI := os.Getenv("NEO4J_URI")
	neo4jPassword := os.Getenv("NEO4J_PASSWORD")
	if neo4jURI == "" || neo4jPassword == "" {
		fmt.Fprintln(os.Stderr, "ERROR: NEO4J_URI and NEO4J_PASSWORD environment variables required")
		logger.Println("ERROR: Neo4j environment variables not set")
		os.Exit(1)
	}
	logger.Printf("Connecting to Neo4j: %s", neo4jURI)

	neo4jDriver, err := neo4j.NewDriverWithContext(neo4jURI, neo4j.BasicAuth("neo4j", neo4jPassword, ""))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to create Neo4j driver: %v\n", err)
		logger.Printf("ERROR: Failed to create Neo4j driver: %v", err)
		os.Exit(1)
	}
	defer neo4jDriver.Close(ctx)

	// Test Neo4j connection
	if err := neo4jDriver.VerifyConnectivity(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Neo4j connectivity test failed: %v\n", err)
		logger.Printf("ERROR: Neo4j connectivity test failed: %v", err)
		os.Exit(1)
	}
	logger.Println("Neo4j connection successful")

	// LLM Client (Gemini) - use proper constructor
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	if geminiAPIKey == "" {
		fmt.Fprintln(os.Stderr, "ERROR: GEMINI_API_KEY environment variable not set")
		logger.Println("ERROR: GEMINI_API_KEY not set")
		os.Exit(1)
	}
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6380"
		logger.Printf("REDIS_ADDR not set, using default: %s", redisAddr)
	}

	// Create minimal config for LLM client
	cfg := &config.Config{
		API: config.APIConfig{
			Provider:    "gemini",
			GeminiKey:   geminiAPIKey,
			GeminiModel: "gemini-2.0-flash",
		},
	}

	// Set Redis environment variable (getRedisAddr checks REDIS_ADDR first)
	os.Setenv("REDIS_ADDR", redisAddr)

	// Set PHASE2_ENABLED for LLM client initialization
	os.Setenv("PHASE2_ENABLED", "true")
	os.Setenv("LLM_PROVIDER", "gemini")

	logger.Println("Initializing LLM client")
	llmClient, err := llm.NewClient(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to create LLM client: %v\n", err)
		logger.Printf("ERROR: Failed to create LLM client: %v", err)
		os.Exit(1)
	}
	logger.Println("LLM client initialized successfully")

	// Create analyzer
	logger.Println("Creating diff analyzer")
	analyzer := diffanalyzer.NewAnalyzer(neo4jDriver, db, llmClient, logger)

	// Analyze diff
	logger.Printf("Starting analysis for repo_id=%d", repoID)
	startTime := time.Now()
	evidence, err := analyzer.Analyze(ctx, repoID, diff)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Analysis failed: %v\n", err)
		logger.Printf("ERROR: Analysis failed: %v", err)
		os.Exit(1)
	}
	duration := time.Since(startTime)
	logger.Printf("Analysis completed successfully in %s", duration)

	// Output results
	if outputJSON {
		logger.Println("Outputting JSON format")
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(evidence); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to encode JSON: %v\n", err)
			logger.Printf("ERROR: JSON encoding failed: %v", err)
			os.Exit(1)
		}
	} else {
		logger.Println("Outputting human-readable format")
		printHumanReadable(evidence, logger)
	}

	logger.Printf("crisk-analyze-diff completed successfully")
	logger.Println("========================================")
}

// maskDSN masks sensitive information in DSN string
func maskDSN(dsn string) string {
	// Simple masking - replace password portion
	// This is a basic implementation; enhance as needed
	return "postgres://***:***@localhost/coderisk"
}

// printHumanReadable formats the evidence in a human-readable way
func printHumanReadable(evidence *diffanalyzer.RiskEvidenceJSON, logger *log.Logger) {
	fmt.Printf("\n")
	fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║              CodeRisk Diff Analysis Results                 ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n")
	fmt.Printf("\n")
	fmt.Printf("Overall Risk: %s\n", getRiskEmoji(evidence.RiskSummary), evidence.RiskSummary)
	fmt.Printf("Blocks Analyzed: %d\n", len(evidence.Blocks))
	fmt.Printf("\n")

	if len(evidence.Blocks) == 0 {
		fmt.Println("No code blocks found in diff.")
		return
	}

	for i, block := range evidence.Blocks {
		fmt.Printf("─────────────────────────────────────────────────────────────\n")
		fmt.Printf("Block %d/%d: %s\n", i+1, len(evidence.Blocks), block.Name)
		fmt.Printf("File: %s\n", block.File)
		fmt.Printf("Change Type: %s\n", block.ChangeType)
		fmt.Printf("Match Type: %s\n", block.MatchType)
		fmt.Printf("\n")

		if block.MatchType == "new_function" {
			fmt.Println("  ℹ️  New function (not yet indexed)")
			continue
		}

		// Temporal Risk
		if block.Risks.Temporal != nil {
			fmt.Printf("  🕒 TEMPORAL RISK:\n")
			fmt.Printf("     Incident Count: %d\n", block.Risks.Temporal.IncidentCount)
			if block.Risks.Temporal.LastIncidentDate != nil {
				fmt.Printf("     Last Incident: %s\n", block.Risks.Temporal.LastIncidentDate.Format("2006-01-02"))
			}
			if len(block.Risks.Temporal.LinkedIssues) > 0 {
				fmt.Printf("     Linked Issues:\n")
				for _, issue := range block.Risks.Temporal.LinkedIssues {
					fmt.Printf("       - #%d: %s (version: %s)\n", issue.Number, issue.Title, issue.VersionAffected)
				}
			}
			if len(block.Risks.Temporal.AffectedVersions) > 0 {
				fmt.Printf("     Affected Versions: %v\n", block.Risks.Temporal.AffectedVersions)
			}
			fmt.Printf("\n")
		}

		// Ownership Risk
		if block.Risks.Ownership != nil {
			fmt.Printf("  👤 OWNERSHIP RISK:\n")
			fmt.Printf("     Status: %s\n", block.Risks.Ownership.Status)
			fmt.Printf("     Original Author: %s\n", block.Risks.Ownership.OriginalAuthor)
			fmt.Printf("     Last Modifier: %s\n", block.Risks.Ownership.LastModifier)
			fmt.Printf("     Days Since Modified: %d\n", block.Risks.Ownership.DaysSinceModified)
			if block.Risks.Ownership.BusFactorWarning {
				fmt.Printf("     ⚠️  Bus Factor Warning: Low contributor count\n")
			}
			if len(block.Risks.Ownership.TopContributors) > 0 {
				fmt.Printf("     Top Contributors:\n")
				for _, contrib := range block.Risks.Ownership.TopContributors {
					fmt.Printf("       - %s (%d modifications)\n", contrib.Email, contrib.ModificationCount)
				}
			}
			fmt.Printf("\n")
		}

		// Coupling Risk
		if block.Risks.Coupling != nil && len(block.Risks.Coupling.CoupledBlocks) > 0 {
			fmt.Printf("  🔗 COUPLING RISK:\n")
			fmt.Printf("     Coupling Score: %.2f\n", block.Risks.Coupling.Score)
			fmt.Printf("     Coupled Blocks:\n")
			for _, coupled := range block.Risks.Coupling.CoupledBlocks {
				fmt.Printf("       - %s (%s)\n", coupled.Name, coupled.File)
				fmt.Printf("         Co-change Rate: %.1f%%, Count: %d, Dynamic Score: %.2f\n",
					coupled.CouplingRate*100, coupled.CoChangeCount, coupled.DynamicCouplingScore)
			}
			fmt.Printf("\n")
		}

		// Change History
		if block.Risks.ChangeHistory != nil && len(block.Risks.ChangeHistory.RecentChanges) > 0 {
			fmt.Printf("  📜 RECENT CHANGES: (showing first 5)\n")
			for j, change := range block.Risks.ChangeHistory.RecentChanges {
				if j >= 5 {
					break
				}
				fmt.Printf("     %d. [%s] %s\n", j+1, change.SHA[:7], change.Message)
				fmt.Printf("        Author: %s, Date: %s\n", change.Author, change.CommittedAt.Format("2006-01-02"))
			}
			fmt.Printf("\n")
		}
	}

	fmt.Printf("─────────────────────────────────────────────────────────────\n")
	fmt.Printf("\n")
	logger.Println("Human-readable output completed")
}

// getRiskEmoji returns an emoji for the risk level
func getRiskEmoji(risk string) string {
	switch risk {
	case "CRITICAL":
		return "🔴"
	case "HIGH":
		return "🟠"
	case "MEDIUM":
		return "🟡"
	case "LOW":
		return "🟢"
	default:
		return "⚪"
	}
}
