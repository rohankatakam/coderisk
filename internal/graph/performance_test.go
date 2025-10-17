package graph

import (
	"context"
	"testing"
	"time"
)

// TestPerformanceBaselines verifies critical queries meet performance targets
// Reference: NEO4J_PERFORMANCE_OPTIMIZATION_GUIDE.md Phase 8
//
// Run with: go test -v -run TestPerformanceBaselines ./internal/graph
func TestPerformanceBaselines(t *testing.T) {
	// Skip in short mode (requires Neo4j connection)
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	// TODO: Set up test Neo4j connection
	// This test requires a populated Neo4j database
	t.Skip("requires Neo4j test database - implement when test infrastructure ready")

	ctx := context.Background()
	profiler := NewPerformanceProfiler()

	// Test Tier 1 metric performance
	t.Run("QueryCoupling", func(t *testing.T) {
		maxDuration := 150 * time.Millisecond

		// TODO: Execute actual query
		_ = ctx
		_ = profiler
		_ = maxDuration

		// Example once test infrastructure is ready:
		// _, err := profiler.Profile(ctx, "QueryCoupling", "test query", func() (any, error) {
		//     return client.QueryCoupling(ctx, "test/file.go")
		// })
		// if err != nil {
		//     t.Fatal(err)
		// }
		//
		// stats := profiler.GetStats("QueryCoupling")
		// if stats.AvgDuration > maxDuration {
		//     t.Errorf("QueryCoupling exceeded baseline: %v > %v", stats.AvgDuration, maxDuration)
		// }
	})
}

// BenchmarkQueryCoupling benchmarks the coupling query
// Run with: go test -bench=BenchmarkQueryCoupling -benchmem ./internal/graph
func BenchmarkQueryCoupling(b *testing.B) {
	// Skip if no test database available
	b.Skip("requires Neo4j test database")

	// TODO: Set up test database
	ctx := context.Background()
	_ = ctx

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// TODO: Execute query
		// _, _ = client.QueryCoupling(ctx, "test/file.go")
	}
}

// BenchmarkBatchCreate benchmarks batch node creation
func BenchmarkBatchCreate(b *testing.B) {
	// Skip if no test database available
	b.Skip("requires Neo4j test database")

	ctx := context.Background()
	_ = ctx

	// Create test nodes
	nodes := make([]GraphNode, 100)
	for i := 0; i < 100; i++ {
		nodes[i] = GraphNode{
			Label: "File",
			Properties: map[string]any{
				"file_path": "test/file.go",
				"loc":       100,
			},
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// TODO: Execute batch create
		// _, _ = backend.CreateNodes(ctx, nodes)
	}
}

// TestRegressionDetection tests the regression detector
func TestRegressionDetection(t *testing.T) {
	detector := NewRegressionDetector()

	// Test case 1: Within baseline
	profile1 := PerformanceProfile{
		Operation:    "QueryCoupling",
		Duration:     100 * time.Millisecond,
		RecordsCount: 50,
	}

	isRegression, _ := detector.Check(profile1)
	if isRegression {
		t.Error("Expected no regression for profile within baseline")
	}

	// Test case 2: Exceeds duration baseline
	profile2 := PerformanceProfile{
		Operation:    "QueryCoupling",
		Duration:     200 * time.Millisecond, // Exceeds 150ms baseline
		RecordsCount: 50,
	}

	isRegression, message := detector.Check(profile2)
	if !isRegression {
		t.Error("Expected regression for profile exceeding duration baseline")
	}
	if message == "" {
		t.Error("Expected regression message")
	}

	// Test case 3: Unknown operation (no baseline)
	profile3 := PerformanceProfile{
		Operation:    "UnknownOperation",
		Duration:     5 * time.Second,
		RecordsCount: 10000,
	}

	isRegression, _ = detector.Check(profile3)
	if isRegression {
		t.Error("Expected no regression for unknown operation (no baseline)")
	}
}

// TestPerformanceProfiler tests the profiler functionality
func TestPerformanceProfiler(t *testing.T) {
	profiler := NewPerformanceProfiler()

	// Profile a fast operation
	_, err := profiler.Profile(context.Background(), "test_op", "SELECT 1", func() (any, error) {
		time.Sleep(10 * time.Millisecond)
		return 42, nil
	})

	if err != nil {
		t.Fatal(err)
	}

	// Check profiles were recorded
	profiles := profiler.GetProfiles()
	if len(profiles) != 1 {
		t.Errorf("Expected 1 profile, got %d", len(profiles))
	}

	// Check stats
	stats := profiler.GetStats("test_op")
	if stats == nil {
		t.Fatal("Expected stats for test_op")
	}

	if stats.SampleCount != 1 {
		t.Errorf("Expected 1 sample, got %d", stats.SampleCount)
	}

	if stats.AvgDuration < 10*time.Millisecond {
		t.Errorf("Expected duration >= 10ms, got %v", stats.AvgDuration)
	}
}

// TestPerformanceStats tests stats calculation
func TestPerformanceStats(t *testing.T) {
	profiler := NewPerformanceProfiler()

	// Add multiple profiles
	for i := 0; i < 5; i++ {
		duration := time.Duration(i+1) * 10 * time.Millisecond
		profiler.profiles = append(profiler.profiles, PerformanceProfile{
			Operation:    "test_op",
			Duration:     duration,
			RecordsCount: i * 10,
		})
	}

	stats := profiler.GetStats("test_op")

	if stats.SampleCount != 5 {
		t.Errorf("Expected 5 samples, got %d", stats.SampleCount)
	}

	if stats.MinDuration != 10*time.Millisecond {
		t.Errorf("Expected min duration 10ms, got %v", stats.MinDuration)
	}

	if stats.MaxDuration != 50*time.Millisecond {
		t.Errorf("Expected max duration 50ms, got %v", stats.MaxDuration)
	}

	// Average of 10,20,30,40,50 = 30ms
	expectedAvg := 30 * time.Millisecond
	if stats.AvgDuration != expectedAvg {
		t.Errorf("Expected avg duration %v, got %v", expectedAvg, stats.AvgDuration)
	}
}
