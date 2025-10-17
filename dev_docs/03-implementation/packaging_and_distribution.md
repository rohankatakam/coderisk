# Packaging and Distribution Strategy

**Last Updated:** 2025-10-13
**Owner:** Engineering Team
**Status:** Active - Ready for Implementation
**Phase:** Pre-Launch Setup

> **📘 Cross-reference:** See [open_core_strategy.md](../00-product/open_core_strategy.md) for strategic rationale, [LICENSE](../../LICENSE) for scope, and [CONTRIBUTING.md](../../CONTRIBUTING.md) for community guidelines.

---

## Executive Summary

This document outlines the packaging and distribution strategy for CodeRisk CLI as an open source tool. We will use **GoReleaser** for automated multi-platform builds and **Homebrew** for macOS distribution, with additional support for direct downloads and Docker.

**Goals:**
1. ✅ One-command install via Homebrew (`brew install crisk`)
2. ✅ Multi-platform binaries (macOS Intel/ARM, Linux, Windows)
3. ✅ Automated releases via GitHub Actions
4. ✅ Version management and changelog automation
5. ✅ 17-minute setup time (one-time per repo: install + API key + Docker + init)

**Success Criteria:**
- CLI install time: <30 seconds (Homebrew)
- Full setup time: ~17 minutes (one-time per repo)
- Platform coverage: macOS, Linux, Windows
- Release cadence: Weekly patches, monthly features
- Community adoption: 1K installs in first month

---

## Distribution Channels

### 1. Homebrew (Primary - macOS/Linux)

**Target:** macOS and Linux developers (80% of audience)

**User Experience:**
```bash
# Tap our formula
brew tap rohankatakam/coderisk

# Install
brew install crisk

# Verify
crisk --version
# CodeRisk v1.0.0
```

**Why Homebrew:**
- ✅ Standard for developer tools on macOS
- ✅ Automatic updates via `brew upgrade`
- ✅ Binary distribution (no compilation needed)
- ✅ Dependency management built-in
- ✅ Trusted by developers

**Implementation:**
- Create `homebrew-coderisk` repository
- Formula points to GitHub Releases
- GoReleaser auto-updates formula on release

### 2. GitHub Releases (Universal)

**Target:** All platforms, manual installers

**Assets per Release:**
- `crisk_Darwin_arm64.tar.gz` (macOS Apple Silicon)
- `crisk_Darwin_x86_64.tar.gz` (macOS Intel)
- `crisk_Linux_arm64.tar.gz` (Linux ARM)
- `crisk_Linux_x86_64.tar.gz` (Linux x64)
- `crisk_Windows_x86_64.zip` (Windows)
- `checksums.txt` (SHA256 verification)

**Why GitHub Releases:**
- ✅ Version history and changelog
- ✅ Direct download links
- ✅ CDN distribution (fast global access)
- ✅ Required for Homebrew formula

### 3. Install Script (One-Liner)

**Target:** Quick setup for all platforms

**User Experience:**
```bash
# macOS/Linux
curl -fsSL https://coderisk.dev/install.sh | bash

# Output:
# ✅ Detected: macOS arm64
# 📦 Downloading crisk v1.0.0...
# ✅ Installed to ~/.local/bin/crisk
# 🎉 Run 'crisk --version' to verify
```

**Script Features:**
- Auto-detect OS and architecture
- Download correct binary from GitHub Releases
- Install to `~/.local/bin` (or system path)
- Add to PATH if needed
- Verify installation

**Why Install Script:**
- ✅ Works on any Unix-like system
- ✅ No package manager required
- ✅ Good for CI/CD environments
- ✅ Single command setup

### 4. Docker Hub (Containerized)

**Target:** CI/CD pipelines, containerized environments

**User Experience:**
```bash
# Pull image
docker pull coderisk/crisk:latest

# Run check
docker run --rm -v $(pwd):/repo coderisk/crisk:latest check
```

**Why Docker:**
- ✅ Consistent environment (includes dependencies)
- ✅ Good for CI/CD integration
- ✅ No local installation needed
- ✅ Version pinning (`crisk:1.0.0`)

### 5. Direct Binary Download

**Target:** Advanced users, air-gapped environments

**User Experience:**
1. Visit https://github.com/rohankatakam/coderisk-go/releases
2. Download binary for your platform
3. Extract and move to PATH
4. Run `crisk --version`

**Why Direct Download:**
- ✅ Works in restricted environments
- ✅ No internet-dependent tools (brew, curl)
- ✅ Manual verification via checksums

---

## GoReleaser Configuration

### Overview

**GoReleaser** automates:
- Multi-platform builds (cross-compilation)
- GitHub Releases creation
- Homebrew formula updates
- Docker image builds
- Changelog generation
- Checksum creation

### Configuration File Structure

**Location:** `.goreleaser.yml` (repository root)

