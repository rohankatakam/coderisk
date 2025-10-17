package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/rohankatakam/coderisk/internal/errors"
)

// ValidationContext specifies what configuration is required
type ValidationContext string

const (
	// ValidationContextInit - crisk init requires Neo4j and PostgreSQL
	ValidationContextInit ValidationContext = "init"
	// ValidationContextCheck - crisk check requires API key for high-risk files
	ValidationContextCheck ValidationContext = "check"
	// ValidationContextIncident - incident commands require PostgreSQL and Neo4j
	ValidationContextIncident ValidationContext = "incident"
	// ValidationContextParse - parse command requires Neo4j
	ValidationContextParse ValidationContext = "parse"
	// ValidationContextAll - validate all configuration
	ValidationContextAll ValidationContext = "all"
)

// ValidationResult holds validation results
type ValidationResult struct {
	Valid   bool
	Errors  []string
	Warnings []string
}

// AddError adds an error to the validation result
func (vr *ValidationResult) AddError(format string, args ...interface{}) {
	vr.Valid = false
	vr.Errors = append(vr.Errors, fmt.Sprintf(format, args...))
}

// AddWarning adds a warning to the validation result
func (vr *ValidationResult) AddWarning(format string, args ...interface{}) {
	vr.Warnings = append(vr.Warnings, fmt.Sprintf(format, args...))
}

// HasErrors returns true if there are any errors
func (vr *ValidationResult) HasErrors() bool {
	return !vr.Valid || len(vr.Errors) > 0
}

