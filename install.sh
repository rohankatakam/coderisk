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

# Setup OpenAI API Key for Phase 2 LLM analysis
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔑 API Key Setup (Optional but Recommended)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "CodeRisk uses OpenAI for deep risk investigation (Phase 2)."
echo "Without an API key, you'll only get Phase 1 baseline checks."
echo ""
read -p "Do you have an OpenAI API key? (y/n): " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo ""
    read -p "Enter your OpenAI API key (starts with sk-...): " -r OPENAI_KEY

    if [ -n "$OPENAI_KEY" ]; then
        # Detect shell and add to appropriate config file
        SHELL_CONFIG=""
        if [ -n "$ZSH_VERSION" ] || [ "$SHELL" = "/bin/zsh" ] || [ "$SHELL" = "/usr/bin/zsh" ]; then
            SHELL_CONFIG="$HOME/.zshrc"
        elif [ -n "$BASH_VERSION" ] || [ "$SHELL" = "/bin/bash" ] || [ "$SHELL" = "/usr/bin/bash" ]; then
            SHELL_CONFIG="$HOME/.bashrc"
        else
            SHELL_CONFIG="$HOME/.profile"
        fi

        # Check if already exists
        if grep -q "export OPENAI_API_KEY=" "$SHELL_CONFIG" 2>/dev/null; then
            echo ""
            echo "⚠️  OPENAI_API_KEY already exists in $SHELL_CONFIG"
            read -p "Do you want to update it? (y/n): " -n 1 -r
            echo ""
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                # Remove old entry and add new one
                grep -v "export OPENAI_API_KEY=" "$SHELL_CONFIG" > "${SHELL_CONFIG}.tmp" 2>/dev/null || true
                mv "${SHELL_CONFIG}.tmp" "$SHELL_CONFIG"
                echo "export OPENAI_API_KEY=\"$OPENAI_KEY\"" >> "$SHELL_CONFIG"
                echo "✅ Updated OPENAI_API_KEY in $SHELL_CONFIG"
            fi
        else
            echo "export OPENAI_API_KEY=\"$OPENAI_KEY\"" >> "$SHELL_CONFIG"
            echo "✅ Added OPENAI_API_KEY to $SHELL_CONFIG"
        fi

        # Export for current session
        export OPENAI_API_KEY="$OPENAI_KEY"
        echo "✅ API key set for current session"
        echo ""
        echo "💡 Restart your shell or run: source $SHELL_CONFIG"
    fi
else
    echo ""
    echo "⏭️  Skipping API key setup."
    echo ""
    echo "To add it later, run:"
    echo "  export OPENAI_API_KEY=\"sk-your-key-here\""
    echo ""
    echo "Or add this line to your shell config (~/.zshrc or ~/.bashrc):"
    echo "  export OPENAI_API_KEY=\"sk-your-key-here\""
    echo ""
    echo "Get your API key at: https://platform.openai.com/api-keys"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🎯 Next Steps"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "1. Start infrastructure:"
echo "   docker compose up -d"
echo ""
echo "2. Initialize a repository:"
echo "   cd /path/to/your/repo"
echo "   crisk init-local"
echo ""
echo "3. Check for risks:"
echo "   crisk check                    # Quick baseline check"
echo "   crisk check --explain          # With Phase 2 LLM analysis"
echo "   crisk check --ai-mode          # JSON output for AI tools"
echo ""
echo "4. Install pre-commit hook (optional):"
echo "   crisk hook install"
echo ""
echo "📚 Full documentation: https://github.com/rohankatakam/coderisk-go"
echo ""
