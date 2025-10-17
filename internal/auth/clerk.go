package auth

import (
	"context"
	"fmt"

	"github.com/pkg/browser"
)

// ClerkClient wraps Clerk authentication (simplified for MVP)
type ClerkClient struct {
	publishableKey string
}

// NewClerkClient creates a new Clerk client
func NewClerkClient() *ClerkClient {
	return &ClerkClient{
		publishableKey: ClerkPublishableKey,
	}
}

// DeviceFlowAuth performs OAuth device flow authentication
// This opens a browser for the user to authenticate and returns a token
func (c *ClerkClient) DeviceFlowAuth(ctx context.Context) (*Token, error) {
	// For MVP, we use a simplified flow:
	// 1. Open browser to authentication page
	// 2. User authenticates and copies their session token
	// 3. We store the token locally

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Welcome to CodeRisk Cloud Authentication")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("📖 Instructions:")
	fmt.Println("   1. Browser will open to CodeRisk authentication page")
	fmt.Println("   2. Sign in with your account (or create one)")
	fmt.Println("   3. Your credentials will be automatically configured")
	fmt.Println()
	fmt.Println("Press Enter to open browser...")
	fmt.Scanln() // Wait for user to press Enter

	// Open browser for authentication
	if err := BrowserAuth(); err != nil {
		fmt.Printf("⚠️  Browser opening failed: %v\n", err)
		fmt.Printf("    Please visit manually: %s\n", AuthCallbackURL)
	}

	fmt.Println()
	fmt.Println("⏳ Waiting for authentication...")
	fmt.Println("   (This feature is in development)")
	fmt.Println()
	fmt.Println("For now, authentication will be completed in Phase 1.4")
	fmt.Println("You can still use CodeRisk with environment variables:")
	fmt.Println("   export OPENAI_API_KEY=sk-...")
	fmt.Println("   export GITHUB_TOKEN=ghp_...")
	fmt.Println()

	// TODO: Implement actual device flow polling
	// For now, return an error indicating manual setup is needed
	return nil, fmt.Errorf("automatic authentication coming in Phase 1.4 - use environment variables for now")
}

// VerifySession verifies a Clerk session token
// For MVP, we'll do basic validation
func (c *ClerkClient) VerifySession(ctx context.Context, sessionToken string) (*User, error) {
	// TODO: Implement proper JWT verification with Clerk
	// For MVP, we'll assume the token is valid if it exists

	// Basic validation
	if sessionToken == "" {
		return nil, fmt.Errorf("session token is empty")
	}

	// For MVP, we'll return a placeholder
	// The real implementation will verify the JWT with Clerk
	return &User{
		ID:    "user_placeholder",
		Email: "user@example.com",
	}, nil
}

// BrowserAuth opens a browser for authentication
func BrowserAuth() error {
	authURL := AuthCallbackURL
	fmt.Printf("🔐 Opening browser for authentication...\n\n")
	fmt.Printf("Visit: %s\n\n", authURL)

	if err := browser.OpenURL(authURL); err != nil {
		fmt.Printf("⚠️  Could not open browser automatically. Please visit the URL above.\n\n")
		return err
	}

	return nil
}
