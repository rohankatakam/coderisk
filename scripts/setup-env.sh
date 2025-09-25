#!/bin/bash

# CodeRisk Environment Setup Script
# Sets up .env file and validates configuration

set -e  # Exit on error

echo "🚀 CodeRisk Environment Setup"
echo "============================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if .env already exists
if [ -f ".env" ]; then
    echo -e "${YELLOW}⚠️  .env file already exists${NC}"
    read -p "Overwrite existing .env file? [y/N]: " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Using existing .env file..."
    else
        echo "Backing up existing .env to .env.backup..."
        cp .env .env.backup
    fi
fi

# Copy example if .env doesn't exist or user chose to overwrite
if [ ! -f ".env" ] || [[ $REPLY =~ ^[Yy]$ ]]; then
    if [ -f ".env.example" ]; then
        cp .env.example .env
        echo -e "${GREEN}✅ Created .env from .env.example${NC}"
    else
        echo -e "${RED}❌ .env.example not found${NC}"
        exit 1
    fi
fi

# Secure the .env file
chmod 600 .env
echo -e "${GREEN}✅ Secured .env file permissions (600)${NC}"

echo ""
echo "🔧 Configuration Setup"
echo "======================"

# Function to prompt for required settings
setup_github_token() {
    if grep -q "^GITHUB_TOKEN=ghp_your_github_token_here" .env || grep -q "^#.*GITHUB_TOKEN=" .env; then
        echo -e "${YELLOW}📝 GitHub Token Setup${NC}"
        echo "You need a GitHub Personal Access Token to use CodeRisk."
        echo "Get one at: https://github.com/settings/tokens"
        echo ""
        echo "Required scopes:"
        echo "  • 'repo' (for private repositories)"
        echo "  • 'public_repo' (for public repositories only)"
        echo ""

        read -p "Enter your GitHub token (ghp_... or github_pat_...): " -s github_token
        echo

        if [[ $github_token =~ ^ghp_[a-zA-Z0-9]{36}$ ]]; then
            # Classic personal access token
            escaped_token=$(echo "$github_token" | sed 's/[[\.*^$()+?{|]/\\&/g')
            sed -i.bak "s/^GITHUB_TOKEN=.*/GITHUB_TOKEN=$escaped_token/" .env
            rm .env.bak 2>/dev/null || true
            echo -e "${GREEN}✅ GitHub token configured (Classic PAT)${NC}"
        elif [[ $github_token =~ ^github_pat_[a-zA-Z0-9_]{68,}$ ]]; then
            # Fine-grained personal access token
            escaped_token=$(echo "$github_token" | sed 's/[[\.*^$()+?{|]/\\&/g')
            sed -i.bak "s/^GITHUB_TOKEN=.*/GITHUB_TOKEN=$escaped_token/" .env
            rm .env.bak 2>/dev/null || true
            echo -e "${GREEN}✅ GitHub token configured (Fine-grained PAT)${NC}"
        else
            echo -e "${YELLOW}⚠️  Token format not recognized${NC}"
            echo -e "${YELLOW}   Expected: ghp_... (classic) or github_pat_... (fine-grained)${NC}"
            echo -e "${YELLOW}   Proceeding anyway...${NC}"
            escaped_token=$(echo "$github_token" | sed 's/[[\.*^$()+?{|]/\\&/g')
            sed -i.bak "s/^GITHUB_TOKEN=.*/GITHUB_TOKEN=$escaped_token/" .env
            rm .env.bak 2>/dev/null || true
        fi
    else
        echo -e "${GREEN}✅ GitHub token already configured${NC}"
    fi
}

# Function to setup OpenAI (optional)
setup_openai() {
    echo ""
    read -p "Do you want to set up OpenAI for enhanced analysis? [y/N]: " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Get your OpenAI API key at: https://platform.openai.com/api-keys"
        read -p "Enter your OpenAI API key (sk-...): " -s openai_key
        echo

        if [[ $openai_key =~ ^sk- ]]; then
            escaped_key=$(echo "$openai_key" | sed 's/[[\.*^$()+?{|]/\\&/g')
            sed -i.bak "s/^OPENAI_API_KEY=.*/OPENAI_API_KEY=$escaped_key/" .env
            rm .env.bak 2>/dev/null || true
            echo -e "${GREEN}✅ OpenAI API key configured${NC}"
        else
            echo -e "${YELLOW}⚠️  API key format looks unusual (expected sk-...)${NC}"
            echo -e "${YELLOW}   Proceeding anyway...${NC}"
            escaped_key=$(echo "$openai_key" | sed 's/[[\.*^$()+?{|]/\\&/g')
            sed -i.bak "s/^OPENAI_API_KEY=.*/OPENAI_API_KEY=$escaped_key/" .env
            rm .env.bak 2>/dev/null || true
        fi
    fi
}

