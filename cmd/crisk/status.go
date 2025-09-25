package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/coderisk/coderisk-go/internal/cache"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show CodeRisk status and cache information",
	Long:  `Display current CodeRisk configuration, cache status, and repository information.`,
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	fmt.Printf("🔍 CodeRisk Status\n")
	fmt.Printf("%s\n", strings.Repeat("═", 50))

	// Configuration info
	fmt.Printf("\n📋 Configuration:\n")
	fmt.Printf("  Mode: %s\n", cfg.Mode)
	fmt.Printf("  Storage: %s\n", cfg.Storage.Type)
	fmt.Printf("  Cache directory: %s\n", cfg.Cache.Directory)

	// Cache status
	cacheManager := cache.NewManager(cfg, logger)

	fmt.Printf("\n💾 Cache Status:\n")

	cacheAge, err := cacheManager.GetCacheAge(ctx)
	if err != nil {
		fmt.Printf("  Status: ❌ Not initialized (run 'crisk init')\n")
		return nil
	}

	fmt.Printf("  Status: ✅ Initialized\n")
	fmt.Printf("  Age: %s\n", formatDuration(cacheAge))

	// Get cache size
	size, err := cacheManager.GetSize(ctx)
	if err == nil {
		fmt.Printf("  Size: %s\n", formatBytes(size))
	}

	// Cache metadata
	metadata, err := cacheManager.GetMetadata(ctx)
	if err == nil {
		fmt.Printf("  Version: %s\n", metadata.Version)
		fmt.Printf("  Files: %d\n", metadata.FileCount)
		fmt.Printf("  Risk sketches: %d\n", metadata.SketchCount)
		if metadata.LastCommit != "" {
			fmt.Printf("  Last commit: %s\n", metadata.LastCommit[:min(8, len(metadata.LastCommit))])
		}
		fmt.Printf("  Last updated: %s\n", metadata.LastUpdated.Format("2006-01-02 15:04:05"))
	}

	// Repository info
	fmt.Printf("\n🔗 Repository:\n")
	repoInfo, err := detectGitRepo()
	if err != nil {
		fmt.Printf("  Status: ❌ Not a git repository\n")
	} else {
		fmt.Printf("  URL: %s\n", repoInfo.URL)
		fmt.Printf("  Branch: %s\n", repoInfo.Branch)
	}

	// Sync configuration
	fmt.Printf("\n🔄 Sync Settings:\n")
	fmt.Printf("  Auto-sync: %v\n", cfg.Sync.AutoSync)
	fmt.Printf("  Fresh threshold: %s\n", cfg.Sync.FreshThreshold)
	fmt.Printf("  Stale threshold: %s\n", cfg.Sync.StaleThreshold)

	// Budget info
	fmt.Printf("\n💰 Budget:\n")
	fmt.Printf("  Daily limit: $%.2f\n", cfg.Budget.DailyLimit)
	fmt.Printf("  Monthly limit: $%.2f\n", cfg.Budget.MonthlyLimit)
	fmt.Printf("  Per-check limit: $%.2f\n", cfg.Budget.PerCheckLimit)

	// Quick health check
	fmt.Printf("\n🏥 Health Check:\n")

	// Check if we can load sketches
	_, err = cacheManager.LoadSketches(ctx)
	if err != nil {
		fmt.Printf("  Risk sketches: ❌ Cannot load (%v)\n", err)
	} else {
		fmt.Printf("  Risk sketches: ✅ Available\n")
	}

	// Check git status
	changes, err := getChangedFiles()
	if err != nil {
		fmt.Printf("  Git integration: ❌ Cannot detect changes\n")
	} else {
		if len(changes) == 0 {
			fmt.Printf("  Working tree: ✅ Clean\n")
		} else {
			fmt.Printf("  Working tree: ⚠️ %d uncommitted changes\n", len(changes))
		}
	}

	fmt.Println("\n💡 Ready! Run 'crisk check' to analyze your changes")

	return nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
