package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/rohankatakam/coderisk/internal/cli"
	"github.com/rohankatakam/coderisk/internal/database"
	"github.com/rohankatakam/coderisk/internal/output"
	"github.com/spf13/cobra"
)

var blameCmd = &cobra.Command{
	Use:   "blame [file_or_directory]",
	Short: "Show ownership and risk attribution for functions",
	Long: `Show ownership and risk attribution for all functions in a file or directory.

Examples:
  # Show ownership for all functions in a file
  crisk blame src/auth.ts

  # Filter to specific function
  crisk blame src/auth.ts -f "loginUser"

  # Scan directory recursively
  crisk blame src/ --recursive

  # Output as JSON
  crisk blame --format=json src/billing.go

  # Output as CSV
  crisk blame --format=csv src/auth.ts`,
	RunE: runBlame,
}

func init() {
	blameCmd.Flags().StringP("function", "f", "", "Filter to specific function")
	blameCmd.Flags().BoolP("recursive", "r", false, "Scan directory recursively")
	blameCmd.Flags().String("format", "table", "Output format: table, json, csv")
}

func runBlame(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if len(args) == 0 {
		return fmt.Errorf("must specify a file or directory path")
	}

	path := args[0]
	functionFilter, _ := cmd.Flags().GetString("function")
	recursive, _ := cmd.Flags().GetBool("recursive")
	format, _ := cmd.Flags().GetString("format")

	// Validate format
	if format != "table" && format != "json" && format != "csv" {
		return fmt.Errorf("invalid format %q, must be: table, json, or csv", format)
	}

	// Initialize database
	db, err := initPostgresSQLX()
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Detect repository
	repoID, _, err := cli.DetectRepoID(ctx, db.DB)
	if err != nil {
		return err
	}

	// Ensure repo is initialized
	if err := cli.EnsureRepoInitialized(ctx, db.DB, repoID); err != nil {
		return err
	}

	// Check if path is directory
	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	formatter := output.NewBlameFormatter(format)

	if fileInfo.IsDir() {
		return runBlameDirectory(ctx, db, repoID, path, recursive, formatter)
	}

	return runBlameFile(ctx, db, repoID, path, functionFilter, formatter)
}

func runBlameFile(ctx context.Context, db *sqlx.DB, repoID int64, filePath string, functionFilter string, formatter *output.BlameFormatter) error {
	// Convert to relative path
	relPath, err := cli.GetRelativePath(filePath)
	if err != nil {
		return err
	}

	// Get blocks for file
	blocks, err := database.GetFileBlocks(ctx, db, repoID, relPath)
	if err != nil {
		return fmt.Errorf("failed to get file blocks: %w", err)
	}

	// Apply function filter
	if functionFilter != "" {
		var filtered []database.BlockWithOwnership
		for _, block := range blocks {
			if block.BlockName == functionFilter {
				filtered = append(filtered, block)
			}
		}
		blocks = filtered

		if len(blocks) == 0 {
			return fmt.Errorf("function %q not found in %s", functionFilter, filePath)
		}
	}

	return formatter.FormatFileBlame(os.Stdout, relPath, blocks)
}

func runBlameDirectory(ctx context.Context, db *sqlx.DB, repoID int64, dirPath string, recursive bool, formatter *output.BlameFormatter) error {
	// Convert to relative path
	relPath, err := cli.GetRelativePath(dirPath)
	if err != nil {
		return err
	}

	var allBlocks []database.BlockWithOwnership

	if recursive {
		// Get all blocks in directory
		blocks, err := database.GetBlocksInDirectory(ctx, db, repoID, relPath)
		if err != nil {
			return fmt.Errorf("failed to get directory blocks: %w", err)
		}
		allBlocks = blocks
	} else {
		// Get only files directly in this directory (not subdirectories)
		files, err := getCodeFilesInDirectory(dirPath, false)
		if err != nil {
			return err
		}

		for _, file := range files {
			relFile, err := cli.GetRelativePath(file)
			if err != nil {
				continue
			}

			blocks, err := database.GetFileBlocks(ctx, db, repoID, relFile)
			if err != nil {
				continue // Skip files with errors
			}

			allBlocks = append(allBlocks, blocks...)
		}
	}

	if len(allBlocks) == 0 {
		fmt.Printf("No functions found in %s\n", dirPath)
		return nil
	}

	// Calculate statistics
	stats := database.CalculateOwnershipStats(allBlocks)

	return formatter.FormatDirectoryBlame(os.Stdout, relPath, stats)
}

// getCodeFilesInDirectory returns code files in a directory
func getCodeFilesInDirectory(dirPath string, recursive bool) ([]string, error) {
	var files []string
	codeExtensions := map[string]bool{
		".go": true, ".py": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
		".java": true, ".cpp": true, ".c": true, ".h": true, ".hpp": true,
		".rs": true, ".rb": true, ".php": true, ".cs": true, ".swift": true,
	}

	if recursive {
		err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				ext := strings.ToLower(filepath.Ext(path))
				if codeExtensions[ext] {
					files = append(files, path)
				}
			}
			return nil
		})
		return files, err
	}

	// Non-recursive: only immediate children
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if codeExtensions[ext] {
				files = append(files, filepath.Join(dirPath, entry.Name()))
			}
		}
	}

	return files, nil
}
