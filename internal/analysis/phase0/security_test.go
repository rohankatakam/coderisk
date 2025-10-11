package phase0

import (
	"strings"
	"testing"
)

func TestDetectSecurityKeywords(t *testing.T) {
	tests := []struct {
		name                    string
		filePath                string
		content                 string
		expectSecuritySensitive bool
		expectKeywords          int // minimum number of keywords expected
		expectPathPatterns      int // minimum number of path patterns expected
		expectForceEscalate     bool
		expectRiskLevel         string
		description             string
	}{
		// TRUE POSITIVES - Security-sensitive files
		{
			name:                    "authentication file with login function",
			filePath:                "internal/auth/handlers.go",
			content:                 "func Login(username, password string) error { return authenticate(username, password) }",
			expectSecuritySensitive: true,
			expectKeywords:          3, // auth, login, password, authenticate
			expectPathPatterns:      1, // auth
			expectForceEscalate:     true,
			expectRiskLevel:         "CRITICAL",
			description:             "Auth file with multiple security keywords should be CRITICAL",
		},
		{
			name:                    "session management file",
			filePath:                "pkg/session/manager.go",
			content:                 "type SessionManager struct { store *SessionStore; jwt string }",
			expectSecuritySensitive: true,
			expectKeywords:          2, // session, jwt
			expectPathPatterns:      0,
			expectForceEscalate:     true,
			expectRiskLevel:         "HIGH",
			description:             "Session management with JWT should be HIGH",
		},
		{
			name:                    "permission checker",
			filePath:                "internal/permissions/checker.go",
			content:                 "func CheckPermission(user User, resource string) bool { return user.Role.HasAccess(resource) }",
			expectSecuritySensitive: true,
			expectKeywords:          3, // permission, role, access
			expectPathPatterns:      1, // permission
			expectForceEscalate:     true,
			expectRiskLevel:         "CRITICAL",
			description:             "Permission checker should be CRITICAL",
		},
		{
			name:                    "crypto operations",
			filePath:                "internal/utils/encryption.go",
			content:                 "func EncryptData(data []byte, key []byte) ([]byte, error) { cipher := aes.NewCipher(key); return cipher.Encrypt(data), nil }",
			expectSecuritySensitive: true,
			expectKeywords:          3, // encrypt, key, cipher
			expectPathPatterns:      0,
			expectForceEscalate:     true,
			expectRiskLevel:         "CRITICAL",
			description:             "Encryption operations should be CRITICAL",
		},
		{
			name:                    "password hashing",
			filePath:                "internal/user/password.go",
			content:                 "func HashPassword(password string, salt []byte) string { return bcrypt.GenerateFromPassword([]byte(password), salt) }",
			expectSecuritySensitive: true,
			expectKeywords:          3, // password, salt, bcrypt
			expectPathPatterns:      0,
			expectForceEscalate:     true,
			expectRiskLevel:         "CRITICAL",
			description:             "Password hashing should be CRITICAL",
		},
		{
			name:                    "admin panel route",
			filePath:                "internal/routes/admin.go",
			content:                 "router.POST(\"/admin/users\", requireAdmin, createUser)",
			expectSecuritySensitive: true,
			expectKeywords:          1, // admin
			expectPathPatterns:      0,
			expectForceEscalate:     false, // Only 1 keyword
			expectRiskLevel:         "HIGH",
			description:             "Admin routes should be HIGH",
		},
		{
			name:                    "oauth callback",
			filePath:                "internal/auth/oauth_callback.go",
			content:                 "func OAuthCallback(code string) (*Token, error) { return exchangeCodeForToken(code) }",
			expectSecuritySensitive: true,
			expectKeywords:          3, // auth, oauth, token
			expectPathPatterns:      1, // auth
			expectForceEscalate:     true,
			expectRiskLevel:         "CRITICAL",
			description:             "OAuth callback should be CRITICAL",
		},
		{
			name:                    "security middleware",
			filePath:                "internal/middleware/security.go",
			content:                 "func SecurityMiddleware() gin.HandlerFunc { return func(c *gin.Context) { validateToken(c); checkPermissions(c) } }",
			expectSecuritySensitive: true,
			expectKeywords:          2, // token (in validateToken), validate (in validateToken) - "permission" not at word boundary in "checkPermissions"
			expectPathPatterns:      1, // security
			expectForceEscalate:     true,
			expectRiskLevel:         "CRITICAL",
			description:             "Security middleware should be CRITICAL",
		},

		// EDGE CASES - Potential false positives (should still detect but with lower confidence)
		{
			name:                    "test file for authentication",
			filePath:                "internal/auth/handlers_test.go",
			content:                 "func TestLogin(t *testing.T) { err := Login(\"user\", \"pass\"); assert.NoError(t, err) }",
			expectSecuritySensitive: true,
			expectKeywords:          2, // auth (in path), login, password
			expectPathPatterns:      1, // auth
			expectForceEscalate:     true,
			expectRiskLevel:         "CRITICAL", // Even test files for auth are critical
			description:             "Auth test files should still be flagged as CRITICAL",
		},
		{
			name:                    "documentation mentioning authentication",
			filePath:                "docs/authentication.md",
			content:                 "# Authentication Guide\n\nThis guide explains how to use the authentication system.",
			expectSecuritySensitive: true,
			expectKeywords:          0, // Path pattern match only, no keywords in plain text content
			expectPathPatterns:      1, // auth (in authentication.md)
			expectForceEscalate:     true,
			expectRiskLevel:         "HIGH", // Path-only match = HIGH (not CRITICAL which requires path + keywords)
			description:             "Auth documentation should be flagged via path pattern",
		},
		{
			name:                    "code mentioning 'author' (contains 'auth')",
			filePath:                "internal/blog/post.go",
			content:                 "type Post struct { Title string; Author string; Content string }",
			expectSecuritySensitive: false, // Word boundary matching prevents "Author" from matching "auth"
			expectKeywords:          0,     // No matches (word boundary prevents false positive)
			expectPathPatterns:      0,
			expectForceEscalate:     false,
			expectRiskLevel:         "",
			description:             "Word boundary matching correctly excludes 'Author' from matching 'auth'",
		},

		// TRUE NEGATIVES - Non-security files
		{
			name:                    "regular business logic",
			filePath:                "internal/services/order_processor.go",
			content:                 "func ProcessOrder(order Order) error { return db.Save(order) }",
			expectSecuritySensitive: false,
			expectKeywords:          0,
			expectPathPatterns:      0,
			expectForceEscalate:     false,
			expectRiskLevel:         "",
			description:             "Regular business logic should not be flagged",
		},
		{
			name:                    "utility functions",
			filePath:                "internal/utils/strings.go",
			content:                 "func ToUpper(s string) string { return strings.ToUpper(s) }",
			expectSecuritySensitive: false,
			expectKeywords:          0,
			expectPathPatterns:      0,
			expectForceEscalate:     false,
			expectRiskLevel:         "",
			description:             "Utility functions should not be flagged",
		},
		{
			name:                    "database models",
			filePath:                "internal/models/product.go",
			content:                 "type Product struct { ID int; Name string; Price float64 }",
			expectSecuritySensitive: false,
			expectKeywords:          0,
			expectPathPatterns:      0,
			expectForceEscalate:     false,
			expectRiskLevel:         "",
			description:             "Database models should not be flagged",
		},
		{
			name:                    "API handlers (non-auth)",
			filePath:                "internal/api/products.go",
			content:                 "func ListProducts(c *gin.Context) { products := db.GetAllProducts(); c.JSON(200, products) }",
			expectSecuritySensitive: false,
			expectKeywords:          0,
			expectPathPatterns:      0,
			expectForceEscalate:     false,
			expectRiskLevel:         "",
			description:             "Non-auth API handlers should not be flagged",
		},
		{
			name:                    "configuration file (non-security)",
			filePath:                "config/database.yaml",
			content:                 "database:\n  host: localhost\n  port: 5432\n  name: myapp",
			expectSecuritySensitive: false,
			expectKeywords:          0,
			expectPathPatterns:      0,
			expectForceEscalate:     false,
			expectRiskLevel:         "",
			description:             "Non-security config should not be flagged",
		},

		// MIXED SCENARIOS
		{
			name:                    "file with one security keyword",
			filePath:                "internal/services/user_service.go",
			content:                 "func GetUserByToken(token string) (*User, error) { return db.FindUserByToken(token) }",
			expectSecuritySensitive: true,
			expectKeywords:          1, // token
			expectPathPatterns:      0,
			expectForceEscalate:     false, // Only 1 keyword
			expectRiskLevel:         "HIGH",
			description:             "Single security keyword should be HIGH (not CRITICAL)",
		},
		{
			name:                    "empty file in auth directory",
			filePath:                "internal/auth/constants.go",
			content:                 "",
			expectSecuritySensitive: true,
			expectKeywords:          1, // auth from path
			expectPathPatterns:      1, // auth in path
			expectForceEscalate:     true,
			expectRiskLevel:         "CRITICAL", // Path pattern + keyword = CRITICAL
			description:             "Auth directory should be flagged even if content is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectSecurityKeywords(tt.filePath, tt.content)

			// Check security-sensitive detection
			if result.IsSecuritySensitive != tt.expectSecuritySensitive {
				t.Errorf("IsSecuritySensitive = %v, want %v\nDescription: %s",
					result.IsSecuritySensitive, tt.expectSecuritySensitive, tt.description)
			}

			// Check keyword count
			if len(result.MatchedKeywords) < tt.expectKeywords {
				t.Errorf("MatchedKeywords count = %d, want at least %d\nKeywords: %v\nDescription: %s",
					len(result.MatchedKeywords), tt.expectKeywords, result.MatchedKeywords, tt.description)
			}

			// Check path pattern count
			if len(result.MatchedPathPatterns) < tt.expectPathPatterns {
				t.Errorf("MatchedPathPatterns count = %d, want at least %d\nPatterns: %v\nDescription: %s",
					len(result.MatchedPathPatterns), tt.expectPathPatterns, result.MatchedPathPatterns, tt.description)
			}

			// Check force escalate
			if result.ShouldForceEscalate() != tt.expectForceEscalate {
				t.Errorf("ShouldForceEscalate = %v, want %v\nDescription: %s",
					result.ShouldForceEscalate(), tt.expectForceEscalate, tt.description)
			}

			// Check risk level
			if result.GetRiskLevel() != tt.expectRiskLevel {
				t.Errorf("GetRiskLevel = %v, want %v\nDescription: %s",
					result.GetRiskLevel(), tt.expectRiskLevel, tt.description)
			}

			// Log result for debugging
			t.Logf("Result: IsSecuritySensitive=%v, Keywords=%v, PathPatterns=%v, ForceEscalate=%v, RiskLevel=%v, Reason=%s",
				result.IsSecuritySensitive,
				result.MatchedKeywords,
				result.MatchedPathPatterns,
				result.ShouldForceEscalate(),
				result.GetRiskLevel(),
				result.Reason)
		})
	}
}

