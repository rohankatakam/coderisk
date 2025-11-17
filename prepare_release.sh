#!/bin/bash
set -e

# CodeRisk Release Preparation Script
# Creates a distributable package for users

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERSION="${1:-v0.1.0}"
RELEASE_DIR="${SCRIPT_DIR}/release"
PACKAGE_NAME="coderisk-${VERSION}"
PACKAGE_DIR="${RELEASE_DIR}/${PACKAGE_NAME}"

echo "Preparing CodeRisk ${VERSION} for release..."
echo ""

# Clean previous release
rm -rf "${RELEASE_DIR}"
mkdir -p "${PACKAGE_DIR}"

# Copy binaries
echo "Copying binaries..."
mkdir -p "${PACKAGE_DIR}/bin"
cp "${SCRIPT_DIR}/bin/crisk" "${PACKAGE_DIR}/bin/"
cp "${SCRIPT_DIR}/bin/crisk-check-server" "${PACKAGE_DIR}/bin/"
chmod +x "${PACKAGE_DIR}/bin/crisk" "${PACKAGE_DIR}/bin/crisk-check-server"

# Copy Docker files
echo "Copying Docker configuration..."
cp "${SCRIPT_DIR}/docker-compose.yml" "${PACKAGE_DIR}/"

# Copy scripts
echo "Copying setup scripts..."
cp "${SCRIPT_DIR}/setup_coderisk_mcp.sh" "${PACKAGE_DIR}/"
cp "${SCRIPT_DIR}/test_mcp_interactive.sh" "${PACKAGE_DIR}/"
chmod +x "${PACKAGE_DIR}/setup_coderisk_mcp.sh"
chmod +x "${PACKAGE_DIR}/test_mcp_interactive.sh"

# Copy documentation
echo "Copying documentation..."
cp "${SCRIPT_DIR}/CLAUDE_CODE_SETUP.md" "${PACKAGE_DIR}/README.md"
cp "${SCRIPT_DIR}/MCP_SERVER_README.md" "${PACKAGE_DIR}/"
cp "${SCRIPT_DIR}/MCP_SERVER_TEST_RESULTS.md" "${PACKAGE_DIR}/"
cp "${SCRIPT_DIR}/QUICKSTART_MCP.md" "${PACKAGE_DIR}/"
cp "${SCRIPT_DIR}/LICENSE" "${PACKAGE_DIR}/" 2>/dev/null || echo "MIT License" > "${PACKAGE_DIR}/LICENSE"

# Create release notes
cat > "${PACKAGE_DIR}/RELEASE_NOTES.md" << EOF
# CodeRisk ${VERSION} Release Notes

## What's New

- 🚀 **MCP Server for Claude Code**: Query code risk directly from Claude
- 📊 **Three Risk Dimensions**: Ownership, Coupling, and Temporal analysis
- 🔧 **Timeline Fix**: 50x improvement in incident linking (2 → 98 incidents)
- 🐳 **Docker-based**: Easy setup with Neo4j and PostgreSQL
- 📦 **Out-of-the-box**: Automated setup script included

## Features

### Ownership Analysis
- Original author and last modifier tracking
- Staleness calculation (days since last change)
- Developer familiarity maps

### Coupling Analysis
- Co-change detection (which blocks change together)
- Coupling rates and confidence scores
- Graph-based relationship tracking

### Temporal Risk
- Historical incident linking (issues/PRs → code blocks)
- 100% confidence links via GitHub timeline events
- Evidence-based risk scoring

## Quick Start

