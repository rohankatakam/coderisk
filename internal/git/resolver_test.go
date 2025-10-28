package git

import (
	"context"
	"testing"
)

// MockGraphClient for testing
type MockGraphClient struct {
	files map[string]bool // Files that exist in graph
}

func (m *MockGraphClient) ExecuteQuery(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	// Check if this is a single path query (exact match)
	if path, ok := params["path"].(string); ok {
		if m.files[path] {
			return []map[string]any{
				{"path": path},
			}, nil
		}
		return []map[string]any{}, nil
	}

	// Check if this is a multi-path query (git follow)
	if paths, ok := params["paths"].([]string); ok {
		var results []map[string]any
		for _, path := range paths {
			if m.files[path] {
				results = append(results, map[string]any{
					"path": path,
				})
			}
		}
		return results, nil
	}

	return []map[string]any{}, nil
}

func TestFileResolver_ExactMatch(t *testing.T) {
	mockGraph := &MockGraphClient{
		files: map[string]bool{
			"apps/web/src/app/page.tsx": true,
			"apps/mobile/src/App.tsx":   true,
		},
	}

	resolver := NewFileResolver("/test/repo", mockGraph)

	// Test exact match
	matches, err := resolver.Resolve(context.Background(), "apps/web/src/app/page.tsx")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(matches) != 1 {
		t.Fatalf("Expected 1 match, got %d", len(matches))
	}

	if matches[0].Method != "exact" {
		t.Errorf("Expected method 'exact', got '%s'", matches[0].Method)
	}

	if matches[0].Confidence != 1.0 {
		t.Errorf("Expected confidence 1.0, got %.2f", matches[0].Confidence)
	}

	if matches[0].HistoricalPath != "apps/web/src/app/page.tsx" {
		t.Errorf("Expected path 'apps/web/src/app/page.tsx', got '%s'", matches[0].HistoricalPath)
	}
}

func TestFileResolver_NoMatch(t *testing.T) {
	mockGraph := &MockGraphClient{
		files: map[string]bool{
			"existing-file.tsx": true,
		},
	}

	resolver := NewFileResolver("/test/repo", mockGraph)

	// Test with new file (no historical data)
	matches, err := resolver.Resolve(context.Background(), "brand-new-file.tsx")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(matches) != 0 {
		t.Errorf("Expected 0 matches for new file, got %d", len(matches))
	}
}

func TestFileResolver_ResolveToAllPaths(t *testing.T) {
	mockGraph := &MockGraphClient{
		files: map[string]bool{
			"apps/web/page.tsx": true,
		},
	}

	resolver := NewFileResolver("/test/repo", mockGraph)

	// Test with file that has historical path
	paths, err := resolver.ResolveToAllPaths(context.Background(), "apps/web/page.tsx")
	if err != nil {
		t.Fatalf("ResolveToAllPaths failed: %v", err)
	}

	if len(paths) != 1 {
		t.Fatalf("Expected 1 path, got %d", len(paths))
	}

	if paths[0] != "apps/web/page.tsx" {
		t.Errorf("Expected 'apps/web/page.tsx', got '%s'", paths[0])
	}
}

func TestFileResolver_ResolveToAllPaths_NewFile(t *testing.T) {
	mockGraph := &MockGraphClient{
		files: map[string]bool{},
	}

	resolver := NewFileResolver("/test/repo", mockGraph)

	// Test with new file - should return current path
	paths, err := resolver.ResolveToAllPaths(context.Background(), "new-file.tsx")
	if err != nil {
		t.Fatalf("ResolveToAllPaths failed: %v", err)
	}

	if len(paths) != 1 {
		t.Fatalf("Expected 1 path (current), got %d", len(paths))
	}

	if paths[0] != "new-file.tsx" {
		t.Errorf("Expected 'new-file.tsx', got '%s'", paths[0])
	}
}

func TestFileResolver_ResolveToSinglePath(t *testing.T) {
	mockGraph := &MockGraphClient{
		files: map[string]bool{
			"historical/path.tsx": true,
		},
	}

	resolver := NewFileResolver("/test/repo", mockGraph)

	// Test with exact match
	path, confidence, err := resolver.ResolveToSinglePath(context.Background(), "historical/path.tsx")
	if err != nil {
		t.Fatalf("ResolveToSinglePath failed: %v", err)
	}

	if path != "historical/path.tsx" {
		t.Errorf("Expected 'historical/path.tsx', got '%s'", path)
	}

	if confidence != 1.0 {
		t.Errorf("Expected confidence 1.0, got %.2f", confidence)
	}
}

func TestFileResolver_ResolveToSinglePath_NewFile(t *testing.T) {
	mockGraph := &MockGraphClient{
		files: map[string]bool{},
	}

	resolver := NewFileResolver("/test/repo", mockGraph)

	// Test with new file - should return current path with low confidence
	path, confidence, err := resolver.ResolveToSinglePath(context.Background(), "new-file.tsx")
	if err != nil {
		t.Fatalf("ResolveToSinglePath failed: %v", err)
	}

	if path != "new-file.tsx" {
		t.Errorf("Expected 'new-file.tsx', got '%s'", path)
	}

	if confidence != 0.3 {
		t.Errorf("Expected confidence 0.3 (low for new file), got %.2f", confidence)
	}
}

func TestFileResolver_ParseUniquePaths(t *testing.T) {
	resolver := &FileResolver{}

	input := []byte(`apps/web/src/page.tsx
apps/web/page.tsx
old/path/page.tsx
apps/web/src/page.tsx
`)

	paths := resolver.parseUniquePaths(input)

	expectedCount := 3 // Duplicates should be removed
	if len(paths) != expectedCount {
		t.Errorf("Expected %d unique paths, got %d", expectedCount, len(paths))
	}

	// Check that all expected paths are present
	pathSet := make(map[string]bool)
	for _, p := range paths {
		pathSet[p] = true
	}

	expectedPaths := []string{
		"apps/web/src/page.tsx",
		"apps/web/page.tsx",
		"old/path/page.tsx",
	}

	for _, expected := range expectedPaths {
		if !pathSet[expected] {
			t.Errorf("Expected path '%s' not found in results", expected)
		}
	}
}

func TestFileResolver_BatchResolve(t *testing.T) {
	mockGraph := &MockGraphClient{
		files: map[string]bool{
			"file1.tsx": true,
			"file2.tsx": true,
		},
	}

	resolver := NewFileResolver("/test/repo", mockGraph)

	// Test batch resolution
	results, err := resolver.BatchResolve(context.Background(), []string{"file1.tsx", "file2.tsx", "file3.tsx"})
	if err != nil {
		t.Fatalf("BatchResolve failed: %v", err)
	}

	// file1.tsx and file2.tsx should have matches
	if len(results["file1.tsx"]) != 1 {
		t.Errorf("Expected 1 match for file1.tsx, got %d", len(results["file1.tsx"]))
	}

	if len(results["file2.tsx"]) != 1 {
		t.Errorf("Expected 1 match for file2.tsx, got %d", len(results["file2.tsx"]))
	}

	// file3.tsx should have no matches (new file)
	if len(results["file3.tsx"]) != 0 {
		t.Errorf("Expected 0 matches for file3.tsx (new file), got %d", len(results["file3.tsx"]))
	}
}
