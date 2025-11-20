package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
		repoPath    string
		repoID      int64
		outputJSON  bool
		logFile     string
		staged      bool
		showHelp    bool
	)

	flag.StringVar(&repoPath, "repo-path", "", "Path to git repository (required)")
	flag.Int64Var(&repoID, "repo-id", 0, "Repository ID in database (required)")
	flag.BoolVar(&outputJSON, "json", false, "Output JSON instead of human-readable")
	flag.StringVar(&logFile, "log-file", "/tmp/crisk-semantic-check.log", "Path to log file")
	flag.BoolVar(&staged, "staged", false, "Analyze only staged changes (default: all uncommitted)")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.Parse()

	if showHelp {
		printHelp()
		os.Exit(0)
	}

	// Setup logging with unbuffered output for real-time monitoring
	logFileHandle, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to open log file %s: %v\n", logFile, err)
		os.Exit(1)
	}
	defer logFileHandle.Close()

	// Create logger with synchronous writes
	logger := log.New(logFileHandle, "", log.LstdFlags|log.Lshortfile)
	logger.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	// Print log file location for user to monitor
	fmt.Printf("📝 Logging to: %s\n", logFile)
	fmt.Printf("   Monitor with: tail -f %s\n\n", logFile)

	logger.Println("========================================")
	logger.Printf("crisk-semantic-check started at %s", time.Now().Format(time.RFC3339))
	logger.Println("========================================")

	// Validate required flags
	if repoPath == "" {
		fmt.Fprintln(os.Stderr, "ERROR: --repo-path is required")
		logger.Println("ERROR: --repo-path not provided")
		flag.Usage()
		os.Exit(1)
	}

	if repoID == 0 {
		fmt.Fprintln(os.Stderr, "ERROR: --repo-id is required")
		logger.Println("ERROR: --repo-id not provided")
		flag.Usage()
		os.Exit(1)
	}

	// Resolve absolute path
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to resolve absolute path: %v\n", err)
		logger.Printf("ERROR: Failed to resolve path: %v", err)
		os.Exit(1)
	}
	logger.Printf("Repository path: %s", absRepoPath)

	// Validate git repository
	if !isGitRepository(absRepoPath) {
		fmt.Fprintf(os.Stderr, "ERROR: %s is not a git repository\n", absRepoPath)
		logger.Printf("ERROR: Not a git repository: %s", absRepoPath)
		os.Exit(1)
	}
	logger.Println("Valid git repository confirmed")

	// Extract git diff
	logger.Println("Extracting git diff...")
	diff, changedFiles, err := extractGitDiff(absRepoPath, staged, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to extract git diff: %v\n", err)
		logger.Printf("ERROR: Git diff extraction failed: %v", err)
		os.Exit(1)
	}

	if diff == "" {
		fmt.Println("No changes detected in repository.")
		logger.Println("No changes detected")
		os.Exit(0)
	}

	logger.Printf("Extracted diff: %d bytes, %d files changed", len(diff), len(changedFiles))
	logger.Printf("Changed files: %v", changedFiles)
	logger.Println("========== RAW DIFF CONTENT ==========")
	logger.Println(diff)
	logger.Println("========== END DIFF CONTENT ==========")

	// Initialize clients
	ctx := context.Background()

	// PostgreSQL
	postgresDSN := os.Getenv("POSTGRES_DSN")
	if postgresDSN == "" {
		fmt.Fprintln(os.Stderr, "ERROR: POSTGRES_DSN environment variable not set")
		logger.Println("ERROR: POSTGRES_DSN not set")
		os.Exit(1)
	}
	logger.Printf("Connecting to PostgreSQL")

	db, err := sql.Open("postgres", postgresDSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to connect to PostgreSQL: %v\n", err)
		logger.Printf("ERROR: PostgreSQL connection failed: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: PostgreSQL ping failed: %v\n", err)
		logger.Printf("ERROR: PostgreSQL ping failed: %v", err)
		os.Exit(1)
	}
	logger.Println("PostgreSQL connected")

	// Neo4j
	neo4jURI := os.Getenv("NEO4J_URI")
	neo4jPassword := os.Getenv("NEO4J_PASSWORD")
	if neo4jURI == "" || neo4jPassword == "" {
		fmt.Fprintln(os.Stderr, "ERROR: NEO4J_URI and NEO4J_PASSWORD required")
		logger.Println("ERROR: Neo4j env vars not set")
		os.Exit(1)
	}
	logger.Printf("Connecting to Neo4j")

	neo4jDriver, err := neo4j.NewDriverWithContext(neo4jURI, neo4j.BasicAuth("neo4j", neo4jPassword, ""))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to create Neo4j driver: %v\n", err)
		logger.Printf("ERROR: Neo4j driver creation failed: %v", err)
		os.Exit(1)
	}
	defer neo4jDriver.Close(ctx)

	if err := neo4jDriver.VerifyConnectivity(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Neo4j connectivity failed: %v\n", err)
		logger.Printf("ERROR: Neo4j connectivity failed: %v", err)
		os.Exit(1)
	}
	logger.Println("Neo4j connected")

	// LLM Client - use proper constructor
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	if geminiAPIKey == "" {
		fmt.Fprintln(os.Stderr, "ERROR: GEMINI_API_KEY not set")
		logger.Println("ERROR: GEMINI_API_KEY not set")
		os.Exit(1)
	}
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6380"
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
		logger.Printf("ERROR: LLM client creation failed: %v", err)
		os.Exit(1)
	}
	logger.Println("LLM client initialized successfully")

	// Create analyzer
	logger.Println("Creating diff analyzer")
	analyzer := diffanalyzer.NewAnalyzer(neo4jDriver, db, llmClient, logger)

	// Analyze diff
	fmt.Printf("\n🔍 Analyzing changes in %s...\n\n", absRepoPath)
	logger.Printf("Starting analysis for repo_id=%d", repoID)
	startTime := time.Now()

	evidence, err := analyzer.Analyze(ctx, repoID, diff)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Analysis failed: %v\n", err)
		logger.Printf("ERROR: Analysis failed: %v", err)
		os.Exit(1)
	}

	duration := time.Since(startTime)
	logger.Printf("Analysis completed in %s", duration)

	// Output results
	if outputJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(evidence); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: JSON encoding failed: %v\n", err)
			logger.Printf("ERROR: JSON encoding failed: %v", err)
			os.Exit(1)
		}
	} else {
		printSemanticGitOutput(evidence, changedFiles, duration)
	}

	logger.Println("crisk-semantic-check completed successfully")
	logger.Println("========================================")
}

