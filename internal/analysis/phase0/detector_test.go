package phase0

import (
	"strings"
	"testing"
	"time"
)

func TestRunPhase0(t *testing.T) {
	tests := []struct {
		name              string
		filePath          string
		content           string
		expectSkipAll     bool // Skip Phase 1 AND Phase 2 (documentation)
		expectForceEscalate bool // Skip Phase 1, go to Phase 2
		expectRunPhase1   bool // Normal flow - proceed to Phase 1
		expectRisk        string
		expectReasonContains string // Partial match for reason
		description       string
	}{
		// DOCUMENTATION SKIP SCENARIOS
		{
			name:             "README - documentation skip",
			filePath:         "README.md",
			content:          "# My Project\n\nThis is documentation.",
			expectSkipAll:    true,
			expectForceEscalate: false,
			expectRunPhase1:  false,
			expectRisk:       "LOW",
			expectReasonContains: "Documentation-only change",
			description:      "README should skip all analysis",
		},
		{
			name:             "API documentation",
			filePath:         "docs/api.md",
			content:          "# API Documentation\n\nEndpoints...",
			expectSkipAll:    true,
			expectForceEscalate: false,
			expectRunPhase1:  false,
			expectRisk:       "LOW",
			expectReasonContains: "Documentation-only",
			description:      "Documentation files should skip analysis",
		},

		// SECURITY FORCE ESCALATION SCENARIOS
		{
			name:             "Security - authentication file",
			filePath:         "internal/auth/login.go",
			content:          "func Login(username, password string) { authenticate() }",
			expectSkipAll:    false,
			expectForceEscalate: true,
			expectRunPhase1:  false,
			expectRisk:       "CRITICAL",
			expectReasonContains: "Security-sensitive change",
			description:      "Security file should force escalate to Phase 2",
		},
		{
			name:             "Security - JWT token handling",
			filePath:         "internal/middleware/auth.go",
			content:          "func ValidateToken(token string) { jwt.Verify(token) }",
			expectSkipAll:    false,
			expectForceEscalate: true,
			expectRunPhase1:  false,
			expectRisk:       "CRITICAL",
			expectReasonContains: "Security-sensitive",
			description:      "JWT handling should force escalate",
		},
		{
			name:             "Security - encryption code",
			filePath:         "internal/crypto/encrypt.go",
			content:          "import \"crypto/aes\"\nfunc Encrypt(data []byte, key []byte) {}",
			expectSkipAll:    false,
			expectForceEscalate: true,
			expectRunPhase1:  false,
			expectRisk:       "CRITICAL",
			expectReasonContains: "Security-sensitive",
			description:      "Encryption code should force escalate",
		},

		// PRODUCTION CONFIG FORCE ESCALATION SCENARIOS
		{
			name:             "Production config - .env.production",
			filePath:         ".env.production",
			content:          "DATABASE_URL=postgres://prod\nAPI_KEY=secret123",
			expectSkipAll:    false,
			expectForceEscalate: true,
			expectRunPhase1:  false,
			expectRisk:       "HIGH", // Security keyword detected, but not security path
			expectReasonContains: "Security-sensitive",
			description:      "Production env file with secrets should force escalate (security takes precedence)",
		},
		{
			name:             "Production config - prod.yaml",
			filePath:         "config/prod.yaml",
			content:          "database:\n  host: prod-db.example.com\n  port: 5432",
			expectSkipAll:    false,
			expectForceEscalate: true,
			expectRunPhase1:  false,
			expectRisk:       "CRITICAL",
			expectReasonContains: "Production configuration",
			description:      "Production YAML should force escalate",
		},
		{
			name:             "Staging config - staging.json",
			filePath:         "config/staging.json",
			content:          "{\"api_url\": \"https://staging.api.example.com\"}",
			expectSkipAll:    false,
			expectForceEscalate: true,
			expectRunPhase1:  false,
			expectRisk:       "HIGH",
			expectReasonContains: "Staging configuration",
			description:      "Staging config should force escalate to HIGH",
		},
		{
			name:             "Unknown environment config - config.yaml",
			filePath:         "config.yaml",
			content:          "database:\n  host: db.example.com",
			expectSkipAll:    false,
			expectForceEscalate: true,
			expectRunPhase1:  false,
			expectRisk:       "HIGH",
			expectReasonContains: "Unknown environment configuration",
			description:      "Unknown config should force escalate (safety-first)",
		},

		// NORMAL FLOW - PROCEED TO PHASE 1
		{
			name:             "Regular code - API handler",
			filePath:         "internal/api/handlers.go",
			content:          "func GetUsers(w http.ResponseWriter, r *http.Request) { users := fetchUsers(); json.NewEncoder(w).Encode(users) }",
			expectSkipAll:    false,
			expectForceEscalate: false,
			expectRunPhase1:  true,
			expectRisk:       "HIGH", // Interface type
			expectReasonContains: "Modification type",
			description:      "Regular API code should proceed to Phase 1",
		},
		{
			name:             "Regular code - business logic",
			filePath:         "internal/services/payment.go",
			content:          "func ProcessPayment(amount int) error { if amount <= 0 { return errors.New(\"invalid amount\") }; return charge(amount) }",
			expectSkipAll:    false,
			expectForceEscalate: false,
			expectRunPhase1:  true,
			expectRisk:       "MODERATE", // Behavioral type
			expectReasonContains: "Modification type",
			description:      "Business logic should proceed to Phase 1",
		},
		{
			name:             "Test file - unit test",
			filePath:         "internal/services/payment_test.go",
			content:          "func TestProcessPayment(t *testing.T) { err := ProcessPayment(100); assert.NoError(t, err) }",
			expectSkipAll:    false,
			expectForceEscalate: false,
			expectRunPhase1:  true,
			expectRisk:       "LOW", // TestQuality type
			expectReasonContains: "Modification type",
			description:      "Test files should proceed to Phase 1 with LOW risk",
		},
		{
			name:             "Development config - .env.local",
			filePath:         ".env.local",
			content:          "DATABASE_URL=postgres://localhost:5432/dev",
			expectSkipAll:    false,
			expectForceEscalate: false,
			expectRunPhase1:  true,
			expectRisk:       "MODERATE", // Configuration type for dev
			expectReasonContains: "Modification type",
			description:      "Dev config should proceed to Phase 1",
		},

		// MULTI-TYPE SCENARIOS
		{
			name:             "Multi-type - security + API + behavioral",
			filePath:         "internal/api/auth/middleware.go",
			content: `package auth
import "net/http"
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Unauthorized", 401)
			return
		}
		validateToken(token)
		next.ServeHTTP(w, r)
	})
}`,
			expectSkipAll:    false,
			expectForceEscalate: true,
			expectRunPhase1:  false,
			expectRisk:       "CRITICAL",
			expectReasonContains: "Security-sensitive",
			description:      "Multi-type with security should force escalate",
		},

		// EDGE CASES
		{
			name:             "Empty file",
			filePath:         "internal/empty.go",
			content:          "",
			expectSkipAll:    false,
			expectForceEscalate: false,
			expectRunPhase1:  true,
			expectRisk:       "UNKNOWN", // No types detected
			expectReasonContains: "No specific risk indicators",
			description:      "Empty file should proceed to Phase 1 with UNKNOWN risk",
		},
		{
			name:             "Minimal content - package declaration",
			filePath:         "main.go",
			content:          "package main",
			expectSkipAll:    false,
			expectForceEscalate: false,
			expectRunPhase1:  true,
			expectRisk:       "HIGH", // Structural (package import)
			expectReasonContains: "Modification type",
			description:      "Package declaration should be structural",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RunPhase0(tt.filePath, tt.content)

			// Check skip analysis flag
			if result.ShouldSkipPhase1And2() != tt.expectSkipAll {
				t.Errorf("ShouldSkipPhase1And2() = %v, want %v\nDescription: %s",
					result.ShouldSkipPhase1And2(), tt.expectSkipAll, tt.description)
			}

			// Check force escalate flag
			if result.ShouldSkipPhase1() != tt.expectForceEscalate {
				t.Errorf("ShouldSkipPhase1() = %v, want %v\nDescription: %s",
					result.ShouldSkipPhase1(), tt.expectForceEscalate, tt.description)
			}

			// Check run Phase 1 flag
			if result.ShouldRunPhase1() != tt.expectRunPhase1 {
				t.Errorf("ShouldRunPhase1() = %v, want %v\nDescription: %s",
					result.ShouldRunPhase1(), tt.expectRunPhase1, tt.description)
			}

			// Check risk level
			if result.GetFinalRiskLevel() != tt.expectRisk {
				t.Errorf("GetFinalRiskLevel() = %v, want %v\nDescription: %s",
					result.GetFinalRiskLevel(), tt.expectRisk, tt.description)
			}

			// Check reason contains expected text
			if !strings.Contains(result.Reason, tt.expectReasonContains) {
				t.Errorf("Reason does not contain %q\nGot: %s\nDescription: %s",
					tt.expectReasonContains, result.Reason, tt.description)
			}

			// Log results for debugging
			t.Logf("Result: SkipAll=%v, ForceEscalate=%v, RunPhase1=%v, Risk=%s, Duration=%s",
				result.ShouldSkipPhase1And2(),
				result.ShouldSkipPhase1(),
				result.ShouldRunPhase1(),
				result.GetFinalRiskLevel(),
				result.Duration)
			t.Logf("Reason: %s", result.Reason)
			t.Logf("Phase Transition: %s", result.GetPhaseTransition())
		})
	}
}

