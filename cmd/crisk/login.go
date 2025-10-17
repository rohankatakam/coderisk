package main

import (
	"context"
	"fmt"

	"github.com/rohankatakam/coderisk/internal/auth"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with CodeRisk cloud services",
	Long: `Authenticate with CodeRisk cloud services using OAuth.

This opens your browser to log in with Clerk authentication.
Your credentials (OpenAI API key, GitHub token) will be managed in the cloud
and synced across all your devices.

All code analysis still runs locally - only auth and credentials are cloud-based.`,
	RunE: runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Welcome to CodeRisk Cloud Authentication")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Check if already authenticated
	if auth.IsAuthenticated() {
		token, _ := auth.LoadToken()
		fmt.Printf("✓ Already authenticated as %s\n", token.User.Email)
		fmt.Printf("→ Token expires: %s\n", token.ExpiresAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
		fmt.Println("Run 'crisk logout' to sign out")
		return nil
	}

	// TODO: Implement full device flow
	// For MVP, we'll use a manual token flow
	fmt.Println("🔐 Opening browser for authentication...")
	fmt.Println()
	fmt.Printf("Visit: %s\n", auth.AuthCallbackURL)
	fmt.Println()
	fmt.Println("After logging in, you'll receive a session token.")
	fmt.Println("We're working on automating this flow!")
	fmt.Println()

	// Open browser
	authClient := auth.NewClerkClient()
	_, err := authClient.DeviceFlowAuth(ctx)
	if err != nil {
		// For MVP, provide instructions for manual flow
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("Manual Setup (Temporary)")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Println("1. Visit: https://coderisk.dev")
		fmt.Println("2. Sign in or create an account")
		fmt.Println("3. Go to Settings > API Keys")
		fmt.Println("4. Add your OpenAI API key and GitHub token")
		fmt.Println("5. Copy your session token from the browser")
		fmt.Println()
		fmt.Println("Then run:")
		fmt.Println("  export CODERISK_SESSION_TOKEN=<your_token>")
		fmt.Println("  crisk whoami")
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Println("Note: Automatic OAuth flow coming soon!")
		fmt.Println()
		return nil
	}

	// Save token
	// token := ... (from device flow)
	// if err := auth.SaveToken(token); err != nil {
	// 	return fmt.Errorf("failed to save token: %w", err)
	// }

	fmt.Println()
	fmt.Println("✓ Successfully authenticated!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("1. Visit: https://coderisk.dev/dashboard/settings")
	fmt.Println("2. Add your OpenAI API key")
	fmt.Println("3. Add your GitHub token")
	fmt.Println("4. Run: crisk init")
	fmt.Println()

	return nil
}

// Helper function to create a token from environment variable (temporary)
func createTokenFromEnv() (*auth.Token, error) {
	// TODO: Remove this once device flow is implemented
	// This is a temporary helper for MVP testing
	return nil, fmt.Errorf("not implemented")
}
