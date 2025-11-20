package llm

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter provides proactive rate limiting for Gemini API using Redis
// Prevents quota exhaustion by checking global counters before API calls
type RateLimiter struct {
	redis    *redis.Client
	rpmLimit int64 // Requests Per Minute
	tpmLimit int64 // Tokens Per Minute
	rpdLimit int64 // Requests Per Day
}

// Gemini Tier 1 Limits (Free tier with gemini-2.0-flash)
// Reference: https://ai.google.dev/pricing#1_5flash
const (
	DefaultRPM = 1000      // Requests per minute
	DefaultTPM = 1_000_000 // Tokens per minute (input + output combined)
	DefaultRPD = 10_000    // Requests per day
)

// NewRateLimiter creates a new rate limiter connected to Redis
// redisAddr: Redis server address (e.g., "localhost:6380")
// Returns error if Redis connection fails
func NewRateLimiter(redisAddr string) (*RateLimiter, error) {
	// Connect to Redis
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   0, // Default database
	})

	// Test connection with ping
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", redisAddr, err)
	}

	return &RateLimiter{
		redis:    client,
		rpmLimit: DefaultRPM,
		tpmLimit: DefaultTPM,
		rpdLimit: DefaultRPD,
	}, nil
}

// CheckAndIncrement checks if we're approaching rate limits and increments counters
// Uses atomic Lua script for consistency across multiple processes
// Returns error if we should throttle (approaching 90% of any limit)
func (r *RateLimiter) CheckAndIncrement(ctx context.Context, estimatedTokens int64) error {
	now := time.Now()

	// Generate time-based keys
	// Format: "gemini:rpm:2025-11-19T14:23" (per minute)
	minuteKey := fmt.Sprintf("gemini:rpm:%s", now.Format("2006-01-02T15:04"))
	tpmKey := fmt.Sprintf("gemini:tpm:%s", now.Format("2006-01-02T15:04"))
	dayKey := fmt.Sprintf("gemini:rpd:%s", now.Format("2006-01-02"))

	// Lua script for atomic increment and threshold check
	// This ensures no race conditions between checking and incrementing
	script := redis.NewScript(`
		local rpm_key = KEYS[1]
		local tpm_key = KEYS[2]
		local rpd_key = KEYS[3]
		local rpm_limit = tonumber(ARGV[1])
		local tpm_limit = tonumber(ARGV[2])
		local rpd_limit = tonumber(ARGV[3])
		local tokens = tonumber(ARGV[4])

		-- Increment counters atomically
		local rpm = redis.call('INCR', rpm_key)
		local tpm = redis.call('INCRBY', tpm_key, tokens)
		local rpd = redis.call('INCR', rpd_key)

		-- Set TTLs if keys are new (first increment)
		-- 70 seconds for minute keys (10s buffer for clock skew)
		-- 86400 seconds (24h) for daily keys
		if rpm == 1 then redis.call('EXPIRE', rpm_key, 70) end
		if tpm == tokens then redis.call('EXPIRE', tpm_key, 70) end
		if rpd == 1 then redis.call('EXPIRE', rpd_key, 86400) end

		-- Check thresholds (90% for proactive throttling, 100% for daily)
		-- We use 90% threshold to prevent hitting limits (proactive)
		if rpm >= rpm_limit * 0.9 then
			return {-1, 'RPM', rpm, rpm_limit}
		end
		if tpm >= tpm_limit * 0.9 then
			return {-2, 'TPM', tpm, tpm_limit}
		end
		if rpd >= rpd_limit then
			return {-3, 'RPD', rpd, rpd_limit}
		end

		-- Success: return current counter values
		return {0, 'OK', rpm, tpm, rpd}
	`)

	// Execute Lua script with keys and arguments
	result, err := script.Run(ctx, r.redis,
		[]string{minuteKey, tpmKey, dayKey},
		r.rpmLimit, r.tpmLimit, r.rpdLimit, estimatedTokens).Result()

	if err != nil {
		return fmt.Errorf("rate limiter Redis operation failed: %w", err)
	}

	// Parse result from Lua script
	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) < 2 {
		return fmt.Errorf("invalid rate limiter response format")
	}

	code := resultSlice[0].(int64)

	// Check if we hit a threshold
	if code < 0 {
		limitType := resultSlice[1].(string)
		current := resultSlice[2].(int64)
		limit := resultSlice[3].(int64)

		// Calculate wait time until next minute (for RPM/TPM) or next day (for RPD)
		var waitTime int
		if code == -3 {
			// Daily quota exceeded - calculate wait until midnight
			tomorrow := now.Add(24 * time.Hour)
			midnight := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, tomorrow.Location())
			waitTime = int(midnight.Sub(now).Seconds())
			return fmt.Errorf("daily quota exceeded: %d/%d requests (resets in %ds)", current, limit, waitTime)
		}

		// For RPM/TPM limits, wait until next minute
		waitTime = 60 - now.Second()
		if waitTime <= 0 {
			waitTime = 1 // Minimum 1 second
		}

		return fmt.Errorf("approaching %s limit (%d/%d), wait %ds", limitType, current, limit, waitTime)
	}

	// Success - we're under all thresholds
	return nil
}

