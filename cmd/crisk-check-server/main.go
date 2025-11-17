package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	mcpinternal "github.com/rohankatakam/coderisk/internal/mcp"
	"github.com/rohankatakam/coderisk/internal/mcp/tools"
	"github.com/rohankatakam/coderisk/internal/config"
	"github.com/rohankatakam/coderisk/internal/llm"
	bolt "go.etcd.io/bbolt"
)

func main() {
	// Redirect logs to file to avoid interfering with MCP stdio protocol
	logFile, err := os.OpenFile("/tmp/crisk-mcp-server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	ctx := context.Background()

	// 1. Get environment variables
	neo4jURI := getEnvOrDefault("NEO4J_URI", "bolt://localhost:7688")
	neo4jPassword := getEnvOrDefault("NEO4J_PASSWORD", "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123")
	postgresDSN := getEnvOrDefault("POSTGRES_DSN", "postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable")
	geminiAPIKey := os.Getenv("GEMINI_API_KEY") // Optional: for diff-based analysis

	// 2. Connect to Neo4j
	driver, err := neo4j.NewDriverWithContext(neo4jURI, neo4j.BasicAuth("neo4j", neo4jPassword, ""))
	if err != nil {
		log.Fatalf("Failed to connect to Neo4j at %s: %v", neo4jURI, err)
	}
	defer driver.Close(ctx)

	// Verify Neo4j connection
	if err := driver.VerifyConnectivity(ctx); err != nil {
		log.Fatalf("Neo4j connectivity check failed: %v", err)
	}
	log.Println("✅ Connected to Neo4j")

	// 3. Connect to PostgreSQL
	pgPool, err := pgxpool.New(ctx, postgresDSN)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgPool.Close()

	// Verify PostgreSQL connection
	if err := pgPool.Ping(ctx); err != nil {
		log.Fatalf("PostgreSQL ping failed: %v", err)
	}
	log.Println("✅ Connected to PostgreSQL")

	// 4. Open bbolt cache
	log.Println("Opening cache database...")
	cacheDB, err := bolt.Open("/tmp/crisk-mcp-cache.db", 0600, nil)
	if err != nil {
		log.Fatalf("Failed to open cache: %v", err)
	}
	defer cacheDB.Close()
	log.Println("✅ Cache database opened")
	log.Println("✅ Cache initialized")

	// 5. Create graph client
	log.Println("Creating graph client...")
	graphClient := mcpinternal.NewLocalGraphClient(driver, pgPool)
	log.Println("✅ Graph client created")

	// 6. Create identity resolver
	log.Println("Creating identity resolver...")
	resolver := mcpinternal.NewIdentityResolver(cacheDB)
	log.Println("✅ Identity resolver created")

	// 6.5. Create diff atomizer (optional, requires GEMINI_API_KEY)
	var diffAtomizer *mcpinternal.DiffAtomizer
	if geminiAPIKey != "" {
		log.Println("Creating LLM client for diff analysis...")
		// Create a minimal config with Gemini API key
		// Set PHASE2_ENABLED temporarily to allow LLM client creation
		os.Setenv("PHASE2_ENABLED", "true")
		cfg := &config.Config{
			API: config.APIConfig{
				GeminiKey:   geminiAPIKey,
				GeminiModel: "gemini-2.0-flash",
			},
		}
		llmClient, err := llm.NewClient(ctx, cfg)
		if err != nil {
			log.Printf("⚠️  Failed to create LLM client (diff analysis disabled): %v", err)
		} else if llmClient.IsEnabled() {
			diffAtomizer = mcpinternal.NewDiffAtomizer(llmClient)
			log.Println("✅ Diff atomizer created (supports diff-based analysis)")
		} else {
			log.Println("⚠️  LLM client not enabled (diff analysis disabled)")
		}
	} else {
		log.Println("⚠️  GEMINI_API_KEY not set - diff-based analysis disabled (file-based analysis still works)")
	}

	// 7. Create MCP server using official SDK
	log.Println("Initializing MCP server...")
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "crisk-check-server",
		Version: "0.1.0",
	}, nil)
	log.Println("✅ MCP server initialized")

	// 8. Add the risk summary tool using the generic AddTool
	type ToolArgs struct {
		FilePath         string   `json:"file_path,omitempty" jsonschema:"path to the file to analyze (optional - tool will auto-detect uncommitted changes if available)"`
		DiffContent      string   `json:"diff_content,omitempty" jsonschema:"optional diff content for uncommitted changes (if not provided, tool will check for uncommitted changes automatically)"`
		RepoRoot         string   `json:"repo_root,omitempty" jsonschema:"optional repository root path for resolving absolute paths (e.g. /Users/user/Documents/project)"`
		MaxCoupledBlocks int      `json:"max_coupled_blocks,omitempty" jsonschema:"maximum number of coupled blocks to return per code block (default: 1)"`
		MaxIncidents     int      `json:"max_incidents,omitempty" jsonschema:"maximum number of incidents to return per code block (default: 1)"`
		MaxBlocks        int      `json:"max_blocks,omitempty" jsonschema:"maximum number of code blocks to return (default: 10, use 0 for all)"`
		BlockTypes       []string `json:"block_types,omitempty" jsonschema:"filter by block types (e.g. ['class', 'function', 'method'])"`
		SummaryOnly      bool     `json:"summary_only,omitempty" jsonschema:"return only aggregated statistics instead of detailed block data"`
		MinStaleness     int      `json:"min_staleness,omitempty" jsonschema:"only return blocks with staleness >= this many days (for finding stale code)"`
		MinIncidents     int      `json:"min_incidents,omitempty" jsonschema:"only return blocks with at least this many incidents (for finding risky code)"`
		MinRiskScore     float64  `json:"min_risk_score,omitempty" jsonschema:"only return blocks with risk score >= threshold (e.g. 5.0 for moderate+ risk)"`
		IncludeRiskScore bool     `json:"include_risk_score,omitempty" jsonschema:"include the calculated risk score in output for debugging/verification"`
		PrioritizeRecent bool     `json:"prioritize_recent,omitempty" jsonschema:"boost risk score for recently changed code (focuses on active development areas)"`
	}

	type ToolOutput struct {
		FilePath     string                  `json:"file_path"`
		TotalBlocks  int                     `json:"total_blocks,omitempty"`
		RiskEvidence []tools.BlockEvidence   `json:"risk_evidence"`
		Warning      string                  `json:"warning,omitempty"`
	}

	// Create the tool instance with diff atomizer support
	riskTool := tools.NewGetRiskSummaryTool(graphClient, resolver, diffAtomizer)

	// Register the tool with the SDK
	mcp.AddTool(server, &mcp.Tool{
		Name:        "crisk.get_risk_summary",
		Description: "Get risk evidence for a file including ownership, coupling, and temporal incident data. Automatically detects and analyzes uncommitted changes when a file_path is provided. Use this tool when the user asks about 'my changes', 'uncommitted changes', or 'risk of changes' in a file.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ToolArgs) (*mcp.CallToolResult, ToolOutput, error) {
		// Convert args to map for existing Execute method
		argsMap := map[string]interface{}{
			"file_path": args.FilePath,
		}
		if args.DiffContent != "" {
			argsMap["diff_content"] = args.DiffContent
		}
		if args.RepoRoot != "" {
			argsMap["repo_root"] = args.RepoRoot
		}
		// Set defaults for limits
		maxCoupled := args.MaxCoupledBlocks
		if maxCoupled == 0 {
			maxCoupled = 1 // Changed default from 5 to 1
		}
		maxIncidents := args.MaxIncidents
		if maxIncidents == 0 {
			maxIncidents = 1 // Changed default from 5 to 1
		}
		maxBlocks := args.MaxBlocks
		if maxBlocks == 0 {
			maxBlocks = 10 // Default to 10 blocks
		}

		argsMap["max_coupled_blocks"] = maxCoupled
		argsMap["max_incidents"] = maxIncidents
		argsMap["max_blocks"] = maxBlocks
		argsMap["block_types"] = args.BlockTypes
		argsMap["summary_only"] = args.SummaryOnly
		argsMap["min_staleness"] = args.MinStaleness
		argsMap["min_incidents"] = args.MinIncidents
		argsMap["min_risk_score"] = args.MinRiskScore
		argsMap["include_risk_score"] = args.IncludeRiskScore
		argsMap["prioritize_recent"] = args.PrioritizeRecent

		// Execute the tool
		result, err := riskTool.Execute(ctx, argsMap)
		if err != nil {
			// Return error as tool result (not protocol error)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: "Error: " + err.Error()},
				},
				IsError: true,
			}, ToolOutput{}, nil
		}

		// Convert result map to typed output
		resultMap := result.(map[string]interface{})
		output := ToolOutput{
			FilePath: resultMap["file_path"].(string),
		}

		if totalBlocks, ok := resultMap["total_blocks"].(int); ok {
			output.TotalBlocks = totalBlocks
		}

		if warning, ok := resultMap["warning"].(string); ok {
			output.Warning = warning
		}

		if evidence, ok := resultMap["risk_evidence"].([]tools.BlockEvidence); ok {
			output.RiskEvidence = evidence
		}

		return &mcp.CallToolResult{}, output, nil
	})

	log.Println("✅ Registered tool: crisk.get_risk_summary")

	// 9. Start server on stdio transport
	log.Println("🚀 MCP server started on stdio")
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
