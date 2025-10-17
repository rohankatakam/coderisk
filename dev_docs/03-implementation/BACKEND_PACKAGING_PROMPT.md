# Claude Code Prompt: Backend Packaging & Distribution

**Session Type:** Backend (coderisk-go repository)
**Estimated Time:** 2-3 hours
**Phase:** Pre-Launch Setup
**Reference Docs:** [packaging_and_distribution.md](packaging_and_distribution.md), [open_core_strategy.md](../00-product/open_core_strategy.md)

---

## Context

You are setting up automated packaging and distribution for the CodeRisk CLI tool. CodeRisk uses an **open core model**: the CLI and local mode are open source (MIT License), while the cloud platform is proprietary.

Read these documents first for complete context:
- [packaging_and_distribution.md](packaging_and_distribution.md) - Complete packaging strategy
- [open_core_strategy.md](../00-product/open_core_strategy.md) - Open source positioning
- [LICENSE](../../LICENSE) - MIT License scope

---

## Objective

Set up automated multi-platform releases using GoReleaser, create a Homebrew formula for easy installation, and implement an install script for universal setup.

---

## Tasks

### 1. GoReleaser Configuration

**Create `.goreleaser.yml` in repository root:**

Requirements:
- Multi-platform builds: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64
- Binary name: `crisk`
- Version injection via ldflags (Version, Commit, BuildDate)
- Archive format: tar.gz (Unix), zip (Windows)
- Naming: `crisk_{{ .Os }}_{{ .Arch }}.tar.gz`
- Generate SHA256 checksums
- Auto-update Homebrew formula
- Generate changelog from commit messages (Conventional Commits)
- Docker image build (alpine base, tag: latest + version)

Refer to GoReleaser documentation for best practices. See packaging_and_distribution.md Section "GoReleaser Configuration" for detailed requirements.

### 2. Version Information

**Update `cmd/crisk/main.go` to support version injection:**

Add variables that GoReleaser can inject at build time:
```go
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildDate = "unknown"
)
```

Add `--version` flag that outputs:
```
CodeRisk v1.0.0
Build: abc1234
Date: 2025-10-13T12:34:56Z
```

### 3. Homebrew Formula Repository

**Create new GitHub repository: `homebrew-coderisk`**

Structure:
```
homebrew-coderisk/
├── Formula/
│   └── crisk.rb
├── README.md
└── LICENSE (MIT)
```

Formula (`crisk.rb`) should:
- Point to GitHub Releases tarball
- Include SHA256 checksum (auto-updated by GoReleaser)
- Test command: `crisk --version`
- Add post-install caveats if needed

README should explain:
```bash
brew tap rohankatakam/coderisk
brew install crisk
```

Configure GoReleaser to auto-update this formula on release.

### 4. Install Script

**Create `install.sh` for universal installation:**

Script should:
1. Detect platform (macOS, Linux, Windows/WSL) and architecture (x86_64, arm64)
2. Fetch latest release from GitHub API
3. Download correct tarball for platform
4. Verify SHA256 checksum
5. Extract binary to `~/.local/bin/crisk` (or `/usr/local/bin` with sudo)
6. Make executable (`chmod +x`)
7. Check if directory is in PATH, provide instructions if not
8. Detect shell (bash, zsh, fish) and offer to add to config
9. Run `crisk --version` to verify
10. Display success message and next steps

Error handling:
- Unsupported platform → show manual download link
- Download failure → retry with backoff
- Checksum mismatch → abort with error
- Handle idempotency (safe to re-run)

This script will be hosted at `https://coderisk.dev/install.sh` (frontend will serve it).

### 5. GitHub Actions Workflow

**Create `.github/workflows/release.yml`:**

Trigger: On push of tags matching `v*` (e.g., `v1.0.0`)

Jobs:
1. Run tests (`go test ./...`)
2. Run GoReleaser
3. Upload artifacts to GitHub Releases
4. Update Homebrew formula
5. Build and push Docker image to Docker Hub