// CheckAndIncrementWithRetry checks rate limits with automatic retry/wait logic
// This is a convenience method that wraps CheckAndIncrement with retry behavior
// Blocks until rate limit window resets, respecting context cancellation
func (r *RateLimiter) CheckAndIncrementWithRetry(ctx context.Context, estimatedTokens int64) error {
	for {
		err := r.CheckAndIncrement(ctx, estimatedTokens)
		if err == nil {
			return nil // Success - proceed with API call
		}

		// Check if it's a daily quota error (fatal, don't retry)
		if strings.Contains(err.Error(), "daily quota exceeded") {
			return err
		}

		// Check if it's a threshold warning (temporary, can retry)
		if strings.Contains(err.Error(), "wait") {
			// Extract wait time from error message
			waitTime := extractWaitTime(err.Error())

			// Log throttling
			// Using fmt.Printf since we don't have a logger here
			// In production, this would use slog
			fmt.Printf("[WARN] Rate limit approaching, throttling for %ds: %s\n", waitTime, err.Error())

			// Wait for the specified duration
			select {
			case <-time.After(time.Duration(waitTime) * time.Second):
				continue // Retry after wait
			case <-ctx.Done():
				return ctx.Err() // Context cancelled
			}
		}

		// Other errors (Redis failure, etc.)
		return err
	}
}

// extractWaitTime parses wait time from error message
// Expected format: "... wait 45s"
func extractWaitTime(errMsg string) int {
	// Use regex to extract "wait Xds" pattern
	re := regexp.MustCompile(`wait (\d+)s`)
	matches := re.FindStringSubmatch(errMsg)
	if len(matches) > 1 {
		waitTime, err := strconv.Atoi(matches[1])
		if err == nil && waitTime > 0 {
			return waitTime
		}
	}
	// Default fallback: wait 60 seconds (one full minute)
	return 60
}

// Close closes the Redis connection
func (r *RateLimiter) Close() error {
	if r.redis != nil {
		return r.redis.Close()
	}
	return nil
}

// GetCurrentUsage returns current usage statistics (for monitoring/debugging)
// Returns (rpm, tpm, rpd, error)
func (r *RateLimiter) GetCurrentUsage(ctx context.Context) (int64, int64, int64, error) {
	now := time.Now()

	minuteKey := fmt.Sprintf("gemini:rpm:%s", now.Format("2006-01-02T15:04"))
	tpmKey := fmt.Sprintf("gemini:tpm:%s", now.Format("2006-01-02T15:04"))
	dayKey := fmt.Sprintf("gemini:rpd:%s", now.Format("2006-01-02"))

	// Get all values in a pipeline for efficiency
	pipe := r.redis.Pipeline()
	rpmCmd := pipe.Get(ctx, minuteKey)
	tpmCmd := pipe.Get(ctx, tpmKey)
	rpdCmd := pipe.Get(ctx, dayKey)

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return 0, 0, 0, fmt.Errorf("failed to get usage stats: %w", err)
	}

	// Parse results (default to 0 if key doesn't exist)
	rpm, _ := rpmCmd.Int64()
	tpm, _ := tpmCmd.Int64()
	rpd, _ := rpdCmd.Int64()

	return rpm, tpm, rpd, nil
}