func TestSecurityKeywordsCoverage(t *testing.T) {
	// Test that all security keywords are properly defined
	if len(SecurityKeywords) == 0 {
		t.Error("SecurityKeywords slice is empty")
	}

	// Test that all path patterns are properly defined
	if len(SecurityPathPatterns) == 0 {
		t.Error("SecurityPathPatterns slice is empty")
	}

	t.Logf("Total security keywords: %d", len(SecurityKeywords))
	t.Logf("Total security path patterns: %d", len(SecurityPathPatterns))
	t.Logf("Keywords: %v", SecurityKeywords)
	t.Logf("Path patterns: %v", SecurityPathPatterns)
}

func TestDetectSecurityKeywords_CaseInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		content  string
	}{
		{
			name:     "uppercase keywords",
			filePath: "AUTH/HANDLERS.GO",
			content:  "FUNC LOGIN(USERNAME, PASSWORD STRING) ERROR",
		},
		{
			name:     "mixed case keywords",
			filePath: "internal/Auth/Handlers.go",
			content:  "func Login(userName, Password string) error",
		},
		{
			name:     "lowercase keywords",
			filePath: "internal/auth/handlers.go",
			content:  "func login(username, password string) error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectSecurityKeywords(tt.filePath, tt.content)

			if !result.IsSecuritySensitive {
				t.Errorf("Case-insensitive matching failed for %s", tt.name)
			}

			if len(result.MatchedKeywords) == 0 {
				t.Error("Expected keywords to be matched with case-insensitive search")
			}

			t.Logf("Matched keywords: %v", result.MatchedKeywords)
		})
	}
}

