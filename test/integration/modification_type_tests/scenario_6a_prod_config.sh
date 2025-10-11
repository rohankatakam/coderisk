#!/bin/bash
# Scenario 6A: Production Configuration Change (Type 3)
# Tests environment-aware risk assessment for production configs

set -e  # Exit on error

SCENARIO_NAME="Scenario 6A: Production Configuration Change (Type 3)"
TEST_DIR="test_sandbox/omnara"
TARGET_FILE=".env.production"
OUTPUT_FILE="test/integration/modification_type_tests/output_scenario_6a.txt"

echo "========================================"
echo "$SCENARIO_NAME"
echo "========================================"
echo ""

# 1. Find crisk binary
if [ -f "./bin/crisk" ]; then
    CRISK_BIN="./bin/crisk"
elif [ -f "./crisk" ]; then
    CRISK_BIN="./crisk"
else
    echo "ERROR: crisk binary not found"
    echo "Expected at: ./bin/crisk or ./crisk"
    echo "Please run: make build"
    exit 1
fi
echo "Using binary: $CRISK_BIN"

# 2. Navigate to test directory
cd "$TEST_DIR" || exit 1

# 3. Ensure clean git state
echo "Checking git status..."
if [ -n "$(git status --porcelain)" ]; then
    echo "ERROR: Git working directory not clean. Please commit or stash changes."
    git status --short
    exit 1
fi
echo "✅ Git working directory clean"
echo ""

# 4. Create .env.production file if it doesn't exist
echo "Creating $TARGET_FILE..."
cat > "$TARGET_FILE" << 'EOF'
# Production Environment Configuration
ENVIRONMENT=production

# Database - CRITICAL: Production database connection
PRODUCTION_DB_URL=postgresql://postgres:ORIGINALPASSWORD@db.production.com:5432/postgres

# API Configuration
API_PORT=8000
FRONTEND_URLS='["https://example.com", "https://www.example.com"]'

# Supabase Configuration
SUPABASE_URL=https://xxxxxxxxxxxx.supabase.co
SUPABASE_ANON_KEY=production-anon-key-here
SUPABASE_SERVICE_ROLE_KEY=production-service-role-key-here

# JWT Keys (production)
JWT_PRIVATE_KEY=production-jwt-private-key-here
JWT_PUBLIC_KEY=production-jwt-public-key-here

# Monitoring
SENTRY_DSN=https://123.us.sentry.io/456

# API Keys
ANTHROPIC_API_KEY=sk-ant-production-key-xxxxxxxxxxxxxxxxxxxxx
EOF

# Add to git staging to track the change
git add "$TARGET_FILE"

# Now modify it
echo "Modifying production database URL..."
sed -i.bak 's/ORIGINALPASSWORD/NEWPASSWORD123/' "$TARGET_FILE"

echo "✅ Changes applied successfully"
echo ""

# 5. Verify changes with git
echo "Verifying git diff..."
git diff --stat "$TARGET_FILE"
echo ""
echo "Git diff preview:"
git diff "$TARGET_FILE"
echo ""

# 6. Run crisk check
echo "Running crisk check..."
cd ../..  # Back to coderisk-go root

# Run crisk check and capture output
"$CRISK_BIN" check "$TEST_DIR/$TARGET_FILE" > "$OUTPUT_FILE" 2>&1 || true

echo "✅ crisk check completed"
echo ""

# 7. Display output
echo "========================================"
echo "ACTUAL OUTPUT:"
echo "========================================"
cat "$OUTPUT_FILE"
echo ""

# 8. Validation checks
echo "========================================"
echo "VALIDATION CHECKS:"
echo "========================================"

# Check for risk level
if grep -q "Risk Level:" "$OUTPUT_FILE"; then
    RISK_LEVEL=$(grep "Risk Level:" "$OUTPUT_FILE" | head -1)
    echo "✅ Risk level found: $RISK_LEVEL"
else
    echo "❌ Risk level missing from output"
fi

# Check for HIGH/CRITICAL risk (expected for production config)
if grep -qi "HIGH\|CRITICAL" "$OUTPUT_FILE"; then
    echo "✅ HIGH or CRITICAL risk detected (expected for production config)"
else
    echo "⚠️  Expected HIGH or CRITICAL risk for production configuration change"
fi

# Check for configuration type detection
if grep -qi "configuration\|config\|environment\|type 3" "$OUTPUT_FILE"; then
    echo "✅ Configuration change detected"
else
    echo "ℹ️  Configuration type not explicitly mentioned (Phase 0 not yet implemented)"
fi

# Check for production environment detection
if grep -qi "production\|prod" "$OUTPUT_FILE"; then
    echo "✅ Production environment context detected"
else
    echo "⚠️  Production environment not explicitly mentioned"
fi

# Check for sensitive value warnings
if grep -qi "sensitive\|credential\|password\|database" "$OUTPUT_FILE"; then
    echo "✅ Sensitive value context detected"
else
    echo "ℹ️  Sensitive value warnings not in output"
fi

echo ""

# 9. Reset changes
echo "Resetting changes..."
cd "$TEST_DIR"
git restore --staged "$TARGET_FILE"  # Unstage
rm -f "$TARGET_FILE"  # Remove the file
rm -f "$TARGET_FILE.bak"  # Remove sed backup

# 10. Verify clean state
if [ -n "$(git status --porcelain)" ]; then
    echo "⚠️  WARNING: Git working directory not fully restored"
    git status --short
else
    echo "✅ Git working directory restored to clean state"
fi
echo ""

echo "========================================"
echo "✅ Test completed"
echo "Output saved to: $OUTPUT_FILE"
echo "========================================"
echo ""
