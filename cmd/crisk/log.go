package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/rohankatakam/coderisk/internal/cli"
	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/output"
	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log [file]",
	Short: "Show semantic git history for a function across renames",
	Long: `Show semantic git history for a function across renames and refactors.

Unlike 'git log', crisk log operates at the function level and annotates
commits with incident history, ownership changes, and complexity evolution.

Examples:
  # Show history of specific function
  crisk log -f "loginUser" src/auth.ts

  # Direct block ID lookup
  crisk log --block-id "uuid-here"

  # Limit to last 20 changes
  crisk log -n 20 -f "processPayment" billing.go

  # Compact one-line format
  crisk log --oneline -f "validateToken" auth.ts`,
	RunE: runLog,
}

func init() {
	logCmd.Flags().StringP("function", "f", "", "Function name to analyze")
	logCmd.Flags().StringP("block-id", "b", "", "Block UUID for direct lookup")
	logCmd.Flags().IntP("limit", "n", 50, "Max commits to show")
	logCmd.Flags().Bool("oneline", false, "Compact one-line format")
	logCmd.Flags().BoolP("verbose", "v", false, "Show full commit messages")
}

func runLog(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get flags
	functionName, _ := cmd.Flags().GetString("function")
	blockIDStr, _ := cmd.Flags().GetString("block-id")
	limit, _ := cmd.Flags().GetInt("limit")
	oneline, _ := cmd.Flags().GetBool("oneline")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Initialize database
	db, err := initPostgresSQLX()
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	var blockID int64
	var block *database.BlockWithHistory

	// Resolve block ID
	if blockIDStr != "" {
		// Direct ID lookup
		blockID, err = strconv.ParseInt(blockIDStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid block ID: %w", err)
		}

		block, err = database.GetBlockWithRenameChain(ctx, db, blockID)
		if err != nil {
			return err
		}

	} else if functionName != "" && len(args) > 0 {
		// Function name + file path
		filePath := args[0]

		// Detect repository
		repoID, _, err := cli.DetectRepoID(ctx, db.DB)
		if err != nil {
			return err
		}

		// Ensure repo is initialized
		if err := cli.EnsureRepoInitialized(ctx, db.DB, repoID); err != nil {
			return err
		}

		// Resolve block by name
		blocks, err := cli.ResolveBlockByName(ctx, db.DB, repoID, functionName, filePath)
		if err != nil {
			return err
		}

		if len(blocks) == 0 {
			return cli.HandleBlockNotFound(ctx, db.DB, repoID, functionName, filePath)
		}

		if len(blocks) > 1 {
			// Ambiguous - show options
			fmt.Fprintf(os.Stderr, "Multiple functions named %q found in %s:\n", functionName, filePath)
			for i, b := range blocks {
				fmt.Fprintf(os.Stderr, "  %d. %s (line %d-%d) %s\n",
					i+1, b.BlockName, b.StartLine, b.EndLine, b.Signature)
			}
			return fmt.Errorf("please specify which function using line numbers or use --block-id")
		}

		// Single match
		blockID = blocks[0].ID

		// Get full block with rename chain
		block, err = database.GetBlockWithRenameChain(ctx, db, blockID)
		if err != nil {
			return err
		}

	} else {
		return fmt.Errorf("must specify either --function with file path, or --block-id")
	}

	// Get block history
	events, err := database.GetBlockHistory(ctx, db, blockID, limit)
	if err != nil {
		return fmt.Errorf("failed to get block history: %w", err)
	}

	if len(events) == 0 {
		fmt.Printf("No history found for %s\n", block.BlockName)
		return nil
	}

	// Format output
	formatter := output.NewLogFormatter(oneline, verbose)
	return formatter.FormatBlockHistory(os.Stdout, block, events)
}