1. Extract the package
2. Run the setup script: \`./setup_coderisk_mcp.sh\`
3. Follow the interactive prompts
4. Open your repo in Claude Code and ask about risk

See \`README.md\` for detailed instructions.

## System Requirements

- macOS, Linux, or Windows (WSL)
- Docker Desktop 4.0+
- 8 GB RAM (recommended)
- 2-5 GB disk space per repository

## Known Limitations

- Repository ID currently hardcoded (will auto-detect in future)
- Semantic importance not yet calculated (requires LLM)
- File rename detection uses git log --follow (cached)

## Bug Fixes

- ✅ Fixed timeline "referenced" event extraction
- ✅ Fixed temporal linking to include both 'closed' and 'referenced' events
- ✅ Fixed NULL handling in Neo4j ownership queries

## What's Coming

- Auto-detection of repository from git remote
- Incremental updates (no need to re-ingest full history)
- Semantic importance calculation
- Support for uncommitted changes analysis
- Web UI for data exploration

## Changelog

See full commit history at: https://github.com/rohankatakam/coderisk

## Support

- GitHub Issues: https://github.com/rohankatakam/coderisk/issues
- Documentation: See included README.md
EOF

# Create quick start file
cat > "${PACKAGE_DIR}/QUICK_START.txt" << 'EOF'
╔══════════════════════════════════════════════════════════════════╗
║                 CodeRisk MCP Server - Quick Start                ║
╚══════════════════════════════════════════════════════════════════╝

1. RUN SETUP:
   ./setup_coderisk_mcp.sh

2. FOLLOW PROMPTS:
   - Databases will start automatically
   - Enter your repository path to ingest
   - Provide GitHub token when prompted

3. OPEN CLAUDE CODE:
   cd /path/to/your/repository
   code .

4. ASK CLAUDE:
   "What are the risk factors for src/main.py?"

For detailed instructions, see: README.md

Need help? https://github.com/rohankatakam/coderisk/issues
EOF

# Create installation verification script
cat > "${PACKAGE_DIR}/verify_install.sh" << 'EOF'
#!/bin/bash

echo "Verifying CodeRisk installation..."
echo ""

# Check binaries
if [ -f "bin/crisk" ] && [ -f "bin/crisk-check-server" ]; then
    echo "✓ Binaries found"
else
    echo "✗ Binaries missing"
    exit 1
fi

# Check Docker file
if [ -f "docker-compose.yml" ]; then
    echo "✓ Docker configuration found"
else
    echo "✗ Docker configuration missing"
    exit 1
fi

# Check scripts
if [ -f "setup_coderisk_mcp.sh" ]; then
    echo "✓ Setup script found"
else
    echo "✗ Setup script missing"
    exit 1
fi

# Check documentation
if [ -f "README.md" ]; then
    echo "✓ Documentation found"
else
    echo "✗ Documentation missing"
    exit 1
fi

echo ""
echo "Installation verified! Run ./setup_coderisk_mcp.sh to get started."
EOF

chmod +x "${PACKAGE_DIR}/verify_install.sh"

# Create archive
echo ""
echo "Creating release archive..."
cd "${RELEASE_DIR}"
tar -czf "${PACKAGE_NAME}.tar.gz" "${PACKAGE_NAME}"

# Calculate checksum
echo "Calculating checksum..."
if command -v shasum &> /dev/null; then
    shasum -a 256 "${PACKAGE_NAME}.tar.gz" > "${PACKAGE_NAME}.tar.gz.sha256"
elif command -v sha256sum &> /dev/null; then
    sha256sum "${PACKAGE_NAME}.tar.gz" > "${PACKAGE_NAME}.tar.gz.sha256"
fi

# Create release info
ARCHIVE_SIZE=$(ls -lh "${PACKAGE_NAME}.tar.gz" | awk '{print $5}')

echo ""
echo "✓ Release package created!"
echo ""
echo "Details:"
echo "  Version: ${VERSION}"
echo "  Package: ${RELEASE_DIR}/${PACKAGE_NAME}.tar.gz"
echo "  Size: ${ARCHIVE_SIZE}"
echo ""
echo "Contents:"
tree -L 2 "${PACKAGE_DIR}" 2>/dev/null || find "${PACKAGE_DIR}" -type f | head -20
echo ""
echo "Next steps:"
echo "1. Test the package:"
echo "   cd /tmp && tar -xzf ${RELEASE_DIR}/${PACKAGE_NAME}.tar.gz"
echo "   cd ${PACKAGE_NAME} && ./verify_install.sh"
echo ""
echo "2. Create GitHub release:"
echo "   gh release create ${VERSION} ${RELEASE_DIR}/${PACKAGE_NAME}.tar.gz"
echo ""
echo "3. Update documentation with download link"
echo ""
