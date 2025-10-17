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
	// TODO: Implement proper device flow
	// For MVP, we'll guide users to the manual flow
	return nil, fmt.Errorf("automatic device flow coming soon - use manual setup for now")
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
