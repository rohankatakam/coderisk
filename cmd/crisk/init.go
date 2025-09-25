package main

import (
	"context"
	"fmt"
	"time"

	"github.com/coderisk/coderisk-go/internal/cache"
	"github.com/coderisk/coderisk-go/internal/github"
	"github.com/coderisk/coderisk-go/internal/ingestion"
	"github.com/coderisk/coderisk-go/internal/models"
	"github.com/coderisk/coderisk-go/internal/storage"
	"github.com/spf13/cobra"
)

var (
	connectURL string
	localOnly  bool
	repoURL    string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize CodeRisk for a repository",
	Long: `Initialize CodeRisk by either connecting to a shared team cache
or performing a full repository ingestion.`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&connectURL, "connect", "", "Connect to existing team cache")
	initCmd.Flags().BoolVar(&localOnly, "local-only", false, "Use local analysis only (no cloud)")
	initCmd.Flags().StringVar(&repoURL, "repo", "", "Repository URL to ingest")
}

func runInit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Try auto-discovery if no flags provided
	if connectURL == "" && !localOnly && repoURL == "" {
		logger.Info("Auto-discovering repository configuration...")
		if err := autoDiscover(ctx); err != nil {
			logger.WithError(err).Debug("Auto-discovery failed")
			return fmt.Errorf("could not auto-discover repository. Use --connect, --local-only, or --repo flag")
		}
		return nil
	}

	// Connect to existing cache
	if connectURL != "" {
		return connectToCache(ctx, connectURL)
	}

	// Local-only mode
	if localOnly {
		return initLocalMode(ctx)
	}

	// Full repository ingestion
	if repoURL != "" {
		return performIngestion(ctx, repoURL)
	}

	return fmt.Errorf("no initialization method specified")
}

func autoDiscover(ctx context.Context) error {
	// Detect git repository
	repoInfo, err := detectGitRepo()
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}

	logger.WithField("repo", repoInfo.URL).Info("Detected repository")

	// Query cache registry
	cacheManager := cache.NewManager(cfg, logger)
	cacheInfo, err := cacheManager.QueryRegistry(ctx, repoInfo.URL)
	if err != nil {
		logger.WithError(err).Debug("No existing cache found")

		// Offer to create new cache
		fmt.Printf("No cache found for %s\n", repoInfo.URL)
		fmt.Println("Would you like to:")
		fmt.Println("1. Initialize shared team cache (recommended, ~$15)")
		fmt.Println("2. Use local-only mode (free, limited features)")
		fmt.Println("3. Cancel")

		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			return performIngestion(ctx, repoInfo.URL)
		case 2:
			return initLocalMode(ctx)
		default:
			return fmt.Errorf("initialization cancelled")
		}
	}

	// Found existing cache
	fmt.Printf("✓ Found team cache for %s\n", repoInfo.URL)
	fmt.Printf("  Last updated: %s\n", cacheInfo.LastUpdated.Format("2006-01-02 15:04"))
	fmt.Printf("  Managed by: %s\n", cacheInfo.Admin)

	return connectToCache(ctx, cacheInfo.URL)
}

func connectToCache(ctx context.Context, url string) error {
	logger.WithField("url", url).Info("Connecting to team cache")

	cacheManager := cache.NewManager(cfg, logger)
	if err := cacheManager.Connect(ctx, url); err != nil {
		return fmt.Errorf("failed to connect to cache: %w", err)
	}

	// Pull initial cache data
	if err := cacheManager.Pull(ctx); err != nil {
		return fmt.Errorf("failed to pull cache: %w", err)
	}

	fmt.Println("✓ Connected to team cache!")
	fmt.Println("✓ Cache synchronized")
	fmt.Println("\nReady to use! Run 'crisk check' to assess risk")

	return nil
}

func initLocalMode(ctx context.Context) error {
	logger.Info("Initializing local-only mode")

	// Create local storage
	store, err := storage.NewSQLiteStore(cfg.Storage.LocalPath, logger)
	if err != nil {
		return fmt.Errorf("failed to create local storage: %w", err)
	}
	defer store.Close()

	// Download local models if needed
	fmt.Println("Setting up local analysis models...")
	// TODO: Download embedding models, etc.

	fmt.Println("✓ Local mode initialized!")
	fmt.Println("\nNote: Local mode provides Level 1 analysis only")
	fmt.Println("Run 'crisk check' to perform risk assessment")

	return nil
}

func performIngestion(ctx context.Context, repoURL string) error {
	logger.WithField("repo", repoURL).Info("Starting full repository ingestion")

	// Parse repository info
	owner, name, err := parseRepoURL(repoURL)
	if err != nil {
		return fmt.Errorf("invalid repository URL: %w", err)
	}

	// Estimate cost
	fmt.Printf("Initializing repository: %s/%s\n", owner, name)
	fmt.Println("Estimated cost: $15-50 (one-time)")
	fmt.Print("Continue? [y/N]: ")

	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "Y" {
		return fmt.Errorf("ingestion cancelled")
	}

	// Create storage
	store, err := createStorage()
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}
	defer store.Close()

	// Create GitHub client
	githubClient := github.NewClient(cfg.GitHub.Token, cfg.GitHub.RateLimit)
	extractor := github.NewExtractor(githubClient, logger)

	// Create ingestion orchestrator
	orchestrator := ingestion.NewOrchestrator(
		extractor,
		store,
		logger,
		cfg,
	)

	// Perform ingestion
	startTime := time.Now()
	fmt.Println("\n⚡ Starting ingestion...")

	result, err := orchestrator.IngestRepository(ctx, owner, name)
	if err != nil {
		return fmt.Errorf("ingestion failed: %w", err)
	}

	duration := time.Since(startTime)

	fmt.Printf("\n✓ Ingestion completed in %s\n", duration)
	fmt.Printf("  Files processed: %d\n", result.FileCount)
	fmt.Printf("  Commits analyzed: %d\n", result.CommitCount)
	fmt.Printf("  Risk sketches created: %d\n", result.SketchCount)
	fmt.Printf("  Total cost: $%.2f\n", result.Cost)

	// Save cache metadata
	metadata := &models.CacheMetadata{
		Version:     generateVersion(),
		RepoID:      fmt.Sprintf("%s/%s", owner, name),
		LastCommit:  result.LastCommit,
		LastUpdated: time.Now(),
		FileCount:   result.FileCount,
		SketchCount: result.SketchCount,
	}

	if err := store.SaveCacheMetadata(ctx, metadata); err != nil {
		logger.WithError(err).Warn("Failed to save cache metadata")
	}

	fmt.Println("\n✓ Repository initialized successfully!")
	fmt.Println("Run 'crisk check' to start analyzing code changes")

	return nil
}

// Helper functions

type gitRepoInfo struct {
	URL    string
	Owner  string
	Name   string
	Branch string
}

func detectGitRepo() (*gitRepoInfo, error) {
	// TODO: Implement git detection using go-git
	return nil, fmt.Errorf("not implemented")
}

func parseRepoURL(url string) (owner, name string, err error) {
	// TODO: Parse GitHub URL
	return "", "", fmt.Errorf("not implemented")
}

func createStorage() (storage.Store, error) {
	switch cfg.Storage.Type {
	case "postgres":
		return storage.NewPostgresStore(cfg.Storage.PostgresDSN, logger)
	case "sqlite":
		return storage.NewSQLiteStore(cfg.Storage.LocalPath, logger)
	default:
		return nil, fmt.Errorf("unknown storage type: %s", cfg.Storage.Type)
	}
}

func generateVersion() string {
	return fmt.Sprintf("v1-%d", time.Now().Unix())
}
