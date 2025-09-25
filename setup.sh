#!/bin/bash

echo "🚀 Setting up CodeRisk Go development environment..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed"
    echo "Please install Go first:"
    echo ""
    echo "Option 1 - Using Homebrew:"
    echo "  brew install go"
    echo ""
    echo "Option 2 - Download from https://golang.org/dl/"
    echo ""
    echo "Then add to your shell profile (~/.zshrc or ~/.bash_profile):"
    echo "  export PATH=/usr/local/go/bin:\$PATH"
    echo "  export GOPATH=\$HOME/go"
    echo "  export PATH=\$PATH:\$GOPATH/bin"
    echo ""
    echo "After installation, run this script again."
    exit 1
fi

echo "✅ Go is installed: $(go version)"

# Download dependencies
echo "📦 Downloading Go dependencies..."
go mod download
go mod tidy

# Create necessary directories
echo "📁 Creating required directories..."
mkdir -p bin
mkdir -p logs
mkdir -p .coderisk/cache

# Check if build works
echo "🔨 Testing build..."
if go build -o bin/crisk ./cmd/crisk; then
    echo "✅ Build successful!"
    echo ""
    echo "🎉 Setup complete! You can now:"
    echo "  1. Run: ./bin/crisk --help"
    echo "  2. Or use: make build"
    echo "  3. Initialize: ./bin/crisk init --help"
    echo ""
    echo "💡 Next steps:"
    echo "  - Set GITHUB_TOKEN environment variable"
    echo "  - Optionally set OPENAI_API_KEY"
    echo "  - Run 'crisk init' in a Git repository"
else
    echo "❌ Build failed. Please check the error messages above."
    exit 1
fi