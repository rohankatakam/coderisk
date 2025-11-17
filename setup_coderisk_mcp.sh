#!/bin/bash
set -e

# CodeRisk MCP Server Setup Script
# This script automates the setup of CodeRisk for Claude Code integration

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CODERISK_BIN="${SCRIPT_DIR}/bin"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo ""
echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   CodeRisk MCP Server Setup Wizard    ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""

# Step 1: Check prerequisites
echo -e "${BLUE}Step 1/5: Checking prerequisites...${NC}"
echo ""

# Check Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}✗ Docker not found${NC}"
    echo "  Please install Docker Desktop: https://www.docker.com/products/docker-desktop"
    exit 1
fi
echo -e "${GREEN}✓ Docker installed${NC}"

# Check if Docker is running
if ! docker ps &> /dev/null; then
    echo -e "${RED}✗ Docker is not running${NC}"
    echo "  Please start Docker Desktop and try again"
    exit 1
fi
echo -e "${GREEN}✓ Docker is running${NC}"

# Check Claude Code
if ! command -v claude &> /dev/null; then
    echo -e "${YELLOW}⚠ Claude Code CLI not found${NC}"
    echo "  You'll need to install Claude Code from https://claude.ai/code"
    echo "  You can still continue setup and add the MCP server manually later."
    echo ""
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
    CLAUDE_AVAILABLE=false
else
    echo -e "${GREEN}✓ Claude Code CLI found${NC}"
    CLAUDE_AVAILABLE=true
fi

# Check binaries
if [ ! -f "${CODERISK_BIN}/crisk" ] || [ ! -f "${CODERISK_BIN}/crisk-check-server" ]; then
    echo -e "${RED}✗ CodeRisk binaries not found${NC}"
    echo "  Expected location: ${CODERISK_BIN}"
    echo "  Please build the project first: make build"
    exit 1
fi
echo -e "${GREEN}✓ CodeRisk binaries found${NC}"

# Make sure binaries are executable
chmod +x "${CODERISK_BIN}/crisk" "${CODERISK_BIN}/crisk-check-server"

echo ""

# Step 2: Start databases
echo -e "${BLUE}Step 2/5: Starting databases...${NC}"
echo ""

# Check if databases are already running
POSTGRES_RUNNING=$(docker ps --filter "name=coderisk-postgres" --filter "status=running" -q)
NEO4J_RUNNING=$(docker ps --filter "name=coderisk-neo4j" --filter "status=running" -q)

if [ -n "$POSTGRES_RUNNING" ] && [ -n "$NEO4J_RUNNING" ]; then
    echo -e "${GREEN}✓ Databases already running${NC}"
else
    echo "Starting Docker containers..."
    cd "${SCRIPT_DIR}"
    docker-compose up -d

    echo "Waiting for databases to be ready (30 seconds)..."
    sleep 30

    # Verify PostgreSQL
    if ! docker exec coderisk-postgres pg_isready -U coderisk &> /dev/null; then
        echo -e "${RED}✗ PostgreSQL failed to start${NC}"
        echo "  Check logs: docker logs coderisk-postgres"
        exit 1
    fi
    echo -e "${GREEN}✓ PostgreSQL started${NC}"

    # Verify Neo4j (check if port is responding)
    if ! docker exec coderisk-neo4j cypher-shell -u neo4j -p CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 "RETURN 1;" &> /dev/null; then
        echo -e "${YELLOW}⚠ Neo4j may still be starting (this is normal)${NC}"
        echo "  It will be ready shortly..."
    fi
    echo -e "${GREEN}✓ Neo4j started${NC}"
fi

echo ""

# Step 3: Repository ingestion
echo -e "${BLUE}Step 3/5: Repository ingestion${NC}"
echo ""
echo "To use CodeRisk with Claude Code, you need to ingest a git repository."
echo ""
read -p "Do you want to ingest a repository now? (y/n) " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    # Ask for repository path
    echo ""
    read -p "Enter the path to your git repository: " REPO_PATH

    # Expand ~ to home directory
    REPO_PATH="${REPO_PATH/#\~/$HOME}"

    # Verify it's a git repository
    if [ ! -d "${REPO_PATH}/.git" ]; then
        echo -e "${RED}✗ Not a git repository: ${REPO_PATH}${NC}"
        echo "  Skipping ingestion. You can run it manually later:"
        echo "  cd /path/to/your/repo && ${CODERISK_BIN}/crisk init --days 365"
    else
        # Ask for GitHub token
        echo ""
        echo "A GitHub Personal Access Token is required to fetch repository data."
        echo "Create one at: https://github.com/settings/tokens (needs 'repo' scope)"
        echo ""
        read -p "Enter your GitHub token: " GITHUB_TOKEN

        # Ask for Gemini API key (optional)
        echo ""
        echo "Gemini API key is optional for AI-powered analysis."
        read -p "Enter Gemini API key (or press Enter to skip): " GEMINI_API_KEY

        # Run ingestion
        echo ""
        echo -e "${BLUE}Starting ingestion (this may take 10-30 minutes)...${NC}"
        echo ""

        cd "${REPO_PATH}"

        export GITHUB_TOKEN
        export NEO4J_URI="bolt://localhost:7688"
        export NEO4J_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"
        export POSTGRES_HOST="localhost"
        export POSTGRES_PORT="5433"
        export POSTGRES_DB="coderisk"
        export POSTGRES_USER="coderisk"
        export POSTGRES_PASSWORD="CHANGE_THIS_PASSWORD_IN_PRODUCTION_123"

        if [ -n "$GEMINI_API_KEY" ]; then
            export GEMINI_API_KEY
            export LLM_PROVIDER="gemini"
        fi

        if "${CODERISK_BIN}/crisk" init --days 365; then
            echo ""
            echo -e "${GREEN}✓ Repository ingested successfully${NC}"
        else
            echo ""
            echo -e "${RED}✗ Ingestion failed${NC}"
            echo "  You can retry manually:"
            echo "  cd ${REPO_PATH} && ${CODERISK_BIN}/crisk init --days 365"
        fi
    fi
