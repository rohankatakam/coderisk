package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds all configuration settings
type Config struct {
	// Deployment mode
	Mode string `yaml:"mode"` // "enterprise", "team", "oss", "local"

	// Storage configuration
	Storage StorageConfig `yaml:"storage"`

	// GitHub configuration
	GitHub GitHubConfig `yaml:"github"`

	// Cache configuration
	Cache CacheConfig `yaml:"cache"`

	// API configuration
	API APIConfig `yaml:"api"`

	// Risk calculation settings
	Risk RiskConfig `yaml:"risk"`

	// Sync settings
	Sync SyncConfig `yaml:"sync"`

	// Budget limits
	Budget BudgetConfig `yaml:"budget"`
}

type StorageConfig struct {
	Type        string `yaml:"type"` // "postgres", "sqlite"
	PostgresDSN string `yaml:"postgres_dsn"`
	LocalPath   string `yaml:"local_path"`
}

type GitHubConfig struct {
	Token     string `yaml:"token"`
	RateLimit int    `yaml:"rate_limit"` // Requests per second
}

type CacheConfig struct {
	Directory      string        `yaml:"directory"`
	TTL            time.Duration `yaml:"ttl"`
	MaxSize        int64         `yaml:"max_size"` // In bytes
	SharedCacheURL string        `yaml:"shared_cache_url"`
}

type APIConfig struct {
	OpenAIKey    string `yaml:"openai_key"`
	OpenAIModel  string `yaml:"openai_model"`
	UseKeychain  bool   `yaml:"use_keychain"`  // Prefer keychain over config file
	CustomLLMURL string `yaml:"custom_llm_url"`
	CustomLLMKey string `yaml:"custom_llm_key"`
	EmbeddingURL string `yaml:"embedding_url"`
	EmbeddingKey string `yaml:"embedding_key"`
}

type RiskConfig struct {
	DefaultLevel      int     `yaml:"default_level"` // 1, 2, or 3
	LowThreshold      float64 `yaml:"low_threshold"`
	MediumThreshold   float64 `yaml:"medium_threshold"`
	HighThreshold     float64 `yaml:"high_threshold"`
	CriticalThreshold float64 `yaml:"critical_threshold"`
}

type SyncConfig struct {
	AutoSync        bool          `yaml:"auto_sync"`
	FreshThreshold  time.Duration `yaml:"fresh_threshold"`
	StaleThreshold  time.Duration `yaml:"stale_threshold"`
	WebhookEndpoint string        `yaml:"webhook_endpoint"`
}

type BudgetConfig struct {
	DailyLimit    float64 `yaml:"daily_limit"`
	MonthlyLimit  float64 `yaml:"monthly_limit"`
	PerCheckLimit float64 `yaml:"per_check_limit"`
	AlertAt       float64 `yaml:"alert_at"` // Percentage of limit
}

// Default returns default configuration
func Default() *Config {
	homeDir, _ := os.UserHomeDir()
	return &Config{
		Mode: "team",
		Storage: StorageConfig{
			Type:      "sqlite",
			LocalPath: filepath.Join(homeDir, ".coderisk", "local.db"),
		},
		GitHub: GitHubConfig{
			RateLimit: 10, // 10 requests per second
		},
		Cache: CacheConfig{
			Directory: filepath.Join(homeDir, ".coderisk", "cache"),
			TTL:       24 * time.Hour,
			MaxSize:   2 * 1024 * 1024 * 1024, // 2GB
		},
		API: APIConfig{
			OpenAIModel: "gpt-4o-mini",
		},
		Risk: RiskConfig{
			DefaultLevel:      1,
			LowThreshold:      0.25,
			MediumThreshold:   0.50,
			HighThreshold:     0.75,
			CriticalThreshold: 0.90,
		},
		Sync: SyncConfig{
			AutoSync:       true,
			FreshThreshold: 30 * time.Minute,
			StaleThreshold: 4 * time.Hour,
		},
		Budget: BudgetConfig{
			DailyLimit:    2.00,
			MonthlyLimit:  60.00,
			PerCheckLimit: 0.04,
			AlertAt:       0.80,
		},
	}
}

// Load loads configuration from file
func Load(path string) (*Config, error) {
	// Load .env files first (in order of precedence)
	loadEnvFiles()

	v := viper.New()
	v.SetConfigType("yaml")

	// Set defaults
	cfg := Default()
	v.SetDefault("mode", cfg.Mode)
	v.SetDefault("storage", cfg.Storage)
	v.SetDefault("github", cfg.GitHub)
	v.SetDefault("cache", cfg.Cache)
	v.SetDefault("risk", cfg.Risk)
	v.SetDefault("sync", cfg.Sync)
	v.SetDefault("budget", cfg.Budget)

	// Load from environment variables
	v.SetEnvPrefix("CODERISK")
	v.AutomaticEnv()

	// Try to find config file
	if path != "" {
		v.SetConfigFile(path)
	} else {
		// Search for config in standard locations
		v.SetConfigName("config")
		v.AddConfigPath(".coderisk")
		v.AddConfigPath(".")
		homeDir, _ := os.UserHomeDir()
		v.AddConfigPath(filepath.Join(homeDir, ".coderisk"))
	}

	// Read config file if it exists
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// Config file not found is OK, use defaults
	}

	// Unmarshal into struct
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply environment variable overrides
	applyEnvOverrides(cfg)

	return cfg, nil
}