// isGitRepository checks if the path is a git repository
func isGitRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// extractGitDiff extracts the git diff from the repository
func extractGitDiff(repoPath string, staged bool, logger *log.Logger) (string, []string, error) {
	// Change to repository directory
	originalDir, err := os.Getwd()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(repoPath); err != nil {
		return "", nil, fmt.Errorf("failed to change to repo directory: %w", err)
	}

	logger.Printf("Changed to repository directory: %s", repoPath)

	// Get list of changed files
	var fileCmd *exec.Cmd
	if staged {
		fileCmd = exec.Command("git", "diff", "--cached", "--name-only")
		logger.Println("Executing: git diff --cached --name-only")
	} else {
		fileCmd = exec.Command("git", "diff", "HEAD", "--name-only")
		logger.Println("Executing: git diff HEAD --name-only")
	}

	filesOutput, err := fileCmd.CombinedOutput()
	if err != nil {
		logger.Printf("ERROR: git diff --name-only failed: %v, output: %s", err, string(filesOutput))
		return "", nil, fmt.Errorf("git diff --name-only failed: %w\nOutput: %s", err, string(filesOutput))
	}

	filesStr := strings.TrimSpace(string(filesOutput))
	var changedFiles []string
	if filesStr != "" {
		changedFiles = strings.Split(filesStr, "\n")
	}

	logger.Printf("Changed files: %d", len(changedFiles))

	// Get full diff
	var diffCmd *exec.Cmd
	if staged {
		diffCmd = exec.Command("git", "diff", "--cached")
		logger.Println("Executing: git diff --cached")
	} else {
		diffCmd = exec.Command("git", "diff", "HEAD")
		logger.Println("Executing: git diff HEAD")
	}

	diffOutput, err := diffCmd.CombinedOutput()
	if err != nil {
		logger.Printf("ERROR: git diff failed: %v, output: %s", err, string(diffOutput))
		return "", nil, fmt.Errorf("git diff failed: %w\nOutput: %s", err, string(diffOutput))
	}

	diff := string(diffOutput)
	logger.Printf("Extracted diff: %d bytes", len(diff))

	return diff, changedFiles, nil
}