func TestPhase0Result_Helpers(t *testing.T) {
	tests := []struct {
		name             string
		result           Phase0Result
		expectSkipAll    bool
		expectSkipPhase1 bool
		expectRunPhase1  bool
		expectTransition string
	}{
		{
			name: "Documentation skip - all analysis",
			result: Phase0Result{
				SkipAnalysis:  true,
				ForceEscalate: false,
				AggregatedRisk: "LOW",
			},
			expectSkipAll:    true,
			expectSkipPhase1: false,
			expectRunPhase1:  false,
			expectTransition: "Skip Phase 1/2 → Return LOW immediately",
		},
		{
			name: "Security force escalate - skip Phase 1 only",
			result: Phase0Result{
				SkipAnalysis:  false,
				ForceEscalate: true,
				AggregatedRisk: "CRITICAL",
			},
			expectSkipAll:    false,
			expectSkipPhase1: true,
			expectRunPhase1:  false,
			expectTransition: "Skip Phase 1 → Escalate to Phase 2 (CRITICAL)",
		},
		{
			name: "Regular code - proceed to Phase 1",
			result: Phase0Result{
				SkipAnalysis:  false,
				ForceEscalate: false,
				AggregatedRisk: "MODERATE",
			},
			expectSkipAll:    false,
			expectSkipPhase1: false,
			expectRunPhase1:  true,
			expectTransition: "Proceed to Phase 1 baseline assessment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.ShouldSkipPhase1And2() != tt.expectSkipAll {
				t.Errorf("ShouldSkipPhase1And2() = %v, want %v",
					tt.result.ShouldSkipPhase1And2(), tt.expectSkipAll)
			}

			if tt.result.ShouldSkipPhase1() != tt.expectSkipPhase1 {
				t.Errorf("ShouldSkipPhase1() = %v, want %v",
					tt.result.ShouldSkipPhase1(), tt.expectSkipPhase1)
			}

			if tt.result.ShouldRunPhase1() != tt.expectRunPhase1 {
				t.Errorf("ShouldRunPhase1() = %v, want %v",
					tt.result.ShouldRunPhase1(), tt.expectRunPhase1)
			}

			transition := tt.result.GetPhaseTransition()
			if transition != tt.expectTransition {
				t.Errorf("GetPhaseTransition() = %q, want %q",
					transition, tt.expectTransition)
			}
		})
	}
}

