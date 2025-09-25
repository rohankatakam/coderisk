package main

import (
	"context"
	"fmt"
	"time"

	"github.com/coderisk/coderisk-go/internal/cache"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull latest cache updates from remote",
	Long: `Synchronize local cache with the shared team cache,
fetching the latest risk sketches and repository data.`,
	RunE: runPull,
}

func runPull(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	startTime := time.Now()

	// Initialize cache manager
	cacheManager := cache.NewManager(cfg, logger)

	fmt.Println("🔄 Pulling cache updates...")

	// Check current cache status
	currentAge, err := cacheManager.GetCacheAge(ctx)
	if err != nil {
		logger.WithError(err).Debug("Could not get cache age")
	} else {
		fmt.Printf("Current cache age: %s\n", formatDuration(currentAge))
	}

	// Perform pull
	if err := cacheManager.Pull(ctx); err != nil {
		return fmt.Errorf("failed to pull cache: %w", err)
	}

	duration := time.Since(startTime)
	fmt.Printf("✅ Cache updated in %s\n", duration.Round(time.Millisecond))

	// Show updated status
	metadata, err := cacheManager.GetMetadata(ctx)
	if err != nil {
		logger.WithError(err).Warn("Could not get updated metadata")
		return nil
	}

	fmt.Printf("\nCache Status:\n")
	fmt.Printf("  Version: %s\n", metadata.Version)
	fmt.Printf("  Files: %d\n", metadata.FileCount)
	fmt.Printf("  Risk sketches: %d\n", metadata.SketchCount)
	fmt.Printf("  Last commit: %s\n", metadata.LastCommit[:8])
	fmt.Printf("  Updated: %s\n", metadata.LastUpdated.Format("2006-01-02 15:04:05"))

	return nil
}
