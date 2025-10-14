#!/bin/bash

# Integration Test Script for CodeRisk Production Readiness
# Tests all critical components before release

set -e  # Exit on error

echo "🧪 CodeRisk Integration Testing"
echo "================================"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
PASSED=0
FAILED=0

# Test helper
test_command() {
    local test_name="$1"
    local command="$2"

    echo -n "Testing: $test_name... "

    if eval "$command" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PASSED${NC}"
        PASSED=$((PASSED + 1))
        return 0
    else
        echo -e "${RED}✗ FAILED${NC}"
        FAILED=$((FAILED + 1))
        return 1
    fi
}

# 1. Build Tests
echo "📦 Build Tests"
echo "-------------"

test_command "Go mod tidy" "go mod tidy"
test_command "Build CLI binary" "go build -o dist/crisk-test ./cmd/crisk"
test_command "Build API binary" "go build -o dist/api-test ./cmd/api"

echo ""

# 2. Unit Tests
echo "🧪 Unit Tests"
echo "------------"

test_command "Keyring unit tests" "go test -v ./internal/config -run TestKeyring"
test_command "Config unit tests" "go test -v ./internal/config -run TestConfig"
test_command "All tests pass" "go test ./..."

echo ""

# 3. Binary Tests
echo "🔧 Binary Tests"
echo "--------------"

test_command "CLI --version works" "./dist/crisk-test --version"
test_command "CLI --help works" "./dist/crisk-test --help"
test_command "Config command exists" "./dist/crisk-test config --help"
test_command "Configure command exists" "./dist/crisk-test configure --help"
test_command "Migrate command exists" "./dist/crisk-test migrate-to-keychain --help"

echo ""

# 4. Configuration Tests
echo "⚙️ Configuration Tests"
echo "--------------------"

# Create a temporary test environment
export CODERISK_TEST=1

test_command "Config list shows defaults" "./dist/crisk-test config list"

# Test config init in a temporary directory
test_command "Config init creates file" "cd /tmp && rm -rf .coderisk && mkdir -p .coderisk && ./dist/crisk-test config init --yes > /dev/null 2>&1 && [ -f .coderisk/config.yaml ]" || true

# If --yes flag doesn't exist, that's okay - skip the test
echo -e "${YELLOW}Note: Config init test requires user input${NC}"

echo ""

# 5. Keychain Tests (macOS only)
if [[ "$(uname)" == "Darwin" ]]; then
    echo "🔐 Keychain Tests (macOS)"
    echo "------------------------"

    # Clean up any existing test keychain entry
    security delete-generic-password -s CodeRisk-Test -a test 2>/dev/null || true

    test_command "Keyring is available" "[ \"$(./dist/crisk-test config list 2>&1 | grep -c 'keychain')\" -gt 0 ]"

    # Test keychain save (requires user approval on macOS)
    echo -e "${YELLOW}Note: Keychain tests may require user approval in GUI${NC}"

    echo ""
fi

# 6. Documentation Tests
echo "📚 Documentation Tests"
echo "--------------------"

test_command "README.md exists" "[ -f README.md ]"
test_command "CHANGELOG.md exists" "[ -f CHANGELOG.md ]"
test_command "LICENSE exists" "[ -f LICENSE ]"
test_command "install.sh exists" "[ -f install.sh ]"
test_command ".goreleaser.yml exists" "[ -f .goreleaser.yml ]"
test_command "GitHub Actions workflow exists" "[ -f .github/workflows/release.yml ]"

echo ""

# 7. GoReleaser Tests
echo "📦 GoReleaser Tests"
echo "------------------"

if command -v goreleaser &> /dev/null; then
    test_command "GoReleaser config is valid" "goreleaser check"
    echo -e "${YELLOW}Note: Full goreleaser build skipped (requires cross-compilation setup)${NC}"
else
    echo -e "${YELLOW}⚠ GoReleaser not installed - skipping validation${NC}"
fi

echo ""

# 8. Docker Tests
echo "🐳 Docker Tests"
echo "--------------"

if command -v docker &> /dev/null; then
    test_command "Dockerfile exists" "[ -f Dockerfile ]"
    test_command "Dockerfile is valid" "docker build -t coderisk-test -f Dockerfile ."

    # Clean up
    docker rmi coderisk-test 2>/dev/null || true
else
    echo -e "${YELLOW}⚠ Docker not available - skipping Docker tests${NC}"
fi

echo ""

# 9. Frontend Tests
echo "🌐 Frontend Tests"
echo "----------------"

if [ -d "/tmp/coderisk-frontend" ]; then
    cd /tmp/coderisk-frontend

    test_command "Frontend package.json exists" "[ -f package.json ]"
    test_command "Pricing page exists" "[ -f app/pricing/page.tsx ]"
    test_command "Open source page exists" "[ -f app/open-source/page.tsx ]"

    cd - > /dev/null
else
    echo -e "${YELLOW}⚠ Frontend directory not found - skipping frontend tests${NC}"
fi

echo ""

# 10. Summary
echo "================================"
echo "📊 Test Summary"
echo "================================"
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed! Ready for production.${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed. Please fix before release.${NC}"
    exit 1
fi
