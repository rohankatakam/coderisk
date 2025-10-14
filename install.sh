#!/bin/bash
# CodeRisk Universal Installer
# Downloads and installs pre-built binaries from GitHub Releases
# POSIX-compliant for maximum compatibility
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# GitHub repository details
REPO_OWNER="rohankatakam"
REPO_NAME="coderisk-go"
BINARY_NAME="crisk"

# Default installation directory
INSTALL_DIR="${HOME}/.local/bin"

# Print colored message
print_message() {
    local color=$1
    shift
    echo -e "${color}$*${NC}"
}

# Print error and exit
error_exit() {
    print_message "$RED" "❌ Error: $*"
    exit 1
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Darwin*)
            echo "darwin"
            ;;
        Linux*)
            echo "linux"
            ;;
        MINGW*|MSYS*|CYGWIN*)
            echo "windows"
            ;;
        *)
            error_exit "Unsupported operating system: $(uname -s)"
            ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)
            echo "x86_64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        i386|i686)
            error_exit "32-bit systems are not supported"
            ;;
        *)
            error_exit "Unsupported architecture: $arch"
            ;;
    esac
}

# Get latest release version from GitHub API
get_latest_version() {
    local version
    if command -v curl > /dev/null 2>&1; then
        version=$(curl -sSf "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest" | \
            grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget > /dev/null 2>&1; then
        version=$(wget -qO- "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest" | \
            grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        error_exit "curl or wget is required but not installed"
    fi

    if [ -z "$version" ]; then
        error_exit "Failed to fetch latest version from GitHub"
    fi

    echo "$version"
}

# Download file with retry
download_file() {
    local url=$1
    local output=$2
    local retries=3
    local count=0

    while [ $count -lt $retries ]; do
        if command -v curl > /dev/null 2>&1; then
            if curl -fsSL "$url" -o "$output"; then
                return 0
            fi
        elif command -v wget > /dev/null 2>&1; then
            if wget -q "$url" -O "$output"; then
                return 0
            fi
        fi
        count=$((count + 1))
        if [ $count -lt $retries ]; then
            print_message "$YELLOW" "⏳ Retry $count/$retries..."
            sleep 2
        fi
    done

    error_exit "Failed to download $url after $retries attempts"
}

# Verify checksum
verify_checksum() {
    local file=$1
    local checksum_file=$2
    local filename
    filename="$(basename "$file")"

    # Extract expected checksum for this file
    local expected_checksum
    expected_checksum=$(grep "$filename" "$checksum_file" | awk '{print $1}')

    if [ -z "$expected_checksum" ]; then
        print_message "$YELLOW" "⚠️  Warning: Checksum not found in checksums.txt"
        return 0
    fi

    # Calculate actual checksum
    local actual_checksum
    if command -v shasum > /dev/null 2>&1; then
        actual_checksum=$(shasum -a 256 "$file" | awk '{print $1}')
    elif command -v sha256sum > /dev/null 2>&1; then
        actual_checksum=$(sha256sum "$file" | awk '{print $1}')
    else
        print_message "$YELLOW" "⚠️  Warning: sha256sum not available, skipping verification"
        return 0
    fi

    if [ "$expected_checksum" != "$actual_checksum" ]; then
        error_exit "Checksum mismatch! Expected: $expected_checksum, Got: $actual_checksum"
    fi

    print_message "$GREEN" "✅ Checksum verified"
}

# Check if directory is in PATH
is_in_path() {
    case ":${PATH}:" in
        *:"$1":*)
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

# Add directory to PATH in shell config
add_to_path() {
    local dir=$1
    local shell_config

    # Detect shell config file
    if [ -n "$ZSH_VERSION" ] || [ "$SHELL" = "/bin/zsh" ] || [ "$SHELL" = "/usr/bin/zsh" ]; then
        shell_config="$HOME/.zshrc"
    elif [ -n "$BASH_VERSION" ] || [ "$SHELL" = "/bin/bash" ] || [ "$SHELL" = "/usr/bin/bash" ]; then
        shell_config="$HOME/.bashrc"
    else
        shell_config="$HOME/.profile"
    fi

    echo ""
    print_message "$YELLOW" "⚠️  $dir is not in your PATH"
    echo ""
    print_message "$BLUE" "To add it, run:"
    echo "  echo 'export PATH=\"$dir:\$PATH\"' >> $shell_config"
    echo "  source $shell_config"
    echo ""
}

# Interactive API key setup with keychain support
setup_api_key() {
    echo ""
    print_message "$BLUE" "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    print_message "$BLUE" "🔑 API Key Setup (PROFESSIONAL SECURITY)"
    print_message "$BLUE" "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "CodeRisk requires an OpenAI API key for LLM-guided risk assessment."
    echo "Cost: \$0.03-0.05 per check (~\$3-5/month for 100 checks)"
    echo ""
    echo "Setup Options:"
    echo "  1. Interactive wizard (recommended, with OS keychain support)"
    echo "  2. Quick setup (save to config file)"
    echo "  3. Skip (configure later)"
    echo ""
    read -p "Choose option (1-3): " -n 1 -r SETUP_CHOICE
    echo ""
    echo ""

    case $SETUP_CHOICE in
        1)
            # Run interactive wizard
            print_message "$BLUE" "🔧 Starting configuration wizard..."
            "$INSTALL_DIR/$BINARY_NAME" configure
            ;;
        2)
            # Quick setup - save to config file
            read -p "Enter your OpenAI API key (starts with sk-...): " -r OPENAI_KEY

            if [ -n "$OPENAI_KEY" ]; then
                mkdir -p "$HOME/.coderisk"
                cat > "$HOME/.coderisk/config.yaml" <<EOF
api:
  openai_key: "$OPENAI_KEY"
  openai_model: "gpt-4o-mini"
  use_keychain: false
budget:
  daily_limit: 2.00
  monthly_limit: 60.00
EOF
                print_message "$GREEN" "✅ API key saved to ~/.coderisk/config.yaml"
                echo ""
                print_message "$YELLOW" "💡 For better security, run: crisk migrate-to-keychain"
            fi
            ;;
        3)
            print_message "$YELLOW" "⏭️  Skipping API key setup."
            echo ""
            print_message "$YELLOW" "⚠️  CodeRisk requires an API key to function!"
            echo ""
            echo "To configure later, run:"
            echo "  crisk configure"
            echo ""
            echo "Or get your API key at: https://platform.openai.com/api-keys"
            ;;
    esac
}

