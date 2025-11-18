package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/rohankatakam/coderisk/internal/config"
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
	Use:   "crisk-init",
	Short: "Orchestrate all microservices for full repository initialization",
	Long: `crisk-init - Microservice Orchestrator

Coordinates execution of all 6 microservices in the correct sequence:
  1. crisk-stage      - Download GitHub data + file identity map
  2. crisk-ingest     - Build 100% confidence graph
  3. crisk-atomize    - Create semantic CodeBlock nodes (MANDATORY)
  4. crisk-index-incident   - Link issues to blocks, calculate temporal risk
  5. crisk-index-ownership  - Calculate ownership signals
  6. crisk-index-coupling   - Calculate coupling signals

This orchestrator ensures each service completes successfully before
starting the next one. Failure in any service stops the pipeline.

Usage:
  cd /path/to/repo
  crisk-init [--days N]`,
	Version: Version,
	RunE:    runOrchestrator,
}

var (
	days    int
	verbose bool
)

func init() {
	rootCmd.Flags().IntVar(&days, "days", 0, "Ingest last N days only (0 = all history)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	rootCmd.SetVersionTemplate(`crisk-init {{.Version}}
Build time: ` + BuildTime + `
Git commit: ` + GitCommit + `
`)
}

func runOrchestrator(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := context.Background()

	fmt.Printf("🚀 crisk-init - Microservice Orchestrator\n")
	fmt.Printf("   Version: %s\n", Version)
	fmt.Println()

	// Detect current repository
	fmt.Printf("[0/6] Detecting repository...\n")
	owner, repo, repoPath, err := detectCurrentRepo()
	if err != nil {
		return err
	}
	fmt.Printf("  ✓ Detected: %s/%s\n", owner, repo)
	fmt.Printf("  ✓ Path: %s\n", repoPath)
	fmt.Println()

	// Validate environment
	if err := validateEnvironment(ctx); err != nil {
		return err
	}

	// Get path to service binaries (same directory as this orchestrator)
	binDir, err := getBinaryDirectory()
	if err != nil {
		return fmt.Errorf("failed to find binary directory: %w", err)
	}

	// Service 1: Stage
	fmt.Printf("[1/6] Running crisk-stage...\n")
	stageArgs := []string{
		"--owner", owner,
		"--repo", repo,
		"--path", repoPath,
	}
	if days > 0 {
		stageArgs = append(stageArgs, "--days", fmt.Sprintf("%d", days))
	}
	if verbose {
		stageArgs = append(stageArgs, "--verbose")
	}

	repoID, err := runService(ctx, binDir, "crisk-stage", stageArgs...)
	if err != nil {
		return fmt.Errorf("crisk-stage failed: %w", err)
	}
	fmt.Printf("  ✓ Stage complete (repo_id=%s)\n\n", repoID)

	// Service 2: Ingest
	fmt.Printf("[2/6] Running crisk-ingest...\n")
	ingestArgs := []string{
		"--repo-id", repoID,
		"--repo-path", repoPath,
	}
	if verbose {
		ingestArgs = append(ingestArgs, "--verbose")
	}

	_, err = runService(ctx, binDir, "crisk-ingest", ingestArgs...)
	if err != nil {
		return fmt.Errorf("crisk-ingest failed: %w", err)
	}
	fmt.Printf("  ✓ Ingest complete\n\n")

	// Service 3: Atomize (MANDATORY)
	fmt.Printf("[3/6] Running crisk-atomize...\n")
	atomizeArgs := []string{
		"--repo-id", repoID,
		"--repo-path", repoPath,
	}
	if verbose {
		atomizeArgs = append(atomizeArgs, "--verbose")
	}

	_, err = runService(ctx, binDir, "crisk-atomize", atomizeArgs...)
	if err != nil {
		return fmt.Errorf("crisk-atomize failed: %w", err)
	}
	fmt.Printf("  ✓ Atomize complete\n\n")

	// Service 4: Index Incident
	fmt.Printf("[4/6] Running crisk-index-incident...\n")
	incidentArgs := []string{
		"--repo-id", repoID,
	}
	if verbose {
		incidentArgs = append(incidentArgs, "--verbose")
	}

	_, err = runService(ctx, binDir, "crisk-index-incident", incidentArgs...)
	if err != nil {
		return fmt.Errorf("crisk-index-incident failed: %w", err)
	}
	fmt.Printf("  ✓ Incident indexing complete\n\n")

	// Service 5: Index Ownership
	fmt.Printf("[5/6] Running crisk-index-ownership...\n")
	ownershipArgs := []string{
		"--repo-id", repoID,
	}
	if verbose {
		ownershipArgs = append(ownershipArgs, "--verbose")
	}

	_, err = runService(ctx, binDir, "crisk-index-ownership", ownershipArgs...)
	if err != nil {
		return fmt.Errorf("crisk-index-ownership failed: %w", err)
	}
	fmt.Printf("  ✓ Ownership indexing complete\n\n")

	// Service 6: Index Coupling
	fmt.Printf("[6/6] Running crisk-index-coupling...\n")
	couplingArgs := []string{
		"--repo-id", repoID,
	}
	if verbose {
		couplingArgs = append(couplingArgs, "--verbose")
	}

	_, err = runService(ctx, binDir, "crisk-index-coupling", couplingArgs...)
	if err != nil {
		return fmt.Errorf("crisk-index-coupling failed: %w", err)
	}
	fmt.Printf("  ✓ Coupling indexing complete\n\n")

	// Summary
	totalDuration := time.Since(startTime)
	fmt.Printf("✅ All services completed successfully!\n")
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("   Repository: %s/%s (ID: %s)\n", owner, repo, repoID)
	fmt.Printf("   Total time: %v\n", totalDuration)
	fmt.Printf("\n🚀 Next steps:\n")
	fmt.Printf("   • Test: crisk check <file>\n")
	fmt.Printf("   • Browse graph: http://localhost:7475 (Neo4j Browser)\n")

	return nil
}

// detectCurrentRepo detects the git repository in the current directory
func detectCurrentRepo() (owner, repo, repoPath string, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find git root directory
	gitRoot := cwd
	for {
		if _, err := os.Stat(filepath.Join(gitRoot, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(gitRoot)
		if parent == gitRoot {
			return "", "", "", fmt.Errorf("not a git repository\n\nRun this command inside a cloned git repository")
		}
		gitRoot = parent
	}

	// Get git remote URL
	cmd := exec.Command("git", "-C", gitRoot, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get git remote: %w\n\nMake sure the repository has an 'origin' remote set", err)
	}

	remoteURL := strings.TrimSpace(string(output))

	// Parse owner/repo from remote URL
	re := regexp.MustCompile(`github\.com[:/]([^/]+)/(.+?)(?:\.git)?$`)
	matches := re.FindStringSubmatch(remoteURL)
	if matches == nil || len(matches) < 3 {
		return "", "", "", fmt.Errorf("could not parse GitHub owner/repo from remote URL: %s", remoteURL)
	}

	owner = matches[1]
	repo = matches[2]
	repoPath = gitRoot

	return owner, repo, repoPath, nil
}

// validateEnvironment checks that required environment variables are set
func validateEnvironment(ctx context.Context) error {
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return fmt.Errorf("GITHUB_TOKEN environment variable not set")
	}

	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	if geminiAPIKey == "" {
		return fmt.Errorf("GEMINI_API_KEY environment variable not set\n\nAtomization requires LLM API access")
	}

	// Validate database connections
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	mode := config.DetectMode()
	result := cfg.ValidateWithMode(config.ValidationContextInit, mode)
	if result.HasErrors() {
		return fmt.Errorf("configuration validation failed:\n%s", result.Error())
	}

	fmt.Printf("  ✓ Environment validated\n")
	return nil
}

// getBinaryDirectory returns the directory containing the service binaries
func getBinaryDirectory() (string, error) {
	// Get the path to the current executable
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}

	// Get the directory containing the executable
	binDir := filepath.Dir(execPath)

	// In development, binaries might be in the same directory or in ../bin
	// Check if crisk-stage exists in the same directory
	if _, err := os.Stat(filepath.Join(binDir, "crisk-stage")); err == nil {
		return binDir, nil
	}

	// Check parent directory / bin
	parentBin := filepath.Join(filepath.Dir(binDir), "bin")
	if _, err := os.Stat(filepath.Join(parentBin, "crisk-stage")); err == nil {
		return parentBin, nil
	}

	// Default to same directory
	return binDir, nil
}

// runService executes a microservice binary and captures output
func runService(ctx context.Context, binDir, serviceName string, args ...string) (string, error) {
	binPath := filepath.Join(binDir, serviceName)

	// Check if binary exists
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		return "", fmt.Errorf("service binary not found: %s\n\nRun 'make build' to compile all services", binPath)
	}

	// Create command
	cmd := exec.CommandContext(ctx, binPath, args...)

	// For crisk-stage, capture output to extract repo_id
	var outputBuf strings.Builder
	if serviceName == "crisk-stage" {
		cmd.Stdout = &outputBuf
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	cmd.Env = os.Environ() // Pass through all environment variables

	// Run command
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("service execution failed: %w", err)
	}

	// Extract repo_id from output if this is crisk-stage
	if serviceName == "crisk-stage" {
		output := outputBuf.String()

		// Print the output to user
		fmt.Print(output)

		// Extract REPO_ID=N from output
		re := regexp.MustCompile(`REPO_ID=(\d+)`)
		matches := re.FindStringSubmatch(output)
		if matches != nil && len(matches) > 1 {
			return matches[1], nil
		}
		return "", fmt.Errorf("failed to extract repo_id from service output")
	}

	return "", nil
}
