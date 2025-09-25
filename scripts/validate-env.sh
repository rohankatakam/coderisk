#!/bin/bash

# CodeRisk Environment Validation Script
# Validates environment configuration and tests connectivity

set -e

echo "🔍 CodeRisk Environment Validation"
echo "=================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

ERRORS=0
WARNINGS=0

# Function to report error
error() {
    echo -e "${RED}❌ $1${NC}"
    ERRORS=$((ERRORS + 1))
}

# Function to report warning
warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
    WARNINGS=$((WARNINGS + 1))
}

# Function to report success
success() {
    echo -e "${GREEN}✅ $1${NC}"
}

# Function to report info
info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Load environment variables from .env files
load_env_files() {
    for env_file in ".env.local" ".env" ".env.example"; do
        if [ -f "$env_file" ]; then
            info "Loading $env_file"
            # Export variables from .env file (compatible with macOS)
            while IFS='=' read -r key value; do
                # Skip empty lines and comments
                [[ -z "$key" || "$key" =~ ^[[:space:]]*# ]] && continue
                # Remove quotes and export
                value="${value%\"}"
                value="${value#\"}"
                export "$key=$value"
            done < "$env_file"
            break
        fi
    done
}

# Check required files
check_files() {
    echo ""
    echo "📁 File Check"
    echo "============="

    if [ -f ".env" ]; then
        success ".env file exists"

        # Check permissions
        PERMS=$(ls -l .env | cut -d' ' -f1)
        if [[ "$PERMS" == "-rw-------" ]]; then
            success ".env file permissions are secure (600)"
        else
            warning ".env file permissions should be 600 for security"
            info "Run: chmod 600 .env"
        fi
    else
        error ".env file not found"
        info "Run: cp .env.example .env"
    fi

    if [ -f ".env.example" ]; then
        success ".env.example template exists"
    else
        warning ".env.example template not found"
    fi

    if [ -f "go.mod" ]; then
        success "Go module file exists"
    else
        error "go.mod not found - not in Go project directory?"
    fi
}

# Check Go installation and binary
check_go_setup() {
    echo ""
    echo "🐹 Go Setup Check"
    echo "================"

    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | cut -d' ' -f3)
        success "Go is installed: $GO_VERSION"
    else
        error "Go is not installed"
        info "Install Go from: https://golang.org/dl/"
        return 1
    fi

    if [ -f "./bin/crisk" ]; then
        success "crisk binary exists at ./bin/crisk"
    elif command -v crisk &> /dev/null; then
        success "crisk binary installed globally"
    else
        warning "crisk binary not found"
        info "Run: make build"
    fi
}

# Validate environment variables
check_env_vars() {
    echo ""
    echo "🔧 Environment Variables"
    echo "======================="

    # Load .env files
    load_env_files

    # Check GitHub token
    if [ -n "$GITHUB_TOKEN" ]; then
        if [[ "$GITHUB_TOKEN" =~ ^ghp_[a-zA-Z0-9]{36}$ ]]; then
            success "GITHUB_TOKEN format is correct (Classic PAT)"
        elif [[ "$GITHUB_TOKEN" =~ ^github_pat_[a-zA-Z0-9_]{68,}$ ]]; then
            success "GITHUB_TOKEN format is correct (Fine-grained PAT)"
        elif [[ "$GITHUB_TOKEN" == "ghp_your_github_token_here" ]]; then
            error "GITHUB_TOKEN still has placeholder value"
            info "Get a token from: https://github.com/settings/tokens"
        else
            warning "GITHUB_TOKEN format looks unusual"
            info "Expected: ghp_... (classic) or github_pat_... (fine-grained)"
        fi
    else
        error "GITHUB_TOKEN is not set"
        info "Set in .env: GITHUB_TOKEN=your_token_here"
    fi

    # Check OpenAI key (optional)
    if [ -n "$OPENAI_API_KEY" ]; then
        if [[ "$OPENAI_API_KEY" =~ ^sk- ]]; then
            success "OPENAI_API_KEY format is correct"
        elif [[ "$OPENAI_API_KEY" == "sk-your_openai_key_here" ]]; then
            warning "OPENAI_API_KEY still has placeholder value"
        else
            warning "OPENAI_API_KEY format looks unusual"
        fi
    else
        info "OPENAI_API_KEY not set (optional for Level 3 analysis)"
    fi

    # Check deployment mode
    if [ -n "$CODERISK_MODE" ]; then
        case "$CODERISK_MODE" in
            local|team|enterprise|oss)
                success "CODERISK_MODE is valid: $CODERISK_MODE"
                ;;
            *)
                warning "CODERISK_MODE has unusual value: $CODERISK_MODE"
                info "Expected: local, team, enterprise, or oss"
                ;;
        esac
    else
        info "CODERISK_MODE not set (will use default: team)"
    fi

    # Check storage type
    if [ -n "$STORAGE_TYPE" ]; then
        case "$STORAGE_TYPE" in
            sqlite|postgres)
                success "STORAGE_TYPE is valid: $STORAGE_TYPE"
                ;;
            *)
                warning "STORAGE_TYPE has unusual value: $STORAGE_TYPE"
                info "Expected: sqlite or postgres"
                ;;
        esac
    else
        info "STORAGE_TYPE not set (will use default: sqlite)"
    fi

    # Check budget limits
    if [ -n "$BUDGET_DAILY_LIMIT" ]; then
        if [[ "$BUDGET_DAILY_LIMIT" =~ ^[0-9]+\.?[0-9]*$ ]]; then
            success "BUDGET_DAILY_LIMIT is numeric: \$$BUDGET_DAILY_LIMIT"
        else
            warning "BUDGET_DAILY_LIMIT should be numeric: $BUDGET_DAILY_LIMIT"
        fi
    fi
}