// loadEnvFiles loads .env files in order of precedence
func loadEnvFiles() {
	// Try to load .env files in order of precedence
	envFiles := []string{
		".env.local",   // Local overrides (highest precedence)
		".env",         // Main environment file
		".env.example", // Example file as fallback
	}

	for _, file := range envFiles {
		if _, err := os.Stat(file); err == nil {
			if err := godotenv.Load(file); err == nil {
				// Successfully loaded, continue to next
				continue
			}
		}
	}

	// Also try loading from home directory
	homeDir, _ := os.UserHomeDir()
	homeEnvFile := filepath.Join(homeDir, ".coderisk", ".env")
	if _, err := os.Stat(homeEnvFile); err == nil {
		godotenv.Load(homeEnvFile)
	}
}

// applyEnvOverrides applies environment variable overrides to config
func applyEnvOverrides(cfg *Config) {
	// GitHub configuration
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		cfg.GitHub.Token = token
	}
	if rateLimit := os.Getenv("GITHUB_RATE_LIMIT"); rateLimit != "" {
		if rate, err := strconv.Atoi(rateLimit); err == nil {
			cfg.GitHub.RateLimit = rate
		}
	}

	// API configuration - UPDATED FOR KEYCHAIN SUPPORT
	// Precedence: 1. Env var (highest) 2. Keychain 3. Config file (lowest)

	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		// Environment variable has highest precedence (for CI/CD)
		cfg.API.OpenAIKey = key
	} else if cfg.API.OpenAIKey == "" {
		// Try keychain if no env var and no config file value
		// This allows config file to be used if explicitly set
		km := NewKeyringManager()
		if km.IsAvailable() {
			if keychainKey, err := km.GetAPIKey(); err == nil && keychainKey != "" {
				cfg.API.OpenAIKey = keychainKey
			}
		}
	}

	if model := os.Getenv("OPENAI_MODEL"); model != "" {
		cfg.API.OpenAIModel = model
	}
	if url := os.Getenv("CUSTOM_LLM_URL"); url != "" {
		cfg.API.CustomLLMURL = url
	}
	if key := os.Getenv("CUSTOM_LLM_KEY"); key != "" {
		cfg.API.CustomLLMKey = key
	}
	if url := os.Getenv("CUSTOM_EMBEDDING_URL"); url != "" {
		cfg.API.EmbeddingURL = url
	}
	if key := os.Getenv("CUSTOM_EMBEDDING_KEY"); key != "" {
		cfg.API.EmbeddingKey = key
	}

	// Storage configuration
	if storageType := os.Getenv("STORAGE_TYPE"); storageType != "" {
		cfg.Storage.Type = storageType
	}
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		cfg.Storage.PostgresDSN = dsn
	}
	if path := os.Getenv("LOCAL_DB_PATH"); path != "" {
		cfg.Storage.LocalPath = expandPath(path)
	}

	// Cache configuration
	if dir := os.Getenv("CACHE_DIRECTORY"); dir != "" {
		cfg.Cache.Directory = expandPath(dir)
	}
	if url := os.Getenv("SHARED_CACHE_URL"); url != "" {
		cfg.Cache.SharedCacheURL = url
	}
	if size := os.Getenv("CACHE_MAX_SIZE"); size != "" {
		if sizeInt, err := strconv.ParseInt(size, 10, 64); err == nil {
			cfg.Cache.MaxSize = sizeInt
		}
	}

	// Budget configuration
	if daily := os.Getenv("BUDGET_DAILY_LIMIT"); daily != "" {
		if amount, err := strconv.ParseFloat(daily, 64); err == nil {
			cfg.Budget.DailyLimit = amount
		}
	}
	if monthly := os.Getenv("BUDGET_MONTHLY_LIMIT"); monthly != "" {
		if amount, err := strconv.ParseFloat(monthly, 64); err == nil {
			cfg.Budget.MonthlyLimit = amount
		}
	}
	if perCheck := os.Getenv("BUDGET_PER_CHECK_LIMIT"); perCheck != "" {
		if amount, err := strconv.ParseFloat(perCheck, 64); err == nil {
			cfg.Budget.PerCheckLimit = amount
		}
	}

	// Sync configuration
	if autoSync := os.Getenv("SYNC_AUTO_SYNC"); autoSync != "" {
		cfg.Sync.AutoSync = autoSync == "true"
	}
	if fresh := os.Getenv("SYNC_FRESH_THRESHOLD_MINUTES"); fresh != "" {
		if minutes, err := strconv.Atoi(fresh); err == nil {
			cfg.Sync.FreshThreshold = time.Duration(minutes) * time.Minute
		}
	}
	if stale := os.Getenv("SYNC_STALE_THRESHOLD_HOURS"); stale != "" {
		if hours, err := strconv.Atoi(stale); err == nil {
			cfg.Sync.StaleThreshold = time.Duration(hours) * time.Hour
		}
	}

	// Risk configuration
	if level := os.Getenv("RISK_DEFAULT_LEVEL"); level != "" {
		if levelInt, err := strconv.Atoi(level); err == nil {
			cfg.Risk.DefaultLevel = levelInt
		}
	}

	// Mode configuration
	if mode := os.Getenv("CODERISK_MODE"); mode != "" {
		cfg.Mode = mode
	}
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if path == "" {
		return path
	}
	if path[0] == '~' {
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, path[1:])
	}
	return path
}

// Save saves configuration to file
func (c *Config) Save(path string) error {
	v := viper.New()
	v.SetConfigType("yaml")

	// Convert struct to map for Viper
	v.Set("mode", c.Mode)
	v.Set("storage", c.Storage)
	v.Set("github", c.GitHub)
	v.Set("cache", c.Cache)
	v.Set("api", c.API)
	v.Set("risk", c.Risk)
	v.Set("sync", c.Sync)
	v.Set("budget", c.Budget)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write config file
	if err := v.WriteConfigAs(path); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
