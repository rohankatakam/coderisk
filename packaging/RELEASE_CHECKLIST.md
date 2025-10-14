# Release Checklist for CodeRisk CLI

This checklist ensures a smooth release process for CodeRisk.

## Pre-Release Setup (One-Time)

### GitHub Repository Setup
- [ ] Repository `coderisk-go` exists and is public
- [ ] Main branch is `main` (or update workflow accordingly)
- [ ] Repository has description: "Lightning-fast AI-powered code risk assessment"
- [ ] Topics added: `golang`, `cli`, `code-quality`, `security`, `ai`, `llm`

### Homebrew Tap Setup
- [ ] Repository `homebrew-coderisk` created (public)
- [ ] Formula directory created: `Formula/`
- [ ] Initial formula committed: `Formula/crisk.rb`
- [ ] Repository description: "Homebrew tap for CodeRisk CLI"

### Docker Hub Setup
- [ ] Docker Hub account created
- [ ] Repository `coderisk/crisk` created (public)
- [ ] Access token generated with Read & Write permissions
- [ ] Token added to GitHub Secrets as `DOCKER_HUB_TOKEN`
- [ ] Username added to GitHub Secrets as `DOCKER_HUB_USERNAME`

### GitHub Secrets Configured
- [ ] `GITHUB_TOKEN` - Auto-provided by GitHub ✅
- [ ] `DOCKER_HUB_USERNAME` - Your Docker Hub username
- [ ] `DOCKER_HUB_TOKEN` - Docker Hub access token

### Files Created
- [ ] `.goreleaser.yml` - GoReleaser configuration
- [ ] `.github/workflows/release.yml` - Release workflow
- [ ] `Dockerfile` - Docker image configuration
- [ ] `install.sh` - Universal installer script
- [ ] `CHANGELOG.md` - Changelog template
- [ ] `packaging/crisk.rb` - Homebrew formula template
- [ ] `packaging/HOMEBREW_SETUP.md` - Setup instructions
- [ ] `packaging/DOCKER_HUB_SETUP.md` - Docker setup guide

## Pre-Release Testing

### Code Quality
- [ ] All tests pass: `go test ./...`
- [ ] Code formatted: `go fmt ./...`
- [ ] No linter warnings: `golangci-lint run`
- [ ] Integration tests pass: `./test/integration/test_check_e2e.sh`

### Version Information
- [ ] Version variables defined in `cmd/crisk/main.go`
- [ ] Version template configured in Cobra
- [ ] `crisk --version` displays correctly

### Local Build Test
- [ ] Build succeeds: `go build -o crisk ./cmd/crisk`
- [ ] Binary runs: `./crisk --version`
- [ ] Binary size reasonable (<50MB uncompressed)

### Docker Test
- [ ] Dockerfile builds: `docker build -t coderisk/crisk:test .`
- [ ] Image runs: `docker run --rm coderisk/crisk:test --version`
- [ ] Image size acceptable (<20MB)

## Beta Release (v0.1.0-beta.1)

### Create Beta Tag
```bash
git tag -a v0.1.0-beta.1 -m "Beta release for testing packaging"
git push origin v0.1.0-beta.1
```

### Verify GitHub Actions
- [ ] Workflow triggered: Check https://github.com/rohankatakam/coderisk-go/actions
- [ ] Tests pass
- [ ] GoReleaser runs successfully
- [ ] Artifacts uploaded

### Verify GitHub Release
- [ ] Release created: https://github.com/rohankatakam/coderisk-go/releases
- [ ] Changelog generated
- [ ] Binaries present for all platforms:
  - [ ] `crisk_darwin_arm64.tar.gz`
  - [ ] `crisk_darwin_x86_64.tar.gz`
  - [ ] `crisk_linux_arm64.tar.gz`
  - [ ] `crisk_linux_x86_64.tar.gz`
  - [ ] `crisk_windows_x86_64.zip`
- [ ] `checksums.txt` present

### Verify Homebrew Formula
- [ ] Formula updated in `homebrew-coderisk` repository
- [ ] Version matches release tag
- [ ] URLs point to correct release
- [ ] Checksums updated

### Verify Docker Images
- [ ] Images on Docker Hub: https://hub.docker.com/r/coderisk/crisk/tags
- [ ] Tags present:
  - [ ] `v0.1.0-beta.1`
  - [ ] `v0.1.0-beta.1-arm64`
  - [ ] `latest` (if this is the latest release)

