package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/coderisk/coderisk-go/internal/config"
	"github.com/coderisk/coderisk-go/internal/models"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

// Manager handles cache operations
type Manager struct {
	config      *config.Config
	logger      *logrus.Logger
	memCache    *cache.Cache
	localPath   string
	remoteURL   string
	isConnected bool
}

// NewManager creates a new cache manager
func NewManager(cfg *config.Config, logger *logrus.Logger) *Manager {
	// Ensure cache directory exists
	if err := os.MkdirAll(cfg.Cache.Directory, 0755); err != nil {
		logger.WithError(err).Warn("Failed to create cache directory")
	}

	return &Manager{
		config:    cfg,
		logger:    logger,
		memCache:  cache.New(5*time.Minute, 10*time.Minute),
		localPath: cfg.Cache.Directory,
	}
}

// CacheInfo contains information about a cache
type CacheInfo struct {
	URL         string
	RepoID      string
	LastUpdated time.Time
	Version     string
	Admin       string
	Size        int64
}

// QueryRegistry queries the cache registry for existing caches
func (m *Manager) QueryRegistry(ctx context.Context, repoURL string) (*CacheInfo, error) {
	// TODO: Implement actual registry query
	// This would query CodeRisk's central registry or team server
	return nil, fmt.Errorf("no cache found for %s", repoURL)
}

// Connect connects to a shared cache
func (m *Manager) Connect(ctx context.Context, url string) error {
	m.logger.WithField("url", url).Info("Connecting to shared cache")

	// TODO: Implement actual connection logic
	// This would establish connection to team cache server
	m.remoteURL = url
	m.isConnected = true

	return nil
}

// Pull synchronizes local cache with remote
func (m *Manager) Pull(ctx context.Context) error {
	if !m.isConnected {
		return fmt.Errorf("not connected to remote cache")
	}

	m.logger.Info("Pulling cache updates")

	// TODO: Implement actual pull logic
	// This would fetch updates from remote cache

	return nil
}

// LoadSketches loads risk sketches from cache
func (m *Manager) LoadSketches(ctx context.Context) ([]*models.RiskSketch, error) {
	// Try memory cache first
	if cached, found := m.memCache.Get("sketches"); found {
		return cached.([]*models.RiskSketch), nil
	}

	// Load from disk
	sketchPath := filepath.Join(m.localPath, "sketches.json")
	data, err := os.ReadFile(sketchPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("cache not initialized: run 'crisk init' first")
		}
		return nil, fmt.Errorf("failed to read sketches: %w", err)
	}

	var sketches []*models.RiskSketch
	if err := json.Unmarshal(data, &sketches); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sketches: %w", err)
	}

	// Cache in memory
	m.memCache.Set("sketches", sketches, cache.DefaultExpiration)

	return sketches, nil
}

// SaveSketches saves risk sketches to cache
func (m *Manager) SaveSketches(ctx context.Context, sketches []*models.RiskSketch) error {
	// Ensure cache directory exists
	if err := os.MkdirAll(m.localPath, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Marshal sketches
	data, err := json.MarshalIndent(sketches, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sketches: %w", err)
	}

	// Write to disk
	sketchPath := filepath.Join(m.localPath, "sketches.json")
	if err := os.WriteFile(sketchPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write sketches: %w", err)
	}

	// Update memory cache
	m.memCache.Set("sketches", sketches, cache.DefaultExpiration)

	return nil
}

// GetCacheAge returns the age of the local cache
func (m *Manager) GetCacheAge(ctx context.Context) (time.Duration, error) {
	metaPath := filepath.Join(m.localPath, "metadata.json")

	info, err := os.Stat(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("cache not initialized")
		}
		return 0, fmt.Errorf("failed to stat metadata: %w", err)
	}

	return time.Since(info.ModTime()), nil
}

// GetMetadata returns cache metadata
func (m *Manager) GetMetadata(ctx context.Context) (*models.CacheMetadata, error) {
	metaPath := filepath.Join(m.localPath, "metadata.json")

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata models.CacheMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// SaveMetadata saves cache metadata
func (m *Manager) SaveMetadata(ctx context.Context, metadata *models.CacheMetadata) error {
	// Ensure cache directory exists
	if err := os.MkdirAll(m.localPath, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metaPath := filepath.Join(m.localPath, "metadata.json")
	if err := os.WriteFile(metaPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// Clear clears the local cache
func (m *Manager) Clear(ctx context.Context) error {
	m.logger.Info("Clearing local cache")

	// Clear memory cache
	m.memCache.Flush()

	// Clear disk cache
	if err := os.RemoveAll(m.localPath); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	return nil
}

// GetSize returns the size of the local cache in bytes
func (m *Manager) GetSize(ctx context.Context) (int64, error) {
	var size int64

	err := filepath.Walk(m.localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to calculate cache size: %w", err)
	}

	return size, nil
}