// Error returns a formatted error message
func (vr *ValidationResult) Error() string {
	if !vr.HasErrors() {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Configuration validation failed:\n")
	for _, err := range vr.Errors {
		sb.WriteString(fmt.Sprintf("  ❌ %s\n", err))
	}

	if len(vr.Warnings) > 0 {
		sb.WriteString("\nWarnings:\n")
		for _, warn := range vr.Warnings {
			sb.WriteString(fmt.Sprintf("  ⚠️  %s\n", warn))
		}
	}

	return sb.String()
}

// Validate validates configuration for the given context with auto-detected mode
func (c *Config) Validate(ctx ValidationContext) *ValidationResult {
	mode := DetectMode()
	return c.ValidateWithMode(ctx, mode)
}

// ValidateWithMode validates configuration for the given context and deployment mode
func (c *Config) ValidateWithMode(ctx ValidationContext, mode DeploymentMode) *ValidationResult {
	result := &ValidationResult{Valid: true}

	switch ctx {
	case ValidationContextInit:
		c.validateNeo4j(result, true, mode)
		c.validatePostgres(result, true, mode)
		c.validateCache(result)
		c.validateGitHub(result, false) // Optional for init
	case ValidationContextCheck:
		c.validateNeo4j(result, true, mode)
		c.validateAPI(result, false) // Required only for high-risk files
		c.validateCache(result)
	case ValidationContextIncident:
		c.validatePostgres(result, true, mode)
		c.validateNeo4j(result, true, mode)
	case ValidationContextParse:
		c.validateNeo4j(result, true, mode)
	case ValidationContextAll:
		c.validateNeo4j(result, true, mode)
		c.validatePostgres(result, false, mode) // Optional in some modes
		c.validateAPI(result, false)
		c.validateCache(result)
		c.validateGitHub(result, false)
		c.validateRisk(result)
		c.validateBudget(result)
	}

	return result
}

// ValidateOrFatal validates configuration and exits if invalid (auto-detects mode)
func (c *Config) ValidateOrFatal(ctx ValidationContext) {
	mode := DetectMode()
	c.ValidateOrFatalWithMode(ctx, mode)
}

// ValidateOrFatalWithMode validates configuration with explicit mode and exits if invalid
func (c *Config) ValidateOrFatalWithMode(ctx ValidationContext, mode DeploymentMode) {
	result := c.ValidateWithMode(ctx, mode)
	if result.HasErrors() {
		// Print the error
		fmt.Println(result.Error())
		fmt.Printf("\nDeployment mode: %s (%s)\n", mode, mode.Description())
		// Exit with error code
		panic(errors.ConfigError(result.Error()))
	}

	// Print warnings if any
	if len(result.Warnings) > 0 {
		fmt.Println("Configuration warnings:")
		for _, warn := range result.Warnings {
			fmt.Printf("  ⚠️  %s\n", warn)
		}
		fmt.Printf("\nDeployment mode: %s\n", mode)
	}
}

func (c *Config) validateNeo4j(result *ValidationResult, required bool, mode DeploymentMode) {
	if c.Neo4j.URI == "" {
		if required {
			result.AddError("NEO4J_URI is required but not set")
		} else {
			result.AddWarning("NEO4J_URI is not set")
		}
	} else {
		// Validate URI format
		if _, err := url.Parse(c.Neo4j.URI); err != nil {
			result.AddError("NEO4J_URI is invalid: %v", err)
		}

		// Check for localhost URI - only matters in packaged/CI mode
		if c.Neo4j.URI == "bolt://localhost:7687" || strings.Contains(c.Neo4j.URI, "localhost") {
			if mode.RequiresSecureCredentials() {
				result.AddError("Neo4j URI uses localhost. In %s mode (%s), you must provide a remote database URI.", mode, mode.Description())
			}
			// In development mode, localhost is expected and acceptable
		}
	}

	if c.Neo4j.User == "" {
		if required {
			result.AddError("NEO4J_USER is required but not set")
		} else {
			result.AddWarning("NEO4J_USER is not set")
		}
	}

	if c.Neo4j.Password == "" {
		if required {
			result.AddError("NEO4J_PASSWORD is required but not set. Set it via environment variable or .env file.")
		} else {
			result.AddWarning("NEO4J_PASSWORD is not set")
		}
	} else {
		// Check for insecure default passwords - MODE-AWARE
		insecurePasswords := []string{
			"coderisk123",
			"CHANGE_THIS_PASSWORD_IN_PRODUCTION_123",
			"password",
			"neo4j",
		}

		// In packaged/CI mode, reject any insecure defaults
		if mode.RequiresSecureCredentials() {
			for _, insecure := range insecurePasswords {
				if c.Neo4j.Password == insecure {
					result.AddError("NEO4J_PASSWORD is set to an insecure default (%s). This is not allowed in %s mode. Set a secure password via %s.", insecure, mode, mode.ConfigSource())
				}
			}
		} else if mode.AllowsDevelopmentDefaults() {
			// In development mode, .env defaults are acceptable for local Docker
			// Only warn if using extremely common passwords
			veryInsecure := []string{"password", "neo4j"}
			for _, insecure := range veryInsecure {
				if c.Neo4j.Password == insecure {
					result.AddWarning("NEO4J_PASSWORD is set to a very common password (%s). Consider changing it even for local development.", insecure)
				}
			}
		}
	}

	if c.Neo4j.Database == "" {
		if required {
			result.AddError("NEO4J_DATABASE is required but not set")
		} else {
			result.AddWarning("NEO4J_DATABASE is not set, will use 'neo4j' as default")
		}
	}
}

func (c *Config) validatePostgres(result *ValidationResult, required bool, mode DeploymentMode) {
	if c.Storage.Type == "postgres" || required {
		if c.Storage.PostgresDSN == "" {
			if required {
				result.AddError("POSTGRES_DSN is required but not set")
			} else {
				result.AddWarning("POSTGRES_DSN is not set")
			}
		} else {
			// Check for insecure defaults - MODE-AWARE
			if strings.Contains(c.Storage.PostgresDSN, "CHANGE_THIS_PASSWORD_IN_PRODUCTION_123") {
				if mode.RequiresSecureCredentials() {
					result.AddError("PostgreSQL DSN contains insecure default password. This is not allowed in %s mode. Set secure credentials via %s.", mode, mode.ConfigSource())
				}
				// In development mode, .env defaults are acceptable for local Docker
			}

			// Check for localhost
			if strings.Contains(c.Storage.PostgresDSN, "@localhost:") || strings.Contains(c.Storage.PostgresDSN, "@localhost/") {
				if mode.RequiresSecureCredentials() {
					result.AddError("PostgreSQL DSN uses localhost. In %s mode (%s), you must provide a remote database DSN.", mode, mode.Description())
				}
				// In development mode, localhost is expected
			}

			// Check for disabled SSL
			if strings.Contains(c.Storage.PostgresDSN, "sslmode=disable") {
				if mode.RequiresSecureCredentials() {
					result.AddError("PostgreSQL DSN has sslmode=disable. This is not allowed in %s mode. Use sslmode=require or sslmode=verify-full.", mode)
				} else if mode.AllowsDevelopmentDefaults() {
					result.AddWarning("PostgreSQL DSN has sslmode=disable. Consider enabling SSL even for local development.")
				}
			}

			// Validate DSN format
			if !strings.HasPrefix(c.Storage.PostgresDSN, "postgres://") && !strings.HasPrefix(c.Storage.PostgresDSN, "postgresql://") {
				result.AddError("POSTGRES_DSN must start with postgres:// or postgresql://")
			}
		}
	} else {
		// If no DSN, validate individual connection parameters
		if required {
			if c.Storage.PostgresHost == "" {
				result.AddError("POSTGRES_HOST is required but not set")
			}
			if c.Storage.PostgresPort == 0 {
				result.AddError("POSTGRES_PORT_EXTERNAL is required but not set")
			}
			if c.Storage.PostgresDB == "" {
				result.AddError("POSTGRES_DB is required but not set")
			}
			if c.Storage.PostgresUser == "" {
				result.AddError("POSTGRES_USER is required but not set")
			}
			if c.Storage.PostgresPassword == "" {
				result.AddError("POSTGRES_PASSWORD is required but not set in .env or environment variable")
			}
		}
	}
}

func (c *Config) validateAPI(result *ValidationResult, required bool) {
	if c.API.OpenAIKey == "" {
		if required {
			result.AddError("OPENAI_API_KEY is required but not set. Set it via environment variable or keychain.")
		} else {
			result.AddWarning("OPENAI_API_KEY is not set. High-risk file investigations will be skipped.")
		}
	}

	if c.API.OpenAIModel == "" {
		result.AddWarning("OPENAI_MODEL is not set, will use default model")
	}

	// Validate custom LLM configuration
	if c.API.CustomLLMURL != "" {
		if _, err := url.Parse(c.API.CustomLLMURL); err != nil {
			result.AddError("CUSTOM_LLM_URL is invalid: %v", err)
		}
	}

	if c.API.EmbeddingURL != "" {
		if _, err := url.Parse(c.API.EmbeddingURL); err != nil {
			result.AddError("CUSTOM_EMBEDDING_URL is invalid: %v", err)
		}
	}
}

func (c *Config) validateCache(result *ValidationResult) {
	if c.Cache.Directory == "" {
		result.AddWarning("CACHE_DIRECTORY is not set, will use default")
	}

	if c.Cache.MaxSize <= 0 {
		result.AddWarning("CACHE_MAX_SIZE is invalid or not set, will use default (2GB)")
	}
}

func (c *Config) validateGitHub(result *ValidationResult, required bool) {
	if c.GitHub.Token == "" {
		if required {
			result.AddError("GITHUB_TOKEN is required but not set")
		} else {
			result.AddWarning("GITHUB_TOKEN is not set. GitHub integration features will be limited.")
		}
	}

	if c.GitHub.RateLimit <= 0 {
		result.AddWarning("GITHUB_RATE_LIMIT is invalid, will use default (10 req/s)")
	}
}

func (c *Config) validateRisk(result *ValidationResult) {
	if c.Risk.DefaultLevel < 1 || c.Risk.DefaultLevel > 3 {
		result.AddWarning("RISK_DEFAULT_LEVEL must be 1, 2, or 3. Got %d, will use 1", c.Risk.DefaultLevel)
	}

	// Validate thresholds are in ascending order
	thresholds := []float64{
		c.Risk.LowThreshold,
		c.Risk.MediumThreshold,
		c.Risk.HighThreshold,
		c.Risk.CriticalThreshold,
	}

	for i := 1; i < len(thresholds); i++ {
		if thresholds[i] <= thresholds[i-1] {
			result.AddError("Risk thresholds must be in ascending order")
			break
		}
	}

	for i, threshold := range thresholds {
		if threshold < 0 || threshold > 1 {
			result.AddError("Risk threshold %d is out of range [0,1]: %.2f", i, threshold)
		}
	}
}

func (c *Config) validateBudget(result *ValidationResult) {
	if c.Budget.DailyLimit < 0 {
		result.AddWarning("BUDGET_DAILY_LIMIT is negative, will use default")
	}

	if c.Budget.MonthlyLimit < 0 {
		result.AddWarning("BUDGET_MONTHLY_LIMIT is negative, will use default")
	}

	if c.Budget.PerCheckLimit < 0 {
		result.AddWarning("BUDGET_PER_CHECK_LIMIT is negative, will use default")
	}

	if c.Budget.AlertAt < 0 || c.Budget.AlertAt > 1 {
		result.AddWarning("BUDGET_ALERT_AT must be between 0 and 1, got %.2f", c.Budget.AlertAt)
	}
}

// RequireNeo4j checks if Neo4j configuration is valid and returns error if not
func (c *Config) RequireNeo4j() error {
	result := &ValidationResult{Valid: true}
	mode := DetectMode()
	c.validateNeo4j(result, true, mode)

	if result.HasErrors() {
		return errors.ConfigError(result.Error())
	}

	return nil
}

// RequirePostgres checks if Postgres configuration is valid and returns error if not
func (c *Config) RequirePostgres() error {
	result := &ValidationResult{Valid: true}
	mode := DetectMode()
	c.validatePostgres(result, true, mode)

	if result.HasErrors() {
		return errors.ConfigError(result.Error())
	}

	return nil
}

// RequireAPI checks if API configuration is valid and returns error if not
func (c *Config) RequireAPI() error {
	result := &ValidationResult{Valid: true}
	c.validateAPI(result, true)

	if result.HasErrors() {
		return errors.ConfigError(result.Error())
	}

	return nil
}

// NOTE: GetString and GetInt functions removed - they are defined in env.go
// Use those functions instead to avoid duplication
