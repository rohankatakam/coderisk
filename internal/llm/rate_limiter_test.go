package llm

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Redis address - uses docker-compose setup
const testRedisAddr = "localhost:6380"

// TestRateLimiter_NewConnection tests rate limiter initialization and Redis connection
func TestRateLimiter_NewConnection(t *testing.T) {
	// Test successful connection
	rl, err := NewRateLimiter(testRedisAddr)
	require.NoError(t, err, "Should connect to Redis successfully")
	require.NotNil(t, rl, "Rate limiter should not be nil")
	assert.Equal(t, int64(DefaultRPM), rl.rpmLimit)
	assert.Equal(t, int64(DefaultTPM), rl.tpmLimit)
	assert.Equal(t, int64(DefaultRPD), rl.rpdLimit)

	// Clean up
	err = rl.Close()
	assert.NoError(t, err)
}

// TestRateLimiter_InvalidConnection tests connection to invalid Redis address
func TestRateLimiter_InvalidConnection(t *testing.T) {
	rl, err := NewRateLimiter("localhost:9999") // Invalid port
	assert.Error(t, err, "Should fail to connect to invalid Redis address")
	assert.Nil(t, rl, "Rate limiter should be nil on connection failure")
}

// TestRateLimiter_CheckAndIncrement_Normal tests normal operation under limits
func TestRateLimiter_CheckAndIncrement_Normal(t *testing.T) {
	rl, err := NewRateLimiter(testRedisAddr)
	require.NoError(t, err)
	defer rl.Close()

	ctx := context.Background()

	// Clean up any existing test keys
	cleanupTestKeys(t, rl.redis)

	// Test normal increment (well under limits)
	for i := 0; i < 10; i++ {
		err := rl.CheckAndIncrement(ctx, 100) // 100 tokens per request
		assert.NoError(t, err, "Should allow requests well under limit")
	}

	// Verify current usage
	rpm, tpm, rpd, err := rl.GetCurrentUsage(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), rpm, "Should have 10 requests in current minute")
	assert.Equal(t, int64(1000), tpm, "Should have 1000 tokens in current minute")
	assert.Equal(t, int64(10), rpd, "Should have 10 requests today")
}

// TestRateLimiter_RPMThrottle tests RPM (Requests Per Minute) throttling at 90%
func TestRateLimiter_RPMThrottle(t *testing.T) {
	rl, err := NewRateLimiter(testRedisAddr)
	require.NoError(t, err)
	defer rl.Close()

	ctx := context.Background()

	// Clean up any existing test keys
	cleanupTestKeys(t, rl.redis)

	// Make requests up to 89% of RPM limit (should succeed)
	// 90% of 1000 = 900, so we make 899 requests (89.9%)
	for i := 0; i < 899; i++ {
		err := rl.CheckAndIncrement(ctx, 10)
		assert.NoError(t, err, "Should allow requests under 90%% threshold")
	}

	// Next request brings us to exactly 900 (90%), should trigger throttle warning
	err = rl.CheckAndIncrement(ctx, 10)
	if err == nil {
		// We might be at exactly 900, try one more
		err = rl.CheckAndIncrement(ctx, 10)
	}
	assert.Error(t, err, "Should throttle at 90%% of RPM limit")
	if err != nil {
		assert.Contains(t, err.Error(), "RPM", "Error should mention RPM limit")
		assert.Contains(t, err.Error(), "wait", "Error should indicate wait time")
	}
}

// TestRateLimiter_TPMThrottle tests TPM (Tokens Per Minute) throttling at 90%
func TestRateLimiter_TPMThrottle(t *testing.T) {
	rl, err := NewRateLimiter(testRedisAddr)
	require.NoError(t, err)
	defer rl.Close()

	ctx := context.Background()

	// Clean up any existing test keys
	cleanupTestKeys(t, rl.redis)

	// Make one large request approaching TPM limit
	// 90% of 1M tokens = 900k tokens
	err = rl.CheckAndIncrement(ctx, 890_000)
	assert.NoError(t, err, "Should allow large request under 90%% TPM")

	// Next request should trigger TPM throttle
	err = rl.CheckAndIncrement(ctx, 10_000)
	assert.Error(t, err, "Should throttle when approaching TPM limit")
	assert.Contains(t, err.Error(), "TPM", "Error should mention TPM limit")
}

// TestRateLimiter_KeyExpiration tests that Redis keys expire properly
func TestRateLimiter_KeyExpiration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping time-based test in short mode")
	}

	rl, err := NewRateLimiter(testRedisAddr)
	require.NoError(t, err)
	defer rl.Close()

	ctx := context.Background()

	// Clean up any existing test keys
	cleanupTestKeys(t, rl.redis)

	// Make a request
	err = rl.CheckAndIncrement(ctx, 1000)
	assert.NoError(t, err)

	// Verify counters exist
	rpm, _, _, err := rl.GetCurrentUsage(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rpm)

	// Wait for keys to expire (70 seconds TTL + buffer)
	t.Log("Waiting 75 seconds for key expiration...")
	time.Sleep(75 * time.Second)

	// Counters should be reset (keys expired)
	rpm, tpm, rpd, err := rl.GetCurrentUsage(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), rpm, "RPM counter should be reset after expiration")
	assert.Equal(t, int64(0), tpm, "TPM counter should be reset after expiration")
	// Note: RPD has 24h TTL, so it won't be reset
	assert.LessOrEqual(t, rpd, int64(1), "RPD counter may still exist")
}