### Test Installations

**Homebrew (macOS):**
```bash
brew tap rohankatakam/coderisk
brew install crisk
crisk --version
```

**Install Script:**
```bash
# Test on clean machine or container
curl -fsSL https://coderisk.dev/install.sh | bash
# Or from GitHub:
curl -fsSL https://raw.githubusercontent.com/rohankatakam/coderisk-go/main/install.sh | bash
```

**Docker:**
```bash
docker pull coderisk/crisk:v0.1.0-beta.1
docker run --rm coderisk/crisk:v0.1.0-beta.1 --version
```

**Direct Download:**
```bash
# Test manual download and extraction
wget https://github.com/rohankatakam/coderisk-go/releases/download/v0.1.0-beta.1/crisk_darwin_arm64.tar.gz
tar -xzf crisk_darwin_arm64.tar.gz
./crisk --version
```

### Issues Found?
- [ ] Document issues in GitHub Issues
- [ ] Fix and create new beta tag: `v0.1.0-beta.2`
- [ ] Repeat testing

## Official Release (v1.0.0)

### Pre-Release Checks
- [ ] Beta testing complete
- [ ] All critical bugs fixed
- [ ] Documentation updated
- [ ] README installation instructions verified
- [ ] CHANGELOG updated with release notes

### Update Version References
- [ ] `CHANGELOG.md` - Update "Unreleased" to version
- [ ] Any hardcoded version strings updated

### Create Release Tag
```bash
git tag -a v1.0.0 -m "Release v1.0.0 - Initial stable release"
git push origin v1.0.0
```

### Monitor Release Process
- [ ] GitHub Actions workflow starts
- [ ] All jobs complete successfully
- [ ] No errors in logs

### Verify Release
- [ ] GitHub Release created with correct content
- [ ] All binaries present and correct sizes
- [ ] Homebrew formula updated
- [ ] Docker images published

### Test Official Release
- [ ] Homebrew install works
- [ ] Install script works
- [ ] Docker images work
- [ ] Version command shows `v1.0.0`

### Post-Release Tasks
- [ ] Copy `install.sh` to frontend repository's `public/` directory
- [ ] Update coderisk.dev website with:
  - [ ] Installation instructions
  - [ ] Link to GitHub releases
  - [ ] Link to Docker Hub
- [ ] Update documentation site (if separate)
- [ ] Create release announcement blog post

### Community Announcement
- [ ] Post on Twitter/X
- [ ] Post on LinkedIn
- [ ] Post on Reddit (r/golang, r/programming)
- [ ] Post on Hacker News (Show HN)
- [ ] Post in relevant Discord/Slack communities
- [ ] Send email to mailing list (if applicable)

### Monitoring
- [ ] Watch GitHub Issues for installation problems
- [ ] Monitor Homebrew analytics (if available)
- [ ] Track Docker Hub pull counts
- [ ] Monitor GitHub stars and forks

## Patch Release (v1.0.1)

For bug fixes:

```bash
# Fix bugs in code
# Update CHANGELOG.md with fixes

git tag -a v1.0.1 -m "Release v1.0.1 - Bug fixes"
git push origin v1.0.1
```

- [ ] Verify automated release
- [ ] Test updated installations
- [ ] Announce if critical fix

## Feature Release (v1.1.0)

For new features:

```bash
# Implement new features
# Update CHANGELOG.md with features

git tag -a v1.1.0 -m "Release v1.1.0 - New features"
git push origin v1.1.0
```

- [ ] Verify automated release
- [ ] Test new features work in installations
- [ ] Announce new features

## Troubleshooting

### Release workflow fails
1. Check GitHub Actions logs
2. Common issues:
   - Tests failing
   - Go build errors
   - Docker Hub authentication
   - GoReleaser configuration errors

### Homebrew formula not updating
1. Check GitHub token permissions
2. Verify `homebrew-coderisk` repository is public
3. Check GoReleaser logs in workflow

### Docker images not publishing
1. Verify Docker Hub secrets are correct
2. Check Docker login step in workflow
3. Verify Docker Hub repository exists and is public

## Resources

- [Semantic Versioning](https://semver.org/)
- [Keep a Changelog](https://keepachangelog.com/)
- [GoReleaser Documentation](https://goreleaser.com/)
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)

---

**Version:** 1.0
**Last Updated:** 2025-10-13
**Maintained by:** Engineering Team