# Main installation logic
main() {
    print_message "$BLUE" "🚀 Installing CodeRisk CLI..."
    echo ""

    # Detect platform
    local os
    local arch
    os=$(detect_os)
    arch=$(detect_arch)
    print_message "$GREEN" "✅ Detected: $os $arch"

    # Get latest version
    print_message "$BLUE" "📡 Fetching latest version..."
    local version
    version=$(get_latest_version)
    print_message "$GREEN" "✅ Latest version: $version"

    # Construct download URLs
    local archive_name="${BINARY_NAME}_${os}_${arch}.tar.gz"
    if [ "$os" = "windows" ]; then
        archive_name="${BINARY_NAME}_${os}_${arch}.zip"
    fi

    local download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/${archive_name}"
    local checksum_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/checksums.txt"

    # Create temporary directory
    local tmp_dir
    tmp_dir=$(mktemp -d 2>/dev/null || mktemp -d -t 'crisk-install')
    trap 'rm -rf "$tmp_dir"' EXIT

    # Download archive
    print_message "$BLUE" "📦 Downloading $archive_name..."
    download_file "$download_url" "$tmp_dir/$archive_name"

    # Download checksums
    print_message "$BLUE" "🔐 Downloading checksums..."
    download_file "$checksum_url" "$tmp_dir/checksums.txt"

    # Verify checksum
    print_message "$BLUE" "🔍 Verifying checksum..."
    verify_checksum "$tmp_dir/$archive_name" "$tmp_dir/checksums.txt"

    # Extract archive
    print_message "$BLUE" "📂 Extracting archive..."
    if [ "$os" = "windows" ]; then
        unzip -q "$tmp_dir/$archive_name" -d "$tmp_dir"
    else
        tar -xzf "$tmp_dir/$archive_name" -C "$tmp_dir"
    fi

    # Create install directory
    mkdir -p "$INSTALL_DIR"

    # Install binary
    print_message "$BLUE" "📥 Installing to $INSTALL_DIR/$BINARY_NAME..."
    cp "$tmp_dir/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"

    print_message "$GREEN" "✅ CodeRisk installed successfully!"
    echo ""
    print_message "$BLUE" "📍 Installation location: $INSTALL_DIR/$BINARY_NAME"

    # Check PATH
    if ! is_in_path "$INSTALL_DIR"; then
        add_to_path "$INSTALL_DIR"
    fi

    # Verify installation
    if command -v "$BINARY_NAME" > /dev/null 2>&1; then
        echo ""
        print_message "$GREEN" "🎉 Verification: $("$BINARY_NAME" --version 2>&1 | head -n 1)"
    fi

    # Interactive API key setup
    setup_api_key

    # Next steps
    echo ""
    print_message "$BLUE" "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    print_message "$BLUE" "🎯 Next Steps (17 minutes one-time per repo)"
    print_message "$BLUE" "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "Setup required for CodeRisk to function:"
    echo ""
    echo "1. Start infrastructure (2 minutes, REQUIRED):"
    echo "   docker compose up -d"
    echo ""
    echo "2. Initialize a repository (10-15 minutes, REQUIRED):"
    echo "   cd /path/to/your/repo"
    echo "   crisk init-local"
    echo "   # Builds graph: Tree-sitter AST + Git history"
    echo ""
    echo "3. Check for risks (2-5 seconds):"
    echo "   crisk check                    # Quick baseline check"
    echo "   crisk check --explain          # Detailed LLM investigation"
    echo ""
    echo "What you get:"
    echo "  ✅ <3% false positive rate (vs 10-20% industry standard)"
    echo "  ✅ 2-5 second checks (agentic graph search)"
    echo "  ✅ Transparent costs (\$0.03-0.05/check, BYOK)"
    echo ""
    print_message "$BLUE" "📚 Full documentation: https://github.com/${REPO_OWNER}/${REPO_NAME}"
    echo ""
}

# Run main function
main
