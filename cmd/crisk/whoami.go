package main

import (
	"context"
	"fmt"
	"time"

	"github.com/rohankatakam/coderisk/internal/auth"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Display current authentication status and user information",
	Long: `Display current authentication status including:
- Authenticated user email
- Token expiry time
- API key configuration status
- Usage statistics for current month`,
	RunE: runWhoami,
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}

func runWhoami(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Check authentication
	if !auth.IsAuthenticated() {
		fmt.Println("⚠️  Not authenticated")
		fmt.Println()
		fmt.Println("Run 'crisk login' to sign in")
		return nil
	}

	// Load token
	token, err := auth.LoadToken()
	if err != nil {
		return fmt.Errorf("failed to load token: %w", err)
	}

	// Display user info
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Authentication Status")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Printf("Email:           %s\n", token.User.Email)
	fmt.Printf("User ID:         %s\n", token.User.ID)
	fmt.Printf("Authenticated:   ✓ Yes\n")

	// Token expiry
	now := time.Now()
	if now.After(token.ExpiresAt) {
		fmt.Printf("Token Expires:   ⚠️  Expired\n")
		fmt.Println()
		fmt.Println("Your token has expired. Run 'crisk login' to refresh.")
		return nil
	} else {
		daysLeft := int(time.Until(token.ExpiresAt).Hours() / 24)
		fmt.Printf("Token Expires:   %s (%d days)\n", token.ExpiresAt.Format("2006-01-02"), daysLeft)
	}

	fmt.Println()

	// Try to fetch credentials
	manager, err := auth.NewManager()
	if err != nil {
		fmt.Println("⚠️  Failed to initialize auth manager:", err)
		return nil
	}

	if err := manager.LoadSession(); err != nil {
		fmt.Println("⚠️  Failed to load session:", err)
		return nil
	}

	// Verify session with Clerk
	if err := manager.VerifyAuthentication(ctx); err != nil {
		fmt.Println("⚠️  Session verification failed:", err)
		fmt.Println("Try running 'crisk login' again")
		return nil
	}

	// Fetch credentials
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("API Keys")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	creds, err := manager.GetCredentials()
	if err != nil {
		fmt.Println("⚠️  Failed to fetch credentials:", err)
		fmt.Println()
		fmt.Println("Visit: https://coderisk.dev/dashboard/settings")
		fmt.Println("to configure your API keys")
		fmt.Println()
		return nil
	}

	// Check OpenAI API key
	if creds.OpenAIAPIKey != "" {
		maskedKey := maskAPIKey(creds.OpenAIAPIKey)
		fmt.Printf("✓ OpenAI API Key:  %s\n", maskedKey)
	} else {
		fmt.Println("✗ OpenAI API Key:  Not configured")
	}

	// Check GitHub token
	if creds.GitHubToken != "" {
		maskedToken := maskAPIKey(creds.GitHubToken)
		fmt.Printf("✓ GitHub Token:    %s\n", maskedToken)
	} else {
		fmt.Println("✗ GitHub Token:    Not configured")
	}

	fmt.Println()

	// Fetch usage stats
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Usage (Current Month)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	stats, err := manager.GetUsageStats()
	if err != nil {
		fmt.Println("⚠️  Failed to fetch usage stats:", err)
	} else {
		fmt.Printf("Commands:        %d\n", stats.TotalCommands)
		fmt.Printf("OpenAI Tokens:   %s\n", formatNumber(stats.TotalTokens))
		fmt.Printf("Cost:            $%.2f\n", stats.TotalCost)
	}

	fmt.Println()

	// Show next steps if credentials missing
	if creds.OpenAIAPIKey == "" || creds.GitHubToken == "" {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("Next Steps")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Println("Visit: https://coderisk.dev/dashboard/settings")
		fmt.Println("to configure your API keys")
		fmt.Println()
	}

	return nil
}

// maskAPIKey masks an API key for display
func maskAPIKey(key string) string {
	if len(key) <= 12 {
		return key
	}
	prefix := key[:8]
	suffix := key[len(key)-4:]
	return fmt.Sprintf("%s...%s", prefix, suffix)
}

// formatNumber formats a number with commas
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%s,%03d", formatNumber(n/1000), n%1000)
}
