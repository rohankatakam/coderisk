package auth

// Embedded cloud service configuration
// These are PUBLIC keys - safe to embed in binary
// Security: RLS policies in Supabase protect user data
const (
	// Clerk configuration (OAuth authentication)
	ClerkPublishableKey = "pk_test_ZnJhbmstYXNwLTIxLmNsZXJrLmFjY291bnRzLmRldiQ"

	// Supabase configuration (database + RLS)
	SupabaseURL     = "https://ehjmjwzlrwvdyaywsgyp.supabase.co"
	SupabaseAnonKey = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImVoam1qd3pscnd2ZHlheXdzZ3lwIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NjA2MjU2OTAsImV4cCI6MjA3NjIwMTY5MH0.x_DpzBopIfCMqgSmUPgo6z_s73lIMc32trSbdTUBt5M"

	// Frontend URLs
	AuthCallbackURL = "https://coderisk.dev/cli-auth"
)