// TestRateLimiter_CheckAndIncrementWithRetry tests automatic retry logic
func TestRateLimiter_CheckAndIncrementWithRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping retry test in short mode")
	}

	rl, err := NewRateLimiter(testRedisAddr)
	require.NoError(t, err)
	defer rl.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Clean up any existing test keys
	cleanupTestKeys(t, rl.redis)

	// Fill up to just below throttle threshold (90% of 1000 RPM = 900)
	// We can safely do 899 requests before hitting the threshold
	for i := 0; i < 899; i++ {
		err := rl.CheckAndIncrement(ctx, 10)
		require.NoError(t, err)
	}

	// This should trigger throttle and wait
	// Since we have a 5 second timeout, it should fail with context deadline exceeded
	startTime := time.Now()
	err = rl.CheckAndIncrementWithRetry(ctx, 10)
	duration := time.Since(startTime)

	// Should fail due to context timeout while waiting
	assert.Error(t, err, "Should fail due to context timeout")
	assert.Greater(t, duration, 4*time.Second, "Should have waited for some time before timeout")
}

// TestRateLimiter_ConcurrentAccess tests concurrent access from multiple goroutines
func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	rl, err := NewRateLimiter(testRedisAddr)
	require.NoError(t, err)
	defer rl.Close()

	ctx := context.Background()

	// Clean up any existing test keys
	cleanupTestKeys(t, rl.redis)

	// Launch 10 concurrent goroutines each making 10 requests
	numGoroutines := 10
	numRequestsPerGoroutine := 10
	expectedTotal := numGoroutines * numRequestsPerGoroutine

	done := make(chan bool)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numRequestsPerGoroutine; j++ {
				_ = rl.CheckAndIncrement(ctx, 100)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify total count is correct (atomic operations should work)
	rpm, tpm, rpd, err := rl.GetCurrentUsage(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(expectedTotal), rpm, "Concurrent requests should be counted atomically")
	assert.Equal(t, int64(expectedTotal*100), tpm, "Token count should be accurate")
	assert.Equal(t, int64(expectedTotal), rpd, "Daily count should match")
}

// TestRateLimiter_GetCurrentUsage tests usage statistics retrieval
func TestRateLimiter_GetCurrentUsage(t *testing.T) {
	rl, err := NewRateLimiter(testRedisAddr)
	require.NoError(t, err)
	defer rl.Close()

	ctx := context.Background()

	// Clean up any existing test keys
	cleanupTestKeys(t, rl.redis)

	// Initially should be zero
	rpm, tpm, rpd, err := rl.GetCurrentUsage(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), rpm)
	assert.Equal(t, int64(0), tpm)
	assert.Equal(t, int64(0), rpd)

	// Make some requests
	for i := 0; i < 5; i++ {
		err := rl.CheckAndIncrement(ctx, 200)
		assert.NoError(t, err)
	}

	// Verify usage is tracked
	rpm, tpm, rpd, err = rl.GetCurrentUsage(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), rpm)
	assert.Equal(t, int64(1000), tpm)
	assert.Equal(t, int64(5), rpd)
}

// TestRateLimiter_extractWaitTime tests wait time extraction from error messages
func TestRateLimiter_extractWaitTime(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected int
	}{
		{
			name:     "Valid wait time",
			errMsg:   "approaching RPM limit (950/1000), wait 45s",
			expected: 45,
		},
		{
			name:     "Single digit wait",
			errMsg:   "approaching TPM limit (900000/1000000), wait 5s",
			expected: 5,
		},
		{
			name:     "Invalid format",
			errMsg:   "some other error",
			expected: 60, // Default fallback
		},
		{
			name:     "No wait time",
			errMsg:   "approaching limit but no time specified",
			expected: 60, // Default fallback
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractWaitTime(tt.errMsg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// testLogger is an interface that both *testing.T and *testing.B implement
type testLogger interface {
	Logf(format string, args ...interface{})
}

// cleanupTestKeys removes all test keys from Redis
func cleanupTestKeys(t testLogger, client *redis.Client) {
	ctx := context.Background()

	// Get all gemini:* keys
	keys, err := client.Keys(ctx, "gemini:*").Result()
	if err != nil {
		t.Logf("Warning: Failed to list Redis keys: %v", err)
		return
	}

	// Delete all keys
	if len(keys) > 0 {
		err = client.Del(ctx, keys...).Err()
		if err != nil {
			t.Logf("Warning: Failed to delete Redis keys: %v", err)
		} else {
			t.Logf("Cleaned up %d Redis keys", len(keys))
		}
	}
}

// Benchmark tests

// BenchmarkRateLimiter_CheckAndIncrement benchmarks the rate limiter performance
func BenchmarkRateLimiter_CheckAndIncrement(b *testing.B) {
	rl, err := NewRateLimiter(testRedisAddr)
	if err != nil {
		b.Fatalf("Failed to create rate limiter: %v", err)
	}
	defer rl.Close()

	ctx := context.Background()

	// Clean up before benchmark
	cleanupTestKeys(b, rl.redis)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = rl.CheckAndIncrement(ctx, 100)
	}
}

// Example usage test
func ExampleRateLimiter_CheckAndIncrement() {
	// Create rate limiter
	rl, err := NewRateLimiter("localhost:6380")
	if err != nil {
		fmt.Printf("Failed to create rate limiter: %v\n", err)
		return
	}
	defer rl.Close()

	ctx := context.Background()

	// Check rate limit before making API call
	err = rl.CheckAndIncrement(ctx, 1000) // Estimated 1000 tokens
	if err != nil {
		fmt.Printf("Rate limit exceeded: %v\n", err)
		return
	}

	fmt.Println("Rate limit OK, proceeding with API call")
	// Output: Rate limit OK, proceeding with API call
}
