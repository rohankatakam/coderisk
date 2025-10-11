#!/bin/bash
set -e

# CodeRisk Installation Script
# Installs the crisk CLI to ~/.local/bin

echo "🚀 Installing CodeRisk..."

# Check for Go
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21+ first:"
    echo "   https://golang.org/dl/"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.21"
if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "❌ Go version $GO_VERSION is too old. Please install Go 1.21+ first."
    exit 1
fi

# Create ~/.local/bin if it doesn't exist
mkdir -p ~/.local/bin

# Build and install
echo "📦 Building CodeRisk CLI..."
go build -o ~/.local/bin/crisk ./cmd/crisk

# Check if ~/.local/bin is in PATH
if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    echo ""
    echo "⚠️  ~/.local/bin is not in your PATH"
    echo ""
    echo "Add it by running one of these commands:"
    echo ""

    # Detect shell
    if [ -n "$ZSH_VERSION" ]; then
        echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.zshrc"
        echo "  source ~/.zshrc"
    elif [ -n "$BASH_VERSION" ]; then
        echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc"
        echo "  source ~/.bashrc"
    else
        echo "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.profile"
        echo "  source ~/.profile"
    fi
    echo ""
fi

echo "✅ CodeRisk installed successfully!"
echo ""
echo "📍 Installation location: ~/.local/bin/crisk"
echo ""
echo "🎯 Next steps:"
echo "   1. Start infrastructure: docker compose up -d"
echo "   2. Initialize a repository: cd /path/to/repo && crisk init-local"
echo "   3. Check for risks: crisk check"
echo ""
echo "📚 Full documentation: https://github.com/rohankatakam/coderisk-go"