Secrets needed:
- `GITHUB_TOKEN` (auto-provided)
- `DOCKER_HUB_USERNAME` and `DOCKER_HUB_TOKEN` (add to repo secrets)

See packaging_and_distribution.md Section "Release Process" for workflow details.

### 6. Dockerfile

**Create `Dockerfile` in repository root:**

Requirements:
- Base image: `alpine:latest`
- Install: `ca-certificates`, `git`
- Copy binary: `COPY crisk /usr/local/bin/crisk`
- Entrypoint: `ENTRYPOINT ["crisk"]`
- Keep image size <20MB

Test usage:
```bash
docker build -t coderisk/crisk:test .
docker run --rm -v $(pwd):/repo coderisk/crisk:test --version
```

### 7. Documentation Updates

**Update `README.md`:**

Add installation section with all methods:
1. Homebrew (recommended for macOS/Linux)
2. Install script (universal one-liner)
3. Docker (for CI/CD)
4. Direct download (GitHub Releases)

Show example for each method.

**Create `CHANGELOG.md`:**

Use Keep a Changelog format. GoReleaser will auto-generate entries, but you should create the initial structure.

### 8. Testing

**Test the complete flow:**

1. Create a test tag: `v0.1.0-beta.1`
2. Push tag to trigger GitHub Actions
3. Verify GitHub Release created with:
   - Binaries for all platforms
   - Checksums file
   - Changelog
4. Test Homebrew install (if formula updated)
5. Test install script manually
6. Test Docker image pull

If beta testing succeeds, create official `v1.0.0` tag.

---

## Success Criteria

- [ ] GoReleaser builds binaries for 5 platforms
- [ ] GitHub Release created automatically on tag push
- [ ] Homebrew formula auto-updates on release
- [ ] Install script works on macOS, Linux
- [ ] Docker image builds and runs correctly
- [ ] `crisk --version` shows correct version info
- [ ] README documents all installation methods
- [ ] CHANGELOG follows Keep a Changelog format

---

## Key Files to Create/Modify

**New files:**
- `.goreleaser.yml` (GoReleaser config)
- `.github/workflows/release.yml` (GitHub Actions)
- `Dockerfile` (Docker image)
- `install.sh` (install script)
- `CHANGELOG.md` (changelog template)

**Modified files:**
- `cmd/crisk/main.go` (add version variables, --version flag)
- `README.md` (add installation section)

**External:**
- Create `homebrew-coderisk` repository on GitHub
- Configure Docker Hub repository `coderisk/crisk`

---

## Notes

- Use Conventional Commits for commit messages (feat:, fix:, docs:, etc.) - GoReleaser uses these for changelog
- Test with beta tag first before `v1.0.0`
- Ensure Docker Hub credentials are in GitHub Secrets
- Homebrew formula will be auto-updated, but verify first release manually
- **Phase 0 works without API key** - Emphasize this in docs!
- **Phase 2 requires OPENAI_API_KEY** - Optional but recommended
- **Docker optional** - Only needed for full graph analysis (Phase 1)
- Install script should be copied to frontend repo's `public/` directory after creation
- Keep install script POSIX-compliant (works on sh, bash, zsh)

---

## Reference Documentation

**Required Reading:**
- [packaging_and_distribution.md](packaging_and_distribution.md) - Complete strategy
- [LICENSE](../../LICENSE) - What's open source vs proprietary

**External References:**
- [GoReleaser Documentation](https://goreleaser.com/quick-start/)
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Keep a Changelog](https://keepachangelog.com/)
- [Semantic Versioning](https://semver.org/)

---

## Questions to Ask if Unclear

1. What should the initial version number be? (Suggest v0.1.0 for beta or v1.0.0 for stable)
2. Should we include release notes in addition to auto-generated changelog?
3. Any specific caveats to include in Homebrew formula post-install?
4. Which Docker Hub account to use? (Need credentials in GitHub Secrets)

---

**Good luck! This will make CodeRisk installable with one command: `brew install crisk`**