# Function to setup deployment mode
setup_deployment_mode() {
    echo ""
    echo "🏗️  Deployment Mode Setup"
    echo "Choose your deployment mode:"
    echo "1) local     - Single developer, local analysis only"
    echo "2) team      - Team collaboration with shared cache (recommended)"
    echo "3) enterprise - Enterprise deployment with custom endpoints"
    echo "4) oss       - Open source project with public cache"
    echo ""
    read -p "Select mode [1-4] (default: 2): " mode_choice

    case $mode_choice in
        1)
            sed -i.bak "s/^CODERISK_MODE=.*/CODERISK_MODE=local/" .env
            echo -e "${GREEN}✅ Set to local mode${NC}"
            ;;
        3)
            sed -i.bak "s/^CODERISK_MODE=.*/CODERISK_MODE=enterprise/" .env
            echo -e "${GREEN}✅ Set to enterprise mode${NC}"
            echo -e "${BLUE}💡 You'll need to configure custom endpoints in .env${NC}"
            ;;
        4)
            sed -i.bak "s/^CODERISK_MODE=.*/CODERISK_MODE=oss/" .env
            echo -e "${GREEN}✅ Set to OSS mode${NC}"
            ;;
        *)
            sed -i.bak "s/^CODERISK_MODE=.*/CODERISK_MODE=team/" .env
            echo -e "${GREEN}✅ Set to team mode (default)${NC}"
            ;;
    esac
    rm .env.bak 2>/dev/null || true
}

# Run setup functions
setup_github_token
setup_openai
setup_deployment_mode

# Validate configuration
echo ""
echo "🔍 Validating Configuration"
echo "==========================="

# Check if crisk binary exists
CRISK_BINARY=""
if [ -f "./bin/crisk" ]; then
    CRISK_BINARY="./bin/crisk"
elif command -v crisk &> /dev/null; then
    CRISK_BINARY="crisk"
else
    echo -e "${YELLOW}⚠️  crisk binary not found. Run 'make build' first.${NC}"
    CRISK_BINARY=""
fi

# Test configuration if binary exists
if [ -n "$CRISK_BINARY" ]; then
    echo "Testing configuration..."

    # Test basic config loading
    if $CRISK_BINARY config get github.token &>/dev/null; then
        echo -e "${GREEN}✅ Configuration loads successfully${NC}"
    else
        echo -e "${YELLOW}⚠️  Configuration may have issues${NC}"
    fi

    # Test GitHub token
    github_token_masked=$($CRISK_BINARY config get github.token 2>/dev/null || echo "not set")
    if [[ "$github_token_masked" == "not set" ]]; then
        echo -e "${RED}❌ GitHub token not detected${NC}"
    else
        echo -e "${GREEN}✅ GitHub token detected: $github_token_masked${NC}"
    fi

    echo ""
    echo "🎉 Setup Complete!"
    echo "=================="
    echo ""
    echo "Next steps:"
    echo "1. Navigate to a Git repository: cd /path/to/your/repo"
    echo "2. Initialize CodeRisk: $CRISK_BINARY init"
    echo "3. Check your code: $CRISK_BINARY check"
    echo ""
    echo "Useful commands:"
    echo "• $CRISK_BINARY status     - Show current status"
    echo "• $CRISK_BINARY config list - View all configuration"
    echo "• $CRISK_BINARY --help     - Show help"

else
    echo ""
    echo "🎉 Environment Setup Complete!"
    echo "=============================="
    echo ""
    echo "Next steps:"
    echo "1. Build CodeRisk: make build"
    echo "2. Navigate to a Git repository: cd /path/to/your/repo"
    echo "3. Initialize CodeRisk: ./bin/crisk init"
    echo "4. Check your code: ./bin/crisk check"
fi

echo ""
echo -e "${BLUE}📚 For more information, see ENVIRONMENT_SETUP.md${NC}"