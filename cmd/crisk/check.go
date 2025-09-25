package main

import (
	"context"
	"fmt"
	"time"

	"github.com/coderisk/coderisk-go/internal/cache"
	"github.com/coderisk/coderisk-go/internal/models"
	"github.com/coderisk/coderisk-go/internal/risk"
	"github.com/spf13/cobra"
	"strings"
)

var (
	offline   bool
	fresh     bool
	explain   bool
	preCommit bool
	level     int
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Perform risk assessment on current changes",
	Long: `Analyze uncommitted changes and provide risk assessment.
This command runs in under 5 seconds using cached risk sketches.`,
	RunE: runCheck,
}

func init() {
	checkCmd.Flags().BoolVar(&offline, "offline", false, "Run in offline mode (no sync)")
	checkCmd.Flags().BoolVar(&fresh, "fresh", false, "Force cache refresh before check")
	checkCmd.Flags().BoolVar(&explain, "explain", false, "Show detailed explanations")
	checkCmd.Flags().BoolVar(&preCommit, "pre-commit", false, "Run as pre-commit hook")
	checkCmd.Flags().IntVar(&level, "level", 1, "Analysis level (1=fast, 2=standard, 3=deep)")
}

func runCheck(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	startTime := time.Now()

	// Initialize cache manager
	cacheManager := cache.NewManager(cfg, logger)

	// Smart sync logic
	if !offline {
		if err := smartSync(ctx, cacheManager); err != nil {
			logger.WithError(err).Warn("Cache sync failed, continuing with local cache")
		}
	}

	// Get changed files
	changes, err := getChangedFiles()
	if err != nil {
		return fmt.Errorf("failed to detect changes: %w", err)
	}

	if len(changes) == 0 {
		fmt.Println("✓ No uncommitted changes detected")
		return nil
	}

	logger.WithField("files", len(changes)).Info("Analyzing changes")

	// Load risk sketches from cache
	sketches, err := cacheManager.LoadSketches(ctx)
	if err != nil {
		return fmt.Errorf("failed to load risk sketches: %w", err)
	}

	// Perform risk calculation
	calculator := risk.NewCalculator(logger, nil)
	assessment, err := calculator.CalculateRisk(ctx, sketches, changes)
	if err != nil {
		return fmt.Errorf("risk calculation failed: %w", err)
	}

	// Display results
	displayResults(assessment, time.Since(startTime))

	// Show explanations if requested
	if explain {
		displayExplanations(assessment)
	}

	// Pre-commit mode: exit with error if risk too high
	if preCommit && assessment.Level == models.RiskLevelCritical {
		return fmt.Errorf("critical risk detected - commit blocked")
	}

	return nil
}

func smartSync(ctx context.Context, cacheManager *cache.Manager) error {
	cacheAge, err := cacheManager.GetCacheAge(ctx)
	if err != nil {
		return err
	}

	freshThreshold := 30 * time.Minute
	staleThreshold := 4 * time.Hour

	// Auto-sync logic
	if cacheAge > staleThreshold {
		fmt.Printf("⚠ Cache is %s old. Syncing...\n", formatDuration(cacheAge))
		if err := cacheManager.Pull(ctx); err != nil {
			return err
		}
		fmt.Println("✓ Updated")
	} else if fresh {
		fmt.Println("Forcing cache refresh...")
		if err := cacheManager.Pull(ctx); err != nil {
			return err
		}
		fmt.Println("✓ Updated")
	} else if cacheAge > freshThreshold {
		fmt.Printf("ℹ Cache age: %s\n", formatDuration(cacheAge))
	}

	return nil
}

func getChangedFiles() ([]string, error) {
	// TODO: Use go-git to detect uncommitted changes
	// For now, return mock data
	return []string{}, nil
}

func displayResults(assessment *models.RiskAssessment, duration time.Duration) {
	// Risk level with color coding
	levelStr := formatRiskLevel(assessment.Level)

	fmt.Printf("\n%s Risk Assessment %s\n", getIcon(assessment.Level), strings.Repeat("═", 50))
	fmt.Printf("Risk Level: %s (Score: %.2f)\n", levelStr, assessment.Score)
	fmt.Printf("Analysis Time: %s\n", duration.Round(time.Millisecond))

	if assessment.BlastRadius > 0.3 {
		fmt.Printf("Blast Radius: %.0f%% of codebase affected\n", assessment.BlastRadius*100)
	}
	if assessment.TestCoverage < 0.5 {
		fmt.Printf("Test Coverage: %.0f%% (consider adding tests)\n", assessment.TestCoverage*100)
	}

	// Top risk factors
	if len(assessment.Factors) > 0 {
		fmt.Println("\nTop Risk Factors:")
		for i, factor := range assessment.Factors {
			fmt.Printf("%d. %s %s (%s)\n",
				i+1,
				getImpactIcon(factor.Impact),
				factor.Signal,
				factor.Impact)
			fmt.Printf("   %s\n", factor.Detail)
		}
	}

	// Suggestions
	if len(assessment.Suggestions) > 0 {
		fmt.Println("\nSuggestions:")
		for _, suggestion := range assessment.Suggestions {
			fmt.Printf("• %s\n", suggestion)
		}
	}

	fmt.Println()
}

func displayExplanations(assessment *models.RiskAssessment) {
	fmt.Println("\n📊 Detailed Analysis")
	fmt.Println(strings.Repeat("─", 60))

	for _, factor := range assessment.Factors {
		fmt.Printf("\n%s %s\n", getImpactIcon(factor.Impact), factor.Signal)
		fmt.Printf("Impact: %s | Score: %.2f\n", factor.Impact, factor.Score)
		fmt.Printf("Details: %s\n", factor.Detail)

		if factor.Evidence != "" {
			fmt.Printf("Evidence: %s\n", factor.Evidence)
		}

		fmt.Printf("Learn more: crisk explain --signal %s\n",
			strings.ToLower(strings.ReplaceAll(factor.Signal, " ", "-")))
	}

	fmt.Printf("\nView full analysis: %s\n", getAnalysisURL(assessment.ID))
}

// Helper functions

func formatRiskLevel(level models.RiskLevel) string {
	colors := map[models.RiskLevel]string{
		models.RiskLevelLow:      "\033[32m", // Green
		models.RiskLevelMedium:   "\033[33m", // Yellow
		models.RiskLevelHigh:     "\033[31m", // Red
		models.RiskLevelCritical: "\033[35m", // Magenta
	}
	reset := "\033[0m"

	color, ok := colors[level]
	if !ok {
		return string(level)
	}

	return fmt.Sprintf("%s%s%s", color, level, reset)
}

func getIcon(level models.RiskLevel) string {
	icons := map[models.RiskLevel]string{
		models.RiskLevelLow:      "✅",
		models.RiskLevelMedium:   "⚠️",
		models.RiskLevelHigh:     "🔴",
		models.RiskLevelCritical: "🚨",
	}
	return icons[level]
}

func getImpactIcon(impact string) string {
	icons := map[string]string{
		"CRITICAL": "🔴",
		"HIGH":     "🟠",
		"MEDIUM":   "🟡",
		"LOW":      "🟢",
	}
	return icons[impact]
}

func formatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d hours", int(d.Hours()))
	}
	return fmt.Sprintf("%d days", int(d.Hours()/24))
}

func getAnalysisURL(id string) string {
	return fmt.Sprintf("https://app.coderisk.ai/analysis/%s", id)
}