// printSemanticGitOutput formats output in "Git on Steroids" style
func printSemanticGitOutput(evidence *diffanalyzer.RiskEvidenceJSON, changedFiles []string, duration time.Duration) {
	fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║           CodeRisk: Git on Steroids - Semantic Check        ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n")
	fmt.Printf("\n")

	// Summary
	riskEmoji := getRiskEmoji(evidence.RiskSummary)
	fmt.Printf("%s Overall Risk: %s\n", riskEmoji, evidence.RiskSummary)
	fmt.Printf("⏱️  Analysis Time: %s\n", duration.Round(time.Millisecond))
	fmt.Printf("📁 Files Changed: %d\n", len(changedFiles))
	fmt.Printf("🔢 Code Blocks: %d\n", len(evidence.Blocks))
	fmt.Printf("\n")

	if len(evidence.Blocks) == 0 {
		fmt.Println("✅ No risky changes detected.")
		return
	}

	// Group blocks by risk level
	critical := []diffanalyzer.BlockRisk{}
	high := []diffanalyzer.BlockRisk{}
	medium := []diffanalyzer.BlockRisk{}
	low := []diffanalyzer.BlockRisk{}

	for _, block := range evidence.Blocks {
		level := calculateBlockRiskLevel(block)
		switch level {
		case "CRITICAL":
			critical = append(critical, block)
		case "HIGH":
			high = append(high, block)
		case "MEDIUM":
			medium = append(medium, block)
		default:
			low = append(low, block)
		}
	}

	// Print by risk level
	if len(critical) > 0 {
		fmt.Printf("🔴 CRITICAL RISK BLOCKS (%d):\n", len(critical))
		fmt.Printf("───────────────────────────────────────────────────────────────\n")
		for _, block := range critical {
			printBlockSummary(block)
		}
		fmt.Printf("\n")
	}

	if len(high) > 0 {
		fmt.Printf("🟠 HIGH RISK BLOCKS (%d):\n", len(high))
		fmt.Printf("───────────────────────────────────────────────────────────────\n")
		for _, block := range high {
			printBlockSummary(block)
		}
		fmt.Printf("\n")
	}

	if len(medium) > 0 {
		fmt.Printf("🟡 MEDIUM RISK BLOCKS (%d):\n", len(medium))
		fmt.Printf("───────────────────────────────────────────────────────────────\n")
		for _, block := range medium {
			printBlockSummary(block)
		}
		fmt.Printf("\n")
	}

	// Recommendations
	fmt.Printf("💡 RECOMMENDATIONS:\n")
	fmt.Printf("───────────────────────────────────────────────────────────────\n")
	if len(critical) > 0 {
		fmt.Printf("  ⚠️  CRITICAL: Review incident history before committing\n")
	}
	if len(high) > 0 {
		fmt.Printf("  ⚠️  HIGH: Consider additional testing and code review\n")
	}
	for _, block := range evidence.Blocks {
		if block.Risks.Ownership != nil && block.Risks.Ownership.BusFactorWarning {
			fmt.Printf("  👥 Bus Factor: Add knowledge sharing for %s\n", block.Name)
			break
		}
	}
	for _, block := range evidence.Blocks {
		if block.Risks.Coupling != nil && len(block.Risks.Coupling.CoupledBlocks) > 0 {
			fmt.Printf("  🔗 Coupling: Test coupled blocks: ")
			names := []string{}
			for _, cb := range block.Risks.Coupling.CoupledBlocks {
				names = append(names, cb.Name)
			}
			fmt.Printf("%s\n", strings.Join(names, ", "))
			break
		}
	}
	fmt.Printf("\n")
}