func TestPhase0Result_Summary(t *testing.T) {
	result := Phase0Result{
		FilePath:      "internal/auth/login.go",
		SkipAnalysis:  false,
		ForceEscalate: true,
		ModificationType: TypeSecurity,
		ModificationTypes: []ModificationType{TypeSecurity, TypeBehavioral},
		AggregatedRisk: "CRITICAL",
		Reason:         "Security-sensitive change detected",
		Duration:       1234 * time.Microsecond,
	}

	summary := result.Summary()

	// Check that summary contains key information
	expectedStrings := []string{
		"Phase 0 Analysis",
		"internal/auth/login.go",
		"CRITICAL",
		"Skip Analysis: false",
		"Force Escalate: true",
		"Security",
		"Behavioral",
		"Security-sensitive change detected",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(summary, expected) {
			t.Errorf("Summary missing expected string %q\nGot: %s", expected, summary)
		}
	}

	t.Logf("Summary:\n%s", summary)
}

func TestPhase0Result_IsHighConfidence(t *testing.T) {
	tests := []struct {
		name           string
		result         Phase0Result
		expectHighConf bool
	}{
		{
			name: "High confidence - documentation skip",
			result: Phase0Result{
				SkipAnalysis:  true,
				ForceEscalate: false,
			},
			expectHighConf: true,
		},
		{
			name: "High confidence - security force escalate",
			result: Phase0Result{
				SkipAnalysis:  false,
				ForceEscalate: true,
			},
			expectHighConf: true,
		},
		{
			name: "Low confidence - regular code",
			result: Phase0Result{
				SkipAnalysis:  false,
				ForceEscalate: false,
			},
			expectHighConf: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.IsHighConfidence() != tt.expectHighConf {
				t.Errorf("IsHighConfidence() = %v, want %v",
					tt.result.IsHighConfidence(), tt.expectHighConf)
			}
		})
	}
}

