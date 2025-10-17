package auth

import (
	"context"
	"fmt"
)

// Manager handles authentication and credential management
type Manager struct {
	clerkClient    *ClerkClient
	supabaseClient *SupabaseClient
	token          *Token
}

// NewManager creates a new authentication manager
func NewManager() (*Manager, error) {
	return &Manager{
		clerkClient: NewClerkClient(),
	}, nil
}

// LoadSession loads the current session from disk
func (m *Manager) LoadSession() error {
	token, err := LoadToken()
	if err != nil {
		return err
	}

	m.token = token

	// Initialize Supabase client with token
	supabaseClient, err := NewSupabaseClient(token.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to initialize Supabase client: %w", err)
	}

	m.supabaseClient = supabaseClient
	return nil
}

// GetCredentials fetches credentials from Supabase
func (m *Manager) GetCredentials() (*Credentials, error) {
	if m.token == nil {
		return nil, fmt.Errorf("not authenticated. Run: crisk login")
	}

	if m.supabaseClient == nil {
		return nil, fmt.Errorf("supabase client not initialized")
	}

	return m.supabaseClient.GetCredentials(m.token.User.ID)
}

// PostUsage posts usage telemetry to Supabase
func (m *Manager) PostUsage(command string, fileCount, nodeCount, openaiTokens int, costUSD float64) error {
	if m.token == nil || m.supabaseClient == nil {
		// Silently skip telemetry if not authenticated
		return nil
	}

	usage := &Usage{
		UserID:       m.token.User.ID,
		Command:      command,
		FileCount:    fileCount,
		NodeCount:    nodeCount,
		OpenAITokens: openaiTokens,
		CostUSD:      costUSD,
	}

	return m.supabaseClient.PostUsage(usage)
}

// GetUsageStats retrieves usage statistics
func (m *Manager) GetUsageStats() (*UsageStats, error) {
	if m.token == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	if m.supabaseClient == nil {
		return nil, fmt.Errorf("supabase client not initialized")
	}

	return m.supabaseClient.GetUsageStats(m.token.User.ID)
}

// GetUser returns the current authenticated user
func (m *Manager) GetUser() (*User, error) {
	if m.token == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	return &m.token.User, nil
}

// GetToken returns the current token
func (m *Manager) GetToken() (*Token, error) {
	if m.token == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	return m.token, nil
}

// VerifyAuthentication checks if the user is authenticated and session is valid
func (m *Manager) VerifyAuthentication(ctx context.Context) error {
	if m.token == nil {
		if err := m.LoadSession(); err != nil {
			return err
		}
	}

	// Verify session with Clerk
	_, err := m.clerkClient.VerifySession(ctx, m.token.AccessToken)
	if err != nil {
		return fmt.Errorf("session verification failed: %w. Try running: crisk login", err)
	}

	return nil
}

// Logout removes the authentication token
func (m *Manager) Logout() error {
	if err := DeleteToken(); err != nil {
		return err
	}

	m.token = nil
	m.supabaseClient = nil

	return nil
}
