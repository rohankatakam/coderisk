package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps Redis client with caching helpers
// Reference: agentic_design.md §4.1 - Caching strategy
type Client struct {
	client *redis.Client
	logger *slog.Logger
	ttl    time.Duration // Default TTL for cached items
}

// NewClient creates a Redis client from connection parameters
// Security: NEVER hardcode credentials (DEVELOPMENT_WORKFLOW.md §3.3)
// Reference: local_deployment.md - Redis configuration
func NewClient(ctx context.Context, host string, port int, password string) (*Client, error) {
	if host == "" {
		return nil, fmt.Errorf("redis host missing")
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // Empty string if no password
		DB:       0,        // Use default DB
	})

	// Verify connectivity (fail fast on startup)
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis at %s: %w", addr, err)
	}

	logger := slog.Default().With("component", "redis")
	logger.Info("redis client connected", "addr", addr)

	// Default TTL: 15 minutes (per agentic_design.md §4.1)
	return &Client{
		client: client,
		logger: logger,
		ttl:    15 * time.Minute,
	}, nil
}

// Close closes the Redis client connection
func (c *Client) Close() error {
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("failed to close redis client: %w", err)
	}
	c.logger.Info("redis client closed")
	return nil
}

// HealthCheck verifies Redis connectivity
// Used by API health endpoint
func (c *Client) HealthCheck(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}
	return nil
}

// Get retrieves a cached value by key and unmarshals into target
// Returns: true if found, false if miss (not an error)
// Reference: agentic_design.md §4.1 - Cache hit optimization
func (c *Client) Get(ctx context.Context, key string, target interface{}) (bool, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		// Cache miss - not an error
		c.logger.Debug("cache miss", "key", key)
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("redis get failed for key %s: %w", key, err)
	}

	// Unmarshal JSON into target
	if err := json.Unmarshal([]byte(val), target); err != nil {
		return false, fmt.Errorf("failed to unmarshal cached value for key %s: %w", key, err)
	}

	c.logger.Debug("cache hit", "key", key)
	return true, nil
}

// Set stores a value in cache with default TTL (15 minutes)
// Value is marshaled to JSON before storage
// Reference: agentic_design.md §4.1 - 15-minute TTL
func (c *Client) Set(ctx context.Context, key string, value interface{}) error {
	return c.SetWithTTL(ctx, key, value, c.ttl)
}

// SetWithTTL stores a value in cache with custom TTL
// Value is marshaled to JSON before storage
func (c *Client) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// Marshal value to JSON
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
	}

	// Store in Redis with TTL
	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set failed for key %s: %w", key, err)
	}

	c.logger.Debug("cache set", "key", key, "ttl", ttl)
	return nil
}

// Delete removes a key from cache
func (c *Client) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis delete failed for key %s: %w", key, err)
	}

	c.logger.Debug("cache delete", "key", key)
	return nil
}

// DeletePattern deletes all keys matching a pattern
// Example: DeletePattern(ctx, "baseline:repo123:*") removes all baseline cache for repo123
// Reference: DEVELOPMENT_WORKFLOW.md §3.1 - Input validation (prevent wildcard abuse)
func (c *Client) DeletePattern(ctx context.Context, pattern string) (int64, error) {
	// Scan for matching keys
	var cursor uint64
	var keys []string

	for {
		var batch []string
		var err error
		batch, cursor, err = c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return 0, fmt.Errorf("redis scan failed for pattern %s: %w", pattern, err)
		}

		keys = append(keys, batch...)

		if cursor == 0 {
			break
		}
	}

	// Delete all matching keys
	if len(keys) == 0 {
		c.logger.Debug("no keys matched pattern", "pattern", pattern)
		return 0, nil
	}

	deleted, err := c.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf("redis delete failed for pattern %s: %w", pattern, err)
	}

	c.logger.Info("cache pattern delete", "pattern", pattern, "deleted", deleted)
	return deleted, nil
}

// CacheKey generates a standardized cache key
// Format: "prefix:repo_id:file_path" for baseline metrics
// Example: "baseline:repo123:src/main.go"
// Reference: agentic_design.md §4.1 - Cache key structure
func CacheKey(prefix, repoID, filePath string) string {
	return fmt.Sprintf("%s:%s:%s", prefix, repoID, filePath)
}

// BaselineCacheKey generates cache key for Phase 1 baseline metrics
// Reference: risk_assessment_methodology.md §2 - Tier 1 metrics
func BaselineCacheKey(repoID, filePath string) string {
	return CacheKey("baseline", repoID, filePath)
}