func TestPhase0Performance(t *testing.T) {
	// Test that Phase 0 completes within performance target (<50ms)
	testCases := []struct {
		filePath string
		content  string
	}{
		{"README.md", "# Documentation\n\nThis is a long README with many lines...\n" + strings.Repeat("More content\n", 100)},
		{"internal/auth/login.go", "package auth\n\nimport \"crypto/jwt\"\n\nfunc Login() {\n" + strings.Repeat("\tauthenticate()\n", 100) + "}"},
		{".env.production", strings.Repeat("KEY=value\n", 100)},
		{"internal/api/handlers.go", "package api\n\nimport \"net/http\"\n\n" + strings.Repeat("func Handler() {}\n", 100)},
	}

	maxDuration := 50 * time.Millisecond

	for _, tc := range testCases {
		t.Run(tc.filePath, func(t *testing.T) {
			result := RunPhase0(tc.filePath, tc.content)

			if result.Duration > maxDuration {
				t.Errorf("Phase 0 took too long: %s (max: %s)", result.Duration, maxDuration)
			}

			t.Logf("Duration: %s (target: <%s)", result.Duration, maxDuration)
		})
	}
}

func TestPhase0Integration_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		content  string
		expectTransition string
		description string
	}{
		{
			name:     "Real scenario - OAuth implementation",
			filePath: "internal/auth/oauth/provider.go",
			content: `package oauth

import (
	"crypto/rand"
	"encoding/base64"
	"github.com/dgrijalva/jwt-go"
)

func GenerateToken(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	secretKey := os.Getenv("JWT_SECRET")
	return token.SignedString([]byte(secretKey))
}`,
			expectTransition: "Skip Phase 1 → Escalate to Phase 2 (CRITICAL)",
			description:      "OAuth implementation should force escalate due to security",
		},
		{
			name:     "Real scenario - Database migration",
			filePath: "migrations/20250101_add_users_table.sql",
			content: `CREATE TABLE users (
	id SERIAL PRIMARY KEY,
	username VARCHAR(255) NOT NULL,
	password_hash VARCHAR(255) NOT NULL,
	created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users(username);`,
			expectTransition: "Skip Phase 1 → Escalate to Phase 2 (CRITICAL)",
			description:      "Database migration with password field should force escalate (security-sensitive)",
		},
		{
			name:     "Real scenario - Dockerfile production",
			filePath: "Dockerfile.production",
			content: `FROM node:18-alpine

WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production

COPY . .
EXPOSE 3000
CMD ["node", "server.js"]`,
			expectTransition: "Skip Phase 1 → Escalate to Phase 2 (CRITICAL)",
			description:      "Production Dockerfile should force escalate",
		},
		{
			name:     "Real scenario - Contributing guide",
			filePath: "CONTRIBUTING.md",
			content: `# Contributing to Our Project

## Getting Started

1. Fork the repository
2. Create a new branch
3. Make your changes
4. Submit a pull request

## Code Style

Please follow our coding standards...`,
			expectTransition: "Skip Phase 1/2 → Return LOW immediately",
			description:      "Contributing guide should skip all analysis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RunPhase0(tt.filePath, tt.content)

			transition := result.GetPhaseTransition()
			if transition != tt.expectTransition {
				t.Errorf("GetPhaseTransition() = %q, want %q\nDescription: %s",
					transition, tt.expectTransition, tt.description)
			}

			t.Logf("Scenario: %s", tt.description)
			t.Logf("Risk Level: %s", result.GetFinalRiskLevel())
			t.Logf("Reason: %s", result.Reason)
			t.Logf("Duration: %s", result.Duration)
			t.Logf("Phase Transition: %s", transition)
		})
	}
}

// Benchmark Phase 0 orchestration
func BenchmarkRunPhase0(b *testing.B) {
	testCases := []struct {
		name     string
		filePath string
		content  string
	}{
		{"Documentation", "README.md", "# Documentation"},
		{"Security", "internal/auth/login.go", "func Login() { authenticate() }"},
		{"Production Config", ".env.production", "API_KEY=secret"},
		{"Regular Code", "internal/api/handlers.go", "func GetUsers() {}"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			_ = RunPhase0(tc.filePath, tc.content)
		}
	}
}