**Key Sections:**

**1. Builds**
- Platforms: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64
- Binary name: `crisk`
- Flags: `-trimpath` (reproducible builds)
- Ldflags: Inject version, commit, date

**2. Archives**
- Format: `tar.gz` (Unix), `zip` (Windows)
- Naming: `crisk_{{ .Os }}_{{ .Arch }}.tar.gz`

**3. Checksums**
- Algorithm: SHA256
- File: `checksums.txt`

**4. Homebrew**
- Repository: `rohankatakam/homebrew-coderisk`
- Formula: `crisk.rb`
- Auto-update on release

**5. Docker**
- Image: `coderisk/crisk`
- Tags: `latest`, `v1.0.0`, `1.0`, `1`
- Base: `alpine:latest` (minimal size)

**6. Changelog**
- Format: Keep a Changelog style
- Groups: Features, Fixes, Docs, Chores
- Generated from commit messages (Conventional Commits)

### Version Injection

**Build-time variables:**
```go
// Injected via ldflags
var (
    Version   = "dev"       // Git tag (e.g., "v1.0.0")
    Commit    = "unknown"   // Git SHA (e.g., "abc1234")
    BuildDate = "unknown"   // ISO 8601 timestamp
)
```

**CLI output:**
```bash
$ crisk --version
CodeRisk v1.0.0
Build: abc1234
Date: 2025-10-13T12:34:56Z
```

---

## Homebrew Formula

### Repository Structure

**Repository:** `rohankatakam/homebrew-coderisk` (public)

```
homebrew-coderisk/
├── Formula/
│   └── crisk.rb          # Homebrew formula
├── README.md             # Tap instructions
└── LICENSE               # MIT License
```

### Formula Template

**File:** `Formula/crisk.rb`

**Key Elements:**
- Description and homepage
- URL to GitHub Release tarball
- SHA256 checksum (auto-updated by GoReleaser)
- Installation instructions
- Test command (`crisk --version`)
- Caveats (post-install message)

**Dependencies:**
- None (static binary)

**Conflicts:**
- None

### Installation Flow

**User runs:**
```bash
brew tap rohankatakam/coderisk
brew install crisk
```

**Homebrew:**
1. Downloads formula from GitHub
2. Downloads binary from GitHub Releases
3. Verifies SHA256 checksum
4. Extracts to Cellar
5. Creates symlink to `/usr/local/bin/crisk`

**Result:** `crisk` available globally

---

## Release Process

### Automated Workflow (Recommended)

**Trigger:** Push Git tag (e.g., `v1.0.0`)

**GitHub Actions Workflow:**
```yaml
name: Release

on:
  push:
    tags:
      - 'v*'  # Trigger on v1.0.0, v1.1.0, etc.

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - Checkout code
      - Setup Go 1.21+
      - Run GoReleaser
      - Upload artifacts to GitHub Releases
      - Update Homebrew formula
      - Build and push Docker image
```

**Steps:**
1. Developer creates tag: `git tag -a v1.0.0 -m "Release v1.0.0"`
2. Push tag: `git push origin v1.0.0`
3. GitHub Actions triggers
4. GoReleaser builds binaries
5. GitHub Release created
6. Homebrew formula updated
7. Docker image pushed

**Duration:** 5-10 minutes

### Manual Release (Fallback)

**If GitHub Actions unavailable:**

```bash
# Export GitHub token
export GITHUB_TOKEN="ghp_..."

# Tag release
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# Run GoReleaser locally
goreleaser release --clean
```

### Versioning Strategy

**Semantic Versioning (SemVer):**
- Major: Breaking changes (v2.0.0)
- Minor: New features (v1.1.0)
- Patch: Bug fixes (v1.0.1)

**Tagging:**
- Format: `vMAJOR.MINOR.PATCH`
- Examples: `v1.0.0`, `v1.2.3`, `v2.0.0-beta.1`

