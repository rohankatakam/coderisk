# Docker Hub Setup Instructions

This guide explains how to set up Docker Hub for automated image publishing.

## Prerequisites

- Docker Hub account (free): https://hub.docker.com/signup
- GitHub repository configured with release workflow

## Step 1: Create Docker Hub Repository

1. Log in to https://hub.docker.com
2. Click "Create Repository"
3. Configure:
   - **Name:** `crisk`
   - **Namespace:** `coderisk` (or your username)
   - **Visibility:** Public
   - **Description:** "Lightning-fast AI-powered code risk assessment CLI"
   - **Full name:** `coderisk/crisk`
4. Click "Create"

**Result:** Repository URL will be `https://hub.docker.com/r/coderisk/crisk`

## Step 2: Create Access Token

1. Go to https://hub.docker.com/settings/security
2. Click "New Access Token"
3. Configure:
   - **Description:** `GitHub Actions - coderisk-go releases`
   - **Access permissions:** `Read & Write`
4. Click "Generate"
5. **IMPORTANT:** Copy the token immediately (you won't see it again!)

Example token format: `dckr_pat_abcdefghijklmnopqrstuvwxyz123456`

## Step 3: Add Secrets to GitHub Repository

1. Go to https://github.com/rohankatakam/coderisk-go/settings/secrets/actions
2. Click "New repository secret"

### Add DOCKER_HUB_USERNAME

- **Name:** `DOCKER_HUB_USERNAME`
- **Value:** Your Docker Hub username (e.g., `coderisk`)
- Click "Add secret"

### Add DOCKER_HUB_TOKEN

- **Name:** `DOCKER_HUB_TOKEN`
- **Value:** The access token you copied in Step 2
- Click "Add secret"

**Security Note:** These secrets are encrypted and only accessible to GitHub Actions workflows.

## Step 4: Verify GitHub Actions Configuration

The `.github/workflows/release.yml` file already includes Docker Hub authentication:

```yaml
- name: Log in to Docker Hub
  uses: docker/login-action@v3
  with:
    username: ${{ secrets.DOCKER_HUB_USERNAME }}
    password: ${{ secrets.DOCKER_HUB_TOKEN }}
```

And the `.goreleaser.yml` file configures multi-arch Docker images:

```yaml
dockers:
  - image_templates:
      - "coderisk/crisk:{{ .Tag }}"
      - "coderisk/crisk:v{{ .Major }}"
      - "coderisk/crisk:v{{ .Major }}.{{ .Minor }}"
      - "coderisk/crisk:latest"
```

## Step 5: Test the Setup

### Option A: Test with Pre-Release

```bash
cd coderisk-go

# Create a test tag
git tag -a v0.1.0-test -m "Test Docker build"
git push origin v0.1.0-test
```

Watch the workflow: https://github.com/rohankatakam/coderisk-go/actions

### Option B: Test Locally

```bash
# Build Docker image locally
docker build -t coderisk/crisk:test .

# Test the image
docker run --rm coderisk/crisk:test --version

# Test with a repository
docker run --rm -v $(pwd):/repo coderisk/crisk:test check
```

## Step 6: Verify Published Images

After a successful release:

1. Check Docker Hub: https://hub.docker.com/r/coderisk/crisk/tags
2. Verify tags exist:
   - `latest`
   - `v1.0.0` (full version)
   - `v1.0` (minor version)
   - `v1` (major version)
   - `v1.0.0-arm64` (ARM variant)
   - `latest-arm64` (ARM variant)

3. Test pulling and running:
   ```bash
   docker pull coderisk/crisk:latest
   docker run --rm coderisk/crisk:latest --version
   ```

## Image Details

### Supported Platforms

- `linux/amd64` (Intel/AMD 64-bit)
- `linux/arm64` (ARM 64-bit, e.g., Apple Silicon)

### Image Size

- Target: <20MB (Alpine-based)
- Actual size will be shown on Docker Hub

### Image Labels

Each image includes metadata:
- `org.opencontainers.image.title` - "CodeRisk CLI"
- `org.opencontainers.image.version` - Release version
- `org.opencontainers.image.revision` - Git commit SHA
- `org.opencontainers.image.source` - GitHub repository URL

## Usage Examples

### Basic Usage

```bash
# Pull latest image
docker pull coderisk/crisk:latest

# Run version check
docker run --rm coderisk/crisk:latest --version

# Check a repository
docker run --rm -v $(pwd):/repo coderisk/crisk:latest check

# With API key
docker run --rm \
  -v $(pwd):/repo \
  -e OPENAI_API_KEY="sk-..." \
  coderisk/crisk:latest check --explain
```

### CI/CD Integration

**GitHub Actions:**
```yaml
- name: Run CodeRisk
  uses: docker://coderisk/crisk:v1
  with:
    args: check
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
```

**GitLab CI:**
```yaml
coderisk:
  image: coderisk/crisk:latest
  script:
    - crisk check
  variables:
    OPENAI_API_KEY: $OPENAI_API_KEY
```

## Maintenance

### Updating Images

Images are automatically built and pushed on each release. No manual intervention needed.

### Revoking Access

If the token is compromised:

1. Go to https://hub.docker.com/settings/security
2. Delete the compromised token
3. Create a new token (Step 2)
4. Update GitHub secret (Step 3)

### Monitoring

Track downloads on Docker Hub:
- Go to https://hub.docker.com/r/coderisk/crisk
- View "Public Stats" tab for pull metrics

## Troubleshooting

### "authentication required" error

- Verify secrets are set correctly in GitHub
- Check token hasn't expired
- Ensure token has "Read & Write" permissions

### Image build fails

- Check GitHub Actions logs
- Verify Dockerfile is valid: `docker build -t test .`
- Ensure all files referenced in Dockerfile exist

### Wrong architecture

- Verify `docker/setup-qemu-action@v3` is in workflow
- Check `docker/setup-buildx-action@v3` is configured
- Use `docker manifest inspect coderisk/crisk:latest` to verify platforms

## Resources

- [Docker Hub Documentation](https://docs.docker.com/docker-hub/)
- [Docker Build Documentation](https://docs.docker.com/build/)
- [GoReleaser Docker Documentation](https://goreleaser.com/customization/docker/)
- [GitHub Actions Docker Login](https://github.com/marketplace/actions/docker-login)

---

**Next Steps:**
1. ✅ Create Docker Hub repository
2. ✅ Generate access token
3. ✅ Add secrets to GitHub
4. ✅ Test with pre-release
5. ✅ Verify images on Docker Hub
6. ✅ Update documentation with Docker usage
