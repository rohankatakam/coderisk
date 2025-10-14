# Homebrew Tap Setup Instructions

This guide explains how to set up the Homebrew tap for CodeRisk CLI.

## Prerequisites

- GitHub account: `rohankatakam`
- GoReleaser configured (already done in `.goreleaser.yml`)
- GitHub token with `repo` and `workflow` permissions

## Step 1: Create Homebrew Tap Repository

1. Go to https://github.com/new
2. Create a new **public** repository named: `homebrew-coderisk`
3. Add description: "Homebrew tap for CodeRisk CLI"
4. Initialize with README
5. Choose MIT License

## Step 2: Set Up Repository Structure

```bash
# Clone the new repository
git clone https://github.com/rohankatakam/homebrew-coderisk.git
cd homebrew-coderisk

# Create Formula directory
mkdir -p Formula

# Copy the template formula
cp /path/to/coderisk-go/packaging/crisk.rb Formula/crisk.rb

# Commit and push
git add Formula/crisk.rb
git commit -m "feat: Add initial Homebrew formula for crisk"
git push origin main
```

## Step 3: Verify GoReleaser Configuration

The `.goreleaser.yml` file in the `coderisk-go` repository already includes the Homebrew configuration:

```yaml
brews:
  - repository:
      owner: rohankatakam
      name: homebrew-coderisk
      token: "{{ .Env.GITHUB_TOKEN }}"
    directory: Formula
    homepage: https://coderisk.dev
    description: "Lightning-fast AI-powered code risk assessment"
    license: MIT
    # ... (rest of config)
```

This means GoReleaser will automatically:
- Update the formula when you create a new release
- Update version numbers
- Update download URLs
- Update SHA256 checksums

## Step 4: Configure GitHub Secrets

The release workflow needs access to update the Homebrew tap.

**The `GITHUB_TOKEN` secret is automatically provided by GitHub Actions** with permission to:
- Create releases in `coderisk-go` repository
- Push to `homebrew-coderisk` repository (if public)

**Important:** Make sure the `homebrew-coderisk` repository is **public**. GitHub's automatic `GITHUB_TOKEN` can only push to public repositories in the same organization/account.

If you need to use a custom token:

1. Go to https://github.com/settings/tokens
2. Create a new token (classic) with scopes:
   - `repo` (full control of private repositories)
   - `workflow` (update GitHub Actions workflows)
3. Add the token to `coderisk-go` repository secrets:
   - Go to https://github.com/rohankatakam/coderisk-go/settings/secrets/actions
   - Click "New repository secret"
   - Name: `HOMEBREW_TAP_TOKEN`
   - Value: Your token
4. Update `.goreleaser.yml`:
   ```yaml
   brews:
     - repository:
         owner: rohankatakam
         name: homebrew-coderisk
         token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
   ```

## Step 5: Test the Setup

### Option A: Test with a Pre-Release

1. Create a pre-release tag:
   ```bash
   cd coderisk-go
   git tag -a v0.1.0-beta.1 -m "Beta release for testing"
   git push origin v0.1.0-beta.1
   ```

2. Watch GitHub Actions run: https://github.com/rohankatakam/coderisk-go/actions

3. Verify the formula was updated:
   ```bash
   # Check the homebrew-coderisk repository
   curl https://raw.githubusercontent.com/rohankatakam/homebrew-coderisk/main/Formula/crisk.rb
   ```

4. Test installation:
   ```bash
   brew tap rohankatakam/coderisk
   brew install crisk
   crisk --version
   ```

### Option B: Test Locally (Without Creating Release)

```bash
cd coderisk-go

# Install GoReleaser if not already installed
brew install goreleaser

# Run GoReleaser in snapshot mode (no Git tags needed)
goreleaser release --snapshot --clean

# Check the dist/ directory for built binaries
ls -la dist/
```

## Step 6: Create Your First Official Release

Once testing is successful:

```bash
cd coderisk-go

# Create the official v1.0.0 tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

This will:
1. Trigger the GitHub Actions release workflow
2. Build binaries for all platforms
3. Create a GitHub Release with changelog
4. Update the Homebrew formula automatically
5. Build and push Docker images

## Step 7: Announce the Release

After successful release:

1. **Update coderisk.dev website** with installation instructions
2. **Copy install.sh to frontend** repository's `public/` directory
3. **Test installation** from a fresh machine:
   ```bash
   brew tap rohankatakam/coderisk
   brew install crisk
   ```
4. **Announce on social media** (Twitter, LinkedIn, etc.)

## Troubleshooting

### Formula not updating

- Check GitHub Actions logs for errors
- Verify `GITHUB_TOKEN` has correct permissions
- Ensure `homebrew-coderisk` repository is public

### Homebrew installation fails

- Verify checksums match in formula
- Check that release artifacts exist on GitHub
- Try `brew update` and retry

### "Permission denied" errors

- Ensure the token has `repo` scope
- Check repository visibility settings

## Maintenance

### Updating the formula manually

If you need to update the formula manually (not recommended):

```bash
cd homebrew-coderisk
# Edit Formula/crisk.rb
git add Formula/crisk.rb
git commit -m "fix: Update formula"
git push origin main
```

### Deprecating old versions

Homebrew automatically handles version updates. Users can upgrade with:

```bash
brew update
brew upgrade crisk
```

## Resources

- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [GoReleaser Homebrew Documentation](https://goreleaser.com/customization/homebrew/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)

---

**Next Steps:**
1. ✅ Create `homebrew-coderisk` repository
2. ✅ Copy formula to `Formula/crisk.rb`
3. ✅ Test with beta release (`v0.1.0-beta.1`)
4. ✅ Create official release (`v1.0.0`)
5. ✅ Update website with installation instructions
6. ✅ Announce to community