**Changelog:**
- Auto-generated from commits
- Follows [Keep a Changelog](https://keepachangelog.com/)
- Groups: Added, Changed, Fixed, Deprecated, Removed, Security

### Pre-Release Testing

**Before tagging:**
1. ✅ All tests pass: `go test ./...`
2. ✅ Build succeeds: `go build ./cmd/crisk`
3. ✅ Integration tests pass: `./test/integration/`
4. ✅ Manual smoke test: Install locally and run `crisk check`
5. ✅ Changelog reviewed: Update `CHANGELOG.md`

**Beta releases:**
- Tag: `v1.0.0-beta.1`
- Formula: `crisk@beta` (separate)
- Purpose: Early testing before stable

---

## Install Script Implementation

### Script Location

**URL:** `https://coderisk.dev/install.sh`
**Repository:** Frontend repo (`public/install.sh`)

### Script Functionality

**1. Platform Detection**
- Detect OS: macOS, Linux, Windows (WSL)
- Detect Architecture: x86_64, arm64
- Validate compatibility

**2. Binary Download**
- Fetch latest release from GitHub API
- Download correct tarball
- Verify SHA256 checksum
- Extract binary

**3. Installation**
- Install to `~/.local/bin/crisk` (default)
- Or `/usr/local/bin/crisk` (with sudo)
- Make executable: `chmod +x`

**4. PATH Setup**
- Check if `~/.local/bin` in PATH
- If not, provide instructions to add
- Detect shell: bash, zsh, fish
- Offer to auto-add to shell config

**5. Verification**
- Run `crisk --version`
- Display success message
- Show next steps

### Error Handling

**Common scenarios:**
- Unsupported platform → Show manual download link
- Download failure → Retry with exponential backoff
- Checksum mismatch → Abort with error message
- PATH not set → Provide clear instructions

### Security Considerations

- ✅ Use HTTPS for all downloads
- ✅ Verify checksums before execution
- ✅ Never execute downloaded script directly (user pipes to bash)
- ✅ Idempotent (safe to re-run)
- ✅ No root required (installs to user directory)

---

## Docker Image

### Dockerfile

**Location:** `Dockerfile` (repository root)

**Base Image:** `alpine:latest` (minimal size)

**Layers:**
1. Install dependencies: `ca-certificates`, `git`
2. Copy binary: `COPY crisk /usr/local/bin/crisk`
3. Set entrypoint: `ENTRYPOINT ["crisk"]`

**Size:** <20MB (binary is statically compiled)

### Docker Hub

**Repository:** `coderisk/crisk`

**Tags:**
- `latest` → Always points to newest release
- `v1.0.0` → Specific version
- `1.0` → Minor version (updates with patches)
- `1` → Major version

**Automated Builds:**
- Trigger: GoReleaser on release
- Push to Docker Hub via GitHub Actions
- Multi-platform: `linux/amd64`, `linux/arm64`

### Usage Examples

**Basic check:**
```bash
docker run --rm -v $(pwd):/repo coderisk/crisk:latest check
```

**With environment variables:**
```bash
docker run --rm \
  -v $(pwd):/repo \
  -e OPENAI_API_KEY="sk-..." \
  coderisk/crisk:latest check --explain
```

**CI/CD integration (GitHub Actions):**
```yaml
- name: Run CodeRisk
  uses: docker://coderisk/crisk:latest
  with:
    args: check
```

---

## Platform-Specific Considerations

### macOS

**Installation:**
- Homebrew (recommended)
- Install script
- Direct download

**Gatekeeper:**
- Binaries must be signed (future: code signing)
- Current: Users may see "unidentified developer" warning
- Workaround: `xattr -d com.apple.quarantine crisk`

**PATH:**
- Homebrew: Automatic (`/usr/local/bin` or `/opt/homebrew/bin`)
- Install script: `~/.local/bin` (must be in PATH)

### Linux

**Installation:**
- Install script (recommended)
- Homebrew (if installed)
- Direct download
- Package managers (future: apt, yum)

**Dependencies:**
- None (static binary)
- For local mode: Docker, Docker Compose

**PATH:**
- Standard: `/usr/local/bin` (requires sudo)
- User: `~/.local/bin` (no sudo)

### Windows

**Installation:**
- Direct download (recommended)
- WSL + install script (for WSL users)

**Path:**
- Add to PATH manually
- Or use WSL for Unix-like experience

**Limitations:**
- Windows-native: Basic CLI only
- WSL recommended for full experience (Docker support)

---

## Binary Signing (Future Enhancement)

### Current State

**Status:** Unsigned binaries
**Impact:** macOS Gatekeeper warnings, SmartScreen on Windows

### Future: Code Signing

**macOS:**
- Apple Developer account required ($99/year)
- Sign with Developer ID certificate
- Notarize with Apple (automated)
- Result: No Gatekeeper warnings

**Windows:**
- EV Code Signing certificate required
- Sign with `signtool.exe`
- Result: No SmartScreen warnings

**Timeline:** After initial launch (Month 3-6)

---

## Distribution Metrics

### Track per Release

**Installation metrics:**
- Total installs (Homebrew analytics)
- Installs by platform (macOS, Linux, Windows)
- Installs by architecture (x86_64, arm64)
- Docker pulls

**Download metrics:**
- GitHub Release downloads
- Install script executions
- Direct binary downloads

**Engagement metrics:**
- Homebrew formula updates (`brew upgrade crisk`)
- Version distribution (how many on latest?)
- Uninstalls (Homebrew analytics)

**Target Metrics (Month 1):**
- 1,000 total installs
- 70% macOS, 25% Linux, 5% Windows
- 80% on latest version within 1 week of release

---

## Documentation Requirements

### Installation Docs

**Location:** `README.md` and website

**Sections:**
1. **Quick Install** (Homebrew one-liner)
2. **Alternative Methods** (install script, Docker, direct download)
3. **Platform-Specific Notes** (macOS, Linux, Windows)
4. **Troubleshooting** (common errors, PATH issues)
5. **Uninstallation** (how to remove)

### Changelog

**Location:** `CHANGELOG.md` (repository root)

**Format:** [Keep a Changelog](https://keepachangelog.com/)

**Auto-generated:** GoReleaser creates from commit messages

**Manual additions:** Release notes, breaking changes

### Release Notes

**Location:** GitHub Releases page

**Template:**
```markdown
## CodeRisk v1.0.0

### 🎉 Highlights
- New feature X
- Performance improvement Y

### ✨ Added
- Feature A (#123)
- Feature B (#456)

### 🐛 Fixed
- Bug fix C (#789)
- Bug fix D (#101)

### 📚 Documentation
- Updated installation guide

### Installation
\`\`\`bash
brew tap rohankatakam/coderisk
brew install crisk
\`\`\`
```

---

## Implementation Checklist

### Phase 1: GoReleaser Setup (Week 1)

- [ ] Create `.goreleaser.yml` configuration
- [ ] Configure multi-platform builds
- [ ] Set up version injection (ldflags)
- [ ] Test local release: `goreleaser release --snapshot --clean`
- [ ] Verify binary artifacts

### Phase 2: Homebrew Formula (Week 1)

- [ ] Create `homebrew-coderisk` repository
- [ ] Create initial `Formula/crisk.rb`
- [ ] Test tap: `brew tap rohankatakam/coderisk`
- [ ] Test install: `brew install crisk`
- [ ] Configure GoReleaser to auto-update formula

### Phase 3: GitHub Actions (Week 1)

- [ ] Create `.github/workflows/release.yml`
- [ ] Configure secrets (GITHUB_TOKEN, DOCKER_HUB_TOKEN)
- [ ] Test release workflow with beta tag
- [ ] Verify GitHub Release creation
- [ ] Verify Homebrew formula update

### Phase 4: Install Script (Week 2)

- [ ] Write `install.sh` script
- [ ] Add to frontend repository (`public/install.sh`)
- [ ] Host at `https://coderisk.dev/install.sh`
- [ ] Test on macOS (Intel, ARM)
- [ ] Test on Linux (Ubuntu, Debian, Arch)

### Phase 5: Docker (Week 2)

- [ ] Create `Dockerfile`
- [ ] Configure GoReleaser Docker builds
- [ ] Set up Docker Hub repository
- [ ] Test image: `docker pull coderisk/crisk:latest`
- [ ] Add Docker usage docs

### Phase 6: Documentation (Week 2)

- [ ] Update `README.md` with install instructions
- [ ] Create `CHANGELOG.md` template
- [ ] Add troubleshooting guide
- [ ] Update website with install commands
- [ ] Add badges (GitHub, Homebrew, Docker)

### Phase 7: First Release (Week 2)

- [ ] Tag `v1.0.0` (or `v0.1.0` for beta)
- [ ] Verify release workflow
- [ ] Test all installation methods
- [ ] Monitor GitHub Release downloads
- [ ] Announce on social media

---

## Troubleshooting Guide

### Common Issues

**1. "command not found: crisk"**
- **Cause:** Binary not in PATH
- **Fix:** Add `~/.local/bin` to PATH or reinstall with Homebrew

**2. "permission denied"**
- **Cause:** Binary not executable
- **Fix:** `chmod +x ~/.local/bin/crisk`

**3. macOS Gatekeeper warning**
- **Cause:** Unsigned binary
- **Fix:** `xattr -d com.apple.quarantine $(which crisk)`

**4. Homebrew install fails**
- **Cause:** Tap not added or formula outdated
- **Fix:** `brew update && brew tap rohankatakam/coderisk`

**5. Docker image not found**
- **Cause:** Image not pulled or wrong tag
- **Fix:** `docker pull coderisk/crisk:latest`

---

## Related Documents

**Product & Strategy:**
- [open_core_strategy.md](../00-product/open_core_strategy.md) - Open source licensing and distribution strategy
- [go_to_market.md](../00-product/go_to_market.md) - Launch strategy (to be updated)

**Technical:**
- [LICENSE](../../LICENSE) - MIT License with scope
- [CONTRIBUTING.md](../../CONTRIBUTING.md) - Contribution guidelines
- [README.md](../../README.md) - Installation instructions

**Implementation:**
- `.goreleaser.yml` - GoReleaser configuration (to be created)
- `.github/workflows/release.yml` - GitHub Actions workflow (to be created)

---

**Last Updated:** 2025-10-13
**Next Review:** After first release (post-mortem)
**Next Steps:** Implement Phase 1 (GoReleaser setup)
