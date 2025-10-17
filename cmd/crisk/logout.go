package main

import (
	"fmt"

	"github.com/rohankatakam/coderisk/internal/auth"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Sign out of CodeRisk cloud services",
	Long: `Sign out of CodeRisk cloud services and remove local authentication token.

This will:
- Delete your local authentication token
- Clear any cached credentials
- Require you to run 'crisk login' again to use cloud features

Note: Your code analysis data remains local and is not affected.`,
	RunE: runLogout,
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}

func runLogout(cmd *cobra.Command, args []string) error {
	// Check if authenticated
	if !auth.IsAuthenticated() {
		fmt.Println("⚠️  Not currently authenticated")
		return nil
	}

	// Get current user info before logout
	token, err := auth.LoadToken()
	if err == nil {
		fmt.Printf("Signing out %s...\n", token.User.Email)
	}

	// Delete token
	if err := auth.DeleteToken(); err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	fmt.Println()
	fmt.Println("✓ Successfully signed out")
	fmt.Println()
	fmt.Println("Run 'crisk login' to sign in again")
	fmt.Println()

	return nil
}
