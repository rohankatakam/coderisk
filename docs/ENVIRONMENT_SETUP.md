# CodeRisk Environment Configuration

## Overview

CodeRisk uses environment variables for configuration, making it easy to manage settings across different deployments (development, team, enterprise) without hardcoding sensitive information.

## Quick Setup

### 1. Create Your Environment File

```bash
# Copy the example file and customize it
cp .env.example .env

# Edit with your settings
nano .env  # or your preferred editor
```

### 2. Required Settings

At minimum, you need:

```bash
# Required: GitHub token for repository access
GITHUB_TOKEN=ghp_your_github_token_here
```

Get your GitHub token at: https://github.com/settings/tokens
- For private repos: Select `repo` scope
- For public repos only: Select `public_repo` scope

### 3. Optional but Recommended

```bash
# For enhanced analysis (Level 3)
OPENAI_API_KEY=sk-your_openai_key_here

# For team deployment
SHARED_CACHE_URL=https://cache.yourteam.com/api/v1
```

## Configuration Hierarchy

CodeRisk loads configuration in this order (later sources override earlier ones):

1. **Built-in defaults** (most conservative settings)
2. **`.env.example`** (template with documented defaults)
3. **`.env`** (your main environment file)
4. **`.env.local`** (local overrides, git-ignored)
5. **`~/.coderisk/.env`** (global user settings)
6. **Environment variables** (highest precedence)
7. **Config file** (`.coderisk/config.yaml`)
8. **Command line flags** (ultimate override)

## Environment File Locations

CodeRisk searches for `.env` files in this order:

1. **`.env.local`** (current directory - for local development)
2. **`.env`** (current directory - main config)
3. **`~/.coderisk/.env`** (user home directory - global config)

## Deployment Scenarios

### Local Development

```bash
# .env
GITHUB_TOKEN=ghp_your_token
STORAGE_TYPE=sqlite
LOCAL_DB_PATH=~/.coderisk/dev.db
CODERISK_MODE=local
LOG_LEVEL=debug
```

### Team Deployment

```bash
# .env
GITHUB_TOKEN=ghp_team_token
CODERISK_MODE=team
SHARED_CACHE_URL=https://cache.yourteam.com/api/v1
OPENAI_API_KEY=sk-team_key
SYNC_AUTO_SYNC=true
```

### Enterprise Deployment

```bash
# .env
GITHUB_TOKEN=ghp_enterprise_token
CODERISK_MODE=enterprise
STORAGE_TYPE=postgres
POSTGRES_DSN=postgres://user:pass@db.company.com:5432/coderisk

# Custom LLM endpoints
CUSTOM_LLM_URL=https://llm.company.com/v1
CUSTOM_LLM_KEY=your_custom_key
CUSTOM_EMBEDDING_URL=https://embeddings.company.com/v1
CUSTOM_EMBEDDING_KEY=your_embedding_key

# Higher limits for enterprise
BUDGET_DAILY_LIMIT=50.00
BUDGET_MONTHLY_LIMIT=1500.00
```

## Security Best Practices

### File Permissions

```bash
# Secure your .env files
chmod 600 .env
chmod 600 .env.local
```

### Git Ignore

The `.gitignore` already includes:
```
.env
.env.local
*.env
```

**Never commit actual `.env` files!**

### Token Management

1. **Use least privilege**: Only grant necessary GitHub scopes
2. **Rotate regularly**: Update tokens every 90 days
3. **Team tokens**: Use dedicated service account tokens
4. **Enterprise**: Integrate with your secret management system

## Environment Variables Reference

### Core Configuration

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `GITHUB_TOKEN` | GitHub Personal Access Token | (required) | `ghp_abc123...` |
| `OPENAI_API_KEY` | OpenAI API key for enhanced analysis | (optional) | `sk-abc123...` |
| `CODERISK_MODE` | Deployment mode | `team` | `local`, `team`, `enterprise`, `oss` |
| `STORAGE_TYPE` | Database backend | `sqlite` | `sqlite`, `postgres` |
| `LOG_LEVEL` | Logging verbosity | `info` | `debug`, `info`, `warn`, `error` |

### Budget Controls

| Variable | Description | Default | Notes |
|----------|-------------|---------|-------|
| `BUDGET_DAILY_LIMIT` | Daily spending limit (USD) | `2.00` | Prevents runaway costs |
| `BUDGET_MONTHLY_LIMIT` | Monthly spending limit (USD) | `60.00` | Total monthly budget |
| `BUDGET_PER_CHECK_LIMIT` | Max cost per risk check | `0.04` | Prevents expensive checks |

