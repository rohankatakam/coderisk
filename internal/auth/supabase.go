package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SupabaseClient wraps the Supabase client for database operations
// Simplified for MVP - will be enhanced with proper API calls
type SupabaseClient struct {
	url     string
	anonKey string
	jwt     string
}

// Credentials represents user credentials stored in Supabase
type Credentials struct {
	OpenAIAPIKey string `json:"openai_api_key"`
	GitHubToken  string `json:"github_token"`
}

// Usage represents usage telemetry data
type Usage struct {
	UserID       string    `json:"user_id"`
	Command      string    `json:"command"`
	FileCount    int       `json:"file_count,omitempty"`
	NodeCount    int       `json:"node_count,omitempty"`
	OpenAITokens int       `json:"openai_tokens,omitempty"`
	CostUSD      float64   `json:"cost_usd,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// UsageStats represents aggregated usage statistics
type UsageStats struct {
	TotalCommands int     `json:"total_commands"`
	TotalTokens   int     `json:"total_tokens"`
	TotalCost     float64 `json:"total_cost"`
}

// NewSupabaseClient creates a new Supabase client with authentication
func NewSupabaseClient(jwt string) (*SupabaseClient, error) {
	// For MVP, we'll use a simplified client
	// TODO: Integrate full supabase-go library with proper error handling
	return &SupabaseClient{
		url:     SupabaseURL,
		anonKey: SupabaseAnonKey,
		jwt:     jwt,
	}, nil
}

// GetCredentials fetches the user's API keys from Supabase
// RLS policy ensures users only get their own credentials
func (s *SupabaseClient) GetCredentials(userID string) (*Credentials, error) {
	// Make HTTP GET request to Supabase REST API
	// URL format: {supabase_url}/rest/v1/user_credentials?user_id=eq.{user_id}
	url := fmt.Sprintf("%s/rest/v1/user_credentials?user_id=eq.%s&select=openai_api_key,github_token", s.url, userID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers for Supabase authentication
	req.Header.Set("apikey", s.anonKey)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.jwt))
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var results []Credentials
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Return first result (RLS ensures only user's own credentials)
	if len(results) == 0 {
		// No credentials found - return empty credentials
		return &Credentials{
			OpenAIAPIKey: "",
			GitHubToken:  "",
		}, nil
	}

	return &results[0], nil
}

// UpdateCredentials updates the user's API keys in Supabase
func (s *SupabaseClient) UpdateCredentials(userID string, creds *Credentials) error {
	// TODO: Implement actual Supabase API call
	return fmt.Errorf("not implemented - use web dashboard at https://coderisk.dev/dashboard/settings")
}

// PostUsage posts usage telemetry to Supabase
func (s *SupabaseClient) PostUsage(usage *Usage) error {
	// Set timestamp if not already set
	if usage.Timestamp.IsZero() {
		usage.Timestamp = time.Now()
	}

	// Make HTTP POST request to Supabase REST API
	url := fmt.Sprintf("%s/rest/v1/usage_logs", s.url)

	// Marshal usage data to JSON
	jsonData, err := json.Marshal(usage)
	if err != nil {
		return fmt.Errorf("failed to marshal usage data: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers for Supabase authentication
	req.Header.Set("apikey", s.anonKey)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.jwt))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=minimal") // Don't need response body

	// Make the request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status (201 Created or 200 OK)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetUsageStats retrieves aggregated usage statistics for the current month
func (s *SupabaseClient) GetUsageStats(userID string) (*UsageStats, error) {
	// Calculate start of current month
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Make HTTP GET request to Supabase REST API with aggregation
	// URL format: {supabase_url}/rest/v1/rpc/get_usage_stats
	url := fmt.Sprintf("%s/rest/v1/rpc/get_usage_stats", s.url)

	// Prepare request body
	requestBody := map[string]interface{}{
		"p_user_id":     userID,
		"p_start_date":  startOfMonth.Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers for Supabase authentication
	req.Header.Set("apikey", s.anonKey)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.jwt))
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// If RPC doesn't exist yet, return zeros gracefully
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
			return &UsageStats{
				TotalCommands: 0,
				TotalTokens:   0,
				TotalCost:     0,
			}, nil
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var stats UsageStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &stats, nil
}

// HealthCheck verifies the Supabase connection is working
func (s *SupabaseClient) HealthCheck() error {
	// TODO: Implement actual health check
	// For MVP, always return success
	return nil
}
