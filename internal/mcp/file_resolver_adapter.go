package mcp

import (
	"context"

	"github.com/rohankatakam/coderisk/internal/git"
	"github.com/rohankatakam/coderisk/internal/mcp/tools"
)

// FileResolverAdapter adapts git.FileResolver to the tools.IdentityResolver interface
// This bridges the proven git.FileResolver (used in `crisk check`) to the MCP server
type FileResolverAdapter struct {
	resolver *git.FileResolver
	repoRoot string // Default repo root for when none is provided
}

// NewFileResolverAdapter creates a new adapter
func NewFileResolverAdapter(resolver *git.FileResolver, defaultRepoRoot string) tools.IdentityResolver {
	return &FileResolverAdapter{
		resolver: resolver,
		repoRoot: defaultRepoRoot,
	}
}

// ResolveHistoricalPaths implements tools.IdentityResolver using the default repo root
func (a *FileResolverAdapter) ResolveHistoricalPaths(ctx context.Context, currentPath string) ([]string, error) {
	return a.resolver.ResolveToAllPaths(ctx, currentPath)
}

// ResolveHistoricalPathsWithRoot implements tools.IdentityResolver with a custom repo root
// Note: git.FileResolver was initialized with a repo root, so this parameter is currently unused
// The resolver will use the repo root it was constructed with
func (a *FileResolverAdapter) ResolveHistoricalPathsWithRoot(ctx context.Context, currentPath string, repoRoot string) ([]string, error) {
	// The git.FileResolver doesn't support dynamic repo root changes after construction
	// For now, we use the same resolver instance regardless of the repoRoot parameter
	// This is acceptable because in practice, the MCP server analyzes a single repository
	// TODO: If multi-repo support is needed, create a resolver pool keyed by repoRoot
	return a.resolver.ResolveToAllPaths(ctx, currentPath)
}