// printBlockSummary prints a one-line summary of a block
func printBlockSummary(block diffanalyzer.BlockRisk) {
	incidentCount := 0
	if block.Risks.Temporal != nil {
		incidentCount = block.Risks.Temporal.IncidentCount
	}

	staleDays := 0
	if block.Risks.Ownership != nil {
		staleDays = block.Risks.Ownership.DaysSinceModified
	}

	couplingCount := 0
	if block.Risks.Coupling != nil {
		couplingCount = len(block.Risks.Coupling.CoupledBlocks)
	}

	fmt.Printf("  • %s (%s)\n", block.Name, block.File)
	fmt.Printf("    Incidents: %d | Staleness: %d days | Coupled blocks: %d\n",
		incidentCount, staleDays, couplingCount)

	if block.Risks.Temporal != nil && len(block.Risks.Temporal.LinkedIssues) > 0 {
		issues := []string{}
		for _, issue := range block.Risks.Temporal.LinkedIssues {
			issues = append(issues, fmt.Sprintf("#%d", issue.Number))
		}
		fmt.Printf("    Issues: %s\n", strings.Join(issues, ", "))
	}
	fmt.Printf("\n")
}

// calculateBlockRiskLevel determines risk level for a block
func calculateBlockRiskLevel(block diffanalyzer.BlockRisk) string {
	score := 0.0

	if block.Risks.Temporal != nil {
		if block.Risks.Temporal.IncidentCount >= 5 {
			score += 40
		} else if block.Risks.Temporal.IncidentCount >= 3 {
			score += 30
		} else if block.Risks.Temporal.IncidentCount >= 1 {
			score += 15
		}
	}

	if block.Risks.Ownership != nil {
		if block.Risks.Ownership.Status == "STALE" {
			score += 20
		}
		if block.Risks.Ownership.BusFactorWarning {
			score += 10
		}
	}

	if block.Risks.Coupling != nil {
		if block.Risks.Coupling.Score > 10 {
			score += 30
		} else if block.Risks.Coupling.Score > 5 {
			score += 20
		} else if block.Risks.Coupling.Score > 0 {
			score += 10
		}
	}

	if score >= 70 {
		return "CRITICAL"
	} else if score >= 50 {
		return "HIGH"
	} else if score >= 30 {
		return "MEDIUM"
	}
	return "LOW"
}

// getRiskEmoji returns emoji for risk level
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

// printHelp prints usage information
func printHelp() {
	fmt.Println("crisk-semantic-check - Git on Steroids: Semantic Code Analysis")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  crisk-semantic-check --repo-path <path> --repo-id <id> [options]")
	fmt.Println()
	fmt.Println("REQUIRED FLAGS:")
	fmt.Println("  --repo-path string    Path to git repository")
	fmt.Println("  --repo-id int         Repository ID in CodeRisk database")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  --staged              Analyze only staged changes (default: all uncommitted)")
	fmt.Println("  --json                Output JSON instead of human-readable format")
	fmt.Println("  --log-file string     Path to log file (default: /tmp/crisk-semantic-check.log)")
	fmt.Println("  --help                Show this help message")
	fmt.Println()
	fmt.Println("ENVIRONMENT VARIABLES:")
	fmt.Println("  POSTGRES_DSN          PostgreSQL connection string (required)")
	fmt.Println("  NEO4J_URI             Neo4j connection URI (required)")
	fmt.Println("  NEO4J_PASSWORD        Neo4j password (required)")
	fmt.Println("  GEMINI_API_KEY        Google Gemini API key (required)")
	fmt.Println("  REDIS_ADDR            Redis address for rate limiting (default: localhost:6380)")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  # Analyze all uncommitted changes")
	fmt.Println("  crisk-semantic-check --repo-path /path/to/repo --repo-id 14")
	fmt.Println()
	fmt.Println("  # Analyze only staged changes")
	fmt.Println("  crisk-semantic-check --repo-path /path/to/repo --repo-id 14 --staged")
	fmt.Println()
	fmt.Println("  # Output JSON for tooling integration")
	fmt.Println("  crisk-semantic-check --repo-path /path/to/repo --repo-id 14 --json")
	fmt.Println()
}