else
    echo ""
    echo -e "${YELLOW}⚠ Skipped ingestion${NC}"
    echo "  You can ingest a repository later with:"
    echo "  cd /path/to/your/repo && ${CODERISK_BIN}/crisk init --days 365"
fi

echo ""

# Step 4: Add to Claude Code
echo -e "${BLUE}Step 4/5: Adding to Claude Code${NC}"
echo ""

if [ "$CLAUDE_AVAILABLE" = false ]; then
    echo -e "${YELLOW}⚠ Claude Code CLI not available${NC}"
    echo "  After installing Claude Code, run:"
    echo ""
    echo "  claude mcp add --transport stdio coderisk \\"
    echo "    --env NEO4J_URI=bolt://localhost:7688 \\"
    echo "    --env NEO4J_PASSWORD=CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \\"
    echo "    --env POSTGRES_DSN=postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable \\"
    echo "    -- ${CODERISK_BIN}/crisk-check-server"
    echo ""
else
    # Check if already added
    if claude mcp list 2>/dev/null | grep -q "coderisk"; then
        echo -e "${YELLOW}⚠ CodeRisk MCP server already configured${NC}"
        echo ""
        read -p "Do you want to update it? (y/n) " -n 1 -r
        echo ""
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            claude mcp remove coderisk
            echo "Removed old configuration"
        else
            echo "Keeping existing configuration"
            echo ""
            echo -e "${BLUE}Step 5/5: Setup complete!${NC}"
            echo ""
            echo -e "${GREEN}✓ All done!${NC}"
            echo ""
            echo "Next steps:"
            echo "1. Open your repository in Claude Code"
            echo "2. Ask Claude: 'What are the risk factors for <filename>?'"
            echo ""
            exit 0
        fi
    fi

    # Add MCP server
    echo "Adding CodeRisk to Claude Code..."
    if claude mcp add --transport stdio coderisk \
        --env NEO4J_URI=bolt://localhost:7688 \
        --env NEO4J_PASSWORD=CHANGE_THIS_PASSWORD_IN_PRODUCTION_123 \
        --env POSTGRES_DSN=postgres://coderisk:CHANGE_THIS_PASSWORD_IN_PRODUCTION_123@localhost:5433/coderisk?sslmode=disable \
        -- "${CODERISK_BIN}/crisk-check-server"; then
        echo -e "${GREEN}✓ Added to Claude Code${NC}"
    else
        echo -e "${RED}✗ Failed to add to Claude Code${NC}"
        echo "  You can add it manually later with the command above"
    fi
fi

echo ""

# Step 5: Completion
echo -e "${BLUE}Step 5/5: Setup complete!${NC}"
echo ""
echo -e "${GREEN}✓ All done!${NC}"
echo ""
echo "Next steps:"
echo "1. Open your repository in Claude Code"
echo "   ${YELLOW}cd /path/to/your/repo && code .${NC}"
echo ""
echo "2. Ask Claude about your code:"
echo "   ${YELLOW}What are the risk factors for src/main.py?${NC}"
echo "   ${YELLOW}Which code blocks have high coupling?${NC}"
echo "   ${YELLOW}Show me ownership details for this file${NC}"
echo ""
echo "Useful commands:"
echo "  • Test MCP server: ${YELLOW}${SCRIPT_DIR}/test_mcp_interactive.sh${NC}"
echo "  • Check databases: ${YELLOW}docker ps | grep coderisk${NC}"
echo "  • View Neo4j: ${YELLOW}open http://localhost:7475${NC}"
echo "  • Ingest new repo: ${YELLOW}cd /path/to/repo && ${CODERISK_BIN}/crisk init --days 365${NC}"
echo ""
echo "For more help, see: ${YELLOW}${SCRIPT_DIR}/CLAUDE_CODE_SETUP.md${NC}"
echo ""