func TestSecurityDetectionResult_Reason(t *testing.T) {
	tests := []struct {
		name           string
		filePath       string
		content        string
		expectContains string
	}{
		{
			name:           "path-based detection",
			filePath:       "internal/auth/handlers.go",
			content:        "",
			expectContains: "security-sensitive file path detected",
		},
		{
			name:           "keyword-based detection",
			filePath:       "internal/handlers.go",
			content:        "func login(username, password string) error",
			expectContains: "security keywords detected",
		},
		{
			name:           "both path and keyword detection",
			filePath:       "internal/auth/handlers.go",
			content:        "func login(username, password string) error",
			expectContains: "security-sensitive file path detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectSecurityKeywords(tt.filePath, tt.content)

			if result.Reason == "" {
				t.Error("Expected non-empty reason for security-sensitive detection")
			}

			if !contains([]string{result.Reason}, tt.expectContains) && !strings.Contains(strings.ToLower(result.Reason), strings.ToLower(tt.expectContains)) {
				t.Errorf("Reason = %q, expected to contain %q", result.Reason, tt.expectContains)
			}

			t.Logf("Reason: %s", result.Reason)
		})
	}
}

// Benchmark security keyword detection performance
func BenchmarkDetectSecurityKeywords(b *testing.B) {
	filePath := "internal/auth/handlers.go"
	content := `
package auth

import (
	"crypto/bcrypt"
	"errors"
	"time"
)

func Login(username, password string) (*Session, error) {
	user, err := db.FindUserByUsername(username)
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	session := &Session{
		UserID:    user.ID,
		Token:     generateJWT(user),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	return session, nil
}

func generateJWT(user *User) string {
	// JWT generation logic
	return ""
}
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DetectSecurityKeywords(filePath, content)
	}
}