### Performance Tuning

| Variable | Description | Default | Notes |
|----------|-------------|---------|-------|
| `GITHUB_RATE_LIMIT` | GitHub API requests/second | `10` | Increase if you have higher limits |
| `CACHE_MAX_SIZE` | Local cache size limit (bytes) | `2147483648` | 2GB default |
| `SYNC_FRESH_THRESHOLD_MINUTES` | Auto-sync threshold | `30` | Minutes before sync |
| `SYNC_STALE_THRESHOLD_HOURS` | Force sync threshold | `4` | Hours before forced sync |

### Custom Endpoints (Enterprise)

| Variable | Description | Example |
|----------|-------------|---------|
| `CUSTOM_LLM_URL` | Custom LLM API endpoint | `https://llm.company.com/v1` |
| `CUSTOM_LLM_KEY` | API key for custom LLM | `your_custom_key` |
| `CUSTOM_EMBEDDING_URL` | Custom embedding endpoint | `https://embed.company.com/v1` |
| `CUSTOM_EMBEDDING_KEY` | API key for embeddings | `your_embedding_key` |

## Validation and Testing

### Check Your Configuration

```bash
# View loaded configuration
crisk config list

# Test GitHub connection
crisk status

# Validate environment
crisk config get github.token  # Should show masked token
```

### Environment Validation Script

```bash
#!/bin/bash
# validate-env.sh

echo "🔍 Validating CodeRisk environment..."

# Check required variables
if [ -z "$GITHUB_TOKEN" ]; then
    echo "❌ GITHUB_TOKEN is required"
    exit 1
fi

echo "✅ GITHUB_TOKEN is set"

# Check GitHub token format
if [[ ! "$GITHUB_TOKEN" =~ ^ghp_[a-zA-Z0-9]{36}$ ]]; then
    echo "⚠️  GITHUB_TOKEN format looks unusual"
fi

# Check OpenAI key if present
if [ -n "$OPENAI_API_KEY" ]; then
    if [[ "$OPENAI_API_KEY" =~ ^sk- ]]; then
        echo "✅ OPENAI_API_KEY format looks correct"
    else
        echo "⚠️  OPENAI_API_KEY format looks unusual"
    fi
fi

# Check file permissions
if [ -f ".env" ]; then
    PERMS=$(ls -l .env | cut -d' ' -f1)
    if [[ "$PERMS" == "-rw-------" ]]; then
        echo "✅ .env file permissions are secure"
    else
        echo "⚠️  .env file should be chmod 600 for security"
    fi
fi

echo "🎉 Environment validation complete!"
```

## Common Issues

### Issue: "GitHub API rate limit exceeded"

**Solution**:
```bash
# Increase rate limit in .env
GITHUB_RATE_LIMIT=5  # Reduce to 5 req/s
```

### Issue: "Config file not found"

**Solution**:
```bash
# Create default config
crisk config init

# Or specify explicit path
crisk --config /path/to/config.yaml status
```

### Issue: "Token permission denied"

**Solution**:
1. Check token scopes at https://github.com/settings/tokens
2. Ensure `repo` scope for private repos or `public_repo` for public
3. Verify token hasn't expired

### Issue: "Database connection failed"

**Solution**:
```bash
# For SQLite issues
rm ~/.coderisk/local.db  # Remove corrupted DB

# For PostgreSQL issues
# Check POSTGRES_DSN format:
POSTGRES_DSN=postgres://user:password@host:port/database?sslmode=disable
```

## Migration from Manual Export

If you're currently using `export` statements:

### Before (manual exports)
```bash
export GITHUB_TOKEN="ghp_abc123"
export OPENAI_API_KEY="sk_abc123"
crisk check
```

### After (.env file)
```bash
# Create .env file with:
GITHUB_TOKEN=ghp_abc123
OPENAI_API_KEY=sk_abc123

# Now just run:
crisk check
```

The `.env` approach is better because:
- ✅ **Persistent**: Settings survive terminal sessions
- ✅ **Shareable**: Team can use same configuration template
- ✅ **Secure**: Files can be properly permissioned
- ✅ **Organized**: All settings in one place
- ✅ **Version controlled**: `.env.example` documents required settings

## Next Steps

1. **Copy the example**: `cp .env.example .env`
2. **Add your tokens**: Edit `.env` with your GitHub token
3. **Test the setup**: Run `crisk status` to verify
4. **Initialize repo**: Run `crisk init` in a Git repository
5. **Start using**: Run `crisk check` to assess code changes

Your environment variables will now be automatically loaded by all CodeRisk programs!