# Test CodeRisk functionality
test_coderisk() {
    echo ""
    echo "🧪 CodeRisk Functionality Test"
    echo "=============================="

    # Find crisk binary
    CRISK_BINARY=""
    if [ -f "./bin/crisk" ]; then
        CRISK_BINARY="./bin/crisk"
    elif command -v crisk &> /dev/null; then
        CRISK_BINARY="crisk"
    else
        error "crisk binary not found for testing"
        return 1
    fi

    # Test basic functionality
    if $CRISK_BINARY --help &>/dev/null; then
        success "crisk --help works"
    else
        error "crisk --help failed"
        return 1
    fi

    # Test configuration loading
    if $CRISK_BINARY config list &>/dev/null; then
        success "Configuration loads successfully"
    else
        error "Configuration loading failed"
        return 1
    fi

    # Test GitHub token detection
    GITHUB_TOKEN_STATUS=$($CRISK_BINARY config get github.token 2>/dev/null || echo "not set")
    if [[ "$GITHUB_TOKEN_STATUS" == "not set" ]]; then
        error "GitHub token not detected by crisk"
    else
        success "GitHub token detected: $GITHUB_TOKEN_STATUS"
    fi

    # Test status command
    if $CRISK_BINARY status &>/dev/null; then
        success "crisk status works"
    else
        warning "crisk status has issues (may be normal if not initialized)"
    fi
}

# Check Git repository
check_git_repo() {
    echo ""
    echo "📝 Git Repository Check"
    echo "======================"

    if git rev-parse --git-dir &>/dev/null; then
        success "Current directory is a Git repository"

        REPO_URL=$(git remote get-url origin 2>/dev/null || echo "no remote")
        if [[ "$REPO_URL" != "no remote" ]]; then
            success "Git remote configured: $REPO_URL"
        else
            info "No Git remote configured"
        fi
    else
        info "Current directory is not a Git repository"
        info "Navigate to a Git repo to test repository-specific features"
    fi
}

# Network connectivity test (if possible)
test_connectivity() {
    echo ""
    echo "🌐 Network Connectivity"
    echo "======================"

    # Test GitHub API (if token is available)
    if [ -n "$GITHUB_TOKEN" ] && command -v curl &>/dev/null; then
        if curl -s -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user &>/dev/null; then
            success "GitHub API connectivity working"
        else
            error "GitHub API connectivity failed"
            info "Check your GitHub token and network connection"
        fi
    else
        info "Skipping GitHub API test (no curl or token)"
    fi

    # Test OpenAI API (if key is available)
    if [ -n "$OPENAI_API_KEY" ] && command -v curl &>/dev/null; then
        if curl -s -H "Authorization: Bearer $OPENAI_API_KEY" https://api.openai.com/v1/models &>/dev/null; then
            success "OpenAI API connectivity working"
        else
            warning "OpenAI API connectivity failed (check key and billing)"
        fi
    else
        info "Skipping OpenAI API test (no curl or key)"
    fi
}

# Main validation flow
main() {
    check_files
    check_go_setup
    check_env_vars
    test_coderisk
    check_git_repo
    test_connectivity

    echo ""
    echo "📊 Validation Summary"
    echo "===================="

    if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
        echo -e "${GREEN}🎉 Perfect! No issues found.${NC}"
        echo -e "${GREEN}✨ CodeRisk is ready to use!${NC}"
    elif [ $ERRORS -eq 0 ]; then
        echo -e "${YELLOW}✅ Good! $WARNINGS warning(s) found.${NC}"
        echo -e "${GREEN}🚀 CodeRisk should work correctly.${NC}"
    else
        echo -e "${RED}❌ $ERRORS error(s) and $WARNINGS warning(s) found.${NC}"
        echo -e "${RED}🛠️  Please fix the errors before using CodeRisk.${NC}"
    fi

    echo ""
    echo "Next steps:"
    if [ $ERRORS -gt 0 ]; then
        echo "1. Fix the errors listed above"
        echo "2. Run this script again: ./scripts/validate-env.sh"
    else
        echo "1. Navigate to a Git repository: cd /path/to/your/repo"
        echo "2. Initialize CodeRisk: crisk init"
        echo "3. Check your code: crisk check"
    fi
    echo ""
    echo "For help, see: ENVIRONMENT_SETUP.md"
}

# Run validation
main